package conflict

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/javaknight1/servicepro/backend/internal/models"
	"github.com/javaknight1/servicepro/backend/internal/repository"
)

// ConflictValidator validates scheduling rules and constraints
type ConflictValidator struct {
	scheduleRepo repository.ScheduleRepositoryInterface
	userRepo     repository.UserRepositoryGormInterface
}

// NewConflictValidator creates a new conflict validator
func NewConflictValidator(
	scheduleRepo repository.ScheduleRepositoryInterface,
	userRepo repository.UserRepositoryGormInterface,
) *ConflictValidator {
	return &ConflictValidator{
		scheduleRepo: scheduleRepo,
		userRepo:     userRepo,
	}
}

// ValidationResult represents the result of a validation check
type ValidationResult struct {
	IsValid  bool              `json:"is_valid"`
	Errors   []ValidationError `json:"errors,omitempty"`
	Warnings []ValidationError `json:"warnings,omitempty"`
}

// ValidationError represents a validation error or warning
type ValidationError struct {
	Field    string `json:"field,omitempty"`
	Code     string `json:"code"`
	Message  string `json:"message"`
	Severity string `json:"severity"` // error, warning
}

// ValidateScheduleCreate validates a new schedule creation request
func (v *ConflictValidator) ValidateScheduleCreate(ctx context.Context, req *ConflictCheckRequest) *ValidationResult {
	result := &ValidationResult{
		IsValid:  true,
		Errors:   []ValidationError{},
		Warnings: []ValidationError{},
	}

	// Basic time validation
	if err := v.validateTimeRange(req.StartTime, req.EndTime); err != nil {
		result.Errors = append(result.Errors, *err)
		result.IsValid = false
	}

	// Validate schedule is not too far in the past
	if warning := v.validateNotTooOld(req.StartTime); warning != nil {
		result.Warnings = append(result.Warnings, *warning)
	}

	// Validate schedule is not too far in the future
	if warning := v.validateNotTooFarFuture(req.StartTime); warning != nil {
		result.Warnings = append(result.Warnings, *warning)
	}

	// Validate duration constraints
	if err := v.validateDuration(req.StartTime, req.EndTime); err != nil {
		result.Errors = append(result.Errors, *err)
		result.IsValid = false
	}

	// Validate technician assignments
	if len(req.AssignedTechIDs) == 0 {
		result.Warnings = append(result.Warnings, ValidationError{
			Field:    "assigned_tech_ids",
			Code:     "no_technicians",
			Message:  "No technicians assigned to this schedule",
			Severity: "warning",
		})
	}

	// Validate technician existence and status
	for i, techID := range req.AssignedTechIDs {
		if err := v.validateTechnician(ctx, techID); err != nil {
			result.Errors = append(result.Errors, ValidationError{
				Field:    fmt.Sprintf("assigned_tech_ids[%d]", i),
				Code:     err.Code,
				Message:  err.Message,
				Severity: "error",
			})
			result.IsValid = false
		}
	}

	// Validate against duplicate technicians
	if err := v.validateNoDuplicateTechnicians(req.AssignedTechIDs); err != nil {
		result.Errors = append(result.Errors, *err)
		result.IsValid = false
	}

	return result
}

// ValidateScheduleUpdate validates a schedule update request
func (v *ConflictValidator) ValidateScheduleUpdate(ctx context.Context, scheduleID uuid.UUID, req *ConflictCheckRequest) *ValidationResult {
	result := v.ValidateScheduleCreate(ctx, req)

	// Get existing schedule
	existingSchedule, err := v.scheduleRepo.GetByID(scheduleID)
	if err != nil {
		result.Errors = append(result.Errors, ValidationError{
			Code:     "schedule_not_found",
			Message:  "Schedule not found",
			Severity: "error",
		})
		result.IsValid = false
		return result
	}

	// Validate that confirmed schedules are not moved more than 24 hours
	if existingSchedule.IsConfirmed {
		if warning := v.validateConfirmedScheduleChange(existingSchedule, req); warning != nil {
			result.Warnings = append(result.Warnings, *warning)
		}
	}

	// Validate that cancelled schedules cannot be updated
	if existingSchedule.IsCancelled {
		result.Errors = append(result.Errors, ValidationError{
			Code:     "schedule_cancelled",
			Message:  "Cannot update a cancelled schedule",
			Severity: "error",
		})
		result.IsValid = false
	}

	return result
}

