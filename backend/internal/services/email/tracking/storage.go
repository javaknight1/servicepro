package tracking

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

// Standard errors
var (
	ErrEventNotFound      = errors.New("tracking event not found")
	ErrInvalidEventType   = errors.New("invalid event type")
	ErrStorageClosed      = errors.New("storage is closed")
	ErrRetentionViolation = errors.New("data retention policy violation")
)

// EventType represents the type of tracking event
type EventType string

const (
	EventTypeOpen        EventType = "open"
	EventTypeClick       EventType = "click"
	EventTypeBounce      EventType = "bounce"
	EventTypeUnsubscribe EventType = "unsubscribe"
	EventTypeComplaint   EventType = "complaint"
	EventTypeDelivery    EventType = "delivery"
)

// TrackingEvent represents a single tracking event
type TrackingEvent struct {
	ID          string                 `json:"id" db:"id"`
	EmailID     string                 `json:"email_id" db:"email_id"`
	RecipientID string                 `json:"recipient_id" db:"recipient_id"`
	Email       string                 `json:"email" db:"email"`
	EventType   EventType              `json:"event_type" db:"event_type"`
	Timestamp   time.Time              `json:"timestamp" db:"timestamp"`
	IPAddress   string                 `json:"ip_address,omitempty" db:"ip_address"`
	UserAgent   string                 `json:"user_agent,omitempty" db:"user_agent"`
	Metadata    map[string]interface{} `json:"metadata,omitempty" db:"metadata"`
	CampaignID  string                 `json:"campaign_id,omitempty" db:"campaign_id"`
	LinkID      string                 `json:"link_id,omitempty" db:"link_id"`
	LinkURL     string                 `json:"link_url,omitempty" db:"link_url"`
	Country     string                 `json:"country,omitempty" db:"country"`
	City        string                 `json:"city,omitempty" db:"city"`
	DeviceType  string                 `json:"device_type,omitempty" db:"device_type"`
	ClientType  string                 `json:"client_type,omitempty" db:"client_type"`
	IsUnique    bool                   `json:"is_unique" db:"is_unique"`
	CreatedAt   time.Time              `json:"created_at" db:"created_at"`
}

// TrackedEmail represents an email being tracked
type TrackedEmail struct {
	ID             string     `json:"id" db:"id"`
	RecipientID    string     `json:"recipient_id" db:"recipient_id"`
	RecipientEmail string     `json:"recipient_email" db:"recipient_email"`
	CampaignID     string     `json:"campaign_id,omitempty" db:"campaign_id"`
	Subject        string     `json:"subject" db:"subject"`
	SentAt         time.Time  `json:"sent_at" db:"sent_at"`
	TrackingToken  string     `json:"tracking_token" db:"tracking_token"`
	PixelEnabled   bool       `json:"pixel_enabled" db:"pixel_enabled"`
	LinkTracking   bool       `json:"link_tracking" db:"link_tracking"`
	OpenCount      int        `json:"open_count" db:"open_count"`
	ClickCount     int        `json:"click_count" db:"click_count"`
	FirstOpenAt    *time.Time `json:"first_open_at,omitempty" db:"first_open_at"`
	LastOpenAt     *time.Time `json:"last_open_at,omitempty" db:"last_open_at"`
	FirstClickAt   *time.Time `json:"first_click_at,omitempty" db:"first_click_at"`
	LastClickAt    *time.Time `json:"last_click_at,omitempty" db:"last_click_at"`
	CreatedAt      time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at" db:"updated_at"`
}

// TrackedLink represents a link being tracked in an email
type TrackedLink struct {
	ID           string    `json:"id" db:"id"`
	EmailID      string    `json:"email_id" db:"email_id"`
	OriginalURL  string    `json:"original_url" db:"original_url"`
	TrackingURL  string    `json:"tracking_url" db:"tracking_url"`
	LinkText     string    `json:"link_text,omitempty" db:"link_text"`
	Position     int       `json:"position" db:"position"`
	ClickCount   int       `json:"click_count" db:"click_count"`
	UniqueClicks int       `json:"unique_clicks" db:"unique_clicks"`
	CreatedAt    time.Time `json:"created_at" db:"created_at"`
	UpdatedAt    time.Time `json:"updated_at" db:"updated_at"`
}

