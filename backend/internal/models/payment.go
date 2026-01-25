package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

// PaymentStatus represents the status of a payment
type PaymentStatus string

const (
	PaymentStatusPending    PaymentStatus = "pending"
	PaymentStatusProcessing PaymentStatus = "processing"
	PaymentStatusSucceeded  PaymentStatus = "succeeded"
	PaymentStatusFailed     PaymentStatus = "failed"
	PaymentStatusCanceled   PaymentStatus = "canceled"
	PaymentStatusRefunded   PaymentStatus = "refunded"
)

// PaymentMethod represents the payment method type
type PaymentMethodType string

const (
	PaymentMethodCard         PaymentMethodType = "card"
	PaymentMethodBankTransfer PaymentMethodType = "bank_transfer"
	PaymentMethodACH          PaymentMethodType = "ach"
	PaymentMethodWallet       PaymentMethodType = "wallet"
)

// Payment represents a payment record
type Payment struct {
	ID        uuid.UUID  `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	UserID    uuid.UUID  `gorm:"type:uuid;not null;index" json:"user_id"`
	InvoiceID *uuid.UUID `gorm:"type:uuid;index" json:"invoice_id,omitempty"`
	OrderID   *uuid.UUID `gorm:"type:uuid;index" json:"order_id,omitempty"`

	// Stripe IDs
	StripePaymentIntentID string `gorm:"type:varchar(255);unique;not null;index" json:"stripe_payment_intent_id"`
	StripeCustomerID      string `gorm:"type:varchar(255);index" json:"stripe_customer_id,omitempty"`
	StripeChargeID        string `gorm:"type:varchar(255);index" json:"stripe_charge_id,omitempty"`

	// Payment Details
	Amount            decimal.Decimal   `gorm:"type:decimal(12,2);not null" json:"amount"`
	Currency          string            `gorm:"type:varchar(3);not null;default:'usd'" json:"currency"`
	Status            PaymentStatus     `gorm:"type:varchar(20);not null;default:'pending';index" json:"status"`
	PaymentMethodType PaymentMethodType `gorm:"type:varchar(50)" json:"payment_method_type,omitempty"`

	// Descriptions
	Description         string `gorm:"type:text" json:"description,omitempty"`
	StatementDescriptor string `gorm:"type:varchar(100)" json:"statement_descriptor,omitempty"`

	// Metadata
	Metadata map[string]string `gorm:"type:jsonb" json:"metadata,omitempty"`

	// Error information
	ErrorCode    string `gorm:"type:varchar(100)" json:"error_code,omitempty"`
	ErrorMessage string `gorm:"type:text" json:"error_message,omitempty"`

	// Receipt
	ReceiptEmail string `gorm:"type:varchar(255)" json:"receipt_email,omitempty"`
	ReceiptURL   string `gorm:"type:text" json:"receipt_url,omitempty"`

	// Timestamps
	ProcessedAt *time.Time `json:"processed_at,omitempty"`
	CreatedAt   time.Time  `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt   time.Time  `gorm:"autoUpdateTime" json:"updated_at"`

	// Relationships
	User    *User           `gorm:"foreignKey:UserID" json:"user,omitempty"`
	Refunds []PaymentRefund `gorm:"foreignKey:PaymentID" json:"refunds,omitempty"`
}

// TableName overrides the table name
func (Payment) TableName() string {
	return "payments"
}

// BeforeCreate hook
func (p *Payment) BeforeCreate(tx *gorm.DB) error {
	if p.ID == uuid.Nil {
		p.ID = uuid.New()
	}
	return nil
}

