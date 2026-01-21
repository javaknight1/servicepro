package services

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/javaknight1/servicepro/backend/internal/models"
)

func setupRevenueTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)

	// Create the revenue_transactions table
	err = db.Exec(`
		CREATE TABLE revenue_transactions (
			id TEXT PRIMARY KEY,
			payment_id TEXT NOT NULL,
			user_id TEXT NOT NULL,
			invoice_id TEXT,
			order_id TEXT,
			transaction_type TEXT NOT NULL,
			transaction_date DATETIME NOT NULL,
			gross_amount DECIMAL(12,2) NOT NULL,
			net_amount DECIMAL(12,2) NOT NULL,
			fee_amount DECIMAL(12,2) NOT NULL DEFAULT 0,
			tax_amount DECIMAL(12,2) NOT NULL DEFAULT 0,
			discount_amount DECIMAL(12,2) NOT NULL DEFAULT 0,
			refund_amount DECIMAL(12,2) NOT NULL DEFAULT 0,
			currency TEXT NOT NULL DEFAULT 'USD',
			exchange_rate DECIMAL(10,4) DEFAULT 1.0000,
			base_currency_amount DECIMAL(12,2),
			revenue_category TEXT,
			revenue_subcategory TEXT,
			tags TEXT,
			status TEXT NOT NULL DEFAULT 'completed',
			is_refunded INTEGER NOT NULL DEFAULT 0,
			is_disputed INTEGER NOT NULL DEFAULT 0,
			metadata TEXT,
			created_at DATETIME,
			updated_at DATETIME
		)
	`).Error
	require.NoError(t, err)

	// Create the revenue_summary_cache table
	err = db.Exec(`
		CREATE TABLE revenue_summary_cache (
			id TEXT PRIMARY KEY,
			period_type TEXT NOT NULL,
			period_start DATETIME NOT NULL,
			period_end DATETIME NOT NULL,
			total_gross_revenue DECIMAL(12,2) NOT NULL DEFAULT 0,
			total_net_revenue DECIMAL(12,2) NOT NULL DEFAULT 0,
			total_fees DECIMAL(12,2) NOT NULL DEFAULT 0,
			total_taxes DECIMAL(12,2) NOT NULL DEFAULT 0,
			total_discounts DECIMAL(12,2) NOT NULL DEFAULT 0,
			total_refunds DECIMAL(12,2) NOT NULL DEFAULT 0,
			transaction_count INTEGER NOT NULL DEFAULT 0,
			payment_count INTEGER NOT NULL DEFAULT 0,
			refund_count INTEGER NOT NULL DEFAULT 0,
			unique_customer_count INTEGER NOT NULL DEFAULT 0,
			average_transaction_value DECIMAL(12,2),
			average_order_value DECIMAL(12,2),
			growth_rate DECIMAL(5,2),
			previous_period_revenue DECIMAL(12,2),
			currency TEXT DEFAULT 'USD',
			metadata TEXT,
			last_updated DATETIME NOT NULL,
			is_stale INTEGER NOT NULL DEFAULT 0
		)
	`).Error
	require.NoError(t, err)

	return db
}

