package mock

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/javaknight1/servicepro/backend/config"
	"github.com/javaknight1/servicepro/backend/pkg/clients/logging"
	"github.com/javaknight1/servicepro/backend/pkg/clients/tracing"
)

func init() {
	tracing.RegisterProvider(tracing.ProviderMock, func(ctx context.Context, cfg *config.Config) (tracing.Client, error) {
		mockCfg := &Config{
			PrintToStdout: true,
			StoreSpans:    false,
		}
		return NewClient(mockCfg, cfg), nil
	})
}

// Config holds configuration for the mock tracing client
type Config struct {
	// PrintToStdout prints spans to stdout (default: true)
	PrintToStdout bool

	// StoreSpans stores spans in memory for testing (default: false)
	StoreSpans bool
}

// Client implements the tracing.Client interface for testing and development
type Client struct {
	config      *Config
	appConfig   *config.Config
	mu          sync.RWMutex
	spans       []*SpanData
	serviceName string
	environment string
}

// SpanData represents span data stored for testing
type SpanData struct {
	Name       string
	Context    tracing.SpanContext
	Kind       tracing.SpanKind
	StartTime  time.Time
	EndTime    time.Time
	Status     tracing.SpanStatus
	StatusDesc string
	Attributes map[string]any
	Events     []tracing.Event
	Errors     []error
}

// NewClient creates a new mock tracing client
func NewClient(cfg *Config, appCfg *config.Config) *Client {
	if cfg == nil {
		cfg = &Config{
			PrintToStdout: true,
			StoreSpans:    false,
		}
	}

	c := &Client{
		config:    cfg,
		appConfig: appCfg,
		spans:     make([]*SpanData, 0),
	}

	if appCfg != nil {
		c.serviceName = "servicepro"
		c.environment = appCfg.Server.Env
	}

	return c
}

// StartSpan implements tracing.Client
func (c *Client) StartSpan(ctx context.Context, name string, opts ...tracing.SpanOption) (context.Context, tracing.Span) {
	options := &tracing.SpanOptions{
		Attributes: make(map[string]any),
	}
	for _, opt := range opts {
		opt(options)
	}

	// Generate span context
	spanCtx := tracing.SpanContext{
		TraceID: generateTraceID(),
		SpanID:  generateSpanID(),
	}

	// Inherit trace ID from parent if exists
	if parentSpan := c.SpanFromContext(ctx); parentSpan != nil {
		parentCtx := parentSpan.SpanContext()
		spanCtx.TraceID = parentCtx.TraceID
		spanCtx.ParentSpanID = parentCtx.SpanID
	}

	startTime := options.StartTime
	if startTime.IsZero() {
		startTime = time.Now()
	}

	span := &mockSpan{
		client:     c,
		name:       name,
		context:    spanCtx,
		kind:       options.Kind,
		startTime:  startTime,
		attributes: options.Attributes,
		events:     make([]tracing.Event, 0),
		errors:     make([]error, 0),
		recording:  true,
	}

	if c.config.PrintToStdout {
		logging.Info(ctx, "[TRACING-MOCK] Starting span", map[string]any{"name": name, "trace_id": spanCtx.TraceID, "span_id": spanCtx.SpanID, "parent_span_id": spanCtx.ParentSpanID})
	}

	return c.ContextWithSpan(ctx, span), span
}

// SpanFromContext implements tracing.Client
func (c *Client) SpanFromContext(ctx context.Context) tracing.Span {
	if span, ok := ctx.Value(tracing.ContextKeySpan).(tracing.Span); ok {
		return span
	}
	return nil
}

// ContextWithSpan implements tracing.Client
func (c *Client) ContextWithSpan(ctx context.Context, span tracing.Span) context.Context {
	return context.WithValue(ctx, tracing.ContextKeySpan, span)
}

// HTTPMiddleware implements tracing.Client
func (c *Client) HTTPMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Extract trace context from incoming headers
		ctx := c.Extract(r.Context(), tracing.NewHTTPCarrier(r.Header))

		// Start a server span
		ctx, span := c.StartSpan(ctx, fmt.Sprintf("%s %s", r.Method, r.URL.Path),
			tracing.WithSpanKind(tracing.SpanKindServer),
			tracing.WithAttributes(map[string]any{
				"http.method":      r.Method,
				"http.url":         r.URL.String(),
				"http.user_agent":  r.UserAgent(),
				"http.remote_addr": r.RemoteAddr,
			}),
		)
		defer span.End()

		// Wrap response writer to capture status code
		wrapped := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

		next.ServeHTTP(wrapped, r.WithContext(ctx))

		span.SetAttribute("http.status_code", wrapped.statusCode)
		if wrapped.statusCode >= 400 {
			span.SetStatus(tracing.SpanStatusError, fmt.Sprintf("HTTP %d", wrapped.statusCode))
		} else {
			span.SetStatus(tracing.SpanStatusOK, "")
		}
	})
}

// Extract implements tracing.Client
func (c *Client) Extract(ctx context.Context, carrier tracing.Carrier) context.Context {
	// Extract trace context from W3C traceparent header
	traceparent := carrier.Get("traceparent")
	if traceparent == "" {
		return ctx
	}

	// Parse traceparent: version-trace_id-span_id-trace_flags
	// This is a simplified parser
	spanCtx := tracing.SpanContext{
		TraceID: traceparent, // In a real impl, parse properly
		Remote:  true,
	}

	return context.WithValue(ctx, tracing.ContextKeySpan, &mockSpan{
		client:    c,
		context:   spanCtx,
		recording: false, // Remote spans are not recording
	})
}

