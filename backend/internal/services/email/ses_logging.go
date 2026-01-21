package email

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch"
	cwtypes "github.com/aws/aws-sdk-go-v2/service/cloudwatch/types"
)

// LogLevel represents the severity of a log message
type LogLevel int

const (
	LogLevelDebug LogLevel = iota
	LogLevelInfo
	LogLevelWarn
	LogLevelError
)

func (l LogLevel) String() string {
	switch l {
	case LogLevelDebug:
		return "DEBUG"
	case LogLevelInfo:
		return "INFO"
	case LogLevelWarn:
		return "WARN"
	case LogLevelError:
		return "ERROR"
	default:
		return "UNKNOWN"
	}
}

// ParseLogLevel parses a string log level
func ParseLogLevel(s string) LogLevel {
	switch s {
	case "debug", "DEBUG":
		return LogLevelDebug
	case "info", "INFO":
		return LogLevelInfo
	case "warn", "WARN", "warning", "WARNING":
		return LogLevelWarn
	case "error", "ERROR":
		return LogLevelError
	default:
		return LogLevelInfo
	}
}

// LogEntry represents a single log entry
type LogEntry struct {
	Timestamp   time.Time              `json:"timestamp"`
	Level       string                 `json:"level"`
	Message     string                 `json:"message"`
	Service     string                 `json:"service"`
	Environment string                 `json:"environment,omitempty"`
	Fields      map[string]interface{} `json:"fields,omitempty"`
	Error       string                 `json:"error,omitempty"`
	TraceID     string                 `json:"trace_id,omitempty"`
}

// SESLogger handles logging for the SES service
type SESLogger struct {
	config *SESConfig
	level  LogLevel
	format string
	output io.Writer
	mu     sync.Mutex

	// Context fields
	service     string
	environment string
	fields      map[string]interface{}
}

// NewSESLogger creates a new SES logger
func NewSESLogger(config *SESConfig) *SESLogger {
	return &SESLogger{
		config:      config,
		level:       ParseLogLevel(config.LogLevel),
		format:      config.LogFormat,
		output:      os.Stdout,
		service:     "ses-email",
		environment: string(config.Environment),
		fields:      make(map[string]interface{}),
	}
}

// SetOutput sets the output writer
func (l *SESLogger) SetOutput(w io.Writer) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.output = w
}

// SetLevel sets the minimum log level
func (l *SESLogger) SetLevel(level LogLevel) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.level = level
}

// WithField returns a logger with an additional field
func (l *SESLogger) WithField(key string, value interface{}) *SESLogger {
	newLogger := *l
	newLogger.fields = make(map[string]interface{})
	for k, v := range l.fields {
		newLogger.fields[k] = v
	}
	newLogger.fields[key] = value
	return &newLogger
}

// WithFields returns a logger with additional fields
func (l *SESLogger) WithFields(fields map[string]interface{}) *SESLogger {
	newLogger := *l
	newLogger.fields = make(map[string]interface{})
	for k, v := range l.fields {
		newLogger.fields[k] = v
	}
	for k, v := range fields {
		newLogger.fields[k] = v
	}
	return &newLogger
}

// Debug logs a debug message
func (l *SESLogger) Debug(msg string, args ...interface{}) {
	l.log(LogLevelDebug, msg, args...)
}

// Info logs an info message
func (l *SESLogger) Info(msg string, args ...interface{}) {
	l.log(LogLevelInfo, msg, args...)
}

// Warn logs a warning message
func (l *SESLogger) Warn(msg string, args ...interface{}) {
	l.log(LogLevelWarn, msg, args...)
}

// Error logs an error message
func (l *SESLogger) Error(msg string, args ...interface{}) {
	l.log(LogLevelError, msg, args...)
}

