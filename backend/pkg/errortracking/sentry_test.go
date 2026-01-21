package errortracking

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/getsentry/sentry-go"
)

// MockTransport is a mock transport for testing
type MockTransport struct {
	events []*sentry.Event
}

func (t *MockTransport) Flush(timeout time.Duration) bool {
	return true
}

func (t *MockTransport) FlushWithContext(ctx context.Context) bool {
	return true
}

func (t *MockTransport) Configure(options sentry.ClientOptions) {}

func (t *MockTransport) SendEvent(event *sentry.Event) {
	t.events = append(t.events, event)
}

func (t *MockTransport) Close() {}

func setupTestSentry(t *testing.T) *MockTransport {
	transport := &MockTransport{}

	err := sentry.Init(sentry.ClientOptions{
		Dsn:              "",
		Transport:        transport,
		AttachStacktrace: true,
	})
	if err != nil {
		t.Fatalf("Failed to initialize Sentry: %v", err)
	}

	return transport
}

func TestCaptureException(t *testing.T) {
	transport := setupTestSentry(t)
	defer sentry.Flush(time.Second)

	testErr := errors.New("test error")
	eventID := CaptureException(testErr)

	// Flush to ensure event is sent
	sentry.Flush(time.Second)

	if eventID == nil {
		t.Error("Expected event ID, got nil")
	}

	if len(transport.events) == 0 {
		t.Error("Expected event to be captured")
	}
}

func TestCaptureError_WithTags(t *testing.T) {
	transport := setupTestSentry(t)
	defer sentry.Flush(time.Second)

	testErr := errors.New("tagged error")
	CaptureError(testErr,
		WithTag("component", "test"),
		WithTag("severity", "high"),
	)

	sentry.Flush(time.Second)

	if len(transport.events) == 0 {
		t.Fatal("Expected event to be captured")
	}

	event := transport.events[len(transport.events)-1]
	if event.Tags["component"] != "test" {
		t.Errorf("Expected tag 'component'='test', got '%s'", event.Tags["component"])
	}
}

func TestCaptureError_WithContext(t *testing.T) {
	transport := setupTestSentry(t)
	defer sentry.Flush(time.Second)

	testErr := errors.New("context error")
	CaptureError(testErr,
		WithContext("user_id", "12345"),
		WithContext("order_id", "ORDER-001"),
	)

	sentry.Flush(time.Second)

	if len(transport.events) == 0 {
		t.Fatal("Expected event to be captured")
	}

	event := transport.events[len(transport.events)-1]
	if event.Extra["user_id"] != "12345" {
		t.Errorf("Expected extra 'user_id'='12345', got '%v'", event.Extra["user_id"])
	}
}

func TestCaptureError_WithUser(t *testing.T) {
	transport := setupTestSentry(t)
	defer sentry.Flush(time.Second)

	testErr := errors.New("user error")
	CaptureError(testErr,
		WithUser("user-123", "user@example.com", "testuser"),
	)

	sentry.Flush(time.Second)

	if len(transport.events) == 0 {
		t.Fatal("Expected event to be captured")
	}

	event := transport.events[len(transport.events)-1]
	if event.User.ID != "user-123" {
		t.Errorf("Expected user ID 'user-123', got '%s'", event.User.ID)
	}
	if event.User.Email != "user@example.com" {
		t.Errorf("Expected user email 'user@example.com', got '%s'", event.User.Email)
	}
}

func TestCaptureError_WithLevel(t *testing.T) {
	transport := setupTestSentry(t)
	defer sentry.Flush(time.Second)

	testErr := errors.New("warning error")
	CaptureError(testErr,
		WithLevel(sentry.LevelWarning),
	)

	sentry.Flush(time.Second)

	if len(transport.events) == 0 {
		t.Fatal("Expected event to be captured")
	}

	event := transport.events[len(transport.events)-1]
	if event.Level != sentry.LevelWarning {
		t.Errorf("Expected level 'warning', got '%s'", event.Level)
	}
}

func TestCaptureError_WithFingerprint(t *testing.T) {
	transport := setupTestSentry(t)
	defer sentry.Flush(time.Second)

	testErr := errors.New("fingerprinted error")
	CaptureError(testErr,
		WithFingerprint("custom", "fingerprint"),
	)

	sentry.Flush(time.Second)

	if len(transport.events) == 0 {
		t.Fatal("Expected event to be captured")
	}

	event := transport.events[len(transport.events)-1]
	if len(event.Fingerprint) != 2 {
		t.Errorf("Expected 2 fingerprint elements, got %d", len(event.Fingerprint))
	}
}

func TestCaptureError_WithRequestContext(t *testing.T) {
	transport := setupTestSentry(t)
	defer sentry.Flush(time.Second)

	req := httptest.NewRequest("POST", "/api/test", nil)
	req.Header.Set("User-Agent", "TestAgent/1.0")

	testErr := errors.New("request error")
	CaptureError(testErr,
		WithRequestContext(req),
	)

	sentry.Flush(time.Second)

	if len(transport.events) == 0 {
		t.Fatal("Expected event to be captured")
	}

	event := transport.events[len(transport.events)-1]
	reqContext, ok := event.Extra["request"].(map[string]interface{})
	if !ok {
		t.Fatal("Expected request context in extra")
	}

	if reqContext["method"] != "POST" {
		t.Errorf("Expected method 'POST', got '%v'", reqContext["method"])
	}
}

