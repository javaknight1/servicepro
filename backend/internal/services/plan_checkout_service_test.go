package services

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stripe/stripe-go/v87"

	"github.com/javaknight1/servicepro/backend/internal/models"
	"github.com/javaknight1/servicepro/backend/internal/repository"
	stripeClient "github.com/javaknight1/servicepro/backend/internal/services/stripe"
)

var (
	soloTier  = &models.MembershipTier{ID: uuid.New(), Name: "solo", MonthlyPriceCents: 3900, AnnualPriceCents: 39000, Enabled: true}
	teamTier  = &models.MembershipTier{ID: uuid.New(), Name: "team", MonthlyPriceCents: 9900, AnnualPriceCents: 99000, Enabled: true}
	basicTier = &models.MembershipTier{ID: uuid.New(), Name: "basic", MonthlyPriceCents: 2900, Enabled: true}
)

type fakePlanStore struct {
	tiers map[uuid.UUID]*models.MembershipTier
	subs  map[uuid.UUID]*models.TenantSubscription
}

func (f *fakePlanStore) GetTierByID(_ context.Context, id uuid.UUID) (*models.MembershipTier, error) {
	if t, ok := f.tiers[id]; ok {
		return t, nil
	}
	return nil, repository.ErrTierNotFound
}

func (f *fakePlanStore) GetTenantSubscription(_ context.Context, tenantID uuid.UUID) (*models.TenantSubscription, error) {
	if s, ok := f.subs[tenantID]; ok {
		return s, nil
	}
	return nil, repository.ErrSubscriptionNotFound
}

type fakeCustomers struct{ calls int }

func (f *fakeCustomers) GetOrCreateStripeCustomer(_ context.Context, tenantID uuid.UUID) (string, error) {
	f.calls++
	return "cus_" + tenantID.String()[:8], nil
}

type fakePrices struct{}

func (fakePrices) priceID(_ context.Context, tier, cycle string) (string, error) {
	return "price_" + stripeClient.PriceLookupKey(tier, cycle), nil
}

type fakePlanAPI struct {
	coupon      *stripeClient.CatalogCoupon
	couponCalls int
	sessions    []stripeClient.SubscriptionCheckoutInput
}

func (f *fakePlanAPI) GetCoupon(_ context.Context, _ string) (*stripeClient.CatalogCoupon, error) {
	f.couponCalls++
	return f.coupon, nil
}

func (f *fakePlanAPI) CreateSubscriptionCheckout(_ context.Context, in stripeClient.SubscriptionCheckoutInput) (string, error) {
	f.sessions = append(f.sessions, in)
	return "https://checkout.stripe.test/s/" + in.PriceID, nil
}

type planFixture struct {
	svc    *PlanCheckoutService
	api    *fakePlanAPI
	store  *fakePlanStore
	tenant uuid.UUID
	now    time.Time
}

func newPlanFixture(paymentsEnabled bool, remaining int64) *planFixture {
	f := &planFixture{
		api: &fakePlanAPI{coupon: &stripeClient.CatalogCoupon{ID: stripeClient.FoundingCouponID, Valid: true,
			MaxRedemptions: 25, TimesRedeemed: 25 - remaining}},
		store: &fakePlanStore{
			tiers: map[uuid.UUID]*models.MembershipTier{soloTier.ID: soloTier, teamTier.ID: teamTier, basicTier.ID: basicTier},
			subs:  map[uuid.UUID]*models.TenantSubscription{},
		},
		tenant: uuid.New(),
		now:    time.Date(2026, 11, 3, 15, 0, 0, 0, time.UTC),
	}
	f.svc = NewPlanCheckoutService(PlanCheckoutConfig{
		Store: f.store, Customers: &fakeCustomers{}, Prices: fakePrices{}, API: f.api,
		PaymentsEnabled: paymentsEnabled, FrontendURL: "https://app.test",
	})
	f.svc.now = func() time.Time { return f.now }
	return f
}

func (f *planFixture) checkout(t *testing.T, tier *models.MembershipTier, cycle models.BillingCycle) (*models.PlanCheckoutResponse, stripeClient.SubscriptionCheckoutInput) {
	t.Helper()
	resp, err := f.svc.StartCheckout(context.Background(), f.tenant, &models.PlanCheckoutRequest{TierID: tier.ID, BillingCycle: cycle})
	require.NoError(t, err)
	return resp, f.api.sessions[len(f.api.sessions)-1]
}

