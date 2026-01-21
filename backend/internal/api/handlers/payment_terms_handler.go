package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"github.com/javaknight1/servicepro/backend/internal/models"
	"github.com/javaknight1/servicepro/backend/internal/services"
)

// PaymentTermsHandler handles payment terms related HTTP requests
type PaymentTermsHandler struct {
	service *services.PaymentTermsService
}

// NewPaymentTermsHandler creates a new payment terms handler
func NewPaymentTermsHandler(service *services.PaymentTermsService) *PaymentTermsHandler {
	return &PaymentTermsHandler{service: service}
}

// CreatePaymentTermRequest represents the request to create a payment term
type CreatePaymentTermRequest struct {
	Name               string          `json:"name" binding:"required"`
	Description        string          `json:"description"`
	TermType           string          `json:"term_type" binding:"required"`
	DaysUntilDue       int             `json:"days_until_due" binding:"required,min=0"`
	DiscountPercentage decimal.Decimal `json:"discount_percentage"`
	DiscountDays       int             `json:"discount_days" binding:"min=0"`
	LateFeePercentage  decimal.Decimal `json:"late_fee_percentage"`
	LateFeeAmount      decimal.Decimal `json:"late_fee_amount"`
	GracePeriodDays    int             `json:"grace_period_days" binding:"min=0"`
	IsDefault          bool            `json:"is_default"`
	IsActive           bool            `json:"is_active"`
}

// UpdatePaymentTermRequest represents the request to update a payment term
type UpdatePaymentTermRequest struct {
	Name               *string          `json:"name"`
	Description        *string          `json:"description"`
	DaysUntilDue       *int             `json:"days_until_due" binding:"omitempty,min=0"`
	DiscountPercentage *decimal.Decimal `json:"discount_percentage"`
	DiscountDays       *int             `json:"discount_days" binding:"omitempty,min=0"`
	LateFeePercentage  *decimal.Decimal `json:"late_fee_percentage"`
	LateFeeAmount      *decimal.Decimal `json:"late_fee_amount"`
	GracePeriodDays    *int             `json:"grace_period_days" binding:"omitempty,min=0"`
	IsDefault          *bool            `json:"is_default"`
	IsActive           *bool            `json:"is_active"`
}

// CalculateDueDateRequest represents the request to calculate a due date
type CalculateDueDateRequest struct {
	IssueDate     time.Time `json:"issue_date" binding:"required"`
	PaymentTermID uuid.UUID `json:"payment_term_id" binding:"required"`
	Timezone      string    `json:"timezone"`
}

// CalculateDiscountRequest represents the request to calculate early payment discount
type CalculateDiscountRequest struct {
	InvoiceAmount decimal.Decimal `json:"invoice_amount" binding:"required"`
	PaymentDate   time.Time       `json:"payment_date" binding:"required"`
	IssueDate     time.Time       `json:"issue_date" binding:"required"`
	PaymentTermID uuid.UUID       `json:"payment_term_id" binding:"required"`
}

// CalculateLateFeeRequest represents the request to calculate late fee
type CalculateLateFeeRequest struct {
	InvoiceAmount decimal.Decimal `json:"invoice_amount" binding:"required"`
	CurrentDate   time.Time       `json:"current_date" binding:"required"`
	DueDate       time.Time       `json:"due_date" binding:"required"`
	PaymentTermID uuid.UUID       `json:"payment_term_id" binding:"required"`
	Timezone      string          `json:"timezone"`
}

// GetPaymentStatusRequest represents the request to get payment status
type GetPaymentStatusRequest struct {
	InvoiceAmount decimal.Decimal `json:"invoice_amount" binding:"required"`
	IssueDate     time.Time       `json:"issue_date" binding:"required"`
	DueDate       time.Time       `json:"due_date" binding:"required"`
	PaymentTermID uuid.UUID       `json:"payment_term_id" binding:"required"`
	CurrentDate   *time.Time      `json:"current_date"`
	Timezone      string          `json:"timezone"`
}

