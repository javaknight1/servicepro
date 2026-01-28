package v1

import (
	"github.com/gin-gonic/gin"

	"github.com/javaknight1/servicepro/backend/internal/api/routeconfigs"
	"github.com/javaknight1/servicepro/backend/internal/permissions"
)

// SetupPaymentTermsRoutes configures payment terms routes
func SetupPaymentTermsRoutes(router *gin.RouterGroup, cfg *routeconfigs.RouteConfig) {
	// Payment terms routes group with authentication
	paymentTerms := router.Group("/payment-terms")
	paymentTerms.Use(cfg.Middleware.PermMiddleware.RequireAuth())
	{
		// Get default payment term - requires "payment_terms.read" permission
		paymentTerms.GET("/default",
			cfg.Middleware.PermMiddleware.RequirePermission(permissions.PaymentTermsRead),
			cfg.Handlers.PaymentTerms.GetDefaultPaymentTerm)

		// Calculation endpoints - require "payment_terms.read" permission
		calc := paymentTerms.Group("/calculate")
		calc.Use(cfg.Middleware.PermMiddleware.RequirePermission(permissions.PaymentTermsRead))
		{
			calc.POST("/due-date", cfg.Handlers.PaymentTerms.CalculateDueDate)
			calc.POST("/discount", cfg.Handlers.PaymentTerms.CalculateDiscount)
			calc.POST("/late-fee", cfg.Handlers.PaymentTerms.CalculateLateFee)
			calc.POST("/status", cfg.Handlers.PaymentTerms.GetPaymentStatus)
			calc.POST("/details", cfg.Handlers.PaymentTerms.CalculatePaymentDetails)
		}

		// List payment terms - requires "payment_terms.list" permission
		paymentTerms.GET("",
			cfg.Middleware.PermMiddleware.RequirePermission(permissions.PaymentTermsList),
			cfg.Handlers.PaymentTerms.ListPaymentTerms)

		// Get payment term - requires "payment_terms.read" permission
		paymentTerms.GET("/:id",
			cfg.Middleware.PermMiddleware.RequirePermission(permissions.PaymentTermsRead),
			cfg.Handlers.PaymentTerms.GetPaymentTerm)

		// Create payment term - requires "payment_terms.create" permission
		paymentTerms.POST("",
			cfg.Middleware.PermMiddleware.RequirePermission(permissions.PaymentTermsCreate),
			cfg.Handlers.PaymentTerms.CreatePaymentTerm)

		// Update payment term - requires "payment_terms.update" permission
		paymentTerms.PUT("/:id",
			cfg.Middleware.PermMiddleware.RequirePermission(permissions.PaymentTermsUpdate),
			cfg.Handlers.PaymentTerms.UpdatePaymentTerm)

		// Delete payment term - requires "payment_terms.delete" permission
		paymentTerms.DELETE("/:id",
			cfg.Middleware.PermMiddleware.RequirePermission(permissions.PaymentTermsDelete),
			cfg.Handlers.PaymentTerms.DeletePaymentTerm)
	}
}
