package validators

import (
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/javaknight1/servicepro/backend/internal/models"
)

var (
	// ErrInvalidJobStatus indicates an invalid job status
	ErrInvalidJobStatus = errors.New("invalid job status")

	// ErrInvalidJobType indicates an invalid job type
	ErrInvalidJobType = errors.New("invalid job type")

	// ErrInvalidJobPriority indicates an invalid job priority
	ErrInvalidJobPriority = errors.New("invalid job priority")

	// ErrInvalidSchedule indicates invalid scheduling dates
	ErrInvalidSchedule = errors.New("invalid schedule: end date must be after start date")

	// ErrScheduleInPast indicates scheduling in the past
	ErrScheduleInPast = errors.New("scheduled start date cannot be in the past")

	// ErrInvalidDuration indicates an invalid duration
	ErrInvalidDuration = errors.New("duration must be positive")

	// ErrInvalidCost indicates an invalid cost
	ErrInvalidCost = errors.New("cost must be non-negative")

	// ErrInvalidStatusTransition indicates an invalid status change
	ErrInvalidStatusTransition = errors.New("invalid status transition")

	// ErrJobNotScheduled indicates job is not scheduled
	ErrJobNotScheduled = errors.New("job must be scheduled before starting")

	// ErrJobNotInProgress indicates job is not in progress
	ErrJobNotInProgress = errors.New("job must be in progress to complete")

	// ErrMissingCustomerID indicates missing customer ID
	ErrMissingCustomerID = errors.New("customer ID is required")

	// ErrMissingTitle indicates missing job title
	ErrMissingTitle = errors.New("job title is required")

	// ErrInvalidServiceAddress indicates invalid service address
	ErrInvalidServiceAddress = errors.New("invalid service address")
)

// JobValidator handles validation logic for jobs
type JobValidator struct{}

// NewJobValidator creates a new job validator
func NewJobValidator() *JobValidator {
	return &JobValidator{}
}

// ValidateCreateRequest validates a create job request
func (v *JobValidator) ValidateCreateRequest(req *models.CreateJobRequest) error {
	if req.CustomerID == uuid.Nil {
		return ErrMissingCustomerID
	}

	if req.Title == "" {
		return ErrMissingTitle
	}

	if err := v.ValidateJobType(req.JobType); err != nil {
		return err
	}

	if req.Priority != "" {
		if err := v.ValidatePriority(req.Priority); err != nil {
			return err
		}
	}

	if err := v.ValidateSchedule(req.ScheduledStartAt, req.ScheduledEndAt); err != nil {
		return err
	}

	if req.EstimatedDuration < 0 {
		return ErrInvalidDuration
	}

	if req.EstimatedCost < 0 {
		return ErrInvalidCost
	}

	if err := v.ValidateServiceAddress(&req.ServiceAddress); err != nil {
		return err
	}

	return nil
}

// ValidateUpdateRequest validates an update job request
func (v *JobValidator) ValidateUpdateRequest(req *models.UpdateJobRequest, currentJob *models.Job) error {
	if req.Status != nil {
		if err := v.ValidateStatusTransition(currentJob.Status, *req.Status); err != nil {
			return err
		}
	}

	if req.JobType != nil {
		if err := v.ValidateJobType(*req.JobType); err != nil {
			return err
		}
	}

	if req.Priority != nil {
		if err := v.ValidatePriority(*req.Priority); err != nil {
			return err
		}
	}

	// Validate schedule if both dates are provided
	var schedStart, schedEnd *time.Time
	if req.ScheduledStartAt != nil {
		schedStart = req.ScheduledStartAt
	} else if currentJob.ScheduledStartAt != nil {
		schedStart = currentJob.ScheduledStartAt
	}

	if req.ScheduledEndAt != nil {
		schedEnd = req.ScheduledEndAt
	} else if currentJob.ScheduledEndAt != nil {
		schedEnd = currentJob.ScheduledEndAt
	}

	if err := v.ValidateSchedule(schedStart, schedEnd); err != nil {
		return err
	}

	if req.EstimatedDuration != nil && *req.EstimatedDuration < 0 {
		return ErrInvalidDuration
	}

	if req.EstimatedCost != nil && *req.EstimatedCost < 0 {
		return ErrInvalidCost
	}

	if req.ActualCost != nil && *req.ActualCost < 0 {
		return ErrInvalidCost
	}

	if req.TaxAmount != nil && *req.TaxAmount < 0 {
		return ErrInvalidCost
	}

	if req.ServiceAddress != nil {
		if err := v.ValidateServiceAddress(req.ServiceAddress); err != nil {
			return err
		}
	}

	return nil
}

