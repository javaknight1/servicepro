package prometheus

import (
	"context"
	"fmt"
	"net/http"
	"sync"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/javaknight1/servicepro/backend/config"
	"github.com/javaknight1/servicepro/backend/pkg/clients/metrics"
)

func init() {
	metrics.RegisterProvider(metrics.ProviderPrometheus, func(ctx context.Context, cfg *config.Config) (metrics.Client, error) {
		promCfg := &Config{
			Registry: prometheus.DefaultRegisterer,
		}
		return NewClient(promCfg, cfg)
	})
}

// Config holds configuration for the Prometheus metrics client
type Config struct {
	// Registry is the Prometheus registry to use (defaults to prometheus.DefaultRegisterer)
	Registry prometheus.Registerer

	// Gatherer is the Prometheus gatherer to use (defaults to prometheus.DefaultGatherer)
	Gatherer prometheus.Gatherer
}

// Client implements the metrics.Client interface using Prometheus
type Client struct {
	config            *Config
	appConfig         *config.Config
	registry          prometheus.Registerer
	gatherer          prometheus.Gatherer
	mu                sync.RWMutex
	counters          map[string]*prometheus.CounterVec
	gauges            map[string]*prometheus.GaugeVec
	histograms        map[string]*prometheus.HistogramVec
	summaries         map[string]*prometheus.SummaryVec
	namespace         string
	defaultDimensions map[string]string
}

// NewClient creates a new Prometheus metrics client
func NewClient(cfg *Config, appCfg *config.Config) (*Client, error) {
	registry := cfg.Registry
	if registry == nil {
		registry = prometheus.DefaultRegisterer
	}

	gatherer := cfg.Gatherer
	if gatherer == nil {
		gatherer = prometheus.DefaultGatherer
	}

	c := &Client{
		config:            cfg,
		appConfig:         appCfg,
		registry:          registry,
		gatherer:          gatherer,
		counters:          make(map[string]*prometheus.CounterVec),
		gauges:            make(map[string]*prometheus.GaugeVec),
		histograms:        make(map[string]*prometheus.HistogramVec),
		summaries:         make(map[string]*prometheus.SummaryVec),
		defaultDimensions: make(map[string]string),
	}

	if appCfg != nil && appCfg.Prometheus.Namespace != "" {
		c.namespace = appCfg.Prometheus.Namespace
	}

	return c, nil
}

// Counter implements metrics.Client
func (c *Client) Counter(opts metrics.CounterOptions) metrics.Counter {
	c.mu.Lock()
	defer c.mu.Unlock()

	name := c.fullName(opts.Namespace, opts.Name)
	if counter, exists := c.counters[name]; exists {
		return &promCounter{vec: counter, labels: opts.Labels}
	}

	counterVec := prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: c.namespace,
		Name:      opts.Name,
		Help:      opts.Help,
	}, opts.Labels)

	c.registry.MustRegister(counterVec)
	c.counters[name] = counterVec

	return &promCounter{vec: counterVec, labels: opts.Labels}
}

// Gauge implements metrics.Client
func (c *Client) Gauge(opts metrics.GaugeOptions) metrics.Gauge {
	c.mu.Lock()
	defer c.mu.Unlock()

	name := c.fullName(opts.Namespace, opts.Name)
	if gauge, exists := c.gauges[name]; exists {
		return &promGauge{vec: gauge, labels: opts.Labels}
	}

	gaugeVec := prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: c.namespace,
		Name:      opts.Name,
		Help:      opts.Help,
	}, opts.Labels)

	c.registry.MustRegister(gaugeVec)
	c.gauges[name] = gaugeVec

	return &promGauge{vec: gaugeVec, labels: opts.Labels}
}

// Histogram implements metrics.Client
func (c *Client) Histogram(opts metrics.HistogramOptions) metrics.Histogram {
	c.mu.Lock()
	defer c.mu.Unlock()

	name := c.fullName(opts.Namespace, opts.Name)
	if histogram, exists := c.histograms[name]; exists {
		return &promHistogram{vec: histogram, labels: opts.Labels}
	}

	buckets := opts.Buckets
	if len(buckets) == 0 {
		buckets = prometheus.DefBuckets
	}

	histogramVec := prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: c.namespace,
		Name:      opts.Name,
		Help:      opts.Help,
		Buckets:   buckets,
	}, opts.Labels)

	c.registry.MustRegister(histogramVec)
	c.histograms[name] = histogramVec

	return &promHistogram{vec: histogramVec, labels: opts.Labels}
}

// Summary implements metrics.Client
func (c *Client) Summary(opts metrics.SummaryOptions) metrics.Summary {
	c.mu.Lock()
	defer c.mu.Unlock()

	name := c.fullName(opts.Namespace, opts.Name)
	if summary, exists := c.summaries[name]; exists {
		return &promSummary{vec: summary, labels: opts.Labels}
	}

	objectives := opts.Objectives
	if objectives == nil {
		objectives = map[float64]float64{0.5: 0.05, 0.9: 0.01, 0.99: 0.001}
	}

	summaryVec := prometheus.NewSummaryVec(prometheus.SummaryOpts{
		Namespace:  c.namespace,
		Name:       opts.Name,
		Help:       opts.Help,
		Objectives: objectives,
		MaxAge:     opts.MaxAge,
	}, opts.Labels)

	c.registry.MustRegister(summaryVec)
	c.summaries[name] = summaryVec

	return &promSummary{vec: summaryVec, labels: opts.Labels}
}

