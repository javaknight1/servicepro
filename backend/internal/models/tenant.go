package models

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Tenant represents an organization/tenant in the multi-tenant system
type Tenant struct {
	ID        uuid.UUID      `json:"id" gorm:"type:uuid;primary_key;default:uuid_generate_v4()"`
	Name      string         `json:"name" gorm:"not null;size:200"`
	Slug      string         `json:"slug" gorm:"uniqueIndex;not null;size:100"`
	OwnerID   uuid.UUID      `json:"owner_id" gorm:"type:uuid;not null"`
	Email     string         `json:"email,omitempty" gorm:"size:255"`
	Phone     string         `json:"phone,omitempty" gorm:"size:20"`
	LogoURL   string         `json:"logo_url,omitempty" gorm:"type:text"`
	Settings  TenantSettings `json:"settings" gorm:"type:jsonb;default:'{}'"`
	IsActive  bool           `json:"is_active" gorm:"default:true"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index"`

	// Relations
	Owner   *User        `json:"owner,omitempty" gorm:"foreignKey:OwnerID"`
	Members []TenantUser `json:"members,omitempty" gorm:"foreignKey:TenantID"`
}

// TableName specifies the table name for Tenant
func (Tenant) TableName() string {
	return "tenants"
}

// TenantSettings represents configurable tenant settings
type TenantSettings struct {
	IsDefault            bool       `json:"isDefault,omitempty"`
	Timezone             string     `json:"timezone,omitempty"`
	DateFormat           string     `json:"dateFormat,omitempty"`
	Currency             string     `json:"currency,omitempty"`
	InvoicePrefix        string     `json:"invoicePrefix,omitempty"`
	QuotePrefix          string     `json:"quotePrefix,omitempty"`
	JobPrefix            string     `json:"jobPrefix,omitempty"`
	DefaultPaymentTermID *uuid.UUID `json:"defaultPaymentTermId,omitempty"`
	DefaultTaxRateID     *uuid.UUID `json:"defaultTaxRateId,omitempty"`
}

// Value implements driver.Valuer for database serialization
func (s TenantSettings) Value() (driver.Value, error) {
	return json.Marshal(s)
}

// Scan implements sql.Scanner for database deserialization
func (s *TenantSettings) Scan(value interface{}) error {
	if value == nil {
		*s = TenantSettings{}
		return nil
	}

	bytes, ok := value.([]byte)
	if !ok {
		return errors.New("failed to scan TenantSettings: invalid type")
	}

	if len(bytes) == 0 {
		*s = TenantSettings{}
		return nil
	}

	return json.Unmarshal(bytes, s)
}

// TenantUser represents a user's membership in a tenant
type TenantUser struct {
	ID         uuid.UUID  `json:"id" gorm:"type:uuid;primary_key;default:uuid_generate_v4()"`
	TenantID   uuid.UUID  `json:"tenant_id" gorm:"type:uuid;not null"`
	UserID     uuid.UUID  `json:"user_id" gorm:"type:uuid;not null"`
	RoleID     uuid.UUID  `json:"role_id" gorm:"type:uuid;not null"`
	InvitedBy  *uuid.UUID `json:"invited_by,omitempty" gorm:"type:uuid"`
	InvitedAt  *time.Time `json:"invited_at,omitempty"`
	AcceptedAt *time.Time `json:"accepted_at,omitempty"`
	IsActive   bool       `json:"is_active" gorm:"default:true"`
	CreatedAt  time.Time  `json:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at"`

	// Relations
	Tenant  *Tenant `json:"tenant,omitempty" gorm:"foreignKey:TenantID"`
	User    *User   `json:"user,omitempty" gorm:"foreignKey:UserID"`
	Role    *Role   `json:"role,omitempty" gorm:"foreignKey:RoleID"`
	Inviter *User   `json:"inviter,omitempty" gorm:"foreignKey:InvitedBy"`
}

// TableName specifies the table name for TenantUser
func (TenantUser) TableName() string {
	return "tenant_users"
}

// ============================================================================
// Request/Response types
// ============================================================================

// CreateTenantRequest represents the request to create a new tenant
type CreateTenantRequest struct {
	Name     string          `json:"name" binding:"required,min=2,max=200"`
	Slug     string          `json:"slug" binding:"required,min=1,max=100"`
	Email    string          `json:"email,omitempty" binding:"omitempty,email"`
	Phone    string          `json:"phone,omitempty"`
	Settings *TenantSettings `json:"settings,omitempty"`
}

