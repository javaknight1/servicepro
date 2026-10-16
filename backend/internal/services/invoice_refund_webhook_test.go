package services

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stripe/stripe-go/v76"

	"github.com/javaknight1/servicepro/backend/internal/models"
	"github.com/javaknight1/servicepro/backend/internal/tenancy"
)

func chargeRefundedEvent(t *testing.T, pi string, refunded int64) *stripe.Event {
	t.Helper()
	raw, err := json.Marshal(map[string]any{
		"id": "ch_1", "object": "charge", "amount": 10000, "amount_refunded": refunded,
		"currency": "usd", "refunded": refunded == 10000, "payment_intent": pi,
	})
	require.NoError(t, err)
	return &stripe.Event{ID: "evt_1", Type: "charge.refunded", Data: &stripe.EventData{Raw: raw}}
}

// T125: charge.refunded was only logged. It now lowers the invoice's
// amount_paid and marks a fully refunded invoice refunded.
func TestChargeRefundedHandler_UpdatesInvoice(t *testing.T) {
	f := setupIsolation(t)
	svc := NewInvoiceService(f.db)
	inv := newSentInvoice(t, f, 100, models.InvoiceStatusSent)
	sys := tenancy.WithSystem(context.Background())
	_, err := svc.MarkInvoiceAsPaid(tenancy.WithTenant(sys, inv.TenantID), inv.ID, dec("100"), "cs_w", "pi_w")
	require.NoError(t, err)

	handle := NewInvoicePaymentHandlerAdapter(svc).CreateChargeRefundedHandler()
	require.NoError(t, handle(context.Background(), chargeRefundedEvent(t, "pi_w", 4000)))
	got := reloadInvoice(t, f, inv.ID)
	assertDec(t, "60", got.AmountPaid, "40.00 refunded")
	assert.Equal(t, models.InvoiceStatusPartiallyPaid, got.Status)

	require.NoError(t, handle(context.Background(), chargeRefundedEvent(t, "pi_w", 10000)))
	got = reloadInvoice(t, f, inv.ID)
	assert.True(t, got.AmountPaid.IsZero())
	assert.Equal(t, models.InvoiceStatusRefunded, got.Status)
}
