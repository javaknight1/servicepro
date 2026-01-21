package analytics

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/google/uuid"
)

// AggregatorConfig holds configuration for the analytics aggregator
type AggregatorConfig struct {
	// HourlyInterval is how often to run hourly aggregation
	HourlyInterval time.Duration

	// DailyInterval is how often to run daily aggregation
	DailyInterval time.Duration

	// WeeklyInterval is how often to run weekly aggregation
	WeeklyInterval time.Duration

	// RetentionDays is how long to keep raw events
	RetentionDays int

	// HourlyRetentionDays is how long to keep hourly metrics
	HourlyRetentionDays int

	// DailyRetentionDays is how long to keep daily metrics
	DailyRetentionDays int

	// Workers is the number of aggregation workers
	Workers int

	// Debug enables debug logging
	Debug bool
}

// DefaultAggregatorConfig returns default aggregator configuration
func DefaultAggregatorConfig() AggregatorConfig {
	return AggregatorConfig{
		HourlyInterval:      time.Hour,
		DailyInterval:       24 * time.Hour,
		WeeklyInterval:      7 * 24 * time.Hour,
		RetentionDays:       30,
		HourlyRetentionDays: 90,
		DailyRetentionDays:  365,
		Workers:             2,
		Debug:               false,
	}
}

// Aggregator handles analytics data aggregation
type Aggregator struct {
	config       AggregatorConfig
	db           *sql.DB
	eventStore   EventStore
	metricsStore MetricsStore
	ctx          context.Context
	cancel       context.CancelFunc
	wg           sync.WaitGroup
}

// NewAggregator creates a new analytics aggregator
func NewAggregator(db *sql.DB, eventStore EventStore, metricsStore MetricsStore, config ...AggregatorConfig) *Aggregator {
	cfg := DefaultAggregatorConfig()
	if len(config) > 0 {
		cfg = config[0]
	}

	ctx, cancel := context.WithCancel(context.Background())

	return &Aggregator{
		config:       cfg,
		db:           db,
		eventStore:   eventStore,
		metricsStore: metricsStore,
		ctx:          ctx,
		cancel:       cancel,
	}
}

// Start starts the aggregation jobs
func (a *Aggregator) Start() {
	// Start hourly aggregation job
	a.wg.Add(1)
	go a.runJob("hourly", a.config.HourlyInterval, a.aggregateHourly)

	// Start daily aggregation job
	a.wg.Add(1)
	go a.runJob("daily", a.config.DailyInterval, a.aggregateDaily)

	// Start weekly aggregation job
	a.wg.Add(1)
	go a.runJob("weekly", a.config.WeeklyInterval, a.aggregateWeekly)

	// Start cleanup job
	a.wg.Add(1)
	go a.runJob("cleanup", 24*time.Hour, a.cleanup)

	if a.config.Debug {
		log.Printf("[Analytics Aggregator] Started with hourly=%s, daily=%s, weekly=%s",
			a.config.HourlyInterval, a.config.DailyInterval, a.config.WeeklyInterval)
	}
}

// Stop stops the aggregation jobs
func (a *Aggregator) Stop() error {
	a.cancel()

	done := make(chan struct{})
	go func() {
		a.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		return nil
	case <-time.After(30 * time.Second):
		return context.DeadlineExceeded
	}
}

// runJob runs an aggregation job on a schedule
func (a *Aggregator) runJob(name string, interval time.Duration, fn func(context.Context) error) {
	defer a.wg.Done()

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-a.ctx.Done():
			return
		case <-ticker.C:
			start := time.Now()
			if err := fn(a.ctx); err != nil {
				log.Printf("[Analytics Aggregator] %s job failed: %v", name, err)
			} else if a.config.Debug {
				log.Printf("[Analytics Aggregator] %s job completed in %s", name, time.Since(start))
			}
		}
	}
}

// aggregateHourly performs hourly aggregation
func (a *Aggregator) aggregateHourly(ctx context.Context) error {
	now := time.Now().UTC()
	periodEnd := now.Truncate(time.Hour)
	periodStart := periodEnd.Add(-time.Hour)

	return a.aggregateForPeriod(ctx, periodStart, periodEnd, "hourly")
}

// aggregateDaily performs daily aggregation
func (a *Aggregator) aggregateDaily(ctx context.Context) error {
	now := time.Now().UTC()
	periodEnd := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	periodStart := periodEnd.AddDate(0, 0, -1)

	return a.aggregateForPeriod(ctx, periodStart, periodEnd, "daily")
}

