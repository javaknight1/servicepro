package email

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"
)

// ============================================
// Configuration Tests
// ============================================

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.Region != "us-east-1" {
		t.Errorf("expected default region us-east-1, got %s", cfg.Region)
	}

	if cfg.MaxRetries != 3 {
		t.Errorf("expected max retries 3, got %d", cfg.MaxRetries)
	}

	if cfg.MaxSendRate != 14.0 {
		t.Errorf("expected max send rate 14.0, got %f", cfg.MaxSendRate)
	}

	if !cfg.SandboxMode {
		t.Error("expected sandbox mode to be true by default")
	}

	if !cfg.EnableDKIM {
		t.Error("expected DKIM to be enabled by default")
	}
}

func TestStagingConfig(t *testing.T) {
	cfg := StagingConfig()

	if cfg.Environment != EnvStaging {
		t.Errorf("expected staging environment, got %s", cfg.Environment)
	}

	if cfg.LogLevel != "debug" {
		t.Errorf("expected debug log level for staging, got %s", cfg.LogLevel)
	}

	if !cfg.SandboxMode {
		t.Error("expected sandbox mode for staging")
	}
}

func TestProductionConfig(t *testing.T) {
	cfg := ProductionConfig()

	if cfg.Environment != EnvProduction {
		t.Errorf("expected production environment, got %s", cfg.Environment)
	}

	if cfg.SandboxMode {
		t.Error("expected sandbox mode to be false for production")
	}

	if cfg.MaxSendRate < 50 {
		t.Errorf("expected max send rate >= 50 for production, got %f", cfg.MaxSendRate)
	}

	if cfg.MaxRetries != 5 {
		t.Errorf("expected max retries 5 for production, got %d", cfg.MaxRetries)
	}
}

func TestConfigValidation(t *testing.T) {
	tests := []struct {
		name        string
		modify      func(*SESConfig)
		expectError bool
	}{
		{
			name:        "valid config",
			modify:      func(c *SESConfig) { c.DefaultFromEmail = "test@example.com" },
			expectError: false,
		},
		{
			name:        "missing region",
			modify:      func(c *SESConfig) { c.Region = "" },
			expectError: true,
		},
		{
			name:        "missing from email",
			modify:      func(c *SESConfig) {},
			expectError: true,
		},
		{
			name:        "negative retries",
			modify:      func(c *SESConfig) { c.DefaultFromEmail = "test@example.com"; c.MaxRetries = -1 },
			expectError: true,
		},
		{
			name:        "zero send rate",
			modify:      func(c *SESConfig) { c.DefaultFromEmail = "test@example.com"; c.MaxSendRate = 0 },
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := DefaultConfig()
			tt.modify(cfg)

			err := cfg.Validate()
			if tt.expectError && err == nil {
				t.Error("expected validation error, got nil")
			}
			if !tt.expectError && err != nil {
				t.Errorf("unexpected validation error: %v", err)
			}
		})
	}
}

func TestConfigClone(t *testing.T) {
	cfg := DefaultConfig()
	cfg.DefaultFromEmail = "test@example.com"
	cfg.VerifiedDomains = []string{"example.com", "test.com"}
	cfg.MetricsDimensions["env"] = "test"

	clone := cfg.Clone()

	// Modify original
	cfg.DefaultFromEmail = "modified@example.com"
	cfg.VerifiedDomains[0] = "modified.com"
	cfg.MetricsDimensions["env"] = "modified"

	// Check clone is unchanged
	if clone.DefaultFromEmail == cfg.DefaultFromEmail {
		t.Error("clone should not be affected by original modification")
	}

	if clone.VerifiedDomains[0] == cfg.VerifiedDomains[0] {
		t.Error("clone domains should not be affected by original modification")
	}

	if clone.MetricsDimensions["env"] == cfg.MetricsDimensions["env"] {
		t.Error("clone dimensions should not be affected by original modification")
	}
}

