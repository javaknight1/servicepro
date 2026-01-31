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
	checker         *permissions.PermissionChecker
	jwtManager      *auth.JWTManager
	cookieManager   *auth.CookieManager
	accessTokenName string
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
				c.JSON(http.StatusUnauthorized, models.ErrorResponse{
					Error:   "unauthorized",
					Message: "Authentication required",
				})
				c.Abort()
				return
			}

			// Parse Bearer token
			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) != 2 || parts[0] != "Bearer" {
				c.JSON(http.StatusUnauthorized, models.ErrorResponse{
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
			c.JSON(http.StatusUnauthorized, models.ErrorResponse{
				Error:   "unauthorized",
				Message: fmt.Sprintf("Invalid or expired token: %v", err),
			})
			c.Abort()
			return
		}

		// Parse user ID from claims
		userID, err := uuid.Parse(claims.UserID)
		if err != nil {
			c.JSON(http.StatusUnauthorized, models.ErrorResponse{
				Error:   "unauthorized",
				Message: "Invalid user ID in token",
			})
			c.Abort()
			return
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
			c.JSON(http.StatusUnauthorized, models.ErrorResponse{
				Error:   "unauthorized",
				Message: "User not authenticated",
			})
			c.Abort()
			return
		}

		// Check permission
		ctx := context.Background()
		hasPermission, err := pm.checker.CheckPermission(ctx, userID, permissionName)
		if err != nil {
			log.Printf("[ERROR] Permission check failed for user %s: %v", userID, err)
			c.JSON(http.StatusInternalServerError, models.ErrorResponse{
				Error:   "permission_check_failed",
				Message: "Failed to verify permissions",
			})
			c.Abort()
			return
		}

		if !hasPermission {
			// Log denied access for security audit
			log.Printf("[SECURITY] Access denied for user %s to permission %s on %s %s",
				userID, permissionName, c.Request.Method, c.Request.URL.Path)

			c.JSON(http.StatusForbidden, models.ErrorResponse{
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

// RequireResourceAction creates middleware that checks for resource.action permission
// Usage: router.POST("/users", middleware.RequireResourceAction("users", "create"), handler)
func (pm *PermissionMiddleware) RequireResourceAction(resource, action string) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()

		// Get user ID from context
		userID, exists := pm.getUserID(c)
		if !exists {
			c.JSON(http.StatusUnauthorized, models.ErrorResponse{
				Error:   "unauthorized",
				Message: "User not authenticated",
			})
			c.Abort()
			return
		}

		// Check resource action permission
		ctx := context.Background()
		hasPermission, err := pm.checker.CheckResourceAction(ctx, userID, resource, action)
		if err != nil {
			log.Printf("[ERROR] Resource action check failed for user %s: %v", userID, err)
			c.JSON(http.StatusInternalServerError, models.ErrorResponse{
				Error:   "permission_check_failed",
				Message: "Failed to verify permissions",
			})
			c.Abort()
			return
		}

		if !hasPermission {
			// Log denied access for security audit
			log.Printf("[SECURITY] Access denied for user %s to %s.%s on %s %s",
				userID, resource, action, c.Request.Method, c.Request.URL.Path)

			c.JSON(http.StatusForbidden, models.ErrorResponse{
				Error:   "forbidden",
				Message: fmt.Sprintf("You do not have permission to %s %s", action, resource),
			})
			c.Abort()
			return
		}

		// Log performance if check took too long
		duration := time.Since(start)
		if duration > 100*time.Millisecond {
			log.Printf("[PERF] Resource action check took %v for user %s", duration, userID)
		}

		c.Next()
	}
}

// RequireAnyPermission creates middleware that checks if user has ANY of the specified permissions
// Usage: router.GET("/dashboard", middleware.RequireAnyPermission("users.list", "admin.access"), handler)
func (pm *PermissionMiddleware) RequireAnyPermission(permissionNames ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, exists := pm.getUserID(c)
		if !exists {
			c.JSON(http.StatusUnauthorized, models.ErrorResponse{
				Error:   "unauthorized",
				Message: "User not authenticated",
			})
			c.Abort()
			return
		}

		ctx := context.Background()
		hasPermission, err := pm.checker.CheckAnyPermission(ctx, userID, permissionNames...)
		if err != nil {
			log.Printf("[ERROR] Any permission check failed for user %s: %v", userID, err)
			c.JSON(http.StatusInternalServerError, models.ErrorResponse{
				Error:   "permission_check_failed",
				Message: "Failed to verify permissions",
			})
			c.Abort()
			return
		}

		if !hasPermission {
			log.Printf("[SECURITY] Access denied for user %s (required any of: %v) on %s %s",
				userID, permissionNames, c.Request.Method, c.Request.URL.Path)

			c.JSON(http.StatusForbidden, models.ErrorResponse{
				Error:   "forbidden",
				Message: "You do not have the required permissions",
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// RequireAllPermissions creates middleware that checks if user has ALL of the specified permissions
// Usage: router.POST("/admin/users", middleware.RequireAllPermissions("users.create", "admin.access"), handler)
func (pm *PermissionMiddleware) RequireAllPermissions(permissionNames ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, exists := pm.getUserID(c)
		if !exists {
			c.JSON(http.StatusUnauthorized, models.ErrorResponse{
				Error:   "unauthorized",
				Message: "User not authenticated",
			})
			c.Abort()
			return
		}

		ctx := context.Background()
		hasPermission, err := pm.checker.CheckAllPermissions(ctx, userID, permissionNames...)
		if err != nil {
			log.Printf("[ERROR] All permissions check failed for user %s: %v", userID, err)
			c.JSON(http.StatusInternalServerError, models.ErrorResponse{
				Error:   "permission_check_failed",
				Message: "Failed to verify permissions",
			})
			c.Abort()
			return
		}

		if !hasPermission {
			log.Printf("[SECURITY] Access denied for user %s (required all of: %v) on %s %s",
				userID, permissionNames, c.Request.Method, c.Request.URL.Path)

			c.JSON(http.StatusForbidden, models.ErrorResponse{
				Error:   "forbidden",
				Message: "You do not have all the required permissions",
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// RequireAdmin creates middleware that checks if user is an admin (hierarchy >= 80)
// Usage: router.GET("/admin/dashboard", middleware.RequireAdmin(), handler)
func (pm *PermissionMiddleware) RequireAdmin() gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, exists := pm.getUserID(c)
		if !exists {
			c.JSON(http.StatusUnauthorized, models.ErrorResponse{
				Error:   "unauthorized",
				Message: "User not authenticated",
			})
			c.Abort()
			return
		}

		ctx := context.Background()
		isAdmin, err := pm.checker.IsAdmin(ctx, userID)
		if err != nil {
			log.Printf("[ERROR] Admin check failed for user %s: %v", userID, err)
			c.JSON(http.StatusInternalServerError, models.ErrorResponse{
				Error:   "permission_check_failed",
				Message: "Failed to verify admin status",
			})
			c.Abort()
			return
		}

		if !isAdmin {
			log.Printf("[SECURITY] Non-admin user %s attempted to access admin route: %s %s",
				userID, c.Request.Method, c.Request.URL.Path)

			c.JSON(http.StatusForbidden, models.ErrorResponse{
				Error:   "forbidden",
				Message: "This action requires administrator privileges",
			})
			c.Abort()
			return
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
			c.JSON(http.StatusUnauthorized, models.ErrorResponse{
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
			c.JSON(http.StatusInternalServerError, models.ErrorResponse{
				Error:   "permission_check_failed",
				Message: "Failed to verify super admin status",
			})
			c.Abort()
			return
		}

		if !isSuperAdmin {
			log.Printf("[SECURITY] Non-super-admin user %s attempted to access super admin route: %s %s",
				userID, c.Request.Method, c.Request.URL.Path)

			c.JSON(http.StatusForbidden, models.ErrorResponse{
				Error:   "forbidden",
				Message: "This action requires super administrator privileges",
			})
			c.Abort()
			return
		}

		c.Next()
	}
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

// GetUserEmail is a helper to get the authenticated user email from context
func GetUserEmail(c *gin.Context) (string, error) {
	value, exists := c.Get(userEmailKey)
	if !exists {
		return "", fmt.Errorf("user email not found in context")
	}

	email, ok := value.(string)
	if !ok {
		return "", fmt.Errorf("invalid user email type in context")
	}

	return email, nil
}
