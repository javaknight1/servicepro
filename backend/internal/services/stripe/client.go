package stripe

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stripe/stripe-go/v76"
	"github.com/stripe/stripe-go/v76/customer"
	"github.com/stripe/stripe-go/v76/paymentintent"
	"github.com/stripe/stripe-go/v76/refund"

	"github.com/javaknight1/servicepro/backend/config"
	"github.com/javaknight1/servicepro/backend/pkg/clients/logging"
)

// Default configuration values
const (
	defaultMaxNetworkRetries = 3
	defaultRequestsPerSecond = 100
)

// Client is the main Stripe client wrapper
type Client struct {
	secretKey      string
	publishableKey string
	webhookSecret  string
	environment    string
	mu             sync.RWMutex

	// Rate limiting
	rateLimiter *rateLimiter
}

// NewClient creates a new Stripe client
func NewClient(cfg *config.Config) (*Client, error) {
	if cfg == nil {
		return nil, errors.New("config is required")
	}

	if cfg.Stripe.SecretKey == "" {
		return nil, errors.New("stripe secret key is required")
	}

	if cfg.Stripe.PublishableKey == "" {
		return nil, errors.New("stripe publishable key is required")
	}

	// Set Stripe API key
	stripe.Key = cfg.Stripe.SecretKey

	// Configure Stripe SDK
	stripe.SetAppInfo(&stripe.AppInfo{
		Name:    "ServicePro",
		Version: "1.0.0",
		URL:     "https://servicepro.com",
	})

	// Set max network retries
	maxRetries := cfg.Stripe.MaxRetries
	if maxRetries <= 0 {
		maxRetries = defaultMaxNetworkRetries
	}
	stripe.SetBackend("api", stripe.GetBackendWithConfig(
		stripe.APIBackend,
		&stripe.BackendConfig{
			MaxNetworkRetries: stripe.Int64(int64(maxRetries)),
		},
	))

	// Determine environment from secret key prefix
	environment := "unknown"
	if len(cfg.Stripe.SecretKey) > 8 {
		if cfg.Stripe.SecretKey[:8] == "sk_test_" {
			environment = "test"
		} else if cfg.Stripe.SecretKey[:8] == "sk_live_" {
			environment = "live"
		}
	}

	client := &Client{
		secretKey:      cfg.Stripe.SecretKey,
		publishableKey: cfg.Stripe.PublishableKey,
		webhookSecret:  cfg.Stripe.WebhookSecret,
		environment:    environment,
		rateLimiter:    newRateLimiter(defaultRequestsPerSecond),
	}

	logging.Info(context.Background(), "Stripe client initialized", map[string]any{"mode": environment})

	return client, nil
}

// GetPublishableKey returns the publishable key (safe for frontend)
func (c *Client) GetPublishableKey() string {
	return c.publishableKey
}

// GetWebhookSecret returns the webhook secret
func (c *Client) GetWebhookSecret() string {
	return c.webhookSecret
}

// IsTestMode returns true if running in test mode
func (c *Client) IsTestMode() bool {
	return c.environment == "test"
}

// PaymentIntentParams represents parameters for creating a payment intent
type PaymentIntentParams struct {
	Amount              int64
	Currency            string
	CustomerID          *string
	PaymentMethodID     *string
	Description         *string
	Metadata            map[string]string
	StatementDescriptor *string
	ReceiptEmail        *string
	SetupFutureUsage    *string
	CaptureMethod       *string
	ConfirmationMethod  *string
	ReturnURL           *string
	IdempotencyKey      *string
}

// PaymentIntentResult represents the result of a payment intent operation
type PaymentIntentResult struct {
	ID                 string
	Amount             int64
	Currency           string
	Status             string
	ClientSecret       string
	CustomerID         string
	PaymentMethodID    string
	ChargeID           string
	Description        string
	Metadata           map[string]string
	CaptureMethod      string
	ConfirmationMethod string
	Created            time.Time
	NextAction         *stripe.PaymentIntentNextAction
	LastPaymentError   *stripe.Error
}

