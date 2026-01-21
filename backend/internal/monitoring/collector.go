package monitoring

import (
	"context"
	"database/sql"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
)

// Collector aggregates metrics from various sources
type Collector struct {
	registry    *MetricsRegistry
	db          *sql.DB
	redis       *redis.Client
	interval    time.Duration
	ctx         context.Context
	cancel      context.CancelFunc
	wg          sync.WaitGroup
	subscribers []MetricsSubscriber
	mu          sync.RWMutex
}

// MetricsSubscriber receives periodic metrics updates
type MetricsSubscriber interface {
	OnMetrics(metrics []Metric)
}

// CollectorConfig holds collector configuration
type CollectorConfig struct {
	Interval time.Duration
	DB       *sql.DB
	Redis    *redis.Client
}

// NewCollector creates a new metrics collector
func NewCollector(registry *MetricsRegistry, config CollectorConfig) *Collector {
	if config.Interval == 0 {
		config.Interval = 15 * time.Second
	}

	ctx, cancel := context.WithCancel(context.Background())

	return &Collector{
		registry: registry,
		db:       config.DB,
		redis:    config.Redis,
		interval: config.Interval,
		ctx:      ctx,
		cancel:   cancel,
	}
}

// Subscribe adds a subscriber to receive metrics updates
func (c *Collector) Subscribe(subscriber MetricsSubscriber) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.subscribers = append(c.subscribers, subscriber)
}

// Start starts the collector
func (c *Collector) Start() {
	c.wg.Add(1)
	go c.run()
}

// Stop stops the collector
func (c *Collector) Stop() {
	c.cancel()
	c.wg.Wait()
}

func (c *Collector) run() {
	defer c.wg.Done()

	ticker := time.NewTicker(c.interval)
	defer ticker.Stop()

	// Collect immediately on start
	c.collect()

	for {
		select {
		case <-c.ctx.Done():
			return
		case <-ticker.C:
			c.collect()
		}
	}
}

func (c *Collector) collect() {
	// Collect database metrics
	if c.db != nil {
		c.collectDatabaseMetrics()
	}

	// Collect Redis metrics
	if c.redis != nil {
		c.collectRedisMetrics()
	}

	// Notify subscribers
	metrics := c.registry.GetAll()
	c.mu.RLock()
	subscribers := c.subscribers
	c.mu.RUnlock()

	for _, sub := range subscribers {
		go sub.OnMetrics(metrics)
	}
}

func (c *Collector) collectDatabaseMetrics() {
	stats := c.db.Stats()

	// Connection pool metrics
	c.registry.Gauge("db_connections_open", "Open database connections", nil).Set(float64(stats.OpenConnections))
	c.registry.Gauge("db_connections_in_use", "Database connections in use", nil).Set(float64(stats.InUse))
	c.registry.Gauge("db_connections_idle", "Idle database connections", nil).Set(float64(stats.Idle))
	c.registry.Gauge("db_connections_max_open", "Max open database connections", nil).Set(float64(stats.MaxOpenConnections))

	// Connection wait metrics
	c.registry.Gauge("db_wait_count_total", "Total connection wait count", nil).Set(float64(stats.WaitCount))
	c.registry.Gauge("db_wait_duration_total_ms", "Total connection wait duration", nil).Set(float64(stats.WaitDuration.Milliseconds()))

	// Connection lifecycle metrics
	c.registry.Gauge("db_max_idle_closed_total", "Connections closed due to max idle", nil).Set(float64(stats.MaxIdleClosed))
	c.registry.Gauge("db_max_lifetime_closed_total", "Connections closed due to max lifetime", nil).Set(float64(stats.MaxLifetimeClosed))
}

func (c *Collector) collectRedisMetrics() {
	stats := c.redis.PoolStats()

	// Pool metrics
	c.registry.Gauge("redis_pool_hits_total", "Redis pool hits", nil).Set(float64(stats.Hits))
	c.registry.Gauge("redis_pool_misses_total", "Redis pool misses", nil).Set(float64(stats.Misses))
	c.registry.Gauge("redis_pool_timeouts_total", "Redis pool timeouts", nil).Set(float64(stats.Timeouts))
	c.registry.Gauge("redis_pool_total_conns", "Redis total connections", nil).Set(float64(stats.TotalConns))
	c.registry.Gauge("redis_pool_idle_conns", "Redis idle connections", nil).Set(float64(stats.IdleConns))
	c.registry.Gauge("redis_pool_stale_conns", "Redis stale connections", nil).Set(float64(stats.StaleConns))

	// Calculate hit rate
	total := stats.Hits + stats.Misses
	if total > 0 {
		hitRate := float64(stats.Hits) / float64(total) * 100
		c.registry.Gauge("redis_cache_hit_rate", "Redis cache hit rate percentage", nil).Set(hitRate)
	}
}

// APIMetrics provides pre-configured metrics for API monitoring
type APIMetrics struct {
	RequestsTotal    *Counter
	RequestDuration  *Histogram
	RequestsInFlight *Gauge
	ResponseSize     *Histogram
	ErrorsTotal      *Counter
	registry         *MetricsRegistry
}

// NewAPIMetrics creates API metrics
func NewAPIMetrics(registry *MetricsRegistry) *APIMetrics {
	return &APIMetrics{
		RequestsTotal: registry.Counter(
			"http_requests_total",
			"Total HTTP requests",
			nil,
		),
		RequestDuration: registry.Histogram(
			"http_request_duration_ms",
			"HTTP request duration in milliseconds",
			nil,
			DefaultBuckets(),
		),
		RequestsInFlight: registry.Gauge(
			"http_requests_in_flight",
			"Current number of HTTP requests being processed",
			nil,
		),
		ResponseSize: registry.Histogram(
			"http_response_size_bytes",
			"HTTP response size in bytes",
			nil,
			[]float64{100, 1000, 10000, 100000, 1000000},
		),
		ErrorsTotal: registry.Counter(
			"http_errors_total",
			"Total HTTP errors",
			nil,
		),
		registry: registry,
	}
}

