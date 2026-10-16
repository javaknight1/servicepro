package handlers

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/javaknight1/servicepro/backend/internal/api/apierror"
	"github.com/javaknight1/servicepro/backend/internal/models"
	"github.com/javaknight1/servicepro/backend/internal/services"
)

// EmailVerificationHandler handles email verification endpoints
type EmailVerificationHandler struct {
	verificationService services.EmailVerificationServiceInterface
}

// NewEmailVerificationHandler creates a new email verification handler
func NewEmailVerificationHandler(verificationService services.EmailVerificationServiceInterface) *EmailVerificationHandler {
	return &EmailVerificationHandler{
		verificationService: verificationService,
	}
}

// VerifyEmail handles GET /api/v1/auth/verify
// @Summary Verify email address
// @Description Verify a user's email address using a verification token
// @Tags auth
// @Produce json
// @Param token query string true "Verification token"
// @Success 200 {object} models.VerifyEmailResponse
// @Failure 400 {object} models.ErrorResponse "Invalid request"
// @Failure 401 {object} models.ErrorResponse "Invalid or expired token"
// @Failure 500 {object} models.ErrorResponse "Internal server error"
// @Router /api/v1/auth/verify [get]
func (h *EmailVerificationHandler) VerifyEmail(c *gin.Context) {
	// Get token from query parameter
	token := c.Query("token")
	if token == "" {
		apierror.WriteResponse(c, http.StatusBadRequest, models.ErrorResponse{
			Error:   "invalid_request",
			Message: "Verification token is required",
		})
		return
	}

	req := models.VerifyEmailRequest{Token: token}

	// Verify email
	err := h.verificationService.VerifyEmail(req.Token)
	if err != nil {
		// Handle different error types
		if errors.Is(err, services.ErrInvalidVerificationToken) {
			apierror.WriteResponse(c, http.StatusUnauthorized, models.ErrorResponse{
				Error:   "invalid_token",
				Message: "Invalid or expired verification token",
			})
			return
		}

		// Internal server error
		apierror.Internal(c, err, "An error occurred while verifying your email")
		return
	}

	// Return success response
	c.JSON(http.StatusOK, models.VerifyEmailResponse{
		Message: "Your email has been successfully verified. You can now access all features.",
	})
}

// ResendVerificationEmail handles POST /api/v1/auth/resend-verification
// @Summary Resend verification email
// @Description Resend the verification email to a user
// @Tags auth
// @Accept json
// @Produce json
// @Param request body models.ResendVerificationRequest true "Email address"
// @Success 200 {object} models.ResendVerificationResponse
// @Failure 400 {object} models.ErrorResponse "Invalid request or already verified"
// @Failure 500 {object} models.ErrorResponse "Internal server error"
// @Router /api/v1/auth/resend-verification [post]
func (h *EmailVerificationHandler) ResendVerificationEmail(c *gin.Context) {
	var req models.ResendVerificationRequest

	// Bind and validate request
	if err := c.ShouldBindJSON(&req); err != nil {
		apierror.BindError(c, err)
		return
	}

	// Resend verification email
	err := h.verificationService.ResendVerificationEmail(req.Email)
	if err != nil {
		// Handle different error types
		if errors.Is(err, services.ErrEmailAlreadyVerified) {
			apierror.WriteResponse(c, http.StatusBadRequest, models.ErrorResponse{
				Error:   "already_verified",
				Message: "Email address is already verified",
			})
			return
		}

		// For security, don't reveal if email exists or internal errors
		// Always return success
		c.JSON(http.StatusOK, models.ResendVerificationResponse{
			Message: "If an account with that email exists and is unverified, a verification email has been sent.",
		})
		return
	}

	// Return success response
	c.JSON(http.StatusOK, models.ResendVerificationResponse{
		Message: "If an account with that email exists and is unverified, a verification email has been sent.",
	})
}
