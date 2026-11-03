package services

import (
	"context"
	"errors"
	"fmt"

	"github.com/javaknight1/servicepro/backend/internal/tenancy"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stripe/stripe-go/v87"

	stripeClient "github.com/javaknight1/servicepro/backend/internal/services/stripe"
	"github.com/javaknight1/servicepro/backend/pkg/clients/logging"
)

// InvoicePaymentHandlerAdapter bridges checkout.session.completed webhook
// events to the invoice service
type InvoicePaymentHandlerAdapter struct {
	invoiceService *InvoiceService
	accounts       StripeAccountTenants
}

// StripeAccountTenants resolves a connected account (event.account) to the
// tenant that owns it (T169).
type StripeAccountTenants interface {
	TenantForStripeAccount(ctx context.Context, stripeAccountID string) (uuid.UUID, bool, error)
}

// SetConnectedAccounts enables connected-account events: without it, events
// from connected accounts are ignored.
func (a *InvoicePaymentHandlerAdapter) SetConnectedAccounts(accounts StripeAccountTenants) {
	a.accounts = accounts
}

// errSkipEvent marks a verified event that is not for us (or not for this
// invoice): acknowledged, never retried, nothing recorded.
var errSkipEvent = errors.New("event skipped")

// scopeToEventAccount returns the context invoice writes for this event run
// in. Events from a connected account (direct charges, T169) run as the
// tenant that owns the account, so an invoice of any other tenant is
// invisible; platform-account events (pre-Connect sessions) stay system-wide.
func (a *InvoicePaymentHandlerAdapter) scopeToEventAccount(ctx context.Context, event *stripe.Event) (context.Context, error) {
	ctx = tenancy.WithSystem(ctx)
	if event.Account == "" {
		return ctx, nil
	}
	if a.accounts == nil {
		logging.Warn(ctx, "[INVOICE-PAYMENT-HANDLER] Connected-account event ignored: Connect not wired", map[string]any{"event_id": event.ID})
		return ctx, errSkipEvent
	}
	tenantID, ok, err := a.accounts.TenantForStripeAccount(ctx, event.Account)
	if err != nil {
		return ctx, err
	}
	if !ok {
		logging.Warn(ctx, "[INVOICE-PAYMENT-HANDLER] Event from an unknown connected account ignored", map[string]any{"event_id": event.ID, "account_id": event.Account})
		return ctx, errSkipEvent
	}
	return tenancy.WithTenant(ctx, tenantID), nil
}

// NewInvoicePaymentHandlerAdapter creates a new adapter
func NewInvoicePaymentHandlerAdapter(invoiceService *InvoiceService) *InvoicePaymentHandlerAdapter {
	return &InvoicePaymentHandlerAdapter{
		invoiceService: invoiceService,
	}
}

// applyCheckoutPayment records a completed Checkout payment against the
// invoice (idempotent per session; see InvoiceService.MarkInvoiceAsPaid).
func (a *InvoicePaymentHandlerAdapter) applyCheckoutPayment(ctx context.Context, invoiceID uuid.UUID, amountPaid decimal.Decimal, checkoutSessionID, paymentIntentID string) error {
	// Mark invoice as paid
	_, err := a.invoiceService.MarkInvoiceAsPaid(ctx, invoiceID, amountPaid, checkoutSessionID, paymentIntentID)
	if err != nil {
		return fmt.Errorf("failed to mark invoice as paid: %w", err)
	}

	logging.Info(ctx, "[INVOICE-PAYMENT-HANDLER] Invoice marked as paid", map[string]any{"invoice_id": invoiceID, "amount": amountPaid.StringFixed(2), "session_id": checkoutSessionID})

	return nil
}

