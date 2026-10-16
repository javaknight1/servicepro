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
	"gorm.io/gorm/logger"

	"github.com/javaknight1/servicepro/backend/internal/models"
	"github.com/javaknight1/servicepro/backend/internal/repository"
	"github.com/javaknight1/servicepro/backend/internal/tenancy"
)

// isolationFixture is a SQLite database with the production tenancy plugin
// installed and two tenants, A and B, each owning one customer.
type isolationFixture struct {
	db         *gorm.DB
	ctxA, ctxB context.Context
	custA      *models.Customer
	custB      *models.Customer
	userA      uuid.UUID
}

func setupIsolation(t *testing.T) *isolationFixture {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	require.NoError(t, err)
	sqlDB, err := db.DB()
	require.NoError(t, err)
	sqlDB.SetMaxOpenConns(1)

	for _, m := range []any{
		&models.User{}, &models.Customer{}, &models.Job{}, &models.JobAssignment{},
		&models.JobMaterial{}, &models.JobNote{}, &models.Quote{}, &models.QuoteItem{},
		&models.Invoice{}, &models.InvoiceLine{}, &models.InvoicePayment{},
		&models.PaymentTerm{}, &models.TaxRate{}, &models.Payment{}, &models.ExportJob{},
	} {
		createSQLiteTableForModel(t, db, m, "")
	}
	require.NoError(t, tenancy.Register(db))

	f := &isolationFixture{
		db:    db,
		ctxA:  tenancy.WithTenant(context.Background(), uuid.New()),
		ctxB:  tenancy.WithTenant(context.Background(), uuid.New()),
		userA: uuid.New(),
	}
	repo := repository.NewCustomerRepository(db)
	f.custA = &models.Customer{ID: uuid.New(), FirstName: "Alice", LastName: "A", Email: "alice@a.test", CustomerType: "residential", Status: "active"}
	f.custB = &models.Customer{ID: uuid.New(), FirstName: "Bob", LastName: "B", Email: "bob@b.test", CustomerType: "residential", Status: "active"}
	require.NoError(t, repo.WithContext(f.ctxA).Create(f.custA))
	require.NoError(t, repo.WithContext(f.ctxB).Create(f.custB))
	return f
}

func TestTenantIsolation_Customers(t *testing.T) {
	f := setupIsolation(t)
	repoA := repository.NewCustomerRepository(f.db).WithContext(f.ctxA)
	repoB := repository.NewCustomerRepository(f.db).WithContext(f.ctxB)

	tidA, _ := tenancy.TenantID(f.ctxA)
	assert.Equal(t, tidA, f.custA.TenantID, "create assigns the caller's tenant")

	_, err := repoB.GetByID(f.custA.ID)
	assert.ErrorIs(t, err, repository.ErrCustomerNotFound)
	_, err = repoB.GetByEmail(f.custA.Email)
	assert.ErrorIs(t, err, repository.ErrCustomerNotFound)

	list, total, err := repoB.List(&models.CustomerListFilter{Page: 1, PageSize: 50})
	require.NoError(t, err)
	assert.Equal(t, int64(1), total)
	require.Len(t, list, 1)
	assert.Equal(t, f.custB.ID, list[0].ID)

	found, err := repoB.Search("alice")
	require.NoError(t, err)
	assert.Empty(t, found)
	ft, ftTotal, err := repoB.FullTextSearch("alice", "", "", "", "", 10, 0, "", "")
	require.NoError(t, err)
	assert.Empty(t, ft)
	assert.Zero(t, ftTotal)

	hijack := *f.custA
	hijack.FirstName = "Pwned"
	assert.Error(t, repoB.Update(&hijack))
	assert.ErrorIs(t, repoB.Delete(f.custA.ID), repository.ErrCustomerNotFound)

	exists, err := repoB.EmailExists(f.custA.Email)
	require.NoError(t, err)
	assert.False(t, exists, "emails are unique per tenant, not globally")

	still, err := repoA.GetByID(f.custA.ID)
	require.NoError(t, err)
	assert.Equal(t, "Alice", still.FirstName)
}

