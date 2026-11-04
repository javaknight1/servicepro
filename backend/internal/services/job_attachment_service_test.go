package services

import (
	"bytes"
	"context"
	"encoding/binary"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"net/http"
	"sort"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apperrors "github.com/javaknight1/servicepro/backend/internal/errors"
	"github.com/javaknight1/servicepro/backend/internal/models"
	"github.com/javaknight1/servicepro/backend/internal/repository"
	"github.com/javaknight1/servicepro/backend/internal/tenancy"
	"github.com/javaknight1/servicepro/backend/pkg/clients/storage/mock"
)

// fakeAttachmentRepo is an in-memory JobAttachmentRepositoryInterface that
// scopes by tenant like the real one; visible lists the jobs the caller can
// see per tenant.
type fakeAttachmentRepo struct {
	visible map[uuid.UUID]map[uuid.UUID]bool
	rows    []models.JobAttachment
}

func (f *fakeAttachmentRepo) JobVisible(_ context.Context, tenantID, jobID uuid.UUID) (bool, error) {
	return f.visible[tenantID][jobID], nil
}

func (f *fakeAttachmentRepo) Create(_ context.Context, tenantID uuid.UUID, a *models.JobAttachment) error {
	a.TenantID = tenantID
	f.rows = append(f.rows, *a)
	return nil
}

func (f *fakeAttachmentRepo) find(tenantID, jobID, id uuid.UUID) int {
	for i, r := range f.rows {
		if r.TenantID == tenantID && r.JobID == jobID && r.ID == id {
			return i
		}
	}
	return -1
}

func (f *fakeAttachmentRepo) Get(_ context.Context, tenantID, jobID, id uuid.UUID) (*models.JobAttachment, error) {
	i := f.find(tenantID, jobID, id)
	if i < 0 {
		return nil, repository.ErrJobAttachmentNotFound
	}
	a := f.rows[i]
	return &a, nil
}

func (f *fakeAttachmentRepo) ListByJob(_ context.Context, tenantID, jobID uuid.UUID) ([]models.JobAttachment, error) {
	var out []models.JobAttachment
	for _, r := range f.rows {
		if r.TenantID == tenantID && r.JobID == jobID {
			out = append(out, r)
		}
	}
	return out, nil
}

func (f *fakeAttachmentRepo) CountByJob(ctx context.Context, tenantID, jobID uuid.UUID) (int64, error) {
	rows, _ := f.ListByJob(ctx, tenantID, jobID)
	return int64(len(rows)), nil
}

func (f *fakeAttachmentRepo) UpdateCaption(_ context.Context, tenantID, jobID, id uuid.UUID, caption *string) error {
	i := f.find(tenantID, jobID, id)
	if i < 0 {
		return repository.ErrJobAttachmentNotFound
	}
	f.rows[i].Caption = caption
	return nil
}

func (f *fakeAttachmentRepo) Delete(_ context.Context, tenantID, jobID, id uuid.UUID) error {
	i := f.find(tenantID, jobID, id)
	if i < 0 {
		return repository.ErrJobAttachmentNotFound
	}
	f.rows = append(f.rows[:i], f.rows[i+1:]...)
	return nil
}

type attachmentFixture struct {
	svc              *JobAttachmentService
	repo             *fakeAttachmentRepo
	store            *mock.Client
	tenantA, tenantB uuid.UUID
	jobA, jobB       uuid.UUID
	user             uuid.UUID
	ctxA, ctxB       context.Context
}

func newAttachmentFixture() *attachmentFixture {
	f := &attachmentFixture{
		tenantA: uuid.New(), tenantB: uuid.New(), jobA: uuid.New(), jobB: uuid.New(), user: uuid.New(),
		store: mock.NewClient(nil),
	}
	f.repo = &fakeAttachmentRepo{visible: map[uuid.UUID]map[uuid.UUID]bool{
		f.tenantA: {f.jobA: true},
		f.tenantB: {f.jobB: true},
	}}
	f.svc = NewJobAttachmentService(f.repo, f.store)
	f.ctxA = tenancy.WithTenant(context.Background(), f.tenantA)
	f.ctxB = tenancy.WithTenant(context.Background(), f.tenantB)
	return f
}

