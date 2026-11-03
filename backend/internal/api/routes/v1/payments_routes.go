package v1

import (
	"github.com/gin-gonic/gin"

	"github.com/javaknight1/servicepro/backend/internal/api/routeconfigs"
	"github.com/javaknight1/servicepro/backend/internal/permissions"
)

// SetupPaymentsRoutes configures Stripe Connect onboarding and status for a
// tenant (T169). Both need membership.billing (owners and admins): payouts
// go to the business's bank account.
func SetupPaymentsRoutes(router *gin.RouterGroup, cfg *routeconfigs.RouteConfig) {
	if cfg.Handlers.PaymentsConnect == nil {
		return
	}
	perm := cfg.Middleware.PermMiddleware
	payments := router.Group("/tenants/:id/payments")
	payments.Use(perm.RequireAuth())
	payments.Use(cfg.Middleware.TenantMW.RequireTenantFromPath("id"))
	payments.Use(perm.RequirePermission(permissions.MembershipBilling))
	{
		payments.POST("/connect", cfg.Handlers.PaymentsConnect.Connect)
		payments.GET("/status", cfg.Handlers.PaymentsConnect.Status)
	}
}
