package monitoring

import (
	"testing"
	"time"
)

func TestCloudWatchConfig(t *testing.T) {
	t.Run("DefaultCloudWatchConfig returns sensible defaults", func(t *testing.T) {
		cfg := DefaultCloudWatchConfig()

		if cfg.Namespace != "ServicePro" {
			t.Errorf("Expected namespace ServicePro, got %s", cfg.Namespace)
		}
		if cfg.FlushInterval != 60*time.Second {
			t.Errorf("Expected flush interval 60s, got %v", cfg.FlushInterval)
		}
		if cfg.BatchSize != 20 {
			t.Errorf("Expected batch size 20, got %d", cfg.BatchSize)
		}
		if !cfg.Enabled {
			t.Error("Expected enabled to be true by default")
		}
		if cfg.Environment != "development" {
			t.Errorf("Expected environment development, got %s", cfg.Environment)
		}
		if cfg.ServiceName != "servicepro-api" {
			t.Errorf("Expected service name servicepro-api, got %s", cfg.ServiceName)
		}
	})
}

func TestCloudWatchExporter(t *testing.T) {
	t.Run("Creates disabled exporter when not enabled", func(t *testing.T) {
		registry := NewMetricsRegistry()
		cfg := CloudWatchConfig{
			Enabled: false,
		}

		exporter, err := NewCloudWatchExporter(registry, cfg)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		if exporter == nil {
			t.Fatal("Expected exporter to be created")
		}

		// Should not panic when calling methods on disabled exporter
		exporter.Start()
		exporter.Stop()

		err = exporter.PutMetric(Metric{Name: "test"})
		if err != nil {
			t.Errorf("PutMetric on disabled exporter should return nil error")
		}

		err = exporter.PutAlarm(AlarmConfig{Name: "test"})
		if err != nil {
			t.Errorf("PutAlarm on disabled exporter should return nil error")
		}
	})

	t.Run("OnMetrics handles disabled exporter", func(t *testing.T) {
		cfg := CloudWatchConfig{
			Enabled: false,
		}

		exporter, _ := NewCloudWatchExporter(nil, cfg)
		exporter.OnMetrics([]Metric{{Name: "test"}})
		// Should not panic
	})
}

func TestStandardAlarms(t *testing.T) {
	t.Run("Returns standard alarm configurations", func(t *testing.T) {
		snsArn := "arn:aws:sns:us-east-1:123456789:alerts"
		alarms := StandardAlarms(snsArn)

		if len(alarms) == 0 {
			t.Fatal("Expected standard alarms to be defined")
		}

		// Check that all alarms have required fields
		for _, alarm := range alarms {
			if alarm.Name == "" {
				t.Error("Alarm name should not be empty")
			}
			if alarm.MetricName == "" {
				t.Errorf("Alarm %s: metric name should not be empty", alarm.Name)
			}
			if alarm.Period == 0 {
				t.Errorf("Alarm %s: period should not be zero", alarm.Name)
			}
			if alarm.EvaluationPeriods == 0 {
				t.Errorf("Alarm %s: evaluation periods should not be zero", alarm.Name)
			}
			if len(alarm.AlarmActions) == 0 {
				t.Errorf("Alarm %s: should have alarm actions", alarm.Name)
			}
		}
	})

	t.Run("Includes expected alarms", func(t *testing.T) {
		snsArn := "arn:aws:sns:us-east-1:123456789:alerts"
		alarms := StandardAlarms(snsArn)

		expectedAlarms := []string{
			"HighErrorRate",
			"HighLatency",
			"HighMemoryUsage",
			"DatabaseConnectionPoolExhausted",
			"LowCacheHitRate",
			"SlowDatabaseQueries",
		}

		alarmNames := make(map[string]bool)
		for _, a := range alarms {
			alarmNames[a.Name] = true
		}

		for _, expected := range expectedAlarms {
			if !alarmNames[expected] {
				t.Errorf("Expected alarm %s not found", expected)
			}
		}
	})

	t.Run("Uses provided SNS ARN", func(t *testing.T) {
		snsArn := "arn:aws:sns:us-west-2:999999999:my-topic"
		alarms := StandardAlarms(snsArn)

		for _, alarm := range alarms {
			if len(alarm.AlarmActions) > 0 && alarm.AlarmActions[0] != snsArn {
				t.Errorf("Alarm %s: expected SNS ARN %s, got %s", alarm.Name, snsArn, alarm.AlarmActions[0])
			}
		}
	})
}

func TestContains(t *testing.T) {
	tests := []struct {
		s        string
		substr   string
		expected bool
	}{
		{"request_duration_ms", "duration", true},
		{"response_latency", "latency", true},
		{"request_size_bytes", "bytes", true},
		{"cpu_percent", "percent", true},
		{"request_count", "count", true},
		{"http_requests_total", "total", true},
		{"http_requests", "total", false},
		{"", "test", false},
		{"test", "", true},
		{"abc", "abcd", false},
	}

	for _, tt := range tests {
		result := contains(tt.s, tt.substr)
		if result != tt.expected {
			t.Errorf("contains(%q, %q) = %v, want %v", tt.s, tt.substr, result, tt.expected)
		}
	}
}

func TestAlarmConfig(t *testing.T) {
	t.Run("AlarmConfig fields", func(t *testing.T) {
		alarm := AlarmConfig{
			Name:              "TestAlarm",
			Description:       "Test alarm description",
			MetricName:        "test_metric",
			Period:            5 * time.Minute,
			EvaluationPeriods: 2,
			Threshold:         100,
			AlarmActions:      []string{"arn:aws:sns:us-east-1:123456789:test"},
			OKActions:         []string{"arn:aws:sns:us-east-1:123456789:test"},
		}

		if alarm.Name != "TestAlarm" {
			t.Errorf("Expected name TestAlarm, got %s", alarm.Name)
		}
		if alarm.Period != 5*time.Minute {
			t.Errorf("Expected period 5m, got %v", alarm.Period)
		}
		if alarm.Threshold != 100 {
			t.Errorf("Expected threshold 100, got %f", alarm.Threshold)
		}
	})
}

func BenchmarkContains(b *testing.B) {
	s := "http_request_duration_milliseconds"
	substr := "duration"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		contains(s, substr)
	}
}
