package routes

import (
	"context"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"

	"gorm.io/gorm"

	"github.com/javaknight1/servicepro/backend/config"
	"github.com/javaknight1/servicepro/backend/internal/api/handlers"
	"github.com/javaknight1/servicepro/backend/internal/api/middleware"
	"github.com/javaknight1/servicepro/backend/internal/permissions"
	"github.com/javaknight1/servicepro/backend/internal/repository"
	"github.com/javaknight1/servicepro/backend/internal/services"
	permissionsSvc "github.com/javaknight1/servicepro/backend/internal/services/permissions"
	"github.com/javaknight1/servicepro/backend/pkg/auth"
	emailclient "github.com/javaknight1/servicepro/backend/pkg/clients/email"

	// Register email providers (blank imports trigger init() registration)
	_ "github.com/javaknight1/servicepro/backend/pkg/clients/email/mock"
	_ "github.com/javaknight1/servicepro/backend/pkg/clients/email/resend"
	_ "github.com/javaknight1/servicepro/backend/pkg/clients/email/ses"

	// Register storage providers (S3 works for AWS S3, Cloudflare R2, MinIO)
	storageclient "github.com/javaknight1/servicepro/backend/pkg/clients/storage"
	_ "github.com/javaknight1/servicepro/backend/pkg/clients/storage/mock"
	_ "github.com/javaknight1/servicepro/backend/pkg/clients/storage/s3"
)

