package config

import (
	"log"
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
)

// Config holds all configuration for the application
type Config struct {
	Server        ServerConfig
	Database      DatabaseConfig
	Redis         RedisConfig
	JWT           JWTConfig
	Auth          AuthConfig
	AWS           AWSConfig
	R2            R2Config
	Resend        ResendConfig
	BetterStack   BetterStackConfig
	Stripe        StripeConfig
	SES           SESConfig
	Sentry        SentryConfig
	Health        HealthConfig
	Analytics     AnalyticsConfig
	Loki          LokiConfig
	Prometheus    PrometheusConfig
	OpenTelemetry OpenTelemetryConfig
	PDF           PDFConfig
	SNS           SNSConfig
}

// AWSConfig holds AWS configuration
type AWSConfig struct {
	Region          string
	SESFromEmail    string
	S3Bucket        string
	AccessKeyID     string
	SecretAccessKey string
}

// R2Config holds Cloudflare R2 storage configuration
type R2Config struct {
	AccountID       string
	AccessKeyID     string
	SecretAccessKey string
	Bucket          string
	Endpoint        string
	PublicURL       string // Optional: for public access URLs
}

// ResendConfig holds Resend email service configuration
type ResendConfig struct {
	APIKey    string
	FromEmail string
	FromName  string
}

// BetterStackConfig holds Better Stack (Logtail) monitoring configuration
type BetterStackConfig struct {
	SourceToken string
	Enabled     bool
}

// ServerConfig holds server configuration
type ServerConfig struct {
	Port        string
	Env         string
	FrontendURL string
}

// DatabaseConfig holds database configuration
type DatabaseConfig struct {
	URL             string
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
}

// RedisConfig holds Redis configuration
type RedisConfig struct {
	URL      string
	Password string
	DB       int
}

// JWTConfig holds JWT configuration
type JWTConfig struct {
	Secret            string
	AccessExpiration  time.Duration
	RefreshExpiration time.Duration
}

// AuthConfig holds authentication configuration
type AuthConfig struct {
	MaxLoginAttempts  int
	LockoutDuration   time.Duration
	RateLimitWindow   time.Duration
	RateLimitAttempts int
}

// StripeConfig holds Stripe configuration
type StripeConfig struct {
	SecretKey      string
	PublishableKey string
	WebhookSecret  string
	LogLevel       string
	MaxRetries     int
}

// SESConfig holds SES email service configuration
type SESConfig struct {
	FromEmail            string
	FromName             string
	ReplyTo              string
	ConfigurationSetName string
	Domain               string
	MaxSendRate          float64
	DailyQuota           int64
	MaxRetries           int
	SandboxMode          bool
	EnableTracking       bool
	EnableDKIM           bool
	MetricsEnabled       bool
	MetricsNamespace     string
	LogLevel             string
}

// SentryConfig holds error tracking configuration
type SentryConfig struct {
	DSN              string
	Environment      string
	Release          string
	SampleRate       float64
	TracesSampleRate float64
	Debug            bool
	ServerName       string
}

// HealthConfig holds health check configuration
type HealthConfig struct {
	Enabled       bool
	CheckDatabase bool
	CheckRedis    bool
	CheckMemory   bool
	CheckDisk     bool
	CheckExternal bool
}

// AnalyticsConfig holds analytics tracking configuration
type AnalyticsConfig struct {
	Enabled bool
	Debug   bool
}

// LokiConfig holds Grafana Loki logging configuration
type LokiConfig struct {
	Enabled  bool
	Endpoint string
	TenantID string
	Username string
	Password string
}

// PrometheusConfig holds Prometheus metrics configuration
type PrometheusConfig struct {
	Enabled      bool
	MetricsPath  string
	Namespace    string
	Subsystem    string
	PushGateway  string
	PushInterval time.Duration
}

// OpenTelemetryConfig holds OpenTelemetry tracing configuration
type OpenTelemetryConfig struct {
	Enabled     bool
	Endpoint    string
	Insecure    bool
	ServiceName string
	SampleRate  float64
}

// PDFConfig holds PDF generation configuration
type PDFConfig struct {
	// StorageType can be "local", "s3", or "both"
	StorageType string
	// LocalPath is the local directory for storing PDFs
	LocalPath string
	// S3Bucket is the S3 bucket for storing PDFs (uses AWS config for credentials)
	S3Bucket string
	// S3Prefix is the key prefix for PDFs in S3
	S3Prefix string
	// S3PublicURL is the public URL base for S3 objects (optional)
	S3PublicURL string
	// CompanyName for branding in PDFs
	CompanyName string
	// CompanyLogo path for branding
	CompanyLogo string
}

// SNSConfig holds AWS SNS configuration
type SNSConfig struct {
	// Enabled enables SNS notifications
	Enabled bool
	// TopicARN is the default SNS topic ARN
	TopicARN string
	// Region overrides AWS.Region for SNS (optional)
	Region string
}

