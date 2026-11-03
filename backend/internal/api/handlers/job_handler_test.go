package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/javaknight1/servicepro/backend/internal/api/middleware"
	"github.com/javaknight1/servicepro/backend/internal/models"
	"github.com/javaknight1/servicepro/backend/internal/services"
)

// MockJobService is a mock implementation of JobServiceInterface
type MockJobService struct {
	mock.Mock
}

func (m *MockJobService) CreateJob(req *models.CreateJobRequest, createdBy uuid.UUID) (*models.JobResponse, error) {
	args := m.Called(req, createdBy)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.JobResponse), args.Error(1)
}

func (m *MockJobService) GetJobByID(id uuid.UUID, userID uuid.UUID, userRole models.UserRole) (*models.JobResponse, error) {
	args := m.Called(id, userID, userRole)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.JobResponse), args.Error(1)
}

func (m *MockJobService) GetJobByJobNumber(jobNumber string, userID uuid.UUID, userRole models.UserRole) (*models.JobResponse, error) {
	args := m.Called(jobNumber, userID, userRole)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.JobResponse), args.Error(1)
}

func (m *MockJobService) UpdateJob(id uuid.UUID, req *models.UpdateJobRequest, userID uuid.UUID, userRole models.UserRole) (*models.JobResponse, error) {
	args := m.Called(id, req, userID, userRole)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.JobResponse), args.Error(1)
}

func (m *MockJobService) DeleteJob(id uuid.UUID, userID uuid.UUID, userRole models.UserRole) error {
	args := m.Called(id, userID, userRole)
	return args.Error(0)
}

func (m *MockJobService) ListJobs(filter *models.JobListFilter, userID uuid.UUID, userRole models.UserRole) ([]models.JobResponse, int64, error) {
	args := m.Called(filter, userID, userRole)
	return args.Get(0).([]models.JobResponse), args.Get(1).(int64), args.Error(2)
}

func (m *MockJobService) GetMyJobs(userID uuid.UUID) ([]models.JobResponse, error) {
	args := m.Called(userID)
	return args.Get(0).([]models.JobResponse), args.Error(1)
}

func (m *MockJobService) GetMyDay(userID uuid.UUID, date string) (*models.MyDayResponse, error) {
	args := m.Called(userID, date)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.MyDayResponse), args.Error(1)
}

func (m *MockJobService) GetScheduledJobs(start, end int64) ([]models.JobResponse, error) {
	args := m.Called(start, end)
	return args.Get(0).([]models.JobResponse), args.Error(1)
}

func (m *MockJobService) GetOverdueJobs() ([]models.JobResponse, error) {
	args := m.Called()
	return args.Get(0).([]models.JobResponse), args.Error(1)
}

func (m *MockJobService) AssignTechnician(jobID, technicianID uuid.UUID, role string, userID uuid.UUID, userRole models.UserRole) error {
	args := m.Called(jobID, technicianID, role, userID, userRole)
	return args.Error(0)
}

func (m *MockJobService) UnassignTechnician(jobID, assignmentID uuid.UUID, userID uuid.UUID, userRole models.UserRole) error {
	args := m.Called(jobID, assignmentID, userID, userRole)
	return args.Error(0)
}

func (m *MockJobService) AddMaterial(jobID uuid.UUID, material *models.JobMaterial, userID uuid.UUID, userRole models.UserRole) error {
	args := m.Called(jobID, material, userID, userRole)
	return args.Error(0)
}

func (m *MockJobService) UpdateMaterial(materialID uuid.UUID, material *models.JobMaterial, userID uuid.UUID, userRole models.UserRole) error {
	args := m.Called(materialID, material, userID, userRole)
	return args.Error(0)
}

func (m *MockJobService) DeleteMaterial(materialID uuid.UUID, jobID uuid.UUID, userID uuid.UUID, userRole models.UserRole) error {
	args := m.Called(materialID, jobID, userID, userRole)
	return args.Error(0)
}

func (m *MockJobService) AddNote(jobID, userID uuid.UUID, note string, isInternal bool, userRole models.UserRole) error {
	args := m.Called(jobID, userID, note, isInternal, userRole)
	return args.Error(0)
}

func (m *MockJobService) GetJobStats() (*models.JobStats, error) {
	args := m.Called()
	stats, _ := args.Get(0).(*models.JobStats)
	return stats, args.Error(1)
}

