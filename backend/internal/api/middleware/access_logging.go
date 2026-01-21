package middleware

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gopkg.in/natefinch/lumberjack.v2"

	"github.com/javaknight1/servicepro/backend/config"
)

// AccessLogger handles access logging and metrics collection
type AccessLogger struct {
	config      *config.AccessLoggingConfig
	logger      *log.Logger
	auditLogger *log.Logger
	metrics     *Metrics
	metricsLock sync.RWMutex
}

// Metrics holds request metrics
type Metrics struct {
	TotalRequests      int64
	SuccessfulRequests int64
	FailedRequests     int64
	DeniedRequests     int64
	TotalResponseTime  time.Duration
	TotalRequestSize   int64
	TotalResponseSize  int64
	RequestsByMethod   map[string]int64
	RequestsByPath     map[string]int64
	RequestsByStatus   map[int]int64
	SlowRequests       int64
}

// AccessLogEntry represents a single access log entry
type AccessLogEntry struct {
	Timestamp    time.Time         `json:"timestamp"`
	RequestID    string            `json:"request_id"`
	Method       string            `json:"method"`
	Path         string            `json:"path"`
	Query        string            `json:"query,omitempty"`
	StatusCode   int               `json:"status_code"`
	ResponseTime int64             `json:"response_time_ms"`
	RequestSize  int64             `json:"request_size,omitempty"`
	ResponseSize int64             `json:"response_size,omitempty"`
	ClientIP     string            `json:"client_ip"`
	UserAgent    string            `json:"user_agent,omitempty"`
	UserID       string            `json:"user_id,omitempty"`
	UserEmail    string            `json:"user_email,omitempty"`
	Headers      map[string]string `json:"headers,omitempty"`
	RequestBody  string            `json:"request_body,omitempty"`
	ResponseBody string            `json:"response_body,omitempty"`
	Error        string            `json:"error,omitempty"`
	Tags         []string          `json:"tags,omitempty"`
}

// AuditLogEntry represents an audit log entry for sensitive operations
type AuditLogEntry struct {
	Timestamp  time.Time `json:"timestamp"`
	UserID     string    `json:"user_id"`
	UserEmail  string    `json:"user_email,omitempty"`
	Action     string    `json:"action"`
	Resource   string    `json:"resource"`
	ResourceID string    `json:"resource_id,omitempty"`
	Result     string    `json:"result"`
	IP         string    `json:"ip"`
	Details    string    `json:"details,omitempty"`
	RequestID  string    `json:"request_id"`
}

// responseWriter wraps gin.ResponseWriter to capture response details
type responseWriter struct {
	gin.ResponseWriter
	body       *bytes.Buffer
	statusCode int
	size       int
}

func (rw *responseWriter) Write(b []byte) (int, error) {
	size, err := rw.ResponseWriter.Write(b)
	rw.size += size

	// Capture response body if enabled
	if rw.body != nil {
		rw.body.Write(b)
	}

	return size, err
}

func (rw *responseWriter) WriteHeader(statusCode int) {
	rw.statusCode = statusCode
	rw.ResponseWriter.WriteHeader(statusCode)
}

// NewAccessLogger creates a new access logger
func NewAccessLogger(cfg *config.AccessLoggingConfig) (*AccessLogger, error) {
	al := &AccessLogger{
		config: cfg,
		metrics: &Metrics{
			RequestsByMethod: make(map[string]int64),
			RequestsByPath:   make(map[string]int64),
			RequestsByStatus: make(map[int]int64),
		},
	}

	// Set up access logger
	if cfg.Enabled {
		var writers []io.Writer

		// File output with rotation
		if cfg.LogToFile {
			// Ensure log directory exists
			logDir := filepath.Dir(cfg.LogFilePath)
			if err := os.MkdirAll(logDir, 0755); err != nil {
				return nil, fmt.Errorf("failed to create log directory: %w", err)
			}

			fileWriter := &lumberjack.Logger{
				Filename:   cfg.LogFilePath,
				MaxSize:    cfg.LogRotation.MaxSize,
				MaxBackups: cfg.LogRotation.MaxBackups,
				MaxAge:     cfg.LogRotation.MaxAge,
				Compress:   cfg.LogRotation.Compress,
			}
			writers = append(writers, fileWriter)
		}

		// Stdout output
		if cfg.LogToStdout {
			writers = append(writers, os.Stdout)
		}

		if len(writers) > 0 {
			multiWriter := io.MultiWriter(writers...)
			al.logger = log.New(multiWriter, "", 0) // No prefix, we'll format ourselves
		}
	}

	// Set up audit logger
	if cfg.EnableAuditTrail {
		auditDir := filepath.Dir(cfg.AuditTrailPath)
		if err := os.MkdirAll(auditDir, 0755); err != nil {
			return nil, fmt.Errorf("failed to create audit directory: %w", err)
		}

		auditWriter := &lumberjack.Logger{
			Filename:   cfg.AuditTrailPath,
			MaxSize:    100,
			MaxBackups: 30,
			MaxAge:     90, // Keep audit logs for 90 days
			Compress:   true,
		}

		al.auditLogger = log.New(auditWriter, "", 0)
	}

	return al, nil
}

