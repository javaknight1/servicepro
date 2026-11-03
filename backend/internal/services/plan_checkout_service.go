package services

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/stripe/stripe-go/v87"

	apperrors "github.com/javaknight1/servicepro/backend/internal/errors"
	"github.com/javaknight1/servicepro/backend/internal/models"
	"github.com/javaknight1/servicepro/backend/internal/repository"
	stripeClient "github.com/javaknight1/servicepro/backend/internal/services/stripe"
	"github.com/javaknight1/servicepro/backend/pkg/clients/logging"
)

// Plan checkout (D15, T210): a tenant picks Solo, Team or Pro and pays in a
// Stripe Checkout session on the platform account. The new subscription
// becomes the tenant's plan only when the signature-verified, deduplicated
// billing webhook reports it active (activateFromStripe), never on the
// redirect back.

// Errors returned by plan checkout.
var (
	ErrPlanBillingUnavailable = apperrors.New(http.StatusServiceUnavailable, "billing_not_configured",
		"Plan billing isn't available on this server yet")
	ErrPlanNotAvailable = apperrors.New(http.StatusNotFound, "tier_not_found",
		"That plan isn't available")
	ErrAlreadySubscribed = apperrors.New(http.StatusConflict, "already_subscribed",
		"Your organization already has a paid plan. Change it from the plan page instead")
	ErrPlanCheckoutFailed = apperrors.New(http.StatusBadGateway, "stripe_unavailable",
		"Stripe couldn't be reached. Please try again")
)

// couponCacheTTL bounds how stale the founding offer's remaining
// redemptions may be.
const couponCacheTTL = time.Minute

// minCheckoutTrial is Stripe's minimum for subscription_data.trial_end in
// Checkout; a trial ending sooner than this is not carried over.
const minCheckoutTrial = 48 * time.Hour

type planCheckoutStore interface {
	GetTierByID(ctx context.Context, id uuid.UUID) (*models.MembershipTier, error)
	GetTenantSubscription(ctx context.Context, tenantID uuid.UUID) (*models.TenantSubscription, error)
}

type stripeCustomerSource interface {
	GetOrCreateStripeCustomer(ctx context.Context, tenantID uuid.UUID) (string, error)
}

type priceResolver interface {
	priceID(ctx context.Context, tier, cycle string) (string, error)
}

// PlanCheckoutService starts plan Checkout sessions and decides the founding
// offer.
type PlanCheckoutService struct {
	store           planCheckoutStore
	customers       stripeCustomerSource
	prices          priceResolver
	api             stripeClient.PlanBillingAPI // nil when Stripe isn't configured
	paymentsEnabled bool                        // INVOICE_PAYMENTS_ENABLED (D10: offer only while payments are on)
	frontendURL     string
	now             func() time.Time

	mu              sync.Mutex
	couponRemaining int64
	couponFetched   time.Time
}

// PlanCheckoutConfig configures PlanCheckoutService.
type PlanCheckoutConfig struct {
	Store           planCheckoutStore
	Customers       stripeCustomerSource
	Prices          priceResolver
	API             stripeClient.PlanBillingAPI
	PaymentsEnabled bool
	FrontendURL     string
}

// NewPlanCheckoutService creates the service.
func NewPlanCheckoutService(cfg PlanCheckoutConfig) *PlanCheckoutService {
	return &PlanCheckoutService{
		store:           cfg.Store,
		customers:       cfg.Customers,
		prices:          cfg.Prices,
		api:             cfg.API,
		paymentsEnabled: cfg.PaymentsEnabled,
		frontendURL:     cfg.FrontendURL,
		now:             time.Now,
	}
}

// NewPlanCheckoutServiceFromStripe wires the service to StripeService's
// customers and catalog prices (nil api when Stripe isn't configured).
func NewPlanCheckoutServiceFromStripe(repo *repository.MembershipRepository, stripeSvc *StripeService, secretKey string, paymentsEnabled bool, frontendURL string) *PlanCheckoutService {
	cfg := PlanCheckoutConfig{Store: repo, PaymentsEnabled: paymentsEnabled, FrontendURL: frontendURL}
	if stripeSvc != nil && stripeSvc.IsConfigured() && secretKey != "" {
		cfg.Customers = stripeSvc
		cfg.Prices = stripeSvc.prices
		cfg.API = stripeClient.NewBillingClient(secretKey)
	}
	return NewPlanCheckoutService(cfg)
}

// Available reports whether plans can be bought on this server.
func (s *PlanCheckoutService) Available() bool {
	return s != nil && s.api != nil && s.customers != nil && s.prices != nil
}

