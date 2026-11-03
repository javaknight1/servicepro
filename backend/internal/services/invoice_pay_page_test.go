package services

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/javaknight1/servicepro/backend/internal/models"
)

// T170: the pay page shows the invoice's own business (never another
// tenant's), and bank first with the fee the business pays for each method,
// computed with the same helper as the Checkout session.
func TestGetPayPage_BusinessAndMethodsAreTheInvoiceTenants(t *testing.T) {
	c := setupConnect(t, true)
	pay, _ := newPaymentService(t, c, true)
	require.NoError(t, c.db.Model(&models.Tenant{}).Where("id = ?", c.tenantA).Update("settings",
		models.TenantSettings{BusinessCity: "Columbus", BusinessState: "OH"}).Error)
	require.NoError(t, c.db.Model(&models.Tenant{}).Where("id = ?", c.tenantB).Update("settings",
		models.TenantSettings{BusinessCity: "Elsewhere", BusinessState: "ZZ"}).Error)

	// Tenant B has a fully active account; tenant A has none yet.
	_, err := c.svc.StartOnboarding(c.ctxB, c.tenantB)
	require.NoError(t, err)
	acctB := "acct_" + c.tenantB.String()[:8]
	c.activate(acctB, true)
	require.NoError(t, c.svc.SyncAccount(context.Background(), acctB))

	_, token := payableInvoice(t, c, 5000) // tenant A, $5,000.00
	page, err := pay.GetPayPage(public, token)
	require.NoError(t, err)
	assert.Equal(t, c.tenantA, page.Invoice.TenantID)
	assert.Equal(t, "Brightline "+c.tenantA.String()[:4], page.BusinessName)
	assert.Equal(t, "Columbus, OH", page.BusinessLocation)
	assert.Empty(t, page.Methods, "tenant A can't take payments: another tenant's account is never used")

	// Card only, then bank too once ACH is active.
	_, err = c.svc.StartOnboarding(c.ctxA, c.tenantA)
	require.NoError(t, err)
	acctA := "acct_" + c.tenantA.String()[:8]
	c.activate(acctA, false)
	require.NoError(t, c.svc.SyncAccount(context.Background(), acctA))
	page, err = pay.GetPayPage(public, token)
	require.NoError(t, err)
	require.Len(t, page.Methods, 1)
	assert.Equal(t, "card", page.Methods[0].Method)
	assertDec(t, "20", page.Methods[0].BusinessFee, "0.4% of $5,000")

	c.activate(acctA, true)
	require.NoError(t, c.svc.SyncAccount(context.Background(), acctA))
	page, err = pay.GetPayPage(public, token)
	require.NoError(t, err)
	require.Len(t, page.Methods, 2)
	assert.Equal(t, "bank", page.Methods[0].Method, "bank first")
	assertDec(t, "10", page.Methods[0].BusinessFee, "1% capped at $10")
	assert.Equal(t, "card", page.Methods[1].Method)
}

// T170: nothing to pay online when the master switch is off, the link has
// expired, or the invoice is already paid — the page still shows the invoice.
func TestGetPayPage_NoMethodsWhenNotPayable(t *testing.T) {
	c := setupConnect(t, true)
	_, err := c.svc.StartOnboarding(c.ctxA, c.tenantA)
	require.NoError(t, err)
	acctA := "acct_" + c.tenantA.String()[:8]
	c.activate(acctA, true)
	require.NoError(t, c.svc.SyncAccount(context.Background(), acctA))
	inv, token := payableInvoice(t, c, 100)

	off, _ := newPaymentService(t, c, false)
	page, err := off.GetPayPage(public, token)
	require.NoError(t, err)
	assert.Empty(t, page.Methods, "master switch off")
	assert.Equal(t, inv.ID, page.Invoice.ID)

	on, _ := newPaymentService(t, c, true)
	require.NoError(t, c.db.WithContext(c.ctxA).Model(&models.Invoice{}).Where("id = ?", inv.ID).
		Update("payment_token_expires_at", time.Now().Add(-time.Hour)).Error)
	page, err = on.GetPayPage(public, token)
	require.NoError(t, err)
	assert.Empty(t, page.Methods, "expired link")

	_, err = on.GetPayPage(public, "no-such-token")
	assert.ErrorIs(t, err, ErrInvalidPaymentToken)
}

func TestBusinessLocation(t *testing.T) {
	assert.Equal(t, "Columbus, OH", businessLocation(models.TenantSettings{BusinessCity: " Columbus ", BusinessState: "OH"}))
	assert.Equal(t, "Columbus", businessLocation(models.TenantSettings{BusinessCity: "Columbus"}))
	assert.Equal(t, "OH", businessLocation(models.TenantSettings{BusinessState: "OH"}))
	assert.Empty(t, businessLocation(models.TenantSettings{}))
}
