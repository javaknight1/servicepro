package analytics

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

// PostgresEventStore implements EventStore using PostgreSQL
type PostgresEventStore struct {
	db *sql.DB
}

// NewPostgresEventStore creates a new PostgreSQL event store
func NewPostgresEventStore(db *sql.DB) *PostgresEventStore {
	return &PostgresEventStore{db: db}
}

// Save saves a single event to the database
func (s *PostgresEventStore) Save(ctx context.Context, event *Event) error {
	return s.SaveBatch(ctx, []*Event{event})
}

// SaveBatch saves multiple events to the database
func (s *PostgresEventStore) SaveBatch(ctx context.Context, events []*Event) error {
	if len(events) == 0 {
		return nil
	}

	// Build bulk insert query
	query := `
		INSERT INTO analytics.events (
			id, name, category, user_id, session_id, tenant_id,
			properties, timestamp, source, version, environment,
			ip_address, user_agent, referrer, device_type,
			country, region, city
		) VALUES `

	values := make([]interface{}, 0, len(events)*18)
	placeholders := make([]string, 0, len(events))

	for i, event := range events {
		offset := i * 18
		placeholders = append(placeholders, fmt.Sprintf(
			"($%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d)",
			offset+1, offset+2, offset+3, offset+4, offset+5, offset+6,
			offset+7, offset+8, offset+9, offset+10, offset+11, offset+12,
			offset+13, offset+14, offset+15, offset+16, offset+17, offset+18,
		))

		propsJSON, err := event.PropertiesJSON()
		if err != nil {
			return fmt.Errorf("failed to marshal properties: %w", err)
		}

		values = append(values,
			event.ID,
			event.Name,
			event.Category,
			event.UserID,
			nullString(event.SessionID),
			event.TenantID,
			propsJSON,
			event.Timestamp,
			event.Source,
			nullString(event.Version),
			event.Environment,
			nullString(event.IPAddress),
			nullString(event.UserAgent),
			nullString(event.Referrer),
			nullString(event.DeviceType),
			nullString(event.Country),
			nullString(event.Region),
			nullString(event.City),
		)
	}

	query += strings.Join(placeholders, ", ")
	query += " ON CONFLICT (id) DO NOTHING"

	_, err := s.db.ExecContext(ctx, query, values...)
	if err != nil {
		return fmt.Errorf("failed to insert events: %w", err)
	}

	return nil
}

// Query queries events from the database
func (s *PostgresEventStore) Query(ctx context.Context, q EventQuery) ([]*Event, error) {
	query, args := s.buildQuery(q, false)
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query events: %w", err)
	}
	defer rows.Close()

	var events []*Event
	for rows.Next() {
		event, err := s.scanEvent(rows)
		if err != nil {
			return nil, err
		}
		events = append(events, event)
	}

	return events, rows.Err()
}

// Count counts events matching the query
func (s *PostgresEventStore) Count(ctx context.Context, q EventQuery) (int64, error) {
	query, args := s.buildQuery(q, true)
	var count int64
	err := s.db.QueryRowContext(ctx, query, args...).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count events: %w", err)
	}
	return count, nil
}

// Delete deletes events older than the specified time
func (s *PostgresEventStore) Delete(ctx context.Context, before time.Time) (int64, error) {
	result, err := s.db.ExecContext(ctx,
		"DELETE FROM analytics.events WHERE timestamp < $1",
		before,
	)
	if err != nil {
		return 0, fmt.Errorf("failed to delete events: %w", err)
	}
	return result.RowsAffected()
}

