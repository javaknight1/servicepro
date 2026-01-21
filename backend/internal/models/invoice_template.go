package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

// TemplateStatus represents the status of an invoice template
type TemplateStatus string

const (
	TemplateStatusDraft      TemplateStatus = "draft"
	TemplateStatusActive     TemplateStatus = "active"
	TemplateStatusArchived   TemplateStatus = "archived"
	TemplateStatusDeprecated TemplateStatus = "deprecated"
)

// PageSize represents standard page sizes
type PageSize string

const (
	PageSizeA4     PageSize = "A4"
	PageSizeLetter PageSize = "Letter"
	PageSizeLegal  PageSize = "Legal"
	PageSizeA5     PageSize = "A5"
)

// PageOrientation represents page orientation
type PageOrientation string

const (
	PageOrientationPortrait  PageOrientation = "portrait"
	PageOrientationLandscape PageOrientation = "landscape"
)

// InvoiceTemplate represents an invoice template
type InvoiceTemplate struct {
	ID          uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	Name        string    `gorm:"type:varchar(200);not null" json:"name"`
	Description string    `gorm:"type:text" json:"description,omitempty"`

	// Template content
	HTMLContent string `gorm:"type:text;not null" json:"html_content"`
	CSSContent  string `gorm:"type:text" json:"css_content,omitempty"`
	HeaderHTML  string `gorm:"type:text" json:"header_html,omitempty"`
	FooterHTML  string `gorm:"type:text" json:"footer_html,omitempty"`

	// Configuration
	Status    TemplateStatus `gorm:"type:template_status;not null;default:draft" json:"status"`
	IsDefault bool           `gorm:"default:false" json:"is_default"`
	Version   int            `gorm:"default:1" json:"version"`

	// Layout settings
	PageSize        PageSize        `gorm:"type:varchar(20);default:A4" json:"page_size"`
	PageOrientation PageOrientation `gorm:"type:varchar(20);default:portrait" json:"page_orientation"`
	MarginTop       decimal.Decimal `gorm:"type:decimal(5,2);default:10.0" json:"margin_top"`
	MarginRight     decimal.Decimal `gorm:"type:decimal(5,2);default:10.0" json:"margin_right"`
	MarginBottom    decimal.Decimal `gorm:"type:decimal(5,2);default:10.0" json:"margin_bottom"`
	MarginLeft      decimal.Decimal `gorm:"type:decimal(5,2);default:10.0" json:"margin_left"`

	// Branding
	LogoURL      string          `gorm:"type:text" json:"logo_url,omitempty"`
	LogoPosition string          `gorm:"type:varchar(50);default:top-left" json:"logo_position,omitempty"`
	LogoWidth    decimal.Decimal `gorm:"type:decimal(5,2)" json:"logo_width,omitempty"`
	LogoHeight   decimal.Decimal `gorm:"type:decimal(5,2)" json:"logo_height,omitempty"`

	// Watermark settings
	WatermarkText     string          `gorm:"type:varchar(200)" json:"watermark_text,omitempty"`
	WatermarkOpacity  decimal.Decimal `gorm:"type:decimal(3,2);default:0.1" json:"watermark_opacity"`
	WatermarkRotation int             `gorm:"default:45" json:"watermark_rotation"`
	WatermarkEnabled  bool            `gorm:"default:false" json:"watermark_enabled"`

	// Page numbers
	ShowPageNumbers    bool   `gorm:"default:true" json:"show_page_numbers"`
	PageNumberFormat   string `gorm:"type:varchar(100);default:'Page {page} of {total}'" json:"page_number_format"`
	PageNumberPosition string `gorm:"type:varchar(50);default:bottom-center" json:"page_number_position"`

	// Custom fonts
	CustomFonts map[string]interface{} `gorm:"type:jsonb" json:"custom_fonts,omitempty"`

	// Dynamic fields mapping
	FieldMappings map[string]interface{} `gorm:"type:jsonb" json:"field_mappings,omitempty"`

	// Preview data
	PreviewData map[string]interface{} `gorm:"type:jsonb" json:"preview_data,omitempty"`

	// Version control
	ParentTemplateID *uuid.UUID `gorm:"type:uuid" json:"parent_template_id,omitempty"`
	VersionNotes     string     `gorm:"type:text" json:"version_notes,omitempty"`

	// Metadata
	Tags      []string   `gorm:"type:text[]" json:"tags,omitempty"`
	CreatedBy uuid.UUID  `gorm:"type:uuid;not null" json:"created_by"`
	UpdatedBy *uuid.UUID `gorm:"type:uuid" json:"updated_by,omitempty"`
	CreatedAt time.Time  `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt time.Time  `gorm:"autoUpdateTime" json:"updated_at"`

	// Relationships
	Assets         []InvoiceTemplateAsset   `gorm:"foreignKey:TemplateID" json:"assets,omitempty"`
	UsageRecords   []InvoiceTemplateUsage   `gorm:"foreignKey:TemplateID" json:"-"`
	VersionHistory []InvoiceTemplateVersion `gorm:"foreignKey:TemplateID" json:"-"`
}

// TableName overrides the table name
func (InvoiceTemplate) TableName() string {
	return "invoice_templates"
}

// InvoiceTemplateAsset represents an asset (logo, image, font) associated with a template
type InvoiceTemplateAsset struct {
	ID         uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	TemplateID uuid.UUID `gorm:"type:uuid;not null" json:"template_id"`

	// Asset information
	AssetType string `gorm:"type:varchar(50);not null" json:"asset_type"`
	AssetName string `gorm:"type:varchar(200);not null" json:"asset_name"`
	FilePath  string `gorm:"type:text;not null" json:"file_path"`
	FileSize  int64  `json:"file_size,omitempty"`
	MimeType  string `gorm:"type:varchar(100)" json:"mime_type,omitempty"`

	// Image-specific fields
	Width  int `json:"width,omitempty"`
	Height int `json:"height,omitempty"`

	// Metadata
	Description string    `gorm:"type:text" json:"description,omitempty"`
	CreatedAt   time.Time `gorm:"autoCreateTime" json:"created_at"`
}

// TableName overrides the table name
func (InvoiceTemplateAsset) TableName() string {
	return "invoice_template_assets"
}

// InvoiceTemplateUsage tracks template usage for invoices
type InvoiceTemplateUsage struct {
	ID         uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	TemplateID uuid.UUID `gorm:"type:uuid;not null" json:"template_id"`
	InvoiceID  uuid.UUID `gorm:"type:uuid;not null" json:"invoice_id"`

	// Generation details
	GeneratedAt      time.Time  `gorm:"default:NOW()" json:"generated_at"`
	GeneratedBy      *uuid.UUID `gorm:"type:uuid" json:"generated_by,omitempty"`
	PDFFilePath      string     `gorm:"type:text" json:"pdf_file_path,omitempty"`
	PDFFileSize      int64      `json:"pdf_file_size,omitempty"`
	GenerationTimeMs int        `json:"generation_time_ms,omitempty"`

	// Success/Error tracking
	Success      bool   `gorm:"default:true" json:"success"`
	ErrorMessage string `gorm:"type:text" json:"error_message,omitempty"`
}

// TableName overrides the table name
func (InvoiceTemplateUsage) TableName() string {
	return "invoice_template_usage"
}

// InvoiceTemplateVersion represents a snapshot of a template at a specific version
type InvoiceTemplateVersion struct {
	ID            uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	TemplateID    uuid.UUID `gorm:"type:uuid;not null" json:"template_id"`
	VersionNumber int       `gorm:"not null" json:"version_number"`

	// Snapshot of template at this version
	HTMLContent   string                 `gorm:"type:text;not null" json:"html_content"`
	CSSContent    string                 `gorm:"type:text" json:"css_content,omitempty"`
	HeaderHTML    string                 `gorm:"type:text" json:"header_html,omitempty"`
	FooterHTML    string                 `gorm:"type:text" json:"footer_html,omitempty"`
	Configuration map[string]interface{} `gorm:"type:jsonb" json:"configuration,omitempty"`

	// Version metadata
	VersionNotes string    `gorm:"type:text" json:"version_notes,omitempty"`
	CreatedBy    uuid.UUID `gorm:"type:uuid;not null" json:"created_by"`
	CreatedAt    time.Time `gorm:"autoCreateTime" json:"created_at"`
}

// TableName overrides the table name
func (InvoiceTemplateVersion) TableName() string {
	return "invoice_template_versions"
}

// TemplateStatistics represents usage statistics for a template
type TemplateStatistics struct {
	TotalUsage            int64           `json:"total_usage"`
	SuccessfulGenerations int64           `json:"successful_generations"`
	FailedGenerations     int64           `json:"failed_generations"`
	AvgGenerationTimeMs   decimal.Decimal `json:"avg_generation_time_ms"`
	LastUsed              *time.Time      `json:"last_used,omitempty"`
	TotalPDFSizeMB        decimal.Decimal `json:"total_pdf_size_mb"`
}

// CreateTemplateRequest represents a request to create a new template
type CreateTemplateRequest struct {
	Name        string         `json:"name" binding:"required"`
	Description string         `json:"description"`
	HTMLContent string         `json:"html_content" binding:"required"`
	CSSContent  string         `json:"css_content"`
	HeaderHTML  string         `json:"header_html"`
	FooterHTML  string         `json:"footer_html"`
	Status      TemplateStatus `json:"status"`
	IsDefault   bool           `json:"is_default"`

	// Layout settings
	PageSize        PageSize        `json:"page_size"`
	PageOrientation PageOrientation `json:"page_orientation"`
	MarginTop       decimal.Decimal `json:"margin_top"`
	MarginRight     decimal.Decimal `json:"margin_right"`
	MarginBottom    decimal.Decimal `json:"margin_bottom"`
	MarginLeft      decimal.Decimal `json:"margin_left"`

	// Branding
	LogoURL      string          `json:"logo_url"`
	LogoPosition string          `json:"logo_position"`
	LogoWidth    decimal.Decimal `json:"logo_width"`
	LogoHeight   decimal.Decimal `json:"logo_height"`

	// Watermark
	WatermarkText     string          `json:"watermark_text"`
	WatermarkOpacity  decimal.Decimal `json:"watermark_opacity"`
	WatermarkRotation int             `json:"watermark_rotation"`
	WatermarkEnabled  bool            `json:"watermark_enabled"`

	// Page numbers
	ShowPageNumbers    bool   `json:"show_page_numbers"`
	PageNumberFormat   string `json:"page_number_format"`
	PageNumberPosition string `json:"page_number_position"`

	// Custom settings
	CustomFonts   map[string]interface{} `json:"custom_fonts"`
	FieldMappings map[string]interface{} `json:"field_mappings"`
	PreviewData   map[string]interface{} `json:"preview_data"`

	Tags []string `json:"tags"`
}

// UpdateTemplateRequest represents a request to update a template
type UpdateTemplateRequest struct {
	Name        *string         `json:"name"`
	Description *string         `json:"description"`
	HTMLContent *string         `json:"html_content"`
	CSSContent  *string         `json:"css_content"`
	HeaderHTML  *string         `json:"header_html"`
	FooterHTML  *string         `json:"footer_html"`
	Status      *TemplateStatus `json:"status"`
	IsDefault   *bool           `json:"is_default"`

	// Layout settings
	PageSize        *PageSize        `json:"page_size"`
	PageOrientation *PageOrientation `json:"page_orientation"`
	MarginTop       *decimal.Decimal `json:"margin_top"`
	MarginRight     *decimal.Decimal `json:"margin_right"`
	MarginBottom    *decimal.Decimal `json:"margin_bottom"`
	MarginLeft      *decimal.Decimal `json:"margin_left"`

	// Branding
	LogoURL      *string          `json:"logo_url"`
	LogoPosition *string          `json:"logo_position"`
	LogoWidth    *decimal.Decimal `json:"logo_width"`
	LogoHeight   *decimal.Decimal `json:"logo_height"`

	// Watermark
	WatermarkText     *string          `json:"watermark_text"`
	WatermarkOpacity  *decimal.Decimal `json:"watermark_opacity"`
	WatermarkRotation *int             `json:"watermark_rotation"`
	WatermarkEnabled  *bool            `json:"watermark_enabled"`

	// Page numbers
	ShowPageNumbers    *bool   `json:"show_page_numbers"`
	PageNumberFormat   *string `json:"page_number_format"`
	PageNumberPosition *string `json:"page_number_position"`

	// Custom settings
	CustomFonts   *map[string]interface{} `json:"custom_fonts"`
	FieldMappings *map[string]interface{} `json:"field_mappings"`
	PreviewData   *map[string]interface{} `json:"preview_data"`

	Tags         *[]string `json:"tags"`
	VersionNotes *string   `json:"version_notes"`
}

// GeneratePDFRequest represents a request to generate a PDF from a template
type GeneratePDFRequest struct {
	TemplateID uuid.UUID              `json:"template_id" binding:"required"`
	InvoiceID  *uuid.UUID             `json:"invoice_id"`
	Data       map[string]interface{} `json:"data" binding:"required"`
	OutputPath *string                `json:"output_path"`
}

// GeneratePDFResponse represents the response after PDF generation
type GeneratePDFResponse struct {
	Success          bool      `json:"success"`
	FilePath         string    `json:"file_path,omitempty"`
	FileSize         int64     `json:"file_size,omitempty"`
	GenerationTimeMs int       `json:"generation_time_ms"`
	Error            string    `json:"error,omitempty"`
	GeneratedAt      time.Time `json:"generated_at"`
}

// TemplatePreviewRequest represents a request to preview a template
type TemplatePreviewRequest struct {
	HTMLContent string                 `json:"html_content" binding:"required"`
	CSSContent  string                 `json:"css_content"`
	Data        map[string]interface{} `json:"data"`
}

// BeforeCreate hook for InvoiceTemplate
func (t *InvoiceTemplate) BeforeCreate(tx *gorm.DB) error {
	if t.ID == uuid.Nil {
		t.ID = uuid.New()
	}
	return nil
}

// BeforeCreate hook for InvoiceTemplateAsset
func (a *InvoiceTemplateAsset) BeforeCreate(tx *gorm.DB) error {
	if a.ID == uuid.Nil {
		a.ID = uuid.New()
	}
	return nil
}

// BeforeCreate hook for InvoiceTemplateUsage
func (u *InvoiceTemplateUsage) BeforeCreate(tx *gorm.DB) error {
	if u.ID == uuid.Nil {
		u.ID = uuid.New()
	}
	return nil
}

// BeforeCreate hook for InvoiceTemplateVersion
func (v *InvoiceTemplateVersion) BeforeCreate(tx *gorm.DB) error {
	if v.ID == uuid.Nil {
		v.ID = uuid.New()
	}
	return nil
}