// CreatePaymentIntent creates a new Stripe Payment Intent
func (c *Client) CreatePaymentIntent(ctx context.Context, params *PaymentIntentParams) (*PaymentIntentResult, error) {
	if params == nil {
		return nil, errors.New("params cannot be nil")
	}

	if params.Amount <= 0 {
		return nil, errors.New("amount must be greater than zero")
	}

	if params.Currency == "" {
		params.Currency = "usd"
	}

	// Wait for rate limiter
	if err := c.rateLimiter.Wait(ctx); err != nil {
		return nil, fmt.Errorf("rate limit exceeded: %w", err)
	}

	// Build Stripe params
	stripeParams := &stripe.PaymentIntentParams{
		Amount:   stripe.Int64(params.Amount),
		Currency: stripe.String(params.Currency),
	}

	// Add optional fields
	if params.CustomerID != nil {
		stripeParams.Customer = params.CustomerID
	}

	if params.PaymentMethodID != nil {
		stripeParams.PaymentMethod = params.PaymentMethodID
	}

	if params.Description != nil {
		stripeParams.Description = params.Description
	}

	if params.Metadata != nil {
		for key, value := range params.Metadata {
			stripeParams.AddMetadata(key, value)
		}
	}

	if params.StatementDescriptor != nil {
		stripeParams.StatementDescriptor = params.StatementDescriptor
	}

	if params.ReceiptEmail != nil {
		stripeParams.ReceiptEmail = params.ReceiptEmail
	}

	if params.SetupFutureUsage != nil {
		stripeParams.SetupFutureUsage = params.SetupFutureUsage
	}

	if params.CaptureMethod != nil {
		stripeParams.CaptureMethod = params.CaptureMethod
	}

	if params.ConfirmationMethod != nil {
		stripeParams.ConfirmationMethod = params.ConfirmationMethod
	}

	if params.ReturnURL != nil {
		stripeParams.ReturnURL = params.ReturnURL
	}

	// Add idempotency key
	if params.IdempotencyKey != nil {
		stripeParams.SetIdempotencyKey(*params.IdempotencyKey)
	} else {
		// Generate one if not provided
		stripeParams.SetIdempotencyKey(uuid.New().String())
	}

	// Add context
	stripeParams.Context = ctx

	// Log the request
	logging.Info(ctx, "Creating payment intent", map[string]any{"amount": params.Amount, "currency": params.Currency})

	// Create payment intent
	pi, err := paymentintent.New(stripeParams)
	if err != nil {
		logging.Error(ctx, "Failed to create payment intent", map[string]any{"error": err})
		return nil, fmt.Errorf("failed to create payment intent: %w", err)
	}

	logging.Info(ctx, "Payment intent created", map[string]any{"id": pi.ID, "status": pi.Status})

	return c.paymentIntentToResult(pi), nil
}

// GetPaymentIntent retrieves a payment intent by ID
func (c *Client) GetPaymentIntent(ctx context.Context, id string) (*PaymentIntentResult, error) {
	if id == "" {
		return nil, errors.New("payment intent ID is required")
	}

	if err := c.rateLimiter.Wait(ctx); err != nil {
		return nil, fmt.Errorf("rate limit exceeded: %w", err)
	}

	params := &stripe.PaymentIntentParams{}
	params.Context = ctx

	pi, err := paymentintent.Get(id, params)
	if err != nil {
		logging.Error(ctx, "Failed to get payment intent", map[string]any{"id": id, "error": err})
		return nil, fmt.Errorf("failed to get payment intent: %w", err)
	}

	return c.paymentIntentToResult(pi), nil
}

// ConfirmPaymentIntent confirms a payment intent
func (c *Client) ConfirmPaymentIntent(ctx context.Context, id string, paymentMethodID *string) (*PaymentIntentResult, error) {
	if id == "" {
		return nil, errors.New("payment intent ID is required")
	}

	if err := c.rateLimiter.Wait(ctx); err != nil {
		return nil, fmt.Errorf("rate limit exceeded: %w", err)
	}

	params := &stripe.PaymentIntentConfirmParams{}
	if paymentMethodID != nil {
		params.PaymentMethod = paymentMethodID
	}
	params.Context = ctx

	logging.Info(ctx, "Confirming payment intent", map[string]any{"id": id})

	pi, err := paymentintent.Confirm(id, params)
	if err != nil {
		logging.Error(ctx, "Failed to confirm payment intent", map[string]any{"id": id, "error": err})
		return nil, fmt.Errorf("failed to confirm payment intent: %w", err)
	}

	logging.Info(ctx, "Payment intent confirmed", map[string]any{"id": pi.ID, "status": pi.Status})

	return c.paymentIntentToResult(pi), nil
}

