package metrics

import (
	"context"
	"net/http"
)

// Client defines the interface for metrics operations.
// Implementations can send metrics to various backends like CloudWatch,
// Prometheus, or stdout for testing.
type Client interface {
	// Counter creates or retrieves a counter metric
	Counter(opts CounterOptions) Counter

	// Gauge creates or retrieves a gauge metric
	Gauge(opts GaugeOptions) Gauge

	// Histogram creates or retrieves a histogram metric
	Histogram(opts HistogramOptions) Histogram

	// Summary creates or retrieves a summary metric
	Summary(opts SummaryOptions) Summary

	// Record records a single metric data point (for push-based systems)
	Record(ctx context.Context, data *MetricData) error

	// RecordBatch records multiple metric data points
	RecordBatch(ctx context.Context, data []*MetricData) error

	// Handler returns an HTTP handler for metrics exposition (for pull-based systems like Prometheus)
	Handler() http.Handler

	// Flush forces any buffered metrics to be sent
	Flush() error

	// Close releases resources and flushes remaining metrics
	Close() error
}

// ContextKey is the type for context keys used by the metrics package
type ContextKey string

const (
	// ContextKeyRequestStart is the context key for request start time
	ContextKeyRequestStart ContextKey = "metrics_request_start"
)
