package mock

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/javaknight1/servicepro/backend/config"
	"github.com/javaknight1/servicepro/backend/pkg/clients/logging"
	"github.com/javaknight1/servicepro/backend/pkg/clients/metrics"
)

func init() {
	metrics.RegisterProvider(metrics.ProviderMock, func(ctx context.Context, cfg *config.Config) (metrics.Client, error) {
		mockCfg := &Config{
			PrintToStdout: true,
			StoreMetrics:  false,
		}
		return NewClient(mockCfg, cfg), nil
	})
}

// Config holds configuration for the mock metrics client
type Config struct {
	// PrintToStdout prints metrics to stdout (default: true)
	PrintToStdout bool

	// StoreMetrics stores metrics in memory for testing (default: false)
	StoreMetrics bool
}

// Client implements the metrics.Client interface for testing and development
type Client struct {
	config            *Config
	appConfig         *config.Config
	mu                sync.RWMutex
	counters          map[string]*mockCounter
	gauges            map[string]*mockGauge
	histograms        map[string]*mockHistogram
	summaries         map[string]*mockSummary
	recordedMetrics   []*metrics.MetricData
	namespace         string
	defaultDimensions map[string]string
}

// NewClient creates a new mock metrics client
func NewClient(cfg *Config, appCfg *config.Config) *Client {
	if cfg == nil {
		cfg = &Config{
			PrintToStdout: true,
			StoreMetrics:  false,
		}
	}

	c := &Client{
		config:            cfg,
		appConfig:         appCfg,
		counters:          make(map[string]*mockCounter),
		gauges:            make(map[string]*mockGauge),
		histograms:        make(map[string]*mockHistogram),
		summaries:         make(map[string]*mockSummary),
		recordedMetrics:   make([]*metrics.MetricData, 0),
		defaultDimensions: make(map[string]string),
	}

	if appCfg != nil && appCfg.Prometheus.Namespace != "" {
		c.namespace = appCfg.Prometheus.Namespace
	}

	return c
}

// Counter implements metrics.Client
func (c *Client) Counter(opts metrics.CounterOptions) metrics.Counter {
	c.mu.Lock()
	defer c.mu.Unlock()

	name := c.fullName(opts.Namespace, opts.Name)
	if counter, exists := c.counters[name]; exists {
		return counter
	}

	counter := &mockCounter{
		client: c,
		name:   name,
		help:   opts.Help,
		labels: opts.Labels,
	}
	c.counters[name] = counter
	return counter
}

// Gauge implements metrics.Client
func (c *Client) Gauge(opts metrics.GaugeOptions) metrics.Gauge {
	c.mu.Lock()
	defer c.mu.Unlock()

	name := c.fullName(opts.Namespace, opts.Name)
	if gauge, exists := c.gauges[name]; exists {
		return gauge
	}

	gauge := &mockGauge{
		client: c,
		name:   name,
		help:   opts.Help,
		labels: opts.Labels,
	}
	c.gauges[name] = gauge
	return gauge
}

// Histogram implements metrics.Client
func (c *Client) Histogram(opts metrics.HistogramOptions) metrics.Histogram {
	c.mu.Lock()
	defer c.mu.Unlock()

	name := c.fullName(opts.Namespace, opts.Name)
	if histogram, exists := c.histograms[name]; exists {
		return histogram
	}

	buckets := opts.Buckets
	if len(buckets) == 0 {
		buckets = metrics.DefaultHistogramBuckets()
	}

	histogram := &mockHistogram{
		client:  c,
		name:    name,
		help:    opts.Help,
		labels:  opts.Labels,
		buckets: buckets,
	}
	c.histograms[name] = histogram
	return histogram
}

// Summary implements metrics.Client
func (c *Client) Summary(opts metrics.SummaryOptions) metrics.Summary {
	c.mu.Lock()
	defer c.mu.Unlock()

	name := c.fullName(opts.Namespace, opts.Name)
	if summary, exists := c.summaries[name]; exists {
		return summary
	}

	summary := &mockSummary{
		client:     c,
		name:       name,
		help:       opts.Help,
		labels:     opts.Labels,
		objectives: opts.Objectives,
	}
	c.summaries[name] = summary
	return summary
}

