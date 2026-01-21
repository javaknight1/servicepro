package monitoring

import (
	"sync"
	"testing"
	"time"
)

type mockSubscriber struct {
	mu       sync.Mutex
	received [][]Metric
}

func (m *mockSubscriber) OnMetrics(metrics []Metric) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.received = append(m.received, metrics)
}

func (m *mockSubscriber) callCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.received)
}

func TestCollector(t *testing.T) {
	t.Run("Creates collector with defaults", func(t *testing.T) {
		r := NewMetricsRegistry()
		c := NewCollector(r, CollectorConfig{})

		if c.interval != 15*time.Second {
			t.Errorf("Expected default interval 15s, got %v", c.interval)
		}
	})

	t.Run("Collector notifies subscribers", func(t *testing.T) {
		r := NewMetricsRegistry()
		r.Counter("test", "Test", nil).Inc()

		sub := &mockSubscriber{}

		c := NewCollector(r, CollectorConfig{
			Interval: 50 * time.Millisecond,
		})
		c.Subscribe(sub)

		c.Start()
		time.Sleep(150 * time.Millisecond)
		c.Stop()

		// Should have received at least 2 calls (initial + at least 1 interval)
		if sub.callCount() < 2 {
			t.Errorf("Expected at least 2 subscriber calls, got %d", sub.callCount())
		}
	})

	t.Run("Multiple subscribers receive updates", func(t *testing.T) {
		r := NewMetricsRegistry()
		r.Counter("test", "Test", nil).Inc()

		sub1 := &mockSubscriber{}
		sub2 := &mockSubscriber{}

		c := NewCollector(r, CollectorConfig{
			Interval: 50 * time.Millisecond,
		})
		c.Subscribe(sub1)
		c.Subscribe(sub2)

		c.Start()
		time.Sleep(100 * time.Millisecond)
		c.Stop()

		if sub1.callCount() == 0 {
			t.Error("Subscriber 1 should have received updates")
		}
		if sub2.callCount() == 0 {
			t.Error("Subscriber 2 should have received updates")
		}
	})

	t.Run("Stop waits for goroutine", func(t *testing.T) {
		r := NewMetricsRegistry()
		c := NewCollector(r, CollectorConfig{
			Interval: 1 * time.Second,
		})

		c.Start()

		done := make(chan bool)
		go func() {
			c.Stop()
			done <- true
		}()

		select {
		case <-done:
			// Success
		case <-time.After(2 * time.Second):
			t.Error("Stop should complete quickly")
		}
	})
}

func TestAPIMetrics(t *testing.T) {
	t.Run("Records request metrics", func(t *testing.T) {
		r := NewMetricsRegistry()
		m := NewAPIMetrics(r)

		m.RecordRequest("GET", "/api/users", 200, 50*time.Millisecond, 1024)
		m.RecordRequest("POST", "/api/users", 201, 100*time.Millisecond, 512)
		m.RecordRequest("GET", "/api/users/1", 404, 20*time.Millisecond, 64)

		if m.RequestsTotal.Value() != 3 {
			t.Errorf("Expected 3 requests, got %d", m.RequestsTotal.Value())
		}

		if m.ErrorsTotal.Value() != 1 {
			t.Errorf("Expected 1 error, got %d", m.ErrorsTotal.Value())
		}
	})

	t.Run("Records response size", func(t *testing.T) {
		r := NewMetricsRegistry()
		m := NewAPIMetrics(r)

		m.RecordRequest("GET", "/api/data", 200, 10*time.Millisecond, 10000)

		if m.ResponseSize.Count() != 1 {
			t.Errorf("Expected 1 observation, got %d", m.ResponseSize.Count())
		}
	})

	t.Run("Tracks 4xx and 5xx as errors", func(t *testing.T) {
		r := NewMetricsRegistry()
		m := NewAPIMetrics(r)

		m.RecordRequest("GET", "/api/users", 200, 10*time.Millisecond, 100) // OK
		m.RecordRequest("GET", "/api/users", 400, 10*time.Millisecond, 50)  // Error
		m.RecordRequest("GET", "/api/users", 404, 10*time.Millisecond, 50)  // Error
		m.RecordRequest("GET", "/api/users", 500, 10*time.Millisecond, 50)  // Error
		m.RecordRequest("GET", "/api/users", 503, 10*time.Millisecond, 50)  // Error

		if m.ErrorsTotal.Value() != 4 {
			t.Errorf("Expected 4 errors, got %d", m.ErrorsTotal.Value())
		}
	})
}

