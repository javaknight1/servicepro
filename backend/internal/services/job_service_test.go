package services

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/javaknight1/servicepro/backend/internal/models"
)

// MockJobRepository is a mock implementation of JobRepositoryInterface
type MockJobRepository struct {
	mock.Mock
}

func (m *MockJobRepository) Create(job *models.Job) error {
	args := m.Called(job)
	return args.Error(0)
}

func (m *MockJobRepository) GetByID(id uuid.UUID) (*models.Job, error) {
	args := m.Called(id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Job), args.Error(1)
}

func (m *MockJobRepository) GetByJobNumber(jobNumber string) (*models.Job, error) {
	args := m.Called(jobNumber)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Job), args.Error(1)
}

func (m *MockJobRepository) Update(job *models.Job) error {
	args := m.Called(job)
	return args.Error(0)
}

func (m *MockJobRepository) Delete(id uuid.UUID) error {
	args := m.Called(id)
	return args.Error(0)
}

func (m *MockJobRepository) List(filter *models.JobListFilter) ([]models.Job, int64, error) {
	args := m.Called(filter)
	return args.Get(0).([]models.Job), args.Get(1).(int64), args.Error(2)
}

func (m *MockJobRepository) GetByCustomer(customerID uuid.UUID, limit, offset int) ([]models.Job, int64, error) {
	args := m.Called(customerID, limit, offset)
	return args.Get(0).([]models.Job), args.Get(1).(int64), args.Error(2)
}

func (m *MockJobRepository) GetByStatus(status models.JobStatus) ([]models.Job, error) {
	args := m.Called(status)
	return args.Get(0).([]models.Job), args.Error(1)
}

func (m *MockJobRepository) GetByAssignedUser(userID uuid.UUID) ([]models.Job, error) {
	args := m.Called(userID)
	return args.Get(0).([]models.Job), args.Error(1)
}

func (m *MockJobRepository) GetScheduledJobs(start, end time.Time) ([]models.Job, error) {
	args := m.Called(start, end)
	return args.Get(0).([]models.Job), args.Error(1)
}

func (m *MockJobRepository) GetOverdueJobs() ([]models.Job, error) {
	args := m.Called()
	return args.Get(0).([]models.Job), args.Error(1)
}

func (m *MockJobRepository) GetJobsRequiringFollowUp() ([]models.Job, error) {
	args := m.Called()
	return args.Get(0).([]models.Job), args.Error(1)
}

func (m *MockJobRepository) AddAssignment(assignment *models.JobAssignment) error {
	args := m.Called(assignment)
	return args.Error(0)
}

func (m *MockJobRepository) RemoveAssignment(assignmentID uuid.UUID) error {
	args := m.Called(assignmentID)
	return args.Error(0)
}

func (m *MockJobRepository) GetJobAssignments(jobID uuid.UUID) ([]models.JobAssignment, error) {
	args := m.Called(jobID)
	return args.Get(0).([]models.JobAssignment), args.Error(1)
}

func (m *MockJobRepository) AddMaterial(material *models.JobMaterial) error {
	args := m.Called(material)
	return args.Error(0)
}

func (m *MockJobRepository) UpdateMaterial(material *models.JobMaterial) error {
	args := m.Called(material)
	return args.Error(0)
}

func (m *MockJobRepository) DeleteMaterial(materialID uuid.UUID) error {
	args := m.Called(materialID)
	return args.Error(0)
}

func (m *MockJobRepository) GetJobMaterials(jobID uuid.UUID) ([]models.JobMaterial, error) {
	args := m.Called(jobID)
	return args.Get(0).([]models.JobMaterial), args.Error(1)
}

func (m *MockJobRepository) AddNote(note *models.JobNote) error {
	args := m.Called(note)
	return args.Error(0)
}

func (m *MockJobRepository) GetJobNotes(jobID uuid.UUID, includeInternal bool) ([]models.JobNote, error) {
	args := m.Called(jobID, includeInternal)
	return args.Get(0).([]models.JobNote), args.Error(1)
}

func (m *MockJobRepository) GetJobStats() (map[string]interface{}, error) {
	args := m.Called()
	return args.Get(0).(map[string]interface{}), args.Error(1)
}

func (m *MockJobRepository) GetTechnicianWorkload(userID uuid.UUID) (map[string]interface{}, error) {
	args := m.Called(userID)
	return args.Get(0).(map[string]interface{}), args.Error(1)
}

