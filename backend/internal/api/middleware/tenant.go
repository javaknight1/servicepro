package middleware

import (
	"context"
	"fmt"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/javaknight1/servicepro/backend/internal/models"
	"github.com/javaknight1/servicepro/backend/internal/tenancy"
)

const (
	// TenantIDKey is the context key for storing the current tenant ID
	TenantIDKey = "tenant_id"
	// TenantKey is the context key for storing the current tenant
	TenantKey = "tenant"
	// TenantHeader is the HTTP header for specifying tenant ID
	TenantHeader = "X-Tenant-ID"
)

const (
	// TenantRoleKey is the context key for the user's role name within the current tenant
	TenantRoleKey = "tenant_role"
	// TenantRoleIDKey is the context key for the user's role ID within the current tenant
	TenantRoleIDKey = "tenant_role_id"
	// TenantRoleLevelKey is the context key for the user's role hierarchy level within the current tenant
	TenantRoleLevelKey = "tenant_role_level"
)

// TenantRepository interface for tenant data access
type TenantRepository interface {
	GetByID(ctx context.Context, id uuid.UUID) (*models.Tenant, error)
	GetBySlug(ctx context.Context, slug string) (*models.Tenant, error)
	UserBelongsToTenant(ctx context.Context, userID, tenantID uuid.UUID) (bool, error)
	GetUserTenants(ctx context.Context, userID uuid.UUID) ([]models.Tenant, error)
	GetUserTenantMembership(ctx context.Context, userID, tenantID uuid.UUID) (*models.TenantUser, error)
}

// TenantMiddleware handles tenant context for multi-tenancy
type TenantMiddleware struct {
	tenantRepo TenantRepository
}

// NewTenantMiddleware creates a new tenant middleware
func NewTenantMiddleware(tenantRepo TenantRepository) *TenantMiddleware {
	return &TenantMiddleware{
		tenantRepo: tenantRepo,
	}
}

// RequireTenant resolves the caller's tenant and verifies membership.
//
// The tenant is taken from the X-Tenant-ID header; when absent, the user's
// first active tenant is used. Either way the user MUST be an active member
// of the tenant and the tenant must be active — the header is only a
// selector, never a grant.
func (tm *TenantMiddleware) RequireTenant() gin.HandlerFunc {
	return func(c *gin.Context) {
		userUUID, ok := authenticatedUserID(c)
		if !ok {
			return
		}

		var tenantID uuid.UUID
		if tenantIDStr := c.GetHeader(TenantHeader); tenantIDStr != "" {
			parsed, err := uuid.Parse(tenantIDStr)
			if err != nil {
				c.AbortWithStatusJSON(http.StatusBadRequest, models.ErrorResponse{
					Error:   "invalid_tenant_id",
					Message: "Invalid tenant ID format",
				})
				return
			}
			tenantID = parsed
		} else {
			tenants, err := tm.tenantRepo.GetUserTenants(c.Request.Context(), userUUID)
			if err != nil {
				log.Printf("[TENANT] Failed to get user tenants: %v", err)
				c.AbortWithStatusJSON(http.StatusInternalServerError, models.ErrorResponse{
					Error:   "tenant_lookup_failed",
					Message: "Failed to determine tenant context",
				})
				return
			}
			if len(tenants) == 0 {
				c.AbortWithStatusJSON(http.StatusForbidden, models.ErrorResponse{
					Error:   "no_tenant_access",
					Message: "User does not belong to any organization",
				})
				return
			}
			tenantID = tenants[0].ID
		}

		if !tm.bindTenant(c, userUUID, tenantID) {
			return
		}
		c.Next()
	}
}

// RequireTenantFromPath binds the tenant named by a path parameter (e.g. the
// :id in /tenants/:id/members) after verifying that the caller is an active
// member of it. The path is authoritative for these routes; an X-Tenant-ID
// header is ignored. Non-members get 404 so tenant IDs can't be probed.
func (tm *TenantMiddleware) RequireTenantFromPath(param string) gin.HandlerFunc {
	return func(c *gin.Context) {
		userUUID, ok := authenticatedUserID(c)
		if !ok {
			return
		}

		tenantID, err := uuid.Parse(c.Param(param))
		if err != nil {
			c.AbortWithStatusJSON(http.StatusBadRequest, models.ErrorResponse{
				Error:   "invalid_tenant_id",
				Message: "Invalid tenant ID format",
			})
			return
		}

		if !tm.bindTenant(c, userUUID, tenantID) {
			return
		}
		c.Next()
	}
}

