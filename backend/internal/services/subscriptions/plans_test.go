package subscriptions

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// mockLogger implements Logger interface for testing
type mockLogger struct{}

func (m *mockLogger) Debugf(format string, v ...interface{}) {}
func (m *mockLogger) Infof(format string, v ...interface{})  {}
func (m *mockLogger) Warnf(format string, v ...interface{})  {}
func (m *mockLogger) Errorf(format string, v ...interface{}) {}

// setupTestDB creates an in-memory SQLite database for testing
func setupTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)

	// Migrate schemas
	err = db.AutoMigrate(&Plan{}, &PlanFeature{}, &Subscription{}, &SubscriptionEvent{})
	require.NoError(t, err)

	// Create subscription_invoices table manually for SQLite compatibility
	err = db.Exec(`
		CREATE TABLE IF NOT EXISTS subscription_invoices (
			id TEXT PRIMARY KEY,
			subscription_id TEXT NOT NULL,
			customer_id TEXT NOT NULL,
			stripe_invoice_id TEXT UNIQUE,
			subtotal INTEGER DEFAULT 0,
			tax INTEGER DEFAULT 0,
			total INTEGER DEFAULT 0,
			amount_due INTEGER DEFAULT 0,
			amount_paid INTEGER DEFAULT 0,
			amount_remaining INTEGER DEFAULT 0,
			currency TEXT DEFAULT 'USD',
			status TEXT NOT NULL,
			collection_method TEXT,
			period_start DATETIME,
			period_end DATETIME,
			due_date DATETIME,
			paid_at DATETIME,
			voided_at DATETIME,
			hosted_invoice_url TEXT,
			invoice_pdf TEXT,
			description TEXT,
			metadata TEXT,
			created_at DATETIME,
			updated_at DATETIME
		)
	`).Error
	require.NoError(t, err)

	// Create invoice_line_items table
	err = db.Exec(`
		CREATE TABLE IF NOT EXISTS invoice_line_items (
			id TEXT PRIMARY KEY,
			invoice_id TEXT NOT NULL,
			description TEXT,
			amount INTEGER DEFAULT 0,
			quantity INTEGER DEFAULT 1,
			unit_amount INTEGER DEFAULT 0,
			currency TEXT DEFAULT 'USD',
			period_start DATETIME,
			period_end DATETIME,
			proration INTEGER DEFAULT 0,
			created_at DATETIME
		)
	`).Error
	require.NoError(t, err)

	// Create subscription_usage_records table manually for SQLite compatibility
	err = db.Exec(`
		CREATE TABLE IF NOT EXISTS subscription_usage_records (
			id TEXT PRIMARY KEY,
			subscription_id TEXT NOT NULL,
			metric_key TEXT NOT NULL,
			quantity INTEGER DEFAULT 0,
			timestamp DATETIME,
			idempotency_key TEXT,
			created_at DATETIME
		)
	`).Error
	require.NoError(t, err)

	// Create index on subscription_usage_records but skip unique constraint for testing
	err = db.Exec(`
		CREATE INDEX IF NOT EXISTS idx_usage_subscription ON subscription_usage_records(subscription_id)
	`).Error
	require.NoError(t, err)

	err = db.Exec(`
		CREATE INDEX IF NOT EXISTS idx_usage_metric ON subscription_usage_records(metric_key)
	`).Error
	require.NoError(t, err)

	return db
}

// createTestPlan creates a plan for testing
func createTestPlan(name string, tier PlanTier, monthlyPrice, annualPrice int64) *Plan {
	return &Plan{
		ID:           uuid.New(),
		Name:         name,
		Description:  "Test plan",
		Tier:         tier,
		MonthlyPrice: monthlyPrice,
		AnnualPrice:  annualPrice,
		Currency:     "USD",
		IsActive:     true,
		IsPublic:     true,
		SortOrder:    1,
	}
}

func TestNewPlanService(t *testing.T) {
	db := setupTestDB(t)
	logger := &mockLogger{}

	t.Run("creates service with default config", func(t *testing.T) {
		service := NewPlanService(nil, db, logger)
		assert.NotNil(t, service)
		assert.NotNil(t, service.config)
		assert.Equal(t, 10*time.Minute, service.config.CacheTTL)
		assert.True(t, service.config.EnableCache)
	})

	t.Run("creates service with custom config", func(t *testing.T) {
		config := &PlanServiceConfig{
			CacheTTL:        5 * time.Minute,
			EnableCache:     false,
			DefaultCurrency: "EUR",
		}
		service := NewPlanService(config, db, logger)
		assert.NotNil(t, service)
		assert.Equal(t, 5*time.Minute, service.config.CacheTTL)
		assert.False(t, service.config.EnableCache)
	})
}

