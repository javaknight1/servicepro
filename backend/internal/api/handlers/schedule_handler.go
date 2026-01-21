package handlers

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/javaknight1/servicepro/backend/internal/models"
	"github.com/javaknight1/servicepro/backend/internal/services"
)

// ScheduleHandler handles HTTP requests for schedule management
type ScheduleHandler struct {
	scheduleService services.ScheduleServiceInterface
}

// NewScheduleHandler creates a new schedule handler
func NewScheduleHandler(scheduleService services.ScheduleServiceInterface) *ScheduleHandler {
	return &ScheduleHandler{
		scheduleService: scheduleService,
	}
}

// CreateSchedule handles POST /api/v1/schedules
// @Summary Create a new schedule
// @Description Creates a new job schedule with conflict detection
// @Tags schedules
// @Accept json
// @Produce json
// @Param request body models.ScheduleRequest true "Schedule details"
// @Success 201 {object} models.ScheduleResponse
// @Failure 400 {object} models.ErrorResponse
// @Failure 401 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Security BearerAuth
// @Router /api/v1/schedules [post]
func (h *ScheduleHandler) CreateSchedule(c *gin.Context) {
	// Get authenticated user
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, models.ErrorResponse{
			Error:   "unauthorized",
			Message: "User not authenticated",
		})
		return
	}

	// Parse request body
	var req models.ScheduleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "invalid_request",
			Message: "Invalid request data: " + err.Error(),
		})
		return
	}

	// Create context with timeout
	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	// Create schedule
	response, err := h.scheduleService.CreateSchedule(ctx, &req, userID.(uuid.UUID))
	if err != nil {
		if errors.Is(err, services.ErrInvalidTimeRange) {
			c.JSON(http.StatusBadRequest, models.ErrorResponse{
				Error:   "invalid_time_range",
				Message: err.Error(),
			})
			return
		}

		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to create schedule",
		})
		return
	}

	c.JSON(http.StatusCreated, response)
}

// GetSchedule handles GET /api/v1/schedules/:id
// @Summary Get a schedule by ID
// @Description Retrieves a schedule by its ID
// @Tags schedules
// @Produce json
// @Param id path string true "Schedule ID (UUID)"
// @Success 200 {object} models.ScheduleResponse
// @Failure 400 {object} models.ErrorResponse
// @Failure 404 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Security BearerAuth
// @Router /api/v1/schedules/{id} [get]
func (h *ScheduleHandler) GetSchedule(c *gin.Context) {
	// Parse schedule ID
	scheduleIDStr := c.Param("id")
	scheduleID, err := uuid.Parse(scheduleIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "invalid_id",
			Message: "Invalid schedule ID format",
		})
		return
	}

	// Create context with timeout
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	// Get schedule
	response, err := h.scheduleService.GetSchedule(ctx, scheduleID)
	if err != nil {
		if errors.Is(err, services.ErrScheduleNotFound) {
			c.JSON(http.StatusNotFound, models.ErrorResponse{
				Error:   "schedule_not_found",
				Message: "Schedule not found",
			})
			return
		}

		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to retrieve schedule",
		})
		return
	}

	c.JSON(http.StatusOK, response)
}

// UpdateSchedule handles PUT /api/v1/schedules/:id
// @Summary Update a schedule
// @Description Updates an existing schedule
// @Tags schedules
// @Accept json
// @Produce json
// @Param id path string true "Schedule ID (UUID)"
// @Param request body models.ScheduleRequest true "Updated schedule details"
// @Success 200 {object} models.ScheduleResponse
// @Failure 400 {object} models.ErrorResponse
// @Failure 401 {object} models.ErrorResponse
// @Failure 404 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Security BearerAuth
// @Router /api/v1/schedules/{id} [put]
func (h *ScheduleHandler) UpdateSchedule(c *gin.Context) {
	// Get authenticated user
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, models.ErrorResponse{
			Error:   "unauthorized",
			Message: "User not authenticated",
		})
		return
	}

	// Parse schedule ID
	scheduleIDStr := c.Param("id")
	scheduleID, err := uuid.Parse(scheduleIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "invalid_id",
			Message: "Invalid schedule ID format",
		})
		return
	}

	// Parse request body
	var req models.ScheduleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "invalid_request",
			Message: "Invalid request data: " + err.Error(),
		})
		return
	}

	// Create context with timeout
	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	// Update schedule
	response, err := h.scheduleService.UpdateSchedule(ctx, scheduleID, &req, userID.(uuid.UUID))
	if err != nil {
		if errors.Is(err, services.ErrScheduleNotFound) {
			c.JSON(http.StatusNotFound, models.ErrorResponse{
				Error:   "schedule_not_found",
				Message: "Schedule not found",
			})
			return
		}

		if errors.Is(err, services.ErrInvalidTimeRange) {
			c.JSON(http.StatusBadRequest, models.ErrorResponse{
				Error:   "invalid_time_range",
				Message: err.Error(),
			})
			return
		}

		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to update schedule",
		})
		return
	}

	c.JSON(http.StatusOK, response)
}

