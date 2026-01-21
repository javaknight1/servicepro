package tracking

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"sync"
	"time"
)

// Standard errors for link tracking
var (
	ErrInvalidLinkToken     = errors.New("invalid link tracking token")
	ErrLinkNotFound         = errors.New("tracked link not found")
	ErrLinkTrackingDisabled = errors.New("link tracking is disabled for this email")
	ErrMalformedURL         = errors.New("malformed URL")
)

// LinkTrackerConfig holds configuration for link tracking
type LinkTrackerConfig struct {
	BaseURL             string        `json:"base_url"`              // Base URL for tracking endpoints
	SecretKey           string        `json:"-"`                     // Secret key for token signing
	TokenExpiration     time.Duration `json:"token_expiration"`      // How long tokens are valid
	PreserveQueryParams bool          `json:"preserve_query_params"` // Preserve original query params
	AllowedDomains      []string      `json:"allowed_domains"`       // Whitelist of allowed redirect domains
	BlockedDomains      []string      `json:"blocked_domains"`       // Blacklist of blocked domains
	CacheClickEvents    bool          `json:"cache_click_events"`    // Cache recent clicks for dedup
	CacheTTL            time.Duration `json:"cache_ttl"`             // Cache TTL
	RespectDNT          bool          `json:"respect_dnt"`           // Respect Do Not Track header
}

// DefaultLinkTrackerConfig returns sensible defaults
func DefaultLinkTrackerConfig() *LinkTrackerConfig {
	return &LinkTrackerConfig{
		BaseURL:             "https://track.example.com",
		TokenExpiration:     90 * 24 * time.Hour,
		PreserveQueryParams: true,
		CacheClickEvents:    true,
		CacheTTL:            24 * time.Hour,
		RespectDNT:          true,
	}
}

// LinkTracker handles email link click tracking
type LinkTracker struct {
	config      *LinkTrackerConfig
	storage     *TrackingStorage
	optoutStore OptoutStore
	geoLocator  GeoLocator
	uaParser    UserAgentParser

	// Cache for deduplication
	clickCache   map[string]time.Time
	clickCacheMu sync.RWMutex

	// Metrics
	metrics *LinkMetrics
}

// LinkMetrics tracks link tracking statistics
type LinkMetrics struct {
	mu               sync.RWMutex
	TotalClicks      int64
	UniqueClicks     int64
	DuplicateClicks  int64
	InvalidTokens    int64
	BlockedRedirects int64
	OptedOutRequests int64
	DNTRequests      int64
	ErrorCount       int64
}

// NewLinkTracker creates a new link tracker
func NewLinkTracker(config *LinkTrackerConfig, storage *TrackingStorage, optoutStore OptoutStore) *LinkTracker {
	if config == nil {
		config = DefaultLinkTrackerConfig()
	}

	lt := &LinkTracker{
		config:      config,
		storage:     storage,
		optoutStore: optoutStore,
		clickCache:  make(map[string]time.Time),
		metrics:     &LinkMetrics{},
	}

	// Start cache cleanup goroutine
	if config.CacheClickEvents {
		go lt.cleanupCache()
	}

	return lt
}

// SetGeoLocator sets the geo locator for IP lookup
func (lt *LinkTracker) SetGeoLocator(gl GeoLocator) {
	lt.geoLocator = gl
}

// SetUserAgentParser sets the user agent parser
func (lt *LinkTracker) SetUserAgentParser(uap UserAgentParser) {
	lt.uaParser = uap
}

// GenerateTrackingURL generates a tracking URL for a link
func (lt *LinkTracker) GenerateTrackingURL(emailID, recipientID, linkID, originalURL string) (string, error) {
	// Validate original URL
	parsedURL, err := url.Parse(originalURL)
	if err != nil {
		return "", ErrMalformedURL
	}

	// Check domain restrictions
	if !lt.isAllowedDomain(parsedURL.Host) {
		return originalURL, nil // Return original if domain not allowed for tracking
	}

	token, err := lt.generateToken(emailID, recipientID, linkID, originalURL)
	if err != nil {
		return "", err
	}

	trackingURL := fmt.Sprintf("%s/click/%s", lt.config.BaseURL, token)
	return trackingURL, nil
}

