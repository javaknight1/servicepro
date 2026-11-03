package v1

import (
	"github.com/gin-gonic/gin"

	"github.com/javaknight1/servicepro/backend/internal/api/routeconfigs"
	"github.com/javaknight1/servicepro/backend/internal/permissions"
)

// SetupAgreementRoutes configures service agreement routes (T167). Agreements
// generate jobs, so they reuse the jobs.* permissions.
func SetupAgreementRoutes(router *gin.RouterGroup, cfg *routeconfigs.RouteConfig) {
	agreements := router.Group("/agreements")
	agreements.Use(cfg.Middleware.PermMiddleware.RequireAuth())
	agreements.Use(cfg.Middleware.TenantMW.RequireTenant())
	// Agreements carry prices and drive the office schedule: office staff
	// only (technicians lack jobs.view_all, T035).
	agreements.Use(cfg.Middleware.PermMiddleware.RequirePermission(permissions.JobsViewAll))
	perm := cfg.Middleware.PermMiddleware.RequirePermission
	h := cfg.Handlers.Agreement
	{
		agreements.GET("", perm(permissions.JobsList), h.ListAgreements)
		agreements.POST("", perm(permissions.JobsCreate), h.CreateAgreement)
		agreements.GET("/:id", perm(permissions.JobsRead), h.GetAgreement)
		agreements.PUT("/:id", perm(permissions.JobsUpdate), h.UpdateAgreement)
		agreements.DELETE("/:id", perm(permissions.JobsDelete), h.DeleteAgreement)
		agreements.POST("/:id/pause", perm(permissions.JobsUpdate), h.PauseAgreement)
		agreements.POST("/:id/resume", perm(permissions.JobsUpdate), h.ResumeAgreement)
		agreements.GET("/:id/preview", perm(permissions.JobsRead), h.PreviewAgreement)
	}
}
