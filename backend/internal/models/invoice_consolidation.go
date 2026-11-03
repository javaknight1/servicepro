package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// InvoiceBilledItem claims one billable thing for one invoice (T188, T225):
// a monthly agreement for a calendar month, one visit of a per-visit
// agreement (visits of monthly agreements are claimed too, so their jobs
// follow the invoice), or one extra-work job. Partial unique indexes
// (migration 000015) make every claim exclusive, so nothing is billed twice.
// Cancelling or deleting the invoice deletes its claims. Tenant-owned.
type InvoiceBilledItem struct {
	ID               uuid.UUID  `gorm:"type:uuid;primary_key"`
	TenantID         uuid.UUID  `gorm:"type:uuid;not null"`
	InvoiceID        uuid.UUID  `gorm:"type:uuid;not null"`
	InvoiceLineID    *uuid.UUID `gorm:"type:uuid"`
	AgreementID      *uuid.UUID `gorm:"type:uuid"`
	BillingMonth     *time.Time `gorm:"type:date"`
	AgreementVisitID *uuid.UUID `gorm:"type:uuid"`
	JobID            *uuid.UUID `gorm:"type:uuid"`
	CreatedAt        time.Time
}

// TableName specifies the table name for InvoiceBilledItem.
func (InvoiceBilledItem) TableName() string { return "invoice_billed_items" }

// ConsolidatedInvoiceRun claims (customer, period) for the scheduled monthly
// run, so it creates at most one draft per customer and period. Tenant-owned.
type ConsolidatedInvoiceRun struct {
	ID          uuid.UUID  `gorm:"type:uuid;primary_key"`
	TenantID    uuid.UUID  `gorm:"type:uuid;not null"`
	CustomerID  uuid.UUID  `gorm:"type:uuid;not null"`
	PeriodStart time.Time  `gorm:"type:date;not null"`
	PeriodEnd   time.Time  `gorm:"type:date;not null"`
	InvoiceID   *uuid.UUID `gorm:"type:uuid"`
	CreatedAt   time.Time
}

// TableName specifies the table name for ConsolidatedInvoiceRun.
func (ConsolidatedInvoiceRun) TableName() string { return "consolidated_invoice_runs" }

// ConsolidateInvoiceRequest is POST /invoices/consolidate (and the query of
// GET /invoices/consolidate/preview): one customer's billable work for a
// period, as tenant-local calendar dates (inclusive).
type ConsolidateInvoiceRequest struct {
	CustomerID  uuid.UUID `json:"customer_id" form:"customer_id" binding:"required"`
	PeriodStart string    `json:"period_start" form:"period_start" binding:"required" example:"2026-10-01"`
	PeriodEnd   string    `json:"period_end" form:"period_end" binding:"required" example:"2026-10-31"`
	// SiteID limits the invoice to one of the customer's sites.
	SiteID *uuid.UUID `json:"site_id,omitempty" form:"site_id"`
}

// Consolidated line kinds.
const (
	ConsolidatedLineAgreementMonth  = "agreement_month"
	ConsolidatedLineAgreementVisits = "agreement_visits"
	ConsolidatedLineExtraWork       = "extra_work"
)

// ConsolidatedInvoiceLine is one would-be line of a consolidated invoice.
type ConsolidatedInvoiceLine struct {
	// Kind: agreement_month (a monthly agreement for one calendar month),
	// agreement_visits (completed visits × price) or extra_work (a job).
	Kind        string     `json:"kind" enums:"agreement_month,agreement_visits,extra_work"`
	SiteID      *uuid.UUID `json:"site_id,omitempty"`
	SiteName    string     `json:"site_name"`
	AgreementID *uuid.UUID `json:"agreement_id,omitempty"`
	JobID       *uuid.UUID `json:"job_id,omitempty"`
	JobNumber   string     `json:"job_number,omitempty"`
	Description string     `json:"description"`
	// Month is the calendar month billed (agreement_month), YYYY-MM.
	Month     string          `json:"month,omitempty" example:"2026-10"`
	Visits    int             `json:"visits"`
	Quantity  decimal.Decimal `json:"quantity" swaggertype:"string" example:"1"`
	UnitPrice decimal.Decimal `json:"unit_price" swaggertype:"string" example:"2400.00"`
	Amount    decimal.Decimal `json:"amount" swaggertype:"string" example:"2400.00"`
	// ApproverName and ApprovalAmount come from the job's latest recorded
	// approval (extra work only).
	ApproverName   string           `json:"approver_name,omitempty"`
	ApprovalAmount *decimal.Decimal `json:"approval_amount,omitempty" swaggertype:"string" example:"450.00"`
	// ApprovalNote is printed under the line: who approved it, or that it
	// stayed under the customer's not-to-exceed limit.
	ApprovalNote string `json:"approval_note,omitempty"`
}

// ConsolidatedInvoicePreview is what POST /invoices/consolidate would create.
type ConsolidatedInvoicePreview struct {
	CustomerID  uuid.UUID                 `json:"customer_id"`
	PeriodStart string                    `json:"period_start" example:"2026-10-01"`
	PeriodEnd   string                    `json:"period_end" example:"2026-10-31"`
	PONumber    string                    `json:"po_number"`
	Lines       []ConsolidatedInvoiceLine `json:"lines"`
	Sites       int                       `json:"sites"`
	Visits      int                       `json:"visits"`
	ExtraWork   int                       `json:"extra_work"`
	Subtotal    decimal.Decimal           `json:"subtotal" swaggertype:"string" example:"9150.00"`
	TaxRate     decimal.Decimal           `json:"tax_rate" swaggertype:"string" example:"0.08"`
	TaxAmount   decimal.Decimal           `json:"tax_amount" swaggertype:"string" example:"732.00"`
	Total       decimal.Decimal           `json:"total" swaggertype:"string" example:"9882.00"`
}
