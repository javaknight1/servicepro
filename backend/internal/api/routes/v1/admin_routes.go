package v1

import (
	"github.com/gin-gonic/gin"

	"github.com/javaknight1/servicepro/backend/internal/api/routeconfigs"
)

// SetupAdminRoutes configures platform administration routes. They require a
// PLATFORM super admin (user_roles), which is never assignable via the API.
func SetupAdminRoutes(router *gin.RouterGroup, cfg *routeconfigs.RouteConfig) {
	admin := router.Group("/admin")
	admin.Use(cfg.Middleware.PermMiddleware.RequireAuth())
	admin.Use(cfg.Middleware.PermMiddleware.RequireSuperAdmin())
	{
		admin.POST("/invite-codes", cfg.Handlers.InviteCode.CreateInviteCode)
		admin.GET("/invite-codes", cfg.Handlers.InviteCode.ListInviteCodes)
		admin.DELETE("/invite-codes/:id", cfg.Handlers.InviteCode.RevokeInviteCode)
	}
}