func TestCaptureMessage(t *testing.T) {
	transport := setupTestSentry(t)
	defer sentry.Flush(time.Second)

	eventID := CaptureMessage("Test message")

	sentry.Flush(time.Second)

	if eventID == nil {
		t.Error("Expected event ID, got nil")
	}

	if len(transport.events) == 0 {
		t.Fatal("Expected event to be captured")
	}

	event := transport.events[len(transport.events)-1]
	if event.Message != "Test message" {
		t.Errorf("Expected message 'Test message', got '%s'", event.Message)
	}
}

func TestCaptureException_NilError(t *testing.T) {
	setupTestSentry(t)
	defer sentry.Flush(time.Second)

	eventID := CaptureException(nil)

	if eventID != nil {
		t.Error("Expected nil event ID for nil error")
	}
}

func TestSanitizeEvent(t *testing.T) {
	event := &sentry.Event{
		Request: &sentry.Request{
			Headers: map[string]string{
				"Authorization": "Bearer secret-token",
				"Cookie":        "session=abc123",
				"Content-Type":  "application/json",
			},
		},
		Extra: map[string]interface{}{
			"password":    "secret123",
			"api_key":     "key123",
			"normal_data": "visible",
		},
	}

	sanitized := sanitizeEvent(event)

	// Check headers are sanitized
	if sanitized.Request.Headers["Authorization"] != "[REDACTED]" {
		t.Error("Authorization header should be redacted")
	}
	if sanitized.Request.Headers["Cookie"] != "[REDACTED]" {
		t.Error("Cookie header should be redacted")
	}
	if sanitized.Request.Headers["Content-Type"] != "application/json" {
		t.Error("Content-Type header should not be redacted")
	}

	// Check extra data is sanitized
	if sanitized.Extra["password"] != "[REDACTED]" {
		t.Error("password field should be redacted")
	}
	if sanitized.Extra["api_key"] != "[REDACTED]" {
		t.Error("api_key field should be redacted")
	}
	if sanitized.Extra["normal_data"] != "visible" {
		t.Error("normal_data field should not be redacted")
	}
}

func TestShouldFilterError(t *testing.T) {
	tests := []struct {
		name     string
		errMsg   string
		expected bool
	}{
		{"context canceled", "context canceled", true},
		{"client disconnected", "client disconnected: write failed", true},
		{"broken pipe", "write: broken pipe", true},
		{"connection reset", "connection reset by peer", true},
		{"normal error", "something went wrong", false},
		{"database error", "database connection failed", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			event := &sentry.Event{}
			hint := &sentry.EventHint{
				OriginalException: errors.New(tt.errMsg),
			}

			result := shouldFilterError(event, hint)
			if result != tt.expected {
				t.Errorf("Expected %v for error '%s', got %v", tt.expected, tt.errMsg, result)
			}
		})
	}
}

func TestStartSpan(t *testing.T) {
	setupTestSentry(t)
	defer sentry.Flush(time.Second)

	ctx := context.Background()
	transaction := StartTransaction(ctx, "test-transaction", "test")
	defer transaction.Finish()

	span := StartSpan(transaction.Context(), "test-operation")
	if span == nil {
		t.Error("Expected span to be created")
	}
	span.Finish()
}

func TestRecoverWithContext_NoPanic(t *testing.T) {
	setupTestSentry(t)
	defer sentry.Flush(time.Second)

	ctx := context.Background()

	// This should not panic
	func() {
		defer RecoverWithContext(ctx)
		// Normal execution
	}()
}

func TestFlush(t *testing.T) {
	setupTestSentry(t)

	result := Flush(time.Second)
	if !result {
		t.Error("Expected flush to succeed")
	}
}

func TestGetClientIP(t *testing.T) {
	tests := []struct {
		name          string
		xForwardedFor string
		xRealIP       string
		remoteAddr    string
		expected      string
	}{
		{
			name:          "X-Forwarded-For present",
			xForwardedFor: "203.0.113.195",
			remoteAddr:    "192.168.1.1:12345",
			expected:      "203.0.113.195",
		},
		{
			name:       "X-Real-IP present",
			xRealIP:    "203.0.113.195",
			remoteAddr: "192.168.1.1:12345",
			expected:   "203.0.113.195",
		},
		{
			name:       "Only RemoteAddr",
			remoteAddr: "192.168.1.1:12345",
			expected:   "192.168.1.1:12345",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, _ := http.NewRequest("GET", "/", nil)
			if tt.xForwardedFor != "" {
				req.Header.Set("X-Forwarded-For", tt.xForwardedFor)
			}
			if tt.xRealIP != "" {
				req.Header.Set("X-Real-IP", tt.xRealIP)
			}
			req.RemoteAddr = tt.remoteAddr

			result := getClientIP(req)
			if result != tt.expected {
				t.Errorf("Expected '%s', got '%s'", tt.expected, result)
			}
		})
	}
}
