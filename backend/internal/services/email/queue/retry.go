package queue

import (
	"context"
	"errors"
	"fmt"
	"math"
	"math/rand"
	"strings"
	"sync"
	"time"
)

// RetryPolicy defines the retry behavior
type RetryPolicy struct {
	MaxRetries      int           `json:"max_retries"`
	InitialDelay    time.Duration `json:"initial_delay"`
	MaxDelay        time.Duration `json:"max_delay"`
	Multiplier      float64       `json:"multiplier"`
	JitterPercent   float64       `json:"jitter_percent"`
	RetryableErrors []string      `json:"retryable_errors"`
}

// DefaultRetryPolicy returns a default retry policy
func DefaultRetryPolicy() *RetryPolicy {
	return &RetryPolicy{
		MaxRetries:    3,
		InitialDelay:  time.Second,
		MaxDelay:      time.Hour,
		Multiplier:    2.0,
		JitterPercent: 0.1,
		RetryableErrors: []string{
			"connection refused",
			"timeout",
			"temporary failure",
			"rate limit",
			"throttl",
			"service unavailable",
			"500",
			"502",
			"503",
			"504",
		},
	}
}

// RetryManager handles retry logic for failed jobs
type RetryManager struct {
	queue   *EmailQueue
	policy  *RetryPolicy
	metrics *RetryMetrics
	mu      sync.RWMutex

	// Circuit breaker
	circuitBreaker *CircuitBreaker

	// Retry callbacks
	onRetryScheduled func(*EmailJob, time.Duration)
	onMaxRetries     func(*EmailJob)
	onRetrySuccess   func(*EmailJob)
}

// RetryMetrics tracks retry statistics
type RetryMetrics struct {
	mu sync.RWMutex

	TotalRetries      int64
	SuccessfulRetries int64
	FailedRetries     int64
	RetryDelays       []time.Duration
	RetryAttempts     map[int]int64 // attempt number -> count
}

// NewRetryManager creates a new retry manager
func NewRetryManager(queue *EmailQueue, policy *RetryPolicy) *RetryManager {
	if policy == nil {
		policy = DefaultRetryPolicy()
	}

	return &RetryManager{
		queue:          queue,
		policy:         policy,
		metrics:        newRetryMetrics(),
		circuitBreaker: NewCircuitBreaker(5, 30*time.Second),
	}
}

// newRetryMetrics creates new retry metrics
func newRetryMetrics() *RetryMetrics {
	return &RetryMetrics{
		RetryDelays:   make([]time.Duration, 0, 100),
		RetryAttempts: make(map[int]int64),
	}
}

// ShouldRetry determines if a job should be retried
func (rm *RetryManager) ShouldRetry(job *EmailJob, err error) bool {
	// Check max retries
	if job.Attempts >= job.MaxRetries {
		return false
	}

	// Check circuit breaker
	if rm.circuitBreaker.IsOpen() {
		return false
	}

	// Check if error is retryable
	if !rm.IsRetryableError(err) {
		return false
	}

	return true
}

// IsRetryableError checks if an error is retryable
func (rm *RetryManager) IsRetryableError(err error) bool {
	if err == nil {
		return false
	}

	errStr := strings.ToLower(err.Error())

	for _, pattern := range rm.policy.RetryableErrors {
		if strings.Contains(errStr, strings.ToLower(pattern)) {
			return true
		}
	}

	// Check for specific error types
	var netErr interface{ Temporary() bool }
	if errors.As(err, &netErr) && netErr.Temporary() {
		return true
	}

	return false
}

// ScheduleRetry schedules a job for retry
func (rm *RetryManager) ScheduleRetry(ctx context.Context, job *EmailJob, err error) error {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	// Calculate delay
	delay := rm.CalculateDelay(job.Attempts)

	// Update job
	job.Status = JobStatusRetrying
	job.LastError = err.Error()
	nextRetry := time.Now().Add(delay)
	job.NextRetryAt = &nextRetry

	// Record metrics
	rm.recordRetryScheduled(job.Attempts, delay)

	// Notify circuit breaker
	rm.circuitBreaker.RecordFailure()

	// Callback
	if rm.onRetryScheduled != nil {
		rm.onRetryScheduled(job, delay)
	}

	return nil
}

// CalculateDelay calculates the retry delay using exponential backoff with jitter
func (rm *RetryManager) CalculateDelay(attempt int) time.Duration {
	return calculateBackoff(attempt, rm.policy.InitialDelay, rm.policy.MaxDelay)
}

