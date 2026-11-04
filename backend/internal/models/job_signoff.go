package models

import (
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

// JobSignoff is the customer's sign-off on a visit (T176): who signed, the
// signature image (in the private object store under SignatureKey), when and
// from which device, and the service report PDF built from the job (under
// ReportKey once generated) with when and to whom it was emailed.
// Tenant-owned; one per job.
type JobSignoff struct {
	ID                   uuid.UUID  `gorm:"type:uuid;primary_key;default:uuid_generate_v4()"`
	TenantID             uuid.UUID  `gorm:"type:uuid;not null"`
	JobID                uuid.UUID  `gorm:"type:uuid;not null"`
	SignerName           string     `gorm:"type:varchar(100);not null"`
	SignerTitle          *string    `gorm:"type:varchar(100)"`
	SignerEmail          *string    `gorm:"type:varchar(255)"`
	SignatureKey         string     `gorm:"type:varchar(500);not null"`
	SignatureWidth       int        `gorm:"not null"`
	SignatureHeight      int        `gorm:"not null"`
	SignedAt             time.Time  `gorm:"not null"`
	IPAddress            *string    `gorm:"type:varchar(45)"`
	UserAgent            *string    `gorm:"type:varchar(500)"`
	CapturedBy           *uuid.UUID `gorm:"type:uuid"`
	Capturer             *User      `gorm:"foreignKey:CapturedBy"`
	ReportKey            *string    `gorm:"type:varchar(500)"`
	ReportGeneratedAt    *time.Time
	ReportEmailClaimedAt *time.Time
	ReportEmailedAt      *time.Time
	// ReportRecipients is the comma-separated list the report went to.
	ReportRecipients *string
	ReportEmailCount int `gorm:"not null;default:0"`
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

// TableName specifies the table name for JobSignoff.
func (JobSignoff) TableName() string { return "job_signoffs" }

// Recipients splits ReportRecipients.
func (s *JobSignoff) Recipients() []string {
	if s.ReportRecipients == nil || *s.ReportRecipients == "" {
		return nil
	}
	return strings.Split(*s.ReportRecipients, ",")
}

// CaptureJobSignoffRequest is POST /jobs/{id}/signoff: the signer and the
// signature drawn on the technician's phone, as a PNG (a data URL from
// canvas.toDataURL("image/png") or bare base64).
type CaptureJobSignoffRequest struct {
	SignerName  string  `json:"signer_name" binding:"required,max=100" example:"Ray Burke"`
	SignerTitle *string `json:"signer_title" binding:"omitempty,max=100" example:"Site manager"`
	// SignerEmail, when given, also receives the service report.
	SignerEmail *string `json:"signer_email" binding:"omitempty,email,max=255" example:"ray.burke@example.com"`
	// Signature is the PNG, at most 512 KB, 3000 px a side.
	Signature string `json:"signature" binding:"required" example:"data:image/png;base64,iVBORw0KGgo..."`
}

// EmailJobReportRequest is POST /jobs/{id}/signoff/email.
type EmailJobReportRequest struct {
	// Resend sends the report again after it has been emailed. Without it a
	// repeated request sends nothing and returns the sign-off as it is.
	Resend bool `json:"resend"`
}

// JobSignoffResponse is a job's sign-off as the API returns it.
type JobSignoffResponse struct {
	ID          uuid.UUID `json:"id"`
	JobID       uuid.UUID `json:"job_id"`
	SignerName  string    `json:"signer_name" example:"Ray Burke"`
	SignerTitle *string   `json:"signer_title,omitempty" example:"Site manager"`
	SignerEmail *string   `json:"signer_email,omitempty" example:"ray.burke@example.com"`
	SignedAt    time.Time `json:"signed_at"`
	// CapturedByName is the user whose phone the customer signed on.
	CapturedBy     *uuid.UUID `json:"captured_by,omitempty"`
	CapturedByName string     `json:"captured_by_name,omitempty" example:"Tasha Green"`
	// SignatureURL is a presigned link to the PNG valid for about an hour;
	// SignaturePath redirects (302) to a fresh one.
	SignatureURL  string `json:"signature_url"`
	SignaturePath string `json:"signature_path" example:"/api/v1/jobs/7b0c.../signoff/signature"`
	// ReportPath redirects (302) to the service report PDF.
	ReportPath        string     `json:"report_path" example:"/api/v1/jobs/7b0c.../signoff/report"`
	ReportGeneratedAt *time.Time `json:"report_generated_at,omitempty"`
	// ReportEmailedAt is when the report was last emailed; Recipients who to.
	ReportEmailedAt  *time.Time `json:"report_emailed_at,omitempty"`
	ReportRecipients []string   `json:"report_recipients"`
	ReportEmailCount int        `json:"report_email_count"`
	// SuggestedRecipients are who the report would go to now: the
	// customer's email, the site contact and the signer (opted-out
	// customers get none).
	SuggestedRecipients []string `json:"suggested_recipients"`
	// Locked: the report has been emailed, so signing again is refused.
	Locked bool `json:"locked"`
}

// JobSignoffEnvelope is GET /jobs/{id}/signoff: the sign-off, or null when
// the job hasn't been signed, plus who the report would go to.
type JobSignoffEnvelope struct {
	Signoff             *JobSignoffResponse `json:"signoff"`
	SuggestedRecipients []string            `json:"suggested_recipients"`
}

// EmailJobReportResponse is POST /jobs/{id}/signoff/email.
type EmailJobReportResponse struct {
	// Sent is false when the report had already been emailed and resend
	// wasn't asked for (a retried request sends nothing).
	Sent    bool               `json:"sent"`
	Signoff JobSignoffResponse `json:"signoff"`
}

// ToResponse builds the response without links; the service fills them.
func (s *JobSignoff) ToResponse() JobSignoffResponse {
	r := JobSignoffResponse{
		ID: s.ID, JobID: s.JobID, SignerName: s.SignerName, SignerTitle: s.SignerTitle,
		SignerEmail: s.SignerEmail, SignedAt: s.SignedAt, CapturedBy: s.CapturedBy,
		SignaturePath:     fmt.Sprintf("/api/v1/jobs/%s/signoff/signature", s.JobID),
		ReportPath:        fmt.Sprintf("/api/v1/jobs/%s/signoff/report", s.JobID),
		ReportGeneratedAt: s.ReportGeneratedAt, ReportEmailedAt: s.ReportEmailedAt,
		ReportRecipients: s.Recipients(), ReportEmailCount: s.ReportEmailCount,
		Locked: s.ReportEmailedAt != nil,
	}
	if r.ReportRecipients == nil {
		r.ReportRecipients = []string{}
	}
	if s.Capturer != nil {
		r.CapturedByName = UserDisplayName(s.Capturer)
	}
	return r
}

// UserDisplayName is "First Last", or the email when the user has no name.
func UserDisplayName(u *User) string {
	if u == nil {
		return ""
	}
	var parts []string
	if u.FirstName != nil && strings.TrimSpace(*u.FirstName) != "" {
		parts = append(parts, strings.TrimSpace(*u.FirstName))
	}
	if u.LastName != nil && strings.TrimSpace(*u.LastName) != "" {
		parts = append(parts, strings.TrimSpace(*u.LastName))
	}
	if len(parts) == 0 {
		return u.Email
	}
	return strings.Join(parts, " ")
}
