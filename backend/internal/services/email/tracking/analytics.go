package tracking

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"
)

// Standard errors for analytics
var (
	ErrInvalidDateRange  = errors.New("invalid date range")
	ErrInvalidGroupBy    = errors.New("invalid group by parameter")
	ErrAnalyticsDisabled = errors.New("analytics is disabled")
)

// AnalyticsConfig holds configuration for analytics
type AnalyticsConfig struct {
	DefaultTimeRange    time.Duration `json:"default_time_range"`
	MaxTimeRange        time.Duration `json:"max_time_range"`
	CacheEnabled        bool          `json:"cache_enabled"`
	CacheTTL            time.Duration `json:"cache_ttl"`
	AggregationInterval time.Duration `json:"aggregation_interval"`
	TablePrefix         string        `json:"table_prefix"`
}

// DefaultAnalyticsConfig returns sensible defaults
func DefaultAnalyticsConfig() *AnalyticsConfig {
	return &AnalyticsConfig{
		DefaultTimeRange:    30 * 24 * time.Hour,  // 30 days
		MaxTimeRange:        365 * 24 * time.Hour, // 1 year
		CacheEnabled:        true,
		CacheTTL:            5 * time.Minute,
		AggregationInterval: 24 * time.Hour, // Daily aggregation
		TablePrefix:         "email_tracking_",
	}
}

// AnalyticsService provides email tracking analytics
type AnalyticsService struct {
	db      *sql.DB
	config  *AnalyticsConfig
	storage *TrackingStorage

	// Cache
	cache   map[string]*cachedResult
	cacheMu sync.RWMutex
}

// cachedResult holds a cached analytics result
type cachedResult struct {
	Data      interface{}
	ExpiresAt time.Time
}

// NewAnalyticsService creates a new analytics service
func NewAnalyticsService(db *sql.DB, storage *TrackingStorage, config *AnalyticsConfig) *AnalyticsService {
	if config == nil {
		config = DefaultAnalyticsConfig()
	}

	as := &AnalyticsService{
		db:      db,
		config:  config,
		storage: storage,
		cache:   make(map[string]*cachedResult),
	}

	// Start cache cleanup
	if config.CacheEnabled {
		go as.cleanupCache()
	}

	return as
}

// cleanupCache periodically removes expired cache entries
func (as *AnalyticsService) cleanupCache() {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		as.cacheMu.Lock()
		now := time.Now()
		for key, result := range as.cache {
			if result.ExpiresAt.Before(now) {
				delete(as.cache, key)
			}
		}
		as.cacheMu.Unlock()
	}
}

// getCached retrieves a cached result
func (as *AnalyticsService) getCached(key string) (interface{}, bool) {
	if !as.config.CacheEnabled {
		return nil, false
	}

	as.cacheMu.RLock()
	defer as.cacheMu.RUnlock()

	result, exists := as.cache[key]
	if !exists || result.ExpiresAt.Before(time.Now()) {
		return nil, false
	}
	return result.Data, true
}

// setCache stores a result in the cache
func (as *AnalyticsService) setCache(key string, data interface{}) {
	if !as.config.CacheEnabled {
		return
	}

	as.cacheMu.Lock()
	defer as.cacheMu.Unlock()

	as.cache[key] = &cachedResult{
		Data:      data,
		ExpiresAt: time.Now().Add(as.config.CacheTTL),
	}
}

// AnalyticsQuery represents a query for analytics data
type AnalyticsQuery struct {
	CampaignID  string    `json:"campaign_id,omitempty"`
	EmailID     string    `json:"email_id,omitempty"`
	RecipientID string    `json:"recipient_id,omitempty"`
	StartDate   time.Time `json:"start_date"`
	EndDate     time.Time `json:"end_date"`
	GroupBy     string    `json:"group_by,omitempty"` // day, week, month, campaign
	EventType   EventType `json:"event_type,omitempty"`
	Limit       int       `json:"limit,omitempty"`
}

// Validate validates the analytics query
func (q *AnalyticsQuery) Validate(config *AnalyticsConfig) error {
	if q.EndDate.Before(q.StartDate) {
		return ErrInvalidDateRange
	}

	if q.EndDate.Sub(q.StartDate) > config.MaxTimeRange {
		return fmt.Errorf("%w: maximum range is %v", ErrInvalidDateRange, config.MaxTimeRange)
	}

	validGroupBy := map[string]bool{"": true, "day": true, "week": true, "month": true, "campaign": true}
	if !validGroupBy[q.GroupBy] {
		return ErrInvalidGroupBy
	}

	return nil
}

