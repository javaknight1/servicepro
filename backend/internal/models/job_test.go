package models

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"
)

func TestJobMaterialBeforeSave(t *testing.T) {
	t.Run("Calculate total cost", func(t *testing.T) {
		material := &JobMaterial{
			JobID:    uuid.New(),
			Name:     "AC Filter",
			Quantity: 2.0,
			Unit:     "each",
			UnitCost: 25.50,
		}

		// Manually call the BeforeSave hook
		material.BeforeSave(&gorm.DB{})
		assert.Equal(t, 51.0, material.TotalCost) // 2 * 25.50
	})

	t.Run("Calculate total cost with decimals", func(t *testing.T) {
		material := &JobMaterial{
			JobID:    uuid.New(),
			Name:     "Refrigerant",
			Quantity: 2.5,
			Unit:     "lbs",
			UnitCost: 40.00,
		}

		material.BeforeSave(&gorm.DB{})
		assert.Equal(t, 100.0, material.TotalCost) // 2.5 * 40.00
	})

	t.Run("Calculate zero cost", func(t *testing.T) {
		material := &JobMaterial{
			JobID:    uuid.New(),
			Name:     "Free Item",
			Quantity: 10.0,
			Unit:     "each",
			UnitCost: 0.0,
		}

		material.BeforeSave(&gorm.DB{})
		assert.Equal(t, 0.0, material.TotalCost)
	})
}

func TestJobToResponse(t *testing.T) {
	jobID := uuid.New()
	customerID := uuid.New()
	userID := uuid.New()
	now := time.Now()

	t.Run("Convert job to response without relationships", func(t *testing.T) {
		job := &Job{
			ID:                jobID,
			JobNumber:         "JOB-20240122-0001",
			CustomerID:        customerID,
			Title:             "AC Installation",
			Description:       "Install new AC unit",
			JobType:           JobTypeInstallation,
			Status:            JobStatusScheduled,
			Priority:          JobPriorityNormal,
			EstimatedDuration: 120,
			EstimatedCost:     500.00,
			CreatedBy:         userID,
			CreatedAt:         now,
			UpdatedAt:         now,
		}

		response := job.ToResponse()
		assert.NotNil(t, response)
		assert.Equal(t, jobID, response.ID)
		assert.Equal(t, "JOB-20240122-0001", response.JobNumber)
		assert.Equal(t, customerID, response.CustomerID)
		assert.Equal(t, "AC Installation", response.Title)
		assert.Equal(t, JobTypeInstallation, response.JobType)
		assert.Equal(t, JobStatusScheduled, response.Status)
		assert.Nil(t, response.Customer)
		assert.Empty(t, response.Assignments)
		assert.Empty(t, response.Materials)
		assert.Empty(t, response.Notes)
	})

	t.Run("Convert job to response with customer", func(t *testing.T) {
		customer := &Customer{
			ID:        customerID,
			FirstName: "John",
			LastName:  "Doe",
			Email:     "john@example.com",
		}

		job := &Job{
			ID:         jobID,
			JobNumber:  "JOB-20240122-0001",
			CustomerID: customerID,
			Customer:   customer,
			Title:      "AC Installation",
			JobType:    JobTypeInstallation,
			Status:     JobStatusScheduled,
			Priority:   JobPriorityNormal,
			CreatedBy:  userID,
		}

		response := job.ToResponse()
		assert.NotNil(t, response.Customer)
		assert.Equal(t, customerID, response.Customer.ID)
		assert.Equal(t, "John", response.Customer.FirstName)
	})

	t.Run("Convert job to response with assignments", func(t *testing.T) {
		assignment := JobAssignment{
			ID:     uuid.New(),
			JobID:  jobID,
			UserID: userID,
			Role:   "lead_technician",
		}

		job := &Job{
			ID:          jobID,
			JobNumber:   "JOB-20240122-0001",
			CustomerID:  customerID,
			Title:       "AC Installation",
			JobType:     JobTypeInstallation,
			Status:      JobStatusScheduled,
			Priority:    JobPriorityNormal,
			CreatedBy:   userID,
			Assignments: []JobAssignment{assignment},
		}

		response := job.ToResponse()
		assert.Len(t, response.Assignments, 1)
		assert.Equal(t, "lead_technician", response.Assignments[0].Role)
	})

	t.Run("Convert job to response with materials", func(t *testing.T) {
		material := JobMaterial{
			ID:        uuid.New(),
			JobID:     jobID,
			Name:      "AC Filter",
			Quantity:  2,
			Unit:      "each",
			UnitCost:  25.50,
			TotalCost: 51.00,
		}

		job := &Job{
			ID:         jobID,
			JobNumber:  "JOB-20240122-0001",
			CustomerID: customerID,
			Title:      "AC Installation",
			JobType:    JobTypeInstallation,
			Status:     JobStatusScheduled,
			Priority:   JobPriorityNormal,
			CreatedBy:  userID,
			Materials:  []JobMaterial{material},
		}

		response := job.ToResponse()
		assert.Len(t, response.Materials, 1)
		assert.Equal(t, "AC Filter", response.Materials[0].Name)
		assert.Equal(t, 51.00, response.Materials[0].TotalCost)
	})

	t.Run("Convert job to response with notes", func(t *testing.T) {
		note := JobNote{
			ID:     uuid.New(),
			JobID:  jobID,
			UserID: userID,
			Note:   "Customer prefers morning appointments",
		}

		job := &Job{
			ID:         jobID,
			JobNumber:  "JOB-20240122-0001",
			CustomerID: customerID,
			Title:      "AC Installation",
			JobType:    JobTypeInstallation,
			Status:     JobStatusScheduled,
			Priority:   JobPriorityNormal,
			CreatedBy:  userID,
			Notes:      []JobNote{note},
		}

		response := job.ToResponse()
		assert.Len(t, response.Notes, 1)
		assert.Equal(t, "Customer prefers morning appointments", response.Notes[0].Note)
	})

	t.Run("Convert job to response with all relationships", func(t *testing.T) {
		customer := &Customer{
			ID:        customerID,
			FirstName: "John",
			LastName:  "Doe",
			Email:     "john@example.com",
		}

		assignment := JobAssignment{
			ID:     uuid.New(),
			JobID:  jobID,
			UserID: userID,
			Role:   "lead_technician",
		}

		material := JobMaterial{
			ID:        uuid.New(),
			JobID:     jobID,
			Name:      "AC Filter",
			Quantity:  2,
			Unit:      "each",
			UnitCost:  25.50,
			TotalCost: 51.00,
		}

		note := JobNote{
			ID:     uuid.New(),
			JobID:  jobID,
			UserID: userID,
			Note:   "Test note",
		}

		job := &Job{
			ID:          jobID,
			JobNumber:   "JOB-20240122-0001",
			CustomerID:  customerID,
			Customer:    customer,
			Title:       "AC Installation",
			JobType:     JobTypeInstallation,
			Status:      JobStatusScheduled,
			Priority:    JobPriorityNormal,
			CreatedBy:   userID,
			Assignments: []JobAssignment{assignment},
			Materials:   []JobMaterial{material},
			Notes:       []JobNote{note},
		}

		response := job.ToResponse()
		assert.NotNil(t, response.Customer)
		assert.Len(t, response.Assignments, 1)
		assert.Len(t, response.Materials, 1)
		assert.Len(t, response.Notes, 1)
	})
}