// calculateBackoff calculates exponential backoff with jitter
func calculateBackoff(attempt int, initialDelay, maxDelay time.Duration) time.Duration {
	if attempt <= 0 {
		attempt = 1
	}

	// Exponential backoff: delay = initial * 2^(attempt-1)
	delay := float64(initialDelay) * math.Pow(2, float64(attempt-1))

	// Cap at max delay
	if delay > float64(maxDelay) {
		delay = float64(maxDelay)
	}

	// Add jitter (±10%)
	jitter := delay * 0.1 * (2*rand.Float64() - 1)
	delay += jitter

	return time.Duration(delay)
}

// RecordSuccess records a successful retry
func (rm *RetryManager) RecordSuccess(job *EmailJob) {
	rm.metrics.mu.Lock()
	defer rm.metrics.mu.Unlock()

	rm.metrics.SuccessfulRetries++

	// Notify circuit breaker
	rm.circuitBreaker.RecordSuccess()

	// Callback
	if rm.onRetrySuccess != nil {
		rm.onRetrySuccess(job)
	}
}

// RecordFailure records a failed retry
func (rm *RetryManager) RecordFailure(job *EmailJob) {
	rm.metrics.mu.Lock()
	defer rm.metrics.mu.Unlock()

	rm.metrics.FailedRetries++

	// Check if max retries reached
	if job.Attempts >= job.MaxRetries && rm.onMaxRetries != nil {
		rm.onMaxRetries(job)
	}
}

// recordRetryScheduled records a scheduled retry
func (rm *RetryManager) recordRetryScheduled(attempt int, delay time.Duration) {
	rm.metrics.mu.Lock()
	defer rm.metrics.mu.Unlock()

	rm.metrics.TotalRetries++
	rm.metrics.RetryAttempts[attempt]++

	if len(rm.metrics.RetryDelays) < 100 {
		rm.metrics.RetryDelays = append(rm.metrics.RetryDelays, delay)
	}
}

// SetCallbacks sets retry callbacks
func (rm *RetryManager) SetCallbacks(
	onRetryScheduled func(*EmailJob, time.Duration),
	onMaxRetries func(*EmailJob),
	onRetrySuccess func(*EmailJob),
) {
	rm.mu.Lock()
	defer rm.mu.Unlock()
	rm.onRetryScheduled = onRetryScheduled
	rm.onMaxRetries = onMaxRetries
	rm.onRetrySuccess = onRetrySuccess
}

// GetMetrics returns retry metrics
func (rm *RetryManager) GetMetrics() RetryStats {
	rm.metrics.mu.RLock()
	defer rm.metrics.mu.RUnlock()

	avgDelay := time.Duration(0)
	if len(rm.metrics.RetryDelays) > 0 {
		var total time.Duration
		for _, d := range rm.metrics.RetryDelays {
			total += d
		}
		avgDelay = total / time.Duration(len(rm.metrics.RetryDelays))
	}

	attempts := make(map[int]int64)
	for k, v := range rm.metrics.RetryAttempts {
		attempts[k] = v
	}

	return RetryStats{
		TotalRetries:       rm.metrics.TotalRetries,
		SuccessfulRetries:  rm.metrics.SuccessfulRetries,
		FailedRetries:      rm.metrics.FailedRetries,
		AvgRetryDelay:      avgDelay,
		RetryAttempts:      attempts,
		CircuitBreakerOpen: rm.circuitBreaker.IsOpen(),
	}
}

// RetryStats holds retry statistics
type RetryStats struct {
	TotalRetries       int64         `json:"total_retries"`
	SuccessfulRetries  int64         `json:"successful_retries"`
	FailedRetries      int64         `json:"failed_retries"`
	AvgRetryDelay      time.Duration `json:"avg_retry_delay"`
	RetryAttempts      map[int]int64 `json:"retry_attempts"`
	CircuitBreakerOpen bool          `json:"circuit_breaker_open"`
}

// ============================================
// Circuit Breaker
// ============================================

// CircuitBreakerState represents the state of a circuit breaker
type CircuitBreakerState string

const (
	CircuitBreakerClosed   CircuitBreakerState = "closed"
	CircuitBreakerOpen     CircuitBreakerState = "open"
	CircuitBreakerHalfOpen CircuitBreakerState = "half_open"
)

