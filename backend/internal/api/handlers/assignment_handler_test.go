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

	"github.com/javaknight1/servicepro/backend/internal/models"
	"github.com/javaknight1/servicepro/backend/internal/services"
	"github.com/javaknight1/servicepro/backend/pkg/notifications"
)

// MockAssignmentService is a mock implementation of AssignmentServiceInterface
type MockAssignmentService struct {
	mock.Mock
}

func (m *MockAssignmentService) CreateAssignment(ctx context.Context, req *services.AssignmentRequest, assignedBy uuid.UUID) (*models.JobAssignment, error) {
	args := m.Called(ctx, req, assignedBy)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.JobAssignment), args.Error(1)
}

func (m *MockAssignmentService) BulkAssignTechnicians(ctx context.Context, req *services.BulkAssignmentRequest, assignedBy uuid.UUID) ([]models.JobAssignment, []error) {
	args := m.Called(ctx, req, assignedBy)
	return args.Get(0).([]models.JobAssignment), args.Get(1).([]error)
}

func (m *MockAssignmentService) RemoveAssignment(ctx context.Context, assignmentID uuid.UUID, removedBy uuid.UUID) error {
	args := m.Called(ctx, assignmentID, removedBy)
	return args.Error(0)
}

func (m *MockAssignmentService) UpdateAssignment(ctx context.Context, assignmentID uuid.UUID, role string, updatedBy uuid.UUID) error {
	args := m.Called(ctx, assignmentID, role, updatedBy)
	return args.Error(0)
}

func (m *MockAssignmentService) CheckTechnicianAvailability(ctx context.Context, technicianID uuid.UUID, startTime, endTime time.Time) (*services.TechnicianAvailability, error) {
	args := m.Called(ctx, technicianID, startTime, endTime)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*services.TechnicianAvailability), args.Error(1)
}

func (m *MockAssignmentService) GetAvailableTechnicians(ctx context.Context, req *services.AvailabilityCheckRequest) ([]*services.TechnicianAvailability, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*services.TechnicianAvailability), args.Error(1)
}

func (m *MockAssignmentService) DetectConflicts(ctx context.Context, technicianID uuid.UUID, startTime, endTime time.Time, excludeJobID *uuid.UUID) ([]models.Job, error) {
	args := m.Called(ctx, technicianID, startTime, endTime, excludeJobID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]models.Job), args.Error(1)
}

func (m *MockAssignmentService) CheckSkillMatch(ctx context.Context, technicianID uuid.UUID, requiredSkills []services.TechnicianSkill) (float64, error) {
	args := m.Called(ctx, technicianID, requiredSkills)
	return args.Get(0).(float64), args.Error(1)
}

func (m *MockAssignmentService) OptimizeAssignments(ctx context.Context, jobID uuid.UUID, requiredSkills []services.TechnicianSkill, startTime, endTime time.Time) ([]*services.TechnicianAvailability, error) {
	args := m.Called(ctx, jobID, requiredSkills, startTime, endTime)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*services.TechnicianAvailability), args.Error(1)
}

func (m *MockAssignmentService) GetTechnicianWorkload(ctx context.Context, technicianID uuid.UUID, startDate, endDate time.Time) (int, error) {
	args := m.Called(ctx, technicianID, startDate, endDate)
	return args.Get(0).(int), args.Error(1)
}

func (m *MockAssignmentService) GetTechniciansBySkill(ctx context.Context, skill services.TechnicianSkill) ([]uuid.UUID, error) {
	args := m.Called(ctx, skill)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]uuid.UUID), args.Error(1)
}

// MockNotificationService is a mock implementation of NotificationServiceInterface
type MockNotificationService struct {
	mock.Mock
}

func (m *MockNotificationService) SendNotification(ctx context.Context, req *notifications.NotificationRequest) (*notifications.NotificationResult, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*notifications.NotificationResult), args.Error(1)
}

func (m *MockNotificationService) SendMultiChannelNotification(ctx context.Context, reqs []*notifications.NotificationRequest) ([]*notifications.NotificationResult, error) {
	args := m.Called(ctx, reqs)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*notifications.NotificationResult), args.Error(1)
}

func (m *MockNotificationService) SendAssignmentCreatedNotification(ctx context.Context, assignment *models.JobAssignment, job *models.Job, technician *models.User) error {
	args := m.Called(ctx, assignment, job, technician)
	return args.Error(0)
}

