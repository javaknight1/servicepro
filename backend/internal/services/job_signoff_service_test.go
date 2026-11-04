package services

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/binary"
	"errors"
	"hash/crc32"
	"image"
	"image/color"
	"image/png"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apperrors "github.com/javaknight1/servicepro/backend/internal/errors"
	"github.com/javaknight1/servicepro/backend/internal/models"
	"github.com/javaknight1/servicepro/backend/internal/repository"
	"github.com/javaknight1/servicepro/backend/internal/services/pdf"
	"github.com/javaknight1/servicepro/backend/internal/tenancy"
	"github.com/javaknight1/servicepro/backend/pkg/clients/email"
	emailmock "github.com/javaknight1/servicepro/backend/pkg/clients/email/mock"
	"github.com/javaknight1/servicepro/backend/pkg/clients/storage"
	"github.com/javaknight1/servicepro/backend/pkg/clients/storage/mock"
)

// fakeSignoffRepo is an in-memory JobSignoffRepositoryInterface scoped by
// tenant like the real one.
type fakeSignoffRepo struct {
	jobs     map[uuid.UUID]map[uuid.UUID]*repository.JobReportSource // tenant -> job
	contacts map[uuid.UUID]*repository.JobContacts
	rows     map[uuid.UUID]*models.JobSignoff // job -> sign-off
}

func (f *fakeSignoffRepo) source(tenantID, jobID uuid.UUID) *repository.JobReportSource {
	return f.jobs[tenantID][jobID]
}

func (f *fakeSignoffRepo) JobVisible(_ context.Context, tenantID, jobID uuid.UUID) (bool, error) {
	return f.source(tenantID, jobID) != nil, nil
}

func (f *fakeSignoffRepo) JobStatus(_ context.Context, tenantID, jobID uuid.UUID) (models.JobStatus, error) {
	src := f.source(tenantID, jobID)
	if src == nil {
		return "", repository.ErrJobSignoffNotFound
	}
	return src.Job.Status, nil
}

func (f *fakeSignoffRepo) row(tenantID, jobID uuid.UUID) *models.JobSignoff {
	r := f.rows[jobID]
	if r == nil || r.TenantID != tenantID {
		return nil
	}
	return r
}

func (f *fakeSignoffRepo) Get(_ context.Context, tenantID, jobID uuid.UUID) (*models.JobSignoff, error) {
	r := f.row(tenantID, jobID)
	if r == nil {
		return nil, repository.ErrJobSignoffNotFound
	}
	c := *r
	return &c, nil
}

func (f *fakeSignoffRepo) Save(_ context.Context, tenantID uuid.UUID, s *models.JobSignoff) (*models.JobSignoff, error) {
	s.TenantID = tenantID
	old := f.row(tenantID, s.JobID)
	if old != nil && old.ReportEmailedAt != nil {
		return nil, repository.ErrJobSignoffLocked
	}
	c := *s
	if old != nil {
		c.ID = old.ID
	} else if c.ID == uuid.Nil {
		c.ID = uuid.New()
	}
	s.ID = c.ID
	f.rows[s.JobID] = &c
	return old, nil
}

func (f *fakeSignoffRepo) SetReport(_ context.Context, tenantID, jobID uuid.UUID, key string, at time.Time) error {
	r := f.row(tenantID, jobID)
	if r == nil {
		return repository.ErrJobSignoffNotFound
	}
	r.ReportKey, r.ReportGeneratedAt = &key, &at
	return nil
}

func (f *fakeSignoffRepo) ClaimEmail(_ context.Context, tenantID, jobID uuid.UUID, resend bool, now, staleBefore time.Time) (bool, error) {
	r := f.row(tenantID, jobID)
	if r == nil || (r.ReportEmailClaimedAt != nil && !r.ReportEmailClaimedAt.Before(staleBefore)) ||
		(!resend && r.ReportEmailedAt != nil) {
		return false, nil
	}
	r.ReportEmailClaimedAt = &now
	return true, nil
}

