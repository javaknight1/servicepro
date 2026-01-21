package sns

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/sns"
	"github.com/aws/aws-sdk-go-v2/service/sns/types"
)

// Standard errors
var (
	ErrNotInitialized     = errors.New("SNS client not initialized")
	ErrInvalidConfig      = errors.New("invalid SNS configuration")
	ErrTopicNotFound      = errors.New("topic not found")
	ErrTopicExists        = errors.New("topic already exists")
	ErrRateLimitExceeded  = errors.New("rate limit exceeded")
	ErrInvalidTopicName   = errors.New("invalid topic name")
	ErrInvalidPolicy      = errors.New("invalid access policy")
	ErrPublishFailed      = errors.New("failed to publish message")
	ErrSubscriptionFailed = errors.New("failed to create subscription")
)

// TopicType represents the type of SNS topic
type TopicType string

const (
	TopicTypeStandard TopicType = "standard"
	TopicTypeFIFO     TopicType = "fifo"
)

// SNSConfig holds the configuration for AWS SNS
type SNSConfig struct {
	// Topic configuration
	TopicARNs    map[string]string       `json:"topic_arns"`    // Map of topic name to ARN
	TopicConfigs map[string]*TopicConfig `json:"topic_configs"` // Topic-specific configurations

	// Rate limiting
	RateLimit       int           `json:"rate_limit"`        // Messages per second
	BurstLimit      int           `json:"burst_limit"`       // Burst capacity
	RateLimitWindow time.Duration `json:"rate_limit_window"` // Window for rate limiting

	// Retry configuration
	RetryAttempts   int           `json:"retry_attempts"`   // Number of retry attempts
	RetryDelay      time.Duration `json:"retry_delay"`      // Initial retry delay
	RetryMaxDelay   time.Duration `json:"retry_max_delay"`  // Maximum retry delay
	RetryMultiplier float64       `json:"retry_multiplier"` // Exponential backoff multiplier

	// AWS configuration
	Region          string `json:"region"`
	AccountID       string `json:"account_id"`
	AccessKeyID     string `json:"-"`                  // Secret - not serialized
	SecretAccessKey string `json:"-"`                  // Secret - not serialized
	SessionToken    string `json:"-"`                  // Optional session token
	Endpoint        string `json:"endpoint,omitempty"` // Custom endpoint (for LocalStack)

	// Logging
	EnableLogging bool   `json:"enable_logging"`
	LogLevel      string `json:"log_level"`

	// Health check
	HealthCheckInterval time.Duration `json:"health_check_interval"`
}

// TopicConfig holds configuration for a specific topic
type TopicConfig struct {
	Name              string            `json:"name"`
	DisplayName       string            `json:"display_name,omitempty"`
	Type              TopicType         `json:"type"`
	FIFOTopic         bool              `json:"fifo_topic"`
	ContentBasedDedup bool              `json:"content_based_dedup"`
	KMSKeyID          string            `json:"kms_key_id,omitempty"`
	Tags              map[string]string `json:"tags,omitempty"`
	Policy            *TopicPolicy      `json:"policy,omitempty"`
	DeliveryPolicy    *DeliveryPolicy   `json:"delivery_policy,omitempty"`
}

// TopicPolicy defines the access policy for a topic
type TopicPolicy struct {
	AllowedPrincipals []string               `json:"allowed_principals"`
	AllowedActions    []string               `json:"allowed_actions"`
	AllowedConditions map[string]interface{} `json:"allowed_conditions,omitempty"`
	DenyPrincipals    []string               `json:"deny_principals,omitempty"`
}

// DeliveryPolicy defines the delivery retry policy
type DeliveryPolicy struct {
	HealthyRetryPolicy *RetryPolicy    `json:"healthyRetryPolicy"`
	ThrottlePolicy     *ThrottlePolicy `json:"throttlePolicy,omitempty"`
}

