package repository

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/javaknight1/servicepro/backend/internal/models"
)

var (
	ErrPaymentMethodNotFound = errors.New("payment method not found")
	ErrBillingEventNotFound  = errors.New("billing event not found")
)

// PaymentRepository implements PaymentRepositoryInterface using GORM
type PaymentRepository struct {
	db *gorm.DB
}

// NewPaymentRepository creates a new payment repository
func NewPaymentRepository(db *gorm.DB) *PaymentRepository {
	return &PaymentRepository{db: db}
}

// CreatePayment creates a new payment record
func (r *PaymentRepository) CreatePayment(payment *models.Payment) error {
	return r.db.Create(payment).Error
}

// GetPaymentByID retrieves a payment by ID
func (r *PaymentRepository) GetPaymentByID(id uuid.UUID) (*models.Payment, error) {
	var payment models.Payment
	err := r.db.Where("id = ?", id).First(&payment).Error
	if err != nil {
		return nil, err
	}
	return &payment, nil
}

// GetPaymentByStripeID retrieves a payment by Stripe payment intent ID
func (r *PaymentRepository) GetPaymentByStripeID(stripePaymentIntentID string) (*models.Payment, error) {
	var payment models.Payment
	err := r.db.Where("stripe_payment_intent_id = ?", stripePaymentIntentID).First(&payment).Error
	if err != nil {
		return nil, err
	}
	return &payment, nil
}

// UpdatePayment updates an existing payment record
func (r *PaymentRepository) UpdatePayment(payment *models.Payment) error {
	return r.db.Save(payment).Error
}

// GetPaymentsByUserID retrieves all payments for a user with optional filters
func (r *PaymentRepository) GetPaymentsByUserID(userID uuid.UUID, filter *models.PaymentListFilter) ([]models.Payment, error) {
	query := r.db.Where("user_id = ?", userID)

	// Apply filters if provided
	if filter != nil {
		if filter.Status != nil {
			query = query.Where("status = ?", *filter.Status)
		}
		if filter.StartDate != nil {
			query = query.Where("created_at >= ?", *filter.StartDate)
		}
		if filter.EndDate != nil {
			query = query.Where("created_at <= ?", *filter.EndDate)
		}
		if filter.MinAmount != nil {
			query = query.Where("amount >= ?", *filter.MinAmount)
		}
		if filter.MaxAmount != nil {
			query = query.Where("amount <= ?", *filter.MaxAmount)
		}

		// Apply pagination
		if filter.Page > 0 && filter.PageSize > 0 {
			offset := (filter.Page - 1) * filter.PageSize
			query = query.Offset(offset).Limit(filter.PageSize)
		}
	}

	var payments []models.Payment
	err := query.Order("created_at DESC").Find(&payments).Error
	if err != nil {
		return nil, err
	}

	return payments, nil
}

// CreateRefund creates a new refund record
func (r *PaymentRepository) CreateRefund(refund *models.PaymentRefund) error {
	return r.db.Create(refund).Error
}

// GetRefundsByPaymentID retrieves all refunds for a payment
func (r *PaymentRepository) GetRefundsByPaymentID(paymentID uuid.UUID) ([]models.PaymentRefund, error) {
	var refunds []models.PaymentRefund
	err := r.db.Where("payment_id = ?", paymentID).Order("created_at DESC").Find(&refunds).Error
	if err != nil {
		return nil, err
	}
	return refunds, nil
}

// ============================================================================
// Saved Payment Methods (for Tenant Subscriptions)
// ============================================================================

// GetSavedPaymentMethodsByTenantID retrieves all saved payment methods for a tenant
func (r *PaymentRepository) GetSavedPaymentMethodsByTenantID(ctx context.Context, tenantID uuid.UUID) ([]models.SavedPaymentMethod, error) {
	var methods []models.SavedPaymentMethod
	err := r.db.WithContext(ctx).
		Where("tenant_id = ?", tenantID).
		Order("is_default DESC, created_at DESC").
		Find(&methods).Error
	if err != nil {
		return nil, err
	}
	return methods, nil
}

// GetSavedPaymentMethodByID retrieves a saved payment method by ID
func (r *PaymentRepository) GetSavedPaymentMethodByID(ctx context.Context, id uuid.UUID) (*models.SavedPaymentMethod, error) {
	var method models.SavedPaymentMethod
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&method).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrPaymentMethodNotFound
		}
		return nil, err
	}
	return &method, nil
}

