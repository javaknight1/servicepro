package services

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/javaknight1/servicepro/backend/internal/models"
)

// TestCreateAssignment_Success tests successful assignment creation
func TestCreateAssignment_Success(t *testing.T) {
	mockJobRepo := new(MockJobRepository)
	mockUserRepo := new(MockUserRepository)
	service := NewAssignmentService(mockJobRepo, mockUserRepo, nil)

	ctx := context.Background()
	jobID := uuid.New()
	technicianID := uuid.New()
	assignedBy := uuid.New()

	now := time.Now()
	startTime := now.Add(24 * time.Hour)
	endTime := startTime.Add(2 * time.Hour)

	job := &models.Job{
		ID:               jobID,
		JobNumber:        "JOB-001",
		Title:            "Test Job",
		Status:           models.JobStatusScheduled,
		ScheduledStartAt: &startTime,
		ScheduledEndAt:   &endTime,
	}

	technician := &models.User{
		ID:    technicianID,
		Email: "tech@example.com",
		Role:  models.UserRoleTechnician,
	}

	req := &AssignmentRequest{
		JobID:        jobID,
		TechnicianID: technicianID,
		Role:         "Lead Technician",
	}

	mockJobRepo.On("GetByID", jobID).Return(job, nil)
	mockUserRepo.On("GetByID", technicianID).Return(technician, nil)
	mockJobRepo.On("GetByAssignedUser", technicianID).Return([]models.Job{}, nil)
	mockJobRepo.On("AddAssignment", mock.AnythingOfType("*models.JobAssignment")).Return(nil)

	assignment, err := service.CreateAssignment(ctx, req, assignedBy)

	require.NoError(t, err)
	assert.NotNil(t, assignment)
	assert.Equal(t, jobID, assignment.JobID)
	assert.Equal(t, technicianID, assignment.UserID)
	assert.Equal(t, "Lead Technician", assignment.Role)
	mockJobRepo.AssertExpectations(t)
	mockUserRepo.AssertExpectations(t)
}

// TestCreateAssignment_JobNotFound tests job not found error
func TestCreateAssignment_JobNotFound(t *testing.T) {
	mockJobRepo := new(MockJobRepository)
	mockUserRepo := new(MockUserRepository)
	service := NewAssignmentService(mockJobRepo, mockUserRepo, nil)

	ctx := context.Background()
	jobID := uuid.New()

	req := &AssignmentRequest{
		JobID:        jobID,
		TechnicianID: uuid.New(),
		Role:         "Technician",
	}

	mockJobRepo.On("GetByID", jobID).Return(nil, gorm.ErrRecordNotFound)

	assignment, err := service.CreateAssignment(ctx, req, uuid.New())

	assert.Error(t, err)
	assert.Nil(t, assignment)
	assert.ErrorIs(t, err, ErrJobNotFound)
	mockJobRepo.AssertExpectations(t)
}

// TestCreateAssignment_TechnicianNotFound tests technician not found error
func TestCreateAssignment_TechnicianNotFound(t *testing.T) {
	mockJobRepo := new(MockJobRepository)
	mockUserRepo := new(MockUserRepository)
	service := NewAssignmentService(mockJobRepo, mockUserRepo, nil)

	ctx := context.Background()
	technicianID := uuid.New()

	job := &models.Job{
		ID:        uuid.New(),
		JobNumber: "JOB-001",
	}

	req := &AssignmentRequest{
		JobID:        job.ID,
		TechnicianID: technicianID,
		Role:         "Technician",
	}

	mockJobRepo.On("GetByID", job.ID).Return(job, nil)
	mockUserRepo.On("GetByID", technicianID).Return(nil, gorm.ErrRecordNotFound)

	assignment, err := service.CreateAssignment(ctx, req, uuid.New())

	assert.Error(t, err)
	assert.Nil(t, assignment)
	assert.ErrorIs(t, err, ErrTechnicianNotFound)
	mockJobRepo.AssertExpectations(t)
	mockUserRepo.AssertExpectations(t)
}

