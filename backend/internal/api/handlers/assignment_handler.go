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
	"github.com/javaknight1/servicepro/backend/pkg/notifications"
)

// AssignmentHandler handles assignment-related HTTP requests
type AssignmentHandler struct {
	assignmentService   services.AssignmentServiceInterface
	notificationService notifications.NotificationServiceInterface
	jobService          services.JobServiceInterface
}

// NewAssignmentHandler creates a new assignment handler
func NewAssignmentHandler(
	assignmentService services.AssignmentServiceInterface,
	notificationService notifications.NotificationServiceInterface,
	jobService services.JobServiceInterface,
) *AssignmentHandler {
	return &AssignmentHandler{
		assignmentService:   assignmentService,
		notificationService: notificationService,
		jobService:          jobService,
	}
}

// CreateAssignment handles POST /api/v1/assignments
// @Summary Create a new assignment
// @Description Assigns a technician to a job with availability and skill validation
// @Tags assignments
// @Accept json
// @Produce json
// @Param assignment body services.AssignmentRequest true "Assignment request"
// @Success 201 {object} models.JobAssignment
// @Failure 400 {object} models.ErrorResponse
// @Failure 401 {object} models.ErrorResponse
// @Failure 404 {object} models.ErrorResponse
// @Failure 409 {object} models.ErrorResponse "Conflict - technician not available or skill mismatch"
// @Failure 500 {object} models.ErrorResponse
// @Security BearerAuth
// @Router /api/v1/assignments [post]
func (h *AssignmentHandler) CreateAssignment(c *gin.Context) {
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

	// Only admins and managers can create assignments
	if userRole != models.UserRoleAdmin && userRole != models.UserRoleManager {
		c.JSON(http.StatusForbidden, models.ErrorResponse{
			Error:   "forbidden",
			Message: "Only administrators and managers can create assignments",
		})
		return
	}

	var req services.AssignmentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "invalid_request",
			Message: "Invalid request data: " + err.Error(),
		})
		return
	}

	// Create assignment with context
	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	assignment, err := h.assignmentService.CreateAssignment(ctx, &req, userID.(uuid.UUID))
	if err != nil {
		switch {
		case errors.Is(err, services.ErrJobNotFound):
			c.JSON(http.StatusNotFound, models.ErrorResponse{
				Error:   "job_not_found",
				Message: "The specified job does not exist",
			})
		case errors.Is(err, services.ErrTechnicianNotFound):
			c.JSON(http.StatusNotFound, models.ErrorResponse{
				Error:   "technician_not_found",
				Message: "The specified technician does not exist",
			})
		case errors.Is(err, services.ErrTechnicianNotAvailable):
			c.JSON(http.StatusConflict, models.ErrorResponse{
				Error:   "technician_not_available",
				Message: "Technician is not available for the requested time slot",
			})
		case errors.Is(err, services.ErrSkillMismatch):
			c.JSON(http.StatusConflict, models.ErrorResponse{
				Error:   "skill_mismatch",
				Message: "Technician does not have sufficient skills for this job",
			})
		default:
			c.JSON(http.StatusInternalServerError, models.ErrorResponse{
				Error:   "internal_error",
				Message: "Failed to create assignment",
			})
		}
		return
	}

	// Send notification asynchronously (don't block response)
	go h.sendAssignmentNotification(assignment)

	c.JSON(http.StatusCreated, assignment)
}

// BulkCreateAssignments handles POST /api/v1/assignments/bulk
// @Summary Create multiple assignments concurrently
// @Description Assigns multiple technicians to a job concurrently
// @Tags assignments
// @Accept json
// @Produce json
// @Param assignments body services.BulkAssignmentRequest true "Bulk assignment request"
// @Success 200 {object} object{assignments=[]models.JobAssignment,errors=[]string}
// @Failure 400 {object} models.ErrorResponse
// @Failure 401 {object} models.ErrorResponse
// @Failure 403 {object} models.ErrorResponse
// @Security BearerAuth
// @Router /api/v1/assignments/bulk [post]
func (h *AssignmentHandler) BulkCreateAssignments(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, models.ErrorResponse{
			Error:   "unauthorized",
			Message: "User not authenticated",
		})
		return
	}

	userRole, _ := c.Get("userRole")
	if userRole != models.UserRoleAdmin && userRole != models.UserRoleManager {
		c.JSON(http.StatusForbidden, models.ErrorResponse{
			Error:   "forbidden",
			Message: "Only administrators and managers can create assignments",
		})
		return
	}

	var req services.BulkAssignmentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "invalid_request",
			Message: "Invalid request data: " + err.Error(),
		})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 30*time.Second)
	defer cancel()

	assignments, errs := h.assignmentService.BulkAssignTechnicians(ctx, &req, userID.(uuid.UUID))

	// Send notifications asynchronously
	for i := range assignments {
		go h.sendAssignmentNotification(&assignments[i])
	}

	// Convert errors to strings
	errorStrings := make([]string, len(errs))
	for i, err := range errs {
		errorStrings[i] = err.Error()
	}

	c.JSON(http.StatusOK, gin.H{
		"assignments": assignments,
		"errors":      errorStrings,
		"total":       len(req.Assignments),
		"succeeded":   len(assignments),
		"failed":      len(errs),
	})
}

