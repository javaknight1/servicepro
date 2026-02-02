package services

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/stripe/stripe-go/v76"

	stripeClient "github.com/javaknight1/servicepro/backend/internal/services/stripe"
	"github.com/javaknight1/servicepro/backend/pkg/clients/logging"
)

// InvoicePaymentHandlerAdapter implements the stripe.InvoicePaymentHandler interface
// and bridges the webhook handler to the invoice service
type InvoicePaymentHandlerAdapter struct {
	invoiceService *InvoiceService
}

// Ensure InvoicePaymentHandlerAdapter implements stripe.InvoicePaymentHandler
var _ stripeClient.InvoicePaymentHandler = (*InvoicePaymentHandlerAdapter)(nil)

// NewInvoicePaymentHandlerAdapter creates a new adapter
func NewInvoicePaymentHandlerAdapter(invoiceService *InvoiceService) *InvoicePaymentHandlerAdapter {
	return &InvoicePaymentHandlerAdapter{
		invoiceService: invoiceService,
	}
}

// HandleInvoicePaymentCompleted handles the invoice payment completion from a Stripe webhook
func (a *InvoicePaymentHandlerAdapter) HandleInvoicePaymentCompleted(
	ctx context.Context,
	invoiceIDStr string,
	amountPaidCents int64,
	checkoutSessionID, paymentIntentID string,
) error {
	// Parse invoice ID
	invoiceID, err := uuid.Parse(invoiceIDStr)
	if err != nil {
		return fmt.Errorf("invalid invoice ID: %w", err)
	}

	// Convert cents to decimal
	amountPaid := stripeClient.ConvertCentsToDecimal(amountPaidCents)

	// Mark invoice as paid
	_, err = a.invoiceService.MarkInvoiceAsPaid(ctx, invoiceID, amountPaid, checkoutSessionID, paymentIntentID)
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

// CreateCheckoutCompletedHandler returns an EventHandler for checkout.session.completed events
func (a *InvoicePaymentHandlerAdapter) CreateCheckoutCompletedHandler() stripeClient.EventHandler {
	return func(ctx context.Context, event *stripe.Event) error {
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

		// Mark invoice as paid
		if err := a.HandleInvoicePaymentCompleted(ctx, invoiceIDStr, session.AmountTotal, session.ID, session.PaymentIntentID); err != nil {
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
