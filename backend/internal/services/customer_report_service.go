package services

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"

	"github.com/javaknight1/servicepro/backend/internal/models"
)

// CustomerReportService defines the interface for customer reporting
type CustomerReportService interface {
	GenerateReport(ctx context.Context, query *models.CustomerReportQuery) (*models.CustomerReportResponse, error)
	GetSegmentBreakdown(ctx context.Context, query *models.CustomerReportQuery) ([]models.SegmentBreakdownData, error)
	GetTypeBreakdown(ctx context.Context, query *models.CustomerReportQuery) ([]models.TypeBreakdownData, error)
	GetGeographyBreakdown(ctx context.Context, query *models.CustomerReportQuery) ([]models.GeographyBreakdownData, error)
	GetAcquisitionTrend(ctx context.Context, query *models.CustomerReportQuery) ([]models.AcquisitionTrendData, error)
	GetTopCustomers(ctx context.Context, query *models.CustomerReportQuery, limit int) ([]models.CustomerMetricsData, error)
	GetAtRiskCustomers(ctx context.Context, limit int) ([]models.CustomerMetricsData, error)
	GetCohortData(ctx context.Context, months int) ([]models.CohortData, error)
	RefreshMetricsCache(ctx context.Context, customerID *uuid.UUID) error
	RecordActivity(ctx context.Context, activity *models.CustomerActivity) error
	ExportReport(ctx context.Context, request *models.CustomerReportExportRequest) (*models.ReportExportResponse, error)
}

type customerReportService struct {
	db *gorm.DB
}

// NewCustomerReportService creates a new customer report service
func NewCustomerReportService(db *gorm.DB) CustomerReportService {
	return &customerReportService{db: db}
}

func (s *customerReportService) GenerateReport(ctx context.Context, query *models.CustomerReportQuery) (*models.CustomerReportResponse, error) {
	if err := s.validateQuery(query); err != nil {
		return nil, fmt.Errorf("invalid query: %w", err)
	}

	s.setDefaultQueryParams(query)

	response := &models.CustomerReportResponse{}

	// Get summary
	summary, err := s.calculateSummary(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate summary: %w", err)
	}
	response.Summary = *summary

	// Get segment breakdown
	segmentData, err := s.GetSegmentBreakdown(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to get segment breakdown: %w", err)
	}
	response.SegmentBreakdown = segmentData

	// Get type breakdown
	typeData, err := s.GetTypeBreakdown(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to get type breakdown: %w", err)
	}
	response.TypeBreakdown = typeData

	// Get geography breakdown
	geoData, err := s.GetGeographyBreakdown(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to get geography breakdown: %w", err)
	}
	response.GeographyBreakdown = geoData

	// Get acquisition trend
	trendData, err := s.GetAcquisitionTrend(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to get acquisition trend: %w", err)
	}
	response.AcquisitionTrend = trendData

	// Get top customers
	topCustomers, err := s.GetTopCustomers(ctx, query, 10)
	if err != nil {
		return nil, fmt.Errorf("failed to get top customers: %w", err)
	}
	response.TopCustomers = topCustomers

	// Get at-risk customers
	atRiskCustomers, err := s.GetAtRiskCustomers(ctx, 10)
	if err != nil {
		return nil, fmt.Errorf("failed to get at-risk customers: %w", err)
	}
	response.AtRiskCustomers = atRiskCustomers

	response.Metadata = models.CustomerReportMetadata{
		GeneratedAt:  time.Now(),
		Query:        *query,
		TotalRecords: summary.TotalCustomers,
		CacheUsed:    false,
	}

	return response, nil
}

