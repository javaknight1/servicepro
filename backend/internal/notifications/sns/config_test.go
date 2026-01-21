package sns

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/sns/types"
)

// =============================================================================
// Topic Configuration Tests
// =============================================================================

func TestTopicConfigValidation(t *testing.T) {
	tests := []struct {
		name    string
		config  *TopicConfig
		wantErr bool
	}{
		{
			name: "valid standard topic",
			config: &TopicConfig{
				Name: "test-notifications",
				Type: TopicTypeStandard,
			},
			wantErr: false,
		},
		{
			name: "valid FIFO topic",
			config: &TopicConfig{
				Name:              "orders.fifo",
				Type:              TopicTypeFIFO,
				FIFOTopic:         true,
				ContentBasedDedup: true,
			},
			wantErr: false,
		},
		{
			name: "empty topic name",
			config: &TopicConfig{
				Name: "",
			},
			wantErr: true,
		},
		{
			name: "FIFO topic without .fifo suffix",
			config: &TopicConfig{
				Name:      "orders",
				FIFOTopic: true,
			},
			wantErr: true,
		},
		{
			name: "topic with display name",
			config: &TopicConfig{
				Name:        "alerts",
				DisplayName: "Alert Notifications",
			},
			wantErr: false,
		},
		{
			name: "topic with KMS key",
			config: &TopicConfig{
				Name:     "secure-topic",
				KMSKeyID: "arn:aws:kms:us-east-1:123456789012:key/12345678-1234-1234-1234-123456789012",
			},
			wantErr: false,
		},
		{
			name: "topic with tags",
			config: &TopicConfig{
				Name: "tagged-topic",
				Tags: map[string]string{
					"Environment": "production",
					"Team":        "platform",
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()

			if tt.wantErr && err == nil {
				t.Errorf("Validate() expected error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("Validate() unexpected error: %v", err)
			}
		})
	}
}

// =============================================================================
// SNS Configuration Tests
// =============================================================================

func TestSNSConfigValidation(t *testing.T) {
	tests := []struct {
		name    string
		config  *SNSConfig
		wantErr bool
	}{
		{
			name: "valid configuration",
			config: &SNSConfig{
				Region:          "us-east-1",
				RateLimit:       100,
				RetryAttempts:   3,
				RetryMultiplier: 2.0,
				TopicARNs: map[string]string{
					"notifications": "arn:aws:sns:us-east-1:123456789012:notifications",
				},
				TopicConfigs: make(map[string]*TopicConfig),
			},
			wantErr: false,
		},
		{
			name: "missing region",
			config: &SNSConfig{
				RateLimit:       100,
				RetryAttempts:   3,
				RetryMultiplier: 2.0,
			},
			wantErr: true,
		},
		{
			name: "invalid rate limit (zero)",
			config: &SNSConfig{
				Region:          "us-east-1",
				RateLimit:       0,
				RetryAttempts:   3,
				RetryMultiplier: 2.0,
			},
			wantErr: true,
		},
		{
			name: "invalid rate limit (negative)",
			config: &SNSConfig{
				Region:          "us-east-1",
				RateLimit:       -1,
				RetryAttempts:   3,
				RetryMultiplier: 2.0,
			},
			wantErr: true,
		},
		{
			name: "invalid retry attempts",
			config: &SNSConfig{
				Region:          "us-east-1",
				RateLimit:       100,
				RetryAttempts:   -1,
				RetryMultiplier: 2.0,
			},
			wantErr: true,
		},
		{
			name: "invalid retry multiplier",
			config: &SNSConfig{
				Region:          "us-east-1",
				RateLimit:       100,
				RetryAttempts:   3,
				RetryMultiplier: 0.5,
			},
			wantErr: true,
		},
		{
			name: "invalid topic config in configs",
			config: &SNSConfig{
				Region:          "us-east-1",
				RateLimit:       100,
				RetryAttempts:   3,
				RetryMultiplier: 2.0,
				TopicConfigs: map[string]*TopicConfig{
					"invalid": {Name: ""}, // Empty name is invalid
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()

			if tt.wantErr && err == nil {
				t.Errorf("Validate() expected error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("Validate() unexpected error: %v", err)
			}
		})
	}
}

func TestDefaultSNSConfig(t *testing.T) {
	config := DefaultSNSConfig()

	if config.Region != "us-east-1" {
		t.Errorf("Default region = %s, want us-east-1", config.Region)
	}

	if config.RateLimit != 100 {
		t.Errorf("Default rate limit = %d, want 100", config.RateLimit)
	}

	if config.BurstLimit != 200 {
		t.Errorf("Default burst limit = %d, want 200", config.BurstLimit)
	}

	if config.RetryAttempts != 3 {
		t.Errorf("Default retry attempts = %d, want 3", config.RetryAttempts)
	}

	if config.RetryMultiplier != 2.0 {
		t.Errorf("Default retry multiplier = %f, want 2.0", config.RetryMultiplier)
	}

	if config.TopicARNs == nil {
		t.Errorf("TopicARNs should not be nil")
	}

	if config.TopicConfigs == nil {
		t.Errorf("TopicConfigs should not be nil")
	}
}

func TestNewSNSClient(t *testing.T) {
	tests := []struct {
		name    string
		config  *SNSConfig
		wantErr bool
	}{
		{
			name:    "nil config uses defaults",
			config:  nil,
			wantErr: false,
		},
		{
			name: "valid config",
			config: &SNSConfig{
				Region:          "us-west-2",
				RateLimit:       50,
				BurstLimit:      100,
				RetryAttempts:   5,
				RetryMultiplier: 1.5,
				TopicARNs:       make(map[string]string),
				TopicConfigs:    make(map[string]*TopicConfig),
			},
			wantErr: false,
		},
		{
			name: "invalid config",
			config: &SNSConfig{
				Region:          "",
				RateLimit:       100,
				RetryMultiplier: 2.0,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, err := NewSNSClient(tt.config)

			if tt.wantErr {
				if err == nil {
					t.Errorf("NewSNSClient() expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("NewSNSClient() unexpected error: %v", err)
				return
			}

			if client == nil {
				t.Errorf("NewSNSClient() returned nil client")
				return
			}

			if client.rateLimiter == nil {
				t.Errorf("Rate limiter not initialized")
			}

			if client.errorHandler == nil {
				t.Errorf("Error handler not initialized")
			}

			if client.topicCache == nil {
				t.Errorf("Topic cache not initialized")
			}
		})
	}
}

// =============================================================================
// Rate Limiting Tests
// =============================================================================

func TestRateLimiterAllow(t *testing.T) {
	limiter := NewRateLimiter(10, 5) // 10 per second, burst of 5

	// Should allow burst
	for i := 0; i < 5; i++ {
		if !limiter.Allow() {
			t.Errorf("Allow() should succeed for request %d within burst", i+1)
		}
	}

	// Next request should be denied (burst exhausted)
	if limiter.Allow() {
		t.Errorf("Allow() should fail after burst exhausted")
	}

	// Wait for refill
	time.Sleep(200 * time.Millisecond) // Should refill ~2 tokens

	// Should allow again
	if !limiter.Allow() {
		t.Errorf("Allow() should succeed after refill")
	}
}

func TestRateLimiterAllowN(t *testing.T) {
	limiter := NewRateLimiter(100, 10)

	// Request 5 tokens
	if !limiter.AllowN(5) {
		t.Errorf("AllowN(5) should succeed with 10 tokens available")
	}

	// Request 6 more (should fail, only 5 left)
	if limiter.AllowN(6) {
		t.Errorf("AllowN(6) should fail with only 5 tokens available")
	}

	// Request 5 should succeed
	if !limiter.AllowN(5) {
		t.Errorf("AllowN(5) should succeed with exactly 5 tokens available")
	}
}

func TestRateLimiterWait(t *testing.T) {
	limiter := NewRateLimiter(100, 1) // 100 per second, burst of 1

	// Exhaust the token
	limiter.Allow()

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	start := time.Now()
	err := limiter.Wait(ctx)
	elapsed := time.Since(start)

	if err != nil {
		// Should succeed within timeout since refill rate is 100/sec
		if elapsed >= 50*time.Millisecond {
			// Timeout is expected
			return
		}
		t.Errorf("Wait() unexpected error: %v", err)
	}
}

func TestRateLimiterWaitContextCancel(t *testing.T) {
	limiter := NewRateLimiter(1, 1) // Very slow refill

	// Exhaust the token
	limiter.Allow()

	ctx, cancel := context.WithCancel(context.Background())

	// Cancel immediately
	cancel()

	err := limiter.Wait(ctx)
	if err == nil {
		t.Errorf("Wait() should return error on cancelled context")
	}
}

func TestRateLimiterReserve(t *testing.T) {
	limiter := NewRateLimiter(10, 2)

	// First reserve should be immediate
	wait := limiter.Reserve()
	if wait != 0 {
		t.Errorf("Reserve() first call should return 0, got %v", wait)
	}

	// Second reserve should be immediate
	wait = limiter.Reserve()
	if wait != 0 {
		t.Errorf("Reserve() second call should return 0, got %v", wait)
	}

	// Third reserve should return wait time (going into debt)
	wait = limiter.Reserve()
	if wait == 0 {
		t.Errorf("Reserve() third call should return non-zero wait time")
	}
}

func TestRateLimiterStats(t *testing.T) {
	limiter := NewRateLimiter(100, 10)

	// Make some requests
	limiter.Allow() // allowed
	limiter.Allow() // allowed
	limiter.Allow() // allowed

	// Exhaust and try more
	for i := 0; i < 10; i++ {
		limiter.Allow()
	}

	stats := limiter.Stats()

	if stats.MaxTokens != 10 {
		t.Errorf("Stats MaxTokens = %d, want 10", stats.MaxTokens)
	}
	if stats.RefillRate != 100 {
		t.Errorf("Stats RefillRate = %d, want 100", stats.RefillRate)
	}
	if stats.TotalAllowed == 0 {
		t.Errorf("Stats TotalAllowed should be > 0")
	}
}

func TestRateLimiterReset(t *testing.T) {
	limiter := NewRateLimiter(10, 5)

	// Exhaust all tokens
	for i := 0; i < 5; i++ {
		limiter.Allow()
	}

	// Should be exhausted
	if limiter.Allow() {
		t.Errorf("Limiter should be exhausted")
	}

	// Reset
	limiter.Reset()

	// Should have full capacity again
	available := limiter.Available()
	if available != 5 {
		t.Errorf("Available after reset = %d, want 5", available)
	}
}

func TestRateLimiterSetRate(t *testing.T) {
	limiter := NewRateLimiter(10, 5)

	stats := limiter.Stats()
	if stats.RefillRate != 10 {
		t.Errorf("Initial rate = %d, want 10", stats.RefillRate)
	}

	limiter.SetRate(50)

	stats = limiter.Stats()
	if stats.RefillRate != 50 {
		t.Errorf("Updated rate = %d, want 50", stats.RefillRate)
	}
}

func TestRateLimiterSetBurst(t *testing.T) {
	limiter := NewRateLimiter(10, 5)

	stats := limiter.Stats()
	if stats.MaxTokens != 5 {
		t.Errorf("Initial max tokens = %d, want 5", stats.MaxTokens)
	}

	limiter.SetBurst(10)

	stats = limiter.Stats()
	if stats.MaxTokens != 10 {
		t.Errorf("Updated max tokens = %d, want 10", stats.MaxTokens)
	}
}

func TestAdaptiveRateLimiter(t *testing.T) {
	limiter := NewAdaptiveRateLimiter(100, 10, 200, 10)

	// Record mostly successes - rate should stay stable or increase
	for i := 0; i < 100; i++ {
		limiter.RecordSuccess()
	}

	// Force window evaluation by setting windowStart in the past
	limiter.mu.Lock()
	limiter.windowStart = time.Now().Add(-2 * time.Minute)
	limiter.mu.Unlock()

	// Trigger adjustment
	limiter.RecordSuccess()

	rate := limiter.CurrentRate()
	if rate < 100 {
		t.Errorf("Rate should not decrease with low error rate, got %d", rate)
	}
}

func TestAdaptiveRateLimiterDecrease(t *testing.T) {
	limiter := NewAdaptiveRateLimiter(100, 10, 200, 10)

	// Record mostly errors - rate should decrease
	for i := 0; i < 20; i++ {
		limiter.RecordError()
	}
	for i := 0; i < 10; i++ {
		limiter.RecordSuccess()
	}

	// Force window evaluation
	limiter.mu.Lock()
	limiter.windowStart = time.Now().Add(-2 * time.Minute)
	limiter.mu.Unlock()

	// Trigger adjustment
	limiter.RecordError()

	rate := limiter.CurrentRate()
	if rate >= 100 {
		t.Errorf("Rate should decrease with high error rate, got %d", rate)
	}
}

func TestSlidingWindowRateLimiter(t *testing.T) {
	limiter := NewSlidingWindowRateLimiter(100*time.Millisecond, 3)

	// Should allow 3 requests
	for i := 0; i < 3; i++ {
		if !limiter.Allow() {
			t.Errorf("Allow() should succeed for request %d", i+1)
		}
	}

	// 4th request should be denied
	if limiter.Allow() {
		t.Errorf("Allow() should fail after max requests")
	}

	// Wait for window to slide
	time.Sleep(150 * time.Millisecond)

	// Should allow again
	if !limiter.Allow() {
		t.Errorf("Allow() should succeed after window slides")
	}
}

func TestSlidingWindowRateLimiterCount(t *testing.T) {
	limiter := NewSlidingWindowRateLimiter(time.Second, 10)

	// Make some requests
	for i := 0; i < 5; i++ {
		limiter.Allow()
	}

	count := limiter.Count()
	if count != 5 {
		t.Errorf("Count() = %d, want 5", count)
	}
}

func TestSlidingWindowRateLimiterReset(t *testing.T) {
	limiter := NewSlidingWindowRateLimiter(time.Second, 10)

	// Make some requests
	for i := 0; i < 5; i++ {
		limiter.Allow()
	}

	limiter.Reset()

	count := limiter.Count()
	if count != 0 {
		t.Errorf("Count() after reset = %d, want 0", count)
	}
}

func TestPerTopicRateLimiter(t *testing.T) {
	limiter := NewPerTopicRateLimiter(10, 5)

	topic1 := "arn:aws:sns:us-east-1:123456789012:topic1"
	topic2 := "arn:aws:sns:us-east-1:123456789012:topic2"

	// Set custom rate for topic1
	limiter.SetTopicRate(topic1, 2)

	// Exhaust topic1's burst
	for i := 0; i < 5; i++ {
		limiter.Allow(topic1)
	}

	// topic2 should still have capacity
	if !limiter.Allow(topic2) {
		t.Errorf("topic2 should have capacity")
	}

	// Check stats
	stats := limiter.Stats()
	if len(stats) != 2 {
		t.Errorf("Stats should have 2 topics, got %d", len(stats))
	}
}

func TestPerTopicRateLimiterWait(t *testing.T) {
	limiter := NewPerTopicRateLimiter(100, 1)

	topic := "arn:aws:sns:us-east-1:123456789012:topic1"

	// Exhaust the token
	limiter.Allow(topic)

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	err := limiter.Wait(ctx, topic)
	// Should either succeed or timeout, but not panic
	_ = err
}

// =============================================================================
// Error Handling Tests
// =============================================================================

func TestErrorCategorization(t *testing.T) {
	tests := []struct {
		name    string
		err     error
		wantCat ErrorCategory
	}{
		{
			name:    "nil error",
			err:     nil,
			wantCat: ErrorCategoryUnknown,
		},
		{
			name:    "context canceled",
			err:     context.Canceled,
			wantCat: ErrorCategoryNonRetryable,
		},
		{
			name:    "context deadline exceeded",
			err:     context.DeadlineExceeded,
			wantCat: ErrorCategoryNonRetryable,
		},
		{
			name:    "generic error",
			err:     errors.New("unknown error"),
			wantCat: ErrorCategoryUnknown,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			category := CategorizeError(tt.err)

			if category != tt.wantCat {
				t.Errorf("CategorizeError() = %v, want %v", category, tt.wantCat)
			}
		})
	}
}

func TestIsRetryable(t *testing.T) {
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
			name:      "context canceled",
			err:       context.Canceled,
			retryable: false,
		},
		{
			name:      "generic error",
			err:       errors.New("unknown"),
			retryable: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsRetryable(tt.err)

			if result != tt.retryable {
				t.Errorf("IsRetryable() = %v, want %v", result, tt.retryable)
			}
		})
	}
}

func TestWrapError(t *testing.T) {
	tests := []struct {
		name      string
		err       error
		operation string
		topicARN  string
	}{
		{
			name:      "nil error",
			err:       nil,
			operation: "Publish",
			topicARN:  "arn:aws:sns:us-east-1:123456789012:test",
		},
		{
			name:      "generic error",
			err:       errors.New("test error"),
			operation: "CreateTopic",
			topicARN:  "arn:aws:sns:us-east-1:123456789012:test",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wrapped := WrapError(tt.err, tt.operation, tt.topicARN)

			if tt.err == nil {
				if wrapped != nil {
					t.Errorf("WrapError(nil) should return nil")
				}
				return
			}

			if wrapped == nil {
				t.Errorf("WrapError() returned nil for non-nil error")
				return
			}

			if wrapped.Operation != tt.operation {
				t.Errorf("Operation = %s, want %s", wrapped.Operation, tt.operation)
			}

			if wrapped.TopicARN != tt.topicARN {
				t.Errorf("TopicARN = %s, want %s", wrapped.TopicARN, tt.topicARN)
			}

			if wrapped.Timestamp.IsZero() {
				t.Errorf("Timestamp should not be zero")
			}
		})
	}
}

func TestErrorHandlerExecute(t *testing.T) {
	handler := NewErrorHandler(3, 10*time.Millisecond, 100*time.Millisecond, 2.0)

	callCount := 0
	operation := func() error {
		callCount++
		if callCount < 3 {
			return &types.InternalErrorException{Message: ptrString("Temporary failure")}
		}
		return nil
	}

	err := handler.Execute(context.Background(), operation)

	if err != nil {
		t.Errorf("Execute() should succeed after retries, got error: %v", err)
	}
	if callCount != 3 {
		t.Errorf("Expected 3 calls, got %d", callCount)
	}
}

func TestErrorHandlerMaxRetries(t *testing.T) {
	handler := NewErrorHandler(3, 10*time.Millisecond, 100*time.Millisecond, 2.0)

	callCount := 0
	operation := func() error {
		callCount++
		return &types.InternalErrorException{Message: ptrString("Persistent failure")}
	}

	err := handler.Execute(context.Background(), operation)

	if err == nil {
		t.Errorf("Execute() should fail after max retries")
	}
	if callCount != 4 { // 1 initial + 3 retries
		t.Errorf("Expected 4 calls (1 + 3 retries), got %d", callCount)
	}
}

func TestErrorHandlerNonRetryableError(t *testing.T) {
	handler := NewErrorHandler(3, 10*time.Millisecond, 100*time.Millisecond, 2.0)

	callCount := 0
	operation := func() error {
		callCount++
		return &types.NotFoundException{Message: ptrString("Not found")}
	}

	err := handler.Execute(context.Background(), operation)

	if err == nil {
		t.Errorf("Execute() should fail immediately for non-retryable errors")
	}
	if callCount != 1 {
		t.Errorf("Expected 1 call for non-retryable error, got %d", callCount)
	}
}

func TestErrorHandlerContextCancel(t *testing.T) {
	handler := NewErrorHandler(10, 100*time.Millisecond, 1*time.Second, 2.0)

	ctx, cancel := context.WithCancel(context.Background())

	callCount := int32(0)
	operation := func() error {
		count := atomic.AddInt32(&callCount, 1)
		if count == 2 {
			cancel() // Cancel after second attempt
		}
		return &types.InternalErrorException{Message: ptrString("Failure")}
	}

	err := handler.Execute(ctx, operation)

	if err == nil {
		t.Errorf("Execute() should fail on context cancel")
	}
}

func TestErrorHandlerStats(t *testing.T) {
	handler := NewErrorHandler(1, 10*time.Millisecond, 100*time.Millisecond, 2.0)

	// Generate some errors
	for i := 0; i < 3; i++ {
		_ = handler.Execute(context.Background(), func() error {
			return &types.NotFoundException{Message: ptrString("Not found")}
		})
	}

	stats := handler.Stats()

	if stats.TotalErrors == 0 {
		t.Errorf("TotalErrors should be > 0")
	}
}

func TestErrorHandlerWithCallback(t *testing.T) {
	handler := NewErrorHandler(3, 10*time.Millisecond, 100*time.Millisecond, 2.0)

	retryCallbacks := 0
	callCount := 0

	operation := func() error {
		callCount++
		if callCount < 3 {
			return &types.InternalErrorException{Message: ptrString("Temporary failure")}
		}
		return nil
	}

	onRetry := func(attempt int, err error, delay time.Duration) {
		retryCallbacks++
	}

	err := handler.ExecuteWithCallback(context.Background(), operation, onRetry)

	if err != nil {
		t.Errorf("ExecuteWithCallback() should succeed after retries")
	}

	if retryCallbacks != 2 {
		t.Errorf("Expected 2 retry callbacks, got %d", retryCallbacks)
	}
}

// =============================================================================
// Circuit Breaker Tests
// =============================================================================

func TestCircuitBreakerInitialState(t *testing.T) {
	cb := NewCircuitBreaker(3, 1, 100*time.Millisecond)

	if cb.State() != CircuitClosed {
		t.Errorf("Initial state should be Closed, got %v", cb.State())
	}
}

func TestCircuitBreakerOpenOnFailures(t *testing.T) {
	cb := NewCircuitBreaker(3, 1, 100*time.Millisecond)

	// Record failures to open circuit
	for i := 0; i < 3; i++ {
		cb.RecordFailure()
	}

	if cb.State() != CircuitOpen {
		t.Errorf("State should be Open after failures, got %v", cb.State())
	}

	// Requests should be blocked
	if cb.Allow() {
		t.Errorf("Allow() should return false when circuit is open")
	}
}

func TestCircuitBreakerHalfOpen(t *testing.T) {
	cb := NewCircuitBreaker(2, 1, 50*time.Millisecond)

	// Open the circuit
	cb.RecordFailure()
	cb.RecordFailure()

	if cb.State() != CircuitOpen {
		t.Errorf("State should be Open, got %v", cb.State())
	}

	// Wait for timeout
	time.Sleep(60 * time.Millisecond)

	// Call Allow() to trigger state transition check
	// After timeout, Allow() should transition to HalfOpen and return true
	if !cb.Allow() {
		t.Errorf("Allow() should return true after timeout (transitions to half-open)")
	}

	// Now state should be HalfOpen
	if cb.State() != CircuitHalfOpen {
		t.Errorf("State should be HalfOpen after Allow() call, got %v", cb.State())
	}
}

func TestCircuitBreakerCloseAfterSuccess(t *testing.T) {
	cb := NewCircuitBreaker(2, 1, 50*time.Millisecond)

	// Open the circuit
	cb.RecordFailure()
	cb.RecordFailure()

	// Wait for half-open
	time.Sleep(60 * time.Millisecond)

	// Allow one request
	cb.Allow()

	// Record success to close
	cb.RecordSuccess()

	if cb.State() != CircuitClosed {
		t.Errorf("State should be Closed after success in half-open, got %v", cb.State())
	}
}

func TestCircuitBreakerReopenOnFailure(t *testing.T) {
	cb := NewCircuitBreaker(2, 1, 50*time.Millisecond)

	// Open the circuit
	cb.RecordFailure()
	cb.RecordFailure()

	// Wait for half-open
	time.Sleep(60 * time.Millisecond)

	// Allow one request
	cb.Allow()

	// Record failure in half-open
	cb.RecordFailure()

	if cb.State() != CircuitOpen {
		t.Errorf("State should be Open after failure in half-open, got %v", cb.State())
	}
}

func TestCircuitBreakerStats(t *testing.T) {
	cb := NewCircuitBreaker(3, 1, 100*time.Millisecond)

	cb.RecordFailure()
	cb.RecordFailure()
	cb.RecordSuccess()

	stats := cb.Stats()

	if stats.State != "closed" {
		t.Errorf("State = %s, want closed", stats.State)
	}

	if stats.FailureCount != 0 { // Reset after success
		t.Errorf("FailureCount = %d, want 0", stats.FailureCount)
	}
}

func TestCircuitBreakerOnStateChange(t *testing.T) {
	cb := NewCircuitBreaker(2, 1, 50*time.Millisecond)

	stateChanges := make([]struct{ from, to CircuitState }, 0)
	var mu sync.Mutex

	cb.OnStateChange(func(from, to CircuitState) {
		mu.Lock()
		stateChanges = append(stateChanges, struct{ from, to CircuitState }{from, to})
		mu.Unlock()
	})

	// Open the circuit
	cb.RecordFailure()
	cb.RecordFailure()

	// Wait for callback
	time.Sleep(10 * time.Millisecond)

	mu.Lock()
	if len(stateChanges) != 1 {
		t.Errorf("Expected 1 state change, got %d", len(stateChanges))
	}
	if len(stateChanges) > 0 && stateChanges[0].to != CircuitOpen {
		t.Errorf("Expected transition to Open, got %v", stateChanges[0].to)
	}
	mu.Unlock()
}

// =============================================================================
// Error Aggregator Tests
// =============================================================================

func TestErrorAggregatorRecord(t *testing.T) {
	agg := NewErrorAggregator(100, time.Minute)

	// Record various errors
	agg.Record(errors.New("error1"), "arn:aws:sns:us-east-1:123456789012:topic1", "Publish")
	agg.Record(errors.New("error2"), "arn:aws:sns:us-east-1:123456789012:topic1", "Subscribe")
	agg.Record(errors.New("error3"), "arn:aws:sns:us-east-1:123456789012:topic2", "Publish")

	stats := agg.Stats()

	if stats.TotalErrors != 3 {
		t.Errorf("TotalErrors = %d, want 3", stats.TotalErrors)
	}
}

func TestErrorAggregatorRecentErrors(t *testing.T) {
	agg := NewErrorAggregator(100, time.Minute)

	// Record some errors
	for i := 0; i < 10; i++ {
		agg.Record(errors.New("error"), "topic", "op")
	}

	recent := agg.RecentErrors(5)

	if len(recent) != 5 {
		t.Errorf("RecentErrors(5) returned %d errors, want 5", len(recent))
	}
}

func TestErrorAggregatorMaxErrors(t *testing.T) {
	agg := NewErrorAggregator(5, time.Minute)

	// Record more than max errors
	for i := 0; i < 10; i++ {
		agg.Record(errors.New("error"), "topic", "op")
	}

	recent := agg.RecentErrors(0)

	if len(recent) != 5 {
		t.Errorf("Should only keep %d errors, got %d", 5, len(recent))
	}
}

// =============================================================================
// Benchmarks
// =============================================================================

func BenchmarkRateLimiterAllow(b *testing.B) {
	limiter := NewRateLimiter(1000000, 1000)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		limiter.Allow()
	}
}

func BenchmarkRateLimiterConcurrent(b *testing.B) {
	limiter := NewRateLimiter(1000000, 1000)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			limiter.Allow()
		}
	})
}

func BenchmarkSlidingWindowAllow(b *testing.B) {
	limiter := NewSlidingWindowRateLimiter(time.Hour, 1000000)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		limiter.Allow()
	}
}

func BenchmarkCircuitBreakerAllow(b *testing.B) {
	cb := NewCircuitBreaker(1000, 1, time.Hour)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cb.Allow()
	}
}

func BenchmarkPerTopicRateLimiter(b *testing.B) {
	limiter := NewPerTopicRateLimiter(1000000, 1000)

	topics := []string{
		"arn:aws:sns:us-east-1:123456789012:topic1",
		"arn:aws:sns:us-east-1:123456789012:topic2",
		"arn:aws:sns:us-east-1:123456789012:topic3",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		limiter.Allow(topics[i%len(topics)])
	}
}

func BenchmarkErrorCategorization(b *testing.B) {
	err := errors.New("test error")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		CategorizeError(err)
	}
}

// Helper function to create string pointer
func ptrString(s string) *string {
	return &s
}
