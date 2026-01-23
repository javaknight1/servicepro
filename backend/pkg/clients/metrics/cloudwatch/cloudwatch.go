package cloudwatch

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch/types"

	"github.com/javaknight1/servicepro/backend/config"
	"github.com/javaknight1/servicepro/backend/pkg/clients/metrics"
)

func init() {
	metrics.RegisterProvider(metrics.ProviderCloudWatch, func(ctx context.Context, cfg *config.Config) (metrics.Client, error) {
		cwCfg := &Config{
			Region:          cfg.AWS.Region,
			AccessKeyID:     cfg.AWS.AccessKeyID,
			SecretAccessKey: cfg.AWS.SecretAccessKey,
			Namespace:       fmt.Sprintf("ServicePro/%s", cfg.Server.Env),
		}
		return NewClient(ctx, cwCfg, cfg)
	})
}

// Config holds configuration for the CloudWatch metrics client
type Config struct {
	// Region is the AWS region (required)
	Region string

	// AccessKeyID for AWS authentication (optional, uses default chain if empty)
	AccessKeyID string

	// SecretAccessKey for AWS authentication (optional, uses default chain if empty)
	SecretAccessKey string

	// Namespace is the CloudWatch namespace for metrics (required)
	Namespace string
}

// Client implements the metrics.Client interface using CloudWatch
type Client struct {
	config            *Config
	appConfig         *config.Config
	client            *cloudwatch.Client
	mu                sync.Mutex
	buffer            []*metrics.MetricData
	flushTicker       *time.Ticker
	stopChan          chan struct{}
	wg                sync.WaitGroup
	counters          map[string]*cwCounter
	gauges            map[string]*cwGauge
	histograms        map[string]*cwHistogram
	summaries         map[string]*cwSummary
	defaultDimensions map[string]string
}

