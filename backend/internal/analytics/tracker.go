package analytics

import (
	"context"
	"log"
	"sync"
	"time"

	appconfig "github.com/javaknight1/servicepro/backend/config"
)

// TrackerConfig holds configuration for the analytics tracker
type TrackerConfig struct {
	// BatchSize is the number of events to batch before sending
	BatchSize int

	// FlushInterval is how often to flush events
	FlushInterval time.Duration

	// MaxQueueSize is the maximum number of events to queue
	MaxQueueSize int

	// Workers is the number of worker goroutines
	Workers int

	// RetryAttempts is the number of retry attempts on failure
	RetryAttempts int

	// RetryDelay is the delay between retries
	RetryDelay time.Duration

	// Environment is the current environment
	Environment string

	// Version is the application version
	Version string

	// Enabled determines if tracking is enabled
	Enabled bool

	// Debug enables debug logging
	Debug bool
}

// DefaultTrackerConfig returns default tracker configuration
func DefaultTrackerConfig() TrackerConfig {
	return TrackerConfig{
		BatchSize:     100,
		FlushInterval: 10 * time.Second,
		MaxQueueSize:  10000,
		Workers:       2,
		RetryAttempts: 3,
		RetryDelay:    time.Second,
		Environment:   "development",
		Version:       "unknown",
		Enabled:       true,
		Debug:         false,
	}
}

// NewTrackerConfigFromAppConfig creates a TrackerConfig from the centralized app config
func NewTrackerConfigFromAppConfig(appCfg *appconfig.Config) TrackerConfig {
	cfg := DefaultTrackerConfig()
	cfg.Environment = appCfg.Server.Env
	cfg.Enabled = appCfg.Analytics.Enabled
	cfg.Debug = appCfg.Analytics.Debug
	return cfg
}

// Tracker is the main analytics tracker
type Tracker struct {
	config    TrackerConfig
	store     EventStore
	queue     chan *Event
	wg        sync.WaitGroup
	ctx       context.Context
	cancel    context.CancelFunc
	mu        sync.Mutex
	buffer    []*Event
	lastFlush time.Time
}

// NewTracker creates a new analytics tracker
func NewTracker(store EventStore, config ...TrackerConfig) *Tracker {
	cfg := DefaultTrackerConfig()
	if len(config) > 0 {
		cfg = config[0]
	}

	ctx, cancel := context.WithCancel(context.Background())

	t := &Tracker{
		config:    cfg,
		store:     store,
		queue:     make(chan *Event, cfg.MaxQueueSize),
		ctx:       ctx,
		cancel:    cancel,
		buffer:    make([]*Event, 0, cfg.BatchSize),
		lastFlush: time.Now(),
	}

	if cfg.Enabled {
		t.start()
	}

	return t
}

// start starts the tracker workers
func (t *Tracker) start() {
	// Start worker goroutines
	for i := 0; i < t.config.Workers; i++ {
		t.wg.Add(1)
		go t.worker()
	}

	// Start flush ticker
	t.wg.Add(1)
	go t.flusher()

	if t.config.Debug {
		log.Printf("[Analytics] Started with %d workers, batch size %d, flush interval %s",
			t.config.Workers, t.config.BatchSize, t.config.FlushInterval)
	}
}

// worker processes events from the queue
func (t *Tracker) worker() {
	defer t.wg.Done()

	for {
		select {
		case <-t.ctx.Done():
			return
		case event := <-t.queue:
			t.addToBuffer(event)
		}
	}
}

// flusher periodically flushes the buffer
func (t *Tracker) flusher() {
	defer t.wg.Done()

	ticker := time.NewTicker(t.config.FlushInterval)
	defer ticker.Stop()

	for {
		select {
		case <-t.ctx.Done():
			return
		case <-ticker.C:
			if err := t.Flush(t.ctx); err != nil {
				log.Printf("[Analytics] Flush error: %v", err)
			}
		}
	}
}

// addToBuffer adds an event to the buffer and flushes if needed
func (t *Tracker) addToBuffer(event *Event) {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.buffer = append(t.buffer, event)

	if len(t.buffer) >= t.config.BatchSize {
		go t.flushBuffer()
	}
}

