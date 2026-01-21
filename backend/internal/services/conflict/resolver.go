package conflict

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/javaknight1/servicepro/backend/internal/models"
	"github.com/javaknight1/servicepro/backend/internal/repository"
)

// ConflictResolver provides intelligent conflict resolution suggestions
type ConflictResolver struct {
	scheduleRepo repository.ScheduleRepositoryInterface
	userRepo     repository.UserRepositoryInterface
}

// NewConflictResolver creates a new conflict resolver
func NewConflictResolver(
	scheduleRepo repository.ScheduleRepositoryInterface,
	userRepo repository.UserRepositoryInterface,
) *ConflictResolver {
	return &ConflictResolver{
		scheduleRepo: scheduleRepo,
		userRepo:     userRepo,
	}
}

// ResolutionStrategy represents a conflict resolution strategy
type ResolutionStrategy struct {
	ID                string      `json:"id"`
	Type              string      `json:"type"`
	Title             string      `json:"title"`
	Description       string      `json:"description"`
	Confidence        float64     `json:"confidence"`   // 0-1 score
	ImpactLevel       string      `json:"impact_level"` // low, medium, high
	AutoApplicable    bool        `json:"auto_applicable"`
	Suggestion        interface{} `json:"suggestion"`
	ConflictsResolved int         `json:"conflicts_resolved"`
}

// TimeSlotSuggestion represents an alternative time slot
type TimeSlotSuggestion struct {
	StartTime      time.Time `json:"start_time"`
	EndTime        time.Time `json:"end_time"`
	AvailableSlots int       `json:"available_slots"`
	Reason         string    `json:"reason"`
}

// TechnicianSuggestion represents an alternative technician
type TechnicianSuggestion struct {
	TechnicianID    uuid.UUID `json:"technician_id"`
	Name            string    `json:"name"`
	Email           string    `json:"email"`
	Availability    string    `json:"availability"`
	CurrentWorkload int       `json:"current_workload"` // minutes
	Skills          []string  `json:"skills,omitempty"`
}

// SplitScheduleSuggestion represents a suggestion to split the schedule
type SplitScheduleSuggestion struct {
	SplitCount int               `json:"split_count"`
	Segments   []ScheduleSegment `json:"segments"`
	Reason     string            `json:"reason"`
}

// ScheduleSegment represents a segment of a split schedule
type ScheduleSegment struct {
	StartTime       time.Time   `json:"start_time"`
	EndTime         time.Time   `json:"end_time"`
	AssignedTechIDs []uuid.UUID `json:"assigned_tech_ids"`
	Notes           string      `json:"notes,omitempty"`
}

// ResolveConflicts generates resolution strategies for detected conflicts
func (r *ConflictResolver) ResolveConflicts(ctx context.Context, req *ConflictCheckRequest, conflicts []ConflictDetail) ([]ResolutionStrategy, error) {
	strategies := []ResolutionStrategy{}

	// Analyze conflict types
	hasTechConflict := false
	hasWorkloadConflict := false
	hasBusinessHourConflict := false
	conflictCount := len(conflicts)

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

	// Strategy 1: Find alternative time slots
	if hasTechConflict {
		timeStrategies, err := r.findAlternativeTimeSlots(ctx, req)
		if err == nil && len(timeStrategies) > 0 {
			strategies = append(strategies, ResolutionStrategy{
				ID:                fmt.Sprintf("time-slot-%d", time.Now().Unix()),
				Type:              "reschedule",
				Title:             "Reschedule to Alternative Time",
				Description:       "Move the schedule to a time when all technicians are available",
				Confidence:        0.85,
				ImpactLevel:       "medium",
				AutoApplicable:    false,
				Suggestion:        timeStrategies,
				ConflictsResolved: conflictCount,
			})
		}
	}

	// Strategy 2: Find alternative technicians
	if hasTechConflict {
		techStrategies, err := r.findAlternativeTechnicians(ctx, req)
		if err == nil && len(techStrategies) > 0 {
			strategies = append(strategies, ResolutionStrategy{
				ID:                fmt.Sprintf("tech-alt-%d", time.Now().Unix()),
				Type:              "reassign_technician",
				Title:             "Assign Different Technician",
				Description:       "Replace conflicting technician(s) with available alternatives",
				Confidence:        0.75,
				ImpactLevel:       "low",
				AutoApplicable:    true,
				Suggestion:        techStrategies,
				ConflictsResolved: conflictCount,
			})
		}
	}

	// Strategy 3: Split the schedule
	if hasWorkloadConflict || (hasTechConflict && r.isLongDuration(req.StartTime, req.EndTime)) {
		splitStrategy, err := r.suggestScheduleSplit(ctx, req)
		if err == nil && splitStrategy != nil {
			strategies = append(strategies, ResolutionStrategy{
				ID:                fmt.Sprintf("split-%d", time.Now().Unix()),
				Type:              "split_schedule",
				Title:             "Split Into Multiple Sessions",
				Description:       "Divide the job into multiple shorter sessions to reduce conflicts",
				Confidence:        0.70,
				ImpactLevel:       "high",
				AutoApplicable:    false,
				Suggestion:        splitStrategy,
				ConflictsResolved: conflictCount,
			})
		}
	}

	// Strategy 4: Request overtime approval
	if hasBusinessHourConflict {
		strategies = append(strategies, ResolutionStrategy{
			ID:                fmt.Sprintf("overtime-%d", time.Now().Unix()),
			Type:              "overtime_approval",
			Title:             "Request Overtime Approval",
			Description:       "Proceed with current schedule after obtaining overtime approval",
			Confidence:        0.60,
			ImpactLevel:       "low",
			AutoApplicable:    false,
			Suggestion:        nil,
			ConflictsResolved: 1,
		})
	}

	// Strategy 5: Delay less important conflicting schedules
	if hasTechConflict {
		delayStrategy, err := r.suggestDelayConflicts(ctx, req, conflicts)
		if err == nil && delayStrategy != nil {
			strategies = append(strategies, *delayStrategy)
		}
	}

	return strategies, nil
}

