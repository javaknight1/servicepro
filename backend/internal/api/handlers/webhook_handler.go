package handlers

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stripe/stripe-go/v76"
	"github.com/stripe/stripe-go/v76/webhook"

	"github.com/javaknight1/servicepro/backend/internal/models"
	"github.com/javaknight1/servicepro/backend/internal/repository"
	"github.com/javaknight1/servicepro/backend/internal/services"
)

// WebhookHandler handles Stripe webhook events
type WebhookHandler struct {
	stripeService *services.StripeService
	webhookSecret string
	tenantRepo    *repository.TenantRepository
}

// NewWebhookHandler creates a new webhook handler
func NewWebhookHandler(
	stripeService *services.StripeService,
	webhookSecret string,
	tenantRepo *repository.TenantRepository,
) *WebhookHandler {
	return &WebhookHandler{
		stripeService: stripeService,
		webhookSecret: webhookSecret,
		tenantRepo:    tenantRepo,
	}
}

// HandleStripeWebhook processes incoming Stripe webhook events
// POST /api/v1/webhooks/stripe
func (h *WebhookHandler) HandleStripeWebhook(c *gin.Context) {
	const MaxBodyBytes = int64(65536)
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, MaxBodyBytes)

	payload, err := io.ReadAll(c.Request.Body)
	if err != nil {
		log.Printf("[WEBHOOK] Error reading request body: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	// Verify webhook signature
	event, err := webhook.ConstructEvent(payload, c.GetHeader("Stripe-Signature"), h.webhookSecret)
	if err != nil {
		log.Printf("[WEBHOOK] Error verifying webhook signature: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid signature"})
		return
	}

	log.Printf("[WEBHOOK] Received event: %s (ID: %s)", event.Type, event.ID)

	ctx := c.Request.Context()

	// Handle different event types
	switch event.Type {
	case "customer.subscription.created",
		"customer.subscription.updated",
		"customer.subscription.deleted":
		h.handleSubscriptionEvent(ctx, &event)

	case "invoice.paid":
		h.handleInvoicePaid(ctx, &event)

	case "invoice.payment_failed":
		h.handleInvoicePaymentFailed(ctx, &event)

	case "payment_method.attached":
		h.handlePaymentMethodAttached(ctx, &event)

	case "payment_method.detached":
		h.handlePaymentMethodDetached(ctx, &event)

	default:
		log.Printf("[WEBHOOK] Unhandled event type: %s", event.Type)
	}

	// Always return 200 to acknowledge receipt
	c.JSON(http.StatusOK, gin.H{"received": true})
}

func (h *WebhookHandler) handleSubscriptionEvent(ctx interface{}, event *stripe.Event) {
	var sub stripe.Subscription
	if err := json.Unmarshal(event.Data.Raw, &sub); err != nil {
		log.Printf("[WEBHOOK] Error parsing subscription: %v", err)
		return
	}

	// Use background context for webhook processing
	bgCtx := ctx.(interface{ Request() *http.Request }).Request().Context()

	switch event.Type {
	case "customer.subscription.created", "customer.subscription.updated":
		if err := h.stripeService.HandleSubscriptionUpdated(bgCtx, &sub); err != nil {
			log.Printf("[WEBHOOK] Error handling subscription updated: %v", err)
		}
	case "customer.subscription.deleted":
		if err := h.stripeService.HandleSubscriptionDeleted(bgCtx, &sub); err != nil {
			log.Printf("[WEBHOOK] Error handling subscription deleted: %v", err)
		}
	}
}

func (h *WebhookHandler) handleInvoicePaid(ctx interface{}, event *stripe.Event) {
	var invoice stripe.Invoice
	if err := json.Unmarshal(event.Data.Raw, &invoice); err != nil {
		log.Printf("[WEBHOOK] Error parsing invoice: %v", err)
		return
	}

	// Get tenant from Stripe customer
	bgCtx := ctx.(interface{ Request() *http.Request }).Request().Context()
	tenant, err := h.tenantRepo.GetByStripeCustomerID(bgCtx, invoice.Customer.ID)
	if err != nil {
		log.Printf("[WEBHOOK] Could not find tenant for customer %s: %v", invoice.Customer.ID, err)
		return
	}

	// Record billing event
	billingEvent := &models.BillingEvent{
		ID:               uuid.New(),
		TenantID:         tenant.ID,
		StripeEventID:    event.ID,
		EventType:        "invoice.paid",
		AmountCents:      intPtr(int(invoice.AmountPaid)),
		Currency:         string(invoice.Currency),
		Status:           models.BillingEventStatusSucceeded,
		Description:      "Invoice paid",
		StripeInvoiceID:  strPtr(invoice.ID),
		StripeInvoiceURL: strPtr(invoice.HostedInvoiceURL),
		CreatedAt:        time.Now(),
	}

	if err := h.stripeService.RecordBillingEvent(bgCtx, tenant.ID, billingEvent); err != nil {
		log.Printf("[WEBHOOK] Error recording billing event: %v", err)
	}
}

func (h *WebhookHandler) handleInvoicePaymentFailed(ctx interface{}, event *stripe.Event) {
	var invoice stripe.Invoice
	if err := json.Unmarshal(event.Data.Raw, &invoice); err != nil {
		log.Printf("[WEBHOOK] Error parsing invoice: %v", err)
		return
	}

	// Get tenant from Stripe customer
	bgCtx := ctx.(interface{ Request() *http.Request }).Request().Context()
	tenant, err := h.tenantRepo.GetByStripeCustomerID(bgCtx, invoice.Customer.ID)
	if err != nil {
		log.Printf("[WEBHOOK] Could not find tenant for customer %s: %v", invoice.Customer.ID, err)
		return
	}

	// Record billing event
	billingEvent := &models.BillingEvent{
		ID:               uuid.New(),
		TenantID:         tenant.ID,
		StripeEventID:    event.ID,
		EventType:        "invoice.payment_failed",
		AmountCents:      intPtr(int(invoice.AmountDue)),
		Currency:         string(invoice.Currency),
		Status:           models.BillingEventStatusFailed,
		Description:      "Invoice payment failed",
		StripeInvoiceID:  strPtr(invoice.ID),
		StripeInvoiceURL: strPtr(invoice.HostedInvoiceURL),
		CreatedAt:        time.Now(),
	}

	if err := h.stripeService.RecordBillingEvent(bgCtx, tenant.ID, billingEvent); err != nil {
		log.Printf("[WEBHOOK] Error recording billing event: %v", err)
	}
}

func (h *WebhookHandler) handlePaymentMethodAttached(ctx interface{}, event *stripe.Event) {
	log.Printf("[WEBHOOK] Payment method attached: %s", event.ID)
	// Payment method attachment is typically handled via the API, not webhooks
	// This is here for audit/logging purposes
}

func (h *WebhookHandler) handlePaymentMethodDetached(ctx interface{}, event *stripe.Event) {
	log.Printf("[WEBHOOK] Payment method detached: %s", event.ID)
	// Payment method detachment is typically handled via the API, not webhooks
	// This is here for audit/logging purposes
}

// Helper functions
func strPtr(s string) *string {
	return &s
}

func intPtr(i int) *int {
	return &i
}
