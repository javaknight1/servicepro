package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/javaknight1/servicepro/backend/internal/models"
	"github.com/javaknight1/servicepro/backend/internal/services"
)

// MockStatusService is a mock implementation of StatusServiceInterface
type MockStatusService struct {
	mock.Mock
}

func (m *MockStatusService) ChangeStatus(ctx context.Context, req *models.StatusChangeRequest, changedBy uuid.UUID) (*models.StatusChangeResponse, error) {
	args := m.Called(ctx, req, changedBy)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.StatusChangeResponse), args.Error(1)
}

func (m *MockStatusService) GetStatusHistory(ctx context.Context, jobID uuid.UUID, limit, offset int) (*models.StatusHistoryResponse, error) {
	args := m.Called(ctx, jobID, limit, offset)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.StatusHistoryResponse), args.Error(1)
}

func (m *MockStatusService) GetAllowedTransitions(ctx context.Context, jobID uuid.UUID) (*models.AllowedTransitionsResponse, error) {
	args := m.Called(ctx, jobID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.AllowedTransitionsResponse), args.Error(1)
}

func (m *MockStatusService) BulkChangeStatus(ctx context.Context, requests []*models.StatusChangeRequest, changedBy uuid.UUID) ([]*models.StatusChangeResponse, []error) {
	args := m.Called(ctx, requests, changedBy)
	return args.Get(0).([]*models.StatusChangeResponse), args.Get(1).([]error)
}

func (m *MockStatusService) ValidateStatusChange(ctx context.Context, jobID uuid.UUID, toStatus models.JobStatus) error {
	args := m.Called(ctx, jobID, toStatus)
	return args.Error(0)
}

// TestChangeStatus_Success tests successful status change
func TestChangeStatus_Success(t *testing.T) {
	mockStatusService := new(MockStatusService)
	handler := NewStatusHandler(mockStatusService)

	userID := uuid.New()
	jobID := uuid.New()

	req := models.StatusChangeRequest{
		ToStatus: models.JobStatusInProgress,
		Reason:   models.ReasonStartWork,
	}

	expectedResponse := &models.StatusChangeResponse{
		JobID:          jobID,
		JobNumber:      "JOB-001",
		PreviousStatus: models.JobStatusScheduled,
		NewStatus:      models.JobStatusInProgress,
		Reason:         models.ReasonStartWork,
		ChangedBy:      userID,
		ChangedAt:      time.Now(),
	}

	mockStatusService.On("ChangeStatus", mock.Anything, mock.MatchedBy(func(r *models.StatusChangeRequest) bool {
		return r.JobID == jobID && r.ToStatus == models.JobStatusInProgress
	}), userID).Return(expectedResponse, nil)

	router := setupTestRouter()
	router = setupAuthContext(router, userID, models.UserRoleAdmin)
	router.POST("/jobs/:id/status", handler.ChangeStatus)

	body, _ := json.Marshal(req)
	httpReq, _ := http.NewRequest("POST", "/jobs/"+jobID.String()+"/status", bytes.NewBuffer(body))
	httpReq.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, httpReq)

	assert.Equal(t, http.StatusOK, w.Code)

	var response models.StatusChangeResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, models.JobStatusInProgress, response.NewStatus)

	mockStatusService.AssertExpectations(t)
}

// TestChangeStatus_Unauthorized tests unauthorized access
func TestChangeStatus_Unauthorized(t *testing.T) {
	mockStatusService := new(MockStatusService)
	handler := NewStatusHandler(mockStatusService)

	router := setupTestRouter()
	router.POST("/jobs/:id/status", handler.ChangeStatus)

	req := models.StatusChangeRequest{
		ToStatus: models.JobStatusInProgress,
	}

	body, _ := json.Marshal(req)
	httpReq, _ := http.NewRequest("POST", "/jobs/"+uuid.New().String()+"/status", bytes.NewBuffer(body))
	httpReq.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, httpReq)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

// TestChangeStatus_Forbidden tests forbidden access for customer role
func TestChangeStatus_Forbidden(t *testing.T) {
	mockStatusService := new(MockStatusService)
	handler := NewStatusHandler(mockStatusService)

	router := setupTestRouter()
	router = setupAuthContext(router, uuid.New(), models.RoleUser)
	router.POST("/jobs/:id/status", handler.ChangeStatus)

	req := models.StatusChangeRequest{
		ToStatus: models.JobStatusInProgress,
	}

	body, _ := json.Marshal(req)
	httpReq, _ := http.NewRequest("POST", "/jobs/"+uuid.New().String()+"/status", bytes.NewBuffer(body))
	httpReq.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, httpReq)

	assert.Equal(t, http.StatusForbidden, w.Code)
}

