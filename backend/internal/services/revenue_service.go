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

type RevenueReportService interface {
	GenerateReport(ctx context.Context, query *models.RevenueReportQuery) (*models.RevenueReportResponse, error)
	GetTimeSeries(ctx context.Context, query *models.RevenueReportQuery) ([]models.RevenueDataPoint, error)
	GetCategoryBreakdown(ctx context.Context, query *models.RevenueReportQuery) ([]models.RevenueByCategoryData, error)
	GetTopCustomers(ctx context.Context, query *models.RevenueReportQuery, limit int) ([]models.CustomerRevenueData, error)
	ExportReport(ctx context.Context, request *models.ReportExportRequest) (*models.ReportExportResponse, error)
	RefreshCache(ctx context.Context, periodType models.PeriodType, periodStart, periodEnd time.Time) error
	GetCachedSummary(ctx context.Context, periodType models.PeriodType, periodStart, periodEnd time.Time) (*models.RevenueSummaryCache, error)
}

type revenueReportService struct {
	db                 *gorm.DB
	cacheExpiryMinutes int
}

func NewRevenueReportService(db *gorm.DB) RevenueReportService {
	return &revenueReportService{
		db:                 db,
		cacheExpiryMinutes: 30, // Cache expires after 30 minutes
	}
}

func (s *revenueReportService) GenerateReport(ctx context.Context, query *models.RevenueReportQuery) (*models.RevenueReportResponse, error) {
	if err := s.validateQuery(query); err != nil {
		return nil, fmt.Errorf("invalid query: %w", err)
	}

	s.setDefaultQueryParams(query)

	response := &models.RevenueReportResponse{}

	summary, err := s.calculateSummary(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate summary: %w", err)
	}
	response.Summary = *summary

	timeSeries, err := s.GetTimeSeries(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to get time series: %w", err)
	}
	response.TimeSeries = timeSeries

	categoryData, err := s.GetCategoryBreakdown(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to get category breakdown: %w", err)
	}
	response.ByCategory = categoryData

	topCustomers, err := s.GetTopCustomers(ctx, query, 10)
	if err != nil {
		return nil, fmt.Errorf("failed to get top customers: %w", err)
	}
	response.TopCustomers = topCustomers

	response.Metadata = models.RevenueReportMetadata{
		GeneratedAt:   time.Now(),
		Query:         *query,
		DataPoints:    len(timeSeries),
		CacheUsed:     false,
		ExecutionTime: 0,
	}

	return response, nil
}

func (s *revenueReportService) calculateSummary(ctx context.Context, query *models.RevenueReportQuery) (*models.RevenueSummary, error) {
	cached, err := s.GetCachedSummary(ctx, query.PeriodType, *query.StartDate, *query.EndDate)
	if err == nil && cached != nil && !cached.IsStale {
		return &models.RevenueSummary{
			TotalGrossRevenue:       cached.TotalGrossRevenue,
			TotalNetRevenue:         cached.TotalNetRevenue,
			TotalFees:               cached.TotalFees,
			TotalTax:                cached.TotalTaxes,
			TotalRefunds:            cached.TotalRefunds,
			TransactionCount:        cached.TransactionCount,
			RefundCount:             cached.RefundCount,
			AverageTransactionValue: cached.AverageTransactionValue,
			GrowthRate:              cached.GrowthRate,
		}, nil
	}

	var result struct {
		TotalGrossRevenue       decimal.Decimal
		TotalNetRevenue         decimal.Decimal
		TotalFees               decimal.Decimal
		TotalTax                decimal.Decimal
		TotalRefunds            decimal.Decimal
		TransactionCount        int64
		RefundCount             int64
		AverageTransactionValue decimal.Decimal
	}

	baseQuery := s.db.WithContext(ctx).Table("revenue_transactions").
		Select(`
			COALESCE(SUM(gross_amount), 0) as total_gross_revenue,
			COALESCE(SUM(net_amount), 0) as total_net_revenue,
			COALESCE(SUM(fee_amount), 0) as total_fees,
			COALESCE(SUM(tax_amount), 0) as total_tax,
			COALESCE(SUM(CASE WHEN transaction_type = 'refund' THEN ABS(net_amount) ELSE 0 END), 0) as total_refunds,
			COUNT(*) as transaction_count,
			COUNT(CASE WHEN transaction_type = 'refund' THEN 1 END) as refund_count,
			COALESCE(AVG(net_amount), 0) as average_transaction_value
		`)

	baseQuery = s.applyQueryFilters(baseQuery, query)

	if err := baseQuery.Scan(&result).Error; err != nil {
		return nil, fmt.Errorf("failed to calculate summary: %w", err)
	}

	growthRate, err := s.calculateGrowthRate(ctx, query)
	if err != nil {
		growthRate = decimal.Zero
	}

	summary := &models.RevenueSummary{
		TotalGrossRevenue:       result.TotalGrossRevenue,
		TotalNetRevenue:         result.TotalNetRevenue,
		TotalFees:               result.TotalFees,
		TotalTax:                result.TotalTax,
		TotalRefunds:            result.TotalRefunds,
		TransactionCount:        int(result.TransactionCount),
		RefundCount:             int(result.RefundCount),
		AverageTransactionValue: result.AverageTransactionValue,
		GrowthRate:              growthRate,
	}

	go s.updateCache(context.Background(), query, summary)

	return summary, nil
}

