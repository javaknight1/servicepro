package subscriptions

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewBillingService(t *testing.T) {
	db := setupTestDB(t)
	logger := &mockLogger{}
	planService := NewPlanService(nil, db, logger)

	t.Run("creates service with default config", func(t *testing.T) {
		service := NewBillingService(nil, db, planService, logger)
		assert.NotNil(t, service)
		assert.NotNil(t, service.config)
		assert.Equal(t, float64(0), service.config.TaxRate)
		assert.Equal(t, 7, service.config.GracePeriodDays)
	})

	t.Run("creates service with custom config", func(t *testing.T) {
		config := &BillingServiceConfig{
			TaxRate:            0.08,
			EnableAutomaticTax: true,
			GracePeriodDays:    14,
		}
		service := NewBillingService(config, db, planService, logger)
		assert.NotNil(t, service)
		assert.Equal(t, 0.08, service.config.TaxRate)
		assert.True(t, service.config.EnableAutomaticTax)
	})
}

func TestBillingService_CalculateProration(t *testing.T) {
	db := setupTestDB(t)
	logger := &mockLogger{}
	planService := NewPlanService(nil, db, logger)
	billingService := NewBillingService(nil, db, planService, logger)
	ctx := context.Background()

	// Create test plans
	currentPlan := createTestPlan("Current", PlanTierStarter, 2900, 29000)
	err := planService.CreatePlan(ctx, currentPlan)
	require.NoError(t, err)

	newPlan := createTestPlan("New Plan", PlanTierProfessional, 7900, 79000)
	err = planService.CreatePlan(ctx, newPlan)
	require.NoError(t, err)

	now := time.Now()
	periodStart := now.Add(-15 * 24 * time.Hour)
	periodEnd := now.Add(15 * 24 * time.Hour)

	sub := &Subscription{
		ID:                 uuid.New(),
		CustomerID:         uuid.New(),
		PlanID:             currentPlan.ID,
		Plan:               currentPlan,
		Status:             SubscriptionStatusActive,
		BillingCycle:       BillingCycleMonthly,
		CurrentPeriodStart: periodStart,
		CurrentPeriodEnd:   periodEnd,
		Currency:           "USD",
	}

	t.Run("calculates upgrade proration", func(t *testing.T) {
		result, err := billingService.CalculateProration(ctx, sub, newPlan, now)
		assert.NoError(t, err)
		assert.NotNil(t, result)

		// Net should be positive (charge) for upgrade
		assert.Greater(t, result.NetAmount, int64(0))
		assert.Greater(t, result.ChargeAmount, result.CreditAmount)
		assert.Contains(t, result.Description, "Upgrade")
	})

	t.Run("calculates downgrade proration", func(t *testing.T) {
		// Create a cheaper plan
		cheapPlan := createTestPlan("Cheap", PlanTierFree, 0, 0)
		err := planService.CreatePlan(ctx, cheapPlan)
		require.NoError(t, err)

		result, err := billingService.CalculateProration(ctx, sub, cheapPlan, now)
		assert.NoError(t, err)
		assert.NotNil(t, result)

		// Net should be negative (credit) for downgrade
		assert.Less(t, result.NetAmount, int64(0))
		assert.Contains(t, result.Description, "Downgrade")
	})

	t.Run("fails with invalid effective date before period", func(t *testing.T) {
		invalidDate := periodStart.Add(-1 * 24 * time.Hour)
		_, err := billingService.CalculateProration(ctx, sub, newPlan, invalidDate)
		assert.Equal(t, ErrInvalidProrationDate, err)
	})

	t.Run("fails with invalid effective date after period", func(t *testing.T) {
		invalidDate := periodEnd.Add(1 * 24 * time.Hour)
		_, err := billingService.CalculateProration(ctx, sub, newPlan, invalidDate)
		assert.Equal(t, ErrInvalidProrationDate, err)
	})

	t.Run("returns correct days remaining", func(t *testing.T) {
		result, err := billingService.CalculateProration(ctx, sub, newPlan, now)
		assert.NoError(t, err)
		assert.GreaterOrEqual(t, result.DaysRemaining, 14)
		assert.LessOrEqual(t, result.DaysRemaining, 16)
		assert.GreaterOrEqual(t, result.TotalDays, 29)
	})
}

