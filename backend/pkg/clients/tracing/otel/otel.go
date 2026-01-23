package otel

import (
	"context"
	"fmt"
	"net/http"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.21.0"
	"go.opentelemetry.io/otel/trace"

	"github.com/javaknight1/servicepro/backend/config"
	"github.com/javaknight1/servicepro/backend/pkg/clients/tracing"
)

func init() {
	tracing.RegisterProvider(tracing.ProviderOTel, func(ctx context.Context, cfg *config.Config) (tracing.Client, error) {
		otelCfg := &Config{
			ServiceName: "servicepro",
			Environment: cfg.Server.Env,
			Endpoint:    cfg.OpenTelemetry.Endpoint,
			Insecure:    cfg.OpenTelemetry.Insecure,
			SampleRate:  cfg.OpenTelemetry.SampleRate,
		}
		return NewClient(ctx, otelCfg, cfg)
	})
}

// Config holds configuration for the OpenTelemetry tracing client
type Config struct {
	// ServiceName is the name of the service
	ServiceName string

	// ServiceVersion is the version of the service
	ServiceVersion string

	// Environment is the deployment environment
	Environment string

	// Endpoint is the OTLP endpoint (e.g., "localhost:4318")
	Endpoint string

	// Headers are additional headers for the OTLP exporter
	Headers map[string]string

	// Insecure disables TLS for the connection
	Insecure bool

	// SampleRate is the sampling rate (0.0 to 1.0)
	SampleRate float64
}

// Client implements the tracing.Client interface using OpenTelemetry
type Client struct {
	config         *Config
	appConfig      *config.Config
	tracerProvider *sdktrace.TracerProvider
	tracer         trace.Tracer
	propagator     propagation.TextMapPropagator
}

// NewClient creates a new OpenTelemetry tracing client
func NewClient(ctx context.Context, cfg *Config, appCfg *config.Config) (*Client, error) {
	if cfg.ServiceName == "" {
		cfg.ServiceName = "servicepro"
	}

	// Create resource with service information
	res, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceName(cfg.ServiceName),
			semconv.ServiceVersion(cfg.ServiceVersion),
			semconv.DeploymentEnvironment(cfg.Environment),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create resource: %w", err)
	}

	// Create OTLP exporter options
	var exporterOpts []otlptracehttp.Option
	if cfg.Endpoint != "" {
		exporterOpts = append(exporterOpts, otlptracehttp.WithEndpoint(cfg.Endpoint))
	}
	if cfg.Insecure {
		exporterOpts = append(exporterOpts, otlptracehttp.WithInsecure())
	}
	if len(cfg.Headers) > 0 {
		exporterOpts = append(exporterOpts, otlptracehttp.WithHeaders(cfg.Headers))
	}

	// Create exporter
	exporter, err := otlptracehttp.New(ctx, exporterOpts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create OTLP exporter: %w", err)
	}

	// Configure sampler
	var sampler sdktrace.Sampler
	if cfg.SampleRate > 0 && cfg.SampleRate < 1 {
		sampler = sdktrace.TraceIDRatioBased(cfg.SampleRate)
	} else if cfg.SampleRate >= 1 {
		sampler = sdktrace.AlwaysSample()
	} else {
		sampler = sdktrace.ParentBased(sdktrace.TraceIDRatioBased(0.1)) // Default 10%
	}

	// Create tracer provider
	tracerProvider := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
		sdktrace.WithSampler(sampler),
	)

	// Set global tracer provider
	otel.SetTracerProvider(tracerProvider)

	// Set up propagators (W3C Trace Context and Baggage)
	propagator := propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	)
	otel.SetTextMapPropagator(propagator)

	tracer := tracerProvider.Tracer(cfg.ServiceName)

	return &Client{
		config:         cfg,
		appConfig:      appCfg,
		tracerProvider: tracerProvider,
		tracer:         tracer,
		propagator:     propagator,
	}, nil
}

