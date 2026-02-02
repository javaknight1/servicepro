package services

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/javaknight1/servicepro/backend/internal/models"
	"github.com/javaknight1/servicepro/backend/pkg/clients/logging"
)

var (
	// ErrInvalidTransition is returned when a status transition is not allowed
	ErrInvalidTransition = errors.New("invalid status transition")
	// ErrQuoteExpired is returned when trying to transition an expired quote
	ErrQuoteExpired = errors.New("quote has expired")
)

// QuoteStatusMachine manages quote status transitions
type QuoteStatusMachine struct {
	db                 *gorm.DB
	notificationSvc    NotificationService
	allowedTransitions map[models.QuoteStatus][]models.QuoteStatus
}

// NotificationService interface for sending notifications
type NotificationService interface {
	SendQuoteStatusNotification(ctx context.Context, notification *models.QuoteStatusNotification) error
}

// NewQuoteStatusMachine creates a new quote status machine
func NewQuoteStatusMachine(db *gorm.DB, notificationSvc NotificationService) *QuoteStatusMachine {
	return &QuoteStatusMachine{
		db:              db,
		notificationSvc: notificationSvc,
		allowedTransitions: map[models.QuoteStatus][]models.QuoteStatus{
			// Draft can transition to sent or be deleted
			models.QuoteStatusDraft: {
				models.QuoteStatusSent,
			},
			// Sent can transition to viewed, accepted, declined, or back to draft
			models.QuoteStatusSent: {
				models.QuoteStatusViewed,
				models.QuoteStatusAccepted,
				models.QuoteStatusDeclined,
				models.QuoteStatusDraft, // Allow editing
			},
			// Viewed can transition to accepted or declined
			models.QuoteStatusViewed: {
				models.QuoteStatusAccepted,
				models.QuoteStatusDeclined,
			},
			// Accepted is a terminal state (can only expire)
			models.QuoteStatusAccepted: {
				models.QuoteStatusExpired,
			},
			// Declined is a terminal state
			models.QuoteStatusDeclined: {},
			// Expired is a terminal state
			models.QuoteStatusExpired: {},
		},
	}
}

// IsTransitionAllowed checks if a status transition is allowed
func (sm *QuoteStatusMachine) IsTransitionAllowed(from, to models.QuoteStatus) bool {
	allowedStates, exists := sm.allowedTransitions[from]
	if !exists {
		return false
	}

	for _, allowed := range allowedStates {
		if allowed == to {
			return true
		}
	}

	return false
}

// GetAllowedTransitions returns all allowed transitions from a given status
func (sm *QuoteStatusMachine) GetAllowedTransitions(from models.QuoteStatus) []models.QuoteStatus {
	return sm.allowedTransitions[from]
}

// ValidateTransition validates a status transition
func (sm *QuoteStatusMachine) ValidateTransition(ctx context.Context, transition *models.QuoteStatusTransition) error {
	// Check if transition is allowed
	if !sm.IsTransitionAllowed(transition.FromStatus, transition.ToStatus) {
		return fmt.Errorf("%w: cannot transition from %s to %s",
			ErrInvalidTransition, transition.FromStatus, transition.ToStatus)
	}

	// Fetch the quote to check expiration
	var quote models.Quote
	if err := sm.db.WithContext(ctx).First(&quote, "id = ?", transition.QuoteID).Error; err != nil {
		return fmt.Errorf("failed to fetch quote: %w", err)
	}

	// Check if quote has expired
	if quote.ValidUntil.Before(time.Now()) && transition.ToStatus != models.QuoteStatusExpired {
		return ErrQuoteExpired
	}

	// Additional validation based on target status
	switch transition.ToStatus {
	case models.QuoteStatusSent:
		// Must have at least one line item
		if len(quote.Items) == 0 {
			return models.QuoteValidationError{
				Field:   "items",
				Message: "cannot send quote without line items",
			}
		}
		// Customer email validation would require loading the customer relationship
		// TODO: Add customer email validation when Customer is preloaded

	case models.QuoteStatusAccepted:
		// Can only accept if sent or viewed
		if transition.FromStatus != models.QuoteStatusSent && transition.FromStatus != models.QuoteStatusViewed {
			return fmt.Errorf("%w: can only accept sent or viewed quotes", ErrInvalidTransition)
		}

	case models.QuoteStatusDeclined:
		// Can only decline if sent or viewed
		if transition.FromStatus != models.QuoteStatusSent && transition.FromStatus != models.QuoteStatusViewed {
			return fmt.Errorf("%w: can only decline sent or viewed quotes", ErrInvalidTransition)
		}
	}

	return nil
}

