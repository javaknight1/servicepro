package repository

import (
	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/javaknight1/servicepro/backend/internal/models"
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
