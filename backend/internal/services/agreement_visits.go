package services

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/javaknight1/servicepro/backend/internal/models"
	"github.com/javaknight1/servicepro/backend/internal/tenancy"
	"github.com/javaknight1/servicepro/backend/internal/utils/epoch"
	"github.com/javaknight1/servicepro/backend/pkg/clients/logging"
)

// AgreementVisitsResult is what one generator run did.
type AgreementVisitsResult struct {
	Agreements int `json:"agreements" example:"31"`
	Created    int `json:"created" example:"12"`
	Failed     int `json:"failed" example:"0"`
}

// GenerateAgreementVisits creates a scheduled job for every visit of every
// active service agreement from today through the next 13 days, "today"
// being the tenant's local date (tenant settings time zone; UTC when unset).
//
// It is idempotent and safe to run late, twice or concurrently: each visit
// is claimed by inserting (agreement_id, visit_date) into agreement_visits
// with ON CONFLICT DO NOTHING, and its job is created in the same
// transaction only when the claim was won. A failed visit rolls back its
// claim and is retried on the next run.
func GenerateAgreementVisits(ctx context.Context, db *gorm.DB, now time.Time) (*AgreementVisitsResult, error) {
	sys := tenancy.WithSystem(ctx)
	var agreements []models.ServiceAgreement
	if err := db.WithContext(sys).Where("status = ?", models.AgreementStatusActive).
		Order("tenant_id, id").Find(&agreements).Error; err != nil {
		return nil, fmt.Errorf("list active agreements: %w", err)
	}

	res := &AgreementVisitsResult{Agreements: len(agreements)}
	locs := map[uuid.UUID]*time.Location{}
	var errs []error
	for i := range agreements {
		a := &agreements[i]
		loc, ok := locs[a.TenantID]
		if !ok {
			l, _, err := tenantLocation(sys, db, a.TenantID)
			if err != nil {
				errs = append(errs, fmt.Errorf("agreement %s: tenant time zone: %w", a.ID, err))
				res.Failed++
				continue
			}
			loc, locs[a.TenantID] = l, l
		}
		created, failed, err := generateForAgreement(tenancy.WithTenant(ctx, a.TenantID), db, a, now, loc)
		res.Created += created
		res.Failed += failed
		if err != nil {
			errs = append(errs, err)
		}
	}
	if len(errs) > 0 {
		logging.Error(ctx, "[AGREEMENT-VISITS] some visits failed", map[string]any{"failed": res.Failed, "created": res.Created})
		return res, errors.Join(errs...)
	}
	return res, nil
}

// generateForAgreement creates the missing visits of one agreement.
func generateForAgreement(ctx context.Context, db *gorm.DB, a *models.ServiceAgreement, now time.Time, loc *time.Location) (created, failed int, err error) {
	today := models.CivilDate(now, loc)
	dates := a.VisitDates(today, today.AddDate(0, 0, AgreementHorizonDays-1))
	if len(dates) == 0 {
		return 0, 0, nil
	}
	// The site's address and notes go on the job; a site deleted since the
	// agreement was made still has them (Unscoped).
	var site models.Site
	if err := db.WithContext(ctx).Unscoped().Where("tenant_id = ? AND id = ?", a.TenantID, a.SiteID).First(&site).Error; err != nil {
		return 0, len(dates), fmt.Errorf("agreement %s: load site: %w", a.ID, err)
	}
	var errs []error
	for _, d := range dates {
		ok, err := generateVisit(ctx, db, a, &site, d, loc)
		switch {
		case err != nil:
			failed++
			errs = append(errs, fmt.Errorf("agreement %s visit %s: %w", a.ID, d.Format(models.DateLayout), err))
		case ok:
			created++
		}
	}
	return created, failed, errors.Join(errs...)
}

// generateVisit claims (agreement, date) and creates its scheduled job in
// one transaction. It reports false when the visit already existed.
func generateVisit(ctx context.Context, db *gorm.DB, a *models.ServiceAgreement, site *models.Site, date time.Time, loc *time.Location) (bool, error) {
	created := false
	err := db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		visitID := uuid.New()
		var claimed []uuid.UUID
		// Raw SQL is not rewritten by the tenancy plugin: tenant_id is set
		// explicitly (and the RLS policy checks it against ctx's tenant).
		if err := tx.Raw(`INSERT INTO agreement_visits (id, tenant_id, agreement_id, visit_date, created_at)
			VALUES (?, ?, ?, ?, ?)
			ON CONFLICT (agreement_id, visit_date) DO NOTHING
			RETURNING id`, visitID, a.TenantID, a.ID, date, time.Now().UTC()).Scan(&claimed).Error; err != nil {
			return fmt.Errorf("claim visit: %w", err)
		}
		if len(claimed) == 0 {
			return nil // another run already created this visit
		}

		start := a.VisitStart(date, loc)
		startAt := start.Unix()
		endAt := start.Add(time.Duration(a.VisitDurationMinutes) * time.Minute).Unix()
		agreementID, siteID := a.ID, a.SiteID
		job := &models.Job{
			ID:                  uuid.New(),
			TenantID:            a.TenantID,
			CustomerID:          a.CustomerID,
			SiteID:              &siteID,
			AgreementID:         &agreementID,
			PONumber:            a.PONumber, // the agreement's PO (T187)
			Title:               a.Title,
			JobType:             a.JobType,
			Status:              models.JobStatusScheduled,
			Priority:            models.JobPriorityNormal,
			ScheduledStartAt:    &startAt,
			ScheduledEndAt:      &endAt,
			EstimatedDuration:   a.VisitDurationMinutes,
			ServiceAddress:      site.ServiceAddress(),
			SpecialInstructions: site.AccessNotes,
			CreatedBy:           a.CreatedBy,
		}
		if err := tx.Create(job).Error; err != nil {
			return fmt.Errorf("create job: %w", err)
		}
		note := "Visit generated by service agreement"
		if err := tx.Create(newTransition(job.ID, models.JobStatusNew, models.JobStatusScheduled, models.ReasonScheduleChange, &note, a.CreatedBy)).Error; err != nil {
			return fmt.Errorf("record job history: %w", err)
		}
		if a.AssignedTechnicianID != nil {
			var n int64
			if err := tx.Model(&models.TenantUser{}).
				Where("tenant_id = ? AND user_id = ? AND is_active = ?", a.TenantID, *a.AssignedTechnicianID, true).
				Count(&n).Error; err != nil {
				return err
			}
			// A technician who has left the account is skipped: the visit
			// is still scheduled, unassigned.
			if n > 0 {
				if err := tx.Create(&models.JobAssignment{
					ID: uuid.New(), JobID: job.ID, UserID: *a.AssignedTechnicianID,
					Role: "technician", AssignedAt: epoch.NowEpoch(),
				}).Error; err != nil {
					return fmt.Errorf("assign technician: %w", err)
				}
			}
		}
		if err := tx.Exec(`UPDATE agreement_visits SET job_id = ? WHERE id = ? AND tenant_id = ?`, job.ID, visitID, a.TenantID).Error; err != nil {
			return fmt.Errorf("link visit job: %w", err)
		}
		created = true
		return nil
	})
	return created, err
}
