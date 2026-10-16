package handlers

import (
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/javaknight1/servicepro/backend/internal/api/apierror"
	"github.com/javaknight1/servicepro/backend/internal/api/middleware"
	"github.com/javaknight1/servicepro/backend/internal/models"
	"github.com/javaknight1/servicepro/backend/internal/services"
)

// InviteCodeHandler exposes platform invite-code administration. Routes are
// restricted to platform super admins (user_roles), never tenant roles.
type InviteCodeHandler struct {
	service *services.InviteCodeService
}

// NewInviteCodeHandler creates an invite code handler.
func NewInviteCodeHandler(service *services.InviteCodeService) *InviteCodeHandler {
	return &InviteCodeHandler{service: service}
}

// CreateInviteCode creates a registration invite code.
// @Summary      Create invite code
// @Description  Creates a platform invite code for invite-only registration. The plaintext code is returned once. Platform super admins only.
// @Tags         Admin
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        body body      models.CreateInviteCodeRequest  true  "Invite code options"
// @Success      201  {object}  models.CreateInviteCodeResponse
// @Failure      400  {object}  models.ErrorResponse
// @Failure      401  {object}  models.ErrorResponse
// @Failure      403  {object}  models.ErrorResponse
// @Failure      500  {object}  models.ErrorResponse
// @Router       /admin/invite-codes [post]
func (h *InviteCodeHandler) CreateInviteCode(c *gin.Context) {
	var req models.CreateInviteCodeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		apierror.BindError(c, err)
		return
	}
	maxUses := 1
	if req.MaxUses != nil {
		maxUses = *req.MaxUses
	}
	hours := 168
	if req.ExpiresInHours != nil {
		hours = *req.ExpiresInHours
	}
	var createdBy *uuid.UUID
	if uid, err := middleware.GetUserID(c); err == nil {
		createdBy = &uid
	}

	code, ic, err := h.service.Create(c.Request.Context(), maxUses, time.Duration(hours)*time.Hour, createdBy, req.Note)
	if err != nil {
		apierror.Internal(c, err, "Failed to create invite code")
		return
	}
	c.JSON(http.StatusCreated, models.CreateInviteCodeResponse{Code: code, InviteCode: *ic})
}

// ListInviteCodes lists invite codes (without their plaintext).
// @Summary      List invite codes
// @Description  Lists platform invite codes, newest first. Platform super admins only.
// @Tags         Admin
// @Produce      json
// @Security     BearerAuth
// @Success      200  {array}   models.InviteCode
// @Failure      401  {object}  models.ErrorResponse
// @Failure      403  {object}  models.ErrorResponse
// @Failure      500  {object}  models.ErrorResponse
// @Router       /admin/invite-codes [get]
func (h *InviteCodeHandler) ListInviteCodes(c *gin.Context) {
	codes, err := h.service.List(c.Request.Context(), 200)
	if err != nil {
		apierror.Internal(c, err, "Failed to list invite codes")
		return
	}
	c.JSON(http.StatusOK, codes)
}

// RevokeInviteCode revokes an invite code.
// @Summary      Revoke invite code
// @Description  Revokes a platform invite code. Platform super admins only.
// @Tags         Admin
// @Produce      json
// @Security     BearerAuth
// @Param        id   path      string  true  "Invite code ID"
// @Success      204
// @Failure      400  {object}  models.ErrorResponse
// @Failure      404  {object}  models.ErrorResponse
// @Failure      500  {object}  models.ErrorResponse
// @Router       /admin/invite-codes/{id} [delete]
func (h *InviteCodeHandler) RevokeInviteCode(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		apierror.WriteResponse(c, http.StatusBadRequest, models.ErrorResponse{Error: "invalid_id", Message: "Invalid invite code ID"})
		return
	}
	if err := h.service.Revoke(c.Request.Context(), id); err != nil {
		if errors.Is(err, services.ErrInviteCodeNotFound) {
			apierror.WriteResponse(c, http.StatusNotFound, models.ErrorResponse{Error: "not_found", Message: "Invite code not found"})
			return
		}
		apierror.Internal(c, err, "Failed to revoke invite code")
		return
	}
	c.Status(http.StatusNoContent)
}