func (m *MockNotificationService) SendAssignmentUpdatedNotification(ctx context.Context, assignment *models.JobAssignment, job *models.Job, technician *models.User, changes string) error {
	args := m.Called(ctx, assignment, job, technician, changes)
	return args.Error(0)
}

func (m *MockNotificationService) SendAssignmentRemovedNotification(ctx context.Context, technicianEmail, technicianName, jobNumber, jobTitle string) error {
	args := m.Called(ctx, technicianEmail, technicianName, jobNumber, jobTitle)
	return args.Error(0)
}

func (m *MockNotificationService) SendBulkNotifications(ctx context.Context, reqs []*notifications.NotificationRequest) ([]*notifications.NotificationResult, []error) {
	args := m.Called(ctx, reqs)
	return args.Get(0).([]*notifications.NotificationResult), args.Get(1).([]error)
}

// Helper function to setup authenticated test context
func setupAuthContext(router *gin.Engine, userID uuid.UUID, userRole models.UserRole) *gin.Engine {
	router.Use(func(c *gin.Context) {
		c.Set("userID", userID)
		c.Set("userRole", userRole)
		c.Next()
	})
	return router
}

// TestCreateAssignment_Success tests successful assignment creation
func TestCreateAssignment_Success(t *testing.T) {
	mockAssignmentService := new(MockAssignmentService)
	mockNotificationService := new(MockNotificationService)
	mockJobService := new(MockJobService)
	handler := NewAssignmentHandler(mockAssignmentService, mockNotificationService, mockJobService)

	userID := uuid.New()
	jobID := uuid.New()
	technicianID := uuid.New()

	req := services.AssignmentRequest{
		JobID:        jobID,
		TechnicianID: technicianID,
		Role:         "Lead Technician",
	}

	expectedAssignment := &models.JobAssignment{
		ID:         uuid.New(),
		JobID:      jobID,
		UserID:     technicianID,
		Role:       "Lead Technician",
		AssignedAt: time.Now(),
	}

	mockAssignmentService.On("CreateAssignment", mock.Anything, &req, userID).Return(expectedAssignment, nil)

	router := setupTestRouter()
	router = setupAuthContext(router, userID, models.UserRoleAdmin)
	router.POST("/assignments", handler.CreateAssignment)

	body, _ := json.Marshal(req)
	httpReq, _ := http.NewRequest("POST", "/assignments", bytes.NewBuffer(body))
	httpReq.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, httpReq)

	assert.Equal(t, http.StatusCreated, w.Code)

	var response models.JobAssignment
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, expectedAssignment.JobID, response.JobID)
	assert.Equal(t, expectedAssignment.UserID, response.UserID)

	mockAssignmentService.AssertExpectations(t)
}

// TestCreateAssignment_Unauthorized tests unauthorized access
func TestCreateAssignment_Unauthorized(t *testing.T) {
	mockAssignmentService := new(MockAssignmentService)
	mockNotificationService := new(MockNotificationService)
	mockJobService := new(MockJobService)
	handler := NewAssignmentHandler(mockAssignmentService, mockNotificationService, mockJobService)

	router := setupTestRouter()
	router.POST("/assignments", handler.CreateAssignment)

	req := services.AssignmentRequest{
		JobID:        uuid.New(),
		TechnicianID: uuid.New(),
		Role:         "Technician",
	}

	body, _ := json.Marshal(req)
	httpReq, _ := http.NewRequest("POST", "/assignments", bytes.NewBuffer(body))
	httpReq.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, httpReq)

	assert.Equal(t, http.StatusUnauthorized, w.Code)

	var response models.ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "unauthorized", response.Error)
}

// TestCreateAssignment_Forbidden tests forbidden access for non-admin/manager
func TestCreateAssignment_Forbidden(t *testing.T) {
	mockAssignmentService := new(MockAssignmentService)
	mockNotificationService := new(MockNotificationService)
	mockJobService := new(MockJobService)
	handler := NewAssignmentHandler(mockAssignmentService, mockNotificationService, mockJobService)

	router := setupTestRouter()
	router = setupAuthContext(router, uuid.New(), models.UserRoleTechnician)
	router.POST("/assignments", handler.CreateAssignment)

	req := services.AssignmentRequest{
		JobID:        uuid.New(),
		TechnicianID: uuid.New(),
		Role:         "Technician",
	}

	body, _ := json.Marshal(req)
	httpReq, _ := http.NewRequest("POST", "/assignments", bytes.NewBuffer(body))
	httpReq.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, httpReq)

	assert.Equal(t, http.StatusForbidden, w.Code)

	var response models.ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "forbidden", response.Error)
}

