package services

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/javaknight1/servicepro/backend/internal/models"
	"github.com/javaknight1/servicepro/backend/internal/repository"
	"github.com/javaknight1/servicepro/backend/pkg/auth"
)

// MockUserRepositoryGorm is a mock implementation of UserRepositoryGormInterface
type MockUserRepositoryGorm struct {
	mock.Mock
}

// Base UserRepositoryInterface methods
func (m *MockUserRepositoryGorm) GetByEmail(email string) (*models.User, error) {
	args := m.Called(email)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.User), args.Error(1)
}

func (m *MockUserRepositoryGorm) Create(email, passwordHash string) (*models.User, error) {
	args := m.Called(email, passwordHash)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.User), args.Error(1)
}

func (m *MockUserRepositoryGorm) IncrementFailedLoginCount(userID int) error {
	args := m.Called(userID)
	return args.Error(0)
}

func (m *MockUserRepositoryGorm) LockAccount(userID int, until time.Time) error {
	args := m.Called(userID, until)
	return args.Error(0)
}

func (m *MockUserRepositoryGorm) ResetFailedLoginCount(userID int) error {
	args := m.Called(userID)
	return args.Error(0)
}

// GORM-specific methods
func (m *MockUserRepositoryGorm) GetByID(id uuid.UUID) (*models.User, error) {
	args := m.Called(id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.User), args.Error(1)
}

func (m *MockUserRepositoryGorm) EmailExists(email string) (bool, error) {
	args := m.Called(email)
	return args.Bool(0), args.Error(1)
}

func (m *MockUserRepositoryGorm) UpdatePassword(userID uuid.UUID, passwordHash string) error {
	args := m.Called(userID, passwordHash)
	return args.Error(0)
}

func (m *MockUserRepositoryGorm) IncrementFailedLoginCountByUUID(userID uuid.UUID) error {
	args := m.Called(userID)
	return args.Error(0)
}

func (m *MockUserRepositoryGorm) ResetFailedLoginCountByUUID(userID uuid.UUID) error {
	args := m.Called(userID)
	return args.Error(0)
}

func (m *MockUserRepositoryGorm) LockAccountByUUID(userID uuid.UUID, lockUntil time.Time) error {
	args := m.Called(userID, lockUntil)
	return args.Error(0)
}

// Email verification methods
func (m *MockUserRepositoryGorm) MarkEmailVerified(userID uuid.UUID) error {
	args := m.Called(userID)
	return args.Error(0)
}

func (m *MockUserRepositoryGorm) UpdateVerificationSentAt(userID uuid.UUID, sentAt *time.Time) error {
	args := m.Called(userID, sentAt)
	return args.Error(0)
}

func (m *MockUserRepositoryGorm) GetUnverifiedUsersForReminder(threshold time.Duration) ([]*models.User, error) {
	args := m.Called(threshold)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.User), args.Error(1)
}

func (m *MockUserRepositoryGorm) UpdateProfilePicture(userID uuid.UUID, url *string) error {
	args := m.Called(userID, url)
	return args.Error(0)
}

func (m *MockUserRepositoryGorm) UpdateProfile(userID uuid.UUID, firstName, lastName *string) error {
	args := m.Called(userID, firstName, lastName)
	return args.Error(0)
}

// setupTestRedis creates a miniredis server and returns a configured redis client
func setupTestRedis(t *testing.T) (*redis.Client, *miniredis.Miniredis) {
	mr := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})
	return client, mr
}

