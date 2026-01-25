package repository

import (
	"errors"
	"math"
	"strings"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/javaknight1/servicepro/backend/internal/models"
	"github.com/javaknight1/servicepro/backend/internal/utils"
)

var (
	ErrCustomerNotFound   = errors.New("customer not found")
	ErrEmailAlreadyExists = errors.New("email already exists")
	ErrPhoneAlreadyExists = errors.New("phone already exists")
)

// CustomerRepository implements CustomerRepositoryInterface using GORM
type CustomerRepository struct {
	db *gorm.DB
}

// NewCustomerRepository creates a new customer repository
func NewCustomerRepository(db *gorm.DB) *CustomerRepository {
	return &CustomerRepository{db: db}
}

// Create creates a new customer
func (r *CustomerRepository) Create(customer *models.Customer) error {
	if err := r.db.Create(customer).Error; err != nil {
		if strings.Contains(err.Error(), "duplicate key") || strings.Contains(err.Error(), "unique constraint") {
			if strings.Contains(err.Error(), "email") {
				return ErrEmailAlreadyExists
			}
		}
		return err
	}
	return nil
}

// GetByID retrieves a customer by ID
func (r *CustomerRepository) GetByID(id uuid.UUID) (*models.Customer, error) {
	var customer models.Customer
	err := r.db.Where("id = ? AND deleted_at IS NULL", id).First(&customer).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrCustomerNotFound
		}
		return nil, err
	}
	return &customer, nil
}

// GetByEmail retrieves a customer by email
func (r *CustomerRepository) GetByEmail(email string) (*models.Customer, error) {
	var customer models.Customer
	err := r.db.Where("LOWER(email) = LOWER(?) AND deleted_at IS NULL", email).First(&customer).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrCustomerNotFound
		}
		return nil, err
	}
	return &customer, nil
}

// Update updates a customer
func (r *CustomerRepository) Update(customer *models.Customer) error {
	result := r.db.Model(customer).Where("id = ? AND deleted_at IS NULL", customer.ID).Updates(customer)
	if result.Error != nil {
		if strings.Contains(result.Error.Error(), "duplicate key") || strings.Contains(result.Error.Error(), "unique constraint") {
			if strings.Contains(result.Error.Error(), "email") {
				return ErrEmailAlreadyExists
			}
		}
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrCustomerNotFound
	}
	return nil
}

// Delete soft deletes a customer
func (r *CustomerRepository) Delete(id uuid.UUID) error {
	result := r.db.Model(&models.Customer{}).Where("id = ?", id).Update("deleted_at", gorm.Expr("CURRENT_TIMESTAMP"))
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrCustomerNotFound
	}
	return nil
}

// List retrieves customers with pagination and filtering
func (r *CustomerRepository) List(filter *models.CustomerListFilter) ([]models.Customer, int64, error) {
	var customers []models.Customer
	var total int64

	// Set defaults
	page := filter.Page
	if page < 1 {
		page = 1
	}
	pageSize := filter.PageSize
	if pageSize < 1 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}

	// Build base query
	query := r.db.Model(&models.Customer{}).Where("deleted_at IS NULL")

	// Apply filters
	if filter.Search != "" {
		searchTerm := "%" + strings.ToLower(filter.Search) + "%"
		query = query.Where(
			"LOWER(first_name) LIKE ? OR LOWER(last_name) LIKE ? OR LOWER(email) LIKE ? OR LOWER(company_name) LIKE ? OR phone_primary LIKE ? OR phone_secondary LIKE ?",
			searchTerm, searchTerm, searchTerm, searchTerm, searchTerm, searchTerm,
		)
	}

	if filter.CustomerType != "" {
		query = query.Where("customer_type = ?", filter.CustomerType)
	}

	if filter.Status != "" {
		query = query.Where("status = ?", filter.Status)
	}

	if filter.State != "" {
		query = query.Where("billing_address_state = ?", strings.ToUpper(filter.State))
	}

	if filter.City != "" {
		query = query.Where("LOWER(billing_address_city) = LOWER(?)", filter.City)
	}

	// Count total
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// Apply sorting (with SQL injection protection)
	orderClause := utils.SafeOrderClause(
		filter.SortBy,
		filter.SortOrder,
		utils.CustomerAllowedSortColumns,
		"created_at",
	)
	query = query.Order(orderClause)

	// Apply pagination
	offset := (page - 1) * pageSize
	query = query.Offset(offset).Limit(pageSize)

	// Execute query
	if err := query.Find(&customers).Error; err != nil {
		return nil, 0, err
	}

	return customers, total, nil
}