func TestTenantIsolation_Jobs(t *testing.T) {
	f := setupIsolation(t)
	repoA := repository.NewJobRepository(f.db).WithContext(f.ctxA)
	repoB := repository.NewJobRepository(f.db).WithContext(f.ctxB)

	job := &models.Job{ID: uuid.New(), CustomerID: f.custA.ID, Title: "Fix sink", Status: models.JobStatusNew, CreatedBy: f.userA}
	require.NoError(t, repoA.Create(job))

	_, err := repoB.GetByID(job.ID)
	assert.Error(t, err)
	_, err = repoB.GetByJobNumber(job.JobNumber)
	assert.Error(t, err)

	jobs, total, err := repoB.List(&models.JobListFilter{})
	require.NoError(t, err)
	assert.Zero(t, total)
	assert.Empty(t, jobs)

	// A full-row Save from tenant B must not overwrite A's job (GORM's Save
	// falls back to an upsert when the scoped UPDATE matches nothing).
	hijack := *job
	hijack.Title = "Pwned"
	_ = repoB.Update(&hijack)
	require.NoError(t, repoB.Delete(job.ID))

	still, err := repoA.GetByID(job.ID)
	require.NoError(t, err)
	assert.Equal(t, "Fix sink", still.Title)
}

func TestTenantIsolation_Quotes(t *testing.T) {
	f := setupIsolation(t)
	repoA := repository.NewQuoteRepository(f.db).WithContext(f.ctxA)
	repoB := repository.NewQuoteRepository(f.db).WithContext(f.ctxB)

	quote := &models.Quote{ID: uuid.New(), CustomerID: f.custA.ID, QuoteNumber: "QUO-1", ValidUntil: time.Now().Add(time.Hour), Status: models.QuoteStatusDraft, CreatedBy: f.userA}
	require.NoError(t, repoA.Create(quote))

	// Tenant B can't attach a quote to A's customer.
	assert.Error(t, repoB.Create(&models.Quote{ID: uuid.New(), CustomerID: f.custA.ID, QuoteNumber: "QUO-X", ValidUntil: time.Now().Add(time.Hour), Status: models.QuoteStatusDraft, CreatedBy: f.userA}))

	_, err := repoB.GetByID(quote.ID)
	assert.Error(t, err)
	quotes, total, err := repoB.List(&models.QuoteQueryParams{Page: 1, PageSize: 50})
	require.NoError(t, err)
	assert.Zero(t, total)
	assert.Empty(t, quotes)

	require.NoError(t, repoB.Delete(quote.ID))
	_ = repoB.UpdateStatus(quote.ID, models.QuoteStatusAccepted)

	still, err := repoA.GetByID(quote.ID)
	require.NoError(t, err)
	assert.Equal(t, models.QuoteStatusDraft, still.Status)
}