// CircuitBreaker implements the circuit breaker pattern
type CircuitBreaker struct {
	mu sync.RWMutex

	state           CircuitBreakerState
	failureCount    int
	successCount    int
	threshold       int
	resetTimeout    time.Duration
	lastFailure     time.Time
	lastStateChange time.Time

	// Half-open settings
	halfOpenRequests    int
	maxHalfOpenRequests int
}

// NewCircuitBreaker creates a new circuit breaker
func NewCircuitBreaker(threshold int, resetTimeout time.Duration) *CircuitBreaker {
	return &CircuitBreaker{
		state:               CircuitBreakerClosed,
		threshold:           threshold,
		resetTimeout:        resetTimeout,
		maxHalfOpenRequests: 3,
		lastStateChange:     time.Now(),
	}
}

// IsOpen returns true if the circuit is open
func (cb *CircuitBreaker) IsOpen() bool {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	switch cb.state {
	case CircuitBreakerOpen:
		// Check if we should move to half-open
		if time.Since(cb.lastFailure) > cb.resetTimeout {
			cb.transitionTo(CircuitBreakerHalfOpen)
			return false
		}
		return true
	case CircuitBreakerHalfOpen:
		return cb.halfOpenRequests >= cb.maxHalfOpenRequests
	default:
		return false
	}
}

// RecordSuccess records a successful operation
func (cb *CircuitBreaker) RecordSuccess() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	switch cb.state {
	case CircuitBreakerHalfOpen:
		cb.successCount++
		if cb.successCount >= cb.maxHalfOpenRequests {
			cb.transitionTo(CircuitBreakerClosed)
		}
	case CircuitBreakerClosed:
		cb.failureCount = 0
	}
}

// RecordFailure records a failed operation
func (cb *CircuitBreaker) RecordFailure() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.lastFailure = time.Now()
	cb.failureCount++

	switch cb.state {
	case CircuitBreakerClosed:
		if cb.failureCount >= cb.threshold {
			cb.transitionTo(CircuitBreakerOpen)
		}
	case CircuitBreakerHalfOpen:
		cb.transitionTo(CircuitBreakerOpen)
	}
}

// transitionTo transitions to a new state
func (cb *CircuitBreaker) transitionTo(state CircuitBreakerState) {
	cb.state = state
	cb.lastStateChange = time.Now()

	switch state {
	case CircuitBreakerClosed:
		cb.failureCount = 0
		cb.successCount = 0
	case CircuitBreakerHalfOpen:
		cb.halfOpenRequests = 0
		cb.successCount = 0
	case CircuitBreakerOpen:
		cb.halfOpenRequests = 0
	}
}

// State returns the current state
func (cb *CircuitBreaker) State() CircuitBreakerState {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	return cb.state
}

// GetStats returns circuit breaker statistics
func (cb *CircuitBreaker) GetStats() CircuitBreakerStats {
	cb.mu.RLock()
	defer cb.mu.RUnlock()

	return CircuitBreakerStats{
		State:           cb.state,
		FailureCount:    cb.failureCount,
		SuccessCount:    cb.successCount,
		LastFailure:     cb.lastFailure,
		LastStateChange: cb.lastStateChange,
	}
}

// CircuitBreakerStats holds circuit breaker statistics
type CircuitBreakerStats struct {
	State           CircuitBreakerState `json:"state"`
	FailureCount    int                 `json:"failure_count"`
	SuccessCount    int                 `json:"success_count"`
	LastFailure     time.Time           `json:"last_failure"`
	LastStateChange time.Time           `json:"last_state_change"`
}

// ============================================
// Retry Strategies
// ============================================

// RetryStrategy defines a retry strategy interface
type RetryStrategy interface {
	NextDelay(attempt int) time.Duration
	ShouldRetry(attempt int, err error) bool
}

// ExponentialBackoffStrategy implements exponential backoff
type ExponentialBackoffStrategy struct {
	InitialDelay time.Duration
	MaxDelay     time.Duration
	Multiplier   float64
	MaxRetries   int
}

// NextDelay returns the next delay for exponential backoff
func (s *ExponentialBackoffStrategy) NextDelay(attempt int) time.Duration {
	delay := float64(s.InitialDelay) * math.Pow(s.Multiplier, float64(attempt-1))
	if delay > float64(s.MaxDelay) {
		delay = float64(s.MaxDelay)
	}
	return time.Duration(delay)
}

