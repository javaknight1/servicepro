package v1

import (
	"github.com/gin-gonic/gin"

	"github.com/javaknight1/servicepro/backend/internal/api/routeconfigs"
	"github.com/javaknight1/servicepro/backend/internal/permissions"
)

// SetupReportRoutes configures report routes
func SetupReportRoutes(router *gin.RouterGroup, cfg *routeconfigs.RouteConfig) {
	reports := router.Group("/reports")
	reports.Use(cfg.Middleware.PermMiddleware.RequireAuth())
	reports.Use(cfg.Middleware.TenantMW.RequireTenant())
	{
		// Revenue report routes
		setupRevenueReportRoutes(reports, cfg)

		// A/R Aging report routes
		setupARAgingReportRoutes(reports, cfg)
	}
}

func setupRevenueReportRoutes(reports *gin.RouterGroup, cfg *routeconfigs.RouteConfig) {
	// Main revenue report - requires "reports.view" permission
	reports.GET("/revenue",
		cfg.Middleware.PermMiddleware.RequirePermission(permissions.ReportsView),
		cfg.Handlers.Revenue.GetRevenueReport)

	// Revenue time series data - requires "reports.view" permission
	reports.GET("/revenue/time-series",
		cfg.Middleware.PermMiddleware.RequirePermission(permissions.ReportsView),
		cfg.Handlers.Revenue.GetRevenueTimeSeries)

	// Revenue by category - requires "reports.view" permission
	reports.GET("/revenue/categories",
		cfg.Middleware.PermMiddleware.RequirePermission(permissions.ReportsView),
		cfg.Handlers.Revenue.GetRevenueCategoryBreakdown)

	// Top customers by revenue - requires "reports.view" permission
	reports.GET("/revenue/customers",
		cfg.Middleware.PermMiddleware.RequirePermission(permissions.ReportsView),
		cfg.Handlers.Revenue.GetTopCustomers)

	// Chart data for frontend - requires "reports.view" permission
	reports.GET("/revenue/chart-data",
		cfg.Middleware.PermMiddleware.RequirePermission(permissions.ReportsView),
		cfg.Handlers.Revenue.GetChartData)

	// Export report - requires "reports.export" permission
	reports.POST("/revenue/export",
		cfg.Middleware.PermMiddleware.RequirePermission(permissions.ReportsExport),
		cfg.Handlers.Revenue.ExportReport)

	// Refresh cache - requires "reports.admin" permission
	reports.POST("/revenue/cache/refresh",
		cfg.Middleware.PermMiddleware.RequirePermission(permissions.ReportsAdmin),
		cfg.Handlers.Revenue.RefreshCache)
}

func setupARAgingReportRoutes(reports *gin.RouterGroup, cfg *routeconfigs.RouteConfig) {
	// A/R aging routes require tenant context for multi-tenancy
	arAging := reports.Group("/ar-aging")
	{
		// Main A/R aging report - requires "reports.view" permission
		arAging.GET("",
			cfg.Middleware.PermMiddleware.RequirePermission(permissions.ReportsView),
			cfg.Handlers.ARAging.GetARAgingReport)

		// Bucket summary - requires "reports.view" permission
		arAging.GET("/buckets",
			cfg.Middleware.PermMiddleware.RequirePermission(permissions.ReportsView),
			cfg.Handlers.ARAging.GetBucketSummary)

		// Invoices by bucket (drill-down) - requires "reports.view" permission
		arAging.GET("/buckets/:bucket/invoices",
			cfg.Middleware.PermMiddleware.RequirePermission(permissions.ReportsView),
			cfg.Handlers.ARAging.GetInvoicesByBucket)

		// Customer summary - requires "reports.view" permission
		arAging.GET("/customers",
			cfg.Middleware.PermMiddleware.RequirePermission(permissions.ReportsView),
			cfg.Handlers.ARAging.GetCustomerSummary)

		// Export A/R aging report - requires "reports.export" permission
		arAging.POST("/export",
			cfg.Middleware.PermMiddleware.RequirePermission(permissions.ReportsExport),
			cfg.Handlers.ARAging.ExportARAgingReport)
	}
}
