package services

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
	"github.com/stripe/stripe-go/v76"
	"github.com/stripe/stripe-go/v76/customer"
	"github.com/stripe/stripe-go/v76/invoice"
	"github.com/stripe/stripe-go/v76/paymentmethod"
	"github.com/stripe/stripe-go/v76/setupintent"
	"github.com/stripe/stripe-go/v76/subscription"

	"github.com/javaknight1/servicepro/backend/config"
	"github.com/javaknight1/servicepro/backend/internal/models"
	"github.com/javaknight1/servicepro/backend/internal/repository"
)

var (
	ErrStripeNotConfigured    = errors.New("stripe is not configured")
	ErrStripeCustomerNotFound = errors.New("stripe customer not found")
	ErrNoDefaultPaymentMethod = errors.New("no default payment method set")
	ErrPaymentMethodNotFound  = errors.New("payment method not found")
	ErrSubscriptionNotFound   = errors.New("subscription not found")
	ErrInvalidBillingCycle    = errors.New("invalid billing cycle")
)

// StripeService handles Stripe API interactions
type StripeService struct {
	config         *config.StripeConfig
	tenantRepo     *repository.TenantRepository
	membershipRepo *repository.MembershipRepository
	paymentRepo    *repository.PaymentRepository
}

// NewStripeService creates a new Stripe service
func NewStripeService(
	cfg *config.StripeConfig,
	tenantRepo *repository.TenantRepository,
	membershipRepo *repository.MembershipRepository,
	paymentRepo *repository.PaymentRepository,
) *StripeService {
	// Initialize Stripe with secret key
	if cfg.SecretKey != "" {
		stripe.Key = cfg.SecretKey
	}

	return &StripeService{
		config:         cfg,
		tenantRepo:     tenantRepo,
		membershipRepo: membershipRepo,
		paymentRepo:    paymentRepo,
	}
}

// IsConfigured checks if Stripe is properly configured
func (s *StripeService) IsConfigured() bool {
	return s.config.SecretKey != ""
}

// ============================================================================
// Customer Management
// ============================================================================

// GetOrCreateStripeCustomer gets existing or creates new Stripe customer for tenant
func (s *StripeService) GetOrCreateStripeCustomer(ctx context.Context, tenantID uuid.UUID) (string, error) {
	if !s.IsConfigured() {
		return "", ErrStripeNotConfigured
	}

	// Get tenant
	tenant, err := s.tenantRepo.GetByID(ctx, tenantID)
	if err != nil {
		return "", fmt.Errorf("failed to get tenant: %w", err)
	}

	// Return existing customer ID if present
	if tenant.StripeCustomerID != nil && *tenant.StripeCustomerID != "" {
		return *tenant.StripeCustomerID, nil
	}

	// Create new Stripe customer
	params := &stripe.CustomerParams{
		Name:  stripe.String(tenant.Name),
		Email: stripe.String(tenant.Email),
		Metadata: map[string]string{
			"tenant_id": tenantID.String(),
		},
	}

	cust, err := customer.New(params)
	if err != nil {
		return "", fmt.Errorf("failed to create Stripe customer: %w", err)
	}

	// Save customer ID to tenant
	if err := s.tenantRepo.UpdateStripeCustomerID(ctx, tenantID, cust.ID); err != nil {
		return "", fmt.Errorf("failed to save Stripe customer ID: %w", err)
	}

	log.Printf("[STRIPE] Created customer %s for tenant %s", cust.ID, tenantID)
	return cust.ID, nil
}

// ============================================================================
// Setup Intent (for adding payment methods)
// ============================================================================

// CreateSetupIntent creates a setup intent for adding a payment method
func (s *StripeService) CreateSetupIntent(ctx context.Context, tenantID uuid.UUID) (*models.CreateSetupIntentResponse, error) {
	if !s.IsConfigured() {
		return nil, ErrStripeNotConfigured
	}

	// Get or create Stripe customer
	customerID, err := s.GetOrCreateStripeCustomer(ctx, tenantID)
	if err != nil {
		return nil, err
	}

	// Create setup intent
	params := &stripe.SetupIntentParams{
		Customer: stripe.String(customerID),
		PaymentMethodTypes: stripe.StringSlice([]string{
			"card",
		}),
		Metadata: map[string]string{
			"tenant_id": tenantID.String(),
		},
	}

	si, err := setupintent.New(params)
	if err != nil {
		return nil, fmt.Errorf("failed to create setup intent: %w", err)
	}

	log.Printf("[STRIPE] Created setup intent %s for tenant %s", si.ID, tenantID)
	return &models.CreateSetupIntentResponse{
		ClientSecret: si.ClientSecret,
	}, nil
}

