package stripe

import (
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/stripe/stripe-go/v76"
)

// Environment represents the Stripe environment (test or live)
type Environment string

const (
	EnvironmentTest Environment = "test"
	EnvironmentLive Environment = "live"
)

// Config holds Stripe configuration
type Config struct {
	// API Keys
	SecretKey      string
	PublishableKey string
	WebhookSecret  string

	// Environment
	Environment Environment

	// API Configuration
	APIVersion        string
	MaxNetworkRetries int
	HTTPClient        interface{} // Custom HTTP client if needed

	// Timeout settings
	ConnectTimeout time.Duration
	Timeout        time.Duration

	// Rate limiting
	MaxConcurrentRequests int
	RequestsPerSecond     int

	// Webhook settings
	WebhookTolerance time.Duration

	// Logging
	LogLevel string
	Logger   StripeLogger
}

// StripeLogger interface for custom logging
type StripeLogger interface {
	Debugf(format string, v ...interface{})
	Infof(format string, v ...interface{})
	Warnf(format string, v ...interface{})
	Errorf(format string, v ...interface{})
}

// DefaultConfig returns a configuration with default values
func DefaultConfig() *Config {
	return &Config{
		APIVersion:            stripe.APIVersion,
		MaxNetworkRetries:     3,
		ConnectTimeout:        30 * time.Second,
		Timeout:               80 * time.Second,
		MaxConcurrentRequests: 100,
		RequestsPerSecond:     100,
		WebhookTolerance:      300 * time.Second, // 5 minutes
		LogLevel:              "info",
	}
}

// LoadFromEnv loads configuration from environment variables
func LoadFromEnv() (*Config, error) {
	config := DefaultConfig()

	// Required: Secret Key
	secretKey := os.Getenv("STRIPE_SECRET_KEY")
	if secretKey == "" {
		return nil, errors.New("STRIPE_SECRET_KEY environment variable is required")
	}
	config.SecretKey = secretKey

	// Required: Publishable Key
	publishableKey := os.Getenv("STRIPE_PUBLISHABLE_KEY")
	if publishableKey == "" {
		return nil, errors.New("STRIPE_PUBLISHABLE_KEY environment variable is required")
	}
	config.PublishableKey = publishableKey

	// Required: Webhook Secret
	webhookSecret := os.Getenv("STRIPE_WEBHOOK_SECRET")
	if webhookSecret == "" {
		return nil, errors.New("STRIPE_WEBHOOK_SECRET environment variable is required")
	}
	config.WebhookSecret = webhookSecret

	// Determine environment from secret key prefix
	if len(secretKey) > 3 {
		if secretKey[:3] == "sk_test_" {
			config.Environment = EnvironmentTest
		} else if secretKey[:3] == "sk_live_" {
			config.Environment = EnvironmentLive
		}
	}

	// Optional: Log Level
	if logLevel := os.Getenv("STRIPE_LOG_LEVEL"); logLevel != "" {
		config.LogLevel = logLevel
	}

	// Optional: Max Network Retries
	if retries := os.Getenv("STRIPE_MAX_RETRIES"); retries != "" {
		var retriesInt int
		fmt.Sscanf(retries, "%d", &retriesInt)
		if retriesInt > 0 {
			config.MaxNetworkRetries = retriesInt
		}
	}

	return config, nil
}

// Validate checks if the configuration is valid
func (c *Config) Validate() error {
	if c.SecretKey == "" {
		return errors.New("secret key is required")
	}

	if c.PublishableKey == "" {
		return errors.New("publishable key is required")
	}

	if c.WebhookSecret == "" {
		return errors.New("webhook secret is required")
	}

	// Validate secret key format
	if len(c.SecretKey) < 10 {
		return errors.New("invalid secret key format")
	}

	// Ensure keys match environment
	isTestSecret := len(c.SecretKey) > 8 && c.SecretKey[:8] == "sk_test_"
	isLiveSecret := len(c.SecretKey) > 8 && c.SecretKey[:8] == "sk_live_"

	isTestPublishable := len(c.PublishableKey) > 8 && c.PublishableKey[:8] == "pk_test_"
	isLivePublishable := len(c.PublishableKey) > 8 && c.PublishableKey[:8] == "pk_live_"

	// Keys must match (both test or both live)
	if (isTestSecret && !isTestPublishable) || (isLiveSecret && !isLivePublishable) {
		return errors.New("secret key and publishable key environments do not match")
	}

	if c.MaxNetworkRetries < 0 {
		return errors.New("max network retries must be non-negative")
	}

	if c.ConnectTimeout <= 0 {
		return errors.New("connect timeout must be positive")
	}

	if c.Timeout <= 0 {
		return errors.New("timeout must be positive")
	}

	return nil
}

// IsTestMode returns true if running in test mode
func (c *Config) IsTestMode() bool {
	return c.Environment == EnvironmentTest
}

// IsLiveMode returns true if running in live mode
func (c *Config) IsLiveMode() bool {
	return c.Environment == EnvironmentLive
}

// GetAPIKey returns the appropriate API key
func (c *Config) GetAPIKey() string {
	return c.SecretKey
}

// GetPublishableKey returns the publishable key (for frontend)
func (c *Config) GetPublishableKey() string {
	return c.PublishableKey
}

// MaskSecretKey returns a masked version of the secret key for logging
func (c *Config) MaskSecretKey() string {
	if len(c.SecretKey) < 12 {
		return "***"
	}
	return c.SecretKey[:12] + "..." + c.SecretKey[len(c.SecretKey)-4:]
}

// MaskPublishableKey returns a masked version of the publishable key
func (c *Config) MaskPublishableKey() string {
	if len(c.PublishableKey) < 12 {
		return "***"
	}
	return c.PublishableKey[:12] + "..." + c.PublishableKey[len(c.PublishableKey)-4:]
}

// String returns a string representation of the config (safe for logging)
func (c *Config) String() string {
	return fmt.Sprintf("StripeConfig{Environment: %s, SecretKey: %s, PublishableKey: %s}",
		c.Environment,
		c.MaskSecretKey(),
		c.MaskPublishableKey(),
	)
}