func (f *fakeSignoffRepo) FinishEmail(_ context.Context, tenantID, jobID uuid.UUID, recipients []string, at time.Time) error {
	r := f.row(tenantID, jobID)
	joined := strings.Join(recipients, ",")
	r.ReportEmailedAt, r.ReportRecipients, r.ReportEmailClaimedAt = &at, &joined, nil
	r.ReportEmailCount++
	return nil
}

func (f *fakeSignoffRepo) ReleaseEmail(_ context.Context, tenantID, jobID uuid.UUID) error {
	if r := f.row(tenantID, jobID); r != nil {
		r.ReportEmailClaimedAt = nil
	}
	return nil
}

func (f *fakeSignoffRepo) Contacts(_ context.Context, tenantID, jobID uuid.UUID) (*repository.JobContacts, error) {
	if f.source(tenantID, jobID) == nil {
		return nil, repository.ErrJobSignoffNotFound
	}
	return f.contacts[jobID], nil
}

func (f *fakeSignoffRepo) ReportSource(_ context.Context, tenantID, jobID uuid.UUID) (*repository.JobReportSource, error) {
	src := f.source(tenantID, jobID)
	if src == nil {
		return nil, repository.ErrJobSignoffNotFound
	}
	return src, nil
}

// failingEmail refuses every message.
type failingEmail struct{ *emailmock.Client }

func (failingEmail) Send(context.Context, *email.EmailMessage) (*email.SendResult, error) {
	return nil, errors.New("provider down")
}

type signoffFixture struct {
	svc              *JobSignoffService
	repo             *fakeSignoffRepo
	store            *mock.Client
	mail             *emailmock.Client
	tenantA, tenantB uuid.UUID
	jobA, jobB       uuid.UUID
	user             uuid.UUID
	ctxA, ctxB       context.Context
}

func testGenerator(t *testing.T) *pdf.Generator {
	t.Helper()
	cfg := pdf.DefaultConfig()
	cfg.Storage.Type = "local"
	cfg.Storage.LocalPath = t.TempDir()
	// Uncompressed, so tests can look for text in the PDF.
	cfg.Performance.EnableCompression = false
	gen, err := pdf.NewGenerator(context.Background(), cfg, nil)
	require.NoError(t, err)
	return gen
}

func newSignoffFixture(t *testing.T) *signoffFixture {
	t.Helper()
	f := &signoffFixture{
		tenantA: uuid.New(), tenantB: uuid.New(), jobA: uuid.New(), jobB: uuid.New(), user: uuid.New(),
		store: mock.NewClient(nil), mail: emailmock.NewClient(nil),
	}
	start := time.Date(2026, 11, 2, 22, 30, 0, 0, time.UTC).Unix() // 5:30 PM in New York
	end := start + 90*60
	actualStart, actualEnd := start+2*60, start+88*60
	company, po, site := "Harborview Property Management", "PO-7731", uuid.New()
	first, last := "Tasha", "Green"
	tech := &models.User{ID: f.user, Email: "tasha@example.com", FirstName: &first, LastName: &last}
	srcA := &repository.JobReportSource{
		Job: models.Job{
			ID: f.jobA, TenantID: f.tenantA, JobNumber: "J-3318", Title: "Elm Street Plaza nightly cleaning",
			Description: "Nightly cleaning, all floors.", Status: models.JobStatusInProgress, SiteID: &site,
			ScheduledStartAt: &start, ScheduledEndAt: &end, ActualStartAt: &actualStart, ActualEndAt: &actualEnd,
			PONumber: &po, EstimatedCost: 999.99,
			Customer:    &models.Customer{CompanyName: &company, Email: "karen@harborview.example"},
			Assignments: []models.JobAssignment{{UserID: f.user, User: tech}},
			Materials:   []models.JobMaterial{{Name: "Trash liners 33 gal", Quantity: 12, Unit: "each", UnitCost: 4321.98}},
			Notes:       []models.JobNote{{Note: "Entry mat by the east door is worn through.", User: tech, CreatedAt: time.Unix(actualEnd, 0)}},
		},
		Tenant: models.Tenant{ID: f.tenantA, Name: "Brightline Facility Services", Email: "office@brightline.example",
			Settings: models.TenantSettings{Timezone: "America/New_York"}},
		Site: &models.Site{Name: "Elm Street Plaza", AddressStreet: "410 Elm St", AddressCity: "Columbus", AddressState: "OH", AddressZip: "43215"},
	}
	srcB := &repository.JobReportSource{
		Job:    models.Job{ID: f.jobB, TenantID: f.tenantB, JobNumber: "J-1", Title: "B job", Status: models.JobStatusInProgress},
		Tenant: models.Tenant{ID: f.tenantB, Name: "B"},
	}
	f.repo = &fakeSignoffRepo{
		jobs: map[uuid.UUID]map[uuid.UUID]*repository.JobReportSource{
			f.tenantA: {f.jobA: srcA},
			f.tenantB: {f.jobB: srcB},
		},
		contacts: map[uuid.UUID]*repository.JobContacts{
			f.jobA: {CustomerEmail: "karen@harborview.example", SiteEmail: "Karen@harborview.example"},
			f.jobB: {CustomerEmail: "b@example.com"},
		},
		rows: map[uuid.UUID]*models.JobSignoff{},
	}
	f.svc = NewJobSignoffService(f.repo, f.store, testGenerator(t), f.mail)
	f.svc.now = func() time.Time { return time.Unix(actualEnd, 0) }
	f.ctxA = tenancy.WithTenant(context.Background(), f.tenantA)
	f.ctxB = tenancy.WithTenant(context.Background(), f.tenantB)
	return f
}

