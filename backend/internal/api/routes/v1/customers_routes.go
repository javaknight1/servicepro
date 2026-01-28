package v1

import (
	"github.com/gin-gonic/gin"

	"github.com/javaknight1/servicepro/backend/internal/api/routeconfigs"
	"github.com/javaknight1/servicepro/backend/internal/permissions"
)

// SetupCustomerRoutes configures customer routes
func SetupCustomerRoutes(router *gin.RouterGroup, cfg *routeconfigs.RouteConfig) {
	customers := router.Group("/customers")
	customers.Use(cfg.Middleware.PermMiddleware.RequireAuth())
	{
		// Create customer - requires "customers.create" permission
		customers.POST("",
			cfg.Middleware.PermMiddleware.RequirePermission(permissions.CustomersCreate),
			cfg.Handlers.Customer.CreateCustomer)

		// List customers - requires "customers.list" permission
		customers.GET("",
			cfg.Middleware.PermMiddleware.RequirePermission(permissions.CustomersList),
			cfg.Handlers.Customer.ListCustomers)

		// Basic search customers - requires "customers.list" permission
		customers.GET("/search",
			cfg.Middleware.PermMiddleware.RequirePermission(permissions.CustomersList),
			cfg.Handlers.Customer.SearchCustomers)

		// Advanced full-text search - requires "customers.list" permission
		customers.GET("/advanced-search",
			cfg.Middleware.PermMiddleware.RequirePermission(permissions.CustomersList),
			cfg.Handlers.Customer.AdvancedSearch)

		// Autocomplete suggestions - requires "customers.list" permission
		customers.GET("/autocomplete",
			cfg.Middleware.PermMiddleware.RequirePermission(permissions.CustomersList),
			cfg.Handlers.Customer.Autocomplete)

		// Search statistics - requires "customers.list" permission
		customers.GET("/stats",
			cfg.Middleware.PermMiddleware.RequirePermission(permissions.CustomersList),
			cfg.Handlers.Customer.GetSearchStats)

		// Get customer by email - requires "customers.read" permission
		customers.GET("/email/:email",
			cfg.Middleware.PermMiddleware.RequirePermission(permissions.CustomersRead),
			cfg.Handlers.Customer.GetCustomerByEmail)

		// Get customer by ID - requires "customers.read" permission
		customers.GET("/:id",
			cfg.Middleware.PermMiddleware.RequirePermission(permissions.CustomersRead),
			cfg.Handlers.Customer.GetCustomer)

		// Update customer - requires "customers.update" permission
		customers.PUT("/:id",
			cfg.Middleware.PermMiddleware.RequirePermission(permissions.CustomersUpdate),
			cfg.Handlers.Customer.UpdateCustomer)

		// Delete customer - requires "customers.delete" permission
		customers.DELETE("/:id",
			cfg.Middleware.PermMiddleware.RequirePermission(permissions.CustomersDelete),
			cfg.Handlers.Customer.DeleteCustomer)

		// Import customers - requires "customers.create" permission
		customers.POST("/import",
			cfg.Middleware.PermMiddleware.RequirePermission(permissions.CustomersCreate),
			cfg.Handlers.ImportExport.ImportCustomers)

		// Get import status - requires "customers.list" permission
		customers.GET("/import/:job_id",
			cfg.Middleware.PermMiddleware.RequirePermission(permissions.CustomersList),
			cfg.Handlers.ImportExport.GetImportStatus)

		// Get import template - requires "customers.list" permission
		customers.GET("/import/template",
			cfg.Middleware.PermMiddleware.RequirePermission(permissions.CustomersList),
			cfg.Handlers.ImportExport.GetImportTemplate)

		// Export customers - requires "customers.list" permission
		customers.GET("/export",
			cfg.Middleware.PermMiddleware.RequirePermission(permissions.CustomersList),
			cfg.Handlers.ImportExport.ExportCustomers)

		// Customer jobs - nested under customers
		customers.GET("/:id/jobs",
			cfg.Middleware.PermMiddleware.RequirePermission(permissions.CustomersRead),
			cfg.Handlers.Job.GetCustomerJobs)

		// Customer quotes - nested under customers
		customers.GET("/:id/quotes",
			cfg.Middleware.PermMiddleware.RequirePermission(permissions.CustomersRead),
			cfg.Handlers.Quote.GetCustomerQuotes)
	}
}
