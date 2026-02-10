package v1

import (
	"github.com/gin-gonic/gin"

	"github.com/javaknight1/servicepro/backend/internal/api/routeconfigs"
	"github.com/javaknight1/servicepro/backend/internal/permissions"
)

// SetupConflictRoutes configures conflict detection routes
func SetupConflictRoutes(router *gin.RouterGroup, cfg *routeconfigs.RouteConfig) {
	conflicts := router.Group("/conflicts")
	conflicts.Use(cfg.Middleware.PermMiddleware.RequireAuth())
	conflicts.Use(cfg.Middleware.TenantMW.RequireTenant())
	{
		conflicts.POST("/check",
			cfg.Middleware.PermMiddleware.RequirePermission(permissions.JobsRead),
			cfg.Handlers.Conflict.CheckConflicts)
	}
}