func (s *revenueReportService) calculateGrowthRate(ctx context.Context, query *models.RevenueReportQuery) (decimal.Decimal, error) {
	if query.StartDate == nil || query.EndDate == nil {
		return decimal.Zero, nil
	}

	duration := query.EndDate.Sub(*query.StartDate)
	previousStart := query.StartDate.Add(-duration)
	previousEnd := *query.StartDate

	var currentRevenue, previousRevenue decimal.Decimal

	err := s.db.WithContext(ctx).Table("revenue_transactions").
		Select("COALESCE(SUM(net_amount), 0)").
		Where("transaction_date >= ? AND transaction_date < ?", query.StartDate, query.EndDate).
		Scan(&currentRevenue).Error
	if err != nil {
		return decimal.Zero, err
	}

	err = s.db.WithContext(ctx).Table("revenue_transactions").
		Select("COALESCE(SUM(net_amount), 0)").
		Where("transaction_date >= ? AND transaction_date < ?", previousStart, previousEnd).
		Scan(&previousRevenue).Error
	if err != nil {
		return decimal.Zero, err
	}

	if previousRevenue.IsZero() {
		return decimal.Zero, nil
	}

	growth := currentRevenue.Sub(previousRevenue).Div(previousRevenue).Mul(decimal.NewFromInt(100))
	return growth, nil
}

func (s *revenueReportService) GetTimeSeries(ctx context.Context, query *models.RevenueReportQuery) ([]models.RevenueDataPoint, error) {
	var dataPoints []models.RevenueDataPoint

	var dateFormat string
	switch query.PeriodType {
	case models.PeriodTypeDaily:
		dateFormat = "DATE_TRUNC('day', transaction_date)"
	case models.PeriodTypeWeekly:
		dateFormat = "DATE_TRUNC('week', transaction_date)"
	case models.PeriodTypeMonthly:
		dateFormat = "DATE_TRUNC('month', transaction_date)"
	case models.PeriodTypeQuarterly:
		dateFormat = "DATE_TRUNC('quarter', transaction_date)"
	case models.PeriodTypeYearly:
		dateFormat = "DATE_TRUNC('year', transaction_date)"
	default:
		dateFormat = "DATE_TRUNC('day', transaction_date)"
	}

	baseQuery := s.db.WithContext(ctx).Table("revenue_transactions").
		Select(fmt.Sprintf(`
			%s as period,
			COALESCE(SUM(gross_amount), 0) as gross_revenue,
			COALESCE(SUM(net_amount), 0) as net_revenue,
			COALESCE(SUM(fee_amount), 0) as fees,
			COALESCE(SUM(tax_amount), 0) as tax,
			COUNT(*) as transaction_count,
			COALESCE(AVG(net_amount), 0) as average_transaction_value
		`, dateFormat))

	baseQuery = s.applyQueryFilters(baseQuery, query)
	baseQuery = baseQuery.Group("period").Order("period ASC")

	if err := baseQuery.Scan(&dataPoints).Error; err != nil {
		return nil, fmt.Errorf("failed to get time series: %w", err)
	}

	return dataPoints, nil
}