// signaturePNG draws a w x h transparent canvas with a dark stroke when ink.
func signaturePNG(t *testing.T, w, h int, ink bool) []byte {
	t.Helper()
	img := image.NewNRGBA(image.Rect(0, 0, w, h))
	if ink {
		for x := w / 10; x < w*9/10; x++ {
			y := h/2 + (x%40 - 20)
			for dy := -2; dy <= 2; dy++ {
				img.Set(x, y+dy, color.NRGBA{20, 20, 20, 255})
			}
		}
	}
	var buf bytes.Buffer
	require.NoError(t, png.Encode(&buf, img))
	return buf.Bytes()
}

// pngBomb is a PNG signature + IHDR declaring w x h pixels and no data.
func pngBomb(w, h uint32) []byte {
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

func dataURL(b []byte) string { return "data:image/png;base64," + base64.StdEncoding.EncodeToString(b) }

func (f *signoffFixture) sign(t *testing.T, ctx context.Context, job uuid.UUID) (*models.JobSignoffResponse, error) {
	t.Helper()
	title, mail := " Site manager ", "Ray.Burke@Harborview.example"
	return f.svc.Capture(ctx, job, f.user, &models.CaptureJobSignoffRequest{
		SignerName: "  Ray Burke\n", SignerTitle: &title, SignerEmail: &mail,
		Signature: dataURL(signaturePNG(t, 900, 300, true)),
	}, "203.0.113.9", "Mozilla/5.0 (iPhone)")
}

func TestDecodeSignature(t *testing.T) {
	ok := signaturePNG(t, 1500, 500, true)
	sig, err := decodeSignature(dataURL(ok))
	require.NoError(t, err)
	assert.Equal(t, storedSignatureMaxWidth, sig.Width, "scaled to the stored width")
	assert.Equal(t, 400, sig.Height)
	assert.Equal(t, "image/png", http.DetectContentType(sig.PNG))
	_, err = decodeSignature(base64.StdEncoding.EncodeToString(ok))
	require.NoError(t, err, "bare base64 is accepted")

	cases := []struct {
		name, input string
		status      int
		code        string
	}{
		{"jpeg data URL", "data:image/jpeg;base64," + base64.StdEncoding.EncodeToString(ok), 415, "unsupported_signature_type"},
		{"html sniffed", base64.StdEncoding.EncodeToString([]byte("<html><script>alert(1)</script></html>")), 415, "unsupported_signature_type"},
		{"not base64", "data:image/png;base64,@@@@", 400, "invalid_signature"},
		{"empty", "", 400, "invalid_signature"},
		{"pixel bomb", base64.StdEncoding.EncodeToString(pngBomb(50000, 50000)), 400, "signature_dimensions"},
		{"too small", dataURL(signaturePNG(t, 50, 20, true)), 400, "signature_dimensions"},
		{"blank canvas", dataURL(signaturePNG(t, 900, 300, false)), 400, "signature_empty"},
		{"too large", base64.StdEncoding.EncodeToString(make([]byte, MaxSignatureBytes+10)), 413, "signature_too_large"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := decodeSignature(tc.input)
			require.Error(t, err)
			assert.Equal(t, tc.status, statusOf(t, err))
			e, _ := apperrors.As(err)
			assert.Equal(t, tc.code, e.Code)
		})
	}
}

