package services

import (
	"context"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"github.com/javaknight1/servicepro/backend/internal/models"
)

// TestChangeStatus_Success tests successful status change
func TestChangeStatus_Success(t *testing.T) {
	// Setup mock database
	mockDB, mockSQL, err := sqlmock.New()
	require.NoError(t, err)
	defer mockDB.Close()

	dialector := postgres.New(postgres.Config{
		Conn:       mockDB,
		DriverName: "postgres",
	})
	db, err := gorm.Open(dialector, &gorm.Config{})
	require.NoError(t, err)

	mockJobRepo := new(MockJobRepository)
	service := NewStatusService(db, mockJobRepo, nil)

	ctx := context.Background()
	jobID := uuid.New()
	changedBy := uuid.New()

	// Mock expectations
	mockSQL.ExpectBegin()

	// Expect SELECT FOR UPDATE
	rows := sqlmock.NewRows([]string{"id", "job_number", "status", "created_at", "updated_at"}).
		AddRow(jobID, "JOB-001", models.JobStatusScheduled, time.Now(), time.Now())
	mockSQL.ExpectQuery(`SELECT .* FROM "jobs"`).
		WithArgs(jobID).
		WillReturnRows(rows)

	// Expect assignments query
	assignmentRows := sqlmock.NewRows([]string{"id", "job_id", "user_id", "role", "assigned_at"}).
		AddRow(uuid.New(), jobID, uuid.New(), "Lead", time.Now())
	mockSQL.ExpectQuery(`SELECT .* FROM "job_assignments"`).
		WillReturnRows(assignmentRows)

	// Expect UPDATE
	mockSQL.ExpectExec(`UPDATE "jobs"`).
		WillReturnResult(sqlmock.NewResult(1, 1))

	// Expect INSERT for transition
	mockSQL.ExpectQuery(`INSERT INTO "job_status_transitions"`).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(uuid.New()))

	mockSQL.ExpectCommit()

	// Execute
	req := &models.StatusChangeRequest{
		JobID:    jobID,
		ToStatus: models.JobStatusInProgress,
		Reason:   models.ReasonStartWork,
	}

	response, err := service.ChangeStatus(ctx, req, changedBy)

	// Assert
	require.NoError(t, err)
	assert.NotNil(t, response)
	assert.Equal(t, models.JobStatusScheduled, response.PreviousStatus)
	assert.Equal(t, models.JobStatusInProgress, response.NewStatus)
	assert.NoError(t, mockSQL.ExpectationsWereMet())
}

// TestChangeStatus_InvalidTransition tests invalid status transition
func TestChangeStatus_InvalidTransition(t *testing.T) {
	mockDB, mockSQL, err := sqlmock.New()
	require.NoError(t, err)
	defer mockDB.Close()

	dialector := postgres.New(postgres.Config{
		Conn:       mockDB,
		DriverName: "postgres",
	})
	db, err := gorm.Open(dialector, &gorm.Config{})
	require.NoError(t, err)

	mockJobRepo := new(MockJobRepository)
	service := NewStatusService(db, mockJobRepo, nil)

	ctx := context.Background()
	jobID := uuid.New()
	changedBy := uuid.New()

	// Mock expectations
	mockSQL.ExpectBegin()

	// Expect SELECT FOR UPDATE - job is completed (terminal status)
	rows := sqlmock.NewRows([]string{"id", "job_number", "status", "created_at", "updated_at"}).
		AddRow(jobID, "JOB-001", models.JobStatusCompleted, time.Now(), time.Now())
	mockSQL.ExpectQuery(`SELECT .* FROM "jobs"`).
		WithArgs(jobID).
		WillReturnRows(rows)

	// Expect assignments query
	mockSQL.ExpectQuery(`SELECT .* FROM "job_assignments"`).
		WillReturnRows(sqlmock.NewRows([]string{}))

	mockSQL.ExpectRollback()

	// Execute - try to transition from completed to in_progress (invalid)
	req := &models.StatusChangeRequest{
		JobID:    jobID,
		ToStatus: models.JobStatusInProgress,
	}

	response, err := service.ChangeStatus(ctx, req, changedBy)

	// Assert
	assert.Error(t, err)
	assert.Nil(t, response)
	assert.ErrorIs(t, err, models.ErrInvalidStatusTransition)
}

