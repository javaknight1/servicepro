package database

import (
	"context"
	"database/sql"
	"time"

	"github.com/javaknight1/servicepro/backend/pkg/clients/logging"
	"github.com/javaknight1/servicepro/backend/pkg/clients/metrics"
)

// KPIMetricsCollector periodically queries PostgreSQL KPI views
// and publishes them as Prometheus gauges.
type KPIMetricsCollector struct {
	db       *sql.DB
	interval time.Duration

	totalTenants  metrics.Gauge
	activeTenants metrics.Gauge
	totalUsers    metrics.Gauge
	totalJobs     metrics.Gauge
	totalQuotes   metrics.Gauge
	totalInvoices metrics.Gauge

	dailyRevenue metrics.Gauge

	arOutstanding metrics.Gauge
}

// NewKPIMetricsCollector creates a new collector that reads KPI views
// from PostgreSQL and publishes them via the metrics client.
func NewKPIMetricsCollector(db *sql.DB, mc metrics.Client, interval time.Duration) *KPIMetricsCollector {
	return &KPIMetricsCollector{
		db:       db,
		interval: interval,
		totalTenants: mc.Gauge(metrics.GaugeOptions{
			Name: "servicepro_tenants_total",
			Help: "Total number of tenants",
		}),
		activeTenants: mc.Gauge(metrics.GaugeOptions{
			Name: "servicepro_tenants_active",
			Help: "Number of active tenants",
		}),
		totalUsers: mc.Gauge(metrics.GaugeOptions{
			Name: "servicepro_users_total",
			Help: "Total number of users",
		}),
		totalJobs: mc.Gauge(metrics.GaugeOptions{
			Name: "servicepro_jobs_total",
			Help: "Total number of jobs",
		}),
		totalQuotes: mc.Gauge(metrics.GaugeOptions{
			Name: "servicepro_quotes_total",
			Help: "Total number of quotes",
		}),
		totalInvoices: mc.Gauge(metrics.GaugeOptions{
			Name: "servicepro_invoices_total",
			Help: "Total number of invoices",
		}),
		dailyRevenue: mc.Gauge(metrics.GaugeOptions{
			Name: "servicepro_revenue_today",
			Help: "Total revenue received today",
		}),
		arOutstanding: mc.Gauge(metrics.GaugeOptions{
			Name:   "servicepro_ar_outstanding",
			Help:   "Outstanding accounts receivable by age bucket",
			Labels: []string{"age_bucket"},
		}),
	}
}

// Start begins the periodic collection loop. It stops when ctx is cancelled.
func (c *KPIMetricsCollector) Start(ctx context.Context) {
	go func() {
		ticker := time.NewTicker(c.interval)
		defer ticker.Stop()

		c.collect(ctx)

		for {
			select {
			case <-ctx.Done():
				logging.Info(ctx, "[KPI] KPI metrics collector stopped", nil)
				return
			case <-ticker.C:
				c.collect(ctx)
			}
		}
	}()
}

func (c *KPIMetricsCollector) collect(ctx context.Context) {
	c.collectOverview(ctx)
	c.collectRevenue(ctx)
	c.collectARAging(ctx)
}

func (c *KPIMetricsCollector) collectOverview(ctx context.Context) {
	var totalTenants, activeTenants, totalUsers, totalJobs, totalQuotes, totalInvoices int64
	err := c.db.QueryRowContext(ctx,
		"SELECT total_tenants, active_tenants, total_users, total_jobs, total_quotes, total_invoices FROM v_platform_overview",
	).Scan(&totalTenants, &activeTenants, &totalUsers, &totalJobs, &totalQuotes, &totalInvoices)
	if err != nil {
		logging.Warn(ctx, "[KPI] Failed to collect platform overview", map[string]any{"error": err.Error()})
		return
	}

	c.totalTenants.Set(float64(totalTenants))
	c.activeTenants.Set(float64(activeTenants))
	c.totalUsers.Set(float64(totalUsers))
	c.totalJobs.Set(float64(totalJobs))
	c.totalQuotes.Set(float64(totalQuotes))
	c.totalInvoices.Set(float64(totalInvoices))
}

func (c *KPIMetricsCollector) collectRevenue(ctx context.Context) {
	var revenue float64
	err := c.db.QueryRowContext(ctx,
		"SELECT COALESCE(total_revenue, 0) FROM v_revenue_daily WHERE day = CURRENT_DATE",
	).Scan(&revenue)
	if err != nil {
		if err == sql.ErrNoRows {
			c.dailyRevenue.Set(0)
			return
		}
		logging.Warn(ctx, "[KPI] Failed to collect daily revenue", map[string]any{"error": err.Error()})
		return
	}

	c.dailyRevenue.Set(revenue)
}

func (c *KPIMetricsCollector) collectARAging(ctx context.Context) {
	rows, err := c.db.QueryContext(ctx, "SELECT age_bucket, outstanding_amount FROM v_ar_aging_summary")
	if err != nil {
		logging.Warn(ctx, "[KPI] Failed to collect A/R aging", map[string]any{"error": err.Error()})
		return
	}
	defer rows.Close()

	for rows.Next() {
		var bucket string
		var amount float64
		if err := rows.Scan(&bucket, &amount); err != nil {
			logging.Warn(ctx, "[KPI] Failed to scan A/R aging row", map[string]any{"error": err.Error()})
			continue
		}
		c.arOutstanding.WithLabels(map[string]string{"age_bucket": bucket}).Set(amount)
	}
}
