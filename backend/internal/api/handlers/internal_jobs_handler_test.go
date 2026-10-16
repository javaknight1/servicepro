package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/javaknight1/servicepro/backend/internal/services"
)

type fakeReminderRunner struct {
	calls  int
	result services.ReminderRunResult
}

func (f *fakeReminderRunner) ProcessAllTenants(context.Context) (services.ReminderRunResult, error) {
	f.calls++
	return f.result, nil
}

func newJobsRouter(secret string, runner ReminderRunner) *gin.Engine {
	gin.SetMode(gin.TestMode)
	h := NewInternalJobsHandler(secret, runner)
	r := gin.New()
	r.Group("/internal/jobs", h.Authorize).POST("/payment-reminders", h.RunPaymentReminders)
	return r
}

func postJob(r *gin.Engine, token string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodPost, "/internal/jobs/payment-reminders", nil)
	if token != "" {
		req.Header.Set(InternalJobsTokenHeader, token)
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func TestInternalJobs_Auth(t *testing.T) {
	cases := []struct {
		name   string
		secret string
		token  string
		want   int
	}{
		{"secret unset -> disabled", "", "anything", http.StatusNotFound},
		{"missing token", "shh", "", http.StatusUnauthorized},
		{"wrong token", "shh", "nope", http.StatusUnauthorized},
		{"valid token", "shh", "shh", http.StatusOK},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			runner := &fakeReminderRunner{}
			w := postJob(newJobsRouter(tc.secret, runner), tc.token)
			if w.Code != tc.want {
				t.Fatalf("status = %d, want %d (body %s)", w.Code, tc.want, w.Body.String())
			}
			if wantCalls := map[bool]int{true: 1, false: 0}[tc.want == http.StatusOK]; runner.calls != wantCalls {
				t.Fatalf("runner called %d times, want %d", runner.calls, wantCalls)
			}
		})
	}
}

func TestInternalJobs_PaymentRemindersResponse(t *testing.T) {
	runner := &fakeReminderRunner{result: services.ReminderRunResult{TenantsProcessed: 2, Sent: 3, Skipped: 1}}
	w := postJob(newJobsRouter("shh", runner), "shh")

	var body PaymentRemindersJobResponse
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body.Job != "payment-reminders" || body.Sent != 3 || body.Skipped != 1 || body.TenantsProcessed != 2 {
		t.Fatalf("unexpected body: %+v", body)
	}
}

func TestInternalJobs_NoService(t *testing.T) {
	w := postJob(newJobsRouter("shh", nil), "shh")
	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503", w.Code)
	}
}