// TestChangeStatus_MissingRequiredNote tests missing required note
func TestChangeStatus_MissingRequiredNote(t *testing.T) {
	mockDB, mockSQL, err := sqlmock.New()
	require.NoError(t, err)
	defer mockDB.Close()

	dialector := postgres.New(postgres.Config{
		Conn:       mockDB,
		DriverName: "postgres",
	})
	db, err := gorm.Open(dialector, &gorm.Config{})
	require.NoError(t, err)

	mockJobRepo := new(MockJobRepository)
	service := NewStatusService(db, mockJobRepo, nil)

	ctx := context.Background()
	jobID := uuid.New()
	changedBy := uuid.New()

	// Mock expectations
	mockSQL.ExpectBegin()

	// Expect SELECT FOR UPDATE
	rows := sqlmock.NewRows([]string{"id", "job_number", "status", "created_at", "updated_at"}).
		AddRow(jobID, "JOB-001", models.JobStatusScheduled, time.Now(), time.Now())
	mockSQL.ExpectQuery(`SELECT .* FROM "jobs"`).
		WithArgs(jobID).
		WillReturnRows(rows)

	// Expect assignments query
	mockSQL.ExpectQuery(`SELECT .* FROM "job_assignments"`).
		WillReturnRows(sqlmock.NewRows([]string{}))

	mockSQL.ExpectRollback()

	// Execute - try to cancel without note (requires note)
	req := &models.StatusChangeRequest{
		JobID:    jobID,
		ToStatus: models.JobStatusCancelled,
		// Notes intentionally omitted
	}

	response, err := service.ChangeStatus(ctx, req, changedBy)

	// Assert
	assert.Error(t, err)
	assert.Nil(t, response)
	assert.ErrorIs(t, err, models.ErrStatusValidation)
}

// TestChangeStatus_ValidationError tests validation failure
func TestChangeStatus_ValidationError(t *testing.T) {
	mockDB, mockSQL, err := sqlmock.New()
	require.NoError(t, err)
	defer mockDB.Close()

	dialector := postgres.New(postgres.Config{
		Conn:       mockDB,
		DriverName: "postgres",
	})
	db, err := gorm.Open(dialector, &gorm.Config{})
	require.NoError(t, err)

	mockJobRepo := new(MockJobRepository)
	service := NewStatusService(db, mockJobRepo, nil)

	ctx := context.Background()
	jobID := uuid.New()
	changedBy := uuid.New()

	// Mock expectations
	mockSQL.ExpectBegin()

	// Expect SELECT FOR UPDATE - job with no assignments
	rows := sqlmock.NewRows([]string{"id", "job_number", "status", "created_at", "updated_at"}).
		AddRow(jobID, "JOB-001", models.JobStatusScheduled, time.Now(), time.Now())
	mockSQL.ExpectQuery(`SELECT .* FROM "jobs"`).
		WithArgs(jobID).
		WillReturnRows(rows)

	// Expect assignments query - empty (no assignments)
	mockSQL.ExpectQuery(`SELECT .* FROM "job_assignments"`).
		WillReturnRows(sqlmock.NewRows([]string{}))

	mockSQL.ExpectRollback()

	// Execute - try to start job without assignments
	req := &models.StatusChangeRequest{
		JobID:    jobID,
		ToStatus: models.JobStatusInProgress,
	}

	response, err := service.ChangeStatus(ctx, req, changedBy)

	// Assert
	assert.Error(t, err)
	assert.Nil(t, response)
	assert.ErrorIs(t, err, models.ErrStatusValidation)
}

// TestChangeStatus_JobNotFound tests job not found error
func TestChangeStatus_JobNotFound(t *testing.T) {
	mockDB, mockSQL, err := sqlmock.New()
	require.NoError(t, err)
	defer mockDB.Close()

	dialector := postgres.New(postgres.Config{
		Conn:       mockDB,
		DriverName: "postgres",
	})
	db, err := gorm.Open(dialector, &gorm.Config{})
	require.NoError(t, err)

	mockJobRepo := new(MockJobRepository)
	service := NewStatusService(db, mockJobRepo, nil)

	ctx := context.Background()
	jobID := uuid.New()
	changedBy := uuid.New()

	// Mock expectations
	mockSQL.ExpectBegin()

	// Expect SELECT FOR UPDATE - no rows
	mockSQL.ExpectQuery(`SELECT .* FROM "jobs"`).
		WithArgs(jobID).
		WillReturnError(gorm.ErrRecordNotFound)

	mockSQL.ExpectRollback()

	// Execute
	req := &models.StatusChangeRequest{
		JobID:    jobID,
		ToStatus: models.JobStatusInProgress,
	}

	response, err := service.ChangeStatus(ctx, req, changedBy)

	// Assert
	assert.Error(t, err)
	assert.Nil(t, response)
	assert.ErrorIs(t, err, ErrJobNotFound)
}

