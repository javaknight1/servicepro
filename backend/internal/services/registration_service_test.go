package services

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/javaknight1/servicepro/backend/internal/models"
	"github.com/javaknight1/servicepro/backend/internal/repository"
	"github.com/javaknight1/servicepro/backend/pkg/auth"
)

// MockEmailService for testing
type MockEmailService struct {
	mock.Mock
}

func (m *MockEmailService) SendWelcomeEmail(to, name string) error {
	args := m.Called(to, name)
	return args.Error(0)
}

func (m *MockEmailService) SendEmail(to, subject, body string) error {
	args := m.Called(to, subject, body)
	return args.Error(0)
}

func (m *MockEmailService) SendPasswordResetEmail(to, resetToken, resetURL string) error {
	args := m.Called(to, resetToken, resetURL)
	return args.Error(0)
}

func (m *MockEmailService) SendPasswordResetConfirmationEmail(to string) error {
	args := m.Called(to)
	return args.Error(0)
}

func (m *MockEmailService) SendEmailVerificationEmail(to, verificationToken, verificationURL string) error {
	args := m.Called(to, verificationToken, verificationURL)
	return args.Error(0)
}

func (m *MockEmailService) SendEmailVerificationReminderEmail(to, verificationToken, verificationURL string) error {
	args := m.Called(to, verificationToken, verificationURL)
	return args.Error(0)
}

func (m *MockEmailService) SendEmailVerificationSuccessEmail(to string) error {
	args := m.Called(to)
	return args.Error(0)
}

// MockUserRepositoryWithEmailExists is a mock repository for testing
type MockUserRepositoryWithEmailExists struct {
	mock.Mock
	EmailExistsFunc func(email string) (bool, error)
}

func (m *MockUserRepositoryWithEmailExists) GetByEmail(email string) (*models.User, error) {
	args := m.Called(email)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.User), args.Error(1)
}

func (m *MockUserRepositoryWithEmailExists) IncrementFailedLoginCount(userID int) error {
	args := m.Called(userID)
	return args.Error(0)
}

func (m *MockUserRepositoryWithEmailExists) LockAccount(userID int, until time.Time) error {
	args := m.Called(userID, until)
	return args.Error(0)
}

func (m *MockUserRepositoryWithEmailExists) ResetFailedLoginCount(userID int) error {
	args := m.Called(userID)
	return args.Error(0)
}

func (m *MockUserRepositoryWithEmailExists) Create(email, passwordHash string) (*models.User, error) {
	args := m.Called(email, passwordHash)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.User), args.Error(1)
}

func (m *MockUserRepositoryWithEmailExists) EmailExists(email string) (bool, error) {
	if m.EmailExistsFunc != nil {
		return m.EmailExistsFunc(email)
	}
	return false, nil
}

func TestRegister_Success(t *testing.T) {
	mockRepo := &MockUserRepositoryWithEmailExists{}
	mockEmail := new(MockEmailService)
	registrationService := NewRegistrationService(mockRepo, mockEmail, nil)

	email := "newuser@example.com"
	password := "SecurePass123!"
	passwordHash, _ := auth.HashPassword(password)

	// Setup mocks
	mockRepo.EmailExistsFunc = func(e string) (bool, error) {
		return false, nil
	}

	user := &models.User{
		ID:           uuid.New(),
		Email:        email,
		PasswordHash: passwordHash,
		Role:         models.RoleUser,
	}

	// Mock expects GetByEmail to be called in the fallback path (non-GORM repo)
	mockRepo.On("GetByEmail", email).Return(nil, repository.ErrUserNotFound)
	mockRepo.On("Create", email, mock.AnythingOfType("string")).Return(user, nil)
	// Welcome email is sent when verification service is nil
	mockEmail.On("SendWelcomeEmail", email, email).Return(nil)

	response, err := registrationService.Register(email, password)

	assert.NoError(t, err)
	assert.NotNil(t, response)
	assert.Equal(t, email, response.Email)
	assert.Equal(t, models.RoleUser, response.Role)

	mockRepo.AssertExpectations(t)
	// Note: Email is sent async, so we can't assert it was called
}

func TestRegister_InvalidEmail(t *testing.T) {
	mockRepo := &MockUserRepositoryWithEmailExists{}
	mockEmail := new(MockEmailService)
	registrationService := NewRegistrationService(mockRepo, mockEmail, nil)

	email := "invalid-email"
	password := "SecurePass123!"

	response, err := registrationService.Register(email, password)

	assert.Error(t, err)
	assert.Nil(t, response)
	assert.Equal(t, auth.ErrInvalidEmailFormat, err)
}

func TestRegister_WeakPassword(t *testing.T) {
	mockRepo := &MockUserRepositoryWithEmailExists{}
	mockEmail := new(MockEmailService)
	registrationService := NewRegistrationService(mockRepo, mockEmail, nil)

	email := "user@example.com"
	password := "weak" // Too short, no special chars, etc.

	response, err := registrationService.Register(email, password)

	assert.Error(t, err)
	assert.Nil(t, response)
	assert.Equal(t, ErrWeakPassword, err)
}

func TestRegister_EmailAlreadyExists(t *testing.T) {
	mockRepo := &MockUserRepositoryWithEmailExists{}
	mockEmail := new(MockEmailService)
	registrationService := NewRegistrationService(mockRepo, mockEmail, nil)

	email := "existing@example.com"
	password := "SecurePass123!"

	// Setup mock to return that email exists
	mockRepo.EmailExistsFunc = func(e string) (bool, error) {
		return true, nil
	}

	// Mock expects GetByEmail to be called in the fallback path (non-GORM repo)
	// Return a user to indicate email exists
	existingUser := &models.User{
		ID:    uuid.New(),
		Email: email,
	}
	mockRepo.On("GetByEmail", email).Return(existingUser, nil)

	response, err := registrationService.Register(email, password)

	assert.Error(t, err)
	assert.Nil(t, response)
	assert.Equal(t, ErrEmailAlreadyExists, err)
}

func TestRegister_PasswordValidationErrors(t *testing.T) {
	testCases := []struct {
		name     string
		password string
	}{
		{"No uppercase", "lowercase123!"},
		{"No lowercase", "UPPERCASE123!"},
		{"No number", "NoNumbers!"},
		{"No special", "NoSpecial123"},
		{"Too short", "Short1!"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockRepo := &MockUserRepositoryWithEmailExists{}
			mockEmail := new(MockEmailService)
			registrationService := NewRegistrationService(mockRepo, mockEmail, nil)

			email := "user@example.com"

			response, err := registrationService.Register(email, tc.password)

			assert.Error(t, err)
			assert.Nil(t, response)
			assert.Equal(t, ErrWeakPassword, err)
		})
	}
}
