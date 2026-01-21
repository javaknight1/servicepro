package monitoring

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch/types"
)

// CloudWatchConfig holds CloudWatch configuration
type CloudWatchConfig struct {
	// Namespace is the CloudWatch namespace for metrics
	Namespace string

	// Region is the AWS region
	Region string

	// FlushInterval is how often to send metrics to CloudWatch
	FlushInterval time.Duration

	// BatchSize is the maximum number of metrics per API call
	BatchSize int

	// Enabled enables/disables CloudWatch integration
	Enabled bool

	// DefaultDimensions are added to all metrics
	DefaultDimensions map[string]string

	// Environment tag
	Environment string

	// Service name tag
	ServiceName string
}

// DefaultCloudWatchConfig returns default CloudWatch configuration
func DefaultCloudWatchConfig() CloudWatchConfig {
	return CloudWatchConfig{
		Namespace:     "ServicePro",
		FlushInterval: 60 * time.Second,
		BatchSize:     20, // CloudWatch limit is 1000, but smaller batches are safer
		Enabled:       true,
		Environment:   "development",
		ServiceName:   "servicepro-api",
	}
}

// CloudWatchExporter exports metrics to AWS CloudWatch
type CloudWatchExporter struct {
	client     *cloudwatch.Client
	config     CloudWatchConfig
	registry   *MetricsRegistry
	ctx        context.Context
	cancel     context.CancelFunc
	wg         sync.WaitGroup
	mu         sync.Mutex
	buffer     []types.MetricDatum
	dimensions []types.Dimension
}

