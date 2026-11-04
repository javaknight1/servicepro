package services

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/google/uuid"

	apperrors "github.com/javaknight1/servicepro/backend/internal/errors"
	"github.com/javaknight1/servicepro/backend/internal/models"
	"github.com/javaknight1/servicepro/backend/internal/repository"
	"github.com/javaknight1/servicepro/backend/internal/tenancy"
	"github.com/javaknight1/servicepro/backend/pkg/clients/logging"
	"github.com/javaknight1/servicepro/backend/pkg/clients/storage"
)

// Job photos and files (T174). Uploads go through the API (multipart), not
// presigned PUTs: the server has to see the bytes anyway to sniff the type,
// check dimensions, strip EXIF and make the thumbnail, and one request is
// simpler for a phone on a weak connection than PUT-then-confirm.

const (
	// MaxAttachmentBytes caps one upload (phone photos are 2-8 MB).
	MaxAttachmentBytes = 15 << 20
	// MaxAttachmentsPerJob bounds what one job can hold.
	MaxAttachmentsPerJob = 100
	// AttachmentLinkTTL is how long a presigned attachment link lives.
	AttachmentLinkTTL = time.Hour
	// attachmentPrefix is the object-key prefix in the private bucket.
	attachmentPrefix = "job-attachments"
	// maxCaptionLength matches job_attachments.caption.
	maxCaptionLength = 500
)

var (
	// ErrAttachmentJobNotFound: the job isn't in the tenant, or the caller is
	// limited to assigned jobs and isn't assigned to it.
	ErrAttachmentJobNotFound = apperrors.NotFound("job_not_found", "Job not found")
	// ErrJobAttachmentNotFound: no such attachment on that job.
	ErrJobAttachmentNotFound = apperrors.NotFound("attachment_not_found", "Attachment not found")
	// ErrAttachmentNoThumbnail: files (PDFs) have no thumbnail.
	ErrAttachmentNoThumbnail = apperrors.NotFound("thumbnail_not_found", "This attachment has no thumbnail")
	// ErrAttachmentTooLarge: over MaxAttachmentBytes.
	ErrAttachmentTooLarge = apperrors.New(http.StatusRequestEntityTooLarge, "file_too_large", "The file is larger than 15 MB")
	// ErrAttachmentType: the bytes are not JPEG, PNG, WebP or PDF.
	ErrAttachmentType = apperrors.New(http.StatusUnsupportedMediaType, "unsupported_file_type", "Only JPEG, PNG and WebP photos and PDF files can be attached")
	// ErrAttachmentDimensions: the photo declares too many pixels.
	ErrAttachmentDimensions = apperrors.BadRequest("image_too_large", "The photo is too large (at most 12000 pixels a side and 40 megapixels)")
	// ErrAttachmentUnreadable: the photo's bytes don't decode.
	ErrAttachmentUnreadable = apperrors.BadRequest("invalid_image", "The photo could not be read")
	// ErrAttachmentEmpty: no bytes.
	ErrAttachmentEmpty = apperrors.BadRequest("empty_file", "The file is empty")
	// ErrAttachmentLimit: the job already holds MaxAttachmentsPerJob.
	ErrAttachmentLimit = apperrors.Conflict("attachment_limit", "This job already has 100 attachments")
	// ErrAttachmentNotYours: callers limited to their assigned jobs
	// (technicians) may change or delete only what they uploaded.
	ErrAttachmentNotYours = apperrors.Forbidden("attachment_not_yours", "You can only change photos you added")
	// ErrAttachmentCaption: caption over 500 characters.
	ErrAttachmentCaption = apperrors.Validation("Caption is too long", map[string]string{"caption": "at most 500 characters"})
)

