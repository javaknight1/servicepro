package monitoring

import (
	"context"
	"runtime"
	"sync"
	"sync/atomic"
	"time"
)

// MetricType represents the type of metric
type MetricType string

const (
	MetricTypeCounter   MetricType = "counter"
	MetricTypeGauge     MetricType = "gauge"
	MetricTypeHistogram MetricType = "histogram"
	MetricTypeSummary   MetricType = "summary"
)

// Metric represents a single metric
type Metric struct {
	Name        string            `json:"name"`
	Type        MetricType        `json:"type"`
	Value       float64           `json:"value"`
	Labels      map[string]string `json:"labels,omitempty"`
	Timestamp   time.Time         `json:"timestamp"`
	Unit        string            `json:"unit,omitempty"`
	Description string            `json:"description,omitempty"`
}

// Counter is a monotonically increasing counter
type Counter struct {
	name        string
	description string
	labels      map[string]string
	value       int64
}

// NewCounter creates a new counter
func NewCounter(name, description string, labels map[string]string) *Counter {
	return &Counter{
		name:        name,
		description: description,
		labels:      labels,
	}
}

// Inc increments the counter by 1
func (c *Counter) Inc() {
	atomic.AddInt64(&c.value, 1)
}

// Add adds the given value to the counter
func (c *Counter) Add(v int64) {
	atomic.AddInt64(&c.value, v)
}

// Value returns the current value
func (c *Counter) Value() int64 {
	return atomic.LoadInt64(&c.value)
}

// Metric returns the metric representation
func (c *Counter) Metric() Metric {
	return Metric{
		Name:        c.name,
		Type:        MetricTypeCounter,
		Value:       float64(c.Value()),
		Labels:      c.labels,
		Timestamp:   time.Now(),
		Description: c.description,
	}
}

// Gauge represents a value that can go up and down
type Gauge struct {
	name        string
	description string
	labels      map[string]string
	value       int64 // stored as int64 for atomic operations, represents float64 bits
}

// NewGauge creates a new gauge
func NewGauge(name, description string, labels map[string]string) *Gauge {
	return &Gauge{
		name:        name,
		description: description,
		labels:      labels,
	}
}

// Set sets the gauge to the given value
func (g *Gauge) Set(v float64) {
	atomic.StoreInt64(&g.value, int64(v*1000000)) // Store with precision
}

// Inc increments the gauge by 1
func (g *Gauge) Inc() {
	atomic.AddInt64(&g.value, 1000000)
}

// Dec decrements the gauge by 1
func (g *Gauge) Dec() {
	atomic.AddInt64(&g.value, -1000000)
}

// Add adds the given value to the gauge
func (g *Gauge) Add(v float64) {
	atomic.AddInt64(&g.value, int64(v*1000000))
}

// Value returns the current value
func (g *Gauge) Value() float64 {
	return float64(atomic.LoadInt64(&g.value)) / 1000000
}

// Metric returns the metric representation
func (g *Gauge) Metric() Metric {
	return Metric{
		Name:        g.name,
		Type:        MetricTypeGauge,
		Value:       g.Value(),
		Labels:      g.labels,
		Timestamp:   time.Now(),
		Description: g.description,
	}
}

// Histogram tracks the distribution of values
type Histogram struct {
	name        string
	description string
	labels      map[string]string
	buckets     []float64
	counts      []int64
	sum         int64
	count       int64
	mu          sync.RWMutex
}

// NewHistogram creates a new histogram with the given buckets
func NewHistogram(name, description string, labels map[string]string, buckets []float64) *Histogram {
	return &Histogram{
		name:        name,
		description: description,
		labels:      labels,
		buckets:     buckets,
		counts:      make([]int64, len(buckets)+1), // +1 for infinity bucket
	}
}

// DefaultBuckets returns default histogram buckets for latency (in milliseconds)
func DefaultBuckets() []float64 {
	return []float64{5, 10, 25, 50, 100, 250, 500, 1000, 2500, 5000, 10000}
}

// Observe records a value
func (h *Histogram) Observe(v float64) {
	h.mu.Lock()
	defer h.mu.Unlock()

	atomic.AddInt64(&h.sum, int64(v*1000000))
	atomic.AddInt64(&h.count, 1)

	for i, bucket := range h.buckets {
		if v <= bucket {
			atomic.AddInt64(&h.counts[i], 1)
			return
		}
	}
	// Value exceeds all buckets, add to infinity bucket
	atomic.AddInt64(&h.counts[len(h.buckets)], 1)
}