// UpdateTenantRequest represents the request to update a tenant
type UpdateTenantRequest struct {
	Name     *string         `json:"name,omitempty" binding:"omitempty,min=2,max=200"`
	Slug     *string         `json:"slug,omitempty" binding:"omitempty,min=1,max=100"`
	Email    *string         `json:"email,omitempty" binding:"omitempty,email"`
	Phone    *string         `json:"phone,omitempty"`
	LogoURL  *string         `json:"logo_url,omitempty"`
	Settings *TenantSettings `json:"settings,omitempty"`
	IsActive *bool           `json:"is_active,omitempty"`
}

// TenantResponse represents the API response for a tenant
type TenantResponse struct {
	ID        uuid.UUID       `json:"id"`
	Name      string          `json:"name"`
	Slug      string          `json:"slug"`
	OwnerID   uuid.UUID       `json:"owner_id"`
	Email     string          `json:"email,omitempty"`
	Phone     string          `json:"phone,omitempty"`
	LogoURL   string          `json:"logo_url,omitempty"`
	Settings  *TenantSettings `json:"settings,omitempty"`
	IsActive  bool            `json:"is_active"`
	CreatedAt time.Time       `json:"created_at"`
	UpdatedAt time.Time       `json:"updated_at"`
	Role      *string         `json:"role,omitempty"` // User's role in this tenant (when listing user's tenants)
}

// ToResponse converts a Tenant model to TenantResponse
func (t *Tenant) ToResponse() TenantResponse {
	return TenantResponse{
		ID:        t.ID,
		Name:      t.Name,
		Slug:      t.Slug,
		OwnerID:   t.OwnerID,
		Email:     t.Email,
		Phone:     t.Phone,
		LogoURL:   t.LogoURL,
		Settings:  &t.Settings,
		IsActive:  t.IsActive,
		CreatedAt: t.CreatedAt,
		UpdatedAt: t.UpdatedAt,
	}
}

// AddMemberRequest represents the request to add a member to a tenant
type AddMemberRequest struct {
	Email  string    `json:"email" binding:"required,email"`
	RoleID uuid.UUID `json:"role_id" binding:"required"`
}

// UpdateMemberRoleRequest represents the request to update a member's role
type UpdateMemberRoleRequest struct {
	RoleID uuid.UUID `json:"role_id" binding:"required"`
}

// TenantMemberResponse represents a member in the tenant
type TenantMemberResponse struct {
	ID                uuid.UUID  `json:"id"`
	UserID            uuid.UUID  `json:"user_id"`
	Email             string     `json:"email"`
	ProfilePictureURL *string    `json:"profile_picture_url,omitempty"`
	RoleID            uuid.UUID  `json:"role_id"`
	RoleName          string     `json:"role_name"`
	IsActive          bool       `json:"is_active"`
	InvitedAt         *time.Time `json:"invited_at,omitempty"`
	AcceptedAt        *time.Time `json:"accepted_at,omitempty"`
	JoinedAt          time.Time  `json:"joined_at"`
}

// ToMemberResponse converts a TenantUser to TenantMemberResponse
func (tu *TenantUser) ToMemberResponse() TenantMemberResponse {
	response := TenantMemberResponse{
		ID:         tu.ID,
		UserID:     tu.UserID,
		RoleID:     tu.RoleID,
		IsActive:   tu.IsActive,
		InvitedAt:  tu.InvitedAt,
		AcceptedAt: tu.AcceptedAt,
		JoinedAt:   tu.CreatedAt,
	}

	if tu.User != nil {
		response.Email = tu.User.Email
		response.ProfilePictureURL = tu.User.ProfilePictureURL
	}

	if tu.Role != nil {
		response.RoleName = tu.Role.Name
	}

	return response
}

// ListTenantsResponse represents the response for listing tenants
type ListTenantsResponse struct {
	Tenants    []TenantResponse `json:"tenants"`
	Total      int64            `json:"total"`
	Page       int              `json:"page"`
	PageSize   int              `json:"page_size"`
	TotalPages int              `json:"total_pages"`
}

// TenantSwitchRequest represents the request to switch active tenant
type TenantSwitchRequest struct {
	TenantID uuid.UUID `json:"tenant_id" binding:"required"`
}

// TenantSwitchResponse represents the response after switching tenant
type TenantSwitchResponse struct {
	Tenant TenantResponse `json:"tenant"`
	Token  string         `json:"token"` // New JWT with tenant context
}
