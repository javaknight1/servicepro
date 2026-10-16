package permissions

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"

	"github.com/javaknight1/servicepro/backend/internal/models"
	"github.com/javaknight1/servicepro/backend/internal/repository"
	"github.com/javaknight1/servicepro/backend/pkg/clients/logging"
)

const (
	// Cache key prefix for user permissions. Full key:
	//   permissions:user:<userID>:tenant:<tenantID>
	// Permissions are evaluated per tenant (F-2), so the cache must be too.
	permissionsCachePrefix = "permissions:user:"
	// Cache TTL for permissions (5 minutes)
	permissionsCacheTTL = 5 * time.Minute
)

// PermissionChecker provides permission checking with Redis caching.
//
// Every check is evaluated for ONE tenant: the user's platform roles
// (user_roles) plus their membership role in that tenant. Roles held in other
// tenants never apply. Pass uuid.Nil as the tenant to evaluate platform roles
// only.
type PermissionChecker struct {
	repo  *repository.PermissionRepository
	redis *redis.Client
}

// NewPermissionChecker creates a new permission checker service
func NewPermissionChecker(repo *repository.PermissionRepository, redisClient *redis.Client) *PermissionChecker {
	return &PermissionChecker{
		repo:  repo,
		redis: redisClient,
	}
}

// CheckPermission checks if a user has a specific permission in a tenant.
func (pc *PermissionChecker) CheckPermission(ctx context.Context, userID, tenantID uuid.UUID, permissionName string) (bool, error) {
	start := time.Now()
	defer func() {
		duration := time.Since(start)
		if duration > 100*time.Millisecond {
			logging.Warn(ctx, "[PERF] Permission check took too long", map[string]any{"user_id": userID, "duration": duration})
		}
	}()

	permissions, err := pc.getUserPermissions(ctx, userID, tenantID)
	if err != nil {
		return false, fmt.Errorf("failed to get user permissions: %w", err)
	}

	checker := models.NewPermissionChecker(permissions)
	return checker.Has(permissionName), nil
}

// CheckResourceAction checks if a user can perform an action on a resource in a tenant.
// Supports wildcard permissions (e.g., "users.*" or "*.*")
func (pc *PermissionChecker) CheckResourceAction(ctx context.Context, userID, tenantID uuid.UUID, resource, action string) (bool, error) {
	start := time.Now()
	defer func() {
		duration := time.Since(start)
		if duration > 100*time.Millisecond {
			logging.Warn(ctx, "[PERF] Resource action check took too long", map[string]any{"user_id": userID, "duration": duration})
		}
	}()

	permissions, err := pc.getUserPermissions(ctx, userID, tenantID)
	if err != nil {
		return false, fmt.Errorf("failed to get user permissions: %w", err)
	}

	checker := models.NewPermissionChecker(permissions)
	return checker.Can(resource, action), nil
}

// CheckAnyPermission checks if a user has any of the specified permissions in a tenant
func (pc *PermissionChecker) CheckAnyPermission(ctx context.Context, userID, tenantID uuid.UUID, permissionNames ...string) (bool, error) {
	permissions, err := pc.getUserPermissions(ctx, userID, tenantID)
	if err != nil {
		return false, fmt.Errorf("failed to get user permissions: %w", err)
	}

	checker := models.NewPermissionChecker(permissions)
	return checker.CanAny(permissionNames...), nil
}

// CheckAllPermissions checks if a user has all of the specified permissions in a tenant
func (pc *PermissionChecker) CheckAllPermissions(ctx context.Context, userID, tenantID uuid.UUID, permissionNames ...string) (bool, error) {
	permissions, err := pc.getUserPermissions(ctx, userID, tenantID)
	if err != nil {
		return false, fmt.Errorf("failed to get user permissions: %w", err)
	}

	checker := models.NewPermissionChecker(permissions)
	return checker.CanAll(permissionNames...), nil
}

// GetUserPermissions retrieves all permissions for a user in a tenant (from cache or DB)
func (pc *PermissionChecker) GetUserPermissions(ctx context.Context, userID, tenantID uuid.UUID) ([]models.Permission, error) {
	return pc.getUserPermissions(ctx, userID, tenantID)
}

// GetUserRoles retrieves the user's platform roles plus their role in the tenant
func (pc *PermissionChecker) GetUserRoles(ctx context.Context, userID, tenantID uuid.UUID) ([]models.Role, error) {
	roles, err := pc.repo.GetUserRoles(userID, tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user roles: %w", err)
	}
	return roles, nil
}

// GetUserHighestRole retrieves the user's highest role in a tenant by hierarchy level
func (pc *PermissionChecker) GetUserHighestRole(ctx context.Context, userID, tenantID uuid.UUID) (*models.Role, error) {
	role, err := pc.repo.GetUserHighestRole(userID, tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user highest role: %w", err)
	}
	return role, nil
}

