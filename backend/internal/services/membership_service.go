package services

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/javaknight1/servicepro/backend/internal/models"
	"github.com/javaknight1/servicepro/backend/internal/repository"
	"github.com/javaknight1/servicepro/backend/pkg/clients/logging"
)

var (
	ErrMembershipTierNotFound   = errors.New("membership tier not found")
	ErrNoActiveSubscription     = errors.New("no active subscription found")
	ErrSameTier                 = errors.New("tenant already on this tier")
	ErrPaymentMethodRequired    = errors.New("payment method required for paid tier")
	ErrStripeSubscriptionFailed = errors.New("failed to create subscription")
)

const (
	// DefaultTierName is the tier assigned to new tenants
	DefaultTierName = "free"
)

// MembershipService handles membership business logic
type MembershipService struct {
	membershipRepo *repository.MembershipRepository
	stripeService  *StripeService // Optional, nil if Stripe not configured
}

// NewMembershipService creates a new membership service
func NewMembershipService(membershipRepo *repository.MembershipRepository, stripeService *StripeService) *MembershipService {
	return &MembershipService{
		membershipRepo: membershipRepo,
		stripeService:  stripeService,
	}
}

// GetAllTiers retrieves all active membership tiers
func (s *MembershipService) GetAllTiers(ctx context.Context) (*models.ListTiersResponse, error) {
	tiers, err := s.membershipRepo.GetAllTiers(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get tiers: %w", err)
	}

	responses := make([]models.MembershipTierResponse, len(tiers))
	for i, tier := range tiers {
		responses[i] = tier.ToResponse()
	}

	return &models.ListTiersResponse{
		Tiers: responses,
	}, nil
}

// GetTierByID retrieves a membership tier by ID
func (s *MembershipService) GetTierByID(ctx context.Context, tierID uuid.UUID) (*models.MembershipTier, error) {
	tier, err := s.membershipRepo.GetTierByID(ctx, tierID)
	if err != nil {
		if errors.Is(err, repository.ErrTierNotFound) {
			return nil, ErrMembershipTierNotFound
		}
		return nil, fmt.Errorf("failed to get tier: %w", err)
	}
	return tier, nil
}

// GetTenantMembership retrieves the current membership for a tenant
func (s *MembershipService) GetTenantMembership(ctx context.Context, tenantID uuid.UUID) (*models.GetMembershipResponse, error) {
	subscription, err := s.membershipRepo.GetTenantSubscription(ctx, tenantID)
	if err != nil {
		if errors.Is(err, repository.ErrSubscriptionNotFound) {
			return nil, ErrNoActiveSubscription
		}
		return nil, fmt.Errorf("failed to get subscription: %w", err)
	}

	// Use ToResponse which handles all fields including PriceCents and IsLegacy
	subResponse := subscription.ToResponse()

	response := &models.GetMembershipResponse{
		Subscription: &subResponse,
	}

	if subscription.Tier != nil {
		tierResponse := subscription.Tier.ToResponse()
		response.Tier = &tierResponse
	}

	return response, nil
}

