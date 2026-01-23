package routes

import (
	"log"

	"github.com/gin-gonic/gin"

	"github.com/javaknight1/servicepro/backend/config"
	"github.com/javaknight1/servicepro/backend/internal/api/handlers"
	"github.com/javaknight1/servicepro/backend/internal/api/middleware"
	"github.com/javaknight1/servicepro/backend/internal/permissions"
	stripeService "github.com/javaknight1/servicepro/backend/internal/services/stripe"
)

// SetupStripeRoutes configures all Stripe related routes (legacy without permissions)
func SetupStripeRoutes(router *gin.RouterGroup, jwtSecret string, appCfg *config.Config) error {
	// Create Stripe client using the main config
	client, err := stripeService.NewClient(appCfg)
	if err != nil {
		log.Printf("Failed to create Stripe client: %v", err)
		return err
	}

	// Create event processor and register default handlers
	eventProcessor := stripeService.NewEventProcessor()
	eventProcessor.RegisterDefaultHandlers()

	// Create webhook handler
	webhookHandler := stripeService.NewWebhookHandler(appCfg, eventProcessor)

	// Create HTTP handler
	stripeHandler := handlers.NewStripeHandler(client, webhookHandler)

	// Determine environment for logging
	environment := "unknown"
	if len(appCfg.Stripe.SecretKey) > 8 {
		if appCfg.Stripe.SecretKey[:8] == "sk_test_" {
			environment = "test"
		} else if appCfg.Stripe.SecretKey[:8] == "sk_live_" {
			environment = "live"
		}
	}

	// Stripe routes group
	stripe := router.Group("/stripe")
	{
		// Webhook endpoint (no authentication required, verified by signature)
		stripe.POST("/webhooks", stripeHandler.HandleWebhook)

		// Apply authentication to all other Stripe routes
		authenticated := stripe.Group("")
		authenticated.Use(middleware.AuthMiddleware(jwtSecret))
		{
			// Payment Intent endpoints
			paymentIntents := authenticated.Group("/payment-intents")
			{
				paymentIntents.POST("", stripeHandler.CreatePaymentIntent)
				paymentIntents.GET("/:id", stripeHandler.GetPaymentIntent)
				paymentIntents.POST("/:id/confirm", stripeHandler.ConfirmPaymentIntent)
				paymentIntents.POST("/:id/cancel", stripeHandler.CancelPaymentIntent)
				paymentIntents.POST("/:id/capture", stripeHandler.CapturePaymentIntent)
			}

			// Customer endpoints
			customers := authenticated.Group("/customers")
			{
				customers.POST("", stripeHandler.CreateCustomer)
				customers.GET("/:id", stripeHandler.GetCustomer)
			}

			// Refund endpoints
			refunds := authenticated.Group("/refunds")
			{
				refunds.POST("", stripeHandler.CreateRefund)
			}

			// Webhook statistics (for monitoring)
			webhooks := authenticated.Group("/webhooks")
			{
				webhooks.GET("/stats", stripeHandler.GetWebhookStats)
			}
		}
	}

	log.Printf("Stripe routes configured successfully in %s mode", environment)
	return nil
}

// SetupStripeRoutesWithPermissions configures Stripe routes with permission middleware
func SetupStripeRoutesWithPermissions(
	router *gin.RouterGroup,
	permMiddleware *middleware.PermissionMiddleware,
	appCfg *config.Config,
) error {
	// Create Stripe client using the main config
	client, err := stripeService.NewClient(appCfg)
	if err != nil {
		log.Printf("Failed to create Stripe client: %v", err)
		return err
	}

	// Create event processor and register default handlers
	eventProcessor := stripeService.NewEventProcessor()
	eventProcessor.RegisterDefaultHandlers()

	// Create webhook handler
	webhookHandler := stripeService.NewWebhookHandler(appCfg, eventProcessor)

	// Create HTTP handler
	stripeHandler := handlers.NewStripeHandler(client, webhookHandler)

	// Determine environment for logging
	environment := "unknown"
	if len(appCfg.Stripe.SecretKey) > 8 {
		if appCfg.Stripe.SecretKey[:8] == "sk_test_" {
			environment = "test"
		} else if appCfg.Stripe.SecretKey[:8] == "sk_live_" {
			environment = "live"
		}
	}

	// Stripe routes group
	stripe := router.Group("/stripe")
	{
		// Webhook endpoint (no authentication required, verified by signature)
		stripe.POST("/webhooks", stripeHandler.HandleWebhook)

		// Apply authentication to all other Stripe routes
		authenticated := stripe.Group("")
		authenticated.Use(permMiddleware.RequireAuth())
		{
			// Payment Intent endpoints - require payments permissions
			paymentIntents := authenticated.Group("/payment-intents")
			{
				// Create payment intent - requires "payments.create" permission
				paymentIntents.POST("",
					permMiddleware.RequirePermission(permissions.PaymentsCreate),
					stripeHandler.CreatePaymentIntent)

				// Get payment intent - requires "payments.read" permission
				paymentIntents.GET("/:id",
					permMiddleware.RequirePermission(permissions.PaymentsRead),
					stripeHandler.GetPaymentIntent)

				// Confirm payment intent - requires "payments.create" permission
				paymentIntents.POST("/:id/confirm",
					permMiddleware.RequirePermission(permissions.PaymentsCreate),
					stripeHandler.ConfirmPaymentIntent)

				// Cancel payment intent - requires "payments.create" permission
				paymentIntents.POST("/:id/cancel",
					permMiddleware.RequirePermission(permissions.PaymentsCreate),
					stripeHandler.CancelPaymentIntent)

				// Capture payment intent - requires "payments.create" permission
				paymentIntents.POST("/:id/capture",
					permMiddleware.RequirePermission(permissions.PaymentsCreate),
					stripeHandler.CapturePaymentIntent)
			}

			// Customer endpoints - require customer permissions
			customers := authenticated.Group("/customers")
			{
				// Create Stripe customer - requires "customers.create" permission
				customers.POST("",
					permMiddleware.RequirePermission(permissions.CustomersCreate),
					stripeHandler.CreateCustomer)

				// Get Stripe customer - requires "customers.read" permission
				customers.GET("/:id",
					permMiddleware.RequirePermission(permissions.CustomersRead),
					stripeHandler.GetCustomer)
			}

			// Refund endpoints - require payments.refund permission
			refunds := authenticated.Group("/refunds")
			{
				// Create refund - requires "payments.refund" permission
				refunds.POST("",
					permMiddleware.RequirePermission(permissions.PaymentsRefund),
					stripeHandler.CreateRefund)
			}

			// Webhook statistics (for monitoring) - require reports.admin permission
			webhooks := authenticated.Group("/webhooks")
			{
				webhooks.GET("/stats",
					permMiddleware.RequirePermission(permissions.ReportsAdmin),
					stripeHandler.GetWebhookStats)
			}
		}
	}

	log.Printf("Stripe routes configured successfully with permissions in %s mode", environment)
	return nil
}