// EmailExists checks if an email already exists
func (r *CustomerRepository) EmailExists(email string) (bool, error) {
	var count int64
	err := r.db.Model(&models.Customer{}).
		Where("LOWER(email) = LOWER(?) AND deleted_at IS NULL", email).
		Count(&count).Error
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// PhoneExists checks if a phone number already exists
func (r *CustomerRepository) PhoneExists(phone string) (bool, error) {
	var count int64
	err := r.db.Model(&models.Customer{}).
		Where("(phone_primary = ? OR phone_secondary = ?) AND deleted_at IS NULL", phone, phone).
		Count(&count).Error
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// Search searches customers by various fields
func (r *CustomerRepository) Search(query string) ([]models.Customer, error) {
	var customers []models.Customer
	searchTerm := "%" + strings.ToLower(query) + "%"

	err := r.db.Where("deleted_at IS NULL").
		Where(
			"LOWER(first_name) LIKE ? OR LOWER(last_name) LIKE ? OR LOWER(email) LIKE ? OR LOWER(company_name) LIKE ? OR phone_primary LIKE ? OR phone_secondary LIKE ?",
			searchTerm, searchTerm, searchTerm, searchTerm, searchTerm, searchTerm,
		).
		Order("last_name ASC, first_name ASC").
		Limit(50).
		Find(&customers).Error

	if err != nil {
		return nil, err
	}

	return customers, nil
}

// GetByStatus retrieves customers by status
func (r *CustomerRepository) GetByStatus(status models.CustomerStatus) ([]models.Customer, error) {
	var customers []models.Customer
	err := r.db.Where("status = ? AND deleted_at IS NULL", status).
		Order("created_at DESC").
		Find(&customers).Error
	if err != nil {
		return nil, err
	}
	return customers, nil
}

// GetByType retrieves customers by type
func (r *CustomerRepository) GetByType(customerType models.CustomerType) ([]models.Customer, error) {
	var customers []models.Customer
	err := r.db.Where("customer_type = ? AND deleted_at IS NULL", customerType).
		Order("created_at DESC").
		Find(&customers).Error
	if err != nil {
		return nil, err
	}
	return customers, nil
}

// GetTotalPages calculates the total number of pages
func GetTotalPages(total int64, pageSize int) int {
	if pageSize == 0 {
		return 0
	}
	return int(math.Ceil(float64(total) / float64(pageSize)))
}

// FullTextSearch performs PostgreSQL full-text search with ranking
func (r *CustomerRepository) FullTextSearch(
	query string,
	customerType models.CustomerType,
	status models.CustomerStatus,
	city, state string,
	limit, offset int,
	sortBy, sortOrder string,
) ([]models.Customer, int64, error) {
	var customers []models.Customer
	var total int64

	// Build base query using the PostgreSQL search function
	baseQuery := `
		SELECT
			id, first_name, last_name, company_name, email,
			phone_primary, phone_secondary,
			billing_address_street, billing_address_city, billing_address_state, billing_address_zip,
			service_address_street, service_address_city, service_address_state, service_address_zip,
			customer_type, status, notes, created_at, updated_at
		FROM search_customers($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`

	// Execute search
	err := r.db.Raw(baseQuery,
		query,
		nullString(string(customerType)),
		nullString(string(status)),
		nullString(city),
		nullString(state),
		limit,
		offset,
		sortBy,
		sortOrder,
	).Scan(&customers).Error

	if err != nil {
		return nil, 0, err
	}

	// Get total count for pagination
	countQuery := `
		SELECT COUNT(*)
		FROM search_customers($1, $2, $3, $4, $5, 999999, 0, 'rank', 'DESC')
	`

	err = r.db.Raw(countQuery,
		query,
		nullString(string(customerType)),
		nullString(string(status)),
		nullString(city),
		nullString(state),
	).Scan(&total).Error

	if err != nil {
		return nil, 0, err
	}

	return customers, total, nil
}

// Autocomplete performs prefix-based autocomplete search
func (r *CustomerRepository) Autocomplete(prefix string, limit int) ([]map[string]interface{}, error) {
	var results []map[string]interface{}

	query := `
		SELECT id, display_name, email, customer_type, status
		FROM autocomplete_customers($1, $2)
	`

	err := r.db.Raw(query, prefix, limit).Scan(&results).Error
	if err != nil {
		return nil, err
	}

	return results, nil
}

// GetSearchStats retrieves search statistics
func (r *CustomerRepository) GetSearchStats() (map[string]interface{}, error) {
	var stats map[string]interface{}

	query := `SELECT * FROM customer_search_stats`

	err := r.db.Raw(query).Scan(&stats).Error
	if err != nil {
		return nil, err
	}

	return stats, nil
}

// Helper function to convert string to nullable string for PostgreSQL
func nullString(s string) interface{} {
	if s == "" {
		return nil
	}
	return s
}
