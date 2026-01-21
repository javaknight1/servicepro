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

// Standard errors for pixel tracking
var (
	ErrInvalidToken          = errors.New("invalid tracking token")
	ErrExpiredToken          = errors.New("tracking token has expired")
	ErrPixelTrackingDisabled = errors.New("pixel tracking is disabled for this email")
	ErrOptedOut              = errors.New("recipient has opted out of tracking")
)

// Transparent 1x1 GIF pixel (43 bytes)
var transparentPixel = []byte{
	0x47, 0x49, 0x46, 0x38, 0x39, 0x61, 0x01, 0x00, 0x01, 0x00,
	0x80, 0x00, 0x00, 0xff, 0xff, 0xff, 0x00, 0x00, 0x00, 0x21,
	0xf9, 0x04, 0x01, 0x00, 0x00, 0x00, 0x00, 0x2c, 0x00, 0x00,
	0x00, 0x00, 0x01, 0x00, 0x01, 0x00, 0x00, 0x02, 0x02, 0x44,
	0x01, 0x00, 0x3b,
}

// PixelTrackerConfig holds configuration for pixel tracking
type PixelTrackerConfig struct {
	BaseURL         string        `json:"base_url"`          // Base URL for tracking endpoints
	SecretKey       string        `json:"-"`                 // Secret key for token signing
	TokenExpiration time.Duration `json:"token_expiration"`  // How long tokens are valid
	CacheOpenEvents bool          `json:"cache_open_events"` // Cache recent opens to detect duplicates
	CacheTTL        time.Duration `json:"cache_ttl"`         // Cache TTL
	RespectDNT      bool          `json:"respect_dnt"`       // Respect Do Not Track header
	AnonymizeIP     bool          `json:"anonymize_ip"`      // Anonymize IP addresses
}

// DefaultPixelTrackerConfig returns sensible defaults
func DefaultPixelTrackerConfig() *PixelTrackerConfig {
	return &PixelTrackerConfig{
		BaseURL:         "https://track.example.com",
		TokenExpiration: 90 * 24 * time.Hour, // 90 days
		CacheOpenEvents: true,
		CacheTTL:        24 * time.Hour,
		RespectDNT:      true,
		AnonymizeIP:     false,
	}
}

// PixelTracker handles email open tracking via tracking pixels
type PixelTracker struct {
	config      *PixelTrackerConfig
	storage     *TrackingStorage
	optoutStore OptoutStore
	geoLocator  GeoLocator
	uaParser    UserAgentParser

	// Cache for deduplication
	openCache   map[string]time.Time
	openCacheMu sync.RWMutex

	// Metrics
	metrics *PixelMetrics
}

// PixelMetrics tracks pixel tracking statistics
type PixelMetrics struct {
	mu               sync.RWMutex
	TotalRequests    int64
	UniqueOpens      int64
	DuplicateOpens   int64
	InvalidTokens    int64
	OptedOutRequests int64
	DNTRequests      int64
	ErrorCount       int64
}

// GeoLocator interface for IP geolocation
type GeoLocator interface {
	Lookup(ip string) (*GeoInfo, error)
}

// GeoInfo contains geolocation data
type GeoInfo struct {
	Country   string  `json:"country"`
	City      string  `json:"city"`
	Region    string  `json:"region"`
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}

// UserAgentParser interface for parsing user agents
type UserAgentParser interface {
	Parse(userAgent string) *UserAgentInfo
}

// UserAgentInfo contains parsed user agent data
type UserAgentInfo struct {
	Browser    string `json:"browser"`
	BrowserVer string `json:"browser_version"`
	OS         string `json:"os"`
	OSVersion  string `json:"os_version"`
	DeviceType string `json:"device_type"` // desktop, mobile, tablet
	ClientType string `json:"client_type"` // browser, email_client, bot
	IsBot      bool   `json:"is_bot"`
}