// ============================================================================
// Payment Method Management
// ============================================================================

// AddPaymentMethod adds a payment method to a tenant after setup intent confirmation
func (s *StripeService) AddPaymentMethod(ctx context.Context, tenantID uuid.UUID, stripePaymentMethodID string, setAsDefault bool) (*models.SavedPaymentMethodResponse, error) {
	if !s.IsConfigured() {
		return nil, ErrStripeNotConfigured
	}

	// Get the payment method from Stripe to get card details
	pm, err := paymentmethod.Get(stripePaymentMethodID, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get payment method from Stripe: %w", err)
	}

	// Ensure payment method is a card
	if pm.Card == nil {
		return nil, fmt.Errorf("payment method is not a card")
	}

	// Get or create Stripe customer and attach payment method
	customerID, err := s.GetOrCreateStripeCustomer(ctx, tenantID)
	if err != nil {
		return nil, err
	}

	// Attach payment method to customer
	attachParams := &stripe.PaymentMethodAttachParams{
		Customer: stripe.String(customerID),
	}
	_, err = paymentmethod.Attach(stripePaymentMethodID, attachParams)
	if err != nil {
		return nil, fmt.Errorf("failed to attach payment method: %w", err)
	}

	// Create saved payment method record
	savedPM := &models.SavedPaymentMethod{
		ID:                    uuid.New(),
		TenantID:              tenantID,
		StripePaymentMethodID: stripePaymentMethodID,
		CardBrand:             string(pm.Card.Brand),
		CardLastFour:          pm.Card.Last4,
		CardExpMonth:          int(pm.Card.ExpMonth),
		CardExpYear:           int(pm.Card.ExpYear),
		IsDefault:             false,
	}

	if err := s.paymentRepo.CreateSavedPaymentMethod(ctx, savedPM); err != nil {
		return nil, fmt.Errorf("failed to save payment method: %w", err)
	}

	// Set as default if requested or if it's the first payment method
	if setAsDefault {
		if err := s.SetDefaultPaymentMethod(ctx, tenantID, savedPM.ID); err != nil {
			log.Printf("[STRIPE] Warning: failed to set default payment method: %v", err)
		}
		savedPM.IsDefault = true
	} else {
		// Check if this is the first payment method
		methods, err := s.paymentRepo.GetSavedPaymentMethodsByTenantID(ctx, tenantID)
		if err == nil && len(methods) == 1 {
			if err := s.SetDefaultPaymentMethod(ctx, tenantID, savedPM.ID); err != nil {
				log.Printf("[STRIPE] Warning: failed to set default payment method: %v", err)
			}
			savedPM.IsDefault = true
		}
	}

	log.Printf("[STRIPE] Added payment method %s for tenant %s", savedPM.ID, tenantID)
	response := savedPM.ToSavedPaymentMethodResponse()
	return &response, nil
}

// GetPaymentMethods retrieves all saved payment methods for a tenant
func (s *StripeService) GetPaymentMethods(ctx context.Context, tenantID uuid.UUID) (*models.ListSavedPaymentMethodsResponse, error) {
	methods, err := s.paymentRepo.GetSavedPaymentMethodsByTenantID(ctx, tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to get payment methods: %w", err)
	}

	responses := make([]models.SavedPaymentMethodResponse, len(methods))
	for i, m := range methods {
		responses[i] = m.ToSavedPaymentMethodResponse()
	}

	return &models.ListSavedPaymentMethodsResponse{
		PaymentMethods: responses,
	}, nil
}

