package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestRequireEdgeProxy(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(EdgeClientIP("s3cret"), RequireEdgeProxy())
	ok := func(c *gin.Context) { c.Status(http.StatusNoContent) }
	r.GET("/api/v1/customers", ok)
	r.POST("/api/v1/stripe/webhooks", ok)
	r.GET("/health/live", ok)
	r.GET("/version", ok)
	r.POST("/internal/jobs/payment-reminders", ok)

	cases := []struct {
		method, path, secret string
		want                 int
	}{
		{http.MethodGet, "/api/v1/customers", "", http.StatusForbidden},
		{http.MethodGet, "/api/v1/customers", "wrong", http.StatusForbidden},
		{http.MethodGet, "/api/v1/customers", "s3cret", http.StatusNoContent},
		{http.MethodPost, "/api/v1/stripe/webhooks", "", http.StatusForbidden},
		{http.MethodGet, "/health/live", "", http.StatusNoContent},
		{http.MethodGet, "/version", "", http.StatusNoContent},
		{http.MethodPost, "/internal/jobs/payment-reminders", "", http.StatusNoContent},
	}
	for _, tc := range cases {
		req := httptest.NewRequest(tc.method, tc.path, nil)
		if tc.secret != "" {
			req.Header.Set(EdgeSecretHeader, tc.secret)
		}
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != tc.want {
			t.Errorf("%s %s secret=%q: got %d, want %d", tc.method, tc.path, tc.secret, w.Code, tc.want)
		}
	}
}
