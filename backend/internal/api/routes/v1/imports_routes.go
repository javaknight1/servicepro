package v1

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/javaknight1/servicepro/backend/internal/api/apierror"
	"github.com/javaknight1/servicepro/backend/internal/api/routeconfigs"
	"github.com/javaknight1/servicepro/backend/internal/imports"
	"github.com/javaknight1/servicepro/backend/internal/models"
	"github.com/javaknight1/servicepro/backend/internal/permissions"
)

// importPermissions: per entity, the permission to preview/start/continue
// an import and the one to read imports. They are the entity's own
// create/list permissions (sites are created under customers.update, as on
// /customers/{id}/sites).
var importPermissions = map[imports.Entity][2]string{
	imports.EntityCustomers: {permissions.CustomersCreate, permissions.CustomersList},
	imports.EntitySites:     {permissions.CustomersUpdate, permissions.CustomersList},
	imports.EntityJobs:      {permissions.JobsCreate, permissions.JobsList},
	imports.EntityInvoices:  {permissions.InvoicesCreate, permissions.InvoicesList},
}

// SetupImportRoutes configures /imports/{entity} (T193).
func SetupImportRoutes(router *gin.RouterGroup, cfg *routeconfigs.RouteConfig) {
	g := router.Group("/imports/:entity")
	g.Use(cfg.Middleware.PermMiddleware.RequireAuth())
	g.Use(cfg.Middleware.TenantMW.RequireTenant())

	pm := cfg.Middleware.PermMiddleware
	write := entityPermission(func(e imports.Entity) gin.HandlerFunc { return pm.RequirePermission(importPermissions[e][0]) })
	read := entityPermission(func(e imports.Entity) gin.HandlerFunc { return pm.RequirePermission(importPermissions[e][1]) })
	h := cfg.Handlers.Imports

	g.POST("/preview", write, h.PreviewImport)
	g.POST("", write, h.StartImport)
	g.GET("", read, h.ListImports)
	g.GET("/:job_id", read, h.GetImport)
	g.POST("/:job_id/run", write, h.RunImport)
}

// entityPermission runs the permission check of the URL's entity; an
// unknown entity is 404.
func entityPermission(check func(imports.Entity) gin.HandlerFunc) gin.HandlerFunc {
	checks := map[string]gin.HandlerFunc{}
	for _, e := range imports.Entities {
		checks[string(e)] = check(e)
	}
	return func(c *gin.Context) {
		h, ok := checks[c.Param("entity")]
		if !ok {
			apierror.WriteResponse(c, http.StatusNotFound, models.ErrorResponse{Error: "not_found", Message: "unknown import type"})
			c.Abort()
			return
		}
		h(c)
	}
}
