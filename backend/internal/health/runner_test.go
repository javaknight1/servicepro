package health

import (
	"context"
	"sync/atomic"
	"testing"
	"time"
)

func TestNewRunner(t *testing.T) {
	checker := NewChecker("1.0.0", "test")
	runner := NewRunner(checker)

	if runner == nil {
		t.Fatal("NewRunner returned nil")
	}

	if runner.IsRunning() {
		t.Error("Runner should not be running initially")
	}
}

func TestRunnerStartStop(t *testing.T) {
	checker := NewChecker("1.0.0", "test")

	checker.Register(CheckConfig{
		Name:     "test-check",
		Interval: 100 * time.Millisecond,
		Check: func(ctx context.Context) CheckResult {
			return CheckResult{Status: StatusHealthy}
		},
	})

	runner := NewRunner(checker, RunnerConfig{
		DefaultInterval: 100 * time.Millisecond,
	})

	runner.Start()

	if !runner.IsRunning() {
		t.Error("Runner should be running after Start")
	}

	// Let it run a few checks
	time.Sleep(250 * time.Millisecond)

	runner.Stop()

	if runner.IsRunning() {
		t.Error("Runner should not be running after Stop")
	}
}

func TestRunnerSubscriber(t *testing.T) {
	checker := NewChecker("1.0.0", "test")

	var checkCount int64

	checker.Register(CheckConfig{
		Name:     "test-check",
		Interval: 50 * time.Millisecond,
		Check: func(ctx context.Context) CheckResult {
			return CheckResult{Status: StatusHealthy}
		},
	})

	runner := NewRunner(checker, RunnerConfig{
		DefaultInterval: 50 * time.Millisecond,
	})

	// Create mock subscriber
	subscriber := &mockSubscriber{
		onCheck: func(result CheckResult) {
			atomic.AddInt64(&checkCount, 1)
		},
	}

	runner.Subscribe(subscriber)
	runner.Start()

	// Wait for checks
	time.Sleep(200 * time.Millisecond)

	runner.Stop()

	count := atomic.LoadInt64(&checkCount)
	if count < 2 {
		t.Errorf("Expected at least 2 check notifications, got %d", count)
	}
}

func TestRunnerMultipleChecks(t *testing.T) {
	checker := NewChecker("1.0.0", "test")

	check1Count := int64(0)
	check2Count := int64(0)

	checker.Register(CheckConfig{
		Name:     "check-1",
		Interval: 50 * time.Millisecond,
		Check: func(ctx context.Context) CheckResult {
			atomic.AddInt64(&check1Count, 1)
			return CheckResult{Status: StatusHealthy}
		},
	})

	checker.Register(CheckConfig{
		Name:     "check-2",
		Interval: 50 * time.Millisecond,
		Check: func(ctx context.Context) CheckResult {
			atomic.AddInt64(&check2Count, 1)
			return CheckResult{Status: StatusHealthy}
		},
	})

	runner := NewRunner(checker, RunnerConfig{
		DefaultInterval: 50 * time.Millisecond,
	})

	runner.Start()
	time.Sleep(200 * time.Millisecond)
	runner.Stop()

	c1 := atomic.LoadInt64(&check1Count)
	c2 := atomic.LoadInt64(&check2Count)

	if c1 < 2 {
		t.Errorf("Expected at least 2 runs for check-1, got %d", c1)
	}

	if c2 < 2 {
		t.Errorf("Expected at least 2 runs for check-2, got %d", c2)
	}
}

func TestUptimeTracker(t *testing.T) {
	tracker := NewUptimeTracker(100)

	if tracker == nil {
		t.Fatal("NewUptimeTracker returned nil")
	}

	// Test uptime
	uptime := tracker.GetUptime()
	if uptime < 0 {
		t.Error("Uptime should be non-negative")
	}
}

func TestUptimeTrackerOnHealthCheck(t *testing.T) {
	tracker := NewUptimeTracker(100)

	// Simulate health checks
	tracker.OnHealthCheck(CheckResult{
		Name:   "test-check",
		Status: StatusHealthy,
	})

	tracker.OnHealthCheck(CheckResult{
		Name:   "test-check",
		Status: StatusHealthy,
	})

	tracker.OnHealthCheck(CheckResult{
		Name:   "test-check",
		Status: StatusUnhealthy,
	})

	availability := tracker.GetAvailability("test-check")

	// 2 out of 3 healthy = 66.67%
	if availability < 66.0 || availability > 67.0 {
		t.Errorf("Availability = %.2f%%, expected ~66.67%%", availability)
	}
}

