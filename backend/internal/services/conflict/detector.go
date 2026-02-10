package conflict

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/javaknight1/servicepro/backend/internal/models"
	"github.com/javaknight1/servicepro/backend/internal/repository"
	"github.com/javaknight1/servicepro/backend/internal/utils/epoch"
)

// ConflictDetector provides real-time conflict detection for schedules
type ConflictDetector struct {
	scheduleRepo repository.ScheduleRepositoryInterface
	jobRepo      repository.JobRepositoryInterface
}

// NewConflictDetector creates a new conflict detector
func NewConflictDetector(scheduleRepo repository.ScheduleRepositoryInterface, jobRepo repository.JobRepositoryInterface) *ConflictDetector {
	return &ConflictDetector{
		scheduleRepo: scheduleRepo,
		jobRepo:      jobRepo,
	}
}

// ConflictCheckRequest represents a request to check for conflicts
type ConflictCheckRequest struct {
	TenantID        uuid.UUID   `json:"-"` // Set from auth context, not from JSON body
	ScheduleID      *uuid.UUID  // nil for new schedules
	JobID           uuid.UUID   `json:"job_id"`
	StartTime       int64       `json:"start_time"`
	EndTime         int64       `json:"end_time"`
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
	StartTime int64     `json:"start_time"`
	EndTime   int64     `json:"end_time"`
	Location  string    `json:"location,omitempty"`
}

// TimeOverlapInfo provides details about time overlap
type TimeOverlapInfo struct {
	OverlapStart   int64 `json:"overlap_start"`
	OverlapEnd     int64 `json:"overlap_end"`
	OverlapMinutes int   `json:"overlap_minutes"`
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

	// Business hour checks disabled - not timezone-aware (compares against UTC, not local time)
	// TODO: Re-enable with proper tenant timezone support

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
			req.TenantID,
			techID,
			req.StartTime,
			req.EndTime,
		)
		if err != nil {
			return nil, err
		}

		// Filter out the current schedule/job if updating
		for _, schedule := range schedules {
			if req.ScheduleID != nil && schedule.ID == *req.ScheduleID {
				continue
			}
			// Exclude schedules belonging to the same job (self-conflict)
			if req.JobID != uuid.Nil && schedule.JobID == req.JobID {
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
				jobTitle := schedule.Title
				if schedule.Job != nil {
					jobNumber = schedule.Job.JobNumber
					if schedule.Job.Title != "" {
						jobTitle = schedule.Job.Title
					}
				}

				description := fmt.Sprintf("Technician is already scheduled for \"%s\"", jobTitle)
				if jobNumber != "" {
					description += fmt.Sprintf(" (%s)", jobNumber)
				}
				description += fmt.Sprintf(" from %s to %s",
					epoch.EpochToTime(schedule.StartTime).Format("Jan 2 3:04 PM"),
					epoch.EpochToTime(schedule.EndTime).Format("3:04 PM"))

				conflicts = append(conflicts, ConflictDetail{
					ConflictType: models.ConflictTypeTechnicianOverlap,
					Severity:     d.calculateSeverity(overlapInfo, &schedule),
					Description:  description,
					ConflictingWith: &ConflictingSchedule{
						ID:        schedule.ID,
						Title:     jobTitle,
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

		// Also check jobs table for overlapping job assignments
		if d.jobRepo != nil {
			jobs, err := d.jobRepo.GetJobsByTechnicianAndDateRange(techID, req.StartTime, req.EndTime)
			if err != nil {
				return nil, err
			}

			for _, job := range jobs {
				// Skip the same job (self-conflict when updating)
				if req.JobID != uuid.Nil && job.ID == req.JobID {
					continue
				}

				if job.ScheduledStartAt != nil && job.ScheduledEndAt != nil {
					if d.hasTimeOverlap(req.StartTime, req.EndTime, *job.ScheduledStartAt, *job.ScheduledEndAt) {
						overlapInfo := d.calculateOverlap(req.StartTime, req.EndTime, *job.ScheduledStartAt, *job.ScheduledEndAt)

						description := fmt.Sprintf("Technician is already assigned to \"%s\"", job.Title)
						if job.JobNumber != "" {
							description += fmt.Sprintf(" (%s)", job.JobNumber)
						}
						description += fmt.Sprintf(" from %s to %s",
							epoch.EpochToTime(*job.ScheduledStartAt).Format("Jan 2 3:04 PM"),
							epoch.EpochToTime(*job.ScheduledEndAt).Format("3:04 PM"))

						conflicts = append(conflicts, ConflictDetail{
							ConflictType: models.ConflictTypeTechnicianOverlap,
							Severity:     models.ConflictSeverityHigh,
							Description:  description,
							ConflictingWith: &ConflictingSchedule{
								ID:        job.ID,
								Title:     job.Title,
								JobNumber: job.JobNumber,
								StartTime: *job.ScheduledStartAt,
								EndTime:   *job.ScheduledEndAt,
							},
							TechnicianID: &techID,
							TimeOverlap:  overlapInfo,
						})
					}
				}
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

	startHour := epoch.EpochHour(req.StartTime)
	endHour := epoch.EpochHour(req.EndTime)
	weekday := epoch.EpochWeekday(req.StartTime)

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
	t := epoch.EpochToTime(req.StartTime)
	dayStart := time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC).Unix()
	dayEnd := dayStart + 86400 // 24 hours in seconds

	for _, techID := range req.AssignedTechIDs {
		// Get all schedules for the day
		schedules, err := d.scheduleRepo.GetByTechnicianAndDateRange(req.TenantID, techID, dayStart, dayEnd)
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
			if req.JobID != uuid.Nil && schedule.JobID == req.JobID {
				continue
			}
			totalMinutes += int(schedule.EndTime-schedule.StartTime) / 60
		}

		// Also count hours from jobs table
		if d.jobRepo != nil {
			jobs, err := d.jobRepo.GetJobsByTechnicianAndDateRange(techID, dayStart, dayEnd)
			if err != nil {
				return nil, err
			}
			for _, job := range jobs {
				if req.JobID != uuid.Nil && job.ID == req.JobID {
					continue
				}
				if job.ScheduledStartAt != nil && job.ScheduledEndAt != nil {
					totalMinutes += int(*job.ScheduledEndAt-*job.ScheduledStartAt) / 60
				}
			}
		}

		// Add the requested schedule duration
		totalMinutes += int(req.EndTime-req.StartTime) / 60

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
func (d *ConflictDetector) hasTimeOverlap(start1, end1, start2, end2 int64) bool {
	return start1 < end2 && end1 > start2
}

// calculateOverlap calculates the overlap between two time ranges
func (d *ConflictDetector) calculateOverlap(start1, end1, start2, end2 int64) *TimeOverlapInfo {
	overlapStart := start1
	if start2 > start1 {
		overlapStart = start2
	}

	overlapEnd := end1
	if end2 < end1 {
		overlapEnd = end2
	}

	return &TimeOverlapInfo{
		OverlapStart:   overlapStart,
		OverlapEnd:     overlapEnd,
		OverlapMinutes: int(overlapEnd-overlapStart) / 60,
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
	if req.StartTime == 0 {
		return fmt.Errorf("start_time is required")
	}
	if req.EndTime == 0 {
		return fmt.Errorf("end_time is required")
	}
	if req.EndTime <= req.StartTime {
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