// TestCreateAssignment_InvalidRequest tests invalid request data
func TestCreateAssignment_InvalidRequest(t *testing.T) {
	mockAssignmentService := new(MockAssignmentService)
	mockNotificationService := new(MockNotificationService)
	mockJobService := new(MockJobService)
	handler := NewAssignmentHandler(mockAssignmentService, mockNotificationService, mockJobService)

	router := setupTestRouter()
	router = setupAuthContext(router, uuid.New(), models.UserRoleAdmin)
	router.POST("/assignments", handler.CreateAssignment)

	httpReq, _ := http.NewRequest("POST", "/assignments", bytes.NewBuffer([]byte("{invalid-json")))
	httpReq.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, httpReq)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var response models.ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "invalid_request", response.Error)
}

// TestCreateAssignment_JobNotFound tests job not found error
func TestCreateAssignment_JobNotFound(t *testing.T) {
	mockAssignmentService := new(MockAssignmentService)
	mockNotificationService := new(MockNotificationService)
	mockJobService := new(MockJobService)
	handler := NewAssignmentHandler(mockAssignmentService, mockNotificationService, mockJobService)

	userID := uuid.New()
	req := services.AssignmentRequest{
		JobID:        uuid.New(),
		TechnicianID: uuid.New(),
		Role:         "Technician",
	}

	mockAssignmentService.On("CreateAssignment", mock.Anything, &req, userID).Return(nil, services.ErrJobNotFound)

	router := setupTestRouter()
	router = setupAuthContext(router, userID, models.UserRoleAdmin)
	router.POST("/assignments", handler.CreateAssignment)

	body, _ := json.Marshal(req)
	httpReq, _ := http.NewRequest("POST", "/assignments", bytes.NewBuffer(body))
	httpReq.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, httpReq)

	assert.Equal(t, http.StatusNotFound, w.Code)

	var response models.ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "job_not_found", response.Error)

	mockAssignmentService.AssertExpectations(t)
}

// TestCreateAssignment_TechnicianNotFound tests technician not found error
func TestCreateAssignment_TechnicianNotFound(t *testing.T) {
	mockAssignmentService := new(MockAssignmentService)
	mockNotificationService := new(MockNotificationService)
	mockJobService := new(MockJobService)
	handler := NewAssignmentHandler(mockAssignmentService, mockNotificationService, mockJobService)

	userID := uuid.New()
	req := services.AssignmentRequest{
		JobID:        uuid.New(),
		TechnicianID: uuid.New(),
		Role:         "Technician",
	}

	mockAssignmentService.On("CreateAssignment", mock.Anything, &req, userID).Return(nil, services.ErrTechnicianNotFound)

	router := setupTestRouter()
	router = setupAuthContext(router, userID, models.UserRoleAdmin)
	router.POST("/assignments", handler.CreateAssignment)

	body, _ := json.Marshal(req)
	httpReq, _ := http.NewRequest("POST", "/assignments", bytes.NewBuffer(body))
	httpReq.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, httpReq)

	assert.Equal(t, http.StatusNotFound, w.Code)

	var response models.ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "technician_not_found", response.Error)

	mockAssignmentService.AssertExpectations(t)
}

// TestCreateAssignment_TechnicianNotAvailable tests technician not available error
func TestCreateAssignment_TechnicianNotAvailable(t *testing.T) {
	mockAssignmentService := new(MockAssignmentService)
	mockNotificationService := new(MockNotificationService)
	mockJobService := new(MockJobService)
	handler := NewAssignmentHandler(mockAssignmentService, mockNotificationService, mockJobService)

	userID := uuid.New()
	req := services.AssignmentRequest{
		JobID:        uuid.New(),
		TechnicianID: uuid.New(),
		Role:         "Technician",
	}

	mockAssignmentService.On("CreateAssignment", mock.Anything, &req, userID).Return(nil, services.ErrTechnicianNotAvailable)

	router := setupTestRouter()
	router = setupAuthContext(router, userID, models.UserRoleAdmin)
	router.POST("/assignments", handler.CreateAssignment)

	body, _ := json.Marshal(req)
	httpReq, _ := http.NewRequest("POST", "/assignments", bytes.NewBuffer(body))
	httpReq.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, httpReq)

	assert.Equal(t, http.StatusConflict, w.Code)

	var response models.ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "technician_not_available", response.Error)

	mockAssignmentService.AssertExpectations(t)
}