func (s *revenueReportService) GetCategoryBreakdown(ctx context.Context, query *models.RevenueReportQuery) ([]models.RevenueByCategoryData, error) {
	var categoryData []models.RevenueByCategoryData

	baseQuery := s.db.WithContext(ctx).Table("revenue_transactions").
		Select(`
			revenue_category as category,
			COALESCE(SUM(net_amount), 0) as revenue,
			COUNT(*) as transaction_count,
			COALESCE(AVG(net_amount), 0) as average_value
		`)

	baseQuery = s.applyQueryFilters(baseQuery, query)
	baseQuery = baseQuery.Group("revenue_category").Order("revenue DESC")

	var results []struct {
		Category         string
		Revenue          decimal.Decimal
		TransactionCount int64
		AverageValue     decimal.Decimal
	}

	if err := baseQuery.Scan(&results).Error; err != nil {
		return nil, fmt.Errorf("failed to get category breakdown: %w", err)
	}

	var totalRevenue decimal.Decimal
	for _, r := range results {
		totalRevenue = totalRevenue.Add(r.Revenue)
	}

	for _, r := range results {
		percentage := decimal.Zero
		if !totalRevenue.IsZero() {
			percentage = r.Revenue.Div(totalRevenue).Mul(decimal.NewFromInt(100))
		}

		categoryData = append(categoryData, models.RevenueByCategoryData{
			Category:         r.Category,
			Revenue:          r.Revenue,
			TransactionCount: int(r.TransactionCount),
			Percentage:       percentage,
			AverageValue:     r.AverageValue,
		})
	}

	return categoryData, nil
}

func (s *revenueReportService) GetTopCustomers(ctx context.Context, query *models.RevenueReportQuery, limit int) ([]models.CustomerRevenueData, error) {
	var customerData []models.CustomerRevenueData

	baseQuery := s.db.WithContext(ctx).Table("revenue_transactions").
		Select(`
			user_id,
			COALESCE(SUM(net_amount), 0) as total_revenue,
			COUNT(*) as transaction_count,
			COALESCE(AVG(net_amount), 0) as average_transaction_value,
			MIN(transaction_date) as first_transaction_date,
			MAX(transaction_date) as last_transaction_date
		`)

	baseQuery = s.applyQueryFilters(baseQuery, query)
	baseQuery = baseQuery.Group("user_id").Order("total_revenue DESC").Limit(limit)

	var results []struct {
		UserID                  uuid.UUID
		TotalRevenue            decimal.Decimal
		TransactionCount        int64
		AverageTransactionValue decimal.Decimal
		FirstTransactionDate    time.Time
		LastTransactionDate     time.Time
	}

	if err := baseQuery.Scan(&results).Error; err != nil {
		return nil, fmt.Errorf("failed to get top customers: %w", err)
	}

	for _, r := range results {
		customerData = append(customerData, models.CustomerRevenueData{
			UserID:                  r.UserID,
			TotalRevenue:            r.TotalRevenue,
			TransactionCount:        int(r.TransactionCount),
			AverageTransactionValue: r.AverageTransactionValue,
			FirstTransactionDate:    r.FirstTransactionDate,
			LastTransactionDate:     r.LastTransactionDate,
			LifetimeValue:           r.TotalRevenue,
		})
	}

	return customerData, nil
}

func (s *revenueReportService) ExportReport(ctx context.Context, request *models.ReportExportRequest) (*models.ReportExportResponse, error) {
	report, err := s.GenerateReport(ctx, &request.Query)
	if err != nil {
		return nil, fmt.Errorf("failed to generate report: %w", err)
	}

	exportID := uuid.New()
	fileName := fmt.Sprintf("revenue_report_%s.%s", time.Now().Format("20060102_150405"), request.Format)

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
		data, err := s.exportToCSV(report, request.IncludeSections)
		if err != nil {
			return nil, fmt.Errorf("failed to export to CSV: %w", err)
		}
		response.Data = data
		if csvStr, ok := data.(string); ok {
			response.FileSize = int64(len(csvStr))
		}

	case models.ExportFormatExcel:
		return nil, fmt.Errorf("excel export not yet implemented")

	case models.ExportFormatPDF:
		return nil, fmt.Errorf("PDF export not yet implemented")

	case models.ExportFormatJSON:
		response.Data = report
		response.FileSize = 0

	default:
		return nil, fmt.Errorf("unsupported export format: %s", request.Format)
	}

	return response, nil
}