func TestUptimeTrackerGetHistory(t *testing.T) {
	tracker := NewUptimeTracker(10)

	// Add checks
	for i := 0; i < 5; i++ {
		tracker.OnHealthCheck(CheckResult{
			Name:      "test-check",
			Status:    StatusHealthy,
			Timestamp: time.Now(),
		})
	}

	history := tracker.GetHistory("test-check")

	if len(history) != 5 {
		t.Errorf("Expected 5 history entries, got %d", len(history))
	}
}

func TestUptimeTrackerHistoryLimit(t *testing.T) {
	tracker := NewUptimeTracker(5)

	// Add more than limit
	for i := 0; i < 10; i++ {
		tracker.OnHealthCheck(CheckResult{
			Name:      "test-check",
			Status:    StatusHealthy,
			Timestamp: time.Now(),
		})
	}

	history := tracker.GetHistory("test-check")

	if len(history) != 5 {
		t.Errorf("Expected 5 history entries (limited), got %d", len(history))
	}
}

func TestUptimeTrackerOnStatusChange(t *testing.T) {
	tracker := NewUptimeTracker(100)

	tracker.OnStatusChange("test-check", StatusHealthy, StatusUnhealthy)
	tracker.OnStatusChange("test-check", StatusUnhealthy, StatusHealthy)

	events := tracker.GetStatusChanges(10)

	if len(events) != 2 {
		t.Errorf("Expected 2 status change events, got %d", len(events))
	}

	if events[0].OldStatus != StatusHealthy {
		t.Errorf("First event old status = %s, want healthy", events[0].OldStatus)
	}

	if events[1].NewStatus != StatusHealthy {
		t.Errorf("Second event new status = %s, want healthy", events[1].NewStatus)
	}
}

func TestUptimeTrackerGetStats(t *testing.T) {
	tracker := NewUptimeTracker(100)

	// Add checks
	tracker.OnHealthCheck(CheckResult{Name: "check-1", Status: StatusHealthy})
	tracker.OnHealthCheck(CheckResult{Name: "check-1", Status: StatusHealthy})
	tracker.OnHealthCheck(CheckResult{Name: "check-2", Status: StatusUnhealthy})

	stats := tracker.GetStats()

	if stats.TotalChecks["check-1"] != 2 {
		t.Errorf("TotalChecks[check-1] = %d, want 2", stats.TotalChecks["check-1"])
	}

	if stats.FailedChecks["check-1"] != 0 {
		t.Errorf("FailedChecks[check-1] = %d, want 0", stats.FailedChecks["check-1"])
	}

	if stats.FailedChecks["check-2"] != 1 {
		t.Errorf("FailedChecks[check-2] = %d, want 1", stats.FailedChecks["check-2"])
	}

	if stats.Availability["check-1"] != 100.0 {
		t.Errorf("Availability[check-1] = %.2f%%, want 100%%", stats.Availability["check-1"])
	}

	if stats.Availability["check-2"] != 0.0 {
		t.Errorf("Availability[check-2] = %.2f%%, want 0%%", stats.Availability["check-2"])
	}
}

func TestUptimeTrackerAllAvailability(t *testing.T) {
	tracker := NewUptimeTracker(100)

	tracker.OnHealthCheck(CheckResult{Name: "check-1", Status: StatusHealthy})
	tracker.OnHealthCheck(CheckResult{Name: "check-2", Status: StatusHealthy})
	tracker.OnHealthCheck(CheckResult{Name: "check-2", Status: StatusUnhealthy})

	availability := tracker.GetAllAvailability()

	if availability["check-1"] != 100.0 {
		t.Errorf("check-1 availability = %.2f%%, want 100%%", availability["check-1"])
	}

	if availability["check-2"] != 50.0 {
		t.Errorf("check-2 availability = %.2f%%, want 50%%", availability["check-2"])
	}
}

// Mock subscriber for testing
type mockSubscriber struct {
	onCheck        func(CheckResult)
	onStatusChange func(string, Status, Status)
}

func (m *mockSubscriber) OnHealthCheck(result CheckResult) {
	if m.onCheck != nil {
		m.onCheck(result)
	}
}

func (m *mockSubscriber) OnStatusChange(name string, oldStatus, newStatus Status) {
	if m.onStatusChange != nil {
		m.onStatusChange(name, oldStatus, newStatus)
	}
}