// MockCustomerRepository is a mock implementation of CustomerRepositoryInterface
type MockCustomerRepository struct {
	mock.Mock
}

func (m *MockCustomerRepository) Create(customer *models.Customer) error {
	args := m.Called(customer)
	return args.Error(0)
}

func (m *MockCustomerRepository) GetByID(id uuid.UUID) (*models.Customer, error) {
	args := m.Called(id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Customer), args.Error(1)
}

func (m *MockCustomerRepository) GetByEmail(email string) (*models.Customer, error) {
	args := m.Called(email)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Customer), args.Error(1)
}

func (m *MockCustomerRepository) Update(customer *models.Customer) error {
	args := m.Called(customer)
	return args.Error(0)
}

func (m *MockCustomerRepository) Delete(id uuid.UUID) error {
	args := m.Called(id)
	return args.Error(0)
}

func (m *MockCustomerRepository) List(filter *models.CustomerListFilter) ([]models.Customer, int64, error) {
	args := m.Called(filter)
	return args.Get(0).([]models.Customer), args.Get(1).(int64), args.Error(2)
}

func (m *MockCustomerRepository) EmailExists(email string) (bool, error) {
	args := m.Called(email)
	return args.Bool(0), args.Error(1)
}

func (m *MockCustomerRepository) PhoneExists(phone string) (bool, error) {
	args := m.Called(phone)
	return args.Bool(0), args.Error(1)
}

func (m *MockCustomerRepository) Search(query string) ([]models.Customer, error) {
	args := m.Called(query)
	return args.Get(0).([]models.Customer), args.Error(1)
}

func (m *MockCustomerRepository) GetByStatus(status models.CustomerStatus) ([]models.Customer, error) {
	args := m.Called(status)
	return args.Get(0).([]models.Customer), args.Error(1)
}

func (m *MockCustomerRepository) GetByType(customerType models.CustomerType) ([]models.Customer, error) {
	args := m.Called(customerType)
	return args.Get(0).([]models.Customer), args.Error(1)
}

func (m *MockCustomerRepository) FullTextSearch(query string, customerType models.CustomerType, status models.CustomerStatus, city, state string, limit, offset int, sortBy, sortOrder string) ([]models.Customer, int64, error) {
	args := m.Called(query, customerType, status, city, state, limit, offset, sortBy, sortOrder)
	return args.Get(0).([]models.Customer), args.Get(1).(int64), args.Error(2)
}

func (m *MockCustomerRepository) Autocomplete(prefix string, limit int) ([]map[string]interface{}, error) {
	args := m.Called(prefix, limit)
	return args.Get(0).([]map[string]interface{}), args.Error(1)
}

func (m *MockCustomerRepository) GetSearchStats() (map[string]interface{}, error) {
	args := m.Called()
	return args.Get(0).(map[string]interface{}), args.Error(1)
}

// MockDB is a mock gorm.DB for testing
type MockDB struct {
	mock.Mock
}

// Helper functions to create test data
func createTestJob() *models.Job {
	now := time.Now()
	scheduledStart := now.Add(24 * time.Hour)
	scheduledEnd := scheduledStart.Add(2 * time.Hour)
	createdBy := uuid.New()

	return &models.Job{
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
		CreatedBy: createdBy,
		CreatedAt: now,
		UpdatedAt: now,
	}
}

func createTestCustomer() *models.Customer {
	return &models.Customer{
		ID:                   uuid.New(),
		FirstName:            "John",
		LastName:             "Doe",
		Email:                "john.doe@example.com",
		PhonePrimary:         "555-0100",
		BillingAddressStreet: "123 Main St",
		BillingAddressCity:   "Austin",
		BillingAddressState:  "TX",
		BillingAddressZip:    "78701",
		CustomerType:         models.CustomerTypeResidential,
		Status:               models.CustomerStatusActive,
	}
}

