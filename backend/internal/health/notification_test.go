package health

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

func TestNewNotifier(t *testing.T) {
	config := DefaultNotificationConfig()
	notifier := NewNotifier(config, "test")

	if notifier == nil {
		t.Fatal("NewNotifier returned nil")
	}
}

func TestNotifierOnHealthCheck(t *testing.T) {
	config := DefaultNotificationConfig()
	notifier := NewNotifier(config, "test")

	// Should track consecutive failures
	notifier.OnHealthCheck(CheckResult{Name: "test-check", Status: StatusUnhealthy})
	notifier.OnHealthCheck(CheckResult{Name: "test-check", Status: StatusUnhealthy})

	notifier.mu.Lock()
	count := notifier.consecutiveFail["test-check"]
	notifier.mu.Unlock()

	if count != 2 {
		t.Errorf("consecutiveFail = %d, want 2", count)
	}

	// Reset on healthy
	notifier.OnHealthCheck(CheckResult{Name: "test-check", Status: StatusHealthy})

	notifier.mu.Lock()
	count = notifier.consecutiveFail["test-check"]
	notifier.mu.Unlock()

	if count != 0 {
		t.Errorf("consecutiveFail after healthy = %d, want 0", count)
	}
}

func TestNotifierDisabled(t *testing.T) {
	config := DefaultNotificationConfig()
	config.Enabled = false

	var called int64
	channel := &mockChannel{
		sendFn: func(ctx context.Context, n Notification) error {
			atomic.AddInt64(&called, 1)
			return nil
		},
	}

	notifier := NewNotifier(config, "test")
	notifier.AddChannel(channel)

	notifier.OnStatusChange("test-check", StatusHealthy, StatusUnhealthy)

	time.Sleep(50 * time.Millisecond)

	if atomic.LoadInt64(&called) != 0 {
		t.Error("Notification should not be sent when disabled")
	}
}

func TestNotifierMinInterval(t *testing.T) {
	config := DefaultNotificationConfig()
	config.Enabled = true
	config.MinInterval = 1 * time.Hour // Long interval
	config.OnlyOnStatusChange = true

	var callCount int64
	channel := &mockChannel{
		sendFn: func(ctx context.Context, n Notification) error {
			atomic.AddInt64(&callCount, 1)
			return nil
		},
	}

	notifier := NewNotifier(config, "test")
	notifier.AddChannel(channel)

	// First notification should be sent
	notifier.OnStatusChange("test-check", StatusHealthy, StatusUnhealthy)

	// Second notification should be blocked by min interval
	notifier.OnStatusChange("test-check", StatusUnhealthy, StatusHealthy)

	time.Sleep(50 * time.Millisecond)

	count := atomic.LoadInt64(&callCount)
	if count != 1 {
		t.Errorf("Expected 1 notification (blocked by interval), got %d", count)
	}
}

func TestNotifierRecoveryDisabled(t *testing.T) {
	config := DefaultNotificationConfig()
	config.Enabled = true
	config.IncludeRecovery = false
	config.MinInterval = 0

	var callCount int64
	channel := &mockChannel{
		sendFn: func(ctx context.Context, n Notification) error {
			atomic.AddInt64(&callCount, 1)
			return nil
		},
	}

	notifier := NewNotifier(config, "test")
	notifier.AddChannel(channel)

	// First record a health check failure to meet alert threshold
	notifier.OnHealthCheck(CheckResult{Name: "test-check", Status: StatusUnhealthy})

	// Failure should notify
	notifier.OnStatusChange("test-check", StatusHealthy, StatusUnhealthy)

	// Recovery should not notify
	notifier.mu.Lock()
	notifier.lastNotified = make(map[string]time.Time) // Reset interval tracking
	notifier.mu.Unlock()
	notifier.OnStatusChange("test-check", StatusUnhealthy, StatusHealthy)

	time.Sleep(50 * time.Millisecond)

	count := atomic.LoadInt64(&callCount)
	if count != 1 {
		t.Errorf("Expected 1 notification (recovery disabled), got %d", count)
	}
}

func TestNotifierAlertThreshold(t *testing.T) {
	config := DefaultNotificationConfig()
	config.Enabled = true
	config.AlertThreshold = 3
	config.MinInterval = 0
	config.OnlyOnStatusChange = false

	var callCount int64
	channel := &mockChannel{
		sendFn: func(ctx context.Context, n Notification) error {
			atomic.AddInt64(&callCount, 1)
			return nil
		},
	}

	notifier := NewNotifier(config, "test")
	notifier.AddChannel(channel)

	// First two failures should not alert
	notifier.OnHealthCheck(CheckResult{Name: "test-check", Status: StatusUnhealthy})
	notifier.OnStatusChange("test-check", StatusHealthy, StatusUnhealthy)

	notifier.OnHealthCheck(CheckResult{Name: "test-check", Status: StatusUnhealthy})
	notifier.mu.Lock()
	notifier.lastNotified = make(map[string]time.Time)
	notifier.mu.Unlock()
	notifier.OnStatusChange("test-check", StatusHealthy, StatusUnhealthy)

	// Third failure should alert
	notifier.OnHealthCheck(CheckResult{Name: "test-check", Status: StatusUnhealthy})
	notifier.mu.Lock()
	notifier.lastNotified = make(map[string]time.Time)
	notifier.mu.Unlock()
	notifier.OnStatusChange("test-check", StatusHealthy, StatusUnhealthy)

	time.Sleep(100 * time.Millisecond)

	count := atomic.LoadInt64(&callCount)
	if count != 1 {
		t.Errorf("Expected 1 notification (after threshold), got %d", count)
	}
}