// TestCreateAssignment_NotTechnicianRole tests error when user is not a technician
func TestCreateAssignment_NotTechnicianRole(t *testing.T) {
	mockJobRepo := new(MockJobRepository)
	mockUserRepo := new(MockUserRepository)
	service := NewAssignmentService(mockJobRepo, mockUserRepo, nil)

	ctx := context.Background()
	userID := uuid.New()

	job := &models.Job{
		ID:        uuid.New(),
		JobNumber: "JOB-001",
	}

	user := &models.User{
		ID:    userID,
		Email: "admin@example.com",
		Role:  models.UserRoleAdmin, // Not a technician!
	}

	req := &AssignmentRequest{
		JobID:        job.ID,
		TechnicianID: userID,
		Role:         "Technician",
	}

	mockJobRepo.On("GetByID", job.ID).Return(job, nil)
	mockUserRepo.On("GetByID", userID).Return(user, nil)

	assignment, err := service.CreateAssignment(ctx, req, uuid.New())

	assert.Error(t, err)
	assert.Nil(t, assignment)
	assert.Contains(t, err.Error(), "must have technician role")
	mockJobRepo.AssertExpectations(t)
	mockUserRepo.AssertExpectations(t)
}

// TestCreateAssignment_TechnicianNotAvailable tests unavailable technician
func TestCreateAssignment_TechnicianNotAvailable(t *testing.T) {
	mockJobRepo := new(MockJobRepository)
	mockUserRepo := new(MockUserRepository)
	service := NewAssignmentService(mockJobRepo, mockUserRepo, nil)

	ctx := context.Background()
	technicianID := uuid.New()

	now := time.Now()
	startTime := now.Add(24 * time.Hour)
	endTime := startTime.Add(2 * time.Hour)

	job := &models.Job{
		ID:               uuid.New(),
		JobNumber:        "JOB-001",
		ScheduledStartAt: &startTime,
		ScheduledEndAt:   &endTime,
	}

	technician := &models.User{
		ID:    technicianID,
		Email: "tech@example.com",
		Role:  models.UserRoleTechnician,
	}

	// Conflicting job
	conflictJob := models.Job{
		ID:               uuid.New(),
		JobNumber:        "JOB-CONFLICT",
		Status:           models.JobStatusScheduled,
		ScheduledStartAt: &startTime,
		ScheduledEndAt:   &endTime,
	}

	req := &AssignmentRequest{
		JobID:        job.ID,
		TechnicianID: technicianID,
		Role:         "Technician",
	}

	mockJobRepo.On("GetByID", job.ID).Return(job, nil)
	mockUserRepo.On("GetByID", technicianID).Return(technician, nil)
	mockJobRepo.On("GetByAssignedUser", technicianID).Return([]models.Job{conflictJob}, nil)

	assignment, err := service.CreateAssignment(ctx, req, uuid.New())

	assert.Error(t, err)
	assert.Nil(t, assignment)
	assert.ErrorIs(t, err, ErrTechnicianNotAvailable)
	mockJobRepo.AssertExpectations(t)
	mockUserRepo.AssertExpectations(t)
}

// TestBulkAssignTechnicians_Success tests bulk assignment
func TestBulkAssignTechnicians_Success(t *testing.T) {
	mockJobRepo := new(MockJobRepository)
	mockUserRepo := new(MockUserRepository)
	service := NewAssignmentService(mockJobRepo, mockUserRepo, nil)

	ctx := context.Background()
	jobID := uuid.New()
	tech1ID := uuid.New()
	tech2ID := uuid.New()

	job := &models.Job{
		ID:        jobID,
		JobNumber: "JOB-001",
	}

	tech1 := &models.User{
		ID:    tech1ID,
		Email: "tech1@example.com",
		Role:  models.UserRoleTechnician,
	}

	tech2 := &models.User{
		ID:    tech2ID,
		Email: "tech2@example.com",
		Role:  models.UserRoleTechnician,
	}

	req := &BulkAssignmentRequest{
		JobID: jobID,
		Assignments: []AssignmentRequest{
			{TechnicianID: tech1ID, Role: "Lead"},
			{TechnicianID: tech2ID, Role: "Assistant"},
		},
	}

	// Setup mocks for both technicians
	mockJobRepo.On("GetByID", jobID).Return(job, nil).Times(2)
	mockUserRepo.On("GetByID", tech1ID).Return(tech1, nil)
	mockUserRepo.On("GetByID", tech2ID).Return(tech2, nil)
	mockJobRepo.On("GetByAssignedUser", tech1ID).Return([]models.Job{}, nil)
	mockJobRepo.On("GetByAssignedUser", tech2ID).Return([]models.Job{}, nil)
	mockJobRepo.On("AddAssignment", mock.AnythingOfType("*models.JobAssignment")).Return(nil).Times(2)

	assignments, errs := service.BulkAssignTechnicians(ctx, req, uuid.New())

	assert.Len(t, assignments, 2)
	assert.Len(t, errs, 0)
	mockJobRepo.AssertExpectations(t)
	mockUserRepo.AssertExpectations(t)
}

