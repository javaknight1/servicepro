package services

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stripe/stripe-go/v87"
	"github.com/stripe/stripe-go/v87/checkout/session"

	"github.com/javaknight1/servicepro/backend/internal/models"
	stripeClient "github.com/javaknight1/servicepro/backend/internal/services/stripe"
	"github.com/javaknight1/servicepro/backend/pkg/clients/logging"
)

var (
	// ErrCheckoutSessionCreationFailed is returned when Stripe checkout session creation fails
	ErrCheckoutSessionCreationFailed = errors.New("failed to create checkout session")
	// ErrInvoicePaymentDisabled is returned when online invoice payment is
	// switched off (INVOICE_PAYMENTS_ENABLED, T261) or the invoice's tenant
	// can't take payments on its own Stripe account yet (T169).
	ErrInvoicePaymentDisabled = errors.New("online invoice payment is disabled")
	// ErrPaymentMethodUnavailable is returned when the customer asks for a
	// method (bank) the tenant's account can't take yet.
	ErrPaymentMethodUnavailable = errors.New("payment method is not available for this invoice")
	// ErrInvalidPaymentMethod is returned for a method other than bank/card.
	ErrInvalidPaymentMethod = errors.New("invalid payment method")
	// ErrPaymentPending is returned while a bank payment for the invoice is
	// still settling (T170).
	ErrPaymentPending = errors.New("a payment for this invoice is already processing")
)

// ConnectedAccountLookup returns a tenant's Stripe connected account (nil
// when it has none). Implemented by PaymentsConnectService.
type ConnectedAccountLookup interface {
	Account(ctx context.Context, tenantID uuid.UUID) (*models.StripeConnectedAccount, error)
}

// InvoicePaymentService creates Stripe Checkout sessions for invoice
// payments as direct charges on the tenant's own Stripe account (D9, T169).
type InvoicePaymentService struct {
	stripeClient   *stripeClient.Client
	invoiceService *InvoiceService
	accounts       ConnectedAccountLookup
	successURL     string
	cancelURL      string
	enabled        bool
	// newSession creates the session (session.New; replaced in tests).
	newSession func(*stripe.CheckoutSessionParams) (*stripe.CheckoutSession, error)
}

// InvoicePaymentServiceConfig holds configuration for the invoice payment service
type InvoicePaymentServiceConfig struct {
	StripeClient   *stripeClient.Client
	InvoiceService *InvoiceService
	// Accounts looks up the invoice's tenant's connected account. Without
	// it no session is ever created.
	Accounts   ConnectedAccountLookup
	SuccessURL string // URL to redirect after successful payment
	CancelURL  string // URL to redirect if payment is cancelled
	// Enabled is the INVOICE_PAYMENTS_ENABLED master switch (T261); each
	// tenant also needs an account that can take cards (T169).
	Enabled bool
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
		accounts:       cfg.Accounts,
		successURL:     cfg.SuccessURL,
		cancelURL:      cfg.CancelURL,
		enabled:        cfg.Enabled,
		newSession:     session.New,
	}, nil
}

// CheckoutSessionResult contains the result of creating a checkout session
type CheckoutSessionResult struct {
	SessionID      string
	URL            string
	InvoiceID      uuid.UUID
	TotalAmount    decimal.Decimal
	Method         string
	ApplicationFee int64 // minor units
}

