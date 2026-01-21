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
	TransitionedAt time.Time              `json:"transitioned_at" gorm:"not null;index"`
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
		JobStatusScheduled,
		JobStatusInProgress,
		JobStatusOnHold,
		JobStatusCompleted,
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
		// From Scheduled
		{
			From:         JobStatusScheduled,
			To:           JobStatusInProgress,
			RequiresNote: false,
			ValidateFunc: func(job *Job) error {
				if len(job.Assignments) == 0 {
					return fmt.Errorf("%w: job must have at least one assignment before starting", ErrStatusValidation)
				}
				return nil
			},
		},
		{
			From:         JobStatusScheduled,
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
			ValidateFunc: func(job *Job) error {
				if job.ActualStartAt == nil {
					return fmt.Errorf("%w: actual_start_at is required for completion", ErrMissingRequiredField)
				}
				return nil
			},
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

		// From Completed (no transitions allowed)
		// From Cancelled (no transitions allowed)
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
	return status == JobStatusCompleted || status == JobStatusCancelled
}

// StatusChangeRequest represents a request to change job status
type StatusChangeRequest struct {
	JobID    uuid.UUID              `json:"job_id"`
	ToStatus JobStatus              `json:"to_status" binding:"required,oneof=scheduled in_progress on_hold completed cancelled"`
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
	ChangedAt      time.Time              `json:"changed_at"`
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
	TransitionedAt time.Time              `json:"transitioned_at"`
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
		response.ChangedByName = t.ChangedByUser.Email
	}

	return response
}
