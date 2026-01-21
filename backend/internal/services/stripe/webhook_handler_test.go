package stripe

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stripe/stripe-go/v76"

	"github.com/javaknight1/servicepro/backend/internal/models"
)

// =============================================================================
// Test Helpers
// =============================================================================

// MockLogger implements StripeLogger for testing
type MockLogger struct {
	mu       sync.Mutex
	messages []string
}

func NewMockLogger() *MockLogger {
	return &MockLogger{messages: make([]string, 0)}
}

func (l *MockLogger) Debugf(format string, v ...interface{}) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.messages = append(l.messages, fmt.Sprintf("[DEBUG] "+format, v...))
}

func (l *MockLogger) Infof(format string, v ...interface{}) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.messages = append(l.messages, fmt.Sprintf("[INFO] "+format, v...))
}

func (l *MockLogger) Warnf(format string, v ...interface{}) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.messages = append(l.messages, fmt.Sprintf("[WARN] "+format, v...))
}

func (l *MockLogger) Errorf(format string, v ...interface{}) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.messages = append(l.messages, fmt.Sprintf("[ERROR] "+format, v...))
}

func (l *MockLogger) GetMessages() []string {
	l.mu.Lock()
	defer l.mu.Unlock()
	result := make([]string, len(l.messages))
	copy(result, l.messages)
	return result
}

func (l *MockLogger) Clear() {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.messages = make([]string, 0)
}

// MockNotifier implements WebhookNotifier for testing
type MockNotifier struct {
	mu                        sync.Mutex
	PaymentSucceededCalls     []PaymentSucceededCall
	PaymentFailedCalls        []PaymentFailedCall
	SubscriptionCanceledCalls []SubscriptionCanceledCall
	DisputeCreatedCalls       []DisputeCreatedCall
	RefundCreatedCalls        []RefundCreatedCall
	WebhookErrorCalls         []WebhookErrorCall
	ShouldFail                bool
}

type PaymentSucceededCall struct {
	PaymentID string
	Amount    int64
	Currency  string
}

type PaymentFailedCall struct {
	PaymentID string
	ErrorMsg  string
}

type SubscriptionCanceledCall struct {
	SubscriptionID string
	CustomerID     string
}

type DisputeCreatedCall struct {
	DisputeID string
	Amount    int64
}

type RefundCreatedCall struct {
	RefundID string
	Amount   int64
}

type WebhookErrorCall struct {
	EventID   string
	EventType string
	Error     error
}

func NewMockNotifier() *MockNotifier {
	return &MockNotifier{}
}

func (n *MockNotifier) NotifyPaymentSucceeded(ctx context.Context, paymentID string, amount int64, currency string) error {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.PaymentSucceededCalls = append(n.PaymentSucceededCalls, PaymentSucceededCall{paymentID, amount, currency})
	if n.ShouldFail {
		return fmt.Errorf("notification failed")
	}
	return nil
}

func (n *MockNotifier) NotifyPaymentFailed(ctx context.Context, paymentID string, errorMsg string) error {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.PaymentFailedCalls = append(n.PaymentFailedCalls, PaymentFailedCall{paymentID, errorMsg})
	if n.ShouldFail {
		return fmt.Errorf("notification failed")
	}
	return nil
}

func (n *MockNotifier) NotifySubscriptionCanceled(ctx context.Context, subscriptionID string, customerID string) error {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.SubscriptionCanceledCalls = append(n.SubscriptionCanceledCalls, SubscriptionCanceledCall{subscriptionID, customerID})
	if n.ShouldFail {
		return fmt.Errorf("notification failed")
	}
	return nil
}

func (n *MockNotifier) NotifyDisputeCreated(ctx context.Context, disputeID string, amount int64) error {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.DisputeCreatedCalls = append(n.DisputeCreatedCalls, DisputeCreatedCall{disputeID, amount})
	if n.ShouldFail {
		return fmt.Errorf("notification failed")
	}
	return nil
}

func (n *MockNotifier) NotifyRefundCreated(ctx context.Context, refundID string, amount int64) error {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.RefundCreatedCalls = append(n.RefundCreatedCalls, RefundCreatedCall{refundID, amount})
	if n.ShouldFail {
		return fmt.Errorf("notification failed")
	}
	return nil
}

