package services

import (
	"errors"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/javaknight1/servicepro/backend/internal/models"
	"github.com/javaknight1/servicepro/backend/internal/repository"
	"github.com/javaknight1/servicepro/backend/pkg/validators"
)

var (
	// ErrJobNotFound indicates job was not found
	ErrJobNotFound = errors.New("job not found")

	// ErrCustomerNotFound indicates customer was not found
	ErrCustomerNotFound = errors.New("customer not found")

	// ErrUnauthorized indicates user is not authorized for this operation
	ErrUnauthorized = errors.New("unauthorized to perform this operation")

	// ErrInvalidJobData indicates invalid job data
	ErrInvalidJobData = errors.New("invalid job data")
)

// JobServiceInterface defines the interface for job service
type JobServiceInterface interface {
	// Basic CRUD operations
	CreateJob(req *models.CreateJobRequest, createdBy uuid.UUID) (*models.JobResponse, error)
	GetJobByID(id uuid.UUID, userID uuid.UUID, userRole models.UserRole) (*models.JobResponse, error)
	GetJobByJobNumber(jobNumber string, userID uuid.UUID, userRole models.UserRole) (*models.JobResponse, error)
	UpdateJob(id uuid.UUID, req *models.UpdateJobRequest, userID uuid.UUID, userRole models.UserRole) (*models.JobResponse, error)
	DeleteJob(id uuid.UUID, userID uuid.UUID, userRole models.UserRole) error

	// List and filter operations
	ListJobs(filter *models.JobListFilter, userID uuid.UUID, userRole models.UserRole) ([]models.JobResponse, int64, error)
	GetCustomerJobs(customerID uuid.UUID, limit, offset int) ([]models.JobResponse, int64, error)
	GetMyJobs(userID uuid.UUID) ([]models.JobResponse, error)
	GetScheduledJobs(start, end time.Time) ([]models.JobResponse, error)
	GetOverdueJobs() ([]models.JobResponse, error)

	// Status transitions
	StartJob(id uuid.UUID, userID uuid.UUID, userRole models.UserRole) (*models.JobResponse, error)
	CompleteJob(id uuid.UUID, completionNotes string, userID uuid.UUID, userRole models.UserRole) (*models.JobResponse, error)
	CancelJob(id uuid.UUID, reason string, userID uuid.UUID, userRole models.UserRole) (*models.JobResponse, error)
	PutOnHold(id uuid.UUID, reason string, userID uuid.UUID, userRole models.UserRole) (*models.JobResponse, error)
	ResumeJob(id uuid.UUID, userID uuid.UUID, userRole models.UserRole) (*models.JobResponse, error)

	// Assignment operations
	AssignTechnician(jobID, technicianID uuid.UUID, role string, userID uuid.UUID, userRole models.UserRole) error
	UnassignTechnician(jobID, technicianID uuid.UUID, userID uuid.UUID, userRole models.UserRole) error

	// Material operations
	AddMaterial(jobID uuid.UUID, material *models.JobMaterial, userID uuid.UUID, userRole models.UserRole) error
	UpdateMaterial(materialID uuid.UUID, material *models.JobMaterial, userID uuid.UUID, userRole models.UserRole) error
	DeleteMaterial(materialID uuid.UUID, jobID uuid.UUID, userID uuid.UUID, userRole models.UserRole) error

	// Note operations
	AddNote(jobID, userID uuid.UUID, note string, isInternal bool, userRole models.UserRole) error

	// Statistics
	GetJobStats() (map[string]interface{}, error)
	GetTechnicianWorkload(userID uuid.UUID) (map[string]interface{}, error)
}

// JobService handles job business logic
type JobService struct {
	jobRepo      repository.JobRepositoryInterface
	customerRepo repository.CustomerRepositoryInterface
	validator    *validators.JobValidator
	db           *gorm.DB
}

