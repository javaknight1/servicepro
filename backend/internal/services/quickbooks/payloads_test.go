package quickbooks

import (
	"testing"
	"time"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/javaknight1/servicepro/backend/internal/models"
)

func d(s string) decimal.Decimal { return decimal.RequireFromString(s) }

func TestInvoicePayloadMatchesOurTotals(t *testing.T) {
	site := "Tower B"
	inv := &models.Invoice{
		InvoiceNumber:  "INV-2026-000123456789", // longer than QuickBooks' 21
		IssueDate:      time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC),
		DueDate:        time.Date(2026, 12, 1, 0, 0, 0, 0, time.UTC),
		DiscountAmount: d("10.00"), TaxAmount: d("7.45"), PONumber: "PO-77",
	}
	lines := []models.InvoiceLine{
		{Description: "Nightly cleaning", Quantity: d("2"), UnitPrice: d("50.00"), Taxable: true, TaxAmount: d("7.45")},
		// 1.5 x 10.33 = 15.495: not a whole cent, so no Qty/UnitPrice.
		{Description: "Supplies", Quantity: d("1.5"), UnitPrice: d("10.33"), SiteName: &site},
		// Line discount: amount is the net.
		{Description: "Windows", Quantity: d("1"), UnitPrice: d("40.00"), DiscountAmount: d("5.00")},
	}
	p := invoicePayload(inv, lines, invoiceRefs{CustomerID: "58", ItemID: "1", TaxCodeID: "3"})

	assert.Equal(t, "INV-2026-00012345678", p.DocNumber[:20])
	assert.Len(t, p.DocNumber, 21)
	assert.Equal(t, "2026-11-01", p.TxnDate)
	assert.Equal(t, "2026-12-01", p.DueDate)
	assert.Equal(t, "58", p.CustomerRef.Value)
	require.Len(t, p.Line, 4)

	assert.Equal(t, "100.00", p.Line[0].Amount.String())
	assert.Equal(t, "2", p.Line[0].SalesItemLineDetail.Qty.String())
	assert.Equal(t, "50.00", p.Line[0].SalesItemLineDetail.UnitPrice.String())
	assert.Equal(t, "TAX", p.Line[0].SalesItemLineDetail.TaxCodeRef.Value)

	assert.Equal(t, "15.50", p.Line[1].Amount.String())
	assert.Nil(t, p.Line[1].SalesItemLineDetail.Qty)
	assert.Equal(t, "Tower B · Supplies", p.Line[1].Description)
	assert.Equal(t, "NON", p.Line[1].SalesItemLineDetail.TaxCodeRef.Value)

	assert.Equal(t, "35.00", p.Line[2].Amount.String())
	assert.Nil(t, p.Line[2].SalesItemLineDetail.UnitPrice)

	assert.Equal(t, "DiscountLineDetail", p.Line[3].DetailType)
	assert.Equal(t, "10.00", p.Line[3].Amount.String())
	assert.Equal(t, "7.45", p.TxnTaxDetail.TotalTax.String())
	assert.Equal(t, "3", p.TxnTaxDetail.TxnTaxCodeRef.Value)
	assert.Contains(t, p.PrivateNote, "PO PO-77")
}

func TestCustomerPayload(t *testing.T) {
	company := "Harbor: View  Mgmt"
	c := &models.Customer{FirstName: "Dana", LastName: "Reed", CompanyName: &company, Email: " d@example.com ",
		PhonePrimary: "555", BillingAddressStreet: "1 Main", BillingAddressCity: "Columbus", BillingAddressState: "OH", BillingAddressZip: "43215"}
	p := customerPayload(c)
	assert.Equal(t, "Harbor- View Mgmt", p.DisplayName, "':' is QuickBooks' sub-customer separator")
	assert.Equal(t, "d@example.com", p.PrimaryEmailAddr.Address)
	assert.Equal(t, "OH", p.BillAddr.CountrySubDivisionCode)

	c.CompanyName = nil
	assert.Equal(t, "Dana Reed", customerPayload(c).DisplayName)
}

func TestPaymentPayloadDropsStripeIDsFromRefNum(t *testing.T) {
	p := paymentPayload(&models.InvoicePayment{Amount: d("108"), PaymentDate: time.Date(2026, 11, 3, 0, 0, 0, 0, time.UTC),
		PaymentMethod: "stripe", ReferenceNumber: "pi_3QxyzLongStripeIdentifier"}, "INV-1", "58", "130", "35")
	assert.Equal(t, "108.00", p.TotalAmt.String())
	assert.Empty(t, p.PaymentRefNum)
	assert.Contains(t, p.PrivateNote, "pi_3QxyzLongStripeIdentifier")
	assert.Equal(t, "35", p.DepositToAccountRef.Value)
	assert.Equal(t, "130", p.Line[0].LinkedTxn[0].TxnID)
}

func TestFormatMoney(t *testing.T) {
	assert.Equal(t, "$9,882.00", formatMoney(d("9882"), "USD"))
	assert.Equal(t, "$1,234,567.50", formatMoney(d("1234567.5"), ""))
	assert.Equal(t, "$5.00", formatMoney(d("5"), "usd"))
	assert.Equal(t, "12.00 CAD", formatMoney(d("12"), "CAD"))
}
