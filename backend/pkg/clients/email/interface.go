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
