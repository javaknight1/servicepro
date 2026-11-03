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

// AgreementVisitGenerator creates upcoming agreement visits
// (services.GenerateAgreementVisits).
type AgreementVisitGenerator func(ctx context.Context, now time.Time) (*services.AgreementVisitsResult, error)

// AgreementVisitsJobResponse is returned by POST /internal/jobs/agreement-visits.
type AgreementVisitsJobResponse struct {
	Job        string `json:"job" example:"agreement-visits"`
	DurationMs int64  `json:"duration_ms" example:"40"`
	services.AgreementVisitsResult
}

// RunAgreementVisits returns the agreement-visits job handler, mounted on the
// /internal/jobs group behind InternalJobsHandler.Authorize.
// @Summary		Run agreement visits job
// @Description	Internal job endpoint (auth: X-Internal-Jobs-Token header = INTERNAL_JOBS_SECRET; 404 when the secret is unset). Creates a scheduled job for every visit of every active service agreement over the next 14 days (each tenant's local dates). Idempotent: each (agreement, date) is claimed once. Mounted at the server root (/internal/jobs/agreement-visits), outside /api.
// @Tags			Internal
// @Produce		json
// @Param			X-Internal-Jobs-Token	header		string	true	"Internal jobs secret"
// @Success		200						{object}	AgreementVisitsJobResponse
// @Failure		401						{object}	models.ErrorResponse
// @Failure		404						{object}	models.ErrorResponse
// @Failure		500						{object}	models.ErrorResponse
// @Router			/internal/jobs/agreement-visits [post]
func RunAgreementVisits(generate AgreementVisitGenerator) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		res, err := generate(c.Request.Context(), start)
		if err != nil {
			fields := map[string]any{"error": err.Error()}
			if res != nil {
				fields["created"], fields["failed"] = res.Created, res.Failed
			}
			logging.Error(c.Request.Context(), "[INTERNAL-JOBS] agreement-visits failed", fields)
			apierror.WriteResponse(c, http.StatusInternalServerError, models.ErrorResponse{Error: "Agreement visits job failed"})
			return
		}
		c.JSON(http.StatusOK, AgreementVisitsJobResponse{
			Job:                   "agreement-visits",
			DurationMs:            time.Since(start).Milliseconds(),
			AgreementVisitsResult: *res,
		})
	}
}
