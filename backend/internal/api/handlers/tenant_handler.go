package handlers

import (
	"errors"
	"fmt"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/javaknight1/servicepro/backend/internal/api/middleware"
	"github.com/javaknight1/servicepro/backend/internal/models"
	"github.com/javaknight1/servicepro/backend/internal/repository"
	"github.com/javaknight1/servicepro/backend/internal/services"
)

// TenantHandler handles tenant-related HTTP requests
type TenantHandler struct {
	tenantService *services.TenantService
}

// NewTenantHandler creates a new tenant handler
func NewTenantHandler(tenantService *services.TenantService) *TenantHandler {
	return &TenantHandler{
		tenantService: tenantService,
	}
}

// CreateTenant creates a new tenant
// POST /api/v1/tenants
func (h *TenantHandler) CreateTenant(c *gin.Context) {
	var req models.CreateTenantRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "invalid_request",
			Message: err.Error(),
		})
		return
	}

	userID, err := middleware.GetUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, models.ErrorResponse{
			Error:   "unauthorized",
			Message: "User not authenticated",
		})
		return
	}

	tenant, err := h.tenantService.CreateTenant(c.Request.Context(), &req, userID)
	if err != nil {
		switch {
		case errors.Is(err, services.ErrInvalidTenantName):
			c.JSON(http.StatusBadRequest, models.ErrorResponse{
				Error:   "invalid_name",
				Message: "Invalid tenant name",
			})
		case errors.Is(err, services.ErrInvalidTenantSlug):
			c.JSON(http.StatusBadRequest, models.ErrorResponse{
				Error:   "invalid_slug",
				Message: "Invalid or already used slug",
			})
		default:
			c.JSON(http.StatusInternalServerError, models.ErrorResponse{
				Error:   "create_failed",
				Message: "Failed to create organization",
			})
		}
		return
	}

	c.JSON(http.StatusCreated, tenant.ToResponse())
}

// GetTenant retrieves a tenant by ID
// GET /api/v1/tenants/:id
func (h *TenantHandler) GetTenant(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "invalid_id",
			Message: "Invalid tenant ID format",
		})
		return
	}

	tenant, err := h.tenantService.GetTenant(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, repository.ErrTenantNotFound) {
			c.JSON(http.StatusNotFound, models.ErrorResponse{
				Error:   "not_found",
				Message: "Organization not found",
			})
			return
		}
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "fetch_failed",
			Message: "Failed to fetch organization",
		})
		return
	}

	c.JSON(http.StatusOK, tenant.ToResponse())
}

// UpdateTenant updates a tenant
// PUT /api/v1/tenants/:id
func (h *TenantHandler) UpdateTenant(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "invalid_id",
			Message: "Invalid tenant ID format",
		})
		return
	}

	var req models.UpdateTenantRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "invalid_request",
			Message: err.Error(),
		})
		return
	}

	tenant, err := h.tenantService.UpdateTenant(c.Request.Context(), id, &req)
	if err != nil {
		switch {
		case errors.Is(err, repository.ErrTenantNotFound):
			c.JSON(http.StatusNotFound, models.ErrorResponse{
				Error:   "not_found",
				Message: "Organization not found",
			})
		case errors.Is(err, services.ErrInvalidTenantName):
			c.JSON(http.StatusBadRequest, models.ErrorResponse{
				Error:   "invalid_name",
				Message: "Invalid tenant name",
			})
		case errors.Is(err, services.ErrInvalidTenantSlug):
			c.JSON(http.StatusBadRequest, models.ErrorResponse{
				Error:   "invalid_slug",
				Message: "Invalid or already used slug",
			})
		default:
			c.JSON(http.StatusInternalServerError, models.ErrorResponse{
				Error:   "update_failed",
				Message: "Failed to update organization",
			})
		}
		return
	}

	c.JSON(http.StatusOK, tenant.ToResponse())
}

// DeleteTenant deletes a tenant
// DELETE /api/v1/tenants/:id
func (h *TenantHandler) DeleteTenant(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "invalid_id",
			Message: "Invalid tenant ID format",
		})
		return
	}

	if err := h.tenantService.DeleteTenant(c.Request.Context(), id); err != nil {
		if errors.Is(err, repository.ErrTenantNotFound) {
			c.JSON(http.StatusNotFound, models.ErrorResponse{
				Error:   "not_found",
				Message: "Organization not found",
			})
			return
		}
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "delete_failed",
			Message: "Failed to delete organization",
		})
		return
	}

	c.JSON(http.StatusNoContent, nil)
}

// ListUserTenants lists all tenants the current user belongs to
// GET /api/v1/tenants
func (h *TenantHandler) ListUserTenants(c *gin.Context) {
	userID, err := middleware.GetUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, models.ErrorResponse{
			Error:   "unauthorized",
			Message: "User not authenticated",
		})
		return
	}

	tenants, err := h.tenantService.GetUserTenants(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "fetch_failed",
			Message: "Failed to fetch organizations",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"tenants": tenants,
	})
}

