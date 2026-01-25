package middleware

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"

	"github.com/javaknight1/servicepro/backend/config"
)

// RateLimitTier represents the tier of rate limiting to apply
type RateLimitTier string

const (
	// TierAnonymous is for unauthenticated requests (by IP)
	TierAnonymous RateLimitTier = "anonymous"
	// TierAuthenticated is for authenticated users
	TierAuthenticated RateLimitTier = "authenticated"
	// TierAdmin is for admin-only endpoints
	TierAdmin RateLimitTier = "admin"
)

// APIRateLimiter provides comprehensive rate limiting for all API endpoints
type APIRateLimiter struct {
	redis    *redis.Client
	config   *config.RateLimitConfig
	fallback *inMemoryRateLimiter
	mu       sync.RWMutex
}

// inMemoryRateLimiter is a fallback when Redis is unavailable
type inMemoryRateLimiter struct {
	buckets map[string]*tokenBucket
	mu      sync.RWMutex
}

// tokenBucket implements the token bucket algorithm for rate limiting
type tokenBucket struct {
	tokens     float64
	maxTokens  float64
	lastRefill time.Time
	refillRate float64 // tokens per second
	mu         sync.Mutex
}

// NewAPIRateLimiter creates a new API rate limiter
func NewAPIRateLimiter(redisClient *redis.Client, cfg *config.RateLimitConfig) *APIRateLimiter {
	limiter := &APIRateLimiter{
		redis:  redisClient,
		config: cfg,
		fallback: &inMemoryRateLimiter{
			buckets: make(map[string]*tokenBucket),
		},
	}

	// Start cleanup routine for in-memory fallback
	go limiter.cleanupRoutine()

	return limiter
}

