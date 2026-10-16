package handlers

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"github.com/javaknight1/servicepro/backend/internal/api/apierror"
	"github.com/javaknight1/servicepro/backend/internal/models"
	"github.com/javaknight1/servicepro/backend/internal/services"
)

// InvoiceHandler handles invoice HTTP requests
type InvoiceHandler struct {
	service *services.InvoiceService
}

// NewInvoiceHandler creates a new invoice handler
func NewInvoiceHandler(service *services.InvoiceService) *InvoiceHandler {
	return &InvoiceHandler{service: service}
}

// CreateInvoiceRequest represents the request to create an invoice
type CreateInvoiceRequest struct {
	CustomerID         uuid.UUID               `json:"customer_id" binding:"required"`
	JobID              *uuid.UUID              `json:"job_id,omitempty"`
	QuoteID            *uuid.UUID              `json:"quote_id,omitempty"`
	IssueDate          *time.Time              `json:"issue_date,omitempty"`
	DueDate            *time.Time              `json:"due_date,omitempty"`
	PaymentTermID      *uuid.UUID              `json:"payment_term_id,omitempty"`
	TaxRateID          *uuid.UUID              `json:"tax_rate_id,omitempty"`
	TaxRate            *decimal.Decimal        `json:"tax_rate,omitempty"` // Direct tax rate (e.g., 0.07 for 7%)
	PONumber           string                  `json:"po_number,omitempty"`
	Notes              string                  `json:"notes,omitempty"`
	TermsAndConditions string                  `json:"terms_and_conditions,omitempty"` // printed on the invoice
	DiscountAmount     decimal.Decimal         `json:"discount_amount,omitempty"`
	Lines              []CreateLineItemRequest `json:"lines" binding:"required,min=1,dive"`
}

// toInvoice maps the request onto a new invoice. Totals, line discounts and
// taxes are computed by the service.
func (req *CreateInvoiceRequest) toInvoice() *models.Invoice {
	invoice := &models.Invoice{
		CustomerID:         req.CustomerID,
		JobID:              req.JobID,
		QuoteID:            req.QuoteID,
		PaymentTermID:      req.PaymentTermID,
		TaxRateID:          req.TaxRateID,
		PONumber:           req.PONumber,
		Notes:              req.Notes,
		TermsAndConditions: req.TermsAndConditions,
		DiscountAmount:     req.DiscountAmount,
	}
	if req.IssueDate != nil {
		invoice.IssueDate = *req.IssueDate
	}
	if req.DueDate != nil {
		invoice.DueDate = *req.DueDate
	}
	invoice.Lines = toInvoiceLines(req.Lines)
	invoice.ApplyTaxRate = req.TaxRate
	return invoice
}

// CreateLineItemRequest represents a line item in create request
type CreateLineItemRequest struct {
	Description        string          `json:"description" binding:"required"`
	Quantity           decimal.Decimal `json:"quantity" binding:"required"`
	UnitPrice          decimal.Decimal `json:"unit_price" binding:"required"`
	UnitOfMeasure      string          `json:"unit_of_measure,omitempty"`
	DiscountPercentage decimal.Decimal `json:"discount_percentage,omitempty"`
	DiscountAmount     decimal.Decimal `json:"discount_amount,omitempty"`
	Taxable            *bool           `json:"taxable,omitempty"`
	ProductID          *uuid.UUID      `json:"product_id,omitempty"`
	ServiceID          *uuid.UUID      `json:"service_id,omitempty"`
}

// UpdateInvoiceRequest represents the request to update an invoice
type UpdateInvoiceRequest struct {
	// Status can not be set: it changes only through send, cancel and payments
	// (T124). Sending it is rejected with 400.
	Status             *models.InvoiceStatus   `json:"status,omitempty" swaggerignore:"true"`
	DueDate            *time.Time              `json:"due_date,omitempty"`
	PaymentTermID      *uuid.UUID              `json:"payment_term_id,omitempty"`
	TaxRateID          *uuid.UUID              `json:"tax_rate_id,omitempty"`
	TaxRate            *decimal.Decimal        `json:"tax_rate,omitempty"` // Direct tax rate (e.g., 0.07 for 7%); without tax_rate_id it clears the invoice's tax_rate_id
	PONumber           *string                 `json:"po_number,omitempty"`
	Notes              *string                 `json:"notes,omitempty"`
	DiscountAmount     *decimal.Decimal        `json:"discount_amount,omitempty"`
	TermsAndConditions *string                 `json:"terms_and_conditions,omitempty"`
	Lines              []CreateLineItemRequest `json:"lines,omitempty"`
}