// CancelPaymentIntent cancels a payment intent
func (c *Client) CancelPaymentIntent(ctx context.Context, id string, reason *string) (*PaymentIntentResult, error) {
	if id == "" {
		return nil, errors.New("payment intent ID is required")
	}

	if err := c.rateLimiter.Wait(ctx); err != nil {
		return nil, fmt.Errorf("rate limit exceeded: %w", err)
	}

	params := &stripe.PaymentIntentCancelParams{}
	if reason != nil {
		params.CancellationReason = reason
	}
	params.Context = ctx

	logging.Info(ctx, "Canceling payment intent", map[string]any{"id": id})

	pi, err := paymentintent.Cancel(id, params)
	if err != nil {
		logging.Error(ctx, "Failed to cancel payment intent", map[string]any{"id": id, "error": err})
		return nil, fmt.Errorf("failed to cancel payment intent: %w", err)
	}

	logging.Info(ctx, "Payment intent canceled", map[string]any{"id": pi.ID})

	return c.paymentIntentToResult(pi), nil
}

// CapturePaymentIntent captures a payment intent (for manual capture)
func (c *Client) CapturePaymentIntent(ctx context.Context, id string, amountToCapture *int64) (*PaymentIntentResult, error) {
	if id == "" {
		return nil, errors.New("payment intent ID is required")
	}

	if err := c.rateLimiter.Wait(ctx); err != nil {
		return nil, fmt.Errorf("rate limit exceeded: %w", err)
	}

	params := &stripe.PaymentIntentCaptureParams{}
	if amountToCapture != nil {
		params.AmountToCapture = amountToCapture
	}
	params.Context = ctx

	logging.Info(ctx, "Capturing payment intent", map[string]any{"id": id})

	pi, err := paymentintent.Capture(id, params)
	if err != nil {
		logging.Error(ctx, "Failed to capture payment intent", map[string]any{"id": id, "error": err})
		return nil, fmt.Errorf("failed to capture payment intent: %w", err)
	}

	logging.Info(ctx, "Payment intent captured", map[string]any{"id": pi.ID, "amount": pi.Amount})

	return c.paymentIntentToResult(pi), nil
}

// CustomerParams represents parameters for creating a customer
type CustomerParams struct {
	Email           *string
	Name            *string
	Phone           *string
	Description     *string
	Metadata        map[string]string
	PaymentMethodID *string
	InvoiceSettings *stripe.CustomerInvoiceSettingsParams
	IdempotencyKey  *string
}

// CustomerResult represents a Stripe customer
type CustomerResult struct {
	ID                   string
	Email                string
	Name                 string
	Phone                string
	Description          string
	Metadata             map[string]string
	DefaultPaymentMethod string
	Created              time.Time
}

