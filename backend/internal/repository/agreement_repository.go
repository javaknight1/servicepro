package repository

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/javaknight1/servicepro/backend/internal/models"
)

// ErrAgreementNotFound: no such agreement in the caller's tenant.
var ErrAgreementNotFound = errors.New("agreement not found")

// AgreementRepositoryInterface is data access for service agreements (T167).
// Every method takes ctx and the tenant and filters tenant_id explicitly.
type AgreementRepositoryInterface interface {
	Create(ctx context.Context, tenantID uuid.UUID, a *models.ServiceAgreement) error
	Get(ctx context.Context, tenantID, id uuid.UUID) (*models.ServiceAgreement, error)
	List(ctx context.Context, tenantID uuid.UUID, filter *models.AgreementListFilter) ([]models.ServiceAgreement, error)
	// Update saves every editable column of a.
	Update(ctx context.Context, tenantID uuid.UUID, a *models.ServiceAgreement) error
	SetStatus(ctx context.Context, tenantID, id uuid.UUID, status models.AgreementStatus) error
	Delete(ctx context.Context, tenantID, id uuid.UUID) error
}

// AgreementRepository implements AgreementRepositoryInterface with GORM.
type AgreementRepository struct {
	db *gorm.DB
}

// NewAgreementRepository creates an agreement repository.
func NewAgreementRepository(db *gorm.DB) *AgreementRepository {
	return &AgreementRepository{db: db}
}

// Create inserts a into tenantID (never the value's own tenant).
func (r *AgreementRepository) Create(ctx context.Context, tenantID uuid.UUID, a *models.ServiceAgreement) error {
	a.TenantID = tenantID
	if a.ID == uuid.Nil {
		a.ID = uuid.New()
	}
	return r.db.WithContext(ctx).Create(a).Error
}

// Get returns one of the tenant's agreements.
func (r *AgreementRepository) Get(ctx context.Context, tenantID, id uuid.UUID) (*models.ServiceAgreement, error) {
	var a models.ServiceAgreement
	err := r.db.WithContext(ctx).Where("tenant_id = ? AND id = ?", tenantID, id).First(&a).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrAgreementNotFound
	}
	if err != nil {
		return nil, err
	}
	return &a, nil
}

// List returns the tenant's agreements, newest first.
func (r *AgreementRepository) List(ctx context.Context, tenantID uuid.UUID, filter *models.AgreementListFilter) ([]models.ServiceAgreement, error) {
	q := r.db.WithContext(ctx).Where("tenant_id = ?", tenantID)
	if filter != nil {
		if filter.CustomerID != nil {
			q = q.Where("customer_id = ?", *filter.CustomerID)
		}
		if filter.SiteID != nil {
			q = q.Where("site_id = ?", *filter.SiteID)
		}
		if filter.Status != nil {
			q = q.Where("status = ?", *filter.Status)
		}
	}
	out := []models.ServiceAgreement{}
	if err := q.Order("created_at DESC").Find(&out).Error; err != nil {
		return nil, err
	}
	return out, nil
}

var agreementColumns = []string{
	"title", "frequency", "weekday_mask", "times_per_week", "day_of_month", "start_date", "end_date",
	"visit_start_minutes", "visit_duration_minutes", "price_per_period", "billing_period",
	"assigned_technician_id", "job_type", "notes", "updated_at",
}

// Update saves a's editable columns (Select writes zero and nil values too).
func (r *AgreementRepository) Update(ctx context.Context, tenantID uuid.UUID, a *models.ServiceAgreement) error {
	// The caller's tenant, never the value's: another tenant's row is then
	// simply not found.
	cp := *a
	cp.TenantID = tenantID
	res := r.db.WithContext(ctx).Model(&models.ServiceAgreement{}).
		Where("tenant_id = ? AND id = ?", tenantID, a.ID).Select(agreementColumns).Updates(&cp)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return ErrAgreementNotFound
	}
	return nil
}

// SetStatus pauses or resumes an agreement.
func (r *AgreementRepository) SetStatus(ctx context.Context, tenantID, id uuid.UUID, status models.AgreementStatus) error {
	res := r.db.WithContext(ctx).Model(&models.ServiceAgreement{}).
		Where("tenant_id = ? AND id = ?", tenantID, id).Update("status", status)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return ErrAgreementNotFound
	}
	return nil
}

// Delete soft-deletes an agreement. Jobs it already generated stay.
func (r *AgreementRepository) Delete(ctx context.Context, tenantID, id uuid.UUID) error {
	res := r.db.WithContext(ctx).Where("tenant_id = ? AND id = ?", tenantID, id).Delete(&models.ServiceAgreement{})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return ErrAgreementNotFound
	}
	return nil
}
