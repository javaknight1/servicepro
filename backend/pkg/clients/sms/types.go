package sms

import (
	"time"
)

// Provider represents an SMS service provider
type Provider string

const (
	ProviderSNS      Provider = "sns"
	ProviderTextBelt Provider = "textbelt"
	ProviderMock     Provider = "mock"
)

// Priority represents SMS priority level
type Priority string

const (
	PriorityLow    Priority = "low"
	PriorityNormal Priority = "normal"
	PriorityHigh   Priority = "high"
	PriorityUrgent Priority = "urgent"
)

// MessageType represents the type of SMS message
type MessageType string

const (
	MessageTypeTransactional MessageType = "transactional"
	MessageTypeMarketing     MessageType = "marketing"
	MessageTypeOTP           MessageType = "otp"
	MessageTypeNotification  MessageType = "notification"
	MessageTypeReminder      MessageType = "reminder"
	MessageTypeAlert         MessageType = "alert"
)

// SMSMessage represents an SMS to be sent
type SMSMessage struct {
	// Recipient phone number (E.164 format recommended: +1234567890)
	PhoneNumber string

	// Message content (plain text)
	Content string

	// Optional fields
	Priority    Priority
	MessageType MessageType

	// Sender ID (where supported by provider/region)
	SenderID string

	// Reference/correlation ID for tracking
	ReferenceID string

	// Metadata for logging/tracking
	Metadata map[string]string

	// Template information (optional)
	TemplateID   string
	TemplateVars map[string]string

	// Scheduling
	ScheduledAt *time.Time
}

// SendResult represents the result of sending an SMS
type SendResult struct {
	// Success status
	Success bool

	// Provider-specific message ID
	MessageID string

	// Provider that was used
	Provider Provider

	// Timestamp when the SMS was sent
	SentAt time.Time

	// Phone number (for batch results)
	PhoneNumber string

	// Error message if failed
	Error string

	// Error code (provider-specific)
	ErrorCode string

	// Number of SMS segments used
	SegmentCount int

	// Cost in provider's currency (if available)
	Cost float64

	// Additional provider-specific info
	Metadata map[string]string
}

// NewSMSMessage creates a new SMSMessage with sensible defaults
func NewSMSMessage(phoneNumber, content string) *SMSMessage {
	return &SMSMessage{
		PhoneNumber: phoneNumber,
		Content:     content,
		Priority:    PriorityNormal,
		MessageType: MessageTypeNotification,
		Metadata:    make(map[string]string),
	}
}

// WithPriority sets the SMS priority
func (m *SMSMessage) WithPriority(priority Priority) *SMSMessage {
	m.Priority = priority
	return m
}

// WithMessageType sets the message type
func (m *SMSMessage) WithMessageType(msgType MessageType) *SMSMessage {
	m.MessageType = msgType
	return m
}

// WithSenderID sets the sender ID
func (m *SMSMessage) WithSenderID(senderID string) *SMSMessage {
	m.SenderID = senderID
	return m
}

// WithReferenceID sets a reference ID for tracking
func (m *SMSMessage) WithReferenceID(refID string) *SMSMessage {
	m.ReferenceID = refID
	return m
}

// WithMetadata adds metadata for logging/tracking
func (m *SMSMessage) WithMetadata(key, value string) *SMSMessage {
	if m.Metadata == nil {
		m.Metadata = make(map[string]string)
	}
	m.Metadata[key] = value
	return m
}

// WithTemplate sets template information
func (m *SMSMessage) WithTemplate(templateID string, vars map[string]string) *SMSMessage {
	m.TemplateID = templateID
	m.TemplateVars = vars
	return m
}

// WithSchedule sets the scheduled send time
func (m *SMSMessage) WithSchedule(scheduledAt time.Time) *SMSMessage {
	m.ScheduledAt = &scheduledAt
	return m
}

// CalculateSegments returns the number of SMS segments based on content length
// Standard SMS: 160 chars (GSM-7) or 70 chars (Unicode)
// Multipart SMS: 153 chars (GSM-7) or 67 chars (Unicode) per segment
func (m *SMSMessage) CalculateSegments() int {
	if m.Content == "" {
		return 0
	}

	// Check if content is pure GSM-7
	isGSM7 := isGSM7Encoded(m.Content)

	length := len([]rune(m.Content))

	if isGSM7 {
		if length <= 160 {
			return 1
		}
		return (length + 152) / 153 // Ceiling division
	}

	// Unicode
	if length <= 70 {
		return 1
	}
	return (length + 66) / 67 // Ceiling division
}

// isGSM7Encoded checks if a string can be encoded with GSM-7
func isGSM7Encoded(s string) bool {
	gsm7Chars := " !\"#$%&'()*+,-./0123456789:;<=>?@ABCDEFGHIJKLMNOPQRSTUVWXYZ[\\]^_`abcdefghijklmnopqrstuvwxyz{|}~\n\r"
	gsm7Extended := "€[]{}~\\|^"

	for _, r := range s {
		found := false
		for _, g := range gsm7Chars {
			if r == g {
				found = true
				break
			}
		}
		if !found {
			for _, g := range gsm7Extended {
				if r == g {
					found = true
					break
				}
			}
		}
		if !found {
			return false
		}
	}
	return true
}
