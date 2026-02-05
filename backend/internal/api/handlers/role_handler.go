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
// @Summary		Get all roles
// @Description	Retrieves all system roles
// @Tags			Roles
// @Accept			json
// @Produce		json
// @Security		BearerAuth
// @Success		200	{array}		models.Role
// @Failure		500	{object}	models.ErrorResponse
// @Router			/roles [get]
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
// @Summary		Get all permissions
// @Description	Retrieves all system permissions
// @Tags			Roles
// @Accept			json
// @Produce		json
// @Security		BearerAuth
// @Success		200	{array}		models.Permission
// @Failure		500	{object}	models.ErrorResponse
// @Router			/permissions [get]
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
// @Summary		Get role permissions
// @Description	Retrieves all permissions assigned to a specific role
// @Tags			Roles
// @Accept			json
// @Produce		json
// @Security		BearerAuth
// @Param			id	path		string	true	"Role ID"
// @Success		200	{array}		models.Permission
// @Failure		500	{object}	models.ErrorResponse
// @Router			/roles/{id}/permissions [get]
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
