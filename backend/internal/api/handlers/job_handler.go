package handlers

import (
	"errors"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/javaknight1/servicepro/backend/internal/models"
	"github.com/javaknight1/servicepro/backend/internal/services"
)

// JobHandler handles job-related endpoints
type JobHandler struct {
	jobService services.JobServiceInterface
}

// NewJobHandler creates a new job handler
func NewJobHandler(jobService services.JobServiceInterface) *JobHandler {
	return &JobHandler{
		jobService: jobService,
	}
}

// getUserContext extracts userID and userRole from the gin context with proper nil checks
// Returns (userID, userRole, error) - if error is not nil, appropriate HTTP response has been sent
func (h *JobHandler) getUserContext(c *gin.Context) (uuid.UUID, models.UserRole, error) {
	// Get user ID from context - the permission middleware uses "user_id" as the key
	userIDValue, exists := c.Get("user_id")
	if !exists || userIDValue == nil {
		c.JSON(http.StatusUnauthorized, models.ErrorResponse{
			Error:   "unauthorized",
			Message: "User not authenticated",
		})
		return uuid.Nil, "", errors.New("user not authenticated")
	}
	userID, ok := userIDValue.(uuid.UUID)
	if !ok {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "internal_error",
			Message: "Invalid user ID in context",
		})
		return uuid.Nil, "", errors.New("invalid user ID")
	}

	// Note: The permission middleware doesn't set a role in context
	// For role-based checks within handlers, we default to RoleUser
	// More granular access control should be done via permission middleware
	userRole := models.RoleUser

	return userID, userRole, nil
}

// CreateJob handles POST /api/v1/jobs
// @Summary Create a new job
// @Description Create a new service job/work order
// @Tags jobs
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body models.CreateJobRequest true "Job creation request"
// @Success 201 {object} models.JobResponse
// @Failure 400 {object} models.ErrorResponse "Invalid request"
// @Failure 401 {object} models.ErrorResponse "Unauthorized"
// @Failure 404 {object} models.ErrorResponse "Customer not found"
// @Failure 500 {object} models.ErrorResponse "Internal server error"
// @Router /api/v1/jobs [post]
func (h *JobHandler) CreateJob(c *gin.Context) {
	// Get user from context (set by auth middleware)
	userID, _, err := h.getUserContext(c)
	if err != nil {
		return // response already sent
	}

	var req models.CreateJobRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "invalid_request",
			Message: "Invalid job data: " + err.Error(),
		})
		return
	}

	// Create job
	job, err := h.jobService.CreateJob(&req, userID)
	if err != nil {
		switch {
		case errors.Is(err, services.ErrCustomerNotFound):
			c.JSON(http.StatusNotFound, models.ErrorResponse{
				Error:   "customer_not_found",
				Message: "The specified customer does not exist",
			})
		case errors.Is(err, gorm.ErrRecordNotFound):
			c.JSON(http.StatusNotFound, models.ErrorResponse{
				Error:   "not_found",
				Message: err.Error(),
			})
		default:
			log.Printf("Error creating job: %v", err)
			c.JSON(http.StatusBadRequest, models.ErrorResponse{
				Error:   "creation_failed",
				Message: err.Error(),
			})
		}
		return
	}

	c.JSON(http.StatusCreated, job)
}

// GetJob handles GET /api/v1/jobs/:id
// @Summary Get a job by ID
// @Description Retrieve a job by its ID
// @Tags jobs
// @Produce json
// @Security BearerAuth
// @Param id path string true "Job ID (UUID)"
// @Success 200 {object} models.JobResponse
// @Failure 400 {object} models.ErrorResponse "Invalid ID format"
// @Failure 401 {object} models.ErrorResponse "Unauthorized"
// @Failure 403 {object} models.ErrorResponse "Forbidden"
// @Failure 404 {object} models.ErrorResponse "Job not found"
// @Failure 500 {object} models.ErrorResponse "Internal server error"
// @Router /api/v1/jobs/{id} [get]
func (h *JobHandler) GetJob(c *gin.Context) {
	// Get user info from context
	userID, userRole, err := h.getUserContext(c)
	if err != nil {
		return // response already sent
	}

	// Parse job ID
	jobID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "invalid_id",
			Message: "Invalid job ID format",
		})
		return
	}

	// Get job
	job, err := h.jobService.GetJobByID(jobID, userID, userRole)
	if err != nil {
		switch {
		case errors.Is(err, services.ErrJobNotFound):
			c.JSON(http.StatusNotFound, models.ErrorResponse{
				Error:   "job_not_found",
				Message: "Job not found",
			})
		case errors.Is(err, services.ErrUnauthorized):
			c.JSON(http.StatusForbidden, models.ErrorResponse{
				Error:   "forbidden",
				Message: "You do not have permission to view this job",
			})
		default:
			log.Printf("Error getting job: %v", err)
			c.JSON(http.StatusInternalServerError, models.ErrorResponse{
				Error:   "internal_error",
				Message: "Failed to retrieve job",
			})
		}
		return
	}

	c.JSON(http.StatusOK, job)
}

