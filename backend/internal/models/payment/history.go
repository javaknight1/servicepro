package payment

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// PaymentStatusHistory represents a historical record of status changes
type PaymentStatusHistory struct {
	ID             uuid.UUID       `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	PaymentID      uuid.UUID       `gorm:"type:uuid;not null;index:idx_payment_history" json:"payment_id"`
	FromStatus     *PaymentStatus  `gorm:"type:varchar(50)" json:"from_status,omitempty"`
	ToStatus       PaymentStatus   `gorm:"type:varchar(50);not null" json:"to_status"`
	FromCategory   *StatusCategory `gorm:"type:varchar(20)" json:"from_category,omitempty"`
	ToCategory     StatusCategory  `gorm:"type:varchar(20);not null" json:"to_category"`
	Reason         string          `gorm:"type:text" json:"reason,omitempty"`
	Metadata       StatusMetadata  `gorm:"type:jsonb" json:"metadata,omitempty"`
	UpdatedBy      *uuid.UUID      `gorm:"type:uuid" json:"updated_by,omitempty"`
	UpdatedByType  string          `gorm:"type:varchar(50)" json:"updated_by_type,omitempty"` // system, user, admin, webhook
	TransitionTime time.Duration   `gorm:"-" json:"transition_time,omitempty"`                // Time since last status change
	CreatedAt      time.Time       `gorm:"autoCreateTime;index:idx_created_at" json:"created_at"`

	// Computed fields
	DaysSinceChange int `gorm:"-" json:"days_since_change,omitempty"`
}

// TableName overrides the table name
func (PaymentStatusHistory) TableName() string {
	return "payment_status_history"
}

// BeforeCreate hook
func (psh *PaymentStatusHistory) BeforeCreate(tx *gorm.DB) error {
	if psh.ID == uuid.Nil {
		psh.ID = uuid.New()
	}
	psh.ToCategory = psh.ToStatus.GetCategory()
	if psh.FromStatus != nil {
		category := psh.FromStatus.GetCategory()
		psh.FromCategory = &category
	}
	return nil
}

// AfterFind hook to compute derived fields
func (psh *PaymentStatusHistory) AfterFind(tx *gorm.DB) error {
	psh.DaysSinceChange = int(time.Since(psh.CreatedAt).Hours() / 24)
	return nil
}

// StatusHistoryQuery represents a query for status history
type StatusHistoryQuery struct {
	PaymentIDs    []uuid.UUID      `json:"payment_ids,omitempty"`
	FromStatuses  []PaymentStatus  `json:"from_statuses,omitempty"`
	ToStatuses    []PaymentStatus  `json:"to_statuses,omitempty"`
	Categories    []StatusCategory `json:"categories,omitempty"`
	UpdatedBy     *uuid.UUID       `json:"updated_by,omitempty"`
	UpdatedByType string           `json:"updated_by_type,omitempty"`
	StartDate     *time.Time       `json:"start_date,omitempty"`
	EndDate       *time.Time       `json:"end_date,omitempty"`
	Limit         int              `json:"limit"`
	Offset        int              `json:"offset"`
	OrderBy       string           `json:"order_by"`  // created_at, payment_id
	OrderDir      string           `json:"order_dir"` // asc, desc
}

// StatusHistoryResponse represents a paginated response of status history
type StatusHistoryResponse struct {
	History    []PaymentStatusHistory `json:"history"`
	Total      int64                  `json:"total"`
	Page       int                    `json:"page"`
	PageSize   int                    `json:"page_size"`
	TotalPages int                    `json:"total_pages"`
}

// StatusTimeline represents a timeline of status changes for a payment
type StatusTimeline struct {
	PaymentID        uuid.UUID             `json:"payment_id"`
	CurrentStatus    PaymentStatus         `json:"current_status"`
	TotalTransitions int                   `json:"total_transitions"`
	Timeline         []StatusTimelineEntry `json:"timeline"`
	KeyMilestones    []StatusMilestone     `json:"key_milestones"`
	Duration         time.Duration         `json:"duration"` // Total time from first to latest status
}

// StatusTimelineEntry represents a single entry in the timeline
type StatusTimelineEntry struct {
	Status        PaymentStatus  `json:"status"`
	Category      StatusCategory `json:"category"`
	Timestamp     time.Time      `json:"timestamp"`
	Duration      time.Duration  `json:"duration"` // Time spent in this status
	Reason        string         `json:"reason,omitempty"`
	UpdatedBy     *uuid.UUID     `json:"updated_by,omitempty"`
	UpdatedByType string         `json:"updated_by_type,omitempty"`
	Metadata      StatusMetadata `json:"metadata,omitempty"`
}

// StatusMilestone represents a key milestone in the payment lifecycle
type StatusMilestone struct {
	Name        string        `json:"name"`
	Status      PaymentStatus `json:"status"`
	Timestamp   time.Time     `json:"timestamp"`
	Description string        `json:"description,omitempty"`
}

// StatusTransitionMetrics represents metrics for status transitions
type StatusTransitionMetrics struct {
	FromStatus      PaymentStatus `json:"from_status"`
	ToStatus        PaymentStatus `json:"to_status"`
	Count           int64         `json:"count"`
	AverageDuration time.Duration `json:"average_duration"`
	MinDuration     time.Duration `json:"min_duration"`
	MaxDuration     time.Duration `json:"max_duration"`
	SuccessRate     float64       `json:"success_rate"`
	Last24Hours     int64         `json:"last_24_hours"`
	Last7Days       int64         `json:"last_7_days"`
	Last30Days      int64         `json:"last_30_days"`
}

// StatusTransitionMatrix represents a matrix of all possible transitions
type StatusTransitionMatrix struct {
	Transitions map[string]map[string]*StatusTransitionMetrics `json:"transitions"`
	GeneratedAt time.Time                                      `json:"generated_at"`
}

// StatusAnomalyDetection represents detection of unusual status patterns
type StatusAnomalyDetection struct {
	PaymentID            uuid.UUID     `json:"payment_id"`
	AnomalyType          string        `json:"anomaly_type"`
	Severity             string        `json:"severity"` // low, medium, high, critical
	Description          string        `json:"description"`
	CurrentStatus        PaymentStatus `json:"current_status"`
	UnusualTransitions   []string      `json:"unusual_transitions,omitempty"`
	TimeInStatus         time.Duration `json:"time_in_status"`
	ExpectedTimeInStatus time.Duration `json:"expected_time_in_status"`
	DetectedAt           time.Time     `json:"detected_at"`
	RequiresAction       bool          `json:"requires_action"`
	SuggestedActions     []string      `json:"suggested_actions,omitempty"`
}

// Common anomaly types
const (
	AnomalyTypeStuck             = "stuck_in_status"
	AnomalyTypeRapidTransition   = "rapid_transition"
	AnomalyTypeInvalidTransition = "invalid_transition"
	AnomalyTypeExpired           = "expired"
	AnomalyTypeMultipleFailures  = "multiple_failures"
	AnomalyTypeUnexpectedRefund  = "unexpected_refund"
	AnomalyTypeDisputePattern    = "dispute_pattern"
)

// Severity levels
const (
	SeverityLow      = "low"
	SeverityMedium   = "medium"
	SeverityHigh     = "high"
	SeverityCritical = "critical"
)

// StatusAuditLog represents an audit log entry for status changes
type StatusAuditLog struct {
	ID              uuid.UUID      `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	PaymentID       uuid.UUID      `gorm:"type:uuid;not null;index" json:"payment_id"`
	Action          string         `gorm:"type:varchar(50);not null" json:"action"` // status_change, manual_override, system_update
	FromStatus      *PaymentStatus `gorm:"type:varchar(50)" json:"from_status,omitempty"`
	ToStatus        PaymentStatus  `gorm:"type:varchar(50);not null" json:"to_status"`
	PerformedBy     *uuid.UUID     `gorm:"type:uuid" json:"performed_by,omitempty"`
	PerformedByType string         `gorm:"type:varchar(50)" json:"performed_by_type,omitempty"`
	IPAddress       string         `gorm:"type:varchar(45)" json:"ip_address,omitempty"`
	UserAgent       string         `gorm:"type:text" json:"user_agent,omitempty"`
	Reason          string         `gorm:"type:text" json:"reason,omitempty"`
	Success         bool           `gorm:"not null;default:true" json:"success"`
	ErrorMessage    string         `gorm:"type:text" json:"error_message,omitempty"`
	Metadata        StatusMetadata `gorm:"type:jsonb" json:"metadata,omitempty"`
	CreatedAt       time.Time      `gorm:"autoCreateTime;index" json:"created_at"`
}

