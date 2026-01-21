package tracking

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"
)

// Standard errors for opt-out handling
var (
	ErrAlreadyOptedOut    = errors.New("recipient has already opted out")
	ErrNotOptedOut        = errors.New("recipient is not opted out")
	ErrInvalidOptoutToken = errors.New("invalid opt-out token")
	ErrOptoutExpired      = errors.New("opt-out token has expired")
)

// OptoutType represents the type of opt-out
type OptoutType string

const (
	OptoutTypeTracking      OptoutType = "tracking"      // Opt out of tracking only
	OptoutTypeMarketing     OptoutType = "marketing"     // Opt out of marketing emails
	OptoutTypeTransactional OptoutType = "transactional" // Opt out of transactional (rare)
	OptoutTypeAll           OptoutType = "all"           // Opt out of all emails
)

// OptoutRecord represents an opt-out record
type OptoutRecord struct {
	ID          string     `json:"id" db:"id"`
	RecipientID string     `json:"recipient_id" db:"recipient_id"`
	Email       string     `json:"email" db:"email"`
	OptoutType  OptoutType `json:"optout_type" db:"optout_type"`
	Reason      string     `json:"reason,omitempty" db:"reason"`
	Source      string     `json:"source,omitempty" db:"source"` // link, preference_center, api, complaint
	CampaignID  string     `json:"campaign_id,omitempty" db:"campaign_id"`
	IPAddress   string     `json:"ip_address,omitempty" db:"ip_address"`
	UserAgent   string     `json:"user_agent,omitempty" db:"user_agent"`
	CreatedAt   time.Time  `json:"created_at" db:"created_at"`
	RevokedAt   *time.Time `json:"revoked_at,omitempty" db:"revoked_at"`
	IsActive    bool       `json:"is_active" db:"is_active"`
}

// OptoutStore defines the interface for opt-out storage
type OptoutStore interface {
	IsOptedOut(ctx context.Context, recipientID string) (bool, error)
	IsOptedOutForType(ctx context.Context, recipientID string, optoutType OptoutType) (bool, error)
	GetOptoutRecord(ctx context.Context, recipientID string) (*OptoutRecord, error)
	RecordOptout(ctx context.Context, record *OptoutRecord) error
	RevokeOptout(ctx context.Context, recipientID string, optoutType OptoutType) error
	GetOptoutsByEmail(ctx context.Context, email string) ([]*OptoutRecord, error)
	GetRecentOptouts(ctx context.Context, since time.Time, limit int) ([]*OptoutRecord, error)
}

// OptoutConfig holds configuration for opt-out handling
type OptoutConfig struct {
	BaseURL             string        `json:"base_url"`
	SecretKey           string        `json:"-"`
	TokenExpiration     time.Duration `json:"token_expiration"`
	RequireConfirmation bool          `json:"require_confirmation"` // Require confirmation page
	AllowResubscribe    bool          `json:"allow_resubscribe"`    // Allow users to resubscribe
	DefaultType         OptoutType    `json:"default_type"`
	TablePrefix         string        `json:"table_prefix"`
	CacheEnabled        bool          `json:"cache_enabled"`
	CacheTTL            time.Duration `json:"cache_ttl"`
}

// DefaultOptoutConfig returns sensible defaults
func DefaultOptoutConfig() *OptoutConfig {
	return &OptoutConfig{
		BaseURL:             "https://track.example.com",
		TokenExpiration:     90 * 24 * time.Hour,
		RequireConfirmation: false,
		AllowResubscribe:    true,
		DefaultType:         OptoutTypeTracking,
		TablePrefix:         "email_tracking_",
		CacheEnabled:        true,
		CacheTTL:            5 * time.Minute,
	}
}

// PostgresOptoutStore implements OptoutStore using PostgreSQL
type PostgresOptoutStore struct {
	db     *sql.DB
	config *OptoutConfig

	// Cache for quick lookups
	cache   map[string]*OptoutRecord
	cacheMu sync.RWMutex
}