// findAlternativeTimeSlots finds available time slots for the schedule
func (r *ConflictResolver) findAlternativeTimeSlots(ctx context.Context, req *ConflictCheckRequest) ([]TimeSlotSuggestion, error) {
	suggestions := []TimeSlotSuggestion{}
	duration := req.EndTime.Sub(req.StartTime)

	// Search for slots in the next 7 days
	searchEnd := req.StartTime.AddDate(0, 0, 7)

	// Try to find slots on the same day first
	sameDaySlots := r.findSlotsOnDay(ctx, req, req.StartTime, duration)
	suggestions = append(suggestions, sameDaySlots...)

	// Then try next business day
	nextDay := r.nextBusinessDay(req.StartTime)
	if nextDay.Before(searchEnd) {
		nextDaySlots := r.findSlotsOnDay(ctx, req, nextDay, duration)
		suggestions = append(suggestions, nextDaySlots...)
	}

	// Try the following business day
	dayAfterNext := r.nextBusinessDay(nextDay)
	if dayAfterNext.Before(searchEnd) {
		dayAfterNextSlots := r.findSlotsOnDay(ctx, req, dayAfterNext, duration)
		suggestions = append(suggestions, dayAfterNextSlots...)
	}

	return suggestions, nil
}

// findSlotsOnDay finds available slots on a specific day
func (r *ConflictResolver) findSlotsOnDay(ctx context.Context, req *ConflictCheckRequest, day time.Time, duration time.Duration) []TimeSlotSuggestion {
	slots := []TimeSlotSuggestion{}

	// Define business hours (8 AM - 6 PM)
	dayStart := time.Date(day.Year(), day.Month(), day.Day(), 8, 0, 0, 0, day.Location())
	dayEnd := time.Date(day.Year(), day.Month(), day.Day(), 18, 0, 0, 0, day.Location())

	// Try slots every hour
	for slotStart := dayStart; slotStart.Add(duration).Before(dayEnd) || slotStart.Add(duration).Equal(dayEnd); slotStart = slotStart.Add(1 * time.Hour) {
		slotEnd := slotStart.Add(duration)

		// Check if all technicians are available
		allAvailable := true
		for _, techID := range req.AssignedTechIDs {
			schedules, _ := r.scheduleRepo.GetByTechnicianAndDateRange(techID, slotStart, slotEnd)
			for _, schedule := range schedules {
				if !schedule.IsCancelled && r.hasOverlap(slotStart, slotEnd, schedule.StartTime, schedule.EndTime) {
					allAvailable = false
					break
				}
			}
			if !allAvailable {
				break
			}
		}

		if allAvailable {
			reason := "All technicians available"
			if !day.Equal(req.StartTime) {
				daysDiff := int(day.Sub(req.StartTime).Hours() / 24)
				reason = fmt.Sprintf("Available %d day(s) later", daysDiff)
			}

			slots = append(slots, TimeSlotSuggestion{
				StartTime:      slotStart,
				EndTime:        slotEnd,
				AvailableSlots: len(req.AssignedTechIDs),
				Reason:         reason,
			})
		}
	}

	return slots
}