// NewJobService creates a new job service
func NewJobService(
	jobRepo repository.JobRepositoryInterface,
	customerRepo repository.CustomerRepositoryInterface,
	db *gorm.DB,
) *JobService {
	return &JobService{
		jobRepo:      jobRepo,
		customerRepo: customerRepo,
		validator:    validators.NewJobValidator(),
		db:           db,
	}
}

// CreateJob creates a new job with validation
func (s *JobService) CreateJob(req *models.CreateJobRequest, createdBy uuid.UUID) (*models.JobResponse, error) {
	// Validate request
	if err := s.validator.ValidateCreateRequest(req); err != nil {
		return nil, err
	}

	// Verify customer exists
	customer, err := s.customerRepo.GetByID(req.CustomerID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrCustomerNotFound
		}
		return nil, err
	}

	// Use customer's address as service address if not provided
	serviceAddress := req.ServiceAddress
	if serviceAddress.Street == "" && serviceAddress.City == "" {
		serviceAddress = models.ServiceAddress{
			Street: customer.BillingAddressStreet,
			City:   customer.BillingAddressCity,
			State:  customer.BillingAddressState,
			Zip:    customer.BillingAddressZip,
		}
	}

	// Create job
	job := &models.Job{
		CustomerID:        req.CustomerID,
		Title:             req.Title,
		Description:       req.Description,
		JobType:           req.JobType,
		Status:            models.JobStatusScheduled,
		Priority:          req.Priority,
		ScheduledStartAt:  req.ScheduledStartAt,
		ScheduledEndAt:    req.ScheduledEndAt,
		EstimatedDuration: req.EstimatedDuration,
		EstimatedCost:     req.EstimatedCost,
		ServiceAddress:    serviceAddress,
		InternalNotes:     req.InternalNotes,
		CustomerNotes:     req.CustomerNotes,
		CreatedBy:         createdBy,
	}

	// Set default priority if not provided
	if job.Priority == "" {
		job.Priority = models.JobPriorityNormal
	}

	if err := s.jobRepo.Create(job); err != nil {
		return nil, err
	}

	// Reload with relationships
	createdJob, err := s.jobRepo.GetByID(job.ID)
	if err != nil {
		return nil, err
	}

	// Generate business rule warnings
	warnings := s.validator.ValidateBusinessRules(createdJob)

	response := createdJob.ToResponse()
	if len(warnings) > 0 {
		response.Warnings = warnings
	}

	return &response, nil
}

// GetJobByID retrieves a job by ID with permission check
func (s *JobService) GetJobByID(id uuid.UUID, userID uuid.UUID, userRole models.UserRole) (*models.JobResponse, error) {
	job, err := s.jobRepo.GetByID(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrJobNotFound
		}
		return nil, err
	}

	// Check permissions
	if err := s.validator.CanUserModifyJob(job, userID, userRole); err != nil {
		return nil, ErrUnauthorized
	}

	response := job.ToResponse()
	return &response, nil
}

// GetJobByJobNumber retrieves a job by job number with permission check
func (s *JobService) GetJobByJobNumber(jobNumber string, userID uuid.UUID, userRole models.UserRole) (*models.JobResponse, error) {
	job, err := s.jobRepo.GetByJobNumber(jobNumber)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrJobNotFound
		}
		return nil, err
	}

	// Check permissions
	if err := s.validator.CanUserModifyJob(job, userID, userRole); err != nil {
		return nil, ErrUnauthorized
	}

	response := job.ToResponse()
	return &response, nil
}

