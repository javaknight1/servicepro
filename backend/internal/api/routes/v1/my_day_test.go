package v1

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/javaknight1/servicepro/backend/internal/models"
	"github.com/javaknight1/servicepro/backend/internal/tenancy"
)

// T173: GET /jobs/my-day end to end on Postgres — the caller's assigned
// visits on one tenant-local day, in time order, with sites and no money.

type myDayFixture struct {
	*techFixture
	loc  *time.Location
	site uuid.UUID
}

func setupMyDayFixture(t *testing.T) *myDayFixture {
	t.Helper()
	f := &myDayFixture{techFixture: setupTechFixture(t)}
	var err error
	f.loc, err = time.LoadLocation("America/Chicago")
	require.NoError(t, err)
	sys := tenancy.WithSystem(context.Background())
	require.NoError(t, f.db.WithContext(sys).Exec(
		`UPDATE tenants SET settings = '{"timezone":"America/Chicago"}' WHERE id = ?`, f.tenantA).Error)
	f.site = uuid.New()
	require.NoError(t, f.db.WithContext(sys).Exec(
		`INSERT INTO service_locations (id, tenant_id, customer_id, name, address_street, address_city, address_state, address_zip,
			access_notes, contact_name, contact_phone)
		 VALUES (?, ?, ?, 'Elm Street Plaza', '410 Elm St', 'Springfield', 'IL', '62701', 'Key code 4471', 'Ray Burke', '555-0100')`,
		f.site, f.tenantA, f.customerA).Error)
	return f
}

// visit creates a job in tenantA starting at local (Chicago) time, assigned
// to assignee unless nil.
func (f *myDayFixture) visit(t *testing.T, number string, local time.Time, status models.JobStatus, assignee *uuid.UUID, site *uuid.UUID) *models.Job {
	t.Helper()
	ctx := tenancy.WithTenant(context.Background(), f.tenantA)
	start := local.Unix()
	end := local.Add(time.Hour).Unix()
	job := &models.Job{
		ID: uuid.New(), TenantID: f.tenantA, CustomerID: f.customerA, JobNumber: number, SiteID: site,
		Title: "Visit " + number, Description: "Vacuum lobby; empty trash", JobType: models.JobTypeMaintenance,
		Status: status, Priority: models.JobPriorityNormal, EstimatedCost: 480,
		ScheduledStartAt: &start, ScheduledEndAt: &end, CreatedBy: f.ownerA,
	}
	require.NoError(t, f.db.WithContext(ctx).Omit("Customer").Create(job).Error)
	if assignee != nil {
		require.NoError(t, f.db.WithContext(ctx).Create(&models.JobAssignment{
			JobID: job.ID, UserID: *assignee, Role: "technician", AssignedAt: time.Now().Unix(),
		}).Error)
	}
	return job
}

func (f *myDayFixture) at(day, hour, minute int) time.Time {
	return time.Date(2026, time.November, day, hour, minute, 0, 0, f.loc)
}

func decodeDay(t *testing.T, body []byte) models.MyDayResponse {
	t.Helper()
	var day models.MyDayResponse
	require.NoError(t, json.Unmarshal(body, &day))
	return day
}

func visitNumbers(day models.MyDayResponse) []string {
	out := []string{}
	for _, v := range day.Visits {
		out = append(out, v.JobNumber)
	}
	return out
}

func TestPostgresMyDayListsTheCallersVisitsInLocalDayOrder(t *testing.T) {
	f := setupMyDayFixture(t)
	tech := f.techA

	// Created out of order; only MD-1 and MD-2 are techA's on Nov 3 (Chicago).
	f.visit(t, "MD-2", f.at(3, 14, 0), models.JobStatusScheduled, &tech, nil)
	f.visit(t, "MD-1", f.at(3, 8, 0), models.JobStatusInProgress, &tech, &f.site)
	f.visit(t, "MD-PREV", f.at(2, 23, 30), models.JobStatusScheduled, &tech, nil)    // Nov 2 local (Nov 3 UTC)
	f.visit(t, "MD-NEXT", f.at(4, 0, 30), models.JobStatusScheduled, &tech, nil)     // Nov 4 local
	f.visit(t, "MD-CXL", f.at(3, 10, 0), models.JobStatusCancelled, &tech, nil)      // cancelled
	f.visit(t, "MD-OTHER", f.at(3, 9, 0), models.JobStatusScheduled, &f.techA2, nil) // someone else's
	f.visit(t, "MD-NONE", f.at(3, 11, 0), models.JobStatusScheduled, nil, nil)       // unassigned
	gone := f.visit(t, "MD-GONE", f.at(3, 12, 0), models.JobStatusScheduled, &tech, nil)
	sys := tenancy.WithSystem(context.Background())
	require.NoError(t, f.db.WithContext(sys).Exec(
		`UPDATE job_assignments SET unassigned_at = ? WHERE job_id = ?`, time.Now().Unix(), gone.ID).Error)

	w := f.do(t, tech, f.tenantA, http.MethodGet, "/jobs/my-day?date=2026-11-03", nil)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	day := decodeDay(t, w.Body.Bytes())
	assert.Equal(t, "2026-11-03", day.Date)
	assert.Equal(t, "America/Chicago", day.Timezone)
	require.Equal(t, []string{"MD-1", "MD-2"}, visitNumbers(day))

	first := day.Visits[0]
	require.NotNil(t, first.Site, "the visit carries its site")
	assert.Equal(t, "Elm Street Plaza", first.Site.Name)
	assert.Equal(t, "410 Elm St", first.Site.Address.Street)
	require.NotNil(t, first.Site.AccessNotes)
	assert.Equal(t, "Key code 4471", *first.Site.AccessNotes)
	require.NotNil(t, first.Site.ContactPhone)
	assert.Equal(t, "555-0100", *first.Site.ContactPhone)
	assert.Nil(t, day.Visits[1].Site)
	for _, v := range day.Visits {
		assert.Zero(t, v.EstimatedCost, "technicians never see money")
		assert.Zero(t, v.TotalAmount)
	}
	assert.NotContains(t, w.Body.String(), "480")

	// The neighbouring days pick up the visits just outside Nov 3.
	w = f.do(t, tech, f.tenantA, http.MethodGet, "/jobs/my-day?date=2026-11-02", nil)
	require.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, []string{"MD-PREV"}, visitNumbers(decodeDay(t, w.Body.Bytes())))
	w = f.do(t, tech, f.tenantA, http.MethodGet, "/jobs/my-day?date=2026-11-04", nil)
	require.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, []string{"MD-NEXT"}, visitNumbers(decodeDay(t, w.Body.Bytes())))

	// The job page carries the site too.
	w = f.do(t, tech, f.tenantA, http.MethodGet, "/jobs/"+first.ID.String(), nil)
	require.Equal(t, http.StatusOK, w.Code)
	var job models.JobResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &job))
	require.NotNil(t, job.Site)
	assert.Equal(t, "Elm Street Plaza", job.Site.Name)
}