func TestPlanService_CreatePlan(t *testing.T) {
	db := setupTestDB(t)
	logger := &mockLogger{}
	service := NewPlanService(nil, db, logger)
	ctx := context.Background()

	t.Run("creates plan successfully", func(t *testing.T) {
		plan := createTestPlan("Starter", PlanTierStarter, 2900, 29000)

		err := service.CreatePlan(ctx, plan)
		assert.NoError(t, err)
		assert.NotEqual(t, uuid.Nil, plan.ID)
	})

	t.Run("fails with empty name", func(t *testing.T) {
		plan := &Plan{
			Tier:         PlanTierStarter,
			MonthlyPrice: 2900,
		}

		err := service.CreatePlan(ctx, plan)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "plan name is required")
	})

	t.Run("fails with duplicate name", func(t *testing.T) {
		plan1 := createTestPlan("Duplicate", PlanTierStarter, 2900, 29000)
		err := service.CreatePlan(ctx, plan1)
		require.NoError(t, err)

		plan2 := createTestPlan("Duplicate", PlanTierProfessional, 7900, 79000)
		err = service.CreatePlan(ctx, plan2)
		assert.Equal(t, ErrPlanAlreadyExists, err)
	})

	t.Run("calculates quarterly price if not set", func(t *testing.T) {
		plan := createTestPlan("WithQuarterly", PlanTierStarter, 2900, 29000)
		plan.QuarterlyPrice = 0

		err := service.CreatePlan(ctx, plan)
		assert.NoError(t, err)
		// Expected: 2900 * 3 * 90 / 100 = 7830
		assert.Equal(t, int64(7830), plan.QuarterlyPrice)
	})

	t.Run("sets default currency", func(t *testing.T) {
		plan := createTestPlan("DefaultCurrency", PlanTierStarter, 2900, 29000)
		plan.Currency = ""

		err := service.CreatePlan(ctx, plan)
		assert.NoError(t, err)
		assert.Equal(t, "USD", plan.Currency)
	})
}

func TestPlanService_GetPlan(t *testing.T) {
	db := setupTestDB(t)
	logger := &mockLogger{}
	service := NewPlanService(nil, db, logger)
	ctx := context.Background()

	t.Run("retrieves existing plan", func(t *testing.T) {
		plan := createTestPlan("GetTest", PlanTierStarter, 2900, 29000)
		err := service.CreatePlan(ctx, plan)
		require.NoError(t, err)

		retrieved, err := service.GetPlan(ctx, plan.ID)
		assert.NoError(t, err)
		assert.Equal(t, plan.Name, retrieved.Name)
		assert.Equal(t, plan.MonthlyPrice, retrieved.MonthlyPrice)
	})

	t.Run("returns error for non-existent plan", func(t *testing.T) {
		_, err := service.GetPlan(ctx, uuid.New())
		assert.Equal(t, ErrPlanNotFound, err)
	})

	t.Run("uses cache on second retrieval", func(t *testing.T) {
		plan := createTestPlan("CacheTest", PlanTierStarter, 2900, 29000)
		err := service.CreatePlan(ctx, plan)
		require.NoError(t, err)

		// First retrieval - populates cache
		_, err = service.GetPlan(ctx, plan.ID)
		require.NoError(t, err)

		// Second retrieval - should use cache
		retrieved, err := service.GetPlan(ctx, plan.ID)
		assert.NoError(t, err)
		assert.Equal(t, plan.Name, retrieved.Name)
	})
}

func TestPlanService_GetPlanByTier(t *testing.T) {
	db := setupTestDB(t)
	logger := &mockLogger{}
	service := NewPlanService(nil, db, logger)
	ctx := context.Background()

	t.Run("retrieves plan by tier", func(t *testing.T) {
		plan := createTestPlan("Professional", PlanTierProfessional, 7900, 79000)
		err := service.CreatePlan(ctx, plan)
		require.NoError(t, err)

		retrieved, err := service.GetPlanByTier(ctx, PlanTierProfessional)
		assert.NoError(t, err)
		assert.Equal(t, PlanTierProfessional, retrieved.Tier)
	})

	t.Run("returns error for non-existent tier", func(t *testing.T) {
		_, err := service.GetPlanByTier(ctx, PlanTierEnterprise)
		assert.Equal(t, ErrPlanNotFound, err)
	})
}

