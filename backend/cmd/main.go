package main

import (
	"context"
	"fmt"
	"log"

	"github.com/gin-gonic/gin"

	"github.com/javaknight1/servicepro/backend/config"
	"github.com/javaknight1/servicepro/backend/internal/api/routes"
	"github.com/javaknight1/servicepro/backend/pkg/clients/errortracking"
	"github.com/javaknight1/servicepro/backend/pkg/database"

	// Register errortracking providers (blank imports trigger init() registration)
	_ "github.com/javaknight1/servicepro/backend/pkg/clients/errortracking/mock"
	_ "github.com/javaknight1/servicepro/backend/pkg/clients/errortracking/sentry"
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

	// Initialize Gin router
	router := gin.Default()

	// Setup routes
	routes.Setup(router, db, redisClient, cfg)

	// Start server
	addr := fmt.Sprintf(":%s", cfg.Server.Port)
	log.Printf("Starting server on %s", addr)
	if err := router.Run(addr); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
