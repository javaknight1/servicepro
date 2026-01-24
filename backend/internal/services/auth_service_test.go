package services

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/javaknight1/servicepro/backend/config"
	"github.com/javaknight1/servicepro/backend/internal/models"
	"github.com/javaknight1/servicepro/backend/internal/repository"
	"github.com/javaknight1/servicepro/backend/pkg/auth"
)

// MockUserRepository is a mock implementation of UserRepository
type MockUserRepository struct {
	mock.Mock
}

func (m *MockUserRepository) GetByEmail(email string) (*models.User, error) {
	args := m.Called(email)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.User), args.Error(1)
}

func (m *MockUserRepository) IncrementFailedLoginCount(userID int) error {
	args := m.Called(userID)
	return args.Error(0)
}

func (m *MockUserRepository) LockAccount(userID int, until time.Time) error {
	args := m.Called(userID, until)
	return args.Error(0)
}

func (m *MockUserRepository) ResetFailedLoginCount(userID int) error {
	args := m.Called(userID)
	return args.Error(0)
}

func (m *MockUserRepository) Create(email, passwordHash string) (*models.User, error) {
	args := m.Called(email, passwordHash)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.User), args.Error(1)
}

// UserRepositoryGormInterface methods
func (m *MockUserRepository) GetByID(id uuid.UUID) (*models.User, error) {
	args := m.Called(id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.User), args.Error(1)
}

func (m *MockUserRepository) IncrementFailedLoginCountByUUID(userID uuid.UUID) error {
	args := m.Called(userID)
	return args.Error(0)
}

func (m *MockUserRepository) LockAccountByUUID(userID uuid.UUID, until time.Time) error {
	args := m.Called(userID, until)
	return args.Error(0)
}

func (m *MockUserRepository) ResetFailedLoginCountByUUID(userID uuid.UUID) error {
	args := m.Called(userID)
	return args.Error(0)
}

func (m *MockUserRepository) EmailExists(email string) (bool, error) {
	args := m.Called(email)
	return args.Bool(0), args.Error(1)
}

func (m *MockUserRepository) UpdatePassword(userID uuid.UUID, passwordHash string) error {
	args := m.Called(userID, passwordHash)
	return args.Error(0)
}

func (m *MockUserRepository) MarkEmailVerified(userID uuid.UUID) error {
	args := m.Called(userID)
	return args.Error(0)
}

func (m *MockUserRepository) UpdateVerificationSentAt(userID uuid.UUID, sentAt *time.Time) error {
	args := m.Called(userID, sentAt)
	return args.Error(0)
}

func (m *MockUserRepository) GetUnverifiedUsersForReminder(threshold time.Duration) ([]*models.User, error) {
	args := m.Called(threshold)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.User), args.Error(1)
}

func (m *MockUserRepository) UpdateProfilePicture(userID uuid.UUID, url *string) error {
	args := m.Called(userID, url)
	return args.Error(0)
}

func getTestAuthConfig() *config.AuthConfig {
	return &config.AuthConfig{
		MaxLoginAttempts:  5,
		LockoutDuration:   time.Minute * 30,
		RateLimitWindow:   time.Minute * 15,
		RateLimitAttempts: 5,
	}
}

func getTestJWTConfig() *config.JWTConfig {
	return &config.JWTConfig{
		Secret:            "test-secret",
		AccessExpiration:  time.Hour,
		RefreshExpiration: time.Hour * 24 * 7,
	}
}

func TestLogin_Success(t *testing.T) {
	mockRepo := new(MockUserRepository)
	jwtManager := auth.NewJWTManager(getTestJWTConfig())
	authService := NewAuthService(mockRepo, jwtManager, getTestAuthConfig())

	password := "SecurePassword123!"
	passwordHash, _ := auth.HashPassword(password)
	userID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")

	user := &models.User{
		ID:               userID,
		Email:            "test@example.com",
		PasswordHash:     passwordHash,
		FailedLoginCount: 0,
		EmailVerified:    true,
	}

	mockRepo.On("GetByEmail", "test@example.com").Return(user, nil)
	mockRepo.On("ResetFailedLoginCountByUUID", userID).Return(nil)

	response, err := authService.Login("test@example.com", password)

	assert.NoError(t, err)
	assert.NotNil(t, response)
	assert.NotEmpty(t, response.Token)
	assert.NotEmpty(t, response.RefreshToken)
	assert.Equal(t, 3600, response.ExpiresIn)

	mockRepo.AssertExpectations(t)
}

