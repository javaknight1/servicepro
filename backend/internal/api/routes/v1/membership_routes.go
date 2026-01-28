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

	// Protected routes - under tenants
	tenants := router.Group("/tenants")
	tenants.Use(cfg.Middleware.PermMiddleware.RequireAuth())
	{
		// Get tenant membership - any authenticated user can view their tenant's membership
		tenants.GET("/:id/membership", cfg.Handlers.Membership.GetTenantMembership)

		// Update tenant membership - requires membership.update permission
		tenants.PUT("/:id/membership",
			cfg.Middleware.PermMiddleware.RequirePermission(permissions.MembershipUpdate),
			cfg.Handlers.Membership.UpdateTenantMembership)

		// Get subscription history - requires membership.update permission
		tenants.GET("/:id/membership/history",
			cfg.Middleware.PermMiddleware.RequirePermission(permissions.MembershipUpdate),
			cfg.Handlers.Membership.GetSubscriptionHistory)

		// Preview subscription change - requires membership.update permission
		tenants.POST("/:id/membership/preview",
			cfg.Middleware.PermMiddleware.RequirePermission(permissions.MembershipUpdate),
			cfg.Handlers.Membership.PreviewSubscriptionChange)
	}
}
