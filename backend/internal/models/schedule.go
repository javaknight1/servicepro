package models

import (
	"errors"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

var (
	// ErrScheduleConflict indicates a scheduling conflict
	ErrScheduleConflict = errors.New("schedule conflict detected")

	// ErrInvalidSchedule indicates invalid schedule data
	ErrInvalidSchedule = errors.New("invalid schedule data")

	// ErrRecurringScheduleRequired indicates recurring schedule fields are missing
	ErrRecurringScheduleRequired = errors.New("recurring schedule fields required")
)

// RecurrenceType represents the type of schedule recurrence
type RecurrenceType string

const (
	RecurrenceNone    RecurrenceType = "none"
	RecurrenceDaily   RecurrenceType = "daily"
	RecurrenceWeekly  RecurrenceType = "weekly"
	RecurrenceMonthly RecurrenceType = "monthly"
	RecurrenceYearly  RecurrenceType = "yearly"
	RecurrenceCustom  RecurrenceType = "custom"
)

// DayOfWeek represents days of the week for recurring schedules
type DayOfWeek int

const (
	Sunday DayOfWeek = iota
	Monday
	Tuesday
	Wednesday
	Thursday
	Friday
	Saturday
)

// Schedule represents a job schedule entry
type Schedule struct {
	ID                  uuid.UUID          `json:"id" gorm:"type:uuid;primary_key;default:uuid_generate_v4()"`
	JobID               uuid.UUID          `json:"job_id" gorm:"type:uuid;not null;index"`
	Job                 *Job               `json:"job,omitempty" gorm:"foreignKey:JobID;constraint:OnDelete:CASCADE"`
	Title               string             `json:"title" gorm:"type:varchar(200);not null"`
	Description         string             `json:"description" gorm:"type:text"`
	StartTime           int64              `json:"start_time" gorm:"type:bigint;not null;index"`
	EndTime             int64              `json:"end_time" gorm:"type:bigint;not null;index"`
	AllDay              bool               `json:"all_day" gorm:"default:false"`
	RecurrenceType      RecurrenceType     `json:"recurrence_type" gorm:"type:varchar(50);default:'none'"`
	RecurringScheduleID *uuid.UUID         `json:"recurring_schedule_id,omitempty" gorm:"type:uuid;index"`
	RecurringSchedule   *RecurringSchedule `json:"recurring_schedule,omitempty" gorm:"foreignKey:RecurringScheduleID;constraint:OnDelete:SET NULL"`
	AssignedTechIDs     []uuid.UUID        `json:"assigned_tech_ids" gorm:"type:uuid[]"`
	Location            string             `json:"location" gorm:"type:varchar(200)"`
	Notes               *string            `json:"notes,omitempty" gorm:"type:text"`
	IsConfirmed         bool               `json:"is_confirmed" gorm:"default:false"`
	ConfirmedAt         *int64             `json:"confirmed_at" gorm:"type:bigint"`
	ConfirmedBy         *uuid.UUID         `json:"confirmed_by" gorm:"type:uuid"`
	IsCancelled         bool               `json:"is_cancelled" gorm:"default:false;index"`
	CancelledAt         *int64             `json:"cancelled_at" gorm:"type:bigint"`
	CancelledBy         *uuid.UUID         `json:"cancelled_by" gorm:"type:uuid"`
	CancellationReason  *string            `json:"cancellation_reason" gorm:"type:text"`
	RemindersEnabled    bool               `json:"reminders_enabled" gorm:"default:true"`
	ReminderTime        *int64             `json:"reminder_time" gorm:"type:bigint"`
	Color               *string            `json:"color" gorm:"type:varchar(7)"` // Hex color code
	CreatedBy           uuid.UUID          `json:"created_by" gorm:"type:uuid;not null"`
	UpdatedBy           *uuid.UUID         `json:"updated_by" gorm:"type:uuid"`
	CreatedAt           time.Time          `json:"created_at"`
	UpdatedAt           time.Time          `json:"updated_at"`
	DeletedAt           gorm.DeletedAt     `json:"-" gorm:"index"`
}

// TableName specifies the table name for Schedule model
func (Schedule) TableName() string {
	return "schedules"
}

// Validate validates the schedule data
func (s *Schedule) Validate() error {
	if s.Title == "" {
		return errors.New("title is required")
	}

	if s.EndTime <= s.StartTime {
		return errors.New("end time must be after start time")
	}

	if s.StartTime == 0 || s.EndTime == 0 {
		return errors.New("start time and end time are required")
	}

	return nil
}

// Duration returns the duration of the schedule in minutes
func (s *Schedule) Duration() int {
	return int(s.EndTime-s.StartTime) / 60
}

// IsOverlapping checks if this schedule overlaps with another
func (s *Schedule) IsOverlapping(other *Schedule) bool {
	return s.StartTime < other.EndTime && s.EndTime > other.StartTime
}

// RecurringSchedule represents a recurring schedule pattern
type RecurringSchedule struct {
	ID              uuid.UUID      `json:"id" gorm:"type:uuid;primary_key;default:uuid_generate_v4()"`
	Title           string         `json:"title" gorm:"type:varchar(200);not null"`
	Description     string         `json:"description" gorm:"type:text"`
	RecurrenceType  RecurrenceType `json:"recurrence_type" gorm:"type:varchar(50);not null"`
	RecurrenceRule  string         `json:"recurrence_rule" gorm:"type:text"` // RFC 5545 RRULE format
	StartDate       int64          `json:"start_date" gorm:"type:bigint;not null"`
	EndDate         *int64         `json:"end_date" gorm:"type:bigint"`
	Interval        int            `json:"interval" gorm:"default:1"`               // e.g., every 2 weeks
	DaysOfWeek      []int          `json:"days_of_week" gorm:"type:int[]"`          // For weekly recurrence
	DayOfMonth      *int           `json:"day_of_month"`                            // For monthly recurrence
	MonthOfYear     *int           `json:"month_of_year"`                           // For yearly recurrence
	Occurrences     *int           `json:"occurrences"`                             // Max number of occurrences
	TimeStart       int            `json:"time_start" gorm:"type:integer;not null"` // Seconds from midnight
	TimeEnd         int            `json:"time_end" gorm:"type:integer;not null"`   // Seconds from midnight
	Duration        int            `json:"duration"`                                // in minutes
	AssignedTechIDs []uuid.UUID    `json:"assigned_tech_ids" gorm:"type:uuid[]"`
	Location        string         `json:"location" gorm:"type:varchar(200)"`
	JobTemplateID   *uuid.UUID     `json:"job_template_id" gorm:"type:uuid"`
	IsActive        bool           `json:"is_active" gorm:"default:true;index"`
	CreatedBy       uuid.UUID      `json:"created_by" gorm:"type:uuid;not null"`
	UpdatedBy       *uuid.UUID     `json:"updated_by" gorm:"type:uuid"`
	CreatedAt       time.Time      `json:"created_at"`
	UpdatedAt       time.Time      `json:"updated_at"`
	DeletedAt       gorm.DeletedAt `json:"-" gorm:"index"`
}

// TableName specifies the table name for RecurringSchedule model
func (RecurringSchedule) TableName() string {
	return "recurring_schedules"
}

// Validate validates the recurring schedule data
func (rs *RecurringSchedule) Validate() error {
	if rs.Title == "" {
		return errors.New("title is required")
	}

	if rs.RecurrenceType == RecurrenceNone {
		return ErrRecurringScheduleRequired
	}

	if rs.Interval < 1 {
		return errors.New("interval must be at least 1")
	}

	if rs.RecurrenceType == RecurrenceWeekly && len(rs.DaysOfWeek) == 0 {
		return errors.New("days of week required for weekly recurrence")
	}

	return nil
}

// ScheduleConflict represents a detected scheduling conflict
type ScheduleConflict struct {
	ID                  uuid.UUID        `json:"id" gorm:"type:uuid;primary_key;default:uuid_generate_v4()"`
	ScheduleID1         uuid.UUID        `json:"schedule_id_1" gorm:"type:uuid;not null;index"`
	Schedule1           *Schedule        `json:"schedule_1,omitempty" gorm:"foreignKey:ScheduleID1;constraint:OnDelete:CASCADE"`
	ScheduleID2         uuid.UUID        `json:"schedule_id_2" gorm:"type:uuid;not null;index"`
	Schedule2           *Schedule        `json:"schedule_2,omitempty" gorm:"foreignKey:ScheduleID2;constraint:OnDelete:CASCADE"`
	ConflictType        ConflictType     `json:"conflict_type" gorm:"type:varchar(50);not null"`
	ConflictingResource *uuid.UUID       `json:"conflicting_resource" gorm:"type:uuid"` // Tech ID, Equipment ID, etc.
	Severity            ConflictSeverity `json:"severity" gorm:"type:varchar(50);not null"`
	Description         string           `json:"description" gorm:"type:text"`
	IsResolved          bool             `json:"is_resolved" gorm:"default:false;index"`
	ResolvedAt          *int64           `json:"resolved_at" gorm:"type:bigint"`
	ResolvedBy          *uuid.UUID       `json:"resolved_by" gorm:"type:uuid"`
	ResolutionNotes     *string          `json:"resolution_notes" gorm:"type:text"`
	DetectedAt          int64            `json:"detected_at" gorm:"type:bigint;not null"`
	CreatedAt           time.Time        `json:"created_at"`
	UpdatedAt           time.Time        `json:"updated_at"`
}

// ConflictType represents the type of scheduling conflict
type ConflictType string

const (
	ConflictTypeTechnicianOverlap ConflictType = "technician_overlap"
	ConflictTypeLocationOverlap   ConflictType = "location_overlap"
	ConflictTypeEquipmentOverlap  ConflictType = "equipment_overlap"
	ConflictTypeTimeOverlap       ConflictType = "time_overlap"
	ConflictTypeBusinessHours     ConflictType = "business_hours"
	ConflictTypeWorkloadExcess    ConflictType = "workload_excess"
)

// ConflictSeverity represents the severity of a conflict
type ConflictSeverity string

const (
	ConflictSeverityLow      ConflictSeverity = "low"
	ConflictSeverityMedium   ConflictSeverity = "medium"
	ConflictSeverityHigh     ConflictSeverity = "high"
	ConflictSeverityCritical ConflictSeverity = "critical"
)

// TableName specifies the table name for ScheduleConflict model
func (ScheduleConflict) TableName() string {
	return "schedule_conflicts"
}

// ScheduleRequest represents a request to create or update a schedule
type ScheduleRequest struct {
	JobID               uuid.UUID      `json:"job_id" binding:"required"`
	Title               string         `json:"title" binding:"required,min=1,max=200"`
	Description         string         `json:"description"`
	StartTime           int64          `json:"start_time" binding:"required"`
	EndTime             int64          `json:"end_time" binding:"required"`
	AllDay              bool           `json:"all_day"`
	RecurrenceType      RecurrenceType `json:"recurrence_type"`
	RecurringScheduleID *uuid.UUID     `json:"recurring_schedule_id"`
	AssignedTechIDs     []uuid.UUID    `json:"assigned_tech_ids"`
	Location            string         `json:"location" binding:"max=200"`
	Notes               *string        `json:"notes"`
	Color               *string        `json:"color" binding:"omitempty,len=7"` // Hex color
	RemindersEnabled    bool           `json:"reminders_enabled"`
	ReminderTime        *int64         `json:"reminder_time"`
}

// ScheduleResponse represents the response for schedule operations
type ScheduleResponse struct {
	ID                  uuid.UUID      `json:"id"`
	JobID               uuid.UUID      `json:"job_id"`
	JobNumber           string         `json:"job_number"`
	Title               string         `json:"title"`
	Description         string         `json:"description"`
	StartTime           int64          `json:"start_time"`
	EndTime             int64          `json:"end_time"`
	AllDay              bool           `json:"all_day"`
	Duration            int            `json:"duration"` // in minutes
	RecurrenceType      RecurrenceType `json:"recurrence_type"`
	RecurringScheduleID *uuid.UUID     `json:"recurring_schedule_id,omitempty"`
	AssignedTechIDs     []uuid.UUID    `json:"assigned_tech_ids"`
	Location            string         `json:"location"`
	Notes               *string        `json:"notes,omitempty"`
	IsConfirmed         bool           `json:"is_confirmed"`
	IsCancelled         bool           `json:"is_cancelled"`
	Color               *string        `json:"color,omitempty"`
	CreatedAt           time.Time      `json:"created_at"`
	UpdatedAt           time.Time      `json:"updated_at"`
}

// ToResponse converts Schedule to ScheduleResponse
func (s *Schedule) ToResponse() ScheduleResponse {
	resp := ScheduleResponse{
		ID:                  s.ID,
		JobID:               s.JobID,
		Title:               s.Title,
		Description:         s.Description,
		StartTime:           s.StartTime,
		EndTime:             s.EndTime,
		AllDay:              s.AllDay,
		Duration:            s.Duration(),
		RecurrenceType:      s.RecurrenceType,
		RecurringScheduleID: s.RecurringScheduleID,
		AssignedTechIDs:     s.AssignedTechIDs,
		Location:            s.Location,
		Notes:               s.Notes,
		IsConfirmed:         s.IsConfirmed,
		IsCancelled:         s.IsCancelled,
		Color:               s.Color,
		CreatedAt:           s.CreatedAt,
		UpdatedAt:           s.UpdatedAt,
	}

	if s.Job != nil {
		resp.JobNumber = s.Job.JobNumber
	}

	return resp
}

// ScheduleListResponse represents a paginated list of schedules
type ScheduleListResponse struct {
	Schedules []ScheduleResponse `json:"schedules"`
	Total     int64              `json:"total"`
	Page      int                `json:"page"`
	PageSize  int                `json:"page_size"`
}

// ScheduleQueryParams represents query parameters for listing schedules
type ScheduleQueryParams struct {
	StartDate *int64     `form:"start_date"`
	EndDate   *int64     `form:"end_date"`
	TechID    *uuid.UUID `form:"tech_id"`
	Status    *string    `form:"status"`
	Confirmed *bool      `form:"confirmed"`
	Cancelled *bool      `form:"cancelled"`
	Page      int        `form:"page" binding:"min=1"`
	PageSize  int        `form:"page_size" binding:"min=1,max=100"`
}
