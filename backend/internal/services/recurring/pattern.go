package recurring

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"

	"github.com/javaknight1/servicepro/backend/internal/models"
	"github.com/javaknight1/servicepro/backend/internal/repository"
)

var (
	ErrInvalidPattern    = errors.New("invalid recurring pattern")
	ErrPatternNotFound   = errors.New("recurring pattern not found")
	ErrInvalidRecurrence = errors.New("invalid recurrence configuration")
)

// PatternHandler handles recurring pattern operations
type PatternHandler struct {
	scheduleRepo repository.ScheduleRepositoryInterface
	logger       *log.Logger
}

// NewPatternHandler creates a new pattern handler
func NewPatternHandler(scheduleRepo repository.ScheduleRepositoryInterface) *PatternHandler {
	return &PatternHandler{
		scheduleRepo: scheduleRepo,
		logger:       log.New(log.Writer(), "[PatternHandler] ", log.LstdFlags),
	}
}

// PatternRequest represents a request to create or update a recurring pattern
type PatternRequest struct {
	Title           string                `json:"title" binding:"required"`
	Description     string                `json:"description"`
	RecurrenceType  models.RecurrenceType `json:"recurrence_type" binding:"required"`
	StartDate       time.Time             `json:"start_date" binding:"required"`
	EndDate         *time.Time            `json:"end_date"`
	Interval        int                   `json:"interval" binding:"min=1"`
	DaysOfWeek      []int                 `json:"days_of_week"`
	DayOfMonth      *int                  `json:"day_of_month"`
	MonthOfYear     *int                  `json:"month_of_year"`
	Occurrences     *int                  `json:"occurrences"`
	TimeStart       string                `json:"time_start" binding:"required"`
	TimeEnd         string                `json:"time_end" binding:"required"`
	AssignedTechIDs []uuid.UUID           `json:"assigned_tech_ids"`
	Location        string                `json:"location"`
	JobTemplateID   *uuid.UUID            `json:"job_template_id"`
}

// PatternResponse represents a recurring pattern response
type PatternResponse struct {
	ID              uuid.UUID             `json:"id"`
	Title           string                `json:"title"`
	Description     string                `json:"description"`
	RecurrenceType  models.RecurrenceType `json:"recurrence_type"`
	RecurrenceRule  string                `json:"recurrence_rule"`
	StartDate       time.Time             `json:"start_date"`
	EndDate         *time.Time            `json:"end_date"`
	Interval        int                   `json:"interval"`
	DaysOfWeek      []int                 `json:"days_of_week"`
	DayOfMonth      *int                  `json:"day_of_month"`
	MonthOfYear     *int                  `json:"month_of_year"`
	Occurrences     *int                  `json:"occurrences"`
	TimeStart       string                `json:"time_start"`
	TimeEnd         string                `json:"time_end"`
	Duration        int                   `json:"duration"`
	AssignedTechIDs []uuid.UUID           `json:"assigned_tech_ids"`
	Location        string                `json:"location"`
	JobTemplateID   *uuid.UUID            `json:"job_template_id"`
	IsActive        bool                  `json:"is_active"`
	CreatedAt       time.Time             `json:"created_at"`
	UpdatedAt       time.Time             `json:"updated_at"`
}

// CreatePattern creates a new recurring pattern
func (h *PatternHandler) CreatePattern(ctx context.Context, req *PatternRequest, createdBy uuid.UUID) (*PatternResponse, error) {
	h.logger.Printf("Creating recurring pattern: %s", req.Title)

	// Validate pattern
	if err := h.validatePattern(req); err != nil {
		return nil, err
	}

	// Calculate duration
	duration := h.calculateDuration(req.TimeStart, req.TimeEnd)

	// Generate RRULE
	rrule := h.generateRRule(req)

	// Create recurring schedule model
	recurring := &models.RecurringSchedule{
		Title:           req.Title,
		Description:     req.Description,
		RecurrenceType:  req.RecurrenceType,
		RecurrenceRule:  rrule,
		StartDate:       req.StartDate,
		EndDate:         req.EndDate,
		Interval:        req.Interval,
		DaysOfWeek:      req.DaysOfWeek,
		DayOfMonth:      req.DayOfMonth,
		MonthOfYear:     req.MonthOfYear,
		Occurrences:     req.Occurrences,
		TimeStart:       req.TimeStart,
		TimeEnd:         req.TimeEnd,
		Duration:        duration,
		AssignedTechIDs: req.AssignedTechIDs,
		Location:        req.Location,
		JobTemplateID:   req.JobTemplateID,
		IsActive:        true,
		CreatedBy:       createdBy,
	}

	// Save to database
	if err := h.scheduleRepo.CreateRecurring(recurring); err != nil {
		h.logger.Printf("Failed to create recurring pattern: %v", err)
		return nil, err
	}

	h.logger.Printf("Recurring pattern created: %s", recurring.ID)
	return h.toResponse(recurring), nil
}