// RetentionPolicy defines data retention rules
type RetentionPolicy struct {
	EventRetention     time.Duration `json:"event_retention"`      // How long to keep individual events
	AggregateRetention time.Duration `json:"aggregate_retention"`  // How long to keep aggregated data
	AnonymizeAfter     time.Duration `json:"anonymize_after"`      // When to anonymize PII
	DeleteIPAfter      time.Duration `json:"delete_ip_after"`      // When to delete IP addresses
	CompactEventsAfter time.Duration `json:"compact_events_after"` // When to compact events to aggregates
}

// DefaultRetentionPolicy returns sensible defaults for GDPR compliance
func DefaultRetentionPolicy() *RetentionPolicy {
	return &RetentionPolicy{
		EventRetention:     90 * 24 * time.Hour,  // 90 days
		AggregateRetention: 365 * 24 * time.Hour, // 1 year
		AnonymizeAfter:     30 * 24 * time.Hour,  // 30 days
		DeleteIPAfter:      7 * 24 * time.Hour,   // 7 days
		CompactEventsAfter: 30 * 24 * time.Hour,  // 30 days
	}
}

// StorageConfig holds configuration for the tracking storage
type StorageConfig struct {
	MaxBatchSize    int              `json:"max_batch_size"`
	FlushInterval   time.Duration    `json:"flush_interval"`
	RetentionPolicy *RetentionPolicy `json:"retention_policy"`
	TablePrefix     string           `json:"table_prefix"`
}

// DefaultStorageConfig returns default storage configuration
func DefaultStorageConfig() *StorageConfig {
	return &StorageConfig{
		MaxBatchSize:    1000,
		FlushInterval:   5 * time.Second,
		RetentionPolicy: DefaultRetentionPolicy(),
		TablePrefix:     "email_tracking_",
	}
}

// TrackingStorage handles persistence of tracking data
type TrackingStorage struct {
	db     *sql.DB
	config *StorageConfig
	closed bool

	// Event buffer for batch inserts
	eventBuffer chan *TrackingEvent
	stopChan    chan struct{}
	doneChan    chan struct{}
}

// NewTrackingStorage creates a new tracking storage instance
func NewTrackingStorage(db *sql.DB, config *StorageConfig) (*TrackingStorage, error) {
	if db == nil {
		return nil, errors.New("database connection is required")
	}

	if config == nil {
		config = DefaultStorageConfig()
	}

	ts := &TrackingStorage{
		db:          db,
		config:      config,
		eventBuffer: make(chan *TrackingEvent, config.MaxBatchSize*2),
		stopChan:    make(chan struct{}),
		doneChan:    make(chan struct{}),
	}

	// Start background flusher
	go ts.flushLoop()

	return ts, nil
}

// flushLoop periodically flushes buffered events to the database
func (ts *TrackingStorage) flushLoop() {
	defer close(ts.doneChan)

	ticker := time.NewTicker(ts.config.FlushInterval)
	defer ticker.Stop()

	batch := make([]*TrackingEvent, 0, ts.config.MaxBatchSize)

	for {
		select {
		case <-ts.stopChan:
			// Flush remaining events
			if len(batch) > 0 {
				ts.flushBatch(context.Background(), batch)
			}
			// Drain the buffer
			for {
				select {
				case event := <-ts.eventBuffer:
					batch = append(batch, event)
					if len(batch) >= ts.config.MaxBatchSize {
						ts.flushBatch(context.Background(), batch)
						batch = batch[:0]
					}
				default:
					if len(batch) > 0 {
						ts.flushBatch(context.Background(), batch)
					}
					return
				}
			}

		case event := <-ts.eventBuffer:
			batch = append(batch, event)
			if len(batch) >= ts.config.MaxBatchSize {
				ts.flushBatch(context.Background(), batch)
				batch = batch[:0]
			}

		case <-ticker.C:
			if len(batch) > 0 {
				ts.flushBatch(context.Background(), batch)
				batch = batch[:0]
			}
		}
	}
}