// TableName overrides the table name
func (StatusAuditLog) TableName() string {
	return "payment_status_audit"
}

// BeforeCreate hook
func (sal *StatusAuditLog) BeforeCreate(tx *gorm.DB) error {
	if sal.ID == uuid.Nil {
		sal.ID = uuid.New()
	}
	return nil
}

// StatusAuditQuery represents a query for audit logs
type StatusAuditQuery struct {
	PaymentIDs      []uuid.UUID `json:"payment_ids,omitempty"`
	Actions         []string    `json:"actions,omitempty"`
	PerformedBy     *uuid.UUID  `json:"performed_by,omitempty"`
	PerformedByType string      `json:"performed_by_type,omitempty"`
	StartDate       *time.Time  `json:"start_date,omitempty"`
	EndDate         *time.Time  `json:"end_date,omitempty"`
	SuccessOnly     bool        `json:"success_only"`
	FailuresOnly    bool        `json:"failures_only"`
	Limit           int         `json:"limit"`
	Offset          int         `json:"offset"`
}

// StatusReportRequest represents a request for a status report
type StatusReportRequest struct {
	PaymentIDs       []uuid.UUID      `json:"payment_ids,omitempty"`
	Statuses         []PaymentStatus  `json:"statuses,omitempty"`
	Categories       []StatusCategory `json:"categories,omitempty"`
	StartDate        *time.Time       `json:"start_date,omitempty"`
	EndDate          *time.Time       `json:"end_date,omitempty"`
	IncludeHistory   bool             `json:"include_history"`
	IncludeMetrics   bool             `json:"include_metrics"`
	IncludeAnomalies bool             `json:"include_anomalies"`
	Format           string           `json:"format"` // json, csv, pdf
}

