package v1

import (
	"github.com/gin-gonic/gin"

	"github.com/javaknight1/servicepro/backend/internal/api/routeconfigs"
)

// SetupWebhookRoutes registers provider webhooks verified by signature
// rather than by session (T159). Registered only when configured.
func SetupWebhookRoutes(router *gin.RouterGroup, cfg *routeconfigs.RouteConfig) {
	if cfg.Handlers.ResendWebhook != nil {
		router.POST("/webhooks/resend", cfg.Handlers.ResendWebhook.HandleWebhook)
	}
}
