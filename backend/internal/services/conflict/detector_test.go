package conflict

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/javaknight1/servicepro/backend/internal/models"
)

// MockScheduleRepository is a mock implementation of ScheduleRepositoryInterface
type MockScheduleRepository struct {
	mock.Mock
}

func (m *MockScheduleRepository) Create(schedule *models.Schedule) error {
	args := m.Called(schedule)
	return args.Error(0)
}

func (m *MockScheduleRepository) GetByID(id uuid.UUID) (*models.Schedule, error) {
	args := m.Called(id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Schedule), args.Error(1)
}

func (m *MockScheduleRepository) Update(schedule *models.Schedule) error {
	args := m.Called(schedule)
	return args.Error(0)
}

func (m *MockScheduleRepository) Delete(id uuid.UUID) error {
	args := m.Called(id)
	return args.Error(0)
}

func (m *MockScheduleRepository) List(params *models.ScheduleQueryParams) ([]models.Schedule, int64, error) {
	args := m.Called(params)
	return args.Get(0).([]models.Schedule), args.Get(1).(int64), args.Error(2)
}

func (m *MockScheduleRepository) GetByDateRange(startDate, endDate time.Time) ([]models.Schedule, error) {
	args := m.Called(startDate, endDate)
	return args.Get(0).([]models.Schedule), args.Error(1)
}

func (m *MockScheduleRepository) GetByTechnicianAndDateRange(techID uuid.UUID, startDate, endDate time.Time) ([]models.Schedule, error) {
	args := m.Called(techID, startDate, endDate)
	if args.Get(0) == nil {
		return []models.Schedule{}, args.Error(1)
	}
	return args.Get(0).([]models.Schedule), args.Error(1)
}

func (m *MockScheduleRepository) GetByJobID(jobID uuid.UUID) ([]models.Schedule, error) {
	args := m.Called(jobID)
	return args.Get(0).([]models.Schedule), args.Error(1)
}

func (m *MockScheduleRepository) DetectConflicts(schedule *models.Schedule) ([]models.ScheduleConflict, error) {
	args := m.Called(schedule)
	return args.Get(0).([]models.ScheduleConflict), args.Error(1)
}

func (m *MockScheduleRepository) GetConflicts(scheduleID uuid.UUID) ([]models.ScheduleConflict, error) {
	args := m.Called(scheduleID)
	return args.Get(0).([]models.ScheduleConflict), args.Error(1)
}

func (m *MockScheduleRepository) ResolveConflict(conflictID uuid.UUID, resolvedBy uuid.UUID, notes *string) error {
	args := m.Called(conflictID, resolvedBy, notes)
	return args.Error(0)
}

func (m *MockScheduleRepository) CreateRecurring(recurring *models.RecurringSchedule) error {
	args := m.Called(recurring)
	return args.Error(0)
}

func (m *MockScheduleRepository) GetRecurringByID(id uuid.UUID) (*models.RecurringSchedule, error) {
	args := m.Called(id)
	return args.Get(0).(*models.RecurringSchedule), args.Error(1)
}

func (m *MockScheduleRepository) UpdateRecurring(recurring *models.RecurringSchedule) error {
	args := m.Called(recurring)
	return args.Error(0)
}

func (m *MockScheduleRepository) DeleteRecurring(id uuid.UUID) error {
	args := m.Called(id)
	return args.Error(0)
}

func (m *MockScheduleRepository) GetActiveRecurring() ([]models.RecurringSchedule, error) {
	args := m.Called()
	return args.Get(0).([]models.RecurringSchedule), args.Error(1)
}

func (m *MockScheduleRepository) GetScheduleCount() (int64, error) {
	args := m.Called()
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockScheduleRepository) GetConflictCount() (int64, error) {
	args := m.Called()
	return args.Get(0).(int64), args.Error(1)
}

// TestCheckConflicts_NoConflicts tests conflict detection with no conflicts
func TestCheckConflicts_NoConflicts(t *testing.T) {
	mockRepo := new(MockScheduleRepository)
	detector := NewConflictDetector(mockRepo)

	techID := uuid.New()
	startTime := time.Now().Add(1 * time.Hour)
	endTime := startTime.Add(2 * time.Hour)

	req := &ConflictCheckRequest{
		JobID:           uuid.New(),
		StartTime:       startTime,
		EndTime:         endTime,
		AssignedTechIDs: []uuid.UUID{techID},
	}

	// Mock: No existing schedules
	mockRepo.On("GetByTechnicianAndDateRange", techID, startTime, endTime).
		Return([]models.Schedule{}, nil)

	// Mock: No workload conflicts (empty day)
	dayStart := time.Date(startTime.Year(), startTime.Month(), startTime.Day(), 0, 0, 0, 0, startTime.Location())
	dayEnd := dayStart.Add(24 * time.Hour)
	mockRepo.On("GetByTechnicianAndDateRange", techID, dayStart, dayEnd).
		Return([]models.Schedule{}, nil)

	ctx := context.Background()
	response, err := detector.CheckConflicts(ctx, req)

	assert.NoError(t, err)
	assert.NotNil(t, response)
	assert.False(t, response.HasConflicts)
	assert.Empty(t, response.Conflicts)
	mockRepo.AssertExpectations(t)
}

