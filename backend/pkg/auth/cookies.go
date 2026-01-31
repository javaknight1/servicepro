package auth

import (
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/javaknight1/servicepro/backend/config"
)

// ErrNoCookie is returned when a cookie is not found
var ErrNoCookie = errors.New("cookie not found")

// CookieManager handles HTTP cookie operations for JWT tokens
type CookieManager struct {
	config            *config.CookieConfig
	accessExpiration  time.Duration
	refreshExpiration time.Duration
}

// NewCookieManager creates a new CookieManager
func NewCookieManager(cfg *config.CookieConfig, accessExp, refreshExp time.Duration) *CookieManager {
	return &CookieManager{
		config:            cfg,
		accessExpiration:  accessExp,
		refreshExpiration: refreshExp,
	}
}

// getSameSite converts string config to http.SameSite
func (cm *CookieManager) getSameSite() http.SameSite {
	switch cm.config.SameSite {
	case "Strict":
		return http.SameSiteStrictMode
	case "None":
		return http.SameSiteNoneMode
	case "Lax":
		return http.SameSiteLaxMode
	default:
		return http.SameSiteLaxMode
	}
}

// SetAccessToken sets the access token as an httpOnly cookie
func (cm *CookieManager) SetAccessToken(c *gin.Context, token string) {
	maxAge := int(cm.accessExpiration.Seconds())
	cm.setCookie(c, cm.config.AccessTokenName, token, maxAge, "/")
}

// SetRefreshToken sets the refresh token as an httpOnly cookie
// The refresh token is restricted to the auth path for security
func (cm *CookieManager) SetRefreshToken(c *gin.Context, token string) {
	maxAge := int(cm.refreshExpiration.Seconds())
	cm.setCookie(c, cm.config.RefreshTokenName, token, maxAge, cm.config.RefreshTokenPath)
}

// SetTokens sets both access and refresh tokens as httpOnly cookies
func (cm *CookieManager) SetTokens(c *gin.Context, accessToken, refreshToken string) {
	cm.SetAccessToken(c, accessToken)
	cm.SetRefreshToken(c, refreshToken)
}

// ClearTokens removes both access and refresh token cookies
func (cm *CookieManager) ClearTokens(c *gin.Context) {
	// Clear access token
	cm.clearCookie(c, cm.config.AccessTokenName, "/")
	// Clear refresh token
	cm.clearCookie(c, cm.config.RefreshTokenName, cm.config.RefreshTokenPath)
}

// GetAccessToken retrieves the access token from cookies
func (cm *CookieManager) GetAccessToken(c *gin.Context) (string, error) {
	token, err := c.Cookie(cm.config.AccessTokenName)
	if err != nil {
		return "", ErrNoCookie
	}
	return token, nil
}

// GetRefreshToken retrieves the refresh token from cookies
func (cm *CookieManager) GetRefreshToken(c *gin.Context) (string, error) {
	token, err := c.Cookie(cm.config.RefreshTokenName)
	if err != nil {
		return "", ErrNoCookie
	}
	return token, nil
}

// setCookie is a helper to set a cookie with consistent settings
func (cm *CookieManager) setCookie(c *gin.Context, name, value string, maxAge int, path string) {
	c.SetSameSite(cm.getSameSite())
	c.SetCookie(
		name,
		value,
		maxAge,
		path,
		cm.config.Domain,
		cm.config.Secure,
		true, // httpOnly - prevents JavaScript access
	)
}

// clearCookie removes a cookie by setting maxAge to -1
func (cm *CookieManager) clearCookie(c *gin.Context, name, path string) {
	c.SetSameSite(cm.getSameSite())
	c.SetCookie(
		name,
		"",
		-1,
		path,
		cm.config.Domain,
		cm.config.Secure,
		true, // httpOnly
	)
}
