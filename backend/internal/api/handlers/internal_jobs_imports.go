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

// Limits for one imports pass: the scheduler's curl gives up after 300 s.
const (
	importsJobMaxJobs = 20
	importsJobBudget  = 4 * time.Minute
)

// ImportRunner runs one pass over queued CSV imports
// (services.ImportService.ProcessPending).
type ImportRunner func(ctx context.Context, maxJobs int, budget time.Duration) (services.ImportRunResult, error)

// ImportsJobResponse is returned by POST /internal/jobs/imports.
type ImportsJobResponse struct {
	Job        string `json:"job" example:"imports"`
	DurationMs int64  `json:"duration_ms" example:"812"`
	services.ImportRunResult
}

// RunImports returns the imports job handler. It is mounted on the
// /internal/jobs group, behind InternalJobsHandler.Authorize.
// @Summary		Run CSV imports job
// @Description	Internal job endpoint (auth: X-Internal-Jobs-Token header = INTERNAL_JOBS_SECRET; 404 when the secret is unset). Runs queued customer CSV imports for every tenant: each job is claimed atomically (pending → processing, FOR UPDATE SKIP LOCKED) before it runs, so overlapping runs never import a job twice; each job imports into its own tenant only. Jobs stuck in processing for 15 minutes are marked failed. Runs up to 20 jobs or 4 minutes per call. Mounted at the server root (/internal/jobs/imports), outside /api.
// @Tags			Internal
// @Produce		json
// @Param			X-Internal-Jobs-Token	header		string	true	"Internal jobs secret"
// @Success		200						{object}	ImportsJobResponse
// @Failure		401						{object}	models.ErrorResponse
// @Failure		404						{object}	models.ErrorResponse
// @Failure		500						{object}	models.ErrorResponse
// @Router			/internal/jobs/imports [post]
func RunImports(run ImportRunner) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		result, err := run(c.Request.Context(), importsJobMaxJobs, importsJobBudget)
		if err != nil {
			logging.Error(c.Request.Context(), "[INTERNAL-JOBS] imports failed", map[string]any{"error": err.Error()})
			apierror.WriteResponse(c, http.StatusInternalServerError, models.ErrorResponse{Error: "Imports job failed"})
			return
		}
		c.JSON(http.StatusOK, ImportsJobResponse{
			Job:             "imports",
			DurationMs:      time.Since(start).Milliseconds(),
			ImportRunResult: result,
		})
	}
}
