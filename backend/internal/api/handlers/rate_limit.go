package handlers

import (
	"sync"
	"time"
)

// =============================================================================
// Download Rate Limiter
// =============================================================================

// DownloadRateLimiter implements rate limiting for downloads
type DownloadRateLimiter struct {
	mu sync.RWMutex

	// Rate limits by user ID
	userLimits map[string]*rateLimitBucket

	// Rate limits by IP address
	ipLimits map[string]*rateLimitBucket

	// Configuration
	config *RateLimitConfig

	// Cleanup ticker
	cleanupTicker *time.Ticker
	stopCleanup   chan struct{}
}

// rateLimitBucket tracks rate limit state using token bucket algorithm
type rateLimitBucket struct {
	tokens     float64
	maxTokens  float64
	refillRate float64 // tokens per second
	lastRefill time.Time
}

// RateLimitConfig contains rate limiter configuration
type RateLimitConfig struct {
	// Per-user limits
	UserRequestsPerMinute int `json:"user_requests_per_minute"`
	UserBurstSize         int `json:"user_burst_size"`

	// Per-IP limits (for anonymous users)
	IPRequestsPerMinute int `json:"ip_requests_per_minute"`
	IPBurstSize         int `json:"ip_burst_size"`

	// Global limits
	GlobalRequestsPerMinute int `json:"global_requests_per_minute"`

	// Bandwidth limits (bytes per second)
	UserBandwidthLimit   int64 `json:"user_bandwidth_limit"`
	GlobalBandwidthLimit int64 `json:"global_bandwidth_limit"`

	// Cleanup interval for stale buckets
	CleanupInterval time.Duration `json:"cleanup_interval"`

	// Bucket expiration time
	BucketExpiration time.Duration `json:"bucket_expiration"`

	// Whitelist of user IDs exempt from rate limiting
	WhitelistedUsers []string `json:"whitelisted_users"`

	// Whitelist of IP addresses exempt from rate limiting
	WhitelistedIPs []string `json:"whitelisted_ips"`
}

// DefaultRateLimitConfig returns default rate limit configuration
func DefaultRateLimitConfig() *RateLimitConfig {
	return &RateLimitConfig{
		UserRequestsPerMinute:   60, // 1 per second average
		UserBurstSize:           10, // Allow bursts of 10
		IPRequestsPerMinute:     30, // Stricter for anonymous
		IPBurstSize:             5,
		GlobalRequestsPerMinute: 1000,
		UserBandwidthLimit:      50 * 1024 * 1024,  // 50 MB/s per user
		GlobalBandwidthLimit:    500 * 1024 * 1024, // 500 MB/s total
		CleanupInterval:         5 * time.Minute,
		BucketExpiration:        30 * time.Minute,
		WhitelistedUsers:        []string{},
		WhitelistedIPs:          []string{},
	}
}

// NewDownloadRateLimiter creates a new rate limiter
func NewDownloadRateLimiter(config *RateLimitConfig) *DownloadRateLimiter {
	if config == nil {
		config = DefaultRateLimitConfig()
	}

	rl := &DownloadRateLimiter{
		userLimits:  make(map[string]*rateLimitBucket),
		ipLimits:    make(map[string]*rateLimitBucket),
		config:      config,
		stopCleanup: make(chan struct{}),
	}

	// Start cleanup goroutine
	if config.CleanupInterval > 0 {
		rl.cleanupTicker = time.NewTicker(config.CleanupInterval)
		go rl.cleanupLoop()
	}

	return rl
}

// Allow checks if a request is allowed for the given user and IP
func (rl *DownloadRateLimiter) Allow(userID, ip string) bool {
	// Check whitelists
	if rl.isWhitelistedUser(userID) || rl.isWhitelistedIP(ip) {
		return true
	}

	rl.mu.Lock()
	defer rl.mu.Unlock()

	// Check user limit if authenticated
	if userID != "" && userID != "anonymous" {
		bucket := rl.getOrCreateUserBucket(userID)
		if !bucket.consume(1) {
			return false
		}
	} else {
		// Check IP limit for anonymous users
		bucket := rl.getOrCreateIPBucket(ip)
		if !bucket.consume(1) {
			return false
		}
	}

	return true
}

// AllowN checks if N requests are allowed
func (rl *DownloadRateLimiter) AllowN(userID, ip string, n int) bool {
	if n <= 0 {
		return true
	}

	// Check whitelists
	if rl.isWhitelistedUser(userID) || rl.isWhitelistedIP(ip) {
		return true
	}

	rl.mu.Lock()
	defer rl.mu.Unlock()

	if userID != "" && userID != "anonymous" {
		bucket := rl.getOrCreateUserBucket(userID)
		return bucket.consume(float64(n))
	}

	bucket := rl.getOrCreateIPBucket(ip)
	return bucket.consume(float64(n))
}

