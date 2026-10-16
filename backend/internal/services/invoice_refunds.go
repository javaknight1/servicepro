package services

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"

	"github.com/javaknight1/servicepro/backend/internal/models"
)

var (
	// ErrRefundExceedsPaid is returned when a refund is larger than the amount paid
	ErrRefundExceedsPaid = errors.New("refund amount exceeds the amount paid")
	// ErrRefundNotAllowed is returned when the invoice has nothing refundable
	ErrRefundNotAllowed = errors.New("only invoices with payments can be refunded")
)

// RecordRefund records money returned to the customer outside Stripe (cash,
// check, a refund made in the Stripe dashboard before webhooks were wired)
// and lowers the invoice's amount_paid in the same transaction (T125).
func (s *InvoiceService) RecordRefund(ctx context.Context, refund *models.InvoiceRefund) (*models.InvoiceRefund, error) {
	if !refund.Amount.IsPositive() {
		return nil, invalidInvoice("refund amount must be greater than zero")
	}
	refund.Amount = refund.Amount.Round(2)
	if refund.RefundDate.IsZero() {
		refund.RefundDate = time.Now()
	}
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		invoice, err := lockInvoice(tx, refund.InvoiceID)
		if err != nil {
			return err
		}
		return insertRefund(tx, invoice, refund)
	})
	if err != nil {
		return nil, err
	}
	return refund, nil
}

// ApplyStripeRefunds records a Stripe charge's refunds against the invoice
// paid by paymentIntentID. Stripe reports the charge's cumulative refunded
// amount, so the refund recorded is the difference from what is already on
// file for that payment intent — computed under the invoice row lock, which
// makes duplicate and out-of-order charge.refunded deliveries harmless.
func (s *InvoiceService) ApplyStripeRefunds(ctx context.Context, paymentIntentID, chargeID string, totalRefunded decimal.Decimal) error {
	if paymentIntentID == "" || !totalRefunded.IsPositive() {
		return nil
	}
	db := s.db.WithContext(ctx)

	// invoice_payments has no tenant_id; the payment intent identifies the
	// invoice (system context: this runs from a signed webhook).
	var payment models.InvoicePayment
	if err := db.Where("stripe_payment_intent_id = ?", paymentIntentID).First(&payment).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil // not one of our invoice payments
		}
		return err
	}

	return db.Transaction(func(tx *gorm.DB) error {
		invoice, err := lockInvoice(tx, payment.InvoiceID)
		if err != nil {
			return err
		}
		var already decimal.Decimal
		var rows []models.InvoiceRefund
		if err := tx.Where("invoice_id = ? AND stripe_payment_intent_id = ?", invoice.ID, paymentIntentID).Find(&rows).Error; err != nil {
			return err
		}
		for _, r := range rows {
			already = already.Add(r.Amount)
		}
		delta := totalRefunded.Sub(already)
		if !delta.IsPositive() {
			return nil // already recorded
		}
		return insertRefund(tx, invoice, &models.InvoiceRefund{
			InvoiceID:             invoice.ID,
			Amount:                delta,
			RefundDate:            time.Now(),
			Method:                "stripe",
			Reason:                "Refunded in Stripe",
			StripePaymentIntentID: nonEmpty(paymentIntentID),
			StripeChargeID:        nonEmpty(chargeID),
		})
	})
}

// insertRefund writes a refund for a locked invoice and moves amount_paid
// and status: back to partially_paid while money remains, refunded at zero.
// A paid job goes back to invoiced. Cancelled invoices keep their status.
func insertRefund(tx *gorm.DB, invoice *models.Invoice, refund *models.InvoiceRefund) error {
	if !invoice.AmountPaid.IsPositive() {
		return ErrRefundNotAllowed
	}
	if refund.Amount.GreaterThan(invoice.AmountPaid) {
		return ErrRefundExceedsPaid
	}
	if refund.ID == uuid.Nil {
		refund.ID = uuid.New()
	}
	refund.InvoiceID = invoice.ID
	if err := tx.Create(refund).Error; err != nil {
		return fmt.Errorf("failed to record refund: %w", err)
	}

	newPaid := invoice.AmountPaid.Sub(refund.Amount)
	updates := map[string]interface{}{"amount_paid": newPaid}
	to := invoice.Status
	if invoice.Status != models.InvoiceStatusCancelled {
		if newPaid.IsZero() {
			to = models.InvoiceStatusRefunded
		} else {
			to = models.InvoiceStatusPartiallyPaid
		}
		if !invoice.Status.CanTransitionTo(to) {
			return ErrInvalidInvoiceTransition
		}
		updates["status"] = to
		if to != models.InvoiceStatusPaid {
			updates["paid_date"] = nil
		}
	}
	if err := tx.Model(&models.Invoice{}).Where("id = ?", invoice.ID).Updates(updates).Error; err != nil {
		return fmt.Errorf("failed to update invoice balance: %w", err)
	}
	if err := recordInvoiceTransition(tx, invoice.ID, invoice.Status, to, "refund", refund.CreatedBy); err != nil {
		return err
	}
	if invoice.Status == models.InvoiceStatusPaid {
		return moveInvoiceJob(tx, invoice, models.JobStatusPaid, models.JobStatusInvoiced,
			fmt.Sprintf("Invoice %s refunded", invoice.InvoiceNumber), invoice.CreatedBy)
	}
	return nil
}
