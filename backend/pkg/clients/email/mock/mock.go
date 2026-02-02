package mock

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/javaknight1/servicepro/backend/config"
	"github.com/javaknight1/servicepro/backend/internal/models"
	"github.com/javaknight1/servicepro/backend/pkg/clients/email"
	"github.com/javaknight1/servicepro/backend/pkg/clients/logging"
)

func init() {
	email.RegisterProvider(email.ProviderMock, func(ctx context.Context, cfg *config.Config) (email.Client, error) {
		mockCfg := &Config{
			FromEmail: cfg.AWS.SES.FromEmail,
			FromName:  cfg.AWS.SES.FromName,
		}
		if mockCfg.FromEmail == "" {
			mockCfg.FromEmail = cfg.Resend.FromEmail
		}
		if mockCfg.FromName == "" {
			mockCfg.FromName = cfg.Resend.FromName
		}
		return NewClient(mockCfg), nil
	})
}

// Config holds configuration for the mock email client
type Config struct {
	FromEmail string
	FromName  string

	// SimulateErrors enables error simulation for testing
	SimulateErrors bool

	// ErrorRate is the rate of simulated errors (0.0 to 1.0)
	ErrorRate float64
}

// Client implements the email.Client interface for testing and development
type Client struct {
	config *Config
	mu     sync.Mutex

	// SentEmails stores all emails sent (for testing inspection)
	SentEmails []*email.EmailMessage
}

// NewClient creates a new mock email client
func NewClient(cfg *Config) *Client {
	if cfg == nil {
		cfg = &Config{
			FromEmail: "noreply@example.com",
			FromName:  "ServicePro (Mock)",
		}
	}
	return &Client{
		config:     cfg,
		SentEmails: make([]*email.EmailMessage, 0),
	}
}

// Send implements email.Client
func (c *Client) Send(ctx context.Context, msg *email.EmailMessage) (*email.SendResult, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Store the sent email for testing
	c.SentEmails = append(c.SentEmails, msg)

	// Log the email
	logging.Info(ctx, "[EMAIL-MOCK] Sent", map[string]any{"to": msg.To, "subject": msg.Subject})

	result := &email.SendResult{
		Success:   true,
		MessageID: fmt.Sprintf("mock-%d", time.Now().UnixNano()),
		Provider:  email.ProviderMock,
		SentAt:    time.Now(),
		Metadata: map[string]string{
			"mock": "true",
		},
	}

	return result, nil
}

// SendWelcomeEmail implements email.Client
func (c *Client) SendWelcomeEmail(ctx context.Context, to, name string) error {
	logging.Info(ctx, "[EMAIL-MOCK] Sending welcome email", map[string]any{"to": to, "name": name})

	msg := email.NewEmailMessage(to, "Welcome to ServicePro!", fmt.Sprintf("Welcome, %s!", name))
	_, err := c.Send(ctx, msg)
	return err
}

// SendPasswordResetEmail implements email.Client
func (c *Client) SendPasswordResetEmail(ctx context.Context, to, resetToken, resetURL string) error {
	fullResetURL := fmt.Sprintf("%s?token=%s", resetURL, resetToken)
	logging.Info(ctx, "[EMAIL-MOCK] Sending password reset email", map[string]any{"to": to, "reset_link": fullResetURL})

	msg := email.NewEmailMessage(to, "Password Reset Request - ServicePro", fmt.Sprintf("Reset your password: %s", fullResetURL))
	_, err := c.Send(ctx, msg)
	return err
}

// SendPasswordResetConfirmationEmail implements email.Client
func (c *Client) SendPasswordResetConfirmationEmail(ctx context.Context, to string) error {
	logging.Info(ctx, "[EMAIL-MOCK] Sending password reset confirmation email", map[string]any{"to": to})

	msg := email.NewEmailMessage(to, "Password Successfully Reset - ServicePro", "Your password has been successfully reset.")
	_, err := c.Send(ctx, msg)
	return err
}

// SendEmailVerificationEmail implements email.Client
func (c *Client) SendEmailVerificationEmail(ctx context.Context, to, verificationToken, verificationURL string) error {
	fullURL := fmt.Sprintf("%s?token=%s", verificationURL, verificationToken)
	logging.Info(ctx, "[EMAIL-MOCK] Sending email verification", map[string]any{"to": to, "verification_link": fullURL})

	msg := email.NewEmailMessage(to, "Verify Your Email Address - ServicePro", fmt.Sprintf("Verify your email: %s", fullURL))
	_, err := c.Send(ctx, msg)
	return err
}

// SendEmailVerificationReminderEmail implements email.Client
func (c *Client) SendEmailVerificationReminderEmail(ctx context.Context, to, verificationToken, verificationURL string) error {
	fullURL := fmt.Sprintf("%s?token=%s", verificationURL, verificationToken)
	logging.Info(ctx, "[EMAIL-MOCK] Sending email verification reminder", map[string]any{"to": to, "verification_link": fullURL})

	msg := email.NewEmailMessage(to, "Reminder: Verify Your Email Address - ServicePro", fmt.Sprintf("Verify your email: %s", fullURL))
	_, err := c.Send(ctx, msg)
	return err
}