func (n *MockNotifier) NotifyWebhookError(ctx context.Context, eventID string, eventType string, err error) error {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.WebhookErrorCalls = append(n.WebhookErrorCalls, WebhookErrorCall{eventID, eventType, err})
	return nil
}

// generateStripeSignature generates a Stripe webhook signature for testing
func generateStripeSignature(payload []byte, secret string, timestamp int64) string {
	signedPayload := fmt.Sprintf("%d.%s", timestamp, string(payload))
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(signedPayload))
	sig := hex.EncodeToString(mac.Sum(nil))
	return fmt.Sprintf("t=%d,v1=%s", timestamp, sig)
}

// createTestPaymentIntentEvent creates a test payment intent event
func createTestPaymentIntentEvent(eventType string, paymentID string, amount int64) ([]byte, error) {
	return createTestPaymentIntentEventWithTimestamp(eventType, paymentID, amount, time.Now().Unix())
}

// createTestPaymentIntentEventWithTimestamp creates a test payment intent event with custom timestamp
func createTestPaymentIntentEventWithTimestamp(eventType string, paymentID string, amount int64, created int64) ([]byte, error) {
	pi := stripe.PaymentIntent{
		ID:       paymentID,
		Amount:   amount,
		Currency: "usd",
		Status:   stripe.PaymentIntentStatusSucceeded,
	}

	piData, err := json.Marshal(pi)
	if err != nil {
		return nil, err
	}

	event := stripe.Event{
		ID:         "evt_" + uuid.New().String()[:8],
		Type:       stripe.EventType(eventType),
		APIVersion: "2023-10-16", // Required by stripe-go SDK
		Created:    created,
		Data: &stripe.EventData{
			Raw: piData,
		},
	}

	return json.Marshal(event)
}

// createTestChargeEvent creates a test charge event
func createTestChargeEvent(eventType string, chargeID string, amount int64) ([]byte, error) {
	ch := stripe.Charge{
		ID:       chargeID,
		Amount:   amount,
		Currency: "usd",
		Status:   "succeeded",
		Paid:     true,
	}

	chData, err := json.Marshal(ch)
	if err != nil {
		return nil, err
	}

	event := stripe.Event{
		ID:         "evt_" + uuid.New().String()[:8],
		Type:       stripe.EventType(eventType),
		APIVersion: "2023-10-16", // Required by stripe-go SDK
		Created:    time.Now().Unix(),
		Data: &stripe.EventData{
			Raw: chData,
		},
	}

	return json.Marshal(event)
}

// createTestRefundEvent creates a test refund event
func createTestRefundEvent(eventType string, refundID string, amount int64) ([]byte, error) {
	ref := stripe.Refund{
		ID:       refundID,
		Amount:   amount,
		Currency: "usd",
		Status:   stripe.RefundStatusSucceeded,
	}

	refData, err := json.Marshal(ref)
	if err != nil {
		return nil, err
	}

	event := stripe.Event{
		ID:         "evt_" + uuid.New().String()[:8],
		Type:       stripe.EventType(eventType),
		APIVersion: "2023-10-16", // Required by stripe-go SDK
		Created:    time.Now().Unix(),
		Data: &stripe.EventData{
			Raw: refData,
		},
	}

	return json.Marshal(event)
}

func setupTestService() (*WebhookHandlerService, *MockLogger, *MockNotifier) {
	logger := NewMockLogger()
	notifier := NewMockNotifier()

	config := &WebhookHandlerConfig{
		WebhookSecret:       "whsec_test_secret_key_12345",
		WebhookTolerance:    5 * time.Minute,
		MaxRetries:          3,
		RetryBaseDelay:      100 * time.Millisecond,
		RetryMaxDelay:       1 * time.Second,
		EnableDBLogging:     false, // Disable DB for unit tests
		EnableNotifications: true,
		ProcessingTimeout:   5 * time.Second,
	}

	service := NewWebhookHandlerService(config, nil, logger, notifier)
	return service, logger, notifier
}