// flushBatch writes a batch of events to the database
func (ts *TrackingStorage) flushBatch(ctx context.Context, events []*TrackingEvent) error {
	if len(events) == 0 {
		return nil
	}

	// Build batch insert query
	valueStrings := make([]string, 0, len(events))
	valueArgs := make([]interface{}, 0, len(events)*15)

	for i, event := range events {
		offset := i * 15
		valueStrings = append(valueStrings, fmt.Sprintf(
			"($%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d)",
			offset+1, offset+2, offset+3, offset+4, offset+5, offset+6, offset+7, offset+8,
			offset+9, offset+10, offset+11, offset+12, offset+13, offset+14, offset+15,
		))

		metadata, _ := json.Marshal(event.Metadata)

		valueArgs = append(valueArgs,
			event.ID, event.EmailID, event.RecipientID, event.Email, event.EventType,
			event.Timestamp, event.IPAddress, event.UserAgent, metadata, event.CampaignID,
			event.LinkID, event.LinkURL, event.Country, event.DeviceType, event.IsUnique,
		)
	}

	query := fmt.Sprintf(`
		INSERT INTO %sevents (
			id, email_id, recipient_id, email, event_type, timestamp, ip_address,
			user_agent, metadata, campaign_id, link_id, link_url, country, device_type, is_unique
		) VALUES %s
		ON CONFLICT (id) DO NOTHING
	`, ts.config.TablePrefix, strings.Join(valueStrings, ","))

	_, err := ts.db.ExecContext(ctx, query, valueArgs...)
	return err
}

// RecordEvent records a tracking event (buffered for batch insert)
func (ts *TrackingStorage) RecordEvent(ctx context.Context, event *TrackingEvent) error {
	if ts.closed {
		return ErrStorageClosed
	}

	if event.ID == "" {
		event.ID = uuid.New().String()
	}
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now()
	}
	if event.CreatedAt.IsZero() {
		event.CreatedAt = time.Now()
	}

	select {
	case ts.eventBuffer <- event:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// RecordEventSync records an event synchronously (for critical events)
func (ts *TrackingStorage) RecordEventSync(ctx context.Context, event *TrackingEvent) error {
	if ts.closed {
		return ErrStorageClosed
	}

	if event.ID == "" {
		event.ID = uuid.New().String()
	}
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now()
	}
	if event.CreatedAt.IsZero() {
		event.CreatedAt = time.Now()
	}

	metadata, _ := json.Marshal(event.Metadata)

	query := fmt.Sprintf(`
		INSERT INTO %sevents (
			id, email_id, recipient_id, email, event_type, timestamp, ip_address,
			user_agent, metadata, campaign_id, link_id, link_url, country, device_type, is_unique
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)
		ON CONFLICT (id) DO NOTHING
	`, ts.config.TablePrefix)

	_, err := ts.db.ExecContext(ctx, query,
		event.ID, event.EmailID, event.RecipientID, event.Email, event.EventType,
		event.Timestamp, event.IPAddress, event.UserAgent, metadata, event.CampaignID,
		event.LinkID, event.LinkURL, event.Country, event.DeviceType, event.IsUnique,
	)

	return err
}

// CreateTrackedEmail creates a new tracked email record
func (ts *TrackingStorage) CreateTrackedEmail(ctx context.Context, email *TrackedEmail) error {
	if ts.closed {
		return ErrStorageClosed
	}

	if email.ID == "" {
		email.ID = uuid.New().String()
	}
	if email.TrackingToken == "" {
		email.TrackingToken = uuid.New().String()
	}
	if email.CreatedAt.IsZero() {
		email.CreatedAt = time.Now()
	}
	email.UpdatedAt = time.Now()

	query := fmt.Sprintf(`
		INSERT INTO %semails (
			id, recipient_id, recipient_email, campaign_id, subject, sent_at,
			tracking_token, pixel_enabled, link_tracking, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	`, ts.config.TablePrefix)

	_, err := ts.db.ExecContext(ctx, query,
		email.ID, email.RecipientID, email.RecipientEmail, email.CampaignID,
		email.Subject, email.SentAt, email.TrackingToken, email.PixelEnabled,
		email.LinkTracking, email.CreatedAt, email.UpdatedAt,
	)

	return err
}

