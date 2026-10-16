package repository

import (
	"context"
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

// WithContext returns a copy of the repository bound to ctx (tenant scoping
// via internal/tenancy reads the tenant from it).
func (r *CustomerRepository) WithContext(ctx context.Context) CustomerRepositoryInterface {
	return &CustomerRepository{db: r.db.WithContext(ctx)}
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

// FullTextSearch searches customers with filters and pagination.
//
// It previously called search_customers()/autocomplete_customers() SQL
// functions that were never created by any migration, and raw SQL bypasses
// tenant scoping (internal/tenancy). It is now a plain GORM query, so the
// caller's tenant filter is applied automatically.
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

	q := r.db.Model(&models.Customer{}).Where("deleted_at IS NULL")
	if term := strings.TrimSpace(query); term != "" {
		like := "%" + strings.ToLower(term) + "%"
		q = q.Where(
			"LOWER(first_name) LIKE ? OR LOWER(last_name) LIKE ? OR LOWER(email) LIKE ? OR LOWER(company_name) LIKE ? OR LOWER(COALESCE(notes, '')) LIKE ?",
			like, like, like, like, like,
		)
	}
	if customerType != "" {
		q = q.Where("customer_type = ?", customerType)
	}
	if status != "" {
		q = q.Where("status = ?", status)
	}
	if city != "" {
		q = q.Where("LOWER(billing_address_city) = LOWER(?)", city)
	}
	if state != "" {
		q = q.Where("billing_address_state = ?", strings.ToUpper(state))
	}

	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if limit <= 0 || limit > 100 {
		limit = 20
	}
	if offset < 0 {
		offset = 0
	}
	order := utils.SafeOrderClause(sortBy, sortOrder, utils.CustomerAllowedSortColumns, "last_name")
	if err := q.Order(order).Limit(limit).Offset(offset).Find(&customers).Error; err != nil {
		return nil, 0, err
	}
	return customers, total, nil
}

// Autocomplete performs prefix-based autocomplete search
func (r *CustomerRepository) Autocomplete(prefix string, limit int) ([]map[string]interface{}, error) {
	if limit <= 0 || limit > 50 {
		limit = 10
	}
	like := strings.ToLower(strings.TrimSpace(prefix)) + "%"

	var customers []models.Customer
	err := r.db.Model(&models.Customer{}).
		Where("deleted_at IS NULL").
		Where("LOWER(first_name) LIKE ? OR LOWER(last_name) LIKE ? OR LOWER(email) LIKE ? OR LOWER(company_name) LIKE ?", like, like, like, like).
		Order("last_name ASC, first_name ASC").
		Limit(limit).
		Find(&customers).Error
	if err != nil {
		return nil, err
	}

	results := make([]map[string]interface{}, 0, len(customers))
	for _, c := range customers {
		display := strings.TrimSpace(c.FirstName + " " + c.LastName)
		if c.CompanyName != nil && *c.CompanyName != "" {
			display = display + " (" + *c.CompanyName + ")"
		}
		results = append(results, map[string]interface{}{
			"id":            c.ID,
			"display_name":  display,
			"email":         c.Email,
			"customer_type": c.CustomerType,
			"status":        c.Status,
		})
	}
	return results, nil
}

// GetSearchStats retrieves customer counts for the caller's tenant
func (r *CustomerRepository) GetSearchStats() (map[string]interface{}, error) {
	type row struct {
		Status string
		Count  int64
	}
	var rows []row
	if err := r.db.Model(&models.Customer{}).
		Select("status, COUNT(*) AS count").
		Where("deleted_at IS NULL").
		Group("status").
		Scan(&rows).Error; err != nil {
		return nil, err
	}

	var total int64
	byStatus := make(map[string]int64, len(rows))
	for _, rw := range rows {
		byStatus[rw.Status] = rw.Count
		total += rw.Count
	}
	return map[string]interface{}{
		"total_customers": total,
		"by_status":       byStatus,
	}, nil
}
