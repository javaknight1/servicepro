package recurring

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"

	"github.com/javaknight1/servicepro/backend/internal/models"
	"github.com/javaknight1/servicepro/backend/internal/repository"
	"github.com/javaknight1/servicepro/backend/internal/utils/epoch"
)

// ScheduleGenerator generates schedule instances from recurring patterns
type ScheduleGenerator struct {
	scheduleRepo  repository.ScheduleRepositoryInterface
	exceptionRepo ExceptionRepositoryInterface
	holidayRepo   HolidayRepositoryInterface
	logger        *log.Logger
}

// NewScheduleGenerator creates a new schedule generator
func NewScheduleGenerator(
	scheduleRepo repository.ScheduleRepositoryInterface,
	exceptionRepo ExceptionRepositoryInterface,
	holidayRepo HolidayRepositoryInterface,
) *ScheduleGenerator {
	return &ScheduleGenerator{
		scheduleRepo:  scheduleRepo,
		exceptionRepo: exceptionRepo,
		holidayRepo:   holidayRepo,
		logger:        log.New(log.Writer(), "[ScheduleGenerator] ", log.LstdFlags),
	}
}

// GenerationRequest represents a request to generate schedules from a recurring pattern
type GenerationRequest struct {
	RecurringScheduleID uuid.UUID
	StartDate           int64
	EndDate             int64
	SkipHolidays        bool
	SkipWeekends        bool
	MaxOccurrences      int
}

// GenerationResult represents the result of schedule generation
type GenerationResult struct {
	GeneratedCount int               `json:"generated_count"`
	SkippedCount   int               `json:"skipped_count"`
	Schedules      []models.Schedule `json:"schedules"`
	SkippedDates   []SkippedDate     `json:"skipped_dates,omitempty"`
}

// SkippedDate represents a date that was skipped during generation
type SkippedDate struct {
	Date   int64  `json:"date"`
	Reason string `json:"reason"`
}

// GenerateSchedules generates schedule instances from a recurring pattern
func (g *ScheduleGenerator) GenerateSchedules(ctx context.Context, req *GenerationRequest) (*GenerationResult, error) {
	g.logger.Printf("Generating schedules for recurring pattern %s from %s to %s",
		req.RecurringScheduleID,
		epoch.EpochToTime(req.StartDate).Format("2006-01-02"),
		epoch.EpochToTime(req.EndDate).Format("2006-01-02"))

	// Get recurring schedule
	recurring, err := g.scheduleRepo.GetRecurringByID(req.RecurringScheduleID)
	if err != nil {
		return nil, fmt.Errorf("failed to get recurring schedule: %w", err)
	}

	if !recurring.IsActive {
		return nil, fmt.Errorf("recurring schedule is not active")
	}

	// Get exceptions for the pattern
	exceptions, err := g.exceptionRepo.GetExceptionsByPattern(req.RecurringScheduleID, req.StartDate, req.EndDate)
	if err != nil {
		g.logger.Printf("Warning: failed to get exceptions: %v", err)
		exceptions = []ScheduleException{}
	}

	// Get holidays if needed
	var holidays []Holiday
	if req.SkipHolidays {
		holidays, err = g.holidayRepo.GetHolidaysByDateRange(req.StartDate, req.EndDate)
		if err != nil {
			g.logger.Printf("Warning: failed to get holidays: %v", err)
			holidays = []Holiday{}
		}
	}

	// Generate occurrences based on pattern type
	occurrences := g.generateOccurrences(recurring, req.StartDate, req.EndDate, req.MaxOccurrences)

	result := &GenerationResult{
		GeneratedCount: 0,
		SkippedCount:   0,
		Schedules:      []models.Schedule{},
		SkippedDates:   []SkippedDate{},
	}

	// Create schedules for each occurrence
	for _, occurrence := range occurrences {
		// Check if date should be skipped
		skip, reason := g.shouldSkipDate(occurrence, exceptions, holidays, req.SkipWeekends)
		if skip {
			result.SkippedCount++
			result.SkippedDates = append(result.SkippedDates, SkippedDate{
				Date:   occurrence,
				Reason: reason,
			})
			continue
		}

		// Create schedule instance
		schedule := g.createScheduleFromPattern(recurring, occurrence)

		// Save to database
		if err := g.scheduleRepo.Create(&schedule); err != nil {
			g.logger.Printf("Failed to create schedule for %s: %v",
				epoch.EpochToTime(occurrence).Format("2006-01-02"), err)
			continue
		}

		result.GeneratedCount++
		result.Schedules = append(result.Schedules, schedule)
	}

	g.logger.Printf("Generated %d schedules, skipped %d dates", result.GeneratedCount, result.SkippedCount)
	return result, nil
}

