//	@title						ServicePro API
//	@version					1.0
//	@description				ServicePro is a field service management platform for service businesses.
//	@description				This API provides endpoints for managing customers, jobs, quotes, invoices, and more.
//
//	@contact.name				ServicePro Support
//	@contact.email				support@servicepro.com
//
//	@license.name				Proprietary
//
//	@host						localhost:8080
//	@BasePath					/api
//
//	@securityDefinitions.apikey	BearerAuth
//	@in							header
//	@name						Authorization
//	@description				Enter your bearer token in the format: Bearer {token}

package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/javaknight1/servicepro/backend/config"
	"github.com/javaknight1/servicepro/backend/internal/api/middleware"
	"github.com/javaknight1/servicepro/backend/internal/api/routes"
	"github.com/javaknight1/servicepro/backend/internal/repository"
	"github.com/javaknight1/servicepro/backend/internal/tenancy"
	analyticsclient "github.com/javaknight1/servicepro/backend/pkg/clients/analytics"
	emailclient "github.com/javaknight1/servicepro/backend/pkg/clients/email"
	errortracking "github.com/javaknight1/servicepro/backend/pkg/clients/errortracking"
	storageclient "github.com/javaknight1/servicepro/backend/pkg/clients/storage"
	"github.com/javaknight1/servicepro/backend/pkg/database"

	// Email providers - blank imports to register providers
	_ "github.com/javaknight1/servicepro/backend/pkg/clients/email/mock"
	_ "github.com/javaknight1/servicepro/backend/pkg/clients/email/resend"
	_ "github.com/javaknight1/servicepro/backend/pkg/clients/email/smtp"

	// Storage providers - blank imports to register providers
	_ "github.com/javaknight1/servicepro/backend/pkg/clients/storage/mock"
	_ "github.com/javaknight1/servicepro/backend/pkg/clients/storage/s3"

	// SMS providers - blank imports to register providers
	smsclient "github.com/javaknight1/servicepro/backend/pkg/clients/sms"
	_ "github.com/javaknight1/servicepro/backend/pkg/clients/sms/mock"
	_ "github.com/javaknight1/servicepro/backend/pkg/clients/sms/textbelt"

	// Analytics providers - blank imports to register providers
	_ "github.com/javaknight1/servicepro/backend/pkg/clients/analytics/mock"
	_ "github.com/javaknight1/servicepro/backend/pkg/clients/analytics/posthog"

	// Metrics providers - blank imports to register providers
	metricsclient "github.com/javaknight1/servicepro/backend/pkg/clients/metrics"
	_ "github.com/javaknight1/servicepro/backend/pkg/clients/metrics/mock"
	_ "github.com/javaknight1/servicepro/backend/pkg/clients/metrics/prometheus"

	// Logging providers - blank imports to register providers
	"github.com/javaknight1/servicepro/backend/pkg/clients/logging"
	_ "github.com/javaknight1/servicepro/backend/pkg/clients/logging/mock"

	// Error tracking providers - blank imports to register providers. Without
	// them errortracking.NewClient always failed and backend errors and panics
	// never reached Sentry (T126).
	_ "github.com/javaknight1/servicepro/backend/pkg/clients/errortracking/mock"
	_ "github.com/javaknight1/servicepro/backend/pkg/clients/errortracking/sentry"
)

func main() {
	if err := run(); err != nil {
		// run's deferred cleanups (Sentry/PostHog flush, DB close) have
		// already executed by the time we exit (T126).
		logging.Error(context.Background(), "Server failed", map[string]any{"error": err.Error()})
		os.Exit(1)
	}
}

