package handlers

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/javaknight1/servicepro/backend/internal/api/apierror"
	"github.com/javaknight1/servicepro/backend/internal/models"
	"github.com/javaknight1/servicepro/backend/pkg/clients/logging"
)

// OverdueMarker marks sent invoices past their due date overdue
// (services.MarkOverdueInvoices).
type OverdueMarker func(ctx context.Context, now time.Time) (int64, error)

// InvoiceOverdueJobResponse is returned by POST /internal/jobs/invoice-overdue.
type InvoiceOverdueJobResponse struct {
	Job        string `json:"job" example:"invoice-overdue"`
	DurationMs int64  `json:"duration_ms" example:"12"`
	Marked     int64  `json:"marked" example:"3"`
}

// RunInvoiceOverdue returns the invoice-overdue job handler. It is mounted on
// the /internal/jobs group, behind InternalJobsHandler.Authorize.
// @Summary		Run invoice overdue job
// @Description	Internal job endpoint (auth: X-Internal-Jobs-Token header = INTERNAL_JOBS_SECRET; 404 when the secret is unset). Moves sent/viewed invoices due before today (UTC) to overdue, for every tenant. Idempotent (one conditional UPDATE). Mounted at the server root (/internal/jobs/invoice-overdue), outside /api.
// @Tags			Internal
// @Produce		json
// @Param			X-Internal-Jobs-Token	header		string	true	"Internal jobs secret"
// @Success		200						{object}	InvoiceOverdueJobResponse
// @Failure		401						{object}	models.ErrorResponse
// @Failure		404						{object}	models.ErrorResponse
// @Failure		500						{object}	models.ErrorResponse
// @Router			/internal/jobs/invoice-overdue [post]
func RunInvoiceOverdue(mark OverdueMarker) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		n, err := mark(c.Request.Context(), start)
		if err != nil {
			logging.Error(c.Request.Context(), "[INTERNAL-JOBS] invoice-overdue failed", map[string]any{"error": err.Error()})
			apierror.WriteResponse(c, http.StatusInternalServerError, models.ErrorResponse{Error: "Invoice overdue job failed"})
			return
		}
		c.JSON(http.StatusOK, InvoiceOverdueJobResponse{
			Job:        "invoice-overdue",
			DurationMs: time.Since(start).Milliseconds(),
			Marked:     n,
		})
	}
}