// RetryPolicy defines retry behavior
type RetryPolicy struct {
	NumRetries         int    `json:"numRetries"`
	NumNoDelayRetries  int    `json:"numNoDelayRetries"`
	MinDelayTarget     int    `json:"minDelayTarget"`
	MaxDelayTarget     int    `json:"maxDelayTarget"`
	NumMinDelayRetries int    `json:"numMinDelayRetries"`
	NumMaxDelayRetries int    `json:"numMaxDelayRetries"`
	BackoffFunction    string `json:"backoffFunction"` // linear, arithmetic, geometric, exponential
}

// ThrottlePolicy defines throttling behavior
type ThrottlePolicy struct {
	MaxReceivesPerSecond int `json:"maxReceivesPerSecond"`
}

// DefaultSNSConfig returns a default configuration
func DefaultSNSConfig() *SNSConfig {
	return &SNSConfig{
		TopicARNs:           make(map[string]string),
		TopicConfigs:        make(map[string]*TopicConfig),
		RateLimit:           100,
		BurstLimit:          200,
		RateLimitWindow:     time.Second,
		RetryAttempts:       3,
		RetryDelay:          100 * time.Millisecond,
		RetryMaxDelay:       5 * time.Second,
		RetryMultiplier:     2.0,
		Region:              "us-east-1",
		EnableLogging:       true,
		LogLevel:            "info",
		HealthCheckInterval: 30 * time.Second,
	}
}

// Validate validates the SNS configuration
func (c *SNSConfig) Validate() error {
	if c.Region == "" {
		return fmt.Errorf("%w: region is required", ErrInvalidConfig)
	}

	if c.RateLimit <= 0 {
		return fmt.Errorf("%w: rate limit must be positive", ErrInvalidConfig)
	}

	if c.RetryAttempts < 0 {
		return fmt.Errorf("%w: retry attempts cannot be negative", ErrInvalidConfig)
	}

	if c.RetryMultiplier < 1.0 {
		return fmt.Errorf("%w: retry multiplier must be >= 1.0", ErrInvalidConfig)
	}

	// Validate topic configs
	for name, cfg := range c.TopicConfigs {
		if err := cfg.Validate(); err != nil {
			return fmt.Errorf("%w: topic %s: %v", ErrInvalidConfig, name, err)
		}
	}

	return nil
}

// Validate validates a topic configuration
func (tc *TopicConfig) Validate() error {
	if tc.Name == "" {
		return fmt.Errorf("topic name is required")
	}

	// FIFO topics must end with .fifo
	if tc.FIFOTopic && len(tc.Name) < 5 {
		return fmt.Errorf("FIFO topic name must end with .fifo")
	}
	if tc.FIFOTopic && tc.Name[len(tc.Name)-5:] != ".fifo" {
		return fmt.Errorf("FIFO topic name must end with .fifo")
	}

	return nil
}

// SNSClient wraps the AWS SNS client with additional functionality
type SNSClient struct {
	client       *sns.Client
	config       *SNSConfig
	rateLimiter  *RateLimiter
	errorHandler *ErrorHandler

	// Topic cache
	topicCache   map[string]string // name -> ARN
	topicCacheMu sync.RWMutex

	// Subscription cache
	subscriptionCache   map[string][]string // topic ARN -> subscription ARNs
	subscriptionCacheMu sync.RWMutex

	// Status
	initialized bool
	mu          sync.RWMutex

	// Metrics
	metrics *SNSMetrics

	// Context for background operations
	ctx    context.Context
	cancel context.CancelFunc
}

// SNSMetrics tracks SNS operations
type SNSMetrics struct {
	mu                   sync.RWMutex
	MessagesPublished    int64
	MessagesFailed       int64
	TopicsCreated        int64
	SubscriptionsCreated int64
	RateLimitHits        int64
	Retries              int64
	LastError            error
	LastErrorTime        time.Time
}

