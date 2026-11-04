package v1

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"image"
	"image/color"
	"image/png"
	"net/http"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/javaknight1/servicepro/backend/internal/models"
	"github.com/javaknight1/servicepro/backend/internal/tenancy"
)

// T176 end to end on Postgres: real middleware (tenant, permissions, the
// technician access scope), handler, service and repository; an in-memory
// object store and email client.

// inkPNG is a w x h transparent canvas, with a pen stroke when ink.
func inkPNG(t *testing.T, w, h int, ink bool) []byte {
	t.Helper()
	img := image.NewNRGBA(image.Rect(0, 0, w, h))
	if ink {
		for x := w / 10; x < w*9/10; x++ {
			for dy := -2; dy <= 2; dy++ {
				img.Set(x, h/2+(x%30-15)+dy, color.NRGBA{10, 10, 10, 255})
			}
		}
	}
	var buf bytes.Buffer
	require.NoError(t, png.Encode(&buf, img))
	return buf.Bytes()
}

func signoffBody(t *testing.T, data []byte) map[string]any {
	return map[string]any{
		"signer_name":  "Ray Burke",
		"signer_title": "Site manager",
		"signer_email": "ray@example.com",
		"signature":    "data:image/png;base64," + base64.StdEncoding.EncodeToString(data),
	}
}

func (f *techFixture) startJob(t *testing.T, job uuid.UUID) {
	t.Helper()
	require.NoError(t, f.db.WithContext(tenancy.WithSystem(context.Background())).
		Exec(`UPDATE jobs SET status = 'in_progress', actual_start_at = EXTRACT(EPOCH FROM NOW())::bigint - 3600 WHERE id = ?`, job).Error)
}

func decodeSignoff(t *testing.T, body []byte) models.JobSignoffResponse {
	t.Helper()
	var s models.JobSignoffResponse
	require.NoError(t, json.Unmarshal(body, &s), string(body))
	return s
}

func TestPostgresJobSignoffTechnicianOnAssignedJob(t *testing.T) {
	f := setupTechFixture(t)
	job := f.assignedJob.ID
	base := "/jobs/" + job.String() + "/signoff"

	// Not started yet: no signing.
	w := f.do(t, f.techA, f.tenantA, http.MethodPost, base, signoffBody(t, inkPNG(t, 900, 300, true)))
	assert.Equal(t, http.StatusConflict, w.Code, w.Body.String())
	assert.Equal(t, "job_not_started", errorCode(t, w))

	// Nothing signed: null, with who the report would go to.
	w = f.do(t, f.techA, f.tenantA, http.MethodGet, base, nil)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	var env models.JobSignoffEnvelope
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &env))
	assert.Nil(t, env.Signoff)
	assert.Equal(t, []string{"pat@example.com"}, env.SuggestedRecipients)
	assert.Equal(t, http.StatusNotFound, f.do(t, f.techA, f.tenantA, http.MethodGet, base+"/report", nil).Code)

	f.startJob(t, job)
	w = f.do(t, f.techA, f.tenantA, http.MethodPost, base, signoffBody(t, inkPNG(t, 900, 300, true)))
	require.Equal(t, http.StatusCreated, w.Code, w.Body.String())
	so := decodeSignoff(t, w.Body.Bytes())
	assert.Equal(t, "Ray Burke", so.SignerName)
	assert.Equal(t, f.techA, *so.CapturedBy)
	assert.Equal(t, []string{"pat@example.com", "ray@example.com"}, so.SuggestedRecipients)
	var row models.JobSignoff
	require.NoError(t, f.db.WithContext(tenancy.WithSystem(context.Background())).First(&row, "job_id = ?", job).Error)
	assert.Equal(t, f.tenantA, row.TenantID)
	require.NotNil(t, row.IPAddress, "IP recorded through middleware.ClientIP")

	// Signature and report redirect to presigned links; the report is stored.
	for _, path := range []string{base + "/signature", base + "/report"} {
		w = f.do(t, f.techA, f.tenantA, http.MethodGet, path, nil)
		require.Equal(t, http.StatusFound, w.Code, path+": "+w.Body.String())
		assert.True(t, strings.HasPrefix(w.Header().Get("Location"), "mock://"), path)
	}
	var reports int
	for key, data := range f.store.GetAllObjects() {
		if strings.HasSuffix(key, "_report.pdf") {
			reports++
			assert.True(t, bytes.HasPrefix(data, []byte("%PDF")))
		}
	}
	assert.Equal(t, 1, reports)

	// Email: once; a retry sends nothing; resend is explicit.
	w = f.do(t, f.techA, f.tenantA, http.MethodPost, base+"/email", nil)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	var sent models.EmailJobReportResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &sent))
	assert.True(t, sent.Sent)
	assert.True(t, sent.Signoff.Locked)
	require.Len(t, f.mail.SentEmails, 1)
	msg := f.mail.SentEmails[0]
	assert.Equal(t, []string{"pat@example.com", "ray@example.com"}, msg.To)
	require.Len(t, msg.Attachments, 1)
	assert.Equal(t, "Service-Report-TECH-1.pdf", msg.Attachments[0].Filename)
	assert.True(t, bytes.HasPrefix(msg.Attachments[0].Content, []byte("%PDF")))

	w = f.do(t, f.techA, f.tenantA, http.MethodPost, base+"/email", map[string]bool{"resend": false})
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &sent))
	assert.False(t, sent.Sent)
	assert.Len(t, f.mail.SentEmails, 1)

	w = f.do(t, f.officeA, f.tenantA, http.MethodPost, base+"/email", map[string]bool{"resend": true})
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &sent))
	assert.True(t, sent.Sent)
	assert.Equal(t, 2, sent.Signoff.ReportEmailCount)
	assert.Len(t, f.mail.SentEmails, 2)

	// Emailed: the sign-off is the record.
	w = f.do(t, f.techA, f.tenantA, http.MethodPost, base, signoffBody(t, inkPNG(t, 900, 300, true)))
	assert.Equal(t, http.StatusConflict, w.Code)
	assert.Equal(t, "signoff_locked", errorCode(t, w))
}

