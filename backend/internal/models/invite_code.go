package models

import (
	"time"

	"github.com/google/uuid"
)

// InviteCode is a platform invite code that allows registration when
// REGISTRATION_MODE=invite_only. Only the SHA-256 hash of the code is stored.
type InviteCode struct {
	ID        uuid.UUID  `json:"id" gorm:"type:uuid;primary_key;default:uuid_generate_v4()"`
	CodeHash  string     `json:"-" gorm:"type:varchar(64);uniqueIndex;not null"`
	MaxUses   int        `json:"max_uses" gorm:"not null;default:1"` // 0 = unlimited
	Uses      int        `json:"uses" gorm:"not null;default:0"`
	ExpiresAt *time.Time `json:"expires_at,omitempty"`
	RevokedAt *time.Time `json:"revoked_at,omitempty"`
	Note      *string    `json:"note,omitempty" gorm:"type:varchar(255)"`
	CreatedBy *uuid.UUID `json:"created_by,omitempty" gorm:"type:uuid"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
}

// TableName specifies the table name for InviteCode
func (InviteCode) TableName() string { return "invite_codes" }

// CreateInviteCodeRequest is the body of POST /admin/invite-codes.
type CreateInviteCodeRequest struct {
	// MaxUses is how many registrations the code allows (0 = unlimited). Default 1.
	MaxUses *int `json:"max_uses,omitempty" binding:"omitempty,min=0,max=10000"`
	// ExpiresInHours sets an expiry relative to now (0 = never). Default 168 (7 days).
	ExpiresInHours *int    `json:"expires_in_hours,omitempty" binding:"omitempty,min=0,max=8760"`
	Note           *string `json:"note,omitempty" binding:"omitempty,max=255"`
}

// CreateInviteCodeResponse returns the plaintext code exactly once.
type CreateInviteCodeResponse struct {
	Code       string     `json:"code"`
	InviteCode InviteCode `json:"invite_code"`
}
