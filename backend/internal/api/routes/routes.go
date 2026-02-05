package routes

import (
	"context"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	"gorm.io/gorm"

	"github.com/javaknight1/servicepro/backend/config"
	_ "github.com/javaknight1/servicepro/backend/docs" // Import generated swagger docs
	"github.com/javaknight1/servicepro/backend/internal/api/handlers"
	"github.com/javaknight1/servicepro/backend/internal/api/middleware"
	v1 "github.com/javaknight1/servicepro/backend/internal/api/routes/v1"
	"github.com/javaknight1/servicepro/backend/internal/health"
	emailclient "github.com/javaknight1/servicepro/backend/pkg/clients/email"
	"github.com/javaknight1/servicepro/backend/pkg/clients/errortracking"
	"github.com/javaknight1/servicepro/backend/pkg/clients/logging"
	metricsclient "github.com/javaknight1/servicepro/backend/pkg/clients/metrics"
	smsclient "github.com/javaknight1/servicepro/backend/pkg/clients/sms"
	storageclient "github.com/javaknight1/servicepro/backend/pkg/clients/storage"
)

// Setup configures all API routes
func Setup(router *gin.Engine, db *gorm.DB, redisClient *redis.Client, email emailclient.Client, storage storageclient.Client, sms smsclient.Client, errorTracker errortracking.Client, metrics metricsclient.Client, cfg *config.Config) {
	// Apply recovery middleware first to catch panics and send to error tracking
	router.Use(middleware.RecoveryMiddleware(errorTracker))

	// Apply CORS middleware (must handle preflight OPTIONS requests before other middleware)
	router.Use(middleware.CORSMiddleware(&cfg.CORS, cfg.Server.Env))

	// Apply security headers to all responses
	securityConfig := middleware.DefaultSecurityHeadersConfig(cfg.Server.Env)
	router.Use(middleware.SecurityHeaders(securityConfig))

	// Build RouteConfig
	routeCfg := NewRouteConfig(db, redisClient, email, storage, sms, cfg)

	// Setup v1 routes
	v1Group := router.Group("/api/v1")
	v1.Setup(v1Group, routeCfg)

	// Setup health check system
	setupHealthEndpoints(router, db, redisClient, cfg)

	// Setup metrics endpoint for Prometheus scraping
	setupMetricsEndpoint(router, metrics, cfg)

	// Setup Swagger documentation endpoint
	setupSwaggerEndpoint(router, cfg)
}

// setupHealthEndpoints configures health check endpoints with deep dependency checks
func setupHealthEndpoints(router *gin.Engine, db *gorm.DB, redisClient *redis.Client, cfg *config.Config) {
	ctx := context.Background()

	// Get the underlying sql.DB for health checks
	sqlDB, err := db.DB()
	if err != nil {
		logging.Warn(ctx, "Could not get sql.DB for health checks", map[string]any{"error": err.Error(), "source": "health"})
	}

	// Setup health checker with configured checks
	checker, runner, _, _ := health.Setup(health.SetupOptions{
		Config: cfg,
		DB:     sqlDB,
		Redis:  redisClient,
	})

	// Start the background health check runner
	health.StartHealthSystem(runner)

	// Create health handler
	healthHandler := handlers.NewHealthHandler(checker)

	// Register health endpoints
	// /health - Detailed health report (for monitoring dashboards)
	router.GET("/health", healthHandler.Health)

	// /health/live - Liveness probe (is the server running?)
	// Kubernetes restarts the pod if this fails
	router.GET("/health/live", healthHandler.Live)

	// /health/ready - Readiness probe (can the server handle traffic?)
	// Kubernetes removes pod from service if this fails
	router.GET("/health/ready", healthHandler.Ready)

	logging.Info(ctx, "Health check endpoints registered", map[string]any{"endpoints": "/health, /health/live, /health/ready", "source": "health"})
}

// setupSwaggerEndpoint configures the Swagger documentation endpoint
func setupSwaggerEndpoint(router *gin.Engine, cfg *config.Config) {
	ctx := context.Background()

	// Only enable Swagger in non-production environments by default
	// Can be overridden with SWAGGER_ENABLED=true
	if cfg.Server.Env == "production" {
		logging.Info(ctx, "Swagger documentation disabled in production", map[string]any{"source": "swagger"})
		return
	}

	// Serve Swagger UI at /api/docs
	router.GET("/api/docs/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	// Also serve the raw swagger.json
	logging.Info(ctx, "Swagger documentation endpoint registered", map[string]any{"path": "/api/docs/index.html", "source": "swagger"})
}

// setupMetricsEndpoint configures the Prometheus metrics endpoint
func setupMetricsEndpoint(router *gin.Engine, metrics metricsclient.Client, cfg *config.Config) {
	ctx := context.Background()

	if !cfg.Prometheus.Enabled {
		logging.Info(ctx, "Prometheus metrics disabled", map[string]any{"note": "set PROMETHEUS_ENABLED=true to enable", "source": "metrics"})
		return
	}

	if metrics == nil {
		logging.Warn(ctx, "Metrics client is nil, skipping /metrics endpoint", map[string]any{"source": "metrics"})
		return
	}

	metricsPath := cfg.Prometheus.MetricsPath
	if metricsPath == "" {
		metricsPath = "/metrics"
	}

	// Wrap the Prometheus HTTP handler for Gin
	router.GET(metricsPath, gin.WrapH(metrics.Handler()))

	logging.Info(ctx, "Prometheus metrics endpoint registered", map[string]any{"path": metricsPath, "source": "metrics"})
}

// NewRouteConfig creates a new RouteConfig with all dependencies initialized
func NewRouteConfig(db *gorm.DB, redis *redis.Client, email emailclient.Client, storage storageclient.Client, sms smsclient.Client, cfg *config.Config) *RouteConfig {
	repos := SetupRepositories(db)
	mw := SetupMiddleware(repos, cfg, redis)
	svc := SetupServices(db, redis, repos, cfg, email, storage, mw.JWTManager, mw.PermChecker)
	handlers := SetupHandlers(cfg, repos, svc, mw)

	return &RouteConfig{
		Config:     cfg,
		Clients:    &Clients{DB: db, Redis: redis, Email: email, Storage: storage, SMS: sms},
		Repos:      repos,
		Middleware: mw,
		Services:   svc,
		Handlers:   handlers,
	}
}
