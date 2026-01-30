package handlers

import (
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
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: err.Error()})
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

	// Convert line items and calculate totals
	invoice.Lines = make([]models.InvoiceLine, len(req.Lines))
	subtotal := decimal.Zero
	totalTax := decimal.Zero

	for i, lineReq := range req.Lines {
		taxable := true
		if lineReq.Taxable != nil {
			taxable = *lineReq.Taxable
		}

		// Apply tax rate from request if provided
		var lineTaxRate decimal.Decimal
		if req.TaxRate != nil {
			lineTaxRate = *req.TaxRate
		}

		// Calculate line total and tax
		lineTotal := lineReq.Quantity.Mul(lineReq.UnitPrice).Sub(lineReq.DiscountAmount)
		var lineTaxAmount decimal.Decimal
		if taxable {
			lineTaxAmount = lineTotal.Mul(lineTaxRate).Round(2)
		}

		subtotal = subtotal.Add(lineTotal)
		totalTax = totalTax.Add(lineTaxAmount)

		invoice.Lines[i] = models.InvoiceLine{
			Description:        lineReq.Description,
			Quantity:           lineReq.Quantity,
			UnitPrice:          lineReq.UnitPrice,
			UnitOfMeasure:      lineReq.UnitOfMeasure,
			DiscountPercentage: lineReq.DiscountPercentage,
			DiscountAmount:     lineReq.DiscountAmount,
			Taxable:            taxable,
			TaxRate:            lineTaxRate,
			TaxAmount:          lineTaxAmount,
			ProductID:          lineReq.ProductID,
			ServiceID:          lineReq.ServiceID,
			SortOrder:          i,
		}
	}

	// Set invoice totals
	invoice.Subtotal = subtotal
	invoice.TaxAmount = totalTax
	invoice.TotalAmount = subtotal.Add(totalTax).Sub(invoice.DiscountAmount)

	// Create invoice
	created, err := h.service.CreateInvoice(ctx, invoice, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
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
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: err.Error()})
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

	// Convert request to invoice model
	updates := &models.Invoice{}

	if req.Status != nil {
		updates.Status = *req.Status
	}

	if req.DueDate != nil {
		updates.DueDate = *req.DueDate
	}

	if req.PaymentTermID != nil {
		updates.PaymentTermID = req.PaymentTermID
	}

	if req.TaxRateID != nil {
		updates.TaxRateID = req.TaxRateID
	}

	if req.PONumber != nil {
		updates.PONumber = *req.PONumber
	}

	if req.Notes != nil {
		updates.Notes = *req.Notes
	}

	if req.DiscountAmount != nil {
		updates.DiscountAmount = *req.DiscountAmount
	}

	if req.TermsAndConditions != nil {
		updates.TermsAndConditions = *req.TermsAndConditions
	}

	// Convert line items if provided and calculate totals
	if req.Lines != nil {
		updates.Lines = make([]models.InvoiceLine, len(req.Lines))
		subtotal := decimal.Zero
		totalTax := decimal.Zero

		for i, lineReq := range req.Lines {
			taxable := true
			if lineReq.Taxable != nil {
				taxable = *lineReq.Taxable
			}

			// Apply tax rate from request if provided
			var lineTaxRate decimal.Decimal
			if req.TaxRate != nil {
				lineTaxRate = *req.TaxRate
			}

			// Calculate line total and tax
			lineTotal := lineReq.Quantity.Mul(lineReq.UnitPrice).Sub(lineReq.DiscountAmount)
			var lineTaxAmount decimal.Decimal
			if taxable {
				lineTaxAmount = lineTotal.Mul(lineTaxRate).Round(2)
			}

			subtotal = subtotal.Add(lineTotal)
			totalTax = totalTax.Add(lineTaxAmount)

			updates.Lines[i] = models.InvoiceLine{
				Description:        lineReq.Description,
				Quantity:           lineReq.Quantity,
				UnitPrice:          lineReq.UnitPrice,
				UnitOfMeasure:      lineReq.UnitOfMeasure,
				DiscountPercentage: lineReq.DiscountPercentage,
				DiscountAmount:     lineReq.DiscountAmount,
				Taxable:            taxable,
				TaxRate:            lineTaxRate,
				TaxAmount:          lineTaxAmount,
				ProductID:          lineReq.ProductID,
				ServiceID:          lineReq.ServiceID,
				SortOrder:          i,
			}
		}

		// Set invoice totals
		discountAmount := decimal.Zero
		if req.DiscountAmount != nil {
			discountAmount = *req.DiscountAmount
		}
		updates.Subtotal = subtotal
		updates.TaxAmount = totalTax
		updates.TotalAmount = subtotal.Add(totalTax).Sub(discountAmount)
	}

	// Update invoice
	updated, err := h.service.UpdateInvoice(ctx, id, updates, userID)
	if err != nil {
		if err == services.ErrInvoiceNotFound {
			c.JSON(http.StatusNotFound, ErrorResponse{Error: "Invoice not found"})
			return
		}
		if err == services.ErrInvoiceAlreadyPaid {
			c.JSON(http.StatusBadRequest, ErrorResponse{Error: err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
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
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: err.Error()})
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
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: err.Error()})
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
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: err.Error()})
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
		CreatedBy:       userID,
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
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: err.Error()})
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
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: err.Error()})
		return
	}

	invoice, err := h.service.CancelInvoice(ctx, id, req.Reason)
	if err != nil {
		if err == services.ErrInvoiceNotFound {
			c.JSON(http.StatusNotFound, ErrorResponse{Error: "Invoice not found"})
			return
		}
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: err.Error()})
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
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Failed to get invoice PDF: " + err.Error()})
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
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: err.Error()})
		return
	}

	// Stream PDF directly to client
	c.Header("Content-Type", "application/pdf")
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filename))
	c.Header("Content-Length", fmt.Sprintf("%d", len(content)))
	c.Data(http.StatusOK, "application/pdf", content)
}
