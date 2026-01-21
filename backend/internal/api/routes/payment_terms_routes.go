package routes

import (
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/javaknight1/servicepro/backend/internal/api/handlers"
	"github.com/javaknight1/servicepro/backend/internal/api/middleware"
	"github.com/javaknight1/servicepro/backend/internal/permissions"
	"github.com/javaknight1/servicepro/backend/internal/services"
)

// SetupPaymentTermsRoutes configures all payment terms related routes (legacy without permissions)
func SetupPaymentTermsRoutes(router *gin.RouterGroup, db *gorm.DB, jwtSecret string) {
	// Create service and handler
	paymentTermsService := services.NewPaymentTermsService(db)
	paymentTermsHandler := handlers.NewPaymentTermsHandler(paymentTermsService)

	// Payment terms routes group
	paymentTerms := router.Group("/payment-terms")
	{
		// Apply authentication to all payment terms routes
		paymentTerms.Use(middleware.AuthMiddleware(jwtSecret))

		// Get default payment term - must be before /:id route
		paymentTerms.GET("/default", paymentTermsHandler.GetDefaultPaymentTerm)

		// Calculation endpoints
		calc := paymentTerms.Group("/calculate")
		{
			calc.POST("/due-date", paymentTermsHandler.CalculateDueDate)
			calc.POST("/discount", paymentTermsHandler.CalculateDiscount)
			calc.POST("/late-fee", paymentTermsHandler.CalculateLateFee)
			calc.POST("/status", paymentTermsHandler.GetPaymentStatus)
			calc.POST("/details", paymentTermsHandler.CalculatePaymentDetails)
		}

		// CRUD endpoints
		paymentTerms.GET("", paymentTermsHandler.ListPaymentTerms)
		paymentTerms.GET("/:id", paymentTermsHandler.GetPaymentTerm)
		paymentTerms.POST("", paymentTermsHandler.CreatePaymentTerm)
		paymentTerms.PUT("/:id", paymentTermsHandler.UpdatePaymentTerm)
		paymentTerms.DELETE("/:id", paymentTermsHandler.DeletePaymentTerm)
	}
}

// SetupPaymentTermsRoutesWithPermissions configures payment terms routes with permission middleware
func SetupPaymentTermsRoutesWithPermissions(
	router *gin.RouterGroup,
	db *gorm.DB,
	permMiddleware *middleware.PermissionMiddleware,
) {
	// Create service and handler
	paymentTermsService := services.NewPaymentTermsService(db)
	paymentTermsHandler := handlers.NewPaymentTermsHandler(paymentTermsService)

	// Payment terms routes group with authentication
	paymentTerms := router.Group("/payment-terms")
	paymentTerms.Use(permMiddleware.RequireAuth())
	{
		// Get default payment term - requires "payment_terms.read" permission
		paymentTerms.GET("/default",
			permMiddleware.RequirePermission(permissions.PaymentTermsRead),
			paymentTermsHandler.GetDefaultPaymentTerm)

		// Calculation endpoints - require "payment_terms.read" permission
		calc := paymentTerms.Group("/calculate")
		calc.Use(permMiddleware.RequirePermission(permissions.PaymentTermsRead))
		{
			calc.POST("/due-date", paymentTermsHandler.CalculateDueDate)
			calc.POST("/discount", paymentTermsHandler.CalculateDiscount)
			calc.POST("/late-fee", paymentTermsHandler.CalculateLateFee)
			calc.POST("/status", paymentTermsHandler.GetPaymentStatus)
			calc.POST("/details", paymentTermsHandler.CalculatePaymentDetails)
		}

		// List payment terms - requires "payment_terms.list" permission
		paymentTerms.GET("",
			permMiddleware.RequirePermission(permissions.PaymentTermsList),
			paymentTermsHandler.ListPaymentTerms)

		// Get payment term - requires "payment_terms.read" permission
		paymentTerms.GET("/:id",
			permMiddleware.RequirePermission(permissions.PaymentTermsRead),
			paymentTermsHandler.GetPaymentTerm)

		// Create payment term - requires "payment_terms.create" permission
		paymentTerms.POST("",
			permMiddleware.RequirePermission(permissions.PaymentTermsCreate),
			paymentTermsHandler.CreatePaymentTerm)

		// Update payment term - requires "payment_terms.update" permission
		paymentTerms.PUT("/:id",
			permMiddleware.RequirePermission(permissions.PaymentTermsUpdate),
			paymentTermsHandler.UpdatePaymentTerm)

		// Delete payment term - requires "payment_terms.delete" permission
		paymentTerms.DELETE("/:id",
			permMiddleware.RequirePermission(permissions.PaymentTermsDelete),
			paymentTermsHandler.DeletePaymentTerm)
	}
}