// CreateCustomer creates a new Stripe customer
func (c *Client) CreateCustomer(ctx context.Context, params *CustomerParams) (*CustomerResult, error) {
	if params == nil {
		return nil, errors.New("params cannot be nil")
	}

	if err := c.rateLimiter.Wait(ctx); err != nil {
		return nil, fmt.Errorf("rate limit exceeded: %w", err)
	}

	stripeParams := &stripe.CustomerParams{}

	if params.Email != nil {
		stripeParams.Email = params.Email
	}

	if params.Name != nil {
		stripeParams.Name = params.Name
	}

	if params.Phone != nil {
		stripeParams.Phone = params.Phone
	}

	if params.Description != nil {
		stripeParams.Description = params.Description
	}

	if params.Metadata != nil {
		for key, value := range params.Metadata {
			stripeParams.AddMetadata(key, value)
		}
	}

	if params.PaymentMethodID != nil {
		stripeParams.PaymentMethod = params.PaymentMethodID
		stripeParams.InvoiceSettings = &stripe.CustomerInvoiceSettingsParams{
			DefaultPaymentMethod: params.PaymentMethodID,
		}
	}

	if params.IdempotencyKey != nil {
		stripeParams.SetIdempotencyKey(*params.IdempotencyKey)
	}

	stripeParams.Context = ctx

	logging.Info(ctx, "Creating Stripe customer", map[string]any{"email": stringPtrValue(params.Email)})

	cust, err := customer.New(stripeParams)
	if err != nil {
		logging.Error(ctx, "Failed to create customer", map[string]any{"error": err})
		return nil, fmt.Errorf("failed to create customer: %w", err)
	}

	logging.Info(ctx, "Customer created", map[string]any{"id": cust.ID})

	return c.customerToResult(cust), nil
}

// GetCustomer retrieves a customer by ID
func (c *Client) GetCustomer(ctx context.Context, id string) (*CustomerResult, error) {
	if id == "" {
		return nil, errors.New("customer ID is required")
	}

	if err := c.rateLimiter.Wait(ctx); err != nil {
		return nil, fmt.Errorf("rate limit exceeded: %w", err)
	}

	params := &stripe.CustomerParams{}
	params.Context = ctx

	cust, err := customer.Get(id, params)
	if err != nil {
		logging.Error(ctx, "Failed to get customer", map[string]any{"id": id, "error": err})
		return nil, fmt.Errorf("failed to get customer: %w", err)
	}

	return c.customerToResult(cust), nil
}

// RefundParams represents parameters for creating a refund
type RefundParams struct {
	ChargeID             *string
	PaymentIntentID      *string
	Amount               *int64
	Reason               *string
	Metadata             map[string]string
	RefundApplicationFee *bool
	ReverseTransfer      *bool
	IdempotencyKey       *string
}

// RefundResult represents a refund
type RefundResult struct {
	ID              string
	Amount          int64
	Currency        string
	ChargeID        string
	PaymentIntentID string
	Status          string
	Reason          string
	Metadata        map[string]string
	Created         time.Time
}

// CreateRefund creates a refund
func (c *Client) CreateRefund(ctx context.Context, params *RefundParams) (*RefundResult, error) {
	if params == nil {
		return nil, errors.New("params cannot be nil")
	}

	if params.ChargeID == nil && params.PaymentIntentID == nil {
		return nil, errors.New("either charge_id or payment_intent_id is required")
	}

	if err := c.rateLimiter.Wait(ctx); err != nil {
		return nil, fmt.Errorf("rate limit exceeded: %w", err)
	}

	stripeParams := &stripe.RefundParams{}

	if params.ChargeID != nil {
		stripeParams.Charge = params.ChargeID
	}

	if params.PaymentIntentID != nil {
		stripeParams.PaymentIntent = params.PaymentIntentID
	}

	if params.Amount != nil {
		stripeParams.Amount = params.Amount
	}

	if params.Reason != nil {
		stripeParams.Reason = params.Reason
	}

	if params.Metadata != nil {
		for key, value := range params.Metadata {
			stripeParams.AddMetadata(key, value)
		}
	}

	if params.RefundApplicationFee != nil {
		stripeParams.RefundApplicationFee = params.RefundApplicationFee
	}

	if params.ReverseTransfer != nil {
		stripeParams.ReverseTransfer = params.ReverseTransfer
	}

	if params.IdempotencyKey != nil {
		stripeParams.SetIdempotencyKey(*params.IdempotencyKey)
	}

	stripeParams.Context = ctx

	logging.Info(ctx, "Creating refund", map[string]any{
		"charge":         stringPtrValue(params.ChargeID),
		"payment_intent": stringPtrValue(params.PaymentIntentID),
		"amount":         params.Amount,
	})

	ref, err := refund.New(stripeParams)
	if err != nil {
		logging.Error(ctx, "Failed to create refund", map[string]any{"error": err})
		return nil, fmt.Errorf("failed to create refund: %w", err)
	}

	logging.Info(ctx, "Refund created", map[string]any{"id": ref.ID, "status": ref.Status})

	return c.refundToResult(ref), nil
}

