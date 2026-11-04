package repository

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/javaknight1/servicepro/backend/internal/models"
	"github.com/javaknight1/servicepro/backend/internal/tenancy"
)

var (
	// ErrJobSignoffNotFound: the job has no sign-off in the caller's tenant.
	ErrJobSignoffNotFound = errors.New("job sign-off not found")
	// ErrJobSignoffLocked: the report was emailed, so the sign-off can't be
	// replaced.
	ErrJobSignoffLocked = errors.New("job sign-off locked: report already emailed")
)

// JobReportSource is everything the service report is built from (T176),
// loaded in the caller's tenant.
type JobReportSource struct {
	Job    models.Job
	Tenant models.Tenant
	Site   *models.Site
	// Transitions are the job's status changes, oldest first.
	Transitions []models.JobStatusTransition
	// Photos are the job's photo attachments, oldest first.
	Photos []models.JobAttachment
}

// JobContacts are who a job's service report goes to by default.
type JobContacts struct {
	CustomerEmail string
	DoNotEmail    bool
	SiteEmail     string
}

// JobSignoffRepositoryInterface is data access for job sign-offs (T176).
// Every method takes ctx (the tenant for the tenancy plugin and RLS, and the
// technician row filter) and the tenant ID, and filters tenant_id too.
type JobSignoffRepositoryInterface interface {
	// JobVisible: the job is in the tenant, not deleted and, for callers
	// limited to assigned jobs, actively assigned to them.
	JobVisible(ctx context.Context, tenantID, jobID uuid.UUID) (bool, error)
	// JobStatus returns the job's status (ErrJobSignoffNotFound if missing).
	JobStatus(ctx context.Context, tenantID, jobID uuid.UUID) (models.JobStatus, error)
	Get(ctx context.Context, tenantID, jobID uuid.UUID) (*models.JobSignoff, error)
	// Save inserts s, or replaces the job's sign-off and clears its report;
	// it returns the replaced row (nil on insert). A sign-off whose report
	// was emailed is ErrJobSignoffLocked.
	Save(ctx context.Context, tenantID uuid.UUID, s *models.JobSignoff) (*models.JobSignoff, error)
	SetReport(ctx context.Context, tenantID, jobID uuid.UUID, key string, at time.Time) error
	// ClaimEmail marks the report as being emailed. It succeeds only when no
	// other send holds a claim newer than staleBefore and, unless resend,
	// the report hasn't been emailed yet.
	ClaimEmail(ctx context.Context, tenantID, jobID uuid.UUID, resend bool, now, staleBefore time.Time) (bool, error)
	// FinishEmail records a sent report and releases the claim.
	FinishEmail(ctx context.Context, tenantID, jobID uuid.UUID, recipients []string, at time.Time) error
	// ReleaseEmail drops the claim after a failed send.
	ReleaseEmail(ctx context.Context, tenantID, jobID uuid.UUID) error
	Contacts(ctx context.Context, tenantID, jobID uuid.UUID) (*JobContacts, error)
	ReportSource(ctx context.Context, tenantID, jobID uuid.UUID) (*JobReportSource, error)
}

// JobSignoffRepository implements JobSignoffRepositoryInterface with GORM.
type JobSignoffRepository struct {
	db *gorm.DB
}

// NewJobSignoffRepository creates a job sign-off repository.
func NewJobSignoffRepository(db *gorm.DB) *JobSignoffRepository {
	return &JobSignoffRepository{db: db}
}

// jobs is the job handle with the same row filter as JobRepository.jobs().
func (r *JobSignoffRepository) jobs(ctx context.Context, tenantID, jobID uuid.UUID) *gorm.DB {
	q := r.db.WithContext(ctx).Model(&models.Job{}).Where("jobs.tenant_id = ? AND jobs.id = ?", tenantID, jobID)
	if userID, ok := tenancy.AssignedJobsUser(ctx); ok {
		q = q.Where(assignedJobsOnly, userID)
	}
	return q
}

// JobVisible reports whether the caller can see the job.
func (r *JobSignoffRepository) JobVisible(ctx context.Context, tenantID, jobID uuid.UUID) (bool, error) {
	var n int64
	if err := r.jobs(ctx, tenantID, jobID).Count(&n).Error; err != nil {
		return false, err
	}
	return n > 0, nil
}