// Record implements metrics.Client
func (c *Client) Record(ctx context.Context, data *metrics.MetricData) error {
	c.applyDefaults(data)

	if c.config.StoreMetrics {
		c.mu.Lock()
		c.recordedMetrics = append(c.recordedMetrics, data)
		c.mu.Unlock()
	}

	if c.config.PrintToStdout {
		c.printMetric(data)
	}

	return nil
}

// RecordBatch implements metrics.Client
func (c *Client) RecordBatch(ctx context.Context, data []*metrics.MetricData) error {
	for _, d := range data {
		if err := c.Record(ctx, d); err != nil {
			return err
		}
	}
	return nil
}

// Handler implements metrics.Client
func (c *Client) Handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c.mu.RLock()
		defer c.mu.RUnlock()

		w.Header().Set("Content-Type", "text/plain")

		// Output in a simple format
		for name, counter := range c.counters {
			fmt.Fprintf(w, "# HELP %s %s\n", name, counter.help)
			fmt.Fprintf(w, "# TYPE %s counter\n", name)
			fmt.Fprintf(w, "%s %f\n", name, counter.value)
		}

		for name, gauge := range c.gauges {
			fmt.Fprintf(w, "# HELP %s %s\n", name, gauge.help)
			fmt.Fprintf(w, "# TYPE %s gauge\n", name)
			fmt.Fprintf(w, "%s %f\n", name, gauge.value)
		}

		for name, histogram := range c.histograms {
			fmt.Fprintf(w, "# HELP %s %s\n", name, histogram.help)
			fmt.Fprintf(w, "# TYPE %s histogram\n", name)
			fmt.Fprintf(w, "%s_count %d\n", name, histogram.count)
			fmt.Fprintf(w, "%s_sum %f\n", name, histogram.sum)
		}
	})
}

// Flush implements metrics.Client
func (c *Client) Flush() error {
	return nil
}

// Close implements metrics.Client
func (c *Client) Close() error {
	return nil
}

func (c *Client) fullName(namespace, name string) string {
	ns := namespace
	if ns == "" {
		ns = c.namespace
	}
	if ns != "" {
		return ns + "_" + name
	}
	return name
}

func (c *Client) applyDefaults(data *metrics.MetricData) {
	if data.Timestamp.IsZero() {
		data.Timestamp = time.Now()
	}
	if data.Dimensions == nil {
		data.Dimensions = make(map[string]string)
	}
	for k, v := range c.defaultDimensions {
		if _, exists := data.Dimensions[k]; !exists {
			data.Dimensions[k] = v
		}
	}
}

func (c *Client) printMetric(data *metrics.MetricData) {
	logging.Info(context.Background(), "[METRICS-MOCK] Metric recorded", map[string]any{"timestamp": data.Timestamp.Format(time.RFC3339), "name": data.Name, "value": data.Value, "type": data.Type, "unit": data.Unit, "dimensions": data.Dimensions})
}

// GetRecordedMetrics returns all recorded metrics (for testing)
func (c *Client) GetRecordedMetrics() []*metrics.MetricData {
	c.mu.RLock()
	defer c.mu.RUnlock()

	result := make([]*metrics.MetricData, len(c.recordedMetrics))
	copy(result, c.recordedMetrics)
	return result
}

// ClearRecordedMetrics clears all recorded metrics (for testing)
func (c *Client) ClearRecordedMetrics() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.recordedMetrics = make([]*metrics.MetricData, 0)
}

// mockCounter implements metrics.Counter
type mockCounter struct {
	client      *Client
	name        string
	help        string
	labels      []string
	labelValues map[string]string
	mu          sync.Mutex
	value       float64
}

func (c *mockCounter) Inc() {
	c.Add(1)
}

func (c *mockCounter) Add(v float64) {
	c.mu.Lock()
	c.value += v
	c.mu.Unlock()

	if c.client.config.PrintToStdout {
		logging.Info(context.Background(), "[METRICS-MOCK] Counter add", map[string]any{"name": c.name, "delta": v, "total": c.value, "labels": c.labelValues})
	}
}

