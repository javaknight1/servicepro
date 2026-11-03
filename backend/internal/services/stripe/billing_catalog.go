package stripe

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/stripe/stripe-go/v87"
)

// ServicePro's own plan catalog on the platform account (D15, T210).
// Products have fixed IDs (sp_solo, sp_team, sp_pro) and prices are found by
// lookup_key (sp_<tier>_monthly / sp_<tier>_annual), so no Stripe price ID
// lives in config. EnsureCatalog creates whatever is missing, idempotently.

// Billing cycles as used in lookup keys (match models.BillingCycle values).
const (
	CycleMonthly = "monthly"
	CycleAnnual  = "annual"
)

// CatalogPlan is one plan in the Stripe catalog. Amounts are USD cents.
type CatalogPlan struct {
	Tier         string // membership_tiers.name
	Name         string // Stripe product name
	MonthlyCents int64
	AnnualCents  int64
}

// PlanCatalog is D15: Solo $39, Team $99, Pro $199 a month; annual = 10x.
// It must match migration 000014's membership_tiers prices.
var PlanCatalog = []CatalogPlan{
	{Tier: "solo", Name: "ServicePro Solo", MonthlyCents: 3900, AnnualCents: 39000},
	{Tier: "team", Name: "ServicePro Team", MonthlyCents: 9900, AnnualCents: 99000},
	{Tier: "pro", Name: "ServicePro Pro", MonthlyCents: 19900, AnnualCents: 199000},
}

// Founding offer (D15): Team at $59/mo for life = $40 off Team, forever, for
// the first 25 redemptions. Applied by ServicePro at checkout to Team monthly
// only (Stripe coupons restrict by product, not price).
const (
	FoundingCouponID             = "sp_founding_team"
	FoundingCouponName           = "Founding Team"
	FoundingCouponAmountOffCents = 4000
	FoundingCouponMaxRedemptions = 25
	FoundingCouponTier           = "team"
)

// CatalogCurrency is the currency of every catalog price.
const CatalogCurrency = "usd"

// ProductID is the fixed Stripe product ID of a tier.
func ProductID(tier string) string { return "sp_" + tier }

// PriceLookupKey is the Stripe price lookup_key of a tier and billing cycle.
func PriceLookupKey(tier, cycle string) string { return fmt.Sprintf("sp_%s_%s", tier, cycle) }

// CatalogPlanFor returns the catalog plan of a tier.
func CatalogPlanFor(tier string) (CatalogPlan, bool) {
	for _, p := range PlanCatalog {
		if p.Tier == tier {
			return p, true
		}
	}
	return CatalogPlan{}, false
}

// TierCycleForLookupKey maps a catalog lookup key back to its tier and cycle.
func TierCycleForLookupKey(key string) (tier, cycle string, ok bool) {
	for _, p := range PlanCatalog {
		for _, c := range []string{CycleMonthly, CycleAnnual} {
			if PriceLookupKey(p.Tier, c) == key {
				return p.Tier, c, true
			}
		}
	}
	return "", "", false
}

// CatalogLookupKeys lists every catalog price lookup key.
func CatalogLookupKeys() []string {
	keys := make([]string, 0, len(PlanCatalog)*2)
	for _, p := range PlanCatalog {
		keys = append(keys, PriceLookupKey(p.Tier, CycleMonthly), PriceLookupKey(p.Tier, CycleAnnual))
	}
	return keys
}

// CatalogPrice is the part of a Stripe price the catalog checks.
type CatalogPrice struct {
	ID         string
	Product    string
	UnitAmount int64
	Interval   string // month | year
	Currency   string
}

// CatalogCoupon is the part of a Stripe coupon the catalog and the founding
// offer need.
type CatalogCoupon struct {
	ID             string
	AmountOff      int64
	Currency       string
	Duration       string
	MaxRedemptions int64
	TimesRedeemed  int64
	Valid          bool
}

// Remaining is how many more times the coupon can be redeemed (0 when it is
// no longer valid).
func (c *CatalogCoupon) Remaining() int64 {
	if c == nil || !c.Valid {
		return 0
	}
	if c.MaxRedemptions <= 0 {
		return 1 << 31 // no cap
	}
	if r := c.MaxRedemptions - c.TimesRedeemed; r > 0 {
		return r
	}
	return 0
}

// CatalogAPI is the Stripe surface EnsureCatalog uses (faked in tests).
// Lookups return (nil, nil) when the object does not exist.
type CatalogAPI interface {
	GetProduct(ctx context.Context, id string) (exists bool, err error)
	CreateProduct(ctx context.Context, id, name, idempotencyKey string) error
	PriceByLookupKey(ctx context.Context, key string) (*CatalogPrice, error)
	CreatePrice(ctx context.Context, product, lookupKey string, unitAmount int64, interval, idempotencyKey string) (*CatalogPrice, error)
	GetCoupon(ctx context.Context, id string) (*CatalogCoupon, error)
	CreateFoundingCoupon(ctx context.Context, idempotencyKey string) (*CatalogCoupon, error)
}