// UpdateJob updates a job with validation and permission check
func (s *JobService) UpdateJob(id uuid.UUID, req *models.UpdateJobRequest, userID uuid.UUID, userRole models.UserRole) (*models.JobResponse, error) {
	// Get existing job
	job, err := s.jobRepo.GetByID(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrJobNotFound
		}
		return nil, err
	}

	// Check permissions
	if err := s.validator.CanUserModifyJob(job, userID, userRole); err != nil {
		return nil, ErrUnauthorized
	}

	// Validate update request
	if err := s.validator.ValidateUpdateRequest(req, job); err != nil {
		return nil, err
	}

	// Apply updates
	if req.Title != nil {
		job.Title = *req.Title
	}
	if req.Description != nil {
		job.Description = *req.Description
	}
	if req.Status != nil {
		job.Status = *req.Status
	}
	if req.Priority != nil {
		job.Priority = *req.Priority
	}
	if req.JobType != nil {
		job.JobType = *req.JobType
	}
	if req.ScheduledStartAt != nil {
		job.ScheduledStartAt = req.ScheduledStartAt
	}
	if req.ScheduledEndAt != nil {
		job.ScheduledEndAt = req.ScheduledEndAt
	}
	if req.EstimatedDuration != nil {
		job.EstimatedDuration = *req.EstimatedDuration
	}
	if req.EstimatedCost != nil {
		job.EstimatedCost = *req.EstimatedCost
	}
	if req.ActualCost != nil {
		job.ActualCost = *req.ActualCost
	}
	if req.TaxAmount != nil {
		job.TaxAmount = *req.TaxAmount
	}
	if req.InternalNotes != nil {
		job.InternalNotes = req.InternalNotes
	}
	if req.CustomerNotes != nil {
		job.CustomerNotes = req.CustomerNotes
	}
	if req.CompletionNotes != nil {
		job.CompletionNotes = req.CompletionNotes
	}
	if req.RequiresFollowUp != nil {
		job.RequiresFollowUp = *req.RequiresFollowUp
	}
	if req.FollowUpDate != nil {
		job.FollowUpDate = req.FollowUpDate
	}
	if req.ServiceAddress != nil {
		job.ServiceAddress = *req.ServiceAddress
	}

	job.UpdatedBy = &userID

	// Save updates
	if err := s.jobRepo.Update(job); err != nil {
		return nil, err
	}

	// Reload with relationships
	updatedJob, err := s.jobRepo.GetByID(id)
	if err != nil {
		return nil, err
	}

	// Generate business rule warnings
	warnings := s.validator.ValidateBusinessRules(updatedJob)

	response := updatedJob.ToResponse()
	if len(warnings) > 0 {
		response.Warnings = warnings
	}

	return &response, nil
}

// DeleteJob soft deletes a job with permission check
func (s *JobService) DeleteJob(id uuid.UUID, userID uuid.UUID, userRole models.UserRole) error {
	// Get existing job
	job, err := s.jobRepo.GetByID(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrJobNotFound
		}
		return err
	}

	// Check permissions
	if err := s.validator.CanUserModifyJob(job, userID, userRole); err != nil {
		return ErrUnauthorized
	}

	// Only allow deletion of scheduled or cancelled jobs
	if job.Status != models.JobStatusScheduled && job.Status != models.JobStatusCancelled {
		return errors.New("can only delete scheduled or cancelled jobs")
	}

	return s.jobRepo.Delete(id)
}

// ListJobs retrieves jobs with filtering
func (s *JobService) ListJobs(filter *models.JobListFilter, userID uuid.UUID, userRole models.UserRole) ([]models.JobResponse, int64, error) {
	// Technicians can only see their assigned jobs unless filter explicitly requests all
	if userRole == models.UserRoleTechnician && filter.AssignedUserID == nil {
		filter.AssignedUserID = &userID
	}

	jobs, total, err := s.jobRepo.List(filter)
	if err != nil {
		return nil, 0, err
	}

	responses := make([]models.JobResponse, len(jobs))
	for i, job := range jobs {
		responses[i] = job.ToResponse()
	}

	return responses, total, nil
}

// GetCustomerJobs retrieves all jobs for a customer
func (s *JobService) GetCustomerJobs(customerID uuid.UUID, limit, offset int) ([]models.JobResponse, int64, error) {
	jobs, total, err := s.jobRepo.GetByCustomer(customerID, limit, offset)
	if err != nil {
		return nil, 0, err
	}

	responses := make([]models.JobResponse, len(jobs))
	for i, job := range jobs {
		responses[i] = job.ToResponse()
	}

	return responses, total, nil
}

