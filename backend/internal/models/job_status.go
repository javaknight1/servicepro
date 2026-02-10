package models

import (
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
)

var (
	// ErrInvalidStatusTransition indicates an invalid status transition
	ErrInvalidStatusTransition = errors.New("invalid status transition")

	// ErrStatusValidation indicates a status validation error
	ErrStatusValidation = errors.New("status validation error")

	// ErrMissingRequiredField indicates a required field is missing
	ErrMissingRequiredField = errors.New("missing required field")
)

// StatusTransitionReason represents the reason for status change
type StatusTransitionReason string

const (
	ReasonStartWork       StatusTransitionReason = "start_work"
	ReasonCompleteWork    StatusTransitionReason = "complete_work"
	ReasonCustomerRequest StatusTransitionReason = "customer_request"
	ReasonTechnicalIssue  StatusTransitionReason = "technical_issue"
	ReasonScheduleChange  StatusTransitionReason = "schedule_change"
	ReasonCancellation    StatusTransitionReason = "cancellation"
	ReasonResume          StatusTransitionReason = "resume"
	ReasonOther           StatusTransitionReason = "other"
)

// JobStatusTransition represents a status change event
type JobStatusTransition struct {
	ID             uuid.UUID              `json:"id" gorm:"type:uuid;primary_key;default:uuid_generate_v4()"`
	JobID          uuid.UUID              `json:"job_id" gorm:"type:uuid;not null;index"`
	Job            *Job                   `json:"job,omitempty" gorm:"foreignKey:JobID;constraint:OnDelete:CASCADE"`
	FromStatus     JobStatus              `json:"from_status" gorm:"type:varchar(50);not null"`
	ToStatus       JobStatus              `json:"to_status" gorm:"type:varchar(50);not null;index"`
	Reason         StatusTransitionReason `json:"reason" gorm:"type:varchar(50)"`
	Notes          *string                `json:"notes,omitempty" gorm:"type:text"`
	ChangedBy      uuid.UUID              `json:"changed_by" gorm:"type:uuid;not null"`
	ChangedByUser  *User                  `json:"changed_by_user,omitempty" gorm:"foreignKey:ChangedBy;constraint:OnDelete:RESTRICT"`
	TransitionedAt int64                  `json:"transitioned_at" gorm:"type:bigint;not null;index"`
	CreatedAt      time.Time              `json:"created_at"`
}

// TableName specifies the table name for JobStatusTransition model
func (JobStatusTransition) TableName() string {
	return "job_status_transitions"
}

// StatusTransitionRule defines valid state transitions
type StatusTransitionRule struct {
	From         JobStatus
	To           JobStatus
	RequiresNote bool
	ValidateFunc func(*Job) error
}

// GetAllValidStatuses returns all valid job statuses
func GetAllValidStatuses() []JobStatus {
	return []JobStatus{
		JobStatusNew,
		JobStatusScheduled,
		JobStatusEnRoute,
		JobStatusInProgress,
		JobStatusOnHold,
		JobStatusCompleted,
		JobStatusInvoiced,
		JobStatusPaid,
		JobStatusCancelled,
	}
}

// IsValidStatus checks if a status is valid
func IsValidStatus(status JobStatus) bool {
	for _, validStatus := range GetAllValidStatuses() {
		if status == validStatus {
			return true
		}
	}
	return false
}