// AccessLogging is the main middleware function for access logging
func (al *AccessLogger) AccessLogging() gin.HandlerFunc {
	return func(c *gin.Context) {
		if !al.config.Enabled {
			c.Next()
			return
		}

		start := time.Now()

		// Generate request ID
		requestID := uuid.New().String()
		c.Set("request_id", requestID)

		// Capture request body if enabled
		var requestBody string
		if al.config.LogRequestBody && c.Request.Body != nil {
			bodyBytes, err := io.ReadAll(c.Request.Body)
			if err == nil {
				// Restore the body for handlers to read
				c.Request.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))

				// Filter sensitive data
				requestBody = al.filterSensitiveData(string(bodyBytes))

				// Limit body size
				if len(requestBody) > al.config.MaxBodyLogSize {
					requestBody = requestBody[:al.config.MaxBodyLogSize] + "...[truncated]"
				}
			}
		}

		// Create custom response writer to capture response
		var responseBodyBuf *bytes.Buffer
		if al.config.LogResponseBody {
			responseBodyBuf = &bytes.Buffer{}
		}

		rw := &responseWriter{
			ResponseWriter: c.Writer,
			body:           responseBodyBuf,
			statusCode:     200, // Default status
		}
		c.Writer = rw

		// Process request
		c.Next()

		// Calculate response time
		duration := time.Since(start)

		// Build log entry
		entry := al.buildLogEntry(c, rw, requestID, duration, requestBody)

		// Log based on configuration and request status
		shouldLog := false

		if rw.statusCode >= 200 && rw.statusCode < 400 && al.config.LogSuccessfulRequests {
			shouldLog = true
		}

		if rw.statusCode >= 400 && al.config.LogFailedRequests {
			shouldLog = true
			entry.Tags = append(entry.Tags, "failed")
		}

		if rw.statusCode == 403 && al.config.LogDeniedAccess {
			shouldLog = true
			entry.Tags = append(entry.Tags, "denied")
		}

		if duration > al.config.SlowRequestThreshold && al.config.LogSlowRequests {
			shouldLog = true
			entry.Tags = append(entry.Tags, "slow")
		}

		if shouldLog {
			al.logEntry(entry)
		}

		// Update metrics
		if al.config.EnableMetrics {
			al.updateMetrics(entry, duration, rw)
		}
	}
}

// AuditLog logs an audit entry for sensitive operations
func (al *AccessLogger) AuditLog(c *gin.Context, action, resource, resourceID, result, details string) {
	if !al.config.EnableAuditTrail || al.auditLogger == nil {
		return
	}

	userID, _ := c.Get("user_id")
	userEmail, _ := c.Get("user_email")
	requestID, _ := c.Get("request_id")

	entry := AuditLogEntry{
		Timestamp:  time.Now(),
		UserID:     fmt.Sprintf("%v", userID),
		UserEmail:  fmt.Sprintf("%v", userEmail),
		Action:     action,
		Resource:   resource,
		ResourceID: resourceID,
		Result:     result,
		IP:         c.ClientIP(),
		Details:    details,
		RequestID:  fmt.Sprintf("%v", requestID),
	}

	jsonEntry, err := json.Marshal(entry)
	if err != nil {
		log.Printf("[ERROR] Failed to marshal audit entry: %v", err)
		return
	}

	al.auditLogger.Println(string(jsonEntry))
}