// UpdateTenantMembership changes a tenant's membership tier
func (s *MembershipService) UpdateTenantMembership(ctx context.Context, tenantID uuid.UUID, req *models.UpdateMembershipRequest, changedBy uuid.UUID) (*models.GetMembershipResponse, error) {
	// Verify the new tier exists and is enabled
	newTier, err := s.membershipRepo.GetTierByID(ctx, req.TierID)
	if err != nil {
		if errors.Is(err, repository.ErrTierNotFound) {
			return nil, ErrMembershipTierNotFound
		}
		return nil, fmt.Errorf("failed to get tier: %w", err)
	}

	// Get current subscription
	currentSub, err := s.membershipRepo.GetTenantSubscription(ctx, tenantID)
	if err != nil && !errors.Is(err, repository.ErrSubscriptionNotFound) {
		return nil, fmt.Errorf("failed to get current subscription: %w", err)
	}

	// Check if already on the same tier with same billing cycle
	if currentSub != nil && currentSub.TierID == req.TierID && currentSub.BillingCycle == req.BillingCycle {
		return nil, ErrSameTier
	}

	// Calculate the price to lock in based on billing cycle
	var priceCents int
	if req.BillingCycle == models.BillingCycleAnnual {
		priceCents = newTier.AnnualPriceCents
	} else {
		priceCents = newTier.MonthlyPriceCents
	}

	// For paid tiers, handle Stripe subscription
	var stripeSubscriptionID string
	if priceCents > 0 && s.stripeService != nil && s.stripeService.IsConfigured() {
		// Check if there's an existing Stripe subscription
		if currentSub != nil && currentSub.StripeSubscriptionID != nil && *currentSub.StripeSubscriptionID != "" {
			// Update existing subscription
			stripeSub, err := s.stripeService.ChangeSubscription(ctx, *currentSub.StripeSubscriptionID, newTier.Name, req.BillingCycle)
			if err != nil {
				if errors.Is(err, ErrNoDefaultPaymentMethod) {
					return nil, ErrPaymentMethodRequired
				}
				return nil, fmt.Errorf("failed to update Stripe subscription: %w", err)
			}
			stripeSubscriptionID = stripeSub.ID
			logging.Info(ctx, "[MEMBERSHIP] Changed Stripe subscription", map[string]any{"subscription_id": stripeSubscriptionID, "tier_name": newTier.Name})
		} else {
			// Create new subscription
			stripeSub, err := s.stripeService.CreateSubscription(ctx, tenantID, newTier.Name, req.BillingCycle)
			if err != nil {
				if errors.Is(err, ErrNoDefaultPaymentMethod) {
					return nil, ErrPaymentMethodRequired
				}
				return nil, fmt.Errorf("failed to create Stripe subscription: %w", err)
			}
			stripeSubscriptionID = stripeSub.ID
			logging.Info(ctx, "[MEMBERSHIP] Created Stripe subscription", map[string]any{"subscription_id": stripeSubscriptionID, "tier_name": newTier.Name})
		}
	} else if priceCents == 0 && currentSub != nil && currentSub.StripeSubscriptionID != nil && *currentSub.StripeSubscriptionID != "" {
		// Downgrading to free tier - cancel existing Stripe subscription
		if s.stripeService != nil && s.stripeService.IsConfigured() {
			if err := s.stripeService.CancelSubscription(ctx, *currentSub.StripeSubscriptionID, true); err != nil {
				logging.Warn(ctx, "[MEMBERSHIP] Failed to cancel Stripe subscription", map[string]any{"error": err})
				// Continue anyway - the subscription will be cancelled eventually
			}
		}
	}

	// Update the subscription in our database
	changedByPtr := &changedBy
	if err := s.membershipRepo.UpdateSubscription(ctx, tenantID, req.TierID, req.BillingCycle, priceCents, changedByPtr, req.ChangeReason); err != nil {
		return nil, fmt.Errorf("failed to update subscription: %w", err)
	}

	// Update Stripe subscription ID if we have one
	if stripeSubscriptionID != "" {
		// Get the updated subscription to update its Stripe ID
		updatedSub, err := s.membershipRepo.GetTenantSubscription(ctx, tenantID)
		if err == nil && updatedSub != nil {
			priceID, _ := s.stripeService.GetPriceIDForTier(newTier.Name, req.BillingCycle)
			if err := s.membershipRepo.UpdateSubscriptionStripeID(ctx, updatedSub.ID, stripeSubscriptionID, priceID); err != nil {
				logging.Warn(ctx, "[MEMBERSHIP] Failed to save Stripe subscription ID", map[string]any{"error": err})
			}
		}
	}

	logging.Info(ctx, "[MEMBERSHIP] Updated tenant membership", map[string]any{"tenant_id": tenantID, "tier_name": newTier.Name, "tier_display_name": newTier.DisplayName, "billing_cycle": req.BillingCycle, "price_cents": priceCents})

	// Return the updated membership
	return s.GetTenantMembership(ctx, tenantID)
}

// AssignDefaultTier assigns the default (free) tier to a new tenant
func (s *MembershipService) AssignDefaultTier(ctx context.Context, tenantID uuid.UUID) error {
	// Get the free tier
	freeTier, err := s.membershipRepo.GetTierByName(ctx, DefaultTierName)
	if err != nil {
		if errors.Is(err, repository.ErrTierNotFound) {
			logging.Warn(ctx, "[MEMBERSHIP] Default tier not found, skipping assignment", map[string]any{"tier_name": DefaultTierName})
			return nil
		}
		return fmt.Errorf("failed to get default tier: %w", err)
	}

	// Create subscription with free tier (price = 0)
	now := time.Now()
	periodEnd := now.AddDate(0, 1, 0) // 1 month period even for free tier
	subscription := &models.TenantSubscription{
		ID:                 uuid.New(),
		TenantID:           tenantID,
		TierID:             freeTier.ID,
		Status:             models.MembershipStatusActive,
		BillingCycle:       models.BillingCycleMonthly,
		PriceCents:         freeTier.MonthlyPriceCents, // 0 for free tier
		StartedAt:          now,
		CurrentPeriodStart: now,
		CurrentPeriodEnd:   &periodEnd,
	}

	if err := s.membershipRepo.CreateSubscription(ctx, subscription); err != nil {
		if errors.Is(err, repository.ErrActiveSubscription) {
			logging.Info(ctx, "[MEMBERSHIP] Tenant already has an active subscription", map[string]any{"tenant_id": tenantID})
			return nil
		}
		return fmt.Errorf("failed to create subscription: %w", err)
	}

	logging.Info(ctx, "[MEMBERSHIP] Assigned default tier to tenant", map[string]any{"tier_name": DefaultTierName, "tenant_id": tenantID})
	return nil
}