// GetStatusTransitionRules returns the state machine transition rules
func GetStatusTransitionRules() []StatusTransitionRule {
	return []StatusTransitionRule{
		// From New
		{
			From:         JobStatusNew,
			To:           JobStatusScheduled,
			RequiresNote: false,
			ValidateFunc: nil,
		},
		{
			From:         JobStatusNew,
			To:           JobStatusCancelled,
			RequiresNote: true,
			ValidateFunc: nil,
		},

		// From Scheduled
		{
			From:         JobStatusScheduled,
			To:           JobStatusEnRoute,
			RequiresNote: false,
			ValidateFunc: nil,
		},
		{
			From:         JobStatusScheduled,
			To:           JobStatusInProgress,
			RequiresNote: false,
			ValidateFunc: nil,
		},
		{
			From:         JobStatusScheduled,
			To:           JobStatusNew,
			RequiresNote: false,
			ValidateFunc: nil, // Backward transition
		},
		{
			From:         JobStatusScheduled,
			To:           JobStatusCancelled,
			RequiresNote: true,
			ValidateFunc: nil,
		},

		// From EnRoute
		{
			From:         JobStatusEnRoute,
			To:           JobStatusInProgress,
			RequiresNote: false,
			ValidateFunc: nil,
		},
		{
			From:         JobStatusEnRoute,
			To:           JobStatusScheduled,
			RequiresNote: false,
			ValidateFunc: nil, // Backward transition
		},
		{
			From:         JobStatusEnRoute,
			To:           JobStatusCancelled,
			RequiresNote: true,
			ValidateFunc: nil,
		},

		// From InProgress
		{
			From:         JobStatusInProgress,
			To:           JobStatusOnHold,
			RequiresNote: true,
			ValidateFunc: nil,
		},
		{
			From:         JobStatusInProgress,
			To:           JobStatusCompleted,
			RequiresNote: false,
			ValidateFunc: nil,
		},
		{
			From:         JobStatusInProgress,
			To:           JobStatusEnRoute,
			RequiresNote: false,
			ValidateFunc: nil, // Backward transition
		},
		{
			From:         JobStatusInProgress,
			To:           JobStatusScheduled,
			RequiresNote: false,
			ValidateFunc: nil, // Backward transition
		},
		{
			From:         JobStatusInProgress,
			To:           JobStatusCancelled,
			RequiresNote: true,
			ValidateFunc: nil,
		},

		// From OnHold
		{
			From:         JobStatusOnHold,
			To:           JobStatusInProgress,
			RequiresNote: false,
			ValidateFunc: nil,
		},
		{
			From:         JobStatusOnHold,
			To:           JobStatusCancelled,
			RequiresNote: true,
			ValidateFunc: nil,
		},

		// From Completed
		{
			From:         JobStatusCompleted,
			To:           JobStatusInvoiced,
			RequiresNote: false,
			ValidateFunc: nil,
		},
		{
			From:         JobStatusCompleted,
			To:           JobStatusInProgress,
			RequiresNote: false,
			ValidateFunc: nil, // Backward transition
		},
		{
			From:         JobStatusCompleted,
			To:           JobStatusCancelled,
			RequiresNote: true,
			ValidateFunc: nil,
		},

		// From Invoiced
		{
			From:         JobStatusInvoiced,
			To:           JobStatusPaid,
			RequiresNote: false,
			ValidateFunc: nil,
		},
		{
			From:         JobStatusInvoiced,
			To:           JobStatusCompleted,
			RequiresNote: false,
			ValidateFunc: nil, // Backward transition
		},

		// From Paid
		{
			From:         JobStatusPaid,
			To:           JobStatusInvoiced,
			RequiresNote: false,
			ValidateFunc: nil, // Backward transition (e.g., refund scenario)
		},

		// From Cancelled (terminal - no outgoing transitions)
	}
}

// GetNextLogicalStatus returns the next forward status in the workflow
func GetNextLogicalStatus(current JobStatus) JobStatus {
	switch current {
	case JobStatusNew:
		return JobStatusScheduled
	case JobStatusScheduled:
		return JobStatusInProgress // Skip en_route by default
	case JobStatusEnRoute:
		return JobStatusInProgress
	case JobStatusInProgress:
		return JobStatusCompleted
	case JobStatusOnHold:
		return JobStatusInProgress
	case JobStatusCompleted:
		return JobStatusInvoiced
	case JobStatusInvoiced:
		return JobStatusPaid
	default:
		return "" // No next status for paid or cancelled
	}
}

// CanTransition checks if a status transition is valid
func CanTransition(from, to JobStatus) bool {
	// Same status is always valid (no-op)
	if from == to {
		return true
	}

	rules := GetStatusTransitionRules()
	for _, rule := range rules {
		if rule.From == from && rule.To == to {
			return true
		}
	}
	return false
}

// GetTransitionRule returns the rule for a specific transition
func GetTransitionRule(from, to JobStatus) (*StatusTransitionRule, error) {
	rules := GetStatusTransitionRules()
	for _, rule := range rules {
		if rule.From == from && rule.To == to {
			return &rule, nil
		}
	}
	return nil, fmt.Errorf("%w: cannot transition from %s to %s", ErrInvalidStatusTransition, from, to)
}

// ValidateTransition validates a status transition with job context
func ValidateTransition(job *Job, toStatus JobStatus, notes *string) error {
	// Check if status is valid
	if !IsValidStatus(toStatus) {
		return fmt.Errorf("%w: invalid status '%s'", ErrStatusValidation, toStatus)
	}

	// Check if transition is allowed
	if !CanTransition(job.Status, toStatus) {
		return fmt.Errorf("%w: cannot transition from %s to %s", ErrInvalidStatusTransition, job.Status, toStatus)
	}

	// No-op transitions are always valid
	if job.Status == toStatus {
		return nil
	}

	// Get the transition rule
	rule, err := GetTransitionRule(job.Status, toStatus)
	if err != nil {
		return err
	}

	// Check if notes are required
	if rule.RequiresNote && (notes == nil || *notes == "") {
		return fmt.Errorf("%w: notes are required for this transition", ErrStatusValidation)
	}

	// Execute custom validation function
	if rule.ValidateFunc != nil {
		if err := rule.ValidateFunc(job); err != nil {
			return err
		}
	}

	return nil
}

