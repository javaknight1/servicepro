package services

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/javaknight1/servicepro/backend/internal/models"
)

func TestInvoiceStatus_TransitionTable(t *testing.T) {
	assert.True(t, models.InvoiceStatusDraft.CanTransitionTo(models.InvoiceStatusSent))
	assert.True(t, models.InvoiceStatusSent.CanTransitionTo(models.InvoiceStatusPaid))
	assert.True(t, models.InvoiceStatusPaid.CanTransitionTo(models.InvoiceStatusRefunded))
	assert.False(t, models.InvoiceStatusDraft.CanTransitionTo(models.InvoiceStatusPaid), "paid only via a payment on an issued invoice")
	assert.False(t, models.InvoiceStatusCancelled.CanTransitionTo(models.InvoiceStatusSent), "cancelled is terminal")
	assert.False(t, models.InvoiceStatusRefunded.CanTransitionTo(models.InvoiceStatusPaid), "refunded is terminal")
	assert.False(t, models.InvoiceStatusPaid.CanTransitionTo(models.InvoiceStatusCancelled))
	assert.Empty(t, models.InvoiceStatusCancelled.AllowedTransitions())
}

// Audit 7.H2: cancelled and refunded invoices could be edited and revived.
func TestClosedInvoices_AreReadOnly(t *testing.T) {
	f := setupIsolation(t)
	svc := NewInvoiceService(f.db)
	for _, st := range []models.InvoiceStatus{models.InvoiceStatusCancelled, models.InvoiceStatusRefunded} {
		inv := newSentInvoice(t, f, 100, st)
		_, err := svc.UpdateInvoice(f.ctxA, inv.ID, &InvoiceUpdate{Notes: strPtr("edit")}, f.userA)
		assert.ErrorIs(t, err, ErrInvoiceClosed, "update %s", st)
		_, err = svc.AddLineItem(f.ctxA, inv.ID, &models.InvoiceLine{Description: "x", Quantity: dec("1"), UnitPrice: dec("1")})
		assert.ErrorIs(t, err, ErrInvoiceClosed, "add line %s", st)
		_, err = svc.SendInvoice(f.ctxA, inv.ID)
		assert.ErrorIs(t, err, ErrInvalidInvoiceTransition, "send %s", st)
		_, err = svc.CancelInvoice(f.ctxA, inv.ID, "again")
		assert.ErrorIs(t, err, ErrInvalidInvoiceTransition, "cancel %s", st)
		assert.Equal(t, st, reloadInvoice(t, f, inv.ID).Status)
	}
}

// Status changes are written to invoice_audit_log in the same transaction.
func TestInvoiceTransitions_AreRecorded(t *testing.T) {
	f := setupIsolation(t)
	svc := NewInvoiceService(f.db)
	inv := newSentInvoice(t, f, 100, models.InvoiceStatusDraft)

	sent, err := svc.SendInvoice(f.ctxA, inv.ID)
	require.NoError(t, err)
	assert.Equal(t, models.InvoiceStatusSent, sent.Status)
	_, err = svc.RecordPayment(f.ctxA, &models.InvoicePayment{InvoiceID: inv.ID, Amount: dec("30"), CreatedBy: &f.userA})
	require.NoError(t, err)

	// Re-sending a partially paid invoice refreshes the link but keeps the status.
	resent, err := svc.SendInvoice(f.ctxA, inv.ID)
	require.NoError(t, err)
	assert.Equal(t, models.InvoiceStatusPartiallyPaid, resent.Status)

	var log []models.InvoiceAuditLog
	require.NoError(t, f.db.Where("invoice_id = ?", inv.ID).Order("created_at, action DESC").Find(&log).Error)
	require.Len(t, log, 2)
	assert.Equal(t, "sent", log[0].Action)
	assert.Equal(t, models.InvoiceStatusDraft, *log[0].FromStatus)
	assert.Equal(t, models.InvoiceStatusSent, *log[0].ToStatus)
	assert.Equal(t, "payment", log[1].Action)
	assert.Equal(t, models.InvoiceStatusPartiallyPaid, *log[1].ToStatus)
}