// RecordRequest records an API request
func (m *APIMetrics) RecordRequest(method, path string, statusCode int, duration time.Duration, responseSize int) {
	m.RequestsTotal.Inc()
	m.RequestDuration.Observe(float64(duration.Milliseconds()))
	m.ResponseSize.Observe(float64(responseSize))

	if statusCode >= 400 {
		m.ErrorsTotal.Inc()
	}

	// Record with labels
	labels := map[string]string{
		"method": method,
		"path":   path,
		"status": statusCodeGroup(statusCode),
	}

	m.registry.Counter("http_requests_by_endpoint", "Requests by endpoint", labels).Inc()
	m.registry.Histogram("http_request_duration_by_endpoint_ms", "Duration by endpoint", labels, DefaultBuckets()).Observe(float64(duration.Milliseconds()))
}

// DatabaseMetrics provides pre-configured metrics for database monitoring
type DatabaseMetrics struct {
	QueriesTotal  *Counter
	QueryDuration *Histogram
	QueryErrors   *Counter
	SlowQueries   *Counter
	registry      *MetricsRegistry
	slowThreshold time.Duration
}

// NewDatabaseMetrics creates database metrics
func NewDatabaseMetrics(registry *MetricsRegistry, slowThreshold time.Duration) *DatabaseMetrics {
	if slowThreshold == 0 {
		slowThreshold = 100 * time.Millisecond
	}

	return &DatabaseMetrics{
		QueriesTotal: registry.Counter(
			"db_queries_total",
			"Total database queries",
			nil,
		),
		QueryDuration: registry.Histogram(
			"db_query_duration_ms",
			"Database query duration in milliseconds",
			nil,
			[]float64{1, 5, 10, 25, 50, 100, 250, 500, 1000},
		),
		QueryErrors: registry.Counter(
			"db_query_errors_total",
			"Total database query errors",
			nil,
		),
		SlowQueries: registry.Counter(
			"db_slow_queries_total",
			"Total slow database queries",
			nil,
		),
		registry:      registry,
		slowThreshold: slowThreshold,
	}
}

// RecordQuery records a database query
func (m *DatabaseMetrics) RecordQuery(operation string, table string, duration time.Duration, err error) {
	m.QueriesTotal.Inc()
	m.QueryDuration.Observe(float64(duration.Milliseconds()))

	if err != nil {
		m.QueryErrors.Inc()
	}

	if duration > m.slowThreshold {
		m.SlowQueries.Inc()
	}

	// Record with labels
	labels := map[string]string{
		"operation": operation,
		"table":     table,
	}
	m.registry.Counter("db_queries_by_table", "Queries by table", labels).Inc()
	m.registry.Histogram("db_query_duration_by_table_ms", "Duration by table", labels, nil).Observe(float64(duration.Milliseconds()))
}

// CacheMetrics provides pre-configured metrics for cache monitoring
type CacheMetrics struct {
	HitsTotal    *Counter
	MissesTotal  *Counter
	SetsTotal    *Counter
	DeletesTotal *Counter
	Errors       *Counter
	registry     *MetricsRegistry
}

// NewCacheMetrics creates cache metrics
func NewCacheMetrics(registry *MetricsRegistry) *CacheMetrics {
	return &CacheMetrics{
		HitsTotal: registry.Counter(
			"cache_hits_total",
			"Total cache hits",
			nil,
		),
		MissesTotal: registry.Counter(
			"cache_misses_total",
			"Total cache misses",
			nil,
		),
		SetsTotal: registry.Counter(
			"cache_sets_total",
			"Total cache sets",
			nil,
		),
		DeletesTotal: registry.Counter(
			"cache_deletes_total",
			"Total cache deletes",
			nil,
		),
		Errors: registry.Counter(
			"cache_errors_total",
			"Total cache errors",
			nil,
		),
		registry: registry,
	}
}

// RecordHit records a cache hit
func (m *CacheMetrics) RecordHit(key string) {
	m.HitsTotal.Inc()
}

// RecordMiss records a cache miss
func (m *CacheMetrics) RecordMiss(key string) {
	m.MissesTotal.Inc()
}

// RecordSet records a cache set
func (m *CacheMetrics) RecordSet(key string) {
	m.SetsTotal.Inc()
}

// RecordDelete records a cache delete
func (m *CacheMetrics) RecordDelete(key string) {
	m.DeletesTotal.Inc()
}

// RecordError records a cache error
func (m *CacheMetrics) RecordError(operation string) {
	m.Errors.Inc()
}

// HitRate returns the cache hit rate percentage
func (m *CacheMetrics) HitRate() float64 {
	hits := m.HitsTotal.Value()
	misses := m.MissesTotal.Value()
	total := hits + misses
	if total == 0 {
		return 0
	}
	return float64(hits) / float64(total) * 100
}

// Helper functions

func statusCodeGroup(code int) string {
	switch {
	case code >= 200 && code < 300:
		return "2xx"
	case code >= 300 && code < 400:
		return "3xx"
	case code >= 400 && code < 500:
		return "4xx"
	case code >= 500:
		return "5xx"
	default:
		return "unknown"
	}
}
