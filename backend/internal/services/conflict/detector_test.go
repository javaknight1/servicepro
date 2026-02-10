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

// MockJobRepository is a mock implementation of JobRepositoryInterface for conflict tests
type MockJobRepository struct {
	mock.Mock
}

func (m *MockJobRepository) Create(job *models.Job) error {
	args := m.Called(job)
	return args.Error(0)
}

func (m *MockJobRepository) GetByID(id uuid.UUID) (*models.Job, error) {
	args := m.Called(id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Job), args.Error(1)
}

func (m *MockJobRepository) GetByJobNumber(jobNumber string) (*models.Job, error) {
	args := m.Called(jobNumber)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Job), args.Error(1)
}

func (m *MockJobRepository) Update(job *models.Job) error {
	args := m.Called(job)
	return args.Error(0)
}

func (m *MockJobRepository) Delete(id uuid.UUID) error {
	args := m.Called(id)
	return args.Error(0)
}

func (m *MockJobRepository) List(filter *models.JobListFilter) ([]models.Job, int64, error) {
	args := m.Called(filter)
	return args.Get(0).([]models.Job), args.Get(1).(int64), args.Error(2)
}

func (m *MockJobRepository) GetByCustomer(customerID uuid.UUID, limit, offset int) ([]models.Job, int64, error) {
	args := m.Called(customerID, limit, offset)
	return args.Get(0).([]models.Job), args.Get(1).(int64), args.Error(2)
}

func (m *MockJobRepository) GetByStatus(status models.JobStatus) ([]models.Job, error) {
	args := m.Called(status)
	return args.Get(0).([]models.Job), args.Error(1)
}

func (m *MockJobRepository) GetByAssignedUser(userID uuid.UUID) ([]models.Job, error) {
	args := m.Called(userID)
	return args.Get(0).([]models.Job), args.Error(1)
}

func (m *MockJobRepository) GetScheduledJobs(start, end int64) ([]models.Job, error) {
	args := m.Called(start, end)
	return args.Get(0).([]models.Job), args.Error(1)
}

func (m *MockJobRepository) GetOverdueJobs() ([]models.Job, error) {
	args := m.Called()
	return args.Get(0).([]models.Job), args.Error(1)
}

func (m *MockJobRepository) GetJobsRequiringFollowUp() ([]models.Job, error) {
	args := m.Called()
	return args.Get(0).([]models.Job), args.Error(1)
}

func (m *MockJobRepository) GetJobsByTechnicianAndDateRange(techID uuid.UUID, start, end int64) ([]models.Job, error) {
	args := m.Called(techID, start, end)
	return args.Get(0).([]models.Job), args.Error(1)
}

func (m *MockJobRepository) AddAssignment(assignment *models.JobAssignment) error {
	args := m.Called(assignment)
	return args.Error(0)
}

func (m *MockJobRepository) RemoveAssignment(assignmentID uuid.UUID) error {
	args := m.Called(assignmentID)
	return args.Error(0)
}

func (m *MockJobRepository) GetJobAssignments(jobID uuid.UUID) ([]models.JobAssignment, error) {
	args := m.Called(jobID)
	return args.Get(0).([]models.JobAssignment), args.Error(1)
}

func (m *MockJobRepository) AddMaterial(material *models.JobMaterial) error {
	args := m.Called(material)
	return args.Error(0)
}

func (m *MockJobRepository) UpdateMaterial(material *models.JobMaterial) error {
	args := m.Called(material)
	return args.Error(0)
}

func (m *MockJobRepository) DeleteMaterial(materialID uuid.UUID) error {
	args := m.Called(materialID)
	return args.Error(0)
}

func (m *MockJobRepository) GetJobMaterials(jobID uuid.UUID) ([]models.JobMaterial, error) {
	args := m.Called(jobID)
	return args.Get(0).([]models.JobMaterial), args.Error(1)
}

func (m *MockJobRepository) AddNote(note *models.JobNote) error {
	args := m.Called(note)
	return args.Error(0)
}

