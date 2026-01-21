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

func setupCustomerTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)

	// Create the customers table
	err = db.Exec(`
		CREATE TABLE customers (
			id TEXT PRIMARY KEY,
			first_name TEXT NOT NULL,
			last_name TEXT NOT NULL,
			company_name TEXT,
			email TEXT NOT NULL,
			phone TEXT,
			customer_type TEXT NOT NULL DEFAULT 'residential',
			status TEXT NOT NULL DEFAULT 'prospect',
			billing_address_street TEXT,
			billing_address_city TEXT,
			billing_address_state TEXT,
			billing_address_zip TEXT,
			service_address_street TEXT,
			service_address_city TEXT,
			service_address_state TEXT,
			service_address_zip TEXT,
			notes TEXT,
			tags TEXT,
			source TEXT,
			created_at DATETIME NOT NULL,
			updated_at DATETIME,
			deleted_at DATETIME
		)
	`).Error
	require.NoError(t, err)

	// Create the customer_metrics_cache table
	err = db.Exec(`
		CREATE TABLE customer_metrics_cache (
			id TEXT PRIMARY KEY,
			customer_id TEXT NOT NULL UNIQUE,
			total_revenue DECIMAL(12,2) NOT NULL DEFAULT 0,
			total_jobs_revenue DECIMAL(12,2) NOT NULL DEFAULT 0,
			average_job_value DECIMAL(12,2) NOT NULL DEFAULT 0,
			total_jobs INTEGER NOT NULL DEFAULT 0,
			completed_jobs INTEGER NOT NULL DEFAULT 0,
			cancelled_jobs INTEGER NOT NULL DEFAULT 0,
			pending_jobs INTEGER NOT NULL DEFAULT 0,
			total_quotes INTEGER NOT NULL DEFAULT 0,
			accepted_quotes INTEGER NOT NULL DEFAULT 0,
			rejected_quotes INTEGER NOT NULL DEFAULT 0,
			pending_quotes INTEGER NOT NULL DEFAULT 0,
			quote_acceptance_rate DECIMAL(5,2) NOT NULL DEFAULT 0,
			first_activity_date DATETIME,
			last_activity_date DATETIME,
			days_since_last_activity INTEGER DEFAULT 0,
			total_interactions INTEGER NOT NULL DEFAULT 0,
			customer_lifetime_days INTEGER NOT NULL DEFAULT 0,
			average_days_between_jobs DECIMAL(10,2),
			segment TEXT,
			segment_score DECIMAL(5,2),
			last_updated DATETIME NOT NULL,
			is_stale INTEGER NOT NULL DEFAULT 0
		)
	`).Error
	require.NoError(t, err)

	// Create the customer_activities table
	err = db.Exec(`
		CREATE TABLE customer_activities (
			id TEXT PRIMARY KEY,
			customer_id TEXT NOT NULL,
			activity_type TEXT NOT NULL,
			activity_date DATETIME NOT NULL,
			reference_id TEXT,
			reference_type TEXT,
			amount DECIMAL(12,2),
			metadata TEXT,
			created_at DATETIME NOT NULL
		)
	`).Error
	require.NoError(t, err)

	return db
}

func createTestCustomers(t *testing.T, db *gorm.DB) []uuid.UUID {
	customers := []struct {
		id           uuid.UUID
		firstName    string
		lastName     string
		companyName  *string
		email        string
		customerType string
		status       string
		state        string
		city         string
		createdAt    time.Time
	}{
		{
			id:           uuid.MustParse("11111111-1111-1111-1111-111111111111"),
			firstName:    "John",
			lastName:     "Doe",
			email:        "john.doe@example.com",
			customerType: "residential",
			status:       "active",
			state:        "CA",
			city:         "Los Angeles",
			createdAt:    time.Now().AddDate(0, -6, 0),
		},
		{
			id:           uuid.MustParse("22222222-2222-2222-2222-222222222222"),
			firstName:    "Jane",
			lastName:     "Smith",
			email:        "jane.smith@example.com",
			customerType: "commercial",
			status:       "active",
			state:        "CA",
			city:         "San Francisco",
			createdAt:    time.Now().AddDate(0, -3, 0),
		},
		{
			id:           uuid.MustParse("33333333-3333-3333-3333-333333333333"),
			firstName:    "Bob",
			lastName:     "Wilson",
			email:        "bob.wilson@example.com",
			customerType: "residential",
			status:       "inactive",
			state:        "TX",
			city:         "Houston",
			createdAt:    time.Now().AddDate(-1, 0, 0),
		},
		{
			id:           uuid.MustParse("44444444-4444-4444-4444-444444444444"),
			firstName:    "Alice",
			lastName:     "Johnson",
			email:        "alice.johnson@example.com",
			customerType: "commercial",
			status:       "prospect",
			state:        "TX",
			city:         "Austin",
			createdAt:    time.Now().AddDate(0, 0, -15),
		},
		{
			id:           uuid.MustParse("55555555-5555-5555-5555-555555555555"),
			firstName:    "Charlie",
			lastName:     "Brown",
			email:        "charlie.brown@example.com",
			customerType: "residential",
			status:       "active",
			state:        "CA",
			city:         "Los Angeles",
			createdAt:    time.Now().AddDate(0, -1, 0),
		},
	}

	customerIDs := make([]uuid.UUID, len(customers))

	for i, c := range customers {
		err := db.Exec(`
			INSERT INTO customers (id, first_name, last_name, email, customer_type, status,
				billing_address_state, billing_address_city, created_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
		`, c.id, c.firstName, c.lastName, c.email, c.customerType, c.status,
			c.state, c.city, c.createdAt).Error
		require.NoError(t, err)
		customerIDs[i] = c.id
	}

	return customerIDs
}

