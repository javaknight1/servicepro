package services

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/javaknight1/servicepro/backend/internal/models"
	"github.com/javaknight1/servicepro/backend/pkg/clients/logging"
)

// MarkPaymentPending records that a bank (ACH) Checkout Session for the
// invoice completed and its money is settling (T170): reminders skip the
// invoice until the payment succeeds or fails. It is a no-op — so webhook
// replays and out-of-order deliveries are safe — when the session was
// already credited, already marked, or the invoice no longer takes payments.
// Runs under the invoice row lock; the tenancy plugin scopes the lookup.
func (s *InvoiceService) MarkPaymentPending(ctx context.Context, id uuid.UUID, checkoutSessionID string) error {
	if checkoutSessionID == "" {
		return fmt.Errorf("checkout session ID is required")
	}
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		invoice, err := lockInvoice(tx, id)
		if err != nil {
			return err
		}
		if invoice.PaymentPendingSessionID != nil && *invoice.PaymentPendingSessionID == checkoutSessionID {
			return nil // replay
		}
		if !invoice.Status.AcceptsPayments() || !invoice.BalanceDue().IsPositive() {
			return nil
		}
		credited, err := sessionCredited(tx, id, checkoutSessionID)
		if err != nil || credited {
			return err // succeeded already: a late "completed" must not re-pend
		}
		if err := tx.Model(&models.Invoice{}).Where("id = ?", id).Updates(map[string]any{
			"payment_pending_at":         time.Now(),
			"payment_pending_session_id": checkoutSessionID,
		}).Error; err != nil {
			return fmt.Errorf("failed to mark payment pending: %w", err)
		}
		logging.Info(ctx, "[INVOICE-SERVICE] Bank payment pending", map[string]any{"invoice_id": id})
		return nil
	})
}

// ClearPaymentPending ends the pending state set by checkoutSessionID (its
// bank payment failed): the invoice keeps the status it had all along and
// reminders resume. Pending set by another session is left alone.
func (s *InvoiceService) ClearPaymentPending(ctx context.Context, id uuid.UUID, checkoutSessionID string) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		invoice, err := lockInvoice(tx, id)
		if err != nil {
			return err
		}
		if !pendingFrom(invoice, checkoutSessionID) {
			return nil
		}
		if err := tx.Model(&models.Invoice{}).Where("id = ?", id).Updates(clearPendingUpdates()).Error; err != nil {
			return fmt.Errorf("failed to clear payment pending: %w", err)
		}
		logging.Warn(ctx, "[INVOICE-SERVICE] Bank payment failed; invoice open again", map[string]any{"invoice_id": id})
		return nil
	})
}

// pendingFrom reports whether the invoice's pending state was set by
// checkoutSessionID.
func pendingFrom(invoice *models.Invoice, checkoutSessionID string) bool {
	return checkoutSessionID != "" && invoice.PaymentPendingSessionID != nil &&
		*invoice.PaymentPendingSessionID == checkoutSessionID
}

// clearPendingUpdates are the column updates that end the pending state.
func clearPendingUpdates() map[string]any {
	return map[string]any{"payment_pending_at": nil, "payment_pending_session_id": nil}
}

// sessionCredited reports whether a Checkout Session already has a payment
// row on the invoice.
func sessionCredited(tx *gorm.DB, invoiceID uuid.UUID, checkoutSessionID string) (bool, error) {
	var n int64
	if err := tx.Model(&models.InvoicePayment{}).
		Where("invoice_id = ? AND stripe_checkout_session_id = ?", invoiceID, checkoutSessionID).
		Count(&n).Error; err != nil {
		return false, err
	}
	return n > 0, nil
}
