package monitoring

import (
	"sync"
	"testing"
	"time"
)

func TestCounter(t *testing.T) {
	t.Run("New counter starts at zero", func(t *testing.T) {
		c := NewCounter("test_counter", "Test counter", nil)
		if c.Value() != 0 {
			t.Errorf("Expected 0, got %d", c.Value())
		}
	})

	t.Run("Inc increments by 1", func(t *testing.T) {
		c := NewCounter("test_counter", "Test counter", nil)
		c.Inc()
		c.Inc()
		c.Inc()
		if c.Value() != 3 {
			t.Errorf("Expected 3, got %d", c.Value())
		}
	})

	t.Run("Add increments by value", func(t *testing.T) {
		c := NewCounter("test_counter", "Test counter", nil)
		c.Add(10)
		c.Add(5)
		if c.Value() != 15 {
			t.Errorf("Expected 15, got %d", c.Value())
		}
	})

	t.Run("Concurrent increments are safe", func(t *testing.T) {
		c := NewCounter("test_counter", "Test counter", nil)
		var wg sync.WaitGroup

		for i := 0; i < 100; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				c.Inc()
			}()
		}

		wg.Wait()
		if c.Value() != 100 {
			t.Errorf("Expected 100, got %d", c.Value())
		}
	})

	t.Run("Metric returns correct representation", func(t *testing.T) {
		labels := map[string]string{"method": "GET", "path": "/api"}
		c := NewCounter("http_requests", "HTTP requests", labels)
		c.Inc()

		metric := c.Metric()
		if metric.Name != "http_requests" {
			t.Errorf("Expected name 'http_requests', got '%s'", metric.Name)
		}
		if metric.Type != MetricTypeCounter {
			t.Errorf("Expected type counter, got %s", metric.Type)
		}
		if metric.Value != 1 {
			t.Errorf("Expected value 1, got %f", metric.Value)
		}
		if metric.Labels["method"] != "GET" {
			t.Errorf("Expected label method=GET, got %s", metric.Labels["method"])
		}
	})
}

func TestGauge(t *testing.T) {
	t.Run("Set and Get", func(t *testing.T) {
		g := NewGauge("test_gauge", "Test gauge", nil)
		g.Set(42.5)
		if v := g.Value(); v != 42.5 {
			t.Errorf("Expected 42.5, got %f", v)
		}
	})

	t.Run("Inc and Dec", func(t *testing.T) {
		g := NewGauge("test_gauge", "Test gauge", nil)
		g.Inc()
		g.Inc()
		g.Dec()
		if v := g.Value(); v != 1 {
			t.Errorf("Expected 1, got %f", v)
		}
	})

	t.Run("Add", func(t *testing.T) {
		g := NewGauge("test_gauge", "Test gauge", nil)
		g.Add(10.5)
		g.Add(-3.5)
		if v := g.Value(); v != 7 {
			t.Errorf("Expected 7, got %f", v)
		}
	})

	t.Run("Concurrent operations are safe", func(t *testing.T) {
		g := NewGauge("test_gauge", "Test gauge", nil)
		var wg sync.WaitGroup

		for i := 0; i < 50; i++ {
			wg.Add(2)
			go func() {
				defer wg.Done()
				g.Inc()
			}()
			go func() {
				defer wg.Done()
				g.Dec()
			}()
		}

		wg.Wait()
		// After equal incs and decs, should be 0
		if v := g.Value(); v != 0 {
			t.Errorf("Expected 0, got %f", v)
		}
	})
}