func TestTenantIsolation_InvoicesAndPayments(t *testing.T) {
	f := setupIsolation(t)
	svc := NewInvoiceService(f.db)
	now := time.Now()

	inv, err := svc.CreateInvoice(f.ctxA, &models.Invoice{
		CustomerID: f.custA.ID, InvoiceNumber: "INV-A-1", IssueDate: now, DueDate: now.Add(24 * time.Hour),
		TotalAmount: decimal.NewFromInt(100),
	}, f.userA)
	require.NoError(t, err)

	_, err = svc.CreateInvoice(f.ctxB, &models.Invoice{
		CustomerID: f.custA.ID, InvoiceNumber: "INV-B-1", IssueDate: now, DueDate: now.Add(24 * time.Hour),
	}, f.userA)
	assert.ErrorIs(t, err, ErrCustomerNotFound, "B can't invoice A's customer")

	_, err = svc.GetInvoice(f.ctxB, inv.ID)
	assert.Error(t, err)
	_, err = svc.GetInvoiceByNumber(f.ctxB, inv.InvoiceNumber)
	assert.Error(t, err)
	_, err = svc.UpdateInvoice(f.ctxB, inv.ID, &models.Invoice{Notes: "pwned"}, f.userA)
	assert.Error(t, err)
	assert.Error(t, svc.DeleteInvoice(f.ctxB, inv.ID))
	_, err = svc.AddLineItem(f.ctxB, inv.ID, &models.InvoiceLine{Description: "x", Quantity: decimal.NewFromInt(1), UnitPrice: decimal.NewFromInt(1)})
	assert.Error(t, err, "B can't add lines to A's invoice")
	_, err = svc.RecordPayment(f.ctxB, &models.InvoicePayment{InvoiceID: inv.ID, Amount: decimal.NewFromInt(10)})
	assert.Error(t, err, "B can't record payments on A's invoice")

	list, err := svc.ListInvoices(f.ctxB, &models.InvoiceFilter{Page: 1, PageSize: 50})
	require.NoError(t, err)
	assert.Zero(t, list.Total)

	still, err := svc.GetInvoice(f.ctxA, inv.ID)
	require.NoError(t, err)
	assert.Empty(t, still.Notes)

	// Payments (Stripe payment records)
	pay := &models.Payment{ID: uuid.New(), UserID: f.userA, InvoiceID: &inv.ID, StripePaymentIntentID: "pi_a", Amount: decimal.NewFromInt(10), Currency: "usd", Status: "succeeded"}
	require.NoError(t, f.db.WithContext(f.ctxA).Omit("Metadata").Create(pay).Error)
	payRepoB := repository.NewPaymentRepository(f.db).WithContext(f.ctxB)
	_, err = payRepoB.GetPaymentByID(pay.ID)
	assert.Error(t, err)
	_, err = payRepoB.GetPaymentByStripeID("pi_a")
	assert.Error(t, err)
	_, err = repository.NewPaymentRepository(f.db).WithContext(f.ctxA).GetPaymentByID(pay.ID)
	assert.NoError(t, err)
}

func TestTenantIsolation_Exports(t *testing.T) {
	f := setupIsolation(t)
	svc := NewUnifiedExportService(f.db, nil, t.TempDir(), "")

	job, err := svc.StartExport(f.ctxA, f.userA, &models.ExportConfig{Format: models.ExportFormatCSV, ExportType: models.ExportTypeCustomers})
	require.NoError(t, err)

	_, err = svc.GetExportStatus(f.ctxB, job.ID)
	assert.Error(t, err)
	_, _, err = svc.GetExportFile(f.ctxB, job.ID)
	assert.Error(t, err)
	assert.Error(t, svc.CancelExport(f.ctxB, job.ID))
	list, err := svc.ListExports(f.ctxB, f.userA, 50)
	require.NoError(t, err)
	assert.Empty(t, list)

	_, err = svc.GetExportStatus(f.ctxA, job.ID)
	assert.NoError(t, err)

	// The export itself only contains the caller's tenant's rows.
	data, n, err := svc.(*unifiedExportService).fetchData(f.ctxB, &models.ExportConfig{Format: models.ExportFormatCSV, ExportType: models.ExportTypeCustomers})
	require.NoError(t, err)
	assert.Equal(t, 1, n)
	customers, ok := data.([]models.Customer)
	require.True(t, ok)
	require.Len(t, customers, 1)
	assert.Equal(t, "bob@b.test", customers[0].Email)
}

func TestTenantIsolation_NoTenantFailsClosed(t *testing.T) {
	f := setupIsolation(t)
	_, err := repository.NewCustomerRepository(f.db).WithContext(context.Background()).GetByID(f.custA.ID)
	assert.Error(t, err)
	var n int64
	assert.ErrorIs(t, f.db.Model(&models.Customer{}).Count(&n).Error, tenancy.ErrNoTenant)
}
