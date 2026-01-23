package tracing

import (
	"time"
)

// Provider represents a tracing service provider
type Provider string

const (
	ProviderSentry Provider = "sentry"
	ProviderOTel   Provider = "otel"
	ProviderMock   Provider = "mock"
)

// SpanKind represents the type of span
type SpanKind int

const (
	SpanKindUnspecified SpanKind = iota
	SpanKindInternal
	SpanKindServer
	SpanKindClient
	SpanKindProducer
	SpanKindConsumer
)

// String returns the string representation of SpanKind
func (k SpanKind) String() string {
	switch k {
	case SpanKindInternal:
		return "internal"
	case SpanKindServer:
		return "server"
	case SpanKindClient:
		return "client"
	case SpanKindProducer:
		return "producer"
	case SpanKindConsumer:
		return "consumer"
	default:
		return "unspecified"
	}
}

// SpanStatus represents the status of a span
type SpanStatus int

const (
	SpanStatusUnset SpanStatus = iota
	SpanStatusOK
	SpanStatusError
)

// String returns the string representation of SpanStatus
func (s SpanStatus) String() string {
	switch s {
	case SpanStatusOK:
		return "ok"
	case SpanStatusError:
		return "error"
	default:
		return "unset"
	}
}

// SpanContext contains the identifiers for a span
type SpanContext struct {
	// TraceID is the unique identifier for the trace
	TraceID string

	// SpanID is the unique identifier for the span
	SpanID string

	// ParentSpanID is the span ID of the parent span
	ParentSpanID string

	// TraceFlags contains the trace flags
	TraceFlags byte

	// TraceState contains vendor-specific trace identification
	TraceState string

	// Remote indicates if the span context was received from a remote source
	Remote bool
}

// IsValid returns true if the span context is valid
func (sc SpanContext) IsValid() bool {
	return sc.TraceID != "" && sc.SpanID != ""
}

// SpanOptions contains options for starting a span
type SpanOptions struct {
	// Kind is the span kind
	Kind SpanKind

	// Attributes are initial attributes for the span
	Attributes map[string]any

	// Links are links to other spans
	Links []SpanLink

	// StartTime overrides the default start time
	StartTime time.Time
}

// SpanLink represents a link to another span
type SpanLink struct {
	// Context is the span context of the linked span
	Context SpanContext

	// Attributes are attributes for the link
	Attributes map[string]any
}

// Event represents a span event
type Event struct {
	// Name is the event name
	Name string

	// Timestamp is when the event occurred
	Timestamp time.Time

	// Attributes are event attributes
	Attributes map[string]any
}

// SamplerDecision represents a sampling decision
type SamplerDecision int

const (
	SamplerDecisionDrop SamplerDecision = iota
	SamplerDecisionRecordOnly
	SamplerDecisionRecordAndSample
)

// SamplerConfig configures sampling behavior
type SamplerConfig struct {
	// Type is the sampler type (always_on, always_off, trace_id_ratio, parent_based)
	Type string

	// Ratio is the sampling ratio for trace_id_ratio sampler (0.0 to 1.0)
	Ratio float64
}

// HTTPPropagator extracts and injects trace context from/to HTTP headers
type HTTPPropagator interface {
	// Extract extracts span context from HTTP headers
	Extract(headers map[string]string) SpanContext

	// Inject injects span context into HTTP headers
	Inject(ctx SpanContext, headers map[string]string)
}

// TracePropagator represents the propagation format
type TracePropagator string

const (
	PropagatorTraceContext TracePropagator = "tracecontext" // W3C Trace Context
	PropagatorB3           TracePropagator = "b3"           // Zipkin B3
	PropagatorB3Multi      TracePropagator = "b3multi"      // Zipkin B3 Multi
	PropagatorJaeger       TracePropagator = "jaeger"       // Jaeger
	PropagatorXRay         TracePropagator = "xray"         // AWS X-Ray
	PropagatorSentry       TracePropagator = "sentry"       // Sentry
)

// ExporterConfig configures trace exporting
type ExporterConfig struct {
	// Type is the exporter type (otlp, jaeger, zipkin, stdout)
	Type string

	// Endpoint is the exporter endpoint
	Endpoint string

	// Headers are additional headers for the exporter
	Headers map[string]string

	// Insecure disables TLS for the connection
	Insecure bool
}

// ResourceAttributes contains resource information
type ResourceAttributes struct {
	// ServiceName is the name of the service
	ServiceName string

	// ServiceVersion is the version of the service
	ServiceVersion string

	// Environment is the deployment environment
	Environment string

	// Additional contains additional resource attributes
	Additional map[string]any
}