// RecordPaymentRequest represents a payment recording request
type RecordPaymentRequest struct {
	Amount          decimal.Decimal `json:"amount" binding:"required,gt=0"`
	PaymentDate     *time.Time      `json:"payment_date,omitempty"`
	PaymentMethod   string          `json:"payment_method,omitempty"`
	ReferenceNumber string          `json:"reference_number,omitempty"`
	Notes           string          `json:"notes,omitempty"`
}

// InvoiceResponse represents the invoice API response
type InvoiceResponse struct {
	*models.Invoice
	IsOverdue   bool `json:"is_overdue"`
	DaysOverdue int  `json:"days_overdue"`
}

// ListInvoices godoc
// @Summary List invoices
// @Description Get a paginated list of invoices with optional filters
// @Tags invoices
// @Accept json
// @Produce json
// @Param customer_id query string false "Filter by customer ID"
// @Param status query string false "Filter by status (draft, sent, viewed, paid, etc.)"
// @Param from_date query string false "Filter by issue date from (YYYY-MM-DD)"
// @Param to_date query string false "Filter by issue date to (YYYY-MM-DD)"
// @Param min_amount query number false "Filter by minimum amount"
// @Param max_amount query number false "Filter by maximum amount"
// @Param is_overdue query boolean false "Filter overdue invoices"
// @Param search query string false "Search by invoice number or notes"
// @Param page query integer false "Page number" default(1)
// @Param page_size query integer false "Page size" default(20)
// @Param sort_by query string false "Sort field" default(created_at)
// @Param sort_order query string false "Sort order (asc/desc)" default(desc)
// @Success 200 {object} models.InvoiceListResponse
// @Failure 400 {object} models.ErrorResponse
// @Failure 401 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Router /invoices [get]
// @Security BearerAuth
func (h *InvoiceHandler) ListInvoices(c *gin.Context) {
	ctx := c.Request.Context()

	// Parse query parameters
	filter := &models.InvoiceFilter{}

	if customerID := c.Query("customer_id"); customerID != "" {
		id, err := uuid.Parse(customerID)
		if err != nil {
			apierror.WriteResponse(c, http.StatusBadRequest, models.ErrorResponse{Error: "Invalid customer_id"})
			return
		}
		filter.CustomerID = &id
	}

	if status := c.Query("status"); status != "" {
		s := models.InvoiceStatus(status)
		filter.Status = &s
	}

	if fromDate := c.Query("from_date"); fromDate != "" {
		date, err := time.Parse("2006-01-02", fromDate)
		if err != nil {
			apierror.WriteResponse(c, http.StatusBadRequest, models.ErrorResponse{Error: "Invalid from_date format"})
			return
		}
		filter.FromDate = &date
	}

	if toDate := c.Query("to_date"); toDate != "" {
		date, err := time.Parse("2006-01-02", toDate)
		if err != nil {
			apierror.WriteResponse(c, http.StatusBadRequest, models.ErrorResponse{Error: "Invalid to_date format"})
			return
		}
		filter.ToDate = &date
	}

	if minAmount := c.Query("min_amount"); minAmount != "" {
		amount, err := decimal.NewFromString(minAmount)
		if err != nil {
			apierror.WriteResponse(c, http.StatusBadRequest, models.ErrorResponse{Error: "Invalid min_amount"})
			return
		}
		filter.MinAmount = &amount
	}

	if maxAmount := c.Query("max_amount"); maxAmount != "" {
		amount, err := decimal.NewFromString(maxAmount)
		if err != nil {
			apierror.WriteResponse(c, http.StatusBadRequest, models.ErrorResponse{Error: "Invalid max_amount"})
			return
		}
		filter.MaxAmount = &amount
	}

	if isOverdue := c.Query("is_overdue"); isOverdue != "" {
		overdue := isOverdue == "true"
		filter.IsOverdue = &overdue
	}

	filter.Search = c.Query("search")

	// Pagination
	if page := c.Query("page"); page != "" {
		var p int
		if _, err := fmt.Sscanf(page, "%d", &p); err == nil {
			filter.Page = p
		}
	}

	if pageSize := c.Query("page_size"); pageSize != "" {
		var ps int
		if _, err := fmt.Sscanf(pageSize, "%d", &ps); err == nil {
			filter.PageSize = ps
		}
	}

	filter.SortBy = c.DefaultQuery("sort_by", "created_at")
	filter.SortOrder = c.DefaultQuery("sort_order", "desc")

	// Get invoices
	response, err := h.service.ListInvoices(ctx, filter)
	if err != nil {
		apierror.WriteResponse(c, http.StatusInternalServerError, models.ErrorResponse{Error: "Failed to list invoices"})
		return
	}

	c.JSON(http.StatusOK, response)
}

