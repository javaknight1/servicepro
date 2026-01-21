package sns

import (
	"context"
	"math"
	"sync"
	"time"
)

// RateLimiter implements a token bucket rate limiter
type RateLimiter struct {
	mu         sync.Mutex
	tokens     float64
	maxTokens  float64
	refillRate float64 // tokens per second
	lastRefill time.Time

	// Metrics
	allowed int64
	denied  int64
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter(ratePerSecond, burstSize int) *RateLimiter {
	return &RateLimiter{
		tokens:     float64(burstSize),
		maxTokens:  float64(burstSize),
		refillRate: float64(ratePerSecond),
		lastRefill: time.Now(),
	}
}

// Allow checks if a request is allowed and consumes a token if so
func (r *RateLimiter) Allow() bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.refill()

	if r.tokens >= 1 {
		r.tokens--
		r.allowed++
		return true
	}

	r.denied++
	return false
}

// AllowN checks if n requests are allowed
func (r *RateLimiter) AllowN(n int) bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.refill()

	if r.tokens >= float64(n) {
		r.tokens -= float64(n)
		r.allowed += int64(n)
		return true
	}

	r.denied++
	return false
}

// Wait blocks until a request is allowed or context is cancelled
func (r *RateLimiter) Wait(ctx context.Context) error {
	for {
		if r.Allow() {
			return nil
		}

		// Calculate wait time for next token
		waitTime := r.timeToNextToken()

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(waitTime):
			// Try again
		}
	}
}

// WaitN blocks until n requests are allowed or context is cancelled
func (r *RateLimiter) WaitN(ctx context.Context, n int) error {
	for {
		if r.AllowN(n) {
			return nil
		}

		// Calculate wait time for n tokens
		waitTime := r.timeToNTokens(n)

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(waitTime):
			// Try again
		}
	}
}

// Reserve reserves a token and returns the wait time
func (r *RateLimiter) Reserve() time.Duration {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.refill()

	if r.tokens >= 1 {
		r.tokens--
		r.allowed++
		return 0
	}

	// Calculate how long until we have a token
	tokensNeeded := 1 - r.tokens
	waitTime := time.Duration(tokensNeeded / r.refillRate * float64(time.Second))

	// Reserve the token (go into debt)
	r.tokens--
	r.allowed++

	return waitTime
}

// refill adds tokens based on elapsed time
func (r *RateLimiter) refill() {
	now := time.Now()
	elapsed := now.Sub(r.lastRefill).Seconds()
	r.tokens = math.Min(r.maxTokens, r.tokens+elapsed*r.refillRate)
	r.lastRefill = now
}

// timeToNextToken returns the time until the next token is available
func (r *RateLimiter) timeToNextToken() time.Duration {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.refill()

	if r.tokens >= 1 {
		return 0
	}

	tokensNeeded := 1 - r.tokens
	return time.Duration(tokensNeeded / r.refillRate * float64(time.Second))
}

// timeToNTokens returns the time until n tokens are available
func (r *RateLimiter) timeToNTokens(n int) time.Duration {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.refill()

	if r.tokens >= float64(n) {
		return 0
	}

	tokensNeeded := float64(n) - r.tokens
	return time.Duration(tokensNeeded / r.refillRate * float64(time.Second))
}

// Available returns the number of available tokens
func (r *RateLimiter) Available() int {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.refill()
	return int(r.tokens)
}

// Reset resets the rate limiter to full capacity
func (r *RateLimiter) Reset() {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.tokens = r.maxTokens
	r.lastRefill = time.Now()
}

// SetRate updates the rate limit
func (r *RateLimiter) SetRate(ratePerSecond int) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.refillRate = float64(ratePerSecond)
}

// SetBurst updates the burst capacity
func (r *RateLimiter) SetBurst(burstSize int) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.maxTokens = float64(burstSize)
	if r.tokens > r.maxTokens {
		r.tokens = r.maxTokens
	}
}

