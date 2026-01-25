package routes

import (
	"log"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/javaknight1/servicepro/backend/config"
	"github.com/javaknight1/servicepro/backend/internal/api/handlers"
	"github.com/javaknight1/servicepro/backend/internal/api/middleware"
	"github.com/javaknight1/servicepro/backend/internal/permissions"
	"github.com/javaknight1/servicepro/backend/internal/repository"
	"github.com/javaknight1/servicepro/backend/internal/services"
)

// SetupBillingRoutes configures billing-related routes for payment methods and billing history
func SetupBillingRoutes(
	router *gin.RouterGroup,
	db *gorm.DB,
	permMiddleware *middleware.PermissionMiddleware,
	tenantMiddleware *middleware.TenantMiddleware,
	cfg *config.Config,
) error {
	// Always set up the config endpoint so frontend knows if Stripe is enabled
	router.GET("/billing/config", func(c *gin.Context) {
		stripeEnabled := cfg.Stripe.SecretKey != "" && cfg.Stripe.PublishableKey != ""
		c.JSON(200, gin.H{
			"stripe_enabled":  stripeEnabled,
			"publishable_key": cfg.Stripe.PublishableKey,
		})
	})

	// Check if Stripe is configured for remaining routes
	if cfg.Stripe.SecretKey == "" {
		log.Println("[BILLING] Stripe not configured, skipping billing routes")
		return nil
	}

	// Initialize repositories
	tenantRepo := repository.NewTenantRepository(db)
	membershipRepo := repository.NewMembershipRepository(db)
	paymentRepo := repository.NewPaymentRepository(db)

	// Initialize Stripe service
	stripeService := services.NewStripeService(
		&cfg.Stripe,
		tenantRepo,
		membershipRepo,
		paymentRepo,
	)

	// Initialize billing handler
	billingHandler := handlers.NewBillingHandler(stripeService)

	// Initialize webhook handler
	webhookHandler := handlers.NewWebhookHandler(
		stripeService,
		cfg.Stripe.WebhookSecret,
		tenantRepo,
	)

	// Webhook endpoint (no authentication, verified by Stripe signature)
	webhooks := router.Group("/webhooks")
	{
		webhooks.POST("/stripe", webhookHandler.HandleStripeWebhook)
	}

	// Billing routes under tenants (protected)
	tenants := router.Group("/tenants")
	tenants.Use(permMiddleware.RequireAuth())
	tenants.Use(tenantMiddleware.RequireTenant())
	{
		billing := tenants.Group("/:id/billing")
		{
			// Setup intent for adding payment methods - requires membership.billing permission
			billing.POST("/setup-intent",
				permMiddleware.RequirePermission(permissions.MembershipBilling),
				billingHandler.CreateSetupIntent)

			// Payment methods
			paymentMethods := billing.Group("/payment-methods")
			{
				// List payment methods - requires membership.billing permission
				paymentMethods.GET("",
					permMiddleware.RequirePermission(permissions.MembershipBilling),
					billingHandler.GetPaymentMethods)

				// Add payment method - requires membership.billing permission
				paymentMethods.POST("",
					permMiddleware.RequirePermission(permissions.MembershipBilling),
					billingHandler.AddPaymentMethod)

				// Set default payment method - requires membership.billing permission
				paymentMethods.PUT("/:pmId/default",
					permMiddleware.RequirePermission(permissions.MembershipBilling),
					billingHandler.SetDefaultPaymentMethod)

				// Delete payment method - requires membership.billing permission
				paymentMethods.DELETE("/:pmId",
					permMiddleware.RequirePermission(permissions.MembershipBilling),
					billingHandler.DeletePaymentMethod)
			}

			// Billing history - requires membership.billing permission
			billing.GET("/history",
				permMiddleware.RequirePermission(permissions.MembershipBilling),
				billingHandler.GetBillingHistory)
		}
	}

	log.Println("[BILLING] Billing routes configured successfully")
	return nil
}
