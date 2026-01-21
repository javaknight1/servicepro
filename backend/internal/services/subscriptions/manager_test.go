package subscriptions

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewManager(t *testing.T) {
	db := setupTestDB(t)
	logger := &mockLogger{}
	planService := NewPlanService(nil, db, logger)
	billingService := NewBillingService(nil, db, planService, logger)

	t.Run("creates manager with default config", func(t *testing.T) {
		manager := NewManager(nil, db, planService, billingService, nil, logger)
		assert.NotNil(t, manager)
		assert.NotNil(t, manager.config)
		assert.True(t, manager.config.AllowDowngrades)
		assert.Equal(t, 3, manager.config.GracePeriodDays)
	})

	t.Run("creates manager with custom config", func(t *testing.T) {
		config := &ManagerConfig{
			AllowDowngrades:      false,
			RequirePaymentMethod: false,
			GracePeriodDays:      7,
			MaxFailedPayments:    5,
		}
		manager := NewManager(config, db, planService, billingService, nil, logger)
		assert.NotNil(t, manager)
		assert.False(t, manager.config.AllowDowngrades)
		assert.Equal(t, 7, manager.config.GracePeriodDays)
	})
}

func TestSubscription_IsActive(t *testing.T) {
	tests := []struct {
		name     string
		status   SubscriptionStatus
		expected bool
	}{
		{"active", SubscriptionStatusActive, true},
		{"trialing", SubscriptionStatusTrialing, true},
		{"past_due", SubscriptionStatusPastDue, false},
		{"canceled", SubscriptionStatusCanceled, false},
		{"unpaid", SubscriptionStatusUnpaid, false},
		{"paused", SubscriptionStatusPaused, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sub := &Subscription{Status: tt.status}
			assert.Equal(t, tt.expected, sub.IsActive())
		})
	}
}

func TestSubscription_IsTrialing(t *testing.T) {
	tests := []struct {
		name     string
		status   SubscriptionStatus
		expected bool
	}{
		{"trialing", SubscriptionStatusTrialing, true},
		{"active", SubscriptionStatusActive, false},
		{"past_due", SubscriptionStatusPastDue, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sub := &Subscription{Status: tt.status}
			assert.Equal(t, tt.expected, sub.IsTrialing())
		})
	}
}

func TestSubscription_DaysUntilRenewal(t *testing.T) {
	t.Run("returns days until period end", func(t *testing.T) {
		sub := &Subscription{
			CurrentPeriodEnd:  time.Now().Add(10 * 24 * time.Hour),
			CancelAtPeriodEnd: false,
		}
		days := sub.DaysUntilRenewal()
		assert.GreaterOrEqual(t, days, 9)
		assert.LessOrEqual(t, days, 10)
	})

	t.Run("returns 0 when cancelled at period end", func(t *testing.T) {
		sub := &Subscription{
			CurrentPeriodEnd:  time.Now().Add(10 * 24 * time.Hour),
			CancelAtPeriodEnd: true,
		}
		assert.Equal(t, 0, sub.DaysUntilRenewal())
	})
}

func TestSubscription_TrialDaysRemaining(t *testing.T) {
	t.Run("returns days remaining in trial", func(t *testing.T) {
		trialEnd := time.Now().Add(7 * 24 * time.Hour)
		sub := &Subscription{
			Status:   SubscriptionStatusTrialing,
			TrialEnd: &trialEnd,
		}
		days := sub.TrialDaysRemaining()
		assert.GreaterOrEqual(t, days, 6)
		assert.LessOrEqual(t, days, 7)
	})

	t.Run("returns 0 when not trialing", func(t *testing.T) {
		trialEnd := time.Now().Add(7 * 24 * time.Hour)
		sub := &Subscription{
			Status:   SubscriptionStatusActive,
			TrialEnd: &trialEnd,
		}
		assert.Equal(t, 0, sub.TrialDaysRemaining())
	})

	t.Run("returns 0 when trial end is nil", func(t *testing.T) {
		sub := &Subscription{
			Status:   SubscriptionStatusTrialing,
			TrialEnd: nil,
		}
		assert.Equal(t, 0, sub.TrialDaysRemaining())
	})

	t.Run("returns 0 when trial has expired", func(t *testing.T) {
		trialEnd := time.Now().Add(-1 * 24 * time.Hour)
		sub := &Subscription{
			Status:   SubscriptionStatusTrialing,
			TrialEnd: &trialEnd,
		}
		assert.Equal(t, 0, sub.TrialDaysRemaining())
	})
}