// =============================================================================
// Configuration Tests
// =============================================================================

func TestDefaultWebhookHandlerConfig(t *testing.T) {
	config := DefaultWebhookHandlerConfig()

	if config.WebhookTolerance != 5*time.Minute {
		t.Errorf("Expected WebhookTolerance 5m, got %v", config.WebhookTolerance)
	}
	if config.MaxRetries != 3 {
		t.Errorf("Expected MaxRetries 3, got %d", config.MaxRetries)
	}
	if !config.EnableDBLogging {
		t.Error("Expected EnableDBLogging to be true")
	}
	if !config.EnableNotifications {
		t.Error("Expected EnableNotifications to be true")
	}
}

func TestNewWebhookHandlerService(t *testing.T) {
	service, logger, notifier := setupTestService()

	if service == nil {
		t.Fatal("Service should not be nil")
	}
	if service.logger != logger {
		t.Error("Logger not set correctly")
	}
	if service.notifier != notifier {
		t.Error("Notifier not set correctly")
	}
	if service.processor == nil {
		t.Error("Processor should not be nil")
	}
}

func TestNewWebhookHandlerServiceWithNilNotifier(t *testing.T) {
	config := DefaultWebhookHandlerConfig()
	config.WebhookSecret = "test_secret"

	service := NewWebhookHandlerService(config, nil, nil, nil)

	if service.notifier == nil {
		t.Error("Notifier should be set to NoOpNotifier when nil")
	}

	// Should not panic when calling notifier methods
	err := service.notifier.NotifyPaymentSucceeded(context.Background(), "test", 100, "usd")
	if err != nil {
		t.Errorf("NoOpNotifier should not return error: %v", err)
	}
}

// =============================================================================
// Input Validation Tests
// =============================================================================