// NewSNSClient creates a new SNS client
func NewSNSClient(cfg *SNSConfig) (*SNSClient, error) {
	if cfg == nil {
		cfg = DefaultSNSConfig()
	}

	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	ctx, cancel := context.WithCancel(context.Background())

	client := &SNSClient{
		config:            cfg,
		topicCache:        make(map[string]string),
		subscriptionCache: make(map[string][]string),
		metrics:           &SNSMetrics{},
		ctx:               ctx,
		cancel:            cancel,
	}

	// Initialize rate limiter
	client.rateLimiter = NewRateLimiter(cfg.RateLimit, cfg.BurstLimit)

	// Initialize error handler
	client.errorHandler = NewErrorHandler(cfg.RetryAttempts, cfg.RetryDelay, cfg.RetryMaxDelay, cfg.RetryMultiplier)

	return client, nil
}

// InitializeSNS initializes the SNS client and validates connectivity
func (c *SNSClient) InitializeSNS(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.initialized {
		return nil
	}

	// Build AWS config options
	var opts []func(*config.LoadOptions) error

	opts = append(opts, config.WithRegion(c.config.Region))

	// Add credentials if provided
	if c.config.AccessKeyID != "" && c.config.SecretAccessKey != "" {
		opts = append(opts, config.WithCredentialsProvider(
			credentials.NewStaticCredentialsProvider(
				c.config.AccessKeyID,
				c.config.SecretAccessKey,
				c.config.SessionToken,
			),
		))
	}

	// Load AWS configuration
	awsCfg, err := config.LoadDefaultConfig(ctx, opts...)
	if err != nil {
		return fmt.Errorf("failed to load AWS config: %w", err)
	}

	// Create SNS client options
	snsOpts := []func(*sns.Options){}

	// Add custom endpoint if provided (for LocalStack/testing)
	if c.config.Endpoint != "" {
		snsOpts = append(snsOpts, func(o *sns.Options) {
			o.BaseEndpoint = aws.String(c.config.Endpoint)
		})
	}

	// Create SNS client
	c.client = sns.NewFromConfig(awsCfg, snsOpts...)

	// Verify connectivity by listing topics
	_, err = c.client.ListTopics(ctx, &sns.ListTopicsInput{})
	if err != nil {
		return fmt.Errorf("failed to connect to SNS: %w", err)
	}

	// Load existing topic ARNs into cache
	if err := c.loadTopicCache(ctx); err != nil {
		if c.config.EnableLogging {
			log.Printf("Warning: failed to load topic cache: %v", err)
		}
	}

	// Merge configured topic ARNs
	for name, arn := range c.config.TopicARNs {
		c.topicCache[name] = arn
	}

	c.initialized = true

	// Start health check goroutine
	if c.config.HealthCheckInterval > 0 {
		go c.healthCheckLoop()
	}

	if c.config.EnableLogging {
		log.Printf("SNS client initialized successfully in region %s", c.config.Region)
	}

	return nil
}

// loadTopicCache loads existing topics into the cache
func (c *SNSClient) loadTopicCache(ctx context.Context) error {
	var nextToken *string

	for {
		output, err := c.client.ListTopics(ctx, &sns.ListTopicsInput{
			NextToken: nextToken,
		})
		if err != nil {
			return err
		}

		for _, topic := range output.Topics {
			if topic.TopicArn != nil {
				// Extract topic name from ARN
				name := extractTopicName(*topic.TopicArn)
				c.topicCache[name] = *topic.TopicArn
			}
		}

		if output.NextToken == nil {
			break
		}
		nextToken = output.NextToken
	}

	return nil
}

// extractTopicName extracts the topic name from an ARN
func extractTopicName(arn string) string {
	// ARN format: arn:aws:sns:region:account-id:topic-name
	for i := len(arn) - 1; i >= 0; i-- {
		if arn[i] == ':' {
			return arn[i+1:]
		}
	}
	return arn
}

// healthCheckLoop performs periodic health checks
func (c *SNSClient) healthCheckLoop() {
	ticker := time.NewTicker(c.config.HealthCheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-c.ctx.Done():
			return
		case <-ticker.C:
			if err := c.healthCheck(); err != nil {
				if c.config.EnableLogging {
					log.Printf("SNS health check failed: %v", err)
				}
				c.metrics.mu.Lock()
				c.metrics.LastError = err
				c.metrics.LastErrorTime = time.Now()
				c.metrics.mu.Unlock()
			}
		}
	}
}

