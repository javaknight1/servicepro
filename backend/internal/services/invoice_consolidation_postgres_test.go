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

	"github.com/javaknight1/servicepro/backend/internal/models"
	"github.com/javaknight1/servicepro/backend/internal/repository"
	"github.com/javaknight1/servicepro/backend/internal/tenancy"
)

// T188 on real Postgres (migration 000015): consolidated invoices, one per
// customer and period, never billing a visit, month or job twice.

var nyLoc = func() *time.Location {
	l, err := time.LoadLocation("America/New_York")
	if err != nil {
		panic(err)
	}
	return l
}()

// consolidationFixture is a tenant in New York with a property-manager
// customer: two sites, a monthly and a per-visit agreement, extra work.
type consolidationFixture struct {
	p            *pgTenant
	svc          *InvoiceService
	lofts, river *models.Site
	monthly      *models.ServiceAgreement
	perVisit     *models.ServiceAgreement
	visitJobs    map[string]uuid.UUID // "monthly-2026-10-02" → job
	approvedJob  uuid.UUID            // WO at the lofts, approved by Karen Doyle
	underNTEJob  uuid.UUID            // at the river site, under the NTE
	novemberJob  uuid.UUID            // extra work finished after October
	scheduledJob uuid.UUID            // extra work not completed
}

func seedConsolidation(t *testing.T, db *gorm.DB) *consolidationFixture {
	t.Helper()
	p := seedPGTenant(t, db)
	sys := tenancy.WithSystem(context.Background())
	require.NoError(t, db.WithContext(sys).Exec(
		`UPDATE tenants SET settings = jsonb_set(COALESCE(settings, '{}'::jsonb), '{timezone}', '"America/New_York"') WHERE id = ?`, p.tenantID).Error)
	require.NoError(t, db.WithContext(sys).Exec(
		`UPDATE customers SET default_po_number = '7731-HV', nte_limit = 500, consolidated_billing = true WHERE id = ?`, p.customerID).Error)

	f := &consolidationFixture{p: p, svc: NewInvoiceService(db), visitJobs: map[string]uuid.UUID{}}
	sites := repository.NewSiteRepository(db)
	f.lofts = &models.Site{CustomerID: p.customerID, Name: "Harborview Lofts", AddressStreet: "220 W Spring St", IsActive: true}
	f.river = &models.Site{CustomerID: p.customerID, Name: "Riverside Commons", AddressStreet: "1 River Rd", IsActive: true}
	require.NoError(t, sites.Create(p.ctx, p.tenantID, f.lofts))
	require.NoError(t, sites.Create(p.ctx, p.tenantID, f.river))

	agreements := NewAgreementService(db, repository.NewAgreementRepository(db))
	var err error
	f.monthly, err = agreements.Create(p.ctx, &models.CreateAgreementRequest{
		CustomerID: p.customerID, SiteID: f.lofts.ID, Title: "Nightly cleaning, Mon–Fri",
		AgreementScheduleInput: models.AgreementScheduleInput{Frequency: models.FrequencyWeekdays},
		StartDate:              "2026-09-01", PricePerPeriod: decimal.NewFromInt(2400),
	}, p.userID)
	require.NoError(t, err)
	f.perVisit, err = agreements.Create(p.ctx, &models.CreateAgreementRequest{
		CustomerID: p.customerID, SiteID: f.river.ID, Title: "Cleaning 3x weekly",
		AgreementScheduleInput: models.AgreementScheduleInput{Frequency: models.FrequencyTimesPerWeek, TimesPerWeek: new(3)},
		StartDate:              "2026-09-01", PricePerPeriod: decimal.NewFromInt(150), BillingPeriod: models.BillingPeriodVisit,
	}, p.userID)
	require.NoError(t, err)

	// Monthly: two completed October visits, one completed in November.
	for _, d := range []string{"2026-10-02", "2026-10-05", "2026-11-02"} {
		f.visitJobs["monthly-"+d] = f.visit(t, db, f.monthly, d, models.JobStatusCompleted)
	}
	// Per visit: two completed in October (one on the 31st, late evening
	// local time), one still scheduled, one in November.
	f.visitJobs["visit-2026-10-07"] = f.visit(t, db, f.perVisit, "2026-10-07", models.JobStatusCompleted)
	f.visitJobs["visit-2026-10-31"] = f.visit(t, db, f.perVisit, "2026-10-31", models.JobStatusCompleted)
	f.visitJobs["visit-2026-10-09"] = f.visit(t, db, f.perVisit, "2026-10-09", models.JobStatusScheduled)
	f.visitJobs["visit-2026-11-04"] = f.visit(t, db, f.perVisit, "2026-11-04", models.JobStatusCompleted)

	// Extra work. 23:30 on Oct 31 in New York is Nov 1 in UTC: still October.
	oct31Late := time.Date(2026, 10, 31, 23, 30, 0, 0, nyLoc)
	f.approvedJob = f.job(t, db, f.lofts.ID, "Carpet extraction, lobby", models.JobStatusCompleted, time.Date(2026, 10, 14, 15, 0, 0, 0, nyLoc), 450)
	f.underNTEJob = f.job(t, db, f.river.ID, "Pressure wash entry", models.JobStatusCompleted, oct31Late, 300)
	f.novemberJob = f.job(t, db, f.river.ID, "Window cleaning", models.JobStatusCompleted, time.Date(2026, 11, 1, 0, 30, 0, 0, nyLoc), 200)
	f.scheduledJob = f.job(t, db, f.lofts.ID, "Strip and wax", models.JobStatusScheduled, time.Date(2026, 10, 20, 15, 0, 0, 0, nyLoc), 900)

	name := "Karen Doyle"
	require.NoError(t, db.WithContext(p.ctx).Create(&models.JobApproval{
		ID: uuid.New(), JobID: f.approvedJob, ApproverName: &name, ApproverEmail: "karen@harborviewpm.com",
		Amount: decimal.NewFromInt(450), ApprovedAt: time.Date(2026, 10, 10, 12, 0, 0, 0, time.UTC), RecordedBy: p.userID,
	}).Error)
	return f
}

