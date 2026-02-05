package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// AgingBucket represents the aging period for an invoice
type AgingBucket string

const (
	AgingBucketCurrent AgingBucket = "current"
	AgingBucket1To30   AgingBucket = "1-30"
	AgingBucket31To60  AgingBucket = "31-60"
	AgingBucket61To90  AgingBucket = "61-90"
	AgingBucket90Plus  AgingBucket = "90+"
)

// ARAgingQuery represents query parameters for A/R aging report
type ARAgingQuery struct {
	AsOfDate     *time.Time `json:"as_of_date" form:"as_of_date"`
	CustomerID   *uuid.UUID `json:"customer_id" form:"customer_id"`
	MinAmount    *float64   `json:"min_amount" form:"min_amount"`
	IncludePaid  bool       `json:"include_paid" form:"include_paid"`
	IncludeDraft bool       `json:"include_draft" form:"include_draft"`
}

// ARAgingBucketSummary represents the summary for a single aging bucket
type ARAgingBucketSummary struct {
	Bucket       AgingBucket     `json:"bucket"`
	DisplayName  string          `json:"display_name"`
	InvoiceCount int             `json:"invoice_count"`
	TotalAmount  decimal.Decimal `json:"total_amount"`
	Percentage   decimal.Decimal `json:"percentage"`
	Color        string          `json:"color"`
}

// ARAgingInvoice represents an invoice in the aging report
type ARAgingInvoice struct {
	InvoiceID     uuid.UUID       `json:"invoice_id"`
	InvoiceNumber string          `json:"invoice_number"`
	CustomerID    uuid.UUID       `json:"customer_id"`
	CustomerName  string          `json:"customer_name"`
	CompanyName   *string         `json:"company_name,omitempty"`
	IssueDate     time.Time       `json:"issue_date"`
	DueDate       time.Time       `json:"due_date"`
	TotalAmount   decimal.Decimal `json:"total_amount"`
	AmountPaid    decimal.Decimal `json:"amount_paid"`
	AmountDue     decimal.Decimal `json:"amount_due"`
	DaysOverdue   int             `json:"days_overdue"`
	Bucket        AgingBucket     `json:"bucket"`
	Status        InvoiceStatus   `json:"status"`
}

// ARAgingCustomerSummary represents A/R summary for a single customer
type ARAgingCustomerSummary struct {
	CustomerID    uuid.UUID       `json:"customer_id"`
	CustomerName  string          `json:"customer_name"`
	CompanyName   *string         `json:"company_name,omitempty"`
	Email         string          `json:"email"`
	InvoiceCount  int             `json:"invoice_count"`
	TotalOwed     decimal.Decimal `json:"total_owed"`
	Current       decimal.Decimal `json:"current"`
	Days1To30     decimal.Decimal `json:"days_1_to_30"`
	Days31To60    decimal.Decimal `json:"days_31_to_60"`
	Days61To90    decimal.Decimal `json:"days_61_to_90"`
	Days90Plus    decimal.Decimal `json:"days_90_plus"`
	OldestInvoice *time.Time      `json:"oldest_invoice,omitempty"`
	LastPayment   *time.Time      `json:"last_payment,omitempty"`
}

// ARAgingReportSummary provides overall A/R summary
type ARAgingReportSummary struct {
	TotalOutstanding   decimal.Decimal `json:"total_outstanding"`
	TotalCurrent       decimal.Decimal `json:"total_current"`
	TotalOverdue       decimal.Decimal `json:"total_overdue"`
	OverduePercentage  decimal.Decimal `json:"overdue_percentage"`
	InvoiceCount       int             `json:"invoice_count"`
	OverdueCount       int             `json:"overdue_count"`
	CustomerCount      int             `json:"customer_count"`
	AverageDaysOverdue decimal.Decimal `json:"average_days_overdue"`
}

// ARAgingReportResponse represents the full A/R aging report
type ARAgingReportResponse struct {
	Summary         ARAgingReportSummary     `json:"summary"`
	Buckets         []ARAgingBucketSummary   `json:"buckets"`
	CustomerSummary []ARAgingCustomerSummary `json:"customer_summary,omitempty"`
	Invoices        []ARAgingInvoice         `json:"invoices,omitempty"`
	Metadata        ARAgingReportMetadata    `json:"metadata"`
}

// ARAgingReportMetadata provides metadata about the report
type ARAgingReportMetadata struct {
	GeneratedAt   time.Time    `json:"generated_at"`
	AsOfDate      time.Time    `json:"as_of_date"`
	Query         ARAgingQuery `json:"query"`
	ExecutionTime int64        `json:"execution_time_ms"`
}

// ARAgingExportRequest represents a request to export A/R aging data
type ARAgingExportRequest struct {
	Query           ARAgingQuery `json:"query"`
	Format          ExportFormat `json:"format"` // csv, json, xlsx
	IncludeSections []string     `json:"include_sections,omitempty"`
	Filename        string       `json:"filename,omitempty"`
}

// GetBucketDisplayName returns a human-readable name for a bucket
func GetBucketDisplayName(bucket AgingBucket) string {
	switch bucket {
	case AgingBucketCurrent:
		return "Current"
	case AgingBucket1To30:
		return "1-30 Days"
	case AgingBucket31To60:
		return "31-60 Days"
	case AgingBucket61To90:
		return "61-90 Days"
	case AgingBucket90Plus:
		return "90+ Days"
	default:
		return string(bucket)
	}
}

// GetBucketColor returns a color for an aging bucket
func GetBucketColor(bucket AgingBucket) string {
	switch bucket {
	case AgingBucketCurrent:
		return "#22C55E" // Green
	case AgingBucket1To30:
		return "#EAB308" // Yellow
	case AgingBucket31To60:
		return "#F97316" // Orange
	case AgingBucket61To90:
		return "#EF4444" // Red
	case AgingBucket90Plus:
		return "#991B1B" // Dark Red
	default:
		return "#9CA3AF"
	}
}

// GetAgingBucket determines the aging bucket for a given number of days overdue
func GetAgingBucket(daysOverdue int) AgingBucket {
	switch {
	case daysOverdue <= 0:
		return AgingBucketCurrent
	case daysOverdue <= 30:
		return AgingBucket1To30
	case daysOverdue <= 60:
		return AgingBucket31To60
	case daysOverdue <= 90:
		return AgingBucket61To90
	default:
		return AgingBucket90Plus
	}
}
