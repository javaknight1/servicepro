package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

// TransactionType represents the type of revenue transaction
type TransactionType string

const (
	TransactionTypePayment    TransactionType = "payment"
	TransactionTypeRefund     TransactionType = "refund"
	TransactionTypeChargeback TransactionType = "chargeback"
	TransactionTypeAdjustment TransactionType = "adjustment"
)

// TransactionStatus represents the status of a revenue transaction
type TransactionStatus string

const (
	TransactionStatusCompleted TransactionStatus = "completed"
	TransactionStatusPending   TransactionStatus = "pending"
	TransactionStatusFailed    TransactionStatus = "failed"
	TransactionStatusReversed  TransactionStatus = "reversed"
)

// PeriodType represents the type of reporting period
type PeriodType string

const (
	PeriodTypeDaily     PeriodType = "daily"
	PeriodTypeWeekly    PeriodType = "weekly"
	PeriodTypeMonthly   PeriodType = "monthly"
	PeriodTypeQuarterly PeriodType = "quarterly"
	PeriodTypeYearly    PeriodType = "yearly"
)

// RevenueTransaction represents a revenue transaction record
type RevenueTransaction struct {
	ID        uuid.UUID  `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	PaymentID uuid.UUID  `gorm:"type:uuid;not null" json:"payment_id"`
	UserID    uuid.UUID  `gorm:"type:uuid;not null" json:"user_id"`
	InvoiceID *uuid.UUID `gorm:"type:uuid" json:"invoice_id,omitempty"`
	OrderID   *uuid.UUID `gorm:"type:uuid" json:"order_id,omitempty"`

	// Transaction details
	TransactionType TransactionType `gorm:"type:varchar(50);not null" json:"transaction_type"`
	TransactionDate time.Time       `gorm:"not null" json:"transaction_date"`

	// Amounts
	GrossAmount    decimal.Decimal `gorm:"type:decimal(12,2);not null" json:"gross_amount"`
	NetAmount      decimal.Decimal `gorm:"type:decimal(12,2);not null" json:"net_amount"`
	FeeAmount      decimal.Decimal `gorm:"type:decimal(12,2);not null;default:0" json:"fee_amount"`
	TaxAmount      decimal.Decimal `gorm:"type:decimal(12,2);not null;default:0" json:"tax_amount"`
	DiscountAmount decimal.Decimal `gorm:"type:decimal(12,2);not null;default:0" json:"discount_amount"`
	RefundAmount   decimal.Decimal `gorm:"type:decimal(12,2);not null;default:0" json:"refund_amount"`

	// Currency and exchange
	Currency           string          `gorm:"type:varchar(3);not null;default:'USD'" json:"currency"`
	ExchangeRate       decimal.Decimal `gorm:"type:decimal(10,4);default:1.0000" json:"exchange_rate"`
	BaseCurrencyAmount decimal.Decimal `gorm:"type:decimal(12,2)" json:"base_currency_amount"`

	// Category and tags
	RevenueCategory    string   `gorm:"type:varchar(100)" json:"revenue_category,omitempty"`
	RevenueSubcategory string   `gorm:"type:varchar(100)" json:"revenue_subcategory,omitempty"`
	Tags               []string `gorm:"type:text[]" json:"tags,omitempty"`

	// Status
	Status     TransactionStatus `gorm:"type:varchar(20);not null;default:'completed'" json:"status"`
	IsRefunded bool              `gorm:"not null;default:false" json:"is_refunded"`
	IsDisputed bool              `gorm:"not null;default:false" json:"is_disputed"`

	// Metadata
	Metadata map[string]interface{} `gorm:"type:jsonb" json:"metadata,omitempty"`

	// Timestamps
	CreatedAt time.Time `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt time.Time `gorm:"autoUpdateTime" json:"updated_at"`
}

// TableName specifies the table name
func (RevenueTransaction) TableName() string {
	return "revenue_transactions"
}

// BeforeCreate hook
func (rt *RevenueTransaction) BeforeCreate(tx *gorm.DB) error {
	if rt.ID == uuid.Nil {
		rt.ID = uuid.New()
	}
	return nil
}

