package repository

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/javaknight1/servicepro/backend/internal/models"
)

var (
	ErrTierNotFound         = errors.New("membership tier not found")
	ErrSubscriptionNotFound = errors.New("subscription not found")
	ErrActiveSubscription   = errors.New("tenant already has an active subscription")
)

// MembershipRepository handles membership data operations
type MembershipRepository struct {
	db *gorm.DB
}

// NewMembershipRepository creates a new membership repository
func NewMembershipRepository(db *gorm.DB) *MembershipRepository {
	return &MembershipRepository{db: db}
}

// GetAllTiers retrieves all enabled membership tiers (for selection/pricing pages)
func (r *MembershipRepository) GetAllTiers(ctx context.Context) ([]models.MembershipTier, error) {
	var tiers []models.MembershipTier
	err := r.db.WithContext(ctx).
		Where("enabled = ?", true).
		Order("sort_order ASC").
		Find(&tiers).Error
	if err != nil {
		return nil, err
	}
	return tiers, nil
}

// GetTierByID retrieves an enabled membership tier by ID (for selection validation)
func (r *MembershipRepository) GetTierByID(ctx context.Context, id uuid.UUID) (*models.MembershipTier, error) {
	var tier models.MembershipTier
	err := r.db.WithContext(ctx).
		Where("id = ? AND enabled = ?", id, true).
		First(&tier).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrTierNotFound
		}
		return nil, err
	}
	return &tier, nil
}

// GetTierByIDIncludingDisabled retrieves any tier by ID (including legacy/disabled tiers)
func (r *MembershipRepository) GetTierByIDIncludingDisabled(ctx context.Context, id uuid.UUID) (*models.MembershipTier, error) {
	var tier models.MembershipTier
	err := r.db.WithContext(ctx).
		Where("id = ?", id).
		First(&tier).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrTierNotFound
		}
		return nil, err
	}
	return &tier, nil
}

// GetTierByName retrieves an enabled membership tier by name
func (r *MembershipRepository) GetTierByName(ctx context.Context, name string) (*models.MembershipTier, error) {
	var tier models.MembershipTier
	err := r.db.WithContext(ctx).
		Where("name = ? AND enabled = ?", name, true).
		First(&tier).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrTierNotFound
		}
		return nil, err
	}
	return &tier, nil
}

// GetTenantSubscription retrieves the current active subscription for a tenant
func (r *MembershipRepository) GetTenantSubscription(ctx context.Context, tenantID uuid.UUID) (*models.TenantSubscription, error) {
	var subscription models.TenantSubscription
	err := r.db.WithContext(ctx).
		Preload("Tier").
		Where("tenant_id = ? AND status = ?", tenantID, models.MembershipStatusActive).
		First(&subscription).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrSubscriptionNotFound
		}
		return nil, err
	}
	return &subscription, nil
}

// CreateSubscription creates a new subscription for a tenant
func (r *MembershipRepository) CreateSubscription(ctx context.Context, subscription *models.TenantSubscription) error {
	// Check if tenant already has an active subscription
	var count int64
	if err := r.db.WithContext(ctx).Model(&models.TenantSubscription{}).
		Where("tenant_id = ? AND status = ?", subscription.TenantID, models.MembershipStatusActive).
		Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return ErrActiveSubscription
	}

	return r.db.WithContext(ctx).Create(subscription).Error
}

