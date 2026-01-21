package auth

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGenerateSecureToken(t *testing.T) {
	length := 32

	token, err := GenerateSecureToken(length)
	assert.NoError(t, err)
	assert.NotEmpty(t, token)

	// Base64 URL encoding of 32 bytes is ~43 characters
	assert.Greater(t, len(token), 40)
}

func TestGeneratePasswordResetToken(t *testing.T) {
	token, err := GeneratePasswordResetToken()
	assert.NoError(t, err)
	assert.NotEmpty(t, token)

	// Should generate a valid base64 URL-safe string
	assert.Greater(t, len(token), 40)
}

func TestGenerateSecureToken_Uniqueness(t *testing.T) {
	// Generate multiple tokens and ensure they're unique
	tokens := make(map[string]bool)
	iterations := 100

	for i := 0; i < iterations; i++ {
		token, err := GeneratePasswordResetToken()
		assert.NoError(t, err)

		// Check if token already exists
		assert.False(t, tokens[token], "Token should be unique")
		tokens[token] = true
	}

	// All tokens should be unique
	assert.Equal(t, iterations, len(tokens))
}

func TestGenerateSecureToken_DifferentLengths(t *testing.T) {
	lengths := []int{16, 32, 64, 128}

	for _, length := range lengths {
		t.Run(string(rune(length)), func(t *testing.T) {
			token, err := GenerateSecureToken(length)
			assert.NoError(t, err)
			assert.NotEmpty(t, token)

			// Verify token is properly generated
			assert.Greater(t, len(token), 0)
		})
	}
}
