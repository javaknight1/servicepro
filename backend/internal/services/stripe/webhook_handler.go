package stripe

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/stripe/stripe-go/v76"
	"github.com/stripe/stripe-go/v76/webhook"
	"gorm.io/gorm"

	"github.com/javaknight1/servicepro/backend/internal/models"
	"github.com/javaknight1/servicepro/backend/pkg/clients/logging"
)

// =============================================================================
// Errors
// =============================================================================

var (
	ErrInvalidSignature      = errors.New("invalid webhook signature")
	ErrEventExpired          = errors.New("webhook event has expired")
	ErrEventAlreadyProcessed = errors.New("event already processed")
	ErrMaxRetriesExceeded    = errors.New("max retries exceeded")
	ErrProcessingFailed      = errors.New("event processing failed")
	ErrDatabaseError         = errors.New("database operation failed")
)

// =============================================================================
// Notification Interface
// =============================================================================

// WebhookNotifier interface for sending notifications on critical events
type WebhookNotifier interface {
	NotifyPaymentSucceeded(ctx context.Context, paymentID string, amount int64, currency string) error
	NotifyPaymentFailed(ctx context.Context, paymentID string, errorMsg string) error
	NotifySubscriptionCanceled(ctx context.Context, subscriptionID string, customerID string) error
	NotifyDisputeCreated(ctx context.Context, disputeID string, amount int64) error
	NotifyRefundCreated(ctx context.Context, refundID string, amount int64) error
	NotifyWebhookError(ctx context.Context, eventID string, eventType string, err error) error
}

// NoOpNotifier is a no-operation notifier for when notifications are disabled
type NoOpNotifier struct{}

// InvoicePaymentHandler interface for handling invoice payment events
type InvoicePaymentHandler interface {
	HandleInvoicePaymentCompleted(ctx context.Context, invoiceID string, amountPaidCents int64, checkoutSessionID, paymentIntentID string) error
	SendInvoiceReceiptEmail(ctx context.Context, invoiceID string) error
}

// NoOpInvoicePaymentHandler is a no-operation handler for when invoice payment handling is disabled
type NoOpInvoicePaymentHandler struct{}

func (n *NoOpInvoicePaymentHandler) HandleInvoicePaymentCompleted(ctx context.Context, invoiceID string, amountPaidCents int64, checkoutSessionID, paymentIntentID string) error {
	return nil
}
func (n *NoOpInvoicePaymentHandler) SendInvoiceReceiptEmail(ctx context.Context, invoiceID string) error {
	return nil
}

func (n *NoOpNotifier) NotifyPaymentSucceeded(ctx context.Context, paymentID string, amount int64, currency string) error {
	return nil
}
func (n *NoOpNotifier) NotifyPaymentFailed(ctx context.Context, paymentID string, errorMsg string) error {
	return nil
}
func (n *NoOpNotifier) NotifySubscriptionCanceled(ctx context.Context, subscriptionID string, customerID string) error {
	return nil
}
func (n *NoOpNotifier) NotifyDisputeCreated(ctx context.Context, disputeID string, amount int64) error {
	return nil
}
func (n *NoOpNotifier) NotifyRefundCreated(ctx context.Context, refundID string, amount int64) error {
	return nil
}
func (n *NoOpNotifier) NotifyWebhookError(ctx context.Context, eventID string, eventType string, err error) error {
	return nil
}

// =============================================================================
// Webhook Handler Service
// =============================================================================

// WebhookHandlerConfig contains configuration for the webhook handler
type WebhookHandlerConfig struct {
	WebhookSecret       string
	WebhookTolerance    time.Duration
	MaxRetries          int
	RetryBaseDelay      time.Duration
	RetryMaxDelay       time.Duration
	EnableDBLogging     bool
	EnableNotifications bool
	ProcessingTimeout   time.Duration
}

// DefaultWebhookHandlerConfig returns default configuration
func DefaultWebhookHandlerConfig() *WebhookHandlerConfig {
	return &WebhookHandlerConfig{
		WebhookTolerance:    5 * time.Minute,
		MaxRetries:          3,
		RetryBaseDelay:      time.Second,
		RetryMaxDelay:       5 * time.Minute,
		EnableDBLogging:     true,
		EnableNotifications: true,
		ProcessingTimeout:   30 * time.Second,
	}
}

// WebhookHandlerService handles Stripe webhooks with database logging and retry support
type WebhookHandlerService struct {
	config                *WebhookHandlerConfig
	db                    *gorm.DB
	notifier              WebhookNotifier
	invoicePaymentHandler InvoicePaymentHandler
	processor             *EventProcessor

	// In-memory idempotency cache
	processedEvents map[string]time.Time
	eventMu         sync.RWMutex

	// Retry queue
	retryQueue chan *models.WebhookEvent
	stopChan   chan struct{}
	wg         sync.WaitGroup
}

