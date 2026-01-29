package services

import (
	"context"
	"errors"
	"fmt"
	"log"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stripe/stripe-go/v76"
	"github.com/stripe/stripe-go/v76/checkout/session"

	"github.com/javaknight1/servicepro/backend/internal/models"
	stripeClient "github.com/javaknight1/servicepro/backend/internal/services/stripe"
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

// CreateCheckoutSessionForInvoice creates a Stripe Checkout session for an invoice payment
func (s *InvoicePaymentService) CreateCheckoutSessionForInvoice(ctx context.Context, token string) (*CheckoutSessionResult, error) {
	// Validate the payment token and get the invoice
	invoice, err := s.invoiceService.ValidatePaymentToken(ctx, token)
	if err != nil {
		return nil, err
	}

	// Build line items from invoice lines
	lineItems := s.buildLineItems(invoice)

	// Build checkout session params
	params := &stripe.CheckoutSessionParams{
		Mode:       stripe.String(string(stripe.CheckoutSessionModePayment)),
		SuccessURL: stripe.String(fmt.Sprintf("%s?session_id={CHECKOUT_SESSION_ID}", s.successURL)),
		CancelURL:  stripe.String(s.cancelURL),
		LineItems:  lineItems,
		Metadata: map[string]string{
			"invoice_id":   invoice.ID.String(),
			"payment_type": "invoice",
		},
		PaymentIntentData: &stripe.CheckoutSessionPaymentIntentDataParams{
			Metadata: map[string]string{
				"invoice_id":   invoice.ID.String(),
				"payment_type": "invoice",
			},
		},
	}

	// Add customer email if available
	if invoice.Customer != nil && invoice.Customer.Email != "" {
		params.CustomerEmail = stripe.String(invoice.Customer.Email)
	}

	// Create the checkout session
	sess, err := session.New(params)
	if err != nil {
		log.Printf("[INVOICE-PAYMENT] Failed to create checkout session for invoice %s: %v", invoice.ID, err)
		return nil, fmt.Errorf("%w: %v", ErrCheckoutSessionCreationFailed, err)
	}

	log.Printf("[INVOICE-PAYMENT] Created checkout session %s for invoice %s", sess.ID, invoice.ID)

	return &CheckoutSessionResult{
		SessionID:   sess.ID,
		URL:         sess.URL,
		InvoiceID:   invoice.ID,
		TotalAmount: invoice.TotalAmount,
	}, nil
}

// buildLineItems converts invoice lines to Stripe checkout line items
func (s *InvoicePaymentService) buildLineItems(invoice *models.Invoice) []*stripe.CheckoutSessionLineItemParams {
	lineItems := make([]*stripe.CheckoutSessionLineItemParams, 0, len(invoice.Lines)+1)

	for _, line := range invoice.Lines {
		// Calculate unit amount in cents
		unitAmount := stripeClient.ConvertDecimalToCents(line.UnitPrice)
		quantity := line.Quantity.IntPart()
		if quantity <= 0 {
			quantity = 1
		}

		lineItem := &stripe.CheckoutSessionLineItemParams{
			PriceData: &stripe.CheckoutSessionLineItemPriceDataParams{
				Currency: stripe.String("usd"),
				ProductData: &stripe.CheckoutSessionLineItemPriceDataProductDataParams{
					Name:        stripe.String(line.Description),
					Description: stripe.String(fmt.Sprintf("Invoice: %s", invoice.InvoiceNumber)),
				},
				UnitAmount: stripe.Int64(unitAmount),
			},
			Quantity: stripe.Int64(quantity),
		}
		lineItems = append(lineItems, lineItem)
	}

	// Add tax as a separate line item if applicable
	if invoice.TaxAmount.GreaterThan(decimal.Zero) {
		taxAmount := stripeClient.ConvertDecimalToCents(invoice.TaxAmount)
		taxLineItem := &stripe.CheckoutSessionLineItemParams{
			PriceData: &stripe.CheckoutSessionLineItemPriceDataParams{
				Currency: stripe.String("usd"),
				ProductData: &stripe.CheckoutSessionLineItemPriceDataProductDataParams{
					Name:        stripe.String("Tax"),
					Description: stripe.String(fmt.Sprintf("Tax for Invoice: %s", invoice.InvoiceNumber)),
				},
				UnitAmount: stripe.Int64(taxAmount),
			},
			Quantity: stripe.Int64(1),
		}
		lineItems = append(lineItems, taxLineItem)
	}

	return lineItems
}

// GetInvoiceByToken retrieves an invoice by payment token for display purposes
func (s *InvoicePaymentService) GetInvoiceByToken(ctx context.Context, token string) (*models.Invoice, error) {
	return s.invoiceService.GetInvoiceByPaymentToken(ctx, token)
}

// ValidateToken validates a payment token without creating a checkout session
func (s *InvoicePaymentService) ValidateToken(ctx context.Context, token string) (*models.Invoice, error) {
	return s.invoiceService.ValidatePaymentToken(ctx, token)
}
