package services

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stripe/stripe-go/v87"

	"github.com/javaknight1/servicepro/backend/internal/models"
)

// bankCheckoutEvent is a Checkout Session event for a bank (ACH) session
// ServicePro created for inv.
func bankCheckoutEvent(t *testing.T, id, typ, account string, inv *models.Invoice, status string) *stripe.Event {
	t.Helper()
	raw, err := json.Marshal(map[string]any{
		"id": "cs_" + inv.ID.String()[:8], "object": "checkout.session", "payment_status": status, "status": "complete",
		"amount_total": 10000, "currency": "usd", "payment_intent": "pi_" + inv.ID.String()[:8],
		"metadata": map[string]string{"invoice_id": inv.ID.String(), "payment_type": "invoice", "pay_method": "bank"},
	})
	require.NoError(t, err)
	return &stripe.Event{ID: id, Type: stripe.EventType(typ), Account: account, Data: &stripe.EventData{Raw: raw}}
}

type pendingFixture struct {
	*connectFixture
	acctA, acctB string
	completed    func(context.Context, *stripe.Event) error
	failed       func(context.Context, *stripe.Event) error
}

func setupPending(t *testing.T) *pendingFixture {
	t.Helper()
	c := setupConnect(t, true)
	_, err := c.svc.StartOnboarding(c.ctxA, c.tenantA)
	require.NoError(t, err)
	_, err = c.svc.StartOnboarding(c.ctxB, c.tenantB)
	require.NoError(t, err)
	adapter := NewInvoicePaymentHandlerAdapter(NewInvoiceService(c.db))
	adapter.SetConnectedAccounts(c.svc)
	return &pendingFixture{
		connectFixture: c,
		acctA:          "acct_" + c.tenantA.String()[:8],
		acctB:          "acct_" + c.tenantB.String()[:8],
		completed:      adapter.CreateCheckoutCompletedHandler(),
		failed:         adapter.CreateCheckoutAsyncFailedHandler(),
	}
}

// T170: a bank Checkout that completes unpaid marks the invoice payment
// pending (status unchanged); the async success records the payment and
// clears it. Replays in any order change nothing more.
func TestPaymentPending_BankCompletedThenSucceeded(t *testing.T) {
	f := setupPending(t)
	ctx := context.Background()
	inv := newSentInvoice(t, f.isolationFixture, 100, models.InvoiceStatusSent)

	require.NoError(t, f.completed(ctx, bankCheckoutEvent(t, "evt_1", "checkout.session.completed", f.acctA, inv, "unpaid")))
	got := reloadInvoice(t, f.isolationFixture, inv.ID)
	require.NotNil(t, got.PaymentPendingAt, "pending after the bank session completed")
	assert.Equal(t, models.InvoiceStatusSent, got.Status, "status machine untouched")
	assert.True(t, got.AmountPaid.IsZero())
	first := *got.PaymentPendingAt

	// Replay: same pending, not moved.
	require.NoError(t, f.completed(ctx, bankCheckoutEvent(t, "evt_1r", "checkout.session.completed", f.acctA, inv, "unpaid")))
	assert.True(t, first.Equal(*reloadInvoice(t, f.isolationFixture, inv.ID).PaymentPendingAt))

	require.NoError(t, f.completed(ctx, bankCheckoutEvent(t, "evt_2", "checkout.session.async_payment_succeeded", f.acctA, inv, "paid")))
	got = reloadInvoice(t, f.isolationFixture, inv.ID)
	assert.Nil(t, got.PaymentPendingAt, "cleared when the bank payment settled")
	assert.Nil(t, got.PaymentPendingSessionID)
	assert.Equal(t, models.InvoiceStatusPaid, got.Status)
	assertDec(t, "100", got.AmountPaid, "credited once")

	// A late or replayed "completed" can't put a paid invoice back in pending,
	// and a replayed success can't credit twice.
	require.NoError(t, f.completed(ctx, bankCheckoutEvent(t, "evt_1rr", "checkout.session.completed", f.acctA, inv, "unpaid")))
	require.NoError(t, f.completed(ctx, bankCheckoutEvent(t, "evt_2r", "checkout.session.async_payment_succeeded", f.acctA, inv, "paid")))
	got = reloadInvoice(t, f.isolationFixture, inv.ID)
	assert.Nil(t, got.PaymentPendingAt)
	assertDec(t, "100", got.AmountPaid, "still credited once")
}