// ValidateJobType validates a job type
func (v *JobValidator) ValidateJobType(jobType models.JobType) error {
	validTypes := map[models.JobType]bool{
		models.JobTypeInstallation: true,
		models.JobTypeMaintenance:  true,
		models.JobTypeRepair:       true,
		models.JobTypeInspection:   true,
		models.JobTypeEmergency:    true,
	}

	if !validTypes[jobType] {
		return fmt.Errorf("%w: %s", ErrInvalidJobType, jobType)
	}

	return nil
}

// ValidatePriority validates a job priority
func (v *JobValidator) ValidatePriority(priority models.JobPriority) error {
	validPriorities := map[models.JobPriority]bool{
		models.JobPriorityLow:    true,
		models.JobPriorityNormal: true,
		models.JobPriorityHigh:   true,
		models.JobPriorityUrgent: true,
	}

	if !validPriorities[priority] {
		return fmt.Errorf("%w: %s", ErrInvalidJobPriority, priority)
	}

	return nil
}

// ValidateStatus validates a job status
func (v *JobValidator) ValidateStatus(status models.JobStatus) error {
	validStatuses := map[models.JobStatus]bool{
		models.JobStatusScheduled:  true,
		models.JobStatusInProgress: true,
		models.JobStatusOnHold:     true,
		models.JobStatusCompleted:  true,
		models.JobStatusCancelled:  true,
	}

	if !validStatuses[status] {
		return fmt.Errorf("%w: %s", ErrInvalidJobStatus, status)
	}

	return nil
}

// ValidateSchedule validates scheduling dates
func (v *JobValidator) ValidateSchedule(startAt, endAt *time.Time) error {
	if startAt == nil && endAt == nil {
		return nil // Both nil is valid
	}

	if startAt != nil && endAt != nil {
		if endAt.Before(*startAt) {
			return ErrInvalidSchedule
		}
	}

	// Don't enforce past date check - allow editing historical jobs
	// if startAt != nil && startAt.Before(time.Now()) {
	// 	return ErrScheduleInPast
	// }

	return nil
}

// ValidateStatusTransition validates if a status transition is allowed
func (v *JobValidator) ValidateStatusTransition(from, to models.JobStatus) error {
	// If status is not changing, allow it
	if from == to {
		return nil
	}

	// Define valid transitions
	validTransitions := map[models.JobStatus]map[models.JobStatus]bool{
		models.JobStatusScheduled: {
			models.JobStatusInProgress: true,
			models.JobStatusOnHold:     true,
			models.JobStatusCancelled:  true,
		},
		models.JobStatusInProgress: {
			models.JobStatusOnHold:    true,
			models.JobStatusCompleted: true,
			models.JobStatusCancelled: true,
		},
		models.JobStatusOnHold: {
			models.JobStatusScheduled:  true,
			models.JobStatusInProgress: true,
			models.JobStatusCancelled:  true,
		},
		models.JobStatusCompleted: {
			// Completed jobs can be reopened
			models.JobStatusInProgress: true,
		},
		models.JobStatusCancelled: {
			// Cancelled jobs can be rescheduled
			models.JobStatusScheduled: true,
		},
	}

	if allowed, exists := validTransitions[from]; exists {
		if allowed[to] {
			return nil
		}
	}

	return fmt.Errorf("%w: cannot transition from '%s' to '%s'", ErrInvalidStatusTransition, from, to)
}

// ValidateServiceAddress validates a service address
func (v *JobValidator) ValidateServiceAddress(addr *models.ServiceAddress) error {
	if addr == nil {
		return nil
	}

	// Address is optional, but if provided, validate it
	if addr.Street == "" && addr.City == "" && addr.State == "" && addr.Zip == "" {
		return nil // All empty is valid (use customer's address)
	}

	// If any field is provided, require street, city, and state
	if addr.Street != "" || addr.City != "" || addr.State != "" {
		if addr.Street == "" {
			return fmt.Errorf("%w: street is required", ErrInvalidServiceAddress)
		}
		if addr.City == "" {
			return fmt.Errorf("%w: city is required", ErrInvalidServiceAddress)
		}
		if addr.State == "" {
			return fmt.Errorf("%w: state is required", ErrInvalidServiceAddress)
		}
		if len(addr.State) != 2 {
			return fmt.Errorf("%w: state must be 2-letter code", ErrInvalidServiceAddress)
		}
	}

	return nil
}

