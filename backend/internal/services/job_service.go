package services

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	apperrors "github.com/javaknight1/servicepro/backend/internal/errors"
	"github.com/javaknight1/servicepro/backend/internal/models"
	"github.com/javaknight1/servicepro/backend/internal/repository"
	"github.com/javaknight1/servicepro/backend/internal/tenancy"
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

	// ErrAssigneeNotFound: the user to assign is unknown or not an active
	// member of the job's tenant (T238; other tenants' users look unknown)
	ErrAssigneeNotFound = apperrors.NotFound("assignee_not_found", "The specified user is not an active member of this account")

	// ErrJobPricesForbidden: a caller who can't see money (technicians,
	// T035) tried to set a job's cost, tax or a material's unit cost.
	ErrJobPricesForbidden = apperrors.Forbidden("forbidden", "You can't change prices or costs on this job")
)

// requireAssignee returns ErrAssigneeNotFound unless userID is an active
// member of tenantID (T238).
func requireAssignee(repo repository.JobRepositoryInterface, tenantID, userID uuid.UUID) error {
	ok, err := repo.IsActiveTenantMember(tenantID, userID)
	if err != nil {
		return err
	}
	if !ok {
		return ErrAssigneeNotFound
	}
	return nil
}

// assignmentError maps the repository's AddAssignment membership/visibility
// failures onto the job domain errors.
func assignmentError(err error) error {
	switch {
	case errors.Is(err, repository.ErrAssigneeNotInTenant):
		return ErrAssigneeNotFound
	case errors.Is(err, gorm.ErrRecordNotFound):
		return ErrJobNotFound
	}
	return err
}

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
	// GetMyDay lists the caller's visits on date (YYYY-MM-DD in the tenant's
	// time zone; "" is today) with their sites (T173).
	GetMyDay(userID uuid.UUID, date string) (*models.MyDayResponse, error)
	GetScheduledJobs(start, end int64) ([]models.JobResponse, error)
	GetOverdueJobs() ([]models.JobResponse, error)

	// Status transitions

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
	GetJobStats() (*models.JobStats, error)
	GetTechnicianWorkload(userID uuid.UUID) (map[string]interface{}, error)

	// Status transition operations
	TransitionStatus(jobID uuid.UUID, toStatus models.JobStatus, reason string, notes *string, userID uuid.UUID, userRole models.UserRole) (*models.JobResponse, error)
	GetStatusHistory(jobID uuid.UUID, limit, offset int, sortOrder string) (*models.StatusHistoryResponse, error)
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

// ctx is the context the service was bound to with WithContext (it carries
// the caller's tenant).
func (s *JobService) ctx() context.Context {
	if s.db != nil && s.db.Statement != nil && s.db.Statement.Context != nil {
		return s.db.Statement.Context
	}
	return context.Background()
}

// respond builds the API response for job; callers who may not see money
// (technicians, T035) get it without prices, costs or billing rules.
func (s *JobService) respond(job *models.Job) models.JobResponse {
	response := job.ToResponse()
	if tenancy.MoneyHidden(s.ctx()) {
		response.HideMoney()
	}
	return response
}

