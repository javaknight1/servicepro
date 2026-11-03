package repository

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/javaknight1/servicepro/backend/internal/models"
	"github.com/javaknight1/servicepro/backend/internal/tenancy"
)

// T169: one connected account per tenant; the first stored ID wins.
func TestConnectedAccountRepository_CreateIfAbsent(t *testing.T) {
	db := openTestDB(t)
	repo := NewConnectedAccountRepository(db)
	a := createTestTenant(t, db)

	none, err := repo.GetByTenant(a.Ctx, a.ID)
	require.NoError(t, err)
	assert.Nil(t, none)

	acct, err := repo.CreateIfAbsent(a.Ctx, a.ID, "acct_a_"+a.ID.String()[:8], models.StripeAccountStatus{CardPaymentsStatus: "pending"})
	require.NoError(t, err)
	require.NotNil(t, acct)
	assert.Equal(t, a.ID, acct.TenantID)
	assert.Equal(t, "pending", acct.CardPaymentsStatus)

	again, err := repo.CreateIfAbsent(a.Ctx, a.ID, "acct_other_"+a.ID.String()[:8], models.StripeAccountStatus{})
	require.NoError(t, err)
	assert.Equal(t, acct.StripeAccountID, again.StripeAccountID, "a second create keeps the first account")

	require.NoError(t, repo.UpdateStatus(a.Ctx, a.ID, models.StripeAccountStatus{
		CardPaymentsStatus: models.CapabilityActive, ACHPaymentsStatus: models.CapabilityActive, PayoutsStatus: models.CapabilityActive,
	}))
	got, err := repo.GetByTenant(a.Ctx, a.ID)
	require.NoError(t, err)
	assert.True(t, got.CanAcceptCards())
	assert.True(t, got.CanAcceptACH())
	assert.True(t, got.OnboardingComplete())
}

// T169 cross-tenant: tenant B can neither read nor update tenant A's account,
// and a webhook lookup by Stripe account ID resolves to A only.
func TestConnectedAccountRepository_CrossTenant(t *testing.T) {
	db := openTestDB(t)
	repo := NewConnectedAccountRepository(db)
	a := createTestTenant(t, db)
	b := createTestTenant(t, db)
	acctID := "acct_x_" + a.ID.String()[:8]

	_, err := repo.CreateIfAbsent(a.Ctx, a.ID, acctID, models.StripeAccountStatus{CardPaymentsStatus: "pending"})
	require.NoError(t, err)

	// B asking for A's tenant ID: the tenancy plugin adds tenant_id = B.
	leaked, err := repo.GetByTenant(b.Ctx, a.ID)
	require.NoError(t, err)
	assert.Nil(t, leaked, "B must not see A's connected account")

	byStripe, err := repo.GetByStripeAccountID(b.Ctx, acctID)
	require.NoError(t, err)
	assert.Nil(t, byStripe, "B's context must not resolve A's Stripe account")

	require.NoError(t, repo.UpdateStatus(b.Ctx, a.ID, models.StripeAccountStatus{CardPaymentsStatus: models.CapabilityActive}))
	stillA, err := repo.GetByTenant(a.Ctx, a.ID)
	require.NoError(t, err)
	assert.Equal(t, "pending", stillA.CardPaymentsStatus, "B's update must not touch A's row")

	// Webhook path: system context resolves the owning tenant.
	sys, err := repo.GetByStripeAccountID(tenancy.WithSystem(t.Context()), acctID)
	require.NoError(t, err)
	require.NotNil(t, sys)
	assert.Equal(t, a.ID, sys.TenantID)

	// Without any tenant context the query fails closed.
	_, err = repo.GetByTenant(t.Context(), a.ID)
	assert.ErrorIs(t, err, tenancy.ErrNoTenant)
}