// JobStatus returns the job's current status.
func (r *JobSignoffRepository) JobStatus(ctx context.Context, tenantID, jobID uuid.UUID) (models.JobStatus, error) {
	var statuses []models.JobStatus
	if err := r.jobs(ctx, tenantID, jobID).Pluck("status", &statuses).Error; err != nil {
		return "", err
	}
	if len(statuses) == 0 {
		return "", ErrJobSignoffNotFound
	}
	return statuses[0], nil
}

func (r *JobSignoffRepository) scoped(ctx context.Context, tenantID, jobID uuid.UUID) *gorm.DB {
	return r.db.WithContext(ctx).Model(&models.JobSignoff{}).Where("tenant_id = ? AND job_id = ?", tenantID, jobID)
}

// Get returns the job's sign-off with the user who captured it.
func (r *JobSignoffRepository) Get(ctx context.Context, tenantID, jobID uuid.UUID) (*models.JobSignoff, error) {
	var s models.JobSignoff
	err := r.db.WithContext(ctx).Preload("Capturer").
		Where("tenant_id = ? AND job_id = ?", tenantID, jobID).First(&s).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrJobSignoffNotFound
	}
	if err != nil {
		return nil, err
	}
	return &s, nil
}

// Save inserts or replaces the job's sign-off in one transaction, the
// existing row locked so two captures can't interleave.
func (r *JobSignoffRepository) Save(ctx context.Context, tenantID uuid.UUID, s *models.JobSignoff) (*models.JobSignoff, error) {
	s.TenantID = tenantID
	var replaced *models.JobSignoff
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// The job must be the tenant's (job_id's foreign key alone would
		// accept another tenant's job).
		var n int64
		if err := tx.Model(&models.Job{}).Where("tenant_id = ? AND id = ?", tenantID, s.JobID).Count(&n).Error; err != nil {
			return err
		}
		if n == 0 {
			return ErrJobSignoffNotFound
		}
		var existing models.JobSignoff
		err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("tenant_id = ? AND job_id = ?", tenantID, s.JobID).First(&existing).Error
		switch {
		case errors.Is(err, gorm.ErrRecordNotFound):
			if s.ID == uuid.Nil {
				s.ID = uuid.New()
			}
			return tx.Omit("Capturer").Create(s).Error
		case err != nil:
			return err
		}
		if existing.ReportEmailedAt != nil {
			return ErrJobSignoffLocked
		}
		replaced = &existing
		s.ID = existing.ID
		s.CreatedAt = existing.CreatedAt
		return tx.Model(&models.JobSignoff{}).Where("id = ? AND tenant_id = ?", existing.ID, tenantID).
			Updates(map[string]any{
				"signer_name": s.SignerName, "signer_title": s.SignerTitle, "signer_email": s.SignerEmail,
				"signature_key": s.SignatureKey, "signature_width": s.SignatureWidth, "signature_height": s.SignatureHeight,
				"signed_at": s.SignedAt, "ip_address": s.IPAddress, "user_agent": s.UserAgent, "captured_by": s.CapturedBy,
				"report_key": nil, "report_generated_at": nil, "updated_at": time.Now(),
			}).Error
	})
	if err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) || isUniqueViolation(err) {
			// A concurrent first capture won; the caller may sign again.
			return nil, ErrJobSignoffLocked
		}
		return nil, err
	}
	return replaced, nil
}

// SetReport records where the generated report is stored.
func (r *JobSignoffRepository) SetReport(ctx context.Context, tenantID, jobID uuid.UUID, key string, at time.Time) error {
	res := r.scoped(ctx, tenantID, jobID).Updates(map[string]any{"report_key": key, "report_generated_at": at})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return ErrJobSignoffNotFound
	}
	return nil
}

// ClaimEmail claims the send with a conditional UPDATE: of two concurrent
// requests only one matches.
func (r *JobSignoffRepository) ClaimEmail(ctx context.Context, tenantID, jobID uuid.UUID, resend bool, now, staleBefore time.Time) (bool, error) {
	q := r.scoped(ctx, tenantID, jobID).
		Where("(report_email_claimed_at IS NULL OR report_email_claimed_at < ?)", staleBefore)
	if !resend {
		q = q.Where("report_emailed_at IS NULL")
	}
	res := q.Update("report_email_claimed_at", now)
	if res.Error != nil {
		return false, res.Error
	}
	return res.RowsAffected == 1, nil
}

