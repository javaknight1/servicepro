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
	Cookie        CookieConfig
	RateLimit     RateLimitConfig
	AWS           AWSConfig
	S3Compatible  S3CompatibleConfig
	SMTP          SMTPConfig
	Resend        ResendConfig
	BetterStack   BetterStackConfig
	Stripe        StripeConfig
	Sentry        SentryConfig
	Health        HealthConfig
	Analytics     AnalyticsConfig
	Loki          LokiConfig
	Prometheus    PrometheusConfig
	OpenTelemetry OpenTelemetryConfig
	PDF           PDFConfig
	CORS          CORSConfig
	SMS           SMSConfig
}

// SMSConfig holds SMS service configuration
type SMSConfig struct {
	// DefaultProvider can be set to force a specific provider
	// If empty, auto-detection is used based on available credentials
	DefaultProvider string

	// SenderID is the sender name/number shown to recipients (where supported)
	SenderID string

	// TextBelt configuration (free tier provider)
	TextBelt TextBeltConfig

	// Rate limiting
	RateLimit SMSRateLimitConfig
}

// TextBeltConfig holds TextBelt API configuration
// TextBelt offers 1 free SMS/day, or unlimited with a paid key
// Website: https://textbelt.com/
type TextBeltConfig struct {
	// APIKey is the TextBelt API key
	// Use "textbelt" for free tier (1 SMS/day)
	// Use a paid key for production
	APIKey string

	// Enabled allows explicitly enabling/disabling TextBelt
	Enabled bool
}

// SMSRateLimitConfig holds SMS rate limiting configuration
type SMSRateLimitConfig struct {
	// Enabled enables rate limiting for SMS
	Enabled bool

	// MaxPerMinute is the maximum SMS per minute
	MaxPerMinute int

	// MaxPerDay is the maximum SMS per day
	MaxPerDay int

	// BurstSize is the maximum burst size
	BurstSize int
}

// AWSConfig holds configuration for all AWS services (SES, SNS, CloudWatch, etc.)
// Note: For S3-compatible storage, use S3CompatibleConfig instead
type AWSConfig struct {
	// Shared AWS credentials and region
	Region          string
	AccessKeyID     string
	SecretAccessKey string

	// AWS service-specific configurations
	SES AWSSESConfig
	SNS AWSSNSConfig
}

// AWSSESConfig holds AWS SES email service configuration
type AWSSESConfig struct {
	FromEmail            string
	FromName             string
	ReplyTo              string
	ConfigurationSetName string  // SES configuration set for tracking
	Domain               string  // Verified domain for sending
	MaxSendRate          float64 // Max emails per second
	DailyQuota           int64   // Daily sending limit
	MaxRetries           int     // Retry attempts for failed sends
	SandboxMode          bool    // Whether SES is in sandbox mode
	EnableTracking       bool    // Enable open/click tracking
	EnableDKIM           bool    // Enable DKIM signing
	MetricsEnabled       bool    // Enable CloudWatch metrics
	MetricsNamespace     string  // CloudWatch namespace for metrics
	LogLevel             string  // Logging level (debug, info, warn, error)
}