// ValidateJobStart validates if a job can be started
func (v *JobValidator) ValidateJobStart(job *models.Job) error {
	if job.Status != models.JobStatusScheduled && job.Status != models.JobStatusOnHold {
		return fmt.Errorf("cannot start job with status '%s'", job.Status)
	}

	return nil
}

// ValidateJobCompletion validates if a job can be completed
func (v *JobValidator) ValidateJobCompletion(job *models.Job) error {
	if job.Status != models.JobStatusInProgress {
		return ErrJobNotInProgress
	}

	if job.ActualStartAt == nil {
		return errors.New("job must have an actual start time to be completed")
	}

	return nil
}

// ValidateMaterial validates a job material
func (v *JobValidator) ValidateMaterial(material *models.JobMaterial) error {
	if material.Name == "" {
		return errors.New("material name is required")
	}

	if material.Quantity <= 0 {
		return errors.New("material quantity must be positive")
	}

	if material.UnitCost < 0 {
		return errors.New("material unit cost must be non-negative")
	}

	if material.Unit == "" {
		return errors.New("material unit is required")
	}

	return nil
}

// ValidateAssignment validates a job assignment
func (v *JobValidator) ValidateAssignment(assignment *models.JobAssignment) error {
	if assignment.JobID == uuid.Nil {
		return errors.New("job ID is required")
	}

	if assignment.UserID == uuid.Nil {
		return errors.New("user ID is required")
	}

	if assignment.HoursWorked < 0 {
		return errors.New("hours worked must be non-negative")
	}

	return nil
}

// ValidateNote validates a job note
func (v *JobValidator) ValidateNote(note *models.JobNote) error {
	if note.JobID == uuid.Nil {
		return errors.New("job ID is required")
	}

	if note.UserID == uuid.Nil {
		return errors.New("user ID is required")
	}

	if note.Note == "" {
		return errors.New("note content is required")
	}

	if len(note.Note) > 10000 {
		return errors.New("note content exceeds maximum length of 10,000 characters")
	}

	return nil
}

// CanUserModifyJob checks if a user can modify a job
func (v *JobValidator) CanUserModifyJob(job *models.Job, userID uuid.UUID, userRole models.UserRole) error {
	// Admins can modify any job
	if userRole == models.RoleAdmin {
		return nil
	}

	// Managers can modify any job
	if userRole == models.RoleManager {
		return nil
	}

	// Technicians can only modify jobs they created or are assigned to
	if userRole == models.RoleTechnician {
		if job.CreatedBy == userID {
			return nil
		}

		// Check if user is assigned to this job
		for _, assignment := range job.Assignments {
			if assignment.UserID == userID && assignment.UnassignedAt == nil {
				return nil
			}
		}

		return errors.New("insufficient permissions to modify this job")
	}

	// Regular users can only access jobs they created
	if userRole == models.RoleUser {
		if job.CreatedBy == userID {
			return nil
		}
		return errors.New("insufficient permissions to access this job")
	}

	return errors.New("insufficient permissions")
}

// ValidateBusinessRules validates complex business rules for jobs
func (v *JobValidator) ValidateBusinessRules(job *models.Job) []string {
	var warnings []string

	// Warn if emergency job is not high priority
	if job.JobType == models.JobTypeEmergency && job.Priority != models.JobPriorityUrgent {
		warnings = append(warnings, "Emergency jobs should typically have urgent priority")
	}

	// Warn if scheduled dates are far in the future
	if job.ScheduledStartAt != nil {
		daysUntil := time.Until(*job.ScheduledStartAt).Hours() / 24
		if daysUntil > 90 {
			warnings = append(warnings, fmt.Sprintf("Job scheduled %.0f days in the future", daysUntil))
		}
	}

	// Warn if estimated duration is very long
	if job.EstimatedDuration > 480 { // > 8 hours
		warnings = append(warnings, "Job estimated duration exceeds 8 hours - consider splitting into multiple jobs")
	}

	// Warn if actual cost significantly differs from estimate
	if job.EstimatedCost > 0 && job.ActualCost > 0 {
		variance := (job.ActualCost - job.EstimatedCost) / job.EstimatedCost * 100
		if variance > 25 {
			warnings = append(warnings, fmt.Sprintf("Actual cost is %.0f%% higher than estimate", variance))
		} else if variance < -25 {
			warnings = append(warnings, fmt.Sprintf("Actual cost is %.0f%% lower than estimate", -variance))
		}
	}

	// Warn if job has no assignments
	if len(job.Assignments) == 0 && job.Status != models.JobStatusScheduled {
		warnings = append(warnings, "Job has no technician assignments")
	}

	return warnings
}
