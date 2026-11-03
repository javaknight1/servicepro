package services

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/javaknight1/servicepro/backend/internal/models"
)

// Nothing moved invoices to overdue. The invoice-overdue job marks sent and
// viewed invoices due before today as overdue, for every tenant, idempotently.
func TestMarkOverdueInvoices(t *testing.T) {
	f := setupIsolation(t)
	now := time.Date(2026, 11, 3, 15, 0, 0, 0, time.UTC)
	yesterday, today := now.AddDate(0, 0, -1), now.Truncate(24*time.Hour)

	mk := func(ctx context.Context, customerID uuid.UUID, status models.InvoiceStatus, due time.Time) uuid.UUID {
		inv := &models.Invoice{CustomerID: customerID, IssueDate: due.AddDate(0, 0, -30), DueDate: due,
			Status: models.InvoiceStatusDraft, TotalAmount: decimal.NewFromInt(100), CreatedBy: f.userA}
		require.NoError(t, f.db.WithContext(ctx).Create(inv).Error)
		require.NoError(t, f.db.WithContext(ctx).Exec("UPDATE invoices SET status = ? WHERE id = ?", status, inv.ID).Error)
		return inv.ID
	}
	sentA := mk(f.ctxA, f.custA.ID, models.InvoiceStatusSent, yesterday)
	viewedB := mk(f.ctxB, f.custB.ID, models.InvoiceStatusViewed, yesterday)
	dueToday := mk(f.ctxA, f.custA.ID, models.InvoiceStatusSent, today)
	draft := mk(f.ctxA, f.custA.ID, models.InvoiceStatusDraft, yesterday)
	partly := mk(f.ctxA, f.custA.ID, models.InvoiceStatusPartiallyPaid, yesterday)
	paid := mk(f.ctxA, f.custA.ID, models.InvoiceStatusPaid, yesterday)

	n, err := MarkOverdueInvoices(context.Background(), f.db, now)
	require.NoError(t, err)
	assert.Equal(t, int64(2), n)

	status := func(ctx context.Context, id uuid.UUID) models.InvoiceStatus {
		var inv models.Invoice
		require.NoError(t, f.db.WithContext(ctx).First(&inv, "id = ?", id).Error)
		return inv.Status
	}
	assert.Equal(t, models.InvoiceStatusOverdue, status(f.ctxA, sentA))
	assert.Equal(t, models.InvoiceStatusOverdue, status(f.ctxB, viewedB))
	assert.Equal(t, models.InvoiceStatusSent, status(f.ctxA, dueToday), "due today is not late yet")
	assert.Equal(t, models.InvoiceStatusDraft, status(f.ctxA, draft))
	assert.Equal(t, models.InvoiceStatusPartiallyPaid, status(f.ctxA, partly))
	assert.Equal(t, models.InvoiceStatusPaid, status(f.ctxA, paid))

	n, err = MarkOverdueInvoices(context.Background(), f.db, now)
	require.NoError(t, err)
	assert.Zero(t, n, "re-running changes nothing")
}
