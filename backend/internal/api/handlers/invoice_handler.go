package handlers

import (
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"github.com/javaknight1/servicepro/backend/internal/models"
	"github.com/javaknight1/servicepro/backend/internal/services"
)

// ErrorResponse represents an error response
type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message,omitempty"`
}

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
	CustomerID     uuid.UUID               `json:"customer_id" binding:"required"`
	JobID          *uuid.UUID              `json:"job_id,omitempty"`
	QuoteID        *uuid.UUID              `json:"quote_id,omitempty"`
	IssueDate      *time.Time              `json:"issue_date,omitempty"`
	DueDate        *time.Time              `json:"due_date,omitempty"`
	PaymentTermID  *uuid.UUID              `json:"payment_term_id,omitempty"`
	TaxRateID      *uuid.UUID              `json:"tax_rate_id,omitempty"`
	TaxRate        *decimal.Decimal        `json:"tax_rate,omitempty"` // Direct tax rate (e.g., 0.07 for 7%)
	PONumber       string                  `json:"po_number,omitempty"`
	Notes          string                  `json:"notes,omitempty"`
	DiscountAmount decimal.Decimal         `json:"discount_amount,omitempty"`
	Lines          []CreateLineItemRequest `json:"lines" binding:"required,min=1,dive"`
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
	Status             *models.InvoiceStatus   `json:"status,omitempty"`
	DueDate            *time.Time              `json:"due_date,omitempty"`
	PaymentTermID      *uuid.UUID              `json:"payment_term_id,omitempty"`
	TaxRateID          *uuid.UUID              `json:"tax_rate_id,omitempty"`
	TaxRate            *decimal.Decimal        `json:"tax_rate,omitempty"` // Direct tax rate (e.g., 0.07 for 7%)
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
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /invoices [get]
// @Security BearerAuth
func (h *InvoiceHandler) ListInvoices(c *gin.Context) {
	ctx := c.Request.Context()

	// Parse query parameters
	filter := &models.InvoiceFilter{}

	if customerID := c.Query("customer_id"); customerID != "" {
		id, err := uuid.Parse(customerID)
		if err != nil {
			c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid customer_id"})
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
			c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid from_date format"})
			return
		}
		filter.FromDate = &date
	}

	if toDate := c.Query("to_date"); toDate != "" {
		date, err := time.Parse("2006-01-02", toDate)
		if err != nil {
			c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid to_date format"})
			return
		}
		filter.ToDate = &date
	}

	if minAmount := c.Query("min_amount"); minAmount != "" {
		amount, err := decimal.NewFromString(minAmount)
		if err != nil {
			c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid min_amount"})
			return
		}
		filter.MinAmount = &amount
	}

	if maxAmount := c.Query("max_amount"); maxAmount != "" {
		amount, err := decimal.NewFromString(maxAmount)
		if err != nil {
			c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid max_amount"})
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
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Failed to list invoices"})
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
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /invoices/{id} [get]
// @Security BearerAuth
func (h *InvoiceHandler) GetInvoice(c *gin.Context) {
	ctx := c.Request.Context()

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid invoice ID"})
		return
	}

	invoice, err := h.service.GetInvoice(ctx, id)
	if err != nil {
		if err == services.ErrInvoiceNotFound {
			c.JSON(http.StatusNotFound, ErrorResponse{Error: "Invoice not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Failed to get invoice"})
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
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /invoices [post]
// @Security BearerAuth
func (h *InvoiceHandler) CreateInvoice(c *gin.Context) {
	ctx := c.Request.Context()

	var req CreateInvoiceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: publicError(err)})
		return
	}

	// Validate line items (decimal.Decimal doesn't support gt/gte validators)
	for i, line := range req.Lines {
		if line.Quantity.LessThanOrEqual(decimal.Zero) {
			c.JSON(http.StatusBadRequest, ErrorResponse{Error: fmt.Sprintf("Line %d: quantity must be greater than 0", i+1)})
			return
		}
		if line.UnitPrice.LessThan(decimal.Zero) {
			c.JSON(http.StatusBadRequest, ErrorResponse{Error: fmt.Sprintf("Line %d: unit_price must be non-negative", i+1)})
			return
		}
	}

	// Get user ID from context (set by auth middleware - uses "user_id" key)
	userIDValue, exists := c.Get("user_id")
	if !exists || userIDValue == nil {
		c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "Unauthorized"})
		return
	}
	userID, ok := userIDValue.(uuid.UUID)
	if !ok {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Invalid user ID in context"})
		return
	}

	// Convert request to invoice model
	invoice := &models.Invoice{
		CustomerID:     req.CustomerID,
		JobID:          req.JobID,
		QuoteID:        req.QuoteID,
		PaymentTermID:  req.PaymentTermID,
		TaxRateID:      req.TaxRateID,
		PONumber:       req.PONumber,
		Notes:          req.Notes,
		DiscountAmount: req.DiscountAmount,
	}

	if req.IssueDate != nil {
		invoice.IssueDate = *req.IssueDate
	}

	if req.DueDate != nil {
		invoice.DueDate = *req.DueDate
	}

	// Totals, line discounts and taxes are computed by the service.
	invoice.Lines = toInvoiceLines(req.Lines)
	invoice.ApplyTaxRate = req.TaxRate

	// Create invoice
	created, err := h.service.CreateInvoice(ctx, invoice, userID)
	if err != nil {
		c.JSON(invoiceErrorStatus(err), ErrorResponse{Error: publicError(err)})
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
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /invoices/{id} [put]
// @Security BearerAuth
func (h *InvoiceHandler) UpdateInvoice(c *gin.Context) {
	ctx := c.Request.Context()

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid invoice ID"})
		return
	}

	var req UpdateInvoiceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: publicError(err)})
		return
	}

	// Get user ID from context (set by auth middleware - uses "user_id" key)
	userIDValue, exists := c.Get("user_id")
	if !exists || userIDValue == nil {
		c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "Unauthorized"})
		return
	}
	userID, ok := userIDValue.(uuid.UUID)
	if !ok {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Invalid user ID in context"})
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
		c.JSON(invoiceErrorStatus(err), ErrorResponse{Error: publicError(err)})
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
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /invoices/{id} [delete]
// @Security BearerAuth
func (h *InvoiceHandler) DeleteInvoice(c *gin.Context) {
	ctx := c.Request.Context()

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid invoice ID"})
		return
	}

	if err := h.service.DeleteInvoice(ctx, id); err != nil {
		if err == services.ErrInvoiceNotFound {
			c.JSON(http.StatusNotFound, ErrorResponse{Error: "Invoice not found"})
			return
		}
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: publicError(err)})
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
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /invoices/{id}/send [post]
// @Security BearerAuth
func (h *InvoiceHandler) SendInvoice(c *gin.Context) {
	ctx := c.Request.Context()

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid invoice ID"})
		return
	}

	invoice, err := h.service.SendInvoice(ctx, id)
	if err != nil {
		if err == services.ErrInvoiceNotFound {
			c.JSON(http.StatusNotFound, ErrorResponse{Error: "Invoice not found"})
			return
		}
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: publicError(err)})
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
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /invoices/{id}/payments [post]
// @Security BearerAuth
func (h *InvoiceHandler) RecordPayment(c *gin.Context) {
	ctx := c.Request.Context()

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid invoice ID"})
		return
	}

	var req RecordPaymentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: publicError(err)})
		return
	}

	// Get user ID from context (set by auth middleware - uses "user_id" key)
	userIDValue, exists := c.Get("user_id")
	if !exists || userIDValue == nil {
		c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "Unauthorized"})
		return
	}
	userID, ok := userIDValue.(uuid.UUID)
	if !ok {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Invalid user ID in context"})
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
		if err == services.ErrInvoiceNotFound {
			c.JSON(http.StatusNotFound, ErrorResponse{Error: "Invoice not found"})
			return
		}
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: publicError(err)})
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
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /invoices/{id}/cancel [post]
// @Security BearerAuth
func (h *InvoiceHandler) CancelInvoice(c *gin.Context) {
	ctx := c.Request.Context()

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid invoice ID"})
		return
	}

	var req struct {
		Reason string `json:"reason" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: publicError(err)})
		return
	}

	invoice, err := h.service.CancelInvoice(ctx, id, req.Reason)
	if err != nil {
		if err == services.ErrInvoiceNotFound {
			c.JSON(http.StatusNotFound, ErrorResponse{Error: "Invoice not found"})
			return
		}
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: publicError(err)})
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
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /invoices/{id}/pdf [get]
// @Security BearerAuth
func (h *InvoiceHandler) GetInvoicePDF(c *gin.Context) {
	ctx := c.Request.Context()

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid invoice ID"})
		return
	}

	content, filename, err := h.service.DownloadInvoicePDF(ctx, id)
	if err != nil {
		if err == services.ErrInvoiceNotFound {
			c.JSON(http.StatusNotFound, ErrorResponse{Error: "Invoice not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Failed to get invoice PDF: " + publicError(err)})
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
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /invoices/{id}/receipt/pdf [get]
// @Security BearerAuth
func (h *InvoiceHandler) GetReceiptPDF(c *gin.Context) {
	ctx := c.Request.Context()

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid invoice ID"})
		return
	}

	content, filename, err := h.service.DownloadReceiptPDF(ctx, id)
	if err != nil {
		if err == services.ErrInvoiceNotFound {
			c.JSON(http.StatusNotFound, ErrorResponse{Error: "Invoice not found"})
			return
		}
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: publicError(err)})
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

// invoiceErrorStatus maps invoice service errors to HTTP status codes:
// not-found 404, business-rule violations 400, anything else 500.
func invoiceErrorStatus(err error) int {
	switch {
	case errors.Is(err, services.ErrInvoiceNotFound), errors.Is(err, services.ErrCustomerNotFound):
		return http.StatusNotFound
	case errors.Is(err, services.ErrInvoiceAlreadyPaid),
		errors.Is(err, services.ErrInvalidLineItem),
		errors.Is(err, services.ErrInvalidDiscount),
		errors.Is(err, services.ErrInvalidTaxRate),
		errors.Is(err, services.ErrTaxRateNotFound):
		return http.StatusBadRequest
	}
	var rule *services.InvoiceRuleError
	if errors.As(err, &rule) {
		return http.StatusBadRequest
	}
	return http.StatusInternalServerError
}
