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

	"github.com/gin-gonic/gin"

	"github.com/javaknight1/servicepro/backend/config"
	"github.com/javaknight1/servicepro/backend/internal/api/routes"
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
)

func main() {
	// Load configuration
	cfg := config.Load()

	// Initialize error tracking using the unified client factory
	// Provider is auto-detected based on configuration:
	// - development env -> Mock
	// - Sentry DSN configured -> Sentry
	// - fallback -> Mock
	errorTrackingClient, err := errortracking.NewClient(context.Background(), cfg)
	if err != nil {
		log.Printf("Warning: Failed to initialize error tracking: %v", err)
	} else {
		defer errorTrackingClient.Close()
		log.Println("Error tracking initialized")
	}

	// Set Gin mode based on environment
	if cfg.Server.Env == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	// Connect to PostgreSQL using GORM
	db, err := database.NewGormDB(&cfg.Database)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	// Get underlying SQL DB for cleanup
	sqlDB, err := db.DB()
	if err != nil {
		log.Fatalf("Failed to get database instance: %v", err)
	}
	defer sqlDB.Close()

	// Connect to Redis
	redisClient, err := database.NewRedisClient(&cfg.Redis)
	if err != nil {
		log.Fatalf("Failed to connect to Redis: %v", err)
	}
	defer redisClient.Close()

	emailClient, err := emailclient.NewClient(context.Background(), cfg)
	if err != nil {
		panic(err)
	}

	// Initialize storage client (optional - may not be configured)
	storageClient, _ := storageclient.NewClient(context.Background(), cfg)

	// Initialize SMS client (optional - may not be configured)
	smsClient, err := smsclient.NewClient(context.Background(), cfg)
	if err != nil {
		log.Printf("SMS client not initialized: %v (SMS notifications disabled)", err)
	} else {
		log.Printf("SMS client initialized: %s", smsClient.GetProviderInfo().DisplayName)
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
			log.Printf("Warning: Failed to set trusted proxies: %v", err)
		}
	}

	// Setup routes
	routes.Setup(router, db, redisClient, emailClient, storageClient, smsClient, errorTrackingClient, cfg)

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
		log.Printf("Starting server on %s", addr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	// Set up signal handling for graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutting down server...")

	// Create shutdown context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), cfg.Server.ShutdownTimeout)
	defer cancel()

	// Attempt graceful shutdown
	if err := srv.Shutdown(ctx); err != nil {
		log.Printf("Server forced to shutdown: %v", err)
	}

	log.Println("Server exited gracefully")
}