func TestJobSignoffCaptureAndReplace(t *testing.T) {
	f := newSignoffFixture(t)
	got, err := f.sign(t, f.ctxA, f.jobA)
	require.NoError(t, err)
	assert.Equal(t, "Ray Burke", got.SignerName, "trimmed, control characters dropped")
	assert.Equal(t, "Site manager", *got.SignerTitle)
	assert.Equal(t, "ray.burke@harborview.example", *got.SignerEmail)
	assert.Equal(t, []string{"karen@harborview.example", "ray.burke@harborview.example"}, got.SuggestedRecipients,
		"customer, site contact (same address, deduplicated) and signer")
	assert.NotEmpty(t, got.SignatureURL)
	assert.False(t, got.Locked)
	row := f.repo.rows[f.jobA]
	assert.Equal(t, "203.0.113.9", *row.IPAddress)
	assert.Equal(t, "Mozilla/5.0 (iPhone)", *row.UserAgent)
	assert.Equal(t, f.user, *row.CapturedBy)
	assert.True(t, strings.HasPrefix(row.SignatureKey, "job-signoffs/"+f.tenantA.String()+"/"+f.jobA.String()+"/"))
	first := row.SignatureKey

	// Signing again replaces the sign-off and removes the old signature.
	again, err := f.sign(t, f.ctxA, f.jobA)
	require.NoError(t, err)
	assert.Equal(t, got.ID, again.ID)
	objects := f.store.GetAllObjects()
	assert.Len(t, objects, 1)
	_, stillThere := objects[first]
	assert.False(t, stillThere)
}

func TestJobSignoffRefusals(t *testing.T) {
	f := newSignoffFixture(t)

	// Another tenant's job doesn't exist.
	_, err := f.sign(t, f.ctxB, f.jobA)
	assert.Equal(t, http.StatusNotFound, statusOf(t, err))
	_, err = f.svc.Get(f.ctxB, f.jobA)
	assert.Equal(t, http.StatusNotFound, statusOf(t, err))

	// A visit that hasn't started can't be signed.
	f.repo.jobs[f.tenantA][f.jobA].Job.Status = models.JobStatusScheduled
	_, err = f.sign(t, f.ctxA, f.jobA)
	assert.Equal(t, http.StatusConflict, statusOf(t, err))
	f.repo.jobs[f.tenantA][f.jobA].Job.Status = models.JobStatusCompleted

	// Blank name, bad email.
	_, err = f.svc.Capture(f.ctxA, f.jobA, f.user, &models.CaptureJobSignoffRequest{SignerName: " \t ", Signature: "x"}, "", "")
	assert.Equal(t, http.StatusBadRequest, statusOf(t, err))
	bad := "not an email"
	_, err = f.svc.Capture(f.ctxA, f.jobA, f.user, &models.CaptureJobSignoffRequest{
		SignerName: "Ray", SignerEmail: &bad, Signature: dataURL(signaturePNG(t, 900, 300, true))}, "", "")
	assert.Equal(t, http.StatusBadRequest, statusOf(t, err))
	assert.Empty(t, f.store.GetAllObjects(), "nothing stored for refused captures")

	// No sign-off yet: reading it is null, its files are 404.
	env, err := f.svc.Get(f.ctxA, f.jobA)
	require.NoError(t, err)
	assert.Nil(t, env.Signoff)
	assert.Equal(t, []string{"karen@harborview.example"}, env.SuggestedRecipients)
	_, err = f.svc.ReportLink(f.ctxA, f.jobA)
	assert.Equal(t, http.StatusNotFound, statusOf(t, err))
	_, err = f.svc.EmailReport(f.ctxA, f.jobA, false)
	assert.Equal(t, http.StatusNotFound, statusOf(t, err))
}