// NewWebhookHandlerService creates a new webhook handler service
func NewWebhookHandlerService(
	config *WebhookHandlerConfig,
	db *gorm.DB,
	notifier WebhookNotifier,
) *WebhookHandlerService {
	if config == nil {
		config = DefaultWebhookHandlerConfig()
	}

	if notifier == nil {
		notifier = &NoOpNotifier{}
	}

	processor := NewEventProcessor()

	service := &WebhookHandlerService{
		config:                config,
		db:                    db,
		notifier:              notifier,
		invoicePaymentHandler: &NoOpInvoicePaymentHandler{},
		processor:             processor,
		processedEvents:       make(map[string]time.Time),
		retryQueue:            make(chan *models.WebhookEvent, 1000),
		stopChan:              make(chan struct{}),
	}

	// Register event handlers
	service.registerEventHandlers()

	return service
}

// SetInvoicePaymentHandler sets the invoice payment handler for processing invoice payment webhooks
func (s *WebhookHandlerService) SetInvoicePaymentHandler(handler InvoicePaymentHandler) {
	if handler != nil {
		s.invoicePaymentHandler = handler
	}
}

// Start starts the background retry processor
func (s *WebhookHandlerService) Start() {
	s.wg.Add(2)

	// Start retry processor
	go s.retryProcessor()

	// Start cleanup routine
	go s.cleanupRoutine()

	logging.Info(context.Background(), "[Stripe] Webhook handler service started", nil)
}

// Stop stops the background processors
func (s *WebhookHandlerService) Stop() {
	close(s.stopChan)
	s.wg.Wait()

	logging.Info(context.Background(), "[Stripe] Webhook handler service stopped", nil)
}

// =============================================================================
// Main Webhook Processing
// =============================================================================

// WebhookInput represents the input for processing a webhook
type WebhookInput struct {
	Payload   []byte
	Signature string
	IPAddress string
	UserAgent string
	Headers   map[string]string
}

// WebhookResult represents the result of processing a webhook
type WebhookResult struct {
	EventID   string `json:"event_id"`
	EventType string `json:"event_type"`
	Processed bool   `json:"processed"`
	Error     string `json:"error,omitempty"`
	WasRetry  bool   `json:"was_retry,omitempty"`
}

// ProcessWebhook processes an incoming Stripe webhook
func (s *WebhookHandlerService) ProcessWebhook(ctx context.Context, input *WebhookInput) (*WebhookResult, error) {
	startTime := time.Now()

	// Validate input
	if err := s.validateInput(input); err != nil {
		return &WebhookResult{Processed: false, Error: err.Error()}, err
	}

	// Verify signature and parse event
	event, err := s.verifyAndParseEvent(input.Payload, input.Signature)
	if err != nil {
		logging.Error(ctx, "[Stripe] Signature verification failed", map[string]any{"error": err})
		return &WebhookResult{Processed: false, Error: "signature verification failed"}, ErrInvalidSignature
	}

	// Check in-memory idempotency cache
	if s.wasEventProcessed(event.ID) {
		logging.Info(ctx, "[Stripe] Event already processed (cache)", map[string]any{"event_id": event.ID})
		return &WebhookResult{
			EventID:   event.ID,
			EventType: string(event.Type),
			Processed: true,
		}, nil
	}

	// Create webhook event record
	webhookEvent := s.createWebhookEvent(event, input)

	// Check database idempotency (if DB logging enabled)
	if s.config.EnableDBLogging && s.db != nil {
		existing, err := s.getEventByExternalID(ctx, event.ID)
		if err == nil && existing != nil {
			if existing.Status == models.WebhookStatusProcessed {
				logging.Info(ctx, "[Stripe] Event already processed (db)", map[string]any{"event_id": event.ID})
				return &WebhookResult{
					EventID:   event.ID,
					EventType: string(event.Type),
					Processed: true,
				}, nil
			}
			// Use existing record for retry
			webhookEvent = existing
		} else {
			// Save new event
			if err := s.saveWebhookEvent(ctx, webhookEvent); err != nil {
				logging.Error(ctx, "[Stripe] Failed to save webhook event", map[string]any{"error": err})
				// Continue processing even if DB save fails
			}
		}
	}

	// Process the event
	webhookEvent.MarkAsProcessing()
	if s.config.EnableDBLogging && s.db != nil {
		s.updateWebhookEvent(ctx, webhookEvent)
	}

	processingCtx, cancel := context.WithTimeout(ctx, s.config.ProcessingTimeout)
	defer cancel()

	processErr := s.processEvent(processingCtx, event, webhookEvent)

	processingTime := time.Since(startTime).Milliseconds()

	if processErr != nil {
		return s.handleProcessingError(ctx, event, webhookEvent, processErr, processingTime)
	}

	// Mark as processed
	webhookEvent.MarkAsProcessed(nil, processingTime)
	if s.config.EnableDBLogging && s.db != nil {
		s.updateWebhookEvent(ctx, webhookEvent)
	}

	// Mark in cache
	s.markEventProcessed(event.ID)

	logging.Info(ctx, "[Stripe] Event processed successfully", map[string]any{"event_id": event.ID, "event_type": event.Type, "processing_time_ms": processingTime})

	return &WebhookResult{
		EventID:   event.ID,
		EventType: string(event.Type),
		Processed: true,
	}, nil
}

// =============================================================================
// Event Handlers Registration
// =============================================================================

