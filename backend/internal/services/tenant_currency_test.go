package services

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/javaknight1/servicepro/backend/internal/models"
)

// T127: invoices and quotes are billed in the tenant's currency setting,
// fixed on the document at create time; another tenant's setting never
// applies, and no (or a bad) setting means USD.
func TestDocumentCurrency_FromTenantSetting(t *testing.T) {
	f := setupIsolation(t)
	f.setTenantSettings(t, f.ctxA, models.TenantSettings{Currency: "eur"})
	svc := NewInvoiceService(f.db)

	invA := createUnnumberedInvoice(t, svc, f.ctxA, f.custA.ID, f.userA)
	assert.Equal(t, "EUR", invA.Currency)
	invB := createUnnumberedInvoice(t, svc, f.ctxB, f.custB.ID, f.userA)
	assert.Equal(t, "USD", invB.Currency, "tenant A's currency does not leak into tenant B")

	q := seedQuote(t, f, models.QuoteStatusDraft)
	assert.Equal(t, "EUR", q.Currency)

	// The setting is fixed on the document: changing it later doesn't
	// change existing invoices.
	f.setTenantSettings(t, f.ctxA, models.TenantSettings{Currency: "GBP"})
	got, err := svc.GetInvoice(f.ctxA, invA.ID)
	require.NoError(t, err)
	assert.Equal(t, "EUR", got.Currency)

	f.setTenantSettings(t, f.ctxA, models.TenantSettings{Currency: "dollars"})
	assert.Equal(t, "USD", createUnnumberedInvoice(t, svc, f.ctxA, f.custA.ID, f.userA).Currency, "invalid setting falls back to USD")
}

// An invoice for a quote bills in the quote's currency, even if the tenant
// setting changed in between.
func TestDocumentCurrency_InvoiceFromQuoteKeepsQuoteCurrency(t *testing.T) {
	f := setupIsolation(t)
	f.setTenantSettings(t, f.ctxA, models.TenantSettings{Currency: "CAD"})
	q := seedQuote(t, f, models.QuoteStatusAccepted)
	f.setTenantSettings(t, f.ctxA, models.TenantSettings{Currency: "EUR"})

	inv, err := NewInvoiceService(f.db).CreateInvoice(f.ctxA, invoiceFor(f, nil, &q.ID), f.userA)
	require.NoError(t, err)
	assert.Equal(t, "CAD", inv.Currency)
}

// Stripe Checkout charges in the invoice's currency (lower-case for Stripe),
// not a hard-coded usd.
func TestInvoiceCurrency_ForStripe(t *testing.T) {
	assert.Equal(t, "eur", invoiceCurrency(&models.Invoice{Currency: "EUR"}))
	assert.Equal(t, "jpy", invoiceCurrency(&models.Invoice{Currency: "jpy"}))
	assert.Equal(t, "usd", invoiceCurrency(&models.Invoice{}), "pre-T127 rows without a currency")

	svc := &InvoicePaymentService{successURL: "https://x/s", cancelURL: "https://x/c"}
	inv := &models.Invoice{ID: uuid.New(), InvoiceNumber: "INV-1", Currency: "JPY",
		TotalAmount: dec("1500"), AmountPaid: dec("0")}
	params, _ := svc.buildCheckoutParams(inv, invoiceCurrency(inv), time.Now())
	require.NotNil(t, params)
	li := params.LineItems[0]
	assert.Equal(t, "jpy", *li.PriceData.Currency)
	assert.Equal(t, int64(1500), *li.PriceData.UnitAmount, "zero-decimal currency in minor units")
}

func TestNormalizeCurrency(t *testing.T) {
	assert.Equal(t, "USD", models.NormalizeCurrency(""))
	assert.Equal(t, "EUR", models.NormalizeCurrency(" eur "))
	assert.Equal(t, "USD", models.NormalizeCurrency("EURO"))
	assert.Equal(t, "USD", models.NormalizeCurrency("E1R"))
}
