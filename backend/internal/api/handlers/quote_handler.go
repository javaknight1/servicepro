package handlers

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/javaknight1/servicepro/backend/internal/api/apierror"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/javaknight1/servicepro/backend/internal/models"
	"github.com/javaknight1/servicepro/backend/internal/services"
)

// QuoteHandler handles quote endpoints
type QuoteHandler struct {
	quoteService services.QuoteServiceInterface
}

// NewQuoteHandler creates a new quote handler
func NewQuoteHandler(quoteService services.QuoteServiceInterface) *QuoteHandler {
	return &QuoteHandler{
		quoteService: quoteService,
	}
}

// CreateQuote handles POST /api/v1/quotes
// @Summary Create new quote
// @Description Creates a new quote with line items and calculates totals
// @Tags quotes
// @Accept json
// @Produce json
// @Param request body models.QuoteRequest true "Quote details"
// @Success 201 {object} models.QuoteResponse
// @Failure 400 {object} models.ErrorResponse "Invalid request"
// @Failure 401 {object} models.ErrorResponse "Unauthorized"
// @Failure 500 {object} models.ErrorResponse "Internal server error"
// @Router /api/v1/quotes [post]
func (h *QuoteHandler) CreateQuote(c *gin.Context) {
	var req models.QuoteRequest

	// Bind and validate request
	if err := c.ShouldBindJSON(&req); err != nil {
		apierror.BindError(c, err)
		return
	}

	// Get user ID from context (set by auth middleware)
	userID, err := getUserIDFromContext(c)
	if err != nil {
		apierror.WriteResponse(c, http.StatusUnauthorized, models.ErrorResponse{
			Error:   "unauthorized",
			Message: "User authentication required",
		})
		return
	}

	// Create quote
	quote, err := h.quoteService.WithContext(c.Request.Context()).CreateQuote(&req, userID)
	if err != nil {
		apierror.Respond(c, err, "Failed to create quote")
		return
	}

	c.JSON(http.StatusCreated, quote)
}

// ListQuotes handles GET /api/v1/quotes
// @Summary List quotes
// @Description Lists quotes with pagination, filtering, and sorting
// @Tags quotes
// @Accept json
// @Produce json
// @Param customer_id query string false "Filter by customer ID"
// @Param status query string false "Filter by status (draft, sent, accepted, rejected, expired)"
// @Param q query string false "Search quote number and customer first/last/company name (case-insensitive, max 100 chars)"
// @Param sort_by query string false "Sort column" Enums(created_at, updated_at, quote_number, status, total, valid_until) default(created_at)
// @Param sort_order query string false "Sort direction" Enums(asc, desc) default(desc)
// @Param page query int false "Page number" default(1)
// @Param page_size query int false "Page size" default(50)
// @Success 200 {object} models.QuoteListResponse
// @Failure 400 {object} models.ErrorResponse "Invalid request (including an unknown sort_by/sort_order)"
// @Failure 401 {object} models.ErrorResponse "Unauthorized"
// @Failure 500 {object} models.ErrorResponse "Internal server error"
// @Router /api/v1/quotes [get]
func (h *QuoteHandler) ListQuotes(c *gin.Context) {
	var params models.QuoteQueryParams

	// Bind query parameters
	if err := c.ShouldBindQuery(&params); err != nil {
		apierror.BindError(c, err)
		return
	}

	// List quotes
	quotes, err := h.quoteService.WithContext(c.Request.Context()).ListQuotes(&params)
	if err != nil {
		apierror.Internal(c, err, "Failed to list quotes")
		return
	}

	c.JSON(http.StatusOK, quotes)
}

// GetQuote handles GET /api/v1/quotes/:id
// @Summary Get quote by ID
// @Description Returns a single quote with all line items
// @Tags quotes
// @Accept json
// @Produce json
// @Param id path string true "Quote ID"
// @Success 200 {object} models.QuoteResponse
// @Failure 400 {object} models.ErrorResponse "Invalid ID"
// @Failure 404 {object} models.ErrorResponse "Quote not found"
// @Failure 500 {object} models.ErrorResponse "Internal server error"
// @Router /api/v1/quotes/{id} [get]
func (h *QuoteHandler) GetQuote(c *gin.Context) {
	// Parse quote ID
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		apierror.WriteResponse(c, http.StatusBadRequest, models.ErrorResponse{
			Error:   "invalid_id",
			Message: "Invalid quote ID format",
		})
		return
	}

	// Get quote
	quote, err := h.quoteService.WithContext(c.Request.Context()).GetQuote(id)
	if err != nil {
		apierror.Respond(c, err, "Failed to get quote")
		return
	}

	c.JSON(http.StatusOK, quote)
}

