package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/javaknight1/servicepro/backend/internal/api/apierror"
	"github.com/javaknight1/servicepro/backend/internal/models"
	"github.com/javaknight1/servicepro/backend/internal/services"
)

// AgreementHandler serves service agreements (T167).
type AgreementHandler struct {
	agreements services.AgreementServiceInterface
}

// NewAgreementHandler creates an agreement handler.
func NewAgreementHandler(agreements services.AgreementServiceInterface) *AgreementHandler {
	return &AgreementHandler{agreements: agreements}
}

func (h *AgreementHandler) respondOne(c *gin.Context, status int, a *models.ServiceAgreement, err error, msg string) {
	if err != nil {
		apierror.Respond(c, err, msg)
		return
	}
	c.JSON(status, a.ToResponse())
}

// ListAgreements handles GET /api/v1/agreements
// @Summary List service agreements
// @Description The account's service agreements, newest first; filter by customer, site or status.
// @Tags agreements
// @Produce json
// @Security BearerAuth
// @Param customer_id query string false "Customer ID"
// @Param site_id query string false "Site ID"
// @Param status query string false "active or paused"
// @Success 200 {object} models.AgreementListResponse
// @Failure 400 {object} models.ErrorResponse "Invalid filter"
// @Failure 500 {object} models.ErrorResponse "Internal server error"
// @Router /api/v1/agreements [get]
func (h *AgreementHandler) ListAgreements(c *gin.Context) {
	var filter models.AgreementListFilter
	if err := c.ShouldBindQuery(&filter); err != nil {
		apierror.BindError(c, err)
		return
	}
	list, err := h.agreements.List(c.Request.Context(), &filter)
	if err != nil {
		apierror.Respond(c, err, "Failed to list agreements")
		return
	}
	resp := models.AgreementListResponse{Agreements: make([]models.AgreementResponse, len(list)), Total: len(list)}
	for i := range list {
		resp.Agreements[i] = list[i].ToResponse()
	}
	c.JSON(http.StatusOK, resp)
}

// GetAgreement handles GET /api/v1/agreements/:id
// @Summary Get a service agreement
// @Tags agreements
// @Produce json
// @Security BearerAuth
// @Param id path string true "Agreement ID"
// @Success 200 {object} models.AgreementResponse
// @Failure 400 {object} models.ErrorResponse "Invalid ID"
// @Failure 404 {object} models.ErrorResponse "Agreement not found"
// @Failure 500 {object} models.ErrorResponse "Internal server error"
// @Router /api/v1/agreements/{id} [get]
func (h *AgreementHandler) GetAgreement(c *gin.Context) {
	id, ok := parseUUIDParam(c, "id", "agreement")
	if !ok {
		return
	}
	a, err := h.agreements.Get(c.Request.Context(), id)
	h.respondOne(c, http.StatusOK, a, err, "Failed to get agreement")
}

// CreateAgreement handles POST /api/v1/agreements
// @Summary Create a service agreement
// @Description Recurring work at one of the customer's sites: frequency (weekdays, weekly on given weekdays, N times a week, monthly on a day), start/end dates, local visit start time, price per month or visit, optional technician. Visits are generated as scheduled jobs up to 14 days ahead.
// @Tags agreements
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body models.CreateAgreementRequest true "Agreement"
// @Success 201 {object} models.AgreementResponse
// @Failure 400 {object} models.ErrorResponse "Invalid request, or the site isn't the customer's"
// @Failure 404 {object} models.ErrorResponse "Customer or technician not found"
// @Failure 500 {object} models.ErrorResponse "Internal server error"
// @Router /api/v1/agreements [post]
func (h *AgreementHandler) CreateAgreement(c *gin.Context) {
	userID, ok := c.Get("user_id")
	uid, isUUID := userID.(uuid.UUID)
	if !ok || !isUUID {
		apierror.Write(c, http.StatusUnauthorized, "unauthorized", "User not authenticated")
		return
	}
	var req models.CreateAgreementRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		apierror.BindError(c, err)
		return
	}
	a, err := h.agreements.Create(c.Request.Context(), &req, uid)
	h.respondOne(c, http.StatusCreated, a, err, "Failed to create agreement")
}

