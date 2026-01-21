package queue

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"sync"
	"sync/atomic"
	"time"

	"github.com/redis/go-redis/v9"
)

// ============================================
// Queue Metrics
// ============================================

// QueueMetrics collects and exposes queue metrics
type QueueMetrics struct {
	queueName string
	mu        sync.RWMutex

	// Counters
	enqueued        atomic.Int64
	dequeued        atomic.Int64
	completed       atomic.Int64
	failed          atomic.Int64
	retried         atomic.Int64
	dead            atomic.Int64
	expired         atomic.Int64
	recovered       atomic.Int64
	enqueueRejected atomic.Int64

	// Gauges
	queueDepth      atomic.Int64
	processingCount atomic.Int64
	scheduledCount  atomic.Int64
	deadCount       atomic.Int64

	// Histograms (sampled)
	enqueueTimes    []time.Duration
	dequeueTimes    []time.Duration
	processingTimes []time.Duration
	maxSamples      int

	// Rate tracking
	enqueueRate    *rateCounter
	dequeueRate    *rateCounter
	completionRate *rateCounter

	// By priority
	enqueuedByPriority map[string]int64
}

// NewQueueMetrics creates new queue metrics
func NewQueueMetrics(queueName string) *QueueMetrics {
	return &QueueMetrics{
		queueName:          queueName,
		enqueueTimes:       make([]time.Duration, 0, 1000),
		dequeueTimes:       make([]time.Duration, 0, 1000),
		processingTimes:    make([]time.Duration, 0, 1000),
		maxSamples:         1000,
		enqueueRate:        newRateCounter(),
		dequeueRate:        newRateCounter(),
		completionRate:     newRateCounter(),
		enqueuedByPriority: make(map[string]int64),
	}
}

// RecordEnqueue records an enqueue operation
func (m *QueueMetrics) RecordEnqueue(priority string) {
	m.enqueued.Add(1)
	m.enqueueRate.record()

	m.mu.Lock()
	m.enqueuedByPriority[priority]++
	m.mu.Unlock()
}

// RecordEnqueueRejected records a rejected enqueue
func (m *QueueMetrics) RecordEnqueueRejected(reason string) {
	m.enqueueRejected.Add(1)
}

// RecordDequeue records a dequeue operation
func (m *QueueMetrics) RecordDequeue() {
	m.dequeued.Add(1)
	m.dequeueRate.record()
}

// RecordComplete records a completed job
func (m *QueueMetrics) RecordComplete(duration time.Duration) {
	m.completed.Add(1)
	m.completionRate.record()

	m.mu.Lock()
	if len(m.processingTimes) >= m.maxSamples {
		m.processingTimes = m.processingTimes[1:]
	}
	m.processingTimes = append(m.processingTimes, duration)
	m.mu.Unlock()
}

// RecordFailed records a failed job
func (m *QueueMetrics) RecordFailed() {
	m.failed.Add(1)
}

// RecordRetry records a retry
func (m *QueueMetrics) RecordRetry(attempt int) {
	m.retried.Add(1)
}

// RecordDead records a dead job
func (m *QueueMetrics) RecordDead(reason string) {
	m.dead.Add(1)
}

// RecordJobExpired records an expired job
func (m *QueueMetrics) RecordJobExpired() {
	m.expired.Add(1)
}

// RecordRecovered records recovered jobs
func (m *QueueMetrics) RecordRecovered(count int) {
	m.recovered.Add(int64(count))
}

