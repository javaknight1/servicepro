package handlers

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/javaknight1/servicepro/backend/internal/models"
	"github.com/javaknight1/servicepro/backend/internal/repository"
)

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

func setupRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	return gin.New()
}

func TestCreateCustomer_Success(t *testing.T) {
	mockRepo := new(MockCustomerRepository)
	handler := NewCustomerHandler(mockRepo)
	router := setupRouter()

	router.POST("/customers", handler.CreateCustomer)

	reqBody := models.CreateCustomerRequest{
		FirstName:            "John",
		LastName:             "Doe",
		Email:                "john.doe@example.com",
		PhonePrimary:         "555-123-4567",
		BillingAddressStreet: "123 Main St",
		BillingAddressCity:   "Springfield",
		BillingAddressState:  "IL",
		BillingAddressZip:    "62701",
		CustomerType:         models.CustomerTypeResidential,
		Status:               models.CustomerStatusActive,
	}

	// Mock expectations
	mockRepo.On("EmailExists", reqBody.Email).Return(false, nil)
	mockRepo.On("Create", mock.AnythingOfType("*models.Customer")).Return(nil)

	body, _ := json.Marshal(reqBody)
	req, _ := http.NewRequest("POST", "/customers", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)
	mockRepo.AssertExpectations(t)
}

func TestCreateCustomer_EmailExists(t *testing.T) {
	mockRepo := new(MockCustomerRepository)
	handler := NewCustomerHandler(mockRepo)
	router := setupRouter()

	router.POST("/customers", handler.CreateCustomer)

	reqBody := models.CreateCustomerRequest{
		FirstName:            "John",
		LastName:             "Doe",
		Email:                "existing@example.com",
		PhonePrimary:         "555-123-4567",
		BillingAddressStreet: "123 Main St",
		BillingAddressCity:   "Springfield",
		BillingAddressState:  "IL",
		BillingAddressZip:    "62701",
		CustomerType:         models.CustomerTypeResidential,
	}

	// Mock expectations - email already exists
	mockRepo.On("EmailExists", reqBody.Email).Return(true, nil)

	body, _ := json.Marshal(reqBody)
	req, _ := http.NewRequest("POST", "/customers", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusConflict, w.Code)

	var response models.ErrorResponse
	json.Unmarshal(w.Body.Bytes(), &response)
	assert.Equal(t, "email_exists", response.Error)
	mockRepo.AssertExpectations(t)
}