// GetMetrics returns current metrics
func (al *AccessLogger) GetMetrics() *Metrics {
	al.metricsLock.RLock()
	defer al.metricsLock.RUnlock()

	// Return a copy to avoid race conditions
	metricsCopy := &Metrics{
		TotalRequests:      al.metrics.TotalRequests,
		SuccessfulRequests: al.metrics.SuccessfulRequests,
		FailedRequests:     al.metrics.FailedRequests,
		DeniedRequests:     al.metrics.DeniedRequests,
		TotalResponseTime:  al.metrics.TotalResponseTime,
		TotalRequestSize:   al.metrics.TotalRequestSize,
		TotalResponseSize:  al.metrics.TotalResponseSize,
		SlowRequests:       al.metrics.SlowRequests,
		RequestsByMethod:   make(map[string]int64),
		RequestsByPath:     make(map[string]int64),
		RequestsByStatus:   make(map[int]int64),
	}

	// Copy maps
	for k, v := range al.metrics.RequestsByMethod {
		metricsCopy.RequestsByMethod[k] = v
	}
	for k, v := range al.metrics.RequestsByPath {
		metricsCopy.RequestsByPath[k] = v
	}
	for k, v := range al.metrics.RequestsByStatus {
		metricsCopy.RequestsByStatus[k] = v
	}

	return metricsCopy
}

// ResetMetrics resets all metrics
func (al *AccessLogger) ResetMetrics() {
	al.metricsLock.Lock()
	defer al.metricsLock.Unlock()

	al.metrics = &Metrics{
		RequestsByMethod: make(map[string]int64),
		RequestsByPath:   make(map[string]int64),
		RequestsByStatus: make(map[int]int64),
	}
}

// FormatMetrics returns a formatted string of current metrics
func (al *AccessLogger) FormatMetrics() string {
	metrics := al.GetMetrics()

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Total Requests: %d\n", metrics.TotalRequests))
	sb.WriteString(fmt.Sprintf("Successful: %d, Failed: %d, Denied: %d\n",
		metrics.SuccessfulRequests, metrics.FailedRequests, metrics.DeniedRequests))

	if metrics.TotalRequests > 0 {
		avgResponseTime := metrics.TotalResponseTime / time.Duration(metrics.TotalRequests)
		sb.WriteString(fmt.Sprintf("Average Response Time: %v\n", avgResponseTime))
	}

	sb.WriteString(fmt.Sprintf("Slow Requests: %d\n", metrics.SlowRequests))

	sb.WriteString("\nRequests by Method:\n")
	for method, count := range metrics.RequestsByMethod {
		sb.WriteString(fmt.Sprintf("  %s: %d\n", method, count))
	}

	sb.WriteString("\nTop Paths:\n")
	for path, count := range metrics.RequestsByPath {
		sb.WriteString(fmt.Sprintf("  %s: %d\n", path, count))
	}

	return sb.String()
}

// Private helper methods

// buildLogEntry constructs a log entry from request/response data
func (al *AccessLogger) buildLogEntry(
	c *gin.Context,
	rw *responseWriter,
	requestID string,
	duration time.Duration,
	requestBody string,
) AccessLogEntry {
	entry := AccessLogEntry{
		Timestamp:    time.Now(),
		RequestID:    requestID,
		Method:       c.Request.Method,
		Path:         c.Request.URL.Path,
		StatusCode:   rw.statusCode,
		ResponseTime: duration.Milliseconds(),
		ClientIP:     c.ClientIP(),
		Tags:         []string{},
	}

	// Add query parameters
	if al.config.LogQueryParams && len(c.Request.URL.RawQuery) > 0 {
		entry.Query = al.filterSensitiveData(c.Request.URL.RawQuery)
	}

	// Add user agent
	entry.UserAgent = c.Request.UserAgent()

	// Add user information if available and configured
	if al.config.IncludeUserInfo {
		if userID, exists := c.Get("user_id"); exists {
			entry.UserID = fmt.Sprintf("%v", userID)
		}
		if userEmail, exists := c.Get("user_email"); exists {
			entry.UserEmail = fmt.Sprintf("%v", userEmail)
		}
	}

	// Add headers
	if al.config.LogHeaders {
		entry.Headers = make(map[string]string)
		for key, values := range c.Request.Header {
			if !al.config.ShouldFilterField(key) {
				entry.Headers[key] = strings.Join(values, ", ")
			}
		}
	}

	// Add request/response sizes
	if al.config.TrackRequestSize {
		entry.RequestSize = c.Request.ContentLength
	}

	if al.config.TrackResponseSize {
		entry.ResponseSize = int64(rw.size)
	}

	// Add request body
	if al.config.LogRequestBody && requestBody != "" {
		entry.RequestBody = requestBody
	}

	// Add response body
	if al.config.LogResponseBody && rw.body != nil {
		responseBody := al.filterSensitiveData(rw.body.String())
		if len(responseBody) > al.config.MaxBodyLogSize {
			responseBody = responseBody[:al.config.MaxBodyLogSize] + "...[truncated]"
		}
		entry.ResponseBody = responseBody
	}

	// Add error if present
	if len(c.Errors) > 0 {
		entry.Error = c.Errors.String()
	}

	return entry
}