// GetAllowedTransitions returns all allowed status transitions from current status
func GetAllowedTransitions(currentStatus JobStatus) []JobStatus {
	var allowed []JobStatus
	rules := GetStatusTransitionRules()

	for _, rule := range rules {
		if rule.From == currentStatus {
			allowed = append(allowed, rule.To)
		}
	}

	return allowed
}

// IsTerminalStatus checks if a status is terminal (no outgoing transitions)
func IsTerminalStatus(status JobStatus) bool {
	// Only cancelled is truly terminal - paid can go back to invoiced
	return status == JobStatusCancelled
}

// IsFinalStatus checks if a status is at the end of the workflow (paid or cancelled)
func IsFinalStatus(status JobStatus) bool {
	return status == JobStatusPaid || status == JobStatusCancelled
}

// StatusChangeRequest represents a request to change job status
type StatusChangeRequest struct {
	JobID    uuid.UUID              `json:"job_id"`
	ToStatus JobStatus              `json:"to_status" binding:"required,oneof=new scheduled en_route in_progress on_hold completed invoiced paid cancelled"`
	Reason   StatusTransitionReason `json:"reason" binding:"omitempty,oneof=start_work complete_work customer_request technical_issue schedule_change cancellation resume other"`
	Notes    *string                `json:"notes,omitempty"`
}

// StatusHistoryRequest represents a request for status history
type StatusHistoryRequest struct {
	JobID     uuid.UUID  `form:"job_id" json:"job_id,omitempty"`
	FromDate  *time.Time `form:"from_date" json:"from_date,omitempty"`
	ToDate    *time.Time `form:"to_date" json:"to_date,omitempty"`
	ChangedBy *uuid.UUID `form:"changed_by" json:"changed_by,omitempty"`
	Limit     *int       `form:"limit" json:"limit,omitempty"`
	Offset    *int       `form:"offset" json:"offset,omitempty"`
}

// StatusChangeResponse represents the response for a status change
type StatusChangeResponse struct {
	JobID          uuid.UUID              `json:"job_id"`
	JobNumber      string                 `json:"job_number"`
	PreviousStatus JobStatus              `json:"previous_status"`
	NewStatus      JobStatus              `json:"new_status"`
	Reason         StatusTransitionReason `json:"reason,omitempty"`
	Notes          *string                `json:"notes,omitempty"`
	ChangedBy      uuid.UUID              `json:"changed_by"`
	ChangedAt      int64                  `json:"changed_at"`
}

// StatusHistoryResponse represents status history for a job
type StatusHistoryResponse struct {
	Transitions []StatusTransitionResponse `json:"transitions"`
	Total       int64                      `json:"total"`
}

// StatusTransitionResponse represents a single status transition
type StatusTransitionResponse struct {
	ID             uuid.UUID              `json:"id"`
	JobID          uuid.UUID              `json:"job_id"`
	JobNumber      string                 `json:"job_number"`
	FromStatus     JobStatus              `json:"from_status"`
	ToStatus       JobStatus              `json:"to_status"`
	Reason         StatusTransitionReason `json:"reason,omitempty"`
	Notes          *string                `json:"notes,omitempty"`
	ChangedBy      uuid.UUID              `json:"changed_by"`
	ChangedByName  string                 `json:"changed_by_name"`
	TransitionedAt int64                  `json:"transitioned_at"`
}

// AllowedTransitionsResponse represents allowed transitions for a job
type AllowedTransitionsResponse struct {
	JobID              uuid.UUID   `json:"job_id"`
	CurrentStatus      JobStatus   `json:"current_status"`
	AllowedTransitions []JobStatus `json:"allowed_transitions"`
	IsTerminal         bool        `json:"is_terminal"`
}

// ToResponse converts JobStatusTransition to response
func (t *JobStatusTransition) ToResponse() StatusTransitionResponse {
	response := StatusTransitionResponse{
		ID:             t.ID,
		JobID:          t.JobID,
		FromStatus:     t.FromStatus,
		ToStatus:       t.ToStatus,
		Reason:         t.Reason,
		Notes:          t.Notes,
		ChangedBy:      t.ChangedBy,
		TransitionedAt: t.TransitionedAt,
	}

	if t.Job != nil {
		response.JobNumber = t.Job.JobNumber
	}

	if t.ChangedByUser != nil {
		firstName := ""
		lastName := ""
		if t.ChangedByUser.FirstName != nil {
			firstName = *t.ChangedByUser.FirstName
		}
		if t.ChangedByUser.LastName != nil {
			lastName = *t.ChangedByUser.LastName
		}
		response.ChangedByName = firstName + " " + lastName
	}

	return response
}
