package services

import (
	"context"
	"errors"
	"log"
	"time"

	"github.com/google/uuid"

	"github.com/javaknight1/servicepro/backend/internal/models"
	"github.com/javaknight1/servicepro/backend/internal/repository"
)

var (
	ErrScheduleNotFound = errors.New("schedule not found")
	ErrInvalidTimeRange = errors.New("invalid time range")
)

// ScheduleServiceInterface defines the interface for schedule business logic
type ScheduleServiceInterface interface {
	CreateSchedule(ctx context.Context, req *models.ScheduleRequest, createdBy uuid.UUID) (*models.ScheduleResponse, error)
	GetSchedule(ctx context.Context, id uuid.UUID) (*models.ScheduleResponse, error)
	UpdateSchedule(ctx context.Context, id uuid.UUID, req *models.ScheduleRequest, updatedBy uuid.UUID) (*models.ScheduleResponse, error)
	DeleteSchedule(ctx context.Context, id uuid.UUID) error
	ListSchedules(ctx context.Context, params *models.ScheduleQueryParams) (*models.ScheduleListResponse, error)
	GetTechnicianSchedule(ctx context.Context, techID uuid.UUID, startDate, endDate time.Time) ([]models.ScheduleResponse, error)
	DetectAndResolveConflicts(ctx context.Context, scheduleID uuid.UUID) ([]models.ScheduleConflict, error)
}

// ScheduleService implements schedule business logic
type ScheduleService struct {
	repo   repository.ScheduleRepositoryInterface
	logger *log.Logger
}

// NewScheduleService creates a new schedule service
func NewScheduleService(repo repository.ScheduleRepositoryInterface) *ScheduleService {
	return &ScheduleService{
		repo:   repo,
		logger: log.New(log.Writer(), "[ScheduleService] ", log.LstdFlags),
	}
}

// CreateSchedule creates a new schedule with conflict detection
func (s *ScheduleService) CreateSchedule(ctx context.Context, req *models.ScheduleRequest, createdBy uuid.UUID) (*models.ScheduleResponse, error) {
	s.logger.Printf("Creating schedule for job %s", req.JobID)

	// Validate time range
	if req.EndTime.Before(req.StartTime) || req.EndTime.Equal(req.StartTime) {
		return nil, ErrInvalidTimeRange
	}

	// Create schedule model
	schedule := &models.Schedule{
		JobID:               req.JobID,
		Title:               req.Title,
		Description:         req.Description,
		StartTime:           req.StartTime,
		EndTime:             req.EndTime,
		AllDay:              req.AllDay,
		RecurrenceType:      req.RecurrenceType,
		RecurringScheduleID: req.RecurringScheduleID,
		AssignedTechIDs:     req.AssignedTechIDs,
		Location:            req.Location,
		Notes:               req.Notes,
		Color:               req.Color,
		RemindersEnabled:    req.RemindersEnabled,
		ReminderTime:        req.ReminderTime,
		CreatedBy:           createdBy,
	}

	// Validate schedule
	if err := schedule.Validate(); err != nil {
		return nil, err
	}

	// Create in database
	if err := s.repo.Create(schedule); err != nil {
		s.logger.Printf("Failed to create schedule: %v", err)
		return nil, err
	}

	// Detect conflicts asynchronously
	go func() {
		conflicts, err := s.repo.DetectConflicts(schedule)
		if err != nil {
			s.logger.Printf("Failed to detect conflicts: %v", err)
			return
		}
		if len(conflicts) > 0 {
			s.logger.Printf("Detected %d conflicts for schedule %s", len(conflicts), schedule.ID)
		}
	}()

	s.logger.Printf("Schedule created successfully: %s", schedule.ID)
	return s.GetSchedule(ctx, schedule.ID)
}

// GetSchedule retrieves a schedule by ID
func (s *ScheduleService) GetSchedule(ctx context.Context, id uuid.UUID) (*models.ScheduleResponse, error) {
	schedule, err := s.repo.GetByID(id)
	if err != nil {
		return nil, ErrScheduleNotFound
	}

	response := schedule.ToResponse()
	return &response, nil
}