// Stats returns rate limiter statistics
func (r *RateLimiter) Stats() RateLimiterStats {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.refill()

	return RateLimiterStats{
		Available:    int(r.tokens),
		MaxTokens:    int(r.maxTokens),
		RefillRate:   int(r.refillRate),
		TotalAllowed: r.allowed,
		TotalDenied:  r.denied,
	}
}

// RateLimiterStats holds statistics for the rate limiter
type RateLimiterStats struct {
	Available    int   `json:"available"`
	MaxTokens    int   `json:"max_tokens"`
	RefillRate   int   `json:"refill_rate"`
	TotalAllowed int64 `json:"total_allowed"`
	TotalDenied  int64 `json:"total_denied"`
}

// AdaptiveRateLimiter adjusts rate based on error rates
type AdaptiveRateLimiter struct {
	*RateLimiter
	mu sync.Mutex

	baseRate    int
	minRate     int
	maxRate     int
	currentRate int

	// Error tracking
	successCount   int64
	errorCount     int64
	windowStart    time.Time
	windowDuration time.Duration

	// Thresholds
	errorThreshold float64 // Error rate threshold to decrease rate
	recoveryRate   float64 // Rate increase factor on recovery
	decreaseRate   float64 // Rate decrease factor on errors
}

// NewAdaptiveRateLimiter creates a new adaptive rate limiter
func NewAdaptiveRateLimiter(baseRate, minRate, maxRate, burstSize int) *AdaptiveRateLimiter {
	return &AdaptiveRateLimiter{
		RateLimiter:    NewRateLimiter(baseRate, burstSize),
		baseRate:       baseRate,
		minRate:        minRate,
		maxRate:        maxRate,
		currentRate:    baseRate,
		windowStart:    time.Now(),
		windowDuration: time.Minute,
		errorThreshold: 0.1, // 10% error rate triggers decrease
		recoveryRate:   1.1, // 10% increase on recovery
		decreaseRate:   0.5, // 50% decrease on errors
	}
}

// RecordSuccess records a successful operation
func (a *AdaptiveRateLimiter) RecordSuccess() {
	a.mu.Lock()
	defer a.mu.Unlock()

	a.successCount++
	a.maybeAdjust()
}

// RecordError records a failed operation
func (a *AdaptiveRateLimiter) RecordError() {
	a.mu.Lock()
	defer a.mu.Unlock()

	a.errorCount++
	a.maybeAdjust()
}

// maybeAdjust adjusts the rate based on error rate
func (a *AdaptiveRateLimiter) maybeAdjust() {
	// Check if we should evaluate the window
	if time.Since(a.windowStart) < a.windowDuration {
		return
	}

	total := a.successCount + a.errorCount
	if total == 0 {
		a.resetWindow()
		return
	}

	errorRate := float64(a.errorCount) / float64(total)

	if errorRate > a.errorThreshold {
		// Decrease rate
		newRate := int(float64(a.currentRate) * a.decreaseRate)
		if newRate < a.minRate {
			newRate = a.minRate
		}
		a.currentRate = newRate
		a.SetRate(newRate)
	} else if errorRate < a.errorThreshold/2 {
		// Increase rate if error rate is very low
		newRate := int(float64(a.currentRate) * a.recoveryRate)
		if newRate > a.maxRate {
			newRate = a.maxRate
		}
		a.currentRate = newRate
		a.SetRate(newRate)
	}

	a.resetWindow()
}

// resetWindow resets the error tracking window
func (a *AdaptiveRateLimiter) resetWindow() {
	a.successCount = 0
	a.errorCount = 0
	a.windowStart = time.Now()
}

// CurrentRate returns the current rate
func (a *AdaptiveRateLimiter) CurrentRate() int {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.currentRate
}

// SlidingWindowRateLimiter implements a sliding window rate limiter
type SlidingWindowRateLimiter struct {
	mu          sync.Mutex
	windowSize  time.Duration
	maxRequests int
	timestamps  []time.Time
}