// GetSnapshot returns a snapshot of current metrics
func (m *QueueMetrics) GetSnapshot() MetricsSnapshot {
	m.mu.RLock()
	defer m.mu.RUnlock()

	avgProcessingTime := time.Duration(0)
	if len(m.processingTimes) > 0 {
		var total time.Duration
		for _, t := range m.processingTimes {
			total += t
		}
		avgProcessingTime = total / time.Duration(len(m.processingTimes))
	}

	p95ProcessingTime := time.Duration(0)
	if len(m.processingTimes) > 0 {
		sorted := make([]time.Duration, len(m.processingTimes))
		copy(sorted, m.processingTimes)
		// Simple p95 - in production use a proper percentile library
		idx := int(float64(len(sorted)) * 0.95)
		if idx >= len(sorted) {
			idx = len(sorted) - 1
		}
		p95ProcessingTime = sorted[idx]
	}

	byPriority := make(map[string]int64)
	for k, v := range m.enqueuedByPriority {
		byPriority[k] = v
	}

	return MetricsSnapshot{
		QueueName: m.queueName,
		Timestamp: time.Now(),

		// Counters
		Enqueued:        m.enqueued.Load(),
		Dequeued:        m.dequeued.Load(),
		Completed:       m.completed.Load(),
		Failed:          m.failed.Load(),
		Retried:         m.retried.Load(),
		Dead:            m.dead.Load(),
		Expired:         m.expired.Load(),
		Recovered:       m.recovered.Load(),
		EnqueueRejected: m.enqueueRejected.Load(),

		// Gauges
		QueueDepth:      m.queueDepth.Load(),
		ProcessingCount: m.processingCount.Load(),
		ScheduledCount:  m.scheduledCount.Load(),
		DeadCount:       m.deadCount.Load(),

		// Rates (per second)
		EnqueueRate:    m.enqueueRate.rate(),
		DequeueRate:    m.dequeueRate.rate(),
		CompletionRate: m.completionRate.rate(),

		// Processing times
		AvgProcessingTime: avgProcessingTime,
		P95ProcessingTime: p95ProcessingTime,

		// By priority
		EnqueuedByPriority: byPriority,
	}
}

// MetricsSnapshot holds a point-in-time snapshot of metrics
type MetricsSnapshot struct {
	QueueName string    `json:"queue_name"`
	Timestamp time.Time `json:"timestamp"`

	// Counters
	Enqueued        int64 `json:"enqueued"`
	Dequeued        int64 `json:"dequeued"`
	Completed       int64 `json:"completed"`
	Failed          int64 `json:"failed"`
	Retried         int64 `json:"retried"`
	Dead            int64 `json:"dead"`
	Expired         int64 `json:"expired"`
	Recovered       int64 `json:"recovered"`
	EnqueueRejected int64 `json:"enqueue_rejected"`

	// Gauges
	QueueDepth      int64 `json:"queue_depth"`
	ProcessingCount int64 `json:"processing_count"`
	ScheduledCount  int64 `json:"scheduled_count"`
	DeadCount       int64 `json:"dead_count"`

	// Rates
	EnqueueRate    float64 `json:"enqueue_rate"`
	DequeueRate    float64 `json:"dequeue_rate"`
	CompletionRate float64 `json:"completion_rate"`

	// Processing times
	AvgProcessingTime time.Duration `json:"avg_processing_time"`
	P95ProcessingTime time.Duration `json:"p95_processing_time"`

	// By priority
	EnqueuedByPriority map[string]int64 `json:"enqueued_by_priority"`
}

// rateCounter tracks rate per second
type rateCounter struct {
	count       int64
	lastReset   time.Time
	currentRate float64
	mu          sync.Mutex
}

func newRateCounter() *rateCounter {
	return &rateCounter{
		lastReset: time.Now(),
	}
}

func (r *rateCounter) record() {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.count++

	// Update rate every second
	elapsed := time.Since(r.lastReset)
	if elapsed >= time.Second {
		r.currentRate = float64(r.count) / elapsed.Seconds()
		r.count = 0
		r.lastReset = time.Now()
	}
}

func (r *rateCounter) rate() float64 {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.currentRate
}

// ============================================
// Monitoring Service
// ============================================

// MonitoringService provides monitoring and alerting for the queue
type MonitoringService struct {
	queue  *EmailQueue
	pool   *WorkerPool
	client *redis.Client
	config *MonitoringConfig
	alerts *AlertManager

	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup

	// Metrics history
	history    []MetricsSnapshot
	historyMu  sync.RWMutex
	maxHistory int
}

// MonitoringConfig holds monitoring configuration
type MonitoringConfig struct {
	// Collection
	CollectionInterval time.Duration `json:"collection_interval"`
	HistoryRetention   int           `json:"history_retention"`

	// Alerting thresholds
	QueueDepthWarning      int64         `json:"queue_depth_warning"`
	QueueDepthCritical     int64         `json:"queue_depth_critical"`
	FailureRateWarning     float64       `json:"failure_rate_warning"`
	FailureRateCritical    float64       `json:"failure_rate_critical"`
	ProcessingTimeWarning  time.Duration `json:"processing_time_warning"`
	ProcessingTimeCritical time.Duration `json:"processing_time_critical"`
	DeadQueueWarning       int64         `json:"dead_queue_warning"`

	// Export
	ExportEnabled  bool          `json:"export_enabled"`
	ExportInterval time.Duration `json:"export_interval"`
	ExportKey      string        `json:"export_key"`
}

