package cloudwatch

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs/types"

	"github.com/javaknight1/servicepro/backend/config"
	"github.com/javaknight1/servicepro/backend/pkg/clients/logging"
)

func init() {
	logging.RegisterProvider(logging.ProviderCloudWatch, func(ctx context.Context, cfg *config.Config) (logging.Client, error) {
		cwCfg := &Config{
			Region:          cfg.AWS.Region,
			AccessKeyID:     cfg.AWS.AccessKeyID,
			SecretAccessKey: cfg.AWS.SecretAccessKey,
			LogGroupName:    fmt.Sprintf("/servicepro/%s", cfg.Server.Env),
			LogStreamName:   fmt.Sprintf("app-%s", time.Now().Format("2006-01-02")),
		}
		return NewClient(ctx, cwCfg, cfg)
	})
}

// Config holds configuration for the CloudWatch Logs client
type Config struct {
	// Region is the AWS region (required)
	Region string

	// AccessKeyID for AWS authentication (optional, uses default chain if empty)
	AccessKeyID string

	// SecretAccessKey for AWS authentication (optional, uses default chain if empty)
	SecretAccessKey string

	// LogGroupName is the CloudWatch Logs group name (required)
	LogGroupName string

	// LogStreamName is the CloudWatch Logs stream name (required)
	LogStreamName string

	// CreateLogGroup creates the log group if it doesn't exist (default: true)
	CreateLogGroup bool

	// CreateLogStream creates the log stream if it doesn't exist (default: true)
	CreateLogStream bool
}

// Client implements the logging.Client interface using CloudWatch Logs
type Client struct {
	config        *Config
	appConfig     *config.Config
	client        *cloudwatchlogs.Client
	mu            sync.Mutex
	buffer        []*logging.LogEntry
	flushTicker   *time.Ticker
	stopChan      chan struct{}
	wg            sync.WaitGroup
	sequenceToken *string
	defaultFields map[string]any
	source        string
}

// NewClient creates a new CloudWatch Logs client
func NewClient(ctx context.Context, cfg *Config, appCfg *config.Config) (*Client, error) {
	if cfg.Region == "" {
		return nil, fmt.Errorf("AWS region is required")
	}

	if cfg.LogGroupName == "" {
		return nil, fmt.Errorf("log group name is required")
	}

	if cfg.LogStreamName == "" {
		return nil, fmt.Errorf("log stream name is required")
	}

	// Build AWS config options
	var opts []func(*awsconfig.LoadOptions) error
	opts = append(opts, awsconfig.WithRegion(cfg.Region))

	if cfg.AccessKeyID != "" && cfg.SecretAccessKey != "" {
		opts = append(opts, awsconfig.WithCredentialsProvider(
			credentials.NewStaticCredentialsProvider(
				cfg.AccessKeyID,
				cfg.SecretAccessKey,
				"",
			),
		))
	}

	awsCfg, err := awsconfig.LoadDefaultConfig(ctx, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	client := cloudwatchlogs.NewFromConfig(awsCfg)

	batchOpts := logging.DefaultBatchOptions()

	c := &Client{
		config:        cfg,
		appConfig:     appCfg,
		client:        client,
		buffer:        make([]*logging.LogEntry, 0, batchOpts.MaxSize),
		stopChan:      make(chan struct{}),
		defaultFields: make(map[string]any),
	}

	// Ensure log group and stream exist
	if err := c.ensureLogGroupAndStream(ctx); err != nil {
		return nil, err
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
		client:        c.client,
		buffer:        c.buffer,
		flushTicker:   c.flushTicker,
		stopChan:      c.stopChan,
		sequenceToken: c.sequenceToken,
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
		client:        c.client,
		buffer:        c.buffer,
		flushTicker:   c.flushTicker,
		stopChan:      c.stopChan,
		sequenceToken: c.sequenceToken,
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

	return c.sendEntries(context.Background(), entries)
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

func (c *Client) ensureLogGroupAndStream(ctx context.Context) error {
	// Create log group if it doesn't exist
	_, err := c.client.CreateLogGroup(ctx, &cloudwatchlogs.CreateLogGroupInput{
		LogGroupName: aws.String(c.config.LogGroupName),
	})
	if err != nil {
		// Ignore if already exists
		if !isResourceAlreadyExistsException(err) {
			return fmt.Errorf("failed to create log group: %w", err)
		}
	}

	// Create log stream if it doesn't exist
	_, err = c.client.CreateLogStream(ctx, &cloudwatchlogs.CreateLogStreamInput{
		LogGroupName:  aws.String(c.config.LogGroupName),
		LogStreamName: aws.String(c.config.LogStreamName),
	})
	if err != nil {
		if !isResourceAlreadyExistsException(err) {
			return fmt.Errorf("failed to create log stream: %w", err)
		}
	}

	return nil
}

func (c *Client) sendEntries(ctx context.Context, entries []*logging.LogEntry) error {
	if len(entries) == 0 {
		return nil
	}

	// Convert to CloudWatch log events
	logEvents := make([]types.InputLogEvent, len(entries))
	for i, entry := range entries {
		// Convert entry to JSON message
		msg := map[string]any{
			"level":   entry.Level.String(),
			"message": entry.Message,
		}

		if entry.Source != "" {
			msg["source"] = entry.Source
		}

		if entry.TraceID != "" {
			msg["trace_id"] = entry.TraceID
		}

		if entry.SpanID != "" {
			msg["span_id"] = entry.SpanID
		}

		if entry.Error != nil {
			msg["error"] = entry.Error
		}

		if entry.HTTP != nil {
			msg["http"] = entry.HTTP
		}

		if entry.User != nil {
			msg["user"] = entry.User
		}

		for k, v := range entry.Fields {
			msg[k] = v
		}

		jsonMsg, _ := json.Marshal(msg)

		logEvents[i] = types.InputLogEvent{
			Message:   aws.String(string(jsonMsg)),
			Timestamp: aws.Int64(entry.Timestamp.UnixMilli()),
		}
	}

	// Send to CloudWatch
	input := &cloudwatchlogs.PutLogEventsInput{
		LogGroupName:  aws.String(c.config.LogGroupName),
		LogStreamName: aws.String(c.config.LogStreamName),
		LogEvents:     logEvents,
	}

	if c.sequenceToken != nil {
		input.SequenceToken = c.sequenceToken
	}

	result, err := c.client.PutLogEvents(ctx, input)
	if err != nil {
		return fmt.Errorf("failed to send logs to CloudWatch: %w", err)
	}

	// Store the next sequence token
	c.sequenceToken = result.NextSequenceToken

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

func isResourceAlreadyExistsException(err error) bool {
	if err == nil {
		return false
	}
	// Check if error message contains "ResourceAlreadyExistsException"
	return contains(err.Error(), "ResourceAlreadyExistsException")
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsAt(s, substr, 0))
}

func containsAt(s, substr string, start int) bool {
	for i := start; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// Ensure Client implements logging.Client
var _ logging.Client = (*Client)(nil)
