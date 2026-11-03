package services

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	apperrors "github.com/javaknight1/servicepro/backend/internal/errors"
	"github.com/javaknight1/servicepro/backend/internal/models"
	"github.com/javaknight1/servicepro/backend/internal/repository"
	"github.com/javaknight1/servicepro/backend/internal/tenancy"
)

// T029/T030 on real Postgres: one-click invoice from a completed job, built
// from the job's time, materials and notes, at most once per job.

// jobInvoiceFixture is a tenant with a default 8% tax rate, a customer with
// a standing PO and a site, and a completed job at that site.
type jobInvoiceFixture struct {
	p    *pgTenant
	db   *gorm.DB
	svc  *InvoiceService
	site *models.Site
	job  *models.Job
}

func seedJobInvoice(t *testing.T, db *gorm.DB, laborRate string) *jobInvoiceFixture {
	t.Helper()
	p := seedPGTenant(t, db)
	f := &jobInvoiceFixture{p: p, db: db, svc: NewInvoiceService(db)}

	tid := p.tenantID
	rate := &models.TaxRate{ID: uuid.New(), TenantID: &tid, Name: "Ohio", Rate: decimal.RequireFromString("0.08"),
		TaxType: models.TaxType("sales_tax"), IsActive: true}
	require.NoError(t, db.WithContext(p.ctx).Create(rate).Error)
	settings := models.TenantSettings{DefaultTaxRateID: &rate.ID}
	if laborRate != "" {
		r := decimal.RequireFromString(laborRate)
		settings.LaborRate = &r
	}
	require.NoError(t, db.WithContext(tenancy.WithSystem(context.Background())).
		Model(&models.Tenant{}).Where("id = ?", p.tenantID).Update("settings", settings).Error)
	require.NoError(t, db.WithContext(tenancy.WithSystem(context.Background())).Exec(
		`UPDATE customers SET default_po_number = 'PO-7731' WHERE id = ?`, p.customerID).Error)

	f.site = &models.Site{CustomerID: p.customerID, Name: "Harborview Lofts", AddressStreet: "220 W Spring St", IsActive: true}
	require.NoError(t, repository.NewSiteRepository(db).Create(p.ctx, p.tenantID, f.site))

	// On site 09:00-11:30: 2.5 hours.
	start := time.Date(2026, 10, 14, 9, 0, 0, 0, time.UTC).Unix()
	end := start + 150*60
	done := "Replaced the lobby vacuum belt."
	siteID := f.site.ID
	f.job = &models.Job{
		ID: uuid.New(), CustomerID: p.customerID, SiteID: &siteID, Title: "Carpet extraction, lobby",
		JobType: models.JobTypeRepair, Status: models.JobStatusCompleted, Priority: models.JobPriorityNormal,
		ActualStartAt: &start, ActualEndAt: &end, EstimatedCost: 400, ActualCost: 450,
		CompletionNotes: &done, CreatedBy: p.userID,
	}
	require.NoError(t, db.WithContext(p.ctx).Create(f.job).Error)

	for _, m := range []models.JobMaterial{
		{Name: "Vacuum belt", Quantity: 2, Unit: "each", UnitCost: 12.5, Billable: true},
		{Name: "Shop towels", Quantity: 1, Unit: "box", UnitCost: 9.99, Billable: false},
	} {
		m.ID, m.JobID = uuid.New(), f.job.ID
		require.NoError(t, db.WithContext(p.ctx).Create(&m).Error)
	}
	for _, n := range []models.JobNote{
		{Note: "Customer approved the extra spot treatment.", IsInternal: false},
		{Note: "Gate code 4411", IsInternal: true},
	} {
		n.ID, n.JobID, n.UserID = uuid.New(), f.job.ID, p.userID
		require.NoError(t, db.WithContext(p.ctx).Create(&n).Error)
	}
	return f
}

func (f *jobInvoiceFixture) countInvoices(t *testing.T) int64 {
	t.Helper()
	var n int64
	require.NoError(t, f.db.WithContext(f.p.ctx).Model(&models.Invoice{}).Where("job_id = ?", f.job.ID).Count(&n).Error)
	return n
}

