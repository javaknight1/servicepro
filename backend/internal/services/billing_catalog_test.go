package services

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakePriceLookup struct {
	ids   map[string]string
	calls int
}

func (f *fakePriceLookup) PriceIDsByLookupKey(_ context.Context, keys []string) (map[string]string, error) {
	f.calls++
	out := map[string]string{}
	for _, k := range keys {
		if id, ok := f.ids[k]; ok {
			out[k] = id
		}
	}
	return out, nil
}

// D15: prices are found by lookup key, not STRIPE_PRICE_* env vars.
func TestCatalogPrices_LookupByKeyWithCache(t *testing.T) {
	ctx := context.Background()
	api := &fakePriceLookup{ids: map[string]string{
		"sp_team_monthly": "price_team_m", "sp_team_annual": "price_team_y", "sp_solo_monthly": "price_solo_m",
	}}
	now := time.Date(2026, 11, 3, 12, 0, 0, 0, time.UTC)
	c := newCatalogPrices(api)
	c.now = func() time.Time { return now }

	id, err := c.priceID(ctx, "team", "monthly")
	require.NoError(t, err)
	assert.Equal(t, "price_team_m", id)
	id, err = c.priceID(ctx, "team", "annual")
	require.NoError(t, err)
	assert.Equal(t, "price_team_y", id)
	assert.Equal(t, 1, api.calls, "one List call serves the whole catalog")

	now = now.Add(priceCacheTTL + time.Second)
	_, err = c.priceID(ctx, "solo", "monthly")
	require.NoError(t, err)
	assert.Equal(t, 2, api.calls, "refreshed after the TTL")

	_, err = c.priceID(ctx, "pro", "monthly")
	assert.ErrorIs(t, err, ErrCatalogPriceMissing)

	_, err = c.priceID(ctx, "basic", "monthly")
	assert.ErrorIs(t, err, ErrTierNotInCatalog, "legacy tiers can't be bought")
	_, err = c.priceID(ctx, "free", "monthly")
	assert.ErrorIs(t, err, ErrTierNotInCatalog)

	_, err = c.priceID(ctx, "team", "weekly")
	assert.ErrorIs(t, err, ErrInvalidBillingCycle)
}
