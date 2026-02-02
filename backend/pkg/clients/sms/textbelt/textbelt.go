// Package textbelt provides an SMS client using the TextBelt API.
//
// TextBelt (https://textbelt.com/) is a simple SMS API that offers:
// - 1 free SMS per day using the API key "textbelt"
// - Unlimited SMS with a paid key
//
// This makes it ideal for MVP/testing scenarios where you need real SMS
// delivery but don't want to pay for a full SMS service yet.
package textbelt

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/javaknight1/servicepro/backend/config"
	"github.com/javaknight1/servicepro/backend/pkg/clients/logging"
	"github.com/javaknight1/servicepro/backend/pkg/clients/sms"
)

const (
	// BaseURL is the TextBelt API endpoint
	BaseURL = "https://textbelt.com/text"

	// FreeTierKey is the special key for free tier (1 SMS/day)
	FreeTierKey = "textbelt"

	// DefaultTimeout for HTTP requests
	DefaultTimeout = 30 * time.Second
)

func init() {
	sms.RegisterProvider(sms.ProviderTextBelt, func(ctx context.Context, cfg *config.Config) (sms.Client, error) {
		apiKey := cfg.SMS.TextBelt.APIKey
		if apiKey == "" {
			return nil, fmt.Errorf("TextBelt API key is required (set TEXTBELT_API_KEY, use 'textbelt' for free tier)")
		}

		return NewClient(&Config{
			APIKey:   apiKey,
			SenderID: cfg.SMS.SenderID,
			Timeout:  DefaultTimeout,
		}), nil
	})
}

// Config holds configuration for the TextBelt SMS client
type Config struct {
	// APIKey is the TextBelt API key
	// Use "textbelt" for free tier (1 SMS/day)
	APIKey string

	// SenderID is the sender name (not used by TextBelt, kept for interface compatibility)
	SenderID string

	// Timeout for HTTP requests
	Timeout time.Duration
}

// Client implements the sms.Client interface using TextBelt
type Client struct {
	config     *Config
	httpClient *http.Client
}

// textBeltResponse represents the API response from TextBelt
type textBeltResponse struct {
	Success        bool   `json:"success"`
	TextID         string `json:"textId,omitempty"`
	QuotaRemaining int    `json:"quotaRemaining"`
	Error          string `json:"error,omitempty"`
}

// NewClient creates a new TextBelt SMS client
func NewClient(cfg *Config) *Client {
	if cfg == nil {
		cfg = &Config{
			APIKey:  FreeTierKey,
			Timeout: DefaultTimeout,
		}
	}
	if cfg.Timeout == 0 {
		cfg.Timeout = DefaultTimeout
	}

	return &Client{
		config: cfg,
		httpClient: &http.Client{
			Timeout: cfg.Timeout,
		},
	}
}

// Send implements sms.Client
func (c *Client) Send(ctx context.Context, msg *sms.SMSMessage) (*sms.SendResult, error) {
	// Prepare form data
	data := url.Values{}
	data.Set("phone", msg.PhoneNumber)
	data.Set("message", msg.Content)
	data.Set("key", c.config.APIKey)

	// Create request
	req, err := http.NewRequestWithContext(ctx, "POST", BaseURL, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	// Send request
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send SMS: %w", err)
	}
	defer resp.Body.Close()

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	// Parse response
	var tbResp textBeltResponse
	if err := json.Unmarshal(body, &tbResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w (body: %s)", err, string(body))
	}

	result := &sms.SendResult{
		Success:      tbResp.Success,
		MessageID:    tbResp.TextID,
		Provider:     sms.ProviderTextBelt,
		SentAt:       time.Now(),
		PhoneNumber:  msg.PhoneNumber,
		SegmentCount: msg.CalculateSegments(),
		Metadata: map[string]string{
			"quota_remaining": fmt.Sprintf("%d", tbResp.QuotaRemaining),
		},
	}

	if !tbResp.Success {
		result.Error = tbResp.Error
		if tbResp.Error == "" {
			result.Error = "SMS delivery failed"
		}
		logging.Error(ctx, "[SMS-TEXTBELT] Failed to send", map[string]any{"phone_number": msg.PhoneNumber, "error": result.Error})
		return result, fmt.Errorf("TextBelt error: %s", result.Error)
	}

	logging.Info(ctx, "[SMS-TEXTBELT] Sent", map[string]any{"phone_number": msg.PhoneNumber, "text_id": tbResp.TextID, "quota_remaining": tbResp.QuotaRemaining})
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
				Provider:    sms.ProviderTextBelt,
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
	msg := sms.NewSMSMessage(phoneNumber, fmt.Sprintf("Your verification code is: %s. Valid for %d minutes.", otp, expiryMinutes)).
		WithMessageType(sms.MessageTypeOTP).
		WithPriority(sms.PriorityHigh)
	_, err := c.Send(ctx, msg)
	return err
}

