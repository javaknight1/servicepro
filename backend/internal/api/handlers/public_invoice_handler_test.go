package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/javaknight1/servicepro/backend/internal/services"
	stripeService "github.com/javaknight1/servicepro/backend/internal/services/stripe"
)

// T261: a pay link opened while online payment is off answers 409 with a
// readable message instead of redirecting to Stripe.
func TestPayInvoice_OnlinePaymentsDisabled(t *testing.T) {
	gin.SetMode(gin.TestMode)
	paySvc, err := services.NewInvoicePaymentService(&services.InvoicePaymentServiceConfig{
		StripeClient:   &stripeService.Client{}, // never called: disabled refuses first
		InvoiceService: services.NewInvoiceService(nil),
		SuccessURL:     "https://app.example.test/ok",
		CancelURL:      "https://app.example.test/cancel",
		Enabled:        false,
	})
	require.NoError(t, err)

	r := gin.New()
	r.GET("/public/invoices/pay/:token", NewPublicInvoiceHandler(paySvc, nil).PayInvoice)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/public/invoices/pay/some-token", nil))

	assert.Equal(t, http.StatusConflict, w.Code)
	assert.Empty(t, w.Header().Get("Location"), "must not redirect to Stripe")
	var body map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.Equal(t, "conflict", body["error"])
	assert.Contains(t, body["message"], "Online payment isn't available")
}
