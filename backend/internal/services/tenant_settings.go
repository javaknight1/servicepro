package services

import (
	"context"
	"fmt"

	"gorm.io/gorm"

	"github.com/javaknight1/servicepro/backend/internal/models"
	"github.com/javaknight1/servicepro/backend/internal/tenancy"
)

// loadTenantSettings returns the settings of the tenant in ctx (T127). The
// tenant comes only from the authenticated context (internal/tenancy).
func loadTenantSettings(ctx context.Context, db *gorm.DB) (models.TenantSettings, error) {
	tenantID, err := tenancy.MustTenantID(ctx)
	if err != nil {
		return models.TenantSettings{}, err
	}
	var tenant models.Tenant
	if err := db.WithContext(ctx).Select("id", "settings").First(&tenant, "id = ?", tenantID).Error; err != nil {
		return models.TenantSettings{}, fmt.Errorf("load tenant settings: %w", err)
	}
	return tenant.Settings, nil
}

// TenantTimezone returns the caller's tenant time zone setting, or "UTC" when
// it is unset or cannot be loaded.
func (s *PaymentTermsService) TenantTimezone(ctx context.Context) string {
	settings, err := loadTenantSettings(ctx, s.db)
	if err != nil || settings.Timezone == "" {
		return "UTC"
	}
	return settings.Timezone
}