// Tests for CreateJob
func TestJobService_CreateJob_Success(t *testing.T) {
	mockJobRepo := new(MockJobRepository)
	mockCustomerRepo := new(MockCustomerRepository)
	service := NewJobService(mockJobRepo, mockCustomerRepo, nil)

	customerID := uuid.New()
	customer := createTestCustomer()
	customer.ID = customerID
	createdBy := uuid.New()

	now := time.Now()
	scheduledStart := now.Add(24 * time.Hour)
	scheduledEnd := scheduledStart.Add(2 * time.Hour)

	req := &models.CreateJobRequest{
		CustomerID:       customerID,
		Title:            "Installation Job",
		Description:      "Install HVAC system",
		JobType:          models.JobTypeInstallation,
		Priority:         models.JobPriorityNormal,
		ScheduledStartAt: &scheduledStart,
		ScheduledEndAt:   &scheduledEnd,
	}

	mockCustomerRepo.On("GetByID", customerID).Return(customer, nil)
	mockJobRepo.On("Create", mock.AnythingOfType("*models.Job")).Return(nil).Run(func(args mock.Arguments) {
		// Set the ID so GetByID can find it
		job := args.Get(0).(*models.Job)
		if job.ID == uuid.Nil {
			job.ID = uuid.New()
		}
	})
	mockJobRepo.On("GetByID", mock.AnythingOfType("uuid.UUID")).Return(&models.Job{
		ID:         uuid.New(),
		CustomerID: customerID,
		Title:      req.Title,
		JobType:    req.JobType,
		Priority:   models.JobPriorityNormal,
		Status:     models.JobStatusScheduled,
	}, nil)

	response, err := service.CreateJob(req, createdBy)

	require.NoError(t, err)
	assert.NotNil(t, response)
	assert.Equal(t, req.Title, response.Title)
	assert.Equal(t, req.JobType, response.JobType)
	mockCustomerRepo.AssertExpectations(t)
	mockJobRepo.AssertExpectations(t)
}

func TestJobService_CreateJob_CustomerNotFound(t *testing.T) {
	mockJobRepo := new(MockJobRepository)
	mockCustomerRepo := new(MockCustomerRepository)
	service := NewJobService(mockJobRepo, mockCustomerRepo, nil)

	customerID := uuid.New()
	createdBy := uuid.New()

	req := &models.CreateJobRequest{
		CustomerID: customerID,
		Title:      "Installation Job",
		JobType:    models.JobTypeInstallation,
	}

	mockCustomerRepo.On("GetByID", customerID).Return(nil, gorm.ErrRecordNotFound)

	response, err := service.CreateJob(req, createdBy)

	assert.Error(t, err)
	assert.Nil(t, response)
	assert.ErrorIs(t, err, ErrCustomerNotFound)
	mockCustomerRepo.AssertExpectations(t)
}

func TestJobService_CreateJob_InvalidData(t *testing.T) {
	mockJobRepo := new(MockJobRepository)
	mockCustomerRepo := new(MockCustomerRepository)
	service := NewJobService(mockJobRepo, mockCustomerRepo, nil)

	createdBy := uuid.New()

	// Missing required fields
	req := &models.CreateJobRequest{
		CustomerID: uuid.Nil, // Invalid
		Title:      "",       // Empty
	}

	response, err := service.CreateJob(req, createdBy)

	assert.Error(t, err)
	assert.Nil(t, response)
}

// Tests for GetJobByID
func TestJobService_GetJobByID_Success_Admin(t *testing.T) {
	mockJobRepo := new(MockJobRepository)
	mockCustomerRepo := new(MockCustomerRepository)
	service := NewJobService(mockJobRepo, mockCustomerRepo, nil)

	job := createTestJob()
	userID := uuid.New()

	mockJobRepo.On("GetByID", job.ID).Return(job, nil)

	response, err := service.GetJobByID(job.ID, userID, models.UserRoleAdmin)

	require.NoError(t, err)
	assert.NotNil(t, response)
	assert.Equal(t, job.ID, response.ID)
	mockJobRepo.AssertExpectations(t)
}

func TestJobService_GetJobByID_NotFound(t *testing.T) {
	mockJobRepo := new(MockJobRepository)
	mockCustomerRepo := new(MockCustomerRepository)
	service := NewJobService(mockJobRepo, mockCustomerRepo, nil)

	jobID := uuid.New()
	userID := uuid.New()

	mockJobRepo.On("GetByID", jobID).Return(nil, gorm.ErrRecordNotFound)

	response, err := service.GetJobByID(jobID, userID, models.UserRoleAdmin)

	assert.Error(t, err)
	assert.Nil(t, response)
	assert.ErrorIs(t, err, ErrJobNotFound)
	mockJobRepo.AssertExpectations(t)
}

