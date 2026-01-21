package middleware

import (
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// SimpleRateLimiter implements an in-memory token bucket rate limiter
// Use this for development/testing. For production, use the Redis-based RateLimiter.
type SimpleRateLimiter struct {
	requests map[string]*bucket
	mu       sync.RWMutex
	rate     int           // requests per window
	window   time.Duration // time window
	cleanup  time.Duration // cleanup interval
}

// bucket represents a token bucket for a specific client
type bucket struct {
	tokens     int
	lastRefill time.Time
	mu         sync.Mutex
}

// NewSimpleRateLimiter creates a new in-memory rate limiter
func NewSimpleRateLimiter(rate int, window time.Duration) *SimpleRateLimiter {
	rl := &SimpleRateLimiter{
		requests: make(map[string]*bucket),
		rate:     rate,
		window:   window,
		cleanup:  time.Minute * 10, // cleanup every 10 minutes
	}

	// Start cleanup routine
	go rl.cleanupRoutine()

	return rl
}

// RateLimit middleware limits requests per IP address
func (rl *SimpleRateLimiter) RateLimit() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get client identifier (IP address)
		clientID := c.ClientIP()

		// Check if request is allowed
		allowed, remaining, resetTime := rl.allowRequest(clientID)

		// Set rate limit headers
		c.Header("X-RateLimit-Limit", fmt.Sprintf("%d", rl.rate))
		c.Header("X-RateLimit-Remaining", fmt.Sprintf("%d", remaining))
		c.Header("X-RateLimit-Reset", fmt.Sprintf("%d", resetTime.Unix()))

		if !allowed {
			c.Header("Retry-After", fmt.Sprintf("%d", int(time.Until(resetTime).Seconds())))
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error":       "Rate limit exceeded",
				"message":     fmt.Sprintf("Too many requests. Limit: %d requests per %v", rl.rate, rl.window),
				"retry_after": int(time.Until(resetTime).Seconds()),
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// allowRequest checks if a request is allowed for the given client
func (rl *SimpleRateLimiter) allowRequest(clientID string) (bool, int, time.Time) {
	rl.mu.Lock()
	b, exists := rl.requests[clientID]
	if !exists {
		b = &bucket{
			tokens:     rl.rate,
			lastRefill: time.Now(),
		}
		rl.requests[clientID] = b
	}
	rl.mu.Unlock()

	b.mu.Lock()
	defer b.mu.Unlock()

	// Refill tokens if window has passed
	now := time.Now()
	if now.Sub(b.lastRefill) >= rl.window {
		b.tokens = rl.rate
		b.lastRefill = now
	}

	// Check if tokens are available
	if b.tokens > 0 {
		b.tokens--
		resetTime := b.lastRefill.Add(rl.window)
		return true, b.tokens, resetTime
	}

	resetTime := b.lastRefill.Add(rl.window)
	return false, 0, resetTime
}

// cleanupRoutine removes old bucket entries
func (rl *SimpleRateLimiter) cleanupRoutine() {
	ticker := time.NewTicker(rl.cleanup)
	defer ticker.Stop()

	for range ticker.C {
		rl.mu.Lock()
		now := time.Now()
		for clientID, b := range rl.requests {
			b.mu.Lock()
			// Remove buckets that haven't been used in 2x the window time
			if now.Sub(b.lastRefill) > rl.window*2 {
				delete(rl.requests, clientID)
			}
			b.mu.Unlock()
		}
		rl.mu.Unlock()
	}
}

// SimpleUserRateLimiter limits requests per authenticated user using in-memory storage
type SimpleUserRateLimiter struct {
	*SimpleRateLimiter
}

// NewSimpleUserRateLimiter creates a rate limiter based on user ID
func NewSimpleUserRateLimiter(rate int, window time.Duration) *SimpleUserRateLimiter {
	return &SimpleUserRateLimiter{
		SimpleRateLimiter: NewSimpleRateLimiter(rate, window),
	}
}

// RateLimit middleware for authenticated users
func (url *SimpleUserRateLimiter) RateLimit() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get user ID from context (set by auth middleware)
		userID, exists := c.Get("userID")
		if !exists {
			// Fall back to IP-based limiting for unauthenticated requests
			url.SimpleRateLimiter.RateLimit()(c)
			return
		}

		clientID := fmt.Sprintf("user:%v", userID)

		allowed, remaining, resetTime := url.allowRequest(clientID)

		c.Header("X-RateLimit-Limit", fmt.Sprintf("%d", url.rate))
		c.Header("X-RateLimit-Remaining", fmt.Sprintf("%d", remaining))
		c.Header("X-RateLimit-Reset", fmt.Sprintf("%d", resetTime.Unix()))

		if !allowed {
			c.Header("Retry-After", fmt.Sprintf("%d", int(time.Until(resetTime).Seconds())))
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error":       "Rate limit exceeded",
				"message":     fmt.Sprintf("Too many requests. Limit: %d requests per %v", url.rate, url.window),
				"retry_after": int(time.Until(resetTime).Seconds()),
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// SimpleEndpointRateLimiter allows different limits for different endpoints using in-memory storage
type SimpleEndpointRateLimiter struct {
	limiters map[string]*SimpleRateLimiter
	mu       sync.RWMutex
}

// NewSimpleEndpointRateLimiter creates an endpoint-specific rate limiter
func NewSimpleEndpointRateLimiter() *SimpleEndpointRateLimiter {
	return &SimpleEndpointRateLimiter{
		limiters: make(map[string]*SimpleRateLimiter),
	}
}

// AddLimit adds a rate limit for a specific endpoint pattern
func (erl *SimpleEndpointRateLimiter) AddLimit(pattern string, rate int, window time.Duration) {
	erl.mu.Lock()
	defer erl.mu.Unlock()
	erl.limiters[pattern] = NewSimpleRateLimiter(rate, window)
}

// RateLimit middleware for endpoint-specific limiting
func (erl *SimpleEndpointRateLimiter) RateLimit() gin.HandlerFunc {
	return func(c *gin.Context) {
		path := c.Request.URL.Path

		erl.mu.RLock()
		limiter, exists := erl.limiters[path]
		erl.mu.RUnlock()

		if exists {
			limiter.RateLimit()(c)
		} else {
			c.Next()
		}
	}
}

// DefaultSimpleRateLimiters provides pre-configured in-memory rate limiters for development
var (
	// GlobalSimpleRateLimiter: 100 requests per minute per IP
	GlobalSimpleRateLimiter = NewSimpleRateLimiter(100, time.Minute)

	// AuthSimpleRateLimiter: 5 login attempts per 15 minutes
	AuthSimpleRateLimiter = NewSimpleRateLimiter(5, time.Minute*15)

	// APISimpleRateLimiter: 60 requests per minute per authenticated user
	APISimpleRateLimiter = NewSimpleUserRateLimiter(60, time.Minute)

	// CreateSimpleRateLimiter: 10 creates per minute (for POST endpoints)
	CreateSimpleRateLimiter = NewSimpleRateLimiter(10, time.Minute)
)
