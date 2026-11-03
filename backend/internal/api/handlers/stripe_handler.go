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
	connectHandler *stripeService.WebhookHandler
	v2Handler      *stripeService.ThinEventHandler
}

// SetConnectWebhooks installs the Connect endpoints (T169); nil leaves one off.
func (h *StripeHandler) SetConnectWebhooks(connect *stripeService.WebhookHandler, v2 *stripeService.ThinEventHandler) {
	h.connectHandler = connect
	h.v2Handler = v2
}

// HasConnectWebhook reports whether the Connect endpoint is configured.
func (h *StripeHandler) HasConnectWebhook() bool { return h.connectHandler != nil }

// HasV2Webhook reports whether the v2 event destination is configured.
func (h *StripeHandler) HasV2Webhook() bool { return h.v2Handler != nil }

// HandleConnectWebhook handles events from connected accounts (direct-charge
// payments, refunds, account.updated), verified with the Connect secret.
func (h *StripeHandler) HandleConnectWebhook(c *gin.Context) {
	h.connectHandler.HandleHTTPWebhook(c.Writer, c.Request)
}

// HandleV2Webhook handles thin v2 events (v2.core.account[...]).
func (h *StripeHandler) HandleV2Webhook(c *gin.Context) {
	h.v2Handler.HandleHTTPWebhook(c.Writer, c.Request)
}

// NewStripeHandler creates a new Stripe handler
func NewStripeHandler(webhookHandler *stripeService.WebhookHandler) *StripeHandler {
	return &StripeHandler{webhookHandler: webhookHandler}
}

// HandleWebhook handles incoming Stripe webhooks (signature-verified)
func (h *StripeHandler) HandleWebhook(c *gin.Context) {
	h.webhookHandler.HandleHTTPWebhook(c.Writer, c.Request)
}
