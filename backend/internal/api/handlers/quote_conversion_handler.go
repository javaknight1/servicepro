package handlers

import (
	"errors"
	"io"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/javaknight1/servicepro/backend/internal/api/apierror"
	"github.com/javaknight1/servicepro/backend/internal/models"
	"github.com/javaknight1/servicepro/backend/internal/services"
)

// QuoteConversionHandler turns accepted quotes into jobs (T028).
type QuoteConversionHandler struct {
	conversions *services.QuoteConversionService
}

// NewQuoteConversionHandler creates the handler.
func NewQuoteConversionHandler(conversions *services.QuoteConversionService) *QuoteConversionHandler {
	return &QuoteConversionHandler{conversions: conversions}
}

// ConvertToJob handles POST /api/v1/quotes/:id/convert-to-job
// @Summary Convert an accepted quote to a job
// @Description Creates a job (status new, ready to schedule) from an accepted quote in one transaction: the quote's customer, the optional site (its address becomes the service address, else the customer's billing address), the quote's line items and notes in the description, and its subtotal as the estimated cost. The job links back to the quote (quote_id); GET /quotes/{id} then returns job_id. A quote converts once: a second call answers 409 already_converted with the job's ID in resource_id. 409 quote_not_accepted for any other status; 409 already_invoiced when the quote was invoiced directly. When the job is invoiced (POST /jobs/{id}/invoice) its lines come from the quote.
// @Tags quotes
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Quote ID"
// @Param request body models.ConvertQuoteToJobRequest false "Optional title, job type and site"
// @Success 201 {object} models.JobResponse
// @Failure 400 {object} models.ErrorResponse "Invalid request, or the site isn't one of the customer's"
// @Failure 401 {object} models.ErrorResponse "Unauthorized"
// @Failure 403 {object} models.ErrorResponse "Missing jobs.create permission"
// @Failure 404 {object} models.ErrorResponse "Quote not found"
// @Failure 409 {object} models.ErrorResponse "Not accepted, already converted (resource_id = job) or already invoiced"
// @Failure 500 {object} models.ErrorResponse "Internal server error"
// @Router /api/v1/quotes/{id}/convert-to-job [post]
func (h *QuoteConversionHandler) ConvertToJob(c *gin.Context) {
	quoteID, ok := parseUUIDParam(c, "id", "quote")
	if !ok {
		return
	}
	raw, _ := c.Get("user_id")
	userID, isUUID := raw.(uuid.UUID)
	if !isUUID {
		apierror.Write(c, http.StatusUnauthorized, "unauthorized", "User not authenticated")
		return
	}
	// The body is optional: an empty POST converts with the defaults.
	var req models.ConvertQuoteToJobRequest
	if err := c.ShouldBindJSON(&req); err != nil && !errors.Is(err, io.EOF) {
		apierror.BindError(c, err)
		return
	}
	job, err := h.conversions.ConvertToJob(c.Request.Context(), quoteID, &req, userID)
	if err != nil {
		apierror.Respond(c, err, "Failed to convert the quote to a job")
		return
	}
	c.JSON(http.StatusCreated, job)
}