// Load reads configuration from environment variables
func Load() *Config {
	// Load .env file if it exists (for local development)
	_ = godotenv.Load()

	return &Config{
		Server: ServerConfig{
			Port:        getEnv("PORT", "8080"),
			Env:         getEnv("ENV", "development"),
			FrontendURL: getEnv("FRONTEND_URL", "http://localhost:3000"),
		},
		Database: DatabaseConfig{
			URL:             getEnv("DATABASE_URL", "postgresql://postgres:password@localhost:5432/servicepro?sslmode=disable"),
			MaxOpenConns:    getEnvAsInt("DB_MAX_OPEN_CONNS", 25),
			MaxIdleConns:    getEnvAsInt("DB_MAX_IDLE_CONNS", 5),
			ConnMaxLifetime: getEnvAsDuration("DB_CONN_MAX_LIFETIME", "5m"),
		},
		Redis: RedisConfig{
			URL:      getEnv("REDIS_URL", "redis://localhost:6379"),
			Password: getEnv("REDIS_PASSWORD", ""),
			DB:       getEnvAsInt("REDIS_DB", 0),
		},
		JWT: JWTConfig{
			Secret:            getEnv("JWT_SECRET", "your-secret-key-change-this-in-production"),
			AccessExpiration:  time.Hour,          // 1 hour as per requirements
			RefreshExpiration: time.Hour * 24 * 7, // 7 days as per requirements
		},
		Auth: AuthConfig{
			MaxLoginAttempts:  5,                // 5 failed attempts before lockout
			LockoutDuration:   time.Minute * 30, // 30 minutes lockout
			RateLimitWindow:   time.Minute * 15, // 15 minutes window
			RateLimitAttempts: 5,                // 5 attempts per window
		},
		AWS: AWSConfig{
			Region:          getEnv("AWS_REGION", "us-east-1"),
			SESFromEmail:    getEnv("AWS_SES_FROM_EMAIL", "noreply@servicepro.com"),
			S3Bucket:        getEnv("AWS_S3_BUCKET", "servicepro-uploads"),
			AccessKeyID:     getEnv("AWS_ACCESS_KEY_ID", ""),
			SecretAccessKey: getEnv("AWS_SECRET_ACCESS_KEY", ""),
		},
		R2: R2Config{
			AccountID:       getEnv("R2_ACCOUNT_ID", ""),
			AccessKeyID:     getEnv("R2_ACCESS_KEY_ID", ""),
			SecretAccessKey: getEnv("R2_SECRET_ACCESS_KEY", ""),
			Bucket:          getEnv("R2_BUCKET", "servicepro-uploads"),
			Endpoint:        getEnv("R2_ENDPOINT", ""),
			PublicURL:       getEnv("R2_PUBLIC_URL", ""),
		},
		Resend: ResendConfig{
			APIKey:    getEnv("RESEND_API_KEY", ""),
			FromEmail: getEnv("RESEND_FROM_EMAIL", "noreply@servicepro.com"),
			FromName:  getEnv("RESEND_FROM_NAME", "ServicePro"),
		},
		BetterStack: BetterStackConfig{
			SourceToken: getEnv("LOGTAIL_SOURCE_TOKEN", ""),
			Enabled:     getEnvAsBool("LOGTAIL_ENABLED", true),
		},
		Stripe: StripeConfig{
			SecretKey:      getEnv("STRIPE_SECRET_KEY", ""),
			PublishableKey: getEnv("STRIPE_PUBLISHABLE_KEY", ""),
			WebhookSecret:  getEnv("STRIPE_WEBHOOK_SECRET", ""),
			LogLevel:       getEnv("STRIPE_LOG_LEVEL", "info"),
			MaxRetries:     getEnvAsInt("STRIPE_MAX_RETRIES", 3),
		},
		SES: SESConfig{
			FromEmail:            getEnv("SES_FROM_EMAIL", getEnv("AWS_SES_FROM_EMAIL", "noreply@servicepro.com")),
			FromName:             getEnv("SES_FROM_NAME", "ServicePro"),
			ReplyTo:              getEnv("SES_REPLY_TO", ""),
			ConfigurationSetName: getEnv("SES_CONFIGURATION_SET", ""),
			Domain:               getEnv("SES_DOMAIN", ""),
			MaxSendRate:          getEnvAsFloat("SES_MAX_SEND_RATE", 14.0),
			DailyQuota:           int64(getEnvAsInt("SES_DAILY_QUOTA", 200)),
			MaxRetries:           getEnvAsInt("SES_MAX_RETRIES", 3),
			SandboxMode:          getEnvAsBool("SES_SANDBOX_MODE", true),
			EnableTracking:       getEnvAsBool("SES_ENABLE_TRACKING", true),
			EnableDKIM:           getEnvAsBool("SES_ENABLE_DKIM", true),
			MetricsEnabled:       getEnvAsBool("SES_METRICS_ENABLED", true),
			MetricsNamespace:     getEnv("SES_METRICS_NAMESPACE", "ServicePro/Email"),
			LogLevel:             getEnv("SES_LOG_LEVEL", "info"),
		},
		Sentry: SentryConfig{
			DSN:              getEnv("SENTRY_DSN", ""),
			Environment:      getEnv("SENTRY_ENVIRONMENT", getEnv("ENV", "development")),
			Release:          getEnv("SENTRY_RELEASE", getEnv("APP_VERSION", "unknown")),
			SampleRate:       getEnvAsFloat("SENTRY_SAMPLE_RATE", 1.0),
			TracesSampleRate: getEnvAsFloat("SENTRY_TRACES_SAMPLE_RATE", 0.1),
			Debug:            getEnvAsBool("SENTRY_DEBUG", false),
			ServerName:       getEnv("SENTRY_SERVER_NAME", ""),
		},
		Health: HealthConfig{
			Enabled:       getEnvAsBool("HEALTH_ENABLED", true),
			CheckDatabase: getEnvAsBool("HEALTH_CHECK_DATABASE", true),
			CheckRedis:    getEnvAsBool("HEALTH_CHECK_REDIS", true),
			CheckMemory:   getEnvAsBool("HEALTH_CHECK_MEMORY", true),
			CheckDisk:     getEnvAsBool("HEALTH_CHECK_DISK", false),
			CheckExternal: getEnvAsBool("HEALTH_CHECK_EXTERNAL", false),
		},
		Analytics: AnalyticsConfig{
			Enabled: getEnvAsBool("ANALYTICS_ENABLED", true),
			Debug:   getEnvAsBool("ANALYTICS_DEBUG", false),
		},
		Loki: LokiConfig{
			Enabled:  getEnvAsBool("LOKI_ENABLED", false),
			Endpoint: getEnv("LOKI_ENDPOINT", ""),
			TenantID: getEnv("LOKI_TENANT_ID", ""),
			Username: getEnv("LOKI_USERNAME", ""),
			Password: getEnv("LOKI_PASSWORD", ""),
		},
		Prometheus: PrometheusConfig{
			Enabled:      getEnvAsBool("PROMETHEUS_ENABLED", false),
			MetricsPath:  getEnv("PROMETHEUS_METRICS_PATH", "/metrics"),
			Namespace:    getEnv("PROMETHEUS_NAMESPACE", "servicepro"),
			Subsystem:    getEnv("PROMETHEUS_SUBSYSTEM", ""),
			PushGateway:  getEnv("PROMETHEUS_PUSH_GATEWAY", ""),
			PushInterval: getEnvAsDuration("PROMETHEUS_PUSH_INTERVAL", "15s"),
		},
		OpenTelemetry: OpenTelemetryConfig{
			Enabled:     getEnvAsBool("OTEL_ENABLED", false),
			Endpoint:    getEnv("OTEL_EXPORTER_OTLP_ENDPOINT", ""),
			Insecure:    getEnvAsBool("OTEL_EXPORTER_OTLP_INSECURE", false),
			ServiceName: getEnv("OTEL_SERVICE_NAME", "servicepro"),
			SampleRate:  getEnvAsFloat("OTEL_TRACES_SAMPLE_RATE", 0.1),
		},
		PDF: PDFConfig{
			StorageType: getEnv("PDF_STORAGE_TYPE", "local"),
			LocalPath:   getEnv("PDF_LOCAL_PATH", "./generated_pdfs"),
			S3Bucket:    getEnv("PDF_S3_BUCKET", ""),
			S3Prefix:    getEnv("PDF_S3_PREFIX", "pdfs/"),
			S3PublicURL: getEnv("PDF_S3_PUBLIC_URL", ""),
			CompanyName: getEnv("PDF_COMPANY_NAME", "ServicePro"),
			CompanyLogo: getEnv("PDF_COMPANY_LOGO", ""),
		},
		SNS: SNSConfig{
			Enabled:  getEnvAsBool("SNS_ENABLED", false),
			TopicARN: getEnv("SNS_TOPIC_ARN", ""),
			Region:   getEnv("SNS_REGION", ""),
		},
	}
}

