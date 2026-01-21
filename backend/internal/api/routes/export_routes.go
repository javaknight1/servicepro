package routes

import (
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/javaknight1/servicepro/backend/internal/api/handlers"
	"github.com/javaknight1/servicepro/backend/internal/api/middleware"
	"github.com/javaknight1/servicepro/backend/internal/permissions"
	"github.com/javaknight1/servicepro/backend/internal/services"
)

// RegisterExportRoutes registers export-related routes (legacy without permissions)
func RegisterExportRoutes(router *gin.RouterGroup, db *gorm.DB, authMiddleware gin.HandlerFunc) {
	// Initialize services
	pdfService := services.NewPDFService("", "", "./exports/pdf")
	exportService := services.NewUnifiedExportService(db, pdfService, "./exports", "")

	// Initialize handler
	exportHandler := handlers.NewExportHandler(exportService)

	// Public routes (no auth required)
	public := router.Group("/exports")
	{
		public.GET("/formats", exportHandler.GetSupportedFormats)
	}

	// Protected routes (require authentication)
	protected := router.Group("/exports")
	protected.Use(authMiddleware)
	{
		// List and create exports
		protected.GET("", exportHandler.ListExports)
		protected.POST("", exportHandler.StartExport)
		protected.POST("/quick", exportHandler.QuickExport)

		// Export job operations
		protected.GET("/:id", exportHandler.GetExportStatus)
		protected.GET("/:id/progress", exportHandler.GetExportProgress)
		protected.GET("/:id/stream", exportHandler.StreamExportProgress)
		protected.GET("/:id/download", exportHandler.DownloadExport)
		protected.POST("/:id/cancel", exportHandler.CancelExport)
	}
}

// RegisterExportRoutesWithPermissions registers export routes with permission middleware
func RegisterExportRoutesWithPermissions(
	router *gin.RouterGroup,
	db *gorm.DB,
	permMiddleware *middleware.PermissionMiddleware,
) {
	// Initialize services
	pdfService := services.NewPDFService("", "", "./exports/pdf")
	exportService := services.NewUnifiedExportService(db, pdfService, "./exports", "")

	// Initialize handler
	exportHandler := handlers.NewExportHandler(exportService)

	// Public routes (no auth required)
	public := router.Group("/exports")
	{
		public.GET("/formats", exportHandler.GetSupportedFormats)
	}

	// Protected routes (require authentication and permissions)
	protected := router.Group("/exports")
	protected.Use(permMiddleware.RequireAuth())
	{
		// List exports - requires "exports.list" permission
		protected.GET("",
			permMiddleware.RequirePermission(permissions.ExportsList),
			exportHandler.ListExports)

		// Create export - requires "exports.create" permission
		protected.POST("",
			permMiddleware.RequirePermission(permissions.ExportsCreate),
			exportHandler.StartExport)

		// Quick export - requires "exports.create" permission
		protected.POST("/quick",
			permMiddleware.RequirePermission(permissions.ExportsCreate),
			exportHandler.QuickExport)

		// Get export status - requires "exports.read" permission
		protected.GET("/:id",
			permMiddleware.RequirePermission(permissions.ExportsRead),
			exportHandler.GetExportStatus)

		// Get export progress - requires "exports.read" permission
		protected.GET("/:id/progress",
			permMiddleware.RequirePermission(permissions.ExportsRead),
			exportHandler.GetExportProgress)

		// Stream export progress - requires "exports.read" permission
		protected.GET("/:id/stream",
			permMiddleware.RequirePermission(permissions.ExportsRead),
			exportHandler.StreamExportProgress)

		// Download export - requires "exports.read" permission
		protected.GET("/:id/download",
			permMiddleware.RequirePermission(permissions.ExportsRead),
			exportHandler.DownloadExport)

		// Cancel export - requires "exports.create" permission (owner can cancel)
		protected.POST("/:id/cancel",
			permMiddleware.RequirePermission(permissions.ExportsCreate),
			exportHandler.CancelExport)
	}
}

// SetupExportCleanupJob sets up a background job to cleanup expired exports
func SetupExportCleanupJob(db *gorm.DB) {
	pdfService := services.NewPDFService("", "", "./exports/pdf")
	exportService := services.NewUnifiedExportService(db, pdfService, "./exports", "")

	// Run cleanup periodically (should be called by a scheduler)
	go func() {
		// This would normally be triggered by a cron job or scheduler
		// For now, we just expose the method to be called externally
		_ = exportService
	}()
}

// ExportMiddleware provides export-related middleware
func ExportMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Add any export-specific middleware logic here
		// For example, rate limiting for exports
		c.Next()
	}
}