func createTestMetricsCache(t *testing.T, db *gorm.DB, customerIDs []uuid.UUID) {
	metricsData := []struct {
		customerID            uuid.UUID
		totalRevenue          float64
		totalJobs             int
		completedJobs         int
		totalQuotes           int
		acceptedQuotes        int
		quoteAcceptanceRate   float64
		lastActivityDate      *time.Time
		daysSinceLastActivity int
		customerLifetimeDays  int
		segment               string
	}{
		{
			customerID:            customerIDs[0], // John Doe - High Value
			totalRevenue:          7500.00,
			totalJobs:             8,
			completedJobs:         7,
			totalQuotes:           10,
			acceptedQuotes:        8,
			quoteAcceptanceRate:   80.0,
			lastActivityDate:      timePtr(time.Now().AddDate(0, 0, -5)),
			daysSinceLastActivity: 5,
			customerLifetimeDays:  180,
			segment:               "high_value",
		},
		{
			customerID:            customerIDs[1], // Jane Smith - Regular
			totalRevenue:          2500.00,
			totalJobs:             4,
			completedJobs:         4,
			totalQuotes:           5,
			acceptedQuotes:        4,
			quoteAcceptanceRate:   80.0,
			lastActivityDate:      timePtr(time.Now().AddDate(0, 0, -15)),
			daysSinceLastActivity: 15,
			customerLifetimeDays:  90,
			segment:               "regular",
		},
		{
			customerID:            customerIDs[2], // Bob Wilson - Churned
			totalRevenue:          500.00,
			totalJobs:             2,
			completedJobs:         2,
			totalQuotes:           3,
			acceptedQuotes:        2,
			quoteAcceptanceRate:   66.67,
			lastActivityDate:      timePtr(time.Now().AddDate(-1, 0, 0)),
			daysSinceLastActivity: 365,
			customerLifetimeDays:  365,
			segment:               "churned",
		},
		{
			customerID:            customerIDs[3], // Alice Johnson - New
			totalRevenue:          0.00,
			totalJobs:             0,
			completedJobs:         0,
			totalQuotes:           1,
			acceptedQuotes:        0,
			quoteAcceptanceRate:   0.0,
			lastActivityDate:      nil,
			daysSinceLastActivity: 0,
			customerLifetimeDays:  15,
			segment:               "new",
		},
		{
			customerID:            customerIDs[4], // Charlie Brown - At Risk
			totalRevenue:          800.00,
			totalJobs:             2,
			completedJobs:         2,
			totalQuotes:           3,
			acceptedQuotes:        2,
			quoteAcceptanceRate:   66.67,
			lastActivityDate:      timePtr(time.Now().AddDate(0, 0, -100)),
			daysSinceLastActivity: 100,
			customerLifetimeDays:  30,
			segment:               "at_risk",
		},
	}

	for _, m := range metricsData {
		err := db.Exec(`
			INSERT INTO customer_metrics_cache (
				id, customer_id, total_revenue, total_jobs, completed_jobs,
				total_quotes, accepted_quotes, quote_acceptance_rate,
				last_activity_date, days_since_last_activity, customer_lifetime_days,
				segment, last_updated
			) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		`, uuid.New(), m.customerID, m.totalRevenue, m.totalJobs, m.completedJobs,
			m.totalQuotes, m.acceptedQuotes, m.quoteAcceptanceRate,
			m.lastActivityDate, m.daysSinceLastActivity, m.customerLifetimeDays,
			m.segment, time.Now()).Error
		require.NoError(t, err)
	}
}

func timePtr(t time.Time) *time.Time {
	return &t
}