// healthCheck performs a health check
func (c *SNSClient) healthCheck() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := c.client.ListTopics(ctx, &sns.ListTopicsInput{})
	return err
}

// CreateTopic creates a new SNS topic
func (c *SNSClient) CreateTopic(ctx context.Context, topicCfg *TopicConfig) (string, error) {
	if !c.isInitialized() {
		return "", ErrNotInitialized
	}

	if err := topicCfg.Validate(); err != nil {
		return "", fmt.Errorf("%w: %v", ErrInvalidTopicName, err)
	}

	// Check rate limit
	if !c.rateLimiter.Allow() {
		c.metrics.mu.Lock()
		c.metrics.RateLimitHits++
		c.metrics.mu.Unlock()
		return "", ErrRateLimitExceeded
	}

	// Check if topic already exists
	c.topicCacheMu.RLock()
	if arn, exists := c.topicCache[topicCfg.Name]; exists {
		c.topicCacheMu.RUnlock()
		return arn, nil // Return existing ARN (CreateTopic is idempotent)
	}
	c.topicCacheMu.RUnlock()

	// Build create topic input
	input := &sns.CreateTopicInput{
		Name: aws.String(topicCfg.Name),
	}

	// Add attributes
	attributes := make(map[string]string)

	if topicCfg.DisplayName != "" {
		attributes["DisplayName"] = topicCfg.DisplayName
	}

	if topicCfg.FIFOTopic {
		attributes["FifoTopic"] = "true"
		if topicCfg.ContentBasedDedup {
			attributes["ContentBasedDeduplication"] = "true"
		}
	}

	if topicCfg.KMSKeyID != "" {
		attributes["KmsMasterKeyId"] = topicCfg.KMSKeyID
	}

	if len(attributes) > 0 {
		input.Attributes = attributes
	}

	// Add tags
	if len(topicCfg.Tags) > 0 {
		tags := make([]types.Tag, 0, len(topicCfg.Tags))
		for k, v := range topicCfg.Tags {
			tags = append(tags, types.Tag{
				Key:   aws.String(k),
				Value: aws.String(v),
			})
		}
		input.Tags = tags
	}

	// Create topic with retry
	var output *sns.CreateTopicOutput
	err := c.errorHandler.Execute(ctx, func() error {
		var err error
		output, err = c.client.CreateTopic(ctx, input)
		return err
	})

	if err != nil {
		c.metrics.mu.Lock()
		c.metrics.MessagesFailed++
		c.metrics.LastError = err
		c.metrics.LastErrorTime = time.Now()
		c.metrics.mu.Unlock()
		return "", fmt.Errorf("failed to create topic: %w", err)
	}

	topicARN := *output.TopicArn

	// Update cache
	c.topicCacheMu.Lock()
	c.topicCache[topicCfg.Name] = topicARN
	c.topicCacheMu.Unlock()

	// Configure policy if provided
	if topicCfg.Policy != nil {
		if err := c.ConfigureTopicPolicy(ctx, topicARN, topicCfg.Policy); err != nil {
			if c.config.EnableLogging {
				log.Printf("Warning: failed to configure topic policy: %v", err)
			}
		}
	}

	// Configure delivery policy if provided
	if topicCfg.DeliveryPolicy != nil {
		if err := c.setDeliveryPolicy(ctx, topicARN, topicCfg.DeliveryPolicy); err != nil {
			if c.config.EnableLogging {
				log.Printf("Warning: failed to configure delivery policy: %v", err)
			}
		}
	}

	c.metrics.mu.Lock()
	c.metrics.TopicsCreated++
	c.metrics.mu.Unlock()

	if c.config.EnableLogging {
		log.Printf("Created SNS topic: %s (ARN: %s)", topicCfg.Name, topicARN)
	}

	return topicARN, nil
}

