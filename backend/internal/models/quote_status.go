package models

import (
	"time"

	"github.com/google/uuid"
)

// QuoteStatus represents the possible states of a quote
type QuoteStatus string

const (
	QuoteStatusDraft    QuoteStatus = "draft"
	QuoteStatusSent     QuoteStatus = "sent"
	QuoteStatusViewed   QuoteStatus = "viewed"
	QuoteStatusAccepted QuoteStatus = "accepted"
	QuoteStatusDeclined QuoteStatus = "declined"
	QuoteStatusExpired  QuoteStatus = "expired"
)

// QuoteStatusEvent represents an event type for quote status changes
type QuoteStatusEvent string

const (
	EventQuoteCreated  QuoteStatusEvent = "quote.created"
	EventQuoteSent     QuoteStatusEvent = "quote.sent"
	EventQuoteViewed   QuoteStatusEvent = "quote.viewed"
	EventQuoteAccepted QuoteStatusEvent = "quote.accepted"
	EventQuoteDeclined QuoteStatusEvent = "quote.declined"
	EventQuoteExpired  QuoteStatusEvent = "quote.expired"
	EventQuoteUpdated  QuoteStatusEvent = "quote.updated"
)

// QuoteStatusHistory represents a historical status change
type QuoteStatusHistory struct {
	ID            uuid.UUID              `json:"id" gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	QuoteID       uuid.UUID              `json:"quote_id" gorm:"type:uuid;not null;index"`
	FromStatus    QuoteStatus            `json:"from_status" gorm:"type:varchar(20)"`
	ToStatus      QuoteStatus            `json:"to_status" gorm:"type:varchar(20);not null"`
	Event         QuoteStatusEvent       `json:"event" gorm:"type:varchar(50);not null"`
	ChangedByID   uuid.UUID              `json:"changed_by_id" gorm:"type:uuid"`
	ChangedByType string                 `json:"changed_by_type" gorm:"type:varchar(20)"` // "user", "customer", "system"
	Reason        string                 `json:"reason" gorm:"type:text"`
	Metadata      map[string]interface{} `json:"metadata" gorm:"type:jsonb"`
	CreatedAt     time.Time              `json:"created_at"`
}

// TableName specifies the table name for QuoteStatusHistory
func (QuoteStatusHistory) TableName() string {
	return "quote_status_history"
}

// QuoteStatusTransition represents a status transition request
type QuoteStatusTransition struct {
	QuoteID       uuid.UUID              `json:"quote_id"`
	FromStatus    QuoteStatus            `json:"from_status"`
	ToStatus      QuoteStatus            `json:"to_status"`
	Event         QuoteStatusEvent       `json:"event"`
	ChangedByID   uuid.UUID              `json:"changed_by_id,omitempty"`
	ChangedByType string                 `json:"changed_by_type,omitempty"`
	Reason        string                 `json:"reason,omitempty"`
	Metadata      map[string]interface{} `json:"metadata,omitempty"`
}

// QuoteStatusNotification represents a WebSocket notification
type QuoteStatusNotification struct {
	Type      string           `json:"type"` // "status_change", "quote_update"
	QuoteID   uuid.UUID        `json:"quote_id"`
	Event     QuoteStatusEvent `json:"event"`
	Status    QuoteStatus      `json:"status"`
	Timestamp time.Time        `json:"timestamp"`
	Data      interface{}      `json:"data,omitempty"`
}

// QuoteValidationError represents a validation error during transition
type QuoteValidationError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

func (e QuoteValidationError) Error() string {
	return e.Message
}
