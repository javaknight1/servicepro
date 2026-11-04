package v1

import (
	"github.com/gin-gonic/gin"

	"github.com/javaknight1/servicepro/backend/internal/api/routeconfigs"
	"github.com/javaknight1/servicepro/backend/internal/permissions"
)

// SetupQuickBooksRoutes configures Settings → Integrations → QuickBooks
// (T192). Everything needs integrations.manage (owners and admins) except
// the OAuth callback, which Intuit's redirect reaches without a session: its
// signed, single-use state names the tenant and user.
func SetupQuickBooksRoutes(router *gin.RouterGroup, cfg *routeconfigs.RouteConfig) {
	h := cfg.Handlers.QuickBooks
	if h == nil {
		return
	}
	router.GET("/integrations/quickbooks/callback", h.Callback)

	perm := cfg.Middleware.PermMiddleware
	g := router.Group("/tenants/:id/integrations/quickbooks")
	g.Use(perm.RequireAuth())
	g.Use(cfg.Middleware.TenantMW.RequireTenantFromPath("id"))
	g.Use(perm.RequirePermission(permissions.IntegrationsManage))
	{
		g.GET("", h.Status)
		g.POST("/connect", h.Connect)
		g.POST("/disconnect", h.Disconnect)
		g.GET("/options", h.Options)
		g.PUT("/settings", h.UpdateSettings)
		g.POST("/sync", h.SyncNow)
		g.POST("/records/:record_id/retry", h.Retry)
	}
}