func createTestTransactions(t *testing.T, db *gorm.DB) []models.RevenueTransaction {
	transactions := []models.RevenueTransaction{
		{
			ID:              uuid.New(),
			PaymentID:       uuid.New(),
			UserID:          uuid.MustParse("11111111-1111-1111-1111-111111111111"),
			TransactionType: models.TransactionTypePayment,
			TransactionDate: time.Now().AddDate(0, 0, -10),
			GrossAmount:     decimal.NewFromFloat(100.00),
			NetAmount:       decimal.NewFromFloat(97.00),
			FeeAmount:       decimal.NewFromFloat(3.00),
			TaxAmount:       decimal.NewFromFloat(8.00),
			Currency:        "USD",
			RevenueCategory: "services",
			Status:          models.TransactionStatusCompleted,
		},
		{
			ID:              uuid.New(),
			PaymentID:       uuid.New(),
			UserID:          uuid.MustParse("11111111-1111-1111-1111-111111111111"),
			TransactionType: models.TransactionTypePayment,
			TransactionDate: time.Now().AddDate(0, 0, -5),
			GrossAmount:     decimal.NewFromFloat(250.00),
			NetAmount:       decimal.NewFromFloat(242.50),
			FeeAmount:       decimal.NewFromFloat(7.50),
			TaxAmount:       decimal.NewFromFloat(20.00),
			Currency:        "USD",
			RevenueCategory: "products",
			Status:          models.TransactionStatusCompleted,
		},
		{
			ID:              uuid.New(),
			PaymentID:       uuid.New(),
			UserID:          uuid.MustParse("22222222-2222-2222-2222-222222222222"),
			TransactionType: models.TransactionTypePayment,
			TransactionDate: time.Now().AddDate(0, 0, -3),
			GrossAmount:     decimal.NewFromFloat(75.00),
			NetAmount:       decimal.NewFromFloat(72.75),
			FeeAmount:       decimal.NewFromFloat(2.25),
			TaxAmount:       decimal.NewFromFloat(6.00),
			Currency:        "USD",
			RevenueCategory: "services",
			Status:          models.TransactionStatusCompleted,
		},
		{
			ID:              uuid.New(),
			PaymentID:       uuid.New(),
			UserID:          uuid.MustParse("33333333-3333-3333-3333-333333333333"),
			TransactionType: models.TransactionTypeRefund,
			TransactionDate: time.Now().AddDate(0, 0, -1),
			GrossAmount:     decimal.NewFromFloat(-50.00),
			NetAmount:       decimal.NewFromFloat(-50.00),
			FeeAmount:       decimal.NewFromFloat(0),
			TaxAmount:       decimal.NewFromFloat(0),
			Currency:        "USD",
			RevenueCategory: "services",
			Status:          models.TransactionStatusCompleted,
		},
	}

	for _, tx := range transactions {
		err := db.Create(&tx).Error
		require.NoError(t, err)
	}

	return transactions
}

func TestRevenueService_GenerateReport(t *testing.T) {
	db := setupRevenueTestDB(t)
	createTestTransactions(t, db)

	service := NewRevenueReportService(db)

	startDate := time.Now().AddDate(0, 0, -30)
	endDate := time.Now().AddDate(0, 0, 1)

	query := &models.RevenueReportQuery{
		StartDate:  &startDate,
		EndDate:    &endDate,
		PeriodType: models.PeriodTypeDaily,
		Currency:   "USD",
	}

	report, err := service.GenerateReport(context.Background(), query)

	require.NoError(t, err)
	assert.NotNil(t, report)
	assert.NotNil(t, report.Summary)
	assert.NotEmpty(t, report.TimeSeries)
}

func TestRevenueService_CalculateSummary(t *testing.T) {
	db := setupRevenueTestDB(t)
	createTestTransactions(t, db)

	service := NewRevenueReportService(db).(*revenueReportService)

	startDate := time.Now().AddDate(0, 0, -30)
	endDate := time.Now().AddDate(0, 0, 1)

	query := &models.RevenueReportQuery{
		StartDate:  &startDate,
		EndDate:    &endDate,
		PeriodType: models.PeriodTypeDaily,
		Currency:   "USD",
	}

	summary, err := service.calculateSummary(context.Background(), query)

	require.NoError(t, err)
	assert.NotNil(t, summary)

	// Total gross revenue: 100 + 250 + 75 - 50 = 375
	expectedGross := decimal.NewFromFloat(375.00)
	assert.True(t, summary.TotalGrossRevenue.Equal(expectedGross),
		"Expected gross revenue %s, got %s", expectedGross.String(), summary.TotalGrossRevenue.String())

	// Total fees: 3.00 + 7.50 + 2.25 = 12.75
	expectedFees := decimal.NewFromFloat(12.75)
	assert.True(t, summary.TotalFees.Equal(expectedFees),
		"Expected fees %s, got %s", expectedFees.String(), summary.TotalFees.String())

	// Transaction count should be 4
	assert.Equal(t, 4, summary.TransactionCount)

	// Refund count should be 1
	assert.Equal(t, 1, summary.RefundCount)
}

func TestRevenueService_GetTimeSeries_Daily(t *testing.T) {
	db := setupRevenueTestDB(t)
	createTestTransactions(t, db)

	service := NewRevenueReportService(db)

	startDate := time.Now().AddDate(0, 0, -30)
	endDate := time.Now().AddDate(0, 0, 1)

	query := &models.RevenueReportQuery{
		StartDate:  &startDate,
		EndDate:    &endDate,
		PeriodType: models.PeriodTypeDaily,
		Currency:   "USD",
	}

	timeSeries, err := service.GetTimeSeries(context.Background(), query)

	require.NoError(t, err)
	assert.NotEmpty(t, timeSeries)

	// Should have data points for different days
	assert.GreaterOrEqual(t, len(timeSeries), 1)
}