// findAlternativeTechnicians finds technicians available during the requested time
func (r *ConflictResolver) findAlternativeTechnicians(ctx context.Context, req *ConflictCheckRequest) ([]TechnicianSuggestion, error) {
	suggestions := []TechnicianSuggestion{}

	// Get all users with technician role
	// This is simplified - in production, you'd query for users with specific role
	// For now, we'll just check all users and see who's available

	// Get conflicting technicians
	conflictingTechs := make(map[uuid.UUID]bool)
	for _, techID := range req.AssignedTechIDs {
		schedules, _ := r.scheduleRepo.GetByTechnicianAndDateRange(techID, req.StartTime, req.EndTime)
		for _, schedule := range schedules {
			if !schedule.IsCancelled && r.hasOverlap(req.StartTime, req.EndTime, schedule.StartTime, schedule.EndTime) {
				conflictingTechs[techID] = true
			}
		}
	}

	// For each conflicting tech, try to find alternatives
	// This is a placeholder - in production, you'd query users with appropriate roles/skills
	// For now, we'll return a placeholder suggestion
	if len(conflictingTechs) > 0 {
		suggestions = append(suggestions, TechnicianSuggestion{
			TechnicianID:    uuid.New(), // Placeholder
			Name:            "Available Technician",
			Email:           "tech@example.com",
			Availability:    "Available during requested time",
			CurrentWorkload: 0,
		})
	}

	return suggestions, nil
}

// suggestScheduleSplit suggests splitting a schedule into multiple segments
func (r *ConflictResolver) suggestScheduleSplit(ctx context.Context, req *ConflictCheckRequest) (*SplitScheduleSuggestion, error) {
	duration := req.EndTime.Sub(req.StartTime)

	// Only suggest split if duration is at least 2 hours
	if duration < 2*time.Hour {
		return nil, nil
	}

	// Suggest splitting into 2 segments
	splitPoint := req.StartTime.Add(duration / 2)

	segments := []ScheduleSegment{
		{
			StartTime:       req.StartTime,
			EndTime:         splitPoint,
			AssignedTechIDs: req.AssignedTechIDs,
			Notes:           "First session",
		},
		{
			StartTime:       splitPoint,
			EndTime:         req.EndTime,
			AssignedTechIDs: req.AssignedTechIDs,
			Notes:           "Second session",
		},
	}

	return &SplitScheduleSuggestion{
		SplitCount: 2,
		Segments:   segments,
		Reason:     "Splitting the job into smaller segments reduces technician workload and increases scheduling flexibility",
	}, nil
}

// suggestDelayConflicts suggests delaying less important conflicting schedules
func (r *ConflictResolver) suggestDelayConflicts(ctx context.Context, req *ConflictCheckRequest, conflicts []ConflictDetail) (*ResolutionStrategy, error) {
	// Find conflicting schedules that are not confirmed
	unconfirmedConflicts := []uuid.UUID{}

	for _, conflict := range conflicts {
		if conflict.ConflictingWith != nil {
			schedule, err := r.scheduleRepo.GetByID(conflict.ConflictingWith.ID)
			if err == nil && !schedule.IsConfirmed {
				unconfirmedConflicts = append(unconfirmedConflicts, schedule.ID)
			}
		}
	}

	if len(unconfirmedConflicts) == 0 {
		return nil, nil
	}

	return &ResolutionStrategy{
		ID:                fmt.Sprintf("delay-%d", time.Now().Unix()),
		Type:              "delay_conflicts",
		Title:             "Reschedule Conflicting Jobs",
		Description:       fmt.Sprintf("Move %d unconfirmed conflicting schedule(s) to later time slots", len(unconfirmedConflicts)),
		Confidence:        0.65,
		ImpactLevel:       "medium",
		AutoApplicable:    false,
		Suggestion:        unconfirmedConflicts,
		ConflictsResolved: len(unconfirmedConflicts),
	}, nil
}

// Helper functions

func (r *ConflictResolver) hasOverlap(start1, end1, start2, end2 time.Time) bool {
	return start1.Before(end2) && end1.After(start2)
}

func (r *ConflictResolver) isLongDuration(start, end time.Time) bool {
	duration := end.Sub(start)
	return duration >= 3*time.Hour
}

func (r *ConflictResolver) nextBusinessDay(date time.Time) time.Time {
	next := date.AddDate(0, 0, 1)

	// Skip weekends
	for next.Weekday() == time.Saturday || next.Weekday() == time.Sunday {
		next = next.AddDate(0, 0, 1)
	}

	// Set to start of business day (8 AM)
	return time.Date(next.Year(), next.Month(), next.Day(), 8, 0, 0, 0, next.Location())
}

// ApplyResolutionStrategy applies an automatic resolution strategy
func (r *ConflictResolver) ApplyResolutionStrategy(ctx context.Context, strategyID string, scheduleID uuid.UUID) error {
	// This is a placeholder for automatic resolution
	// In production, this would actually apply the suggested changes
	return fmt.Errorf("automatic resolution not yet implemented")
}

// GetResolutionHistory retrieves the history of conflict resolutions
func (r *ConflictResolver) GetResolutionHistory(ctx context.Context, scheduleID uuid.UUID) ([]models.ScheduleConflict, error) {
	return r.scheduleRepo.GetConflicts(scheduleID)
}