// UpdateSubscription updates a tenant's subscription to a new tier
func (r *MembershipRepository) UpdateSubscription(ctx context.Context, tenantID, newTierID uuid.UUID, billingCycle models.BillingCycle, priceCents int, changedBy *uuid.UUID, reason string) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// End the current active subscription
		now := time.Now()
		result := tx.Model(&models.TenantSubscription{}).
			Where("tenant_id = ? AND status = ?", tenantID, models.MembershipStatusActive).
			Updates(map[string]any{
				"status":     models.MembershipStatusCancelled,
				"ended_at":   now,
				"changed_by": changedBy,
			})
		if result.Error != nil {
			return result.Error
		}

		// Calculate next billing period end
		var periodEnd time.Time
		if billingCycle == models.BillingCycleAnnual {
			periodEnd = now.AddDate(1, 0, 0) // 1 year from now
		} else {
			periodEnd = now.AddDate(0, 1, 0) // 1 month from now
		}

		// Create new subscription with the new tier
		newSubscription := &models.TenantSubscription{
			ID:                 uuid.New(),
			TenantID:           tenantID,
			TierID:             newTierID,
			Status:             models.MembershipStatusActive,
			BillingCycle:       billingCycle,
			PriceCents:         priceCents,
			StartedAt:          now,
			CurrentPeriodStart: now,
			CurrentPeriodEnd:   &periodEnd,
			ChangedBy:          changedBy,
			ChangeReason:       reason,
		}

		return tx.Create(newSubscription).Error
	})
}

// GetTierLimitations retrieves the limitations for a specific tier
func (r *MembershipRepository) GetTierLimitations(ctx context.Context, tierID uuid.UUID) ([]models.MembershipLimitation, error) {
	var limitations []models.MembershipLimitation
	err := r.db.WithContext(ctx).
		Where("tier_id = ?", tierID).
		Find(&limitations).Error
	if err != nil {
		return nil, err
	}
	return limitations, nil
}

// GetSubscriptionHistory retrieves all subscriptions for a tenant (for history)
func (r *MembershipRepository) GetSubscriptionHistory(ctx context.Context, tenantID uuid.UUID) ([]models.TenantSubscription, error) {
	var subscriptions []models.TenantSubscription
	err := r.db.WithContext(ctx).
		Preload("Tier").
		Preload("User").
		Where("tenant_id = ?", tenantID).
		Order("created_at DESC").
		Find(&subscriptions).Error
	if err != nil {
		return nil, err
	}
	return subscriptions, nil
}

// ============================================================================
// Stripe Integration Methods
// ============================================================================

// GetSubscriptionByStripeID retrieves a subscription by Stripe subscription ID
func (r *MembershipRepository) GetSubscriptionByStripeID(ctx context.Context, stripeSubscriptionID string) (*models.TenantSubscription, error) {
	var subscription models.TenantSubscription
	err := r.db.WithContext(ctx).
		Preload("Tier").
		Where("stripe_subscription_id = ?", stripeSubscriptionID).
		First(&subscription).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrSubscriptionNotFound
		}
		return nil, err
	}
	return &subscription, nil
}

// UpdateSubscriptionStripeID updates the Stripe subscription ID for a subscription
func (r *MembershipRepository) UpdateSubscriptionStripeID(ctx context.Context, subscriptionID uuid.UUID, stripeSubscriptionID, stripePriceID string) error {
	return r.db.WithContext(ctx).Model(&models.TenantSubscription{}).
		Where("id = ?", subscriptionID).
		Updates(map[string]any{
			"stripe_subscription_id": stripeSubscriptionID,
			"stripe_price_id":        stripePriceID,
		}).Error
}

// UpdateSubscriptionFromStripe updates subscription fields from Stripe webhook data
func (r *MembershipRepository) UpdateSubscriptionFromStripe(ctx context.Context, stripeSubscriptionID string, status models.MembershipStatus, currentPeriodEnd *time.Time, cancelAtPeriodEnd bool) error {
	return r.db.WithContext(ctx).Model(&models.TenantSubscription{}).
		Where("stripe_subscription_id = ?", stripeSubscriptionID).
		Updates(map[string]any{
			"status":               status,
			"current_period_end":   currentPeriodEnd,
			"cancel_at_period_end": cancelAtPeriodEnd,
		}).Error
}

// CancelSubscriptionAtPeriodEnd marks a subscription to cancel at period end
func (r *MembershipRepository) CancelSubscriptionAtPeriodEnd(ctx context.Context, subscriptionID uuid.UUID, cancel bool) error {
	return r.db.WithContext(ctx).Model(&models.TenantSubscription{}).
		Where("id = ?", subscriptionID).
		Update("cancel_at_period_end", cancel).Error
}