func TestGetFromAddress(t *testing.T) {
	cfg := DefaultConfig()
	cfg.DefaultFromEmail = "support@example.com"
	cfg.DefaultFromName = "Support Team"

	addr := cfg.GetFromAddress()
	expected := "Support Team <support@example.com>"

	if addr != expected {
		t.Errorf("expected %s, got %s", expected, addr)
	}

	// Without name
	cfg.DefaultFromName = ""
	addr = cfg.GetFromAddress()
	if addr != cfg.DefaultFromEmail {
		t.Errorf("expected %s, got %s", cfg.DefaultFromEmail, addr)
	}
}

// ============================================
// Email Validation Tests
// ============================================

func TestIsValidEmail(t *testing.T) {
	tests := []struct {
		email    string
		expected bool
	}{
		{"test@example.com", true},
		{"user.name@domain.co.uk", true},
		{"user+tag@example.com", true},
		{"", false},
		{"notanemail", false},
		{"@nodomain.com", false},
		{"noat.com", false},
		{"multiple@@at.com", false},
		{"a@b", true}, // Minimal valid email
	}

	for _, tt := range tests {
		t.Run(tt.email, func(t *testing.T) {
			result := isValidEmail(tt.email)
			if result != tt.expected {
				t.Errorf("isValidEmail(%s) = %v, expected %v", tt.email, result, tt.expected)
			}
		})
	}
}

// ============================================
// Error Handler Tests
// ============================================

func TestErrorHandlerIsRetryable(t *testing.T) {
	handler := NewSESErrorHandler(DefaultConfig())

	tests := []struct {
		name      string
		err       error
		retryable bool
	}{
		{
			name:      "nil error",
			err:       nil,
			retryable: false,
		},
		{
			name:      "generic error",
			err:       errors.New("generic error"),
			retryable: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := handler.IsRetryable(tt.err)
			if result != tt.retryable {
				t.Errorf("IsRetryable() = %v, expected %v", result, tt.retryable)
			}
		})
	}
}

func TestSESErrorWrap(t *testing.T) {
	handler := NewSESErrorHandler(DefaultConfig())

	// Test wrapping a regular error
	originalErr := errors.New("original error")
	wrappedErr := handler.Wrap(originalErr)

	sesErr, ok := wrappedErr.(*SESError)
	if !ok {
		t.Fatal("expected wrapped error to be *SESError")
	}

	if sesErr.Cause != originalErr {
		t.Error("expected cause to be original error")
	}

	// Test that already wrapped errors are returned as-is
	rewrapped := handler.Wrap(sesErr)
	if rewrapped != sesErr {
		t.Error("expected already wrapped error to be returned as-is")
	}

	// Test nil error
	if handler.Wrap(nil) != nil {
		t.Error("expected nil error to return nil")
	}
}

func TestGetErrorMessage(t *testing.T) {
	handler := NewSESErrorHandler(DefaultConfig())

	// Test with unknown error code
	msg := handler.GetErrorMessage(errors.New("unknown error"))
	if msg == "" {
		t.Error("expected non-empty error message")
	}
}

func TestClassifyError(t *testing.T) {
	handler := NewSESErrorHandler(DefaultConfig())

	// Test with generic error
	classification := handler.ClassifyError(errors.New("generic error"))
	if classification != "unknown" {
		t.Errorf("expected 'unknown' classification, got %s", classification)
	}
}

// ============================================
// Logging Tests
// ============================================

func TestSESLogger(t *testing.T) {
	cfg := DefaultConfig()
	cfg.LogLevel = "debug"
	cfg.LogFormat = "json"

	logger := NewSESLogger(cfg)

	// Capture output
	var buf bytes.Buffer
	logger.SetOutput(&buf)

	// Test each log level
	logger.Debug("debug message", "key", "value")
	logger.Info("info message", "count", 42)
	logger.Warn("warn message")
	logger.Error("error message", "error", errors.New("test error"))

	output := buf.String()

	// Check JSON format
	if !strings.Contains(output, `"level":"DEBUG"`) {
		t.Error("expected DEBUG level in output")
	}
	if !strings.Contains(output, `"level":"INFO"`) {
		t.Error("expected INFO level in output")
	}
	if !strings.Contains(output, `"level":"WARN"`) {
		t.Error("expected WARN level in output")
	}
	if !strings.Contains(output, `"level":"ERROR"`) {
		t.Error("expected ERROR level in output")
	}
}