func (s *WebhookHandlerService) registerEventHandlers() {
	// Payment Intent Events
	s.processor.RegisterHandler(EventPaymentIntentSucceeded, s.handlePaymentIntentSucceeded)
	s.processor.RegisterHandler(EventPaymentIntentFailed, s.handlePaymentIntentFailed)
	s.processor.RegisterHandler(EventPaymentIntentCanceled, s.handlePaymentIntentCanceled)

	// Charge Events
	s.processor.RegisterHandler(EventChargeSucceeded, s.handleChargeSucceeded)
	s.processor.RegisterHandler(EventChargeFailed, s.handleChargeFailed)
	s.processor.RegisterHandler(EventChargeRefunded, s.handleChargeRefunded)
	s.processor.RegisterHandler(EventChargeDisputed, s.handleChargeDisputed)

	// Invoice Events
	s.processor.RegisterHandler(EventInvoicePaid, s.handleInvoicePaid)
	s.processor.RegisterHandler(EventInvoicePaymentFailed, s.handleInvoicePaymentFailed)

	// Subscription Events
	s.processor.RegisterHandler(EventSubscriptionCreated, s.handleSubscriptionCreated)
	s.processor.RegisterHandler(EventSubscriptionUpdated, s.handleSubscriptionUpdated)
	s.processor.RegisterHandler(EventSubscriptionDeleted, s.handleSubscriptionDeleted)

	// Customer Events
	s.processor.RegisterHandler(EventCustomerCreated, s.handleCustomerCreated)
	s.processor.RegisterHandler(EventCustomerUpdated, s.handleCustomerUpdated)
	s.processor.RegisterHandler(EventCustomerDeleted, s.handleCustomerDeleted)

	// Refund Events
	s.processor.RegisterHandler(EventRefundCreated, s.handleRefundCreated)
	s.processor.RegisterHandler(EventRefundUpdated, s.handleRefundUpdated)

	// Checkout Session Events
	s.processor.RegisterHandler(EventCheckoutSessionCompleted, s.handleCheckoutSessionCompleted)

	logging.Info(context.Background(), "[Stripe] Registered webhook event handlers", nil)
}

// =============================================================================
// Payment Intent Handlers
// =============================================================================

func (s *WebhookHandlerService) handlePaymentIntentSucceeded(ctx context.Context, event *stripe.Event) error {
	pi, err := ParsePaymentIntent(event)
	if err != nil {
		return fmt.Errorf("failed to parse payment intent: %w", err)
	}

	logging.Info(ctx, "[Stripe] Payment intent succeeded", map[string]any{"id": pi.ID, "amount": pi.Amount, "currency": pi.Currency})

	// Update payment record in database
	if s.db != nil {
		if err := s.updatePaymentStatus(ctx, pi.ID, models.PaymentStatusSucceeded, nil); err != nil {
			logging.Error(ctx, "[Stripe] Failed to update payment status", map[string]any{"error": err})
		}
	}

	// Send notification
	if s.config.EnableNotifications {
		if err := s.notifier.NotifyPaymentSucceeded(ctx, pi.ID, pi.Amount, pi.Currency); err != nil {
			logging.Error(ctx, "[Stripe] Failed to send payment success notification", map[string]any{"error": err})
		}
	}

	return nil
}

func (s *WebhookHandlerService) handlePaymentIntentFailed(ctx context.Context, event *stripe.Event) error {
	pi, err := ParsePaymentIntent(event)
	if err != nil {
		return fmt.Errorf("failed to parse payment intent: %w", err)
	}

	logging.Warn(ctx, "[Stripe] Payment intent failed", map[string]any{"id": pi.ID, "amount": pi.Amount, "currency": pi.Currency})

	// Update payment record
	if s.db != nil {
		failureMsg := "Payment failed"
		if err := s.updatePaymentStatus(ctx, pi.ID, models.PaymentStatusFailed, &failureMsg); err != nil {
			logging.Error(ctx, "[Stripe] Failed to update payment status", map[string]any{"error": err})
		}
	}

	// Send notification
	if s.config.EnableNotifications {
		if err := s.notifier.NotifyPaymentFailed(ctx, pi.ID, "Payment processing failed"); err != nil {
			logging.Error(ctx, "[Stripe] Failed to send payment failure notification", map[string]any{"error": err})
		}
	}

	return nil
}

func (s *WebhookHandlerService) handlePaymentIntentCanceled(ctx context.Context, event *stripe.Event) error {
	pi, err := ParsePaymentIntent(event)
	if err != nil {
		return fmt.Errorf("failed to parse payment intent: %w", err)
	}

	logging.Info(ctx, "[Stripe] Payment intent canceled", map[string]any{"id": pi.ID})

	if s.db != nil {
		if err := s.updatePaymentStatus(ctx, pi.ID, models.PaymentStatusCanceled, nil); err != nil {
			logging.Error(ctx, "[Stripe] Failed to update payment status", map[string]any{"error": err})
		}
	}

	return nil
}

// =============================================================================
// Charge Handlers
// =============================================================================

func (s *WebhookHandlerService) handleChargeSucceeded(ctx context.Context, event *stripe.Event) error {
	ch, err := ParseCharge(event)
	if err != nil {
		return fmt.Errorf("failed to parse charge: %w", err)
	}

	logging.Info(ctx, "[Stripe] Charge succeeded", map[string]any{"id": ch.ID, "amount": ch.Amount, "currency": ch.Currency})

	return nil
}

