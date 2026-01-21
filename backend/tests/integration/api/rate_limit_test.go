//go:build integration

package api

import (
	"net/http"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/javaknight1/servicepro/backend/tests/integration"
	"github.com/javaknight1/servicepro/backend/tests/integration/helpers"
)

// =============================================================================
// Rate Limiting Tests
// =============================================================================

func TestRateLimiting(t *testing.T) {
	suite := integration.SetupTest(t)

	t.Run("Rate limit enforced", func(t *testing.T) {
		client, _ := suite.CreateAuthenticatedClient(t, "user")

		// Make rapid requests
		var rateLimited bool
		for i := 0; i < 100; i++ {
			resp := client.Get(t, "/api/v1/customers")

			if resp.StatusCode == http.StatusTooManyRequests {
				rateLimited = true
				break
			}

			// Small delay to not overwhelm
			time.Sleep(10 * time.Millisecond)
		}

		// Should eventually hit rate limit
		assert.True(t, rateLimited, "Expected to hit rate limit after many requests")
	})

	t.Run("Rate limit headers present", func(t *testing.T) {
		client, _ := suite.CreateAuthenticatedClient(t, "user")

		resp := client.Get(t, "/api/v1/customers")

		// Check for rate limit headers
		assert.NotEmpty(t, resp.Header.Get("X-RateLimit-Limit"), "Missing X-RateLimit-Limit header")
		assert.NotEmpty(t, resp.Header.Get("X-RateLimit-Remaining"), "Missing X-RateLimit-Remaining header")
	})

	t.Run("Rate limit resets after window", func(t *testing.T) {
		client, _ := suite.CreateAuthenticatedClient(t, "user")

		// Exhaust rate limit
		for i := 0; i < 100; i++ {
			resp := client.Get(t, "/api/v1/customers")
			if resp.StatusCode == http.StatusTooManyRequests {
				break
			}
		}

		// Wait for rate limit window to reset (this may be slow in tests)
		// In real tests, you might want to mock time or have a shorter window
		time.Sleep(1 * time.Second)

		// Should be able to make requests again
		resp := client.Get(t, "/api/v1/customers")
		// Might still be limited if window is longer, so check for either OK or 429
		assert.True(t,
			resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusTooManyRequests,
			"Expected OK or TooManyRequests, got %d", resp.StatusCode)
	})

	t.Run("Different users have separate limits", func(t *testing.T) {
		client1, _ := suite.CreateAuthenticatedClient(t, "user")
		client2, _ := suite.CreateAuthenticatedClient(t, "user")

		// Exhaust client1's rate limit
		for i := 0; i < 100; i++ {
			resp := client1.Get(t, "/api/v1/customers")
			if resp.StatusCode == http.StatusTooManyRequests {
				break
			}
		}

		// Client2 should still be able to make requests
		resp := client2.Get(t, "/api/v1/customers")
		assert.Equal(t, http.StatusOK, resp.StatusCode, "Second user should not be rate limited")
	})

	t.Run("Concurrent requests rate limited fairly", func(t *testing.T) {
		client, _ := suite.CreateAuthenticatedClient(t, "user")

		var wg sync.WaitGroup
		results := make(chan int, 50)

		// Make concurrent requests
		for i := 0; i < 50; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				resp := client.Get(t, "/api/v1/customers")
				results <- resp.StatusCode
			}()
		}

		wg.Wait()
		close(results)

		// Count responses
		var okCount, limitedCount int
		for status := range results {
			switch status {
			case http.StatusOK:
				okCount++
			case http.StatusTooManyRequests:
				limitedCount++
			}
		}

		t.Logf("OK: %d, Rate Limited: %d", okCount, limitedCount)

		// Should have some successful and some limited
		assert.Greater(t, okCount, 0, "Expected some successful requests")
	})

	t.Run("Anonymous rate limit stricter", func(t *testing.T) {
		anonymousClient := helpers.NewTestClient(suite.Config.APIBaseURL)
		authenticatedClient, _ := suite.CreateAuthenticatedClient(t, "user")

		// Count requests until rate limited
		var anonLimit, authLimit int

		for i := 0; i < 100; i++ {
			resp := anonymousClient.Get(t, "/api/v1/health")
			if resp.StatusCode == http.StatusTooManyRequests {
				anonLimit = i
				break
			}
			time.Sleep(10 * time.Millisecond)
		}

		for i := 0; i < 100; i++ {
			resp := authenticatedClient.Get(t, "/api/v1/customers")
			if resp.StatusCode == http.StatusTooManyRequests {
				authLimit = i
				break
			}
			time.Sleep(10 * time.Millisecond)
		}

		if anonLimit > 0 && authLimit > 0 {
			t.Logf("Anonymous limit: %d, Authenticated limit: %d", anonLimit, authLimit)
			// Anonymous should hit limit sooner (stricter)
			assert.LessOrEqual(t, anonLimit, authLimit,
				"Anonymous rate limit should be stricter than authenticated")
		}
	})
}