// InvalidateUserPermissions invalidates the cached permissions for a user in
// EVERY tenant. Call this when user roles or memberships are modified.
func (pc *PermissionChecker) InvalidateUserPermissions(ctx context.Context, userID uuid.UUID) error {
	pattern := fmt.Sprintf("%s%s:*", permissionsCachePrefix, userID.String())
	var keys []string
	iter := pc.redis.Scan(ctx, 0, pattern, 100).Iterator()
	for iter.Next(ctx) {
		keys = append(keys, iter.Val())
	}
	if err := iter.Err(); err != nil {
		logging.Warn(ctx, "[PERMISSIONS] Failed to scan permissions cache for user", map[string]any{"user_id": userID, "error": err})
		return err
	}
	if len(keys) > 0 {
		if err := pc.redis.Del(ctx, keys...).Err(); err != nil {
			logging.Warn(ctx, "[PERMISSIONS] Failed to invalidate permissions cache for user", map[string]any{"user_id": userID, "error": err})
			return err
		}
	}
	logging.Info(ctx, "[PERMISSIONS] Invalidated permissions cache for user", map[string]any{"user_id": userID, "keys": len(keys)})
	return nil
}

// RefreshUserPermissions forces a refresh of cached permissions for a user in a tenant
func (pc *PermissionChecker) RefreshUserPermissions(ctx context.Context, userID, tenantID uuid.UUID) error {
	if err := pc.InvalidateUserPermissions(ctx, userID); err != nil {
		return err
	}
	_, err := pc.getUserPermissions(ctx, userID, tenantID)
	return err
}

// getUserPermissions retrieves permissions from cache or DB
func (pc *PermissionChecker) getUserPermissions(ctx context.Context, userID, tenantID uuid.UUID) ([]models.Permission, error) {
	cacheKey := pc.getCacheKey(userID, tenantID)

	cached, err := pc.redis.Get(ctx, cacheKey).Result()
	if err == nil {
		var permissions []models.Permission
		if err := json.Unmarshal([]byte(cached), &permissions); err != nil {
			logging.Warn(ctx, "[PERMISSIONS] Failed to unmarshal cached permissions for user", map[string]any{"user_id": userID, "error": err})
		} else {
			return permissions, nil
		}
	}

	permissions, err := pc.repo.GetUserPermissions(userID, tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch permissions from database: %w", err)
	}

	if err := pc.cachePermissions(ctx, cacheKey, permissions); err != nil {
		logging.Warn(ctx, "[PERMISSIONS] Failed to cache permissions for user", map[string]any{"user_id": userID, "error": err})
	}

	return permissions, nil
}

// cachePermissions stores permissions in Redis cache
func (pc *PermissionChecker) cachePermissions(ctx context.Context, cacheKey string, permissions []models.Permission) error {
	data, err := json.Marshal(permissions)
	if err != nil {
		return fmt.Errorf("failed to marshal permissions: %w", err)
	}

	if err := pc.redis.Set(ctx, cacheKey, data, permissionsCacheTTL).Err(); err != nil {
		return fmt.Errorf("failed to set cache: %w", err)
	}
	return nil
}

// getCacheKey generates the Redis cache key for a user's permissions in a tenant
func (pc *PermissionChecker) getCacheKey(userID, tenantID uuid.UUID) string {
	return fmt.Sprintf("%s%s:tenant:%s", permissionsCachePrefix, userID.String(), tenantID.String())
}

// IsAdmin checks if a user has admin privileges (hierarchy level >= 80) in a tenant
func (pc *PermissionChecker) IsAdmin(ctx context.Context, userID, tenantID uuid.UUID) (bool, error) {
	role, err := pc.repo.GetUserHighestRole(userID, tenantID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("failed to get user highest role: %w", err)
	}
	return role.IsAdmin(), nil
}

// IsSuperAdmin checks if a user has super admin privileges (hierarchy level = 100).
// Super admin is a PLATFORM role: only user_roles is considered, never a
// tenant membership role (super_admin must not be obtainable via a tenant).
func (pc *PermissionChecker) IsSuperAdmin(ctx context.Context, userID uuid.UUID) (bool, error) {
	role, err := pc.repo.GetUserHighestRole(userID, uuid.Nil)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("failed to get user highest role: %w", err)
	}
	return role.IsSuperAdmin(), nil
}

// GetPermissionsByResource returns all permissions for a specific resource in a tenant
func (pc *PermissionChecker) GetPermissionsByResource(ctx context.Context, userID, tenantID uuid.UUID, resource string) ([]models.Permission, error) {
	permissions, err := pc.getUserPermissions(ctx, userID, tenantID)
	if err != nil {
		return nil, err
	}

	checker := models.NewPermissionChecker(permissions)
	return checker.GetPermissionsByResource(resource), nil
}
