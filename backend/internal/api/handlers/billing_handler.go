package handlers

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/javaknight1/servicepro/backend/internal/api/middleware"
	"github.com/javaknight1/servicepro/backend/internal/models"
	"github.com/javaknight1/servicepro/backend/internal/services"
)

// BillingHandler handles billing-related HTTP requests
type BillingHandler struct {
	stripeService *services.StripeService
}

// NewBillingHandler creates a new billing handler
func NewBillingHandler(stripeService *services.StripeService) *BillingHandler {
	return &BillingHandler{
		stripeService: stripeService,
	}
}

// CreateSetupIntent creates a setup intent for adding a payment method
// @Summary		Create setup intent
// @Description	Creates a Stripe setup intent for adding a new payment method
// @Tags			Billing
// @Accept			json
// @Produce		json
// @Security		BearerAuth
// @Param			id	path		string	true	"Tenant ID"
// @Success		200	{object}	models.CreateSetupIntentResponse
// @Failure		400	{object}	models.ErrorResponse
// @Failure		401	{object}	models.ErrorResponse
// @Failure		403	{object}	models.ErrorResponse
// @Failure		500	{object}	models.ErrorResponse
// @Failure		503	{object}	models.ErrorResponse
// @Router			/tenants/{id}/billing/setup-intent [post]
func (h *BillingHandler) CreateSetupIntent(c *gin.Context) {
	tenantID, err := h.getTenantID(c)
	if err != nil {
		return
	}

	response, err := h.stripeService.CreateSetupIntent(c.Request.Context(), tenantID)
	if err != nil {
		if errors.Is(err, services.ErrStripeNotConfigured) {
			c.JSON(http.StatusServiceUnavailable, models.ErrorResponse{
				Error:   "stripe_not_configured",
				Message: "Stripe is not configured",
			})
			return
		}
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "setup_intent_failed",
			Message: "Failed to create setup intent",
		})
		return
	}

	c.JSON(http.StatusOK, response)
}

// AddPaymentMethod adds a new payment method after setup intent confirmation
// @Summary		Add payment method
// @Description	Adds a new payment method after Stripe setup intent confirmation
// @Tags			Billing
// @Accept			json
// @Produce		json
// @Security		BearerAuth
// @Param			id		path		string								true	"Tenant ID"
// @Param			request	body		models.AddSavedPaymentMethodRequest	true	"Payment method request"
// @Success		201		{object}	models.SavedPaymentMethodResponse
// @Failure		400		{object}	models.ErrorResponse
// @Failure		401		{object}	models.ErrorResponse
// @Failure		403		{object}	models.ErrorResponse
// @Failure		500		{object}	models.ErrorResponse
// @Failure		503		{object}	models.ErrorResponse
// @Router			/tenants/{id}/billing/payment-methods [post]
func (h *BillingHandler) AddPaymentMethod(c *gin.Context) {
	tenantID, err := h.getTenantID(c)
	if err != nil {
		return
	}

	var req models.AddSavedPaymentMethodRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "invalid_request",
			Message: err.Error(),
		})
		return
	}

	response, err := h.stripeService.AddPaymentMethod(c.Request.Context(), tenantID, req.PaymentMethodID, req.SetAsDefault)
	if err != nil {
		if errors.Is(err, services.ErrStripeNotConfigured) {
			c.JSON(http.StatusServiceUnavailable, models.ErrorResponse{
				Error:   "stripe_not_configured",
				Message: "Stripe is not configured",
			})
			return
		}
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "add_payment_method_failed",
			Message: "Failed to add payment method",
		})
		return
	}

	c.JSON(http.StatusCreated, response)
}

// GetPaymentMethods retrieves all payment methods for a tenant
// @Summary		Get payment methods
// @Description	Retrieves all saved payment methods for a tenant
// @Tags			Billing
// @Accept			json
// @Produce		json
// @Security		BearerAuth
// @Param			id	path		string	true	"Tenant ID"
// @Success		200	{object}	models.ListSavedPaymentMethodsResponse
// @Failure		400	{object}	models.ErrorResponse
// @Failure		401	{object}	models.ErrorResponse
// @Failure		403	{object}	models.ErrorResponse
// @Failure		500	{object}	models.ErrorResponse
// @Router			/tenants/{id}/billing/payment-methods [get]
func (h *BillingHandler) GetPaymentMethods(c *gin.Context) {
	tenantID, err := h.getTenantID(c)
	if err != nil {
		return
	}

	response, err := h.stripeService.GetPaymentMethods(c.Request.Context(), tenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "fetch_failed",
			Message: "Failed to fetch payment methods",
		})
		return
	}

	c.JSON(http.StatusOK, response)
}

