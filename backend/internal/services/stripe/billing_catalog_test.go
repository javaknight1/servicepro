package stripe

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeCatalog is an in-memory Stripe catalog that records idempotency keys.
type fakeCatalog struct {
	products map[string]bool
	prices   map[string]*CatalogPrice // by lookup key
	coupon   *CatalogCoupon
	idemKeys []string
	creates  int
}

func newFakeCatalog() *fakeCatalog {
	return &fakeCatalog{products: map[string]bool{}, prices: map[string]*CatalogPrice{}}
}

func (f *fakeCatalog) GetProduct(_ context.Context, id string) (bool, error) {
	return f.products[id], nil
}

func (f *fakeCatalog) CreateProduct(_ context.Context, id, _, idem string) error {
	f.products[id] = true
	f.idemKeys = append(f.idemKeys, idem)
	f.creates++
	return nil
}

func (f *fakeCatalog) PriceByLookupKey(_ context.Context, key string) (*CatalogPrice, error) {
	return f.prices[key], nil
}

func (f *fakeCatalog) CreatePrice(_ context.Context, product, key string, amount int64, interval, idem string) (*CatalogPrice, error) {
	p := &CatalogPrice{ID: "price_" + key, Product: product, UnitAmount: amount, Interval: interval, Currency: "usd"}
	f.prices[key] = p
	f.idemKeys = append(f.idemKeys, idem)
	f.creates++
	return p, nil
}

func (f *fakeCatalog) GetCoupon(_ context.Context, _ string) (*CatalogCoupon, error) {
	return f.coupon, nil
}

func (f *fakeCatalog) CreateFoundingCoupon(_ context.Context, idem string) (*CatalogCoupon, error) {
	f.coupon = &CatalogCoupon{ID: FoundingCouponID, AmountOff: FoundingCouponAmountOffCents, Currency: "usd",
		Duration: "forever", MaxRedemptions: FoundingCouponMaxRedemptions, Valid: true}
	f.idemKeys = append(f.idemKeys, idem)
	f.creates++
	return f.coupon, nil
}

func TestEnsureCatalog_CreatesOnceThenIsIdempotent(t *testing.T) {
	ctx := context.Background()
	f := newFakeCatalog()

	res, err := EnsureCatalog(ctx, f)
	require.NoError(t, err)
	assert.Equal(t, 10, f.creates, "3 products, 6 prices, 1 coupon")
	for _, r := range res {
		assert.Equal(t, "created", r.Action, r.Key)
	}
	assert.Equal(t, int64(9900), f.prices["sp_team_monthly"].UnitAmount)
	assert.Equal(t, int64(99000), f.prices["sp_team_annual"].UnitAmount)
	assert.Equal(t, "year", f.prices["sp_pro_annual"].Interval)
	assert.Equal(t, int64(199000), f.prices["sp_pro_annual"].UnitAmount)
	assert.Equal(t, int64(3900), f.prices["sp_solo_monthly"].UnitAmount)
	assert.Contains(t, f.idemKeys, "sp-catalog:coupon:sp_founding_team")
	assert.Contains(t, f.idemKeys, "sp-catalog:price:sp_team_monthly:9900:month")
	assert.Len(t, f.idemKeys, 10, "every create carries an idempotency key")

	res, err = EnsureCatalog(ctx, f)
	require.NoError(t, err)
	assert.Equal(t, 10, f.creates, "a second run creates nothing")
	for _, r := range res {
		assert.Equal(t, "exists", r.Action, r.Key)
	}
}

func TestEnsureCatalog_ReportsMismatchWithoutChanging(t *testing.T) {
	f := newFakeCatalog()
	f.products["sp_team"] = true
	f.prices["sp_team_monthly"] = &CatalogPrice{ID: "price_old", Product: "sp_team", UnitAmount: 7900, Interval: "month", Currency: "usd"}

	res, err := EnsureCatalog(context.Background(), f)
	require.NoError(t, err)
	var found bool
	for _, r := range res {
		if r.Key == "sp_team_monthly" {
			found = true
			assert.Equal(t, "mismatch", r.Action)
			assert.Equal(t, "price_old", r.ID)
		}
	}
	assert.True(t, found)
	assert.Equal(t, int64(7900), f.prices["sp_team_monthly"].UnitAmount, "existing price left alone")
}

func TestCatalogKeys(t *testing.T) {
	assert.Equal(t, "sp_solo_monthly", PriceLookupKey("solo", CycleMonthly))
	assert.Equal(t, "sp_team_annual", PriceLookupKey("team", CycleAnnual))
	assert.Equal(t, "sp_pro", ProductID("pro"))
	assert.Len(t, CatalogLookupKeys(), 6)

	tier, cycle, ok := TierCycleForLookupKey("sp_pro_annual")
	assert.True(t, ok)
	assert.Equal(t, "pro", tier)
	assert.Equal(t, CycleAnnual, cycle)
	_, _, ok = TierCycleForLookupKey("sp_basic_monthly")
	assert.False(t, ok, "legacy tiers are not in the catalog")

	for _, p := range PlanCatalog {
		assert.Equal(t, p.MonthlyCents*10, p.AnnualCents, "annual = 10x monthly (D15) for %s", p.Tier)
	}
}

func TestCouponRemaining(t *testing.T) {
	assert.Equal(t, int64(25), (&CatalogCoupon{Valid: true, MaxRedemptions: 25}).Remaining())
	assert.Equal(t, int64(1), (&CatalogCoupon{Valid: true, MaxRedemptions: 25, TimesRedeemed: 24}).Remaining())
	assert.Zero(t, (&CatalogCoupon{Valid: true, MaxRedemptions: 25, TimesRedeemed: 25}).Remaining())
	assert.Zero(t, (&CatalogCoupon{Valid: false, MaxRedemptions: 25}).Remaining())
	var nilCoupon *CatalogCoupon
	assert.Zero(t, nilCoupon.Remaining())
}
