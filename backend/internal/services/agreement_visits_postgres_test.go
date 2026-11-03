package services

import (
	"context"
	"sync"
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

// T167 on real Postgres (migration 000012): concurrent generator runs claim
// each (agreement, date) once, in the tenant's time zone.
func TestPostgres_GenerateAgreementVisits_ConcurrentRunsDontDuplicate(t *testing.T) {
	db := setupPostgres(t)
	p := seedPGTenant(t, db)
	sys := tenancy.WithSystem(context.Background())
	require.NoError(t, db.WithContext(sys).Exec(
		`UPDATE tenants SET settings = jsonb_set(COALESCE(settings, '{}'::jsonb), '{timezone}', '"America/New_York"') WHERE id = ?`, p.tenantID).Error)
	require.NoError(t, db.WithContext(sys).Exec(
		`INSERT INTO tenant_users (tenant_id, user_id, role_id, is_active) VALUES (?, ?, '00000000-0000-0000-0000-000000000002', true)`,
		p.tenantID, p.userID).Error)

	site := &models.Site{CustomerID: p.customerID, Name: "Lofts", AddressStreet: "220 W Spring St", IsActive: true}
	require.NoError(t, repository.NewSiteRepository(db).Create(p.ctx, p.tenantID, site))
	svc := NewAgreementService(db, repository.NewAgreementRepository(db))
	a, err := svc.Create(p.ctx, &models.CreateAgreementRequest{
		CustomerID: p.customerID, SiteID: site.ID, Title: "Nightly cleaning",
		AgreementScheduleInput: models.AgreementScheduleInput{Frequency: models.FrequencyWeekdays},
		StartDate:              "2026-01-01", PricePerPeriod: decimal.NewFromInt(2400), AssignedTechnicianID: &p.userID,
	}, p.userID)
	require.NoError(t, err)

	// Spring-forward weekend in New York, eight runs at once.
	now := time.Date(2026, 3, 7, 3, 30, 0, 0, time.UTC) // Fri Mar 6, 22:30 local
	var wg sync.WaitGroup
	var mu sync.Mutex
	total := 0
	for range 8 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			res, err := GenerateAgreementVisits(context.Background(), db, now)
			assert.NoError(t, err)
			if res != nil {
				mu.Lock()
				total += res.Created
				mu.Unlock()
			}
		}()
	}
	wg.Wait()

	want := a.VisitDates(time.Date(2026, 3, 6, 0, 0, 0, 0, time.UTC), time.Date(2026, 3, 19, 0, 0, 0, 0, time.UTC))
	assert.Equal(t, len(want), total, "each visit created by exactly one run")

	var rows []struct {
		VisitDate time.Time
		JobID     *uuid.UUID
		Start     int64
	}
	require.NoError(t, db.WithContext(p.ctx).Raw(`
		SELECT v.visit_date, v.job_id, j.scheduled_start_at AS start
		FROM agreement_visits v JOIN jobs j ON j.id = v.job_id
		WHERE v.agreement_id = ? ORDER BY v.visit_date`, a.ID).Scan(&rows).Error)
	require.Len(t, rows, len(want))
	ny, _ := time.LoadLocation("America/New_York")
	for i, r := range rows {
		assert.Equal(t, want[i], r.VisitDate.UTC())
		start := time.Unix(r.Start, 0).In(ny)
		assert.Equal(t, "18:00", start.Format("15:04"), "local start on %s", want[i].Format(models.DateLayout))
		assert.Equal(t, want[i], models.CivilDate(start, ny))
	}
	var jobs int64
	require.NoError(t, db.WithContext(p.ctx).Model(&models.Job{}).Where("agreement_id = ?", a.ID).Count(&jobs).Error)
	assert.Equal(t, int64(len(want)), jobs, "no orphan jobs")
}