// DefaultMonitoringConfig returns default configuration
func DefaultMonitoringConfig() *MonitoringConfig {
	return &MonitoringConfig{
		CollectionInterval:     10 * time.Second,
		HistoryRetention:       1000,
		QueueDepthWarning:      1000,
		QueueDepthCritical:     10000,
		FailureRateWarning:     0.05,
		FailureRateCritical:    0.10,
		ProcessingTimeWarning:  10 * time.Second,
		ProcessingTimeCritical: 30 * time.Second,
		DeadQueueWarning:       100,
		ExportEnabled:          true,
		ExportInterval:         time.Minute,
		ExportKey:              "eq:metrics:history",
	}
}

// NewMonitoringService creates a new monitoring service
func NewMonitoringService(queue *EmailQueue, pool *WorkerPool, config *MonitoringConfig) *MonitoringService {
	if config == nil {
		config = DefaultMonitoringConfig()
	}

	ctx, cancel := context.WithCancel(context.Background())

	return &MonitoringService{
		queue:      queue,
		pool:       pool,
		client:     queue.client,
		config:     config,
		alerts:     NewAlertManager(),
		ctx:        ctx,
		cancel:     cancel,
		history:    make([]MetricsSnapshot, 0, config.HistoryRetention),
		maxHistory: config.HistoryRetention,
	}
}

// Start starts the monitoring service
func (ms *MonitoringService) Start() {
	ms.wg.Add(1)
	go ms.collectMetrics()

	if ms.config.ExportEnabled {
		ms.wg.Add(1)
		go ms.exportMetrics()
	}

	ms.wg.Add(1)
	go ms.checkAlerts()
}

// Stop stops the monitoring service
func (ms *MonitoringService) Stop() {
	ms.cancel()
	ms.wg.Wait()
}

// collectMetrics periodically collects metrics
func (ms *MonitoringService) collectMetrics() {
	defer ms.wg.Done()

	ticker := time.NewTicker(ms.config.CollectionInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ms.ctx.Done():
			return
		case <-ticker.C:
			ms.collect()
		}
	}
}

// collect collects current metrics
func (ms *MonitoringService) collect() {
	ctx, cancel := context.WithTimeout(ms.ctx, 5*time.Second)
	defer cancel()

	// Get queue stats
	stats, err := ms.queue.GetStats(ctx)
	if err != nil {
		log.Printf("Error collecting queue stats: %v", err)
		return
	}

	// Get metrics snapshot
	snapshot := ms.queue.metrics.GetSnapshot()

	// Update gauges
	ms.queue.metrics.queueDepth.Store(stats.Pending)
	ms.queue.metrics.processingCount.Store(stats.Processing)
	ms.queue.metrics.scheduledCount.Store(stats.Scheduled)
	ms.queue.metrics.deadCount.Store(stats.Dead)

	// Add to history
	ms.historyMu.Lock()
	if len(ms.history) >= ms.maxHistory {
		ms.history = ms.history[1:]
	}
	ms.history = append(ms.history, snapshot)
	ms.historyMu.Unlock()
}

// exportMetrics exports metrics to Redis
func (ms *MonitoringService) exportMetrics() {
	defer ms.wg.Done()

	ticker := time.NewTicker(ms.config.ExportInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ms.ctx.Done():
			return
		case <-ticker.C:
			ms.export()
		}
	}
}

// export exports metrics to Redis
func (ms *MonitoringService) export() {
	ms.historyMu.RLock()
	if len(ms.history) == 0 {
		ms.historyMu.RUnlock()
		return
	}
	latest := ms.history[len(ms.history)-1]
	ms.historyMu.RUnlock()

	data, err := json.Marshal(latest)
	if err != nil {
		return
	}

	ctx, cancel := context.WithTimeout(ms.ctx, 5*time.Second)
	defer cancel()

	// Store latest metrics
	ms.client.Set(ctx, ms.config.ExportKey+":latest", data, time.Hour)

	// Store in time series (list)
	ms.client.LPush(ctx, ms.config.ExportKey+":series", data)
	ms.client.LTrim(ctx, ms.config.ExportKey+":series", 0, int64(ms.maxHistory-1))
}