// RewriteLinks rewrites all links in HTML content to use tracking URLs
func (lt *LinkTracker) RewriteLinks(ctx context.Context, htmlContent, emailID, recipientID string) (string, error) {
	// Regular expression to find href attributes
	linkRegex := regexp.MustCompile(`(?i)href\s*=\s*["']([^"']+)["']`)

	position := 0
	result := linkRegex.ReplaceAllStringFunc(htmlContent, func(match string) string {
		// Extract the URL
		submatches := linkRegex.FindStringSubmatch(match)
		if len(submatches) < 2 {
			return match
		}

		originalURL := submatches[1]

		// Skip non-http links (mailto:, tel:, javascript:, etc.)
		if !strings.HasPrefix(strings.ToLower(originalURL), "http") {
			return match
		}

		// Skip unsubscribe links
		if strings.Contains(strings.ToLower(originalURL), "unsubscribe") {
			return match
		}

		position++

		// Create tracked link record
		trackedLink := &TrackedLink{
			EmailID:     emailID,
			OriginalURL: originalURL,
			Position:    position,
		}

		err := lt.storage.CreateTrackedLink(ctx, trackedLink)
		if err != nil {
			return match // On error, keep original
		}

		// Generate tracking URL
		trackingURL, err := lt.GenerateTrackingURL(emailID, recipientID, trackedLink.ID, originalURL)
		if err != nil {
			return match
		}

		// Update the tracked link with the tracking URL
		trackedLink.TrackingURL = trackingURL

		// Return the modified href
		return fmt.Sprintf(`href="%s"`, trackingURL)
	})

	return result, nil
}

// RewriteLinksWithText rewrites links and captures link text
func (lt *LinkTracker) RewriteLinksWithText(ctx context.Context, htmlContent, emailID, recipientID string) (string, error) {
	// More complex regex to capture link text
	linkWithTextRegex := regexp.MustCompile(`(?is)<a\s+([^>]*href\s*=\s*["']([^"']+)["'][^>]*)>([^<]*)</a>`)

	position := 0
	result := linkWithTextRegex.ReplaceAllStringFunc(htmlContent, func(match string) string {
		submatches := linkWithTextRegex.FindStringSubmatch(match)
		if len(submatches) < 4 {
			return match
		}

		attributes := submatches[1]
		originalURL := submatches[2]
		linkText := strings.TrimSpace(submatches[3])

		// Skip non-http links
		if !strings.HasPrefix(strings.ToLower(originalURL), "http") {
			return match
		}

		// Skip unsubscribe links
		if strings.Contains(strings.ToLower(originalURL), "unsubscribe") {
			return match
		}

		position++

		// Create tracked link record
		trackedLink := &TrackedLink{
			EmailID:     emailID,
			OriginalURL: originalURL,
			LinkText:    linkText,
			Position:    position,
		}

		err := lt.storage.CreateTrackedLink(ctx, trackedLink)
		if err != nil {
			return match
		}

		// Generate tracking URL
		trackingURL, err := lt.GenerateTrackingURL(emailID, recipientID, trackedLink.ID, originalURL)
		if err != nil {
			return match
		}

		// Replace URL in attributes
		newAttributes := strings.Replace(attributes, originalURL, trackingURL, 1)
		return fmt.Sprintf(`<a %s>%s</a>`, newAttributes, linkText)
	})

	return result, nil
}