func (s *customerReportService) calculateSummary(ctx context.Context, query *models.CustomerReportQuery) (*models.CustomerReportSummary, error) {
	var summary models.CustomerReportSummary

	// Get total customers by status
	var statusCounts []struct {
		Status string
		Count  int
	}

	statusQuery := s.db.WithContext(ctx).Table("customers").
		Select("status, COUNT(*) as count").
		Where("deleted_at IS NULL").
		Group("status")

	statusQuery = s.applyBaseFilters(statusQuery, query)

	if err := statusQuery.Scan(&statusCounts).Error; err != nil {
		return nil, fmt.Errorf("failed to count by status: %w", err)
	}

	for _, sc := range statusCounts {
		summary.TotalCustomers += sc.Count
		switch sc.Status {
		case string(models.CustomerStatusActive):
			summary.ActiveCustomers = sc.Count
		case string(models.CustomerStatusInactive):
			summary.InactiveCustomers = sc.Count
		case string(models.CustomerStatusProspect):
			summary.Prospects = sc.Count
		}
	}

	// Get new customers this period
	newCustomersQuery := s.db.WithContext(ctx).Table("customers").
		Where("deleted_at IS NULL")

	if query.StartDate != nil {
		newCustomersQuery = newCustomersQuery.Where("created_at >= ?", query.StartDate)
	}
	if query.EndDate != nil {
		newCustomersQuery = newCustomersQuery.Where("created_at <= ?", query.EndDate)
	}

	var newCustomersCount int64
	if err := newCustomersQuery.Count(&newCustomersCount).Error; err != nil {
		return nil, fmt.Errorf("failed to count new customers: %w", err)
	}
	summary.NewCustomersThisPeriod = int(newCustomersCount)

	// Get revenue metrics from metrics cache
	var revenueMetrics struct {
		TotalRevenue decimal.Decimal
		AvgRevenue   decimal.Decimal
		AvgJobs      decimal.Decimal
		AvgQuoteRate decimal.Decimal
		ChurnedCount int64
	}

	metricsQuery := s.db.WithContext(ctx).Table("customer_metrics_cache cmc").
		Select(`
			COALESCE(SUM(cmc.total_revenue), 0) as total_revenue,
			COALESCE(AVG(cmc.total_revenue), 0) as avg_revenue,
			COALESCE(AVG(cmc.total_jobs), 0) as avg_jobs,
			COALESCE(AVG(cmc.quote_acceptance_rate), 0) as avg_quote_rate,
			COUNT(CASE WHEN cmc.segment = 'churned' THEN 1 END) as churned_count
		`).
		Joins("JOIN customers c ON c.id = cmc.customer_id").
		Where("c.deleted_at IS NULL")

	metricsQuery = s.applyMetricsFilters(metricsQuery, query)

	if err := metricsQuery.Scan(&revenueMetrics).Error; err != nil {
		// Metrics cache might not exist yet, use direct calculation
		summary.TotalRevenue = decimal.Zero
		summary.AverageRevenuePerCustomer = decimal.Zero
	} else {
		summary.TotalRevenue = revenueMetrics.TotalRevenue
		summary.AverageRevenuePerCustomer = revenueMetrics.AvgRevenue
		summary.AverageJobsPerCustomer = revenueMetrics.AvgJobs
		summary.OverallQuoteAcceptanceRate = revenueMetrics.AvgQuoteRate
		summary.ChurnedCustomers = int(revenueMetrics.ChurnedCount)
	}

	// Calculate growth rate (compare to previous period)
	if query.StartDate != nil && query.EndDate != nil {
		summary.GrowthRate = s.calculateGrowthRate(ctx, query)
	}

	// Calculate retention rate
	summary.RetentionRate = s.calculateRetentionRate(ctx, query)

	return &summary, nil
}

func (s *customerReportService) GetSegmentBreakdown(ctx context.Context, query *models.CustomerReportQuery) ([]models.SegmentBreakdownData, error) {
	var results []struct {
		Segment      string
		Count        int64
		TotalRevenue decimal.Decimal
	}

	segmentQuery := s.db.WithContext(ctx).Table("customer_metrics_cache cmc").
		Select(`
			cmc.segment,
			COUNT(*) as count,
			COALESCE(SUM(cmc.total_revenue), 0) as total_revenue
		`).
		Joins("JOIN customers c ON c.id = cmc.customer_id").
		Where("c.deleted_at IS NULL").
		Group("cmc.segment")

	segmentQuery = s.applyMetricsFilters(segmentQuery, query)

	if err := segmentQuery.Scan(&results).Error; err != nil {
		return nil, fmt.Errorf("failed to get segment breakdown: %w", err)
	}

	// Calculate total for percentages
	var totalCount int64
	for _, r := range results {
		totalCount += r.Count
	}

	var breakdown []models.SegmentBreakdownData
	for _, r := range results {
		segment := models.CustomerSegment(r.Segment)
		percentage := decimal.Zero
		avgRevenue := decimal.Zero

		if totalCount > 0 {
			percentage = decimal.NewFromInt(r.Count * 100).Div(decimal.NewFromInt(totalCount))
		}
		if r.Count > 0 {
			avgRevenue = r.TotalRevenue.Div(decimal.NewFromInt(r.Count))
		}

		breakdown = append(breakdown, models.SegmentBreakdownData{
			Segment:        segment,
			DisplayName:    models.GetSegmentDisplayName(segment),
			Count:          int(r.Count),
			Percentage:     percentage,
			TotalRevenue:   r.TotalRevenue,
			AverageRevenue: avgRevenue,
			Color:          models.GetSegmentColor(segment),
		})
	}

	return breakdown, nil
}