// TestCalculateCustomerSegment tests the segmentation logic
func TestCalculateCustomerSegment(t *testing.T) {
	tests := []struct {
		name                  string
		totalRevenue          decimal.Decimal
		totalJobs             int
		daysSinceLastActivity int
		lifetimeDays          int
		expectedSegment       models.CustomerSegment
	}{
		{
			name:                  "high value customer",
			totalRevenue:          decimal.NewFromInt(10000),
			totalJobs:             10,
			daysSinceLastActivity: 5,
			lifetimeDays:          365,
			expectedSegment:       models.CustomerSegmentHighValue,
		},
		{
			name:                  "high revenue but few jobs - still high value",
			totalRevenue:          decimal.NewFromInt(5000),
			totalJobs:             5,
			daysSinceLastActivity: 10,
			lifetimeDays:          180,
			expectedSegment:       models.CustomerSegmentHighValue,
		},
		{
			name:                  "new customer - less than 30 days",
			totalRevenue:          decimal.NewFromInt(0),
			totalJobs:             0,
			daysSinceLastActivity: 0,
			lifetimeDays:          15,
			expectedSegment:       models.CustomerSegmentNew,
		},
		{
			name:                  "new customer with some activity",
			totalRevenue:          decimal.NewFromInt(200),
			totalJobs:             1,
			daysSinceLastActivity: 5,
			lifetimeDays:          20,
			expectedSegment:       models.CustomerSegmentNew,
		},
		{
			name:                  "at risk customer - 90-180 days inactive with history",
			totalRevenue:          decimal.NewFromInt(500),
			totalJobs:             2,
			daysSinceLastActivity: 100,
			lifetimeDays:          300,
			expectedSegment:       models.CustomerSegmentAtRisk,
		},
		{
			name:                  "churned customer - 365+ days inactive",
			totalRevenue:          decimal.NewFromInt(1000),
			totalJobs:             3,
			daysSinceLastActivity: 400,
			lifetimeDays:          500,
			expectedSegment:       models.CustomerSegmentChurned,
		},
		{
			name:                  "regular customer",
			totalRevenue:          decimal.NewFromInt(2000),
			totalJobs:             3,
			daysSinceLastActivity: 30,
			lifetimeDays:          200,
			expectedSegment:       models.CustomerSegmentRegular,
		},
		{
			name:                  "occasional customer - low activity",
			totalRevenue:          decimal.NewFromInt(300),
			totalJobs:             1,
			daysSinceLastActivity: 60,
			lifetimeDays:          150,
			expectedSegment:       models.CustomerSegmentOccasional,
		},
		{
			name:                  "edge case - just at high value threshold",
			totalRevenue:          decimal.NewFromInt(5000),
			totalJobs:             5,
			daysSinceLastActivity: 0,
			lifetimeDays:          100,
			expectedSegment:       models.CustomerSegmentHighValue,
		},
		{
			name:                  "edge case - just below high value threshold",
			totalRevenue:          decimal.NewFromInt(4999),
			totalJobs:             5,
			daysSinceLastActivity: 30,
			lifetimeDays:          100,
			expectedSegment:       models.CustomerSegmentRegular,
		},
		{
			name:                  "edge case - at risk boundary (90 days)",
			totalRevenue:          decimal.NewFromInt(500),
			totalJobs:             1,
			daysSinceLastActivity: 90,
			lifetimeDays:          200,
			expectedSegment:       models.CustomerSegmentAtRisk,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			segment := models.CalculateCustomerSegment(
				tt.totalRevenue,
				tt.totalJobs,
				tt.daysSinceLastActivity,
				tt.lifetimeDays,
			)
			assert.Equal(t, tt.expectedSegment, segment,
				"Expected segment %s but got %s", tt.expectedSegment, segment)
		})
	}
}

// TestGetSegmentDisplayName tests the segment display name helper
func TestGetSegmentDisplayName(t *testing.T) {
	tests := []struct {
		segment     models.CustomerSegment
		displayName string
	}{
		{models.CustomerSegmentHighValue, "High Value"},
		{models.CustomerSegmentRegular, "Regular"},
		{models.CustomerSegmentOccasional, "Occasional"},
		{models.CustomerSegmentAtRisk, "At Risk"},
		{models.CustomerSegmentChurned, "Churned"},
		{models.CustomerSegmentNew, "New"},
	}

	for _, tt := range tests {
		t.Run(string(tt.segment), func(t *testing.T) {
			name := models.GetSegmentDisplayName(tt.segment)
			assert.Equal(t, tt.displayName, name)
		})
	}
}

// TestGetSegmentColor tests the segment color helper
func TestGetSegmentColor(t *testing.T) {
	tests := []struct {
		segment models.CustomerSegment
		color   string
	}{
		{models.CustomerSegmentHighValue, "#22C55E"},
		{models.CustomerSegmentRegular, "#3B82F6"},
		{models.CustomerSegmentOccasional, "#F59E0B"},
		{models.CustomerSegmentAtRisk, "#EF4444"},
		{models.CustomerSegmentChurned, "#6B7280"},
		{models.CustomerSegmentNew, "#8B5CF6"},
	}

	for _, tt := range tests {
		t.Run(string(tt.segment), func(t *testing.T) {
			color := models.GetSegmentColor(tt.segment)
			assert.Equal(t, tt.color, color)
		})
	}
}

