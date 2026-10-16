package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestRequireCSRFHeader(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(RequireCSRFHeader("access_token", "refresh_token"))
	ok := func(c *gin.Context) { c.Status(http.StatusNoContent) }
	r.POST("/x", ok)
	r.GET("/x", ok)

	cases := []struct {
		name, method, cookie, header string
		want                         int
	}{
		{"cookie mutation without header", http.MethodPost, "access_token=t", "", http.StatusForbidden},
		{"refresh cookie without header", http.MethodPost, "refresh_token=t", "", http.StatusForbidden},
		{"cookie mutation with header", http.MethodPost, "access_token=t", CSRFHeaderValue, http.StatusNoContent},
		{"wrong header value", http.MethodPost, "access_token=t", "fetch", http.StatusForbidden},
		{"no cookie (bearer/webhook)", http.MethodPost, "", "", http.StatusNoContent},
		{"safe method", http.MethodGet, "access_token=t", "", http.StatusNoContent},
	}
	for _, tc := range cases {
		req := httptest.NewRequest(tc.method, "/x", nil)
		if tc.cookie != "" {
			req.Header.Set("Cookie", tc.cookie)
		}
		if tc.header != "" {
			req.Header.Set(CSRFHeader, tc.header)
		}
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != tc.want {
			t.Errorf("%s: got %d, want %d", tc.name, w.Code, tc.want)
		}
	}
}
