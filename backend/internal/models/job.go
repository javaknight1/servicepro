package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// JobStatus represents the status of a job
type JobStatus string

const (
	JobStatusNew        JobStatus = "new"
	JobStatusScheduled  JobStatus = "scheduled"
	JobStatusEnRoute    JobStatus = "en_route"
	JobStatusInProgress JobStatus = "in_progress"
	JobStatusOnHold     JobStatus = "on_hold"
	JobStatusCompleted  JobStatus = "completed"
	JobStatusInvoiced   JobStatus = "invoiced"
	JobStatusPaid       JobStatus = "paid"
	JobStatusCancelled  JobStatus = "cancelled"
)

// JobPriority represents the priority level of a job
type JobPriority string

const (
	JobPriorityLow    JobPriority = "low"
	JobPriorityNormal JobPriority = "normal"
	JobPriorityHigh   JobPriority = "high"
	JobPriorityUrgent JobPriority = "urgent"
)

// JobType represents the type of job
type JobType string

const (
	JobTypeInstallation JobType = "installation"
	JobTypeMaintenance  JobType = "maintenance"
	JobTypeRepair       JobType = "repair"
	JobTypeInspection   JobType = "inspection"
	JobTypeEmergency    JobType = "emergency"
)

// Job represents a service job/work order
type Job struct {
	ID                  uuid.UUID       `json:"id" gorm:"type:uuid;primary_key;default:uuid_generate_v4()"`
	JobNumber           string          `json:"job_number" gorm:"type:varchar(50);uniqueIndex;not null"`
	CustomerID          uuid.UUID       `json:"customer_id" gorm:"type:uuid;not null;index"`
	Customer            *Customer       `json:"customer,omitempty" gorm:"foreignKey:CustomerID;constraint:OnDelete:RESTRICT"`
	Title               string          `json:"title" gorm:"type:varchar(200);not null"`
	Description         string          `json:"description" gorm:"type:text"`
	JobType             JobType         `json:"job_type" gorm:"type:varchar(50);not null;index"`
	Status              JobStatus       `json:"status" gorm:"type:varchar(50);not null;default:'scheduled';index"`
	Priority            JobPriority     `json:"priority" gorm:"type:varchar(50);not null;default:'normal';index"`
	ScheduledStartAt    *time.Time      `json:"scheduled_start_at" gorm:"index"`
	ScheduledEndAt      *time.Time      `json:"scheduled_end_at"`
	ActualStartAt       *time.Time      `json:"actual_start_at"`
	ActualEndAt         *time.Time      `json:"actual_end_at"`
	EstimatedDuration   int             `json:"estimated_duration"` // in minutes
	ActualDuration      int             `json:"actual_duration"`    // in minutes
	ServiceAddress      ServiceAddress  `json:"service_address" gorm:"embedded;embeddedPrefix:service_"`
	EstimatedCost       float64         `json:"estimated_cost" gorm:"type:decimal(10,2)"`
	ActualCost          float64         `json:"actual_cost" gorm:"type:decimal(10,2)"`
	TaxAmount           float64         `json:"tax_amount" gorm:"type:decimal(10,2)"`
	TotalAmount         float64         `json:"total_amount" gorm:"type:decimal(10,2)"`
	Assignments         []JobAssignment `json:"assignments,omitempty" gorm:"foreignKey:JobID;constraint:OnDelete:CASCADE"`
	Materials           []JobMaterial   `json:"materials,omitempty" gorm:"foreignKey:JobID;constraint:OnDelete:CASCADE"`
	Notes               []JobNote       `json:"notes,omitempty" gorm:"foreignKey:JobID;constraint:OnDelete:CASCADE"`
	InternalNotes       *string         `json:"internal_notes,omitempty" gorm:"type:text"`
	CustomerNotes       *string         `json:"customer_notes,omitempty" gorm:"type:text"`
	CompletionNotes     *string         `json:"completion_notes,omitempty" gorm:"type:text"`
	SpecialInstructions *string         `json:"special_instructions,omitempty" gorm:"type:text"`
	RequiredMaterials   *string         `json:"required_materials,omitempty" gorm:"type:text"`
	RequiresFollowUp    bool            `json:"requires_follow_up" gorm:"default:false"`
	FollowUpDate        *time.Time      `json:"follow_up_date"`
	CreatedBy           uuid.UUID       `json:"created_by" gorm:"type:uuid;not null"`
	UpdatedBy           *uuid.UUID      `json:"updated_by" gorm:"type:uuid"`
	CreatedAt           time.Time       `json:"created_at"`
	UpdatedAt           time.Time       `json:"updated_at"`
	DeletedAt           gorm.DeletedAt  `json:"-" gorm:"index"`
}

