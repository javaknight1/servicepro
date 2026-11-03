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
	"github.com/javaknight1/servicepro/backend/internal/repository"
	"github.com/javaknight1/servicepro/backend/internal/tenancy"
)

// T167: service agreements and the visit generator.

type agreementFixture struct {
	*isolationFixture
	svc   *AgreementService
	siteA *models.Site
	ny    *time.Location
}

func setupAgreements(t *testing.T) *agreementFixture {
	t.Helper()
	f, sites := setupSites(t)
	createSQLiteTableForModel(t, f.db, &models.ServiceAgreement{}, "")
	createSQLiteTableForModel(t, f.db, &models.AgreementVisit{}, "UNIQUE (agreement_id, visit_date)")
	f.addMember(t, f.ctxA, f.userA, true)

	ny, err := time.LoadLocation("America/New_York")
	require.NoError(t, err)
	for _, ctx := range []context.Context{f.ctxA, f.ctxB} {
		id, _ := tenancy.TenantID(ctx)
		tenant := &models.Tenant{ID: id, Name: "T " + id.String()[:8], Slug: "t-" + id.String()[:8], OwnerID: f.userA}
		tenant.Settings.Timezone = "America/New_York"
		require.NoError(t, f.db.Create(tenant).Error)
	}
	site, err := sites.Create(f.ctxA, f.custA.ID, newSiteReq("Harborview Lofts"))
	require.NoError(t, err)
	return &agreementFixture{isolationFixture: f, svc: NewAgreementService(f.db, repository.NewAgreementRepository(f.db)), siteA: site, ny: ny}
}

func (f *agreementFixture) create(t *testing.T, sched models.AgreementScheduleInput) *models.ServiceAgreement {
	t.Helper()
	a, err := f.svc.Create(f.ctxA, &models.CreateAgreementRequest{
		CustomerID: f.custA.ID, SiteID: f.siteA.ID, Title: "Nightly cleaning",
		AgreementScheduleInput: sched, StartDate: "2026-01-01",
		PricePerPeriod: decimal.RequireFromString("2400"), AssignedTechnicianID: &f.userA,
	}, f.userA)
	require.NoError(t, err)
	return a
}

var weekdays = models.AgreementScheduleInput{Frequency: models.FrequencyWeekdays}

// visitJobs returns the agreement's jobs as local visit dates (counting each).
func (f *agreementFixture) visitJobs(t *testing.T, agreementID uuid.UUID) map[time.Time]int {
	t.Helper()
	var jobs []models.Job
	require.NoError(t, f.db.WithContext(f.ctxA).Where("agreement_id = ?", agreementID).Find(&jobs).Error)
	out := map[time.Time]int{}
	for _, j := range jobs {
		require.NotNil(t, j.ScheduledStartAt)
		start := time.Unix(*j.ScheduledStartAt, 0).In(f.ny)
		assert.Equal(t, "18:00", start.Format("15:04"), "visits start 18:00 local")
		assert.Equal(t, models.JobStatusScheduled, j.Status)
		out[models.CivilDate(start, f.ny)]++
	}
	return out
}

func TestAgreementService_CRUDPauseResumePreview(t *testing.T) {
	f := setupAgreements(t)
	a := f.create(t, models.AgreementScheduleInput{Frequency: models.FrequencyWeekly, Weekdays: []int{2, 5}})
	resp := a.ToResponse()
	assert.Equal(t, []int{2, 5}, resp.Weekdays)
	assert.Equal(t, "18:00", resp.VisitStartTime)
	assert.Equal(t, "2400", a.PricePerPeriod.String())

	title, start := "Cleaning 3x weekly", "07:15"
	three := 3
	updated, err := f.svc.Update(f.ctxA, a.ID, &models.UpdateAgreementRequest{
		Title: &title, VisitStartTime: &start,
		Schedule: &models.AgreementScheduleInput{Frequency: models.FrequencyTimesPerWeek, TimesPerWeek: &three},
	})
	require.NoError(t, err)
	assert.Equal(t, title, updated.Title)
	assert.Equal(t, []int{1, 3, 5}, updated.ToResponse().Weekdays, "default M/W/F spread")
	assert.Equal(t, 7*60+15, updated.VisitStartMinutes)

	preview, err := f.svc.Preview(f.ctxA, a.ID, 14)
	require.NoError(t, err)
	assert.Equal(t, "America/New_York", preview.Timezone)
	assert.Len(t, preview.Visits, 6, "3 a week for 2 weeks")

	paused, err := f.svc.SetStatus(f.ctxA, a.ID, models.AgreementStatusPaused)
	require.NoError(t, err)
	assert.Equal(t, models.AgreementStatusPaused, paused.Status)
	preview, err = f.svc.Preview(f.ctxA, a.ID, 14)
	require.NoError(t, err)
	assert.Empty(t, preview.Visits, "paused: no visits")
	_, err = f.svc.SetStatus(f.ctxA, a.ID, models.AgreementStatusActive)
	require.NoError(t, err)

	list, err := f.svc.List(f.ctxA, &models.AgreementListFilter{CustomerID: &f.custA.ID})
	require.NoError(t, err)
	assert.Len(t, list, 1)

	require.NoError(t, f.svc.Delete(f.ctxA, a.ID))
	_, err = f.svc.Get(f.ctxA, a.ID)
	require.ErrorIs(t, err, ErrAgreementNotFound)
}

