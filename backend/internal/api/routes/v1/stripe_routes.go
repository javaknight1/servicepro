package v1

import (
	"log"

	"github.com/gin-gonic/gin"

	"github.com/javaknight1/servicepro/backend/internal/api/routeconfigs"
	"github.com/javaknight1/servicepro/backend/internal/permissions"
)

// SetupStripeRoutes configures Stripe-specific routes
func SetupStripeRoutes(router *gin.RouterGroup, cfg *routeconfigs.RouteConfig) {
	// Skip if Stripe is not configured
	if cfg.Handlers.Stripe == nil {
		log.Println("[STRIPE] Stripe not configured, skipping Stripe routes")
		return
	}

	// Determine environment for logging
	environment := "unknown"
	if len(cfg.Config.Stripe.SecretKey) > 8 {
		if cfg.Config.Stripe.SecretKey[:8] == "sk_test_" {
			environment = "test"
		} else if cfg.Config.Stripe.SecretKey[:8] == "sk_live_" {
			environment = "live"
		}
	}

	stripe := router.Group("/stripe")
	{
		// Webhook endpoint (no authentication required, verified by signature)
		stripe.POST("/webhooks", cfg.Handlers.Stripe.HandleWebhook)

		// Apply authentication to all other Stripe routes
		authenticated := stripe.Group("")
		authenticated.Use(cfg.Middleware.PermMiddleware.RequireAuth())
		{
			// Payment Intent endpoints - require payments permissions
			paymentIntents := authenticated.Group("/payment-intents")
			{
				// Create payment intent - requires "payments.create" permission
				paymentIntents.POST("",
					cfg.Middleware.PermMiddleware.RequirePermission(permissions.PaymentsCreate),
					cfg.Handlers.Stripe.CreatePaymentIntent)

				// Get payment intent - requires "payments.read" permission
				paymentIntents.GET("/:id",
					cfg.Middleware.PermMiddleware.RequirePermission(permissions.PaymentsRead),
					cfg.Handlers.Stripe.GetPaymentIntent)

				// Confirm payment intent - requires "payments.create" permission
				paymentIntents.POST("/:id/confirm",
					cfg.Middleware.PermMiddleware.RequirePermission(permissions.PaymentsCreate),
					cfg.Handlers.Stripe.ConfirmPaymentIntent)

				// Cancel payment intent - requires "payments.create" permission
				paymentIntents.POST("/:id/cancel",
					cfg.Middleware.PermMiddleware.RequirePermission(permissions.PaymentsCreate),
					cfg.Handlers.Stripe.CancelPaymentIntent)

				// Capture payment intent - requires "payments.create" permission
				paymentIntents.POST("/:id/capture",
					cfg.Middleware.PermMiddleware.RequirePermission(permissions.PaymentsCreate),
					cfg.Handlers.Stripe.CapturePaymentIntent)
			}

			// Customer endpoints - require customer permissions
			customers := authenticated.Group("/customers")
			{
				// Create Stripe customer - requires "customers.create" permission
				customers.POST("",
					cfg.Middleware.PermMiddleware.RequirePermission(permissions.CustomersCreate),
					cfg.Handlers.Stripe.CreateCustomer)

				// Get Stripe customer - requires "customers.read" permission
				customers.GET("/:id",
					cfg.Middleware.PermMiddleware.RequirePermission(permissions.CustomersRead),
					cfg.Handlers.Stripe.GetCustomer)
			}

			// Refund endpoints - require payments.refund permission
			refunds := authenticated.Group("/refunds")
			{
				// Create refund - requires "payments.refund" permission
				refunds.POST("",
					cfg.Middleware.PermMiddleware.RequirePermission(permissions.PaymentsRefund),
					cfg.Handlers.Stripe.CreateRefund)
			}

			// Webhook statistics (for monitoring) - require reports.admin permission
			webhooks := authenticated.Group("/webhooks")
			{
				webhooks.GET("/stats",
					cfg.Middleware.PermMiddleware.RequirePermission(permissions.ReportsAdmin),
					cfg.Handlers.Stripe.GetWebhookStats)
			}
		}
	}

	log.Printf("[STRIPE] Stripe routes configured successfully in %s mode", environment)
}
