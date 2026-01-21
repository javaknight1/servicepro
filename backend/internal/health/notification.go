package health

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"
)

// NotificationConfig holds notification configuration
type NotificationConfig struct {
	// Enabled enables/disables notifications
	Enabled bool

	// MinInterval is the minimum time between notifications for the same check
	MinInterval time.Duration

	// Channels is the list of notification channels
	Channels []NotificationChannel

	// OnlyOnStatusChange only sends notifications when status changes
	OnlyOnStatusChange bool

	// IncludeRecovery sends notifications when status recovers to healthy
	IncludeRecovery bool

	// AlertThreshold is how many consecutive failures before alerting
	AlertThreshold int
}

// DefaultNotificationConfig returns default notification config
func DefaultNotificationConfig() NotificationConfig {
	return NotificationConfig{
		Enabled:            true,
		MinInterval:        5 * time.Minute,
		OnlyOnStatusChange: true,
		IncludeRecovery:    true,
		AlertThreshold:     1,
	}
}

// NotificationChannel is an interface for sending notifications
type NotificationChannel interface {
	Send(ctx context.Context, notification Notification) error
	Name() string
}

// Notification represents a health notification
type Notification struct {
	Title       string                 `json:"title"`
	Message     string                 `json:"message"`
	Severity    Severity               `json:"severity"`
	CheckName   string                 `json:"check_name"`
	Status      Status                 `json:"status"`
	OldStatus   Status                 `json:"old_status,omitempty"`
	Timestamp   time.Time              `json:"timestamp"`
	Environment string                 `json:"environment"`
	Details     map[string]interface{} `json:"details,omitempty"`
}

// Severity represents notification severity
type Severity string

const (
	SeverityInfo     Severity = "info"
	SeverityWarning  Severity = "warning"
	SeverityCritical Severity = "critical"
	SeverityRecovery Severity = "recovery"
)

// Notifier manages health notifications
type Notifier struct {
	config          NotificationConfig
	mu              sync.RWMutex
	lastNotified    map[string]time.Time
	consecutiveFail map[string]int
	environment     string
}

// NewNotifier creates a new health notifier
func NewNotifier(config NotificationConfig, environment string) *Notifier {
	return &Notifier{
		config:          config,
		lastNotified:    make(map[string]time.Time),
		consecutiveFail: make(map[string]int),
		environment:     environment,
	}
}

// OnHealthCheck implements Subscriber interface
func (n *Notifier) OnHealthCheck(result CheckResult) {
	if !n.config.Enabled {
		return
	}

	n.mu.Lock()
	defer n.mu.Unlock()

	// Track consecutive failures
	if result.Status == StatusUnhealthy {
		n.consecutiveFail[result.Name]++
	} else {
		n.consecutiveFail[result.Name] = 0
	}
}

// OnStatusChange implements Subscriber interface
func (n *Notifier) OnStatusChange(name string, oldStatus, newStatus Status) {
	if !n.config.Enabled {
		return
	}

	// Check if we should send notification
	if !n.shouldNotify(name, oldStatus, newStatus) {
		return
	}

	// Build notification
	notification := n.buildNotification(name, oldStatus, newStatus)

	// Send to all channels
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	for _, channel := range n.config.Channels {
		go func(ch NotificationChannel) {
			if err := ch.Send(ctx, notification); err != nil {
				log.Printf("[Health] Failed to send notification via %s: %v", ch.Name(), err)
			}
		}(channel)
	}

	// Update last notified time
	n.mu.Lock()
	n.lastNotified[name] = time.Now()
	n.mu.Unlock()
}

