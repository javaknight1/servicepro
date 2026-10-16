package middleware

import "github.com/gin-gonic/gin"

// RateLimiterInterface defines the interface for rate limiting
type RateLimiterInterface interface {
	LoginRateLimit() gin.HandlerFunc
	PasswordResetRateLimit() gin.HandlerFunc
	RecordLoginFailure(email, ip string)
	ClearLoginAttempts(email, ip string) error
}