// TestChangeStatus_InvalidJobID tests invalid job ID format
func TestChangeStatus_InvalidJobID(t *testing.T) {
	mockStatusService := new(MockStatusService)
	handler := NewStatusHandler(mockStatusService)

	router := setupTestRouter()
	router = setupAuthContext(router, uuid.New(), models.UserRoleAdmin)
	router.POST("/jobs/:id/status", handler.ChangeStatus)

	req := models.StatusChangeRequest{
		ToStatus: models.JobStatusInProgress,
	}

	body, _ := json.Marshal(req)
	httpReq, _ := http.NewRequest("POST", "/jobs/invalid-uuid/status", bytes.NewBuffer(body))
	httpReq.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, httpReq)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var response models.ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "invalid_id", response.Error)
}

// TestChangeStatus_InvalidRequest tests invalid request body
func TestChangeStatus_InvalidRequest(t *testing.T) {
	mockStatusService := new(MockStatusService)
	handler := NewStatusHandler(mockStatusService)

	router := setupTestRouter()
	router = setupAuthContext(router, uuid.New(), models.UserRoleAdmin)
	router.POST("/jobs/:id/status", handler.ChangeStatus)

	httpReq, _ := http.NewRequest("POST", "/jobs/"+uuid.New().String()+"/status", bytes.NewBuffer([]byte("{invalid-json")))
	httpReq.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, httpReq)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// TestChangeStatus_JobNotFound tests job not found error
func TestChangeStatus_JobNotFound(t *testing.T) {
	mockStatusService := new(MockStatusService)
	handler := NewStatusHandler(mockStatusService)

	userID := uuid.New()
	jobID := uuid.New()

	req := models.StatusChangeRequest{
		ToStatus: models.JobStatusInProgress,
	}

	mockStatusService.On("ChangeStatus", mock.Anything, mock.Anything, userID).Return(nil, services.ErrJobNotFound)

	router := setupTestRouter()
	router = setupAuthContext(router, userID, models.UserRoleAdmin)
	router.POST("/jobs/:id/status", handler.ChangeStatus)

	body, _ := json.Marshal(req)
	httpReq, _ := http.NewRequest("POST", "/jobs/"+jobID.String()+"/status", bytes.NewBuffer(body))
	httpReq.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, httpReq)

	assert.Equal(t, http.StatusNotFound, w.Code)

	var response models.ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "job_not_found", response.Error)

	mockStatusService.AssertExpectations(t)
}

// TestChangeStatus_InvalidTransition tests invalid status transition
func TestChangeStatus_InvalidTransition(t *testing.T) {
	mockStatusService := new(MockStatusService)
	handler := NewStatusHandler(mockStatusService)

	userID := uuid.New()
	jobID := uuid.New()

	req := models.StatusChangeRequest{
		ToStatus: models.JobStatusInProgress,
	}

	mockStatusService.On("ChangeStatus", mock.Anything, mock.Anything, userID).Return(nil, models.ErrInvalidStatusTransition)

	router := setupTestRouter()
	router = setupAuthContext(router, userID, models.UserRoleAdmin)
	router.POST("/jobs/:id/status", handler.ChangeStatus)

	body, _ := json.Marshal(req)
	httpReq, _ := http.NewRequest("POST", "/jobs/"+jobID.String()+"/status", bytes.NewBuffer(body))
	httpReq.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, httpReq)

	assert.Equal(t, http.StatusConflict, w.Code)

	var response models.ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "invalid_transition", response.Error)

	mockStatusService.AssertExpectations(t)
}