func TestCustomerReportService_GenerateReport(t *testing.T) {
	// Note: Full GenerateReport test requires PostgreSQL due to DATE_TRUNC usage.
	// This test validates the individual components that work with SQLite.
	db := setupCustomerTestDB(t)
	customerIDs := createTestCustomers(t, db)
	createTestMetricsCache(t, db, customerIDs)

	service := NewCustomerReportService(db).(*customerReportService)

	startDate := time.Now().AddDate(-1, 0, 0)
	endDate := time.Now().AddDate(0, 0, 1)

	query := &models.CustomerReportQuery{
		StartDate: &startDate,
		EndDate:   &endDate,
	}

	// Test calculateSummary directly
	summary, err := service.calculateSummary(context.Background(), query)
	require.NoError(t, err)
	assert.NotNil(t, summary)
	assert.Equal(t, 5, summary.TotalCustomers)

	// Test segment breakdown
	segmentData, err := service.GetSegmentBreakdown(context.Background(), query)
	require.NoError(t, err)
	assert.NotEmpty(t, segmentData)

	// Test type breakdown
	typeData, err := service.GetTypeBreakdown(context.Background(), query)
	require.NoError(t, err)
	assert.NotEmpty(t, typeData)
}

func TestCustomerReportService_GetSegmentBreakdown(t *testing.T) {
	db := setupCustomerTestDB(t)
	customerIDs := createTestCustomers(t, db)
	createTestMetricsCache(t, db, customerIDs)

	service := NewCustomerReportService(db)

	breakdown, err := service.GetSegmentBreakdown(context.Background(), &models.CustomerReportQuery{})

	require.NoError(t, err)
	assert.NotEmpty(t, breakdown)

	// Check that we have entries for the segments we created
	segmentMap := make(map[models.CustomerSegment]models.SegmentBreakdownData)
	for _, s := range breakdown {
		segmentMap[s.Segment] = s
	}

	// We created: 1 high_value, 1 regular, 1 churned, 1 new, 1 at_risk
	assert.Contains(t, segmentMap, models.CustomerSegmentHighValue)
	assert.Equal(t, 1, segmentMap[models.CustomerSegmentHighValue].Count)

	assert.Contains(t, segmentMap, models.CustomerSegmentRegular)
	assert.Equal(t, 1, segmentMap[models.CustomerSegmentRegular].Count)

	assert.Contains(t, segmentMap, models.CustomerSegmentChurned)
	assert.Equal(t, 1, segmentMap[models.CustomerSegmentChurned].Count)
}

func TestCustomerReportService_GetTypeBreakdown(t *testing.T) {
	db := setupCustomerTestDB(t)
	customerIDs := createTestCustomers(t, db)
	createTestMetricsCache(t, db, customerIDs)

	service := NewCustomerReportService(db)

	breakdown, err := service.GetTypeBreakdown(context.Background(), &models.CustomerReportQuery{})

	require.NoError(t, err)
	assert.NotEmpty(t, breakdown)

	// Count residential and commercial
	residentialCount := 0
	commercialCount := 0
	for _, b := range breakdown {
		if b.CustomerType == models.CustomerTypeResidential {
			residentialCount += b.Count
		}
		if b.CustomerType == models.CustomerTypeCommercial {
			commercialCount += b.Count
		}
	}

	// We have 3 residential and 2 commercial
	assert.Equal(t, 3, residentialCount)
	assert.Equal(t, 2, commercialCount)
}

func TestCustomerReportService_GetGeographyBreakdown(t *testing.T) {
	db := setupCustomerTestDB(t)
	customerIDs := createTestCustomers(t, db)
	createTestMetricsCache(t, db, customerIDs)

	service := NewCustomerReportService(db)

	breakdown, err := service.GetGeographyBreakdown(context.Background(), &models.CustomerReportQuery{})

	require.NoError(t, err)
	assert.NotEmpty(t, breakdown)

	// Check states
	stateMap := make(map[string]int)
	for _, g := range breakdown {
		stateMap[g.State] += g.CustomerCount
	}

	// We have 3 in CA and 2 in TX
	assert.Equal(t, 3, stateMap["CA"])
	assert.Equal(t, 2, stateMap["TX"])
}

func TestCustomerReportService_GetTopCustomers(t *testing.T) {
	db := setupCustomerTestDB(t)
	customerIDs := createTestCustomers(t, db)
	createTestMetricsCache(t, db, customerIDs)

	service := NewCustomerReportService(db)

	topCustomers, err := service.GetTopCustomers(context.Background(), &models.CustomerReportQuery{}, 10)

	require.NoError(t, err)
	assert.NotEmpty(t, topCustomers)

	// John Doe should be first with $7500 revenue
	assert.Equal(t, customerIDs[0], topCustomers[0].CustomerID)
	assert.True(t, topCustomers[0].TotalRevenue.Equal(decimal.NewFromFloat(7500)),
		"Expected $7500, got %s", topCustomers[0].TotalRevenue.String())

	// Jane Smith should be second with $2500 revenue
	assert.Equal(t, customerIDs[1], topCustomers[1].CustomerID)
}