func (l *SESLogger) log(level LogLevel, msg string, args ...interface{}) {
	if level < l.level {
		return
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	entry := LogEntry{
		Timestamp:   time.Now().UTC(),
		Level:       level.String(),
		Message:     msg,
		Service:     l.service,
		Environment: l.environment,
		Fields:      make(map[string]interface{}),
	}

	// Copy base fields
	for k, v := range l.fields {
		entry.Fields[k] = v
	}

	// Parse args as key-value pairs
	for i := 0; i < len(args)-1; i += 2 {
		if key, ok := args[i].(string); ok {
			entry.Fields[key] = args[i+1]
		}
	}

	// Extract error if present
	if err, ok := entry.Fields["error"]; ok {
		if errVal, ok := err.(error); ok {
			entry.Error = errVal.Error()
			delete(entry.Fields, "error")
		}
	}

	// Format and write
	var output string
	if l.format == "json" {
		data, _ := json.Marshal(entry)
		output = string(data)
	} else {
		output = l.formatText(entry)
	}

	fmt.Fprintln(l.output, output)
}

func (l *SESLogger) formatText(entry LogEntry) string {
	result := fmt.Sprintf("[%s] %s %s: %s",
		entry.Timestamp.Format(time.RFC3339),
		entry.Level,
		entry.Service,
		entry.Message,
	)

	for k, v := range entry.Fields {
		result += fmt.Sprintf(" %s=%v", k, v)
	}

	if entry.Error != "" {
		result += fmt.Sprintf(" error=%s", entry.Error)
	}

	return result
}

// ============================================
// CloudWatch Metrics
// ============================================

// SESMetrics handles CloudWatch metrics for SES
type SESMetrics struct {
	client     *cloudwatch.Client
	config     *SESConfig
	namespace  string
	dimensions []cwtypes.Dimension
	mu         sync.Mutex

	// Buffered metrics
	buffer     []cwtypes.MetricDatum
	bufferSize int
	flushTimer *time.Timer
}

// NewSESMetrics creates a new metrics handler
func NewSESMetrics(ctx context.Context, config *SESConfig) (*SESMetrics, error) {
	client, err := config.CreateCloudWatchClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create CloudWatch client: %w", err)
	}

	// Build dimensions
	var dimensions []cwtypes.Dimension
	for k, v := range config.MetricsDimensions {
		dimensions = append(dimensions, cwtypes.Dimension{
			Name:  aws.String(k),
			Value: aws.String(v),
		})
	}

	// Add default dimensions
	dimensions = append(dimensions,
		cwtypes.Dimension{
			Name:  aws.String("Service"),
			Value: aws.String("SES"),
		},
		cwtypes.Dimension{
			Name:  aws.String("Region"),
			Value: aws.String(config.Region),
		},
	)

	m := &SESMetrics{
		client:     client,
		config:     config,
		namespace:  config.MetricsNamespace,
		dimensions: dimensions,
		buffer:     make([]cwtypes.MetricDatum, 0, 20),
		bufferSize: 20,
	}

	return m, nil
}

// RecordSendSuccess records a successful email send
func (m *SESMetrics) RecordSendSuccess(ctx context.Context, duration time.Duration) {
	m.recordMetric("EmailsSent", 1, cwtypes.StandardUnitCount)
	m.recordMetric("SendLatency", float64(duration.Milliseconds()), cwtypes.StandardUnitMilliseconds)
	m.recordMetric("SendSuccessRate", 1, cwtypes.StandardUnitCount)
}

// RecordSendFailure records a failed email send
func (m *SESMetrics) RecordSendFailure(ctx context.Context, err error) {
	m.recordMetric("EmailsFailed", 1, cwtypes.StandardUnitCount)
	m.recordMetric("SendSuccessRate", 0, cwtypes.StandardUnitCount)

	// Record by error type
	handler := NewSESErrorHandler(m.config)
	classification := handler.ClassifyError(err)
	m.recordMetricWithDimension("ErrorsByType", 1, cwtypes.StandardUnitCount,
		"ErrorType", classification)
}

// RecordRetry records a retry attempt
func (m *SESMetrics) RecordRetry(ctx context.Context) {
	m.recordMetric("RetryAttempts", 1, cwtypes.StandardUnitCount)
}