func (m *MockJobRepository) GetJobNotes(jobID uuid.UUID, includeInternal bool) ([]models.JobNote, error) {
	args := m.Called(jobID, includeInternal)
	return args.Get(0).([]models.JobNote), args.Error(1)
}

func (m *MockJobRepository) GetJobStats() (map[string]interface{}, error) {
	args := m.Called()
	return args.Get(0).(map[string]interface{}), args.Error(1)
}

func (m *MockJobRepository) GetTechnicianWorkload(userID uuid.UUID) (map[string]interface{}, error) {
	args := m.Called(userID)
	return args.Get(0).(map[string]interface{}), args.Error(1)
}

func (m *MockJobRepository) CreateStatusTransition(transition *models.JobStatusTransition) error {
	args := m.Called(transition)
	return args.Error(0)
}

func (m *MockJobRepository) GetStatusHistory(jobID uuid.UUID, limit, offset int, sortOrder string) ([]models.JobStatusTransition, int64, error) {
	args := m.Called(jobID, limit, offset, sortOrder)
	return args.Get(0).([]models.JobStatusTransition), args.Get(1).(int64), args.Error(2)
}

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

func (m *MockScheduleRepository) GetByDateRange(tenantID uuid.UUID, startDate, endDate int64) ([]models.Schedule, error) {
	args := m.Called(tenantID, startDate, endDate)
	return args.Get(0).([]models.Schedule), args.Error(1)
}

func (m *MockScheduleRepository) GetByTechnicianAndDateRange(tenantID uuid.UUID, techID uuid.UUID, startDate, endDate int64) ([]models.Schedule, error) {
	args := m.Called(tenantID, techID, startDate, endDate)
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
	mockJobRepo := new(MockJobRepository)
	detector := NewConflictDetector(mockRepo, mockJobRepo)

	tenantID := uuid.New()
	techID := uuid.New()
	// Use a fixed time that's within business hours on a weekday (Monday at 9 AM)
	startTime := time.Date(2024, 1, 15, 9, 0, 0, 0, time.UTC).Unix() // Monday
	endTime := startTime + 2*3600                                    // +2 hours

	req := &ConflictCheckRequest{
		TenantID:        tenantID,
		JobID:           uuid.New(),
		StartTime:       startTime,
		EndTime:         endTime,
		AssignedTechIDs: []uuid.UUID{techID},
	}

	// Mock: No existing schedules
	mockRepo.On("GetByTechnicianAndDateRange", tenantID, techID, startTime, endTime).
		Return([]models.Schedule{}, nil)

	// Mock: No existing jobs with conflicts
	mockJobRepo.On("GetJobsByTechnicianAndDateRange", techID, startTime, endTime).
		Return([]models.Job{}, nil)

	// Mock: No workload conflicts (empty day)
	dayStart := time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC).Unix()
	dayEnd := dayStart + 86400
	mockRepo.On("GetByTechnicianAndDateRange", tenantID, techID, dayStart, dayEnd).
		Return([]models.Schedule{}, nil)
	mockJobRepo.On("GetJobsByTechnicianAndDateRange", techID, dayStart, dayEnd).
		Return([]models.Job{}, nil)

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
	mockJobRepo := new(MockJobRepository)
	detector := NewConflictDetector(mockRepo, mockJobRepo)

	tenantID := uuid.New()
	techID := uuid.New()
	startTime := time.Date(2024, 1, 15, 9, 0, 0, 0, time.UTC).Unix()
	endTime := startTime + 2*3600 // +2 hours

	req := &ConflictCheckRequest{
		TenantID:        tenantID,
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
		StartTime:       startTime + 30*60, // +30 minutes
		EndTime:         endTime + 30*60,   // +30 minutes
		AssignedTechIDs: []uuid.UUID{techID},
		IsCancelled:     false,
		IsConfirmed:     true,
	}

	mockRepo.On("GetByTechnicianAndDateRange", tenantID, techID, startTime, endTime).
		Return([]models.Schedule{existingSchedule}, nil)
	mockJobRepo.On("GetJobsByTechnicianAndDateRange", techID, startTime, endTime).
		Return([]models.Job{}, nil)

	// Mock workload check
	dayStart := time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC).Unix()
	dayEnd := dayStart + 86400
	mockRepo.On("GetByTechnicianAndDateRange", tenantID, techID, dayStart, dayEnd).
		Return([]models.Schedule{existingSchedule}, nil)
	mockJobRepo.On("GetJobsByTechnicianAndDateRange", techID, dayStart, dayEnd).
		Return([]models.Job{}, nil)

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
			assert.Equal(t, existingSchedule.ID, conflict.ConflictingWith.ID)
		}
	}
	assert.True(t, hasOverlapConflict, "Should detect technician overlap conflict")

	mockRepo.AssertExpectations(t)
}