func TestRevenueService_GetCategoryBreakdown(t *testing.T) {
	db := setupRevenueTestDB(t)
	createTestTransactions(t, db)

	service := NewRevenueReportService(db)

	startDate := time.Now().AddDate(0, 0, -30)
	endDate := time.Now().AddDate(0, 0, 1)

	query := &models.RevenueReportQuery{
		StartDate:  &startDate,
		EndDate:    &endDate,
		PeriodType: models.PeriodTypeDaily,
		Currency:   "USD",
	}

	categories, err := service.GetCategoryBreakdown(context.Background(), query)

	require.NoError(t, err)
	assert.NotEmpty(t, categories)

	// Should have "services" and "products" categories
	categoryMap := make(map[string]models.RevenueByCategoryData)
	for _, cat := range categories {
		categoryMap[cat.Category] = cat
	}

	assert.Contains(t, categoryMap, "services")
	assert.Contains(t, categoryMap, "products")

	// Services revenue: 97 + 72.75 - 50 = 119.75
	servicesData := categoryMap["services"]
	assert.Equal(t, 3, servicesData.TransactionCount)

	// Products revenue: 242.50
	productsData := categoryMap["products"]
	assert.Equal(t, 1, productsData.TransactionCount)
}

func TestRevenueService_GetTopCustomers(t *testing.T) {
	db := setupRevenueTestDB(t)
	createTestTransactions(t, db)

	service := NewRevenueReportService(db)

	startDate := time.Now().AddDate(0, 0, -30)
	endDate := time.Now().AddDate(0, 0, 1)

	query := &models.RevenueReportQuery{
		StartDate:  &startDate,
		EndDate:    &endDate,
		PeriodType: models.PeriodTypeDaily,
		Currency:   "USD",
	}

	customers, err := service.GetTopCustomers(context.Background(), query, 10)

	require.NoError(t, err)
	assert.NotEmpty(t, customers)

	// Customer 1 (user 11111111) should be first with highest revenue (97 + 242.50 = 339.50)
	assert.Equal(t, uuid.MustParse("11111111-1111-1111-1111-111111111111"), customers[0].UserID)
	assert.Equal(t, 2, customers[0].TransactionCount)
}

func TestRevenueService_QueryFilters(t *testing.T) {
	db := setupRevenueTestDB(t)
	createTestTransactions(t, db)

	service := NewRevenueReportService(db)

	startDate := time.Now().AddDate(0, 0, -30)
	endDate := time.Now().AddDate(0, 0, 1)

	t.Run("filter by category", func(t *testing.T) {
		query := &models.RevenueReportQuery{
			StartDate:  &startDate,
			EndDate:    &endDate,
			PeriodType: models.PeriodTypeDaily,
			Category:   "services",
		}

		report, err := service.GenerateReport(context.Background(), query)

		require.NoError(t, err)
		assert.NotNil(t, report)
		// Should only have service transactions (3 total)
		assert.Equal(t, 3, report.Summary.TransactionCount)
	})

	t.Run("filter by user", func(t *testing.T) {
		userID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
		query := &models.RevenueReportQuery{
			StartDate:  &startDate,
			EndDate:    &endDate,
			PeriodType: models.PeriodTypeDaily,
			UserID:     &userID,
		}

		report, err := service.GenerateReport(context.Background(), query)

		require.NoError(t, err)
		assert.NotNil(t, report)
		// User 1 has 2 transactions
		assert.Equal(t, 2, report.Summary.TransactionCount)
	})

	t.Run("filter by date range", func(t *testing.T) {
		recentStart := time.Now().AddDate(0, 0, -7)
		query := &models.RevenueReportQuery{
			StartDate:  &recentStart,
			EndDate:    &endDate,
			PeriodType: models.PeriodTypeDaily,
		}

		report, err := service.GenerateReport(context.Background(), query)

		require.NoError(t, err)
		assert.NotNil(t, report)
		// Should have transactions from last 7 days (3 transactions)
		assert.Equal(t, 3, report.Summary.TransactionCount)
	})
}