// getOrCreateUserBucket gets or creates a rate limit bucket for a user
func (rl *DownloadRateLimiter) getOrCreateUserBucket(userID string) *rateLimitBucket {
	if bucket, ok := rl.userLimits[userID]; ok {
		return bucket
	}

	bucket := &rateLimitBucket{
		tokens:     float64(rl.config.UserBurstSize),
		maxTokens:  float64(rl.config.UserBurstSize),
		refillRate: float64(rl.config.UserRequestsPerMinute) / 60.0,
		lastRefill: time.Now(),
	}
	rl.userLimits[userID] = bucket
	return bucket
}

// getOrCreateIPBucket gets or creates a rate limit bucket for an IP
func (rl *DownloadRateLimiter) getOrCreateIPBucket(ip string) *rateLimitBucket {
	if bucket, ok := rl.ipLimits[ip]; ok {
		return bucket
	}

	bucket := &rateLimitBucket{
		tokens:     float64(rl.config.IPBurstSize),
		maxTokens:  float64(rl.config.IPBurstSize),
		refillRate: float64(rl.config.IPRequestsPerMinute) / 60.0,
		lastRefill: time.Now(),
	}
	rl.ipLimits[ip] = bucket
	return bucket
}

// consume attempts to consume tokens from the bucket
func (b *rateLimitBucket) consume(tokens float64) bool {
	now := time.Now()

	// Refill tokens based on time elapsed
	elapsed := now.Sub(b.lastRefill).Seconds()
	b.tokens += elapsed * b.refillRate
	if b.tokens > b.maxTokens {
		b.tokens = b.maxTokens
	}
	b.lastRefill = now

	// Check if enough tokens available
	if b.tokens < tokens {
		return false
	}

	b.tokens -= tokens
	return true
}

// available returns the current number of available tokens
func (b *rateLimitBucket) available() float64 {
	now := time.Now()
	elapsed := now.Sub(b.lastRefill).Seconds()
	tokens := b.tokens + elapsed*b.refillRate
	if tokens > b.maxTokens {
		tokens = b.maxTokens
	}
	return tokens
}

// isWhitelistedUser checks if a user is whitelisted
func (rl *DownloadRateLimiter) isWhitelistedUser(userID string) bool {
	for _, id := range rl.config.WhitelistedUsers {
		if id == userID {
			return true
		}
	}
	return false
}

// isWhitelistedIP checks if an IP is whitelisted
func (rl *DownloadRateLimiter) isWhitelistedIP(ip string) bool {
	for _, addr := range rl.config.WhitelistedIPs {
		if addr == ip {
			return true
		}
	}
	return false
}

// GetUserStatus returns the rate limit status for a user
func (rl *DownloadRateLimiter) GetUserStatus(userID string) *RateLimitStatus {
	rl.mu.RLock()
	defer rl.mu.RUnlock()

	bucket, ok := rl.userLimits[userID]
	if !ok {
		return &RateLimitStatus{
			Allowed:         true,
			RemainingTokens: float64(rl.config.UserBurstSize),
			MaxTokens:       float64(rl.config.UserBurstSize),
			RefillRate:      float64(rl.config.UserRequestsPerMinute) / 60.0,
		}
	}

	available := bucket.available()
	return &RateLimitStatus{
		Allowed:         available >= 1,
		RemainingTokens: available,
		MaxTokens:       bucket.maxTokens,
		RefillRate:      bucket.refillRate,
		ResetTime:       time.Now().Add(time.Duration((bucket.maxTokens-available)/bucket.refillRate) * time.Second),
	}
}

// GetIPStatus returns the rate limit status for an IP
func (rl *DownloadRateLimiter) GetIPStatus(ip string) *RateLimitStatus {
	rl.mu.RLock()
	defer rl.mu.RUnlock()

	bucket, ok := rl.ipLimits[ip]
	if !ok {
		return &RateLimitStatus{
			Allowed:         true,
			RemainingTokens: float64(rl.config.IPBurstSize),
			MaxTokens:       float64(rl.config.IPBurstSize),
			RefillRate:      float64(rl.config.IPRequestsPerMinute) / 60.0,
		}
	}

	available := bucket.available()
	return &RateLimitStatus{
		Allowed:         available >= 1,
		RemainingTokens: available,
		MaxTokens:       bucket.maxTokens,
		RefillRate:      bucket.refillRate,
		ResetTime:       time.Now().Add(time.Duration((bucket.maxTokens-available)/bucket.refillRate) * time.Second),
	}
}

// RateLimitStatus contains rate limit status information
type RateLimitStatus struct {
	Allowed         bool      `json:"allowed"`
	RemainingTokens float64   `json:"remaining_tokens"`
	MaxTokens       float64   `json:"max_tokens"`
	RefillRate      float64   `json:"refill_rate"` // tokens per second
	ResetTime       time.Time `json:"reset_time,omitempty"`
}

// Reset resets the rate limiter for a specific user
func (rl *DownloadRateLimiter) Reset(userID string) {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	delete(rl.userLimits, userID)
}

// ResetIP resets the rate limiter for a specific IP
func (rl *DownloadRateLimiter) ResetIP(ip string) {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	delete(rl.ipLimits, ip)
}

