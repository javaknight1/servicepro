package handlers

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

func TestRunQuoteExpiry(t *testing.T) {
	gin.SetMode(gin.TestMode)
	jobs := NewInternalJobsHandler("s3cret", nil)
	r := gin.New()
	r.POST("/internal/jobs/quote-expiry", jobs.Authorize, RunQuoteExpiry(func(context.Context, time.Time) (int64, error) { return 3, nil }))
	r.POST("/internal/jobs/broken", jobs.Authorize, RunQuoteExpiry(func(context.Context, time.Time) (int64, error) { return 0, errors.New("db down") }))

	do := func(path, token string) *httptest.ResponseRecorder {
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, path, nil)
		if token != "" {
			req.Header.Set(InternalJobsTokenHeader, token)
		}
		r.ServeHTTP(w, req)
		return w
	}

	if w := do("/internal/jobs/quote-expiry", ""); w.Code != http.StatusUnauthorized {
		t.Fatalf("no token: %d", w.Code)
	}
	w := do("/internal/jobs/quote-expiry", "s3cret")
	if w.Code != http.StatusOK || !strings.Contains(w.Body.String(), `"expired":3`) {
		t.Fatalf("ok run: %d %s", w.Code, w.Body.String())
	}
	if w := do("/internal/jobs/broken", "s3cret"); w.Code != http.StatusInternalServerError || strings.Contains(w.Body.String(), "db down") {
		t.Fatalf("failure must be 500 without internals: %d %s", w.Code, w.Body.String())
	}
}
