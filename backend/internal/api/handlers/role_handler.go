package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/javaknight1/servicepro/backend/internal/models"
	"github.com/javaknight1/servicepro/backend/internal/repository"
)

// RoleHandler handles role-related HTTP requests
type RoleHandler struct {
	permissionRepo *repository.PermissionRepository
}

// NewRoleHandler creates a new role handler
func NewRoleHandler(permissionRepo *repository.PermissionRepository) *RoleHandler {
	return &RoleHandler{
		permissionRepo: permissionRepo,
	}
}

// GetRoles returns all system roles
// GET /api/v1/roles
func (h *RoleHandler) GetRoles(c *gin.Context) {
	roles, err := h.permissionRepo.GetAllRoles()
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "fetch_failed",
			Message: "Failed to fetch roles",
		})
		return
	}

	c.JSON(http.StatusOK, roles)
}

// GetPermissions returns all system permissions
// GET /api/v1/permissions
func (h *RoleHandler) GetPermissions(c *gin.Context) {
	permissions, err := h.permissionRepo.GetAllPermissions()
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "fetch_failed",
			Message: "Failed to fetch permissions",
		})
		return
	}

	c.JSON(http.StatusOK, permissions)
}

// GetRolePermissions returns permissions for a specific role
// GET /api/v1/roles/:id/permissions
func (h *RoleHandler) GetRolePermissions(c *gin.Context) {
	roleID := c.Param("id")

	permissions, err := h.permissionRepo.GetRolePermissions(roleID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "fetch_failed",
			Message: "Failed to fetch role permissions",
		})
		return
	}

	c.JSON(http.StatusOK, permissions)
}