// SetDefaultPaymentMethod sets a payment method as default
func (s *StripeService) SetDefaultPaymentMethod(ctx context.Context, tenantID, paymentMethodID uuid.UUID) error {
	// Get the payment method to ensure it belongs to the tenant
	pm, err := s.paymentRepo.GetSavedPaymentMethodByID(ctx, paymentMethodID)
	if err != nil {
		return ErrPaymentMethodNotFound
	}
	if pm.TenantID != tenantID {
		return ErrPaymentMethodNotFound
	}

	// Update Stripe customer's default payment method
	if s.IsConfigured() {
		tenant, err := s.tenantRepo.GetByID(ctx, tenantID)
		if err == nil && tenant.StripeCustomerID != nil {
			params := &stripe.CustomerParams{
				InvoiceSettings: &stripe.CustomerInvoiceSettingsParams{
					DefaultPaymentMethod: stripe.String(pm.StripePaymentMethodID),
				},
			}
			_, err = customer.Update(*tenant.StripeCustomerID, params)
			if err != nil {
				log.Printf("[STRIPE] Warning: failed to update Stripe default payment method: %v", err)
			}
		}
	}

	// Update in database
	return s.paymentRepo.SetDefaultSavedPaymentMethod(ctx, tenantID, paymentMethodID)
}

// DeletePaymentMethod removes a payment method
func (s *StripeService) DeletePaymentMethod(ctx context.Context, tenantID, paymentMethodID uuid.UUID) error {
	// Get the payment method
	pm, err := s.paymentRepo.GetSavedPaymentMethodByID(ctx, paymentMethodID)
	if err != nil {
		return ErrPaymentMethodNotFound
	}
	if pm.TenantID != tenantID {
		return ErrPaymentMethodNotFound
	}

	// Detach from Stripe
	if s.IsConfigured() {
		_, err = paymentmethod.Detach(pm.StripePaymentMethodID, nil)
		if err != nil {
			log.Printf("[STRIPE] Warning: failed to detach payment method from Stripe: %v", err)
		}
	}

	// Delete from database
	return s.paymentRepo.DeleteSavedPaymentMethod(ctx, paymentMethodID, tenantID)
}

// ============================================================================
// Subscription Management
// ============================================================================

// GetPriceIDForTier gets the Stripe price ID for a tier and billing cycle
func (s *StripeService) GetPriceIDForTier(tierName string, billingCycle models.BillingCycle) (string, error) {
	switch tierName {
	case "free":
		return s.config.Prices.FreeMonthly, nil
	case "basic":
		if billingCycle == models.BillingCycleAnnual {
			return s.config.Prices.BasicYearly, nil
		}
		return s.config.Prices.BasicMonthly, nil
	case "pro":
		if billingCycle == models.BillingCycleAnnual {
			return s.config.Prices.ProYearly, nil
		}
		return s.config.Prices.ProMonthly, nil
	default:
		return "", fmt.Errorf("unknown tier: %s", tierName)
	}
}

// CreateSubscription creates a Stripe subscription for a tenant
func (s *StripeService) CreateSubscription(ctx context.Context, tenantID uuid.UUID, tierName string, billingCycle models.BillingCycle) (*stripe.Subscription, error) {
	if !s.IsConfigured() {
		return nil, ErrStripeNotConfigured
	}

	// Get Stripe customer
	customerID, err := s.GetOrCreateStripeCustomer(ctx, tenantID)
	if err != nil {
		return nil, err
	}

	// Get price ID
	priceID, err := s.GetPriceIDForTier(tierName, billingCycle)
	if err != nil {
		return nil, err
	}
	if priceID == "" {
		return nil, fmt.Errorf("price ID not configured for tier %s", tierName)
	}

	// Get default payment method
	defaultPM, err := s.paymentRepo.GetDefaultSavedPaymentMethod(ctx, tenantID)
	if err != nil {
		return nil, ErrNoDefaultPaymentMethod
	}

	// Create subscription
	params := &stripe.SubscriptionParams{
		Customer: stripe.String(customerID),
		Items: []*stripe.SubscriptionItemsParams{
			{
				Price: stripe.String(priceID),
			},
		},
		DefaultPaymentMethod: stripe.String(defaultPM.StripePaymentMethodID),
		Metadata: map[string]string{
			"tenant_id": tenantID.String(),
		},
	}

	sub, err := subscription.New(params)
	if err != nil {
		return nil, fmt.Errorf("failed to create subscription: %w", err)
	}

	log.Printf("[STRIPE] Created subscription %s for tenant %s", sub.ID, tenantID)
	return sub, nil
}