// ServiceAddress represents the address where service will be performed
type ServiceAddress struct {
	Street string `json:"street" gorm:"type:varchar(100)"`
	City   string `json:"city" gorm:"type:varchar(50)"`
	State  string `json:"state" gorm:"type:char(2)"`
	Zip    string `json:"zip" gorm:"type:varchar(10)"`
}

// TableName specifies the table name for Job model
func (Job) TableName() string {
	return "jobs"
}

// BeforeCreate generates a job number if not provided
func (j *Job) BeforeCreate(tx *gorm.DB) error {
	if j.JobNumber == "" {
		// Generate job number: JOB-YYYYMMDD-XXXX
		now := time.Now()
		var count int64
		tx.Model(&Job{}).
			Where("DATE(created_at) = ?", now.Format("2006-01-02")).
			Count(&count)

		j.JobNumber = now.Format("JOB-20060102-") + padLeft(int(count)+1, 4)
	}
	return nil
}

// JobAssignment represents assignment of technicians/employees to a job
type JobAssignment struct {
	ID           uuid.UUID      `json:"id" gorm:"type:uuid;primary_key;default:uuid_generate_v4()"`
	JobID        uuid.UUID      `json:"job_id" gorm:"type:uuid;not null;index"`
	Job          *Job           `json:"job,omitempty" gorm:"foreignKey:JobID;constraint:OnDelete:CASCADE"`
	UserID       uuid.UUID      `json:"user_id" gorm:"type:uuid;not null;index"`
	User         *User          `json:"user,omitempty" gorm:"foreignKey:UserID;constraint:OnDelete:RESTRICT"`
	Role         string         `json:"role" gorm:"type:varchar(50)"` // e.g., "lead_technician", "assistant"
	AssignedAt   time.Time      `json:"assigned_at" gorm:"not null"`
	UnassignedAt *time.Time     `json:"unassigned_at"`
	HoursWorked  float64        `json:"hours_worked" gorm:"type:decimal(5,2);default:0"`
	Notes        *string        `json:"notes,omitempty" gorm:"type:text"`
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
	DeletedAt    gorm.DeletedAt `json:"-" gorm:"index"`
}

// TableName specifies the table name for JobAssignment model
func (JobAssignment) TableName() string {
	return "job_assignments"
}

