package v1

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"hash/crc32"
	"image"
	"image/jpeg"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/javaknight1/servicepro/backend/internal/models"
	"github.com/javaknight1/servicepro/backend/internal/services"
)

// T174 end to end on Postgres: real middleware (tenant, permissions, the
// technician access scope), handler, service and repository; an in-memory
// object store.

func smallJPEG(t *testing.T) []byte {
	t.Helper()
	var buf bytes.Buffer
	require.NoError(t, jpeg.Encode(&buf, image.NewRGBA(image.Rect(0, 0, 32, 24)), nil))
	return buf.Bytes()
}

// bombPNG is a PNG signature + IHDR declaring w x h pixels and no data.
func bombPNG(w, h uint32) []byte {
	var buf bytes.Buffer
	buf.Write([]byte{0x89, 'P', 'N', 'G', '\r', '\n', 0x1a, '\n'})
	ihdr := make([]byte, 13)
	binary.BigEndian.PutUint32(ihdr[0:], w)
	binary.BigEndian.PutUint32(ihdr[4:], h)
	ihdr[8], ihdr[9] = 8, 6
	_ = binary.Write(&buf, binary.BigEndian, uint32(len(ihdr)))
	chunk := append([]byte("IHDR"), ihdr...)
	buf.Write(chunk)
	_ = binary.Write(&buf, binary.BigEndian, crc32.ChecksumIEEE(chunk))
	return buf.Bytes()
}

// upload posts data as multipart field "file" (declared as image/jpeg
// whatever it is: the server sniffs).
func (f *techFixture) upload(t *testing.T, user, tenant, job uuid.UUID, name string, data []byte, caption string) *httptest.ResponseRecorder {
	t.Helper()
	var body bytes.Buffer
	mw := multipart.NewWriter(&body)
	part, err := mw.CreateFormFile("file", name)
	require.NoError(t, err)
	_, err = part.Write(data)
	require.NoError(t, err)
	if caption != "" {
		require.NoError(t, mw.WriteField("caption", caption))
	}
	require.NoError(t, mw.Close())
	req := httptest.NewRequest(http.MethodPost, "/api/v1/jobs/"+job.String()+"/attachments", &body)
	token, err := f.jwt.GenerateAccessToken(user.String(), "u@example.com")
	require.NoError(t, err)
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("X-Tenant-ID", tenant.String())
	req.Header.Set("Content-Type", mw.FormDataContentType())
	w := httptest.NewRecorder()
	f.engine.ServeHTTP(w, req)
	return w
}

func decodeAttachment(t *testing.T, w *httptest.ResponseRecorder) models.JobAttachmentResponse {
	t.Helper()
	var a models.JobAttachmentResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &a), w.Body.String())
	return a
}

func errorCode(t *testing.T, w *httptest.ResponseRecorder) string {
	t.Helper()
	var e models.ErrorResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &e), w.Body.String())
	return e.Error
}

func TestPostgresJobAttachmentsTechnicianOnAssignedJob(t *testing.T) {
	f := setupTechFixture(t)
	job := f.assignedJob.ID
	base := "/jobs/" + job.String() + "/attachments"

	w := f.upload(t, f.techA, f.tenantA, job, "IMG_2041.jpg", smallJPEG(t), "Lobby floor")
	require.Equal(t, http.StatusCreated, w.Code, w.Body.String())
	mine := decodeAttachment(t, w)
	assert.Equal(t, models.JobAttachmentPhoto, mine.Kind)
	assert.Equal(t, "Lobby floor", *mine.Caption)
	assert.Equal(t, f.techA, *mine.UploadedBy)

	w = f.do(t, f.techA, f.tenantA, http.MethodGet, base, nil)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	var list models.JobAttachmentListResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &list))
	require.Equal(t, 1, list.Total)
	assert.True(t, strings.HasPrefix(list.Attachments[0].ThumbURL, "mock://"))

	// Download and thumbnail redirect to presigned links.
	for _, kind := range []string{"file", "thumb"} {
		w = f.do(t, f.techA, f.tenantA, http.MethodGet, base+"/"+mine.ID.String()+"/"+kind, nil)
		require.Equal(t, http.StatusFound, w.Code, kind)
		assert.True(t, strings.HasPrefix(w.Header().Get("Location"), "mock://"), kind)
	}

	// The office adds one; the technician can't delete or caption it.
	w = f.upload(t, f.officeA, f.tenantA, job, "office.jpg", smallJPEG(t), "")
	require.Equal(t, http.StatusCreated, w.Code, w.Body.String())
	office := decodeAttachment(t, w)
	w = f.do(t, f.techA, f.tenantA, http.MethodDelete, base+"/"+office.ID.String(), nil)
	assert.Equal(t, http.StatusForbidden, w.Code)
	assert.Equal(t, "attachment_not_yours", errorCode(t, w))
	w = f.do(t, f.techA, f.tenantA, http.MethodPatch, base+"/"+office.ID.String(), map[string]string{"caption": "x"})
	assert.Equal(t, http.StatusForbidden, w.Code)

	// Their own: caption and delete.
	w = f.do(t, f.techA, f.tenantA, http.MethodPatch, base+"/"+mine.ID.String(), map[string]string{"caption": "Restroom 2F"})
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	assert.Equal(t, "Restroom 2F", *decodeAttachment(t, w).Caption)
	w = f.do(t, f.techA, f.tenantA, http.MethodDelete, base+"/"+mine.ID.String(), nil)
	require.Equal(t, http.StatusNoContent, w.Code, w.Body.String())

	// Office staff delete anyone's.
	w = f.do(t, f.officeA, f.tenantA, http.MethodDelete, base+"/"+office.ID.String(), nil)
	require.Equal(t, http.StatusNoContent, w.Code, w.Body.String())
	assert.Empty(t, f.store.GetAllObjects())
}