// GetTrackedEmail retrieves a tracked email by ID or token
func (ts *TrackingStorage) GetTrackedEmail(ctx context.Context, idOrToken string) (*TrackedEmail, error) {
	if ts.closed {
		return nil, ErrStorageClosed
	}

	query := fmt.Sprintf(`
		SELECT id, recipient_id, recipient_email, campaign_id, subject, sent_at,
			tracking_token, pixel_enabled, link_tracking, open_count, click_count,
			first_open_at, last_open_at, first_click_at, last_click_at, created_at, updated_at
		FROM %semails
		WHERE id = $1 OR tracking_token = $1
	`, ts.config.TablePrefix)

	email := &TrackedEmail{}
	err := ts.db.QueryRowContext(ctx, query, idOrToken).Scan(
		&email.ID, &email.RecipientID, &email.RecipientEmail, &email.CampaignID,
		&email.Subject, &email.SentAt, &email.TrackingToken, &email.PixelEnabled,
		&email.LinkTracking, &email.OpenCount, &email.ClickCount, &email.FirstOpenAt,
		&email.LastOpenAt, &email.FirstClickAt, &email.LastClickAt, &email.CreatedAt, &email.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, ErrEventNotFound
	}

	return email, err
}

// UpdateTrackedEmailStats updates open/click statistics for an email
func (ts *TrackingStorage) UpdateTrackedEmailStats(ctx context.Context, emailID string, eventType EventType) error {
	if ts.closed {
		return ErrStorageClosed
	}

	now := time.Now()
	var query string

	switch eventType {
	case EventTypeOpen:
		query = fmt.Sprintf(`
			UPDATE %semails SET
				open_count = open_count + 1,
				first_open_at = COALESCE(first_open_at, $2),
				last_open_at = $2,
				updated_at = $2
			WHERE id = $1
		`, ts.config.TablePrefix)
	case EventTypeClick:
		query = fmt.Sprintf(`
			UPDATE %semails SET
				click_count = click_count + 1,
				first_click_at = COALESCE(first_click_at, $2),
				last_click_at = $2,
				updated_at = $2
			WHERE id = $1
		`, ts.config.TablePrefix)
	default:
		return nil
	}

	_, err := ts.db.ExecContext(ctx, query, emailID, now)
	return err
}

// CreateTrackedLink creates a new tracked link record
func (ts *TrackingStorage) CreateTrackedLink(ctx context.Context, link *TrackedLink) error {
	if ts.closed {
		return ErrStorageClosed
	}

	if link.ID == "" {
		link.ID = uuid.New().String()
	}
	if link.CreatedAt.IsZero() {
		link.CreatedAt = time.Now()
	}
	link.UpdatedAt = time.Now()

	query := fmt.Sprintf(`
		INSERT INTO %slinks (
			id, email_id, original_url, tracking_url, link_text, position, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`, ts.config.TablePrefix)

	_, err := ts.db.ExecContext(ctx, query,
		link.ID, link.EmailID, link.OriginalURL, link.TrackingURL,
		link.LinkText, link.Position, link.CreatedAt, link.UpdatedAt,
	)

	return err
}

// GetTrackedLink retrieves a tracked link by ID
func (ts *TrackingStorage) GetTrackedLink(ctx context.Context, linkID string) (*TrackedLink, error) {
	if ts.closed {
		return nil, ErrStorageClosed
	}

	query := fmt.Sprintf(`
		SELECT id, email_id, original_url, tracking_url, link_text, position,
			click_count, unique_clicks, created_at, updated_at
		FROM %slinks
		WHERE id = $1
	`, ts.config.TablePrefix)

	link := &TrackedLink{}
	err := ts.db.QueryRowContext(ctx, query, linkID).Scan(
		&link.ID, &link.EmailID, &link.OriginalURL, &link.TrackingURL,
		&link.LinkText, &link.Position, &link.ClickCount, &link.UniqueClicks,
		&link.CreatedAt, &link.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, ErrEventNotFound
	}

	return link, err
}

// UpdateLinkStats updates click statistics for a link
func (ts *TrackingStorage) UpdateLinkStats(ctx context.Context, linkID string, isUnique bool) error {
	if ts.closed {
		return ErrStorageClosed
	}

	query := fmt.Sprintf(`
		UPDATE %slinks SET
			click_count = click_count + 1,
			unique_clicks = unique_clicks + CASE WHEN $2 THEN 1 ELSE 0 END,
			updated_at = $3
		WHERE id = $1
	`, ts.config.TablePrefix)

	_, err := ts.db.ExecContext(ctx, query, linkID, isUnique, time.Now())
	return err
}