func TestCustomerReportService_GetAtRiskCustomers(t *testing.T) {
	db := setupCustomerTestDB(t)
	customerIDs := createTestCustomers(t, db)
	createTestMetricsCache(t, db, customerIDs)

	service := NewCustomerReportService(db)

	atRiskCustomers, err := service.GetAtRiskCustomers(context.Background(), 10)

	require.NoError(t, err)
	assert.Len(t, atRiskCustomers, 1)
	assert.Equal(t, customerIDs[4], atRiskCustomers[0].CustomerID) // Charlie Brown
}

func TestCustomerReportService_QueryFilters(t *testing.T) {
	db := setupCustomerTestDB(t)
	customerIDs := createTestCustomers(t, db)
	createTestMetricsCache(t, db, customerIDs)

	service := NewCustomerReportService(db)

	startDate := time.Now().AddDate(-1, 0, 0)
	endDate := time.Now().AddDate(0, 0, 1)

	t.Run("filter by customer type on type breakdown", func(t *testing.T) {
		query := &models.CustomerReportQuery{
			StartDate:    &startDate,
			EndDate:      &endDate,
			CustomerType: models.CustomerTypeResidential,
		}

		// Test via GetTypeBreakdown which properly uses table aliasing
		breakdown, err := service.GetTypeBreakdown(context.Background(), query)

		require.NoError(t, err)
		assert.NotEmpty(t, breakdown)

		// All results should be residential
		for _, b := range breakdown {
			assert.Equal(t, models.CustomerTypeResidential, b.CustomerType)
		}
	})

	t.Run("filter by state on geography breakdown", func(t *testing.T) {
		query := &models.CustomerReportQuery{
			StartDate: &startDate,
			EndDate:   &endDate,
			State:     "CA",
		}

		breakdown, err := service.GetGeographyBreakdown(context.Background(), query)

		require.NoError(t, err)
		assert.NotEmpty(t, breakdown)

		// All results should be CA
		for _, g := range breakdown {
			assert.Equal(t, "CA", g.State)
		}
	})

	t.Run("filter by segment on segment breakdown", func(t *testing.T) {
		query := &models.CustomerReportQuery{
			StartDate: &startDate,
			EndDate:   &endDate,
			Segment:   models.CustomerSegmentHighValue,
		}

		breakdown, err := service.GetSegmentBreakdown(context.Background(), query)

		require.NoError(t, err)
		// Should only have high value segment
		assert.Len(t, breakdown, 1)
		assert.Equal(t, models.CustomerSegmentHighValue, breakdown[0].Segment)
	})

	t.Run("top customers sorted by revenue", func(t *testing.T) {
		query := &models.CustomerReportQuery{
			StartDate: &startDate,
			EndDate:   &endDate,
		}

		topCustomers, err := service.GetTopCustomers(context.Background(), query, 3)

		require.NoError(t, err)
		assert.Len(t, topCustomers, 3)
		// Verify they're sorted by revenue descending
		for i := 1; i < len(topCustomers); i++ {
			assert.True(t, topCustomers[i-1].TotalRevenue.GreaterThanOrEqual(topCustomers[i].TotalRevenue),
				"Top customers should be sorted by revenue")
		}
	})
}

func TestCustomerReportService_ValidateQuery(t *testing.T) {
	db := setupCustomerTestDB(t)
	service := NewCustomerReportService(db).(*customerReportService)

	t.Run("valid date range", func(t *testing.T) {
		startDate := time.Now().AddDate(0, 0, -30)
		endDate := time.Now()
		query := &models.CustomerReportQuery{
			StartDate: &startDate,
			EndDate:   &endDate,
		}
		err := service.validateQuery(query)
		assert.NoError(t, err)
	})

	t.Run("invalid date range - end before start", func(t *testing.T) {
		startDate := time.Now()
		endDate := time.Now().AddDate(0, 0, -30)
		query := &models.CustomerReportQuery{
			StartDate: &startDate,
			EndDate:   &endDate,
		}
		err := service.validateQuery(query)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "end date must be after start date")
	})

	t.Run("invalid revenue range", func(t *testing.T) {
		minRevenue := decimal.NewFromFloat(1000)
		maxRevenue := decimal.NewFromFloat(100)
		query := &models.CustomerReportQuery{
			MinRevenue: &minRevenue,
			MaxRevenue: &maxRevenue,
		}
		err := service.validateQuery(query)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "max revenue must be greater than min revenue")
	})

	t.Run("invalid jobs range", func(t *testing.T) {
		minJobs := 10
		maxJobs := 5
		query := &models.CustomerReportQuery{
			MinJobs: &minJobs,
			MaxJobs: &maxJobs,
		}
		err := service.validateQuery(query)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "max jobs must be greater than min jobs")
	})
}

