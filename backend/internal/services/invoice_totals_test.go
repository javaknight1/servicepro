package services

import (
	"testing"
	"time"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/javaknight1/servicepro/backend/internal/models"
)

func strPtr(s string) *string { return &s }

func dec(s string) decimal.Decimal { return decimal.RequireFromString(s) }

// oneLine is a single taxable line worth amount (quantity 1).
func oneLine(amount int64) []models.InvoiceLine {
	return []models.InvoiceLine{{Description: "Service", Quantity: decimal.NewFromInt(1), UnitPrice: decimal.NewFromInt(amount), Taxable: true}}
}

func assertDec(t *testing.T, want string, got decimal.Decimal, msg string) {
	t.Helper()
	assert.True(t, dec(want).Equal(got), "%s: want %s, got %s", msg, want, got)
}

// Audit 7.M7: the invoice-level discount is applied before tax, line
// discount percentages are applied, and non-taxable lines are not taxed.
func TestCalculateInvoiceTotals_DiscountBeforeTax(t *testing.T) {
	lines := []models.InvoiceLine{
		{Quantity: dec("2"), UnitPrice: dec("50"), Taxable: true},                                 // 100.00
		{Quantity: dec("1"), UnitPrice: dec("200"), DiscountPercentage: dec("25"), Taxable: true}, // 200 - 50 = 150.00
		{Quantity: dec("1"), UnitPrice: dec("50"), Taxable: false},                                // 50.00, untaxed
	}
	got, err := calculateInvoiceTotals(lines, dec("30"), dec("0.10"))
	require.NoError(t, err)

	assertDec(t, "300", got.Subtotal, "subtotal")
	assertDec(t, "50", lines[1].DiscountAmount, "line discount from percentage")
	// Discount 30 is allocated 10/15/5; tax = (100-10 + 150-15) * 10% = 22.50
	assertDec(t, "9", lines[0].TaxAmount, "line 1 tax")
	assertDec(t, "13.5", lines[1].TaxAmount, "line 2 tax")
	assertDec(t, "0", lines[2].TaxAmount, "non-taxable line")
	assertDec(t, "22.5", got.Tax, "tax")
	assertDec(t, "292.5", got.Total, "total = 300 - 30 + 22.50")
}

func TestCalculateInvoiceTotals_RoundsPerLineAndSumsExactly(t *testing.T) {
	lines := []models.InvoiceLine{
		{Quantity: dec("3"), UnitPrice: dec("0.33"), Taxable: true},
		{Quantity: dec("1.5"), UnitPrice: dec("19.99"), Taxable: true},
	}
	got, err := calculateInvoiceTotals(lines, dec("0.01"), dec("0.0825"))
	require.NoError(t, err)
	sumTax := lines[0].TaxAmount.Add(lines[1].TaxAmount)
	assert.True(t, sumTax.Equal(got.Tax), "line taxes add up to the invoice tax")
	assert.True(t, got.Subtotal.Equal(lineNet(lines[0]).Add(lineNet(lines[1]))), "lines add up to the subtotal")
	assert.True(t, got.Total.Equal(got.Subtotal.Sub(got.Discount).Add(got.Tax)))
	assert.Equal(t, int32(-2), got.Total.Round(2).Exponent())
}

func TestCalculateInvoiceTotals_RejectsBadInput(t *testing.T) {
	_, err := calculateInvoiceTotals(oneLine(10), dec("11"), decimal.Zero)
	assert.ErrorIs(t, err, ErrInvalidDiscount)
	_, err = calculateInvoiceTotals(oneLine(10), dec("-1"), decimal.Zero)
	assert.ErrorIs(t, err, ErrInvalidDiscount)
	_, err = calculateInvoiceTotals(oneLine(10), decimal.Zero, dec("1.5"))
	assert.ErrorIs(t, err, ErrInvalidTaxRate)
	_, err = calculateInvoiceTotals([]models.InvoiceLine{{Quantity: dec("0"), UnitPrice: dec("1")}}, decimal.Zero, decimal.Zero)
	assert.ErrorIs(t, err, ErrInvalidLineItem)
	_, err = calculateInvoiceTotals([]models.InvoiceLine{{Quantity: dec("1"), UnitPrice: dec("10"), DiscountAmount: dec("11")}}, decimal.Zero, decimal.Zero)
	assert.ErrorIs(t, err, ErrInvalidLineItem)
}

