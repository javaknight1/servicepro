package services

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/javaknight1/servicepro/backend/internal/models"
)

// T261: while online payment is off, no Checkout session is ever created —
// not even for a valid token — so no money can settle to the platform account.
func TestCreateCheckoutSession_DisabledRefusesBeforeAnyLookup(t *testing.T) {
	svc := &InvoicePaymentService{enabled: false} // no invoice service: must not be reached
	res, err := svc.CreateCheckoutSessionForInvoice(context.Background(), "any-token")
	require.ErrorIs(t, err, ErrInvoicePaymentDisabled)
	assert.Nil(t, res)
}

// T261: the emailed pay link exists only when online payment is on.
func TestInvoicePaymentURL_OnlyWhenOnlinePaymentsOn(t *testing.T) {
	token := "tok123"
	inv := &models.Invoice{PaymentToken: &token}

	off := NewInvoiceServiceWithConfig(nil, &InvoiceServiceConfig{BackendURL: "https://api.example.test"})
	assert.Empty(t, off.invoicePaymentURL(inv), "payments off: no pay link")

	on := NewInvoiceServiceWithConfig(nil, &InvoiceServiceConfig{BackendURL: "https://api.example.test", OnlinePayments: true})
	assert.Equal(t, "https://api.example.test/api/v1/public/invoices/pay/tok123", on.invoicePaymentURL(inv))
	assert.Empty(t, on.invoicePaymentURL(&models.Invoice{}), "no token: no pay link")
}
