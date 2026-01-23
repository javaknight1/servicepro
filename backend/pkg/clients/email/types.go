package email

import (
	"time"
)

// Provider represents an email service provider
type Provider string

const (
	ProviderSES    Provider = "ses"
	ProviderResend Provider = "resend"
	ProviderMock   Provider = "mock"
)

// Priority represents email priority level
type Priority string

const (
	PriorityLow    Priority = "low"
	PriorityNormal Priority = "normal"
	PriorityHigh   Priority = "high"
)

// EmailMessage represents an email to be sent
type EmailMessage struct {
	// Recipients
	To  []string
	CC  []string
	BCC []string

	// Sender
	From     string
	FromName string
	ReplyTo  string

	// Content
	Subject  string
	HTMLBody string
	TextBody string

	// Optional fields
	Priority    Priority
	Tags        map[string]string
	Headers     map[string]string
	Attachments []Attachment

	// Tracking
	TrackOpens  bool
	TrackClicks bool

	// Metadata for logging/tracking
	Metadata map[string]string
}

// Attachment represents an email attachment
type Attachment struct {
	Filename    string
	ContentType string
	Content     []byte
	ContentID   string // For inline attachments
	Inline      bool
}

// SendResult represents the result of sending an email
type SendResult struct {
	// Success status
	Success bool

	// Provider-specific message ID
	MessageID string

	// Provider that was used
	Provider Provider

	// Timestamp when the email was sent
	SentAt time.Time

	// Error message if failed
	Error string

	// Additional provider-specific info
	Metadata map[string]string
}

// NewEmailMessage creates a new EmailMessage with sensible defaults
func NewEmailMessage(to, subject, htmlBody string) *EmailMessage {
	return &EmailMessage{
		To:       []string{to},
		Subject:  subject,
		HTMLBody: htmlBody,
		Priority: PriorityNormal,
		Tags:     make(map[string]string),
		Headers:  make(map[string]string),
		Metadata: make(map[string]string),
	}
}

// WithTextBody sets the plain text body
func (m *EmailMessage) WithTextBody(text string) *EmailMessage {
	m.TextBody = text
	return m
}

// WithReplyTo sets the reply-to address
func (m *EmailMessage) WithReplyTo(replyTo string) *EmailMessage {
	m.ReplyTo = replyTo
	return m
}

// WithCC adds CC recipients
func (m *EmailMessage) WithCC(cc ...string) *EmailMessage {
	m.CC = append(m.CC, cc...)
	return m
}

// WithBCC adds BCC recipients
func (m *EmailMessage) WithBCC(bcc ...string) *EmailMessage {
	m.BCC = append(m.BCC, bcc...)
	return m
}

// WithPriority sets the email priority
func (m *EmailMessage) WithPriority(priority Priority) *EmailMessage {
	m.Priority = priority
	return m
}

// WithTag adds a tag to the email
func (m *EmailMessage) WithTag(key, value string) *EmailMessage {
	if m.Tags == nil {
		m.Tags = make(map[string]string)
	}
	m.Tags[key] = value
	return m
}

// WithHeader adds a custom header
func (m *EmailMessage) WithHeader(key, value string) *EmailMessage {
	if m.Headers == nil {
		m.Headers = make(map[string]string)
	}
	m.Headers[key] = value
	return m
}

// WithAttachment adds an attachment
func (m *EmailMessage) WithAttachment(attachment Attachment) *EmailMessage {
	m.Attachments = append(m.Attachments, attachment)
	return m
}

// WithTracking enables open and click tracking
func (m *EmailMessage) WithTracking(trackOpens, trackClicks bool) *EmailMessage {
	m.TrackOpens = trackOpens
	m.TrackClicks = trackClicks
	return m
}

// WithMetadata adds metadata for logging/tracking
func (m *EmailMessage) WithMetadata(key, value string) *EmailMessage {
	if m.Metadata == nil {
		m.Metadata = make(map[string]string)
	}
	m.Metadata[key] = value
	return m
}