// GetInvoice godoc
// @Summary Get invoice
// @Description Get a single invoice by ID
// @Tags invoices
// @Accept json
// @Produce json
// @Param id path string true "Invoice ID"
// @Success 200 {object} InvoiceResponse
// @Failure 400 {object} models.ErrorResponse
// @Failure 404 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Router /invoices/{id} [get]
// @Security BearerAuth
func (h *InvoiceHandler) GetInvoice(c *gin.Context) {
	ctx := c.Request.Context()

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		apierror.WriteResponse(c, http.StatusBadRequest, models.ErrorResponse{Error: "Invalid invoice ID"})
		return
	}

	invoice, err := h.service.GetInvoice(ctx, id)
	if err != nil {
		if errors.Is(err, services.ErrInvoiceNotFound) {
			apierror.WriteResponse(c, http.StatusNotFound, models.ErrorResponse{Error: "Invoice not found"})
			return
		}
		apierror.WriteResponse(c, http.StatusInternalServerError, models.ErrorResponse{Error: "Failed to get invoice"})
		return
	}

	// Calculate overdue status
	response := &InvoiceResponse{
		Invoice:     invoice,
		IsOverdue:   false,
		DaysOverdue: 0,
	}

	if invoice.DueDate.Before(time.Now()) && invoice.AmountDue.GreaterThan(decimal.Zero) {
		response.IsOverdue = true
		response.DaysOverdue = int(time.Since(invoice.DueDate).Hours() / 24)
	}

	c.JSON(http.StatusOK, response)
}

// CreateInvoice godoc
// @Summary Create invoice
// @Description Create a new invoice
// @Tags invoices
// @Accept json
// @Produce json
// @Param invoice body CreateInvoiceRequest true "Invoice data"
// @Success 201 {object} models.Invoice
// @Failure 400 {object} models.ErrorResponse
// @Failure 401 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Router /invoices [post]
// @Security BearerAuth
func (h *InvoiceHandler) CreateInvoice(c *gin.Context) {
	ctx := c.Request.Context()

	var req CreateInvoiceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		apierror.BindError(c, err)
		return
	}

	// Validate line items (decimal.Decimal doesn't support gt/gte validators)
	for i, line := range req.Lines {
		if line.Quantity.LessThanOrEqual(decimal.Zero) {
			apierror.WriteResponse(c, http.StatusBadRequest, models.ErrorResponse{Error: fmt.Sprintf("Line %d: quantity must be greater than 0", i+1)})
			return
		}
		if line.UnitPrice.LessThan(decimal.Zero) {
			apierror.WriteResponse(c, http.StatusBadRequest, models.ErrorResponse{Error: fmt.Sprintf("Line %d: unit_price must be non-negative", i+1)})
			return
		}
	}

	// Get user ID from context (set by auth middleware - uses "user_id" key)
	userIDValue, exists := c.Get("user_id")
	if !exists || userIDValue == nil {
		apierror.WriteResponse(c, http.StatusUnauthorized, models.ErrorResponse{Error: "Unauthorized"})
		return
	}
	userID, ok := userIDValue.(uuid.UUID)
	if !ok {
		apierror.WriteResponse(c, http.StatusInternalServerError, models.ErrorResponse{Error: "Invalid user ID in context"})
		return
	}

	created, err := h.service.CreateInvoice(ctx, req.toInvoice(), userID)
	if err != nil {
		respondInvoiceError(c, err, "Failed to create invoice")
		return
	}

	c.JSON(http.StatusCreated, created)
}