// visit inserts an agreement visit on date with its job in status.
func (f *consolidationFixture) visit(t *testing.T, db *gorm.DB, a *models.ServiceAgreement, date string, status models.JobStatus) uuid.UUID {
	t.Helper()
	d, err := time.Parse(models.DateLayout, date)
	require.NoError(t, err)
	start := a.VisitStart(d, nyLoc).Unix()
	agreementID, siteID := a.ID, a.SiteID
	job := &models.Job{
		ID: uuid.New(), CustomerID: a.CustomerID, SiteID: &siteID, AgreementID: &agreementID, Title: a.Title,
		JobType: models.JobType("maintenance"), Status: status, Priority: models.JobPriorityNormal,
		ScheduledStartAt: &start, CreatedBy: f.p.userID,
	}
	require.NoError(t, db.WithContext(f.p.ctx).Create(job).Error)
	require.NoError(t, db.WithContext(f.p.ctx).Exec(
		`INSERT INTO agreement_visits (id, tenant_id, agreement_id, visit_date, job_id) VALUES (?, ?, ?, ?, ?)`,
		uuid.New(), f.p.tenantID, a.ID, d, job.ID).Error)
	return job.ID
}

// job inserts an extra-work job (no agreement) finished at end.
func (f *consolidationFixture) job(t *testing.T, db *gorm.DB, siteID uuid.UUID, title string, status models.JobStatus, end time.Time, total float64) uuid.UUID {
	t.Helper()
	endAt := end.Unix()
	job := &models.Job{
		ID: uuid.New(), CustomerID: f.p.customerID, SiteID: &siteID, Title: title,
		JobType: models.JobType("repair"), Status: status, Priority: models.JobPriorityNormal,
		ScheduledStartAt: &endAt, ActualEndAt: &endAt, TotalAmount: total, CreatedBy: f.p.userID,
	}
	require.NoError(t, db.WithContext(f.p.ctx).Create(job).Error)
	return job.ID
}

func (f *consolidationFixture) october() *models.ConsolidateInvoiceRequest {
	return &models.ConsolidateInvoiceRequest{CustomerID: f.p.customerID, PeriodStart: "2026-10-01", PeriodEnd: "2026-10-31"}
}

func jobStatus(t *testing.T, db *gorm.DB, ctx context.Context, id uuid.UUID) models.JobStatus {
	t.Helper()
	var j models.Job
	require.NoError(t, db.WithContext(ctx).First(&j, "id = ?", id).Error)
	return j.Status
}

