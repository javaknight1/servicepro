package services

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/javaknight1/servicepro/backend/internal/models"
)

type recordedEnqueue struct {
	tenant  uuid.UUID
	entity  string
	localID uuid.UUID
}

type accountingRecorder struct {
	mu  sync.Mutex
	got []recordedEnqueue
}

func (a *accountingRecorder) EnqueueAfterCommit(_ context.Context, tenantID uuid.UUID, entity string, localID uuid.UUID) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.got = append(a.got, recordedEnqueue{tenantID, entity, localID})
}

// T192: recorded payments (manual and Stripe) are queued for the tenant's
// accounting sync after their transaction commits; a refused payment isn't.
func TestPostgresInvoicePaymentsQueueAccountingSync(t *testing.T) {
	db := setupPostgres(t)
	p := seedPGTenant(t, db)
	svc := NewInvoiceService(db)
	rec := &accountingRecorder{}
	svc.SetAccountingSync(rec)

	invoiceID := uuid.New()
	require.NoError(t, db.WithContext(p.ctx).Exec(
		`INSERT INTO invoices (id, tenant_id, customer_id, invoice_number, issue_date, due_date, status,
			subtotal, total_amount, amount_paid, created_by)
		 VALUES (?, ?, ?, ?, CURRENT_DATE, CURRENT_DATE, 'sent', 100, 100, 0, ?)`,
		invoiceID, p.tenantID, p.customerID, "ACC-"+p.tenantID.String()[:8], p.userID).Error)

	pay, err := svc.RecordPayment(p.ctx, &models.InvoicePayment{InvoiceID: invoiceID, Amount: decimal.NewFromInt(40), PaymentDate: time.Now()})
	require.NoError(t, err)
	_, err = svc.RecordPayment(p.ctx, &models.InvoicePayment{InvoiceID: invoiceID, Amount: decimal.NewFromInt(1000)})
	require.ErrorIs(t, err, ErrOverpayment)

	_, err = svc.MarkInvoiceAsPaid(p.ctx, invoiceID, decimal.NewFromInt(60), "cs_test_1", "pi_test_1")
	require.NoError(t, err)
	_, err = svc.MarkInvoiceAsPaid(p.ctx, invoiceID, decimal.NewFromInt(60), "cs_test_1", "pi_test_1")
	require.NoError(t, err, "duplicate webhook delivery")

	require.Len(t, rec.got, 2, "one per payment actually recorded")
	assert.Equal(t, recordedEnqueue{p.tenantID, "payment", pay.ID}, rec.got[0])
	assert.Equal(t, p.tenantID, rec.got[1].tenant)
	assert.Equal(t, "payment", rec.got[1].entity)
	assert.NotEqual(t, pay.ID, rec.got[1].localID)
}
