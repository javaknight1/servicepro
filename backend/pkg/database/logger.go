package database

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"sync"
	"time"

	"gorm.io/gorm/logger"

	"github.com/javaknight1/servicepro/backend/pkg/clients/logging"
	"github.com/javaknight1/servicepro/backend/pkg/clients/metrics"
)

// SQL parsing regexes compiled once at package level
var (
	insertRegex = regexp.MustCompile(`(?i)INSERT\s+INTO\s+"?(\w+)"?`)
	updateRegex = regexp.MustCompile(`(?i)UPDATE\s+"?(\w+)"?`)
	deleteRegex = regexp.MustCompile(`(?i)DELETE\s+FROM\s+"?(\w+)"?`)
	selectRegex = regexp.MustCompile(`(?i)FROM\s+"?(\w+)"?`)
)

const (
	maxSQLLength = 1024
	logPrefix    = "[DB-QUERY]"
)

// sqlInfo holds parsed SQL operation info
type sqlInfo struct {
	Operation string
	Table     string
}

// QueryLoggerConfig holds configuration for the query logger
type QueryLoggerConfig struct {
	SlowQueryThreshold time.Duration
	AlertThreshold     time.Duration
	LogAllQueries      bool
}

// QueryLogger implements gorm.io/gorm/logger.Interface with structured logging and metrics
type QueryLogger struct {
	config   QueryLoggerConfig
	logLevel logger.LogLevel

	mu            sync.RWMutex
	metricsClient metrics.Client
	queryDuration metrics.Histogram
	queryTotal    metrics.Counter
	queryErrors   metrics.Counter
}

// NewQueryLogger creates a new QueryLogger with the given configuration
func NewQueryLogger(cfg *QueryLoggerConfig) *QueryLogger {
	return &QueryLogger{
		config:   *cfg,
		logLevel: logger.Warn,
	}
}