func (s *WebhookHandlerService) handleChargeFailed(ctx context.Context, event *stripe.Event) error {
	ch, err := ParseCharge(event)
	if err != nil {
		return fmt.Errorf("failed to parse charge: %w", err)
	}

	logging.Warn(ctx, "[Stripe] Charge failed", map[string]any{"id": ch.ID, "failure_code": ch.FailureCode, "failure_message": ch.FailureMessage})

	return nil
}

func (s *WebhookHandlerService) handleChargeRefunded(ctx context.Context, event *stripe.Event) error {
	ch, err := ParseCharge(event)
	if err != nil {
		return fmt.Errorf("failed to parse charge: %w", err)
	}

	logging.Info(ctx, "[Stripe] Charge refunded", map[string]any{"id": ch.ID, "amount": ch.Amount, "currency": ch.Currency})

	// Update payment record
	if s.db != nil && ch.PaymentIntentID != "" {
		status := models.PaymentStatusRefunded
		if !ch.Refunded {
			status = models.PaymentStatusPartiallyRefunded
		}
		if err := s.updatePaymentStatus(ctx, ch.PaymentIntentID, status, nil); err != nil {
			logging.Error(ctx, "[Stripe] Failed to update payment status", map[string]any{"error": err})
		}
	}

	return nil
}

func (s *WebhookHandlerService) handleChargeDisputed(ctx context.Context, event *stripe.Event) error {
	ch, err := ParseCharge(event)
	if err != nil {
		return fmt.Errorf("failed to parse charge: %w", err)
	}

	logging.Warn(ctx, "[Stripe] Charge disputed", map[string]any{"id": ch.ID, "amount": ch.Amount, "currency": ch.Currency})

	// Update payment record
	if s.db != nil && ch.PaymentIntentID != "" {
		if err := s.updatePaymentStatus(ctx, ch.PaymentIntentID, models.PaymentStatusDisputed, nil); err != nil {
			logging.Error(ctx, "[Stripe] Failed to update payment status", map[string]any{"error": err})
		}
	}

	// Critical event - send notification
	if s.config.EnableNotifications {
		if err := s.notifier.NotifyDisputeCreated(ctx, ch.ID, ch.Amount); err != nil {
			logging.Error(ctx, "[Stripe] Failed to send dispute notification", map[string]any{"error": err})
		}
	}

	return nil
}

// =============================================================================
// Invoice Handlers
// =============================================================================

func (s *WebhookHandlerService) handleInvoicePaid(ctx context.Context, event *stripe.Event) error {
	var invoice stripe.Invoice
	if err := json.Unmarshal(event.Data.Raw, &invoice); err != nil {
		return fmt.Errorf("failed to parse invoice: %w", err)
	}

	logging.Info(ctx, "[Stripe] Invoice paid", map[string]any{"id": invoice.ID, "amount": invoice.AmountPaid, "currency": invoice.Currency})

	return nil
}

func (s *WebhookHandlerService) handleInvoicePaymentFailed(ctx context.Context, event *stripe.Event) error {
	var invoice stripe.Invoice
	if err := json.Unmarshal(event.Data.Raw, &invoice); err != nil {
		return fmt.Errorf("failed to parse invoice: %w", err)
	}

	logging.Warn(ctx, "[Stripe] Invoice payment failed", map[string]any{"id": invoice.ID, "amount": invoice.AmountDue, "currency": invoice.Currency})

	return nil
}

// =============================================================================
// Subscription Handlers
// =============================================================================

func (s *WebhookHandlerService) handleSubscriptionCreated(ctx context.Context, event *stripe.Event) error {
	var sub stripe.Subscription
	if err := json.Unmarshal(event.Data.Raw, &sub); err != nil {
		return fmt.Errorf("failed to parse subscription: %w", err)
	}

	logging.Info(ctx, "[Stripe] Subscription created", map[string]any{"id": sub.ID, "status": sub.Status})

	// Create/update subscription record
	if s.db != nil {
		if err := s.upsertSubscription(ctx, &sub); err != nil {
			logging.Error(ctx, "[Stripe] Failed to save subscription", map[string]any{"error": err})
		}
	}

	return nil
}

func (s *WebhookHandlerService) handleSubscriptionUpdated(ctx context.Context, event *stripe.Event) error {
	var sub stripe.Subscription
	if err := json.Unmarshal(event.Data.Raw, &sub); err != nil {
		return fmt.Errorf("failed to parse subscription: %w", err)
	}

	logging.Info(ctx, "[Stripe] Subscription updated", map[string]any{"id": sub.ID, "status": sub.Status})

	if s.db != nil {
		if err := s.upsertSubscription(ctx, &sub); err != nil {
			logging.Error(ctx, "[Stripe] Failed to update subscription", map[string]any{"error": err})
		}
	}

	return nil
}