// isAllowedDomain checks if a domain is allowed for tracking
func (lt *LinkTracker) isAllowedDomain(domain string) bool {
	domain = strings.ToLower(domain)

	// Check blocked domains
	for _, blocked := range lt.config.BlockedDomains {
		if strings.Contains(domain, strings.ToLower(blocked)) {
			return false
		}
	}

	// If whitelist is empty, allow all non-blocked
	if len(lt.config.AllowedDomains) == 0 {
		return true
	}

	// Check allowed domains
	for _, allowed := range lt.config.AllowedDomains {
		if strings.Contains(domain, strings.ToLower(allowed)) {
			return true
		}
	}

	return false
}

// generateToken creates a signed link tracking token
func (lt *LinkTracker) generateToken(emailID, recipientID, linkID, originalURL string) (string, error) {
	// Token format: emailID|recipientID|linkID|urlHash|timestamp|signature
	timestamp := time.Now().Unix()
	urlHash := lt.hashURL(originalURL)
	data := fmt.Sprintf("%s|%s|%s|%s|%d", emailID, recipientID, linkID, urlHash, timestamp)

	signature := lt.sign(data)
	token := fmt.Sprintf("%s|%s", data, signature)

	// Base64 URL encode
	return base64.RawURLEncoding.EncodeToString([]byte(token)), nil
}

// ParseLinkToken validates and parses a link tracking token
func (lt *LinkTracker) ParseLinkToken(token string) (*LinkTokenData, error) {
	// Decode base64
	decoded, err := base64.RawURLEncoding.DecodeString(token)
	if err != nil {
		return nil, ErrInvalidLinkToken
	}

	parts := strings.Split(string(decoded), "|")
	if len(parts) != 6 {
		return nil, ErrInvalidLinkToken
	}

	emailID := parts[0]
	recipientID := parts[1]
	linkID := parts[2]
	urlHash := parts[3]
	timestamp := parts[4]
	signature := parts[5]

	// Verify signature
	data := fmt.Sprintf("%s|%s|%s|%s|%s", emailID, recipientID, linkID, urlHash, timestamp)
	if !lt.verify(data, signature) {
		return nil, ErrInvalidLinkToken
	}

	// Parse timestamp
	var ts int64
	fmt.Sscanf(timestamp, "%d", &ts)
	tokenTime := time.Unix(ts, 0)

	// Check expiration
	if lt.config.TokenExpiration > 0 && time.Since(tokenTime) > lt.config.TokenExpiration {
		return nil, ErrExpiredToken
	}

	return &LinkTokenData{
		EmailID:     emailID,
		RecipientID: recipientID,
		LinkID:      linkID,
		URLHash:     urlHash,
		Timestamp:   tokenTime,
	}, nil
}

// LinkTokenData contains parsed link token information
type LinkTokenData struct {
	EmailID     string
	RecipientID string
	LinkID      string
	URLHash     string
	Timestamp   time.Time
}

// hashURL creates a short hash of a URL
func (lt *LinkTracker) hashURL(url string) string {
	h := sha256.Sum256([]byte(url))
	return hex.EncodeToString(h[:8]) // First 8 bytes (16 hex chars)
}

// sign creates an HMAC signature
func (lt *LinkTracker) sign(data string) string {
	h := hmac.New(sha256.New, []byte(lt.config.SecretKey))
	h.Write([]byte(data))
	return hex.EncodeToString(h.Sum(nil))
}

// verify checks an HMAC signature
func (lt *LinkTracker) verify(data, signature string) bool {
	expected := lt.sign(data)
	return hmac.Equal([]byte(expected), []byte(signature))
}