// ShouldRetry checks if should retry
func (s *ExponentialBackoffStrategy) ShouldRetry(attempt int, err error) bool {
	return attempt < s.MaxRetries
}

// LinearBackoffStrategy implements linear backoff
type LinearBackoffStrategy struct {
	InitialDelay time.Duration
	Increment    time.Duration
	MaxDelay     time.Duration
	MaxRetries   int
}

// NextDelay returns the next delay for linear backoff
func (s *LinearBackoffStrategy) NextDelay(attempt int) time.Duration {
	delay := s.InitialDelay + time.Duration(attempt-1)*s.Increment
	if delay > s.MaxDelay {
		delay = s.MaxDelay
	}
	return delay
}

// ShouldRetry checks if should retry
func (s *LinearBackoffStrategy) ShouldRetry(attempt int, err error) bool {
	return attempt < s.MaxRetries
}

// ConstantBackoffStrategy implements constant delay
type ConstantBackoffStrategy struct {
	Delay      time.Duration
	MaxRetries int
}

// NextDelay returns a constant delay
func (s *ConstantBackoffStrategy) NextDelay(attempt int) time.Duration {
	return s.Delay
}

// ShouldRetry checks if should retry
func (s *ConstantBackoffStrategy) ShouldRetry(attempt int, err error) bool {
	return attempt < s.MaxRetries
}

// DecorrelatedJitterStrategy implements AWS-style decorrelated jitter
type DecorrelatedJitterStrategy struct {
	Base       time.Duration
	Cap        time.Duration
	MaxRetries int
	lastDelay  time.Duration
	mu         sync.Mutex
}

// NextDelay returns the next delay with decorrelated jitter
func (s *DecorrelatedJitterStrategy) NextDelay(attempt int) time.Duration {
	s.mu.Lock()
	defer s.mu.Unlock()

	if attempt == 1 || s.lastDelay == 0 {
		s.lastDelay = s.Base
	}

	// sleep = min(cap, random_between(base, sleep * 3))
	maxSleep := s.lastDelay * 3
	if maxSleep > s.Cap {
		maxSleep = s.Cap
	}

	delay := s.Base + time.Duration(rand.Float64()*float64(maxSleep-s.Base))
	s.lastDelay = delay

	return delay
}

// ShouldRetry checks if should retry
func (s *DecorrelatedJitterStrategy) ShouldRetry(attempt int, err error) bool {
	return attempt < s.MaxRetries
}

// ============================================
// Retry Executor
// ============================================

// RetryExecutor executes operations with retry logic
type RetryExecutor struct {
	strategy RetryStrategy
	onRetry  func(attempt int, err error, nextDelay time.Duration)
}

// NewRetryExecutor creates a new retry executor
func NewRetryExecutor(strategy RetryStrategy) *RetryExecutor {
	return &RetryExecutor{strategy: strategy}
}

// SetOnRetry sets the retry callback
func (re *RetryExecutor) SetOnRetry(fn func(attempt int, err error, nextDelay time.Duration)) {
	re.onRetry = fn
}

// Execute executes an operation with retries
func (re *RetryExecutor) Execute(ctx context.Context, operation func() error) error {
	var lastErr error

	for attempt := 1; ; attempt++ {
		err := operation()
		if err == nil {
			return nil
		}

		lastErr = err

		if !re.strategy.ShouldRetry(attempt, err) {
			break
		}

		delay := re.strategy.NextDelay(attempt)

		if re.onRetry != nil {
			re.onRetry(attempt, err, delay)
		}

		select {
		case <-ctx.Done():
			return fmt.Errorf("context cancelled: %w", ctx.Err())
		case <-time.After(delay):
		}
	}

	return lastErr
}

// ExecuteWithResult executes an operation with retries and returns a result
func ExecuteWithResult[T any](ctx context.Context, re *RetryExecutor, operation func() (T, error)) (T, error) {
	var result T
	var lastErr error

	for attempt := 1; ; attempt++ {
		r, err := operation()
		if err == nil {
			return r, nil
		}

		lastErr = err

		if !re.strategy.ShouldRetry(attempt, err) {
			break
		}

		delay := re.strategy.NextDelay(attempt)

		if re.onRetry != nil {
			re.onRetry(attempt, err, delay)
		}

		select {
		case <-ctx.Done():
			return result, fmt.Errorf("context cancelled: %w", ctx.Err())
		case <-time.After(delay):
		}
	}

	return result, lastErr
}