// aggregateWeekly performs weekly aggregation
func (a *Aggregator) aggregateWeekly(ctx context.Context) error {
	now := time.Now().UTC()
	// Get start of current week (Monday)
	weekday := int(now.Weekday())
	if weekday == 0 {
		weekday = 7
	}
	periodEnd := time.Date(now.Year(), now.Month(), now.Day()-weekday+1, 0, 0, 0, 0, time.UTC)
	periodStart := periodEnd.AddDate(0, 0, -7)

	return a.aggregateForPeriod(ctx, periodStart, periodEnd, "weekly")
}

// aggregateForPeriod aggregates events for a specific time period
func (a *Aggregator) aggregateForPeriod(ctx context.Context, periodStart, periodEnd time.Time, granularity string) error {
	// Aggregate event counts by name
	if err := a.aggregateEventCounts(ctx, periodStart, periodEnd, granularity); err != nil {
		return fmt.Errorf("failed to aggregate event counts: %w", err)
	}

	// Aggregate API latencies
	if err := a.aggregateAPILatencies(ctx, periodStart, periodEnd, granularity); err != nil {
		return fmt.Errorf("failed to aggregate API latencies: %w", err)
	}

	// Aggregate user activity
	if err := a.aggregateUserActivity(ctx, periodStart, periodEnd, granularity); err != nil {
		return fmt.Errorf("failed to aggregate user activity: %w", err)
	}

	// Aggregate business metrics
	if err := a.aggregateBusinessMetrics(ctx, periodStart, periodEnd, granularity); err != nil {
		return fmt.Errorf("failed to aggregate business metrics: %w", err)
	}

	return nil
}

// aggregateEventCounts aggregates event counts by name
func (a *Aggregator) aggregateEventCounts(ctx context.Context, periodStart, periodEnd time.Time, granularity string) error {
	query := `
		SELECT name, category, tenant_id, COUNT(*) as count
		FROM analytics.events
		WHERE timestamp >= $1 AND timestamp < $2
		GROUP BY name, category, tenant_id
	`

	rows, err := a.db.QueryContext(ctx, query, periodStart, periodEnd)
	if err != nil {
		return err
	}
	defer rows.Close()

	var metrics []*AggregatedMetric
	for rows.Next() {
		var name, category string
		var tenantID *uuid.UUID
		var count int64

		if err := rows.Scan(&name, &category, &tenantID, &count); err != nil {
			return err
		}

		metrics = append(metrics, &AggregatedMetric{
			ID:          uuid.New(),
			MetricName:  "event_count",
			Dimension:   "event_name",
			DimValue:    name,
			TenantID:    tenantID,
			Value:       float64(count),
			Count:       count,
			PeriodStart: periodStart,
			PeriodEnd:   periodEnd,
			Granularity: granularity,
			Metadata: map[string]interface{}{
				"category": category,
			},
			CreatedAt: time.Now().UTC(),
		})
	}

	if len(metrics) > 0 {
		return a.metricsStore.SaveMetrics(ctx, metrics)
	}
	return nil
}

