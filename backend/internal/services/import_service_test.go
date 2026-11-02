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
	"github.com/javaknight1/servicepro/backend/internal/repository"
	"github.com/javaknight1/servicepro/backend/internal/tenancy"
)

// importFixture is a migrated test database (openProvisioningTestDB) with
// the import service, skipped when import_jobs.payload (migration 000009) is
// not there yet.
type importFixture struct {
	db  *gorm.DB
	svc *ImportService
	tag string
}

type importTenant struct {
	id     uuid.UUID
	userID uuid.UUID
	ctx    context.Context
}

func newImportFixture(t *testing.T) *importFixture {
	t.Helper()
	db := openProvisioningTestDB(t)
	var n int64
	require.NoError(t, db.Raw(`SELECT count(*) FROM information_schema.columns
		WHERE table_name = 'import_jobs' AND column_name = 'payload'`).Scan(&n).Error)
	if n == 0 {
		t.Skip("Skipping - test database is not migrated to 000009 (import_jobs.payload)")
	}
	return &importFixture{db: db, svc: NewImportService(db, repository.NewCustomerRepository(db)), tag: uuid.NewString()[:8]}
}

// tenant inserts a tenant and its owner; the tenant delete cascades to its
// customers and import jobs.
func (f *importFixture) tenant(t *testing.T) importTenant {
	t.Helper()
	tt := importTenant{id: uuid.New(), userID: uuid.New()}
	short := tt.id.String()[:8]
	require.NoError(t, f.db.Exec(`INSERT INTO users (id, email, password_hash) VALUES (?, ?, 'x')`,
		tt.userID, "import-owner-"+short+"-"+f.tag+"@example.com").Error)
	require.NoError(t, f.db.Exec(`INSERT INTO tenants (id, name, slug, owner_id) VALUES (?, ?, ?, ?)`,
		tt.id, "Import "+short, "imp-"+short, tt.userID).Error)
	t.Cleanup(func() {
		_ = f.db.Exec(`DELETE FROM tenants WHERE id = ?`, tt.id).Error
		_ = f.db.Exec(`DELETE FROM users WHERE id = ?`, tt.userID).Error
	})
	tt.ctx = tenancy.WithTenant(context.Background(), tt.id)
	return tt
}

func (f *importFixture) row(name string) models.CSVImportRow {
	return models.CSVImportRow{
		FirstName: name, LastName: "Imported", Email: name + "-" + f.tag + "@example.com",
		PhonePrimary: "555-123-4567", BillingAddressStreet: "1 Main St", BillingAddressCity: "Austin",
		BillingAddressState: "TX", BillingAddressZip: "78701", CustomerType: "commercial",
	}
}

func (f *importFixture) customers(t *testing.T, tenantID uuid.UUID) []models.Customer {
	t.Helper()
	var out []models.Customer
	require.NoError(t, f.db.Raw(`SELECT * FROM customers WHERE tenant_id = ? AND deleted_at IS NULL ORDER BY first_name`, tenantID).Scan(&out).Error)
	return out
}

func (f *importFixture) job(t *testing.T, id uuid.UUID) models.ImportJob {
	t.Helper()
	var j models.ImportJob
	require.NoError(t, f.db.Raw(`SELECT * FROM import_jobs WHERE id = ?`, id).Scan(&j).Error)
	return j
}

