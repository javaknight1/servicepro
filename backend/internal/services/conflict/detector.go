package conflict

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/javaknight1/servicepro/backend/internal/models"
	"github.com/javaknight1/servicepro/backend/internal/repository"
)

// ConflictDetector provides real-time conflict detection for schedules
type ConflictDetector struct {
	scheduleRepo repository.ScheduleRepositoryInterface
}

// NewConflictDetector creates a new conflict detector
func NewConflictDetector(scheduleRepo repository.ScheduleRepositoryInterface) *ConflictDetector {
	return &ConflictDetector{
		scheduleRepo: scheduleRepo,
	}
}

// ConflictCheckRequest represents a request to check for conflicts
type ConflictCheckRequest struct {
	ScheduleID      *uuid.UUID  // nil for new schedules
	JobID           uuid.UUID   `json:"job_id"`
	StartTime       time.Time   `json:"start_time"`
	EndTime         time.Time   `json:"end_time"`
	AssignedTechIDs []uuid.UUID `json:"assigned_tech_ids"`
	Location        string      `json:"location,omitempty"`
}

// ConflictCheckResponse represents the result of a conflict check
type ConflictCheckResponse struct {
	HasConflicts bool                   `json:"has_conflicts"`
	Conflicts    []ConflictDetail       `json:"conflicts"`
	Suggestions  []ResolutionSuggestion `json:"suggestions,omitempty"`
}

// ConflictDetail provides detailed information about a specific conflict
type ConflictDetail struct {
	ConflictType    models.ConflictType     `json:"conflict_type"`
	Severity        models.ConflictSeverity `json:"severity"`
	Description     string                  `json:"description"`
	ConflictingWith *ConflictingSchedule    `json:"conflicting_with,omitempty"`
	TechnicianID    *uuid.UUID              `json:"technician_id,omitempty"`
	TimeOverlap     *TimeOverlapInfo        `json:"time_overlap,omitempty"`
}

// ConflictingSchedule represents a schedule that conflicts with the requested schedule
type ConflictingSchedule struct {
	ID        uuid.UUID `json:"id"`
	Title     string    `json:"title"`
	JobNumber string    `json:"job_number,omitempty"`
	StartTime time.Time `json:"start_time"`
	EndTime   time.Time `json:"end_time"`
	Location  string    `json:"location,omitempty"`
}

// TimeOverlapInfo provides details about time overlap
type TimeOverlapInfo struct {
	OverlapStart   time.Time `json:"overlap_start"`
	OverlapEnd     time.Time `json:"overlap_end"`
	OverlapMinutes int       `json:"overlap_minutes"`
}

// ResolutionSuggestion provides suggestions for resolving conflicts
type ResolutionSuggestion struct {
	Type        string `json:"type"`
	Description string `json:"description"`
	Action      string `json:"action"`
	Priority    int    `json:"priority"`
}

// CheckConflicts performs real-time conflict detection for a schedule
func (d *ConflictDetector) CheckConflicts(ctx context.Context, req *ConflictCheckRequest) (*ConflictCheckResponse, error) {
	if err := d.validateRequest(req); err != nil {
		return nil, fmt.Errorf("invalid request: %w", err)
	}

	response := &ConflictCheckResponse{
		HasConflicts: false,
		Conflicts:    []ConflictDetail{},
		Suggestions:  []ResolutionSuggestion{},
	}

	// Check for technician overlap conflicts
	techConflicts, err := d.checkTechnicianConflicts(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to check technician conflicts: %w", err)
	}
	response.Conflicts = append(response.Conflicts, techConflicts...)

	// Check for location conflicts (optional - if same location can't have multiple jobs)
	// This is commented out as it depends on business rules
	// locationConflicts, err := d.checkLocationConflicts(ctx, req)
	// if err != nil {
	// 	return nil, fmt.Errorf("failed to check location conflicts: %w", err)
	// }
	// response.Conflicts = append(response.Conflicts, locationConflicts...)

	// Check for business hour violations
	businessHourConflicts := d.checkBusinessHours(req)
	response.Conflicts = append(response.Conflicts, businessHourConflicts...)

	// Check for excessive workload
	workloadConflicts, err := d.checkTechnicianWorkload(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to check workload: %w", err)
	}
	response.Conflicts = append(response.Conflicts, workloadConflicts...)

	// Set HasConflicts flag
	response.HasConflicts = len(response.Conflicts) > 0

	// Generate resolution suggestions if conflicts exist
	if response.HasConflicts {
		response.Suggestions = d.generateSuggestions(ctx, req, response.Conflicts)
	}

	return response, nil
}

