package services

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"

	"github.com/javaknight1/servicepro/backend/internal/models"
	"github.com/javaknight1/servicepro/backend/internal/repository"
)

var (
	ErrMembershipTierNotFound = errors.New("membership tier not found")
	ErrNoActiveSubscription   = errors.New("no active subscription found")
	ErrSameTier               = errors.New("tenant already on this tier")
)

const (
	// DefaultTierName is the tier assigned to new tenants
	DefaultTierName = "free"
)

// MembershipService handles membership business logic
type MembershipService struct {
	membershipRepo *repository.MembershipRepository
}

// NewMembershipService creates a new membership service
func NewMembershipService(membershipRepo *repository.MembershipRepository) *MembershipService {
	return &MembershipService{
		membershipRepo: membershipRepo,
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

	// Update the subscription
	changedByPtr := &changedBy
	if err := s.membershipRepo.UpdateSubscription(ctx, tenantID, req.TierID, req.BillingCycle, priceCents, changedByPtr, req.ChangeReason); err != nil {
		return nil, fmt.Errorf("failed to update subscription: %w", err)
	}

	log.Printf("[MEMBERSHIP] Updated tenant %s to tier %s (%s) with %s billing at %d cents",
		tenantID, newTier.Name, newTier.DisplayName, req.BillingCycle, priceCents)

	// Return the updated membership
	return s.GetTenantMembership(ctx, tenantID)
}

// AssignDefaultTier assigns the default (free) tier to a new tenant
func (s *MembershipService) AssignDefaultTier(ctx context.Context, tenantID uuid.UUID) error {
	// Get the free tier
	freeTier, err := s.membershipRepo.GetTierByName(ctx, DefaultTierName)
	if err != nil {
		if errors.Is(err, repository.ErrTierNotFound) {
			log.Printf("[MEMBERSHIP] WARNING: Default tier '%s' not found, skipping assignment", DefaultTierName)
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
			log.Printf("[MEMBERSHIP] Tenant %s already has an active subscription", tenantID)
			return nil
		}
		return fmt.Errorf("failed to create subscription: %w", err)
	}

	log.Printf("[MEMBERSHIP] Assigned default tier '%s' to tenant %s", DefaultTierName, tenantID)
	return nil
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