// JobAttachmentServiceInterface is the job-attachment use case the handler
// calls. The tenant (and any technician row filter) always comes from ctx.
type JobAttachmentServiceInterface interface {
	Upload(ctx context.Context, jobID, userID uuid.UUID, filename string, body io.Reader, caption *string) (*models.JobAttachmentResponse, error)
	List(ctx context.Context, jobID uuid.UUID) ([]models.JobAttachmentResponse, error)
	// Link returns a presigned link to the stored file, or to its thumbnail.
	Link(ctx context.Context, jobID, id uuid.UUID, thumb bool) (string, error)
	UpdateCaption(ctx context.Context, jobID, id, userID uuid.UUID, req *models.UpdateJobAttachmentRequest) (*models.JobAttachmentResponse, error)
	Delete(ctx context.Context, jobID, id, userID uuid.UUID) error
}

// JobAttachmentService stores job photos and files in the object store.
type JobAttachmentService struct {
	repo    repository.JobAttachmentRepositoryInterface
	storage storage.Client
	// decode bounds concurrent photo decodes: each can hold up to
	// MaxAttachmentPixels * 4 bytes.
	decode chan struct{}
}

// NewJobAttachmentService creates the service.
func NewJobAttachmentService(repo repository.JobAttachmentRepositoryInterface, storageClient storage.Client) *JobAttachmentService {
	return &JobAttachmentService{repo: repo, storage: storageClient, decode: make(chan struct{}, 2)}
}

// visibleJob resolves the tenant and checks the caller can see the job.
func (s *JobAttachmentService) visibleJob(ctx context.Context, jobID uuid.UUID) (uuid.UUID, error) {
	tenantID, err := tenancy.MustTenantID(ctx)
	if err != nil {
		return uuid.Nil, err
	}
	ok, err := s.repo.JobVisible(ctx, tenantID, jobID)
	if err != nil {
		return uuid.Nil, fmt.Errorf("check job: %w", err)
	}
	if !ok {
		return uuid.Nil, ErrAttachmentJobNotFound
	}
	return tenantID, nil
}

func (s *JobAttachmentService) get(ctx context.Context, jobID, id uuid.UUID) (uuid.UUID, *models.JobAttachment, error) {
	tenantID, err := s.visibleJob(ctx, jobID)
	if err != nil {
		return uuid.Nil, nil, err
	}
	a, err := s.repo.Get(ctx, tenantID, jobID, id)
	if errors.Is(err, repository.ErrJobAttachmentNotFound) {
		return uuid.Nil, nil, ErrJobAttachmentNotFound
	}
	if err != nil {
		return uuid.Nil, nil, err
	}
	return tenantID, a, nil
}

// mayChange: office staff (who see every job) change any attachment;
// callers limited to their assigned jobs only their own.
func mayChange(ctx context.Context, a *models.JobAttachment, userID uuid.UUID) bool {
	if _, limited := tenancy.AssignedJobsUser(ctx); !limited {
		return true
	}
	return a.UploadedBy != nil && *a.UploadedBy == userID
}

// cleanFilename keeps the base name, without control characters, at most
// 255 bytes.
func cleanFilename(name string) string {
	name = filepath.Base(strings.ReplaceAll(name, "\\", "/"))
	name = strings.Map(func(r rune) rune {
		if unicode.IsControl(r) {
			return -1
		}
		return r
	}, strings.TrimSpace(name))
	if name == "." || name == "/" {
		name = ""
	}
	for len(name) > 255 {
		_, size := utf8.DecodeLastRuneInString(name)
		name = name[:len(name)-size]
	}
	return name
}

// cleanCaption trims; empty means none.
func cleanCaption(caption *string) (*string, error) {
	if caption == nil {
		return nil, nil
	}
	c := strings.TrimSpace(*caption)
	if c == "" {
		return nil, nil
	}
	if utf8.RuneCountInString(c) > maxCaptionLength {
		return nil, ErrAttachmentCaption
	}
	return &c, nil
}