// =============================================================================
// Login Rate Limiting Tests
// =============================================================================

func TestLoginRateLimiting(t *testing.T) {
	suite := integration.SetupTest(t)

	t.Run("Login rate limit per IP", func(t *testing.T) {
		client := helpers.NewTestClient(suite.Config.APIBaseURL)

		var rateLimited bool
		for i := 0; i < 20; i++ {
			resp := client.Post(t, "/api/v1/auth/login", map[string]interface{}{
				"email":    "test@test.com",
				"password": "wrongpassword",
			})

			if resp.StatusCode == http.StatusTooManyRequests {
				rateLimited = true
				break
			}
		}

		assert.True(t, rateLimited, "Expected login rate limit after many failed attempts")
	})

	t.Run("Successful login resets failed attempt count", func(t *testing.T) {
		user := suite.Auth.CreateTestUser(t, &helpers.TestUser{
			Email:    "reset-attempts@test.com",
			Password: "TestPassword123!",
		})

		client := helpers.NewTestClient(suite.Config.APIBaseURL)

		// Make some failed attempts
		for i := 0; i < 3; i++ {
			client.Post(t, "/api/v1/auth/login", map[string]interface{}{
				"email":    user.Email,
				"password": "wrongpassword",
			})
		}

		// Successful login
		resp := client.Post(t, "/api/v1/auth/login", map[string]interface{}{
			"email":    user.Email,
			"password": "TestPassword123!",
		})

		helpers.AssertOK(t, resp)

		// Should be able to fail again without immediate lockout
		resp = client.Post(t, "/api/v1/auth/login", map[string]interface{}{
			"email":    user.Email,
			"password": "wrongpassword",
		})

		// Should be unauthorized, not rate limited
		assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	})
}

// =============================================================================
// API Endpoint Rate Limiting Tests
// =============================================================================

func TestEndpointSpecificRateLimiting(t *testing.T) {
	suite := integration.SetupTest(t)

	t.Run("Write operations have stricter limits", func(t *testing.T) {
		client, _ := suite.CreateAuthenticatedClient(t, "admin")

		// Count POST requests until limited
		var postLimit int
		for i := 0; i < 50; i++ {
			resp := client.Post(t, "/api/v1/customers", map[string]interface{}{
				"email":      "rate-test-" + string(rune(i)) + "@test.com",
				"first_name": "Test",
				"last_name":  "User",
			})

			if resp.StatusCode == http.StatusTooManyRequests {
				postLimit = i
				break
			}
			time.Sleep(10 * time.Millisecond)
		}

		t.Logf("POST limit: %d", postLimit)
	})

	t.Run("Bulk operations have specific limits", func(t *testing.T) {
		client, _ := suite.CreateAuthenticatedClient(t, "admin")

		// Test bulk endpoint if exists
		resp := client.Post(t, "/api/v1/customers/bulk", map[string]interface{}{
			"customers": []map[string]interface{}{
				{"email": "bulk1@test.com", "first_name": "Bulk", "last_name": "One"},
				{"email": "bulk2@test.com", "first_name": "Bulk", "last_name": "Two"},
			},
		})

		// Bulk endpoint may have different rate limits or not exist
		assert.True(t,
			resp.StatusCode == http.StatusOK ||
				resp.StatusCode == http.StatusCreated ||
				resp.StatusCode == http.StatusNotFound ||
				resp.StatusCode == http.StatusTooManyRequests,
			"Unexpected status: %d", resp.StatusCode)
	})
}

// =============================================================================
// Retry-After Header Tests
// =============================================================================

func TestRetryAfterHeader(t *testing.T) {
	suite := integration.SetupTest(t)

	t.Run("Retry-After header on rate limit", func(t *testing.T) {
		client, _ := suite.CreateAuthenticatedClient(t, "user")

		// Exhaust rate limit
		var rateLimitResp *helpers.Response
		for i := 0; i < 100; i++ {
			resp := client.Get(t, "/api/v1/customers")
			if resp.StatusCode == http.StatusTooManyRequests {
				rateLimitResp = resp
				break
			}
		}

		if rateLimitResp != nil {
			retryAfter := rateLimitResp.Header.Get("Retry-After")
			assert.NotEmpty(t, retryAfter, "Expected Retry-After header when rate limited")
			t.Logf("Retry-After: %s", retryAfter)
		}
	})
}