// SetMetricsClient wires in the metrics client after it has been initialized.
// This is safe to call concurrently with Trace.
func (l *QueryLogger) SetMetricsClient(mc metrics.Client) {
	if mc == nil {
		return
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	l.metricsClient = mc

	l.queryDuration = mc.Histogram(metrics.HistogramOptions{
		Name:    "db_query_duration_seconds",
		Help:    "Duration of database queries in seconds",
		Labels:  []string{"operation", "table"},
		Buckets: []float64{.001, .005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5, 10},
	})

	l.queryTotal = mc.Counter(metrics.CounterOptions{
		Name:   "db_queries_total",
		Help:   "Total number of database queries",
		Labels: []string{"operation", "table"},
	})

	l.queryErrors = mc.Counter(metrics.CounterOptions{
		Name:   "db_query_errors_total",
		Help:   "Total number of database query errors",
		Labels: []string{"operation", "table"},
	})
}

// LogMode returns a new logger with the given log level
func (l *QueryLogger) LogMode(level logger.LogLevel) logger.Interface {
	return &QueryLogger{
		config:   l.config,
		logLevel: level,
	}
}

// Info logs info-level GORM messages
func (l *QueryLogger) Info(ctx context.Context, msg string, data ...any) {
	if l.logLevel >= logger.Info {
		logging.Info(ctx, fmt.Sprintf("%s %s", logPrefix, fmt.Sprintf(msg, data...)), nil)
	}
}

// Warn logs warn-level GORM messages
func (l *QueryLogger) Warn(ctx context.Context, msg string, data ...any) {
	if l.logLevel >= logger.Warn {
		logging.Warn(ctx, fmt.Sprintf("%s %s", logPrefix, fmt.Sprintf(msg, data...)), nil)
	}
}

// Error logs error-level GORM messages
func (l *QueryLogger) Error(ctx context.Context, msg string, data ...any) {
	if l.logLevel >= logger.Error {
		logging.Error(ctx, fmt.Sprintf("%s %s", logPrefix, fmt.Sprintf(msg, data...)), nil)
	}
}

// Trace is called by GORM after every query. It handles timing, logging, and metrics.
func (l *QueryLogger) Trace(ctx context.Context, begin time.Time, fc func() (sql string, rowsAffected int64), err error) {
	elapsed := time.Since(begin)
	sql, rows := fc()
	info := parseSQLInfo(sql)

	labels := map[string]string{
		"operation": info.Operation,
		"table":     info.Table,
	}

	// Record metrics if available
	l.mu.RLock()
	if l.queryDuration != nil {
		l.queryDuration.WithLabels(labels).Observe(elapsed.Seconds())
	}
	if l.queryTotal != nil {
		l.queryTotal.WithLabels(labels).Inc()
	}
	if err != nil && l.queryErrors != nil {
		l.queryErrors.WithLabels(labels).Inc()
	}
	l.mu.RUnlock()

	fields := map[string]any{
		"duration_ms": float64(elapsed.Nanoseconds()) / 1e6,
		"operation":   info.Operation,
		"table":       info.Table,
		"rows":        rows,
	}

	truncatedSQL := truncateSQL(sql)

	// Error queries
	if err != nil {
		fields["error"] = err.Error()
		fields["sql"] = truncatedSQL
		logging.Error(ctx, fmt.Sprintf("%s Query error", logPrefix), fields)
		return
	}

	// Critical slow query alert
	if l.config.AlertThreshold > 0 && elapsed >= l.config.AlertThreshold {
		fields["alert"] = "critical_slow_query"
		fields["sql"] = truncatedSQL
		logging.Error(ctx, fmt.Sprintf("%s Critical slow query", logPrefix), fields)
		return
	}

	// Slow query warning
	if l.config.SlowQueryThreshold > 0 && elapsed >= l.config.SlowQueryThreshold {
		fields["sql"] = truncatedSQL
		logging.Warn(ctx, fmt.Sprintf("%s Slow query", logPrefix), fields)
		return
	}

	// Debug logging for all queries
	if l.config.LogAllQueries {
		fields["sql"] = truncatedSQL
		logging.Debug(ctx, fmt.Sprintf("%s Query", logPrefix), fields)
	}
}

// parseSQLInfo extracts the operation type and table name from a SQL string
func parseSQLInfo(sql string) sqlInfo {
	if sql == "" {
		return sqlInfo{Operation: "OTHER", Table: "unknown"}
	}

	trimmed := strings.TrimSpace(sql)
	upper := strings.ToUpper(trimmed)

	switch {
	case strings.HasPrefix(upper, "INSERT"):
		if matches := insertRegex.FindStringSubmatch(trimmed); len(matches) > 1 {
			return sqlInfo{Operation: "INSERT", Table: matches[1]}
		}
		return sqlInfo{Operation: "INSERT", Table: "unknown"}

	case strings.HasPrefix(upper, "UPDATE"):
		if matches := updateRegex.FindStringSubmatch(trimmed); len(matches) > 1 {
			return sqlInfo{Operation: "UPDATE", Table: matches[1]}
		}
		return sqlInfo{Operation: "UPDATE", Table: "unknown"}

	case strings.HasPrefix(upper, "DELETE"):
		if matches := deleteRegex.FindStringSubmatch(trimmed); len(matches) > 1 {
			return sqlInfo{Operation: "DELETE", Table: matches[1]}
		}
		return sqlInfo{Operation: "DELETE", Table: "unknown"}

	case strings.HasPrefix(upper, "SELECT"):
		if matches := selectRegex.FindStringSubmatch(trimmed); len(matches) > 1 {
			return sqlInfo{Operation: "SELECT", Table: matches[1]}
		}
		return sqlInfo{Operation: "SELECT", Table: "unknown"}

	default:
		return sqlInfo{Operation: "OTHER", Table: "unknown"}
	}
}

// truncateSQL truncates long SQL strings to prevent log bloat
func truncateSQL(sql string) string {
	if len(sql) <= maxSQLLength {
		return sql
	}
	return sql[:maxSQLLength] + "...[truncated]"
}
