package routes

import (
	"github.com/redis/go-redis/v9"

	"github.com/javaknight1/servicepro/backend/config"
	"github.com/javaknight1/servicepro/backend/internal/api/middleware"
	permissionsSvc "github.com/javaknight1/servicepro/backend/internal/services/permissions"
	"github.com/javaknight1/servicepro/backend/pkg/auth"
)

// SetupMiddleware initializes all middleware instances
func SetupMiddleware(
	repos *Repositories,
	cfg *config.Config,
	redisClient *redis.Client,
) *Middleware {
	// JWT Manager
	jwtManager := auth.NewJWTManager(&cfg.JWT)

	// Permission checker
	permChecker := permissionsSvc.NewPermissionChecker(repos.Permission, redisClient)

	// Permission middleware
	permMiddleware := middleware.NewPermissionMiddleware(permChecker, jwtManager)

	// Tenant middleware
	tenantMW := middleware.NewTenantMiddleware(repos.Tenant)

	// Rate limiters
	rateLimiter := middleware.NewRateLimiter(redisClient, &cfg.Auth)
	apiRateLimiter := middleware.NewAPIRateLimiter(redisClient, &cfg.RateLimit)

	return &Middleware{
		JWTManager:     jwtManager,
		PermChecker:    permChecker,
		PermMiddleware: permMiddleware,
		TenantMW:       tenantMW,
		RateLimiter:    rateLimiter,
		APIRateLimiter: apiRateLimiter,
	}
}