func TestPostgresJobSignoffTechnicianUnassignedJobIs404(t *testing.T) {
	f := setupTechFixture(t)
	f.startJob(t, f.otherJob.ID)
	base := "/jobs/" + f.otherJob.ID.String() + "/signoff"
	// The office signs techA2's job.
	w := f.do(t, f.officeA, f.tenantA, http.MethodPost, base, signoffBody(t, inkPNG(t, 900, 300, true)))
	require.Equal(t, http.StatusCreated, w.Code, w.Body.String())

	w = f.do(t, f.techA, f.tenantA, http.MethodPost, base, signoffBody(t, inkPNG(t, 900, 300, true)))
	assert.Equal(t, http.StatusNotFound, w.Code, w.Body.String())
	assert.Equal(t, "job_not_found", errorCode(t, w))
	for _, path := range []string{base, base + "/signature", base + "/report"} {
		assert.Equal(t, http.StatusNotFound, f.do(t, f.techA, f.tenantA, http.MethodGet, path, nil).Code, path)
	}
	assert.Equal(t, http.StatusNotFound, f.do(t, f.techA, f.tenantA, http.MethodPost, base+"/email", nil).Code)
	assert.Empty(t, f.mail.SentEmails)
}

func TestPostgresJobSignoffCrossTenant(t *testing.T) {
	f := setupTechFixture(t)
	f.startJob(t, f.assignedJob.ID)
	base := "/jobs/" + f.assignedJob.ID.String() + "/signoff"
	w := f.do(t, f.ownerA, f.tenantA, http.MethodPost, base, signoffBody(t, inkPNG(t, 900, 300, true)))
	require.Equal(t, http.StatusCreated, w.Code, w.Body.String())

	// Tenant B's owner: tenant A's job and sign-off don't exist.
	for _, path := range []string{base, base + "/signature", base + "/report"} {
		assert.Equal(t, http.StatusNotFound, f.do(t, f.ownerB, f.tenantB, http.MethodGet, path, nil).Code, path)
	}
	w = f.do(t, f.ownerB, f.tenantB, http.MethodPost, base, signoffBody(t, inkPNG(t, 900, 300, true)))
	assert.Equal(t, http.StatusNotFound, w.Code)
	assert.Equal(t, http.StatusNotFound, f.do(t, f.ownerB, f.tenantB, http.MethodPost, base+"/email", nil).Code)
	// Naming tenant A in the header without membership.
	assert.Equal(t, http.StatusForbidden, f.do(t, f.ownerB, f.tenantA, http.MethodGet, base, nil).Code)
	assert.Empty(t, f.mail.SentEmails)

	var n int64
	require.NoError(t, f.db.WithContext(tenancy.WithSystem(context.Background())).
		Model(&models.JobSignoff{}).Where("tenant_id = ?", f.tenantB).Count(&n).Error)
	assert.Zero(t, n)
}

func TestPostgresJobSignoffRejectsBadSignatures(t *testing.T) {
	f := setupTechFixture(t)
	job := f.assignedJob.ID
	f.startJob(t, job)
	base := "/jobs/" + job.String() + "/signoff"
	post := func(body any) (int, string) {
		w := f.do(t, f.techA, f.tenantA, http.MethodPost, base, body)
		return w.Code, errorCode(t, w)
	}

	body := signoffBody(t, nil)
	body["signature"] = "data:image/png;base64," + base64.StdEncoding.EncodeToString([]byte("<html><script>alert(1)</script></html>"))
	code, errCode := post(body)
	assert.Equal(t, http.StatusUnsupportedMediaType, code)
	assert.Equal(t, "unsupported_signature_type", errCode)

	code, errCode = post(signoffBody(t, bombPNG(50000, 50000)))
	assert.Equal(t, http.StatusBadRequest, code)
	assert.Equal(t, "signature_dimensions", errCode)

	code, errCode = post(signoffBody(t, inkPNG(t, 900, 300, false)))
	assert.Equal(t, http.StatusBadRequest, code)
	assert.Equal(t, "signature_empty", errCode)

	code, errCode = post(signoffBody(t, make([]byte, 700<<10)))
	assert.Equal(t, http.StatusRequestEntityTooLarge, code)
	assert.Equal(t, "signature_too_large", errCode)

	body = signoffBody(t, inkPNG(t, 900, 300, true))
	delete(body, "signer_name")
	code, _ = post(body)
	assert.Equal(t, http.StatusBadRequest, code)

	assert.Empty(t, f.store.GetAllObjects())
}
