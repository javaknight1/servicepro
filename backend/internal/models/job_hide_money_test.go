package models

import (
	"testing"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
)

// T035: technicians get job responses without prices, costs or the
// customer's billing rules, but with what they need to do the work.
func TestJobResponseHideMoney(t *testing.T) {
	limit := decimal.NewFromInt(500)
	po := "PO-1"
	job := &Job{
		ID: uuid.New(), Title: "Clean lobby", Status: JobStatusScheduled,
		EstimatedCost: 120, ActualCost: 130, TaxAmount: 9.1, TotalAmount: 139.1,
		ServiceAddress: ServiceAddress{Street: "1 Main St", City: "Springfield"},
		Customer: &Customer{
			FirstName: "Pat", LastName: "Lee", PhonePrimary: "555-0100",
			BillingRules: BillingRules{NTELimit: &limit, PORequired: true, DefaultPONumber: &po},
		},
		Materials: []JobMaterial{{Name: "Filter", Quantity: 2, Unit: "each", UnitCost: 9, TotalCost: 18}},
	}
	r := job.ToResponse()
	r.Warnings = []string{"Actual cost is 8% higher than estimate"}
	invoiceID := uuid.New()
	r.InvoiceID = &invoiceID
	r.HideMoney()

	assert.Zero(t, r.EstimatedCost)
	assert.Zero(t, r.ActualCost)
	assert.Zero(t, r.TaxAmount)
	assert.Zero(t, r.TotalAmount)
	assert.Zero(t, r.Materials[0].UnitCost)
	assert.Zero(t, r.Materials[0].TotalCost)
	assert.Equal(t, 2.0, r.Materials[0].Quantity, "quantities stay")
	assert.Nil(t, r.Customer.NTELimit)
	assert.False(t, r.Customer.PORequired)
	assert.Nil(t, r.Customer.DefaultPONumber)
	assert.Nil(t, r.Warnings)
	assert.Nil(t, r.InvoiceID, "no link to the invoice")
	assert.Equal(t, "555-0100", r.Customer.PhonePrimary, "contact details stay")
	assert.Equal(t, "1 Main St", r.ServiceAddress.Street)
}
