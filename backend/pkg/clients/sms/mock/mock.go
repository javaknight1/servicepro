package mock

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/javaknight1/servicepro/backend/config"
	"github.com/javaknight1/servicepro/backend/pkg/clients/sms"
)

func init() {
	sms.RegisterProvider(sms.ProviderMock, func(ctx context.Context, cfg *config.Config) (sms.Client, error) {
		return NewClient(&Config{
			SenderID: cfg.SMS.SenderID,
		}), nil
	})
}

// Config holds configuration for the mock SMS client
type Config struct {
	SenderID string

	// SimulateErrors enables error simulation for testing
	SimulateErrors bool

	// ErrorRate is the rate of simulated errors (0.0 to 1.0)
	ErrorRate float64
}

// Client implements the sms.Client interface for testing and development
type Client struct {
	config *Config
	mu     sync.Mutex

	// SentMessages stores all SMS sent (for testing inspection)
	SentMessages []*sms.SMSMessage
}

// NewClient creates a new mock SMS client
func NewClient(cfg *Config) *Client {
	if cfg == nil {
		cfg = &Config{
			SenderID: "MockSMS",
		}
	}
	return &Client{
		config:       cfg,
		SentMessages: make([]*sms.SMSMessage, 0),
	}
}

// Send implements sms.Client
func (c *Client) Send(ctx context.Context, msg *sms.SMSMessage) (*sms.SendResult, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Store the sent message for testing
	c.SentMessages = append(c.SentMessages, msg)

	// Log the SMS
	log.Printf("[SMS-MOCK] To=%s Content=%q (Segments: %d)", msg.PhoneNumber, msg.Content, msg.CalculateSegments())

	result := &sms.SendResult{
		Success:      true,
		MessageID:    fmt.Sprintf("mock-sms-%d", time.Now().UnixNano()),
		Provider:     sms.ProviderMock,
		SentAt:       time.Now(),
		PhoneNumber:  msg.PhoneNumber,
		SegmentCount: msg.CalculateSegments(),
		Metadata: map[string]string{
			"mock": "true",
		},
	}

	return result, nil
}

// SendBatch implements sms.Client
func (c *Client) SendBatch(ctx context.Context, messages []*sms.SMSMessage) []sms.SendResult {
	results := make([]sms.SendResult, len(messages))
	for i, msg := range messages {
		result, err := c.Send(ctx, msg)
		if err != nil {
			results[i] = sms.SendResult{
				Success:     false,
				Provider:    sms.ProviderMock,
				PhoneNumber: msg.PhoneNumber,
				Error:       err.Error(),
			}
		} else {
			results[i] = *result
		}
	}
	return results
}

// SendOTP implements sms.Client
func (c *Client) SendOTP(ctx context.Context, phoneNumber, otp string, expiryMinutes int) error {
	log.Printf("[SMS-MOCK] OTP to %s: %s (expires in %d minutes)", phoneNumber, otp, expiryMinutes)

	msg := sms.NewSMSMessage(phoneNumber, fmt.Sprintf("Your verification code is: %s. Valid for %d minutes.", otp, expiryMinutes)).
		WithMessageType(sms.MessageTypeOTP).
		WithPriority(sms.PriorityHigh)
	_, err := c.Send(ctx, msg)
	return err
}

// SendNotification implements sms.Client
func (c *Client) SendNotification(ctx context.Context, phoneNumber, message string) error {
	log.Printf("[SMS-MOCK] Notification to %s: %s", phoneNumber, message)

	msg := sms.NewSMSMessage(phoneNumber, message).
		WithMessageType(sms.MessageTypeNotification)
	_, err := c.Send(ctx, msg)
	return err
}

