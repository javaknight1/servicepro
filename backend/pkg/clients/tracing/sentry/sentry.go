package sentry

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/getsentry/sentry-go"

	"github.com/javaknight1/servicepro/backend/config"
	"github.com/javaknight1/servicepro/backend/pkg/clients/tracing"
)

func init() {
	tracing.RegisterProvider(tracing.ProviderSentry, func(ctx context.Context, cfg *config.Config) (tracing.Client, error) {
		sentryCfg := &Config{
			DSN:              cfg.Sentry.DSN,
			Environment:      cfg.Server.Env,
			Release:          cfg.Sentry.Release,
			TracesSampleRate: cfg.Sentry.TracesSampleRate,
			Debug:            cfg.Sentry.Debug,
		}
		return NewClient(sentryCfg, cfg)
	})
}

// Config holds configuration for the Sentry tracing client
type Config struct {
	// DSN is the Sentry DSN (required)
	DSN string

	// Environment is the deployment environment
	Environment string

	// Release is the release version
	Release string

	// TracesSampleRate is the sample rate for tracing (0.0 to 1.0)
	TracesSampleRate float64

	// Debug enables debug mode
	Debug bool
}

// Client implements the tracing.Client interface using Sentry
type Client struct {
	config      *Config
	appConfig   *config.Config
	initialized bool
}

// NewClient creates a new Sentry tracing client
func NewClient(cfg *Config, appCfg *config.Config) (*Client, error) {
	if cfg.DSN == "" {
		return nil, fmt.Errorf("Sentry DSN is required")
	}

	sampleRate := cfg.TracesSampleRate
	if sampleRate <= 0 {
		sampleRate = 0.1 // Default 10% sampling
	}

	err := sentry.Init(sentry.ClientOptions{
		Dsn:              cfg.DSN,
		Environment:      cfg.Environment,
		Release:          cfg.Release,
		TracesSampleRate: sampleRate,
		Debug:            cfg.Debug,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to initialize Sentry: %w", err)
	}

	return &Client{
		config:      cfg,
		appConfig:   appCfg,
		initialized: true,
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

	// Get or create hub from context
	hub := sentry.GetHubFromContext(ctx)
	if hub == nil {
		hub = sentry.CurrentHub().Clone()
		ctx = sentry.SetHubOnContext(ctx, hub)
	}

	// Determine operation based on span kind
	op := name
	switch options.Kind {
	case tracing.SpanKindServer:
		op = "http.server"
	case tracing.SpanKindClient:
		op = "http.client"
	}

	var sentrySpan *sentry.Span

	// Check if there's a parent span
	if parentSpan := sentry.SpanFromContext(ctx); parentSpan != nil {
		sentrySpan = parentSpan.StartChild(op)
		sentrySpan.Description = name
	} else {
		// Start a new transaction
		sentrySpan = sentry.StartTransaction(ctx, name, sentry.WithOpName(op))
	}

	// Set initial attributes
	for k, v := range options.Attributes {
		sentrySpan.SetData(k, v)
	}

	span := &sentrySpanWrapper{
		client: c,
		span:   sentrySpan,
		name:   name,
	}

	return sentrySpan.Context(), span
}

// SpanFromContext implements tracing.Client
func (c *Client) SpanFromContext(ctx context.Context) tracing.Span {
	sentrySpan := sentry.SpanFromContext(ctx)
	if sentrySpan == nil {
		return nil
	}
	return &sentrySpanWrapper{
		client: c,
		span:   sentrySpan,
		name:   sentrySpan.Description,
	}
}

// ContextWithSpan implements tracing.Client
func (c *Client) ContextWithSpan(ctx context.Context, span tracing.Span) context.Context {
	if wrapper, ok := span.(*sentrySpanWrapper); ok {
		return wrapper.span.Context()
	}
	return ctx
}

// HTTPMiddleware implements tracing.Client
func (c *Client) HTTPMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hub := sentry.CurrentHub().Clone()
		ctx := sentry.SetHubOnContext(r.Context(), hub)

		// Extract trace context from incoming headers
		ctx = c.Extract(ctx, tracing.NewHTTPCarrier(r.Header))

		// Start transaction
		transaction := sentry.StartTransaction(ctx, fmt.Sprintf("%s %s", r.Method, r.URL.Path),
			sentry.WithOpName("http.server"),
		)
		defer transaction.Finish()

		transaction.SetData("http.method", r.Method)
		transaction.SetData("http.url", r.URL.String())
		transaction.SetData("http.user_agent", r.UserAgent())

		// Wrap response writer
		wrapped := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

		next.ServeHTTP(wrapped, r.WithContext(transaction.Context()))

		transaction.SetData("http.status_code", wrapped.statusCode)
		if wrapped.statusCode >= 400 {
			transaction.Status = sentry.SpanStatusInternalError
		} else {
			transaction.Status = sentry.SpanStatusOK
		}
	})
}

