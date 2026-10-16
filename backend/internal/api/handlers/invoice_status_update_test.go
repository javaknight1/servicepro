package handlers

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// Audit 7.H2: PUT /invoices/{id} {"status":"paid"} marked an invoice paid
// with no money received. Status is no longer settable through update.
func TestUpdateInvoice_RejectsStatus(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := NewInvoiceHandler(nil) // rejected before the service is reached

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{{Key: "id", Value: uuid.NewString()}}
	c.Request = httptest.NewRequest(http.MethodPut, "/invoices/x", strings.NewReader(`{"status":"paid"}`))
	c.Request.Header.Set("Content-Type", "application/json")

	h.UpdateInvoice(c)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400 (%s)", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "status can't be set") {
		t.Fatalf("body = %s", w.Body.String())
	}
}