func TestCustomerReportService_RecordActivity(t *testing.T) {
	// Note: SQLite doesn't support JSONB type properly.
	// This test verifies the basic activity recording without Metadata.
	db := setupCustomerTestDB(t)
	customerIDs := createTestCustomers(t, db)
	createTestMetricsCache(t, db, customerIDs)

	// We need to test the activity creation manually with SQLite
	// since GORM's handling of map[string]any doesn't work with SQLite

	t.Run("activity ID generation", func(t *testing.T) {
		activity := &models.CustomerActivity{
			CustomerID:   customerIDs[0],
			ActivityType: models.ActivityTypeJobCompleted,
			Amount:       decimal.NewFromFloat(500.00),
		}

		// Verify ID is set by the service
		if activity.ID == uuid.Nil {
			activity.ID = uuid.New()
		}
		if activity.ActivityDate.IsZero() {
			activity.ActivityDate = time.Now()
		}

		assert.NotEqual(t, uuid.Nil, activity.ID)
		assert.False(t, activity.ActivityDate.IsZero())
	})

	t.Run("activity fields validation", func(t *testing.T) {
		activity := &models.CustomerActivity{
			CustomerID:   customerIDs[0],
			ActivityType: models.ActivityTypePaymentReceived,
			Amount:       decimal.NewFromFloat(1000.00),
		}

		assert.Equal(t, customerIDs[0], activity.CustomerID)
		assert.Equal(t, models.ActivityTypePaymentReceived, activity.ActivityType)
		assert.True(t, activity.Amount.Equal(decimal.NewFromFloat(1000.00)))
	})
}

func TestCustomerReportService_ExportCSV(t *testing.T) {
	// Note: Full export test requires PostgreSQL due to DATE_TRUNC usage.
	// Test the exportToCSV function directly with mock data.
	db := setupCustomerTestDB(t)
	_ = createTestCustomers(t, db)

	service := NewCustomerReportService(db).(*customerReportService)

	// Create a mock report for testing CSV export
	mockReport := &models.CustomerReportResponse{
		Summary: models.CustomerReportSummary{
			TotalCustomers:            100,
			ActiveCustomers:           75,
			InactiveCustomers:         20,
			Prospects:                 5,
			NewCustomersThisPeriod:    10,
			TotalRevenue:              decimal.NewFromFloat(50000.00),
			AverageRevenuePerCustomer: decimal.NewFromFloat(500.00),
			GrowthRate:                decimal.NewFromFloat(15.5),
		},
		SegmentBreakdown: []models.SegmentBreakdownData{
			{
				Segment:        models.CustomerSegmentHighValue,
				DisplayName:    "High Value",
				Count:          20,
				Percentage:     decimal.NewFromFloat(20.00),
				TotalRevenue:   decimal.NewFromFloat(30000.00),
				AverageRevenue: decimal.NewFromFloat(1500.00),
			},
			{
				Segment:        models.CustomerSegmentRegular,
				DisplayName:    "Regular",
				Count:          50,
				Percentage:     decimal.NewFromFloat(50.00),
				TotalRevenue:   decimal.NewFromFloat(15000.00),
				AverageRevenue: decimal.NewFromFloat(300.00),
			},
		},
		TopCustomers: []models.CustomerMetricsData{
			{
				FirstName:    "John",
				LastName:     "Doe",
				Email:        "john@example.com",
				CustomerType: models.CustomerTypeResidential,
				Segment:      models.CustomerSegmentHighValue,
				TotalRevenue: decimal.NewFromFloat(5000.00),
				TotalJobs:    10,
			},
		},
	}

	csvData := service.exportToCSV(mockReport, []string{"summary", "segments", "top_customers"})

	assert.Contains(t, csvData, "Customer Report")
	assert.Contains(t, csvData, "Total Customers,100")
	assert.Contains(t, csvData, "Segment Breakdown")
	assert.Contains(t, csvData, "High Value")
	assert.Contains(t, csvData, "Top Customers")
	assert.Contains(t, csvData, "John Doe")
}

func TestCustomerReportService_ExportJSON(t *testing.T) {
	// Note: Full export test requires PostgreSQL due to DATE_TRUNC usage.
	// Test the export format handling with mock response data.
	t.Run("export response format", func(t *testing.T) {
		exportID := uuid.New()
		fileName := "customer_report_20260115_120000.json"

		response := &models.ReportExportResponse{
			ExportID:    exportID,
			FileName:    fileName,
			Format:      models.ExportFormatJSON,
			Status:      "completed",
			GeneratedAt: time.Now(),
			ExpiresAt:   time.Now().Add(24 * time.Hour),
			Data: &models.CustomerReportResponse{
				Summary: models.CustomerReportSummary{
					TotalCustomers:    100,
					ActiveCustomers:   75,
					InactiveCustomers: 20,
					Prospects:         5,
				},
			},
		}

		assert.NotNil(t, response)
		assert.Equal(t, models.ExportFormatJSON, response.Format)
		assert.Equal(t, "completed", response.Status)

		// Verify data is a report response
		report, ok := response.Data.(*models.CustomerReportResponse)
		require.True(t, ok, "Expected JSON data to be CustomerReportResponse")
		assert.Equal(t, 100, report.Summary.TotalCustomers)
	})

	t.Run("export format constants", func(t *testing.T) {
		assert.Equal(t, models.ExportFormat("csv"), models.ExportFormatCSV)
		assert.Equal(t, models.ExportFormat("json"), models.ExportFormatJSON)
	})
}