// aggregateAPILatencies aggregates API latency metrics
func (a *Aggregator) aggregateAPILatencies(ctx context.Context, periodStart, periodEnd time.Time, granularity string) error {
	query := `
		SELECT
			properties->>'path' as path,
			properties->>'method' as method,
			tenant_id,
			COUNT(*) as request_count,
			AVG((properties->>'duration_ms')::float) as avg_duration,
			MAX((properties->>'duration_ms')::float) as max_duration,
			MIN((properties->>'duration_ms')::float) as min_duration,
			PERCENTILE_CONT(0.95) WITHIN GROUP (ORDER BY (properties->>'duration_ms')::float) as p95_duration,
			PERCENTILE_CONT(0.99) WITHIN GROUP (ORDER BY (properties->>'duration_ms')::float) as p99_duration
		FROM analytics.events
		WHERE name = 'api_latency'
		  AND timestamp >= $1 AND timestamp < $2
		  AND properties->>'path' IS NOT NULL
		GROUP BY properties->>'path', properties->>'method', tenant_id
	`

	rows, err := a.db.QueryContext(ctx, query, periodStart, periodEnd)
	if err != nil {
		return err
	}
	defer rows.Close()

	var metrics []*AggregatedMetric
	for rows.Next() {
		var path, method string
		var tenantID *uuid.UUID
		var requestCount int64
		var avgDuration, maxDuration, minDuration, p95Duration, p99Duration float64

		if err := rows.Scan(&path, &method, &tenantID, &requestCount, &avgDuration, &maxDuration, &minDuration, &p95Duration, &p99Duration); err != nil {
			return err
		}

		// Create metric for average latency
		metrics = append(metrics, &AggregatedMetric{
			ID:          uuid.New(),
			MetricName:  "api_latency_avg",
			Dimension:   "endpoint",
			DimValue:    method + " " + path,
			TenantID:    tenantID,
			Value:       avgDuration,
			Count:       requestCount,
			PeriodStart: periodStart,
			PeriodEnd:   periodEnd,
			Granularity: granularity,
			Metadata: map[string]interface{}{
				"method":       method,
				"path":         path,
				"max_duration": maxDuration,
				"min_duration": minDuration,
				"p95_duration": p95Duration,
				"p99_duration": p99Duration,
			},
			CreatedAt: time.Now().UTC(),
		})

		// Create metric for request count
		metrics = append(metrics, &AggregatedMetric{
			ID:          uuid.New(),
			MetricName:  "api_request_count",
			Dimension:   "endpoint",
			DimValue:    method + " " + path,
			TenantID:    tenantID,
			Value:       float64(requestCount),
			Count:       requestCount,
			PeriodStart: periodStart,
			PeriodEnd:   periodEnd,
			Granularity: granularity,
			Metadata: map[string]interface{}{
				"method": method,
				"path":   path,
			},
			CreatedAt: time.Now().UTC(),
		})
	}

	if len(metrics) > 0 {
		return a.metricsStore.SaveMetrics(ctx, metrics)
	}
	return nil
}

// aggregateUserActivity aggregates user activity metrics
func (a *Aggregator) aggregateUserActivity(ctx context.Context, periodStart, periodEnd time.Time, granularity string) error {
	// Count unique users
	query := `
		SELECT tenant_id, COUNT(DISTINCT user_id) as unique_users
		FROM analytics.events
		WHERE timestamp >= $1 AND timestamp < $2
		  AND user_id IS NOT NULL
		GROUP BY tenant_id
	`

	rows, err := a.db.QueryContext(ctx, query, periodStart, periodEnd)
	if err != nil {
		return err
	}
	defer rows.Close()

	var metrics []*AggregatedMetric
	for rows.Next() {
		var tenantID *uuid.UUID
		var uniqueUsers int64

		if err := rows.Scan(&tenantID, &uniqueUsers); err != nil {
			return err
		}

		metrics = append(metrics, &AggregatedMetric{
			ID:          uuid.New(),
			MetricName:  "unique_users",
			Dimension:   "total",
			DimValue:    "all",
			TenantID:    tenantID,
			Value:       float64(uniqueUsers),
			Count:       uniqueUsers,
			PeriodStart: periodStart,
			PeriodEnd:   periodEnd,
			Granularity: granularity,
			CreatedAt:   time.Now().UTC(),
		})
	}

	// Count unique sessions
	sessionsQuery := `
		SELECT tenant_id, COUNT(DISTINCT session_id) as unique_sessions
		FROM analytics.events
		WHERE timestamp >= $1 AND timestamp < $2
		  AND session_id IS NOT NULL AND session_id != ''
		GROUP BY tenant_id
	`

	sessionsRows, err := a.db.QueryContext(ctx, sessionsQuery, periodStart, periodEnd)
	if err != nil {
		return err
	}
	defer sessionsRows.Close()

	for sessionsRows.Next() {
		var tenantID *uuid.UUID
		var uniqueSessions int64

		if err := sessionsRows.Scan(&tenantID, &uniqueSessions); err != nil {
			return err
		}

		metrics = append(metrics, &AggregatedMetric{
			ID:          uuid.New(),
			MetricName:  "unique_sessions",
			Dimension:   "total",
			DimValue:    "all",
			TenantID:    tenantID,
			Value:       float64(uniqueSessions),
			Count:       uniqueSessions,
			PeriodStart: periodStart,
			PeriodEnd:   periodEnd,
			Granularity: granularity,
			CreatedAt:   time.Now().UTC(),
		})
	}

	if len(metrics) > 0 {
		return a.metricsStore.SaveMetrics(ctx, metrics)
	}
	return nil
}