// SetDefaultPaymentMethod sets a payment method as default
// @Summary		Set default payment method
// @Description	Sets a payment method as the default for the tenant
// @Tags			Billing
// @Accept			json
// @Produce		json
// @Security		BearerAuth
// @Param			id		path		string	true	"Tenant ID"
// @Param			pmId	path		string	true	"Payment Method ID"
// @Success		200		{object}	map[string]string
// @Failure		400		{object}	models.ErrorResponse
// @Failure		401		{object}	models.ErrorResponse
// @Failure		403		{object}	models.ErrorResponse
// @Failure		404		{object}	models.ErrorResponse
// @Failure		500		{object}	models.ErrorResponse
// @Router			/tenants/{id}/billing/payment-methods/{pmId}/default [put]
func (h *BillingHandler) SetDefaultPaymentMethod(c *gin.Context) {
	tenantID, err := h.getTenantID(c)
	if err != nil {
		return
	}

	pmIDStr := c.Param("pmId")
	pmID, err := uuid.Parse(pmIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "invalid_id",
			Message: "Invalid payment method ID format",
		})
		return
	}

	err = h.stripeService.SetDefaultPaymentMethod(c.Request.Context(), tenantID, pmID)
	if err != nil {
		if errors.Is(err, services.ErrPaymentMethodNotFound) {
			c.JSON(http.StatusNotFound, models.ErrorResponse{
				Error:   "not_found",
				Message: "Payment method not found",
			})
			return
		}
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "update_failed",
			Message: "Failed to set default payment method",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Default payment method updated"})
}

// DeletePaymentMethod removes a payment method
// @Summary		Delete payment method
// @Description	Removes a saved payment method from the tenant
// @Tags			Billing
// @Accept			json
// @Produce		json
// @Security		BearerAuth
// @Param			id		path		string	true	"Tenant ID"
// @Param			pmId	path		string	true	"Payment Method ID"
// @Success		200		{object}	map[string]string
// @Failure		400		{object}	models.ErrorResponse
// @Failure		401		{object}	models.ErrorResponse
// @Failure		403		{object}	models.ErrorResponse
// @Failure		404		{object}	models.ErrorResponse
// @Failure		500		{object}	models.ErrorResponse
// @Router			/tenants/{id}/billing/payment-methods/{pmId} [delete]
func (h *BillingHandler) DeletePaymentMethod(c *gin.Context) {
	tenantID, err := h.getTenantID(c)
	if err != nil {
		return
	}

	pmIDStr := c.Param("pmId")
	pmID, err := uuid.Parse(pmIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "invalid_id",
			Message: "Invalid payment method ID format",
		})
		return
	}

	err = h.stripeService.DeletePaymentMethod(c.Request.Context(), tenantID, pmID)
	if err != nil {
		if errors.Is(err, services.ErrPaymentMethodNotFound) {
			c.JSON(http.StatusNotFound, models.ErrorResponse{
				Error:   "not_found",
				Message: "Payment method not found",
			})
			return
		}
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "delete_failed",
			Message: "Failed to delete payment method",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Payment method deleted"})
}

// GetBillingHistory retrieves billing history for a tenant
// @Summary		Get billing history
// @Description	Retrieves billing history and past transactions for a tenant
// @Tags			Billing
// @Accept			json
// @Produce		json
// @Security		BearerAuth
// @Param			id			path		string	true	"Tenant ID"
// @Param			page		query		int		false	"Page number"	default(1)
// @Param			page_size	query		int		false	"Page size"		default(20)
// @Success		200			{object}	models.ListBillingEventsResponse
// @Failure		400			{object}	models.ErrorResponse
// @Failure		401			{object}	models.ErrorResponse
// @Failure		403			{object}	models.ErrorResponse
// @Failure		500			{object}	models.ErrorResponse
// @Router			/tenants/{id}/billing/history [get]
func (h *BillingHandler) GetBillingHistory(c *gin.Context) {
	tenantID, err := h.getTenantID(c)
	if err != nil {
		return
	}

	// Parse pagination parameters
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	response, err := h.stripeService.GetBillingHistory(c.Request.Context(), tenantID, page, pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "fetch_failed",
			Message: "Failed to fetch billing history",
		})
		return
	}

	c.JSON(http.StatusOK, response)
}

// GetStripePublishableKey returns the Stripe publishable key for frontend
// @Summary		Get billing config
// @Description	Returns Stripe configuration including whether Stripe is enabled
// @Tags			Billing
// @Accept			json
// @Produce		json
// @Security		BearerAuth
// @Success		200	{object}	map[string]interface{}
// @Router			/billing/config [get]
func (h *BillingHandler) GetStripePublishableKey(c *gin.Context) {
	if !h.stripeService.IsConfigured() {
		c.JSON(http.StatusOK, gin.H{
			"stripe_enabled":  false,
			"publishable_key": "",
		})
		return
	}

	// The publishable key should be passed to this handler via configuration
	// For now, we'll just indicate that Stripe is enabled
	c.JSON(http.StatusOK, gin.H{
		"stripe_enabled": true,
	})
}

// Helper to get tenant ID from path parameter
func (h *BillingHandler) getTenantID(c *gin.Context) (uuid.UUID, error) {
	idStr := c.Param("id")
	tenantID, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "invalid_id",
			Message: "Invalid tenant ID format",
		})
		return uuid.Nil, err
	}

	// Verify user has access to this tenant
	userTenantID, err := middleware.GetTenantID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, models.ErrorResponse{
			Error:   "unauthorized",
			Message: "User not authenticated",
		})
		return uuid.Nil, err
	}

	if tenantID != userTenantID {
		c.JSON(http.StatusForbidden, models.ErrorResponse{
			Error:   "forbidden",
			Message: "Access denied to this tenant",
		})
		return uuid.Nil, errors.New("access denied")
	}

	return tenantID, nil
}