// checkTechnicianConflicts checks for technician scheduling conflicts
func (d *ConflictDetector) checkTechnicianConflicts(ctx context.Context, req *ConflictCheckRequest) ([]ConflictDetail, error) {
	if len(req.AssignedTechIDs) == 0 {
		return []ConflictDetail{}, nil
	}

	conflicts := []ConflictDetail{}

	// Check each assigned technician for conflicts
	for _, techID := range req.AssignedTechIDs {
		// Get technician's schedules in the requested time range
		schedules, err := d.scheduleRepo.GetByTechnicianAndDateRange(
			techID,
			req.StartTime,
			req.EndTime,
		)
		if err != nil {
			return nil, err
		}

		// Filter out the current schedule if updating
		for _, schedule := range schedules {
			if req.ScheduleID != nil && schedule.ID == *req.ScheduleID {
				continue
			}

			// Skip cancelled schedules
			if schedule.IsCancelled {
				continue
			}

			// Check for time overlap
			if d.hasTimeOverlap(req.StartTime, req.EndTime, schedule.StartTime, schedule.EndTime) {
				overlapInfo := d.calculateOverlap(req.StartTime, req.EndTime, schedule.StartTime, schedule.EndTime)

				jobNumber := ""
				if schedule.Job != nil {
					jobNumber = schedule.Job.JobNumber
				}
				conflicts = append(conflicts, ConflictDetail{
					ConflictType: models.ConflictTypeTechnicianOverlap,
					Severity:     d.calculateSeverity(overlapInfo, &schedule),
					Description:  fmt.Sprintf("Technician is already scheduled during this time"),
					ConflictingWith: &ConflictingSchedule{
						ID:        schedule.ID,
						Title:     schedule.Title,
						JobNumber: jobNumber,
						StartTime: schedule.StartTime,
						EndTime:   schedule.EndTime,
						Location:  schedule.Location,
					},
					TechnicianID: &techID,
					TimeOverlap:  overlapInfo,
				})
			}
		}
	}

	return conflicts, nil
}

// checkBusinessHours validates that schedule is within business hours
func (d *ConflictDetector) checkBusinessHours(req *ConflictCheckRequest) []ConflictDetail {
	conflicts := []ConflictDetail{}

	// Define business hours (8 AM - 6 PM, Monday-Friday)
	businessHourStart := 8
	businessHourEnd := 18

	startHour := req.StartTime.Hour()
	endHour := req.EndTime.Hour()
	weekday := req.StartTime.Weekday()

	// Check weekend
	if weekday == time.Saturday || weekday == time.Sunday {
		conflicts = append(conflicts, ConflictDetail{
			ConflictType: models.ConflictTypeBusinessHours,
			Severity:     models.ConflictSeverityLow,
			Description:  "Schedule is on a weekend - may require overtime approval",
		})
	}

	// Check hours
	if startHour < businessHourStart || endHour > businessHourEnd {
		conflicts = append(conflicts, ConflictDetail{
			ConflictType: models.ConflictTypeBusinessHours,
			Severity:     models.ConflictSeverityMedium,
			Description:  fmt.Sprintf("Schedule is outside business hours (%d:00 - %d:00) - may require overtime approval", businessHourStart, businessHourEnd),
		})
	}

	return conflicts
}

// checkTechnicianWorkload checks if technician has excessive workload
func (d *ConflictDetector) checkTechnicianWorkload(ctx context.Context, req *ConflictCheckRequest) ([]ConflictDetail, error) {
	conflicts := []ConflictDetail{}

	// Get the full day for workload calculation
	dayStart := time.Date(req.StartTime.Year(), req.StartTime.Month(), req.StartTime.Day(), 0, 0, 0, 0, req.StartTime.Location())
	dayEnd := dayStart.Add(24 * time.Hour)

	for _, techID := range req.AssignedTechIDs {
		// Get all schedules for the day
		schedules, err := d.scheduleRepo.GetByTechnicianAndDateRange(techID, dayStart, dayEnd)
		if err != nil {
			return nil, err
		}

		// Calculate total work hours for the day
		totalMinutes := 0
		for _, schedule := range schedules {
			if schedule.IsCancelled {
				continue
			}
			if req.ScheduleID != nil && schedule.ID == *req.ScheduleID {
				continue
			}
			duration := schedule.EndTime.Sub(schedule.StartTime)
			totalMinutes += int(duration.Minutes())
		}

		// Add the requested schedule duration
		requestDuration := req.EndTime.Sub(req.StartTime)
		totalMinutes += int(requestDuration.Minutes())

		// Check if exceeds 10 hours (600 minutes)
		maxDailyMinutes := 600
		if totalMinutes > maxDailyMinutes {
			conflicts = append(conflicts, ConflictDetail{
				ConflictType: models.ConflictTypeWorkloadExcess,
				Severity:     models.ConflictSeverityHigh,
				Description:  fmt.Sprintf("Technician workload exceeds %d hours for the day (total: %.1f hours)", maxDailyMinutes/60, float64(totalMinutes)/60),
				TechnicianID: &techID,
			})
		} else if totalMinutes > 480 { // 8 hours
			conflicts = append(conflicts, ConflictDetail{
				ConflictType: models.ConflictTypeWorkloadExcess,
				Severity:     models.ConflictSeverityMedium,
				Description:  fmt.Sprintf("Technician workload exceeds 8 hours for the day (total: %.1f hours) - may require overtime", float64(totalMinutes)/60),
				TechnicianID: &techID,
			})
		}
	}

	return conflicts, nil
}