// NewPostgresOptoutStore creates a new PostgreSQL-backed opt-out store
func NewPostgresOptoutStore(db *sql.DB, config *OptoutConfig) *PostgresOptoutStore {
	if config == nil {
		config = DefaultOptoutConfig()
	}

	store := &PostgresOptoutStore{
		db:     db,
		config: config,
		cache:  make(map[string]*OptoutRecord),
	}

	if config.CacheEnabled {
		go store.cleanupCache()
	}

	return store
}

// cleanupCache periodically clears the cache
func (s *PostgresOptoutStore) cleanupCache() {
	ticker := time.NewTicker(s.config.CacheTTL)
	defer ticker.Stop()

	for range ticker.C {
		s.cacheMu.Lock()
		s.cache = make(map[string]*OptoutRecord)
		s.cacheMu.Unlock()
	}
}

// IsOptedOut checks if a recipient has opted out of any tracking
func (s *PostgresOptoutStore) IsOptedOut(ctx context.Context, recipientID string) (bool, error) {
	// Check cache
	if s.config.CacheEnabled {
		s.cacheMu.RLock()
		if record, exists := s.cache[recipientID]; exists {
			s.cacheMu.RUnlock()
			return record.IsActive, nil
		}
		s.cacheMu.RUnlock()
	}

	query := fmt.Sprintf(`
		SELECT EXISTS(
			SELECT 1 FROM %soptouts
			WHERE recipient_id = $1 AND is_active = TRUE
		)
	`, s.config.TablePrefix)

	var exists bool
	err := s.db.QueryRowContext(ctx, query, recipientID).Scan(&exists)
	if err != nil {
		return false, err
	}

	return exists, nil
}

// IsOptedOutForType checks if a recipient has opted out of a specific type
func (s *PostgresOptoutStore) IsOptedOutForType(ctx context.Context, recipientID string, optoutType OptoutType) (bool, error) {
	query := fmt.Sprintf(`
		SELECT EXISTS(
			SELECT 1 FROM %soptouts
			WHERE recipient_id = $1 AND is_active = TRUE
			AND (optout_type = $2 OR optout_type = 'all')
		)
	`, s.config.TablePrefix)

	var exists bool
	err := s.db.QueryRowContext(ctx, query, recipientID, optoutType).Scan(&exists)
	if err != nil {
		return false, err
	}

	return exists, nil
}

// GetOptoutRecord retrieves the opt-out record for a recipient
func (s *PostgresOptoutStore) GetOptoutRecord(ctx context.Context, recipientID string) (*OptoutRecord, error) {
	// Check cache
	if s.config.CacheEnabled {
		s.cacheMu.RLock()
		if record, exists := s.cache[recipientID]; exists {
			s.cacheMu.RUnlock()
			return record, nil
		}
		s.cacheMu.RUnlock()
	}

	query := fmt.Sprintf(`
		SELECT id, recipient_id, email, optout_type, reason, source, campaign_id,
			ip_address, user_agent, created_at, revoked_at, is_active
		FROM %soptouts
		WHERE recipient_id = $1 AND is_active = TRUE
		ORDER BY created_at DESC
		LIMIT 1
	`, s.config.TablePrefix)

	record := &OptoutRecord{}
	err := s.db.QueryRowContext(ctx, query, recipientID).Scan(
		&record.ID, &record.RecipientID, &record.Email, &record.OptoutType,
		&record.Reason, &record.Source, &record.CampaignID, &record.IPAddress,
		&record.UserAgent, &record.CreatedAt, &record.RevokedAt, &record.IsActive,
	)

	if err == sql.ErrNoRows {
		return nil, ErrNotOptedOut
	}
	if err != nil {
		return nil, err
	}

	// Update cache
	if s.config.CacheEnabled {
		s.cacheMu.Lock()
		s.cache[recipientID] = record
		s.cacheMu.Unlock()
	}

	return record, nil
}