// AddMember adds a member to a tenant
// POST /api/v1/tenants/:id/members
func (h *TenantHandler) AddMember(c *gin.Context) {
	idStr := c.Param("id")
	tenantID, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "invalid_id",
			Message: "Invalid tenant ID format",
		})
		return
	}

	var req models.AddMemberRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "invalid_request",
			Message: err.Error(),
		})
		return
	}

	userID, err := middleware.GetUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, models.ErrorResponse{
			Error:   "unauthorized",
			Message: "User not authenticated",
		})
		return
	}

	member, err := h.tenantService.AddMember(c.Request.Context(), tenantID, &req, userID)
	if err != nil {
		if errors.Is(err, repository.ErrUserAlreadyInTenant) {
			c.JSON(http.StatusConflict, models.ErrorResponse{
				Error:   "already_member",
				Message: "User is already a member of this organization",
			})
			return
		}
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "add_member_failed",
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusCreated, member)
}

// RemoveMember removes a member from a tenant
// DELETE /api/v1/tenants/:id/members/:userId
func (h *TenantHandler) RemoveMember(c *gin.Context) {
	idStr := c.Param("id")
	tenantID, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "invalid_id",
			Message: "Invalid tenant ID format",
		})
		return
	}

	userIDStr := c.Param("userId")
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "invalid_user_id",
			Message: "Invalid user ID format",
		})
		return
	}

	if err := h.tenantService.RemoveMember(c.Request.Context(), tenantID, userID); err != nil {
		switch {
		case errors.Is(err, repository.ErrUserNotInTenant):
			c.JSON(http.StatusNotFound, models.ErrorResponse{
				Error:   "not_member",
				Message: "User is not a member of this organization",
			})
		case errors.Is(err, services.ErrCannotRemoveOwner):
			c.JSON(http.StatusForbidden, models.ErrorResponse{
				Error:   "cannot_remove_owner",
				Message: "Cannot remove the organization owner",
			})
		default:
			c.JSON(http.StatusInternalServerError, models.ErrorResponse{
				Error:   "remove_member_failed",
				Message: "Failed to remove member",
			})
		}
		return
	}

	c.JSON(http.StatusNoContent, nil)
}

// UpdateMemberRole updates a member's role
// PUT /api/v1/tenants/:id/members/:userId/role
func (h *TenantHandler) UpdateMemberRole(c *gin.Context) {
	idStr := c.Param("id")
	tenantID, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "invalid_id",
			Message: "Invalid tenant ID format",
		})
		return
	}

	userIDStr := c.Param("userId")
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "invalid_user_id",
			Message: "Invalid user ID format",
		})
		return
	}

	var req models.UpdateMemberRoleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "invalid_request",
			Message: err.Error(),
		})
		return
	}

	if err := h.tenantService.UpdateMemberRole(c.Request.Context(), tenantID, userID, req.RoleID); err != nil {
		switch {
		case errors.Is(err, repository.ErrUserNotInTenant):
			c.JSON(http.StatusNotFound, models.ErrorResponse{
				Error:   "not_member",
				Message: "User is not a member of this organization",
			})
		case errors.Is(err, services.ErrCannotChangeOwnerRole):
			c.JSON(http.StatusForbidden, models.ErrorResponse{
				Error:   "cannot_change_owner_role",
				Message: "Cannot change the organization owner's role",
			})
		default:
			c.JSON(http.StatusInternalServerError, models.ErrorResponse{
				Error:   "update_role_failed",
				Message: "Failed to update member role",
			})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Role updated successfully"})
}

// GetTenantMembers gets all members of a tenant
// GET /api/v1/tenants/:id/members
func (h *TenantHandler) GetTenantMembers(c *gin.Context) {
	idStr := c.Param("id")
	tenantID, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "invalid_id",
			Message: "Invalid tenant ID format",
		})
		return
	}

	// Verify user belongs to this tenant
	userID, err := middleware.GetUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, models.ErrorResponse{
			Error:   "unauthorized",
			Message: "User not authenticated",
		})
		return
	}

	log.Printf("[TENANT-HANDLER] GetTenantMembers: Checking if user %s belongs to tenant %s", userID, tenantID)

	belongs, err := h.tenantService.UserBelongsToTenant(c.Request.Context(), userID, tenantID)
	if err != nil {
		log.Printf("[TENANT-HANDLER] GetTenantMembers: Error checking membership: %v", err)
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "membership_check_failed",
			Message: fmt.Sprintf("Failed to check membership: %v", err),
		})
		return
	}

	log.Printf("[TENANT-HANDLER] GetTenantMembers: UserBelongsToTenant result=%v", belongs)

	if !belongs {
		c.JSON(http.StatusForbidden, models.ErrorResponse{
			Error:   "forbidden",
			Message: fmt.Sprintf("User %s is not a member of tenant %s", userID, tenantID),
		})
		return
	}

	members, err := h.tenantService.GetTenantMembers(c.Request.Context(), tenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "fetch_failed",
			Message: "Failed to fetch members",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"members": members,
	})
}
