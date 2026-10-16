package services

import (
	"errors"
	"time"

	"github.com/google/uuid"

	"github.com/javaknight1/servicepro/backend/config"
	"github.com/javaknight1/servicepro/backend/internal/models"
	"github.com/javaknight1/servicepro/backend/internal/repository"
	"github.com/javaknight1/servicepro/backend/pkg/auth"
)

var (
	ErrInvalidCredentials  = errors.New("invalid email or password")
	ErrAccountLocked       = errors.New("account is locked due to too many failed login attempts")
	ErrEmailNotVerified    = errors.New("email address has not been verified")
	ErrInvalidRefreshToken = errors.New("invalid or expired refresh token")
)

// AuthService handles authentication business logic
type AuthService struct {
	userRepo   repository.UserRepositoryInterface
	jwtManager *auth.JWTManager
	config     *config.AuthConfig
}

// NewAuthService creates a new auth service
func NewAuthService(
	userRepo repository.UserRepositoryInterface,
	jwtManager *auth.JWTManager,
	cfg *config.AuthConfig,
) *AuthService {
	return &AuthService{
		userRepo:   userRepo,
		jwtManager: jwtManager,
		config:     cfg,
	}
}

// Login authenticates a user and returns tokens
func (s *AuthService) Login(email, password string) (*models.LoginResponse, error) {
	// Get user by email
	user, err := s.userRepo.GetByEmail(email)
	if err != nil {
		if errors.Is(err, repository.ErrUserNotFound) {
			return nil, ErrInvalidCredentials
		}
		return nil, err
	}

	// Check if account is locked
	if user.IsLocked() {
		return nil, ErrAccountLocked
	}

	// Check if email is verified
	if !user.EmailVerified {
		return nil, ErrEmailNotVerified
	}

	// Verify password
	if err := auth.ComparePassword(user.PasswordHash, password); err != nil {
		// Password is incorrect - increment failed login count
		if err := s.handleFailedLogin(user); err != nil {
			return nil, err
		}
		return nil, ErrInvalidCredentials
	}

	// Password is correct - reset failed login count
	if gormRepo, ok := s.userRepo.(repository.UserRepositoryGormInterface); ok {
		if err := gormRepo.ResetFailedLoginCountByUUID(user.ID); err != nil {
			return nil, err
		}
	}

	// Generate tokens bound to the user's current token version
	accessToken, refreshToken, err := s.jwtManager.GenerateTokenPairForVersion(user.ID.String(), user.Email, user.TokenVersion)
	if err != nil {
		return nil, err
	}

	return &models.LoginResponse{
		Token:        accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    s.jwtManager.GetAccessTokenDuration(),
	}, nil
}

// handleFailedLogin handles failed login attempts and account lockout
func (s *AuthService) handleFailedLogin(user *models.User) error {
	// Use GORM repository if available
	if gormRepo, ok := s.userRepo.(repository.UserRepositoryGormInterface); ok {
		// Increment failed login count
		if err := gormRepo.IncrementFailedLoginCountByUUID(user.ID); err != nil {
			return err
		}

		// Check if we should lock the account
		newFailedCount := user.FailedLoginCount + 1
		if newFailedCount >= s.config.MaxLoginAttempts {
			// Lock the account
			lockUntil := time.Now().Add(s.config.LockoutDuration)
			if err := gormRepo.LockAccountByUUID(user.ID, lockUntil); err != nil {
				return err
			}
		}
	}

	return nil
}

// RefreshToken validates a refresh token and issues new tokens
func (s *AuthService) RefreshToken(refreshToken string) (*models.LoginResponse, error) {
	// Validate the refresh token
	claims, err := s.jwtManager.ValidateToken(refreshToken)
	if err != nil {
		return nil, ErrInvalidRefreshToken
	}

	// Ensure this is a refresh token
	if claims.Type != auth.RefreshToken {
		return nil, ErrInvalidRefreshToken
	}

	// Verify user still exists and is not locked
	user, err := s.userRepo.GetByEmail(claims.Email)
	if err != nil {
		return nil, ErrInvalidRefreshToken
	}

	// Tokens issued before a logout or password reset are revoked.
	if user.ID.String() != claims.UserID || user.TokenVersion != claims.TokenVersion {
		return nil, ErrInvalidRefreshToken
	}

	if user.IsLocked() {
		return nil, ErrAccountLocked
	}

	// Check if email is verified
	if !user.EmailVerified {
		return nil, ErrEmailNotVerified
	}

	// Generate new tokens
	accessToken, newRefreshToken, err := s.jwtManager.GenerateTokenPairForVersion(user.ID.String(), user.Email, user.TokenVersion)
	if err != nil {
		return nil, err
	}

	return &models.LoginResponse{
		Token:        accessToken,
		RefreshToken: newRefreshToken,
		ExpiresIn:    s.jwtManager.GetAccessTokenDuration(),
	}, nil
}

// tokenVersionBumper is implemented by repositories that can revoke all of a
// user's tokens.
type tokenVersionBumper interface {
	IncrementTokenVersion(userID uuid.UUID) error
}

// Logout revokes every outstanding access and refresh token of the user who
// owns token (refresh or access). Invalid or expired tokens are ignored: there
// is nothing left to revoke.
func (s *AuthService) Logout(token string) error {
	if token == "" {
		return nil
	}
	claims, err := s.jwtManager.ValidateToken(token)
	if err != nil {
		return nil
	}
	userID, err := uuid.Parse(claims.UserID)
	if err != nil {
		return nil
	}
	if b, ok := s.userRepo.(tokenVersionBumper); ok {
		return b.IncrementTokenVersion(userID)
	}
	return nil
}
