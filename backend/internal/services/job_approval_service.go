package services

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"

	apperrors "github.com/javaknight1/servicepro/backend/internal/errors"
	"github.com/javaknight1/servicepro/backend/internal/models"
	"github.com/javaknight1/servicepro/backend/internal/repository"
	"github.com/javaknight1/servicepro/backend/internal/tenancy"
)

// Not-to-exceed (NTE) rule (T187): extra work for a customer with an NTE
// limit whose estimated cost is above the limit can't be scheduled until an
// approval for at least that cost is recorded.
var (
	// ErrApprovalRequired: scheduling extra work over the customer's limit
	// without the approver's OK.
	ErrApprovalRequired = apperrors.Conflict("approval_required",
		"This work is over the customer's not-to-exceed limit; record the approver's OK before scheduling it")
	// ErrApproverEmailRequired: no approver email given and the customer has
	// no billing approver on file.
	ErrApproverEmailRequired = apperrors.Validation("The approver's email is required",
		map[string]string{"approver_email": "is required (the customer has no billing approver on file)"})
	// ErrInvalidApprovalAmount: a negative approval amount.
	ErrInvalidApprovalAmount = apperrors.Validation("Invalid approval amount",
		map[string]string{"amount": "must be zero or more"})
)

// jobCost is a job's estimated cost as a decimal (jobs still store float64).
func jobCost(job *models.Job) decimal.Decimal {
	return decimal.NewFromFloat(job.EstimatedCost).Round(2)
}

// isExtraWork reports whether the job is extra work: anything not generated
// by a service agreement. Until agreements generate jobs (T167) every job is.
func isExtraWork(_ *models.Job) bool {
	return true
}

// loadJobCustomer returns job.Customer, loading it (tenant-scoped) if needed.
func loadJobCustomer(ctx context.Context, db *gorm.DB, job *models.Job) (*models.Customer, error) {
	if job.Customer != nil {
		return job.Customer, nil
	}
	if db == nil {
		return nil, errors.New("job approvals: no database")
	}
	var customer models.Customer
	if err := db.WithContext(ctx).Where("id = ?", job.CustomerID).First(&customer).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrCustomerNotFound
		}
		return nil, err
	}
	return &customer, nil
}

// requireNTEApproval returns ErrApprovalRequired when job is extra work over
// its customer's not-to-exceed limit and no approval covers its estimated
// cost. Called before a job moves to scheduled.
func requireNTEApproval(ctx context.Context, db *gorm.DB, job *models.Job) error {
	if !isExtraWork(job) {
		return nil
	}
	if db == nil && job.Customer == nil {
		return nil // repository-mock unit tests only: the server always has a DB
	}
	customer, err := loadJobCustomer(ctx, db, job)
	if err != nil {
		return err
	}
	cost := jobCost(job)
	if customer.NTELimit == nil || !cost.GreaterThan(*customer.NTELimit) {
		return nil
	}
	if db == nil {
		return errors.New("job approvals: no database")
	}
	ok, err := repository.NewJobApprovalRepository(db).Covers(ctx, job.TenantID, job.ID, cost)
	if err != nil {
		return err
	}
	if !ok {
		return ErrApprovalRequired
	}
	return nil
}

// JobApprovalServiceInterface records and lists approvals (handler-facing).
type JobApprovalServiceInterface interface {
	Record(ctx context.Context, jobID uuid.UUID, req *models.RecordJobApprovalRequest, recordedBy uuid.UUID) (*models.JobApproval, error)
	List(ctx context.Context, jobID uuid.UUID) ([]models.JobApproval, error)
}

// JobApprovalService records approvers' OKs for extra work.
type JobApprovalService struct {
	db        *gorm.DB
	approvals repository.JobApprovalRepositoryInterface
}

// NewJobApprovalService creates a job approval service.
func NewJobApprovalService(db *gorm.DB, approvals repository.JobApprovalRepositoryInterface) *JobApprovalService {
	return &JobApprovalService{db: db, approvals: approvals}
}

// tenantJob loads one of the caller's tenant's jobs.
func (s *JobApprovalService) tenantJob(ctx context.Context, jobID uuid.UUID) (uuid.UUID, *models.Job, error) {
	tenantID, err := tenancy.MustTenantID(ctx)
	if err != nil {
		return uuid.Nil, nil, err
	}
	var job models.Job
	err = s.db.WithContext(ctx).Preload("Customer").Where("tenant_id = ? AND id = ?", tenantID, jobID).First(&job).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return uuid.Nil, nil, ErrJobNotFound
	}
	if err != nil {
		return uuid.Nil, nil, err
	}
	return tenantID, &job, nil
}

// Record stores the approver's OK for the job.
func (s *JobApprovalService) Record(ctx context.Context, jobID uuid.UUID, req *models.RecordJobApprovalRequest, recordedBy uuid.UUID) (*models.JobApproval, error) {
	tenantID, job, err := s.tenantJob(ctx, jobID)
	if err != nil {
		return nil, err
	}
	approval := &models.JobApproval{
		JobID:        job.ID,
		ApproverName: req.ApproverName,
		Notes:        req.Notes,
		RecordedBy:   recordedBy,
		ApprovedAt:   time.Now().UTC(),
		Amount:       jobCost(job),
	}
	switch {
	case req.ApproverEmail != nil && strings.TrimSpace(*req.ApproverEmail) != "":
		approval.ApproverEmail = strings.TrimSpace(*req.ApproverEmail)
	case job.Customer != nil && job.Customer.BillingApproverEmail != nil:
		approval.ApproverEmail = *job.Customer.BillingApproverEmail
		if approval.ApproverName == nil {
			approval.ApproverName = job.Customer.BillingApproverName
		}
	default:
		return nil, ErrApproverEmailRequired
	}
	if req.Amount != nil {
		if req.Amount.IsNegative() {
			return nil, ErrInvalidApprovalAmount
		}
		approval.Amount = req.Amount.Round(2)
	}
	if req.ApprovedAt != nil {
		approval.ApprovedAt = req.ApprovedAt.UTC()
	}
	if err := s.approvals.Create(ctx, tenantID, approval); err != nil {
		return nil, err
	}
	return approval, nil
}

// List returns the job's approvals, newest first.
func (s *JobApprovalService) List(ctx context.Context, jobID uuid.UUID) ([]models.JobApproval, error) {
	tenantID, job, err := s.tenantJob(ctx, jobID)
	if err != nil {
		return nil, err
	}
	return s.approvals.ListForJob(ctx, tenantID, job.ID)
}
