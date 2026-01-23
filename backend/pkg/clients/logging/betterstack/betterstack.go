package betterstack

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/javaknight1/servicepro/backend/config"
	"github.com/javaknight1/servicepro/backend/pkg/clients/logging"
)

func init() {
	logging.RegisterProvider(logging.ProviderBetterStack, func(ctx context.Context, cfg *config.Config) (logging.Client, error) {
		bsCfg := &Config{
			SourceToken: cfg.BetterStack.SourceToken,
		}
		return NewClient(bsCfg, cfg)
	})
}

const (
	// DefaultEndpoint is the Better Stack Logs (Logtail) ingestion endpoint
	DefaultEndpoint = "https://in.logs.betterstack.com"
)

// Config holds configuration for the Better Stack logging client
type Config struct {
	// SourceToken is the Better Stack source token (required)
	SourceToken string

	// Endpoint is the ingestion endpoint (default: DefaultEndpoint)
	Endpoint string

	// HTTPTimeout is the timeout for HTTP requests (default: 10s)
	HTTPTimeout time.Duration
}

// Client implements the logging.Client interface using Better Stack
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

// NewClient creates a new Better Stack logging client
func NewClient(cfg *Config, appCfg *config.Config) (*Client, error) {
	if cfg.SourceToken == "" {
		return nil, fmt.Errorf("Better Stack source token is required")
	}

	if cfg.Endpoint == "" {
		cfg.Endpoint = DefaultEndpoint
	}

	if cfg.HTTPTimeout == 0 {
		cfg.HTTPTimeout = 10 * time.Second
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
	// Stop the flush loop
	close(c.stopChan)
	c.flushTicker.Stop()
	c.wg.Wait()

	// Final flush
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

func (c *Client) sendEntries(entries []*logging.LogEntry) error {
	if len(entries) == 0 {
		return nil
	}

	// Convert entries to Better Stack format
	bsEntries := make([]map[string]any, len(entries))
	for i, entry := range entries {
		bsEntry := map[string]any{
			"dt":      entry.Timestamp.Format(time.RFC3339Nano),
			"level":   entry.Level.String(),
			"message": entry.Message,
		}

		if entry.Source != "" {
			bsEntry["source"] = entry.Source
		}

		if entry.TraceID != "" {
			bsEntry["trace_id"] = entry.TraceID
		}

		if entry.SpanID != "" {
			bsEntry["span_id"] = entry.SpanID
		}

		if entry.Error != nil {
			bsEntry["error"] = entry.Error
		}

		if entry.HTTP != nil {
			bsEntry["http"] = entry.HTTP
		}

		if entry.User != nil {
			bsEntry["user"] = entry.User
		}

		// Add custom fields
		for k, v := range entry.Fields {
			bsEntry[k] = v
		}

		bsEntries[i] = bsEntry
	}

	// Marshal to JSON
	data, err := json.Marshal(bsEntries)
	if err != nil {
		return fmt.Errorf("failed to marshal log entries: %w", err)
	}

	// Send to Better Stack
	req, err := http.NewRequest("POST", c.config.Endpoint, bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.config.SourceToken)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send logs to Better Stack: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("Better Stack returned error %d: %s", resp.StatusCode, string(body))
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

// Ensure Client implements logging.Client
var _ logging.Client = (*Client)(nil)
