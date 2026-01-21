package validators

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	"github.com/javaknight1/servicepro/backend/internal/models"
)

func TestValidateCreateRequest(t *testing.T) {
	validator := NewJobValidator()

	t.Run("Valid create request", func(t *testing.T) {
		req := &models.CreateJobRequest{
			CustomerID:  uuid.New(),
			Title:       "AC Installation",
			Description: "Install new AC unit",
			JobType:     models.JobTypeInstallation,
			Priority:    models.JobPriorityNormal,
			ServiceAddress: models.ServiceAddress{
				Street: "123 Main St",
				City:   "Austin",
				State:  "TX",
				Zip:    "78701",
			},
		}

		err := validator.ValidateCreateRequest(req)
		assert.NoError(t, err)
	})

	t.Run("Missing customer ID", func(t *testing.T) {
		req := &models.CreateJobRequest{
			CustomerID: uuid.Nil,
			Title:      "AC Installation",
			JobType:    models.JobTypeInstallation,
		}

		err := validator.ValidateCreateRequest(req)
		assert.ErrorIs(t, err, ErrMissingCustomerID)
	})

	t.Run("Missing title", func(t *testing.T) {
		req := &models.CreateJobRequest{
			CustomerID: uuid.New(),
			Title:      "",
			JobType:    models.JobTypeInstallation,
		}

		err := validator.ValidateCreateRequest(req)
		assert.ErrorIs(t, err, ErrMissingTitle)
	})

	t.Run("Invalid job type", func(t *testing.T) {
		req := &models.CreateJobRequest{
			CustomerID: uuid.New(),
			Title:      "Test Job",
			JobType:    "invalid_type",
		}

		err := validator.ValidateCreateRequest(req)
		assert.ErrorIs(t, err, ErrInvalidJobType)
	})

	t.Run("Invalid priority", func(t *testing.T) {
		req := &models.CreateJobRequest{
			CustomerID: uuid.New(),
			Title:      "Test Job",
			JobType:    models.JobTypeRepair,
			Priority:   "invalid_priority",
		}

		err := validator.ValidateCreateRequest(req)
		assert.ErrorIs(t, err, ErrInvalidJobPriority)
	})

	t.Run("Negative estimated duration", func(t *testing.T) {
		req := &models.CreateJobRequest{
			CustomerID:        uuid.New(),
			Title:             "Test Job",
			JobType:           models.JobTypeRepair,
			EstimatedDuration: -30,
		}

		err := validator.ValidateCreateRequest(req)
		assert.ErrorIs(t, err, ErrInvalidDuration)
	})

	t.Run("Negative estimated cost", func(t *testing.T) {
		req := &models.CreateJobRequest{
			CustomerID:    uuid.New(),
			Title:         "Test Job",
			JobType:       models.JobTypeRepair,
			EstimatedCost: -100.50,
		}

		err := validator.ValidateCreateRequest(req)
		assert.ErrorIs(t, err, ErrInvalidCost)
	})

	t.Run("Invalid service address - missing street", func(t *testing.T) {
		req := &models.CreateJobRequest{
			CustomerID: uuid.New(),
			Title:      "Test Job",
			JobType:    models.JobTypeRepair,
			ServiceAddress: models.ServiceAddress{
				City:  "Austin",
				State: "TX",
			},
		}

		err := validator.ValidateCreateRequest(req)
		assert.ErrorIs(t, err, ErrInvalidServiceAddress)
	})

	t.Run("Invalid service address - invalid state code", func(t *testing.T) {
		req := &models.CreateJobRequest{
			CustomerID: uuid.New(),
			Title:      "Test Job",
			JobType:    models.JobTypeRepair,
			ServiceAddress: models.ServiceAddress{
				Street: "123 Main St",
				City:   "Austin",
				State:  "TEXAS", // Should be 2 letters
			},
		}

		err := validator.ValidateCreateRequest(req)
		assert.ErrorIs(t, err, ErrInvalidServiceAddress)
	})
}

