package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// PaymentReminder represents a payment reminder sent for an overdue invoice
type PaymentReminder struct {
	ID             uuid.UUID      `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	TenantID       uuid.UUID      `gorm:"type:uuid;not null" json:"tenant_id"`
	InvoiceID      uuid.UUID      `gorm:"type:uuid;not null" json:"invoice_id"`
	CustomerID     uuid.UUID      `gorm:"type:uuid;not null" json:"customer_id"`
	ReminderNumber int            `gorm:"not null;default:1" json:"reminder_number"`
	Channel        string         `gorm:"type:notification_channel;not null;default:'email'" json:"channel"`
	Status         string         `gorm:"type:notification_status;not null;default:'pending'" json:"status"`
	Tone           string         `gorm:"type:varchar(20);not null;default:'friendly'" json:"tone"`
	Recipient      string         `gorm:"type:varchar(255);not null" json:"recipient"`
	Subject        string         `gorm:"type:varchar(500)" json:"subject"`
	ErrorMessage   *string        `gorm:"type:text" json:"error_message,omitempty"`
	SentAt         *int64         `gorm:"type:bigint" json:"sent_at,omitempty"`
	CreatedAt      time.Time      `json:"created_at"`
	UpdatedAt      time.Time      `json:"updated_at"`
	DeletedAt      gorm.DeletedAt `json:"deleted_at,omitempty"`

	// Relationships
	Invoice  *Invoice  `gorm:"foreignKey:InvoiceID" json:"invoice,omitempty"`
	Customer *Customer `gorm:"foreignKey:CustomerID" json:"customer,omitempty"`
}

// TableName specifies the table name for PaymentReminder
func (PaymentReminder) TableName() string {
	return "payment_reminders"
}

// PaymentReminderResponse represents the API response for a payment reminder
type PaymentReminderResponse struct {
	ID             uuid.UUID `json:"id"`
	InvoiceID      uuid.UUID `json:"invoice_id"`
	InvoiceNumber  string    `json:"invoice_number"`
	CustomerName   string    `json:"customer_name"`
	ReminderNumber int       `json:"reminder_number"`
	Channel        string    `json:"channel"`
	Status         string    `json:"status"`
	Tone           string    `json:"tone"`
	Recipient      string    `json:"recipient"`
	SentAt         *int64    `json:"sent_at,omitempty"`
	ErrorMessage   *string   `json:"error_message,omitempty"`
	CreatedAt      time.Time `json:"created_at"`
}

// ToResponse converts a PaymentReminder to PaymentReminderResponse
func (r *PaymentReminder) ToResponse() PaymentReminderResponse {
	resp := PaymentReminderResponse{
		ID:             r.ID,
		InvoiceID:      r.InvoiceID,
		ReminderNumber: r.ReminderNumber,
		Channel:        r.Channel,
		Status:         r.Status,
		Tone:           r.Tone,
		Recipient:      r.Recipient,
		SentAt:         r.SentAt,
		ErrorMessage:   r.ErrorMessage,
		CreatedAt:      r.CreatedAt,
	}
	if r.Invoice != nil {
		resp.InvoiceNumber = r.Invoice.InvoiceNumber
	}
	if r.Customer != nil {
		resp.CustomerName = r.Customer.GetDisplayName()
	}
	return resp
}

// PaymentReminderSettingsRequest represents the request to update reminder settings
type PaymentReminderSettingsRequest struct {
	PaymentRemindersEnabled bool  `json:"paymentRemindersEnabled"`
	ReminderDaysAfterDue    []int `json:"reminderDaysAfterDue"`
	MaxRemindersPerInvoice  int   `json:"maxRemindersPerInvoice"`
}

// PaymentReminderSettingsResponse represents the response for reminder settings
type PaymentReminderSettingsResponse struct {
	PaymentRemindersEnabled bool  `json:"paymentRemindersEnabled"`
	ReminderDaysAfterDue    []int `json:"reminderDaysAfterDue"`
	MaxRemindersPerInvoice  int   `json:"maxRemindersPerInvoice"`
}
