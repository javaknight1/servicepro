package permissions

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"

	"github.com/javaknight1/servicepro/backend/internal/models"
	"github.com/javaknight1/servicepro/backend/internal/repository"
	"github.com/javaknight1/servicepro/backend/pkg/clients/logging"
)

const (
	// Cache key prefix for user permissions
	permissionsCachePrefix = "permissions:user:"
	// Cache TTL for permissions (5 minutes)
	permissionsCacheTTL = 5 * time.Minute
	// Maximum cache refresh attempts
	maxCacheRefreshAttempts = 3
)

// PermissionChecker provides permission checking with Redis caching
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

// CheckPermission checks if a user has a specific permission
// Returns true if the user has the permission, false otherwise
// Uses Redis caching for performance
func (pc *PermissionChecker) CheckPermission(ctx context.Context, userID uuid.UUID, permissionName string) (bool, error) {
	start := time.Now()
	defer func() {
		duration := time.Since(start)
		if duration > 100*time.Millisecond {
			logging.Warn(ctx, "[PERF] Permission check took too long", map[string]any{"user_id": userID, "duration": duration})
		}
	}()

	// Get user permissions from cache or DB
	permissions, err := pc.getUserPermissions(ctx, userID)
	if err != nil {
		return false, fmt.Errorf("failed to get user permissions: %w", err)
	}

	// Check if user has the permission
	checker := models.NewPermissionChecker(permissions)
	return checker.Has(permissionName), nil
}

// CheckResourceAction checks if a user can perform an action on a resource
// Supports wildcard permissions (e.g., "users.*" or "*.*")
func (pc *PermissionChecker) CheckResourceAction(ctx context.Context, userID uuid.UUID, resource, action string) (bool, error) {
	start := time.Now()
	defer func() {
		duration := time.Since(start)
		if duration > 100*time.Millisecond {
			logging.Warn(ctx, "[PERF] Resource action check took too long", map[string]any{"user_id": userID, "duration": duration})
		}
	}()

	// Get user permissions from cache or DB
	permissions, err := pc.getUserPermissions(ctx, userID)
	if err != nil {
		return false, fmt.Errorf("failed to get user permissions: %w", err)
	}

	// Check if user can perform action on resource
	checker := models.NewPermissionChecker(permissions)
	return checker.Can(resource, action), nil
}

// CheckAnyPermission checks if a user has any of the specified permissions
func (pc *PermissionChecker) CheckAnyPermission(ctx context.Context, userID uuid.UUID, permissionNames ...string) (bool, error) {
	permissions, err := pc.getUserPermissions(ctx, userID)
	if err != nil {
		return false, fmt.Errorf("failed to get user permissions: %w", err)
	}

	checker := models.NewPermissionChecker(permissions)
	return checker.CanAny(permissionNames...), nil
}

// CheckAllPermissions checks if a user has all of the specified permissions
func (pc *PermissionChecker) CheckAllPermissions(ctx context.Context, userID uuid.UUID, permissionNames ...string) (bool, error) {
	permissions, err := pc.getUserPermissions(ctx, userID)
	if err != nil {
		return false, fmt.Errorf("failed to get user permissions: %w", err)
	}

	checker := models.NewPermissionChecker(permissions)
	return checker.CanAll(permissionNames...), nil
}

// GetUserPermissions retrieves all permissions for a user (from cache or DB)
func (pc *PermissionChecker) GetUserPermissions(ctx context.Context, userID uuid.UUID) ([]models.Permission, error) {
	return pc.getUserPermissions(ctx, userID)
}

// GetUserRoles retrieves all active roles for a user
func (pc *PermissionChecker) GetUserRoles(ctx context.Context, userID uuid.UUID) ([]models.Role, error) {
	roles, err := pc.repo.GetUserRoles(userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user roles: %w", err)
	}
	return roles, nil
}

// GetUserHighestRole retrieves the user's highest role by hierarchy level
func (pc *PermissionChecker) GetUserHighestRole(ctx context.Context, userID uuid.UUID) (*models.Role, error) {
	role, err := pc.repo.GetUserHighestRole(userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user highest role: %w", err)
	}
	return role, nil
}

// InvalidateUserPermissions invalidates the cached permissions for a user
// Call this when user roles or permissions are modified
func (pc *PermissionChecker) InvalidateUserPermissions(ctx context.Context, userID uuid.UUID) error {
	cacheKey := pc.getCacheKey(userID)
	err := pc.redis.Del(ctx, cacheKey).Err()
	if err != nil {
		logging.Warn(ctx, "[PERMISSIONS] Failed to invalidate permissions cache for user", map[string]any{"user_id": userID, "error": err})
		return err
	}
	logging.Info(ctx, "[PERMISSIONS] Invalidated permissions cache for user", map[string]any{"user_id": userID})
	return nil
}