// Helper functions to convert Stripe objects to our result types

func (c *Client) paymentIntentToResult(pi *stripe.PaymentIntent) *PaymentIntentResult {
	result := &PaymentIntentResult{
		ID:                 pi.ID,
		Amount:             pi.Amount,
		Currency:           string(pi.Currency),
		Status:             string(pi.Status),
		ClientSecret:       pi.ClientSecret,
		Description:        pi.Description,
		Metadata:           pi.Metadata,
		CaptureMethod:      string(pi.CaptureMethod),
		ConfirmationMethod: string(pi.ConfirmationMethod),
		Created:            time.Unix(pi.Created, 0),
		NextAction:         pi.NextAction,
	}

	if pi.Customer != nil {
		result.CustomerID = pi.Customer.ID
	}

	if pi.PaymentMethod != nil {
		result.PaymentMethodID = pi.PaymentMethod.ID
	}

	if pi.LatestCharge != nil {
		result.ChargeID = pi.LatestCharge.ID
	}

	if pi.LastPaymentError != nil {
		result.LastPaymentError = pi.LastPaymentError
	}

	return result
}

func (c *Client) customerToResult(cust *stripe.Customer) *CustomerResult {
	result := &CustomerResult{
		ID:          cust.ID,
		Email:       cust.Email,
		Name:        cust.Name,
		Phone:       cust.Phone,
		Description: cust.Description,
		Metadata:    cust.Metadata,
		Created:     time.Unix(cust.Created, 0),
	}

	if cust.InvoiceSettings != nil && cust.InvoiceSettings.DefaultPaymentMethod != nil {
		result.DefaultPaymentMethod = cust.InvoiceSettings.DefaultPaymentMethod.ID
	}

	return result
}

func (c *Client) refundToResult(ref *stripe.Refund) *RefundResult {
	result := &RefundResult{
		ID:       ref.ID,
		Amount:   ref.Amount,
		Currency: string(ref.Currency),
		Status:   string(ref.Status),
		Reason:   string(ref.Reason),
		Metadata: ref.Metadata,
		Created:  time.Unix(ref.Created, 0),
	}

	if ref.Charge != nil {
		result.ChargeID = ref.Charge.ID
	}

	if ref.PaymentIntent != nil {
		result.PaymentIntentID = ref.PaymentIntent.ID
	}

	return result
}

// Helper functions

func stringPtrValue(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

// ConvertDecimalToCents converts a decimal amount to cents (integer)
func ConvertDecimalToCents(amount decimal.Decimal) int64 {
	return amount.Mul(decimal.NewFromInt(100)).IntPart()
}

// ConvertCentsToDecimal converts cents (integer) to decimal amount
func ConvertCentsToDecimal(cents int64) decimal.Decimal {
	return decimal.NewFromInt(cents).Div(decimal.NewFromInt(100))
}

// Rate limiter implementation

type rateLimiter struct {
	tokens chan struct{}
	ticker *time.Ticker
	done   chan bool
}

func newRateLimiter(requestsPerSecond int) *rateLimiter {
	rl := &rateLimiter{
		tokens: make(chan struct{}, requestsPerSecond),
		ticker: time.NewTicker(time.Second / time.Duration(requestsPerSecond)),
		done:   make(chan bool),
	}

	// Fill initial tokens
	for i := 0; i < requestsPerSecond; i++ {
		rl.tokens <- struct{}{}
	}

	// Refill tokens
	go func() {
		for {
			select {
			case <-rl.ticker.C:
				select {
				case rl.tokens <- struct{}{}:
				default:
					// Channel full, skip
				}
			case <-rl.done:
				return
			}
		}
	}()

	return rl
}

func (rl *rateLimiter) Wait(ctx context.Context) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-rl.tokens:
		return nil
	}
}

func (rl *rateLimiter) Stop() {
	rl.ticker.Stop()
	close(rl.done)
}
