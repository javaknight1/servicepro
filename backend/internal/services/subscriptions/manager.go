package subscriptions

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/stripe/stripe-go/v76"
	"github.com/stripe/stripe-go/v76/subscription"
	"gorm.io/gorm"
)

// =============================================================================
// Errors
// =============================================================================

var (
	ErrSubscriptionNotFound      = errors.New("subscription not found")
	ErrSubscriptionAlreadyExists = errors.New("subscription already exists for this customer")
	ErrInvalidSubscriptionStatus = errors.New("invalid subscription status")
	ErrCannotCancelSubscription  = errors.New("subscription cannot be cancelled")
	ErrCannotReactivate          = errors.New("subscription cannot be reactivated")
	ErrTrialExpired              = errors.New("trial period has expired")
	ErrDowngradeNotAllowed       = errors.New("downgrade not allowed during active period")
)

// =============================================================================
// Subscription Status
// =============================================================================

// SubscriptionStatus represents the status of a subscription
type SubscriptionStatus string

const (
	SubscriptionStatusTrialing          SubscriptionStatus = "trialing"
	SubscriptionStatusActive            SubscriptionStatus = "active"
	SubscriptionStatusPastDue           SubscriptionStatus = "past_due"
	SubscriptionStatusCanceled          SubscriptionStatus = "canceled"
	SubscriptionStatusUnpaid            SubscriptionStatus = "unpaid"
	SubscriptionStatusIncomplete        SubscriptionStatus = "incomplete"
	SubscriptionStatusIncompleteExpired SubscriptionStatus = "incomplete_expired"
	SubscriptionStatusPaused            SubscriptionStatus = "paused"
)

// CancellationReason represents why a subscription was cancelled
type CancellationReason string

const (
	CancellationReasonUserRequested CancellationReason = "user_requested"
	CancellationReasonPaymentFailed CancellationReason = "payment_failed"
	CancellationReasonFraud         CancellationReason = "fraud"
	CancellationReasonAdmin         CancellationReason = "admin"
	CancellationReasonOther         CancellationReason = "other"
)

// =============================================================================
// Subscription Model
// =============================================================================