// GetSavedPaymentMethodByStripeID retrieves a saved payment method by Stripe ID
func (r *PaymentRepository) GetSavedPaymentMethodByStripeID(ctx context.Context, stripePaymentMethodID string) (*models.SavedPaymentMethod, error) {
	var method models.SavedPaymentMethod
	err := r.db.WithContext(ctx).Where("stripe_payment_method_id = ?", stripePaymentMethodID).First(&method).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrPaymentMethodNotFound
		}
		return nil, err
	}
	return &method, nil
}

// GetDefaultSavedPaymentMethod retrieves the default payment method for a tenant
func (r *PaymentRepository) GetDefaultSavedPaymentMethod(ctx context.Context, tenantID uuid.UUID) (*models.SavedPaymentMethod, error) {
	var method models.SavedPaymentMethod
	err := r.db.WithContext(ctx).
		Where("tenant_id = ? AND is_default = ?", tenantID, true).
		First(&method).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrPaymentMethodNotFound
		}
		return nil, err
	}
	return &method, nil
}

// CreateSavedPaymentMethod creates a new saved payment method
func (r *PaymentRepository) CreateSavedPaymentMethod(ctx context.Context, method *models.SavedPaymentMethod) error {
	return r.db.WithContext(ctx).Create(method).Error
}

// SetDefaultSavedPaymentMethod sets a payment method as default for a tenant
func (r *PaymentRepository) SetDefaultSavedPaymentMethod(ctx context.Context, tenantID, paymentMethodID uuid.UUID) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Remove default from all other payment methods
		if err := tx.Model(&models.SavedPaymentMethod{}).
			Where("tenant_id = ? AND is_default = ?", tenantID, true).
			Update("is_default", false).Error; err != nil {
			return err
		}

		// Set the new default
		result := tx.Model(&models.SavedPaymentMethod{}).
			Where("id = ? AND tenant_id = ?", paymentMethodID, tenantID).
			Update("is_default", true)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return ErrPaymentMethodNotFound
		}

		return nil
	})
}

// DeleteSavedPaymentMethod deletes a saved payment method
func (r *PaymentRepository) DeleteSavedPaymentMethod(ctx context.Context, id, tenantID uuid.UUID) error {
	result := r.db.WithContext(ctx).
		Where("id = ? AND tenant_id = ?", id, tenantID).
		Delete(&models.SavedPaymentMethod{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrPaymentMethodNotFound
	}
	return nil
}

// ============================================================================
// Billing Events (for Subscription History)
// ============================================================================

// GetBillingEventsByTenantID retrieves billing events for a tenant with pagination
func (r *PaymentRepository) GetBillingEventsByTenantID(ctx context.Context, tenantID uuid.UUID, page, pageSize int) ([]models.BillingEvent, int64, error) {
	var events []models.BillingEvent
	var total int64

	// Get total count
	if err := r.db.WithContext(ctx).Model(&models.BillingEvent{}).
		Where("tenant_id = ?", tenantID).
		Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// Get paginated results
	offset := (page - 1) * pageSize
	err := r.db.WithContext(ctx).
		Where("tenant_id = ?", tenantID).
		Order("created_at DESC").
		Offset(offset).
		Limit(pageSize).
		Find(&events).Error
	if err != nil {
		return nil, 0, err
	}

	return events, total, nil
}

// GetBillingEventByStripeEventID retrieves a billing event by Stripe event ID
func (r *PaymentRepository) GetBillingEventByStripeEventID(ctx context.Context, stripeEventID string) (*models.BillingEvent, error) {
	var event models.BillingEvent
	err := r.db.WithContext(ctx).Where("stripe_event_id = ?", stripeEventID).First(&event).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrBillingEventNotFound
		}
		return nil, err
	}
	return &event, nil
}

// CreateBillingEvent creates a new billing event
func (r *PaymentRepository) CreateBillingEvent(ctx context.Context, event *models.BillingEvent) error {
	return r.db.WithContext(ctx).Create(event).Error
}

// BillingEventExists checks if a billing event already exists (for idempotency)
func (r *PaymentRepository) BillingEventExists(ctx context.Context, stripeEventID string) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&models.BillingEvent{}).
		Where("stripe_event_id = ?", stripeEventID).
		Count(&count).Error
	if err != nil {
		return false, err
	}
	return count > 0, nil
}