// ValidateConflictResolution validates a conflict resolution request
func (v *ConflictValidator) ValidateConflictResolution(ctx context.Context, conflictID uuid.UUID, resolvedBy uuid.UUID, notes string) *ValidationResult {
	result := &ValidationResult{
		IsValid:  true,
		Errors:   []ValidationError{},
		Warnings: []ValidationError{},
	}

	// Validate notes are provided
	if notes == "" {
		result.Warnings = append(result.Warnings, ValidationError{
			Field:    "notes",
			Code:     "missing_notes",
			Message:  "Resolution notes are recommended for audit trail",
			Severity: "warning",
		})
	}

	// Validate resolver has permission (would check user role)
	// This is a placeholder - actual implementation would check permissions
	if resolvedBy == uuid.Nil {
		result.Errors = append(result.Errors, ValidationError{
			Field:    "resolved_by",
			Code:     "invalid_resolver",
			Message:  "Invalid resolver ID",
			Severity: "error",
		})
		result.IsValid = false
	}

	return result
}

// validateTimeRange validates that end time is after start time
func (v *ConflictValidator) validateTimeRange(startTime, endTime time.Time) *ValidationError {
	if endTime.Before(startTime) || endTime.Equal(startTime) {
		return &ValidationError{
			Field:    "end_time",
			Code:     "invalid_time_range",
			Message:  "End time must be after start time",
			Severity: "error",
		}
	}
	return nil
}

// validateNotTooOld validates that schedule is not too far in the past
func (v *ConflictValidator) validateNotTooOld(startTime time.Time) *ValidationError {
	// Allow scheduling up to 7 days in the past
	maxPastDays := 7
	cutoff := time.Now().AddDate(0, 0, -maxPastDays)

	if startTime.Before(cutoff) {
		return &ValidationError{
			Field:    "start_time",
			Code:     "schedule_too_old",
			Message:  fmt.Sprintf("Schedule start time is more than %d days in the past", maxPastDays),
			Severity: "warning",
		}
	}
	return nil
}

// validateNotTooFarFuture validates that schedule is not too far in the future
func (v *ConflictValidator) validateNotTooFarFuture(startTime time.Time) *ValidationError {
	// Warn if scheduling more than 6 months in advance
	maxFutureMonths := 6
	cutoff := time.Now().AddDate(0, maxFutureMonths, 0)

	if startTime.After(cutoff) {
		return &ValidationError{
			Field:    "start_time",
			Code:     "schedule_too_far_future",
			Message:  fmt.Sprintf("Schedule start time is more than %d months in the future", maxFutureMonths),
			Severity: "warning",
		}
	}
	return nil
}

// validateDuration validates schedule duration constraints
func (v *ConflictValidator) validateDuration(startTime, endTime time.Time) *ValidationError {
	duration := endTime.Sub(startTime)

	// Minimum duration: 15 minutes
	minDuration := 15 * time.Minute
	if duration < minDuration {
		return &ValidationError{
			Field:    "duration",
			Code:     "duration_too_short",
			Message:  fmt.Sprintf("Schedule duration must be at least %d minutes", int(minDuration.Minutes())),
			Severity: "error",
		}
	}

	// Maximum duration: 12 hours
	maxDuration := 12 * time.Hour
	if duration > maxDuration {
		return &ValidationError{
			Field:    "duration",
			Code:     "duration_too_long",
			Message:  fmt.Sprintf("Schedule duration cannot exceed %d hours", int(maxDuration.Hours())),
			Severity: "error",
		}
	}

	return nil
}

// validateTechnician validates that a technician exists and is active
func (v *ConflictValidator) validateTechnician(ctx context.Context, techID uuid.UUID) *ValidationError {
	// Get user from repository
	user, err := v.userRepo.GetByID(techID)
	if err != nil {
		return &ValidationError{
			Code:     "technician_not_found",
			Message:  fmt.Sprintf("Technician with ID %s not found", techID),
			Severity: "error",
		}
	}

	// Check if user account is locked
	if user.IsLocked() {
		return &ValidationError{
			Code:     "technician_locked",
			Message:  fmt.Sprintf("Technician %s has a locked account", user.Email),
			Severity: "error",
		}
	}

	// Check if user has technician role
	// This is a simplified check - actual implementation would check user roles
	// For now, we assume all users can be technicians

	return nil
}

