package webhooks

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	stripeservice "github.com/javaknight1/servicepro/backend/internal/services/stripe"
)

// =============================================================================
// Test Helpers
// =============================================================================

func init() {
	gin.SetMode(gin.TestMode)
}

// MockNotifier for testing
type MockNotifier struct{}

func (n *MockNotifier) NotifyPaymentSucceeded(ctx context.Context, paymentID string, amount int64, currency string) error {
	return nil
}
func (n *MockNotifier) NotifyPaymentFailed(ctx context.Context, paymentID string, errorMsg string) error {
	return nil
}
func (n *MockNotifier) NotifySubscriptionCanceled(ctx context.Context, subscriptionID string, customerID string) error {
	return nil
}
func (n *MockNotifier) NotifyDisputeCreated(ctx context.Context, disputeID string, amount int64) error {
	return nil
}
func (n *MockNotifier) NotifyRefundCreated(ctx context.Context, refundID string, amount int64) error {
	return nil
}
func (n *MockNotifier) NotifyWebhookError(ctx context.Context, eventID string, eventType string, err error) error {
	return nil
}

const testWebhookSecret = "whsec_test_secret_key_12345"

func setupTestHandler() (*StripeWebhookHandler, *gin.Engine) {
	notifier := &MockNotifier{}

	config := &stripeservice.WebhookHandlerConfig{
		WebhookSecret:       testWebhookSecret,
		WebhookTolerance:    5 * time.Minute,
		MaxRetries:          3,
		RetryBaseDelay:      100 * time.Millisecond,
		RetryMaxDelay:       1 * time.Second,
		EnableDBLogging:     false,
		EnableNotifications: true,
		ProcessingTimeout:   5 * time.Second,
	}

	service := stripeservice.NewWebhookHandlerService(config, nil, notifier)
	handler := NewStripeWebhookHandler(service)

	router := gin.New()
	router.POST("/api/v1/webhooks/stripe", handler.HandleWebhook)
	router.GET("/api/v1/webhooks/health", handler.HealthCheck)

	return handler, router
}

// =============================================================================
// Webhook Endpoint Tests
// =============================================================================

// Note: Full webhook signature verification and event processing is tested
// in the service tests (webhook_handler_test.go).
// These HTTP handler tests focus on routing, response formats, and error handling.

func TestHandleWebhookMissingSignature(t *testing.T) {
	_, router := setupTestHandler()

	payload := []byte(`{"id": "evt_test", "type": "payment_intent.succeeded"}`)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/webhooks/stripe", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	// No Stripe-Signature header

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected status 401, got %d", w.Code)
	}

	var response WebhookErrorResponse
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if response.Code != "MISSING_SIGNATURE" {
		t.Errorf("Expected code MISSING_SIGNATURE, got %s", response.Code)
	}
	if response.Received {
		t.Error("Expected received=false")
	}
}

func TestHandleWebhookInvalidSignature(t *testing.T) {
	_, router := setupTestHandler()

	payload := []byte(`{"id": "evt_test", "type": "payment_intent.succeeded"}`)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/webhooks/stripe", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Stripe-Signature", "invalid_signature")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected status 401, got %d", w.Code)
	}

	var response WebhookErrorResponse
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if response.Code != "INVALID_SIGNATURE" {
		t.Errorf("Expected code INVALID_SIGNATURE, got %s", response.Code)
	}
}

func TestHandleWebhookEmptyBody(t *testing.T) {
	_, router := setupTestHandler()

	req := httptest.NewRequest(http.MethodPost, "/api/v1/webhooks/stripe", bytes.NewReader([]byte{}))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Stripe-Signature", "some_signature")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}

	var response WebhookErrorResponse
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if response.Code != "EMPTY_BODY" {
		t.Errorf("Expected code EMPTY_BODY, got %s", response.Code)
	}
}

func TestHandleWebhookMalformedSignature(t *testing.T) {
	_, router := setupTestHandler()

	payload := []byte(`{"id": "evt_test", "type": "payment_intent.succeeded"}`)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/webhooks/stripe", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Stripe-Signature", "t=12345,v1=invalid")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Malformed signature should return 401
	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected status 401, got %d", w.Code)
	}
}

// =============================================================================
// Health Check Tests
// =============================================================================

func TestHealthCheck(t *testing.T) {
	_, router := setupTestHandler()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/webhooks/health", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if response["status"] != "healthy" {
		t.Errorf("Expected status healthy, got %v", response["status"])
	}
	if response["service"] != "stripe-webhook-handler" {
		t.Errorf("Expected service stripe-webhook-handler, got %v", response["service"])
	}
}

// =============================================================================
// Route Registration Tests
// =============================================================================

func TestRegisterRoutes(t *testing.T) {
	handler, _ := setupTestHandler()

	router := gin.New()
	group := router.Group("/api/v1")
	handler.RegisterRoutes(group)

	// Test webhook route exists
	req := httptest.NewRequest(http.MethodPost, "/api/v1/webhooks/stripe", bytes.NewReader([]byte("test")))
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Should get an error (missing signature), but not 404
	if w.Code == http.StatusNotFound {
		t.Error("Webhook route not registered")
	}

	// Test health route exists
	req = httptest.NewRequest(http.MethodGet, "/api/v1/webhooks/health", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Health route not registered or failed: %d", w.Code)
	}
}

