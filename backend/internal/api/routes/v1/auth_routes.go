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
		auth.POST("/neon/provision", cfg.Handlers.NeonAuth.Provision)
		auth.POST("/reset-request", cfg.Middleware.RateLimiter.PasswordResetRateLimit(), cfg.Handlers.PasswordReset.RequestPasswordReset)
		auth.POST("/reset-password", cfg.Handlers.PasswordReset.ResetPassword)
		auth.GET("/verify", cfg.Handlers.EmailVerification.VerifyEmail)
		auth.POST("/resend-verification", cfg.Handlers.EmailVerification.ResendVerificationEmail)
	}
}
