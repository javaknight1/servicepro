package handlers

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/javaknight1/servicepro/backend/pkg/clients/logging"
)

// QuoteExpirer expires sent quotes past valid_until (services.ExpireQuotes).
type QuoteExpirer func(ctx context.Context, now time.Time) (int64, error)

// QuoteExpiryJobResponse is returned by POST /internal/jobs/quote-expiry.
type QuoteExpiryJobResponse struct {
	Job        string `json:"job" example:"quote-expiry"`
	DurationMs int64  `json:"duration_ms" example:"12"`
	Expired    int64  `json:"expired" example:"3"`
}

// RunQuoteExpiry returns the quote-expiry job handler. It is mounted on the
// /internal/jobs group, behind InternalJobsHandler.Authorize.
// @Summary		Run quote expiry job
// @Description	Internal job endpoint (auth: X-Internal-Jobs-Token header = INTERNAL_JOBS_SECRET; 404 when the secret is unset). Moves sent/viewed quotes past valid_until to expired, for every tenant. Idempotent (one conditional UPDATE). Mounted at the server root (/internal/jobs/quote-expiry), outside /api.
// @Tags			Internal
// @Produce		json
// @Param			X-Internal-Jobs-Token	header		string	true	"Internal jobs secret"
// @Success		200						{object}	QuoteExpiryJobResponse
// @Failure		401						{object}	ErrorResponse
// @Failure		404						{object}	ErrorResponse
// @Failure		500						{object}	ErrorResponse
// @Router			/internal/jobs/quote-expiry [post]
func RunQuoteExpiry(expire QuoteExpirer) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		n, err := expire(c.Request.Context(), start)
		if err != nil {
			logging.Error(c.Request.Context(), "[INTERNAL-JOBS] quote-expiry failed", map[string]any{"error": err.Error()})
			c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Quote expiry job failed"})
			return
		}
		c.JSON(http.StatusOK, QuoteExpiryJobResponse{
			Job:        "quote-expiry",
			DurationMs: time.Since(start).Milliseconds(),
			Expired:    n,
		})
	}
}
