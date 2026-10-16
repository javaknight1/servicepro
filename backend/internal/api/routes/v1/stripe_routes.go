package v1

import (
	"log"

	"github.com/gin-gonic/gin"

	"github.com/javaknight1/servicepro/backend/internal/api/routeconfigs"
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
		switch cfg.Config.Stripe.SecretKey[:8] {
		case "sk_test_":
			environment = "test"
		case "sk_live_":
			environment = "live"
		}
	}

	stripe := router.Group("/stripe")
	{
		// Webhook endpoint (no authentication required, verified by signature).
		// The raw payment-intent/customer/refund passthrough routes were
		// removed (T105 / F-4): see handlers.StripeHandler.
		stripe.POST("/webhooks", cfg.Handlers.Stripe.HandleWebhook)
	}

	log.Printf("[STRIPE] Stripe routes configured successfully in %s mode", environment)
}
