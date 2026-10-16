package middleware

import (
	"bytes"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/javaknight1/servicepro/backend/config"
)

func TestLoginBackoffSchedule(t *testing.T) {
	assert.Zero(t, loginBackoff(1))
	assert.Zero(t, loginBackoff(2))
	assert.Equal(t, time.Second, loginBackoff(3))
	assert.Equal(t, 4*time.Second, loginBackoff(5))
	assert.Equal(t, loginPairMaxBackoff, loginBackoff(40))
}

func TestLoginThrottle_PerAccountAndIP(t *testing.T) {
	mr, err := miniredis.Run()
	require.NoError(t, err)
	defer mr.Close()
	rl := NewRateLimiter(redis.NewClient(&redis.Options{Addr: mr.Addr()}), &config.AuthConfig{})

	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.POST("/login", rl.LoginRateLimit(), func(c *gin.Context) { c.Status(http.StatusUnauthorized) })
	attempt := func(ip string) int {
		req := httptest.NewRequest(http.MethodPost, "/login", bytes.NewBufferString(`{"email":"victim@example.com","password":"WrongPassword1!"}`))
		req.Header.Set("Content-Type", "application/json")
		req.RemoteAddr = net.JoinHostPort(ip, "1234")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		return w.Code
	}

	attacker, victim := "198.51.100.1", "203.0.113.5"
	for i := 0; i < 3; i++ {
		require.Equal(t, http.StatusUnauthorized, attempt(attacker))
		rl.RecordLoginFailure("victim@example.com", attacker)
	}
	assert.Equal(t, http.StatusTooManyRequests, attempt(attacker), "attacker is backed off")
	assert.Equal(t, http.StatusUnauthorized, attempt(victim), "victim on another IP is not locked out")

	// The backoff expires.
	mr.FastForward(2 * time.Second)
	assert.Equal(t, http.StatusUnauthorized, attempt(attacker))

	// Success clears the pair.
	require.NoError(t, rl.ClearLoginAttempts("victim@example.com", attacker))
	rl.RecordLoginFailure("victim@example.com", attacker)
	assert.Equal(t, http.StatusUnauthorized, attempt(attacker))
}