func (s *customerReportService) GetTypeBreakdown(ctx context.Context, query *models.CustomerReportQuery) ([]models.TypeBreakdownData, error) {
	var results []struct {
		CustomerType string
		Status       string
		Count        int64
		TotalRevenue decimal.Decimal
	}

	typeQuery := s.db.WithContext(ctx).Table("customers c").
		Select(`
			c.customer_type,
			c.status,
			COUNT(*) as count,
			COALESCE(SUM(cmc.total_revenue), 0) as total_revenue
		`).
		Joins("LEFT JOIN customer_metrics_cache cmc ON cmc.customer_id = c.id").
		Where("c.deleted_at IS NULL").
		Group("c.customer_type, c.status")

	typeQuery = s.applyBaseFilters(typeQuery, query)

	if err := typeQuery.Scan(&results).Error; err != nil {
		return nil, fmt.Errorf("failed to get type breakdown: %w", err)
	}

	// Calculate total for percentages
	var totalCount int64
	for _, r := range results {
		totalCount += r.Count
	}

	var breakdown []models.TypeBreakdownData
	for _, r := range results {
		percentage := decimal.Zero
		if totalCount > 0 {
			percentage = decimal.NewFromInt(r.Count * 100).Div(decimal.NewFromInt(totalCount))
		}

		breakdown = append(breakdown, models.TypeBreakdownData{
			CustomerType: models.CustomerType(r.CustomerType),
			Status:       models.CustomerStatus(r.Status),
			Count:        int(r.Count),
			Percentage:   percentage,
			TotalRevenue: r.TotalRevenue,
		})
	}

	return breakdown, nil
}

func (s *customerReportService) GetGeographyBreakdown(ctx context.Context, query *models.CustomerReportQuery) ([]models.GeographyBreakdownData, error) {
	var results []struct {
		State         string
		City          string
		CustomerCount int64
		ActiveCount   int64
		TotalRevenue  decimal.Decimal
	}

	geoQuery := s.db.WithContext(ctx).Table("customers c").
		Select(`
			c.billing_address_state as state,
			c.billing_address_city as city,
			COUNT(*) as customer_count,
			COUNT(CASE WHEN c.status = 'active' THEN 1 END) as active_count,
			COALESCE(SUM(cmc.total_revenue), 0) as total_revenue
		`).
		Joins("LEFT JOIN customer_metrics_cache cmc ON cmc.customer_id = c.id").
		Where("c.deleted_at IS NULL").
		Group("c.billing_address_state, c.billing_address_city").
		Order("customer_count DESC").
		Limit(50)

	geoQuery = s.applyBaseFilters(geoQuery, query)

	if err := geoQuery.Scan(&results).Error; err != nil {
		return nil, fmt.Errorf("failed to get geography breakdown: %w", err)
	}

	// Calculate total for percentages
	var totalCount int64
	for _, r := range results {
		totalCount += r.CustomerCount
	}

	var breakdown []models.GeographyBreakdownData
	for _, r := range results {
		percentage := decimal.Zero
		if totalCount > 0 {
			percentage = decimal.NewFromInt(r.CustomerCount * 100).Div(decimal.NewFromInt(totalCount))
		}

		breakdown = append(breakdown, models.GeographyBreakdownData{
			State:         r.State,
			City:          r.City,
			CustomerCount: int(r.CustomerCount),
			ActiveCount:   int(r.ActiveCount),
			Percentage:    percentage,
			TotalRevenue:  r.TotalRevenue,
		})
	}

	return breakdown, nil
}