// CreateCheckoutSessionForInvoice creates a Stripe Checkout session ON the
// tenant's connected account (a direct charge) for exactly the invoice's
// balance due (T119 / audit 7.C2), with ServicePro's application fee for the
// chosen method (D10). method is "bank", "card" or "" (bank when the
// account takes ACH, else card).
func (s *InvoicePaymentService) CreateCheckoutSessionForInvoice(ctx context.Context, token, method string) (*CheckoutSessionResult, error) {
	// Checked first: with the master switch off no session is ever created (T261).
	if !s.enabled || s.accounts == nil {
		return nil, ErrInvoicePaymentDisabled
	}
	method = strings.ToLower(strings.TrimSpace(method))
	if method != "" && method != stripeClient.PayMethodBank && method != stripeClient.PayMethodCard {
		return nil, ErrInvalidPaymentMethod
	}

	// Validate the payment token and get the invoice (payable statuses only,
	// with a positive balance).
	invoice, err := s.invoiceService.ValidatePaymentToken(ctx, token)
	if err != nil {
		return nil, err
	}
	// A bank payment is already settling: a second one would overpay (T170).
	if invoice.PaymentPendingAt != nil {
		return nil, ErrPaymentPending
	}

	// The money goes to the tenant's own Stripe account: no account that
	// can take cards, no session (T169).
	acct, err := s.accounts.Account(ctx, invoice.TenantID)
	if err != nil {
		return nil, fmt.Errorf("load connected account: %w", err)
	}
	if !acct.CanAcceptCards() {
		return nil, ErrInvoicePaymentDisabled
	}
	switch {
	case method == "" && acct.CanAcceptACH():
		method = stripeClient.PayMethodBank
	case method == "":
		method = stripeClient.PayMethodCard
	case method == stripeClient.PayMethodBank && !acct.CanAcceptACH():
		return nil, ErrPaymentMethodUnavailable
	}

	params, balance, fee := s.buildCheckoutParams(invoice, invoiceCurrency(invoice), time.Now(), acct.StripeAccountID, method)
	if params == nil {
		return nil, ErrInvoiceNotPayable
	}

	sess, err := s.newSession(params)
	if err != nil {
		logging.Error(ctx, "[INVOICE-PAYMENT] Failed to create checkout session for invoice", map[string]any{"invoice_id": invoice.ID, "method": method, "error": err})
		return nil, fmt.Errorf("%w: %w", ErrCheckoutSessionCreationFailed, err)
	}

	logging.Info(ctx, "[INVOICE-PAYMENT] Created direct-charge checkout session for invoice", map[string]any{
		"session_id": sess.ID, "invoice_id": invoice.ID, "tenant_id": invoice.TenantID, "method": method, "application_fee": fee,
	})

	return &CheckoutSessionResult{
		SessionID:      sess.ID,
		URL:            sess.URL,
		InvoiceID:      invoice.ID,
		TotalAmount:    balance,
		Method:         method,
		ApplicationFee: fee,
	}, nil
}

// invoiceCurrency is the Stripe (lower-case ISO 4217) currency an invoice is
// charged in: the currency stored on the invoice at create time from the
// tenant's setting (T127), USD for anything unusable.
func invoiceCurrency(invoice *models.Invoice) string {
	return strings.ToLower(models.NormalizeCurrency(invoice.Currency))
}