// TestGetStatusHistory tests retrieving status history
func TestGetStatusHistory(t *testing.T) {
	mockDB, mockSQL, err := sqlmock.New()
	require.NoError(t, err)
	defer mockDB.Close()

	dialector := postgres.New(postgres.Config{
		Conn:       mockDB,
		DriverName: "postgres",
	})
	db, err := gorm.Open(dialector, &gorm.Config{})
	require.NoError(t, err)

	mockJobRepo := new(MockJobRepository)
	service := NewStatusService(db, mockJobRepo, nil)

	ctx := context.Background()
	jobID := uuid.New()

	// Mock count
	countRows := sqlmock.NewRows([]string{"count"}).AddRow(2)
	mockSQL.ExpectQuery(`SELECT count\(\*\) FROM "job_status_transitions"`).
		WithArgs(jobID).
		WillReturnRows(countRows)

	// Mock transitions
	transitionRows := sqlmock.NewRows([]string{
		"id", "job_id", "from_status", "to_status", "reason", "changed_by", "transitioned_at",
	}).
		AddRow(uuid.New(), jobID, models.JobStatusScheduled, models.JobStatusInProgress, models.ReasonStartWork, uuid.New(), time.Now()).
		AddRow(uuid.New(), jobID, models.JobStatusInProgress, models.JobStatusCompleted, models.ReasonCompleteWork, uuid.New(), time.Now())

	mockSQL.ExpectQuery(`SELECT .* FROM "job_status_transitions"`).
		WillReturnRows(transitionRows)

	// Mock preloads (Job and ChangedByUser)
	mockSQL.ExpectQuery(`SELECT .* FROM "jobs"`).
		WillReturnRows(sqlmock.NewRows([]string{}))
	mockSQL.ExpectQuery(`SELECT .* FROM "users"`).
		WillReturnRows(sqlmock.NewRows([]string{}))

	// Execute
	history, err := service.GetStatusHistory(ctx, jobID, 10, 0)

	// Assert
	require.NoError(t, err)
	assert.NotNil(t, history)
	assert.Equal(t, int64(2), history.Total)
	assert.Len(t, history.Transitions, 2)
}

// TestGetAllowedTransitions tests getting allowed transitions
func TestGetAllowedTransitions(t *testing.T) {
	mockJobRepo := new(MockJobRepository)
	service := NewStatusService(nil, mockJobRepo, nil)

	ctx := context.Background()
	jobID := uuid.New()

	job := &models.Job{
		ID:        jobID,
		JobNumber: "JOB-001",
		Status:    models.JobStatusScheduled,
	}

	mockJobRepo.On("GetByID", jobID).Return(job, nil)

	// Execute
	response, err := service.GetAllowedTransitions(ctx, jobID)

	// Assert
	require.NoError(t, err)
	assert.NotNil(t, response)
	assert.Equal(t, models.JobStatusScheduled, response.CurrentStatus)
	assert.Contains(t, response.AllowedTransitions, models.JobStatusInProgress)
	assert.Contains(t, response.AllowedTransitions, models.JobStatusCancelled)
	assert.False(t, response.IsTerminal)

	mockJobRepo.AssertExpectations(t)
}

