package email

import (
	"context"
	"fmt"
	"math"
	"math/rand"
	"sync"
	"sync/atomic"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ses"
	"github.com/aws/aws-sdk-go-v2/service/ses/types"
)

// EmailPriority represents the priority level of an email
type EmailPriority int

const (
	PriorityLow EmailPriority = iota
	PriorityNormal
	PriorityHigh
	PriorityCritical
)

// Email represents an email message to be sent
type Email struct {
	// Recipients
	To      []string
	CC      []string
	BCC     []string
	ReplyTo []string

	// Content
	Subject  string
	TextBody string
	HTMLBody string

	// Sender (optional, uses default if empty)
	From     string
	FromName string

	// Optional
	Headers     map[string]string
	Tags        map[string]string
	Attachments []Attachment
	Priority    EmailPriority

	// Template
	TemplateName string
	TemplateData map[string]interface{}

	// Tracking
	ConfigurationSetName string
	MessageID            string
}

// Attachment represents an email attachment
type Attachment struct {
	Filename    string
	ContentType string
	Data        []byte
	ContentID   string // For inline attachments
}

// SendResult represents the result of sending an email
type SendResult struct {
	MessageID string
	RequestID string
	Success   bool
	Error     error
	Attempts  int
	Duration  time.Duration
	Timestamp time.Time
}

// BatchSendResult represents the result of a batch send operation
type BatchSendResult struct {
	Total      int
	Successful int
	Failed     int
	Results    []SendResult
	Duration   time.Duration
}

// SESClient wraps the AWS SES client with additional functionality
type SESClient struct {
	client       *ses.Client
	config       *SESConfig
	logger       *SESLogger
	metrics      *SESMetrics
	errorHandler *SESErrorHandler

	// Rate limiting
	rateLimiter *rateLimiter

	// Statistics
	stats *clientStats

	// State
	mu     sync.RWMutex
	closed bool
}

// clientStats tracks client statistics
type clientStats struct {
	totalSent    int64
	totalFailed  int64
	totalRetries int64
	lastSentAt   atomic.Value // time.Time
	lastErrorAt  atomic.Value // time.Time
}

// rateLimiter implements token bucket rate limiting
type rateLimiter struct {
	mu         sync.Mutex
	tokens     float64
	maxTokens  float64
	refillRate float64 // tokens per second
	lastRefill time.Time
}

// NewSESClient creates a new SES client with the given configuration
func NewSESClient(ctx context.Context, config *SESConfig) (*SESClient, error) {
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	// Create AWS SES client
	awsClient, err := config.CreateSESClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create SES client: %w", err)
	}

	// Create logger
	logger := NewSESLogger(config)

	// Create metrics (may be nil if metrics disabled)
	var metrics *SESMetrics
	if config.MetricsEnabled {
		metrics, err = NewSESMetrics(ctx, config)
		if err != nil {
			logger.Warn("Failed to create metrics client, continuing without metrics", "error", err)
		}
	}

	// Create error handler
	errorHandler := NewSESErrorHandler(config)

	client := &SESClient{
		client:       awsClient,
		config:       config,
		logger:       logger,
		metrics:      metrics,
		errorHandler: errorHandler,
		rateLimiter:  newRateLimiter(config.MaxSendRate),
		stats:        &clientStats{},
	}

	logger.Info("SES client initialized",
		"region", config.Region,
		"environment", config.Environment,
		"sandbox", config.SandboxMode,
	)

	return client, nil
}