// SendEmailVerificationSuccessEmail implements email.Client
func (c *Client) SendEmailVerificationSuccessEmail(ctx context.Context, to string) error {
	logging.Info(ctx, "[EMAIL-MOCK] Sending email verification success", map[string]any{"to": to})

	msg := email.NewEmailMessage(to, "Email Verified Successfully - ServicePro", "Your email has been successfully verified.")
	_, err := c.Send(ctx, msg)
	return err
}

// SendOrganizationInviteEmail implements email.Client
func (c *Client) SendOrganizationInviteEmail(ctx context.Context, to, orgName, inviterName, roleName, actionURL string, userExists bool) error {
	var subject string
	if userExists {
		subject = fmt.Sprintf("You've been invited to join %s on ServicePro", orgName)
		logging.Info(ctx, "[EMAIL-MOCK] Sending organization invite to existing user", map[string]any{"to": to, "org_name": orgName, "role": roleName, "invitation_link": actionURL})
	} else {
		subject = fmt.Sprintf("You've been invited to join %s on ServicePro", orgName)
		logging.Info(ctx, "[EMAIL-MOCK] Sending organization invite to new user", map[string]any{"to": to, "org_name": orgName, "role": roleName, "invitation_link": actionURL})
	}

	msg := email.NewEmailMessage(to, subject, fmt.Sprintf("Invitation from %s to join %s. Action URL: %s", inviterName, orgName, actionURL))
	_, err := c.Send(ctx, msg)
	return err
}

// SendQuoteEmail implements email.Client
func (c *Client) SendQuoteEmail(ctx context.Context, to string, quote *models.Quote, pdfAttachment *email.Attachment, downloadURL string) error {
	logging.Info(ctx, "[EMAIL-MOCK] Sending quote email", map[string]any{"to": to, "quote_number": quote.QuoteNumber, "download_url": downloadURL})

	subject := fmt.Sprintf("Quote %s from ServicePro", quote.QuoteNumber)
	body := fmt.Sprintf("Quote #%s - Total: $%s. Valid until: %s",
		quote.QuoteNumber, quote.Total.StringFixed(2), quote.ValidUntil.Format("January 2, 2006"))

	msg := email.NewEmailMessage(to, subject, body)
	if pdfAttachment != nil {
		msg.WithAttachment(*pdfAttachment)
	}
	_, err := c.Send(ctx, msg)
	return err
}

// SendInvoiceEmail implements email.Client
func (c *Client) SendInvoiceEmail(ctx context.Context, to string, invoice *models.Invoice, paymentURL string, pdfAttachment *email.Attachment, downloadURL string) error {
	logging.Info(ctx, "[EMAIL-MOCK] Sending invoice email", map[string]any{"to": to, "invoice_number": invoice.InvoiceNumber, "payment_url": paymentURL, "download_url": downloadURL})

	subject := fmt.Sprintf("Invoice %s from ServicePro", invoice.InvoiceNumber)
	body := fmt.Sprintf("Invoice #%s - Total: $%s. Pay here: %s",
		invoice.InvoiceNumber, invoice.TotalAmount.StringFixed(2), paymentURL)

	msg := email.NewEmailMessage(to, subject, body)
	if pdfAttachment != nil {
		msg.WithAttachment(*pdfAttachment)
	}
	_, err := c.Send(ctx, msg)
	return err
}

// SendPaymentReceiptEmail implements email.Client
func (c *Client) SendPaymentReceiptEmail(ctx context.Context, to string, invoice *models.Invoice, pdfAttachment *email.Attachment, downloadURL string) error {
	logging.Info(ctx, "[EMAIL-MOCK] Sending payment receipt email", map[string]any{"to": to, "invoice_number": invoice.InvoiceNumber, "download_url": downloadURL})

	subject := fmt.Sprintf("Payment Receipt for Invoice %s - ServicePro", invoice.InvoiceNumber)
	body := fmt.Sprintf("Payment received for Invoice #%s - Amount Paid: $%s. Thank you!",
		invoice.InvoiceNumber, invoice.AmountPaid.StringFixed(2))

	msg := email.NewEmailMessage(to, subject, body)
	if pdfAttachment != nil {
		msg.WithAttachment(*pdfAttachment)
	}
	_, err := c.Send(ctx, msg)
	return err
}

// HealthCheck implements email.Client
func (c *Client) HealthCheck(ctx context.Context) error {
	return nil
}

// Close implements email.Client
func (c *Client) Close() error {
	return nil
}

// GetSentEmails returns all emails sent through this mock client
func (c *Client) GetSentEmails() []*email.EmailMessage {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Return a copy to prevent race conditions
	emails := make([]*email.EmailMessage, len(c.SentEmails))
	copy(emails, c.SentEmails)
	return emails
}

// ClearSentEmails clears the sent emails list
func (c *Client) ClearSentEmails() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.SentEmails = make([]*email.EmailMessage, 0)
}

// LastSentEmail returns the last email sent, or nil if none
func (c *Client) LastSentEmail() *email.EmailMessage {
	c.mu.Lock()
	defer c.mu.Unlock()

	if len(c.SentEmails) == 0 {
		return nil
	}
	return c.SentEmails[len(c.SentEmails)-1]
}

// Ensure Client implements email.Client
var _ email.Client = (*Client)(nil)
