package services

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/javaknight1/servicepro/backend/internal/models"
	"github.com/javaknight1/servicepro/backend/internal/tenancy"
)

// T188: the monthly run creates one draft per consolidated-billing customer
// on the 1st in the tenant's time zone, however often it runs.
func TestPostgres_GenerateConsolidatedInvoices(t *testing.T) {
	db := setupPostgres(t)
	f := seedConsolidation(t, db)
	sys := tenancy.WithSystem(context.Background())

	// A second consolidated customer with no work, and a tenant (UTC) whose
	// customer bills per job.
	idle := uuid.New()
	require.NoError(t, db.WithContext(sys).Exec(
		`INSERT INTO customers (id, tenant_id, first_name, last_name, email, phone_primary,
			billing_address_street, billing_address_city, billing_address_state, billing_address_zip, consolidated_billing)
		 VALUES (?, ?, 'Idle', 'Owner', ?, '555-555-5555', '1 Main St', 'Springfield', 'IL', '62701', true)`,
		idle, f.p.tenantID, "idle-"+idle.String()[:8]+"@example.com").Error)
	perJob := seedPGTenant(t, db)

	// 23:00 on Oct 31 in New York: not the 1st yet.
	res, err := f.svc.GenerateConsolidatedInvoices(context.Background(), time.Date(2026, 11, 1, 3, 0, 0, 0, time.UTC))
	require.NoError(t, err)
	assert.Equal(t, 0, res.Customers)

	// 01:00 on Nov 1 in New York, eight runs at once.
	now := time.Date(2026, 11, 1, 5, 0, 0, 0, time.UTC)
	var wg sync.WaitGroup
	var mu sync.Mutex
	total := ConsolidatedInvoicesResult{}
	for range 8 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			r, err := f.svc.GenerateConsolidatedInvoices(context.Background(), now)
			assert.NoError(t, err)
			if r != nil {
				mu.Lock()
				total.Created += r.Created
				total.Empty += r.Empty
				total.Skipped += r.Skipped
				total.Failed += r.Failed
				mu.Unlock()
			}
		}()
	}
	wg.Wait()
	assert.Equal(t, 1, total.Created)
	assert.Equal(t, 1, total.Empty)
	assert.Equal(t, 14, total.Skipped)
	assert.Zero(t, total.Failed)

	var invoices []models.Invoice
	require.NoError(t, db.WithContext(f.p.ctx).Preload("Lines").Find(&invoices).Error)
	require.Len(t, invoices, 1)
	inv := invoices[0]
	assert.Equal(t, models.InvoiceStatusDraft, inv.Status, "drafts are not sent")
	assert.Equal(t, f.p.customerID, inv.CustomerID)
	assert.Equal(t, "2026-10-01", inv.BillingPeriodStart.Format(models.DateLayout))
	assert.Equal(t, "2026-10-31", inv.BillingPeriodEnd.Format(models.DateLayout))
	assert.Equal(t, "3450.00", inv.TotalAmount.StringFixed(2))
	assert.Equal(t, "7731-HV", inv.PONumber)

	var runs []models.ConsolidatedInvoiceRun
	require.NoError(t, db.WithContext(f.p.ctx).Order("invoice_id NULLS LAST").Find(&runs).Error)
	require.Len(t, runs, 2)
	require.NotNil(t, runs[0].InvoiceID)
	assert.Equal(t, inv.ID, *runs[0].InvoiceID)
	assert.Nil(t, runs[1].InvoiceID, "no work: claimed without an invoice")

	var other int64
	require.NoError(t, db.WithContext(perJob.ctx).Model(&models.Invoice{}).Count(&other).Error)
	assert.Zero(t, other, "per-job customers are left alone")

	// Later the same day: nothing new.
	res, err = f.svc.GenerateConsolidatedInvoices(context.Background(), now.Add(5*time.Hour))
	require.NoError(t, err)
	assert.Equal(t, 0, res.Created)
	assert.Equal(t, 2, res.Skipped)
}
