package services

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/javaknight1/servicepro/backend/internal/models"
	"github.com/javaknight1/servicepro/backend/internal/repository"
)

func TestSendVerificationEmail_Success(t *testing.T) {
	mockRepo := new(MockUserRepositoryGorm)
	mockEmail := new(MockEmailService)
	redisClient, _ := setupTestRedis(t)
	defer redisClient.Close()

	service := NewEmailVerificationService(
		mockRepo,
		mockEmail,
		redisClient,
		"http://localhost:5173/verify-email",
	)

	userID := uuid.New()
	user := &models.User{
		ID:            userID,
		Email:         "test@example.com",
		EmailVerified: false,
	}

	// Setup expectations
	mockRepo.On("GetByID", userID).Return(user, nil)
	mockRepo.On("UpdateVerificationSentAt", userID, mock.AnythingOfType("*time.Time")).Return(nil)
	mockEmail.On("SendEmailVerificationEmail", user.Email, mock.AnythingOfType("string"), "http://localhost:5173/verify-email").Return(nil)

	// Execute
	err := service.SendVerificationEmail(userID)

	// Assert
	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)

	// Verify token was stored in Redis
	keys, _ := redisClient.Keys(context.Background(), "email_verification:*").Result()
	assert.Equal(t, 1, len(keys), "Should have 1 verification token in Redis")
}

func TestSendVerificationEmail_UserNotFound(t *testing.T) {
	mockRepo := new(MockUserRepositoryGorm)
	mockEmail := new(MockEmailService)
	redisClient, _ := setupTestRedis(t)
	defer redisClient.Close()

	service := NewEmailVerificationService(
		mockRepo,
		mockEmail,
		redisClient,
		"http://localhost:5173/verify-email",
	)

	userID := uuid.New()

	// Setup expectations
	mockRepo.On("GetByID", userID).Return(nil, repository.ErrUserNotFound)

	// Execute
	err := service.SendVerificationEmail(userID)

	// Assert
	assert.Error(t, err)
	assert.Equal(t, ErrUserNotFoundForVerification, err)
	mockRepo.AssertExpectations(t)
}

func TestSendVerificationEmail_AlreadyVerified(t *testing.T) {
	mockRepo := new(MockUserRepositoryGorm)
	mockEmail := new(MockEmailService)
	redisClient, _ := setupTestRedis(t)
	defer redisClient.Close()

	service := NewEmailVerificationService(
		mockRepo,
		mockEmail,
		redisClient,
		"http://localhost:5173/verify-email",
	)

	userID := uuid.New()
	user := &models.User{
		ID:            userID,
		Email:         "test@example.com",
		EmailVerified: true, // Already verified
	}

	// Setup expectations
	mockRepo.On("GetByID", userID).Return(user, nil)

	// Execute
	err := service.SendVerificationEmail(userID)

	// Assert
	assert.Error(t, err)
	assert.Equal(t, ErrEmailAlreadyVerified, err)
	mockRepo.AssertExpectations(t)
}

func TestVerifyEmail_Success(t *testing.T) {
	mockRepo := new(MockUserRepositoryGorm)
	mockEmail := new(MockEmailService)
	redisClient, _ := setupTestRedis(t)
	defer redisClient.Close()

	service := NewEmailVerificationService(
		mockRepo,
		mockEmail,
		redisClient,
		"http://localhost:5173/verify-email",
	)

	token := "valid-token-123"
	email := "test@example.com"
	userID := uuid.New()
	user := &models.User{
		ID:            userID,
		Email:         email,
		EmailVerified: false,
	}

	// Store token in Redis
	ctx := context.Background()
	key := VerificationTokenPrefix + token
	redisClient.Set(ctx, key, email, VerificationTokenExpiry)

	// Setup expectations
	mockRepo.On("GetByEmail", email).Return(user, nil)
	mockRepo.On("MarkEmailVerified", userID).Return(nil)
	mockEmail.On("SendEmailVerificationSuccessEmail", email).Return(nil)

	// Execute
	err := service.VerifyEmail(token)

	// Assert
	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)

	// Verify token was deleted from Redis
	_, err = redisClient.Get(ctx, key).Result()
	assert.ErrorIs(t, err, redis.Nil, "Token should be deleted after verification")
}

