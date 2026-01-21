package sns

import (
	"context"
	"errors"
	"fmt"
	"math"
	"math/rand"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/sns/types"
	"github.com/aws/smithy-go"
)

// SNS-specific error types
var (
	ErrThrottling         = errors.New("request throttled by AWS")
	ErrInternalError      = errors.New("AWS internal error")
	ErrAuthorizationError = errors.New("authorization error")
	ErrEndpointDisabled   = errors.New("endpoint is disabled")
	ErrInvalidParameter   = errors.New("invalid parameter")
	ErrNotAuthorized      = errors.New("not authorized")
	ErrPlatformDisabled   = errors.New("platform application disabled")
	ErrKMSAccessDenied    = errors.New("KMS access denied")
	ErrKMSDisabled        = errors.New("KMS key disabled")
	ErrKMSInvalidState    = errors.New("KMS key in invalid state")
	ErrKMSNotFound        = errors.New("KMS key not found")
	ErrKMSOptInRequired   = errors.New("KMS opt-in required")
	ErrKMSThrottling      = errors.New("KMS throttling")
	ErrNetworkError       = errors.New("network error")
	ErrTimeout            = errors.New("operation timed out")
	ErrContextCanceled    = errors.New("operation canceled")
	ErrMaxRetriesExceeded = errors.New("maximum retries exceeded")
)

// ErrorCategory categorizes errors for handling decisions
type ErrorCategory int

const (
	ErrorCategoryUnknown ErrorCategory = iota
	ErrorCategoryRetryable
	ErrorCategoryNonRetryable
	ErrorCategoryThrottling
	ErrorCategoryAuth
	ErrorCategoryNetwork
	ErrorCategoryInternal
)

// String returns a string representation of the error category
func (c ErrorCategory) String() string {
	switch c {
	case ErrorCategoryRetryable:
		return "retryable"
	case ErrorCategoryNonRetryable:
		return "non_retryable"
	case ErrorCategoryThrottling:
		return "throttling"
	case ErrorCategoryAuth:
		return "auth"
	case ErrorCategoryNetwork:
		return "network"
	case ErrorCategoryInternal:
		return "internal"
	default:
		return "unknown"
	}
}

// SNSError wraps errors with additional context
type SNSError struct {
	OriginalError error
	Category      ErrorCategory
	Retryable     bool
	Message       string
	Code          string
	RequestID     string
	TopicARN      string
	Operation     string
	Timestamp     time.Time
	RetryCount    int
}

// Error implements the error interface
func (e *SNSError) Error() string {
	parts := []string{e.Message}

	if e.Code != "" {
		parts = append(parts, fmt.Sprintf("code=%s", e.Code))
	}
	if e.TopicARN != "" {
		parts = append(parts, fmt.Sprintf("topic=%s", e.TopicARN))
	}
	if e.RequestID != "" {
		parts = append(parts, fmt.Sprintf("request_id=%s", e.RequestID))
	}

	return strings.Join(parts, ", ")
}

// Unwrap returns the underlying error
func (e *SNSError) Unwrap() error {
	return e.OriginalError
}

// Is checks if the error matches a target
func (e *SNSError) Is(target error) bool {
	return errors.Is(e.OriginalError, target)
}

// CategorizeError determines the category of an error
func CategorizeError(err error) ErrorCategory {
	if err == nil {
		return ErrorCategoryUnknown
	}

	// Check for context errors
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return ErrorCategoryNonRetryable
	}

	// Check for network errors
	var netErr net.Error
	if errors.As(err, &netErr) {
		if netErr.Timeout() {
			return ErrorCategoryNetwork
		}
		return ErrorCategoryRetryable
	}

	// Check for AWS API errors
	var apiErr smithy.APIError
	if errors.As(err, &apiErr) {
		code := apiErr.ErrorCode()
		return categorizeAWSError(code)
	}

	// Check for specific SNS exceptions
	var throttle *types.ThrottledException
	if errors.As(err, &throttle) {
		return ErrorCategoryThrottling
	}

	var authErr *types.AuthorizationErrorException
	if errors.As(err, &authErr) {
		return ErrorCategoryAuth
	}

	var internalErr *types.InternalErrorException
	if errors.As(err, &internalErr) {
		return ErrorCategoryInternal
	}

	var invalidParam *types.InvalidParameterException
	if errors.As(err, &invalidParam) {
		return ErrorCategoryNonRetryable
	}

	var notFound *types.NotFoundException
	if errors.As(err, &notFound) {
		return ErrorCategoryNonRetryable
	}

	return ErrorCategoryUnknown
}