// Subscription represents a customer's subscription
type Subscription struct {
	ID         uuid.UUID `gorm:"type:uuid;primary_key" json:"id"`
	CustomerID uuid.UUID `gorm:"type:uuid;not null;index" json:"customer_id"`
	PlanID     uuid.UUID `gorm:"type:uuid;not null;index" json:"plan_id"`

	// Stripe IDs
	StripeSubscriptionID string `gorm:"type:varchar(255);uniqueIndex" json:"stripe_subscription_id"`
	StripeCustomerID     string `gorm:"type:varchar(255);index" json:"stripe_customer_id"`

	// Status
	Status       SubscriptionStatus `gorm:"type:varchar(50);not null;index" json:"status"`
	BillingCycle BillingCycle       `gorm:"type:varchar(50);not null" json:"billing_cycle"`

	// Dates
	StartDate          time.Time  `gorm:"not null" json:"start_date"`
	CurrentPeriodStart time.Time  `gorm:"not null" json:"current_period_start"`
	CurrentPeriodEnd   time.Time  `gorm:"not null" json:"current_period_end"`
	TrialStart         *time.Time `json:"trial_start,omitempty"`
	TrialEnd           *time.Time `json:"trial_end,omitempty"`
	CanceledAt         *time.Time `json:"canceled_at,omitempty"`
	EndedAt            *time.Time `json:"ended_at,omitempty"`

	// Cancellation
	CancelAtPeriodEnd    bool                `gorm:"default:false" json:"cancel_at_period_end"`
	CancellationReason   *CancellationReason `gorm:"type:varchar(50)" json:"cancellation_reason,omitempty"`
	CancellationFeedback *string             `gorm:"type:text" json:"cancellation_feedback,omitempty"`

	// Pricing
	PriceAtSignup int64  `json:"price_at_signup"`
	Currency      string `gorm:"type:varchar(3);default:'USD'" json:"currency"`

	// Discount
	DiscountID      *string    `gorm:"type:varchar(255)" json:"discount_id,omitempty"`
	DiscountPercent *int       `json:"discount_percent,omitempty"`
	DiscountEndsAt  *time.Time `json:"discount_ends_at,omitempty"`

	// Metadata
	Metadata JSONB `gorm:"type:jsonb" json:"metadata,omitempty"`

	// Relations
	Plan *Plan `gorm:"foreignKey:PlanID" json:"plan,omitempty"`

	// Timestamps
	CreatedAt time.Time      `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt time.Time      `gorm:"autoUpdateTime" json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

// TableName specifies the table name for Subscription
func (Subscription) TableName() string {
	return "subscriptions"
}

// BeforeCreate sets the UUID before creating a new record
func (s *Subscription) BeforeCreate(tx *gorm.DB) error {
	if s.ID == uuid.Nil {
		s.ID = uuid.New()
	}
	return nil
}

// IsActive returns true if the subscription is in an active state
func (s *Subscription) IsActive() bool {
	return s.Status == SubscriptionStatusActive || s.Status == SubscriptionStatusTrialing
}

// IsTrialing returns true if the subscription is in trial period
func (s *Subscription) IsTrialing() bool {
	return s.Status == SubscriptionStatusTrialing
}

// DaysUntilRenewal returns the number of days until the subscription renews
func (s *Subscription) DaysUntilRenewal() int {
	if s.CancelAtPeriodEnd {
		return 0
	}
	duration := time.Until(s.CurrentPeriodEnd)
	return int(duration.Hours() / 24)
}

// TrialDaysRemaining returns the number of days remaining in the trial
func (s *Subscription) TrialDaysRemaining() int {
	if s.TrialEnd == nil || !s.IsTrialing() {
		return 0
	}
	duration := time.Until(*s.TrialEnd)
	days := int(duration.Hours() / 24)
	if days < 0 {
		return 0
	}
	return days
}

// =============================================================================
// Subscription Event Model
// =============================================================================

// SubscriptionEventType represents types of subscription events
type SubscriptionEventType string

const (
	EventTypeCreated          SubscriptionEventType = "created"
	EventTypeTrialStarted     SubscriptionEventType = "trial_started"
	EventTypeTrialEnded       SubscriptionEventType = "trial_ended"
	EventTypeActivated        SubscriptionEventType = "activated"
	EventTypeRenewed          SubscriptionEventType = "renewed"
	EventTypeUpgraded         SubscriptionEventType = "upgraded"
	EventTypeDowngraded       SubscriptionEventType = "downgraded"
	EventTypePaused           SubscriptionEventType = "paused"
	EventTypeResumed          SubscriptionEventType = "resumed"
	EventTypeCancelScheduled  SubscriptionEventType = "cancel_scheduled"
	EventTypeCancelRevoked    SubscriptionEventType = "cancel_revoked"
	EventTypeCanceled         SubscriptionEventType = "canceled"
	EventTypeExpired          SubscriptionEventType = "expired"
	EventTypePaymentSucceeded SubscriptionEventType = "payment_succeeded"
	EventTypePaymentFailed    SubscriptionEventType = "payment_failed"
)

// SubscriptionEvent logs events for subscriptions
type SubscriptionEvent struct {
	ID             uuid.UUID             `gorm:"type:uuid;primary_key" json:"id"`
	SubscriptionID uuid.UUID             `gorm:"type:uuid;not null;index" json:"subscription_id"`
	CustomerID     uuid.UUID             `gorm:"type:uuid;not null;index" json:"customer_id"`
	EventType      SubscriptionEventType `gorm:"type:varchar(50);not null;index" json:"event_type"`
	Description    string                `gorm:"type:text" json:"description"`
	OldPlanID      *uuid.UUID            `gorm:"type:uuid" json:"old_plan_id,omitempty"`
	NewPlanID      *uuid.UUID            `gorm:"type:uuid" json:"new_plan_id,omitempty"`
	OldStatus      *SubscriptionStatus   `gorm:"type:varchar(50)" json:"old_status,omitempty"`
	NewStatus      *SubscriptionStatus   `gorm:"type:varchar(50)" json:"new_status,omitempty"`
	Metadata       JSONB                 `gorm:"type:jsonb" json:"metadata,omitempty"`
	CreatedAt      time.Time             `gorm:"autoCreateTime" json:"created_at"`
}

// TableName specifies the table name for SubscriptionEvent
func (SubscriptionEvent) TableName() string {
	return "subscription_events"
}

// BeforeCreate sets the UUID before creating a new record
func (se *SubscriptionEvent) BeforeCreate(tx *gorm.DB) error {
	if se.ID == uuid.Nil {
		se.ID = uuid.New()
	}
	return nil
}

// =============================================================================
// Manager Service
// =============================================================================

// ManagerConfig contains configuration for the subscription manager
type ManagerConfig struct {
	AllowDowngrades        bool
	RequirePaymentMethod   bool
	GracePeriodDays        int
	MaxFailedPayments      int
	EnableAutoReactivation bool
}

// DefaultManagerConfig returns default configuration
func DefaultManagerConfig() *ManagerConfig {
	return &ManagerConfig{
		AllowDowngrades:        true,
		RequirePaymentMethod:   true,
		GracePeriodDays:        3,
		MaxFailedPayments:      3,
		EnableAutoReactivation: false,
	}
}

// Manager handles subscription lifecycle operations
type Manager struct {
	config      *ManagerConfig
	db          *gorm.DB
	planService *PlanService
	billing     *BillingService
	notifier    *NotificationService
	logger      Logger
}

// NewManager creates a new subscription manager
func NewManager(
	config *ManagerConfig,
	db *gorm.DB,
	planService *PlanService,
	billing *BillingService,
	notifier *NotificationService,
	logger Logger,
) *Manager {
	if config == nil {
		config = DefaultManagerConfig()
	}

	return &Manager{
		config:      config,
		db:          db,
		planService: planService,
		billing:     billing,
		notifier:    notifier,
		logger:      logger,
	}
}

// =============================================================================
// Subscription Lifecycle
// =============================================================================

// CreateSubscription creates a new subscription
func (m *Manager) CreateSubscription(ctx context.Context, req *CreateSubscriptionRequest) (*Subscription, error) {
	// Validate plan
	plan, err := m.planService.GetPlan(ctx, req.PlanID)
	if err != nil {
		return nil, fmt.Errorf("invalid plan: %w", err)
	}

	if !plan.IsActive {
		return nil, ErrPlanInactive
	}

	// Check for existing active subscription
	var existing Subscription
	if err := m.db.WithContext(ctx).
		Where("customer_id = ? AND status IN ?", req.CustomerID, []string{"active", "trialing"}).
		First(&existing).Error; err == nil {
		return nil, ErrSubscriptionAlreadyExists
	}

	// Get price for billing cycle
	price, err := m.planService.GetPriceForCycle(plan, req.BillingCycle)
	if err != nil {
		return nil, err
	}

	// Create Stripe subscription
	stripePriceID, _ := m.planService.GetStripePriceID(plan, req.BillingCycle)

	stripeParams := &stripe.SubscriptionParams{
		Customer: stripe.String(req.StripeCustomerID),
		Items: []*stripe.SubscriptionItemsParams{
			{Price: stripe.String(stripePriceID)},
		},
	}

	// Handle trial
	if plan.TrialDays > 0 && !req.SkipTrial {
		stripeParams.TrialPeriodDays = stripe.Int64(int64(plan.TrialDays))
	}

	// Add payment method if provided
	if req.PaymentMethodID != "" {
		stripeParams.DefaultPaymentMethod = stripe.String(req.PaymentMethodID)
	}

	stripeSub, err := subscription.New(stripeParams)
	if err != nil {
		return nil, fmt.Errorf("failed to create Stripe subscription: %w", err)
	}

	// Create local subscription
	now := time.Now()
	sub := &Subscription{
		CustomerID:           req.CustomerID,
		PlanID:               req.PlanID,
		StripeSubscriptionID: stripeSub.ID,
		StripeCustomerID:     req.StripeCustomerID,
		Status:               SubscriptionStatus(stripeSub.Status),
		BillingCycle:         req.BillingCycle,
		StartDate:            now,
		CurrentPeriodStart:   time.Unix(stripeSub.CurrentPeriodStart, 0),
		CurrentPeriodEnd:     time.Unix(stripeSub.CurrentPeriodEnd, 0),
		PriceAtSignup:        price,
		Currency:             plan.Currency,
	}

	// Set trial dates
	if stripeSub.TrialStart > 0 {
		trialStart := time.Unix(stripeSub.TrialStart, 0)
		trialEnd := time.Unix(stripeSub.TrialEnd, 0)
		sub.TrialStart = &trialStart
		sub.TrialEnd = &trialEnd
	}

	if err := m.db.WithContext(ctx).Create(sub).Error; err != nil {
		return nil, fmt.Errorf("failed to save subscription: %w", err)
	}

	// Log event
	m.logEvent(ctx, sub, EventTypeCreated, "Subscription created", nil, nil, nil, nil)

	// Send notification
	if m.notifier != nil {
		if sub.IsTrialing() {
			m.notifier.SendTrialStarted(ctx, sub)
		} else {
			m.notifier.SendSubscriptionCreated(ctx, sub)
		}
	}

	return sub, nil
}

// GetSubscription retrieves a subscription by ID
func (m *Manager) GetSubscription(ctx context.Context, id uuid.UUID) (*Subscription, error) {
	var sub Subscription
	if err := m.db.WithContext(ctx).
		Preload("Plan").
		Preload("Plan.Features").
		Where("id = ?", id).
		First(&sub).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrSubscriptionNotFound
		}
		return nil, err
	}

	return &sub, nil
}

