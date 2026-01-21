package services

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"

	"github.com/javaknight1/servicepro/backend/internal/models"
)

// PaymentTermsService handles payment terms business logic
type PaymentTermsService struct {
	db *gorm.DB
}

// NewPaymentTermsService creates a new payment terms service
func NewPaymentTermsService(db *gorm.DB) *PaymentTermsService {
	return &PaymentTermsService{db: db}
}

// PaymentCalculationResult represents the result of payment calculations
type PaymentCalculationResult struct {
	DueDate                time.Time       `json:"due_date"`
	DiscountAvailableUntil *time.Time      `json:"discount_available_until,omitempty"`
	DiscountAmount         decimal.Decimal `json:"discount_amount"`
	LateFeeAmount          decimal.Decimal `json:"late_fee_amount"`
	TotalDue               decimal.Decimal `json:"total_due"`
	IsEarlyPaymentEligible bool            `json:"is_early_payment_eligible"`
	IsOverdue              bool            `json:"is_overdue"`
	DaysOverdue            int             `json:"days_overdue"`
	GracePeriodEnds        *time.Time      `json:"grace_period_ends,omitempty"`
}

// PaymentStatus represents the status of a payment relative to terms
type PaymentStatus struct {
	Status               string          `json:"status"` // "early", "on_time", "grace_period", "overdue"
	DueDate              time.Time       `json:"due_date"`
	DaysUntilDue         int             `json:"days_until_due,omitempty"`
	DaysOverdue          int             `json:"days_overdue,omitempty"`
	DiscountEligible     bool            `json:"discount_eligible"`
	DiscountAmount       decimal.Decimal `json:"discount_amount,omitempty"`
	LateFeeApplied       bool            `json:"late_fee_applied"`
	LateFeeAmount        decimal.Decimal `json:"late_fee_amount,omitempty"`
	InGracePeriod        bool            `json:"in_grace_period"`
	GracePeriodEnds      *time.Time      `json:"grace_period_ends,omitempty"`
	RecommendedAction    string          `json:"recommended_action"`
	NotificationRequired bool            `json:"notification_required"`
	NotificationType     string          `json:"notification_type,omitempty"`
}

// CreatePaymentTerm creates a new payment term
func (s *PaymentTermsService) CreatePaymentTerm(ctx context.Context, term *models.PaymentTerm) error {
	if err := s.validatePaymentTerm(term); err != nil {
		return err
	}

	// If this is set as default, unset other defaults
	if term.IsDefault {
		if err := s.unsetOtherDefaults(ctx, uuid.Nil); err != nil {
			return fmt.Errorf("failed to unset other defaults: %w", err)
		}
	}

	if err := s.db.WithContext(ctx).Create(term).Error; err != nil {
		return fmt.Errorf("failed to create payment term: %w", err)
	}

	return nil
}

// GetPaymentTerm retrieves a payment term by ID
func (s *PaymentTermsService) GetPaymentTerm(ctx context.Context, id uuid.UUID) (*models.PaymentTerm, error) {
	var term models.PaymentTerm
	if err := s.db.WithContext(ctx).Where("id = ?", id).First(&term).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("payment term not found")
		}
		return nil, fmt.Errorf("failed to get payment term: %w", err)
	}
	return &term, nil
}

// GetDefaultPaymentTerm retrieves the default payment term
func (s *PaymentTermsService) GetDefaultPaymentTerm(ctx context.Context) (*models.PaymentTerm, error) {
	var term models.PaymentTerm
	if err := s.db.WithContext(ctx).Where("is_default = ? AND is_active = ?", true, true).First(&term).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("no default payment term found")
		}
		return nil, fmt.Errorf("failed to get default payment term: %w", err)
	}
	return &term, nil
}

// ListPaymentTerms retrieves all active payment terms
func (s *PaymentTermsService) ListPaymentTerms(ctx context.Context, activeOnly bool) ([]models.PaymentTerm, error) {
	var terms []models.PaymentTerm
	query := s.db.WithContext(ctx)

	if activeOnly {
		query = query.Where("is_active = ?", true)
	}

	if err := query.Order("name ASC").Find(&terms).Error; err != nil {
		return nil, fmt.Errorf("failed to list payment terms: %w", err)
	}
	return terms, nil
}

// UpdatePaymentTerm updates an existing payment term
func (s *PaymentTermsService) UpdatePaymentTerm(ctx context.Context, id uuid.UUID, updates *models.PaymentTerm) error {
	if err := s.validatePaymentTerm(updates); err != nil {
		return err
	}

	// If setting as default, unset other defaults
	if updates.IsDefault {
		if err := s.unsetOtherDefaults(ctx, id); err != nil {
			return fmt.Errorf("failed to unset other defaults: %w", err)
		}
	}

	if err := s.db.WithContext(ctx).Model(&models.PaymentTerm{}).Where("id = ?", id).Updates(updates).Error; err != nil {
		return fmt.Errorf("failed to update payment term: %w", err)
	}

	return nil
}