// shouldNotify determines if a notification should be sent
func (n *Notifier) shouldNotify(name string, oldStatus, newStatus Status) bool {
	n.mu.RLock()
	defer n.mu.RUnlock()

	// Check minimum interval
	if lastTime, ok := n.lastNotified[name]; ok {
		if time.Since(lastTime) < n.config.MinInterval {
			return false
		}
	}

	// Check if only status changes should trigger notifications
	if n.config.OnlyOnStatusChange && oldStatus == newStatus {
		return false
	}

	// Check if recovery notifications are enabled
	if newStatus == StatusHealthy && !n.config.IncludeRecovery {
		return false
	}

	// Check alert threshold
	if newStatus == StatusUnhealthy {
		consecutiveFails := n.consecutiveFail[name]
		if consecutiveFails < n.config.AlertThreshold {
			return false
		}
	}

	return true
}

// buildNotification creates a notification from status change
func (n *Notifier) buildNotification(name string, oldStatus, newStatus Status) Notification {
	severity := n.getSeverity(oldStatus, newStatus)
	title := n.getTitle(name, newStatus, severity)
	message := n.getMessage(name, oldStatus, newStatus)

	return Notification{
		Title:       title,
		Message:     message,
		Severity:    severity,
		CheckName:   name,
		Status:      newStatus,
		OldStatus:   oldStatus,
		Timestamp:   time.Now(),
		Environment: n.environment,
	}
}

// getSeverity determines notification severity
func (n *Notifier) getSeverity(oldStatus, newStatus Status) Severity {
	if newStatus == StatusHealthy {
		return SeverityRecovery
	}
	if newStatus == StatusUnhealthy {
		return SeverityCritical
	}
	if newStatus == StatusDegraded {
		return SeverityWarning
	}
	return SeverityInfo
}

// getTitle creates notification title
func (n *Notifier) getTitle(name string, status Status, severity Severity) string {
	switch severity {
	case SeverityCritical:
		return fmt.Sprintf("🔴 CRITICAL: %s is unhealthy", name)
	case SeverityWarning:
		return fmt.Sprintf("🟡 WARNING: %s is degraded", name)
	case SeverityRecovery:
		return fmt.Sprintf("🟢 RECOVERED: %s is healthy", name)
	default:
		return fmt.Sprintf("ℹ️ INFO: %s status changed to %s", name, status)
	}
}

// getMessage creates notification message
func (n *Notifier) getMessage(name string, oldStatus, newStatus Status) string {
	return fmt.Sprintf(
		"Health check '%s' changed from %s to %s in %s environment",
		name, oldStatus, newStatus, n.environment,
	)
}

// AddChannel adds a notification channel
func (n *Notifier) AddChannel(channel NotificationChannel) {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.config.Channels = append(n.config.Channels, channel)
}

// Notification channels

// SlackChannel sends notifications to Slack
type SlackChannel struct {
	WebhookURL string
	Channel    string
	Username   string
}

// NewSlackChannel creates a new Slack notification channel
func NewSlackChannel(webhookURL, channel, username string) *SlackChannel {
	return &SlackChannel{
		WebhookURL: webhookURL,
		Channel:    channel,
		Username:   username,
	}
}

func (s *SlackChannel) Name() string {
	return "slack"
}

func (s *SlackChannel) Send(ctx context.Context, notification Notification) error {
	color := s.getColor(notification.Severity)

	payload := map[string]interface{}{
		"channel":  s.Channel,
		"username": s.Username,
		"attachments": []map[string]interface{}{
			{
				"color": color,
				"title": notification.Title,
				"text":  notification.Message,
				"fields": []map[string]interface{}{
					{"title": "Check", "value": notification.CheckName, "short": true},
					{"title": "Status", "value": string(notification.Status), "short": true},
					{"title": "Environment", "value": notification.Environment, "short": true},
					{"title": "Time", "value": notification.Timestamp.Format(time.RFC3339), "short": true},
				},
			},
		},
	}

	return s.sendWebhook(ctx, payload)
}

func (s *SlackChannel) getColor(severity Severity) string {
	switch severity {
	case SeverityCritical:
		return "danger"
	case SeverityWarning:
		return "warning"
	case SeverityRecovery:
		return "good"
	default:
		return "#439FE0"
	}
}