func TestBillingService_GetBillingSchedule(t *testing.T) {
	db := setupTestDB(t)
	logger := &mockLogger{}
	planService := NewPlanService(nil, db, logger)
	billingService := NewBillingService(nil, db, planService, logger)
	ctx := context.Background()

	plan := createTestPlan("SchedulePlan", PlanTierStarter, 2900, 29000)
	err := planService.CreatePlan(ctx, plan)
	require.NoError(t, err)

	now := time.Now()
	sub := &Subscription{
		ID:                 uuid.New(),
		CustomerID:         uuid.New(),
		PlanID:             plan.ID,
		Status:             SubscriptionStatusActive,
		BillingCycle:       BillingCycleMonthly,
		CurrentPeriodStart: now,
		CurrentPeriodEnd:   now.Add(30 * 24 * time.Hour),
		Currency:           "USD",
	}
	err = db.Create(sub).Error
	require.NoError(t, err)

	t.Run("returns billing schedule with upcoming periods", func(t *testing.T) {
		schedule, err := billingService.GetBillingSchedule(ctx, sub.ID, 3)
		assert.NoError(t, err)
		assert.NotNil(t, schedule)

		assert.True(t, schedule.CurrentPeriod.IsCurrent)
		assert.True(t, schedule.CurrentPeriod.IsPaid)
		assert.Equal(t, BillingCycleMonthly, schedule.BillingCycle)
		assert.Equal(t, "USD", schedule.Currency)
		assert.Len(t, schedule.UpcomingPeriods, 3)
	})

	t.Run("upcoming periods have correct amounts", func(t *testing.T) {
		schedule, err := billingService.GetBillingSchedule(ctx, sub.ID, 2)
		assert.NoError(t, err)

		for _, period := range schedule.UpcomingPeriods {
			assert.Equal(t, int64(2900), period.Amount)
			assert.False(t, period.IsCurrent)
			assert.False(t, period.IsPaid)
		}
	})

	t.Run("upcoming periods are sequential", func(t *testing.T) {
		schedule, err := billingService.GetBillingSchedule(ctx, sub.ID, 3)
		assert.NoError(t, err)

		currentEnd := schedule.CurrentPeriod.EndDate
		for _, period := range schedule.UpcomingPeriods {
			// Each period should start when the previous one ends
			assert.Equal(t, currentEnd.Truncate(time.Second), period.StartDate.Truncate(time.Second))
			currentEnd = period.EndDate
		}
	})
}

func TestBillingService_GetInvoice(t *testing.T) {
	db := setupTestDB(t)
	logger := &mockLogger{}
	planService := NewPlanService(nil, db, logger)
	billingService := NewBillingService(nil, db, planService, logger)
	ctx := context.Background()

	// Create a test invoice
	invoice := &Invoice{
		ID:              uuid.New(),
		SubscriptionID:  uuid.New(),
		CustomerID:      uuid.New(),
		StripeInvoiceID: "inv_test123",
		Subtotal:        2900,
		Tax:             0,
		Total:           2900,
		AmountDue:       2900,
		AmountPaid:      0,
		Currency:        "USD",
		Status:          InvoiceStatusOpen,
		PeriodStart:     time.Now(),
		PeriodEnd:       time.Now().Add(30 * 24 * time.Hour),
	}
	err := db.Create(invoice).Error
	require.NoError(t, err)

	t.Run("retrieves invoice by ID", func(t *testing.T) {
		retrieved, err := billingService.GetInvoice(ctx, invoice.ID)
		assert.NoError(t, err)
		assert.Equal(t, invoice.ID, retrieved.ID)
		assert.Equal(t, invoice.Total, retrieved.Total)
	})

	t.Run("returns error for non-existent invoice", func(t *testing.T) {
		_, err := billingService.GetInvoice(ctx, uuid.New())
		assert.Equal(t, ErrInvoiceNotFound, err)
	})
}