func (c *mockCounter) WithLabels(labels map[string]string) metrics.Counter {
	return &mockCounter{
		client:      c.client,
		name:        c.name,
		help:        c.help,
		labels:      c.labels,
		labelValues: labels,
		value:       c.value,
	}
}

// mockGauge implements metrics.Gauge
type mockGauge struct {
	client      *Client
	name        string
	help        string
	labels      []string
	labelValues map[string]string
	mu          sync.Mutex
	value       float64
}

func (g *mockGauge) Set(v float64) {
	g.mu.Lock()
	g.value = v
	g.mu.Unlock()

	if g.client.config.PrintToStdout {
		logging.Info(context.Background(), "[METRICS-MOCK] Gauge set", map[string]any{"name": g.name, "value": v, "labels": g.labelValues})
	}
}

func (g *mockGauge) Inc() {
	g.Add(1)
}

func (g *mockGauge) Dec() {
	g.Sub(1)
}

func (g *mockGauge) Add(v float64) {
	g.mu.Lock()
	g.value += v
	g.mu.Unlock()

	if g.client.config.PrintToStdout {
		logging.Info(context.Background(), "[METRICS-MOCK] Gauge add", map[string]any{"name": g.name, "delta": v, "total": g.value, "labels": g.labelValues})
	}
}

func (g *mockGauge) Sub(v float64) {
	g.mu.Lock()
	g.value -= v
	g.mu.Unlock()

	if g.client.config.PrintToStdout {
		logging.Info(context.Background(), "[METRICS-MOCK] Gauge sub", map[string]any{"name": g.name, "delta": v, "total": g.value, "labels": g.labelValues})
	}
}

func (g *mockGauge) WithLabels(labels map[string]string) metrics.Gauge {
	return &mockGauge{
		client:      g.client,
		name:        g.name,
		help:        g.help,
		labels:      g.labels,
		labelValues: labels,
		value:       g.value,
	}
}

// mockHistogram implements metrics.Histogram
type mockHistogram struct {
	client      *Client
	name        string
	help        string
	labels      []string
	labelValues map[string]string
	buckets     []float64
	mu          sync.Mutex
	count       uint64
	sum         float64
}

func (h *mockHistogram) Observe(v float64) {
	h.mu.Lock()
	h.count++
	h.sum += v
	h.mu.Unlock()

	if h.client.config.PrintToStdout {
		logging.Info(context.Background(), "[METRICS-MOCK] Histogram observed", map[string]any{"name": h.name, "value": v, "count": h.count, "sum": h.sum, "labels": h.labelValues})
	}
}

func (h *mockHistogram) WithLabels(labels map[string]string) metrics.Histogram {
	return &mockHistogram{
		client:      h.client,
		name:        h.name,
		help:        h.help,
		labels:      h.labels,
		labelValues: labels,
		buckets:     h.buckets,
		count:       h.count,
		sum:         h.sum,
	}
}

// mockSummary implements metrics.Summary
type mockSummary struct {
	client      *Client
	name        string
	help        string
	labels      []string
	labelValues map[string]string
	objectives  map[float64]float64
	mu          sync.Mutex
	count       uint64
	sum         float64
}

func (s *mockSummary) Observe(v float64) {
	s.mu.Lock()
	s.count++
	s.sum += v
	s.mu.Unlock()

	if s.client.config.PrintToStdout {
		logging.Info(context.Background(), "[METRICS-MOCK] Summary observed", map[string]any{"name": s.name, "value": v, "count": s.count, "sum": s.sum, "labels": s.labelValues})
	}
}

func (s *mockSummary) WithLabels(labels map[string]string) metrics.Summary {
	return &mockSummary{
		client:      s.client,
		name:        s.name,
		help:        s.help,
		labels:      s.labels,
		labelValues: labels,
		objectives:  s.objectives,
		count:       s.count,
		sum:         s.sum,
	}
}

// Ensure Client implements metrics.Client
var _ metrics.Client = (*Client)(nil)
