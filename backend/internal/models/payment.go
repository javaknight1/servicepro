package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

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
