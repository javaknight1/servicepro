package handlers

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/javaknight1/servicepro/backend/internal/api/apierror"
	"github.com/javaknight1/servicepro/backend/internal/models"
	"github.com/javaknight1/servicepro/backend/internal/services"
	"github.com/javaknight1/servicepro/backend/pkg/clients/logging"
)

// ConsolidatedInvoiceGenerator creates the monthly consolidated drafts
// (InvoiceService.GenerateConsolidatedInvoices).
type ConsolidatedInvoiceGenerator func(ctx context.Context, now time.Time) (*services.ConsolidatedInvoicesResult, error)

// ConsolidatedInvoicesJobResponse is returned by POST /internal/jobs/consolidated-invoices.
type ConsolidatedInvoicesJobResponse struct {
	Job        string `json:"job" example:"consolidated-invoices"`
	DurationMs int64  `json:"duration_ms" example:"40"`
	services.ConsolidatedInvoicesResult
}

// RunConsolidatedInvoices returns the consolidated-invoices job handler,
// mounted on the /internal/jobs group behind InternalJobsHandler.Authorize.
// @Summary		Run consolidated invoices job
// @Description	Internal job endpoint (auth: X-Internal-Jobs-Token header = INTERNAL_JOBS_SECRET; 404 when the secret is unset). On the 1st of the month in each tenant's time zone, creates a draft consolidated invoice for the previous month for every customer with consolidated billing (T188). Drafts are not sent. Idempotent: each (customer, period) is claimed once, so hourly runs create at most one draft. Mounted at the server root (/internal/jobs/consolidated-invoices), outside /api.
// @Tags			Internal
// @Produce		json
// @Param			X-Internal-Jobs-Token	header		string	true	"Internal jobs secret"
// @Success		200						{object}	ConsolidatedInvoicesJobResponse
// @Failure		401						{object}	models.ErrorResponse
// @Failure		404						{object}	models.ErrorResponse
// @Failure		500						{object}	models.ErrorResponse
// @Router			/internal/jobs/consolidated-invoices [post]
func RunConsolidatedInvoices(generate ConsolidatedInvoiceGenerator) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		res, err := generate(c.Request.Context(), start)
		if err != nil {
			fields := map[string]any{"error": err.Error()}
			if res != nil {
				fields["created"], fields["failed"] = res.Created, res.Failed
			}
			logging.Error(c.Request.Context(), "[INTERNAL-JOBS] consolidated-invoices failed", fields)
			apierror.WriteResponse(c, http.StatusInternalServerError, models.ErrorResponse{Error: "Consolidated invoices job failed"})
			return
		}
		c.JSON(http.StatusOK, ConsolidatedInvoicesJobResponse{
			Job:                        "consolidated-invoices",
			DurationMs:                 time.Since(start).Milliseconds(),
			ConsolidatedInvoicesResult: *res,
		})
	}
}