// NewClient creates a new CloudWatch metrics client
func NewClient(ctx context.Context, cfg *Config, appCfg *config.Config) (*Client, error) {
	if cfg.Region == "" {
		return nil, fmt.Errorf("AWS region is required")
	}

	if cfg.Namespace == "" {
		return nil, fmt.Errorf("CloudWatch namespace is required")
	}

	// Build AWS config options
	var opts []func(*awsconfig.LoadOptions) error
	opts = append(opts, awsconfig.WithRegion(cfg.Region))

	if cfg.AccessKeyID != "" && cfg.SecretAccessKey != "" {
		opts = append(opts, awsconfig.WithCredentialsProvider(
			credentials.NewStaticCredentialsProvider(
				cfg.AccessKeyID,
				cfg.SecretAccessKey,
				"",
			),
		))
	}

	awsCfg, err := awsconfig.LoadDefaultConfig(ctx, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	client := cloudwatch.NewFromConfig(awsCfg)

	batchOpts := metrics.DefaultBatchOptions()

	c := &Client{
		config:            cfg,
		appConfig:         appCfg,
		client:            client,
		buffer:            make([]*metrics.MetricData, 0, batchOpts.MaxSize),
		stopChan:          make(chan struct{}),
		counters:          make(map[string]*cwCounter),
		gauges:            make(map[string]*cwGauge),
		histograms:        make(map[string]*cwHistogram),
		summaries:         make(map[string]*cwSummary),
		defaultDimensions: make(map[string]string),
	}

	// Start background flush ticker
	c.flushTicker = time.NewTicker(batchOpts.FlushInterval)
	c.wg.Add(1)
	go c.flushLoop()

	return c, nil
}

// Counter implements metrics.Client
func (c *Client) Counter(opts metrics.CounterOptions) metrics.Counter {
	c.mu.Lock()
	defer c.mu.Unlock()

	name := c.fullName(opts.Namespace, opts.Name)
	if counter, exists := c.counters[name]; exists {
		return counter
	}

	counter := &cwCounter{
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

	gauge := &cwGauge{
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

	histogram := &cwHistogram{
		client:  c,
		name:    name,
		help:    opts.Help,
		labels:  opts.Labels,
		buckets: opts.Buckets,
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

	summary := &cwSummary{
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

	c.mu.Lock()
	c.buffer = append(c.buffer, data)
	shouldFlush := len(c.buffer) >= c.getBatchSize()
	c.mu.Unlock()

	if shouldFlush {
		return c.Flush()
	}

	return nil
}

// RecordBatch implements metrics.Client
func (c *Client) RecordBatch(ctx context.Context, data []*metrics.MetricData) error {
	for _, d := range data {
		c.applyDefaults(d)
		c.mu.Lock()
		c.buffer = append(c.buffer, d)
		c.mu.Unlock()
	}
	return c.Flush()
}

// Handler implements metrics.Client (not applicable for CloudWatch)
func (c *Client) Handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotImplemented)
		w.Write([]byte("CloudWatch is a push-based metrics system; use /metrics with Prometheus instead"))
	})
}

// Flush implements metrics.Client
func (c *Client) Flush() error {
	c.mu.Lock()
	if len(c.buffer) == 0 {
		c.mu.Unlock()
		return nil
	}

	data := c.buffer
	c.buffer = make([]*metrics.MetricData, 0, c.getBatchSize())
	c.mu.Unlock()

	return c.sendMetrics(context.Background(), data)
}

// Close implements metrics.Client
func (c *Client) Close() error {
	close(c.stopChan)
	c.flushTicker.Stop()
	c.wg.Wait()

	return c.Flush()
}

func (c *Client) flushLoop() {
	defer c.wg.Done()

	for {
		select {
		case <-c.flushTicker.C:
			c.Flush()
		case <-c.stopChan:
			return
		}
	}
}

func (c *Client) sendMetrics(ctx context.Context, data []*metrics.MetricData) error {
	if len(data) == 0 {
		return nil
	}

	// CloudWatch PutMetricData allows max 1000 metrics per call
	// and max 150 metrics per request for high resolution
	const maxMetricsPerCall = 20

	for i := 0; i < len(data); i += maxMetricsPerCall {
		end := i + maxMetricsPerCall
		if end > len(data) {
			end = len(data)
		}

		batch := data[i:end]
		metricData := make([]types.MetricDatum, len(batch))

		for j, d := range batch {
			datum := types.MetricDatum{
				MetricName: aws.String(d.Name),
				Timestamp:  aws.Time(d.Timestamp),
			}

			// Set value or statistic values
			if d.StatisticValues != nil {
				datum.StatisticValues = &types.StatisticSet{
					SampleCount: aws.Float64(d.StatisticValues.SampleCount),
					Sum:         aws.Float64(d.StatisticValues.Sum),
					Minimum:     aws.Float64(d.StatisticValues.Minimum),
					Maximum:     aws.Float64(d.StatisticValues.Maximum),
				}
			} else {
				datum.Value = aws.Float64(d.Value)
			}

			// Set unit
			if d.Unit != "" {
				datum.Unit = types.StandardUnit(d.Unit)
			}

			// Set dimensions
			if len(d.Dimensions) > 0 {
				dims := make([]types.Dimension, 0, len(d.Dimensions))
				for k, v := range d.Dimensions {
					dims = append(dims, types.Dimension{
						Name:  aws.String(k),
						Value: aws.String(v),
					})
				}
				datum.Dimensions = dims
			}

			metricData[j] = datum
		}

		_, err := c.client.PutMetricData(ctx, &cloudwatch.PutMetricDataInput{
			Namespace:  aws.String(c.config.Namespace),
			MetricData: metricData,
		})
		if err != nil {
			return fmt.Errorf("failed to send metrics to CloudWatch: %w", err)
		}
	}

	return nil
}

func (c *Client) fullName(namespace, name string) string {
	ns := namespace
	if ns == "" {
		ns = c.config.Namespace
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

func (c *Client) getBatchSize() int {
	return 100
}

func (c *Client) recordMetric(name string, value float64, unit metrics.MetricUnit, dimensions map[string]string) {
	data := &metrics.MetricData{
		Name:       name,
		Value:      value,
		Unit:       unit,
		Timestamp:  time.Now(),
		Dimensions: dimensions,
	}
	c.Record(context.Background(), data)
}

// cwCounter implements metrics.Counter for CloudWatch
type cwCounter struct {
	client      *Client
	name        string
	help        string
	labels      []string
	labelValues map[string]string
	mu          sync.Mutex
	value       float64
}

func (c *cwCounter) Inc() {
	c.Add(1)
}

func (c *cwCounter) Add(v float64) {
	c.mu.Lock()
	c.value += v
	c.mu.Unlock()

	c.client.recordMetric(c.name, v, metrics.UnitCount, c.labelValues)
}

func (c *cwCounter) WithLabels(labels map[string]string) metrics.Counter {
	return &cwCounter{
		client:      c.client,
		name:        c.name,
		help:        c.help,
		labels:      c.labels,
		labelValues: labels,
	}
}

// cwGauge implements metrics.Gauge for CloudWatch
type cwGauge struct {
	client      *Client
	name        string
	help        string
	labels      []string
	labelValues map[string]string
	mu          sync.Mutex
	value       float64
}

func (g *cwGauge) Set(v float64) {
	g.mu.Lock()
	g.value = v
	g.mu.Unlock()

	g.client.recordMetric(g.name, v, metrics.UnitNone, g.labelValues)
}

func (g *cwGauge) Inc() {
	g.Add(1)
}

func (g *cwGauge) Dec() {
	g.Sub(1)
}

func (g *cwGauge) Add(v float64) {
	g.mu.Lock()
	g.value += v
	val := g.value
	g.mu.Unlock()

	g.client.recordMetric(g.name, val, metrics.UnitNone, g.labelValues)
}

func (g *cwGauge) Sub(v float64) {
	g.mu.Lock()
	g.value -= v
	val := g.value
	g.mu.Unlock()

	g.client.recordMetric(g.name, val, metrics.UnitNone, g.labelValues)
}

func (g *cwGauge) WithLabels(labels map[string]string) metrics.Gauge {
	return &cwGauge{
		client:      g.client,
		name:        g.name,
		help:        g.help,
		labels:      g.labels,
		labelValues: labels,
	}
}

// cwHistogram implements metrics.Histogram for CloudWatch
type cwHistogram struct {
	client      *Client
	name        string
	help        string
	labels      []string
	labelValues map[string]string
	buckets     []float64
	mu          sync.Mutex
	count       uint64
	sum         float64
	min         float64
	max         float64
}

func (h *cwHistogram) Observe(v float64) {
	h.mu.Lock()
	h.count++
	h.sum += v
	if h.count == 1 || v < h.min {
		h.min = v
	}
	if h.count == 1 || v > h.max {
		h.max = v
	}
	count := h.count
	sum := h.sum
	min := h.min
	max := h.max
	h.mu.Unlock()

	// Send as statistic set for better CloudWatch aggregation
	data := &metrics.MetricData{
		Name:       h.name,
		Timestamp:  time.Now(),
		Dimensions: h.labelValues,
		StatisticValues: &metrics.StatisticValues{
			SampleCount: float64(count),
			Sum:         sum,
			Minimum:     min,
			Maximum:     max,
		},
	}
	h.client.Record(context.Background(), data)
}

func (h *cwHistogram) WithLabels(labels map[string]string) metrics.Histogram {
	return &cwHistogram{
		client:      h.client,
		name:        h.name,
		help:        h.help,
		labels:      h.labels,
		labelValues: labels,
		buckets:     h.buckets,
	}
}

// cwSummary implements metrics.Summary for CloudWatch
type cwSummary struct {
	client      *Client
	name        string
	help        string
	labels      []string
	labelValues map[string]string
	objectives  map[float64]float64
	mu          sync.Mutex
	count       uint64
	sum         float64
	min         float64
	max         float64
}

func (s *cwSummary) Observe(v float64) {
	s.mu.Lock()
	s.count++
	s.sum += v
	if s.count == 1 || v < s.min {
		s.min = v
	}
	if s.count == 1 || v > s.max {
		s.max = v
	}
	count := s.count
	sum := s.sum
	min := s.min
	max := s.max
	s.mu.Unlock()

	data := &metrics.MetricData{
		Name:       s.name,
		Timestamp:  time.Now(),
		Dimensions: s.labelValues,
		StatisticValues: &metrics.StatisticValues{
			SampleCount: float64(count),
			Sum:         sum,
			Minimum:     min,
			Maximum:     max,
		},
	}
	s.client.Record(context.Background(), data)
}

func (s *cwSummary) WithLabels(labels map[string]string) metrics.Summary {
	return &cwSummary{
		client:      s.client,
		name:        s.name,
		help:        s.help,
		labels:      s.labels,
		labelValues: labels,
		objectives:  s.objectives,
	}
}

// Ensure Client implements metrics.Client
var _ metrics.Client = (*Client)(nil)
