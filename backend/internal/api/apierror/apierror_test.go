package apierror

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"

	apperrors "github.com/javaknight1/servicepro/backend/internal/errors"
	"github.com/javaknight1/servicepro/backend/internal/models"
	"github.com/javaknight1/servicepro/backend/pkg/clients/logging"
)

var errQuoteNotAcceptable = apperrors.Conflict("cannot_accept", "This quote can no longer be accepted")

// run executes h behind a stand-in for middleware.RequestID and decodes the
// envelope.
func run(t *testing.T, h gin.HandlerFunc, body string) (int, models.ErrorResponse) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Request = c.Request.WithContext(logging.WithRequestID(c.Request.Context(), "req-test-0001"))
		c.Next()
	})
	r.POST("/x", h)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/x", strings.NewReader(body)))
	var resp models.ErrorResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("body is not the envelope: %q (%v)", w.Body.String(), err)
	}
	if resp.RequestID != "req-test-0001" {
		t.Fatalf("request_id = %q, want req-test-0001 (body %s)", resp.RequestID, w.Body.String())
	}
	return w.Code, resp
}

func TestRespond_DomainErrorKeepsStatusCodeAndMessage(t *testing.T) {
	wrapped := fmt.Errorf("accept quote 42: %w", errQuoteNotAcceptable)
	status, resp := run(t, func(c *gin.Context) { Respond(c, wrapped, "Failed to accept quote") }, "")
	if status != http.StatusConflict || resp.Error != "cannot_accept" || resp.Message != "This quote can no longer be accepted" {
		t.Fatalf("got %d %+v", status, resp)
	}
}

func TestRespond_DomainErrorWithCauseHidesCause(t *testing.T) {
	err := errQuoteNotAcceptable.WithCause(fmt.Errorf("pq: relation \"quotes\" does not exist"))
	status, resp := run(t, func(c *gin.Context) { Respond(c, err, "") }, "")
	if status != http.StatusConflict || strings.Contains(resp.Message, "pq:") {
		t.Fatalf("got %d %+v", status, resp)
	}
}

func TestRespond_InfrastructureErrorIsGeneric500(t *testing.T) {
	err := fmt.Errorf("failed to accept quote: %w", fmt.Errorf("pq: connection refused"))
	status, resp := run(t, func(c *gin.Context) { Respond(c, err, "Failed to accept quote") }, "")
	if status != http.StatusInternalServerError || resp.Error != "internal_error" || resp.Message != "Failed to accept quote" {
		t.Fatalf("got %d %+v", status, resp)
	}

	status, resp = run(t, func(c *gin.Context) { Respond(c, err, "") }, "")
	if status != http.StatusInternalServerError || strings.Contains(resp.Message, "pq") || resp.Message == "" {
		t.Fatalf("default 500 message leaked or empty: %+v", resp)
	}
}

type bindReq struct {
	Email      string `json:"email" binding:"required,email"`
	CustomerID string `json:"customer_id" binding:"required,uuid"`
	Count      int    `json:"count"`
}

func bindHandler(c *gin.Context) {
	var req bindReq
	if err := c.ShouldBindJSON(&req); err != nil {
		BindError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func TestBindError_ValidationReportsJSONFieldNames(t *testing.T) {
	status, resp := run(t, bindHandler, `{"email":"nope"}`)
	if status != http.StatusBadRequest || resp.Error != apperrors.CodeValidation {
		t.Fatalf("got %d %+v", status, resp)
	}
	if resp.Fields["email"] != "email must be a valid email address" || resp.Fields["customer_id"] != "customer_id is required" {
		t.Fatalf("fields = %v", resp.Fields)
	}
	if !strings.HasPrefix(resp.Message, "Validation failed: ") {
		t.Fatalf("message = %q", resp.Message)
	}
}

func TestBindError_MalformedAndWrongType(t *testing.T) {
	status, resp := run(t, bindHandler, `{"email":`)
	if status != http.StatusBadRequest || resp.Error != CodeMalformedJSON {
		t.Fatalf("malformed: got %d %+v", status, resp)
	}
	status, resp = run(t, bindHandler, `{"email":"a@b.co","customer_id":"6f1c0f3e-8f0b-4a40-9d2e-1f5b8a7c9d10","count":"x"}`)
	if status != http.StatusBadRequest || resp.Error != apperrors.CodeValidation || resp.Fields["count"] == "" {
		t.Fatalf("wrong type: got %d %+v", status, resp)
	}
}

func TestWriteResponse_NormalisesLegacySentenceInError(t *testing.T) {
	status, resp := run(t, func(c *gin.Context) {
		WriteResponse(c, http.StatusNotFound, models.ErrorResponse{Error: "Invoice not found"})
	}, "")
	if status != http.StatusNotFound || resp.Error != "not_found" || resp.Message != "Invoice not found" {
		t.Fatalf("got %d %+v", status, resp)
	}
}

func TestWriteResponse_CodeWithoutMessageGetsHumanMessage(t *testing.T) {
	_, resp := run(t, func(c *gin.Context) {
		WriteResponse(c, http.StatusBadRequest, models.ErrorResponse{Error: "invalid_tenant_id"})
	}, "")
	if resp.Error != "invalid_tenant_id" || resp.Message != "Invalid tenant id" {
		t.Fatalf("got %+v", resp)
	}
}

func TestDomainSentinelIdentity(t *testing.T) {
	other := apperrors.Conflict("cannot_accept", "different sentinel, same code")
	decorated := errQuoteNotAcceptable.WithMessage("Quote expired").WithCause(fmt.Errorf("x"))
	if !errors.Is(decorated, errQuoteNotAcceptable) {
		t.Fatal("decorated copy no longer matches its sentinel")
	}
	if errors.Is(decorated, other) || errors.Is(other, errQuoteNotAcceptable) {
		t.Fatal("distinct sentinels with the same code must not match")
	}
}
