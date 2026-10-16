package handlers

import (
	"errors"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/javaknight1/servicepro/backend/internal/api/apierror"
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
// @Failure 403 {object} models.ErrorResponse "Email not verified (only after a correct password)"
// @Failure 429 {object} models.ErrorResponse "Too many attempts (backoff)"
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
			apierror.BindError(c, err)
			return
		}
	}

	ip := middleware.ClientIP(c)

	// Attempt login
	response, err := h.authService.Login(req.Email, req.Password)
	if err != nil {
		// Unknown email, wrong password and locked accounts all get the same
		// response (F-12); "not verified" is only revealed after the correct
		// password was supplied.
		switch {
		case errors.Is(err, services.ErrInvalidCredentials), errors.Is(err, services.ErrAccountLocked):
			h.rateLimiter.RecordLoginFailure(req.Email, ip)
			apierror.WriteResponse(c, http.StatusUnauthorized, models.ErrorResponse{
				Error:   "invalid_credentials",
				Message: "Invalid email or password",
			})
			return

		case errors.Is(err, services.ErrEmailNotVerified):
			apierror.WriteResponse(c, http.StatusForbidden, models.ErrorResponse{
				Error:   "email_not_verified",
				Message: "Please verify your email address before logging in. Check your inbox for the verification link.",
			})
			return

		default:
			// Internal server error
			apierror.Internal(c, err, "An error occurred while processing your request")
			return
		}
	}

	// Clear rate limit on successful login
	_ = h.rateLimiter.ClearLoginAttempts(req.Email, ip)

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
		apierror.WriteResponse(c, http.StatusUnauthorized, models.ErrorResponse{
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
			apierror.WriteResponse(c, http.StatusUnauthorized, models.ErrorResponse{
				Error:   "invalid_token",
				Message: "Invalid or expired refresh token",
			})
			return

		case errors.Is(err, services.ErrAccountLocked):
			h.cookieManager.ClearTokens(c)
			apierror.WriteResponse(c, http.StatusForbidden, models.ErrorResponse{
				Error:   "account_locked",
				Message: "Account is locked. Please contact support.",
			})
			return

		case errors.Is(err, services.ErrEmailNotVerified):
			h.cookieManager.ClearTokens(c)
			apierror.WriteResponse(c, http.StatusForbidden, models.ErrorResponse{
				Error:   "email_not_verified",
				Message: "Please verify your email address. Check your inbox for the verification link.",
			})
			return

		default:
			apierror.Internal(c, err, "An error occurred while refreshing the token")
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
	// Revoke the session server-side (bumps the user's token version, which
	// invalidates every access and refresh token issued so far), then clear
	// the cookies.
	token, _ := h.cookieManager.GetRefreshToken(c)
	if token == "" {
		token, _ = h.cookieManager.GetAccessToken(c)
	}
	if err := h.authService.Logout(token); err != nil {
		log.Printf("[AUTH] Failed to revoke tokens on logout: %v", err)
	}

	// Clear both access and refresh token cookies
	h.cookieManager.ClearTokens(c)

	c.JSON(http.StatusOK, gin.H{
		"message": "Logged out successfully",
	})
}