// GetPattern retrieves a recurring pattern by ID
func (h *PatternHandler) GetPattern(ctx context.Context, id uuid.UUID) (*PatternResponse, error) {
	recurring, err := h.scheduleRepo.GetRecurringByID(id)
	if err != nil {
		return nil, ErrPatternNotFound
	}

	return h.toResponse(recurring), nil
}

// UpdatePattern updates an existing recurring pattern
func (h *PatternHandler) UpdatePattern(ctx context.Context, id uuid.UUID, req *PatternRequest, updatedBy uuid.UUID) (*PatternResponse, error) {
	h.logger.Printf("Updating recurring pattern: %s", id)

	// Get existing pattern
	recurring, err := h.scheduleRepo.GetRecurringByID(id)
	if err != nil {
		return nil, ErrPatternNotFound
	}

	// Validate pattern
	if err := h.validatePattern(req); err != nil {
		return nil, err
	}

	// Calculate duration
	duration := h.calculateDuration(req.TimeStart, req.TimeEnd)

	// Generate RRULE
	rrule := h.generateRRule(req)

	// Update fields
	recurring.Title = req.Title
	recurring.Description = req.Description
	recurring.RecurrenceType = req.RecurrenceType
	recurring.RecurrenceRule = rrule
	recurring.StartDate = req.StartDate
	recurring.EndDate = req.EndDate
	recurring.Interval = req.Interval
	recurring.DaysOfWeek = req.DaysOfWeek
	recurring.DayOfMonth = req.DayOfMonth
	recurring.MonthOfYear = req.MonthOfYear
	recurring.Occurrences = req.Occurrences
	recurring.TimeStart = req.TimeStart
	recurring.TimeEnd = req.TimeEnd
	recurring.Duration = duration
	recurring.AssignedTechIDs = req.AssignedTechIDs
	recurring.Location = req.Location
	recurring.JobTemplateID = req.JobTemplateID
	recurring.UpdatedBy = &updatedBy

	// Save to database
	if err := h.scheduleRepo.UpdateRecurring(recurring); err != nil {
		h.logger.Printf("Failed to update recurring pattern: %v", err)
		return nil, err
	}

	h.logger.Printf("Recurring pattern updated: %s", id)
	return h.toResponse(recurring), nil
}

// DeletePattern deletes a recurring pattern
func (h *PatternHandler) DeletePattern(ctx context.Context, id uuid.UUID) error {
	h.logger.Printf("Deleting recurring pattern: %s", id)

	// Verify pattern exists
	if _, err := h.scheduleRepo.GetRecurringByID(id); err != nil {
		return ErrPatternNotFound
	}

	if err := h.scheduleRepo.DeleteRecurring(id); err != nil {
		h.logger.Printf("Failed to delete recurring pattern: %v", err)
		return err
	}

	h.logger.Printf("Recurring pattern deleted: %s", id)
	return nil
}

// ActivatePattern activates a recurring pattern
func (h *PatternHandler) ActivatePattern(ctx context.Context, id uuid.UUID) error {
	recurring, err := h.scheduleRepo.GetRecurringByID(id)
	if err != nil {
		return ErrPatternNotFound
	}

	recurring.IsActive = true
	return h.scheduleRepo.UpdateRecurring(recurring)
}

// DeactivatePattern deactivates a recurring pattern
func (h *PatternHandler) DeactivatePattern(ctx context.Context, id uuid.UUID) error {
	recurring, err := h.scheduleRepo.GetRecurringByID(id)
	if err != nil {
		return ErrPatternNotFound
	}

	recurring.IsActive = false
	return h.scheduleRepo.UpdateRecurring(recurring)
}

// ListActivePatterns retrieves all active recurring patterns
func (h *PatternHandler) ListActivePatterns(ctx context.Context) ([]PatternResponse, error) {
	patterns, err := h.scheduleRepo.GetActiveRecurring()
	if err != nil {
		return nil, err
	}

	responses := make([]PatternResponse, len(patterns))
	for i, pattern := range patterns {
		responses[i] = *h.toResponse(&pattern)
	}

	return responses, nil
}

