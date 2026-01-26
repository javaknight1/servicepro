package config

import (
	"log"
	"os"
	"strconv"
	"strings"
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
	RateLimit     RateLimitConfig
	AWS           AWSConfig
	S3Compatible  S3CompatibleConfig
	SMTP          SMTPConfig
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
	CORS          CORSConfig
}

// AWSConfig holds configuration for AWS-specific services (SES, CloudWatch, SNS)
// Note: For S3-compatible storage, use S3CompatibleConfig instead
type AWSConfig struct {
	Region          string
	SESFromEmail    string
	AccessKeyID     string
	SecretAccessKey string
}

// S3CompatibleConfig holds S3-compatible storage configuration
// Works with any S3-compatible service (AWS S3, Cloudflare R2, MinIO, etc.)
type S3CompatibleConfig struct {
	Endpoint        string // Custom endpoint for S3-compatible services (leave empty for AWS)
	Bucket          string
	Region          string
	AccessKeyID     string
	SecretAccessKey string
	UsePathStyle    bool   // Required for MinIO and some S3-compatible services
	DisableSSL      bool   // For local development (e.g., MinIO without TLS)
	PublicURL       string // Optional: base URL for public access (e.g., CDN URL)
}

// ResendConfig holds Resend email service configuration
type ResendConfig struct {
	APIKey    string
	FromEmail string
	FromName  string
}