// UpdateInvoice godoc
// @Summary Update invoice
// @Description Update an existing invoice
// @Tags invoices
// @Accept json
// @Produce json
// @Param id path string true "Invoice ID"
// @Param invoice body UpdateInvoiceRequest true "Invoice updates"
// @Success 200 {object} models.Invoice
// @Failure 400 {object} models.ErrorResponse
// @Failure 404 {object} models.ErrorResponse
// @Failure 409 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Router /invoices/{id} [put]
// @Security BearerAuth
func (h *InvoiceHandler) UpdateInvoice(c *gin.Context) {
	ctx := c.Request.Context()

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		apierror.WriteResponse(c, http.StatusBadRequest, models.ErrorResponse{Error: "Invalid invoice ID"})
		return
	}

	var req UpdateInvoiceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		apierror.BindError(c, err)
		return
	}
	if req.Status != nil {
		// {"status":"paid"} used to mark an invoice paid with no money
		// received, and revived cancelled invoices (audit 7.H2).
		apierror.WriteResponse(c, http.StatusBadRequest, models.ErrorResponse{Error: "status can't be set directly; send, cancel or record a payment instead"})
		return
	}

	// Get user ID from context (set by auth middleware - uses "user_id" key)
	userIDValue, exists := c.Get("user_id")
	if !exists || userIDValue == nil {
		apierror.WriteResponse(c, http.StatusUnauthorized, models.ErrorResponse{Error: "Unauthorized"})
		return
	}
	userID, ok := userIDValue.(uuid.UUID)
	if !ok {
		apierror.WriteResponse(c, http.StatusInternalServerError, models.ErrorResponse{Error: "Invalid user ID in context"})
		return
	}

	// Totals are recomputed by the service whatever changed.
	upd := &services.InvoiceUpdate{
		DueDate:            req.DueDate,
		PaymentTermID:      req.PaymentTermID,
		TaxRateID:          req.TaxRateID,
		TaxRate:            req.TaxRate,
		PONumber:           req.PONumber,
		Notes:              req.Notes,
		TermsAndConditions: req.TermsAndConditions,
		DiscountAmount:     req.DiscountAmount,
	}
	if req.Lines != nil {
		upd.Lines = toInvoiceLines(req.Lines)
	}

	updated, err := h.service.UpdateInvoice(ctx, id, upd, userID)
	if err != nil {
		respondInvoiceError(c, err, "Failed to update invoice")
		return
	}

	c.JSON(http.StatusOK, updated)
}

// DeleteInvoice godoc
// @Summary Delete invoice
// @Description Soft delete an invoice
// @Tags invoices
// @Accept json
// @Produce json
// @Param id path string true "Invoice ID"
// @Success 204
// @Failure 400 {object} models.ErrorResponse
// @Failure 404 {object} models.ErrorResponse
// @Failure 409 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Router /invoices/{id} [delete]
// @Security BearerAuth
func (h *InvoiceHandler) DeleteInvoice(c *gin.Context) {
	ctx := c.Request.Context()

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		apierror.WriteResponse(c, http.StatusBadRequest, models.ErrorResponse{Error: "Invalid invoice ID"})
		return
	}

	if err := h.service.DeleteInvoice(ctx, id); err != nil {
		if errors.Is(err, services.ErrInvoiceNotFound) {
			apierror.WriteResponse(c, http.StatusNotFound, models.ErrorResponse{Error: "Invoice not found"})
			return
		}
		respondInvoiceError(c, err, "Failed to delete invoice")
		return
	}

	c.Status(http.StatusNoContent)
}

// SendInvoice godoc
// @Summary Send invoice
// @Description Mark invoice as sent and update sent_date
// @Tags invoices
// @Accept json
// @Produce json
// @Param id path string true "Invoice ID"
// @Success 200 {object} models.Invoice
// @Failure 400 {object} models.ErrorResponse
// @Failure 404 {object} models.ErrorResponse
// @Failure 409 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Router /invoices/{id}/send [post]
// @Security BearerAuth
func (h *InvoiceHandler) SendInvoice(c *gin.Context) {
	ctx := c.Request.Context()

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		apierror.WriteResponse(c, http.StatusBadRequest, models.ErrorResponse{Error: "Invalid invoice ID"})
		return
	}

	invoice, err := h.service.SendInvoice(ctx, id)
	if err != nil {
		if errors.Is(err, services.ErrInvoiceNotFound) {
			apierror.WriteResponse(c, http.StatusNotFound, models.ErrorResponse{Error: "Invoice not found"})
			return
		}
		respondInvoiceError(c, err, "Failed to send invoice")
		return
	}

	c.JSON(http.StatusOK, invoice)
}