func TestRequestPasswordReset_ValidEmail_UserExists(t *testing.T) {
	// Setup
	mockUserRepo := new(MockUserRepositoryGorm)
	mockEmailService := new(MockEmailService)
	redisClient, _ := setupTestRedis(t)
	defer redisClient.Close()

	testUser := &models.User{
		ID:    uuid.MustParse("550e8400-e29b-41d4-a716-446655440000"),
		Email: "test@example.com",
	}

	mockUserRepo.On("GetByEmail", "test@example.com").Return(testUser, nil)
	mockEmailService.On("SendPasswordResetEmail", mock.Anything, "test@example.com", mock.AnythingOfType("string"), "http://localhost:5173/reset-password").Return(nil)

	service := NewPasswordResetService(mockUserRepo, mockEmailService, redisClient, "http://localhost:5173/reset-password")

	// Execute
	err := service.RequestPasswordReset("test@example.com")

	// Assert
	assert.NoError(t, err)
	mockUserRepo.AssertExpectations(t)

	// Wait a bit for async email to be sent
	time.Sleep(100 * time.Millisecond)
	mockEmailService.AssertExpectations(t)

	// Verify token was stored in Redis
	keys, _ := redisClient.Keys(context.Background(), "password_reset:*").Result()
	assert.Equal(t, 1, len(keys), "Should have 1 reset token in Redis")

	// Verify token has correct expiry
	if len(keys) > 0 {
		ttl, _ := redisClient.TTL(context.Background(), keys[0]).Result()
		assert.Greater(t, ttl.Seconds(), float64(3500), "Token should have ~1 hour expiry")
		assert.LessOrEqual(t, ttl.Seconds(), float64(3600), "Token should not exceed 1 hour expiry")
	}
}

func TestRequestPasswordReset_UserNotFound(t *testing.T) {
	// Setup
	mockUserRepo := new(MockUserRepositoryGorm)
	mockEmailService := new(MockEmailService)
	redisClient, _ := setupTestRedis(t)
	defer redisClient.Close()

	mockUserRepo.On("GetByEmail", "nonexistent@example.com").Return(nil, repository.ErrUserNotFound)

	service := NewPasswordResetService(mockUserRepo, mockEmailService, redisClient, "http://localhost:5173/reset-password")

	// Execute
	err := service.RequestPasswordReset("nonexistent@example.com")

	// Assert - should return nil for security (don't reveal if email exists)
	assert.NoError(t, err)
	mockUserRepo.AssertExpectations(t)

	// Email should NOT be called
	mockEmailService.AssertNotCalled(t, "SendPasswordResetEmail")

	// Verify no token was stored in Redis
	keys, _ := redisClient.Keys(context.Background(), "password_reset:*").Result()
	assert.Equal(t, 0, len(keys), "Should have no reset tokens for non-existent user")
}

func TestRequestPasswordReset_InvalidEmail(t *testing.T) {
	// Setup
	mockUserRepo := new(MockUserRepositoryGorm)
	mockEmailService := new(MockEmailService)
	redisClient, _ := setupTestRedis(t)
	defer redisClient.Close()

	service := NewPasswordResetService(mockUserRepo, mockEmailService, redisClient, "http://localhost:5173/reset-password")

	// Execute
	err := service.RequestPasswordReset("invalid-email")

	// Assert
	assert.Error(t, err)
	assert.ErrorIs(t, err, auth.ErrInvalidEmailFormat)

	// Repository and email service should not be called
	mockUserRepo.AssertNotCalled(t, "GetByEmail")
	mockEmailService.AssertNotCalled(t, "SendPasswordResetEmail")
}

func TestRequestPasswordReset_DatabaseError(t *testing.T) {
	// Setup
	mockUserRepo := new(MockUserRepositoryGorm)
	mockEmailService := new(MockEmailService)
	redisClient, _ := setupTestRedis(t)
	defer redisClient.Close()

	dbError := errors.New("database connection error")
	mockUserRepo.On("GetByEmail", "test@example.com").Return(nil, dbError)

	service := NewPasswordResetService(mockUserRepo, mockEmailService, redisClient, "http://localhost:5173/reset-password")

	// Execute
	err := service.RequestPasswordReset("test@example.com")

	// Assert
	assert.Error(t, err)
	assert.Equal(t, dbError, err)
	mockUserRepo.AssertExpectations(t)
}

