package services

import (
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/javaknight1/servicepro/backend/internal/models"
	"github.com/javaknight1/servicepro/backend/internal/tenancy"
)

// newSentInvoice creates an invoice for tenant A with the given total and
// moves it to status (bypassing the send flow).
func newSentInvoice(t *testing.T, f *isolationFixture, total int64, status models.InvoiceStatus) *models.Invoice {
	t.Helper()
	svc := NewInvoiceService(f.db)
	now := time.Now()
	inv, err := svc.CreateInvoice(f.ctxA, &models.Invoice{
		CustomerID: f.custA.ID, InvoiceNumber: "INV-" + uuid.NewString()[:8], IssueDate: now, DueDate: now.Add(24 * time.Hour),
		Lines: oneLine(total),
	}, f.userA)
	require.NoError(t, err)
	require.NoError(t, f.db.WithContext(f.ctxA).Model(&models.Invoice{}).Where("id = ?", inv.ID).Update("status", status).Error)
	inv.Status = status
	return inv
}

func reloadInvoice(t *testing.T, f *isolationFixture, id uuid.UUID) *models.Invoice {
	t.Helper()
	var inv models.Invoice
	require.NoError(t, f.db.WithContext(f.ctxA).First(&inv, "id = ?", id).Error)
	return &inv
}

// T118 / audit 7.C1: recording a payment must update amount_paid, status and
// paid_date — previously only the payment row was written ("triggers will
// update invoice", but no trigger exists).
func TestRecordPayment_UpdatesInvoiceBalanceAndStatus(t *testing.T) {
	f := setupIsolation(t)
	svc := NewInvoiceService(f.db)
	inv := newSentInvoice(t, f, 100, models.InvoiceStatusSent)

	_, err := svc.RecordPayment(f.ctxA, &models.InvoicePayment{InvoiceID: inv.ID, Amount: decimal.NewFromInt(40), CreatedBy: &f.userA})
	require.NoError(t, err)
	got := reloadInvoice(t, f, inv.ID)
	assert.True(t, got.AmountPaid.Equal(decimal.NewFromInt(40)), "amount_paid = %s", got.AmountPaid)
	assert.Equal(t, models.InvoiceStatusPartiallyPaid, got.Status)
	assert.Nil(t, got.PaidDate, "not paid in full yet")

	_, err = svc.RecordPayment(f.ctxA, &models.InvoicePayment{InvoiceID: inv.ID, Amount: decimal.NewFromInt(61), CreatedBy: &f.userA})
	assert.ErrorIs(t, err, ErrOverpayment, "the overpayment guard sees the first payment")

	_, err = svc.RecordPayment(f.ctxA, &models.InvoicePayment{InvoiceID: inv.ID, Amount: decimal.NewFromInt(60), CreatedBy: &f.userA})
	require.NoError(t, err)
	got = reloadInvoice(t, f, inv.ID)
	assert.True(t, got.AmountPaid.Equal(decimal.NewFromInt(100)))
	assert.Equal(t, models.InvoiceStatusPaid, got.Status)
	assert.NotNil(t, got.PaidDate)

	_, err = svc.RecordPayment(f.ctxA, &models.InvoicePayment{InvoiceID: inv.ID, Amount: decimal.NewFromInt(1), CreatedBy: &f.userA})
	assert.ErrorIs(t, err, ErrInvoiceAlreadyPaid)

	var n int64
	require.NoError(t, f.db.Model(&models.InvoicePayment{}).Where("invoice_id = ?", inv.ID).Count(&n).Error)
	assert.Equal(t, int64(2), n)
}

func TestRecordPayment_RejectsClosedAndDraftInvoices(t *testing.T) {
	f := setupIsolation(t)
	svc := NewInvoiceService(f.db)
	for _, st := range []models.InvoiceStatus{models.InvoiceStatusDraft, models.InvoiceStatusCancelled, models.InvoiceStatusRefunded} {
		inv := newSentInvoice(t, f, 100, st)
		_, err := svc.RecordPayment(f.ctxA, &models.InvoicePayment{InvoiceID: inv.ID, Amount: decimal.NewFromInt(10), CreatedBy: &f.userA})
		assert.ErrorIs(t, err, ErrPaymentNotAllowed, "status %s", st)
		assert.True(t, reloadInvoice(t, f, inv.ID).AmountPaid.IsZero())
	}
}

