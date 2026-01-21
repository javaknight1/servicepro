package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

// InvoiceStatus represents the status of an invoice
type InvoiceStatus string

const (
	InvoiceStatusDraft         InvoiceStatus = "draft"
	InvoiceStatusSent          InvoiceStatus = "sent"
	InvoiceStatusViewed        InvoiceStatus = "viewed"
	InvoiceStatusPaid          InvoiceStatus = "paid"
	InvoiceStatusPartiallyPaid InvoiceStatus = "partially_paid"
	InvoiceStatusOverdue       InvoiceStatus = "overdue"
	InvoiceStatusCancelled     InvoiceStatus = "cancelled"
	InvoiceStatusRefunded      InvoiceStatus = "refunded"
)

// PaymentTermType represents the type of payment terms
type PaymentTermType string

const (
	PaymentTermDueOnReceipt PaymentTermType = "due_on_receipt"
	PaymentTermNet7         PaymentTermType = "net_7"
	PaymentTermNet10        PaymentTermType = "net_10"
	PaymentTermNet15        PaymentTermType = "net_15"
	PaymentTermNet30        PaymentTermType = "net_30"
	PaymentTermNet60        PaymentTermType = "net_60"
	PaymentTermNet90        PaymentTermType = "net_90"
	PaymentTermCustom       PaymentTermType = "custom"
)

// TaxType represents the type of tax
type TaxType string

const (
	TaxTypeSalesTax TaxType = "sales_tax"
	TaxTypeVAT      TaxType = "vat"
	TaxTypeGST      TaxType = "gst"
	TaxTypeHST      TaxType = "hst"
	TaxTypeExempt   TaxType = "exempt"
)

// TaxRate represents a tax rate configuration
type TaxRate struct {
	ID            uuid.UUID       `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	Name          string          `gorm:"type:varchar(100);not null" json:"name"`
	Description   string          `gorm:"type:text" json:"description,omitempty"`
	Rate          decimal.Decimal `gorm:"type:decimal(10,4);not null" json:"rate"`
	TaxType       TaxType         `gorm:"type:tax_type;not null;default:sales_tax" json:"tax_type"`
	Region        string          `gorm:"type:varchar(100)" json:"region,omitempty"`
	IsCompound    bool            `gorm:"default:false" json:"is_compound"`
	IsActive      bool            `gorm:"default:true" json:"is_active"`
	EffectiveDate *time.Time      `gorm:"type:date" json:"effective_date,omitempty"`
	ExpiryDate    *time.Time      `gorm:"type:date" json:"expiry_date,omitempty"`
	CreatedAt     time.Time       `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt     time.Time       `gorm:"autoUpdateTime" json:"updated_at"`
}

// TableName overrides the table name
func (TaxRate) TableName() string {
	return "tax_rates"
}

// PaymentTerm represents payment terms configuration
type PaymentTerm struct {
	ID                 uuid.UUID       `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	Name               string          `gorm:"type:varchar(100);not null" json:"name"`
	Description        string          `gorm:"type:text" json:"description,omitempty"`
	TermType           PaymentTermType `gorm:"type:payment_term_type;not null;default:net_30" json:"term_type"`
	DaysUntilDue       int             `gorm:"type:integer" json:"days_until_due"`
	DiscountPercentage decimal.Decimal `gorm:"type:decimal(5,2)" json:"discount_percentage,omitempty"`
	DiscountDays       int             `gorm:"type:integer" json:"discount_days,omitempty"`
	LateFeePercentage  decimal.Decimal `gorm:"type:decimal(5,2)" json:"late_fee_percentage,omitempty"`
	LateFeeAmount      decimal.Decimal `gorm:"type:decimal(10,2)" json:"late_fee_amount,omitempty"`
	GracePeriodDays    int             `gorm:"default:0" json:"grace_period_days"`
	IsDefault          bool            `gorm:"default:false" json:"is_default"`
	IsActive           bool            `gorm:"default:true" json:"is_active"`
	CreatedAt          time.Time       `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt          time.Time       `gorm:"autoUpdateTime" json:"updated_at"`
}

