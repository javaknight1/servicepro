package stripe

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"sync"

	"github.com/stripe/stripe-go/v76"
)

// EventHandler is a function that processes a specific event type
type EventHandler func(ctx context.Context, event *stripe.Event) error

// EventProcessor processes Stripe webhook events
type EventProcessor struct {
	handlers map[string][]EventHandler
	mu       sync.RWMutex

	// Event statistics
	stats     *EventStatistics
	statsLock sync.RWMutex
}

// EventStatistics tracks event processing statistics
type EventStatistics struct {
	TotalProcessed        int64
	SuccessfullyProcessed int64
	FailedProcessed       int64
	EventCounts           map[string]int64
}

// NewEventProcessor creates a new event processor
func NewEventProcessor() *EventProcessor {
	return &EventProcessor{
		handlers: make(map[string][]EventHandler),
		stats: &EventStatistics{
			EventCounts: make(map[string]int64),
		},
	}
}

// RegisterHandler registers a handler for a specific event type
func (p *EventProcessor) RegisterHandler(eventType string, handler EventHandler) {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.handlers[eventType] = append(p.handlers[eventType], handler)
	log.Printf("[Stripe] Registered handler for event type: %s", eventType)
}

// ProcessEvent processes a Stripe webhook event
func (p *EventProcessor) ProcessEvent(ctx context.Context, event *stripe.Event) error {
	if event == nil {
		return errors.New("event cannot be nil")
	}

	p.statsLock.Lock()
	p.stats.TotalProcessed++
	p.stats.EventCounts[string(event.Type)]++
	p.statsLock.Unlock()

	log.Printf("[Stripe] Processing event: id=%s, type=%s", event.ID, event.Type)

	// Get handlers for this event type
	p.mu.RLock()
	handlers, exists := p.handlers[string(event.Type)]
	p.mu.RUnlock()

	if !exists || len(handlers) == 0 {
		log.Printf("[Stripe] No handlers registered for event type: %s", event.Type)
		// Not an error - just no handlers registered
		p.statsLock.Lock()
		p.stats.SuccessfullyProcessed++
		p.statsLock.Unlock()
		return nil
	}

	// Execute all handlers for this event type
	var lastErr error
	for _, handler := range handlers {
		if err := handler(ctx, event); err != nil {
			log.Printf("[Stripe] Handler failed for event %s (type %s): %v",
				event.ID, event.Type, err)
			lastErr = err
			// Continue processing other handlers
		}
	}

	if lastErr != nil {
		p.statsLock.Lock()
		p.stats.FailedProcessed++
		p.statsLock.Unlock()
		return fmt.Errorf("one or more handlers failed: %w", lastErr)
	}

	p.statsLock.Lock()
	p.stats.SuccessfullyProcessed++
	p.statsLock.Unlock()

	log.Printf("[Stripe] Event processed successfully: id=%s, type=%s", event.ID, event.Type)

	return nil
}

// GetStatistics returns event processing statistics
func (p *EventProcessor) GetStatistics() *EventStatistics {
	p.statsLock.RLock()
	defer p.statsLock.RUnlock()

	// Create a copy
	stats := &EventStatistics{
		TotalProcessed:        p.stats.TotalProcessed,
		SuccessfullyProcessed: p.stats.SuccessfullyProcessed,
		FailedProcessed:       p.stats.FailedProcessed,
		EventCounts:           make(map[string]int64),
	}

	for k, v := range p.stats.EventCounts {
		stats.EventCounts[k] = v
	}

	return stats
}

// ResetStatistics resets all statistics (for testing)
func (p *EventProcessor) ResetStatistics() {
	p.statsLock.Lock()
	defer p.statsLock.Unlock()

	p.stats = &EventStatistics{
		EventCounts: make(map[string]int64),
	}
}