func TestJobService_GetJobByID_TechnicianUnauthorized(t *testing.T) {
	mockJobRepo := new(MockJobRepository)
	mockCustomerRepo := new(MockCustomerRepository)
	service := NewJobService(mockJobRepo, mockCustomerRepo, nil)

	job := createTestJob()
	technicianID := uuid.New()

	mockJobRepo.On("GetByID", job.ID).Return(job, nil)

	response, err := service.GetJobByID(job.ID, technicianID, models.UserRoleTechnician)

	assert.Error(t, err)
	assert.Nil(t, response)
	assert.ErrorIs(t, err, ErrUnauthorized)
	mockJobRepo.AssertExpectations(t)
}

// Tests for UpdateJob
func TestJobService_UpdateJob_Success(t *testing.T) {
	mockJobRepo := new(MockJobRepository)
	mockCustomerRepo := new(MockCustomerRepository)
	service := NewJobService(mockJobRepo, mockCustomerRepo, nil)

	job := createTestJob()
	userID := job.CreatedBy

	newTitle := "Updated Job Title"
	newPriority := models.JobPriorityHigh

	req := &models.UpdateJobRequest{
		Title:    &newTitle,
		Priority: &newPriority,
	}

	mockJobRepo.On("GetByID", job.ID).Return(job, nil)
	mockJobRepo.On("Update", mock.AnythingOfType("*models.Job")).Return(nil)

	response, err := service.UpdateJob(job.ID, req, userID, models.UserRoleTechnician)

	require.NoError(t, err)
	assert.NotNil(t, response)
	mockJobRepo.AssertExpectations(t)
}

func TestJobService_UpdateJob_Unauthorized(t *testing.T) {
	mockJobRepo := new(MockJobRepository)
	mockCustomerRepo := new(MockCustomerRepository)
	service := NewJobService(mockJobRepo, mockCustomerRepo, nil)

	job := createTestJob()
	differentUserID := uuid.New()

	req := &models.UpdateJobRequest{
		Title: stringPtr("New Title"),
	}

	mockJobRepo.On("GetByID", job.ID).Return(job, nil)

	response, err := service.UpdateJob(job.ID, req, differentUserID, models.UserRoleTechnician)

	assert.Error(t, err)
	assert.Nil(t, response)
	assert.ErrorIs(t, err, ErrUnauthorized)
	mockJobRepo.AssertExpectations(t)
}

// Tests for StartJob
func TestJobService_StartJob_Success(t *testing.T) {
	mockJobRepo := new(MockJobRepository)
	mockCustomerRepo := new(MockCustomerRepository)
	service := NewJobService(mockJobRepo, mockCustomerRepo, nil)

	job := createTestJob()
	job.Status = models.JobStatusScheduled

	mockJobRepo.On("GetByID", job.ID).Return(job, nil)
	mockJobRepo.On("Update", mock.AnythingOfType("*models.Job")).Return(nil)

	response, err := service.StartJob(job.ID, job.CreatedBy, models.UserRoleTechnician)

	require.NoError(t, err)
	assert.NotNil(t, response)
	mockJobRepo.AssertExpectations(t)
}

// Tests for CompleteJob
func TestJobService_CompleteJob_Success(t *testing.T) {
	mockJobRepo := new(MockJobRepository)
	mockCustomerRepo := new(MockCustomerRepository)
	service := NewJobService(mockJobRepo, mockCustomerRepo, nil)

	job := createTestJob()
	job.Status = models.JobStatusInProgress
	now := time.Now()
	job.ActualStartAt = &now // Set actual start time for completion

	mockJobRepo.On("GetByID", job.ID).Return(job, nil)
	mockJobRepo.On("Update", mock.AnythingOfType("*models.Job")).Return(nil)

	response, err := service.CompleteJob(job.ID, "Job completed successfully", job.CreatedBy, models.UserRoleTechnician)

	require.NoError(t, err)
	assert.NotNil(t, response)
	mockJobRepo.AssertExpectations(t)
}

// Tests for CancelJob
func TestJobService_CancelJob_AdminSuccess(t *testing.T) {
	mockJobRepo := new(MockJobRepository)
	mockCustomerRepo := new(MockCustomerRepository)
	service := NewJobService(mockJobRepo, mockCustomerRepo, nil)

	job := createTestJob()
	adminID := uuid.New()

	mockJobRepo.On("GetByID", job.ID).Return(job, nil)
	mockJobRepo.On("Update", mock.AnythingOfType("*models.Job")).Return(nil)

	response, err := service.CancelJob(job.ID, "Customer requested cancellation", adminID, models.UserRoleAdmin)

	require.NoError(t, err)
	assert.NotNil(t, response)
	mockJobRepo.AssertExpectations(t)
}

