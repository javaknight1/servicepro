package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// UserRole represents user roles in the system
type UserRole string

const (
	RoleUser       UserRole = "user"
	RoleAdmin      UserRole = "admin"
	RoleManager    UserRole = "manager"
	RoleTechnician UserRole = "technician"

	// Legacy aliases for backwards compatibility
	UserRoleAdmin      = RoleAdmin
	UserRoleManager    = RoleManager
	UserRoleTechnician = RoleTechnician
)

// User represents a user in the system
type User struct {
	ID                 uuid.UUID  `json:"id" gorm:"type:uuid;primary_key;default:uuid_generate_v4()"`
	Email              string     `json:"email" gorm:"uniqueIndex;not null"`
	PasswordHash       string     `json:"-" gorm:"not null"`
	Role               UserRole   `json:"role" gorm:"type:varchar(50);not null;default:'user'"`
	FirstName          *string    `json:"first_name" gorm:"column:first_name;size:100"`
	LastName           *string    `json:"last_name" gorm:"column:last_name;size:100"`
	ProfilePictureURL  *string    `json:"profile_picture_url" gorm:"column:profile_picture_url"`
	EmailVerified      bool       `json:"email_verified" gorm:"default:false"`
	VerificationSentAt *time.Time `json:"-"`
	FailedLoginCount   int        `json:"-" gorm:"default:0"`
	LastFailedLoginAt  *time.Time `json:"-"`
	LockedUntil        *time.Time `json:"-"`
	// TokenVersion is embedded in every JWT; bumping it (logout, password
	// reset) revokes all outstanding access and refresh tokens. Read-only
	// for GORM writes: it is only changed by explicit UPDATEs.
	TokenVersion int `json:"-" gorm:"->;column:token_version"`
	// AuthUserID is the Neon Auth (managed Better Auth) user id — the `sub`
	// of its JWTs (T158). Nil until the user signs in through Neon Auth.
	AuthUserID *string `json:"-" gorm:"column:auth_user_id"`
	// TermsVersion is the version of the Terms of Service and Privacy Policy
	// accepted at sign-up (CurrentTermsVersion at the time), with the moment
	// it was accepted (T255). Nil for users created before it was recorded.
	TermsVersion    *string        `json:"-" gorm:"column:terms_version;size:32"`
	TermsAcceptedAt *time.Time     `json:"-" gorm:"column:terms_accepted_at"`
	CreatedAt       time.Time      `json:"created_at"`
	UpdatedAt       time.Time      `json:"updated_at"`
	DeletedAt       gorm.DeletedAt `json:"-" gorm:"index"`
}

// IsLocked checks if the user account is currently locked
func (u *User) IsLocked() bool {
	if u.LockedUntil == nil {
		return false
	}
	return time.Now().Before(*u.LockedUntil)
}

// LoginRequest represents the login request payload
type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=8"`
}

// LoginResponse represents the login response payload (used internally by auth service)
type LoginResponse struct {
	Token        string `json:"token"`
	RefreshToken string `json:"refreshToken"`
	ExpiresIn    int    `json:"expiresIn"` // in seconds
}

// LoginCookieResponse represents the login response when using httpOnly cookies
// Tokens are sent via cookies, not in the response body
type LoginCookieResponse struct {
	Message   string `json:"message"`
	ExpiresIn int    `json:"expiresIn"` // in seconds, for client-side expiry tracking
}

// CurrentTermsVersion identifies the Terms of Service and Privacy Policy a
// new user accepts: the date of the current legal drafts (frontend
// LEGAL_DRAFT_DATE). Bump both when the documents change (T207).
const CurrentTermsVersion = "2026-10-16"

// RegisterRequest represents the registration request payload
type RegisterRequest struct {
	Email           string  `json:"email" binding:"required,email"`
	Password        string  `json:"password" binding:"required,min=8"`
	InvitationToken *string `json:"invitation_token,omitempty"` // Optional invitation token to auto-join organization
	// InviteCode is required when REGISTRATION_MODE=invite_only and no
	// invitation token for this email is supplied.
	InviteCode *string `json:"invite_code,omitempty"`
	// AcceptTerms must be true: the user agrees to the Terms of Service and
	// Privacy Policy (CurrentTermsVersion is recorded on the user, T255).
	AcceptTerms bool `json:"accept_terms"`
}

// RegisterResponse represents the registration response payload
type RegisterResponse struct {
	ID        uuid.UUID `json:"id"`
	Email     string    `json:"email"`
	Role      UserRole  `json:"role"`
	CreatedAt time.Time `json:"createdAt"`
}

// UpdateProfileRequest represents the request to update user profile
type UpdateProfileRequest struct {
	FirstName *string `json:"first_name"`
	LastName  *string `json:"last_name"`
}

// ErrorResponse is the standard API error envelope (T223, docs/api/errors.md).
// Write it through internal/api/apierror, which fills request_id and
// normalises code/message, rather than with c.JSON directly.
type ErrorResponse struct {
	// Error is a stable snake_case machine code (e.g. "not_found", "cannot_accept").
	Error string `json:"error" example:"cannot_accept"`
	// Message is human-readable and safe to show to the user.
	Message string `json:"message,omitempty" example:"This quote can no longer be accepted"`
	// RequestID matches the X-Request-ID response header and the server's log lines.
	RequestID string `json:"request_id,omitempty" example:"3f6c1b9e-2a4d-4e7b-9a51-0c8d2f1e6b7a"`
	// Fields maps request field names to problems (validation errors only).
	Fields map[string]string `json:"fields,omitempty"`
	// Details lists extra messages, e.g. per-row import errors.
	Details []string `json:"details,omitempty"`
}

// ProfilePictureUploadResponse represents the response for profile picture upload
type ProfilePictureUploadResponse struct {
	ProfilePictureURL string `json:"profile_picture_url"`
}

// NeonProvisionRequest is the optional body of POST /auth/neon/provision.
// Only a brand-new user needs one of the two (invite-only registration).
type NeonProvisionRequest struct {
	InviteCode      *string `json:"invite_code,omitempty"`
	InvitationToken *string `json:"invitation_token,omitempty"`
}

// NeonProvisionResponse reports what POST /auth/neon/provision did.
type NeonProvisionResponse struct {
	UserID             string `json:"user_id"`
	Created            bool   `json:"created"`
	Linked             bool   `json:"linked"`
	InvitationAccepted bool   `json:"invitation_accepted"`
}
