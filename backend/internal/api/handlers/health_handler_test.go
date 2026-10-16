package handlers

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"

	"github.com/javaknight1/servicepro/backend/internal/health"
)

func TestHealth_DetailsOnlyForAuthorizedCallers(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := NewHealthHandler(health.NewChecker("test-version", "production"))
	h.SetDetailAccess(false, "jobs-secret")
	r := gin.New()
	r.GET("/health", h.Health)

	call := func(remote, header, value string) string {
		req := httptest.NewRequest(http.MethodGet, "/health", nil)
		req.RemoteAddr = remote
		if header != "" {
			req.Header.Set(header, value)
		}
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		return w.Body.String()
	}

	public := call("203.0.113.9:443", "", "")
	assert.NotContains(t, public, "checks")
	assert.Contains(t, public, `"status"`)

	assert.NotContains(t, call("203.0.113.9:443", InternalJobsTokenHeader, "wrong"), "checks")
	assert.Contains(t, call("203.0.113.9:443", InternalJobsTokenHeader, "jobs-secret"), "checks")
	assert.Contains(t, call("203.0.113.9:443", "Authorization", "Bearer jobs-secret"), "checks")
	assert.Contains(t, call("127.0.0.1:5555", "", ""), "checks")
}
