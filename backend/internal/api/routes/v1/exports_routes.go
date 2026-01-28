package v1

import (
	"github.com/gin-gonic/gin"

	"github.com/javaknight1/servicepro/backend/internal/api/routeconfigs"
	"github.com/javaknight1/servicepro/backend/internal/permissions"
)

// SetupExportRoutes configures export routes
func SetupExportRoutes(router *gin.RouterGroup, cfg *routeconfigs.RouteConfig) {
	// Public routes (no auth required)
	public := router.Group("/exports")
	{
		public.GET("/formats", cfg.Handlers.Export.GetSupportedFormats)
	}

	// Protected routes (require authentication and permissions)
	protected := router.Group("/exports")
	protected.Use(cfg.Middleware.PermMiddleware.RequireAuth())
	{
		// List exports - requires "exports.list" permission
		protected.GET("",
			cfg.Middleware.PermMiddleware.RequirePermission(permissions.ExportsList),
			cfg.Handlers.Export.ListExports)

		// Create export - requires "exports.create" permission
		protected.POST("",
			cfg.Middleware.PermMiddleware.RequirePermission(permissions.ExportsCreate),
			cfg.Handlers.Export.StartExport)

		// Quick export - requires "exports.create" permission
		protected.POST("/quick",
			cfg.Middleware.PermMiddleware.RequirePermission(permissions.ExportsCreate),
			cfg.Handlers.Export.QuickExport)

		// Get export status - requires "exports.read" permission
		protected.GET("/:id",
			cfg.Middleware.PermMiddleware.RequirePermission(permissions.ExportsRead),
			cfg.Handlers.Export.GetExportStatus)

		// Get export progress - requires "exports.read" permission
		protected.GET("/:id/progress",
			cfg.Middleware.PermMiddleware.RequirePermission(permissions.ExportsRead),
			cfg.Handlers.Export.GetExportProgress)

		// Stream export progress - requires "exports.read" permission
		protected.GET("/:id/stream",
			cfg.Middleware.PermMiddleware.RequirePermission(permissions.ExportsRead),
			cfg.Handlers.Export.StreamExportProgress)

		// Download export - requires "exports.read" permission
		protected.GET("/:id/download",
			cfg.Middleware.PermMiddleware.RequirePermission(permissions.ExportsRead),
			cfg.Handlers.Export.DownloadExport)

		// Cancel export - requires "exports.create" permission (owner can cancel)
		protected.POST("/:id/cancel",
			cfg.Middleware.PermMiddleware.RequirePermission(permissions.ExportsCreate),
			cfg.Handlers.Export.CancelExport)
	}
}