func (s *SlackChannel) sendWebhook(ctx context.Context, payload map[string]interface{}) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.WebhookURL, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("slack webhook returned status %d", resp.StatusCode)
	}

	return nil
}

// WebhookChannel sends notifications to a generic webhook
type WebhookChannel struct {
	URL     string
	Headers map[string]string
}

// NewWebhookChannel creates a new webhook notification channel
func NewWebhookChannel(url string, headers map[string]string) *WebhookChannel {
	return &WebhookChannel{
		URL:     url,
		Headers: headers,
	}
}

func (w *WebhookChannel) Name() string {
	return "webhook"
}

func (w *WebhookChannel) Send(ctx context.Context, notification Notification) error {
	body, err := json.Marshal(notification)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, w.URL, bytes.NewReader(body))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")
	for key, value := range w.Headers {
		req.Header.Set(key, value)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("webhook returned status %d", resp.StatusCode)
	}

	return nil
}

// LogChannel logs notifications to the standard logger
type LogChannel struct {
	Prefix string
}

// NewLogChannel creates a new log notification channel
func NewLogChannel(prefix string) *LogChannel {
	return &LogChannel{Prefix: prefix}
}

func (l *LogChannel) Name() string {
	return "log"
}

func (l *LogChannel) Send(ctx context.Context, notification Notification) error {
	log.Printf("%s [%s] %s: %s (check: %s, status: %s)",
		l.Prefix,
		notification.Severity,
		notification.Title,
		notification.Message,
		notification.CheckName,
		notification.Status,
	)
	return nil
}

// EmailChannel sends notifications via email (placeholder)
type EmailChannel struct {
	SMTPHost string
	SMTPPort int
	From     string
	To       []string
	Username string
	Password string
}

// NewEmailChannel creates a new email notification channel
func NewEmailChannel(host string, port int, from string, to []string) *EmailChannel {
	return &EmailChannel{
		SMTPHost: host,
		SMTPPort: port,
		From:     from,
		To:       to,
	}
}

func (e *EmailChannel) Name() string {
	return "email"
}

func (e *EmailChannel) Send(ctx context.Context, notification Notification) error {
	// Placeholder - actual implementation would use net/smtp
	log.Printf("[Email] Would send email to %v: %s", e.To, notification.Title)
	return nil
}

// PagerDutyChannel sends notifications to PagerDuty
type PagerDutyChannel struct {
	RoutingKey string
}

// NewPagerDutyChannel creates a new PagerDuty notification channel
func NewPagerDutyChannel(routingKey string) *PagerDutyChannel {
	return &PagerDutyChannel{RoutingKey: routingKey}
}

func (p *PagerDutyChannel) Name() string {
	return "pagerduty"
}

func (p *PagerDutyChannel) Send(ctx context.Context, notification Notification) error {
	eventAction := "trigger"
	if notification.Severity == SeverityRecovery {
		eventAction = "resolve"
	}

	payload := map[string]interface{}{
		"routing_key":  p.RoutingKey,
		"event_action": eventAction,
		"dedup_key":    notification.CheckName,
		"payload": map[string]interface{}{
			"summary":   notification.Title,
			"severity":  p.mapSeverity(notification.Severity),
			"source":    notification.Environment,
			"timestamp": notification.Timestamp.Format(time.RFC3339),
			"custom_details": map[string]interface{}{
				"check_name": notification.CheckName,
				"status":     notification.Status,
				"message":    notification.Message,
			},
		},
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		"https://events.pagerduty.com/v2/enqueue", bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("pagerduty returned status %d", resp.StatusCode)
	}

	return nil
}

func (p *PagerDutyChannel) mapSeverity(severity Severity) string {
	switch severity {
	case SeverityCritical:
		return "critical"
	case SeverityWarning:
		return "warning"
	default:
		return "info"
	}
}
