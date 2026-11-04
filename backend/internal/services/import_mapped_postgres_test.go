package services

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/javaknight1/servicepro/backend/internal/imports"
	"github.com/javaknight1/servicepro/backend/internal/models"
	"github.com/javaknight1/servicepro/backend/internal/repository"
	"github.com/javaknight1/servicepro/backend/internal/tenancy"
)

// Mapped imports (T193) against a freshly migrated database
// ($SERVICEPRO_TEST_PG_URL; skipped when unset).

func mappedImportSvc(db *gorm.DB) *ImportService {
	return NewImportService(db, repository.NewCustomerRepository(db))
}

// readFixture maps one internal/imports testdata file automatically.
func readFixture(t *testing.T, file string, src imports.Source, e imports.Entity) []imports.Record {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("..", "imports", "testdata", file))
	require.NoError(t, err)
	tbl, err := imports.ReadTable(data, file)
	require.NoError(t, err)
	p, err := imports.Prepare(tbl, src, e, nil)
	require.NoError(t, err)
	require.Empty(t, p.Missing)
	return p.Records
}

// runMapped queues records for the tenant and runs the job to the end.
func runMapped(t *testing.T, svc *ImportService, p *pgTenant, src imports.Source, e imports.Entity, recs []imports.Record) *models.ImportJob {
	t.Helper()
	job, err := svc.CreateMappedImportJob(p.ctx, p.userID, string(e)+".csv", src, e, recs)
	require.NoError(t, err)
	claimed, err := svc.ClaimImportJob(p.ctx, job.ID)
	require.NoError(t, err)
	require.NotNil(t, claimed)
	done, err := svc.RunImportJob(p.ctx, claimed)
	require.NoError(t, err)
	return done
}

func count(t *testing.T, db *gorm.DB, table string, tenantID uuid.UUID) int64 {
	t.Helper()
	var n int64
	require.NoError(t, db.WithContext(tenancy.WithSystem(context.Background())).
		Raw(`SELECT count(*) FROM `+table+` WHERE tenant_id = ? AND deleted_at IS NULL`, tenantID).Scan(&n).Error)
	return n
}

func assertCounts(t *testing.T, job *models.ImportJob, ok, skipped, failed int) {
	t.Helper()
	assert.Equal(t, ok, job.SuccessfulRows, "successful")
	assert.Equal(t, skipped, job.SkippedRows, "skipped")
	assert.Equal(t, failed, job.FailedRows, "failed")
	assert.Equal(t, ok+skipped+failed, job.ProcessedRows, "processed")
}