func (s *customerReportService) GetAcquisitionTrend(ctx context.Context, query *models.CustomerReportQuery) ([]models.AcquisitionTrendData, error) {
	var results []struct {
		Period       time.Time
		NewCustomers int64
		Active       int64
		Prospects    int64
	}

	trendQuery := s.db.WithContext(ctx).Table("customers").
		Select(`
			DATE_TRUNC('month', created_at) as period,
			COUNT(*) as new_customers,
			COUNT(CASE WHEN status = 'active' THEN 1 END) as active,
			COUNT(CASE WHEN status = 'prospect' THEN 1 END) as prospects
		`).
		Where("deleted_at IS NULL").
		Group("DATE_TRUNC('month', created_at)").
		Order("period DESC").
		Limit(12)

	if query.StartDate != nil {
		trendQuery = trendQuery.Where("created_at >= ?", query.StartDate)
	}
	if query.EndDate != nil {
		trendQuery = trendQuery.Where("created_at <= ?", query.EndDate)
	}

	if err := trendQuery.Scan(&results).Error; err != nil {
		return nil, fmt.Errorf("failed to get acquisition trend: %w", err)
	}

	var trend []models.AcquisitionTrendData
	for i, r := range results {
		netGrowth := int(r.NewCustomers)
		if i < len(results)-1 {
			netGrowth = int(r.NewCustomers) - int(results[i+1].NewCustomers)
		}

		trend = append(trend, models.AcquisitionTrendData{
			Period:          r.Period,
			NewCustomers:    int(r.NewCustomers),
			ActiveCustomers: int(r.Active),
			Prospects:       int(r.Prospects),
			NetGrowth:       netGrowth,
		})
	}

	return trend, nil
}

func (s *customerReportService) GetTopCustomers(ctx context.Context, query *models.CustomerReportQuery, limit int) ([]models.CustomerMetricsData, error) {
	var results []models.CustomerMetricsData

	topQuery := s.db.WithContext(ctx).Table("customers c").
		Select(`
			c.id as customer_id,
			c.first_name,
			c.last_name,
			c.company_name,
			c.email,
			c.customer_type,
			c.status,
			c.billing_address_state as state,
			c.billing_address_city as city,
			COALESCE(cmc.segment, 'new') as segment,
			COALESCE(cmc.total_revenue, 0) as total_revenue,
			COALESCE(cmc.total_jobs, 0) as total_jobs,
			COALESCE(cmc.completed_jobs, 0) as completed_jobs,
			COALESCE(cmc.total_quotes, 0) as total_quotes,
			COALESCE(cmc.accepted_quotes, 0) as accepted_quotes,
			COALESCE(cmc.quote_acceptance_rate, 0) as quote_acceptance_rate,
			COALESCE(cmc.average_job_value, 0) as average_job_value,
			cmc.last_activity_date,
			COALESCE(cmc.days_since_last_activity, 0) as days_since_last_activity,
			COALESCE(cmc.customer_lifetime_days, 0) as customer_lifetime_days
		`).
		Joins("LEFT JOIN customer_metrics_cache cmc ON cmc.customer_id = c.id").
		Where("c.deleted_at IS NULL").
		Order("total_revenue DESC").
		Limit(limit)

	topQuery = s.applyBaseFilters(topQuery, query)

	if err := topQuery.Scan(&results).Error; err != nil {
		return nil, fmt.Errorf("failed to get top customers: %w", err)
	}

	return results, nil
}