// buildQuery builds the SQL query from EventQuery
func (s *PostgresEventStore) buildQuery(q EventQuery, countOnly bool) (string, []interface{}) {
	var query strings.Builder
	var args []interface{}
	argIndex := 1

	if countOnly {
		query.WriteString("SELECT COUNT(*) FROM analytics.events WHERE 1=1")
	} else {
		query.WriteString(`
			SELECT id, name, category, user_id, session_id, tenant_id,
			       properties, timestamp, source, version, environment,
			       ip_address, user_agent, referrer, device_type,
			       country, region, city
			FROM analytics.events WHERE 1=1
		`)
	}

	if len(q.Names) > 0 {
		placeholders := make([]string, len(q.Names))
		for i, name := range q.Names {
			placeholders[i] = fmt.Sprintf("$%d", argIndex)
			args = append(args, name)
			argIndex++
		}
		query.WriteString(fmt.Sprintf(" AND name IN (%s)", strings.Join(placeholders, ", ")))
	}

	if len(q.Categories) > 0 {
		placeholders := make([]string, len(q.Categories))
		for i, cat := range q.Categories {
			placeholders[i] = fmt.Sprintf("$%d", argIndex)
			args = append(args, cat)
			argIndex++
		}
		query.WriteString(fmt.Sprintf(" AND category IN (%s)", strings.Join(placeholders, ", ")))
	}

	if q.UserID != nil {
		query.WriteString(fmt.Sprintf(" AND user_id = $%d", argIndex))
		args = append(args, q.UserID)
		argIndex++
	}

	if q.TenantID != nil {
		query.WriteString(fmt.Sprintf(" AND tenant_id = $%d", argIndex))
		args = append(args, q.TenantID)
		argIndex++
	}

	if q.SessionID != "" {
		query.WriteString(fmt.Sprintf(" AND session_id = $%d", argIndex))
		args = append(args, q.SessionID)
		argIndex++
	}

	if q.StartTime != nil {
		query.WriteString(fmt.Sprintf(" AND timestamp >= $%d", argIndex))
		args = append(args, q.StartTime)
		argIndex++
	}

	if q.EndTime != nil {
		query.WriteString(fmt.Sprintf(" AND timestamp <= $%d", argIndex))
		args = append(args, q.EndTime)
		argIndex++
	}

	if !countOnly {
		orderBy := "timestamp"
		if q.OrderBy != "" {
			orderBy = q.OrderBy
		}
		orderDir := "DESC"
		if q.OrderDir != "" {
			orderDir = strings.ToUpper(q.OrderDir)
		}
		query.WriteString(fmt.Sprintf(" ORDER BY %s %s", orderBy, orderDir))

		if q.Limit > 0 {
			query.WriteString(fmt.Sprintf(" LIMIT $%d", argIndex))
			args = append(args, q.Limit)
			argIndex++
		}

		if q.Offset > 0 {
			query.WriteString(fmt.Sprintf(" OFFSET $%d", argIndex))
			args = append(args, q.Offset)
		}
	}

	return query.String(), args
}

// scanEvent scans a row into an Event
func (s *PostgresEventStore) scanEvent(rows *sql.Rows) (*Event, error) {
	var event Event
	var propsJSON []byte
	var sessionID, version, ipAddress, userAgent, referrer, deviceType sql.NullString
	var country, region, city sql.NullString

	err := rows.Scan(
		&event.ID,
		&event.Name,
		&event.Category,
		&event.UserID,
		&sessionID,
		&event.TenantID,
		&propsJSON,
		&event.Timestamp,
		&event.Source,
		&version,
		&event.Environment,
		&ipAddress,
		&userAgent,
		&referrer,
		&deviceType,
		&country,
		&region,
		&city,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to scan event: %w", err)
	}

	if len(propsJSON) > 0 {
		if err := json.Unmarshal(propsJSON, &event.Properties); err != nil {
			return nil, fmt.Errorf("failed to unmarshal properties: %w", err)
		}
	}

	event.SessionID = sessionID.String
	event.Version = version.String
	event.IPAddress = ipAddress.String
	event.UserAgent = userAgent.String
	event.Referrer = referrer.String
	event.DeviceType = deviceType.String
	event.Country = country.String
	event.Region = region.String
	event.City = city.String

	return &event, nil
}

// Helper functions

func nullString(s string) sql.NullString {
	if s == "" {
		return sql.NullString{}
	}
	return sql.NullString{String: s, Valid: true}
}

// AggregatedMetric represents an aggregated analytics metric
type AggregatedMetric struct {
	ID          uuid.UUID              `json:"id" db:"id"`
	MetricName  string                 `json:"metric_name" db:"metric_name"`
	Dimension   string                 `json:"dimension" db:"dimension"`
	DimValue    string                 `json:"dim_value" db:"dim_value"`
	TenantID    *uuid.UUID             `json:"tenant_id,omitempty" db:"tenant_id"`
	Value       float64                `json:"value" db:"value"`
	Count       int64                  `json:"count" db:"count"`
	PeriodStart time.Time              `json:"period_start" db:"period_start"`
	PeriodEnd   time.Time              `json:"period_end" db:"period_end"`
	Granularity string                 `json:"granularity" db:"granularity"` // hourly, daily, weekly, monthly
	Metadata    map[string]interface{} `json:"metadata,omitempty" db:"metadata"`
	CreatedAt   time.Time              `json:"created_at" db:"created_at"`
}

// MetricsStore is the interface for storing aggregated metrics
type MetricsStore interface {
	SaveMetric(ctx context.Context, metric *AggregatedMetric) error
	SaveMetrics(ctx context.Context, metrics []*AggregatedMetric) error
	GetMetrics(ctx context.Context, query MetricQuery) ([]*AggregatedMetric, error)
	DeleteOldMetrics(ctx context.Context, before time.Time, granularity string) (int64, error)
}

// MetricQuery represents a query for aggregated metrics
type MetricQuery struct {
	MetricNames []string   `json:"metric_names,omitempty"`
	Dimensions  []string   `json:"dimensions,omitempty"`
	TenantID    *uuid.UUID `json:"tenant_id,omitempty"`
	StartTime   *time.Time `json:"start_time,omitempty"`
	EndTime     *time.Time `json:"end_time,omitempty"`
	Granularity string     `json:"granularity,omitempty"`
	Limit       int        `json:"limit,omitempty"`
}

// PostgresMetricsStore implements MetricsStore using PostgreSQL
type PostgresMetricsStore struct {
	db *sql.DB
}