// NewCloudWatchExporter creates a new CloudWatch exporter
func NewCloudWatchExporter(registry *MetricsRegistry, cfg CloudWatchConfig) (*CloudWatchExporter, error) {
	if !cfg.Enabled {
		return &CloudWatchExporter{config: cfg}, nil
	}

	// Load AWS config
	awsCfg, err := config.LoadDefaultConfig(context.Background(),
		config.WithRegion(cfg.Region),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	client := cloudwatch.NewFromConfig(awsCfg)

	ctx, cancel := context.WithCancel(context.Background())

	// Build default dimensions
	dimensions := []types.Dimension{
		{
			Name:  aws.String("Environment"),
			Value: aws.String(cfg.Environment),
		},
		{
			Name:  aws.String("Service"),
			Value: aws.String(cfg.ServiceName),
		},
	}

	for name, value := range cfg.DefaultDimensions {
		dimensions = append(dimensions, types.Dimension{
			Name:  aws.String(name),
			Value: aws.String(value),
		})
	}

	return &CloudWatchExporter{
		client:     client,
		config:     cfg,
		registry:   registry,
		ctx:        ctx,
		cancel:     cancel,
		buffer:     make([]types.MetricDatum, 0, cfg.BatchSize),
		dimensions: dimensions,
	}, nil
}

// Start starts the CloudWatch exporter
func (e *CloudWatchExporter) Start() {
	if !e.config.Enabled {
		return
	}

	e.wg.Add(1)
	go e.run()

	log.Printf("[CloudWatch] Exporter started, namespace: %s, interval: %s",
		e.config.Namespace, e.config.FlushInterval)
}

// Stop stops the CloudWatch exporter
func (e *CloudWatchExporter) Stop() {
	if !e.config.Enabled {
		return
	}

	e.cancel()
	e.wg.Wait()

	// Final flush
	e.flush()

	log.Println("[CloudWatch] Exporter stopped")
}

func (e *CloudWatchExporter) run() {
	defer e.wg.Done()

	ticker := time.NewTicker(e.config.FlushInterval)
	defer ticker.Stop()

	for {
		select {
		case <-e.ctx.Done():
			return
		case <-ticker.C:
			e.collectAndFlush()
		}
	}
}

func (e *CloudWatchExporter) collectAndFlush() {
	// Get all metrics from registry
	metrics := e.registry.GetAll()

	e.mu.Lock()
	for _, metric := range metrics {
		datum := e.metricToDatum(metric)
		e.buffer = append(e.buffer, datum)

		// Flush if buffer is full
		if len(e.buffer) >= e.config.BatchSize {
			e.flushLocked()
		}
	}
	e.mu.Unlock()

	// Flush remaining
	e.flush()
}

func (e *CloudWatchExporter) metricToDatum(metric Metric) types.MetricDatum {
	dimensions := make([]types.Dimension, len(e.dimensions))
	copy(dimensions, e.dimensions)

	// Add metric-specific labels as dimensions
	for name, value := range metric.Labels {
		dimensions = append(dimensions, types.Dimension{
			Name:  aws.String(name),
			Value: aws.String(value),
		})
	}

	datum := types.MetricDatum{
		MetricName: aws.String(metric.Name),
		Value:      aws.Float64(metric.Value),
		Timestamp:  aws.Time(metric.Timestamp),
		Dimensions: dimensions,
	}

	// Set unit based on metric name
	datum.Unit = e.getUnit(metric.Name)

	return datum
}

func (e *CloudWatchExporter) getUnit(metricName string) types.StandardUnit {
	switch {
	case contains(metricName, "duration") || contains(metricName, "latency"):
		return types.StandardUnitMilliseconds
	case contains(metricName, "bytes") || contains(metricName, "size"):
		return types.StandardUnitBytes
	case contains(metricName, "percent") || contains(metricName, "rate"):
		return types.StandardUnitPercent
	case contains(metricName, "count") || contains(metricName, "total"):
		return types.StandardUnitCount
	default:
		return types.StandardUnitNone
	}
}

func (e *CloudWatchExporter) flush() {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.flushLocked()
}

func (e *CloudWatchExporter) flushLocked() {
	if len(e.buffer) == 0 {
		return
	}

	// CloudWatch allows max 1000 metrics per request, but we batch smaller
	for i := 0; i < len(e.buffer); i += e.config.BatchSize {
		end := i + e.config.BatchSize
		if end > len(e.buffer) {
			end = len(e.buffer)
		}

		batch := e.buffer[i:end]

		_, err := e.client.PutMetricData(context.Background(), &cloudwatch.PutMetricDataInput{
			Namespace:  aws.String(e.config.Namespace),
			MetricData: batch,
		})

		if err != nil {
			log.Printf("[CloudWatch] Failed to put metrics: %v", err)
		}
	}

	e.buffer = e.buffer[:0]
}

// OnMetrics implements MetricsSubscriber interface
func (e *CloudWatchExporter) OnMetrics(metrics []Metric) {
	if !e.config.Enabled {
		return
	}

	e.mu.Lock()
	defer e.mu.Unlock()

	for _, metric := range metrics {
		datum := e.metricToDatum(metric)
		e.buffer = append(e.buffer, datum)
	}
}

// PutMetric sends a single metric to CloudWatch immediately
func (e *CloudWatchExporter) PutMetric(metric Metric) error {
	if !e.config.Enabled {
		return nil
	}

	datum := e.metricToDatum(metric)

	_, err := e.client.PutMetricData(context.Background(), &cloudwatch.PutMetricDataInput{
		Namespace:  aws.String(e.config.Namespace),
		MetricData: []types.MetricDatum{datum},
	})

	return err
}

// PutAlarm creates or updates a CloudWatch alarm
func (e *CloudWatchExporter) PutAlarm(alarm AlarmConfig) error {
	if !e.config.Enabled {
		return nil
	}

	input := &cloudwatch.PutMetricAlarmInput{
		AlarmName:          aws.String(alarm.Name),
		AlarmDescription:   aws.String(alarm.Description),
		MetricName:         aws.String(alarm.MetricName),
		Namespace:          aws.String(e.config.Namespace),
		Statistic:          alarm.Statistic,
		Period:             aws.Int32(int32(alarm.Period.Seconds())),
		EvaluationPeriods:  aws.Int32(int32(alarm.EvaluationPeriods)),
		Threshold:          aws.Float64(alarm.Threshold),
		ComparisonOperator: alarm.ComparisonOperator,
		Dimensions:         e.dimensions,
		TreatMissingData:   aws.String("notBreaching"),
	}

	if len(alarm.AlarmActions) > 0 {
		input.AlarmActions = alarm.AlarmActions
	}
	if len(alarm.OKActions) > 0 {
		input.OKActions = alarm.OKActions
	}

	_, err := e.client.PutMetricAlarm(context.Background(), input)
	return err
}

// AlarmConfig holds configuration for a CloudWatch alarm
type AlarmConfig struct {
	Name               string
	Description        string
	MetricName         string
	Statistic          types.Statistic
	Period             time.Duration
	EvaluationPeriods  int
	Threshold          float64
	ComparisonOperator types.ComparisonOperator
	AlarmActions       []string
	OKActions          []string
}

// Helper function
func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// StandardAlarms returns a set of standard monitoring alarms
func StandardAlarms(snsTopicARN string) []AlarmConfig {
	return []AlarmConfig{
		{
			Name:               "HighErrorRate",
			Description:        "HTTP error rate is above threshold",
			MetricName:         "http_errors_total",
			Statistic:          types.StatisticSum,
			Period:             5 * time.Minute,
			EvaluationPeriods:  2,
			Threshold:          100,
			ComparisonOperator: types.ComparisonOperatorGreaterThanThreshold,
			AlarmActions:       []string{snsTopicARN},
		},
		{
			Name:               "HighLatency",
			Description:        "API latency is above threshold",
			MetricName:         "http_request_duration_ms",
			Statistic:          types.StatisticAverage,
			Period:             5 * time.Minute,
			EvaluationPeriods:  3,
			Threshold:          1000, // 1 second
			ComparisonOperator: types.ComparisonOperatorGreaterThanThreshold,
			AlarmActions:       []string{snsTopicARN},
		},
		{
			Name:               "HighMemoryUsage",
			Description:        "Memory usage is above threshold",
			MetricName:         "system_memory_usage_percent",
			Statistic:          types.StatisticAverage,
			Period:             5 * time.Minute,
			EvaluationPeriods:  3,
			Threshold:          85,
			ComparisonOperator: types.ComparisonOperatorGreaterThanThreshold,
			AlarmActions:       []string{snsTopicARN},
		},
		{
			Name:               "DatabaseConnectionPoolExhausted",
			Description:        "Database connection pool is nearly exhausted",
			MetricName:         "db_connections_in_use",
			Statistic:          types.StatisticMaximum,
			Period:             1 * time.Minute,
			EvaluationPeriods:  2,
			Threshold:          45, // Assuming max 50 connections
			ComparisonOperator: types.ComparisonOperatorGreaterThanThreshold,
			AlarmActions:       []string{snsTopicARN},
		},
		{
			Name:               "LowCacheHitRate",
			Description:        "Cache hit rate is below threshold",
			MetricName:         "redis_cache_hit_rate",
			Statistic:          types.StatisticAverage,
			Period:             15 * time.Minute,
			EvaluationPeriods:  2,
			Threshold:          70,
			ComparisonOperator: types.ComparisonOperatorLessThanThreshold,
			AlarmActions:       []string{snsTopicARN},
		},
		{
			Name:               "SlowDatabaseQueries",
			Description:        "Too many slow database queries",
			MetricName:         "db_slow_queries_total",
			Statistic:          types.StatisticSum,
			Period:             5 * time.Minute,
			EvaluationPeriods:  2,
			Threshold:          50,
			ComparisonOperator: types.ComparisonOperatorGreaterThanThreshold,
			AlarmActions:       []string{snsTopicARN},
		},
	}
}
