package services

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"

	apperrors "github.com/javaknight1/servicepro/backend/internal/errors"
	"github.com/javaknight1/servicepro/backend/internal/models"
)

// Tax rate domain errors (T237).
var (
	// ErrTaxRateMissing: no such tax rate in this tenant (or the global defaults)
	ErrTaxRateMissing = apperrors.NotFound("tax_rate_not_found", "Tax rate not found")
	// ErrTaxRateReadOnly: global default rates can be used but not changed by a tenant
	ErrTaxRateReadOnly = apperrors.Forbidden("tax_rate_read_only", "Default tax rates can't be changed")
	// ErrTaxRateInUse: invoices reference the rate (WithMessage gives the count)
	ErrTaxRateInUse = apperrors.Conflict("tax_rate_in_use", "This tax rate is used by invoices and can't be deleted")
	// ErrInvalidTaxRateInput: validation failed (WithMessage gives the reason)
	ErrInvalidTaxRateInput = apperrors.BadRequest("invalid_tax_rate", "Invalid tax rate")
)

// validTaxTypes are the values of the tax_type enum.
var validTaxTypes = map[models.TaxType]bool{
	models.TaxTypeSalesTax: true, models.TaxTypeVAT: true, models.TaxTypeGST: true,
	models.TaxTypeHST: true, models.TaxTypeExempt: true,
}

// TaxRateService manages a tenant's tax rates. Tenants see their own rates
// plus the global defaults (tenant_id IS NULL), and can only create, change
// or delete their own. Every query filters tenant_id explicitly; the tenancy
// plugin (tax_rates is SharedRead) is a second layer, not the only one.
type TaxRateService struct {
	db *gorm.DB
}

// NewTaxRateService creates a tax rate service.
func NewTaxRateService(db *gorm.DB) *TaxRateService {
	return &TaxRateService{db: db}
}

// TaxRateInput is a create (all fields) or a partial update (nil = unchanged).
type TaxRateInput struct {
	Name        *string
	Description *string
	Rate        *decimal.Decimal // fraction: 0.0825 = 8.25%
	TaxType     *models.TaxType
	Region      *string
	IsCompound  *bool
	IsActive    *bool
}

// visible scopes a query to the tenant's rows plus the global defaults.
func (s *TaxRateService) visible(ctx context.Context, tenantID uuid.UUID) *gorm.DB {
	return s.db.WithContext(ctx).Where("(tax_rates.tenant_id = ? OR tax_rates.tenant_id IS NULL)", tenantID)
}

// List returns the tenant's tax rates and the global defaults, by name.
func (s *TaxRateService) List(ctx context.Context, tenantID uuid.UUID, activeOnly bool) ([]models.TaxRate, error) {
	rates := []models.TaxRate{}
	q := s.visible(ctx, tenantID)
	if activeOnly {
		q = q.Where("is_active = ?", true)
	}
	if err := q.Order("name ASC").Find(&rates).Error; err != nil {
		return nil, fmt.Errorf("failed to list tax rates: %w", err)
	}
	return rates, nil
}

// Get returns one tax rate the tenant can see (its own or a global default).
func (s *TaxRateService) Get(ctx context.Context, tenantID, id uuid.UUID) (*models.TaxRate, error) {
	var rate models.TaxRate
	if err := s.visible(ctx, tenantID).Where("id = ?", id).First(&rate).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrTaxRateMissing
		}
		return nil, fmt.Errorf("failed to get tax rate: %w", err)
	}
	return &rate, nil
}

// owned returns a tax rate the tenant may change: another tenant's rate is
// not found, a global default is read-only.
func (s *TaxRateService) owned(ctx context.Context, tenantID, id uuid.UUID) (*models.TaxRate, error) {
	rate, err := s.Get(ctx, tenantID, id)
	if err != nil {
		return nil, err
	}
	if rate.TenantID == nil {
		return nil, ErrTaxRateReadOnly
	}
	return rate, nil
}

// Create adds a tax rate owned by the tenant. Name, Rate and the tax type
// default when omitted: the type to sales_tax and IsActive to true.
func (s *TaxRateService) Create(ctx context.Context, tenantID uuid.UUID, in TaxRateInput) (*models.TaxRate, error) {
	rate := &models.TaxRate{ID: uuid.New(), TenantID: &tenantID, TaxType: models.TaxTypeSalesTax, IsActive: true}
	if in.Name == nil || in.Rate == nil {
		return nil, invalidTaxRate("name and rate are required")
	}
	applyTaxRateInput(rate, in)
	if err := validateTaxRate(rate); err != nil {
		return nil, err
	}
	if err := s.db.WithContext(ctx).Create(rate).Error; err != nil {
		return nil, fmt.Errorf("failed to create tax rate: %w", err)
	}
	return rate, nil
}