// StartSpan implements tracing.Client
func (c *Client) StartSpan(ctx context.Context, name string, opts ...tracing.SpanOption) (context.Context, tracing.Span) {
	options := &tracing.SpanOptions{
		Attributes: make(map[string]any),
	}
	for _, opt := range opts {
		opt(options)
	}

	// Build OTel span options
	var otelOpts []trace.SpanStartOption

	// Set span kind
	switch options.Kind {
	case tracing.SpanKindServer:
		otelOpts = append(otelOpts, trace.WithSpanKind(trace.SpanKindServer))
	case tracing.SpanKindClient:
		otelOpts = append(otelOpts, trace.WithSpanKind(trace.SpanKindClient))
	case tracing.SpanKindProducer:
		otelOpts = append(otelOpts, trace.WithSpanKind(trace.SpanKindProducer))
	case tracing.SpanKindConsumer:
		otelOpts = append(otelOpts, trace.WithSpanKind(trace.SpanKindConsumer))
	case tracing.SpanKindInternal:
		otelOpts = append(otelOpts, trace.WithSpanKind(trace.SpanKindInternal))
	}

	// Set attributes
	if len(options.Attributes) > 0 {
		attrs := make([]attribute.KeyValue, 0, len(options.Attributes))
		for k, v := range options.Attributes {
			attrs = append(attrs, toAttribute(k, v))
		}
		otelOpts = append(otelOpts, trace.WithAttributes(attrs...))
	}

	// Set start time
	if !options.StartTime.IsZero() {
		otelOpts = append(otelOpts, trace.WithTimestamp(options.StartTime))
	}

	// Set links
	if len(options.Links) > 0 {
		links := make([]trace.Link, len(options.Links))
		for i, link := range options.Links {
			sc := toOTelSpanContext(link.Context)
			var attrs []attribute.KeyValue
			for k, v := range link.Attributes {
				attrs = append(attrs, toAttribute(k, v))
			}
			links[i] = trace.Link{
				SpanContext: sc,
				Attributes:  attrs,
			}
		}
		otelOpts = append(otelOpts, trace.WithLinks(links...))
	}

	ctx, otelSpan := c.tracer.Start(ctx, name, otelOpts...)

	return ctx, &otelSpanWrapper{
		span: otelSpan,
	}
}

// SpanFromContext implements tracing.Client
func (c *Client) SpanFromContext(ctx context.Context) tracing.Span {
	otelSpan := trace.SpanFromContext(ctx)
	if otelSpan == nil || !otelSpan.SpanContext().IsValid() {
		return nil
	}
	return &otelSpanWrapper{span: otelSpan}
}

// ContextWithSpan implements tracing.Client
func (c *Client) ContextWithSpan(ctx context.Context, span tracing.Span) context.Context {
	if wrapper, ok := span.(*otelSpanWrapper); ok {
		return trace.ContextWithSpan(ctx, wrapper.span)
	}
	return ctx
}

// HTTPMiddleware implements tracing.Client
func (c *Client) HTTPMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Extract trace context from incoming request
		ctx := c.propagator.Extract(r.Context(), propagation.HeaderCarrier(r.Header))

		// Start server span
		ctx, span := c.tracer.Start(ctx, fmt.Sprintf("%s %s", r.Method, r.URL.Path),
			trace.WithSpanKind(trace.SpanKindServer),
			trace.WithAttributes(
				semconv.HTTPRequestMethodKey.String(r.Method),
				semconv.URLFull(r.URL.String()),
				semconv.UserAgentOriginal(r.UserAgent()),
				attribute.String("net.peer.ip", r.RemoteAddr),
			),
		)
		defer span.End()

		// Wrap response writer
		wrapped := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

		next.ServeHTTP(wrapped, r.WithContext(ctx))

		span.SetAttributes(semconv.HTTPResponseStatusCode(wrapped.statusCode))
		if wrapped.statusCode >= 400 {
			span.SetStatus(codes.Error, fmt.Sprintf("HTTP %d", wrapped.statusCode))
		} else {
			span.SetStatus(codes.Ok, "")
		}
	})
}

// Extract implements tracing.Client
func (c *Client) Extract(ctx context.Context, carrier tracing.Carrier) context.Context {
	return c.propagator.Extract(ctx, &carrierAdapter{carrier: carrier})
}

// Inject implements tracing.Client
func (c *Client) Inject(ctx context.Context, carrier tracing.Carrier) context.Context {
	c.propagator.Inject(ctx, &carrierAdapter{carrier: carrier})
	return ctx
}

// Flush implements tracing.Client
func (c *Client) Flush() error {
	return c.tracerProvider.ForceFlush(context.Background())
}

// Close implements tracing.Client
func (c *Client) Close() error {
	return c.tracerProvider.Shutdown(context.Background())
}

