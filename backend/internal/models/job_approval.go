package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// JobApproval records the customer's approver OK-ing extra work on a job up
// to Amount (T187). Office staff record it on the approver's behalf (V1:
// there is no customer-facing approval flow yet). Tenant-owned.
type JobApproval struct {
	ID            uuid.UUID       `json:"id" gorm:"type:uuid;primary_key;default:uuid_generate_v4()"`
	TenantID      uuid.UUID       `json:"-" gorm:"type:uuid;not null;index"`
	JobID         uuid.UUID       `json:"job_id" gorm:"type:uuid;not null;index"`
	ApproverName  *string         `json:"approver_name" gorm:"type:varchar(100)"`
	ApproverEmail string          `json:"approver_email" gorm:"type:varchar(255);not null"`
	Amount        decimal.Decimal `json:"amount" gorm:"type:numeric(12,2);not null" swaggertype:"string" example:"750.00"`
	ApprovedAt    time.Time       `json:"approved_at" gorm:"not null"`
	Notes         *string         `json:"notes" gorm:"type:text"`
	RecordedBy    uuid.UUID       `json:"recorded_by" gorm:"type:uuid;not null"`
	CreatedAt     time.Time       `json:"created_at"`
}

// TableName specifies the table name for JobApproval.
func (JobApproval) TableName() string { return "job_approvals" }

// RecordJobApprovalRequest is POST /jobs/{id}/approval. Omitted approver
// fields default to the customer's billing approver; an omitted amount to
// the job's estimated cost; approved_at to now.
type RecordJobApprovalRequest struct {
	ApproverName  *string          `json:"approver_name" binding:"omitempty,max=100"`
	ApproverEmail *string          `json:"approver_email" binding:"omitempty,email,max=255"`
	Amount        *decimal.Decimal `json:"amount" swaggertype:"string" example:"750.00"`
	ApprovedAt    *time.Time       `json:"approved_at"`
	Notes         *string          `json:"notes" binding:"omitempty,max=2000"`
}

// JobApprovalListResponse is GET /jobs/{id}/approvals.
type JobApprovalListResponse struct {
	Approvals []JobApproval `json:"approvals"`
}
