package services

import (
	"context"
	"errors"
	"strings"

	"github.com/google/uuid"
	"gorm.io/gorm"

	apperrors "github.com/javaknight1/servicepro/backend/internal/errors"
	"github.com/javaknight1/servicepro/backend/internal/models"
	"github.com/javaknight1/servicepro/backend/internal/repository"
	"github.com/javaknight1/servicepro/backend/internal/utils/epoch"
	"github.com/javaknight1/servicepro/backend/pkg/validators"
)

// Job domain errors (T223): typed, so the API returns their status, code
// and message as-is; anything else from the job service is a 500.
var (
	// ErrJobNotFound indicates job was not found (or belongs to another tenant)
	ErrJobNotFound = apperrors.NotFound("job_not_found", "Job not found")

	// ErrCustomerNotFound indicates customer was not found
	ErrCustomerNotFound = apperrors.NotFound("customer_not_found", "The specified customer does not exist")

	// ErrUnauthorized indicates user is not authorized for this operation
	ErrUnauthorized = apperrors.Forbidden("forbidden", "You do not have permission to access or change this job")

	// ErrInvalidJobData indicates invalid job data; WithMessage carries the reason
	ErrInvalidJobData = apperrors.BadRequest("invalid_job_data", "Invalid job data")

	// ErrJobCannotBeDeleted: only scheduled or cancelled jobs can be deleted
	ErrJobCannotBeDeleted = apperrors.Conflict("cannot_delete", "Only scheduled or cancelled jobs can be deleted")

	// ErrJobAssignmentNotFound: the technician isn't assigned to the job
	ErrJobAssignmentNotFound = apperrors.NotFound("assignment_not_found", "That technician isn't assigned to this job")

	// ErrMaterialNotFound: no such material on the job
	ErrMaterialNotFound = apperrors.NotFound("material_not_found", "Material not found")
)

// invalidJobData turns a validator error (fixed, user-safe text such as
// "job title is required") into a 400 that keeps the reason.
func invalidJobData(err error) error {
	msg := err.Error()
	if msg == "" {
		return ErrInvalidJobData
	}
	return ErrInvalidJobData.WithMessage(strings.ToUpper(msg[:1]) + msg[1:]).WithCause(err)
}

