package health

import (
	"os"
	"strconv"
	"time"
)

// Config holds all health check configuration
type Config struct {
	// Enabled enables/disables health checks
	Enabled bool

	// Version is the application version
	Version string

	// Environment is the current environment
	Environment string

	// Checks holds configuration for individual checks
	Checks ChecksConfig

	// Intervals holds check intervals
	Intervals IntervalsConfig

	// Timeouts holds check timeouts
	Timeouts TimeoutsConfig

	// Notifications holds notification configuration
	Notifications NotificationConfig

	// Dependencies holds external dependency configurations
	Dependencies DependenciesConfig
}

// ChecksConfig holds which checks are enabled
type ChecksConfig struct {
	Database    bool
	Redis       bool
	Memory      bool
	DiskSpace   bool
	ExternalAPI bool
}

// IntervalsConfig holds check intervals
type IntervalsConfig struct {
	Default     time.Duration
	Database    time.Duration
	Redis       time.Duration
	Memory      time.Duration
	DiskSpace   time.Duration
	ExternalAPI time.Duration
}

// TimeoutsConfig holds check timeouts
type TimeoutsConfig struct {
	Default     time.Duration
	Database    time.Duration
	Redis       time.Duration
	ExternalAPI time.Duration
}

// DependenciesConfig holds external dependency configurations
type DependenciesConfig struct {
	// External service URLs to health check
	ExternalServices []ExternalServiceConfig
}

// ExternalServiceConfig holds configuration for an external service check
type ExternalServiceConfig struct {
	Name           string
	URL            string
	ExpectedStatus int
	Critical       bool
	Timeout        time.Duration
	Interval       time.Duration
}

// DefaultConfig returns default health check configuration
func DefaultConfig() Config {
	return Config{
		Enabled:     true,
		Version:     getEnv("APP_VERSION", "unknown"),
		Environment: getEnv("ENVIRONMENT", "development"),
		Checks: ChecksConfig{
			Database:    true,
			Redis:       true,
			Memory:      true,
			DiskSpace:   false,
			ExternalAPI: false,
		},
		Intervals: IntervalsConfig{
			Default:     30 * time.Second,
			Database:    15 * time.Second,
			Redis:       15 * time.Second,
			Memory:      60 * time.Second,
			DiskSpace:   5 * time.Minute,
			ExternalAPI: 60 * time.Second,
		},
		Timeouts: TimeoutsConfig{
			Default:     5 * time.Second,
			Database:    3 * time.Second,
			Redis:       2 * time.Second,
			ExternalAPI: 10 * time.Second,
		},
		Notifications: DefaultNotificationConfig(),
	}
}

// LoadConfigFromEnv loads configuration from environment variables
func LoadConfigFromEnv() Config {
	config := DefaultConfig()

	// Basic settings
	config.Enabled = getEnvBool("HEALTH_ENABLED", true)
	config.Version = getEnv("APP_VERSION", "unknown")
	config.Environment = getEnv("ENVIRONMENT", "development")

	// Check enables
	config.Checks.Database = getEnvBool("HEALTH_CHECK_DATABASE", true)
	config.Checks.Redis = getEnvBool("HEALTH_CHECK_REDIS", true)
	config.Checks.Memory = getEnvBool("HEALTH_CHECK_MEMORY", true)
	config.Checks.DiskSpace = getEnvBool("HEALTH_CHECK_DISK", false)
	config.Checks.ExternalAPI = getEnvBool("HEALTH_CHECK_EXTERNAL", false)

	// Intervals
	config.Intervals.Default = getEnvDuration("HEALTH_INTERVAL_DEFAULT", 30*time.Second)
	config.Intervals.Database = getEnvDuration("HEALTH_INTERVAL_DATABASE", 15*time.Second)
	config.Intervals.Redis = getEnvDuration("HEALTH_INTERVAL_REDIS", 15*time.Second)
	config.Intervals.Memory = getEnvDuration("HEALTH_INTERVAL_MEMORY", 60*time.Second)
	config.Intervals.DiskSpace = getEnvDuration("HEALTH_INTERVAL_DISK", 5*time.Minute)
	config.Intervals.ExternalAPI = getEnvDuration("HEALTH_INTERVAL_EXTERNAL", 60*time.Second)

	// Timeouts
	config.Timeouts.Default = getEnvDuration("HEALTH_TIMEOUT_DEFAULT", 5*time.Second)
	config.Timeouts.Database = getEnvDuration("HEALTH_TIMEOUT_DATABASE", 3*time.Second)
	config.Timeouts.Redis = getEnvDuration("HEALTH_TIMEOUT_REDIS", 2*time.Second)
	config.Timeouts.ExternalAPI = getEnvDuration("HEALTH_TIMEOUT_EXTERNAL", 10*time.Second)

	// Notifications
	config.Notifications.Enabled = getEnvBool("HEALTH_NOTIFICATIONS_ENABLED", true)
	config.Notifications.MinInterval = getEnvDuration("HEALTH_NOTIFICATIONS_MIN_INTERVAL", 5*time.Minute)
	config.Notifications.OnlyOnStatusChange = getEnvBool("HEALTH_NOTIFICATIONS_STATUS_CHANGE_ONLY", true)
	config.Notifications.IncludeRecovery = getEnvBool("HEALTH_NOTIFICATIONS_INCLUDE_RECOVERY", true)
	config.Notifications.AlertThreshold = getEnvInt("HEALTH_NOTIFICATIONS_ALERT_THRESHOLD", 1)

	return config
}

// Validate validates the configuration
func (c *Config) Validate() error {
	if c.Intervals.Default <= 0 {
		c.Intervals.Default = 30 * time.Second
	}
	if c.Timeouts.Default <= 0 {
		c.Timeouts.Default = 5 * time.Second
	}
	return nil
}

// Helper functions

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if parsed, err := strconv.ParseBool(value); err == nil {
			return parsed
		}
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if parsed, err := strconv.Atoi(value); err == nil {
			return parsed
		}
	}
	return defaultValue
}

func getEnvDuration(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if parsed, err := time.ParseDuration(value); err == nil {
			return parsed
		}
	}
	return defaultValue
}

// HealthCheckPresets provides common health check configurations
type HealthCheckPresets struct{}

// Development returns health check config for development
func (HealthCheckPresets) Development() Config {
	config := DefaultConfig()
	config.Environment = "development"
	config.Intervals.Default = 60 * time.Second
	config.Notifications.Enabled = false
	return config
}

// Staging returns health check config for staging
func (HealthCheckPresets) Staging() Config {
	config := DefaultConfig()
	config.Environment = "staging"
	config.Intervals.Default = 30 * time.Second
	config.Notifications.Enabled = true
	config.Notifications.AlertThreshold = 2
	return config
}

// Production returns health check config for production
func (HealthCheckPresets) Production() Config {
	config := DefaultConfig()
	config.Environment = "production"
	config.Intervals.Default = 15 * time.Second
	config.Intervals.Database = 10 * time.Second
	config.Intervals.Redis = 10 * time.Second
	config.Notifications.Enabled = true
	config.Notifications.AlertThreshold = 1
	config.Notifications.MinInterval = 1 * time.Minute
	return config
}

// Presets provides access to preset configurations
var Presets = HealthCheckPresets{}