// Count returns the total count of observations
func (h *Histogram) Count() int64 {
	return atomic.LoadInt64(&h.count)
}

// Sum returns the sum of all observations
func (h *Histogram) Sum() float64 {
	return float64(atomic.LoadInt64(&h.sum)) / 1000000
}

// Mean returns the mean of all observations
func (h *Histogram) Mean() float64 {
	count := h.Count()
	if count == 0 {
		return 0
	}
	return h.Sum() / float64(count)
}

// Percentile returns an approximate percentile (0-100)
func (h *Histogram) Percentile(p float64) float64 {
	h.mu.RLock()
	defer h.mu.RUnlock()

	if h.count == 0 {
		return 0
	}

	target := int64(float64(h.count) * p / 100)
	cumulative := int64(0)

	for i, count := range h.counts {
		cumulative += count
		if cumulative >= target {
			if i < len(h.buckets) {
				return h.buckets[i]
			}
			return h.buckets[len(h.buckets)-1] * 2 // Approximate for infinity bucket
		}
	}
	return 0
}

// Metric returns the metric representation
func (h *Histogram) Metric() Metric {
	return Metric{
		Name:        h.name,
		Type:        MetricTypeHistogram,
		Value:       h.Mean(),
		Labels:      h.labels,
		Timestamp:   time.Now(),
		Description: h.description,
	}
}

// Summary provides percentile calculations over a sliding time window
type Summary struct {
	name        string
	description string
	labels      map[string]string
	values      []float64
	maxSize     int
	sum         float64
	count       int64
	mu          sync.RWMutex
}

// NewSummary creates a new summary
func NewSummary(name, description string, labels map[string]string, maxSize int) *Summary {
	if maxSize <= 0 {
		maxSize = 1000
	}
	return &Summary{
		name:        name,
		description: description,
		labels:      labels,
		values:      make([]float64, 0, maxSize),
		maxSize:     maxSize,
	}
}

// Observe records a value
func (s *Summary) Observe(v float64) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if len(s.values) >= s.maxSize {
		// Remove oldest value
		s.sum -= s.values[0]
		s.values = s.values[1:]
	}

	s.values = append(s.values, v)
	s.sum += v
	atomic.AddInt64(&s.count, 1)
}

// Count returns the total observations
func (s *Summary) Count() int64 {
	return atomic.LoadInt64(&s.count)
}

// Mean returns the mean of current window
func (s *Summary) Mean() float64 {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if len(s.values) == 0 {
		return 0
	}
	return s.sum / float64(len(s.values))
}

// Metric returns the metric representation
func (s *Summary) Metric() Metric {
	return Metric{
		Name:        s.name,
		Type:        MetricTypeSummary,
		Value:       s.Mean(),
		Labels:      s.labels,
		Timestamp:   time.Now(),
		Description: s.description,
	}
}

// MetricsRegistry holds all registered metrics
type MetricsRegistry struct {
	mu         sync.RWMutex
	counters   map[string]*Counter
	gauges     map[string]*Gauge
	histograms map[string]*Histogram
	summaries  map[string]*Summary
}

// NewMetricsRegistry creates a new metrics registry
func NewMetricsRegistry() *MetricsRegistry {
	return &MetricsRegistry{
		counters:   make(map[string]*Counter),
		gauges:     make(map[string]*Gauge),
		histograms: make(map[string]*Histogram),
		summaries:  make(map[string]*Summary),
	}
}

// Counter gets or creates a counter
func (r *MetricsRegistry) Counter(name, description string, labels map[string]string) *Counter {
	key := r.makeKey(name, labels)
	r.mu.Lock()
	defer r.mu.Unlock()

	if c, exists := r.counters[key]; exists {
		return c
	}

	c := NewCounter(name, description, labels)
	r.counters[key] = c
	return c
}

// Gauge gets or creates a gauge
func (r *MetricsRegistry) Gauge(name, description string, labels map[string]string) *Gauge {
	key := r.makeKey(name, labels)
	r.mu.Lock()
	defer r.mu.Unlock()

	if g, exists := r.gauges[key]; exists {
		return g
	}

	g := NewGauge(name, description, labels)
	r.gauges[key] = g
	return g
}

