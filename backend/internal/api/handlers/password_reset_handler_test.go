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
	"github.com/javaknight1/servicepro/backend/pkg/auth"
)

// MockPasswordResetService is a mock implementation of PasswordResetService
type MockPasswordResetService struct {
	mock.Mock
}

func (m *MockPasswordResetService) RequestPasswordReset(email string) error {
	args := m.Called(email)
	return args.Error(0)
}

func (m *MockPasswordResetService) ResetPassword(token, newPassword string) error {
	args := m.Called(token, newPassword)
	return args.Error(0)
}

func TestRequestPasswordReset_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockPasswordResetService)
	handler := NewPasswordResetHandler(mockService)

	// Setup mock
	mockService.On("RequestPasswordReset", "test@example.com").Return(nil)

	// Create request
	reqBody := models.PasswordResetRequestRequest{
		Email: "test@example.com",
	}
	jsonBody, _ := json.Marshal(reqBody)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request, _ = http.NewRequest("POST", "/api/v1/auth/reset-request", bytes.NewBuffer(jsonBody))
	c.Request.Header.Set("Content-Type", "application/json")

	// Execute
	handler.RequestPasswordReset(c)

	// Assert
	assert.Equal(t, http.StatusOK, w.Code)

	var response models.PasswordResetRequestResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "If an account with that email exists, a password reset link has been sent.", response.Message)

	mockService.AssertExpectations(t)
}

func TestRequestPasswordReset_InvalidJSON(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockPasswordResetService)
	handler := NewPasswordResetHandler(mockService)

	// Create request with invalid JSON
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request, _ = http.NewRequest("POST", "/api/v1/auth/reset-request", bytes.NewBuffer([]byte("invalid json")))
	c.Request.Header.Set("Content-Type", "application/json")

	// Execute
	handler.RequestPasswordReset(c)

	// Assert
	assert.Equal(t, http.StatusBadRequest, w.Code)

	var response models.ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "invalid_request", response.Error)
}

func TestRequestPasswordReset_InvalidEmail(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockPasswordResetService)
	handler := NewPasswordResetHandler(mockService)

	// Create request with invalid email that fails Gin validation
	reqBody := models.PasswordResetRequestRequest{
		Email: "not-an-email",
	}
	jsonBody, _ := json.Marshal(reqBody)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request, _ = http.NewRequest("POST", "/api/v1/auth/reset-request", bytes.NewBuffer(jsonBody))
	c.Request.Header.Set("Content-Type", "application/json")

	// Execute
	handler.RequestPasswordReset(c)

	// Assert - Gin's email validation should catch this
	assert.Equal(t, http.StatusBadRequest, w.Code)

	var response models.ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "invalid_request", response.Error)

	// Service should not be called since Gin validation fails
	mockService.AssertNotCalled(t, "RequestPasswordReset")
}

func TestRequestPasswordReset_ServiceError_ReturnsSuccess(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockPasswordResetService)
	handler := NewPasswordResetHandler(mockService)

	// Setup mock to return a generic error (should still return success for security)
	mockService.On("RequestPasswordReset", "test@example.com").Return(errors.New("database error"))

	// Create request
	reqBody := models.PasswordResetRequestRequest{
		Email: "test@example.com",
	}
	jsonBody, _ := json.Marshal(reqBody)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request, _ = http.NewRequest("POST", "/api/v1/auth/reset-request", bytes.NewBuffer(jsonBody))
	c.Request.Header.Set("Content-Type", "application/json")

	// Execute
	handler.RequestPasswordReset(c)

	// Assert - should return 200 OK to prevent email enumeration
	assert.Equal(t, http.StatusOK, w.Code)

	var response models.PasswordResetRequestResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "If an account with that email exists, a password reset link has been sent.", response.Message)

	mockService.AssertExpectations(t)
}

func TestResetPassword_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockPasswordResetService)
	handler := NewPasswordResetHandler(mockService)

	// Setup mock
	mockService.On("ResetPassword", "valid-token-123", "NewStrongP@ss123").Return(nil)

	// Create request
	reqBody := models.PasswordResetRequest{
		Token:       "valid-token-123",
		NewPassword: "NewStrongP@ss123",
	}
	jsonBody, _ := json.Marshal(reqBody)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request, _ = http.NewRequest("POST", "/api/v1/auth/reset-password", bytes.NewBuffer(jsonBody))
	c.Request.Header.Set("Content-Type", "application/json")

	// Execute
	handler.ResetPassword(c)

	// Assert
	assert.Equal(t, http.StatusOK, w.Code)

	var response models.PasswordResetResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "Password has been successfully reset. You can now log in with your new password.", response.Message)

	mockService.AssertExpectations(t)
}

