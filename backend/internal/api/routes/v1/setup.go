package v1

import (
	"github.com/gin-gonic/gin"

	"github.com/javaknight1/servicepro/backend/internal/api/routeconfigs"
)

// Setup configures all v1 API routes
func Setup(router *gin.RouterGroup, cfg *routeconfigs.RouteConfig) {
	// Apply general rate limiting to all API routes
	router.Use(cfg.Middleware.APIRateLimiter.DynamicRateLimit())

	// Setup route groups
	SetupAuthRoutes(router, cfg)
	SetupWebhookRoutes(router, cfg)
	SetupVersionRoutes(router, cfg)
	SetupUserRoutes(router, cfg)
	SetupCustomerRoutes(router, cfg)
	SetupJobRoutes(router, cfg)
	SetupQuoteRoutes(router, cfg)
	SetupInvoiceRoutes(router, cfg)
	SetupReportRoutes(router, cfg)
	SetupTenantRoutes(router, cfg)
	SetupRoleRoutes(router, cfg)
	SetupMembershipRoutes(router, cfg)
	SetupStripeRoutes(router, cfg)
	SetupBillingRoutes(router, cfg)
	SetupExportRoutes(router, cfg)
	SetupPaymentTermsRoutes(router, cfg)
	SetupConflictRoutes(router, cfg)
	SetupPaymentReminderRoutes(router, cfg)
}