// otelSpanWrapper wraps an OTel span to implement tracing.Span
type otelSpanWrapper struct {
	span trace.Span
}

func (s *otelSpanWrapper) End() {
	s.span.End()
}

func (s *otelSpanWrapper) EndWithOptions(opts ...tracing.SpanEndOption) {
	s.span.End()
}

func (s *otelSpanWrapper) SetName(name string) {
	s.span.SetName(name)
}

func (s *otelSpanWrapper) SetStatus(status tracing.SpanStatus, description string) {
	switch status {
	case tracing.SpanStatusOK:
		s.span.SetStatus(codes.Ok, description)
	case tracing.SpanStatusError:
		s.span.SetStatus(codes.Error, description)
	default:
		s.span.SetStatus(codes.Unset, description)
	}
}

func (s *otelSpanWrapper) SetAttributes(attrs map[string]any) {
	otelAttrs := make([]attribute.KeyValue, 0, len(attrs))
	for k, v := range attrs {
		otelAttrs = append(otelAttrs, toAttribute(k, v))
	}
	s.span.SetAttributes(otelAttrs...)
}

func (s *otelSpanWrapper) SetAttribute(key string, value any) {
	s.span.SetAttributes(toAttribute(key, value))
}

func (s *otelSpanWrapper) AddEvent(name string, attrs map[string]any) {
	var otelAttrs []attribute.KeyValue
	for k, v := range attrs {
		otelAttrs = append(otelAttrs, toAttribute(k, v))
	}
	s.span.AddEvent(name, trace.WithAttributes(otelAttrs...))
}

func (s *otelSpanWrapper) RecordError(err error) {
	if err == nil {
		return
	}
	s.span.RecordError(err)
	s.span.SetStatus(codes.Error, err.Error())
}

func (s *otelSpanWrapper) SpanContext() tracing.SpanContext {
	sc := s.span.SpanContext()
	return tracing.SpanContext{
		TraceID:    sc.TraceID().String(),
		SpanID:     sc.SpanID().String(),
		TraceFlags: byte(sc.TraceFlags()),
		TraceState: sc.TraceState().String(),
		Remote:     sc.IsRemote(),
	}
}

func (s *otelSpanWrapper) IsRecording() bool {
	return s.span.IsRecording()
}

// carrierAdapter adapts tracing.Carrier to propagation.TextMapCarrier
type carrierAdapter struct {
	carrier tracing.Carrier
}

func (a *carrierAdapter) Get(key string) string {
	return a.carrier.Get(key)
}

func (a *carrierAdapter) Set(key, value string) {
	a.carrier.Set(key, value)
}

func (a *carrierAdapter) Keys() []string {
	return a.carrier.Keys()
}

// responseWriter wraps http.ResponseWriter to capture status code
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

// toAttribute converts a Go value to an OTel attribute
func toAttribute(key string, value any) attribute.KeyValue {
	switch v := value.(type) {
	case string:
		return attribute.String(key, v)
	case int:
		return attribute.Int(key, v)
	case int64:
		return attribute.Int64(key, v)
	case float64:
		return attribute.Float64(key, v)
	case bool:
		return attribute.Bool(key, v)
	case []string:
		return attribute.StringSlice(key, v)
	case []int:
		return attribute.IntSlice(key, v)
	case []int64:
		return attribute.Int64Slice(key, v)
	case []float64:
		return attribute.Float64Slice(key, v)
	case []bool:
		return attribute.BoolSlice(key, v)
	default:
		return attribute.String(key, fmt.Sprintf("%v", v))
	}
}

// toOTelSpanContext converts our SpanContext to OTel SpanContext
func toOTelSpanContext(sc tracing.SpanContext) trace.SpanContext {
	var traceID trace.TraceID
	var spanID trace.SpanID

	if len(sc.TraceID) == 32 {
		copy(traceID[:], []byte(sc.TraceID))
	}
	if len(sc.SpanID) == 16 {
		copy(spanID[:], []byte(sc.SpanID))
	}

	return trace.NewSpanContext(trace.SpanContextConfig{
		TraceID:    traceID,
		SpanID:     spanID,
		TraceFlags: trace.TraceFlags(sc.TraceFlags),
		Remote:     sc.Remote,
	})
}

// Ensure Client implements tracing.Client
var _ tracing.Client = (*Client)(nil)

// Ensure otelSpanWrapper implements tracing.Span
var _ tracing.Span = (*otelSpanWrapper)(nil)