func TestRevenueService_GrowthRateCalculation(t *testing.T) {
	db := setupRevenueTestDB(t)

	// Create transactions in two periods
	user1 := uuid.New()

	// Previous period transactions (30-60 days ago)
	for i := 0; i < 3; i++ {
		tx := models.RevenueTransaction{
			ID:              uuid.New(),
			PaymentID:       uuid.New(),
			UserID:          user1,
			TransactionType: models.TransactionTypePayment,
			TransactionDate: time.Now().AddDate(0, 0, -45+i),
			GrossAmount:     decimal.NewFromFloat(100.00),
			NetAmount:       decimal.NewFromFloat(95.00),
			FeeAmount:       decimal.NewFromFloat(5.00),
			Currency:        "USD",
			Status:          models.TransactionStatusCompleted,
		}
		err := db.Create(&tx).Error
		require.NoError(t, err)
	}

	// Current period transactions (last 30 days)
	for i := 0; i < 5; i++ {
		tx := models.RevenueTransaction{
			ID:              uuid.New(),
			PaymentID:       uuid.New(),
			UserID:          user1,
			TransactionType: models.TransactionTypePayment,
			TransactionDate: time.Now().AddDate(0, 0, -15+i),
			GrossAmount:     decimal.NewFromFloat(100.00),
			NetAmount:       decimal.NewFromFloat(95.00),
			FeeAmount:       decimal.NewFromFloat(5.00),
			Currency:        "USD",
			Status:          models.TransactionStatusCompleted,
		}
		err := db.Create(&tx).Error
		require.NoError(t, err)
	}

	service := NewRevenueReportService(db).(*revenueReportService)

	startDate := time.Now().AddDate(0, 0, -30)
	endDate := time.Now()

	query := &models.RevenueReportQuery{
		StartDate:  &startDate,
		EndDate:    &endDate,
		PeriodType: models.PeriodTypeDaily,
	}

	growthRate, err := service.calculateGrowthRate(context.Background(), query)

	require.NoError(t, err)
	// Current: 5 * 95 = 475, Previous: 3 * 95 = 285
	// Growth: (475 - 285) / 285 * 100 = 66.67%
	expectedGrowth := decimal.NewFromFloat(66.67)
	assert.True(t, growthRate.Sub(expectedGrowth).Abs().LessThan(decimal.NewFromFloat(1)),
		"Expected growth rate around 66.67%%, got %s", growthRate.String())
}

func TestRevenueService_ExportCSV(t *testing.T) {
	db := setupRevenueTestDB(t)
	createTestTransactions(t, db)

	service := NewRevenueReportService(db)

	startDate := time.Now().AddDate(0, 0, -30)
	endDate := time.Now().AddDate(0, 0, 1)

	request := &models.ReportExportRequest{
		Query: models.RevenueReportQuery{
			StartDate:  &startDate,
			EndDate:    &endDate,
			PeriodType: models.PeriodTypeDaily,
		},
		Format:          models.ExportFormatCSV,
		IncludeSections: []string{"summary", "timeseries"},
	}

	response, err := service.ExportReport(context.Background(), request)

	require.NoError(t, err)
	assert.NotNil(t, response)
	assert.Equal(t, models.ExportFormatCSV, response.Format)
	assert.Equal(t, "completed", response.Status)

	// Check CSV data is present
	csvData, ok := response.Data.(string)
	require.True(t, ok, "Expected CSV data to be a string")
	assert.Contains(t, csvData, "Revenue Report")
	assert.Contains(t, csvData, "Total Gross Revenue")
}

func TestRevenueService_ValidateQuery(t *testing.T) {
	db := setupRevenueTestDB(t)
	service := NewRevenueReportService(db).(*revenueReportService)

	t.Run("valid date range", func(t *testing.T) {
		startDate := time.Now().AddDate(0, 0, -30)
		endDate := time.Now()
		query := &models.RevenueReportQuery{
			StartDate: &startDate,
			EndDate:   &endDate,
		}
		err := service.validateQuery(query)
		assert.NoError(t, err)
	})

	t.Run("invalid date range - end before start", func(t *testing.T) {
		startDate := time.Now()
		endDate := time.Now().AddDate(0, 0, -30)
		query := &models.RevenueReportQuery{
			StartDate: &startDate,
			EndDate:   &endDate,
		}
		err := service.validateQuery(query)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "end date must be after start date")
	})

	t.Run("invalid amount range", func(t *testing.T) {
		minAmount := decimal.NewFromFloat(1000)
		maxAmount := decimal.NewFromFloat(100)
		query := &models.RevenueReportQuery{
			MinAmount: &minAmount,
			MaxAmount: &maxAmount,
		}
		err := service.validateQuery(query)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "max amount must be greater than min amount")
	})
}