// TestCreateAssignment_SkillMismatch tests skill mismatch error
func TestCreateAssignment_SkillMismatch(t *testing.T) {
	mockAssignmentService := new(MockAssignmentService)
	mockNotificationService := new(MockNotificationService)
	mockJobService := new(MockJobService)
	handler := NewAssignmentHandler(mockAssignmentService, mockNotificationService, mockJobService)

	userID := uuid.New()
	req := services.AssignmentRequest{
		JobID:        uuid.New(),
		TechnicianID: uuid.New(),
		Role:         "Technician",
	}

	mockAssignmentService.On("CreateAssignment", mock.Anything, &req, userID).Return(nil, services.ErrSkillMismatch)

	router := setupTestRouter()
	router = setupAuthContext(router, userID, models.UserRoleAdmin)
	router.POST("/assignments", handler.CreateAssignment)

	body, _ := json.Marshal(req)
	httpReq, _ := http.NewRequest("POST", "/assignments", bytes.NewBuffer(body))
	httpReq.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, httpReq)

	assert.Equal(t, http.StatusConflict, w.Code)

	var response models.ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "skill_mismatch", response.Error)

	mockAssignmentService.AssertExpectations(t)
}

// TestCreateAssignment_InternalError tests internal error
func TestCreateAssignment_InternalError(t *testing.T) {
	mockAssignmentService := new(MockAssignmentService)
	mockNotificationService := new(MockNotificationService)
	mockJobService := new(MockJobService)
	handler := NewAssignmentHandler(mockAssignmentService, mockNotificationService, mockJobService)

	userID := uuid.New()
	req := services.AssignmentRequest{
		JobID:        uuid.New(),
		TechnicianID: uuid.New(),
		Role:         "Technician",
	}

	mockAssignmentService.On("CreateAssignment", mock.Anything, &req, userID).Return(nil, errors.New("database error"))

	router := setupTestRouter()
	router = setupAuthContext(router, userID, models.UserRoleAdmin)
	router.POST("/assignments", handler.CreateAssignment)

	body, _ := json.Marshal(req)
	httpReq, _ := http.NewRequest("POST", "/assignments", bytes.NewBuffer(body))
	httpReq.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, httpReq)

	assert.Equal(t, http.StatusInternalServerError, w.Code)

	var response models.ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "internal_error", response.Error)

	mockAssignmentService.AssertExpectations(t)
}

// TestBulkCreateAssignments_Success tests successful bulk assignment creation
func TestBulkCreateAssignments_Success(t *testing.T) {
	mockAssignmentService := new(MockAssignmentService)
	mockNotificationService := new(MockNotificationService)
	mockJobService := new(MockJobService)
	handler := NewAssignmentHandler(mockAssignmentService, mockNotificationService, mockJobService)

	userID := uuid.New()
	jobID := uuid.New()

	req := services.BulkAssignmentRequest{
		JobID: jobID,
		Assignments: []services.AssignmentRequest{
			{TechnicianID: uuid.New(), Role: "Lead"},
			{TechnicianID: uuid.New(), Role: "Assistant"},
		},
	}

	assignments := []models.JobAssignment{
		{ID: uuid.New(), JobID: jobID, UserID: req.Assignments[0].TechnicianID, Role: "Lead"},
		{ID: uuid.New(), JobID: jobID, UserID: req.Assignments[1].TechnicianID, Role: "Assistant"},
	}

	mockAssignmentService.On("BulkAssignTechnicians", mock.Anything, &req, userID).Return(assignments, []error{})

	router := setupTestRouter()
	router = setupAuthContext(router, userID, models.UserRoleManager)
	router.POST("/assignments/bulk", handler.BulkCreateAssignments)

	body, _ := json.Marshal(req)
	httpReq, _ := http.NewRequest("POST", "/assignments/bulk", bytes.NewBuffer(body))
	httpReq.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, httpReq)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, float64(2), response["succeeded"])
	assert.Equal(t, float64(0), response["failed"])

	mockAssignmentService.AssertExpectations(t)
}