// GetQuoteByNumber handles GET /api/v1/quotes/number/:quoteNumber
// @Summary Get quote by quote number
// @Description Returns a single quote by quote number
// @Tags quotes
// @Accept json
// @Produce json
// @Param quoteNumber path string true "Quote Number"
// @Success 200 {object} models.QuoteResponse
// @Failure 404 {object} models.ErrorResponse "Quote not found"
// @Failure 500 {object} models.ErrorResponse "Internal server error"
// @Router /api/v1/quotes/number/{quoteNumber} [get]
func (h *QuoteHandler) GetQuoteByNumber(c *gin.Context) {
	quoteNumber := c.Param("quoteNumber")

	// Get quote
	quote, err := h.quoteService.WithContext(c.Request.Context()).GetQuoteByNumber(quoteNumber)
	if err != nil {
		apierror.Respond(c, err, "Failed to get quote")
		return
	}

	c.JSON(http.StatusOK, quote)
}

// UpdateQuote handles PUT /api/v1/quotes/:id
// @Summary Update quote
// @Description Updates quote details and recalculates totals
// @Tags quotes
// @Accept json
// @Produce json
// @Param id path string true "Quote ID"
// @Param request body models.QuoteRequest true "Quote details"
// @Success 200 {object} models.QuoteResponse
// @Failure 400 {object} models.ErrorResponse "Invalid request"
// @Failure 401 {object} models.ErrorResponse "Unauthorized"
// @Failure 409 {object} models.ErrorResponse "Quote cannot be edited in its current status"
// @Failure 404 {object} models.ErrorResponse "Quote not found"
// @Failure 500 {object} models.ErrorResponse "Internal server error"
// @Router /api/v1/quotes/{id} [put]
func (h *QuoteHandler) UpdateQuote(c *gin.Context) {
	// Parse quote ID
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		apierror.WriteResponse(c, http.StatusBadRequest, models.ErrorResponse{
			Error:   "invalid_id",
			Message: "Invalid quote ID format",
		})
		return
	}

	var req models.QuoteRequest

	// Bind and validate request
	if err := c.ShouldBindJSON(&req); err != nil {
		apierror.BindError(c, err)
		return
	}

	// Get user ID from context
	userID, err := getUserIDFromContext(c)
	if err != nil {
		apierror.WriteResponse(c, http.StatusUnauthorized, models.ErrorResponse{
			Error:   "unauthorized",
			Message: "User authentication required",
		})
		return
	}

	// Update quote
	quote, err := h.quoteService.WithContext(c.Request.Context()).UpdateQuote(id, &req, userID)
	if err != nil {
		apierror.Respond(c, err, "Failed to update quote")
		return
	}

	c.JSON(http.StatusOK, quote)
}

// DeleteQuote handles DELETE /api/v1/quotes/:id
// @Summary Delete quote
// @Description Soft deletes a quote (only allowed for draft quotes)
// @Tags quotes
// @Accept json
// @Produce json
// @Param id path string true "Quote ID"
// @Success 204 "No Content"
// @Failure 400 {object} models.ErrorResponse "Invalid ID"
// @Failure 409 {object} models.ErrorResponse "Quote cannot be deleted in its current status"
// @Failure 404 {object} models.ErrorResponse "Quote not found"
// @Failure 500 {object} models.ErrorResponse "Internal server error"
// @Router /api/v1/quotes/{id} [delete]
func (h *QuoteHandler) DeleteQuote(c *gin.Context) {
	// Parse quote ID
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		apierror.WriteResponse(c, http.StatusBadRequest, models.ErrorResponse{
			Error:   "invalid_id",
			Message: "Invalid quote ID format",
		})
		return
	}

	// Delete quote
	err = h.quoteService.WithContext(c.Request.Context()).DeleteQuote(id)
	if err != nil {
		apierror.Respond(c, err, "Failed to delete quote")
		return
	}

	c.Status(http.StatusNoContent)
}