func TestPostgres_InvoiceJob_LaborMaterialsNotes(t *testing.T) {
	db := setupPostgres(t)
	f := seedJobInvoice(t, db, "85.50")

	inv, err := f.svc.InvoiceJob(f.p.ctx, f.job.ID, f.p.userID)
	require.NoError(t, err)
	assert.Equal(t, models.InvoiceStatusDraft, inv.Status)
	require.NotNil(t, inv.JobID)
	assert.Equal(t, f.job.ID, *inv.JobID)
	assert.Nil(t, inv.QuoteID)
	assert.Equal(t, "PO-7731", inv.PONumber)

	require.Len(t, inv.Lines, 2, "labor + the billable material; the non-billable one is left out")
	labor := inv.Lines[0]
	assert.Contains(t, labor.Description, "Labor · Carpet extraction, lobby")
	assert.Equal(t, "2.50", labor.Quantity.StringFixed(2))
	assert.Equal(t, "85.50", labor.UnitPrice.StringFixed(2))
	assert.Equal(t, "hour", labor.UnitOfMeasure)
	mat := inv.Lines[1]
	assert.Equal(t, "Vacuum belt", mat.Description)
	assert.Equal(t, "2.00", mat.Quantity.StringFixed(2))
	assert.Equal(t, "12.50", mat.UnitPrice.StringFixed(2))
	for _, l := range inv.Lines {
		require.NotNil(t, l.SiteName)
		assert.Equal(t, "Harborview Lofts", *l.SiteName)
		assert.True(t, l.Taxable)
	}

	// Server-side totals: 2.5 × 85.50 = 213.75; + 25.00 = 238.75; 8% = 19.10.
	assert.Equal(t, "238.75", inv.Subtotal.StringFixed(2))
	assert.Equal(t, "19.10", inv.TaxAmount.StringFixed(2))
	assert.Equal(t, "257.85", inv.TotalAmount.StringFixed(2))

	assert.Contains(t, inv.Notes, "Replaced the lobby vacuum belt.")
	assert.Contains(t, inv.Notes, "Customer approved the extra spot treatment.")
	assert.NotContains(t, inv.Notes, "Gate code", "internal notes never reach the customer")

	assert.Equal(t, models.JobStatusInvoiced, jobStatus(t, db, f.p.ctx, f.job.ID))
	var history int64
	require.NoError(t, db.WithContext(f.p.ctx).Model(&models.JobStatusTransition{}).
		Where("job_id = ? AND to_status = ?", f.job.ID, models.JobStatusInvoiced).Count(&history).Error)
	assert.Equal(t, int64(1), history)

	// GET /jobs/{id} names the invoice.
	jobs := NewJobService(repository.NewJobRepository(db), repository.NewCustomerRepository(db), db).WithContext(f.p.ctx)
	resp, err := jobs.GetJobByID(f.job.ID, f.p.userID, models.UserRole("admin"))
	require.NoError(t, err)
	require.NotNil(t, resp.InvoiceID)
	assert.Equal(t, inv.ID, *resp.InvoiceID)
}

// Without a labor rate the job's amount is one line (actual cost here).
func TestPostgres_InvoiceJob_NoRateBillsJobAmount(t *testing.T) {
	db := setupPostgres(t)
	f := seedJobInvoice(t, db, "")

	inv, err := f.svc.InvoiceJob(f.p.ctx, f.job.ID, f.p.userID)
	require.NoError(t, err)
	require.Len(t, inv.Lines, 2)
	assert.Equal(t, "Carpet extraction, lobby · "+f.job.JobNumber, inv.Lines[0].Description)
	assert.Equal(t, "450.00", inv.Lines[0].UnitPrice.StringFixed(2))
	assert.Equal(t, "475.00", inv.Subtotal.StringFixed(2))
	assert.Equal(t, "513.00", inv.TotalAmount.StringFixed(2))
}

// A job converted from a quote bills the quote's lines and links the quote.
func TestPostgres_InvoiceJob_FromQuote(t *testing.T) {
	db := setupPostgres(t)
	f := seedJobInvoice(t, db, "85.50")
	q := seedPGQuote(t, db, f.p, models.QuoteStatusAccepted)
	require.NoError(t, db.WithContext(f.p.ctx).Model(&models.Job{}).Where("id = ?", f.job.ID).Update("quote_id", q.ID).Error)

	inv, err := f.svc.InvoiceJob(f.p.ctx, f.job.ID, f.p.userID)
	require.NoError(t, err)
	require.NotNil(t, inv.QuoteID)
	assert.Equal(t, q.ID, *inv.QuoteID)
	require.Len(t, inv.Lines, 3, "two quote lines + the billable material; no labor on a quoted job")
	assert.Equal(t, "Carpet extraction, lobby", inv.Lines[0].Description)
	assert.Equal(t, "150.00", inv.Lines[0].UnitPrice.StringFixed(2))
	assert.Equal(t, "Spot treatment", inv.Lines[1].Description)
	// 300 + 45.50 + 25 = 370.50
	assert.Equal(t, "370.50", inv.Subtotal.StringFixed(2))

	// The quote can't be invoiced again on its own.
	_, err = f.svc.CreateInvoice(f.p.ctx, &models.Invoice{CustomerID: f.p.customerID, QuoteID: &q.ID,
		Lines: []models.InvoiceLine{{Description: "x", Quantity: decimal.NewFromInt(1), UnitPrice: decimal.NewFromInt(1)}}}, f.p.userID)
	assert.ErrorIs(t, err, ErrAlreadyInvoiced)
}

