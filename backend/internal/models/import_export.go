package models

import (
	"time"

	"github.com/google/uuid"
)

// ImportJobStatus represents the status of an import job
type ImportJobStatus string

const (
	ImportJobStatusPending    ImportJobStatus = "pending"
	ImportJobStatusProcessing ImportJobStatus = "processing"
	ImportJobStatusCompleted  ImportJobStatus = "completed"
	ImportJobStatusFailed     ImportJobStatus = "failed"
	ImportJobStatusCancelled  ImportJobStatus = "cancelled"
)

// ImportJob represents an import job
type ImportJob struct {
	ID             uuid.UUID       `json:"id" gorm:"type:uuid;primary_key;default:uuid_generate_v4()"`
	TenantID       uuid.UUID       `json:"tenant_id" gorm:"type:uuid;not null;index"` // set from the caller's tenant (internal/tenancy)
	UserID         uuid.UUID       `json:"user_id" gorm:"type:uuid;not null"`
	Filename       string          `json:"filename" gorm:"type:varchar(255);not null"`
	Status         ImportJobStatus `json:"status" gorm:"type:varchar(20);not null;default:'pending'"`
	TotalRows      int             `json:"total_rows" gorm:"not null;default:0"`
	ProcessedRows  int             `json:"processed_rows" gorm:"not null;default:0"`
	SuccessfulRows int             `json:"successful_rows" gorm:"not null;default:0"`
	FailedRows     int             `json:"failed_rows" gorm:"not null;default:0"`
	// SkippedRows: rows not imported on purpose (already imported, or an
	// invoice already paid). Counted in ProcessedRows, not in Successful.
	SkippedRows  int     `json:"skipped_rows" gorm:"not null;default:0"`
	ErrorSummary *string `json:"error_summary" gorm:"type:text"`
	// Source and Entity say what a mapped import (T193) reads: source
	// generic/jobber/housecall_pro (servicepro = the customer template) and
	// entity customers/sites/jobs/invoices.
	Source string `json:"source" gorm:"type:varchar(30);not null;default:'servicepro'"`
	Entity string `json:"entity" gorm:"type:varchar(20);not null;default:'customers'"`
	// Attempts counts interrupted runs of a resumable job (T193).
	Attempts int `json:"-" gorm:"not null;default:0"`
	// Payload holds the parsed rows and options of a pending/processing job
	// (JSON); it is cleared when the job finishes. Never sent to clients.
	Payload     *string    `json:"-" gorm:"type:jsonb"`
	StartedAt   *time.Time `json:"started_at"`
	CompletedAt *time.Time `json:"completed_at"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

// TableName specifies the table name for ImportJob model
func (ImportJob) TableName() string {
	return "import_jobs"
}

// ImportError represents an error that occurred during import
type ImportError struct {
	ID        uuid.UUID `json:"id" gorm:"type:uuid;primary_key;default:uuid_generate_v4()"`
	JobID     uuid.UUID `json:"job_id" gorm:"type:uuid;not null"`
	RowNumber int       `json:"row_number" gorm:"not null"`
	Field     string    `json:"field" gorm:"type:varchar(100)"`
	Error     string    `json:"error" gorm:"type:text;not null"`
	RowData   string    `json:"row_data" gorm:"type:text"` // JSON representation of the row
	CreatedAt time.Time `json:"created_at"`
}

// TableName specifies the table name for ImportError model
func (ImportError) TableName() string {
	return "import_errors"
}

// ImportRequest represents the request payload for importing customers
type ImportRequest struct {
	Format            string `form:"format" binding:"required,oneof=csv json"`
	DuplicateHandling string `form:"duplicate_handling" binding:"omitempty,oneof=skip update error"`
	DryRun            bool   `form:"dry_run"`
}

// ImportResponse represents the response for an import request
type ImportResponse struct {
	JobID     uuid.UUID       `json:"job_id"`
	Status    ImportJobStatus `json:"status"`
	Message   string          `json:"message"`
	TotalRows int             `json:"total_rows,omitempty"`
}

// ImportStatusResponse represents the status of an import job
type ImportStatusResponse struct {
	JobID          uuid.UUID             `json:"job_id"`
	Status         ImportJobStatus       `json:"status"`
	TotalRows      int                   `json:"total_rows"`
	ProcessedRows  int                   `json:"processed_rows"`
	SuccessfulRows int                   `json:"successful_rows"`
	FailedRows     int                   `json:"failed_rows"`
	Progress       float64               `json:"progress"` // Percentage 0-100
	ErrorSummary   *string               `json:"error_summary,omitempty"`
	Errors         []ImportErrorResponse `json:"errors,omitempty"`
	StartedAt      *time.Time            `json:"started_at,omitempty"`
	CompletedAt    *time.Time            `json:"completed_at,omitempty"`
	CreatedAt      time.Time             `json:"created_at"`
}

// ImportExternalRef records which tenant row an imported external record
// became (T193), so re-importing a file finds it instead of duplicating it.
type ImportExternalRef struct {
	ID          uuid.UUID  `gorm:"type:uuid;primary_key;default:uuid_generate_v4()"`
	TenantID    uuid.UUID  `gorm:"type:uuid;not null"` // set from the caller's tenant (internal/tenancy)
	Source      string     `gorm:"type:varchar(30);not null"`
	Entity      string     `gorm:"type:varchar(20);not null"`
	ExternalID  string     `gorm:"type:varchar(255);not null"`
	RecordID    uuid.UUID  `gorm:"type:uuid;not null"`
	ImportJobID *uuid.UUID `gorm:"type:uuid"`
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// TableName specifies the table name for ImportExternalRef.
func (ImportExternalRef) TableName() string { return "import_external_refs" }

// ImportErrorResponse represents an error in the import process
type ImportErrorResponse struct {
	RowNumber int    `json:"row_number"`
	Field     string `json:"field,omitempty"`
	Error     string `json:"error"`
	RowData   string `json:"row_data,omitempty"`
}

// ExportFormat represents the export format
type ExportFormat string

const (
	ExportFormatCSV   ExportFormat = "csv"
	ExportFormatJSON  ExportFormat = "json"
	ExportFormatExcel ExportFormat = "xlsx"
	ExportFormatPDF   ExportFormat = "pdf"
)

// CSVImportRow represents a row from the CSV import file
type CSVImportRow struct {
	FirstName            string `csv:"first_name"`
	LastName             string `csv:"last_name"`
	CompanyName          string `csv:"company_name"`
	Email                string `csv:"email"`
	PhonePrimary         string `csv:"phone_primary"`
	PhoneSecondary       string `csv:"phone_secondary"`
	BillingAddressStreet string `csv:"billing_address_street"`
	BillingAddressCity   string `csv:"billing_address_city"`
	BillingAddressState  string `csv:"billing_address_state"`
	BillingAddressZip    string `csv:"billing_address_zip"`
	ServiceAddressStreet string `csv:"service_address_street"`
	ServiceAddressCity   string `csv:"service_address_city"`
	ServiceAddressState  string `csv:"service_address_state"`
	ServiceAddressZip    string `csv:"service_address_zip"`
	CustomerType         string `csv:"customer_type"`
	Status               string `csv:"status"`
	Notes                string `csv:"notes"`
}

// Validate validates the CSV import row
func (r *CSVImportRow) Validate() []string {
	var errors []string

	if r.FirstName == "" {
		errors = append(errors, "first_name is required")
	}
	if r.LastName == "" {
		errors = append(errors, "last_name is required")
	}
	if r.Email == "" {
		errors = append(errors, "email is required")
	}
	if r.PhonePrimary == "" {
		errors = append(errors, "phone_primary is required")
	}
	if r.BillingAddressStreet == "" {
		errors = append(errors, "billing_address_street is required")
	}
	if r.BillingAddressCity == "" {
		errors = append(errors, "billing_address_city is required")
	}
	if r.BillingAddressState == "" {
		errors = append(errors, "billing_address_state is required")
	} else if len(r.BillingAddressState) != 2 {
		errors = append(errors, "billing_address_state must be 2 characters")
	}
	if r.BillingAddressZip == "" {
		errors = append(errors, "billing_address_zip is required")
	}
	if r.CustomerType == "" {
		errors = append(errors, "customer_type is required")
	} else if r.CustomerType != "residential" && r.CustomerType != "commercial" {
		errors = append(errors, "customer_type must be 'residential' or 'commercial'")
	}

	return errors
}

// ToCustomer converts a CSVImportRow to a Customer model
func (r *CSVImportRow) ToCustomer() *Customer {
	customer := &Customer{
		FirstName:            r.FirstName,
		LastName:             r.LastName,
		Email:                r.Email,
		PhonePrimary:         r.PhonePrimary,
		BillingAddressStreet: r.BillingAddressStreet,
		BillingAddressCity:   r.BillingAddressCity,
		BillingAddressState:  r.BillingAddressState,
		BillingAddressZip:    r.BillingAddressZip,
		CustomerType:         CustomerType(r.CustomerType),
	}

	// Optional fields
	if r.CompanyName != "" {
		customer.CompanyName = &r.CompanyName
	}
	if r.PhoneSecondary != "" {
		customer.PhoneSecondary = &r.PhoneSecondary
	}
	if r.ServiceAddressStreet != "" {
		customer.ServiceAddressStreet = &r.ServiceAddressStreet
	}
	if r.ServiceAddressCity != "" {
		customer.ServiceAddressCity = &r.ServiceAddressCity
	}
	if r.ServiceAddressState != "" {
		customer.ServiceAddressState = &r.ServiceAddressState
	}
	if r.ServiceAddressZip != "" {
		customer.ServiceAddressZip = &r.ServiceAddressZip
	}
	if r.Status != "" {
		customer.Status = CustomerStatus(r.Status)
	} else {
		customer.Status = CustomerStatusProspect
	}
	if r.Notes != "" {
		customer.Notes = &r.Notes
	}

	return customer
}
