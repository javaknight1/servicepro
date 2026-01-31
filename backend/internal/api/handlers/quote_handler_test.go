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
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/javaknight1/servicepro/backend/internal/models"
	"github.com/javaknight1/servicepro/backend/internal/services"
)

// MockQuoteService is a mock implementation of QuoteService
type MockQuoteService struct {
	mock.Mock
}

func (m *MockQuoteService) CreateQuote(req *models.QuoteRequest, userID uuid.UUID) (*models.QuoteResponse, error) {
	args := m.Called(req, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.QuoteResponse), args.Error(1)
}

func (m *MockQuoteService) GetQuote(id uuid.UUID) (*models.QuoteResponse, error) {
	args := m.Called(id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.QuoteResponse), args.Error(1)
}

func (m *MockQuoteService) GetQuoteByNumber(quoteNumber string) (*models.QuoteResponse, error) {
	args := m.Called(quoteNumber)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.QuoteResponse), args.Error(1)
}

func (m *MockQuoteService) ListQuotes(params *models.QuoteQueryParams) (*models.QuoteListResponse, error) {
	args := m.Called(params)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.QuoteListResponse), args.Error(1)
}

func (m *MockQuoteService) UpdateQuote(id uuid.UUID, req *models.QuoteRequest, userID uuid.UUID) (*models.QuoteResponse, error) {
	args := m.Called(id, req, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.QuoteResponse), args.Error(1)
}

func (m *MockQuoteService) DeleteQuote(id uuid.UUID) error {
	args := m.Called(id)
	return args.Error(0)
}

func (m *MockQuoteService) SendQuote(id uuid.UUID) error {
	args := m.Called(id)
	return args.Error(0)
}

func (m *MockQuoteService) AcceptQuote(id uuid.UUID) error {
	args := m.Called(id)
	return args.Error(0)
}

func (m *MockQuoteService) RejectQuote(id uuid.UUID) error {
	args := m.Called(id)
	return args.Error(0)
}

func (m *MockQuoteService) GetQuotesByCustomer(customerID uuid.UUID) ([]models.QuoteResponse, error) {
	args := m.Called(customerID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]models.QuoteResponse), args.Error(1)
}

func (m *MockQuoteService) GetQuoteStats() (*services.QuoteStats, error) {
	args := m.Called()
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*services.QuoteStats), args.Error(1)
}

func (m *MockQuoteService) GetQuotePDFURL(id uuid.UUID) (string, time.Time, error) {
	args := m.Called(id)
	return args.String(0), args.Get(1).(time.Time), args.Error(2)
}

func (m *MockQuoteService) DownloadQuotePDF(id uuid.UUID) ([]byte, string, error) {
	args := m.Called(id)
	if args.Get(0) == nil {
		return nil, "", args.Error(2)
	}
	return args.Get(0).([]byte), args.String(1), args.Error(2)
}

func setupQuoteTestRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	return gin.New()
}

func TestCreateQuote_Success(t *testing.T) {
	mockQuoteService := new(MockQuoteService)
	handler := NewQuoteHandler(mockQuoteService)

	userID := uuid.New()
	customerID := uuid.New()
	quoteID := uuid.New()
	validUntil := time.Now().Add(30 * 24 * time.Hour)

	quoteRequest := models.QuoteRequest{
		CustomerID: customerID,
		ValidUntil: validUntil,
		TaxRate:    decimal.NewFromFloat(0.0825),
		Items: []models.QuoteItemRequest{
			{
				Description: "Test Service",
				Quantity:    decimal.NewFromInt(1),
				UnitPrice:   decimal.NewFromFloat(100.00),
				SortOrder:   1,
			},
		},
	}

	expectedResponse := &models.QuoteResponse{
		ID:          quoteID,
		CustomerID:  customerID,
		QuoteNumber: "QUO-2024-0001",
		Status:      models.QuoteStatusDraft,
		ValidUntil:  validUntil,
		Subtotal:    decimal.NewFromFloat(100.00),
		TaxRate:     decimal.NewFromFloat(0.0825),
		TaxAmount:   decimal.NewFromFloat(8.25),
		Total:       decimal.NewFromFloat(108.25),
		Items: []models.QuoteItemResponse{
			{
				ID:          uuid.New(),
				QuoteID:     quoteID,
				Description: "Test Service",
				Quantity:    decimal.NewFromInt(1),
				UnitPrice:   decimal.NewFromFloat(100.00),
				Total:       decimal.NewFromFloat(100.00),
				SortOrder:   1,
			},
		},
		CreatedBy: userID,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	mockQuoteService.On("CreateQuote", mock.AnythingOfType("*models.QuoteRequest"), userID).Return(expectedResponse, nil)

	router := setupQuoteTestRouter()
	router.POST("/quotes", func(c *gin.Context) {
		c.Set("user_id", userID)
		handler.CreateQuote(c)
	})

	body, _ := json.Marshal(quoteRequest)
	req, _ := http.NewRequest("POST", "/quotes", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)

	var response models.QuoteResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, expectedResponse.QuoteNumber, response.QuoteNumber)
	assert.Equal(t, expectedResponse.Status, response.Status)

	mockQuoteService.AssertExpectations(t)
}