// categorizeAWSError categorizes an error by its AWS error code
func categorizeAWSError(code string) ErrorCategory {
	switch code {
	// Throttling errors
	case "Throttling", "ThrottlingException", "RequestThrottled",
		"ProvisionedThroughputExceededException", "RequestLimitExceeded":
		return ErrorCategoryThrottling

	// Retryable errors
	case "ServiceUnavailable", "ServiceUnavailableException",
		"InternalError", "InternalErrorException", "InternalFailure",
		"RequestTimeout", "RequestTimeoutException":
		return ErrorCategoryRetryable

	// Auth errors
	case "AccessDenied", "AccessDeniedException",
		"UnauthorizedAccess", "AuthorizationError",
		"InvalidClientTokenId", "MissingAuthenticationToken":
		return ErrorCategoryAuth

	// Network errors
	case "NetworkingError", "ConnectionError":
		return ErrorCategoryNetwork

	// Non-retryable errors
	case "InvalidParameter", "InvalidParameterException",
		"InvalidParameterValue", "MissingParameter",
		"ValidationError", "ValidationException",
		"MalformedInput", "InvalidInput":
		return ErrorCategoryNonRetryable

	default:
		return ErrorCategoryUnknown
	}
}

// IsRetryable determines if an error is retryable
func IsRetryable(err error) bool {
	category := CategorizeError(err)
	switch category {
	case ErrorCategoryRetryable, ErrorCategoryThrottling, ErrorCategoryNetwork, ErrorCategoryInternal:
		return true
	default:
		return false
	}
}

// WrapError wraps an error with SNS-specific context
func WrapError(err error, operation, topicARN string) *SNSError {
	if err == nil {
		return nil
	}

	snsErr := &SNSError{
		OriginalError: err,
		Category:      CategorizeError(err),
		Retryable:     IsRetryable(err),
		Message:       err.Error(),
		TopicARN:      topicARN,
		Operation:     operation,
		Timestamp:     time.Now(),
	}

	// Extract AWS-specific details
	var apiErr smithy.APIError
	if errors.As(err, &apiErr) {
		snsErr.Code = apiErr.ErrorCode()
		snsErr.Message = apiErr.ErrorMessage()
	}

	return snsErr
}

// ErrorHandler handles errors with retry logic
type ErrorHandler struct {
	maxRetries   int
	baseDelay    time.Duration
	maxDelay     time.Duration
	multiplier   float64
	jitterFactor float64

	// Metrics
	mu           sync.RWMutex
	totalRetries int64
	totalErrors  int64
	errorCounts  map[ErrorCategory]int64
}

// NewErrorHandler creates a new error handler
func NewErrorHandler(maxRetries int, baseDelay, maxDelay time.Duration, multiplier float64) *ErrorHandler {
	return &ErrorHandler{
		maxRetries:   maxRetries,
		baseDelay:    baseDelay,
		maxDelay:     maxDelay,
		multiplier:   multiplier,
		jitterFactor: 0.2,
		errorCounts:  make(map[ErrorCategory]int64),
	}
}

// Execute executes a function with retry logic
func (h *ErrorHandler) Execute(ctx context.Context, fn func() error) error {
	var lastErr error

	for attempt := 0; attempt <= h.maxRetries; attempt++ {
		err := fn()
		if err == nil {
			return nil
		}

		lastErr = err
		category := CategorizeError(err)

		h.mu.Lock()
		h.totalErrors++
		h.errorCounts[category]++
		h.mu.Unlock()

		// Don't retry non-retryable errors
		if !IsRetryable(err) {
			return WrapError(err, "", "")
		}

		// Don't retry if we've exhausted attempts
		if attempt >= h.maxRetries {
			break
		}

		h.mu.Lock()
		h.totalRetries++
		h.mu.Unlock()

		// Calculate delay with exponential backoff and jitter
		delay := h.calculateDelay(attempt, category)

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(delay):
			// Continue to next attempt
		}
	}

	return fmt.Errorf("%w: %v", ErrMaxRetriesExceeded, lastErr)
}

// calculateDelay calculates the delay for a retry attempt
func (h *ErrorHandler) calculateDelay(attempt int, category ErrorCategory) time.Duration {
	// Base exponential backoff
	delay := float64(h.baseDelay) * math.Pow(h.multiplier, float64(attempt))

	// Cap at max delay
	if delay > float64(h.maxDelay) {
		delay = float64(h.maxDelay)
	}

	// Add extra delay for throttling
	if category == ErrorCategoryThrottling {
		delay *= 2
	}

	// Add jitter
	jitter := delay * h.jitterFactor * (rand.Float64()*2 - 1)
	delay += jitter

	return time.Duration(delay)
}