// NewPixelTracker creates a new pixel tracker
func NewPixelTracker(config *PixelTrackerConfig, storage *TrackingStorage, optoutStore OptoutStore) *PixelTracker {
	if config == nil {
		config = DefaultPixelTrackerConfig()
	}

	pt := &PixelTracker{
		config:      config,
		storage:     storage,
		optoutStore: optoutStore,
		openCache:   make(map[string]time.Time),
		metrics:     &PixelMetrics{},
	}

	// Start cache cleanup goroutine
	if config.CacheOpenEvents {
		go pt.cleanupCache()
	}

	return pt
}

// SetGeoLocator sets the geo locator for IP lookup
func (pt *PixelTracker) SetGeoLocator(gl GeoLocator) {
	pt.geoLocator = gl
}

// SetUserAgentParser sets the user agent parser
func (pt *PixelTracker) SetUserAgentParser(uap UserAgentParser) {
	pt.uaParser = uap
}

// GeneratePixelURL generates a tracking pixel URL for an email
func (pt *PixelTracker) GeneratePixelURL(emailID, recipientID string) (string, error) {
	token, err := pt.generateToken(emailID, recipientID)
	if err != nil {
		return "", err
	}

	pixelURL := fmt.Sprintf("%s/pixel/%s.gif", pt.config.BaseURL, token)
	return pixelURL, nil
}

// GeneratePixelHTML generates the HTML for embedding a tracking pixel
func (pt *PixelTracker) GeneratePixelHTML(emailID, recipientID string) (string, error) {
	url, err := pt.GeneratePixelURL(emailID, recipientID)
	if err != nil {
		return "", err
	}

	// Use multiple methods to improve tracking reliability
	html := fmt.Sprintf(`<!-- Email tracking pixel -->
<img src="%s" width="1" height="1" alt="" style="display:block;width:1px;height:1px;border:0;" />
<div style="display:none;font-size:1px;color:#ffffff;line-height:1px;max-height:0px;max-width:0px;opacity:0;overflow:hidden;">
<img src="%s" width="1" height="1" />
</div>`, url, url)

	return html, nil
}

// InjectPixel injects a tracking pixel into HTML email content
func (pt *PixelTracker) InjectPixel(htmlContent, emailID, recipientID string) (string, error) {
	pixelHTML, err := pt.GeneratePixelHTML(emailID, recipientID)
	if err != nil {
		return "", err
	}

	// Try to inject before </body> tag
	bodyCloseRegex := regexp.MustCompile(`(?i)</body>`)
	if bodyCloseRegex.MatchString(htmlContent) {
		return bodyCloseRegex.ReplaceAllString(htmlContent, pixelHTML+"</body>"), nil
	}

	// Fallback: append to end
	return htmlContent + pixelHTML, nil
}

// generateToken creates a signed tracking token
func (pt *PixelTracker) generateToken(emailID, recipientID string) (string, error) {
	// Token format: emailID|recipientID|timestamp|signature
	timestamp := time.Now().Unix()
	data := fmt.Sprintf("%s|%s|%d", emailID, recipientID, timestamp)

	signature := pt.sign(data)
	token := fmt.Sprintf("%s|%s", data, signature)

	// Base64 URL encode
	return base64.RawURLEncoding.EncodeToString([]byte(token)), nil
}

// ParseToken validates and parses a tracking token
func (pt *PixelTracker) ParseToken(token string) (*TokenData, error) {
	// Decode base64
	decoded, err := base64.RawURLEncoding.DecodeString(token)
	if err != nil {
		return nil, ErrInvalidToken
	}

	parts := strings.Split(string(decoded), "|")
	if len(parts) != 4 {
		return nil, ErrInvalidToken
	}

	emailID := parts[0]
	recipientID := parts[1]
	timestamp := parts[2]
	signature := parts[3]

	// Verify signature
	data := fmt.Sprintf("%s|%s|%s", emailID, recipientID, timestamp)
	if !pt.verify(data, signature) {
		return nil, ErrInvalidToken
	}

	// Parse timestamp
	var ts int64
	fmt.Sscanf(timestamp, "%d", &ts)
	tokenTime := time.Unix(ts, 0)

	// Check expiration
	if pt.config.TokenExpiration > 0 && time.Since(tokenTime) > pt.config.TokenExpiration {
		return nil, ErrExpiredToken
	}

	return &TokenData{
		EmailID:     emailID,
		RecipientID: recipientID,
		Timestamp:   tokenTime,
	}, nil
}