// DashboardSummary contains summary statistics for the dashboard
type DashboardSummary struct {
	TotalSent         int64      `json:"total_sent"`
	TotalDelivered    int64      `json:"total_delivered"`
	TotalOpens        int64      `json:"total_opens"`
	UniqueOpens       int64      `json:"unique_opens"`
	TotalClicks       int64      `json:"total_clicks"`
	UniqueClicks      int64      `json:"unique_clicks"`
	TotalBounces      int64      `json:"total_bounces"`
	TotalComplaints   int64      `json:"total_complaints"`
	TotalUnsubscribes int64      `json:"total_unsubscribes"`
	OpenRate          float64    `json:"open_rate"`
	ClickRate         float64    `json:"click_rate"`
	ClickToOpenRate   float64    `json:"click_to_open_rate"`
	BounceRate        float64    `json:"bounce_rate"`
	ComplaintRate     float64    `json:"complaint_rate"`
	UnsubscribeRate   float64    `json:"unsubscribe_rate"`
	Period            string     `json:"period"`
	TrendComparison   *TrendData `json:"trend_comparison,omitempty"`
}

// TrendData shows comparison with previous period
type TrendData struct {
	OpenRateChange        float64 `json:"open_rate_change"`
	ClickRateChange       float64 `json:"click_rate_change"`
	UnsubscribeRateChange float64 `json:"unsubscribe_rate_change"`
}

// GetDashboardSummary retrieves summary statistics for the dashboard
func (as *AnalyticsService) GetDashboardSummary(ctx context.Context, query *AnalyticsQuery) (*DashboardSummary, error) {
	if query.StartDate.IsZero() {
		query.StartDate = time.Now().Add(-as.config.DefaultTimeRange)
	}
	if query.EndDate.IsZero() {
		query.EndDate = time.Now()
	}

	if err := query.Validate(as.config); err != nil {
		return nil, err
	}

	// Check cache
	cacheKey := fmt.Sprintf("summary:%s:%s:%s", query.CampaignID, query.StartDate.Format("2006-01-02"), query.EndDate.Format("2006-01-02"))
	if cached, ok := as.getCached(cacheKey); ok {
		return cached.(*DashboardSummary), nil
	}

	summary := &DashboardSummary{}

	// Build base query conditions
	conditions := []string{"timestamp >= $1", "timestamp <= $2"}
	args := []interface{}{query.StartDate, query.EndDate}
	argIndex := 3

	if query.CampaignID != "" {
		conditions = append(conditions, fmt.Sprintf("campaign_id = $%d", argIndex))
		args = append(args, query.CampaignID)
		argIndex++
	}

	whereClause := strings.Join(conditions, " AND ")

	// Get event counts
	eventQuery := fmt.Sprintf(`
		SELECT
			event_type,
			COUNT(*) as total,
			COUNT(CASE WHEN is_unique THEN 1 END) as unique_count
		FROM %sevents
		WHERE %s
		GROUP BY event_type
	`, as.config.TablePrefix, whereClause)

	rows, err := as.db.QueryContext(ctx, eventQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query events: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var eventType string
		var total, unique int64
		if err := rows.Scan(&eventType, &total, &unique); err != nil {
			return nil, err
		}

		switch EventType(eventType) {
		case EventTypeDelivery:
			summary.TotalDelivered = total
		case EventTypeOpen:
			summary.TotalOpens = total
			summary.UniqueOpens = unique
		case EventTypeClick:
			summary.TotalClicks = total
			summary.UniqueClicks = unique
		case EventTypeBounce:
			summary.TotalBounces = total
		case EventTypeComplaint:
			summary.TotalComplaints = total
		case EventTypeUnsubscribe:
			summary.TotalUnsubscribes = total
		}
	}

	// Get total sent from emails table
	sentQuery := fmt.Sprintf(`
		SELECT COUNT(*) FROM %semails
		WHERE sent_at >= $1 AND sent_at <= $2
	`, as.config.TablePrefix)
	sentArgs := []interface{}{query.StartDate, query.EndDate}

	if query.CampaignID != "" {
		sentQuery = fmt.Sprintf(`
			SELECT COUNT(*) FROM %semails
			WHERE sent_at >= $1 AND sent_at <= $2 AND campaign_id = $3
		`, as.config.TablePrefix)
		sentArgs = append(sentArgs, query.CampaignID)
	}

	as.db.QueryRowContext(ctx, sentQuery, sentArgs...).Scan(&summary.TotalSent)

	// Calculate rates
	if summary.TotalSent > 0 {
		summary.OpenRate = float64(summary.UniqueOpens) / float64(summary.TotalSent) * 100
		summary.ClickRate = float64(summary.UniqueClicks) / float64(summary.TotalSent) * 100
		summary.BounceRate = float64(summary.TotalBounces) / float64(summary.TotalSent) * 100
		summary.ComplaintRate = float64(summary.TotalComplaints) / float64(summary.TotalSent) * 100
		summary.UnsubscribeRate = float64(summary.TotalUnsubscribes) / float64(summary.TotalSent) * 100
	}

	if summary.UniqueOpens > 0 {
		summary.ClickToOpenRate = float64(summary.UniqueClicks) / float64(summary.UniqueOpens) * 100
	}

	summary.Period = fmt.Sprintf("%s to %s", query.StartDate.Format("2006-01-02"), query.EndDate.Format("2006-01-02"))

	// Calculate trend (compare to previous period)
	periodDuration := query.EndDate.Sub(query.StartDate)
	prevQuery := &AnalyticsQuery{
		CampaignID: query.CampaignID,
		StartDate:  query.StartDate.Add(-periodDuration),
		EndDate:    query.StartDate,
	}

	prevSummary, err := as.getPreviousPeriodStats(ctx, prevQuery)
	if err == nil && prevSummary != nil {
		summary.TrendComparison = &TrendData{
			OpenRateChange:        summary.OpenRate - prevSummary.OpenRate,
			ClickRateChange:       summary.ClickRate - prevSummary.ClickRate,
			UnsubscribeRateChange: summary.UnsubscribeRate - prevSummary.UnsubscribeRate,
		}
	}

	as.setCache(cacheKey, summary)
	return summary, nil
}