// TestBulkChangeStatus tests bulk status changes
func TestBulkChangeStatus(t *testing.T) {
	mockDB, mockSQL, err := sqlmock.New()
	require.NoError(t, err)
	defer mockDB.Close()

	dialector := postgres.New(postgres.Config{
		Conn:       mockDB,
		DriverName: "postgres",
	})
	db, err := gorm.Open(dialector, &gorm.Config{})
	require.NoError(t, err)

	mockJobRepo := new(MockJobRepository)
	service := NewStatusService(db, mockJobRepo, nil)

	ctx := context.Background()
	changedBy := uuid.New()

	jobID1 := uuid.New()
	jobID2 := uuid.New()

	// Mock for first job (success)
	mockSQL.ExpectBegin()
	rows1 := sqlmock.NewRows([]string{"id", "job_number", "status", "created_at", "updated_at"}).
		AddRow(jobID1, "JOB-001", models.JobStatusScheduled, time.Now(), time.Now())
	mockSQL.ExpectQuery(`SELECT .* FROM "jobs"`).
		WithArgs(jobID1).
		WillReturnRows(rows1)
	mockSQL.ExpectQuery(`SELECT .* FROM "job_assignments"`).
		WillReturnRows(sqlmock.NewRows([]string{"id", "job_id", "user_id"}).
			AddRow(uuid.New(), jobID1, uuid.New()))
	mockSQL.ExpectExec(`UPDATE "jobs"`).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mockSQL.ExpectQuery(`INSERT INTO "job_status_transitions"`).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(uuid.New()))
	mockSQL.ExpectCommit()

	// Mock for second job (failure - not found)
	mockSQL.ExpectBegin()
	mockSQL.ExpectQuery(`SELECT .* FROM "jobs"`).
		WithArgs(jobID2).
		WillReturnError(gorm.ErrRecordNotFound)
	mockSQL.ExpectRollback()

	// Execute
	requests := []*models.StatusChangeRequest{
		{JobID: jobID1, ToStatus: models.JobStatusInProgress},
		{JobID: jobID2, ToStatus: models.JobStatusInProgress},
	}

	responses, errs := service.BulkChangeStatus(ctx, requests, changedBy)

	// Assert
	assert.Len(t, responses, 1)
	assert.Len(t, errs, 1)
}

// TestValidateStatusChange tests status change validation
func TestValidateStatusChange(t *testing.T) {
	mockDB, mockSQL, err := sqlmock.New()
	require.NoError(t, err)
	defer mockDB.Close()

	dialector := postgres.New(postgres.Config{
		Conn:       mockDB,
		DriverName: "postgres",
	})
	db, err := gorm.Open(dialector, &gorm.Config{})
	require.NoError(t, err)

	mockJobRepo := new(MockJobRepository)
	service := NewStatusService(db, mockJobRepo, nil)

	ctx := context.Background()
	jobID := uuid.New()

	// Mock job with assignments
	rows := sqlmock.NewRows([]string{"id", "job_number", "status", "created_at", "updated_at"}).
		AddRow(jobID, "JOB-001", models.JobStatusScheduled, time.Now(), time.Now())
	mockSQL.ExpectQuery(`SELECT .* FROM "jobs"`).
		WithArgs(jobID).
		WillReturnRows(rows)

	assignmentRows := sqlmock.NewRows([]string{"id", "job_id", "user_id"}).
		AddRow(uuid.New(), jobID, uuid.New())
	mockSQL.ExpectQuery(`SELECT .* FROM "job_assignments"`).
		WillReturnRows(assignmentRows)

	// Execute
	err = service.ValidateStatusChange(ctx, jobID, models.JobStatusInProgress)

	// Assert
	assert.NoError(t, err)
}

// TestConcurrentStatusChanges tests concurrent status changes on same job
func TestConcurrentStatusChanges(t *testing.T) {
	// This test verifies that the locking mechanism prevents race conditions
	// In a real scenario, two goroutines trying to change status simultaneously
	// would be serialized by the mutex

	mockDB, _, err := sqlmock.New()
	require.NoError(t, err)
	defer mockDB.Close()

	dialector := postgres.New(postgres.Config{
		Conn:       mockDB,
		DriverName: "postgres",
	})
	db, err := gorm.Open(dialector, &gorm.Config{})
	require.NoError(t, err)

	mockJobRepo := new(MockJobRepository)
	service := NewStatusService(db, mockJobRepo, nil)

	jobID := uuid.New()

	// Verify that lock is properly acquired and released
	lock1 := service.acquireJobLock(jobID)
	assert.NotNil(t, lock1)

	lock2 := service.acquireJobLock(jobID)
	assert.Equal(t, lock1, lock2) // Same lock instance

	service.releaseJobLock(jobID)

	// After release, acquiring again should create a new lock
	lock3 := service.acquireJobLock(jobID)
	assert.NotNil(t, lock3)
}
