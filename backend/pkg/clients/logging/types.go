package logging

import (
	"time"
)

// Provider represents a logging service provider
type Provider string

const (
	ProviderBetterStack Provider = "betterstack"
	ProviderCloudWatch  Provider = "cloudwatch"
	ProviderLoki        Provider = "loki"
	ProviderMock        Provider = "mock"
)

// Level represents log severity level
type Level int

const (
	LevelDebug Level = iota
	LevelInfo
	LevelWarn
	LevelError
	LevelFatal
)

// String returns the string representation of the level
func (l Level) String() string {
	switch l {
	case LevelDebug:
		return "DEBUG"
	case LevelInfo:
		return "INFO"
	case LevelWarn:
		return "WARN"
	case LevelError:
		return "ERROR"
	case LevelFatal:
		return "FATAL"
	default:
		return "UNKNOWN"
	}
}

// ParseLevel parses a string into a Level
func ParseLevel(s string) Level {
	switch s {
	case "DEBUG", "debug":
		return LevelDebug
	case "INFO", "info":
		return LevelInfo
	case "WARN", "warn", "WARNING", "warning":
		return LevelWarn
	case "ERROR", "error":
		return LevelError
	case "FATAL", "fatal":
		return LevelFatal
	default:
		return LevelInfo
	}
}

// LogEntry represents a single log entry
type LogEntry struct {
	// Timestamp when the log entry was created
	Timestamp time.Time `json:"timestamp"`

	// Level is the severity level
	Level Level `json:"level"`

	// Message is the log message
	Message string `json:"message"`

	// Fields are structured key-value pairs
	Fields map[string]any `json:"fields,omitempty"`

	// Source identifies where the log came from
	Source string `json:"source,omitempty"`

	// TraceID for distributed tracing correlation
	TraceID string `json:"trace_id,omitempty"`

	// SpanID for distributed tracing correlation
	SpanID string `json:"span_id,omitempty"`

	// Error contains error details if this is an error log
	Error *ErrorInfo `json:"error,omitempty"`

	// HTTP contains HTTP request/response info if applicable
	HTTP *HTTPInfo `json:"http,omitempty"`

	// User contains user information if applicable
	User *UserInfo `json:"user,omitempty"`
}

// ErrorInfo contains error details
type ErrorInfo struct {
	Type       string   `json:"type,omitempty"`
	Message    string   `json:"message,omitempty"`
	StackTrace string   `json:"stack_trace,omitempty"`
	Code       string   `json:"code,omitempty"`
	Tags       []string `json:"tags,omitempty"`
}

// HTTPInfo contains HTTP request/response details
type HTTPInfo struct {
	Method     string            `json:"method,omitempty"`
	URL        string            `json:"url,omitempty"`
	Path       string            `json:"path,omitempty"`
	StatusCode int               `json:"status_code,omitempty"`
	Duration   time.Duration     `json:"duration,omitempty"`
	UserAgent  string            `json:"user_agent,omitempty"`
	RemoteAddr string            `json:"remote_addr,omitempty"`
	Headers    map[string]string `json:"headers,omitempty"`
}

// UserInfo contains user identification
type UserInfo struct {
	ID       string `json:"id,omitempty"`
	Email    string `json:"email,omitempty"`
	TenantID string `json:"tenant_id,omitempty"`
}

// NewLogEntry creates a new log entry with the given level and message
func NewLogEntry(level Level, message string) *LogEntry {
	return &LogEntry{
		Timestamp: time.Now().UTC(),
		Level:     level,
		Message:   message,
		Fields:    make(map[string]any),
	}
}

// WithField adds a field to the log entry
func (e *LogEntry) WithField(key string, value any) *LogEntry {
	if e.Fields == nil {
		e.Fields = make(map[string]any)
	}
	e.Fields[key] = value
	return e
}

// WithFields adds multiple fields to the log entry
func (e *LogEntry) WithFields(fields map[string]any) *LogEntry {
	if e.Fields == nil {
		e.Fields = make(map[string]any)
	}
	for k, v := range fields {
		e.Fields[k] = v
	}
	return e
}

// WithSource sets the source
func (e *LogEntry) WithSource(source string) *LogEntry {
	e.Source = source
	return e
}

// WithTraceID sets the trace ID
func (e *LogEntry) WithTraceID(traceID string) *LogEntry {
	e.TraceID = traceID
	return e
}

// WithSpanID sets the span ID
func (e *LogEntry) WithSpanID(spanID string) *LogEntry {
	e.SpanID = spanID
	return e
}

// WithError sets error information
func (e *LogEntry) WithError(err error) *LogEntry {
	if err != nil {
		e.Error = &ErrorInfo{
			Message: err.Error(),
		}
	}
	return e
}

// WithErrorInfo sets detailed error information
func (e *LogEntry) WithErrorInfo(info *ErrorInfo) *LogEntry {
	e.Error = info
	return e
}

// WithHTTP sets HTTP information
func (e *LogEntry) WithHTTP(info *HTTPInfo) *LogEntry {
	e.HTTP = info
	return e
}

// WithUser sets user information
func (e *LogEntry) WithUser(info *UserInfo) *LogEntry {
	e.User = info
	return e
}

// BatchOptions configures batch logging behavior
type BatchOptions struct {
	// MaxSize is the maximum number of entries to batch before flushing
	MaxSize int

	// FlushInterval is the maximum time between flushes
	FlushInterval time.Duration

	// RetryAttempts is the number of retry attempts on failure
	RetryAttempts int

	// RetryDelay is the initial delay between retries
	RetryDelay time.Duration
}

// DefaultBatchOptions returns sensible default batch options
func DefaultBatchOptions() *BatchOptions {
	return &BatchOptions{
		MaxSize:       100,
		FlushInterval: 5 * time.Second,
		RetryAttempts: 3,
		RetryDelay:    1 * time.Second,
	}
}