// getPreviousPeriodStats gets stats for the previous period (for trend comparison)
func (as *AnalyticsService) getPreviousPeriodStats(ctx context.Context, query *AnalyticsQuery) (*DashboardSummary, error) {
	// Simplified version without trend calculation to avoid recursion
	summary := &DashboardSummary{}

	conditions := []string{"timestamp >= $1", "timestamp <= $2"}
	args := []interface{}{query.StartDate, query.EndDate}
	argIndex := 3

	if query.CampaignID != "" {
		conditions = append(conditions, fmt.Sprintf("campaign_id = $%d", argIndex))
		args = append(args, query.CampaignID)
	}

	whereClause := strings.Join(conditions, " AND ")

	eventQuery := fmt.Sprintf(`
		SELECT
			event_type,
			COUNT(CASE WHEN is_unique THEN 1 END) as unique_count
		FROM %sevents
		WHERE %s
		GROUP BY event_type
	`, as.config.TablePrefix, whereClause)

	rows, err := as.db.QueryContext(ctx, eventQuery, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var eventType string
		var unique int64
		if err := rows.Scan(&eventType, &unique); err != nil {
			continue
		}

		switch EventType(eventType) {
		case EventTypeOpen:
			summary.UniqueOpens = unique
		case EventTypeClick:
			summary.UniqueClicks = unique
		case EventTypeUnsubscribe:
			summary.TotalUnsubscribes = unique
		}
	}

	// Get total sent
	sentQuery := fmt.Sprintf(`
		SELECT COUNT(*) FROM %semails
		WHERE sent_at >= $1 AND sent_at <= $2
	`, as.config.TablePrefix)
	as.db.QueryRowContext(ctx, sentQuery, query.StartDate, query.EndDate).Scan(&summary.TotalSent)

	if summary.TotalSent > 0 {
		summary.OpenRate = float64(summary.UniqueOpens) / float64(summary.TotalSent) * 100
		summary.ClickRate = float64(summary.UniqueClicks) / float64(summary.TotalSent) * 100
		summary.UnsubscribeRate = float64(summary.TotalUnsubscribes) / float64(summary.TotalSent) * 100
	}

	return summary, nil
}

// TimeSeriesData represents time-bucketed analytics data
type TimeSeriesData struct {
	Timestamp    time.Time `json:"timestamp"`
	Date         string    `json:"date"`
	Opens        int64     `json:"opens"`
	UniqueOpens  int64     `json:"unique_opens"`
	Clicks       int64     `json:"clicks"`
	UniqueClicks int64     `json:"unique_clicks"`
	Bounces      int64     `json:"bounces"`
	Unsubscribes int64     `json:"unsubscribes"`
	Sent         int64     `json:"sent"`
	OpenRate     float64   `json:"open_rate"`
	ClickRate    float64   `json:"click_rate"`
}