// validateNoDuplicateTechnicians validates that no technician is assigned twice
func (v *ConflictValidator) validateNoDuplicateTechnicians(techIDs []uuid.UUID) *ValidationError {
	seen := make(map[uuid.UUID]bool)
	for _, id := range techIDs {
		if seen[id] {
			return &ValidationError{
				Field:    "assigned_tech_ids",
				Code:     "duplicate_technician",
				Message:  fmt.Sprintf("Technician %s is assigned multiple times", id),
				Severity: "error",
			}
		}
		seen[id] = true
	}
	return nil
}

// validateConfirmedScheduleChange validates changes to confirmed schedules
func (v *ConflictValidator) validateConfirmedScheduleChange(existing *models.Schedule, req *ConflictCheckRequest) *ValidationError {
	// Check if time is being changed significantly
	startTimeDiff := req.StartTime.Sub(existing.StartTime).Abs()
	endTimeDiff := req.EndTime.Sub(existing.EndTime).Abs()

	// Warn if moving by more than 24 hours
	maxChange := 24 * time.Hour
	if startTimeDiff > maxChange || endTimeDiff > maxChange {
		return &ValidationError{
			Code:     "confirmed_schedule_major_change",
			Message:  "Moving a confirmed schedule by more than 24 hours may require customer notification",
			Severity: "warning",
		}
	}

	return nil
}

// ValidateTechnicianAvailability validates technician availability based on their schedule
func (v *ConflictValidator) ValidateTechnicianAvailability(ctx context.Context, techID uuid.UUID, startTime, endTime time.Time) (bool, string) {
	// Get technician's schedules for the time range
	schedules, err := v.scheduleRepo.GetByTechnicianAndDateRange(techID, startTime, endTime)
	if err != nil {
		return false, fmt.Sprintf("Failed to check availability: %v", err)
	}

	// Check for any overlapping schedules
	for _, schedule := range schedules {
		if schedule.IsCancelled {
			continue
		}

		if startTime.Before(schedule.EndTime) && endTime.After(schedule.StartTime) {
			return false, fmt.Sprintf("Technician has a conflicting schedule: %s (%s to %s)",
				schedule.Title,
				schedule.StartTime.Format("3:04 PM"),
				schedule.EndTime.Format("3:04 PM"))
		}
	}

	return true, ""
}

// ValidateResourceAvailability validates that required resources are available
// This is a placeholder for future resource management
func (v *ConflictValidator) ValidateResourceAvailability(ctx context.Context, resourceIDs []uuid.UUID, startTime, endTime time.Time) (bool, string) {
	// Placeholder implementation
	// In a real system, this would check equipment, vehicles, tools, etc.
	return true, ""
}

// ValidateBusinessRules validates custom business rules
func (v *ConflictValidator) ValidateBusinessRules(ctx context.Context, req *ConflictCheckRequest) *ValidationResult {
	result := &ValidationResult{
		IsValid:  true,
		Errors:   []ValidationError{},
		Warnings: []ValidationError{},
	}

	// Rule: No scheduling on holidays
	if v.isHoliday(req.StartTime) {
		result.Warnings = append(result.Warnings, ValidationError{
			Code:     "holiday_scheduling",
			Message:  "Schedule is on a holiday - may require special approval",
			Severity: "warning",
		})
	}

	// Rule: Maximum 2 jobs per technician per day (example business rule)
	for _, techID := range req.AssignedTechIDs {
		dayStart := time.Date(req.StartTime.Year(), req.StartTime.Month(), req.StartTime.Day(), 0, 0, 0, 0, req.StartTime.Location())
		dayEnd := dayStart.Add(24 * time.Hour)

		schedules, err := v.scheduleRepo.GetByTechnicianAndDateRange(techID, dayStart, dayEnd)
		if err == nil && len(schedules) >= 5 {
			result.Warnings = append(result.Warnings, ValidationError{
				Code:     "high_job_count",
				Message:  fmt.Sprintf("Technician has %d jobs scheduled for this day", len(schedules)),
				Severity: "warning",
			})
		}
	}

	return result
}

// isHoliday checks if a date is a holiday
// This is a simplified implementation - in production, this would check against a holiday calendar
func (v *ConflictValidator) isHoliday(date time.Time) bool {
	// Simplified: Just check for New Year's Day and Christmas
	month := date.Month()
	day := date.Day()

	if (month == time.January && day == 1) || (month == time.December && day == 25) {
		return true
	}

	return false
}
