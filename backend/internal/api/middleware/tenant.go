package middleware

import (
	"context"
	"fmt"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/javaknight1/servicepro/backend/internal/models"
)

const (
	// TenantIDKey is the context key for storing the current tenant ID
	TenantIDKey = "tenant_id"
	// TenantKey is the context key for storing the current tenant
	TenantKey = "tenant"
	// TenantHeader is the HTTP header for specifying tenant ID
	TenantHeader = "X-Tenant-ID"
)

// TenantRoleKey is the context key for the user's role within the current tenant
const TenantRoleKey = "tenant_role"

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

// RequireTenant is middleware that validates and sets the tenant context
// It expects either X-Tenant-ID header or tenant_id from JWT claims
func (tm *TenantMiddleware) RequireTenant() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get user ID from context (set by RequireAuth)
		userID, exists := c.Get(userIDKey)
		if !exists {
			c.JSON(http.StatusUnauthorized, models.ErrorResponse{
				Error:   "unauthorized",
				Message: "User not authenticated",
			})
			c.Abort()
			return
		}

		userUUID, ok := userID.(uuid.UUID)
		if !ok {
			c.JSON(http.StatusUnauthorized, models.ErrorResponse{
				Error:   "unauthorized",
				Message: "Invalid user ID format",
			})
			c.Abort()
			return
		}

		// Try to get tenant ID from header first
		tenantIDStr := c.GetHeader(TenantHeader)
		var tenantID uuid.UUID
		var err error

		if tenantIDStr != "" {
			tenantID, err = uuid.Parse(tenantIDStr)
			if err != nil {
				c.JSON(http.StatusBadRequest, models.ErrorResponse{
					Error:   "invalid_tenant_id",
					Message: "Invalid tenant ID format",
				})
				c.Abort()
				return
			}
		} else {
			// Try to get from JWT claims (set during token generation with tenant context)
			tenantIDFromClaims, exists := c.Get("jwt_tenant_id")
			if exists {
				tenantID, ok = tenantIDFromClaims.(uuid.UUID)
				if !ok {
					// Try string format
					if tidStr, ok := tenantIDFromClaims.(string); ok {
						tenantID, err = uuid.Parse(tidStr)
						if err != nil {
							c.JSON(http.StatusBadRequest, models.ErrorResponse{
								Error:   "invalid_tenant_id",
								Message: "Invalid tenant ID in token",
							})
							c.Abort()
							return
						}
					}
				}
			}
		}

		// If no tenant ID provided, try to get user's default tenant
		if tenantID == uuid.Nil {
			tenants, err := tm.tenantRepo.GetUserTenants(c.Request.Context(), userUUID)
			if err != nil {
				log.Printf("[ERROR] Failed to get user tenants: %v", err)
				c.JSON(http.StatusInternalServerError, models.ErrorResponse{
					Error:   "tenant_lookup_failed",
					Message: "Failed to determine tenant context",
				})
				c.Abort()
				return
			}

			if len(tenants) == 0 {
				c.JSON(http.StatusForbidden, models.ErrorResponse{
					Error:   "no_tenant_access",
					Message: "User does not belong to any organization",
				})
				c.Abort()
				return
			}

			// Use first tenant as default (could be enhanced to use user preferences)
			tenantID = tenants[0].ID
		}

		// Verify user has access to this tenant and get their role
		membership, err := tm.tenantRepo.GetUserTenantMembership(c.Request.Context(), userUUID, tenantID)
		if err != nil {
			if membership == nil {
				log.Printf("[SECURITY] User %s attempted to access tenant %s without membership",
					userUUID, tenantID)
				c.JSON(http.StatusForbidden, models.ErrorResponse{
					Error:   "tenant_access_denied",
					Message: "You do not have access to this organization",
				})
				c.Abort()
				return
			}
			log.Printf("[ERROR] Failed to verify tenant membership: %v", err)
			c.JSON(http.StatusInternalServerError, models.ErrorResponse{
				Error:   "tenant_verification_failed",
				Message: "Failed to verify tenant access",
			})
			c.Abort()
			return
		}

		// Get tenant details
		tenant, err := tm.tenantRepo.GetByID(c.Request.Context(), tenantID)
		if err != nil {
			log.Printf("[ERROR] Failed to get tenant: %v", err)
			c.JSON(http.StatusInternalServerError, models.ErrorResponse{
				Error:   "tenant_lookup_failed",
				Message: "Failed to load organization",
			})
			c.Abort()
			return
		}

		if tenant == nil || !tenant.IsActive {
			c.JSON(http.StatusForbidden, models.ErrorResponse{
				Error:   "tenant_inactive",
				Message: "This organization is not active",
			})
			c.Abort()
			return
		}

		// Set tenant context
		c.Set(TenantIDKey, tenantID)
		c.Set(TenantKey, tenant)

		// Set user's tenant role in context
		if membership.Role.Name != "" {
			log.Printf("[DEBUG] RequireTenant: setting tenant_role=%s for user=%s in tenant=%s",
				membership.Role.Name, userUUID, tenantID)
			c.Set(TenantRoleKey, membership.Role.Name)
		} else {
			log.Printf("[DEBUG] RequireTenant: membership.Role.Name is empty for user=%s in tenant=%s (role_id=%s)",
				userUUID, tenantID, membership.RoleID)
		}

		c.Next()
	}
}

// OptionalTenant is middleware that sets tenant context if available, but doesn't require it
func (tm *TenantMiddleware) OptionalTenant() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Try to get tenant ID from header
		tenantIDStr := c.GetHeader(TenantHeader)
		if tenantIDStr == "" {
			c.Next()
			return
		}

		tenantID, err := uuid.Parse(tenantIDStr)
		if err != nil {
			// Invalid format, just continue without tenant context
			c.Next()
			return
		}

		// Get user ID if available
		userID, exists := c.Get(userIDKey)
		if !exists {
			c.Next()
			return
		}

		userUUID, ok := userID.(uuid.UUID)
		if !ok {
			c.Next()
			return
		}

		// Verify membership (silently fail if not)
		belongsTo, _ := tm.tenantRepo.UserBelongsToTenant(c.Request.Context(), userUUID, tenantID)
		if !belongsTo {
			c.Next()
			return
		}

		// Set tenant context
		tenant, err := tm.tenantRepo.GetByID(c.Request.Context(), tenantID)
		if err == nil && tenant != nil && tenant.IsActive {
			c.Set(TenantIDKey, tenantID)
			c.Set(TenantKey, tenant)
		}

		c.Next()
	}
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