func TestImportService_QueuedJobRunsEndToEnd(t *testing.T) {
	f := newImportFixture(t)
	a := f.tenant(t)

	bad := f.row("Carol")
	bad.Email = "not-an-email"
	rows := []models.CSVImportRow{f.row("Alice"), f.row("Bob"), bad}
	job, err := f.svc.CreateImportJob(a.ctx, a.userID, "customers.csv", rows, "error", false)
	require.NoError(t, err)
	assert.Equal(t, models.ImportJobStatusPending, job.Status)
	assert.Equal(t, a.id, job.TenantID, "the job belongs to the caller's tenant")

	res, err := f.svc.ProcessPending(context.Background(), 50, time.Minute)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, res.Claimed, 1)

	got := f.job(t, job.ID)
	assert.Equal(t, models.ImportJobStatusCompleted, got.Status)
	assert.Equal(t, 3, got.ProcessedRows)
	assert.Equal(t, 2, got.SuccessfulRows)
	assert.Equal(t, 1, got.FailedRows)
	assert.NotNil(t, got.StartedAt)
	assert.NotNil(t, got.CompletedAt)
	assert.Nil(t, got.Payload, "the payload is cleared once the job finishes")

	custs := f.customers(t, a.id)
	require.Len(t, custs, 2)
	assert.Equal(t, "Alice", custs[0].FirstName)
	assert.Equal(t, "Bob", custs[1].FirstName)

	// Per-row result for the failed row, readable through the tenant.
	errs, err := f.svc.GetImportErrors(a.ctx, job.ID, 0)
	require.NoError(t, err)
	require.Len(t, errs, 1)
	assert.Equal(t, 3, errs[0].RowNumber)
	assert.Contains(t, errs[0].Error, "email")
	assert.Contains(t, errs[0].RowData, "not-an-email")

	// A second pass finds nothing to do: the job is not imported twice.
	_, err = f.svc.ProcessPending(context.Background(), 50, time.Minute)
	require.NoError(t, err)
	assert.Len(t, f.customers(t, a.id), 2)
}

func TestImportService_DuplicateHandling(t *testing.T) {
	f := newImportFixture(t)
	a := f.tenant(t)

	first, err := f.svc.CreateImportJob(a.ctx, a.userID, "a.csv", []models.CSVImportRow{f.row("Dana")}, "error", false)
	require.NoError(t, err)
	run := func(id uuid.UUID) *models.ImportJob {
		claimed, err := f.svc.ClaimImportJob(a.ctx, id)
		require.NoError(t, err)
		require.NotNil(t, claimed)
		done, err := f.svc.RunImportJob(a.ctx, claimed)
		require.NoError(t, err)
		return done
	}
	run(first.ID)

	again := f.row("Dana")
	again.BillingAddressCity = "Dallas"
	for _, tc := range []struct {
		mode       string
		wantStatus models.ImportJobStatus
		wantCity   string
	}{
		{"error", models.ImportJobStatusFailed, "Austin"},
		{"skip", models.ImportJobStatusCompleted, "Austin"},
		{"update", models.ImportJobStatusCompleted, "Dallas"},
	} {
		job, err := f.svc.CreateImportJob(a.ctx, a.userID, "b.csv", []models.CSVImportRow{again}, tc.mode, false)
		require.NoError(t, err)
		assert.Equal(t, tc.wantStatus, run(job.ID).Status, tc.mode)
		custs := f.customers(t, a.id)
		require.Len(t, custs, 1, tc.mode)
		assert.Equal(t, tc.wantCity, custs[0].BillingAddressCity, tc.mode)
	}

	// Dry run writes nothing.
	job, err := f.svc.CreateImportJob(a.ctx, a.userID, "c.csv", []models.CSVImportRow{f.row("Eve")}, "error", true)
	require.NoError(t, err)
	assert.Equal(t, 1, run(job.ID).SuccessfulRows)
	assert.Len(t, f.customers(t, a.id), 1)
}

func TestImportService_ConcurrentClaimsRunAJobOnce(t *testing.T) {
	f := newImportFixture(t)
	a := f.tenant(t)

	job, err := f.svc.CreateImportJob(a.ctx, a.userID, "x.csv", []models.CSVImportRow{f.row("Frank")}, "error", false)
	require.NoError(t, err)

	const workers = 8
	var wg sync.WaitGroup
	var mu sync.Mutex
	var winners []*models.ImportJob
	for range workers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			c, err := f.svc.ClaimImportJob(a.ctx, job.ID)
			assert.NoError(t, err)
			if c != nil {
				mu.Lock()
				winners = append(winners, c)
				mu.Unlock()
			}
		}()
	}
	wg.Wait()
	require.Len(t, winners, 1, "exactly one claimant wins")
	assert.Equal(t, models.ImportJobStatusProcessing, winners[0].Status)

	// The global runner cannot take it either while it is processing.
	_, err = f.svc.ProcessPending(context.Background(), 50, time.Minute)
	require.NoError(t, err)
	assert.Equal(t, models.ImportJobStatusProcessing, f.job(t, job.ID).Status)
	assert.Empty(t, f.customers(t, a.id))

	_, err = f.svc.RunImportJob(a.ctx, winners[0])
	require.NoError(t, err)
	assert.Len(t, f.customers(t, a.id), 1)
}