func TestPostgres_ConsolidatedInvoice_PreviewLines(t *testing.T) {
	db := setupPostgres(t)
	f := seedConsolidation(t, db)

	p, err := f.svc.PreviewConsolidatedInvoice(f.p.ctx, f.october())
	require.NoError(t, err)
	assert.Equal(t, "7731-HV", p.PONumber)
	assert.Equal(t, "2026-10-01", p.PeriodStart)
	assert.Equal(t, "2026-10-31", p.PeriodEnd)
	require.Len(t, p.Lines, 4)

	// Grouped by site, agreements before extra work.
	l := p.Lines[0]
	assert.Equal(t, models.ConsolidatedLineAgreementMonth, l.Kind)
	assert.Equal(t, "Harborview Lofts", l.SiteName)
	assert.Equal(t, "Nightly cleaning, Mon–Fri · 2 visits", l.Description)
	assert.Equal(t, "2026-10", l.Month)
	assert.Equal(t, 2, l.Visits)
	assert.Equal(t, "2400.00", l.Amount.StringFixed(2))

	l = p.Lines[1]
	assert.Equal(t, models.ConsolidatedLineExtraWork, l.Kind)
	assert.Equal(t, "Harborview Lofts", l.SiteName)
	assert.Contains(t, l.Description, "Carpet extraction, lobby · ")
	assert.Equal(t, "Karen Doyle", l.ApproverName)
	assert.Equal(t, "450.00", l.ApprovalAmount.StringFixed(2))
	assert.Equal(t, "Approved by Karen Doyle for 450.00", l.ApprovalNote)
	assert.Equal(t, "450.00", l.Amount.StringFixed(2))

	l = p.Lines[2]
	assert.Equal(t, models.ConsolidatedLineAgreementVisits, l.Kind)
	assert.Equal(t, "Riverside Commons", l.SiteName)
	assert.Equal(t, "Cleaning 3x weekly · 2 visits", l.Description)
	assert.Equal(t, "2", l.Quantity.String())
	assert.Equal(t, "300.00", l.Amount.StringFixed(2))

	l = p.Lines[3]
	assert.Equal(t, models.ConsolidatedLineExtraWork, l.Kind)
	assert.Equal(t, "Riverside Commons", l.SiteName, "finished 23:30 on Oct 31 local: October")
	assert.Equal(t, "Under 500.00 NTE", l.ApprovalNote)
	assert.Empty(t, l.ApproverName)

	assert.Equal(t, 2, p.Sites)
	assert.Equal(t, 4, p.Visits)
	assert.Equal(t, 2, p.ExtraWork)
	assert.Equal(t, "3450.00", p.Subtotal.StringFixed(2))
	assert.Equal(t, "3450.00", p.Total.StringFixed(2))

	// One site only.
	req := f.october()
	req.SiteID = &f.river.ID
	p, err = f.svc.PreviewConsolidatedInvoice(f.p.ctx, req)
	require.NoError(t, err)
	require.Len(t, p.Lines, 2)
	assert.Equal(t, "Riverside Commons", p.Lines[0].SiteName)
	assert.Equal(t, "600.00", p.Subtotal.StringFixed(2))

	// The preview writes nothing.
	var claims int64
	require.NoError(t, db.WithContext(f.p.ctx).Model(&models.InvoiceBilledItem{}).Count(&claims).Error)
	assert.Zero(t, claims)
	assert.Equal(t, models.JobStatusCompleted, jobStatus(t, db, f.p.ctx, f.approvedJob))
}

func TestPostgres_ConsolidatedInvoice_SpansMonths(t *testing.T) {
	db := setupPostgres(t)
	f := seedConsolidation(t, db)

	// Oct 15 – Nov 14 touches October and November: the monthly agreement
	// bills both months in full; visits only inside the period.
	p, err := f.svc.PreviewConsolidatedInvoice(f.p.ctx, &models.ConsolidateInvoiceRequest{
		CustomerID: f.p.customerID, PeriodStart: "2026-10-15", PeriodEnd: "2026-11-14",
	})
	require.NoError(t, err)
	var months []string
	var visitLine *models.ConsolidatedInvoiceLine
	for i, l := range p.Lines {
		switch l.Kind {
		case models.ConsolidatedLineAgreementMonth:
			months = append(months, l.Month)
		case models.ConsolidatedLineAgreementVisits:
			visitLine = &p.Lines[i]
		}
	}
	assert.Equal(t, []string{"2026-10", "2026-11"}, months)
	assert.Contains(t, p.Lines[0].Description, "October 2026")
	require.NotNil(t, visitLine)
	assert.Equal(t, 2, visitLine.Visits, "Oct 31 and Nov 4")
}