func (s *revenueReportService) exportToCSV(report *models.RevenueReportResponse, includeSections []string) (interface{}, error) {
	csvContent := "Revenue Report\n\n"

	csvContent += "Summary\n"
	csvContent += fmt.Sprintf("Total Gross Revenue,%s\n", report.Summary.TotalGrossRevenue.String())
	csvContent += fmt.Sprintf("Total Net Revenue,%s\n", report.Summary.TotalNetRevenue.String())
	csvContent += fmt.Sprintf("Total Fees,%s\n", report.Summary.TotalFees.String())
	csvContent += fmt.Sprintf("Total Tax,%s\n", report.Summary.TotalTax.String())
	csvContent += fmt.Sprintf("Total Discounts,%s\n", report.Summary.TotalDiscounts.String())
	csvContent += fmt.Sprintf("Total Refunds,%s\n", report.Summary.TotalRefunds.String())
	csvContent += fmt.Sprintf("Transaction Count,%d\n", report.Summary.TransactionCount)
	csvContent += fmt.Sprintf("Average Transaction Value,%s\n", report.Summary.AverageTransactionValue.String())
	csvContent += fmt.Sprintf("Growth Rate,%s%%\n\n", report.Summary.GrowthRate.String())

	includeTimeSeries := len(includeSections) == 0
	for _, section := range includeSections {
		if section == "timeseries" {
			includeTimeSeries = true
			break
		}
	}

	if includeTimeSeries && len(report.TimeSeries) > 0 {
		csvContent += "Time Series\n"
		csvContent += "Period,Gross Revenue,Net Revenue,Fees,Tax,Transaction Count\n"
		for _, dp := range report.TimeSeries {
			csvContent += fmt.Sprintf("%s,%s,%s,%s,%s,%d\n",
				dp.Period.Format("2006-01-02"),
				dp.GrossRevenue.String(),
				dp.NetRevenue.String(),
				dp.Fees.String(),
				dp.Tax.String(),
				dp.TransactionCount,
			)
		}
		csvContent += "\n"
	}

	// Include category breakdown
	includeCategories := len(includeSections) == 0
	for _, section := range includeSections {
		if section == "categories" {
			includeCategories = true
			break
		}
	}

	if includeCategories && len(report.ByCategory) > 0 {
		csvContent += "Category Breakdown\n"
		csvContent += "Category,Revenue,Transaction Count,Average Value,Percentage\n"
		for _, cat := range report.ByCategory {
			csvContent += fmt.Sprintf("%s,%s,%d,%s,%s%%\n",
				cat.Category,
				cat.Revenue.String(),
				cat.TransactionCount,
				cat.AverageValue.String(),
				cat.Percentage.String(),
			)
		}
		csvContent += "\n"
	}

	return csvContent, nil
}

