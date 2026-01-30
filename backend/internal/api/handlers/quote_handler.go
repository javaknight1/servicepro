package handlers

import (
	"errors"
	"fmt"
	"log"
	"net/http"

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
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "invalid_request",
			Message: "Invalid quote data: " + err.Error(),
		})
		return
	}

	// Get user ID from context (set by auth middleware)
	userID, err := getUserIDFromContext(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, models.ErrorResponse{
			Error:   "unauthorized",
			Message: "User authentication required",
		})
		return
	}

	// Create quote
	quote, err := h.quoteService.CreateQuote(&req, userID)
	if err != nil {
		if errors.Is(err, services.ErrInvalidQuoteData) {
			c.JSON(http.StatusBadRequest, models.ErrorResponse{
				Error:   "invalid_data",
				Message: err.Error(),
			})
			return
		}

		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to create quote",
		})
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
// @Param page query int false "Page number" default(1)
// @Param page_size query int false "Page size" default(50)
// @Success 200 {object} models.QuoteListResponse
// @Failure 400 {object} models.ErrorResponse "Invalid request"
// @Failure 401 {object} models.ErrorResponse "Unauthorized"
// @Failure 500 {object} models.ErrorResponse "Internal server error"
// @Router /api/v1/quotes [get]
func (h *QuoteHandler) ListQuotes(c *gin.Context) {
	var params models.QuoteQueryParams

	// Bind query parameters
	if err := c.ShouldBindQuery(&params); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "invalid_request",
			Message: "Invalid query parameters: " + err.Error(),
		})
		return
	}

	// List quotes
	quotes, err := h.quoteService.ListQuotes(&params)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to list quotes",
		})
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
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "invalid_id",
			Message: "Invalid quote ID format",
		})
		return
	}

	// Get quote
	quote, err := h.quoteService.GetQuote(id)
	if err != nil {
		if errors.Is(err, services.ErrQuoteNotFound) {
			c.JSON(http.StatusNotFound, models.ErrorResponse{
				Error:   "not_found",
				Message: "Quote not found",
			})
			return
		}

		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to get quote",
		})
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
	quote, err := h.quoteService.GetQuoteByNumber(quoteNumber)
	if err != nil {
		if errors.Is(err, services.ErrQuoteNotFound) {
			c.JSON(http.StatusNotFound, models.ErrorResponse{
				Error:   "not_found",
				Message: "Quote not found",
			})
			return
		}

		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to get quote",
		})
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
// @Failure 403 {object} models.ErrorResponse "Quote cannot be edited"
// @Failure 404 {object} models.ErrorResponse "Quote not found"
// @Failure 500 {object} models.ErrorResponse "Internal server error"
// @Router /api/v1/quotes/{id} [put]
func (h *QuoteHandler) UpdateQuote(c *gin.Context) {
	// Parse quote ID
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "invalid_id",
			Message: "Invalid quote ID format",
		})
		return
	}

	var req models.QuoteRequest

	// Bind and validate request
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "invalid_request",
			Message: "Invalid quote data: " + err.Error(),
		})
		return
	}

	// Get user ID from context
	userID, err := getUserIDFromContext(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, models.ErrorResponse{
			Error:   "unauthorized",
			Message: "User authentication required",
		})
		return
	}

	// Update quote
	quote, err := h.quoteService.UpdateQuote(id, &req, userID)
	if err != nil {
		if errors.Is(err, services.ErrQuoteNotFound) {
			c.JSON(http.StatusNotFound, models.ErrorResponse{
				Error:   "not_found",
				Message: "Quote not found",
			})
			return
		}

		if errors.Is(err, services.ErrQuoteCannotBeEdited) {
			c.JSON(http.StatusForbidden, models.ErrorResponse{
				Error:   "cannot_edit",
				Message: "Quote cannot be edited in current status",
			})
			return
		}

		if errors.Is(err, services.ErrInvalidQuoteData) {
			c.JSON(http.StatusBadRequest, models.ErrorResponse{
				Error:   "invalid_data",
				Message: err.Error(),
			})
			return
		}

		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to update quote",
		})
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
// @Failure 403 {object} models.ErrorResponse "Quote cannot be deleted"
// @Failure 404 {object} models.ErrorResponse "Quote not found"
// @Failure 500 {object} models.ErrorResponse "Internal server error"
// @Router /api/v1/quotes/{id} [delete]
func (h *QuoteHandler) DeleteQuote(c *gin.Context) {
	// Parse quote ID
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "invalid_id",
			Message: "Invalid quote ID format",
		})
		return
	}

	// Delete quote
	err = h.quoteService.DeleteQuote(id)
	if err != nil {
		if errors.Is(err, services.ErrQuoteNotFound) {
			c.JSON(http.StatusNotFound, models.ErrorResponse{
				Error:   "not_found",
				Message: "Quote not found",
			})
			return
		}

		if errors.Is(err, services.ErrQuoteCannotBeDeleted) {
			c.JSON(http.StatusForbidden, models.ErrorResponse{
				Error:   "cannot_delete",
				Message: "Quote can only be deleted in draft status",
			})
			return
		}

		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to delete quote",
		})
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
// @Failure 403 {object} models.ErrorResponse "Quote cannot be sent"
// @Failure 404 {object} models.ErrorResponse "Quote not found"
// @Failure 500 {object} models.ErrorResponse "Internal server error"
// @Router /api/v1/quotes/{id}/send [post]
func (h *QuoteHandler) SendQuote(c *gin.Context) {
	// Parse quote ID
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "invalid_id",
			Message: "Invalid quote ID format",
		})
		return
	}

	// Send quote
	err = h.quoteService.SendQuote(id)
	if err != nil {
		if errors.Is(err, services.ErrQuoteNotFound) {
			c.JSON(http.StatusNotFound, models.ErrorResponse{
				Error:   "not_found",
				Message: "Quote not found",
			})
			return
		}

		if errors.Is(err, services.ErrQuoteCannotBeSent) {
			c.JSON(http.StatusForbidden, models.ErrorResponse{
				Error:   "cannot_send",
				Message: "Quote cannot be sent in current status",
			})
			return
		}

		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to send quote",
		})
		return
	}

	// Get updated quote
	quote, err := h.quoteService.GetQuote(id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "internal_error",
			Message: "Quote sent but failed to fetch updated quote",
		})
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
// @Failure 403 {object} models.ErrorResponse "Quote cannot be accepted"
// @Failure 404 {object} models.ErrorResponse "Quote not found"
// @Failure 500 {object} models.ErrorResponse "Internal server error"
// @Router /api/v1/quotes/{id}/accept [post]
func (h *QuoteHandler) AcceptQuote(c *gin.Context) {
	// Parse quote ID
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "invalid_id",
			Message: "Invalid quote ID format",
		})
		return
	}

	// Accept quote
	err = h.quoteService.AcceptQuote(id)
	if err != nil {
		if errors.Is(err, services.ErrQuoteNotFound) {
			c.JSON(http.StatusNotFound, models.ErrorResponse{
				Error:   "not_found",
				Message: "Quote not found",
			})
			return
		}

		if errors.Is(err, services.ErrQuoteCannotBeAccepted) {
			c.JSON(http.StatusForbidden, models.ErrorResponse{
				Error:   "cannot_accept",
				Message: "Quote cannot be accepted in current status",
			})
			return
		}

		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to accept quote",
		})
		return
	}

	// Get updated quote
	quote, err := h.quoteService.GetQuote(id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "internal_error",
			Message: "Quote accepted but failed to fetch updated quote",
		})
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
// @Failure 403 {object} models.ErrorResponse "Quote cannot be rejected"
// @Failure 404 {object} models.ErrorResponse "Quote not found"
// @Failure 500 {object} models.ErrorResponse "Internal server error"
// @Router /api/v1/quotes/{id}/reject [post]
func (h *QuoteHandler) RejectQuote(c *gin.Context) {
	// Parse quote ID
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "invalid_id",
			Message: "Invalid quote ID format",
		})
		return
	}

	// Reject quote
	err = h.quoteService.RejectQuote(id)
	if err != nil {
		if errors.Is(err, services.ErrQuoteNotFound) {
			c.JSON(http.StatusNotFound, models.ErrorResponse{
				Error:   "not_found",
				Message: "Quote not found",
			})
			return
		}

		if errors.Is(err, services.ErrQuoteCannotBeRejected) {
			c.JSON(http.StatusForbidden, models.ErrorResponse{
				Error:   "cannot_reject",
				Message: "Quote cannot be rejected in current status",
			})
			return
		}

		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to reject quote",
		})
		return
	}

	// Get updated quote
	quote, err := h.quoteService.GetQuote(id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "internal_error",
			Message: "Quote rejected but failed to fetch updated quote",
		})
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
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "invalid_id",
			Message: "Invalid customer ID format",
		})
		return
	}

	// Get customer quotes
	quotes, err := h.quoteService.GetQuotesByCustomer(id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to get customer quotes",
		})
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
	stats, err := h.quoteService.GetQuoteStats()
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to get quote statistics",
		})
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
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "invalid_id",
			Message: "Invalid quote ID format",
		})
		return
	}

	// Download PDF content directly
	content, filename, err := h.quoteService.DownloadQuotePDF(id)
	if err != nil {
		if errors.Is(err, services.ErrQuoteNotFound) {
			c.JSON(http.StatusNotFound, models.ErrorResponse{
				Error:   "not_found",
				Message: "Quote not found",
			})
			return
		}

		log.Printf("[QUOTE-HANDLER] Failed to get quote PDF for %s: %v", id, err)
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to get quote PDF: " + err.Error(),
		})
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