func TestPostgres_ConsolidatedInvoice_CreateBillsOnce(t *testing.T) {
	db := setupPostgres(t)
	f := seedConsolidation(t, db)

	inv, err := f.svc.ConsolidateInvoice(f.p.ctx, f.october(), f.p.userID)
	require.NoError(t, err)
	assert.Equal(t, models.InvoiceStatusDraft, inv.Status)
	assert.Equal(t, "7731-HV", inv.PONumber)
	require.NotNil(t, inv.BillingPeriodStart)
	assert.Equal(t, "2026-10-01", inv.BillingPeriodStart.Format(models.DateLayout))
	assert.Equal(t, "2026-10-31", inv.BillingPeriodEnd.Format(models.DateLayout))
	assert.Equal(t, "3450.00", inv.TotalAmount.StringFixed(2))
	require.Len(t, inv.Lines, 4)
	assert.Equal(t, "Harborview Lofts", *inv.Lines[0].SiteName)
	assert.Equal(t, "Approved by Karen Doyle for 450.00", *inv.Lines[1].ApprovalNote)
	assert.Equal(t, "Riverside Commons", *inv.Lines[3].SiteName)

	for _, id := range []uuid.UUID{f.approvedJob, f.underNTEJob, f.visitJobs["monthly-2026-10-02"], f.visitJobs["visit-2026-10-31"]} {
		assert.Equal(t, models.JobStatusInvoiced, jobStatus(t, db, f.p.ctx, id))
	}
	for _, id := range []uuid.UUID{f.novemberJob, f.visitJobs["monthly-2026-11-02"], f.visitJobs["visit-2026-11-04"]} {
		assert.Equal(t, models.JobStatusCompleted, jobStatus(t, db, f.p.ctx, id), "outside the period")
	}
	assert.Equal(t, models.JobStatusScheduled, jobStatus(t, db, f.p.ctx, f.scheduledJob))

	// Same period again: nothing left.
	_, err = f.svc.ConsolidateInvoice(f.p.ctx, f.october(), f.p.userID)
	assert.ErrorIs(t, err, ErrNothingToInvoice)

	// An overlapping period bills only what is new: November's month,
	// November's visit and the November job.
	p, err := f.svc.PreviewConsolidatedInvoice(f.p.ctx, &models.ConsolidateInvoiceRequest{
		CustomerID: f.p.customerID, PeriodStart: "2026-10-15", PeriodEnd: "2026-11-30",
	})
	require.NoError(t, err)
	require.Len(t, p.Lines, 3)
	assert.Equal(t, "2026-11", p.Lines[0].Month)
	assert.Equal(t, 1, p.Lines[0].Visits)
	assert.Equal(t, "2750.00", p.Subtotal.StringFixed(2))

	// A job invoiced on its own isn't swept into a consolidated invoice, and
	// a consolidated job can't be invoiced on its own.
	_, err = f.svc.CreateInvoice(f.p.ctx, &models.Invoice{
		CustomerID: f.p.customerID, JobID: &f.approvedJob,
		Lines: []models.InvoiceLine{{Description: "x", Quantity: decimal.NewFromInt(1), UnitPrice: decimal.NewFromInt(1)}},
	}, f.p.userID)
	assert.ErrorIs(t, err, ErrAlreadyInvoiced)

	// Paying the invoice moves every billed job to paid; cancelling an
	// unpaid one gives the work back.
	_, err = f.svc.SendInvoice(f.p.ctx, inv.ID)
	require.NoError(t, err)
	_, err = f.svc.CancelInvoice(f.p.ctx, inv.ID, "wrong period")
	require.NoError(t, err)
	assert.Equal(t, models.JobStatusCompleted, jobStatus(t, db, f.p.ctx, f.approvedJob))
	assert.Equal(t, models.JobStatusCompleted, jobStatus(t, db, f.p.ctx, f.visitJobs["monthly-2026-10-02"]))
	again, err := f.svc.ConsolidateInvoice(f.p.ctx, f.october(), f.p.userID)
	require.NoError(t, err)
	assert.Equal(t, "3450.00", again.TotalAmount.StringFixed(2))

	_, err = f.svc.SendInvoice(f.p.ctx, again.ID)
	require.NoError(t, err)
	_, err = f.svc.RecordPayment(f.p.ctx, &models.InvoicePayment{InvoiceID: again.ID, Amount: again.TotalAmount, PaymentDate: time.Now()})
	require.NoError(t, err)
	assert.Equal(t, models.JobStatusPaid, jobStatus(t, db, f.p.ctx, f.approvedJob))
	assert.Equal(t, models.JobStatusPaid, jobStatus(t, db, f.p.ctx, f.visitJobs["visit-2026-10-07"]))
}