// TestGetStatusHistory_Success tests successful status history retrieval
func TestGetStatusHistory_Success(t *testing.T) {
	mockStatusService := new(MockStatusService)
	handler := NewStatusHandler(mockStatusService)

	jobID := uuid.New()

	expectedHistory := &models.StatusHistoryResponse{
		Transitions: []models.StatusTransitionResponse{
			{
				ID:             uuid.New(),
				JobID:          jobID,
				JobNumber:      "JOB-001",
				FromStatus:     models.JobStatusScheduled,
				ToStatus:       models.JobStatusInProgress,
				ChangedBy:      uuid.New(),
				TransitionedAt: time.Now(),
			},
		},
		Total: 1,
	}

	mockStatusService.On("GetStatusHistory", mock.Anything, jobID, 50, 0).Return(expectedHistory, nil)

	router := setupTestRouter()
	router = setupAuthContext(router, uuid.New(), models.UserRoleAdmin)
	router.GET("/jobs/:id/status/history", handler.GetStatusHistory)

	httpReq, _ := http.NewRequest("GET", "/jobs/"+jobID.String()+"/status/history", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, httpReq)

	assert.Equal(t, http.StatusOK, w.Code)

	var response models.StatusHistoryResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, int64(1), response.Total)
	assert.Len(t, response.Transitions, 1)

	mockStatusService.AssertExpectations(t)
}

// TestGetAllowedTransitions_Success tests successful allowed transitions retrieval
func TestGetAllowedTransitions_Success(t *testing.T) {
	mockStatusService := new(MockStatusService)
	handler := NewStatusHandler(mockStatusService)

	jobID := uuid.New()

	expectedTransitions := &models.AllowedTransitionsResponse{
		JobID:              jobID,
		CurrentStatus:      models.JobStatusScheduled,
		AllowedTransitions: []models.JobStatus{models.JobStatusInProgress, models.JobStatusCancelled},
		IsTerminal:         false,
	}

	mockStatusService.On("GetAllowedTransitions", mock.Anything, jobID).Return(expectedTransitions, nil)

	router := setupTestRouter()
	router = setupAuthContext(router, uuid.New(), models.UserRoleAdmin)
	router.GET("/jobs/:id/status/transitions", handler.GetAllowedTransitions)

	httpReq, _ := http.NewRequest("GET", "/jobs/"+jobID.String()+"/status/transitions", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, httpReq)

	assert.Equal(t, http.StatusOK, w.Code)

	var response models.AllowedTransitionsResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, models.JobStatusScheduled, response.CurrentStatus)
	assert.Len(t, response.AllowedTransitions, 2)

	mockStatusService.AssertExpectations(t)
}

// TestValidateStatusChange_Valid tests valid status change validation
func TestValidateStatusChange_Valid(t *testing.T) {
	mockStatusService := new(MockStatusService)
	handler := NewStatusHandler(mockStatusService)

	jobID := uuid.New()

	req := map[string]string{
		"to_status": "in_progress",
	}

	mockStatusService.On("ValidateStatusChange", mock.Anything, jobID, models.JobStatusInProgress).Return(nil)

	router := setupTestRouter()
	router = setupAuthContext(router, uuid.New(), models.UserRoleAdmin)
	router.POST("/jobs/:id/status/validate", handler.ValidateStatusChange)

	body, _ := json.Marshal(req)
	httpReq, _ := http.NewRequest("POST", "/jobs/"+jobID.String()+"/status/validate", bytes.NewBuffer(body))
	httpReq.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, httpReq)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.True(t, response["valid"].(bool))

	mockStatusService.AssertExpectations(t)
}

// TestValidateStatusChange_Invalid tests invalid status change validation
func TestValidateStatusChange_Invalid(t *testing.T) {
	mockStatusService := new(MockStatusService)
	handler := NewStatusHandler(mockStatusService)

	jobID := uuid.New()

	req := map[string]string{
		"to_status": "in_progress",
	}

	mockStatusService.On("ValidateStatusChange", mock.Anything, jobID, models.JobStatusInProgress).
		Return(errors.New("job must have at least one assignment"))

	router := setupTestRouter()
	router = setupAuthContext(router, uuid.New(), models.UserRoleAdmin)
	router.POST("/jobs/:id/status/validate", handler.ValidateStatusChange)

	body, _ := json.Marshal(req)
	httpReq, _ := http.NewRequest("POST", "/jobs/"+jobID.String()+"/status/validate", bytes.NewBuffer(body))
	httpReq.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, httpReq)

	assert.Equal(t, http.StatusConflict, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.False(t, response["valid"].(bool))

	mockStatusService.AssertExpectations(t)
}