// RecordOptout records a new opt-out
func (s *PostgresOptoutStore) RecordOptout(ctx context.Context, record *OptoutRecord) error {
	// Check if already opted out
	existing, err := s.GetOptoutRecord(ctx, record.RecipientID)
	if err == nil && existing != nil && existing.IsActive {
		return ErrAlreadyOptedOut
	}

	if record.ID == "" {
		record.ID = generateUUID()
	}
	if record.CreatedAt.IsZero() {
		record.CreatedAt = time.Now()
	}
	record.IsActive = true

	query := fmt.Sprintf(`
		INSERT INTO %soptouts (
			id, recipient_id, email, optout_type, reason, source, campaign_id,
			ip_address, user_agent, created_at, is_active
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	`, s.config.TablePrefix)

	_, err = s.db.ExecContext(ctx, query,
		record.ID, record.RecipientID, record.Email, record.OptoutType,
		record.Reason, record.Source, record.CampaignID, record.IPAddress,
		record.UserAgent, record.CreatedAt, record.IsActive,
	)

	if err != nil {
		return err
	}

	// Update cache
	if s.config.CacheEnabled {
		s.cacheMu.Lock()
		s.cache[record.RecipientID] = record
		s.cacheMu.Unlock()
	}

	return nil
}

// RevokeOptout revokes an opt-out (resubscribe)
func (s *PostgresOptoutStore) RevokeOptout(ctx context.Context, recipientID string, optoutType OptoutType) error {
	if !s.config.AllowResubscribe {
		return errors.New("resubscription is not allowed")
	}

	query := fmt.Sprintf(`
		UPDATE %soptouts SET
			is_active = FALSE,
			revoked_at = $2
		WHERE recipient_id = $1 AND is_active = TRUE AND optout_type = $3
	`, s.config.TablePrefix)

	result, err := s.db.ExecContext(ctx, query, recipientID, time.Now(), optoutType)
	if err != nil {
		return err
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return ErrNotOptedOut
	}

	// Clear cache
	if s.config.CacheEnabled {
		s.cacheMu.Lock()
		delete(s.cache, recipientID)
		s.cacheMu.Unlock()
	}

	return nil
}

// GetOptoutsByEmail retrieves all opt-out records for an email address
func (s *PostgresOptoutStore) GetOptoutsByEmail(ctx context.Context, email string) ([]*OptoutRecord, error) {
	query := fmt.Sprintf(`
		SELECT id, recipient_id, email, optout_type, reason, source, campaign_id,
			ip_address, user_agent, created_at, revoked_at, is_active
		FROM %soptouts
		WHERE email = $1
		ORDER BY created_at DESC
	`, s.config.TablePrefix)

	rows, err := s.db.QueryContext(ctx, query, email)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var records []*OptoutRecord
	for rows.Next() {
		record := &OptoutRecord{}
		err := rows.Scan(
			&record.ID, &record.RecipientID, &record.Email, &record.OptoutType,
			&record.Reason, &record.Source, &record.CampaignID, &record.IPAddress,
			&record.UserAgent, &record.CreatedAt, &record.RevokedAt, &record.IsActive,
		)
		if err != nil {
			continue
		}
		records = append(records, record)
	}

	return records, nil
}

