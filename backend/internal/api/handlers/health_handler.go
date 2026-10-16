package handlers

import (
	"crypto/subtle"
	"net"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/javaknight1/servicepro/backend/internal/health"
)

// HealthHandler handles health check endpoints
type HealthHandler struct {
	checker *health.Checker

	// alwaysDetailed returns the full report to everyone (development).
	alwaysDetailed bool
	// detailToken unlocks the full report via X-Internal-Jobs-Token or
	// "Authorization: Bearer" (INTERNAL_JOBS_SECRET).
	detailToken string
}

// SetDetailAccess configures who may see the detailed health report.
func (h *HealthHandler) SetDetailAccess(alwaysDetailed bool, token string) {
	h.alwaysDetailed = alwaysDetailed
	h.detailToken = token
}

// detailAllowed reports whether the caller may see dependency details
// (error strings, memory, versions of dependencies): development, a valid
// token, or a loopback caller.
func (h *HealthHandler) detailAllowed(c *gin.Context) bool {
	if h.alwaysDetailed {
		return true
	}
	if h.detailToken != "" {
		for _, got := range []string{
			c.GetHeader(InternalJobsTokenHeader),
			strings.TrimPrefix(c.GetHeader("Authorization"), "Bearer "),
		} {
			got = strings.TrimSpace(got)
			if got != "" && subtle.ConstantTimeCompare([]byte(got), []byte(h.detailToken)) == 1 {
				return true
			}
		}
	}
	host, _, err := net.SplitHostPort(c.Request.RemoteAddr)
	if err != nil {
		host = c.Request.RemoteAddr
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

// respond writes the full report or, for public callers, only its status.
func (h *HealthHandler) respond(c *gin.Context, report health.HealthReport) {
	statusCode := http.StatusOK
	if report.Status == health.StatusUnhealthy {
		statusCode = http.StatusServiceUnavailable
	}
	if h.detailAllowed(c) {
		c.JSON(statusCode, report)
		return
	}
	c.JSON(statusCode, gin.H{"status": report.Status})
}

// NewHealthHandler creates a new health handler
func NewHealthHandler(checker *health.Checker) *HealthHandler {
	return &HealthHandler{
		checker: checker,
	}
}

// Live handles the liveness probe endpoint
// @Summary		Liveness probe
// @Description	Returns 200 if the server is running (does not check dependencies). Kubernetes uses this to determine if the pod should be restarted.
// @Tags			Health
// @Accept			json
// @Produce		json
// @Success		200	{object}	map[string]string
// @Router			/health/live [get]
func (h *HealthHandler) Live(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status": "alive",
	})
}

// Ready handles the readiness probe endpoint
// @Summary		Readiness probe
// @Description	Returns 200 if all critical dependencies are healthy (use as the Cloud Run startup probe). The "schema" check reports migration state (ok/ahead/behind/dirty); behind or dirty returns 503. Redis reports "not_configured" or "in_memory_fallback" without failing. Includes version and git_sha.
// @Tags			Health
// @Accept			json
// @Produce		json
// @Success		200	{object}	health.HealthReport
// @Failure		503	{object}	health.HealthReport
// @Router			/health/ready [get]
func (h *HealthHandler) Ready(c *gin.Context) {
	h.respond(c, h.checker.GetReport(c.Request.Context()))
}

// Health handles the general health endpoint
// @Summary		Health check
// @Description	Returns {"status"} publicly; the detailed report (dependency checks, version, git SHA) only in development, from loopback, or with X-Internal-Jobs-Token / Bearer INTERNAL_JOBS_SECRET
// @Tags			Health
// @Accept			json
// @Produce		json
// @Success		200	{object}	health.HealthReport
// @Failure		503	{object}	health.HealthReport
// @Router			/health [get]
func (h *HealthHandler) Health(c *gin.Context) {
	h.respond(c, h.checker.GetReport(c.Request.Context()))
}
