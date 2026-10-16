package services

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/javaknight1/servicepro/backend/internal/models"
	"github.com/javaknight1/servicepro/backend/internal/repository"
)

func setupTenantCreateDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	sqlDB, err := db.DB()
	require.NoError(t, err)
	sqlDB.SetMaxOpenConns(1)

	createSQLiteTableForModel(t, db, &models.Tenant{}, "")
	require.NoError(t, db.Exec(`CREATE UNIQUE INDEX uq_tenants_slug_live ON tenants(slug) WHERE deleted_at IS NULL`).Error)
	createSQLiteTableForModel(t, db, &models.TenantUser{}, "UNIQUE (tenant_id, user_id)")
	createSQLiteTableForModel(t, db, &models.Role{}, "")
	createSQLiteTableForModel(t, db, &models.MembershipTier{}, "")
	createSQLiteTableForModel(t, db, &models.TenantSubscription{}, "")
	return db
}

// T226: a failure after the tenant row is written (here: the owner role is
// missing) used to soft-delete the tenant, which kept its slug taken forever.
// Creation is now one transaction: nothing is left behind and the slug can be
// used again.
func TestCreateTenant_IsAtomic(t *testing.T) {
	db := setupTenantCreateDB(t)
	svc := NewTenantService(repository.NewTenantRepository(db), nil)
	ctx := context.Background()
	owner := uuid.New()
	req := &models.CreateTenantRequest{Name: "Acme HVAC", Slug: "acme-hvac"}

	_, err := svc.CreateTenant(ctx, req, owner)
	require.Error(t, err, "no admin role yet")

	var tenants int64
	require.NoError(t, db.Unscoped().Model(&models.Tenant{}).Count(&tenants).Error)
	assert.Zero(t, tenants, "the failed signup left a tenant row behind")

	adminRole := uuid.New()
	require.NoError(t, db.Exec(`INSERT INTO roles (id, name, hierarchy_level, created_at, updated_at) VALUES (?, 'admin', 80, ?, ?)`,
		adminRole.String(), time.Now(), time.Now()).Error)
	tierID := uuid.New()
	require.NoError(t, db.Exec(`INSERT INTO membership_tiers (id, name, enabled, monthly_price_cents, created_at, updated_at) VALUES (?, ?, true, 0, ?, ?)`,
		tierID.String(), DefaultTierName, time.Now(), time.Now()).Error)

	tenant, err := svc.CreateTenant(ctx, req, owner)
	require.NoError(t, err, "the slug is free again")

	var member models.TenantUser
	require.NoError(t, db.First(&member, "tenant_id = ? AND user_id = ?", tenant.ID, owner).Error)
	assert.Equal(t, adminRole, member.RoleID, "owner role looked up by name, not a hard-coded UUID")
	assert.True(t, member.IsActive)

	var sub models.TenantSubscription
	require.NoError(t, db.First(&sub, "tenant_id = ?", tenant.ID).Error)
	assert.Equal(t, tierID, sub.TierID)

	_, err = svc.CreateTenant(ctx, req, uuid.New())
	assert.ErrorIs(t, err, ErrInvalidTenantSlug, "a live tenant still owns the slug")
}