func TestManager_GetSubscription(t *testing.T) {
	db := setupTestDB(t)
	logger := &mockLogger{}
	planService := NewPlanService(nil, db, logger)
	billingService := NewBillingService(nil, db, planService, logger)
	manager := NewManager(nil, db, planService, billingService, nil, logger)
	ctx := context.Background()

	// Create a test plan first
	plan := createTestPlan("TestPlan", PlanTierStarter, 2900, 29000)
	err := planService.CreatePlan(ctx, plan)
	require.NoError(t, err)

	// Create a test subscription directly
	sub := &Subscription{
		ID:                 uuid.New(),
		CustomerID:         uuid.New(),
		PlanID:             plan.ID,
		Status:             SubscriptionStatusActive,
		BillingCycle:       BillingCycleMonthly,
		StartDate:          time.Now(),
		CurrentPeriodStart: time.Now(),
		CurrentPeriodEnd:   time.Now().Add(30 * 24 * time.Hour),
		Currency:           "USD",
	}
	err = db.Create(sub).Error
	require.NoError(t, err)

	t.Run("retrieves subscription by ID", func(t *testing.T) {
		retrieved, err := manager.GetSubscription(ctx, sub.ID)
		assert.NoError(t, err)
		assert.Equal(t, sub.ID, retrieved.ID)
		assert.NotNil(t, retrieved.Plan)
	})

	t.Run("returns error for non-existent subscription", func(t *testing.T) {
		_, err := manager.GetSubscription(ctx, uuid.New())
		assert.Equal(t, ErrSubscriptionNotFound, err)
	})
}

func TestManager_GetSubscriptionByCustomer(t *testing.T) {
	db := setupTestDB(t)
	logger := &mockLogger{}
	planService := NewPlanService(nil, db, logger)
	billingService := NewBillingService(nil, db, planService, logger)
	manager := NewManager(nil, db, planService, billingService, nil, logger)
	ctx := context.Background()

	plan := createTestPlan("CustomerPlan", PlanTierStarter, 2900, 29000)
	err := planService.CreatePlan(ctx, plan)
	require.NoError(t, err)

	customerID := uuid.New()
	sub := &Subscription{
		ID:                 uuid.New(),
		CustomerID:         customerID,
		PlanID:             plan.ID,
		Status:             SubscriptionStatusActive,
		BillingCycle:       BillingCycleMonthly,
		StartDate:          time.Now(),
		CurrentPeriodStart: time.Now(),
		CurrentPeriodEnd:   time.Now().Add(30 * 24 * time.Hour),
		Currency:           "USD",
	}
	err = db.Create(sub).Error
	require.NoError(t, err)

	t.Run("retrieves subscription by customer", func(t *testing.T) {
		retrieved, err := manager.GetSubscriptionByCustomer(ctx, customerID)
		assert.NoError(t, err)
		assert.Equal(t, customerID, retrieved.CustomerID)
	})

	t.Run("returns error for customer without subscription", func(t *testing.T) {
		_, err := manager.GetSubscriptionByCustomer(ctx, uuid.New())
		assert.Equal(t, ErrSubscriptionNotFound, err)
	})
}

