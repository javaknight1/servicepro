package middleware

import (
	"net/http"
	"net/http/httptest"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/javaknight1/servicepro/backend/config"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func TestAPIRateLimiter_RateLimitDisabled(t *testing.T) {
	cfg := &config.RateLimitConfig{
		Enabled:            false,
		AnonymousLimit:     10,
		AuthenticatedLimit: 100,
		Window:             time.Minute,
	}

	limiter := NewAPIRateLimiter(nil, cfg)

	router := gin.New()
	router.GET("/test", limiter.RateLimitAnonymous(), func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	// Make requests - should all pass since rate limiting is disabled
	for i := 0; i < 100; i++ {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/test", nil)
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Request %d: expected 200, got %d", i, w.Code)
		}
	}
}

func TestAPIRateLimiter_AnonymousLimit(t *testing.T) {
	cfg := &config.RateLimitConfig{
		Enabled:            true,
		UseRedis:           false, // Use in-memory
		AnonymousLimit:     5,
		AuthenticatedLimit: 100,
		AdminLimit:         50,
		BurstMultiplier:    1.0, // No burst
		Window:             time.Second * 10,
		ExcludedPaths:      []string{"/health"},
	}

	limiter := NewAPIRateLimiter(nil, cfg)

	router := gin.New()
	router.GET("/test", limiter.RateLimitAnonymous(), func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	// First 5 requests should succeed
	for i := 0; i < 5; i++ {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/test", nil)
		req.Header.Set("X-Forwarded-For", "192.168.1.1")
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Request %d: expected 200, got %d", i+1, w.Code)
		}

		// Check rate limit headers
		limitHeader := w.Header().Get("X-RateLimit-Limit")
		if limitHeader != "5" {
			t.Errorf("Expected X-RateLimit-Limit=5, got %s", limitHeader)
		}

		remainingHeader := w.Header().Get("X-RateLimit-Remaining")
		expectedRemaining := strconv.Itoa(4 - i)
		if remainingHeader != expectedRemaining {
			t.Errorf("Request %d: expected X-RateLimit-Remaining=%s, got %s", i+1, expectedRemaining, remainingHeader)
		}
	}

	// 6th request should be rate limited
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	req.Header.Set("X-Forwarded-For", "192.168.1.1")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusTooManyRequests {
		t.Errorf("Expected 429, got %d", w.Code)
	}

	// Check Retry-After header
	retryAfter := w.Header().Get("Retry-After")
	if retryAfter == "" {
		t.Error("Expected Retry-After header to be set")
	}
}

func TestAPIRateLimiter_AuthenticatedLimit(t *testing.T) {
	cfg := &config.RateLimitConfig{
		Enabled:            true,
		UseRedis:           false,
		AnonymousLimit:     5,
		AuthenticatedLimit: 10,
		AdminLimit:         50,
		BurstMultiplier:    1.0,
		Window:             time.Second * 10,
	}

	limiter := NewAPIRateLimiter(nil, cfg)

	// Use a consistent user ID for all requests
	consistentUserID := uuid.New()

	router := gin.New()
	// Simulate authentication middleware with consistent user
	router.Use(func(c *gin.Context) {
		c.Set("userID", consistentUserID)
		c.Next()
	})
	router.GET("/test", limiter.RateLimitAuthenticated(), func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	// First 10 requests should succeed
	for i := 0; i < 10; i++ {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/test", nil)
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Request %d: expected 200, got %d", i+1, w.Code)
		}

		limitHeader := w.Header().Get("X-RateLimit-Limit")
		if limitHeader != "10" {
			t.Errorf("Expected X-RateLimit-Limit=10, got %s", limitHeader)
		}
	}

	// 11th request should be rate limited
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusTooManyRequests {
		t.Errorf("Expected 429, got %d", w.Code)
	}
}

func TestAPIRateLimiter_AdminLimit(t *testing.T) {
	cfg := &config.RateLimitConfig{
		Enabled:            true,
		UseRedis:           false,
		AnonymousLimit:     100,
		AuthenticatedLimit: 100,
		AdminLimit:         3,
		BurstMultiplier:    1.0,
		Window:             time.Second * 10,
	}

	limiter := NewAPIRateLimiter(nil, cfg)

	// Use a consistent user ID for all requests
	consistentUserID := uuid.New()

	router := gin.New()
	// Simulate authentication middleware with consistent user
	router.Use(func(c *gin.Context) {
		c.Set("userID", consistentUserID)
		c.Set("userRole", "admin")
		c.Next()
	})
	router.GET("/admin/test", limiter.RateLimitAdmin(), func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	// First 3 requests should succeed
	for i := 0; i < 3; i++ {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/admin/test", nil)
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Request %d: expected 200, got %d", i+1, w.Code)
		}

		limitHeader := w.Header().Get("X-RateLimit-Limit")
		if limitHeader != "3" {
			t.Errorf("Expected X-RateLimit-Limit=3, got %s", limitHeader)
		}
	}

	// 4th request should be rate limited
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/admin/test", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusTooManyRequests {
		t.Errorf("Expected 429, got %d", w.Code)
	}
}