// GetTimeSeries retrieves time-series analytics data
func (as *AnalyticsService) GetTimeSeries(ctx context.Context, query *AnalyticsQuery) ([]*TimeSeriesData, error) {
	if query.StartDate.IsZero() {
		query.StartDate = time.Now().Add(-as.config.DefaultTimeRange)
	}
	if query.EndDate.IsZero() {
		query.EndDate = time.Now()
	}
	if query.GroupBy == "" {
		query.GroupBy = "day"
	}

	if err := query.Validate(as.config); err != nil {
		return nil, err
	}

	// Check cache
	cacheKey := fmt.Sprintf("timeseries:%s:%s:%s:%s", query.CampaignID, query.GroupBy, query.StartDate.Format("2006-01-02"), query.EndDate.Format("2006-01-02"))
	if cached, ok := as.getCached(cacheKey); ok {
		return cached.([]*TimeSeriesData), nil
	}

	// Determine date truncation
	var dateTrunc string
	switch query.GroupBy {
	case "day":
		dateTrunc = "day"
	case "week":
		dateTrunc = "week"
	case "month":
		dateTrunc = "month"
	default:
		dateTrunc = "day"
	}

	conditions := []string{"timestamp >= $1", "timestamp <= $2"}
	args := []interface{}{query.StartDate, query.EndDate}
	argIndex := 3

	if query.CampaignID != "" {
		conditions = append(conditions, fmt.Sprintf("campaign_id = $%d", argIndex))
		args = append(args, query.CampaignID)
	}

	whereClause := strings.Join(conditions, " AND ")

	// Query for aggregated data
	tsQuery := fmt.Sprintf(`
		SELECT
			date_trunc('%s', timestamp) as period,
			event_type,
			COUNT(*) as total,
			COUNT(CASE WHEN is_unique THEN 1 END) as unique_count
		FROM %sevents
		WHERE %s
		GROUP BY date_trunc('%s', timestamp), event_type
		ORDER BY period
	`, dateTrunc, as.config.TablePrefix, whereClause, dateTrunc)

	rows, err := as.db.QueryContext(ctx, tsQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query time series: %w", err)
	}
	defer rows.Close()

	// Aggregate by period
	periodData := make(map[time.Time]*TimeSeriesData)

	for rows.Next() {
		var period time.Time
		var eventType string
		var total, unique int64

		if err := rows.Scan(&period, &eventType, &total, &unique); err != nil {
			continue
		}

		if _, exists := periodData[period]; !exists {
			periodData[period] = &TimeSeriesData{
				Timestamp: period,
				Date:      period.Format("2006-01-02"),
			}
		}

		data := periodData[period]
		switch EventType(eventType) {
		case EventTypeOpen:
			data.Opens = total
			data.UniqueOpens = unique
		case EventTypeClick:
			data.Clicks = total
			data.UniqueClicks = unique
		case EventTypeBounce:
			data.Bounces = total
		case EventTypeUnsubscribe:
			data.Unsubscribes = total
		}
	}

	// Get sent counts per period
	sentQuery := fmt.Sprintf(`
		SELECT
			date_trunc('%s', sent_at) as period,
			COUNT(*) as sent
		FROM %semails
		WHERE sent_at >= $1 AND sent_at <= $2
	`, dateTrunc, as.config.TablePrefix)

	sentArgs := []interface{}{query.StartDate, query.EndDate}
	if query.CampaignID != "" {
		sentQuery += " AND campaign_id = $3"
		sentArgs = append(sentArgs, query.CampaignID)
	}
	sentQuery += fmt.Sprintf(" GROUP BY date_trunc('%s', sent_at)", dateTrunc)

	sentRows, err := as.db.QueryContext(ctx, sentQuery, sentArgs...)
	if err == nil {
		defer sentRows.Close()
		for sentRows.Next() {
			var period time.Time
			var sent int64
			if sentRows.Scan(&period, &sent) == nil {
				if data, exists := periodData[period]; exists {
					data.Sent = sent
				} else {
					periodData[period] = &TimeSeriesData{
						Timestamp: period,
						Date:      period.Format("2006-01-02"),
						Sent:      sent,
					}
				}
			}
		}
	}

	// Calculate rates and convert to slice
	result := make([]*TimeSeriesData, 0, len(periodData))
	for _, data := range periodData {
		if data.Sent > 0 {
			data.OpenRate = float64(data.UniqueOpens) / float64(data.Sent) * 100
			data.ClickRate = float64(data.UniqueClicks) / float64(data.Sent) * 100
		}
		result = append(result, data)
	}

	// Sort by timestamp
	sort.Slice(result, func(i, j int) bool {
		return result[i].Timestamp.Before(result[j].Timestamp)
	})

	as.setCache(cacheKey, result)
	return result, nil
}