// Common Event Type Constants
const (
	// Payment Intent Events
	EventPaymentIntentSucceeded               = "payment_intent.succeeded"
	EventPaymentIntentFailed                  = "payment_intent.payment_failed"
	EventPaymentIntentCreated                 = "payment_intent.created"
	EventPaymentIntentCanceled                = "payment_intent.canceled"
	EventPaymentIntentProcessing              = "payment_intent.processing"
	EventPaymentIntentRequiresAction          = "payment_intent.requires_action"
	EventPaymentIntentAmountCapturableUpdated = "payment_intent.amount_capturable_updated"

	// Charge Events
	EventChargeSucceeded = "charge.succeeded"
	EventChargeFailed    = "charge.failed"
	EventChargeRefunded  = "charge.refunded"
	EventChargeDisputed  = "charge.dispute.created"
	EventChargeCaptured  = "charge.captured"
	EventChargeUpdated   = "charge.updated"

	// Customer Events
	EventCustomerCreated = "customer.created"
	EventCustomerUpdated = "customer.updated"
	EventCustomerDeleted = "customer.deleted"

	// Payment Method Events
	EventPaymentMethodAttached = "payment_method.attached"
	EventPaymentMethodDetached = "payment_method.detached"
	EventPaymentMethodUpdated  = "payment_method.updated"

	// Refund Events
	EventRefundCreated = "refund.created"
	EventRefundUpdated = "refund.updated"
	EventRefundFailed  = "refund.failed"

	// Invoice Events
	EventInvoiceCreated               = "invoice.created"
	EventInvoiceFinalized             = "invoice.finalized"
	EventInvoicePaid                  = "invoice.paid"
	EventInvoicePaymentFailed         = "invoice.payment_failed"
	EventInvoicePaymentActionRequired = "invoice.payment_action_required"

	// Subscription Events
	EventSubscriptionCreated = "customer.subscription.created"
	EventSubscriptionUpdated = "customer.subscription.updated"
	EventSubscriptionDeleted = "customer.subscription.deleted"
)

// PaymentIntentData represents parsed payment intent data from an event
type PaymentIntentData struct {
	ID                 string
	Amount             int64
	Currency           string
	Status             string
	CustomerID         string
	PaymentMethodID    string
	Description        string
	Metadata           map[string]string
	CaptureMethod      string
	ConfirmationMethod string
}

// ChargeData represents parsed charge data from an event
type ChargeData struct {
	ID              string
	Amount          int64
	Currency        string
	Status          string
	CustomerID      string
	PaymentIntentID string
	Description     string
	Metadata        map[string]string
	Paid            bool
	Refunded        bool
	FailureCode     string
	FailureMessage  string
}

// CustomerData represents parsed customer data from an event
type CustomerData struct {
	ID          string
	Email       string
	Name        string
	Description string
	Metadata    map[string]string
}

// RefundData represents parsed refund data from an event
type RefundData struct {
	ID              string
	Amount          int64
	Currency        string
	ChargeID        string
	PaymentIntentID string
	Reason          string
	Status          string
	Metadata        map[string]string
}

// ParsePaymentIntent parses a payment intent from an event
func ParsePaymentIntent(event *stripe.Event) (*PaymentIntentData, error) {
	var pi stripe.PaymentIntent
	if err := json.Unmarshal(event.Data.Raw, &pi); err != nil {
		return nil, fmt.Errorf("failed to parse payment intent: %w", err)
	}

	data := &PaymentIntentData{
		ID:                 pi.ID,
		Amount:             pi.Amount,
		Currency:           string(pi.Currency),
		Status:             string(pi.Status),
		Description:        pi.Description,
		Metadata:           pi.Metadata,
		CaptureMethod:      string(pi.CaptureMethod),
		ConfirmationMethod: string(pi.ConfirmationMethod),
	}

	if pi.Customer != nil {
		data.CustomerID = pi.Customer.ID
	}

	if pi.PaymentMethod != nil {
		data.PaymentMethodID = pi.PaymentMethod.ID
	}

	return data, nil
}

// ParseCharge parses a charge from an event
func ParseCharge(event *stripe.Event) (*ChargeData, error) {
	var ch stripe.Charge
	if err := json.Unmarshal(event.Data.Raw, &ch); err != nil {
		return nil, fmt.Errorf("failed to parse charge: %w", err)
	}

	data := &ChargeData{
		ID:          ch.ID,
		Amount:      ch.Amount,
		Currency:    string(ch.Currency),
		Status:      string(ch.Status),
		Description: ch.Description,
		Metadata:    ch.Metadata,
		Paid:        ch.Paid,
		Refunded:    ch.Refunded,
	}

	if ch.Customer != nil {
		data.CustomerID = ch.Customer.ID
	}

	if ch.PaymentIntent != nil {
		data.PaymentIntentID = ch.PaymentIntent.ID
	}

	if ch.FailureCode != "" {
		data.FailureCode = ch.FailureCode
		data.FailureMessage = ch.FailureMessage
	}

	return data, nil
}

