package services

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"

	apperrors "github.com/javaknight1/servicepro/backend/internal/errors"
	"github.com/javaknight1/servicepro/backend/internal/models"
	"github.com/javaknight1/servicepro/backend/internal/repository"
	"github.com/javaknight1/servicepro/backend/internal/tenancy"
)

// Service agreement domain errors (T167).
var (
	// ErrAgreementNotFound: no such agreement in the caller's tenant.
	ErrAgreementNotFound = apperrors.NotFound("agreement_not_found", "Agreement not found")
	// ErrTechnicianNotFound: the technician isn't an active member of the tenant.
	ErrTechnicianNotFound = apperrors.NotFound("technician_not_found", "The technician is not an active member of this account")
)

const (
	// AgreementHorizonDays is how far ahead the generator creates visits.
	AgreementHorizonDays = 14
	// maxPreviewDays bounds GET /agreements/{id}/preview.
	maxPreviewDays = 92
	// defaultVisitStart is 18:00 (evening cleaning), in minutes.
	defaultVisitStart = 18 * 60
)

// AgreementServiceInterface is what the agreement handler calls. The tenant
// always comes from ctx.
type AgreementServiceInterface interface {
	List(ctx context.Context, filter *models.AgreementListFilter) ([]models.ServiceAgreement, error)
	Get(ctx context.Context, id uuid.UUID) (*models.ServiceAgreement, error)
	Create(ctx context.Context, req *models.CreateAgreementRequest, userID uuid.UUID) (*models.ServiceAgreement, error)
	Update(ctx context.Context, id uuid.UUID, req *models.UpdateAgreementRequest) (*models.ServiceAgreement, error)
	SetStatus(ctx context.Context, id uuid.UUID, status models.AgreementStatus) (*models.ServiceAgreement, error)
	Delete(ctx context.Context, id uuid.UUID) error
	Preview(ctx context.Context, id uuid.UUID, days int) (*models.AgreementPreviewResponse, error)
}

// AgreementService manages service agreements and generates their visits.
type AgreementService struct {
	db         *gorm.DB
	agreements repository.AgreementRepositoryInterface
}

// NewAgreementService creates an agreement service.
func NewAgreementService(db *gorm.DB, agreements repository.AgreementRepositoryInterface) *AgreementService {
	return &AgreementService{db: db, agreements: agreements}
}

func agreementError(err error) error {
	if errors.Is(err, repository.ErrAgreementNotFound) {
		return ErrAgreementNotFound
	}
	return err
}

func (s *AgreementService) tenant(ctx context.Context) (uuid.UUID, error) {
	return tenancy.MustTenantID(ctx)
}

// List returns the caller's agreements.
func (s *AgreementService) List(ctx context.Context, filter *models.AgreementListFilter) ([]models.ServiceAgreement, error) {
	tenantID, err := s.tenant(ctx)
	if err != nil {
		return nil, err
	}
	return s.agreements.List(ctx, tenantID, filter)
}

// Get returns one of the caller's agreements.
func (s *AgreementService) Get(ctx context.Context, id uuid.UUID) (*models.ServiceAgreement, error) {
	tenantID, err := s.tenant(ctx)
	if err != nil {
		return nil, err
	}
	a, err := s.agreements.Get(ctx, tenantID, id)
	return a, agreementError(err)
}

// requireTechnician checks that userID is an active member of tenantID.
func (s *AgreementService) requireTechnician(ctx context.Context, tenantID, userID uuid.UUID) error {
	var n int64
	if err := s.db.WithContext(ctx).Model(&models.TenantUser{}).
		Where("tenant_id = ? AND user_id = ? AND is_active = ?", tenantID, userID, true).
		Count(&n).Error; err != nil {
		return err
	}
	if n == 0 {
		return ErrTechnicianNotFound
	}
	return nil
}

// applySchedule copies a schedule input onto a.
func applySchedule(a *models.ServiceAgreement, in *models.AgreementScheduleInput) {
	a.Frequency = in.Frequency
	a.WeekdayMask = 0
	a.TimesPerWeek = nil
	a.DayOfMonth = nil
	switch in.Frequency {
	case models.FrequencyWeekly, models.FrequencyTimesPerWeek:
		a.WeekdayMask = models.WeekdayMaskOf(in.Weekdays)
		if in.Frequency == models.FrequencyTimesPerWeek {
			a.TimesPerWeek = in.TimesPerWeek
		}
	case models.FrequencyMonthly:
		a.DayOfMonth = in.DayOfMonth
	}
}