// Send sends a single email with retry logic
func (c *SESClient) Send(ctx context.Context, email *Email) (*SendResult, error) {
	c.mu.RLock()
	if c.closed {
		c.mu.RUnlock()
		return nil, ErrClientClosed
	}
	c.mu.RUnlock()

	startTime := time.Now()
	result := &SendResult{
		Timestamp: startTime,
	}

	// Validate email
	if err := c.validateEmail(email); err != nil {
		result.Error = err
		return result, err
	}

	// Apply rate limiting
	if err := c.rateLimiter.wait(ctx); err != nil {
		result.Error = err
		return result, err
	}

	// Build the SES request
	input, err := c.buildSendEmailInput(email)
	if err != nil {
		result.Error = err
		return result, err
	}

	// Send with retry logic
	var lastErr error
	for attempt := 0; attempt <= c.config.MaxRetries; attempt++ {
		result.Attempts = attempt + 1

		// Check context
		if ctx.Err() != nil {
			result.Error = ctx.Err()
			return result, ctx.Err()
		}

		// Send the email
		output, err := c.client.SendEmail(ctx, input)
		if err != nil {
			lastErr = err

			// Check if error is retryable
			if !c.errorHandler.IsRetryable(err) {
				c.logger.Error("Non-retryable error sending email",
					"error", err,
					"to", email.To,
					"attempt", attempt+1,
				)
				result.Error = err
				atomic.AddInt64(&c.stats.totalFailed, 1)
				c.stats.lastErrorAt.Store(time.Now())

				if c.metrics != nil {
					c.metrics.RecordSendFailure(ctx, err)
				}
				return result, c.errorHandler.Wrap(err)
			}

			// Calculate backoff delay
			delay := c.calculateBackoff(attempt)
			c.logger.Warn("Retrying email send",
				"error", err,
				"attempt", attempt+1,
				"maxRetries", c.config.MaxRetries,
				"delay", delay,
			)

			atomic.AddInt64(&c.stats.totalRetries, 1)

			// Wait before retry
			select {
			case <-ctx.Done():
				result.Error = ctx.Err()
				return result, ctx.Err()
			case <-time.After(delay):
				continue
			}
		}

		// Success
		result.MessageID = aws.ToString(output.MessageId)
		result.Success = true
		result.Duration = time.Since(startTime)

		atomic.AddInt64(&c.stats.totalSent, 1)
		c.stats.lastSentAt.Store(time.Now())

		c.logger.Info("Email sent successfully",
			"messageID", result.MessageID,
			"to", email.To,
			"attempts", result.Attempts,
			"duration", result.Duration,
		)

		if c.metrics != nil {
			c.metrics.RecordSendSuccess(ctx, result.Duration)
		}

		return result, nil
	}

	// All retries exhausted
	result.Error = lastErr
	result.Duration = time.Since(startTime)
	atomic.AddInt64(&c.stats.totalFailed, 1)
	c.stats.lastErrorAt.Store(time.Now())

	c.logger.Error("All retry attempts exhausted",
		"error", lastErr,
		"to", email.To,
		"attempts", result.Attempts,
	)

	if c.metrics != nil {
		c.metrics.RecordSendFailure(ctx, lastErr)
	}

	return result, c.errorHandler.Wrap(lastErr)
}

// SendBatch sends multiple emails with concurrency control
func (c *SESClient) SendBatch(ctx context.Context, emails []*Email, concurrency int) *BatchSendResult {
	startTime := time.Now()
	result := &BatchSendResult{
		Total:   len(emails),
		Results: make([]SendResult, len(emails)),
	}

	if concurrency <= 0 {
		concurrency = 5
	}

	// Create a semaphore for concurrency control
	sem := make(chan struct{}, concurrency)
	var wg sync.WaitGroup
	var mu sync.Mutex

	for i, email := range emails {
		wg.Add(1)
		go func(idx int, e *Email) {
			defer wg.Done()

			// Acquire semaphore
			select {
			case sem <- struct{}{}:
				defer func() { <-sem }()
			case <-ctx.Done():
				mu.Lock()
				result.Results[idx] = SendResult{
					Error:     ctx.Err(),
					Timestamp: time.Now(),
				}
				result.Failed++
				mu.Unlock()
				return
			}

			// Send the email
			sendResult, _ := c.Send(ctx, e)

			mu.Lock()
			result.Results[idx] = *sendResult
			if sendResult.Success {
				result.Successful++
			} else {
				result.Failed++
			}
			mu.Unlock()
		}(i, email)
	}

	wg.Wait()
	result.Duration = time.Since(startTime)

	c.logger.Info("Batch send completed",
		"total", result.Total,
		"successful", result.Successful,
		"failed", result.Failed,
		"duration", result.Duration,
	)

	return result
}