func TestRegisterAdminRoutes(t *testing.T) {
	handler, _ := setupTestHandler()

	router := gin.New()
	group := router.Group("/api/v1")
	handler.RegisterAdminRoutes(group)

	// Test events route exists
	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/webhooks/events", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Should get an error (db not configured), but not 404
	if w.Code == http.StatusNotFound {
		t.Error("Admin events route not registered")
	}

	// Test stats route exists
	req = httptest.NewRequest(http.MethodGet, "/api/v1/admin/webhooks/stats", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code == http.StatusNotFound {
		t.Error("Admin stats route not registered")
	}
}

// =============================================================================
// IP Validation Middleware Tests
// =============================================================================

func TestValidateStripeIPMiddleware(t *testing.T) {
	allowedIPs := []string{"192.168.1.1", "10.0.0.1"}

	router := gin.New()
	router.Use(ValidateStripeIP(allowedIPs))
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	tests := []struct {
		name       string
		clientIP   string
		expectCode int
	}{
		{
			name:       "allowed IP",
			clientIP:   "192.168.1.1",
			expectCode: http.StatusOK,
		},
		{
			name:       "another allowed IP",
			clientIP:   "10.0.0.1",
			expectCode: http.StatusOK,
		},
		{
			name:       "blocked IP",
			clientIP:   "8.8.8.8",
			expectCode: http.StatusForbidden,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			req.RemoteAddr = tt.clientIP + ":12345"

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			if w.Code != tt.expectCode {
				t.Errorf("Expected status %d, got %d", tt.expectCode, w.Code)
			}
		})
	}
}

func TestValidateStripeIPMiddlewareEmpty(t *testing.T) {
	// With empty whitelist, all IPs should be allowed
	router := gin.New()
	router.Use(ValidateStripeIP([]string{}))
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.RemoteAddr = "any.ip.address:12345"

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200 with empty whitelist, got %d", w.Code)
	}
}

// =============================================================================
// Response Type Tests
// =============================================================================

func TestWebhookSuccessResponseJSON(t *testing.T) {
	response := WebhookSuccessResponse{
		Received:  true,
		EventID:   "evt_test123",
		EventType: "payment_intent.succeeded",
	}

	data, err := json.Marshal(response)
	if err != nil {
		t.Fatalf("Failed to marshal response: %v", err)
	}

	var parsed WebhookSuccessResponse
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if !parsed.Received {
		t.Error("Expected received=true")
	}
	if parsed.EventID != "evt_test123" {
		t.Errorf("Expected event_id evt_test123, got %s", parsed.EventID)
	}
	if parsed.EventType != "payment_intent.succeeded" {
		t.Errorf("Expected event_type payment_intent.succeeded, got %s", parsed.EventType)
	}
}

func TestWebhookErrorResponseJSON(t *testing.T) {
	response := WebhookErrorResponse{
		Received: false,
		Error:    "Test error",
		Code:     "TEST_ERROR",
	}

	data, err := json.Marshal(response)
	if err != nil {
		t.Fatalf("Failed to marshal response: %v", err)
	}

	var parsed WebhookErrorResponse
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if parsed.Received {
		t.Error("Expected received=false")
	}
	if parsed.Error != "Test error" {
		t.Errorf("Expected error 'Test error', got %s", parsed.Error)
	}
	if parsed.Code != "TEST_ERROR" {
		t.Errorf("Expected code TEST_ERROR, got %s", parsed.Code)
	}
}

// =============================================================================
// Content Type Tests
// =============================================================================

func TestHandleWebhookContentType(t *testing.T) {
	_, router := setupTestHandler()

	payload := []byte(`{"id": "evt_test", "type": "payment_intent.succeeded"}`)

	// Should accept any content type (Stripe doesn't always send application/json)
	contentTypes := []string{
		"application/json",
		"application/json; charset=utf-8",
		"text/plain",
		"",
	}

	for _, ct := range contentTypes {
		t.Run(ct, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/api/v1/webhooks/stripe", bytes.NewReader(payload))
			if ct != "" {
				req.Header.Set("Content-Type", ct)
			}
			req.Header.Set("Stripe-Signature", "t=123,v1=test")

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			// Should fail with invalid signature, not content type error
			if w.Code == http.StatusUnsupportedMediaType {
				t.Errorf("Should not reject content type: %s", ct)
			}
		})
	}
}

// =============================================================================
// Benchmark Tests
// =============================================================================

func BenchmarkHandleWebhookValidation(b *testing.B) {
	_, router := setupTestHandler()

	payload := []byte(`{"id": "evt_test", "type": "payment_intent.succeeded"}`)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/webhooks/stripe", bytes.NewReader(payload))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Stripe-Signature", "invalid_sig")

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
	}
}

func BenchmarkHealthCheck(b *testing.B) {
	_, router := setupTestHandler()

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/webhooks/health", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
	}
}