// CalculatePaymentDetailsRequest represents the request to calculate payment details
type CalculatePaymentDetailsRequest struct {
	InvoiceAmount decimal.Decimal `json:"invoice_amount" binding:"required"`
	IssueDate     time.Time       `json:"issue_date" binding:"required"`
	PaymentTermID uuid.UUID       `json:"payment_term_id" binding:"required"`
	PaymentDate   *time.Time      `json:"payment_date"`
	Timezone      string          `json:"timezone"`
}

// ListPaymentTerms godoc
// @Summary List payment terms
// @Description Get a list of all payment terms
// @Tags payment-terms
// @Accept json
// @Produce json
// @Param active_only query boolean false "Return only active payment terms"
// @Success 200 {array} models.PaymentTerm
// @Failure 500 {object} ErrorResponse
// @Router /payment-terms [get]
// @Security BearerAuth
func (h *PaymentTermsHandler) ListPaymentTerms(c *gin.Context) {
	ctx := c.Request.Context()

	activeOnly := c.DefaultQuery("active_only", "false") == "true"

	terms, err := h.service.ListPaymentTerms(ctx, activeOnly)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Failed to list payment terms",
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, terms)
}

// GetPaymentTerm godoc
// @Summary Get payment term
// @Description Get a payment term by ID
// @Tags payment-terms
// @Accept json
// @Produce json
// @Param id path string true "Payment Term ID"
// @Success 200 {object} models.PaymentTerm
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Router /payment-terms/{id} [get]
// @Security BearerAuth
func (h *PaymentTermsHandler) GetPaymentTerm(c *gin.Context) {
	ctx := c.Request.Context()

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid payment term ID"})
		return
	}

	term, err := h.service.GetPaymentTerm(ctx, id)
	if err != nil {
		c.JSON(http.StatusNotFound, ErrorResponse{
			Error:   "Payment term not found",
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, term)
}

// GetDefaultPaymentTerm godoc
// @Summary Get default payment term
// @Description Get the default payment term
// @Tags payment-terms
// @Accept json
// @Produce json
// @Success 200 {object} models.PaymentTerm
// @Failure 404 {object} ErrorResponse
// @Router /payment-terms/default [get]
// @Security BearerAuth
func (h *PaymentTermsHandler) GetDefaultPaymentTerm(c *gin.Context) {
	ctx := c.Request.Context()

	term, err := h.service.GetDefaultPaymentTerm(ctx)
	if err != nil {
		c.JSON(http.StatusNotFound, ErrorResponse{
			Error:   "Default payment term not found",
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, term)
}

// CreatePaymentTerm godoc
// @Summary Create payment term
// @Description Create a new payment term
// @Tags payment-terms
// @Accept json
// @Produce json
// @Param request body CreatePaymentTermRequest true "Payment term details"
// @Success 201 {object} models.PaymentTerm
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /payment-terms [post]
// @Security BearerAuth
func (h *PaymentTermsHandler) CreatePaymentTerm(c *gin.Context) {
	ctx := c.Request.Context()

	var req CreatePaymentTermRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid request",
			Message: err.Error(),
		})
		return
	}

	term := &models.PaymentTerm{
		Name:               req.Name,
		Description:        req.Description,
		TermType:           models.PaymentTermType(req.TermType),
		DaysUntilDue:       req.DaysUntilDue,
		DiscountPercentage: req.DiscountPercentage,
		DiscountDays:       req.DiscountDays,
		LateFeePercentage:  req.LateFeePercentage,
		LateFeeAmount:      req.LateFeeAmount,
		GracePeriodDays:    req.GracePeriodDays,
		IsDefault:          req.IsDefault,
		IsActive:           req.IsActive,
	}

	if err := h.service.CreatePaymentTerm(ctx, term); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Failed to create payment term",
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusCreated, term)
}