// Audit 7.H3: totals are recomputed on the server on every update — a
// discount change alone updates total_amount, and lines sent without a tax
// rate keep the invoice's rate instead of zeroing tax.
func TestUpdateInvoice_RecalculatesTotalsOnEveryChange(t *testing.T) {
	f := setupIsolation(t)
	svc := NewInvoiceService(f.db)
	now := time.Now()
	rate := dec("0.10")
	inv, err := svc.CreateInvoice(f.ctxA, &models.Invoice{
		CustomerID: f.custA.ID, InvoiceNumber: "INV-TOT-1", IssueDate: now, DueDate: now.Add(24 * time.Hour),
		Lines: oneLine(100), ApplyTaxRate: &rate,
		// Client-supplied totals are ignored.
		Subtotal: dec("1"), TaxAmount: dec("1"), TotalAmount: dec("1"),
	}, f.userA)
	require.NoError(t, err)
	assertDec(t, "110", inv.TotalAmount, "create computes totals server-side")

	discount := dec("20")
	got, err := svc.UpdateInvoice(f.ctxA, inv.ID, &InvoiceUpdate{DiscountAmount: &discount}, f.userA)
	require.NoError(t, err)
	assertDec(t, "8", got.TaxAmount, "tax on the discounted amount")
	assertDec(t, "88", got.TotalAmount, "discount-only update recalculates the total")

	got, err = svc.UpdateInvoice(f.ctxA, inv.ID, &InvoiceUpdate{Lines: oneLine(200)}, f.userA)
	require.NoError(t, err)
	assertDec(t, "18", got.TaxAmount, "lines without tax_rate keep the invoice's 10% rate")
	assertDec(t, "198", got.TotalAmount, "200 - 20 + 18")

	newRate := dec("0.05")
	got, err = svc.UpdateInvoice(f.ctxA, inv.ID, &InvoiceUpdate{TaxRate: &newRate}, f.userA)
	require.NoError(t, err)
	assertDec(t, "9", got.TaxAmount, "rate-only update recalculates tax")
	assertDec(t, "189", got.TotalAmount, "200 - 20 + 9")
	require.Len(t, got.Lines, 1)
	assertDec(t, "9", got.Lines[0].TaxAmount, "line tax rewritten")
}

func TestUpdateInvoice_TotalCannotDropBelowAmountPaid(t *testing.T) {
	f := setupIsolation(t)
	svc := NewInvoiceService(f.db)
	inv := newSentInvoice(t, f, 100, models.InvoiceStatusSent)
	_, err := svc.RecordPayment(f.ctxA, &models.InvoicePayment{InvoiceID: inv.ID, Amount: dec("80"), CreatedBy: &f.userA})
	require.NoError(t, err)

	_, err = svc.UpdateInvoice(f.ctxA, inv.ID, &InvoiceUpdate{Lines: oneLine(50)}, f.userA)
	var rule *InvoiceRuleError
	assert.ErrorAs(t, err, &rule)
}

// The invoice PDF's line amounts add up to its subtotal, and it shows the
// discount and the real balance due.
func TestBuildInvoicePDFData_LinesSumToTotals(t *testing.T) {
	lines := []models.InvoiceLine{
		{Description: "a", Quantity: dec("2"), UnitPrice: dec("50"), DiscountAmount: dec("10"), Taxable: true},
		{Description: "b", Quantity: dec("1"), UnitPrice: dec("25"), Taxable: true},
	}
	totals, err := calculateInvoiceTotals(lines, dec("5"), dec("0.1"))
	require.NoError(t, err)
	inv := &models.Invoice{Lines: lines, AmountPaid: dec("20")}
	setInvoiceTotals(inv, totals)

	data := (&DocumentPDFService{}).buildInvoicePDFData(inv)
	sum := 0.0
	for _, li := range data.LineItems {
		sum += li.Amount
	}
	assert.InDelta(t, data.Subtotal, sum, 0.001)
	assert.InDelta(t, 5, data.Discount, 0.001)
	assert.InDelta(t, data.Total, data.Subtotal-data.Discount+data.TaxAmount, 0.001)
	assert.InDelta(t, data.Total-20, data.AmountDue, 0.001)
}
