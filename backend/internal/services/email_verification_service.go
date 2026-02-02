package services

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"

	"github.com/javaknight1/servicepro/backend/internal/repository"
	"github.com/javaknight1/servicepro/backend/pkg/auth"
	"github.com/javaknight1/servicepro/backend/pkg/clients/email"
	"github.com/javaknight1/servicepro/backend/pkg/clients/logging"
)

const (
	// Verification token expiry duration (24 hours)
	VerificationTokenExpiry = 24 * time.Hour

	// Redis key prefix for verification tokens
	VerificationTokenPrefix = "email_verification:"

	// Reminder threshold (24 hours after sending)
	ReminderThreshold = 24 * time.Hour
)

var (
	ErrInvalidVerificationToken    = errors.New("invalid or expired verification token")
	ErrEmailAlreadyVerified        = errors.New("email already verified")
	ErrUserNotFoundForVerification = errors.New("user not found")
)

// EmailVerificationServiceInterface defines the interface for email verification operations
type EmailVerificationServiceInterface interface {
	SendVerificationEmail(userID uuid.UUID) error
	VerifyEmail(token string) error
	ResendVerificationEmail(email string) error
	SendReminderEmails() error
	HandleBounce(email string) error
}

// EmailVerificationService handles email verification operations
type EmailVerificationService struct {
	userRepo        repository.UserRepositoryInterface
	emailClient     email.Client
	redisClient     *redis.Client
	verificationURL string // Frontend verification URL
}

// NewEmailVerificationService creates a new email verification service
func NewEmailVerificationService(
	userRepo repository.UserRepositoryInterface,
	emailClient email.Client,
	redisClient *redis.Client,
	verificationURL string,
) *EmailVerificationService {
	return &EmailVerificationService{
		userRepo:        userRepo,
		emailClient:     emailClient,
		redisClient:     redisClient,
		verificationURL: verificationURL,
	}
}

// SendVerificationEmail sends a verification email to the user
func (s *EmailVerificationService) SendVerificationEmail(userID uuid.UUID) error {
	// Get user from repository
	gormRepo, ok := s.userRepo.(repository.UserRepositoryGormInterface)
	if !ok {
		return errors.New("GORM repository required for email verification")
	}

	user, err := gormRepo.GetByID(userID)
	if err != nil {
		if errors.Is(err, repository.ErrUserNotFound) {
			return ErrUserNotFoundForVerification
		}
		return err
	}

	// Check if email is already verified
	if user.EmailVerified {
		return ErrEmailAlreadyVerified
	}

	// Generate secure verification token
	verificationToken, err := auth.GenerateEmailVerificationToken()
	if err != nil {
		return fmt.Errorf("failed to generate verification token: %w", err)
	}

	// Store token in Redis with 24-hour expiry
	ctx := context.Background()
	key := VerificationTokenPrefix + verificationToken

	// Store user email as value
	err = s.redisClient.Set(ctx, key, user.Email, VerificationTokenExpiry).Err()
	if err != nil {
		return fmt.Errorf("failed to store verification token: %w", err)
	}

	// Update verification_sent_at timestamp
	now := time.Now()
	if err := gormRepo.UpdateVerificationSentAt(userID, &now); err != nil {
		logging.Error(ctx, "[EMAIL-VERIFICATION] Failed to update verification_sent_at for user", map[string]any{"user_id": userID, "error": err})
	}

	// Send verification email asynchronously
	go func() {
		bgCtx := context.Background()
		if err := s.emailClient.SendEmailVerificationEmail(bgCtx, user.Email, verificationToken, s.verificationURL); err != nil {
			logging.Error(bgCtx, "[EMAIL-VERIFICATION] Failed to send verification email", map[string]any{"email": user.Email, "error": err})
		}
	}()

	return nil
}

// VerifyEmail verifies a user's email using a valid token
func (s *EmailVerificationService) VerifyEmail(token string) error {
	// Retrieve email from Redis using token
	ctx := context.Background()
	key := VerificationTokenPrefix + token

	emailAddr, err := s.redisClient.Get(ctx, key).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return ErrInvalidVerificationToken
		}
		return fmt.Errorf("failed to retrieve verification token: %w", err)
	}

	// Get user by email
	user, err := s.userRepo.GetByEmail(emailAddr)
	if err != nil {
		if errors.Is(err, repository.ErrUserNotFound) {
			return ErrUserNotFoundForVerification
		}
		return err
	}

	// Check if already verified
	if user.EmailVerified {
		// Delete token and return success (idempotent)
		s.redisClient.Del(ctx, key)
		return nil
	}

	// Mark email as verified using GORM repository
	gormRepo, ok := s.userRepo.(repository.UserRepositoryGormInterface)
	if !ok {
		return errors.New("GORM repository required for email verification")
	}

	if err := gormRepo.MarkEmailVerified(user.ID); err != nil {
		return fmt.Errorf("failed to mark email as verified: %w", err)
	}

	// Delete the used token from Redis
	s.redisClient.Del(ctx, key)

	// Send confirmation email asynchronously
	go func() {
		bgCtx := context.Background()
		if err := s.emailClient.SendEmailVerificationSuccessEmail(bgCtx, user.Email); err != nil {
			logging.Error(bgCtx, "[EMAIL-VERIFICATION] Failed to send verification success email", map[string]any{"email": user.Email, "error": err})
		}
	}()

	return nil
}