// Idempotent: the second call is a 409 naming the first invoice; cancelling
// that invoice frees the job to be invoiced again.
func TestPostgres_InvoiceJob_Once(t *testing.T) {
	db := setupPostgres(t)
	f := seedJobInvoice(t, db, "85.50")

	first, err := f.svc.InvoiceJob(f.p.ctx, f.job.ID, f.p.userID)
	require.NoError(t, err)
	_, err = f.svc.InvoiceJob(f.p.ctx, f.job.ID, f.p.userID)
	require.ErrorIs(t, err, ErrJobAlreadyInvoiced)
	de, ok := apperrors.As(err)
	require.True(t, ok)
	assert.Equal(t, 409, de.Status)
	assert.Equal(t, first.ID.String(), de.ResourceID)

	_, err = f.svc.CancelInvoice(f.p.ctx, first.ID, "wrong rate")
	require.NoError(t, err)
	assert.Equal(t, models.JobStatusCompleted, jobStatus(t, db, f.p.ctx, f.job.ID))
	second, err := f.svc.InvoiceJob(f.p.ctx, f.job.ID, f.p.userID)
	require.NoError(t, err)
	assert.NotEqual(t, first.ID, second.ID)
}

func TestPostgres_InvoiceJob_Concurrent(t *testing.T) {
	db := setupPostgres(t)
	f := seedJobInvoice(t, db, "85.50")

	const n = 8
	var wg sync.WaitGroup
	errs := make([]error, n)
	ids := make([]uuid.UUID, n)
	for i := range n {
		wg.Go(func() {
			inv, err := f.svc.InvoiceJob(f.p.ctx, f.job.ID, f.p.userID)
			errs[i] = err
			if err == nil {
				ids[i] = inv.ID
			}
		})
	}
	wg.Wait()
	var winner uuid.UUID
	for i, err := range errs {
		if err == nil {
			assert.Equal(t, uuid.Nil, winner, "only one call may create the invoice")
			winner = ids[i]
			continue
		}
		require.True(t, errors.Is(err, ErrJobAlreadyInvoiced), "unexpected error: %v", err)
	}
	require.NotEqual(t, uuid.Nil, winner)
	for _, err := range errs {
		if de, ok := apperrors.As(err); ok {
			assert.Equal(t, winner.String(), de.ResourceID)
		}
	}
	assert.Equal(t, int64(1), f.countInvoices(t))
}

func TestPostgres_InvoiceJob_RefusesNotCompletedAndVisits(t *testing.T) {
	db := setupPostgres(t)
	f := seedJobInvoice(t, db, "")

	for _, st := range []models.JobStatus{models.JobStatusScheduled, models.JobStatusInProgress, models.JobStatusCancelled} {
		require.NoError(t, db.WithContext(f.p.ctx).Model(&models.Job{}).Where("id = ?", f.job.ID).Update("status", st).Error)
		_, err := f.svc.InvoiceJob(f.p.ctx, f.job.ID, f.p.userID)
		assert.ErrorIs(t, err, ErrJobNotInvoiceable, "status %s", st)
	}
	assert.Zero(t, f.countInvoices(t))

	// Agreement visits are billed by the agreement.
	agreement, err := NewAgreementService(db, repository.NewAgreementRepository(db)).Create(f.p.ctx, &models.CreateAgreementRequest{
		CustomerID: f.p.customerID, SiteID: f.site.ID, Title: "Nightly cleaning",
		AgreementScheduleInput: models.AgreementScheduleInput{Frequency: models.FrequencyWeekdays},
		StartDate:              "2026-09-01", PricePerPeriod: decimal.NewFromInt(2400),
	}, f.p.userID)
	require.NoError(t, err)
	require.NoError(t, db.WithContext(f.p.ctx).Model(&models.Job{}).Where("id = ?", f.job.ID).
		Updates(map[string]any{"status": models.JobStatusCompleted, "agreement_id": agreement.ID}).Error)
	_, err = f.svc.InvoiceJob(f.p.ctx, f.job.ID, f.p.userID)
	assert.ErrorIs(t, err, ErrAgreementVisitInvoice)
	assert.Zero(t, f.countInvoices(t))
}

