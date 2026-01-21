package handlers

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/javaknight1/servicepro/backend/internal/models"
	"github.com/javaknight1/servicepro/backend/internal/repository"
)

// CustomerHandler handles customer endpoints
type CustomerHandler struct {
	customerRepo repository.CustomerRepositoryInterface
}

// NewCustomerHandler creates a new customer handler
func NewCustomerHandler(customerRepo repository.CustomerRepositoryInterface) *CustomerHandler {
	return &CustomerHandler{
		customerRepo: customerRepo,
	}
}

// CreateCustomer handles POST /api/v1/customers
// @Summary Create a new customer
// @Description Create a new customer with the provided information
// @Tags customers
// @Accept json
// @Produce json
// @Param request body models.CreateCustomerRequest true "Customer information"
// @Success 201 {object} models.CustomerResponse
// @Failure 400 {object} models.ErrorResponse "Invalid request"
// @Failure 409 {object} models.ErrorResponse "Email already exists"
// @Failure 500 {object} models.ErrorResponse "Internal server error"
// @Router /api/v1/customers [post]
func (h *CustomerHandler) CreateCustomer(c *gin.Context) {
	var req models.CreateCustomerRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "invalid_request",
			Message: err.Error(),
		})
		return
	}

	// Check if email already exists
	exists, err := h.customerRepo.EmailExists(req.Email)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to check email existence",
		})
		return
	}
	if exists {
		c.JSON(http.StatusConflict, models.ErrorResponse{
			Error:   "email_exists",
			Message: "A customer with this email already exists",
		})
		return
	}

	// Create customer model
	customer := &models.Customer{
		FirstName:            req.FirstName,
		LastName:             req.LastName,
		CompanyName:          req.CompanyName,
		Email:                req.Email,
		PhonePrimary:         req.PhonePrimary,
		PhoneSecondary:       req.PhoneSecondary,
		BillingAddressStreet: req.BillingAddressStreet,
		BillingAddressCity:   req.BillingAddressCity,
		BillingAddressState:  req.BillingAddressState,
		BillingAddressZip:    req.BillingAddressZip,
		ServiceAddressStreet: req.ServiceAddressStreet,
		ServiceAddressCity:   req.ServiceAddressCity,
		ServiceAddressState:  req.ServiceAddressState,
		ServiceAddressZip:    req.ServiceAddressZip,
		CustomerType:         req.CustomerType,
		Status:               req.Status,
		Notes:                req.Notes,
	}

	// Set default status if not provided
	if customer.Status == "" {
		customer.Status = models.CustomerStatusProspect
	}

	// Create customer
	if err := h.customerRepo.Create(customer); err != nil {
		if errors.Is(err, repository.ErrEmailAlreadyExists) {
			c.JSON(http.StatusConflict, models.ErrorResponse{
				Error:   "email_exists",
				Message: "A customer with this email already exists",
			})
			return
		}
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to create customer",
		})
		return
	}

	c.JSON(http.StatusCreated, customer.ToResponse())
}

// GetCustomer handles GET /api/v1/customers/:id
// @Summary Get a customer by ID
// @Description Retrieve a customer by their ID
// @Tags customers
// @Produce json
// @Param id path string true "Customer ID"
// @Success 200 {object} models.CustomerResponse
// @Failure 400 {object} models.ErrorResponse "Invalid ID"
// @Failure 404 {object} models.ErrorResponse "Customer not found"
// @Failure 500 {object} models.ErrorResponse "Internal server error"
// @Router /api/v1/customers/{id} [get]
func (h *CustomerHandler) GetCustomer(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "invalid_id",
			Message: "Invalid customer ID format",
		})
		return
	}

	customer, err := h.customerRepo.GetByID(id)
	if err != nil {
		if errors.Is(err, repository.ErrCustomerNotFound) {
			c.JSON(http.StatusNotFound, models.ErrorResponse{
				Error:   "customer_not_found",
				Message: "Customer not found",
			})
			return
		}
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to retrieve customer",
		})
		return
	}

	c.JSON(http.StatusOK, customer.ToResponse())
}