// D15/D10: the founding coupon goes on Team monthly only, only while invoice
// payments are on and redemptions remain.
func TestPlanCheckout_FoundingOfferEligibility(t *testing.T) {
	cases := []struct {
		name      string
		payments  bool
		remaining int64
		tier      *models.MembershipTier
		cycle     models.BillingCycle
		want      bool
	}{
		{"team monthly, payments on, redemptions left", true, 3, teamTier, models.BillingCycleMonthly, true},
		{"payments off", false, 3, teamTier, models.BillingCycleMonthly, false},
		{"no redemptions left", true, 0, teamTier, models.BillingCycleMonthly, false},
		{"team annual", true, 3, teamTier, models.BillingCycleAnnual, false},
		{"solo monthly", true, 3, soloTier, models.BillingCycleMonthly, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			f := newPlanFixture(tc.payments, tc.remaining)
			resp, in := f.checkout(t, tc.tier, tc.cycle)
			assert.Equal(t, tc.want, resp.FoundingOffer)
			if tc.want {
				assert.Equal(t, stripeClient.FoundingCouponID, in.CouponID)
				assert.Equal(t, "true", in.Metadata[stripeClient.MetaFoundingOffer])
			} else {
				assert.Empty(t, in.CouponID)
				assert.Empty(t, in.Metadata[stripeClient.MetaFoundingOffer])
			}
		})
	}
}

func TestPlanCheckout_FoundingOfferCachedBriefly(t *testing.T) {
	f := newPlanFixture(true, 2)
	ctx := context.Background()
	assert.True(t, f.svc.FoundingOfferAvailable(ctx))
	assert.True(t, f.svc.FoundingOfferAvailable(ctx))
	assert.Equal(t, 1, f.api.couponCalls, "cached within the TTL")

	f.api.coupon.TimesRedeemed = 25
	f.now = f.now.Add(couponCacheTTL + time.Second)
	assert.False(t, f.svc.FoundingOfferAvailable(ctx), "re-checked after the TTL: sold out")
	assert.Equal(t, 2, f.api.couponCalls)

	f.api.coupon = nil // coupon missing in this Stripe mode
	f.now = f.now.Add(couponCacheTTL + time.Second)
	assert.False(t, f.svc.FoundingOfferAvailable(ctx))
}

func TestPlanCheckout_SessionShapeAndTrialCarryOver(t *testing.T) {
	f := newPlanFixture(false, 0)
	trialEnd := f.now.AddDate(0, 0, 20)
	f.store.subs[f.tenant] = &models.TenantSubscription{TenantID: f.tenant, TierID: teamTier.ID, TrialEndsAt: &trialEnd}

	resp, in := f.checkout(t, soloTier, models.BillingCycleAnnual)
	assert.Contains(t, resp.URL, "price_sp_solo_annual")
	assert.Equal(t, "price_sp_solo_annual", in.PriceID)
	assert.Equal(t, trialEnd.Unix(), in.TrialEnd, "remaining trial days carry over")
	assert.Equal(t, f.tenant.String(), in.Metadata[stripeClient.MetaTenantID])
	assert.Equal(t, "solo", in.Metadata[stripeClient.MetaTier])
	assert.Equal(t, "annual", in.Metadata[stripeClient.MetaBillingCycle])
	assert.Contains(t, in.SuccessURL, "checkout=success")
	assert.NotEmpty(t, in.IdempotencyKey)

	_, again := f.checkout(t, soloTier, models.BillingCycleAnnual)
	assert.Equal(t, in.IdempotencyKey, again.IdempotencyKey, "a double-click reuses one session")

	soon := f.now.Add(24 * time.Hour)
	f.store.subs[f.tenant].TrialEndsAt = &soon
	_, in = f.checkout(t, soloTier, models.BillingCycleAnnual)
	assert.Zero(t, in.TrialEnd, "under Stripe's 48h minimum: billed now")
}

func TestPlanCheckout_Rejections(t *testing.T) {
	ctx := context.Background()
	f := newPlanFixture(true, 5)

	_, err := f.svc.StartCheckout(ctx, f.tenant, &models.PlanCheckoutRequest{TierID: basicTier.ID, BillingCycle: models.BillingCycleMonthly})
	assert.ErrorIs(t, err, ErrPlanNotAvailable, "legacy Basic can't be bought")
	_, err = f.svc.StartCheckout(ctx, f.tenant, &models.PlanCheckoutRequest{TierID: uuid.New(), BillingCycle: models.BillingCycleMonthly})
	assert.ErrorIs(t, err, ErrPlanNotAvailable)

	subID := "sub_123"
	f.store.subs[f.tenant] = &models.TenantSubscription{TenantID: f.tenant, StripeSubscriptionID: &subID}
	_, err = f.svc.StartCheckout(ctx, f.tenant, &models.PlanCheckoutRequest{TierID: teamTier.ID, BillingCycle: models.BillingCycleMonthly})
	assert.ErrorIs(t, err, ErrAlreadySubscribed)

	off := NewPlanCheckoutService(PlanCheckoutConfig{Store: f.store})
	_, err = off.StartCheckout(ctx, f.tenant, &models.PlanCheckoutRequest{TierID: teamTier.ID, BillingCycle: models.BillingCycleMonthly})
	assert.ErrorIs(t, err, ErrPlanBillingUnavailable)
	assert.False(t, off.FoundingOfferAvailable(ctx))
	assert.Empty(t, f.api.sessions, "no session for any rejection")
}