// RevenueSummaryCache represents cached revenue summary data
type RevenueSummaryCache struct {
	ID          uuid.UUID  `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	PeriodType  PeriodType `gorm:"type:varchar(20);not null" json:"period_type"`
	PeriodStart time.Time  `gorm:"not null" json:"period_start"`
	PeriodEnd   time.Time  `gorm:"not null" json:"period_end"`

	// Aggregated metrics
	TotalGrossRevenue decimal.Decimal `gorm:"type:decimal(12,2);not null;default:0" json:"total_gross_revenue"`
	TotalNetRevenue   decimal.Decimal `gorm:"type:decimal(12,2);not null;default:0" json:"total_net_revenue"`
	TotalFees         decimal.Decimal `gorm:"type:decimal(12,2);not null;default:0" json:"total_fees"`
	TotalTaxes        decimal.Decimal `gorm:"type:decimal(12,2);not null;default:0" json:"total_taxes"`
	TotalDiscounts    decimal.Decimal `gorm:"type:decimal(12,2);not null;default:0" json:"total_discounts"`
	TotalRefunds      decimal.Decimal `gorm:"type:decimal(12,2);not null;default:0" json:"total_refunds"`

	// Transaction counts
	TransactionCount    int `gorm:"not null;default:0" json:"transaction_count"`
	PaymentCount        int `gorm:"not null;default:0" json:"payment_count"`
	RefundCount         int `gorm:"not null;default:0" json:"refund_count"`
	UniqueCustomerCount int `gorm:"not null;default:0" json:"unique_customer_count"`

	// Average metrics
	AverageTransactionValue decimal.Decimal `gorm:"type:decimal(12,2)" json:"average_transaction_value"`
	AverageOrderValue       decimal.Decimal `gorm:"type:decimal(12,2)" json:"average_order_value"`

	// Growth metrics
	GrowthRate            decimal.Decimal `gorm:"type:decimal(5,2)" json:"growth_rate"`
	PreviousPeriodRevenue decimal.Decimal `gorm:"type:decimal(12,2)" json:"previous_period_revenue"`

	// Metadata
	Currency string                 `gorm:"type:varchar(3);default:'USD'" json:"currency"`
	Metadata map[string]interface{} `gorm:"type:jsonb" json:"metadata,omitempty"`

	// Cache control
	LastUpdated time.Time `gorm:"not null" json:"last_updated"`
	IsStale     bool      `gorm:"not null;default:false" json:"is_stale"`
}

// TableName specifies the table name
func (RevenueSummaryCache) TableName() string {
	return "revenue_summary_cache"
}

// RevenueReportQuery represents a query for revenue reports
type RevenueReportQuery struct {
	StartDate       *time.Time       `json:"start_date" form:"start_date"`
	EndDate         *time.Time       `json:"end_date" form:"end_date"`
	PeriodType      PeriodType       `json:"period_type" form:"period_type"`
	Currency        string           `json:"currency" form:"currency"`
	Category        string           `json:"category" form:"category"`
	Subcategory     string           `json:"subcategory" form:"subcategory"`
	UserID          *uuid.UUID       `json:"user_id" form:"user_id"`
	TransactionType *TransactionType `json:"transaction_type" form:"transaction_type"`
	IncludeRefunds  bool             `json:"include_refunds" form:"include_refunds"`
	Grouping        string           `json:"grouping" form:"grouping"` // day, week, month, quarter, year, category
	MinAmount       *decimal.Decimal `json:"min_amount" form:"min_amount"`
	MaxAmount       *decimal.Decimal `json:"max_amount" form:"max_amount"`
	Page            int              `json:"page" form:"page"`
	PageSize        int              `json:"page_size" form:"page_size"`
}

// RevenueReportResponse represents the response for revenue reports
type RevenueReportResponse struct {
	Summary      RevenueSummary          `json:"summary"`
	TimeSeries   []RevenueDataPoint      `json:"time_series"`
	ByCategory   []RevenueByCategoryData `json:"by_category,omitempty"`
	TopCustomers []CustomerRevenueData   `json:"top_customers,omitempty"`
	Metadata     RevenueReportMetadata   `json:"metadata"`
}

// RevenueSummary provides summary metrics
type RevenueSummary struct {
	TotalGrossRevenue       decimal.Decimal `json:"total_gross_revenue"`
	TotalNetRevenue         decimal.Decimal `json:"total_net_revenue"`
	TotalFees               decimal.Decimal `json:"total_fees"`
	TotalTax                decimal.Decimal `json:"total_tax"`
	TotalDiscounts          decimal.Decimal `json:"total_discounts"`
	TotalRefunds            decimal.Decimal `json:"total_refunds"`
	TransactionCount        int             `json:"transaction_count"`
	PaymentCount            int             `json:"payment_count"`
	RefundCount             int             `json:"refund_count"`
	UniqueCustomerCount     int             `json:"unique_customer_count"`
	AverageTransactionValue decimal.Decimal `json:"average_transaction_value"`
	AverageOrderValue       decimal.Decimal `json:"average_order_value"`
	GrowthRate              decimal.Decimal `json:"growth_rate"`
	PreviousPeriodRevenue   decimal.Decimal `json:"previous_period_revenue"`
	Currency                string          `json:"currency"`
}

// RevenueDataPoint represents a single data point in a time series
type RevenueDataPoint struct {
	Period                  time.Time       `json:"period" gorm:"column:period"`
	GrossRevenue            decimal.Decimal `json:"gross_revenue" gorm:"column:gross_revenue"`
	NetRevenue              decimal.Decimal `json:"net_revenue" gorm:"column:net_revenue"`
	Fees                    decimal.Decimal `json:"fees" gorm:"column:fees"`
	Tax                     decimal.Decimal `json:"tax" gorm:"column:tax"`
	Discounts               decimal.Decimal `json:"discounts" gorm:"column:discounts"`
	Refunds                 decimal.Decimal `json:"refunds" gorm:"column:refunds"`
	TransactionCount        int             `json:"transaction_count" gorm:"column:transaction_count"`
	UniqueCustomers         int             `json:"unique_customers" gorm:"column:unique_customers"`
	AverageTransactionValue decimal.Decimal `json:"average_transaction_value" gorm:"column:average_transaction_value"`
}

// RevenueByCategoryData represents revenue breakdown by category
type RevenueByCategoryData struct {
	Category         string          `json:"category" gorm:"column:category"`
	Subcategory      string          `json:"subcategory,omitempty" gorm:"column:subcategory"`
	Revenue          decimal.Decimal `json:"revenue" gorm:"column:revenue"`
	TransactionCount int             `json:"transaction_count" gorm:"column:transaction_count"`
	AverageValue     decimal.Decimal `json:"average_value" gorm:"column:average_value"`
	Percentage       decimal.Decimal `json:"percentage"`
}

// CustomerRevenueData represents revenue data for a customer
type CustomerRevenueData struct {
	UserID                  uuid.UUID       `json:"user_id" gorm:"column:user_id"`
	UserEmail               string          `json:"user_email,omitempty"`
	UserName                string          `json:"user_name,omitempty"`
	TotalRevenue            decimal.Decimal `json:"total_revenue" gorm:"column:total_revenue"`
	TransactionCount        int             `json:"transaction_count" gorm:"column:transaction_count"`
	AverageTransactionValue decimal.Decimal `json:"average_transaction_value" gorm:"column:average_transaction_value"`
	FirstTransactionDate    time.Time       `json:"first_transaction_date" gorm:"column:first_transaction_date"`
	LastTransactionDate     time.Time       `json:"last_transaction_date" gorm:"column:last_transaction_date"`
	LifetimeValue           decimal.Decimal `json:"lifetime_value"`
}

// RevenueReportMetadata provides metadata about the report
type RevenueReportMetadata struct {
	GeneratedAt   time.Time          `json:"generated_at"`
	Query         RevenueReportQuery `json:"query"`
	DataPoints    int                `json:"data_points"`
	CacheUsed     bool               `json:"cache_used"`
	ExecutionTime int64              `json:"execution_time_ms"`
}

// DateRange represents a date range
type DateRange struct {
	Start time.Time `json:"start"`
	End   time.Time `json:"end"`
}

// RevenueComparison compares revenue between two periods
type RevenueComparison struct {
	CurrentPeriod  RevenueSummary `json:"current_period"`
	PreviousPeriod RevenueSummary `json:"previous_period"`
	Growth         GrowthMetrics  `json:"growth"`
}

// GrowthMetrics represents growth calculations
type GrowthMetrics struct {
	RevenueGrowth         decimal.Decimal `json:"revenue_growth"`
	RevenueGrowthRate     decimal.Decimal `json:"revenue_growth_rate"`
	TransactionGrowth     int             `json:"transaction_growth"`
	TransactionGrowthRate decimal.Decimal `json:"transaction_growth_rate"`
	CustomerGrowth        int             `json:"customer_growth"`
	CustomerGrowthRate    decimal.Decimal `json:"customer_growth_rate"`
}

// ReportExportRequest represents a request to export revenue data
type ReportExportRequest struct {
	Query           RevenueReportQuery `json:"query"`
	Format          ExportFormat       `json:"format"` // csv, xlsx, pdf, json
	IncludeSections []string           `json:"include_sections,omitempty"`
	IncludeCharts   bool               `json:"include_charts"`
	Filename        string             `json:"filename,omitempty"`
}

// ReportExportResponse represents the response for an export request
type ReportExportResponse struct {
	ExportID    uuid.UUID    `json:"export_id"`
	FileName    string       `json:"file_name"`
	Format      ExportFormat `json:"format"`
	Status      string       `json:"status"`
	Data        interface{}  `json:"data,omitempty"`
	FileSize    int64        `json:"file_size"`
	ExpiresAt   time.Time    `json:"expires_at"`
	GeneratedAt time.Time    `json:"generated_at"`
}

// RevenueExportRequest is an alias for ReportExportRequest (backward compatibility)
type RevenueExportRequest = ReportExportRequest

// RevenueExportResponse is an alias for ReportExportResponse (backward compatibility)
type RevenueExportResponse = ReportExportResponse

// RevenueChartData represents data formatted for charts
type RevenueChartData struct {
	Labels   []string               `json:"labels"`
	Datasets []RevenueChartDataset  `json:"datasets"`
	Options  map[string]interface{} `json:"options,omitempty"`
}

// RevenueChartDataset represents a dataset in a chart
type RevenueChartDataset struct {
	Label           string            `json:"label"`
	Data            []decimal.Decimal `json:"data"`
	BackgroundColor string            `json:"backgroundColor,omitempty"`
	BorderColor     string            `json:"borderColor,omitempty"`
	Fill            bool              `json:"fill"`
	Tension         float64           `json:"tension,omitempty"`
}

// RevenueMetrics represents key performance indicators
type RevenueMetrics struct {
	MRR                   decimal.Decimal `json:"mrr"`  // Monthly Recurring Revenue
	ARR                   decimal.Decimal `json:"arr"`  // Annual Recurring Revenue
	ARPU                  decimal.Decimal `json:"arpu"` // Average Revenue Per User
	ChurnRate             decimal.Decimal `json:"churn_rate"`
	CustomerLifetimeValue decimal.Decimal `json:"customer_lifetime_value"`
	RevenuePerCustomer    decimal.Decimal `json:"revenue_per_customer"`
	Currency              string          `json:"currency"`
	CalculatedAt          time.Time       `json:"calculated_at"`
}

// RevenueForecast represents forecasted revenue
type RevenueForecast struct {
	Period            string          `json:"period"`
	ForecastedRevenue decimal.Decimal `json:"forecasted_revenue"`
	LowerBound        decimal.Decimal `json:"lower_bound"`
	UpperBound        decimal.Decimal `json:"upper_bound"`
	Confidence        decimal.Decimal `json:"confidence"`
}

// RevenueAlert represents an alert condition
type RevenueAlert struct {
	ID            uuid.UUID              `gorm:"type:uuid;primary_key" json:"id"`
	Name          string                 `gorm:"type:varchar(200);not null" json:"name"`
	Description   string                 `gorm:"type:text" json:"description,omitempty"`
	Condition     map[string]interface{} `gorm:"type:jsonb;not null" json:"condition"`
	IsActive      bool                   `gorm:"not null;default:true" json:"is_active"`
	LastTriggered *time.Time             `json:"last_triggered,omitempty"`
	CreatedAt     time.Time              `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt     time.Time              `gorm:"autoUpdateTime" json:"updated_at"`
}