// HandleClickRequest handles an HTTP request for link click tracking
func (lt *LinkTracker) HandleClickRequest(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	lt.metrics.mu.Lock()
	lt.metrics.TotalClicks++
	lt.metrics.mu.Unlock()

	// Extract token from URL (e.g., /click/{token})
	path := r.URL.Path
	token := strings.TrimPrefix(path, "/click/")

	// Parse token
	tokenData, err := lt.ParseLinkToken(token)
	if err != nil {
		lt.metrics.mu.Lock()
		lt.metrics.InvalidTokens++
		lt.metrics.mu.Unlock()
		http.Error(w, "Invalid link", http.StatusBadRequest)
		return
	}

	// Get the tracked link
	link, err := lt.storage.GetTrackedLink(ctx, tokenData.LinkID)
	if err != nil {
		lt.metrics.mu.Lock()
		lt.metrics.ErrorCount++
		lt.metrics.mu.Unlock()
		http.Error(w, "Link not found", http.StatusNotFound)
		return
	}

	// Verify URL hash matches
	if lt.hashURL(link.OriginalURL) != tokenData.URLHash {
		lt.metrics.mu.Lock()
		lt.metrics.InvalidTokens++
		lt.metrics.mu.Unlock()
		http.Error(w, "Invalid link", http.StatusBadRequest)
		return
	}

	// Check domain restrictions for redirect
	parsedURL, err := url.Parse(link.OriginalURL)
	if err != nil || !lt.isSafeRedirect(parsedURL) {
		lt.metrics.mu.Lock()
		lt.metrics.BlockedRedirects++
		lt.metrics.mu.Unlock()
		http.Error(w, "Redirect blocked", http.StatusForbidden)
		return
	}

	// Check if opted out
	shouldTrack := true
	if lt.optoutStore != nil {
		optedOut, _ := lt.optoutStore.IsOptedOut(ctx, tokenData.RecipientID)
		if optedOut {
			lt.metrics.mu.Lock()
			lt.metrics.OptedOutRequests++
			lt.metrics.mu.Unlock()
			shouldTrack = false
		}
	}

	// Respect Do Not Track header
	if lt.config.RespectDNT && r.Header.Get("DNT") == "1" {
		lt.metrics.mu.Lock()
		lt.metrics.DNTRequests++
		lt.metrics.mu.Unlock()
		shouldTrack = false
	}

	// Record the click event (async)
	if shouldTrack {
		isUnique := lt.checkAndCacheClick(tokenData.EmailID, tokenData.RecipientID, tokenData.LinkID)
		go lt.recordClick(ctx, tokenData, link, r, isUnique)
	}

	// Preserve query parameters if configured
	redirectURL := link.OriginalURL
	if lt.config.PreserveQueryParams && r.URL.RawQuery != "" {
		if strings.Contains(redirectURL, "?") {
			redirectURL += "&" + r.URL.RawQuery
		} else {
			redirectURL += "?" + r.URL.RawQuery
		}
	}

	// Redirect to original URL
	http.Redirect(w, r, redirectURL, http.StatusTemporaryRedirect)
}

// isSafeRedirect checks if the redirect URL is safe
func (lt *LinkTracker) isSafeRedirect(u *url.URL) bool {
	// Must be http or https
	if u.Scheme != "http" && u.Scheme != "https" {
		return false
	}

	// Check blocked domains
	host := strings.ToLower(u.Host)
	for _, blocked := range lt.config.BlockedDomains {
		if strings.Contains(host, strings.ToLower(blocked)) {
			return false
		}
	}

	// Block obvious malicious patterns
	maliciousPatterns := []string{
		"javascript:",
		"data:",
		"vbscript:",
		"file:",
	}
	loweredURL := strings.ToLower(u.String())
	for _, pattern := range maliciousPatterns {
		if strings.Contains(loweredURL, pattern) {
			return false
		}
	}

	return true
}