// SendQuote handles POST /api/v1/quotes/:id/send
// @Summary Send quote
// @Description Marks a quote as sent
// @Tags quotes
// @Accept json
// @Produce json
// @Param id path string true "Quote ID"
// @Success 200 {object} models.QuoteResponse
// @Failure 400 {object} models.ErrorResponse "Invalid ID"
// @Failure 409 {object} models.ErrorResponse "Quote cannot be sent in its current status"
// @Failure 404 {object} models.ErrorResponse "Quote not found"
// @Failure 500 {object} models.ErrorResponse "Internal server error"
// @Router /api/v1/quotes/{id}/send [post]
func (h *QuoteHandler) SendQuote(c *gin.Context) {
	// Parse quote ID
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		apierror.WriteResponse(c, http.StatusBadRequest, models.ErrorResponse{
			Error:   "invalid_id",
			Message: "Invalid quote ID format",
		})
		return
	}

	// Send quote
	err = h.quoteService.WithContext(c.Request.Context()).SendQuote(id)
	if err != nil {
		apierror.Respond(c, err, "Failed to send quote")
		return
	}

	// Get updated quote
	quote, err := h.quoteService.WithContext(c.Request.Context()).GetQuote(id)
	if err != nil {
		apierror.Internal(c, err, "Quote sent but failed to fetch updated quote")
		return
	}

	c.JSON(http.StatusOK, quote)
}

// AcceptQuote handles POST /api/v1/quotes/:id/accept
// @Summary Accept quote
// @Description Marks a quote as accepted
// @Tags quotes
// @Accept json
// @Produce json
// @Param id path string true "Quote ID"
// @Success 200 {object} models.QuoteResponse
// @Failure 400 {object} models.ErrorResponse "Invalid ID"
// @Failure 409 {object} models.ErrorResponse "Quote cannot be accepted in its current status"
// @Failure 404 {object} models.ErrorResponse "Quote not found"
// @Failure 500 {object} models.ErrorResponse "Internal server error"
// @Router /api/v1/quotes/{id}/accept [post]
func (h *QuoteHandler) AcceptQuote(c *gin.Context) {
	// Parse quote ID
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		apierror.WriteResponse(c, http.StatusBadRequest, models.ErrorResponse{
			Error:   "invalid_id",
			Message: "Invalid quote ID format",
		})
		return
	}

	// Accept quote
	err = h.quoteService.WithContext(c.Request.Context()).AcceptQuote(id)
	if err != nil {
		apierror.Respond(c, err, "Failed to accept quote")
		return
	}

	// Get updated quote
	quote, err := h.quoteService.WithContext(c.Request.Context()).GetQuote(id)
	if err != nil {
		apierror.Internal(c, err, "Quote accepted but failed to fetch updated quote")
		return
	}

	c.JSON(http.StatusOK, quote)
}

// ReviseQuote handles POST /api/v1/quotes/:id/revise
// @Summary Revise quote
// @Description Takes a sent, viewed, expired or declined quote back to draft so it can be edited and sent again (only drafts are editable)
// @Tags quotes
// @Accept json
// @Produce json
// @Param id path string true "Quote ID"
// @Success 200 {object} models.QuoteResponse
// @Failure 400 {object} models.ErrorResponse "Invalid ID"
// @Failure 404 {object} models.ErrorResponse "Quote not found"
// @Failure 409 {object} models.ErrorResponse "Quote cannot be revised in its status"
// @Failure 500 {object} models.ErrorResponse "Internal server error"
// @Router /api/v1/quotes/{id}/revise [post]
func (h *QuoteHandler) ReviseQuote(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		apierror.WriteResponse(c, http.StatusBadRequest, models.ErrorResponse{Error: "invalid_id", Message: "Invalid quote ID format"})
		return
	}
	svc := h.quoteService.WithContext(c.Request.Context())
	if err := svc.ReviseQuote(id); err != nil {
		apierror.Respond(c, err, "Failed to revise quote")
		return
	}
	quote, err := svc.GetQuote(id)
	if err != nil {
		apierror.Internal(c, err, "Quote revised but failed to fetch updated quote")
		return
	}
	c.JSON(http.StatusOK, quote)
}