// Inject implements tracing.Client
func (c *Client) Inject(ctx context.Context, carrier tracing.Carrier) context.Context {
	span := c.SpanFromContext(ctx)
	if span == nil {
		return ctx
	}

	spanCtx := span.SpanContext()
	if !spanCtx.IsValid() {
		return ctx
	}

	// Inject W3C traceparent header
	traceparent := fmt.Sprintf("00-%s-%s-01", spanCtx.TraceID, spanCtx.SpanID)
	carrier.Set("traceparent", traceparent)

	return ctx
}

// Flush implements tracing.Client
func (c *Client) Flush() error {
	return nil
}

// Close implements tracing.Client
func (c *Client) Close() error {
	return nil
}

func (c *Client) storeSpan(span *SpanData) {
	if !c.config.StoreSpans {
		return
	}
	c.mu.Lock()
	c.spans = append(c.spans, span)
	c.mu.Unlock()
}

// GetSpans returns all stored spans (for testing)
func (c *Client) GetSpans() []*SpanData {
	c.mu.RLock()
	defer c.mu.RUnlock()

	result := make([]*SpanData, len(c.spans))
	copy(result, c.spans)
	return result
}

// ClearSpans clears all stored spans (for testing)
func (c *Client) ClearSpans() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.spans = make([]*SpanData, 0)
}

// mockSpan implements tracing.Span
type mockSpan struct {
	client     *Client
	name       string
	context    tracing.SpanContext
	kind       tracing.SpanKind
	startTime  time.Time
	endTime    time.Time
	status     tracing.SpanStatus
	statusDesc string
	attributes map[string]any
	events     []tracing.Event
	errors     []error
	recording  bool
	mu         sync.Mutex
}

func (s *mockSpan) End() {
	s.EndWithOptions()
}

func (s *mockSpan) EndWithOptions(opts ...tracing.SpanEndOption) {
	s.mu.Lock()
	s.endTime = time.Now()
	s.mu.Unlock()

	if s.client.config.PrintToStdout {
		duration := s.endTime.Sub(s.startTime)
		logging.Info(context.Background(), "[TRACING-MOCK] Ending span", map[string]any{"name": s.name, "trace_id": s.context.TraceID, "span_id": s.context.SpanID, "duration": duration.String(), "status": s.status.String()})
	}

	if s.client.config.StoreSpans {
		s.client.storeSpan(&SpanData{
			Name:       s.name,
			Context:    s.context,
			Kind:       s.kind,
			StartTime:  s.startTime,
			EndTime:    s.endTime,
			Status:     s.status,
			StatusDesc: s.statusDesc,
			Attributes: s.attributes,
			Events:     s.events,
			Errors:     s.errors,
		})
	}
}

func (s *mockSpan) SetName(name string) {
	s.mu.Lock()
	s.name = name
	s.mu.Unlock()
}

func (s *mockSpan) SetStatus(status tracing.SpanStatus, description string) {
	s.mu.Lock()
	s.status = status
	s.statusDesc = description
	s.mu.Unlock()
}

func (s *mockSpan) SetAttributes(attrs map[string]any) {
	s.mu.Lock()
	for k, v := range attrs {
		s.attributes[k] = v
	}
	s.mu.Unlock()
}

func (s *mockSpan) SetAttribute(key string, value any) {
	s.mu.Lock()
	s.attributes[key] = value
	s.mu.Unlock()
}

func (s *mockSpan) AddEvent(name string, attrs map[string]any) {
	s.mu.Lock()
	s.events = append(s.events, tracing.Event{
		Name:       name,
		Timestamp:  time.Now(),
		Attributes: attrs,
	})
	s.mu.Unlock()

	if s.client.config.PrintToStdout {
		logging.Info(context.Background(), "[TRACING-MOCK] Event", map[string]any{"event_name": name, "span_name": s.name, "attrs": attrs})
	}
}

func (s *mockSpan) RecordError(err error) {
	if err == nil {
		return
	}

	s.mu.Lock()
	s.errors = append(s.errors, err)
	s.status = tracing.SpanStatusError
	s.statusDesc = err.Error()
	s.mu.Unlock()

	if s.client.config.PrintToStdout {
		logging.Error(context.Background(), "[TRACING-MOCK] Error in span", map[string]any{"span_name": s.name, "error": err})
	}
}

func (s *mockSpan) SpanContext() tracing.SpanContext {
	return s.context
}

func (s *mockSpan) IsRecording() bool {
	return s.recording
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

// generateTraceID generates a random 32-character hex trace ID
func generateTraceID() string {
	b := make([]byte, 16)
	rand.Read(b)
	return hex.EncodeToString(b)
}

// generateSpanID generates a random 16-character hex span ID
func generateSpanID() string {
	b := make([]byte, 8)
	rand.Read(b)
	return hex.EncodeToString(b)
}

// Ensure Client implements tracing.Client
var _ tracing.Client = (*Client)(nil)

// Ensure mockSpan implements tracing.Span
var _ tracing.Span = (*mockSpan)(nil)