// DeletePaymentTerm deletes a payment term
func (s *PaymentTermsService) DeletePaymentTerm(ctx context.Context, id uuid.UUID) error {
	// Check if this payment term is used by any invoices
	var count int64
	if err := s.db.WithContext(ctx).Model(&models.Invoice{}).Where("payment_term_id = ?", id).Count(&count).Error; err != nil {
		return fmt.Errorf("failed to check payment term usage: %w", err)
	}

	if count > 0 {
		return fmt.Errorf("cannot delete payment term: it is used by %d invoice(s)", count)
	}

	if err := s.db.WithContext(ctx).Delete(&models.PaymentTerm{}, id).Error; err != nil {
		return fmt.Errorf("failed to delete payment term: %w", err)
	}

	return nil
}

// CalculateDueDate calculates the due date based on payment terms
func (s *PaymentTermsService) CalculateDueDate(ctx context.Context, issueDate time.Time, paymentTermID uuid.UUID, timezone string) (time.Time, error) {
	term, err := s.GetPaymentTerm(ctx, paymentTermID)
	if err != nil {
		return time.Time{}, err
	}

	return s.CalculateDueDateFromTerm(issueDate, term, timezone), nil
}

// CalculateDueDateFromTerm calculates due date from a payment term object
func (s *PaymentTermsService) CalculateDueDateFromTerm(issueDate time.Time, term *models.PaymentTerm, timezone string) time.Time {
	// Load timezone
	loc, err := time.LoadLocation(timezone)
	if err != nil {
		loc = time.UTC
	}

	// Convert issue date to specified timezone
	issueDateInTz := issueDate.In(loc)

	// Calculate due date by adding the number of days
	dueDate := issueDateInTz.AddDate(0, 0, term.DaysUntilDue)

	return dueDate
}

// CalculateEarlyPaymentDiscount calculates the discount for early payment
func (s *PaymentTermsService) CalculateEarlyPaymentDiscount(
	ctx context.Context,
	invoiceAmount decimal.Decimal,
	paymentDate time.Time,
	issueDate time.Time,
	paymentTermID uuid.UUID,
) (discount decimal.Decimal, eligible bool, err error) {
	term, err := s.GetPaymentTerm(ctx, paymentTermID)
	if err != nil {
		return decimal.Zero, false, err
	}

	return s.CalculateEarlyPaymentDiscountFromTerm(invoiceAmount, paymentDate, issueDate, term)
}

// CalculateEarlyPaymentDiscountFromTerm calculates discount from a payment term object
func (s *PaymentTermsService) CalculateEarlyPaymentDiscountFromTerm(
	invoiceAmount decimal.Decimal,
	paymentDate time.Time,
	issueDate time.Time,
	term *models.PaymentTerm,
) (discount decimal.Decimal, eligible bool, err error) {
	// Check if early payment discount is configured
	if term.DiscountDays == 0 || term.DiscountPercentage.IsZero() {
		return decimal.Zero, false, nil
	}

	// Calculate the discount deadline
	discountDeadline := issueDate.AddDate(0, 0, term.DiscountDays)

	// Check if payment is within the discount period
	if paymentDate.After(discountDeadline) {
		return decimal.Zero, false, nil
	}

	// Calculate discount amount
	discountAmount := invoiceAmount.Mul(term.DiscountPercentage).Div(decimal.NewFromInt(100))
	discountAmount = discountAmount.Round(2)

	return discountAmount, true, nil
}

// CalculateLateFee calculates the late fee for overdue payment
func (s *PaymentTermsService) CalculateLateFee(
	ctx context.Context,
	invoiceAmount decimal.Decimal,
	currentDate time.Time,
	dueDate time.Time,
	paymentTermID uuid.UUID,
	timezone string,
) (lateFee decimal.Decimal, applicable bool, err error) {
	term, err := s.GetPaymentTerm(ctx, paymentTermID)
	if err != nil {
		return decimal.Zero, false, err
	}

	return s.CalculateLateFeeFromTerm(invoiceAmount, currentDate, dueDate, term, timezone)
}

