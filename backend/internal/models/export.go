package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// ExportJobStatus represents the status of an export job
type ExportJobStatus string

const (
	ExportJobStatusPending    ExportJobStatus = "pending"
	ExportJobStatusProcessing ExportJobStatus = "processing"
	ExportJobStatusCompleted  ExportJobStatus = "completed"
	ExportJobStatusFailed     ExportJobStatus = "failed"
	ExportJobStatusCancelled  ExportJobStatus = "cancelled"
)

// ExportType represents the type of data being exported
type ExportType string

const (
	ExportTypeCustomers      ExportType = "customers"
	ExportTypeJobs           ExportType = "jobs"
	ExportTypeInvoices       ExportType = "invoices"
	ExportTypeQuotes         ExportType = "quotes"
	ExportTypeRevenueReport  ExportType = "revenue_report"
	ExportTypeCustomerReport ExportType = "customer_report"
)

// ExportJob represents an export job with progress tracking
type ExportJob struct {
	ID         uuid.UUID       `json:"id" gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	UserID     uuid.UUID       `json:"user_id" gorm:"type:uuid;not null"`
	ExportType ExportType      `json:"export_type" gorm:"type:varchar(50);not null"`
	Format     ExportFormat    `json:"format" gorm:"type:varchar(20);not null"`
	Status     ExportJobStatus `json:"status" gorm:"type:varchar(20);not null;default:'pending'"`

	// Progress tracking
	TotalRecords     int             `json:"total_records" gorm:"not null;default:0"`
	ProcessedRecords int             `json:"processed_records" gorm:"not null;default:0"`
	Progress         decimal.Decimal `json:"progress" gorm:"type:decimal(5,2);default:0"` // 0-100

	// File info
	Filename    string `json:"filename" gorm:"type:varchar(255)"`
	FilePath    string `json:"file_path" gorm:"type:varchar(500)"`
	FileSize    int64  `json:"file_size" gorm:"default:0"`
	MimeType    string `json:"mime_type" gorm:"type:varchar(100)"`
	DownloadURL string `json:"download_url" gorm:"type:varchar(500)"`

	// Configuration
	Query           string   `json:"query" gorm:"type:jsonb"`   // JSON serialized query params
	IncludeSections []string `json:"include_sections" gorm:"-"` // Not stored in DB
	Encoding        string   `json:"encoding" gorm:"type:varchar(20);default:'utf-8'"`

	// Error handling
	ErrorMessage *string `json:"error_message" gorm:"type:text"`
	ErrorDetails *string `json:"error_details" gorm:"type:text"`

	// Timestamps
	StartedAt   *time.Time `json:"started_at"`
	CompletedAt *time.Time `json:"completed_at"`
	ExpiresAt   *time.Time `json:"expires_at"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

func (ExportJob) TableName() string {
	return "export_jobs"
}

// ExportJobResponse represents the API response for an export job
type ExportJobResponse struct {
	ID               uuid.UUID       `json:"id"`
	ExportType       ExportType      `json:"export_type"`
	Format           ExportFormat    `json:"format"`
	Status           ExportJobStatus `json:"status"`
	TotalRecords     int             `json:"total_records"`
	ProcessedRecords int             `json:"processed_records"`
	Progress         float64         `json:"progress"`
	Filename         string          `json:"filename,omitempty"`
	FileSize         int64           `json:"file_size,omitempty"`
	DownloadURL      string          `json:"download_url,omitempty"`
	ErrorMessage     *string         `json:"error_message,omitempty"`
	StartedAt        *time.Time      `json:"started_at,omitempty"`
	CompletedAt      *time.Time      `json:"completed_at,omitempty"`
	ExpiresAt        *time.Time      `json:"expires_at,omitempty"`
	CreatedAt        time.Time       `json:"created_at"`
}

// ToResponse converts ExportJob to ExportJobResponse
func (j *ExportJob) ToResponse() ExportJobResponse {
	return ExportJobResponse{
		ID:               j.ID,
		ExportType:       j.ExportType,
		Format:           j.Format,
		Status:           j.Status,
		TotalRecords:     j.TotalRecords,
		ProcessedRecords: j.ProcessedRecords,
		Progress:         j.Progress.InexactFloat64(),
		Filename:         j.Filename,
		FileSize:         j.FileSize,
		DownloadURL:      j.DownloadURL,
		ErrorMessage:     j.ErrorMessage,
		StartedAt:        j.StartedAt,
		CompletedAt:      j.CompletedAt,
		ExpiresAt:        j.ExpiresAt,
		CreatedAt:        j.CreatedAt,
	}
}

// ExportConfig represents configuration for an export operation
type ExportConfig struct {
	Format          ExportFormat `json:"format" binding:"required,oneof=csv json xlsx pdf"`
	ExportType      ExportType   `json:"export_type" binding:"required"`
	IncludeSections []string     `json:"include_sections"`
	Fields          []string     `json:"fields"`   // Specific fields to include
	Encoding        string       `json:"encoding"` // utf-8, utf-16, etc.

	// Format-specific options
	CSVOptions   *CSVExportOptions   `json:"csv_options,omitempty"`
	PDFOptions   *PDFExportOptions   `json:"pdf_options,omitempty"`
	ExcelOptions *ExcelExportOptions `json:"excel_options,omitempty"`

	// Query parameters for filtering data
	Query interface{} `json:"query,omitempty"`
}

// CSVExportOptions contains CSV-specific export options
type CSVExportOptions struct {
	Delimiter     string `json:"delimiter"` // comma, tab, semicolon
	IncludeHeader bool   `json:"include_header"`
	QuoteAll      bool   `json:"quote_all"`
	LineEnding    string `json:"line_ending"` // unix (\n) or windows (\r\n)
	BOM           bool   `json:"bom"`         // Include UTF-8 BOM for Excel compatibility
}

// PDFExportOptions contains PDF-specific export options
type PDFExportOptions struct {
	PageSize         string  `json:"page_size"`   // A4, Letter, etc.
	Orientation      string  `json:"orientation"` // portrait, landscape
	Title            string  `json:"title"`
	IncludeSummary   bool    `json:"include_summary"`
	IncludeCharts    bool    `json:"include_charts"`
	WatermarkText    string  `json:"watermark_text"`
	WatermarkOpacity float64 `json:"watermark_opacity"`
	HeaderText       string  `json:"header_text"`
	FooterText       string  `json:"footer_text"`
	ShowPageNumbers  bool    `json:"show_page_numbers"`
}

// ExcelExportOptions contains Excel-specific export options
type ExcelExportOptions struct {
	SheetName       string         `json:"sheet_name"`
	IncludeFormulas bool           `json:"include_formulas"`
	FreezePanes     bool           `json:"freeze_panes"`
	AutoFilter      bool           `json:"auto_filter"`
	ColumnWidths    map[string]int `json:"column_widths"`
}

// ExportProgress represents real-time export progress
type ExportProgress struct {
	JobID            uuid.UUID       `json:"job_id"`
	Status           ExportJobStatus `json:"status"`
	TotalRecords     int             `json:"total_records"`
	ProcessedRecords int             `json:"processed_records"`
	Progress         float64         `json:"progress"`      // 0-100
	CurrentPhase     string          `json:"current_phase"` // fetching, processing, writing, finalizing
	EstimatedTimeMs  int64           `json:"estimated_time_ms"`
	ElapsedTimeMs    int64           `json:"elapsed_time_ms"`
	Message          string          `json:"message,omitempty"`
}

// ExportResult represents the final result of an export
type ExportResult struct {
	Success      bool      `json:"success"`
	JobID        uuid.UUID `json:"job_id"`
	Filename     string    `json:"filename"`
	FilePath     string    `json:"file_path"`
	FileSize     int64     `json:"file_size"`
	MimeType     string    `json:"mime_type"`
	DownloadURL  string    `json:"download_url"`
	RecordCount  int       `json:"record_count"`
	GeneratedAt  time.Time `json:"generated_at"`
	ExpiresAt    time.Time `json:"expires_at"`
	DurationMs   int64     `json:"duration_ms"`
	ErrorMessage string    `json:"error_message,omitempty"`
}

// ExportMimeTypes maps export formats to MIME types
var ExportMimeTypes = map[ExportFormat]string{
	ExportFormatCSV:   "text/csv; charset=utf-8",
	ExportFormatJSON:  "application/json; charset=utf-8",
	ExportFormatExcel: "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
	ExportFormatPDF:   "application/pdf",
}

// ExportFileExtensions maps export formats to file extensions
var ExportFileExtensions = map[ExportFormat]string{
	ExportFormatCSV:   ".csv",
	ExportFormatJSON:  ".json",
	ExportFormatExcel: ".xlsx",
	ExportFormatPDF:   ".pdf",
}

// GetMimeType returns the MIME type for the export format
func (f ExportFormat) GetMimeType() string {
	if mime, ok := ExportMimeTypes[f]; ok {
		return mime
	}
	return "application/octet-stream"
}

// GetExtension returns the file extension for the export format
func (f ExportFormat) GetExtension() string {
	if ext, ok := ExportFileExtensions[f]; ok {
		return ext
	}
	return ".bin"
}

// DefaultCSVOptions returns default CSV export options
func DefaultCSVOptions() *CSVExportOptions {
	return &CSVExportOptions{
		Delimiter:     ",",
		IncludeHeader: true,
		QuoteAll:      false,
		LineEnding:    "\n",
		BOM:           true, // Include BOM for Excel compatibility
	}
}

// DefaultPDFOptions returns default PDF export options
func DefaultPDFOptions() *PDFExportOptions {
	return &PDFExportOptions{
		PageSize:         "A4",
		Orientation:      "portrait",
		IncludeSummary:   true,
		IncludeCharts:    false,
		WatermarkOpacity: 0.1,
		ShowPageNumbers:  true,
	}
}

// DefaultExcelOptions returns default Excel export options
func DefaultExcelOptions() *ExcelExportOptions {
	return &ExcelExportOptions{
		SheetName:       "Export",
		IncludeFormulas: false,
		FreezePanes:     true,
		AutoFilter:      true,
	}
}