func TestHistogram(t *testing.T) {
	t.Run("Observe records values", func(t *testing.T) {
		h := NewHistogram("request_duration", "Request duration", nil, []float64{10, 50, 100, 500})
		h.Observe(5)
		h.Observe(25)
		h.Observe(75)
		h.Observe(200)
		h.Observe(1000)

		if h.Count() != 5 {
			t.Errorf("Expected count 5, got %d", h.Count())
		}
	})

	t.Run("Sum is correct", func(t *testing.T) {
		h := NewHistogram("request_duration", "Request duration", nil, DefaultBuckets())
		h.Observe(10)
		h.Observe(20)
		h.Observe(30)

		if s := h.Sum(); s != 60 {
			t.Errorf("Expected sum 60, got %f", s)
		}
	})

	t.Run("Mean is correct", func(t *testing.T) {
		h := NewHistogram("request_duration", "Request duration", nil, DefaultBuckets())
		h.Observe(10)
		h.Observe(20)
		h.Observe(30)

		if m := h.Mean(); m != 20 {
			t.Errorf("Expected mean 20, got %f", m)
		}
	})

	t.Run("Mean returns 0 for empty histogram", func(t *testing.T) {
		h := NewHistogram("request_duration", "Request duration", nil, DefaultBuckets())
		if m := h.Mean(); m != 0 {
			t.Errorf("Expected mean 0, got %f", m)
		}
	})

	t.Run("Percentile approximation", func(t *testing.T) {
		h := NewHistogram("request_duration", "Request duration", nil, []float64{10, 50, 100, 500, 1000})

		// Add values that will fall into different buckets
		for i := 0; i < 50; i++ {
			h.Observe(5) // <= 10
		}
		for i := 0; i < 30; i++ {
			h.Observe(25) // <= 50
		}
		for i := 0; i < 20; i++ {
			h.Observe(75) // <= 100
		}

		// p50 should be in the first bucket (10)
		p50 := h.Percentile(50)
		if p50 != 10 {
			t.Errorf("Expected p50 = 10, got %f", p50)
		}

		// p90 should be higher
		p90 := h.Percentile(90)
		if p90 < 50 {
			t.Errorf("Expected p90 >= 50, got %f", p90)
		}
	})
}

func TestSummary(t *testing.T) {
	t.Run("Observe and Mean", func(t *testing.T) {
		s := NewSummary("request_duration", "Request duration", nil, 100)
		s.Observe(10)
		s.Observe(20)
		s.Observe(30)

		if m := s.Mean(); m != 20 {
			t.Errorf("Expected mean 20, got %f", m)
		}
	})

	t.Run("Sliding window behavior", func(t *testing.T) {
		s := NewSummary("request_duration", "Request duration", nil, 3) // max 3 values
		s.Observe(10)
		s.Observe(20)
		s.Observe(30)
		s.Observe(100) // This should push out 10

		// Mean should be (20 + 30 + 100) / 3 = 50
		if m := s.Mean(); m != 50 {
			t.Errorf("Expected mean 50, got %f", m)
		}
	})

	t.Run("Count tracks all observations", func(t *testing.T) {
		s := NewSummary("request_duration", "Request duration", nil, 2)
		s.Observe(1)
		s.Observe(2)
		s.Observe(3)
		s.Observe(4)

		// Count should track all observations
		if c := s.Count(); c != 4 {
			t.Errorf("Expected count 4, got %d", c)
		}
	})
}

