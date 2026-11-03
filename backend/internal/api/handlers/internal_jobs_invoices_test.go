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

func TestRunInvoiceOverdue(t *testing.T) {
	gin.SetMode(gin.TestMode)
	jobs := NewInternalJobsHandler("s3cret", nil)
	r := gin.New()
	r.POST("/internal/jobs/invoice-overdue", jobs.Authorize, RunInvoiceOverdue(func(context.Context, time.Time) (int64, error) { return 4, nil }))
	r.POST("/internal/jobs/broken", jobs.Authorize, RunInvoiceOverdue(func(context.Context, time.Time) (int64, error) { return 0, errors.New("db down") }))

	do := func(path, token string) *httptest.ResponseRecorder {
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, path, nil)
		if token != "" {
			req.Header.Set(InternalJobsTokenHeader, token)
		}
		r.ServeHTTP(w, req)
		return w
	}

	if w := do("/internal/jobs/invoice-overdue", ""); w.Code != http.StatusUnauthorized {
		t.Fatalf("no token: %d", w.Code)
	}
	w := do("/internal/jobs/invoice-overdue", "s3cret")
	if w.Code != http.StatusOK || !strings.Contains(w.Body.String(), `"marked":4`) {
		t.Fatalf("ok run: %d %s", w.Code, w.Body.String())
	}
	if w := do("/internal/jobs/broken", "s3cret"); w.Code != http.StatusInternalServerError || strings.Contains(w.Body.String(), "db down") {
		t.Fatalf("failure must be 500 without internals: %d %s", w.Code, w.Body.String())
	}
}