// respondAll is respond for a list of jobs.
func (s *JobService) respondAll(jobs []models.Job) []models.JobResponse {
	responses := make([]models.JobResponse, len(jobs))
	for i := range jobs {
		responses[i] = s.respond(&jobs[i])
	}
	return responses
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
		if errors.Is(err, gorm.ErrRecordNotFound) || errors.Is(err, repository.ErrCustomerNotFound) {
			return nil, ErrCustomerNotFound
		}
		return nil, err
	}

	// The site, if any, must be one of this customer's active sites (T186).
	var site *models.Site
	if req.SiteID != nil {
		site, err = resolveJobSite(s.ctx(), s.db, customer.ID, *req.SiteID)
		if err != nil {
			return nil, err
		}
	}

	// Use the site's (else the customer's) address as service address if not provided
	serviceAddress := req.ServiceAddress
	if serviceAddress.Street == "" && serviceAddress.City == "" && site != nil {
		serviceAddress = site.ServiceAddress()
	}
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
		SiteID:              req.SiteID,
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
		PONumber:            models.CleanPONumber(req.PONumber),
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

	// Every assignee must be an active member of the job's tenant (the
	// customer lookup above is tenant-scoped, so its tenant is the caller's).
	// Checked before anything is written (T238).
	for _, a := range req.Assignments {
		if err := requireAssignee(s.jobRepo, customer.TenantID, a.UserID); err != nil {
			return nil, err
		}
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
				// Assignees were validated above; a failure here (e.g. a
				// membership deactivated meanwhile) doesn't undo the job —
				// assignments can be added later.
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

	response := s.respond(createdJob)
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

	response := s.respond(job)
	// The site's address, access notes and contact, for the technician's
	// visit page (T173).
	responses := []models.JobResponse{response}
	if err := s.attachSites([]models.Job{*job}, responses); err != nil {
		return nil, err
	}
	response = responses[0]
	// An invoiced job names its invoice ("View invoice", T029) — not for
	// callers who can't see money (technicians, T035).
	if s.db != nil && !tenancy.MoneyHidden(s.ctx()) &&
		(job.Status == models.JobStatusInvoiced || job.Status == models.JobStatusPaid) {
		invoiceID, err := jobInvoiceID(s.db.WithContext(s.ctx()), job)
		if err != nil {
			return nil, err
		}
		response.InvoiceID = invoiceID
	}
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

	response := s.respond(job)
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

	// Callers who can't see money can't set it either (T035).
	if tenancy.MoneyHidden(s.ctx()) && (req.EstimatedCost != nil || req.ActualCost != nil || req.TaxAmount != nil) {
		return nil, ErrJobPricesForbidden
	}

	// Validate update request
	if err := s.validator.ValidateUpdateRequest(req, job); err != nil {
		return nil, invalidJobData(err)
	}

	// A new site must be one of the job's customer's active sites (T186).
	if req.SiteID != nil {
		if _, err := resolveJobSite(s.ctx(), s.db, job.CustomerID, *req.SiteID); err != nil {
			return nil, err
		}
		job.SiteID = req.SiteID
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
	if req.PONumber != nil {
		job.PONumber = models.CleanPONumber(req.PONumber)
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

	response := s.respond(updatedJob)
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

	return s.respondAll(jobs), total, nil
}

// GetCustomerJobs retrieves all jobs for a customer
func (s *JobService) GetCustomerJobs(customerID uuid.UUID, limit, offset int) ([]models.JobResponse, int64, error) {
	jobs, total, err := s.jobRepo.GetByCustomer(customerID, limit, offset)
	if err != nil {
		return nil, 0, err
	}

	return s.respondAll(jobs), total, nil
}

// GetMyJobs retrieves jobs assigned to the current user
func (s *JobService) GetMyJobs(userID uuid.UUID) ([]models.JobResponse, error) {
	jobs, err := s.jobRepo.GetByAssignedUser(userID)
	if err != nil {
		return nil, err
	}

	return s.respondAll(jobs), nil
}

// ErrInvalidDay: GET /jobs/my-day's date isn't a YYYY-MM-DD date.
var ErrInvalidDay = apperrors.BadRequest("invalid_date", "date must be a calendar date (YYYY-MM-DD)")

// GetMyDay lists the visits assigned to userID that start on date in the
// tenant's time zone (today when date is ""), in time order, each with its
// site (T173). Money is hidden for callers who can't see it, as everywhere.
func (s *JobService) GetMyDay(userID uuid.UUID, date string) (*models.MyDayResponse, error) {
	tenantID, ok := tenancy.TenantID(s.ctx())
	if !ok || s.db == nil {
		return nil, errors.New("job service: no tenant in context")
	}
	loc, tzName, err := tenantLocation(s.ctx(), s.db, tenantID)
	if err != nil {
		return nil, err
	}
	var day time.Time
	if date == "" {
		day = models.CivilDate(time.Now(), loc)
	} else if day, err = time.Parse(models.DateLayout, date); err != nil {
		return nil, ErrInvalidDay
	}
	// Local midnight to the next local midnight (23 or 25 hours across DST).
	start := time.Date(day.Year(), day.Month(), day.Day(), 0, 0, 0, 0, loc)
	end := time.Date(day.Year(), day.Month(), day.Day()+1, 0, 0, 0, 0, loc)

	jobs, err := s.jobRepo.GetAssignedScheduledBetween(userID, start.Unix(), end.Unix())
	if err != nil {
		return nil, err
	}
	visits := s.respondAll(jobs)
	if err := s.attachSites(jobs, visits); err != nil {
		return nil, err
	}
	return &models.MyDayResponse{Date: day.Format(models.DateLayout), Timezone: tzName, Visits: visits}, nil
}

// attachSites sets each response's Site from its job's SiteID (responses[i]
// is jobs[i]); a site the tenant no longer has is left out.
func (s *JobService) attachSites(jobs []models.Job, responses []models.JobResponse) error {
	var ids []uuid.UUID
	seen := map[uuid.UUID]bool{}
	for _, j := range jobs {
		if j.SiteID != nil && !seen[*j.SiteID] {
			seen[*j.SiteID] = true
			ids = append(ids, *j.SiteID)
		}
	}
	if len(ids) == 0 {
		return nil
	}
	sites, err := s.jobRepo.GetSites(ids)
	if err != nil {
		return err
	}
	byID := make(map[uuid.UUID]*models.Site, len(sites))
	for i := range sites {
		byID[sites[i].ID] = &sites[i]
	}
	for i, j := range jobs {
		if j.SiteID == nil {
			continue
		}
		if site, ok := byID[*j.SiteID]; ok {
			responses[i].Site = site.Summary()
		}
	}
	return nil
}

// GetScheduledJobs retrieves jobs scheduled within a date range
func (s *JobService) GetScheduledJobs(start, end int64) ([]models.JobResponse, error) {
	jobs, err := s.jobRepo.GetScheduledJobs(start, end)
	if err != nil {
		return nil, err
	}

	return s.respondAll(jobs), nil
}

// GetOverdueJobs retrieves overdue jobs
func (s *JobService) GetOverdueJobs() ([]models.JobResponse, error) {
	jobs, err := s.jobRepo.GetOverdueJobs()
	if err != nil {
		return nil, err
	}

	return s.respondAll(jobs), nil
}

// AssignTechnician assigns a technician to a job
func (s *JobService) AssignTechnician(jobID, technicianID uuid.UUID, role string, userID uuid.UUID, userRole models.UserRole) error {
	// Get job to check permissions
	job, err := s.jobRepo.GetByID(jobID)
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

	// The technician must be an active member of the job's tenant (T238)
	if err := requireAssignee(s.jobRepo, job.TenantID, technicianID); err != nil {
		return err
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

	return assignmentError(s.jobRepo.AddAssignment(assignment))
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
	// Callers who can't see money log quantities only; the office sets the
	// unit cost (T035).
	if tenancy.MoneyHidden(s.ctx()) && material.UnitCost != 0 {
		return ErrJobPricesForbidden
	}

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
func (s *JobService) GetJobStats() (*models.JobStats, error) {
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
	// Extra work over the customer's not-to-exceed limit needs the
	// approver's OK before it is scheduled (T187).
	if to == models.JobStatusScheduled {
		if err := requireNTEApproval(s.ctx(), s.db, job); err != nil {
			return err
		}
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
		response := s.respond(job)
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

	response := s.respond(updatedJob)
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

	// job_status_transitions has no tenant_id: only read the history of a
	// job the caller can see (their tenant's; a technician's assigned ones).
	if _, err := s.jobRepo.GetByID(jobID); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrJobNotFound
		}
		return nil, err
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