// TestBulkCreateAssignments_PartialSuccess tests partial success with some errors
func TestBulkCreateAssignments_PartialSuccess(t *testing.T) {
	mockAssignmentService := new(MockAssignmentService)
	mockNotificationService := new(MockNotificationService)
	mockJobService := new(MockJobService)
	handler := NewAssignmentHandler(mockAssignmentService, mockNotificationService, mockJobService)

	userID := uuid.New()
	jobID := uuid.New()

	req := services.BulkAssignmentRequest{
		JobID: jobID,
		Assignments: []services.AssignmentRequest{
			{TechnicianID: uuid.New(), Role: "Lead"},
			{TechnicianID: uuid.New(), Role: "Assistant"},
		},
	}

	assignments := []models.JobAssignment{
		{ID: uuid.New(), JobID: jobID, UserID: req.Assignments[0].TechnicianID, Role: "Lead"},
	}

	errs := []error{
		fmt.Errorf("technician %s: not available", req.Assignments[1].TechnicianID),
	}

	mockAssignmentService.On("BulkAssignTechnicians", mock.Anything, &req, userID).Return(assignments, errs)

	router := setupTestRouter()
	router = setupAuthContext(router, userID, models.UserRoleAdmin)
	router.POST("/assignments/bulk", handler.BulkCreateAssignments)

	body, _ := json.Marshal(req)
	httpReq, _ := http.NewRequest("POST", "/assignments/bulk", bytes.NewBuffer(body))
	httpReq.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, httpReq)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, float64(1), response["succeeded"])
	assert.Equal(t, float64(1), response["failed"])

	mockAssignmentService.AssertExpectations(t)
}

// TestRemoveAssignment_Success tests successful assignment removal
func TestRemoveAssignment_Success(t *testing.T) {
	mockAssignmentService := new(MockAssignmentService)
	mockNotificationService := new(MockNotificationService)
	mockJobService := new(MockJobService)
	handler := NewAssignmentHandler(mockAssignmentService, mockNotificationService, mockJobService)

	userID := uuid.New()
	assignmentID := uuid.New()

	mockAssignmentService.On("RemoveAssignment", mock.Anything, assignmentID, userID).Return(nil)

	router := setupTestRouter()
	router = setupAuthContext(router, userID, models.UserRoleAdmin)
	router.DELETE("/assignments/:id", handler.RemoveAssignment)

	httpReq, _ := http.NewRequest("DELETE", "/assignments/"+assignmentID.String(), nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, httpReq)

	assert.Equal(t, http.StatusNoContent, w.Code)

	mockAssignmentService.AssertExpectations(t)
}

// TestRemoveAssignment_InvalidID tests invalid assignment ID
func TestRemoveAssignment_InvalidID(t *testing.T) {
	mockAssignmentService := new(MockAssignmentService)
	mockNotificationService := new(MockNotificationService)
	mockJobService := new(MockJobService)
	handler := NewAssignmentHandler(mockAssignmentService, mockNotificationService, mockJobService)

	router := setupTestRouter()
	router = setupAuthContext(router, uuid.New(), models.UserRoleAdmin)
	router.DELETE("/assignments/:id", handler.RemoveAssignment)

	httpReq, _ := http.NewRequest("DELETE", "/assignments/invalid-uuid", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, httpReq)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var response models.ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "invalid_id", response.Error)
}

// TestRemoveAssignment_NotFound tests assignment not found error
func TestRemoveAssignment_NotFound(t *testing.T) {
	mockAssignmentService := new(MockAssignmentService)
	mockNotificationService := new(MockNotificationService)
	mockJobService := new(MockJobService)
	handler := NewAssignmentHandler(mockAssignmentService, mockNotificationService, mockJobService)

	userID := uuid.New()
	assignmentID := uuid.New()

	mockAssignmentService.On("RemoveAssignment", mock.Anything, assignmentID, userID).Return(services.ErrAssignmentNotFound)

	router := setupTestRouter()
	router = setupAuthContext(router, userID, models.UserRoleAdmin)
	router.DELETE("/assignments/:id", handler.RemoveAssignment)

	httpReq, _ := http.NewRequest("DELETE", "/assignments/"+assignmentID.String(), nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, httpReq)

	assert.Equal(t, http.StatusNotFound, w.Code)

	var response models.ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "assignment_not_found", response.Error)

	mockAssignmentService.AssertExpectations(t)
}