// GetMyJobs retrieves jobs assigned to the current user
func (s *JobService) GetMyJobs(userID uuid.UUID) ([]models.JobResponse, error) {
	jobs, err := s.jobRepo.GetByAssignedUser(userID)
	if err != nil {
		return nil, err
	}

	responses := make([]models.JobResponse, len(jobs))
	for i, job := range jobs {
		responses[i] = job.ToResponse()
	}

	return responses, nil
}

// GetScheduledJobs retrieves jobs scheduled within a date range
func (s *JobService) GetScheduledJobs(start, end time.Time) ([]models.JobResponse, error) {
	jobs, err := s.jobRepo.GetScheduledJobs(start, end)
	if err != nil {
		return nil, err
	}

	responses := make([]models.JobResponse, len(jobs))
	for i, job := range jobs {
		responses[i] = job.ToResponse()
	}

	return responses, nil
}

// GetOverdueJobs retrieves overdue jobs
func (s *JobService) GetOverdueJobs() ([]models.JobResponse, error) {
	jobs, err := s.jobRepo.GetOverdueJobs()
	if err != nil {
		return nil, err
	}

	responses := make([]models.JobResponse, len(jobs))
	for i, job := range jobs {
		responses[i] = job.ToResponse()
	}

	return responses, nil
}

// StartJob transitions a job to in_progress status
func (s *JobService) StartJob(id uuid.UUID, userID uuid.UUID, userRole models.UserRole) (*models.JobResponse, error) {
	job, err := s.jobRepo.GetByID(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrJobNotFound
		}
		return nil, err
	}

	// Check permissions
	if err := s.validator.CanUserModifyJob(job, userID, userRole); err != nil {
		return nil, ErrUnauthorized
	}

	// Validate can start
	if err := s.validator.ValidateJobStart(job); err != nil {
		return nil, err
	}

	// Update status and set actual start time
	now := time.Now()
	job.Status = models.JobStatusInProgress
	job.ActualStartAt = &now
	job.UpdatedBy = &userID

	if err := s.jobRepo.Update(job); err != nil {
		return nil, err
	}

	updatedJob, err := s.jobRepo.GetByID(id)
	if err != nil {
		return nil, err
	}

	response := updatedJob.ToResponse()
	return &response, nil
}

// CompleteJob transitions a job to completed status
func (s *JobService) CompleteJob(id uuid.UUID, completionNotes string, userID uuid.UUID, userRole models.UserRole) (*models.JobResponse, error) {
	job, err := s.jobRepo.GetByID(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrJobNotFound
		}
		return nil, err
	}

	// Check permissions
	if err := s.validator.CanUserModifyJob(job, userID, userRole); err != nil {
		return nil, ErrUnauthorized
	}

	// Validate can complete
	if err := s.validator.ValidateJobCompletion(job); err != nil {
		return nil, err
	}

	// Update status and set actual end time
	now := time.Now()
	job.Status = models.JobStatusCompleted
	job.ActualEndAt = &now
	job.CompletionNotes = &completionNotes
	job.UpdatedBy = &userID

	if err := s.jobRepo.Update(job); err != nil {
		return nil, err
	}

	updatedJob, err := s.jobRepo.GetByID(id)
	if err != nil {
		return nil, err
	}

	response := updatedJob.ToResponse()
	return &response, nil
}

// CancelJob transitions a job to cancelled status
func (s *JobService) CancelJob(id uuid.UUID, reason string, userID uuid.UUID, userRole models.UserRole) (*models.JobResponse, error) {
	job, err := s.jobRepo.GetByID(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrJobNotFound
		}
		return nil, err
	}

	// Check permissions (only admin and manager can cancel jobs)
	if userRole != models.UserRoleAdmin && userRole != models.UserRoleManager {
		return nil, ErrUnauthorized
	}

	// Validate status transition
	if err := s.validator.ValidateStatusTransition(job.Status, models.JobStatusCancelled); err != nil {
		return nil, err
	}

	// Update status and add cancellation note
	job.Status = models.JobStatusCancelled
	if reason != "" {
		notes := ""
		if job.InternalNotes != nil {
			notes = *job.InternalNotes
		}
		notes = notes + "\n\nCancellation Reason: " + reason
		job.InternalNotes = &notes
	}
	job.UpdatedBy = &userID

	if err := s.jobRepo.Update(job); err != nil {
		return nil, err
	}

	updatedJob, err := s.jobRepo.GetByID(id)
	if err != nil {
		return nil, err
	}

	response := updatedJob.ToResponse()
	return &response, nil
}