func TestManager_GetSubscriptionByStripeID(t *testing.T) {
	db := setupTestDB(t)
	logger := &mockLogger{}
	planService := NewPlanService(nil, db, logger)
	billingService := NewBillingService(nil, db, planService, logger)
	manager := NewManager(nil, db, planService, billingService, nil, logger)
	ctx := context.Background()

	plan := createTestPlan("StripePlan", PlanTierStarter, 2900, 29000)
	err := planService.CreatePlan(ctx, plan)
	require.NoError(t, err)

	stripeSubID := "sub_test123"
	sub := &Subscription{
		ID:                   uuid.New(),
		CustomerID:           uuid.New(),
		PlanID:               plan.ID,
		StripeSubscriptionID: stripeSubID,
		Status:               SubscriptionStatusActive,
		BillingCycle:         BillingCycleMonthly,
		StartDate:            time.Now(),
		CurrentPeriodStart:   time.Now(),
		CurrentPeriodEnd:     time.Now().Add(30 * 24 * time.Hour),
		Currency:             "USD",
	}
	err = db.Create(sub).Error
	require.NoError(t, err)

	t.Run("retrieves subscription by Stripe ID", func(t *testing.T) {
		retrieved, err := manager.GetSubscriptionByStripeID(ctx, stripeSubID)
		assert.NoError(t, err)
		assert.Equal(t, stripeSubID, retrieved.StripeSubscriptionID)
	})

	t.Run("returns error for non-existent Stripe ID", func(t *testing.T) {
		_, err := manager.GetSubscriptionByStripeID(ctx, "sub_nonexistent")
		assert.Equal(t, ErrSubscriptionNotFound, err)
	})
}

func TestManager_ListSubscriptions(t *testing.T) {
	db := setupTestDB(t)
	logger := &mockLogger{}
	planService := NewPlanService(nil, db, logger)
	billingService := NewBillingService(nil, db, planService, logger)
	manager := NewManager(nil, db, planService, billingService, nil, logger)
	ctx := context.Background()

	plan := createTestPlan("ListPlan", PlanTierStarter, 2900, 29000)
	err := planService.CreatePlan(ctx, plan)
	require.NoError(t, err)

	// Create multiple subscriptions
	customerID := uuid.New()
	for i := 0; i < 3; i++ {
		sub := &Subscription{
			ID:                 uuid.New(),
			CustomerID:         customerID,
			PlanID:             plan.ID,
			Status:             SubscriptionStatusActive,
			BillingCycle:       BillingCycleMonthly,
			StartDate:          time.Now(),
			CurrentPeriodStart: time.Now(),
			CurrentPeriodEnd:   time.Now().Add(30 * 24 * time.Hour),
			Currency:           "USD",
		}
		err := db.Create(sub).Error
		require.NoError(t, err)
	}

	t.Run("lists all subscriptions", func(t *testing.T) {
		opts := &ListSubscriptionsOptions{}
		subs, total, err := manager.ListSubscriptions(ctx, opts)
		assert.NoError(t, err)
		assert.GreaterOrEqual(t, total, int64(3))
		assert.NotEmpty(t, subs)
	})

	t.Run("filters by customer ID", func(t *testing.T) {
		opts := &ListSubscriptionsOptions{CustomerID: &customerID}
		subs, total, err := manager.ListSubscriptions(ctx, opts)
		assert.NoError(t, err)
		assert.Equal(t, int64(3), total)
		for _, s := range subs {
			assert.Equal(t, customerID, s.CustomerID)
		}
	})

	t.Run("filters by status", func(t *testing.T) {
		status := SubscriptionStatusActive
		opts := &ListSubscriptionsOptions{Status: &status}
		subs, _, err := manager.ListSubscriptions(ctx, opts)
		assert.NoError(t, err)
		for _, s := range subs {
			assert.Equal(t, SubscriptionStatusActive, s.Status)
		}
	})

	t.Run("applies limit and offset", func(t *testing.T) {
		opts := &ListSubscriptionsOptions{Limit: 2, Offset: 0}
		subs, total, err := manager.ListSubscriptions(ctx, opts)
		assert.NoError(t, err)
		assert.GreaterOrEqual(t, total, int64(3))
		assert.LessOrEqual(t, len(subs), 2)
	})
}