// parseAgreementDates sets start/end from YYYY-MM-DD strings.
func parseAgreementDate(field, value string, fields map[string]string) *time.Time {
	d, err := models.ParseDate(strings.TrimSpace(value))
	if err != nil {
		fields[field] = "must be a date (YYYY-MM-DD)"
		return nil
	}
	return &d
}

func parseVisitStart(value string, fields map[string]string) (int, bool) {
	m, err := models.ParseClock(strings.TrimSpace(value))
	if err != nil {
		fields["visit_start_time"] = "must be HH:MM (24-hour)"
		return 0, false
	}
	return m, true
}

// validateAgreement runs the schedule and money checks, merging fields.
func validateAgreement(a *models.ServiceAgreement, fields map[string]string) error {
	for k, v := range a.ValidateSchedule() {
		fields[k] = v
	}
	if a.PricePerPeriod.IsNegative() {
		fields["price_per_period"] = "must be zero or more"
	} else if a.PricePerPeriod.GreaterThanOrEqual(decimal.New(1, 10)) {
		fields["price_per_period"] = "is too large"
	}
	if len(fields) > 0 {
		return apperrors.Validation("Invalid agreement", fields)
	}
	return nil
}

// Create adds an agreement at one of the customer's active sites.
func (s *AgreementService) Create(ctx context.Context, req *models.CreateAgreementRequest, userID uuid.UUID) (*models.ServiceAgreement, error) {
	tenantID, err := requireTenantCustomer(ctx, s.db, req.CustomerID)
	if err != nil {
		return nil, err
	}
	if _, err := resolveJobSite(ctx, s.db, req.CustomerID, req.SiteID); err != nil {
		return nil, err
	}
	a := &models.ServiceAgreement{
		CustomerID:           req.CustomerID,
		SiteID:               req.SiteID,
		Title:                strings.TrimSpace(req.Title),
		VisitStartMinutes:    defaultVisitStart,
		VisitDurationMinutes: 60,
		PricePerPeriod:       req.PricePerPeriod.Round(2),
		BillingPeriod:        models.BillingPeriodMonth,
		AssignedTechnicianID: req.AssignedTechnicianID,
		JobType:              models.JobTypeMaintenance,
		Status:               models.AgreementStatusActive,
		Notes:                req.Notes,
		CreatedBy:            userID,
	}
	applySchedule(a, &req.AgreementScheduleInput)
	fields := map[string]string{}
	if a.Title == "" {
		fields["title"] = "is required"
	}
	if d := parseAgreementDate("start_date", req.StartDate, fields); d != nil {
		a.StartDate = *d
	}
	if req.EndDate != nil && strings.TrimSpace(*req.EndDate) != "" {
		a.EndDate = parseAgreementDate("end_date", *req.EndDate, fields)
	}
	if req.VisitStartTime != nil {
		if m, ok := parseVisitStart(*req.VisitStartTime, fields); ok {
			a.VisitStartMinutes = m
		}
	}
	if req.VisitDurationMinutes != nil {
		a.VisitDurationMinutes = *req.VisitDurationMinutes
	}
	if req.BillingPeriod != "" {
		a.BillingPeriod = req.BillingPeriod
	}
	if req.JobType != "" {
		a.JobType = req.JobType
	}
	if err := validateAgreement(a, fields); err != nil {
		return nil, err
	}
	if a.AssignedTechnicianID != nil {
		if err := s.requireTechnician(ctx, tenantID, *a.AssignedTechnicianID); err != nil {
			return nil, err
		}
	}
	if err := s.agreements.Create(ctx, tenantID, a); err != nil {
		return nil, err
	}
	return s.Get(ctx, a.ID)
}

