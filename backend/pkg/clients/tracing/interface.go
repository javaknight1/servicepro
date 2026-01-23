package tracing

import (
	"context"
	"net/http"
)

// Client defines the interface for distributed tracing operations.
// Implementations can use various backends like OpenTelemetry, Sentry, etc.
type Client interface {
	// StartSpan starts a new span with the given name
	StartSpan(ctx context.Context, name string, opts ...SpanOption) (context.Context, Span)

	// SpanFromContext returns the current span from context, or nil if none
	SpanFromContext(ctx context.Context) Span

	// ContextWithSpan returns a new context with the given span
	ContextWithSpan(ctx context.Context, span Span) context.Context

	// HTTPMiddleware returns middleware for tracing HTTP requests
	HTTPMiddleware(next http.Handler) http.Handler

	// Extract extracts span context from carrier (e.g., HTTP headers)
	Extract(ctx context.Context, carrier Carrier) context.Context

	// Inject injects span context into carrier (e.g., HTTP headers)
	Inject(ctx context.Context, carrier Carrier) context.Context

	// Flush forces any buffered traces to be exported
	Flush() error

	// Close releases resources and flushes remaining traces
	Close() error
}

// Span represents a single operation within a trace
type Span interface {
	// End completes the span
	End()

	// EndWithOptions completes the span with options
	EndWithOptions(opts ...SpanEndOption)

	// SetName sets the span name
	SetName(name string)

	// SetStatus sets the span status
	SetStatus(status SpanStatus, description string)

	// SetAttributes sets multiple attributes on the span
	SetAttributes(attrs map[string]any)

	// SetAttribute sets a single attribute on the span
	SetAttribute(key string, value any)

	// AddEvent adds an event to the span
	AddEvent(name string, attrs map[string]any)

	// RecordError records an error on the span
	RecordError(err error)

	// SpanContext returns the span context
	SpanContext() SpanContext

	// IsRecording returns true if the span is recording
	IsRecording() bool
}

// SpanOption is a function that configures span options
type SpanOption func(*SpanOptions)

// WithSpanKind sets the span kind
func WithSpanKind(kind SpanKind) SpanOption {
	return func(opts *SpanOptions) {
		opts.Kind = kind
	}
}

// WithAttributes sets initial span attributes
func WithAttributes(attrs map[string]any) SpanOption {
	return func(opts *SpanOptions) {
		opts.Attributes = attrs
	}
}

// WithLinks adds links to the span
func WithLinks(links ...SpanLink) SpanOption {
	return func(opts *SpanOptions) {
		opts.Links = append(opts.Links, links...)
	}
}

// SpanEndOption is a function that configures span end options
type SpanEndOption func(*spanEndOptions)

type spanEndOptions struct {
	// can add options like EndTime if needed
}

// Carrier is the interface for trace context propagation
type Carrier interface {
	// Get returns the value for a key
	Get(key string) string

	// Set sets a key-value pair
	Set(key, value string)

	// Keys returns all keys
	Keys() []string
}

// HTTPCarrier implements Carrier for HTTP headers
type HTTPCarrier struct {
	Headers http.Header
}

// NewHTTPCarrier creates a new HTTPCarrier from HTTP headers
func NewHTTPCarrier(headers http.Header) *HTTPCarrier {
	return &HTTPCarrier{Headers: headers}
}

// Get returns the value for a key
func (c *HTTPCarrier) Get(key string) string {
	return c.Headers.Get(key)
}

// Set sets a key-value pair
func (c *HTTPCarrier) Set(key, value string) {
	c.Headers.Set(key, value)
}

// Keys returns all keys
func (c *HTTPCarrier) Keys() []string {
	keys := make([]string, 0, len(c.Headers))
	for k := range c.Headers {
		keys = append(keys, k)
	}
	return keys
}

// MapCarrier implements Carrier for a map
type MapCarrier map[string]string

// Get returns the value for a key
func (c MapCarrier) Get(key string) string {
	return c[key]
}

// Set sets a key-value pair
func (c MapCarrier) Set(key, value string) {
	c[key] = value
}

// Keys returns all keys
func (c MapCarrier) Keys() []string {
	keys := make([]string, 0, len(c))
	for k := range c {
		keys = append(keys, k)
	}
	return keys
}

// ContextKey is the type for context keys used by the tracing package
type ContextKey string

const (
	// ContextKeySpan is the context key for the current span
	ContextKeySpan ContextKey = "tracing_span"
)