func TestSESLoggerLevel(t *testing.T) {
	cfg := DefaultConfig()
	cfg.LogLevel = "warn"
	cfg.LogFormat = "text"

	logger := NewSESLogger(cfg)

	var buf bytes.Buffer
	logger.SetOutput(&buf)

	logger.Debug("debug message")
	logger.Info("info message")
	logger.Warn("warn message")
	logger.Error("error message")

	output := buf.String()

	// Debug and Info should be filtered out
	if strings.Contains(output, "debug message") {
		t.Error("debug message should be filtered")
	}
	if strings.Contains(output, "info message") {
		t.Error("info message should be filtered")
	}
	if !strings.Contains(output, "warn message") {
		t.Error("warn message should be present")
	}
	if !strings.Contains(output, "error message") {
		t.Error("error message should be present")
	}
}

func TestSESLoggerWithFields(t *testing.T) {
	cfg := DefaultConfig()
	cfg.LogFormat = "json"
	logger := NewSESLogger(cfg)

	var buf bytes.Buffer
	logger.SetOutput(&buf)

	// Create logger with fields
	loggerWithFields := logger.WithField("request_id", "12345")
	loggerWithFields.Info("test message")

	output := buf.String()
	if !strings.Contains(output, "12345") {
		t.Error("expected request_id in output")
	}
}

func TestParseLogLevel(t *testing.T) {
	tests := []struct {
		input    string
		expected LogLevel
	}{
		{"debug", LogLevelDebug},
		{"DEBUG", LogLevelDebug},
		{"info", LogLevelInfo},
		{"INFO", LogLevelInfo},
		{"warn", LogLevelWarn},
		{"WARN", LogLevelWarn},
		{"warning", LogLevelWarn},
		{"error", LogLevelError},
		{"ERROR", LogLevelError},
		{"unknown", LogLevelInfo}, // default
		{"", LogLevelInfo},        // default
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := ParseLogLevel(tt.input)
			if result != tt.expected {
				t.Errorf("ParseLogLevel(%s) = %v, expected %v", tt.input, result, tt.expected)
			}
		})
	}
}

// ============================================
// Audit Log Tests
// ============================================

func TestAuditLog(t *testing.T) {
	cfg := DefaultConfig()
	audit := NewAuditLog(cfg)

	// Test that logging doesn't panic
	email := &Email{
		To:      []string{"test@example.com"},
		Subject: "Test Subject",
	}

	result := &SendResult{
		MessageID: "test-message-id",
		Success:   true,
		Duration:  100 * time.Millisecond,
		Attempts:  1,
	}

	audit.LogEmailSent(email, result)
	audit.LogEmailFailed(email, errors.New("test error"), 3)
	audit.LogDomainVerified("example.com", "success")
	audit.LogConfigurationChange("admin", "max_retries", "3", "5")
}

func TestAuditLogDisabled(t *testing.T) {
	cfg := DefaultConfig()
	audit := NewAuditLog(cfg)
	audit.SetEnabled(false)

	// These should not panic even when disabled
	email := &Email{
		To:      []string{"test@example.com"},
		Subject: "Test Subject",
	}

	result := &SendResult{
		MessageID: "test-message-id",
		Success:   true,
	}

	audit.LogEmailSent(email, result)
	audit.LogEmailFailed(email, errors.New("test error"), 3)
}

// ============================================
// Domain Manager Tests
// ============================================