// DeleteSchedule handles DELETE /api/v1/schedules/:id
// @Summary Delete a schedule
// @Description Deletes a schedule (soft delete)
// @Tags schedules
// @Param id path string true "Schedule ID (UUID)"
// @Success 204
// @Failure 400 {object} models.ErrorResponse
// @Failure 401 {object} models.ErrorResponse
// @Failure 404 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Security BearerAuth
// @Router /api/v1/schedules/{id} [delete]
func (h *ScheduleHandler) DeleteSchedule(c *gin.Context) {
	// Parse schedule ID
	scheduleIDStr := c.Param("id")
	scheduleID, err := uuid.Parse(scheduleIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "invalid_id",
			Message: "Invalid schedule ID format",
		})
		return
	}

	// Create context with timeout
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	// Delete schedule
	if err := h.scheduleService.DeleteSchedule(ctx, scheduleID); err != nil {
		if errors.Is(err, services.ErrScheduleNotFound) {
			c.JSON(http.StatusNotFound, models.ErrorResponse{
				Error:   "schedule_not_found",
				Message: "Schedule not found",
			})
			return
		}

		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to delete schedule",
		})
		return
	}

	c.Status(http.StatusNoContent)
}

// ListSchedules handles GET /api/v1/schedules
// @Summary List schedules
// @Description Retrieves a paginated list of schedules with filters
// @Tags schedules
// @Produce json
// @Param start_date query string false "Start date (RFC3339)"
// @Param end_date query string false "End date (RFC3339)"
// @Param tech_id query string false "Technician ID (UUID)"
// @Param confirmed query boolean false "Filter by confirmation status"
// @Param cancelled query boolean false "Filter by cancellation status"
// @Param page query int false "Page number" default(1)
// @Param page_size query int false "Page size" default(50)
// @Success 200 {object} models.ScheduleListResponse
// @Failure 400 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Security BearerAuth
// @Router /api/v1/schedules [get]
func (h *ScheduleHandler) ListSchedules(c *gin.Context) {
	// Parse query parameters
	var params models.ScheduleQueryParams
	if err := c.ShouldBindQuery(&params); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "invalid_query",
			Message: "Invalid query parameters: " + err.Error(),
		})
		return
	}

	// Set defaults
	if params.Page == 0 {
		params.Page = 1
	}
	if params.PageSize == 0 {
		params.PageSize = 50
	}

	// Create context with timeout
	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	// List schedules
	response, err := h.scheduleService.ListSchedules(ctx, &params)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to list schedules",
		})
		return
	}

	c.JSON(http.StatusOK, response)
}

// GetTechnicianSchedule handles GET /api/v1/schedules/technician/:tech_id
// @Summary Get technician schedule
// @Description Retrieves all schedules for a technician within a date range
// @Tags schedules
// @Produce json
// @Param tech_id path string true "Technician ID (UUID)"
// @Param start_date query string true "Start date (RFC3339)"
// @Param end_date query string true "End date (RFC3339)"
// @Success 200 {array} models.ScheduleResponse
// @Failure 400 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Security BearerAuth
// @Router /api/v1/schedules/technician/{tech_id} [get]
func (h *ScheduleHandler) GetTechnicianSchedule(c *gin.Context) {
	// Parse tech ID
	techIDStr := c.Param("tech_id")
	techID, err := uuid.Parse(techIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "invalid_id",
			Message: "Invalid technician ID format",
		})
		return
	}

	// Parse date parameters
	startDateStr := c.Query("start_date")
	endDateStr := c.Query("end_date")

	if startDateStr == "" || endDateStr == "" {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "missing_parameters",
			Message: "start_date and end_date are required",
		})
		return
	}

	startDate, err := time.Parse(time.RFC3339, startDateStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "invalid_date",
			Message: "Invalid start_date format",
		})
		return
	}

	endDate, err := time.Parse(time.RFC3339, endDateStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "invalid_date",
			Message: "Invalid end_date format",
		})
		return
	}

	// Create context with timeout
	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	// Get technician schedule
	schedules, err := h.scheduleService.GetTechnicianSchedule(ctx, techID, startDate, endDate)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to retrieve technician schedule",
		})
		return
	}

	c.JSON(http.StatusOK, schedules)
}