// Histogram gets or creates a histogram
func (r *MetricsRegistry) Histogram(name, description string, labels map[string]string, buckets []float64) *Histogram {
	key := r.makeKey(name, labels)
	r.mu.Lock()
	defer r.mu.Unlock()

	if h, exists := r.histograms[key]; exists {
		return h
	}

	if buckets == nil {
		buckets = DefaultBuckets()
	}
	h := NewHistogram(name, description, labels, buckets)
	r.histograms[key] = h
	return h
}

// Summary gets or creates a summary
func (r *MetricsRegistry) Summary(name, description string, labels map[string]string, maxSize int) *Summary {
	key := r.makeKey(name, labels)
	r.mu.Lock()
	defer r.mu.Unlock()

	if s, exists := r.summaries[key]; exists {
		return s
	}

	s := NewSummary(name, description, labels, maxSize)
	r.summaries[key] = s
	return s
}

// GetAll returns all metrics
func (r *MetricsRegistry) GetAll() []Metric {
	r.mu.RLock()
	defer r.mu.RUnlock()

	metrics := make([]Metric, 0)

	for _, c := range r.counters {
		metrics = append(metrics, c.Metric())
	}
	for _, g := range r.gauges {
		metrics = append(metrics, g.Metric())
	}
	for _, h := range r.histograms {
		metrics = append(metrics, h.Metric())
	}
	for _, s := range r.summaries {
		metrics = append(metrics, s.Metric())
	}

	return metrics
}

// makeKey creates a unique key for a metric
func (r *MetricsRegistry) makeKey(name string, labels map[string]string) string {
	key := name
	for k, v := range labels {
		key += ":" + k + "=" + v
	}
	return key
}

// RuntimeMetrics collects Go runtime metrics
type RuntimeMetrics struct {
	registry *MetricsRegistry
	interval time.Duration
	ctx      context.Context
	cancel   context.CancelFunc
}

// NewRuntimeMetrics creates a new runtime metrics collector
func NewRuntimeMetrics(registry *MetricsRegistry, interval time.Duration) *RuntimeMetrics {
	ctx, cancel := context.WithCancel(context.Background())
	return &RuntimeMetrics{
		registry: registry,
		interval: interval,
		ctx:      ctx,
		cancel:   cancel,
	}
}

// Start starts collecting runtime metrics
func (r *RuntimeMetrics) Start() {
	go r.collect()
}

// Stop stops collecting runtime metrics
func (r *RuntimeMetrics) Stop() {
	r.cancel()
}

func (r *RuntimeMetrics) collect() {
	ticker := time.NewTicker(r.interval)
	defer ticker.Stop()

	// Create gauges
	goroutines := r.registry.Gauge("go_goroutines", "Number of goroutines", nil)
	heapAlloc := r.registry.Gauge("go_heap_alloc_bytes", "Heap allocation in bytes", nil)
	heapSys := r.registry.Gauge("go_heap_sys_bytes", "Heap system bytes", nil)
	heapObjects := r.registry.Gauge("go_heap_objects", "Number of heap objects", nil)
	gcPauseTotal := r.registry.Gauge("go_gc_pause_total_ns", "Total GC pause time in nanoseconds", nil)
	numGC := r.registry.Gauge("go_gc_cycles", "Number of GC cycles", nil)
	cpuCount := r.registry.Gauge("go_cpu_count", "Number of CPUs", nil)

	for {
		select {
		case <-r.ctx.Done():
			return
		case <-ticker.C:
			var m runtime.MemStats
			runtime.ReadMemStats(&m)

			goroutines.Set(float64(runtime.NumGoroutine()))
			heapAlloc.Set(float64(m.HeapAlloc))
			heapSys.Set(float64(m.HeapSys))
			heapObjects.Set(float64(m.HeapObjects))
			gcPauseTotal.Set(float64(m.PauseTotalNs))
			numGC.Set(float64(m.NumGC))
			cpuCount.Set(float64(runtime.NumCPU()))
		}
	}
}

// Global default registry
var defaultRegistry = NewMetricsRegistry()

// DefaultRegistry returns the default metrics registry
func DefaultRegistry() *MetricsRegistry {
	return defaultRegistry
}

// SetDefaultRegistry sets the default registry
func SetDefaultRegistry(r *MetricsRegistry) {
	defaultRegistry = r
}