func (s *WebhookHandlerService) handleSubscriptionDeleted(ctx context.Context, event *stripe.Event) error {
	var sub stripe.Subscription
	if err := json.Unmarshal(event.Data.Raw, &sub); err != nil {
		return fmt.Errorf("failed to parse subscription: %w", err)
	}

	logging.Info(ctx, "[Stripe] Subscription deleted", map[string]any{"id": sub.ID})

	if s.db != nil {
		if err := s.upsertSubscription(ctx, &sub); err != nil {
			logging.Error(ctx, "[Stripe] Failed to update subscription", map[string]any{"error": err})
		}
	}

	// Critical event - send notification
	if s.config.EnableNotifications && sub.Customer != nil {
		if err := s.notifier.NotifySubscriptionCanceled(ctx, sub.ID, sub.Customer.ID); err != nil {
			logging.Error(ctx, "[Stripe] Failed to send subscription canceled notification", map[string]any{"error": err})
		}
	}

	return nil
}

// =============================================================================
// Customer Handlers
// =============================================================================

func (s *WebhookHandlerService) handleCustomerCreated(ctx context.Context, event *stripe.Event) error {
	cust, err := ParseCustomer(event)
	if err != nil {
		return fmt.Errorf("failed to parse customer: %w", err)
	}

	logging.Info(ctx, "[Stripe] Customer created", map[string]any{"id": cust.ID, "email": cust.Email})

	return nil
}

func (s *WebhookHandlerService) handleCustomerUpdated(ctx context.Context, event *stripe.Event) error {
	cust, err := ParseCustomer(event)
	if err != nil {
		return fmt.Errorf("failed to parse customer: %w", err)
	}

	logging.Info(ctx, "[Stripe] Customer updated", map[string]any{"id": cust.ID})

	return nil
}

func (s *WebhookHandlerService) handleCustomerDeleted(ctx context.Context, event *stripe.Event) error {
	cust, err := ParseCustomer(event)
	if err != nil {
		return fmt.Errorf("failed to parse customer: %w", err)
	}

	logging.Info(ctx, "[Stripe] Customer deleted", map[string]any{"id": cust.ID})

	return nil
}

// =============================================================================
// Refund Handlers
// =============================================================================

func (s *WebhookHandlerService) handleRefundCreated(ctx context.Context, event *stripe.Event) error {
	ref, err := ParseRefund(event)
	if err != nil {
		return fmt.Errorf("failed to parse refund: %w", err)
	}

	logging.Info(ctx, "[Stripe] Refund created", map[string]any{"id": ref.ID, "amount": ref.Amount, "currency": ref.Currency})

	// Send notification
	if s.config.EnableNotifications {
		if err := s.notifier.NotifyRefundCreated(ctx, ref.ID, ref.Amount); err != nil {
			logging.Error(ctx, "[Stripe] Failed to send refund notification", map[string]any{"error": err})
		}
	}

	return nil
}

func (s *WebhookHandlerService) handleRefundUpdated(ctx context.Context, event *stripe.Event) error {
	ref, err := ParseRefund(event)
	if err != nil {
		return fmt.Errorf("failed to parse refund: %w", err)
	}

	logging.Info(ctx, "[Stripe] Refund updated", map[string]any{"id": ref.ID, "status": ref.Status})

	return nil
}

// =============================================================================
// Checkout Session Handlers
// =============================================================================

func (s *WebhookHandlerService) handleCheckoutSessionCompleted(ctx context.Context, event *stripe.Event) error {
	session, err := ParseCheckoutSession(event)
	if err != nil {
		return fmt.Errorf("failed to parse checkout session: %w", err)
	}

	logging.Info(ctx, "[Stripe] Checkout session completed", map[string]any{"id": session.ID, "payment_status": session.PaymentStatus})

	// Check if this is an invoice payment
	paymentType, hasPaymentType := session.Metadata["payment_type"]
	invoiceID, hasInvoiceID := session.Metadata["invoice_id"]

	if !hasPaymentType || paymentType != "invoice" || !hasInvoiceID {
		logging.Info(ctx, "[Stripe] Checkout session is not an invoice payment, skipping", map[string]any{"session_id": session.ID})
		return nil
	}

	// Verify payment was successful
	if session.PaymentStatus != "paid" {
		logging.Info(ctx, "[Stripe] Checkout session payment status is not paid, not marking invoice as paid", map[string]any{"session_id": session.ID, "payment_status": session.PaymentStatus})
		return nil
	}

	// Handle the invoice payment
	if err := s.invoicePaymentHandler.HandleInvoicePaymentCompleted(
		ctx,
		invoiceID,
		session.AmountTotal,
		session.ID,
		session.PaymentIntentID,
	); err != nil {
		logging.Error(ctx, "[Stripe] Failed to handle invoice payment", map[string]any{"invoice_id": invoiceID, "error": err})
		return fmt.Errorf("failed to handle invoice payment: %w", err)
	}

	logging.Info(ctx, "[Stripe] Invoice marked as paid via checkout session", map[string]any{"invoice_id": invoiceID, "session_id": session.ID})

	// Send receipt email asynchronously
	go func() {
		bgCtx := context.Background()
		if err := s.invoicePaymentHandler.SendInvoiceReceiptEmail(bgCtx, invoiceID); err != nil {
			logging.Error(bgCtx, "[Stripe] Failed to send receipt email", map[string]any{"invoice_id": invoiceID, "error": err})
		}
	}()

	return nil
}

