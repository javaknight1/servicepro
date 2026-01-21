package auth

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/javaknight1/servicepro/backend/config"
)

func getTestJWTConfig() *config.JWTConfig {
	return &config.JWTConfig{
		Secret:            "test-secret-key",
		AccessExpiration:  time.Hour,
		RefreshExpiration: time.Hour * 24 * 7,
	}
}

func TestGenerateAccessToken(t *testing.T) {
	manager := NewJWTManager(getTestJWTConfig())
	userID := "550e8400-e29b-41d4-a716-446655440000"
	email := "test@example.com"

	token, err := manager.GenerateAccessToken(userID, email)
	assert.NoError(t, err)
	assert.NotEmpty(t, token)
}

func TestGenerateRefreshToken(t *testing.T) {
	manager := NewJWTManager(getTestJWTConfig())
	userID := "550e8400-e29b-41d4-a716-446655440000"
	email := "test@example.com"

	token, err := manager.GenerateRefreshToken(userID, email)
	assert.NoError(t, err)
	assert.NotEmpty(t, token)
}

func TestValidateToken_Success(t *testing.T) {
	manager := NewJWTManager(getTestJWTConfig())
	userID := "550e8400-e29b-41d4-a716-446655440000"
	email := "test@example.com"

	token, _ := manager.GenerateAccessToken(userID, email)

	claims, err := manager.ValidateToken(token)
	assert.NoError(t, err)
	assert.NotNil(t, claims)
	assert.Equal(t, userID, claims.UserID)
	assert.Equal(t, email, claims.Email)
	assert.Equal(t, AccessToken, claims.Type)
}

func TestValidateToken_InvalidToken(t *testing.T) {
	manager := NewJWTManager(getTestJWTConfig())
	invalidToken := "invalid.token.here"

	claims, err := manager.ValidateToken(invalidToken)
	assert.Error(t, err)
	assert.Nil(t, claims)
	assert.Equal(t, ErrInvalidToken, err)
}

func TestValidateToken_ExpiredToken(t *testing.T) {
	// Create a manager with very short expiration
	cfg := getTestJWTConfig()
	cfg.AccessExpiration = time.Millisecond
	manager := NewJWTManager(cfg)

	userID := "550e8400-e29b-41d4-a716-446655440000"
	email := "test@example.com"

	token, _ := manager.GenerateAccessToken(userID, email)

	// Wait for token to expire
	time.Sleep(time.Millisecond * 10)

	claims, err := manager.ValidateToken(token)
	assert.Error(t, err)
	assert.Nil(t, claims)
	assert.Equal(t, ErrExpiredToken, err)
}

func TestGetAccessTokenDuration(t *testing.T) {
	manager := NewJWTManager(getTestJWTConfig())
	duration := manager.GetAccessTokenDuration()
	assert.Equal(t, 3600, duration) // 1 hour in seconds
}
