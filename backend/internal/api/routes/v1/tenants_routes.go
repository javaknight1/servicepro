package v1

import (
	"github.com/gin-gonic/gin"

	"github.com/javaknight1/servicepro/backend/internal/api/routeconfigs"
	"github.com/javaknight1/servicepro/backend/internal/permissions"
)

// SetupTenantRoutes configures tenant routes.
//
// Every /tenants/:id/... route runs RequireTenantFromPath("id") first: the
// caller must be an ACTIVE member of :id, and permissions are then evaluated
// for that tenant only (F-1, F-5). Role-hierarchy rules (no self role change,
// no granting above your own level, super_admin never assignable) are
// enforced in TenantService / InvitationService.
func SetupTenantRoutes(router *gin.RouterGroup, cfg *routeconfigs.RouteConfig) {
	perm := cfg.Middleware.PermMiddleware

	tenants := router.Group("/tenants")
	tenants.Use(perm.RequireAuth())
	{
		// List the caller's own tenants
		tenants.GET("", cfg.Handlers.Tenant.ListUserTenants)

		// Create tenant - any authenticated user; they become its admin
		tenants.POST("", cfg.Handlers.Tenant.CreateTenant)

		tenant := tenants.Group("/:id")
		tenant.Use(cfg.Middleware.TenantMW.RequireTenantFromPath("id"))
		{
			tenant.GET("", perm.RequirePermission(permissions.TenantsRead), cfg.Handlers.Tenant.GetTenant)
			tenant.PUT("", perm.RequirePermission(permissions.TenantsUpdate), cfg.Handlers.Tenant.UpdateTenant)
			tenant.DELETE("", perm.RequirePermission(permissions.TenantsDelete), cfg.Handlers.Tenant.DeleteTenant)

			// Member management
			members := tenant.Group("/members")
			{
				// Any member can view the member list
				members.GET("", perm.RequirePermission(permissions.TenantsRead), cfg.Handlers.Tenant.GetTenantMembers)
				members.POST("", perm.RequirePermission(permissions.TenantsManage), cfg.Handlers.Tenant.AddMember)
				members.DELETE("/:userId", perm.RequirePermission(permissions.TenantsManage), cfg.Handlers.Tenant.RemoveMember)
				members.PUT("/:userId/role", perm.RequirePermission(permissions.TenantsManage), cfg.Handlers.Tenant.UpdateMemberRole)
			}

			// Invitation management
			invitations := tenant.Group("/invitations")
			invitations.Use(perm.RequirePermission(permissions.TenantsManage))
			{
				invitations.POST("", cfg.Handlers.Invitation.InviteMember)
				invitations.GET("", cfg.Handlers.Invitation.GetPendingInvitations)
				invitations.DELETE("/:invitation_id", cfg.Handlers.Invitation.CancelInvitation)
				invitations.POST("/:invitation_id/resend", cfg.Handlers.Invitation.ResendInvitation)
			}
		}
	}

	// Invitation routes (mixed auth - some public, some protected)
	invitationsGroup := router.Group("/invitations")
	{
		// Get invitation by token (for registration page) - PUBLIC, no auth required
		invitationsGroup.GET("/:token", cfg.Handlers.Invitation.GetInvitationByToken)

		// Accept invitation (requires authentication) - PROTECTED
		invitationsGroup.POST("/:token/accept",
			perm.RequireAuth(),
			cfg.Handlers.Invitation.AcceptInvitation)
	}
}