// Upload stores a photo or PDF on the job.
func (s *JobAttachmentService) Upload(ctx context.Context, jobID, userID uuid.UUID, filename string, body io.Reader, caption *string) (*models.JobAttachmentResponse, error) {
	caption, err := cleanCaption(caption)
	if err != nil {
		return nil, err
	}
	tenantID, err := s.visibleJob(ctx, jobID)
	if err != nil {
		return nil, err
	}
	n, err := s.repo.CountByJob(ctx, tenantID, jobID)
	if err != nil {
		return nil, err
	}
	if n >= MaxAttachmentsPerJob {
		return nil, ErrAttachmentLimit
	}

	data, err := io.ReadAll(io.LimitReader(body, MaxAttachmentBytes+1))
	if err != nil {
		var tooBig *http.MaxBytesError
		if errors.As(err, &tooBig) {
			return nil, ErrAttachmentTooLarge
		}
		return nil, fmt.Errorf("read upload: %w", err)
	}
	if len(data) > MaxAttachmentBytes {
		return nil, ErrAttachmentTooLarge
	}
	if len(data) == 0 {
		return nil, ErrAttachmentEmpty
	}
	contentType := sniffAttachment(data)
	if contentType == "" {
		return nil, ErrAttachmentType
	}

	id := uuid.New()
	base := fmt.Sprintf("%s/%s/%s/%s", attachmentPrefix, tenantID, jobID, id)
	a := &models.JobAttachment{
		ID: id, JobID: jobID, UploadedBy: &userID,
		OriginalFilename: cleanFilename(filename), Caption: caption,
	}
	objects := map[string][]byte{}
	if contentType == contentTypePDF {
		a.Kind = models.JobAttachmentFile
		a.ContentType = contentTypePDF
		a.ObjectKey = base + ".pdf"
		objects[a.ObjectKey] = data
	} else {
		photo, err := s.process(ctx, data)
		if err != nil {
			return nil, err
		}
		thumbKey := base + "_thumb.jpg"
		a.Kind = models.JobAttachmentPhoto
		a.ContentType = contentTypeJPEG
		a.ObjectKey = base + ".jpg"
		a.ThumbKey = &thumbKey
		a.Width, a.Height = &photo.Width, &photo.Height
		objects[a.ObjectKey] = photo.Full
		objects[thumbKey] = photo.Thumb
		data = photo.Full
	}
	a.SizeBytes = int64(len(data))

	var stored []string
	cleanup := func() {
		for _, key := range stored {
			if err := s.storage.Delete(ctx, key); err != nil {
				logging.Warn(ctx, "[JOB-ATTACHMENT] Failed to remove orphaned object", map[string]any{"error": err})
			}
		}
	}
	for key, bytes := range objects {
		ct := contentTypeJPEG
		if key == a.ObjectKey {
			ct = a.ContentType
		}
		in := storage.NewUploadInput(key).WithContentType(ct).WithCacheControl("private, max-age=3600")
		if _, err := s.storage.UploadBytes(ctx, bytes, in); err != nil {
			cleanup()
			return nil, fmt.Errorf("store attachment: %w", err)
		}
		stored = append(stored, key)
	}
	if err := s.repo.Create(ctx, tenantID, a); err != nil {
		cleanup()
		return nil, fmt.Errorf("save attachment: %w", err)
	}
	logging.Info(ctx, "[JOB-ATTACHMENT] Attachment added", map[string]any{
		"job_id": jobID.String(), "attachment_id": id.String(), "kind": string(a.Kind), "size_bytes": a.SizeBytes,
	})
	created, err := s.repo.Get(ctx, tenantID, jobID, id)
	if err != nil {
		return nil, err
	}
	resp, err := s.respond(ctx, created)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

// process decodes a photo with at most cap(s.decode) running at once.
func (s *JobAttachmentService) process(ctx context.Context, data []byte) (*processedPhoto, error) {
	select {
	case s.decode <- struct{}{}:
	case <-ctx.Done():
		return nil, ctx.Err()
	}
	defer func() { <-s.decode }()
	return processPhoto(data)
}

// respond adds presigned links to the attachment.
func (s *JobAttachmentService) respond(ctx context.Context, a *models.JobAttachment) (models.JobAttachmentResponse, error) {
	r := a.ToResponse()
	url, err := s.storage.GetPresignedURL(ctx, a.ObjectKey, AttachmentLinkTTL)
	if err != nil {
		return r, fmt.Errorf("presign attachment: %w", err)
	}
	r.URL = url
	if a.ThumbKey != nil {
		if r.ThumbURL, err = s.storage.GetPresignedURL(ctx, *a.ThumbKey, AttachmentLinkTTL); err != nil {
			return r, fmt.Errorf("presign thumbnail: %w", err)
		}
	}
	return r, nil
}

// List returns the job's attachments, oldest first, with fresh links.
func (s *JobAttachmentService) List(ctx context.Context, jobID uuid.UUID) ([]models.JobAttachmentResponse, error) {
	tenantID, err := s.visibleJob(ctx, jobID)
	if err != nil {
		return nil, err
	}
	rows, err := s.repo.ListByJob(ctx, tenantID, jobID)
	if err != nil {
		return nil, err
	}
	out := make([]models.JobAttachmentResponse, len(rows))
	for i := range rows {
		if out[i], err = s.respond(ctx, &rows[i]); err != nil {
			return nil, err
		}
	}
	return out, nil
}

// Link returns a presigned link to the file or its thumbnail.
func (s *JobAttachmentService) Link(ctx context.Context, jobID, id uuid.UUID, thumb bool) (string, error) {
	_, a, err := s.get(ctx, jobID, id)
	if err != nil {
		return "", err
	}
	key := a.ObjectKey
	if thumb {
		if a.ThumbKey == nil {
			return "", ErrAttachmentNoThumbnail
		}
		key = *a.ThumbKey
	}
	url, err := s.storage.GetPresignedURL(ctx, key, AttachmentLinkTTL)
	if err != nil {
		return "", fmt.Errorf("presign attachment: %w", err)
	}
	return url, nil
}

// UpdateCaption sets or clears the caption.
func (s *JobAttachmentService) UpdateCaption(ctx context.Context, jobID, id, userID uuid.UUID, req *models.UpdateJobAttachmentRequest) (*models.JobAttachmentResponse, error) {
	caption, err := cleanCaption(req.Caption)
	if err != nil {
		return nil, err
	}
	tenantID, a, err := s.get(ctx, jobID, id)
	if err != nil {
		return nil, err
	}
	if !mayChange(ctx, a, userID) {
		return nil, ErrAttachmentNotYours
	}
	if err := s.repo.UpdateCaption(ctx, tenantID, jobID, id, caption); err != nil {
		if errors.Is(err, repository.ErrJobAttachmentNotFound) {
			return nil, ErrJobAttachmentNotFound
		}
		return nil, err
	}
	a.Caption = caption
	resp, err := s.respond(ctx, a)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

// Delete removes the attachment and its stored objects.
func (s *JobAttachmentService) Delete(ctx context.Context, jobID, id, userID uuid.UUID) error {
	tenantID, a, err := s.get(ctx, jobID, id)
	if err != nil {
		return err
	}
	if !mayChange(ctx, a, userID) {
		return ErrAttachmentNotYours
	}
	if err := s.repo.Delete(ctx, tenantID, jobID, id); err != nil {
		if errors.Is(err, repository.ErrJobAttachmentNotFound) {
			return ErrJobAttachmentNotFound
		}
		return err
	}
	keys := []string{a.ObjectKey}
	if a.ThumbKey != nil {
		keys = append(keys, *a.ThumbKey)
	}
	for _, key := range keys {
		if err := s.storage.Delete(ctx, key); err != nil {
			// The row is gone, so nothing links to it; log for a sweep.
			logging.Warn(ctx, "[JOB-ATTACHMENT] Failed to delete stored object", map[string]any{"error": err, "attachment_id": id.String()})
		}
	}
	return nil
}