func TestBillingService_ListInvoices(t *testing.T) {
	db := setupTestDB(t)
	logger := &mockLogger{}
	planService := NewPlanService(nil, db, logger)
	billingService := NewBillingService(nil, db, planService, logger)
	ctx := context.Background()

	subscriptionID := uuid.New()

	// Create test invoices
	for i := 0; i < 5; i++ {
		invoice := &Invoice{
			ID:              uuid.New(),
			SubscriptionID:  subscriptionID,
			CustomerID:      uuid.New(),
			StripeInvoiceID: uuid.New().String(),
			Subtotal:        2900,
			Total:           2900,
			Currency:        "USD",
			Status:          InvoiceStatusPaid,
			PeriodStart:     time.Now().Add(time.Duration(-i) * 30 * 24 * time.Hour),
			PeriodEnd:       time.Now().Add(time.Duration(-i+1) * 30 * 24 * time.Hour),
		}
		err := db.Create(invoice).Error
		require.NoError(t, err)
	}

	t.Run("lists invoices for subscription", func(t *testing.T) {
		invoices, total, err := billingService.ListInvoices(ctx, subscriptionID, 10, 0)
		assert.NoError(t, err)
		assert.Equal(t, int64(5), total)
		assert.Len(t, invoices, 5)
	})

	t.Run("applies limit", func(t *testing.T) {
		invoices, total, err := billingService.ListInvoices(ctx, subscriptionID, 2, 0)
		assert.NoError(t, err)
		assert.Equal(t, int64(5), total)
		assert.Len(t, invoices, 2)
	})

	t.Run("applies offset", func(t *testing.T) {
		invoices, total, err := billingService.ListInvoices(ctx, subscriptionID, 10, 3)
		assert.NoError(t, err)
		assert.Equal(t, int64(5), total)
		assert.Len(t, invoices, 2)
	})
}

func TestBillingService_UpdateInvoiceStatus(t *testing.T) {
	db := setupTestDB(t)
	logger := &mockLogger{}
	planService := NewPlanService(nil, db, logger)
	billingService := NewBillingService(nil, db, planService, logger)
	ctx := context.Background()

	invoice := &Invoice{
		ID:              uuid.New(),
		SubscriptionID:  uuid.New(),
		CustomerID:      uuid.New(),
		StripeInvoiceID: "inv_update_test",
		Subtotal:        2900,
		Total:           2900,
		Currency:        "USD",
		Status:          InvoiceStatusOpen,
		PeriodStart:     time.Now(),
		PeriodEnd:       time.Now().Add(30 * 24 * time.Hour),
	}
	err := db.Create(invoice).Error
	require.NoError(t, err)

	t.Run("updates status to paid", func(t *testing.T) {
		err := billingService.UpdateInvoiceStatus(ctx, invoice.StripeInvoiceID, InvoiceStatusPaid)
		assert.NoError(t, err)

		// Verify the update
		var updated Invoice
		err = db.Where("id = ?", invoice.ID).First(&updated).Error
		require.NoError(t, err)
		assert.Equal(t, InvoiceStatusPaid, updated.Status)
		assert.NotNil(t, updated.PaidAt)
	})

	t.Run("updates status to void", func(t *testing.T) {
		invoice2 := &Invoice{
			ID:              uuid.New(),
			SubscriptionID:  uuid.New(),
			CustomerID:      uuid.New(),
			StripeInvoiceID: "inv_void_test",
			Subtotal:        2900,
			Total:           2900,
			Currency:        "USD",
			Status:          InvoiceStatusOpen,
			PeriodStart:     time.Now(),
			PeriodEnd:       time.Now().Add(30 * 24 * time.Hour),
		}
		err := db.Create(invoice2).Error
		require.NoError(t, err)

		err = billingService.UpdateInvoiceStatus(ctx, invoice2.StripeInvoiceID, InvoiceStatusVoid)
		assert.NoError(t, err)

		var updated Invoice
		err = db.Where("id = ?", invoice2.ID).First(&updated).Error
		require.NoError(t, err)
		assert.Equal(t, InvoiceStatusVoid, updated.Status)
		assert.NotNil(t, updated.VoidedAt)
	})
}

