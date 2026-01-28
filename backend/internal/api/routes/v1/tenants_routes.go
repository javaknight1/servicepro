package v1

import (
	"github.com/gin-gonic/gin"

	"github.com/javaknight1/servicepro/backend/internal/api/routeconfigs"
	"github.com/javaknight1/servicepro/backend/internal/permissions"
)

// SetupTenantRoutes configures tenant routes
func SetupTenantRoutes(router *gin.RouterGroup, cfg *routeconfigs.RouteConfig) {
	// Tenant routes group with authentication
	tenants := router.Group("/tenants")
	tenants.Use(cfg.Middleware.PermMiddleware.RequireAuth())
	{
		// List user's tenants - any authenticated user can list their own tenants
		tenants.GET("", cfg.Handlers.Tenant.ListUserTenants)

		// Create tenant - any authenticated user can create a tenant
		tenants.POST("", cfg.Handlers.Tenant.CreateTenant)

		// Get tenant - requires "tenants.read" permission
		tenants.GET("/:id",
			cfg.Middleware.PermMiddleware.RequirePermission(permissions.TenantsRead),
			cfg.Handlers.Tenant.GetTenant)

		// Update tenant - requires "tenants.update" permission
		tenants.PUT("/:id",
			cfg.Middleware.PermMiddleware.RequirePermission(permissions.TenantsUpdate),
			cfg.Handlers.Tenant.UpdateTenant)

		// Delete tenant - requires "tenants.delete" permission
		tenants.DELETE("/:id",
			cfg.Middleware.PermMiddleware.RequirePermission(permissions.TenantsDelete),
			cfg.Handlers.Tenant.DeleteTenant)

		// Member management
		members := tenants.Group("/:id/members")
		{
			// Get members - any tenant member can view the member list
			members.GET("", cfg.Handlers.Tenant.GetTenantMembers)

			// Add member - only tenant admins can add members (checked in handler)
			members.POST("", cfg.Handlers.Tenant.AddMember)

			// Remove member - only tenant admins can remove members (checked in handler)
			members.DELETE("/:userId", cfg.Handlers.Tenant.RemoveMember)

			// Update member role - only tenant admins can update roles (checked in handler)
			members.PUT("/:userId/role", cfg.Handlers.Tenant.UpdateMemberRole)
		}

		// Invitation management
		invitations := tenants.Group("/:id/invitations")
		{
			// Invite member - only tenant admins can invite members
			invitations.POST("", cfg.Handlers.Invitation.InviteMember)

			// Get pending invitations
			invitations.GET("", cfg.Handlers.Invitation.GetPendingInvitations)

			// Cancel invitation
			invitations.DELETE("/:invitation_id", cfg.Handlers.Invitation.CancelInvitation)

			// Resend invitation
			invitations.POST("/:invitation_id/resend", cfg.Handlers.Invitation.ResendInvitation)
		}
	}

	// Invitation routes (mixed auth - some public, some protected)
	invitationsGroup := router.Group("/invitations")
	{
		// Get invitation by token (for registration page) - PUBLIC, no auth required
		invitationsGroup.GET("/:token", cfg.Handlers.Invitation.GetInvitationByToken)

		// Accept invitation (requires authentication) - PROTECTED
		invitationsGroup.POST("/:token/accept",
			cfg.Middleware.PermMiddleware.RequireAuth(),
			cfg.Handlers.Invitation.AcceptInvitation)
	}
}