// RecordBatchSend records batch send metrics
func (m *SESMetrics) RecordBatchSend(ctx context.Context, total, successful, failed int, duration time.Duration) {
	m.recordMetric("BatchSize", float64(total), cwtypes.StandardUnitCount)
	m.recordMetric("BatchSuccessful", float64(successful), cwtypes.StandardUnitCount)
	m.recordMetric("BatchFailed", float64(failed), cwtypes.StandardUnitCount)
	m.recordMetric("BatchDuration", float64(duration.Milliseconds()), cwtypes.StandardUnitMilliseconds)
}

// RecordQuotaUsage records quota usage
func (m *SESMetrics) RecordQuotaUsage(ctx context.Context, used, max float64) {
	m.recordMetric("QuotaUsed", used, cwtypes.StandardUnitCount)
	m.recordMetric("QuotaMax", max, cwtypes.StandardUnitCount)
	if max > 0 {
		m.recordMetric("QuotaUtilization", (used/max)*100, cwtypes.StandardUnitPercent)
	}
}

// RecordRateLimit records rate limit metrics
func (m *SESMetrics) RecordRateLimit(ctx context.Context, currentRate, maxRate float64) {
	m.recordMetric("CurrentSendRate", currentRate, cwtypes.StandardUnitCountSecond)
	m.recordMetric("MaxSendRate", maxRate, cwtypes.StandardUnitCountSecond)
}

func (m *SESMetrics) recordMetric(name string, value float64, unit cwtypes.StandardUnit) {
	m.mu.Lock()
	defer m.mu.Unlock()

	datum := cwtypes.MetricDatum{
		MetricName: aws.String(name),
		Value:      aws.Float64(value),
		Unit:       unit,
		Timestamp:  aws.Time(time.Now()),
		Dimensions: m.dimensions,
	}

	m.buffer = append(m.buffer, datum)

	if len(m.buffer) >= m.bufferSize {
		m.flush()
	} else if m.flushTimer == nil {
		m.flushTimer = time.AfterFunc(10*time.Second, func() {
			m.mu.Lock()
			defer m.mu.Unlock()
			m.flush()
		})
	}
}

func (m *SESMetrics) recordMetricWithDimension(name string, value float64, unit cwtypes.StandardUnit, dimName, dimValue string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Add extra dimension
	dimensions := make([]cwtypes.Dimension, len(m.dimensions)+1)
	copy(dimensions, m.dimensions)
	dimensions[len(m.dimensions)] = cwtypes.Dimension{
		Name:  aws.String(dimName),
		Value: aws.String(dimValue),
	}

	datum := cwtypes.MetricDatum{
		MetricName: aws.String(name),
		Value:      aws.Float64(value),
		Unit:       unit,
		Timestamp:  aws.Time(time.Now()),
		Dimensions: dimensions,
	}

	m.buffer = append(m.buffer, datum)

	if len(m.buffer) >= m.bufferSize {
		m.flush()
	}
}

func (m *SESMetrics) flush() {
	if len(m.buffer) == 0 {
		return
	}

	if m.flushTimer != nil {
		m.flushTimer.Stop()
		m.flushTimer = nil
	}

	// CloudWatch allows max 20 metrics per request
	data := m.buffer
	m.buffer = make([]cwtypes.MetricDatum, 0, m.bufferSize)

	// Send in background
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		_, err := m.client.PutMetricData(ctx, &cloudwatch.PutMetricDataInput{
			Namespace:  aws.String(m.namespace),
			MetricData: data,
		})
		if err != nil {
			log.Printf("Failed to send CloudWatch metrics: %v", err)
		}
	}()
}

// Flush forces a flush of all buffered metrics
func (m *SESMetrics) Flush() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.flush()
}

// Close closes the metrics handler
func (m *SESMetrics) Close() {
	m.Flush()
}

// ============================================
// Audit Logging
// ============================================

