package loki

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strconv"
	"sync"
	"time"

	"github.com/javaknight1/servicepro/backend/config"
	"github.com/javaknight1/servicepro/backend/pkg/clients/logging"
)

func init() {
	logging.RegisterProvider(logging.ProviderLoki, func(ctx context.Context, cfg *config.Config) (logging.Client, error) {
		lokiCfg := &Config{
			Endpoint: cfg.Loki.Endpoint,
			TenantID: cfg.Loki.TenantID,
			Username: cfg.Loki.Username,
			Password: cfg.Loki.Password,
		}
		return NewClient(lokiCfg, cfg)
	})
}

const (
	// DefaultEndpoint is a common Loki push endpoint
	DefaultPushPath = "/loki/api/v1/push"
)

// Config holds configuration for the Loki logging client
type Config struct {
	// Endpoint is the Loki push API endpoint (required)
	// e.g., "http://localhost:3100" or "https://logs-prod-us-central1.grafana.net"
	Endpoint string

	// TenantID is the X-Scope-OrgID header value for multi-tenant Loki (optional)
	TenantID string

	// Username for basic auth (optional, used with Grafana Cloud)
	Username string

	// Password/API key for basic auth (optional, used with Grafana Cloud)
	Password string

	// HTTPTimeout is the timeout for HTTP requests (default: 10s)
	HTTPTimeout time.Duration

	// Labels are static labels added to all log streams
	Labels map[string]string
}

// Client implements the logging.Client interface using Grafana Loki
type Client struct {
	config        *Config
	appConfig     *config.Config
	httpClient    *http.Client
	mu            sync.Mutex
	buffer        []*logging.LogEntry
	flushTicker   *time.Ticker
	stopChan      chan struct{}
	wg            sync.WaitGroup
	defaultFields map[string]any
	source        string
}

// NewClient creates a new Loki logging client
func NewClient(cfg *Config, appCfg *config.Config) (*Client, error) {
	if cfg.Endpoint == "" {
		return nil, fmt.Errorf("Loki endpoint is required")
	}

	if cfg.HTTPTimeout == 0 {
		cfg.HTTPTimeout = 10 * time.Second
	}

	if cfg.Labels == nil {
		cfg.Labels = make(map[string]string)
	}

	batchOpts := logging.DefaultBatchOptions()

	c := &Client{
		config:    cfg,
		appConfig: appCfg,
		httpClient: &http.Client{
			Timeout: cfg.HTTPTimeout,
		},
		buffer:        make([]*logging.LogEntry, 0, batchOpts.MaxSize),
		stopChan:      make(chan struct{}),
		defaultFields: make(map[string]any),
	}

	// Start background flush ticker
	c.flushTicker = time.NewTicker(batchOpts.FlushInterval)
	c.wg.Add(1)
	go c.flushLoop()

	return c, nil
}

// Log implements logging.Client
func (c *Client) Log(ctx context.Context, entry *logging.LogEntry) error {
	c.applyDefaults(entry)

	// Extract trace info from context
	traceID, spanID := logging.ExtractTraceInfo(ctx)
	if traceID != "" && entry.TraceID == "" {
		entry.TraceID = traceID
	}
	if spanID != "" && entry.SpanID == "" {
		entry.SpanID = spanID
	}

	// Extract user info from context
	if entry.User == nil {
		entry.User = logging.ExtractUserInfo(ctx)
	}

	c.mu.Lock()
	c.buffer = append(c.buffer, entry)
	shouldFlush := len(c.buffer) >= c.getBatchSize()
	c.mu.Unlock()

	if shouldFlush {
		return c.Flush()
	}

	return nil
}

// LogBatch implements logging.Client
func (c *Client) LogBatch(ctx context.Context, entries []*logging.LogEntry) error {
	for _, entry := range entries {
		c.applyDefaults(entry)

		traceID, spanID := logging.ExtractTraceInfo(ctx)
		if traceID != "" && entry.TraceID == "" {
			entry.TraceID = traceID
		}
		if spanID != "" && entry.SpanID == "" {
			entry.SpanID = spanID
		}

		if entry.User == nil {
			entry.User = logging.ExtractUserInfo(ctx)
		}

		c.mu.Lock()
		c.buffer = append(c.buffer, entry)
		c.mu.Unlock()
	}

	return c.Flush()
}

// Debug implements logging.Client
func (c *Client) Debug(ctx context.Context, msg string, fields map[string]any) {
	entry := logging.NewLogEntry(logging.LevelDebug, msg).WithFields(fields)
	c.Log(ctx, entry)
}

// Info implements logging.Client
func (c *Client) Info(ctx context.Context, msg string, fields map[string]any) {
	entry := logging.NewLogEntry(logging.LevelInfo, msg).WithFields(fields)
	c.Log(ctx, entry)
}

// Warn implements logging.Client
func (c *Client) Warn(ctx context.Context, msg string, fields map[string]any) {
	entry := logging.NewLogEntry(logging.LevelWarn, msg).WithFields(fields)
	c.Log(ctx, entry)
}

// Error implements logging.Client
func (c *Client) Error(ctx context.Context, msg string, fields map[string]any) {
	entry := logging.NewLogEntry(logging.LevelError, msg).WithFields(fields)
	c.Log(ctx, entry)
}

// Fatal implements logging.Client
func (c *Client) Fatal(ctx context.Context, msg string, fields map[string]any) {
	entry := logging.NewLogEntry(logging.LevelFatal, msg).WithFields(fields)
	c.Log(ctx, entry)
}

