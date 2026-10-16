package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/stripe/stripe-go/v76"

	"github.com/javaknight1/servicepro/backend/internal/models"
	"github.com/javaknight1/servicepro/backend/internal/repository"
	stripeClient "github.com/javaknight1/servicepro/backend/internal/services/stripe"
	"github.com/javaknight1/servicepro/backend/internal/tenancy"
	"github.com/javaknight1/servicepro/backend/pkg/clients/logging"
)

// billingWebhookStore is what the platform-billing webhook handlers need from
// StripeService (an interface so the handlers can be tested without Stripe or
// a database).
type billingWebhookStore interface {
	HandleSubscriptionUpdated(ctx context.Context, sub *stripe.Subscription) error
	HandleSubscriptionDeleted(ctx context.Context, sub *stripe.Subscription) error
	TenantIDForStripeCustomer(ctx context.Context, stripeCustomerID string) (uuid.UUID, error)
	RecordBillingEvent(ctx context.Context, tenantID uuid.UUID, event *models.BillingEvent) error
}

// TenantIDForStripeCustomer resolves the tenant that owns a Stripe customer
// (platform billing). Returns repository.ErrTenantNotFound when no tenant has
// that customer ID.
func (s *StripeService) TenantIDForStripeCustomer(ctx context.Context, stripeCustomerID string) (uuid.UUID, error) {
	tenant, err := s.tenantRepo.GetByStripeCustomerID(ctx, stripeCustomerID)
	if err != nil {
		return uuid.Nil, err
	}
	return tenant.ID, nil
}

// RegisterBillingWebhookHandlers registers ServicePro's own subscription
// billing events (customer.subscription.*, invoice.paid,
// invoice.payment_failed) on the single signature-verified, deduplicated
// webhook endpoint POST /api/v1/stripe/webhooks (T122). They replace the
// legacy /api/v1/webhooks/stripe handler, which panicked on every one of these
// events.
//
// A returned error makes the endpoint answer 503 so Stripe retries; events we
// cannot act on (unknown customer) are acknowledged.
func RegisterBillingWebhookHandlers(p *stripeClient.EventProcessor, store billingWebhookStore) {
	subHandler := func(ctx context.Context, event *stripe.Event) error {
		ctx = tenancy.WithSystem(ctx)
		var sub stripe.Subscription
		if err := json.Unmarshal(event.Data.Raw, &sub); err != nil {
			return fmt.Errorf("parse subscription: %w", err)
		}
		if string(event.Type) == stripeClient.EventSubscriptionDeleted {
			return store.HandleSubscriptionDeleted(ctx, &sub)
		}
		return store.HandleSubscriptionUpdated(ctx, &sub)
	}
	p.RegisterHandler(stripeClient.EventSubscriptionCreated, subHandler)
	p.RegisterHandler(stripeClient.EventSubscriptionUpdated, subHandler)
	p.RegisterHandler(stripeClient.EventSubscriptionDeleted, subHandler)

	invoiceHandler := func(status models.BillingEventStatus, description string, amount func(*stripe.Invoice) int64) stripeClient.EventHandler {
		return func(ctx context.Context, event *stripe.Event) error {
			ctx = tenancy.WithSystem(ctx)
			var inv stripe.Invoice
			if err := json.Unmarshal(event.Data.Raw, &inv); err != nil {
				return fmt.Errorf("parse invoice: %w", err)
			}
			if inv.Customer == nil || inv.Customer.ID == "" {
				logging.Warn(ctx, "[BILLING-WEBHOOK] Invoice event without customer, ignoring", map[string]any{"event_id": event.ID})
				return nil
			}
			tenantID, err := store.TenantIDForStripeCustomer(ctx, inv.Customer.ID)
			if err != nil {
				if errors.Is(err, repository.ErrTenantNotFound) {
					// Not one of our tenants' platform subscriptions; retrying won't help.
					logging.Info(ctx, "[BILLING-WEBHOOK] No tenant for Stripe customer, ignoring", map[string]any{"event_id": event.ID})
					return nil
				}
				return fmt.Errorf("look up tenant: %w", err)
			}
			amt := int(amount(&inv))
			be := &models.BillingEvent{
				ID:               uuid.New(),
				TenantID:         tenantID,
				StripeEventID:    event.ID,
				EventType:        string(event.Type),
				AmountCents:      &amt,
				Currency:         string(inv.Currency),
				Status:           status,
				Description:      description,
				StripeInvoiceID:  strPtrOrNil(inv.ID),
				StripeInvoiceURL: strPtrOrNil(inv.HostedInvoiceURL),
				CreatedAt:        time.Now(),
			}
			return store.RecordBillingEvent(ctx, tenantID, be)
		}
	}
	p.RegisterHandler(stripeClient.EventInvoicePaid,
		invoiceHandler(models.BillingEventStatusSucceeded, "Invoice paid", func(i *stripe.Invoice) int64 { return i.AmountPaid }))
	p.RegisterHandler(stripeClient.EventInvoicePaymentFailed,
		invoiceHandler(models.BillingEventStatusFailed, "Invoice payment failed", func(i *stripe.Invoice) int64 { return i.AmountDue }))
}

func strPtrOrNil(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
