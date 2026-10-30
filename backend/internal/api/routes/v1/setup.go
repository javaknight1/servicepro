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
	SetupWebhookRoutes(router, cfg)
	SetupBillingRoutes(router, cfg)
	setupExportJobRoutes(router, cfg)
	SetupPaymentTermsRoutes(router, cfg)
	SetupTaxRateRoutes(router, cfg)
	SetupConflictRoutes(router, cfg)
	SetupPaymentReminderRoutes(router, cfg)
	SetupAdminRoutes(router, cfg)
}

// setupExportJobRoutes registers the async export-job API only when
// EXPORT_JOBS_ENABLED is set. No screen uses it yet, so it stays off by
// default (T259); report CSV downloads live on their own routes.
func setupExportJobRoutes(router *gin.RouterGroup, cfg *routeconfigs.RouteConfig) {
	if cfg.Config == nil || !cfg.Config.Server.ExportJobsEnabled {
		return
	}
	SetupExportRoutes(router, cfg)
}
