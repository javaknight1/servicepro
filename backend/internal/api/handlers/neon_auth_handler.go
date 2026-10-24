package handlers

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/javaknight1/servicepro/backend/internal/api/apierror"
	"github.com/javaknight1/servicepro/backend/internal/models"
	"github.com/javaknight1/servicepro/backend/internal/services"
	"github.com/javaknight1/servicepro/backend/pkg/auth"
	"github.com/javaknight1/servicepro/backend/pkg/auth/neonauth"
)

// NeonTokenVerifier verifies Neon Auth JWTs (*neonauth.Verifier).
type NeonTokenVerifier interface {
	Verify(ctx context.Context, token string) (*neonauth.Claims, error)
}

// NeonProvisioner links or creates the user for a Neon Auth identity.
type NeonProvisioner interface {
	Provision(ctx context.Context, id services.NeonIdentity, inviteCode, invitationToken *string) (*services.NeonProvisionResult, error)
}

// NeonAuthHandler handles the Neon Auth (managed Better Auth) endpoints (T158).
type NeonAuthHandler struct {
	verifier    NeonTokenVerifier // nil when NEON_AUTH_BASE_URL is unset
	provisioner NeonProvisioner
}

// NewNeonAuthHandler creates the handler. verifier may be nil (Neon Auth off).
func NewNeonAuthHandler(verifier NeonTokenVerifier, provisioner NeonProvisioner) *NeonAuthHandler {
	return &NeonAuthHandler{verifier: verifier, provisioner: provisioner}
}

// Provision handles POST /api/v1/auth/neon/provision
// @Summary Link or create the ServicePro user for a Neon Auth sign-in
// @Description Called after signing in with Neon Auth when the API answers 401 account_not_linked. Authenticated by the Neon Auth JWT (Bearer). Links an existing user with the same verified email, or creates one when an invitation for that email or a valid invite code admits it. Idempotent.
// @Tags auth
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body models.NeonProvisionRequest false "Invite code or invitation token (needed only for a new user)"
// @Success 200 {object} models.NeonProvisionResponse
// @Failure 400 {object} models.ErrorResponse "invalid_email"
// @Failure 401 {object} models.ErrorResponse "Missing or invalid Neon Auth token"
// @Failure 403 {object} models.ErrorResponse "email_not_verified, registration_closed, invalid_invite_code or invalid_invitation"
// @Failure 404 {object} models.ErrorResponse "Neon Auth is not enabled"
// @Failure 409 {object} models.ErrorResponse "identity_conflict: the email is linked to another sign-in"
// @Failure 500 {object} models.ErrorResponse "Internal server error"
// @Router /api/v1/auth/neon/provision [post]
func (h *NeonAuthHandler) Provision(c *gin.Context) {
	if h.verifier == nil {
		apierror.WriteResponse(c, http.StatusNotFound, models.ErrorResponse{Error: "not_found", Message: "Neon Auth is not enabled"})
		return
	}
	token, ok := strings.CutPrefix(c.GetHeader("Authorization"), "Bearer ")
	if !ok || !neonauth.IsNeonToken(token) {
		apierror.WriteResponse(c, http.StatusUnauthorized, models.ErrorResponse{Error: "unauthorized", Message: "A Neon Auth bearer token is required"})
		return
	}
	claims, err := h.verifier.Verify(c.Request.Context(), token)
	if err != nil {
		apierror.WriteResponse(c, http.StatusUnauthorized, models.ErrorResponse{Error: "unauthorized", Message: "Invalid or expired token"})
		return
	}

	var req models.NeonProvisionRequest
	if c.Request.ContentLength != 0 {
		if err := c.ShouldBindJSON(&req); err != nil {
			apierror.BindError(c, err)
			return
		}
	}

	res, err := h.provisioner.Provision(c.Request.Context(), services.NeonIdentity{
		AuthUserID:    claims.Subject,
		Email:         claims.Email,
		EmailVerified: claims.EmailVerified,
	}, req.InviteCode, req.InvitationToken)
	if err != nil {
		writeProvisionError(c, err)
		return
	}
	c.JSON(http.StatusOK, models.NeonProvisionResponse{
		UserID:             res.UserID.String(),
		Created:            res.Created,
		Linked:             res.Linked,
		InvitationAccepted: res.InvitationAccepted,
	})
}

func writeProvisionError(c *gin.Context, err error) {
	forbidden := func(code, msg string) {
		apierror.WriteResponse(c, http.StatusForbidden, models.ErrorResponse{Error: code, Message: msg})
	}
	switch {
	case errors.Is(err, services.ErrNeonEmailNotVerified):
		forbidden("email_not_verified", "Verify your email address, then try again.")
	case errors.Is(err, services.ErrRegistrationClosed):
		forbidden("registration_closed", "ServicePro is invitation only. A valid invite code or invitation is required.")
	case errors.Is(err, services.ErrInvalidInviteCode):
		forbidden("invalid_invite_code", "That invite code is not valid. It may have expired, been used up or been revoked.")
	case errors.Is(err, services.ErrInvalidInvitation):
		forbidden("invalid_invitation", "This invitation is no longer valid for this email address.")
	case errors.Is(err, services.ErrAuthIdentityConflict):
		apierror.WriteResponse(c, http.StatusConflict, models.ErrorResponse{Error: "identity_conflict", Message: "This email is already linked to a different sign-in."})
	case errors.Is(err, auth.ErrInvalidEmailFormat):
		apierror.WriteResponse(c, http.StatusBadRequest, models.ErrorResponse{Error: "invalid_email", Message: "Invalid email format"})
	default:
		apierror.Internal(c, err, "An error occurred while processing your request")
	}
}
