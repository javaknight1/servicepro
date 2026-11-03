package v1

import (
	"github.com/gin-gonic/gin"

	"github.com/javaknight1/servicepro/backend/internal/api/routeconfigs"
	"github.com/javaknight1/servicepro/backend/internal/permissions"
)

// SetupJobRoutes configures job routes
func SetupJobRoutes(router *gin.RouterGroup, cfg *routeconfigs.RouteConfig) {
	jobs := router.Group("/jobs")
	jobs.Use(cfg.Middleware.PermMiddleware.RequireAuth())
	jobs.Use(cfg.Middleware.TenantMW.RequireTenant())
	{
		// Create job - requires "jobs.create" permission
		jobs.POST("",
			cfg.Middleware.PermMiddleware.RequirePermission(permissions.JobsCreate),
			cfg.Handlers.Job.CreateJob)

		// List jobs with filters - requires "jobs.list" permission
		jobs.GET("",
			cfg.Middleware.PermMiddleware.RequirePermission(permissions.JobsList),
			cfg.Handlers.Job.ListJobs)

		// Get my assigned jobs - requires authentication only
		jobs.GET("/my",
			cfg.Handlers.Job.GetMyJobs)

		// Get scheduled jobs - requires "jobs.list" permission
		jobs.GET("/scheduled",
			cfg.Middleware.PermMiddleware.RequirePermission(permissions.JobsList),
			cfg.Handlers.Job.GetScheduledJobs)

		// Get overdue jobs - requires "jobs.list" permission
		jobs.GET("/overdue",
			cfg.Middleware.PermMiddleware.RequirePermission(permissions.JobsList),
			cfg.Handlers.Job.GetOverdueJobs)

		// Get job statistics - requires "jobs.list" permission
		jobs.GET("/stats",
			cfg.Middleware.PermMiddleware.RequirePermission(permissions.JobsList),
			cfg.Handlers.Job.GetJobStats)

		// Get technician workload - requires "jobs.list" and, since it covers
		// any technician, "jobs.view_all" (office only, T035)
		jobs.GET("/technicians/:id/workload",
			cfg.Middleware.PermMiddleware.RequirePermission(permissions.JobsViewAll),
			cfg.Middleware.PermMiddleware.RequirePermission(permissions.JobsList),
			cfg.Handlers.Job.GetTechnicianWorkload)

		// Get job by job number - requires "jobs.read" permission
		jobs.GET("/number/:jobNumber",
			cfg.Middleware.PermMiddleware.RequirePermission(permissions.JobsRead),
			cfg.Handlers.Job.GetJobByNumber)

		// Get job by ID - requires "jobs.read" permission
		jobs.GET("/:id",
			cfg.Middleware.PermMiddleware.RequirePermission(permissions.JobsRead),
			cfg.Handlers.Job.GetJob)

		// Update job - requires "jobs.update" permission
		jobs.PUT("/:id",
			cfg.Middleware.PermMiddleware.RequirePermission(permissions.JobsUpdate),
			cfg.Handlers.Job.UpdateJob)

		// Delete job - requires "jobs.delete" permission
		jobs.DELETE("/:id",
			cfg.Middleware.PermMiddleware.RequirePermission(permissions.JobsDelete),
			cfg.Handlers.Job.DeleteJob)

		// Status transitions (the job state machine) - require "jobs.update"
		jobs.POST("/:id/transition",
			cfg.Middleware.PermMiddleware.RequirePermission(permissions.JobsUpdate),
			cfg.Handlers.Job.TransitionStatus)

		// Approver OKs for extra work over the customer's not-to-exceed
		// limit (T187): recorded by office staff. Approvals carry amounts, so
		// both need "jobs.view_all" (technicians lack it, T035).
		jobs.POST("/:id/approval",
			cfg.Middleware.PermMiddleware.RequirePermission(permissions.JobsViewAll),
			cfg.Middleware.PermMiddleware.RequirePermission(permissions.JobsUpdate),
			cfg.Handlers.JobApproval.RecordApproval)
		jobs.GET("/:id/approvals",
			cfg.Middleware.PermMiddleware.RequirePermission(permissions.JobsViewAll),
			cfg.Middleware.PermMiddleware.RequirePermission(permissions.JobsRead),
			cfg.Handlers.JobApproval.ListApprovals)

		// Invoice a completed job (T029) - creates an invoice, so it
		// requires "invoices.create"
		jobs.POST("/:id/invoice",
			cfg.Middleware.PermMiddleware.RequirePermission(permissions.InvoicesCreate),
			cfg.Handlers.Invoice.InvoiceJob)

		// Status history endpoint
		jobs.GET("/:id/status-history",
			cfg.Middleware.PermMiddleware.RequirePermission(permissions.JobsRead),
			cfg.Handlers.Job.GetStatusHistory)

		// Job assignments - require "jobs.assign" permission
		jobs.POST("/:id/assign",
			cfg.Middleware.PermMiddleware.RequirePermission(permissions.JobsAssign),
			cfg.Handlers.Job.AssignTechnician)

		jobs.DELETE("/:id/assign/:technicianId",
			cfg.Middleware.PermMiddleware.RequirePermission(permissions.JobsAssign),
			cfg.Handlers.Job.UnassignTechnician)

		// Job materials - require "jobs.update" permission
		jobs.POST("/:id/materials",
			cfg.Middleware.PermMiddleware.RequirePermission(permissions.JobsUpdate),
			cfg.Handlers.Job.AddMaterial)

		// Job notes - require "jobs.update" permission
		jobs.POST("/:id/notes",
			cfg.Middleware.PermMiddleware.RequirePermission(permissions.JobsUpdate),
			cfg.Handlers.Job.AddNote)
	}
}