// AuditLog records audit events for compliance
type AuditLog struct {
	logger  *SESLogger
	enabled bool
	mu      sync.Mutex
}

// AuditEvent represents an auditable event
type AuditEvent struct {
	Timestamp time.Time              `json:"timestamp"`
	EventType string                 `json:"event_type"`
	Actor     string                 `json:"actor,omitempty"`
	Resource  string                 `json:"resource"`
	Action    string                 `json:"action"`
	Status    string                 `json:"status"`
	Details   map[string]interface{} `json:"details,omitempty"`
	RequestID string                 `json:"request_id,omitempty"`
	IPAddress string                 `json:"ip_address,omitempty"`
}

// NewAuditLog creates a new audit logger
func NewAuditLog(config *SESConfig) *AuditLog {
	return &AuditLog{
		logger:  NewSESLogger(config).WithField("audit", true),
		enabled: true,
	}
}

// SetEnabled enables or disables audit logging
func (a *AuditLog) SetEnabled(enabled bool) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.enabled = enabled
}

// LogEmailSent logs a successful email send
func (a *AuditLog) LogEmailSent(email *Email, result *SendResult) {
	if !a.enabled {
		return
	}

	a.log(AuditEvent{
		EventType: "EMAIL_SENT",
		Resource:  "email",
		Action:    "send",
		Status:    "success",
		Details: map[string]interface{}{
			"message_id": result.MessageID,
			"to":         email.To,
			"subject":    email.Subject,
			"duration":   result.Duration.String(),
			"attempts":   result.Attempts,
		},
	})
}

// LogEmailFailed logs a failed email send
func (a *AuditLog) LogEmailFailed(email *Email, err error, attempts int) {
	if !a.enabled {
		return
	}

	a.log(AuditEvent{
		EventType: "EMAIL_FAILED",
		Resource:  "email",
		Action:    "send",
		Status:    "failure",
		Details: map[string]interface{}{
			"to":       email.To,
			"subject":  email.Subject,
			"error":    err.Error(),
			"attempts": attempts,
		},
	})
}

// LogDomainVerified logs a domain verification
func (a *AuditLog) LogDomainVerified(domain string, status string) {
	if !a.enabled {
		return
	}

	a.log(AuditEvent{
		EventType: "DOMAIN_VERIFIED",
		Resource:  "domain",
		Action:    "verify",
		Status:    status,
		Details: map[string]interface{}{
			"domain": domain,
		},
	})
}

// LogConfigurationChange logs a configuration change
func (a *AuditLog) LogConfigurationChange(actor, setting, oldValue, newValue string) {
	if !a.enabled {
		return
	}

	a.log(AuditEvent{
		EventType: "CONFIG_CHANGE",
		Resource:  "configuration",
		Action:    "update",
		Status:    "success",
		Actor:     actor,
		Details: map[string]interface{}{
			"setting":   setting,
			"old_value": oldValue,
			"new_value": newValue,
		},
	})
}

func (a *AuditLog) log(event AuditEvent) {
	a.mu.Lock()
	defer a.mu.Unlock()

	event.Timestamp = time.Now().UTC()

	data, _ := json.Marshal(event)
	a.logger.Info(string(data))
}

// ============================================
// Structured Logging Helper
// ============================================

// LogContext provides context for structured logging
type LogContext struct {
	TraceID   string
	RequestID string
	UserID    string
	Email     string
}

// WithContext returns a logger with context fields
func (l *SESLogger) WithContext(ctx LogContext) *SESLogger {
	fields := make(map[string]interface{})

	if ctx.TraceID != "" {
		fields["trace_id"] = ctx.TraceID
	}
	if ctx.RequestID != "" {
		fields["request_id"] = ctx.RequestID
	}
	if ctx.UserID != "" {
		fields["user_id"] = ctx.UserID
	}
	if ctx.Email != "" {
		fields["email"] = ctx.Email
	}

	return l.WithFields(fields)
}
