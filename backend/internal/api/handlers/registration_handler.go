package handlers

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/javaknight1/servicepro/backend/internal/models"
	"github.com/javaknight1/servicepro/backend/internal/services"
	"github.com/javaknight1/servicepro/backend/pkg/auth"
)

// RegistrationHandler handles user registration endpoints
type RegistrationHandler struct {
	registrationService services.RegistrationServiceInterface
}

// NewRegistrationHandler creates a new registration handler
func NewRegistrationHandler(registrationService services.RegistrationServiceInterface) *RegistrationHandler {
	return &RegistrationHandler{
		registrationService: registrationService,
	}
}

// Register handles POST /api/v1/auth/register
// @Summary User registration
// @Description Register a new user account
// @Tags auth
// @Accept json
// @Produce json
// @Param request body models.RegisterRequest true "Registration details"
// @Success 201 {object} models.RegisterResponse
// @Failure 400 {object} models.ErrorResponse "Invalid request or validation error"
// @Failure 409 {object} models.ErrorResponse "Email already exists"
// @Failure 500 {object} models.ErrorResponse "Internal server error"
// @Router /api/v1/auth/register [post]
func (h *RegistrationHandler) Register(c *gin.Context) {
	var req models.RegisterRequest

	// Bind and validate request
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "invalid_request",
			Message: "Invalid email or password format",
		})
		return
	}

	// Attempt registration
	response, err := h.registrationService.Register(req.Email, req.Password)
	if err != nil {
		// Handle different error types
		switch {
		case errors.Is(err, auth.ErrInvalidEmailFormat):
			c.JSON(http.StatusBadRequest, models.ErrorResponse{
				Error:   "invalid_email",
				Message: "Invalid email format",
			})
			return

		case errors.Is(err, services.ErrWeakPassword):
			c.JSON(http.StatusBadRequest, models.ErrorResponse{
				Error:   "weak_password",
				Message: "Password must be at least 8 characters with uppercase, lowercase, number, and special character",
			})
			return

		case errors.Is(err, auth.ErrPasswordTooShort):
			c.JSON(http.StatusBadRequest, models.ErrorResponse{
				Error:   "weak_password",
				Message: err.Error(),
			})
			return

		case errors.Is(err, auth.ErrPasswordNoUppercase),
			errors.Is(err, auth.ErrPasswordNoLowercase),
			errors.Is(err, auth.ErrPasswordNoNumber),
			errors.Is(err, auth.ErrPasswordNoSpecial):
			c.JSON(http.StatusBadRequest, models.ErrorResponse{
				Error:   "weak_password",
				Message: err.Error(),
			})
			return

		case errors.Is(err, services.ErrEmailAlreadyExists):
			c.JSON(http.StatusConflict, models.ErrorResponse{
				Error:   "email_exists",
				Message: "An account with this email already exists",
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

	// Return success response with 201 Created
	c.JSON(http.StatusCreated, response)
}
