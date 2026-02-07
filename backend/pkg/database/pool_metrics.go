package database

import (
	"context"
	"database/sql"
	"time"

	"github.com/javaknight1/servicepro/backend/pkg/clients/logging"
	"github.com/javaknight1/servicepro/backend/pkg/clients/metrics"
)

// PoolMetricsCollector periodically collects database connection pool stats
// and publishes them as Prometheus gauges.
type PoolMetricsCollector struct {
	db       *sql.DB
	interval time.Duration

	openConns    metrics.Gauge
	idleConns    metrics.Gauge
	inUseConns   metrics.Gauge
	maxOpenConns metrics.Gauge
	waitCount    metrics.Gauge
	waitDuration metrics.Gauge
}

// NewPoolMetricsCollector creates a new collector that reads pool stats
// from the given *sql.DB and publishes them via the metrics client.
func NewPoolMetricsCollector(db *sql.DB, mc metrics.Client, interval time.Duration) *PoolMetricsCollector {
	return &PoolMetricsCollector{
		db:       db,
		interval: interval,
		openConns: mc.Gauge(metrics.GaugeOptions{
			Name: "db_connections_open",
			Help: "Number of open database connections",
		}),
		idleConns: mc.Gauge(metrics.GaugeOptions{
			Name: "db_connections_idle",
			Help: "Number of idle database connections",
		}),
		inUseConns: mc.Gauge(metrics.GaugeOptions{
			Name: "db_connections_in_use",
			Help: "Number of in-use database connections",
		}),
		maxOpenConns: mc.Gauge(metrics.GaugeOptions{
			Name: "db_connections_max_open",
			Help: "Maximum number of open database connections",
		}),
		waitCount: mc.Gauge(metrics.GaugeOptions{
			Name: "db_connections_wait_count_total",
			Help: "Total number of connections waited for",
		}),
		waitDuration: mc.Gauge(metrics.GaugeOptions{
			Name: "db_connections_wait_duration_seconds_total",
			Help: "Total time waited for database connections in seconds",
		}),
	}
}

// Start begins the periodic collection loop. It stops when ctx is cancelled.
func (c *PoolMetricsCollector) Start(ctx context.Context) {
	go func() {
		ticker := time.NewTicker(c.interval)
		defer ticker.Stop()

		// Collect once immediately
		c.collect()

		for {
			select {
			case <-ctx.Done():
				logging.Info(ctx, "[DB-POOL] Pool metrics collector stopped", nil)
				return
			case <-ticker.C:
				c.collect()
			}
		}
	}()
}

func (c *PoolMetricsCollector) collect() {
	stats := c.db.Stats()

	c.openConns.Set(float64(stats.OpenConnections))
	c.idleConns.Set(float64(stats.Idle))
	c.inUseConns.Set(float64(stats.InUse))
	c.maxOpenConns.Set(float64(stats.MaxOpenConnections))
	c.waitCount.Set(float64(stats.WaitCount))
	c.waitDuration.Set(stats.WaitDuration.Seconds())
}