// TestCheckConflicts_TechnicianOverlap tests detection of technician overlap
func TestCheckConflicts_TechnicianOverlap(t *testing.T) {
	mockRepo := new(MockScheduleRepository)
	detector := NewConflictDetector(mockRepo)

	techID := uuid.New()
	startTime := time.Date(2024, 1, 15, 9, 0, 0, 0, time.UTC)
	endTime := startTime.Add(2 * time.Hour)

	req := &ConflictCheckRequest{
		JobID:           uuid.New(),
		StartTime:       startTime,
		EndTime:         endTime,
		AssignedTechIDs: []uuid.UUID{techID},
	}

	// Existing overlapping schedule
	existingSchedule := models.Schedule{
		ID:              uuid.New(),
		JobID:           uuid.New(),
		Title:           "Existing Job",
		StartTime:       startTime.Add(30 * time.Minute),
		EndTime:         endTime.Add(30 * time.Minute),
		AssignedTechIDs: []uuid.UUID{techID},
		IsCancelled:     false,
		IsConfirmed:     true,
	}

	mockRepo.On("GetByTechnicianAndDateRange", techID, startTime, endTime).
		Return([]models.Schedule{existingSchedule}, nil)

	// Mock workload check
	dayStart := time.Date(startTime.Year(), startTime.Month(), startTime.Day(), 0, 0, 0, 0, startTime.Location())
	dayEnd := dayStart.Add(24 * time.Hour)
	mockRepo.On("GetByTechnicianAndDateRange", techID, dayStart, dayEnd).
		Return([]models.Schedule{existingSchedule}, nil)

	ctx := context.Background()
	response, err := detector.CheckConflicts(ctx, req)

	assert.NoError(t, err)
	assert.NotNil(t, response)
	assert.True(t, response.HasConflicts)
	assert.NotEmpty(t, response.Conflicts)

	// Check conflict details
	hasOverlapConflict := false
	for _, conflict := range response.Conflicts {
		if conflict.ConflictType == models.ConflictTypeTechnicianOverlap {
			hasOverlapConflict = true
			assert.Equal(t, models.ConflictSeverityHigh, conflict.Severity)
			assert.NotNil(t, conflict.ConflictingWith)
			assert.Equal(t, existingSchedule.ID.String(), conflict.ConflictingWith.ID)
		}
	}
	assert.True(t, hasOverlapConflict, "Should detect technician overlap conflict")

	mockRepo.AssertExpectations(t)
}

// TestCheckConflicts_BusinessHours tests business hours validation
func TestCheckConflicts_BusinessHours(t *testing.T) {
	mockRepo := new(MockScheduleRepository)
	detector := NewConflictDetector(mockRepo)

	techID := uuid.New()
	// Schedule on Saturday
	startTime := time.Date(2024, 1, 20, 9, 0, 0, 0, time.UTC) // Saturday
	endTime := startTime.Add(2 * time.Hour)

	req := &ConflictCheckRequest{
		JobID:           uuid.New(),
		StartTime:       startTime,
		EndTime:         endTime,
		AssignedTechIDs: []uuid.UUID{techID},
	}

	mockRepo.On("GetByTechnicianAndDateRange", techID, startTime, endTime).
		Return([]models.Schedule{}, nil)

	dayStart := time.Date(startTime.Year(), startTime.Month(), startTime.Day(), 0, 0, 0, 0, startTime.Location())
	dayEnd := dayStart.Add(24 * time.Hour)
	mockRepo.On("GetByTechnicianAndDateRange", techID, dayStart, dayEnd).
		Return([]models.Schedule{}, nil)

	ctx := context.Background()
	response, err := detector.CheckConflicts(ctx, req)

	assert.NoError(t, err)
	assert.NotNil(t, response)
	assert.True(t, response.HasConflicts)

	// Should have business hours conflict
	hasBusinessHourConflict := false
	for _, conflict := range response.Conflicts {
		if conflict.ConflictType == models.ConflictTypeBusinessHours {
			hasBusinessHourConflict = true
			break
		}
	}
	assert.True(t, hasBusinessHourConflict, "Should detect weekend scheduling")

	mockRepo.AssertExpectations(t)
}

