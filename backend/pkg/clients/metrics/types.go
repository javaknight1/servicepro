package metrics

import (
	"time"
)

// Provider represents a metrics service provider
type Provider string

const (
	ProviderCloudWatch Provider = "cloudwatch"
	ProviderPrometheus Provider = "prometheus"
	ProviderMock       Provider = "mock"
)

// MetricType represents the type of metric
type MetricType string

const (
	// MetricTypeCounter is a monotonically increasing counter
	MetricTypeCounter MetricType = "counter"

	// MetricTypeGauge is a value that can go up or down
	MetricTypeGauge MetricType = "gauge"

	// MetricTypeHistogram tracks the distribution of values
	MetricTypeHistogram MetricType = "histogram"

	// MetricTypeSummary tracks quantiles of values
	MetricTypeSummary MetricType = "summary"
)

// MetricUnit represents the unit of a metric
type MetricUnit string

const (
	UnitNone         MetricUnit = ""
	UnitCount        MetricUnit = "Count"
	UnitBytes        MetricUnit = "Bytes"
	UnitSeconds      MetricUnit = "Seconds"
	UnitMilliseconds MetricUnit = "Milliseconds"
	UnitMicroseconds MetricUnit = "Microseconds"
	UnitPercent      MetricUnit = "Percent"
	UnitBytesPerSec  MetricUnit = "Bytes/Second"
	UnitCountPerSec  MetricUnit = "Count/Second"
)

// MetricData represents a single metric data point
type MetricData struct {
	// Name is the metric name
	Name string

	// Value is the metric value
	Value float64

	// Type is the metric type
	Type MetricType

	// Unit is the metric unit
	Unit MetricUnit

	// Timestamp is when the metric was recorded
	Timestamp time.Time

	// Dimensions are key-value pairs for filtering/grouping
	Dimensions map[string]string

	// StatisticValues for pre-aggregated data
	StatisticValues *StatisticValues
}

// StatisticValues contains pre-aggregated statistics
type StatisticValues struct {
	SampleCount float64
	Sum         float64
	Minimum     float64
	Maximum     float64
}

// HistogramData contains histogram-specific data
type HistogramData struct {
	// Buckets maps bucket upper bounds to counts
	Buckets map[float64]uint64

	// Sum is the sum of all observed values
	Sum float64

	// Count is the total number of observations
	Count uint64
}

// CounterOptions configures a counter metric
type CounterOptions struct {
	// Name is the metric name (required)
	Name string

	// Help is a description of the metric
	Help string

	// Labels are the label names for this metric
	Labels []string

	// Namespace is an optional prefix
	Namespace string
}

// GaugeOptions configures a gauge metric
type GaugeOptions struct {
	// Name is the metric name (required)
	Name string

	// Help is a description of the metric
	Help string

	// Labels are the label names for this metric
	Labels []string

	// Namespace is an optional prefix
	Namespace string
}

// HistogramOptions configures a histogram metric
type HistogramOptions struct {
	// Name is the metric name (required)
	Name string

	// Help is a description of the metric
	Help string

	// Labels are the label names for this metric
	Labels []string

	// Buckets defines the histogram bucket boundaries
	Buckets []float64

	// Namespace is an optional prefix
	Namespace string
}

// SummaryOptions configures a summary metric
type SummaryOptions struct {
	// Name is the metric name (required)
	Name string

	// Help is a description of the metric
	Help string

	// Labels are the label names for this metric
	Labels []string

	// Objectives maps quantile to acceptable error
	Objectives map[float64]float64

	// MaxAge defines the duration for which an observation stays relevant
	MaxAge time.Duration

	// Namespace is an optional prefix
	Namespace string
}

// Counter represents a counter metric
type Counter interface {
	// Inc increments the counter by 1
	Inc()

	// Add adds the given value to the counter
	Add(float64)

	// WithLabels returns a counter with the given label values
	WithLabels(labels map[string]string) Counter
}

// Gauge represents a gauge metric
type Gauge interface {
	// Set sets the gauge to the given value
	Set(float64)

	// Inc increments the gauge by 1
	Inc()

	// Dec decrements the gauge by 1
	Dec()

	// Add adds the given value to the gauge
	Add(float64)

	// Sub subtracts the given value from the gauge
	Sub(float64)

	// WithLabels returns a gauge with the given label values
	WithLabels(labels map[string]string) Gauge
}

// Histogram represents a histogram metric
type Histogram interface {
	// Observe adds a single observation to the histogram
	Observe(float64)

	// WithLabels returns a histogram with the given label values
	WithLabels(labels map[string]string) Histogram
}

// Summary represents a summary metric
type Summary interface {
	// Observe adds a single observation to the summary
	Observe(float64)

	// WithLabels returns a summary with the given label values
	WithLabels(labels map[string]string) Summary
}

// Timer is a helper for measuring durations
type Timer struct {
	observer Histogram
	start    time.Time
}

// NewTimer creates a new timer that will observe duration to the given histogram
func NewTimer(h Histogram) *Timer {
	return &Timer{
		observer: h,
		start:    time.Now(),
	}
}

// ObserveDuration records the duration since the timer was created
func (t *Timer) ObserveDuration() time.Duration {
	d := time.Since(t.start)
	t.observer.Observe(d.Seconds())
	return d
}

// DefaultHistogramBuckets returns default bucket boundaries
func DefaultHistogramBuckets() []float64 {
	return []float64{.005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5, 10}
}

// HTTPHistogramBuckets returns bucket boundaries suitable for HTTP latencies
func HTTPHistogramBuckets() []float64 {
	return []float64{.001, .005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5, 10}
}

// BatchOptions configures batch metrics behavior
type BatchOptions struct {
	// MaxSize is the maximum number of metrics to batch before sending
	MaxSize int

	// FlushInterval is the maximum time between flushes
	FlushInterval time.Duration
}

// DefaultBatchOptions returns sensible default batch options
func DefaultBatchOptions() *BatchOptions {
	return &BatchOptions{
		MaxSize:       100,
		FlushInterval: 60 * time.Second,
	}
}