// RemoveAssignment handles DELETE /api/v1/assignments/:id
// @Summary Remove an assignment
// @Description Removes a technician from a job assignment
// @Tags assignments
// @Produce json
// @Param id path string true "Assignment ID (UUID)"
// @Success 204 "No Content"
// @Failure 400 {object} models.ErrorResponse
// @Failure 401 {object} models.ErrorResponse
// @Failure 403 {object} models.ErrorResponse
// @Failure 404 {object} models.ErrorResponse
// @Security BearerAuth
// @Router /api/v1/assignments/{id} [delete]
func (h *AssignmentHandler) RemoveAssignment(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, models.ErrorResponse{
			Error:   "unauthorized",
			Message: "User not authenticated",
		})
		return
	}

	userRole, _ := c.Get("userRole")
	if userRole != models.UserRoleAdmin && userRole != models.UserRoleManager {
		c.JSON(http.StatusForbidden, models.ErrorResponse{
			Error:   "forbidden",
			Message: "Only administrators and managers can remove assignments",
		})
		return
	}

	assignmentID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "invalid_id",
			Message: "Invalid assignment ID format",
		})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	if err := h.assignmentService.RemoveAssignment(ctx, assignmentID, userID.(uuid.UUID)); err != nil {
		if errors.Is(err, services.ErrAssignmentNotFound) {
			c.JSON(http.StatusNotFound, models.ErrorResponse{
				Error:   "assignment_not_found",
				Message: "Assignment not found",
			})
			return
		}

		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to remove assignment",
		})
		return
	}

	c.Status(http.StatusNoContent)
}

// UpdateAssignment handles PUT /api/v1/assignments/:id
// @Summary Update an assignment role
// @Description Updates the role of a technician assignment
// @Tags assignments
// @Accept json
// @Produce json
// @Param id path string true "Assignment ID (UUID)"
// @Param update body object{role=string} true "Update request"
// @Success 200 {object} object{message=string}
// @Failure 400 {object} models.ErrorResponse
// @Failure 401 {object} models.ErrorResponse
// @Failure 403 {object} models.ErrorResponse
// @Failure 404 {object} models.ErrorResponse
// @Security BearerAuth
// @Router /api/v1/assignments/{id} [put]
func (h *AssignmentHandler) UpdateAssignment(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, models.ErrorResponse{
			Error:   "unauthorized",
			Message: "User not authenticated",
		})
		return
	}

	userRole, _ := c.Get("userRole")
	if userRole != models.UserRoleAdmin && userRole != models.UserRoleManager {
		c.JSON(http.StatusForbidden, models.ErrorResponse{
			Error:   "forbidden",
			Message: "Only administrators and managers can update assignments",
		})
		return
	}

	assignmentID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "invalid_id",
			Message: "Invalid assignment ID format",
		})
		return
	}

	var req struct {
		Role string `json:"role" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "invalid_request",
			Message: "Invalid request data: " + err.Error(),
		})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	if err := h.assignmentService.UpdateAssignment(ctx, assignmentID, req.Role, userID.(uuid.UUID)); err != nil {
		if errors.Is(err, services.ErrAssignmentNotFound) {
			c.JSON(http.StatusNotFound, models.ErrorResponse{
				Error:   "assignment_not_found",
				Message: "Assignment not found",
			})
			return
		}

		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to update assignment",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Assignment updated successfully",
	})
}

// CheckAvailability handles POST /api/v1/assignments/availability/check
// @Summary Check technician availability
// @Description Checks if a technician is available for a specific time slot
// @Tags assignments
// @Accept json
// @Produce json
// @Param request body object{technician_id=string,start_time=string,end_time=string} true "Availability check request"
// @Success 200 {object} services.TechnicianAvailability
// @Failure 400 {object} models.ErrorResponse
// @Failure 401 {object} models.ErrorResponse
// @Security BearerAuth
// @Router /api/v1/assignments/availability/check [post]
func (h *AssignmentHandler) CheckAvailability(c *gin.Context) {
	_, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, models.ErrorResponse{
			Error:   "unauthorized",
			Message: "User not authenticated",
		})
		return
	}

	var req struct {
		TechnicianID uuid.UUID `json:"technician_id" binding:"required"`
		StartTime    int64     `json:"start_time" binding:"required"`
		EndTime      int64     `json:"end_time" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "invalid_request",
			Message: "Invalid request data: " + err.Error(),
		})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	availability, err := h.assignmentService.CheckTechnicianAvailability(ctx, req.TechnicianID, req.StartTime, req.EndTime)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to check availability: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, availability)
}