func TestBillingService_RecordUsage(t *testing.T) {
	db := setupTestDB(t)
	logger := &mockLogger{}
	planService := NewPlanService(nil, db, logger)
	billingService := NewBillingService(nil, db, planService, logger)
	ctx := context.Background()

	subscriptionID := uuid.New()

	t.Run("records usage successfully", func(t *testing.T) {
		err := billingService.RecordUsage(ctx, subscriptionID, "api_calls", 100, time.Now(), "key123")
		assert.NoError(t, err)

		// Verify record was created
		var record UsageRecord
		err = db.Where("subscription_id = ? AND metric_key = ?", subscriptionID, "api_calls").First(&record).Error
		assert.NoError(t, err)
		assert.Equal(t, int64(100), record.Quantity)
	})

	t.Run("records multiple usage entries", func(t *testing.T) {
		for i := 0; i < 5; i++ {
			err := billingService.RecordUsage(ctx, subscriptionID, "storage_gb", int64(i+1), time.Now(), uuid.New().String())
			assert.NoError(t, err)
		}

		var count int64
		err := db.Model(&UsageRecord{}).Where("subscription_id = ? AND metric_key = ?", subscriptionID, "storage_gb").Count(&count).Error
		assert.NoError(t, err)
		assert.Equal(t, int64(5), count)
	})
}

func TestBillingService_GetUsageSummary(t *testing.T) {
	db := setupTestDB(t)
	logger := &mockLogger{}
	planService := NewPlanService(nil, db, logger)
	billingService := NewBillingService(nil, db, planService, logger)
	ctx := context.Background()

	subscriptionID := uuid.New()
	now := time.Now()
	startDate := now.Add(-7 * 24 * time.Hour)
	endDate := now.Add(1 * 24 * time.Hour)

	// Create usage records
	records := []UsageRecord{
		{ID: uuid.New(), SubscriptionID: subscriptionID, MetricKey: "api_calls", Quantity: 100, Timestamp: now.Add(-1 * 24 * time.Hour)},
		{ID: uuid.New(), SubscriptionID: subscriptionID, MetricKey: "api_calls", Quantity: 200, Timestamp: now.Add(-2 * 24 * time.Hour)},
		{ID: uuid.New(), SubscriptionID: subscriptionID, MetricKey: "storage_gb", Quantity: 5, Timestamp: now.Add(-1 * 24 * time.Hour)},
	}
	for _, r := range records {
		err := db.Create(&r).Error
		require.NoError(t, err)
	}

	t.Run("returns aggregated usage summary", func(t *testing.T) {
		summary, err := billingService.GetUsageSummary(ctx, subscriptionID, startDate, endDate)
		assert.NoError(t, err)
		assert.Equal(t, int64(300), summary["api_calls"])
		assert.Equal(t, int64(5), summary["storage_gb"])
	})

	t.Run("returns empty summary for no usage", func(t *testing.T) {
		summary, err := billingService.GetUsageSummary(ctx, uuid.New(), startDate, endDate)
		assert.NoError(t, err)
		assert.Empty(t, summary)
	})
}

func TestBillingService_getPeriodDuration(t *testing.T) {
	db := setupTestDB(t)
	logger := &mockLogger{}
	planService := NewPlanService(nil, db, logger)
	billingService := NewBillingService(nil, db, planService, logger)

	tests := []struct {
		name     string
		cycle    BillingCycle
		expected time.Duration
	}{
		{"monthly", BillingCycleMonthly, 30 * 24 * time.Hour},
		{"quarterly", BillingCycleQuarterly, 90 * 24 * time.Hour},
		{"annual", BillingCycleAnnual, 365 * 24 * time.Hour},
		{"default", BillingCycle("unknown"), 30 * 24 * time.Hour},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			duration := billingService.getPeriodDuration(tt.cycle)
			assert.Equal(t, tt.expected, duration)
		})
	}
}

func TestInvoiceStatus_Constants(t *testing.T) {
	assert.Equal(t, InvoiceStatus("draft"), InvoiceStatusDraft)
	assert.Equal(t, InvoiceStatus("open"), InvoiceStatusOpen)
	assert.Equal(t, InvoiceStatus("paid"), InvoiceStatusPaid)
	assert.Equal(t, InvoiceStatus("uncollectible"), InvoiceStatusUncollectible)
	assert.Equal(t, InvoiceStatus("void"), InvoiceStatusVoid)
}

func TestInvoice_TableName(t *testing.T) {
	inv := Invoice{}
	assert.Equal(t, "subscription_invoices", inv.TableName())
}

func TestInvoiceLineItem_TableName(t *testing.T) {
	li := InvoiceLineItem{}
	assert.Equal(t, "invoice_line_items", li.TableName())
}

func TestUsageRecord_TableName(t *testing.T) {
	ur := UsageRecord{}
	assert.Equal(t, "subscription_usage_records", ur.TableName())
}