// Update changes the fields present in req. Jobs already generated for
// upcoming visits are left as they are.
func (s *AgreementService) Update(ctx context.Context, id uuid.UUID, req *models.UpdateAgreementRequest) (*models.ServiceAgreement, error) {
	tenantID, err := s.tenant(ctx)
	if err != nil {
		return nil, err
	}
	a, err := s.agreements.Get(ctx, tenantID, id)
	if err != nil {
		return nil, agreementError(err)
	}
	fields := map[string]string{}
	if req.Title != nil {
		a.Title = strings.TrimSpace(*req.Title)
		if a.Title == "" {
			fields["title"] = "is required"
		}
	}
	if req.Schedule != nil {
		applySchedule(a, req.Schedule)
	}
	if req.StartDate != nil {
		if d := parseAgreementDate("start_date", *req.StartDate, fields); d != nil {
			a.StartDate = *d
		}
	}
	switch {
	case req.ClearEndDate:
		a.EndDate = nil
	case req.EndDate != nil:
		a.EndDate = parseAgreementDate("end_date", *req.EndDate, fields)
	}
	if req.VisitStartTime != nil {
		if m, ok := parseVisitStart(*req.VisitStartTime, fields); ok {
			a.VisitStartMinutes = m
		}
	}
	if req.VisitDurationMinutes != nil {
		a.VisitDurationMinutes = *req.VisitDurationMinutes
	}
	if req.PricePerPeriod != nil {
		a.PricePerPeriod = req.PricePerPeriod.Round(2)
	}
	if req.BillingPeriod != nil {
		a.BillingPeriod = *req.BillingPeriod
	}
	switch {
	case req.ClearTechnician:
		a.AssignedTechnicianID = nil
	case req.AssignedTechnicianID != nil:
		if err := s.requireTechnician(ctx, tenantID, *req.AssignedTechnicianID); err != nil {
			return nil, err
		}
		a.AssignedTechnicianID = req.AssignedTechnicianID
	}
	if req.JobType != nil {
		a.JobType = *req.JobType
	}
	if req.Notes != nil {
		a.Notes = req.Notes
	}
	if err := validateAgreement(a, fields); err != nil {
		return nil, err
	}
	if err := s.agreements.Update(ctx, tenantID, a); err != nil {
		return nil, agreementError(err)
	}
	return s.Get(ctx, id)
}

// SetStatus pauses (no new visits are generated) or resumes an agreement.
func (s *AgreementService) SetStatus(ctx context.Context, id uuid.UUID, status models.AgreementStatus) (*models.ServiceAgreement, error) {
	tenantID, err := s.tenant(ctx)
	if err != nil {
		return nil, err
	}
	if err := s.agreements.SetStatus(ctx, tenantID, id, status); err != nil {
		return nil, agreementError(err)
	}
	return s.Get(ctx, id)
}

// Delete removes an agreement; jobs it generated stay.
func (s *AgreementService) Delete(ctx context.Context, id uuid.UUID) error {
	tenantID, err := s.tenant(ctx)
	if err != nil {
		return err
	}
	return agreementError(s.agreements.Delete(ctx, tenantID, id))
}

// tenantLocation returns tenantID's time zone (UTC when unset or unknown).
func tenantLocation(ctx context.Context, db *gorm.DB, tenantID uuid.UUID) (*time.Location, string, error) {
	var tenant models.Tenant
	if err := db.WithContext(ctx).Select("id", "settings").First(&tenant, "id = ?", tenantID).Error; err != nil {
		return nil, "", err
	}
	name := tenant.Settings.Timezone
	loc := loadLocationOrUTC(name)
	return loc, loc.String(), nil
}

// visitPreviews lists a's visits from today (in loc) for days days.
func visitPreviews(a *models.ServiceAgreement, now time.Time, loc *time.Location, days int) []models.AgreementVisitPreview {
	today := models.CivilDate(now, loc)
	out := []models.AgreementVisitPreview{}
	for _, d := range a.VisitDates(today, today.AddDate(0, 0, days-1)) {
		start := a.VisitStart(d, loc)
		out = append(out, models.AgreementVisitPreview{
			Date:     d.Format(models.DateLayout),
			StartsAt: start,
			EndsAt:   start.Add(time.Duration(a.VisitDurationMinutes) * time.Minute),
		})
	}
	return out
}

// Preview lists the agreement's next visits over the coming days (default
// 14), in the tenant's time zone. A paused agreement has none.
func (s *AgreementService) Preview(ctx context.Context, id uuid.UUID, days int) (*models.AgreementPreviewResponse, error) {
	if days <= 0 {
		days = AgreementHorizonDays
	}
	if days > maxPreviewDays {
		days = maxPreviewDays
	}
	a, err := s.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	loc, name, err := tenantLocation(ctx, s.db, a.TenantID)
	if err != nil {
		return nil, err
	}
	resp := &models.AgreementPreviewResponse{AgreementID: a.ID, Timezone: name, Visits: []models.AgreementVisitPreview{}}
	if a.Status == models.AgreementStatusActive {
		resp.Visits = visitPreviews(a, time.Now(), loc, days)
	}
	return resp, nil
}
