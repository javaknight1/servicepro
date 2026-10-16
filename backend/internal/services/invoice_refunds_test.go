package services

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/javaknight1/servicepro/backend/internal/models"
	"github.com/javaknight1/servicepro/backend/internal/tenancy"
)

// T125 / audit 7.H5: refunds never reduced amount_paid and "refunded" was
// never set, so refunded money still counted as collected.
func TestRecordRefund_PartialThenFull(t *testing.T) {
	f := setupIsolation(t)
	svc := NewInvoiceService(f.db)
	inv := newSentInvoice(t, f, 100, models.InvoiceStatusSent)
	_, err := svc.RecordPayment(f.ctxA, &models.InvoicePayment{InvoiceID: inv.ID, Amount: dec("100"), CreatedBy: &f.userA})
	require.NoError(t, err)
	require.Equal(t, models.InvoiceStatusPaid, reloadInvoice(t, f, inv.ID).Status)

	_, err = svc.RecordRefund(f.ctxA, &models.InvoiceRefund{InvoiceID: inv.ID, Amount: dec("30"), CreatedBy: &f.userA})
	require.NoError(t, err)
	got := reloadInvoice(t, f, inv.ID)
	assertDec(t, "70", got.AmountPaid, "amount_paid after partial refund")
	assert.Equal(t, models.InvoiceStatusPartiallyPaid, got.Status)
	assert.Nil(t, got.PaidDate)

	_, err = svc.RecordRefund(f.ctxA, &models.InvoiceRefund{InvoiceID: inv.ID, Amount: dec("70.01"), CreatedBy: &f.userA})
	assert.ErrorIs(t, err, ErrRefundExceedsPaid)

	_, err = svc.RecordRefund(f.ctxA, &models.InvoiceRefund{InvoiceID: inv.ID, Amount: dec("70"), CreatedBy: &f.userA})
	require.NoError(t, err)
	got = reloadInvoice(t, f, inv.ID)
	assert.True(t, got.AmountPaid.IsZero())
	assert.Equal(t, models.InvoiceStatusRefunded, got.Status)

	_, err = svc.RecordRefund(f.ctxA, &models.InvoiceRefund{InvoiceID: inv.ID, Amount: dec("1"), CreatedBy: &f.userA})
	assert.ErrorIs(t, err, ErrRefundNotAllowed)

	full, err := svc.GetInvoice(f.ctxA, inv.ID)
	require.NoError(t, err)
	assert.Len(t, full.Refunds, 2)

	_, err = svc.RecordRefund(f.ctxB, &models.InvoiceRefund{InvoiceID: inv.ID, Amount: dec("1")})
	assert.ErrorIs(t, err, ErrInvoiceNotFound, "another tenant can't refund A's invoice")
}

// Stripe reports the cumulative refunded amount; replays and out-of-order
// deliveries record each refunded cent once.
func TestApplyStripeRefunds_CumulativeAndIdempotent(t *testing.T) {
	f := setupIsolation(t)
	svc := NewInvoiceService(f.db)
	inv := newSentInvoice(t, f, 100, models.InvoiceStatusSent)
	sys := tenancy.WithSystem(context.Background()) // webhooks run with no tenant
	_, err := svc.MarkInvoiceAsPaid(tenancy.WithTenant(sys, inv.TenantID), inv.ID, dec("100"), "cs_r", "pi_r")
	require.NoError(t, err)

	require.NoError(t, svc.ApplyStripeRefunds(sys, "pi_r", "ch_r", dec("25")))
	require.NoError(t, svc.ApplyStripeRefunds(sys, "pi_r", "ch_r", dec("25"))) // replay
	got := reloadInvoice(t, f, inv.ID)
	assertDec(t, "75", got.AmountPaid, "one 25.00 refund")
	assert.Equal(t, models.InvoiceStatusPartiallyPaid, got.Status)

	require.NoError(t, svc.ApplyStripeRefunds(sys, "pi_r", "ch_r", dec("100")))
	require.NoError(t, svc.ApplyStripeRefunds(sys, "pi_r", "ch_r", dec("25"))) // stale, out of order
	got = reloadInvoice(t, f, inv.ID)
	assert.True(t, got.AmountPaid.IsZero())
	assert.Equal(t, models.InvoiceStatusRefunded, got.Status)

	var refunds []models.InvoiceRefund
	require.NoError(t, f.db.Where("invoice_id = ?", inv.ID).Find(&refunds).Error)
	assert.Len(t, refunds, 2)

	assert.NoError(t, svc.ApplyStripeRefunds(sys, "pi_unknown", "ch_x", dec("10")), "charges that aren't invoice payments are ignored")
}

// A refunded paid job goes back to invoiced.
func TestRefund_MovesPaidJobBackToInvoiced(t *testing.T) {
	f := setupIsolation(t)
	svc := NewInvoiceService(f.db)
	job := seedJob(t, f, models.JobStatusCompleted)
	inv, err := svc.CreateInvoice(f.ctxA, invoiceFor(f, &job.ID, nil), f.userA)
	require.NoError(t, err)
	_, err = svc.SendInvoice(f.ctxA, inv.ID)
	require.NoError(t, err)
	_, err = svc.RecordPayment(f.ctxA, &models.InvoicePayment{InvoiceID: inv.ID, Amount: dec("100"), CreatedBy: &f.userA})
	require.NoError(t, err)

	_, err = svc.RecordRefund(f.ctxA, &models.InvoiceRefund{InvoiceID: inv.ID, Amount: dec("10"), CreatedBy: &f.userA})
	require.NoError(t, err)
	var j models.Job
	require.NoError(t, f.db.WithContext(f.ctxA).First(&j, "id = ?", job.ID).Error)
	assert.Equal(t, models.JobStatusInvoiced, j.Status)
}
