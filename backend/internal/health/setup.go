package health

import (
	"database/sql"
	"log"

	"github.com/redis/go-redis/v9"
)

// SetupOptions holds options for setting up health checks
type SetupOptions struct {
	Config         Config
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
	// Create checker
	checker := NewChecker(opts.Config.Version, opts.Config.Environment)

	// Register database check
	if opts.Config.Checks.Database && opts.DB != nil {
		checker.Register(CheckConfig{
			Name:     "database",
			Timeout:  opts.Config.Timeouts.Database,
			Interval: opts.Config.Intervals.Database,
			Critical: true,
			Check:    DatabaseCheck(opts.DB),
		})
		log.Println("[Health] Registered database check")
	}

	// Register Redis check
	if opts.Config.Checks.Redis && opts.Redis != nil {
		checker.Register(CheckConfig{
			Name:     "redis",
			Timeout:  opts.Config.Timeouts.Redis,
			Interval: opts.Config.Intervals.Redis,
			Critical: false, // Redis may be optional
			Check:    RedisCheck(opts.Redis),
		})
		log.Println("[Health] Registered Redis check")
	}

	// Register memory check
	if opts.Config.Checks.Memory {
		checker.Register(CheckConfig{
			Name:     "memory",
			Timeout:  opts.Config.Timeouts.Default,
			Interval: opts.Config.Intervals.Memory,
			Critical: false,
			Check:    MemoryCheck(80.0, 95.0), // Warning at 80%, critical at 95%
		})
		log.Println("[Health] Registered memory check")
	}

	// Register external service checks
	for _, svc := range opts.Config.Dependencies.ExternalServices {
		timeout := svc.Timeout
		if timeout == 0 {
			timeout = opts.Config.Timeouts.ExternalAPI
		}
		interval := svc.Interval
		if interval == 0 {
			interval = opts.Config.Intervals.ExternalAPI
		}

		checker.Register(CheckConfig{
			Name:     svc.Name,
			Timeout:  timeout,
			Interval: interval,
			Critical: svc.Critical,
			Check:    HTTPCheck(svc.Name, svc.URL, svc.ExpectedStatus),
		})
		log.Printf("[Health] Registered external service check: %s", svc.Name)
	}

	// Create uptime tracker
	uptimeTracker := NewUptimeTracker(1000) // Keep 1000 history entries

	// Create runner
	runner := NewRunner(checker, RunnerConfig{
		DefaultInterval: opts.Config.Intervals.Default,
	})

	// Subscribe uptime tracker to runner
	runner.Subscribe(uptimeTracker)

	// Create notifier
	notifier := NewNotifier(opts.Config.Notifications, opts.Config.Environment)

	// Add notification channels
	if opts.SlackWebhook != "" {
		channel := opts.SlackChannel
		if channel == "" {
			channel = "#alerts"
		}
		notifier.AddChannel(NewSlackChannel(opts.SlackWebhook, channel, "Health Check Bot"))
		log.Println("[Health] Added Slack notification channel")
	}

	if opts.PagerDutyKey != "" {
		notifier.AddChannel(NewPagerDutyChannel(opts.PagerDutyKey))
		log.Println("[Health] Added PagerDuty notification channel")
	}

	if opts.WebhookURL != "" {
		notifier.AddChannel(NewWebhookChannel(opts.WebhookURL, opts.WebhookHeaders))
		log.Println("[Health] Added webhook notification channel")
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
		Config: DefaultConfig(),
		DB:     db,
		Redis:  redisClient,
	})
	return checker, runner, uptimeTracker
}

// StartHealthSystem starts the health check system
func StartHealthSystem(runner *Runner) {
	runner.Start()
	log.Println("[Health] Health check system started")
}

// StopHealthSystem stops the health check system
func StopHealthSystem(runner *Runner) {
	runner.Stop()
	log.Println("[Health] Health check system stopped")
}
