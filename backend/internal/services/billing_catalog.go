package services

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	stripeClient "github.com/javaknight1/servicepro/backend/internal/services/stripe"
)

// Errors from the plan catalog lookup (D15).
var (
	// ErrTierNotInCatalog: the tier has no Stripe price (legacy Free/Basic).
	ErrTierNotInCatalog = errors.New("tier is not sold in the plan catalog")
	// ErrCatalogPriceMissing: the lookup key has no active price in Stripe
	// (run `make stripe-catalog` for this Stripe mode).
	ErrCatalogPriceMissing = errors.New("catalog price not found in Stripe")
)

// priceCacheTTL bounds how long a lookup_key -> price ID map is reused.
// Prices change only when the catalog script moves a lookup key.
const priceCacheTTL = 10 * time.Minute

// catalogPrices resolves catalog price IDs by Stripe lookup_key, caching
// the whole catalog (one List call) for priceCacheTTL. It replaces the old
// STRIPE_PRICE_* env vars.
type catalogPrices struct {
	api stripeClient.PriceLookup
	now func() time.Time

	mu      sync.Mutex
	ids     map[string]string
	fetched time.Time
}

func newCatalogPrices(api stripeClient.PriceLookup) *catalogPrices {
	return &catalogPrices{api: api, now: time.Now}
}

// priceID returns the active Stripe price ID of a catalog tier and cycle.
func (c *catalogPrices) priceID(ctx context.Context, tier, cycle string) (string, error) {
	if _, ok := stripeClient.CatalogPlanFor(tier); !ok {
		return "", fmt.Errorf("%w: %s", ErrTierNotInCatalog, tier)
	}
	if cycle != stripeClient.CycleMonthly && cycle != stripeClient.CycleAnnual {
		return "", ErrInvalidBillingCycle
	}
	key := stripeClient.PriceLookupKey(tier, cycle)

	c.mu.Lock()
	defer c.mu.Unlock()
	if c.ids == nil || c.now().Sub(c.fetched) > priceCacheTTL || c.ids[key] == "" {
		ids, err := c.api.PriceIDsByLookupKey(ctx, stripeClient.CatalogLookupKeys())
		if err != nil {
			return "", fmt.Errorf("look up catalog prices: %w", err)
		}
		c.ids, c.fetched = ids, c.now()
	}
	id := c.ids[key]
	if id == "" {
		return "", fmt.Errorf("%w: %s", ErrCatalogPriceMissing, key)
	}
	return id, nil
}