// TableName specifies the table name
func (RevenueAlert) TableName() string {
	return "revenue_alerts"
}

// Helper methods

// CalculateGrowthRate calculates the growth rate between two values
func CalculateGrowthRate(current, previous decimal.Decimal) decimal.Decimal {
	if previous.IsZero() {
		return decimal.Zero
	}
	return current.Sub(previous).Div(previous).Mul(decimal.NewFromInt(100))
}

// FormatCurrency formats a decimal amount as currency
func FormatCurrency(amount decimal.Decimal, currency string) string {
	switch currency {
	case "USD":
		return "$" + amount.StringFixed(2)
	case "EUR":
		return "€" + amount.StringFixed(2)
	case "GBP":
		return "£" + amount.StringFixed(2)
	default:
		return amount.StringFixed(2) + " " + currency
	}
}

// GetPeriodStart returns the start of a period based on the period type
func GetPeriodStart(t time.Time, periodType PeriodType) time.Time {
	switch periodType {
	case PeriodTypeDaily:
		return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
	case PeriodTypeWeekly:
		// Start of week (Monday)
		weekday := int(t.Weekday())
		if weekday == 0 {
			weekday = 7
		}
		return time.Date(t.Year(), t.Month(), t.Day()-weekday+1, 0, 0, 0, 0, t.Location())
	case PeriodTypeMonthly:
		return time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, t.Location())
	case PeriodTypeQuarterly:
		month := ((int(t.Month())-1)/3)*3 + 1
		return time.Date(t.Year(), time.Month(month), 1, 0, 0, 0, 0, t.Location())
	case PeriodTypeYearly:
		return time.Date(t.Year(), 1, 1, 0, 0, 0, 0, t.Location())
	default:
		return t
	}
}

// GetPeriodEnd returns the end of a period based on the period type
func GetPeriodEnd(start time.Time, periodType PeriodType) time.Time {
	switch periodType {
	case PeriodTypeDaily:
		return start.AddDate(0, 0, 1)
	case PeriodTypeWeekly:
		return start.AddDate(0, 0, 7)
	case PeriodTypeMonthly:
		return start.AddDate(0, 1, 0)
	case PeriodTypeQuarterly:
		return start.AddDate(0, 3, 0)
	case PeriodTypeYearly:
		return start.AddDate(1, 0, 0)
	default:
		return start
	}
}