// NewPostgresMetricsStore creates a new PostgreSQL metrics store
func NewPostgresMetricsStore(db *sql.DB) *PostgresMetricsStore {
	return &PostgresMetricsStore{db: db}
}

// SaveMetric saves a single aggregated metric
func (s *PostgresMetricsStore) SaveMetric(ctx context.Context, metric *AggregatedMetric) error {
	return s.SaveMetrics(ctx, []*AggregatedMetric{metric})
}

// SaveMetrics saves multiple aggregated metrics
func (s *PostgresMetricsStore) SaveMetrics(ctx context.Context, metrics []*AggregatedMetric) error {
	if len(metrics) == 0 {
		return nil
	}

	query := `
		INSERT INTO analytics.aggregated_metrics (
			id, metric_name, dimension, dim_value, tenant_id,
			value, count, period_start, period_end, granularity,
			metadata, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		ON CONFLICT (metric_name, dimension, dim_value, period_start, granularity, COALESCE(tenant_id, '00000000-0000-0000-0000-000000000000'))
		DO UPDATE SET value = EXCLUDED.value, count = EXCLUDED.count
	`

	for _, metric := range metrics {
		metadataJSON, err := json.Marshal(metric.Metadata)
		if err != nil {
			return fmt.Errorf("failed to marshal metadata: %w", err)
		}

		_, err = s.db.ExecContext(ctx, query,
			metric.ID,
			metric.MetricName,
			metric.Dimension,
			metric.DimValue,
			metric.TenantID,
			metric.Value,
			metric.Count,
			metric.PeriodStart,
			metric.PeriodEnd,
			metric.Granularity,
			metadataJSON,
			metric.CreatedAt,
		)
		if err != nil {
			return fmt.Errorf("failed to insert metric: %w", err)
		}
	}

	return nil
}

// GetMetrics retrieves aggregated metrics
func (s *PostgresMetricsStore) GetMetrics(ctx context.Context, q MetricQuery) ([]*AggregatedMetric, error) {
	var query strings.Builder
	var args []interface{}
	argIndex := 1

	query.WriteString(`
		SELECT id, metric_name, dimension, dim_value, tenant_id,
		       value, count, period_start, period_end, granularity,
		       metadata, created_at
		FROM analytics.aggregated_metrics WHERE 1=1
	`)

	if len(q.MetricNames) > 0 {
		placeholders := make([]string, len(q.MetricNames))
		for i, name := range q.MetricNames {
			placeholders[i] = fmt.Sprintf("$%d", argIndex)
			args = append(args, name)
			argIndex++
		}
		query.WriteString(fmt.Sprintf(" AND metric_name IN (%s)", strings.Join(placeholders, ", ")))
	}

	if q.TenantID != nil {
		query.WriteString(fmt.Sprintf(" AND tenant_id = $%d", argIndex))
		args = append(args, q.TenantID)
		argIndex++
	}

	if q.StartTime != nil {
		query.WriteString(fmt.Sprintf(" AND period_start >= $%d", argIndex))
		args = append(args, q.StartTime)
		argIndex++
	}

	if q.EndTime != nil {
		query.WriteString(fmt.Sprintf(" AND period_end <= $%d", argIndex))
		args = append(args, q.EndTime)
		argIndex++
	}

	if q.Granularity != "" {
		query.WriteString(fmt.Sprintf(" AND granularity = $%d", argIndex))
		args = append(args, q.Granularity)
		argIndex++
	}

	query.WriteString(" ORDER BY period_start DESC")

	if q.Limit > 0 {
		query.WriteString(fmt.Sprintf(" LIMIT $%d", argIndex))
		args = append(args, q.Limit)
	}

	rows, err := s.db.QueryContext(ctx, query.String(), args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query metrics: %w", err)
	}
	defer rows.Close()

	var metrics []*AggregatedMetric
	for rows.Next() {
		var m AggregatedMetric
		var metadataJSON []byte

		err := rows.Scan(
			&m.ID,
			&m.MetricName,
			&m.Dimension,
			&m.DimValue,
			&m.TenantID,
			&m.Value,
			&m.Count,
			&m.PeriodStart,
			&m.PeriodEnd,
			&m.Granularity,
			&metadataJSON,
			&m.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan metric: %w", err)
		}

		if len(metadataJSON) > 0 {
			if err := json.Unmarshal(metadataJSON, &m.Metadata); err != nil {
				return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
			}
		}

		metrics = append(metrics, &m)
	}

	return metrics, rows.Err()
}

// DeleteOldMetrics deletes metrics older than the specified time
func (s *PostgresMetricsStore) DeleteOldMetrics(ctx context.Context, before time.Time, granularity string) (int64, error) {
	result, err := s.db.ExecContext(ctx,
		"DELETE FROM analytics.aggregated_metrics WHERE period_end < $1 AND granularity = $2",
		before, granularity,
	)
	if err != nil {
		return 0, fmt.Errorf("failed to delete metrics: %w", err)
	}
	return result.RowsAffected()
}