// testJPEG encodes a w x h JPEG whose top-left 1/4 x 1/4 block is red and
// the rest blue.
func testJPEG(t *testing.T, w, h int) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			c := color.RGBA{0, 0, 255, 255}
			if x < w/4 && y < h/4 {
				c = color.RGBA{255, 0, 0, 255}
			}
			img.Set(x, y, c)
		}
	}
	var buf bytes.Buffer
	require.NoError(t, jpeg.Encode(&buf, img, &jpeg.Options{Quality: 95}))
	return buf.Bytes()
}

// withEXIF splices an APP1 EXIF segment (orientation tag + a fake GPS
// marker string) right after the JPEG's SOI.
func withEXIF(t *testing.T, data []byte, orientation uint16) []byte {
	t.Helper()
	var tiff bytes.Buffer
	tiff.WriteString("MM")
	_ = binary.Write(&tiff, binary.BigEndian, uint16(42))
	_ = binary.Write(&tiff, binary.BigEndian, uint32(8))
	_ = binary.Write(&tiff, binary.BigEndian, uint16(1)) // one entry
	_ = binary.Write(&tiff, binary.BigEndian, uint16(0x0112))
	_ = binary.Write(&tiff, binary.BigEndian, uint16(3)) // SHORT
	_ = binary.Write(&tiff, binary.BigEndian, uint32(1))
	_ = binary.Write(&tiff, binary.BigEndian, orientation)
	_ = binary.Write(&tiff, binary.BigEndian, uint16(0))
	_ = binary.Write(&tiff, binary.BigEndian, uint32(0)) // no next IFD
	tiff.WriteString("GPS-SECRET-41.4993N")
	payload := append([]byte("Exif\x00\x00"), tiff.Bytes()...)
	seg := []byte{0xFF, 0xE1}
	seg = binary.BigEndian.AppendUint16(seg, uint16(len(payload)+2))
	seg = append(seg, payload...)
	out := append([]byte{}, data[:2]...)
	out = append(out, seg...)
	return append(out, data[2:]...)
}

func upload(f *attachmentFixture, ctx context.Context, job uuid.UUID, name string, data []byte) (*models.JobAttachmentResponse, error) {
	return f.svc.Upload(ctx, job, f.user, name, bytes.NewReader(data), nil)
}

func statusOf(t *testing.T, err error) int {
	t.Helper()
	e, ok := apperrors.As(err)
	require.True(t, ok, "want a domain error, got %v", err)
	return e.Status
}