// CampaignPerformance contains performance metrics for a campaign
type CampaignPerformance struct {
	CampaignID      string     `json:"campaign_id"`
	TotalSent       int64      `json:"total_sent"`
	TotalDelivered  int64      `json:"total_delivered"`
	UniqueOpens     int64      `json:"unique_opens"`
	UniqueClicks    int64      `json:"unique_clicks"`
	Bounces         int64      `json:"bounces"`
	Unsubscribes    int64      `json:"unsubscribes"`
	OpenRate        float64    `json:"open_rate"`
	ClickRate       float64    `json:"click_rate"`
	ClickToOpenRate float64    `json:"click_to_open_rate"`
	BounceRate      float64    `json:"bounce_rate"`
	FirstSentAt     *time.Time `json:"first_sent_at,omitempty"`
	LastSentAt      *time.Time `json:"last_sent_at,omitempty"`
}

// GetCampaignPerformance retrieves performance metrics for campaigns
func (as *AnalyticsService) GetCampaignPerformance(ctx context.Context, query *AnalyticsQuery) ([]*CampaignPerformance, error) {
	if query.StartDate.IsZero() {
		query.StartDate = time.Now().Add(-as.config.DefaultTimeRange)
	}
	if query.EndDate.IsZero() {
		query.EndDate = time.Now()
	}

	// Check cache
	cacheKey := fmt.Sprintf("campaigns:%s:%s", query.StartDate.Format("2006-01-02"), query.EndDate.Format("2006-01-02"))
	if cached, ok := as.getCached(cacheKey); ok {
		return cached.([]*CampaignPerformance), nil
	}

	// Get campaign metrics from events
	eventQuery := fmt.Sprintf(`
		SELECT
			campaign_id,
			event_type,
			COUNT(*) as total,
			COUNT(CASE WHEN is_unique THEN 1 END) as unique_count
		FROM %sevents
		WHERE timestamp >= $1 AND timestamp <= $2 AND campaign_id IS NOT NULL AND campaign_id != ''
		GROUP BY campaign_id, event_type
	`, as.config.TablePrefix)

	rows, err := as.db.QueryContext(ctx, eventQuery, query.StartDate, query.EndDate)
	if err != nil {
		return nil, fmt.Errorf("failed to query campaign performance: %w", err)
	}
	defer rows.Close()

	campaigns := make(map[string]*CampaignPerformance)

	for rows.Next() {
		var campaignID, eventType string
		var total, unique int64

		if err := rows.Scan(&campaignID, &eventType, &total, &unique); err != nil {
			continue
		}

		if _, exists := campaigns[campaignID]; !exists {
			campaigns[campaignID] = &CampaignPerformance{
				CampaignID: campaignID,
			}
		}

		perf := campaigns[campaignID]
		switch EventType(eventType) {
		case EventTypeDelivery:
			perf.TotalDelivered = total
		case EventTypeOpen:
			perf.UniqueOpens = unique
		case EventTypeClick:
			perf.UniqueClicks = unique
		case EventTypeBounce:
			perf.Bounces = total
		case EventTypeUnsubscribe:
			perf.Unsubscribes = total
		}
	}

	// Get sent counts and dates per campaign
	sentQuery := fmt.Sprintf(`
		SELECT
			campaign_id,
			COUNT(*) as sent,
			MIN(sent_at) as first_sent,
			MAX(sent_at) as last_sent
		FROM %semails
		WHERE sent_at >= $1 AND sent_at <= $2 AND campaign_id IS NOT NULL AND campaign_id != ''
		GROUP BY campaign_id
	`, as.config.TablePrefix)

	sentRows, err := as.db.QueryContext(ctx, sentQuery, query.StartDate, query.EndDate)
	if err == nil {
		defer sentRows.Close()
		for sentRows.Next() {
			var campaignID string
			var sent int64
			var firstSent, lastSent time.Time

			if sentRows.Scan(&campaignID, &sent, &firstSent, &lastSent) == nil {
				if perf, exists := campaigns[campaignID]; exists {
					perf.TotalSent = sent
					perf.FirstSentAt = &firstSent
					perf.LastSentAt = &lastSent
				}
			}
		}
	}

	// Calculate rates
	result := make([]*CampaignPerformance, 0, len(campaigns))
	for _, perf := range campaigns {
		if perf.TotalSent > 0 {
			perf.OpenRate = float64(perf.UniqueOpens) / float64(perf.TotalSent) * 100
			perf.ClickRate = float64(perf.UniqueClicks) / float64(perf.TotalSent) * 100
			perf.BounceRate = float64(perf.Bounces) / float64(perf.TotalSent) * 100
		}
		if perf.UniqueOpens > 0 {
			perf.ClickToOpenRate = float64(perf.UniqueClicks) / float64(perf.UniqueOpens) * 100
		}
		result = append(result, perf)
	}

	// Sort by sent count descending
	sort.Slice(result, func(i, j int) bool {
		return result[i].TotalSent > result[j].TotalSent
	})

	if query.Limit > 0 && len(result) > query.Limit {
		result = result[:query.Limit]
	}

	as.setCache(cacheKey, result)
	return result, nil
}

