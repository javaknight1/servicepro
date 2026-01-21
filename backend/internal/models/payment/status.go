package payment

import (
	"database/sql/driver"
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// PaymentStatus represents the current status of a payment
type PaymentStatus string

// Payment status constants
const (
	StatusPending           PaymentStatus = "pending"
	StatusAwaitingPayment   PaymentStatus = "awaiting_payment"
	StatusPaymentReceived   PaymentStatus = "payment_received"
	StatusProcessing        PaymentStatus = "processing"
	StatusAuthorized        PaymentStatus = "authorized"
	StatusCaptured          PaymentStatus = "captured"
	StatusPartiallyPaid     PaymentStatus = "partially_paid"
	StatusPaid              PaymentStatus = "paid"
	StatusSucceeded         PaymentStatus = "succeeded"
	StatusFailed            PaymentStatus = "failed"
	StatusDeclined          PaymentStatus = "declined"
	StatusExpired           PaymentStatus = "expired"
	StatusCanceled          PaymentStatus = "canceled"
	StatusRefundRequested   PaymentStatus = "refund_requested"
	StatusRefundProcessing  PaymentStatus = "refund_processing"
	StatusPartiallyRefunded PaymentStatus = "partially_refunded"
	StatusRefunded          PaymentStatus = "refunded"
	StatusDisputed          PaymentStatus = "disputed"
	StatusChargedBack       PaymentStatus = "charged_back"
	StatusOnHold            PaymentStatus = "on_hold"
	StatusUnderReview       PaymentStatus = "under_review"
	StatusRequiresAction    PaymentStatus = "requires_action"
)

// Valid status values
var validStatuses = map[PaymentStatus]bool{
	StatusPending:           true,
	StatusAwaitingPayment:   true,
	StatusPaymentReceived:   true,
	StatusProcessing:        true,
	StatusAuthorized:        true,
	StatusCaptured:          true,
	StatusPartiallyPaid:     true,
	StatusPaid:              true,
	StatusSucceeded:         true,
	StatusFailed:            true,
	StatusDeclined:          true,
	StatusExpired:           true,
	StatusCanceled:          true,
	StatusRefundRequested:   true,
	StatusRefundProcessing:  true,
	StatusPartiallyRefunded: true,
	StatusRefunded:          true,
	StatusDisputed:          true,
	StatusChargedBack:       true,
	StatusOnHold:            true,
	StatusUnderReview:       true,
	StatusRequiresAction:    true,
}

// IsValid checks if the status is valid
func (s PaymentStatus) IsValid() bool {
	return validStatuses[s]
}

// String returns the string representation of the status
func (s PaymentStatus) String() string {
	return string(s)
}

// Scan implements the sql.Scanner interface
func (s *PaymentStatus) Scan(value interface{}) error {
	if value == nil {
		*s = StatusPending
		return nil
	}

	str, ok := value.(string)
	if !ok {
		bytes, ok := value.([]byte)
		if !ok {
			return fmt.Errorf("failed to scan PaymentStatus: %v", value)
		}
		str = string(bytes)
	}

	*s = PaymentStatus(str)
	return nil
}

// Value implements the driver.Valuer interface
func (s PaymentStatus) Value() (driver.Value, error) {
	return string(s), nil
}

// StatusCategory represents the category of a payment status
type StatusCategory string

const (
	CategoryInitial    StatusCategory = "initial"
	CategoryInProgress StatusCategory = "in_progress"
	CategorySuccess    StatusCategory = "success"
	CategoryFailure    StatusCategory = "failure"
	CategoryRefund     StatusCategory = "refund"
	CategoryDispute    StatusCategory = "dispute"
	CategoryOnHold     StatusCategory = "on_hold"
)

// GetCategory returns the category for a payment status
func (s PaymentStatus) GetCategory() StatusCategory {
	switch s {
	case StatusPending, StatusAwaitingPayment:
		return CategoryInitial
	case StatusPaymentReceived, StatusProcessing, StatusAuthorized,
		StatusCaptured, StatusPartiallyPaid, StatusRequiresAction:
		return CategoryInProgress
	case StatusPaid, StatusSucceeded:
		return CategorySuccess
	case StatusFailed, StatusDeclined, StatusExpired, StatusCanceled:
		return CategoryFailure
	case StatusRefundRequested, StatusRefundProcessing,
		StatusPartiallyRefunded, StatusRefunded:
		return CategoryRefund
	case StatusDisputed, StatusChargedBack:
		return CategoryDispute
	case StatusOnHold, StatusUnderReview:
		return CategoryOnHold
	default:
		return CategoryInitial
	}
}

// IsFinal returns true if the status is a final state
func (s PaymentStatus) IsFinal() bool {
	switch s {
	case StatusSucceeded, StatusPaid, StatusFailed, StatusDeclined,
		StatusExpired, StatusCanceled, StatusRefunded, StatusChargedBack:
		return true
	default:
		return false
	}
}

// CanTransitionTo checks if a status can transition to another status
func (s PaymentStatus) CanTransitionTo(target PaymentStatus) bool {
	// Define valid state transitions
	validTransitions := map[PaymentStatus][]PaymentStatus{
		StatusPending: {
			StatusAwaitingPayment,
			StatusPaymentReceived,
			StatusProcessing,
			StatusCanceled,
			StatusExpired,
			StatusFailed,
		},
		StatusAwaitingPayment: {
			StatusPaymentReceived,
			StatusProcessing,
			StatusCanceled,
			StatusExpired,
			StatusFailed,
		},
		StatusPaymentReceived: {
			StatusProcessing,
			StatusAuthorized,
			StatusFailed,
			StatusOnHold,
		},
		StatusProcessing: {
			StatusAuthorized,
			StatusCaptured,
			StatusPaid,
			StatusSucceeded,
			StatusFailed,
			StatusOnHold,
			StatusUnderReview,
			StatusRequiresAction,
		},
		StatusAuthorized: {
			StatusCaptured,
			StatusPaid,
			StatusSucceeded,
			StatusExpired,
			StatusCanceled,
		},
		StatusCaptured: {
			StatusPaid,
			StatusSucceeded,
			StatusRefundRequested,
			StatusDisputed,
		},
		StatusPartiallyPaid: {
			StatusPaid,
			StatusSucceeded,
			StatusFailed,
			StatusCanceled,
		},
		StatusPaid: {
			StatusRefundRequested,
			StatusDisputed,
			StatusSucceeded,
		},
		StatusSucceeded: {
			StatusRefundRequested,
			StatusDisputed,
			StatusPartiallyRefunded,
			StatusRefunded,
		},
		StatusRefundRequested: {
			StatusRefundProcessing,
			StatusFailed,
		},
		StatusRefundProcessing: {
			StatusPartiallyRefunded,
			StatusRefunded,
			StatusFailed,
		},
		StatusPartiallyRefunded: {
			StatusRefundRequested,
			StatusRefunded,
		},
		StatusDisputed: {
			StatusSucceeded,
			StatusChargedBack,
			StatusRefunded,
		},
		StatusOnHold: {
			StatusProcessing,
			StatusUnderReview,
			StatusCanceled,
		},
		StatusUnderReview: {
			StatusProcessing,
			StatusSucceeded,
			StatusFailed,
			StatusCanceled,
		},
		StatusRequiresAction: {
			StatusProcessing,
			StatusSucceeded,
			StatusFailed,
		},
		// Final states can only transition to specific states
		StatusFailed:      {},
		StatusDeclined:    {},
		StatusExpired:     {},
		StatusCanceled:    {},
		StatusRefunded:    {},
		StatusChargedBack: {},
	}

	allowedTransitions, exists := validTransitions[s]
	if !exists {
		return false
	}

	for _, allowed := range allowedTransitions {
		if allowed == target {
			return true
		}
	}

	return false
}

// StatusMetadata contains additional information about a status
type StatusMetadata struct {
	Reason          string            `json:"reason,omitempty"`
	ErrorCode       string            `json:"error_code,omitempty"`
	ErrorMessage    string            `json:"error_message,omitempty"`
	ProviderStatus  string            `json:"provider_status,omitempty"`
	ProviderMessage string            `json:"provider_message,omitempty"`
	RefundAmount    float64           `json:"refund_amount,omitempty"`
	DisputeReason   string            `json:"dispute_reason,omitempty"`
	ReviewNotes     string            `json:"review_notes,omitempty"`
	CustomFields    map[string]string `json:"custom_fields,omitempty"`
}

// PaymentStatusRecord represents the current status of a payment
type PaymentStatusRecord struct {
	ID             uuid.UUID      `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	PaymentID      uuid.UUID      `gorm:"type:uuid;not null;unique;index" json:"payment_id"`
	Status         PaymentStatus  `gorm:"type:varchar(50);not null;index" json:"status"`
	PreviousStatus *PaymentStatus `gorm:"type:varchar(50)" json:"previous_status,omitempty"`
	StatusCategory StatusCategory `gorm:"type:varchar(20);not null;index" json:"status_category"`
	Metadata       StatusMetadata `gorm:"type:jsonb" json:"metadata,omitempty"`
	UpdatedBy      *uuid.UUID     `gorm:"type:uuid" json:"updated_by,omitempty"`
	UpdatedAt      time.Time      `gorm:"autoUpdateTime" json:"updated_at"`
	CreatedAt      time.Time      `gorm:"autoCreateTime" json:"created_at"`
	Version        int            `gorm:"not null;default:1" json:"version"` // For optimistic locking
}

// TableName overrides the table name
func (PaymentStatusRecord) TableName() string {
	return "payment_status"
}

// BeforeCreate hook
func (psr *PaymentStatusRecord) BeforeCreate(tx *gorm.DB) error {
	if psr.ID == uuid.Nil {
		psr.ID = uuid.New()
	}
	psr.StatusCategory = psr.Status.GetCategory()
	return nil
}

// BeforeUpdate hook
func (psr *PaymentStatusRecord) BeforeUpdate(tx *gorm.DB) error {
	psr.StatusCategory = psr.Status.GetCategory()
	psr.Version++
	return nil
}

// StatusTransition represents a status change request
type StatusTransition struct {
	PaymentID   uuid.UUID      `json:"payment_id" binding:"required"`
	FromStatus  PaymentStatus  `json:"from_status"`
	ToStatus    PaymentStatus  `json:"to_status" binding:"required"`
	Reason      string         `json:"reason"`
	Metadata    StatusMetadata `json:"metadata,omitempty"`
	UpdatedBy   *uuid.UUID     `json:"updated_by,omitempty"`
	Force       bool           `json:"force"` // Skip validation checks
	NotifyUser  bool           `json:"notify_user"`
	NotifyAdmin bool           `json:"notify_admin"`
}

// Validate validates the status transition
func (st *StatusTransition) Validate() error {
	if st.PaymentID == uuid.Nil {
		return fmt.Errorf("payment_id is required")
	}

	if !st.ToStatus.IsValid() {
		return fmt.Errorf("invalid target status: %s", st.ToStatus)
	}

	if !st.Force {
		if st.FromStatus != "" && !st.FromStatus.IsValid() {
			return fmt.Errorf("invalid from status: %s", st.FromStatus)
		}

		if st.FromStatus != "" && !st.FromStatus.CanTransitionTo(st.ToStatus) {
			return fmt.Errorf("invalid status transition from %s to %s", st.FromStatus, st.ToStatus)
		}
	}

	return nil
}

// StatusQuery represents a query for payment statuses
type StatusQuery struct {
	PaymentIDs    []uuid.UUID      `json:"payment_ids,omitempty"`
	Statuses      []PaymentStatus  `json:"statuses,omitempty"`
	Categories    []StatusCategory `json:"categories,omitempty"`
	UpdatedAfter  *time.Time       `json:"updated_after,omitempty"`
	UpdatedBefore *time.Time       `json:"updated_before,omitempty"`
	FinalOnly     bool             `json:"final_only"`
	NonFinalOnly  bool             `json:"non_final_only"`
	Page          int              `json:"page"`
	PageSize      int              `json:"page_size"`
}

// StatusStatistics represents statistics about payment statuses
type StatusStatistics struct {
	TotalPayments         int64                    `json:"total_payments"`
	StatusCounts          map[PaymentStatus]int64  `json:"status_counts"`
	CategoryCounts        map[StatusCategory]int64 `json:"category_counts"`
	AverageProcessingTime time.Duration            `json:"average_processing_time"`
	SuccessRate           float64                  `json:"success_rate"`
	FailureRate           float64                  `json:"failure_rate"`
	RefundRate            float64                  `json:"refund_rate"`
	DisputeRate           float64                  `json:"dispute_rate"`
	GeneratedAt           time.Time                `json:"generated_at"`
}

// StatusSummary provides a summary of a payment's status
type StatusSummary struct {
	PaymentID       uuid.UUID      `json:"payment_id"`
	CurrentStatus   PaymentStatus  `json:"current_status"`
	StatusCategory  StatusCategory `json:"status_category"`
	IsFinal         bool           `json:"is_final"`
	TransitionCount int            `json:"transition_count"`
	FirstStatusAt   time.Time      `json:"first_status_at"`
	LastStatusAt    time.Time      `json:"last_status_at"`
	TimeInStatus    time.Duration  `json:"time_in_status"`
	Metadata        StatusMetadata `json:"metadata,omitempty"`
}

// StatusHealth represents health metrics for status tracking
type StatusHealth struct {
	IsHealthy           bool      `json:"is_healthy"`
	StuckPaymentsCount  int64     `json:"stuck_payments_count"`
	OldPendingCount     int64     `json:"old_pending_count"`
	LongProcessingCount int64     `json:"long_processing_count"`
	UnnotifiedChanges   int64     `json:"unnotified_changes"`
	LastHealthCheck     time.Time `json:"last_health_check"`
	RecommendedActions  []string  `json:"recommended_actions,omitempty"`
}
