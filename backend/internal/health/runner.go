package health

import (
	"context"
	"log"
	"sync"
	"time"
)

// Runner manages periodic health check execution
type Runner struct {
	checker     *Checker
	mu          sync.RWMutex
	running     bool
	stopChan    chan struct{}
	wg          sync.WaitGroup
	intervals   map[string]time.Duration
	defaultInt  time.Duration
	subscribers []Subscriber
}

// Subscriber receives health check updates
type Subscriber interface {
	OnHealthCheck(result CheckResult)
	OnStatusChange(name string, oldStatus, newStatus Status)
}

// RunnerConfig holds configuration for the health check runner
type RunnerConfig struct {
	DefaultInterval time.Duration
	Intervals       map[string]time.Duration
}

// DefaultRunnerConfig returns default runner configuration
func DefaultRunnerConfig() RunnerConfig {
	return RunnerConfig{
		DefaultInterval: 30 * time.Second,
		Intervals:       make(map[string]time.Duration),
	}
}

// NewRunner creates a new health check runner
func NewRunner(checker *Checker, config ...RunnerConfig) *Runner {
	cfg := DefaultRunnerConfig()
	if len(config) > 0 {
		cfg = config[0]
	}

	return &Runner{
		checker:    checker,
		stopChan:   make(chan struct{}),
		intervals:  cfg.Intervals,
		defaultInt: cfg.DefaultInterval,
	}
}

// Subscribe adds a subscriber to receive health updates
func (r *Runner) Subscribe(subscriber Subscriber) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.subscribers = append(r.subscribers, subscriber)
}

// Start starts the periodic health check runner
func (r *Runner) Start() {
	r.mu.Lock()
	if r.running {
		r.mu.Unlock()
		return
	}
	r.running = true
	r.stopChan = make(chan struct{})
	r.mu.Unlock()

	// Register status change callback
	r.checker.OnStatusChange(func(name string, oldStatus, newStatus Status) {
		r.mu.RLock()
		subscribers := r.subscribers
		r.mu.RUnlock()

		for _, sub := range subscribers {
			go sub.OnStatusChange(name, oldStatus, newStatus)
		}
	})

	// Get all registered checks
	r.checker.mu.RLock()
	checks := make(map[string]CheckConfig)
	for name, config := range r.checker.checks {
		checks[name] = config
	}
	r.checker.mu.RUnlock()

	// Start a goroutine for each check
	for name, config := range checks {
		interval := r.getInterval(name, config.Interval)
		r.wg.Add(1)
		go r.runCheckLoop(name, interval)
	}

	log.Printf("[Health] Runner started with %d checks", len(checks))
}

// Stop stops the health check runner
func (r *Runner) Stop() {
	r.mu.Lock()
	if !r.running {
		r.mu.Unlock()
		return
	}
	r.running = false
	close(r.stopChan)
	r.mu.Unlock()

	r.wg.Wait()
	log.Println("[Health] Runner stopped")
}

// IsRunning returns true if the runner is active
func (r *Runner) IsRunning() bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.running
}

// runCheckLoop runs a single health check on an interval
func (r *Runner) runCheckLoop(name string, interval time.Duration) {
	defer r.wg.Done()

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	// Run immediately on start
	r.executeCheck(name)

	for {
		select {
		case <-r.stopChan:
			return
		case <-ticker.C:
			r.executeCheck(name)
		}
	}
}

// executeCheck runs a check and notifies subscribers
func (r *Runner) executeCheck(name string) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	result := r.checker.RunCheck(ctx, name)

	r.mu.RLock()
	subscribers := r.subscribers
	r.mu.RUnlock()

	for _, sub := range subscribers {
		go sub.OnHealthCheck(result)
	}
}

// getInterval returns the interval for a specific check
func (r *Runner) getInterval(name string, defaultInterval time.Duration) time.Duration {
	if interval, ok := r.intervals[name]; ok {
		return interval
	}
	if defaultInterval > 0 {
		return defaultInterval
	}
	return r.defaultInt
}

// SetInterval sets the interval for a specific check
func (r *Runner) SetInterval(name string, interval time.Duration) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.intervals[name] = interval
}

// UptimeTracker tracks service uptime and availability
type UptimeTracker struct {
	mu            sync.RWMutex
	startTime     time.Time
	checkResults  map[string][]CheckResult
	maxHistory    int
	totalChecks   map[string]int64
	failedChecks  map[string]int64
	lastStatus    map[string]Status
	statusChanges []StatusChangeEvent
}

// StatusChangeEvent represents a status change
type StatusChangeEvent struct {
	Name      string    `json:"name"`
	OldStatus Status    `json:"old_status"`
	NewStatus Status    `json:"new_status"`
	Timestamp time.Time `json:"timestamp"`
}

