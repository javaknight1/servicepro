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

	handler := NewRegistrationHandler(mockService, nil)

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
	handler := NewRegistrationHandler(mockService, nil)

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
	assert.Equal(t, "malformed_json", response.Error)
}

func TestRegisterHandler_InvalidEmail(t *testing.T) {
	mockService := new(MockRegistrationService)
	handler := NewRegistrationHandler(mockService, nil)

	// Note: Gin's binding validation catches invalid email format before service is called
	// The service mock is not expected to be called because binding fails first

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
	assert.Equal(t, "validation_failed", response.Error)

	mockService.AssertExpectations(t)
}

func TestRegisterHandler_PasswordTooShort(t *testing.T) {
	mockService := new(MockRegistrationService)
	handler := NewRegistrationHandler(mockService, nil)

	// Note: Gin's binding validation catches password < 8 chars before service is called
	// The service mock is not expected to be called because binding fails first

	router := setupTestRouter()
	router.POST("/register", handler.Register)

	registerReq := models.RegisterRequest{
		Email:    "user@example.com",
		Password: "weak", // Only 4 characters, fails min=8 binding
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
	assert.Equal(t, "validation_failed", response.Error)

	mockService.AssertExpectations(t)
}

func TestRegisterHandler_WeakPassword(t *testing.T) {
	mockService := new(MockRegistrationService)
	handler := NewRegistrationHandler(mockService, nil)

	// Password is 8+ chars but fails service's strength requirements (no special chars, etc.)
	mockService.On("Register", "user@example.com", "weakpass").Return(nil, services.ErrWeakPassword)

	router := setupTestRouter()
	router.POST("/register", handler.Register)

	registerReq := models.RegisterRequest{
		Email:    "user@example.com",
		Password: "weakpass", // 8 characters but no special chars
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
	handler := NewRegistrationHandler(mockService, nil)

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
	handler := NewRegistrationHandler(mockService, nil)

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
			handler := NewRegistrationHandler(mockService, nil)

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

// T229: invite-only rejections carry distinct codes so the UI need not guess.
func TestRegisterHandler_InviteOnlyErrorCodes(t *testing.T) {
	code := "NOPE-NOPE-NOPE-NOPE"
	token := "tok-unknown"
	blank := ""
	cases := []struct {
		name            string
		inviteCode      *string
		invitationToken *string
		wantError       string
	}{
		{"no code or invitation", nil, nil, "registration_closed"},
		{"blank code", &blank, nil, "registration_closed"},
		{"rejected invite code", &code, nil, "invalid_invite_code"},
		{"rejected invitation", nil, &token, "invalid_invitation"},
		{"rejected code and invitation", &code, &token, "invalid_invite_code"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			mockService := new(MockRegistrationService)
			handler := NewRegistrationHandler(mockService, nil)
			// No invite-code or invitation store: every supplied value is rejected.
			handler.SetRegistrationGate(&services.RegistrationGate{Mode: services.RegistrationModeInviteOnly})

			router := setupTestRouter()
			router.POST("/register", handler.Register)

			body, _ := json.Marshal(models.RegisterRequest{
				Email:           "newuser@example.com",
				Password:        "SecurePass123!",
				InviteCode:      tc.inviteCode,
				InvitationToken: tc.invitationToken,
			})
			req, _ := http.NewRequest("POST", "/register", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, http.StatusForbidden, w.Code)
			var resp models.ErrorResponse
			assert.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
			assert.Equal(t, tc.wantError, resp.Error)
			assert.NotEmpty(t, resp.Message)
			mockService.AssertNotCalled(t, "Register", mock.Anything, mock.Anything)
		})
	}
}