// ConfigureTopicPolicy sets the access policy for a topic
func (c *SNSClient) ConfigureTopicPolicy(ctx context.Context, topicARN string, policy *TopicPolicy) error {
	if !c.isInitialized() {
		return ErrNotInitialized
	}

	if policy == nil {
		return ErrInvalidPolicy
	}

	// Build IAM policy document
	policyDoc, err := c.buildPolicyDocument(topicARN, policy)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrInvalidPolicy, err)
	}

	// Set topic attribute
	_, err = c.client.SetTopicAttributes(ctx, &sns.SetTopicAttributesInput{
		TopicArn:       aws.String(topicARN),
		AttributeName:  aws.String("Policy"),
		AttributeValue: aws.String(policyDoc),
	})

	if err != nil {
		return fmt.Errorf("failed to set topic policy: %w", err)
	}

	if c.config.EnableLogging {
		log.Printf("Configured policy for topic: %s", topicARN)
	}

	return nil
}

// buildPolicyDocument creates an IAM policy document
func (c *SNSClient) buildPolicyDocument(topicARN string, policy *TopicPolicy) (string, error) {
	statements := []map[string]interface{}{}

	// Allow statement
	if len(policy.AllowedPrincipals) > 0 && len(policy.AllowedActions) > 0 {
		allowStmt := map[string]interface{}{
			"Sid":       "AllowPolicy",
			"Effect":    "Allow",
			"Principal": map[string]interface{}{},
			"Action":    policy.AllowedActions,
			"Resource":  topicARN,
		}

		// Handle principals
		if len(policy.AllowedPrincipals) == 1 && policy.AllowedPrincipals[0] == "*" {
			allowStmt["Principal"] = "*"
		} else {
			allowStmt["Principal"] = map[string]interface{}{
				"AWS": policy.AllowedPrincipals,
			}
		}

		// Add conditions if provided
		if len(policy.AllowedConditions) > 0 {
			allowStmt["Condition"] = policy.AllowedConditions
		}

		statements = append(statements, allowStmt)
	}

	// Deny statement
	if len(policy.DenyPrincipals) > 0 {
		denyStmt := map[string]interface{}{
			"Sid":    "DenyPolicy",
			"Effect": "Deny",
			"Principal": map[string]interface{}{
				"AWS": policy.DenyPrincipals,
			},
			"Action":   "*",
			"Resource": topicARN,
		}
		statements = append(statements, denyStmt)
	}

	policyDoc := map[string]interface{}{
		"Version":   "2012-10-17",
		"Id":        fmt.Sprintf("%s-policy", extractTopicName(topicARN)),
		"Statement": statements,
	}

	jsonBytes, err := json.Marshal(policyDoc)
	if err != nil {
		return "", err
	}

	return string(jsonBytes), nil
}

// setDeliveryPolicy sets the delivery policy for a topic
func (c *SNSClient) setDeliveryPolicy(ctx context.Context, topicARN string, policy *DeliveryPolicy) error {
	policyJSON, err := json.Marshal(map[string]interface{}{
		"http": policy,
	})
	if err != nil {
		return err
	}

	_, err = c.client.SetTopicAttributes(ctx, &sns.SetTopicAttributesInput{
		TopicArn:       aws.String(topicARN),
		AttributeName:  aws.String("DeliveryPolicy"),
		AttributeValue: aws.String(string(policyJSON)),
	})

	return err
}

// DeleteTopic deletes an SNS topic
func (c *SNSClient) DeleteTopic(ctx context.Context, topicARN string) error {
	if !c.isInitialized() {
		return ErrNotInitialized
	}

	// Check rate limit
	if !c.rateLimiter.Allow() {
		c.metrics.mu.Lock()
		c.metrics.RateLimitHits++
		c.metrics.mu.Unlock()
		return ErrRateLimitExceeded
	}

	err := c.errorHandler.Execute(ctx, func() error {
		_, err := c.client.DeleteTopic(ctx, &sns.DeleteTopicInput{
			TopicArn: aws.String(topicARN),
		})
		return err
	})

	if err != nil {
		return fmt.Errorf("failed to delete topic: %w", err)
	}

	// Remove from cache
	c.topicCacheMu.Lock()
	for name, arn := range c.topicCache {
		if arn == topicARN {
			delete(c.topicCache, name)
			break
		}
	}
	c.topicCacheMu.Unlock()

	if c.config.EnableLogging {
		log.Printf("Deleted SNS topic: %s", topicARN)
	}

	return nil
}