// GetRecentOptouts retrieves recent opt-out records
func (s *PostgresOptoutStore) GetRecentOptouts(ctx context.Context, since time.Time, limit int) ([]*OptoutRecord, error) {
	query := fmt.Sprintf(`
		SELECT id, recipient_id, email, optout_type, reason, source, campaign_id,
			ip_address, user_agent, created_at, revoked_at, is_active
		FROM %soptouts
		WHERE created_at >= $1
		ORDER BY created_at DESC
		LIMIT $2
	`, s.config.TablePrefix)

	rows, err := s.db.QueryContext(ctx, query, since, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var records []*OptoutRecord
	for rows.Next() {
		record := &OptoutRecord{}
		err := rows.Scan(
			&record.ID, &record.RecipientID, &record.Email, &record.OptoutType,
			&record.Reason, &record.Source, &record.CampaignID, &record.IPAddress,
			&record.UserAgent, &record.CreatedAt, &record.RevokedAt, &record.IsActive,
		)
		if err != nil {
			continue
		}
		records = append(records, record)
	}

	return records, nil
}

// OptoutManager handles opt-out operations
type OptoutManager struct {
	config  *OptoutConfig
	store   OptoutStore
	storage *TrackingStorage

	metrics *OptoutMetrics
}

// OptoutMetrics tracks opt-out statistics
type OptoutMetrics struct {
	mu                sync.RWMutex
	TotalOptouts      int64
	TotalResubscribes int64
	ByType            map[OptoutType]int64
	BySource          map[string]int64
}

// NewOptoutManager creates a new opt-out manager
func NewOptoutManager(config *OptoutConfig, store OptoutStore, storage *TrackingStorage) *OptoutManager {
	if config == nil {
		config = DefaultOptoutConfig()
	}

	return &OptoutManager{
		config:  config,
		store:   store,
		storage: storage,
		metrics: &OptoutMetrics{
			ByType:   make(map[OptoutType]int64),
			BySource: make(map[string]int64),
		},
	}
}

// GenerateOptoutURL generates an opt-out URL for a recipient
func (m *OptoutManager) GenerateOptoutURL(emailID, recipientID, email string) (string, error) {
	token, err := m.generateToken(emailID, recipientID, email)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%s/optout/%s", m.config.BaseURL, token), nil
}

// GenerateOptoutLink generates an HTML link for opt-out
func (m *OptoutManager) GenerateOptoutLink(emailID, recipientID, email, linkText string) (string, error) {
	url, err := m.GenerateOptoutURL(emailID, recipientID, email)
	if err != nil {
		return "", err
	}

	if linkText == "" {
		linkText = "Unsubscribe from tracking"
	}

	return fmt.Sprintf(`<a href="%s" style="color:#666;font-size:12px;">%s</a>`, url, linkText), nil
}

// InjectOptoutLink injects an opt-out link into HTML email content
func (m *OptoutManager) InjectOptoutLink(htmlContent, emailID, recipientID, email string) (string, error) {
	link, err := m.GenerateOptoutLink(emailID, recipientID, email, "Click here to opt out of tracking")
	if err != nil {
		return "", err
	}

	// Try to inject before </body> tag
	if strings.Contains(strings.ToLower(htmlContent), "</body>") {
		return strings.Replace(
			htmlContent,
			"</body>",
			fmt.Sprintf(`<div style="text-align:center;margin:20px;font-size:11px;color:#999;">%s</div></body>`, link),
			1,
		), nil
	}

	// Fallback: append to end
	return htmlContent + fmt.Sprintf(`<div style="text-align:center;margin:20px;font-size:11px;color:#999;">%s</div>`, link), nil
}

// generateToken creates a signed opt-out token
func (m *OptoutManager) generateToken(emailID, recipientID, email string) (string, error) {
	timestamp := time.Now().Unix()
	data := fmt.Sprintf("%s|%s|%s|%d", emailID, recipientID, email, timestamp)

	signature := m.sign(data)
	token := fmt.Sprintf("%s|%s", data, signature)

	return base64.RawURLEncoding.EncodeToString([]byte(token)), nil
}

// ParseOptoutToken validates and parses an opt-out token
func (m *OptoutManager) ParseOptoutToken(token string) (*OptoutTokenData, error) {
	decoded, err := base64.RawURLEncoding.DecodeString(token)
	if err != nil {
		return nil, ErrInvalidOptoutToken
	}

	parts := strings.Split(string(decoded), "|")
	if len(parts) != 5 {
		return nil, ErrInvalidOptoutToken
	}

	emailID := parts[0]
	recipientID := parts[1]
	email := parts[2]
	timestamp := parts[3]
	signature := parts[4]

	// Verify signature
	data := fmt.Sprintf("%s|%s|%s|%s", emailID, recipientID, email, timestamp)
	if !m.verify(data, signature) {
		return nil, ErrInvalidOptoutToken
	}

	// Parse timestamp
	var ts int64
	fmt.Sscanf(timestamp, "%d", &ts)
	tokenTime := time.Unix(ts, 0)

	// Check expiration
	if m.config.TokenExpiration > 0 && time.Since(tokenTime) > m.config.TokenExpiration {
		return nil, ErrOptoutExpired
	}

	return &OptoutTokenData{
		EmailID:     emailID,
		RecipientID: recipientID,
		Email:       email,
		Timestamp:   tokenTime,
	}, nil
}

// OptoutTokenData contains parsed opt-out token information
type OptoutTokenData struct {
	EmailID     string
	RecipientID string
	Email       string
	Timestamp   time.Time
}

// sign creates an HMAC signature
func (m *OptoutManager) sign(data string) string {
	h := hmac.New(sha256.New, []byte(m.config.SecretKey))
	h.Write([]byte(data))
	return hex.EncodeToString(h.Sum(nil))
}

// verify checks an HMAC signature
func (m *OptoutManager) verify(data, signature string) bool {
	expected := m.sign(data)
	return hmac.Equal([]byte(expected), []byte(signature))
}

// ProcessOptout processes an opt-out request
func (m *OptoutManager) ProcessOptout(ctx context.Context, tokenData *OptoutTokenData, optoutType OptoutType, reason string, r *http.Request) error {
	// Get client info
	var ipAddress, userAgent string
	if r != nil {
		ipAddress = getClientIP(r)
		userAgent = r.Header.Get("User-Agent")
	}

	// Create opt-out record
	record := &OptoutRecord{
		RecipientID: tokenData.RecipientID,
		Email:       tokenData.Email,
		OptoutType:  optoutType,
		Reason:      reason,
		Source:      "link",
		CampaignID:  getCampaignFromEmail(ctx, m.storage, tokenData.EmailID),
		IPAddress:   ipAddress,
		UserAgent:   userAgent,
	}

	err := m.store.RecordOptout(ctx, record)
	if err != nil {
		return err
	}

	// Update metrics
	m.metrics.mu.Lock()
	m.metrics.TotalOptouts++
	m.metrics.ByType[optoutType]++
	m.metrics.BySource["link"]++
	m.metrics.mu.Unlock()

	// Record tracking event
	if m.storage != nil {
		event := &TrackingEvent{
			EmailID:     tokenData.EmailID,
			RecipientID: tokenData.RecipientID,
			Email:       tokenData.Email,
			EventType:   EventTypeUnsubscribe,
			Timestamp:   time.Now(),
			IPAddress:   ipAddress,
			UserAgent:   userAgent,
			Metadata: map[string]interface{}{
				"optout_type": string(optoutType),
				"reason":      reason,
			},
		}
		m.storage.RecordEvent(ctx, event)
	}

	return nil
}

// ProcessResubscribe processes a resubscribe request
func (m *OptoutManager) ProcessResubscribe(ctx context.Context, recipientID string, optoutType OptoutType) error {
	err := m.store.RevokeOptout(ctx, recipientID, optoutType)
	if err != nil {
		return err
	}

	m.metrics.mu.Lock()
	m.metrics.TotalResubscribes++
	m.metrics.mu.Unlock()

	return nil
}

// HandleOptoutRequest handles an HTTP request for opt-out
func (m *OptoutManager) HandleOptoutRequest(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Extract token from URL
	path := r.URL.Path
	token := strings.TrimPrefix(path, "/optout/")

	// Parse token
	tokenData, err := m.ParseOptoutToken(token)
	if err != nil {
		http.Error(w, "Invalid or expired opt-out link", http.StatusBadRequest)
		return
	}

	// Check if already opted out
	optedOut, _ := m.store.IsOptedOut(ctx, tokenData.RecipientID)
	if optedOut {
		m.serveAlreadyOptedOutPage(w)
		return
	}

	// Handle GET - show confirmation page
	if r.Method == http.MethodGet && m.config.RequireConfirmation {
		m.serveConfirmationPage(w, token, tokenData)
		return
	}

	// Process opt-out
	optoutType := m.config.DefaultType
	if t := r.URL.Query().Get("type"); t != "" {
		optoutType = OptoutType(t)
	}

	reason := r.URL.Query().Get("reason")
	if r.Method == http.MethodPost {
		r.ParseForm()
		if t := r.FormValue("type"); t != "" {
			optoutType = OptoutType(t)
		}
		if re := r.FormValue("reason"); re != "" {
			reason = re
		}
	}

	err = m.ProcessOptout(ctx, tokenData, optoutType, reason, r)
	if err != nil {
		http.Error(w, "Failed to process opt-out", http.StatusInternalServerError)
		return
	}

	m.serveSuccessPage(w, tokenData)
}

// serveConfirmationPage serves the opt-out confirmation page
func (m *OptoutManager) serveConfirmationPage(w http.ResponseWriter, token string, tokenData *OptoutTokenData) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprintf(w, `<!DOCTYPE html>
<html>
<head>
	<title>Confirm Opt-Out</title>
	<style>
		body { font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif; margin: 0; padding: 40px; background: #f5f5f5; }
		.container { max-width: 500px; margin: 0 auto; background: white; padding: 40px; border-radius: 8px; box-shadow: 0 2px 4px rgba(0,0,0,0.1); }
		h1 { color: #333; margin-top: 0; }
		p { color: #666; line-height: 1.6; }
		.email { font-weight: bold; color: #333; }
		form { margin-top: 20px; }
		label { display: block; margin: 15px 0 5px; color: #333; }
		select, textarea { width: 100%%; padding: 10px; border: 1px solid #ddd; border-radius: 4px; font-size: 14px; }
		button { background: #dc3545; color: white; border: none; padding: 12px 24px; font-size: 16px; border-radius: 4px; cursor: pointer; margin-top: 20px; }
		button:hover { background: #c82333; }
		.cancel { background: #6c757d; margin-left: 10px; }
		.cancel:hover { background: #5a6268; }
	</style>
</head>
<body>
	<div class="container">
		<h1>Confirm Opt-Out</h1>
		<p>You are about to opt out of email tracking for:</p>
		<p class="email">%s</p>
		<form method="POST" action="/optout/%s">
			<label for="type">Opt-out from:</label>
			<select name="type" id="type">
				<option value="tracking">Email tracking only</option>
				<option value="marketing">Marketing emails</option>
				<option value="all">All emails</option>
			</select>
			<label for="reason">Reason (optional):</label>
			<textarea name="reason" id="reason" rows="3" placeholder="Tell us why you're opting out..."></textarea>
			<button type="submit">Confirm Opt-Out</button>
			<a href="javascript:history.back()"><button type="button" class="cancel">Cancel</button></a>
		</form>
	</div>
</body>
</html>`, tokenData.Email, token)
}