// GetJobByNumber handles GET /api/v1/jobs/number/:jobNumber
// @Summary Get a job by job number
// @Description Retrieve a job by its job number
// @Tags jobs
// @Produce json
// @Security BearerAuth
// @Param jobNumber path string true "Job Number (e.g., JOB-20240122-0001)"
// @Success 200 {object} models.JobResponse
// @Failure 401 {object} models.ErrorResponse "Unauthorized"
// @Failure 403 {object} models.ErrorResponse "Forbidden"
// @Failure 404 {object} models.ErrorResponse "Job not found"
// @Failure 500 {object} models.ErrorResponse "Internal server error"
// @Router /api/v1/jobs/number/{jobNumber} [get]
func (h *JobHandler) GetJobByNumber(c *gin.Context) {
	// Get user info from context
	userID, userRole, err := h.getUserContext(c)
	if err != nil {
		return // response already sent
	}

	jobNumber := c.Param("jobNumber")
	if jobNumber == "" {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "invalid_request",
			Message: "Job number is required",
		})
		return
	}

	// Get job
	job, err := h.jobService.GetJobByJobNumber(jobNumber, userID, userRole)
	if err != nil {
		switch {
		case errors.Is(err, services.ErrJobNotFound):
			c.JSON(http.StatusNotFound, models.ErrorResponse{
				Error:   "job_not_found",
				Message: "Job not found",
			})
		case errors.Is(err, services.ErrUnauthorized):
			c.JSON(http.StatusForbidden, models.ErrorResponse{
				Error:   "forbidden",
				Message: "You do not have permission to view this job",
			})
		default:
			log.Printf("Error getting job by number: %v", err)
			c.JSON(http.StatusInternalServerError, models.ErrorResponse{
				Error:   "internal_error",
				Message: "Failed to retrieve job",
			})
		}
		return
	}

	c.JSON(http.StatusOK, job)
}

// UpdateJob handles PUT /api/v1/jobs/:id
// @Summary Update a job
// @Description Update an existing job
// @Tags jobs
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Job ID (UUID)"
// @Param request body models.UpdateJobRequest true "Job update request"
// @Success 200 {object} models.JobResponse
// @Failure 400 {object} models.ErrorResponse "Invalid request"
// @Failure 401 {object} models.ErrorResponse "Unauthorized"
// @Failure 403 {object} models.ErrorResponse "Forbidden"
// @Failure 404 {object} models.ErrorResponse "Job not found"
// @Failure 500 {object} models.ErrorResponse "Internal server error"
// @Router /api/v1/jobs/{id} [put]
func (h *JobHandler) UpdateJob(c *gin.Context) {
	// Get user info from context
	userID, userRole, err := h.getUserContext(c)
	if err != nil {
		return // response already sent
	}

	// Parse job ID
	jobID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "invalid_id",
			Message: "Invalid job ID format",
		})
		return
	}

	var req models.UpdateJobRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "invalid_request",
			Message: "Invalid job data: " + err.Error(),
		})
		return
	}

	// Update job
	job, err := h.jobService.UpdateJob(jobID, &req, userID, userRole)
	if err != nil {
		switch {
		case errors.Is(err, services.ErrJobNotFound):
			c.JSON(http.StatusNotFound, models.ErrorResponse{
				Error:   "job_not_found",
				Message: "Job not found",
			})
		case errors.Is(err, services.ErrUnauthorized):
			c.JSON(http.StatusForbidden, models.ErrorResponse{
				Error:   "forbidden",
				Message: "You do not have permission to update this job",
			})
		default:
			log.Printf("Error updating job: %v", err)
			c.JSON(http.StatusBadRequest, models.ErrorResponse{
				Error:   "update_failed",
				Message: err.Error(),
			})
		}
		return
	}

	c.JSON(http.StatusOK, job)
}

// DeleteJob handles DELETE /api/v1/jobs/:id
// @Summary Delete a job
// @Description Soft delete a job (only scheduled or cancelled jobs)
// @Tags jobs
// @Produce json
// @Security BearerAuth
// @Param id path string true "Job ID (UUID)"
// @Success 204 "No Content"
// @Failure 400 {object} models.ErrorResponse "Invalid request or job cannot be deleted"
// @Failure 401 {object} models.ErrorResponse "Unauthorized"
// @Failure 403 {object} models.ErrorResponse "Forbidden"
// @Failure 404 {object} models.ErrorResponse "Job not found"
// @Failure 500 {object} models.ErrorResponse "Internal server error"
// @Router /api/v1/jobs/{id} [delete]
func (h *JobHandler) DeleteJob(c *gin.Context) {
	// Get user info from context
	userID, userRole, err := h.getUserContext(c)
	if err != nil {
		return // response already sent
	}

	// Parse job ID
	jobID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "invalid_id",
			Message: "Invalid job ID format",
		})
		return
	}

	// Delete job
	err = h.jobService.DeleteJob(jobID, userID, userRole)
	if err != nil {
		switch {
		case errors.Is(err, services.ErrJobNotFound):
			c.JSON(http.StatusNotFound, models.ErrorResponse{
				Error:   "job_not_found",
				Message: "Job not found",
			})
		case errors.Is(err, services.ErrUnauthorized):
			c.JSON(http.StatusForbidden, models.ErrorResponse{
				Error:   "forbidden",
				Message: "You do not have permission to delete this job",
			})
		default:
			log.Printf("Error deleting job: %v", err)
			c.JSON(http.StatusBadRequest, models.ErrorResponse{
				Error:   "delete_failed",
				Message: err.Error(),
			})
		}
		return
	}

	c.Status(http.StatusNoContent)
}