// RecipientEngagement contains engagement metrics for a recipient
type RecipientEngagement struct {
	RecipientID     string     `json:"recipient_id"`
	Email           string     `json:"email,omitempty"`
	TotalReceived   int64      `json:"total_received"`
	TotalOpened     int64      `json:"total_opened"`
	TotalClicked    int64      `json:"total_clicked"`
	EngagementScore float64    `json:"engagement_score"`
	LastOpenAt      *time.Time `json:"last_open_at,omitempty"`
	LastClickAt     *time.Time `json:"last_click_at,omitempty"`
	IsActive        bool       `json:"is_active"`
}

// GetTopEngagedRecipients retrieves the most engaged recipients
func (as *AnalyticsService) GetTopEngagedRecipients(ctx context.Context, query *AnalyticsQuery) ([]*RecipientEngagement, error) {
	if query.Limit == 0 {
		query.Limit = 100
	}
	if query.StartDate.IsZero() {
		query.StartDate = time.Now().Add(-as.config.DefaultTimeRange)
	}
	if query.EndDate.IsZero() {
		query.EndDate = time.Now()
	}

	engagementQuery := fmt.Sprintf(`
		SELECT
			recipient_id,
			COUNT(DISTINCT email_id) FILTER (WHERE event_type = 'open' AND is_unique) as opened,
			COUNT(DISTINCT email_id) FILTER (WHERE event_type = 'click' AND is_unique) as clicked,
			MAX(timestamp) FILTER (WHERE event_type = 'open') as last_open,
			MAX(timestamp) FILTER (WHERE event_type = 'click') as last_click
		FROM %sevents
		WHERE timestamp >= $1 AND timestamp <= $2
		GROUP BY recipient_id
		ORDER BY (COUNT(DISTINCT email_id) FILTER (WHERE event_type = 'open' AND is_unique) +
				  COUNT(DISTINCT email_id) FILTER (WHERE event_type = 'click' AND is_unique) * 2) DESC
		LIMIT $3
	`, as.config.TablePrefix)

	rows, err := as.db.QueryContext(ctx, engagementQuery, query.StartDate, query.EndDate, query.Limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query engagement: %w", err)
	}
	defer rows.Close()

	var result []*RecipientEngagement

	for rows.Next() {
		eng := &RecipientEngagement{}
		var lastOpen, lastClick sql.NullTime

		if err := rows.Scan(&eng.RecipientID, &eng.TotalOpened, &eng.TotalClicked, &lastOpen, &lastClick); err != nil {
			continue
		}

		if lastOpen.Valid {
			eng.LastOpenAt = &lastOpen.Time
		}
		if lastClick.Valid {
			eng.LastClickAt = &lastClick.Time
		}

		// Calculate engagement score (opens + 2*clicks)
		eng.EngagementScore = float64(eng.TotalOpened) + float64(eng.TotalClicked)*2

		// Check if active (engaged in last 30 days)
		thirtyDaysAgo := time.Now().Add(-30 * 24 * time.Hour)
		eng.IsActive = (lastOpen.Valid && lastOpen.Time.After(thirtyDaysAgo)) ||
			(lastClick.Valid && lastClick.Time.After(thirtyDaysAgo))

		result = append(result, eng)
	}

	return result, nil
}

// DeviceBreakdown contains device usage statistics
type DeviceBreakdown struct {
	DeviceType string  `json:"device_type"`
	Count      int64   `json:"count"`
	Percentage float64 `json:"percentage"`
}

