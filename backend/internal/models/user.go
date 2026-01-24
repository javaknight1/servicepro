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
	ID                 uuid.UUID      `json:"id" gorm:"type:uuid;primary_key;default:uuid_generate_v4()"`
	Email              string         `json:"email" gorm:"uniqueIndex;not null"`
	PasswordHash       string         `json:"-" gorm:"not null"`
	Role               UserRole       `json:"role" gorm:"type:varchar(50);not null;default:'user'"`
	FirstName          *string        `json:"first_name" gorm:"column:first_name;size:100"`
	LastName           *string        `json:"last_name" gorm:"column:last_name;size:100"`
	ProfilePictureURL  *string        `json:"profile_picture_url" gorm:"column:profile_picture_url"`
	EmailVerified      bool           `json:"email_verified" gorm:"default:false"`
	VerificationSentAt *time.Time     `json:"-"`
	FailedLoginCount   int            `json:"-" gorm:"default:0"`
	LastFailedLoginAt  *time.Time     `json:"-"`
	LockedUntil        *time.Time     `json:"-"`
	CreatedAt          time.Time      `json:"created_at"`
	UpdatedAt          time.Time      `json:"updated_at"`
	DeletedAt          gorm.DeletedAt `json:"-" gorm:"index"`
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

// LoginResponse represents the login response payload
type LoginResponse struct {
	Token        string `json:"token"`
	RefreshToken string `json:"refreshToken"`
	ExpiresIn    int    `json:"expiresIn"` // in seconds
}

// RegisterRequest represents the registration request payload
type RegisterRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=8"`
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

// ErrorResponse represents an error response
type ErrorResponse struct {
	Error   string   `json:"error"`
	Message string   `json:"message,omitempty"`
	Details []string `json:"details,omitempty"`
}

// ProfilePictureUploadResponse represents the response for profile picture upload
type ProfilePictureUploadResponse struct {
	ProfilePictureURL string `json:"profile_picture_url"`
}