// ListJobs handles GET /api/v1/jobs
// @Summary List jobs with filters
// @Description Retrieve a paginated list of jobs with optional filters
// @Tags jobs
// @Produce json
// @Security BearerAuth
// @Param customer_id query string false "Customer ID (UUID)"
// @Param status query string false "Job status (scheduled, in_progress, on_hold, completed, cancelled)"
// @Param priority query string false "Job priority (low, normal, high, urgent)"
// @Param job_type query string false "Job type (installation, maintenance, repair, inspection, emergency)"
// @Param assigned_user_id query string false "Assigned user ID (UUID)"
// @Param scheduled_start_from query string false "Scheduled start from (RFC3339 format)"
// @Param scheduled_start_to query string false "Scheduled start to (RFC3339 format)"
// @Param requires_follow_up query boolean false "Requires follow-up"
// @Param sort_by query string false "Sort field (default: created_at)"
// @Param sort_order query string false "Sort order (ASC or DESC, default: DESC)"
// @Param limit query int false "Page size (default: 20)"
// @Param offset query int false "Offset (default: 0)"
// @Success 200 {object} models.JobListResponse
// @Failure 400 {object} models.ErrorResponse "Invalid request"
// @Failure 401 {object} models.ErrorResponse "Unauthorized"
// @Failure 500 {object} models.ErrorResponse "Internal server error"
// @Router /api/v1/jobs [get]
func (h *JobHandler) ListJobs(c *gin.Context) {
	// Get user info from context
	userID, userRole, err := h.getUserContext(c)
	if err != nil {
		return // response already sent
	}

	// Parse filter parameters
	var filter models.JobListFilter
	if err := c.ShouldBindQuery(&filter); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "invalid_request",
			Message: "Invalid query parameters: " + err.Error(),
		})
		return
	}

	// Set default pagination if not provided
	if filter.Limit == nil {
		defaultLimit := 20
		filter.Limit = &defaultLimit
	}
	if filter.Offset == nil {
		defaultOffset := 0
		filter.Offset = &defaultOffset
	}

	// Get jobs
	jobs, total, err := h.jobService.ListJobs(&filter, userID, userRole)
	if err != nil {
		log.Printf("Error listing jobs: %v", err)
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to retrieve jobs",
		})
		return
	}

	// Calculate pagination
	page := *filter.Offset / *filter.Limit
	totalPages := int(total) / *filter.Limit
	if int(total)%*filter.Limit > 0 {
		totalPages++
	}

	c.JSON(http.StatusOK, models.JobListResponse{
		Jobs:       jobs,
		Total:      total,
		Page:       page,
		PageSize:   *filter.Limit,
		TotalPages: totalPages,
	})
}

// GetMyJobs handles GET /api/v1/jobs/my
// @Summary Get my assigned jobs
// @Description Retrieve jobs assigned to the current user
// @Tags jobs
// @Produce json
// @Security BearerAuth
// @Success 200 {array} models.JobResponse
// @Failure 401 {object} models.ErrorResponse "Unauthorized"
// @Failure 500 {object} models.ErrorResponse "Internal server error"
// @Router /api/v1/jobs/my [get]
func (h *JobHandler) GetMyJobs(c *gin.Context) {
	// Get user ID from context
	userID, _, err := h.getUserContext(c)
	if err != nil {
		return // response already sent
	}

	// Get jobs
	jobs, err := h.jobService.GetMyJobs(userID)
	if err != nil {
		log.Printf("Error getting user jobs: %v", err)
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to retrieve jobs",
		})
		return
	}

	c.JSON(http.StatusOK, jobs)
}

// GetScheduledJobs handles GET /api/v1/jobs/scheduled
// @Summary Get scheduled jobs in date range
// @Description Retrieve jobs scheduled within a specific date range
// @Tags jobs
// @Produce json
// @Security BearerAuth
// @Param start query string true "Start date (RFC3339 format)"
// @Param end query string true "End date (RFC3339 format)"
// @Success 200 {array} models.JobResponse
// @Failure 400 {object} models.ErrorResponse "Invalid request"
// @Failure 401 {object} models.ErrorResponse "Unauthorized"
// @Failure 500 {object} models.ErrorResponse "Internal server error"
// @Router /api/v1/jobs/scheduled [get]
func (h *JobHandler) GetScheduledJobs(c *gin.Context) {
	startStr := c.Query("start")
	endStr := c.Query("end")

	if startStr == "" || endStr == "" {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "invalid_request",
			Message: "Start and end dates are required",
		})
		return
	}

	start, err := time.Parse(time.RFC3339, startStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "invalid_request",
			Message: "Invalid start date format. Use RFC3339 format (e.g., 2024-01-22T10:00:00Z)",
		})
		return
	}

	end, err := time.Parse(time.RFC3339, endStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "invalid_request",
			Message: "Invalid end date format. Use RFC3339 format (e.g., 2024-01-22T18:00:00Z)",
		})
		return
	}

	jobs, err := h.jobService.GetScheduledJobs(start, end)
	if err != nil {
		log.Printf("Error getting scheduled jobs: %v", err)
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to retrieve scheduled jobs",
		})
		return
	}

	c.JSON(http.StatusOK, jobs)
}