// GetDeviceBreakdown retrieves device type breakdown for opens/clicks
func (as *AnalyticsService) GetDeviceBreakdown(ctx context.Context, query *AnalyticsQuery) ([]*DeviceBreakdown, error) {
	if query.StartDate.IsZero() {
		query.StartDate = time.Now().Add(-as.config.DefaultTimeRange)
	}
	if query.EndDate.IsZero() {
		query.EndDate = time.Now()
	}
	if query.EventType == "" {
		query.EventType = EventTypeOpen
	}

	conditions := []string{"timestamp >= $1", "timestamp <= $2", fmt.Sprintf("event_type = '%s'", query.EventType)}
	args := []interface{}{query.StartDate, query.EndDate}

	if query.CampaignID != "" {
		conditions = append(conditions, "campaign_id = $3")
		args = append(args, query.CampaignID)
	}

	whereClause := strings.Join(conditions, " AND ")

	deviceQuery := fmt.Sprintf(`
		SELECT
			COALESCE(NULLIF(device_type, ''), 'unknown') as device,
			COUNT(*) as count
		FROM %sevents
		WHERE %s
		GROUP BY device_type
		ORDER BY count DESC
	`, as.config.TablePrefix, whereClause)

	rows, err := as.db.QueryContext(ctx, deviceQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query device breakdown: %w", err)
	}
	defer rows.Close()

	var results []*DeviceBreakdown
	var total int64

	for rows.Next() {
		breakdown := &DeviceBreakdown{}
		if err := rows.Scan(&breakdown.DeviceType, &breakdown.Count); err != nil {
			continue
		}
		total += breakdown.Count
		results = append(results, breakdown)
	}

	// Calculate percentages
	for _, breakdown := range results {
		if total > 0 {
			breakdown.Percentage = float64(breakdown.Count) / float64(total) * 100
		}
	}

	return results, nil
}

// GeoBreakdown contains geographic distribution statistics
type GeoBreakdown struct {
	Country    string  `json:"country"`
	Count      int64   `json:"count"`
	Percentage float64 `json:"percentage"`
}

// GetGeoBreakdown retrieves geographic distribution of opens/clicks
func (as *AnalyticsService) GetGeoBreakdown(ctx context.Context, query *AnalyticsQuery) ([]*GeoBreakdown, error) {
	if query.StartDate.IsZero() {
		query.StartDate = time.Now().Add(-as.config.DefaultTimeRange)
	}
	if query.EndDate.IsZero() {
		query.EndDate = time.Now()
	}
	if query.EventType == "" {
		query.EventType = EventTypeOpen
	}

	conditions := []string{"timestamp >= $1", "timestamp <= $2", fmt.Sprintf("event_type = '%s'", query.EventType), "country IS NOT NULL", "country != ''"}
	args := []interface{}{query.StartDate, query.EndDate}

	if query.CampaignID != "" {
		conditions = append(conditions, "campaign_id = $3")
		args = append(args, query.CampaignID)
	}

	whereClause := strings.Join(conditions, " AND ")

	geoQuery := fmt.Sprintf(`
		SELECT
			country,
			COUNT(*) as count
		FROM %sevents
		WHERE %s
		GROUP BY country
		ORDER BY count DESC
		LIMIT 50
	`, as.config.TablePrefix, whereClause)

	rows, err := as.db.QueryContext(ctx, geoQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query geo breakdown: %w", err)
	}
	defer rows.Close()

	var results []*GeoBreakdown
	var total int64

	for rows.Next() {
		breakdown := &GeoBreakdown{}
		if err := rows.Scan(&breakdown.Country, &breakdown.Count); err != nil {
			continue
		}
		total += breakdown.Count
		results = append(results, breakdown)
	}

	// Calculate percentages
	for _, breakdown := range results {
		if total > 0 {
			breakdown.Percentage = float64(breakdown.Count) / float64(total) * 100
		}
	}

	return results, nil
}

// EmailPerformance contains detailed performance metrics for a single email
type EmailPerformance struct {
	EmailID        string             `json:"email_id"`
	Subject        string             `json:"subject"`
	RecipientEmail string             `json:"recipient_email"`
	CampaignID     string             `json:"campaign_id,omitempty"`
	SentAt         time.Time          `json:"sent_at"`
	OpenCount      int                `json:"open_count"`
	ClickCount     int                `json:"click_count"`
	FirstOpenAt    *time.Time         `json:"first_open_at,omitempty"`
	LastOpenAt     *time.Time         `json:"last_open_at,omitempty"`
	FirstClickAt   *time.Time         `json:"first_click_at,omitempty"`
	LastClickAt    *time.Time         `json:"last_click_at,omitempty"`
	Links          []*LinkPerformance `json:"links,omitempty"`
	EventTimeline  []*TimelineEvent   `json:"event_timeline,omitempty"`
}

