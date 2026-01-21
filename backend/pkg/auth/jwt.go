package auth

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"github.com/javaknight1/servicepro/backend/config"
)

var (
	ErrInvalidToken = errors.New("invalid token")
	ErrExpiredToken = errors.New("token has expired")
)

// TokenType represents the type of token
type TokenType string

const (
	AccessToken  TokenType = "access"
	RefreshToken TokenType = "refresh"
)

// Claims represents the JWT claims
type Claims struct {
	UserID   string    `json:"user_id"` // Changed to string to support both int and UUID
	Email    string    `json:"email"`
	TenantID string    `json:"tenant_id,omitempty"` // Current active tenant
	Type     TokenType `json:"type"`
	jwt.RegisteredClaims
}

// JWTManager handles JWT token operations
type JWTManager struct {
	secretKey            string
	accessTokenDuration  time.Duration
	refreshTokenDuration time.Duration
}

// NewJWTManager creates a new JWT manager
func NewJWTManager(cfg *config.JWTConfig) *JWTManager {
	return &JWTManager{
		secretKey:            cfg.Secret,
		accessTokenDuration:  cfg.AccessExpiration,
		refreshTokenDuration: cfg.RefreshExpiration,
	}
}

// GenerateAccessToken generates an access token for a user
func (m *JWTManager) GenerateAccessToken(userID string, email string) (string, error) {
	return m.generateToken(userID, email, "", AccessToken, m.accessTokenDuration)
}

// GenerateAccessTokenWithTenant generates an access token with tenant context
func (m *JWTManager) GenerateAccessTokenWithTenant(userID string, email string, tenantID string) (string, error) {
	return m.generateToken(userID, email, tenantID, AccessToken, m.accessTokenDuration)
}

// GenerateRefreshToken generates a refresh token for a user
func (m *JWTManager) GenerateRefreshToken(userID string, email string) (string, error) {
	return m.generateToken(userID, email, "", RefreshToken, m.refreshTokenDuration)
}

// GenerateRefreshTokenWithTenant generates a refresh token with tenant context
func (m *JWTManager) GenerateRefreshTokenWithTenant(userID string, email string, tenantID string) (string, error) {
	return m.generateToken(userID, email, tenantID, RefreshToken, m.refreshTokenDuration)
}

// generateToken generates a JWT token
func (m *JWTManager) generateToken(userID string, email string, tenantID string, tokenType TokenType, duration time.Duration) (string, error) {
	now := time.Now()

	claims := &Claims{
		UserID:   userID,
		Email:    email,
		TenantID: tenantID,
		Type:     tokenType,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(duration)),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(m.secretKey))
	if err != nil {
		return "", fmt.Errorf("failed to sign token: %w", err)
	}

	return tokenString, nil
}

// ValidateToken validates a JWT token and returns the claims
func (m *JWTManager) ValidateToken(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(
		tokenString,
		&Claims{},
		func(token *jwt.Token) (interface{}, error) {
			// Verify signing method
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
			}
			return []byte(m.secretKey), nil
		},
	)

	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, ErrExpiredToken
		}
		return nil, ErrInvalidToken
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, ErrInvalidToken
	}

	return claims, nil
}

// GetAccessTokenDuration returns the access token duration in seconds
func (m *JWTManager) GetAccessTokenDuration() int {
	return int(m.accessTokenDuration.Seconds())
}

// GenerateTokenPair generates both access and refresh tokens
func (m *JWTManager) GenerateTokenPair(userID string, email string) (accessToken string, refreshToken string, err error) {
	accessToken, err = m.GenerateAccessToken(userID, email)
	if err != nil {
		return "", "", err
	}

	refreshToken, err = m.GenerateRefreshToken(userID, email)
	if err != nil {
		return "", "", err
	}

	return accessToken, refreshToken, nil
}

// GenerateTokenPairWithTenant generates both access and refresh tokens with tenant context
func (m *JWTManager) GenerateTokenPairWithTenant(userID string, email string, tenantID string) (accessToken string, refreshToken string, err error) {
	accessToken, err = m.GenerateAccessTokenWithTenant(userID, email, tenantID)
	if err != nil {
		return "", "", err
	}

	refreshToken, err = m.GenerateRefreshTokenWithTenant(userID, email, tenantID)
	if err != nil {
		return "", "", err
	}

	return accessToken, refreshToken, nil
}
