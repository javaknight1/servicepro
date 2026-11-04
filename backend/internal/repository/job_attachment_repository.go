package repository

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/javaknight1/servicepro/backend/internal/models"
	"github.com/javaknight1/servicepro/backend/internal/tenancy"
)

// ErrJobAttachmentNotFound: no such attachment on that job in the caller's
// tenant. Other tenants' attachments look exactly like missing ones.
var ErrJobAttachmentNotFound = errors.New("job attachment not found")

// JobAttachmentRepositoryInterface is data access for job photos and files
// (T174). Every method takes ctx (which carries the tenant for the tenancy
// plugin and the RLS backstop, and the technician row filter) and the tenant
// ID, and filters tenant_id explicitly too.
type JobAttachmentRepositoryInterface interface {
	// JobVisible reports whether the caller can see the job: it is in the
	// tenant, not deleted, and — for callers limited to their assigned jobs
	// (tenancy.WithAssignedJobsOnly) — actively assigned to them.
	JobVisible(ctx context.Context, tenantID, jobID uuid.UUID) (bool, error)
	Create(ctx context.Context, tenantID uuid.UUID, a *models.JobAttachment) error
	Get(ctx context.Context, tenantID, jobID, id uuid.UUID) (*models.JobAttachment, error)
	// ListByJob returns the job's attachments, oldest first, with uploaders.
	ListByJob(ctx context.Context, tenantID, jobID uuid.UUID) ([]models.JobAttachment, error)
	CountByJob(ctx context.Context, tenantID, jobID uuid.UUID) (int64, error)
	UpdateCaption(ctx context.Context, tenantID, jobID, id uuid.UUID, caption *string) error
	Delete(ctx context.Context, tenantID, jobID, id uuid.UUID) error
}

// JobAttachmentRepository implements JobAttachmentRepositoryInterface with GORM.
type JobAttachmentRepository struct {
	db *gorm.DB
}

// NewJobAttachmentRepository creates a job-attachment repository.
func NewJobAttachmentRepository(db *gorm.DB) *JobAttachmentRepository {
	return &JobAttachmentRepository{db: db}
}

// JobVisible applies the same row filter as JobRepository.jobs(), so a job a
// technician can't open is a job whose photos they can't touch either.
func (r *JobAttachmentRepository) JobVisible(ctx context.Context, tenantID, jobID uuid.UUID) (bool, error) {
	q := r.db.WithContext(ctx).Model(&models.Job{}).Where("jobs.tenant_id = ? AND jobs.id = ?", tenantID, jobID)
	if userID, ok := tenancy.AssignedJobsUser(ctx); ok {
		q = q.Where(assignedJobsOnly, userID)
	}
	var n int64
	if err := q.Count(&n).Error; err != nil {
		return false, err
	}
	return n > 0, nil
}

// Create inserts a into tenantID (the tenant always comes from the caller,
// never from the value).
func (r *JobAttachmentRepository) Create(ctx context.Context, tenantID uuid.UUID, a *models.JobAttachment) error {
	a.TenantID = tenantID
	if a.ID == uuid.Nil {
		a.ID = uuid.New()
	}
	return r.db.WithContext(ctx).Omit("Uploader").Create(a).Error
}

// Get returns one of the job's attachments.
func (r *JobAttachmentRepository) Get(ctx context.Context, tenantID, jobID, id uuid.UUID) (*models.JobAttachment, error) {
	var a models.JobAttachment
	err := r.db.WithContext(ctx).Preload("Uploader").
		Where("tenant_id = ? AND job_id = ? AND id = ?", tenantID, jobID, id).
		First(&a).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrJobAttachmentNotFound
	}
	if err != nil {
		return nil, err
	}
	return &a, nil
}

// ListByJob returns the job's attachments, oldest first.
func (r *JobAttachmentRepository) ListByJob(ctx context.Context, tenantID, jobID uuid.UUID) ([]models.JobAttachment, error) {
	var out []models.JobAttachment
	err := r.db.WithContext(ctx).Preload("Uploader").
		Where("tenant_id = ? AND job_id = ?", tenantID, jobID).
		Order("created_at ASC, id ASC").
		Find(&out).Error
	return out, err
}

// CountByJob counts the job's attachments.
func (r *JobAttachmentRepository) CountByJob(ctx context.Context, tenantID, jobID uuid.UUID) (int64, error) {
	var n int64
	err := r.db.WithContext(ctx).Model(&models.JobAttachment{}).
		Where("tenant_id = ? AND job_id = ?", tenantID, jobID).
		Count(&n).Error
	return n, err
}

// UpdateCaption sets (or, with nil, clears) the caption.
func (r *JobAttachmentRepository) UpdateCaption(ctx context.Context, tenantID, jobID, id uuid.UUID, caption *string) error {
	res := r.db.WithContext(ctx).Model(&models.JobAttachment{}).
		Where("tenant_id = ? AND job_id = ? AND id = ?", tenantID, jobID, id).
		Update("caption", caption)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return ErrJobAttachmentNotFound
	}
	return nil
}

// Delete removes the row (the caller removes the stored objects).
func (r *JobAttachmentRepository) Delete(ctx context.Context, tenantID, jobID, id uuid.UUID) error {
	res := r.db.WithContext(ctx).
		Where("tenant_id = ? AND job_id = ? AND id = ?", tenantID, jobID, id).
		Delete(&models.JobAttachment{})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return ErrJobAttachmentNotFound
	}
	return nil
}
