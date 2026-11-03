package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"

	"github.com/javaknight1/servicepro/backend/internal/models"
)

// JobApprovalRepositoryInterface is data access for job approvals (T187).
// Every method takes ctx and the tenant and filters tenant_id explicitly.
type JobApprovalRepositoryInterface interface {
	Create(ctx context.Context, tenantID uuid.UUID, approval *models.JobApproval) error
	ListForJob(ctx context.Context, tenantID, jobID uuid.UUID) ([]models.JobApproval, error)
	// Covers reports whether the job has an approval for at least amount.
	Covers(ctx context.Context, tenantID, jobID uuid.UUID, amount decimal.Decimal) (bool, error)
}

// JobApprovalRepository implements JobApprovalRepositoryInterface with GORM.
type JobApprovalRepository struct {
	db *gorm.DB
}

// NewJobApprovalRepository creates a job approval repository.
func NewJobApprovalRepository(db *gorm.DB) *JobApprovalRepository {
	return &JobApprovalRepository{db: db}
}

// Create inserts an approval into tenantID (never the value's own tenant).
func (r *JobApprovalRepository) Create(ctx context.Context, tenantID uuid.UUID, approval *models.JobApproval) error {
	approval.TenantID = tenantID
	if approval.ID == uuid.Nil {
		approval.ID = uuid.New()
	}
	return r.db.WithContext(ctx).Create(approval).Error
}

// ListForJob returns the job's approvals, newest first.
func (r *JobApprovalRepository) ListForJob(ctx context.Context, tenantID, jobID uuid.UUID) ([]models.JobApproval, error) {
	out := []models.JobApproval{}
	err := r.db.WithContext(ctx).Where("tenant_id = ? AND job_id = ?", tenantID, jobID).
		Order("approved_at DESC, created_at DESC").Find(&out).Error
	return out, err
}

// Covers reports whether some approval of the job is for amount or more.
// Amounts are compared as decimals in Go (a job has a handful of approvals).
func (r *JobApprovalRepository) Covers(ctx context.Context, tenantID, jobID uuid.UUID, amount decimal.Decimal) (bool, error) {
	approvals, err := r.ListForJob(ctx, tenantID, jobID)
	if err != nil {
		return false, err
	}
	for _, a := range approvals {
		if a.Amount.GreaterThanOrEqual(amount) {
			return true, nil
		}
	}
	return false, nil
}