// TableName overrides the table name
func (PaymentTerm) TableName() string {
	return "payment_terms"
}

// Invoice represents an invoice
type Invoice struct {
	ID            uuid.UUID  `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	InvoiceNumber string     `gorm:"type:varchar(50);uniqueIndex;not null" json:"invoice_number"`
	CustomerID    uuid.UUID  `gorm:"type:uuid;not null;index" json:"customer_id"`
	JobID         *uuid.UUID `gorm:"type:uuid;index" json:"job_id,omitempty"`
	QuoteID       *uuid.UUID `gorm:"type:uuid;index" json:"quote_id,omitempty"`

	// Status and dates
	Status     InvoiceStatus `gorm:"type:invoice_status;not null;default:draft;index" json:"status"`
	IssueDate  time.Time     `gorm:"type:date;not null;default:CURRENT_DATE;index" json:"issue_date"`
	DueDate    time.Time     `gorm:"type:date;not null;index" json:"due_date"`
	SentDate   *time.Time    `gorm:"type:timestamp" json:"sent_date,omitempty"`
	ViewedDate *time.Time    `gorm:"type:timestamp" json:"viewed_date,omitempty"`
	PaidDate   *time.Time    `gorm:"type:timestamp" json:"paid_date,omitempty"`

	// Financial details
	Subtotal       decimal.Decimal `gorm:"type:decimal(12,2);not null;default:0.00" json:"subtotal"`
	TaxAmount      decimal.Decimal `gorm:"type:decimal(12,2);not null;default:0.00" json:"tax_amount"`
	DiscountAmount decimal.Decimal `gorm:"type:decimal(12,2);default:0.00" json:"discount_amount"`
	TotalAmount    decimal.Decimal `gorm:"type:decimal(12,2);not null;default:0.00" json:"total_amount"`
	AmountPaid     decimal.Decimal `gorm:"type:decimal(12,2);default:0.00" json:"amount_paid"`
	AmountDue      decimal.Decimal `gorm:"type:decimal(12,2);->" json:"amount_due"` // Computed column

	// References
	PaymentTermID *uuid.UUID `gorm:"type:uuid" json:"payment_term_id,omitempty"`
	TaxRateID     *uuid.UUID `gorm:"type:uuid" json:"tax_rate_id,omitempty"`

	// Additional information
	PONumber           string `gorm:"type:varchar(100)" json:"po_number,omitempty"`
	Notes              string `gorm:"type:text" json:"notes,omitempty"`
	TermsAndConditions string `gorm:"type:text" json:"terms_and_conditions,omitempty"`
	FooterText         string `gorm:"type:text" json:"footer_text,omitempty"`

	// Metadata
	CreatedBy uuid.UUID      `gorm:"type:uuid;not null" json:"created_by"`
	UpdatedBy *uuid.UUID     `gorm:"type:uuid" json:"updated_by,omitempty"`
	CreatedAt time.Time      `gorm:"autoCreateTime;index" json:"created_at"`
	UpdatedAt time.Time      `gorm:"autoUpdateTime" json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"deleted_at,omitempty"`

	// Relationships
	Customer    *User            `gorm:"foreignKey:CustomerID" json:"customer,omitempty"`
	PaymentTerm *PaymentTerm     `gorm:"foreignKey:PaymentTermID" json:"payment_term,omitempty"`
	TaxRate     *TaxRate         `gorm:"foreignKey:TaxRateID" json:"tax_rate,omitempty"`
	Lines       []InvoiceLine    `gorm:"foreignKey:InvoiceID" json:"lines,omitempty"`
	Payments    []InvoicePayment `gorm:"foreignKey:InvoiceID" json:"payments,omitempty"`
}

// TableName overrides the table name
func (Invoice) TableName() string {
	return "invoices"
}

// BeforeCreate hook to ensure invoice number is generated
func (i *Invoice) BeforeCreate(tx *gorm.DB) error {
	if i.ID == uuid.Nil {
		i.ID = uuid.New()
	}
	return nil
}

// InvoiceLine represents a line item in an invoice
type InvoiceLine struct {
	ID        uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	InvoiceID uuid.UUID `gorm:"type:uuid;not null;index" json:"invoice_id"`

	// Line item details
	Description   string          `gorm:"type:text;not null" json:"description"`
	Quantity      decimal.Decimal `gorm:"type:decimal(10,2);not null;default:1.00" json:"quantity"`
	UnitPrice     decimal.Decimal `gorm:"type:decimal(12,2);not null" json:"unit_price"`
	UnitOfMeasure string          `gorm:"type:varchar(50);default:each" json:"unit_of_measure"`

	// Calculations
	DiscountPercentage decimal.Decimal `gorm:"type:decimal(5,2);default:0.00" json:"discount_percentage"`
	DiscountAmount     decimal.Decimal `gorm:"type:decimal(12,2);default:0.00" json:"discount_amount"`
	Taxable            bool            `gorm:"default:true" json:"taxable"`
	TaxRate            decimal.Decimal `gorm:"type:decimal(10,4);default:0.00" json:"tax_rate"`
	TaxAmount          decimal.Decimal `gorm:"type:decimal(12,2);default:0.00" json:"tax_amount"`

	// Computed totals (calculated by database triggers)
	LineTotal        decimal.Decimal `gorm:"type:decimal(12,2);->" json:"line_total"`
	LineTotalWithTax decimal.Decimal `gorm:"type:decimal(12,2);->" json:"line_total_with_tax"`

	// References
	ProductID *uuid.UUID `gorm:"type:uuid;index" json:"product_id,omitempty"`
	ServiceID *uuid.UUID `gorm:"type:uuid;index" json:"service_id,omitempty"`

	// Ordering
	SortOrder int `gorm:"not null;default:0" json:"sort_order"`

	// Metadata
	CreatedAt time.Time `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt time.Time `gorm:"autoUpdateTime" json:"updated_at"`

	// Relationships
	Invoice *Invoice `gorm:"foreignKey:InvoiceID" json:"invoice,omitempty"`
}