// CancelSubscription cancels a Stripe subscription (at period end)
func (s *StripeService) CancelSubscription(ctx context.Context, stripeSubscriptionID string, immediate bool) error {
	if !s.IsConfigured() {
		return ErrStripeNotConfigured
	}

	if immediate {
		// Cancel immediately
		_, err := subscription.Cancel(stripeSubscriptionID, nil)
		if err != nil {
			return fmt.Errorf("failed to cancel subscription: %w", err)
		}
	} else {
		// Cancel at period end
		params := &stripe.SubscriptionParams{
			CancelAtPeriodEnd: stripe.Bool(true),
		}
		_, err := subscription.Update(stripeSubscriptionID, params)
		if err != nil {
			return fmt.Errorf("failed to cancel subscription at period end: %w", err)
		}
	}

	log.Printf("[STRIPE] Cancelled subscription %s (immediate: %v)", stripeSubscriptionID, immediate)
	return nil
}

// ReactivateSubscription reactivates a subscription that was set to cancel at period end
func (s *StripeService) ReactivateSubscription(ctx context.Context, stripeSubscriptionID string) error {
	if !s.IsConfigured() {
		return ErrStripeNotConfigured
	}

	params := &stripe.SubscriptionParams{
		CancelAtPeriodEnd: stripe.Bool(false),
	}
	_, err := subscription.Update(stripeSubscriptionID, params)
	if err != nil {
		return fmt.Errorf("failed to reactivate subscription: %w", err)
	}

	log.Printf("[STRIPE] Reactivated subscription %s", stripeSubscriptionID)
	return nil
}

// ChangeSubscription changes a subscription to a different plan
func (s *StripeService) ChangeSubscription(ctx context.Context, stripeSubscriptionID string, newTierName string, billingCycle models.BillingCycle) (*stripe.Subscription, error) {
	if !s.IsConfigured() {
		return nil, ErrStripeNotConfigured
	}

	// Get the subscription
	sub, err := subscription.Get(stripeSubscriptionID, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get subscription: %w", err)
	}

	// Get new price ID
	newPriceID, err := s.GetPriceIDForTier(newTierName, billingCycle)
	if err != nil {
		return nil, err
	}
	if newPriceID == "" {
		return nil, fmt.Errorf("price ID not configured for tier %s", newTierName)
	}

	// Update subscription with new price
	params := &stripe.SubscriptionParams{
		Items: []*stripe.SubscriptionItemsParams{
			{
				ID:    stripe.String(sub.Items.Data[0].ID),
				Price: stripe.String(newPriceID),
			},
		},
		ProrationBehavior: stripe.String("create_prorations"),
	}

	updatedSub, err := subscription.Update(stripeSubscriptionID, params)
	if err != nil {
		return nil, fmt.Errorf("failed to update subscription: %w", err)
	}

	log.Printf("[STRIPE] Changed subscription %s to %s (%s)", stripeSubscriptionID, newTierName, billingCycle)
	return updatedSub, nil
}

// ============================================================================
// Subscription Preview
// ============================================================================