// UpdateCustomer handles PUT /api/v1/customers/:id
// @Summary Update a customer
// @Description Update an existing customer's information
// @Tags customers
// @Accept json
// @Produce json
// @Param id path string true "Customer ID"
// @Param request body models.UpdateCustomerRequest true "Updated customer information"
// @Success 200 {object} models.CustomerResponse
// @Failure 400 {object} models.ErrorResponse "Invalid request"
// @Failure 404 {object} models.ErrorResponse "Customer not found"
// @Failure 409 {object} models.ErrorResponse "Email already exists"
// @Failure 500 {object} models.ErrorResponse "Internal server error"
// @Router /api/v1/customers/{id} [put]
func (h *CustomerHandler) UpdateCustomer(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "invalid_id",
			Message: "Invalid customer ID format",
		})
		return
	}

	var req models.UpdateCustomerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "invalid_request",
			Message: err.Error(),
		})
		return
	}

	// Get existing customer
	customer, err := h.customerRepo.GetByID(id)
	if err != nil {
		if errors.Is(err, repository.ErrCustomerNotFound) {
			c.JSON(http.StatusNotFound, models.ErrorResponse{
				Error:   "customer_not_found",
				Message: "Customer not found",
			})
			return
		}
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to retrieve customer",
		})
		return
	}

	// Update fields if provided
	if req.FirstName != nil {
		customer.FirstName = *req.FirstName
	}
	if req.LastName != nil {
		customer.LastName = *req.LastName
	}
	if req.CompanyName != nil {
		customer.CompanyName = req.CompanyName
	}
	if req.Email != nil {
		customer.Email = *req.Email
	}
	if req.PhonePrimary != nil {
		customer.PhonePrimary = *req.PhonePrimary
	}
	if req.PhoneSecondary != nil {
		customer.PhoneSecondary = req.PhoneSecondary
	}
	if req.BillingAddressStreet != nil {
		customer.BillingAddressStreet = *req.BillingAddressStreet
	}
	if req.BillingAddressCity != nil {
		customer.BillingAddressCity = *req.BillingAddressCity
	}
	if req.BillingAddressState != nil {
		customer.BillingAddressState = *req.BillingAddressState
	}
	if req.BillingAddressZip != nil {
		customer.BillingAddressZip = *req.BillingAddressZip
	}
	if req.ServiceAddressStreet != nil {
		customer.ServiceAddressStreet = req.ServiceAddressStreet
	}
	if req.ServiceAddressCity != nil {
		customer.ServiceAddressCity = req.ServiceAddressCity
	}
	if req.ServiceAddressState != nil {
		customer.ServiceAddressState = req.ServiceAddressState
	}
	if req.ServiceAddressZip != nil {
		customer.ServiceAddressZip = req.ServiceAddressZip
	}
	if req.CustomerType != nil {
		customer.CustomerType = *req.CustomerType
	}
	if req.Status != nil {
		customer.Status = *req.Status
	}
	if req.Notes != nil {
		customer.Notes = req.Notes
	}

	// Update customer
	if err := h.customerRepo.Update(customer); err != nil {
		if errors.Is(err, repository.ErrEmailAlreadyExists) {
			c.JSON(http.StatusConflict, models.ErrorResponse{
				Error:   "email_exists",
				Message: "A customer with this email already exists",
			})
			return
		}
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to update customer",
		})
		return
	}

	c.JSON(http.StatusOK, customer.ToResponse())
}

// DeleteCustomer handles DELETE /api/v1/customers/:id
// @Summary Delete a customer
// @Description Soft delete a customer by ID
// @Tags customers
// @Produce json
// @Param id path string true "Customer ID"
// @Success 204 "No content"
// @Failure 400 {object} models.ErrorResponse "Invalid ID"
// @Failure 404 {object} models.ErrorResponse "Customer not found"
// @Failure 500 {object} models.ErrorResponse "Internal server error"
// @Router /api/v1/customers/{id} [delete]
func (h *CustomerHandler) DeleteCustomer(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "invalid_id",
			Message: "Invalid customer ID format",
		})
		return
	}

	if err := h.customerRepo.Delete(id); err != nil {
		if errors.Is(err, repository.ErrCustomerNotFound) {
			c.JSON(http.StatusNotFound, models.ErrorResponse{
				Error:   "customer_not_found",
				Message: "Customer not found",
			})
			return
		}
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to delete customer",
		})
		return
	}

	c.Status(http.StatusNoContent)
}

