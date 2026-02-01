package routes

import (
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"

	"github.com/javaknight1/servicepro/backend/config"
	"github.com/javaknight1/servicepro/backend/internal/api/middleware"
	v1 "github.com/javaknight1/servicepro/backend/internal/api/routes/v1"
	emailclient "github.com/javaknight1/servicepro/backend/pkg/clients/email"
	smsclient "github.com/javaknight1/servicepro/backend/pkg/clients/sms"
	storageclient "github.com/javaknight1/servicepro/backend/pkg/clients/storage"
)

// Setup configures all API routes
func Setup(router *gin.Engine, db *gorm.DB, redis *redis.Client, email emailclient.Client, storage storageclient.Client, sms smsclient.Client, cfg *config.Config) {
	// Apply CORS middleware first (must handle preflight OPTIONS requests before other middleware)
	router.Use(middleware.CORSMiddleware(&cfg.CORS, cfg.Server.Env))

	// Apply security headers to all responses
	securityConfig := middleware.DefaultSecurityHeadersConfig(cfg.Server.Env)
	router.Use(middleware.SecurityHeaders(securityConfig))

	// Build RouteConfig
	routeCfg := NewRouteConfig(db, redis, email, storage, sms, cfg)

	// Setup v1 routes
	v1Group := router.Group("/api/v1")
	v1.Setup(v1Group, routeCfg)

	// Health check endpoint (basic)
	router.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"status": "healthy",
		})
	})
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