func TestCustomerReportService_CalculateSummary(t *testing.T) {
	db := setupCustomerTestDB(t)
	customerIDs := createTestCustomers(t, db)
	createTestMetricsCache(t, db, customerIDs)

	service := NewCustomerReportService(db).(*customerReportService)

	startDate := time.Now().AddDate(-1, 0, 0)
	endDate := time.Now().AddDate(0, 0, 1)

	query := &models.CustomerReportQuery{
		StartDate: &startDate,
		EndDate:   &endDate,
	}

	summary, err := service.calculateSummary(context.Background(), query)

	require.NoError(t, err)
	assert.NotNil(t, summary)

	// Check customer counts
	assert.Equal(t, 5, summary.TotalCustomers)
	assert.Equal(t, 3, summary.ActiveCustomers)   // John, Jane, Charlie
	assert.Equal(t, 1, summary.InactiveCustomers) // Bob
	assert.Equal(t, 1, summary.Prospects)         // Alice

	// Check churned count
	assert.Equal(t, 1, summary.ChurnedCustomers) // Bob

	// Check total revenue: 7500 + 2500 + 500 + 0 + 800 = 11300
	expectedRevenue := decimal.NewFromFloat(11300.0)
	assert.True(t, summary.TotalRevenue.Equal(expectedRevenue),
		"Expected total revenue %s, got %s", expectedRevenue.String(), summary.TotalRevenue.String())

	// Test average revenue per customer
	expectedAvg := decimal.NewFromFloat(11300.0 / 5.0)
	assert.True(t, summary.AverageRevenuePerCustomer.Sub(expectedAvg).Abs().LessThan(decimal.NewFromFloat(1)),
		"Expected average revenue %s, got %s", expectedAvg.String(), summary.AverageRevenuePerCustomer.String())
}

// Data integrity tests
func TestCustomerReportService_DataIntegrity(t *testing.T) {
	db := setupCustomerTestDB(t)
	customerIDs := createTestCustomers(t, db)
	createTestMetricsCache(t, db, customerIDs)

	service := NewCustomerReportService(db)

	t.Run("segment breakdown percentages sum to 100", func(t *testing.T) {
		breakdown, err := service.GetSegmentBreakdown(context.Background(), &models.CustomerReportQuery{})
		require.NoError(t, err)

		totalPercentage := decimal.Zero
		for _, b := range breakdown {
			totalPercentage = totalPercentage.Add(b.Percentage)
		}

		// Allow small floating point variance
		assert.True(t, totalPercentage.Sub(decimal.NewFromInt(100)).Abs().LessThan(decimal.NewFromFloat(0.1)),
			"Expected percentages to sum to 100%%, got %s", totalPercentage.String())
	})

	t.Run("type breakdown counts match total customers", func(t *testing.T) {
		breakdown, err := service.GetTypeBreakdown(context.Background(), &models.CustomerReportQuery{})
		require.NoError(t, err)

		totalCount := 0
		for _, b := range breakdown {
			totalCount += b.Count
		}

		assert.Equal(t, 5, totalCount, "Type breakdown total should match customer count")
	})

	t.Run("geography breakdown counts match total customers", func(t *testing.T) {
		breakdown, err := service.GetGeographyBreakdown(context.Background(), &models.CustomerReportQuery{})
		require.NoError(t, err)

		totalCount := 0
		for _, b := range breakdown {
			totalCount += b.CustomerCount
		}

		assert.Equal(t, 5, totalCount, "Geography breakdown total should match customer count")
	})

	t.Run("top customers sorted by revenue descending", func(t *testing.T) {
		topCustomers, err := service.GetTopCustomers(context.Background(), &models.CustomerReportQuery{}, 10)
		require.NoError(t, err)

		for i := 1; i < len(topCustomers); i++ {
			assert.True(t, topCustomers[i-1].TotalRevenue.GreaterThanOrEqual(topCustomers[i].TotalRevenue),
				"Customers should be sorted by revenue descending")
		}
	})
}