func TestPostgres_ConsolidatedInvoice_DeleteDraftReleasesWork(t *testing.T) {
	db := setupPostgres(t)
	f := seedConsolidation(t, db)
	inv, err := f.svc.ConsolidateInvoice(f.p.ctx, f.october(), f.p.userID)
	require.NoError(t, err)
	require.NoError(t, f.svc.DeleteInvoice(f.p.ctx, inv.ID))
	assert.Equal(t, models.JobStatusCompleted, jobStatus(t, db, f.p.ctx, f.underNTEJob))
	p, err := f.svc.PreviewConsolidatedInvoice(f.p.ctx, f.october())
	require.NoError(t, err)
	assert.Len(t, p.Lines, 4)
}

func TestPostgres_ConsolidatedInvoice_ConcurrentCallsBillOnce(t *testing.T) {
	db := setupPostgres(t)
	f := seedConsolidation(t, db)

	var wg sync.WaitGroup
	var mu sync.Mutex
	created := 0
	for range 8 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := f.svc.ConsolidateInvoice(f.p.ctx, f.october(), f.p.userID)
			mu.Lock()
			defer mu.Unlock()
			switch {
			case err == nil:
				created++
			case errors.Is(err, ErrNothingToInvoice), errors.Is(err, ErrConsolidationRace):
			default:
				t.Errorf("unexpected error: %v", err)
			}
		}()
	}
	wg.Wait()
	assert.Equal(t, 1, created)

	var invoices int64
	require.NoError(t, db.WithContext(f.p.ctx).Model(&models.Invoice{}).Where("billing_period_start IS NOT NULL").Count(&invoices).Error)
	assert.Equal(t, int64(1), invoices, "no orphan invoices from losing calls")
	var claims int64
	require.NoError(t, db.WithContext(f.p.ctx).Model(&models.InvoiceBilledItem{}).Count(&claims).Error)
	// One month + 2 monthly visits + 2 per-visit visits + 2 jobs.
	assert.Equal(t, int64(7), claims)
}

func TestPostgres_ConsolidatedInvoice_CrossTenant(t *testing.T) {
	db := setupPostgres(t)
	f := seedConsolidation(t, db)
	other := seedPGTenant(t, db)

	// Tenant B can't preview or invoice tenant A's customer.
	_, err := f.svc.PreviewConsolidatedInvoice(other.ctx, f.october())
	assert.ErrorIs(t, err, ErrCustomerNotFound)
	_, err = f.svc.ConsolidateInvoice(other.ctx, f.october(), other.userID)
	assert.ErrorIs(t, err, ErrCustomerNotFound)

	// Tenant B's own customer with tenant A's site.
	req := &models.ConsolidateInvoiceRequest{CustomerID: other.customerID, PeriodStart: "2026-10-01", PeriodEnd: "2026-10-31", SiteID: &f.lofts.ID}
	_, err = f.svc.PreviewConsolidatedInvoice(other.ctx, req)
	assert.ErrorIs(t, err, ErrConsolidationSiteNotFound)

	// Tenant B's customer has nothing to bill; tenant A's work stays unbilled.
	req.SiteID = nil
	_, err = f.svc.ConsolidateInvoice(other.ctx, req, other.userID)
	assert.ErrorIs(t, err, ErrNothingToInvoice)
	assert.Equal(t, models.JobStatusCompleted, jobStatus(t, db, f.p.ctx, f.approvedJob))

	// Tenant B can't see tenant A's claims.
	_, err = f.svc.ConsolidateInvoice(f.p.ctx, f.october(), f.p.userID)
	require.NoError(t, err)
	var claims int64
	require.NoError(t, db.WithContext(other.ctx).Model(&models.InvoiceBilledItem{}).Count(&claims).Error)
	assert.Zero(t, claims)
}

func TestConsolidationPeriodValidation(t *testing.T) {
	for _, tc := range []struct{ start, end string }{
		{"2026-10-31", "2026-10-01"},
		{"2026-13-01", "2026-12-01"},
		{"2026-01-01", "2027-06-01"},
		{"", "2026-10-31"},
	} {
		_, _, err := parseConsolidationPeriod(&models.ConsolidateInvoiceRequest{PeriodStart: tc.start, PeriodEnd: tc.end})
		assert.Error(t, err, "%s..%s", tc.start, tc.end)
	}
	s, e, err := parseConsolidationPeriod(&models.ConsolidateInvoiceRequest{PeriodStart: "2028-02-01", PeriodEnd: "2028-02-29"})
	require.NoError(t, err)
	assert.Equal(t, 28, int(e.Sub(s).Hours()/24))
}
