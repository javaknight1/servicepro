package handlers

import (
	"github.com/gin-gonic/gin"

	stripeService "github.com/javaknight1/servicepro/backend/internal/services/stripe"
)

// StripeHandler handles Stripe webhook deliveries.
//
// The former raw passthrough endpoints (/stripe/payment-intents/*,
// /stripe/customers/*, /stripe/refunds, /stripe/webhooks/stats) were removed
// (T105 / audit F-4): they acted on any Stripe object ID supplied by the
// caller on the platform account, letting a tenant refund or read other
// tenants' payments. Nothing in the frontend used them. Invoice payments go
// through the token-based Checkout flow (PublicInvoiceHandler) and tenant
// billing through BillingHandler, which only act on IDs stored for the
// caller's tenant.
type StripeHandler struct {
	webhookHandler *stripeService.WebhookHandler
}

// NewStripeHandler creates a new Stripe handler
func NewStripeHandler(webhookHandler *stripeService.WebhookHandler) *StripeHandler {
	return &StripeHandler{webhookHandler: webhookHandler}
}

// HandleWebhook handles incoming Stripe webhooks (signature-verified)
func (h *StripeHandler) HandleWebhook(c *gin.Context) {
	h.webhookHandler.HandleHTTPWebhook(c.Writer, c.Request)
}
