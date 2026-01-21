package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// CustomerSegment represents a customer segment classification
type CustomerSegment string

const (
	CustomerSegmentHighValue  CustomerSegment = "high_value"
	CustomerSegmentRegular    CustomerSegment = "regular"
	CustomerSegmentOccasional CustomerSegment = "occasional"
	CustomerSegmentAtRisk     CustomerSegment = "at_risk"
	CustomerSegmentChurned    CustomerSegment = "churned"
	CustomerSegmentNew        CustomerSegment = "new"
)

// CustomerActivityType represents types of customer activities
type CustomerActivityType string

const (
	ActivityTypeJobCreated      CustomerActivityType = "job_created"
	ActivityTypeJobCompleted    CustomerActivityType = "job_completed"
	ActivityTypeQuoteSent       CustomerActivityType = "quote_sent"
	ActivityTypeQuoteAccepted   CustomerActivityType = "quote_accepted"
	ActivityTypePaymentReceived CustomerActivityType = "payment_received"
	ActivityTypeNoteAdded       CustomerActivityType = "note_added"
)

// CustomerActivity represents a customer activity record
type CustomerActivity struct {
	ID            uuid.UUID            `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	CustomerID    uuid.UUID            `gorm:"type:uuid;not null" json:"customer_id"`
	ActivityType  CustomerActivityType `gorm:"type:varchar(50);not null" json:"activity_type"`
	ActivityDate  time.Time            `gorm:"not null;default:CURRENT_TIMESTAMP" json:"activity_date"`
	ReferenceID   *uuid.UUID           `gorm:"type:uuid" json:"reference_id,omitempty"`
	ReferenceType *string              `gorm:"type:varchar(50)" json:"reference_type,omitempty"`
	Amount        decimal.Decimal      `gorm:"type:decimal(12,2)" json:"amount,omitempty"`
	Metadata      map[string]any       `gorm:"type:jsonb" json:"metadata,omitempty"`
	CreatedAt     time.Time            `gorm:"not null;default:CURRENT_TIMESTAMP" json:"created_at"`
}

func (CustomerActivity) TableName() string {
	return "customer_activities"
}

// CustomerMetricsCache represents cached customer metrics
type CustomerMetricsCache struct {
	ID         uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	CustomerID uuid.UUID `gorm:"type:uuid;not null;uniqueIndex" json:"customer_id"`

	// Financial metrics
	TotalRevenue     decimal.Decimal `gorm:"type:decimal(12,2);not null;default:0" json:"total_revenue"`
	TotalJobsRevenue decimal.Decimal `gorm:"type:decimal(12,2);not null;default:0" json:"total_jobs_revenue"`
	AverageJobValue  decimal.Decimal `gorm:"type:decimal(12,2);not null;default:0" json:"average_job_value"`

	// Job metrics
	TotalJobs     int `gorm:"not null;default:0" json:"total_jobs"`
	CompletedJobs int `gorm:"not null;default:0" json:"completed_jobs"`
	CancelledJobs int `gorm:"not null;default:0" json:"cancelled_jobs"`
	PendingJobs   int `gorm:"not null;default:0" json:"pending_jobs"`

	// Quote metrics
	TotalQuotes         int             `gorm:"not null;default:0" json:"total_quotes"`
	AcceptedQuotes      int             `gorm:"not null;default:0" json:"accepted_quotes"`
	RejectedQuotes      int             `gorm:"not null;default:0" json:"rejected_quotes"`
	PendingQuotes       int             `gorm:"not null;default:0" json:"pending_quotes"`
	QuoteAcceptanceRate decimal.Decimal `gorm:"type:decimal(5,2);not null;default:0" json:"quote_acceptance_rate"`

	// Activity metrics
	FirstActivityDate     *time.Time `json:"first_activity_date,omitempty"`
	LastActivityDate      *time.Time `json:"last_activity_date,omitempty"`
	DaysSinceLastActivity int        `gorm:"default:0" json:"days_since_last_activity"`
	TotalInteractions     int        `gorm:"not null;default:0" json:"total_interactions"`

	// Lifecycle metrics
	CustomerLifetimeDays   int             `gorm:"not null;default:0" json:"customer_lifetime_days"`
	AverageDaysBetweenJobs decimal.Decimal `gorm:"type:decimal(10,2)" json:"average_days_between_jobs,omitempty"`

	// Segmentation
	Segment      CustomerSegment `gorm:"type:varchar(50)" json:"segment"`
	SegmentScore decimal.Decimal `gorm:"type:decimal(5,2)" json:"segment_score,omitempty"`

	// Cache control
	LastUpdated time.Time `gorm:"not null" json:"last_updated"`
	IsStale     bool      `gorm:"not null;default:false" json:"is_stale"`
}

func (CustomerMetricsCache) TableName() string {
	return "customer_metrics_cache"
}

// CustomerSegmentDefinition represents a segment definition
type CustomerSegmentDefinition struct {
	ID          uuid.UUID      `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	Name        string         `gorm:"type:varchar(50);not null;uniqueIndex" json:"name"`
	Description string         `gorm:"type:text" json:"description,omitempty"`
	Criteria    map[string]any `gorm:"type:jsonb;not null" json:"criteria"`
	Color       string         `gorm:"type:varchar(7)" json:"color,omitempty"`
	Priority    int            `gorm:"not null;default:0" json:"priority"`
	IsActive    bool           `gorm:"not null;default:true" json:"is_active"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
}

func (CustomerSegmentDefinition) TableName() string {
	return "customer_segments"
}

// CustomerReportQuery represents query parameters for customer reports
type CustomerReportQuery struct {
	StartDate    *time.Time       `json:"start_date" form:"start_date"`
	EndDate      *time.Time       `json:"end_date" form:"end_date"`
	Segment      CustomerSegment  `json:"segment" form:"segment"`
	CustomerType CustomerType     `json:"customer_type" form:"customer_type"`
	Status       CustomerStatus   `json:"status" form:"status"`
	State        string           `json:"state" form:"state"`
	City         string           `json:"city" form:"city"`
	MinRevenue   *decimal.Decimal `json:"min_revenue" form:"min_revenue"`
	MaxRevenue   *decimal.Decimal `json:"max_revenue" form:"max_revenue"`
	MinJobs      *int             `json:"min_jobs" form:"min_jobs"`
	MaxJobs      *int             `json:"max_jobs" form:"max_jobs"`
	SortBy       string           `json:"sort_by" form:"sort_by"`
	SortOrder    string           `json:"sort_order" form:"sort_order"`
	Page         int              `json:"page" form:"page"`
	PageSize     int              `json:"page_size" form:"page_size"`
}

// CustomerReportResponse represents the response for customer reports
type CustomerReportResponse struct {
	Summary            CustomerReportSummary    `json:"summary"`
	SegmentBreakdown   []SegmentBreakdownData   `json:"segment_breakdown"`
	TypeBreakdown      []TypeBreakdownData      `json:"type_breakdown"`
	GeographyBreakdown []GeographyBreakdownData `json:"geography_breakdown,omitempty"`
	AcquisitionTrend   []AcquisitionTrendData   `json:"acquisition_trend,omitempty"`
	TopCustomers       []CustomerMetricsData    `json:"top_customers,omitempty"`
	AtRiskCustomers    []CustomerMetricsData    `json:"at_risk_customers,omitempty"`
	Metadata           CustomerReportMetadata   `json:"metadata"`
}

// CustomerReportSummary provides summary metrics for customer report
type CustomerReportSummary struct {
	TotalCustomers             int             `json:"total_customers"`
	ActiveCustomers            int             `json:"active_customers"`
	InactiveCustomers          int             `json:"inactive_customers"`
	Prospects                  int             `json:"prospects"`
	NewCustomersThisPeriod     int             `json:"new_customers_this_period"`
	ChurnedCustomers           int             `json:"churned_customers"`
	TotalRevenue               decimal.Decimal `json:"total_revenue"`
	AverageRevenuePerCustomer  decimal.Decimal `json:"average_revenue_per_customer"`
	AverageLifetimeValue       decimal.Decimal `json:"average_lifetime_value"`
	AverageJobsPerCustomer     decimal.Decimal `json:"average_jobs_per_customer"`
	OverallQuoteAcceptanceRate decimal.Decimal `json:"overall_quote_acceptance_rate"`
	GrowthRate                 decimal.Decimal `json:"growth_rate"`
	RetentionRate              decimal.Decimal `json:"retention_rate"`
}

// SegmentBreakdownData represents segment statistics
type SegmentBreakdownData struct {
	Segment        CustomerSegment `json:"segment"`
	DisplayName    string          `json:"display_name"`
	Count          int             `json:"count"`
	Percentage     decimal.Decimal `json:"percentage"`
	TotalRevenue   decimal.Decimal `json:"total_revenue"`
	AverageRevenue decimal.Decimal `json:"average_revenue"`
	Color          string          `json:"color"`
}

// TypeBreakdownData represents customer type statistics
type TypeBreakdownData struct {
	CustomerType CustomerType    `json:"customer_type"`
	Status       CustomerStatus  `json:"status"`
	Count        int             `json:"count"`
	Percentage   decimal.Decimal `json:"percentage"`
	TotalRevenue decimal.Decimal `json:"total_revenue"`
}

// GeographyBreakdownData represents geographic distribution
type GeographyBreakdownData struct {
	State         string          `json:"state"`
	City          string          `json:"city,omitempty"`
	CustomerCount int             `json:"customer_count"`
	ActiveCount   int             `json:"active_count"`
	Percentage    decimal.Decimal `json:"percentage"`
	TotalRevenue  decimal.Decimal `json:"total_revenue"`
}

// AcquisitionTrendData represents customer acquisition over time
type AcquisitionTrendData struct {
	Period           time.Time `json:"period"`
	NewCustomers     int       `json:"new_customers"`
	ActiveCustomers  int       `json:"active_customers"`
	Prospects        int       `json:"prospects"`
	ChurnedCustomers int       `json:"churned_customers"`
	NetGrowth        int       `json:"net_growth"`
}

// CustomerMetricsData represents detailed metrics for a customer
type CustomerMetricsData struct {
	CustomerID            uuid.UUID       `json:"customer_id"`
	FirstName             string          `json:"first_name"`
	LastName              string          `json:"last_name"`
	CompanyName           *string         `json:"company_name,omitempty"`
	Email                 string          `json:"email"`
	CustomerType          CustomerType    `json:"customer_type"`
	Status                CustomerStatus  `json:"status"`
	Segment               CustomerSegment `json:"segment"`
	TotalRevenue          decimal.Decimal `json:"total_revenue"`
	TotalJobs             int             `json:"total_jobs"`
	CompletedJobs         int             `json:"completed_jobs"`
	TotalQuotes           int             `json:"total_quotes"`
	AcceptedQuotes        int             `json:"accepted_quotes"`
	QuoteAcceptanceRate   decimal.Decimal `json:"quote_acceptance_rate"`
	AverageJobValue       decimal.Decimal `json:"average_job_value"`
	LastActivityDate      *time.Time      `json:"last_activity_date,omitempty"`
	DaysSinceLastActivity int             `json:"days_since_last_activity"`
	CustomerLifetimeDays  int             `json:"customer_lifetime_days"`
	State                 string          `json:"state"`
	City                  string          `json:"city"`
}

// CustomerReportMetadata provides metadata about the report
type CustomerReportMetadata struct {
	GeneratedAt   time.Time           `json:"generated_at"`
	Query         CustomerReportQuery `json:"query"`
	TotalRecords  int                 `json:"total_records"`
	CacheUsed     bool                `json:"cache_used"`
	ExecutionTime int64               `json:"execution_time_ms"`
}

// CustomerActivitySummary represents activity summary for a customer
type CustomerActivitySummary struct {
	CustomerID       uuid.UUID                    `json:"customer_id"`
	TotalActivities  int                          `json:"total_activities"`
	ByType           map[CustomerActivityType]int `json:"by_type"`
	FirstActivity    *time.Time                   `json:"first_activity,omitempty"`
	LastActivity     *time.Time                   `json:"last_activity,omitempty"`
	RecentActivities []CustomerActivity           `json:"recent_activities,omitempty"`
}

// CohortData represents customer cohort retention data
type CohortData struct {
	CohortMonth     time.Time       `json:"cohort_month"`
	CohortSize      int             `json:"cohort_size"`
	Month1Retained  int             `json:"month_1_retained"`
	Month2Retained  int             `json:"month_2_retained"`
	Month3Retained  int             `json:"month_3_retained"`
	Month6Retained  int             `json:"month_6_retained"`
	RetentionRate1M decimal.Decimal `json:"retention_rate_1m"`
	RetentionRate3M decimal.Decimal `json:"retention_rate_3m"`
	RetentionRate6M decimal.Decimal `json:"retention_rate_6m"`
}

// CustomerExportRequest represents a request to export customer data
type CustomerReportExportRequest struct {
	Query           CustomerReportQuery `json:"query"`
	Format          ExportFormat        `json:"format"` // csv, json, xlsx
	IncludeSections []string            `json:"include_sections,omitempty"`
	Filename        string              `json:"filename,omitempty"`
}

// CustomerChartData represents data formatted for charts
type CustomerChartData struct {
	Labels   []string               `json:"labels"`
	Datasets []CustomerChartDataset `json:"datasets"`
	Options  map[string]any         `json:"options,omitempty"`
}

// CustomerChartDataset represents a dataset in a chart
type CustomerChartDataset struct {
	Label           string    `json:"label"`
	Data            []float64 `json:"data"`
	BackgroundColor []string  `json:"backgroundColor,omitempty"`
	BorderColor     string    `json:"borderColor,omitempty"`
	Fill            bool      `json:"fill"`
}

// GetSegmentDisplayName returns a human-readable name for a segment
func GetSegmentDisplayName(segment CustomerSegment) string {
	switch segment {
	case CustomerSegmentHighValue:
		return "High Value"
	case CustomerSegmentRegular:
		return "Regular"
	case CustomerSegmentOccasional:
		return "Occasional"
	case CustomerSegmentAtRisk:
		return "At Risk"
	case CustomerSegmentChurned:
		return "Churned"
	case CustomerSegmentNew:
		return "New"
	default:
		return string(segment)
	}
}

// GetSegmentColor returns a color for a segment
func GetSegmentColor(segment CustomerSegment) string {
	switch segment {
	case CustomerSegmentHighValue:
		return "#22C55E" // Green
	case CustomerSegmentRegular:
		return "#3B82F6" // Blue
	case CustomerSegmentOccasional:
		return "#F59E0B" // Amber
	case CustomerSegmentAtRisk:
		return "#EF4444" // Red
	case CustomerSegmentChurned:
		return "#6B7280" // Gray
	case CustomerSegmentNew:
		return "#8B5CF6" // Purple
	default:
		return "#9CA3AF"
	}
}

// CalculateCustomerSegment determines the segment for a customer based on metrics
func CalculateCustomerSegment(totalRevenue decimal.Decimal, totalJobs int, daysSinceLastActivity int, lifetimeDays int) CustomerSegment {
	// High value customers
	if totalRevenue.GreaterThanOrEqual(decimal.NewFromInt(5000)) && totalJobs >= 5 {
		return CustomerSegmentHighValue
	}

	// New customers (less than 30 days old)
	if lifetimeDays <= 30 {
		return CustomerSegmentNew
	}

	// At risk customers (90-180 days inactive, had previous activity)
	if daysSinceLastActivity >= 90 && daysSinceLastActivity <= 180 && totalJobs >= 1 {
		return CustomerSegmentAtRisk
	}

	// Churned customers (365+ days inactive)
	if daysSinceLastActivity >= 365 {
		return CustomerSegmentChurned
	}

	// Regular customers
	if totalRevenue.GreaterThanOrEqual(decimal.NewFromInt(1000)) && totalJobs >= 2 && daysSinceLastActivity <= 180 {
		return CustomerSegmentRegular
	}

	// Occasional customers
	if totalRevenue.GreaterThanOrEqual(decimal.NewFromInt(100)) && totalJobs >= 1 {
		return CustomerSegmentOccasional
	}

	// Default: new
	return CustomerSegmentNew
}