func TestJobAttachmentUploadPhotoStripsEXIFAndRotates(t *testing.T) {
	f := newAttachmentFixture()
	data := withEXIF(t, testJPEG(t, 400, 200), 6)
	require.Equal(t, 6, jpegOrientation(data))

	caption := "  Lobby floor  "
	got, err := f.svc.Upload(f.ctxA, f.jobA, f.user, "../../IMG_2041.jpg", bytes.NewReader(data), &caption)
	require.NoError(t, err)
	assert.Equal(t, models.JobAttachmentPhoto, got.Kind)
	assert.Equal(t, "image/jpeg", got.ContentType)
	assert.Equal(t, "IMG_2041.jpg", got.Filename, "path parts are dropped")
	require.NotNil(t, got.Caption)
	assert.Equal(t, "Lobby floor", *got.Caption)
	require.NotNil(t, got.Width)
	assert.Equal(t, 200, *got.Width, "orientation 6 turns 400x200 upright to 200x400")
	assert.Equal(t, 400, *got.Height)
	assert.NotEmpty(t, got.URL)
	assert.NotEmpty(t, got.ThumbURL)
	assert.Equal(t, "/api/v1/jobs/"+f.jobA.String()+"/attachments/"+got.ID.String()+"/file", got.FilePath)
	assert.NotEmpty(t, got.ThumbPath)

	objects := f.store.GetAllObjects()
	require.Len(t, objects, 2)
	row := f.repo.rows[0]
	assert.True(t, strings.HasPrefix(row.ObjectKey, "job-attachments/"+f.tenantA.String()+"/"+f.jobA.String()+"/"))
	stored := objects[row.ObjectKey]
	assert.NotContains(t, string(stored), "GPS-SECRET", "EXIF is stripped")
	assert.Equal(t, 1, jpegOrientation(stored))
	img, err := jpeg.Decode(bytes.NewReader(stored))
	require.NoError(t, err)
	assert.Equal(t, image.Pt(200, 400), img.Bounds().Size())
	// The red block was top-left; turned 90 degrees clockwise it is top-right.
	r, _, b, _ := img.At(190, 10).RGBA()
	assert.Greater(t, r, b, "top-right is red")
	r, _, b, _ = img.At(10, 10).RGBA()
	assert.Greater(t, b, r, "top-left is blue")
	assert.Equal(t, int64(len(stored)), got.SizeBytes)
}

func TestJobAttachmentLargePhotoIsScaledWithThumbnail(t *testing.T) {
	f := newAttachmentFixture()
	img := image.NewNRGBA(image.Rect(0, 0, 3000, 1500))
	var buf bytes.Buffer
	require.NoError(t, png.Encode(&buf, img))
	got, err := upload(f, f.ctxA, f.jobA, "scan.png", buf.Bytes())
	require.NoError(t, err)
	assert.Equal(t, AttachmentMaxSide, *got.Width)
	assert.Equal(t, 1024, *got.Height)
	thumb, err := jpeg.Decode(bytes.NewReader(f.store.GetAllObjects()[*f.repo.rows[0].ThumbKey]))
	require.NoError(t, err)
	assert.Equal(t, image.Pt(400, 200), thumb.Bounds().Size())
	assert.Equal(t, "image/jpeg", got.ContentType, "PNGs are stored as JPEG")
}

func TestJobAttachmentPDFStoredAsFile(t *testing.T) {
	f := newAttachmentFixture()
	pdf := []byte("%PDF-1.4\n1 0 obj << >> endobj\ntrailer << >>\n%%EOF\n")
	got, err := upload(f, f.ctxA, f.jobA, "work-order.pdf", pdf)
	require.NoError(t, err)
	assert.Equal(t, models.JobAttachmentFile, got.Kind)
	assert.Equal(t, "application/pdf", got.ContentType)
	assert.Empty(t, got.ThumbURL)
	assert.Empty(t, got.ThumbPath)
	assert.Nil(t, got.Width)
	assert.Equal(t, pdf, f.store.GetAllObjects()[f.repo.rows[0].ObjectKey])

	_, err = f.svc.Link(f.ctxA, f.jobA, got.ID, true)
	assert.ErrorIs(t, err, ErrAttachmentNoThumbnail)
}

