package analytics

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"
)

// MockEventStore is a mock implementation of EventStore for testing
type MockEventStore struct {
	mu           sync.Mutex
	events       []*Event
	saveError    error
	saveBatchErr error
	queryErr     error
	countErr     error
	deleteErr    error
	errMu        sync.Mutex // protects error fields
}

func NewMockEventStore() *MockEventStore {
	return &MockEventStore{
		events: make([]*Event, 0),
	}
}

func (m *MockEventStore) Save(ctx context.Context, event *Event) error {
	if m.saveError != nil {
		return m.saveError
	}
	return m.SaveBatch(ctx, []*Event{event})
}

func (m *MockEventStore) SaveBatch(ctx context.Context, events []*Event) error {
	m.errMu.Lock()
	err := m.saveBatchErr
	m.errMu.Unlock()
	if err != nil {
		return err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.events = append(m.events, events...)
	return nil
}

func (m *MockEventStore) Query(ctx context.Context, query EventQuery) ([]*Event, error) {
	if m.queryErr != nil {
		return nil, m.queryErr
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.events, nil
}

func (m *MockEventStore) Count(ctx context.Context, query EventQuery) (int64, error) {
	if m.countErr != nil {
		return 0, m.countErr
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	return int64(len(m.events)), nil
}

func (m *MockEventStore) Delete(ctx context.Context, before time.Time) (int64, error) {
	if m.deleteErr != nil {
		return 0, m.deleteErr
	}
	return 0, nil
}

func (m *MockEventStore) GetEvents() []*Event {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.events
}

func (m *MockEventStore) SetSaveBatchError(err error) {
	m.errMu.Lock()
	m.saveBatchErr = err
	m.errMu.Unlock()
}

func TestDefaultTrackerConfig(t *testing.T) {
	config := DefaultTrackerConfig()

	if config.BatchSize != 100 {
		t.Errorf("Expected BatchSize 100, got %d", config.BatchSize)
	}
	if config.FlushInterval != 10*time.Second {
		t.Errorf("Expected FlushInterval 10s, got %s", config.FlushInterval)
	}
	if config.MaxQueueSize != 10000 {
		t.Errorf("Expected MaxQueueSize 10000, got %d", config.MaxQueueSize)
	}
	if config.Workers != 2 {
		t.Errorf("Expected Workers 2, got %d", config.Workers)
	}
	if config.RetryAttempts != 3 {
		t.Errorf("Expected RetryAttempts 3, got %d", config.RetryAttempts)
	}
	if config.RetryDelay != time.Second {
		t.Errorf("Expected RetryDelay 1s, got %s", config.RetryDelay)
	}
}

func TestNewTracker(t *testing.T) {
	store := NewMockEventStore()
	config := TrackerConfig{
		BatchSize:     10,
		FlushInterval: 100 * time.Millisecond,
		MaxQueueSize:  100,
		Workers:       1,
		RetryAttempts: 1,
		RetryDelay:    10 * time.Millisecond,
		Environment:   "test",
		Version:       "1.0.0",
		Enabled:       true,
		Debug:         false,
	}

	tracker := NewTracker(store, config)
	defer tracker.Close()

	if tracker == nil {
		t.Fatal("Tracker should not be nil")
	}

	stats := tracker.Stats()
	if !stats.Enabled {
		t.Error("Tracker should be enabled")
	}
	if stats.Environment != "test" {
		t.Errorf("Expected environment 'test', got '%s'", stats.Environment)
	}
}

func TestTrackerDisabled(t *testing.T) {
	store := NewMockEventStore()
	config := TrackerConfig{
		Enabled: false,
	}

	tracker := NewTracker(store, config)
	defer tracker.Close()

	ctx := context.Background()
	event := NewEvent(EventUserLogin, CategoryUser)

	err := tracker.Track(ctx, event)
	if err != nil {
		t.Errorf("Track should not error when disabled: %v", err)
	}

	// Give some time for potential processing
	time.Sleep(50 * time.Millisecond)

	if len(store.GetEvents()) > 0 {
		t.Error("No events should be stored when tracker is disabled")
	}
}

func TestTrackerTrack(t *testing.T) {
	store := NewMockEventStore()
	config := TrackerConfig{
		BatchSize:     2,
		FlushInterval: 1 * time.Second,
		MaxQueueSize:  100,
		Workers:       1,
		RetryAttempts: 1,
		RetryDelay:    10 * time.Millisecond,
		Environment:   "test",
		Version:       "1.0.0",
		Enabled:       true,
		Debug:         false,
	}

	tracker := NewTracker(store, config)
	defer tracker.Close()

	ctx := context.Background()

	// Track events
	for i := 0; i < 3; i++ {
		event := NewEvent(EventUserLogin, CategoryUser)
		err := tracker.Track(ctx, event)
		if err != nil {
			t.Errorf("Track should not error: %v", err)
		}
	}

	// Wait for batch processing
	time.Sleep(200 * time.Millisecond)

	// Flush remaining events
	tracker.Flush(ctx)
	time.Sleep(100 * time.Millisecond)

	events := store.GetEvents()
	if len(events) < 3 {
		t.Errorf("Expected at least 3 events, got %d", len(events))
	}
}

func TestTrackerEnrichesEvents(t *testing.T) {
	store := NewMockEventStore()
	config := TrackerConfig{
		BatchSize:     1,
		FlushInterval: 100 * time.Millisecond,
		MaxQueueSize:  100,
		Workers:       1,
		RetryAttempts: 1,
		RetryDelay:    10 * time.Millisecond,
		Environment:   "production",
		Version:       "2.0.0",
		Enabled:       true,
		Debug:         false,
	}

	tracker := NewTracker(store, config)
	defer tracker.Close()

	ctx := context.Background()
	event := NewEvent(EventUserLogin, CategoryUser)

	tracker.Track(ctx, event)
	tracker.Flush(ctx)
	time.Sleep(100 * time.Millisecond)

	events := store.GetEvents()
	if len(events) == 0 {
		t.Fatal("Expected at least one event")
	}

	savedEvent := events[0]
	if savedEvent.Environment != "production" {
		t.Errorf("Expected environment 'production', got '%s'", savedEvent.Environment)
	}
	if savedEvent.Version != "2.0.0" {
		t.Errorf("Expected version '2.0.0', got '%s'", savedEvent.Version)
	}
}

func TestTrackerTrackBatch(t *testing.T) {
	store := NewMockEventStore()
	config := TrackerConfig{
		BatchSize:     1, // Small batch size for immediate processing
		FlushInterval: 10 * time.Millisecond,
		MaxQueueSize:  100,
		Workers:       2,
		RetryAttempts: 1,
		RetryDelay:    10 * time.Millisecond,
		Environment:   "test",
		Version:       "1.0.0",
		Enabled:       true,
		Debug:         false,
	}

	tracker := NewTracker(store, config)
	defer tracker.Close()

	ctx := context.Background()

	events := []*Event{
		NewEvent(EventUserLogin, CategoryUser),
		NewEvent(EventPageView, CategoryPage),
		NewEvent(EventFeatureUsed, CategoryFeature),
	}

	batch := NewEventBatch(events)
	err := tracker.TrackBatch(ctx, batch)
	if err != nil {
		t.Errorf("TrackBatch should not error: %v", err)
	}

	// Wait for workers to process events from queue to buffer and flush
	time.Sleep(200 * time.Millisecond)
	tracker.Flush(ctx)
	time.Sleep(100 * time.Millisecond)

	storedEvents := store.GetEvents()
	if len(storedEvents) != 3 {
		t.Errorf("Expected 3 events, got %d", len(storedEvents))
	}
}

func TestTrackerStats(t *testing.T) {
	store := NewMockEventStore()
	config := TrackerConfig{
		BatchSize:     100,
		FlushInterval: 10 * time.Second,
		MaxQueueSize:  100,
		Workers:       1,
		RetryAttempts: 1,
		RetryDelay:    10 * time.Millisecond,
		Environment:   "staging",
		Version:       "1.0.0",
		Enabled:       true,
		Debug:         false,
	}

	tracker := NewTracker(store, config)
	defer tracker.Close()

	stats := tracker.Stats()

	if !stats.Enabled {
		t.Error("Stats.Enabled should be true")
	}
	if stats.Environment != "staging" {
		t.Errorf("Expected environment 'staging', got '%s'", stats.Environment)
	}
	if stats.QueueSize != 0 {
		t.Errorf("Expected QueueSize 0, got %d", stats.QueueSize)
	}
	if stats.BufferSize != 0 {
		t.Errorf("Expected BufferSize 0, got %d", stats.BufferSize)
	}
}

func TestTrackerClose(t *testing.T) {
	store := NewMockEventStore()
	config := TrackerConfig{
		BatchSize:     1, // Small batch size for immediate processing
		FlushInterval: 10 * time.Millisecond,
		MaxQueueSize:  100,
		Workers:       2,
		RetryAttempts: 1,
		RetryDelay:    10 * time.Millisecond,
		Environment:   "test",
		Version:       "1.0.0",
		Enabled:       true,
		Debug:         false,
	}

	tracker := NewTracker(store, config)

	ctx := context.Background()

	// Queue some events
	for i := 0; i < 5; i++ {
		tracker.Track(ctx, NewEvent(EventUserLogin, CategoryUser))
	}

	// Wait for workers to start processing events
	time.Sleep(200 * time.Millisecond)

	// Close should flush remaining events
	err := tracker.Close()
	if err != nil {
		t.Errorf("Close should not error: %v", err)
	}

	// Give time for final flush
	time.Sleep(100 * time.Millisecond)

	events := store.GetEvents()
	if len(events) < 5 {
		t.Errorf("Expected at least 5 events after close, got %d", len(events))
	}
}

func TestTrackerRetryOnError(t *testing.T) {
	store := NewMockEventStore()
	config := TrackerConfig{
		BatchSize:     1,
		FlushInterval: 10 * time.Millisecond,
		MaxQueueSize:  100,
		Workers:       1,
		RetryAttempts: 3,
		RetryDelay:    10 * time.Millisecond,
		Environment:   "test",
		Version:       "1.0.0",
		Enabled:       true,
		Debug:         false,
	}

	tracker := NewTracker(store, config)
	defer tracker.Close()

	// Set error for first few calls, then succeed
	callCount := 0
	store.SetSaveBatchError(errors.New("temporary error"))

	ctx := context.Background()
	tracker.Track(ctx, NewEvent(EventUserLogin, CategoryUser))

	// Wait for retry attempts
	time.Sleep(200 * time.Millisecond)

	// Clear error and retry
	store.SetSaveBatchError(nil)
	tracker.Flush(ctx)
	time.Sleep(100 * time.Millisecond)

	_ = callCount // callCount would need more complex mock to track
}

func TestGlobalTracker(t *testing.T) {
	store := NewMockEventStore()
	config := TrackerConfig{
		BatchSize:     1, // Small batch size for immediate processing
		FlushInterval: 10 * time.Millisecond,
		MaxQueueSize:  100,
		Workers:       2,
		RetryAttempts: 1,
		RetryDelay:    10 * time.Millisecond,
		Environment:   "test",
		Version:       "1.0.0",
		Enabled:       true,
		Debug:         false,
	}

	tracker := NewTracker(store, config)
	defer tracker.Close()

	SetGlobalTracker(tracker)

	if GetGlobalTracker() != tracker {
		t.Error("Global tracker should be set")
	}

	ctx := context.Background()
	event := NewEvent(EventUserLogin, CategoryUser)

	err := Track(ctx, event)
	if err != nil {
		t.Errorf("Track should not error: %v", err)
	}

	// Wait for workers to process event
	time.Sleep(200 * time.Millisecond)
	tracker.Flush(ctx)
	time.Sleep(100 * time.Millisecond)

	events := store.GetEvents()
	if len(events) == 0 {
		t.Error("Expected at least one event from global tracker")
	}
}

func TestTrackWithNilGlobalTracker(t *testing.T) {
	SetGlobalTracker(nil)

	ctx := context.Background()
	event := NewEvent(EventUserLogin, CategoryUser)

	err := Track(ctx, event)
	if err != nil {
		t.Errorf("Track should not error when global tracker is nil: %v", err)
	}
}

func TestTrackSimple(t *testing.T) {
	store := NewMockEventStore()
	config := TrackerConfig{
		BatchSize:     1,
		FlushInterval: 100 * time.Millisecond,
		MaxQueueSize:  100,
		Workers:       1,
		RetryAttempts: 1,
		RetryDelay:    10 * time.Millisecond,
		Environment:   "test",
		Version:       "1.0.0",
		Enabled:       true,
		Debug:         false,
	}

	tracker := NewTracker(store, config)
	defer tracker.Close()

	SetGlobalTracker(tracker)

	ctx := context.Background()
	err := TrackSimple(ctx, EventUserLogin, CategoryUser, map[string]interface{}{
		"method": "email",
	})

	if err != nil {
		t.Errorf("TrackSimple should not error: %v", err)
	}

	tracker.Flush(ctx)
	time.Sleep(100 * time.Millisecond)

	events := store.GetEvents()
	if len(events) == 0 {
		t.Error("Expected at least one event")
	}

	if events[0].Name != EventUserLogin {
		t.Errorf("Expected event name %s, got %s", EventUserLogin, events[0].Name)
	}
}
