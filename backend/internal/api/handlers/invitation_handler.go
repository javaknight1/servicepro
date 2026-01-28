package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/javaknight1/servicepro/backend/internal/models"
	"github.com/javaknight1/servicepro/backend/internal/services"
)

// InvitationHandler handles invitation HTTP requests
type InvitationHandler struct {
	invitationService *services.InvitationService
}

// NewInvitationHandler creates a new invitation handler
func NewInvitationHandler(invitationService *services.InvitationService) *InvitationHandler {
	return &InvitationHandler{
		invitationService: invitationService,
	}
}

// InviteMemberRequest represents the request to invite a member
type InviteMemberRequest struct {
	Email  string    `json:"email" binding:"required,email"`
	RoleID uuid.UUID `json:"role_id" binding:"required"`
}

// InviteMember godoc
// @Summary Invite a member to organization
// @Description Invite a user to join the organization by email
// @Tags invitations
// @Accept json
// @Produce json
// @Param tenant_id path string true "Tenant ID"
// @Param invitation body InviteMemberRequest true "Invitation details"
// @Success 201 {object} models.InvitationResponse
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 409 {object} ErrorResponse "User already a member"
// @Failure 500 {object} ErrorResponse
// @Router /tenants/{tenant_id}/invitations [post]
// @Security BearerAuth
func (h *InvitationHandler) InviteMember(c *gin.Context) {
	ctx := c.Request.Context()

	tenantID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid tenant ID"})
		return
	}

	var req InviteMemberRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: err.Error()})
		return
	}

	// Get user ID from context
	userIDValue, exists := c.Get("user_id")
	if !exists || userIDValue == nil {
		c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "Unauthorized"})
		return
	}
	userID, ok := userIDValue.(uuid.UUID)
	if !ok {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Invalid user ID in context"})
		return
	}

	invitation, err := h.invitationService.InviteMember(ctx, tenantID, req.Email, req.RoleID, userID)
	if err != nil {
		switch err {
		case services.ErrUserAlreadyMember:
			c.JSON(http.StatusConflict, ErrorResponse{Error: "User is already a member of this organization"})
		case services.ErrPendingInvitationExists:
			c.JSON(http.StatusConflict, ErrorResponse{Error: "A pending invitation already exists for this email"})
		default:
			c.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		}
		return
	}

	c.JSON(http.StatusCreated, invitation)
}

// GetPendingInvitations godoc
// @Summary Get pending invitations
// @Description Get all pending invitations for the organization
// @Tags invitations
// @Accept json
// @Produce json
// @Param tenant_id path string true "Tenant ID"
// @Success 200 {array} models.InvitationResponse
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /tenants/{tenant_id}/invitations [get]
// @Security BearerAuth
func (h *InvitationHandler) GetPendingInvitations(c *gin.Context) {
	ctx := c.Request.Context()

	tenantID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid tenant ID"})
		return
	}

	invitations, err := h.invitationService.GetPendingInvitationsForTenant(ctx, tenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, invitations)
}

// CancelInvitation godoc
// @Summary Cancel an invitation
// @Description Cancel a pending invitation
// @Tags invitations
// @Accept json
// @Produce json
// @Param tenant_id path string true "Tenant ID"
// @Param invitation_id path string true "Invitation ID"
// @Success 204
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /tenants/{tenant_id}/invitations/{invitation_id} [delete]
// @Security BearerAuth
func (h *InvitationHandler) CancelInvitation(c *gin.Context) {
	ctx := c.Request.Context()

	invitationID, err := uuid.Parse(c.Param("invitation_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid invitation ID"})
		return
	}

	err = h.invitationService.CancelInvitation(ctx, invitationID)
	if err != nil {
		if err == services.ErrInvitationNotFound {
			c.JSON(http.StatusNotFound, ErrorResponse{Error: "Invitation not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}

	c.Status(http.StatusNoContent)
}

// ResendInvitation godoc
// @Summary Resend an invitation
// @Description Resend an invitation email
// @Tags invitations
// @Accept json
// @Produce json
// @Param tenant_id path string true "Tenant ID"
// @Param invitation_id path string true "Invitation ID"
// @Success 200 {object} map[string]string
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /tenants/{tenant_id}/invitations/{invitation_id}/resend [post]
// @Security BearerAuth
func (h *InvitationHandler) ResendInvitation(c *gin.Context) {
	ctx := c.Request.Context()

	invitationID, err := uuid.Parse(c.Param("invitation_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid invitation ID"})
		return
	}

	err = h.invitationService.ResendInvitation(ctx, invitationID)
	if err != nil {
		if err == services.ErrInvitationNotFound {
			c.JSON(http.StatusNotFound, ErrorResponse{Error: "Invitation not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Invitation resent successfully"})
}

// GetInvitationByToken godoc
// @Summary Get invitation by token
// @Description Get invitation details by token (for registration page)
// @Tags invitations
// @Accept json
// @Produce json
// @Param token path string true "Invitation token"
// @Success 200 {object} models.InvitationResponse
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 410 {object} ErrorResponse "Invitation expired"
// @Failure 500 {object} ErrorResponse
// @Router /invitations/{token} [get]
func (h *InvitationHandler) GetInvitationByToken(c *gin.Context) {
	ctx := c.Request.Context()

	token := c.Param("token")
	if token == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Token is required"})
		return
	}

	invitation, err := h.invitationService.GetInvitationByToken(ctx, token)
	if err != nil {
		if err == services.ErrInvitationNotFound {
			c.JSON(http.StatusNotFound, ErrorResponse{Error: "Invitation not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}

	if invitation.IsExpired() {
		c.JSON(http.StatusGone, ErrorResponse{Error: "Invitation has expired"})
		return
	}

	if invitation.Status != models.InvitationStatusPending {
		c.JSON(http.StatusGone, ErrorResponse{Error: "Invitation is no longer valid"})
		return
	}

	c.JSON(http.StatusOK, invitation.ToResponse(false))
}

// AcceptInvitation godoc
// @Summary Accept an invitation
// @Description Accept an invitation (for existing users)
// @Tags invitations
// @Accept json
// @Produce json
// @Param token path string true "Invitation token"
// @Success 200 {object} map[string]string
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 410 {object} ErrorResponse "Invitation expired or already used"
// @Failure 500 {object} ErrorResponse
// @Router /invitations/{token}/accept [post]
// @Security BearerAuth
func (h *InvitationHandler) AcceptInvitation(c *gin.Context) {
	ctx := c.Request.Context()

	token := c.Param("token")
	if token == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Token is required"})
		return
	}

	// Get user ID from context
	userIDValue, exists := c.Get("user_id")
	if !exists || userIDValue == nil {
		c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "Unauthorized"})
		return
	}
	userID, ok := userIDValue.(uuid.UUID)
	if !ok {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Invalid user ID in context"})
		return
	}

	err := h.invitationService.AcceptInvitation(ctx, token, userID)
	if err != nil {
		switch err {
		case services.ErrInvitationNotFound:
			c.JSON(http.StatusNotFound, ErrorResponse{Error: "Invitation not found"})
		case services.ErrInvitationExpired:
			c.JSON(http.StatusGone, ErrorResponse{Error: "Invitation has expired"})
		case services.ErrInvitationAlreadyUsed:
			c.JSON(http.StatusGone, ErrorResponse{Error: "Invitation has already been used"})
		default:
			c.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Invitation accepted successfully"})
}
