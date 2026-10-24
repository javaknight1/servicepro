package routes

import (
	"context"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"

	"github.com/javaknight1/servicepro/backend/config"
	"github.com/javaknight1/servicepro/backend/internal/api/middleware"
	permissionsSvc "github.com/javaknight1/servicepro/backend/internal/services/permissions"
	"github.com/javaknight1/servicepro/backend/pkg/auth"
	"github.com/javaknight1/servicepro/backend/pkg/auth/neonauth"
	"github.com/javaknight1/servicepro/backend/pkg/clients/logging"
)

// SetupMiddleware initializes all middleware instances
func SetupMiddleware(
	repos *Repositories,
	cfg *config.Config,
	redisClient *redis.Client,
) *Middleware {
	// JWT Manager
	jwtManager := auth.NewJWTManager(&cfg.JWT)

	// Cookie Manager for httpOnly JWT cookies
	cookieManager := auth.NewCookieManager(&cfg.Cookie, cfg.JWT.AccessExpiration, cfg.JWT.RefreshExpiration)

	// Permission checker
	permChecker := permissionsSvc.NewPermissionChecker(repos.Permission, redisClient)

	// Permission middleware (with cookie support)
	permMiddleware := middleware.NewPermissionMiddleware(permChecker, jwtManager, cookieManager, cfg.Cookie.AccessTokenName)
	// Revocation: reject tokens issued before a logout / password reset.
	if versions, ok := repos.User.(interface {
		GetTokenVersion(uuid.UUID) (int, error)
	}); ok {
		permMiddleware.SetTokenVersionLookup(versions.GetTokenVersion)
	}
	// Neon Auth (T158): also accept Neon Auth JWTs for linked users.
	neonVerifier := newNeonAuthVerifier(cfg)
	if authUsers, ok := repos.User.(interface {
		GetIDByAuthUserID(string) (uuid.UUID, error)
	}); ok && neonVerifier != nil {
		permMiddleware.SetNeonAuth(neonVerifier, authUsers.GetIDByAuthUserID)
	}

	// Tenant middleware
	tenantMW := middleware.NewTenantMiddleware(repos.Tenant)

	// Rate limiters
	rateLimiter := middleware.NewRateLimiter(redisClient, &cfg.Auth)
	apiRateLimiter := middleware.NewAPIRateLimiter(redisClient, &cfg.RateLimit)
	// Key authenticated traffic per user (F-22): the limiter runs before
	// RequireAuth, so it validates the access token itself.
	apiRateLimiter.SetIdentityResolver(func(c *gin.Context) (string, bool) {
		token, _ := cookieManager.GetAccessToken(c)
		if token == "" {
			if h := c.GetHeader("Authorization"); strings.HasPrefix(h, "Bearer ") {
				token = strings.TrimPrefix(h, "Bearer ")
			}
		}
		if token == "" {
			return "", false
		}
		if neonVerifier != nil && neonauth.IsNeonToken(token) {
			claims, err := neonVerifier.Verify(c.Request.Context(), token)
			if err != nil {
				return "", false
			}
			return "neon:" + claims.Subject, true
		}
		claims, err := jwtManager.ValidateToken(token)
		if err != nil || claims.Type != auth.AccessToken {
			return "", false
		}
		return claims.UserID, true
	})

	return &Middleware{
		JWTManager:     jwtManager,
		CookieManager:  cookieManager,
		PermChecker:    permChecker,
		PermMiddleware: permMiddleware,
		TenantMW:       tenantMW,
		RateLimiter:    rateLimiter,
		APIRateLimiter: apiRateLimiter,
	}
}

// newNeonAuthVerifier returns the Neon Auth verifier when NEON_AUTH_BASE_URL
// is set, else nil (Neon Auth tokens are then not accepted).
func newNeonAuthVerifier(cfg *config.Config) *neonauth.Verifier {
	if cfg.Auth.NeonAuthBaseURL == "" {
		return nil
	}
	v, err := neonauth.New(cfg.Auth.NeonAuthBaseURL, cfg.Auth.NeonAuthJWKSURL)
	if err != nil {
		logging.Error(context.Background(), "[AUTH] Invalid NEON_AUTH_BASE_URL; Neon Auth tokens will be rejected", map[string]any{"error": err.Error()})
		return nil
	}
	return v
}