func TestCreateCustomer_InvalidRequest(t *testing.T) {
	mockRepo := new(MockCustomerRepository)
	handler := NewCustomerHandler(mockRepo)
	router := setupRouter()

	router.POST("/customers", handler.CreateCustomer)

	// Invalid JSON
	req, _ := http.NewRequest("POST", "/customers", bytes.NewBuffer([]byte("invalid json")))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestGetCustomer_Success(t *testing.T) {
	mockRepo := new(MockCustomerRepository)
	handler := NewCustomerHandler(mockRepo)
	router := setupRouter()

	router.GET("/customers/:id", handler.GetCustomer)

	customerID := uuid.New()
	customer := &models.Customer{
		ID:                   customerID,
		FirstName:            "John",
		LastName:             "Doe",
		Email:                "john@example.com",
		PhonePrimary:         "555-123-4567",
		BillingAddressStreet: "123 Main St",
		BillingAddressCity:   "Springfield",
		BillingAddressState:  "IL",
		BillingAddressZip:    "62701",
		CustomerType:         models.CustomerTypeResidential,
		Status:               models.CustomerStatusActive,
	}

	mockRepo.On("GetByID", customerID).Return(customer, nil)

	req, _ := http.NewRequest("GET", "/customers/"+customerID.String(), nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response models.CustomerResponse
	json.Unmarshal(w.Body.Bytes(), &response)
	assert.Equal(t, customerID, response.ID)
	assert.Equal(t, "John Doe", response.DisplayName)
	mockRepo.AssertExpectations(t)
}

func TestGetCustomer_NotFound(t *testing.T) {
	mockRepo := new(MockCustomerRepository)
	handler := NewCustomerHandler(mockRepo)
	router := setupRouter()

	router.GET("/customers/:id", handler.GetCustomer)

	customerID := uuid.New()
	mockRepo.On("GetByID", customerID).Return(nil, repository.ErrCustomerNotFound)

	req, _ := http.NewRequest("GET", "/customers/"+customerID.String(), nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)

	var response models.ErrorResponse
	json.Unmarshal(w.Body.Bytes(), &response)
	assert.Equal(t, "customer_not_found", response.Error)
	mockRepo.AssertExpectations(t)
}

func TestGetCustomer_InvalidID(t *testing.T) {
	mockRepo := new(MockCustomerRepository)
	handler := NewCustomerHandler(mockRepo)
	router := setupRouter()

	router.GET("/customers/:id", handler.GetCustomer)

	req, _ := http.NewRequest("GET", "/customers/invalid-uuid", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var response models.ErrorResponse
	json.Unmarshal(w.Body.Bytes(), &response)
	assert.Equal(t, "invalid_id", response.Error)
}

func TestUpdateCustomer_Success(t *testing.T) {
	mockRepo := new(MockCustomerRepository)
	handler := NewCustomerHandler(mockRepo)
	router := setupRouter()

	router.PUT("/customers/:id", handler.UpdateCustomer)

	customerID := uuid.New()
	existingCustomer := &models.Customer{
		ID:                   customerID,
		FirstName:            "John",
		LastName:             "Doe",
		Email:                "john@example.com",
		PhonePrimary:         "555-123-4567",
		BillingAddressStreet: "123 Main St",
		BillingAddressCity:   "Springfield",
		BillingAddressState:  "IL",
		BillingAddressZip:    "62701",
		CustomerType:         models.CustomerTypeResidential,
		Status:               models.CustomerStatusActive,
	}

	newFirstName := "Jane"
	reqBody := models.UpdateCustomerRequest{
		FirstName: &newFirstName,
	}

	mockRepo.On("GetByID", customerID).Return(existingCustomer, nil)
	mockRepo.On("Update", mock.AnythingOfType("*models.Customer")).Return(nil)

	body, _ := json.Marshal(reqBody)
	req, _ := http.NewRequest("PUT", "/customers/"+customerID.String(), bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response models.CustomerResponse
	json.Unmarshal(w.Body.Bytes(), &response)
	assert.Equal(t, "Jane", response.FirstName)
	mockRepo.AssertExpectations(t)
}

func TestUpdateCustomer_NotFound(t *testing.T) {
	mockRepo := new(MockCustomerRepository)
	handler := NewCustomerHandler(mockRepo)
	router := setupRouter()

	router.PUT("/customers/:id", handler.UpdateCustomer)

	customerID := uuid.New()
	newFirstName := "Jane"
	reqBody := models.UpdateCustomerRequest{
		FirstName: &newFirstName,
	}

	mockRepo.On("GetByID", customerID).Return(nil, repository.ErrCustomerNotFound)

	body, _ := json.Marshal(reqBody)
	req, _ := http.NewRequest("PUT", "/customers/"+customerID.String(), bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
	mockRepo.AssertExpectations(t)
}

func TestDeleteCustomer_Success(t *testing.T) {
	mockRepo := new(MockCustomerRepository)
	handler := NewCustomerHandler(mockRepo)
	router := setupRouter()

	router.DELETE("/customers/:id", handler.DeleteCustomer)

	customerID := uuid.New()
	mockRepo.On("Delete", customerID).Return(nil)

	req, _ := http.NewRequest("DELETE", "/customers/"+customerID.String(), nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNoContent, w.Code)
	mockRepo.AssertExpectations(t)
}

func TestDeleteCustomer_NotFound(t *testing.T) {
	mockRepo := new(MockCustomerRepository)
	handler := NewCustomerHandler(mockRepo)
	router := setupRouter()

	router.DELETE("/customers/:id", handler.DeleteCustomer)

	customerID := uuid.New()
	mockRepo.On("Delete", customerID).Return(repository.ErrCustomerNotFound)

	req, _ := http.NewRequest("DELETE", "/customers/"+customerID.String(), nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
	mockRepo.AssertExpectations(t)
}

func TestListCustomers_Success(t *testing.T) {
	mockRepo := new(MockCustomerRepository)
	handler := NewCustomerHandler(mockRepo)
	router := setupRouter()

	router.GET("/customers", handler.ListCustomers)

	customers := []models.Customer{
		{
			ID:                   uuid.New(),
			FirstName:            "John",
			LastName:             "Doe",
			Email:                "john@example.com",
			PhonePrimary:         "555-123-4567",
			BillingAddressStreet: "123 Main St",
			BillingAddressCity:   "Springfield",
			BillingAddressState:  "IL",
			BillingAddressZip:    "62701",
			CustomerType:         models.CustomerTypeResidential,
			Status:               models.CustomerStatusActive,
		},
	}

	mockRepo.On("List", mock.AnythingOfType("*models.CustomerListFilter")).Return(customers, int64(1), nil)

	req, _ := http.NewRequest("GET", "/customers?page=1&page_size=20", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response models.CustomerListResponse
	json.Unmarshal(w.Body.Bytes(), &response)
	assert.Equal(t, 1, len(response.Customers))
	assert.Equal(t, int64(1), response.Total)
	mockRepo.AssertExpectations(t)
}

func TestListCustomers_WithFilters(t *testing.T) {
	mockRepo := new(MockCustomerRepository)
	handler := NewCustomerHandler(mockRepo)
	router := setupRouter()

	router.GET("/customers", handler.ListCustomers)

	customers := []models.Customer{}
	mockRepo.On("List", mock.MatchedBy(func(filter *models.CustomerListFilter) bool {
		return filter.Search == "john" &&
			filter.Status == models.CustomerStatusActive &&
			filter.CustomerType == models.CustomerTypeResidential
	})).Return(customers, int64(0), nil)

	req, _ := http.NewRequest("GET", "/customers?search=john&status=active&customer_type=residential", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	mockRepo.AssertExpectations(t)
}

func TestSearchCustomers_Success(t *testing.T) {
	mockRepo := new(MockCustomerRepository)
	handler := NewCustomerHandler(mockRepo)
	router := setupRouter()

	router.GET("/customers/search", handler.SearchCustomers)

	customers := []models.Customer{
		{
			ID:        uuid.New(),
			FirstName: "John",
			LastName:  "Doe",
			Email:     "john@example.com",
		},
	}

	mockRepo.On("Search", "john").Return(customers, nil)

	req, _ := http.NewRequest("GET", "/customers/search?q=john", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response []models.CustomerResponse
	json.Unmarshal(w.Body.Bytes(), &response)
	assert.Equal(t, 1, len(response))
	mockRepo.AssertExpectations(t)
}

func TestSearchCustomers_MissingQuery(t *testing.T) {
	mockRepo := new(MockCustomerRepository)
	handler := NewCustomerHandler(mockRepo)
	router := setupRouter()

	router.GET("/customers/search", handler.SearchCustomers)

	req, _ := http.NewRequest("GET", "/customers/search", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var response models.ErrorResponse
	json.Unmarshal(w.Body.Bytes(), &response)
	assert.Equal(t, "invalid_request", response.Error)
}

func TestGetCustomerByEmail_Success(t *testing.T) {
	mockRepo := new(MockCustomerRepository)
	handler := NewCustomerHandler(mockRepo)
	router := setupRouter()

	router.GET("/customers/email/:email", handler.GetCustomerByEmail)

	customer := &models.Customer{
		ID:        uuid.New(),
		FirstName: "John",
		LastName:  "Doe",
		Email:     "john@example.com",
	}

	mockRepo.On("GetByEmail", "john@example.com").Return(customer, nil)

	req, _ := http.NewRequest("GET", "/customers/email/john@example.com", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	mockRepo.AssertExpectations(t)
}

func TestGetCustomerByEmail_NotFound(t *testing.T) {
	mockRepo := new(MockCustomerRepository)
	handler := NewCustomerHandler(mockRepo)
	router := setupRouter()

	router.GET("/customers/email/:email", handler.GetCustomerByEmail)

	mockRepo.On("GetByEmail", "notfound@example.com").Return(nil, repository.ErrCustomerNotFound)

	req, _ := http.NewRequest("GET", "/customers/email/notfound@example.com", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
	mockRepo.AssertExpectations(t)
}

func TestCreateCustomer_RepositoryError(t *testing.T) {
	mockRepo := new(MockCustomerRepository)
	handler := NewCustomerHandler(mockRepo)
	router := setupRouter()

	router.POST("/customers", handler.CreateCustomer)

	reqBody := models.CreateCustomerRequest{
		FirstName:            "John",
		LastName:             "Doe",
		Email:                "john.doe@example.com",
		PhonePrimary:         "555-123-4567",
		BillingAddressStreet: "123 Main St",
		BillingAddressCity:   "Springfield",
		BillingAddressState:  "IL",
		BillingAddressZip:    "62701",
		CustomerType:         models.CustomerTypeResidential,
	}

	// Mock expectations - internal error
	mockRepo.On("EmailExists", reqBody.Email).Return(false, nil)
	mockRepo.On("Create", mock.AnythingOfType("*models.Customer")).Return(errors.New("database error"))

	body, _ := json.Marshal(reqBody)
	req, _ := http.NewRequest("POST", "/customers", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	mockRepo.AssertExpectations(t)
}

func TestAdvancedSearch_Success(t *testing.T) {
	mockRepo := new(MockCustomerRepository)
	handler := NewCustomerHandler(mockRepo)
	router := setupRouter()

	router.GET("/customers/advanced-search", handler.AdvancedSearch)

	customers := []models.Customer{
		{
			ID:        uuid.New(),
			FirstName: "John",
			LastName:  "Doe",
			Email:     "john@example.com",
		},
	}

	mockRepo.On("FullTextSearch", "john", models.CustomerType(""), models.CustomerStatus(""), "", "", 20, 0, "rank", "DESC").
		Return(customers, int64(1), nil)

	req, _ := http.NewRequest("GET", "/customers/advanced-search?q=john", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	mockRepo.AssertExpectations(t)
}

func TestAdvancedSearch_WithFilters(t *testing.T) {
	mockRepo := new(MockCustomerRepository)
	handler := NewCustomerHandler(mockRepo)
	router := setupRouter()

	router.GET("/customers/advanced-search", handler.AdvancedSearch)

	customers := []models.Customer{
		{
			ID:           uuid.New(),
			FirstName:    "John",
			LastName:     "Doe",
			Email:        "john@example.com",
			CustomerType: models.CustomerTypeCommercial,
			Status:       models.CustomerStatusActive,
		},
	}

	mockRepo.On("FullTextSearch", "john", models.CustomerTypeCommercial, models.CustomerStatusActive, "Austin", "TX", 20, 0, "rank", "DESC").
		Return(customers, int64(1), nil)

	req, _ := http.NewRequest("GET", "/customers/advanced-search?q=john&type=commercial&status=active&city=Austin&state=TX", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	mockRepo.AssertExpectations(t)
}

func TestAdvancedSearch_WithPagination(t *testing.T) {
	mockRepo := new(MockCustomerRepository)
	handler := NewCustomerHandler(mockRepo)
	router := setupRouter()

	router.GET("/customers/advanced-search", handler.AdvancedSearch)

	customers := []models.Customer{
		{
			ID:        uuid.New(),
			FirstName: "John",
			LastName:  "Doe",
			Email:     "john@example.com",
		},
	}

	mockRepo.On("FullTextSearch", "john", models.CustomerType(""), models.CustomerStatus(""), "", "", 50, 50, "last_name", "ASC").
		Return(customers, int64(100), nil)

	req, _ := http.NewRequest("GET", "/customers/advanced-search?q=john&page=2&limit=50&sort=last_name&order=ASC", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	mockRepo.AssertExpectations(t)
}

func TestAdvancedSearch_MissingQuery(t *testing.T) {
	mockRepo := new(MockCustomerRepository)
	handler := NewCustomerHandler(mockRepo)
	router := setupRouter()

	router.GET("/customers/advanced-search", handler.AdvancedSearch)

	req, _ := http.NewRequest("GET", "/customers/advanced-search", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestAdvancedSearch_RepositoryError(t *testing.T) {
	mockRepo := new(MockCustomerRepository)
	handler := NewCustomerHandler(mockRepo)
	router := setupRouter()

	router.GET("/customers/advanced-search", handler.AdvancedSearch)

	mockRepo.On("FullTextSearch", "john", models.CustomerType(""), models.CustomerStatus(""), "", "", 20, 0, "rank", "DESC").
		Return([]models.Customer{}, int64(0), errors.New("database error"))

	req, _ := http.NewRequest("GET", "/customers/advanced-search?q=john", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	mockRepo.AssertExpectations(t)
}

func TestAutocomplete_Success(t *testing.T) {
	mockRepo := new(MockCustomerRepository)
	handler := NewCustomerHandler(mockRepo)
	router := setupRouter()

	router.GET("/customers/autocomplete", handler.Autocomplete)

	results := []map[string]interface{}{
		{
			"id":           uuid.New().String(),
			"display_name": "John Doe",
			"email":        "john@example.com",
		},
	}

	mockRepo.On("Autocomplete", "john", 10).Return(results, nil)

	req, _ := http.NewRequest("GET", "/customers/autocomplete?q=john", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	mockRepo.AssertExpectations(t)
}

func TestAutocomplete_WithLimit(t *testing.T) {
	mockRepo := new(MockCustomerRepository)
	handler := NewCustomerHandler(mockRepo)
	router := setupRouter()

	router.GET("/customers/autocomplete", handler.Autocomplete)

	results := []map[string]interface{}{}

	mockRepo.On("Autocomplete", "john", 25).Return(results, nil)

	req, _ := http.NewRequest("GET", "/customers/autocomplete?q=john&limit=25", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	mockRepo.AssertExpectations(t)
}

func TestAutocomplete_MissingQuery(t *testing.T) {
	mockRepo := new(MockCustomerRepository)
	handler := NewCustomerHandler(mockRepo)
	router := setupRouter()

	router.GET("/customers/autocomplete", handler.Autocomplete)

	req, _ := http.NewRequest("GET", "/customers/autocomplete", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestAutocomplete_RepositoryError(t *testing.T) {
	mockRepo := new(MockCustomerRepository)
	handler := NewCustomerHandler(mockRepo)
	router := setupRouter()

	router.GET("/customers/autocomplete", handler.Autocomplete)

	mockRepo.On("Autocomplete", "john", 10).Return([]map[string]interface{}{}, errors.New("database error"))

	req, _ := http.NewRequest("GET", "/customers/autocomplete?q=john", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	mockRepo.AssertExpectations(t)
}

func TestGetSearchStats_Success(t *testing.T) {
	mockRepo := new(MockCustomerRepository)
	handler := NewCustomerHandler(mockRepo)
	router := setupRouter()

	router.GET("/customers/stats", handler.GetSearchStats)

	stats := map[string]interface{}{
		"total_customers":      100,
		"active_customers":     80,
		"commercial_customers": 30,
	}

	mockRepo.On("GetSearchStats").Return(stats, nil)

	req, _ := http.NewRequest("GET", "/customers/stats", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	mockRepo.AssertExpectations(t)
}

func TestGetSearchStats_RepositoryError(t *testing.T) {
	mockRepo := new(MockCustomerRepository)
	handler := NewCustomerHandler(mockRepo)
	router := setupRouter()

	router.GET("/customers/stats", handler.GetSearchStats)

	mockRepo.On("GetSearchStats").Return(map[string]interface{}{}, errors.New("database error"))

	req, _ := http.NewRequest("GET", "/customers/stats", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	mockRepo.AssertExpectations(t)
}