// NewSlidingWindowRateLimiter creates a new sliding window rate limiter
func NewSlidingWindowRateLimiter(windowSize time.Duration, maxRequests int) *SlidingWindowRateLimiter {
	return &SlidingWindowRateLimiter{
		windowSize:  windowSize,
		maxRequests: maxRequests,
		timestamps:  make([]time.Time, 0, maxRequests),
	}
}

// Allow checks if a request is allowed
func (s *SlidingWindowRateLimiter) Allow() bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	windowStart := now.Add(-s.windowSize)

	// Remove timestamps outside the window
	validStart := 0
	for i, ts := range s.timestamps {
		if ts.After(windowStart) {
			validStart = i
			break
		}
		if i == len(s.timestamps)-1 {
			validStart = len(s.timestamps)
		}
	}
	s.timestamps = s.timestamps[validStart:]

	// Check if we're under the limit
	if len(s.timestamps) < s.maxRequests {
		s.timestamps = append(s.timestamps, now)
		return true
	}

	return false
}

// Count returns the number of requests in the current window
func (s *SlidingWindowRateLimiter) Count() int {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	windowStart := now.Add(-s.windowSize)

	count := 0
	for _, ts := range s.timestamps {
		if ts.After(windowStart) {
			count++
		}
	}

	return count
}

// Reset clears all timestamps
func (s *SlidingWindowRateLimiter) Reset() {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.timestamps = s.timestamps[:0]
}

// PerTopicRateLimiter manages rate limits per topic
type PerTopicRateLimiter struct {
	mu           sync.RWMutex
	limiters     map[string]*RateLimiter
	defaultRate  int
	defaultBurst int
	topicRates   map[string]int
}

// NewPerTopicRateLimiter creates a new per-topic rate limiter
func NewPerTopicRateLimiter(defaultRate, defaultBurst int) *PerTopicRateLimiter {
	return &PerTopicRateLimiter{
		limiters:     make(map[string]*RateLimiter),
		defaultRate:  defaultRate,
		defaultBurst: defaultBurst,
		topicRates:   make(map[string]int),
	}
}

// SetTopicRate sets a custom rate for a specific topic
func (p *PerTopicRateLimiter) SetTopicRate(topicARN string, rate int) {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.topicRates[topicARN] = rate
	if limiter, exists := p.limiters[topicARN]; exists {
		limiter.SetRate(rate)
	}
}

// Allow checks if a request to a topic is allowed
func (p *PerTopicRateLimiter) Allow(topicARN string) bool {
	limiter := p.getLimiter(topicARN)
	return limiter.Allow()
}

// Wait waits for permission to send to a topic
func (p *PerTopicRateLimiter) Wait(ctx context.Context, topicARN string) error {
	limiter := p.getLimiter(topicARN)
	return limiter.Wait(ctx)
}

// getLimiter gets or creates a limiter for a topic
func (p *PerTopicRateLimiter) getLimiter(topicARN string) *RateLimiter {
	p.mu.RLock()
	limiter, exists := p.limiters[topicARN]
	p.mu.RUnlock()

	if exists {
		return limiter
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	// Double-check after acquiring write lock
	if limiter, exists = p.limiters[topicARN]; exists {
		return limiter
	}

	// Determine rate for this topic
	rate := p.defaultRate
	if customRate, hasCustom := p.topicRates[topicARN]; hasCustom {
		rate = customRate
	}

	limiter = NewRateLimiter(rate, p.defaultBurst)
	p.limiters[topicARN] = limiter
	return limiter
}

// Stats returns statistics for all topics
func (p *PerTopicRateLimiter) Stats() map[string]RateLimiterStats {
	p.mu.RLock()
	defer p.mu.RUnlock()

	stats := make(map[string]RateLimiterStats, len(p.limiters))
	for topic, limiter := range p.limiters {
		stats[topic] = limiter.Stats()
	}
	return stats
}