// run starts the server and blocks until shutdown. It returns errors instead
// of exiting so deferred flushes and closes always run.
func run() error {
	// Load configuration
	cfg := config.Load()

	// Initialize logging client first so all subsequent initialization can use it
	logClient, err := logging.NewClient(context.Background(), cfg)
	if err != nil {
		log.Printf("Warning: Failed to initialize logging client: %v", err)
	} else {
		logging.SetDefault(logClient)
		defer logClient.Close()
	}

	// Initialize error tracking using the unified client factory
	// Provider is auto-detected based on configuration:
	// - development env -> Mock
	// - Sentry DSN configured -> Sentry
	// - fallback -> Mock
	ctx := context.Background()
	errorTrackingClient, err := errortracking.NewClient(ctx, cfg)
	if err != nil {
		logging.Warn(ctx, "Failed to initialize error tracking", map[string]any{"error": err.Error()})
	} else {
		defer errorTrackingClient.Close()
		logging.Info(ctx, "Error tracking initialized", nil)
	}

	// Initialize analytics client
	analyticsClient, err := analyticsclient.NewClient(ctx, cfg)
	if err != nil {
		logging.Warn(ctx, "Failed to initialize analytics client", map[string]any{"error": err.Error()})
	} else {
		defer analyticsClient.Close()
		logging.Info(ctx, "Analytics client initialized", nil)
	}

	// Gin debug mode only for local development (any deployed env, including
	// preview/staging, runs in release mode).
	if cfg.Server.Env != "development" {
		gin.SetMode(gin.ReleaseMode)
	}

	// Connect to PostgreSQL using GORM
	db, queryLogger, err := database.NewGormDB(&cfg.Database)
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}
	logging.Info(ctx, "Database connected", nil)

	// Scope every query on tenant-owned tables to the caller's tenant (F-3).
	if err := tenancy.Register(db); err != nil {
		return fmt.Errorf("failed to register tenancy plugin: %w", err)
	}

	// Get underlying SQL DB for cleanup
	sqlDB, err := db.DB()
	if err != nil {
		return fmt.Errorf("failed to get database instance: %w", err)
	}
	defer sqlDB.Close()

	// Apply pending migrations (development default). Deployed environments
	// migrate from the release pipeline (`cmd/migrate up`) before deploying.
	if cfg.Database.RunMigrationsOnStartup {
		if err := database.RunMigrations(cfg.Database.URL); err != nil {
			return fmt.Errorf("apply migrations at startup: %w", err)
		}
	}

	// Compare the database schema with the migrations embedded in this build;
	// /health/ready returns 503 while it is behind or dirty.
	schemaMonitor := database.NewSchemaMonitor(cfg.Database.URL)
	logging.Info(ctx, "[MIGRATE] Schema state", map[string]any{
		"status":   schemaMonitor.State().Status,
		"applied":  schemaMonitor.State().Applied,
		"expected": schemaMonitor.State().Expected,
	})

	// Connect to Redis (optional): falls back to an in-process in-memory store
	// when REDIS_URL is unset or Redis is unreachable at boot.
	redisConn, err := database.ConnectRedisOrFallback(&cfg.Redis)
	if err != nil {
		return fmt.Errorf("failed to initialize Redis or its in-memory fallback: %w", err)
	}
	defer redisConn.Close()
	redisClient := redisConn.Client
	logging.Info(ctx, "Redis ready", map[string]any{"mode": redisConn.Mode})

	emailClient, err := emailclient.NewClient(ctx, cfg)
	if err != nil {
		return fmt.Errorf("failed to initialize email client: %w", err)
	}
	logging.Info(ctx, "Email client initialized", nil)
	// Never email addresses that bounced or complained (Resend webhook, T159).
	if sc, ok := emailClient.(interface{ SetSuppressor(emailclient.Suppressor) }); ok {
		sc.SetSuppressor(repository.NewEmailSuppressionRepository(db))
	}

	// Initialize storage client (optional - may not be configured)
	storageClient, err := storageclient.NewClient(ctx, cfg)
	if err != nil {
		logging.Warn(ctx, "Storage client not initialized", map[string]any{"error": err.Error()})
	} else if storageClient != nil {
		logging.Info(ctx, "Storage client initialized", nil)
	}

	// Initialize SMS client (optional - may not be configured)
	smsClient, err := smsclient.NewClient(ctx, cfg)
	if err != nil {
		logging.Warn(ctx, "SMS client not initialized", map[string]any{"error": err.Error(), "note": "SMS notifications disabled"})
		smsClient = nil // reminders fall back to email
	} else {
		logging.Info(ctx, "SMS client initialized", map[string]any{"provider": smsClient.GetProviderInfo().DisplayName})
	}

	// Initialize metrics client (for Prometheus /metrics endpoint)
	metricsClient, err := metricsclient.NewClient(ctx, cfg)
	if err != nil {
		logging.Warn(ctx, "Metrics client not initialized", map[string]any{"error": err.Error()})
	} else {
		defer metricsClient.Close()
		logging.Info(ctx, "Metrics client initialized", nil)

		// Wire metrics into the database query logger
		queryLogger.SetMetricsClient(metricsClient)

		// Start connection pool metrics collector
		poolCollector := database.NewPoolMetricsCollector(sqlDB, metricsClient, 15*time.Second)
		poolCollector.Start(ctx)

		// Start KPI metrics collector (platform overview, revenue, A/R aging).
		// It polls the database every minute, so it is a background worker:
		// off by default outside development (keeps Supabase connections idle).
		if cfg.Server.RunBackgroundWorkers {
			kpiCollector := database.NewKPIMetricsCollector(sqlDB, metricsClient, 60*time.Second)
			kpiCollector.Start(ctx)
		}
	}
	if !cfg.Server.RunBackgroundWorkers {
		logging.Info(ctx, "In-process background workers disabled (RUN_BACKGROUND_WORKERS=false); trigger /internal/jobs/* from a scheduler", nil)
	}

	// Initialize Gin router
	// Use gin.New() instead of gin.Default() so we can use our custom RecoveryMiddleware
	// that integrates with error tracking (Sentry)
	router := gin.New()
	// Correlation ID first, so every response (even an edge rejection) and
	// every log line for the request carries X-Request-ID (T223).
	router.Use(middleware.RequestID())
	// Resolve the real visitor IP from the edge proxy headers before anything
	// (logger, rate limiters) reads c.ClientIP().
	router.Use(middleware.EdgeClientIP(cfg.Server.EdgeProxySecret))
	// In production, /api/* is only reachable through the Cloudflare Worker
	// (it adds the edge secret); direct *.run.app calls are refused (F-6).
	if cfg.Server.Env == "production" && cfg.Server.EdgeProxySecret != "" {
		router.Use(middleware.RequireEdgeProxy())
	}
	router.Use(gin.LoggerWithFormatter(middleware.AccessLogFormatter)) // access log with request_id

	// Apply Gin-specific configuration
	router.MaxMultipartMemory = cfg.Server.MaxMultipartMemory
	if len(cfg.Server.TrustedProxies) > 0 {
		if err := router.SetTrustedProxies(cfg.Server.TrustedProxies); err != nil {
			logging.Warn(ctx, "Failed to set trusted proxies", map[string]any{"error": err.Error()})
		}
	} else {
		// No trusted proxies configured: never trust caller-supplied
		// X-Forwarded-For (Gin trusts every proxy by default, which let any
		// client pick its rate-limit IP, F-7). Behind the edge proxy the
		// visitor IP comes from the verified X-SP-Client-IP header instead.
		if err := router.SetTrustedProxies(nil); err != nil {
			logging.Warn(ctx, "Failed to disable trusted proxies", map[string]any{"error": err.Error()})
		}
	}

	// Setup routes
	routes.Setup(router, db, redisClient, emailClient, storageClient, smsClient, errorTrackingClient, metricsClient, analyticsClient, cfg, routes.PlatformOptions{
		Schema:    schemaMonitor,
		RedisMode: redisConn.Mode,
	})

	// Create HTTP server with timeouts
	addr := fmt.Sprintf(":%s", cfg.Server.Port)
	srv := &http.Server{
		Addr:              addr,
		Handler:           router,
		ReadTimeout:       cfg.Server.ReadTimeout,
		WriteTimeout:      cfg.Server.WriteTimeout,
		IdleTimeout:       cfg.Server.IdleTimeout,
		ReadHeaderTimeout: cfg.Server.ReadHeaderTimeout,
	}

	// Start server in a goroutine
	serverErr := make(chan error, 1)
	go func() {
		logging.Info(ctx, "Starting server", map[string]any{"addr": addr})
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErr <- err
		}
	}()

	// Set up signal handling for graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	select {
	case err := <-serverErr:
		return fmt.Errorf("failed to start server: %w", err)
	case <-quit:
	}
	logging.Info(ctx, "Shutting down server...", nil)

	// Create shutdown context with timeout
	shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.Server.ShutdownTimeout)
	defer cancel()

	// Attempt graceful shutdown
	if err := srv.Shutdown(shutdownCtx); err != nil {
		logging.Error(ctx, "Server forced to shutdown", map[string]any{"error": err.Error()})
	}

	logging.Info(ctx, "Server exited gracefully", nil)
	return nil
}
