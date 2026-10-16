package middleware

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/javaknight1/servicepro/backend/internal/models"
)

// T223: middleware-written errors (panic recovery, unknown route, CSRF)
// use the envelope and carry the request ID from the header.
func TestMiddlewareErrors_UseEnvelopeWithRequestID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(RequestID())
	r.Use(RecoveryMiddleware(nil))
	r.NoRoute(NotFoundHandler())
	r.GET("/panic", func(c *gin.Context) { panic("boom: pq: secret detail") })

	for _, tc := range []struct {
		path   string
		status int
		code   string
	}{
		{"/panic", http.StatusInternalServerError, "internal_error"},
		{"/nope", http.StatusNotFound, "not_found"},
	} {
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, tc.path, nil))
		var body models.ErrorResponse
		if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
			t.Fatalf("%s: body is not the envelope: %q", tc.path, w.Body.String())
		}
		if w.Code != tc.status || body.Error != tc.code || body.Message == "" {
			t.Fatalf("%s: got %d %+v", tc.path, w.Code, body)
		}
		if body.RequestID == "" || body.RequestID != w.Header().Get(RequestIDHeader) {
			t.Fatalf("%s: request_id %q vs header %q", tc.path, body.RequestID, w.Header().Get(RequestIDHeader))
		}
		if tc.path == "/panic" && body.Message != "An unexpected error occurred" {
			t.Fatalf("panic text leaked: %q", body.Message)
		}
	}
}
