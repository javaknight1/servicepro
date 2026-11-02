package services

import (
	"context"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/javaknight1/servicepro/backend/internal/models"
	"github.com/javaknight1/servicepro/backend/internal/tenancy"
	"github.com/javaknight1/servicepro/backend/pkg/clients/email"
	"github.com/javaknight1/servicepro/backend/pkg/clients/logging"
)

// withTenantSender attaches the tenant as the sending business, so emails to
// its customers go out as "<Business> via ServicePro" with replies to the
// business (T265). On any lookup problem the platform sender is used.
func withTenantSender(ctx context.Context, db *gorm.DB, tenantID uuid.UUID) context.Context {
	if db == nil || tenantID == uuid.Nil {
		return ctx
	}
	var t models.Tenant
	if err := db.WithContext(tenancy.WithSystem(ctx)).Select("id", "name", "email").
		First(&t, "id = ?", tenantID).Error; err != nil {
		logging.Warn(ctx, "[EMAIL] Tenant sender lookup failed; using platform sender", map[string]any{"tenant_id": tenantID, "error": err})
		return ctx
	}
	return tenantSenderFor(ctx, &t)
}

// tenantSenderFor is withTenantSender for an already-loaded tenant.
func tenantSenderFor(ctx context.Context, t *models.Tenant) context.Context {
	if t == nil || t.Name == "" {
		return ctx
	}
	return email.WithSender(ctx, email.Sender{BusinessName: t.Name, ReplyTo: t.Email})
}
