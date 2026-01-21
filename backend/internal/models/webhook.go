package models

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// =============================================================================
// Webhook Event Model
// =============================================================================

// WebhookEventStatus represents the processing status of a webhook event
type WebhookEventStatus string

const (
	WebhookStatusPending    WebhookEventStatus = "pending"
	WebhookStatusProcessing WebhookEventStatus = "processing"
	WebhookStatusProcessed  WebhookEventStatus = "processed"
	WebhookStatusFailed     WebhookEventStatus = "failed"
	WebhookStatusRetrying   WebhookEventStatus = "retrying"
	WebhookStatusSkipped    WebhookEventStatus = "skipped"
)

// WebhookEventSource represents the source of the webhook
type WebhookEventSource string

const (
	WebhookSourceStripe   WebhookEventSource = "stripe"
	WebhookSourceInternal WebhookEventSource = "internal"
)

// WebhookEvent represents a logged webhook event in the database
type WebhookEvent struct {
	ID               uuid.UUID          `gorm:"type:uuid;primary_key" json:"id"`
	Source           WebhookEventSource `gorm:"type:varchar(50);not null;index" json:"source"`
	ExternalEventID  string             `gorm:"type:varchar(255);not null;uniqueIndex:idx_source_event" json:"external_event_id"`
	EventType        string             `gorm:"type:varchar(100);not null;index" json:"event_type"`
	Status           WebhookEventStatus `gorm:"type:varchar(50);not null;default:'pending';index" json:"status"`
	Payload          JSONB              `gorm:"type:jsonb" json:"payload"`
	Headers          JSONB              `gorm:"type:jsonb" json:"headers,omitempty"`
	Response         JSONB              `gorm:"type:jsonb" json:"response,omitempty"`
	ErrorMessage     *string            `gorm:"type:text" json:"error_message,omitempty"`
	ErrorCode        *string            `gorm:"type:varchar(100)" json:"error_code,omitempty"`
	RetryCount       int                `gorm:"default:0" json:"retry_count"`
	MaxRetries       int                `gorm:"default:3" json:"max_retries"`
	NextRetryAt      *time.Time         `gorm:"index" json:"next_retry_at,omitempty"`
	ProcessedAt      *time.Time         `json:"processed_at,omitempty"`
	IPAddress        string             `gorm:"type:varchar(45)" json:"ip_address,omitempty"`
	UserAgent        string             `gorm:"type:varchar(500)" json:"user_agent,omitempty"`
	ProcessingTimeMs *int64             `json:"processing_time_ms,omitempty"`
	Metadata         JSONB              `gorm:"type:jsonb" json:"metadata,omitempty"`
	CreatedAt        time.Time          `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt        time.Time          `gorm:"autoUpdateTime" json:"updated_at"`
}

// TableName specifies the table name for WebhookEvent
func (WebhookEvent) TableName() string {
	return "webhook_events"
}

// BeforeCreate sets the UUID before creating a new record
func (w *WebhookEvent) BeforeCreate(tx *gorm.DB) error {
	if w.ID == uuid.Nil {
		w.ID = uuid.New()
	}
	return nil
}

// MarkAsProcessing updates the status to processing
func (w *WebhookEvent) MarkAsProcessing() {
	w.Status = WebhookStatusProcessing
}

// MarkAsProcessed updates the status to processed
func (w *WebhookEvent) MarkAsProcessed(response interface{}, processingTimeMs int64) {
	w.Status = WebhookStatusProcessed
	now := time.Now()
	w.ProcessedAt = &now
	w.ProcessingTimeMs = &processingTimeMs

	if response != nil {
		if data, err := json.Marshal(response); err == nil {
			w.Response = data
		}
	}
}

// MarkAsFailed updates the status to failed with error details
func (w *WebhookEvent) MarkAsFailed(errMsg, errCode string) {
	w.Status = WebhookStatusFailed
	w.ErrorMessage = &errMsg
	if errCode != "" {
		w.ErrorCode = &errCode
	}
}

// MarkForRetry updates the status to retrying and schedules next retry
func (w *WebhookEvent) MarkForRetry(nextRetryAt time.Time, errMsg string) {
	w.Status = WebhookStatusRetrying
	w.RetryCount++
	w.NextRetryAt = &nextRetryAt
	w.ErrorMessage = &errMsg
}

// CanRetry checks if the event can be retried
func (w *WebhookEvent) CanRetry() bool {
	return w.RetryCount < w.MaxRetries
}

// IsTerminal checks if the event is in a terminal state
func (w *WebhookEvent) IsTerminal() bool {
	return w.Status == WebhookStatusProcessed ||
		w.Status == WebhookStatusFailed ||
		w.Status == WebhookStatusSkipped
}

// =============================================================================
// JSONB type for PostgreSQL
// =============================================================================

// JSONB is a custom type for PostgreSQL JSONB columns
type JSONB json.RawMessage

// Value implements the driver.Valuer interface
func (j JSONB) Value() (driver.Value, error) {
	if len(j) == 0 {
		return nil, nil
	}
	return []byte(j), nil
}

// Scan implements the sql.Scanner interface
func (j *JSONB) Scan(value interface{}) error {
	if value == nil {
		*j = nil
		return nil
	}

	switch v := value.(type) {
	case []byte:
		*j = make([]byte, len(v))
		copy(*j, v)
	case string:
		*j = []byte(v)
	default:
		return errors.New("type assertion to []byte or string failed")
	}

	return nil
}

// MarshalJSON returns the JSON encoding
func (j JSONB) MarshalJSON() ([]byte, error) {
	if len(j) == 0 {
		return []byte("null"), nil
	}
	return j, nil
}

// UnmarshalJSON sets the value from JSON
func (j *JSONB) UnmarshalJSON(data []byte) error {
	if j == nil {
		return errors.New("JSONB: UnmarshalJSON on nil pointer")
	}
	*j = append((*j)[0:0], data...)
	return nil
}

// String returns the string representation
func (j JSONB) String() string {
	return string(j)
}

// =============================================================================
// Webhook Delivery Log Model (for tracking delivery attempts)
// =============================================================================

// WebhookDeliveryLog tracks individual delivery attempts for retry scenarios
type WebhookDeliveryLog struct {
	ID              uuid.UUID `gorm:"type:uuid;primary_key" json:"id"`
	WebhookEventID  uuid.UUID `gorm:"type:uuid;not null;index" json:"webhook_event_id"`
	AttemptNumber   int       `gorm:"not null" json:"attempt_number"`
	StatusCode      int       `json:"status_code,omitempty"`
	RequestHeaders  JSONB     `gorm:"type:jsonb" json:"request_headers,omitempty"`
	ResponseHeaders JSONB     `gorm:"type:jsonb" json:"response_headers,omitempty"`
	ResponseBody    *string   `gorm:"type:text" json:"response_body,omitempty"`
	ErrorMessage    *string   `gorm:"type:text" json:"error_message,omitempty"`
	DurationMs      int64     `json:"duration_ms"`
	CreatedAt       time.Time `gorm:"autoCreateTime" json:"created_at"`
}

// TableName specifies the table name for WebhookDeliveryLog
func (WebhookDeliveryLog) TableName() string {
	return "webhook_delivery_logs"
}

// BeforeCreate sets the UUID before creating a new record
func (w *WebhookDeliveryLog) BeforeCreate(tx *gorm.DB) error {
	if w.ID == uuid.Nil {
		w.ID = uuid.New()
	}
	return nil
}

// =============================================================================
// Payment Record Model (for tracking payment updates from webhooks)
// =============================================================================

// Additional PaymentStatus values for webhook processing
// Note: Base PaymentStatus type is defined in payment.go
const (
	PaymentStatusRequiresAction    PaymentStatus = "requires_action"
	PaymentStatusRequiresCapture   PaymentStatus = "requires_capture"
	PaymentStatusPartiallyRefunded PaymentStatus = "partially_refunded"
	PaymentStatusDisputed          PaymentStatus = "disputed"
)

// PaymentRecord represents a payment record linked to Stripe
type PaymentRecord struct {
	ID                 uuid.UUID     `gorm:"type:uuid;primary_key" json:"id"`
	CustomerID         uuid.UUID     `gorm:"type:uuid;index" json:"customer_id"`
	StripePaymentID    string        `gorm:"type:varchar(255);uniqueIndex" json:"stripe_payment_id"`
	StripeCustomerID   *string       `gorm:"type:varchar(255);index" json:"stripe_customer_id,omitempty"`
	StripeInvoiceID    *string       `gorm:"type:varchar(255);index" json:"stripe_invoice_id,omitempty"`
	Amount             int64         `gorm:"not null" json:"amount"`
	Currency           string        `gorm:"type:varchar(3);not null" json:"currency"`
	Status             PaymentStatus `gorm:"type:varchar(50);not null;index" json:"status"`
	Description        *string       `gorm:"type:text" json:"description,omitempty"`
	FailureCode        *string       `gorm:"type:varchar(100)" json:"failure_code,omitempty"`
	FailureMessage     *string       `gorm:"type:text" json:"failure_message,omitempty"`
	RefundedAmount     int64         `gorm:"default:0" json:"refunded_amount"`
	Metadata           JSONB         `gorm:"type:jsonb" json:"metadata,omitempty"`
	LastWebhookEventID *uuid.UUID    `gorm:"type:uuid" json:"last_webhook_event_id,omitempty"`
	PaidAt             *time.Time    `json:"paid_at,omitempty"`
	CreatedAt          time.Time     `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt          time.Time     `gorm:"autoUpdateTime" json:"updated_at"`
}

