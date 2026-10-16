package handlers

import (
	"context"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/javaknight1/servicepro/backend/internal/api/apierror"
	"github.com/javaknight1/servicepro/backend/internal/api/middleware"
	"github.com/javaknight1/servicepro/backend/internal/models"
	"github.com/javaknight1/servicepro/backend/internal/repository"
	"github.com/javaknight1/servicepro/backend/internal/services"
)

// EffectivePermissions computes the permission names a user holds in one
// tenant (implemented by the per-tenant permissions.PermissionChecker).
type EffectivePermissions interface {
	EffectivePermissionNames(ctx context.Context, userID, tenantID uuid.UUID) ([]string, error)
}

// TenantHandler handles tenant-related HTTP requests
type TenantHandler struct {
	tenantService *services.TenantService
	permissions   EffectivePermissions
}

// NewTenantHandler creates a new tenant handler
func NewTenantHandler(tenantService *services.TenantService) *TenantHandler {
	return &TenantHandler{
		tenantService: tenantService,
	}
}

// SetPermissions wires the per-tenant permission checker used by GetMyAccess.
func (h *TenantHandler) SetPermissions(p EffectivePermissions) {
	h.permissions = p
}

// GetMyAccess returns the caller's role and effective permissions in a tenant
// @Summary		Get my access in a tenant
// @Description	Returns the caller's membership role and the exact permission names the API grants them in this tenant. Any active member may call it; non-members get 403.
// @Tags			Tenants
// @Produce		json
// @Security		BearerAuth
// @Param			id	path		string	true	"Tenant ID"
// @Success		200	{object}	models.TenantAccessResponse
// @Failure		400	{object}	models.ErrorResponse
// @Failure		401	{object}	models.ErrorResponse
// @Failure		403	{object}	models.ErrorResponse
// @Failure		500	{object}	models.ErrorResponse
// @Router			/tenants/{id}/me [get]
func (h *TenantHandler) GetMyAccess(c *gin.Context) {
	userID, err := middleware.GetUserID(c)
	if err != nil {
		apierror.WriteResponse(c, http.StatusUnauthorized, models.ErrorResponse{
			Error:   "unauthorized",
			Message: "User not authenticated",
		})
		return
	}
	// The tenant comes from RequireTenantFromPath, which verified membership.
	tenantID, err := middleware.GetTenantID(c)
	if err != nil {
		apierror.WriteResponse(c, http.StatusBadRequest, models.ErrorResponse{
			Error:   "invalid_tenant_id",
			Message: "Invalid tenant ID",
		})
		return
	}
	if h.permissions == nil {
		apierror.Internal(c, err, "An error occurred while processing your request")
		return
	}

	names, err := h.permissions.EffectivePermissionNames(c.Request.Context(), userID, tenantID)
	if err != nil {
		apierror.Internal(c, err, "Failed to load permissions")
		return
	}
	if names == nil {
		names = []string{} // always a JSON array
	}

	resp := models.TenantAccessResponse{
		TenantID:    tenantID,
		Role:        c.GetString(middleware.TenantRoleKey),
		RoleLevel:   middleware.GetTenantRoleLevel(c),
		Permissions: names,
	}
	if roleID, ok := c.Get(middleware.TenantRoleIDKey); ok {
		if id, ok := roleID.(uuid.UUID); ok {
			resp.RoleID = id
		}
	}
	c.JSON(http.StatusOK, resp)
}