// GetOverdueJobs handles GET /api/v1/jobs/overdue
// @Summary Get overdue jobs
// @Description Retrieve jobs that are past their scheduled end date
// @Tags jobs
// @Produce json
// @Security BearerAuth
// @Success 200 {array} models.JobResponse
// @Failure 401 {object} models.ErrorResponse "Unauthorized"
// @Failure 500 {object} models.ErrorResponse "Internal server error"
// @Router /api/v1/jobs/overdue [get]
func (h *JobHandler) GetOverdueJobs(c *gin.Context) {
	jobs, err := h.jobService.GetOverdueJobs()
	if err != nil {
		log.Printf("Error getting overdue jobs: %v", err)
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to retrieve overdue jobs",
		})
		return
	}

	c.JSON(http.StatusOK, jobs)
}

// StartJob handles POST /api/v1/jobs/:id/start
// @Summary Start a job
// @Description Transition a job to in_progress status
// @Tags jobs
// @Produce json
// @Security BearerAuth
// @Param id path string true "Job ID (UUID)"
// @Success 200 {object} models.JobResponse
// @Failure 400 {object} models.ErrorResponse "Invalid request or job cannot be started"
// @Failure 401 {object} models.ErrorResponse "Unauthorized"
// @Failure 403 {object} models.ErrorResponse "Forbidden"
// @Failure 404 {object} models.ErrorResponse "Job not found"
// @Failure 500 {object} models.ErrorResponse "Internal server error"
// @Router /api/v1/jobs/{id}/start [post]
func (h *JobHandler) StartJob(c *gin.Context) {
	// Get user info from context
	userID, userRole, err := h.getUserContext(c)
	if err != nil {
		return // response already sent
	}

	// Parse job ID
	jobID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "invalid_id",
			Message: "Invalid job ID format",
		})
		return
	}

	// Start job
	job, err := h.jobService.StartJob(jobID, userID, userRole)
	if err != nil {
		switch {
		case errors.Is(err, services.ErrJobNotFound):
			c.JSON(http.StatusNotFound, models.ErrorResponse{
				Error:   "job_not_found",
				Message: "Job not found",
			})
		case errors.Is(err, services.ErrUnauthorized):
			c.JSON(http.StatusForbidden, models.ErrorResponse{
				Error:   "forbidden",
				Message: "You do not have permission to start this job",
			})
		default:
			log.Printf("Error starting job: %v", err)
			c.JSON(http.StatusBadRequest, models.ErrorResponse{
				Error:   "start_failed",
				Message: err.Error(),
			})
		}
		return
	}

	c.JSON(http.StatusOK, job)
}

// CompleteJob handles POST /api/v1/jobs/:id/complete
// @Summary Complete a job
// @Description Transition a job to completed status
// @Tags jobs
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Job ID (UUID)"
// @Param request body object{completion_notes=string} false "Completion notes"
// @Success 200 {object} models.JobResponse
// @Failure 400 {object} models.ErrorResponse "Invalid request or job cannot be completed"
// @Failure 401 {object} models.ErrorResponse "Unauthorized"
// @Failure 403 {object} models.ErrorResponse "Forbidden"
// @Failure 404 {object} models.ErrorResponse "Job not found"
// @Failure 500 {object} models.ErrorResponse "Internal server error"
// @Router /api/v1/jobs/{id}/complete [post]
func (h *JobHandler) CompleteJob(c *gin.Context) {
	// Get user info from context
	userID, userRole, err := h.getUserContext(c)
	if err != nil {
		return // response already sent
	}

	// Parse job ID
	jobID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "invalid_id",
			Message: "Invalid job ID format",
		})
		return
	}

	// Get completion notes from request body
	var req struct {
		CompletionNotes string `json:"completion_notes"`
	}
	_ = c.ShouldBindJSON(&req)

	// Complete job
	job, err := h.jobService.CompleteJob(jobID, req.CompletionNotes, userID, userRole)
	if err != nil {
		switch {
		case errors.Is(err, services.ErrJobNotFound):
			c.JSON(http.StatusNotFound, models.ErrorResponse{
				Error:   "job_not_found",
				Message: "Job not found",
			})
		case errors.Is(err, services.ErrUnauthorized):
			c.JSON(http.StatusForbidden, models.ErrorResponse{
				Error:   "forbidden",
				Message: "You do not have permission to complete this job",
			})
		default:
			log.Printf("Error completing job: %v", err)
			c.JSON(http.StatusBadRequest, models.ErrorResponse{
				Error:   "complete_failed",
				Message: err.Error(),
			})
		}
		return
	}

	c.JSON(http.StatusOK, job)
}

