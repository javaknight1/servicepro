package models

import (
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Site is one of a customer's service locations (T186): a building or
// property the tenant does work at, with its own address, access notes and
// site contact. Table service_locations; tenant-owned (internal/tenancy).
type Site struct {
	ID            uuid.UUID      `json:"id" gorm:"type:uuid;primary_key;default:uuid_generate_v4()"`
	TenantID      uuid.UUID      `json:"tenant_id" gorm:"type:uuid;not null;index"`
	CustomerID    uuid.UUID      `json:"customer_id" gorm:"type:uuid;not null;index"`
	Name          string         `json:"name" gorm:"type:varchar(200);not null"`
	AddressStreet string         `json:"address_street" gorm:"type:varchar(100);not null;default:''"`
	AddressCity   string         `json:"address_city" gorm:"type:varchar(50);not null;default:''"`
	AddressState  string         `json:"address_state" gorm:"type:varchar(2);not null;default:''"`
	AddressZip    string         `json:"address_zip" gorm:"type:varchar(10);not null;default:''"`
	AccessNotes   *string        `json:"access_notes" gorm:"type:text"`
	ContactName   *string        `json:"contact_name" gorm:"type:varchar(100)"`
	ContactPhone  *string        `json:"contact_phone" gorm:"type:varchar(20)"`
	ContactEmail  *string        `json:"contact_email" gorm:"type:varchar(255)"`
	IsActive      bool           `json:"is_active" gorm:"not null"` // no gorm default:true: it would store false as true
	CreatedAt     time.Time      `json:"created_at"`
	UpdatedAt     time.Time      `json:"updated_at"`
	DeletedAt     gorm.DeletedAt `json:"-" gorm:"index"`
}

// TableName specifies the table name for Site.
func (Site) TableName() string { return "service_locations" }

// FullAddress is the site's address on one line ("" parts are skipped).
func (s *Site) FullAddress() string {
	var parts []string
	for _, p := range []string{s.AddressStreet, s.AddressCity, strings.TrimSpace(s.AddressState + " " + s.AddressZip)} {
		if strings.TrimSpace(p) != "" {
			parts = append(parts, p)
		}
	}
	return strings.Join(parts, ", ")
}

// ServiceAddress is the site's address in the job's embedded form.
func (s *Site) ServiceAddress() ServiceAddress {
	return ServiceAddress{Street: s.AddressStreet, City: s.AddressCity, State: s.AddressState, Zip: s.AddressZip}
}

// CreateSiteRequest is POST /customers/{id}/sites.
type CreateSiteRequest struct {
	Name          string  `json:"name" binding:"required,max=200"`
	AddressStreet string  `json:"address_street" binding:"required,max=100"`
	AddressCity   string  `json:"address_city" binding:"required,max=50"`
	AddressState  string  `json:"address_state" binding:"required,len=2"`
	AddressZip    string  `json:"address_zip" binding:"required,max=10"`
	AccessNotes   *string `json:"access_notes" binding:"omitempty,max=5000"`
	ContactName   *string `json:"contact_name" binding:"omitempty,max=100"`
	ContactPhone  *string `json:"contact_phone" binding:"omitempty,max=20"`
	ContactEmail  *string `json:"contact_email" binding:"omitempty,email,max=255"`
	// IsActive defaults to true when omitted.
	IsActive *bool `json:"is_active"`
}

// UpdateSiteRequest is PUT /customers/{id}/sites/{siteId}; omitted fields
// are left unchanged.
type UpdateSiteRequest struct {
	Name          *string `json:"name" binding:"omitempty,min=1,max=200"`
	AddressStreet *string `json:"address_street" binding:"omitempty,max=100"`
	AddressCity   *string `json:"address_city" binding:"omitempty,max=50"`
	AddressState  *string `json:"address_state" binding:"omitempty,len=2"`
	AddressZip    *string `json:"address_zip" binding:"omitempty,max=10"`
	AccessNotes   *string `json:"access_notes" binding:"omitempty,max=5000"`
	ContactName   *string `json:"contact_name" binding:"omitempty,max=100"`
	ContactPhone  *string `json:"contact_phone" binding:"omitempty,max=20"`
	ContactEmail  *string `json:"contact_email" binding:"omitempty,email,max=255"`
	IsActive      *bool   `json:"is_active"`
}

// SiteResponse is a site as the API returns it.
type SiteResponse struct {
	ID            uuid.UUID `json:"id"`
	CustomerID    uuid.UUID `json:"customer_id"`
	Name          string    `json:"name"`
	AddressStreet string    `json:"address_street"`
	AddressCity   string    `json:"address_city"`
	AddressState  string    `json:"address_state"`
	AddressZip    string    `json:"address_zip"`
	AddressFull   string    `json:"address_full"`
	AccessNotes   *string   `json:"access_notes"`
	ContactName   *string   `json:"contact_name"`
	ContactPhone  *string   `json:"contact_phone"`
	ContactEmail  *string   `json:"contact_email"`
	IsActive      bool      `json:"is_active"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

// SiteListResponse is GET /customers/{id}/sites.
type SiteListResponse struct {
	Sites []SiteResponse `json:"sites"`
	Total int            `json:"total"`
}

// ToResponse converts a Site to its API form.
func (s *Site) ToResponse() SiteResponse {
	return SiteResponse{
		ID:            s.ID,
		CustomerID:    s.CustomerID,
		Name:          s.Name,
		AddressStreet: s.AddressStreet,
		AddressCity:   s.AddressCity,
		AddressState:  s.AddressState,
		AddressZip:    s.AddressZip,
		AddressFull:   s.FullAddress(),
		AccessNotes:   s.AccessNotes,
		ContactName:   s.ContactName,
		ContactPhone:  s.ContactPhone,
		ContactEmail:  s.ContactEmail,
		IsActive:      s.IsActive,
		CreatedAt:     s.CreatedAt,
		UpdatedAt:     s.UpdatedAt,
	}
}
