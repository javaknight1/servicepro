package health

import (
	"context"
	"database/sql"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/javaknight1/servicepro/backend/config"
	"github.com/javaknight1/servicepro/backend/pkg/clients/logging"
)

// Default intervals for health checks
const (
	defaultCheckInterval    = 30 * time.Second
	defaultDatabaseInterval = 15 * time.Second
	defaultRedisInterval    = 15 * time.Second
	defaultMemoryInterval   = 60 * time.Second

	defaultCheckTimeout    = 5 * time.Second
	defaultDatabaseTimeout = 3 * time.Second
	defaultRedisTimeout    = 2 * time.Second
)

// SetupOptions holds options for setting up health checks
type SetupOptions struct {
	Config         *config.Config
	DB             *sql.DB
	Redis          *redis.Client
	SlackWebhook   string
	SlackChannel   string
	PagerDutyKey   string
	WebhookURL     string
	WebhookHeaders map[string]string
}

// Setup creates and configures a complete health check system
func Setup(opts SetupOptions) (*Checker, *Runner, *UptimeTracker, *Notifier) {
	ctx := context.Background()
	cfg := opts.Config

	// Create checker
	version := "unknown"
	environment := "development"
	if cfg != nil {
		environment = cfg.Server.Env
	}
	checker := NewChecker(version, environment)

	// Register database check
	if (cfg == nil || cfg.Health.CheckDatabase) && opts.DB != nil {
		checker.Register(CheckConfig{
			Name:     "database",
			Timeout:  defaultDatabaseTimeout,
			Interval: defaultDatabaseInterval,
			Critical: true,
			Check:    DatabaseCheck(opts.DB),
		})
		logging.Info(ctx, "Registered database check", map[string]any{"source": "health"})
	}

	// Register Redis check
	if (cfg == nil || cfg.Health.CheckRedis) && opts.Redis != nil {
		checker.Register(CheckConfig{
			Name:     "redis",
			Timeout:  defaultRedisTimeout,
			Interval: defaultRedisInterval,
			Critical: false, // Redis may be optional
			Check:    RedisCheck(opts.Redis),
		})
		logging.Info(ctx, "Registered Redis check", map[string]any{"source": "health"})
	}

	// Register memory check
	if cfg == nil || cfg.Health.CheckMemory {
		checker.Register(CheckConfig{
			Name:     "memory",
			Timeout:  defaultCheckTimeout,
			Interval: defaultMemoryInterval,
			Critical: false,
			Check:    MemoryCheck(80.0, 95.0), // Warning at 80%, critical at 95%
		})
		logging.Info(ctx, "Registered memory check", map[string]any{"source": "health"})
	}

	// Create uptime tracker
	uptimeTracker := NewUptimeTracker(1000) // Keep 1000 history entries

	// Create runner
	runner := NewRunner(checker, RunnerConfig{
		DefaultInterval: defaultCheckInterval,
	})

	// Subscribe uptime tracker to runner
	runner.Subscribe(uptimeTracker)

	// Create notifier with default config
	notifier := NewNotifier(DefaultNotificationConfig(), environment)

	// Add notification channels
	if opts.SlackWebhook != "" {
		channel := opts.SlackChannel
		if channel == "" {
			channel = "#alerts"
		}
		notifier.AddChannel(NewSlackChannel(opts.SlackWebhook, channel, "Health Check Bot"))
		logging.Info(ctx, "Added Slack notification channel", map[string]any{"source": "health"})
	}

	if opts.PagerDutyKey != "" {
		notifier.AddChannel(NewPagerDutyChannel(opts.PagerDutyKey))
		logging.Info(ctx, "Added PagerDuty notification channel", map[string]any{"source": "health"})
	}

	if opts.WebhookURL != "" {
		notifier.AddChannel(NewWebhookChannel(opts.WebhookURL, opts.WebhookHeaders))
		logging.Info(ctx, "Added webhook notification channel", map[string]any{"source": "health"})
	}

	// Always add log channel
	notifier.AddChannel(NewLogChannel("[Health]"))

	// Subscribe notifier to runner
	runner.Subscribe(notifier)

	return checker, runner, uptimeTracker, notifier
}

// QuickSetup creates a simple health check system with minimal configuration
func QuickSetup(db *sql.DB, redisClient *redis.Client) (*Checker, *Runner, *UptimeTracker) {
	checker, runner, uptimeTracker, _ := Setup(SetupOptions{
		Config: nil, // Use defaults
		DB:     db,
		Redis:  redisClient,
	})
	return checker, runner, uptimeTracker
}

// StartHealthSystem starts the health check system
func StartHealthSystem(runner *Runner) {
	runner.Start()
	logging.Info(context.Background(), "Health check system started", map[string]any{"source": "health"})
}

// StopHealthSystem stops the health check system
func StopHealthSystem(runner *Runner) {
	runner.Stop()
	logging.Info(context.Background(), "Health check system stopped", map[string]any{"source": "health"})
}