func TestLogin_InvalidCredentials_UserNotFound(t *testing.T) {
	mockRepo := new(MockUserRepository)
	jwtManager := auth.NewJWTManager(getTestJWTConfig())
	authService := NewAuthService(mockRepo, jwtManager, getTestAuthConfig())

	mockRepo.On("GetByEmail", "notfound@example.com").Return(nil, repository.ErrUserNotFound)

	response, err := authService.Login("notfound@example.com", "password")

	assert.Error(t, err)
	assert.Nil(t, response)
	assert.Equal(t, ErrInvalidCredentials, err)

	mockRepo.AssertExpectations(t)
}

func TestLogin_InvalidCredentials_WrongPassword(t *testing.T) {
	mockRepo := new(MockUserRepository)
	jwtManager := auth.NewJWTManager(getTestJWTConfig())
	authService := NewAuthService(mockRepo, jwtManager, getTestAuthConfig())

	password := "CorrectPassword123!"
	passwordHash, _ := auth.HashPassword(password)
	userID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")

	user := &models.User{
		ID:               userID,
		Email:            "test@example.com",
		PasswordHash:     passwordHash,
		FailedLoginCount: 0,
		EmailVerified:    true,
	}

	mockRepo.On("GetByEmail", "test@example.com").Return(user, nil)
	mockRepo.On("IncrementFailedLoginCountByUUID", userID).Return(nil)

	response, err := authService.Login("test@example.com", "WrongPassword")

	assert.Error(t, err)
	assert.Nil(t, response)
	assert.Equal(t, ErrInvalidCredentials, err)

	mockRepo.AssertExpectations(t)
}

func TestLogin_AccountLocked(t *testing.T) {
	mockRepo := new(MockUserRepository)
	jwtManager := auth.NewJWTManager(getTestJWTConfig())
	authService := NewAuthService(mockRepo, jwtManager, getTestAuthConfig())

	password := "SecurePassword123!"
	passwordHash, _ := auth.HashPassword(password)
	lockedUntil := time.Now().Add(time.Minute * 30)
	userID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")

	user := &models.User{
		ID:               userID,
		Email:            "test@example.com",
		PasswordHash:     passwordHash,
		FailedLoginCount: 5,
		LockedUntil:      &lockedUntil,
	}

	mockRepo.On("GetByEmail", "test@example.com").Return(user, nil)

	response, err := authService.Login("test@example.com", password)

	assert.Error(t, err)
	assert.Nil(t, response)
	assert.Equal(t, ErrAccountLocked, err)

	mockRepo.AssertExpectations(t)
}

func TestLogin_AccountLockout_AfterMaxAttempts(t *testing.T) {
	mockRepo := new(MockUserRepository)
	jwtManager := auth.NewJWTManager(getTestJWTConfig())
	authService := NewAuthService(mockRepo, jwtManager, getTestAuthConfig())

	password := "CorrectPassword123!"
	passwordHash, _ := auth.HashPassword(password)
	userID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")

	user := &models.User{
		ID:               userID,
		Email:            "test@example.com",
		PasswordHash:     passwordHash,
		FailedLoginCount: 4, // One more attempt will trigger lockout
		EmailVerified:    true,
	}

	mockRepo.On("GetByEmail", "test@example.com").Return(user, nil)
	mockRepo.On("IncrementFailedLoginCountByUUID", userID).Return(nil)
	mockRepo.On("LockAccountByUUID", userID, mock.AnythingOfType("time.Time")).Return(nil)

	response, err := authService.Login("test@example.com", "WrongPassword")

	assert.Error(t, err)
	assert.Nil(t, response)
	assert.Equal(t, ErrInvalidCredentials, err)

	mockRepo.AssertExpectations(t)
}
