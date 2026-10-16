package middleware

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/javaknight1/servicepro/backend/internal/api/apierror"
	"github.com/javaknight1/servicepro/backend/internal/models"
	"github.com/javaknight1/servicepro/backend/internal/services/permissions"
	"github.com/javaknight1/servicepro/backend/pkg/auth"
)

const (
	// Context keys for storing user information
	userIDKey      = "user_id"
	userEmailKey   = "user_email"
	permissionsKey = "user_permissions"
)

// PermissionMiddleware handles permission checking for routes
type PermissionMiddleware struct {
	tokenVersions   TokenVersionLookup
	checker         *permissions.PermissionChecker
	jwtManager      *auth.JWTManager
	cookieManager   *auth.CookieManager
	accessTokenName string
}

// TokenVersionLookup returns a user's current token version (users.token_version).
type TokenVersionLookup func(userID uuid.UUID) (int, error)

// SetTokenVersionLookup enables revocation checks: access tokens whose
// version differs from the user's current one (after logout or a password
// reset) are rejected.
func (pm *PermissionMiddleware) SetTokenVersionLookup(lookup TokenVersionLookup) {
	pm.tokenVersions = lookup
}

// NewPermissionMiddleware creates a new permission middleware
func NewPermissionMiddleware(checker *permissions.PermissionChecker, jwtManager *auth.JWTManager, cookieManager *auth.CookieManager, accessTokenName string) *PermissionMiddleware {
	return &PermissionMiddleware{
		checker:         checker,
		jwtManager:      jwtManager,
		cookieManager:   cookieManager,
		accessTokenName: accessTokenName,
	}
}

// RequireAuth is a middleware that validates JWT and sets user context
// This should be applied before any permission checks
// Token is read from httpOnly cookie first, with fallback to Authorization header for backward compatibility
func (pm *PermissionMiddleware) RequireAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		var tokenString string

		// Try to get token from httpOnly cookie first
		if pm.cookieManager != nil {
			if cookieToken, err := pm.cookieManager.GetAccessToken(c); err == nil && cookieToken != "" {
				tokenString = cookieToken
			}
		}

		// Fall back to Authorization header if no cookie found (backward compatibility)
		if tokenString == "" {
			authHeader := c.GetHeader("Authorization")
			if authHeader == "" {
				apierror.WriteResponse(c, http.StatusUnauthorized, models.ErrorResponse{
					Error:   "unauthorized",
					Message: "Authentication required",
				})
				c.Abort()
				return
			}

			// Parse Bearer token
			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) != 2 || parts[0] != "Bearer" {
				apierror.WriteResponse(c, http.StatusUnauthorized, models.ErrorResponse{
					Error:   "unauthorized",
					Message: "Invalid authorization header format",
				})
				c.Abort()
				return
			}

			tokenString = parts[1]
		}

		// Validate token
		claims, err := pm.jwtManager.ValidateToken(tokenString)
		if err != nil {
			apierror.WriteResponse(c, http.StatusUnauthorized, models.ErrorResponse{
				Error:   "unauthorized",
				Message: "Invalid or expired token",
			})
			c.Abort()
			return
		}

		// Only access tokens authenticate API calls; a (long-lived) refresh
		// token must never work as a bearer token (F-11).
		if claims.Type != auth.AccessToken {
			apierror.WriteResponse(c, http.StatusUnauthorized, models.ErrorResponse{
				Error:   "unauthorized",
				Message: "Invalid or expired token",
			})
			c.Abort()
			return
		}

		// Parse user ID from claims
		userID, err := uuid.Parse(claims.UserID)
		if err != nil {
			apierror.WriteResponse(c, http.StatusUnauthorized, models.ErrorResponse{
				Error:   "unauthorized",
				Message: "Invalid user ID in token",
			})
			c.Abort()
			return
		}

		// Reject tokens revoked by logout / password reset.
		if pm.tokenVersions != nil {
			current, err := pm.tokenVersions(userID)
			if err != nil || current != claims.TokenVersion {
				apierror.WriteResponse(c, http.StatusUnauthorized, models.ErrorResponse{
					Error:   "unauthorized",
					Message: "Session has expired, please sign in again",
				})
				c.Abort()
				return
			}
		}

		// Set user information in context
		c.Set(userIDKey, userID)
		c.Set(userEmailKey, claims.Email)

		c.Next()
	}
}