// CreateTenant creates a new tenant
// @Summary		Create tenant
// @Description	Creates a new organization/tenant
// @Tags			Tenants
// @Accept			json
// @Produce		json
// @Security		BearerAuth
// @Param			request	body		models.CreateTenantRequest	true	"Create tenant request"
// @Success		201		{object}	models.TenantResponse
// @Failure		400		{object}	models.ErrorResponse
// @Failure		401		{object}	models.ErrorResponse
// @Failure		500		{object}	models.ErrorResponse
// @Router			/tenants [post]
func (h *TenantHandler) CreateTenant(c *gin.Context) {
	var req models.CreateTenantRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		apierror.BindError(c, err)
		return
	}

	userID, err := middleware.GetUserID(c)
	if err != nil {
		apierror.WriteResponse(c, http.StatusUnauthorized, models.ErrorResponse{
			Error:   "unauthorized",
			Message: "User not authenticated",
		})
		return
	}

	tenant, err := h.tenantService.CreateTenant(c.Request.Context(), &req, userID)
	if err != nil {
		switch {
		case errors.Is(err, services.ErrInvalidTenantName):
			apierror.WriteResponse(c, http.StatusBadRequest, models.ErrorResponse{
				Error:   "invalid_name",
				Message: "Invalid tenant name",
			})
		case errors.Is(err, services.ErrInvalidTenantSlug):
			apierror.WriteResponse(c, http.StatusBadRequest, models.ErrorResponse{
				Error:   "invalid_slug",
				Message: "Invalid or already used slug",
			})
		default:
			apierror.Internal(c, err, "Failed to create organization")
		}
		return
	}

	c.JSON(http.StatusCreated, tenant.ToResponse())
}

// GetTenant retrieves a tenant by ID
// @Summary		Get tenant
// @Description	Retrieves a tenant by ID
// @Tags			Tenants
// @Accept			json
// @Produce		json
// @Security		BearerAuth
// @Param			id	path		string	true	"Tenant ID"
// @Success		200	{object}	models.TenantResponse
// @Failure		400	{object}	models.ErrorResponse
// @Failure		404	{object}	models.ErrorResponse
// @Failure		500	{object}	models.ErrorResponse
// @Router			/tenants/{id} [get]
func (h *TenantHandler) GetTenant(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		apierror.WriteResponse(c, http.StatusBadRequest, models.ErrorResponse{
			Error:   "invalid_id",
			Message: "Invalid tenant ID format",
		})
		return
	}

	tenant, err := h.tenantService.GetTenant(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, repository.ErrTenantNotFound) {
			apierror.WriteResponse(c, http.StatusNotFound, models.ErrorResponse{
				Error:   "not_found",
				Message: "Organization not found",
			})
			return
		}
		apierror.Internal(c, err, "Failed to fetch organization")
		return
	}

	c.JSON(http.StatusOK, tenant.ToResponse())
}

// UpdateTenant updates a tenant
// @Summary		Update tenant
// @Description	Updates an existing tenant
// @Tags			Tenants
// @Accept			json
// @Produce		json
// @Security		BearerAuth
// @Param			id		path		string						true	"Tenant ID"
// @Param			request	body		models.UpdateTenantRequest	true	"Update tenant request"
// @Success		200		{object}	models.TenantResponse
// @Failure		400		{object}	models.ErrorResponse
// @Failure		404		{object}	models.ErrorResponse
// @Failure		500		{object}	models.ErrorResponse
// @Router			/tenants/{id} [put]
func (h *TenantHandler) UpdateTenant(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		apierror.WriteResponse(c, http.StatusBadRequest, models.ErrorResponse{
			Error:   "invalid_id",
			Message: "Invalid tenant ID format",
		})
		return
	}

	var req models.UpdateTenantRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		apierror.BindError(c, err)
		return
	}

	tenant, err := h.tenantService.UpdateTenant(c.Request.Context(), id, &req)
	if err != nil {
		switch {
		case errors.Is(err, repository.ErrTenantNotFound):
			apierror.WriteResponse(c, http.StatusNotFound, models.ErrorResponse{
				Error:   "not_found",
				Message: "Organization not found",
			})
		case errors.Is(err, services.ErrInvalidTenantName):
			apierror.WriteResponse(c, http.StatusBadRequest, models.ErrorResponse{
				Error:   "invalid_name",
				Message: "Invalid tenant name",
			})
		case errors.Is(err, services.ErrInvalidTenantSlug):
			apierror.WriteResponse(c, http.StatusBadRequest, models.ErrorResponse{
				Error:   "invalid_slug",
				Message: "Invalid or already used slug",
			})
		default:
			apierror.Internal(c, err, "Failed to update organization")
		}
		return
	}

	c.JSON(http.StatusOK, tenant.ToResponse())
}

