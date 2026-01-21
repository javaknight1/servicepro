package auth

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
)

// GenerateSecureToken generates a cryptographically secure random token
func GenerateSecureToken(length int) (string, error) {
	// Create a byte slice of the specified length
	bytes := make([]byte, length)

	// Read random bytes from crypto/rand
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("failed to generate random token: %w", err)
	}

	// Encode to base64 URL-safe format (no padding)
	token := base64.RawURLEncoding.EncodeToString(bytes)

	return token, nil
}

// GeneratePasswordResetToken generates a 32-byte secure token for password reset
func GeneratePasswordResetToken() (string, error) {
	return GenerateSecureToken(32)
}

// GenerateEmailVerificationToken generates a 32-byte secure token for email verification
func GenerateEmailVerificationToken() (string, error) {
	return GenerateSecureToken(32)
}