// TestCheckConflicts_WorkloadExcess tests workload validation
func TestCheckConflicts_WorkloadExcess(t *testing.T) {
	mockRepo := new(MockScheduleRepository)
	detector := NewConflictDetector(mockRepo)

	techID := uuid.New()
	startTime := time.Date(2024, 1, 15, 14, 0, 0, 0, time.UTC)
	endTime := startTime.Add(4 * time.Hour) // 4-hour job

	req := &ConflictCheckRequest{
		JobID:           uuid.New(),
		StartTime:       startTime,
		EndTime:         endTime,
		AssignedTechIDs: []uuid.UUID{techID},
	}

	// Existing schedules totaling 7 hours
	existingSchedules := []models.Schedule{
		{
			ID:              uuid.New(),
			StartTime:       time.Date(2024, 1, 15, 8, 0, 0, 0, time.UTC),
			EndTime:         time.Date(2024, 1, 15, 12, 0, 0, 0, time.UTC), // 4 hours
			AssignedTechIDs: []uuid.UUID{techID},
			IsCancelled:     false,
		},
		{
			ID:              uuid.New(),
			StartTime:       time.Date(2024, 1, 15, 12, 30, 0, 0, time.UTC),
			EndTime:         time.Date(2024, 1, 15, 13, 30, 0, 0, time.UTC), // 1 hour
			AssignedTechIDs: []uuid.UUID{techID},
			IsCancelled:     false,
		},
	}

	mockRepo.On("GetByTechnicianAndDateRange", techID, startTime, endTime).
		Return([]models.Schedule{}, nil) // No overlap

	dayStart := time.Date(startTime.Year(), startTime.Month(), startTime.Day(), 0, 0, 0, 0, startTime.Location())
	dayEnd := dayStart.Add(24 * time.Hour)
	mockRepo.On("GetByTechnicianAndDateRange", techID, dayStart, dayEnd).
		Return(existingSchedules, nil)

	ctx := context.Background()
	response, err := detector.CheckConflicts(ctx, req)

	assert.NoError(t, err)
	assert.NotNil(t, response)
	assert.True(t, response.HasConflicts)

	// Should have workload excess conflict
	hasWorkloadConflict := false
	for _, conflict := range response.Conflicts {
		if conflict.ConflictType == models.ConflictTypeWorkloadExcess {
			hasWorkloadConflict = true
			// Total: 4 + 1 + 4 = 9 hours (exceeds 8-hour threshold)
			assert.Contains(t, conflict.Description, "9.0 hours")
			break
		}
	}
	assert.True(t, hasWorkloadConflict, "Should detect workload excess")

	mockRepo.AssertExpectations(t)
}

// TestCheckConflicts_InvalidRequest tests validation errors
func TestCheckConflicts_InvalidRequest(t *testing.T) {
	mockRepo := new(MockScheduleRepository)
	detector := NewConflictDetector(mockRepo)

	ctx := context.Background()

	// Test: End time before start time
	req := &ConflictCheckRequest{
		JobID:           uuid.New(),
		StartTime:       time.Now(),
		EndTime:         time.Now().Add(-1 * time.Hour),
		AssignedTechIDs: []uuid.UUID{uuid.New()},
	}

	response, err := detector.CheckConflicts(ctx, req)
	assert.Error(t, err)
	assert.Nil(t, response)
}

// TestCheckConflicts_MultipleTechnicians tests conflicts with multiple technicians
func TestCheckConflicts_MultipleTechnicians(t *testing.T) {
	mockRepo := new(MockScheduleRepository)
	detector := NewConflictDetector(mockRepo)

	tech1 := uuid.New()
	tech2 := uuid.New()
	startTime := time.Date(2024, 1, 15, 9, 0, 0, 0, time.UTC)
	endTime := startTime.Add(2 * time.Hour)

	req := &ConflictCheckRequest{
		JobID:           uuid.New(),
		StartTime:       startTime,
		EndTime:         endTime,
		AssignedTechIDs: []uuid.UUID{tech1, tech2},
	}

	// Tech1 has no conflicts
	mockRepo.On("GetByTechnicianAndDateRange", tech1, startTime, endTime).
		Return([]models.Schedule{}, nil)
	dayStart := time.Date(startTime.Year(), startTime.Month(), startTime.Day(), 0, 0, 0, 0, startTime.Location())
	dayEnd := dayStart.Add(24 * time.Hour)
	mockRepo.On("GetByTechnicianAndDateRange", tech1, dayStart, dayEnd).
		Return([]models.Schedule{}, nil)

	// Tech2 has a conflict
	conflictingSchedule := models.Schedule{
		ID:              uuid.New(),
		StartTime:       startTime,
		EndTime:         endTime,
		AssignedTechIDs: []uuid.UUID{tech2},
		IsCancelled:     false,
	}
	mockRepo.On("GetByTechnicianAndDateRange", tech2, startTime, endTime).
		Return([]models.Schedule{conflictingSchedule}, nil)
	mockRepo.On("GetByTechnicianAndDateRange", tech2, dayStart, dayEnd).
		Return([]models.Schedule{conflictingSchedule}, nil)

	ctx := context.Background()
	response, err := detector.CheckConflicts(ctx, req)

	assert.NoError(t, err)
	assert.NotNil(t, response)
	assert.True(t, response.HasConflicts)

	// Should detect conflict for tech2
	hasConflict := false
	for _, conflict := range response.Conflicts {
		if conflict.ConflictType == models.ConflictTypeTechnicianOverlap && conflict.TechnicianID != nil {
			if *conflict.TechnicianID == tech2 {
				hasConflict = true
			}
		}
	}
	assert.True(t, hasConflict, "Should detect conflict for tech2")

	mockRepo.AssertExpectations(t)
}
