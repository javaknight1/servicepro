package email

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch"
	"github.com/aws/aws-sdk-go-v2/service/ses"
)

// Environment represents the deployment environment
type Environment string

const (
	EnvStaging    Environment = "staging"
	EnvProduction Environment = "production"
	EnvLocal      Environment = "local"
)

// SESConfig holds all configuration for the SES email service
type SESConfig struct {
	// AWS Configuration
	Region          string
	AccessKeyID     string
	SecretAccessKey string

	// Environment
	Environment Environment

	// Email Settings
	DefaultFromEmail     string
	DefaultFromName      string
	DefaultReplyTo       string
	ConfigurationSetName string

	// Domain Settings
	Domain          string
	VerifiedDomains []string

	// Rate Limiting
	MaxSendRate float64 // emails per second
	DailyQuota  int64

	// Retry Configuration
	MaxRetries      int
	RetryBaseDelay  time.Duration
	RetryMaxDelay   time.Duration
	RetryMultiplier float64

	// Timeouts
	ConnectionTimeout time.Duration
	RequestTimeout    time.Duration

	// Feature Flags
	EnableTracking bool
	EnableDKIM     bool
	EnableFeedback bool
	SandboxMode    bool

	// CloudWatch Metrics
	MetricsEnabled    bool
	MetricsNamespace  string
	MetricsDimensions map[string]string

	// Logging
	LogLevel  string
	LogFormat string

	// Templates
	TemplatePrefix string
}

// DefaultConfig returns a configuration with sensible defaults
func DefaultConfig() *SESConfig {
	return &SESConfig{
		Region:            "us-east-1",
		Environment:       EnvLocal,
		DefaultFromName:   "ServicePro",
		MaxSendRate:       14.0, // SES default for sandbox
		DailyQuota:        200,  // SES sandbox limit
		MaxRetries:        3,
		RetryBaseDelay:    100 * time.Millisecond,
		RetryMaxDelay:     30 * time.Second,
		RetryMultiplier:   2.0,
		ConnectionTimeout: 10 * time.Second,
		RequestTimeout:    30 * time.Second,
		EnableTracking:    true,
		EnableDKIM:        true,
		EnableFeedback:    true,
		SandboxMode:       true,
		MetricsEnabled:    true,
		MetricsNamespace:  "ServicePro/Email",
		MetricsDimensions: make(map[string]string),
		LogLevel:          "info",
		LogFormat:         "json",
		TemplatePrefix:    "servicepro_",
	}
}

// StagingConfig returns configuration optimized for staging environment
func StagingConfig() *SESConfig {
	cfg := DefaultConfig()
	cfg.Environment = EnvStaging
	cfg.SandboxMode = true
	cfg.MaxSendRate = 14.0
	cfg.DailyQuota = 200
	cfg.LogLevel = "debug"
	cfg.MetricsDimensions["Environment"] = "staging"
	return cfg
}

// ProductionConfig returns configuration optimized for production environment
func ProductionConfig() *SESConfig {
	cfg := DefaultConfig()
	cfg.Environment = EnvProduction
	cfg.SandboxMode = false
	cfg.MaxSendRate = 50.0 // Adjust based on your SES limits
	cfg.DailyQuota = 50000 // Adjust based on your SES limits
	cfg.LogLevel = "info"
	cfg.MetricsDimensions["Environment"] = "production"
	cfg.MaxRetries = 5
	cfg.RetryMaxDelay = 60 * time.Second
	return cfg
}

// LoadConfigFromEnv loads configuration from environment variables
func LoadConfigFromEnv() (*SESConfig, error) {
	cfg := DefaultConfig()

	// AWS Configuration
	if v := os.Getenv("AWS_REGION"); v != "" {
		cfg.Region = v
	}
	if v := os.Getenv("AWS_ACCESS_KEY_ID"); v != "" {
		cfg.AccessKeyID = v
	}
	if v := os.Getenv("AWS_SECRET_ACCESS_KEY"); v != "" {
		cfg.SecretAccessKey = v
	}

	// Environment
	if v := os.Getenv("APP_ENV"); v != "" {
		switch v {
		case "production", "prod":
			cfg.Environment = EnvProduction
		case "staging", "stage":
			cfg.Environment = EnvStaging
		default:
			cfg.Environment = EnvLocal
		}
	}

	// Email Settings
	if v := os.Getenv("SES_FROM_EMAIL"); v != "" {
		cfg.DefaultFromEmail = v
	}
	if v := os.Getenv("SES_FROM_NAME"); v != "" {
		cfg.DefaultFromName = v
	}
	if v := os.Getenv("SES_REPLY_TO"); v != "" {
		cfg.DefaultReplyTo = v
	}
	if v := os.Getenv("SES_CONFIGURATION_SET"); v != "" {
		cfg.ConfigurationSetName = v
	}

	// Domain Settings
	if v := os.Getenv("SES_DOMAIN"); v != "" {
		cfg.Domain = v
	}

	// Rate Limiting
	if v := os.Getenv("SES_MAX_SEND_RATE"); v != "" {
		rate, err := strconv.ParseFloat(v, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid SES_MAX_SEND_RATE: %w", err)
		}
		cfg.MaxSendRate = rate
	}
	if v := os.Getenv("SES_DAILY_QUOTA"); v != "" {
		quota, err := strconv.ParseInt(v, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid SES_DAILY_QUOTA: %w", err)
		}
		cfg.DailyQuota = quota
	}

	// Retry Configuration
	if v := os.Getenv("SES_MAX_RETRIES"); v != "" {
		retries, err := strconv.Atoi(v)
		if err != nil {
			return nil, fmt.Errorf("invalid SES_MAX_RETRIES: %w", err)
		}
		cfg.MaxRetries = retries
	}

	// Feature Flags
	if v := os.Getenv("SES_SANDBOX_MODE"); v != "" {
		cfg.SandboxMode = v == "true" || v == "1"
	}
	if v := os.Getenv("SES_ENABLE_TRACKING"); v != "" {
		cfg.EnableTracking = v == "true" || v == "1"
	}
	if v := os.Getenv("SES_ENABLE_DKIM"); v != "" {
		cfg.EnableDKIM = v == "true" || v == "1"
	}

	// CloudWatch
	if v := os.Getenv("SES_METRICS_ENABLED"); v != "" {
		cfg.MetricsEnabled = v == "true" || v == "1"
	}
	if v := os.Getenv("SES_METRICS_NAMESPACE"); v != "" {
		cfg.MetricsNamespace = v
	}

	// Logging
	if v := os.Getenv("SES_LOG_LEVEL"); v != "" {
		cfg.LogLevel = v
	}

	// Apply environment-specific overrides
	switch cfg.Environment {
	case EnvProduction:
		cfg.SandboxMode = false
		if cfg.MaxSendRate < 50 {
			cfg.MaxSendRate = 50
		}
	case EnvStaging:
		cfg.MetricsDimensions["Environment"] = "staging"
	}

	// Validate configuration
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}

