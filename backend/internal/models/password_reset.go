package models

// PasswordResetRequestRequest represents the password reset request payload
type PasswordResetRequestRequest struct {
	Email string `json:"email" binding:"required,email"`
}

// PasswordResetRequestResponse represents the password reset request response
type PasswordResetRequestResponse struct {
	Message string `json:"message"`
}

// PasswordResetRequest represents the password reset payload with token
type PasswordResetRequest struct {
	Token       string `json:"token" binding:"required"`
	NewPassword string `json:"newPassword" binding:"required,min=8"`
}

// PasswordResetResponse represents the password reset response
type PasswordResetResponse struct {
	Message string `json:"message"`
}