func TestVerifyEmail_InvalidToken(t *testing.T) {
	mockRepo := new(MockUserRepositoryGorm)
	mockEmail := new(MockEmailService)
	redisClient, _ := setupTestRedis(t)
	defer redisClient.Close()

	service := NewEmailVerificationService(
		mockRepo,
		mockEmail,
		redisClient,
		"http://localhost:5173/verify-email",
	)

	token := "invalid-token"

	// Execute - token not in Redis
	err := service.VerifyEmail(token)

	// Assert
	assert.Error(t, err)
	assert.Equal(t, ErrInvalidVerificationToken, err)
}

func TestVerifyEmail_Idempotent(t *testing.T) {
	mockRepo := new(MockUserRepositoryGorm)
	mockEmail := new(MockEmailService)
	redisClient, _ := setupTestRedis(t)
	defer redisClient.Close()

	service := NewEmailVerificationService(
		mockRepo,
		mockEmail,
		redisClient,
		"http://localhost:5173/verify-email",
	)

	token := "valid-token-123"
	email := "test@example.com"
	userID := uuid.New()
	user := &models.User{
		ID:            userID,
		Email:         email,
		EmailVerified: true, // Already verified
	}

	// Store token in Redis
	ctx := context.Background()
	key := VerificationTokenPrefix + token
	redisClient.Set(ctx, key, email, VerificationTokenExpiry)

	// Setup expectations
	mockRepo.On("GetByEmail", email).Return(user, nil)

	// Execute
	err := service.VerifyEmail(token)

	// Assert - should succeed without error (idempotent)
	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)

	// Verify token was deleted from Redis
	_, err = redisClient.Get(ctx, key).Result()
	assert.ErrorIs(t, err, redis.Nil, "Token should be deleted even when already verified")
}

func TestResendVerificationEmail_Success(t *testing.T) {
	mockRepo := new(MockUserRepositoryGorm)
	mockEmail := new(MockEmailService)
	redisClient, _ := setupTestRedis(t)
	defer redisClient.Close()

	service := NewEmailVerificationService(
		mockRepo,
		mockEmail,
		redisClient,
		"http://localhost:5173/verify-email",
	)

	email := "test@example.com"
	userID := uuid.New()
	user := &models.User{
		ID:            userID,
		Email:         email,
		EmailVerified: false,
	}

	// Setup expectations
	// ResendVerificationEmail calls GetByEmail first, then calls SendVerificationEmail which calls GetByID
	mockRepo.On("GetByEmail", email).Return(user, nil)
	mockRepo.On("GetByID", userID).Return(user, nil) // SendVerificationEmail internally calls GetByID
	mockRepo.On("UpdateVerificationSentAt", userID, mock.AnythingOfType("*time.Time")).Return(nil)
	mockEmail.On("SendEmailVerificationEmail", email, mock.AnythingOfType("string"), "http://localhost:5173/verify-email").Return(nil)

	// Execute
	err := service.ResendVerificationEmail(email)

	// Assert
	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)

	// Verify token was stored in Redis
	keys, _ := redisClient.Keys(context.Background(), "email_verification:*").Result()
	assert.Equal(t, 1, len(keys), "Should have 1 verification token in Redis")
}

func TestResendVerificationEmail_AlreadyVerified(t *testing.T) {
	mockRepo := new(MockUserRepositoryGorm)
	mockEmail := new(MockEmailService)
	redisClient, _ := setupTestRedis(t)
	defer redisClient.Close()

	service := NewEmailVerificationService(
		mockRepo,
		mockEmail,
		redisClient,
		"http://localhost:5173/verify-email",
	)

	email := "test@example.com"
	userID := uuid.New()
	user := &models.User{
		ID:            userID,
		Email:         email,
		EmailVerified: true, // Already verified
	}

	// Setup expectations
	mockRepo.On("GetByEmail", email).Return(user, nil)

	// Execute
	err := service.ResendVerificationEmail(email)

	// Assert
	assert.Error(t, err)
	assert.Equal(t, ErrEmailAlreadyVerified, err)
	mockRepo.AssertExpectations(t)
}

