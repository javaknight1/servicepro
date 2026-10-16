package middleware

import (
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/javaknight1/servicepro/backend/internal/api/apierror"
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
		clientID := ClientIP(c)

		// Check if request is allowed
		allowed, remaining, resetTime := rl.allowRequest(clientID)

		// Set rate limit headers
		c.Header("X-RateLimit-Limit", fmt.Sprintf("%d", rl.rate))
		c.Header("X-RateLimit-Remaining", fmt.Sprintf("%d", remaining))
		c.Header("X-RateLimit-Reset", fmt.Sprintf("%d", resetTime.Unix()))

		if !allowed {
			c.Header("Retry-After", fmt.Sprintf("%d", int(time.Until(resetTime).Seconds())))
			apierror.Write(c, http.StatusTooManyRequests, "rate_limit_exceeded", fmt.Sprintf("Too many requests. Limit: %d requests per %v", rl.rate, rl.window))
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