// PreviewSubscriptionChange previews what will happen when changing subscriptions
func (s *StripeService) PreviewSubscriptionChange(ctx context.Context, tenantID uuid.UUID, newTierName string, billingCycle models.BillingCycle) (*models.PreviewSubscriptionChangeResponse, error) {
	response := &models.PreviewSubscriptionChangeResponse{
		NewTierName:     newTierName,
		NewBillingCycle: billingCycle,
		Warnings:        []string{},
	}

	// Get the new tier to get pricing
	newTier, err := s.membershipRepo.GetTierByName(ctx, newTierName)
	if err != nil {
		return nil, fmt.Errorf("failed to get new tier: %w", err)
	}

	// Calculate new price
	if billingCycle == models.BillingCycleAnnual {
		response.NewPriceCents = newTier.AnnualPriceCents
	} else {
		response.NewPriceCents = newTier.MonthlyPriceCents
	}

	// Get current subscription
	currentSub, err := s.membershipRepo.GetTenantSubscription(ctx, tenantID)
	if err != nil {
		// No current subscription - this is a new subscription
		response.AmountDueTodayCents = response.NewPriceCents
		now := time.Now()
		if billingCycle == models.BillingCycleAnnual {
			nextBilling := now.AddDate(1, 0, 0)
			response.NextBillingDate = &nextBilling
		} else {
			nextBilling := now.AddDate(0, 1, 0)
			response.NextBillingDate = &nextBilling
		}
		response.NextBillingAmount = response.NewPriceCents
		return response, nil
	}

	// Fill in current plan info
	if currentSub.Tier != nil {
		response.CurrentTierName = currentSub.Tier.DisplayName
	}
	response.CurrentPriceCents = currentSub.PriceCents
	response.CurrentBillingCycle = currentSub.BillingCycle
	response.CurrentPeriodEnd = currentSub.CurrentPeriodEnd

	// Calculate days remaining
	if currentSub.CurrentPeriodEnd != nil {
		daysRemaining := int(time.Until(*currentSub.CurrentPeriodEnd).Hours() / 24)
		if daysRemaining < 0 {
			daysRemaining = 0
		}
		response.DaysRemaining = daysRemaining
	}

	// Determine change type
	response.IsUpgrade = response.NewPriceCents > response.CurrentPriceCents
	response.IsDowngrade = response.NewPriceCents < response.CurrentPriceCents
	response.IsCancellation = response.NewPriceCents == 0 && response.CurrentPriceCents > 0

	// Handle cancellation (downgrade to free)
	if response.IsCancellation {
		response.AmountDueTodayCents = 0
		response.ProrationCreditCents = 0 // No refund for immediate cancellation
		response.Warnings = append(response.Warnings,
			fmt.Sprintf("Your current subscription will be cancelled immediately. You will lose %d days of remaining access.", response.DaysRemaining))
		return response, nil
	}

	// For paid-to-paid changes, use Stripe to preview the invoice
	if s.IsConfigured() && currentSub.StripeSubscriptionID != nil && *currentSub.StripeSubscriptionID != "" {
		preview, err := s.previewStripeInvoice(ctx, tenantID, *currentSub.StripeSubscriptionID, newTierName, billingCycle)
		if err != nil {
			log.Printf("[STRIPE] Warning: failed to preview invoice: %v", err)
			// Fall back to manual calculation
			response.AmountDueTodayCents = response.NewPriceCents
		} else {
			response.ProrationCreditCents = preview.prorationCredit
			response.AmountDueTodayCents = preview.amountDue
			response.NextBillingDate = preview.nextBillingDate
			response.NextBillingAmount = preview.nextBillingAmount
		}
	} else {
		// No existing Stripe subscription - charge full amount
		response.AmountDueTodayCents = response.NewPriceCents
		now := time.Now()
		if billingCycle == models.BillingCycleAnnual {
			nextBilling := now.AddDate(1, 0, 0)
			response.NextBillingDate = &nextBilling
		} else {
			nextBilling := now.AddDate(0, 1, 0)
			response.NextBillingDate = &nextBilling
		}
		response.NextBillingAmount = response.NewPriceCents
	}

	// Add contextual warnings
	if response.IsDowngrade && response.DaysRemaining > 0 {
		response.Warnings = append(response.Warnings,
			"Your plan will be changed immediately. Any unused credit will be applied to your next invoice.")
	}

	return response, nil
}

type invoicePreview struct {
	prorationCredit   int
	amountDue         int
	nextBillingDate   *time.Time
	nextBillingAmount int
}

// previewStripeInvoice uses Stripe's upcoming invoice API to preview changes
func (s *StripeService) previewStripeInvoice(ctx context.Context, tenantID uuid.UUID, stripeSubID string, newTierName string, billingCycle models.BillingCycle) (*invoicePreview, error) {
	// Get the current subscription
	sub, err := subscription.Get(stripeSubID, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get subscription: %w", err)
	}

	// Get the new price ID
	newPriceID, err := s.GetPriceIDForTier(newTierName, billingCycle)
	if err != nil {
		return nil, err
	}

	// Get the customer ID
	customerID, err := s.GetOrCreateStripeCustomer(ctx, tenantID)
	if err != nil {
		return nil, err
	}

	// Preview the upcoming invoice with the subscription change
	params := &stripe.InvoiceUpcomingParams{
		Customer:     stripe.String(customerID),
		Subscription: stripe.String(stripeSubID),
		SubscriptionItems: []*stripe.SubscriptionItemsParams{
			{
				ID:    stripe.String(sub.Items.Data[0].ID),
				Price: stripe.String(newPriceID),
			},
		},
		SubscriptionProrationBehavior: stripe.String("create_prorations"),
	}

	upcomingInvoice, err := invoice.Upcoming(params)
	if err != nil {
		return nil, fmt.Errorf("failed to preview invoice: %w", err)
	}

	preview := &invoicePreview{
		amountDue:         int(upcomingInvoice.AmountDue),
		nextBillingAmount: int(upcomingInvoice.Total),
	}

	// Calculate proration credit from line items
	for _, line := range upcomingInvoice.Lines.Data {
		if line.Proration {
			// Negative amounts are credits
			preview.prorationCredit += int(-line.Amount)
		}
	}

	// Next billing date
	if upcomingInvoice.NextPaymentAttempt > 0 {
		nextBilling := time.Unix(upcomingInvoice.NextPaymentAttempt, 0)
		preview.nextBillingDate = &nextBilling
	} else if sub.CurrentPeriodEnd > 0 {
		nextBilling := time.Unix(sub.CurrentPeriodEnd, 0)
		preview.nextBillingDate = &nextBilling
	}

	return preview, nil
}