// TestCheckTechnicianAvailability_Available tests availability check
func TestCheckTechnicianAvailability_Available(t *testing.T) {
	mockJobRepo := new(MockJobRepository)
	mockUserRepo := new(MockUserRepository)
	service := NewAssignmentService(mockJobRepo, mockUserRepo, nil)

	ctx := context.Background()
	technicianID := uuid.New()
	startTime := time.Now().Add(24 * time.Hour)
	endTime := startTime.Add(2 * time.Hour)

	mockJobRepo.On("GetByAssignedUser", technicianID).Return([]models.Job{}, nil)

	availability, err := service.CheckTechnicianAvailability(ctx, technicianID, startTime, endTime)

	require.NoError(t, err)
	assert.NotNil(t, availability)
	assert.True(t, availability.Available)
	assert.Len(t, availability.Conflicts, 0)
	assert.Equal(t, 0, availability.Workload)
	mockJobRepo.AssertExpectations(t)
}

// TestCheckTechnicianAvailability_NotAvailable tests unavailable technician
func TestCheckTechnicianAvailability_NotAvailable(t *testing.T) {
	mockJobRepo := new(MockJobRepository)
	mockUserRepo := new(MockUserRepository)
	service := NewAssignmentService(mockJobRepo, mockUserRepo, nil)

	ctx := context.Background()
	technicianID := uuid.New()
	startTime := time.Now().Add(24 * time.Hour)
	endTime := startTime.Add(2 * time.Hour)

	conflictJob := models.Job{
		ID:               uuid.New(),
		JobNumber:        "JOB-CONFLICT",
		Title:            "Conflicting Job",
		Status:           models.JobStatusScheduled,
		ScheduledStartAt: &startTime,
		ScheduledEndAt:   &endTime,
	}

	mockJobRepo.On("GetByAssignedUser", technicianID).Return([]models.Job{conflictJob}, nil)

	availability, err := service.CheckTechnicianAvailability(ctx, technicianID, startTime, endTime)

	require.NoError(t, err)
	assert.NotNil(t, availability)
	assert.False(t, availability.Available)
	assert.Len(t, availability.Conflicts, 1)
	mockJobRepo.AssertExpectations(t)
}

// TestCheckTechnicianAvailability_InvalidTimeSlot tests invalid time slot
func TestCheckTechnicianAvailability_InvalidTimeSlot(t *testing.T) {
	mockJobRepo := new(MockJobRepository)
	mockUserRepo := new(MockUserRepository)
	service := NewAssignmentService(mockJobRepo, mockUserRepo, nil)

	ctx := context.Background()
	technicianID := uuid.New()
	startTime := time.Now().Add(24 * time.Hour)
	endTime := startTime.Add(-2 * time.Hour) // End before start!

	availability, err := service.CheckTechnicianAvailability(ctx, technicianID, startTime, endTime)

	assert.Error(t, err)
	assert.Nil(t, availability)
	assert.ErrorIs(t, err, ErrInvalidTimeSlot)
}

