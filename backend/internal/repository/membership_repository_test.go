package repository

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/javaknight1/servicepro/backend/internal/models"
	"github.com/javaknight1/servicepro/backend/internal/tenancy"
)

// Catalog tier IDs from migration 000014.
var (
	teamTierID = uuid.MustParse("00000000-0000-0000-0001-000000000005")
	soloTierID = uuid.MustParse("00000000-0000-0000-0001-000000000004")
)

func seedTrial(tb testing.TB, db *gorm.DB, tenantID uuid.UUID) {
	tb.Helper()
	end := time.Now().AddDate(0, 0, 30)
	require.NoError(tb, db.Create(&models.TenantSubscription{
		ID: uuid.New(), TenantID: tenantID, TierID: teamTierID, Status: models.MembershipStatusActive,
		BillingCycle: models.BillingCycleMonthly, StartedAt: time.Now(), CurrentPeriodStart: time.Now(),
		CurrentPeriodEnd: &end, TrialEndsAt: &end,
	}).Error)
}

// T210: activating a Checkout subscription ends only that tenant's trial
// (cross-tenant), and a redelivered event changes nothing (idempotent).
func TestActivateStripeSubscription_TenantScopedAndIdempotent(t *testing.T) {
	db := openTestDB(t)
	var n int64
	if err := db.Raw("SELECT count(*) FROM membership_tiers WHERE id = ?", soloTierID).Scan(&n).Error; err != nil || n == 0 {
		t.Skip("test database is not migrated to 000014")
	}
	sys := db.WithContext(tenancy.WithSystem(context.Background()))
	a, b := createTestTenant(t, sys), createTestTenant(t, sys)
	seedTrial(t, sys, a.ID)
	seedTrial(t, sys, b.ID)

	repo := NewMembershipRepository(db)
	ctx := tenancy.WithSystem(context.Background())
	end := time.Now().AddDate(0, 1, 0)
	act := StripeActivation{
		TierID: soloTierID, BillingCycle: models.BillingCycleMonthly, PriceCents: 3900,
		StripeSubscriptionID: "sub_test_" + uuid.NewString()[:8], StripePriceID: "price_test",
		CurrentPeriodEnd: &end, Reason: "checkout",
	}
	created, err := repo.ActivateStripeSubscription(ctx, a.ID, act)
	require.NoError(t, err)
	assert.True(t, created)

	subA, err := repo.GetTenantSubscription(ctx, a.ID)
	require.NoError(t, err)
	assert.Equal(t, soloTierID, subA.TierID)
	assert.Nil(t, subA.TrialEndsAt)
	require.NotNil(t, subA.StripeSubscriptionID)
	assert.Equal(t, act.StripeSubscriptionID, *subA.StripeSubscriptionID)

	subB, err := repo.GetTenantSubscription(ctx, b.ID)
	require.NoError(t, err)
	assert.Equal(t, teamTierID, subB.TierID, "tenant B's trial is untouched")
	assert.NotNil(t, subB.TrialEndsAt)

	// ActiveTierName reads each tenant's own plan (cross-tenant).
	nameA, err := repo.ActiveTierName(ctx, a.ID)
	require.NoError(t, err)
	assert.Equal(t, "solo", nameA)
	nameB, err := repo.ActiveTierName(ctx, b.ID)
	require.NoError(t, err)
	assert.Equal(t, "team", nameB)
	none, err := repo.ActiveTierName(ctx, uuid.New())
	require.NoError(t, err)
	assert.Empty(t, none)

	created, err = repo.ActivateStripeSubscription(ctx, a.ID, act)
	require.NoError(t, err)
	assert.False(t, created, "redelivery is a no-op")
	var rows int64
	require.NoError(t, sys.Model(&models.TenantSubscription{}).Where("tenant_id = ?", a.ID).Count(&rows).Error)
	assert.Equal(t, int64(2), rows, "trial row (ended) + one paid row")
}