// checkAlerts checks for alert conditions
func (ms *MonitoringService) checkAlerts() {
	defer ms.wg.Done()

	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ms.ctx.Done():
			return
		case <-ticker.C:
			ms.evaluateAlerts()
		}
	}
}

// evaluateAlerts evaluates alert conditions
func (ms *MonitoringService) evaluateAlerts() {
	ms.historyMu.RLock()
	if len(ms.history) == 0 {
		ms.historyMu.RUnlock()
		return
	}
	latest := ms.history[len(ms.history)-1]
	ms.historyMu.RUnlock()

	// Queue depth alerts
	if latest.QueueDepth >= ms.config.QueueDepthCritical {
		ms.alerts.Fire(Alert{
			Name:     "queue_depth_critical",
			Severity: SeverityCritical,
			Message:  fmt.Sprintf("Queue depth is critically high: %d", latest.QueueDepth),
			Value:    float64(latest.QueueDepth),
		})
	} else if latest.QueueDepth >= ms.config.QueueDepthWarning {
		ms.alerts.Fire(Alert{
			Name:     "queue_depth_warning",
			Severity: SeverityWarning,
			Message:  fmt.Sprintf("Queue depth is high: %d", latest.QueueDepth),
			Value:    float64(latest.QueueDepth),
		})
	} else {
		ms.alerts.Resolve("queue_depth_critical")
		ms.alerts.Resolve("queue_depth_warning")
	}

	// Failure rate alerts
	if latest.Completed > 0 {
		failureRate := float64(latest.Failed) / float64(latest.Completed+latest.Failed)

		if failureRate >= ms.config.FailureRateCritical {
			ms.alerts.Fire(Alert{
				Name:     "failure_rate_critical",
				Severity: SeverityCritical,
				Message:  fmt.Sprintf("Failure rate is critically high: %.2f%%", failureRate*100),
				Value:    failureRate,
			})
		} else if failureRate >= ms.config.FailureRateWarning {
			ms.alerts.Fire(Alert{
				Name:     "failure_rate_warning",
				Severity: SeverityWarning,
				Message:  fmt.Sprintf("Failure rate is high: %.2f%%", failureRate*100),
				Value:    failureRate,
			})
		} else {
			ms.alerts.Resolve("failure_rate_critical")
			ms.alerts.Resolve("failure_rate_warning")
		}
	}

	// Processing time alerts
	if latest.P95ProcessingTime >= ms.config.ProcessingTimeCritical {
		ms.alerts.Fire(Alert{
			Name:     "processing_time_critical",
			Severity: SeverityCritical,
			Message:  fmt.Sprintf("P95 processing time is critically high: %s", latest.P95ProcessingTime),
			Value:    float64(latest.P95ProcessingTime.Milliseconds()),
		})
	} else if latest.P95ProcessingTime >= ms.config.ProcessingTimeWarning {
		ms.alerts.Fire(Alert{
			Name:     "processing_time_warning",
			Severity: SeverityWarning,
			Message:  fmt.Sprintf("P95 processing time is high: %s", latest.P95ProcessingTime),
			Value:    float64(latest.P95ProcessingTime.Milliseconds()),
		})
	} else {
		ms.alerts.Resolve("processing_time_critical")
		ms.alerts.Resolve("processing_time_warning")
	}

	// Dead queue alerts
	if latest.DeadCount >= ms.config.DeadQueueWarning {
		ms.alerts.Fire(Alert{
			Name:     "dead_queue_warning",
			Severity: SeverityWarning,
			Message:  fmt.Sprintf("Dead queue is growing: %d jobs", latest.DeadCount),
			Value:    float64(latest.DeadCount),
		})
	} else {
		ms.alerts.Resolve("dead_queue_warning")
	}
}

// GetMetrics returns the latest metrics
func (ms *MonitoringService) GetMetrics() MetricsSnapshot {
	ms.historyMu.RLock()
	defer ms.historyMu.RUnlock()

	if len(ms.history) == 0 {
		return MetricsSnapshot{}
	}
	return ms.history[len(ms.history)-1]
}