// serveSuccessPage serves the opt-out success page
func (m *OptoutManager) serveSuccessPage(w http.ResponseWriter, tokenData *OptoutTokenData) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprintf(w, `<!DOCTYPE html>
<html>
<head>
	<title>Opted Out Successfully</title>
	<style>
		body { font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif; margin: 0; padding: 40px; background: #f5f5f5; }
		.container { max-width: 500px; margin: 0 auto; background: white; padding: 40px; border-radius: 8px; box-shadow: 0 2px 4px rgba(0,0,0,0.1); text-align: center; }
		h1 { color: #28a745; margin-top: 0; }
		p { color: #666; line-height: 1.6; }
		.checkmark { font-size: 64px; color: #28a745; }
	</style>
</head>
<body>
	<div class="container">
		<div class="checkmark">&#10003;</div>
		<h1>You've Opted Out</h1>
		<p>You have been successfully opted out of email tracking.</p>
		<p>Email: <strong>%s</strong></p>
		<p>You can close this window now.</p>
	</div>
</body>
</html>`, tokenData.Email)
}

// serveAlreadyOptedOutPage serves a page for already opted-out users
func (m *OptoutManager) serveAlreadyOptedOutPage(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprint(w, `<!DOCTYPE html>
<html>
<head>
	<title>Already Opted Out</title>
	<style>
		body { font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif; margin: 0; padding: 40px; background: #f5f5f5; }
		.container { max-width: 500px; margin: 0 auto; background: white; padding: 40px; border-radius: 8px; box-shadow: 0 2px 4px rgba(0,0,0,0.1); text-align: center; }
		h1 { color: #17a2b8; margin-top: 0; }
		p { color: #666; line-height: 1.6; }
		.info { font-size: 64px; color: #17a2b8; }
	</style>
</head>
<body>
	<div class="container">
		<div class="info">&#8505;</div>
		<h1>Already Opted Out</h1>
		<p>You have already opted out of email tracking.</p>
		<p>No further action is needed.</p>
	</div>
</body>
</html>`)
}

