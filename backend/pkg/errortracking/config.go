package errortracking

import (
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/getsentry/sentry-go"
)

// Environment variable names
const (
	EnvSentryDSN              = "SENTRY_DSN"
	EnvSentryEnvironment      = "SENTRY_ENVIRONMENT"
	EnvSentryRelease          = "SENTRY_RELEASE"
	EnvSentrySampleRate       = "SENTRY_SAMPLE_RATE"
	EnvSentryTracesSampleRate = "SENTRY_TRACES_SAMPLE_RATE"
	EnvSentryDebug            = "SENTRY_DEBUG"
	EnvSentryServerName       = "SENTRY_SERVER_NAME"
	EnvEnvironment            = "ENVIRONMENT"
	EnvAppVersion             = "APP_VERSION"
)

// ConfigFromEnv creates a Config from environment variables
func ConfigFromEnv() Config {
	cfg := Config{
		DSN:              getEnv(EnvSentryDSN, ""),
		Environment:      getEnv(EnvSentryEnvironment, getEnv(EnvEnvironment, "development")),
		Release:          getEnv(EnvSentryRelease, getEnv(EnvAppVersion, "unknown")),
		SampleRate:       getEnvFloat(EnvSentrySampleRate, 1.0),
		TracesSampleRate: getEnvFloat(EnvSentryTracesSampleRate, 0.1),
		Debug:            getEnvBool(EnvSentryDebug, false),
		ServerName:       getEnv(EnvSentryServerName, getHostname()),
		BeforeSend:       DefaultBeforeSend,
	}

	// Adjust sample rates based on environment
	if cfg.Environment == "production" {
		// Lower sample rates in production to reduce costs
		if cfg.SampleRate == 1.0 {
			cfg.SampleRate = 0.5
		}
		if cfg.TracesSampleRate == 0.1 {
			cfg.TracesSampleRate = 0.05
		}
	}

	return cfg
}

// SamplingConfig holds configuration for error and trace sampling
type SamplingConfig struct {
	// ErrorSampleRate is the rate at which errors are sampled (0.0-1.0)
	ErrorSampleRate float64

	// TracesSampleRate is the rate at which traces are sampled (0.0-1.0)
	TracesSampleRate float64

	// HighPriorityErrors are error types that should always be captured
	HighPriorityErrors []string

	// LowPriorityErrors are error types that should be sampled at a lower rate
	LowPriorityErrors []string

	// LowPrioritySampleRate is the rate for low priority errors
	LowPrioritySampleRate float64
}

// DefaultSamplingConfig returns default sampling configuration
func DefaultSamplingConfig() SamplingConfig {
	return SamplingConfig{
		ErrorSampleRate:  1.0,
		TracesSampleRate: 0.1,
		HighPriorityErrors: []string{
			"panic",
			"fatal",
			"critical",
			"security",
		},
		LowPriorityErrors: []string{
			"validation",
			"not_found",
			"bad_request",
		},
		LowPrioritySampleRate: 0.1,
	}
}

// CreateBeforeSendWithSampling creates a BeforeSend function with custom sampling
func CreateBeforeSendWithSampling(samplingCfg SamplingConfig) func(event *sentry.Event, hint *sentry.EventHint) *sentry.Event {
	return func(event *sentry.Event, hint *sentry.EventHint) *sentry.Event {
		// Always apply base filtering
		event = sanitizeEvent(event)

		if shouldFilterError(event, hint) {
			return nil
		}

		// Check for high priority errors - always capture
		for _, tag := range samplingCfg.HighPriorityErrors {
			if containsTag(event, tag) {
				return event
			}
		}

		// Check for low priority errors - sample at lower rate
		for _, tag := range samplingCfg.LowPriorityErrors {
			if containsTag(event, tag) {
				if !shouldSample(samplingCfg.LowPrioritySampleRate) {
					return nil
				}
				return event
			}
		}

		// Apply default error sample rate
		if !shouldSample(samplingCfg.ErrorSampleRate) {
			return nil
		}

		return event
	}
}

// FilterRules defines rules for filtering errors
type FilterRules struct {
	// IgnoreErrors are error message patterns to ignore
	IgnoreErrors []string

	// IgnoreTransactions are transaction name patterns to ignore
	IgnoreTransactions []string

	// IgnoreURLs are URL patterns to ignore
	IgnoreURLs []string

	// AllowURLs are URL patterns to allow (if set, only these are captured)
	AllowURLs []string

	// IgnoreUserAgents are user agent patterns to ignore
	IgnoreUserAgents []string
}