// RejectQuote handles POST /api/v1/quotes/:id/reject
// @Summary Reject quote
// @Description Marks a quote as rejected
// @Tags quotes
// @Accept json
// @Produce json
// @Param id path string true "Quote ID"
// @Success 200 {object} models.QuoteResponse
// @Failure 400 {object} models.ErrorResponse "Invalid ID"
// @Failure 409 {object} models.ErrorResponse "Quote cannot be rejected in its current status"
// @Failure 404 {object} models.ErrorResponse "Quote not found"
// @Failure 500 {object} models.ErrorResponse "Internal server error"
// @Router /api/v1/quotes/{id}/reject [post]
func (h *QuoteHandler) RejectQuote(c *gin.Context) {
	// Parse quote ID
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		apierror.WriteResponse(c, http.StatusBadRequest, models.ErrorResponse{
			Error:   "invalid_id",
			Message: "Invalid quote ID format",
		})
		return
	}

	// Reject quote
	err = h.quoteService.WithContext(c.Request.Context()).RejectQuote(id)
	if err != nil {
		apierror.Respond(c, err, "Failed to reject quote")
		return
	}

	// Get updated quote
	quote, err := h.quoteService.WithContext(c.Request.Context()).GetQuote(id)
	if err != nil {
		apierror.Internal(c, err, "Quote rejected but failed to fetch updated quote")
		return
	}

	c.JSON(http.StatusOK, quote)
}

// GetCustomerQuotes handles GET /api/v1/customers/:id/quotes
// @Summary Get customer quotes
// @Description Retrieves all quotes for a customer
// @Tags quotes
// @Accept json
// @Produce json
// @Param id path string true "Customer ID"
// @Success 200 {array} models.QuoteResponse
// @Failure 400 {object} models.ErrorResponse "Invalid ID"
// @Failure 500 {object} models.ErrorResponse "Internal server error"
// @Router /api/v1/customers/{id}/quotes [get]
func (h *QuoteHandler) GetCustomerQuotes(c *gin.Context) {
	// Parse customer ID
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		apierror.WriteResponse(c, http.StatusBadRequest, models.ErrorResponse{
			Error:   "invalid_id",
			Message: "Invalid customer ID format",
		})
		return
	}

	// Get customer quotes
	quotes, err := h.quoteService.WithContext(c.Request.Context()).GetQuotesByCustomer(id)
	if err != nil {
		apierror.Internal(c, err, "Failed to get customer quotes")
		return
	}

	c.JSON(http.StatusOK, quotes)
}

// GetQuoteStats handles GET /api/v1/quotes/stats
// @Summary Get quote statistics
// @Description Retrieves quote statistics by status
// @Tags quotes
// @Accept json
// @Produce json
// @Success 200 {object} services.QuoteStats
// @Failure 500 {object} models.ErrorResponse "Internal server error"
// @Router /api/v1/quotes/stats [get]
func (h *QuoteHandler) GetQuoteStats(c *gin.Context) {
	stats, err := h.quoteService.WithContext(c.Request.Context()).GetQuoteStats()
	if err != nil {
		apierror.Internal(c, err, "Failed to get quote statistics")
		return
	}

	c.JSON(http.StatusOK, stats)
}

// GetQuotePDF handles GET /api/v1/quotes/:id/pdf
// @Summary Get quote PDF download URL
// @Description Returns a presigned URL for downloading the quote PDF
// @Tags quotes
// @Accept json
// @Produce application/pdf
// @Param id path string true "Quote ID"
// @Success 200 {file} binary "PDF file"
// @Failure 400 {object} models.ErrorResponse "Invalid ID"
// @Failure 404 {object} models.ErrorResponse "Quote not found"
// @Failure 500 {object} models.ErrorResponse "Internal server error"
// @Router /api/v1/quotes/{id}/pdf [get]
func (h *QuoteHandler) GetQuotePDF(c *gin.Context) {
	// Parse quote ID
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		apierror.WriteResponse(c, http.StatusBadRequest, models.ErrorResponse{
			Error:   "invalid_id",
			Message: "Invalid quote ID format",
		})
		return
	}

	// Download PDF content directly
	content, filename, err := h.quoteService.WithContext(c.Request.Context()).DownloadQuotePDF(id)
	if err != nil {
		apierror.Respond(c, err, "Failed to get quote PDF")
		return
	}

	// Stream PDF directly to client
	c.Header("Content-Type", "application/pdf")
	c.Header("Content-Disposition", "attachment; filename=\""+filename+"\"")
	c.Header("Content-Length", fmt.Sprintf("%d", len(content)))
	c.Data(http.StatusOK, "application/pdf", content)
}

// getUserIDFromContext extracts user ID from gin context
func getUserIDFromContext(c *gin.Context) (uuid.UUID, error) {
	userIDInterface, exists := c.Get("user_id")
	if !exists {
		return uuid.Nil, errors.New("user_id not found in context")
	}

	userID, ok := userIDInterface.(uuid.UUID)
	if !ok {
		return uuid.Nil, errors.New("user_id is not a valid UUID")
	}

	return userID, nil
}
