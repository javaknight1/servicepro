package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/javaknight1/servicepro/backend/internal/api/apierror"
	"github.com/javaknight1/servicepro/backend/internal/models"
)

// InvoiceJob godoc
// @Summary Generate the invoice for a completed job
// @Description Creates a draft invoice from a completed job and moves the job to invoiced, all in one transaction (T029). Lines are built by the server (T030): a job converted from a quote bills the quote's lines (and links the quote); otherwise labor = actual time on site (actual start/end, else actual duration) × the tenant's laborRate setting, or — no rate or no time recorded — one line for the job's amount (total, else actual cost, else estimate); then each billable material at its unit cost. Lines carry the job's site name; the job's completion notes and non-internal notes become the invoice notes; the customer's default PO and the tenant's default payment term and tax rate apply; totals are computed server-side. Idempotent: a job is invoiced once — a second (or concurrent) call answers 409 already_invoiced with the existing invoice's ID in resource_id, including when a consolidated invoice billed the job. 409 job_not_completed for other statuses; 409 agreement_visit for agreement visits (billed on the consolidated invoice).
// @Tags jobs
// @Produce json
// @Param id path string true "Job ID"
// @Success 201 {object} models.Invoice
// @Failure 400 {object} models.ErrorResponse "Invalid job ID"
// @Failure 401 {object} models.ErrorResponse "Unauthorized"
// @Failure 403 {object} models.ErrorResponse "Missing invoices.create permission"
// @Failure 404 {object} models.ErrorResponse "Job not found"
// @Failure 409 {object} models.ErrorResponse "Not completed, an agreement visit, or already invoiced (resource_id = the invoice)"
// @Failure 500 {object} models.ErrorResponse "Internal server error"
// @Router /api/v1/jobs/{id}/invoice [post]
// @Security BearerAuth
func (h *InvoiceHandler) InvoiceJob(c *gin.Context) {
	jobID, ok := parseUUIDParam(c, "id", "job")
	if !ok {
		return
	}
	raw, _ := c.Get("user_id")
	userID, isUUID := raw.(uuid.UUID)
	if !isUUID {
		apierror.Write(c, http.StatusUnauthorized, "unauthorized", "User not authenticated")
		return
	}
	var invoice *models.Invoice
	var err error
	invoice, err = h.service.InvoiceJob(c.Request.Context(), jobID, userID)
	if err != nil {
		respondInvoiceError(c, err, "Failed to create the invoice for this job")
		return
	}
	c.JSON(http.StatusCreated, invoice)
}