// Transition performs a status transition with validation and history tracking
func (sm *QuoteStatusMachine) Transition(ctx context.Context, transition *models.QuoteStatusTransition) error {
	// Validate the transition
	if err := sm.ValidateTransition(ctx, transition); err != nil {
		return err
	}

	// Start a transaction
	return sm.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Update the quote status
		if err := tx.Model(&models.Quote{}).
			Where("id = ? AND status = ?", transition.QuoteID, transition.FromStatus).
			Update("status", transition.ToStatus).Error; err != nil {
			return fmt.Errorf("failed to update quote status: %w", err)
		}

		// Create history record
		history := &models.QuoteStatusHistory{
			QuoteID:       transition.QuoteID,
			FromStatus:    transition.FromStatus,
			ToStatus:      transition.ToStatus,
			Event:         transition.Event,
			ChangedByID:   transition.ChangedByID,
			ChangedByType: transition.ChangedByType,
			Reason:        transition.Reason,
			Metadata:      transition.Metadata,
			CreatedAt:     time.Now(),
		}

		if err := tx.Create(history).Error; err != nil {
			return fmt.Errorf("failed to create history record: %w", err)
		}

		// Send notification
		if sm.notificationSvc != nil {
			notification := &models.QuoteStatusNotification{
				Type:      "status_change",
				QuoteID:   transition.QuoteID,
				Event:     transition.Event,
				Status:    transition.ToStatus,
				Timestamp: time.Now(),
				Data: map[string]interface{}{
					"from_status": transition.FromStatus,
					"reason":      transition.Reason,
				},
			}

			// Send notification asynchronously (don't block transaction)
			go func() {
				bgCtx := context.Background()
				if err := sm.notificationSvc.SendQuoteStatusNotification(bgCtx, notification); err != nil {
					// Log error but don't fail the transaction
					logging.Error(bgCtx, "[QUOTE-STATUS] Failed to send notification", map[string]any{"error": err})
				}
			}()
		}

		return nil
	})
}

// SendQuote transitions a quote from draft to sent
func (sm *QuoteStatusMachine) SendQuote(ctx context.Context, quoteID uuid.UUID, changedBy uuid.UUID) error {
	var quote models.Quote
	if err := sm.db.WithContext(ctx).First(&quote, "id = ?", quoteID).Error; err != nil {
		return fmt.Errorf("failed to fetch quote: %w", err)
	}

	transition := &models.QuoteStatusTransition{
		QuoteID:       quoteID,
		FromStatus:    quote.Status,
		ToStatus:      models.QuoteStatusSent,
		Event:         models.EventQuoteSent,
		ChangedByID:   changedBy,
		ChangedByType: "user",
		Metadata: map[string]interface{}{
			"sent_at": time.Now(),
		},
	}

	return sm.Transition(ctx, transition)
}

// MarkViewed transitions a quote from sent to viewed
func (sm *QuoteStatusMachine) MarkViewed(ctx context.Context, quoteID uuid.UUID) error {
	var quote models.Quote
	if err := sm.db.WithContext(ctx).First(&quote, "id = ?", quoteID).Error; err != nil {
		return fmt.Errorf("failed to fetch quote: %w", err)
	}

	// Only mark as viewed if currently sent
	if quote.Status != models.QuoteStatusSent {
		return nil // Already viewed or in different state
	}

	transition := &models.QuoteStatusTransition{
		QuoteID:       quoteID,
		FromStatus:    quote.Status,
		ToStatus:      models.QuoteStatusViewed,
		Event:         models.EventQuoteViewed,
		ChangedByType: "customer",
		Metadata: map[string]interface{}{
			"viewed_at": time.Now(),
		},
	}

	return sm.Transition(ctx, transition)
}