func (s *customerReportService) GetAtRiskCustomers(ctx context.Context, limit int) ([]models.CustomerMetricsData, error) {
	var results []models.CustomerMetricsData

	atRiskQuery := s.db.WithContext(ctx).Table("customers c").
		Select(`
			c.id as customer_id,
			c.first_name,
			c.last_name,
			c.company_name,
			c.email,
			c.customer_type,
			c.status,
			c.billing_address_state as state,
			c.billing_address_city as city,
			cmc.segment,
			COALESCE(cmc.total_revenue, 0) as total_revenue,
			COALESCE(cmc.total_jobs, 0) as total_jobs,
			COALESCE(cmc.completed_jobs, 0) as completed_jobs,
			COALESCE(cmc.total_quotes, 0) as total_quotes,
			COALESCE(cmc.accepted_quotes, 0) as accepted_quotes,
			COALESCE(cmc.quote_acceptance_rate, 0) as quote_acceptance_rate,
			COALESCE(cmc.average_job_value, 0) as average_job_value,
			cmc.last_activity_date,
			COALESCE(cmc.days_since_last_activity, 0) as days_since_last_activity,
			COALESCE(cmc.customer_lifetime_days, 0) as customer_lifetime_days
		`).
		Joins("JOIN customer_metrics_cache cmc ON cmc.customer_id = c.id").
		Where("c.deleted_at IS NULL").
		Where("cmc.segment = ?", models.CustomerSegmentAtRisk).
		Order("cmc.total_revenue DESC").
		Limit(limit)

	if err := atRiskQuery.Scan(&results).Error; err != nil {
		return nil, fmt.Errorf("failed to get at-risk customers: %w", err)
	}

	return results, nil
}

func (s *customerReportService) GetCohortData(ctx context.Context, months int) ([]models.CohortData, error) {
	var results []struct {
		CohortMonth    time.Time
		CohortSize     int64
		Month1Retained int64
		Month2Retained int64
		Month3Retained int64
		Month6Retained int64
	}

	cohortQuery := s.db.WithContext(ctx).Raw(`
		SELECT * FROM v_customer_cohorts
		ORDER BY cohort_month DESC
		LIMIT ?
	`, months)

	if err := cohortQuery.Scan(&results).Error; err != nil {
		return nil, fmt.Errorf("failed to get cohort data: %w", err)
	}

	var cohorts []models.CohortData
	for _, r := range results {
		retention1M := decimal.Zero
		retention3M := decimal.Zero
		retention6M := decimal.Zero

		if r.CohortSize > 0 {
			retention1M = decimal.NewFromInt(r.Month1Retained * 100).Div(decimal.NewFromInt(r.CohortSize))
			retention3M = decimal.NewFromInt(r.Month3Retained * 100).Div(decimal.NewFromInt(r.CohortSize))
			retention6M = decimal.NewFromInt(r.Month6Retained * 100).Div(decimal.NewFromInt(r.CohortSize))
		}

		cohorts = append(cohorts, models.CohortData{
			CohortMonth:     r.CohortMonth,
			CohortSize:      int(r.CohortSize),
			Month1Retained:  int(r.Month1Retained),
			Month2Retained:  int(r.Month2Retained),
			Month3Retained:  int(r.Month3Retained),
			Month6Retained:  int(r.Month6Retained),
			RetentionRate1M: retention1M,
			RetentionRate3M: retention3M,
			RetentionRate6M: retention6M,
		})
	}

	return cohorts, nil
}

func (s *customerReportService) RefreshMetricsCache(ctx context.Context, customerID *uuid.UUID) error {
	var query string
	var args []any

	if customerID != nil {
		query = "SELECT refresh_customer_metrics_cache($1)"
		args = append(args, *customerID)
	} else {
		query = "SELECT refresh_customer_metrics_cache(NULL)"
	}

	if err := s.db.WithContext(ctx).Exec(query, args...).Error; err != nil {
		return fmt.Errorf("failed to refresh metrics cache: %w", err)
	}

	return nil
}

func (s *customerReportService) RecordActivity(ctx context.Context, activity *models.CustomerActivity) error {
	if activity.ID == uuid.Nil {
		activity.ID = uuid.New()
	}
	if activity.ActivityDate.IsZero() {
		activity.ActivityDate = time.Now()
	}

	if err := s.db.WithContext(ctx).Create(activity).Error; err != nil {
		return fmt.Errorf("failed to record activity: %w", err)
	}

	// Mark metrics cache as stale
	s.db.WithContext(ctx).Model(&models.CustomerMetricsCache{}).
		Where("customer_id = ?", activity.CustomerID).
		Update("is_stale", true)

	return nil
}