// validatePattern validates a recurring pattern request
func (h *PatternHandler) validatePattern(req *PatternRequest) error {
	// Validate recurrence type
	switch req.RecurrenceType {
	case models.RecurrenceNone:
		return ErrInvalidRecurrence

	case models.RecurrenceDaily:
		// Daily requires interval
		if req.Interval < 1 {
			return fmt.Errorf("daily recurrence requires interval >= 1")
		}

	case models.RecurrenceWeekly:
		// Weekly requires days of week
		if len(req.DaysOfWeek) == 0 {
			return fmt.Errorf("weekly recurrence requires days of week")
		}
		// Validate days of week (0-6)
		for _, day := range req.DaysOfWeek {
			if day < 0 || day > 6 {
				return fmt.Errorf("invalid day of week: %d (must be 0-6)", day)
			}
		}

	case models.RecurrenceMonthly:
		// Monthly requires day of month
		if req.DayOfMonth == nil {
			return fmt.Errorf("monthly recurrence requires day of month")
		}
		if *req.DayOfMonth < 1 || *req.DayOfMonth > 31 {
			return fmt.Errorf("invalid day of month: %d (must be 1-31)", *req.DayOfMonth)
		}

	case models.RecurrenceYearly:
		// Yearly requires month and day
		if req.MonthOfYear == nil {
			return fmt.Errorf("yearly recurrence requires month of year")
		}
		if *req.MonthOfYear < 1 || *req.MonthOfYear > 12 {
			return fmt.Errorf("invalid month of year: %d (must be 1-12)", *req.MonthOfYear)
		}
		if req.DayOfMonth == nil {
			return fmt.Errorf("yearly recurrence requires day of month")
		}
		if *req.DayOfMonth < 1 || *req.DayOfMonth > 31 {
			return fmt.Errorf("invalid day of month: %d (must be 1-31)", *req.DayOfMonth)
		}
	}

	// Validate time format
	if _, err := time.Parse("15:04", req.TimeStart); err != nil {
		return fmt.Errorf("invalid time start format (expected HH:MM): %w", err)
	}
	if _, err := time.Parse("15:04", req.TimeEnd); err != nil {
		return fmt.Errorf("invalid time end format (expected HH:MM): %w", err)
	}

	// Validate end date is after start date
	if req.EndDate != nil && req.EndDate.Before(req.StartDate) {
		return fmt.Errorf("end date must be after start date")
	}

	return nil
}

// calculateDuration calculates duration in minutes
func (h *PatternHandler) calculateDuration(timeStart, timeEnd string) int {
	start, _ := time.Parse("15:04", timeStart)
	end, _ := time.Parse("15:04", timeEnd)
	return int(end.Sub(start).Minutes())
}

// generateRRule generates an RFC 5545 RRULE string
func (h *PatternHandler) generateRRule(req *PatternRequest) string {
	rrule := fmt.Sprintf("FREQ=%s", h.getFrequency(req.RecurrenceType))

	if req.Interval > 1 {
		rrule += fmt.Sprintf(";INTERVAL=%d", req.Interval)
	}

	switch req.RecurrenceType {
	case models.RecurrenceWeekly:
		if len(req.DaysOfWeek) > 0 {
			days := h.formatDaysOfWeek(req.DaysOfWeek)
			rrule += fmt.Sprintf(";BYDAY=%s", days)
		}

	case models.RecurrenceMonthly:
		if req.DayOfMonth != nil {
			rrule += fmt.Sprintf(";BYMONTHDAY=%d", *req.DayOfMonth)
		}

	case models.RecurrenceYearly:
		if req.MonthOfYear != nil {
			rrule += fmt.Sprintf(";BYMONTH=%d", *req.MonthOfYear)
		}
		if req.DayOfMonth != nil {
			rrule += fmt.Sprintf(";BYMONTHDAY=%d", *req.DayOfMonth)
		}
	}

	if req.EndDate != nil {
		rrule += fmt.Sprintf(";UNTIL=%s", req.EndDate.Format("20060102T150405Z"))
	} else if req.Occurrences != nil {
		rrule += fmt.Sprintf(";COUNT=%d", *req.Occurrences)
	}

	return rrule
}

// getFrequency converts recurrence type to RRULE frequency
func (h *PatternHandler) getFrequency(recurrenceType models.RecurrenceType) string {
	switch recurrenceType {
	case models.RecurrenceDaily:
		return "DAILY"
	case models.RecurrenceWeekly:
		return "WEEKLY"
	case models.RecurrenceMonthly:
		return "MONTHLY"
	case models.RecurrenceYearly:
		return "YEARLY"
	default:
		return "DAILY"
	}
}

// formatDaysOfWeek formats days of week for RRULE
func (h *PatternHandler) formatDaysOfWeek(days []int) string {
	dayNames := []string{"SU", "MO", "TU", "WE", "TH", "FR", "SA"}
	result := ""

	for i, day := range days {
		if i > 0 {
			result += ","
		}
		if day >= 0 && day < 7 {
			result += dayNames[day]
		}
	}

	return result
}

// toResponse converts RecurringSchedule to PatternResponse
func (h *PatternHandler) toResponse(recurring *models.RecurringSchedule) *PatternResponse {
	return &PatternResponse{
		ID:              recurring.ID,
		Title:           recurring.Title,
		Description:     recurring.Description,
		RecurrenceType:  recurring.RecurrenceType,
		RecurrenceRule:  recurring.RecurrenceRule,
		StartDate:       recurring.StartDate,
		EndDate:         recurring.EndDate,
		Interval:        recurring.Interval,
		DaysOfWeek:      recurring.DaysOfWeek,
		DayOfMonth:      recurring.DayOfMonth,
		MonthOfYear:     recurring.MonthOfYear,
		Occurrences:     recurring.Occurrences,
		TimeStart:       recurring.TimeStart,
		TimeEnd:         recurring.TimeEnd,
		Duration:        recurring.Duration,
		AssignedTechIDs: recurring.AssignedTechIDs,
		Location:        recurring.Location,
		JobTemplateID:   recurring.JobTemplateID,
		IsActive:        recurring.IsActive,
		CreatedAt:       recurring.CreatedAt,
		UpdatedAt:       recurring.UpdatedAt,
	}
}