// GetLinksByEmail retrieves all tracked links for an email
func (ts *TrackingStorage) GetLinksByEmail(ctx context.Context, emailID string) ([]*TrackedLink, error) {
	if ts.closed {
		return nil, ErrStorageClosed
	}

	query := fmt.Sprintf(`
		SELECT id, email_id, original_url, tracking_url, link_text, position,
			click_count, unique_clicks, created_at, updated_at
		FROM %slinks
		WHERE email_id = $1
		ORDER BY position
	`, ts.config.TablePrefix)

	rows, err := ts.db.QueryContext(ctx, query, emailID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var links []*TrackedLink
	for rows.Next() {
		link := &TrackedLink{}
		err := rows.Scan(
			&link.ID, &link.EmailID, &link.OriginalURL, &link.TrackingURL,
			&link.LinkText, &link.Position, &link.ClickCount, &link.UniqueClicks,
			&link.CreatedAt, &link.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		links = append(links, link)
	}

	return links, rows.Err()
}

// GetEvents retrieves events with optional filtering
func (ts *TrackingStorage) GetEvents(ctx context.Context, filter EventFilter) ([]*TrackingEvent, error) {
	if ts.closed {
		return nil, ErrStorageClosed
	}

	query, args := ts.buildEventQuery(filter)

	rows, err := ts.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []*TrackingEvent
	for rows.Next() {
		event := &TrackingEvent{}
		var metadata []byte
		err := rows.Scan(
			&event.ID, &event.EmailID, &event.RecipientID, &event.Email, &event.EventType,
			&event.Timestamp, &event.IPAddress, &event.UserAgent, &metadata, &event.CampaignID,
			&event.LinkID, &event.LinkURL, &event.Country, &event.DeviceType, &event.IsUnique,
			&event.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		if metadata != nil {
			json.Unmarshal(metadata, &event.Metadata)
		}
		events = append(events, event)
	}

	return events, rows.Err()
}

// EventFilter defines filters for querying events
type EventFilter struct {
	EmailID     string
	RecipientID string
	CampaignID  string
	EventType   EventType
	StartTime   time.Time
	EndTime     time.Time
	Limit       int
	Offset      int
}

// buildEventQuery constructs a query from filter parameters
func (ts *TrackingStorage) buildEventQuery(filter EventFilter) (string, []interface{}) {
	conditions := []string{}
	args := []interface{}{}
	argIndex := 1

	if filter.EmailID != "" {
		conditions = append(conditions, fmt.Sprintf("email_id = $%d", argIndex))
		args = append(args, filter.EmailID)
		argIndex++
	}

	if filter.RecipientID != "" {
		conditions = append(conditions, fmt.Sprintf("recipient_id = $%d", argIndex))
		args = append(args, filter.RecipientID)
		argIndex++
	}

	if filter.CampaignID != "" {
		conditions = append(conditions, fmt.Sprintf("campaign_id = $%d", argIndex))
		args = append(args, filter.CampaignID)
		argIndex++
	}

	if filter.EventType != "" {
		conditions = append(conditions, fmt.Sprintf("event_type = $%d", argIndex))
		args = append(args, filter.EventType)
		argIndex++
	}

	if !filter.StartTime.IsZero() {
		conditions = append(conditions, fmt.Sprintf("timestamp >= $%d", argIndex))
		args = append(args, filter.StartTime)
		argIndex++
	}

	if !filter.EndTime.IsZero() {
		conditions = append(conditions, fmt.Sprintf("timestamp <= $%d", argIndex))
		args = append(args, filter.EndTime)
		argIndex++
	}

	query := fmt.Sprintf(`
		SELECT id, email_id, recipient_id, email, event_type, timestamp, ip_address,
			user_agent, metadata, campaign_id, link_id, link_url, country, device_type,
			is_unique, created_at
		FROM %sevents
	`, ts.config.TablePrefix)

	if len(conditions) > 0 {
		query += " WHERE " + strings.Join(conditions, " AND ")
	}

	query += " ORDER BY timestamp DESC"

	if filter.Limit > 0 {
		query += fmt.Sprintf(" LIMIT $%d", argIndex)
		args = append(args, filter.Limit)
		argIndex++
	}

	if filter.Offset > 0 {
		query += fmt.Sprintf(" OFFSET $%d", argIndex)
		args = append(args, filter.Offset)
	}

	return query, args
}

// CheckUniqueEvent checks if this is a unique event for the recipient
func (ts *TrackingStorage) CheckUniqueEvent(ctx context.Context, emailID, recipientID string, eventType EventType) (bool, error) {
	if ts.closed {
		return false, ErrStorageClosed
	}

	query := fmt.Sprintf(`
		SELECT COUNT(*) FROM %sevents
		WHERE email_id = $1 AND recipient_id = $2 AND event_type = $3
	`, ts.config.TablePrefix)

	var count int
	err := ts.db.QueryRowContext(ctx, query, emailID, recipientID, eventType).Scan(&count)
	if err != nil {
		return false, err
	}

	return count == 0, nil
}

// ApplyRetentionPolicy applies data retention rules
func (ts *TrackingStorage) ApplyRetentionPolicy(ctx context.Context) (*RetentionResult, error) {
	if ts.closed {
		return nil, ErrStorageClosed
	}

	result := &RetentionResult{}
	policy := ts.config.RetentionPolicy
	now := time.Now()

	// Delete old events
	if policy.EventRetention > 0 {
		cutoff := now.Add(-policy.EventRetention)
		res, err := ts.db.ExecContext(ctx, fmt.Sprintf(`
			DELETE FROM %sevents WHERE timestamp < $1
		`, ts.config.TablePrefix), cutoff)
		if err != nil {
			return nil, fmt.Errorf("failed to delete old events: %w", err)
		}
		result.EventsDeleted, _ = res.RowsAffected()
	}

	// Anonymize PII (IP addresses, user agents)
	if policy.AnonymizeAfter > 0 {
		cutoff := now.Add(-policy.AnonymizeAfter)
		res, err := ts.db.ExecContext(ctx, fmt.Sprintf(`
			UPDATE %sevents SET
				ip_address = 'anonymized',
				user_agent = 'anonymized'
			WHERE timestamp < $1 AND ip_address != 'anonymized'
		`, ts.config.TablePrefix), cutoff)
		if err != nil {
			return nil, fmt.Errorf("failed to anonymize events: %w", err)
		}
		result.EventsAnonymized, _ = res.RowsAffected()
	}

	// Delete IP addresses earlier than general anonymization
	if policy.DeleteIPAfter > 0 && policy.DeleteIPAfter < policy.AnonymizeAfter {
		cutoff := now.Add(-policy.DeleteIPAfter)
		res, err := ts.db.ExecContext(ctx, fmt.Sprintf(`
			UPDATE %sevents SET ip_address = NULL
			WHERE timestamp < $1 AND ip_address IS NOT NULL AND ip_address != 'anonymized'
		`, ts.config.TablePrefix), cutoff)
		if err != nil {
			return nil, fmt.Errorf("failed to delete IP addresses: %w", err)
		}
		result.IPsDeleted, _ = res.RowsAffected()
	}

	return result, nil
}

// RetentionResult holds the results of applying retention policy
type RetentionResult struct {
	EventsDeleted    int64 `json:"events_deleted"`
	EventsAnonymized int64 `json:"events_anonymized"`
	IPsDeleted       int64 `json:"ips_deleted"`
	EventsCompacted  int64 `json:"events_compacted"`
}

// DeleteRecipientData deletes all tracking data for a recipient (GDPR right to erasure)
func (ts *TrackingStorage) DeleteRecipientData(ctx context.Context, recipientID string) error {
	if ts.closed {
		return ErrStorageClosed
	}

	tx, err := ts.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Delete events
	_, err = tx.ExecContext(ctx, fmt.Sprintf(`
		DELETE FROM %sevents WHERE recipient_id = $1
	`, ts.config.TablePrefix), recipientID)
	if err != nil {
		return err
	}

	// Delete tracked emails
	_, err = tx.ExecContext(ctx, fmt.Sprintf(`
		DELETE FROM %semails WHERE recipient_id = $1
	`, ts.config.TablePrefix), recipientID)
	if err != nil {
		return err
	}

	return tx.Commit()
}

// Close shuts down the storage gracefully
func (ts *TrackingStorage) Close() error {
	if ts.closed {
		return nil
	}

	ts.closed = true
	close(ts.stopChan)

	// Wait for flush to complete
	<-ts.doneChan

	return nil
}

// GetMigrationSQL returns SQL for creating tracking tables
func GetMigrationSQL(tablePrefix string) string {
	return fmt.Sprintf(`
-- Email tracking events table
CREATE TABLE IF NOT EXISTS %sevents (
	id UUID PRIMARY KEY,
	email_id UUID NOT NULL,
	recipient_id VARCHAR(255) NOT NULL,
	email VARCHAR(255) NOT NULL,
	event_type VARCHAR(50) NOT NULL,
	timestamp TIMESTAMPTZ NOT NULL,
	ip_address VARCHAR(45),
	user_agent TEXT,
	metadata JSONB,
	campaign_id VARCHAR(255),
	link_id UUID,
	link_url TEXT,
	country VARCHAR(2),
	city VARCHAR(100),
	device_type VARCHAR(50),
	client_type VARCHAR(50),
	is_unique BOOLEAN DEFAULT FALSE,
	created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_%sevents_email_id ON %sevents(email_id);
CREATE INDEX IF NOT EXISTS idx_%sevents_recipient_id ON %sevents(recipient_id);
CREATE INDEX IF NOT EXISTS idx_%sevents_campaign_id ON %sevents(campaign_id);
CREATE INDEX IF NOT EXISTS idx_%sevents_event_type ON %sevents(event_type);
CREATE INDEX IF NOT EXISTS idx_%sevents_timestamp ON %sevents(timestamp);

-- Tracked emails table
CREATE TABLE IF NOT EXISTS %semails (
	id UUID PRIMARY KEY,
	recipient_id VARCHAR(255) NOT NULL,
	recipient_email VARCHAR(255) NOT NULL,
	campaign_id VARCHAR(255),
	subject TEXT NOT NULL,
	sent_at TIMESTAMPTZ NOT NULL,
	tracking_token UUID NOT NULL UNIQUE,
	pixel_enabled BOOLEAN DEFAULT TRUE,
	link_tracking BOOLEAN DEFAULT TRUE,
	open_count INTEGER DEFAULT 0,
	click_count INTEGER DEFAULT 0,
	first_open_at TIMESTAMPTZ,
	last_open_at TIMESTAMPTZ,
	first_click_at TIMESTAMPTZ,
	last_click_at TIMESTAMPTZ,
	created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
	updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_%semails_recipient_id ON %semails(recipient_id);
CREATE INDEX IF NOT EXISTS idx_%semails_campaign_id ON %semails(campaign_id);
CREATE INDEX IF NOT EXISTS idx_%semails_tracking_token ON %semails(tracking_token);

-- Tracked links table
CREATE TABLE IF NOT EXISTS %slinks (
	id UUID PRIMARY KEY,
	email_id UUID NOT NULL REFERENCES %semails(id) ON DELETE CASCADE,
	original_url TEXT NOT NULL,
	tracking_url TEXT NOT NULL,
	link_text TEXT,
	position INTEGER NOT NULL,
	click_count INTEGER DEFAULT 0,
	unique_clicks INTEGER DEFAULT 0,
	created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
	updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_%slinks_email_id ON %slinks(email_id);

-- Aggregated statistics table (for performance)
CREATE TABLE IF NOT EXISTS %saggregates (
	id UUID PRIMARY KEY,
	date DATE NOT NULL,
	campaign_id VARCHAR(255),
	event_type VARCHAR(50) NOT NULL,
	total_count INTEGER DEFAULT 0,
	unique_count INTEGER DEFAULT 0,
	metadata JSONB,
	created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
	updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
	UNIQUE(date, campaign_id, event_type)
);

CREATE INDEX IF NOT EXISTS idx_%saggregates_date ON %saggregates(date);
CREATE INDEX IF NOT EXISTS idx_%saggregates_campaign_id ON %saggregates(campaign_id);
`,
		tablePrefix,              // events
		tablePrefix, tablePrefix, // events index
		tablePrefix, tablePrefix, // events index
		tablePrefix, tablePrefix, // events index
		tablePrefix, tablePrefix, // events index
		tablePrefix, tablePrefix, // events index
		tablePrefix,              // emails
		tablePrefix, tablePrefix, // emails index
		tablePrefix, tablePrefix, // emails index
		tablePrefix, tablePrefix, // emails index
		tablePrefix, tablePrefix, // links
		tablePrefix, tablePrefix, // links index
		tablePrefix,              // aggregates
		tablePrefix, tablePrefix, // aggregates index
		tablePrefix, tablePrefix, // aggregates index
	)
}