func TestMetricsRegistry(t *testing.T) {
	t.Run("Gets or creates counter", func(t *testing.T) {
		r := NewMetricsRegistry()
		c1 := r.Counter("test", "Test", nil)
		c1.Inc()

		c2 := r.Counter("test", "Test", nil)
		if c1 != c2 {
			t.Error("Expected same counter instance")
		}
		if c2.Value() != 1 {
			t.Errorf("Expected value 1, got %d", c2.Value())
		}
	})

	t.Run("Different labels create different counters", func(t *testing.T) {
		r := NewMetricsRegistry()
		c1 := r.Counter("test", "Test", map[string]string{"a": "1"})
		c2 := r.Counter("test", "Test", map[string]string{"a": "2"})

		c1.Inc()
		if c2.Value() != 0 {
			t.Error("Expected separate counter instances for different labels")
		}
	})

	t.Run("Gets or creates gauge", func(t *testing.T) {
		r := NewMetricsRegistry()
		g1 := r.Gauge("test", "Test", nil)
		g1.Set(42)

		g2 := r.Gauge("test", "Test", nil)
		if g1 != g2 {
			t.Error("Expected same gauge instance")
		}
	})

	t.Run("Gets or creates histogram", func(t *testing.T) {
		r := NewMetricsRegistry()
		h1 := r.Histogram("test", "Test", nil, nil)
		h1.Observe(10)

		h2 := r.Histogram("test", "Test", nil, nil)
		if h1 != h2 {
			t.Error("Expected same histogram instance")
		}
	})

	t.Run("GetAll returns all metrics", func(t *testing.T) {
		r := NewMetricsRegistry()
		r.Counter("counter1", "Counter 1", nil).Inc()
		r.Gauge("gauge1", "Gauge 1", nil).Set(10)
		r.Histogram("hist1", "Histogram 1", nil, nil).Observe(5)
		r.Summary("summary1", "Summary 1", nil, 100).Observe(20)

		metrics := r.GetAll()
		if len(metrics) != 4 {
			t.Errorf("Expected 4 metrics, got %d", len(metrics))
		}

		// Verify metric types
		typeCount := make(map[MetricType]int)
		for _, m := range metrics {
			typeCount[m.Type]++
		}

		if typeCount[MetricTypeCounter] != 1 {
			t.Error("Expected 1 counter")
		}
		if typeCount[MetricTypeGauge] != 1 {
			t.Error("Expected 1 gauge")
		}
		if typeCount[MetricTypeHistogram] != 1 {
			t.Error("Expected 1 histogram")
		}
		if typeCount[MetricTypeSummary] != 1 {
			t.Error("Expected 1 summary")
		}
	})

	t.Run("Concurrent access is safe", func(t *testing.T) {
		r := NewMetricsRegistry()
		var wg sync.WaitGroup

		for i := 0; i < 100; i++ {
			wg.Add(1)
			go func(n int) {
				defer wg.Done()
				r.Counter("counter", "Test", nil).Inc()
				r.Gauge("gauge", "Test", nil).Inc()
				r.Histogram("hist", "Test", nil, nil).Observe(float64(n))
			}(i)
		}

		wg.Wait()

		c := r.Counter("counter", "Test", nil)
		if c.Value() != 100 {
			t.Errorf("Expected counter value 100, got %d", c.Value())
		}
	})
}

func TestDefaultRegistry(t *testing.T) {
	t.Run("DefaultRegistry returns same instance", func(t *testing.T) {
		r1 := DefaultRegistry()
		r2 := DefaultRegistry()
		if r1 != r2 {
			t.Error("Expected same registry instance")
		}
	})

	t.Run("SetDefaultRegistry changes instance", func(t *testing.T) {
		original := DefaultRegistry()
		newRegistry := NewMetricsRegistry()

		SetDefaultRegistry(newRegistry)
		if DefaultRegistry() != newRegistry {
			t.Error("Expected new registry")
		}

		// Restore original
		SetDefaultRegistry(original)
	})
}

func TestRuntimeMetrics(t *testing.T) {
	t.Run("Collects runtime metrics", func(t *testing.T) {
		r := NewMetricsRegistry()
		rm := NewRuntimeMetrics(r, 100*time.Millisecond)

		rm.Start()
		time.Sleep(200 * time.Millisecond)
		rm.Stop()

		metrics := r.GetAll()
		if len(metrics) == 0 {
			t.Error("Expected runtime metrics to be collected")
		}

		// Check for expected metrics
		metricNames := make(map[string]bool)
		for _, m := range metrics {
			metricNames[m.Name] = true
		}

		expectedMetrics := []string{
			"go_goroutines",
			"go_heap_alloc_bytes",
			"go_heap_sys_bytes",
			"go_heap_objects",
			"go_gc_pause_total_ns",
			"go_gc_cycles",
			"go_cpu_count",
		}

		for _, name := range expectedMetrics {
			if !metricNames[name] {
				t.Errorf("Expected metric %s not found", name)
			}
		}
	})
}

func BenchmarkCounter(b *testing.B) {
	c := NewCounter("benchmark_counter", "Benchmark counter", nil)
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		c.Inc()
	}
}

func BenchmarkCounterParallel(b *testing.B) {
	c := NewCounter("benchmark_counter", "Benchmark counter", nil)
	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			c.Inc()
		}
	})
}

func BenchmarkHistogramObserve(b *testing.B) {
	h := NewHistogram("benchmark_histogram", "Benchmark histogram", nil, DefaultBuckets())
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		h.Observe(float64(i % 1000))
	}
}

func BenchmarkRegistryCounter(b *testing.B) {
	r := NewMetricsRegistry()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		r.Counter("test", "Test", nil).Inc()
	}
}