func TestImportMappedPostgres_JobberEndToEndAndIdempotent(t *testing.T) {
	db := setupPostgres(t)
	svc := mappedImportSvc(db)
	a := seedPGTenant(t, db)

	// Clients (+ their service property as the first site).
	clients := readFixture(t, "jobber_clients.csv", imports.SourceJobber, imports.EntityCustomers)
	job := runMapped(t, svc, a, imports.SourceJobber, imports.EntityCustomers, clients)
	assert.Equal(t, models.ImportJobStatusCompleted, job.Status)
	assertCounts(t, job, 4, 0, 1)
	assert.EqualValues(t, 5, count(t, db, "customers", a.tenantID), "4 imported + the seeded one")
	assert.EqualValues(t, 2, count(t, db, "service_locations", a.tenantID), "Riverside Tower and Harbor Bldg A")

	errs, err := svc.GetImportErrors(a.ctx, job.ID, 0)
	require.NoError(t, err)
	require.Len(t, errs, 1)
	assert.Equal(t, 5, errs[0].RowNumber)
	assert.Equal(t, "email", errs[0].Field)

	var riverside models.Customer
	require.NoError(t, db.WithContext(a.ctx).Where("email = ?", "ap@riversidepm.example").First(&riverside).Error)
	assert.Equal(t, "Dana", riverside.FirstName)
	assert.Equal(t, "400 Congress Ave, Suite 210", riverside.BillingAddressStreet)
	assert.Equal(t, models.CustomerStatusActive, riverside.Status)

	// The same file again: nothing new.
	again := runMapped(t, svc, a, imports.SourceJobber, imports.EntityCustomers, readFixture(t, "jobber_clients.csv", imports.SourceJobber, imports.EntityCustomers))
	assertCounts(t, again, 0, 4, 1)
	assert.EqualValues(t, 5, count(t, db, "customers", a.tenantID))
	assert.EqualValues(t, 2, count(t, db, "service_locations", a.tenantID))

	// Properties: one already there, two new, one for an unknown client.
	props := runMapped(t, svc, a, imports.SourceJobber, imports.EntitySites, readFixture(t, "jobber_properties.csv", imports.SourceJobber, imports.EntitySites))
	assertCounts(t, props, 2, 1, 1)
	assert.EqualValues(t, 4, count(t, db, "service_locations", a.tenantID))
	errs, err = svc.GetImportErrors(a.ctx, props.ID, 0)
	require.NoError(t, err)
	require.Len(t, errs, 1)
	assert.Contains(t, errs[0].Error, "import customers first")

	// Jobs.
	jobs := runMapped(t, svc, a, imports.SourceJobber, imports.EntityJobs, readFixture(t, "jobber_jobs.csv", imports.SourceJobber, imports.EntityJobs))
	assertCounts(t, jobs, 3, 0, 1)
	var j101 models.Job
	require.NoError(t, db.WithContext(a.ctx).Where("job_number = ?", "101").First(&j101).Error)
	assert.Equal(t, riverside.ID, j101.CustomerID)
	assert.Equal(t, models.JobStatusCompleted, j101.Status)
	require.NotNil(t, j101.SiteID)
	var tower models.Site
	require.NoError(t, db.WithContext(a.ctx).First(&tower, "id = ?", *j101.SiteID).Error)
	assert.Equal(t, "Riverside Tower", tower.Name, "matched to the client's property by street")
	require.NotNil(t, j101.ScheduledStartAt)
	assert.Equal(t, time.Date(2024, 3, 4, 8, 0, 0, 0, time.UTC).Unix(), *j101.ScheduledStartAt, "date-only visits start at 08:00 tenant time (UTC here)")
	jobsAgain := runMapped(t, svc, a, imports.SourceJobber, imports.EntityJobs, readFixture(t, "jobber_jobs.csv", imports.SourceJobber, imports.EntityJobs))
	assertCounts(t, jobsAgain, 0, 3, 1)
	assert.EqualValues(t, 3, count(t, db, "jobs", a.tenantID))

	// Open invoices: the balance only, as a sent invoice.
	invs := runMapped(t, svc, a, imports.SourceJobber, imports.EntityInvoices, readFixture(t, "jobber_invoices.csv", imports.SourceJobber, imports.EntityInvoices))
	assertCounts(t, invs, 2, 2, 1)
	var partial models.Invoice
	require.NoError(t, db.WithContext(a.ctx).Preload("Lines").Where("invoice_number = ?", "1003").First(&partial).Error)
	assert.Equal(t, models.InvoiceStatusSent, partial.Status)
	assert.True(t, partial.TotalAmount.Equal(decimal.RequireFromString("850.00")), partial.TotalAmount.String())
	assert.True(t, partial.AmountDue.Equal(decimal.RequireFromString("850.00")), partial.AmountDue.String())
	assert.True(t, partial.AmountPaid.IsZero())
	assert.Equal(t, "2024-06-07", partial.DueDate.Format("2006-01-02"))
	require.Len(t, partial.Lines, 1)
	assert.Contains(t, partial.Lines[0].Description, "Jobber invoice 1003")
	assert.Contains(t, partial.Notes, "Original total 1850.00")
	invsAgain := runMapped(t, svc, a, imports.SourceJobber, imports.EntityInvoices, readFixture(t, "jobber_invoices.csv", imports.SourceJobber, imports.EntityInvoices))
	assertCounts(t, invsAgain, 0, 4, 1)
	assert.EqualValues(t, 2, count(t, db, "invoices", a.tenantID))

	// A record deleted after its import is imported again.
	require.NoError(t, db.WithContext(a.ctx).Delete(&models.Job{}, "id = ?", j101.ID).Error)
	jobsThird := runMapped(t, svc, a, imports.SourceJobber, imports.EntityJobs, readFixture(t, "jobber_jobs.csv", imports.SourceJobber, imports.EntityJobs))
	assertCounts(t, jobsThird, 1, 2, 1)
}