// TestUpdateAssignment_Success tests successful assignment update
func TestUpdateAssignment_Success(t *testing.T) {
	mockAssignmentService := new(MockAssignmentService)
	mockNotificationService := new(MockNotificationService)
	mockJobService := new(MockJobService)
	handler := NewAssignmentHandler(mockAssignmentService, mockNotificationService, mockJobService)

	userID := uuid.New()
	assignmentID := uuid.New()

	updateReq := map[string]string{
		"role": "Senior Technician",
	}

	mockAssignmentService.On("UpdateAssignment", mock.Anything, assignmentID, "Senior Technician", userID).Return(nil)

	router := setupTestRouter()
	router = setupAuthContext(router, userID, models.UserRoleManager)
	router.PUT("/assignments/:id", handler.UpdateAssignment)

	body, _ := json.Marshal(updateReq)
	httpReq, _ := http.NewRequest("PUT", "/assignments/"+assignmentID.String(), bytes.NewBuffer(body))
	httpReq.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, httpReq)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]string
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Contains(t, response["message"], "successfully")

	mockAssignmentService.AssertExpectations(t)
}

// TestUpdateAssignment_InvalidRequest tests invalid update request
func TestUpdateAssignment_InvalidRequest(t *testing.T) {
	mockAssignmentService := new(MockAssignmentService)
	mockNotificationService := new(MockNotificationService)
	mockJobService := new(MockJobService)
	handler := NewAssignmentHandler(mockAssignmentService, mockNotificationService, mockJobService)

	router := setupTestRouter()
	router = setupAuthContext(router, uuid.New(), models.UserRoleAdmin)
	router.PUT("/assignments/:id", handler.UpdateAssignment)

	assignmentID := uuid.New()
	httpReq, _ := http.NewRequest("PUT", "/assignments/"+assignmentID.String(), bytes.NewBuffer([]byte("{}")))
	httpReq.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, httpReq)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var response models.ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "invalid_request", response.Error)
}

// TestCheckAvailability_Success tests successful availability check
func TestCheckAvailability_Success(t *testing.T) {
	mockAssignmentService := new(MockAssignmentService)
	mockNotificationService := new(MockNotificationService)
	mockJobService := new(MockJobService)
	handler := NewAssignmentHandler(mockAssignmentService, mockNotificationService, mockJobService)

	technicianID := uuid.New()
	startTime := time.Now().Add(24 * time.Hour)
	endTime := startTime.Add(2 * time.Hour)

	req := map[string]interface{}{
		"technician_id": technicianID.String(),
		"start_time":    startTime.Format(time.RFC3339),
		"end_time":      endTime.Format(time.RFC3339),
	}

	availability := &services.TechnicianAvailability{
		TechnicianID: technicianID,
		Available:    true,
		Conflicts:    []string{},
		Workload:     2,
	}

	mockAssignmentService.On("CheckTechnicianAvailability", mock.Anything, technicianID, mock.AnythingOfType("time.Time"), mock.AnythingOfType("time.Time")).Return(availability, nil)

	router := setupTestRouter()
	router = setupAuthContext(router, uuid.New(), models.UserRoleAdmin)
	router.POST("/availability/check", handler.CheckAvailability)

	body, _ := json.Marshal(req)
	httpReq, _ := http.NewRequest("POST", "/availability/check", bytes.NewBuffer(body))
	httpReq.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, httpReq)

	assert.Equal(t, http.StatusOK, w.Code)

	var response services.TechnicianAvailability
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.True(t, response.Available)

	mockAssignmentService.AssertExpectations(t)
}