// PreviewSubscriptionChange previews what will happen when changing subscriptions
func (s *MembershipService) PreviewSubscriptionChange(ctx context.Context, tenantID uuid.UUID, req *models.PreviewSubscriptionChangeRequest) (*models.PreviewSubscriptionChangeResponse, error) {
	// Get the new tier
	newTier, err := s.membershipRepo.GetTierByID(ctx, req.TierID)
	if err != nil {
		if errors.Is(err, repository.ErrTierNotFound) {
			return nil, ErrMembershipTierNotFound
		}
		return nil, fmt.Errorf("failed to get tier: %w", err)
	}

	// If Stripe is configured, use its preview
	if s.stripeService != nil && s.stripeService.IsConfigured() {
		return s.stripeService.PreviewSubscriptionChange(ctx, tenantID, newTier.Name, req.BillingCycle)
	}

	// Otherwise, do a simple manual calculation
	response := &models.PreviewSubscriptionChangeResponse{
		NewTierName:     newTier.DisplayName,
		NewBillingCycle: req.BillingCycle,
		Warnings:        []string{},
	}

	// Calculate new price
	if req.BillingCycle == models.BillingCycleAnnual {
		response.NewPriceCents = newTier.AnnualPriceCents
	} else {
		response.NewPriceCents = newTier.MonthlyPriceCents
	}

	// Get current subscription
	currentSub, err := s.membershipRepo.GetTenantSubscription(ctx, tenantID)
	if err == nil && currentSub != nil {
		if currentSub.Tier != nil {
			response.CurrentTierName = currentSub.Tier.DisplayName
		}
		response.CurrentPriceCents = currentSub.PriceCents
		response.CurrentBillingCycle = currentSub.BillingCycle
		response.CurrentPeriodEnd = currentSub.CurrentPeriodEnd

		if currentSub.CurrentPeriodEnd != nil {
			daysRemaining := int(time.Until(*currentSub.CurrentPeriodEnd).Hours() / 24)
			if daysRemaining < 0 {
				daysRemaining = 0
			}
			response.DaysRemaining = daysRemaining
		}

		response.IsUpgrade = response.NewPriceCents > response.CurrentPriceCents
		response.IsDowngrade = response.NewPriceCents < response.CurrentPriceCents
		response.IsCancellation = response.NewPriceCents == 0 && response.CurrentPriceCents > 0

		if response.IsCancellation {
			response.Warnings = append(response.Warnings,
				fmt.Sprintf("Your current subscription will be cancelled immediately. You will lose %d days of remaining access.", response.DaysRemaining))
		}
	}

	response.AmountDueTodayCents = response.NewPriceCents
	now := time.Now()
	if req.BillingCycle == models.BillingCycleAnnual {
		nextBilling := now.AddDate(1, 0, 0)
		response.NextBillingDate = &nextBilling
	} else {
		nextBilling := now.AddDate(0, 1, 0)
		response.NextBillingDate = &nextBilling
	}
	response.NextBillingAmount = response.NewPriceCents

	return response, nil
}

// GetSubscriptionHistory retrieves the subscription history for a tenant
func (s *MembershipService) GetSubscriptionHistory(ctx context.Context, tenantID uuid.UUID) ([]models.TenantSubscriptionResponse, error) {
	subscriptions, err := s.membershipRepo.GetSubscriptionHistory(ctx, tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to get subscription history: %w", err)
	}

	responses := make([]models.TenantSubscriptionResponse, len(subscriptions))
	for i, sub := range subscriptions {
		responses[i] = sub.ToResponse()
	}

	return responses, nil
}
