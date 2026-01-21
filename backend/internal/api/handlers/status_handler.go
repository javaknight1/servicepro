package handlers

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/javaknight1/servicepro/backend/internal/models"
	"github.com/javaknight1/servicepro/backend/internal/services"
)

// StatusHandler handles job status-related HTTP requests
type StatusHandler struct {
	statusService services.StatusServiceInterface
}

// NewStatusHandler creates a new status handler
func NewStatusHandler(statusService services.StatusServiceInterface) *StatusHandler {
	return &StatusHandler{
		statusService: statusService,
	}
}

// ChangeStatus handles POST /api/v1/jobs/:id/status
// @Summary Change job status
// @Description Changes the status of a job with validation and state machine rules
// @Tags job-status
// @Accept json
// @Produce json
// @Param id path string true "Job ID (UUID)"
// @Param request body models.StatusChangeRequest true "Status change request"
// @Success 200 {object} models.StatusChangeResponse
// @Failure 400 {object} models.ErrorResponse "Invalid request or validation error"
// @Failure 401 {object} models.ErrorResponse "Unauthorized"
// @Failure 403 {object} models.ErrorResponse "Forbidden"
// @Failure 404 {object} models.ErrorResponse "Job not found"
// @Failure 409 {object} models.ErrorResponse "Invalid status transition"
// @Failure 500 {object} models.ErrorResponse "Internal server error"
// @Security BearerAuth
// @Router /api/v1/jobs/{id}/status [post]
func (h *StatusHandler) ChangeStatus(c *gin.Context) {
	// Get authenticated user
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, models.ErrorResponse{
			Error:   "unauthorized",
			Message: "User not authenticated",
		})
		return
	}

	userRole, _ := c.Get("userRole")

	// Only admins, managers, and technicians can change status
	if userRole != models.UserRoleAdmin &&
		userRole != models.UserRoleManager &&
		userRole != models.UserRoleTechnician {
		c.JSON(http.StatusForbidden, models.ErrorResponse{
			Error:   "forbidden",
			Message: "Insufficient permissions to change job status",
		})
		return
	}

	// Parse job ID from path
	jobIDStr := c.Param("id")
	jobID, err := uuid.Parse(jobIDStr)
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
			Message: "Invalid request data: " + err.Error(),
		})
		return
	}

	// Set job ID from path parameter
	req.JobID = jobID

	// Create context with timeout
	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	// Change status
	response, err := h.statusService.ChangeStatus(ctx, &req, userID.(uuid.UUID))
	if err != nil {
		switch {
		case errors.Is(err, services.ErrJobNotFound):
			c.JSON(http.StatusNotFound, models.ErrorResponse{
				Error:   "job_not_found",
				Message: "Job not found",
			})
		case errors.Is(err, models.ErrInvalidStatusTransition):
			c.JSON(http.StatusConflict, models.ErrorResponse{
				Error:   "invalid_transition",
				Message: err.Error(),
			})
		case errors.Is(err, models.ErrStatusValidation):
			c.JSON(http.StatusBadRequest, models.ErrorResponse{
				Error:   "validation_error",
				Message: err.Error(),
			})
		case errors.Is(err, services.ErrJobLocked):
			c.JSON(http.StatusConflict, models.ErrorResponse{
				Error:   "job_locked",
				Message: "Job is locked by another operation, please try again",
			})
		default:
			c.JSON(http.StatusInternalServerError, models.ErrorResponse{
				Error:   "internal_error",
				Message: "Failed to change job status",
			})
		}
		return
	}

	c.JSON(http.StatusOK, response)
}