// SendNotification implements sms.Client
func (c *Client) SendNotification(ctx context.Context, phoneNumber, message string) error {
	msg := sms.NewSMSMessage(phoneNumber, message).
		WithMessageType(sms.MessageTypeNotification)
	_, err := c.Send(ctx, msg)
	return err
}

// SendJobUpdate implements sms.Client
func (c *Client) SendJobUpdate(ctx context.Context, phoneNumber, jobNumber, status, message string) error {
	content := fmt.Sprintf("Job #%s - %s: %s", jobNumber, status, message)
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
	msg := sms.NewSMSMessage(phoneNumber, content).
		WithMessageType(sms.MessageTypeReminder).
		WithPriority(sms.PriorityHigh)
	_, err := c.Send(ctx, msg)
	return err
}

// SendInvoiceNotification implements sms.Client
func (c *Client) SendInvoiceNotification(ctx context.Context, phoneNumber, invoiceNumber, amount, paymentURL string) error {
	content := fmt.Sprintf("Invoice #%s for $%s is ready. Pay online: %s", invoiceNumber, amount, paymentURL)
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
	msg := sms.NewSMSMessage(phoneNumber, content).
		WithMessageType(sms.MessageTypeTransactional).
		WithMetadata("invoice_number", invoiceNumber).
		WithMetadata("amount", amount)
	_, err := c.Send(ctx, msg)
	return err
}

// HealthCheck implements sms.Client
func (c *Client) HealthCheck(ctx context.Context) error {
	// TextBelt doesn't have a dedicated health check endpoint
	// We'll make a status check by sending a test request with an invalid number format
	// This verifies the API is reachable without consuming quota

	data := url.Values{}
	data.Set("phone", "0") // Invalid number - will fail validation, not consume quota
	data.Set("message", "health check")
	data.Set("key", c.config.APIKey)

	req, err := http.NewRequestWithContext(ctx, "POST", BaseURL, bytes.NewBufferString(data.Encode()))
	if err != nil {
		return fmt.Errorf("failed to create health check request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("TextBelt health check failed: %w", err)
	}
	defer resp.Body.Close()

	// If we get a response (even an error for invalid number), the API is working
	if resp.StatusCode >= 500 {
		return fmt.Errorf("TextBelt API returned server error: %d", resp.StatusCode)
	}

	return nil
}

// Close implements sms.Client
func (c *Client) Close() error {
	// HTTP client doesn't need explicit cleanup
	return nil
}

// GetProviderInfo implements sms.Client
func (c *Client) GetProviderInfo() sms.ProviderInfo {
	dailyLimit := 0 // Unlimited with paid key
	hasFreeTier := false

	if c.config.APIKey == FreeTierKey {
		dailyLimit = 1
		hasFreeTier = true
	}

	return sms.ProviderInfo{
		Provider:    sms.ProviderTextBelt,
		DisplayName: "TextBelt",
		IsMock:      false,
		HasFreeTier: hasFreeTier,
		DailyLimit:  dailyLimit,
		RateLimit:   0, // No documented rate limit
	}
}

// Ensure Client implements sms.Client
var _ sms.Client = (*Client)(nil)
