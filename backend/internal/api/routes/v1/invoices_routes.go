package v1

import (
	"time"

	"github.com/gin-gonic/gin"

	"github.com/javaknight1/servicepro/backend/internal/api/middleware"
	"github.com/javaknight1/servicepro/backend/internal/api/routeconfigs"
	"github.com/javaknight1/servicepro/backend/internal/permissions"
)

// SetupInvoiceRoutes configures invoice routes
func SetupInvoiceRoutes(router *gin.RouterGroup, cfg *routeconfigs.RouteConfig) {
	// Create rate limiters
	listRateLimiter := middleware.NewSimpleRateLimiter(100, time.Minute)
	createRateLimiter := middleware.NewSimpleRateLimiter(20, time.Minute)
	updateRateLimiter := middleware.NewSimpleRateLimiter(30, time.Minute)

	// Invoice routes group with authentication
	invoices := router.Group("/invoices")
	invoices.Use(cfg.Middleware.PermMiddleware.RequireAuth())
	{
		// List invoices - requires "invoices.list" permission
		invoices.GET("",
			listRateLimiter.RateLimit(),
			cfg.Middleware.PermMiddleware.RequirePermission(permissions.InvoicesList),
			cfg.Handlers.Invoice.ListInvoices)

		// Get single invoice - requires "invoices.read" permission
		invoices.GET("/:id",
			cfg.Middleware.PermMiddleware.RequirePermission(permissions.InvoicesRead),
			cfg.Handlers.Invoice.GetInvoice)

		// Create invoice - requires "invoices.create" permission
		invoices.POST("",
			createRateLimiter.RateLimit(),
			cfg.Middleware.PermMiddleware.RequirePermission(permissions.InvoicesCreate),
			cfg.Handlers.Invoice.CreateInvoice)

		// Update invoice - requires "invoices.update" permission
		invoices.PUT("/:id",
			updateRateLimiter.RateLimit(),
			cfg.Middleware.PermMiddleware.RequirePermission(permissions.InvoicesUpdate),
			cfg.Handlers.Invoice.UpdateInvoice)

		// Delete invoice - requires "invoices.delete" permission
		invoices.DELETE("/:id",
			cfg.Middleware.PermMiddleware.RequirePermission(permissions.InvoicesDelete),
			cfg.Handlers.Invoice.DeleteInvoice)

		// Send invoice - requires "invoices.execute" permission
		invoices.POST("/:id/send",
			cfg.Middleware.PermMiddleware.RequirePermission(permissions.InvoicesExecute),
			cfg.Handlers.Invoice.SendInvoice)

		// Record payment - requires "payments.create" permission
		invoices.POST("/:id/payments",
			cfg.Middleware.PermMiddleware.RequirePermission(permissions.PaymentsCreate),
			cfg.Handlers.Invoice.RecordPayment)

		// Cancel invoice - requires "invoices.update" permission
		invoices.POST("/:id/cancel",
			cfg.Middleware.PermMiddleware.RequirePermission(permissions.InvoicesUpdate),
			cfg.Handlers.Invoice.CancelInvoice)
	}

	// Public invoice routes (e.g., for customer portal)
	publicRateLimiter := middleware.NewSimpleRateLimiter(10, time.Minute)
	public := router.Group("/public/invoices")
	{
		public.Use(publicRateLimiter.RateLimit())
		public.GET("/:invoice_number", cfg.Handlers.Invoice.GetInvoice)
	}
}
