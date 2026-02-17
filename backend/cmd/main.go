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
	"github.com/javaknight1/servicepro/backend/internal/api/routes"
	analyticsclient "github.com/javaknight1/servicepro/backend/pkg/clients/analytics"
	emailclient "github.com/javaknight1/servicepro/backend/pkg/clients/email"
	errortracking "github.com/javaknight1/servicepro/backend/pkg/clients/errortracking"
	storageclient "github.com/javaknight1/servicepro/backend/pkg/clients/storage"
	"github.com/javaknight1/servicepro/backend/pkg/database"

	// Email providers - blank imports to register providers
	_ "github.com/javaknight1/servicepro/backend/pkg/clients/email/mock"
	_ "github.com/javaknight1/servicepro/backend/pkg/clients/email/resend"
	_ "github.com/javaknight1/servicepro/backend/pkg/clients/email/ses"
	_ "github.com/javaknight1/servicepro/backend/pkg/clients/email/smtp"

	// Storage providers - blank imports to register providers
	_ "github.com/javaknight1/servicepro/backend/pkg/clients/storage/mock"
	_ "github.com/javaknight1/servicepro/backend/pkg/clients/storage/s3"

	// SMS providers - blank imports to register providers
	smsclient "github.com/javaknight1/servicepro/backend/pkg/clients/sms"
	_ "github.com/javaknight1/servicepro/backend/pkg/clients/sms/mock"
	_ "github.com/javaknight1/servicepro/backend/pkg/clients/sms/sns"
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
)

func main() {
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

	// Set Gin mode based on environment
	if cfg.Server.Env == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	// Connect to PostgreSQL using GORM
	db, queryLogger, err := database.NewGormDB(&cfg.Database)
	if err != nil {
		logging.Fatal(ctx, "Failed to connect to database", map[string]any{"error": err.Error()})
		os.Exit(1)
	}
	logging.Info(ctx, "Database connected", nil)

	// Get underlying SQL DB for cleanup
	sqlDB, err := db.DB()
	if err != nil {
		logging.Fatal(ctx, "Failed to get database instance", map[string]any{"error": err.Error()})
		os.Exit(1)
	}
	defer sqlDB.Close()

	// Connect to Redis
	redisClient, err := database.NewRedisClient(&cfg.Redis)
	if err != nil {
		logging.Fatal(ctx, "Failed to connect to Redis", map[string]any{"error": err.Error()})
		os.Exit(1)
	}
	defer redisClient.Close()
	logging.Info(ctx, "Redis connected", nil)

	emailClient, err := emailclient.NewClient(ctx, cfg)
	if err != nil {
		logging.Fatal(ctx, "Failed to initialize email client", map[string]any{"error": err.Error()})
		os.Exit(1)
	}
	logging.Info(ctx, "Email client initialized", nil)

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

		// Start KPI metrics collector (platform overview, revenue, A/R aging)
		kpiCollector := database.NewKPIMetricsCollector(sqlDB, metricsClient, 60*time.Second)
		kpiCollector.Start(ctx)
	}

	// Initialize Gin router
	// Use gin.New() instead of gin.Default() so we can use our custom RecoveryMiddleware
	// that integrates with error tracking (Sentry)
	router := gin.New()
	router.Use(gin.Logger()) // Add Gin's logger middleware

	// Apply Gin-specific configuration
	router.MaxMultipartMemory = cfg.Server.MaxMultipartMemory
	if len(cfg.Server.TrustedProxies) > 0 {
		if err := router.SetTrustedProxies(cfg.Server.TrustedProxies); err != nil {
			logging.Warn(ctx, "Failed to set trusted proxies", map[string]any{"error": err.Error()})
		}
	}

	// Setup routes
	routes.Setup(router, db, redisClient, emailClient, storageClient, smsClient, errorTrackingClient, metricsClient, analyticsClient, cfg)

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
	go func() {
		logging.Info(ctx, "Starting server", map[string]any{"addr": addr})
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logging.Fatal(ctx, "Failed to start server", map[string]any{"error": err.Error()})
			os.Exit(1)
		}
	}()

	// Set up signal handling for graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	logging.Info(ctx, "Shutting down server...", nil)

	// Create shutdown context with timeout
	shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.Server.ShutdownTimeout)
	defer cancel()

	// Attempt graceful shutdown
	if err := srv.Shutdown(shutdownCtx); err != nil {
		logging.Error(ctx, "Server forced to shutdown", map[string]any{"error": err.Error()})
	}

	logging.Info(ctx, "Server exited gracefully", nil)
}
