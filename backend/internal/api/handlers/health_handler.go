package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/javaknight1/servicepro/backend/internal/health"
)

// HealthHandler handles health check endpoints
type HealthHandler struct {
	checker *health.Checker
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
// @Description	Returns 200 if all critical dependencies are healthy. Kubernetes uses this to determine if traffic should be sent to the pod.
// @Tags			Health
// @Accept			json
// @Produce		json
// @Success		200	{object}	health.HealthReport
// @Failure		503	{object}	health.HealthReport
// @Router			/health/ready [get]
func (h *HealthHandler) Ready(c *gin.Context) {
	report := h.checker.GetReport(c.Request.Context())

	statusCode := http.StatusOK
	if report.Status == health.StatusUnhealthy {
		statusCode = http.StatusServiceUnavailable
	}

	c.JSON(statusCode, report)
}

// Health handles the general health endpoint
// @Summary		Health check
// @Description	Returns detailed health information including all dependency check results
// @Tags			Health
// @Accept			json
// @Produce		json
// @Success		200	{object}	health.HealthReport
// @Failure		503	{object}	health.HealthReport
// @Router			/health [get]
func (h *HealthHandler) Health(c *gin.Context) {
	report := h.checker.GetReport(c.Request.Context())

	statusCode := http.StatusOK
	if report.Status == health.StatusUnhealthy {
		statusCode = http.StatusServiceUnavailable
	}

	c.JSON(statusCode, report)
}