// PaymentRefund represents a refund for a payment
type PaymentRefund struct {
	ID        uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	PaymentID uuid.UUID `gorm:"type:uuid;not null;index" json:"payment_id"`

	// Stripe ID
	StripeRefundID string `gorm:"type:varchar(255);unique;not null;index" json:"stripe_refund_id"`

	// Refund Details
	Amount   decimal.Decimal `gorm:"type:decimal(12,2);not null" json:"amount"`
	Currency string          `gorm:"type:varchar(3);not null" json:"currency"`
	Status   string          `gorm:"type:varchar(20);not null" json:"status"`
	Reason   string          `gorm:"type:varchar(100)" json:"reason,omitempty"`

	// Metadata
	Metadata map[string]string `gorm:"type:jsonb" json:"metadata,omitempty"`

	// Timestamps
	ProcessedAt *time.Time `json:"processed_at,omitempty"`
	CreatedAt   time.Time  `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt   time.Time  `gorm:"autoUpdateTime" json:"updated_at"`

	// Relationships
	Payment *Payment `gorm:"foreignKey:PaymentID" json:"payment,omitempty"`
}

// TableName overrides the table name
func (PaymentRefund) TableName() string {
	return "payment_refunds"
}

// BeforeCreate hook
func (pr *PaymentRefund) BeforeCreate(tx *gorm.DB) error {
	if pr.ID == uuid.Nil {
		pr.ID = uuid.New()
	}
	return nil
}

// CreatePaymentIntentRequest represents request to create a payment intent
type CreatePaymentIntentRequest struct {
	Amount              float64           `json:"amount" binding:"required,gt=0"`
	Currency            string            `json:"currency" binding:"required,len=3"`
	InvoiceID           *uuid.UUID        `json:"invoice_id"`
	OrderID             *uuid.UUID        `json:"order_id"`
	PaymentMethodID     *string           `json:"payment_method_id"`
	CustomerID          *string           `json:"customer_id"`
	Description         string            `json:"description"`
	StatementDescriptor string            `json:"statement_descriptor"`
	ReceiptEmail        string            `json:"receipt_email"`
	Metadata            map[string]string `json:"metadata"`
	CaptureMethod       string            `json:"capture_method"` // automatic or manual
	SavePaymentMethod   bool              `json:"save_payment_method"`
}

// CreatePaymentIntentResponse represents response after creating payment intent
type CreatePaymentIntentResponse struct {
	PaymentID      uuid.UUID   `json:"payment_id"`
	ClientSecret   string      `json:"client_secret"`
	Status         string      `json:"status"`
	Amount         float64     `json:"amount"`
	Currency       string      `json:"currency"`
	RequiresAction bool        `json:"requires_action"`
	NextAction     interface{} `json:"next_action,omitempty"`
}

// ConfirmPaymentRequest represents request to confirm a payment
type ConfirmPaymentRequest struct {
	PaymentID       uuid.UUID `json:"payment_id" binding:"required"`
	PaymentMethodID *string   `json:"payment_method_id"`
	ReturnURL       *string   `json:"return_url"`
}

// ConfirmPaymentResponse represents response after confirming payment
type ConfirmPaymentResponse struct {
	PaymentID      uuid.UUID   `json:"payment_id"`
	Status         string      `json:"status"`
	Amount         float64     `json:"amount"`
	Currency       string      `json:"currency"`
	RequiresAction bool        `json:"requires_action"`
	NextAction     interface{} `json:"next_action,omitempty"`
	ReceiptURL     string      `json:"receipt_url,omitempty"`
}

// CreateRefundRequest represents request to create a refund
type CreateRefundRequest struct {
	PaymentID uuid.UUID         `json:"payment_id" binding:"required"`
	Amount    *float64          `json:"amount"` // Partial refund if specified
	Reason    string            `json:"reason"`
	Metadata  map[string]string `json:"metadata"`
}

// CreateRefundResponse represents response after creating refund
type CreateRefundResponse struct {
	RefundID  uuid.UUID       `json:"refund_id"`
	PaymentID uuid.UUID       `json:"payment_id"`
	Amount    decimal.Decimal `json:"amount"`
	Currency  string          `json:"currency"`
	Status    string          `json:"status"`
	Reason    string          `json:"reason,omitempty"`
	CreatedAt time.Time       `json:"created_at"`
}

// PaymentMethodResponse represents a saved payment method
type PaymentMethodResponse struct {
	ID             string          `json:"id"`
	Type           string          `json:"type"`
	Card           *CardDetails    `json:"card,omitempty"`
	BillingDetails *BillingDetails `json:"billing_details,omitempty"`
	Created        time.Time       `json:"created"`
}

// CardDetails represents card information
type CardDetails struct {
	Brand    string `json:"brand"`
	Last4    string `json:"last4"`
	ExpMonth int    `json:"exp_month"`
	ExpYear  int    `json:"exp_year"`
	Country  string `json:"country,omitempty"`
}

// BillingDetails represents billing information
type BillingDetails struct {
	Name    string   `json:"name,omitempty"`
	Email   string   `json:"email,omitempty"`
	Phone   string   `json:"phone,omitempty"`
	Address *Address `json:"address,omitempty"`
}

// Address represents an address
type Address struct {
	Line1      string `json:"line1,omitempty"`
	Line2      string `json:"line2,omitempty"`
	City       string `json:"city,omitempty"`
	State      string `json:"state,omitempty"`
	PostalCode string `json:"postal_code,omitempty"`
	Country    string `json:"country,omitempty"`
}

// PaymentReceiptResponse represents a payment receipt
type PaymentReceiptResponse struct {
	PaymentID         uuid.UUID       `json:"payment_id"`
	Amount            decimal.Decimal `json:"amount"`
	Currency          string          `json:"currency"`
	Status            string          `json:"status"`
	PaymentMethodType string          `json:"payment_method_type,omitempty"`
	Description       string          `json:"description,omitempty"`
	ReceiptURL        string          `json:"receipt_url,omitempty"`
	ReceiptEmail      string          `json:"receipt_email,omitempty"`
	ProcessedAt       *time.Time      `json:"processed_at,omitempty"`
	CreatedAt         time.Time       `json:"created_at"`
	Card              *CardDetails    `json:"card,omitempty"`
	RefundedAmount    decimal.Decimal `json:"refunded_amount,omitempty"`
	NetAmount         decimal.Decimal `json:"net_amount"`
}

// PaymentListFilter represents filters for listing payments
type PaymentListFilter struct {
	UserID    *uuid.UUID
	Status    *PaymentStatus
	StartDate *time.Time
	EndDate   *time.Time
	MinAmount *decimal.Decimal
	MaxAmount *decimal.Decimal
	Page      int
	PageSize  int
}

// PaymentSummary represents payment summary statistics
type PaymentSummary struct {
	TotalPayments   int64           `json:"total_payments"`
	SuccessfulCount int64           `json:"successful_count"`
	FailedCount     int64           `json:"failed_count"`
	RefundedCount   int64           `json:"refunded_count"`
	TotalAmount     decimal.Decimal `json:"total_amount"`
	RefundedAmount  decimal.Decimal `json:"refunded_amount"`
	NetAmount       decimal.Decimal `json:"net_amount"`
}

// ============================================================================
// Tenant-Level Payment Models (for Subscriptions)
// ============================================================================

// BillingEventStatus represents the status of a billing event
type BillingEventStatus string

const (
	BillingEventStatusSucceeded BillingEventStatus = "succeeded"
	BillingEventStatusFailed    BillingEventStatus = "failed"
	BillingEventStatusPending   BillingEventStatus = "pending"
)

// SavedPaymentMethod represents a saved payment method for a tenant
type SavedPaymentMethod struct {
	ID                    uuid.UUID `json:"id" gorm:"type:uuid;primary_key;default:uuid_generate_v4()"`
	TenantID              uuid.UUID `json:"tenant_id" gorm:"type:uuid;not null;index"`
	StripePaymentMethodID string    `json:"stripe_payment_method_id" gorm:"uniqueIndex;not null;size:255"`
	CardBrand             string    `json:"card_brand" gorm:"not null;size:50"`
	CardLastFour          string    `json:"card_last_four" gorm:"not null;size:4"`
	CardExpMonth          int       `json:"card_exp_month" gorm:"not null"`
	CardExpYear           int       `json:"card_exp_year" gorm:"not null"`
	IsDefault             bool      `json:"is_default" gorm:"not null;default:false"`
	CreatedAt             time.Time `json:"created_at"`
	UpdatedAt             time.Time `json:"updated_at"`

	// Relations
	Tenant *Tenant `json:"tenant,omitempty" gorm:"foreignKey:TenantID"`
}

// TableName specifies the table name for SavedPaymentMethod
func (SavedPaymentMethod) TableName() string {
	return "payment_methods"
}

// BeforeCreate hook
func (pm *SavedPaymentMethod) BeforeCreate(tx *gorm.DB) error {
	if pm.ID == uuid.Nil {
		pm.ID = uuid.New()
	}
	return nil
}

// ToSavedPaymentMethodResponse converts a SavedPaymentMethod to response format
func (pm *SavedPaymentMethod) ToSavedPaymentMethodResponse() SavedPaymentMethodResponse {
	return SavedPaymentMethodResponse{
		ID:           pm.ID,
		CardBrand:    pm.CardBrand,
		CardLastFour: pm.CardLastFour,
		CardExpMonth: pm.CardExpMonth,
		CardExpYear:  pm.CardExpYear,
		IsDefault:    pm.IsDefault,
		CreatedAt:    pm.CreatedAt,
	}
}

// BillingEvent represents a billing event (payment, invoice, etc.)
type BillingEvent struct {
	ID               uuid.UUID          `json:"id" gorm:"type:uuid;primary_key;default:uuid_generate_v4()"`
	TenantID         uuid.UUID          `json:"tenant_id" gorm:"type:uuid;not null;index"`
	StripeEventID    string             `json:"stripe_event_id" gorm:"uniqueIndex;not null;size:255"`
	EventType        string             `json:"event_type" gorm:"not null;size:100;index"`
	AmountCents      *int               `json:"amount_cents,omitempty"`
	Currency         string             `json:"currency" gorm:"size:3;default:'usd'"`
	Status           BillingEventStatus `json:"status" gorm:"not null;size:50"`
	Description      string             `json:"description,omitempty" gorm:"type:text"`
	StripeInvoiceID  *string            `json:"stripe_invoice_id,omitempty" gorm:"size:255"`
	StripeInvoiceURL *string            `json:"stripe_invoice_url,omitempty" gorm:"type:text"`
	Metadata         map[string]any     `json:"metadata,omitempty" gorm:"type:jsonb"`
	CreatedAt        time.Time          `json:"created_at" gorm:"index"`

	// Relations
	Tenant *Tenant `json:"tenant,omitempty" gorm:"foreignKey:TenantID"`
}

// TableName specifies the table name for BillingEvent
func (BillingEvent) TableName() string {
	return "billing_events"
}

// BeforeCreate hook
func (be *BillingEvent) BeforeCreate(tx *gorm.DB) error {
	if be.ID == uuid.Nil {
		be.ID = uuid.New()
	}
	return nil
}

// ToBillingEventResponse converts a BillingEvent to response format
func (be *BillingEvent) ToBillingEventResponse() BillingEventResponse {
	return BillingEventResponse{
		ID:               be.ID,
		EventType:        be.EventType,
		AmountCents:      be.AmountCents,
		Currency:         be.Currency,
		Status:           be.Status,
		Description:      be.Description,
		StripeInvoiceURL: be.StripeInvoiceURL,
		CreatedAt:        be.CreatedAt,
	}
}

// ============================================================================
// Tenant-Level Payment Request/Response Types
// ============================================================================

// SavedPaymentMethodResponse represents the API response for a saved payment method
type SavedPaymentMethodResponse struct {
	ID           uuid.UUID `json:"id"`
	CardBrand    string    `json:"card_brand"`
	CardLastFour string    `json:"card_last_four"`
	CardExpMonth int       `json:"card_exp_month"`
	CardExpYear  int       `json:"card_exp_year"`
	IsDefault    bool      `json:"is_default"`
	CreatedAt    time.Time `json:"created_at"`
}

// ListSavedPaymentMethodsResponse represents the response for listing saved payment methods
type ListSavedPaymentMethodsResponse struct {
	PaymentMethods []SavedPaymentMethodResponse `json:"payment_methods"`
}

// AddSavedPaymentMethodRequest represents the request to add a saved payment method
type AddSavedPaymentMethodRequest struct {
	PaymentMethodID string `json:"payment_method_id" binding:"required"`
	SetAsDefault    bool   `json:"set_as_default"`
}

// SetDefaultSavedPaymentMethodRequest represents the request to set a default payment method
type SetDefaultSavedPaymentMethodRequest struct {
	PaymentMethodID uuid.UUID `json:"payment_method_id" binding:"required"`
}

// BillingEventResponse represents the API response for a billing event
type BillingEventResponse struct {
	ID               uuid.UUID          `json:"id"`
	EventType        string             `json:"event_type"`
	AmountCents      *int               `json:"amount_cents,omitempty"`
	Currency         string             `json:"currency"`
	Status           BillingEventStatus `json:"status"`
	Description      string             `json:"description,omitempty"`
	StripeInvoiceURL *string            `json:"stripe_invoice_url,omitempty"`
	CreatedAt        time.Time          `json:"created_at"`
}

// ListBillingEventsResponse represents the response for listing billing events
type ListBillingEventsResponse struct {
	Events     []BillingEventResponse `json:"events"`
	Total      int64                  `json:"total"`
	Page       int                    `json:"page"`
	PageSize   int                    `json:"page_size"`
	TotalPages int                    `json:"total_pages"`
}

// CreateSetupIntentResponse represents the response for creating a setup intent
type CreateSetupIntentResponse struct {
	ClientSecret string `json:"client_secret"`
}

// CreateCheckoutSessionRequest represents the request to create a checkout session
type CreateCheckoutSessionRequest struct {
	TierID       uuid.UUID    `json:"tier_id" binding:"required"`
	BillingCycle BillingCycle `json:"billing_cycle" binding:"required,oneof=monthly annual"`
	SuccessURL   string       `json:"success_url" binding:"required,url"`
	CancelURL    string       `json:"cancel_url" binding:"required,url"`
}

// CreateCheckoutSessionResponse represents the response for creating a checkout session
type CreateCheckoutSessionResponse struct {
	SessionID string `json:"session_id"`
	URL       string `json:"url"`
}
