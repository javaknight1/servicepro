package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// CustomerType represents the type of customer
type CustomerType string

const (
	CustomerTypeResidential CustomerType = "residential"
	CustomerTypeCommercial  CustomerType = "commercial"
)

// CustomerStatus represents the status of a customer
type CustomerStatus string

const (
	CustomerStatusActive   CustomerStatus = "active"
	CustomerStatusInactive CustomerStatus = "inactive"
	CustomerStatusProspect CustomerStatus = "prospect"
)

// Customer represents a customer in the system
type Customer struct {
	ID                   uuid.UUID      `json:"id" gorm:"type:uuid;primary_key;default:uuid_generate_v4()"`
	FirstName            string         `json:"first_name" gorm:"type:varchar(50);not null"`
	LastName             string         `json:"last_name" gorm:"type:varchar(50);not null"`
	CompanyName          *string        `json:"company_name" gorm:"type:varchar(100)"`
	Email                string         `json:"email" gorm:"type:varchar(255);uniqueIndex:idx_customers_email;not null"`
	PhonePrimary         string         `json:"phone_primary" gorm:"type:varchar(20);not null"`
	PhoneSecondary       *string        `json:"phone_secondary" gorm:"type:varchar(20)"`
	BillingAddressStreet string         `json:"billing_address_street" gorm:"type:varchar(100);not null"`
	BillingAddressCity   string         `json:"billing_address_city" gorm:"type:varchar(50);not null"`
	BillingAddressState  string         `json:"billing_address_state" gorm:"type:char(2);not null"`
	BillingAddressZip    string         `json:"billing_address_zip" gorm:"type:varchar(10);not null"`
	ServiceAddressStreet *string        `json:"service_address_street" gorm:"type:varchar(100)"`
	ServiceAddressCity   *string        `json:"service_address_city" gorm:"type:varchar(50)"`
	ServiceAddressState  *string        `json:"service_address_state" gorm:"type:char(2)"`
	ServiceAddressZip    *string        `json:"service_address_zip" gorm:"type:varchar(10)"`
	CustomerType         CustomerType   `json:"customer_type" gorm:"type:customer_type;not null;default:'residential'"`
	Status               CustomerStatus `json:"status" gorm:"type:customer_status;not null;default:'prospect'"`
	Notes                *string        `json:"notes" gorm:"type:text"`
	CreatedAt            time.Time      `json:"created_at"`
	UpdatedAt            time.Time      `json:"updated_at"`
	DeletedAt            gorm.DeletedAt `json:"-" gorm:"index"`
}

// TableName specifies the table name for Customer model
func (Customer) TableName() string {
	return "customers"
}

// GetDisplayName returns a formatted display name for the customer
func (c *Customer) GetDisplayName() string {
	if c.CompanyName != nil && *c.CompanyName != "" {
		return *c.CompanyName + " (" + c.FirstName + " " + c.LastName + ")"
	}
	return c.FirstName + " " + c.LastName
}

// GetBillingAddress returns the full formatted billing address
func (c *Customer) GetBillingAddress() string {
	return c.BillingAddressStreet + ", " + c.BillingAddressCity + ", " + c.BillingAddressState + " " + c.BillingAddressZip
}

// GetServiceAddress returns the full formatted service address
func (c *Customer) GetServiceAddress() string {
	if c.ServiceAddressStreet != nil {
		return *c.ServiceAddressStreet + ", " + *c.ServiceAddressCity + ", " + *c.ServiceAddressState + " " + *c.ServiceAddressZip
	}
	return c.GetBillingAddress()
}

// HasSeparateServiceAddress checks if the customer has a different service address
func (c *Customer) HasSeparateServiceAddress() bool {
	return c.ServiceAddressStreet != nil
}

// CreateCustomerRequest represents the request payload for creating a customer
type CreateCustomerRequest struct {
	FirstName            string         `json:"first_name" binding:"required,max=50"`
	LastName             string         `json:"last_name" binding:"required,max=50"`
	CompanyName          *string        `json:"company_name" binding:"omitempty,max=100"`
	Email                string         `json:"email" binding:"required,email,max=255"`
	PhonePrimary         string         `json:"phone_primary" binding:"required,max=20"`
	PhoneSecondary       *string        `json:"phone_secondary" binding:"omitempty,max=20"`
	BillingAddressStreet string         `json:"billing_address_street" binding:"required,max=100"`
	BillingAddressCity   string         `json:"billing_address_city" binding:"required,max=50"`
	BillingAddressState  string         `json:"billing_address_state" binding:"required,len=2"`
	BillingAddressZip    string         `json:"billing_address_zip" binding:"required,max=10"`
	ServiceAddressStreet *string        `json:"service_address_street" binding:"omitempty,max=100"`
	ServiceAddressCity   *string        `json:"service_address_city" binding:"omitempty,max=50"`
	ServiceAddressState  *string        `json:"service_address_state" binding:"omitempty,len=2"`
	ServiceAddressZip    *string        `json:"service_address_zip" binding:"omitempty,max=10"`
	CustomerType         CustomerType   `json:"customer_type" binding:"required,oneof=residential commercial"`
	Status               CustomerStatus `json:"status" binding:"omitempty,oneof=active inactive prospect"`
	Notes                *string        `json:"notes" binding:"omitempty"`
}