// =============================================================================
// Helper Methods
// =============================================================================

func (s *WebhookHandlerService) validateInput(input *WebhookInput) error {
	if input == nil {
		return errors.New("input cannot be nil")
	}
	if len(input.Payload) == 0 {
		return errors.New("payload is empty")
	}
	if input.Signature == "" {
		return errors.New("signature is missing")
	}
	return nil
}

func (s *WebhookHandlerService) verifyAndParseEvent(payload []byte, signature string) (*stripe.Event, error) {
	event, err := webhook.ConstructEvent(payload, signature, s.config.WebhookSecret)
	if err != nil {
		return nil, fmt.Errorf("signature verification failed: %w", err)
	}

	// Check timestamp for replay attacks
	if time.Since(time.Unix(event.Created, 0)) > s.config.WebhookTolerance {
		return nil, ErrEventExpired
	}

	return &event, nil
}

func (s *WebhookHandlerService) createWebhookEvent(event *stripe.Event, input *WebhookInput) *models.WebhookEvent {
	headersJSON, _ := json.Marshal(input.Headers)

	return &models.WebhookEvent{
		ID:              uuid.New(),
		Source:          models.WebhookSourceStripe,
		ExternalEventID: event.ID,
		EventType:       string(event.Type),
		Status:          models.WebhookStatusPending,
		Payload:         input.Payload,
		Headers:         headersJSON,
		IPAddress:       input.IPAddress,
		UserAgent:       input.UserAgent,
		MaxRetries:      s.config.MaxRetries,
	}
}

func (s *WebhookHandlerService) processEvent(ctx context.Context, event *stripe.Event, webhookEvent *models.WebhookEvent) error {
	return s.processor.ProcessEvent(ctx, event)
}

func (s *WebhookHandlerService) handleProcessingError(
	ctx context.Context,
	event *stripe.Event,
	webhookEvent *models.WebhookEvent,
	processErr error,
	processingTimeMs int64,
) (*WebhookResult, error) {
	errMsg := processErr.Error()

	logging.Error(ctx, "[Stripe] Event processing failed", map[string]any{"event_id": event.ID, "event_type": event.Type, "error": processErr})

	// Check if we should retry
	if webhookEvent.CanRetry() {
		nextRetry := s.calculateNextRetry(webhookEvent.RetryCount)
		webhookEvent.MarkForRetry(nextRetry, errMsg)

		if s.config.EnableDBLogging && s.db != nil {
			s.updateWebhookEvent(ctx, webhookEvent)
		}

		// Queue for retry
		select {
		case s.retryQueue <- webhookEvent:
			logging.Info(ctx, "[Stripe] Event queued for retry", map[string]any{"event_id": event.ID, "attempt": webhookEvent.RetryCount, "next_retry": nextRetry})
		default:
			logging.Warn(ctx, "[Stripe] Retry queue full, event will not be retried", map[string]any{"event_id": event.ID})
		}

		return &WebhookResult{
			EventID:   event.ID,
			EventType: string(event.Type),
			Processed: false,
			Error:     "processing failed, will retry",
			WasRetry:  webhookEvent.RetryCount > 1,
		}, nil
	}

	// Max retries exceeded
	webhookEvent.MarkAsFailed(errMsg, "MAX_RETRIES_EXCEEDED")
	if s.config.EnableDBLogging && s.db != nil {
		s.updateWebhookEvent(ctx, webhookEvent)
	}

	// Notify about failure
	if s.config.EnableNotifications {
		s.notifier.NotifyWebhookError(ctx, event.ID, string(event.Type), processErr)
	}

	return &WebhookResult{
		EventID:   event.ID,
		EventType: string(event.Type),
		Processed: false,
		Error:     "processing failed after max retries",
	}, ErrMaxRetriesExceeded
}

func (s *WebhookHandlerService) calculateNextRetry(retryCount int) time.Time {
	// Exponential backoff with jitter
	delay := s.config.RetryBaseDelay * time.Duration(math.Pow(2, float64(retryCount)))
	if delay > s.config.RetryMaxDelay {
		delay = s.config.RetryMaxDelay
	}

	// Add jitter (up to 10%)
	jitter := time.Duration(float64(delay) * 0.1 * (0.5 - float64(time.Now().UnixNano()%100)/100))
	delay += jitter

	return time.Now().Add(delay)
}

// =============================================================================
// Idempotency Cache
// =============================================================================

func (s *WebhookHandlerService) wasEventProcessed(eventID string) bool {
	s.eventMu.RLock()
	defer s.eventMu.RUnlock()
	_, exists := s.processedEvents[eventID]
	return exists
}

func (s *WebhookHandlerService) markEventProcessed(eventID string) {
	s.eventMu.Lock()
	defer s.eventMu.Unlock()
	s.processedEvents[eventID] = time.Now()
}

// =============================================================================
// Database Operations
// =============================================================================

func (s *WebhookHandlerService) saveWebhookEvent(ctx context.Context, event *models.WebhookEvent) error {
	if s.db == nil {
		return nil
	}
	return s.db.WithContext(ctx).Create(event).Error
}

