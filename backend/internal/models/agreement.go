package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

// AgreementFrequency is how often a service agreement's visits happen (T167).
type AgreementFrequency string

const (
	// FrequencyWeekdays: every Monday to Friday ("nightly cleaning, Mon–Fri").
	FrequencyWeekdays AgreementFrequency = "weekdays"
	// FrequencyWeekly: every week on the given weekdays ("Tue/Fri").
	FrequencyWeekly AgreementFrequency = "weekly"
	// FrequencyTimesPerWeek: N visits a week on evenly spread default days
	// (1: Mon; 2: Mon/Thu; 3: Mon/Wed/Fri; 4: Mon/Tue/Thu/Fri; 5: Mon–Fri;
	// 6: Mon–Sat; 7: every day), or on the given weekdays when exactly N
	// are given.
	FrequencyTimesPerWeek AgreementFrequency = "times_per_week"
	// FrequencyMonthly: once a month on a day of the month; in shorter months
	// a day past the end (29–31) falls on the last day.
	FrequencyMonthly AgreementFrequency = "monthly"
)

// AgreementStatus is whether an agreement generates visits.
type AgreementStatus string

const (
	AgreementStatusActive AgreementStatus = "active"
	AgreementStatusPaused AgreementStatus = "paused"
)

// BillingPeriod says what PricePerPeriod is the price of.
type BillingPeriod string

const (
	BillingPeriodMonth BillingPeriod = "month"
	BillingPeriodVisit BillingPeriod = "visit"
)

// DateLayout is the API form of calendar dates (start/end/visit dates).
const DateLayout = "2006-01-02"

// ServiceAgreement is recurring work at one customer site (T167): a
// frequency, a price and an optional technician. An internal job generates a
// scheduled job per visit (agreement_visits claims each date once).
// Tenant-owned.
type ServiceAgreement struct {
	ID         uuid.UUID `json:"id" gorm:"type:uuid;primary_key;default:uuid_generate_v4()"`
	TenantID   uuid.UUID `json:"-" gorm:"type:uuid;not null;index"`
	CustomerID uuid.UUID `json:"customer_id" gorm:"type:uuid;not null"`
	SiteID     uuid.UUID `json:"site_id" gorm:"type:uuid;not null"`
	Title      string    `json:"title" gorm:"type:varchar(200);not null"`

	Frequency AgreementFrequency `json:"frequency" gorm:"type:varchar(20);not null"`
	// WeekdayMask has bit d set for weekday d (0 = Sunday … 6 = Saturday).
	WeekdayMask  int  `json:"-" gorm:"column:weekday_mask;type:smallint;not null"`
	TimesPerWeek *int `json:"times_per_week" gorm:"type:smallint"`
	DayOfMonth   *int `json:"day_of_month" gorm:"type:smallint"`

	// StartDate and EndDate are calendar dates (midnight UTC; DATE columns).
	StartDate time.Time  `json:"-" gorm:"type:date;not null"`
	EndDate   *time.Time `json:"-" gorm:"type:date"`
	// VisitStartMinutes is the visit's local start time, in minutes after
	// midnight in the tenant's time zone.
	VisitStartMinutes    int `json:"-" gorm:"not null"`
	VisitDurationMinutes int `json:"visit_duration_minutes" gorm:"not null"`

	PricePerPeriod decimal.Decimal `json:"price_per_period" gorm:"type:numeric(12,2);not null" swaggertype:"string" example:"2400.00"`
	BillingPeriod  BillingPeriod   `json:"billing_period" gorm:"type:varchar(10);not null"`

	AssignedTechnicianID *uuid.UUID      `json:"assigned_technician_id" gorm:"type:uuid"`
	JobType              JobType         `json:"job_type" gorm:"type:varchar(50);not null"`
	Status               AgreementStatus `json:"status" gorm:"type:varchar(10);not null"`
	Notes                *string         `json:"notes" gorm:"type:text"`
	// PONumber is the customer's PO for the service (T187); generated visits
	// carry it to their jobs and consolidated invoices use it.
	PONumber *string `json:"po_number" gorm:"column:po_number;type:varchar(100)"`

	CreatedBy uuid.UUID      `json:"created_by" gorm:"type:uuid;not null"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index"`
}