// TestDetectConflicts tests conflict detection
func TestDetectConflicts(t *testing.T) {
	mockJobRepo := new(MockJobRepository)
	mockUserRepo := new(MockUserRepository)
	service := NewAssignmentService(mockJobRepo, mockUserRepo, nil)

	ctx := context.Background()
	technicianID := uuid.New()

	now := time.Now()
	startTime := now.Add(24 * time.Hour)
	endTime := startTime.Add(2 * time.Hour)

	conflictJob1 := models.Job{
		ID:               uuid.New(),
		Status:           models.JobStatusScheduled,
		ScheduledStartAt: &startTime,
		ScheduledEndAt:   &endTime,
	}

	completedJob := models.Job{
		ID:               uuid.New(),
		Status:           models.JobStatusCompleted, // Should not be a conflict
		ScheduledStartAt: &startTime,
		ScheduledEndAt:   &endTime,
	}

	futureJob := models.Job{
		ID:     uuid.New(),
		Status: models.JobStatusScheduled,
		ScheduledStartAt: func() *time.Time {
			t := startTime.Add(10 * time.Hour)
			return &t
		}(),
		ScheduledEndAt: func() *time.Time {
			t := startTime.Add(12 * time.Hour)
			return &t
		}(),
	}

	mockJobRepo.On("GetByAssignedUser", technicianID).Return([]models.Job{
		conflictJob1,
		completedJob,
		futureJob,
	}, nil)

	conflicts, err := service.DetectConflicts(ctx, technicianID, startTime, endTime, nil)

	require.NoError(t, err)
	assert.Len(t, conflicts, 1) // Only conflictJob1 should be detected
	assert.Equal(t, conflictJob1.ID, conflicts[0].ID)
	mockJobRepo.AssertExpectations(t)
}

// TestCheckSkillMatch tests skill matching
func TestCheckSkillMatch(t *testing.T) {
	mockJobRepo := new(MockJobRepository)
	mockUserRepo := new(MockUserRepository)
	service := NewAssignmentService(mockJobRepo, mockUserRepo, nil)

	ctx := context.Background()
	technicianID := uuid.New()

	tests := []struct {
		name           string
		requiredSkills []TechnicianSkill
		expectedMatch  float64
	}{
		{
			name:           "Full match",
			requiredSkills: []TechnicianSkill{SkillHVACInstallation, SkillHVACRepair},
			expectedMatch:  100.0,
		},
		{
			name:           "Partial match",
			requiredSkills: []TechnicianSkill{SkillHVACInstallation, SkillPlumbing},
			expectedMatch:  50.0,
		},
		{
			name:           "No match",
			requiredSkills: []TechnicianSkill{SkillPlumbing, SkillElectrical},
			expectedMatch:  0.0,
		},
		{
			name:           "No requirements",
			requiredSkills: []TechnicianSkill{},
			expectedMatch:  100.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			match, err := service.CheckSkillMatch(ctx, technicianID, tt.requiredSkills)
			require.NoError(t, err)
			assert.Equal(t, tt.expectedMatch, match)
		})
	}
}

// TestGetTechnicianWorkload tests workload calculation
func TestGetTechnicianWorkload(t *testing.T) {
	mockJobRepo := new(MockJobRepository)
	mockUserRepo := new(MockUserRepository)
	service := NewAssignmentService(mockJobRepo, mockUserRepo, nil)

	ctx := context.Background()
	technicianID := uuid.New()
	startDate := time.Now()
	endDate := startDate.Add(7 * 24 * time.Hour)

	jobs := []models.Job{
		{
			ID:     uuid.New(),
			Status: models.JobStatusScheduled,
			ScheduledStartAt: func() *time.Time {
				t := startDate.Add(24 * time.Hour)
				return &t
			}(),
		},
		{
			ID:     uuid.New(),
			Status: models.JobStatusInProgress,
			ScheduledStartAt: func() *time.Time {
				t := startDate.Add(48 * time.Hour)
				return &t
			}(),
		},
		{
			ID:     uuid.New(),
			Status: models.JobStatusCompleted, // Should not count
			ScheduledStartAt: func() *time.Time {
				t := startDate.Add(72 * time.Hour)
				return &t
			}(),
		},
	}

	mockJobRepo.On("GetByAssignedUser", technicianID).Return(jobs, nil)

	workload, err := service.GetTechnicianWorkload(ctx, technicianID, startDate, endDate)

	require.NoError(t, err)
	assert.Equal(t, 2, workload) // Only active jobs
	mockJobRepo.AssertExpectations(t)
}