// A job billed on a consolidated invoice can't be invoiced on its own (and
// the 409 names the consolidated invoice); a job invoiced on its own is left
// off the next consolidated invoice.
func TestPostgres_InvoiceJob_ConsolidatedGuard(t *testing.T) {
	db := setupPostgres(t)
	c := seedConsolidation(t, db)

	consolidated, err := c.svc.ConsolidateInvoice(c.p.ctx, c.october(), c.p.userID)
	require.NoError(t, err)
	_, err = c.svc.InvoiceJob(c.p.ctx, c.approvedJob, c.p.userID)
	require.ErrorIs(t, err, ErrJobAlreadyInvoiced)
	de, _ := apperrors.As(err)
	assert.Equal(t, consolidated.ID.String(), de.ResourceID)

	// An agreement visit billed through its agreement's month names the
	// same invoice.
	_, err = c.svc.InvoiceJob(c.p.ctx, c.visitJobs["monthly-2026-10-02"], c.p.userID)
	require.ErrorIs(t, err, ErrJobAlreadyInvoiced)
	de, _ = apperrors.As(err)
	assert.Equal(t, consolidated.ID.String(), de.ResourceID)

	// Even if the job's status were moved back by hand, the claim still
	// blocks a separate invoice (POST /invoices with job_id too).
	require.NoError(t, db.WithContext(c.p.ctx).Model(&models.Job{}).Where("id = ?", c.approvedJob).
		Update("status", models.JobStatusCompleted).Error)
	_, err = c.svc.InvoiceJob(c.p.ctx, c.approvedJob, c.p.userID)
	require.ErrorIs(t, err, ErrJobAlreadyInvoiced)
	_, err = c.svc.CreateInvoice(c.p.ctx, &models.Invoice{CustomerID: c.p.customerID, JobID: &c.approvedJob,
		Lines: []models.InvoiceLine{{Description: "x", Quantity: decimal.NewFromInt(1), UnitPrice: decimal.NewFromInt(1)}}}, c.p.userID)
	assert.ErrorIs(t, err, ErrAlreadyInvoiced)

	// The other way round: November's extra work, invoiced on its own first,
	// is not on November's consolidated invoice.
	own, err := c.svc.InvoiceJob(c.p.ctx, c.novemberJob, c.p.userID)
	require.NoError(t, err)
	assert.Equal(t, "200.00", own.Subtotal.StringFixed(2))
	preview, err := c.svc.PreviewConsolidatedInvoice(c.p.ctx, &models.ConsolidateInvoiceRequest{
		CustomerID: c.p.customerID, PeriodStart: "2026-11-01", PeriodEnd: "2026-11-30"})
	require.NoError(t, err)
	for _, l := range preview.Lines {
		if l.JobID != nil {
			assert.NotEqual(t, c.novemberJob, *l.JobID)
		}
	}
}

// Cross-tenant: tenant B can't invoice (or probe) tenant A's job.
func TestPostgres_InvoiceJob_CrossTenant(t *testing.T) {
	db := setupPostgres(t)
	f := seedJobInvoice(t, db, "85.50")
	b := seedPGTenant(t, db)

	_, err := f.svc.InvoiceJob(b.ctx, f.job.ID, b.userID)
	require.ErrorIs(t, err, ErrJobNotFound)
	assert.Zero(t, f.countInvoices(t))
	assert.Equal(t, models.JobStatusCompleted, jobStatus(t, db, f.p.ctx, f.job.ID))
}

func TestLaborHours(t *testing.T) {
	at := func(sec int64) *int64 { return &sec }
	cases := []struct {
		name string
		job  models.Job
		want string
	}{
		{"start/end", models.Job{ActualStartAt: at(0), ActualEndAt: at(9000)}, "2.50"},
		{"rounds to cents of an hour", models.Job{ActualStartAt: at(0), ActualEndAt: at(4000)}, "1.11"},
		{"end before start falls back to duration", models.Job{ActualStartAt: at(100), ActualEndAt: at(50), ActualDuration: 45}, "0.75"},
		{"duration only", models.Job{ActualDuration: 90}, "1.50"},
		{"nothing recorded", models.Job{}, "0.00"},
	}
	for _, tc := range cases {
		assert.Equal(t, tc.want, laborHours(&tc.job).StringFixed(2), tc.name)
	}
}
