package subscriptions

import (
	"context"
	"errors"
	"fmt"
	"math"
	"time"

	"github.com/google/uuid"
	"github.com/stripe/stripe-go/v76"
	"github.com/stripe/stripe-go/v76/invoice"
	"github.com/stripe/stripe-go/v76/invoiceitem"
	stripeSub "github.com/stripe/stripe-go/v76/subscription"
	"gorm.io/gorm"
)

// =============================================================================
// Errors
// =============================================================================

var (
	ErrInvoiceNotFound         = errors.New("invoice not found")
	ErrInvalidProrationDate    = errors.New("invalid proration date")
	ErrBillingNotConfigured    = errors.New("billing not configured")
	ErrUpcomingInvoiceNotFound = errors.New("upcoming invoice not found")
)

// =============================================================================
// Billing Models
// =============================================================================

// Invoice represents a billing invoice
type Invoice struct {
	ID              uuid.UUID `gorm:"type:uuid;primary_key" json:"id"`
	SubscriptionID  uuid.UUID `gorm:"type:uuid;not null;index" json:"subscription_id"`
	CustomerID      uuid.UUID `gorm:"type:uuid;not null;index" json:"customer_id"`
	StripeInvoiceID string    `gorm:"type:varchar(255);uniqueIndex" json:"stripe_invoice_id"`

	// Amounts
	Subtotal        int64  `json:"subtotal"`
	Tax             int64  `json:"tax"`
	Total           int64  `json:"total"`
	AmountDue       int64  `json:"amount_due"`
	AmountPaid      int64  `json:"amount_paid"`
	AmountRemaining int64  `json:"amount_remaining"`
	Currency        string `gorm:"type:varchar(3)" json:"currency"`

	// Status
	Status           InvoiceStatus `gorm:"type:varchar(50);not null;index" json:"status"`
	CollectionMethod string        `gorm:"type:varchar(50)" json:"collection_method"`

	// Dates
	PeriodStart time.Time  `json:"period_start"`
	PeriodEnd   time.Time  `json:"period_end"`
	DueDate     *time.Time `json:"due_date,omitempty"`
	PaidAt      *time.Time `json:"paid_at,omitempty"`
	VoidedAt    *time.Time `json:"voided_at,omitempty"`

	// URLs
	HostedInvoiceURL string `gorm:"type:varchar(500)" json:"hosted_invoice_url,omitempty"`
	InvoicePDF       string `gorm:"type:varchar(500)" json:"invoice_pdf,omitempty"`

	// Line items
	LineItems []InvoiceLineItem `gorm:"foreignKey:InvoiceID" json:"line_items,omitempty"`

	// Metadata
	Description string `gorm:"type:text" json:"description,omitempty"`
	Metadata    JSONB  `gorm:"type:jsonb" json:"metadata,omitempty"`

	// Timestamps
	CreatedAt time.Time `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt time.Time `gorm:"autoUpdateTime" json:"updated_at"`
}

// TableName specifies the table name for Invoice
func (Invoice) TableName() string {
	return "subscription_invoices"
}

// BeforeCreate sets the UUID before creating a new record
func (i *Invoice) BeforeCreate(tx *gorm.DB) error {
	if i.ID == uuid.Nil {
		i.ID = uuid.New()
	}
	return nil
}

// InvoiceStatus represents the status of an invoice
type InvoiceStatus string

const (
	InvoiceStatusDraft         InvoiceStatus = "draft"
	InvoiceStatusOpen          InvoiceStatus = "open"
	InvoiceStatusPaid          InvoiceStatus = "paid"
	InvoiceStatusUncollectible InvoiceStatus = "uncollectible"
	InvoiceStatusVoid          InvoiceStatus = "void"
)

// InvoiceLineItem represents a line item on an invoice
type InvoiceLineItem struct {
	ID          uuid.UUID `gorm:"type:uuid;primary_key" json:"id"`
	InvoiceID   uuid.UUID `gorm:"type:uuid;not null;index" json:"invoice_id"`
	Description string    `gorm:"type:varchar(500)" json:"description"`
	Quantity    int       `json:"quantity"`
	UnitAmount  int64     `json:"unit_amount"`
	Amount      int64     `json:"amount"`
	Currency    string    `gorm:"type:varchar(3)" json:"currency"`
	Proration   bool      `json:"proration"`
	PeriodStart time.Time `json:"period_start"`
	PeriodEnd   time.Time `json:"period_end"`
	CreatedAt   time.Time `gorm:"autoCreateTime" json:"created_at"`
}

// TableName specifies the table name for InvoiceLineItem
func (InvoiceLineItem) TableName() string {
	return "invoice_line_items"
}

// BeforeCreate sets the UUID before creating a new record
func (li *InvoiceLineItem) BeforeCreate(tx *gorm.DB) error {
	if li.ID == uuid.Nil {
		li.ID = uuid.New()
	}
	return nil
}

// =============================================================================
// Billing Service
// =============================================================================

// BillingServiceConfig contains configuration for the billing service
type BillingServiceConfig struct {
	TaxRate                 float64
	EnableAutomaticTax      bool
	DefaultCollectionMethod string
	GracePeriodDays         int
}

// DefaultBillingServiceConfig returns default configuration
func DefaultBillingServiceConfig() *BillingServiceConfig {
	return &BillingServiceConfig{
		TaxRate:                 0,
		EnableAutomaticTax:      false,
		DefaultCollectionMethod: "charge_automatically",
		GracePeriodDays:         7,
	}
}

// BillingService handles billing operations
type BillingService struct {
	config      *BillingServiceConfig
	db          *gorm.DB
	planService *PlanService
	logger      Logger
}

// NewBillingService creates a new billing service
func NewBillingService(config *BillingServiceConfig, db *gorm.DB, planService *PlanService, logger Logger) *BillingService {
	if config == nil {
		config = DefaultBillingServiceConfig()
	}

	return &BillingService{
		config:      config,
		db:          db,
		planService: planService,
		logger:      logger,
	}
}

// =============================================================================
// Proration Calculations
// =============================================================================

// ProrationResult contains the result of a proration calculation
type ProrationResult struct {
	CreditAmount  int64     `json:"credit_amount"`
	ChargeAmount  int64     `json:"charge_amount"`
	NetAmount     int64     `json:"net_amount"`
	DaysRemaining int       `json:"days_remaining"`
	TotalDays     int       `json:"total_days"`
	EffectiveDate time.Time `json:"effective_date"`
	Description   string    `json:"description"`
}

// CalculateProration calculates proration for a plan change
func (s *BillingService) CalculateProration(
	ctx context.Context,
	subscription *Subscription,
	newPlan *Plan,
	effectiveDate time.Time,
) (*ProrationResult, error) {
	if effectiveDate.Before(subscription.CurrentPeriodStart) || effectiveDate.After(subscription.CurrentPeriodEnd) {
		return nil, ErrInvalidProrationDate
	}

	// Calculate days
	periodDuration := subscription.CurrentPeriodEnd.Sub(subscription.CurrentPeriodStart)
	totalDays := int(periodDuration.Hours() / 24)
	if totalDays <= 0 {
		totalDays = 1
	}

	remainingDuration := subscription.CurrentPeriodEnd.Sub(effectiveDate)
	daysRemaining := int(remainingDuration.Hours() / 24)
	if daysRemaining < 0 {
		daysRemaining = 0
	}

	// Get current price
	currentPrice, err := s.planService.GetPriceForCycle(subscription.Plan, subscription.BillingCycle)
	if err != nil {
		return nil, err
	}

	// Get new price
	newPrice, err := s.planService.GetPriceForCycle(newPlan, subscription.BillingCycle)
	if err != nil {
		return nil, err
	}

	// Calculate daily rates
	currentDailyRate := float64(currentPrice) / float64(totalDays)
	newDailyRate := float64(newPrice) / float64(totalDays)

	// Calculate credit for unused time on current plan
	creditAmount := int64(math.Round(currentDailyRate * float64(daysRemaining)))

	// Calculate charge for new plan for remaining time
	chargeAmount := int64(math.Round(newDailyRate * float64(daysRemaining)))

	// Net amount (positive = charge, negative = credit)
	netAmount := chargeAmount - creditAmount

	result := &ProrationResult{
		CreditAmount:  creditAmount,
		ChargeAmount:  chargeAmount,
		NetAmount:     netAmount,
		DaysRemaining: daysRemaining,
		TotalDays:     totalDays,
		EffectiveDate: effectiveDate,
	}

	if netAmount > 0 {
		result.Description = fmt.Sprintf("Upgrade from %s to %s - prorated charge", subscription.Plan.Name, newPlan.Name)
	} else if netAmount < 0 {
		result.Description = fmt.Sprintf("Downgrade from %s to %s - prorated credit", subscription.Plan.Name, newPlan.Name)
	} else {
		result.Description = fmt.Sprintf("Plan change from %s to %s - no proration", subscription.Plan.Name, newPlan.Name)
	}

	return result, nil
}

// PreviewPlanChange previews the billing impact of a plan change
func (s *BillingService) PreviewPlanChange(ctx context.Context, subscriptionID, newPlanID uuid.UUID) (*PlanChangePreview, error) {
	// Get the Stripe subscription preview
	var sub Subscription
	if err := s.db.WithContext(ctx).
		Preload("Plan").
		Where("id = ?", subscriptionID).
		First(&sub).Error; err != nil {
		return nil, err
	}

	newPlan, err := s.planService.GetPlan(ctx, newPlanID)
	if err != nil {
		return nil, err
	}

	// Get new Stripe price ID
	newPriceID, _ := s.planService.GetStripePriceID(newPlan, sub.BillingCycle)

	// Create preview using Stripe's upcoming invoice API
	params := &stripe.InvoiceUpcomingParams{
		Customer:     stripe.String(sub.StripeCustomerID),
		Subscription: stripe.String(sub.StripeSubscriptionID),
		SubscriptionItems: []*stripe.SubscriptionItemsParams{
			{Price: stripe.String(newPriceID)},
		},
		SubscriptionProrationBehavior: stripe.String("create_prorations"),
	}

	inv, err := invoice.Upcoming(params)
	if err != nil {
		// Fall back to manual calculation
		proration, err := s.CalculateProration(ctx, &sub, newPlan, time.Now())
		if err != nil {
			return nil, err
		}

		return &PlanChangePreview{
			CurrentPlan:       sub.Plan,
			NewPlan:           newPlan,
			ImmediateCharge:   proration.NetAmount,
			NextBillingDate:   sub.CurrentPeriodEnd,
			NextBillingAmount: s.getPriceForCycle(newPlan, sub.BillingCycle),
			ProrationDetails:  proration,
		}, nil
	}

	return &PlanChangePreview{
		CurrentPlan:          sub.Plan,
		NewPlan:              newPlan,
		ImmediateCharge:      inv.AmountDue,
		NextBillingDate:      sub.CurrentPeriodEnd,
		NextBillingAmount:    s.getPriceForCycle(newPlan, sub.BillingCycle),
		StripeInvoicePreview: inv,
	}, nil
}

func (s *BillingService) getPriceForCycle(plan *Plan, cycle BillingCycle) int64 {
	price, _ := s.planService.GetPriceForCycle(plan, cycle)
	return price
}

// PlanChangePreview contains preview information for a plan change
type PlanChangePreview struct {
	CurrentPlan          *Plan            `json:"current_plan"`
	NewPlan              *Plan            `json:"new_plan"`
	ImmediateCharge      int64            `json:"immediate_charge"`
	NextBillingDate      time.Time        `json:"next_billing_date"`
	NextBillingAmount    int64            `json:"next_billing_amount"`
	ProrationDetails     *ProrationResult `json:"proration_details,omitempty"`
	StripeInvoicePreview *stripe.Invoice  `json:"-"`
}

// =============================================================================
// Invoice Operations
// =============================================================================

// GetUpcomingInvoice gets the upcoming invoice for a subscription
func (s *BillingService) GetUpcomingInvoice(ctx context.Context, subscriptionID uuid.UUID) (*UpcomingInvoice, error) {
	var sub Subscription
	if err := s.db.WithContext(ctx).
		Preload("Plan").
		Where("id = ?", subscriptionID).
		First(&sub).Error; err != nil {
		return nil, err
	}

	params := &stripe.InvoiceUpcomingParams{
		Subscription: stripe.String(sub.StripeSubscriptionID),
	}

	inv, err := invoice.Upcoming(params)
	if err != nil {
		return nil, ErrUpcomingInvoiceNotFound
	}

	lineItems := make([]UpcomingInvoiceLineItem, 0, len(inv.Lines.Data))
	for _, li := range inv.Lines.Data {
		lineItems = append(lineItems, UpcomingInvoiceLineItem{
			Description: li.Description,
			Amount:      li.Amount,
			Quantity:    int(li.Quantity),
			Proration:   li.Proration,
		})
	}

	return &UpcomingInvoice{
		Subtotal:        inv.Subtotal,
		Tax:             inv.Tax,
		Total:           inv.Total,
		AmountDue:       inv.AmountDue,
		Currency:        string(inv.Currency),
		PeriodStart:     time.Unix(inv.PeriodStart, 0),
		PeriodEnd:       time.Unix(inv.PeriodEnd, 0),
		NextPaymentDate: time.Unix(inv.NextPaymentAttempt, 0),
		LineItems:       lineItems,
	}, nil
}

// UpcomingInvoice represents an upcoming invoice
type UpcomingInvoice struct {
	Subtotal        int64                     `json:"subtotal"`
	Tax             int64                     `json:"tax"`
	Total           int64                     `json:"total"`
	AmountDue       int64                     `json:"amount_due"`
	Currency        string                    `json:"currency"`
	PeriodStart     time.Time                 `json:"period_start"`
	PeriodEnd       time.Time                 `json:"period_end"`
	NextPaymentDate time.Time                 `json:"next_payment_date"`
	LineItems       []UpcomingInvoiceLineItem `json:"line_items"`
}

// UpcomingInvoiceLineItem represents a line item in an upcoming invoice
type UpcomingInvoiceLineItem struct {
	Description string `json:"description"`
	Amount      int64  `json:"amount"`
	Quantity    int    `json:"quantity"`
	Proration   bool   `json:"proration"`
}

// CreateInvoiceFromStripe creates a local invoice record from Stripe
func (s *BillingService) CreateInvoiceFromStripe(ctx context.Context, stripeInvoice *stripe.Invoice) (*Invoice, error) {
	// Get subscription
	var sub Subscription
	if err := s.db.WithContext(ctx).
		Where("stripe_subscription_id = ?", stripeInvoice.Subscription.ID).
		First(&sub).Error; err != nil {
		return nil, fmt.Errorf("subscription not found: %w", err)
	}

	inv := &Invoice{
		SubscriptionID:   sub.ID,
		CustomerID:       sub.CustomerID,
		StripeInvoiceID:  stripeInvoice.ID,
		Subtotal:         stripeInvoice.Subtotal,
		Tax:              stripeInvoice.Tax,
		Total:            stripeInvoice.Total,
		AmountDue:        stripeInvoice.AmountDue,
		AmountPaid:       stripeInvoice.AmountPaid,
		AmountRemaining:  stripeInvoice.AmountRemaining,
		Currency:         string(stripeInvoice.Currency),
		Status:           InvoiceStatus(stripeInvoice.Status),
		CollectionMethod: string(stripeInvoice.CollectionMethod),
		PeriodStart:      time.Unix(stripeInvoice.PeriodStart, 0),
		PeriodEnd:        time.Unix(stripeInvoice.PeriodEnd, 0),
		HostedInvoiceURL: stripeInvoice.HostedInvoiceURL,
		InvoicePDF:       stripeInvoice.InvoicePDF,
	}

	if stripeInvoice.DueDate > 0 {
		dueDate := time.Unix(stripeInvoice.DueDate, 0)
		inv.DueDate = &dueDate
	}

	// Create line items
	for _, li := range stripeInvoice.Lines.Data {
		unitAmount := int64(0)
		if li.Price != nil {
			unitAmount = li.Price.UnitAmount
		}
		inv.LineItems = append(inv.LineItems, InvoiceLineItem{
			Description: li.Description,
			Quantity:    int(li.Quantity),
			UnitAmount:  unitAmount,
			Amount:      li.Amount,
			Currency:    string(li.Currency),
			Proration:   li.Proration,
			PeriodStart: time.Unix(li.Period.Start, 0),
			PeriodEnd:   time.Unix(li.Period.End, 0),
		})
	}

	if err := s.db.WithContext(ctx).Create(inv).Error; err != nil {
		return nil, fmt.Errorf("failed to create invoice: %w", err)
	}

	return inv, nil
}

// UpdateInvoiceStatus updates an invoice status from Stripe webhook
func (s *BillingService) UpdateInvoiceStatus(ctx context.Context, stripeInvoiceID string, status InvoiceStatus) error {
	var inv Invoice
	if err := s.db.WithContext(ctx).
		Where("stripe_invoice_id = ?", stripeInvoiceID).
		First(&inv).Error; err != nil {
		return err
	}

	updates := map[string]interface{}{
		"status": status,
	}

	if status == InvoiceStatusPaid {
		now := time.Now()
		updates["paid_at"] = now
	} else if status == InvoiceStatusVoid {
		now := time.Now()
		updates["voided_at"] = now
	}

	return s.db.WithContext(ctx).Model(&inv).Updates(updates).Error
}

// ListInvoices lists invoices for a subscription
func (s *BillingService) ListInvoices(ctx context.Context, subscriptionID uuid.UUID, limit, offset int) ([]*Invoice, int64, error) {
	var invoices []*Invoice
	var total int64

	query := s.db.WithContext(ctx).
		Where("subscription_id = ?", subscriptionID)

	if err := query.Model(&Invoice{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if err := query.
		Preload("LineItems").
		Order("created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&invoices).Error; err != nil {
		return nil, 0, err
	}

	return invoices, total, nil
}

// GetInvoice retrieves an invoice by ID
func (s *BillingService) GetInvoice(ctx context.Context, invoiceID uuid.UUID) (*Invoice, error) {
	var inv Invoice
	if err := s.db.WithContext(ctx).
		Preload("LineItems").
		Where("id = ?", invoiceID).
		First(&inv).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrInvoiceNotFound
		}
		return nil, err
	}

	return &inv, nil
}

// =============================================================================
// Billing Schedule
// =============================================================================

// BillingSchedule represents the billing schedule for a subscription
type BillingSchedule struct {
	CurrentPeriod   BillingPeriod   `json:"current_period"`
	UpcomingPeriods []BillingPeriod `json:"upcoming_periods"`
	BillingCycle    BillingCycle    `json:"billing_cycle"`
	NextBillingDate time.Time       `json:"next_billing_date"`
	AmountDue       int64           `json:"amount_due"`
	Currency        string          `json:"currency"`
}

// BillingPeriod represents a billing period
type BillingPeriod struct {
	StartDate time.Time `json:"start_date"`
	EndDate   time.Time `json:"end_date"`
	Amount    int64     `json:"amount"`
	IsCurrent bool      `json:"is_current"`
	IsPaid    bool      `json:"is_paid"`
}

// GetBillingSchedule gets the billing schedule for a subscription
func (s *BillingService) GetBillingSchedule(ctx context.Context, subscriptionID uuid.UUID, periodsAhead int) (*BillingSchedule, error) {
	var sub Subscription
	if err := s.db.WithContext(ctx).
		Preload("Plan").
		Where("id = ?", subscriptionID).
		First(&sub).Error; err != nil {
		return nil, err
	}

	price, _ := s.planService.GetPriceForCycle(sub.Plan, sub.BillingCycle)

	schedule := &BillingSchedule{
		CurrentPeriod: BillingPeriod{
			StartDate: sub.CurrentPeriodStart,
			EndDate:   sub.CurrentPeriodEnd,
			Amount:    price,
			IsCurrent: true,
			IsPaid:    true, // Current period is always paid
		},
		BillingCycle:    sub.BillingCycle,
		NextBillingDate: sub.CurrentPeriodEnd,
		AmountDue:       price,
		Currency:        sub.Currency,
	}

	// Calculate upcoming periods
	periodDuration := s.getPeriodDuration(sub.BillingCycle)
	currentEnd := sub.CurrentPeriodEnd

	for i := 0; i < periodsAhead; i++ {
		periodStart := currentEnd
		periodEnd := periodStart.Add(periodDuration)

		schedule.UpcomingPeriods = append(schedule.UpcomingPeriods, BillingPeriod{
			StartDate: periodStart,
			EndDate:   periodEnd,
			Amount:    price,
			IsCurrent: false,
			IsPaid:    false,
		})

		currentEnd = periodEnd
	}

	return schedule, nil
}

func (s *BillingService) getPeriodDuration(cycle BillingCycle) time.Duration {
	switch cycle {
	case BillingCycleMonthly:
		return 30 * 24 * time.Hour // Approximately 1 month
	case BillingCycleQuarterly:
		return 90 * 24 * time.Hour // Approximately 3 months
	case BillingCycleAnnual:
		return 365 * 24 * time.Hour // Approximately 1 year
	default:
		return 30 * 24 * time.Hour
	}
}

// =============================================================================
// Credits and Adjustments
// =============================================================================

// ApplyCredit applies a credit to a subscription
func (s *BillingService) ApplyCredit(ctx context.Context, subscriptionID uuid.UUID, amount int64, description string) error {
	var sub Subscription
	if err := s.db.WithContext(ctx).
		Where("id = ?", subscriptionID).
		First(&sub).Error; err != nil {
		return err
	}

	// Create a credit invoice item in Stripe
	params := &stripe.InvoiceItemParams{
		Customer:     stripe.String(sub.StripeCustomerID),
		Amount:       stripe.Int64(-amount), // Negative for credit
		Currency:     stripe.String(sub.Currency),
		Description:  stripe.String(description),
		Subscription: stripe.String(sub.StripeSubscriptionID),
	}

	_, err := invoiceitem.New(params)
	if err != nil {
		return fmt.Errorf("failed to apply credit: %w", err)
	}

	return nil
}

// ApplyCharge applies an additional charge to a subscription
func (s *BillingService) ApplyCharge(ctx context.Context, subscriptionID uuid.UUID, amount int64, description string) error {
	var sub Subscription
	if err := s.db.WithContext(ctx).
		Where("id = ?", subscriptionID).
		First(&sub).Error; err != nil {
		return err
	}

	params := &stripe.InvoiceItemParams{
		Customer:     stripe.String(sub.StripeCustomerID),
		Amount:       stripe.Int64(amount),
		Currency:     stripe.String(sub.Currency),
		Description:  stripe.String(description),
		Subscription: stripe.String(sub.StripeSubscriptionID),
	}

	_, err := invoiceitem.New(params)
	if err != nil {
		return fmt.Errorf("failed to apply charge: %w", err)
	}

	return nil
}

// =============================================================================
// Usage-based Billing
// =============================================================================

// UsageRecord represents a usage record for metered billing
type UsageRecord struct {
	ID             uuid.UUID `gorm:"type:uuid;primary_key" json:"id"`
	SubscriptionID uuid.UUID `gorm:"type:uuid;not null;index" json:"subscription_id"`
	MetricKey      string    `gorm:"type:varchar(100);not null;index" json:"metric_key"`
	Quantity       int64     `json:"quantity"`
	Timestamp      time.Time `json:"timestamp"`
	IdempotencyKey string    `gorm:"type:varchar(255);uniqueIndex" json:"idempotency_key,omitempty"`
	CreatedAt      time.Time `gorm:"autoCreateTime" json:"created_at"`
}

// TableName specifies the table name for UsageRecord
func (UsageRecord) TableName() string {
	return "subscription_usage_records"
}

// BeforeCreate sets the UUID before creating a new record
func (ur *UsageRecord) BeforeCreate(tx *gorm.DB) error {
	if ur.ID == uuid.Nil {
		ur.ID = uuid.New()
	}
	return nil
}

// RecordUsage records usage for metered billing
func (s *BillingService) RecordUsage(ctx context.Context, subscriptionID uuid.UUID, metricKey string, quantity int64, timestamp time.Time, idempotencyKey string) error {
	record := &UsageRecord{
		SubscriptionID: subscriptionID,
		MetricKey:      metricKey,
		Quantity:       quantity,
		Timestamp:      timestamp,
		IdempotencyKey: idempotencyKey,
	}

	return s.db.WithContext(ctx).Create(record).Error
}

// GetUsageSummary gets usage summary for a subscription
func (s *BillingService) GetUsageSummary(ctx context.Context, subscriptionID uuid.UUID, startDate, endDate time.Time) (map[string]int64, error) {
	type result struct {
		MetricKey     string
		TotalQuantity int64
	}

	var results []result
	if err := s.db.WithContext(ctx).
		Model(&UsageRecord{}).
		Select("metric_key, SUM(quantity) as total_quantity").
		Where("subscription_id = ? AND timestamp >= ? AND timestamp < ?", subscriptionID, startDate, endDate).
		Group("metric_key").
		Find(&results).Error; err != nil {
		return nil, err
	}

	summary := make(map[string]int64)
	for _, r := range results {
		summary[r.MetricKey] = r.TotalQuantity
	}

	return summary, nil
}

// =============================================================================
// Subscription Items (for multi-item subscriptions)
// =============================================================================

// AddSubscriptionItem adds an item to a subscription
func (s *BillingService) AddSubscriptionItem(ctx context.Context, subscriptionID uuid.UUID, priceID string, quantity int64) error {
	var sub Subscription
	if err := s.db.WithContext(ctx).
		Where("id = ?", subscriptionID).
		First(&sub).Error; err != nil {
		return err
	}

	params := &stripe.SubscriptionParams{
		Items: []*stripe.SubscriptionItemsParams{
			{
				Price:    stripe.String(priceID),
				Quantity: stripe.Int64(quantity),
			},
		},
	}

	_, err := stripeSub.Update(sub.StripeSubscriptionID, params)
	return err
}

// UpdateSubscriptionItemQuantity updates the quantity of a subscription item
func (s *BillingService) UpdateSubscriptionItemQuantity(ctx context.Context, subscriptionID uuid.UUID, itemID string, quantity int64) error {
	var sub Subscription
	if err := s.db.WithContext(ctx).
		Where("id = ?", subscriptionID).
		First(&sub).Error; err != nil {
		return err
	}

	params := &stripe.SubscriptionParams{
		Items: []*stripe.SubscriptionItemsParams{
			{
				ID:       stripe.String(itemID),
				Quantity: stripe.Int64(quantity),
			},
		},
	}

	_, err := stripeSub.Update(sub.StripeSubscriptionID, params)
	return err
}
