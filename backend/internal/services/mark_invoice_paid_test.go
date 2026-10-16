package services

import (
	"testing"
	"time"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/javaknight1/servicepro/backend/internal/models"
	"github.com/javaknight1/servicepro/backend/internal/tenancy"
)

// Duplicate Stripe deliveries of the same checkout session must not credit a
// partially paid invoice twice (F-10).
func TestMarkInvoiceAsPaid_IdempotentPerSession(t *testing.T) {
	f := setupIsolation(t)
	svc := NewInvoiceService(f.db)
	now := time.Now()

	inv, err := svc.CreateInvoice(f.ctxA, &models.Invoice{
		CustomerID: f.custA.ID, InvoiceNumber: "INV-PAY-1", IssueDate: now, DueDate: now.Add(24 * time.Hour),
		Lines: oneLine(100),
	}, f.userA)
	require.NoError(t, err)

	// Webhooks run with a system context (no tenant session).
	ctx := tenancy.WithSystem(f.ctxB)
	ctx = tenancy.WithTenant(ctx, inv.TenantID)

	_, err = svc.MarkInvoiceAsPaid(ctx, inv.ID, decimal.NewFromInt(40), "cs_1", "pi_1")
	require.NoError(t, err)
	got, err := svc.MarkInvoiceAsPaid(ctx, inv.ID, decimal.NewFromInt(40), "cs_1", "pi_1")
	require.NoError(t, err)
	assert.True(t, got.AmountPaid.Equal(decimal.NewFromInt(40)), "replayed session not credited twice: %s", got.AmountPaid)
	assert.Equal(t, models.InvoiceStatusPartiallyPaid, got.Status)

	got, err = svc.MarkInvoiceAsPaid(ctx, inv.ID, decimal.NewFromInt(60), "cs_2", "pi_2")
	require.NoError(t, err)
	assert.True(t, got.AmountPaid.Equal(decimal.NewFromInt(100)))
	assert.Equal(t, models.InvoiceStatusPaid, got.Status)
}