// JobMaterial represents materials/parts used in a job
type JobMaterial struct {
	ID          uuid.UUID      `json:"id" gorm:"type:uuid;primary_key;default:uuid_generate_v4()"`
	JobID       uuid.UUID      `json:"job_id" gorm:"type:uuid;not null;index"`
	Job         *Job           `json:"job,omitempty" gorm:"foreignKey:JobID;constraint:OnDelete:CASCADE"`
	Name        string         `json:"name" gorm:"type:varchar(200);not null"`
	Description *string        `json:"description,omitempty" gorm:"type:text"`
	SKU         *string        `json:"sku,omitempty" gorm:"type:varchar(100)"`
	Quantity    float64        `json:"quantity" gorm:"type:decimal(10,2);not null"`
	Unit        string         `json:"unit" gorm:"type:varchar(20)"` // e.g., "each", "ft", "gal"
	UnitCost    float64        `json:"unit_cost" gorm:"type:decimal(10,2);not null"`
	TotalCost   float64        `json:"total_cost" gorm:"type:decimal(10,2);not null"`
	Billable    bool           `json:"billable" gorm:"default:true"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `json:"-" gorm:"index"`
}

// TableName specifies the table name for JobMaterial model
func (JobMaterial) TableName() string {
	return "job_materials"
}

// BeforeSave calculates total cost
func (m *JobMaterial) BeforeSave(tx *gorm.DB) error {
	m.TotalCost = m.Quantity * m.UnitCost
	return nil
}

// JobNote represents notes/comments added to a job
type JobNote struct {
	ID         uuid.UUID      `json:"id" gorm:"type:uuid;primary_key;default:uuid_generate_v4()"`
	JobID      uuid.UUID      `json:"job_id" gorm:"type:uuid;not null;index"`
	Job        *Job           `json:"job,omitempty" gorm:"foreignKey:JobID;constraint:OnDelete:CASCADE"`
	UserID     uuid.UUID      `json:"user_id" gorm:"type:uuid;not null"`
	User       *User          `json:"user,omitempty" gorm:"foreignKey:UserID;constraint:OnDelete:RESTRICT"`
	Note       string         `json:"note" gorm:"type:text;not null"`
	IsInternal bool           `json:"is_internal" gorm:"default:false"` // Internal notes not visible to customer
	CreatedAt  time.Time      `json:"created_at"`
	UpdatedAt  time.Time      `json:"updated_at"`
	DeletedAt  gorm.DeletedAt `json:"-" gorm:"index"`
}

// TableName specifies the table name for JobNote model
func (JobNote) TableName() string {
	return "job_notes"
}

// CreateJobAssignmentRequest represents an assignment to be created with a job
type CreateJobAssignmentRequest struct {
	UserID uuid.UUID `json:"user_id" binding:"required"`
	Role   string    `json:"role" binding:"omitempty"`
}

// CreateJobRequest represents the request payload for creating a job
type CreateJobRequest struct {
	CustomerID          uuid.UUID                    `json:"customer_id" binding:"required"`
	Title               string                       `json:"title" binding:"required,max=200"`
	Description         string                       `json:"description" binding:"omitempty,max=5000"`
	JobType             JobType                      `json:"job_type" binding:"required,oneof=installation maintenance repair inspection emergency"`
	Priority            JobPriority                  `json:"priority" binding:"omitempty,oneof=low normal high urgent"`
	ScheduledStartAt    *time.Time                   `json:"scheduled_start_at"`
	ScheduledEndAt      *time.Time                   `json:"scheduled_end_at"`
	EstimatedDuration   int                          `json:"estimated_duration" binding:"omitempty,min=0"` // in minutes
	ServiceAddress      ServiceAddress               `json:"service_address"`
	EstimatedCost       float64                      `json:"estimated_cost" binding:"omitempty,min=0"`
	InternalNotes       *string                      `json:"internal_notes"`
	CustomerNotes       *string                      `json:"customer_notes"`
	SpecialInstructions *string                      `json:"special_instructions"`
	RequiredMaterials   *string                      `json:"required_materials"`
	Assignments         []CreateJobAssignmentRequest `json:"assignments" binding:"omitempty"`
}

// UpdateJobRequest represents the request payload for updating a job
type UpdateJobRequest struct {
	Title               *string         `json:"title" binding:"omitempty,max=200"`
	Description         *string         `json:"description" binding:"omitempty,max=5000"`
	JobType             *JobType        `json:"job_type" binding:"omitempty,oneof=installation maintenance repair inspection emergency"`
	Status              *JobStatus      `json:"status" binding:"omitempty,oneof=new scheduled en_route in_progress on_hold completed invoiced paid cancelled"`
	Priority            *JobPriority    `json:"priority" binding:"omitempty,oneof=low normal high urgent"`
	ScheduledStartAt    *time.Time      `json:"scheduled_start_at"`
	ScheduledEndAt      *time.Time      `json:"scheduled_end_at"`
	ActualStartAt       *time.Time      `json:"actual_start_at"`
	ActualEndAt         *time.Time      `json:"actual_end_at"`
	EstimatedDuration   *int            `json:"estimated_duration" binding:"omitempty,min=0"`
	ServiceAddress      *ServiceAddress `json:"service_address"`
	EstimatedCost       *float64        `json:"estimated_cost" binding:"omitempty,min=0"`
	ActualCost          *float64        `json:"actual_cost" binding:"omitempty,min=0"`
	TaxAmount           *float64        `json:"tax_amount" binding:"omitempty,min=0"`
	InternalNotes       *string         `json:"internal_notes"`
	CustomerNotes       *string         `json:"customer_notes"`
	CompletionNotes     *string         `json:"completion_notes"`
	SpecialInstructions *string         `json:"special_instructions"`
	RequiredMaterials   *string         `json:"required_materials"`
	RequiresFollowUp    *bool           `json:"requires_follow_up"`
	FollowUpDate        *time.Time      `json:"follow_up_date"`
}

// JobResponse represents the response payload for a job
type JobResponse struct {
	ID                  uuid.UUID               `json:"id"`
	JobNumber           string                  `json:"job_number"`
	CustomerID          uuid.UUID               `json:"customer_id"`
	Customer            *CustomerResponse       `json:"customer,omitempty"`
	Title               string                  `json:"title"`
	Description         string                  `json:"description"`
	JobType             JobType                 `json:"job_type"`
	Status              JobStatus               `json:"status"`
	Priority            JobPriority             `json:"priority"`
	ScheduledStartAt    *time.Time              `json:"scheduled_start_at,omitempty"`
	ScheduledEndAt      *time.Time              `json:"scheduled_end_at,omitempty"`
	ActualStartAt       *time.Time              `json:"actual_start_at,omitempty"`
	ActualEndAt         *time.Time              `json:"actual_end_at,omitempty"`
	EstimatedDuration   int                     `json:"estimated_duration"`
	ActualDuration      int                     `json:"actual_duration"`
	ServiceAddress      ServiceAddress          `json:"service_address"`
	EstimatedCost       float64                 `json:"estimated_cost"`
	ActualCost          float64                 `json:"actual_cost"`
	TaxAmount           float64                 `json:"tax_amount"`
	TotalAmount         float64                 `json:"total_amount"`
	Assignments         []JobAssignmentResponse `json:"assignments,omitempty"`
	Materials           []JobMaterialResponse   `json:"materials,omitempty"`
	Notes               []JobNoteResponse       `json:"notes,omitempty"`
	InternalNotes       *string                 `json:"internal_notes,omitempty"`
	CustomerNotes       *string                 `json:"customer_notes,omitempty"`
	CompletionNotes     *string                 `json:"completion_notes,omitempty"`
	SpecialInstructions *string                 `json:"special_instructions,omitempty"`
	RequiredMaterials   *string                 `json:"required_materials,omitempty"`
	RequiresFollowUp    bool                    `json:"requires_follow_up"`
	FollowUpDate        *time.Time              `json:"follow_up_date,omitempty"`
	NextStatus          *JobStatus              `json:"next_status,omitempty"`
	Warnings            []string                `json:"warnings,omitempty"`
	CreatedAt           time.Time               `json:"created_at"`
	UpdatedAt           time.Time               `json:"updated_at"`
}

// JobAssignmentResponse represents the response for a job assignment
type JobAssignmentResponse struct {
	ID           uuid.UUID  `json:"id"`
	JobID        uuid.UUID  `json:"job_id"`
	UserID       uuid.UUID  `json:"user_id"`
	UserName     string     `json:"user_name"`
	Role         string     `json:"role"`
	AssignedAt   time.Time  `json:"assigned_at"`
	UnassignedAt *time.Time `json:"unassigned_at,omitempty"`
	HoursWorked  float64    `json:"hours_worked"`
	Notes        *string    `json:"notes,omitempty"`
}

// JobMaterialResponse represents the response for a job material
type JobMaterialResponse struct {
	ID          uuid.UUID `json:"id"`
	JobID       uuid.UUID `json:"job_id"`
	Name        string    `json:"name"`
	Description *string   `json:"description,omitempty"`
	SKU         *string   `json:"sku,omitempty"`
	Quantity    float64   `json:"quantity"`
	Unit        string    `json:"unit"`
	UnitCost    float64   `json:"unit_cost"`
	TotalCost   float64   `json:"total_cost"`
	Billable    bool      `json:"billable"`
}

// JobNoteResponse represents the response for a job note
type JobNoteResponse struct {
	ID         uuid.UUID `json:"id"`
	JobID      uuid.UUID `json:"job_id"`
	UserID     uuid.UUID `json:"user_id"`
	UserName   string    `json:"user_name"`
	Note       string    `json:"note"`
	IsInternal bool      `json:"is_internal"`
	CreatedAt  time.Time `json:"created_at"`
}

// JobListFilter represents query parameters for filtering jobs
type JobListFilter struct {
	CustomerID         *uuid.UUID   `form:"customer_id" json:"customer_id,omitempty"`
	Status             *JobStatus   `form:"status" json:"status,omitempty"`
	Priority           *JobPriority `form:"priority" json:"priority,omitempty"`
	JobType            *JobType     `form:"job_type" json:"job_type,omitempty"`
	AssignedUserID     *uuid.UUID   `form:"assigned_user_id" json:"assigned_user_id,omitempty"`
	ScheduledStartFrom *time.Time   `form:"scheduled_start_from" json:"scheduled_start_from,omitempty"`
	ScheduledStartTo   *time.Time   `form:"scheduled_start_to" json:"scheduled_start_to,omitempty"`
	RequiresFollowUp   *bool        `form:"requires_follow_up" json:"requires_follow_up,omitempty"`
	SortBy             *string      `form:"sort_by" json:"sort_by,omitempty"`
	SortOrder          *string      `form:"sort_order" json:"sort_order,omitempty"`
	Limit              *int         `form:"limit" json:"limit,omitempty"`
	Offset             *int         `form:"offset" json:"offset,omitempty"`
}

// JobListResponse represents a paginated list of jobs
type JobListResponse struct {
	Jobs       []JobResponse `json:"jobs"`
	Total      int64         `json:"total"`
	Page       int           `json:"page"`
	PageSize   int           `json:"page_size"`
	TotalPages int           `json:"total_pages"`
}

// ToResponse converts a Job model to JobResponse
func (j *Job) ToResponse() JobResponse {
	response := JobResponse{
		ID:                  j.ID,
		JobNumber:           j.JobNumber,
		CustomerID:          j.CustomerID,
		Title:               j.Title,
		Description:         j.Description,
		JobType:             j.JobType,
		Status:              j.Status,
		Priority:            j.Priority,
		ScheduledStartAt:    j.ScheduledStartAt,
		ScheduledEndAt:      j.ScheduledEndAt,
		ActualStartAt:       j.ActualStartAt,
		ActualEndAt:         j.ActualEndAt,
		EstimatedDuration:   j.EstimatedDuration,
		ActualDuration:      j.ActualDuration,
		ServiceAddress:      j.ServiceAddress,
		EstimatedCost:       j.EstimatedCost,
		ActualCost:          j.ActualCost,
		TaxAmount:           j.TaxAmount,
		TotalAmount:         j.TotalAmount,
		InternalNotes:       j.InternalNotes,
		CustomerNotes:       j.CustomerNotes,
		CompletionNotes:     j.CompletionNotes,
		SpecialInstructions: j.SpecialInstructions,
		RequiredMaterials:   j.RequiredMaterials,
		RequiresFollowUp:    j.RequiresFollowUp,
		FollowUpDate:        j.FollowUpDate,
		CreatedAt:           j.CreatedAt,
		UpdatedAt:           j.UpdatedAt,
	}

	// Add next logical status
	nextStatus := GetNextLogicalStatus(j.Status)
	if nextStatus != "" {
		response.NextStatus = &nextStatus
	}

	// Include customer if loaded
	if j.Customer != nil {
		customerResp := j.Customer.ToResponse()
		response.Customer = &customerResp
	}

	// Include assignments if loaded
	if len(j.Assignments) > 0 {
		response.Assignments = make([]JobAssignmentResponse, len(j.Assignments))
		for i, assignment := range j.Assignments {
			response.Assignments[i] = JobAssignmentResponse{
				ID:           assignment.ID,
				JobID:        assignment.JobID,
				UserID:       assignment.UserID,
				Role:         assignment.Role,
				AssignedAt:   assignment.AssignedAt,
				UnassignedAt: assignment.UnassignedAt,
				HoursWorked:  assignment.HoursWorked,
				Notes:        assignment.Notes,
			}
			if assignment.User != nil {
				response.Assignments[i].UserName = assignment.User.Email
			}
		}
	}

	// Include materials if loaded
	if len(j.Materials) > 0 {
		response.Materials = make([]JobMaterialResponse, len(j.Materials))
		for i, material := range j.Materials {
			response.Materials[i] = JobMaterialResponse{
				ID:          material.ID,
				JobID:       material.JobID,
				Name:        material.Name,
				Description: material.Description,
				SKU:         material.SKU,
				Quantity:    material.Quantity,
				Unit:        material.Unit,
				UnitCost:    material.UnitCost,
				TotalCost:   material.TotalCost,
				Billable:    material.Billable,
			}
		}
	}

	// Include notes if loaded
	if len(j.Notes) > 0 {
		response.Notes = make([]JobNoteResponse, len(j.Notes))
		for i, note := range j.Notes {
			response.Notes[i] = JobNoteResponse{
				ID:         note.ID,
				JobID:      note.JobID,
				UserID:     note.UserID,
				Note:       note.Note,
				IsInternal: note.IsInternal,
				CreatedAt:  note.CreatedAt,
			}
			if note.User != nil {
				response.Notes[i].UserName = note.User.Email
			}
		}
	}

	return response
}

// Helper function to pad numbers with leading zeros
func padLeft(num, width int) string {
	s := ""
	for i := 0; i < width; i++ {
		s += "0"
	}
	s += string(rune('0' + num))
	return s[len(s)-width:]
}
