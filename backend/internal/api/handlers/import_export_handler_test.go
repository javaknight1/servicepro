package handlers

import (
	"bytes"
	"context"
	"errors"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/javaknight1/servicepro/backend/internal/services"
)

// importRequest builds a multipart CSV upload to /customers/import.
func importRequest(t *testing.T, csv []byte) *http.Request {
	t.Helper()
	var body bytes.Buffer
	w := multipart.NewWriter(&body)
	if err := w.WriteField("format", "csv"); err != nil {
		t.Fatal(err)
	}
	fw, err := w.CreateFormFile("file", "customers.csv")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := fw.Write(csv); err != nil {
		t.Fatal(err)
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodPost, "/customers/import", &body)
	req.Header.Set("Content-Type", w.FormDataContentType())
	return req
}

func serveImport(t *testing.T, csv []byte) *httptest.ResponseRecorder {
	t.Helper()
	gin.SetMode(gin.TestMode)
	r := gin.New()
	// The caps are checked before the service is used, so none is needed.
	r.POST("/customers/import", NewImportExportHandler(nil).ImportCustomers)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, importRequest(t, csv))
	return w
}

const importHeader = "first_name,last_name,email,phone_primary,billing_address_street,billing_address_city,billing_address_state,billing_address_zip,customer_type\n"

// Regression (F-23): an oversized upload is rejected with 413 without being
// read into memory.
func TestImportCustomers_RejectsOversizedUpload(t *testing.T) {
	big := bytes.Repeat([]byte("x"), int(services.ImportMaxUploadBytes)+256<<10)
	w := serveImport(t, append([]byte(importHeader), big...))
	if w.Code != http.StatusRequestEntityTooLarge || !strings.Contains(w.Body.String(), "file_too_large") {
		t.Fatalf("want 413 file_too_large, got %d %s", w.Code, w.Body.String())
	}
}

func TestImportCustomers_RejectsTooManyRows(t *testing.T) {
	var b strings.Builder
	b.WriteString(importHeader)
	for range services.ImportMaxRows + 1 {
		b.WriteString("A,B,a@example.com,555-123-4567,1 Main,Austin,TX,78701,commercial\n")
	}
	w := serveImport(t, []byte(b.String()))
	if w.Code != http.StatusBadRequest || !strings.Contains(w.Body.String(), "file_too_large") {
		t.Fatalf("want 400 file_too_large, got %d %s", w.Code, w.Body.String())
	}
}

func TestRunImports(t *testing.T) {
	gin.SetMode(gin.TestMode)
	jobs := NewInternalJobsHandler("s3cret", nil)
	r := gin.New()
	var gotMax int
	var gotBudget time.Duration
	r.POST("/internal/jobs/imports", jobs.Authorize, RunImports(func(_ context.Context, maxJobs int, budget time.Duration) (services.ImportRunResult, error) {
		gotMax, gotBudget = maxJobs, budget
		return services.ImportRunResult{Claimed: 2, Completed: 1, Failed: 1}, nil
	}))
	r.POST("/internal/jobs/broken", jobs.Authorize, RunImports(func(context.Context, int, time.Duration) (services.ImportRunResult, error) {
		return services.ImportRunResult{}, errors.New("db down")
	}))

	do := func(path, token string) *httptest.ResponseRecorder {
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, path, nil)
		if token != "" {
			req.Header.Set(InternalJobsTokenHeader, token)
		}
		r.ServeHTTP(w, req)
		return w
	}

	if w := do("/internal/jobs/imports", "wrong"); w.Code != http.StatusUnauthorized {
		t.Fatalf("bad token: %d", w.Code)
	}
	w := do("/internal/jobs/imports", "s3cret")
	if w.Code != http.StatusOK || !strings.Contains(w.Body.String(), `"claimed":2`) || !strings.Contains(w.Body.String(), `"job":"imports"`) {
		t.Fatalf("ok run: %d %s", w.Code, w.Body.String())
	}
	if gotMax <= 0 || gotBudget <= 0 || gotBudget >= 5*time.Minute {
		t.Fatalf("runner limits: max=%d budget=%s (must finish inside the scheduler's 300 s)", gotMax, gotBudget)
	}
	if w := do("/internal/jobs/broken", "s3cret"); w.Code != http.StatusInternalServerError || strings.Contains(w.Body.String(), "db down") {
		t.Fatalf("failure must be 500 without internals: %d %s", w.Code, w.Body.String())
	}
}