// TableName specifies the table name for ServiceAgreement.
func (ServiceAgreement) TableName() string { return "service_agreements" }

// AgreementVisit claims one visit date of an agreement (unique per agreement
// and date), so the generator creates each visit's job exactly once.
type AgreementVisit struct {
	ID          uuid.UUID  `json:"id" gorm:"type:uuid;primary_key"`
	TenantID    uuid.UUID  `json:"-" gorm:"type:uuid;not null"`
	AgreementID uuid.UUID  `json:"agreement_id" gorm:"type:uuid;not null"`
	VisitDate   time.Time  `json:"visit_date" gorm:"type:date;not null"`
	JobID       *uuid.UUID `json:"job_id" gorm:"type:uuid"`
	CreatedAt   time.Time  `json:"created_at"`
}

// TableName specifies the table name for AgreementVisit.
func (AgreementVisit) TableName() string { return "agreement_visits" }

// AgreementScheduleInput is an agreement's frequency in API form.
type AgreementScheduleInput struct {
	Frequency AgreementFrequency `json:"frequency" binding:"required,oneof=weekdays weekly times_per_week monthly" enums:"weekdays,weekly,times_per_week,monthly"`
	// Weekdays (0 = Sunday … 6 = Saturday): required for weekly; optional
	// for times_per_week (exactly times_per_week of them).
	Weekdays []int `json:"weekdays" binding:"omitempty,dive,min=0,max=6"`
	// TimesPerWeek (1–7): required for times_per_week.
	TimesPerWeek *int `json:"times_per_week" binding:"omitempty,min=1,max=7"`
	// DayOfMonth (1–31): required for monthly.
	DayOfMonth *int `json:"day_of_month" binding:"omitempty,min=1,max=31"`
}

// CreateAgreementRequest is POST /agreements.
type CreateAgreementRequest struct {
	CustomerID uuid.UUID `json:"customer_id" binding:"required"`
	SiteID     uuid.UUID `json:"site_id" binding:"required"`
	Title      string    `json:"title" binding:"required,max=200"`
	AgreementScheduleInput
	// StartDate / EndDate are YYYY-MM-DD (end optional, inclusive).
	StartDate string  `json:"start_date" binding:"required" example:"2026-11-02"`
	EndDate   *string `json:"end_date" example:"2027-10-31"`
	// VisitStartTime is the local start time, HH:MM (24-hour) in the
	// tenant's time zone. Default 18:00.
	VisitStartTime       *string         `json:"visit_start_time" example:"18:00"`
	VisitDurationMinutes *int            `json:"visit_duration_minutes" binding:"omitempty,min=1,max=1440"`
	PricePerPeriod       decimal.Decimal `json:"price_per_period" swaggertype:"string" example:"2400.00"`
	// BillingPeriod: what the price is for — "month" (default) or "visit".
	BillingPeriod        BillingPeriod `json:"billing_period" binding:"omitempty,oneof=month visit" enums:"month,visit"`
	AssignedTechnicianID *uuid.UUID    `json:"assigned_technician_id"`
	JobType              JobType       `json:"job_type" binding:"omitempty,oneof=installation maintenance repair inspection emergency"`
	Notes                *string       `json:"notes" binding:"omitempty,max=5000"`
	// PONumber is the customer's PO for this service (T187).
	PONumber *string `json:"po_number" binding:"omitempty,max=100"`
}