func (s *customerReportService) ExportReport(ctx context.Context, request *models.CustomerReportExportRequest) (*models.ReportExportResponse, error) {
	report, err := s.GenerateReport(ctx, &request.Query)
	if err != nil {
		return nil, fmt.Errorf("failed to generate report: %w", err)
	}

	exportID := uuid.New()
	fileName := fmt.Sprintf("customer_report_%s.%s", time.Now().Format("20060102_150405"), request.Format)

	response := &models.ReportExportResponse{
		ExportID:    exportID,
		FileName:    fileName,
		Format:      request.Format,
		Status:      "completed",
		GeneratedAt: time.Now(),
		ExpiresAt:   time.Now().Add(24 * time.Hour),
	}

	switch request.Format {
	case models.ExportFormatCSV:
		data := s.exportToCSV(report, request.IncludeSections)
		response.Data = data
		response.FileSize = int64(len(data))

	case models.ExportFormatJSON:
		response.Data = report
		response.FileSize = 0

	default:
		return nil, fmt.Errorf("unsupported export format: %s", request.Format)
	}

	return response, nil
}

func (s *customerReportService) exportToCSV(report *models.CustomerReportResponse, includeSections []string) string {
	csvContent := "Customer Report\n\n"

	// Summary section
	csvContent += "Summary\n"
	csvContent += fmt.Sprintf("Total Customers,%d\n", report.Summary.TotalCustomers)
	csvContent += fmt.Sprintf("Active Customers,%d\n", report.Summary.ActiveCustomers)
	csvContent += fmt.Sprintf("Inactive Customers,%d\n", report.Summary.InactiveCustomers)
	csvContent += fmt.Sprintf("Prospects,%d\n", report.Summary.Prospects)
	csvContent += fmt.Sprintf("New Customers This Period,%d\n", report.Summary.NewCustomersThisPeriod)
	csvContent += fmt.Sprintf("Total Revenue,%s\n", report.Summary.TotalRevenue.String())
	csvContent += fmt.Sprintf("Average Revenue Per Customer,%s\n", report.Summary.AverageRevenuePerCustomer.String())
	csvContent += fmt.Sprintf("Growth Rate,%s%%\n\n", report.Summary.GrowthRate.String())

	// Segment breakdown
	includeSegments := len(includeSections) == 0
	for _, section := range includeSections {
		if section == "segments" {
			includeSegments = true
			break
		}
	}

	if includeSegments && len(report.SegmentBreakdown) > 0 {
		csvContent += "Segment Breakdown\n"
		csvContent += "Segment,Count,Percentage,Total Revenue,Average Revenue\n"
		for _, seg := range report.SegmentBreakdown {
			csvContent += fmt.Sprintf("%s,%d,%s%%,%s,%s\n",
				seg.DisplayName,
				seg.Count,
				seg.Percentage.String(),
				seg.TotalRevenue.String(),
				seg.AverageRevenue.String(),
			)
		}
		csvContent += "\n"
	}

	// Top customers
	includeTopCustomers := len(includeSections) == 0
	for _, section := range includeSections {
		if section == "top_customers" {
			includeTopCustomers = true
			break
		}
	}

	if includeTopCustomers && len(report.TopCustomers) > 0 {
		csvContent += "Top Customers\n"
		csvContent += "Name,Email,Type,Segment,Total Revenue,Total Jobs\n"
		for _, cust := range report.TopCustomers {
			name := cust.FirstName + " " + cust.LastName
			if cust.CompanyName != nil {
				name = *cust.CompanyName
			}
			csvContent += fmt.Sprintf("%s,%s,%s,%s,%s,%d\n",
				name,
				cust.Email,
				cust.CustomerType,
				cust.Segment,
				cust.TotalRevenue.String(),
				cust.TotalJobs,
			)
		}
		csvContent += "\n"
	}

	return csvContent
}

func (s *customerReportService) calculateGrowthRate(ctx context.Context, query *models.CustomerReportQuery) decimal.Decimal {
	if query.StartDate == nil || query.EndDate == nil {
		return decimal.Zero
	}

	duration := query.EndDate.Sub(*query.StartDate)
	previousStart := query.StartDate.Add(-duration)
	previousEnd := *query.StartDate

	var currentCount, previousCount int64

	s.db.WithContext(ctx).Table("customers").
		Where("deleted_at IS NULL").
		Where("created_at >= ? AND created_at < ?", query.StartDate, query.EndDate).
		Count(&currentCount)

	s.db.WithContext(ctx).Table("customers").
		Where("deleted_at IS NULL").
		Where("created_at >= ? AND created_at < ?", previousStart, previousEnd).
		Count(&previousCount)

	if previousCount == 0 {
		return decimal.Zero
	}

	growth := decimal.NewFromInt(currentCount - previousCount).
		Div(decimal.NewFromInt(previousCount)).
		Mul(decimal.NewFromInt(100))

	return growth
}