// FinishEmail records the send.
func (r *JobSignoffRepository) FinishEmail(ctx context.Context, tenantID, jobID uuid.UUID, recipients []string, at time.Time) error {
	return r.scoped(ctx, tenantID, jobID).Updates(map[string]any{
		"report_emailed_at":       at,
		"report_recipients":       strings.Join(recipients, ","),
		"report_email_count":      gorm.Expr("report_email_count + 1"),
		"report_email_claimed_at": nil,
	}).Error
}

// ReleaseEmail drops the claim.
func (r *JobSignoffRepository) ReleaseEmail(ctx context.Context, tenantID, jobID uuid.UUID) error {
	return r.scoped(ctx, tenantID, jobID).Update("report_email_claimed_at", nil).Error
}

// Contacts returns the customer's email and opt-out and the site contact.
func (r *JobSignoffRepository) Contacts(ctx context.Context, tenantID, jobID uuid.UUID) (*JobContacts, error) {
	var row struct {
		CustomerEmail string
		DoNotEmail    bool
		SiteEmail     *string
	}
	q := r.jobs(ctx, tenantID, jobID).
		Select("customers.email AS customer_email, customers.do_not_email AS do_not_email, service_locations.contact_email AS site_email").
		Joins("JOIN customers ON customers.id = jobs.customer_id AND customers.tenant_id = jobs.tenant_id").
		Joins("LEFT JOIN service_locations ON service_locations.id = jobs.site_id AND service_locations.tenant_id = jobs.tenant_id AND service_locations.deleted_at IS NULL")
	res := q.Scan(&row)
	if res.Error != nil {
		return nil, res.Error
	}
	if res.RowsAffected == 0 {
		return nil, ErrJobSignoffNotFound
	}
	c := &JobContacts{CustomerEmail: row.CustomerEmail, DoNotEmail: row.DoNotEmail}
	if row.SiteEmail != nil {
		c.SiteEmail = *row.SiteEmail
	}
	return c, nil
}

// ReportSource loads the job with its customer, crew, notes (customer
// visible only) and materials, the site, the tenant, the status history and
// the photos.
func (r *JobSignoffRepository) ReportSource(ctx context.Context, tenantID, jobID uuid.UUID) (*JobReportSource, error) {
	src := &JobReportSource{}
	err := r.jobs(ctx, tenantID, jobID).
		Preload("Customer").
		Preload("Assignments", "unassigned_at IS NULL").
		Preload("Assignments.User").
		Preload("Materials", func(db *gorm.DB) *gorm.DB { return db.Order("created_at ASC") }).
		Preload("Notes", func(db *gorm.DB) *gorm.DB { return db.Where("is_internal = ?", false).Order("created_at ASC") }).
		Preload("Notes.User").
		First(&src.Job).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrJobSignoffNotFound
	}
	if err != nil {
		return nil, err
	}
	db := r.db.WithContext(ctx)
	if err := db.Select("id", "name", "email", "phone", "settings").
		Where("id = ?", tenantID).First(&src.Tenant).Error; err != nil {
		return nil, err
	}
	if src.Job.SiteID != nil {
		var site models.Site
		err := db.Where("tenant_id = ? AND id = ?", tenantID, *src.Job.SiteID).First(&site).Error
		switch {
		case err == nil:
			src.Site = &site
		case !errors.Is(err, gorm.ErrRecordNotFound):
			return nil, err
		}
	}
	// job_status_transitions has no tenant_id: reached through the job
	// loaded above in the caller's tenant.
	if err := db.Where("job_id = ?", jobID).Order("transitioned_at ASC").
		Find(&src.Transitions).Error; err != nil {
		return nil, err
	}
	if err := db.Where("tenant_id = ? AND job_id = ? AND kind = ?", tenantID, jobID, models.JobAttachmentPhoto).
		Order("created_at ASC, id ASC").Find(&src.Photos).Error; err != nil {
		return nil, err
	}
	return src, nil
}
