package services

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/javaknight1/servicepro/backend/internal/models"
	"github.com/javaknight1/servicepro/backend/internal/tenancy"
)

// legacyTestCtx is the tenant for SQLite fixtures that don't install the
// tenancy plugin: document numbers are allocated per tenant.
var legacyTestCtx = tenancy.WithTenant(context.Background(), uuid.MustParse("00000000-0000-0000-0000-00000000a11c"))

// createTenantCountersTable mirrors migration 000008 for SQLite fixtures.
func createTenantCountersTable(t *testing.T, db *gorm.DB) {
	t.Helper()
	require.NoError(t, db.Exec(`CREATE TABLE IF NOT EXISTS tenant_counters (
		tenant_id TEXT NOT NULL, kind TEXT NOT NULL, next_value INTEGER NOT NULL DEFAULT 1,
		PRIMARY KEY (tenant_id, kind))`).Error)
}

func createUnnumberedInvoice(t *testing.T, svc *InvoiceService, ctx context.Context, customerID, userID uuid.UUID) *models.Invoice {
	t.Helper()
	now := time.Now()
	inv, err := svc.CreateInvoice(ctx, &models.Invoice{
		CustomerID: customerID, IssueDate: now, DueDate: now.Add(24 * time.Hour), Lines: oneLine(10),
	}, userID)
	require.NoError(t, err)
	return inv
}

// T120 / audit 7.C3: deleting the newest invoice used to make MAX()+1 hand
// out the deleted (still unique-constrained) number again, so every later
// create failed. Numbers are per tenant and never reused.
func TestInvoiceNumbering_SurvivesSoftDeleteAndIsPerTenant(t *testing.T) {
	f := setupIsolation(t)
	svc := NewInvoiceService(f.db)

	a1 := createUnnumberedInvoice(t, svc, f.ctxA, f.custA.ID, f.userA)
	a2 := createUnnumberedInvoice(t, svc, f.ctxA, f.custA.ID, f.userA)
	assert.Equal(t, "INV-000001", a1.InvoiceNumber)
	assert.Equal(t, "INV-000002", a2.InvoiceNumber)

	require.NoError(t, svc.DeleteInvoice(f.ctxA, a2.ID))
	a3 := createUnnumberedInvoice(t, svc, f.ctxA, f.custA.ID, f.userA)
	assert.Equal(t, "INV-000003", a3.InvoiceNumber, "a deleted number is not reused")

	b1 := createUnnumberedInvoice(t, svc, f.ctxB, f.custB.ID, f.userA)
	assert.Equal(t, "INV-000001", b1.InvoiceNumber, "tenant B has its own sequence")
}

// Numbers already used by an older scheme are skipped, not collided with.
func TestInvoiceNumbering_SkipsTakenNumbers(t *testing.T) {
	f := setupIsolation(t)
	svc := NewInvoiceService(f.db)
	now := time.Now()
	_, err := svc.CreateInvoice(f.ctxA, &models.Invoice{CustomerID: f.custA.ID, InvoiceNumber: "INV-000001",
		IssueDate: now, DueDate: now.Add(time.Hour), Lines: oneLine(1)}, f.userA)
	require.NoError(t, err)
	inv := createUnnumberedInvoice(t, svc, f.ctxA, f.custA.ID, f.userA)
	assert.Equal(t, "INV-000002", inv.InvoiceNumber)
}

// Audit 7.C4: job numbering used COUNT(today)+1 through the soft-delete
// scope, so after any deletion job creation failed until midnight.
func TestJobNumbering_SurvivesSoftDelete(t *testing.T) {
	f := setupIsolation(t)
	mk := func() *models.Job {
		job := &models.Job{ID: uuid.New(), CustomerID: f.custA.ID, Title: "Job", JobType: models.JobTypeRepair,
			Status: models.JobStatusNew, Priority: models.JobPriorityNormal, CreatedBy: f.userA}
		require.NoError(t, f.db.WithContext(f.ctxA).Create(job).Error)
		return job
	}
	j1, j2 := mk(), mk()
	assert.Equal(t, "JOB-000001", j1.JobNumber)
	assert.Equal(t, "JOB-000002", j2.JobNumber)
	require.NoError(t, f.db.WithContext(f.ctxA).Delete(&models.Job{}, "id = ?", j2.ID).Error)
	assert.Equal(t, "JOB-000003", mk().JobNumber)
}