// TokenData contains parsed token information
type TokenData struct {
	EmailID     string
	RecipientID string
	Timestamp   time.Time
}

// sign creates an HMAC signature
func (pt *PixelTracker) sign(data string) string {
	h := hmac.New(sha256.New, []byte(pt.config.SecretKey))
	h.Write([]byte(data))
	return hex.EncodeToString(h.Sum(nil))
}

// verify checks an HMAC signature
func (pt *PixelTracker) verify(data, signature string) bool {
	expected := pt.sign(data)
	return hmac.Equal([]byte(expected), []byte(signature))
}

// HandlePixelRequest handles an HTTP request for the tracking pixel
func (pt *PixelTracker) HandlePixelRequest(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	pt.metrics.mu.Lock()
	pt.metrics.TotalRequests++
	pt.metrics.mu.Unlock()

	// Extract token from URL (e.g., /pixel/{token}.gif)
	path := r.URL.Path
	token := strings.TrimPrefix(path, "/pixel/")
	token = strings.TrimSuffix(token, ".gif")

	// Respect Do Not Track header
	if pt.config.RespectDNT && r.Header.Get("DNT") == "1" {
		pt.metrics.mu.Lock()
		pt.metrics.DNTRequests++
		pt.metrics.mu.Unlock()
		pt.servePixel(w)
		return
	}

	// Parse token
	tokenData, err := pt.ParseToken(token)
	if err != nil {
		pt.metrics.mu.Lock()
		pt.metrics.InvalidTokens++
		pt.metrics.mu.Unlock()
		pt.servePixel(w)
		return
	}

	// Check if opted out
	if pt.optoutStore != nil {
		optedOut, _ := pt.optoutStore.IsOptedOut(ctx, tokenData.RecipientID)
		if optedOut {
			pt.metrics.mu.Lock()
			pt.metrics.OptedOutRequests++
			pt.metrics.mu.Unlock()
			pt.servePixel(w)
			return
		}
	}

	// Check for duplicate within cache window
	isUnique := pt.checkAndCacheOpen(tokenData.EmailID, tokenData.RecipientID)

	// Record the open event
	go pt.recordOpen(ctx, tokenData, r, isUnique)

	// Serve the pixel
	pt.servePixel(w)
}

// servePixel writes the transparent pixel to the response
func (pt *PixelTracker) servePixel(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "image/gif")
	w.Header().Set("Content-Length", fmt.Sprintf("%d", len(transparentPixel)))
	w.Header().Set("Cache-Control", "no-store, no-cache, must-revalidate, max-age=0")
	w.Header().Set("Pragma", "no-cache")
	w.Header().Set("Expires", "0")
	w.WriteHeader(http.StatusOK)
	w.Write(transparentPixel)
}

// checkAndCacheOpen checks if this is a unique open and caches it
func (pt *PixelTracker) checkAndCacheOpen(emailID, recipientID string) bool {
	if !pt.config.CacheOpenEvents {
		return true // Can't determine, assume unique
	}

	key := fmt.Sprintf("%s:%s", emailID, recipientID)

	pt.openCacheMu.Lock()
	defer pt.openCacheMu.Unlock()

	if _, exists := pt.openCache[key]; exists {
		pt.metrics.mu.Lock()
		pt.metrics.DuplicateOpens++
		pt.metrics.mu.Unlock()
		return false
	}

	pt.openCache[key] = time.Now()
	pt.metrics.mu.Lock()
	pt.metrics.UniqueOpens++
	pt.metrics.mu.Unlock()
	return true
}

