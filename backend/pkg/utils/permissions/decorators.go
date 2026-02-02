package permissions

import (
	"context"
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/javaknight1/servicepro/backend/internal/models"
	"github.com/javaknight1/servicepro/backend/internal/services/permissions"
	"github.com/javaknight1/servicepro/backend/pkg/clients/logging"
)

// HandlerWithPermissionCheck wraps a handler function with permission checking
// This is a decorator pattern for handlers that need permission checks
type HandlerWithPermissionCheck func(c *gin.Context, userID uuid.UUID)

// RequirePermission decorates a handler with permission checking
// Usage:
//
//	handler := func(c *gin.Context, userID uuid.UUID) {
//	    // Your handler logic here
//	}
//	router.GET("/users", permissions.RequirePermission(checker, "users.list", handler))
func RequirePermission(checker *permissions.PermissionChecker, permissionName string, handler HandlerWithPermissionCheck) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, err := getUserIDFromContext(c)
		if err != nil {
			c.JSON(401, models.ErrorResponse{
				Error:   "unauthorized",
				Message: "User not authenticated",
			})
			c.Abort()
			return
		}

		ctx := context.Background()
		hasPermission, err := checker.CheckPermission(ctx, userID, permissionName)
		if err != nil {
			logging.Error(ctx, "[PERMISSIONS] Permission check failed", map[string]any{"error": err})
			c.JSON(500, models.ErrorResponse{
				Error:   "permission_check_failed",
				Message: "Failed to verify permissions",
			})
			c.Abort()
			return
		}

		if !hasPermission {
			logging.Warn(ctx, "[SECURITY] Access denied", map[string]any{"user_id": userID, "permission": permissionName})
			c.JSON(403, models.ErrorResponse{
				Error:   "forbidden",
				Message: fmt.Sprintf("Missing required permission: %s", permissionName),
			})
			c.Abort()
			return
		}

		// Call the actual handler with userID
		handler(c, userID)
	}
}

// RequireResourceAction decorates a handler with resource.action permission checking
func RequireResourceAction(checker *permissions.PermissionChecker, resource, action string, handler HandlerWithPermissionCheck) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, err := getUserIDFromContext(c)
		if err != nil {
			c.JSON(401, models.ErrorResponse{
				Error:   "unauthorized",
				Message: "User not authenticated",
			})
			c.Abort()
			return
		}

		ctx := context.Background()
		hasPermission, err := checker.CheckResourceAction(ctx, userID, resource, action)
		if err != nil {
			logging.Error(ctx, "[PERMISSIONS] Resource action check failed", map[string]any{"error": err})
			c.JSON(500, models.ErrorResponse{
				Error:   "permission_check_failed",
				Message: "Failed to verify permissions",
			})
			c.Abort()
			return
		}

		if !hasPermission {
			logging.Warn(ctx, "[SECURITY] Access denied", map[string]any{"user_id": userID, "resource": resource, "action": action})
			c.JSON(403, models.ErrorResponse{
				Error:   "forbidden",
				Message: fmt.Sprintf("You do not have permission to %s %s", action, resource),
			})
			c.Abort()
			return
		}

		handler(c, userID)
	}
}

// RouteProtection provides helper methods for protecting routes
type RouteProtection struct {
	checker *permissions.PermissionChecker
}

// NewRouteProtection creates a new route protection helper
func NewRouteProtection(checker *permissions.PermissionChecker) *RouteProtection {
	return &RouteProtection{
		checker: checker,
	}
}

// Protect creates a protected route with permission checking
// Usage:
//
//	protection := permissions.NewRouteProtection(checker)
//	router.GET("/users", protection.Protect("users.list", func(c *gin.Context, userID uuid.UUID) {
//	    // Handler logic
//	}))
func (rp *RouteProtection) Protect(permissionName string, handler HandlerWithPermissionCheck) gin.HandlerFunc {
	return RequirePermission(rp.checker, permissionName, handler)
}

// ProtectResourceAction creates a protected route with resource.action permission checking
func (rp *RouteProtection) ProtectResourceAction(resource, action string, handler HandlerWithPermissionCheck) gin.HandlerFunc {
	return RequireResourceAction(rp.checker, resource, action, handler)
}

// ProtectAdmin creates an admin-only route
func (rp *RouteProtection) ProtectAdmin(handler HandlerWithPermissionCheck) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, err := getUserIDFromContext(c)
		if err != nil {
			c.JSON(401, models.ErrorResponse{
				Error:   "unauthorized",
				Message: "User not authenticated",
			})
			c.Abort()
			return
		}

		ctx := context.Background()
		isAdmin, err := rp.checker.IsAdmin(ctx, userID)
		if err != nil {
			logging.Error(ctx, "[PERMISSIONS] Admin check failed", map[string]any{"error": err})
			c.JSON(500, models.ErrorResponse{
				Error:   "permission_check_failed",
				Message: "Failed to verify admin status",
			})
			c.Abort()
			return
		}

		if !isAdmin {
			logging.Warn(ctx, "[SECURITY] Non-admin user attempted admin access", map[string]any{"user_id": userID})
			c.JSON(403, models.ErrorResponse{
				Error:   "forbidden",
				Message: "Administrator privileges required",
			})
			c.Abort()
			return
		}

		handler(c, userID)
	}
}