func TestImportService_ConcurrentRunnersSplitJobs(t *testing.T) {
	f := newImportFixture(t)
	a := f.tenant(t)

	const n = 6
	ids := make([]uuid.UUID, n)
	for i := range n {
		job, err := f.svc.CreateImportJob(a.ctx, a.userID, "p.csv", []models.CSVImportRow{f.row(fmt.Sprintf("P%d", i))}, "error", false)
		require.NoError(t, err)
		ids[i] = job.ID
	}

	var wg sync.WaitGroup
	for range 4 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := f.svc.ProcessPending(context.Background(), 50, time.Minute)
			assert.NoError(t, err)
		}()
	}
	wg.Wait()

	for _, id := range ids {
		assert.Equal(t, models.ImportJobStatusCompleted, f.job(t, id).Status)
	}
	assert.Len(t, f.customers(t, a.id), n, "every row imported exactly once")
}

func TestImportService_CrossTenant(t *testing.T) {
	f := newImportFixture(t)
	a, b := f.tenant(t), f.tenant(t)

	// Tenant B already has a customer with the email tenant A imports.
	shared := f.row("Grace")
	seed, err := f.svc.CreateImportJob(b.ctx, b.userID, "b.csv", []models.CSVImportRow{shared}, "error", false)
	require.NoError(t, err)
	claimed, err := f.svc.ClaimImportJob(b.ctx, seed.ID)
	require.NoError(t, err)
	_, err = f.svc.RunImportJob(b.ctx, claimed)
	require.NoError(t, err)

	changed := shared
	changed.BillingAddressCity = "Hijacked"
	job, err := f.svc.CreateImportJob(a.ctx, a.userID, "a.csv", []models.CSVImportRow{changed}, "update", false)
	require.NoError(t, err)

	// B can neither claim, read, nor list errors of A's job.
	c, err := f.svc.ClaimImportJob(b.ctx, job.ID)
	require.NoError(t, err)
	assert.Nil(t, c)
	_, err = f.svc.GetImportJob(b.ctx, job.ID)
	assert.Error(t, err)
	_, err = f.svc.GetImportErrors(b.ctx, job.ID, 0)
	assert.Error(t, err)

	// The runner imports A's job into A only, whatever context it runs in.
	c, err = f.svc.ClaimImportJob(a.ctx, job.ID)
	require.NoError(t, err)
	require.NotNil(t, c)
	done, err := f.svc.RunImportJob(b.ctx, c)
	require.NoError(t, err)
	assert.Equal(t, models.ImportJobStatusCompleted, done.Status)

	bCusts := f.customers(t, b.id)
	require.Len(t, bCusts, 1)
	assert.Equal(t, "Austin", bCusts[0].BillingAddressCity, "tenant B's customer is untouched")
	aCusts := f.customers(t, a.id)
	require.Len(t, aCusts, 1, "the row became a new customer in tenant A")
	assert.Equal(t, "Hijacked", aCusts[0].BillingAddressCity)
}

func TestImportService_StaleProcessingJobFails(t *testing.T) {
	f := newImportFixture(t)
	a := f.tenant(t)

	job, err := f.svc.CreateImportJob(a.ctx, a.userID, "s.csv", []models.CSVImportRow{f.row("Hank")}, "error", false)
	require.NoError(t, err)
	claimed, err := f.svc.ClaimImportJob(a.ctx, job.ID)
	require.NoError(t, err)
	require.NotNil(t, claimed)
	old := time.Now().Add(-time.Hour)
	require.NoError(t, f.db.Exec(`UPDATE import_jobs SET updated_at = ? WHERE id = ?`, old, job.ID).Error)

	res, err := f.svc.ProcessPending(context.Background(), 50, time.Minute)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, res.Interrupted, int64(1))
	got := f.job(t, job.ID)
	assert.Equal(t, models.ImportJobStatusFailed, got.Status)
	assert.Nil(t, got.Payload)
	require.NotNil(t, got.ErrorSummary)
	assert.Contains(t, *got.ErrorSummary, "interrupted")
	assert.Empty(t, f.customers(t, a.id), "an interrupted job is not re-run")

	// The original runner finishing late must not overwrite the failure.
	_, err = f.svc.RunImportJob(a.ctx, claimed)
	require.NoError(t, err)
	assert.Equal(t, models.ImportJobStatusFailed, f.job(t, job.ID).Status)
}