func TestJobService_CancelJob_TechnicianUnauthorized(t *testing.T) {
	mockJobRepo := new(MockJobRepository)
	mockCustomerRepo := new(MockCustomerRepository)
	service := NewJobService(mockJobRepo, mockCustomerRepo, nil)

	job := createTestJob()
	techID := uuid.New()

	mockJobRepo.On("GetByID", job.ID).Return(job, nil)

	response, err := service.CancelJob(job.ID, "Reason", techID, models.UserRoleTechnician)

	assert.Error(t, err)
	assert.Nil(t, response)
	assert.ErrorIs(t, err, ErrUnauthorized)
	mockJobRepo.AssertExpectations(t)
}

// Tests for ListJobs
func TestJobService_ListJobs_AdminSeesAll(t *testing.T) {
	mockJobRepo := new(MockJobRepository)
	mockCustomerRepo := new(MockCustomerRepository)
	service := NewJobService(mockJobRepo, mockCustomerRepo, nil)

	jobs := []models.Job{*createTestJob(), *createTestJob()}
	limit := 10
	offset := 0
	filter := &models.JobListFilter{
		Limit:  &limit,
		Offset: &offset,
	}

	mockJobRepo.On("List", filter).Return(jobs, int64(2), nil)

	responses, total, err := service.ListJobs(filter, uuid.New(), models.UserRoleAdmin)

	require.NoError(t, err)
	assert.Len(t, responses, 2)
	assert.Equal(t, int64(2), total)
	mockJobRepo.AssertExpectations(t)
}

func TestJobService_ListJobs_TechnicianFiltered(t *testing.T) {
	mockJobRepo := new(MockJobRepository)
	mockCustomerRepo := new(MockCustomerRepository)
	service := NewJobService(mockJobRepo, mockCustomerRepo, nil)

	techID := uuid.New()
	jobs := []models.Job{*createTestJob()}
	limit := 10
	offset := 0
	filter := &models.JobListFilter{
		Limit:  &limit,
		Offset: &offset,
	}

	// Service should modify filter to include AssignedUserID for technicians
	mockJobRepo.On("List", mock.MatchedBy(func(f *models.JobListFilter) bool {
		return f.AssignedUserID != nil && *f.AssignedUserID == techID
	})).Return(jobs, int64(1), nil)

	responses, total, err := service.ListJobs(filter, techID, models.UserRoleTechnician)

	require.NoError(t, err)
	assert.Len(t, responses, 1)
	assert.Equal(t, int64(1), total)
	mockJobRepo.AssertExpectations(t)
}

// Tests for AssignTechnician
func TestJobService_AssignTechnician_Success(t *testing.T) {
	mockJobRepo := new(MockJobRepository)
	mockCustomerRepo := new(MockCustomerRepository)
	service := NewJobService(mockJobRepo, mockCustomerRepo, nil)

	job := createTestJob()
	techID := uuid.New()
	adminID := uuid.New()

	mockJobRepo.On("GetByID", job.ID).Return(job, nil)
	mockJobRepo.On("AddAssignment", mock.AnythingOfType("*models.JobAssignment")).Return(nil)

	err := service.AssignTechnician(job.ID, techID, "Lead Technician", adminID, models.UserRoleAdmin)

	require.NoError(t, err)
	mockJobRepo.AssertExpectations(t)
}

func TestJobService_AssignTechnician_TechnicianUnauthorized(t *testing.T) {
	mockJobRepo := new(MockJobRepository)
	mockCustomerRepo := new(MockCustomerRepository)
	service := NewJobService(mockJobRepo, mockCustomerRepo, nil)

	job := createTestJob()
	techID := uuid.New()

	mockJobRepo.On("GetByID", job.ID).Return(job, nil)

	err := service.AssignTechnician(job.ID, techID, "Role", techID, models.UserRoleTechnician)

	assert.Error(t, err)
	assert.ErrorIs(t, err, ErrUnauthorized)
	mockJobRepo.AssertExpectations(t)
}