// TestCheckConflicts_BusinessHours tests business hours validation
// NOTE: Business hours check is currently disabled (not timezone-aware).
// This test verifies that no false positives are returned when the check is disabled.
func TestCheckConflicts_BusinessHours(t *testing.T) {
	mockRepo := new(MockScheduleRepository)
	mockJobRepo := new(MockJobRepository)
	detector := NewConflictDetector(mockRepo, mockJobRepo)

	tenantID := uuid.New()
	techID := uuid.New()
	// Schedule on Saturday
	startTime := time.Date(2024, 1, 20, 9, 0, 0, 0, time.UTC).Unix() // Saturday
	endTime := startTime + 2*3600                                    // +2 hours

	req := &ConflictCheckRequest{
		TenantID:        tenantID,
		JobID:           uuid.New(),
		StartTime:       startTime,
		EndTime:         endTime,
		AssignedTechIDs: []uuid.UUID{techID},
	}

	mockRepo.On("GetByTechnicianAndDateRange", tenantID, techID, startTime, endTime).
		Return([]models.Schedule{}, nil)
	mockJobRepo.On("GetJobsByTechnicianAndDateRange", techID, startTime, endTime).
		Return([]models.Job{}, nil)

	dayStart := time.Date(2024, 1, 20, 0, 0, 0, 0, time.UTC).Unix()
	dayEnd := dayStart + 86400
	mockRepo.On("GetByTechnicianAndDateRange", tenantID, techID, dayStart, dayEnd).
		Return([]models.Schedule{}, nil)
	mockJobRepo.On("GetJobsByTechnicianAndDateRange", techID, dayStart, dayEnd).
		Return([]models.Job{}, nil)

	ctx := context.Background()
	response, err := detector.CheckConflicts(ctx, req)

	assert.NoError(t, err)
	assert.NotNil(t, response)
	// Business hours check is disabled (not timezone-aware), so no conflicts expected
	assert.False(t, response.HasConflicts)

	mockRepo.AssertExpectations(t)
}