// logEntry writes a log entry based on configured format
func (al *AccessLogger) logEntry(entry AccessLogEntry) {
	if al.logger == nil {
		return
	}

	var logLine string

	switch al.config.LogFormat {
	case "json":
		jsonEntry, err := json.Marshal(entry)
		if err != nil {
			log.Printf("[ERROR] Failed to marshal log entry: %v", err)
			return
		}
		logLine = string(jsonEntry)

	case "combined":
		// Apache combined log format
		logLine = fmt.Sprintf("%s - %s [%s] \"%s %s %s\" %d %d \"%s\" \"%s\" %dms",
			entry.ClientIP,
			entry.UserID,
			entry.Timestamp.Format("02/Jan/2006:15:04:05 -0700"),
			entry.Method,
			entry.Path,
			"HTTP/1.1",
			entry.StatusCode,
			entry.ResponseSize,
			"-", // Referer (not captured)
			entry.UserAgent,
			entry.ResponseTime,
		)

	default: // "text"
		logLine = fmt.Sprintf("[%s] %s | %d | %dms | %s | %s %s",
			entry.Timestamp.Format("2006-01-02 15:04:05"),
			entry.RequestID,
			entry.StatusCode,
			entry.ResponseTime,
			entry.ClientIP,
			entry.Method,
			entry.Path,
		)
		if entry.UserID != "" {
			logLine += fmt.Sprintf(" | User: %s", entry.UserID)
		}
		if len(entry.Tags) > 0 {
			logLine += fmt.Sprintf(" | Tags: %s", strings.Join(entry.Tags, ","))
		}
	}

	al.logger.Println(logLine)
}

// updateMetrics updates the metrics based on the request
func (al *AccessLogger) updateMetrics(entry AccessLogEntry, duration time.Duration, rw *responseWriter) {
	al.metricsLock.Lock()
	defer al.metricsLock.Unlock()

	al.metrics.TotalRequests++

	// Status-based metrics
	if rw.statusCode >= 200 && rw.statusCode < 400 {
		al.metrics.SuccessfulRequests++
	} else if rw.statusCode >= 400 {
		al.metrics.FailedRequests++
	}

	if rw.statusCode == 403 {
		al.metrics.DeniedRequests++
	}

	// Timing metrics
	if al.config.TrackResponseTime {
		al.metrics.TotalResponseTime += duration
	}

	if duration > al.config.SlowRequestThreshold {
		al.metrics.SlowRequests++
	}

	// Size metrics
	if al.config.TrackRequestSize {
		al.metrics.TotalRequestSize += entry.RequestSize
	}

	if al.config.TrackResponseSize {
		al.metrics.TotalResponseSize += entry.ResponseSize
	}

	// Method breakdown
	al.metrics.RequestsByMethod[entry.Method]++

	// Path breakdown (limit to top paths to avoid unbounded growth)
	if len(al.metrics.RequestsByPath) < 1000 {
		al.metrics.RequestsByPath[entry.Path]++
	}

	// Status breakdown
	al.metrics.RequestsByStatus[entry.StatusCode]++
}

// filterSensitiveData removes or masks sensitive data from log content
func (al *AccessLogger) filterSensitiveData(content string) string {
	if !al.config.FilterSensitiveData {
		return content
	}

	filtered := content

	// Filter sensitive parameters
	for _, param := range al.config.SensitiveParams {
		// Simple replacement - could be enhanced with regex
		filtered = strings.ReplaceAll(filtered, param, "[FILTERED]")
	}

	return filtered
}