// checkAndCacheClick checks if this is a unique click and caches it
func (lt *LinkTracker) checkAndCacheClick(emailID, recipientID, linkID string) bool {
	if !lt.config.CacheClickEvents {
		return true // Can't determine, assume unique
	}

	key := fmt.Sprintf("%s:%s:%s", emailID, recipientID, linkID)

	lt.clickCacheMu.Lock()
	defer lt.clickCacheMu.Unlock()

	if _, exists := lt.clickCache[key]; exists {
		lt.metrics.mu.Lock()
		lt.metrics.DuplicateClicks++
		lt.metrics.mu.Unlock()
		return false
	}

	lt.clickCache[key] = time.Now()
	lt.metrics.mu.Lock()
	lt.metrics.UniqueClicks++
	lt.metrics.mu.Unlock()
	return true
}

// recordClick records a link click event
func (lt *LinkTracker) recordClick(ctx context.Context, tokenData *LinkTokenData, link *TrackedLink, r *http.Request, isUnique bool) {
	// Skip recording if storage is not configured
	if lt.storage == nil {
		return
	}

	// Get client info
	ipAddress := lt.getClientIP(r)
	userAgent := r.Header.Get("User-Agent")

	// Parse user agent
	var deviceType, clientType string
	if lt.uaParser != nil {
		uaInfo := lt.uaParser.Parse(userAgent)
		if uaInfo != nil {
			deviceType = uaInfo.DeviceType
			clientType = uaInfo.ClientType
		}
	}

	// Get geo info
	var country, city string
	if lt.geoLocator != nil {
		geoInfo, err := lt.geoLocator.Lookup(ipAddress)
		if err == nil && geoInfo != nil {
			country = geoInfo.Country
			city = geoInfo.City
		}
	}

	// Get email info for campaign ID
	var campaignID, recipientEmail string
	email, err := lt.storage.GetTrackedEmail(ctx, tokenData.EmailID)
	if err == nil && email != nil {
		campaignID = email.CampaignID
		recipientEmail = email.RecipientEmail
	}

	// Check uniqueness from database if not cached
	if !lt.config.CacheClickEvents {
		isUnique, _ = lt.storage.CheckUniqueEvent(ctx, tokenData.EmailID, tokenData.RecipientID, EventTypeClick)
	}

	// Create tracking event
	event := &TrackingEvent{
		EmailID:     tokenData.EmailID,
		RecipientID: tokenData.RecipientID,
		Email:       recipientEmail,
		EventType:   EventTypeClick,
		Timestamp:   time.Now(),
		IPAddress:   ipAddress,
		UserAgent:   userAgent,
		DeviceType:  deviceType,
		ClientType:  clientType,
		Country:     country,
		City:        city,
		CampaignID:  campaignID,
		LinkID:      tokenData.LinkID,
		LinkURL:     link.OriginalURL,
		IsUnique:    isUnique,
		Metadata: map[string]interface{}{
			"link_text":     link.LinkText,
			"link_position": link.Position,
			"referer":       r.Header.Get("Referer"),
		},
	}

	// Record the event
	err = lt.storage.RecordEvent(ctx, event)
	if err != nil {
		lt.metrics.mu.Lock()
		lt.metrics.ErrorCount++
		lt.metrics.mu.Unlock()
		return
	}

	// Update link stats
	lt.storage.UpdateLinkStats(ctx, tokenData.LinkID, isUnique)

	// Update email stats
	lt.storage.UpdateTrackedEmailStats(ctx, tokenData.EmailID, EventTypeClick)
}

// getClientIP extracts the client IP from the request
func (lt *LinkTracker) getClientIP(r *http.Request) string {
	// Check X-Forwarded-For header
	xff := r.Header.Get("X-Forwarded-For")
	if xff != "" {
		ips := strings.Split(xff, ",")
		if len(ips) > 0 {
			return strings.TrimSpace(ips[0])
		}
	}

	// Check X-Real-IP header
	xri := r.Header.Get("X-Real-IP")
	if xri != "" {
		return xri
	}

	// Fall back to RemoteAddr
	ip := r.RemoteAddr
	if colonIdx := strings.LastIndex(ip, ":"); colonIdx != -1 {
		ip = ip[:colonIdx]
	}
	return strings.Trim(ip, "[]")
}