// ExecuteWithCallback executes with retry and calls back on each attempt
func (h *ErrorHandler) ExecuteWithCallback(ctx context.Context, fn func() error, onRetry func(attempt int, err error, delay time.Duration)) error {
	var lastErr error

	for attempt := 0; attempt <= h.maxRetries; attempt++ {
		err := fn()
		if err == nil {
			return nil
		}

		lastErr = err
		category := CategorizeError(err)

		h.mu.Lock()
		h.totalErrors++
		h.errorCounts[category]++
		h.mu.Unlock()

		if !IsRetryable(err) {
			return WrapError(err, "", "")
		}

		if attempt >= h.maxRetries {
			break
		}

		h.mu.Lock()
		h.totalRetries++
		h.mu.Unlock()

		delay := h.calculateDelay(attempt, category)

		if onRetry != nil {
			onRetry(attempt+1, err, delay)
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(delay):
		}
	}

	return fmt.Errorf("%w: %v", ErrMaxRetriesExceeded, lastErr)
}

// Stats returns error handler statistics
func (h *ErrorHandler) Stats() ErrorHandlerStats {
	h.mu.RLock()
	defer h.mu.RUnlock()

	errorCounts := make(map[string]int64, len(h.errorCounts))
	for cat, count := range h.errorCounts {
		errorCounts[cat.String()] = count
	}

	return ErrorHandlerStats{
		TotalRetries: h.totalRetries,
		TotalErrors:  h.totalErrors,
		ErrorCounts:  errorCounts,
	}
}

// ErrorHandlerStats holds error handler statistics
type ErrorHandlerStats struct {
	TotalRetries int64            `json:"total_retries"`
	TotalErrors  int64            `json:"total_errors"`
	ErrorCounts  map[string]int64 `json:"error_counts"`
}

// CircuitBreaker implements the circuit breaker pattern
type CircuitBreaker struct {
	mu sync.Mutex

	state           CircuitState
	failureCount    int
	successCount    int
	lastFailure     time.Time
	lastStateChange time.Time

	// Configuration
	failureThreshold int
	successThreshold int
	timeout          time.Duration

	// Callbacks
	onStateChange func(from, to CircuitState)
}

// CircuitState represents the state of the circuit breaker
type CircuitState int

const (
	CircuitClosed CircuitState = iota
	CircuitOpen
	CircuitHalfOpen
)

// String returns a string representation of the circuit state
func (s CircuitState) String() string {
	switch s {
	case CircuitClosed:
		return "closed"
	case CircuitOpen:
		return "open"
	case CircuitHalfOpen:
		return "half_open"
	default:
		return "unknown"
	}
}

// NewCircuitBreaker creates a new circuit breaker
func NewCircuitBreaker(failureThreshold, successThreshold int, timeout time.Duration) *CircuitBreaker {
	return &CircuitBreaker{
		state:            CircuitClosed,
		failureThreshold: failureThreshold,
		successThreshold: successThreshold,
		timeout:          timeout,
		lastStateChange:  time.Now(),
	}
}

// OnStateChange sets a callback for state changes
func (cb *CircuitBreaker) OnStateChange(fn func(from, to CircuitState)) {
	cb.onStateChange = fn
}

// Allow checks if a request is allowed through the circuit
func (cb *CircuitBreaker) Allow() bool {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	switch cb.state {
	case CircuitClosed:
		return true
	case CircuitOpen:
		// Check if timeout has passed
		if time.Since(cb.lastStateChange) > cb.timeout {
			cb.transitionTo(CircuitHalfOpen)
			return true
		}
		return false
	case CircuitHalfOpen:
		return true
	default:
		return false
	}
}

// RecordSuccess records a successful operation
func (cb *CircuitBreaker) RecordSuccess() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	switch cb.state {
	case CircuitClosed:
		cb.failureCount = 0
	case CircuitHalfOpen:
		cb.successCount++
		if cb.successCount >= cb.successThreshold {
			cb.transitionTo(CircuitClosed)
		}
	}
}

// RecordFailure records a failed operation
func (cb *CircuitBreaker) RecordFailure() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.lastFailure = time.Now()

	switch cb.state {
	case CircuitClosed:
		cb.failureCount++
		if cb.failureCount >= cb.failureThreshold {
			cb.transitionTo(CircuitOpen)
		}
	case CircuitHalfOpen:
		cb.transitionTo(CircuitOpen)
	}
}

