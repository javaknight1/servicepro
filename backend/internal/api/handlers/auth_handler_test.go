package handlers

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/javaknight1/servicepro/backend/config"
	"github.com/javaknight1/servicepro/backend/internal/models"
	"github.com/javaknight1/servicepro/backend/internal/services"
	"github.com/javaknight1/servicepro/backend/pkg/auth"
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

func createTestCookieManager() *auth.CookieManager {
	cfg := &config.CookieConfig{
		Domain:           "",
		Secure:           false,
		SameSite:         "Lax",
		AccessTokenName:  "access_token",
		RefreshTokenName: "refresh_token",
		RefreshTokenPath: "/api/v1/auth",
	}
	return auth.NewCookieManager(cfg, time.Hour, 7*24*time.Hour)
}

func TestLogin_Success(t *testing.T) {
	mockAuthService := new(MockAuthService)
	mockRateLimiter := new(MockRateLimiter)

	handler := NewAuthHandler(mockAuthService, mockRateLimiter, createTestCookieManager())

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

	// Verify response body (tokens are now in cookies, not body)
	var response models.LoginCookieResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "Login successful", response.Message)
	assert.Equal(t, expectedResponse.ExpiresIn, response.ExpiresIn)

	// Verify cookies are set
	cookies := w.Result().Cookies()
	var accessTokenCookie, refreshTokenCookie *http.Cookie
	for _, cookie := range cookies {
		if cookie.Name == "access_token" {
			accessTokenCookie = cookie
		} else if cookie.Name == "refresh_token" {
			refreshTokenCookie = cookie
		}
	}

	assert.NotNil(t, accessTokenCookie, "access_token cookie should be set")
	assert.NotNil(t, refreshTokenCookie, "refresh_token cookie should be set")
	assert.Equal(t, expectedResponse.Token, accessTokenCookie.Value)
	assert.Equal(t, expectedResponse.RefreshToken, refreshTokenCookie.Value)
	assert.True(t, accessTokenCookie.HttpOnly, "access_token should be httpOnly")
	assert.True(t, refreshTokenCookie.HttpOnly, "refresh_token should be httpOnly")

	mockAuthService.AssertExpectations(t)
	mockRateLimiter.AssertExpectations(t)
}

func TestLogin_InvalidRequest(t *testing.T) {
	mockAuthService := new(MockAuthService)
	mockRateLimiter := new(MockRateLimiter)

	handler := NewAuthHandler(mockAuthService, mockRateLimiter, createTestCookieManager())

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

	handler := NewAuthHandler(mockAuthService, mockRateLimiter, createTestCookieManager())

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

	handler := NewAuthHandler(mockAuthService, mockRateLimiter, createTestCookieManager())

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

	handler := NewAuthHandler(mockAuthService, mockRateLimiter, createTestCookieManager())

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