// SendInvoiceReceiptEmail sends a payment receipt email for the invoice
func (a *InvoicePaymentHandlerAdapter) SendInvoiceReceiptEmail(ctx context.Context, invoiceIDStr string) error {
	// Parse invoice ID
	invoiceID, err := uuid.Parse(invoiceIDStr)
	if err != nil {
		return fmt.Errorf("invalid invoice ID: %w", err)
	}

	// Get the invoice
	invoice, err := a.invoiceService.GetInvoice(ctx, invoiceID)
	if err != nil {
		return fmt.Errorf("failed to get invoice: %w", err)
	}

	// Send receipt email
	a.invoiceService.SendReceiptEmail(invoice)

	return nil
}

// CreateCheckoutCompletedHandler returns an EventHandler for
// checkout.session.completed and checkout.session.async_payment_succeeded
// (ACH settles days after the session completes unpaid). Recording is
// idempotent per Checkout Session, so the pair can never credit twice.
func (a *InvoicePaymentHandlerAdapter) CreateCheckoutCompletedHandler() stripeClient.EventHandler {
	return func(ctx context.Context, event *stripe.Event) error {
		// Signed Stripe event: no tenant session. The invoice is identified by
		// our own invoice_id metadata set when we created the session, within
		// the tenant that owns the connected account the event came from.
		ctx, err := a.scopeToEventAccount(ctx, event)
		if errors.Is(err, errSkipEvent) {
			return nil
		}
		if err != nil {
			return err
		}
		// Parse the checkout session
		session, err := stripeClient.ParseCheckoutSession(event)
		if err != nil {
			return fmt.Errorf("failed to parse checkout session: %w", err)
		}

		// Check if this is an invoice payment (via metadata)
		invoiceIDStr, ok := session.Metadata["invoice_id"]
		if !ok {
			// Not an invoice payment, ignore
			logging.Info(ctx, "[INVOICE-PAYMENT-HANDLER] Checkout session has no invoice_id metadata, skipping", map[string]any{"session_id": session.ID})
			return nil
		}

		paymentType, _ := session.Metadata["payment_type"]
		if paymentType != "invoice" {
			// Not an invoice payment type, ignore
			logging.Info(ctx, "[INVOICE-PAYMENT-HANDLER] Checkout session is not an invoice payment, skipping", map[string]any{"session_id": session.ID})
			return nil
		}

		// Verify payment was successful
		if session.PaymentStatus != "paid" {
			logging.Info(ctx, "[INVOICE-PAYMENT-HANDLER] Checkout session payment status is not paid", map[string]any{"session_id": session.ID, "payment_status": session.PaymentStatus})
			return nil
		}

		logging.Info(ctx, "[INVOICE-PAYMENT-HANDLER] Processing invoice payment", map[string]any{"invoice_id": invoiceIDStr, "session_id": session.ID, "amount": session.AmountTotal})

		invoiceID, err := uuid.Parse(invoiceIDStr)
		if err != nil {
			return fmt.Errorf("invalid invoice ID: %w", err)
		}
		if event.Account != "" {
			// The invoice must belong to the account's tenant (scoped read).
			if _, err := a.invoiceService.GetInvoice(ctx, invoiceID); err != nil {
				if errors.Is(err, ErrInvoiceNotFound) {
					logging.Error(ctx, "[INVOICE-PAYMENT-HANDLER] Connected-account payment for an invoice of another tenant ignored", map[string]any{"event_id": event.ID, "account_id": event.Account, "invoice_id": invoiceID})
					return nil
				}
				return fmt.Errorf("failed to load invoice: %w", err)
			}
		}
		// Stripe reports the amount in the session currency's minor units.
		amount := stripeClient.FromMinorUnits(session.AmountTotal, session.Currency)
		if err := a.applyCheckoutPayment(ctx, invoiceID, amount, session.ID, session.PaymentIntentID); err != nil {
			return fmt.Errorf("failed to handle invoice payment: %w", err)
		}

		// Send receipt email
		if err := a.SendInvoiceReceiptEmail(ctx, invoiceIDStr); err != nil {
			// Log but don't fail - payment was already processed
			logging.Error(ctx, "[INVOICE-PAYMENT-HANDLER] Failed to send receipt email for invoice", map[string]any{"invoice_id": invoiceIDStr, "error": err})
		}

		return nil
	}
}
