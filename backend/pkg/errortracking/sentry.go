package errortracking

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"runtime"
	"time"

	"github.com/getsentry/sentry-go"
)

// Config holds the configuration for error tracking
type Config struct {
	DSN              string
	Environment      string
	Release          string
	SampleRate       float64
	TracesSampleRate float64
	Debug            bool
	ServerName       string
	BeforeSend       func(event *sentry.Event, hint *sentry.EventHint) *sentry.Event
}

// DefaultConfig returns a default configuration
func DefaultConfig() Config {
	return Config{
		DSN:              os.Getenv("SENTRY_DSN"),
		Environment:      getEnvOrDefault("ENVIRONMENT", "development"),
		Release:          getEnvOrDefault("APP_VERSION", "unknown"),
		SampleRate:       1.0,
		TracesSampleRate: 0.1,
		Debug:            os.Getenv("SENTRY_DEBUG") == "true",
		ServerName:       getHostname(),
	}
}

// Init initializes the Sentry SDK with the provided configuration
func Init(cfg Config) error {
	if cfg.DSN == "" {
		return fmt.Errorf("sentry DSN is required")
	}

	err := sentry.Init(sentry.ClientOptions{
		Dsn:              cfg.DSN,
		Environment:      cfg.Environment,
		Release:          cfg.Release,
		SampleRate:       cfg.SampleRate,
		TracesSampleRate: cfg.TracesSampleRate,
		Debug:            cfg.Debug,
		ServerName:       cfg.ServerName,
		BeforeSend:       cfg.BeforeSend,
		AttachStacktrace: true,
		EnableTracing:    true,
		Integrations: func(integrations []sentry.Integration) []sentry.Integration {
			// Filter out integrations if needed
			return integrations
		},
	})
	if err != nil {
		return fmt.Errorf("failed to initialize sentry: %w", err)
	}

	return nil
}

// InitWithDefaults initializes Sentry with default configuration
func InitWithDefaults() error {
	cfg := DefaultConfig()
	cfg.BeforeSend = DefaultBeforeSend
	return Init(cfg)
}

// DefaultBeforeSend is the default event filter
func DefaultBeforeSend(event *sentry.Event, hint *sentry.EventHint) *sentry.Event {
	// Filter out sensitive data
	event = sanitizeEvent(event)

	// Filter out specific errors
	if shouldFilterError(event, hint) {
		return nil
	}

	return event
}

// Flush waits for pending events to be sent
func Flush(timeout time.Duration) bool {
	return sentry.Flush(timeout)
}

// Close flushes and closes the Sentry client
func Close() {
	sentry.Flush(2 * time.Second)
}

// CaptureException captures an error with optional context
func CaptureException(err error, ctx ...context.Context) *sentry.EventID {
	if err == nil {
		return nil
	}

	hub := getHub(ctx...)
	return hub.CaptureException(err)
}

// CaptureMessage captures a message with optional context
func CaptureMessage(message string, ctx ...context.Context) *sentry.EventID {
	hub := getHub(ctx...)
	return hub.CaptureMessage(message)
}

// CaptureError captures an error with additional context and tags
func CaptureError(err error, opts ...ErrorOption) *sentry.EventID {
	if err == nil {
		return nil
	}

	options := &errorOptions{}
	for _, opt := range opts {
		opt(options)
	}

	hub := sentry.CurrentHub().Clone()

	// Add tags
	for key, value := range options.tags {
		hub.Scope().SetTag(key, value)
	}

	// Add extra context
	for key, value := range options.extra {
		hub.Scope().SetExtra(key, value)
	}

	// Set user if provided
	if options.user != nil {
		hub.Scope().SetUser(*options.user)
	}

	// Set level if provided
	if options.level != "" {
		hub.Scope().SetLevel(options.level)
	}

	// Add fingerprint for grouping
	if len(options.fingerprint) > 0 {
		hub.Scope().SetFingerprint(options.fingerprint)
	}

	return hub.CaptureException(err)
}

// ErrorOption is a functional option for error capture
type ErrorOption func(*errorOptions)

type errorOptions struct {
	tags        map[string]string
	extra       map[string]interface{}
	user        *sentry.User
	level       sentry.Level
	fingerprint []string
	context     context.Context
}

// WithTags adds tags to the error
func WithTags(tags map[string]string) ErrorOption {
	return func(o *errorOptions) {
		if o.tags == nil {
			o.tags = make(map[string]string)
		}
		for k, v := range tags {
			o.tags[k] = v
		}
	}
}

// WithTag adds a single tag to the error
func WithTag(key, value string) ErrorOption {
	return func(o *errorOptions) {
		if o.tags == nil {
			o.tags = make(map[string]string)
		}
		o.tags[key] = value
	}
}

// WithExtra adds extra context to the error
func WithExtra(extra map[string]interface{}) ErrorOption {
	return func(o *errorOptions) {
		if o.extra == nil {
			o.extra = make(map[string]interface{})
		}
		for k, v := range extra {
			o.extra[k] = v
		}
	}
}