func TestJobSignoffEmailIsIdempotent(t *testing.T) {
	f := newSignoffFixture(t)
	_, err := f.sign(t, f.ctxA, f.jobA)
	require.NoError(t, err)

	res, err := f.svc.EmailReport(f.ctxA, f.jobA, false)
	require.NoError(t, err)
	assert.True(t, res.Sent)
	require.Len(t, f.mail.SentEmails, 1)
	msg := f.mail.SentEmails[0]
	assert.Equal(t, []string{"karen@harborview.example", "ray.burke@harborview.example"}, msg.To)
	assert.Equal(t, "Service report: Elm Street Plaza (J-3318)", msg.Subject)
	require.Len(t, msg.Attachments, 1)
	assert.Equal(t, "Service-Report-J-3318.pdf", msg.Attachments[0].Filename)
	assert.True(t, bytes.HasPrefix(msg.Attachments[0].Content, []byte("%PDF")))
	assert.Contains(t, msg.HTMLBody, "Ray Burke, Site manager signed off on Mon, Nov 2 at 6:58 PM EST")
	assert.True(t, res.Signoff.Locked)
	assert.Equal(t, 1, res.Signoff.ReportEmailCount)
	assert.Equal(t, msg.To, res.Signoff.ReportRecipients)

	// A retried request sends nothing.
	res, err = f.svc.EmailReport(f.ctxA, f.jobA, false)
	require.NoError(t, err)
	assert.False(t, res.Sent)
	assert.Len(t, f.mail.SentEmails, 1)

	// Once emailed the sign-off is the record: it can't be replaced.
	_, err = f.sign(t, f.ctxA, f.jobA)
	assert.Equal(t, http.StatusConflict, statusOf(t, err))

	// An explicit resend sends the stored report again.
	stored := f.store.GetAllObjects()[*f.repo.rows[f.jobA].ReportKey]
	res, err = f.svc.EmailReport(f.ctxA, f.jobA, true)
	require.NoError(t, err)
	assert.True(t, res.Sent)
	require.Len(t, f.mail.SentEmails, 2)
	assert.Equal(t, stored, f.mail.SentEmails[1].Attachments[0].Content, "the same document as before")
	assert.Equal(t, 2, res.Signoff.ReportEmailCount)

	// Another tenant can't send it.
	_, err = f.svc.EmailReport(f.ctxB, f.jobA, true)
	assert.Equal(t, http.StatusNotFound, statusOf(t, err))
}

func TestJobSignoffEmailClaimAndFailure(t *testing.T) {
	f := newSignoffFixture(t)
	_, err := f.sign(t, f.ctxA, f.jobA)
	require.NoError(t, err)

	// A send in flight (claimed a minute ago) blocks a second one.
	claimed := f.svc.now().Add(-time.Minute)
	f.repo.rows[f.jobA].ReportEmailClaimedAt = &claimed
	_, err = f.svc.EmailReport(f.ctxA, f.jobA, false)
	assert.Equal(t, http.StatusConflict, statusOf(t, err))
	assert.Empty(t, f.mail.SentEmails)

	// A stale claim (crashed sender) is taken over.
	stale := f.svc.now().Add(-time.Hour)
	f.repo.rows[f.jobA].ReportEmailClaimedAt = &stale

	// The provider fails: the claim is released so a retry can send.
	f.svc.email = failingEmail{f.mail}
	_, err = f.svc.EmailReport(f.ctxA, f.jobA, false)
	assert.Equal(t, http.StatusBadGateway, statusOf(t, err))
	assert.Nil(t, f.repo.rows[f.jobA].ReportEmailClaimedAt)
	assert.Nil(t, f.repo.rows[f.jobA].ReportEmailedAt)

	f.svc.email = f.mail
	res, err := f.svc.EmailReport(f.ctxA, f.jobA, false)
	require.NoError(t, err)
	assert.True(t, res.Sent)

	// Opted-out customer without a signer email: nobody to send to.
	f.repo.contacts[f.jobB] = &repository.JobContacts{CustomerEmail: "b@example.com", DoNotEmail: true}
	f.repo.jobs[f.tenantB][f.jobB].Job.Status = models.JobStatusCompleted
	_, err = f.svc.Capture(f.ctxB, f.jobB, f.user, &models.CaptureJobSignoffRequest{
		SignerName: "Bo", Signature: dataURL(signaturePNG(t, 900, 300, true))}, "", "")
	require.NoError(t, err)
	_, err = f.svc.EmailReport(f.ctxB, f.jobB, false)
	assert.Equal(t, http.StatusConflict, statusOf(t, err))
}

