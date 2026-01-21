package middleware

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"

	"github.com/javaknight1/servicepro/backend/config"
	"github.com/javaknight1/servicepro/backend/internal/models"
)

// RateLimiter handles rate limiting using Redis
type RateLimiter struct {
	redis  *redis.Client
	config *config.AuthConfig
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter(redisClient *redis.Client, cfg *config.AuthConfig) *RateLimiter {
	return &RateLimiter{
		redis:  redisClient,
		config: cfg,
	}
}

// LoginRateLimit implements rate limiting for login endpoint
func (rl *RateLimiter) LoginRateLimit() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get IP address or email for rate limiting
		ip := c.ClientIP()

		// Try to get email from request body for more accurate rate limiting
		var loginReq models.LoginRequest
		if err := c.ShouldBindJSON(&loginReq); err == nil {
			// Use email for rate limiting if available
			ip = loginReq.Email
			// Re-bind the body for the next handler
			c.Set("login_request", loginReq)
		}

		key := fmt.Sprintf("ratelimit:login:%s", ip)
		ctx := context.Background()

		// Increment attempt count
		count, err := rl.redis.Incr(ctx, key).Result()
		if err != nil {
			// If Redis fails, allow the request but log the error
			c.Next()
			return
		}

		// Set expiry on first attempt
		if count == 1 {
			rl.redis.Expire(ctx, key, rl.config.RateLimitWindow)
		}

		// Check if rate limit exceeded
		if count > int64(rl.config.RateLimitAttempts) {
			// Get TTL to inform user when they can retry
			ttl, _ := rl.redis.TTL(ctx, key).Result()

			c.JSON(http.StatusTooManyRequests, models.ErrorResponse{
				Error:   "rate_limit_exceeded",
				Message: fmt.Sprintf("Too many login attempts. Please try again in %v", ttl.Round(time.Second)),
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// ClearLoginAttempts clears the rate limit for a successful login
func (rl *RateLimiter) ClearLoginAttempts(identifier string) error {
	key := fmt.Sprintf("ratelimit:login:%s", identifier)
	ctx := context.Background()
	return rl.redis.Del(ctx, key).Err()
}

// PasswordResetRateLimit implements rate limiting for password reset request endpoint
// Limit: 3 requests per 15 minutes per IP/email
func (rl *RateLimiter) PasswordResetRateLimit() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get IP address
		ip := c.ClientIP()

		// Try to get email from request body for more accurate rate limiting
		var resetReq models.PasswordResetRequestRequest
		if err := c.ShouldBindJSON(&resetReq); err == nil {
			// Use email for rate limiting if available
			ip = resetReq.Email
			// Re-bind the body for the next handler
			c.Set("reset_request", resetReq)
		}

		key := fmt.Sprintf("ratelimit:password_reset:%s", ip)
		ctx := context.Background()

		// Increment attempt count
		count, err := rl.redis.Incr(ctx, key).Result()
		if err != nil {
			// If Redis fails, allow the request but log the error
			c.Next()
			return
		}

		// Set expiry on first attempt (15 minutes)
		if count == 1 {
			rl.redis.Expire(ctx, key, time.Minute*15)
		}

		// Check if rate limit exceeded (3 attempts per 15 minutes)
		maxAttempts := int64(3)
		if count > maxAttempts {
			// Get TTL to inform user when they can retry
			ttl, _ := rl.redis.TTL(ctx, key).Result()

			c.JSON(http.StatusTooManyRequests, models.ErrorResponse{
				Error:   "rate_limit_exceeded",
				Message: fmt.Sprintf("Too many password reset requests. Please try again in %v", ttl.Round(time.Second)),
			})
			c.Abort()
			return
		}

		c.Next()
	}
}