// CancelJob handles POST /api/v1/jobs/:id/cancel
// @Summary Cancel a job
// @Description Transition a job to cancelled status (admin/manager only)
// @Tags jobs
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Job ID (UUID)"
// @Param request body object{reason=string} false "Cancellation reason"
// @Success 200 {object} models.JobResponse
// @Failure 400 {object} models.ErrorResponse "Invalid request"
// @Failure 401 {object} models.ErrorResponse "Unauthorized"
// @Failure 403 {object} models.ErrorResponse "Forbidden"
// @Failure 404 {object} models.ErrorResponse "Job not found"
// @Failure 500 {object} models.ErrorResponse "Internal server error"
// @Router /api/v1/jobs/{id}/cancel [post]
func (h *JobHandler) CancelJob(c *gin.Context) {
	// Get user info from context
	userID, userRole, err := h.getUserContext(c)
	if err != nil {
		return // response already sent
	}

	// Parse job ID
	jobID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "invalid_id",
			Message: "Invalid job ID format",
		})
		return
	}

	// Get cancellation reason from request body
	var req struct {
		Reason string `json:"reason"`
	}
	_ = c.ShouldBindJSON(&req)

	// Cancel job
	job, err := h.jobService.CancelJob(jobID, req.Reason, userID, userRole)
	if err != nil {
		switch {
		case errors.Is(err, services.ErrJobNotFound):
			c.JSON(http.StatusNotFound, models.ErrorResponse{
				Error:   "job_not_found",
				Message: "Job not found",
			})
		case errors.Is(err, services.ErrUnauthorized):
			c.JSON(http.StatusForbidden, models.ErrorResponse{
				Error:   "forbidden",
				Message: "Only admins and managers can cancel jobs",
			})
		default:
			log.Printf("Error cancelling job: %v", err)
			c.JSON(http.StatusBadRequest, models.ErrorResponse{
				Error:   "cancel_failed",
				Message: err.Error(),
			})
		}
		return
	}

	c.JSON(http.StatusOK, job)
}

// PutOnHold handles POST /api/v1/jobs/:id/hold
// @Summary Put a job on hold
// @Description Transition a job to on_hold status
// @Tags jobs
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Job ID (UUID)"
// @Param request body object{reason=string} false "On-hold reason"
// @Success 200 {object} models.JobResponse
// @Failure 400 {object} models.ErrorResponse "Invalid request"
// @Failure 401 {object} models.ErrorResponse "Unauthorized"
// @Failure 403 {object} models.ErrorResponse "Forbidden"
// @Failure 404 {object} models.ErrorResponse "Job not found"
// @Failure 500 {object} models.ErrorResponse "Internal server error"
// @Router /api/v1/jobs/{id}/hold [post]
func (h *JobHandler) PutOnHold(c *gin.Context) {
	// Get user info from context
	userID, userRole, err := h.getUserContext(c)
	if err != nil {
		return // response already sent
	}

	// Parse job ID
	jobID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "invalid_id",
			Message: "Invalid job ID format",
		})
		return
	}

	// Get reason from request body
	var req struct {
		Reason string `json:"reason"`
	}
	_ = c.ShouldBindJSON(&req)

	// Put job on hold
	job, err := h.jobService.PutOnHold(jobID, req.Reason, userID, userRole)
	if err != nil {
		switch {
		case errors.Is(err, services.ErrJobNotFound):
			c.JSON(http.StatusNotFound, models.ErrorResponse{
				Error:   "job_not_found",
				Message: "Job not found",
			})
		case errors.Is(err, services.ErrUnauthorized):
			c.JSON(http.StatusForbidden, models.ErrorResponse{
				Error:   "forbidden",
				Message: "You do not have permission to put this job on hold",
			})
		default:
			log.Printf("Error putting job on hold: %v", err)
			c.JSON(http.StatusBadRequest, models.ErrorResponse{
				Error:   "hold_failed",
				Message: err.Error(),
			})
		}
		return
	}

	c.JSON(http.StatusOK, job)
}

// ResumeJob handles POST /api/v1/jobs/:id/resume
// @Summary Resume a job from on_hold
// @Description Transition a job from on_hold to in_progress
// @Tags jobs
// @Produce json
// @Security BearerAuth
// @Param id path string true "Job ID (UUID)"
// @Success 200 {object} models.JobResponse
// @Failure 400 {object} models.ErrorResponse "Invalid request"
// @Failure 401 {object} models.ErrorResponse "Unauthorized"
// @Failure 403 {object} models.ErrorResponse "Forbidden"
// @Failure 404 {object} models.ErrorResponse "Job not found"
// @Failure 500 {object} models.ErrorResponse "Internal server error"
// @Router /api/v1/jobs/{id}/resume [post]
func (h *JobHandler) ResumeJob(c *gin.Context) {
	// Get user info from context
	userID, userRole, err := h.getUserContext(c)
	if err != nil {
		return // response already sent
	}

	// Parse job ID
	jobID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "invalid_id",
			Message: "Invalid job ID format",
		})
		return
	}

	// Resume job
	job, err := h.jobService.ResumeJob(jobID, userID, userRole)
	if err != nil {
		switch {
		case errors.Is(err, services.ErrJobNotFound):
			c.JSON(http.StatusNotFound, models.ErrorResponse{
				Error:   "job_not_found",
				Message: "Job not found",
			})
		case errors.Is(err, services.ErrUnauthorized):
			c.JSON(http.StatusForbidden, models.ErrorResponse{
				Error:   "forbidden",
				Message: "You do not have permission to resume this job",
			})
		default:
			log.Printf("Error resuming job: %v", err)
			c.JSON(http.StatusBadRequest, models.ErrorResponse{
				Error:   "resume_failed",
				Message: err.Error(),
			})
		}
		return
	}

	c.JSON(http.StatusOK, job)
}

