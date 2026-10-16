package v1

import (
	"github.com/gin-gonic/gin"

	"github.com/javaknight1/servicepro/backend/internal/api/routeconfigs"
)

// SetupAuthRoutes configures authentication routes
func SetupAuthRoutes(router *gin.RouterGroup, cfg *routeconfigs.RouteConfig) {
	auth := router.Group("/auth")
	{
		auth.POST("/login", cfg.Middleware.RateLimiter.LoginRateLimit(), cfg.Handlers.Auth.Login)
		auth.POST("/refresh", cfg.Handlers.Auth.RefreshToken)
		auth.POST("/logout", cfg.Handlers.Auth.Logout)
		auth.POST("/register", cfg.Handlers.Registration.Register)
		auth.POST("/reset-request", cfg.Middleware.RateLimiter.PasswordResetRateLimit(), cfg.Handlers.PasswordReset.RequestPasswordReset)
		auth.POST("/reset-password", cfg.Handlers.PasswordReset.ResetPassword)
		auth.GET("/verify", cfg.Handlers.EmailVerification.VerifyEmail)
		auth.POST("/resend-verification", cfg.Handlers.EmailVerification.ResendVerificationEmail)
	}
}

// SetupWebhookRoutes configures webhook routes for external services
func SetupWebhookRoutes(router *gin.RouterGroup, cfg *routeconfigs.RouteConfig) {
	webhooks := router.Group("/webhooks")
	{
		// AWS SES webhooks
		webhooks.POST("/ses-bounce", cfg.Handlers.EmailVerification.HandleSESBounce)
		webhooks.POST("/ses-notification", cfg.Handlers.EmailVerification.HandleSESNotification)

		// Stripe webhooks (handled separately in billing routes if configured)
		if cfg.Handlers.Webhook != nil {
			webhooks.POST("/stripe", cfg.Handlers.Webhook.HandleStripeWebhook)
		}
	}
}