// FoundingOfferAvailable reports whether the founding offer (Team at $59/mo)
// is open: invoice payments are on and the coupon has redemptions left,
// checked with Stripe and cached for a minute. Any error closes the offer.
func (s *PlanCheckoutService) FoundingOfferAvailable(ctx context.Context) bool {
	if !s.paymentsEnabled || !s.Available() {
		return false
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.couponFetched.IsZero() && s.now().Sub(s.couponFetched) < couponCacheTTL {
		return s.couponRemaining > 0
	}
	coupon, err := s.api.GetCoupon(ctx, stripeClient.FoundingCouponID)
	if err != nil {
		logging.Warn(ctx, "[PLAN-CHECKOUT] Founding coupon lookup failed; offer closed for now", map[string]any{"error": err})
		return false
	}
	s.couponRemaining, s.couponFetched = coupon.Remaining(), s.now()
	return s.couponRemaining > 0
}

// foundingOfferApplies: Team monthly while the offer is open.
func (s *PlanCheckoutService) foundingOfferApplies(ctx context.Context, tier string, cycle models.BillingCycle) bool {
	return tier == stripeClient.FoundingCouponTier && cycle == models.BillingCycleMonthly && s.FoundingOfferAvailable(ctx)
}

// StartCheckout creates a Checkout session for tenantID on the chosen plan
// and returns its URL. A trial with more than two days left carries over as
// the subscription's trial, so choosing early doesn't cut the trial short.
func (s *PlanCheckoutService) StartCheckout(ctx context.Context, tenantID uuid.UUID, req *models.PlanCheckoutRequest) (*models.PlanCheckoutResponse, error) {
	if !s.Available() {
		return nil, ErrPlanBillingUnavailable
	}
	tier, err := s.store.GetTierByID(ctx, req.TierID)
	if err != nil {
		if errors.Is(err, repository.ErrTierNotFound) {
			return nil, ErrPlanNotAvailable
		}
		return nil, fmt.Errorf("load tier: %w", err)
	}
	if _, ok := stripeClient.CatalogPlanFor(tier.Name); !ok {
		return nil, ErrPlanNotAvailable
	}

	current, err := s.store.GetTenantSubscription(ctx, tenantID)
	if err != nil && !errors.Is(err, repository.ErrSubscriptionNotFound) {
		return nil, fmt.Errorf("load subscription: %w", err)
	}
	if current != nil && current.StripeSubscriptionID != nil && *current.StripeSubscriptionID != "" {
		return nil, ErrAlreadySubscribed
	}

	priceID, err := s.prices.priceID(ctx, tier.Name, string(req.BillingCycle))
	if err != nil {
		if errors.Is(err, ErrCatalogPriceMissing) {
			logging.Error(ctx, "[PLAN-CHECKOUT] Catalog price missing in Stripe; run make stripe-catalog", map[string]any{"tier": tier.Name, "cycle": req.BillingCycle})
			return nil, ErrPlanBillingUnavailable.WithCause(err)
		}
		return nil, ErrPlanCheckoutFailed.WithCause(err)
	}
	customerID, err := s.customers.GetOrCreateStripeCustomer(ctx, tenantID)
	if err != nil {
		return nil, ErrPlanCheckoutFailed.WithCause(err)
	}

	founding := s.foundingOfferApplies(ctx, tier.Name, req.BillingCycle)
	in := stripeClient.SubscriptionCheckoutInput{
		CustomerID: customerID,
		PriceID:    priceID,
		SuccessURL: s.frontendURL + "/settings/organization?tab=membership&checkout=success",
		CancelURL:  s.frontendURL + "/settings/organization?tab=membership&checkout=cancelled",
		Metadata: map[string]string{
			stripeClient.MetaTenantID:     tenantID.String(),
			stripeClient.MetaTier:         tier.Name,
			stripeClient.MetaBillingCycle: string(req.BillingCycle),
		},
	}
	if founding {
		in.CouponID = stripeClient.FoundingCouponID
		in.Metadata[stripeClient.MetaFoundingOffer] = "true"
	}
	now := s.now()
	if current != nil && current.TrialEndsAt != nil && current.TrialEndsAt.Sub(now) > minCheckoutTrial {
		in.TrialEnd = current.TrialEndsAt.Unix()
	}
	// Same tenant, plan, discount and trial within the hour reuse one session.
	in.IdempotencyKey = fmt.Sprintf("plan-checkout:%s:%s:%s:%t:%d:%s",
		tenantID, tier.Name, req.BillingCycle, founding, in.TrialEnd, now.UTC().Format("2006010215"))

	url, err := s.api.CreateSubscriptionCheckout(ctx, in)
	if err != nil {
		logging.Error(ctx, "[PLAN-CHECKOUT] Checkout session failed", map[string]any{"tenant_id": tenantID, "tier": tier.Name, "error": err})
		return nil, ErrPlanCheckoutFailed.WithCause(err)
	}
	logging.Info(ctx, "[PLAN-CHECKOUT] Checkout session created", map[string]any{"tenant_id": tenantID, "tier": tier.Name, "cycle": req.BillingCycle, "founding_offer": founding})
	return &models.PlanCheckoutResponse{URL: url, FoundingOffer: founding}, nil
}

// checkoutActivationStore is what activating a Checkout subscription needs
// (faked in tests).
type checkoutActivationStore interface {
	TenantIDForStripeCustomer(ctx context.Context, stripeCustomerID string) (uuid.UUID, error)
	CatalogTier(ctx context.Context, name string) (*models.MembershipTier, error)
	ActivateStripeSubscription(ctx context.Context, tenantID uuid.UUID, a repository.StripeActivation) (bool, error)
}

// activateFromStripe makes an active Stripe subscription with no row of ours
// the tenant's plan. The tenant comes from our own customer -> tenant
// mapping; the plan from the price's catalog lookup key. Events that aren't
// ours (unknown customer, non-catalog price, metadata naming another tenant)
// are acknowledged and ignored. Repeats are no-ops (ActivateStripeSubscription
// is idempotent).
func activateFromStripe(ctx context.Context, store checkoutActivationStore, sub *stripe.Subscription, periodEnd *time.Time) error {
	if sub.Customer == nil || sub.Customer.ID == "" || sub.Items == nil || len(sub.Items.Data) == 0 ||
		sub.Items.Data[0] == nil || sub.Items.Data[0].Price == nil {
		logging.Warn(ctx, "[BILLING-WEBHOOK] Subscription without customer or price; ignored", map[string]any{"subscription_id": sub.ID})
		return nil
	}
	tenantID, err := store.TenantIDForStripeCustomer(ctx, sub.Customer.ID)
	if err != nil {
		if errors.Is(err, repository.ErrTenantNotFound) {
			logging.Info(ctx, "[BILLING-WEBHOOK] Subscription for an unknown customer; ignored", map[string]any{"subscription_id": sub.ID})
			return nil
		}
		return fmt.Errorf("look up tenant: %w", err)
	}
	if ref := sub.Metadata[stripeClient.MetaTenantID]; ref != "" && ref != tenantID.String() {
		logging.Warn(ctx, "[BILLING-WEBHOOK] Subscription metadata names another tenant; ignored", map[string]any{"subscription_id": sub.ID})
		return nil
	}
	price := sub.Items.Data[0].Price
	tierName, cycle, ok := stripeClient.TierCycleForLookupKey(price.LookupKey)
	if !ok {
		logging.Warn(ctx, "[BILLING-WEBHOOK] Subscription price is not in the plan catalog; ignored", map[string]any{"subscription_id": sub.ID})
		return nil
	}
	tier, err := store.CatalogTier(ctx, tierName)
	if err != nil {
		return fmt.Errorf("load tier %s: %w", tierName, err)
	}
	priceCents := int(price.UnitAmount)
	reason := "checkout"
	if sub.Metadata[stripeClient.MetaFoundingOffer] == "true" {
		priceCents -= stripeClient.FoundingCouponAmountOffCents
		reason = "checkout: founding offer"
	}
	created, err := store.ActivateStripeSubscription(ctx, tenantID, repository.StripeActivation{
		TierID:               tier.ID,
		BillingCycle:         models.BillingCycle(cycle),
		PriceCents:           priceCents,
		StripeSubscriptionID: sub.ID,
		StripePriceID:        price.ID,
		CurrentPeriodEnd:     periodEnd,
		CancelAtPeriodEnd:    sub.CancelAtPeriodEnd,
		Reason:               reason,
	})
	if err != nil {
		return fmt.Errorf("activate subscription: %w", err)
	}
	if created {
		logging.Info(ctx, "[BILLING-WEBHOOK] Plan activated from Checkout", map[string]any{"tenant_id": tenantID, "tier": tierName, "cycle": cycle})
	}
	return nil
}

// stripeActivationStore adapts StripeService's repositories.
type stripeActivationStore struct{ s *StripeService }

func (a stripeActivationStore) TenantIDForStripeCustomer(ctx context.Context, id string) (uuid.UUID, error) {
	return a.s.TenantIDForStripeCustomer(ctx, id)
}

func (a stripeActivationStore) CatalogTier(ctx context.Context, name string) (*models.MembershipTier, error) {
	return a.s.membershipRepo.GetTierByName(ctx, name)
}

func (a stripeActivationStore) ActivateStripeSubscription(ctx context.Context, tenantID uuid.UUID, act repository.StripeActivation) (bool, error) {
	return a.s.membershipRepo.ActivateStripeSubscription(ctx, tenantID, act)
}

func (s *StripeService) activateCheckoutSubscription(ctx context.Context, sub *stripe.Subscription, periodEnd *time.Time) error {
	return activateFromStripe(ctx, stripeActivationStore{s}, sub, periodEnd)
}