func TestPlanService_ListPlans(t *testing.T) {
	db := setupTestDB(t)
	logger := &mockLogger{}
	service := NewPlanService(nil, db, logger)
	ctx := context.Background()

	// Create test plans
	plans := []*Plan{
		createTestPlan("Free", PlanTierFree, 0, 0),
		createTestPlan("Starter List", PlanTierStarter, 2900, 29000),
		createTestPlan("Pro List", PlanTierProfessional, 7900, 79000),
	}
	plans[0].SortOrder = 1
	plans[1].SortOrder = 2
	plans[2].SortOrder = 3

	for _, p := range plans {
		err := service.CreatePlan(ctx, p)
		require.NoError(t, err)
	}

	t.Run("lists all public plans", func(t *testing.T) {
		result, err := service.ListPlans(ctx, false)
		assert.NoError(t, err)
		assert.Len(t, result, 3)
	})

	t.Run("excludes private plans when includePrivate is false", func(t *testing.T) {
		privatePlan := createTestPlan("Private", PlanTierBusiness, 19900, 199000)
		privatePlan.IsPublic = false
		err := service.CreatePlan(ctx, privatePlan)
		require.NoError(t, err)

		result, err := service.ListPlans(ctx, false)
		assert.NoError(t, err)
		// Should not include private plan
		for _, p := range result {
			assert.True(t, p.IsPublic)
		}
	})

	t.Run("includes private plans when includePrivate is true", func(t *testing.T) {
		result, err := service.ListPlans(ctx, true)
		assert.NoError(t, err)
		assert.GreaterOrEqual(t, len(result), 3)
	})
}

func TestPlanService_UpdatePlan(t *testing.T) {
	db := setupTestDB(t)
	logger := &mockLogger{}
	service := NewPlanService(nil, db, logger)
	ctx := context.Background()

	t.Run("updates plan successfully", func(t *testing.T) {
		plan := createTestPlan("ToUpdate", PlanTierStarter, 2900, 29000)
		err := service.CreatePlan(ctx, plan)
		require.NoError(t, err)

		updates := map[string]interface{}{
			"name":          "Updated Name",
			"monthly_price": int64(3900),
		}

		updated, err := service.UpdatePlan(ctx, plan.ID, updates)
		assert.NoError(t, err)
		assert.Equal(t, "Updated Name", updated.Name)
		assert.Equal(t, int64(3900), updated.MonthlyPrice)
	})

	t.Run("returns error for non-existent plan", func(t *testing.T) {
		_, err := service.UpdatePlan(ctx, uuid.New(), map[string]interface{}{"name": "Test"})
		assert.Equal(t, ErrPlanNotFound, err)
	})
}

func TestPlanService_DeletePlan(t *testing.T) {
	db := setupTestDB(t)
	logger := &mockLogger{}
	service := NewPlanService(nil, db, logger)
	ctx := context.Background()

	t.Run("deletes plan without subscriptions", func(t *testing.T) {
		plan := createTestPlan("ToDelete", PlanTierStarter, 2900, 29000)
		err := service.CreatePlan(ctx, plan)
		require.NoError(t, err)

		err = service.DeletePlan(ctx, plan.ID)
		assert.NoError(t, err)

		// Verify plan is deleted
		_, err = service.GetPlan(ctx, plan.ID)
		assert.Equal(t, ErrPlanNotFound, err)
	})
}

func TestPlanService_AddFeature(t *testing.T) {
	db := setupTestDB(t)
	logger := &mockLogger{}
	service := NewPlanService(nil, db, logger)
	ctx := context.Background()

	t.Run("adds feature to plan", func(t *testing.T) {
		plan := createTestPlan("WithFeature", PlanTierStarter, 2900, 29000)
		err := service.CreatePlan(ctx, plan)
		require.NoError(t, err)

		feature := &PlanFeature{
			FeatureKey:   "projects",
			FeatureName:  "Projects",
			FeatureType:  FeatureTypeNumeric,
			NumericValue: ptr(int64(10)),
			IsEnabled:    true,
		}

		err = service.AddFeature(ctx, plan.ID, feature)
		assert.NoError(t, err)
		assert.Equal(t, plan.ID, feature.PlanID)
	})

	t.Run("fails for non-existent plan", func(t *testing.T) {
		feature := &PlanFeature{
			FeatureKey:  "test",
			FeatureName: "Test",
			FeatureType: FeatureTypeBoolean,
			IsEnabled:   true,
		}

		err := service.AddFeature(ctx, uuid.New(), feature)
		assert.Equal(t, ErrPlanNotFound, err)
	})
}