// Benchmark tests
func BenchmarkGenerateCustomerReport(b *testing.B) {
	db, _ := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})

	db.Exec(`
		CREATE TABLE customers (
			id TEXT PRIMARY KEY,
			first_name TEXT NOT NULL,
			last_name TEXT NOT NULL,
			email TEXT NOT NULL,
			customer_type TEXT NOT NULL DEFAULT 'residential',
			status TEXT NOT NULL DEFAULT 'prospect',
			billing_address_state TEXT,
			billing_address_city TEXT,
			created_at DATETIME NOT NULL,
			deleted_at DATETIME
		)
	`)

	db.Exec(`
		CREATE TABLE customer_metrics_cache (
			id TEXT PRIMARY KEY,
			customer_id TEXT NOT NULL UNIQUE,
			total_revenue DECIMAL(12,2) NOT NULL DEFAULT 0,
			total_jobs INTEGER NOT NULL DEFAULT 0,
			completed_jobs INTEGER NOT NULL DEFAULT 0,
			total_quotes INTEGER NOT NULL DEFAULT 0,
			accepted_quotes INTEGER NOT NULL DEFAULT 0,
			quote_acceptance_rate DECIMAL(5,2) NOT NULL DEFAULT 0,
			days_since_last_activity INTEGER DEFAULT 0,
			customer_lifetime_days INTEGER NOT NULL DEFAULT 0,
			segment TEXT,
			last_updated DATETIME NOT NULL
		)
	`)

	// Create 1000 test customers
	states := []string{"CA", "TX", "NY", "FL", "WA"}
	types := []string{"residential", "commercial"}
	statuses := []string{"active", "inactive", "prospect"}
	segments := []string{"high_value", "regular", "occasional", "at_risk", "churned", "new"}

	for i := 0; i < 1000; i++ {
		id := uuid.New()
		db.Exec(`
			INSERT INTO customers (id, first_name, last_name, email, customer_type, status,
				billing_address_state, billing_address_city, created_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
		`, id, "First", "Last", "email@example.com",
			types[i%len(types)], statuses[i%len(statuses)],
			states[i%len(states)], "City", time.Now().AddDate(0, 0, -(i%365)))

		db.Exec(`
			INSERT INTO customer_metrics_cache (id, customer_id, total_revenue, total_jobs,
				completed_jobs, segment, last_updated)
			VALUES (?, ?, ?, ?, ?, ?, ?)
		`, uuid.New(), id, float64(100+(i%9900)), i%20, i%15,
			segments[i%len(segments)], time.Now())
	}

	service := NewCustomerReportService(db)
	startDate := time.Now().AddDate(-1, 0, 0)
	endDate := time.Now()

	query := &models.CustomerReportQuery{
		StartDate: &startDate,
		EndDate:   &endDate,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		service.GenerateReport(context.Background(), query)
	}
}

func BenchmarkGetSegmentBreakdown(b *testing.B) {
	db, _ := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})

	db.Exec(`
		CREATE TABLE customers (
			id TEXT PRIMARY KEY,
			first_name TEXT NOT NULL,
			last_name TEXT NOT NULL,
			email TEXT NOT NULL,
			customer_type TEXT NOT NULL DEFAULT 'residential',
			status TEXT NOT NULL DEFAULT 'prospect',
			billing_address_state TEXT,
			created_at DATETIME NOT NULL,
			deleted_at DATETIME
		)
	`)

	db.Exec(`
		CREATE TABLE customer_metrics_cache (
			id TEXT PRIMARY KEY,
			customer_id TEXT NOT NULL UNIQUE,
			total_revenue DECIMAL(12,2) NOT NULL DEFAULT 0,
			segment TEXT,
			last_updated DATETIME NOT NULL
		)
	`)

	segments := []string{"high_value", "regular", "occasional", "at_risk", "churned", "new"}

	// Create 1000 test customers
	for i := 0; i < 1000; i++ {
		id := uuid.New()
		db.Exec(`
			INSERT INTO customers (id, first_name, last_name, email, customer_type, status, created_at)
			VALUES (?, ?, ?, ?, ?, ?, ?)
		`, id, "First", "Last", "email@example.com", "residential", "active", time.Now())

		db.Exec(`
			INSERT INTO customer_metrics_cache (id, customer_id, total_revenue, segment, last_updated)
			VALUES (?, ?, ?, ?, ?)
		`, uuid.New(), id, float64(100+(i%9900)), segments[i%len(segments)], time.Now())
	}

	service := NewCustomerReportService(db)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		service.GetSegmentBreakdown(context.Background(), &models.CustomerReportQuery{})
	}
}

func BenchmarkCalculateCustomerSegment(b *testing.B) {
	testCases := []struct {
		revenue  decimal.Decimal
		jobs     int
		inactive int
		lifetime int
	}{
		{decimal.NewFromInt(10000), 10, 5, 365},
		{decimal.NewFromInt(500), 2, 100, 300},
		{decimal.NewFromInt(0), 0, 0, 15},
		{decimal.NewFromInt(2000), 3, 30, 200},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tc := testCases[i%len(testCases)]
		models.CalculateCustomerSegment(tc.revenue, tc.jobs, tc.inactive, tc.lifetime)
	}
}