// SendTemplated sends an email using an SES template
func (c *SESClient) SendTemplated(ctx context.Context, email *Email) (*SendResult, error) {
	if email.TemplateName == "" {
		return nil, ErrTemplateRequired
	}

	c.mu.RLock()
	if c.closed {
		c.mu.RUnlock()
		return nil, ErrClientClosed
	}
	c.mu.RUnlock()

	startTime := time.Now()
	result := &SendResult{
		Timestamp: startTime,
	}

	// Apply rate limiting
	if err := c.rateLimiter.wait(ctx); err != nil {
		result.Error = err
		return result, err
	}

	// Build template data JSON
	templateData, err := c.buildTemplateData(email.TemplateData)
	if err != nil {
		result.Error = err
		return result, err
	}

	// Get from address
	fromAddr := email.From
	if fromAddr == "" {
		fromAddr = c.config.GetFromAddress()
	}

	// Build input
	input := &ses.SendTemplatedEmailInput{
		Source:       aws.String(fromAddr),
		Template:     aws.String(c.config.TemplatePrefix + email.TemplateName),
		TemplateData: aws.String(templateData),
		Destination: &types.Destination{
			ToAddresses:  email.To,
			CcAddresses:  email.CC,
			BccAddresses: email.BCC,
		},
	}

	if c.config.ConfigurationSetName != "" {
		input.ConfigurationSetName = aws.String(c.config.ConfigurationSetName)
	}

	// Add tags
	if len(email.Tags) > 0 {
		var tags []types.MessageTag
		for k, v := range email.Tags {
			tags = append(tags, types.MessageTag{
				Name:  aws.String(k),
				Value: aws.String(v),
			})
		}
		input.Tags = tags
	}

	// Send with retry logic
	var lastErr error
	for attempt := 0; attempt <= c.config.MaxRetries; attempt++ {
		result.Attempts = attempt + 1

		output, err := c.client.SendTemplatedEmail(ctx, input)
		if err != nil {
			lastErr = err

			if !c.errorHandler.IsRetryable(err) {
				result.Error = err
				return result, c.errorHandler.Wrap(err)
			}

			delay := c.calculateBackoff(attempt)
			select {
			case <-ctx.Done():
				result.Error = ctx.Err()
				return result, ctx.Err()
			case <-time.After(delay):
				continue
			}
		}

		result.MessageID = aws.ToString(output.MessageId)
		result.Success = true
		result.Duration = time.Since(startTime)

		c.logger.Info("Templated email sent successfully",
			"messageID", result.MessageID,
			"template", email.TemplateName,
			"to", email.To,
		)

		return result, nil
	}

	result.Error = lastErr
	result.Duration = time.Since(startTime)
	return result, c.errorHandler.Wrap(lastErr)
}

