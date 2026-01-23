package email

import (
	"context"
)

// Client defines the interface for email operations.
// This interface provides both low-level Send capability and
// convenience methods for common email types.
type Client interface {
	// Send sends an email message
	Send(ctx context.Context, msg *EmailMessage) (*SendResult, error)

	// Convenience methods for common email types
	// These match the existing EmailServiceInterface methods

	// SendWelcomeEmail sends a welcome email to a new user
	SendWelcomeEmail(ctx context.Context, to, name string) error

	// SendPasswordResetEmail sends a password reset email with token
	SendPasswordResetEmail(ctx context.Context, to, resetToken, resetURL string) error

	// SendPasswordResetConfirmationEmail sends confirmation after successful password reset
	SendPasswordResetConfirmationEmail(ctx context.Context, to string) error

	// SendEmailVerificationEmail sends an email verification link to the user
	SendEmailVerificationEmail(ctx context.Context, to, verificationToken, verificationURL string) error

	// SendEmailVerificationReminderEmail sends a reminder to verify email
	SendEmailVerificationReminderEmail(ctx context.Context, to, verificationToken, verificationURL string) error

	// SendEmailVerificationSuccessEmail sends confirmation after successful email verification
	SendEmailVerificationSuccessEmail(ctx context.Context, to string) error

	// Lifecycle methods

	// HealthCheck verifies the email service is operational
	HealthCheck(ctx context.Context) error

	// Close releases any resources held by the client
	Close() error
}

// LegacyAdapter wraps the Client interface to implement the old EmailServiceInterface.
// This provides backward compatibility for existing code.
type LegacyAdapter struct {
	client Client
}

// NewLegacyAdapter creates a new adapter that wraps a Client to provide
// the old EmailServiceInterface
func NewLegacyAdapter(client Client) *LegacyAdapter {
	return &LegacyAdapter{client: client}
}

// SendWelcomeEmail implements EmailServiceInterface
func (a *LegacyAdapter) SendWelcomeEmail(to, name string) error {
	return a.client.SendWelcomeEmail(context.Background(), to, name)
}

// SendPasswordResetEmail implements EmailServiceInterface
func (a *LegacyAdapter) SendPasswordResetEmail(to, resetToken, resetURL string) error {
	return a.client.SendPasswordResetEmail(context.Background(), to, resetToken, resetURL)
}

// SendPasswordResetConfirmationEmail implements EmailServiceInterface
func (a *LegacyAdapter) SendPasswordResetConfirmationEmail(to string) error {
	return a.client.SendPasswordResetConfirmationEmail(context.Background(), to)
}

// SendEmailVerificationEmail implements EmailServiceInterface
func (a *LegacyAdapter) SendEmailVerificationEmail(to, verificationToken, verificationURL string) error {
	return a.client.SendEmailVerificationEmail(context.Background(), to, verificationToken, verificationURL)
}

// SendEmailVerificationReminderEmail implements EmailServiceInterface
func (a *LegacyAdapter) SendEmailVerificationReminderEmail(to, verificationToken, verificationURL string) error {
	return a.client.SendEmailVerificationReminderEmail(context.Background(), to, verificationToken, verificationURL)
}

// SendEmailVerificationSuccessEmail implements EmailServiceInterface
func (a *LegacyAdapter) SendEmailVerificationSuccessEmail(to string) error {
	return a.client.SendEmailVerificationSuccessEmail(context.Background(), to)
}

// SendEmail implements EmailServiceInterface
func (a *LegacyAdapter) SendEmail(to, subject, body string) error {
	msg := NewEmailMessage(to, subject, body)
	_, err := a.client.Send(context.Background(), msg)
	return err
}