// The report renders with photos and the signature, in the tenant's time
// zone, and never carries prices.
func TestJobSignoffReportPDF(t *testing.T) {
	f := newSignoffFixture(t)
	_, err := f.sign(t, f.ctxA, f.jobA)
	require.NoError(t, err)
	// Two photos with thumbnails, one whose object is missing.
	src := f.repo.jobs[f.tenantA][f.jobA]
	for i, name := range []string{"Lobby floor", "Restroom 2F", "Missing"} {
		thumb := "job-attachments/thumb-" + name + ".jpg"
		if i < 2 {
			_, err := f.store.UploadBytes(context.Background(), testJPEG(t, 400, 300), storage.NewUploadInput(thumb))
			require.NoError(t, err)
		}
		caption := name
		src.Photos = append(src.Photos, models.JobAttachment{ID: uuid.New(), ObjectKey: "x", ThumbKey: &thumb, Caption: &caption,
			CreatedAt: time.Date(2026, 11, 2, 22, 41, 0, 0, time.UTC)})
	}
	so := f.repo.rows[f.jobA]
	so.Capturer = &models.User{Email: "tasha@example.com"}

	data := f.svc.reportData(context.Background(), src, so)
	assert.Equal(t, "Mon, Nov 2, 2026", data.VisitDate)
	assert.Equal(t, "5:30 PM - 7:00 PM", data.Scheduled)
	assert.Equal(t, "5:32 PM - 6:58 PM (1 h 26 min)", data.OnSite)
	assert.Equal(t, "Tasha Green", data.Technicians)
	assert.Equal(t, "Elm Street Plaza", data.SiteName)
	assert.Equal(t, "410 Elm St, Columbus, OH 43215", data.SiteAddress)
	assert.Equal(t, []pdf.ProofOfServiceMaterial{{Name: "Trash liners 33 gal", Quantity: "12 each"}}, data.Materials)
	assert.Len(t, data.Photos, 2)
	assert.Equal(t, 1, data.MorePhotos, "a photo that can't be loaded is counted, not fatal")
	assert.Equal(t, "5:41 PM", data.Photos[0].Taken)
	require.NotNil(t, data.Signature)
	assert.NotEmpty(t, data.Signature.PNG)
	assert.Equal(t, "On tasha@example.com's device", data.Signature.Device)

	rep, key, err := f.svc.buildAndStore(f.ctxA, f.tenantA, so)
	require.NoError(t, err)
	assert.True(t, bytes.HasPrefix(rep.Content, []byte("%PDF")))
	assert.Greater(t, len(rep.Content), 5000, "photos and signature are embedded")
	assert.Equal(t, rep.Content, f.store.GetAllObjects()[key])
	for _, text := range []string{"Service report", "J-3318", "Ray Burke, Site manager", "Trash liners 33 gal", "12 each",
		"5:32 PM - 6:58 PM \\(1 h 26 min\\)", "Lobby floor", "Page 1/2", "Page 2/2", "Entry mat by the east door"} {
		assert.Contains(t, string(rep.Content), text)
	}
	for _, price := range []string{"4321.98", "4,321.98", "999.99"} {
		assert.NotContains(t, string(rep.Content), price)
	}
	link, err := f.svc.ReportLink(f.ctxA, f.jobA)
	require.NoError(t, err)
	assert.NotEmpty(t, link)
}