// PutOnHold transitions a job to on_hold status
func (s *JobService) PutOnHold(id uuid.UUID, reason string, userID uuid.UUID, userRole models.UserRole) (*models.JobResponse, error) {
	job, err := s.jobRepo.GetByID(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrJobNotFound
		}
		return nil, err
	}

	// Check permissions
	if err := s.validator.CanUserModifyJob(job, userID, userRole); err != nil {
		return nil, ErrUnauthorized
	}

	// Validate status transition
	if err := s.validator.ValidateStatusTransition(job.Status, models.JobStatusOnHold); err != nil {
		return nil, err
	}

	// Update status and add reason
	job.Status = models.JobStatusOnHold
	if reason != "" {
		notes := ""
		if job.InternalNotes != nil {
			notes = *job.InternalNotes
		}
		notes = notes + "\n\nOn Hold Reason: " + reason
		job.InternalNotes = &notes
	}
	job.UpdatedBy = &userID

	if err := s.jobRepo.Update(job); err != nil {
		return nil, err
	}

	updatedJob, err := s.jobRepo.GetByID(id)
	if err != nil {
		return nil, err
	}

	response := updatedJob.ToResponse()
	return &response, nil
}

// ResumeJob transitions a job from on_hold back to in_progress
func (s *JobService) ResumeJob(id uuid.UUID, userID uuid.UUID, userRole models.UserRole) (*models.JobResponse, error) {
	job, err := s.jobRepo.GetByID(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrJobNotFound
		}
		return nil, err
	}

	// Check permissions
	if err := s.validator.CanUserModifyJob(job, userID, userRole); err != nil {
		return nil, ErrUnauthorized
	}

	// Must be on hold to resume
	if job.Status != models.JobStatusOnHold {
		return nil, errors.New("can only resume jobs that are on hold")
	}

	// Validate status transition
	if err := s.validator.ValidateStatusTransition(job.Status, models.JobStatusInProgress); err != nil {
		return nil, err
	}

	// Update status
	job.Status = models.JobStatusInProgress
	job.UpdatedBy = &userID

	if err := s.jobRepo.Update(job); err != nil {
		return nil, err
	}

	updatedJob, err := s.jobRepo.GetByID(id)
	if err != nil {
		return nil, err
	}

	response := updatedJob.ToResponse()
	return &response, nil
}

// AssignTechnician assigns a technician to a job
func (s *JobService) AssignTechnician(jobID, technicianID uuid.UUID, role string, userID uuid.UUID, userRole models.UserRole) error {
	// Get job to check permissions
	_, err := s.jobRepo.GetByID(jobID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrJobNotFound
		}
		return err
	}

	// Only admin and manager can assign technicians
	if userRole != models.UserRoleAdmin && userRole != models.UserRoleManager {
		return ErrUnauthorized
	}

	assignment := &models.JobAssignment{
		JobID:  jobID,
		UserID: technicianID,
		Role:   role,
	}

	// Validate assignment
	if err := s.validator.ValidateAssignment(assignment); err != nil {
		return err
	}

	return s.jobRepo.AddAssignment(assignment)
}