func (s *WebhookHandlerService) updateWebhookEvent(ctx context.Context, event *models.WebhookEvent) error {
	if s.db == nil {
		return nil
	}
	return s.db.WithContext(ctx).Save(event).Error
}

func (s *WebhookHandlerService) getEventByExternalID(ctx context.Context, externalID string) (*models.WebhookEvent, error) {
	if s.db == nil {
		return nil, errors.New("database not configured")
	}

	var event models.WebhookEvent
	err := s.db.WithContext(ctx).
		Where("source = ? AND external_event_id = ?", models.WebhookSourceStripe, externalID).
		First(&event).Error

	if err != nil {
		return nil, err
	}
	return &event, nil
}

func (s *WebhookHandlerService) updatePaymentStatus(ctx context.Context, stripePaymentID string, status models.PaymentStatus, failureMsg *string) error {
	if s.db == nil {
		return nil
	}

	updates := map[string]interface{}{
		"status":     status,
		"updated_at": time.Now(),
	}

	if failureMsg != nil {
		updates["failure_message"] = *failureMsg
	}

	if status == models.PaymentStatusSucceeded {
		now := time.Now()
		updates["paid_at"] = &now
	}

	return s.db.WithContext(ctx).
		Model(&models.PaymentRecord{}).
		Where("stripe_payment_id = ?", stripePaymentID).
		Updates(updates).Error
}

func (s *WebhookHandlerService) upsertSubscription(ctx context.Context, sub *stripe.Subscription) error {
	if s.db == nil {
		return nil
	}

	record := &models.SubscriptionRecord{
		StripeSubscriptionID: sub.ID,
		StripeCustomerID:     sub.Customer.ID,
		Status:               models.SubscriptionStatus(sub.Status),
		CurrentPeriodStart:   time.Unix(sub.CurrentPeriodStart, 0),
		CurrentPeriodEnd:     time.Unix(sub.CurrentPeriodEnd, 0),
		CancelAtPeriodEnd:    sub.CancelAtPeriodEnd,
	}

	if sub.CanceledAt > 0 {
		t := time.Unix(sub.CanceledAt, 0)
		record.CanceledAt = &t
	}

	if sub.EndedAt > 0 {
		t := time.Unix(sub.EndedAt, 0)
		record.EndedAt = &t
	}

	if sub.TrialStart > 0 {
		t := time.Unix(sub.TrialStart, 0)
		record.TrialStart = &t
	}

	if sub.TrialEnd > 0 {
		t := time.Unix(sub.TrialEnd, 0)
		record.TrialEnd = &t
	}

	if len(sub.Items.Data) > 0 {
		record.StripePriceID = sub.Items.Data[0].Price.ID
	}

	return s.db.WithContext(ctx).
		Where("stripe_subscription_id = ?", sub.ID).
		Assign(record).
		FirstOrCreate(record).Error
}

// =============================================================================
// Background Processors
// =============================================================================

func (s *WebhookHandlerService) retryProcessor() {
	defer s.wg.Done()

	for {
		select {
		case <-s.stopChan:
			return
		case event := <-s.retryQueue:
			s.processRetry(event)
		}
	}
}

func (s *WebhookHandlerService) processRetry(webhookEvent *models.WebhookEvent) {
	if webhookEvent.NextRetryAt != nil && time.Now().Before(*webhookEvent.NextRetryAt) {
		// Not ready for retry yet, re-queue
		time.Sleep(time.Until(*webhookEvent.NextRetryAt))
	}

	ctx, cancel := context.WithTimeout(context.Background(), s.config.ProcessingTimeout)
	defer cancel()

	// Parse the stored event
	var stripeEvent stripe.Event
	if err := json.Unmarshal(webhookEvent.Payload, &stripeEvent); err != nil {
		logging.Error(ctx, "[Stripe] Failed to unmarshal event for retry", map[string]any{"error": err})
		webhookEvent.MarkAsFailed("failed to unmarshal event", "UNMARSHAL_ERROR")
		s.updateWebhookEvent(ctx, webhookEvent)
		return
	}

	// Process
	webhookEvent.MarkAsProcessing()
	s.updateWebhookEvent(ctx, webhookEvent)

	startTime := time.Now()
	err := s.processEvent(ctx, &stripeEvent, webhookEvent)
	processingTime := time.Since(startTime).Milliseconds()

	if err != nil {
		logging.Error(ctx, "[Stripe] Retry failed for event", map[string]any{"event_id": webhookEvent.ExternalEventID, "error": err})

		if webhookEvent.CanRetry() {
			nextRetry := s.calculateNextRetry(webhookEvent.RetryCount)
			webhookEvent.MarkForRetry(nextRetry, err.Error())
			s.updateWebhookEvent(ctx, webhookEvent)

			// Re-queue
			select {
			case s.retryQueue <- webhookEvent:
			default:
				logging.Warn(ctx, "[Stripe] Retry queue full, dropping event", map[string]any{"event_id": webhookEvent.ExternalEventID})
			}
		} else {
			webhookEvent.MarkAsFailed(err.Error(), "MAX_RETRIES_EXCEEDED")
			s.updateWebhookEvent(ctx, webhookEvent)

			if s.config.EnableNotifications {
				s.notifier.NotifyWebhookError(ctx, webhookEvent.ExternalEventID, webhookEvent.EventType, err)
			}
		}
		return
	}

	// Success
	webhookEvent.MarkAsProcessed(nil, processingTime)
	s.updateWebhookEvent(ctx, webhookEvent)
	s.markEventProcessed(webhookEvent.ExternalEventID)

	logging.Info(ctx, "[Stripe] Retry succeeded for event", map[string]any{"event_id": webhookEvent.ExternalEventID})
}