func TestPostgresMyDayIsOnlyTheCallersOwnAssignments(t *testing.T) {
	f := setupMyDayFixture(t)
	f.visit(t, "MD-T1", f.at(3, 8, 0), models.JobStatusScheduled, &f.techA, nil)
	f.visit(t, "MD-T2", f.at(3, 9, 0), models.JobStatusScheduled, &f.techA2, nil)
	f.visit(t, "MD-M", f.at(3, 10, 0), models.JobStatusScheduled, &f.officeA, nil)

	w := f.do(t, f.techA2, f.tenantA, http.MethodGet, "/jobs/my-day?date=2026-11-03", nil)
	require.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, []string{"MD-T2"}, visitNumbers(decodeDay(t, w.Body.Bytes())))

	// Office staff see every job elsewhere, but My Day is still only theirs.
	w = f.do(t, f.officeA, f.tenantA, http.MethodGet, "/jobs/my-day?date=2026-11-03", nil)
	require.Equal(t, http.StatusOK, w.Code)
	day := decodeDay(t, w.Body.Bytes())
	assert.Equal(t, []string{"MD-M"}, visitNumbers(day))
	assert.Equal(t, 480.0, day.Visits[0].EstimatedCost, "office staff keep prices")
}

func TestPostgresMyDayCrossTenant(t *testing.T) {
	f := setupMyDayFixture(t)
	f.visit(t, "MD-A", f.at(3, 8, 0), models.JobStatusScheduled, &f.techA, &f.site)

	// Tenant B's owner sees nothing of tenant A's day.
	w := f.do(t, f.ownerB, f.tenantB, http.MethodGet, "/jobs/my-day?date=2026-11-03", nil)
	require.Equal(t, http.StatusOK, w.Code)
	day := decodeDay(t, w.Body.Bytes())
	assert.Empty(t, day.Visits)
	assert.Equal(t, "UTC", day.Timezone, "tenant B has no time zone set")

	// A technician of tenant A can't read tenant B's day by switching header.
	w = f.do(t, f.techA, f.tenantB, http.MethodGet, "/jobs/my-day?date=2026-11-03", nil)
	assert.Equal(t, http.StatusForbidden, w.Code)

	// A job pointing at another tenant's site never shows that site.
	sys := tenancy.WithSystem(context.Background())
	var customerB string
	require.NoError(t, f.db.WithContext(sys).Raw(`SELECT id::text FROM customers WHERE tenant_id = ?`, f.tenantB).Row().Scan(&customerB))
	siteB := uuid.New()
	require.NoError(t, f.db.WithContext(sys).Exec(
		`INSERT INTO service_locations (id, tenant_id, customer_id, name, access_notes) VALUES (?, ?, ?, 'B Tower', 'B secret code')`,
		siteB, f.tenantB, customerB).Error)
	f.visit(t, "MD-X", f.at(3, 9, 0), models.JobStatusScheduled, &f.techA, &siteB)
	w = f.do(t, f.techA, f.tenantA, http.MethodGet, "/jobs/my-day?date=2026-11-03", nil)
	require.Equal(t, http.StatusOK, w.Code)
	day = decodeDay(t, w.Body.Bytes())
	require.Equal(t, []string{"MD-A", "MD-X"}, visitNumbers(day))
	assert.NotNil(t, day.Visits[0].Site)
	assert.Nil(t, day.Visits[1].Site, "tenant B's site is not readable from tenant A")
	assert.NotContains(t, w.Body.String(), "B secret code")
}

func TestPostgresMyDayDefaultsToTodayAndRejectsBadDates(t *testing.T) {
	f := setupMyDayFixture(t)

	w := f.do(t, f.techA, f.tenantA, http.MethodGet, "/jobs/my-day", nil)
	require.Equal(t, http.StatusOK, w.Code)
	today := time.Now().In(f.loc).Format(models.DateLayout)
	assert.Equal(t, today, decodeDay(t, w.Body.Bytes()).Date)

	for _, bad := range []string{"2026-13-01", "11/03/2026", "tomorrow"} {
		w = f.do(t, f.techA, f.tenantA, http.MethodGet, "/jobs/my-day?date="+bad, nil)
		assert.Equal(t, http.StatusBadRequest, w.Code, bad)
		assert.Contains(t, w.Body.String(), "invalid_date", bad)
	}
}