func TestJobAttachmentRejectsBadUploads(t *testing.T) {
	f := newAttachmentFixture()
	cases := []struct {
		name   string
		data   []byte
		want   error
		status int
	}{
		{"html sniffed, not trusted by name", []byte("<html><script>alert(1)</script></html>"), ErrAttachmentType, http.StatusUnsupportedMediaType},
		{"gif", []byte("GIF89a\x01\x00\x01\x00\x00\x00\x00;"), ErrAttachmentType, http.StatusUnsupportedMediaType},
		{"empty", nil, ErrAttachmentEmpty, http.StatusBadRequest},
		{"truncated jpeg", testJPEG(t, 64, 64)[:40], ErrAttachmentUnreadable, http.StatusBadRequest},
		{"decompression bomb", pngHeaderOnly(50000, 50000), ErrAttachmentDimensions, http.StatusBadRequest},
		{"too many pixels within the side limit", pngHeaderOnly(10000, 10000), ErrAttachmentDimensions, http.StatusBadRequest},
		{"oversized", append([]byte("%PDF-1.4\n"), make([]byte, MaxAttachmentBytes)...), ErrAttachmentTooLarge, http.StatusRequestEntityTooLarge},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := upload(f, f.ctxA, f.jobA, "photo.jpg", tc.data)
			require.ErrorIs(t, err, tc.want)
			assert.Equal(t, tc.status, statusOf(t, err))
		})
	}
	assert.Empty(t, f.store.GetAllObjects(), "nothing is stored for a refused upload")
	assert.Empty(t, f.repo.rows)

	long := strings.Repeat("x", 501)
	_, err := f.svc.Upload(f.ctxA, f.jobA, f.user, "a.jpg", bytes.NewReader(testJPEG(t, 8, 8)), &long)
	assert.ErrorIs(t, err, ErrAttachmentCaption)
}

func TestJobAttachmentLimitPerJob(t *testing.T) {
	f := newAttachmentFixture()
	for i := 0; i < MaxAttachmentsPerJob; i++ {
		f.repo.rows = append(f.repo.rows, models.JobAttachment{ID: uuid.New(), TenantID: f.tenantA, JobID: f.jobA})
	}
	_, err := upload(f, f.ctxA, f.jobA, "a.jpg", testJPEG(t, 8, 8))
	assert.ErrorIs(t, err, ErrAttachmentLimit)
}

func TestJobAttachmentCrossTenant(t *testing.T) {
	f := newAttachmentFixture()
	got, err := upload(f, f.ctxA, f.jobA, "a.jpg", testJPEG(t, 8, 8))
	require.NoError(t, err)

	// Tenant B can't see tenant A's job or reach the attachment through its own job.
	_, err = f.svc.List(f.ctxB, f.jobA)
	assert.ErrorIs(t, err, ErrAttachmentJobNotFound)
	_, err = upload(f, f.ctxB, f.jobA, "b.jpg", testJPEG(t, 8, 8))
	assert.ErrorIs(t, err, ErrAttachmentJobNotFound)
	_, err = f.svc.Link(f.ctxB, f.jobB, got.ID, false)
	assert.ErrorIs(t, err, ErrJobAttachmentNotFound)
	assert.ErrorIs(t, f.svc.Delete(f.ctxB, f.jobB, got.ID, f.user), ErrJobAttachmentNotFound)
	caption := "pwned"
	_, err = f.svc.UpdateCaption(f.ctxB, f.jobB, got.ID, f.user, &models.UpdateJobAttachmentRequest{Caption: &caption})
	assert.ErrorIs(t, err, ErrJobAttachmentNotFound)
	list, err := f.svc.List(f.ctxB, f.jobB)
	require.NoError(t, err)
	assert.Empty(t, list)

	// Without a tenant nothing works.
	_, err = f.svc.List(context.Background(), f.jobA)
	assert.ErrorIs(t, err, tenancy.ErrNoTenant)

	list, err = f.svc.List(f.ctxA, f.jobA)
	require.NoError(t, err)
	require.Len(t, list, 1)
	assert.Equal(t, got.ID, list[0].ID)
}

