package v1

import (
	"log"

	"github.com/gin-gonic/gin"

	"github.com/javaknight1/servicepro/backend/internal/api/routeconfigs"
	"github.com/javaknight1/servicepro/backend/internal/permissions"
)

// SetupBillingRoutes configures billing-related routes for payment methods and billing history
func SetupBillingRoutes(router *gin.RouterGroup, cfg *routeconfigs.RouteConfig) {
	// Always set up the config endpoint so frontend knows if Stripe is enabled
	router.GET("/billing/config", func(c *gin.Context) {
		stripeEnabled := cfg.Config.Stripe.SecretKey != "" && cfg.Config.Stripe.PublishableKey != ""
		c.JSON(200, gin.H{
			"stripe_enabled":  stripeEnabled,
			"publishable_key": cfg.Config.Stripe.PublishableKey,
		})
	})

	// Skip if Stripe/Billing is not configured
	if cfg.Handlers.Billing == nil {
		log.Println("[BILLING] Stripe not configured, skipping billing routes")
		return
	}

	// Billing routes under tenants (protected)
	tenants := router.Group("/tenants")
	tenants.Use(cfg.Middleware.PermMiddleware.RequireAuth())
	tenants.Use(cfg.Middleware.TenantMW.RequireTenant())
	{
		billing := tenants.Group("/:id/billing")
		{
			// Setup intent for adding payment methods - requires membership.billing permission
			billing.POST("/setup-intent",
				cfg.Middleware.PermMiddleware.RequirePermission(permissions.MembershipBilling),
				cfg.Handlers.Billing.CreateSetupIntent)

			// Payment methods
			paymentMethods := billing.Group("/payment-methods")
			{
				// List payment methods - requires membership.billing permission
				paymentMethods.GET("",
					cfg.Middleware.PermMiddleware.RequirePermission(permissions.MembershipBilling),
					cfg.Handlers.Billing.GetPaymentMethods)

				// Add payment method - requires membership.billing permission
				paymentMethods.POST("",
					cfg.Middleware.PermMiddleware.RequirePermission(permissions.MembershipBilling),
					cfg.Handlers.Billing.AddPaymentMethod)

				// Set default payment method - requires membership.billing permission
				paymentMethods.PUT("/:pmId/default",
					cfg.Middleware.PermMiddleware.RequirePermission(permissions.MembershipBilling),
					cfg.Handlers.Billing.SetDefaultPaymentMethod)

				// Delete payment method - requires membership.billing permission
				paymentMethods.DELETE("/:pmId",
					cfg.Middleware.PermMiddleware.RequirePermission(permissions.MembershipBilling),
					cfg.Handlers.Billing.DeletePaymentMethod)
			}

			// Billing history - requires membership.billing permission
			billing.GET("/history",
				cfg.Middleware.PermMiddleware.RequirePermission(permissions.MembershipBilling),
				cfg.Handlers.Billing.GetBillingHistory)
		}
	}

	log.Println("[BILLING] Billing routes configured successfully")
}