func TestRecordPayment_CrossTenantIsNotFound(t *testing.T) {
	f := setupIsolation(t)
	svc := NewInvoiceService(f.db)
	inv := newSentInvoice(t, f, 100, models.InvoiceStatusSent)
	_, err := svc.RecordPayment(f.ctxB, &models.InvoicePayment{InvoiceID: inv.ID, Amount: decimal.NewFromInt(10)})
	assert.ErrorIs(t, err, ErrInvoiceNotFound)
}

// Audit 7.M15: invoices with money recorded against them can't be deleted or
// cancelled (that would hide received money from A/R and reports).
func TestDeleteAndCancel_RefuseInvoicesWithPayments(t *testing.T) {
	f := setupIsolation(t)
	svc := NewInvoiceService(f.db)
	inv := newSentInvoice(t, f, 100, models.InvoiceStatusSent)
	_, err := svc.RecordPayment(f.ctxA, &models.InvoicePayment{InvoiceID: inv.ID, Amount: decimal.NewFromInt(10), CreatedBy: &f.userA})
	require.NoError(t, err)

	assert.ErrorIs(t, svc.DeleteInvoice(f.ctxA, inv.ID), ErrInvoiceHasPayments)
	_, err = svc.CancelInvoice(f.ctxA, inv.ID, "changed mind")
	assert.ErrorIs(t, err, ErrInvoiceHasPayments)
	assert.Equal(t, models.InvoiceStatusPartiallyPaid, reloadInvoice(t, f, inv.ID).Status)

	unpaid := newSentInvoice(t, f, 100, models.InvoiceStatusSent)
	_, err = svc.CancelInvoice(f.ctxA, unpaid.ID, "duplicate")
	require.NoError(t, err)
}

// Stripe payments write a ledger row and stay idempotent per checkout session
// even after a different session was applied in between.
func TestMarkInvoiceAsPaid_WritesLedgerRow(t *testing.T) {
	f := setupIsolation(t)
	svc := NewInvoiceService(f.db)
	inv := newSentInvoice(t, f, 100, models.InvoiceStatusSent)
	ctx := tenancy.WithTenant(tenancy.WithSystem(f.ctxB), inv.TenantID)

	_, err := svc.MarkInvoiceAsPaid(ctx, inv.ID, decimal.NewFromInt(30), "cs_a", "pi_a")
	require.NoError(t, err)
	_, err = svc.MarkInvoiceAsPaid(ctx, inv.ID, decimal.NewFromInt(30), "cs_b", "pi_b")
	require.NoError(t, err)
	got, err := svc.MarkInvoiceAsPaid(ctx, inv.ID, decimal.NewFromInt(30), "cs_a", "pi_a") // late replay
	require.NoError(t, err)
	assert.True(t, got.AmountPaid.Equal(decimal.NewFromInt(60)), "amount_paid = %s", got.AmountPaid)

	var payments []models.InvoicePayment
	require.NoError(t, f.db.Where("invoice_id = ?", inv.ID).Order("amount").Find(&payments).Error)
	require.Len(t, payments, 2)
	for _, p := range payments {
		assert.Equal(t, "stripe", p.PaymentMethod)
		assert.Nil(t, p.CreatedBy)
		require.NotNil(t, p.StripeCheckoutSessionID)
	}
}

// Money Stripe has already taken for a cancelled invoice is recorded, but the
// invoice is not revived to paid (audit 7.H4).
func TestMarkInvoiceAsPaid_CancelledInvoiceKeepsStatus(t *testing.T) {
	f := setupIsolation(t)
	svc := NewInvoiceService(f.db)
	inv := newSentInvoice(t, f, 100, models.InvoiceStatusCancelled)
	ctx := tenancy.WithTenant(tenancy.WithSystem(f.ctxB), inv.TenantID)

	got, err := svc.MarkInvoiceAsPaid(ctx, inv.ID, decimal.NewFromInt(100), "cs_x", "pi_x")
	require.NoError(t, err)
	assert.Equal(t, models.InvoiceStatusCancelled, got.Status)
	assert.True(t, got.AmountPaid.Equal(decimal.NewFromInt(100)))
}

