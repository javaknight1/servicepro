package v1

import (
	"github.com/gin-gonic/gin"

	"github.com/javaknight1/servicepro/backend/internal/api/routeconfigs"
)

// SetupRoleRoutes configures role and permission routes
func SetupRoleRoutes(router *gin.RouterGroup, cfg *routeconfigs.RouteConfig) {
	roles := router.Group("/roles")
	roles.Use(cfg.Middleware.PermMiddleware.RequireAuth())
	{
		roles.GET("", cfg.Handlers.Role.GetRoles)
		roles.GET("/:id/permissions", cfg.Handlers.Role.GetRolePermissions)
	}

	// Permissions endpoint
	router.GET("/permissions", cfg.Middleware.PermMiddleware.RequireAuth(), cfg.Handlers.Role.GetPermissions)
}
