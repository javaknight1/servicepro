package services

import (
	"context"
	"fmt"

	"github.com/stripe/stripe-go/v76"

	stripeClient "github.com/javaknight1/servicepro/backend/internal/services/stripe"
	"github.com/javaknight1/servicepro/backend/internal/tenancy"
	"github.com/javaknight1/servicepro/backend/pkg/clients/logging"
)

// CreateChargeRefundedHandler returns the charge.refunded handler for
// invoice payments (T125). It runs inside the signature-verified,
// deduplicated webhook processor; ApplyStripeRefunds is idempotent on top of
// that (it records the charge's cumulative refunded amount once).
func (a *InvoicePaymentHandlerAdapter) CreateChargeRefundedHandler() stripeClient.EventHandler {
	return func(ctx context.Context, event *stripe.Event) error {
		// Signed Stripe event: no tenant session; the payment intent selects
		// our invoice payment.
		ctx = tenancy.WithSystem(ctx)
		ch, err := stripeClient.ParseCharge(event)
		if err != nil {
			return fmt.Errorf("failed to parse charge: %w", err)
		}
		if ch.PaymentIntentID == "" || ch.AmountRefunded <= 0 {
			return nil
		}
		total := stripeClient.FromMinorUnits(ch.AmountRefunded, ch.Currency)
		if err := a.invoiceService.ApplyStripeRefunds(ctx, ch.PaymentIntentID, ch.ID, total); err != nil {
			return fmt.Errorf("failed to apply refund: %w", err)
		}
		logging.Info(ctx, "[INVOICE-PAYMENT-HANDLER] Charge refund applied", map[string]any{"charge_id": ch.ID, "refunded_total": total.StringFixed(2)})
		return nil
	}
}
