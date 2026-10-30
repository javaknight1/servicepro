package middleware

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/javaknight1/servicepro/backend/config"
)

// Regression: the login and password-reset limiters bound the JSON body to
// read the email, which consumed it, so the handler saw an empty body and
// /auth/reset-request always answered 400 malformed_json (found 2026-10-30
// in production). The handler must still receive the full body.
func TestRateLimiters_LeaveRequestBodyForHandler(t *testing.T) {
	gin.SetMode(gin.TestMode)
	mr, err := miniredis.Run()
	require.NoError(t, err)
	defer mr.Close()
	rl := NewRateLimiter(redis.NewClient(&redis.Options{Addr: mr.Addr()}), &config.AuthConfig{})

	cases := []struct {
		name       string
		middleware gin.HandlerFunc
		body       string
	}{
		{"password reset", rl.PasswordResetRateLimit(), `{"email":"a@example.test"}`},
		{"login", rl.LoginRateLimit(), `{"email":"a@example.test","password":"Secret123!"}`},
		{"login missing password", rl.LoginRateLimit(), `{"email":"a@example.test"}`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var got map[string]any
			r := gin.New()
			r.POST("/x", tc.middleware, func(c *gin.Context) {
				require.NoError(t, c.ShouldBindJSON(&got), "handler must still see the body")
				c.Status(http.StatusNoContent)
			})
			w := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodPost, "/x", bytes.NewBufferString(tc.body))
			req.Header.Set("Content-Type", "application/json")
			r.ServeHTTP(w, req)
			assert.Equal(t, http.StatusNoContent, w.Code)
			assert.Equal(t, "a@example.test", got["email"])
		})
	}
}