// Setup configures all API routes
func Setup(router *gin.Engine, db *gorm.DB, redisClient *redis.Client, cfg *config.Config) {
	// Initialize repositories
	userRepo := repository.NewUserRepositoryGorm(db)
	customerRepo := repository.NewCustomerRepository(db)
	permissionRepo := repository.NewPermissionRepository(db)
	jobRepo := repository.NewJobRepository(db)
	quoteRepo := repository.NewQuoteRepository(db)

	// Initialize services
	jwtManager := auth.NewJWTManager(&cfg.JWT)
	authService := services.NewAuthService(userRepo, jwtManager, &cfg.Auth)

	// Initialize email service using the new unified client factory
	// Provider is auto-detected based on configuration:
	// - development env -> Mock
	// - Resend API key present -> Resend
	// - AWS SES configured -> SES
	// - fallback -> Mock
	emailClient, err := emailclient.NewClient(context.Background(), cfg)
	if err != nil {
		// If email client creation fails, this is critical
		panic(err)
	}
	// Initialize email verification service (before registration service)
	// Verification URL uses frontend URL from config
	verificationURL := cfg.Server.FrontendURL + "/verify-email"
	emailVerificationService := services.NewEmailVerificationService(userRepo, emailClient, redisClient, verificationURL)

	registrationService := services.NewRegistrationService(userRepo, emailClient, emailVerificationService)

	// Initialize password reset service
	// Reset URL uses frontend URL from config
	resetURL := cfg.Server.FrontendURL + "/reset-password"
	passwordResetService := services.NewPasswordResetService(userRepo, emailClient, redisClient, resetURL)

	// Initialize permission checker
	permissionChecker := permissionsSvc.NewPermissionChecker(permissionRepo, redisClient)

	// Initialize import service
	importService := services.NewImportService(db, customerRepo, redisClient)

	// Initialize job service
	jobService := services.NewJobService(jobRepo, customerRepo, db)

	// Initialize quote service
	quoteService := services.NewQuoteService(quoteRepo)

	// Initialize revenue service
	revenueService := services.NewRevenueReportService(db)

	// Initialize customer report service
	customerReportService := services.NewCustomerReportService(db)

	// Initialize storage client for profile pictures
	storageClient, err := storageclient.NewClient(context.Background(), cfg)
	if err != nil {
		// Log error but don't fail - storage may not be configured in all environments
		_ = err
	}

	// Initialize profile picture service
	var profilePictureService *services.ProfilePictureService
	if storageClient != nil {
		profilePictureService = services.NewProfilePictureService(userRepo, storageClient)
	}

	// Initialize middleware
	rateLimiter := middleware.NewRateLimiter(redisClient, &cfg.Auth)
	permissionMiddleware := middleware.NewPermissionMiddleware(permissionChecker, jwtManager)

	// Initialize handlers
	authHandler := handlers.NewAuthHandler(authService, rateLimiter)
	registrationHandler := handlers.NewRegistrationHandler(registrationService)
	passwordResetHandler := handlers.NewPasswordResetHandler(passwordResetService)
	emailVerificationHandler := handlers.NewEmailVerificationHandler(emailVerificationService)
	customerHandler := handlers.NewCustomerHandler(customerRepo)
	importExportHandler := handlers.NewImportExportHandler(importService)
	jobHandler := handlers.NewJobHandler(jobService)
	quoteHandler := handlers.NewQuoteHandler(quoteService)
	revenueHandler := handlers.NewRevenueHandler(revenueService)
	customerReportHandler := handlers.NewCustomerReportHandler(customerReportService)

	// Initialize user handler
	userHandler := handlers.NewUserHandler(userRepo)

	// Initialize profile picture handler
	var profilePictureHandler *handlers.ProfilePictureHandler
	if profilePictureService != nil {
		profilePictureHandler = handlers.NewProfilePictureHandler(profilePictureService)
	}

	// API v1 routes
	v1 := router.Group("/api/v1")
	{
		// Auth routes
		auth := v1.Group("/auth")
		{
			auth.POST("/login", rateLimiter.LoginRateLimit(), authHandler.Login)
			auth.POST("/refresh", authHandler.RefreshToken)
			auth.POST("/register", registrationHandler.Register)
			auth.POST("/reset-request", rateLimiter.PasswordResetRateLimit(), passwordResetHandler.RequestPasswordReset)
			auth.POST("/reset-password", passwordResetHandler.ResetPassword)
			auth.GET("/verify", emailVerificationHandler.VerifyEmail)
			auth.POST("/resend-verification", emailVerificationHandler.ResendVerificationEmail)
		}

		// Webhook routes for AWS SES
		webhooks := v1.Group("/webhooks")
		{
			webhooks.POST("/ses-bounce", emailVerificationHandler.HandleSESBounce)
			webhooks.POST("/ses-notification", emailVerificationHandler.HandleSESNotification)
		}

		// User profile routes (protected with JWT authentication)
		users := v1.Group("/users")
		users.Use(permissionMiddleware.RequireAuth())
		{
			// Current user endpoint
			users.GET("/me", userHandler.GetCurrentUser)

			// Profile picture endpoints
			if profilePictureHandler != nil {
				users.POST("/me/profile-picture", profilePictureHandler.UploadProfilePicture)
				users.DELETE("/me/profile-picture", profilePictureHandler.DeleteProfilePicture)
			}
		}

		// Customer routes (protected with JWT authentication and permissions)
		customers := v1.Group("/customers")
		customers.Use(permissionMiddleware.RequireAuth())
		{
			// Create customer - requires "customers.create" permission
			customers.POST("",
				permissionMiddleware.RequirePermission(permissions.CustomersCreate),
				customerHandler.CreateCustomer)

			// List customers - requires "customers.list" permission
			customers.GET("",
				permissionMiddleware.RequirePermission(permissions.CustomersList),
				customerHandler.ListCustomers)

			// Basic search customers - requires "customers.list" permission
			customers.GET("/search",
				permissionMiddleware.RequirePermission(permissions.CustomersList),
				customerHandler.SearchCustomers)

			// Advanced full-text search - requires "customers.list" permission
			customers.GET("/advanced-search",
				permissionMiddleware.RequirePermission(permissions.CustomersList),
				customerHandler.AdvancedSearch)

			// Autocomplete suggestions - requires "customers.list" permission
			customers.GET("/autocomplete",
				permissionMiddleware.RequirePermission(permissions.CustomersList),
				customerHandler.Autocomplete)

			// Search statistics - requires "customers.list" permission
			customers.GET("/stats",
				permissionMiddleware.RequirePermission(permissions.CustomersList),
				customerHandler.GetSearchStats)

			// Get customer by email - requires "customers.read" permission
			customers.GET("/email/:email",
				permissionMiddleware.RequirePermission(permissions.CustomersRead),
				customerHandler.GetCustomerByEmail)

			// Get customer by ID - requires "customers.read" permission
			customers.GET("/:id",
				permissionMiddleware.RequirePermission(permissions.CustomersRead),
				customerHandler.GetCustomer)

			// Update customer - requires "customers.update" permission
			customers.PUT("/:id",
				permissionMiddleware.RequirePermission(permissions.CustomersUpdate),
				customerHandler.UpdateCustomer)

			// Delete customer - requires "customers.delete" permission
			customers.DELETE("/:id",
				permissionMiddleware.RequirePermission(permissions.CustomersDelete),
				customerHandler.DeleteCustomer)

			// Import/Export routes
			// Import customers - requires "customers.create" permission
			customers.POST("/import",
				permissionMiddleware.RequirePermission(permissions.CustomersCreate),
				importExportHandler.ImportCustomers)

			// Get import status - requires "customers.list" permission
			customers.GET("/import/:job_id",
				permissionMiddleware.RequirePermission(permissions.CustomersList),
				importExportHandler.GetImportStatus)

			// Get import template - requires "customers.list" permission
			customers.GET("/import/template",
				permissionMiddleware.RequirePermission(permissions.CustomersList),
				importExportHandler.GetImportTemplate)

			// Export customers - requires "customers.list" permission
			customers.GET("/export",
				permissionMiddleware.RequirePermission(permissions.CustomersList),
				importExportHandler.ExportCustomers)
		}

		// Job routes (protected with JWT authentication and permissions)
		jobs := v1.Group("/jobs")
		jobs.Use(permissionMiddleware.RequireAuth())
		{
			// Create job - requires "jobs.create" permission
			jobs.POST("",
				permissionMiddleware.RequirePermission(permissions.JobsCreate),
				jobHandler.CreateJob)

			// List jobs with filters - requires "jobs.list" permission
			jobs.GET("",
				permissionMiddleware.RequirePermission(permissions.JobsList),
				jobHandler.ListJobs)

			// Get my assigned jobs - requires authentication only
			jobs.GET("/my",
				jobHandler.GetMyJobs)

			// Get scheduled jobs - requires "jobs.list" permission
			jobs.GET("/scheduled",
				permissionMiddleware.RequirePermission(permissions.JobsList),
				jobHandler.GetScheduledJobs)

			// Get overdue jobs - requires "jobs.list" permission
			jobs.GET("/overdue",
				permissionMiddleware.RequirePermission(permissions.JobsList),
				jobHandler.GetOverdueJobs)

			// Get job statistics - requires "jobs.list" permission
			jobs.GET("/stats",
				permissionMiddleware.RequirePermission(permissions.JobsList),
				jobHandler.GetJobStats)

			// Get technician workload - requires "jobs.list" permission
			jobs.GET("/technicians/:id/workload",
				permissionMiddleware.RequirePermission(permissions.JobsList),
				jobHandler.GetTechnicianWorkload)

			// Get job by job number - requires "jobs.read" permission
			jobs.GET("/number/:jobNumber",
				permissionMiddleware.RequirePermission(permissions.JobsRead),
				jobHandler.GetJobByNumber)

			// Get job by ID - requires "jobs.read" permission
			jobs.GET("/:id",
				permissionMiddleware.RequirePermission(permissions.JobsRead),
				jobHandler.GetJob)

			// Update job - requires "jobs.update" permission
			jobs.PUT("/:id",
				permissionMiddleware.RequirePermission(permissions.JobsUpdate),
				jobHandler.UpdateJob)

			// Delete job - requires "jobs.delete" permission
			jobs.DELETE("/:id",
				permissionMiddleware.RequirePermission(permissions.JobsDelete),
				jobHandler.DeleteJob)

			// Job status transitions - require "jobs.update" permission
			jobs.POST("/:id/start",
				permissionMiddleware.RequirePermission(permissions.JobsUpdate),
				jobHandler.StartJob)

			jobs.POST("/:id/complete",
				permissionMiddleware.RequirePermission(permissions.JobsUpdate),
				jobHandler.CompleteJob)

			jobs.POST("/:id/cancel",
				permissionMiddleware.RequirePermission(permissions.JobsDelete),
				jobHandler.CancelJob)

			jobs.POST("/:id/hold",
				permissionMiddleware.RequirePermission(permissions.JobsUpdate),
				jobHandler.PutOnHold)

			jobs.POST("/:id/resume",
				permissionMiddleware.RequirePermission(permissions.JobsUpdate),
				jobHandler.ResumeJob)

			// Job assignments - require "jobs.assign" permission
			jobs.POST("/:id/assign",
				permissionMiddleware.RequirePermission(permissions.JobsAssign),
				jobHandler.AssignTechnician)

			jobs.DELETE("/:id/assign/:technicianId",
				permissionMiddleware.RequirePermission(permissions.JobsAssign),
				jobHandler.UnassignTechnician)

			// Job materials - require "jobs.update" permission
			jobs.POST("/:id/materials",
				permissionMiddleware.RequirePermission(permissions.JobsUpdate),
				jobHandler.AddMaterial)

			// Job notes - require "jobs.update" permission
			jobs.POST("/:id/notes",
				permissionMiddleware.RequirePermission(permissions.JobsUpdate),
				jobHandler.AddNote)
		}

		// Customer jobs - nested under customers
		customers.GET("/:id/jobs",
			permissionMiddleware.RequirePermission(permissions.CustomersRead),
			jobHandler.GetCustomerJobs)

		// Customer quotes - nested under customers
		customers.GET("/:id/quotes",
			permissionMiddleware.RequirePermission(permissions.CustomersRead),
			quoteHandler.GetCustomerQuotes)

		// Quote routes (protected with JWT authentication and permissions)
		quotes := v1.Group("/quotes")
		quotes.Use(permissionMiddleware.RequireAuth())
		{
			// Create quote - requires "quotes.create" permission
			quotes.POST("",
				permissionMiddleware.RequirePermission(permissions.QuotesCreate),
				quoteHandler.CreateQuote)

			// List quotes - requires "quotes.list" permission
			quotes.GET("",
				permissionMiddleware.RequirePermission(permissions.QuotesList),
				quoteHandler.ListQuotes)

			// Get quote statistics - requires "quotes.list" permission
			quotes.GET("/stats",
				permissionMiddleware.RequirePermission(permissions.QuotesList),
				quoteHandler.GetQuoteStats)

			// Get quote by quote number - requires "quotes.read" permission
			quotes.GET("/number/:quoteNumber",
				permissionMiddleware.RequirePermission(permissions.QuotesRead),
				quoteHandler.GetQuoteByNumber)

			// Get quote by ID - requires "quotes.read" permission
			quotes.GET("/:id",
				permissionMiddleware.RequirePermission(permissions.QuotesRead),
				quoteHandler.GetQuote)

			// Update quote - requires "quotes.update" permission
			quotes.PUT("/:id",
				permissionMiddleware.RequirePermission(permissions.QuotesUpdate),
				quoteHandler.UpdateQuote)

			// Delete quote - requires "quotes.delete" permission
			quotes.DELETE("/:id",
				permissionMiddleware.RequirePermission(permissions.QuotesDelete),
				quoteHandler.DeleteQuote)

			// Quote status transitions - require "quotes.update" permission
			quotes.POST("/:id/send",
				permissionMiddleware.RequirePermission(permissions.QuotesUpdate),
				quoteHandler.SendQuote)

			quotes.POST("/:id/accept",
				permissionMiddleware.RequirePermission(permissions.QuotesApprove),
				quoteHandler.AcceptQuote)

			quotes.POST("/:id/reject",
				permissionMiddleware.RequirePermission(permissions.QuotesApprove),
				quoteHandler.RejectQuote)
		}

		// Revenue report routes (protected with JWT authentication and permissions)
		reports := v1.Group("/reports")
		reports.Use(permissionMiddleware.RequireAuth())
		{
			// Main revenue report - requires "reports.view" permission
			reports.GET("/revenue",
				permissionMiddleware.RequirePermission(permissions.ReportsView),
				revenueHandler.GetRevenueReport)

			// Revenue time series data - requires "reports.view" permission
			reports.GET("/revenue/time-series",
				permissionMiddleware.RequirePermission(permissions.ReportsView),
				revenueHandler.GetRevenueTimeSeries)

			// Revenue by category - requires "reports.view" permission
			reports.GET("/revenue/categories",
				permissionMiddleware.RequirePermission(permissions.ReportsView),
				revenueHandler.GetRevenueCategoryBreakdown)

			// Top customers by revenue - requires "reports.view" permission
			reports.GET("/revenue/customers",
				permissionMiddleware.RequirePermission(permissions.ReportsView),
				revenueHandler.GetTopCustomers)

			// Chart data for frontend - requires "reports.view" permission
			reports.GET("/revenue/chart-data",
				permissionMiddleware.RequirePermission(permissions.ReportsView),
				revenueHandler.GetChartData)

			// Export report - requires "reports.export" permission
			reports.POST("/revenue/export",
				permissionMiddleware.RequirePermission(permissions.ReportsExport),
				revenueHandler.ExportReport)

			// Refresh cache - requires "reports.admin" permission
			reports.POST("/revenue/cache/refresh",
				permissionMiddleware.RequirePermission(permissions.ReportsAdmin),
				revenueHandler.RefreshCache)

			// Customer report routes
			// Main customer report - requires "reports.view" permission
			reports.GET("/customers",
				permissionMiddleware.RequirePermission(permissions.ReportsView),
				customerReportHandler.GetCustomerReport)

			// Customer segment breakdown - requires "reports.view" permission
			reports.GET("/customers/segments",
				permissionMiddleware.RequirePermission(permissions.ReportsView),
				customerReportHandler.GetSegmentBreakdown)

			// Customer type breakdown - requires "reports.view" permission
			reports.GET("/customers/types",
				permissionMiddleware.RequirePermission(permissions.ReportsView),
				customerReportHandler.GetTypeBreakdown)

			// Customer geography breakdown - requires "reports.view" permission
			reports.GET("/customers/geography",
				permissionMiddleware.RequirePermission(permissions.ReportsView),
				customerReportHandler.GetGeographyBreakdown)

			// Customer acquisition trend - requires "reports.view" permission
			reports.GET("/customers/acquisition",
				permissionMiddleware.RequirePermission(permissions.ReportsView),
				customerReportHandler.GetAcquisitionTrend)

			// Top customers - requires "reports.view" permission
			reports.GET("/customers/top",
				permissionMiddleware.RequirePermission(permissions.ReportsView),
				customerReportHandler.GetTopCustomers)

			// At-risk customers - requires "reports.view" permission
			reports.GET("/customers/at-risk",
				permissionMiddleware.RequirePermission(permissions.ReportsView),
				customerReportHandler.GetAtRiskCustomers)

			// Customer cohort data - requires "reports.view" permission
			reports.GET("/customers/cohorts",
				permissionMiddleware.RequirePermission(permissions.ReportsView),
				customerReportHandler.GetCohortData)

			// Customer chart data - requires "reports.view" permission
			reports.GET("/customers/chart-data",
				permissionMiddleware.RequirePermission(permissions.ReportsView),
				customerReportHandler.GetChartData)

			// Export customer report - requires "reports.export" permission
			reports.POST("/customers/export",
				permissionMiddleware.RequirePermission(permissions.ReportsExport),
				customerReportHandler.ExportReport)

			// Refresh customer metrics cache - requires "reports.admin" permission
			reports.POST("/customers/cache/refresh",
				permissionMiddleware.RequirePermission(permissions.ReportsAdmin),
				customerReportHandler.RefreshMetricsCache)
		}
	}

	// Health check endpoint (basic)
	router.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"status": "healthy",
		})
	})

	// Setup additional route groups with permissions
	// Invoice routes
	SetupInvoiceRoutesWithPermissions(v1, db, permissionMiddleware)
	SetupPublicInvoiceRoutes(v1, db)

	// Payment terms routes
	SetupPaymentTermsRoutesWithPermissions(v1, db, permissionMiddleware)

	// Export routes
	RegisterExportRoutesWithPermissions(v1, db, permissionMiddleware)

	// Tenant routes
	tenantRepo := repository.NewTenantRepository(db)
	tenantService := services.NewTenantService(tenantRepo, userRepo)
	// Set the permission cache invalidator so tenant membership changes invalidate cached permissions
	tenantService.SetPermissionInvalidator(permissionChecker)
	tenantHandler := handlers.NewTenantHandler(tenantService)
	SetupTenantRoutes(v1, tenantHandler, permissionMiddleware)

	// Role and Permission routes (read-only)
	roleHandler := handlers.NewRoleHandler(permissionRepo)
	roles := v1.Group("/roles")
	roles.Use(permissionMiddleware.RequireAuth())
	{
		roles.GET("", roleHandler.GetRoles)
		roles.GET("/:id/permissions", roleHandler.GetRolePermissions)
	}
	v1.GET("/permissions", permissionMiddleware.RequireAuth(), roleHandler.GetPermissions)

	// Stripe routes (for Stripe-specific endpoints and webhooks)
	if err := SetupStripeRoutesWithPermissions(v1, permissionMiddleware, cfg); err != nil {
		// Log error but don't fail - Stripe may not be configured in all environments
		// In production, this should be handled more strictly
		_ = err
	}
}
