package routes

import (
	"github.com/gin-gonic/gin"

	"github.com/javaknight1/servicepro/backend/internal/api/handlers"
	"github.com/javaknight1/servicepro/backend/internal/api/middleware"
	"github.com/javaknight1/servicepro/backend/internal/permissions"
)

// SetupMembershipRoutes configures all membership-related routes
func SetupMembershipRoutes(
	router *gin.RouterGroup,
	membershipHandler *handlers.MembershipHandler,
	permMiddleware *middleware.PermissionMiddleware,
) {
	// Public route - anyone can view membership tiers (for pricing page)
	router.GET("/membership-tiers", membershipHandler.GetAllTiers)

	// Protected routes - under tenants
	tenants := router.Group("/tenants")
	tenants.Use(permMiddleware.RequireAuth())
	{
		// Get tenant membership - any authenticated user can view their tenant's membership
		tenants.GET("/:id/membership", membershipHandler.GetTenantMembership)

		// Update tenant membership - requires membership.update permission
		tenants.PUT("/:id/membership",
			permMiddleware.RequirePermission(permissions.MembershipUpdate),
			membershipHandler.UpdateTenantMembership)

		// Get subscription history - requires membership.update permission
		tenants.GET("/:id/membership/history",
			permMiddleware.RequirePermission(permissions.MembershipUpdate),
			membershipHandler.GetSubscriptionHistory)
	}
}
