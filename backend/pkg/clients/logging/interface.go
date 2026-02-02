package logging

import (
	"context"
	"sync"
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

// Global logger instance
var (
	globalMu     sync.RWMutex
	globalLogger Client
)

// SetDefault sets the global default logger.
// This should be called once during application initialization.
func SetDefault(logger Client) {
	globalMu.Lock()
	defer globalMu.Unlock()
	globalLogger = logger
}

// L returns the global logger. If no logger has been set, it returns a no-op logger.
// This is the primary way to access the logger throughout the application.
func L() Client {
	globalMu.RLock()
	defer globalMu.RUnlock()
	if globalLogger == nil {
		return &noopClient{}
	}
	return globalLogger
}

// Default returns the global logger (alias for L).
func Default() Client {
	return L()
}

// Convenience functions that use the global logger

// Debug logs a debug message using the global logger
func Debug(ctx context.Context, msg string, fields map[string]any) {
	L().Debug(ctx, msg, fields)
}

// Info logs an info message using the global logger
func Info(ctx context.Context, msg string, fields map[string]any) {
	L().Info(ctx, msg, fields)
}

// Warn logs a warning message using the global logger
func Warn(ctx context.Context, msg string, fields map[string]any) {
	L().Warn(ctx, msg, fields)
}

// Error logs an error message using the global logger
func Error(ctx context.Context, msg string, fields map[string]any) {
	L().Error(ctx, msg, fields)
}

// Fatal logs a fatal message using the global logger
func Fatal(ctx context.Context, msg string, fields map[string]any) {
	L().Fatal(ctx, msg, fields)
}

// noopClient is a no-op implementation used when no logger is configured
type noopClient struct{}

func (n *noopClient) Log(ctx context.Context, entry *LogEntry) error               { return nil }
func (n *noopClient) LogBatch(ctx context.Context, entries []*LogEntry) error      { return nil }
func (n *noopClient) Debug(ctx context.Context, msg string, fields map[string]any) {}
func (n *noopClient) Info(ctx context.Context, msg string, fields map[string]any)  {}
func (n *noopClient) Warn(ctx context.Context, msg string, fields map[string]any)  {}
func (n *noopClient) Error(ctx context.Context, msg string, fields map[string]any) {}
func (n *noopClient) Fatal(ctx context.Context, msg string, fields map[string]any) {}
func (n *noopClient) WithFields(fields map[string]any) Client                      { return n }
func (n *noopClient) WithSource(source string) Client                              { return n }
func (n *noopClient) Flush() error                                                 { return nil }
func (n *noopClient) Close() error                                                 { return nil }
func (n *noopClient) Handler() any                                                 { return nil }