// buildCheckoutParams builds a Checkout session with ONE line item for the
// balance due (total_amount - amount_paid), in the currency's minor units,
// as a direct charge on accountID (Stripe-Account header) with ServicePro's
// application fee for method. Bank sessions offer only us_bank_account;
// card sessions offer the account's dynamic methods except ACH, so the fee
// always matches the method actually used. It returns nil when nothing is
// owed.
//
// Per-line items were wrong: fractional quantities were truncated, line and
// invoice discounts and earlier payments were ignored.
//
// The idempotency key covers the invoice, account, method, amount and the
// hour, so a double-click or client retry reuses one session instead of
// creating several, while a new payment link after a partial payment
// (different amount), another method or a later visit gets a fresh session.
func (s *InvoicePaymentService) buildCheckoutParams(invoice *models.Invoice, currency string, now time.Time, accountID, method string) (*stripe.CheckoutSessionParams, decimal.Decimal, int64) {
	balance := invoice.BalanceDue()
	units := stripeClient.ToMinorUnits(balance, currency)
	if units <= 0 {
		return nil, balance, 0
	}
	fee := stripeClient.ApplicationFee(units, currency, method)

	metadata := map[string]string{
		"invoice_id":   invoice.ID.String(),
		"tenant_id":    invoice.TenantID.String(),
		"payment_type": "invoice",
		"pay_method":   method,
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
	if fee > 0 {
		params.PaymentIntentData.ApplicationFeeAmount = stripe.Int64(fee)
	}
	if method == stripeClient.PayMethodBank {
		params.AllowedPaymentMethodTypes = stripe.StringSlice([]string{"us_bank_account"})
	} else {
		params.ExcludedPaymentMethodTypes = stripe.StringSlice([]string{"us_bank_account"})
	}
	if invoice.Customer != nil && invoice.Customer.Email != "" {
		params.CustomerEmail = stripe.String(invoice.Customer.Email)
	}
	params.SetStripeAccount(accountID)
	params.SetIdempotencyKey(fmt.Sprintf("invoice-checkout:%s:%s:%s:%s:%d:%s",
		invoice.ID, accountID, method, currency, units, now.UTC().Format("2006010215")))
	return params, balance, fee
}

// ValidateToken validates a payment token without creating a checkout session
func (s *InvoicePaymentService) ValidateToken(ctx context.Context, token string) (*models.Invoice, error) {
	return s.invoiceService.ValidatePaymentToken(ctx, token)
}

// PayOption is one way a customer can pay an invoice on the pay page
// (mockup 17), with what ServicePro's fee for it costs the business.
type PayOption struct {
	Method      string          // stripeClient.PayMethodBank or PayMethodCard
	BusinessFee decimal.Decimal // major units of the invoice currency
}

// PayPage is what the public pay page shows: the invoice, the business that
// sent it, and the methods it can be paid with right now.
type PayPage struct {
	Invoice          *models.Invoice
	BusinessName     string
	BusinessLocation string      // "Columbus, OH"; "" when not set
	Methods          []PayOption // bank first; empty when it can't be paid online
	// PaymentPending: a bank payment is settling; no methods are offered.
	PaymentPending bool
}

// GetPayPage loads what the public pay page needs for a payment token. The
// business is the invoice's own tenant (never a client-supplied ID). Methods
// are listed only when online payment is on, the link is still valid, the
// invoice has a balance, and the tenant's own Stripe account can take them;
// each carries the fee CreateCheckoutSessionForInvoice would charge.
func (s *InvoicePaymentService) GetPayPage(ctx context.Context, token string) (*PayPage, error) {
	invoice, err := s.invoiceService.GetInvoiceByPaymentToken(ctx, token)
	if err != nil {
		return nil, err
	}
	page := &PayPage{Invoice: invoice, Methods: []PayOption{}}

	var tenant models.Tenant
	if err := s.invoiceService.db.WithContext(ctx).Select("id", "name", "settings").
		First(&tenant, "id = ?", invoice.TenantID).Error; err != nil {
		return nil, fmt.Errorf("load invoice business: %w", err)
	}
	page.BusinessName = tenant.Name
	page.BusinessLocation = businessLocation(tenant.Settings)

	page.PaymentPending = invoice.PaymentPendingAt != nil
	if page.PaymentPending || !s.enabled || s.accounts == nil || !invoice.IsPaymentTokenValid() {
		return page, nil
	}
	acct, err := s.accounts.Account(ctx, invoice.TenantID)
	if err != nil {
		return nil, fmt.Errorf("load connected account: %w", err)
	}
	if !acct.CanAcceptCards() {
		return page, nil
	}
	currency := invoiceCurrency(invoice)
	units := stripeClient.ToMinorUnits(invoice.BalanceDue(), currency)
	option := func(method string) PayOption {
		fee := stripeClient.ApplicationFee(units, currency, method)
		return PayOption{Method: method, BusinessFee: stripeClient.FromMinorUnits(fee, currency)}
	}
	if acct.CanAcceptACH() {
		page.Methods = append(page.Methods, option(stripeClient.PayMethodBank))
	}
	page.Methods = append(page.Methods, option(stripeClient.PayMethodCard))
	return page, nil
}

// businessLocation is "City, ST" from the tenant's settings, or whichever
// half is set.
func businessLocation(st models.TenantSettings) string {
	city, state := strings.TrimSpace(st.BusinessCity), strings.TrimSpace(st.BusinessState)
	switch {
	case city != "" && state != "":
		return city + ", " + state
	case city != "":
		return city
	}
	return state
}