func TestResetPassword_ValidToken_StrongPassword(t *testing.T) {
	// Setup
	mockUserRepo := new(MockUserRepositoryGorm)
	mockEmailService := new(MockEmailService)
	redisClient, _ := setupTestRedis(t)
	defer redisClient.Close()

	testUser := &models.User{
		ID:    uuid.MustParse("550e8400-e29b-41d4-a716-446655440000"),
		Email: "test@example.com",
	}

	// Store a test token in Redis
	testToken := "test-reset-token-123"
	ctx := context.Background()
	redisClient.Set(ctx, ResetTokenPrefix+testToken, testUser.Email, ResetTokenExpiry)

	mockUserRepo.On("GetByEmail", "test@example.com").Return(testUser, nil)
	mockUserRepo.On("UpdatePassword", testUser.ID, mock.AnythingOfType("string")).Return(nil)
	mockUserRepo.On("ResetFailedLoginCountByUUID", testUser.ID).Return(nil)
	mockEmailService.On("SendPasswordResetConfirmationEmail", mock.Anything, "test@example.com").Return(nil)

	service := NewPasswordResetService(mockUserRepo, mockEmailService, redisClient, "http://localhost:5173/reset-password")

	// Execute
	err := service.ResetPassword(testToken, "NewStrongP@ss123")

	// Assert
	assert.NoError(t, err)
	mockUserRepo.AssertExpectations(t)

	// Wait for async email
	time.Sleep(100 * time.Millisecond)
	mockEmailService.AssertExpectations(t)

	// Verify token was deleted from Redis
	_, err = redisClient.Get(ctx, ResetTokenPrefix+testToken).Result()
	assert.ErrorIs(t, err, redis.Nil, "Token should be deleted after successful reset")
}

func TestResetPassword_InvalidToken(t *testing.T) {
	// Setup
	mockUserRepo := new(MockUserRepositoryGorm)
	mockEmailService := new(MockEmailService)
	redisClient, _ := setupTestRedis(t)
	defer redisClient.Close()

	service := NewPasswordResetService(mockUserRepo, mockEmailService, redisClient, "http://localhost:5173/reset-password")

	// Execute with non-existent token
	err := service.ResetPassword("invalid-token", "NewStrongP@ss123")

	// Assert
	assert.Error(t, err)
	assert.ErrorIs(t, err, ErrInvalidResetToken)

	// No repository or email calls should be made
	mockUserRepo.AssertNotCalled(t, "GetByEmail")
	mockUserRepo.AssertNotCalled(t, "UpdatePassword")
	mockEmailService.AssertNotCalled(t, "SendPasswordResetConfirmationEmail")
}

func TestResetPassword_WeakPassword(t *testing.T) {
	testCases := []struct {
		name     string
		password string
	}{
		{"too short", "Short1!"},
		{"no uppercase", "weakpassword123!"},
		{"no lowercase", "WEAKPASSWORD123!"},
		{"no number", "WeakPassword!"},
		{"no special", "WeakPassword123"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Setup
			mockUserRepo := new(MockUserRepositoryGorm)
			mockEmailService := new(MockEmailService)
			redisClient, _ := setupTestRedis(t)
			defer redisClient.Close()

			// Store a test token in Redis
			testToken := "test-reset-token-" + tc.name
			ctx := context.Background()
			redisClient.Set(ctx, ResetTokenPrefix+testToken, "test@example.com", ResetTokenExpiry)

			service := NewPasswordResetService(mockUserRepo, mockEmailService, redisClient, "http://localhost:5173/reset-password")

			// Execute
			err := service.ResetPassword(testToken, tc.password)

			// Assert
			assert.Error(t, err)
			assert.ErrorIs(t, err, ErrWeakPassword)

			// Token should still exist in Redis (not deleted on validation failure)
			val, err := redisClient.Get(ctx, ResetTokenPrefix+testToken).Result()
			assert.NoError(t, err, "Token should still exist after password validation failure")
			assert.Equal(t, "test@example.com", val)

			// No repository calls should be made
			mockUserRepo.AssertNotCalled(t, "UpdatePassword")
			mockEmailService.AssertNotCalled(t, "SendPasswordResetConfirmationEmail")
		})
	}
}