func TestValidateUpdateRequest(t *testing.T) {
	validator := NewJobValidator()

	currentJob := &models.Job{
		Status:   models.JobStatusScheduled,
		JobType:  models.JobTypeInstallation,
		Priority: models.JobPriorityNormal,
	}

	t.Run("Valid update request", func(t *testing.T) {
		newStatus := models.JobStatusInProgress
		req := &models.UpdateJobRequest{
			Status: &newStatus,
		}

		err := validator.ValidateUpdateRequest(req, currentJob)
		assert.NoError(t, err)
	})

	t.Run("Invalid status transition", func(t *testing.T) {
		newStatus := models.JobStatusCompleted
		req := &models.UpdateJobRequest{
			Status: &newStatus,
		}

		err := validator.ValidateUpdateRequest(req, currentJob)
		assert.ErrorIs(t, err, ErrInvalidStatusTransition)
	})

	t.Run("Invalid job type in update", func(t *testing.T) {
		invalidType := models.JobType("invalid")
		req := &models.UpdateJobRequest{
			JobType: &invalidType,
		}

		err := validator.ValidateUpdateRequest(req, currentJob)
		assert.ErrorIs(t, err, ErrInvalidJobType)
	})

	t.Run("Negative actual cost", func(t *testing.T) {
		cost := -50.0
		req := &models.UpdateJobRequest{
			ActualCost: &cost,
		}

		err := validator.ValidateUpdateRequest(req, currentJob)
		assert.ErrorIs(t, err, ErrInvalidCost)
	})

	t.Run("Negative tax amount", func(t *testing.T) {
		tax := -10.0
		req := &models.UpdateJobRequest{
			TaxAmount: &tax,
		}

		err := validator.ValidateUpdateRequest(req, currentJob)
		assert.ErrorIs(t, err, ErrInvalidCost)
	})
}