// TimelineEvent represents an event in the email timeline
type TimelineEvent struct {
	Timestamp time.Time              `json:"timestamp"`
	EventType EventType              `json:"event_type"`
	Details   map[string]interface{} `json:"details,omitempty"`
}

// GetEmailPerformance retrieves detailed performance for a specific email
func (as *AnalyticsService) GetEmailPerformance(ctx context.Context, emailID string) (*EmailPerformance, error) {
	// Get email details
	email, err := as.storage.GetTrackedEmail(ctx, emailID)
	if err != nil {
		return nil, err
	}

	perf := &EmailPerformance{
		EmailID:        email.ID,
		Subject:        email.Subject,
		RecipientEmail: email.RecipientEmail,
		CampaignID:     email.CampaignID,
		SentAt:         email.SentAt,
		OpenCount:      email.OpenCount,
		ClickCount:     email.ClickCount,
		FirstOpenAt:    email.FirstOpenAt,
		LastOpenAt:     email.LastOpenAt,
		FirstClickAt:   email.FirstClickAt,
		LastClickAt:    email.LastClickAt,
	}

	// Get link performance
	links, err := as.storage.GetLinksByEmail(ctx, emailID)
	if err == nil {
		perf.Links = make([]*LinkPerformance, len(links))
		for i, link := range links {
			perf.Links[i] = &LinkPerformance{
				LinkID:       link.ID,
				OriginalURL:  link.OriginalURL,
				LinkText:     link.LinkText,
				Position:     link.Position,
				TotalClicks:  link.ClickCount,
				UniqueClicks: link.UniqueClicks,
			}
		}
	}

	// Get event timeline
	events, err := as.storage.GetEvents(ctx, EventFilter{
		EmailID: emailID,
		Limit:   100,
	})
	if err == nil {
		perf.EventTimeline = make([]*TimelineEvent, len(events))
		for i, event := range events {
			perf.EventTimeline[i] = &TimelineEvent{
				Timestamp: event.Timestamp,
				EventType: event.EventType,
				Details: map[string]interface{}{
					"device_type": event.DeviceType,
					"country":     event.Country,
					"link_url":    event.LinkURL,
				},
			}
		}
	}

	return perf, nil
}

// AggregateStats aggregates events into daily statistics
func (as *AnalyticsService) AggregateStats(ctx context.Context, date time.Time) error {
	startOfDay := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, date.Location())
	endOfDay := startOfDay.Add(24 * time.Hour)

	// Aggregate by campaign and event type
	aggQuery := fmt.Sprintf(`
		INSERT INTO %saggregates (id, date, campaign_id, event_type, total_count, unique_count, created_at, updated_at)
		SELECT
			gen_random_uuid(),
			$1::date,
			COALESCE(campaign_id, ''),
			event_type,
			COUNT(*),
			COUNT(CASE WHEN is_unique THEN 1 END),
			NOW(),
			NOW()
		FROM %sevents
		WHERE timestamp >= $1 AND timestamp < $2
		GROUP BY campaign_id, event_type
		ON CONFLICT (date, campaign_id, event_type)
		DO UPDATE SET
			total_count = EXCLUDED.total_count,
			unique_count = EXCLUDED.unique_count,
			updated_at = NOW()
	`, as.config.TablePrefix, as.config.TablePrefix)

	_, err := as.db.ExecContext(ctx, aggQuery, startOfDay, endOfDay)
	return err
}

// ExportAnalytics exports analytics data in JSON format
func (as *AnalyticsService) ExportAnalytics(ctx context.Context, query *AnalyticsQuery) ([]byte, error) {
	summary, err := as.GetDashboardSummary(ctx, query)
	if err != nil {
		return nil, err
	}

	timeSeries, err := as.GetTimeSeries(ctx, query)
	if err != nil {
		return nil, err
	}

	campaigns, err := as.GetCampaignPerformance(ctx, query)
	if err != nil {
		return nil, err
	}

	export := map[string]interface{}{
		"summary":     summary,
		"time_series": timeSeries,
		"campaigns":   campaigns,
		"exported_at": time.Now(),
		"query":       query,
	}

	return json.MarshalIndent(export, "", "  ")
}