// UpdateCustomerRequest represents the request payload for updating a customer
type UpdateCustomerRequest struct {
	FirstName            *string         `json:"first_name" binding:"omitempty,max=50"`
	LastName             *string         `json:"last_name" binding:"omitempty,max=50"`
	CompanyName          *string         `json:"company_name" binding:"omitempty,max=100"`
	Email                *string         `json:"email" binding:"omitempty,email,max=255"`
	PhonePrimary         *string         `json:"phone_primary" binding:"omitempty,max=20"`
	PhoneSecondary       *string         `json:"phone_secondary" binding:"omitempty,max=20"`
	BillingAddressStreet *string         `json:"billing_address_street" binding:"omitempty,max=100"`
	BillingAddressCity   *string         `json:"billing_address_city" binding:"omitempty,max=50"`
	BillingAddressState  *string         `json:"billing_address_state" binding:"omitempty,len=2"`
	BillingAddressZip    *string         `json:"billing_address_zip" binding:"omitempty,max=10"`
	ServiceAddressStreet *string         `json:"service_address_street" binding:"omitempty,max=100"`
	ServiceAddressCity   *string         `json:"service_address_city" binding:"omitempty,max=50"`
	ServiceAddressState  *string         `json:"service_address_state" binding:"omitempty,len=2"`
	ServiceAddressZip    *string         `json:"service_address_zip" binding:"omitempty,max=10"`
	CustomerType         *CustomerType   `json:"customer_type" binding:"omitempty,oneof=residential commercial"`
	Status               *CustomerStatus `json:"status" binding:"omitempty,oneof=active inactive prospect"`
	Notes                *string         `json:"notes" binding:"omitempty"`
}

// CustomerResponse represents the response payload for a customer
type CustomerResponse struct {
	ID                   uuid.UUID      `json:"id"`
	FirstName            string         `json:"first_name"`
	LastName             string         `json:"last_name"`
	DisplayName          string         `json:"display_name"`
	CompanyName          *string        `json:"company_name"`
	Email                string         `json:"email"`
	PhonePrimary         string         `json:"phone_primary"`
	PhoneSecondary       *string        `json:"phone_secondary"`
	BillingAddressStreet string         `json:"billing_address_street"`
	BillingAddressCity   string         `json:"billing_address_city"`
	BillingAddressState  string         `json:"billing_address_state"`
	BillingAddressZip    string         `json:"billing_address_zip"`
	BillingAddressFull   string         `json:"billing_address_full"`
	ServiceAddressStreet *string        `json:"service_address_street"`
	ServiceAddressCity   *string        `json:"service_address_city"`
	ServiceAddressState  *string        `json:"service_address_state"`
	ServiceAddressZip    *string        `json:"service_address_zip"`
	ServiceAddressFull   string         `json:"service_address_full"`
	CustomerType         CustomerType   `json:"customer_type"`
	Status               CustomerStatus `json:"status"`
	Notes                *string        `json:"notes"`
	CreatedAt            time.Time      `json:"created_at"`
	UpdatedAt            time.Time      `json:"updated_at"`
}

// ToResponse converts a Customer model to a CustomerResponse
func (c *Customer) ToResponse() CustomerResponse {
	return CustomerResponse{
		ID:                   c.ID,
		FirstName:            c.FirstName,
		LastName:             c.LastName,
		DisplayName:          c.GetDisplayName(),
		CompanyName:          c.CompanyName,
		Email:                c.Email,
		PhonePrimary:         c.PhonePrimary,
		PhoneSecondary:       c.PhoneSecondary,
		BillingAddressStreet: c.BillingAddressStreet,
		BillingAddressCity:   c.BillingAddressCity,
		BillingAddressState:  c.BillingAddressState,
		BillingAddressZip:    c.BillingAddressZip,
		BillingAddressFull:   c.GetBillingAddress(),
		ServiceAddressStreet: c.ServiceAddressStreet,
		ServiceAddressCity:   c.ServiceAddressCity,
		ServiceAddressState:  c.ServiceAddressState,
		ServiceAddressZip:    c.ServiceAddressZip,
		ServiceAddressFull:   c.GetServiceAddress(),
		CustomerType:         c.CustomerType,
		Status:               c.Status,
		Notes:                c.Notes,
		CreatedAt:            c.CreatedAt,
		UpdatedAt:            c.UpdatedAt,
	}
}

// CustomerListFilter represents query parameters for filtering customers
type CustomerListFilter struct {
	Search       string         `form:"search"`
	CustomerType CustomerType   `form:"customer_type"`
	Status       CustomerStatus `form:"status"`
	State        string         `form:"state"`
	City         string         `form:"city"`
	Page         int            `form:"page"`
	PageSize     int            `form:"page_size"`
	SortBy       string         `form:"sort_by"`
	SortOrder    string         `form:"sort_order"`
}

// CustomerListResponse represents a paginated list of customers
type CustomerListResponse struct {
	Customers  []CustomerResponse `json:"customers"`
	Total      int64              `json:"total"`
	Page       int                `json:"page"`
	PageSize   int                `json:"page_size"`
	TotalPages int                `json:"total_pages"`
}