// HTTPHandler returns an http.Handler for the opt-out endpoint
func (m *OptoutManager) HTTPHandler() http.Handler {
	return http.HandlerFunc(m.HandleOptoutRequest)
}

// GetMetrics returns current opt-out metrics
func (m *OptoutManager) GetMetrics() *OptoutMetrics {
	m.metrics.mu.RLock()
	defer m.metrics.mu.RUnlock()

	return &OptoutMetrics{
		TotalOptouts:      m.metrics.TotalOptouts,
		TotalResubscribes: m.metrics.TotalResubscribes,
		ByType:            copyMap(m.metrics.ByType),
		BySource:          copyStringMap(m.metrics.BySource),
	}
}

// Helper functions

func getClientIP(r *http.Request) string {
	xff := r.Header.Get("X-Forwarded-For")
	if xff != "" {
		ips := strings.Split(xff, ",")
		if len(ips) > 0 {
			return strings.TrimSpace(ips[0])
		}
	}

	xri := r.Header.Get("X-Real-IP")
	if xri != "" {
		return xri
	}

	ip := r.RemoteAddr
	if colonIdx := strings.LastIndex(ip, ":"); colonIdx != -1 {
		ip = ip[:colonIdx]
	}
	return strings.Trim(ip, "[]")
}

func getCampaignFromEmail(ctx context.Context, storage *TrackingStorage, emailID string) string {
	if storage == nil {
		return ""
	}
	email, err := storage.GetTrackedEmail(ctx, emailID)
	if err != nil {
		return ""
	}
	return email.CampaignID
}