// hasTimeOverlap checks if two time ranges overlap
func (d *ConflictDetector) hasTimeOverlap(start1, end1, start2, end2 time.Time) bool {
	return start1.Before(end2) && end1.After(start2)
}

// calculateOverlap calculates the overlap between two time ranges
func (d *ConflictDetector) calculateOverlap(start1, end1, start2, end2 time.Time) *TimeOverlapInfo {
	overlapStart := start1
	if start2.After(start1) {
		overlapStart = start2
	}

	overlapEnd := end1
	if end2.Before(end1) {
		overlapEnd = end2
	}

	duration := overlapEnd.Sub(overlapStart)

	return &TimeOverlapInfo{
		OverlapStart:   overlapStart,
		OverlapEnd:     overlapEnd,
		OverlapMinutes: int(duration.Minutes()),
	}
}

// calculateSeverity determines the severity of a conflict based on overlap and schedule importance
func (d *ConflictDetector) calculateSeverity(overlap *TimeOverlapInfo, schedule *models.Schedule) models.ConflictSeverity {
	// If schedule is confirmed, it's high severity
	if schedule.IsConfirmed {
		return models.ConflictSeverityHigh
	}

	// If overlap is more than 2 hours, it's high severity
	if overlap.OverlapMinutes > 120 {
		return models.ConflictSeverityHigh
	}

	// If overlap is more than 30 minutes, it's medium severity
	if overlap.OverlapMinutes > 30 {
		return models.ConflictSeverityMedium
	}

	return models.ConflictSeverityLow
}

// generateSuggestions generates resolution suggestions based on conflicts
func (d *ConflictDetector) generateSuggestions(ctx context.Context, req *ConflictCheckRequest, conflicts []ConflictDetail) []ResolutionSuggestion {
	suggestions := []ResolutionSuggestion{}

	// Count conflict types
	hasTechConflict := false
	hasWorkloadConflict := false
	hasBusinessHourConflict := false

	for _, conflict := range conflicts {
		switch conflict.ConflictType {
		case models.ConflictTypeTechnicianOverlap:
			hasTechConflict = true
		case models.ConflictTypeWorkloadExcess:
			hasWorkloadConflict = true
		case models.ConflictTypeBusinessHours:
			hasBusinessHourConflict = true
		}
	}

	// Suggest alternative technicians
	if hasTechConflict {
		suggestions = append(suggestions, ResolutionSuggestion{
			Type:        "reassign_technician",
			Description: "Assign a different technician who is available during this time",
			Action:      "show_available_technicians",
			Priority:    1,
		})
	}

	// Suggest rescheduling
	if hasTechConflict || hasWorkloadConflict {
		suggestions = append(suggestions, ResolutionSuggestion{
			Type:        "reschedule",
			Description: "Move the schedule to a different time slot",
			Action:      "show_alternative_times",
			Priority:    2,
		})
	}

	// Suggest splitting the job
	if hasWorkloadConflict {
		suggestions = append(suggestions, ResolutionSuggestion{
			Type:        "split_job",
			Description: "Split the job into multiple shorter sessions",
			Action:      "split_schedule",
			Priority:    3,
		})
	}

	// Suggest overtime approval
	if hasBusinessHourConflict {
		suggestions = append(suggestions, ResolutionSuggestion{
			Type:        "overtime_approval",
			Description: "Request overtime approval for after-hours work",
			Action:      "request_approval",
			Priority:    4,
		})
	}

	return suggestions
}

// validateRequest validates the conflict check request
func (d *ConflictDetector) validateRequest(req *ConflictCheckRequest) error {
	if req.StartTime.IsZero() {
		return fmt.Errorf("start_time is required")
	}
	if req.EndTime.IsZero() {
		return fmt.Errorf("end_time is required")
	}
	if req.EndTime.Before(req.StartTime) || req.EndTime.Equal(req.StartTime) {
		return fmt.Errorf("end_time must be after start_time")
	}
	return nil
}

// GetConflictsBySchedule retrieves existing conflicts for a schedule
func (d *ConflictDetector) GetConflictsBySchedule(ctx context.Context, scheduleID uuid.UUID) ([]models.ScheduleConflict, error) {
	return d.scheduleRepo.GetConflicts(scheduleID)
}

// ResolveConflict marks a conflict as resolved
func (d *ConflictDetector) ResolveConflict(ctx context.Context, conflictID uuid.UUID, resolvedBy uuid.UUID, notes string) error {
	return d.scheduleRepo.ResolveConflict(conflictID, resolvedBy, &notes)
}
