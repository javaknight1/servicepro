package v1

import (
	"github.com/gin-gonic/gin"

	"github.com/javaknight1/servicepro/backend/internal/api/routeconfigs"
	"github.com/javaknight1/servicepro/backend/internal/permissions"
)

// SetupQuoteRoutes configures quote routes
func SetupQuoteRoutes(router *gin.RouterGroup, cfg *routeconfigs.RouteConfig) {
	quotes := router.Group("/quotes")
	quotes.Use(cfg.Middleware.PermMiddleware.RequireAuth())
	{
		// Create quote - requires "quotes.create" permission
		quotes.POST("",
			cfg.Middleware.PermMiddleware.RequirePermission(permissions.QuotesCreate),
			cfg.Handlers.Quote.CreateQuote)

		// List quotes - requires "quotes.list" permission
		quotes.GET("",
			cfg.Middleware.PermMiddleware.RequirePermission(permissions.QuotesList),
			cfg.Handlers.Quote.ListQuotes)

		// Get quote statistics - requires "quotes.list" permission
		quotes.GET("/stats",
			cfg.Middleware.PermMiddleware.RequirePermission(permissions.QuotesList),
			cfg.Handlers.Quote.GetQuoteStats)

		// Get quote by quote number - requires "quotes.read" permission
		quotes.GET("/number/:quoteNumber",
			cfg.Middleware.PermMiddleware.RequirePermission(permissions.QuotesRead),
			cfg.Handlers.Quote.GetQuoteByNumber)

		// Get quote by ID - requires "quotes.read" permission
		quotes.GET("/:id",
			cfg.Middleware.PermMiddleware.RequirePermission(permissions.QuotesRead),
			cfg.Handlers.Quote.GetQuote)

		// Update quote - requires "quotes.update" permission
		quotes.PUT("/:id",
			cfg.Middleware.PermMiddleware.RequirePermission(permissions.QuotesUpdate),
			cfg.Handlers.Quote.UpdateQuote)

		// Delete quote - requires "quotes.delete" permission
		quotes.DELETE("/:id",
			cfg.Middleware.PermMiddleware.RequirePermission(permissions.QuotesDelete),
			cfg.Handlers.Quote.DeleteQuote)

		// Quote status transitions - require "quotes.update" permission
		quotes.POST("/:id/send",
			cfg.Middleware.PermMiddleware.RequirePermission(permissions.QuotesUpdate),
			cfg.Handlers.Quote.SendQuote)

		quotes.POST("/:id/accept",
			cfg.Middleware.PermMiddleware.RequirePermission(permissions.QuotesApprove),
			cfg.Handlers.Quote.AcceptQuote)

		quotes.POST("/:id/reject",
			cfg.Middleware.PermMiddleware.RequirePermission(permissions.QuotesApprove),
			cfg.Handlers.Quote.RejectQuote)

		// Get quote PDF download URL - requires "quotes.read" permission
		quotes.GET("/:id/pdf",
			cfg.Middleware.PermMiddleware.RequirePermission(permissions.QuotesRead),
			cfg.Handlers.Quote.GetQuotePDF)
	}
}