func TestResetPassword_UserNotFound(t *testing.T) {
	// Setup
	mockUserRepo := new(MockUserRepositoryGorm)
	mockEmailService := new(MockEmailService)
	redisClient, _ := setupTestRedis(t)
	defer redisClient.Close()

	// Store a test token in Redis
	testToken := "test-reset-token-123"
	ctx := context.Background()
	redisClient.Set(ctx, ResetTokenPrefix+testToken, "deleted@example.com", ResetTokenExpiry)

	mockUserRepo.On("GetByEmail", "deleted@example.com").Return(nil, repository.ErrUserNotFound)

	service := NewPasswordResetService(mockUserRepo, mockEmailService, redisClient, "http://localhost:5173/reset-password")

	// Execute
	err := service.ResetPassword(testToken, "NewStrongP@ss123")

	// Assert
	assert.Error(t, err)
	assert.ErrorIs(t, err, ErrUserNotFoundForReset)
	mockUserRepo.AssertExpectations(t)

	// No password update should be attempted
	mockUserRepo.AssertNotCalled(t, "UpdatePassword")
	mockEmailService.AssertNotCalled(t, "SendPasswordResetConfirmationEmail")
}

func TestResetPassword_PasswordUpdateFailure(t *testing.T) {
	// Setup
	mockUserRepo := new(MockUserRepositoryGorm)
	mockEmailService := new(MockEmailService)
	redisClient, _ := setupTestRedis(t)
	defer redisClient.Close()

	testUser := &models.User{
		ID:    uuid.MustParse("550e8400-e29b-41d4-a716-446655440000"),
		Email: "test@example.com",
	}

	// Store a test token in Redis
	testToken := "test-reset-token-123"
	ctx := context.Background()
	redisClient.Set(ctx, ResetTokenPrefix+testToken, testUser.Email, ResetTokenExpiry)

	dbError := errors.New("database update failed")
	mockUserRepo.On("GetByEmail", "test@example.com").Return(testUser, nil)
	mockUserRepo.On("UpdatePassword", testUser.ID, mock.AnythingOfType("string")).Return(dbError)

	service := NewPasswordResetService(mockUserRepo, mockEmailService, redisClient, "http://localhost:5173/reset-password")

	// Execute
	err := service.ResetPassword(testToken, "NewStrongP@ss123")

	// Assert
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to update password")
	mockUserRepo.AssertExpectations(t)

	// Confirmation email should not be sent
	mockEmailService.AssertNotCalled(t, "SendPasswordResetConfirmationEmail")
}

func TestResetPassword_TokenExpiry(t *testing.T) {
	// Setup
	mockUserRepo := new(MockUserRepositoryGorm)
	mockEmailService := new(MockEmailService)
	redisClient, mr := setupTestRedis(t)
	defer redisClient.Close()

	// Store a test token with short expiry
	testToken := "test-reset-token-expiry"
	ctx := context.Background()
	redisClient.Set(ctx, ResetTokenPrefix+testToken, "test@example.com", 5*time.Second)

	// Fast forward time to expire the token
	mr.FastForward(6 * time.Second)

	service := NewPasswordResetService(mockUserRepo, mockEmailService, redisClient, "http://localhost:5173/reset-password")

	// Execute
	err := service.ResetPassword(testToken, "NewStrongP@ss123")

	// Assert
	assert.Error(t, err)
	assert.ErrorIs(t, err, ErrInvalidResetToken, "Expired token should be treated as invalid")
}