func TestGetDomainVerificationRecord(t *testing.T) {
	cfg := DefaultConfig()
	ctx := context.Background()

	// Create domain manager (will fail without AWS credentials, but we can test helpers)
	manager := &DomainManager{
		config: cfg,
		logger: NewSESLogger(cfg),
	}

	record := manager.GetDomainVerificationRecord("example.com", "abc123token")

	if record.Name != "_amazonses.example.com" {
		t.Errorf("expected _amazonses.example.com, got %s", record.Name)
	}

	if record.Type != "TXT" {
		t.Errorf("expected TXT record type, got %s", record.Type)
	}

	if record.Value != "abc123token" {
		t.Errorf("expected abc123token, got %s", record.Value)
	}

	_ = ctx // Used to show this test doesn't need actual AWS connection
}

func TestGetDKIMRecords(t *testing.T) {
	cfg := DefaultConfig()
	manager := &DomainManager{
		config: cfg,
		logger: NewSESLogger(cfg),
	}

	tokens := []string{"token1", "token2", "token3"}
	records := manager.GetDKIMRecords("example.com", tokens)

	if len(records) != 3 {
		t.Errorf("expected 3 DKIM records, got %d", len(records))
	}

	expectedNames := []string{
		"token1._domainkey.example.com",
		"token2._domainkey.example.com",
		"token3._domainkey.example.com",
	}

	for i, record := range records {
		if record.Name != expectedNames[i] {
			t.Errorf("expected name %s, got %s", expectedNames[i], record.Name)
		}
		if record.Type != "CNAME" {
			t.Errorf("expected CNAME record type, got %s", record.Type)
		}
		expectedValue := fmt.Sprintf("%s.dkim.amazonses.com", tokens[i])
		if record.Value != expectedValue {
			t.Errorf("expected value %s, got %s", expectedValue, record.Value)
		}
	}
}

func TestGetMailFromDomainRecords(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Region = "us-east-1"
	manager := &DomainManager{
		config: cfg,
		logger: NewSESLogger(cfg),
	}

	records := manager.GetMailFromDomainRecords("mail.example.com", "us-east-1")

	if len(records) != 2 {
		t.Errorf("expected 2 records, got %d", len(records))
	}

	// Check MX record
	mxFound := false
	spfFound := false
	for _, record := range records {
		if record.Type == "MX" {
			mxFound = true
			if !strings.Contains(record.Value, "feedback-smtp.us-east-1.amazonses.com") {
				t.Errorf("unexpected MX value: %s", record.Value)
			}
		}
		if record.Type == "TXT" {
			spfFound = true
			if !strings.Contains(record.Value, "include:amazonses.com") {
				t.Errorf("unexpected SPF value: %s", record.Value)
			}
		}
	}

	if !mxFound {
		t.Error("MX record not found")
	}
	if !spfFound {
		t.Error("SPF record not found")
	}
}

// ============================================
// Bulk Error Result Tests
// ============================================

func TestBulkErrorResult(t *testing.T) {
	result := &BulkErrorResult{
		SuccessCount: 8,
		FailureCount: 2,
		Errors: []BulkErrorItem{
			{Index: 2, Email: "bad1@example.com", Error: errors.New("error 1")},
			{Index: 5, Email: "bad2@example.com", Error: errors.New("error 2")},
		},
	}

	if !result.HasErrors() {
		t.Error("expected HasErrors to return true")
	}

	summary := result.Summary()
	if !strings.Contains(summary, "8 succeeded") {
		t.Error("expected summary to contain success count")
	}
	if !strings.Contains(summary, "2 failed") {
		t.Error("expected summary to contain failure count")
	}
}

func TestBulkErrorResultGetRetryable(t *testing.T) {
	handler := NewSESErrorHandler(DefaultConfig())

	result := &BulkErrorResult{
		Errors: []BulkErrorItem{
			{Index: 0, Error: errors.New("retryable error")},
			{Index: 1, Error: errors.New("another error")},
		},
	}

	retryable := result.GetRetryableErrors(handler)

	// All generic errors are considered retryable
	if len(retryable) != 2 {
		t.Errorf("expected 2 retryable errors, got %d", len(retryable))
	}
}

// ============================================
// Rate Limiter Tests
// ============================================