// JobServiceInterface defines the interface for job service
type JobServiceInterface interface {
	// WithContext returns a service bound to ctx (the request context), so
	// every query is scoped to the caller's tenant.
	WithContext(ctx context.Context) JobServiceInterface

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
	GetScheduledJobs(start, end int64) ([]models.JobResponse, error)
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

	// Status transition operations
	TransitionStatus(jobID uuid.UUID, toStatus models.JobStatus, reason string, notes *string, userID uuid.UUID, userRole models.UserRole) (*models.JobResponse, error)
	GetStatusHistory(jobID uuid.UUID, limit, offset int, sortOrder string) (*models.StatusHistoryResponse, error)
	GetAllowedTransitions(jobID uuid.UUID, userID uuid.UUID, userRole models.UserRole) (*models.AllowedTransitionsResponse, error)
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

// WithContext returns a copy of the service whose repositories and DB handle
// run with ctx.
func (s *JobService) WithContext(ctx context.Context) JobServiceInterface {
	cp := *s
	cp.jobRepo = s.jobRepo.WithContext(ctx)
	cp.customerRepo = s.customerRepo.WithContext(ctx)
	if s.db != nil {
		cp.db = s.db.WithContext(ctx)
	}
	return &cp
}

// CreateJob creates a new job with validation
func (s *JobService) CreateJob(req *models.CreateJobRequest, createdBy uuid.UUID) (*models.JobResponse, error) {
	// Validate request
	if err := s.validator.ValidateCreateRequest(req); err != nil {
		return nil, invalidJobData(err)
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
		ID:                  uuid.New(),
		CustomerID:          req.CustomerID,
		Title:               req.Title,
		Description:         req.Description,
		JobType:             req.JobType,
		Status:              models.JobStatusNew,
		Priority:            req.Priority,
		ScheduledStartAt:    req.ScheduledStartAt,
		ScheduledEndAt:      req.ScheduledEndAt,
		EstimatedDuration:   req.EstimatedDuration,
		EstimatedCost:       req.EstimatedCost,
		ServiceAddress:      serviceAddress,
		InternalNotes:       req.InternalNotes,
		CustomerNotes:       req.CustomerNotes,
		SpecialInstructions: req.SpecialInstructions,
		RequiredMaterials:   req.RequiredMaterials,
		CreatedBy:           createdBy,
	}

	// Compute ScheduledEndAt from start + duration if not explicitly set
	if job.ScheduledEndAt == nil && job.ScheduledStartAt != nil && job.EstimatedDuration > 0 {
		endAt := *job.ScheduledStartAt + int64(job.EstimatedDuration)*60
		job.ScheduledEndAt = &endAt
	}

	// Set default priority if not provided
	if job.Priority == "" {
		job.Priority = models.JobPriorityNormal
	}

	// The job and its first history row are written together. from_status is
	// NOT NULL, so creation is recorded as new -> new (it used to pass "",
	// which always failed and the error was swallowed — audit 7.M1).
	created := "Job created"
	if err := s.jobRepo.CreateWithTransition(job, newTransition(uuid.Nil, models.JobStatusNew, models.JobStatusNew, models.ReasonOther, &created, createdBy)); err != nil {
		return nil, err
	}

	// Create assignments if provided
	if len(req.Assignments) > 0 {
		for _, assignmentReq := range req.Assignments {
			assignment := &models.JobAssignment{
				JobID:      job.ID,
				UserID:     assignmentReq.UserID,
				Role:       assignmentReq.Role,
				AssignedAt: epoch.NowEpoch(), // was left 0 (1970) — audit 7.M2
			}
			if assignment.Role == "" {
				assignment.Role = "technician"
			}
			if err := s.jobRepo.AddAssignment(assignment); err != nil {
				// Log but don't fail the job creation
				// The job was created successfully, assignments can be added later
				continue
			}
		}
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
		return nil, invalidJobData(err)
	}

	// Apply updates
	if req.Title != nil {
		job.Title = *req.Title
	}
	if req.Description != nil {
		job.Description = *req.Description
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
	// Recompute ScheduledEndAt from start + duration when either changes
	if (req.ScheduledStartAt != nil || req.EstimatedDuration != nil) && job.ScheduledStartAt != nil && job.EstimatedDuration > 0 {
		endAt := *job.ScheduledStartAt + int64(job.EstimatedDuration)*60
		job.ScheduledEndAt = &endAt
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
	if req.SpecialInstructions != nil {
		job.SpecialInstructions = req.SpecialInstructions
	}
	if req.RequiredMaterials != nil {
		job.RequiredMaterials = req.RequiredMaterials
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
		return ErrJobCannotBeDeleted
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
func (s *JobService) GetScheduledJobs(start, end int64) ([]models.JobResponse, error) {
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

	if err := s.applyTransition(job, models.JobStatusInProgress, models.ReasonStartWork, nil, userID); err != nil {
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

	var notes *string
	if completionNotes != "" {
		notes = &completionNotes
		job.CompletionNotes = &completionNotes
	}
	if err := s.applyTransition(job, models.JobStatusCompleted, models.ReasonCompleteWork, notes, userID); err != nil {
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

	var notes *string
	if reason != "" {
		notes = &reason
		internal := ""
		if job.InternalNotes != nil {
			internal = *job.InternalNotes
		}
		internal = internal + "\n\nCancellation Reason: " + reason
		job.InternalNotes = &internal
	}
	if err := s.applyTransition(job, models.JobStatusCancelled, models.ReasonCancellation, notes, userID); err != nil {
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

	var notes *string
	if reason != "" {
		notes = &reason
		internal := ""
		if job.InternalNotes != nil {
			internal = *job.InternalNotes
		}
		internal = internal + "\n\nOn Hold Reason: " + reason
		job.InternalNotes = &internal
	}
	if err := s.applyTransition(job, models.JobStatusOnHold, models.ReasonTechnicalIssue, notes, userID); err != nil {
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
		return nil, models.ErrInvalidStatusTransition.WithMessage("Only jobs that are on hold can be resumed")
	}

	if err := s.applyTransition(job, models.JobStatusInProgress, models.ReasonResume, nil, userID); err != nil {
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
		JobID:      jobID,
		UserID:     technicianID,
		Role:       role,
		AssignedAt: epoch.NowEpoch(), // was left 0 (1970) — audit 7.M2
	}

	// Validate assignment
	if err := s.validator.ValidateAssignment(assignment); err != nil {
		return invalidJobData(err)
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
			nowEpoch := epoch.NowEpoch()
			assignment.UnassignedAt = &nowEpoch
			return s.jobRepo.RemoveAssignment(assignment.ID)
		}
	}

	return ErrJobAssignmentNotFound
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
		return invalidJobData(err)
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
		return ErrMaterialNotFound
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
		return invalidJobData(err)
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

	// job_materials has no tenant_id: only delete a material of this job.
	materials, err := s.jobRepo.GetJobMaterials(jobID)
	if err != nil {
		return err
	}
	for _, m := range materials {
		if m.ID == materialID {
			return s.jobRepo.DeleteMaterial(materialID)
		}
	}
	return ErrMaterialNotFound
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
		return invalidJobData(err)
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

// newTransition builds a job history row.
func newTransition(jobID uuid.UUID, from, to models.JobStatus, reason models.StatusTransitionReason, notes *string, changedBy uuid.UUID) *models.JobStatusTransition {
	return &models.JobStatusTransition{
		ID:             uuid.New(),
		JobID:          jobID,
		FromStatus:     from,
		ToStatus:       to,
		Reason:         reason,
		Notes:          notes,
		ChangedBy:      changedBy,
		TransitionedAt: epoch.NowEpoch(),
	}
}

// applyTransition is the only way a job's status changes (T124 / audit
// 7.H7): it validates the move against the job state machine in
// models/job_status.go (models.ValidateTransition — the one source of truth;
// the old duplicate rules in pkg/validators are gone), stamps
// ActualStartAt/ActualEndAt, saves the job and records the history row.
func (s *JobService) applyTransition(job *models.Job, to models.JobStatus, reason models.StatusTransitionReason, notes *string, userID uuid.UUID) error {
	if err := models.ValidateTransition(job, to, notes); err != nil {
		return err
	}
	from := job.Status
	job.Status = to
	job.UpdatedBy = &userID

	now := epoch.NowEpoch()
	switch to {
	case models.JobStatusInProgress:
		if job.ActualStartAt == nil {
			job.ActualStartAt = &now
		}
	case models.JobStatusCompleted:
		if job.ActualStartAt == nil {
			job.ActualStartAt = &now
		}
		job.ActualEndAt = &now
	}

	return s.jobRepo.UpdateWithTransition(job, newTransition(job.ID, from, to, reason, notes, userID))
}

// TransitionStatus transitions a job to a new status
func (s *JobService) TransitionStatus(jobID uuid.UUID, toStatus models.JobStatus, reason string, notes *string, userID uuid.UUID, userRole models.UserRole) (*models.JobResponse, error) {
	job, err := s.jobRepo.GetByID(jobID)
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

	// No-op if same status
	if job.Status == toStatus {
		if err := models.ValidateTransition(job, toStatus, notes); err != nil {
			return nil, err
		}
		response := job.ToResponse()
		return &response, nil
	}

	if err := s.applyTransition(job, toStatus, models.StatusTransitionReason(reason), notes, userID); err != nil {
		return nil, err
	}

	// Reload with relationships
	updatedJob, err := s.jobRepo.GetByID(jobID)
	if err != nil {
		return nil, err
	}

	response := updatedJob.ToResponse()
	return &response, nil
}

// GetStatusHistory retrieves the status history for a job
func (s *JobService) GetStatusHistory(jobID uuid.UUID, limit, offset int, sortOrder string) (*models.StatusHistoryResponse, error) {
	// Set defaults
	if limit <= 0 {
		limit = 20
	}
	if sortOrder != "asc" && sortOrder != "desc" {
		sortOrder = "desc"
	}

	transitions, total, err := s.jobRepo.GetStatusHistory(jobID, limit, offset, sortOrder)
	if err != nil {
		return nil, err
	}

	responses := make([]models.StatusTransitionResponse, len(transitions))
	for i, transition := range transitions {
		responses[i] = transition.ToResponse()
	}

	return &models.StatusHistoryResponse{
		Transitions: responses,
		Total:       total,
	}, nil
}

// GetAllowedTransitions returns the statuses a job can move to next, from
// the single job state machine (models/job_status.go).
func (s *JobService) GetAllowedTransitions(jobID uuid.UUID, userID uuid.UUID, userRole models.UserRole) (*models.AllowedTransitionsResponse, error) {
	job, err := s.jobRepo.GetByID(jobID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrJobNotFound
		}
		return nil, err
	}
	if err := s.validator.CanUserModifyJob(job, userID, userRole); err != nil {
		return nil, ErrUnauthorized
	}
	allowed := models.GetAllowedTransitions(job.Status)
	if allowed == nil {
		allowed = []models.JobStatus{}
	}
	return &models.AllowedTransitionsResponse{
		JobID:              job.ID,
		CurrentStatus:      job.Status,
		AllowedTransitions: allowed,
		IsTerminal:         models.IsTerminalStatus(job.Status),
	}, nil
}