// UnassignTechnician removes a technician from a job
func (s *JobService) UnassignTechnician(jobID, technicianID uuid.UUID, userID uuid.UUID, userRole models.UserRole) error {
	// Get job to check permissions
	job, err := s.jobRepo.GetByID(jobID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrJobNotFound
		}
		return err
	}

	// Only admin and manager can unassign technicians
	if userRole != models.UserRoleAdmin && userRole != models.UserRoleManager {
		return ErrUnauthorized
	}

	// Find the assignment
	for _, assignment := range job.Assignments {
		if assignment.UserID == technicianID && assignment.UnassignedAt == nil {
			now := time.Now()
			assignment.UnassignedAt = &now
			return s.jobRepo.RemoveAssignment(assignment.ID)
		}
	}

	return errors.New("assignment not found")
}

// AddMaterial adds a material to a job
func (s *JobService) AddMaterial(jobID uuid.UUID, material *models.JobMaterial, userID uuid.UUID, userRole models.UserRole) error {
	// Get job to check permissions
	job, err := s.jobRepo.GetByID(jobID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrJobNotFound
		}
		return err
	}

	// Check permissions
	if err := s.validator.CanUserModifyJob(job, userID, userRole); err != nil {
		return ErrUnauthorized
	}

	material.JobID = jobID

	// Validate material
	if err := s.validator.ValidateMaterial(material); err != nil {
		return err
	}

	return s.jobRepo.AddMaterial(material)
}

// UpdateMaterial updates a job material
func (s *JobService) UpdateMaterial(materialID uuid.UUID, material *models.JobMaterial, userID uuid.UUID, userRole models.UserRole) error {
	// Get the material first to find the job
	materials, err := s.jobRepo.GetJobMaterials(material.JobID)
	if err != nil {
		return err
	}

	var existingMaterial *models.JobMaterial
	for i := range materials {
		if materials[i].ID == materialID {
			existingMaterial = &materials[i]
			break
		}
	}

	if existingMaterial == nil {
		return errors.New("material not found")
	}

	// Get job to check permissions
	job, err := s.jobRepo.GetByID(existingMaterial.JobID)
	if err != nil {
		return err
	}

	// Check permissions
	if err := s.validator.CanUserModifyJob(job, userID, userRole); err != nil {
		return ErrUnauthorized
	}

	// Update fields
	existingMaterial.Name = material.Name
	existingMaterial.Description = material.Description
	existingMaterial.SKU = material.SKU
	existingMaterial.Quantity = material.Quantity
	existingMaterial.Unit = material.Unit
	existingMaterial.UnitCost = material.UnitCost
	existingMaterial.Billable = material.Billable

	// Validate material
	if err := s.validator.ValidateMaterial(existingMaterial); err != nil {
		return err
	}

	return s.jobRepo.UpdateMaterial(existingMaterial)
}

// DeleteMaterial deletes a job material
func (s *JobService) DeleteMaterial(materialID uuid.UUID, jobID uuid.UUID, userID uuid.UUID, userRole models.UserRole) error {
	// Get job to check permissions
	job, err := s.jobRepo.GetByID(jobID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrJobNotFound
		}
		return err
	}

	// Check permissions
	if err := s.validator.CanUserModifyJob(job, userID, userRole); err != nil {
		return ErrUnauthorized
	}

	return s.jobRepo.DeleteMaterial(materialID)
}

// AddNote adds a note to a job
func (s *JobService) AddNote(jobID, userID uuid.UUID, note string, isInternal bool, userRole models.UserRole) error {
	// Get job to check it exists
	_, err := s.jobRepo.GetByID(jobID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrJobNotFound
		}
		return err
	}

	jobNote := &models.JobNote{
		JobID:      jobID,
		UserID:     userID,
		Note:       note,
		IsInternal: isInternal,
	}

	// Validate note
	if err := s.validator.ValidateNote(jobNote); err != nil {
		return err
	}

	return s.jobRepo.AddNote(jobNote)
}

// GetJobStats retrieves job statistics
func (s *JobService) GetJobStats() (map[string]interface{}, error) {
	return s.jobRepo.GetJobStats()
}

// GetTechnicianWorkload retrieves workload statistics for a technician
func (s *JobService) GetTechnicianWorkload(userID uuid.UUID) (map[string]interface{}, error) {
	return s.jobRepo.GetTechnicianWorkload(userID)
}