// UpdateSchedule updates an existing schedule
func (s *ScheduleService) UpdateSchedule(ctx context.Context, id uuid.UUID, req *models.ScheduleRequest, updatedBy uuid.UUID) (*models.ScheduleResponse, error) {
	s.logger.Printf("Updating schedule %s", id)

	// Get existing schedule
	schedule, err := s.repo.GetByID(id)
	if err != nil {
		return nil, ErrScheduleNotFound
	}

	// Validate time range
	if req.EndTime.Before(req.StartTime) || req.EndTime.Equal(req.StartTime) {
		return nil, ErrInvalidTimeRange
	}

	// Update fields
	schedule.Title = req.Title
	schedule.Description = req.Description
	schedule.StartTime = req.StartTime
	schedule.EndTime = req.EndTime
	schedule.AllDay = req.AllDay
	schedule.RecurrenceType = req.RecurrenceType
	schedule.RecurringScheduleID = req.RecurringScheduleID
	schedule.AssignedTechIDs = req.AssignedTechIDs
	schedule.Location = req.Location
	schedule.Notes = req.Notes
	schedule.Color = req.Color
	schedule.RemindersEnabled = req.RemindersEnabled
	schedule.ReminderTime = req.ReminderTime
	schedule.UpdatedBy = &updatedBy

	// Validate schedule
	if err := schedule.Validate(); err != nil {
		return nil, err
	}

	// Update in database
	if err := s.repo.Update(schedule); err != nil {
		s.logger.Printf("Failed to update schedule: %v", err)
		return nil, err
	}

	// Re-detect conflicts
	go func() {
		conflicts, err := s.repo.DetectConflicts(schedule)
		if err != nil {
			s.logger.Printf("Failed to detect conflicts: %v", err)
			return
		}
		if len(conflicts) > 0 {
			s.logger.Printf("Detected %d conflicts for updated schedule %s", len(conflicts), schedule.ID)
		}
	}()

	s.logger.Printf("Schedule updated successfully: %s", id)
	return s.GetSchedule(ctx, id)
}

// DeleteSchedule soft deletes a schedule
func (s *ScheduleService) DeleteSchedule(ctx context.Context, id uuid.UUID) error {
	s.logger.Printf("Deleting schedule %s", id)

	// Verify schedule exists
	if _, err := s.repo.GetByID(id); err != nil {
		return ErrScheduleNotFound
	}

	if err := s.repo.Delete(id); err != nil {
		s.logger.Printf("Failed to delete schedule: %v", err)
		return err
	}

	s.logger.Printf("Schedule deleted successfully: %s", id)
	return nil
}

// ListSchedules retrieves a paginated list of schedules
func (s *ScheduleService) ListSchedules(ctx context.Context, params *models.ScheduleQueryParams) (*models.ScheduleListResponse, error) {
	schedules, total, err := s.repo.List(params)
	if err != nil {
		return nil, err
	}

	responses := make([]models.ScheduleResponse, len(schedules))
	for i, schedule := range schedules {
		responses[i] = schedule.ToResponse()
	}

	return &models.ScheduleListResponse{
		Schedules: responses,
		Total:     total,
		Page:      params.Page,
		PageSize:  params.PageSize,
	}, nil
}

// GetTechnicianSchedule retrieves all schedules for a technician in a date range
func (s *ScheduleService) GetTechnicianSchedule(ctx context.Context, techID uuid.UUID, startDate, endDate time.Time) ([]models.ScheduleResponse, error) {
	schedules, err := s.repo.GetByTechnicianAndDateRange(techID, startDate, endDate)
	if err != nil {
		return nil, err
	}

	responses := make([]models.ScheduleResponse, len(schedules))
	for i, schedule := range schedules {
		responses[i] = schedule.ToResponse()
	}

	return responses, nil
}

// DetectAndResolveConflicts detects conflicts for a schedule
func (s *ScheduleService) DetectAndResolveConflicts(ctx context.Context, scheduleID uuid.UUID) ([]models.ScheduleConflict, error) {
	schedule, err := s.repo.GetByID(scheduleID)
	if err != nil {
		return nil, ErrScheduleNotFound
	}

	conflicts, err := s.repo.DetectConflicts(schedule)
	if err != nil {
		return nil, err
	}

	s.logger.Printf("Detected %d conflicts for schedule %s", len(conflicts), scheduleID)
	return conflicts, nil
}