// ParseCustomer parses a customer from an event
func ParseCustomer(event *stripe.Event) (*CustomerData, error) {
	var cust stripe.Customer
	if err := json.Unmarshal(event.Data.Raw, &cust); err != nil {
		return nil, fmt.Errorf("failed to parse customer: %w", err)
	}

	return &CustomerData{
		ID:          cust.ID,
		Email:       cust.Email,
		Name:        cust.Name,
		Description: cust.Description,
		Metadata:    cust.Metadata,
	}, nil
}

// ParseRefund parses a refund from an event
func ParseRefund(event *stripe.Event) (*RefundData, error) {
	var ref stripe.Refund
	if err := json.Unmarshal(event.Data.Raw, &ref); err != nil {
		return nil, fmt.Errorf("failed to parse refund: %w", err)
	}

	data := &RefundData{
		ID:       ref.ID,
		Amount:   ref.Amount,
		Currency: string(ref.Currency),
		Reason:   string(ref.Reason),
		Status:   string(ref.Status),
		Metadata: ref.Metadata,
	}

	if ref.Charge != nil {
		data.ChargeID = ref.Charge.ID
	}

	if ref.PaymentIntent != nil {
		data.PaymentIntentID = ref.PaymentIntent.ID
	}

	return data, nil
}

// Default event handlers

// DefaultPaymentSucceededHandler is a default handler for successful payments
func DefaultPaymentSucceededHandler() EventHandler {
	return func(ctx context.Context, event *stripe.Event) error {
		pi, err := ParsePaymentIntent(event)
		if err != nil {
			return err
		}

		log.Printf("[Stripe] Payment succeeded: id=%s, amount=%d, currency=%s",
			pi.ID, pi.Amount, pi.Currency)

		return nil
	}
}

// DefaultPaymentFailedHandler is a default handler for failed payments
func DefaultPaymentFailedHandler() EventHandler {
	return func(ctx context.Context, event *stripe.Event) error {
		pi, err := ParsePaymentIntent(event)
		if err != nil {
			return err
		}

		log.Printf("[Stripe] Payment failed: id=%s, amount=%d, currency=%s",
			pi.ID, pi.Amount, pi.Currency)

		return nil
	}
}

// DefaultChargeRefundedHandler is a default handler for refunded charges
func DefaultChargeRefundedHandler() EventHandler {
	return func(ctx context.Context, event *stripe.Event) error {
		ch, err := ParseCharge(event)
		if err != nil {
			return err
		}

		log.Printf("[Stripe] Charge refunded: id=%s, amount=%d, currency=%s",
			ch.ID, ch.Amount, ch.Currency)

		return nil
	}
}

// DefaultCustomerCreatedHandler is a default handler for customer creation
func DefaultCustomerCreatedHandler() EventHandler {
	return func(ctx context.Context, event *stripe.Event) error {
		cust, err := ParseCustomer(event)
		if err != nil {
			return err
		}

		log.Printf("[Stripe] Customer created: id=%s, email=%s", cust.ID, cust.Email)

		return nil
	}
}

// RegisterDefaultHandlers registers default event handlers
func (p *EventProcessor) RegisterDefaultHandlers() {
	// Payment Intent Events
	p.RegisterHandler(EventPaymentIntentSucceeded, DefaultPaymentSucceededHandler())
	p.RegisterHandler(EventPaymentIntentFailed, DefaultPaymentFailedHandler())

	// Charge Events
	p.RegisterHandler(EventChargeRefunded, DefaultChargeRefundedHandler())

	// Customer Events
	p.RegisterHandler(EventCustomerCreated, DefaultCustomerCreatedHandler())

	log.Printf("[Stripe] Registered default event handlers")
}
