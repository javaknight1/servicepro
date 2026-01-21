package models

// VerifyEmailRequest represents the email verification request payload
type VerifyEmailRequest struct {
	Token string `json:"token" binding:"required"`
}

// VerifyEmailResponse represents the email verification response
type VerifyEmailResponse struct {
	Message string `json:"message"`
}

// ResendVerificationRequest represents the resend verification email request
type ResendVerificationRequest struct {
	Email string `json:"email" binding:"required,email"`
}

// ResendVerificationResponse represents the resend verification response
type ResendVerificationResponse struct {
	Message string `json:"message"`
}

// SESBounceNotification represents AWS SES bounce notification
type SESBounceNotification struct {
	NotificationType string      `json:"notificationType"`
	Bounce           BounceEvent `json:"bounce"`
	Mail             MailEvent   `json:"mail"`
}

// BounceEvent represents the bounce event details
type BounceEvent struct {
	BounceType        string             `json:"bounceType"`
	BounceSubType     string             `json:"bounceSubType"`
	BouncedRecipients []BouncedRecipient `json:"bouncedRecipients"`
	Timestamp         string             `json:"timestamp"`
	FeedbackID        string             `json:"feedbackId"`
}

// BouncedRecipient represents a recipient that bounced
type BouncedRecipient struct {
	EmailAddress   string `json:"emailAddress"`
	Action         string `json:"action"`
	Status         string `json:"status"`
	DiagnosticCode string `json:"diagnosticCode"`
}

// MailEvent represents the mail event details
type MailEvent struct {
	Timestamp        string   `json:"timestamp"`
	Source           string   `json:"source"`
	SourceArn        string   `json:"sourceArn"`
	SourceIP         string   `json:"sourceIp"`
	SendingAccountID string   `json:"sendingAccountId"`
	MessageID        string   `json:"messageId"`
	Destination      []string `json:"destination"`
}