func TestRateLimiter(t *testing.T) {
	limiter := newRateLimiter(10.0) // 10 per second

	ctx := context.Background()

	// Should be able to consume immediately
	start := time.Now()
	err := limiter.wait(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	elapsed := time.Since(start)

	if elapsed > 100*time.Millisecond {
		t.Errorf("first request should be immediate, took %v", elapsed)
	}
}

func TestRateLimiterContextCancellation(t *testing.T) {
	limiter := newRateLimiter(0.1) // Very slow rate

	// Consume all tokens
	ctx := context.Background()
	for i := 0; i < 10; i++ {
		limiter.wait(ctx)
	}

	// Try with cancelled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	err := limiter.wait(ctx)
	if err != context.Canceled {
		t.Errorf("expected context.Canceled, got %v", err)
	}
}

// ============================================
// SendResult Tests
// ============================================

func TestSendResult(t *testing.T) {
	result := &SendResult{
		MessageID: "test-123",
		Success:   true,
		Attempts:  2,
		Duration:  150 * time.Millisecond,
		Timestamp: time.Now(),
	}

	if !result.Success {
		t.Error("expected success to be true")
	}

	if result.MessageID != "test-123" {
		t.Errorf("expected message ID test-123, got %s", result.MessageID)
	}
}

// ============================================
// Verification Status Tests
// ============================================

func TestVerificationStatusString(t *testing.T) {
	tests := []struct {
		status   VerificationStatus
		expected string
	}{
		{VerificationPending, "Pending"},
		{VerificationSuccess, "Success"},
		{VerificationFailed, "Failed"},
		{VerificationTemporary, "TemporaryFailure"},
		{VerificationNotStarted, "NotStarted"},
	}

	for _, tt := range tests {
		if string(tt.status) != tt.expected {
			t.Errorf("expected %s, got %s", tt.expected, tt.status)
		}
	}
}

// ============================================
// Integration Test Helpers
// ============================================

// MockSESClient is a mock implementation for testing
type MockSESClient struct {
	SendFunc func(ctx context.Context, email *Email) (*SendResult, error)
}

func (m *MockSESClient) Send(ctx context.Context, email *Email) (*SendResult, error) {
	if m.SendFunc != nil {
		return m.SendFunc(ctx, email)
	}
	return &SendResult{
		MessageID: "mock-message-id",
		Success:   true,
		Attempts:  1,
		Duration:  10 * time.Millisecond,
		Timestamp: time.Now(),
	}, nil
}

func TestMockSESClient(t *testing.T) {
	mock := &MockSESClient{
		SendFunc: func(ctx context.Context, email *Email) (*SendResult, error) {
			return &SendResult{
				MessageID: "custom-id",
				Success:   true,
			}, nil
		},
	}

	ctx := context.Background()
	email := &Email{
		To:      []string{"test@example.com"},
		Subject: "Test",
	}

	result, err := mock.Send(ctx, email)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.MessageID != "custom-id" {
		t.Errorf("expected custom-id, got %s", result.MessageID)
	}
}

// ============================================
// Benchmark Tests
// ============================================

func BenchmarkIsValidEmail(b *testing.B) {
	email := "user@example.com"
	for i := 0; i < b.N; i++ {
		isValidEmail(email)
	}
}

func BenchmarkConfigValidation(b *testing.B) {
	cfg := DefaultConfig()
	cfg.DefaultFromEmail = "test@example.com"

	for i := 0; i < b.N; i++ {
		cfg.Validate()
	}
}

func BenchmarkLoggerJSON(b *testing.B) {
	cfg := DefaultConfig()
	cfg.LogFormat = "json"
	logger := NewSESLogger(cfg)

	var buf bytes.Buffer
	logger.SetOutput(&buf)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		logger.Info("test message", "key1", "value1", "key2", 42)
		buf.Reset()
	}
}

func BenchmarkLoggerText(b *testing.B) {
	cfg := DefaultConfig()
	cfg.LogFormat = "text"
	logger := NewSESLogger(cfg)

	var buf bytes.Buffer
	logger.SetOutput(&buf)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		logger.Info("test message", "key1", "value1", "key2", 42)
		buf.Reset()
	}
}