func TestJobStatus(t *testing.T) {
	t.Run("All job statuses are defined", func(t *testing.T) {
		statuses := []JobStatus{
			JobStatusScheduled,
			JobStatusInProgress,
			JobStatusOnHold,
			JobStatusCompleted,
			JobStatusCancelled,
		}

		assert.Len(t, statuses, 5)
		assert.Equal(t, JobStatus("scheduled"), JobStatusScheduled)
		assert.Equal(t, JobStatus("in_progress"), JobStatusInProgress)
		assert.Equal(t, JobStatus("on_hold"), JobStatusOnHold)
		assert.Equal(t, JobStatus("completed"), JobStatusCompleted)
		assert.Equal(t, JobStatus("cancelled"), JobStatusCancelled)
	})
}

func TestJobPriority(t *testing.T) {
	t.Run("All job priorities are defined", func(t *testing.T) {
		priorities := []JobPriority{
			JobPriorityLow,
			JobPriorityNormal,
			JobPriorityHigh,
			JobPriorityUrgent,
		}

		assert.Len(t, priorities, 4)
		assert.Equal(t, JobPriority("low"), JobPriorityLow)
		assert.Equal(t, JobPriority("normal"), JobPriorityNormal)
		assert.Equal(t, JobPriority("high"), JobPriorityHigh)
		assert.Equal(t, JobPriority("urgent"), JobPriorityUrgent)
	})
}

func TestJobType(t *testing.T) {
	t.Run("All job types are defined", func(t *testing.T) {
		types := []JobType{
			JobTypeInstallation,
			JobTypeMaintenance,
			JobTypeRepair,
			JobTypeInspection,
			JobTypeEmergency,
		}

		assert.Len(t, types, 5)
		assert.Equal(t, JobType("installation"), JobTypeInstallation)
		assert.Equal(t, JobType("maintenance"), JobTypeMaintenance)
		assert.Equal(t, JobType("repair"), JobTypeRepair)
		assert.Equal(t, JobType("inspection"), JobTypeInspection)
		assert.Equal(t, JobType("emergency"), JobTypeEmergency)
	})
}

