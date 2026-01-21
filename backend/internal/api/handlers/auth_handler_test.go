package handlers

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/javaknight1/servicepro/backend/internal/models"
	"github.com/javaknight1/servicepro/backend/internal/services"
)

// MockAuthService is a mock implementation of AuthService
type MockAuthService struct {
	mock.Mock
}

func (m *MockAuthService) Login(email, password string) (*models.LoginResponse, error) {
	args := m.Called(email, password)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.LoginResponse), args.Error(1)
}

func (m *MockAuthService) RefreshToken(refreshToken string) (*models.LoginResponse, error) {
	args := m.Called(refreshToken)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.LoginResponse), args.Error(1)
}

// MockRateLimiter is a mock implementation of RateLimiter
type MockRateLimiter struct {
	mock.Mock
}

func (m *MockRateLimiter) LoginRateLimit() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()
	}
}

func (m *MockRateLimiter) PasswordResetRateLimit() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()
	}
}

func (m *MockRateLimiter) ClearLoginAttempts(identifier string) error {
	args := m.Called(identifier)
	return args.Error(0)
}

func setupTestRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	return gin.New()
}

func TestLogin_Success(t *testing.T) {
	mockAuthService := new(MockAuthService)
	mockRateLimiter := new(MockRateLimiter)

	handler := NewAuthHandler(mockAuthService, mockRateLimiter)

	expectedResponse := &models.LoginResponse{
		Token:        "test-access-token",
		RefreshToken: "test-refresh-token",
		ExpiresIn:    3600,
	}

	mockAuthService.On("Login", "test@example.com", "SecurePassword123!").Return(expectedResponse, nil)
	mockRateLimiter.On("ClearLoginAttempts", "test@example.com").Return(nil)

	router := setupTestRouter()
	router.POST("/login", handler.Login)

	loginReq := models.LoginRequest{
		Email:    "test@example.com",
		Password: "SecurePassword123!",
	}
	body, _ := json.Marshal(loginReq)

	req, _ := http.NewRequest("POST", "/login", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response models.LoginResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, expectedResponse.Token, response.Token)
	assert.Equal(t, expectedResponse.RefreshToken, response.RefreshToken)
	assert.Equal(t, expectedResponse.ExpiresIn, response.ExpiresIn)

	mockAuthService.AssertExpectations(t)
	mockRateLimiter.AssertExpectations(t)
}

func TestLogin_InvalidRequest(t *testing.T) {
	mockAuthService := new(MockAuthService)
	mockRateLimiter := new(MockRateLimiter)

	handler := NewAuthHandler(mockAuthService, mockRateLimiter)

	router := setupTestRouter()
	router.POST("/login", handler.Login)

	// Invalid JSON
	req, _ := http.NewRequest("POST", "/login", bytes.NewBuffer([]byte("{invalid-json")))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var response models.ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "invalid_request", response.Error)
}

func TestLogin_InvalidCredentials(t *testing.T) {
	mockAuthService := new(MockAuthService)
	mockRateLimiter := new(MockRateLimiter)

	handler := NewAuthHandler(mockAuthService, mockRateLimiter)

	mockAuthService.On("Login", "test@example.com", "WrongPassword").Return(nil, services.ErrInvalidCredentials)

	router := setupTestRouter()
	router.POST("/login", handler.Login)

	loginReq := models.LoginRequest{
		Email:    "test@example.com",
		Password: "WrongPassword",
	}
	body, _ := json.Marshal(loginReq)

	req, _ := http.NewRequest("POST", "/login", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)

	var response models.ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "invalid_credentials", response.Error)

	mockAuthService.AssertExpectations(t)
}

func TestLogin_AccountLocked(t *testing.T) {
	mockAuthService := new(MockAuthService)
	mockRateLimiter := new(MockRateLimiter)

	handler := NewAuthHandler(mockAuthService, mockRateLimiter)

	mockAuthService.On("Login", "test@example.com", "SecurePassword123!").Return(nil, services.ErrAccountLocked)

	router := setupTestRouter()
	router.POST("/login", handler.Login)

	loginReq := models.LoginRequest{
		Email:    "test@example.com",
		Password: "SecurePassword123!",
	}
	body, _ := json.Marshal(loginReq)

	req, _ := http.NewRequest("POST", "/login", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)

	var response models.ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "account_locked", response.Error)

	mockAuthService.AssertExpectations(t)
}

func TestLogin_InternalError(t *testing.T) {
	mockAuthService := new(MockAuthService)
	mockRateLimiter := new(MockRateLimiter)

	handler := NewAuthHandler(mockAuthService, mockRateLimiter)

	mockAuthService.On("Login", "test@example.com", "SecurePassword123!").Return(nil, errors.New("database error"))

	router := setupTestRouter()
	router.POST("/login", handler.Login)

	loginReq := models.LoginRequest{
		Email:    "test@example.com",
		Password: "SecurePassword123!",
	}
	body, _ := json.Marshal(loginReq)

	req, _ := http.NewRequest("POST", "/login", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)

	var response models.ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "internal_error", response.Error)

	mockAuthService.AssertExpectations(t)
}