// AssignTechnician handles POST /api/v1/jobs/:id/assign
// @Summary Assign a technician to a job
// @Description Assign a user to a job (admin/manager only)
// @Tags jobs
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Job ID (UUID)"
// @Param request body object{technician_id=string,role=string} true "Assignment details"
// @Success 200 {object} object{message=string}
// @Failure 400 {object} models.ErrorResponse "Invalid request"
// @Failure 401 {object} models.ErrorResponse "Unauthorized"
// @Failure 403 {object} models.ErrorResponse "Forbidden"
// @Failure 404 {object} models.ErrorResponse "Job not found"
// @Failure 500 {object} models.ErrorResponse "Internal server error"
// @Router /api/v1/jobs/{id}/assign [post]
func (h *JobHandler) AssignTechnician(c *gin.Context) {
	// Get user info from context
	userID, userRole, err := h.getUserContext(c)
	if err != nil {
		return // response already sent
	}

	// Parse job ID
	jobID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "invalid_id",
			Message: "Invalid job ID format",
		})
		return
	}

	// Parse request body
	var req struct {
		TechnicianID string `json:"technician_id" binding:"required"`
		Role         string `json:"role"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "invalid_request",
			Message: "Invalid request: " + err.Error(),
		})
		return
	}

	technicianID, err := uuid.Parse(req.TechnicianID)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "invalid_request",
			Message: "Invalid technician ID format",
		})
		return
	}

	// Assign technician
	err = h.jobService.AssignTechnician(jobID, technicianID, req.Role, userID, userRole)
	if err != nil {
		switch {
		case errors.Is(err, services.ErrJobNotFound):
			c.JSON(http.StatusNotFound, models.ErrorResponse{
				Error:   "job_not_found",
				Message: "Job not found",
			})
		case errors.Is(err, services.ErrUnauthorized):
			c.JSON(http.StatusForbidden, models.ErrorResponse{
				Error:   "forbidden",
				Message: "Only admins and managers can assign technicians",
			})
		default:
			log.Printf("Error assigning technician: %v", err)
			c.JSON(http.StatusBadRequest, models.ErrorResponse{
				Error:   "assignment_failed",
				Message: err.Error(),
			})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Technician assigned successfully",
	})
}

// UnassignTechnician handles DELETE /api/v1/jobs/:id/assign/:technicianId
// @Summary Unassign a technician from a job
// @Description Remove a technician assignment from a job (admin/manager only)
// @Tags jobs
// @Produce json
// @Security BearerAuth
// @Param id path string true "Job ID (UUID)"
// @Param technicianId path string true "Technician ID (UUID)"
// @Success 200 {object} object{message=string}
// @Failure 400 {object} models.ErrorResponse "Invalid request"
// @Failure 401 {object} models.ErrorResponse "Unauthorized"
// @Failure 403 {object} models.ErrorResponse "Forbidden"
// @Failure 404 {object} models.ErrorResponse "Job or assignment not found"
// @Failure 500 {object} models.ErrorResponse "Internal server error"
// @Router /api/v1/jobs/{id}/assign/{technicianId} [delete]
func (h *JobHandler) UnassignTechnician(c *gin.Context) {
	// Get user info from context
	userID, userRole, err := h.getUserContext(c)
	if err != nil {
		return // response already sent
	}

	// Parse job ID
	jobID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "invalid_id",
			Message: "Invalid job ID format",
		})
		return
	}

	// Parse technician ID
	technicianID, err := uuid.Parse(c.Param("technicianId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "invalid_id",
			Message: "Invalid technician ID format",
		})
		return
	}

	// Unassign technician
	err = h.jobService.UnassignTechnician(jobID, technicianID, userID, userRole)
	if err != nil {
		switch {
		case errors.Is(err, services.ErrJobNotFound):
			c.JSON(http.StatusNotFound, models.ErrorResponse{
				Error:   "job_not_found",
				Message: "Job not found",
			})
		case errors.Is(err, services.ErrUnauthorized):
			c.JSON(http.StatusForbidden, models.ErrorResponse{
				Error:   "forbidden",
				Message: "Only admins and managers can unassign technicians",
			})
		default:
			log.Printf("Error unassigning technician: %v", err)
			c.JSON(http.StatusBadRequest, models.ErrorResponse{
				Error:   "unassignment_failed",
				Message: err.Error(),
			})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Technician unassigned successfully",
	})
}

// AddMaterial handles POST /api/v1/jobs/:id/materials
// @Summary Add a material to a job
// @Description Add a material/part to a job
// @Tags jobs
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Job ID (UUID)"
// @Param request body models.JobMaterial true "Material details"
// @Success 201 {object} object{message=string}
// @Failure 400 {object} models.ErrorResponse "Invalid request"
// @Failure 401 {object} models.ErrorResponse "Unauthorized"
// @Failure 403 {object} models.ErrorResponse "Forbidden"
// @Failure 404 {object} models.ErrorResponse "Job not found"
// @Failure 500 {object} models.ErrorResponse "Internal server error"
// @Router /api/v1/jobs/{id}/materials [post]
func (h *JobHandler) AddMaterial(c *gin.Context) {
	// Get user info from context
	userID, userRole, err := h.getUserContext(c)
	if err != nil {
		return // response already sent
	}

	// Parse job ID
	jobID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "invalid_id",
			Message: "Invalid job ID format",
		})
		return
	}

	var material models.JobMaterial
	if err := c.ShouldBindJSON(&material); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "invalid_request",
			Message: "Invalid material data: " + err.Error(),
		})
		return
	}

	// Add material
	err = h.jobService.AddMaterial(jobID, &material, userID, userRole)
	if err != nil {
		switch {
		case errors.Is(err, services.ErrJobNotFound):
			c.JSON(http.StatusNotFound, models.ErrorResponse{
				Error:   "job_not_found",
				Message: "Job not found",
			})
		case errors.Is(err, services.ErrUnauthorized):
			c.JSON(http.StatusForbidden, models.ErrorResponse{
				Error:   "forbidden",
				Message: "You do not have permission to add materials to this job",
			})
		default:
			log.Printf("Error adding material: %v", err)
			c.JSON(http.StatusBadRequest, models.ErrorResponse{
				Error:   "add_material_failed",
				Message: err.Error(),
			})
		}
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "Material added successfully",
	})
}

// AddNote handles POST /api/v1/jobs/:id/notes
// @Summary Add a note to a job
// @Description Add a timestamped note to a job
// @Tags jobs
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Job ID (UUID)"
// @Param request body object{note=string,is_internal=boolean} true "Note details"
// @Success 201 {object} object{message=string}
// @Failure 400 {object} models.ErrorResponse "Invalid request"
// @Failure 401 {object} models.ErrorResponse "Unauthorized"
// @Failure 404 {object} models.ErrorResponse "Job not found"
// @Failure 500 {object} models.ErrorResponse "Internal server error"
// @Router /api/v1/jobs/{id}/notes [post]
func (h *JobHandler) AddNote(c *gin.Context) {
	// Get user info from context
	userID, userRole, err := h.getUserContext(c)
	if err != nil {
		return // response already sent
	}

	// Parse job ID
	jobID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "invalid_id",
			Message: "Invalid job ID format",
		})
		return
	}

	var req struct {
		Note       string `json:"note" binding:"required"`
		IsInternal bool   `json:"is_internal"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "invalid_request",
			Message: "Invalid note data: " + err.Error(),
		})
		return
	}

	// Add note
	err = h.jobService.AddNote(jobID, userID, req.Note, req.IsInternal, userRole)
	if err != nil {
		switch {
		case errors.Is(err, services.ErrJobNotFound):
			c.JSON(http.StatusNotFound, models.ErrorResponse{
				Error:   "job_not_found",
				Message: "Job not found",
			})
		default:
			log.Printf("Error adding note: %v", err)
			c.JSON(http.StatusBadRequest, models.ErrorResponse{
				Error:   "add_note_failed",
				Message: err.Error(),
			})
		}
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "Note added successfully",
	})
}