// CatalogResult is one line of what EnsureCatalog found or did.
type CatalogResult struct {
	Kind   string // product | price | coupon
	Key    string // product ID, lookup key or coupon ID
	ID     string // Stripe ID
	Action string // created | exists | mismatch
	Note   string
}

// EnsureCatalog makes sure every product, price and the founding coupon
// exist. It never changes or archives an existing object: a price whose
// amount or interval differs from the catalog is reported as a mismatch
// (prices are immutable; fix it in the Dashboard or move the lookup key).
// Every create carries a deterministic idempotency key.
func EnsureCatalog(ctx context.Context, api CatalogAPI) ([]CatalogResult, error) {
	var out []CatalogResult
	for _, p := range PlanCatalog {
		pid := ProductID(p.Tier)
		exists, err := api.GetProduct(ctx, pid)
		if err != nil {
			return out, fmt.Errorf("get product %s: %w", pid, err)
		}
		if exists {
			out = append(out, CatalogResult{Kind: "product", Key: pid, ID: pid, Action: "exists"})
		} else {
			if err := api.CreateProduct(ctx, pid, p.Name, "sp-catalog:product:"+pid); err != nil {
				return out, fmt.Errorf("create product %s: %w", pid, err)
			}
			out = append(out, CatalogResult{Kind: "product", Key: pid, ID: pid, Action: "created"})
		}

		for _, c := range []struct {
			cycle, interval string
			amount          int64
		}{{CycleMonthly, "month", p.MonthlyCents}, {CycleAnnual, "year", p.AnnualCents}} {
			key := PriceLookupKey(p.Tier, c.cycle)
			price, err := api.PriceByLookupKey(ctx, key)
			if err != nil {
				return out, fmt.Errorf("look up price %s: %w", key, err)
			}
			if price == nil {
				idem := fmt.Sprintf("sp-catalog:price:%s:%d:%s", key, c.amount, c.interval)
				created, err := api.CreatePrice(ctx, pid, key, c.amount, c.interval, idem)
				if err != nil {
					return out, fmt.Errorf("create price %s: %w", key, err)
				}
				out = append(out, CatalogResult{Kind: "price", Key: key, ID: created.ID, Action: "created"})
				continue
			}
			r := CatalogResult{Kind: "price", Key: key, ID: price.ID, Action: "exists"}
			if price.UnitAmount != c.amount || price.Interval != c.interval || price.Product != pid || price.Currency != CatalogCurrency {
				r.Action = "mismatch"
				r.Note = fmt.Sprintf("want %d %s/%s on %s, have %d %s/%s on %s",
					c.amount, CatalogCurrency, c.interval, pid, price.UnitAmount, price.Currency, price.Interval, price.Product)
			}
			out = append(out, r)
		}
	}

	coupon, err := api.GetCoupon(ctx, FoundingCouponID)
	if err != nil {
		return out, fmt.Errorf("get coupon %s: %w", FoundingCouponID, err)
	}
	if coupon == nil {
		created, err := api.CreateFoundingCoupon(ctx, "sp-catalog:coupon:"+FoundingCouponID)
		if err != nil {
			return out, fmt.Errorf("create coupon %s: %w", FoundingCouponID, err)
		}
		out = append(out, CatalogResult{Kind: "coupon", Key: FoundingCouponID, ID: created.ID, Action: "created"})
	} else {
		r := CatalogResult{Kind: "coupon", Key: FoundingCouponID, ID: coupon.ID, Action: "exists",
			Note: fmt.Sprintf("%d of %d redeemed", coupon.TimesRedeemed, coupon.MaxRedemptions)}
		if coupon.AmountOff != FoundingCouponAmountOffCents || coupon.Duration != "forever" || coupon.MaxRedemptions != FoundingCouponMaxRedemptions {
			r.Action = "mismatch"
			r.Note = fmt.Sprintf("want %d off forever, max %d; have %d off %s, max %d",
				FoundingCouponAmountOffCents, FoundingCouponMaxRedemptions, coupon.AmountOff, coupon.Duration, coupon.MaxRedemptions)
		}
		out = append(out, r)
	}
	return out, nil
}

// PriceLookup finds catalog price IDs by lookup key (faked in tests).
type PriceLookup interface {
	// PriceIDsByLookupKey returns active price IDs keyed by lookup key;
	// missing keys are absent from the map.
	PriceIDsByLookupKey(ctx context.Context, keys []string) (map[string]string, error)
}

// BillingClient implements CatalogAPI and the platform-billing calls with
// the platform key.
type BillingClient struct {
	api *stripe.Client
}

// NewBillingClient creates a billing client for the platform's secret key.
func NewBillingClient(secretKey string) *BillingClient {
	return &BillingClient{api: stripe.NewClient(secretKey)}
}

var (
	_ CatalogAPI  = (*BillingClient)(nil)
	_ PriceLookup = (*BillingClient)(nil)
)

func isNotFound(err error) bool {
	var se *stripe.Error
	return errors.As(err, &se) && (se.HTTPStatusCode == http.StatusNotFound || se.Code == stripe.ErrorCodeResourceMissing)
}