func TestNewTrialInfo(t *testing.T) {
	now := time.Date(2026, 11, 3, 12, 0, 0, 0, time.UTC)
	end := now.Add(29*24*time.Hour + time.Hour)
	info := models.NewTrialInfo(&models.TenantSubscription{TrialEndsAt: &end}, now)
	require.NotNil(t, info)
	assert.Equal(t, 30, info.DaysLeft, "rounded up")
	assert.False(t, info.Ended)

	past := now.Add(-time.Hour)
	info = models.NewTrialInfo(&models.TenantSubscription{TrialEndsAt: &past}, now)
	require.NotNil(t, info)
	assert.True(t, info.Ended)
	assert.Zero(t, info.DaysLeft)

	sub := "sub_1"
	assert.Nil(t, models.NewTrialInfo(&models.TenantSubscription{TrialEndsAt: &end, StripeSubscriptionID: &sub}, now), "a paid plan is not a trial")
	assert.Nil(t, models.NewTrialInfo(&models.TenantSubscription{}, now))
}

// --- webhook activation ---

type fakeActivationStore struct {
	tenants     map[string]uuid.UUID // customer -> tenant
	activations map[string]repository.StripeActivation
	tenantOf    map[string]uuid.UUID
}

func newFakeActivationStore() *fakeActivationStore {
	return &fakeActivationStore{tenants: map[string]uuid.UUID{}, activations: map[string]repository.StripeActivation{}, tenantOf: map[string]uuid.UUID{}}
}

func (f *fakeActivationStore) TenantIDForStripeCustomer(_ context.Context, id string) (uuid.UUID, error) {
	if t, ok := f.tenants[id]; ok {
		return t, nil
	}
	return uuid.Nil, repository.ErrTenantNotFound
}

func (f *fakeActivationStore) CatalogTier(_ context.Context, name string) (*models.MembershipTier, error) {
	for _, t := range []*models.MembershipTier{soloTier, teamTier} {
		if t.Name == name {
			return t, nil
		}
	}
	return nil, repository.ErrTierNotFound
}

// ActivateStripeSubscription mirrors the repository's idempotency.
func (f *fakeActivationStore) ActivateStripeSubscription(_ context.Context, tenantID uuid.UUID, a repository.StripeActivation) (bool, error) {
	if _, ok := f.activations[a.StripeSubscriptionID]; ok {
		return false, nil
	}
	f.activations[a.StripeSubscriptionID] = a
	f.tenantOf[a.StripeSubscriptionID] = tenantID
	return true, nil
}

func checkoutSub(id, customer, lookupKey string, amount int64, meta map[string]string) *stripe.Subscription {
	return &stripe.Subscription{
		ID:       id,
		Status:   stripe.SubscriptionStatusActive,
		Customer: &stripe.Customer{ID: customer},
		Metadata: meta,
		Items: &stripe.SubscriptionItemList{Data: []*stripe.SubscriptionItem{{
			Price: &stripe.Price{ID: "price_x", LookupKey: lookupKey, UnitAmount: amount},
		}}},
	}
}

func TestActivateFromStripe(t *testing.T) {
	ctx := context.Background()
	store := newFakeActivationStore()
	tenantA, tenantB := uuid.New(), uuid.New()
	store.tenants["cus_a"] = tenantA
	store.tenants["cus_b"] = tenantB
	end := time.Now().AddDate(0, 1, 0)

	// Founding Team monthly: $99 - $40 locked in.
	sub := checkoutSub("sub_a", "cus_a", "sp_team_monthly", 9900,
		map[string]string{stripeClient.MetaTenantID: tenantA.String(), stripeClient.MetaFoundingOffer: "true"})
	require.NoError(t, activateFromStripe(ctx, store, sub, &end))
	a := store.activations["sub_a"]
	assert.Equal(t, tenantA, store.tenantOf["sub_a"])
	assert.Equal(t, teamTier.ID, a.TierID)
	assert.Equal(t, models.BillingCycleMonthly, a.BillingCycle)
	assert.Equal(t, 5900, a.PriceCents)

	// Redelivery / created+updated pair: still one activation.
	require.NoError(t, activateFromStripe(ctx, store, sub, &end))
	assert.Len(t, store.activations, 1)

	// Metadata naming another tenant than the customer's owner: ignored.
	require.NoError(t, activateFromStripe(ctx, store,
		checkoutSub("sub_x", "cus_b", "sp_solo_monthly", 3900, map[string]string{stripeClient.MetaTenantID: tenantA.String()}), &end))
	// Unknown customer, non-catalog price: acknowledged, ignored.
	require.NoError(t, activateFromStripe(ctx, store, checkoutSub("sub_y", "cus_zzz", "sp_solo_monthly", 3900, nil), &end))
	require.NoError(t, activateFromStripe(ctx, store, checkoutSub("sub_z", "cus_b", "legacy_basic", 2900, nil), &end))
	assert.Len(t, store.activations, 1)

	// Plain Solo annual for tenant B.
	require.NoError(t, activateFromStripe(ctx, store, checkoutSub("sub_b", "cus_b", "sp_solo_annual", 39000, nil), &end))
	assert.Equal(t, tenantB, store.tenantOf["sub_b"])
	assert.Equal(t, 39000, store.activations["sub_b"].PriceCents)
	assert.Equal(t, models.BillingCycleAnnual, store.activations["sub_b"].BillingCycle)
}