// GetHistory returns metrics history
func (ms *MonitoringService) GetHistory(count int) []MetricsSnapshot {
	ms.historyMu.RLock()
	defer ms.historyMu.RUnlock()

	if count > len(ms.history) {
		count = len(ms.history)
	}

	result := make([]MetricsSnapshot, count)
	copy(result, ms.history[len(ms.history)-count:])
	return result
}

// GetAlerts returns active alerts
func (ms *MonitoringService) GetAlerts() []Alert {
	return ms.alerts.GetActive()
}

// ============================================
// Alert Manager
// ============================================

// AlertSeverity represents alert severity
type AlertSeverity string

const (
	SeverityInfo     AlertSeverity = "info"
	SeverityWarning  AlertSeverity = "warning"
	SeverityCritical AlertSeverity = "critical"
)

// Alert represents an alert
type Alert struct {
	Name       string        `json:"name"`
	Severity   AlertSeverity `json:"severity"`
	Message    string        `json:"message"`
	Value      float64       `json:"value"`
	FiredAt    time.Time     `json:"fired_at"`
	ResolvedAt *time.Time    `json:"resolved_at,omitempty"`
}

// AlertManager manages alerts
type AlertManager struct {
	mu       sync.RWMutex
	alerts   map[string]*Alert
	handlers []AlertHandler
}

// AlertHandler handles alert notifications
type AlertHandler func(Alert)

// NewAlertManager creates a new alert manager
func NewAlertManager() *AlertManager {
	return &AlertManager{
		alerts: make(map[string]*Alert),
	}
}

// AddHandler adds an alert handler
func (am *AlertManager) AddHandler(handler AlertHandler) {
	am.mu.Lock()
	defer am.mu.Unlock()
	am.handlers = append(am.handlers, handler)
}

// Fire fires an alert
func (am *AlertManager) Fire(alert Alert) {
	am.mu.Lock()
	defer am.mu.Unlock()

	existing, exists := am.alerts[alert.Name]
	if exists && existing.ResolvedAt == nil {
		// Already active, update value
		existing.Value = alert.Value
		existing.Message = alert.Message
		return
	}

	alert.FiredAt = time.Now()
	am.alerts[alert.Name] = &alert

	// Notify handlers
	for _, handler := range am.handlers {
		go handler(alert)
	}
}

// Resolve resolves an alert
func (am *AlertManager) Resolve(name string) {
	am.mu.Lock()
	defer am.mu.Unlock()

	alert, exists := am.alerts[name]
	if !exists || alert.ResolvedAt != nil {
		return
	}

	now := time.Now()
	alert.ResolvedAt = &now
}

// GetActive returns active alerts
func (am *AlertManager) GetActive() []Alert {
	am.mu.RLock()
	defer am.mu.RUnlock()

	active := make([]Alert, 0)
	for _, alert := range am.alerts {
		if alert.ResolvedAt == nil {
			active = append(active, *alert)
		}
	}
	return active
}

// GetAll returns all alerts
func (am *AlertManager) GetAll() []Alert {
	am.mu.RLock()
	defer am.mu.RUnlock()

	all := make([]Alert, 0, len(am.alerts))
	for _, alert := range am.alerts {
		all = append(all, *alert)
	}
	return all
}

// ============================================
// Health Check
// ============================================

// HealthStatus represents health status
type HealthStatus string

const (
	HealthStatusHealthy   HealthStatus = "healthy"
	HealthStatusDegraded  HealthStatus = "degraded"
	HealthStatusUnhealthy HealthStatus = "unhealthy"
)

// HealthCheck holds health check results
type HealthCheck struct {
	Status     HealthStatus               `json:"status"`
	Timestamp  time.Time                  `json:"timestamp"`
	Components map[string]ComponentHealth `json:"components"`
	Alerts     []Alert                    `json:"alerts"`
}

// ComponentHealth holds component health
type ComponentHealth struct {
	Status  HealthStatus  `json:"status"`
	Message string        `json:"message,omitempty"`
	Latency time.Duration `json:"latency,omitempty"`
}