func TestDatabaseMetrics(t *testing.T) {
	t.Run("Records query metrics", func(t *testing.T) {
		r := NewMetricsRegistry()
		m := NewDatabaseMetrics(r, 100*time.Millisecond)

		m.RecordQuery("SELECT", "users", 50*time.Millisecond, nil)
		m.RecordQuery("INSERT", "users", 30*time.Millisecond, nil)
		m.RecordQuery("SELECT", "orders", 200*time.Millisecond, nil) // Slow

		if m.QueriesTotal.Value() != 3 {
			t.Errorf("Expected 3 queries, got %d", m.QueriesTotal.Value())
		}

		if m.SlowQueries.Value() != 1 {
			t.Errorf("Expected 1 slow query, got %d", m.SlowQueries.Value())
		}
	})

	t.Run("Records query errors", func(t *testing.T) {
		r := NewMetricsRegistry()
		m := NewDatabaseMetrics(r, 100*time.Millisecond)

		m.RecordQuery("SELECT", "users", 50*time.Millisecond, nil)
		m.RecordQuery("INSERT", "users", 30*time.Millisecond, errMock)

		if m.QueryErrors.Value() != 1 {
			t.Errorf("Expected 1 query error, got %d", m.QueryErrors.Value())
		}
	})

	t.Run("Uses default slow threshold", func(t *testing.T) {
		r := NewMetricsRegistry()
		m := NewDatabaseMetrics(r, 0) // Should default to 100ms

		m.RecordQuery("SELECT", "users", 50*time.Millisecond, nil)
		m.RecordQuery("SELECT", "users", 150*time.Millisecond, nil)

		if m.SlowQueries.Value() != 1 {
			t.Errorf("Expected 1 slow query, got %d", m.SlowQueries.Value())
		}
	})
}

func TestCacheMetrics(t *testing.T) {
	t.Run("Records cache operations", func(t *testing.T) {
		r := NewMetricsRegistry()
		m := NewCacheMetrics(r)

		m.RecordHit("user:1")
		m.RecordHit("user:2")
		m.RecordMiss("user:3")
		m.RecordSet("user:3")
		m.RecordDelete("user:1")

		if m.HitsTotal.Value() != 2 {
			t.Errorf("Expected 2 hits, got %d", m.HitsTotal.Value())
		}
		if m.MissesTotal.Value() != 1 {
			t.Errorf("Expected 1 miss, got %d", m.MissesTotal.Value())
		}
		if m.SetsTotal.Value() != 1 {
			t.Errorf("Expected 1 set, got %d", m.SetsTotal.Value())
		}
		if m.DeletesTotal.Value() != 1 {
			t.Errorf("Expected 1 delete, got %d", m.DeletesTotal.Value())
		}
	})

	t.Run("Calculates hit rate", func(t *testing.T) {
		r := NewMetricsRegistry()
		m := NewCacheMetrics(r)

		// 80% hit rate
		for i := 0; i < 80; i++ {
			m.RecordHit("key")
		}
		for i := 0; i < 20; i++ {
			m.RecordMiss("key")
		}

		rate := m.HitRate()
		if rate != 80 {
			t.Errorf("Expected hit rate 80%%, got %.2f%%", rate)
		}
	})

	t.Run("Hit rate is 0 with no operations", func(t *testing.T) {
		r := NewMetricsRegistry()
		m := NewCacheMetrics(r)

		if m.HitRate() != 0 {
			t.Errorf("Expected hit rate 0, got %.2f", m.HitRate())
		}
	})

	t.Run("Records errors", func(t *testing.T) {
		r := NewMetricsRegistry()
		m := NewCacheMetrics(r)

		m.RecordError("get")
		m.RecordError("set")

		if m.Errors.Value() != 2 {
			t.Errorf("Expected 2 errors, got %d", m.Errors.Value())
		}
	})
}

func TestStatusCodeGroup(t *testing.T) {
	tests := []struct {
		code     int
		expected string
	}{
		{200, "2xx"},
		{201, "2xx"},
		{204, "2xx"},
		{301, "3xx"},
		{304, "3xx"},
		{400, "4xx"},
		{404, "4xx"},
		{500, "5xx"},
		{503, "5xx"},
		{100, "unknown"},
	}

	for _, tt := range tests {
		t.Run(string(rune(tt.code)), func(t *testing.T) {
			result := statusCodeGroup(tt.code)
			if result != tt.expected {
				t.Errorf("statusCodeGroup(%d) = %s, want %s", tt.code, result, tt.expected)
			}
		})
	}
}

// Mock error for testing
type mockError struct{}

func (e mockError) Error() string { return "mock error" }

var errMock = mockError{}

func BenchmarkAPIMetrics(b *testing.B) {
	r := NewMetricsRegistry()
	m := NewAPIMetrics(r)
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		m.RecordRequest("GET", "/api/users", 200, 50*time.Millisecond, 1024)
	}
}

func BenchmarkDatabaseMetrics(b *testing.B) {
	r := NewMetricsRegistry()
	m := NewDatabaseMetrics(r, 100*time.Millisecond)
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		m.RecordQuery("SELECT", "users", 50*time.Millisecond, nil)
	}
}

func BenchmarkCacheMetrics(b *testing.B) {
	r := NewMetricsRegistry()
	m := NewCacheMetrics(r)
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		m.RecordHit("user:1")
	}
}