// NewUptimeTracker creates a new uptime tracker
func NewUptimeTracker(maxHistory int) *UptimeTracker {
	if maxHistory <= 0 {
		maxHistory = 100
	}

	return &UptimeTracker{
		startTime:     time.Now(),
		checkResults:  make(map[string][]CheckResult),
		maxHistory:    maxHistory,
		totalChecks:   make(map[string]int64),
		failedChecks:  make(map[string]int64),
		lastStatus:    make(map[string]Status),
		statusChanges: make([]StatusChangeEvent, 0),
	}
}

// OnHealthCheck implements Subscriber interface
func (t *UptimeTracker) OnHealthCheck(result CheckResult) {
	t.mu.Lock()
	defer t.mu.Unlock()

	// Add to history
	history := t.checkResults[result.Name]
	history = append(history, result)
	if len(history) > t.maxHistory {
		history = history[1:]
	}
	t.checkResults[result.Name] = history

	// Update counters
	t.totalChecks[result.Name]++
	if result.Status == StatusUnhealthy {
		t.failedChecks[result.Name]++
	}

	t.lastStatus[result.Name] = result.Status
}

// OnStatusChange implements Subscriber interface
func (t *UptimeTracker) OnStatusChange(name string, oldStatus, newStatus Status) {
	t.mu.Lock()
	defer t.mu.Unlock()

	event := StatusChangeEvent{
		Name:      name,
		OldStatus: oldStatus,
		NewStatus: newStatus,
		Timestamp: time.Now(),
	}

	t.statusChanges = append(t.statusChanges, event)

	// Keep limited history
	if len(t.statusChanges) > t.maxHistory*10 {
		t.statusChanges = t.statusChanges[1:]
	}
}

// GetUptime returns the service uptime
func (t *UptimeTracker) GetUptime() time.Duration {
	return time.Since(t.startTime)
}

// GetAvailability returns the availability percentage for a check
func (t *UptimeTracker) GetAvailability(name string) float64 {
	t.mu.RLock()
	defer t.mu.RUnlock()

	total := t.totalChecks[name]
	if total == 0 {
		return 100.0
	}

	failed := t.failedChecks[name]
	return float64(total-failed) / float64(total) * 100
}

// GetAllAvailability returns availability for all checks
func (t *UptimeTracker) GetAllAvailability() map[string]float64 {
	t.mu.RLock()
	defer t.mu.RUnlock()

	availability := make(map[string]float64)
	for name, total := range t.totalChecks {
		if total == 0 {
			availability[name] = 100.0
		} else {
			failed := t.failedChecks[name]
			availability[name] = float64(total-failed) / float64(total) * 100
		}
	}
	return availability
}

// GetHistory returns the check history for a component
func (t *UptimeTracker) GetHistory(name string) []CheckResult {
	t.mu.RLock()
	defer t.mu.RUnlock()

	history := t.checkResults[name]
	if history == nil {
		return []CheckResult{}
	}

	// Return a copy
	result := make([]CheckResult, len(history))
	copy(result, history)
	return result
}

// GetStatusChanges returns recent status changes
func (t *UptimeTracker) GetStatusChanges(limit int) []StatusChangeEvent {
	t.mu.RLock()
	defer t.mu.RUnlock()

	if limit <= 0 || limit > len(t.statusChanges) {
		limit = len(t.statusChanges)
	}

	// Return most recent
	start := len(t.statusChanges) - limit
	if start < 0 {
		start = 0
	}

	result := make([]StatusChangeEvent, limit)
	copy(result, t.statusChanges[start:])
	return result
}

// GetStats returns uptime statistics
func (t *UptimeTracker) GetStats() UptimeStats {
	t.mu.RLock()
	defer t.mu.RUnlock()

	stats := UptimeStats{
		Uptime:        time.Since(t.startTime),
		StartTime:     t.startTime,
		Availability:  make(map[string]float64),
		TotalChecks:   make(map[string]int64),
		FailedChecks:  make(map[string]int64),
		CurrentStatus: make(map[string]Status),
	}

	for name, total := range t.totalChecks {
		stats.TotalChecks[name] = total
		stats.FailedChecks[name] = t.failedChecks[name]
		if total > 0 {
			stats.Availability[name] = float64(total-t.failedChecks[name]) / float64(total) * 100
		} else {
			stats.Availability[name] = 100.0
		}
	}

	for name, status := range t.lastStatus {
		stats.CurrentStatus[name] = status
	}

	return stats
}

// UptimeStats holds uptime statistics
type UptimeStats struct {
	Uptime        time.Duration      `json:"uptime_seconds"`
	StartTime     time.Time          `json:"start_time"`
	Availability  map[string]float64 `json:"availability_pct"`
	TotalChecks   map[string]int64   `json:"total_checks"`
	FailedChecks  map[string]int64   `json:"failed_checks"`
	CurrentStatus map[string]Status  `json:"current_status"`
}