// Record implements metrics.Client (not typically used with Prometheus)
func (c *Client) Record(ctx context.Context, data *metrics.MetricData) error {
	// Prometheus is primarily pull-based, but we can record via the appropriate metric type
	switch data.Type {
	case metrics.MetricTypeCounter:
		counter := c.Counter(metrics.CounterOptions{Name: data.Name})
		if len(data.Dimensions) > 0 {
			counter = counter.WithLabels(data.Dimensions)
		}
		counter.Add(data.Value)
	case metrics.MetricTypeGauge:
		gauge := c.Gauge(metrics.GaugeOptions{Name: data.Name})
		if len(data.Dimensions) > 0 {
			gauge = gauge.WithLabels(data.Dimensions)
		}
		gauge.Set(data.Value)
	case metrics.MetricTypeHistogram:
		histogram := c.Histogram(metrics.HistogramOptions{Name: data.Name})
		if len(data.Dimensions) > 0 {
			histogram = histogram.WithLabels(data.Dimensions)
		}
		histogram.Observe(data.Value)
	case metrics.MetricTypeSummary:
		summary := c.Summary(metrics.SummaryOptions{Name: data.Name})
		if len(data.Dimensions) > 0 {
			summary = summary.WithLabels(data.Dimensions)
		}
		summary.Observe(data.Value)
	default:
		return fmt.Errorf("unknown metric type: %s", data.Type)
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
	return promhttp.HandlerFor(c.gatherer, promhttp.HandlerOpts{})
}

// Flush implements metrics.Client (no-op for Prometheus, metrics are scraped)
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

// promCounter wraps a Prometheus CounterVec
type promCounter struct {
	vec         *prometheus.CounterVec
	labels      []string
	labelValues prometheus.Labels
}

func (c *promCounter) Inc() {
	c.getCounter().Inc()
}

func (c *promCounter) Add(v float64) {
	c.getCounter().Add(v)
}

func (c *promCounter) WithLabels(labels map[string]string) metrics.Counter {
	return &promCounter{
		vec:         c.vec,
		labels:      c.labels,
		labelValues: labels,
	}
}

func (c *promCounter) getCounter() prometheus.Counter {
	if c.labelValues != nil {
		return c.vec.With(c.labelValues)
	}
	return c.vec.With(prometheus.Labels{})
}

// promGauge wraps a Prometheus GaugeVec
type promGauge struct {
	vec         *prometheus.GaugeVec
	labels      []string
	labelValues prometheus.Labels
}

func (g *promGauge) Set(v float64) {
	g.getGauge().Set(v)
}

func (g *promGauge) Inc() {
	g.getGauge().Inc()
}

func (g *promGauge) Dec() {
	g.getGauge().Dec()
}

func (g *promGauge) Add(v float64) {
	g.getGauge().Add(v)
}

func (g *promGauge) Sub(v float64) {
	g.getGauge().Sub(v)
}

func (g *promGauge) WithLabels(labels map[string]string) metrics.Gauge {
	return &promGauge{
		vec:         g.vec,
		labels:      g.labels,
		labelValues: labels,
	}
}

func (g *promGauge) getGauge() prometheus.Gauge {
	if g.labelValues != nil {
		return g.vec.With(g.labelValues)
	}
	return g.vec.With(prometheus.Labels{})
}

// promHistogram wraps a Prometheus HistogramVec
type promHistogram struct {
	vec         *prometheus.HistogramVec
	labels      []string
	labelValues prometheus.Labels
}

func (h *promHistogram) Observe(v float64) {
	h.getHistogram().Observe(v)
}

func (h *promHistogram) WithLabels(labels map[string]string) metrics.Histogram {
	return &promHistogram{
		vec:         h.vec,
		labels:      h.labels,
		labelValues: labels,
	}
}

func (h *promHistogram) getHistogram() prometheus.Observer {
	if h.labelValues != nil {
		return h.vec.With(h.labelValues)
	}
	return h.vec.With(prometheus.Labels{})
}

// promSummary wraps a Prometheus SummaryVec
type promSummary struct {
	vec         *prometheus.SummaryVec
	labels      []string
	labelValues prometheus.Labels
}

func (s *promSummary) Observe(v float64) {
	s.getSummary().Observe(v)
}

func (s *promSummary) WithLabels(labels map[string]string) metrics.Summary {
	return &promSummary{
		vec:         s.vec,
		labels:      s.labels,
		labelValues: labels,
	}
}

func (s *promSummary) getSummary() prometheus.Observer {
	if s.labelValues != nil {
		return s.vec.With(s.labelValues)
	}
	return s.vec.With(prometheus.Labels{})
}

// Ensure Client implements metrics.Client
var _ metrics.Client = (*Client)(nil)