// UpdateAgreementRequest is PUT /agreements/{id}; omitted fields are left
// unchanged. Schedule changes apply to visits not yet generated; jobs
// already created for upcoming visits stay (edit or cancel them).
type UpdateAgreementRequest struct {
	Title *string `json:"title" binding:"omitempty,min=1,max=200"`
	// Schedule replaces the whole frequency when present.
	Schedule             *AgreementScheduleInput `json:"schedule"`
	StartDate            *string                 `json:"start_date" example:"2026-11-02"`
	EndDate              *string                 `json:"end_date" example:"2027-10-31"`
	ClearEndDate         bool                    `json:"clear_end_date"`
	VisitStartTime       *string                 `json:"visit_start_time" example:"18:00"`
	VisitDurationMinutes *int                    `json:"visit_duration_minutes" binding:"omitempty,min=1,max=1440"`
	PricePerPeriod       *decimal.Decimal        `json:"price_per_period" swaggertype:"string" example:"2400.00"`
	BillingPeriod        *BillingPeriod          `json:"billing_period" binding:"omitempty,oneof=month visit" enums:"month,visit"`
	AssignedTechnicianID *uuid.UUID              `json:"assigned_technician_id"`
	ClearTechnician      bool                    `json:"clear_technician"`
	JobType              *JobType                `json:"job_type" binding:"omitempty,oneof=installation maintenance repair inspection emergency"`
	Notes                *string                 `json:"notes" binding:"omitempty,max=5000"`
	// PONumber replaces the PO when present; "" clears it (T187).
	PONumber *string `json:"po_number" binding:"omitempty,max=100"`
}

// AgreementResponse is an agreement as the API returns it.
type AgreementResponse struct {
	ID                   uuid.UUID          `json:"id"`
	CustomerID           uuid.UUID          `json:"customer_id"`
	SiteID               uuid.UUID          `json:"site_id"`
	Title                string             `json:"title"`
	Frequency            AgreementFrequency `json:"frequency"`
	Weekdays             []int              `json:"weekdays"`
	TimesPerWeek         *int               `json:"times_per_week,omitempty"`
	DayOfMonth           *int               `json:"day_of_month,omitempty"`
	StartDate            string             `json:"start_date" example:"2026-11-02"`
	EndDate              *string            `json:"end_date,omitempty" example:"2027-10-31"`
	VisitStartTime       string             `json:"visit_start_time" example:"18:00"`
	VisitDurationMinutes int                `json:"visit_duration_minutes"`
	PricePerPeriod       decimal.Decimal    `json:"price_per_period" swaggertype:"string" example:"2400.00"`
	BillingPeriod        BillingPeriod      `json:"billing_period"`
	AssignedTechnicianID *uuid.UUID         `json:"assigned_technician_id,omitempty"`
	JobType              JobType            `json:"job_type"`
	Status               AgreementStatus    `json:"status"`
	Notes                *string            `json:"notes,omitempty"`
	PONumber             *string            `json:"po_number,omitempty"`
	CreatedAt            time.Time          `json:"created_at"`
	UpdatedAt            time.Time          `json:"updated_at"`
}

// AgreementListResponse is GET /agreements.
type AgreementListResponse struct {
	Agreements []AgreementResponse `json:"agreements"`
	Total      int                 `json:"total"`
}

// AgreementListFilter narrows GET /agreements.
type AgreementListFilter struct {
	CustomerID *uuid.UUID       `form:"customer_id"`
	SiteID     *uuid.UUID       `form:"site_id"`
	Status     *AgreementStatus `form:"status" binding:"omitempty,oneof=active paused"`
}

// AgreementVisitPreview is one upcoming visit.
type AgreementVisitPreview struct {
	Date     string    `json:"date" example:"2026-11-02"`
	StartsAt time.Time `json:"starts_at"`
	EndsAt   time.Time `json:"ends_at"`
}

// AgreementPreviewResponse is GET /agreements/{id}/preview.
type AgreementPreviewResponse struct {
	AgreementID uuid.UUID               `json:"agreement_id"`
	Timezone    string                  `json:"timezone" example:"America/New_York"`
	Visits      []AgreementVisitPreview `json:"visits"`
}
