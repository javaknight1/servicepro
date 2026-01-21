package recurring

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
)

// ScheduleException represents an exception to a recurring pattern
type ScheduleException struct {
	ID                 uuid.UUID  `json:"id"`
	RecurringPatternID uuid.UUID  `json:"recurring_pattern_id"`
	ExceptionDate      time.Time  `json:"exception_date"`
	ExceptionType      string     `json:"exception_type"` // skip, reschedule, modify
	Reason             string     `json:"reason"`
	RescheduledDate    *time.Time `json:"rescheduled_date,omitempty"`
	Modifications      *string    `json:"modifications,omitempty"` // JSON field for schedule modifications
	CreatedBy          uuid.UUID  `json:"created_by"`
	CreatedAt          time.Time  `json:"created_at"`
	UpdatedAt          time.Time  `json:"updated_at"`
}

// ExceptionType constants
const (
	ExceptionTypeSkip       = "skip"
	ExceptionTypeReschedule = "reschedule"
	ExceptionTypeModify     = "modify"
)

// ExceptionRequest represents a request to create an exception
type ExceptionRequest struct {
	RecurringPatternID uuid.UUID  `json:"recurring_pattern_id" binding:"required"`
	ExceptionDate      time.Time  `json:"exception_date" binding:"required"`
	ExceptionType      string     `json:"exception_type" binding:"required"`
	Reason             string     `json:"reason" binding:"required"`
	RescheduledDate    *time.Time `json:"rescheduled_date"`
	Modifications      *string    `json:"modifications"`
}