func TestAgreementService_Validation(t *testing.T) {
	f := setupAgreements(t)
	other := &models.Customer{ID: uuid.New(), FirstName: "O", LastName: "A", Email: "o@a.test", CustomerType: "commercial", Status: "active"}
	require.NoError(t, repository.NewCustomerRepository(f.db).WithContext(f.ctxA).Create(other))

	base := func() *models.CreateAgreementRequest {
		return &models.CreateAgreementRequest{CustomerID: f.custA.ID, SiteID: f.siteA.ID, Title: "Clean",
			AgreementScheduleInput: weekdays, StartDate: "2026-01-01", PricePerPeriod: decimal.NewFromInt(100)}
	}
	req := base()
	req.CustomerID = other.ID
	_, err := f.svc.Create(f.ctxA, req, f.userA)
	require.ErrorIs(t, err, ErrSiteCustomerMismatch, "site must be the customer's")

	req = base()
	stranger := uuid.New()
	req.AssignedTechnicianID = &stranger
	_, err = f.svc.Create(f.ctxA, req, f.userA)
	require.ErrorIs(t, err, ErrTechnicianNotFound)

	for name, mutate := range map[string]func(r *models.CreateAgreementRequest){
		"bad start date": func(r *models.CreateAgreementRequest) { r.StartDate = "01/02/2026" },
		"end before start": func(r *models.CreateAgreementRequest) {
			e := "2025-12-31"
			r.EndDate = &e
		},
		"bad time":       func(r *models.CreateAgreementRequest) { s := "6pm"; r.VisitStartTime = &s },
		"negative price": func(r *models.CreateAgreementRequest) { r.PricePerPeriod = decimal.NewFromInt(-1) },
		"weekly without days": func(r *models.CreateAgreementRequest) {
			r.AgreementScheduleInput = models.AgreementScheduleInput{Frequency: models.FrequencyWeekly}
		},
		"monthly without day": func(r *models.CreateAgreementRequest) {
			r.AgreementScheduleInput = models.AgreementScheduleInput{Frequency: models.FrequencyMonthly}
		},
	} {
		t.Run(name, func(t *testing.T) {
			r := base()
			mutate(r)
			_, err := f.svc.Create(f.ctxA, r, f.userA)
			require.Error(t, err)
			assert.Contains(t, err.Error(), "Invalid agreement")
		})
	}
}

// expectedVisitDates is the set of visit dates generator runs from first to
// last should have produced (each run covers its local today + 13 days).
func expectedVisitDates(a *models.ServiceAgreement, first, last time.Time, loc *time.Location) map[time.Time]bool {
	out := map[time.Time]bool{}
	for _, d := range a.VisitDates(models.CivilDate(first, loc), models.CivilDate(last, loc).AddDate(0, 0, AgreementHorizonDays-1)) {
		out[d] = true
	}
	return out
}

func runHourly(t *testing.T, f *agreementFixture, from, to time.Time) {
	t.Helper()
	for now := from; !now.After(to); now = now.Add(time.Hour) {
		_, err := GenerateAgreementVisits(context.Background(), f.db, now)
		require.NoError(t, err, "run at %s", now)
	}
}

// Hourly runs through both DST changes (and a month end) create every visit
// exactly once, at 18:00 local.
func TestGenerateAgreementVisits_IdempotentAcrossDST(t *testing.T) {
	daily := 7
	for name, span := range map[string][2]time.Time{
		"spring forward": {time.Date(2026, 3, 6, 0, 0, 0, 0, time.UTC), time.Date(2026, 3, 10, 0, 0, 0, 0, time.UTC)},
		"fall back":      {time.Date(2026, 10, 30, 0, 0, 0, 0, time.UTC), time.Date(2026, 11, 3, 0, 0, 0, 0, time.UTC)},
		"leap month end": {time.Date(2028, 2, 27, 0, 0, 0, 0, time.UTC), time.Date(2028, 3, 2, 0, 0, 0, 0, time.UTC)},
	} {
		t.Run(name, func(t *testing.T) {
			f := setupAgreements(t)
			for _, sched := range []models.AgreementScheduleInput{
				weekdays,
				{Frequency: models.FrequencyTimesPerWeek, TimesPerWeek: &daily},
				{Frequency: models.FrequencyMonthly, DayOfMonth: intPtr(31)},
			} {
				a := f.create(t, sched)
				runHourly(t, f, span[0], span[1])
				got := f.visitJobs(t, a.ID)
				want := expectedVisitDates(a, span[0], span[1], f.ny)
				assert.Len(t, got, len(want), "%s: no missing or extra dates", sched.Frequency)
				for d, n := range got {
					assert.True(t, want[d], "%s: unexpected visit %s", sched.Frequency, d.Format(models.DateLayout))
					assert.Equal(t, 1, n, "%s: one job for %s", sched.Frequency, d.Format(models.DateLayout))
				}
				var visits int64
				require.NoError(t, f.db.WithContext(f.ctxA).Model(&models.AgreementVisit{}).Where("agreement_id = ?", a.ID).Count(&visits).Error)
				assert.Equal(t, int64(len(want)), visits, "one claim per visit")
			}
		})
	}
}