// TableName overrides the table name
func (InvoiceLine) TableName() string {
	return "invoice_lines"
}

// InvoicePayment represents a payment made towards an invoice
type InvoicePayment struct {
	ID        uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	InvoiceID uuid.UUID `gorm:"type:uuid;not null;index" json:"invoice_id"`

	// Payment details
	Amount          decimal.Decimal `gorm:"type:decimal(12,2);not null" json:"amount"`
	PaymentDate     time.Time       `gorm:"type:date;not null;default:CURRENT_DATE;index" json:"payment_date"`
	PaymentMethod   string          `gorm:"type:varchar(50)" json:"payment_method,omitempty"`
	ReferenceNumber string          `gorm:"type:varchar(100)" json:"reference_number,omitempty"`
	Notes           string          `gorm:"type:text" json:"notes,omitempty"`

	// Metadata
	CreatedBy uuid.UUID `gorm:"type:uuid;not null" json:"created_by"`
	CreatedAt time.Time `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt time.Time `gorm:"autoUpdateTime" json:"updated_at"`

	// Relationships
	Invoice *Invoice `gorm:"foreignKey:InvoiceID" json:"invoice,omitempty"`
}

// TableName overrides the table name
func (InvoicePayment) TableName() string {
	return "invoice_payments"
}

// InvoiceAuditLog represents an audit log entry for invoice changes
type InvoiceAuditLog struct {
	ID        uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	InvoiceID uuid.UUID `gorm:"type:uuid;not null;index" json:"invoice_id"`

	// Audit details
	Action    string `gorm:"type:varchar(50);not null;index" json:"action"`
	FieldName string `gorm:"type:varchar(100)" json:"field_name,omitempty"`
	OldValue  string `gorm:"type:text" json:"old_value,omitempty"`
	NewValue  string `gorm:"type:text" json:"new_value,omitempty"`

	// Status transition tracking
	FromStatus *InvoiceStatus `gorm:"type:invoice_status" json:"from_status,omitempty"`
	ToStatus   *InvoiceStatus `gorm:"type:invoice_status" json:"to_status,omitempty"`

	// User and metadata
	ChangedBy     *uuid.UUID `gorm:"type:uuid" json:"changed_by,omitempty"`
	ChangedByType string     `gorm:"type:varchar(50)" json:"changed_by_type,omitempty"`
	IPAddress     string     `gorm:"type:inet" json:"ip_address,omitempty"`
	UserAgent     string     `gorm:"type:text" json:"user_agent,omitempty"`

	// Timestamp
	CreatedAt time.Time `gorm:"autoCreateTime;index" json:"created_at"`
}

// TableName overrides the table name
func (InvoiceAuditLog) TableName() string {
	return "invoice_audit_log"
}

// InvoiceSummary represents a view of invoice summary data
type InvoiceSummary struct {
	ID            uuid.UUID       `gorm:"type:uuid;primary_key" json:"id"`
	InvoiceNumber string          `json:"invoice_number"`
	CustomerID    uuid.UUID       `json:"customer_id"`
	Status        InvoiceStatus   `json:"status"`
	IssueDate     time.Time       `json:"issue_date"`
	DueDate       time.Time       `json:"due_date"`
	TotalAmount   decimal.Decimal `json:"total_amount"`
	AmountPaid    decimal.Decimal `json:"amount_paid"`
	AmountDue     decimal.Decimal `json:"amount_due"`
	LineItemCount int             `json:"line_item_count"`
	IsOverdue     bool            `json:"is_overdue"`
	DaysOverdue   int             `json:"days_overdue"`
}

// TableName specifies the view name
func (InvoiceSummary) TableName() string {
	return "invoice_summary"
}

// RevenueByMonth represents monthly revenue statistics
type RevenueByMonth struct {
	Month            time.Time       `gorm:"type:timestamp" json:"month"`
	InvoiceCount     int             `json:"invoice_count"`
	TotalRevenue     decimal.Decimal `json:"total_revenue"`
	TotalPaid        decimal.Decimal `json:"total_paid"`
	TotalOutstanding decimal.Decimal `json:"total_outstanding"`
}

// TableName specifies the view name
func (RevenueByMonth) TableName() string {
	return "revenue_by_month"
}

// InvoiceFilter represents filter criteria for listing invoices
type InvoiceFilter struct {
	CustomerID *uuid.UUID
	Status     *InvoiceStatus
	FromDate   *time.Time
	ToDate     *time.Time
	MinAmount  *decimal.Decimal
	MaxAmount  *decimal.Decimal
	IsOverdue  *bool
	Search     string // Search in invoice number, customer name, or notes
	Page       int
	PageSize   int
	SortBy     string // invoice_number, issue_date, due_date, total_amount, etc.
	SortOrder  string // asc, desc
}

// InvoiceListResponse represents the response for listing invoices
type InvoiceListResponse struct {
	Invoices   []Invoice `json:"invoices"`
	Total      int64     `json:"total"`
	Page       int       `json:"page"`
	PageSize   int       `json:"page_size"`
	TotalPages int       `json:"total_pages"`
}

// InvoiceValidationError represents validation errors
type InvoiceValidationError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

// InvoiceValidationResult represents the result of invoice validation
type InvoiceValidationResult struct {
	IsValid bool                     `json:"is_valid"`
	Errors  []InvoiceValidationError `json:"errors,omitempty"`
}
