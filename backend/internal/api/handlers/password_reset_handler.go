package handlers

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/javaknight1/servicepro/backend/internal/models"
	"github.com/javaknight1/servicepro/backend/internal/services"
	"github.com/javaknight1/servicepro/backend/pkg/auth"
)

// PasswordResetHandler handles password reset endpoints
type PasswordResetHandler struct {
	passwordResetService services.PasswordResetServiceInterface
}

// NewPasswordResetHandler creates a new password reset handler
func NewPasswordResetHandler(passwordResetService services.PasswordResetServiceInterface) *PasswordResetHandler {
	return &PasswordResetHandler{
		passwordResetService: passwordResetService,
	}
}

// RequestPasswordReset handles POST /api/v1/auth/reset-request
// @Summary Request password reset
// @Description Request a password reset token to be sent via email
// @Tags auth
// @Accept json
// @Produce json
// @Param request body models.PasswordResetRequestRequest true "Email address"
// @Success 200 {object} models.PasswordResetRequestResponse
// @Failure 400 {object} models.ErrorResponse "Invalid request"
// @Failure 429 {object} models.ErrorResponse "Rate limit exceeded"
// @Failure 500 {object} models.ErrorResponse "Internal server error"
// @Router /api/v1/auth/reset-request [post]
func (h *PasswordResetHandler) RequestPasswordReset(c *gin.Context) {
	var req models.PasswordResetRequestRequest

	// Bind and validate request
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "invalid_request",
			Message: "Invalid email format",
		})
		return
	}

	// Process password reset request
	err := h.passwordResetService.RequestPasswordReset(req.Email)
	if err != nil {
		// Handle different error types
		if errors.Is(err, auth.ErrInvalidEmailFormat) {
			c.JSON(http.StatusBadRequest, models.ErrorResponse{
				Error:   "invalid_email",
				Message: "Invalid email format",
			})
			return
		}

		// For security, don't reveal if email exists or internal errors
		// Always return success to prevent email enumeration
		c.JSON(http.StatusOK, models.PasswordResetRequestResponse{
			Message: "If an account with that email exists, a password reset link has been sent.",
		})
		return
	}

	// Return success response
	c.JSON(http.StatusOK, models.PasswordResetRequestResponse{
		Message: "If an account with that email exists, a password reset link has been sent.",
	})
}

// ResetPassword handles POST /api/v1/auth/reset-password
// @Summary Reset password
// @Description Reset password using a valid reset token
// @Tags auth
// @Accept json
// @Produce json
// @Param request body models.PasswordResetRequest true "Reset token and new password"
// @Success 200 {object} models.PasswordResetResponse
// @Failure 400 {object} models.ErrorResponse "Invalid request or weak password"
// @Failure 401 {object} models.ErrorResponse "Invalid or expired token"
// @Failure 500 {object} models.ErrorResponse "Internal server error"
// @Router /api/v1/auth/reset-password [post]
func (h *PasswordResetHandler) ResetPassword(c *gin.Context) {
	var req models.PasswordResetRequest

	// Bind and validate request
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "invalid_request",
			Message: "Token and password are required",
		})
		return
	}

	// Process password reset
	err := h.passwordResetService.ResetPassword(req.Token, req.NewPassword)
	if err != nil {
		// Handle different error types
		switch {
		case errors.Is(err, services.ErrInvalidResetToken):
			c.JSON(http.StatusUnauthorized, models.ErrorResponse{
				Error:   "invalid_token",
				Message: "Invalid or expired reset token",
			})
			return

		case errors.Is(err, services.ErrWeakPassword):
			c.JSON(http.StatusBadRequest, models.ErrorResponse{
				Error:   "weak_password",
				Message: "Password must be at least 8 characters with uppercase, lowercase, number, and special character",
			})
			return

		case errors.Is(err, auth.ErrPasswordTooShort),
			errors.Is(err, auth.ErrPasswordNoUppercase),
			errors.Is(err, auth.ErrPasswordNoLowercase),
			errors.Is(err, auth.ErrPasswordNoNumber),
			errors.Is(err, auth.ErrPasswordNoSpecial):
			c.JSON(http.StatusBadRequest, models.ErrorResponse{
				Error:   "weak_password",
				Message: err.Error(),
			})
			return

		default:
			// Internal server error
			c.JSON(http.StatusInternalServerError, models.ErrorResponse{
				Error:   "internal_error",
				Message: "An error occurred while processing your request",
			})
			return
		}
	}

	// Return success response
	c.JSON(http.StatusOK, models.PasswordResetResponse{
		Message: "Password has been successfully reset. You can now log in with your new password.",
	})
}