func intPtr(v int) *int { return &v }

func TestGenerateAgreementVisits_JobDetails(t *testing.T) {
	f := setupAgreements(t)
	a := f.create(t, weekdays)
	now := time.Date(2026, 11, 2, 15, 0, 0, 0, time.UTC) // Monday 10:00 in New York
	res, err := GenerateAgreementVisits(context.Background(), f.db, now)
	require.NoError(t, err)
	assert.Equal(t, 10, res.Created, "Mon–Fri over 14 days")

	again, err := GenerateAgreementVisits(context.Background(), f.db, now)
	require.NoError(t, err)
	assert.Zero(t, again.Created, "second run creates nothing")

	var jobs []models.Job
	require.NoError(t, f.db.WithContext(f.ctxA).Preload("Assignments").Where("agreement_id = ?", a.ID).Find(&jobs).Error)
	require.Len(t, jobs, 10)
	tenantA, _ := tenancy.TenantID(f.ctxA)
	for _, j := range jobs {
		assert.Equal(t, tenantA, j.TenantID)
		assert.Equal(t, f.siteA.ID, *j.SiteID)
		assert.Equal(t, "220 W Spring St", j.ServiceAddress.Street)
		require.Len(t, j.Assignments, 1, "assigned technician")
		assert.Equal(t, f.userA, j.Assignments[0].UserID)
		assert.Equal(t, int64(60*60), *j.ScheduledEndAt-*j.ScheduledStartAt)
		assert.Len(t, jobHistory(t, f.isolationFixture, j.ID), 1, "history row")
	}

	// Agreement visits are not extra work: the NTE rule doesn't apply.
	setCustomerRules(t, f.isolationFixture, f.ctxA, f.custA.ID, map[string]any{"nte_limit": decimal.Zero})
	jobs[0].EstimatedCost = 999
	require.NoError(t, requireNTEApproval(f.ctxA, f.db, &jobs[0]))
}

func TestGenerateAgreementVisits_SkipsPausedAndEnded(t *testing.T) {
	f := setupAgreements(t)
	paused := f.create(t, weekdays)
	_, err := f.svc.SetStatus(f.ctxA, paused.ID, models.AgreementStatusPaused)
	require.NoError(t, err)
	ended := f.create(t, weekdays)
	end := "2026-10-31"
	_, err = f.svc.Update(f.ctxA, ended.ID, &models.UpdateAgreementRequest{EndDate: &end})
	require.NoError(t, err)

	res, err := GenerateAgreementVisits(context.Background(), f.db, time.Date(2026, 11, 2, 15, 0, 0, 0, time.UTC))
	require.NoError(t, err)
	assert.Equal(t, 1, res.Agreements, "only the active one is considered")
	assert.Zero(t, res.Created)
}

func TestAgreementService_CrossTenant(t *testing.T) {
	f := setupAgreements(t)
	a := f.create(t, weekdays)

	_, err := f.svc.Get(f.ctxB, a.ID)
	require.ErrorIs(t, err, ErrAgreementNotFound)
	title := "pwned"
	_, err = f.svc.Update(f.ctxB, a.ID, &models.UpdateAgreementRequest{Title: &title})
	require.ErrorIs(t, err, ErrAgreementNotFound)
	_, err = f.svc.SetStatus(f.ctxB, a.ID, models.AgreementStatusPaused)
	require.ErrorIs(t, err, ErrAgreementNotFound)
	_, err = f.svc.Preview(f.ctxB, a.ID, 14)
	require.ErrorIs(t, err, ErrAgreementNotFound)
	require.ErrorIs(t, f.svc.Delete(f.ctxB, a.ID), ErrAgreementNotFound)
	list, err := f.svc.List(f.ctxB, nil)
	require.NoError(t, err)
	assert.Empty(t, list)

	// Tenant B can't build an agreement on tenant A's customer or site.
	_, err = f.svc.Create(f.ctxB, &models.CreateAgreementRequest{CustomerID: f.custA.ID, SiteID: f.siteA.ID, Title: "x",
		AgreementScheduleInput: weekdays, StartDate: "2026-01-01"}, f.userA)
	require.ErrorIs(t, err, ErrCustomerNotFound)
	_, err = f.svc.Create(f.ctxB, &models.CreateAgreementRequest{CustomerID: f.custB.ID, SiteID: f.siteA.ID, Title: "x",
		AgreementScheduleInput: weekdays, StartDate: "2026-01-01"}, f.userA)
	require.ErrorIs(t, err, ErrSiteCustomerMismatch)

	got, err := f.svc.Get(f.ctxA, a.ID)
	require.NoError(t, err)
	assert.Equal(t, "Nightly cleaning", got.Title)
	assert.Equal(t, models.AgreementStatusActive, got.Status)
}