// RecordPayment godoc
// @Summary Record payment
// @Description Record a payment for an invoice
// @Tags invoices
// @Accept json
// @Produce json
// @Param id path string true "Invoice ID"
// @Param payment body RecordPaymentRequest true "Payment data"
// @Success 201 {object} models.InvoicePayment
// @Failure 400 {object} models.ErrorResponse
// @Failure 404 {object} models.ErrorResponse
// @Failure 409 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Router /invoices/{id}/payments [post]
// @Security BearerAuth
func (h *InvoiceHandler) RecordPayment(c *gin.Context) {
	ctx := c.Request.Context()

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		apierror.WriteResponse(c, http.StatusBadRequest, models.ErrorResponse{Error: "Invalid invoice ID"})
		return
	}

	var req RecordPaymentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		apierror.BindError(c, err)
		return
	}

	// Get user ID from context (set by auth middleware - uses "user_id" key)
	userIDValue, exists := c.Get("user_id")
	if !exists || userIDValue == nil {
		apierror.WriteResponse(c, http.StatusUnauthorized, models.ErrorResponse{Error: "Unauthorized"})
		return
	}
	userID, ok := userIDValue.(uuid.UUID)
	if !ok {
		apierror.WriteResponse(c, http.StatusInternalServerError, models.ErrorResponse{Error: "Invalid user ID in context"})
		return
	}

	// Create payment
	payment := &models.InvoicePayment{
		InvoiceID:       id,
		Amount:          req.Amount,
		PaymentMethod:   req.PaymentMethod,
		ReferenceNumber: req.ReferenceNumber,
		Notes:           req.Notes,
		CreatedBy:       &userID,
	}

	if req.PaymentDate != nil {
		payment.PaymentDate = *req.PaymentDate
	} else {
		payment.PaymentDate = time.Now()
	}

	created, err := h.service.RecordPayment(ctx, payment)
	if err != nil {
		if errors.Is(err, services.ErrInvoiceNotFound) {
			apierror.WriteResponse(c, http.StatusNotFound, models.ErrorResponse{Error: "Invoice not found"})
			return
		}
		respondInvoiceError(c, err, "Failed to record payment")
		return
	}

	c.JSON(http.StatusCreated, created)
}