// recordOpen records an email open event
func (pt *PixelTracker) recordOpen(ctx context.Context, tokenData *TokenData, r *http.Request, isUnique bool) {
	// Skip recording if storage is not configured
	if pt.storage == nil {
		return
	}

	// Get client info
	ipAddress := pt.getClientIP(r)
	if pt.config.AnonymizeIP {
		ipAddress = anonymizeIP(ipAddress)
	}
	userAgent := r.Header.Get("User-Agent")

	// Parse user agent
	var deviceType, clientType string
	if pt.uaParser != nil {
		uaInfo := pt.uaParser.Parse(userAgent)
		if uaInfo != nil {
			deviceType = uaInfo.DeviceType
			clientType = uaInfo.ClientType
		}
	}

	// Get geo info
	var country, city string
	if pt.geoLocator != nil {
		geoInfo, err := pt.geoLocator.Lookup(ipAddress)
		if err == nil && geoInfo != nil {
			country = geoInfo.Country
			city = geoInfo.City
		}
	}

	// Get email info for campaign ID
	var campaignID string
	email, err := pt.storage.GetTrackedEmail(ctx, tokenData.EmailID)
	if err == nil && email != nil {
		campaignID = email.CampaignID
	}

	// Check uniqueness from database if not cached
	if !pt.config.CacheOpenEvents {
		isUnique, _ = pt.storage.CheckUniqueEvent(ctx, tokenData.EmailID, tokenData.RecipientID, EventTypeOpen)
	}

	// Create tracking event
	event := &TrackingEvent{
		EmailID:     tokenData.EmailID,
		RecipientID: tokenData.RecipientID,
		EventType:   EventTypeOpen,
		Timestamp:   time.Now(),
		IPAddress:   ipAddress,
		UserAgent:   userAgent,
		DeviceType:  deviceType,
		ClientType:  clientType,
		Country:     country,
		City:        city,
		CampaignID:  campaignID,
		IsUnique:    isUnique,
		Metadata: map[string]interface{}{
			"referer": r.Header.Get("Referer"),
			"accept":  r.Header.Get("Accept"),
		},
	}

	// Record the event
	err = pt.storage.RecordEvent(ctx, event)
	if err != nil {
		pt.metrics.mu.Lock()
		pt.metrics.ErrorCount++
		pt.metrics.mu.Unlock()
		return
	}

	// Update email stats
	pt.storage.UpdateTrackedEmailStats(ctx, tokenData.EmailID, EventTypeOpen)
}