// ResendVerificationEmail resends the verification email to a user
func (s *EmailVerificationService) ResendVerificationEmail(emailAddr string) error {
	// Validate email format
	if err := auth.ValidateEmail(emailAddr); err != nil {
		return err
	}

	// Get user by email
	user, err := s.userRepo.GetByEmail(emailAddr)
	if err != nil {
		if errors.Is(err, repository.ErrUserNotFound) {
			// For security, don't reveal if email exists
			// Return success but don't send email
			logging.Info(context.Background(), "[EMAIL-VERIFICATION] Verification resend requested for non-existent email", map[string]any{"email": emailAddr})
			return nil
		}
		return err
	}

	// Check if already verified
	if user.EmailVerified {
		return ErrEmailAlreadyVerified
	}

	// Send verification email using user ID
	return s.SendVerificationEmail(user.ID)
}

// SendReminderEmails sends reminder emails to users who haven't verified after 24 hours
func (s *EmailVerificationService) SendReminderEmails() error {
	gormRepo, ok := s.userRepo.(repository.UserRepositoryGormInterface)
	if !ok {
		return errors.New("GORM repository required for email verification")
	}

	ctx := context.Background()

	// Get unverified users who were sent verification 24+ hours ago
	users, err := gormRepo.GetUnverifiedUsersForReminder(ReminderThreshold)
	if err != nil {
		return fmt.Errorf("failed to get unverified users: %w", err)
	}

	logging.Info(ctx, "[EMAIL-VERIFICATION] Sending verification reminders", map[string]any{"user_count": len(users)})

	// Send reminder to each user
	for _, user := range users {
		// Generate new verification token
		verificationToken, err := auth.GenerateEmailVerificationToken()
		if err != nil {
			logging.Error(ctx, "[EMAIL-VERIFICATION] Failed to generate token for user", map[string]any{"user_id": user.ID, "error": err})
			continue
		}

		// Store token in Redis
		key := VerificationTokenPrefix + verificationToken
		err = s.redisClient.Set(ctx, key, user.Email, VerificationTokenExpiry).Err()
		if err != nil {
			logging.Error(ctx, "[EMAIL-VERIFICATION] Failed to store verification token for user", map[string]any{"user_id": user.ID, "error": err})
			continue
		}

		// Update verification_sent_at
		now := time.Now()
		if err := gormRepo.UpdateVerificationSentAt(user.ID, &now); err != nil {
			logging.Error(ctx, "[EMAIL-VERIFICATION] Failed to update verification_sent_at for user", map[string]any{"user_id": user.ID, "error": err})
		}

		// Send reminder email (not async for batch processing)
		if err := s.emailClient.SendEmailVerificationReminderEmail(ctx, user.Email, verificationToken, s.verificationURL); err != nil {
			logging.Error(ctx, "[EMAIL-VERIFICATION] Failed to send verification reminder", map[string]any{"email": user.Email, "error": err})
		} else {
			logging.Info(ctx, "[EMAIL-VERIFICATION] Sent verification reminder", map[string]any{"email": user.Email})
		}

		// Add small delay to avoid overwhelming email service
		time.Sleep(100 * time.Millisecond)
	}

	return nil
}

// HandleBounce handles AWS SES bounce notifications
func (s *EmailVerificationService) HandleBounce(emailAddr string) error {
	ctx := context.Background()
	logging.Info(ctx, "[EMAIL-VERIFICATION] Processing bounce for email", map[string]any{"email": emailAddr})

	// Get user by email
	user, err := s.userRepo.GetByEmail(emailAddr)
	if err != nil {
		if errors.Is(err, repository.ErrUserNotFound) {
			// Email might have been deleted
			logging.Info(ctx, "[EMAIL-VERIFICATION] Bounce received for non-existent user", map[string]any{"email": emailAddr})
			return nil
		}
		return err
	}

	// If email was not verified, we should take action
	if !user.EmailVerified {
		gormRepo, ok := s.userRepo.(repository.UserRepositoryGormInterface)
		if !ok {
			return errors.New("GORM repository required for bounce handling")
		}

		// Mark bounce in database (we could add a bounced field later)
		// For now, just log it
		logging.Warn(ctx, "[EMAIL-VERIFICATION] Hard bounce received for unverified user", map[string]any{"user_id": user.ID, "email": emailAddr})

		// Could implement:
		// - Flag account as having invalid email
		// - Prevent further email sends
		// - Require manual verification
		// - Delete account after X days

		// Update verification_sent_at to null to prevent reminder sends
		if err := gormRepo.UpdateVerificationSentAt(user.ID, nil); err != nil {
			logging.Error(ctx, "[EMAIL-VERIFICATION] Failed to clear verification_sent_at for bounced user", map[string]any{"user_id": user.ID, "error": err})
		}
	}

	return nil
}
