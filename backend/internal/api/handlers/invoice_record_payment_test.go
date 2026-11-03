package handlers

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// POST /invoices/{id}/payments returned 500 for every payment: a `gt=0` tag
// on the decimal amount made the validator panic.
func TestRecordPaymentRequest_ValidatesWithoutPanic(t *testing.T) {
	req := RecordPaymentRequest{Amount: decimal.RequireFromString("125.50")}
	if err := binding.Validator.ValidateStruct(&req); err != nil {
		t.Fatalf("valid payment rejected: %v", err)
	}
}

func TestRecordPayment_RejectsNonPositiveAmount(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := NewInvoiceHandler(nil) // rejected before the service is reached

	for _, amount := range []string{`"0"`, `"-5"`, `0`} {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Params = gin.Params{{Key: "id", Value: uuid.NewString()}}
		c.Request = httptest.NewRequest(http.MethodPost, "/invoices/x/payments", strings.NewReader(`{"amount":`+amount+`}`))
		c.Request.Header.Set("Content-Type", "application/json")

		h.RecordPayment(c)

		if w.Code != http.StatusBadRequest {
			t.Fatalf("amount %s: status = %d, want 400 (%s)", amount, w.Code, w.Body.String())
		}
	}
}
