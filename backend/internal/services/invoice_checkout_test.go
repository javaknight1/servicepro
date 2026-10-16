package services

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/javaknight1/servicepro/backend/internal/models"
)

// T119 / audit 7.C2: Checkout charges exactly total_amount - amount_paid as
// one line item — not per-line amounts with truncated quantities, ignored
// discounts and ignored earlier payments.
func TestBuildCheckoutParams_ChargesBalanceDue(t *testing.T) {
	svc := &InvoicePaymentService{successURL: "https://app.test/ok", cancelURL: "https://app.test/cancel"}
	lines := []models.InvoiceLine{
		{Description: "Labour", Quantity: dec("1.5"), UnitPrice: dec("80"), Taxable: true}, // 120 (1.5 truncated to 1 before)
		{Description: "Parts", Quantity: dec("1"), UnitPrice: dec("50"), DiscountPercentage: dec("10"), Taxable: true},
	}
	totals, err := calculateInvoiceTotals(lines, dec("15"), dec("0.0825"))
	require.NoError(t, err)
	inv := &models.Invoice{ID: uuid.New(), InvoiceNumber: "INV-9", Lines: lines, AmountPaid: dec("40.10")}
	setInvoiceTotals(inv, totals)

	now := time.Date(2026, 10, 16, 14, 5, 0, 0, time.UTC)
	params, balance := svc.buildCheckoutParams(inv, "usd", now)
	require.NotNil(t, params)
	require.Len(t, params.LineItems, 1)
	li := params.LineItems[0]
	assert.Equal(t, int64(1), *li.Quantity)
	assert.Equal(t, "usd", *li.PriceData.Currency)
	// total = 165 - 15 + 12.38 = 162.38; minus 40.10 paid = 122.28
	assertDec(t, "162.38", inv.TotalAmount, "total")
	assertDec(t, "122.28", balance, "balance")
	assert.Equal(t, int64(12228), *li.PriceData.UnitAmount)
	assert.Equal(t, inv.ID.String(), params.Metadata["invoice_id"])
	require.NotNil(t, params.IdempotencyKey)

	again, _ := svc.buildCheckoutParams(inv, "usd", now.Add(10*time.Minute))
	assert.Equal(t, *params.IdempotencyKey, *again.IdempotencyKey, "a retry within the hour reuses the session")
	inv.AmountPaid = dec("50")
	other, _ := svc.buildCheckoutParams(inv, "usd", now)
	assert.NotEqual(t, *params.IdempotencyKey, *other.IdempotencyKey, "a different balance gets a new session")

	inv.AmountPaid = inv.TotalAmount
	none, _ := svc.buildCheckoutParams(inv, "usd", now)
	assert.Nil(t, none, "nothing owed, no session")
}

// Partially paid (and viewed/overdue) invoices can be paid online; closed or
// fully paid ones can't.
func TestValidatePaymentToken_PayableStatuses(t *testing.T) {
	f := setupIsolation(t)
	svc := NewInvoiceService(f.db)
	future := time.Now().Add(time.Hour)
	cases := map[models.InvoiceStatus]bool{
		models.InvoiceStatusSent:          true,
		models.InvoiceStatusViewed:        true,
		models.InvoiceStatusOverdue:       true,
		models.InvoiceStatusPartiallyPaid: true,
		models.InvoiceStatusDraft:         false,
		models.InvoiceStatusPaid:          false,
		models.InvoiceStatusCancelled:     false,
		models.InvoiceStatusRefunded:      false,
	}
	for status, payable := range cases {
		inv := newSentInvoice(t, f, 100, status)
		token := uuid.NewString()
		require.NoError(t, f.db.WithContext(f.ctxA).Model(&models.Invoice{}).Where("id = ?", inv.ID).
			Updates(map[string]any{"payment_token": token, "payment_token_expires_at": future}).Error)
		_, err := svc.ValidatePaymentToken(f.ctxA, token)
		if payable {
			assert.NoError(t, err, "status %s", status)
		} else {
			assert.ErrorIs(t, err, ErrInvoiceNotPayable, "status %s", status)
		}
	}
}
