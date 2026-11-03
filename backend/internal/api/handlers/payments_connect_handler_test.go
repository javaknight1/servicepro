package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/javaknight1/servicepro/backend/internal/api/middleware"
	"github.com/javaknight1/servicepro/backend/internal/services"
)

// T169: without Stripe configured, connecting answers a typed 503 instead of
// a generic 500, and a request without a verified tenant is refused.
func TestPaymentsConnectHandler_NotConfigured(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := NewPaymentsConnectHandler(services.NewPaymentsConnectService(services.PaymentsConnectServiceConfig{}))

	r := gin.New()
	r.POST("/tenants/:id/payments/connect", func(c *gin.Context) {
		c.Set(middleware.TenantIDKey, uuid.New()) // what RequireTenantFromPath sets
		c.Next()
	}, h.Connect)
	r.POST("/no-tenant", h.Connect)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/tenants/"+uuid.NewString()+"/payments/connect", nil))
	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
	var body map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.Equal(t, "payments_not_configured", body["error"])

	w = httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/no-tenant", nil))
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}
