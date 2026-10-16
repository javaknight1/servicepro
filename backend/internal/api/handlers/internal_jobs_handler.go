package handlers

import (
	"context"
	"crypto/subtle"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/javaknight1/servicepro/backend/internal/api/apierror"
	"github.com/javaknight1/servicepro/backend/internal/models"
	"github.com/javaknight1/servicepro/backend/internal/services"
	"github.com/javaknight1/servicepro/backend/pkg/clients/logging"
)

// InternalJobsTokenHeader carries INTERNAL_JOBS_SECRET on /internal/jobs/*.
const InternalJobsTokenHeader = "X-Internal-Jobs-Token"

// ReminderRunner runs one automatic payment-reminder pass
// (implemented by *services.PaymentReminderService).
type ReminderRunner interface {
	ProcessAllTenants(ctx context.Context) (services.ReminderRunResult, error)
}

// InternalJobsHandler exposes periodic work as idempotent HTTP endpoints so
// an external scheduler (GitHub Actions cron / Cloud Scheduler) can trigger
// it. On Cloud Run, in-process tickers are unreliable: CPU is throttled
// between requests and instances scale to zero.
type InternalJobsHandler struct {
	secret    string
	reminders ReminderRunner
}

// NewInternalJobsHandler creates the handler. An empty secret disables every
// job endpoint (404).
func NewInternalJobsHandler(secret string, reminders ReminderRunner) *InternalJobsHandler {
	return &InternalJobsHandler{secret: secret, reminders: reminders}
}

// PaymentRemindersJobResponse is returned by POST /internal/jobs/payment-reminders.
type PaymentRemindersJobResponse struct {
	Job        string `json:"job" example:"payment-reminders"`
	DurationMs int64  `json:"duration_ms" example:"412"`
	services.ReminderRunResult
}

// Authorize rejects requests without a valid X-Internal-Jobs-Token: 404 when
// INTERNAL_JOBS_SECRET is unset (endpoints disabled), 401 on a missing or
// wrong token (constant-time compare).
func (h *InternalJobsHandler) Authorize(c *gin.Context) {
	if h.secret == "" {
		apierror.WriteResponse(c, http.StatusNotFound, models.ErrorResponse{Error: "Not found"})
		return
	}
	got := strings.TrimSpace(c.GetHeader(InternalJobsTokenHeader))
	if got == "" || subtle.ConstantTimeCompare([]byte(got), []byte(h.secret)) != 1 {
		apierror.WriteResponse(c, http.StatusUnauthorized, models.ErrorResponse{Error: "Invalid internal jobs token"})
		return
	}
	c.Next()
}

// RunPaymentReminders runs one automatic payment-reminder pass.
// @Summary		Run payment reminders job
// @Description	Internal job endpoint (auth: X-Internal-Jobs-Token header = INTERNAL_JOBS_SECRET; 404 when the secret is unset). Sends due automatic payment reminders for every active tenant with reminders enabled. Idempotent: each (invoice, reminder number) is sent at most once, so re-running or overlapping runs send nothing twice. Mounted at the server root (/internal/jobs/payment-reminders), outside /api.
// @Tags			Internal
// @Produce		json
// @Param			X-Internal-Jobs-Token	header		string	true	"Internal jobs secret"
// @Success		200						{object}	PaymentRemindersJobResponse
// @Failure		401						{object}	models.ErrorResponse
// @Failure		404						{object}	models.ErrorResponse
// @Failure		500						{object}	models.ErrorResponse
// @Failure		503						{object}	models.ErrorResponse
// @Router			/internal/jobs/payment-reminders [post]
func (h *InternalJobsHandler) RunPaymentReminders(c *gin.Context) {
	if h.reminders == nil {
		apierror.WriteResponse(c, http.StatusServiceUnavailable, models.ErrorResponse{Error: "Payment reminder service not configured"})
		return
	}

	start := time.Now()
	result, err := h.reminders.ProcessAllTenants(c.Request.Context())
	if err != nil {
		logging.Error(c.Request.Context(), "[INTERNAL-JOBS] payment-reminders failed", map[string]any{"error": err.Error()})
		apierror.WriteResponse(c, http.StatusInternalServerError, models.ErrorResponse{Error: "Payment reminders job failed"})
		return
	}

	c.JSON(http.StatusOK, PaymentRemindersJobResponse{
		Job:               "payment-reminders",
		DurationMs:        time.Since(start).Milliseconds(),
		ReminderRunResult: result,
	})
}