// UpdateAgreement handles PUT /api/v1/agreements/:id
// @Summary Update a service agreement
// @Description Changes the fields present in the body. Visits already generated (the next 14 days) keep their jobs.
// @Tags agreements
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Agreement ID"
// @Param request body models.UpdateAgreementRequest true "Fields to change"
// @Success 200 {object} models.AgreementResponse
// @Failure 400 {object} models.ErrorResponse "Invalid request"
// @Failure 404 {object} models.ErrorResponse "Agreement or technician not found"
// @Failure 500 {object} models.ErrorResponse "Internal server error"
// @Router /api/v1/agreements/{id} [put]
func (h *AgreementHandler) UpdateAgreement(c *gin.Context) {
	id, ok := parseUUIDParam(c, "id", "agreement")
	if !ok {
		return
	}
	var req models.UpdateAgreementRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		apierror.BindError(c, err)
		return
	}
	a, err := h.agreements.Update(c.Request.Context(), id, &req)
	h.respondOne(c, http.StatusOK, a, err, "Failed to update agreement")
}

// PauseAgreement handles POST /api/v1/agreements/:id/pause
// @Summary Pause a service agreement
// @Description No new visits are generated while paused; jobs already generated stay.
// @Tags agreements
// @Produce json
// @Security BearerAuth
// @Param id path string true "Agreement ID"
// @Success 200 {object} models.AgreementResponse
// @Failure 400 {object} models.ErrorResponse "Invalid ID"
// @Failure 404 {object} models.ErrorResponse "Agreement not found"
// @Failure 500 {object} models.ErrorResponse "Internal server error"
// @Router /api/v1/agreements/{id}/pause [post]
func (h *AgreementHandler) PauseAgreement(c *gin.Context) {
	h.setStatus(c, models.AgreementStatusPaused)
}

// ResumeAgreement handles POST /api/v1/agreements/:id/resume
// @Summary Resume a paused service agreement
// @Description Visits are generated again from the next generator run.
// @Tags agreements
// @Produce json
// @Security BearerAuth
// @Param id path string true "Agreement ID"
// @Success 200 {object} models.AgreementResponse
// @Failure 400 {object} models.ErrorResponse "Invalid ID"
// @Failure 404 {object} models.ErrorResponse "Agreement not found"
// @Failure 500 {object} models.ErrorResponse "Internal server error"
// @Router /api/v1/agreements/{id}/resume [post]
func (h *AgreementHandler) ResumeAgreement(c *gin.Context) {
	h.setStatus(c, models.AgreementStatusActive)
}

func (h *AgreementHandler) setStatus(c *gin.Context, status models.AgreementStatus) {
	id, ok := parseUUIDParam(c, "id", "agreement")
	if !ok {
		return
	}
	a, err := h.agreements.SetStatus(c.Request.Context(), id, status)
	h.respondOne(c, http.StatusOK, a, err, "Failed to change agreement status")
}

// DeleteAgreement handles DELETE /api/v1/agreements/:id
// @Summary Delete a service agreement
// @Description Soft-deletes the agreement; jobs it generated stay.
// @Tags agreements
// @Security BearerAuth
// @Param id path string true "Agreement ID"
// @Success 204 "Deleted"
// @Failure 400 {object} models.ErrorResponse "Invalid ID"
// @Failure 404 {object} models.ErrorResponse "Agreement not found"
// @Failure 500 {object} models.ErrorResponse "Internal server error"
// @Router /api/v1/agreements/{id} [delete]
func (h *AgreementHandler) DeleteAgreement(c *gin.Context) {
	id, ok := parseUUIDParam(c, "id", "agreement")
	if !ok {
		return
	}
	if err := h.agreements.Delete(c.Request.Context(), id); err != nil {
		apierror.Respond(c, err, "Failed to delete agreement")
		return
	}
	c.Status(http.StatusNoContent)
}

// PreviewAgreement handles GET /api/v1/agreements/:id/preview
// @Summary Preview an agreement's next visits
// @Description The visit dates and start/end times over the next days (default 14, at most 92), in the tenant's time zone. Empty when paused.
// @Tags agreements
// @Produce json
// @Security BearerAuth
// @Param id path string true "Agreement ID"
// @Param days query int false "Days ahead (default 14, max 92)"
// @Success 200 {object} models.AgreementPreviewResponse
// @Failure 400 {object} models.ErrorResponse "Invalid ID or days"
// @Failure 404 {object} models.ErrorResponse "Agreement not found"
// @Failure 500 {object} models.ErrorResponse "Internal server error"
// @Router /api/v1/agreements/{id}/preview [get]
func (h *AgreementHandler) PreviewAgreement(c *gin.Context) {
	id, ok := parseUUIDParam(c, "id", "agreement")
	if !ok {
		return
	}
	days := 0
	if v := c.Query("days"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n < 1 {
			apierror.Write(c, http.StatusBadRequest, "invalid_days", "days must be a positive number")
			return
		}
		days = n
	}
	resp, err := h.agreements.Preview(c.Request.Context(), id, days)
	if err != nil {
		apierror.Respond(c, err, "Failed to preview agreement")
		return
	}
	c.JSON(http.StatusOK, resp)
}