func TestImportMappedPostgres_HousecallPro(t *testing.T) {
	db := setupPostgres(t)
	svc := mappedImportSvc(db)
	a := seedPGTenant(t, db)

	cust := runMapped(t, svc, a, imports.SourceHousecallPro, imports.EntityCustomers, readFixture(t, "hcp_customers.csv", imports.SourceHousecallPro, imports.EntityCustomers))
	assertCounts(t, cust, 2, 0, 1)

	jobs := runMapped(t, svc, a, imports.SourceHousecallPro, imports.EntityJobs, readFixture(t, "hcp_jobs.csv", imports.SourceHousecallPro, imports.EntityJobs))
	assertCounts(t, jobs, 3, 0, 1) // the canceled job names no known customer
	var j models.Job
	require.NoError(t, db.WithContext(a.ctx).Where("job_number = ?", "2041").First(&j).Error)
	require.NotNil(t, j.ScheduledEndAt)
	assert.Equal(t, 150, j.EstimatedDuration, "9:00 to 11:30")
	assert.Equal(t, "600 Commerce St, Floor 3", j.ServiceAddress.Street)

	invs := runMapped(t, svc, a, imports.SourceHousecallPro, imports.EntityInvoices, readFixture(t, "hcp_jobs.csv", imports.SourceHousecallPro, imports.EntityInvoices))
	assertCounts(t, invs, 1, 3, 0)
	assert.EqualValues(t, 1, count(t, db, "invoices", a.tenantID))
}

func TestImportMappedPostgres_CrossTenant(t *testing.T) {
	db := setupPostgres(t)
	svc := mappedImportSvc(db)
	a, b := seedPGTenant(t, db), seedPGTenant(t, db)

	// Tenant B already has Riverside (same email) and imports first.
	clients := readFixture(t, "jobber_clients.csv", imports.SourceJobber, imports.EntityCustomers)
	runMapped(t, svc, b, imports.SourceJobber, imports.EntityCustomers, clients[:1])
	bCustomers := count(t, db, "customers", b.tenantID)

	// Tenant A importing the same J-IDs and emails creates its own rows:
	// B's refs and customers are invisible to it.
	jobA := runMapped(t, svc, a, imports.SourceJobber, imports.EntityCustomers, readFixture(t, "jobber_clients.csv", imports.SourceJobber, imports.EntityCustomers))
	assertCounts(t, jobA, 4, 0, 1)
	assert.Equal(t, bCustomers, count(t, db, "customers", b.tenantID), "tenant B untouched")

	// A's jobs attach to A's Riverside, never B's.
	runMapped(t, svc, a, imports.SourceJobber, imports.EntityJobs, readFixture(t, "jobber_jobs.csv", imports.SourceJobber, imports.EntityJobs))
	var aRiverside, bRiverside models.Customer
	require.NoError(t, db.WithContext(a.ctx).Where("email = ?", "ap@riversidepm.example").First(&aRiverside).Error)
	require.NoError(t, db.WithContext(b.ctx).Where("email = ?", "ap@riversidepm.example").First(&bRiverside).Error)
	require.NotEqual(t, aRiverside.ID, bRiverside.ID)
	var bJobs int64
	require.NoError(t, db.WithContext(tenancy.WithSystem(context.Background())).Model(&models.Job{}).Where("customer_id = ?", bRiverside.ID).Count(&bJobs).Error)
	assert.Zero(t, bJobs)
	assert.EqualValues(t, 0, count(t, db, "jobs", b.tenantID))

	// B can't read, claim or list errors of A's job.
	_, err := svc.GetImportJob(b.ctx, jobA.ID)
	assert.ErrorIs(t, err, ErrImportJobNotFound)
	_, err = svc.GetImportErrors(b.ctx, jobA.ID, 0)
	assert.Error(t, err)
	list, err := svc.ListImportJobs(b.ctx, nil, 50)
	require.NoError(t, err)
	for _, j := range list {
		assert.Equal(t, b.tenantID, j.TenantID)
	}

	// RLS: tenant B's context sees none of A's refs, even unfiltered.
	var refs []models.ImportExternalRef
	require.NoError(t, db.WithContext(b.ctx).Find(&refs).Error)
	for _, r := range refs {
		assert.Equal(t, b.tenantID, r.TenantID)
	}
}