func TestPlanService_HasFeature(t *testing.T) {
	db := setupTestDB(t)
	logger := &mockLogger{}
	service := NewPlanService(nil, db, logger)
	ctx := context.Background()

	plan := createTestPlan("FeatureCheck", PlanTierStarter, 2900, 29000)
	err := service.CreatePlan(ctx, plan)
	require.NoError(t, err)

	feature := &PlanFeature{
		FeatureKey:  "api_access",
		FeatureName: "API Access",
		FeatureType: FeatureTypeBoolean,
		IsEnabled:   true,
	}
	err = service.AddFeature(ctx, plan.ID, feature)
	require.NoError(t, err)

	t.Run("returns true for enabled feature", func(t *testing.T) {
		has, err := service.HasFeature(ctx, plan.ID, "api_access")
		assert.NoError(t, err)
		assert.True(t, has)
	})

	t.Run("returns false for non-existent feature", func(t *testing.T) {
		has, err := service.HasFeature(ctx, plan.ID, "non_existent")
		assert.NoError(t, err)
		assert.False(t, has)
	})
}

func TestPlanService_GetFeatureLimit(t *testing.T) {
	db := setupTestDB(t)
	logger := &mockLogger{}
	service := NewPlanService(nil, db, logger)
	ctx := context.Background()

	plan := createTestPlan("LimitCheck", PlanTierStarter, 2900, 29000)
	err := service.CreatePlan(ctx, plan)
	require.NoError(t, err)

	feature := &PlanFeature{
		FeatureKey:   "projects",
		FeatureName:  "Projects",
		FeatureType:  FeatureTypeNumeric,
		NumericValue: ptr(int64(10)),
		IsEnabled:    true,
	}
	err = service.AddFeature(ctx, plan.ID, feature)
	require.NoError(t, err)

	t.Run("returns numeric limit", func(t *testing.T) {
		limit, err := service.GetFeatureLimit(ctx, plan.ID, "projects")
		assert.NoError(t, err)
		assert.Equal(t, int64(10), limit)
	})

	t.Run("returns 0 for non-existent feature", func(t *testing.T) {
		limit, err := service.GetFeatureLimit(ctx, plan.ID, "non_existent")
		assert.NoError(t, err)
		assert.Equal(t, int64(0), limit)
	})
}

func TestPlanService_GetPriceForCycle(t *testing.T) {
	db := setupTestDB(t)
	logger := &mockLogger{}
	service := NewPlanService(nil, db, logger)

	plan := &Plan{
		MonthlyPrice:   2900,
		QuarterlyPrice: 7800,
		AnnualPrice:    29000,
	}

	tests := []struct {
		name     string
		cycle    BillingCycle
		expected int64
		hasError bool
	}{
		{"monthly", BillingCycleMonthly, 2900, false},
		{"quarterly", BillingCycleQuarterly, 7800, false},
		{"annual", BillingCycleAnnual, 29000, false},
		{"invalid", BillingCycle("invalid"), 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			price, err := service.GetPriceForCycle(plan, tt.cycle)
			if tt.hasError {
				assert.Equal(t, ErrInvalidBillingCycle, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, price)
			}
		})
	}
}

func TestPlanService_CalculateSavings(t *testing.T) {
	db := setupTestDB(t)
	logger := &mockLogger{}
	service := NewPlanService(nil, db, logger)

	plan := &Plan{
		MonthlyPrice:   2900,  // $29/month
		QuarterlyPrice: 7800,  // $78/quarter (vs $87 = 10% savings)
		AnnualPrice:    29000, // $290/year (vs $348 = ~17% savings)
	}

	savings := service.CalculateSavings(plan)

	// Annual: 2900 * 12 = 34800, savings = 34800 - 29000 = 5800
	assert.Equal(t, int64(5800), savings["annual_savings"])
	assert.Equal(t, int64(16), savings["annual_savings_percent"]) // 5800/34800 * 100 = 16%

	// Quarterly: 2900 * 3 = 8700, savings = 8700 - 7800 = 900
	assert.Equal(t, int64(900), savings["quarterly_savings"])
	assert.Equal(t, int64(10), savings["quarterly_savings_percent"]) // 900/8700 * 100 = 10%
}