func (m *MockJobService) GetTechnicianWorkload(userID uuid.UUID) (map[string]interface{}, error) {
	args := m.Called(userID)
	return args.Get(0).(map[string]interface{}), args.Error(1)
}

func (m *MockJobService) GetCustomerJobs(customerID uuid.UUID, limit, offset int) ([]models.JobResponse, int64, error) {
	args := m.Called(customerID, limit, offset)
	return args.Get(0).([]models.JobResponse), args.Get(1).(int64), args.Error(2)
}

func (m *MockJobService) TransitionStatus(jobID uuid.UUID, toStatus models.JobStatus, reason string, notes *string, userID uuid.UUID, userRole models.UserRole) (*models.JobResponse, error) {
	args := m.Called(jobID, toStatus, reason, notes, userID, userRole)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.JobResponse), args.Error(1)
}

func (m *MockJobService) GetStatusHistory(jobID uuid.UUID, limit, offset int, sortOrder string) (*models.StatusHistoryResponse, error) {
	args := m.Called(jobID, limit, offset, sortOrder)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.StatusHistoryResponse), args.Error(1)
}

// Helper functions
func setupJobTestRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	return gin.New()
}

func int64Ptr(v int64) *int64 { return &v }

func createTestJobResponse() *models.JobResponse {
	now := time.Now()
	scheduledStart := now.Add(24 * time.Hour).Unix()
	scheduledEnd := now.Add(26 * time.Hour).Unix()

	return &models.JobResponse{
		ID:               uuid.New(),
		JobNumber:        "JOB-20240101-0001",
		CustomerID:       uuid.New(),
		Title:            "Test Job",
		Description:      "Test Description",
		JobType:          models.JobTypeInstallation,
		Status:           models.JobStatusScheduled,
		Priority:         models.JobPriorityNormal,
		ScheduledStartAt: &scheduledStart,
		ScheduledEndAt:   &scheduledEnd,
		ServiceAddress: models.ServiceAddress{
			Street: "123 Main St",
			City:   "Austin",
			State:  "TX",
			Zip:    "78701",
		},
		CreatedAt: now,
		UpdatedAt: now,
	}
}

// Tests for CreateJob
func TestJobHandler_CreateJob_Success(t *testing.T) {
	mockService := new(MockJobService)
	handler := NewJobHandler(mockService)
	router := setupJobTestRouter()

	userID := uuid.New()
	userRole := models.RoleUser

	router.POST("/jobs", func(c *gin.Context) {
		c.Set("user_id", userID)
		c.Set("userRole", userRole)
		handler.CreateJob(c)
	})

	customerID := uuid.New()
	now := time.Now()
	scheduledStart := now.Add(24 * time.Hour).Unix()
	scheduledEnd := now.Add(26 * time.Hour).Unix()

	reqBody := &models.CreateJobRequest{
		CustomerID:       customerID,
		Title:            "Installation Job",
		Description:      "Install HVAC system",
		JobType:          models.JobTypeInstallation,
		Priority:         models.JobPriorityNormal,
		ScheduledStartAt: &scheduledStart,
		ScheduledEndAt:   &scheduledEnd,
	}

	response := createTestJobResponse()
	mockService.On("CreateJob", mock.AnythingOfType("*models.CreateJobRequest"), userID).Return(response, nil)

	jsonBody, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/jobs", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)

	var respBody models.JobResponse
	err := json.Unmarshal(w.Body.Bytes(), &respBody)
	require.NoError(t, err)
	assert.Equal(t, response.ID, respBody.ID)
	mockService.AssertExpectations(t)
}