// generateOccurrences generates occurrence dates based on the recurring pattern
func (g *ScheduleGenerator) generateOccurrences(
	recurring *models.RecurringSchedule,
	startDate, endDate int64,
	maxOccurrences int,
) []int64 {
	occurrences := []int64{}

	// Start from the recurring schedule's start date or requested start date, whichever is later
	current := recurring.StartDate
	if startDate > current {
		current = startDate
	}

	// End at the recurring schedule's end date or requested end date, whichever is earlier
	end := endDate
	if recurring.EndDate != nil && *recurring.EndDate < end {
		end = *recurring.EndDate
	}

	maxCount := maxOccurrences
	if maxCount == 0 {
		maxCount = 365 // Default max to prevent infinite loops
	}

	switch recurring.RecurrenceType {
	case models.RecurrenceDaily:
		occurrences = g.generateDailyOccurrences(current, end, recurring.Interval, maxCount)

	case models.RecurrenceWeekly:
		occurrences = g.generateWeeklyOccurrences(current, end, recurring.Interval, recurring.DaysOfWeek, maxCount)

	case models.RecurrenceMonthly:
		occurrences = g.generateMonthlyOccurrences(current, end, recurring.Interval, recurring.DayOfMonth, maxCount)

	case models.RecurrenceYearly:
		occurrences = g.generateYearlyOccurrences(current, end, recurring.Interval, recurring.MonthOfYear, recurring.DayOfMonth, maxCount)

	default:
		g.logger.Printf("Unknown recurrence type: %s", recurring.RecurrenceType)
	}

	// Apply max occurrences limit from recurring schedule
	if recurring.Occurrences != nil && len(occurrences) > *recurring.Occurrences {
		occurrences = occurrences[:*recurring.Occurrences]
	}

	return occurrences
}

// generateDailyOccurrences generates daily occurrences
func (g *ScheduleGenerator) generateDailyOccurrences(start, end int64, interval, maxCount int) []int64 {
	occurrences := []int64{}
	current := epoch.EpochToTime(start)
	endTime := epoch.EpochToTime(end)

	for !current.After(endTime) {
		if len(occurrences) >= maxCount {
			break
		}
		occurrences = append(occurrences, current.Unix())
		current = current.AddDate(0, 0, interval)
	}

	return occurrences
}

// generateWeeklyOccurrences generates weekly occurrences on specific days
func (g *ScheduleGenerator) generateWeeklyOccurrences(start, end int64, interval int, daysOfWeek []int, maxCount int) []int64 {
	occurrences := []int64{}

	if len(daysOfWeek) == 0 {
		return occurrences
	}

	startTime := epoch.EpochToTime(start)
	endTime := epoch.EpochToTime(end)
	current := startTime

	for !current.After(endTime) {
		if len(occurrences) >= maxCount {
			break
		}

		// Check each day of the current week
		for day := 0; day < 7; day++ {
			checkDate := current.AddDate(0, 0, day)

			if checkDate.Before(startTime) {
				continue
			}
			if checkDate.After(endTime) {
				break
			}

			// Check if this day of week is in the pattern
			weekday := int(checkDate.Weekday())
			for _, targetDay := range daysOfWeek {
				if weekday == targetDay {
					occurrences = append(occurrences, checkDate.Unix())
					if len(occurrences) >= maxCount {
						return occurrences
					}
				}
			}
		}

		// Move to next week interval
		current = current.AddDate(0, 0, interval*7)
	}

	return occurrences
}

// generateMonthlyOccurrences generates monthly occurrences on a specific day
func (g *ScheduleGenerator) generateMonthlyOccurrences(start, end int64, interval int, dayOfMonth *int, maxCount int) []int64 {
	occurrences := []int64{}

	startTime := epoch.EpochToTime(start)
	endTime := epoch.EpochToTime(end)

	if dayOfMonth == nil {
		day := startTime.Day()
		dayOfMonth = &day
	}

	current := startTime
	for !current.After(endTime) {
		if len(occurrences) >= maxCount {
			break
		}

		year := current.Year()
		month := current.Month()

		// Handle months with fewer days
		daysInMonth := time.Date(year, month+1, 0, 0, 0, 0, 0, time.UTC).Day()
		day := *dayOfMonth
		if day > daysInMonth {
			day = daysInMonth
		}

		occurrence := time.Date(year, month, day, 0, 0, 0, 0, time.UTC)

		if !occurrence.Before(startTime) && !occurrence.After(endTime) {
			occurrences = append(occurrences, occurrence.Unix())
		}

		// Move to next month interval
		current = current.AddDate(0, interval, 0)
	}

	return occurrences
}