// aggregateBusinessMetrics aggregates business-specific metrics
func (a *Aggregator) aggregateBusinessMetrics(ctx context.Context, periodStart, periodEnd time.Time, granularity string) error {
	// Aggregate by business event type
	query := `
		SELECT
			name,
			tenant_id,
			COUNT(*) as count,
			SUM((properties->>'value')::float) as total_value,
			AVG((properties->>'value')::float) as avg_value
		FROM analytics.events
		WHERE category = 'business'
		  AND timestamp >= $1 AND timestamp < $2
		GROUP BY name, tenant_id
	`

	rows, err := a.db.QueryContext(ctx, query, periodStart, periodEnd)
	if err != nil {
		return err
	}
	defer rows.Close()

	var metrics []*AggregatedMetric
	for rows.Next() {
		var name string
		var tenantID *uuid.UUID
		var count int64
		var totalValue, avgValue sql.NullFloat64

		if err := rows.Scan(&name, &tenantID, &count, &totalValue, &avgValue); err != nil {
			return err
		}

		// Count metric
		metrics = append(metrics, &AggregatedMetric{
			ID:          uuid.New(),
			MetricName:  "business_event_count",
			Dimension:   "event_type",
			DimValue:    name,
			TenantID:    tenantID,
			Value:       float64(count),
			Count:       count,
			PeriodStart: periodStart,
			PeriodEnd:   periodEnd,
			Granularity: granularity,
			CreatedAt:   time.Now().UTC(),
		})

		// Total value metric (if applicable)
		if totalValue.Valid {
			metrics = append(metrics, &AggregatedMetric{
				ID:          uuid.New(),
				MetricName:  "business_event_value",
				Dimension:   "event_type",
				DimValue:    name,
				TenantID:    tenantID,
				Value:       totalValue.Float64,
				Count:       count,
				PeriodStart: periodStart,
				PeriodEnd:   periodEnd,
				Granularity: granularity,
				Metadata: map[string]interface{}{
					"avg_value": avgValue.Float64,
				},
				CreatedAt: time.Now().UTC(),
			})
		}
	}

	if len(metrics) > 0 {
		return a.metricsStore.SaveMetrics(ctx, metrics)
	}
	return nil
}

// cleanup removes old data based on retention policies
func (a *Aggregator) cleanup(ctx context.Context) error {
	now := time.Now().UTC()

	// Delete old raw events
	eventsBefore := now.AddDate(0, 0, -a.config.RetentionDays)
	eventsDeleted, err := a.eventStore.Delete(ctx, eventsBefore)
	if err != nil {
		return fmt.Errorf("failed to delete old events: %w", err)
	}
	if a.config.Debug && eventsDeleted > 0 {
		log.Printf("[Analytics Aggregator] Deleted %d events older than %s", eventsDeleted, eventsBefore)
	}

	// Delete old hourly metrics
	hourlyBefore := now.AddDate(0, 0, -a.config.HourlyRetentionDays)
	hourlyDeleted, err := a.metricsStore.DeleteOldMetrics(ctx, hourlyBefore, "hourly")
	if err != nil {
		return fmt.Errorf("failed to delete old hourly metrics: %w", err)
	}
	if a.config.Debug && hourlyDeleted > 0 {
		log.Printf("[Analytics Aggregator] Deleted %d hourly metrics older than %s", hourlyDeleted, hourlyBefore)
	}

	// Delete old daily metrics
	dailyBefore := now.AddDate(0, 0, -a.config.DailyRetentionDays)
	dailyDeleted, err := a.metricsStore.DeleteOldMetrics(ctx, dailyBefore, "daily")
	if err != nil {
		return fmt.Errorf("failed to delete old daily metrics: %w", err)
	}
	if a.config.Debug && dailyDeleted > 0 {
		log.Printf("[Analytics Aggregator] Deleted %d daily metrics older than %s", dailyDeleted, dailyBefore)
	}

	return nil
}

// RunAggregationNow runs all aggregation jobs immediately
func (a *Aggregator) RunAggregationNow(ctx context.Context) error {
	if err := a.aggregateHourly(ctx); err != nil {
		return fmt.Errorf("hourly aggregation failed: %w", err)
	}
	if err := a.aggregateDaily(ctx); err != nil {
		return fmt.Errorf("daily aggregation failed: %w", err)
	}
	if err := a.aggregateWeekly(ctx); err != nil {
		return fmt.Errorf("weekly aggregation failed: %w", err)
	}
	return nil
}

// RunCleanupNow runs the cleanup job immediately
func (a *Aggregator) RunCleanupNow(ctx context.Context) error {
	return a.cleanup(ctx)
}
