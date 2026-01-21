package routes

import (
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/javaknight1/servicepro/backend/internal/api/handlers"
	"github.com/javaknight1/servicepro/backend/internal/api/middleware"
	"github.com/javaknight1/servicepro/backend/internal/permissions"
	"github.com/javaknight1/servicepro/backend/internal/services"
)

// SetupInvoiceRoutes configures all invoice-related routes
func SetupInvoiceRoutes(router *gin.RouterGroup, db *gorm.DB, jwtSecret string) {
	// Create service and handler
	invoiceService := services.NewInvoiceService(db)
	invoiceHandler := handlers.NewInvoiceHandler(invoiceService)

	// Create rate limiters (in-memory for development, use Redis-based for production)
	listRateLimiter := middleware.NewSimpleRateLimiter(100, time.Minute)  // 100 requests per minute for list
	createRateLimiter := middleware.NewSimpleRateLimiter(20, time.Minute) // 20 creates per minute
	updateRateLimiter := middleware.NewSimpleRateLimiter(30, time.Minute) // 30 updates per minute

	// Invoice routes group
	invoices := router.Group("/invoices")
	{
		// Apply authentication to all invoice routes
		invoices.Use(middleware.AuthMiddleware(jwtSecret))

		// List invoices - GET /api/v1/invoices
		invoices.GET("", listRateLimiter.RateLimit(), invoiceHandler.ListInvoices)

		// Get single invoice - GET /api/v1/invoices/:id
		invoices.GET("/:id", invoiceHandler.GetInvoice)

		// Create invoice - POST /api/v1/invoices
		invoices.POST("", createRateLimiter.RateLimit(), invoiceHandler.CreateInvoice)

		// Update invoice - PUT /api/v1/invoices/:id
		invoices.PUT("/:id", updateRateLimiter.RateLimit(), invoiceHandler.UpdateInvoice)

		// Delete invoice - DELETE /api/v1/invoices/:id
		invoices.DELETE("/:id", invoiceHandler.DeleteInvoice)

		// Send invoice - POST /api/v1/invoices/:id/send
		invoices.POST("/:id/send", invoiceHandler.SendInvoice)

		// Record payment - POST /api/v1/invoices/:id/payments
		invoices.POST("/:id/payments", invoiceHandler.RecordPayment)

		// Cancel invoice - POST /api/v1/invoices/:id/cancel
		invoices.POST("/:id/cancel", invoiceHandler.CancelInvoice)
	}
}

// SetupInvoiceRoutesWithPermissions configures invoice routes with permission middleware
func SetupInvoiceRoutesWithPermissions(
	router *gin.RouterGroup,
	db *gorm.DB,
	permMiddleware *middleware.PermissionMiddleware,
) {
	// Create service and handler
	invoiceService := services.NewInvoiceService(db)
	invoiceHandler := handlers.NewInvoiceHandler(invoiceService)

	// Create rate limiters
	listRateLimiter := middleware.NewSimpleRateLimiter(100, time.Minute)
	createRateLimiter := middleware.NewSimpleRateLimiter(20, time.Minute)
	updateRateLimiter := middleware.NewSimpleRateLimiter(30, time.Minute)

	// Invoice routes group with authentication
	invoices := router.Group("/invoices")
	invoices.Use(permMiddleware.RequireAuth())
	{
		// List invoices - requires "invoices.list" permission
		invoices.GET("",
			listRateLimiter.RateLimit(),
			permMiddleware.RequirePermission(permissions.InvoicesList),
			invoiceHandler.ListInvoices)

		// Get single invoice - requires "invoices.read" permission
		invoices.GET("/:id",
			permMiddleware.RequirePermission(permissions.InvoicesRead),
			invoiceHandler.GetInvoice)

		// Create invoice - requires "invoices.create" permission
		invoices.POST("",
			createRateLimiter.RateLimit(),
			permMiddleware.RequirePermission(permissions.InvoicesCreate),
			invoiceHandler.CreateInvoice)

		// Update invoice - requires "invoices.update" permission
		invoices.PUT("/:id",
			updateRateLimiter.RateLimit(),
			permMiddleware.RequirePermission(permissions.InvoicesUpdate),
			invoiceHandler.UpdateInvoice)

		// Delete invoice - requires "invoices.delete" permission
		invoices.DELETE("/:id",
			permMiddleware.RequirePermission(permissions.InvoicesDelete),
			invoiceHandler.DeleteInvoice)

		// Send invoice - requires "invoices.execute" permission
		invoices.POST("/:id/send",
			permMiddleware.RequirePermission(permissions.InvoicesExecute),
			invoiceHandler.SendInvoice)

		// Record payment - requires "payments.create" permission
		invoices.POST("/:id/payments",
			permMiddleware.RequirePermission(permissions.PaymentsCreate),
			invoiceHandler.RecordPayment)

		// Cancel invoice - requires "invoices.update" permission (or could be a separate permission)
		invoices.POST("/:id/cancel",
			permMiddleware.RequirePermission(permissions.InvoicesUpdate),
			invoiceHandler.CancelInvoice)
	}
}

// SetupPublicInvoiceRoutes sets up public invoice routes (e.g., for customer portal)
func SetupPublicInvoiceRoutes(router *gin.RouterGroup, db *gorm.DB) {
	invoiceService := services.NewInvoiceService(db)
	invoiceHandler := handlers.NewInvoiceHandler(invoiceService)

	// Public rate limiter (more restrictive, in-memory for development)
	publicRateLimiter := middleware.NewSimpleRateLimiter(10, time.Minute)

	public := router.Group("/public/invoices")
	{
		public.Use(publicRateLimiter.RateLimit())

		// Get invoice by invoice number (for customer viewing)
		// This would typically require a token or hash for security
		public.GET("/:invoice_number", invoiceHandler.GetInvoice)
	}
}