func TestResendVerificationEmail_UserNotFound(t *testing.T) {
	mockRepo := new(MockUserRepositoryGorm)
	mockEmail := new(MockEmailService)
	redisClient, _ := setupTestRedis(t)
	defer redisClient.Close()

	service := NewEmailVerificationService(
		mockRepo,
		mockEmail,
		redisClient,
		"http://localhost:5173/verify-email",
	)

	email := "nonexistent@example.com"

	// Setup expectations
	mockRepo.On("GetByEmail", email).Return(nil, repository.ErrUserNotFound)

	// Execute - should succeed silently for security (don't reveal if email exists)
	err := service.ResendVerificationEmail(email)

	// Assert - no error (security feature)
	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

func TestSendReminderEmails_Success(t *testing.T) {
	mockRepo := new(MockUserRepositoryGorm)
	mockEmail := new(MockEmailService)
	redisClient, _ := setupTestRedis(t)
	defer redisClient.Close()

	service := NewEmailVerificationService(
		mockRepo,
		mockEmail,
		redisClient,
		"http://localhost:5173/verify-email",
	)

	user1 := &models.User{
		ID:            uuid.New(),
		Email:         "user1@example.com",
		EmailVerified: false,
	}
	user2 := &models.User{
		ID:            uuid.New(),
		Email:         "user2@example.com",
		EmailVerified: false,
	}
	users := []*models.User{user1, user2}

	// Setup expectations
	mockRepo.On("GetUnverifiedUsersForReminder", ReminderThreshold).Return(users, nil)
	mockRepo.On("UpdateVerificationSentAt", user1.ID, mock.AnythingOfType("*time.Time")).Return(nil)
	mockRepo.On("UpdateVerificationSentAt", user2.ID, mock.AnythingOfType("*time.Time")).Return(nil)
	mockEmail.On("SendEmailVerificationReminderEmail", user1.Email, mock.AnythingOfType("string"), "http://localhost:5173/verify-email").Return(nil)
	mockEmail.On("SendEmailVerificationReminderEmail", user2.Email, mock.AnythingOfType("string"), "http://localhost:5173/verify-email").Return(nil)

	// Execute
	err := service.SendReminderEmails()

	// Assert
	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)

	// Verify tokens were stored in Redis
	keys, _ := redisClient.Keys(context.Background(), "email_verification:*").Result()
	assert.Equal(t, 2, len(keys), "Should have 2 verification tokens in Redis")
}

func TestSendReminderEmails_NoUsers(t *testing.T) {
	mockRepo := new(MockUserRepositoryGorm)
	mockEmail := new(MockEmailService)
	redisClient, _ := setupTestRedis(t)
	defer redisClient.Close()

	service := NewEmailVerificationService(
		mockRepo,
		mockEmail,
		redisClient,
		"http://localhost:5173/verify-email",
	)

	// Setup expectations - no users to remind
	mockRepo.On("GetUnverifiedUsersForReminder", ReminderThreshold).Return([]*models.User{}, nil)

	// Execute
	err := service.SendReminderEmails()

	// Assert
	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)

	// Verify no tokens in Redis
	keys, _ := redisClient.Keys(context.Background(), "email_verification:*").Result()
	assert.Equal(t, 0, len(keys), "Should have no tokens in Redis")
}

func TestHandleBounce_Success(t *testing.T) {
	mockRepo := new(MockUserRepositoryGorm)
	mockEmail := new(MockEmailService)
	redisClient, _ := setupTestRedis(t)
	defer redisClient.Close()

	service := NewEmailVerificationService(
		mockRepo,
		mockEmail,
		redisClient,
		"http://localhost:5173/verify-email",
	)

	email := "bounced@example.com"
	userID := uuid.New()
	user := &models.User{
		ID:            userID,
		Email:         email,
		EmailVerified: false,
	}

	// Setup expectations
	mockRepo.On("GetByEmail", email).Return(user, nil)
	// HandleBounce clears verification_sent_at to prevent reminder sends
	mockRepo.On("UpdateVerificationSentAt", userID, (*time.Time)(nil)).Return(nil)

	// Execute
	err := service.HandleBounce(email)

	// Assert
	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

func TestHandleBounce_UserNotFound(t *testing.T) {
	mockRepo := new(MockUserRepositoryGorm)
	mockEmail := new(MockEmailService)
	redisClient, _ := setupTestRedis(t)
	defer redisClient.Close()

	service := NewEmailVerificationService(
		mockRepo,
		mockEmail,
		redisClient,
		"http://localhost:5173/verify-email",
	)

	email := "nonexistent@example.com"

	// Setup expectations
	mockRepo.On("GetByEmail", email).Return(nil, repository.ErrUserNotFound)

	// Execute - should succeed silently
	err := service.HandleBounce(email)

	// Assert - no error
	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}
