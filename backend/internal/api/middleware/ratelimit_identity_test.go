package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/javaknight1/servicepro/backend/config"
)

func TestDynamicRateLimit_RecognisesAuthenticatedUsers(t *testing.T) {
	gin.SetMode(gin.TestMode)
	rl := NewAPIRateLimiter(nil, &config.RateLimitConfig{
		Enabled: true, UseRedis: false, AnonymousLimit: 1, AuthenticatedLimit: 5,
		BurstMultiplier: 1, Window: time.Minute,
	})
	rl.SetIdentityResolver(func(c *gin.Context) (string, bool) {
		if c.GetHeader("Authorization") == "Bearer good" {
			return "user-1", true
		}
		return "", false
	})
	r := gin.New()
	r.Use(rl.DynamicRateLimit())
	r.GET("/x", func(c *gin.Context) { c.Status(http.StatusNoContent) })

	call := func(auth string) int {
		req := httptest.NewRequest(http.MethodGet, "/x", nil)
		req.RemoteAddr = "198.51.100.7:1"
		if auth != "" {
			req.Header.Set("Authorization", auth)
		}
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		return w.Code
	}

	for i := 0; i < 3; i++ {
		if code := call("Bearer good"); code != http.StatusNoContent {
			t.Fatalf("authenticated request %d got %d (should use the per-user limit)", i, code)
		}
	}
	if call("") != http.StatusNoContent {
		t.Fatal("first anonymous request should pass")
	}
	if call("") != http.StatusTooManyRequests {
		t.Fatal("second anonymous request should hit the per-IP limit")
	}
}
