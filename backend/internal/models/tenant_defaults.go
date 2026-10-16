package models

import (
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Tenant document defaults (T127): TenantSettings.DefaultPaymentTermID and
// DefaultTaxRateID are applied when an invoice/quote is created without a
// payment term / tax rate. The settings endpoint stores whatever ID it is
// given, so each lookup is restricted to the tenant's own rows and the
// shared global rows (tenant_id IS NULL); another tenant's ID, a deleted
// row or an inactive one resolves to nothing and no default is applied.

// DefaultPaymentTerm returns the tenant's default payment term, or nil.
func DefaultPaymentTerm(db *gorm.DB, tenantID uuid.UUID, settings TenantSettings) (*PaymentTerm, error) {
	if settings.DefaultPaymentTermID == nil || tenantID == uuid.Nil {
		return nil, nil
	}
	var rows []PaymentTerm
	if err := db.Where("id = ? AND (tenant_id = ? OR tenant_id IS NULL) AND is_active = ?",
		*settings.DefaultPaymentTermID, tenantID, true).Limit(1).Find(&rows).Error; err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, nil
	}
	return &rows[0], nil
}

// DefaultTaxRate returns the tenant's default tax rate, or nil.
func DefaultTaxRate(db *gorm.DB, tenantID uuid.UUID, settings TenantSettings) (*TaxRate, error) {
	if settings.DefaultTaxRateID == nil || tenantID == uuid.Nil {
		return nil, nil
	}
	var rows []TaxRate
	if err := db.Where("id = ? AND (tenant_id = ? OR tenant_id IS NULL) AND is_active = ?",
		*settings.DefaultTaxRateID, tenantID, true).Limit(1).Find(&rows).Error; err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, nil
	}
	return &rows[0], nil
}
