package v1

import (
	"github.com/gin-gonic/gin"

	"github.com/javaknight1/servicepro/backend/internal/api/routeconfigs"
	"github.com/javaknight1/servicepro/backend/internal/permissions"
)

// SetupMembershipRoutes configures membership routes
func SetupMembershipRoutes(router *gin.RouterGroup, cfg *routeconfigs.RouteConfig) {
	// Public route - anyone can view membership tiers (for pricing page)
	router.GET("/membership-tiers", cfg.Handlers.Membership.GetAllTiers)

	// Protected routes - under tenants. The caller must be an active member of
	// :id; permissions are evaluated in that tenant (F-5).
	tenants := router.Group("/tenants/:id")
	tenants.Use(cfg.Middleware.PermMiddleware.RequireAuth())
	tenants.Use(cfg.Middleware.TenantMW.RequireTenantFromPath("id"))
	{
		// Get tenant membership - any member of the tenant
		tenants.GET("/membership",
			cfg.Middleware.PermMiddleware.RequirePermission(permissions.TenantsRead),
			cfg.Handlers.Membership.GetTenantMembership)

		// Update tenant membership - requires membership.update permission
		tenants.PUT("/membership",
			cfg.Middleware.PermMiddleware.RequirePermission(permissions.MembershipUpdate),
			cfg.Handlers.Membership.UpdateTenantMembership)

		// Get subscription history - requires membership.update permission
		tenants.GET("/membership/history",
			cfg.Middleware.PermMiddleware.RequirePermission(permissions.MembershipUpdate),
			cfg.Handlers.Membership.GetSubscriptionHistory)

		// Preview subscription change - requires membership.update permission
		tenants.POST("/membership/preview",
			cfg.Middleware.PermMiddleware.RequirePermission(permissions.MembershipUpdate),
			cfg.Handlers.Membership.PreviewSubscriptionChange)
	}
}