func TestValidateInput(t *testing.T) {
	service, _, _ := setupTestService()

	tests := []struct {
		name    string
		input   *WebhookInput
		wantErr bool
	}{
		{
			name:    "nil input",
			input:   nil,
			wantErr: true,
		},
		{
			name: "empty payload",
			input: &WebhookInput{
				Payload:   []byte{},
				Signature: "test_sig",
			},
			wantErr: true,
		},
		{
			name: "empty signature",
			input: &WebhookInput{
				Payload:   []byte("test"),
				Signature: "",
			},
			wantErr: true,
		},
		{
			name: "valid input",
			input: &WebhookInput{
				Payload:   []byte("test"),
				Signature: "test_sig",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := service.validateInput(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateInput() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// =============================================================================
// Signature Verification Tests
// =============================================================================

func TestVerifyAndParseEvent(t *testing.T) {
	service, _, _ := setupTestService()

	// Create a valid event
	payload, _ := createTestPaymentIntentEvent(EventPaymentIntentSucceeded, "pi_test123", 1000)
	timestamp := time.Now().Unix()
	signature := generateStripeSignature(payload, "whsec_test_secret_key_12345", timestamp)

	event, err := service.verifyAndParseEvent(payload, signature)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if event == nil {
		t.Fatal("Event should not be nil")
	}
}

func TestVerifyAndParseEventInvalidSignature(t *testing.T) {
	service, _, _ := setupTestService()

	payload, _ := createTestPaymentIntentEvent(EventPaymentIntentSucceeded, "pi_test123", 1000)

	// Invalid signature
	_, err := service.verifyAndParseEvent(payload, "invalid_signature")
	if err == nil {
		t.Error("Expected error for invalid signature")
	}
}

func TestVerifyAndParseEventExpired(t *testing.T) {
	service, _, _ := setupTestService()
	service.config.WebhookTolerance = 1 * time.Second

	// Create event with old Created timestamp in the payload
	payload, _ := createTestPaymentIntentEventWithTimestamp(EventPaymentIntentSucceeded, "pi_test123", 1000, time.Now().Add(-10*time.Second).Unix())

	// Use current timestamp for signature (signature is valid)
	timestamp := time.Now().Unix()
	signature := generateStripeSignature(payload, "whsec_test_secret_key_12345", timestamp)

	_, err := service.verifyAndParseEvent(payload, signature)
	if err != ErrEventExpired {
		t.Errorf("Expected ErrEventExpired, got: %v", err)
	}
}

// =============================================================================
// Idempotency Tests
// =============================================================================

func TestIdempotencyCache(t *testing.T) {
	service, _, _ := setupTestService()

	eventID := "evt_test123"

	// First check - should not be processed
	if service.wasEventProcessed(eventID) {
		t.Error("Event should not be marked as processed initially")
	}

	// Mark as processed
	service.markEventProcessed(eventID)

	// Second check - should be processed
	if !service.wasEventProcessed(eventID) {
		t.Error("Event should be marked as processed")
	}
}

func TestCleanupProcessedEvents(t *testing.T) {
	service, _, _ := setupTestService()

	// Add some events
	service.processedEvents["old_event"] = time.Now().Add(-25 * time.Hour)
	service.processedEvents["new_event"] = time.Now()

	// Run cleanup
	service.cleanupProcessedEvents()

	// Old event should be removed
	if _, exists := service.processedEvents["old_event"]; exists {
		t.Error("Old event should be cleaned up")
	}

	// New event should remain
	if _, exists := service.processedEvents["new_event"]; !exists {
		t.Error("New event should remain")
	}
}

// =============================================================================
// Event Handler Tests
// =============================================================================

func TestHandlePaymentIntentSucceeded(t *testing.T) {
	service, logger, notifier := setupTestService()
	ctx := context.Background()

	// Create event
	pi := stripe.PaymentIntent{
		ID:       "pi_test123",
		Amount:   5000,
		Currency: "usd",
		Status:   stripe.PaymentIntentStatusSucceeded,
	}

	piData, _ := json.Marshal(pi)
	event := &stripe.Event{
		ID:   "evt_test",
		Type: stripe.EventTypePaymentIntentSucceeded,
		Data: &stripe.EventData{
			Raw: piData,
		},
	}

	err := service.handlePaymentIntentSucceeded(ctx, event)
	if err != nil {
		t.Fatalf("Handler returned error: %v", err)
	}

	// Check notification was sent
	if len(notifier.PaymentSucceededCalls) != 1 {
		t.Errorf("Expected 1 notification call, got %d", len(notifier.PaymentSucceededCalls))
	}

	if notifier.PaymentSucceededCalls[0].Amount != 5000 {
		t.Errorf("Expected amount 5000, got %d", notifier.PaymentSucceededCalls[0].Amount)
	}

	// Check logging
	messages := logger.GetMessages()
	hasLog := false
	for _, msg := range messages {
		if contains(msg, "Payment intent succeeded") {
			hasLog = true
			break
		}
	}
	if !hasLog {
		t.Error("Expected payment success log message")
	}
}

func TestHandlePaymentIntentFailed(t *testing.T) {
	service, _, notifier := setupTestService()
	ctx := context.Background()

	pi := stripe.PaymentIntent{
		ID:       "pi_test123",
		Amount:   5000,
		Currency: "usd",
		Status:   stripe.PaymentIntentStatusCanceled,
	}

	piData, _ := json.Marshal(pi)
	event := &stripe.Event{
		ID:   "evt_test",
		Type: stripe.EventTypePaymentIntentPaymentFailed,
		Data: &stripe.EventData{
			Raw: piData,
		},
	}

	err := service.handlePaymentIntentFailed(ctx, event)
	if err != nil {
		t.Fatalf("Handler returned error: %v", err)
	}

	// Check notification was sent
	if len(notifier.PaymentFailedCalls) != 1 {
		t.Errorf("Expected 1 notification call, got %d", len(notifier.PaymentFailedCalls))
	}
}

func TestHandleChargeDisputed(t *testing.T) {
	service, _, notifier := setupTestService()
	ctx := context.Background()

	ch := stripe.Charge{
		ID:       "ch_test123",
		Amount:   5000,
		Currency: "usd",
	}

	chData, _ := json.Marshal(ch)
	event := &stripe.Event{
		ID:   "evt_test",
		Type: stripe.EventType(EventChargeDisputed),
		Data: &stripe.EventData{
			Raw: chData,
		},
	}

	err := service.handleChargeDisputed(ctx, event)
	if err != nil {
		t.Fatalf("Handler returned error: %v", err)
	}

	// Dispute is a critical event - notification should be sent
	if len(notifier.DisputeCreatedCalls) != 1 {
		t.Errorf("Expected 1 dispute notification, got %d", len(notifier.DisputeCreatedCalls))
	}
}

func TestHandleRefundCreated(t *testing.T) {
	service, _, notifier := setupTestService()
	ctx := context.Background()

	ref := stripe.Refund{
		ID:       "re_test123",
		Amount:   2500,
		Currency: "usd",
		Status:   stripe.RefundStatusSucceeded,
	}

	refData, _ := json.Marshal(ref)
	event := &stripe.Event{
		ID:   "evt_test",
		Type: stripe.EventType(EventRefundCreated),
		Data: &stripe.EventData{
			Raw: refData,
		},
	}

	err := service.handleRefundCreated(ctx, event)
	if err != nil {
		t.Fatalf("Handler returned error: %v", err)
	}

	// Refund notification should be sent
	if len(notifier.RefundCreatedCalls) != 1 {
		t.Errorf("Expected 1 refund notification, got %d", len(notifier.RefundCreatedCalls))
	}

	if notifier.RefundCreatedCalls[0].Amount != 2500 {
		t.Errorf("Expected amount 2500, got %d", notifier.RefundCreatedCalls[0].Amount)
	}
}

// =============================================================================
// Retry Logic Tests
// =============================================================================

func TestCalculateNextRetry(t *testing.T) {
	service, _, _ := setupTestService()
	service.config.RetryBaseDelay = 1 * time.Second
	service.config.RetryMaxDelay = 1 * time.Minute

	tests := []struct {
		retryCount int
		minDelay   time.Duration
		maxDelay   time.Duration
	}{
		{0, 900 * time.Millisecond, 1100 * time.Millisecond},  // ~1s
		{1, 1800 * time.Millisecond, 2200 * time.Millisecond}, // ~2s
		{2, 3600 * time.Millisecond, 4400 * time.Millisecond}, // ~4s
		{10, 55 * time.Second, 65 * time.Second},              // Capped at max
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("retry_%d", tt.retryCount), func(t *testing.T) {
			nextRetry := service.calculateNextRetry(tt.retryCount)
			delay := time.Until(nextRetry)

			if delay < tt.minDelay || delay > tt.maxDelay {
				t.Errorf("Delay %v outside expected range [%v, %v]", delay, tt.minDelay, tt.maxDelay)
			}
		})
	}
}

func TestCanRetry(t *testing.T) {
	event := &models.WebhookEvent{
		MaxRetries: 3,
	}

	tests := []struct {
		retryCount int
		canRetry   bool
	}{
		{0, true},
		{1, true},
		{2, true},
		{3, false},
		{4, false},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("retry_%d", tt.retryCount), func(t *testing.T) {
			event.RetryCount = tt.retryCount
			if event.CanRetry() != tt.canRetry {
				t.Errorf("RetryCount=%d, expected CanRetry=%v, got %v",
					tt.retryCount, tt.canRetry, event.CanRetry())
			}
		})
	}
}

// =============================================================================
// Webhook Event Model Tests
// =============================================================================

func TestWebhookEventStatusTransitions(t *testing.T) {
	event := &models.WebhookEvent{
		Status: models.WebhookStatusPending,
	}

	// Mark as processing
	event.MarkAsProcessing()
	if event.Status != models.WebhookStatusProcessing {
		t.Errorf("Expected status processing, got %v", event.Status)
	}

	// Mark as processed
	event.MarkAsProcessed(nil, 100)
	if event.Status != models.WebhookStatusProcessed {
		t.Errorf("Expected status processed, got %v", event.Status)
	}
	if event.ProcessedAt == nil {
		t.Error("ProcessedAt should be set")
	}
	if *event.ProcessingTimeMs != 100 {
		t.Errorf("Expected processing time 100, got %d", *event.ProcessingTimeMs)
	}
}

func TestWebhookEventMarkAsFailed(t *testing.T) {
	event := &models.WebhookEvent{
		Status: models.WebhookStatusProcessing,
	}

	event.MarkAsFailed("test error", "TEST_ERROR")

	if event.Status != models.WebhookStatusFailed {
		t.Errorf("Expected status failed, got %v", event.Status)
	}
	if event.ErrorMessage == nil || *event.ErrorMessage != "test error" {
		t.Error("Error message not set correctly")
	}
	if event.ErrorCode == nil || *event.ErrorCode != "TEST_ERROR" {
		t.Error("Error code not set correctly")
	}
}

func TestWebhookEventMarkForRetry(t *testing.T) {
	event := &models.WebhookEvent{
		Status:     models.WebhookStatusProcessing,
		RetryCount: 1,
	}

	nextRetry := time.Now().Add(time.Minute)
	event.MarkForRetry(nextRetry, "retry error")

	if event.Status != models.WebhookStatusRetrying {
		t.Errorf("Expected status retrying, got %v", event.Status)
	}
	if event.RetryCount != 2 {
		t.Errorf("Expected retry count 2, got %d", event.RetryCount)
	}
	if event.NextRetryAt == nil {
		t.Error("NextRetryAt should be set")
	}
}

func TestWebhookEventIsTerminal(t *testing.T) {
	tests := []struct {
		status     models.WebhookEventStatus
		isTerminal bool
	}{
		{models.WebhookStatusPending, false},
		{models.WebhookStatusProcessing, false},
		{models.WebhookStatusRetrying, false},
		{models.WebhookStatusProcessed, true},
		{models.WebhookStatusFailed, true},
		{models.WebhookStatusSkipped, true},
	}

	for _, tt := range tests {
		t.Run(string(tt.status), func(t *testing.T) {
			event := &models.WebhookEvent{Status: tt.status}
			if event.IsTerminal() != tt.isTerminal {
				t.Errorf("Status %v: expected terminal=%v, got %v",
					tt.status, tt.isTerminal, event.IsTerminal())
			}
		})
	}
}

// =============================================================================
// Concurrent Processing Tests
// =============================================================================

func TestConcurrentEventProcessing(t *testing.T) {
	service, _, _ := setupTestService()
	ctx := context.Background()

	var processedCount int32
	var wg sync.WaitGroup

	// Simulate concurrent webhook processing
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()

			eventID := fmt.Sprintf("evt_concurrent_%d", idx)

			// Check and mark atomically
			if !service.wasEventProcessed(eventID) {
				service.markEventProcessed(eventID)
				atomic.AddInt32(&processedCount, 1)

				// Simulate processing
				time.Sleep(time.Millisecond)

				// Process event (simulated)
				_ = ctx
			}
		}(i)
	}

	wg.Wait()

	// All events should be processed exactly once
	if processedCount != 50 {
		t.Errorf("Expected 50 processed events, got %d", processedCount)
	}

	// Verify cache size
	service.eventMu.RLock()
	cacheSize := len(service.processedEvents)
	service.eventMu.RUnlock()

	if cacheSize != 50 {
		t.Errorf("Expected cache size 50, got %d", cacheSize)
	}
}

// =============================================================================
// Service Lifecycle Tests
// =============================================================================

func TestServiceStartStop(t *testing.T) {
	service, logger, _ := setupTestService()

	// Start service
	service.Start()

	// Give it time to start
	time.Sleep(50 * time.Millisecond)

	// Stop service
	service.Stop()

	// Check logs
	messages := logger.GetMessages()
	hasStartLog := false
	hasStopLog := false

	for _, msg := range messages {
		if contains(msg, "started") {
			hasStartLog = true
		}
		if contains(msg, "stopped") {
			hasStopLog = true
		}
	}

	if !hasStartLog {
		t.Error("Expected start log message")
	}
	if !hasStopLog {
		t.Error("Expected stop log message")
	}
}

// =============================================================================
// NoOpNotifier Tests
// =============================================================================

func TestNoOpNotifier(t *testing.T) {
	notifier := &NoOpNotifier{}
	ctx := context.Background()

	// All methods should return nil without error
	if err := notifier.NotifyPaymentSucceeded(ctx, "test", 100, "usd"); err != nil {
		t.Errorf("NotifyPaymentSucceeded should return nil: %v", err)
	}
	if err := notifier.NotifyPaymentFailed(ctx, "test", "error"); err != nil {
		t.Errorf("NotifyPaymentFailed should return nil: %v", err)
	}
	if err := notifier.NotifySubscriptionCanceled(ctx, "sub", "cust"); err != nil {
		t.Errorf("NotifySubscriptionCanceled should return nil: %v", err)
	}
	if err := notifier.NotifyDisputeCreated(ctx, "dispute", 100); err != nil {
		t.Errorf("NotifyDisputeCreated should return nil: %v", err)
	}
	if err := notifier.NotifyRefundCreated(ctx, "refund", 100); err != nil {
		t.Errorf("NotifyRefundCreated should return nil: %v", err)
	}
	if err := notifier.NotifyWebhookError(ctx, "event", "type", nil); err != nil {
		t.Errorf("NotifyWebhookError should return nil: %v", err)
	}
}

// =============================================================================
// Create Webhook Event Tests
// =============================================================================

func TestCreateWebhookEvent(t *testing.T) {
	service, _, _ := setupTestService()

	stripeEvent := &stripe.Event{
		ID:      "evt_test123",
		Type:    stripe.EventTypePaymentIntentSucceeded,
		Created: time.Now().Unix(),
	}

	input := &WebhookInput{
		Payload:   []byte(`{"test": "payload"}`),
		Signature: "test_sig",
		IPAddress: "192.168.1.1",
		UserAgent: "TestAgent/1.0",
		Headers: map[string]string{
			"Content-Type": "application/json",
		},
	}

	webhookEvent := service.createWebhookEvent(stripeEvent, input)

	if webhookEvent.ID == uuid.Nil {
		t.Error("Webhook event ID should be set")
	}
	if webhookEvent.Source != models.WebhookSourceStripe {
		t.Errorf("Expected source stripe, got %v", webhookEvent.Source)
	}
	if webhookEvent.ExternalEventID != "evt_test123" {
		t.Errorf("Expected external ID evt_test123, got %v", webhookEvent.ExternalEventID)
	}
	if webhookEvent.EventType != string(stripe.EventTypePaymentIntentSucceeded) {
		t.Errorf("Unexpected event type: %v", webhookEvent.EventType)
	}
	if webhookEvent.Status != models.WebhookStatusPending {
		t.Errorf("Expected status pending, got %v", webhookEvent.Status)
	}
	if webhookEvent.IPAddress != "192.168.1.1" {
		t.Errorf("Expected IP 192.168.1.1, got %v", webhookEvent.IPAddress)
	}
	if webhookEvent.UserAgent != "TestAgent/1.0" {
		t.Errorf("Expected user agent TestAgent/1.0, got %v", webhookEvent.UserAgent)
	}
}

// =============================================================================
// Helper Functions
// =============================================================================

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsImpl(s, substr))
}

func containsImpl(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// =============================================================================
// Benchmark Tests
// =============================================================================

func BenchmarkIdempotencyCheck(b *testing.B) {
	service, _, _ := setupTestService()

	// Pre-populate with some events
	for i := 0; i < 1000; i++ {
		service.markEventProcessed(fmt.Sprintf("evt_%d", i))
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		service.wasEventProcessed(fmt.Sprintf("evt_%d", i%1000))
	}
}

func BenchmarkMarkEventProcessed(b *testing.B) {
	service, _, _ := setupTestService()

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		service.markEventProcessed(fmt.Sprintf("evt_%d", i))
	}
}

func BenchmarkValidateInput(b *testing.B) {
	service, _, _ := setupTestService()

	input := &WebhookInput{
		Payload:   make([]byte, 1000),
		Signature: "t=123456789,v1=abcdef1234567890",
		IPAddress: "192.168.1.1",
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		service.validateInput(input)
	}
}