// T170: a failed bank payment clears pending and leaves the invoice in its
// prior status, so reminders resume; a failure for another session doesn't
// touch it.
func TestPaymentPending_BankFailedReturnsToPriorStatus(t *testing.T) {
	f := setupPending(t)
	ctx := context.Background()
	inv := newSentInvoice(t, f.isolationFixture, 100, models.InvoiceStatusOverdue)

	require.NoError(t, f.completed(ctx, bankCheckoutEvent(t, "evt_1", "checkout.session.completed", f.acctA, inv, "unpaid")))
	require.NotNil(t, reloadInvoice(t, f.isolationFixture, inv.ID).PaymentPendingAt)

	// A failure for a different session leaves this one pending.
	invSvc := NewInvoiceService(f.db)
	require.NoError(t, invSvc.ClearPaymentPending(f.ctxA, inv.ID, "cs_other"))
	require.NotNil(t, reloadInvoice(t, f.isolationFixture, inv.ID).PaymentPendingAt)

	require.NoError(t, f.failed(ctx, bankCheckoutEvent(t, "evt_3", "checkout.session.async_payment_failed", f.acctA, inv, "unpaid")))
	require.NoError(t, f.failed(ctx, bankCheckoutEvent(t, "evt_3r", "checkout.session.async_payment_failed", f.acctA, inv, "unpaid")))
	got := reloadInvoice(t, f.isolationFixture, inv.ID)
	assert.Nil(t, got.PaymentPendingAt)
	assert.Equal(t, models.InvoiceStatusOverdue, got.Status, "back to (still) its prior status")
	assert.True(t, got.AmountPaid.IsZero())
}

// T170: only the invoice tenant's own account can mark or clear pending.
func TestPaymentPending_OtherTenantsAccountIgnored(t *testing.T) {
	f := setupPending(t)
	ctx := context.Background()
	inv := newSentInvoice(t, f.isolationFixture, 100, models.InvoiceStatusSent)

	require.NoError(t, f.completed(ctx, bankCheckoutEvent(t, "evt_b", "checkout.session.completed", f.acctB, inv, "unpaid")))
	require.NoError(t, f.completed(ctx, bankCheckoutEvent(t, "evt_x", "checkout.session.completed", "acct_stranger", inv, "unpaid")))
	assert.Nil(t, reloadInvoice(t, f.isolationFixture, inv.ID).PaymentPendingAt)

	require.NoError(t, f.completed(ctx, bankCheckoutEvent(t, "evt_a", "checkout.session.completed", f.acctA, inv, "unpaid")))
	require.NoError(t, f.failed(ctx, bankCheckoutEvent(t, "evt_bf", "checkout.session.async_payment_failed", f.acctB, inv, "unpaid")))
	assert.NotNil(t, reloadInvoice(t, f.isolationFixture, inv.ID).PaymentPendingAt, "tenant B's account can't clear it")
}

// T170: a card session completing unpaid is not "payment pending".
func TestPaymentPending_OnlyForBankSessions(t *testing.T) {
	f := setupPending(t)
	inv := newSentInvoice(t, f.isolationFixture, 100, models.InvoiceStatusSent)
	require.NoError(t, f.completed(context.Background(),
		checkoutEvent(t, "evt_c", "checkout.session.completed", f.acctA, inv, "unpaid")))
	assert.Nil(t, reloadInvoice(t, f.isolationFixture, inv.ID).PaymentPendingAt)
}

// T170: while a bank payment settles the pay link starts no second payment
// and the pay page offers none.
func TestPaymentPending_BlocksASecondCheckout(t *testing.T) {
	f := setupPending(t)
	f.activate(f.acctA, true)
	require.NoError(t, f.svc.SyncAccount(context.Background(), f.acctA))
	pay, sessions := newPaymentService(t, f.connectFixture, true)
	inv, token := payableInvoice(t, f.connectFixture, 100)

	require.NoError(t, f.completed(context.Background(), bankCheckoutEvent(t, "evt_1", "checkout.session.completed", f.acctA, inv, "unpaid")))

	_, err := pay.CreateCheckoutSessionForInvoice(public, token, "card")
	assert.ErrorIs(t, err, ErrPaymentPending)
	assert.Empty(t, *sessions)

	page, err := pay.GetPayPage(public, token)
	require.NoError(t, err)
	assert.True(t, page.PaymentPending)
	assert.Empty(t, page.Methods)
}
