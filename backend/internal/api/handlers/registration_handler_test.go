package handlers

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/javaknight1/servicepro/backend/internal/models"
	"github.com/javaknight1/servicepro/backend/internal/services"
	"github.com/javaknight1/servicepro/backend/pkg/auth"
)

// MockRegistrationService is a mock implementation of RegistrationService
type MockRegistrationService struct {
	mock.Mock
}

func (m *MockRegistrationService) Register(email, password string) (*models.RegisterResponse, error) {
	args := m.Called(email, password)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.RegisterResponse), args.Error(1)
}

func TestRegisterHandler_Success(t *testing.T) {
	mockService := new(MockRegistrationService)

	expectedResponse := &models.RegisterResponse{
		ID:        uuid.New(),
		Email:     "newuser@example.com",
		Role:      models.RoleUser,
		CreatedAt: time.Now(),
	}

	mockService.On("Register", "newuser@example.com", "SecurePass123!").Return(expectedResponse, nil)

	handler := NewRegistrationHandler(mockService)

	router := setupTestRouter()
	router.POST("/register", handler.Register)

	registerReq := models.RegisterRequest{
		Email:    "newuser@example.com",
		Password: "SecurePass123!",
	}
	body, _ := json.Marshal(registerReq)

	req, _ := http.NewRequest("POST", "/register", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)

	var response models.RegisterResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, expectedResponse.Email, response.Email)
	assert.Equal(t, expectedResponse.Role, response.Role)

	mockService.AssertExpectations(t)
}

func TestRegisterHandler_InvalidRequest(t *testing.T) {
	mockService := new(MockRegistrationService)
	handler := NewRegistrationHandler(mockService)

	router := setupTestRouter()
	router.POST("/register", handler.Register)

	// Invalid JSON
	req, _ := http.NewRequest("POST", "/register", bytes.NewBuffer([]byte("{invalid-json")))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var response models.ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "invalid_request", response.Error)
}

func TestRegisterHandler_InvalidEmail(t *testing.T) {
	mockService := new(MockRegistrationService)
	handler := NewRegistrationHandler(mockService)

	mockService.On("Register", "invalid-email", "SecurePass123!").Return(nil, auth.ErrInvalidEmailFormat)

	router := setupTestRouter()
	router.POST("/register", handler.Register)

	registerReq := models.RegisterRequest{
		Email:    "invalid-email",
		Password: "SecurePass123!",
	}
	body, _ := json.Marshal(registerReq)

	req, _ := http.NewRequest("POST", "/register", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var response models.ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "invalid_email", response.Error)

	mockService.AssertExpectations(t)
}

func TestRegisterHandler_WeakPassword(t *testing.T) {
	mockService := new(MockRegistrationService)
	handler := NewRegistrationHandler(mockService)

	mockService.On("Register", "user@example.com", "weak").Return(nil, services.ErrWeakPassword)

	router := setupTestRouter()
	router.POST("/register", handler.Register)

	registerReq := models.RegisterRequest{
		Email:    "user@example.com",
		Password: "weak",
	}
	body, _ := json.Marshal(registerReq)

	req, _ := http.NewRequest("POST", "/register", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var response models.ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "weak_password", response.Error)

	mockService.AssertExpectations(t)
}

func TestRegisterHandler_EmailAlreadyExists(t *testing.T) {
	mockService := new(MockRegistrationService)
	handler := NewRegistrationHandler(mockService)

	mockService.On("Register", "existing@example.com", "SecurePass123!").Return(nil, services.ErrEmailAlreadyExists)

	router := setupTestRouter()
	router.POST("/register", handler.Register)

	registerReq := models.RegisterRequest{
		Email:    "existing@example.com",
		Password: "SecurePass123!",
	}
	body, _ := json.Marshal(registerReq)

	req, _ := http.NewRequest("POST", "/register", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusConflict, w.Code)

	var response models.ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "email_exists", response.Error)

	mockService.AssertExpectations(t)
}

func TestRegisterHandler_InternalError(t *testing.T) {
	mockService := new(MockRegistrationService)
	handler := NewRegistrationHandler(mockService)

	mockService.On("Register", "user@example.com", "SecurePass123!").Return(nil, errors.New("database error"))

	router := setupTestRouter()
	router.POST("/register", handler.Register)

	registerReq := models.RegisterRequest{
		Email:    "user@example.com",
		Password: "SecurePass123!",
	}
	body, _ := json.Marshal(registerReq)

	req, _ := http.NewRequest("POST", "/register", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)

	var response models.ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "internal_error", response.Error)

	mockService.AssertExpectations(t)
}

func TestRegisterHandler_PasswordValidationErrors(t *testing.T) {
	testCases := []struct {
		name          string
		returnError   error
		expectedError string
	}{
		{"Password too short", auth.ErrPasswordTooShort, "weak_password"},
		{"No uppercase", auth.ErrPasswordNoUppercase, "weak_password"},
		{"No lowercase", auth.ErrPasswordNoLowercase, "weak_password"},
		{"No number", auth.ErrPasswordNoNumber, "weak_password"},
		{"No special", auth.ErrPasswordNoSpecial, "weak_password"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockService := new(MockRegistrationService)
			handler := NewRegistrationHandler(mockService)

			mockService.On("Register", "user@example.com", "testpass").Return(nil, tc.returnError)

			router := setupTestRouter()
			router.POST("/register", handler.Register)

			registerReq := models.RegisterRequest{
				Email:    "user@example.com",
				Password: "testpass",
			}
			body, _ := json.Marshal(registerReq)

			req, _ := http.NewRequest("POST", "/register", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			assert.Equal(t, http.StatusBadRequest, w.Code)

			var response models.ErrorResponse
			err := json.Unmarshal(w.Body.Bytes(), &response)
			assert.NoError(t, err)
			assert.Equal(t, tc.expectedError, response.Error)

			mockService.AssertExpectations(t)
		})
	}
}