// transitionTo transitions to a new state
func (cb *CircuitBreaker) transitionTo(state CircuitState) {
	oldState := cb.state
	cb.state = state
	cb.lastStateChange = time.Now()

	// Reset counters
	cb.failureCount = 0
	cb.successCount = 0

	if cb.onStateChange != nil {
		go cb.onStateChange(oldState, state)
	}
}

// State returns the current circuit state
func (cb *CircuitBreaker) State() CircuitState {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	return cb.state
}

// Stats returns circuit breaker statistics
func (cb *CircuitBreaker) Stats() CircuitBreakerStats {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	return CircuitBreakerStats{
		State:           cb.state.String(),
		FailureCount:    cb.failureCount,
		SuccessCount:    cb.successCount,
		LastFailure:     cb.lastFailure,
		LastStateChange: cb.lastStateChange,
	}
}

// CircuitBreakerStats holds circuit breaker statistics
type CircuitBreakerStats struct {
	State           string    `json:"state"`
	FailureCount    int       `json:"failure_count"`
	SuccessCount    int       `json:"success_count"`
	LastFailure     time.Time `json:"last_failure"`
	LastStateChange time.Time `json:"last_state_change"`
}

// ErrorAggregator collects and aggregates errors for monitoring
type ErrorAggregator struct {
	mu sync.RWMutex

	errors         []ErrorRecord
	maxErrors      int
	errorsByType   map[ErrorCategory]int64
	errorsByCode   map[string]int64
	windowStart    time.Time
	windowDuration time.Duration
}

// ErrorRecord holds information about an error
type ErrorRecord struct {
	Error     error
	Category  ErrorCategory
	Code      string
	TopicARN  string
	Operation string
	Timestamp time.Time
}

// NewErrorAggregator creates a new error aggregator
func NewErrorAggregator(maxErrors int, windowDuration time.Duration) *ErrorAggregator {
	return &ErrorAggregator{
		errors:         make([]ErrorRecord, 0, maxErrors),
		maxErrors:      maxErrors,
		errorsByType:   make(map[ErrorCategory]int64),
		errorsByCode:   make(map[string]int64),
		windowStart:    time.Now(),
		windowDuration: windowDuration,
	}
}

// Record records an error
func (ea *ErrorAggregator) Record(err error, topicARN, operation string) {
	ea.mu.Lock()
	defer ea.mu.Unlock()

	category := CategorizeError(err)
	code := ""

	var apiErr smithy.APIError
	if errors.As(err, &apiErr) {
		code = apiErr.ErrorCode()
	}

	record := ErrorRecord{
		Error:     err,
		Category:  category,
		Code:      code,
		TopicARN:  topicARN,
		Operation: operation,
		Timestamp: time.Now(),
	}

	// Add to errors list
	ea.errors = append(ea.errors, record)
	if len(ea.errors) > ea.maxErrors {
		ea.errors = ea.errors[1:]
	}

	// Update counts
	ea.errorsByType[category]++
	if code != "" {
		ea.errorsByCode[code]++
	}

	// Reset window if needed
	if time.Since(ea.windowStart) > ea.windowDuration {
		ea.resetWindow()
	}
}

// resetWindow resets the aggregation window
func (ea *ErrorAggregator) resetWindow() {
	ea.errorsByType = make(map[ErrorCategory]int64)
	ea.errorsByCode = make(map[string]int64)
	ea.windowStart = time.Now()
}

// RecentErrors returns recent errors
func (ea *ErrorAggregator) RecentErrors(limit int) []ErrorRecord {
	ea.mu.RLock()
	defer ea.mu.RUnlock()

	if limit <= 0 || limit > len(ea.errors) {
		limit = len(ea.errors)
	}

	result := make([]ErrorRecord, limit)
	copy(result, ea.errors[len(ea.errors)-limit:])
	return result
}

// Stats returns error aggregation statistics
func (ea *ErrorAggregator) Stats() ErrorAggregatorStats {
	ea.mu.RLock()
	defer ea.mu.RUnlock()

	errorsByType := make(map[string]int64, len(ea.errorsByType))
	for cat, count := range ea.errorsByType {
		errorsByType[cat.String()] = count
	}

	return ErrorAggregatorStats{
		TotalErrors:  int64(len(ea.errors)),
		ErrorsByType: errorsByType,
		ErrorsByCode: ea.errorsByCode,
		WindowStart:  ea.windowStart,
	}
}

// ErrorAggregatorStats holds error aggregator statistics
type ErrorAggregatorStats struct {
	TotalErrors  int64            `json:"total_errors"`
	ErrorsByType map[string]int64 `json:"errors_by_type"`
	ErrorsByCode map[string]int64 `json:"errors_by_code"`
	WindowStart  time.Time        `json:"window_start"`
}
