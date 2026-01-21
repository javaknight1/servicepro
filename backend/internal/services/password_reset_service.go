package services

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/javaknight1/servicepro/backend/internal/repository"
	"github.com/javaknight1/servicepro/backend/pkg/auth"
	"github.com/javaknight1/servicepro/backend/pkg/email"
)

const (
	// Reset token expiry duration (1 hour)
	ResetTokenExpiry = time.Hour

	// Redis key prefix for reset tokens
	ResetTokenPrefix = "password_reset:"
)

var (
	ErrInvalidResetToken    = errors.New("invalid or expired reset token")
	ErrUserNotFoundForReset = errors.New("user not found")
)

// PasswordResetServiceInterface defines the interface for password reset operations
type PasswordResetServiceInterface interface {
	RequestPasswordReset(email string) error
	ResetPassword(token, newPassword string) error
}

// PasswordResetService handles password reset operations
type PasswordResetService struct {
	userRepo     repository.UserRepositoryInterface
	emailService email.EmailServiceInterface
	redisClient  *redis.Client
	resetURL     string // Frontend reset URL
}

// NewPasswordResetService creates a new password reset service
func NewPasswordResetService(
	userRepo repository.UserRepositoryInterface,
	emailService email.EmailServiceInterface,
	redisClient *redis.Client,
	resetURL string,
) *PasswordResetService {
	return &PasswordResetService{
		userRepo:     userRepo,
		emailService: emailService,
		redisClient:  redisClient,
		resetURL:     resetURL,
	}
}

// RequestPasswordReset initiates a password reset request
func (s *PasswordResetService) RequestPasswordReset(emailAddr string) error {
	// Validate email format
	if err := auth.ValidateEmail(emailAddr); err != nil {
		return err
	}

	// Check if user exists
	user, err := s.userRepo.GetByEmail(emailAddr)
	if err != nil {
		if errors.Is(err, repository.ErrUserNotFound) {
			// For security, don't reveal if email exists
			// Return success but don't send email
			log.Printf("Password reset requested for non-existent email: %s", emailAddr)
			return nil
		}
		return err
	}

	// Generate secure reset token
	resetToken, err := auth.GeneratePasswordResetToken()
	if err != nil {
		return fmt.Errorf("failed to generate reset token: %w", err)
	}

	// Store token in Redis with 1-hour expiry
	ctx := context.Background()
	key := ResetTokenPrefix + resetToken

	// Store user email as value
	err = s.redisClient.Set(ctx, key, user.Email, ResetTokenExpiry).Err()
	if err != nil {
		return fmt.Errorf("failed to store reset token: %w", err)
	}

	// Send reset email asynchronously
	go func() {
		if err := s.emailService.SendPasswordResetEmail(user.Email, resetToken, s.resetURL); err != nil {
			log.Printf("Failed to send password reset email to %s: %v", user.Email, err)
		}
	}()

	return nil
}

// ResetPassword resets the user's password using a valid token
func (s *PasswordResetService) ResetPassword(token, newPassword string) error {
	// Validate password strength
	if err := auth.ValidatePassword(newPassword); err != nil {
		return ErrWeakPassword
	}

	// Retrieve email from Redis using token
	ctx := context.Background()
	key := ResetTokenPrefix + token

	emailAddr, err := s.redisClient.Get(ctx, key).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return ErrInvalidResetToken
		}
		return fmt.Errorf("failed to retrieve reset token: %w", err)
	}

	// Get user by email
	user, err := s.userRepo.GetByEmail(emailAddr)
	if err != nil {
		if errors.Is(err, repository.ErrUserNotFound) {
			return ErrUserNotFoundForReset
		}
		return err
	}

	// Hash new password
	passwordHash, err := auth.HashPassword(newPassword)
	if err != nil {
		return fmt.Errorf("failed to hash password: %w", err)
	}

	// Update password using GORM repository if available
	if gormRepo, ok := s.userRepo.(repository.UserRepositoryGormInterface); ok {
		if err := gormRepo.UpdatePassword(user.ID, passwordHash); err != nil {
			return fmt.Errorf("failed to update password: %w", err)
		}
	} else {
		return errors.New("password update not supported by this repository")
	}

	// Delete the used token from Redis
	s.redisClient.Del(ctx, key)

	// Reset any failed login attempts and unlock account
	if gormRepo, ok := s.userRepo.(repository.UserRepositoryGormInterface); ok {
		_ = gormRepo.ResetFailedLoginCountByUUID(user.ID)
	}

	// Send confirmation email asynchronously
	go func() {
		if err := s.emailService.SendPasswordResetConfirmationEmail(user.Email); err != nil {
			log.Printf("Failed to send password reset confirmation email to %s: %v", user.Email, err)
		}
	}()

	return nil
}
