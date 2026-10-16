package services

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/javaknight1/servicepro/backend/internal/models"
	"github.com/javaknight1/servicepro/backend/internal/repository"
	"github.com/javaknight1/servicepro/backend/internal/tenancy"
)

func seedTerm(t *testing.T, f *isolationFixture, ctx context.Context, days int) *models.PaymentTerm {
	t.Helper()
	tid, _ := tenancy.TenantID(ctx)
	term := &models.PaymentTerm{ID: uuid.New(), TenantID: &tid, Name: "Net", TermType: models.PaymentTermType("net_30"),
		DaysUntilDue: days, IsActive: true}
	require.NoError(t, f.db.WithContext(ctx).Create(term).Error)
	return term
}

func seedRate(t *testing.T, f *isolationFixture, ctx context.Context, rate string, active bool) *models.TaxRate {
	t.Helper()
	tid, _ := tenancy.TenantID(ctx)
	tr := &models.TaxRate{ID: uuid.New(), TenantID: &tid, Name: "Tax " + rate, Rate: dec(rate),
		TaxType: models.TaxType("sales_tax"), IsActive: active}
	require.NoError(t, f.db.WithContext(ctx).Create(tr).Error)
	return tr
}

func newDraftInvoice(f *isolationFixture) *models.Invoice {
	return &models.Invoice{CustomerID: f.custA.ID, IssueDate: time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC), Lines: oneLine(100)}
}

// T127: an invoice created without a payment term / tax rate gets the
// tenant's defaults (due date and tax computed from them).
func TestInvoiceCreate_AppliesTenantDefaults(t *testing.T) {
	f := setupIsolation(t)
	term := seedTerm(t, f, f.ctxA, 15)
	rate := seedRate(t, f, f.ctxA, "0.1", true)
	f.setTenantSettings(t, f.ctxA, models.TenantSettings{DefaultPaymentTermID: &term.ID, DefaultTaxRateID: &rate.ID})
	svc := NewInvoiceService(f.db)

	inv, err := svc.CreateInvoice(f.ctxA, newDraftInvoice(f), f.userA)
	require.NoError(t, err)
	require.NotNil(t, inv.PaymentTermID)
	assert.Equal(t, term.ID, *inv.PaymentTermID)
	require.NotNil(t, inv.TaxRateID)
	assert.Equal(t, rate.ID, *inv.TaxRateID)
	assert.Equal(t, "2026-03-16", inv.DueDate.Format("2006-01-02"), "due date from the default term")
	assertDec(t, "10", inv.TaxAmount, "tax from the default rate")

	// Anything the request specifies wins over the defaults.
	other := seedTerm(t, f, f.ctxA, 45)
	zero := decimal.Zero
	req := newDraftInvoice(f)
	req.PaymentTermID = &other.ID
	req.ApplyTaxRate = &zero
	inv, err = svc.CreateInvoice(f.ctxA, req, f.userA)
	require.NoError(t, err)
	assert.Equal(t, other.ID, *inv.PaymentTermID)
	assert.Nil(t, inv.TaxRateID)
	assertDec(t, "0", inv.TaxAmount, "explicit zero tax rate")
}

// Default IDs pointing at another tenant's rows (or an inactive rate) are
// ignored: the lookup is tenant-scoped.
func TestInvoiceCreate_IgnoresForeignOrInactiveDefaults(t *testing.T) {
	f := setupIsolation(t)
	foreignTerm := seedTerm(t, f, f.ctxB, 15)
	foreignRate := seedRate(t, f, f.ctxB, "0.2", true)
	f.setTenantSettings(t, f.ctxA, models.TenantSettings{DefaultPaymentTermID: &foreignTerm.ID, DefaultTaxRateID: &foreignRate.ID})
	svc := NewInvoiceService(f.db)

	inv, err := svc.CreateInvoice(f.ctxA, newDraftInvoice(f), f.userA)
	require.NoError(t, err)
	assert.Nil(t, inv.PaymentTermID, "tenant B's term is not applied in tenant A")
	assert.Nil(t, inv.TaxRateID, "tenant B's tax rate is not applied in tenant A")
	assertDec(t, "0", inv.TaxAmount, "no tax")

	inactive := seedRate(t, f, f.ctxA, "0.3", false)
	f.setTenantSettings(t, f.ctxA, models.TenantSettings{DefaultTaxRateID: &inactive.ID})
	inv, err = svc.CreateInvoice(f.ctxA, newDraftInvoice(f), f.userA)
	require.NoError(t, err)
	assert.Nil(t, inv.TaxRateID, "inactive default rate is not applied")
}

// Quotes created without a tax rate take the tenant's default tax rate; an
// explicit rate (including 0) is kept, and another tenant's rate never
// applies.
func TestQuoteCreate_AppliesTenantDefaultTaxRate(t *testing.T) {
	f := setupIsolation(t)
	rate := seedRate(t, f, f.ctxA, "0.1", true)
	f.setTenantSettings(t, f.ctxA, models.TenantSettings{DefaultTaxRateID: &rate.ID})
	svc := quoteServiceFor(f)

	q, err := svc.CreateQuote(quoteRequest(f.custA.ID), f.userA)
	require.NoError(t, err)
	assertDec(t, "0.1", q.TaxRate, "default tax rate")
	assertDec(t, "10", q.TaxAmount, "tax computed from the default")
	assertDec(t, "110", q.Total, "total includes the default tax")

	req := quoteRequest(f.custA.ID)
	zero := decimal.Zero
	req.TaxRate = &zero
	q, err = svc.CreateQuote(req, f.userA)
	require.NoError(t, err)
	assertDec(t, "0", q.TaxRate, "explicit zero kept")

	// Tenant B pointing at tenant A's rate gets no tax.
	f.setTenantSettings(t, f.ctxB, models.TenantSettings{DefaultTaxRateID: &rate.ID})
	svcB := NewQuoteService(repository.NewQuoteRepository(f.db)).WithContext(f.ctxB)
	qb, err := svcB.CreateQuote(quoteRequest(f.custB.ID), f.userA)
	require.NoError(t, err)
	assertDec(t, "0", qb.TaxRate, "tenant A's rate is not applied in tenant B")
}