// DeleteTenant deletes a tenant
// @Summary		Delete tenant
// @Description	Deletes a tenant and all associated data
// @Tags			Tenants
// @Accept			json
// @Produce		json
// @Security		BearerAuth
// @Param			id	path	string	true	"Tenant ID"
// @Success		204
// @Failure		400	{object}	models.ErrorResponse
// @Failure		404	{object}	models.ErrorResponse
// @Failure		500	{object}	models.ErrorResponse
// @Router			/tenants/{id} [delete]
func (h *TenantHandler) DeleteTenant(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		apierror.WriteResponse(c, http.StatusBadRequest, models.ErrorResponse{
			Error:   "invalid_id",
			Message: "Invalid tenant ID format",
		})
		return
	}

	if err := h.tenantService.DeleteTenant(c.Request.Context(), id); err != nil {
		if errors.Is(err, repository.ErrTenantNotFound) {
			apierror.WriteResponse(c, http.StatusNotFound, models.ErrorResponse{
				Error:   "not_found",
				Message: "Organization not found",
			})
			return
		}
		apierror.Internal(c, err, "Failed to delete organization")
		return
	}

	c.JSON(http.StatusNoContent, nil)
}

// ListUserTenants lists all tenants the current user belongs to
// @Summary		List user tenants
// @Description	Lists all organizations the current user belongs to
// @Tags			Tenants
// @Accept			json
// @Produce		json
// @Security		BearerAuth
// @Success		200	{object}	map[string][]models.TenantResponse
// @Failure		401	{object}	models.ErrorResponse
// @Failure		500	{object}	models.ErrorResponse
// @Router			/tenants [get]
func (h *TenantHandler) ListUserTenants(c *gin.Context) {
	userID, err := middleware.GetUserID(c)
	if err != nil {
		apierror.WriteResponse(c, http.StatusUnauthorized, models.ErrorResponse{
			Error:   "unauthorized",
			Message: "User not authenticated",
		})
		return
	}

	tenants, err := h.tenantService.GetUserTenants(c.Request.Context(), userID)
	if err != nil {
		apierror.Internal(c, err, "Failed to fetch organizations")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"tenants": tenants,
	})
}

// AddMember adds a member to a tenant
// @Summary		Add member
// @Description	Adds a new member to the organization
// @Tags			Tenants
// @Accept			json
// @Produce		json
// @Security		BearerAuth
// @Param			id		path		string					true	"Tenant ID"
// @Param			request	body		models.AddMemberRequest	true	"Add member request"
// @Success		201		{object}	models.TenantMemberResponse
// @Failure		400		{object}	models.ErrorResponse
// @Failure		401		{object}	models.ErrorResponse
// @Failure		403		{object}	models.ErrorResponse
// @Failure		409		{object}	models.ErrorResponse
// @Failure		500		{object}	models.ErrorResponse
// @Router			/tenants/{id}/members [post]
func (h *TenantHandler) AddMember(c *gin.Context) {
	idStr := c.Param("id")
	tenantID, err := uuid.Parse(idStr)
	if err != nil {
		apierror.WriteResponse(c, http.StatusBadRequest, models.ErrorResponse{
			Error:   "invalid_id",
			Message: "Invalid tenant ID format",
		})
		return
	}

	var req models.AddMemberRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		apierror.BindError(c, err)
		return
	}

	userID, err := middleware.GetUserID(c)
	if err != nil {
		apierror.WriteResponse(c, http.StatusUnauthorized, models.ErrorResponse{
			Error:   "unauthorized",
			Message: "User not authenticated",
		})
		return
	}

	actor := services.MemberActor{UserID: userID, RoleLevel: middleware.GetTenantRoleLevel(c)}
	member, err := h.tenantService.AddMember(c.Request.Context(), tenantID, &req, actor)
	if err != nil {
		switch {
		case errors.Is(err, repository.ErrUserAlreadyInTenant):
			apierror.WriteResponse(c, http.StatusConflict, models.ErrorResponse{
				Error:   "already_member",
				Message: "User is already a member of this organization",
			})
		case writeRoleRuleError(c, err):
		case errors.Is(err, services.ErrMemberNotAddable):
			apierror.WriteResponse(c, http.StatusBadRequest, models.ErrorResponse{
				Error:   "add_member_failed",
				Message: "Could not add this member. Send them an invitation instead.",
			})
		default:
			apierror.Internal(c, err, "Failed to add member")
		}
		return
	}

	c.JSON(http.StatusCreated, member)
}

