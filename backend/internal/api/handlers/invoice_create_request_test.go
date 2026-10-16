package handlers

import (
	"encoding/json"
	"testing"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// T237: POST /invoices takes terms_and_conditions (the web app used to set
// them with a follow-up PUT) and keeps each line's discount.
func TestCreateInvoiceRequest_ToInvoice(t *testing.T) {
	body := `{
		"customer_id": "4f6d1c1e-8f0a-4c43-9d0c-6a2b7f3e9a11",
		"tax_rate_id": "7a1b2c3d-0000-4000-8000-000000000001",
		"notes": "Thanks",
		"terms_and_conditions": "Net 15. Late fee 1.5%/month.",
		"lines": [
			{"description": "Labor", "quantity": "2", "unit_price": "100", "discount_percentage": "10"},
			{"description": "Parts", "quantity": "1", "unit_price": "50", "discount_amount": "5", "taxable": false}
		]
	}`
	var req CreateInvoiceRequest
	require.NoError(t, json.Unmarshal([]byte(body), &req))

	inv := req.toInvoice()
	assert.Equal(t, "Net 15. Late fee 1.5%/month.", inv.TermsAndConditions)
	assert.Equal(t, "Thanks", inv.Notes)
	require.NotNil(t, inv.TaxRateID)
	assert.Equal(t, "7a1b2c3d-0000-4000-8000-000000000001", inv.TaxRateID.String())
	require.Len(t, inv.Lines, 2)
	assert.True(t, inv.Lines[0].DiscountPercentage.Equal(decimal.NewFromInt(10)))
	assert.True(t, inv.Lines[1].DiscountAmount.Equal(decimal.NewFromInt(5)))
	assert.False(t, inv.Lines[1].Taxable)
}
