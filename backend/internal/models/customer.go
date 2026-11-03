package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
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

// PreferredContactMethod represents how a customer prefers to be contacted
type PreferredContactMethod string

const (
	ContactMethodEmail PreferredContactMethod = "email"
	ContactMethodPhone PreferredContactMethod = "phone"
	ContactMethodSMS   PreferredContactMethod = "sms"
	ContactMethodMail  PreferredContactMethod = "mail"
	ContactMethodAny   PreferredContactMethod = "any"
)

// Customer represents a customer in the system
type Customer struct {
	ID                   uuid.UUID      `json:"id" gorm:"type:uuid;primary_key;default:uuid_generate_v4()"`
	TenantID             uuid.UUID      `json:"tenant_id" gorm:"type:uuid;not null;index"` // set from the caller's tenant (internal/tenancy)
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

	// Contact preferences
	PreferredContactMethod PreferredContactMethod `json:"preferred_contact_method" gorm:"type:varchar(20);default:'any'"`
	DoNotEmail             bool                   `json:"do_not_email" gorm:"default:false"`
	DoNotCall              bool                   `json:"do_not_call" gorm:"default:false"`
	DoNotSMS               bool                   `json:"do_not_sms" gorm:"default:false"`
	DoNotMail              bool                   `json:"do_not_mail" gorm:"default:false"`

	// Marketing consent tracking
	MarketingConsent   bool       `json:"marketing_consent" gorm:"default:false"`
	MarketingConsentAt *time.Time `json:"marketing_consent_at" gorm:"type:timestamp"`
	MarketingConsentIP *string    `json:"marketing_consent_ip" gorm:"type:varchar(45)"` // Supports IPv6

	// Billing rules (T187)
	BillingRules

	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index"`

	// Billing rules to take from the tenant's defaults on insert (T235);
	// unexported, so GORM never maps them.
	inheritConsolidated bool
	inheritPORequired   bool
}

// InheritTenantBillingDefaults makes the customer, when inserted, take the
// tenant's default consolidated-billing and PO-required rules
// (TenantSettings.DefaultConsolidatedBilling / DefaultPORequired, set in
// onboarding) for each rule the create request left out (T235).
func (c *Customer) InheritTenantBillingDefaults(r *BillingRulesRequest) {
	c.inheritConsolidated = r == nil || r.ConsolidatedBilling == nil
	c.inheritPORequired = r == nil || r.PORequired == nil
}

// BeforeCreate applies the tenant's billing defaults requested with
// InheritTenantBillingDefaults, read in the insert's transaction.
func (c *Customer) BeforeCreate(tx *gorm.DB) error {
	if !c.inheritConsolidated && !c.inheritPORequired {
		return nil
	}
	tenantID := documentTenantID(tx, c.TenantID)
	if tenantID == uuid.Nil {
		return nil
	}
	settings := TenantSettingsInTx(tx.Session(&gorm.Session{NewDB: true}), tenantID)
	if c.inheritConsolidated {
		c.ConsolidatedBilling = settings.DefaultConsolidatedBilling
	}
	if c.inheritPORequired {
		c.PORequired = settings.DefaultPORequired
	}
	return nil
}

// BillingRules are a customer's purchase-order, approval and invoicing
// rules (T187). Embedded in Customer and in its API response.
type BillingRules struct {
	// PORequired: invoices can't be sent without a PO number.
	PORequired bool `json:"po_required" gorm:"not null"`
	// DefaultPONumber is copied onto new invoices that have none.
	DefaultPONumber *string `json:"default_po_number" gorm:"type:varchar(100)"`
	// The customer's named approver for extra work.
	BillingApproverName  *string `json:"billing_approver_name" gorm:"type:varchar(100)"`
	BillingApproverEmail *string `json:"billing_approver_email" gorm:"type:varchar(255)"`
	// NTELimit (not-to-exceed): extra work estimated above it can't be
	// scheduled until the approver's OK is recorded. Nil means no limit.
	NTELimit *decimal.Decimal `json:"nte_limit" gorm:"type:numeric(12,2)" swaggertype:"string" example:"500.00"`
	// ConsolidatedBilling: one monthly invoice per customer, a line per site (T188).
	ConsolidatedBilling bool `json:"consolidated_billing" gorm:"not null"`
}