func TestLogChannel(t *testing.T) {
	channel := NewLogChannel("[Test]")

	if channel.Name() != "log" {
		t.Errorf("Name() = %s, want 'log'", channel.Name())
	}

	err := channel.Send(context.Background(), Notification{
		Title:     "Test Title",
		Message:   "Test Message",
		Severity:  SeverityWarning,
		CheckName: "test-check",
	})

	if err != nil {
		t.Errorf("Send() error = %v, want nil", err)
	}
}

func TestWebhookChannel(t *testing.T) {
	var receivedNotification bool

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedNotification = true
		if r.Method != http.MethodPost {
			t.Errorf("Method = %s, want POST", r.Method)
		}
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("Content-Type = %s, want application/json", r.Header.Get("Content-Type"))
		}
		if r.Header.Get("X-Custom") != "header" {
			t.Errorf("X-Custom = %s, want 'header'", r.Header.Get("X-Custom"))
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	channel := NewWebhookChannel(server.URL, map[string]string{
		"X-Custom": "header",
	})

	if channel.Name() != "webhook" {
		t.Errorf("Name() = %s, want 'webhook'", channel.Name())
	}

	err := channel.Send(context.Background(), Notification{
		Title:     "Test",
		Message:   "Test message",
		Severity:  SeverityCritical,
		CheckName: "test-check",
	})

	if err != nil {
		t.Errorf("Send() error = %v, want nil", err)
	}

	if !receivedNotification {
		t.Error("Webhook should have received notification")
	}
}

func TestWebhookChannelError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	channel := NewWebhookChannel(server.URL, nil)

	err := channel.Send(context.Background(), Notification{
		Title: "Test",
	})

	if err == nil {
		t.Error("Send() should return error for 500 status")
	}
}

func TestSlackChannelGetColor(t *testing.T) {
	channel := NewSlackChannel("http://example.com", "#channel", "bot")

	tests := []struct {
		severity Severity
		want     string
	}{
		{SeverityCritical, "danger"},
		{SeverityWarning, "warning"},
		{SeverityRecovery, "good"},
		{SeverityInfo, "#439FE0"},
	}

	for _, tt := range tests {
		got := channel.getColor(tt.severity)
		if got != tt.want {
			t.Errorf("getColor(%s) = %s, want %s", tt.severity, got, tt.want)
		}
	}
}

func TestNotificationSeverity(t *testing.T) {
	config := DefaultNotificationConfig()
	notifier := NewNotifier(config, "test")

	tests := []struct {
		oldStatus Status
		newStatus Status
		want      Severity
	}{
		{StatusHealthy, StatusUnhealthy, SeverityCritical},
		{StatusHealthy, StatusDegraded, SeverityWarning},
		{StatusUnhealthy, StatusHealthy, SeverityRecovery},
		{StatusDegraded, StatusUnknown, SeverityInfo},
	}

	for _, tt := range tests {
		got := notifier.getSeverity(tt.oldStatus, tt.newStatus)
		if got != tt.want {
			t.Errorf("getSeverity(%s, %s) = %s, want %s",
				tt.oldStatus, tt.newStatus, got, tt.want)
		}
	}
}

func TestNotificationTitle(t *testing.T) {
	config := DefaultNotificationConfig()
	notifier := NewNotifier(config, "test")

	title := notifier.getTitle("database", StatusUnhealthy, SeverityCritical)
	if title == "" {
		t.Error("Title should not be empty")
	}

	title = notifier.getTitle("database", StatusHealthy, SeverityRecovery)
	if title == "" {
		t.Error("Recovery title should not be empty")
	}
}

func TestPagerDutyChannelMapSeverity(t *testing.T) {
	channel := NewPagerDutyChannel("test-key")

	tests := []struct {
		severity Severity
		want     string
	}{
		{SeverityCritical, "critical"},
		{SeverityWarning, "warning"},
		{SeverityInfo, "info"},
		{SeverityRecovery, "info"},
	}

	for _, tt := range tests {
		got := channel.mapSeverity(tt.severity)
		if got != tt.want {
			t.Errorf("mapSeverity(%s) = %s, want %s", tt.severity, got, tt.want)
		}
	}
}

func TestEmailChannel(t *testing.T) {
	channel := NewEmailChannel("smtp.example.com", 587, "from@example.com", []string{"to@example.com"})

	if channel.Name() != "email" {
		t.Errorf("Name() = %s, want 'email'", channel.Name())
	}

	// Email channel is a placeholder, just test it doesn't error
	err := channel.Send(context.Background(), Notification{
		Title: "Test",
	})

	if err != nil {
		t.Errorf("Send() error = %v, want nil (placeholder)", err)
	}
}

// Mock channel for testing
type mockChannel struct {
	name   string
	sendFn func(ctx context.Context, n Notification) error
}

func (m *mockChannel) Name() string {
	if m.name != "" {
		return m.name
	}
	return "mock"
}

func (m *mockChannel) Send(ctx context.Context, n Notification) error {
	if m.sendFn != nil {
		return m.sendFn(ctx, n)
	}
	return nil
}
