package middleware

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"strings"
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

// Login throttling (F-12). Failures are counted per (account, IP) with
// exponential backoff, so an attacker can't lock a victim out from their own
// address; a much higher per-account threshold with a short cool-down limits
// distributed guessing; a per-IP attempt cap limits spraying. Counters are
// keyed on the submitted email whether or not the account exists, and every
// throttle returns the same 429, so throttling reveals nothing.
const (
	loginPairFreeFailures    = 3                // failures per (account, IP) before backoff starts
	loginPairMaxBackoff      = 15 * time.Minute // backoff cap per (account, IP)
	loginAccountThreshold    = 50               // failures per account (all IPs) in the window
	loginAccountCooldown     = 5 * time.Minute
	loginIPAttemptsPerWindow = 100 // attempts per IP (all accounts) in the window
	loginFailureWindow       = 15 * time.Minute
)

func loginEmailKey(email string) string {
	sum := sha256.Sum256([]byte(strings.ToLower(strings.TrimSpace(email))))
	return hex.EncodeToString(sum[:8])
}

func loginPairKey(email, ip string) string {
	return fmt.Sprintf("login:pair:%s:%s", loginEmailKey(email), ip)
}

// loginBackoff returns how long the (account, IP) pair must wait after its
// n-th consecutive failure.
func loginBackoff(failures int64) time.Duration {
	if failures < loginPairFreeFailures {
		return 0
	}
	d := time.Second << uint(failures-loginPairFreeFailures) // 1s, 2s, 4s, ...
	if d <= 0 || d > loginPairMaxBackoff {
		return loginPairMaxBackoff
	}
	return d
}

// LoginRateLimit rejects login attempts that are currently throttled.
func (rl *RateLimiter) LoginRateLimit() gin.HandlerFunc {
	return func(c *gin.Context) {
		ip := ClientIP(c)
		var loginReq models.LoginRequest
		email := ""
		if err := c.ShouldBindJSON(&loginReq); err == nil {
			email = loginReq.Email
			// Re-bind the body for the next handler
			c.Set("login_request", loginReq)
		}
		ctx := c.Request.Context()

		throttled := func() {
			c.AbortWithStatusJSON(http.StatusTooManyRequests, models.ErrorResponse{
				Error:   "rate_limit_exceeded",
				Message: "Too many login attempts. Please wait a moment and try again.",
			})
		}

		// Per-IP attempt cap (all accounts).
		ipKey := "login:ip:" + ip
		count, err := rl.redis.Incr(ctx, ipKey).Result()
		if err == nil {
			if count == 1 {
				rl.redis.Expire(ctx, ipKey, loginFailureWindow)
			}
			if count > loginIPAttemptsPerWindow {
				throttled()
				return
			}
		}

		if email != "" {
			// (account, IP) backoff and per-account cool-down.
			if n, _ := rl.redis.Exists(ctx, loginPairKey(email, ip)+":block", "login:acct:"+loginEmailKey(email)+":cooldown").Result(); n > 0 {
				throttled()
				return
			}
		}

		c.Next()
	}
}

// RecordLoginFailure counts a failed login for the (account, IP) pair and the
// account, applying backoff / cool-down as thresholds are crossed.
func (rl *RateLimiter) RecordLoginFailure(email, ip string) {
	if email == "" {
		return
	}
	ctx := context.Background()
	pair := loginPairKey(email, ip)
	failures, err := rl.redis.Incr(ctx, pair).Result()
	if err == nil {
		rl.redis.Expire(ctx, pair, loginFailureWindow)
		if d := loginBackoff(failures); d > 0 {
			rl.redis.Set(ctx, pair+":block", "1", d)
		}
	}

	acct := "login:acct:" + loginEmailKey(email)
	total, err := rl.redis.Incr(ctx, acct).Result()
	if err == nil {
		if total == 1 {
			rl.redis.Expire(ctx, acct, loginFailureWindow)
		}
		if total >= loginAccountThreshold {
			rl.redis.Set(ctx, acct+":cooldown", "1", loginAccountCooldown)
			rl.redis.Del(ctx, acct)
		}
	}
}

// ClearLoginAttempts resets the (account, IP) backoff after a successful login.
func (rl *RateLimiter) ClearLoginAttempts(email, ip string) error {
	ctx := context.Background()
	pair := loginPairKey(email, ip)
	return rl.redis.Del(ctx, pair, pair+":block").Err()
}

// PasswordResetRateLimit implements rate limiting for password reset request endpoint
// Limit: 3 requests per 15 minutes per IP/email
func (rl *RateLimiter) PasswordResetRateLimit() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get IP address
		ip := ClientIP(c)

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
