package repository

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/javaknight1/servicepro/backend/internal/models"
)

// ConnectedAccountRepositoryInterface stores each tenant's Stripe Connect
// account (T169). Every method takes the tenant explicitly, except the
// webhook lookup by Stripe account ID, which returns the owning tenant.
type ConnectedAccountRepositoryInterface interface {
	// GetByTenant returns the tenant's account, or (nil, nil) when it has none.
	GetByTenant(ctx context.Context, tenantID uuid.UUID) (*models.StripeConnectedAccount, error)
	// CreateIfAbsent stores the tenant's account unless one exists already and
	// returns the stored row (the first one wins on a race).
	CreateIfAbsent(ctx context.Context, tenantID uuid.UUID, stripeAccountID string, status models.StripeAccountStatus) (*models.StripeConnectedAccount, error)
	// UpdateStatus caches the account's statuses for the tenant.
	UpdateStatus(ctx context.Context, tenantID uuid.UUID, status models.StripeAccountStatus) error
	// GetByStripeAccountID resolves a webhook's account to its row (and tenant),
	// or (nil, nil) when the account is not ours.
	GetByStripeAccountID(ctx context.Context, stripeAccountID string) (*models.StripeConnectedAccount, error)
}

// ConnectedAccountRepository is the GORM implementation.
type ConnectedAccountRepository struct {
	db *gorm.DB
}

// NewConnectedAccountRepository creates the repository.
func NewConnectedAccountRepository(db *gorm.DB) *ConnectedAccountRepository {
	return &ConnectedAccountRepository{db: db}
}

var _ ConnectedAccountRepositoryInterface = (*ConnectedAccountRepository)(nil)

// GetByTenant returns the tenant's account, or (nil, nil).
func (r *ConnectedAccountRepository) GetByTenant(ctx context.Context, tenantID uuid.UUID) (*models.StripeConnectedAccount, error) {
	var acct models.StripeConnectedAccount
	err := r.db.WithContext(ctx).Where("tenant_id = ?", tenantID).First(&acct).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &acct, nil
}

// CreateIfAbsent inserts the account (ON CONFLICT (tenant_id) DO NOTHING) and
// returns whichever row is stored for the tenant.
func (r *ConnectedAccountRepository) CreateIfAbsent(ctx context.Context, tenantID uuid.UUID, stripeAccountID string, status models.StripeAccountStatus) (*models.StripeConnectedAccount, error) {
	now := time.Now()
	row := &models.StripeConnectedAccount{
		TenantID:           tenantID,
		StripeAccountID:    stripeAccountID,
		Livemode:           status.Livemode,
		CardPaymentsStatus: status.CardPaymentsStatus,
		ACHPaymentsStatus:  status.ACHPaymentsStatus,
		PayoutsStatus:      status.PayoutsStatus,
		RequirementsStatus: status.RequirementsStatus,
		SyncedAt:           &now,
	}
	if err := r.db.WithContext(ctx).
		Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "tenant_id"}}, DoNothing: true}).
		Create(row).Error; err != nil {
		return nil, err
	}
	return r.GetByTenant(ctx, tenantID)
}

// UpdateStatus caches the account's statuses for the tenant.
func (r *ConnectedAccountRepository) UpdateStatus(ctx context.Context, tenantID uuid.UUID, status models.StripeAccountStatus) error {
	return r.db.WithContext(ctx).Model(&models.StripeConnectedAccount{}).
		Where("tenant_id = ?", tenantID).
		Updates(map[string]any{
			"livemode":             status.Livemode,
			"card_payments_status": status.CardPaymentsStatus,
			"ach_payments_status":  status.ACHPaymentsStatus,
			"payouts_status":       status.PayoutsStatus,
			"requirements_status":  status.RequirementsStatus,
			"synced_at":            time.Now(),
			"updated_at":           time.Now(),
		}).Error
}

// GetByStripeAccountID resolves a webhook's account, or (nil, nil).
func (r *ConnectedAccountRepository) GetByStripeAccountID(ctx context.Context, stripeAccountID string) (*models.StripeConnectedAccount, error) {
	if stripeAccountID == "" {
		return nil, nil
	}
	var acct models.StripeConnectedAccount
	err := r.db.WithContext(ctx).Where("stripe_account_id = ?", stripeAccountID).First(&acct).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &acct, nil
}