// SendRaw sends a raw email (for advanced use cases with MIME)
func (c *SESClient) SendRaw(ctx context.Context, rawMessage []byte, from string, to []string) (*SendResult, error) {
	c.mu.RLock()
	if c.closed {
		c.mu.RUnlock()
		return nil, ErrClientClosed
	}
	c.mu.RUnlock()

	startTime := time.Now()
	result := &SendResult{
		Timestamp: startTime,
	}

	// Apply rate limiting
	if err := c.rateLimiter.wait(ctx); err != nil {
		result.Error = err
		return result, err
	}

	input := &ses.SendRawEmailInput{
		RawMessage: &types.RawMessage{
			Data: rawMessage,
		},
		Source:       aws.String(from),
		Destinations: to,
	}

	if c.config.ConfigurationSetName != "" {
		input.ConfigurationSetName = aws.String(c.config.ConfigurationSetName)
	}

	// Send with retry logic
	var lastErr error
	for attempt := 0; attempt <= c.config.MaxRetries; attempt++ {
		result.Attempts = attempt + 1

		output, err := c.client.SendRawEmail(ctx, input)
		if err != nil {
			lastErr = err

			if !c.errorHandler.IsRetryable(err) {
				result.Error = err
				return result, c.errorHandler.Wrap(err)
			}

			delay := c.calculateBackoff(attempt)
			select {
			case <-ctx.Done():
				result.Error = ctx.Err()
				return result, ctx.Err()
			case <-time.After(delay):
				continue
			}
		}

		result.MessageID = aws.ToString(output.MessageId)
		result.Success = true
		result.Duration = time.Since(startTime)

		return result, nil
	}

	result.Error = lastErr
	result.Duration = time.Since(startTime)
	return result, c.errorHandler.Wrap(lastErr)
}

// GetSendQuota returns the current send quota information
func (c *SESClient) GetSendQuota(ctx context.Context) (*ses.GetSendQuotaOutput, error) {
	return c.client.GetSendQuota(ctx, &ses.GetSendQuotaInput{})
}

// GetSendStatistics returns send statistics for the past two weeks
func (c *SESClient) GetSendStatistics(ctx context.Context) (*ses.GetSendStatisticsOutput, error) {
	return c.client.GetSendStatistics(ctx, &ses.GetSendStatisticsInput{})
}

// GetStats returns client statistics
func (c *SESClient) GetStats() (sent, failed, retries int64, lastSent, lastError time.Time) {
	sent = atomic.LoadInt64(&c.stats.totalSent)
	failed = atomic.LoadInt64(&c.stats.totalFailed)
	retries = atomic.LoadInt64(&c.stats.totalRetries)

	if t, ok := c.stats.lastSentAt.Load().(time.Time); ok {
		lastSent = t
	}
	if t, ok := c.stats.lastErrorAt.Load().(time.Time); ok {
		lastError = t
	}
	return
}

// Close closes the SES client
func (c *SESClient) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed {
		return nil
	}

	c.closed = true
	c.logger.Info("SES client closed")
	return nil
}

// HealthCheck performs a health check on the SES service
func (c *SESClient) HealthCheck(ctx context.Context) error {
	_, err := c.GetSendQuota(ctx)
	if err != nil {
		return fmt.Errorf("SES health check failed: %w", err)
	}
	return nil
}

// Private helper methods

func (c *SESClient) validateEmail(email *Email) error {
	if len(email.To) == 0 {
		return ErrNoRecipients
	}

	if email.Subject == "" {
		return ErrEmptySubject
	}

	if email.TextBody == "" && email.HTMLBody == "" {
		return ErrEmptyBody
	}

	// Validate email addresses
	for _, addr := range email.To {
		if !isValidEmail(addr) {
			return fmt.Errorf("%w: %s", ErrInvalidEmail, addr)
		}
	}

	return nil
}

