package logging

import (
	"context"
)

// Client defines the interface for logging operations.
// Implementations can send logs to various backends like Better Stack,
// CloudWatch Logs, Loki, or stdout.
type Client interface {
	// Log sends a single log entry
	Log(ctx context.Context, entry *LogEntry) error

	// LogBatch sends multiple log entries
	LogBatch(ctx context.Context, entries []*LogEntry) error

	// Convenience methods for common log levels

	// Debug logs a debug message
	Debug(ctx context.Context, msg string, fields map[string]any)

	// Info logs an info message
	Info(ctx context.Context, msg string, fields map[string]any)

	// Warn logs a warning message
	Warn(ctx context.Context, msg string, fields map[string]any)

	// Error logs an error message
	Error(ctx context.Context, msg string, fields map[string]any)

	// Fatal logs a fatal message (does not exit the program)
	Fatal(ctx context.Context, msg string, fields map[string]any)

	// WithFields returns a new logger with the given fields pre-set
	WithFields(fields map[string]any) Client

	// WithSource returns a new logger with the given source pre-set
	WithSource(source string) Client

	// Flush forces any buffered logs to be sent
	Flush() error

	// Close releases resources and flushes remaining logs
	Close() error
}

// ContextKey is the type for context keys used by the logging package
type ContextKey string

const (
	// ContextKeyTraceID is the context key for trace ID
	ContextKeyTraceID ContextKey = "trace_id"

	// ContextKeySpanID is the context key for span ID
	ContextKeySpanID ContextKey = "span_id"

	// ContextKeyUserID is the context key for user ID
	ContextKeyUserID ContextKey = "user_id"

	// ContextKeyTenantID is the context key for tenant ID
	ContextKeyTenantID ContextKey = "tenant_id"

	// ContextKeyRequestID is the context key for request ID
	ContextKeyRequestID ContextKey = "request_id"
)

// ExtractTraceInfo extracts trace information from context
func ExtractTraceInfo(ctx context.Context) (traceID, spanID string) {
	if v := ctx.Value(ContextKeyTraceID); v != nil {
		traceID, _ = v.(string)
	}
	if v := ctx.Value(ContextKeySpanID); v != nil {
		spanID, _ = v.(string)
	}
	return
}

// ExtractUserInfo extracts user information from context
func ExtractUserInfo(ctx context.Context) *UserInfo {
	info := &UserInfo{}
	if v := ctx.Value(ContextKeyUserID); v != nil {
		info.ID, _ = v.(string)
	}
	if v := ctx.Value(ContextKeyTenantID); v != nil {
		info.TenantID, _ = v.(string)
	}
	if info.ID == "" && info.TenantID == "" {
		return nil
	}
	return info
}

// WithTraceID adds trace ID to context
func WithTraceID(ctx context.Context, traceID string) context.Context {
	return context.WithValue(ctx, ContextKeyTraceID, traceID)
}

// WithSpanID adds span ID to context
func WithSpanID(ctx context.Context, spanID string) context.Context {
	return context.WithValue(ctx, ContextKeySpanID, spanID)
}

// WithUserID adds user ID to context
func WithUserID(ctx context.Context, userID string) context.Context {
	return context.WithValue(ctx, ContextKeyUserID, userID)
}

// WithTenantID adds tenant ID to context
func WithTenantID(ctx context.Context, tenantID string) context.Context {
	return context.WithValue(ctx, ContextKeyTenantID, tenantID)
}

// WithRequestID adds request ID to context
func WithRequestID(ctx context.Context, requestID string) context.Context {
	return context.WithValue(ctx, ContextKeyRequestID, requestID)
}
