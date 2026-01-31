package handlers

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/javaknight1/servicepro/backend/internal/api/middleware"
	"github.com/javaknight1/servicepro/backend/internal/models"
	"github.com/javaknight1/servicepro/backend/internal/services"
	"github.com/javaknight1/servicepro/backend/pkg/auth"
)

// AuthHandler handles authentication endpoints
type AuthHandler struct {
	authService   services.AuthServiceInterface
	rateLimiter   middleware.RateLimiterInterface
	cookieManager *auth.CookieManager
}

// NewAuthHandler creates a new auth handler
func NewAuthHandler(authService services.AuthServiceInterface, rateLimiter middleware.RateLimiterInterface, cookieManager *auth.CookieManager) *AuthHandler {
	return &AuthHandler{
		authService:   authService,
		rateLimiter:   rateLimiter,
		cookieManager: cookieManager,
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

	// Set tokens as httpOnly cookies
	h.cookieManager.SetTokens(c, response.Token, response.RefreshToken)

	// Return success response without tokens in body (they're in cookies)
	c.JSON(http.StatusOK, models.LoginCookieResponse{
		Message:   "Login successful",
		ExpiresIn: response.ExpiresIn,
	})
}

// RefreshToken handles POST /api/v1/auth/refresh
// @Summary Refresh access token
// @Description Exchange refresh token for new access and refresh tokens
// @Tags auth
// @Accept json
// @Produce json
// @Success 200 {object} models.LoginCookieResponse
// @Failure 401 {object} models.ErrorResponse "Invalid or expired refresh token"
// @Failure 403 {object} models.ErrorResponse "Account locked"
// @Failure 500 {object} models.ErrorResponse "Internal server error"
// @Router /api/v1/auth/refresh [post]
func (h *AuthHandler) RefreshToken(c *gin.Context) {
	// Get refresh token from cookie
	refreshToken, err := h.cookieManager.GetRefreshToken(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, models.ErrorResponse{
			Error:   "invalid_request",
			Message: "Refresh token not found",
		})
		return
	}

	// Attempt to refresh tokens
	response, err := h.authService.RefreshToken(refreshToken)
	if err != nil {
		switch {
		case errors.Is(err, services.ErrInvalidRefreshToken):
			// Clear invalid cookies
			h.cookieManager.ClearTokens(c)
			c.JSON(http.StatusUnauthorized, models.ErrorResponse{
				Error:   "invalid_token",
				Message: "Invalid or expired refresh token",
			})
			return

		case errors.Is(err, services.ErrAccountLocked):
			h.cookieManager.ClearTokens(c)
			c.JSON(http.StatusForbidden, models.ErrorResponse{
				Error:   "account_locked",
				Message: "Account is locked. Please contact support.",
			})
			return

		case errors.Is(err, services.ErrEmailNotVerified):
			h.cookieManager.ClearTokens(c)
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

	// Set new tokens as httpOnly cookies
	h.cookieManager.SetTokens(c, response.Token, response.RefreshToken)

	// Return success response without tokens in body
	c.JSON(http.StatusOK, models.LoginCookieResponse{
		Message:   "Token refreshed successfully",
		ExpiresIn: response.ExpiresIn,
	})
}

// Logout handles POST /api/v1/auth/logout
// @Summary User logout
// @Description Clear authentication cookies
// @Tags auth
// @Produce json
// @Success 200 {object} object{message=string}
// @Router /api/v1/auth/logout [post]
func (h *AuthHandler) Logout(c *gin.Context) {
	// Clear both access and refresh token cookies
	h.cookieManager.ClearTokens(c)

	c.JSON(http.StatusOK, gin.H{
		"message": "Logged out successfully",
	})
}
