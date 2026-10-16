package handlers

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/javaknight1/servicepro/backend/internal/api/apierror"
	"github.com/javaknight1/servicepro/backend/internal/api/middleware"
	"github.com/javaknight1/servicepro/backend/internal/models"
	"github.com/javaknight1/servicepro/backend/internal/services"
)

// PaymentReminderHandler handles payment reminder HTTP requests
type PaymentReminderHandler struct {
	service *services.PaymentReminderService
}

// NewPaymentReminderHandler creates a new payment reminder handler
func NewPaymentReminderHandler(service *services.PaymentReminderService) *PaymentReminderHandler {
	return &PaymentReminderHandler{service: service}
}

// GetReminderHistory godoc
// @Summary Get reminder history for an invoice
// @Description Get the list of payment reminders sent for a specific invoice
// @Tags invoices
// @Accept json
// @Produce json
// @Param id path string true "Invoice ID"
// @Success 200 {array} models.PaymentReminderResponse
// @Failure 400 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Router /invoices/{id}/reminders [get]
// @Security BearerAuth
func (h *PaymentReminderHandler) GetReminderHistory(c *gin.Context) {
	ctx := c.Request.Context()

	invoiceID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		apierror.WriteResponse(c, http.StatusBadRequest, models.ErrorResponse{Error: "Invalid invoice ID"})
		return
	}

	tenantID, err := middleware.GetTenantID(c)
	if err != nil {
		apierror.WriteResponse(c, http.StatusUnauthorized, models.ErrorResponse{Error: "Tenant context required"})
		return
	}

	reminders, err := h.service.GetReminderHistory(ctx, tenantID, invoiceID)
	if err != nil {
		apierror.WriteResponse(c, http.StatusInternalServerError, models.ErrorResponse{Error: "Failed to get reminder history"})
		return
	}

	c.JSON(http.StatusOK, reminders)
}

// SendManualReminder godoc
// @Summary Send a manual payment reminder
// @Description Send a payment reminder for a specific invoice
// @Tags invoices
// @Accept json
// @Produce json
// @Param id path string true "Invoice ID"
// @Success 201 {object} models.PaymentReminderResponse
// @Failure 400 {object} models.ErrorResponse
// @Failure 404 {object} models.ErrorResponse
// @Failure 409 {object} models.ErrorResponse "Invoice status does not take reminders"
// @Failure 500 {object} models.ErrorResponse
// @Router /invoices/{id}/reminders [post]
// @Security BearerAuth
func (h *PaymentReminderHandler) SendManualReminder(c *gin.Context) {
	ctx := c.Request.Context()

	invoiceID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		apierror.WriteResponse(c, http.StatusBadRequest, models.ErrorResponse{Error: "Invalid invoice ID"})
		return
	}

	tenantID, err := middleware.GetTenantID(c)
	if err != nil {
		apierror.WriteResponse(c, http.StatusUnauthorized, models.ErrorResponse{Error: "Tenant context required"})
		return
	}

	reminder, err := h.service.SendManualReminder(ctx, tenantID, invoiceID)
	if err != nil {
		if errors.Is(err, services.ErrInvoiceNotFound) {
			apierror.WriteResponse(c, http.StatusNotFound, models.ErrorResponse{Error: "Invoice not found"})
			return
		}
		apierror.Respond(c, err, "Failed to send reminder")
		return
	}

	c.JSON(http.StatusCreated, reminder)
}

// GetReminderSettings godoc
// @Summary Get payment reminder settings
// @Description Get the payment reminder configuration for the current tenant
// @Tags settings
// @Accept json
// @Produce json
// @Success 200 {object} models.PaymentReminderSettingsResponse
// @Failure 500 {object} models.ErrorResponse
// @Router /settings/payment-reminders [get]
// @Security BearerAuth
func (h *PaymentReminderHandler) GetReminderSettings(c *gin.Context) {
	ctx := c.Request.Context()

	tenantID, err := middleware.GetTenantID(c)
	if err != nil {
		apierror.WriteResponse(c, http.StatusUnauthorized, models.ErrorResponse{Error: "Tenant context required"})
		return
	}

	settings, err := h.service.GetReminderSettings(ctx, tenantID)
	if err != nil {
		apierror.WriteResponse(c, http.StatusInternalServerError, models.ErrorResponse{Error: "Failed to get reminder settings"})
		return
	}

	c.JSON(http.StatusOK, settings)
}

// UpdateReminderSettings godoc
// @Summary Update payment reminder settings
// @Description Update the payment reminder configuration for the current tenant
// @Tags settings
// @Accept json
// @Produce json
// @Param settings body models.PaymentReminderSettingsRequest true "Reminder settings"
// @Success 200 {object} models.PaymentReminderSettingsResponse
// @Failure 400 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Router /settings/payment-reminders [put]
// @Security BearerAuth
func (h *PaymentReminderHandler) UpdateReminderSettings(c *gin.Context) {
	ctx := c.Request.Context()

	tenantID, err := middleware.GetTenantID(c)
	if err != nil {
		apierror.WriteResponse(c, http.StatusUnauthorized, models.ErrorResponse{Error: "Tenant context required"})
		return
	}

	var req models.PaymentReminderSettingsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		apierror.BindError(c, err)
		return
	}

	if err := h.service.UpdateReminderSettings(ctx, tenantID, &req); err != nil {
		apierror.WriteResponse(c, http.StatusInternalServerError, models.ErrorResponse{Error: "Failed to update reminder settings"})
		return
	}

	// Return updated settings
	settings, err := h.service.GetReminderSettings(ctx, tenantID)
	if err != nil {
		apierror.WriteResponse(c, http.StatusInternalServerError, models.ErrorResponse{Error: "Settings updated but failed to reload"})
		return
	}

	c.JSON(http.StatusOK, settings)
}