func (c *SESClient) buildSendEmailInput(email *Email) (*ses.SendEmailInput, error) {
	// Get from address
	fromAddr := email.From
	if fromAddr == "" {
		fromAddr = c.config.GetFromAddress()
	} else if email.FromName != "" {
		fromAddr = fmt.Sprintf("%s <%s>", email.FromName, email.From)
	}

	// Build destination
	destination := &types.Destination{
		ToAddresses:  email.To,
		CcAddresses:  email.CC,
		BccAddresses: email.BCC,
	}

	// Build message body
	body := &types.Body{}
	if email.TextBody != "" {
		body.Text = &types.Content{
			Charset: aws.String("UTF-8"),
			Data:    aws.String(email.TextBody),
		}
	}
	if email.HTMLBody != "" {
		body.Html = &types.Content{
			Charset: aws.String("UTF-8"),
			Data:    aws.String(email.HTMLBody),
		}
	}

	// Build message
	message := &types.Message{
		Subject: &types.Content{
			Charset: aws.String("UTF-8"),
			Data:    aws.String(email.Subject),
		},
		Body: body,
	}

	// Build input
	input := &ses.SendEmailInput{
		Source:      aws.String(fromAddr),
		Destination: destination,
		Message:     message,
	}

	// Add reply-to
	if len(email.ReplyTo) > 0 {
		input.ReplyToAddresses = email.ReplyTo
	} else if c.config.DefaultReplyTo != "" {
		input.ReplyToAddresses = []string{c.config.DefaultReplyTo}
	}

	// Add configuration set
	configSetName := email.ConfigurationSetName
	if configSetName == "" {
		configSetName = c.config.ConfigurationSetName
	}
	if configSetName != "" {
		input.ConfigurationSetName = aws.String(configSetName)
	}

	// Add tags
	if len(email.Tags) > 0 {
		var tags []types.MessageTag
		for k, v := range email.Tags {
			tags = append(tags, types.MessageTag{
				Name:  aws.String(k),
				Value: aws.String(v),
			})
		}
		input.Tags = tags
	}

	return input, nil
}

func (c *SESClient) buildTemplateData(data map[string]interface{}) (string, error) {
	if data == nil {
		return "{}", nil
	}

	// Simple JSON encoding (in production, use encoding/json)
	result := "{"
	first := true
	for k, v := range data {
		if !first {
			result += ","
		}
		first = false
		result += fmt.Sprintf(`"%s":"%v"`, k, v)
	}
	result += "}"
	return result, nil
}

func (c *SESClient) calculateBackoff(attempt int) time.Duration {
	delay := float64(c.config.RetryBaseDelay) * math.Pow(c.config.RetryMultiplier, float64(attempt))

	// Add jitter (±10%)
	jitter := delay * 0.1 * (rand.Float64()*2 - 1)
	delay += jitter

	if delay > float64(c.config.RetryMaxDelay) {
		delay = float64(c.config.RetryMaxDelay)
	}

	return time.Duration(delay)
}

// Rate limiter implementation

func newRateLimiter(rate float64) *rateLimiter {
	return &rateLimiter{
		tokens:     rate,
		maxTokens:  rate,
		refillRate: rate,
		lastRefill: time.Now(),
	}
}

func (r *rateLimiter) wait(ctx context.Context) error {
	r.mu.Lock()

	// Refill tokens based on elapsed time
	now := time.Now()
	elapsed := now.Sub(r.lastRefill).Seconds()
	r.tokens = math.Min(r.maxTokens, r.tokens+elapsed*r.refillRate)
	r.lastRefill = now

	// Wait if no tokens available
	if r.tokens < 1 {
		waitTime := time.Duration((1 - r.tokens) / r.refillRate * float64(time.Second))
		r.mu.Unlock()

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(waitTime):
		}

		r.mu.Lock()
		// Recalculate tokens after waiting
		now = time.Now()
		elapsed = now.Sub(r.lastRefill).Seconds()
		r.tokens = math.Min(r.maxTokens, r.tokens+elapsed*r.refillRate)
		r.lastRefill = now
	}

	// Consume a token
	r.tokens--
	r.mu.Unlock()
	return nil
}

// Email validation helper

func isValidEmail(email string) bool {
	// Basic email validation - in production use a proper validator
	if len(email) < 3 || len(email) > 254 {
		return false
	}

	atIndex := -1
	for i, c := range email {
		if c == '@' {
			if atIndex != -1 {
				return false // Multiple @ signs
			}
			atIndex = i
		}
	}

	if atIndex < 1 || atIndex >= len(email)-1 {
		return false
	}

	return true
}