func TestValidateJobType(t *testing.T) {
	validator := NewJobValidator()

	tests := []struct {
		name     string
		jobType  models.JobType
		hasError bool
	}{
		{"Installation", models.JobTypeInstallation, false},
		{"Maintenance", models.JobTypeMaintenance, false},
		{"Repair", models.JobTypeRepair, false},
		{"Inspection", models.JobTypeInspection, false},
		{"Emergency", models.JobTypeEmergency, false},
		{"Invalid type", models.JobType("invalid"), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.ValidateJobType(tt.jobType)
			if tt.hasError {
				assert.ErrorIs(t, err, ErrInvalidJobType)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidatePriority(t *testing.T) {
	validator := NewJobValidator()

	tests := []struct {
		name     string
		priority models.JobPriority
		hasError bool
	}{
		{"Low", models.JobPriorityLow, false},
		{"Normal", models.JobPriorityNormal, false},
		{"High", models.JobPriorityHigh, false},
		{"Urgent", models.JobPriorityUrgent, false},
		{"Invalid priority", models.JobPriority("invalid"), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.ValidatePriority(tt.priority)
			if tt.hasError {
				assert.ErrorIs(t, err, ErrInvalidJobPriority)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateStatus(t *testing.T) {
	validator := NewJobValidator()

	tests := []struct {
		name     string
		status   models.JobStatus
		hasError bool
	}{
		{"Scheduled", models.JobStatusScheduled, false},
		{"In Progress", models.JobStatusInProgress, false},
		{"On Hold", models.JobStatusOnHold, false},
		{"Completed", models.JobStatusCompleted, false},
		{"Cancelled", models.JobStatusCancelled, false},
		{"Invalid status", models.JobStatus("invalid"), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.ValidateStatus(tt.status)
			if tt.hasError {
				assert.ErrorIs(t, err, ErrInvalidJobStatus)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateSchedule(t *testing.T) {
	validator := NewJobValidator()
	now := time.Now()
	future := now.Add(24 * time.Hour)
	past := now.Add(-24 * time.Hour)

	t.Run("Both nil is valid", func(t *testing.T) {
		err := validator.ValidateSchedule(nil, nil)
		assert.NoError(t, err)
	})

	t.Run("Valid schedule - future dates", func(t *testing.T) {
		start := future
		end := future.Add(2 * time.Hour)
		err := validator.ValidateSchedule(&start, &end)
		assert.NoError(t, err)
	})

	t.Run("Valid schedule - past dates (allowed for historical jobs)", func(t *testing.T) {
		start := past
		end := past.Add(2 * time.Hour)
		err := validator.ValidateSchedule(&start, &end)
		assert.NoError(t, err)
	})

	t.Run("Invalid schedule - end before start", func(t *testing.T) {
		start := future
		end := future.Add(-1 * time.Hour)
		err := validator.ValidateSchedule(&start, &end)
		assert.ErrorIs(t, err, ErrInvalidSchedule)
	})

	t.Run("Only start date is valid", func(t *testing.T) {
		start := future
		err := validator.ValidateSchedule(&start, nil)
		assert.NoError(t, err)
	})

	t.Run("Only end date is valid", func(t *testing.T) {
		end := future
		err := validator.ValidateSchedule(nil, &end)
		assert.NoError(t, err)
	})
}

func TestValidateStatusTransition(t *testing.T) {
	validator := NewJobValidator()

	tests := []struct {
		name     string
		from     models.JobStatus
		to       models.JobStatus
		hasError bool
	}{
		// Same status transitions (always allowed)
		{"Same status - scheduled", models.JobStatusScheduled, models.JobStatusScheduled, false},

		// Valid transitions from Scheduled
		{"Scheduled to In Progress", models.JobStatusScheduled, models.JobStatusInProgress, false},
		{"Scheduled to On Hold", models.JobStatusScheduled, models.JobStatusOnHold, false},
		{"Scheduled to Cancelled", models.JobStatusScheduled, models.JobStatusCancelled, false},
		{"Scheduled to Completed - invalid", models.JobStatusScheduled, models.JobStatusCompleted, true},

		// Valid transitions from In Progress
		{"In Progress to On Hold", models.JobStatusInProgress, models.JobStatusOnHold, false},
		{"In Progress to Completed", models.JobStatusInProgress, models.JobStatusCompleted, false},
		{"In Progress to Cancelled", models.JobStatusInProgress, models.JobStatusCancelled, false},
		{"In Progress to Scheduled - invalid", models.JobStatusInProgress, models.JobStatusScheduled, true},

		// Valid transitions from On Hold
		{"On Hold to Scheduled", models.JobStatusOnHold, models.JobStatusScheduled, false},
		{"On Hold to In Progress", models.JobStatusOnHold, models.JobStatusInProgress, false},
		{"On Hold to Cancelled", models.JobStatusOnHold, models.JobStatusCancelled, false},
		{"On Hold to Completed - invalid", models.JobStatusOnHold, models.JobStatusCompleted, true},

		// Valid transitions from Completed
		{"Completed to In Progress (reopen)", models.JobStatusCompleted, models.JobStatusInProgress, false},
		{"Completed to Scheduled - invalid", models.JobStatusCompleted, models.JobStatusScheduled, true},
		{"Completed to On Hold - invalid", models.JobStatusCompleted, models.JobStatusOnHold, true},

		// Valid transitions from Cancelled
		{"Cancelled to Scheduled (reschedule)", models.JobStatusCancelled, models.JobStatusScheduled, false},
		{"Cancelled to In Progress - invalid", models.JobStatusCancelled, models.JobStatusInProgress, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.ValidateStatusTransition(tt.from, tt.to)
			if tt.hasError {
				assert.ErrorIs(t, err, ErrInvalidStatusTransition)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateServiceAddress(t *testing.T) {
	validator := NewJobValidator()

	t.Run("Nil address is valid", func(t *testing.T) {
		err := validator.ValidateServiceAddress(nil)
		assert.NoError(t, err)
	})

	t.Run("Empty address is valid", func(t *testing.T) {
		addr := &models.ServiceAddress{}
		err := validator.ValidateServiceAddress(addr)
		assert.NoError(t, err)
	})

	t.Run("Valid complete address", func(t *testing.T) {
		addr := &models.ServiceAddress{
			Street: "123 Main St",
			City:   "Austin",
			State:  "TX",
			Zip:    "78701",
		}
		err := validator.ValidateServiceAddress(addr)
		assert.NoError(t, err)
	})

	t.Run("Missing street", func(t *testing.T) {
		addr := &models.ServiceAddress{
			City:  "Austin",
			State: "TX",
		}
		err := validator.ValidateServiceAddress(addr)
		assert.ErrorIs(t, err, ErrInvalidServiceAddress)
	})

	t.Run("Missing city", func(t *testing.T) {
		addr := &models.ServiceAddress{
			Street: "123 Main St",
			State:  "TX",
		}
		err := validator.ValidateServiceAddress(addr)
		assert.ErrorIs(t, err, ErrInvalidServiceAddress)
	})

	t.Run("Missing state", func(t *testing.T) {
		addr := &models.ServiceAddress{
			Street: "123 Main St",
			City:   "Austin",
		}
		err := validator.ValidateServiceAddress(addr)
		assert.ErrorIs(t, err, ErrInvalidServiceAddress)
	})

	t.Run("Invalid state code - too long", func(t *testing.T) {
		addr := &models.ServiceAddress{
			Street: "123 Main St",
			City:   "Austin",
			State:  "TEXAS",
		}
		err := validator.ValidateServiceAddress(addr)
		assert.ErrorIs(t, err, ErrInvalidServiceAddress)
	})

	t.Run("Invalid state code - too short", func(t *testing.T) {
		addr := &models.ServiceAddress{
			Street: "123 Main St",
			City:   "Austin",
			State:  "T",
		}
		err := validator.ValidateServiceAddress(addr)
		assert.ErrorIs(t, err, ErrInvalidServiceAddress)
	})
}

func TestValidateMaterial(t *testing.T) {
	validator := NewJobValidator()

	t.Run("Valid material", func(t *testing.T) {
		material := &models.JobMaterial{
			JobID:    uuid.New(),
			Name:     "AC Filter",
			Quantity: 2,
			Unit:     "each",
			UnitCost: 25.50,
		}
		err := validator.ValidateMaterial(material)
		assert.NoError(t, err)
	})

	t.Run("Missing name", func(t *testing.T) {
		material := &models.JobMaterial{
			JobID:    uuid.New(),
			Name:     "",
			Quantity: 2,
			Unit:     "each",
			UnitCost: 25.50,
		}
		err := validator.ValidateMaterial(material)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "name is required")
	})

	t.Run("Zero quantity", func(t *testing.T) {
		material := &models.JobMaterial{
			JobID:    uuid.New(),
			Name:     "AC Filter",
			Quantity: 0,
			Unit:     "each",
			UnitCost: 25.50,
		}
		err := validator.ValidateMaterial(material)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "quantity must be positive")
	})

	t.Run("Negative quantity", func(t *testing.T) {
		material := &models.JobMaterial{
			JobID:    uuid.New(),
			Name:     "AC Filter",
			Quantity: -2,
			Unit:     "each",
			UnitCost: 25.50,
		}
		err := validator.ValidateMaterial(material)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "quantity must be positive")
	})

	t.Run("Negative unit cost", func(t *testing.T) {
		material := &models.JobMaterial{
			JobID:    uuid.New(),
			Name:     "AC Filter",
			Quantity: 2,
			Unit:     "each",
			UnitCost: -25.50,
		}
		err := validator.ValidateMaterial(material)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "unit cost must be non-negative")
	})

	t.Run("Missing unit", func(t *testing.T) {
		material := &models.JobMaterial{
			JobID:    uuid.New(),
			Name:     "AC Filter",
			Quantity: 2,
			Unit:     "",
			UnitCost: 25.50,
		}
		err := validator.ValidateMaterial(material)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "unit is required")
	})
}

func TestValidateAssignment(t *testing.T) {
	validator := NewJobValidator()

	t.Run("Valid assignment", func(t *testing.T) {
		assignment := &models.JobAssignment{
			JobID:       uuid.New(),
			UserID:      uuid.New(),
			Role:        "lead_technician",
			HoursWorked: 4.5,
		}
		err := validator.ValidateAssignment(assignment)
		assert.NoError(t, err)
	})

	t.Run("Missing job ID", func(t *testing.T) {
		assignment := &models.JobAssignment{
			JobID:       uuid.Nil,
			UserID:      uuid.New(),
			HoursWorked: 4.5,
		}
		err := validator.ValidateAssignment(assignment)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "job ID is required")
	})

	t.Run("Missing user ID", func(t *testing.T) {
		assignment := &models.JobAssignment{
			JobID:       uuid.New(),
			UserID:      uuid.Nil,
			HoursWorked: 4.5,
		}
		err := validator.ValidateAssignment(assignment)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "user ID is required")
	})

	t.Run("Negative hours worked", func(t *testing.T) {
		assignment := &models.JobAssignment{
			JobID:       uuid.New(),
			UserID:      uuid.New(),
			HoursWorked: -2.0,
		}
		err := validator.ValidateAssignment(assignment)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "hours worked must be non-negative")
	})
}

func TestValidateNote(t *testing.T) {
	validator := NewJobValidator()

	t.Run("Valid note", func(t *testing.T) {
		note := &models.JobNote{
			JobID:  uuid.New(),
			UserID: uuid.New(),
			Note:   "Customer requested morning appointment",
		}
		err := validator.ValidateNote(note)
		assert.NoError(t, err)
	})

	t.Run("Missing job ID", func(t *testing.T) {
		note := &models.JobNote{
			JobID:  uuid.Nil,
			UserID: uuid.New(),
			Note:   "Test note",
		}
		err := validator.ValidateNote(note)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "job ID is required")
	})

	t.Run("Missing user ID", func(t *testing.T) {
		note := &models.JobNote{
			JobID:  uuid.New(),
			UserID: uuid.Nil,
			Note:   "Test note",
		}
		err := validator.ValidateNote(note)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "user ID is required")
	})

	t.Run("Empty note content", func(t *testing.T) {
		note := &models.JobNote{
			JobID:  uuid.New(),
			UserID: uuid.New(),
			Note:   "",
		}
		err := validator.ValidateNote(note)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "note content is required")
	})

	t.Run("Note exceeds maximum length", func(t *testing.T) {
		longNote := make([]byte, 10001)
		for i := range longNote {
			longNote[i] = 'a'
		}
		note := &models.JobNote{
			JobID:  uuid.New(),
			UserID: uuid.New(),
			Note:   string(longNote),
		}
		err := validator.ValidateNote(note)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "exceeds maximum length")
	})
}

func TestCanUserModifyJob(t *testing.T) {
	validator := NewJobValidator()
	jobID := uuid.New()
	userID := uuid.New()
	otherUserID := uuid.New()

	job := &models.Job{
		ID:        jobID,
		CreatedBy: userID,
		Assignments: []models.JobAssignment{
			{
				JobID:  jobID,
				UserID: userID,
			},
		},
	}

	t.Run("Admin can modify any job", func(t *testing.T) {
		err := validator.CanUserModifyJob(job, otherUserID, models.UserRoleAdmin)
		assert.NoError(t, err)
	})

	t.Run("Manager can modify any job", func(t *testing.T) {
		err := validator.CanUserModifyJob(job, otherUserID, models.UserRoleManager)
		assert.NoError(t, err)
	})

	t.Run("Technician can modify job they created", func(t *testing.T) {
		err := validator.CanUserModifyJob(job, userID, models.UserRoleTechnician)
		assert.NoError(t, err)
	})

	t.Run("Technician can modify job they're assigned to", func(t *testing.T) {
		jobWithAssignment := &models.Job{
			ID:        jobID,
			CreatedBy: otherUserID,
			Assignments: []models.JobAssignment{
				{
					JobID:  jobID,
					UserID: userID,
				},
			},
		}
		err := validator.CanUserModifyJob(jobWithAssignment, userID, models.UserRoleTechnician)
		assert.NoError(t, err)
	})

	t.Run("Technician cannot modify job they didn't create and aren't assigned to", func(t *testing.T) {
		unrelatedJob := &models.Job{
			ID:          jobID,
			CreatedBy:   otherUserID,
			Assignments: []models.JobAssignment{},
		}
		err := validator.CanUserModifyJob(unrelatedJob, userID, models.UserRoleTechnician)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "insufficient permissions")
	})

	t.Run("Unknown role cannot modify job", func(t *testing.T) {
		err := validator.CanUserModifyJob(job, userID, "unknown_role")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "insufficient permissions")
	})
}

func TestValidateBusinessRules(t *testing.T) {
	validator := NewJobValidator()

	t.Run("Emergency job without urgent priority", func(t *testing.T) {
		job := &models.Job{
			JobType:  models.JobTypeEmergency,
			Priority: models.JobPriorityNormal,
			Status:   models.JobStatusScheduled,
		}
		warnings := validator.ValidateBusinessRules(job)
		assert.Len(t, warnings, 1)
		assert.Contains(t, warnings[0], "Emergency jobs should typically have urgent priority")
	})

	t.Run("Job scheduled far in future", func(t *testing.T) {
		futureDate := time.Now().Add(100 * 24 * time.Hour)
		job := &models.Job{
			JobType:          models.JobTypeInstallation,
			Priority:         models.JobPriorityNormal,
			Status:           models.JobStatusScheduled,
			ScheduledStartAt: &futureDate,
		}
		warnings := validator.ValidateBusinessRules(job)
		assert.Len(t, warnings, 1)
		assert.Contains(t, warnings[0], "days in the future")
	})

	t.Run("Estimated duration exceeds 8 hours", func(t *testing.T) {
		job := &models.Job{
			JobType:           models.JobTypeInstallation,
			Priority:          models.JobPriorityNormal,
			Status:            models.JobStatusScheduled,
			EstimatedDuration: 500, // > 8 hours (480 minutes)
		}
		warnings := validator.ValidateBusinessRules(job)
		assert.Len(t, warnings, 1)
		assert.Contains(t, warnings[0], "exceeds 8 hours")
	})

	t.Run("Actual cost significantly higher than estimate", func(t *testing.T) {
		job := &models.Job{
			JobType:       models.JobTypeRepair,
			Priority:      models.JobPriorityNormal,
			Status:        models.JobStatusScheduled,
			EstimatedCost: 100.0,
			ActualCost:    150.0, // 50% higher
		}
		warnings := validator.ValidateBusinessRules(job)
		assert.Len(t, warnings, 1)
		assert.Contains(t, warnings[0], "higher than estimate")
	})

	t.Run("Actual cost significantly lower than estimate", func(t *testing.T) {
		job := &models.Job{
			JobType:       models.JobTypeRepair,
			Priority:      models.JobPriorityNormal,
			Status:        models.JobStatusScheduled,
			EstimatedCost: 100.0,
			ActualCost:    50.0, // 50% lower
		}
		warnings := validator.ValidateBusinessRules(job)
		assert.Len(t, warnings, 1)
		assert.Contains(t, warnings[0], "lower than estimate")
	})

	t.Run("Job in progress with no assignments", func(t *testing.T) {
		job := &models.Job{
			JobType:     models.JobTypeRepair,
			Priority:    models.JobPriorityNormal,
			Status:      models.JobStatusInProgress,
			Assignments: []models.JobAssignment{},
		}
		warnings := validator.ValidateBusinessRules(job)
		assert.Len(t, warnings, 1)
		assert.Contains(t, warnings[0], "no technician assignments")
	})

	t.Run("Job with no warnings", func(t *testing.T) {
		job := &models.Job{
			JobType:       models.JobTypeInstallation,
			Priority:      models.JobPriorityNormal,
			Status:        models.JobStatusScheduled,
			EstimatedCost: 100.0,
			ActualCost:    105.0, // Within 25% threshold
		}
		warnings := validator.ValidateBusinessRules(job)
		assert.Len(t, warnings, 0)
	})

	t.Run("Job with multiple warnings", func(t *testing.T) {
		futureDate := time.Now().Add(100 * 24 * time.Hour)
		job := &models.Job{
			JobType:           models.JobTypeEmergency,
			Priority:          models.JobPriorityNormal, // Should be urgent
			Status:            models.JobStatusInProgress,
			ScheduledStartAt:  &futureDate,              // Too far in future
			EstimatedDuration: 500,                      // Too long
			Assignments:       []models.JobAssignment{}, // No assignments
		}
		warnings := validator.ValidateBusinessRules(job)
		assert.GreaterOrEqual(t, len(warnings), 2, "Should have multiple warnings")
	})
}

func TestValidateJobStart(t *testing.T) {
	validator := NewJobValidator()

	t.Run("Can start scheduled job", func(t *testing.T) {
		job := &models.Job{
			Status: models.JobStatusScheduled,
		}
		err := validator.ValidateJobStart(job)
		assert.NoError(t, err)
	})

	t.Run("Can start on-hold job", func(t *testing.T) {
		job := &models.Job{
			Status: models.JobStatusOnHold,
		}
		err := validator.ValidateJobStart(job)
		assert.NoError(t, err)
	})

	t.Run("Cannot start in-progress job", func(t *testing.T) {
		job := &models.Job{
			Status: models.JobStatusInProgress,
		}
		err := validator.ValidateJobStart(job)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "cannot start job")
	})

	t.Run("Cannot start completed job", func(t *testing.T) {
		job := &models.Job{
			Status: models.JobStatusCompleted,
		}
		err := validator.ValidateJobStart(job)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "cannot start job")
	})

	t.Run("Cannot start cancelled job", func(t *testing.T) {
		job := &models.Job{
			Status: models.JobStatusCancelled,
		}
		err := validator.ValidateJobStart(job)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "cannot start job")
	})
}

func TestValidateJobCompletion(t *testing.T) {
	validator := NewJobValidator()
	now := time.Now()

	t.Run("Can complete in-progress job with start time", func(t *testing.T) {
		job := &models.Job{
			Status:        models.JobStatusInProgress,
			ActualStartAt: &now,
		}
		err := validator.ValidateJobCompletion(job)
		assert.NoError(t, err)
	})

	t.Run("Cannot complete scheduled job", func(t *testing.T) {
		job := &models.Job{
			Status:        models.JobStatusScheduled,
			ActualStartAt: &now,
		}
		err := validator.ValidateJobCompletion(job)
		assert.ErrorIs(t, err, ErrJobNotInProgress)
	})

	t.Run("Cannot complete job without start time", func(t *testing.T) {
		job := &models.Job{
			Status:        models.JobStatusInProgress,
			ActualStartAt: nil,
		}
		err := validator.ValidateJobCompletion(job)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "actual start time")
	})
}