// getEnv reads an environment variable or returns a default value
func getEnv(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}

// getEnvAsInt reads an environment variable as integer or returns a default value
func getEnvAsInt(key string, defaultValue int) int {
	valueStr := os.Getenv(key)
	if valueStr == "" {
		return defaultValue
	}
	value, err := strconv.Atoi(valueStr)
	if err != nil {
		log.Printf("Warning: invalid integer value for %s, using default: %d", key, defaultValue)
		return defaultValue
	}
	return value
}

// getEnvAsDuration reads an environment variable as duration or returns a default value
func getEnvAsDuration(key string, defaultValue string) time.Duration {
	valueStr := os.Getenv(key)
	if valueStr == "" {
		valueStr = defaultValue
	}
	value, err := time.ParseDuration(valueStr)
	if err != nil {
		log.Printf("Warning: invalid duration value for %s, using default: %s", key, defaultValue)
		defaultDuration, _ := time.ParseDuration(defaultValue)
		return defaultDuration
	}
	return value
}

// getEnvAsFloat reads an environment variable as float64 or returns a default value
func getEnvAsFloat(key string, defaultValue float64) float64 {
	valueStr := os.Getenv(key)
	if valueStr == "" {
		return defaultValue
	}
	value, err := strconv.ParseFloat(valueStr, 64)
	if err != nil {
		log.Printf("Warning: invalid float value for %s, using default: %f", key, defaultValue)
		return defaultValue
	}
	return value
}