// WithFields implements logging.Client
func (c *Client) WithFields(fields map[string]any) logging.Client {
	newClient := &Client{
		config:        c.config,
		appConfig:     c.appConfig,
		httpClient:    c.httpClient,
		buffer:        c.buffer,
		flushTicker:   c.flushTicker,
		stopChan:      c.stopChan,
		defaultFields: make(map[string]any),
		source:        c.source,
	}
	for k, v := range c.defaultFields {
		newClient.defaultFields[k] = v
	}
	for k, v := range fields {
		newClient.defaultFields[k] = v
	}
	return newClient
}

// WithSource implements logging.Client
func (c *Client) WithSource(source string) logging.Client {
	newClient := &Client{
		config:        c.config,
		appConfig:     c.appConfig,
		httpClient:    c.httpClient,
		buffer:        c.buffer,
		flushTicker:   c.flushTicker,
		stopChan:      c.stopChan,
		defaultFields: make(map[string]any),
		source:        source,
	}
	for k, v := range c.defaultFields {
		newClient.defaultFields[k] = v
	}
	return newClient
}

// Flush implements logging.Client
func (c *Client) Flush() error {
	c.mu.Lock()
	if len(c.buffer) == 0 {
		c.mu.Unlock()
		return nil
	}

	entries := c.buffer
	c.buffer = make([]*logging.LogEntry, 0, c.getBatchSize())
	c.mu.Unlock()

	return c.sendEntries(entries)
}

// Close implements logging.Client
func (c *Client) Close() error {
	close(c.stopChan)
	c.flushTicker.Stop()
	c.wg.Wait()

	return c.Flush()
}

func (c *Client) flushLoop() {
	defer c.wg.Done()

	for {
		select {
		case <-c.flushTicker.C:
			c.Flush()
		case <-c.stopChan:
			return
		}
	}
}

// lokiPushRequest represents the Loki push API request format
type lokiPushRequest struct {
	Streams []lokiStream `json:"streams"`
}

// lokiStream represents a log stream in Loki format
type lokiStream struct {
	Stream map[string]string `json:"stream"`
	Values [][]string        `json:"values"`
}

func (c *Client) sendEntries(entries []*logging.LogEntry) error {
	if len(entries) == 0 {
		return nil
	}

	// Group entries by labels (level + source)
	streams := make(map[string]*lokiStream)

	for _, entry := range entries {
		// Build labels for this entry
		labels := make(map[string]string)

		// Add static labels from config
		for k, v := range c.config.Labels {
			labels[k] = v
		}

		// Add level label
		labels["level"] = entry.Level.String()

		// Add source label
		source := entry.Source
		if source == "" {
			source = "app"
		}
		labels["source"] = source

		// Create a key for grouping
		labelKey := labelsToKey(labels)

		stream, exists := streams[labelKey]
		if !exists {
			stream = &lokiStream{
				Stream: labels,
				Values: make([][]string, 0),
			}
			streams[labelKey] = stream
		}

		// Build the log line as JSON
		logLine := map[string]any{
			"message": entry.Message,
		}

		if entry.TraceID != "" {
			logLine["trace_id"] = entry.TraceID
		}

		if entry.SpanID != "" {
			logLine["span_id"] = entry.SpanID
		}

		if entry.Error != nil {
			logLine["error"] = entry.Error
		}

		if entry.HTTP != nil {
			logLine["http"] = entry.HTTP
		}

		if entry.User != nil {
			logLine["user"] = entry.User
		}

		for k, v := range entry.Fields {
			logLine[k] = v
		}

		lineJSON, _ := json.Marshal(logLine)

		// Loki expects [timestamp_ns, log_line]
		timestamp := strconv.FormatInt(entry.Timestamp.UnixNano(), 10)
		stream.Values = append(stream.Values, []string{timestamp, string(lineJSON)})
	}

	// Convert map to slice
	streamSlice := make([]lokiStream, 0, len(streams))
	for _, stream := range streams {
		streamSlice = append(streamSlice, *stream)
	}

	// Build request
	pushReq := lokiPushRequest{
		Streams: streamSlice,
	}

	data, err := json.Marshal(pushReq)
	if err != nil {
		return fmt.Errorf("failed to marshal log entries: %w", err)
	}

	// Send to Loki
	url := c.config.Endpoint + DefaultPushPath
	req, err := http.NewRequest("POST", url, bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	// Add tenant ID header if configured
	if c.config.TenantID != "" {
		req.Header.Set("X-Scope-OrgID", c.config.TenantID)
	}

	// Add basic auth if configured
	if c.config.Username != "" && c.config.Password != "" {
		req.SetBasicAuth(c.config.Username, c.config.Password)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send logs to Loki: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("Loki returned error %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

func (c *Client) applyDefaults(entry *logging.LogEntry) {
	if entry.Source == "" && c.source != "" {
		entry.Source = c.source
	}
	for k, v := range c.defaultFields {
		if _, exists := entry.Fields[k]; !exists {
			if entry.Fields == nil {
				entry.Fields = make(map[string]any)
			}
			entry.Fields[k] = v
		}
	}
}

func (c *Client) getBatchSize() int {
	return 100
}

// labelsToKey creates a consistent string key from labels for grouping
func labelsToKey(labels map[string]string) string {
	keys := make([]string, 0, len(labels))
	for k := range labels {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var buf bytes.Buffer
	for i, k := range keys {
		if i > 0 {
			buf.WriteByte(',')
		}
		buf.WriteString(k)
		buf.WriteByte('=')
		buf.WriteString(labels[k])
	}
	return buf.String()
}

// Ensure Client implements logging.Client
var _ logging.Client = (*Client)(nil)