// Validate checks if the configuration is valid
func (c *SESConfig) Validate() error {
	if c.Region == "" {
		return fmt.Errorf("AWS region is required")
	}

	if c.DefaultFromEmail == "" {
		return fmt.Errorf("default from email is required")
	}

	if c.MaxRetries < 0 {
		return fmt.Errorf("max retries must be non-negative")
	}

	if c.MaxSendRate <= 0 {
		return fmt.Errorf("max send rate must be positive")
	}

	if c.RetryBaseDelay <= 0 {
		c.RetryBaseDelay = 100 * time.Millisecond
	}

	if c.RetryMaxDelay <= 0 {
		c.RetryMaxDelay = 30 * time.Second
	}

	if c.RetryMultiplier <= 1 {
		c.RetryMultiplier = 2.0
	}

	return nil
}

// LoadAWSConfig creates an AWS SDK configuration from the SES config
func (c *SESConfig) LoadAWSConfig(ctx context.Context) (aws.Config, error) {
	var opts []func(*config.LoadOptions) error

	// Set region
	opts = append(opts, config.WithRegion(c.Region))

	// Set credentials if provided
	if c.AccessKeyID != "" && c.SecretAccessKey != "" {
		opts = append(opts, config.WithCredentialsProvider(
			credentials.NewStaticCredentialsProvider(
				c.AccessKeyID,
				c.SecretAccessKey,
				"",
			),
		))
	}

	// Load the configuration
	awsCfg, err := config.LoadDefaultConfig(ctx, opts...)
	if err != nil {
		return aws.Config{}, fmt.Errorf("failed to load AWS config: %w", err)
	}

	return awsCfg, nil
}

// CreateSESClient creates a new SES client from the configuration
func (c *SESConfig) CreateSESClient(ctx context.Context) (*ses.Client, error) {
	awsCfg, err := c.LoadAWSConfig(ctx)
	if err != nil {
		return nil, err
	}

	client := ses.NewFromConfig(awsCfg)
	return client, nil
}

// CreateCloudWatchClient creates a new CloudWatch client from the configuration
func (c *SESConfig) CreateCloudWatchClient(ctx context.Context) (*cloudwatch.Client, error) {
	awsCfg, err := c.LoadAWSConfig(ctx)
	if err != nil {
		return nil, err
	}

	client := cloudwatch.NewFromConfig(awsCfg)
	return client, nil
}

// GetFromAddress returns the formatted from address
func (c *SESConfig) GetFromAddress() string {
	if c.DefaultFromName != "" {
		return fmt.Sprintf("%s <%s>", c.DefaultFromName, c.DefaultFromEmail)
	}
	return c.DefaultFromEmail
}

// GetReplyTo returns the reply-to address or defaults to from address
func (c *SESConfig) GetReplyTo() string {
	if c.DefaultReplyTo != "" {
		return c.DefaultReplyTo
	}
	return c.DefaultFromEmail
}

// IsSandbox returns true if running in sandbox mode
func (c *SESConfig) IsSandbox() bool {
	return c.SandboxMode
}

// IsProduction returns true if running in production environment
func (c *SESConfig) IsProduction() bool {
	return c.Environment == EnvProduction
}

// Clone creates a deep copy of the configuration
func (c *SESConfig) Clone() *SESConfig {
	clone := *c
	clone.VerifiedDomains = make([]string, len(c.VerifiedDomains))
	copy(clone.VerifiedDomains, c.VerifiedDomains)
	clone.MetricsDimensions = make(map[string]string)
	for k, v := range c.MetricsDimensions {
		clone.MetricsDimensions[k] = v
	}
	return &clone
}