// Extract implements tracing.Client
func (c *Client) Extract(ctx context.Context, carrier tracing.Carrier) context.Context {
	// Sentry uses sentry-trace header
	sentryTrace := carrier.Get("sentry-trace")
	baggage := carrier.Get("baggage")

	if sentryTrace != "" {
		hub := sentry.GetHubFromContext(ctx)
		if hub == nil {
			hub = sentry.CurrentHub().Clone()
			ctx = sentry.SetHubOnContext(ctx, hub)
		}

		// Continue trace from incoming request
		_ = sentry.ContinueFromHeaders(sentryTrace, baggage)
	}

	return ctx
}

// Inject implements tracing.Client
func (c *Client) Inject(ctx context.Context, carrier tracing.Carrier) context.Context {
	span := sentry.SpanFromContext(ctx)
	if span == nil {
		return ctx
	}

	carrier.Set("sentry-trace", span.ToSentryTrace())
	carrier.Set("baggage", span.ToBaggage())

	return ctx
}

// Flush implements tracing.Client
func (c *Client) Flush() error {
	sentry.Flush(5 * time.Second)
	return nil
}

// Close implements tracing.Client
func (c *Client) Close() error {
	sentry.Flush(5 * time.Second)
	return nil
}

// sentrySpanWrapper wraps a Sentry span to implement tracing.Span
type sentrySpanWrapper struct {
	client *Client
	span   *sentry.Span
	name   string
	mu     sync.Mutex
}

func (s *sentrySpanWrapper) End() {
	s.span.Finish()
}

func (s *sentrySpanWrapper) EndWithOptions(opts ...tracing.SpanEndOption) {
	s.span.Finish()
}

func (s *sentrySpanWrapper) SetName(name string) {
	s.mu.Lock()
	s.name = name
	s.span.Description = name
	s.mu.Unlock()
}

func (s *sentrySpanWrapper) SetStatus(status tracing.SpanStatus, description string) {
	switch status {
	case tracing.SpanStatusOK:
		s.span.Status = sentry.SpanStatusOK
	case tracing.SpanStatusError:
		s.span.Status = sentry.SpanStatusInternalError
	default:
		s.span.Status = sentry.SpanStatusUndefined
	}
}

func (s *sentrySpanWrapper) SetAttributes(attrs map[string]any) {
	for k, v := range attrs {
		s.span.SetData(k, v)
	}
}

func (s *sentrySpanWrapper) SetAttribute(key string, value any) {
	s.span.SetData(key, value)
}

func (s *sentrySpanWrapper) AddEvent(name string, attrs map[string]any) {
	// Sentry doesn't have a direct equivalent to span events
	// We can add breadcrumbs instead
	hub := sentry.GetHubFromContext(s.span.Context())
	if hub != nil {
		hub.AddBreadcrumb(&sentry.Breadcrumb{
			Type:      "default",
			Category:  "span_event",
			Message:   name,
			Data:      attrs,
			Timestamp: time.Now(),
		}, nil)
	}
}

func (s *sentrySpanWrapper) RecordError(err error) {
	if err == nil {
		return
	}

	s.span.Status = sentry.SpanStatusInternalError
	s.span.SetData("error", err.Error())

	// Capture error to Sentry
	hub := sentry.GetHubFromContext(s.span.Context())
	if hub != nil {
		hub.CaptureException(err)
	}
}

func (s *sentrySpanWrapper) SpanContext() tracing.SpanContext {
	return tracing.SpanContext{
		TraceID: string(s.span.TraceID[:]),
		SpanID:  string(s.span.SpanID[:]),
	}
}

func (s *sentrySpanWrapper) IsRecording() bool {
	return s.span.Sampled == sentry.SampledTrue
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

// Ensure Client implements tracing.Client
var _ tracing.Client = (*Client)(nil)

// Ensure sentrySpanWrapper implements tracing.Span
var _ tracing.Span = (*sentrySpanWrapper)(nil)