// flushBuffer flushes the current buffer
func (t *Tracker) flushBuffer() {
	t.mu.Lock()
	if len(t.buffer) == 0 {
		t.mu.Unlock()
		return
	}

	events := make([]*Event, len(t.buffer))
	copy(events, t.buffer)
	t.buffer = t.buffer[:0]
	t.lastFlush = time.Now()
	t.mu.Unlock()

	if err := t.saveWithRetry(t.ctx, events); err != nil {
		log.Printf("[Analytics] Failed to save %d events: %v", len(events), err)
	} else if t.config.Debug {
		log.Printf("[Analytics] Saved %d events", len(events))
	}
}

// saveWithRetry saves events with retry logic
func (t *Tracker) saveWithRetry(ctx context.Context, events []*Event) error {
	var lastErr error

	for attempt := 0; attempt < t.config.RetryAttempts; attempt++ {
		if err := t.store.SaveBatch(ctx, events); err != nil {
			lastErr = err
			if attempt < t.config.RetryAttempts-1 {
				time.Sleep(t.config.RetryDelay * time.Duration(attempt+1))
			}
			continue
		}
		return nil
	}

	return lastErr
}

// Track tracks a single event
func (t *Tracker) Track(ctx context.Context, event *Event) error {
	if !t.config.Enabled {
		return nil
	}

	// Enrich event with default values
	if event.Environment == "" {
		event.Environment = t.config.Environment
	}
	if event.Version == "" {
		event.Version = t.config.Version
	}

	select {
	case t.queue <- event:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	default:
		// Queue is full, try to flush and retry
		if t.config.Debug {
			log.Printf("[Analytics] Queue full, dropping event: %s", event.Name)
		}
		return nil
	}
}

// TrackBatch tracks a batch of events
func (t *Tracker) TrackBatch(ctx context.Context, batch *EventBatch) error {
	if !t.config.Enabled {
		return nil
	}

	for _, event := range batch.Events {
		if err := t.Track(ctx, event); err != nil {
			return err
		}
	}
	return nil
}

// Flush flushes all pending events
func (t *Tracker) Flush(ctx context.Context) error {
	t.flushBuffer()
	return nil
}

// Close stops the tracker and flushes remaining events
func (t *Tracker) Close() error {
	// First, drain any remaining events from the queue into the buffer
	// before canceling the context
drainLoop:
	for {
		select {
		case event := <-t.queue:
			t.addToBuffer(event)
		default:
			break drainLoop
		}
	}

	// Flush remaining events in the buffer
	t.flushBuffer()

	// Now cancel the context to stop workers
	t.cancel()

	// Wait for workers to finish
	done := make(chan struct{})
	go func() {
		t.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		return nil
	case <-time.After(5 * time.Second):
		return context.DeadlineExceeded
	}
}

// Stats returns tracker statistics
func (t *Tracker) Stats() TrackerStats {
	t.mu.Lock()
	defer t.mu.Unlock()

	return TrackerStats{
		QueueSize:   len(t.queue),
		BufferSize:  len(t.buffer),
		LastFlush:   t.lastFlush,
		Enabled:     t.config.Enabled,
		Environment: t.config.Environment,
	}
}

// TrackerStats holds tracker statistics
type TrackerStats struct {
	QueueSize   int       `json:"queue_size"`
	BufferSize  int       `json:"buffer_size"`
	LastFlush   time.Time `json:"last_flush"`
	Enabled     bool      `json:"enabled"`
	Environment string    `json:"environment"`
}

// Global tracker instance
var (
	globalTracker *Tracker
	trackerOnce   sync.Once
)

// SetGlobalTracker sets the global tracker instance
func SetGlobalTracker(tracker *Tracker) {
	globalTracker = tracker
}

// GetGlobalTracker returns the global tracker instance
func GetGlobalTracker() *Tracker {
	return globalTracker
}

// Track tracks an event using the global tracker
func Track(ctx context.Context, event *Event) error {
	if globalTracker == nil {
		return nil
	}
	return globalTracker.Track(ctx, event)
}

// TrackSimple is a convenience function for tracking simple events
func TrackSimple(ctx context.Context, name EventName, category EventCategory, props map[string]interface{}) error {
	event := NewEvent(name, category).WithProperties(props)
	return Track(ctx, event)
}