func TestDefaultBillingServiceConfig(t *testing.T) {
	config := DefaultBillingServiceConfig()

	assert.Equal(t, float64(0), config.TaxRate)
	assert.False(t, config.EnableAutomaticTax)
	assert.Equal(t, "charge_automatically", config.DefaultCollectionMethod)
	assert.Equal(t, 7, config.GracePeriodDays)
}

func TestProrationResult_Fields(t *testing.T) {
	now := time.Now()
	result := &ProrationResult{
		CreditAmount:  1000,
		ChargeAmount:  2500,
		NetAmount:     1500,
		DaysRemaining: 15,
		TotalDays:     30,
		EffectiveDate: now,
		Description:   "Upgrade proration",
	}

	assert.Equal(t, int64(1000), result.CreditAmount)
	assert.Equal(t, int64(2500), result.ChargeAmount)
	assert.Equal(t, int64(1500), result.NetAmount)
	assert.Equal(t, 15, result.DaysRemaining)
	assert.Equal(t, 30, result.TotalDays)
	assert.Equal(t, now, result.EffectiveDate)
	assert.Equal(t, "Upgrade proration", result.Description)
}

func TestBillingPeriod_Fields(t *testing.T) {
	now := time.Now()
	period := BillingPeriod{
		StartDate: now,
		EndDate:   now.Add(30 * 24 * time.Hour),
		Amount:    2900,
		IsCurrent: true,
		IsPaid:    true,
	}

	assert.Equal(t, now, period.StartDate)
	assert.Equal(t, int64(2900), period.Amount)
	assert.True(t, period.IsCurrent)
	assert.True(t, period.IsPaid)
}

func TestBillingSchedule_Fields(t *testing.T) {
	now := time.Now()
	schedule := BillingSchedule{
		CurrentPeriod: BillingPeriod{
			StartDate: now,
			EndDate:   now.Add(30 * 24 * time.Hour),
			Amount:    2900,
			IsCurrent: true,
		},
		UpcomingPeriods: []BillingPeriod{
			{StartDate: now.Add(30 * 24 * time.Hour), Amount: 2900},
		},
		BillingCycle:    BillingCycleMonthly,
		NextBillingDate: now.Add(30 * 24 * time.Hour),
		AmountDue:       2900,
		Currency:        "USD",
	}

	assert.NotNil(t, schedule.CurrentPeriod)
	assert.Len(t, schedule.UpcomingPeriods, 1)
	assert.Equal(t, BillingCycleMonthly, schedule.BillingCycle)
	assert.Equal(t, "USD", schedule.Currency)
}

func TestUpcomingInvoice_Fields(t *testing.T) {
	now := time.Now()
	upcoming := UpcomingInvoice{
		Subtotal:        2900,
		Tax:             232,
		Total:           3132,
		AmountDue:       3132,
		Currency:        "USD",
		PeriodStart:     now,
		PeriodEnd:       now.Add(30 * 24 * time.Hour),
		NextPaymentDate: now.Add(30 * 24 * time.Hour),
		LineItems: []UpcomingInvoiceLineItem{
			{Description: "Starter Plan", Amount: 2900, Quantity: 1},
		},
	}

	assert.Equal(t, int64(2900), upcoming.Subtotal)
	assert.Equal(t, int64(232), upcoming.Tax)
	assert.Equal(t, int64(3132), upcoming.Total)
	assert.Len(t, upcoming.LineItems, 1)
}

func TestPlanChangePreview_Fields(t *testing.T) {
	currentPlan := &Plan{Name: "Starter"}
	newPlan := &Plan{Name: "Professional"}
	now := time.Now()

	preview := PlanChangePreview{
		CurrentPlan:       currentPlan,
		NewPlan:           newPlan,
		ImmediateCharge:   5000,
		NextBillingDate:   now.Add(30 * 24 * time.Hour),
		NextBillingAmount: 7900,
		ProrationDetails: &ProrationResult{
			NetAmount: 5000,
		},
	}

	assert.Equal(t, "Starter", preview.CurrentPlan.Name)
	assert.Equal(t, "Professional", preview.NewPlan.Name)
	assert.Equal(t, int64(5000), preview.ImmediateCharge)
	assert.NotNil(t, preview.ProrationDetails)
}