// BillingRulesRequest sets billing rules on create or update; omitted fields
// are left unchanged. An empty string clears a text field.
type BillingRulesRequest struct {
	PORequired           *bool            `json:"po_required"`
	DefaultPONumber      *string          `json:"default_po_number" binding:"omitempty,max=100"`
	BillingApproverName  *string          `json:"billing_approver_name" binding:"omitempty,max=100"`
	BillingApproverEmail *string          `json:"billing_approver_email" binding:"omitempty,max=255"`
	NTELimit             *decimal.Decimal `json:"nte_limit" swaggertype:"string" example:"500.00"`
	// ClearNTELimit removes the not-to-exceed limit.
	ClearNTELimit       bool  `json:"clear_nte_limit"`
	ConsolidatedBilling *bool `json:"consolidated_billing"`
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

	// Contact preferences
	PreferredContactMethod PreferredContactMethod `json:"preferred_contact_method" binding:"omitempty,oneof=email phone sms mail any"`
	DoNotEmail             *bool                  `json:"do_not_email" binding:"omitempty"`
	DoNotCall              *bool                  `json:"do_not_call" binding:"omitempty"`
	DoNotSMS               *bool                  `json:"do_not_sms" binding:"omitempty"`
	DoNotMail              *bool                  `json:"do_not_mail" binding:"omitempty"`

	// Marketing consent
	MarketingConsent *bool `json:"marketing_consent" binding:"omitempty"`

	// Billing rules (T187)
	BillingRulesRequest
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

	// Contact preferences
	PreferredContactMethod *PreferredContactMethod `json:"preferred_contact_method" binding:"omitempty,oneof=email phone sms mail any"`
	DoNotEmail             *bool                   `json:"do_not_email" binding:"omitempty"`
	DoNotCall              *bool                   `json:"do_not_call" binding:"omitempty"`
	DoNotSMS               *bool                   `json:"do_not_sms" binding:"omitempty"`
	DoNotMail              *bool                   `json:"do_not_mail" binding:"omitempty"`

	// Marketing consent
	MarketingConsent *bool `json:"marketing_consent" binding:"omitempty"`

	// Billing rules (T187)
	BillingRulesRequest
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

	// Contact preferences
	PreferredContactMethod PreferredContactMethod `json:"preferred_contact_method"`
	DoNotEmail             bool                   `json:"do_not_email"`
	DoNotCall              bool                   `json:"do_not_call"`
	DoNotSMS               bool                   `json:"do_not_sms"`
	DoNotMail              bool                   `json:"do_not_mail"`

	// Marketing consent
	MarketingConsent   bool       `json:"marketing_consent"`
	MarketingConsentAt *time.Time `json:"marketing_consent_at"`

	// Billing rules (T187)
	BillingRules

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// ToResponse converts a Customer model to a CustomerResponse
func (c *Customer) ToResponse() CustomerResponse {
	return CustomerResponse{
		ID:                     c.ID,
		FirstName:              c.FirstName,
		LastName:               c.LastName,
		DisplayName:            c.GetDisplayName(),
		CompanyName:            c.CompanyName,
		Email:                  c.Email,
		PhonePrimary:           c.PhonePrimary,
		PhoneSecondary:         c.PhoneSecondary,
		BillingAddressStreet:   c.BillingAddressStreet,
		BillingAddressCity:     c.BillingAddressCity,
		BillingAddressState:    c.BillingAddressState,
		BillingAddressZip:      c.BillingAddressZip,
		BillingAddressFull:     c.GetBillingAddress(),
		ServiceAddressStreet:   c.ServiceAddressStreet,
		ServiceAddressCity:     c.ServiceAddressCity,
		ServiceAddressState:    c.ServiceAddressState,
		ServiceAddressZip:      c.ServiceAddressZip,
		ServiceAddressFull:     c.GetServiceAddress(),
		CustomerType:           c.CustomerType,
		Status:                 c.Status,
		Notes:                  c.Notes,
		PreferredContactMethod: c.PreferredContactMethod,
		DoNotEmail:             c.DoNotEmail,
		DoNotCall:              c.DoNotCall,
		DoNotSMS:               c.DoNotSMS,
		DoNotMail:              c.DoNotMail,
		MarketingConsent:       c.MarketingConsent,
		MarketingConsentAt:     c.MarketingConsentAt,
		BillingRules:           c.BillingRules,
		CreatedAt:              c.CreatedAt,
		UpdatedAt:              c.UpdatedAt,
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

// CustomerStats is GET /customers/stats: the caller's customer counts (T257).
type CustomerStats struct {
	TotalCustomers    int64            `json:"total_customers"`
	ActiveCustomers   int64            `json:"active_customers"`
	InactiveCustomers int64            `json:"inactive_customers"`
	Prospects         int64            `json:"prospects"`
	ByStatus          map[string]int64 `json:"by_status"`
}