// CalculateLateFeeFromTerm calculates late fee from a payment term object
func (s *PaymentTermsService) CalculateLateFeeFromTerm(
	invoiceAmount decimal.Decimal,
	currentDate time.Time,
	dueDate time.Time,
	term *models.PaymentTerm,
	timezone string,
) (lateFee decimal.Decimal, applicable bool, err error) {
	// Load timezone
	loc, err := time.LoadLocation(timezone)
	if err != nil {
		loc = time.UTC
	}

	currentDateInTz := currentDate.In(loc)
	dueDateInTz := dueDate.In(loc)

	// Calculate grace period end date
	gracePeriodEnd := dueDateInTz.AddDate(0, 0, term.GracePeriodDays)

	// Check if we're past the grace period
	if currentDateInTz.Before(gracePeriodEnd) || currentDateInTz.Equal(gracePeriodEnd) {
		return decimal.Zero, false, nil
	}

	// Calculate late fee based on percentage
	if !term.LateFeePercentage.IsZero() {
		lateFee = invoiceAmount.Mul(term.LateFeePercentage).Div(decimal.NewFromInt(100))
		lateFee = lateFee.Round(2)
	}

	// Add fixed late fee amount if configured
	if !term.LateFeeAmount.IsZero() {
		lateFee = lateFee.Add(term.LateFeeAmount)
	}

	return lateFee, !lateFee.IsZero(), nil
}

// GetPaymentStatus determines the current payment status
func (s *PaymentTermsService) GetPaymentStatus(
	ctx context.Context,
	invoiceAmount decimal.Decimal,
	issueDate time.Time,
	dueDate time.Time,
	paymentTermID uuid.UUID,
	currentDate time.Time,
	timezone string,
) (*PaymentStatus, error) {
	term, err := s.GetPaymentTerm(ctx, paymentTermID)
	if err != nil {
		return nil, err
	}

	return s.GetPaymentStatusFromTerm(invoiceAmount, issueDate, dueDate, term, currentDate, timezone)
}

// GetPaymentStatusFromTerm determines payment status from a payment term object
func (s *PaymentTermsService) GetPaymentStatusFromTerm(
	invoiceAmount decimal.Decimal,
	issueDate time.Time,
	dueDate time.Time,
	term *models.PaymentTerm,
	currentDate time.Time,
	timezone string,
) (*PaymentStatus, error) {
	// Load timezone
	loc, err := time.LoadLocation(timezone)
	if err != nil {
		loc = time.UTC
	}

	currentDateInTz := currentDate.In(loc)
	dueDateInTz := dueDate.In(loc)
	issueDateInTz := issueDate.In(loc)

	status := &PaymentStatus{
		DueDate: dueDateInTz,
	}

	// Calculate days until/past due
	daysDiff := int(dueDateInTz.Sub(currentDateInTz).Hours() / 24)

	if daysDiff > 0 {
		status.DaysUntilDue = daysDiff
	} else if daysDiff < 0 {
		status.DaysOverdue = -daysDiff
	}

	// Check early payment discount eligibility
	if term.DiscountDays > 0 && !term.DiscountPercentage.IsZero() {
		discountDeadline := issueDateInTz.AddDate(0, 0, term.DiscountDays)
		if currentDateInTz.Before(discountDeadline) || currentDateInTz.Equal(discountDeadline) {
			status.DiscountEligible = true
			discountAmount := invoiceAmount.Mul(term.DiscountPercentage).Div(decimal.NewFromInt(100))
			status.DiscountAmount = discountAmount.Round(2)
			status.Status = "early"
			status.RecommendedAction = fmt.Sprintf("Pay before %s to receive %.2f%% discount",
				discountDeadline.Format("2006-01-02"), term.DiscountPercentage)
			status.NotificationRequired = true
			status.NotificationType = "early_payment_discount_available"
		}
	}

	// Check if payment is on time
	if currentDateInTz.Before(dueDateInTz) {
		if status.Status == "" {
			status.Status = "on_time"
			if daysDiff <= 7 {
				status.RecommendedAction = fmt.Sprintf("Payment due in %d days", daysDiff)
				status.NotificationRequired = true
				status.NotificationType = "payment_due_soon"
			}
		}
		return status, nil
	}

	// Check grace period
	gracePeriodEnd := dueDateInTz.AddDate(0, 0, term.GracePeriodDays)
	if currentDateInTz.Before(gracePeriodEnd) || currentDateInTz.Equal(gracePeriodEnd) {
		status.Status = "grace_period"
		status.InGracePeriod = true
		status.GracePeriodEnds = &gracePeriodEnd
		status.RecommendedAction = fmt.Sprintf("Grace period ends %s", gracePeriodEnd.Format("2006-01-02"))
		status.NotificationRequired = true
		status.NotificationType = "payment_in_grace_period"
		return status, nil
	}

	// Payment is overdue
	status.Status = "overdue"

	// Calculate late fee
	lateFee, applicable, err := s.CalculateLateFeeFromTerm(invoiceAmount, currentDateInTz, dueDateInTz, term, timezone)
	if err != nil {
		return nil, err
	}

	if applicable {
		status.LateFeeApplied = true
		status.LateFeeAmount = lateFee
	}

	status.RecommendedAction = fmt.Sprintf("Payment overdue by %d days. Late fee: %s",
		status.DaysOverdue, lateFee.String())
	status.NotificationRequired = true
	status.NotificationType = "payment_overdue"

	return status, nil
}