func TestRevenueService_Cache(t *testing.T) {
	db := setupRevenueTestDB(t)
	createTestTransactions(t, db)

	service := NewRevenueReportService(db)

	periodStart := time.Now().AddDate(0, 0, -30)
	periodEnd := time.Now()

	t.Run("refresh cache", func(t *testing.T) {
		err := service.RefreshCache(context.Background(), models.PeriodTypeMonthly, periodStart, periodEnd)
		require.NoError(t, err)
	})

	t.Run("get cached summary", func(t *testing.T) {
		cached, err := service.GetCachedSummary(context.Background(), models.PeriodTypeMonthly, periodStart, periodEnd)
		require.NoError(t, err)
		assert.NotNil(t, cached)
		assert.False(t, cached.IsStale)
	})
}

// Benchmark tests
func BenchmarkGenerateReport(b *testing.B) {
	db, _ := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})

	db.Exec(`
		CREATE TABLE revenue_transactions (
			id TEXT PRIMARY KEY,
			payment_id TEXT NOT NULL,
			user_id TEXT NOT NULL,
			invoice_id TEXT,
			order_id TEXT,
			transaction_type TEXT NOT NULL,
			transaction_date DATETIME NOT NULL,
			gross_amount DECIMAL(12,2) NOT NULL,
			net_amount DECIMAL(12,2) NOT NULL,
			fee_amount DECIMAL(12,2) NOT NULL DEFAULT 0,
			tax_amount DECIMAL(12,2) NOT NULL DEFAULT 0,
			discount_amount DECIMAL(12,2) NOT NULL DEFAULT 0,
			refund_amount DECIMAL(12,2) NOT NULL DEFAULT 0,
			currency TEXT NOT NULL DEFAULT 'USD',
			revenue_category TEXT,
			status TEXT NOT NULL DEFAULT 'completed',
			created_at DATETIME,
			updated_at DATETIME
		)
	`)

	// Create 1000 test transactions
	for i := 0; i < 1000; i++ {
		tx := models.RevenueTransaction{
			ID:              uuid.New(),
			PaymentID:       uuid.New(),
			UserID:          uuid.New(),
			TransactionType: models.TransactionTypePayment,
			TransactionDate: time.Now().AddDate(0, 0, -(i % 365)),
			GrossAmount:     decimal.NewFromFloat(float64(100 + (i % 900))),
			NetAmount:       decimal.NewFromFloat(float64(95 + (i % 850))),
			FeeAmount:       decimal.NewFromFloat(5.00),
			Currency:        "USD",
			Status:          models.TransactionStatusCompleted,
		}
		db.Create(&tx)
	}

	service := NewRevenueReportService(db)
	startDate := time.Now().AddDate(-1, 0, 0)
	endDate := time.Now()

	query := &models.RevenueReportQuery{
		StartDate:  &startDate,
		EndDate:    &endDate,
		PeriodType: models.PeriodTypeMonthly,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		service.GenerateReport(context.Background(), query)
	}
}

func BenchmarkGetTimeSeries(b *testing.B) {
	db, _ := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})

	db.Exec(`
		CREATE TABLE revenue_transactions (
			id TEXT PRIMARY KEY,
			payment_id TEXT NOT NULL,
			user_id TEXT NOT NULL,
			transaction_type TEXT NOT NULL,
			transaction_date DATETIME NOT NULL,
			gross_amount DECIMAL(12,2) NOT NULL,
			net_amount DECIMAL(12,2) NOT NULL,
			fee_amount DECIMAL(12,2) NOT NULL DEFAULT 0,
			tax_amount DECIMAL(12,2) NOT NULL DEFAULT 0,
			currency TEXT NOT NULL DEFAULT 'USD',
			status TEXT NOT NULL DEFAULT 'completed'
		)
	`)

	// Create 1000 test transactions
	for i := 0; i < 1000; i++ {
		tx := models.RevenueTransaction{
			ID:              uuid.New(),
			PaymentID:       uuid.New(),
			UserID:          uuid.New(),
			TransactionType: models.TransactionTypePayment,
			TransactionDate: time.Now().AddDate(0, 0, -(i % 365)),
			GrossAmount:     decimal.NewFromFloat(float64(100 + (i % 900))),
			NetAmount:       decimal.NewFromFloat(float64(95 + (i % 850))),
			FeeAmount:       decimal.NewFromFloat(5.00),
			Currency:        "USD",
			Status:          models.TransactionStatusCompleted,
		}
		db.Create(&tx)
	}

	service := NewRevenueReportService(db)
	startDate := time.Now().AddDate(-1, 0, 0)
	endDate := time.Now()

	query := &models.RevenueReportQuery{
		StartDate:  &startDate,
		EndDate:    &endDate,
		PeriodType: models.PeriodTypeDaily,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		service.GetTimeSeries(context.Background(), query)
	}
}
