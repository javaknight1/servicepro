//go:build integration

package integration

import (
	"fmt"
	"os"
	"strconv"
	"sync"
	"time"
)

// =============================================================================
// Test Configuration
// =============================================================================

// TestConfig holds all configuration for integration tests
type TestConfig struct {
	// Database
	DatabaseURL string

	// Redis
	RedisURL string

	// Stripe Mock
	StripeAPIKey        string
	StripeWebhookSecret string
	StripeAPIBase       string

	// AWS/LocalStack
	AWSEndpointURL     string
	AWSAccessKeyID     string
	AWSSecretAccessKey string
	AWSRegion          string

	// S3 Buckets
	S3DocumentsBucket string
	S3InvoicesBucket  string
	S3ExportsBucket   string

	// Email
	SMTPHost   string
	SMTPPort   int
	MailHogAPI string

	// JWT
	JWTSecret string

	// Server
	APIBaseURL string
	Timeout    time.Duration
}

var (
	testConfig     *TestConfig
	testConfigOnce sync.Once
)

// LoadTestConfig loads configuration from environment variables
func LoadTestConfig() *TestConfig {
	testConfigOnce.Do(func() {
		testConfig = &TestConfig{
			// Database
			DatabaseURL: getEnv("TEST_DATABASE_URL", "postgresql://test_user:test_password@localhost:5433/servicepro_test?sslmode=disable"),

			// Redis
			RedisURL: getEnv("REDIS_URL", "redis://localhost:6380"),

			// Stripe Mock
			StripeAPIKey:        getEnv("STRIPE_API_KEY", "sk_test_mock_key"),
			StripeWebhookSecret: getEnv("STRIPE_WEBHOOK_SECRET", "whsec_test_secret"),
			StripeAPIBase:       getEnv("STRIPE_API_BASE", "http://localhost:12111"),

			// AWS/LocalStack
			AWSEndpointURL:     getEnv("AWS_ENDPOINT_URL", "http://localhost:4566"),
			AWSAccessKeyID:     getEnv("AWS_ACCESS_KEY_ID", "test"),
			AWSSecretAccessKey: getEnv("AWS_SECRET_ACCESS_KEY", "test"),
			AWSRegion:          getEnv("AWS_REGION", "us-east-1"),

			// S3 Buckets
			S3DocumentsBucket: getEnv("S3_DOCUMENTS_BUCKET", "servicepro-test-documents"),
			S3InvoicesBucket:  getEnv("S3_INVOICES_BUCKET", "servicepro-test-invoices"),
			S3ExportsBucket:   getEnv("S3_EXPORTS_BUCKET", "servicepro-test-exports"),

			// Email
			SMTPHost:   getEnv("SMTP_HOST", "localhost"),
			SMTPPort:   getEnvInt("SMTP_PORT", 1025),
			MailHogAPI: getEnv("MAILHOG_API", "http://localhost:8025"),

			// JWT
			JWTSecret: getEnv("JWT_SECRET", "test-jwt-secret-for-integration-tests"),

			// Server
			APIBaseURL: getEnv("API_BASE_URL", "http://localhost:8081"),
			Timeout:    getEnvDuration("TEST_TIMEOUT", 30*time.Second),
		}
	})
	return testConfig
}

// GetTestConfig returns the loaded test configuration
func GetTestConfig() *TestConfig {
	return LoadTestConfig()
}

// =============================================================================
// Environment Helpers
// =============================================================================

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intVal, err := strconv.Atoi(value); err == nil {
			return intVal
		}
	}
	return defaultValue
}

func getEnvDuration(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if duration, err := time.ParseDuration(value); err == nil {
			return duration
		}
	}
	return defaultValue
}

// =============================================================================
// Test Environment Variables
// =============================================================================

// SetTestEnv sets up environment variables for testing
func SetTestEnv() {
	cfg := LoadTestConfig()

	os.Setenv("ENV", "test")
	os.Setenv("DATABASE_URL", cfg.DatabaseURL)
	os.Setenv("REDIS_URL", cfg.RedisURL)
	os.Setenv("JWT_SECRET", cfg.JWTSecret)
	os.Setenv("STRIPE_API_KEY", cfg.StripeAPIKey)
	os.Setenv("STRIPE_WEBHOOK_SECRET", cfg.StripeWebhookSecret)
	os.Setenv("STRIPE_API_BASE", cfg.StripeAPIBase)
	os.Setenv("AWS_ENDPOINT_URL", cfg.AWSEndpointURL)
	os.Setenv("AWS_ACCESS_KEY_ID", cfg.AWSAccessKeyID)
	os.Setenv("AWS_SECRET_ACCESS_KEY", cfg.AWSSecretAccessKey)
	os.Setenv("AWS_REGION", cfg.AWSRegion)
}

// IsCI returns true if running in CI environment
func IsCI() bool {
	return os.Getenv("CI") == "true" || os.Getenv("GITHUB_ACTIONS") == "true"
}

// SkipIfCI skips the test if running in CI
func SkipIfCI(reason string) string {
	if IsCI() {
		return fmt.Sprintf("Skipped in CI: %s", reason)
	}
	return ""
}