// CalculatePaymentDetails calculates comprehensive payment details
func (s *PaymentTermsService) CalculatePaymentDetails(
	ctx context.Context,
	invoiceAmount decimal.Decimal,
	issueDate time.Time,
	paymentTermID uuid.UUID,
	paymentDate *time.Time,
	timezone string,
) (*PaymentCalculationResult, error) {
	term, err := s.GetPaymentTerm(ctx, paymentTermID)
	if err != nil {
		return nil, err
	}

	// Load timezone
	loc, err := time.LoadLocation(timezone)
	if err != nil {
		loc = time.UTC
	}

	issueDateInTz := issueDate.In(loc)

	result := &PaymentCalculationResult{
		DueDate:        s.CalculateDueDateFromTerm(issueDateInTz, term, timezone),
		DiscountAmount: decimal.Zero,
		LateFeeAmount:  decimal.Zero,
		TotalDue:       invoiceAmount,
	}

	// Calculate discount availability
	if term.DiscountDays > 0 && !term.DiscountPercentage.IsZero() {
		discountDeadline := issueDateInTz.AddDate(0, 0, term.DiscountDays)
		result.DiscountAvailableUntil = &discountDeadline

		// If payment date is provided, check if discount applies
		if paymentDate != nil {
			discount, eligible, _ := s.CalculateEarlyPaymentDiscountFromTerm(
				invoiceAmount, *paymentDate, issueDateInTz, term,
			)
			result.DiscountAmount = discount
			result.IsEarlyPaymentEligible = eligible
			if eligible {
				result.TotalDue = result.TotalDue.Sub(discount)
			}
		} else {
			// Show potential discount
			result.DiscountAmount = invoiceAmount.Mul(term.DiscountPercentage).Div(decimal.NewFromInt(100)).Round(2)
			currentTime := time.Now().In(loc)
			result.IsEarlyPaymentEligible = currentTime.Before(discountDeadline) || currentTime.Equal(discountDeadline)
		}
	}

	// Calculate late fee if payment date is provided
	if paymentDate != nil {
		lateFee, applicable, _ := s.CalculateLateFeeFromTerm(
			invoiceAmount, *paymentDate, result.DueDate, term, timezone,
		)
		if applicable {
			result.LateFeeAmount = lateFee
			result.TotalDue = result.TotalDue.Add(lateFee)
		}

		// Check if overdue
		paymentDateInTz := paymentDate.In(loc)
		gracePeriodEnd := result.DueDate.AddDate(0, 0, term.GracePeriodDays)
		if paymentDateInTz.After(gracePeriodEnd) {
			result.IsOverdue = true
			result.DaysOverdue = int(paymentDateInTz.Sub(result.DueDate).Hours() / 24)
			result.GracePeriodEnds = &gracePeriodEnd
		}
	}

	return result, nil
}

// validatePaymentTerm validates payment term data
func (s *PaymentTermsService) validatePaymentTerm(term *models.PaymentTerm) error {
	if term.Name == "" {
		return errors.New("payment term name is required")
	}

	if term.DaysUntilDue < 0 {
		return errors.New("days until due must be non-negative")
	}

	if !term.DiscountPercentage.IsZero() {
		if term.DiscountPercentage.LessThan(decimal.Zero) || term.DiscountPercentage.GreaterThan(decimal.NewFromInt(100)) {
			return errors.New("discount percentage must be between 0 and 100")
		}
	}

	if term.DiscountDays < 0 {
		return errors.New("discount days must be non-negative")
	}

	if term.DiscountDays > term.DaysUntilDue {
		return errors.New("discount days cannot exceed days until due")
	}

	if !term.LateFeePercentage.IsZero() {
		if term.LateFeePercentage.LessThan(decimal.Zero) || term.LateFeePercentage.GreaterThan(decimal.NewFromInt(100)) {
			return errors.New("late fee percentage must be between 0 and 100")
		}
	}

	if !term.LateFeeAmount.IsZero() && term.LateFeeAmount.LessThan(decimal.Zero) {
		return errors.New("late fee amount must be non-negative")
	}

	if term.GracePeriodDays < 0 {
		return errors.New("grace period days must be non-negative")
	}

	return nil
}

// unsetOtherDefaults unsets the is_default flag on all other payment terms
func (s *PaymentTermsService) unsetOtherDefaults(ctx context.Context, excludeID uuid.UUID) error {
	query := s.db.WithContext(ctx).Model(&models.PaymentTerm{}).
		Where("is_default = ?", true)

	if excludeID != uuid.Nil {
		query = query.Where("id != ?", excludeID)
	}

	return query.Update("is_default", false).Error
}