// GetAvailableTechnicians handles POST /api/v1/assignments/availability/search
// @Summary Find available technicians
// @Description Gets all available technicians for a time slot with optional skill filtering
// @Tags assignments
// @Accept json
// @Produce json
// @Param request body services.AvailabilityCheckRequest true "Search request"
// @Success 200 {array} services.TechnicianAvailability
// @Failure 400 {object} models.ErrorResponse
// @Failure 401 {object} models.ErrorResponse
// @Security BearerAuth
// @Router /api/v1/assignments/availability/search [post]
func (h *AssignmentHandler) GetAvailableTechnicians(c *gin.Context) {
	_, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, models.ErrorResponse{
			Error:   "unauthorized",
			Message: "User not authenticated",
		})
		return
	}

	var req services.AvailabilityCheckRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "invalid_request",
			Message: "Invalid request data: " + err.Error(),
		})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	technicians, err := h.assignmentService.GetAvailableTechnicians(ctx, &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to get available technicians: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, technicians)
}

// OptimizeAssignments handles POST /api/v1/assignments/optimize
// @Summary Optimize technician assignments
// @Description Finds the best available technicians for a job based on skills and workload
// @Tags assignments
// @Accept json
// @Produce json
// @Param request body object{job_id=string,required_skills=[]string,start_time=string,end_time=string} true "Optimization request"
// @Success 200 {array} services.TechnicianAvailability
// @Failure 400 {object} models.ErrorResponse
// @Failure 401 {object} models.ErrorResponse
// @Security BearerAuth
// @Router /api/v1/assignments/optimize [post]
func (h *AssignmentHandler) OptimizeAssignments(c *gin.Context) {
	_, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, models.ErrorResponse{
			Error:   "unauthorized",
			Message: "User not authenticated",
		})
		return
	}

	var req struct {
		JobID          uuid.UUID                  `json:"job_id" binding:"required"`
		RequiredSkills []services.TechnicianSkill `json:"required_skills"`
		StartTime      int64                      `json:"start_time" binding:"required"`
		EndTime        int64                      `json:"end_time" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "invalid_request",
			Message: "Invalid request data: " + err.Error(),
		})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	technicians, err := h.assignmentService.OptimizeAssignments(ctx, req.JobID, req.RequiredSkills, req.StartTime, req.EndTime)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to optimize assignments: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"recommended_technicians": technicians,
		"total":                   len(technicians),
	})
}

// GetTechnicianWorkload handles GET /api/v1/assignments/workload/:technician_id
// @Summary Get technician workload
// @Description Gets the current workload for a technician
// @Tags assignments
// @Produce json
// @Param technician_id path string true "Technician ID (UUID)"
// @Param start_date query string false "Start date (RFC3339)"
// @Param end_date query string false "End date (RFC3339)"
// @Success 200 {object} object{technician_id=string,workload=int,active_jobs=int}
// @Failure 400 {object} models.ErrorResponse
// @Failure 401 {object} models.ErrorResponse
// @Security BearerAuth
// @Router /api/v1/assignments/workload/{technician_id} [get]
func (h *AssignmentHandler) GetTechnicianWorkload(c *gin.Context) {
	_, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, models.ErrorResponse{
			Error:   "unauthorized",
			Message: "User not authenticated",
		})
		return
	}

	technicianID, err := uuid.Parse(c.Param("technician_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "invalid_id",
			Message: "Invalid technician ID format",
		})
		return
	}

	// Parse date range (default to +/- 7 days)
	startDate := time.Now().AddDate(0, 0, -7).Unix() // Last 7 days
	endDate := time.Now().AddDate(0, 0, 7).Unix()    // Next 7 days

	if startDateStr := c.Query("start_date"); startDateStr != "" {
		if parsed, err := strconv.ParseInt(startDateStr, 10, 64); err == nil {
			startDate = parsed
		}
	}

	if endDateStr := c.Query("end_date"); endDateStr != "" {
		if parsed, err := strconv.ParseInt(endDateStr, 10, 64); err == nil {
			endDate = parsed
		}
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	workload, err := h.assignmentService.GetTechnicianWorkload(ctx, technicianID, startDate, endDate)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to get workload: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"technician_id": technicianID,
		"workload":      workload,
		"start_date":    startDate,
		"end_date":      endDate,
	})
}

// Helper method to send assignment notifications asynchronously
func (h *AssignmentHandler) sendAssignmentNotification(assignment *models.JobAssignment) {
	// Skip if notification service is not available
	if h.notificationService == nil {
		return
	}

	// This would be enhanced to fetch job and user details and send notification
	// For now, it's a placeholder for the async notification pattern
	// In production, you'd:
	// 1. Fetch job details
	// 2. Fetch technician details
	// 3. Call h.notificationService.SendAssignmentCreatedNotification(...)
}