// TableName specifies the table name for PaymentRecord
func (PaymentRecord) TableName() string {
	return "payment_records"
}

// BeforeCreate sets the UUID before creating a new record
func (p *PaymentRecord) BeforeCreate(tx *gorm.DB) error {
	if p.ID == uuid.Nil {
		p.ID = uuid.New()
	}
	return nil
}

// =============================================================================
// Subscription Status Model
// =============================================================================

// SubscriptionStatus represents a subscription status
type SubscriptionStatus string

const (
	SubscriptionStatusActive            SubscriptionStatus = "active"
	SubscriptionStatusPastDue           SubscriptionStatus = "past_due"
	SubscriptionStatusCanceled          SubscriptionStatus = "canceled"
	SubscriptionStatusUnpaid            SubscriptionStatus = "unpaid"
	SubscriptionStatusIncomplete        SubscriptionStatus = "incomplete"
	SubscriptionStatusIncompleteExpired SubscriptionStatus = "incomplete_expired"
	SubscriptionStatusTrialing          SubscriptionStatus = "trialing"
	SubscriptionStatusPaused            SubscriptionStatus = "paused"
)

// SubscriptionRecord represents a subscription linked to Stripe
type SubscriptionRecord struct {
	ID                   uuid.UUID          `gorm:"type:uuid;primary_key" json:"id"`
	CustomerID           uuid.UUID          `gorm:"type:uuid;index" json:"customer_id"`
	StripeSubscriptionID string             `gorm:"type:varchar(255);uniqueIndex" json:"stripe_subscription_id"`
	StripeCustomerID     string             `gorm:"type:varchar(255);index" json:"stripe_customer_id"`
	StripePriceID        string             `gorm:"type:varchar(255)" json:"stripe_price_id"`
	Status               SubscriptionStatus `gorm:"type:varchar(50);not null;index" json:"status"`
	CurrentPeriodStart   time.Time          `json:"current_period_start"`
	CurrentPeriodEnd     time.Time          `json:"current_period_end"`
	CancelAtPeriodEnd    bool               `gorm:"default:false" json:"cancel_at_period_end"`
	CanceledAt           *time.Time         `json:"canceled_at,omitempty"`
	EndedAt              *time.Time         `json:"ended_at,omitempty"`
	TrialStart           *time.Time         `json:"trial_start,omitempty"`
	TrialEnd             *time.Time         `json:"trial_end,omitempty"`
	Metadata             JSONB              `gorm:"type:jsonb" json:"metadata,omitempty"`
	LastWebhookEventID   *uuid.UUID         `gorm:"type:uuid" json:"last_webhook_event_id,omitempty"`
	CreatedAt            time.Time          `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt            time.Time          `gorm:"autoUpdateTime" json:"updated_at"`
}

// TableName specifies the table name for SubscriptionRecord
func (SubscriptionRecord) TableName() string {
	return "subscription_records"
}

// BeforeCreate sets the UUID before creating a new record
func (s *SubscriptionRecord) BeforeCreate(tx *gorm.DB) error {
	if s.ID == uuid.Nil {
		s.ID = uuid.New()
	}
	return nil
}

// IsActive returns true if subscription is in an active state
func (s *SubscriptionRecord) IsActive() bool {
	return s.Status == SubscriptionStatusActive || s.Status == SubscriptionStatusTrialing
}

// =============================================================================
// Query Options
// =============================================================================

// WebhookEventQueryOptions contains options for querying webhook events
type WebhookEventQueryOptions struct {
	Source    WebhookEventSource
	EventType string
	Status    WebhookEventStatus
	StartDate *time.Time
	EndDate   *time.Time
	Limit     int
	Offset    int
}

// WebhookEventStats contains statistics about webhook events
type WebhookEventStats struct {
	TotalEvents     int64            `json:"total_events"`
	ProcessedEvents int64            `json:"processed_events"`
	FailedEvents    int64            `json:"failed_events"`
	PendingEvents   int64            `json:"pending_events"`
	RetryingEvents  int64            `json:"retrying_events"`
	EventsByType    map[string]int64 `json:"events_by_type"`
	AvgProcessingMs float64          `json:"avg_processing_ms"`
}