func TestAPIRateLimiter_ExcludedPaths(t *testing.T) {
	cfg := &config.RateLimitConfig{
		Enabled:            true,
		UseRedis:           false,
		AnonymousLimit:     1, // Very low limit
		AuthenticatedLimit: 1,
		AdminLimit:         1,
		BurstMultiplier:    1.0,
		Window:             time.Minute,
		ExcludedPaths:      []string{"/health", "/api/v1/version"},
	}

	limiter := NewAPIRateLimiter(nil, cfg)

	router := gin.New()
	router.GET("/health", limiter.RateLimitAnonymous(), func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})
	router.GET("/api/v1/version", limiter.RateLimitAnonymous(), func(c *gin.Context) {
		c.JSON(200, gin.H{"version": "1.0"})
	})

	// Make many requests to excluded paths - should all pass
	for i := 0; i < 100; i++ {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/health", nil)
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Health request %d: expected 200, got %d", i, w.Code)
		}

		w = httptest.NewRecorder()
		req, _ = http.NewRequest("GET", "/api/v1/version", nil)
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Version request %d: expected 200, got %d", i, w.Code)
		}
	}
}

func TestAPIRateLimiter_BurstMultiplier(t *testing.T) {
	cfg := &config.RateLimitConfig{
		Enabled:            true,
		UseRedis:           false,
		AnonymousLimit:     10,
		AuthenticatedLimit: 100,
		AdminLimit:         50,
		BurstMultiplier:    1.5, // 50% burst = 15 total
		Window:             time.Second * 10,
	}

	limiter := NewAPIRateLimiter(nil, cfg)

	router := gin.New()
	router.GET("/test", limiter.RateLimitAnonymous(), func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	// With 1.5x burst, we should be able to make 15 requests
	successCount := 0
	for i := 0; i < 20; i++ {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/test", nil)
		req.Header.Set("X-Forwarded-For", "10.0.0.1")
		router.ServeHTTP(w, req)

		if w.Code == http.StatusOK {
			successCount++
		}
	}

	// Should have allowed ~15 requests (10 * 1.5)
	if successCount != 15 {
		t.Errorf("Expected 15 successful requests with burst, got %d", successCount)
	}
}

func TestAPIRateLimiter_DynamicRateLimit(t *testing.T) {
	cfg := &config.RateLimitConfig{
		Enabled:            true,
		UseRedis:           false,
		AnonymousLimit:     3,
		AuthenticatedLimit: 10,
		AdminLimit:         50,
		BurstMultiplier:    1.0,
		Window:             time.Second * 10,
	}

	limiter := NewAPIRateLimiter(nil, cfg)

	t.Run("anonymous user gets anonymous limit", func(t *testing.T) {
		router := gin.New()
		router.GET("/test", limiter.DynamicRateLimit(), func(c *gin.Context) {
			c.JSON(200, gin.H{"status": "ok"})
		})

		successCount := 0
		for i := 0; i < 10; i++ {
			w := httptest.NewRecorder()
			req, _ := http.NewRequest("GET", "/test", nil)
			req.Header.Set("X-Forwarded-For", "172.16.0.1")
			router.ServeHTTP(w, req)

			if w.Code == http.StatusOK {
				successCount++
			}
		}

		if successCount != 3 {
			t.Errorf("Expected 3 successful requests for anonymous, got %d", successCount)
		}
	})

	t.Run("authenticated user gets authenticated limit", func(t *testing.T) {
		router := gin.New()
		userID := uuid.New()
		router.Use(func(c *gin.Context) {
			c.Set("userID", userID)
			c.Next()
		})
		router.GET("/test", limiter.DynamicRateLimit(), func(c *gin.Context) {
			c.JSON(200, gin.H{"status": "ok"})
		})

		successCount := 0
		for i := 0; i < 15; i++ {
			w := httptest.NewRecorder()
			req, _ := http.NewRequest("GET", "/test", nil)
			router.ServeHTTP(w, req)

			if w.Code == http.StatusOK {
				successCount++
			}
		}

		if successCount != 10 {
			t.Errorf("Expected 10 successful requests for authenticated, got %d", successCount)
		}
	})
}

func TestAPIRateLimiter_DifferentUsers(t *testing.T) {
	// Test that different users have separate rate limit buckets
	cfg := &config.RateLimitConfig{
		Enabled:            true,
		UseRedis:           false,
		AnonymousLimit:     100, // High anonymous limit
		AuthenticatedLimit: 2,   // Low authenticated limit
		AdminLimit:         50,
		BurstMultiplier:    1.0,
		Window:             time.Minute,
	}

	limiter := NewAPIRateLimiter(nil, cfg)

	// Create different users
	users := []uuid.UUID{uuid.New(), uuid.New(), uuid.New()}

	// Each user should have its own rate limit bucket
	for _, userID := range users {
		router := gin.New()
		router.Use(func(c *gin.Context) {
			c.Set("userID", userID)
			c.Next()
		})
		router.GET("/test", limiter.RateLimitAuthenticated(), func(c *gin.Context) {
			c.JSON(200, gin.H{"status": "ok"})
		})

		// First 2 requests should succeed
		for i := 0; i < 2; i++ {
			w := httptest.NewRecorder()
			req, _ := http.NewRequest("GET", "/test", nil)
			router.ServeHTTP(w, req)

			if w.Code != http.StatusOK {
				t.Errorf("User %s request %d: expected 200, got %d", userID, i+1, w.Code)
			}
		}

		// 3rd request for this user should be blocked
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/test", nil)
		router.ServeHTTP(w, req)

		if w.Code != http.StatusTooManyRequests {
			t.Errorf("User %s: expected 429 on 3rd request, got %d", userID, w.Code)
		}
	}
}

func TestAPIRateLimiter_ConcurrentRequests(t *testing.T) {
	cfg := &config.RateLimitConfig{
		Enabled:            true,
		UseRedis:           false,
		AnonymousLimit:     100,
		AuthenticatedLimit: 1000,
		AdminLimit:         50,
		BurstMultiplier:    1.0,
		Window:             time.Second * 10,
	}

	limiter := NewAPIRateLimiter(nil, cfg)

	router := gin.New()
	router.GET("/test", limiter.RateLimitAnonymous(), func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	var wg sync.WaitGroup
	results := make(chan int, 200)

	// Make concurrent requests
	for i := 0; i < 200; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			w := httptest.NewRecorder()
			req, _ := http.NewRequest("GET", "/test", nil)
			req.Header.Set("X-Forwarded-For", "10.10.10.10")
			router.ServeHTTP(w, req)
			results <- w.Code
		}()
	}

	wg.Wait()
	close(results)

	successCount := 0
	rateLimitedCount := 0
	for code := range results {
		if code == http.StatusOK {
			successCount++
		} else if code == http.StatusTooManyRequests {
			rateLimitedCount++
		}
	}

	// Should have ~100 successes and ~100 rate limited
	if successCount != 100 {
		t.Errorf("Expected 100 successful requests, got %d", successCount)
	}
	if rateLimitedCount != 100 {
		t.Errorf("Expected 100 rate limited requests, got %d", rateLimitedCount)
	}
}