// AWSSNSConfig holds AWS SNS configuration
type AWSSNSConfig struct {
	Enabled  bool
	TopicARN string
	Region   string // Optional: overrides AWS.Region for SNS
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
	BackendURL  string
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

// CookieConfig holds HTTP cookie configuration for JWT tokens
type CookieConfig struct {
	// Domain is the domain for cookies (e.g., ".servicepro.com" for production)
	Domain string
	// Secure indicates if cookies should only be sent over HTTPS
	Secure bool
	// SameSite controls cross-site cookie behavior ("Strict", "Lax", or "None")
	SameSite string
	// AccessTokenName is the cookie name for the access token
	AccessTokenName string
	// RefreshTokenName is the cookie name for the refresh token
	RefreshTokenName string
	// RefreshTokenPath restricts refresh token cookie to auth endpoints
	RefreshTokenPath string
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

// SentryConfig holds error tracking configuration
type SentryConfig struct {
	DSN              string  // Sentry DSN for error reporting
	Environment      string  // Environment name (development, staging, production)
	Release          string  // Application version/release
	SampleRate       float64 // Error sample rate (0.0 to 1.0)
	TracesSampleRate float64 // Performance tracing sample rate (0.0 to 1.0)
	Debug            bool    // Enable Sentry debug mode
	ServerName       string  // Server identifier
}

// HealthConfig holds health check configuration
type HealthConfig struct {
	Enabled       bool // Enable health check endpoints
	CheckDatabase bool // Include database connectivity check
	CheckRedis    bool // Include Redis connectivity check
	CheckMemory   bool // Include memory usage check
	CheckDisk     bool // Include disk space check
	CheckExternal bool // Include external service checks
}

// AnalyticsConfig holds analytics tracking configuration
type AnalyticsConfig struct {
	Enabled bool // Enable analytics tracking
	Debug   bool // Enable debug logging for analytics
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
	Enabled      bool          // Enable Prometheus metrics
	MetricsPath  string        // HTTP path for metrics endpoint (default: /metrics)
	Namespace    string        // Metrics namespace prefix
	Subsystem    string        // Metrics subsystem prefix
	PushGateway  string        // Push gateway URL (for batch jobs)
	PushInterval time.Duration // Interval for pushing metrics
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
// Default values that should NEVER be used in production
const (
	insecureJWTSecretDefault = "your-secret-key-change-this-in-production"
)

func Load() *Config {
	// Load .env file if it exists (for local development)
	_ = godotenv.Load()

	cfg := &Config{
		Server: ServerConfig{
			Port:        getEnv("PORT", "8080"),
			Env:         getEnv("ENV", "development"),
			FrontendURL: getEnv("FRONTEND_URL", "http://localhost:3000"),
			BackendURL:  getEnv("BACKEND_URL", "http://localhost:8080"),
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
		Cookie: CookieConfig{
			Domain:           getEnv("COOKIE_DOMAIN", ""),          // Empty for development (localhost)
			Secure:           getEnvAsBool("COOKIE_SECURE", false), // true in production (HTTPS only)
			SameSite:         getEnv("COOKIE_SAMESITE", "Lax"),     // Lax provides CSRF protection
			AccessTokenName:  getEnv("COOKIE_ACCESS_TOKEN_NAME", "access_token"),
			RefreshTokenName: getEnv("COOKIE_REFRESH_TOKEN_NAME", "refresh_token"),
			RefreshTokenPath: getEnv("COOKIE_REFRESH_TOKEN_PATH", "/api/v1/auth"),
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
			AccessKeyID:     getEnv("AWS_ACCESS_KEY_ID", ""),
			SecretAccessKey: getEnv("AWS_SECRET_ACCESS_KEY", ""),
			SES: AWSSESConfig{
				FromEmail:            getEnv("AWS_SES_FROM_EMAIL", "noreply@servicepro.com"),
				FromName:             getEnv("AWS_SES_FROM_NAME", "ServicePro"),
				ReplyTo:              getEnv("AWS_SES_REPLY_TO", ""),
				ConfigurationSetName: getEnv("AWS_SES_CONFIGURATION_SET", ""),
				Domain:               getEnv("AWS_SES_DOMAIN", ""),
				MaxSendRate:          getEnvAsFloat("AWS_SES_MAX_SEND_RATE", 14.0),
				DailyQuota:           int64(getEnvAsInt("AWS_SES_DAILY_QUOTA", 200)),
				MaxRetries:           getEnvAsInt("AWS_SES_MAX_RETRIES", 3),
				SandboxMode:          getEnvAsBool("AWS_SES_SANDBOX_MODE", true),
				EnableTracking:       getEnvAsBool("AWS_SES_ENABLE_TRACKING", true),
				EnableDKIM:           getEnvAsBool("AWS_SES_ENABLE_DKIM", true),
				MetricsEnabled:       getEnvAsBool("AWS_SES_METRICS_ENABLED", true),
				MetricsNamespace:     getEnv("AWS_SES_METRICS_NAMESPACE", "ServicePro/Email"),
				LogLevel:             getEnv("AWS_SES_LOG_LEVEL", "info"),
			},
			SNS: AWSSNSConfig{
				Enabled:  getEnvAsBool("AWS_SNS_ENABLED", false),
				TopicARN: getEnv("AWS_SNS_TOPIC_ARN", ""),
				Region:   getEnv("AWS_SNS_REGION", ""),
			},
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
			WebhookSecret:  getEnvOrFile("STRIPE_WEBHOOK_SECRET", "STRIPE_WEBHOOK_SECRET_FILE", ""),
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
		CORS: CORSConfig{
			Enabled:          getEnvAsBool("CORS_ENABLED", true),
			AllowedOrigins:   getEnvAsStringSlice("CORS_ALLOWED_ORIGINS", []string{"http://localhost:3000", "http://localhost:5173"}),
			AllowedMethods:   getEnvAsStringSlice("CORS_ALLOWED_METHODS", []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"}),
			AllowedHeaders:   getEnvAsStringSlice("CORS_ALLOWED_HEADERS", []string{"Accept", "Authorization", "Content-Type", "X-Request-ID"}),
			ExposedHeaders:   getEnvAsStringSlice("CORS_EXPOSED_HEADERS", []string{"X-Request-ID", "X-RateLimit-Remaining", "X-RateLimit-Reset"}),
			AllowCredentials: getEnvAsBool("CORS_ALLOW_CREDENTIALS", true),
			MaxAge:           getEnvAsInt("CORS_MAX_AGE", 86400), // 24 hours
		},
		SMS: SMSConfig{
			DefaultProvider: getEnv("SMS_DEFAULT_PROVIDER", ""),
			SenderID:        getEnv("SMS_SENDER_ID", "ServicePro"),
			TextBelt: TextBeltConfig{
				APIKey:  getEnv("TEXTBELT_API_KEY", ""),
				Enabled: getEnvAsBool("TEXTBELT_ENABLED", false),
			},
			RateLimit: SMSRateLimitConfig{
				Enabled:      getEnvAsBool("SMS_RATE_LIMIT_ENABLED", true),
				MaxPerMinute: getEnvAsInt("SMS_RATE_LIMIT_MAX_PER_MINUTE", 10),
				MaxPerDay:    getEnvAsInt("SMS_RATE_LIMIT_MAX_PER_DAY", 100),
				BurstSize:    getEnvAsInt("SMS_RATE_LIMIT_BURST_SIZE", 5),
			},
		},
	}

	// Validate configuration for production environments
	validateConfig(cfg)

	return cfg
}

// validateConfig checks that security-critical configuration is properly set.
// Missing or insecure values will cause the application to exit in ALL environments.
// Security should be consistent - there's no good reason to run with insecure defaults.
func validateConfig(cfg *Config) {
	var errors []string

	// Check JWT secret - required in all environments
	if cfg.JWT.Secret == "" {
		errors = append(errors, "JWT_SECRET is required but not set")
	} else if cfg.JWT.Secret == insecureJWTSecretDefault {
		errors = append(errors, "JWT_SECRET is using the insecure default value - please set a unique secret (hint: use 'openssl rand -base64 32' to generate one)")
	} else if len(cfg.JWT.Secret) < 32 {
		errors = append(errors, "JWT_SECRET must be at least 32 characters for security")
	}

	// Check Stripe webhook secret (required if Stripe is configured)
	if cfg.Stripe.SecretKey != "" && cfg.Stripe.WebhookSecret == "" {
		errors = append(errors, "STRIPE_WEBHOOK_SECRET is required when Stripe is configured - webhooks without signature verification are a security risk")
	}

	// Any configuration errors are fatal
	if len(errors) > 0 {
		log.Println("[CONFIG] Configuration errors detected:")
		for _, err := range errors {
			log.Printf("[CONFIG]   - %s", err)
		}
		log.Println("[CONFIG]")
		log.Println("[CONFIG] To fix: copy .env.example to .env and configure the required values")
		log.Println("[CONFIG] Generate a secure JWT secret with: openssl rand -base64 32")
		log.Fatal("[CONFIG] Application cannot start with insecure configuration.")
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

// getEnvOrFile returns the value from an environment variable, or if empty,
// tries to read from a file specified by fileEnvKey. Falls back to defaultValue.
// This is useful for secrets that may be provided by external processes (e.g., Stripe CLI).
func getEnvOrFile(key, fileEnvKey, defaultValue string) string {
	// First, try the environment variable
	value := os.Getenv(key)
	if value != "" {
		return value
	}

	// If a file path is specified, try to read from it
	filePath := os.Getenv(fileEnvKey)
	if filePath != "" {
		data, err := os.ReadFile(filePath)
		if err == nil {
			value = strings.TrimSpace(string(data))
			if value != "" {
				log.Printf("[CONFIG] Loaded %s from file: %s", key, filePath)
				return value
			}
		}
		// File doesn't exist yet - this is okay for Stripe CLI which writes it after startup
	}

	return defaultValue
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