// RateLimit returns middleware that applies rate limiting based on the tier
func (rl *APIRateLimiter) RateLimit(tier RateLimitTier) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Check if rate limiting is enabled
		if !rl.config.Enabled {
			c.Next()
			return
		}

		// Check if path is excluded
		path := c.Request.URL.Path
		for _, excluded := range rl.config.ExcludedPaths {
			if strings.HasPrefix(path, excluded) {
				c.Next()
				return
			}
		}

		// Determine the identifier and limit based on tier
		identifier, limit := rl.getIdentifierAndLimit(c, tier)

		// Apply burst multiplier
		burstLimit := int(float64(limit) * rl.config.BurstMultiplier)

		// Check rate limit
		allowed, remaining, resetTime, err := rl.checkRateLimit(c.Request.Context(), identifier, limit, burstLimit)

		// Set rate limit headers
		c.Header("X-RateLimit-Limit", strconv.Itoa(limit))
		c.Header("X-RateLimit-Remaining", strconv.Itoa(remaining))
		c.Header("X-RateLimit-Reset", strconv.FormatInt(resetTime.Unix(), 10))

		if err != nil {
			// Log error but allow request if rate limiting fails
			// In production, you might want to fail closed instead
			c.Next()
			return
		}

		if !allowed {
			retryAfter := int(time.Until(resetTime).Seconds())
			if retryAfter < 1 {
				retryAfter = 1
			}
			c.Header("Retry-After", strconv.Itoa(retryAfter))

			c.JSON(http.StatusTooManyRequests, gin.H{
				"error":       "rate_limit_exceeded",
				"message":     fmt.Sprintf("Too many requests. Limit: %d requests per %v", limit, rl.config.Window),
				"retry_after": retryAfter,
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// RateLimitAnonymous applies rate limiting for anonymous/unauthenticated requests
func (rl *APIRateLimiter) RateLimitAnonymous() gin.HandlerFunc {
	return rl.RateLimit(TierAnonymous)
}

// RateLimitAuthenticated applies rate limiting for authenticated users
func (rl *APIRateLimiter) RateLimitAuthenticated() gin.HandlerFunc {
	return rl.RateLimit(TierAuthenticated)
}

// RateLimitAdmin applies stricter rate limiting for admin endpoints
func (rl *APIRateLimiter) RateLimitAdmin() gin.HandlerFunc {
	return rl.RateLimit(TierAdmin)
}

// DynamicRateLimit applies rate limiting based on context (auto-detects tier)
func (rl *APIRateLimiter) DynamicRateLimit() gin.HandlerFunc {
	return func(c *gin.Context) {
		if !rl.config.Enabled {
			c.Next()
			return
		}

		// Check if path is excluded
		path := c.Request.URL.Path
		for _, excluded := range rl.config.ExcludedPaths {
			if strings.HasPrefix(path, excluded) {
				c.Next()
				return
			}
		}

		// Auto-detect tier based on authentication status
		tier := TierAnonymous
		if userID, exists := c.Get("userID"); exists && userID != nil {
			tier = TierAuthenticated

			// Check if user is admin (if role info is available)
			if role, exists := c.Get("userRole"); exists {
				if roleStr, ok := role.(string); ok {
					if roleStr == "admin" || roleStr == "super_admin" {
						// Admin users still get authenticated limits, not admin endpoint limits
						tier = TierAuthenticated
					}
				}
			}
		}

		// Apply rate limiting
		rl.RateLimit(tier)(c)
	}
}

// getIdentifierAndLimit returns the rate limit identifier and limit based on tier
func (rl *APIRateLimiter) getIdentifierAndLimit(c *gin.Context, tier RateLimitTier) (string, int) {
	switch tier {
	case TierAuthenticated:
		// Use user ID if available
		if userID, exists := c.Get("userID"); exists {
			if uid, ok := userID.(uuid.UUID); ok {
				return fmt.Sprintf("ratelimit:user:%s", uid.String()), rl.config.AuthenticatedLimit
			}
			if uidStr, ok := userID.(string); ok {
				return fmt.Sprintf("ratelimit:user:%s", uidStr), rl.config.AuthenticatedLimit
			}
		}
		// Fall back to IP if no user ID
		return fmt.Sprintf("ratelimit:ip:%s", c.ClientIP()), rl.config.AnonymousLimit

	case TierAdmin:
		// Admin endpoints use lower limits to prevent abuse
		if userID, exists := c.Get("userID"); exists {
			if uid, ok := userID.(uuid.UUID); ok {
				return fmt.Sprintf("ratelimit:admin:%s", uid.String()), rl.config.AdminLimit
			}
			if uidStr, ok := userID.(string); ok {
				return fmt.Sprintf("ratelimit:admin:%s", uidStr), rl.config.AdminLimit
			}
		}
		return fmt.Sprintf("ratelimit:admin:ip:%s", c.ClientIP()), rl.config.AdminLimit

	default: // TierAnonymous
		return fmt.Sprintf("ratelimit:ip:%s", c.ClientIP()), rl.config.AnonymousLimit
	}
}

// checkRateLimit checks if a request is allowed under the rate limit
func (rl *APIRateLimiter) checkRateLimit(ctx context.Context, identifier string, limit, burstLimit int) (bool, int, time.Time, error) {
	if rl.config.UseRedis && rl.redis != nil {
		return rl.checkRateLimitRedis(ctx, identifier, limit, burstLimit)
	}
	return rl.checkRateLimitInMemory(identifier, limit, burstLimit)
}

// checkRateLimitRedis uses Redis for distributed rate limiting
func (rl *APIRateLimiter) checkRateLimitRedis(ctx context.Context, identifier string, limit, burstLimit int) (bool, int, time.Time, error) {
	now := time.Now()

	// Use sliding window algorithm with Redis
	pipe := rl.redis.Pipeline()

	// Remove old entries outside the window
	cutoff := now.Add(-rl.config.Window).UnixMicro()
	pipe.ZRemRangeByScore(ctx, identifier, "0", strconv.FormatInt(cutoff, 10))

	// Count current requests in window
	countCmd := pipe.ZCard(ctx, identifier)

	// Add current request
	pipe.ZAdd(ctx, identifier, redis.Z{
		Score:  float64(now.UnixMicro()),
		Member: fmt.Sprintf("%d:%d", now.UnixMicro(), now.UnixNano()%1000000),
	})

	// Set expiry on the key
	pipe.Expire(ctx, identifier, rl.config.Window+time.Second)

	_, err := pipe.Exec(ctx)
	if err != nil {
		// Fall back to in-memory on Redis error
		return rl.checkRateLimitInMemory(identifier, limit, burstLimit)
	}

	count := int(countCmd.Val())
	remaining := burstLimit - count - 1
	if remaining < 0 {
		remaining = 0
	}

	resetTime := now.Add(rl.config.Window)

	// Allow up to burst limit
	if count >= burstLimit {
		return false, 0, resetTime, nil
	}

	return true, remaining, resetTime, nil
}

// checkRateLimitInMemory uses in-memory token bucket for rate limiting
func (rl *APIRateLimiter) checkRateLimitInMemory(identifier string, limit, burstLimit int) (bool, int, time.Time, error) {
	rl.fallback.mu.Lock()
	bucket, exists := rl.fallback.buckets[identifier]
	if !exists {
		bucket = &tokenBucket{
			tokens:     float64(burstLimit),
			maxTokens:  float64(burstLimit),
			lastRefill: time.Now(),
			refillRate: float64(limit) / rl.config.Window.Seconds(),
		}
		rl.fallback.buckets[identifier] = bucket
	}
	rl.fallback.mu.Unlock()

	bucket.mu.Lock()
	defer bucket.mu.Unlock()

	// Refill tokens based on time elapsed
	now := time.Now()
	elapsed := now.Sub(bucket.lastRefill)
	bucket.tokens += elapsed.Seconds() * bucket.refillRate
	if bucket.tokens > bucket.maxTokens {
		bucket.tokens = bucket.maxTokens
	}
	bucket.lastRefill = now

	// Check if we have tokens available
	if bucket.tokens >= 1 {
		bucket.tokens--
		remaining := int(bucket.tokens)
		resetTime := now.Add(time.Duration(float64(time.Second) / bucket.refillRate))
		return true, remaining, resetTime, nil
	}

	// Calculate when the next token will be available
	resetTime := now.Add(time.Duration(float64(time.Second) / bucket.refillRate))
	return false, 0, resetTime, nil
}

// cleanupRoutine periodically removes old bucket entries from memory
func (rl *APIRateLimiter) cleanupRoutine() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		rl.fallback.mu.Lock()
		now := time.Now()
		for id, bucket := range rl.fallback.buckets {
			bucket.mu.Lock()
			// Remove buckets that haven't been used in 2x the window
			if now.Sub(bucket.lastRefill) > rl.config.Window*2 {
				delete(rl.fallback.buckets, id)
			}
			bucket.mu.Unlock()
		}
		rl.fallback.mu.Unlock()
	}
}

// GetRateLimitStats returns current rate limit statistics (for monitoring)
func (rl *APIRateLimiter) GetRateLimitStats() map[string]interface{} {
	rl.fallback.mu.RLock()
	defer rl.fallback.mu.RUnlock()

	return map[string]interface{}{
		"enabled":                rl.config.Enabled,
		"use_redis":              rl.config.UseRedis,
		"anonymous_limit":        rl.config.AnonymousLimit,
		"authenticated_limit":    rl.config.AuthenticatedLimit,
		"admin_limit":            rl.config.AdminLimit,
		"window":                 rl.config.Window.String(),
		"in_memory_bucket_count": len(rl.fallback.buckets),
	}
}