func TestManager_RevokeCancellation(t *testing.T) {
	db := setupTestDB(t)
	logger := &mockLogger{}
	planService := NewPlanService(nil, db, logger)
	billingService := NewBillingService(nil, db, planService, logger)
	manager := NewManager(nil, db, planService, billingService, nil, logger)
	ctx := context.Background()

	plan := createTestPlan("RevokePlan", PlanTierStarter, 2900, 29000)
	err := planService.CreatePlan(ctx, plan)
	require.NoError(t, err)

	t.Run("returns subscription unchanged if not scheduled for cancellation", func(t *testing.T) {
		sub := &Subscription{
			ID:                 uuid.New(),
			CustomerID:         uuid.New(),
			PlanID:             plan.ID,
			Status:             SubscriptionStatusActive,
			BillingCycle:       BillingCycleMonthly,
			CancelAtPeriodEnd:  false,
			StartDate:          time.Now(),
			CurrentPeriodStart: time.Now(),
			CurrentPeriodEnd:   time.Now().Add(30 * 24 * time.Hour),
			Currency:           "USD",
		}
		err := db.Create(sub).Error
		require.NoError(t, err)

		result, err := manager.RevokeCancellation(ctx, sub.ID)
		assert.NoError(t, err)
		assert.False(t, result.CancelAtPeriodEnd)
	})
}

func TestManager_GetEvents(t *testing.T) {
	db := setupTestDB(t)
	logger := &mockLogger{}
	planService := NewPlanService(nil, db, logger)
	billingService := NewBillingService(nil, db, planService, logger)
	manager := NewManager(nil, db, planService, billingService, nil, logger)
	ctx := context.Background()

	plan := createTestPlan("EventPlan", PlanTierStarter, 2900, 29000)
	err := planService.CreatePlan(ctx, plan)
	require.NoError(t, err)

	sub := &Subscription{
		ID:                 uuid.New(),
		CustomerID:         uuid.New(),
		PlanID:             plan.ID,
		Status:             SubscriptionStatusActive,
		BillingCycle:       BillingCycleMonthly,
		StartDate:          time.Now(),
		CurrentPeriodStart: time.Now(),
		CurrentPeriodEnd:   time.Now().Add(30 * 24 * time.Hour),
		Currency:           "USD",
	}
	err = db.Create(sub).Error
	require.NoError(t, err)

	// Create test events
	for i := 0; i < 5; i++ {
		event := &SubscriptionEvent{
			ID:             uuid.New(),
			SubscriptionID: sub.ID,
			CustomerID:     sub.CustomerID,
			EventType:      EventTypeCreated,
			Description:    "Test event",
		}
		err := db.Create(event).Error
		require.NoError(t, err)
	}

	t.Run("retrieves all events", func(t *testing.T) {
		events, err := manager.GetEvents(ctx, sub.ID, 0)
		assert.NoError(t, err)
		assert.Len(t, events, 5)
	})

	t.Run("limits events", func(t *testing.T) {
		events, err := manager.GetEvents(ctx, sub.ID, 3)
		assert.NoError(t, err)
		assert.Len(t, events, 3)
	})
}

func TestSubscription_ToResponse(t *testing.T) {
	plan := &Plan{
		ID:           uuid.New(),
		Name:         "Test Plan",
		Description:  "Test Description",
		Tier:         PlanTierStarter,
		MonthlyPrice: 2900,
		AnnualPrice:  29000,
		Currency:     "USD",
	}

	trialEnd := time.Now().Add(7 * 24 * time.Hour)
	sub := &Subscription{
		ID:                 uuid.New(),
		CustomerID:         uuid.New(),
		Plan:               plan,
		Status:             SubscriptionStatusTrialing,
		BillingCycle:       BillingCycleMonthly,
		CurrentPeriodStart: time.Now(),
		CurrentPeriodEnd:   time.Now().Add(30 * 24 * time.Hour),
		TrialEnd:           &trialEnd,
		CancelAtPeriodEnd:  false,
		CreatedAt:          time.Now(),
	}

	response := sub.ToResponse()

	assert.Equal(t, sub.ID, response.ID)
	assert.Equal(t, sub.CustomerID, response.CustomerID)
	assert.Equal(t, sub.Status, response.Status)
	assert.Equal(t, sub.BillingCycle, response.BillingCycle)
	assert.NotNil(t, response.Plan)
	assert.Equal(t, plan.Name, response.Plan.Name)
	assert.GreaterOrEqual(t, response.TrialDaysRemaining, 0)
}