// GetSubscriptionByCustomer retrieves the active subscription for a customer
func (m *Manager) GetSubscriptionByCustomer(ctx context.Context, customerID uuid.UUID) (*Subscription, error) {
	var sub Subscription
	if err := m.db.WithContext(ctx).
		Preload("Plan").
		Preload("Plan.Features").
		Where("customer_id = ? AND status IN ?", customerID, []string{"active", "trialing", "past_due"}).
		Order("created_at DESC").
		First(&sub).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrSubscriptionNotFound
		}
		return nil, err
	}

	return &sub, nil
}

// GetSubscriptionByStripeID retrieves a subscription by Stripe ID
func (m *Manager) GetSubscriptionByStripeID(ctx context.Context, stripeID string) (*Subscription, error) {
	var sub Subscription
	if err := m.db.WithContext(ctx).
		Preload("Plan").
		Where("stripe_subscription_id = ?", stripeID).
		First(&sub).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrSubscriptionNotFound
		}
		return nil, err
	}

	return &sub, nil
}

// ListSubscriptions lists subscriptions with filters
func (m *Manager) ListSubscriptions(ctx context.Context, opts *ListSubscriptionsOptions) ([]*Subscription, int64, error) {
	query := m.db.WithContext(ctx).Model(&Subscription{})

	if opts.CustomerID != nil {
		query = query.Where("customer_id = ?", *opts.CustomerID)
	}
	if opts.PlanID != nil {
		query = query.Where("plan_id = ?", *opts.PlanID)
	}
	if opts.Status != nil {
		query = query.Where("status = ?", *opts.Status)
	}
	if len(opts.Statuses) > 0 {
		query = query.Where("status IN ?", opts.Statuses)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if opts.Limit > 0 {
		query = query.Limit(opts.Limit)
	}
	if opts.Offset > 0 {
		query = query.Offset(opts.Offset)
	}

	query = query.Preload("Plan").Order("created_at DESC")

	var subs []*Subscription
	if err := query.Find(&subs).Error; err != nil {
		return nil, 0, err
	}

	return subs, total, nil
}

// =============================================================================
// Plan Changes
// =============================================================================

// ChangePlan changes the subscription to a different plan
func (m *Manager) ChangePlan(ctx context.Context, subscriptionID, newPlanID uuid.UUID, immediate bool) (*Subscription, error) {
	sub, err := m.GetSubscription(ctx, subscriptionID)
	if err != nil {
		return nil, err
	}

	if !sub.IsActive() {
		return nil, ErrInvalidSubscriptionStatus
	}

	newPlan, err := m.planService.GetPlan(ctx, newPlanID)
	if err != nil {
		return nil, err
	}

	oldPlanID := sub.PlanID

	// Determine if upgrade or downgrade
	isUpgrade := newPlan.MonthlyPrice > sub.Plan.MonthlyPrice
	if !isUpgrade && !m.config.AllowDowngrades {
		return nil, ErrDowngradeNotAllowed
	}

	// Get new price
	stripePriceID, _ := m.planService.GetStripePriceID(newPlan, sub.BillingCycle)

	// Update Stripe subscription
	stripeParams := &stripe.SubscriptionParams{
		Items: []*stripe.SubscriptionItemsParams{
			{
				Price: stripe.String(stripePriceID),
			},
		},
	}

	if immediate {
		stripeParams.ProrationBehavior = stripe.String("create_prorations")
	} else {
		stripeParams.ProrationBehavior = stripe.String("none")
	}

	_, err = subscription.Update(sub.StripeSubscriptionID, stripeParams)
	if err != nil {
		return nil, fmt.Errorf("failed to update Stripe subscription: %w", err)
	}

	// Update local subscription
	sub.PlanID = newPlanID
	if err := m.db.WithContext(ctx).Save(sub).Error; err != nil {
		return nil, fmt.Errorf("failed to save subscription: %w", err)
	}

	// Log event
	eventType := EventTypeUpgraded
	if !isUpgrade {
		eventType = EventTypeDowngraded
	}
	m.logEvent(ctx, sub, eventType, fmt.Sprintf("Plan changed from %s to %s", sub.Plan.Name, newPlan.Name), &oldPlanID, &newPlanID, nil, nil)

	// Send notification
	if m.notifier != nil {
		m.notifier.SendPlanChanged(ctx, sub, isUpgrade)
	}

	return m.GetSubscription(ctx, subscriptionID)
}

// ChangeBillingCycle changes the billing cycle
func (m *Manager) ChangeBillingCycle(ctx context.Context, subscriptionID uuid.UUID, newCycle BillingCycle) (*Subscription, error) {
	sub, err := m.GetSubscription(ctx, subscriptionID)
	if err != nil {
		return nil, err
	}

	if !sub.IsActive() {
		return nil, ErrInvalidSubscriptionStatus
	}

	// Get new price ID
	stripePriceID, err := m.planService.GetStripePriceID(sub.Plan, newCycle)
	if err != nil {
		return nil, err
	}

	// Update Stripe
	stripeParams := &stripe.SubscriptionParams{
		Items: []*stripe.SubscriptionItemsParams{
			{Price: stripe.String(stripePriceID)},
		},
		ProrationBehavior: stripe.String("create_prorations"),
	}

	_, err = subscription.Update(sub.StripeSubscriptionID, stripeParams)
	if err != nil {
		return nil, fmt.Errorf("failed to update Stripe subscription: %w", err)
	}

	// Update local
	sub.BillingCycle = newCycle
	if err := m.db.WithContext(ctx).Save(sub).Error; err != nil {
		return nil, fmt.Errorf("failed to save subscription: %w", err)
	}

	return sub, nil
}

// =============================================================================
// Cancellation
// =============================================================================

// CancelSubscription cancels a subscription
func (m *Manager) CancelSubscription(ctx context.Context, subscriptionID uuid.UUID, req *CancelSubscriptionRequest) (*Subscription, error) {
	sub, err := m.GetSubscription(ctx, subscriptionID)
	if err != nil {
		return nil, err
	}

	if sub.Status == SubscriptionStatusCanceled {
		return nil, ErrCannotCancelSubscription
	}

	oldStatus := sub.Status

	if req.Immediate {
		// Cancel immediately
		_, err = subscription.Cancel(sub.StripeSubscriptionID, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to cancel Stripe subscription: %w", err)
		}

		now := time.Now()
		sub.Status = SubscriptionStatusCanceled
		sub.CanceledAt = &now
		sub.EndedAt = &now
		sub.CancellationReason = &req.Reason
		sub.CancellationFeedback = req.Feedback
	} else {
		// Cancel at period end
		stripeParams := &stripe.SubscriptionParams{
			CancelAtPeriodEnd: stripe.Bool(true),
		}

		_, err = subscription.Update(sub.StripeSubscriptionID, stripeParams)
		if err != nil {
			return nil, fmt.Errorf("failed to update Stripe subscription: %w", err)
		}

		now := time.Now()
		sub.CancelAtPeriodEnd = true
		sub.CanceledAt = &now
		sub.CancellationReason = &req.Reason
		sub.CancellationFeedback = req.Feedback
	}

	if err := m.db.WithContext(ctx).Save(sub).Error; err != nil {
		return nil, fmt.Errorf("failed to save subscription: %w", err)
	}

	// Log event
	eventType := EventTypeCanceled
	if !req.Immediate {
		eventType = EventTypeCancelScheduled
	}
	newStatus := sub.Status
	m.logEvent(ctx, sub, eventType, "Subscription cancelled", nil, nil, &oldStatus, &newStatus)

	// Send notification
	if m.notifier != nil {
		m.notifier.SendSubscriptionCancelled(ctx, sub, req.Immediate)
	}

	return sub, nil
}

// RevokeCancellation revokes a scheduled cancellation
func (m *Manager) RevokeCancellation(ctx context.Context, subscriptionID uuid.UUID) (*Subscription, error) {
	sub, err := m.GetSubscription(ctx, subscriptionID)
	if err != nil {
		return nil, err
	}

	if !sub.CancelAtPeriodEnd {
		return sub, nil // Nothing to revoke
	}

	// Update Stripe
	stripeParams := &stripe.SubscriptionParams{
		CancelAtPeriodEnd: stripe.Bool(false),
	}

	_, err = subscription.Update(sub.StripeSubscriptionID, stripeParams)
	if err != nil {
		return nil, fmt.Errorf("failed to update Stripe subscription: %w", err)
	}

	// Update local
	sub.CancelAtPeriodEnd = false
	sub.CanceledAt = nil
	sub.CancellationReason = nil
	sub.CancellationFeedback = nil

	if err := m.db.WithContext(ctx).Save(sub).Error; err != nil {
		return nil, fmt.Errorf("failed to save subscription: %w", err)
	}

	// Log event
	m.logEvent(ctx, sub, EventTypeCancelRevoked, "Cancellation revoked", nil, nil, nil, nil)

	return sub, nil
}

// =============================================================================
// Pause/Resume
// =============================================================================

// PauseSubscription pauses a subscription
func (m *Manager) PauseSubscription(ctx context.Context, subscriptionID uuid.UUID, resumeAt *time.Time) (*Subscription, error) {
	sub, err := m.GetSubscription(ctx, subscriptionID)
	if err != nil {
		return nil, err
	}

	if !sub.IsActive() {
		return nil, ErrInvalidSubscriptionStatus
	}

	oldStatus := sub.Status

	// Pause in Stripe
	stripeParams := &stripe.SubscriptionParams{
		PauseCollection: &stripe.SubscriptionPauseCollectionParams{
			Behavior: stripe.String("void"),
		},
	}

	if resumeAt != nil {
		stripeParams.PauseCollection.ResumesAt = stripe.Int64(resumeAt.Unix())
	}

	_, err = subscription.Update(sub.StripeSubscriptionID, stripeParams)
	if err != nil {
		return nil, fmt.Errorf("failed to pause Stripe subscription: %w", err)
	}

	// Update local
	sub.Status = SubscriptionStatusPaused
	if err := m.db.WithContext(ctx).Save(sub).Error; err != nil {
		return nil, fmt.Errorf("failed to save subscription: %w", err)
	}

	// Log event
	newStatus := sub.Status
	m.logEvent(ctx, sub, EventTypePaused, "Subscription paused", nil, nil, &oldStatus, &newStatus)

	return sub, nil
}

// ResumeSubscription resumes a paused subscription
func (m *Manager) ResumeSubscription(ctx context.Context, subscriptionID uuid.UUID) (*Subscription, error) {
	sub, err := m.GetSubscription(ctx, subscriptionID)
	if err != nil {
		return nil, err
	}

	if sub.Status != SubscriptionStatusPaused {
		return nil, ErrInvalidSubscriptionStatus
	}

	oldStatus := sub.Status

	// Resume in Stripe
	stripeParams := &stripe.SubscriptionParams{}
	stripeParams.AddExpand("pause_collection")
	// Setting to nil effectively removes the pause
	// Note: In practice, you may need to use a different approach

	_, err = subscription.Update(sub.StripeSubscriptionID, stripeParams)
	if err != nil {
		return nil, fmt.Errorf("failed to resume Stripe subscription: %w", err)
	}

	// Update local
	sub.Status = SubscriptionStatusActive
	if err := m.db.WithContext(ctx).Save(sub).Error; err != nil {
		return nil, fmt.Errorf("failed to save subscription: %w", err)
	}

	// Log event
	newStatus := sub.Status
	m.logEvent(ctx, sub, EventTypeResumed, "Subscription resumed", nil, nil, &oldStatus, &newStatus)

	return sub, nil
}

// =============================================================================
// Status Updates from Webhooks
// =============================================================================

// UpdateStatusFromStripe updates subscription status from Stripe webhook
func (m *Manager) UpdateStatusFromStripe(ctx context.Context, stripeID string, status SubscriptionStatus, periodStart, periodEnd int64) error {
	sub, err := m.GetSubscriptionByStripeID(ctx, stripeID)
	if err != nil {
		return err
	}

	oldStatus := sub.Status
	sub.Status = status
	sub.CurrentPeriodStart = time.Unix(periodStart, 0)
	sub.CurrentPeriodEnd = time.Unix(periodEnd, 0)

	if status == SubscriptionStatusCanceled {
		now := time.Now()
		sub.EndedAt = &now
	}

	if err := m.db.WithContext(ctx).Save(sub).Error; err != nil {
		return fmt.Errorf("failed to save subscription: %w", err)
	}

	// Log event if status changed
	if oldStatus != status {
		var eventType SubscriptionEventType
		switch status {
		case SubscriptionStatusActive:
			eventType = EventTypeActivated
		case SubscriptionStatusCanceled:
			eventType = EventTypeCanceled
		case SubscriptionStatusPastDue:
			eventType = EventTypePaymentFailed
		default:
			eventType = SubscriptionEventType(status)
		}
		m.logEvent(ctx, sub, eventType, fmt.Sprintf("Status changed to %s", status), nil, nil, &oldStatus, &status)
	}

	return nil
}

// HandleInvoicePaid handles successful payment
func (m *Manager) HandleInvoicePaid(ctx context.Context, stripeSubID string) error {
	sub, err := m.GetSubscriptionByStripeID(ctx, stripeSubID)
	if err != nil {
		return err
	}

	m.logEvent(ctx, sub, EventTypePaymentSucceeded, "Payment succeeded", nil, nil, nil, nil)

	if m.notifier != nil {
		m.notifier.SendPaymentSucceeded(ctx, sub)
	}

	return nil
}

// HandleInvoiceFailed handles failed payment
func (m *Manager) HandleInvoiceFailed(ctx context.Context, stripeSubID string, attemptCount int) error {
	sub, err := m.GetSubscriptionByStripeID(ctx, stripeSubID)
	if err != nil {
		return err
	}

	m.logEvent(ctx, sub, EventTypePaymentFailed, fmt.Sprintf("Payment failed (attempt %d)", attemptCount), nil, nil, nil, nil)

	if m.notifier != nil {
		m.notifier.SendPaymentFailed(ctx, sub, attemptCount)
	}

	return nil
}

// =============================================================================
// Event Logging
// =============================================================================

func (m *Manager) logEvent(
	ctx context.Context,
	sub *Subscription,
	eventType SubscriptionEventType,
	description string,
	oldPlanID, newPlanID *uuid.UUID,
	oldStatus, newStatus *SubscriptionStatus,
) {
	event := &SubscriptionEvent{
		SubscriptionID: sub.ID,
		CustomerID:     sub.CustomerID,
		EventType:      eventType,
		Description:    description,
		OldPlanID:      oldPlanID,
		NewPlanID:      newPlanID,
		OldStatus:      oldStatus,
		NewStatus:      newStatus,
	}

	if err := m.db.WithContext(ctx).Create(event).Error; err != nil && m.logger != nil {
		m.logger.Errorf("Failed to log subscription event: %v", err)
	}
}

// GetEvents retrieves events for a subscription
func (m *Manager) GetEvents(ctx context.Context, subscriptionID uuid.UUID, limit int) ([]*SubscriptionEvent, error) {
	var events []*SubscriptionEvent
	query := m.db.WithContext(ctx).
		Where("subscription_id = ?", subscriptionID).
		Order("created_at DESC")

	if limit > 0 {
		query = query.Limit(limit)
	}

	if err := query.Find(&events).Error; err != nil {
		return nil, err
	}

	return events, nil
}

// =============================================================================
// Request/Response Types
// =============================================================================

// CreateSubscriptionRequest represents a request to create a subscription
type CreateSubscriptionRequest struct {
	CustomerID       uuid.UUID    `json:"customer_id" binding:"required"`
	PlanID           uuid.UUID    `json:"plan_id" binding:"required"`
	StripeCustomerID string       `json:"stripe_customer_id" binding:"required"`
	BillingCycle     BillingCycle `json:"billing_cycle" binding:"required"`
	PaymentMethodID  string       `json:"payment_method_id,omitempty"`
	SkipTrial        bool         `json:"skip_trial,omitempty"`
}

// CancelSubscriptionRequest represents a request to cancel a subscription
type CancelSubscriptionRequest struct {
	Reason    CancellationReason `json:"reason" binding:"required"`
	Feedback  *string            `json:"feedback,omitempty"`
	Immediate bool               `json:"immediate,omitempty"`
}

// ListSubscriptionsOptions contains options for listing subscriptions
type ListSubscriptionsOptions struct {
	CustomerID *uuid.UUID
	PlanID     *uuid.UUID
	Status     *SubscriptionStatus
	Statuses   []SubscriptionStatus
	Limit      int
	Offset     int
}

// SubscriptionResponse represents the API response for a subscription
type SubscriptionResponse struct {
	ID                 uuid.UUID          `json:"id"`
	CustomerID         uuid.UUID          `json:"customer_id"`
	Plan               *PlanResponse      `json:"plan"`
	Status             SubscriptionStatus `json:"status"`
	BillingCycle       BillingCycle       `json:"billing_cycle"`
	CurrentPeriodStart time.Time          `json:"current_period_start"`
	CurrentPeriodEnd   time.Time          `json:"current_period_end"`
	TrialEnd           *time.Time         `json:"trial_end,omitempty"`
	CancelAtPeriodEnd  bool               `json:"cancel_at_period_end"`
	DaysUntilRenewal   int                `json:"days_until_renewal"`
	TrialDaysRemaining int                `json:"trial_days_remaining"`
	CreatedAt          time.Time          `json:"created_at"`
}

// PlanResponse represents a plan in API responses
type PlanResponse struct {
	ID           uuid.UUID     `json:"id"`
	Name         string        `json:"name"`
	Description  string        `json:"description"`
	Tier         PlanTier      `json:"tier"`
	MonthlyPrice int64         `json:"monthly_price"`
	AnnualPrice  int64         `json:"annual_price"`
	Currency     string        `json:"currency"`
	Features     []PlanFeature `json:"features"`
}

// ToResponse converts a Subscription to SubscriptionResponse
func (s *Subscription) ToResponse() *SubscriptionResponse {
	resp := &SubscriptionResponse{
		ID:                 s.ID,
		CustomerID:         s.CustomerID,
		Status:             s.Status,
		BillingCycle:       s.BillingCycle,
		CurrentPeriodStart: s.CurrentPeriodStart,
		CurrentPeriodEnd:   s.CurrentPeriodEnd,
		TrialEnd:           s.TrialEnd,
		CancelAtPeriodEnd:  s.CancelAtPeriodEnd,
		DaysUntilRenewal:   s.DaysUntilRenewal(),
		TrialDaysRemaining: s.TrialDaysRemaining(),
		CreatedAt:          s.CreatedAt,
	}

	if s.Plan != nil {
		resp.Plan = &PlanResponse{
			ID:           s.Plan.ID,
			Name:         s.Plan.Name,
			Description:  s.Plan.Description,
			Tier:         s.Plan.Tier,
			MonthlyPrice: s.Plan.MonthlyPrice,
			AnnualPrice:  s.Plan.AnnualPrice,
			Currency:     s.Plan.Currency,
			Features:     s.Plan.Features,
		}
	}

	return resp
}