// Holiday represents a holiday in the calendar
type Holiday struct {
	ID          uuid.UUID `json:"id"`
	Name        string    `json:"name"`
	Date        time.Time `json:"date"`
	Country     string    `json:"country"`
	IsRecurring bool      `json:"is_recurring"` // e.g., Christmas
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// HolidayRequest represents a request to create a holiday
type HolidayRequest struct {
	Name        string    `json:"name" binding:"required"`
	Date        time.Time `json:"date" binding:"required"`
	Country     string    `json:"country"`
	IsRecurring bool      `json:"is_recurring"`
}

// ExceptionRepositoryInterface defines the interface for exception repository
type ExceptionRepositoryInterface interface {
	CreateException(exception *ScheduleException) error
	GetExceptionByID(id uuid.UUID) (*ScheduleException, error)
	GetExceptionsByPattern(patternID uuid.UUID, startDate, endDate time.Time) ([]ScheduleException, error)
	UpdateException(exception *ScheduleException) error
	DeleteException(id uuid.UUID) error
	ListExceptions(patternID uuid.UUID) ([]ScheduleException, error)
}

// HolidayRepositoryInterface defines the interface for holiday repository
type HolidayRepositoryInterface interface {
	CreateHoliday(holiday *Holiday) error
	GetHolidayByID(id uuid.UUID) (*Holiday, error)
	GetHolidaysByDateRange(startDate, endDate time.Time) ([]Holiday, error)
	GetHolidaysByCountry(country string) ([]Holiday, error)
	UpdateHoliday(holiday *Holiday) error
	DeleteHoliday(id uuid.UUID) error
	ListHolidays() ([]Holiday, error)
}

// ExceptionHandler handles schedule exceptions
type ExceptionHandler struct {
	exceptionRepo ExceptionRepositoryInterface
	logger        *log.Logger
}

// NewExceptionHandler creates a new exception handler
func NewExceptionHandler(exceptionRepo ExceptionRepositoryInterface) *ExceptionHandler {
	return &ExceptionHandler{
		exceptionRepo: exceptionRepo,
		logger:        log.New(log.Writer(), "[ExceptionHandler] ", log.LstdFlags),
	}
}

// CreateException creates a new schedule exception
func (h *ExceptionHandler) CreateException(ctx context.Context, req *ExceptionRequest, createdBy uuid.UUID) (*ScheduleException, error) {
	h.logger.Printf("Creating exception for pattern %s on %s",
		req.RecurringPatternID, req.ExceptionDate.Format("2006-01-02"))

	// Validate exception type
	if err := h.validateExceptionType(req.ExceptionType); err != nil {
		return nil, err
	}

	// Validate reschedule date if needed
	if req.ExceptionType == ExceptionTypeReschedule && req.RescheduledDate == nil {
		return nil, fmt.Errorf("rescheduled date is required for reschedule exception")
	}

	exception := &ScheduleException{
		ID:                 uuid.New(),
		RecurringPatternID: req.RecurringPatternID,
		ExceptionDate:      req.ExceptionDate,
		ExceptionType:      req.ExceptionType,
		Reason:             req.Reason,
		RescheduledDate:    req.RescheduledDate,
		Modifications:      req.Modifications,
		CreatedBy:          createdBy,
		CreatedAt:          time.Now(),
		UpdatedAt:          time.Now(),
	}

	if err := h.exceptionRepo.CreateException(exception); err != nil {
		h.logger.Printf("Failed to create exception: %v", err)
		return nil, err
	}

	h.logger.Printf("Exception created: %s", exception.ID)
	return exception, nil
}

// GetException retrieves an exception by ID
func (h *ExceptionHandler) GetException(ctx context.Context, id uuid.UUID) (*ScheduleException, error) {
	return h.exceptionRepo.GetExceptionByID(id)
}

// UpdateException updates an existing exception
func (h *ExceptionHandler) UpdateException(ctx context.Context, id uuid.UUID, req *ExceptionRequest) (*ScheduleException, error) {
	h.logger.Printf("Updating exception: %s", id)

	// Get existing exception
	exception, err := h.exceptionRepo.GetExceptionByID(id)
	if err != nil {
		return nil, fmt.Errorf("exception not found")
	}

	// Validate exception type
	if err := h.validateExceptionType(req.ExceptionType); err != nil {
		return nil, err
	}

	// Update fields
	exception.ExceptionDate = req.ExceptionDate
	exception.ExceptionType = req.ExceptionType
	exception.Reason = req.Reason
	exception.RescheduledDate = req.RescheduledDate
	exception.Modifications = req.Modifications
	exception.UpdatedAt = time.Now()

	if err := h.exceptionRepo.UpdateException(exception); err != nil {
		h.logger.Printf("Failed to update exception: %v", err)
		return nil, err
	}

	h.logger.Printf("Exception updated: %s", id)
	return exception, nil
}

// DeleteException deletes an exception
func (h *ExceptionHandler) DeleteException(ctx context.Context, id uuid.UUID) error {
	h.logger.Printf("Deleting exception: %s", id)

	if err := h.exceptionRepo.DeleteException(id); err != nil {
		h.logger.Printf("Failed to delete exception: %v", err)
		return err
	}

	h.logger.Printf("Exception deleted: %s", id)
	return nil
}

// ListExceptions lists all exceptions for a recurring pattern
func (h *ExceptionHandler) ListExceptions(ctx context.Context, patternID uuid.UUID) ([]ScheduleException, error) {
	return h.exceptionRepo.ListExceptions(patternID)
}

// GetExceptionsByDateRange retrieves exceptions within a date range
func (h *ExceptionHandler) GetExceptionsByDateRange(ctx context.Context, patternID uuid.UUID, startDate, endDate time.Time) ([]ScheduleException, error) {
	return h.exceptionRepo.GetExceptionsByPattern(patternID, startDate, endDate)
}

// validateExceptionType validates the exception type
func (h *ExceptionHandler) validateExceptionType(exceptionType string) error {
	switch exceptionType {
	case ExceptionTypeSkip, ExceptionTypeReschedule, ExceptionTypeModify:
		return nil
	default:
		return fmt.Errorf("invalid exception type: %s (must be skip, reschedule, or modify)", exceptionType)
	}
}

// HolidayHandler handles holiday calendar operations
type HolidayHandler struct {
	holidayRepo HolidayRepositoryInterface
	logger      *log.Logger
}

// NewHolidayHandler creates a new holiday handler
func NewHolidayHandler(holidayRepo HolidayRepositoryInterface) *HolidayHandler {
	return &HolidayHandler{
		holidayRepo: holidayRepo,
		logger:      log.New(log.Writer(), "[HolidayHandler] ", log.LstdFlags),
	}
}

// CreateHoliday creates a new holiday
func (h *HolidayHandler) CreateHoliday(ctx context.Context, req *HolidayRequest) (*Holiday, error) {
	h.logger.Printf("Creating holiday: %s on %s", req.Name, req.Date.Format("2006-01-02"))

	holiday := &Holiday{
		ID:          uuid.New(),
		Name:        req.Name,
		Date:        req.Date,
		Country:     req.Country,
		IsRecurring: req.IsRecurring,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	if err := h.holidayRepo.CreateHoliday(holiday); err != nil {
		h.logger.Printf("Failed to create holiday: %v", err)
		return nil, err
	}

	h.logger.Printf("Holiday created: %s", holiday.ID)
	return holiday, nil
}

// GetHoliday retrieves a holiday by ID
func (h *HolidayHandler) GetHoliday(ctx context.Context, id uuid.UUID) (*Holiday, error) {
	return h.holidayRepo.GetHolidayByID(id)
}

// UpdateHoliday updates an existing holiday
func (h *HolidayHandler) UpdateHoliday(ctx context.Context, id uuid.UUID, req *HolidayRequest) (*Holiday, error) {
	h.logger.Printf("Updating holiday: %s", id)

	holiday, err := h.holidayRepo.GetHolidayByID(id)
	if err != nil {
		return nil, fmt.Errorf("holiday not found")
	}

	holiday.Name = req.Name
	holiday.Date = req.Date
	holiday.Country = req.Country
	holiday.IsRecurring = req.IsRecurring
	holiday.UpdatedAt = time.Now()

	if err := h.holidayRepo.UpdateHoliday(holiday); err != nil {
		h.logger.Printf("Failed to update holiday: %v", err)
		return nil, err
	}

	h.logger.Printf("Holiday updated: %s", id)
	return holiday, nil
}

// DeleteHoliday deletes a holiday
func (h *HolidayHandler) DeleteHoliday(ctx context.Context, id uuid.UUID) error {
	h.logger.Printf("Deleting holiday: %s", id)

	if err := h.holidayRepo.DeleteHoliday(id); err != nil {
		h.logger.Printf("Failed to delete holiday: %v", err)
		return err
	}

	h.logger.Printf("Holiday deleted: %s", id)
	return nil
}

// ListHolidays lists all holidays
func (h *HolidayHandler) ListHolidays(ctx context.Context) ([]Holiday, error) {
	return h.holidayRepo.ListHolidays()
}

// GetHolidaysByCountry retrieves holidays for a specific country
func (h *HolidayHandler) GetHolidaysByCountry(ctx context.Context, country string) ([]Holiday, error) {
	return h.holidayRepo.GetHolidaysByCountry(country)
}

// GetHolidaysByDateRange retrieves holidays within a date range
func (h *HolidayHandler) GetHolidaysByDateRange(ctx context.Context, startDate, endDate time.Time) ([]Holiday, error) {
	return h.holidayRepo.GetHolidaysByDateRange(startDate, endDate)
}

// ImportUSHolidays imports common US holidays for a given year
func (h *HolidayHandler) ImportUSHolidays(ctx context.Context, year int) error {
	h.logger.Printf("Importing US holidays for year %d", year)

	holidays := []HolidayRequest{
		{Name: "New Year's Day", Date: time.Date(year, 1, 1, 0, 0, 0, 0, time.UTC), Country: "US", IsRecurring: true},
		{Name: "Independence Day", Date: time.Date(year, 7, 4, 0, 0, 0, 0, time.UTC), Country: "US", IsRecurring: true},
		{Name: "Veterans Day", Date: time.Date(year, 11, 11, 0, 0, 0, 0, time.UTC), Country: "US", IsRecurring: true},
		{Name: "Christmas Day", Date: time.Date(year, 12, 25, 0, 0, 0, 0, time.UTC), Country: "US", IsRecurring: true},
	}

	// Add floating holidays (calculated)
	holidays = append(holidays, h.calculateFloatingHolidays(year)...)

	for _, req := range holidays {
		if _, err := h.CreateHoliday(ctx, &req); err != nil {
			h.logger.Printf("Failed to import holiday %s: %v", req.Name, err)
		}
	}

	h.logger.Printf("US holidays imported for year %d", year)
	return nil
}

// calculateFloatingHolidays calculates floating holidays for a given year
func (h *HolidayHandler) calculateFloatingHolidays(year int) []HolidayRequest {
	holidays := []HolidayRequest{}

	// Memorial Day - Last Monday in May
	memorialDay := h.getLastMonday(year, time.May)
	holidays = append(holidays, HolidayRequest{
		Name:        "Memorial Day",
		Date:        memorialDay,
		Country:     "US",
		IsRecurring: true,
	})

	// Labor Day - First Monday in September
	laborDay := h.getFirstMonday(year, time.September)
	holidays = append(holidays, HolidayRequest{
		Name:        "Labor Day",
		Date:        laborDay,
		Country:     "US",
		IsRecurring: true,
	})

	// Thanksgiving - Fourth Thursday in November
	thanksgiving := h.getFourthThursday(year, time.November)
	holidays = append(holidays, HolidayRequest{
		Name:        "Thanksgiving",
		Date:        thanksgiving,
		Country:     "US",
		IsRecurring: true,
	})

	return holidays
}

// getLastMonday gets the last Monday of a month
func (h *HolidayHandler) getLastMonday(year int, month time.Month) time.Time {
	// Start from last day of month
	lastDay := time.Date(year, month+1, 0, 0, 0, 0, 0, time.UTC)

	// Go backwards to find Monday
	for lastDay.Weekday() != time.Monday {
		lastDay = lastDay.AddDate(0, 0, -1)
	}

	return lastDay
}

// getFirstMonday gets the first Monday of a month
func (h *HolidayHandler) getFirstMonday(year int, month time.Month) time.Time {
	// Start from first day of month
	firstDay := time.Date(year, month, 1, 0, 0, 0, 0, time.UTC)

	// Go forward to find Monday
	for firstDay.Weekday() != time.Monday {
		firstDay = firstDay.AddDate(0, 0, 1)
	}

	return firstDay
}

// getFourthThursday gets the fourth Thursday of a month
func (h *HolidayHandler) getFourthThursday(year int, month time.Month) time.Time {
	// Start from first day of month
	firstDay := time.Date(year, month, 1, 0, 0, 0, 0, time.UTC)

	// Find first Thursday
	for firstDay.Weekday() != time.Thursday {
		firstDay = firstDay.AddDate(0, 0, 1)
	}

	// Add 3 weeks to get fourth Thursday
	return firstDay.AddDate(0, 0, 21)
}