func TestSubscriptionStatus_Constants(t *testing.T) {
	assert.Equal(t, SubscriptionStatus("trialing"), SubscriptionStatusTrialing)
	assert.Equal(t, SubscriptionStatus("active"), SubscriptionStatusActive)
	assert.Equal(t, SubscriptionStatus("past_due"), SubscriptionStatusPastDue)
	assert.Equal(t, SubscriptionStatus("canceled"), SubscriptionStatusCanceled)
	assert.Equal(t, SubscriptionStatus("unpaid"), SubscriptionStatusUnpaid)
	assert.Equal(t, SubscriptionStatus("incomplete"), SubscriptionStatusIncomplete)
	assert.Equal(t, SubscriptionStatus("incomplete_expired"), SubscriptionStatusIncompleteExpired)
	assert.Equal(t, SubscriptionStatus("paused"), SubscriptionStatusPaused)
}

func TestCancellationReason_Constants(t *testing.T) {
	assert.Equal(t, CancellationReason("user_requested"), CancellationReasonUserRequested)
	assert.Equal(t, CancellationReason("payment_failed"), CancellationReasonPaymentFailed)
	assert.Equal(t, CancellationReason("fraud"), CancellationReasonFraud)
	assert.Equal(t, CancellationReason("admin"), CancellationReasonAdmin)
	assert.Equal(t, CancellationReason("other"), CancellationReasonOther)
}

func TestSubscriptionEventType_Constants(t *testing.T) {
	assert.Equal(t, SubscriptionEventType("created"), EventTypeCreated)
	assert.Equal(t, SubscriptionEventType("trial_started"), EventTypeTrialStarted)
	assert.Equal(t, SubscriptionEventType("trial_ended"), EventTypeTrialEnded)
	assert.Equal(t, SubscriptionEventType("activated"), EventTypeActivated)
	assert.Equal(t, SubscriptionEventType("renewed"), EventTypeRenewed)
	assert.Equal(t, SubscriptionEventType("upgraded"), EventTypeUpgraded)
	assert.Equal(t, SubscriptionEventType("downgraded"), EventTypeDowngraded)
	assert.Equal(t, SubscriptionEventType("paused"), EventTypePaused)
	assert.Equal(t, SubscriptionEventType("resumed"), EventTypeResumed)
	assert.Equal(t, SubscriptionEventType("cancel_scheduled"), EventTypeCancelScheduled)
	assert.Equal(t, SubscriptionEventType("cancel_revoked"), EventTypeCancelRevoked)
	assert.Equal(t, SubscriptionEventType("canceled"), EventTypeCanceled)
	assert.Equal(t, SubscriptionEventType("expired"), EventTypeExpired)
	assert.Equal(t, SubscriptionEventType("payment_succeeded"), EventTypePaymentSucceeded)
	assert.Equal(t, SubscriptionEventType("payment_failed"), EventTypePaymentFailed)
}

func TestDefaultManagerConfig(t *testing.T) {
	config := DefaultManagerConfig()

	assert.True(t, config.AllowDowngrades)
	assert.True(t, config.RequirePaymentMethod)
	assert.Equal(t, 3, config.GracePeriodDays)
	assert.Equal(t, 3, config.MaxFailedPayments)
	assert.False(t, config.EnableAutoReactivation)
}

func TestSubscriptionEvent_TableName(t *testing.T) {
	event := SubscriptionEvent{}
	assert.Equal(t, "subscription_events", event.TableName())
}

func TestSubscription_TableName(t *testing.T) {
	sub := Subscription{}
	assert.Equal(t, "subscriptions", sub.TableName())
}