// Two staff members recording payments at the same moment must not overpay
// the invoice: the second waits on the row lock and then sees the first.
func TestPostgres_RecordPayment_ConcurrentPaymentsCannotOverpay(t *testing.T) {
	db := setupPostgres(t)
	tn := seedPGTenant(t, db)
	svc := NewInvoiceService(db)
	now := time.Now()
	inv, err := svc.CreateInvoice(tn.ctx, &models.Invoice{
		CustomerID: tn.customerID, InvoiceNumber: "INV-RACE-1", IssueDate: now, DueDate: now.Add(24 * time.Hour),
		Lines: oneLine(100),
	}, tn.userID)
	require.NoError(t, err)
	require.NoError(t, db.WithContext(tn.ctx).Model(&models.Invoice{}).Where("id = ?", inv.ID).Update("status", models.InvoiceStatusSent).Error)

	widenLockWindow(t)
	const workers = 8
	var wg sync.WaitGroup
	errs := make([]error, workers)
	start := make(chan struct{})
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			<-start
			_, errs[i] = svc.RecordPayment(tn.ctx, &models.InvoicePayment{
				InvoiceID: inv.ID, Amount: decimal.NewFromInt(60), PaymentMethod: "check", CreatedBy: &tn.userID,
			})
		}(i)
	}
	close(start)
	wg.Wait()

	ok := 0
	for _, err := range errs {
		if err == nil {
			ok++
		} else {
			assert.ErrorIs(t, err, ErrOverpayment)
		}
	}
	assert.Equal(t, 1, ok, "exactly one 60.00 payment fits a 100.00 invoice")

	got, err := svc.GetInvoice(tn.ctx, inv.ID)
	require.NoError(t, err)
	assert.True(t, got.AmountPaid.Equal(decimal.NewFromInt(60)), "amount_paid = %s", got.AmountPaid)
	assert.True(t, got.AmountDue.Equal(decimal.NewFromInt(40)), "generated amount_due = %s", got.AmountDue)
	assert.Equal(t, models.InvoiceStatusPartiallyPaid, got.Status)
	assert.Len(t, got.Payments, 1)
}

// Concurrent duplicate Stripe deliveries of one session credit it once.
func TestPostgres_MarkInvoiceAsPaid_ConcurrentReplaysCreditOnce(t *testing.T) {
	db := setupPostgres(t)
	tn := seedPGTenant(t, db)
	svc := NewInvoiceService(db)
	now := time.Now()
	inv, err := svc.CreateInvoice(tn.ctx, &models.Invoice{
		CustomerID: tn.customerID, InvoiceNumber: "INV-RACE-2", IssueDate: now, DueDate: now.Add(24 * time.Hour),
		Lines: oneLine(100),
	}, tn.userID)
	require.NoError(t, err)
	require.NoError(t, db.WithContext(tn.ctx).Model(&models.Invoice{}).Where("id = ?", inv.ID).Update("status", models.InvoiceStatusSent).Error)

	widenLockWindow(t)
	var wg sync.WaitGroup
	for i := 0; i < 6; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, _ = svc.MarkInvoiceAsPaid(tenancy.WithSystem(tn.ctx), inv.ID, decimal.NewFromInt(100), "cs_dup", "pi_dup")
		}()
	}
	wg.Wait()

	got, err := svc.GetInvoice(tn.ctx, inv.ID)
	require.NoError(t, err)
	assert.True(t, got.AmountPaid.Equal(decimal.NewFromInt(100)), "amount_paid = %s", got.AmountPaid)
	assert.Equal(t, models.InvoiceStatusPaid, got.Status)
	assert.Len(t, got.Payments, 1)
}

// widenLockWindow makes every locked invoice read pause briefly, so
// concurrent callers really overlap: without the row lock they would all read
// the same balance and the tests above fail.
func widenLockWindow(t *testing.T) {
	t.Helper()
	afterInvoiceLockHook = func() { time.Sleep(30 * time.Millisecond) }
	t.Cleanup(func() { afterInvoiceLockHook = nil })
}