// GetTopicARN returns the ARN for a topic name
func (c *SNSClient) GetTopicARN(topicName string) (string, error) {
	c.topicCacheMu.RLock()
	defer c.topicCacheMu.RUnlock()

	arn, exists := c.topicCache[topicName]
	if !exists {
		return "", ErrTopicNotFound
	}
	return arn, nil
}

// GetTopicAttributes retrieves attributes for a topic
func (c *SNSClient) GetTopicAttributes(ctx context.Context, topicARN string) (map[string]string, error) {
	if !c.isInitialized() {
		return nil, ErrNotInitialized
	}

	output, err := c.client.GetTopicAttributes(ctx, &sns.GetTopicAttributesInput{
		TopicArn: aws.String(topicARN),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get topic attributes: %w", err)
	}

	return output.Attributes, nil
}

// Publish publishes a message to a topic
func (c *SNSClient) Publish(ctx context.Context, topicARN, message string, attributes map[string]string) (string, error) {
	if !c.isInitialized() {
		return "", ErrNotInitialized
	}

	// Check rate limit
	if !c.rateLimiter.Allow() {
		c.metrics.mu.Lock()
		c.metrics.RateLimitHits++
		c.metrics.mu.Unlock()
		return "", ErrRateLimitExceeded
	}

	input := &sns.PublishInput{
		TopicArn: aws.String(topicARN),
		Message:  aws.String(message),
	}

	// Add message attributes
	if len(attributes) > 0 {
		msgAttrs := make(map[string]types.MessageAttributeValue)
		for k, v := range attributes {
			msgAttrs[k] = types.MessageAttributeValue{
				DataType:    aws.String("String"),
				StringValue: aws.String(v),
			}
		}
		input.MessageAttributes = msgAttrs
	}

	var output *sns.PublishOutput
	err := c.errorHandler.Execute(ctx, func() error {
		var err error
		output, err = c.client.Publish(ctx, input)
		return err
	})

	if err != nil {
		c.metrics.mu.Lock()
		c.metrics.MessagesFailed++
		c.metrics.LastError = err
		c.metrics.LastErrorTime = time.Now()
		c.metrics.mu.Unlock()
		return "", fmt.Errorf("%w: %v", ErrPublishFailed, err)
	}

	c.metrics.mu.Lock()
	c.metrics.MessagesPublished++
	c.metrics.mu.Unlock()

	return *output.MessageId, nil
}

// Subscribe creates a subscription to a topic
func (c *SNSClient) Subscribe(ctx context.Context, topicARN, protocol, endpoint string) (string, error) {
	if !c.isInitialized() {
		return "", ErrNotInitialized
	}

	// Check rate limit
	if !c.rateLimiter.Allow() {
		c.metrics.mu.Lock()
		c.metrics.RateLimitHits++
		c.metrics.mu.Unlock()
		return "", ErrRateLimitExceeded
	}

	var output *sns.SubscribeOutput
	err := c.errorHandler.Execute(ctx, func() error {
		var err error
		output, err = c.client.Subscribe(ctx, &sns.SubscribeInput{
			TopicArn: aws.String(topicARN),
			Protocol: aws.String(protocol),
			Endpoint: aws.String(endpoint),
		})
		return err
	})

	if err != nil {
		return "", fmt.Errorf("%w: %v", ErrSubscriptionFailed, err)
	}

	subscriptionARN := ""
	if output.SubscriptionArn != nil {
		subscriptionARN = *output.SubscriptionArn
	}

	// Update subscription cache
	c.subscriptionCacheMu.Lock()
	c.subscriptionCache[topicARN] = append(c.subscriptionCache[topicARN], subscriptionARN)
	c.subscriptionCacheMu.Unlock()

	c.metrics.mu.Lock()
	c.metrics.SubscriptionsCreated++
	c.metrics.mu.Unlock()

	if c.config.EnableLogging {
		log.Printf("Created subscription: %s -> %s (%s)", topicARN, endpoint, protocol)
	}

	return subscriptionARN, nil
}

// Unsubscribe removes a subscription
func (c *SNSClient) Unsubscribe(ctx context.Context, subscriptionARN string) error {
	if !c.isInitialized() {
		return ErrNotInitialized
	}

	_, err := c.client.Unsubscribe(ctx, &sns.UnsubscribeInput{
		SubscriptionArn: aws.String(subscriptionARN),
	})

	if err != nil {
		return fmt.Errorf("failed to unsubscribe: %w", err)
	}

	return nil
}

// ValidateConfiguration validates the current configuration and connectivity
func (c *SNSClient) ValidateConfiguration(ctx context.Context) error {
	if !c.isInitialized() {
		return ErrNotInitialized
	}

	// Validate connectivity
	if err := c.healthCheck(); err != nil {
		return fmt.Errorf("connectivity check failed: %w", err)
	}

	// Validate all configured topics exist
	c.topicCacheMu.RLock()
	topicsToCheck := make(map[string]string, len(c.topicCache))
	for k, v := range c.topicCache {
		topicsToCheck[k] = v
	}
	c.topicCacheMu.RUnlock()

	for name, arn := range topicsToCheck {
		_, err := c.client.GetTopicAttributes(ctx, &sns.GetTopicAttributesInput{
			TopicArn: aws.String(arn),
		})
		if err != nil {
			return fmt.Errorf("topic %s (ARN: %s) validation failed: %w", name, arn, err)
		}
	}

	return nil
}

// SetupErrorHandling configures error handling behavior
func (c *SNSClient) SetupErrorHandling(retryAttempts int, retryDelay, maxDelay time.Duration, multiplier float64) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.errorHandler = NewErrorHandler(retryAttempts, retryDelay, maxDelay, multiplier)

	c.config.RetryAttempts = retryAttempts
	c.config.RetryDelay = retryDelay
	c.config.RetryMaxDelay = maxDelay
	c.config.RetryMultiplier = multiplier
}