// ============================================================================
// Billing History
// ============================================================================

// GetBillingHistory retrieves billing events for a tenant
func (s *StripeService) GetBillingHistory(ctx context.Context, tenantID uuid.UUID, page, pageSize int) (*models.ListBillingEventsResponse, error) {
	events, total, err := s.paymentRepo.GetBillingEventsByTenantID(ctx, tenantID, page, pageSize)
	if err != nil {
		return nil, fmt.Errorf("failed to get billing history: %w", err)
	}

	responses := make([]models.BillingEventResponse, len(events))
	for i, e := range events {
		responses[i] = e.ToBillingEventResponse()
	}

	totalPages := int(total) / pageSize
	if int(total)%pageSize > 0 {
		totalPages++
	}

	return &models.ListBillingEventsResponse{
		Events:     responses,
		Total:      total,
		Page:       page,
		PageSize:   pageSize,
		TotalPages: totalPages,
	}, nil
}

// RecordBillingEvent records a billing event (called from webhook handler)
func (s *StripeService) RecordBillingEvent(ctx context.Context, tenantID uuid.UUID, event *models.BillingEvent) error {
	// Check for duplicate event
	exists, err := s.paymentRepo.BillingEventExists(ctx, event.StripeEventID)
	if err != nil {
		return fmt.Errorf("failed to check billing event existence: %w", err)
	}
	if exists {
		log.Printf("[STRIPE] Skipping duplicate billing event: %s", event.StripeEventID)
		return nil
	}

	event.TenantID = tenantID
	if err := s.paymentRepo.CreateBillingEvent(ctx, event); err != nil {
		return fmt.Errorf("failed to create billing event: %w", err)
	}

	log.Printf("[STRIPE] Recorded billing event %s for tenant %s", event.StripeEventID, tenantID)
	return nil
}

// ============================================================================
// Webhook Processing
// ============================================================================

// HandleSubscriptionUpdated handles subscription.updated webhook
func (s *StripeService) HandleSubscriptionUpdated(ctx context.Context, sub *stripe.Subscription) error {
	stripeSubID := sub.ID

	// Map Stripe status to our status
	var status models.MembershipStatus
	switch sub.Status {
	case stripe.SubscriptionStatusActive, stripe.SubscriptionStatusTrialing:
		status = models.MembershipStatusActive
	case stripe.SubscriptionStatusCanceled:
		status = models.MembershipStatusCancelled
	case stripe.SubscriptionStatusPastDue, stripe.SubscriptionStatusUnpaid:
		status = models.MembershipStatusPending
	default:
		status = models.MembershipStatusActive
	}

	// Convert period end
	periodEnd := time.Unix(sub.CurrentPeriodEnd, 0)

	// Update subscription in database
	if err := s.membershipRepo.UpdateSubscriptionFromStripe(ctx, stripeSubID, status, &periodEnd, sub.CancelAtPeriodEnd); err != nil {
		return fmt.Errorf("failed to update subscription: %w", err)
	}

	log.Printf("[STRIPE] Updated subscription %s: status=%s, cancel_at_period_end=%v", stripeSubID, status, sub.CancelAtPeriodEnd)
	return nil
}

// HandleSubscriptionDeleted handles subscription.deleted webhook
func (s *StripeService) HandleSubscriptionDeleted(ctx context.Context, sub *stripe.Subscription) error {
	stripeSubID := sub.ID

	// Update subscription to cancelled
	if err := s.membershipRepo.UpdateSubscriptionFromStripe(ctx, stripeSubID, models.MembershipStatusCancelled, nil, false); err != nil {
		return fmt.Errorf("failed to update subscription: %w", err)
	}

	log.Printf("[STRIPE] Subscription %s deleted", stripeSubID)
	return nil
}