// TestGetAvailableTechnicians_Success tests successful search for available technicians
func TestGetAvailableTechnicians_Success(t *testing.T) {
	mockAssignmentService := new(MockAssignmentService)
	mockNotificationService := new(MockNotificationService)
	mockJobService := new(MockJobService)
	handler := NewAssignmentHandler(mockAssignmentService, mockNotificationService, mockJobService)

	startTime := time.Now().Add(24 * time.Hour)
	endTime := startTime.Add(2 * time.Hour)

	req := services.AvailabilityCheckRequest{
		StartTime:      startTime,
		EndTime:        endTime,
		RequiredSkills: []services.TechnicianSkill{services.SkillHVACInstallation},
	}

	technicians := []*services.TechnicianAvailability{
		{
			TechnicianID: uuid.New(),
			Available:    true,
			Conflicts:    []string{},
			Workload:     1,
		},
		{
			TechnicianID: uuid.New(),
			Available:    true,
			Conflicts:    []string{},
			Workload:     2,
		},
	}

	mockAssignmentService.On("GetAvailableTechnicians", mock.Anything, mock.AnythingOfType("*services.AvailabilityCheckRequest")).Return(technicians, nil)

	router := setupTestRouter()
	router = setupAuthContext(router, uuid.New(), models.UserRoleManager)
	router.POST("/availability/search", handler.GetAvailableTechnicians)

	body, _ := json.Marshal(req)
	httpReq, _ := http.NewRequest("POST", "/availability/search", bytes.NewBuffer(body))
	httpReq.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, httpReq)

	assert.Equal(t, http.StatusOK, w.Code)

	var response []*services.TechnicianAvailability
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Len(t, response, 2)

	mockAssignmentService.AssertExpectations(t)
}

// TestOptimizeAssignments_Success tests successful optimization
func TestOptimizeAssignments_Success(t *testing.T) {
	mockAssignmentService := new(MockAssignmentService)
	mockNotificationService := new(MockNotificationService)
	mockJobService := new(MockJobService)
	handler := NewAssignmentHandler(mockAssignmentService, mockNotificationService, mockJobService)

	jobID := uuid.New()
	startTime := time.Now().Add(24 * time.Hour)
	endTime := startTime.Add(2 * time.Hour)

	req := map[string]interface{}{
		"job_id":          jobID.String(),
		"required_skills": []string{"hvac_installation"},
		"start_time":      startTime.Format(time.RFC3339),
		"end_time":        endTime.Format(time.RFC3339),
	}

	technicians := []*services.TechnicianAvailability{
		{
			TechnicianID: uuid.New(),
			Available:    true,
			SkillMatch:   100.0,
			Workload:     1,
		},
	}

	mockAssignmentService.On("OptimizeAssignments", mock.Anything, jobID, mock.AnythingOfType("[]services.TechnicianSkill"), mock.AnythingOfType("time.Time"), mock.AnythingOfType("time.Time")).Return(technicians, nil)

	router := setupTestRouter()
	router = setupAuthContext(router, uuid.New(), models.UserRoleAdmin)
	router.POST("/optimize", handler.OptimizeAssignments)

	body, _ := json.Marshal(req)
	httpReq, _ := http.NewRequest("POST", "/optimize", bytes.NewBuffer(body))
	httpReq.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, httpReq)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Contains(t, response, "recommended_technicians")

	mockAssignmentService.AssertExpectations(t)
}

// TestGetTechnicianWorkload_Success tests successful workload retrieval
func TestGetTechnicianWorkload_Success(t *testing.T) {
	mockAssignmentService := new(MockAssignmentService)
	mockNotificationService := new(MockNotificationService)
	mockJobService := new(MockJobService)
	handler := NewAssignmentHandler(mockAssignmentService, mockNotificationService, mockJobService)

	technicianID := uuid.New()

	mockAssignmentService.On("GetTechnicianWorkload", mock.Anything, technicianID, mock.AnythingOfType("time.Time"), mock.AnythingOfType("time.Time")).Return(5, nil)

	router := setupTestRouter()
	router = setupAuthContext(router, uuid.New(), models.UserRoleManager)
	router.GET("/workload/:technician_id", handler.GetTechnicianWorkload)

	httpReq, _ := http.NewRequest("GET", "/workload/"+technicianID.String(), nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, httpReq)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, float64(5), response["workload"])

	mockAssignmentService.AssertExpectations(t)
}

// TestGetTechnicianWorkload_InvalidID tests invalid technician ID
func TestGetTechnicianWorkload_InvalidID(t *testing.T) {
	mockAssignmentService := new(MockAssignmentService)
	mockNotificationService := new(MockNotificationService)
	mockJobService := new(MockJobService)
	handler := NewAssignmentHandler(mockAssignmentService, mockNotificationService, mockJobService)

	router := setupTestRouter()
	router = setupAuthContext(router, uuid.New(), models.UserRoleAdmin)
	router.GET("/workload/:technician_id", handler.GetTechnicianWorkload)

	httpReq, _ := http.NewRequest("GET", "/workload/invalid-uuid", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, httpReq)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var response models.ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "invalid_id", response.Error)
}
