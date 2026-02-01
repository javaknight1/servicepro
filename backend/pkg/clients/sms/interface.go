package sms

import (
	"context"
)

// Client defines the interface for SMS operations.
// This interface provides both low-level Send capability and
// convenience methods for common SMS types.
type Client interface {
	// Send sends an SMS message
	Send(ctx context.Context, msg *SMSMessage) (*SendResult, error)

	// SendBatch sends multiple SMS messages
	SendBatch(ctx context.Context, messages []*SMSMessage) []SendResult

	// Convenience methods for common SMS types

	// SendOTP sends a one-time password to the specified phone number
	SendOTP(ctx context.Context, phoneNumber, otp string, expiryMinutes int) error

	// SendNotification sends a general notification
	SendNotification(ctx context.Context, phoneNumber, message string) error

	// SendJobUpdate sends a job status update notification
	SendJobUpdate(ctx context.Context, phoneNumber, jobNumber, status, message string) error

	// SendAppointmentReminder sends an appointment reminder
	SendAppointmentReminder(ctx context.Context, phoneNumber, appointmentTime, jobDescription string) error

	// SendInvoiceNotification sends an invoice notification with payment link
	SendInvoiceNotification(ctx context.Context, phoneNumber, invoiceNumber, amount, paymentURL string) error

	// SendPaymentConfirmation sends a payment confirmation
	SendPaymentConfirmation(ctx context.Context, phoneNumber, invoiceNumber, amount string) error

	// Lifecycle methods

	// HealthCheck verifies the SMS service is operational
	HealthCheck(ctx context.Context) error

	// Close releases any resources held by the client
	Close() error

	// GetProviderInfo returns information about the current provider
	GetProviderInfo() ProviderInfo
}

// ProviderInfo contains information about the SMS provider
type ProviderInfo struct {
	Provider    Provider
	DisplayName string
	IsMock      bool
	HasFreeTier bool
	DailyLimit  int
	RateLimit   float64 // Messages per second
}
