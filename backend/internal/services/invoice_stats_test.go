package services

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"

	"github.com/javaknight1/servicepro/backend/internal/models"
	"github.com/javaknight1/servicepro/backend/internal/tenancy"
)

// T257: GET /invoices/stats sums the caller's receivables — open balances,
// the overdue part of them, and payments less refunds this month.
func TestGetInvoiceStats_SumsOpenOverdueAndCollected(t *testing.T) {
	f := setupIsolation(t)
	svc := NewInvoiceService(f.db)

	open := newSentInvoice(t, f, 100, models.InvoiceStatusSent)
	_, err := svc.RecordPayment(f.ctxA, &models.InvoicePayment{
		InvoiceID: open.ID, Amount: decimal.NewFromInt(40), PaymentDate: time.Now().UTC(), CreatedBy: &f.userA,
	})
	require.NoError(t, err)

	late := newSentInvoice(t, f, 50, models.InvoiceStatusOverdue)
	require.NoError(t, f.db.WithContext(f.ctxA).Model(&models.Invoice{}).Where("id = ?", late.ID).
		Update("due_date", time.Now().UTC().AddDate(0, 0, -10)).Error)

	newSentInvoice(t, f, 999, models.InvoiceStatusDraft)     // never issued
	newSentInvoice(t, f, 999, models.InvoiceStatusCancelled) // closed

	require.NoError(t, f.db.Create(&models.InvoiceRefund{
		ID: uuid.New(), InvoiceID: open.ID, Amount: decimal.NewFromInt(5), RefundDate: time.Now().UTC(),
	}).Error)

	stats, err := svc.GetInvoiceStats(f.ctxA)
	require.NoError(t, err)
	assertDec(t, "110", stats.TotalOutstanding, "outstanding = 60 left on the open invoice + 50 overdue")
	require.Equal(t, int64(2), stats.OutstandingInvoices)
	assertDec(t, "50", stats.OverdueAmount, "overdue")
	require.Equal(t, int64(1), stats.OverdueInvoices)
	assertDec(t, "35", stats.CollectedThisMonth, "collected = 40 paid - 5 refunded")
}

// Hard rule 1: tenant B sees none of tenant A's receivables.
func TestGetInvoiceStats_CrossTenant(t *testing.T) {
	f := setupIsolation(t)
	svc := NewInvoiceService(f.db)

	inv := newSentInvoice(t, f, 100, models.InvoiceStatusSent)
	_, err := svc.RecordPayment(f.ctxA, &models.InvoicePayment{
		InvoiceID: inv.ID, Amount: decimal.NewFromInt(25), PaymentDate: time.Now().UTC(), CreatedBy: &f.userA,
	})
	require.NoError(t, err)

	stats, err := svc.GetInvoiceStats(f.ctxB)
	require.NoError(t, err)
	require.True(t, stats.TotalOutstanding.IsZero(), "outstanding %s", stats.TotalOutstanding)
	require.Zero(t, stats.OutstandingInvoices)
	require.True(t, stats.OverdueAmount.IsZero())
	require.True(t, stats.CollectedThisMonth.IsZero(), "collected %s", stats.CollectedThisMonth)

	statsA, err := svc.GetInvoiceStats(f.ctxA)
	require.NoError(t, err)
	assertDec(t, "75", statsA.TotalOutstanding, "A's own outstanding")
	assertDec(t, "25", statsA.CollectedThisMonth, "A's own collected")
}

// Without a tenant in the context the stats fail closed.
func TestGetInvoiceStats_NoTenant(t *testing.T) {
	f := setupIsolation(t)
	_, err := NewInvoiceService(f.db).GetInvoiceStats(context.Background())
	require.ErrorIs(t, err, tenancy.ErrNoTenant)
}