// GetJobStats handles GET /api/v1/jobs/stats
// @Summary Get job statistics
// @Description Get statistics about jobs (counts by status, priority, type, etc.)
// @Tags jobs
// @Produce json
// @Security BearerAuth
// @Success 200 {object} map[string]interface{}
// @Failure 401 {object} models.ErrorResponse "Unauthorized"
// @Failure 500 {object} models.ErrorResponse "Internal server error"
// @Router /api/v1/jobs/stats [get]
func (h *JobHandler) GetJobStats(c *gin.Context) {
	stats, err := h.jobService.GetJobStats()
	if err != nil {
		log.Printf("Error getting job stats: %v", err)
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to retrieve statistics",
		})
		return
	}

	c.JSON(http.StatusOK, stats)
}

// GetTechnicianWorkload handles GET /api/v1/jobs/technicians/:id/workload
// @Summary Get technician workload
// @Description Get workload statistics for a specific technician
// @Tags jobs
// @Produce json
// @Security BearerAuth
// @Param id path string true "Technician ID (UUID)"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} models.ErrorResponse "Invalid ID"
// @Failure 401 {object} models.ErrorResponse "Unauthorized"
// @Failure 500 {object} models.ErrorResponse "Internal server error"
// @Router /api/v1/jobs/technicians/{id}/workload [get]
func (h *JobHandler) GetTechnicianWorkload(c *gin.Context) {
	// Parse technician ID
	technicianID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "invalid_id",
			Message: "Invalid technician ID format",
		})
		return
	}

	workload, err := h.jobService.GetTechnicianWorkload(technicianID)
	if err != nil {
		log.Printf("Error getting technician workload: %v", err)
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to retrieve workload statistics",
		})
		return
	}

	c.JSON(http.StatusOK, workload)
}