// cleanupCache periodically removes expired entries from the cache
func (lt *LinkTracker) cleanupCache() {
	ticker := time.NewTicker(10 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		lt.clickCacheMu.Lock()
		cutoff := time.Now().Add(-lt.config.CacheTTL)
		for key, timestamp := range lt.clickCache {
			if timestamp.Before(cutoff) {
				delete(lt.clickCache, key)
			}
		}
		lt.clickCacheMu.Unlock()
	}
}

// GetMetrics returns current link tracking metrics
func (lt *LinkTracker) GetMetrics() *LinkMetrics {
	lt.metrics.mu.RLock()
	defer lt.metrics.mu.RUnlock()

	return &LinkMetrics{
		TotalClicks:      lt.metrics.TotalClicks,
		UniqueClicks:     lt.metrics.UniqueClicks,
		DuplicateClicks:  lt.metrics.DuplicateClicks,
		InvalidTokens:    lt.metrics.InvalidTokens,
		BlockedRedirects: lt.metrics.BlockedRedirects,
		OptedOutRequests: lt.metrics.OptedOutRequests,
		DNTRequests:      lt.metrics.DNTRequests,
		ErrorCount:       lt.metrics.ErrorCount,
	}
}

// HTTPHandler returns an http.Handler for the link tracking endpoint
func (lt *LinkTracker) HTTPHandler() http.Handler {
	return http.HandlerFunc(lt.HandleClickRequest)
}

// GetLinkPerformance retrieves performance metrics for all links in an email
func (lt *LinkTracker) GetLinkPerformance(ctx context.Context, emailID string) ([]*LinkPerformance, error) {
	links, err := lt.storage.GetLinksByEmail(ctx, emailID)
	if err != nil {
		return nil, err
	}

	performances := make([]*LinkPerformance, len(links))
	for i, link := range links {
		performances[i] = &LinkPerformance{
			LinkID:       link.ID,
			OriginalURL:  link.OriginalURL,
			LinkText:     link.LinkText,
			Position:     link.Position,
			TotalClicks:  link.ClickCount,
			UniqueClicks: link.UniqueClicks,
		}
	}

	return performances, nil
}

// LinkPerformance contains performance data for a tracked link
type LinkPerformance struct {
	LinkID       string  `json:"link_id"`
	OriginalURL  string  `json:"original_url"`
	LinkText     string  `json:"link_text,omitempty"`
	Position     int     `json:"position"`
	TotalClicks  int     `json:"total_clicks"`
	UniqueClicks int     `json:"unique_clicks"`
	ClickRate    float64 `json:"click_rate,omitempty"` // Filled by analytics
}

// LinkRewriter provides a builder interface for link rewriting
type LinkRewriter struct {
	tracker         *LinkTracker
	emailID         string
	recipientID     string
	preserveAnchors bool
	excludePatterns []string
}

// NewLinkRewriter creates a new link rewriter
func NewLinkRewriter(tracker *LinkTracker, emailID, recipientID string) *LinkRewriter {
	return &LinkRewriter{
		tracker:         tracker,
		emailID:         emailID,
		recipientID:     recipientID,
		excludePatterns: []string{"unsubscribe"},
	}
}

// PreserveAnchors sets whether to preserve anchor links (#)
func (r *LinkRewriter) PreserveAnchors(preserve bool) *LinkRewriter {
	r.preserveAnchors = preserve
	return r
}

// ExcludePatterns sets URL patterns to exclude from tracking
func (r *LinkRewriter) ExcludePatterns(patterns []string) *LinkRewriter {
	r.excludePatterns = patterns
	return r
}

// Rewrite rewrites all links in the HTML content
func (r *LinkRewriter) Rewrite(ctx context.Context, htmlContent string) (string, error) {
	return r.tracker.RewriteLinksWithText(ctx, htmlContent, r.emailID, r.recipientID)
}
