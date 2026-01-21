package routes

import (
	"github.com/gin-gonic/gin"

	"github.com/javaknight1/servicepro/backend/internal/api/handlers"
	"github.com/javaknight1/servicepro/backend/internal/api/middleware"
	"github.com/javaknight1/servicepro/backend/internal/permissions"
)

// SetupTenantRoutes configures all tenant-related routes
func SetupTenantRoutes(
	router *gin.RouterGroup,
	tenantHandler *handlers.TenantHandler,
	permMiddleware *middleware.PermissionMiddleware,
) {
	// Tenant routes group with authentication
	tenants := router.Group("/tenants")
	tenants.Use(permMiddleware.RequireAuth())
	{
		// List user's tenants - any authenticated user can list their own tenants
		// (Returns only tenants the user belongs to, so no permission check needed)
		tenants.GET("", tenantHandler.ListUserTenants)

		// Create tenant - any authenticated user can create a tenant
		// (They become the owner/admin of the new tenant)
		tenants.POST("", tenantHandler.CreateTenant)

		// Get tenant - requires "tenants.read" permission
		tenants.GET("/:id",
			permMiddleware.RequirePermission(permissions.TenantsRead),
			tenantHandler.GetTenant)

		// Update tenant - requires "tenants.update" permission
		tenants.PUT("/:id",
			permMiddleware.RequirePermission(permissions.TenantsUpdate),
			tenantHandler.UpdateTenant)

		// Delete tenant - requires "tenants.delete" permission
		tenants.DELETE("/:id",
			permMiddleware.RequirePermission(permissions.TenantsDelete),
			tenantHandler.DeleteTenant)

		// Member management
		// Note: These endpoints check tenant membership in the handler
		// rather than using permission middleware (which checks user_roles, not tenant_users)
		members := tenants.Group("/:id/members")
		{
			// Get members - any tenant member can view the member list
			members.GET("", tenantHandler.GetTenantMembers)

			// Add member - only tenant admins can add members (checked in handler)
			members.POST("", tenantHandler.AddMember)

			// Remove member - only tenant admins can remove members (checked in handler)
			members.DELETE("/:userId", tenantHandler.RemoveMember)

			// Update member role - only tenant admins can update roles (checked in handler)
			members.PUT("/:userId/role", tenantHandler.UpdateMemberRole)
		}
	}
}