// RefreshUserPermissions forces a refresh of cached permissions for a user
func (pc *PermissionChecker) RefreshUserPermissions(ctx context.Context, userID uuid.UUID) error {
	// First invalidate existing cache
	if err := pc.InvalidateUserPermissions(ctx, userID); err != nil {
		return err
	}

	// Then fetch fresh permissions (which will cache them)
	_, err := pc.getUserPermissions(ctx, userID)
	return err
}

// getUserPermissions retrieves permissions from cache or DB
// This is the core caching logic
func (pc *PermissionChecker) getUserPermissions(ctx context.Context, userID uuid.UUID) ([]models.Permission, error) {
	cacheKey := pc.getCacheKey(userID)

	// Try to get from cache first
	cached, err := pc.redis.Get(ctx, cacheKey).Result()
	if err == nil {
		// Cache hit - deserialize and return
		var permissions []models.Permission
		if err := json.Unmarshal([]byte(cached), &permissions); err != nil {
			logging.Warn(ctx, "[PERMISSIONS] Failed to unmarshal cached permissions for user", map[string]any{"user_id": userID, "error": err})
			// Fall through to fetch from DB
		} else {
			logging.Info(ctx, "[PERMISSIONS] Cache hit for user permissions", map[string]any{"user_id": userID})
			return permissions, nil
		}
	}

	// Cache miss or error - fetch from database
	logging.Info(ctx, "[PERMISSIONS] Cache miss for user permissions", map[string]any{"user_id": userID})
	permissions, err := pc.repo.GetUserPermissions(userID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch permissions from database: %w", err)
	}

	// Cache the permissions
	if err := pc.cachePermissions(ctx, userID, permissions); err != nil {
		// Log error but don't fail the request
		logging.Warn(ctx, "[PERMISSIONS] Failed to cache permissions for user", map[string]any{"user_id": userID, "error": err})
	}

	return permissions, nil
}

// cachePermissions stores permissions in Redis cache
func (pc *PermissionChecker) cachePermissions(ctx context.Context, userID uuid.UUID, permissions []models.Permission) error {
	cacheKey := pc.getCacheKey(userID)

	// Serialize permissions to JSON
	data, err := json.Marshal(permissions)
	if err != nil {
		return fmt.Errorf("failed to marshal permissions: %w", err)
	}

	// Store in Redis with TTL
	err = pc.redis.Set(ctx, cacheKey, data, permissionsCacheTTL).Err()
	if err != nil {
		return fmt.Errorf("failed to set cache: %w", err)
	}

	logging.Info(ctx, "[PERMISSIONS] Cached permissions for user", map[string]any{"user_id": userID, "ttl": permissionsCacheTTL})
	return nil
}

// getCacheKey generates the Redis cache key for a user's permissions
func (pc *PermissionChecker) getCacheKey(userID uuid.UUID) string {
	return fmt.Sprintf("%s%s", permissionsCachePrefix, userID.String())
}

// IsAdmin checks if a user has admin privileges (hierarchy level >= 80)
func (pc *PermissionChecker) IsAdmin(ctx context.Context, userID uuid.UUID) (bool, error) {
	role, err := pc.repo.GetUserHighestRole(userID)
	if err != nil {
		return false, fmt.Errorf("failed to get user highest role: %w", err)
	}
	return role.IsAdmin(), nil
}

// IsSuperAdmin checks if a user has super admin privileges (hierarchy level = 100)
func (pc *PermissionChecker) IsSuperAdmin(ctx context.Context, userID uuid.UUID) (bool, error) {
	role, err := pc.repo.GetUserHighestRole(userID)
	if err != nil {
		return false, fmt.Errorf("failed to get user highest role: %w", err)
	}
	return role.IsSuperAdmin(), nil
}

// GetPermissionsByResource returns all permissions for a specific resource
func (pc *PermissionChecker) GetPermissionsByResource(ctx context.Context, userID uuid.UUID, resource string) ([]models.Permission, error) {
	permissions, err := pc.getUserPermissions(ctx, userID)
	if err != nil {
		return nil, err
	}

	checker := models.NewPermissionChecker(permissions)
	return checker.GetPermissionsByResource(resource), nil
}
