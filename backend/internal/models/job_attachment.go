package models

import (
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

// JobAttachmentKind says how an attachment is stored and shown.
type JobAttachmentKind string

const (
	// JobAttachmentPhoto is an image, re-encoded as JPEG (EXIF stripped,
	// orientation applied) with a thumbnail.
	JobAttachmentPhoto JobAttachmentKind = "photo"
	// JobAttachmentFile is a document (PDF) stored as uploaded, no thumbnail.
	JobAttachmentFile JobAttachmentKind = "file"
)

// JobAttachment is a photo or file attached to a job (T174). The bytes live
// in the private object store under ObjectKey (and ThumbKey for photos); the
// API hands out short-lived presigned links only. Tenant-owned.
type JobAttachment struct {
	ID               uuid.UUID         `gorm:"type:uuid;primary_key;default:uuid_generate_v4()"`
	TenantID         uuid.UUID         `gorm:"type:uuid;not null"`
	JobID            uuid.UUID         `gorm:"type:uuid;not null"`
	UploadedBy       *uuid.UUID        `gorm:"type:uuid"`
	Uploader         *User             `gorm:"foreignKey:UploadedBy"`
	Kind             JobAttachmentKind `gorm:"type:varchar(20);not null"`
	ObjectKey        string            `gorm:"type:varchar(500);not null"`
	ThumbKey         *string           `gorm:"type:varchar(500)"`
	OriginalFilename string            `gorm:"type:varchar(255);not null"`
	ContentType      string            `gorm:"type:varchar(100);not null"`
	SizeBytes        int64             `gorm:"not null"`
	Width            *int
	Height           *int
	Caption          *string `gorm:"type:varchar(500)"`
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

// TableName specifies the table name for JobAttachment.
func (JobAttachment) TableName() string { return "job_attachments" }

// JobAttachmentResponse is one attachment as the API returns it.
type JobAttachmentResponse struct {
	ID    uuid.UUID         `json:"id"`
	JobID uuid.UUID         `json:"job_id"`
	Kind  JobAttachmentKind `json:"kind" enums:"photo,file" example:"photo"`
	// Filename is the name the file had on the uploader's device.
	Filename    string  `json:"filename" example:"IMG_2041.jpg"`
	ContentType string  `json:"content_type" example:"image/jpeg"`
	SizeBytes   int64   `json:"size_bytes" example:"412345"`
	Width       *int    `json:"width,omitempty" example:"2048"`
	Height      *int    `json:"height,omitempty" example:"1536"`
	Caption     *string `json:"caption,omitempty" example:"Lobby floor"`
	// UploadedBy is the uploader's user ID (absent once the user is deleted).
	UploadedBy     *uuid.UUID `json:"uploaded_by,omitempty"`
	UploadedByName string     `json:"uploaded_by_name,omitempty" example:"Tasha Green"`
	// URL and ThumbURL are presigned links valid for about an hour (fetch the
	// list again for fresh ones). ThumbURL is set for photos only.
	URL      string `json:"url"`
	ThumbURL string `json:"thumb_url,omitempty"`
	// FilePath and ThumbPath are API paths that redirect (302) to fresh
	// presigned links; stable enough to put in an invoice or report.
	FilePath  string    `json:"file_path" example:"/api/v1/jobs/7b0c.../attachments/1f2e.../file"`
	ThumbPath string    `json:"thumb_path,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

// JobAttachmentListResponse is GET /jobs/{id}/attachments.
type JobAttachmentListResponse struct {
	Attachments []JobAttachmentResponse `json:"attachments"`
	Total       int                     `json:"total"`
}

// UpdateJobAttachmentRequest is PATCH /jobs/{id}/attachments/{attachmentId}.
// An empty caption clears it.
type UpdateJobAttachmentRequest struct {
	Caption *string `json:"caption" binding:"omitempty,max=500" example:"Restroom 2F"`
}

// ToResponse builds the response without links; the service fills them.
func (a *JobAttachment) ToResponse() JobAttachmentResponse {
	r := JobAttachmentResponse{
		ID: a.ID, JobID: a.JobID, Kind: a.Kind, Filename: a.OriginalFilename,
		ContentType: a.ContentType, SizeBytes: a.SizeBytes, Width: a.Width, Height: a.Height,
		Caption: a.Caption, UploadedBy: a.UploadedBy, CreatedAt: a.CreatedAt,
		FilePath: fmt.Sprintf("/api/v1/jobs/%s/attachments/%s/file", a.JobID, a.ID),
	}
	if a.ThumbKey != nil {
		r.ThumbPath = fmt.Sprintf("/api/v1/jobs/%s/attachments/%s/thumb", a.JobID, a.ID)
	}
	if a.Uploader != nil {
		var parts []string
		if a.Uploader.FirstName != nil && *a.Uploader.FirstName != "" {
			parts = append(parts, *a.Uploader.FirstName)
		}
		if a.Uploader.LastName != nil && *a.Uploader.LastName != "" {
			parts = append(parts, *a.Uploader.LastName)
		}
		r.UploadedByName = strings.Join(parts, " ")
		if r.UploadedByName == "" {
			r.UploadedByName = a.Uploader.Email
		}
	}
	return r
}