// ListCustomers handles GET /api/v1/customers
// @Summary List customers
// @Description Get a paginated list of customers with optional filtering
// @Tags customers
// @Produce json
// @Param search query string false "Search term"
// @Param customer_type query string false "Customer type (residential/commercial)"
// @Param status query string false "Status (active/inactive/prospect)"
// @Param state query string false "State code"
// @Param city query string false "City name"
// @Param page query int false "Page number" default(1)
// @Param page_size query int false "Page size" default(20)
// @Param sort_by query string false "Sort field" default(created_at)
// @Param sort_order query string false "Sort order (ASC/DESC)" default(DESC)
// @Success 200 {object} models.CustomerListResponse
// @Failure 400 {object} models.ErrorResponse "Invalid request"
// @Failure 500 {object} models.ErrorResponse "Internal server error"
// @Router /api/v1/customers [get]
func (h *CustomerHandler) ListCustomers(c *gin.Context) {
	var filter models.CustomerListFilter

	if err := c.ShouldBindQuery(&filter); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "invalid_request",
			Message: err.Error(),
		})
		return
	}

	customers, total, err := h.customerRepo.List(&filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to retrieve customers",
		})
		return
	}

	// Convert to response format
	responses := make([]models.CustomerResponse, len(customers))
	for i, customer := range customers {
		responses[i] = customer.ToResponse()
	}

	// Calculate pagination info
	pageSize := filter.PageSize
	if pageSize < 1 {
		pageSize = 20
	}
	page := filter.Page
	if page < 1 {
		page = 1
	}

	response := models.CustomerListResponse{
		Customers:  responses,
		Total:      total,
		Page:       page,
		PageSize:   pageSize,
		TotalPages: repository.GetTotalPages(total, pageSize),
	}

	c.JSON(http.StatusOK, response)
}

// SearchCustomers handles GET /api/v1/customers/search
// @Summary Search customers
// @Description Search customers by name, email, company, or phone
// @Tags customers
// @Produce json
// @Param q query string true "Search query"
// @Success 200 {array} models.CustomerResponse
// @Failure 400 {object} models.ErrorResponse "Invalid request"
// @Failure 500 {object} models.ErrorResponse "Internal server error"
// @Router /api/v1/customers/search [get]
func (h *CustomerHandler) SearchCustomers(c *gin.Context) {
	query := c.Query("q")
	if query == "" {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "invalid_request",
			Message: "Search query is required",
		})
		return
	}

	customers, err := h.customerRepo.Search(query)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to search customers",
		})
		return
	}

	// Convert to response format
	responses := make([]models.CustomerResponse, len(customers))
	for i, customer := range customers {
		responses[i] = customer.ToResponse()
	}

	c.JSON(http.StatusOK, responses)
}

// GetCustomerByEmail handles GET /api/v1/customers/email/:email
// @Summary Get customer by email
// @Description Retrieve a customer by their email address
// @Tags customers
// @Produce json
// @Param email path string true "Customer email"
// @Success 200 {object} models.CustomerResponse
// @Failure 404 {object} models.ErrorResponse "Customer not found"
// @Failure 500 {object} models.ErrorResponse "Internal server error"
// @Router /api/v1/customers/email/{email} [get]
func (h *CustomerHandler) GetCustomerByEmail(c *gin.Context) {
	email := c.Param("email")

	customer, err := h.customerRepo.GetByEmail(email)
	if err != nil {
		if errors.Is(err, repository.ErrCustomerNotFound) {
			c.JSON(http.StatusNotFound, models.ErrorResponse{
				Error:   "customer_not_found",
				Message: "Customer not found",
			})
			return
		}
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to retrieve customer",
		})
		return
	}

	c.JSON(http.StatusOK, customer.ToResponse())
}

