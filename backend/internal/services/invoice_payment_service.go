package services

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stripe/stripe-go/v76"
	"github.com/stripe/stripe-go/v76/checkout/session"

	"github.com/javaknight1/servicepro/backend/internal/models"
	stripeClient "github.com/javaknight1/servicepro/backend/internal/services/stripe"
	"github.com/javaknight1/servicepro/backend/pkg/clients/logging"
)

var (
	// ErrCheckoutSessionCreationFailed is returned when Stripe checkout session creation fails
	ErrCheckoutSessionCreationFailed = errors.New("failed to create checkout session")
	// ErrInvoicePaymentDisabled is returned when invoice payment is not configured
	ErrInvoicePaymentDisabled = errors.New("invoice payment is not configured")
)

// InvoicePaymentService handles invoice payment operations via Stripe
type InvoicePaymentService struct {
	stripeClient   *stripeClient.Client
	invoiceService *InvoiceService
	successURL     string
	cancelURL      string
}

// InvoicePaymentServiceConfig holds configuration for the invoice payment service
type InvoicePaymentServiceConfig struct {
	StripeClient   *stripeClient.Client
	InvoiceService *InvoiceService
	SuccessURL     string // URL to redirect after successful payment
	CancelURL      string // URL to redirect if payment is cancelled
}

// NewInvoicePaymentService creates a new invoice payment service
func NewInvoicePaymentService(cfg *InvoicePaymentServiceConfig) (*InvoicePaymentService, error) {
	if cfg == nil {
		return nil, errors.New("config is required")
	}
	if cfg.StripeClient == nil {
		return nil, errors.New("stripe client is required")
	}
	if cfg.InvoiceService == nil {
		return nil, errors.New("invoice service is required")
	}
	if cfg.SuccessURL == "" {
		return nil, errors.New("success URL is required")
	}
	if cfg.CancelURL == "" {
		return nil, errors.New("cancel URL is required")
	}

	return &InvoicePaymentService{
		stripeClient:   cfg.StripeClient,
		invoiceService: cfg.InvoiceService,
		successURL:     cfg.SuccessURL,
		cancelURL:      cfg.CancelURL,
	}, nil
}

// CheckoutSessionResult contains the result of creating a checkout session
type CheckoutSessionResult struct {
	SessionID   string
	URL         string
	InvoiceID   uuid.UUID
	TotalAmount decimal.Decimal
}

// CreateCheckoutSessionForInvoice creates a Stripe Checkout session that
// charges exactly the invoice's balance due (T119 / audit 7.C2).
func (s *InvoicePaymentService) CreateCheckoutSessionForInvoice(ctx context.Context, token string) (*CheckoutSessionResult, error) {
	// Validate the payment token and get the invoice (payable statuses only,
	// with a positive balance).
	invoice, err := s.invoiceService.ValidatePaymentToken(ctx, token)
	if err != nil {
		return nil, err
	}

	params, balance := s.buildCheckoutParams(invoice, invoiceCurrency(invoice), time.Now())
	if params == nil {
		return nil, ErrInvoiceNotPayable
	}

	// Create the checkout session
	sess, err := session.New(params)
	if err != nil {
		logging.Error(ctx, "[INVOICE-PAYMENT] Failed to create checkout session for invoice", map[string]any{"invoice_id": invoice.ID, "error": err})
		return nil, fmt.Errorf("%w: %v", ErrCheckoutSessionCreationFailed, err)
	}

	logging.Info(ctx, "[INVOICE-PAYMENT] Created checkout session for invoice", map[string]any{"session_id": sess.ID, "invoice_id": invoice.ID})

	return &CheckoutSessionResult{
		SessionID:   sess.ID,
		URL:         sess.URL,
		InvoiceID:   invoice.ID,
		TotalAmount: balance,
	}, nil
}

// invoiceCurrency is the ISO currency invoices are charged in. Today every
// tenant bills in USD; T127 makes this the tenant's currency setting.
func invoiceCurrency(_ *models.Invoice) string {
	return "usd"
}

// buildCheckoutParams builds a Checkout session with ONE line item for the
// balance due (total_amount - amount_paid), in the currency's minor units.
// Per-line items were wrong: fractional quantities were truncated, line and
// invoice discounts and earlier payments were ignored. It returns nil when
// nothing is owed.
//
// The idempotency key covers the invoice, the amount and the hour, so a
// double-click or client retry reuses one session instead of creating
// several, while a new payment link after a partial payment (different
// amount) or a later visit gets a fresh session.
func (s *InvoicePaymentService) buildCheckoutParams(invoice *models.Invoice, currency string, now time.Time) (*stripe.CheckoutSessionParams, decimal.Decimal) {
	balance := invoice.BalanceDue()
	units := stripeClient.ToMinorUnits(balance, currency)
	if units <= 0 {
		return nil, balance
	}

	metadata := map[string]string{
		"invoice_id":   invoice.ID.String(),
		"payment_type": "invoice",
	}
	params := &stripe.CheckoutSessionParams{
		Mode:       stripe.String(string(stripe.CheckoutSessionModePayment)),
		SuccessURL: stripe.String(fmt.Sprintf("%s?session_id={CHECKOUT_SESSION_ID}", s.successURL)),
		CancelURL:  stripe.String(s.cancelURL),
		LineItems: []*stripe.CheckoutSessionLineItemParams{{
			PriceData: &stripe.CheckoutSessionLineItemPriceDataParams{
				Currency: stripe.String(currency),
				ProductData: &stripe.CheckoutSessionLineItemPriceDataProductDataParams{
					Name:        stripe.String(fmt.Sprintf("Invoice %s", invoice.InvoiceNumber)),
					Description: stripe.String("Balance due"),
				},
				UnitAmount: stripe.Int64(units),
			},
			Quantity: stripe.Int64(1),
		}},
		Metadata: metadata,
		PaymentIntentData: &stripe.CheckoutSessionPaymentIntentDataParams{
			Metadata: metadata,
		},
	}
	if invoice.Customer != nil && invoice.Customer.Email != "" {
		params.CustomerEmail = stripe.String(invoice.Customer.Email)
	}
	params.SetIdempotencyKey(fmt.Sprintf("invoice-checkout:%s:%s:%d:%s",
		invoice.ID, currency, units, now.UTC().Format("2006010215")))
	return params, balance
}

// GetInvoiceByToken retrieves an invoice by payment token for display purposes
func (s *InvoicePaymentService) GetInvoiceByToken(ctx context.Context, token string) (*models.Invoice, error) {
	return s.invoiceService.GetInvoiceByPaymentToken(ctx, token)
}

// ValidateToken validates a payment token without creating a checkout session
func (s *InvoicePaymentService) ValidateToken(ctx context.Context, token string) (*models.Invoice, error) {
	return s.invoiceService.ValidatePaymentToken(ctx, token)
}