// T127: numbers use the tenant's configured prefixes; another tenant's
// prefix never applies, and unset prefixes fall back to INV-/QUO-/JOB-.
func TestDocumentNumbering_UsesTenantPrefixes(t *testing.T) {
	f := setupIsolation(t)
	f.setTenantSettings(t, f.ctxA, models.TenantSettings{InvoicePrefix: " ACME-", QuotePrefix: "Q", JobPrefix: "WO-"})
	f.setTenantSettings(t, f.ctxB, models.TenantSettings{Timezone: "UTC"})
	svc := NewInvoiceService(f.db)

	assert.Equal(t, "ACME-000001", createUnnumberedInvoice(t, svc, f.ctxA, f.custA.ID, f.userA).InvoiceNumber)
	assert.Equal(t, "INV-000001", createUnnumberedInvoice(t, svc, f.ctxB, f.custB.ID, f.userA).InvoiceNumber,
		"tenant A's prefix does not leak into tenant B")

	q := &models.Quote{ID: uuid.New(), CustomerID: f.custA.ID, Status: models.QuoteStatusDraft,
		ValidUntil: time.Now().Add(72 * time.Hour), CreatedBy: f.userA}
	require.NoError(t, f.db.WithContext(f.ctxA).Create(q).Error)
	assert.Equal(t, "Q000001", q.QuoteNumber)

	job := &models.Job{ID: uuid.New(), CustomerID: f.custA.ID, Title: "Job", JobType: models.JobTypeRepair,
		Status: models.JobStatusNew, Priority: models.JobPriorityNormal, CreatedBy: f.userA}
	require.NoError(t, f.db.WithContext(f.ctxA).Create(job).Error)
	assert.Equal(t, "WO-000001", job.JobNumber)
}

func TestCounterPrefix_FallsBackOnUnusablePrefix(t *testing.T) {
	assert.Equal(t, "INV-", models.CounterPrefix(models.TenantSettings{}, models.CounterInvoice))
	assert.Equal(t, "INV-", models.CounterPrefix(models.TenantSettings{InvoicePrefix: "   "}, models.CounterInvoice))
	assert.Equal(t, "INV-", models.CounterPrefix(models.TenantSettings{InvoicePrefix: "A\nB"}, models.CounterInvoice))
	assert.Equal(t, "INV-", models.CounterPrefix(models.TenantSettings{InvoicePrefix: "THIS-PREFIX-IS-WAY-TOO-LONG-"}, models.CounterInvoice))
	assert.Equal(t, "QUO-", models.CounterPrefix(models.TenantSettings{InvoicePrefix: "X-"}, models.CounterQuote))
	assert.Equal(t, "JOB-", models.CounterPrefix(models.TenantSettings{}, models.CounterJob))
}

func TestQuoteNumbering_UsesTenantCounter(t *testing.T) {
	f := setupIsolation(t)
	q := &models.Quote{ID: uuid.New(), CustomerID: f.custA.ID, Status: models.QuoteStatusDraft,
		ValidUntil: time.Now().Add(72 * time.Hour), CreatedBy: f.userA}
	require.NoError(t, f.db.WithContext(f.ctxA).Create(q).Error)
	assert.Equal(t, "QUO-000001", q.QuoteNumber)
}

// Concurrent creates in one tenant get distinct, gap-free numbers.
func TestPostgres_InvoiceNumbering_Concurrent(t *testing.T) {
	db := setupPostgres(t)
	tn := seedPGTenant(t, db)
	svc := NewInvoiceService(db)

	const workers = 10
	numbers := make([]string, workers)
	errs := make([]error, workers)
	var wg sync.WaitGroup
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			now := time.Now()
			inv, err := svc.CreateInvoice(tn.ctx, &models.Invoice{CustomerID: tn.customerID, IssueDate: now,
				DueDate: now.Add(time.Hour), Lines: oneLine(5)}, tn.userID)
			errs[i] = err
			if err == nil {
				numbers[i] = inv.InvoiceNumber
			}
		}(i)
	}
	wg.Wait()

	seen := map[string]bool{}
	for i := 0; i < workers; i++ {
		require.NoError(t, errs[i])
		seen[numbers[i]] = true
	}
	for n := 1; n <= workers; n++ {
		assert.True(t, seen[fmt.Sprintf("INV-%06d", n)], "INV-%06d allocated", n)
	}

	// Soft-delete the newest, and creation still works.
	var newest models.Invoice
	require.NoError(t, db.WithContext(tn.ctx).Order("invoice_number DESC").First(&newest).Error)
	require.NoError(t, svc.DeleteInvoice(tn.ctx, newest.ID))
	now := time.Now()
	inv, err := svc.CreateInvoice(tn.ctx, &models.Invoice{CustomerID: tn.customerID, IssueDate: now,
		DueDate: now.Add(time.Hour), Lines: oneLine(5)}, tn.userID)
	require.NoError(t, err)
	assert.Equal(t, fmt.Sprintf("INV-%06d", workers+1), inv.InvoiceNumber)
}