func TestPostgresJobAttachmentsTechnicianUnassignedJobIs404(t *testing.T) {
	f := setupTechFixture(t)
	// An attachment on techA2's job, added by the office.
	w := f.upload(t, f.officeA, f.tenantA, f.otherJob.ID, "a.jpg", smallJPEG(t), "")
	require.Equal(t, http.StatusCreated, w.Code, w.Body.String())
	att := decodeAttachment(t, w)
	base := "/jobs/" + f.otherJob.ID.String() + "/attachments"

	w = f.upload(t, f.techA, f.tenantA, f.otherJob.ID, "b.jpg", smallJPEG(t), "")
	assert.Equal(t, http.StatusNotFound, w.Code, w.Body.String())
	assert.Equal(t, "job_not_found", errorCode(t, w))
	for _, path := range []string{base, base + "/" + att.ID.String() + "/file", base + "/" + att.ID.String() + "/thumb"} {
		w = f.do(t, f.techA, f.tenantA, http.MethodGet, path, nil)
		assert.Equal(t, http.StatusNotFound, w.Code, path)
	}
	w = f.do(t, f.techA, f.tenantA, http.MethodDelete, base+"/"+att.ID.String(), nil)
	assert.Equal(t, http.StatusNotFound, w.Code)

	// The attachment can't be reached through the technician's own job either.
	w = f.do(t, f.techA, f.tenantA, http.MethodGet, "/jobs/"+f.assignedJob.ID.String()+"/attachments/"+att.ID.String()+"/file", nil)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestPostgresJobAttachmentsCrossTenant(t *testing.T) {
	f := setupTechFixture(t)
	w := f.upload(t, f.ownerA, f.tenantA, f.assignedJob.ID, "a.jpg", smallJPEG(t), "")
	require.Equal(t, http.StatusCreated, w.Code, w.Body.String())
	att := decodeAttachment(t, w)
	base := "/jobs/" + f.assignedJob.ID.String() + "/attachments"

	// Tenant B's owner: tenant A's job and attachment don't exist.
	w = f.do(t, f.ownerB, f.tenantB, http.MethodGet, base, nil)
	assert.Equal(t, http.StatusNotFound, w.Code)
	w = f.do(t, f.ownerB, f.tenantB, http.MethodGet, base+"/"+att.ID.String()+"/file", nil)
	assert.Equal(t, http.StatusNotFound, w.Code)
	w = f.do(t, f.ownerB, f.tenantB, http.MethodDelete, base+"/"+att.ID.String(), nil)
	assert.Equal(t, http.StatusNotFound, w.Code)
	w = f.upload(t, f.ownerB, f.tenantB, f.assignedJob.ID, "b.jpg", smallJPEG(t), "")
	assert.Equal(t, http.StatusNotFound, w.Code)
	// Through tenant B's own job.
	w = f.do(t, f.ownerB, f.tenantB, http.MethodGet, "/jobs/"+f.jobB.ID.String()+"/attachments/"+att.ID.String()+"/file", nil)
	assert.Equal(t, http.StatusNotFound, w.Code)
	// Naming tenant A in the header without membership.
	w = f.do(t, f.ownerB, f.tenantA, http.MethodGet, base, nil)
	assert.Equal(t, http.StatusForbidden, w.Code)
}

func TestPostgresJobAttachmentsRejectBadUploads(t *testing.T) {
	f := setupTechFixture(t)
	job := f.assignedJob.ID

	w := f.upload(t, f.techA, f.tenantA, job, "evil.jpg", []byte("<html><script>alert(1)</script>"), "")
	assert.Equal(t, http.StatusUnsupportedMediaType, w.Code, w.Body.String())
	assert.Equal(t, "unsupported_file_type", errorCode(t, w))

	w = f.upload(t, f.techA, f.tenantA, job, "bomb.png", bombPNG(50000, 50000), "")
	assert.Equal(t, http.StatusBadRequest, w.Code, w.Body.String())
	assert.Equal(t, "image_too_large", errorCode(t, w))

	big := append([]byte{0xFF, 0xD8, 0xFF}, make([]byte, services.MaxAttachmentBytes+128<<10)...)
	w = f.upload(t, f.techA, f.tenantA, job, "big.jpg", big, "")
	assert.Equal(t, http.StatusRequestEntityTooLarge, w.Code, w.Body.String())
	assert.Equal(t, "file_too_large", errorCode(t, w))

	w = f.do(t, f.techA, f.tenantA, http.MethodPost, "/jobs/"+job.String()+"/attachments", nil)
	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Equal(t, "file_required", errorCode(t, w))

	assert.Empty(t, f.store.GetAllObjects())
}