func TestServiceAddress(t *testing.T) {
	t.Run("Create service address", func(t *testing.T) {
		addr := ServiceAddress{
			Street: "123 Main St",
			City:   "Austin",
			State:  "TX",
			Zip:    "78701",
		}

		assert.Equal(t, "123 Main St", addr.Street)
		assert.Equal(t, "Austin", addr.City)
		assert.Equal(t, "TX", addr.State)
		assert.Equal(t, "78701", addr.Zip)
	})

	t.Run("Empty service address", func(t *testing.T) {
		addr := ServiceAddress{}

		assert.Empty(t, addr.Street)
		assert.Empty(t, addr.City)
		assert.Empty(t, addr.State)
		assert.Empty(t, addr.Zip)
	})
}

func TestCreateJobRequest(t *testing.T) {
	t.Run("Create valid job request", func(t *testing.T) {
		req := CreateJobRequest{
			CustomerID:  uuid.New(),
			Title:       "AC Installation",
			Description: "Install new AC unit",
			JobType:     JobTypeInstallation,
			Priority:    JobPriorityNormal,
			ServiceAddress: ServiceAddress{
				Street: "123 Main St",
				City:   "Austin",
				State:  "TX",
				Zip:    "78701",
			},
			EstimatedDuration: 120,
			EstimatedCost:     500.00,
		}

		assert.NotEqual(t, uuid.Nil, req.CustomerID)
		assert.Equal(t, "AC Installation", req.Title)
		assert.Equal(t, JobTypeInstallation, req.JobType)
	})
}

func TestUpdateJobRequest(t *testing.T) {
	t.Run("Create update request with pointers", func(t *testing.T) {
		title := "Updated Title"
		status := JobStatusInProgress
		cost := 550.00

		req := UpdateJobRequest{
			Title:      &title,
			Status:     &status,
			ActualCost: &cost,
		}

		assert.NotNil(t, req.Title)
		assert.Equal(t, "Updated Title", *req.Title)
		assert.NotNil(t, req.Status)
		assert.Equal(t, JobStatusInProgress, *req.Status)
		assert.NotNil(t, req.ActualCost)
		assert.Equal(t, 550.00, *req.ActualCost)
	})

	t.Run("Empty update request", func(t *testing.T) {
		req := UpdateJobRequest{}

		assert.Nil(t, req.Title)
		assert.Nil(t, req.Status)
		assert.Nil(t, req.ActualCost)
	})

	t.Run("Partial update request", func(t *testing.T) {
		status := JobStatusCompleted

		req := UpdateJobRequest{
			Status: &status,
		}

		assert.NotNil(t, req.Status)
		assert.Nil(t, req.Title)
		assert.Nil(t, req.Description)
	})
}

// Note: Database integration tests require PostgreSQL with UUID extension
// These tests verify the model structures and business logic without database
func TestJobStructure(t *testing.T) {
	t.Run("Create job instance", func(t *testing.T) {
		job := &Job{
			ID:         uuid.New(),
			CustomerID: uuid.New(),
			Title:      "AC Installation",
			JobType:    JobTypeInstallation,
			Status:     JobStatusScheduled,
			Priority:   JobPriorityNormal,
			CreatedBy:  uuid.New(),
		}

		assert.NotEqual(t, uuid.Nil, job.ID)
		assert.NotEqual(t, uuid.Nil, job.CustomerID)
		assert.Equal(t, "AC Installation", job.Title)
		assert.Equal(t, JobTypeInstallation, job.JobType)
		assert.Equal(t, JobStatusScheduled, job.Status)
		assert.Equal(t, JobPriorityNormal, job.Priority)
	})

	t.Run("Job with assignments", func(t *testing.T) {
		jobID := uuid.New()
		userID := uuid.New()

		assignment := JobAssignment{
			ID:     uuid.New(),
			JobID:  jobID,
			UserID: userID,
			Role:   "lead_technician",
		}

		job := &Job{
			ID:          jobID,
			Assignments: []JobAssignment{assignment},
		}

		assert.Len(t, job.Assignments, 1)
		assert.Equal(t, "lead_technician", job.Assignments[0].Role)
	})

	t.Run("Job with materials", func(t *testing.T) {
		jobID := uuid.New()

		material := JobMaterial{
			ID:        uuid.New(),
			JobID:     jobID,
			Name:      "AC Filter",
			Quantity:  2,
			Unit:      "each",
			UnitCost:  25.50,
			TotalCost: 51.00,
		}

		job := &Job{
			ID:        jobID,
			Materials: []JobMaterial{material},
		}

		assert.Len(t, job.Materials, 1)
		assert.Equal(t, "AC Filter", job.Materials[0].Name)
		assert.Equal(t, 51.0, job.Materials[0].TotalCost)
	})

	t.Run("Job with notes", func(t *testing.T) {
		jobID := uuid.New()
		userID := uuid.New()

		note := JobNote{
			ID:     uuid.New(),
			JobID:  jobID,
			UserID: userID,
			Note:   "Customer requested afternoon appointment",
		}

		job := &Job{
			ID:    jobID,
			Notes: []JobNote{note},
		}

		assert.Len(t, job.Notes, 1)
		assert.Equal(t, "Customer requested afternoon appointment", job.Notes[0].Note)
	})
}