// DefaultFilterRules returns default filtering rules
func DefaultFilterRules() FilterRules {
	return FilterRules{
		IgnoreErrors: []string{
			"context canceled",
			"context deadline exceeded",
			"client disconnected",
			"broken pipe",
			"connection reset by peer",
			"write: connection reset",
			"net/http: request canceled",
			"EOF",
		},
		IgnoreTransactions: []string{
			"GET /health",
			"GET /ready",
			"GET /metrics",
			"GET /favicon.ico",
		},
		IgnoreURLs: []string{
			"/health",
			"/ready",
			"/metrics",
			"/favicon.ico",
			"/.well-known/",
		},
		IgnoreUserAgents: []string{
			"kube-probe",
			"GoogleHC",
			"ELB-HealthChecker",
		},
	}
}

// CreateFilteredBeforeSend creates a BeforeSend function with custom filter rules
func CreateFilteredBeforeSend(rules FilterRules) func(event *sentry.Event, hint *sentry.EventHint) *sentry.Event {
	return func(event *sentry.Event, hint *sentry.EventHint) *sentry.Event {
		// Apply base sanitization
		event = sanitizeEvent(event)

		// Check ignored errors
		if hint != nil && hint.OriginalException != nil {
			errMsg := hint.OriginalException.Error()
			for _, pattern := range rules.IgnoreErrors {
				if strings.Contains(strings.ToLower(errMsg), strings.ToLower(pattern)) {
					return nil
				}
			}
		}

		// Check event message
		if event.Message != "" {
			for _, pattern := range rules.IgnoreErrors {
				if strings.Contains(strings.ToLower(event.Message), strings.ToLower(pattern)) {
					return nil
				}
			}
		}

		// Check ignored transactions
		if event.Transaction != "" {
			for _, pattern := range rules.IgnoreTransactions {
				if matchesPattern(event.Transaction, pattern) {
					return nil
				}
			}
		}

		// Check ignored URLs
		if event.Request != nil && event.Request.URL != "" {
			// Check allow list first
			if len(rules.AllowURLs) > 0 {
				allowed := false
				for _, pattern := range rules.AllowURLs {
					if matchesPattern(event.Request.URL, pattern) {
						allowed = true
						break
					}
				}
				if !allowed {
					return nil
				}
			}

			// Check deny list
			for _, pattern := range rules.IgnoreURLs {
				if matchesPattern(event.Request.URL, pattern) {
					return nil
				}
			}
		}

		// Check ignored user agents
		if event.Request != nil {
			userAgent := event.Request.Headers["User-Agent"]
			for _, pattern := range rules.IgnoreUserAgents {
				if strings.Contains(strings.ToLower(userAgent), strings.ToLower(pattern)) {
					return nil
				}
			}
		}

		return event
	}
}

// RateLimitConfig holds configuration for rate limiting error reports
type RateLimitConfig struct {
	// MaxEventsPerSecond is the maximum number of events to send per second
	MaxEventsPerSecond int

	// BurstSize is the maximum burst size
	BurstSize int

	// PerKeyLimits enables per-error-type limiting
	PerKeyLimits bool
}

// Helper functions

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvFloat(key string, defaultValue float64) float64 {
	if value := os.Getenv(key); value != "" {
		if f, err := strconv.ParseFloat(value, 64); err == nil {
			return f
		}
	}
	return defaultValue
}

func getEnvBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		return strings.ToLower(value) == "true" || value == "1"
	}
	return defaultValue
}

func containsTag(event *sentry.Event, tag string) bool {
	for _, t := range event.Tags {
		if strings.Contains(strings.ToLower(t), strings.ToLower(tag)) {
			return true
		}
	}
	return false
}

func shouldSample(rate float64) bool {
	if rate >= 1.0 {
		return true
	}
	if rate <= 0 {
		return false
	}
	// Simple random sampling based on time
	return float64(time.Now().UnixNano()%1000)/1000 < rate
}

func matchesPattern(s, pattern string) bool {
	// Simple pattern matching - supports * as wildcard
	if pattern == "*" {
		return true
	}

	if strings.HasPrefix(pattern, "*") && strings.HasSuffix(pattern, "*") {
		return strings.Contains(s, pattern[1:len(pattern)-1])
	}

	if strings.HasPrefix(pattern, "*") {
		return strings.HasSuffix(s, pattern[1:])
	}

	if strings.HasSuffix(pattern, "*") {
		return strings.HasPrefix(s, pattern[:len(pattern)-1])
	}

	return s == pattern
}