// UpdatePaymentTerm godoc
// @Summary Update payment term
// @Description Update an existing payment term
// @Tags payment-terms
// @Accept json
// @Produce json
// @Param id path string true "Payment Term ID"
// @Param request body UpdatePaymentTermRequest true "Payment term updates"
// @Success 200 {object} models.PaymentTerm
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /payment-terms/{id} [put]
// @Security BearerAuth
func (h *PaymentTermsHandler) UpdatePaymentTerm(c *gin.Context) {
	ctx := c.Request.Context()

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid payment term ID"})
		return
	}

	var req UpdatePaymentTermRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid request",
			Message: err.Error(),
		})
		return
	}

	// Build updates
	updates := &models.PaymentTerm{}
	if req.Name != nil {
		updates.Name = *req.Name
	}
	if req.Description != nil {
		updates.Description = *req.Description
	}
	if req.DaysUntilDue != nil {
		updates.DaysUntilDue = *req.DaysUntilDue
	}
	if req.DiscountPercentage != nil {
		updates.DiscountPercentage = *req.DiscountPercentage
	}
	if req.DiscountDays != nil {
		updates.DiscountDays = *req.DiscountDays
	}
	if req.LateFeePercentage != nil {
		updates.LateFeePercentage = *req.LateFeePercentage
	}
	if req.LateFeeAmount != nil {
		updates.LateFeeAmount = *req.LateFeeAmount
	}
	if req.GracePeriodDays != nil {
		updates.GracePeriodDays = *req.GracePeriodDays
	}
	if req.IsDefault != nil {
		updates.IsDefault = *req.IsDefault
	}
	if req.IsActive != nil {
		updates.IsActive = *req.IsActive
	}

	if err := h.service.UpdatePaymentTerm(ctx, id, updates); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Failed to update payment term",
			Message: err.Error(),
		})
		return
	}

	// Fetch updated term
	term, err := h.service.GetPaymentTerm(ctx, id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Failed to fetch updated payment term",
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, term)
}

// DeletePaymentTerm godoc
// @Summary Delete payment term
// @Description Delete a payment term
// @Tags payment-terms
// @Accept json
// @Produce json
// @Param id path string true "Payment Term ID"
// @Success 204
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /payment-terms/{id} [delete]
// @Security BearerAuth
func (h *PaymentTermsHandler) DeletePaymentTerm(c *gin.Context) {
	ctx := c.Request.Context()

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid payment term ID"})
		return
	}

	if err := h.service.DeletePaymentTerm(ctx, id); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Failed to delete payment term",
			Message: err.Error(),
		})
		return
	}

	c.Status(http.StatusNoContent)
}

// CalculateDueDate godoc
// @Summary Calculate due date
// @Description Calculate the due date based on payment terms
// @Tags payment-terms
// @Accept json
// @Produce json
// @Param request body CalculateDueDateRequest true "Due date calculation parameters"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} ErrorResponse
// @Router /payment-terms/calculate/due-date [post]
// @Security BearerAuth
func (h *PaymentTermsHandler) CalculateDueDate(c *gin.Context) {
	ctx := c.Request.Context()

	var req CalculateDueDateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid request",
			Message: err.Error(),
		})
		return
	}

	timezone := req.Timezone
	if timezone == "" {
		timezone = "UTC"
	}

	dueDate, err := h.service.CalculateDueDate(ctx, req.IssueDate, req.PaymentTermID, timezone)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Failed to calculate due date",
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"due_date":   dueDate,
		"issue_date": req.IssueDate,
		"timezone":   timezone,
	})
}