func TestJobHandler_CreateJob_Unauthorized(t *testing.T) {
	mockService := new(MockJobService)
	handler := NewJobHandler(mockService)
	router := setupJobTestRouter()

	// No userID in context
	router.POST("/jobs", handler.CreateJob)

	reqBody := &models.CreateJobRequest{
		CustomerID: uuid.New(),
		Title:      "Test Job",
		JobType:    models.JobTypeInstallation,
	}

	jsonBody, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/jobs", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestJobHandler_CreateJob_InvalidJSON(t *testing.T) {
	mockService := new(MockJobService)
	handler := NewJobHandler(mockService)
	router := setupJobTestRouter()

	userID := uuid.New()
	router.POST("/jobs", func(c *gin.Context) {
		c.Set("user_id", userID)
		c.Set("userRole", models.RoleUser)
		handler.CreateJob(c)
	})

	req := httptest.NewRequest(http.MethodPost, "/jobs", bytes.NewBufferString("invalid json"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestJobHandler_CreateJob_CustomerNotFound(t *testing.T) {
	mockService := new(MockJobService)
	handler := NewJobHandler(mockService)
	router := setupJobTestRouter()

	userID := uuid.New()
	router.POST("/jobs", func(c *gin.Context) {
		c.Set("user_id", userID)
		c.Set("userRole", models.RoleUser)
		handler.CreateJob(c)
	})

	reqBody := &models.CreateJobRequest{
		CustomerID: uuid.New(),
		Title:      "Test Job",
		JobType:    models.JobTypeInstallation,
	}

	mockService.On("CreateJob", mock.AnythingOfType("*models.CreateJobRequest"), userID).
		Return(nil, services.ErrCustomerNotFound)

	jsonBody, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/jobs", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
	mockService.AssertExpectations(t)
}

// Tests for GetJob
func TestJobHandler_GetJob_Success(t *testing.T) {
	mockService := new(MockJobService)
	handler := NewJobHandler(mockService)
	router := setupJobTestRouter()

	userID := uuid.New()
	jobID := uuid.New()

	router.GET("/jobs/:id", func(c *gin.Context) {
		c.Set("user_id", userID)
		c.Set("userRole", models.RoleUser)
		handler.GetJob(c)
	})

	response := createTestJobResponse()
	response.ID = jobID
	mockService.On("GetJobByID", jobID, userID, models.RoleUser).Return(response, nil)

	req := httptest.NewRequest(http.MethodGet, "/jobs/"+jobID.String(), nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var respBody models.JobResponse
	err := json.Unmarshal(w.Body.Bytes(), &respBody)
	require.NoError(t, err)
	assert.Equal(t, jobID, respBody.ID)
	mockService.AssertExpectations(t)
}

func TestJobHandler_GetJob_InvalidUUID(t *testing.T) {
	mockService := new(MockJobService)
	handler := NewJobHandler(mockService)
	router := setupJobTestRouter()

	userID := uuid.New()
	router.GET("/jobs/:id", func(c *gin.Context) {
		c.Set("user_id", userID)
		c.Set("userRole", models.RoleUser)
		handler.GetJob(c)
	})

	req := httptest.NewRequest(http.MethodGet, "/jobs/invalid-uuid", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestJobHandler_GetJob_NotFound(t *testing.T) {
	mockService := new(MockJobService)
	handler := NewJobHandler(mockService)
	router := setupJobTestRouter()

	userID := uuid.New()
	jobID := uuid.New()

	router.GET("/jobs/:id", func(c *gin.Context) {
		c.Set("user_id", userID)
		c.Set("userRole", models.RoleUser)
		handler.GetJob(c)
	})

	mockService.On("GetJobByID", jobID, userID, models.RoleUser).
		Return(nil, services.ErrJobNotFound)

	req := httptest.NewRequest(http.MethodGet, "/jobs/"+jobID.String(), nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
	mockService.AssertExpectations(t)
}

// Tests for ListJobs
func TestJobHandler_ListJobs_Success(t *testing.T) {
	mockService := new(MockJobService)
	handler := NewJobHandler(mockService)
	router := setupJobTestRouter()

	userID := uuid.New()
	router.GET("/jobs", func(c *gin.Context) {
		c.Set("user_id", userID)
		c.Set("userRole", models.RoleUser)
		handler.ListJobs(c)
	})

	jobs := []models.JobResponse{*createTestJobResponse(), *createTestJobResponse()}
	mockService.On("ListJobs", mock.AnythingOfType("*models.JobListFilter"), userID, models.RoleUser).
		Return(jobs, int64(2), nil)

	req := httptest.NewRequest(http.MethodGet, "/jobs?limit=10&offset=0", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var respBody struct {
		Jobs  []models.JobResponse `json:"jobs"`
		Total int64                `json:"total"`
	}
	err := json.Unmarshal(w.Body.Bytes(), &respBody)
	require.NoError(t, err)
	assert.Len(t, respBody.Jobs, 2)
	assert.Equal(t, int64(2), respBody.Total)
	mockService.AssertExpectations(t)
}

// Tests for UpdateJob
func TestJobHandler_UpdateJob_Success(t *testing.T) {
	mockService := new(MockJobService)
	handler := NewJobHandler(mockService)
	router := setupJobTestRouter()

	userID := uuid.New()
	jobID := uuid.New()

	router.PUT("/jobs/:id", func(c *gin.Context) {
		c.Set("user_id", userID)
		c.Set("userRole", models.RoleUser)
		handler.UpdateJob(c)
	})

	newTitle := "Updated Title"
	reqBody := &models.UpdateJobRequest{
		Title: &newTitle,
	}

	response := createTestJobResponse()
	response.ID = jobID
	response.Title = newTitle

	mockService.On("UpdateJob", jobID, mock.AnythingOfType("*models.UpdateJobRequest"), userID, models.RoleUser).
		Return(response, nil)

	jsonBody, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPut, "/jobs/"+jobID.String(), bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	mockService.AssertExpectations(t)
}

// Tests for DeleteJob
func TestJobHandler_DeleteJob_Success(t *testing.T) {
	mockService := new(MockJobService)
	handler := NewJobHandler(mockService)
	router := setupJobTestRouter()

	userID := uuid.New()
	jobID := uuid.New()

	router.DELETE("/jobs/:id", func(c *gin.Context) {
		c.Set("user_id", userID)
		c.Set("userRole", models.RoleUser)
		handler.DeleteJob(c)
	})

	mockService.On("DeleteJob", jobID, userID, models.RoleUser).Return(nil)

	req := httptest.NewRequest(http.MethodDelete, "/jobs/"+jobID.String(), nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNoContent, w.Code)
	mockService.AssertExpectations(t)
}

// Tests for TransitionStatus — the one status endpoint (T267)
func TestJobHandler_TransitionStatus_Success(t *testing.T) {
	mockService := new(MockJobService)
	handler := NewJobHandler(mockService)
	router := setupJobTestRouter()

	userID, jobID := uuid.New(), uuid.New()
	router.POST("/jobs/:id/transition", func(c *gin.Context) {
		c.Set("user_id", userID)
		c.Set("userRole", models.RoleUser)
		handler.TransitionStatus(c)
	})

	response := createTestJobResponse()
	response.ID = jobID
	response.Status = models.JobStatusOnHold
	response.AllowedTransitions = models.GetAllowedTransitions(models.JobStatusOnHold)
	notes := "waiting on parts"
	mockService.On("TransitionStatus", jobID, models.JobStatusOnHold, "technical_issue", &notes, userID, models.RoleUser).
		Return(response, nil)

	body, _ := json.Marshal(map[string]string{"to_status": "on_hold", "reason": "technical_issue", "notes": notes})
	req := httptest.NewRequest(http.MethodPost, "/jobs/"+jobID.String()+"/transition", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	var respBody models.JobResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &respBody))
	assert.Equal(t, models.JobStatusOnHold, respBody.Status)
	assert.ElementsMatch(t, []models.JobStatus{models.JobStatusInProgress, models.JobStatusCancelled}, respBody.AllowedTransitions)
	mockService.AssertExpectations(t)
}

// T223: an invalid status transition reaches the client as a 409 with its
// own message, and an unexpected error is a 500 that leaks nothing.
func TestJobHandler_TransitionStatus_DomainAndInfraErrors(t *testing.T) {
	for _, tc := range []struct {
		serviceErr error
		status     int
		code       string
		message    string
	}{
		{fmt.Errorf("transition: %w", models.ErrInvalidStatusTransition.WithMessage("A job can't move from completed to in_progress")),
			http.StatusConflict, "invalid_status_transition", "A job can't move from completed to in_progress"},
		{services.ErrJobNotFound, http.StatusNotFound, "job_not_found", "Job not found"},
		{services.ErrUnauthorized, http.StatusForbidden, "forbidden", ""},
		{errors.New("pq: deadlock detected"), http.StatusInternalServerError, "internal_error", "Failed to change job status"},
	} {
		mockService := new(MockJobService)
		handler := NewJobHandler(mockService)
		router := setupJobTestRouter()
		router.Use(middleware.RequestID())
		userID, jobID := uuid.New(), uuid.New()
		router.POST("/jobs/:id/transition", func(c *gin.Context) {
			c.Set("user_id", userID)
			c.Set("userRole", models.RoleUser)
			handler.TransitionStatus(c)
		})
		mockService.On("TransitionStatus", jobID, models.JobStatusInProgress, "", (*string)(nil), userID, models.RoleUser).
			Return(nil, tc.serviceErr)

		req := httptest.NewRequest(http.MethodPost, "/jobs/"+jobID.String()+"/transition",
			bytes.NewBufferString(`{"to_status":"in_progress"}`))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		var body models.ErrorResponse
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
		assert.Equal(t, tc.status, w.Code, tc.serviceErr.Error())
		assert.Equal(t, tc.code, body.Error)
		if tc.message != "" {
			assert.Equal(t, tc.message, body.Message)
		}
		assert.NotContains(t, body.Message, "pq:")
		assert.NotEmpty(t, body.RequestID)
	}
}

// Tests for AssignTechnician
func TestJobHandler_AssignTechnician_Success(t *testing.T) {
	mockService := new(MockJobService)
	handler := NewJobHandler(mockService)
	router := setupJobTestRouter()

	userID := uuid.New()
	jobID := uuid.New()
	techID := uuid.New()

	router.POST("/jobs/:id/assign", func(c *gin.Context) {
		c.Set("user_id", userID)
		c.Set("userRole", models.RoleUser)
		handler.AssignTechnician(c)
	})

	reqBody := map[string]interface{}{
		"technician_id": techID.String(),
		"role":          "Lead Technician",
	}

	mockService.On("AssignTechnician", jobID, techID, "Lead Technician", userID, models.RoleUser).
		Return(nil)

	jsonBody, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/jobs/"+jobID.String()+"/assign", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	mockService.AssertExpectations(t)
}

// Tests for GetJobStats
func TestJobHandler_GetJobStats_Success(t *testing.T) {
	mockService := new(MockJobService)
	handler := NewJobHandler(mockService)
	router := setupJobTestRouter()

	userID := uuid.New()
	router.GET("/jobs/stats", func(c *gin.Context) {
		c.Set("user_id", userID)
		c.Set("userRole", models.RoleUser)
		handler.GetJobStats(c)
	})

	stats := models.NewJobStats(map[string]int64{"new": 4, "scheduled": 30, "en_route": 1, "in_progress": 20})

	mockService.On("GetJobStats").Return(stats, nil)

	req := httptest.NewRequest(http.MethodGet, "/jobs/stats", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	// The dashboard reads these exact keys (T257).
	var respBody map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &respBody)
	require.NoError(t, err)
	assert.Equal(t, float64(55), respBody["total_jobs"])
	assert.Equal(t, float64(4), respBody["pending_jobs"])
	assert.Equal(t, float64(30), respBody["scheduled_jobs"])
	assert.Equal(t, float64(21), respBody["in_progress_jobs"])
	mockService.AssertExpectations(t)
}

// WithContext implements the interface; the mock ignores the context.
func (m *MockJobService) WithContext(_ context.Context) services.JobServiceInterface { return m }

// T233: an explicit billable=false must reach the service as false; an absent
// billable field still defaults to billable.
func TestJobHandler_AddMaterial_Billable(t *testing.T) {
	cases := []struct {
		name string
		body string
		want bool
	}{
		{"explicit false", `{"name":"Pipe","quantity":1,"unit_cost":5,"billable":false}`, false},
		{"explicit true", `{"name":"Pipe","quantity":1,"unit_cost":5,"billable":true}`, true},
		{"absent defaults to true", `{"name":"Pipe","quantity":1,"unit_cost":5}`, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			mockService := new(MockJobService)
			handler := NewJobHandler(mockService)
			router := setupJobTestRouter()
			userID := uuid.New()
			jobID := uuid.New()
			router.POST("/jobs/:id/materials", func(c *gin.Context) {
				c.Set("user_id", userID)
				handler.AddMaterial(c)
			})

			var got *models.JobMaterial
			mockService.On("AddMaterial", jobID, mock.AnythingOfType("*models.JobMaterial"), userID, mock.Anything).
				Run(func(args mock.Arguments) { got = args.Get(1).(*models.JobMaterial) }).
				Return(nil)

			req := httptest.NewRequest(http.MethodPost, "/jobs/"+jobID.String()+"/materials", bytes.NewBufferString(tc.body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			require.NotNil(t, got, "service not called (status %d: %s)", w.Code, w.Body.String())
			assert.Equal(t, tc.want, got.Billable)
		})
	}
}
