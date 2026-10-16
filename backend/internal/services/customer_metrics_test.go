package services

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/javaknight1/servicepro/backend/internal/models"
	"github.com/javaknight1/servicepro/backend/internal/tenancy"
)

type metricsFixture struct {
	db               *gorm.DB
	svc              *customerReportService
	tenantA, tenantB uuid.UUID
	custA, custB     uuid.UUID
}

func setupMetricsFixture(t *testing.T) *metricsFixture {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	sqlDB, err := db.DB()
	require.NoError(t, err)
	sqlDB.SetMaxOpenConns(1)
	require.NoError(t, tenancy.Register(db))

	createSQLiteTableForModel(t, db, &models.Customer{}, "")
	createSQLiteTableForModel(t, db, &models.Job{}, "")
	createSQLiteTableForModel(t, db, &models.Invoice{}, "")
	createSQLiteTableForModel(t, db, &models.Quote{}, "")
	createSQLiteTableForModel(t, db, &models.CustomerMetricsCache{}, "UNIQUE (customer_id)")

	f := &metricsFixture{db: db, svc: &customerReportService{db: db},
		tenantA: uuid.New(), tenantB: uuid.New(), custA: uuid.New(), custB: uuid.New()}

	created := time.Now().AddDate(0, -3, 0)
	for _, c := range []struct {
		id, tenant uuid.UUID
		email      string
	}{{f.custA, f.tenantA, "a@example.com"}, {f.custB, f.tenantB, "b@example.com"}} {
		require.NoError(t, db.Exec(
			`INSERT INTO customers (id, tenant_id, first_name, last_name, email, customer_type, status, created_at, updated_at)
			 VALUES (?, ?, 'Pat', 'Doe', ?, 'commercial', 'active', ?, ?)`,
			c.id.String(), c.tenant.String(), c.email, created, created).Error)
	}

	// Tenant A's customer: two completed jobs, one paid invoice, one accepted
	// and one declined quote; first job ~45 days after the customer was created.
	jobAt := created.AddDate(0, 0, 45)
	for i, st := range []string{"completed", "paid"} {
		require.NoError(t, db.Exec(
			`INSERT INTO jobs (id, tenant_id, job_number, customer_id, title, status, total_amount, created_by, created_at, updated_at)
			 VALUES (?, ?, ?, ?, 'Service', ?, 300, ?, ?, ?)`,
			uuid.New().String(), f.tenantA.String(), "J-"+st, f.custA.String(), st, uuid.New().String(),
			jobAt.AddDate(0, 0, i*10), jobAt).Error)
	}
	require.NoError(t, db.Exec(
		`INSERT INTO invoices (id, tenant_id, invoice_number, customer_id, status, issue_date, due_date, total_amount, amount_paid, amount_due, created_by, created_at, updated_at)
		 VALUES (?, ?, 'INV-1', ?, 'paid', ?, ?, 600, 600, 0, ?, ?, ?)`,
		uuid.New().String(), f.tenantA.String(), f.custA.String(), jobAt, jobAt, uuid.New().String(), jobAt, jobAt).Error)
	for _, st := range []string{"accepted", "declined"} {
		require.NoError(t, db.Exec(
			`INSERT INTO quotes (id, tenant_id, quote_number, customer_id, status, created_by, created_at, updated_at)
			 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
			uuid.New().String(), f.tenantA.String(), "Q-"+st, f.custA.String(), st, uuid.New().String(), created, created).Error)
	}
	return f
}

// T123: customer_metrics_cache was never populated (the SQL function behind
// RefreshMetricsCache did not exist), so top/at-risk/segment reports were
// empty. The report endpoints now compute it, tenant-scoped.
func TestCustomerMetrics_PopulatedOnRead(t *testing.T) {
	f := setupMetricsFixture(t)
	ctxA := tenancy.WithTenant(context.Background(), f.tenantA)

	top, err := f.svc.GetTopCustomers(ctxA, &models.CustomerReportQuery{}, 10)
	require.NoError(t, err)
	require.Len(t, top, 1, "only tenant A's customer")
	assert.Equal(t, f.custA, top[0].CustomerID)
	assert.True(t, top[0].TotalRevenue.Equal(decimal.NewFromInt(600)), "revenue = %s", top[0].TotalRevenue)
	assert.Equal(t, 2, top[0].TotalJobs)
	assert.Equal(t, 2, top[0].CompletedJobs)

	var rows []models.CustomerMetricsCache
	require.NoError(t, f.db.Find(&rows).Error)
	require.Len(t, rows, 1, "no cache row written for the other tenant's customer")
	assert.Equal(t, f.custA, rows[0].CustomerID)
	assert.True(t, rows[0].QuoteAcceptanceRate.Equal(decimal.NewFromInt(50)))
	assert.NotEmpty(t, rows[0].Segment)

	// A second read with a fresh cache does not rewrite it.
	before := rows[0].LastUpdated
	_, err = f.svc.GetTopCustomers(ctxA, &models.CustomerReportQuery{}, 10)
	require.NoError(t, err)
	require.NoError(t, f.db.First(&rows[0], "customer_id = ?", f.custA).Error)
	assert.True(t, rows[0].LastUpdated.Equal(before))
}

func TestCustomerMetrics_RefreshIsTenantScoped(t *testing.T) {
	f := setupMetricsFixture(t)
	ctxB := tenancy.WithTenant(context.Background(), f.tenantB)

	err := f.svc.RefreshMetricsCache(ctxB, &f.custA)
	assert.ErrorIs(t, err, ErrCustomerNotFound, "tenant B cannot refresh tenant A's customer")

	require.NoError(t, f.svc.RefreshMetricsCache(ctxB, nil))
	var rows []models.CustomerMetricsCache
	require.NoError(t, f.db.Find(&rows).Error)
	require.Len(t, rows, 1)
	assert.Equal(t, f.custB, rows[0].CustomerID)

	assert.ErrorIs(t, f.svc.RefreshMetricsCache(context.Background(), nil), tenancy.ErrNoTenant)
}

// T123: cohort analysis returned "unavailable" because v_customer_cohorts
// never existed.
func TestCustomerCohorts(t *testing.T) {
	f := setupMetricsFixture(t)
	ctxA := tenancy.WithTenant(context.Background(), f.tenantA)

	cohorts, err := f.svc.GetCohortData(ctxA, 12)
	require.NoError(t, err)
	require.Len(t, cohorts, 1)
	c := cohorts[0]
	assert.Equal(t, 1, c.CohortSize)
	assert.Equal(t, 1, c.Month1Retained, "activity ~45-55 days after signup counts for month 1")
	assert.Equal(t, 0, c.Month3Retained)
	assert.True(t, c.RetentionRate1M.Equal(decimal.NewFromInt(100)))
}

// T123: a filtered summary referenced alias "c" on an un-aliased customers
// table ("missing FROM-clause entry for table c").
func TestCustomerReportSummary_WithFilters(t *testing.T) {
	f := setupMetricsFixture(t)
	ctxA := tenancy.WithTenant(context.Background(), f.tenantA)

	summary, err := f.svc.calculateSummary(ctxA, &models.CustomerReportQuery{
		Status:       models.CustomerStatusActive,
		CustomerType: models.CustomerType("commercial"),
	})
	require.NoError(t, err)
	assert.Equal(t, 1, summary.TotalCustomers)
	assert.Equal(t, 1, summary.ActiveCustomers)
}