// GetStatusHistory handles GET /api/v1/jobs/:id/status/history
// @Summary Get job status history
// @Description Retrieves the status change history for a job
// @Tags job-status
// @Produce json
// @Param id path string true "Job ID (UUID)"
// @Param limit query int false "Limit (default: 50)"
// @Param offset query int false "Offset (default: 0)"
// @Success 200 {object} models.StatusHistoryResponse
// @Failure 400 {object} models.ErrorResponse
// @Failure 401 {object} models.ErrorResponse
// @Failure 404 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Security BearerAuth
// @Router /api/v1/jobs/{id}/status/history [get]
func (h *StatusHandler) GetStatusHistory(c *gin.Context) {
	// Check authentication
	_, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, models.ErrorResponse{
			Error:   "unauthorized",
			Message: "User not authenticated",
		})
		return
	}

	// Parse job ID
	jobIDStr := c.Param("id")
	jobID, err := uuid.Parse(jobIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "invalid_id",
			Message: "Invalid job ID format",
		})
		return
	}

	// Parse pagination parameters
	limit := 50
	offset := 0

	if limitStr := c.Query("limit"); limitStr != "" {
		if parsedLimit, err := strconv.Atoi(limitStr); err == nil && parsedLimit > 0 {
			limit = parsedLimit
		}
	}

	if offsetStr := c.Query("offset"); offsetStr != "" {
		if parsedOffset, err := strconv.Atoi(offsetStr); err == nil && parsedOffset >= 0 {
			offset = parsedOffset
		}
	}

	// Create context with timeout
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	// Get history
	history, err := h.statusService.GetStatusHistory(ctx, jobID, limit, offset)
	if err != nil {
		if errors.Is(err, services.ErrJobNotFound) {
			c.JSON(http.StatusNotFound, models.ErrorResponse{
				Error:   "job_not_found",
				Message: "Job not found",
			})
			return
		}

		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to retrieve status history",
		})
		return
	}

	c.JSON(http.StatusOK, history)
}

// GetAllowedTransitions handles GET /api/v1/jobs/:id/status/transitions
// @Summary Get allowed status transitions
// @Description Returns the allowed status transitions for a job based on its current status
// @Tags job-status
// @Produce json
// @Param id path string true "Job ID (UUID)"
// @Success 200 {object} models.AllowedTransitionsResponse
// @Failure 400 {object} models.ErrorResponse
// @Failure 401 {object} models.ErrorResponse
// @Failure 404 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Security BearerAuth
// @Router /api/v1/jobs/{id}/status/transitions [get]
func (h *StatusHandler) GetAllowedTransitions(c *gin.Context) {
	// Check authentication
	_, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, models.ErrorResponse{
			Error:   "unauthorized",
			Message: "User not authenticated",
		})
		return
	}

	// Parse job ID
	jobIDStr := c.Param("id")
	jobID, err := uuid.Parse(jobIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "invalid_id",
			Message: "Invalid job ID format",
		})
		return
	}

	// Create context with timeout
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	// Get allowed transitions
	transitions, err := h.statusService.GetAllowedTransitions(ctx, jobID)
	if err != nil {
		if errors.Is(err, services.ErrJobNotFound) {
			c.JSON(http.StatusNotFound, models.ErrorResponse{
				Error:   "job_not_found",
				Message: "Job not found",
			})
			return
		}

		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to get allowed transitions",
		})
		return
	}

	c.JSON(http.StatusOK, transitions)
}