// CheckHealth performs a health check
func (ms *MonitoringService) CheckHealth(ctx context.Context) HealthCheck {
	health := HealthCheck{
		Status:     HealthStatusHealthy,
		Timestamp:  time.Now(),
		Components: make(map[string]ComponentHealth),
	}

	// Check Redis
	start := time.Now()
	if err := ms.client.Ping(ctx).Err(); err != nil {
		health.Components["redis"] = ComponentHealth{
			Status:  HealthStatusUnhealthy,
			Message: err.Error(),
		}
		health.Status = HealthStatusUnhealthy
	} else {
		health.Components["redis"] = ComponentHealth{
			Status:  HealthStatusHealthy,
			Latency: time.Since(start),
		}
	}

	// Check queue
	stats, err := ms.queue.GetStats(ctx)
	if err != nil {
		health.Components["queue"] = ComponentHealth{
			Status:  HealthStatusUnhealthy,
			Message: err.Error(),
		}
		health.Status = HealthStatusUnhealthy
	} else {
		queueStatus := HealthStatusHealthy
		message := ""

		if stats.Pending > ms.config.QueueDepthCritical {
			queueStatus = HealthStatusUnhealthy
			message = "Queue depth critical"
		} else if stats.Pending > ms.config.QueueDepthWarning {
			queueStatus = HealthStatusDegraded
			message = "Queue depth high"
		}

		health.Components["queue"] = ComponentHealth{
			Status:  queueStatus,
			Message: message,
		}

		if queueStatus == HealthStatusUnhealthy {
			health.Status = HealthStatusUnhealthy
		} else if queueStatus == HealthStatusDegraded && health.Status == HealthStatusHealthy {
			health.Status = HealthStatusDegraded
		}
	}

	// Check workers
	if ms.pool != nil {
		workerCount := ms.pool.WorkerCount()
		workerStatus := HealthStatusHealthy

		if workerCount == 0 {
			workerStatus = HealthStatusUnhealthy
		}

		health.Components["workers"] = ComponentHealth{
			Status:  workerStatus,
			Message: fmt.Sprintf("%d active workers", workerCount),
		}

		if workerStatus == HealthStatusUnhealthy {
			health.Status = HealthStatusUnhealthy
		}
	}

	// Include active alerts
	health.Alerts = ms.GetAlerts()

	return health
}

// ============================================
// Utility Functions
// ============================================

// generateJobID generates a unique job ID
func generateJobID() string {
	return fmt.Sprintf("job_%d_%d", time.Now().UnixNano(), rand.Int63n(10000))
}

// generateWorkerID generates a unique worker ID
func generateWorkerID() string {
	return fmt.Sprintf("worker_%d_%d", time.Now().UnixNano(), rand.Int63n(10000))
}

// Prometheus-compatible metrics export
func (m *QueueMetrics) PrometheusMetrics() string {
	snapshot := m.GetSnapshot()

	return fmt.Sprintf(`
# HELP email_queue_enqueued_total Total emails enqueued
# TYPE email_queue_enqueued_total counter
email_queue_enqueued_total{queue="%s"} %d

# HELP email_queue_dequeued_total Total emails dequeued
# TYPE email_queue_dequeued_total counter
email_queue_dequeued_total{queue="%s"} %d

# HELP email_queue_completed_total Total emails completed
# TYPE email_queue_completed_total counter
email_queue_completed_total{queue="%s"} %d

# HELP email_queue_failed_total Total emails failed
# TYPE email_queue_failed_total counter
email_queue_failed_total{queue="%s"} %d

# HELP email_queue_depth Current queue depth
# TYPE email_queue_depth gauge
email_queue_depth{queue="%s"} %d

# HELP email_queue_processing_count Current processing count
# TYPE email_queue_processing_count gauge
email_queue_processing_count{queue="%s"} %d

# HELP email_queue_dead_count Dead queue count
# TYPE email_queue_dead_count gauge
email_queue_dead_count{queue="%s"} %d

# HELP email_queue_processing_duration_seconds Average processing duration
# TYPE email_queue_processing_duration_seconds gauge
email_queue_processing_duration_seconds{queue="%s"} %f
`,
		snapshot.QueueName, snapshot.Enqueued,
		snapshot.QueueName, snapshot.Dequeued,
		snapshot.QueueName, snapshot.Completed,
		snapshot.QueueName, snapshot.Failed,
		snapshot.QueueName, snapshot.QueueDepth,
		snapshot.QueueName, snapshot.ProcessingCount,
		snapshot.QueueName, snapshot.DeadCount,
		snapshot.QueueName, snapshot.AvgProcessingTime.Seconds(),
	)
}