// SMTPConfig holds SMTP email configuration (for Mailpit in development)
type SMTPConfig struct {
	Host      string
	Port      int
	Username  string
	Password  string
	FromEmail string
	FromName  string
	UseTLS    bool // Implicit TLS (port 465)
	StartTLS  bool // STARTTLS (port 587)
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

// RateLimitConfig holds API rate limiting configuration
type RateLimitConfig struct {
	// Enabled enables/disables rate limiting
	Enabled bool
	// UseRedis enables Redis-backed rate limiting (falls back to in-memory if false or Redis unavailable)
	UseRedis bool
	// AnonymousLimit is the max requests per minute for unauthenticated users (by IP)
	AnonymousLimit int
	// AuthenticatedLimit is the max requests per minute for authenticated users
	AuthenticatedLimit int
	// AdminLimit is the max requests per minute for admin users
	AdminLimit int
	// BurstMultiplier allows burst capacity (e.g., 1.5 = 50% burst above limit)
	BurstMultiplier float64
	// Window is the time window for rate limiting
	Window time.Duration
	// TrustedProxies is a list of trusted proxy IPs (for X-Forwarded-For handling)
	TrustedProxies []string
	// ExcludedPaths are paths excluded from rate limiting (e.g., health checks)
	ExcludedPaths []string
}

// StripeConfig holds Stripe configuration
type StripeConfig struct {
	SecretKey      string
	PublishableKey string
	WebhookSecret  string
	LogLevel       string
	MaxRetries     int
	Prices         StripePrices
}

// StripePrices holds Stripe price IDs for subscription tiers
type StripePrices struct {
	FreeMonthly  string
	BasicMonthly string
	BasicYearly  string
	ProMonthly   string
	ProYearly    string
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
	// S3Bucket is the S3-compatible bucket for storing PDFs
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

// CORSConfig holds CORS (Cross-Origin Resource Sharing) configuration
type CORSConfig struct {
	// Enabled enables CORS handling
	Enabled bool
	// AllowedOrigins is a list of allowed origins (e.g., ["https://app.servicepro.com"])
	// In development, defaults to ["http://localhost:3000", "http://localhost:5173"]
	// IMPORTANT: Never use "*" in production with credentials
	AllowedOrigins []string
	// AllowedMethods is a list of allowed HTTP methods
	AllowedMethods []string
	// AllowedHeaders is a list of allowed request headers
	AllowedHeaders []string
	// ExposedHeaders is a list of headers exposed to the browser
	ExposedHeaders []string
	// AllowCredentials indicates whether credentials (cookies, auth headers) are allowed
	AllowCredentials bool
	// MaxAge is the max age (in seconds) for preflight request caching
	MaxAge int
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
		RateLimit: RateLimitConfig{
			Enabled:            getEnvAsBool("RATE_LIMIT_ENABLED", true),
			UseRedis:           getEnvAsBool("RATE_LIMIT_USE_REDIS", true),
			AnonymousLimit:     getEnvAsInt("RATE_LIMIT_ANONYMOUS", 100),      // 100 req/min for anonymous
			AuthenticatedLimit: getEnvAsInt("RATE_LIMIT_AUTHENTICATED", 1000), // 1000 req/min for authenticated
			AdminLimit:         getEnvAsInt("RATE_LIMIT_ADMIN", 50),           // 50 req/min for admin endpoints
			BurstMultiplier:    getEnvAsFloat("RATE_LIMIT_BURST", 1.2),        // 20% burst capacity
			Window:             getEnvAsDuration("RATE_LIMIT_WINDOW", "1m"),   // 1 minute window
			TrustedProxies:     getEnvAsStringSlice("RATE_LIMIT_TRUSTED_PROXIES", []string{}),
			ExcludedPaths:      getEnvAsStringSlice("RATE_LIMIT_EXCLUDED_PATHS", []string{"/health", "/api/v1/version"}),
		},
		AWS: AWSConfig{
			Region:          getEnv("AWS_REGION", "us-east-1"),
			SESFromEmail:    getEnv("AWS_SES_FROM_EMAIL", "noreply@servicepro.com"),
			AccessKeyID:     getEnv("AWS_ACCESS_KEY_ID", ""),
			SecretAccessKey: getEnv("AWS_SECRET_ACCESS_KEY", ""),
		},
		S3Compatible: S3CompatibleConfig{
			Endpoint:        getEnv("S3_COMPATIBLE_ENDPOINT", ""),
			Bucket:          getEnv("S3_COMPATIBLE_BUCKET", "servicepro-uploads"),
			Region:          getEnv("S3_COMPATIBLE_REGION", "us-east-1"),
			AccessKeyID:     getEnv("S3_COMPATIBLE_ACCESS_KEY_ID", ""),
			SecretAccessKey: getEnv("S3_COMPATIBLE_SECRET_ACCESS_KEY", ""),
			UsePathStyle:    getEnvAsBool("S3_COMPATIBLE_USE_PATH_STYLE", false),
			DisableSSL:      getEnvAsBool("S3_COMPATIBLE_DISABLE_SSL", false),
			PublicURL:       getEnv("S3_COMPATIBLE_PUBLIC_URL", ""),
		},
		SMTP: SMTPConfig{
			Host:      getEnv("SMTP_HOST", ""),
			Port:      getEnvAsInt("SMTP_PORT", 1025),
			Username:  getEnv("SMTP_USERNAME", ""),
			Password:  getEnv("SMTP_PASSWORD", ""),
			FromEmail: getEnv("SMTP_FROM_EMAIL", "noreply@servicepro.local"),
			FromName:  getEnv("SMTP_FROM_NAME", "ServicePro"),
			UseTLS:    getEnvAsBool("SMTP_USE_TLS", false),
			StartTLS:  getEnvAsBool("SMTP_START_TLS", false),
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
			Prices: StripePrices{
				FreeMonthly:  getEnv("STRIPE_PRICE_FREE_MONTHLY", ""),
				BasicMonthly: getEnv("STRIPE_PRICE_BASIC_MONTHLY", ""),
				BasicYearly:  getEnv("STRIPE_PRICE_BASIC_YEARLY", ""),
				ProMonthly:   getEnv("STRIPE_PRICE_PRO_MONTHLY", ""),
				ProYearly:    getEnv("STRIPE_PRICE_PRO_YEARLY", ""),
			},
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
		CORS: CORSConfig{
			Enabled:          getEnvAsBool("CORS_ENABLED", true),
			AllowedOrigins:   getEnvAsStringSlice("CORS_ALLOWED_ORIGINS", []string{"http://localhost:3000", "http://localhost:5173"}),
			AllowedMethods:   getEnvAsStringSlice("CORS_ALLOWED_METHODS", []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"}),
			AllowedHeaders:   getEnvAsStringSlice("CORS_ALLOWED_HEADERS", []string{"Accept", "Authorization", "Content-Type", "X-Request-ID"}),
			ExposedHeaders:   getEnvAsStringSlice("CORS_EXPOSED_HEADERS", []string{"X-Request-ID", "X-RateLimit-Remaining", "X-RateLimit-Reset"}),
			AllowCredentials: getEnvAsBool("CORS_ALLOW_CREDENTIALS", true),
			MaxAge:           getEnvAsInt("CORS_MAX_AGE", 86400), // 24 hours
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

// getEnvAsStringSlice reads an environment variable as comma-separated strings or returns a default value
func getEnvAsStringSlice(key string, defaultValue []string) []string {
	valueStr := os.Getenv(key)
	if valueStr == "" {
		return defaultValue
	}
	// Split by comma and trim whitespace
	parts := make([]string, 0)
	for _, part := range splitAndTrim(valueStr, ",") {
		if part != "" {
			parts = append(parts, part)
		}
	}
	if len(parts) == 0 {
		return defaultValue
	}
	return parts
}

// splitAndTrim splits a string by separator and trims whitespace from each part
func splitAndTrim(s string, sep string) []string {
	parts := make([]string, 0)
	for _, part := range strings.Split(s, sep) {
		trimmed := strings.TrimSpace(part)
		parts = append(parts, trimmed)
	}
	return parts
}
