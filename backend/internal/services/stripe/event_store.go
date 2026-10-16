package stripe

import (
	"context"
	"errors"
	"sync"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// ClaimResult says what to do with a delivered event.
type ClaimResult int

const (
	// ClaimProcess means this delivery owns the event and must process it.
	ClaimProcess ClaimResult = iota
	// ClaimDuplicate means the event was already processed: acknowledge it.
	ClaimDuplicate
	// ClaimInProgress means another delivery is processing it right now:
	// answer non-2xx so Stripe retries later.
	ClaimInProgress
)

// staleClaimAfter is how long a "processing" claim may sit before another
// delivery may take over (the original instance crashed mid-processing).
const staleClaimAfter = 10 * time.Minute

// EventStore deduplicates Stripe events across restarts and instances (F-10).
type EventStore interface {
	Claim(ctx context.Context, eventID, eventType string) (ClaimResult, error)
	MarkProcessed(ctx context.Context, eventID string) error
	// Release drops a claim after a processing failure so Stripe's retry is
	// processed again.
	Release(ctx context.Context, eventID string) error
}

// StripeWebhookEvent is a row of stripe_webhook_events (migration 000006).
type StripeWebhookEvent struct {
	EventID     string     `gorm:"column:event_id;primaryKey"`
	EventType   string     `gorm:"column:event_type"`
	Status      string     `gorm:"column:status"`
	ReceivedAt  time.Time  `gorm:"column:received_at"`
	ProcessedAt *time.Time `gorm:"column:processed_at"`
}

// TableName specifies the table name.
func (StripeWebhookEvent) TableName() string { return "stripe_webhook_events" }

const (
	eventStatusProcessing = "processing"
	eventStatusProcessed  = "processed"
)

// GormEventStore persists processed event IDs in Postgres.
type GormEventStore struct {
	db *gorm.DB
}

// NewGormEventStore creates a persistent event store.
func NewGormEventStore(db *gorm.DB) *GormEventStore { return &GormEventStore{db: db} }

// Claim inserts the event ID (INSERT ... ON CONFLICT DO NOTHING) before
// processing; the first delivery to insert it wins.
func (s *GormEventStore) Claim(ctx context.Context, eventID, eventType string) (ClaimResult, error) {
	now := time.Now()
	res := s.db.WithContext(ctx).Clauses(clause.OnConflict{DoNothing: true}).Create(&StripeWebhookEvent{
		EventID: eventID, EventType: eventType, Status: eventStatusProcessing, ReceivedAt: now,
	})
	if res.Error != nil {
		return ClaimInProgress, res.Error
	}
	if res.RowsAffected == 1 {
		return ClaimProcess, nil
	}

	var existing StripeWebhookEvent
	if err := s.db.WithContext(ctx).First(&existing, "event_id = ?", eventID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ClaimInProgress, nil // released concurrently; let Stripe retry
		}
		return ClaimInProgress, err
	}
	if existing.Status == eventStatusProcessed {
		return ClaimDuplicate, nil
	}
	// Take over a stale claim left by a crashed instance.
	res = s.db.WithContext(ctx).Model(&StripeWebhookEvent{}).
		Where("event_id = ? AND status = ? AND received_at < ?", eventID, eventStatusProcessing, now.Add(-staleClaimAfter)).
		Update("received_at", now)
	if res.Error != nil {
		return ClaimInProgress, res.Error
	}
	if res.RowsAffected == 1 {
		return ClaimProcess, nil
	}
	return ClaimInProgress, nil
}

// MarkProcessed records successful processing.
func (s *GormEventStore) MarkProcessed(ctx context.Context, eventID string) error {
	now := time.Now()
	return s.db.WithContext(ctx).Model(&StripeWebhookEvent{}).
		Where("event_id = ?", eventID).
		Updates(map[string]interface{}{"status": eventStatusProcessed, "processed_at": now}).Error
}

// Release removes an unprocessed claim.
func (s *GormEventStore) Release(ctx context.Context, eventID string) error {
	return s.db.WithContext(ctx).
		Where("event_id = ? AND status = ?", eventID, eventStatusProcessing).
		Delete(&StripeWebhookEvent{}).Error
}

// memoryEventStore is the per-process fallback used when no database store is
// configured (tests, local tools). It does not survive restarts.
type memoryEventStore struct {
	mu     sync.Mutex
	events map[string]string
}

func newMemoryEventStore() *memoryEventStore {
	return &memoryEventStore{events: make(map[string]string)}
}

func (m *memoryEventStore) Claim(_ context.Context, eventID, _ string) (ClaimResult, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	switch m.events[eventID] {
	case eventStatusProcessed:
		return ClaimDuplicate, nil
	case eventStatusProcessing:
		return ClaimInProgress, nil
	}
	m.events[eventID] = eventStatusProcessing
	return ClaimProcess, nil
}

func (m *memoryEventStore) MarkProcessed(_ context.Context, eventID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.events[eventID] = eventStatusProcessed
	return nil
}

func (m *memoryEventStore) Release(_ context.Context, eventID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.events[eventID] == eventStatusProcessing {
		delete(m.events, eventID)
	}
	return nil
}

func (m *memoryEventStore) count() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.events)
}