// SendJobUpdate implements sms.Client
func (c *Client) SendJobUpdate(ctx context.Context, phoneNumber, jobNumber, status, message string) error {
	content := fmt.Sprintf("Job #%s - %s: %s", jobNumber, status, message)
	log.Printf("[SMS-MOCK] Job update to %s: %s", phoneNumber, content)

	msg := sms.NewSMSMessage(phoneNumber, content).
		WithMessageType(sms.MessageTypeNotification).
		WithMetadata("job_number", jobNumber).
		WithMetadata("status", status)
	_, err := c.Send(ctx, msg)
	return err
}

// SendAppointmentReminder implements sms.Client
func (c *Client) SendAppointmentReminder(ctx context.Context, phoneNumber, appointmentTime, jobDescription string) error {
	content := fmt.Sprintf("Reminder: Your appointment for %s is scheduled at %s.", jobDescription, appointmentTime)
	log.Printf("[SMS-MOCK] Appointment reminder to %s: %s", phoneNumber, content)

	msg := sms.NewSMSMessage(phoneNumber, content).
		WithMessageType(sms.MessageTypeReminder).
		WithPriority(sms.PriorityHigh)
	_, err := c.Send(ctx, msg)
	return err
}

// SendInvoiceNotification implements sms.Client
func (c *Client) SendInvoiceNotification(ctx context.Context, phoneNumber, invoiceNumber, amount, paymentURL string) error {
	content := fmt.Sprintf("Invoice #%s for $%s is ready. Pay online: %s", invoiceNumber, amount, paymentURL)
	log.Printf("[SMS-MOCK] Invoice notification to %s: %s", phoneNumber, content)

	msg := sms.NewSMSMessage(phoneNumber, content).
		WithMessageType(sms.MessageTypeTransactional).
		WithMetadata("invoice_number", invoiceNumber).
		WithMetadata("amount", amount)
	_, err := c.Send(ctx, msg)
	return err
}

// SendPaymentConfirmation implements sms.Client
func (c *Client) SendPaymentConfirmation(ctx context.Context, phoneNumber, invoiceNumber, amount string) error {
	content := fmt.Sprintf("Payment received! Invoice #%s - $%s. Thank you!", invoiceNumber, amount)
	log.Printf("[SMS-MOCK] Payment confirmation to %s: %s", phoneNumber, content)

	msg := sms.NewSMSMessage(phoneNumber, content).
		WithMessageType(sms.MessageTypeTransactional).
		WithMetadata("invoice_number", invoiceNumber).
		WithMetadata("amount", amount)
	_, err := c.Send(ctx, msg)
	return err
}

// HealthCheck implements sms.Client
func (c *Client) HealthCheck(ctx context.Context) error {
	return nil
}

// Close implements sms.Client
func (c *Client) Close() error {
	return nil
}

// GetProviderInfo implements sms.Client
func (c *Client) GetProviderInfo() sms.ProviderInfo {
	return sms.ProviderInfo{
		Provider:    sms.ProviderMock,
		DisplayName: "Mock SMS (Development)",
		IsMock:      true,
		HasFreeTier: true,
		DailyLimit:  0, // Unlimited
		RateLimit:   0, // Unlimited
	}
}

// GetSentMessages returns all SMS sent through this mock client
func (c *Client) GetSentMessages() []*sms.SMSMessage {
	c.mu.Lock()
	defer c.mu.Unlock()

	messages := make([]*sms.SMSMessage, len(c.SentMessages))
	copy(messages, c.SentMessages)
	return messages
}

// ClearSentMessages clears the sent messages list
func (c *Client) ClearSentMessages() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.SentMessages = make([]*sms.SMSMessage, 0)
}

// LastSentMessage returns the last SMS sent, or nil if none
func (c *Client) LastSentMessage() *sms.SMSMessage {
	c.mu.Lock()
	defer c.mu.Unlock()

	if len(c.SentMessages) == 0 {
		return nil
	}
	return c.SentMessages[len(c.SentMessages)-1]
}

// Ensure Client implements sms.Client
var _ sms.Client = (*Client)(nil)