// CalculateDiscount godoc
// @Summary Calculate early payment discount
// @Description Calculate the early payment discount
// @Tags payment-terms
// @Accept json
// @Produce json
// @Param request body CalculateDiscountRequest true "Discount calculation parameters"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} ErrorResponse
// @Router /payment-terms/calculate/discount [post]
// @Security BearerAuth
func (h *PaymentTermsHandler) CalculateDiscount(c *gin.Context) {
	ctx := c.Request.Context()

	var req CalculateDiscountRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid request",
			Message: err.Error(),
		})
		return
	}

	discount, eligible, err := h.service.CalculateEarlyPaymentDiscount(
		ctx, req.InvoiceAmount, req.PaymentDate, req.IssueDate, req.PaymentTermID,
	)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Failed to calculate discount",
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"discount_amount":       discount,
		"eligible":              eligible,
		"invoice_amount":        req.InvoiceAmount,
		"amount_after_discount": req.InvoiceAmount.Sub(discount),
	})
}

// CalculateLateFee godoc
// @Summary Calculate late fee
// @Description Calculate the late fee for overdue payment
// @Tags payment-terms
// @Accept json
// @Produce json
// @Param request body CalculateLateFeeRequest true "Late fee calculation parameters"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} ErrorResponse
// @Router /payment-terms/calculate/late-fee [post]
// @Security BearerAuth
func (h *PaymentTermsHandler) CalculateLateFee(c *gin.Context) {
	ctx := c.Request.Context()

	var req CalculateLateFeeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid request",
			Message: err.Error(),
		})
		return
	}

	timezone := req.Timezone
	if timezone == "" {
		timezone = "UTC"
	}

	lateFee, applicable, err := h.service.CalculateLateFee(
		ctx, req.InvoiceAmount, req.CurrentDate, req.DueDate, req.PaymentTermID, timezone,
	)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Failed to calculate late fee",
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"late_fee_amount": lateFee,
		"applicable":      applicable,
		"invoice_amount":  req.InvoiceAmount,
		"total_amount":    req.InvoiceAmount.Add(lateFee),
	})
}

// GetPaymentStatus godoc
// @Summary Get payment status
// @Description Get the current payment status with notifications
// @Tags payment-terms
// @Accept json
// @Produce json
// @Param request body GetPaymentStatusRequest true "Payment status parameters"
// @Success 200 {object} services.PaymentStatus
// @Failure 400 {object} ErrorResponse
// @Router /payment-terms/calculate/status [post]
// @Security BearerAuth
func (h *PaymentTermsHandler) GetPaymentStatus(c *gin.Context) {
	ctx := c.Request.Context()

	var req GetPaymentStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid request",
			Message: err.Error(),
		})
		return
	}

	timezone := req.Timezone
	if timezone == "" {
		timezone = "UTC"
	}

	currentDate := time.Now()
	if req.CurrentDate != nil {
		currentDate = *req.CurrentDate
	}

	status, err := h.service.GetPaymentStatus(
		ctx, req.InvoiceAmount, req.IssueDate, req.DueDate, req.PaymentTermID, currentDate, timezone,
	)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Failed to get payment status",
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, status)
}

// CalculatePaymentDetails godoc
// @Summary Calculate payment details
// @Description Calculate comprehensive payment details including discounts and late fees
// @Tags payment-terms
// @Accept json
// @Produce json
// @Param request body CalculatePaymentDetailsRequest true "Payment details calculation parameters"
// @Success 200 {object} services.PaymentCalculationResult
// @Failure 400 {object} ErrorResponse
// @Router /payment-terms/calculate/details [post]
// @Security BearerAuth
func (h *PaymentTermsHandler) CalculatePaymentDetails(c *gin.Context) {
	ctx := c.Request.Context()

	var req CalculatePaymentDetailsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid request",
			Message: err.Error(),
		})
		return
	}

	timezone := req.Timezone
	if timezone == "" {
		timezone = "UTC"
	}

	details, err := h.service.CalculatePaymentDetails(
		ctx, req.InvoiceAmount, req.IssueDate, req.PaymentTermID, req.PaymentDate, timezone,
	)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Failed to calculate payment details",
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, details)
}
