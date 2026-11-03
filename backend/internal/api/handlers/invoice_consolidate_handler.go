package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/javaknight1/servicepro/backend/internal/api/apierror"
	"github.com/javaknight1/servicepro/backend/internal/models"
)

// PreviewConsolidatedInvoice godoc
// @Summary Preview a consolidated invoice
// @Description The lines and totals POST /invoices/consolidate would create for one customer and period, without writing anything (T188). Monthly agreements bill price_per_period once per calendar month the period touches; per-visit agreements bill completed visits in the period × price; each completed extra-work job finished by the end of the period is its own line with its approval. Work already billed is left out. Dates are the tenant's local calendar dates, inclusive.
// @Tags invoices
// @Produce json
// @Param customer_id query string true "Customer ID"
// @Param period_start query string true "First day of the period (YYYY-MM-DD)"
// @Param period_end query string true "Last day of the period (YYYY-MM-DD)"
// @Param site_id query string false "Only this site"
// @Success 200 {object} models.ConsolidatedInvoicePreview
// @Failure 400 {object} models.ErrorResponse
// @Failure 401 {object} models.ErrorResponse
// @Failure 404 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Router /invoices/consolidate/preview [get]
// @Security BearerAuth
func (h *InvoiceHandler) PreviewConsolidatedInvoice(c *gin.Context) {
	var req models.ConsolidateInvoiceRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		apierror.BindError(c, err)
		return
	}
	preview, err := h.service.PreviewConsolidatedInvoice(c.Request.Context(), &req)
	if err != nil {
		respondInvoiceError(c, err, "Failed to preview the consolidated invoice")
		return
	}
	c.JSON(http.StatusOK, preview)
}

// ConsolidateInvoice godoc
// @Summary Create a consolidated invoice
// @Description Creates one draft invoice for the customer's unbilled work in the period (see the preview), a line per site per agreement plus extra work, with the customer's default PO, and moves the included jobs to invoiced — all in one transaction (T188). Each month, visit and job is billed once: work already on another invoice is skipped; 409 nothing_to_invoice when nothing is left, 409 already_invoiced when a concurrent request billed some of it first. Cancelling or deleting the invoice releases the work.
// @Tags invoices
// @Accept json
// @Produce json
// @Param request body models.ConsolidateInvoiceRequest true "Customer and period"
// @Success 201 {object} models.Invoice
// @Failure 400 {object} models.ErrorResponse
// @Failure 401 {object} models.ErrorResponse
// @Failure 404 {object} models.ErrorResponse
// @Failure 409 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Router /invoices/consolidate [post]
// @Security BearerAuth
func (h *InvoiceHandler) ConsolidateInvoice(c *gin.Context) {
	var req models.ConsolidateInvoiceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		apierror.BindError(c, err)
		return
	}
	raw, _ := c.Get("user_id")
	userID, ok := raw.(uuid.UUID)
	if !ok {
		apierror.WriteResponse(c, http.StatusUnauthorized, models.ErrorResponse{Error: "Unauthorized"})
		return
	}
	invoice, err := h.service.ConsolidateInvoice(c.Request.Context(), &req, userID)
	if err != nil {
		respondInvoiceError(c, err, "Failed to create the consolidated invoice")
		return
	}
	c.JSON(http.StatusCreated, invoice)
}
