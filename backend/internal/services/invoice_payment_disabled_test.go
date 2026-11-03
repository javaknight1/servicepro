package services

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/javaknight1/servicepro/backend/internal/models"
)

// T261: while online payment is off, no Checkout session is ever created —
// not even for a valid token — so no money can settle to the platform account.
func TestCreateCheckoutSession_DisabledRefusesBeforeAnyLookup(t *testing.T) {
	svc := &InvoicePaymentService{enabled: false} // no invoice service: must not be reached
	res, err := svc.CreateCheckoutSessionForInvoice(context.Background(), "any-token", "")
	require.ErrorIs(t, err, ErrInvoicePaymentDisabled)
	assert.Nil(t, res)
}

type payLinksFor map[uuid.UUID]bool

func (p payLinksFor) CanAcceptPayments(_ context.Context, tenantID uuid.UUID) bool {
	return p[tenantID]
}

// T261 + T169: the emailed pay link exists only when online payment is on
// AND the invoice's tenant can take payments on its own Stripe account.
func TestInvoicePaymentURL_OnlyWhenOnlinePaymentsOn(t *testing.T) {
	ctx := context.Background()
	token := "tok123"
	ready, notReady := uuid.New(), uuid.New()
	inv := &models.Invoice{TenantID: ready, PaymentToken: &token}

	off := NewInvoiceServiceWithConfig(nil, &InvoiceServiceConfig{FrontendURL: "https://app.example.test"})
	off.SetPayLinkPolicy(payLinksFor{ready: true})
	assert.Empty(t, off.invoicePaymentURL(ctx, inv), "payments off: no pay link")

	on := NewInvoiceServiceWithConfig(nil, &InvoiceServiceConfig{FrontendURL: "https://app.example.test", OnlinePayments: true})
	assert.Empty(t, on.invoicePaymentURL(ctx, inv), "no Connect policy: no pay link")
	on.SetPayLinkPolicy(payLinksFor{ready: true})
	assert.Equal(t, "https://app.example.test/pay/tok123", on.invoicePaymentURL(ctx, inv))
	assert.Empty(t, on.invoicePaymentURL(ctx, &models.Invoice{TenantID: ready}), "no token: no pay link")
	assert.Empty(t, on.invoicePaymentURL(ctx, &models.Invoice{TenantID: notReady, PaymentToken: &token}),
		"tenant without an active Stripe account: no pay link")
}
