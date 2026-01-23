package models

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

var (
	ErrPermissionNotFound      = errors.New("permission not found")
	ErrPermissionNameRequired  = errors.New("permission name is required")
	ErrPermissionNameTooLong   = errors.New("permission name must be 100 characters or less")
	ErrResourceRequired        = errors.New("resource is required")
	ErrActionRequired          = errors.New("action is required")
	ErrInvalidAction           = errors.New("invalid action")
	ErrInvalidPermissionFormat = errors.New("invalid permission format (expected resource.action)")
)

// ValidActions defines the allowed permission actions
var ValidActions = []string{"create", "read", "update", "delete", "list", "manage", "execute", "approve", "assign", "grant"}

// Permission represents a granular permission in the system
type Permission struct {
	ID          uuid.UUID `json:"id" gorm:"type:uuid;primary_key;default:uuid_generate_v4()"`
	Name        string    `json:"name" gorm:"type:varchar(100);uniqueIndex;not null"`
	Description string    `json:"description" gorm:"type:text"`

	// Resource-based access control
	Resource string `json:"resource" gorm:"type:varchar(100);not null;index:idx_resource_action"`
	Action   string `json:"action" gorm:"type:varchar(50);not null;index:idx_resource_action"`

	// Relationships
	Roles []Role `json:"roles,omitempty" gorm:"many2many:role_permissions;"`

	// Audit fields
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `json:"deleted_at,omitempty" gorm:"index"`
}

// TableName overrides the table name
func (Permission) TableName() string {
	return "permissions"
}

// BeforeCreate hook to validate before creating
func (p *Permission) BeforeCreate(tx *gorm.DB) error {
	if err := p.Validate(); err != nil {
		return err
	}

	// Set UUID if not provided
	if p.ID == uuid.Nil {
		p.ID = uuid.New()
	}

	// Auto-generate name from resource and action if not provided
	if p.Name == "" {
		p.Name = fmt.Sprintf("%s.%s", p.Resource, p.Action)
	}

	return nil
}

// BeforeUpdate hook to validate before updating
func (p *Permission) BeforeUpdate(tx *gorm.DB) error {
	return p.Validate()
}

// Validate performs validation on the permission
func (p *Permission) Validate() error {
	// Name is required
	if p.Name == "" {
		return ErrPermissionNameRequired
	}

	// Name length check
	if len(p.Name) > 100 {
		return ErrPermissionNameTooLong
	}

	// Resource is required
	if p.Resource == "" {
		return ErrResourceRequired
	}

	// Action is required
	if p.Action == "" {
		return ErrActionRequired
	}

	// Validate action
	if !IsValidAction(p.Action) {
		return ErrInvalidAction
	}

	return nil
}

// IsValidAction checks if the action is valid
func IsValidAction(action string) bool {
	action = strings.ToLower(action)
	for _, valid := range ValidActions {
		if action == valid {
			return true
		}
	}
	return false
}

// ParsePermissionName parses a permission name (e.g., "users.create") into resource and action
// Splits on the first dot only, so "api.users.create" becomes resource="api", action="users.create"
func ParsePermissionName(name string) (resource, action string, err error) {
	parts := strings.SplitN(name, ".", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", "", ErrInvalidPermissionFormat
	}
	return parts[0], parts[1], nil
}

// Matches checks if this permission matches the given resource and action
func (p *Permission) Matches(resource, action string) bool {
	return p.Resource == resource && p.Action == action
}

// MatchesName checks if this permission matches the given permission name
func (p *Permission) MatchesName(name string) bool {
	return p.Name == name
}

// GetFullName returns the full permission name (resource.action)
func (p *Permission) GetFullName() string {
	if p.Name != "" {
		return p.Name
	}
	return fmt.Sprintf("%s.%s", p.Resource, p.Action)
}

// IsWildcard checks if this permission is a wildcard permission (e.g., "users.*")
func (p *Permission) IsWildcard() bool {
	return p.Action == "*"
}

// CoversAction checks if this permission covers the given action
// Wildcard permissions (*) cover all actions
func (p *Permission) CoversAction(action string) bool {
	return p.Action == action || p.Action == "*"
}

// CoversResource checks if this permission covers operations on the given resource
func (p *Permission) CoversResource(resource string) bool {
	return p.Resource == resource || p.Resource == "*"
}

// Covers checks if this permission covers a specific resource and action
func (p *Permission) Covers(resource, action string) bool {
	return p.CoversResource(resource) && p.CoversAction(action)
}

// PermissionResponse is the response DTO for permission information
type PermissionResponse struct {
	ID          uuid.UUID `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Resource    string    `json:"resource"`
	Action      string    `json:"action"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// ToResponse converts Permission to PermissionResponse
func (p *Permission) ToResponse() *PermissionResponse {
	return &PermissionResponse{
		ID:          p.ID,
		Name:        p.Name,
		Description: p.Description,
		Resource:    p.Resource,
		Action:      p.Action,
		CreatedAt:   p.CreatedAt,
		UpdatedAt:   p.UpdatedAt,
	}
}

// PermissionChecker provides methods to check permissions
type PermissionChecker struct {
	permissions []Permission
}

// NewPermissionChecker creates a new permission checker
func NewPermissionChecker(permissions []Permission) *PermissionChecker {
	return &PermissionChecker{
		permissions: permissions,
	}
}

// Has checks if the checker has a specific permission by name
func (pc *PermissionChecker) Has(permissionName string) bool {
	for _, perm := range pc.permissions {
		if perm.MatchesName(permissionName) {
			return true
		}
	}
	return false
}

// Can checks if the checker can perform an action on a resource
func (pc *PermissionChecker) Can(resource, action string) bool {
	for _, perm := range pc.permissions {
		if perm.Covers(resource, action) {
			return true
		}
	}
	return false
}

// CanAny checks if the checker has any of the given permissions
func (pc *PermissionChecker) CanAny(permissionNames ...string) bool {
	for _, name := range permissionNames {
		if pc.Has(name) {
			return true
		}
	}
	return false
}

// CanAll checks if the checker has all of the given permissions
func (pc *PermissionChecker) CanAll(permissionNames ...string) bool {
	for _, name := range permissionNames {
		if !pc.Has(name) {
			return false
		}
	}
	return true
}

// GetPermissions returns all permissions
func (pc *PermissionChecker) GetPermissions() []Permission {
	return pc.permissions
}

// GetPermissionsByResource returns all permissions for a specific resource
func (pc *PermissionChecker) GetPermissionsByResource(resource string) []Permission {
	var result []Permission
	for _, perm := range pc.permissions {
		if perm.Resource == resource {
			result = append(result, perm)
		}
	}
	return result
}

// GetPermissionsByAction returns all permissions with a specific action
func (pc *PermissionChecker) GetPermissionsByAction(action string) []Permission {
	var result []Permission
	for _, perm := range pc.permissions {
		if perm.Action == action {
			result = append(result, perm)
		}
	}
	return result
}

// ResourcePermissions groups permissions by resource
type ResourcePermissions struct {
	Resource    string       `json:"resource"`
	Permissions []Permission `json:"permissions"`
}

// GroupByResource groups permissions by resource
func (pc *PermissionChecker) GroupByResource() []ResourcePermissions {
	grouped := make(map[string][]Permission)

	for _, perm := range pc.permissions {
		grouped[perm.Resource] = append(grouped[perm.Resource], perm)
	}

	result := make([]ResourcePermissions, 0, len(grouped))
	for resource, perms := range grouped {
		result = append(result, ResourcePermissions{
			Resource:    resource,
			Permissions: perms,
		})
	}

	return result
}
