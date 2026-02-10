package v1

import (
	"time"

	"github.com/gin-gonic/gin"

	"github.com/javaknight1/servicepro/backend/internal/api/middleware"
	"github.com/javaknight1/servicepro/backend/internal/api/routeconfigs"
	"github.com/javaknight1/servicepro/backend/internal/permissions"
)

// SetupPaymentReminderRoutes configures payment reminder routes
func SetupPaymentReminderRoutes(router *gin.RouterGroup, cfg *routeconfigs.RouteConfig) {
	if cfg.Handlers.PaymentReminder == nil {
		return
	}

	reminderRateLimiter := middleware.NewSimpleRateLimiter(20, time.Minute)

	// Invoice reminder routes (nested under /invoices/:id/reminders)
	invoices := router.Group("/invoices")
	invoices.Use(cfg.Middleware.PermMiddleware.RequireAuth())
	invoices.Use(cfg.Middleware.TenantMW.RequireTenant())
	{
		// Get reminder history for an invoice
		invoices.GET("/:id/reminders",
			reminderRateLimiter.RateLimit(),
			cfg.Middleware.PermMiddleware.RequirePermission(permissions.InvoicesRead),
			cfg.Handlers.PaymentReminder.GetReminderHistory)

		// Send manual reminder for an invoice
		invoices.POST("/:id/reminders",
			reminderRateLimiter.RateLimit(),
			cfg.Middleware.PermMiddleware.RequirePermission(permissions.InvoicesExecute),
			cfg.Handlers.PaymentReminder.SendManualReminder)
	}

	// Settings routes for payment reminders
	settings := router.Group("/settings")
	settings.Use(cfg.Middleware.PermMiddleware.RequireAuth())
	settings.Use(cfg.Middleware.TenantMW.RequireTenant())
	{
		settings.GET("/payment-reminders",
			cfg.Middleware.PermMiddleware.RequirePermission(permissions.SettingsRead),
			cfg.Handlers.PaymentReminder.GetReminderSettings)

		settings.PUT("/payment-reminders",
			cfg.Middleware.PermMiddleware.RequirePermission(permissions.SettingsUpdate),
			cfg.Handlers.PaymentReminder.UpdateReminderSettings)
	}
}
