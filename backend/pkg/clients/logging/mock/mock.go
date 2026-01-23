package mock

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/javaknight1/servicepro/backend/config"
	"github.com/javaknight1/servicepro/backend/pkg/clients/logging"
)

func init() {
	logging.RegisterProvider(logging.ProviderMock, func(ctx context.Context, cfg *config.Config) (logging.Client, error) {
		mockCfg := &Config{
			PrintToStdout: true,
			JSONFormat:    false,
		}
		return NewClient(mockCfg, cfg), nil
	})
}

// Config holds configuration for the mock logging client
type Config struct {
	// PrintToStdout prints logs to stdout (default: true)
	PrintToStdout bool

	// JSONFormat uses JSON format for stdout (default: false)
	JSONFormat bool

	// StoreEntries stores entries in memory for testing (default: false)
	StoreEntries bool
}

// Client implements the logging.Client interface for testing and development
type Client struct {
	config        *Config
	appConfig     *config.Config
	mu            sync.Mutex
	entries       []*logging.LogEntry
	defaultFields map[string]any
	source        string
}

// NewClient creates a new mock logging client
func NewClient(cfg *Config, appCfg *config.Config) *Client {
	if cfg == nil {
		cfg = &Config{
			PrintToStdout: true,
			JSONFormat:    false,
		}
	}

	return &Client{
		config:        cfg,
		appConfig:     appCfg,
		entries:       make([]*logging.LogEntry, 0),
		defaultFields: make(map[string]any),
	}
}

// Log implements logging.Client
func (c *Client) Log(ctx context.Context, entry *logging.LogEntry) error {
	// Apply default fields
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

	// Store entry if configured
	if c.config.StoreEntries {
		c.mu.Lock()
		c.entries = append(c.entries, entry)
		c.mu.Unlock()
	}

	// Print to stdout if configured
	if c.config.PrintToStdout {
		c.printEntry(entry)
	}

	return nil
}

// LogBatch implements logging.Client
func (c *Client) LogBatch(ctx context.Context, entries []*logging.LogEntry) error {
	for _, entry := range entries {
		if err := c.Log(ctx, entry); err != nil {
			return err
		}
	}
	return nil
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
		entries:       c.entries,
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
		entries:       c.entries,
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
	return nil
}

// Close implements logging.Client
func (c *Client) Close() error {
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

func (c *Client) printEntry(entry *logging.LogEntry) {
	if c.config.JSONFormat {
		data, _ := json.Marshal(entry)
		log.Println(string(data))
	} else {
		timestamp := entry.Timestamp.Format(time.RFC3339)
		level := entry.Level.String()
		source := entry.Source
		if source == "" {
			source = "app"
		}

		msg := fmt.Sprintf("[%s] %s [%s] %s", timestamp, level, source, entry.Message)

		if len(entry.Fields) > 0 {
			fieldsJSON, _ := json.Marshal(entry.Fields)
			msg += " " + string(fieldsJSON)
		}

		if entry.Error != nil {
			msg += fmt.Sprintf(" error=%q", entry.Error.Message)
		}

		log.Println(msg)
	}
}

// GetEntries returns all stored log entries (for testing)
func (c *Client) GetEntries() []*logging.LogEntry {
	c.mu.Lock()
	defer c.mu.Unlock()

	entries := make([]*logging.LogEntry, len(c.entries))
	copy(entries, c.entries)
	return entries
}

// ClearEntries clears all stored log entries (for testing)
func (c *Client) ClearEntries() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.entries = make([]*logging.LogEntry, 0)
}

// Ensure Client implements logging.Client
var _ logging.Client = (*Client)(nil)
