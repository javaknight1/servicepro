package v1

import (
	"github.com/gin-gonic/gin"

	"github.com/javaknight1/servicepro/backend/internal/api/routeconfigs"
	"github.com/javaknight1/servicepro/backend/internal/permissions"
)

// SetupTaxRateRoutes configures the tax rate routes (T237).
func SetupTaxRateRoutes(router *gin.RouterGroup, cfg *routeconfigs.RouteConfig) {
	taxRates := router.Group("/tax-rates")
	taxRates.Use(cfg.Middleware.PermMiddleware.RequireAuth())
	taxRates.Use(cfg.Middleware.TenantMW.RequireTenant())
	{
		read := cfg.Middleware.PermMiddleware.RequirePermission(permissions.TaxRatesRead)
		manage := cfg.Middleware.PermMiddleware.RequirePermission(permissions.TaxRatesManage)

		taxRates.GET("", read, cfg.Handlers.TaxRate.ListTaxRates)
		taxRates.GET("/:id", read, cfg.Handlers.TaxRate.GetTaxRate)
		taxRates.POST("", manage, cfg.Handlers.TaxRate.CreateTaxRate)
		taxRates.PUT("/:id", manage, cfg.Handlers.TaxRate.UpdateTaxRate)
		taxRates.DELETE("/:id", manage, cfg.Handlers.TaxRate.DeleteTaxRate)
	}
}
