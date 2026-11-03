package stripe

import (
	"context"

	"github.com/stripe/stripe-go/v87"
)

// Metadata keys ServicePro puts on plan Checkout sessions and the
// subscriptions they create (T210).
const (
	MetaTenantID      = "tenant_id"
	MetaTier          = "tier"
	MetaBillingCycle  = "billing_cycle"
	MetaFoundingOffer = "founding_offer" // "true" when the founding coupon was applied
)

// SubscriptionCheckoutInput is a Checkout session for a platform plan.
type SubscriptionCheckoutInput struct {
	CustomerID     string
	PriceID        string
	CouponID       string // "" = no discount
	TrialEnd       int64  // unix seconds; 0 = charge now
	SuccessURL     string
	CancelURL      string
	Metadata       map[string]string // copied to the session and the subscription
	IdempotencyKey string
}

// PlanBillingAPI is the platform-billing surface plan checkout uses (faked
// in tests).
type PlanBillingAPI interface {
	// GetCoupon returns the coupon, or nil when it doesn't exist.
	GetCoupon(ctx context.Context, id string) (*CatalogCoupon, error)
	// CreateSubscriptionCheckout returns the hosted Checkout URL.
	CreateSubscriptionCheckout(ctx context.Context, in SubscriptionCheckoutInput) (string, error)
}

var _ PlanBillingAPI = (*BillingClient)(nil)

// CreateSubscriptionCheckout creates a subscription-mode Checkout session on
// the platform account and returns its URL. The subscription's state reaches
// ServicePro only through the billing webhook.
func (c *BillingClient) CreateSubscriptionCheckout(ctx context.Context, in SubscriptionCheckoutInput) (string, error) {
	params := &stripe.CheckoutSessionCreateParams{
		Mode:       stripe.String(string(stripe.CheckoutSessionModeSubscription)),
		Customer:   stripe.String(in.CustomerID),
		SuccessURL: stripe.String(in.SuccessURL),
		CancelURL:  stripe.String(in.CancelURL),
		LineItems: []*stripe.CheckoutSessionCreateLineItemParams{{
			Price:    stripe.String(in.PriceID),
			Quantity: stripe.Int64(1),
		}},
		Metadata: in.Metadata,
		SubscriptionData: &stripe.CheckoutSessionCreateSubscriptionDataParams{
			Metadata: in.Metadata,
		},
	}
	if ref := in.Metadata[MetaTenantID]; ref != "" {
		params.ClientReferenceID = stripe.String(ref)
	}
	if in.CouponID != "" {
		params.Discounts = []*stripe.CheckoutSessionCreateDiscountParams{{Coupon: stripe.String(in.CouponID)}}
	}
	if in.TrialEnd > 0 {
		params.SubscriptionData.TrialEnd = stripe.Int64(in.TrialEnd)
	}
	params.SetIdempotencyKey(in.IdempotencyKey)
	sess, err := c.api.V1CheckoutSessions.Create(ctx, params)
	if err != nil {
		return "", err
	}
	return sess.URL, nil
}