// RequirePermission creates middleware that checks for a specific permission
// Usage: router.GET("/users", middleware.RequirePermission("users.list"), handler)
func (pm *PermissionMiddleware) RequirePermission(permissionName string) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()

		// Get user ID from context (set by RequireAuth)
		userID, exists := pm.getUserID(c)
		if !exists {
			apierror.WriteResponse(c, http.StatusUnauthorized, models.ErrorResponse{
				Error:   "unauthorized",
				Message: "User not authenticated",
			})
			c.Abort()
			return
		}

		// Check permission
		ctx := context.Background()
		hasPermission, err := pm.checker.CheckPermission(ctx, userID, tenantIDFromContext(c), permissionName)
		if err != nil {
			log.Printf("[ERROR] Permission check failed for user %s: %v", userID, err)
			apierror.Internal(c, err, "Failed to verify permissions")
			c.Abort()
			return
		}

		if !hasPermission {
			// Log denied access for security audit
			log.Printf("[SECURITY] Access denied for user %s to permission %s on %s %s",
				userID, permissionName, c.Request.Method, c.Request.URL.Path)

			apierror.WriteResponse(c, http.StatusForbidden, models.ErrorResponse{
				Error:   "forbidden",
				Message: fmt.Sprintf("You do not have permission to perform this action (required: %s)", permissionName),
			})
			c.Abort()
			return
		}

		// Log performance if check took too long
		duration := time.Since(start)
		if duration > 100*time.Millisecond {
			log.Printf("[PERF] Permission check took %v for user %s", duration, userID)
		}

		c.Next()
	}
}

// RequireSuperAdmin creates middleware that checks if user is a super admin (hierarchy = 100)
// Usage: router.DELETE("/admin/system", middleware.RequireSuperAdmin(), handler)
func (pm *PermissionMiddleware) RequireSuperAdmin() gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, exists := pm.getUserID(c)
		if !exists {
			apierror.WriteResponse(c, http.StatusUnauthorized, models.ErrorResponse{
				Error:   "unauthorized",
				Message: "User not authenticated",
			})
			c.Abort()
			return
		}

		ctx := context.Background()
		isSuperAdmin, err := pm.checker.IsSuperAdmin(ctx, userID)
		if err != nil {
			log.Printf("[ERROR] Super admin check failed for user %s: %v", userID, err)
			apierror.Internal(c, err, "Failed to verify super admin status")
			c.Abort()
			return
		}

		if !isSuperAdmin {
			log.Printf("[SECURITY] Non-super-admin user %s attempted to access super admin route: %s %s",
				userID, c.Request.Method, c.Request.URL.Path)

			apierror.WriteResponse(c, http.StatusForbidden, models.ErrorResponse{
				Error:   "forbidden",
				Message: "This action requires super administrator privileges",
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// tenantIDFromContext returns the tenant resolved by RequireTenant (or
// RequireTenantFromPath). Permission checks are evaluated for that tenant
// only. Without a tenant in context only platform roles (user_roles) apply,
// so every tenant-scoped route MUST run a tenant middleware before any
// permission middleware.
func tenantIDFromContext(c *gin.Context) uuid.UUID {
	if v, ok := c.Get(TenantIDKey); ok {
		if id, ok := v.(uuid.UUID); ok {
			return id
		}
	}
	return uuid.Nil
}

// getUserID extracts the user ID from gin context
func (pm *PermissionMiddleware) getUserID(c *gin.Context) (uuid.UUID, bool) {
	value, exists := c.Get(userIDKey)
	if !exists {
		return uuid.Nil, false
	}

	userID, ok := value.(uuid.UUID)
	if !ok {
		return uuid.Nil, false
	}

	return userID, true
}

// GetUserID is a helper to get the authenticated user ID from context
// This can be used in handlers to get the current user
func GetUserID(c *gin.Context) (uuid.UUID, error) {
	value, exists := c.Get(userIDKey)
	if !exists {
		return uuid.Nil, fmt.Errorf("user ID not found in context")
	}

	userID, ok := value.(uuid.UUID)
	if !ok {
		return uuid.Nil, fmt.Errorf("invalid user ID type in context")
	}

	return userID, nil
}
