package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// QuoteTemplate represents a reusable quote template
type QuoteTemplate struct {
	ID          uuid.UUID `json:"id" gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	Name        string    `json:"name" gorm:"type:varchar(200);not null"`
	Description string    `json:"description" gorm:"type:text"`
	Category    string    `json:"category" gorm:"type:varchar(100);index"`

	// Template content
	Content   string             `json:"content" gorm:"type:text;not null"`
	Variables []TemplateVariable `json:"variables" gorm:"type:jsonb"`
	LineItems []TemplateLineItem `json:"line_items" gorm:"type:jsonb"`

	// Default settings
	ValidDays      int             `json:"valid_days" gorm:"default:30"`
	PaymentTerms   string          `json:"payment_terms" gorm:"type:varchar(500)"`
	DeliveryInfo   string          `json:"delivery_info" gorm:"type:varchar(500)"`
	WarrantyInfo   string          `json:"warranty_info" gorm:"type:varchar(500)"`
	DefaultTaxRate decimal.Decimal `json:"default_tax_rate" gorm:"type:decimal(10,4)"`

	// Metadata
	Tags       []string `json:"tags" gorm:"type:text[]"`
	Version    int      `json:"version" gorm:"default:1"`
	IsActive   bool     `json:"is_active" gorm:"default:true"`
	IsPublic   bool     `json:"is_public" gorm:"default:false"`
	UsageCount int      `json:"usage_count" gorm:"default:0"`

	// Ownership
	CreatedBy uuid.UUID `json:"created_by" gorm:"type:uuid;not null"`
	UpdatedBy uuid.UUID `json:"updated_by" gorm:"type:uuid"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// TableName specifies the table name
func (QuoteTemplate) TableName() string {
	return "quote_templates"
}

// TemplateVariable represents a variable that can be substituted in the template
type TemplateVariable struct {
	Name         string              `json:"name"`        // e.g., "customer_name", "project_name"
	Type         string              `json:"type"`        // "text", "number", "date", "currency"
	Label        string              `json:"label"`       // Display label
	Description  string              `json:"description"` // Help text
	DefaultValue interface{}         `json:"default_value,omitempty"`
	Required     bool                `json:"required"`
	Placeholder  string              `json:"placeholder,omitempty"`
	Validation   *VariableValidation `json:"validation,omitempty"`
}

// VariableValidation defines validation rules for template variables
type VariableValidation struct {
	MinLength *int     `json:"min_length,omitempty"`
	MaxLength *int     `json:"max_length,omitempty"`
	MinValue  *float64 `json:"min_value,omitempty"`
	MaxValue  *float64 `json:"max_value,omitempty"`
	Pattern   *string  `json:"pattern,omitempty"` // Regex pattern
	Options   []string `json:"options,omitempty"` // For select/dropdown
}

// TemplateLineItem represents a line item template
type TemplateLineItem struct {
	Description string          `json:"description"`
	Quantity    decimal.Decimal `json:"quantity"`
	UnitPrice   decimal.Decimal `json:"unit_price"`
	IsVariable  bool            `json:"is_variable"` // If true, values can be overridden
	Category    string          `json:"category,omitempty"`
}

// TemplateCategory represents a template category
type TemplateCategory struct {
	ID          uuid.UUID `json:"id" gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	Name        string    `json:"name" gorm:"type:varchar(100);not null;unique"`
	Description string    `json:"description" gorm:"type:text"`
	Icon        string    `json:"icon" gorm:"type:varchar(50)"`  // Icon name for UI
	Color       string    `json:"color" gorm:"type:varchar(20)"` // Color code
	SortOrder   int       `json:"sort_order" gorm:"default:0"`
	IsActive    bool      `json:"is_active" gorm:"default:true"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// TableName specifies the table name
func (TemplateCategory) TableName() string {
	return "template_categories"
}

// TemplateImportExport represents the structure for importing/exporting templates
type TemplateImportExport struct {
	Version    string                   `json:"version"`
	ExportedAt time.Time                `json:"exported_at"`
	Templates  []QuoteTemplateExport    `json:"templates"`
	Categories []TemplateCategoryExport `json:"categories,omitempty"`
}

// QuoteTemplateExport represents a template for export (without IDs and timestamps)
type QuoteTemplateExport struct {
	Name           string             `json:"name"`
	Description    string             `json:"description"`
	Category       string             `json:"category"`
	Content        string             `json:"content"`
	Variables      []TemplateVariable `json:"variables"`
	LineItems      []TemplateLineItem `json:"line_items"`
	ValidDays      int                `json:"valid_days"`
	PaymentTerms   string             `json:"payment_terms"`
	DeliveryInfo   string             `json:"delivery_info"`
	WarrantyInfo   string             `json:"warranty_info"`
	DefaultTaxRate decimal.Decimal    `json:"default_tax_rate"`
	Tags           []string           `json:"tags"`
	IsPublic       bool               `json:"is_public"`
}

// TemplateCategoryExport represents a category for export
type TemplateCategoryExport struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Icon        string `json:"icon"`
	Color       string `json:"color"`
	SortOrder   int    `json:"sort_order"`
}

// TemplateVariableMap represents resolved variables for template rendering
type TemplateVariableMap map[string]interface{}

// TemplateRenderRequest represents a request to render a template
type TemplateRenderRequest struct {
	TemplateID uuid.UUID           `json:"template_id"`
	Variables  TemplateVariableMap `json:"variables"`
	CustomerID *uuid.UUID          `json:"customer_id,omitempty"`
}

// TemplateRenderResult represents the result of template rendering
type TemplateRenderResult struct {
	Content      string              `json:"content"`
	LineItems    []QuoteItem         `json:"line_items"`
	PaymentTerms string              `json:"payment_terms"`
	DeliveryInfo string              `json:"delivery_info"`
	WarrantyInfo string              `json:"warranty_info"`
	TaxRate      decimal.Decimal     `json:"tax_rate"`
	ValidUntil   time.Time           `json:"valid_until"`
	Variables    TemplateVariableMap `json:"variables"`
}

// TemplateListFilter represents filters for listing templates
type TemplateListFilter struct {
	Category  string
	Tags      []string
	Search    string
	IsActive  *bool
	IsPublic  *bool
	CreatedBy *uuid.UUID
	Page      int
	PageSize  int
	SortBy    string
	SortOrder string
}

// TemplateStats represents statistics for templates
type TemplateStats struct {
	TotalTemplates  int             `json:"total_templates"`
	ActiveTemplates int             `json:"active_templates"`
	TotalCategories int             `json:"total_categories"`
	TotalUsage      int             `json:"total_usage"`
	ByCategory      map[string]int  `json:"by_category"`
	MostUsed        []QuoteTemplate `json:"most_used"`
	RecentlyUpdated []QuoteTemplate `json:"recently_updated"`
}