// CancelInvoice godoc
// @Summary Cancel invoice
// @Description Cancel an invoice with a reason
// @Tags invoices
// @Accept json
// @Produce json
// @Param id path string true "Invoice ID"
// @Param body body map[string]string true "Cancellation reason"
// @Success 200 {object} models.Invoice
// @Failure 400 {object} models.ErrorResponse
// @Failure 404 {object} models.ErrorResponse
// @Failure 409 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Router /invoices/{id}/cancel [post]
// @Security BearerAuth
func (h *InvoiceHandler) CancelInvoice(c *gin.Context) {
	ctx := c.Request.Context()

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		apierror.WriteResponse(c, http.StatusBadRequest, models.ErrorResponse{Error: "Invalid invoice ID"})
		return
	}

	var req struct {
		Reason string `json:"reason" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		apierror.BindError(c, err)
		return
	}

	invoice, err := h.service.CancelInvoice(ctx, id, req.Reason)
	if err != nil {
		if errors.Is(err, services.ErrInvoiceNotFound) {
			apierror.WriteResponse(c, http.StatusNotFound, models.ErrorResponse{Error: "Invoice not found"})
			return
		}
		respondInvoiceError(c, err, "Failed to cancel invoice")
		return
	}

	c.JSON(http.StatusOK, invoice)
}

// GetInvoicePDF godoc
// @Summary Get invoice PDF download URL
// @Description Returns a presigned URL for downloading the invoice PDF
// @Tags invoices
// @Accept json
// @Produce json
// @Param id path string true "Invoice ID"
// @Success 200 {file} binary "PDF file"
// @Failure 400 {object} models.ErrorResponse
// @Failure 404 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Router /invoices/{id}/pdf [get]
// @Security BearerAuth
func (h *InvoiceHandler) GetInvoicePDF(c *gin.Context) {
	ctx := c.Request.Context()

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		apierror.WriteResponse(c, http.StatusBadRequest, models.ErrorResponse{Error: "Invalid invoice ID"})
		return
	}

	content, filename, err := h.service.DownloadInvoicePDF(ctx, id)
	if err != nil {
		if errors.Is(err, services.ErrInvoiceNotFound) {
			apierror.WriteResponse(c, http.StatusNotFound, models.ErrorResponse{Error: "Invoice not found"})
			return
		}
		apierror.Internal(c, err, "Failed to get invoice PDF")
		return
	}

	// Stream PDF directly to client
	c.Header("Content-Type", "application/pdf")
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filename))
	c.Header("Content-Length", fmt.Sprintf("%d", len(content)))
	c.Data(http.StatusOK, "application/pdf", content)
}

// GetReceiptPDF godoc
// @Summary Get receipt PDF
// @Description Downloads the payment receipt PDF directly
// @Tags invoices
// @Accept json
// @Produce application/pdf
// @Param id path string true "Invoice ID"
// @Success 200 {file} binary "PDF file"
// @Failure 400 {object} models.ErrorResponse
// @Failure 404 {object} models.ErrorResponse
// @Failure 409 {object} models.ErrorResponse "Invoice is not paid"
// @Failure 500 {object} models.ErrorResponse
// @Router /invoices/{id}/receipt/pdf [get]
// @Security BearerAuth
func (h *InvoiceHandler) GetReceiptPDF(c *gin.Context) {
	ctx := c.Request.Context()

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		apierror.WriteResponse(c, http.StatusBadRequest, models.ErrorResponse{Error: "Invalid invoice ID"})
		return
	}

	content, filename, err := h.service.DownloadReceiptPDF(ctx, id)
	if err != nil {
		if errors.Is(err, services.ErrInvoiceNotFound) {
			apierror.WriteResponse(c, http.StatusNotFound, models.ErrorResponse{Error: "Invoice not found"})
			return
		}
		apierror.Respond(c, err, "Failed to get receipt PDF")
		return
	}

	// Stream PDF directly to client
	c.Header("Content-Type", "application/pdf")
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filename))
	c.Header("Content-Length", fmt.Sprintf("%d", len(content)))
	c.Data(http.StatusOK, "application/pdf", content)
}

// toInvoiceLines maps request lines to models; money fields are filled in by
// the service's totals calculation. Lines are taxable unless taxable=false.
func toInvoiceLines(reqLines []CreateLineItemRequest) []models.InvoiceLine {
	lines := make([]models.InvoiceLine, len(reqLines))
	for i, l := range reqLines {
		taxable := l.Taxable == nil || *l.Taxable
		lines[i] = models.InvoiceLine{
			Description:        l.Description,
			Quantity:           l.Quantity,
			UnitPrice:          l.UnitPrice,
			UnitOfMeasure:      l.UnitOfMeasure,
			DiscountPercentage: l.DiscountPercentage,
			DiscountAmount:     l.DiscountAmount,
			Taxable:            taxable,
			ProductID:          l.ProductID,
			ServiceID:          l.ServiceID,
			SortOrder:          i,
		}
	}
	return lines
}

// invoiceErrors maps invoice service sentinels to a status and code:
// not-found 404, closed/forbidden status changes 409, business-rule
// violations 400. Anything else goes through apierror.Respond (typed
// domain error, or a logged 500).
var invoiceErrors = []struct {
	err    error
	status int
	code   string
}{
	{services.ErrInvoiceNotFound, http.StatusNotFound, "not_found"},
	{services.ErrCustomerNotFound, http.StatusNotFound, "customer_not_found"},
	{services.ErrInvoiceClosed, http.StatusConflict, "invoice_closed"},
	{services.ErrInvalidInvoiceTransition, http.StatusConflict, "invalid_status_transition"},
	{services.ErrInvoiceAlreadyPaid, http.StatusBadRequest, "invoice_already_paid"},
	{services.ErrInvoiceHasPayments, http.StatusBadRequest, "invoice_has_payments"},
	{services.ErrPaymentNotAllowed, http.StatusBadRequest, "payment_not_allowed"},
	{services.ErrOverpayment, http.StatusBadRequest, "overpayment"},
	{services.ErrRefundExceedsPaid, http.StatusBadRequest, "refund_exceeds_paid"},
	{services.ErrRefundNotAllowed, http.StatusBadRequest, "refund_not_allowed"},
	{services.ErrInvalidLineItem, http.StatusBadRequest, "invalid_line_item"},
	{services.ErrInvalidDiscount, http.StatusBadRequest, "invalid_discount"},
	{services.ErrInvalidTaxRate, http.StatusBadRequest, "invalid_tax_rate"},
	{services.ErrTaxRateNotFound, http.StatusBadRequest, "tax_rate_not_found"},
	{services.ErrAlreadyInvoiced, http.StatusBadRequest, "already_invoiced"},
	{services.ErrInvoiceSourceNotFound, http.StatusBadRequest, "invoice_source_not_found"},
	{services.ErrInvoiceSourceCustomer, http.StatusBadRequest, "invoice_source_customer"},
	{services.ErrInvoiceSourceStatus, http.StatusBadRequest, "invoice_source_status"},
	{services.ErrQuoteNotAccepted, http.StatusBadRequest, "quote_not_accepted"},
}

// respondInvoiceError writes an invoice service error in the standard
// envelope. Business-rule messages are fixed, user-safe sentences ("payment
// amount exceeds outstanding balance", "invalid line item on line 2 - …"),
// so the full text is kept; internalMessage is used only for a 500.
func respondInvoiceError(c *gin.Context, err error, internalMessage string) {
	for _, e := range invoiceErrors {
		if errors.Is(err, e.err) {
			msg := publicError(err)
			if msg == genericErrorMessage {
				msg = e.err.Error()
			}
			apierror.Write(c, e.status, e.code, sentenceCase(msg))
			return
		}
	}
	var rule *services.InvoiceRuleError
	if errors.As(err, &rule) {
		apierror.Write(c, http.StatusBadRequest, "invalid_invoice", sentenceCase(rule.Error()))
		return
	}
	apierror.Respond(c, err, internalMessage)
}

// sentenceCase capitalises the first letter of a service message.
func sentenceCase(msg string) string {
	if msg == "" {
		return msg
	}
	return strings.ToUpper(msg[:1]) + msg[1:]
}

// RecordRefundRequest records money returned to the customer
type RecordRefundRequest struct {
	Amount     decimal.Decimal `json:"amount" binding:"required"`
	RefundDate *time.Time      `json:"refund_date,omitempty"`
	Method     string          `json:"method,omitempty"`
	Reason     string          `json:"reason,omitempty"`
}

// RecordRefund godoc
// @Summary Record refund
// @Description Record money returned to the customer (cash, check, or a refund made outside the Stripe webhook). Lowers amount_paid; a refund to zero marks the invoice refunded. Stripe refunds are recorded automatically from charge.refunded.
// @Tags invoices
// @Accept json
// @Produce json
// @Param id path string true "Invoice ID"
// @Param refund body RecordRefundRequest true "Refund data"
// @Success 201 {object} models.InvoiceRefund
// @Failure 400 {object} models.ErrorResponse
// @Failure 404 {object} models.ErrorResponse
// @Failure 409 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Router /invoices/{id}/refunds [post]
// @Security BearerAuth
func (h *InvoiceHandler) RecordRefund(c *gin.Context) {
	ctx := c.Request.Context()

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		apierror.WriteResponse(c, http.StatusBadRequest, models.ErrorResponse{Error: "Invalid invoice ID"})
		return
	}
	var req RecordRefundRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		apierror.BindError(c, err)
		return
	}
	userIDValue, exists := c.Get("user_id")
	userID, ok := userIDValue.(uuid.UUID)
	if !exists || !ok {
		apierror.WriteResponse(c, http.StatusUnauthorized, models.ErrorResponse{Error: "Unauthorized"})
		return
	}

	refund := &models.InvoiceRefund{
		InvoiceID: id,
		Amount:    req.Amount,
		Method:    req.Method,
		Reason:    req.Reason,
		CreatedBy: &userID,
	}
	if req.RefundDate != nil {
		refund.RefundDate = *req.RefundDate
	}
	created, err := h.service.RecordRefund(ctx, refund)
	if err != nil {
		respondInvoiceError(c, err, "Failed to record refund")
		return
	}
	c.JSON(http.StatusCreated, created)
}