// RemoveMember removes a member from a tenant
// @Summary		Remove member
// @Description	Removes a member from the organization
// @Tags			Tenants
// @Accept			json
// @Produce		json
// @Security		BearerAuth
// @Param			id		path	string	true	"Tenant ID"
// @Param			userId	path	string	true	"User ID to remove"
// @Success		204
// @Failure		400		{object}	models.ErrorResponse
// @Failure		403		{object}	models.ErrorResponse
// @Failure		404		{object}	models.ErrorResponse
// @Failure		500		{object}	models.ErrorResponse
// @Router			/tenants/{id}/members/{userId} [delete]
func (h *TenantHandler) RemoveMember(c *gin.Context) {
	idStr := c.Param("id")
	tenantID, err := uuid.Parse(idStr)
	if err != nil {
		apierror.WriteResponse(c, http.StatusBadRequest, models.ErrorResponse{
			Error:   "invalid_id",
			Message: "Invalid tenant ID format",
		})
		return
	}

	userIDStr := c.Param("userId")
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		apierror.WriteResponse(c, http.StatusBadRequest, models.ErrorResponse{
			Error:   "invalid_user_id",
			Message: "Invalid user ID format",
		})
		return
	}

	actorID, err := middleware.GetUserID(c)
	if err != nil {
		apierror.WriteResponse(c, http.StatusUnauthorized, models.ErrorResponse{
			Error:   "unauthorized",
			Message: "User not authenticated",
		})
		return
	}
	actor := services.MemberActor{UserID: actorID, RoleLevel: middleware.GetTenantRoleLevel(c)}

	if err := h.tenantService.RemoveMember(c.Request.Context(), tenantID, userID, actor); err != nil {
		switch {
		case writeRoleRuleError(c, err):
		case errors.Is(err, repository.ErrUserNotInTenant):
			apierror.WriteResponse(c, http.StatusNotFound, models.ErrorResponse{
				Error:   "not_member",
				Message: "User is not a member of this organization",
			})
		case errors.Is(err, services.ErrCannotRemoveOwner):
			apierror.WriteResponse(c, http.StatusForbidden, models.ErrorResponse{
				Error:   "cannot_remove_owner",
				Message: "Cannot remove the organization owner",
			})
		default:
			apierror.Internal(c, err, "Failed to remove member")
		}
		return
	}

	c.JSON(http.StatusNoContent, nil)
}