// TestBulkChangeStatus_Success tests successful bulk status change
func TestBulkChangeStatus_Success(t *testing.T) {
	mockStatusService := new(MockStatusService)
	handler := NewStatusHandler(mockStatusService)

	userID := uuid.New()
	jobID1 := uuid.New()
	jobID2 := uuid.New()

	requests := []*models.StatusChangeRequest{
		{JobID: jobID1, ToStatus: models.JobStatusInProgress},
		{JobID: jobID2, ToStatus: models.JobStatusInProgress},
	}

	responses := []*models.StatusChangeResponse{
		{
			JobID:          jobID1,
			JobNumber:      "JOB-001",
			PreviousStatus: models.JobStatusScheduled,
			NewStatus:      models.JobStatusInProgress,
			ChangedBy:      userID,
			ChangedAt:      time.Now(),
		},
		{
			JobID:          jobID2,
			JobNumber:      "JOB-002",
			PreviousStatus: models.JobStatusScheduled,
			NewStatus:      models.JobStatusInProgress,
			ChangedBy:      userID,
			ChangedAt:      time.Now(),
		},
	}

	mockStatusService.On("BulkChangeStatus", mock.Anything, mock.MatchedBy(func(reqs []*models.StatusChangeRequest) bool {
		return len(reqs) == 2
	}), userID).Return(responses, []error{})

	router := setupTestRouter()
	router = setupAuthContext(router, userID, models.UserRoleAdmin)
	router.POST("/jobs/status/bulk", handler.BulkChangeStatus)

	body, _ := json.Marshal(requests)
	httpReq, _ := http.NewRequest("POST", "/jobs/status/bulk", bytes.NewBuffer(body))
	httpReq.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, httpReq)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, float64(2), response["success_count"])
	assert.Equal(t, float64(0), response["failure_count"])

	mockStatusService.AssertExpectations(t)
}

// TestBulkChangeStatus_PartialFailure tests partial failure in bulk change
func TestBulkChangeStatus_PartialFailure(t *testing.T) {
	mockStatusService := new(MockStatusService)
	handler := NewStatusHandler(mockStatusService)

	userID := uuid.New()
	jobID1 := uuid.New()
	jobID2 := uuid.New()

	requests := []*models.StatusChangeRequest{
		{JobID: jobID1, ToStatus: models.JobStatusInProgress},
		{JobID: jobID2, ToStatus: models.JobStatusInProgress},
	}

	responses := []*models.StatusChangeResponse{
		{
			JobID:          jobID1,
			JobNumber:      "JOB-001",
			PreviousStatus: models.JobStatusScheduled,
			NewStatus:      models.JobStatusInProgress,
			ChangedBy:      userID,
			ChangedAt:      time.Now(),
		},
	}

	errs := []error{
		errors.New("job JOB-002: job not found"),
	}

	mockStatusService.On("BulkChangeStatus", mock.Anything, mock.Anything, userID).Return(responses, errs)

	router := setupTestRouter()
	router = setupAuthContext(router, userID, models.UserRoleAdmin)
	router.POST("/jobs/status/bulk", handler.BulkChangeStatus)

	body, _ := json.Marshal(requests)
	httpReq, _ := http.NewRequest("POST", "/jobs/status/bulk", bytes.NewBuffer(body))
	httpReq.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, httpReq)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, float64(1), response["success_count"])
	assert.Equal(t, float64(1), response["failure_count"])

	mockStatusService.AssertExpectations(t)
}

// TestBulkChangeStatus_Forbidden tests forbidden access for technician role
func TestBulkChangeStatus_Forbidden(t *testing.T) {
	mockStatusService := new(MockStatusService)
	handler := NewStatusHandler(mockStatusService)

	router := setupTestRouter()
	router = setupAuthContext(router, uuid.New(), models.UserRoleTechnician)
	router.POST("/jobs/status/bulk", handler.BulkChangeStatus)

	requests := []*models.StatusChangeRequest{
		{JobID: uuid.New(), ToStatus: models.JobStatusInProgress},
	}

	body, _ := json.Marshal(requests)
	httpReq, _ := http.NewRequest("POST", "/jobs/status/bulk", bytes.NewBuffer(body))
	httpReq.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, httpReq)

	assert.Equal(t, http.StatusForbidden, w.Code)
}

// TestGetStatusStatistics tests status statistics endpoint
func TestGetStatusStatistics(t *testing.T) {
	mockStatusService := new(MockStatusService)
	handler := NewStatusHandler(mockStatusService)

	router := setupTestRouter()
	router = setupAuthContext(router, uuid.New(), models.UserRoleAdmin)
	router.GET("/jobs/status/statistics", handler.GetStatusStatistics)

	httpReq, _ := http.NewRequest("GET", "/jobs/status/statistics", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, httpReq)

	require.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Contains(t, response, "status_counts")
	assert.Contains(t, response, "transition_metrics")
}