func TestAPIRateLimiter_GetRateLimitStats(t *testing.T) {
	cfg := &config.RateLimitConfig{
		Enabled:            true,
		UseRedis:           false,
		AnonymousLimit:     100,
		AuthenticatedLimit: 1000,
		AdminLimit:         50,
		BurstMultiplier:    1.2,
		Window:             time.Minute,
	}

	limiter := NewAPIRateLimiter(nil, cfg)

	stats := limiter.GetRateLimitStats()

	if stats["enabled"] != true {
		t.Error("Expected enabled=true")
	}
	if stats["anonymous_limit"] != 100 {
		t.Errorf("Expected anonymous_limit=100, got %v", stats["anonymous_limit"])
	}
	if stats["authenticated_limit"] != 1000 {
		t.Errorf("Expected authenticated_limit=1000, got %v", stats["authenticated_limit"])
	}
	if stats["admin_limit"] != 50 {
		t.Errorf("Expected admin_limit=50, got %v", stats["admin_limit"])
	}
}

func TestAPIRateLimiter_ResponseFormat(t *testing.T) {
	cfg := &config.RateLimitConfig{
		Enabled:            true,
		UseRedis:           false,
		AnonymousLimit:     1,
		AuthenticatedLimit: 100,
		AdminLimit:         50,
		BurstMultiplier:    1.0,
		Window:             time.Minute,
	}

	limiter := NewAPIRateLimiter(nil, cfg)

	router := gin.New()
	router.GET("/test", limiter.RateLimitAnonymous(), func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	// First request succeeds
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected 200, got %d", w.Code)
	}

	// Second request gets rate limited
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("GET", "/test", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusTooManyRequests {
		t.Errorf("Expected 429, got %d", w.Code)
	}

	// Check response body contains expected fields
	body := w.Body.String()
	if body == "" {
		t.Error("Expected non-empty response body")
	}

	// Should contain error, message, and retry_after
	if w.Header().Get("Retry-After") == "" {
		t.Error("Expected Retry-After header")
	}
	if w.Header().Get("X-RateLimit-Limit") == "" {
		t.Error("Expected X-RateLimit-Limit header")
	}
	if w.Header().Get("X-RateLimit-Remaining") == "" {
		t.Error("Expected X-RateLimit-Remaining header")
	}
	if w.Header().Get("X-RateLimit-Reset") == "" {
		t.Error("Expected X-RateLimit-Reset header")
	}
}