func (s *WebhookHandlerService) cleanupRoutine() {
	defer s.wg.Done()

	ticker := time.NewTicker(time.Hour)
	defer ticker.Stop()

	for {
		select {
		case <-s.stopChan:
			return
		case <-ticker.C:
			s.cleanupProcessedEvents()
		}
	}
}

func (s *WebhookHandlerService) cleanupProcessedEvents() {
	s.eventMu.Lock()
	defer s.eventMu.Unlock()

	cutoff := time.Now().Add(-24 * time.Hour)
	for eventID, processedAt := range s.processedEvents {
		if processedAt.Before(cutoff) {
			delete(s.processedEvents, eventID)
		}
	}

	logging.Info(context.Background(), "[Stripe] Cleaned up processed events cache", map[string]any{"count": len(s.processedEvents)})
}

// =============================================================================
// Query Methods
// =============================================================================

// GetWebhookEvents retrieves webhook events based on query options
func (s *WebhookHandlerService) GetWebhookEvents(ctx context.Context, opts *models.WebhookEventQueryOptions) ([]*models.WebhookEvent, error) {
	if s.db == nil {
		return nil, errors.New("database not configured")
	}

	query := s.db.WithContext(ctx).Model(&models.WebhookEvent{})

	if opts.Source != "" {
		query = query.Where("source = ?", opts.Source)
	}
	if opts.EventType != "" {
		query = query.Where("event_type = ?", opts.EventType)
	}
	if opts.Status != "" {
		query = query.Where("status = ?", opts.Status)
	}
	if opts.StartDate != nil {
		query = query.Where("created_at >= ?", opts.StartDate)
	}
	if opts.EndDate != nil {
		query = query.Where("created_at <= ?", opts.EndDate)
	}

	if opts.Limit > 0 {
		query = query.Limit(opts.Limit)
	}
	if opts.Offset > 0 {
		query = query.Offset(opts.Offset)
	}

	query = query.Order("created_at DESC")

	var events []*models.WebhookEvent
	if err := query.Find(&events).Error; err != nil {
		return nil, err
	}

	return events, nil
}

// GetWebhookStats retrieves webhook event statistics
func (s *WebhookHandlerService) GetWebhookStats(ctx context.Context) (*models.WebhookEventStats, error) {
	if s.db == nil {
		return nil, errors.New("database not configured")
	}

	stats := &models.WebhookEventStats{
		EventsByType: make(map[string]int64),
	}

	// Total counts by status
	var results []struct {
		Status string
		Count  int64
	}

	if err := s.db.WithContext(ctx).
		Model(&models.WebhookEvent{}).
		Select("status, count(*) as count").
		Group("status").
		Scan(&results).Error; err != nil {
		return nil, err
	}

	for _, r := range results {
		switch models.WebhookEventStatus(r.Status) {
		case models.WebhookStatusProcessed:
			stats.ProcessedEvents = r.Count
		case models.WebhookStatusFailed:
			stats.FailedEvents = r.Count
		case models.WebhookStatusPending:
			stats.PendingEvents = r.Count
		case models.WebhookStatusRetrying:
			stats.RetryingEvents = r.Count
		}
		stats.TotalEvents += r.Count
	}

	// Average processing time
	var avgResult struct {
		AvgTime float64
	}
	s.db.WithContext(ctx).
		Model(&models.WebhookEvent{}).
		Select("AVG(processing_time_ms) as avg_time").
		Where("processing_time_ms IS NOT NULL").
		Scan(&avgResult)
	stats.AvgProcessingMs = avgResult.AvgTime

	// Events by type
	var typeResults []struct {
		EventType string
		Count     int64
	}
	s.db.WithContext(ctx).
		Model(&models.WebhookEvent{}).
		Select("event_type, count(*) as count").
		Group("event_type").
		Scan(&typeResults)

	for _, r := range typeResults {
		stats.EventsByType[r.EventType] = r.Count
	}

	return stats, nil
}

// ReprocessEvent manually reprocesses a failed webhook event
func (s *WebhookHandlerService) ReprocessEvent(ctx context.Context, eventID uuid.UUID) error {
	if s.db == nil {
		return errors.New("database not configured")
	}

	var webhookEvent models.WebhookEvent
	if err := s.db.WithContext(ctx).First(&webhookEvent, eventID).Error; err != nil {
		return fmt.Errorf("event not found: %w", err)
	}

	// Reset retry count and status
	webhookEvent.RetryCount = 0
	webhookEvent.Status = models.WebhookStatusPending

	// Queue for processing
	select {
	case s.retryQueue <- &webhookEvent:
		logging.Info(ctx, "[Stripe] Event queued for reprocessing", map[string]any{"event_id": eventID})
	default:
		return errors.New("retry queue is full")
	}

	return nil
}