// ResetAll resets all rate limiters
func (rl *DownloadRateLimiter) ResetAll() {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	rl.userLimits = make(map[string]*rateLimitBucket)
	rl.ipLimits = make(map[string]*rateLimitBucket)
}

// AddWhitelistedUser adds a user to the whitelist
func (rl *DownloadRateLimiter) AddWhitelistedUser(userID string) {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	// Check if already whitelisted
	for _, id := range rl.config.WhitelistedUsers {
		if id == userID {
			return
		}
	}
	rl.config.WhitelistedUsers = append(rl.config.WhitelistedUsers, userID)
}

// RemoveWhitelistedUser removes a user from the whitelist
func (rl *DownloadRateLimiter) RemoveWhitelistedUser(userID string) {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	filtered := make([]string, 0)
	for _, id := range rl.config.WhitelistedUsers {
		if id != userID {
			filtered = append(filtered, id)
		}
	}
	rl.config.WhitelistedUsers = filtered
}

// AddWhitelistedIP adds an IP to the whitelist
func (rl *DownloadRateLimiter) AddWhitelistedIP(ip string) {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	// Check if already whitelisted
	for _, addr := range rl.config.WhitelistedIPs {
		if addr == ip {
			return
		}
	}
	rl.config.WhitelistedIPs = append(rl.config.WhitelistedIPs, ip)
}

// RemoveWhitelistedIP removes an IP from the whitelist
func (rl *DownloadRateLimiter) RemoveWhitelistedIP(ip string) {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	filtered := make([]string, 0)
	for _, addr := range rl.config.WhitelistedIPs {
		if addr != ip {
			filtered = append(filtered, addr)
		}
	}
	rl.config.WhitelistedIPs = filtered
}

// cleanupLoop periodically cleans up stale buckets
func (rl *DownloadRateLimiter) cleanupLoop() {
	for {
		select {
		case <-rl.cleanupTicker.C:
			rl.cleanup()
		case <-rl.stopCleanup:
			return
		}
	}
}

// cleanup removes stale rate limit buckets
func (rl *DownloadRateLimiter) cleanup() {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	expiration := time.Now().Add(-rl.config.BucketExpiration)

	// Clean up user buckets
	for userID, bucket := range rl.userLimits {
		if bucket.lastRefill.Before(expiration) {
			delete(rl.userLimits, userID)
		}
	}

	// Clean up IP buckets
	for ip, bucket := range rl.ipLimits {
		if bucket.lastRefill.Before(expiration) {
			delete(rl.ipLimits, ip)
		}
	}
}

// Stop stops the rate limiter and cleanup goroutine
func (rl *DownloadRateLimiter) Stop() {
	if rl.cleanupTicker != nil {
		rl.cleanupTicker.Stop()
		close(rl.stopCleanup)
	}
}

// Stats returns rate limiter statistics
func (rl *DownloadRateLimiter) Stats() *RateLimiterStats {
	rl.mu.RLock()
	defer rl.mu.RUnlock()

	return &RateLimiterStats{
		ActiveUserBuckets: len(rl.userLimits),
		ActiveIPBuckets:   len(rl.ipLimits),
		WhitelistedUsers:  len(rl.config.WhitelistedUsers),
		WhitelistedIPs:    len(rl.config.WhitelistedIPs),
	}
}

// RateLimiterStats contains rate limiter statistics
type RateLimiterStats struct {
	ActiveUserBuckets int `json:"active_user_buckets"`
	ActiveIPBuckets   int `json:"active_ip_buckets"`
	WhitelistedUsers  int `json:"whitelisted_users"`
	WhitelistedIPs    int `json:"whitelisted_ips"`
}

// UpdateConfig updates the rate limiter configuration
func (rl *DownloadRateLimiter) UpdateConfig(config *RateLimitConfig) {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	if config.UserRequestsPerMinute > 0 {
		rl.config.UserRequestsPerMinute = config.UserRequestsPerMinute
	}
	if config.UserBurstSize > 0 {
		rl.config.UserBurstSize = config.UserBurstSize
	}
	if config.IPRequestsPerMinute > 0 {
		rl.config.IPRequestsPerMinute = config.IPRequestsPerMinute
	}
	if config.IPBurstSize > 0 {
		rl.config.IPBurstSize = config.IPBurstSize
	}
	if config.GlobalRequestsPerMinute > 0 {
		rl.config.GlobalRequestsPerMinute = config.GlobalRequestsPerMinute
	}
	if config.UserBandwidthLimit > 0 {
		rl.config.UserBandwidthLimit = config.UserBandwidthLimit
	}
	if config.GlobalBandwidthLimit > 0 {
		rl.config.GlobalBandwidthLimit = config.GlobalBandwidthLimit
	}

	// Update whitelists if provided
	if len(config.WhitelistedUsers) > 0 {
		rl.config.WhitelistedUsers = config.WhitelistedUsers
	}
	if len(config.WhitelistedIPs) > 0 {
		rl.config.WhitelistedIPs = config.WhitelistedIPs
	}
}