// StatusReport represents a comprehensive status report
type StatusReport struct {
	ReportID        uuid.UUID                 `json:"report_id"`
	GeneratedAt     time.Time                 `json:"generated_at"`
	Period          DateRange                 `json:"period"`
	Summary         StatusReportSummary       `json:"summary"`
	Payments        []PaymentStatusSummary    `json:"payments,omitempty"`
	Transitions     []StatusTransitionMetrics `json:"transitions,omitempty"`
	Anomalies       []StatusAnomalyDetection  `json:"anomalies,omitempty"`
	Recommendations []string                  `json:"recommendations,omitempty"`
}

// DateRange represents a date range
type DateRange struct {
	Start time.Time `json:"start"`
	End   time.Time `json:"end"`
}

// StatusReportSummary provides summary statistics for a report
type StatusReportSummary struct {
	TotalPayments            int64                    `json:"total_payments"`
	StatusDistribution       map[PaymentStatus]int64  `json:"status_distribution"`
	CategoryDistribution     map[StatusCategory]int64 `json:"category_distribution"`
	TotalTransitions         int64                    `json:"total_transitions"`
	AvgTransitionsPerPayment float64                  `json:"avg_transitions_per_payment"`
	SuccessRate              float64                  `json:"success_rate"`
	FailureRate              float64                  `json:"failure_rate"`
	AnomaliesDetected        int64                    `json:"anomalies_detected"`
}

// PaymentStatusSummary provides a summary for a single payment
type PaymentStatusSummary struct {
	PaymentID       uuid.UUID     `json:"payment_id"`
	CurrentStatus   PaymentStatus `json:"current_status"`
	FirstStatus     PaymentStatus `json:"first_status"`
	TransitionCount int           `json:"transition_count"`
	FirstStatusAt   time.Time     `json:"first_status_at"`
	LastStatusAt    time.Time     `json:"last_status_at"`
	TotalDuration   time.Duration `json:"total_duration"`
	HasAnomalies    bool          `json:"has_anomalies"`
}