// AcceptQuote transitions a quote to accepted
func (sm *QuoteStatusMachine) AcceptQuote(ctx context.Context, quoteID uuid.UUID, customerID uuid.UUID, reason string) error {
	var quote models.Quote
	if err := sm.db.WithContext(ctx).First(&quote, "id = ?", quoteID).Error; err != nil {
		return fmt.Errorf("failed to fetch quote: %w", err)
	}

	transition := &models.QuoteStatusTransition{
		QuoteID:       quoteID,
		FromStatus:    quote.Status,
		ToStatus:      models.QuoteStatusAccepted,
		Event:         models.EventQuoteAccepted,
		ChangedByID:   customerID,
		ChangedByType: "customer",
		Reason:        reason,
		Metadata: map[string]interface{}{
			"accepted_at": time.Now(),
		},
	}

	return sm.Transition(ctx, transition)
}

// DeclineQuote transitions a quote to declined
func (sm *QuoteStatusMachine) DeclineQuote(ctx context.Context, quoteID uuid.UUID, customerID uuid.UUID, reason string) error {
	var quote models.Quote
	if err := sm.db.WithContext(ctx).First(&quote, "id = ?", quoteID).Error; err != nil {
		return fmt.Errorf("failed to fetch quote: %w", err)
	}

	transition := &models.QuoteStatusTransition{
		QuoteID:       quoteID,
		FromStatus:    quote.Status,
		ToStatus:      models.QuoteStatusDeclined,
		Event:         models.EventQuoteDeclined,
		ChangedByID:   customerID,
		ChangedByType: "customer",
		Reason:        reason,
		Metadata: map[string]interface{}{
			"declined_at": time.Now(),
		},
	}

	return sm.Transition(ctx, transition)
}

// ExpireQuote transitions a quote to expired
func (sm *QuoteStatusMachine) ExpireQuote(ctx context.Context, quoteID uuid.UUID) error {
	var quote models.Quote
	if err := sm.db.WithContext(ctx).First(&quote, "id = ?", quoteID).Error; err != nil {
		return fmt.Errorf("failed to fetch quote: %w", err)
	}

	transition := &models.QuoteStatusTransition{
		QuoteID:       quoteID,
		FromStatus:    quote.Status,
		ToStatus:      models.QuoteStatusExpired,
		Event:         models.EventQuoteExpired,
		ChangedByType: "system",
		Metadata: map[string]interface{}{
			"expired_at":  time.Now(),
			"valid_until": quote.ValidUntil,
		},
	}

	return sm.Transition(ctx, transition)
}

// GetStatusHistory retrieves the status history for a quote
func (sm *QuoteStatusMachine) GetStatusHistory(ctx context.Context, quoteID uuid.UUID) ([]models.QuoteStatusHistory, error) {
	var history []models.QuoteStatusHistory
	if err := sm.db.WithContext(ctx).
		Where("quote_id = ?", quoteID).
		Order("created_at DESC").
		Find(&history).Error; err != nil {
		return nil, fmt.Errorf("failed to fetch status history: %w", err)
	}

	return history, nil
}

// GetLatestStatusChange retrieves the most recent status change for a quote
func (sm *QuoteStatusMachine) GetLatestStatusChange(ctx context.Context, quoteID uuid.UUID) (*models.QuoteStatusHistory, error) {
	var history models.QuoteStatusHistory
	if err := sm.db.WithContext(ctx).
		Where("quote_id = ?", quoteID).
		Order("created_at DESC").
		First(&history).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to fetch latest status change: %w", err)
	}

	return &history, nil
}

// CheckAndExpireQuotes checks for expired quotes and transitions them
func (sm *QuoteStatusMachine) CheckAndExpireQuotes(ctx context.Context) error {
	var quotes []models.Quote

	// Find quotes that are past their valid_until date but not yet expired
	if err := sm.db.WithContext(ctx).
		Where("valid_until < ? AND status NOT IN ?", time.Now(),
			[]models.QuoteStatus{models.QuoteStatusExpired, models.QuoteStatusAccepted, models.QuoteStatusDeclined}).
		Find(&quotes).Error; err != nil {
		return fmt.Errorf("failed to fetch expired quotes: %w", err)
	}

	// Expire each quote
	for _, quote := range quotes {
		if err := sm.ExpireQuote(ctx, quote.ID); err != nil {
			// Log error but continue with other quotes
			logging.Error(ctx, "[QUOTE-STATUS] Failed to expire quote", map[string]any{"quote_id": quote.ID, "error": err})
		}
	}

	return nil
}
