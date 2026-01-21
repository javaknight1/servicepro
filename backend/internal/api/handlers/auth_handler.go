package handlers

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/javaknight1/servicepro/backend/internal/api/middleware"
	"github.com/javaknight1/servicepro/backend/internal/models"
	"github.com/javaknight1/servicepro/backend/internal/services"
)

// AuthHandler handles authentication endpoints
type AuthHandler struct {
	authService services.AuthServiceInterface
	rateLimiter middleware.RateLimiterInterface
}

// NewAuthHandler creates a new auth handler
func NewAuthHandler(authService services.AuthServiceInterface, rateLimiter middleware.RateLimiterInterface) *AuthHandler {
	return &AuthHandler{
		authService: authService,
		rateLimiter: rateLimiter,
	}
}

// Login handles POST /api/v1/auth/login
// @Summary User login
// @Description Authenticate user and return JWT tokens
// @Tags auth
// @Accept json
// @Produce json
// @Param request body models.LoginRequest true "Login credentials"
// @Success 200 {object} models.LoginResponse
// @Failure 400 {object} models.ErrorResponse "Invalid request"
// @Failure 401 {object} models.ErrorResponse "Invalid credentials"
// @Failure 403 {object} models.ErrorResponse "Account locked"
// @Failure 429 {object} models.ErrorResponse "Rate limit exceeded"
// @Failure 500 {object} models.ErrorResponse "Internal server error"
// @Router /api/v1/auth/login [post]
func (h *AuthHandler) Login(c *gin.Context) {
	var req models.LoginRequest

	// Check if request was already bound by rate limiter
	if cachedReq, exists := c.Get("login_request"); exists {
		req = cachedReq.(models.LoginRequest)
	} else {
		// Bind and validate request
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, models.ErrorResponse{
				Error:   "invalid_request",
				Message: "Invalid email or password format",
			})
			return
		}
	}

	// Attempt login
	response, err := h.authService.Login(req.Email, req.Password)
	if err != nil {
		// Handle different error types
		switch {
		case errors.Is(err, services.ErrInvalidCredentials):
			c.JSON(http.StatusUnauthorized, models.ErrorResponse{
				Error:   "invalid_credentials",
				Message: "Invalid email or password",
			})
			return

		case errors.Is(err, services.ErrAccountLocked):
			c.JSON(http.StatusForbidden, models.ErrorResponse{
				Error:   "account_locked",
				Message: "Account is locked due to too many failed login attempts. Please try again later.",
			})
			return

		case errors.Is(err, services.ErrEmailNotVerified):
			c.JSON(http.StatusForbidden, models.ErrorResponse{
				Error:   "email_not_verified",
				Message: "Please verify your email address before logging in. Check your inbox for the verification link.",
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

	// Clear rate limit on successful login
	_ = h.rateLimiter.ClearLoginAttempts(req.Email)

	// Return success response
	c.JSON(http.StatusOK, response)
}

// RefreshToken handles POST /api/v1/auth/refresh
// @Summary Refresh access token
// @Description Exchange refresh token for new access and refresh tokens
// @Tags auth
// @Accept json
// @Produce json
// @Param request body object{refresh_token=string} true "Refresh token"
// @Success 200 {object} models.LoginResponse
// @Failure 400 {object} models.ErrorResponse "Invalid request"
// @Failure 401 {object} models.ErrorResponse "Invalid or expired refresh token"
// @Failure 403 {object} models.ErrorResponse "Account locked"
// @Failure 500 {object} models.ErrorResponse "Internal server error"
// @Router /api/v1/auth/refresh [post]
func (h *AuthHandler) RefreshToken(c *gin.Context) {
	var req struct {
		RefreshToken string `json:"refresh_token" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "invalid_request",
			Message: "Refresh token is required",
		})
		return
	}

	// Attempt to refresh tokens
	response, err := h.authService.RefreshToken(req.RefreshToken)
	if err != nil {
		switch {
		case errors.Is(err, services.ErrInvalidRefreshToken):
			c.JSON(http.StatusUnauthorized, models.ErrorResponse{
				Error:   "invalid_token",
				Message: "Invalid or expired refresh token",
			})
			return

		case errors.Is(err, services.ErrAccountLocked):
			c.JSON(http.StatusForbidden, models.ErrorResponse{
				Error:   "account_locked",
				Message: "Account is locked. Please contact support.",
			})
			return

		case errors.Is(err, services.ErrEmailNotVerified):
			c.JSON(http.StatusForbidden, models.ErrorResponse{
				Error:   "email_not_verified",
				Message: "Please verify your email address. Check your inbox for the verification link.",
			})
			return

		default:
			c.JSON(http.StatusInternalServerError, models.ErrorResponse{
				Error:   "internal_error",
				Message: "An error occurred while refreshing the token",
			})
			return
		}
	}

	c.JSON(http.StatusOK, response)
}