func generateUUID() string {
	// Simple UUID generation
	b := make([]byte, 16)
	for i := range b {
		b[i] = byte(time.Now().UnixNano() >> (i * 8))
	}
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}

func copyMap(m map[OptoutType]int64) map[OptoutType]int64 {
	result := make(map[OptoutType]int64, len(m))
	for k, v := range m {
		result[k] = v
	}
	return result
}

func copyStringMap(m map[string]int64) map[string]int64 {
	result := make(map[string]int64, len(m))
	for k, v := range m {
		result[k] = v
	}
	return result
}

// GetOptoutMigrationSQL returns SQL for creating opt-out tables
func GetOptoutMigrationSQL(tablePrefix string) string {
	return fmt.Sprintf(`
-- Opt-out records table
CREATE TABLE IF NOT EXISTS %soptouts (
	id UUID PRIMARY KEY,
	recipient_id VARCHAR(255) NOT NULL,
	email VARCHAR(255) NOT NULL,
	optout_type VARCHAR(50) NOT NULL,
	reason TEXT,
	source VARCHAR(50),
	campaign_id VARCHAR(255),
	ip_address VARCHAR(45),
	user_agent TEXT,
	created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
	revoked_at TIMESTAMPTZ,
	is_active BOOLEAN DEFAULT TRUE
);

CREATE INDEX IF NOT EXISTS idx_%soptouts_recipient_id ON %soptouts(recipient_id);
CREATE INDEX IF NOT EXISTS idx_%soptouts_email ON %soptouts(email);
CREATE INDEX IF NOT EXISTS idx_%soptouts_is_active ON %soptouts(is_active);
CREATE INDEX IF NOT EXISTS idx_%soptouts_created_at ON %soptouts(created_at);
`,
		tablePrefix,
		tablePrefix, tablePrefix,
		tablePrefix, tablePrefix,
		tablePrefix, tablePrefix,
		tablePrefix, tablePrefix,
	)
}