// Update changes the given fields of one of the tenant's own tax rates.
func (s *TaxRateService) Update(ctx context.Context, tenantID, id uuid.UUID, in TaxRateInput) (*models.TaxRate, error) {
	rate, err := s.owned(ctx, tenantID, id)
	if err != nil {
		return nil, err
	}
	applyTaxRateInput(rate, in)
	if err := validateTaxRate(rate); err != nil {
		return nil, err
	}
	fields := map[string]interface{}{
		"name":        rate.Name,
		"description": rate.Description,
		"rate":        rate.Rate,
		"tax_type":    rate.TaxType,
		"region":      rate.Region,
		"is_compound": rate.IsCompound,
		"is_active":   rate.IsActive,
	}
	res := s.db.WithContext(ctx).Model(&models.TaxRate{}).
		Where("id = ? AND tenant_id = ?", id, tenantID).Updates(fields)
	if res.Error != nil {
		return nil, fmt.Errorf("failed to update tax rate: %w", res.Error)
	}
	if res.RowsAffected == 0 {
		return nil, ErrTaxRateMissing
	}
	return s.Get(ctx, tenantID, id)
}

// Delete removes one of the tenant's own tax rates. A rate invoices still
// reference can't be deleted (deactivate it instead).
func (s *TaxRateService) Delete(ctx context.Context, tenantID, id uuid.UUID) error {
	if _, err := s.owned(ctx, tenantID, id); err != nil {
		return err
	}
	var count int64
	if err := s.db.WithContext(ctx).Model(&models.Invoice{}).
		Where("tenant_id = ? AND tax_rate_id = ?", tenantID, id).Count(&count).Error; err != nil {
		return fmt.Errorf("failed to check tax rate usage: %w", err)
	}
	if count > 0 {
		return ErrTaxRateInUse.WithMessage(fmt.Sprintf(
			"This tax rate is used by %d invoice(s) and can't be deleted; deactivate it instead", count))
	}
	res := s.db.WithContext(ctx).Where("id = ? AND tenant_id = ?", id, tenantID).Delete(&models.TaxRate{})
	if res.Error != nil {
		return fmt.Errorf("failed to delete tax rate: %w", res.Error)
	}
	if res.RowsAffected == 0 {
		return ErrTaxRateMissing
	}
	return nil
}

func applyTaxRateInput(rate *models.TaxRate, in TaxRateInput) {
	if in.Name != nil {
		rate.Name = strings.TrimSpace(*in.Name)
	}
	if in.Description != nil {
		rate.Description = *in.Description
	}
	if in.Rate != nil {
		rate.Rate = *in.Rate
	}
	if in.TaxType != nil {
		rate.TaxType = *in.TaxType
	}
	if in.Region != nil {
		rate.Region = *in.Region
	}
	if in.IsCompound != nil {
		rate.IsCompound = *in.IsCompound
	}
	if in.IsActive != nil {
		rate.IsActive = *in.IsActive
	}
}

func validateTaxRate(rate *models.TaxRate) error {
	switch {
	case rate.Name == "":
		return invalidTaxRate("name is required")
	case len(rate.Name) > 100:
		return invalidTaxRate("name must be at most 100 characters")
	case len(rate.Region) > 100:
		return invalidTaxRate("region must be at most 100 characters")
	case rate.Rate.IsNegative() || rate.Rate.GreaterThan(decimal.NewFromInt(1)):
		return invalidTaxRate("rate must be a fraction between 0 and 1 (0.0825 = 8.25%)")
	case rate.Rate.Exponent() < -4 && !rate.Rate.Equal(rate.Rate.Round(4)):
		return invalidTaxRate("rate can have at most 4 decimal places")
	case !validTaxTypes[rate.TaxType]:
		return invalidTaxRate("tax_type must be one of sales_tax, vat, gst, hst, exempt")
	}
	return nil
}

// invalidTaxRate is ErrInvalidTaxRateInput with the validation reason.
func invalidTaxRate(reason string) error {
	return ErrInvalidTaxRateInput.WithMessage(strings.ToUpper(reason[:1]) + reason[1:])
}