func (s *customerReportService) calculateRetentionRate(ctx context.Context, query *models.CustomerReportQuery) decimal.Decimal {
	// Simple retention: active customers / (total customers - new customers)
	var activeCount, totalCount, newCount int64

	baseQuery := s.db.WithContext(ctx).Table("customers").Where("deleted_at IS NULL")

	baseQuery.Count(&totalCount)
	baseQuery.Where("status = ?", models.CustomerStatusActive).Count(&activeCount)

	if query.StartDate != nil {
		s.db.WithContext(ctx).Table("customers").
			Where("deleted_at IS NULL").
			Where("created_at >= ?", query.StartDate).
			Count(&newCount)
	}

	existingCustomers := totalCount - newCount
	if existingCustomers <= 0 {
		return decimal.NewFromInt(100)
	}

	retention := decimal.NewFromInt(activeCount).
		Div(decimal.NewFromInt(existingCustomers)).
		Mul(decimal.NewFromInt(100))

	return retention
}

func (s *customerReportService) applyBaseFilters(query *gorm.DB, reportQuery *models.CustomerReportQuery) *gorm.DB {
	if reportQuery.CustomerType != "" {
		query = query.Where("c.customer_type = ?", reportQuery.CustomerType)
	}

	if reportQuery.Status != "" {
		query = query.Where("c.status = ?", reportQuery.Status)
	}

	if reportQuery.State != "" {
		query = query.Where("c.billing_address_state = ?", reportQuery.State)
	}

	if reportQuery.City != "" {
		query = query.Where("LOWER(c.billing_address_city) = LOWER(?)", reportQuery.City)
	}

	return query
}

func (s *customerReportService) applyMetricsFilters(query *gorm.DB, reportQuery *models.CustomerReportQuery) *gorm.DB {
	query = s.applyBaseFilters(query, reportQuery)

	if reportQuery.Segment != "" {
		query = query.Where("cmc.segment = ?", reportQuery.Segment)
	}

	if reportQuery.MinRevenue != nil {
		query = query.Where("cmc.total_revenue >= ?", reportQuery.MinRevenue)
	}

	if reportQuery.MaxRevenue != nil {
		query = query.Where("cmc.total_revenue <= ?", reportQuery.MaxRevenue)
	}

	if reportQuery.MinJobs != nil {
		query = query.Where("cmc.total_jobs >= ?", reportQuery.MinJobs)
	}

	if reportQuery.MaxJobs != nil {
		query = query.Where("cmc.total_jobs <= ?", reportQuery.MaxJobs)
	}

	return query
}

func (s *customerReportService) validateQuery(query *models.CustomerReportQuery) error {
	if query.StartDate != nil && query.EndDate != nil {
		if query.EndDate.Before(*query.StartDate) {
			return fmt.Errorf("end date must be after start date")
		}
	}

	if query.MinRevenue != nil && query.MaxRevenue != nil {
		if query.MaxRevenue.LessThan(*query.MinRevenue) {
			return fmt.Errorf("max revenue must be greater than min revenue")
		}
	}

	if query.MinJobs != nil && query.MaxJobs != nil {
		if *query.MaxJobs < *query.MinJobs {
			return fmt.Errorf("max jobs must be greater than min jobs")
		}
	}

	return nil
}

func (s *customerReportService) setDefaultQueryParams(query *models.CustomerReportQuery) {
	if query.Page < 1 {
		query.Page = 1
	}

	if query.PageSize < 1 {
		query.PageSize = 50
	}

	if query.PageSize > 100 {
		query.PageSize = 100
	}

	if query.SortBy == "" {
		query.SortBy = "created_at"
	}

	if query.SortOrder == "" {
		query.SortOrder = "desc"
	}
}