func (s *revenueReportService) RefreshCache(ctx context.Context, periodType models.PeriodType, periodStart, periodEnd time.Time) error {
	var cache models.RevenueSummaryCache

	cacheErr := s.db.WithContext(ctx).
		Where("period_type = ? AND period_start = ? AND period_end = ?", periodType, periodStart, periodEnd).
		First(&cache).Error

	if cacheErr != nil && cacheErr != gorm.ErrRecordNotFound {
		return fmt.Errorf("failed to check cache: %w", cacheErr)
	}

	isNewRecord := cacheErr == gorm.ErrRecordNotFound

	query := &models.RevenueReportQuery{
		StartDate:  &periodStart,
		EndDate:    &periodEnd,
		PeriodType: periodType,
	}

	summary, err := s.calculateSummary(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to calculate summary: %w", err)
	}

	if isNewRecord {
		cache = models.RevenueSummaryCache{
			ID:          uuid.New(),
			PeriodType:  periodType,
			PeriodStart: periodStart,
			PeriodEnd:   periodEnd,
		}
	}

	cache.TotalGrossRevenue = summary.TotalGrossRevenue
	cache.TotalNetRevenue = summary.TotalNetRevenue
	cache.TotalFees = summary.TotalFees
	cache.TotalTaxes = summary.TotalTax
	cache.TotalRefunds = summary.TotalRefunds
	cache.TransactionCount = summary.TransactionCount
	cache.RefundCount = summary.RefundCount
	cache.AverageTransactionValue = summary.AverageTransactionValue
	cache.GrowthRate = summary.GrowthRate
	cache.IsStale = false
	cache.LastUpdated = time.Now()

	if isNewRecord {
		if err := s.db.WithContext(ctx).Create(&cache).Error; err != nil {
			return fmt.Errorf("failed to create cache: %w", err)
		}
	} else {
		if err := s.db.WithContext(ctx).Save(&cache).Error; err != nil {
			return fmt.Errorf("failed to update cache: %w", err)
		}
	}

	return nil
}

func (s *revenueReportService) GetCachedSummary(ctx context.Context, periodType models.PeriodType, periodStart, periodEnd time.Time) (*models.RevenueSummaryCache, error) {
	var cache models.RevenueSummaryCache

	err := s.db.WithContext(ctx).
		Where("period_type = ? AND period_start = ? AND period_end = ?", periodType, periodStart, periodEnd).
		First(&cache).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get cache: %w", err)
	}

	cacheAge := time.Since(cache.LastUpdated)
	if cacheAge > time.Duration(s.cacheExpiryMinutes)*time.Minute {
		cache.IsStale = true
	}

	return &cache, nil
}

func (s *revenueReportService) updateCache(ctx context.Context, query *models.RevenueReportQuery, summary *models.RevenueSummary) {
	if query.StartDate == nil || query.EndDate == nil {
		return
	}

	_ = s.RefreshCache(ctx, query.PeriodType, *query.StartDate, *query.EndDate)
}

func (s *revenueReportService) applyQueryFilters(query *gorm.DB, reportQuery *models.RevenueReportQuery) *gorm.DB {
	if reportQuery.StartDate != nil {
		query = query.Where("transaction_date >= ?", reportQuery.StartDate)
	}

	if reportQuery.EndDate != nil {
		query = query.Where("transaction_date < ?", reportQuery.EndDate)
	}

	if reportQuery.Currency != "" {
		query = query.Where("currency = ?", reportQuery.Currency)
	}

	if reportQuery.Category != "" {
		query = query.Where("revenue_category = ?", reportQuery.Category)
	}

	if reportQuery.UserID != nil {
		query = query.Where("user_id = ?", reportQuery.UserID)
	}

	if reportQuery.TransactionType != nil {
		query = query.Where("transaction_type = ?", reportQuery.TransactionType)
	}

	if reportQuery.MinAmount != nil {
		query = query.Where("net_amount >= ?", reportQuery.MinAmount)
	}

	if reportQuery.MaxAmount != nil {
		query = query.Where("net_amount <= ?", reportQuery.MaxAmount)
	}

	return query
}

func (s *revenueReportService) validateQuery(query *models.RevenueReportQuery) error {
	if query.StartDate != nil && query.EndDate != nil {
		if query.EndDate.Before(*query.StartDate) {
			return fmt.Errorf("end date must be after start date")
		}
	}

	if query.MinAmount != nil && query.MaxAmount != nil {
		if query.MaxAmount.LessThan(*query.MinAmount) {
			return fmt.Errorf("max amount must be greater than min amount")
		}
	}

	return nil
}

func (s *revenueReportService) setDefaultQueryParams(query *models.RevenueReportQuery) {
	if query.StartDate == nil {
		start := time.Now().AddDate(0, -1, 0)
		query.StartDate = &start
	}

	if query.EndDate == nil {
		end := time.Now()
		query.EndDate = &end
	}

	if query.PeriodType == "" {
		query.PeriodType = models.PeriodTypeDaily
	}

	if query.Currency == "" {
		query.Currency = "USD"
	}
}