func TestImportMappedPostgres_ChunksAndResume(t *testing.T) {
	db := setupPostgres(t)
	svc := mappedImportSvc(db)
	a := seedPGTenant(t, db)

	var b strings.Builder
	b.WriteString("Name,Email,Phone,Street,City,State,Zip\n")
	for i := range 60 {
		fmt.Fprintf(&b, "Person %d,p%d@chunk.example,512-555-%04d,%d Main St,Austin,TX,78701\n", i, i, i, i)
	}
	tbl, err := imports.ReadTable([]byte(b.String()), "chunk.csv")
	require.NoError(t, err)
	p, err := imports.Prepare(tbl, imports.SourceGeneric, imports.EntityCustomers, nil)
	require.NoError(t, err)
	require.Empty(t, p.Missing)

	job, err := svc.CreateMappedImportJob(a.ctx, a.userID, "chunk.csv", imports.SourceGeneric, imports.EntityCustomers, p.Records)
	require.NoError(t, err)

	// A tiny budget runs one batch per chunk and hands the job back.
	chunks := 0
	for {
		claimed, err := svc.ClaimImportJob(a.ctx, job.ID)
		require.NoError(t, err)
		if claimed == nil {
			break
		}
		chunks++
		done, err := svc.RunImportChunk(a.ctx, claimed, time.Nanosecond)
		require.NoError(t, err)
		if done.Status != models.ImportJobStatusPending {
			errs, err := svc.GetImportErrors(a.ctx, job.ID, 3)
			require.NoError(t, err)
			assert.Empty(t, errs)
			assert.Equal(t, models.ImportJobStatusCompleted, done.Status)
			break
		}
		assert.Equal(t, chunks*importBatchRows, done.ProcessedRows)
	}
	assert.Equal(t, 3, chunks, "60 rows in batches of 25")
	assert.EqualValues(t, 61, count(t, db, "customers", a.tenantID))

	// An interrupted mapped job resumes from its cursor instead of failing.
	job2, err := svc.CreateMappedImportJob(a.ctx, a.userID, "again.csv", imports.SourceGeneric, imports.EntityCustomers, p.Records)
	require.NoError(t, err)
	claimed, err := svc.ClaimImportJob(a.ctx, job2.ID)
	require.NoError(t, err)
	require.NotNil(t, claimed)
	require.NoError(t, db.WithContext(tenancy.WithSystem(context.Background())).
		Exec(`UPDATE import_jobs SET updated_at = ?, processed_rows = 25, skipped_rows = 25 WHERE id = ?`, time.Now().Add(-time.Hour), job2.ID).Error)
	res, err := svc.ProcessPending(context.Background(), 10, time.Minute)
	require.NoError(t, err)
	assert.EqualValues(t, 1, res.Interrupted)
	var got models.ImportJob
	require.NoError(t, db.WithContext(a.ctx).First(&got, "id = ?", job2.ID).Error)
	assert.Equal(t, models.ImportJobStatusCompleted, got.Status)
	assert.Equal(t, 1, got.Attempts)
	assert.Equal(t, 60, got.SkippedRows, "already imported rows are skipped, the cursor kept")
	assert.EqualValues(t, 61, count(t, db, "customers", a.tenantID))
}