// generateYearlyOccurrences generates yearly occurrences on a specific date
func (g *ScheduleGenerator) generateYearlyOccurrences(start, end int64, interval int, monthOfYear, dayOfMonth *int, maxCount int) []int64 {
	occurrences := []int64{}

	startTime := epoch.EpochToTime(start)
	endTime := epoch.EpochToTime(end)

	month := startTime.Month()
	day := startTime.Day()

	if monthOfYear != nil {
		month = time.Month(*monthOfYear)
	}
	if dayOfMonth != nil {
		day = *dayOfMonth
	}

	currentYear := startTime.Year()
	endYear := endTime.Year()

	for year := currentYear; year <= endYear; year += interval {
		if len(occurrences) >= maxCount {
			break
		}

		occurrence := time.Date(year, month, day, 0, 0, 0, 0, time.UTC)

		if !occurrence.Before(startTime) && !occurrence.After(endTime) {
			occurrences = append(occurrences, occurrence.Unix())
		}
	}

	return occurrences
}

// shouldSkipDate checks if a date should be skipped
func (g *ScheduleGenerator) shouldSkipDate(
	dateEpoch int64,
	exceptions []ScheduleException,
	holidays []Holiday,
	skipWeekends bool,
) (bool, string) {
	date := epoch.EpochToTime(dateEpoch)

	// Check exceptions
	for _, exception := range exceptions {
		if g.isSameDay(date, exception.ExceptionDate) {
			return true, fmt.Sprintf("Exception: %s", exception.Reason)
		}
	}

	// Check holidays
	for _, holiday := range holidays {
		if g.isSameDay(date, holiday.Date) {
			return true, fmt.Sprintf("Holiday: %s", holiday.Name)
		}
	}

	// Check weekends
	if skipWeekends && (date.Weekday() == time.Saturday || date.Weekday() == time.Sunday) {
		return true, "Weekend"
	}

	return false, ""
}

// isSameDay checks if two dates are the same day
func (g *ScheduleGenerator) isSameDay(date1, date2 time.Time) bool {
	y1, m1, d1 := date1.Date()
	y2, m2, d2 := date2.Date()
	return y1 == y2 && m1 == m2 && d1 == d2
}

// createScheduleFromPattern creates a schedule instance from a recurring pattern
func (g *ScheduleGenerator) createScheduleFromPattern(recurring *models.RecurringSchedule, dateEpoch int64) models.Schedule {
	date := epoch.EpochToTime(dateEpoch)

	// Use seconds-from-midnight fields to compute start/end epoch
	startHour, startMin := epoch.SecondsFromMidnightToTime(recurring.TimeStart)
	endHour, endMin := epoch.SecondsFromMidnightToTime(recurring.TimeEnd)

	start := time.Date(date.Year(), date.Month(), date.Day(),
		startHour, startMin, 0, 0, time.UTC).Unix()
	end := time.Date(date.Year(), date.Month(), date.Day(),
		endHour, endMin, 0, 0, time.UTC).Unix()

	schedule := models.Schedule{
		ID:                  uuid.New(),
		JobID:               *recurring.JobTemplateID,
		Title:               recurring.Title,
		Description:         recurring.Description,
		StartTime:           start,
		EndTime:             end,
		RecurrenceType:      recurring.RecurrenceType,
		RecurringScheduleID: &recurring.ID,
		AssignedTechIDs:     recurring.AssignedTechIDs,
		Location:            recurring.Location,
		IsConfirmed:         false,
		IsCancelled:         false,
		CreatedBy:           recurring.CreatedBy,
	}

	return schedule
}

// PreviewSchedules generates a preview without saving to database
func (g *ScheduleGenerator) PreviewSchedules(ctx context.Context, req *GenerationRequest) (*GenerationResult, error) {
	g.logger.Printf("Previewing schedules for recurring pattern %s", req.RecurringScheduleID)

	// Get recurring schedule
	recurring, err := g.scheduleRepo.GetRecurringByID(req.RecurringScheduleID)
	if err != nil {
		return nil, fmt.Errorf("failed to get recurring schedule: %w", err)
	}

	// Get exceptions
	exceptions, _ := g.exceptionRepo.GetExceptionsByPattern(req.RecurringScheduleID, req.StartDate, req.EndDate)

	// Get holidays if needed
	var holidays []Holiday
	if req.SkipHolidays {
		holidays, _ = g.holidayRepo.GetHolidaysByDateRange(req.StartDate, req.EndDate)
	}

	// Generate occurrences
	occurrences := g.generateOccurrences(recurring, req.StartDate, req.EndDate, req.MaxOccurrences)

	result := &GenerationResult{
		GeneratedCount: 0,
		SkippedCount:   0,
		Schedules:      []models.Schedule{},
		SkippedDates:   []SkippedDate{},
	}

	// Preview schedules (don't save)
	for _, occurrence := range occurrences {
		skip, reason := g.shouldSkipDate(occurrence, exceptions, holidays, req.SkipWeekends)
		if skip {
			result.SkippedCount++
			result.SkippedDates = append(result.SkippedDates, SkippedDate{
				Date:   occurrence,
				Reason: reason,
			})
			continue
		}

		schedule := g.createScheduleFromPattern(recurring, occurrence)
		result.GeneratedCount++
		result.Schedules = append(result.Schedules, schedule)
	}

	return result, nil
}
