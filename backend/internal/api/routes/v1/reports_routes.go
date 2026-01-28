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
	{
		// Revenue report routes
		setupRevenueReportRoutes(reports, cfg)

		// Customer report routes
		setupCustomerReportRoutes(reports, cfg)
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

func setupCustomerReportRoutes(reports *gin.RouterGroup, cfg *routeconfigs.RouteConfig) {
	// Main customer report - requires "reports.view" permission
	reports.GET("/customers",
		cfg.Middleware.PermMiddleware.RequirePermission(permissions.ReportsView),
		cfg.Handlers.CustomerReport.GetCustomerReport)

	// Customer segment breakdown - requires "reports.view" permission
	reports.GET("/customers/segments",
		cfg.Middleware.PermMiddleware.RequirePermission(permissions.ReportsView),
		cfg.Handlers.CustomerReport.GetSegmentBreakdown)

	// Customer type breakdown - requires "reports.view" permission
	reports.GET("/customers/types",
		cfg.Middleware.PermMiddleware.RequirePermission(permissions.ReportsView),
		cfg.Handlers.CustomerReport.GetTypeBreakdown)

	// Customer geography breakdown - requires "reports.view" permission
	reports.GET("/customers/geography",
		cfg.Middleware.PermMiddleware.RequirePermission(permissions.ReportsView),
		cfg.Handlers.CustomerReport.GetGeographyBreakdown)

	// Customer acquisition trend - requires "reports.view" permission
	reports.GET("/customers/acquisition",
		cfg.Middleware.PermMiddleware.RequirePermission(permissions.ReportsView),
		cfg.Handlers.CustomerReport.GetAcquisitionTrend)

	// Top customers - requires "reports.view" permission
	reports.GET("/customers/top",
		cfg.Middleware.PermMiddleware.RequirePermission(permissions.ReportsView),
		cfg.Handlers.CustomerReport.GetTopCustomers)

	// At-risk customers - requires "reports.view" permission
	reports.GET("/customers/at-risk",
		cfg.Middleware.PermMiddleware.RequirePermission(permissions.ReportsView),
		cfg.Handlers.CustomerReport.GetAtRiskCustomers)

	// Customer cohort data - requires "reports.view" permission
	reports.GET("/customers/cohorts",
		cfg.Middleware.PermMiddleware.RequirePermission(permissions.ReportsView),
		cfg.Handlers.CustomerReport.GetCohortData)

	// Customer chart data - requires "reports.view" permission
	reports.GET("/customers/chart-data",
		cfg.Middleware.PermMiddleware.RequirePermission(permissions.ReportsView),
		cfg.Handlers.CustomerReport.GetChartData)

	// Export customer report - requires "reports.export" permission
	reports.POST("/customers/export",
		cfg.Middleware.PermMiddleware.RequirePermission(permissions.ReportsExport),
		cfg.Handlers.CustomerReport.ExportReport)

	// Refresh customer metrics cache - requires "reports.admin" permission
	reports.POST("/customers/cache/refresh",
		cfg.Middleware.PermMiddleware.RequirePermission(permissions.ReportsAdmin),
		cfg.Handlers.CustomerReport.RefreshMetricsCache)
}