// getClientIP extracts the client IP from the request
func (pt *PixelTracker) getClientIP(r *http.Request) string {
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

// anonymizeIP masks the last octet of an IPv4 or last 80 bits of IPv6
func anonymizeIP(ip string) string {
	if strings.Contains(ip, ".") {
		// IPv4: mask last octet
		parts := strings.Split(ip, ".")
		if len(parts) == 4 {
			return fmt.Sprintf("%s.%s.%s.0", parts[0], parts[1], parts[2])
		}
	} else if strings.Contains(ip, ":") {
		// IPv6: mask last 80 bits (keep first 48 bits)
		parts := strings.Split(ip, ":")
		if len(parts) >= 3 {
			return fmt.Sprintf("%s:%s:%s::", parts[0], parts[1], parts[2])
		}
	}
	return ip
}

// cleanupCache periodically removes expired entries from the cache
func (pt *PixelTracker) cleanupCache() {
	ticker := time.NewTicker(10 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		pt.openCacheMu.Lock()
		cutoff := time.Now().Add(-pt.config.CacheTTL)
		for key, timestamp := range pt.openCache {
			if timestamp.Before(cutoff) {
				delete(pt.openCache, key)
			}
		}
		pt.openCacheMu.Unlock()
	}
}

// GetMetrics returns current pixel tracking metrics
func (pt *PixelTracker) GetMetrics() *PixelMetrics {
	pt.metrics.mu.RLock()
	defer pt.metrics.mu.RUnlock()

	return &PixelMetrics{
		TotalRequests:    pt.metrics.TotalRequests,
		UniqueOpens:      pt.metrics.UniqueOpens,
		DuplicateOpens:   pt.metrics.DuplicateOpens,
		InvalidTokens:    pt.metrics.InvalidTokens,
		OptedOutRequests: pt.metrics.OptedOutRequests,
		DNTRequests:      pt.metrics.DNTRequests,
		ErrorCount:       pt.metrics.ErrorCount,
	}
}

// HTTPHandler returns an http.Handler for the pixel tracking endpoint
func (pt *PixelTracker) HTTPHandler() http.Handler {
	return http.HandlerFunc(pt.HandlePixelRequest)
}

// SimpleUserAgentParser provides basic user agent parsing
type SimpleUserAgentParser struct{}

// Parse implements UserAgentParser
func (s *SimpleUserAgentParser) Parse(userAgent string) *UserAgentInfo {
	ua := strings.ToLower(userAgent)
	info := &UserAgentInfo{}

	// Detect bots
	botPatterns := []string{"bot", "crawler", "spider", "googlebot", "bingbot", "slurp", "duckduck"}
	for _, pattern := range botPatterns {
		if strings.Contains(ua, pattern) {
			info.IsBot = true
			info.ClientType = "bot"
			return info
		}
	}

	// Detect device type
	if strings.Contains(ua, "mobile") || strings.Contains(ua, "android") {
		if strings.Contains(ua, "tablet") || strings.Contains(ua, "ipad") {
			info.DeviceType = "tablet"
		} else {
			info.DeviceType = "mobile"
		}
	} else {
		info.DeviceType = "desktop"
	}

	// Detect email clients
	emailClients := []string{"outlook", "thunderbird", "apple mail", "gmail", "yahoo"}
	for _, client := range emailClients {
		if strings.Contains(ua, client) {
			info.ClientType = "email_client"
			return info
		}
	}

	// Detect browsers
	if strings.Contains(ua, "chrome") {
		info.Browser = "Chrome"
		info.ClientType = "browser"
	} else if strings.Contains(ua, "firefox") {
		info.Browser = "Firefox"
		info.ClientType = "browser"
	} else if strings.Contains(ua, "safari") {
		info.Browser = "Safari"
		info.ClientType = "browser"
	} else if strings.Contains(ua, "edge") {
		info.Browser = "Edge"
		info.ClientType = "browser"
	} else {
		info.ClientType = "unknown"
	}

	// Detect OS
	if strings.Contains(ua, "windows") {
		info.OS = "Windows"
	} else if strings.Contains(ua, "mac os") || strings.Contains(ua, "macos") {
		info.OS = "macOS"
	} else if strings.Contains(ua, "linux") {
		info.OS = "Linux"
	} else if strings.Contains(ua, "android") {
		info.OS = "Android"
	} else if strings.Contains(ua, "ios") || strings.Contains(ua, "iphone") || strings.Contains(ua, "ipad") {
		info.OS = "iOS"
	}

	return info
}

// PixelURLBuilder helps build tracking pixel URLs
type PixelURLBuilder struct {
	baseURL string
	tracker *PixelTracker
}

// NewPixelURLBuilder creates a new URL builder
func NewPixelURLBuilder(tracker *PixelTracker) *PixelURLBuilder {
	return &PixelURLBuilder{
		baseURL: tracker.config.BaseURL,
		tracker: tracker,
	}
}

// Build creates a tracking pixel URL
func (b *PixelURLBuilder) Build(emailID, recipientID string) (*url.URL, error) {
	urlStr, err := b.tracker.GeneratePixelURL(emailID, recipientID)
	if err != nil {
		return nil, err
	}
	return url.Parse(urlStr)
}

// BuildWithParams creates a URL with additional query parameters
func (b *PixelURLBuilder) BuildWithParams(emailID, recipientID string, params map[string]string) (*url.URL, error) {
	u, err := b.Build(emailID, recipientID)
	if err != nil {
		return nil, err
	}

	q := u.Query()
	for key, value := range params {
		q.Set(key, value)
	}
	u.RawQuery = q.Encode()

	return u, nil
}
