package handlers

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/javaknight1/servicepro/backend/internal/api/middleware"
	"github.com/javaknight1/servicepro/backend/internal/models"
	"github.com/javaknight1/servicepro/backend/internal/services"
)

// T223: invoice service errors reach the client in the envelope with a
// stable code and the business-rule sentence; infrastructure errors are a
// generic 500 with the request ID.
func TestRespondInvoiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	for _, tc := range []struct {
		err     error
		status  int
		code    string
		message string
	}{
		{services.ErrOverpayment, http.StatusBadRequest, "overpayment", "Payment amount exceeds outstanding balance"},
		{fmt.Errorf("%w on line 2 - quantity must be greater than 0", services.ErrInvalidLineItem), http.StatusBadRequest, "invalid_line_item", "Invalid line item on line 2 - quantity must be greater than 0"},
		{fmt.Errorf("record payment: %w", services.ErrInvoiceClosed), http.StatusConflict, "invoice_closed", "Paid, cancelled and refunded invoices can't be edited"},
		{services.ErrInvoiceNotFound, http.StatusNotFound, "not_found", "Invoice not found"},
		{services.ErrReceiptNotAvailable, http.StatusConflict, "receipt_not_available", "A receipt is only available for paid invoices"},
		{fmt.Errorf("failed to record payment: %w", errors.New("pq: could not serialize access")), http.StatusInternalServerError, "internal_error", "Failed to record payment"},
	} {
		r := gin.New()
		r.Use(middleware.RequestID())
		r.POST("/x", func(c *gin.Context) { respondInvoiceError(c, tc.err, "Failed to record payment") })
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/x", nil))

		var body models.ErrorResponse
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
		assert.Equal(t, tc.status, w.Code, tc.err.Error())
		assert.Equal(t, tc.code, body.Error)
		assert.Equal(t, tc.message, body.Message)
		assert.NotEmpty(t, body.RequestID)
	}
}
