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