// ValidateStatusChange handles POST /api/v1/jobs/:id/status/validate
// @Summary Validate status change
// @Description Validates if a status change is allowed without actually applying it
// @Tags job-status
// @Accept json
// @Produce json
// @Param id path string true "Job ID (UUID)"
// @Param request body object{to_status=string} true "Status to validate"
// @Success 200 {object} object{valid=bool,message=string}
// @Failure 400 {object} models.ErrorResponse
// @Failure 401 {object} models.ErrorResponse
// @Failure 404 {object} models.ErrorResponse
// @Failure 409 {object} models.ErrorResponse
// @Security BearerAuth
// @Router /api/v1/jobs/{id}/status/validate [post]
func (h *StatusHandler) ValidateStatusChange(c *gin.Context) {
	// Check authentication
	_, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, models.ErrorResponse{
			Error:   "unauthorized",
			Message: "User not authenticated",
		})
		return
	}

	// Parse job ID
	jobIDStr := c.Param("id")
	jobID, err := uuid.Parse(jobIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "invalid_id",
			Message: "Invalid job ID format",
		})
		return
	}

	// Parse request
	var req struct {
		ToStatus models.JobStatus `json:"to_status" binding:"required,oneof=scheduled in_progress on_hold completed cancelled"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "invalid_request",
			Message: "Invalid request data: " + err.Error(),
		})
		return
	}

	// Create context with timeout
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	// Validate
	err = h.statusService.ValidateStatusChange(ctx, jobID, req.ToStatus)
	if err != nil {
		if errors.Is(err, services.ErrJobNotFound) {
			c.JSON(http.StatusNotFound, models.ErrorResponse{
				Error:   "job_not_found",
				Message: "Job not found",
			})
			return
		}

		// Return validation result
		c.JSON(http.StatusConflict, gin.H{
			"valid":   false,
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"valid":   true,
		"message": "Status change is valid",
	})
}

// BulkChangeStatus handles POST /api/v1/jobs/status/bulk
// @Summary Bulk change job status
// @Description Changes status for multiple jobs concurrently
// @Tags job-status
// @Accept json
// @Produce json
// @Param requests body []models.StatusChangeRequest true "Array of status change requests"
// @Success 200 {object} object{succeeded=[]models.StatusChangeResponse,failed=[]string,total=int,success_count=int,failure_count=int}
// @Failure 400 {object} models.ErrorResponse
// @Failure 401 {object} models.ErrorResponse
// @Failure 403 {object} models.ErrorResponse
// @Security BearerAuth
// @Router /api/v1/jobs/status/bulk [post]
func (h *StatusHandler) BulkChangeStatus(c *gin.Context) {
	// Get authenticated user
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, models.ErrorResponse{
			Error:   "unauthorized",
			Message: "User not authenticated",
		})
		return
	}

	userRole, _ := c.Get("userRole")

	// Only admins and managers can bulk change status
	if userRole != models.UserRoleAdmin && userRole != models.UserRoleManager {
		c.JSON(http.StatusForbidden, models.ErrorResponse{
			Error:   "forbidden",
			Message: "Only administrators and managers can perform bulk status changes",
		})
		return
	}

	// Parse request body
	var requests []*models.StatusChangeRequest
	if err := c.ShouldBindJSON(&requests); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "invalid_request",
			Message: "Invalid request data: " + err.Error(),
		})
		return
	}

	if len(requests) == 0 {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "invalid_request",
			Message: "At least one status change request is required",
		})
		return
	}

	// Create context with timeout (longer for bulk operations)
	ctx, cancel := context.WithTimeout(c.Request.Context(), 30*time.Second)
	defer cancel()

	// Bulk change status
	responses, errs := h.statusService.BulkChangeStatus(ctx, requests, userID.(uuid.UUID))

	// Convert errors to strings
	errorStrings := make([]string, len(errs))
	for i, err := range errs {
		errorStrings[i] = err.Error()
	}

	c.JSON(http.StatusOK, gin.H{
		"succeeded":     responses,
		"failed":        errorStrings,
		"total":         len(requests),
		"success_count": len(responses),
		"failure_count": len(errs),
	})
}

// GetStatusStatistics handles GET /api/v1/jobs/status/statistics
// @Summary Get status statistics
// @Description Returns statistics about job statuses and transitions
// @Tags job-status
// @Produce json
// @Param from_date query string false "From date (RFC3339)"
// @Param to_date query string false "To date (RFC3339)"
// @Success 200 {object} object{status_counts=object,transition_metrics=object}
// @Failure 401 {object} models.ErrorResponse
// @Failure 403 {object} models.ErrorResponse
// @Security BearerAuth
// @Router /api/v1/jobs/status/statistics [get]
func (h *StatusHandler) GetStatusStatistics(c *gin.Context) {
	// Check authentication
	_, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, models.ErrorResponse{
			Error:   "unauthorized",
			Message: "User not authenticated",
		})
		return
	}

	userRole, _ := c.Get("userRole")

	// Only admins and managers can view statistics
	if userRole != models.UserRoleAdmin && userRole != models.UserRoleManager {
		c.JSON(http.StatusForbidden, models.ErrorResponse{
			Error:   "forbidden",
			Message: "Only administrators and managers can view statistics",
		})
		return
	}

	// Note: This is a placeholder for status statistics
	// In a real implementation, you would query the database for counts
	// and use the transition metrics from the service

	c.JSON(http.StatusOK, gin.H{
		"status_counts": gin.H{
			"scheduled":   0,
			"in_progress": 0,
			"on_hold":     0,
			"completed":   0,
			"cancelled":   0,
		},
		"transition_metrics": gin.H{
			"message": "Transition metrics are tracked in the service layer",
		},
	})
}