// AdvancedSearch handles GET /api/v1/customers/advanced-search
// @Summary Advanced full-text search with ranking
// @Description Search customers using PostgreSQL full-text search with ranking and filters
// @Tags customers
// @Produce json
// @Param q query string true "Search query"
// @Param type query string false "Customer type (residential/commercial)"
// @Param status query string false "Status (active/inactive/prospect)"
// @Param city query string false "City name"
// @Param state query string false "State code (2 letters)"
// @Param page query int false "Page number" default(1)
// @Param limit query int false "Results per page" default(20)
// @Param sort query string false "Sort field" default(rank)
// @Param order query string false "Sort order (ASC/DESC)" default(DESC)
// @Success 200 {object} models.CustomerListResponse
// @Failure 400 {object} models.ErrorResponse "Invalid request"
// @Failure 500 {object} models.ErrorResponse "Internal server error"
// @Router /api/v1/customers/advanced-search [get]
func (h *CustomerHandler) AdvancedSearch(c *gin.Context) {
	// Parse query parameters
	query := c.Query("q")
	customerType := models.CustomerType(c.Query("type"))
	status := models.CustomerStatus(c.Query("status"))
	city := c.Query("city")
	state := c.Query("state")

	page := 1
	if pageStr := c.Query("page"); pageStr != "" {
		if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
			page = p
		}
	}

	limit := 20
	if limitStr := c.Query("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			limit = l
			if limit > 100 {
				limit = 100
			}
		}
	}

	sortBy := c.DefaultQuery("sort", "rank")
	sortOrder := strings.ToUpper(c.DefaultQuery("order", "DESC"))

	if sortOrder != "ASC" && sortOrder != "DESC" {
		sortOrder = "DESC"
	}

	// Validate query
	if query == "" {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "invalid_request",
			Message: "Search query is required",
		})
		return
	}

	// Calculate offset
	offset := (page - 1) * limit

	// Perform full-text search
	customers, total, err := h.customerRepo.FullTextSearch(
		query,
		customerType,
		status,
		city,
		state,
		limit,
		offset,
		sortBy,
		sortOrder,
	)

	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to search customers",
		})
		return
	}

	// Convert to response format
	responses := make([]models.CustomerResponse, len(customers))
	for i, customer := range customers {
		responses[i] = customer.ToResponse()
	}

	// Calculate pagination info
	response := models.CustomerListResponse{
		Customers:  responses,
		Total:      total,
		Page:       page,
		PageSize:   limit,
		TotalPages: repository.GetTotalPages(total, limit),
	}

	c.JSON(http.StatusOK, response)
}

// Autocomplete handles GET /api/v1/customers/autocomplete
// @Summary Autocomplete customer search
// @Description Get autocomplete suggestions for customer search
// @Tags customers
// @Produce json
// @Param q query string true "Search prefix"
// @Param limit query int false "Maximum results" default(10)
// @Success 200 {array} map[string]interface{}
// @Failure 400 {object} models.ErrorResponse "Invalid request"
// @Failure 500 {object} models.ErrorResponse "Internal server error"
// @Router /api/v1/customers/autocomplete [get]
func (h *CustomerHandler) Autocomplete(c *gin.Context) {
	prefix := c.Query("q")
	if prefix == "" {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "invalid_request",
			Message: "Search prefix is required",
		})
		return
	}

	limit := 10
	if limitStr := c.Query("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			limit = l
			if limit > 50 {
				limit = 50
			}
		}
	}

	results, err := h.customerRepo.Autocomplete(prefix, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to get autocomplete suggestions",
		})
		return
	}

	c.JSON(http.StatusOK, results)
}

// GetSearchStats handles GET /api/v1/customers/stats
// @Summary Get customer search statistics
// @Description Get statistics about customers for search optimization
// @Tags customers
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Failure 500 {object} models.ErrorResponse "Internal server error"
// @Router /api/v1/customers/stats [get]
func (h *CustomerHandler) GetSearchStats(c *gin.Context) {
	stats, err := h.customerRepo.GetSearchStats()
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to get search statistics",
		})
		return
	}

	c.JSON(http.StatusOK, stats)
}