// GetCustomerJobs handles GET /api/v1/customers/:id/jobs
// @Summary Get jobs for a customer
// @Description Retrieve all jobs for a specific customer
// @Tags jobs
// @Produce json
// @Security BearerAuth
// @Param id path string true "Customer ID (UUID)"
// @Param limit query int false "Limit (default: 20)"
// @Param offset query int false "Offset (default: 0)"
// @Success 200 {object} models.JobListResponse
// @Failure 400 {object} models.ErrorResponse "Invalid request"
// @Failure 401 {object} models.ErrorResponse "Unauthorized"
// @Failure 500 {object} models.ErrorResponse "Internal server error"
// @Router /api/v1/customers/{id}/jobs [get]
func (h *JobHandler) GetCustomerJobs(c *gin.Context) {
	// Parse customer ID
	customerID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "invalid_id",
			Message: "Invalid customer ID format",
		})
		return
	}

	// Parse pagination params
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	// Get jobs
	jobs, total, err := h.jobService.GetCustomerJobs(customerID, limit, offset)
	if err != nil {
		log.Printf("Error getting customer jobs: %v", err)
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to retrieve jobs",
		})
		return
	}

	// Calculate pagination
	page := offset / limit
	totalPages := int(total) / limit
	if int(total)%limit > 0 {
		totalPages++
	}

	c.JSON(http.StatusOK, models.JobListResponse{
		Jobs:       jobs,
		Total:      total,
		Page:       page,
		PageSize:   limit,
		TotalPages: totalPages,
	})
}

// TransitionStatus handles POST /api/v1/jobs/:id/transition
// @Summary Transition job status
// @Description Transition a job to a new status
// @Tags jobs
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Job ID (UUID)"
// @Param request body models.StatusChangeRequest true "Status transition request"
// @Success 200 {object} models.JobResponse
// @Failure 400 {object} models.ErrorResponse "Invalid request or invalid transition"
// @Failure 401 {object} models.ErrorResponse "Unauthorized"
// @Failure 403 {object} models.ErrorResponse "Forbidden"
// @Failure 404 {object} models.ErrorResponse "Job not found"
// @Failure 500 {object} models.ErrorResponse "Internal server error"
// @Router /api/v1/jobs/{id}/transition [post]
func (h *JobHandler) TransitionStatus(c *gin.Context) {
	// Get user info from context
	userID, userRole, err := h.getUserContext(c)
	if err != nil {
		return // response already sent
	}

	// Parse job ID
	jobID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "invalid_id",
			Message: "Invalid job ID format",
		})
		return
	}

	// Parse request body
	var req models.StatusChangeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "invalid_request",
			Message: "Invalid request: " + err.Error(),
		})
		return
	}

	// Transition status
	job, err := h.jobService.TransitionStatus(jobID, req.ToStatus, string(req.Reason), req.Notes, userID, userRole)
	if err != nil {
		switch {
		case errors.Is(err, services.ErrJobNotFound):
			c.JSON(http.StatusNotFound, models.ErrorResponse{
				Error:   "job_not_found",
				Message: "Job not found",
			})
		case errors.Is(err, services.ErrUnauthorized):
			c.JSON(http.StatusForbidden, models.ErrorResponse{
				Error:   "forbidden",
				Message: "You do not have permission to transition this job",
			})
		default:
			log.Printf("Error transitioning job status: %v", err)
			c.JSON(http.StatusBadRequest, models.ErrorResponse{
				Error:   "transition_failed",
				Message: err.Error(),
			})
		}
		return
	}

	c.JSON(http.StatusOK, job)
}

// GetStatusHistory handles GET /api/v1/jobs/:id/status-history
// @Summary Get job status history
// @Description Retrieve the status transition history for a job
// @Tags jobs
// @Produce json
// @Security BearerAuth
// @Param id path string true "Job ID (UUID)"
// @Param limit query int false "Limit (default: 20)"
// @Param offset query int false "Offset (default: 0)"
// @Param sort query string false "Sort order: 'asc' or 'desc' (default: 'desc')"
// @Success 200 {object} models.StatusHistoryResponse
// @Failure 400 {object} models.ErrorResponse "Invalid request"
// @Failure 401 {object} models.ErrorResponse "Unauthorized"
// @Failure 500 {object} models.ErrorResponse "Internal server error"
// @Router /api/v1/jobs/{id}/status-history [get]
func (h *JobHandler) GetStatusHistory(c *gin.Context) {
	// Parse job ID
	jobID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "invalid_id",
			Message: "Invalid job ID format",
		})
		return
	}

	// Parse query params
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))
	sortOrder := c.DefaultQuery("sort", "desc")

	// Get status history
	history, err := h.jobService.GetStatusHistory(jobID, limit, offset, sortOrder)
	if err != nil {
		log.Printf("Error getting status history: %v", err)
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to retrieve status history",
		})
		return
	}

	c.JSON(http.StatusOK, history)
}