// bindTenant verifies membership and stores the tenant in both the gin
// context and the request context (for repositories). Returns false (and
// aborts) on failure.
func (tm *TenantMiddleware) bindTenant(c *gin.Context, userID, tenantID uuid.UUID) bool {
	ctx := c.Request.Context()

	membership, err := tm.tenantRepo.GetUserTenantMembership(ctx, userID, tenantID)
	if err != nil || membership == nil {
		if membership == nil {
			log.Printf("[SECURITY] User %s attempted to access tenant %s without membership", userID, tenantID)
			c.AbortWithStatusJSON(http.StatusForbidden, models.ErrorResponse{
				Error:   "tenant_access_denied",
				Message: "You do not have access to this organization",
			})
			return false
		}
		log.Printf("[TENANT] Failed to verify tenant membership: %v", err)
		c.AbortWithStatusJSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "tenant_verification_failed",
			Message: "Failed to verify tenant access",
		})
		return false
	}

	tenant, err := tm.tenantRepo.GetByID(ctx, tenantID)
	if err != nil || tenant == nil || !tenant.IsActive {
		if err != nil {
			log.Printf("[TENANT] Failed to load tenant %s: %v", tenantID, err)
		}
		c.AbortWithStatusJSON(http.StatusForbidden, models.ErrorResponse{
			Error:   "tenant_inactive",
			Message: "This organization is not active",
		})
		return false
	}

	c.Set(TenantIDKey, tenantID)
	c.Set(TenantKey, tenant)
	c.Set(TenantRoleIDKey, membership.RoleID)
	if membership.Role != nil {
		c.Set(TenantRoleKey, membership.Role.Name)
		c.Set(TenantRoleLevelKey, membership.Role.HierarchyLevel)
	} else {
		c.Set(TenantRoleLevelKey, 0)
	}
	c.Request = c.Request.WithContext(tenancy.WithTenant(ctx, tenantID))
	return true
}

// authenticatedUserID returns the user set by RequireAuth or aborts with 401.
func authenticatedUserID(c *gin.Context) (uuid.UUID, bool) {
	userID, exists := c.Get(userIDKey)
	if !exists {
		c.AbortWithStatusJSON(http.StatusUnauthorized, models.ErrorResponse{
			Error:   "unauthorized",
			Message: "User not authenticated",
		})
		return uuid.Nil, false
	}
	userUUID, ok := userID.(uuid.UUID)
	if !ok {
		c.AbortWithStatusJSON(http.StatusUnauthorized, models.ErrorResponse{
			Error:   "unauthorized",
			Message: "Invalid user ID format",
		})
		return uuid.Nil, false
	}
	return userUUID, true
}

// GetTenantID is a helper to get the current tenant ID from context
func GetTenantID(c *gin.Context) (uuid.UUID, error) {
	value, exists := c.Get(TenantIDKey)
	if !exists {
		return uuid.Nil, fmt.Errorf("tenant ID not found in context")
	}

	tenantID, ok := value.(uuid.UUID)
	if !ok {
		return uuid.Nil, fmt.Errorf("invalid tenant ID type in context")
	}

	return tenantID, nil
}

// GetTenantRoleLevel returns the caller's role hierarchy level in the current
// tenant (0 when unknown).
func GetTenantRoleLevel(c *gin.Context) int {
	if v, ok := c.Get(TenantRoleLevelKey); ok {
		if level, ok := v.(int); ok {
			return level
		}
	}
	return 0
}

// GetTenant is a helper to get the current tenant from context
func GetTenant(c *gin.Context) (*models.Tenant, error) {
	value, exists := c.Get(TenantKey)
	if !exists {
		return nil, fmt.Errorf("tenant not found in context")
	}

	tenant, ok := value.(*models.Tenant)
	if !ok {
		return nil, fmt.Errorf("invalid tenant type in context")
	}

	return tenant, nil
}

// MustGetTenantID returns the tenant ID or panics if not found
// Use this only when RequireTenant middleware is guaranteed to have run
func MustGetTenantID(c *gin.Context) uuid.UUID {
	tenantID, err := GetTenantID(c)
	if err != nil {
		panic(err)
	}
	return tenantID
}