// TestCheckConflicts_WorkloadExcess tests workload validation
func TestCheckConflicts_WorkloadExcess(t *testing.T) {
	mockRepo := new(MockScheduleRepository)
	mockJobRepo := new(MockJobRepository)
	detector := NewConflictDetector(mockRepo, mockJobRepo)

	tenantID := uuid.New()
	techID := uuid.New()
	startTime := time.Date(2024, 1, 15, 14, 0, 0, 0, time.UTC).Unix()
	endTime := startTime + 4*3600 // 4-hour job

	req := &ConflictCheckRequest{
		TenantID:        tenantID,
		JobID:           uuid.New(),
		StartTime:       startTime,
		EndTime:         endTime,
		AssignedTechIDs: []uuid.UUID{techID},
	}

	// Existing schedules totaling 5 hours
	existingSchedules := []models.Schedule{
		{
			ID:              uuid.New(),
			StartTime:       time.Date(2024, 1, 15, 8, 0, 0, 0, time.UTC).Unix(),
			EndTime:         time.Date(2024, 1, 15, 12, 0, 0, 0, time.UTC).Unix(), // 4 hours
			AssignedTechIDs: []uuid.UUID{techID},
			IsCancelled:     false,
		},
		{
			ID:              uuid.New(),
			StartTime:       time.Date(2024, 1, 15, 12, 30, 0, 0, time.UTC).Unix(),
			EndTime:         time.Date(2024, 1, 15, 13, 30, 0, 0, time.UTC).Unix(), // 1 hour
			AssignedTechIDs: []uuid.UUID{techID},
			IsCancelled:     false,
		},
	}

	mockRepo.On("GetByTechnicianAndDateRange", tenantID, techID, startTime, endTime).
		Return([]models.Schedule{}, nil) // No overlap
	mockJobRepo.On("GetJobsByTechnicianAndDateRange", techID, startTime, endTime).
		Return([]models.Job{}, nil)

	dayStart := time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC).Unix()
	dayEnd := dayStart + 86400
	mockRepo.On("GetByTechnicianAndDateRange", tenantID, techID, dayStart, dayEnd).
		Return(existingSchedules, nil)
	mockJobRepo.On("GetJobsByTechnicianAndDateRange", techID, dayStart, dayEnd).
		Return([]models.Job{}, nil)

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
	mockJobRepo := new(MockJobRepository)
	detector := NewConflictDetector(mockRepo, mockJobRepo)

	ctx := context.Background()

	// Test: End time before start time
	now := time.Now().Unix()
	req := &ConflictCheckRequest{
		TenantID:        uuid.New(),
		JobID:           uuid.New(),
		StartTime:       now,
		EndTime:         now - 3600, // -1 hour
		AssignedTechIDs: []uuid.UUID{uuid.New()},
	}

	response, err := detector.CheckConflicts(ctx, req)
	assert.Error(t, err)
	assert.Nil(t, response)
}

// TestCheckConflicts_MultipleTechnicians tests conflicts with multiple technicians
func TestCheckConflicts_MultipleTechnicians(t *testing.T) {
	mockRepo := new(MockScheduleRepository)
	mockJobRepo := new(MockJobRepository)
	detector := NewConflictDetector(mockRepo, mockJobRepo)

	tenantID := uuid.New()
	tech1 := uuid.New()
	tech2 := uuid.New()
	startTime := time.Date(2024, 1, 15, 9, 0, 0, 0, time.UTC).Unix()
	endTime := startTime + 2*3600 // +2 hours

	req := &ConflictCheckRequest{
		TenantID:        tenantID,
		JobID:           uuid.New(),
		StartTime:       startTime,
		EndTime:         endTime,
		AssignedTechIDs: []uuid.UUID{tech1, tech2},
	}

	// Tech1 has no conflicts
	mockRepo.On("GetByTechnicianAndDateRange", tenantID, tech1, startTime, endTime).
		Return([]models.Schedule{}, nil)
	mockJobRepo.On("GetJobsByTechnicianAndDateRange", tech1, startTime, endTime).
		Return([]models.Job{}, nil)
	dayStart := time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC).Unix()
	dayEnd := dayStart + 86400
	mockRepo.On("GetByTechnicianAndDateRange", tenantID, tech1, dayStart, dayEnd).
		Return([]models.Schedule{}, nil)
	mockJobRepo.On("GetJobsByTechnicianAndDateRange", tech1, dayStart, dayEnd).
		Return([]models.Job{}, nil)

	// Tech2 has a conflict
	conflictingSchedule := models.Schedule{
		ID:              uuid.New(),
		StartTime:       startTime,
		EndTime:         endTime,
		AssignedTechIDs: []uuid.UUID{tech2},
		IsCancelled:     false,
	}
	mockRepo.On("GetByTechnicianAndDateRange", tenantID, tech2, startTime, endTime).
		Return([]models.Schedule{conflictingSchedule}, nil)
	mockJobRepo.On("GetJobsByTechnicianAndDateRange", tech2, startTime, endTime).
		Return([]models.Job{}, nil)
	mockRepo.On("GetByTechnicianAndDateRange", tenantID, tech2, dayStart, dayEnd).
		Return([]models.Schedule{conflictingSchedule}, nil)
	mockJobRepo.On("GetJobsByTechnicianAndDateRange", tech2, dayStart, dayEnd).
		Return([]models.Job{}, nil)

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