// GetMetrics returns current metrics
func (c *SNSClient) GetMetrics() *SNSMetrics {
	c.metrics.mu.RLock()
	defer c.metrics.mu.RUnlock()

	return &SNSMetrics{
		MessagesPublished:    c.metrics.MessagesPublished,
		MessagesFailed:       c.metrics.MessagesFailed,
		TopicsCreated:        c.metrics.TopicsCreated,
		SubscriptionsCreated: c.metrics.SubscriptionsCreated,
		RateLimitHits:        c.metrics.RateLimitHits,
		Retries:              c.metrics.Retries,
		LastError:            c.metrics.LastError,
		LastErrorTime:        c.metrics.LastErrorTime,
	}
}

// isInitialized checks if the client is initialized
func (c *SNSClient) isInitialized() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.initialized
}

// Close gracefully shuts down the SNS client
func (c *SNSClient) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.initialized {
		return nil
	}

	c.cancel()
	c.initialized = false

	if c.config.EnableLogging {
		log.Println("SNS client closed")
	}

	return nil
}

// ListTopics lists all topics
func (c *SNSClient) ListTopics(ctx context.Context) ([]string, error) {
	if !c.isInitialized() {
		return nil, ErrNotInitialized
	}

	var topicARNs []string
	var nextToken *string

	for {
		output, err := c.client.ListTopics(ctx, &sns.ListTopicsInput{
			NextToken: nextToken,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to list topics: %w", err)
		}

		for _, topic := range output.Topics {
			if topic.TopicArn != nil {
				topicARNs = append(topicARNs, *topic.TopicArn)
			}
		}

		if output.NextToken == nil {
			break
		}
		nextToken = output.NextToken
	}

	return topicARNs, nil
}

// GetClient returns the underlying SNS client (for advanced operations)
func (c *SNSClient) GetClient() *sns.Client {
	return c.client
}