func TestCreateQuote_InvalidRequest(t *testing.T) {
	mockQuoteService := new(MockQuoteService)
	handler := NewQuoteHandler(mockQuoteService)

	router := setupQuoteTestRouter()
	router.POST("/quotes", func(c *gin.Context) {
		c.Set("user_id", uuid.New())
		handler.CreateQuote(c)
	})

	// Invalid JSON
	req, _ := http.NewRequest("POST", "/quotes", bytes.NewBuffer([]byte("invalid json")))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestCreateQuote_Unauthorized(t *testing.T) {
	mockQuoteService := new(MockQuoteService)
	handler := NewQuoteHandler(mockQuoteService)

	router := setupQuoteTestRouter()
	router.POST("/quotes", handler.CreateQuote)

	quoteRequest := models.QuoteRequest{
		CustomerID: uuid.New(),
		ValidUntil: time.Now().Add(30 * 24 * time.Hour),
		TaxRate:    decimal.NewFromFloat(0.0825),
		Items: []models.QuoteItemRequest{
			{
				Description: "Test Service",
				Quantity:    decimal.NewFromInt(1),
				UnitPrice:   decimal.NewFromFloat(100.00),
			},
		},
	}

	body, _ := json.Marshal(quoteRequest)
	req, _ := http.NewRequest("POST", "/quotes", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestGetQuote_Success(t *testing.T) {
	mockQuoteService := new(MockQuoteService)
	handler := NewQuoteHandler(mockQuoteService)

	quoteID := uuid.New()
	expectedResponse := &models.QuoteResponse{
		ID:          quoteID,
		QuoteNumber: "QUO-2024-0001",
		Status:      models.QuoteStatusDraft,
		Subtotal:    decimal.NewFromFloat(100.00),
		TaxRate:     decimal.NewFromFloat(0.0825),
		TaxAmount:   decimal.NewFromFloat(8.25),
		Total:       decimal.NewFromFloat(108.25),
	}

	mockQuoteService.On("GetQuote", quoteID).Return(expectedResponse, nil)

	router := setupQuoteTestRouter()
	router.GET("/quotes/:id", handler.GetQuote)

	req, _ := http.NewRequest("GET", "/quotes/"+quoteID.String(), nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response models.QuoteResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, expectedResponse.QuoteNumber, response.QuoteNumber)

	mockQuoteService.AssertExpectations(t)
}

func TestGetQuote_NotFound(t *testing.T) {
	mockQuoteService := new(MockQuoteService)
	handler := NewQuoteHandler(mockQuoteService)

	quoteID := uuid.New()
	mockQuoteService.On("GetQuote", quoteID).Return(nil, services.ErrQuoteNotFound)

	router := setupQuoteTestRouter()
	router.GET("/quotes/:id", handler.GetQuote)

	req, _ := http.NewRequest("GET", "/quotes/"+quoteID.String(), nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)

	mockQuoteService.AssertExpectations(t)
}

func TestGetQuote_InvalidID(t *testing.T) {
	mockQuoteService := new(MockQuoteService)
	handler := NewQuoteHandler(mockQuoteService)

	router := setupQuoteTestRouter()
	router.GET("/quotes/:id", handler.GetQuote)

	req, _ := http.NewRequest("GET", "/quotes/invalid-uuid", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestListQuotes_Success(t *testing.T) {
	mockQuoteService := new(MockQuoteService)
	handler := NewQuoteHandler(mockQuoteService)

	expectedResponse := &models.QuoteListResponse{
		Quotes: []models.QuoteResponse{
			{
				ID:          uuid.New(),
				QuoteNumber: "QUO-2024-0001",
				Status:      models.QuoteStatusDraft,
			},
		},
		Total:    1,
		Page:     1,
		PageSize: 50,
	}

	mockQuoteService.On("ListQuotes", mock.AnythingOfType("*models.QuoteQueryParams")).Return(expectedResponse, nil)

	router := setupQuoteTestRouter()
	router.GET("/quotes", handler.ListQuotes)

	req, _ := http.NewRequest("GET", "/quotes?page=1&page_size=50", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response models.QuoteListResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, expectedResponse.Total, response.Total)

	mockQuoteService.AssertExpectations(t)
}

func TestUpdateQuote_Success(t *testing.T) {
	mockQuoteService := new(MockQuoteService)
	handler := NewQuoteHandler(mockQuoteService)

	userID := uuid.New()
	quoteID := uuid.New()
	customerID := uuid.New()
	validUntil := time.Now().Add(30 * 24 * time.Hour)

	quoteRequest := models.QuoteRequest{
		CustomerID: customerID,
		ValidUntil: validUntil,
		TaxRate:    decimal.NewFromFloat(0.0825),
		Items: []models.QuoteItemRequest{
			{
				Description: "Updated Service",
				Quantity:    decimal.NewFromInt(2),
				UnitPrice:   decimal.NewFromFloat(150.00),
			},
		},
	}

	expectedResponse := &models.QuoteResponse{
		ID:          quoteID,
		QuoteNumber: "QUO-2024-0001",
		Status:      models.QuoteStatusDraft,
		Subtotal:    decimal.NewFromFloat(300.00),
		TaxRate:     decimal.NewFromFloat(0.0825),
		TaxAmount:   decimal.NewFromFloat(24.75),
		Total:       decimal.NewFromFloat(324.75),
	}

	mockQuoteService.On("UpdateQuote", quoteID, mock.AnythingOfType("*models.QuoteRequest"), userID).Return(expectedResponse, nil)

	router := setupQuoteTestRouter()
	router.PUT("/quotes/:id", func(c *gin.Context) {
		c.Set("user_id", userID)
		handler.UpdateQuote(c)
	})

	body, _ := json.Marshal(quoteRequest)
	req, _ := http.NewRequest("PUT", "/quotes/"+quoteID.String(), bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response models.QuoteResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, expectedResponse.Total.String(), response.Total.String())

	mockQuoteService.AssertExpectations(t)
}

func TestUpdateQuote_CannotBeEdited(t *testing.T) {
	mockQuoteService := new(MockQuoteService)
	handler := NewQuoteHandler(mockQuoteService)

	userID := uuid.New()
	quoteID := uuid.New()

	quoteRequest := models.QuoteRequest{
		CustomerID: uuid.New(),
		ValidUntil: time.Now().Add(30 * 24 * time.Hour),
		TaxRate:    decimal.NewFromFloat(0.0825),
		Items: []models.QuoteItemRequest{
			{
				Description: "Test",
				Quantity:    decimal.NewFromInt(1),
				UnitPrice:   decimal.NewFromFloat(100.00),
			},
		},
	}

	mockQuoteService.On("UpdateQuote", quoteID, mock.AnythingOfType("*models.QuoteRequest"), userID).Return(nil, services.ErrQuoteCannotBeEdited)

	router := setupQuoteTestRouter()
	router.PUT("/quotes/:id", func(c *gin.Context) {
		c.Set("user_id", userID)
		handler.UpdateQuote(c)
	})

	body, _ := json.Marshal(quoteRequest)
	req, _ := http.NewRequest("PUT", "/quotes/"+quoteID.String(), bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)

	mockQuoteService.AssertExpectations(t)
}

func TestDeleteQuote_Success(t *testing.T) {
	mockQuoteService := new(MockQuoteService)
	handler := NewQuoteHandler(mockQuoteService)

	quoteID := uuid.New()
	mockQuoteService.On("DeleteQuote", quoteID).Return(nil)

	router := setupQuoteTestRouter()
	router.DELETE("/quotes/:id", handler.DeleteQuote)

	req, _ := http.NewRequest("DELETE", "/quotes/"+quoteID.String(), nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNoContent, w.Code)

	mockQuoteService.AssertExpectations(t)
}

func TestDeleteQuote_CannotBeDeleted(t *testing.T) {
	mockQuoteService := new(MockQuoteService)
	handler := NewQuoteHandler(mockQuoteService)

	quoteID := uuid.New()
	mockQuoteService.On("DeleteQuote", quoteID).Return(services.ErrQuoteCannotBeDeleted)

	router := setupQuoteTestRouter()
	router.DELETE("/quotes/:id", handler.DeleteQuote)

	req, _ := http.NewRequest("DELETE", "/quotes/"+quoteID.String(), nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)

	mockQuoteService.AssertExpectations(t)
}

func TestSendQuote_Success(t *testing.T) {
	mockQuoteService := new(MockQuoteService)
	handler := NewQuoteHandler(mockQuoteService)

	quoteID := uuid.New()
	expectedResponse := &models.QuoteResponse{
		ID:          quoteID,
		QuoteNumber: "QUO-2024-0001",
		Status:      models.QuoteStatusSent,
	}

	mockQuoteService.On("SendQuote", quoteID).Return(nil)
	mockQuoteService.On("GetQuote", quoteID).Return(expectedResponse, nil)

	router := setupQuoteTestRouter()
	router.POST("/quotes/:id/send", handler.SendQuote)

	req, _ := http.NewRequest("POST", "/quotes/"+quoteID.String()+"/send", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response models.QuoteResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, models.QuoteStatusSent, response.Status)

	mockQuoteService.AssertExpectations(t)
}

func TestAcceptQuote_Success(t *testing.T) {
	mockQuoteService := new(MockQuoteService)
	handler := NewQuoteHandler(mockQuoteService)

	quoteID := uuid.New()
	expectedResponse := &models.QuoteResponse{
		ID:          quoteID,
		QuoteNumber: "QUO-2024-0001",
		Status:      models.QuoteStatusAccepted,
	}

	mockQuoteService.On("AcceptQuote", quoteID).Return(nil)
	mockQuoteService.On("GetQuote", quoteID).Return(expectedResponse, nil)

	router := setupQuoteTestRouter()
	router.POST("/quotes/:id/accept", handler.AcceptQuote)

	req, _ := http.NewRequest("POST", "/quotes/"+quoteID.String()+"/accept", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response models.QuoteResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, models.QuoteStatusAccepted, response.Status)

	mockQuoteService.AssertExpectations(t)
}

func TestRejectQuote_Success(t *testing.T) {
	mockQuoteService := new(MockQuoteService)
	handler := NewQuoteHandler(mockQuoteService)

	quoteID := uuid.New()
	expectedResponse := &models.QuoteResponse{
		ID:          quoteID,
		QuoteNumber: "QUO-2024-0001",
		Status:      models.QuoteStatusDeclined,
	}

	mockQuoteService.On("RejectQuote", quoteID).Return(nil)
	mockQuoteService.On("GetQuote", quoteID).Return(expectedResponse, nil)

	router := setupQuoteTestRouter()
	router.POST("/quotes/:id/reject", handler.RejectQuote)

	req, _ := http.NewRequest("POST", "/quotes/"+quoteID.String()+"/reject", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response models.QuoteResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, models.QuoteStatusDeclined, response.Status)

	mockQuoteService.AssertExpectations(t)
}

func TestGetCustomerQuotes_Success(t *testing.T) {
	mockQuoteService := new(MockQuoteService)
	handler := NewQuoteHandler(mockQuoteService)

	customerID := uuid.New()
	expectedQuotes := []models.QuoteResponse{
		{
			ID:          uuid.New(),
			CustomerID:  customerID,
			QuoteNumber: "QUO-2024-0001",
			Status:      models.QuoteStatusDraft,
		},
		{
			ID:          uuid.New(),
			CustomerID:  customerID,
			QuoteNumber: "QUO-2024-0002",
			Status:      models.QuoteStatusSent,
		},
	}

	mockQuoteService.On("GetQuotesByCustomer", customerID).Return(expectedQuotes, nil)

	router := setupQuoteTestRouter()
	router.GET("/customers/:id/quotes", handler.GetCustomerQuotes)

	req, _ := http.NewRequest("GET", "/customers/"+customerID.String()+"/quotes", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response []models.QuoteResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Len(t, response, 2)

	mockQuoteService.AssertExpectations(t)
}

func TestGetQuoteStats_Success(t *testing.T) {
	mockQuoteService := new(MockQuoteService)
	handler := NewQuoteHandler(mockQuoteService)

	expectedStats := &services.QuoteStats{
		TotalQuotes:    100,
		DraftQuotes:    30,
		SentQuotes:     40,
		AcceptedQuotes: 20,
		DeclinedQuotes: 5,
		ExpiredQuotes:  5,
	}

	mockQuoteService.On("GetQuoteStats").Return(expectedStats, nil)

	router := setupQuoteTestRouter()
	router.GET("/quotes/stats", handler.GetQuoteStats)

	req, _ := http.NewRequest("GET", "/quotes/stats", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response services.QuoteStats
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, expectedStats.TotalQuotes, response.TotalQuotes)
	assert.Equal(t, expectedStats.AcceptedQuotes, response.AcceptedQuotes)

	mockQuoteService.AssertExpectations(t)
}

func TestGetQuoteStats_Error(t *testing.T) {
	mockQuoteService := new(MockQuoteService)
	handler := NewQuoteHandler(mockQuoteService)

	mockQuoteService.On("GetQuoteStats").Return(nil, errors.New("database error"))

	router := setupQuoteTestRouter()
	router.GET("/quotes/stats", handler.GetQuoteStats)

	req, _ := http.NewRequest("GET", "/quotes/stats", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)

	mockQuoteService.AssertExpectations(t)
}