func TestJobAttachmentTechnicianChangesOnlyOwn(t *testing.T) {
	f := newAttachmentFixture()
	office := f.user
	byOffice, err := upload(f, f.ctxA, f.jobA, "a.jpg", testJPEG(t, 8, 8))
	require.NoError(t, err)

	tech := uuid.New()
	techCtx := tenancy.WithAssignedJobsOnly(f.ctxA, tech)
	byTech, err := f.svc.Upload(techCtx, f.jobA, tech, "b.jpg", bytes.NewReader(testJPEG(t, 8, 8)), nil)
	require.NoError(t, err)

	caption := "Restroom 2F"
	_, err = f.svc.UpdateCaption(techCtx, f.jobA, byOffice.ID, tech, &models.UpdateJobAttachmentRequest{Caption: &caption})
	assert.ErrorIs(t, err, ErrAttachmentNotYours)
	assert.ErrorIs(t, f.svc.Delete(techCtx, f.jobA, byOffice.ID, tech), ErrAttachmentNotYours)

	updated, err := f.svc.UpdateCaption(techCtx, f.jobA, byTech.ID, tech, &models.UpdateJobAttachmentRequest{Caption: &caption})
	require.NoError(t, err)
	assert.Equal(t, caption, *updated.Caption)
	empty := " "
	updated, err = f.svc.UpdateCaption(techCtx, f.jobA, byTech.ID, tech, &models.UpdateJobAttachmentRequest{Caption: &empty})
	require.NoError(t, err)
	assert.Nil(t, updated.Caption, "a blank caption clears it")
	require.NoError(t, f.svc.Delete(techCtx, f.jobA, byTech.ID, tech))

	// Office staff (not limited to assigned jobs) delete anyone's; the
	// stored objects go with the row.
	require.NoError(t, f.svc.Delete(f.ctxA, f.jobA, byOffice.ID, office))
	assert.Empty(t, f.repo.rows)
	assert.Empty(t, f.store.GetAllObjects())
}

func TestJobAttachmentListOldestFirstWithLinks(t *testing.T) {
	f := newAttachmentFixture()
	for _, n := range []string{"1.jpg", "2.jpg"} {
		_, err := upload(f, f.ctxA, f.jobA, n, testJPEG(t, 8, 8))
		require.NoError(t, err)
	}
	list, err := f.svc.List(f.ctxA, f.jobA)
	require.NoError(t, err)
	require.Len(t, list, 2)
	names := []string{list[0].Filename, list[1].Filename}
	assert.True(t, sort.StringsAreSorted(names))
	for _, a := range list {
		assert.NotEmpty(t, a.URL)
		assert.NotEmpty(t, a.ThumbURL)
	}
	link, err := f.svc.Link(f.ctxA, f.jobA, list[0].ID, true)
	require.NoError(t, err)
	assert.Contains(t, link, "_thumb.jpg")
}

func TestJPEGOrientationParser(t *testing.T) {
	base := testJPEG(t, 8, 8)
	assert.Equal(t, 1, jpegOrientation(base))
	for o := uint16(1); o <= 8; o++ {
		assert.Equal(t, int(o), jpegOrientation(withEXIF(t, base, o)))
	}
	assert.Equal(t, 1, jpegOrientation(withEXIF(t, base, 9)), "out of range")
	assert.Equal(t, 1, jpegOrientation([]byte{0xFF, 0xD8, 0xFF, 0xE1, 0xFF, 0xFF}), "truncated segment")
	assert.Equal(t, 1, jpegOrientation(nil))
}

func TestApplyOrientationAllEight(t *testing.T) {
	// 3x2 source, pixel value = index so positions can be tracked.
	src := image.NewRGBA(image.Rect(0, 0, 3, 2))
	for y := 0; y < 2; y++ {
		for x := 0; x < 3; x++ {
			src.SetRGBA(x, y, color.RGBA{uint8(y*3 + x), 0, 0, 255})
		}
	}
	at := func(img *image.RGBA, x, y int) uint8 { return img.RGBAAt(x, y).R }
	// The source's top-left pixel (0) ends up at these positions.
	want := map[int]image.Point{1: {0, 0}, 2: {2, 0}, 3: {2, 1}, 4: {0, 1}, 5: {0, 0}, 6: {1, 0}, 7: {1, 2}, 8: {0, 2}}
	for o, p := range want {
		out := applyOrientation(src, o)
		if o >= 5 {
			assert.Equal(t, image.Pt(2, 3), out.Bounds().Size(), "orientation %d", o)
		}
		assert.Equal(t, uint8(0), at(out, p.X, p.Y), "orientation %d", o)
	}
}