// ProtectSuperAdmin creates a super-admin-only route
func (rp *RouteProtection) ProtectSuperAdmin(handler HandlerWithPermissionCheck) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, err := getUserIDFromContext(c)
		if err != nil {
			c.JSON(401, models.ErrorResponse{
				Error:   "unauthorized",
				Message: "User not authenticated",
			})
			c.Abort()
			return
		}

		ctx := context.Background()
		isSuperAdmin, err := rp.checker.IsSuperAdmin(ctx, userID)
		if err != nil {
			logging.Error(ctx, "[PERMISSIONS] Super admin check failed", map[string]any{"error": err})
			c.JSON(500, models.ErrorResponse{
				Error:   "permission_check_failed",
				Message: "Failed to verify super admin status",
			})
			c.Abort()
			return
		}

		if !isSuperAdmin {
			logging.Warn(ctx, "[SECURITY] Non-super-admin user attempted super admin access", map[string]any{"user_id": userID})
			c.JSON(403, models.ErrorResponse{
				Error:   "forbidden",
				Message: "Super administrator privileges required",
			})
			c.Abort()
			return
		}

		handler(c, userID)
	}
}

// PermissionValidator provides methods for validating permissions in business logic
type PermissionValidator struct {
	checker *permissions.PermissionChecker
}

// NewPermissionValidator creates a new permission validator
func NewPermissionValidator(checker *permissions.PermissionChecker) *PermissionValidator {
	return &PermissionValidator{
		checker: checker,
	}
}

// ValidatePermission validates a permission and returns an error if not allowed
// This is useful for permission checks within service layer
func (pv *PermissionValidator) ValidatePermission(ctx context.Context, userID uuid.UUID, permissionName string) error {
	hasPermission, err := pv.checker.CheckPermission(ctx, userID, permissionName)
	if err != nil {
		return fmt.Errorf("permission check failed: %w", err)
	}

	if !hasPermission {
		return fmt.Errorf("user %s does not have permission: %s", userID, permissionName)
	}

	return nil
}

// ValidateResourceAction validates a resource.action permission
func (pv *PermissionValidator) ValidateResourceAction(ctx context.Context, userID uuid.UUID, resource, action string) error {
	hasPermission, err := pv.checker.CheckResourceAction(ctx, userID, resource, action)
	if err != nil {
		return fmt.Errorf("resource action check failed: %w", err)
	}

	if !hasPermission {
		return fmt.Errorf("user %s cannot %s %s", userID, action, resource)
	}

	return nil
}

// ValidateAnyPermission validates that user has at least one of the permissions
func (pv *PermissionValidator) ValidateAnyPermission(ctx context.Context, userID uuid.UUID, permissionNames ...string) error {
	hasPermission, err := pv.checker.CheckAnyPermission(ctx, userID, permissionNames...)
	if err != nil {
		return fmt.Errorf("any permission check failed: %w", err)
	}

	if !hasPermission {
		return fmt.Errorf("user %s does not have any of the required permissions: %v", userID, permissionNames)
	}

	return nil
}

// ValidateAllPermissions validates that user has all of the permissions
func (pv *PermissionValidator) ValidateAllPermissions(ctx context.Context, userID uuid.UUID, permissionNames ...string) error {
	hasPermission, err := pv.checker.CheckAllPermissions(ctx, userID, permissionNames...)
	if err != nil {
		return fmt.Errorf("all permissions check failed: %w", err)
	}

	if !hasPermission {
		return fmt.Errorf("user %s does not have all required permissions: %v", userID, permissionNames)
	}

	return nil
}

// ValidateAdmin validates that user is an admin
func (pv *PermissionValidator) ValidateAdmin(ctx context.Context, userID uuid.UUID) error {
	isAdmin, err := pv.checker.IsAdmin(ctx, userID)
	if err != nil {
		return fmt.Errorf("admin check failed: %w", err)
	}

	if !isAdmin {
		return fmt.Errorf("user %s is not an administrator", userID)
	}

	return nil
}

// ValidateSuperAdmin validates that user is a super admin
func (pv *PermissionValidator) ValidateSuperAdmin(ctx context.Context, userID uuid.UUID) error {
	isSuperAdmin, err := pv.checker.IsSuperAdmin(ctx, userID)
	if err != nil {
		return fmt.Errorf("super admin check failed: %w", err)
	}

	if !isSuperAdmin {
		return fmt.Errorf("user %s is not a super administrator", userID)
	}

	return nil
}

// getUserIDFromContext extracts user ID from gin context
func getUserIDFromContext(c *gin.Context) (uuid.UUID, error) {
	value, exists := c.Get("user_id")
	if !exists {
		return uuid.Nil, fmt.Errorf("user_id not found in context")
	}

	userID, ok := value.(uuid.UUID)
	if !ok {
		return uuid.Nil, fmt.Errorf("invalid user_id type in context")
	}

	return userID, nil
}

// CanUserPerform is a simple helper to check if a user can perform an action
// This is useful for conditional UI rendering or business logic
func CanUserPerform(ctx context.Context, checker *permissions.PermissionChecker, userID uuid.UUID, permissionName string) bool {
	hasPermission, err := checker.CheckPermission(ctx, userID, permissionName)
	if err != nil {
		logging.Warn(ctx, "[PERMISSIONS] Permission check error", map[string]any{"user_id": userID, "error": err})
		return false
	}
	return hasPermission
}

// CanUserPerformAction is a helper to check if user can perform resource.action
func CanUserPerformAction(ctx context.Context, checker *permissions.PermissionChecker, userID uuid.UUID, resource, action string) bool {
	hasPermission, err := checker.CheckResourceAction(ctx, userID, resource, action)
	if err != nil {
		logging.Warn(ctx, "[PERMISSIONS] Resource action check error", map[string]any{"user_id": userID, "error": err})
		return false
	}
	return hasPermission
}