func TestPlanService_ComparePlans(t *testing.T) {
	db := setupTestDB(t)
	logger := &mockLogger{}
	service := NewPlanService(nil, db, logger)
	ctx := context.Background()

	// Create two plans with features
	plan1 := createTestPlan("Plan A", PlanTierStarter, 2900, 29000)
	err := service.CreatePlan(ctx, plan1)
	require.NoError(t, err)

	feature1 := &PlanFeature{
		FeatureKey:   "projects",
		FeatureName:  "Projects",
		FeatureType:  FeatureTypeNumeric,
		NumericValue: ptr(int64(5)),
		IsEnabled:    true,
	}
	err = service.AddFeature(ctx, plan1.ID, feature1)
	require.NoError(t, err)

	plan2 := createTestPlan("Plan B", PlanTierProfessional, 7900, 79000)
	err = service.CreatePlan(ctx, plan2)
	require.NoError(t, err)

	feature2 := &PlanFeature{
		FeatureKey:   "projects",
		FeatureName:  "Projects",
		FeatureType:  FeatureTypeNumeric,
		NumericValue: ptr(int64(20)),
		IsEnabled:    true,
	}
	err = service.AddFeature(ctx, plan2.ID, feature2)
	require.NoError(t, err)

	t.Run("compares two plans", func(t *testing.T) {
		comparison, err := service.ComparePlans(ctx, plan1.ID, plan2.ID)
		assert.NoError(t, err)
		assert.Equal(t, plan1.ID, comparison.Plan1.ID)
		assert.Equal(t, plan2.ID, comparison.Plan2.ID)
		assert.NotEmpty(t, comparison.FeatureComparisons)
	})
}

func TestPlanService_Cache(t *testing.T) {
	db := setupTestDB(t)
	logger := &mockLogger{}
	config := &PlanServiceConfig{
		CacheTTL:        100 * time.Millisecond,
		EnableCache:     true,
		DefaultCurrency: "USD",
	}
	service := NewPlanService(config, db, logger)
	ctx := context.Background()

	plan := createTestPlan("CacheExpiry", PlanTierStarter, 2900, 29000)
	err := service.CreatePlan(ctx, plan)
	require.NoError(t, err)

	t.Run("cache expires after TTL", func(t *testing.T) {
		// First retrieval - populates cache
		_, err := service.GetPlan(ctx, plan.ID)
		require.NoError(t, err)

		// Wait for cache to expire
		time.Sleep(150 * time.Millisecond)

		// Verify cache is expired (getFromCache returns nil)
		cached := service.getFromCache(plan.ID)
		assert.Nil(t, cached)
	})

	t.Run("ClearCache invalidates all cached data", func(t *testing.T) {
		// Populate cache
		_, err := service.GetPlan(ctx, plan.ID)
		require.NoError(t, err)

		// Clear cache
		service.ClearCache()

		// Verify cache is empty
		cached := service.getFromCache(plan.ID)
		assert.Nil(t, cached)
	})
}

func TestGetDefaultPlans(t *testing.T) {
	plans := GetDefaultPlans()

	assert.Len(t, plans, 5) // Free, Starter, Professional, Business, Enterprise

	// Verify tier order
	tiers := []PlanTier{PlanTierFree, PlanTierStarter, PlanTierProfessional, PlanTierBusiness, PlanTierEnterprise}
	for i, plan := range plans {
		assert.Equal(t, tiers[i], plan.Tier)
		assert.True(t, plan.IsActive)
		assert.True(t, plan.IsPublic)
		assert.NotEmpty(t, plan.Features)
	}
}

func TestBillingCycleConstants(t *testing.T) {
	assert.Equal(t, BillingCycle("monthly"), BillingCycleMonthly)
	assert.Equal(t, BillingCycle("quarterly"), BillingCycleQuarterly)
	assert.Equal(t, BillingCycle("annual"), BillingCycleAnnual)
}

func TestPlanTierConstants(t *testing.T) {
	assert.Equal(t, PlanTier("free"), PlanTierFree)
	assert.Equal(t, PlanTier("starter"), PlanTierStarter)
	assert.Equal(t, PlanTier("professional"), PlanTierProfessional)
	assert.Equal(t, PlanTier("business"), PlanTierBusiness)
	assert.Equal(t, PlanTier("enterprise"), PlanTierEnterprise)
}

func TestFeatureTypeConstants(t *testing.T) {
	assert.Equal(t, FeatureType("boolean"), FeatureTypeBoolean)
	assert.Equal(t, FeatureType("numeric"), FeatureTypeNumeric)
	assert.Equal(t, FeatureType("text"), FeatureTypeText)
}