// GetProduct reports whether a product with this ID exists.
func (c *BillingClient) GetProduct(ctx context.Context, id string) (bool, error) {
	_, err := c.api.V1Products.Retrieve(ctx, id, nil)
	if err != nil {
		if isNotFound(err) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

// CreateProduct creates a product with a fixed ID.
func (c *BillingClient) CreateProduct(ctx context.Context, id, name, idempotencyKey string) error {
	params := &stripe.ProductCreateParams{
		ID:       stripe.String(id),
		Name:     stripe.String(name),
		Metadata: map[string]string{"app": "servicepro", "catalog": "d15"},
	}
	params.SetIdempotencyKey(idempotencyKey)
	_, err := c.api.V1Products.Create(ctx, params)
	return err
}

// PriceByLookupKey returns the active price with this lookup key, or nil.
func (c *BillingClient) PriceByLookupKey(ctx context.Context, key string) (*CatalogPrice, error) {
	params := &stripe.PriceListParams{Active: stripe.Bool(true), LookupKeys: stripe.StringSlice([]string{key})}
	for p, err := range c.api.V1Prices.List(ctx, params).All(ctx) {
		if err != nil {
			return nil, err
		}
		return toCatalogPrice(p), nil
	}
	return nil, nil
}

// CreatePrice creates a recurring USD price with a lookup key.
func (c *BillingClient) CreatePrice(ctx context.Context, product, lookupKey string, unitAmount int64, interval, idempotencyKey string) (*CatalogPrice, error) {
	params := &stripe.PriceCreateParams{
		Product:    stripe.String(product),
		Currency:   stripe.String(CatalogCurrency),
		UnitAmount: stripe.Int64(unitAmount),
		LookupKey:  stripe.String(lookupKey),
		Nickname:   stripe.String(lookupKey),
		Recurring:  &stripe.PriceCreateRecurringParams{Interval: stripe.String(interval)},
	}
	params.SetIdempotencyKey(idempotencyKey)
	p, err := c.api.V1Prices.Create(ctx, params)
	if err != nil {
		return nil, err
	}
	return toCatalogPrice(p), nil
}

func toCatalogPrice(p *stripe.Price) *CatalogPrice {
	cp := &CatalogPrice{ID: p.ID, UnitAmount: p.UnitAmount, Currency: string(p.Currency)}
	if p.Product != nil {
		cp.Product = p.Product.ID
	}
	if p.Recurring != nil {
		cp.Interval = string(p.Recurring.Interval)
	}
	return cp
}

// PriceIDsByLookupKey returns active price IDs keyed by lookup key.
func (c *BillingClient) PriceIDsByLookupKey(ctx context.Context, keys []string) (map[string]string, error) {
	params := &stripe.PriceListParams{Active: stripe.Bool(true), LookupKeys: stripe.StringSlice(keys)}
	out := make(map[string]string, len(keys))
	for p, err := range c.api.V1Prices.List(ctx, params).All(ctx) {
		if err != nil {
			return nil, err
		}
		if p.LookupKey != "" {
			out[p.LookupKey] = p.ID
		}
	}
	return out, nil
}

// GetCoupon returns the coupon, or nil when it doesn't exist.
func (c *BillingClient) GetCoupon(ctx context.Context, id string) (*CatalogCoupon, error) {
	cp, err := c.api.V1Coupons.Retrieve(ctx, id, nil)
	if err != nil {
		if isNotFound(err) {
			return nil, nil
		}
		return nil, err
	}
	return &CatalogCoupon{
		ID:             cp.ID,
		AmountOff:      cp.AmountOff,
		Currency:       string(cp.Currency),
		Duration:       string(cp.Duration),
		MaxRedemptions: cp.MaxRedemptions,
		TimesRedeemed:  cp.TimesRedeemed,
		Valid:          cp.Valid,
	}, nil
}

// CreateFoundingCoupon creates the founding coupon: $40 off Team, forever,
// max 25 redemptions.
func (c *BillingClient) CreateFoundingCoupon(ctx context.Context, idempotencyKey string) (*CatalogCoupon, error) {
	params := &stripe.CouponCreateParams{
		ID:             stripe.String(FoundingCouponID),
		Name:           stripe.String(FoundingCouponName),
		AmountOff:      stripe.Int64(FoundingCouponAmountOffCents),
		Currency:       stripe.String(CatalogCurrency),
		Duration:       stripe.String(string(stripe.CouponDurationForever)),
		MaxRedemptions: stripe.Int64(FoundingCouponMaxRedemptions),
		AppliesTo: &stripe.CouponCreateAppliesToParams{
			Products: stripe.StringSlice([]string{ProductID(FoundingCouponTier)}),
		},
		Metadata: map[string]string{"app": "servicepro", "offer": "founding"},
	}
	params.SetIdempotencyKey(idempotencyKey)
	cp, err := c.api.V1Coupons.Create(ctx, params)
	if err != nil {
		return nil, err
	}
	return &CatalogCoupon{ID: cp.ID, AmountOff: cp.AmountOff, Currency: string(cp.Currency), Duration: string(cp.Duration),
		MaxRedemptions: cp.MaxRedemptions, TimesRedeemed: cp.TimesRedeemed, Valid: cp.Valid}, nil
}