// UpdateMemberRole updates a member's role
// @Summary		Update member role
// @Description	Updates the role assigned to a member in the organization
// @Tags			Tenants
// @Accept			json
// @Produce		json
// @Security		BearerAuth
// @Param			id		path		string							true	"Tenant ID"
// @Param			userId	path		string							true	"User ID"
// @Param			request	body		models.UpdateMemberRoleRequest	true	"Update role request"
// @Success		200		{object}	map[string]string
// @Failure		400		{object}	models.ErrorResponse
// @Failure		403		{object}	models.ErrorResponse
// @Failure		404		{object}	models.ErrorResponse
// @Failure		500		{object}	models.ErrorResponse
// @Router			/tenants/{id}/members/{userId}/role [put]
func (h *TenantHandler) UpdateMemberRole(c *gin.Context) {
	idStr := c.Param("id")
	tenantID, err := uuid.Parse(idStr)
	if err != nil {
		apierror.WriteResponse(c, http.StatusBadRequest, models.ErrorResponse{
			Error:   "invalid_id",
			Message: "Invalid tenant ID format",
		})
		return
	}

	userIDStr := c.Param("userId")
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		apierror.WriteResponse(c, http.StatusBadRequest, models.ErrorResponse{
			Error:   "invalid_user_id",
			Message: "Invalid user ID format",
		})
		return
	}

	var req models.UpdateMemberRoleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		apierror.BindError(c, err)
		return
	}

	actorID, err := middleware.GetUserID(c)
	if err != nil {
		apierror.WriteResponse(c, http.StatusUnauthorized, models.ErrorResponse{
			Error:   "unauthorized",
			Message: "User not authenticated",
		})
		return
	}
	actor := services.MemberActor{UserID: actorID, RoleLevel: middleware.GetTenantRoleLevel(c)}

	if err := h.tenantService.UpdateMemberRole(c.Request.Context(), tenantID, userID, req.RoleID, actor); err != nil {
		switch {
		case writeRoleRuleError(c, err):
		case errors.Is(err, repository.ErrUserNotInTenant):
			apierror.WriteResponse(c, http.StatusNotFound, models.ErrorResponse{
				Error:   "not_member",
				Message: "User is not a member of this organization",
			})
		case errors.Is(err, services.ErrCannotChangeOwnerRole):
			apierror.WriteResponse(c, http.StatusForbidden, models.ErrorResponse{
				Error:   "cannot_change_owner_role",
				Message: "Cannot change the organization owner's role",
			})
		default:
			apierror.Internal(c, err, "Failed to update member role")
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Role updated successfully"})
}

// GetTenantMembers gets all members of a tenant
// @Summary		Get tenant members
// @Description	Retrieves all members of an organization
// @Tags			Tenants
// @Accept			json
// @Produce		json
// @Security		BearerAuth
// @Param			id	path		string	true	"Tenant ID"
// @Success		200	{object}	map[string][]models.TenantMemberResponse
// @Failure		400	{object}	models.ErrorResponse
// @Failure		401	{object}	models.ErrorResponse
// @Failure		403	{object}	models.ErrorResponse
// @Failure		500	{object}	models.ErrorResponse
// @Router			/tenants/{id}/members [get]
func (h *TenantHandler) GetTenantMembers(c *gin.Context) {
	idStr := c.Param("id")
	tenantID, err := uuid.Parse(idStr)
	if err != nil {
		apierror.WriteResponse(c, http.StatusBadRequest, models.ErrorResponse{
			Error:   "invalid_id",
			Message: "Invalid tenant ID format",
		})
		return
	}

	// Membership in tenantID is verified by RequireTenantFromPath.
	members, err := h.tenantService.GetTenantMembers(c.Request.Context(), tenantID)
	if err != nil {
		apierror.Internal(c, err, "Failed to fetch members")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"members": members,
	})
}

// writeRoleRuleError maps the membership role-hierarchy errors to 403/400
// responses. It returns false when err is not one of them.
func writeRoleRuleError(c *gin.Context, err error) bool {
	switch {
	case errors.Is(err, services.ErrCannotChangeOwnRole):
		apierror.WriteResponse(c, http.StatusForbidden, models.ErrorResponse{
			Error:   "cannot_change_own_role",
			Message: "You cannot change your own role",
		})
	case errors.Is(err, services.ErrInsufficientRoleLevel):
		apierror.WriteResponse(c, http.StatusForbidden, models.ErrorResponse{
			Error:   "insufficient_role",
			Message: "You cannot grant a role above your own or manage a member who outranks you",
		})
	case errors.Is(err, services.ErrRoleNotAssignable):
		apierror.WriteResponse(c, http.StatusBadRequest, models.ErrorResponse{
			Error:   "invalid_role",
			Message: "This role cannot be assigned",
		})
	default:
		return false
	}
	return true
}