func TestResetPassword_InvalidJSON(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockPasswordResetService)
	handler := NewPasswordResetHandler(mockService)

	// Create request with invalid JSON
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request, _ = http.NewRequest("POST", "/api/v1/auth/reset-password", bytes.NewBuffer([]byte("invalid json")))
	c.Request.Header.Set("Content-Type", "application/json")

	// Execute
	handler.ResetPassword(c)

	// Assert
	assert.Equal(t, http.StatusBadRequest, w.Code)

	var response models.ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "invalid_request", response.Error)
}

func TestResetPassword_InvalidToken(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockPasswordResetService)
	handler := NewPasswordResetHandler(mockService)

	// Setup mock to return invalid token error
	mockService.On("ResetPassword", "invalid-token", "NewStrongP@ss123").Return(services.ErrInvalidResetToken)

	// Create request
	reqBody := models.PasswordResetRequest{
		Token:       "invalid-token",
		NewPassword: "NewStrongP@ss123",
	}
	jsonBody, _ := json.Marshal(reqBody)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request, _ = http.NewRequest("POST", "/api/v1/auth/reset-password", bytes.NewBuffer(jsonBody))
	c.Request.Header.Set("Content-Type", "application/json")

	// Execute
	handler.ResetPassword(c)

	// Assert
	assert.Equal(t, http.StatusUnauthorized, w.Code)

	var response models.ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "invalid_token", response.Error)
	assert.Equal(t, "Invalid or expired reset token", response.Message)

	mockService.AssertExpectations(t)
}

func TestResetPassword_WeakPassword(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockPasswordResetService)
	handler := NewPasswordResetHandler(mockService)

	// Setup mock to return weak password error
	// Password "12345678" passes Gin's min=8 validation but fails strength check
	mockService.On("ResetPassword", "valid-token", "12345678").Return(services.ErrWeakPassword)

	// Create request
	reqBody := models.PasswordResetRequest{
		Token:       "valid-token",
		NewPassword: "12345678", // Passes min length but weak
	}
	jsonBody, _ := json.Marshal(reqBody)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request, _ = http.NewRequest("POST", "/api/v1/auth/reset-password", bytes.NewBuffer(jsonBody))
	c.Request.Header.Set("Content-Type", "application/json")

	// Execute
	handler.ResetPassword(c)

	// Assert
	assert.Equal(t, http.StatusBadRequest, w.Code)

	var response models.ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "weak_password", response.Error)

	mockService.AssertExpectations(t)
}

func TestResetPassword_PasswordValidationErrors(t *testing.T) {
	testCases := []struct {
		name          string
		serviceError  error
		expectedError string
	}{
		{"Password too short", auth.ErrPasswordTooShort, "weak_password"},
		{"No uppercase", auth.ErrPasswordNoUppercase, "weak_password"},
		{"No lowercase", auth.ErrPasswordNoLowercase, "weak_password"},
		{"No number", auth.ErrPasswordNoNumber, "weak_password"},
		{"No special char", auth.ErrPasswordNoSpecial, "weak_password"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			gin.SetMode(gin.TestMode)

			mockService := new(MockPasswordResetService)
			handler := NewPasswordResetHandler(mockService)

			// Setup mock to return specific password validation error
			mockService.On("ResetPassword", "valid-token", "testpass").Return(tc.serviceError)

			// Create request
			reqBody := models.PasswordResetRequest{
				Token:       "valid-token",
				NewPassword: "testpass",
			}
			jsonBody, _ := json.Marshal(reqBody)

			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request, _ = http.NewRequest("POST", "/api/v1/auth/reset-password", bytes.NewBuffer(jsonBody))
			c.Request.Header.Set("Content-Type", "application/json")

			// Execute
			handler.ResetPassword(c)

			// Assert
			assert.Equal(t, http.StatusBadRequest, w.Code)

			var response models.ErrorResponse
			err := json.Unmarshal(w.Body.Bytes(), &response)
			assert.NoError(t, err)
			assert.Equal(t, tc.expectedError, response.Error)

			mockService.AssertExpectations(t)
		})
	}
}

func TestResetPassword_InternalError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockPasswordResetService)
	handler := NewPasswordResetHandler(mockService)

	// Setup mock to return a generic internal error
	mockService.On("ResetPassword", "valid-token", "NewStrongP@ss123").Return(errors.New("database connection failed"))

	// Create request
	reqBody := models.PasswordResetRequest{
		Token:       "valid-token",
		NewPassword: "NewStrongP@ss123",
	}
	jsonBody, _ := json.Marshal(reqBody)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request, _ = http.NewRequest("POST", "/api/v1/auth/reset-password", bytes.NewBuffer(jsonBody))
	c.Request.Header.Set("Content-Type", "application/json")

	// Execute
	handler.ResetPassword(c)

	// Assert
	assert.Equal(t, http.StatusInternalServerError, w.Code)

	var response models.ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "internal_error", response.Error)
	assert.Equal(t, "An error occurred while processing your request", response.Message)

	mockService.AssertExpectations(t)
}