// WithContext adds a single context value
func WithContext(key string, value interface{}) ErrorOption {
	return func(o *errorOptions) {
		if o.extra == nil {
			o.extra = make(map[string]interface{})
		}
		o.extra[key] = value
	}
}

// WithUser adds user information to the error
func WithUser(id, email, username string) ErrorOption {
	return func(o *errorOptions) {
		o.user = &sentry.User{
			ID:       id,
			Email:    email,
			Username: username,
		}
	}
}

// WithLevel sets the error level
func WithLevel(level sentry.Level) ErrorOption {
	return func(o *errorOptions) {
		o.level = level
	}
}

// WithFingerprint sets custom fingerprint for error grouping
func WithFingerprint(fingerprint ...string) ErrorOption {
	return func(o *errorOptions) {
		o.fingerprint = fingerprint
	}
}

// WithRequestContext adds HTTP request context
func WithRequestContext(r *http.Request) ErrorOption {
	return func(o *errorOptions) {
		if o.extra == nil {
			o.extra = make(map[string]interface{})
		}
		o.extra["request"] = map[string]interface{}{
			"method":     r.Method,
			"url":        r.URL.String(),
			"user_agent": r.UserAgent(),
			"remote_ip":  getClientIP(r),
		}
	}
}

// StartSpan starts a new Sentry span for tracing
func StartSpan(ctx context.Context, operation string, opts ...sentry.SpanOption) *sentry.Span {
	return sentry.StartSpan(ctx, operation, opts...)
}

// StartTransaction starts a new Sentry transaction
func StartTransaction(ctx context.Context, name, operation string) *sentry.Span {
	return sentry.StartTransaction(ctx, name, sentry.WithOpName(operation))
}

// RecoverWithContext recovers from panics and reports to Sentry
func RecoverWithContext(ctx context.Context) {
	if err := recover(); err != nil {
		hub := sentry.GetHubFromContext(ctx)
		if hub == nil {
			hub = sentry.CurrentHub().Clone()
		}

		// Capture stack trace
		buf := make([]byte, 64<<10)
		buf = buf[:runtime.Stack(buf, false)]

		hub.Scope().SetExtra("stacktrace", string(buf))

		switch v := err.(type) {
		case error:
			hub.CaptureException(v)
		case string:
			hub.CaptureMessage(v)
		default:
			hub.CaptureMessage(fmt.Sprintf("%v", v))
		}

		hub.Flush(2 * time.Second)

		// Re-panic after reporting
		panic(err)
	}
}

// Helper functions

func getHub(ctx ...context.Context) *sentry.Hub {
	if len(ctx) > 0 && ctx[0] != nil {
		if hub := sentry.GetHubFromContext(ctx[0]); hub != nil {
			return hub
		}
	}
	return sentry.CurrentHub()
}

func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getHostname() string {
	hostname, err := os.Hostname()
	if err != nil {
		return "unknown"
	}
	return hostname
}

func getClientIP(r *http.Request) string {
	// Check X-Forwarded-For header
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		return xff
	}
	// Check X-Real-IP header
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return xri
	}
	return r.RemoteAddr
}

func sanitizeEvent(event *sentry.Event) *sentry.Event {
	// Remove sensitive headers
	sensitiveHeaders := []string{
		"Authorization",
		"Cookie",
		"Set-Cookie",
		"X-API-Key",
		"X-Auth-Token",
	}

	if event.Request != nil {
		for _, header := range sensitiveHeaders {
			delete(event.Request.Headers, header)
		}
	}

	// Redact sensitive extra data
	sensitiveKeys := []string{
		"password",
		"secret",
		"token",
		"api_key",
		"apikey",
		"credit_card",
		"ssn",
	}

	for key := range event.Extra {
		for _, sensitive := range sensitiveKeys {
			if containsInsensitive(key, sensitive) {
				event.Extra[key] = "[REDACTED]"
			}
		}
	}

	return event
}

func shouldFilterError(event *sentry.Event, hint *sentry.EventHint) bool {
	// Filter out specific error types
	if hint != nil && hint.OriginalException != nil {
		errMsg := hint.OriginalException.Error()

		// Filter context canceled errors
		if errMsg == "context canceled" {
			return true
		}

		// Filter client disconnection errors
		if containsInsensitive(errMsg, "client disconnected") ||
			containsInsensitive(errMsg, "broken pipe") ||
			containsInsensitive(errMsg, "connection reset by peer") {
			return true
		}
	}

	return false
}

func containsInsensitive(s, substr string) bool {
	return len(s) >= len(substr) &&
		(s == substr ||
			len(s) > len(substr) &&
				(containsLower(s, substr)))
}

func containsLower(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if equalFoldAt(s, i, substr) {
			return true
		}
	}
	return false
}

func equalFoldAt(s string, i int, substr string) bool {
	for j := 0; j < len(substr); j++ {
		c1 := s[i+j]
		c2 := substr[j]
		if c1 != c2 && toLower(c1) != toLower(c2) {
			return false
		}
	}
	return true
}

func toLower(c byte) byte {
	if c >= 'A' && c <= 'Z' {
		return c + 32
	}
	return c
}
