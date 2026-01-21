package models

import (
	"errors"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

var (
	ErrRoleNotFound           = errors.New("role not found")
	ErrRoleNameRequired       = errors.New("role name is required")
	ErrRoleNameTooLong        = errors.New("role name must be 100 characters or less")
	ErrInvalidHierarchyLevel  = errors.New("hierarchy level must be between 0 and 100")
	ErrCircularRoleHierarchy  = errors.New("circular role hierarchy detected")
	ErrCannotDeleteSystemRole = errors.New("cannot delete system role")
)

// Role represents a user role in the system with hierarchical support
type Role struct {
	ID          uuid.UUID `json:"id" gorm:"type:uuid;primary_key;default:uuid_generate_v4()"`
	Name        string    `json:"name" gorm:"type:varchar(100);uniqueIndex;not null"`
	Description string    `json:"description" gorm:"type:text"`

	// Hierarchy support
	ParentRoleID   *uuid.UUID `json:"parent_role_id,omitempty" gorm:"type:uuid"`
	ParentRole     *Role      `json:"parent_role,omitempty" gorm:"foreignKey:ParentRoleID"`
	ChildRoles     []Role     `json:"child_roles,omitempty" gorm:"foreignKey:ParentRoleID"`
	HierarchyLevel int        `json:"hierarchy_level" gorm:"default:0"`

	// Relationships
	Permissions []Permission `json:"permissions,omitempty" gorm:"many2many:role_permissions;"`
	Users       []User       `json:"users,omitempty" gorm:"many2many:user_roles;"`

	// Audit fields
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `json:"deleted_at,omitempty" gorm:"index"`
}

// TableName overrides the table name
func (Role) TableName() string {
	return "roles"
}

// BeforeCreate hook to validate before creating
func (r *Role) BeforeCreate(tx *gorm.DB) error {
	if err := r.Validate(); err != nil {
		return err
	}

	// Set UUID if not provided
	if r.ID == uuid.Nil {
		r.ID = uuid.New()
	}

	return nil
}

// BeforeUpdate hook to validate before updating
func (r *Role) BeforeUpdate(tx *gorm.DB) error {
	return r.Validate()
}

// BeforeDelete hook to prevent deletion of system roles
func (r *Role) BeforeDelete(tx *gorm.DB) error {
	// Prevent deletion of system roles (hierarchy level >= 80)
	if r.HierarchyLevel >= 80 {
		return ErrCannotDeleteSystemRole
	}
	return nil
}

// Validate performs validation on the role
func (r *Role) Validate() error {
	// Name is required
	if r.Name == "" {
		return ErrRoleNameRequired
	}

	// Name length check
	if len(r.Name) > 100 {
		return ErrRoleNameTooLong
	}

	// Hierarchy level validation
	if r.HierarchyLevel < 0 || r.HierarchyLevel > 100 {
		return ErrInvalidHierarchyLevel
	}

	return nil
}

// IsSystemRole checks if this is a system-defined role (cannot be deleted)
func (r *Role) IsSystemRole() bool {
	return r.HierarchyLevel >= 80
}

// IsSuperAdmin checks if this role is super admin
func (r *Role) IsSuperAdmin() bool {
	return r.Name == "super_admin" || r.HierarchyLevel == 100
}

// IsAdmin checks if this role is admin or higher
func (r *Role) IsAdmin() bool {
	return r.HierarchyLevel >= 80
}

// HasPermission checks if the role has a specific permission
func (r *Role) HasPermission(permissionName string) bool {
	for _, perm := range r.Permissions {
		if perm.Name == permissionName {
			return true
		}
	}
	return false
}

// HasPermissionForResource checks if role has permission for a resource and action
func (r *Role) HasPermissionForResource(resource, action string) bool {
	for _, perm := range r.Permissions {
		if perm.Resource == resource && perm.Action == action {
			return true
		}
	}
	return false
}

// GetAllPermissions returns all permissions including inherited from parent roles
func (r *Role) GetAllPermissions() []Permission {
	permissions := make(map[uuid.UUID]Permission)

	// Add own permissions
	for _, perm := range r.Permissions {
		permissions[perm.ID] = perm
	}

	// Add parent role permissions (recursive)
	if r.ParentRole != nil {
		parentPerms := r.ParentRole.GetAllPermissions()
		for _, perm := range parentPerms {
			permissions[perm.ID] = perm
		}
	}

	// Convert map to slice
	result := make([]Permission, 0, len(permissions))
	for _, perm := range permissions {
		result = append(result, perm)
	}

	return result
}

// HasPermissionRecursive checks permission including parent roles
func (r *Role) HasPermissionRecursive(permissionName string) bool {
	// Check own permissions first
	if r.HasPermission(permissionName) {
		return true
	}

	// Check parent role permissions
	if r.ParentRole != nil {
		return r.ParentRole.HasPermissionRecursive(permissionName)
	}

	return false
}

// IsHigherThan checks if this role is higher in hierarchy than another role
func (r *Role) IsHigherThan(other *Role) bool {
	return r.HierarchyLevel > other.HierarchyLevel
}

// IsLowerThan checks if this role is lower in hierarchy than another role
func (r *Role) IsLowerThan(other *Role) bool {
	return r.HierarchyLevel < other.HierarchyLevel
}

// CanManage checks if this role can manage another role
// A role can manage roles lower in the hierarchy
func (r *Role) CanManage(other *Role) bool {
	return r.IsHigherThan(other)
}

// GetDescendants returns all child roles recursively
func (r *Role) GetDescendants() []Role {
	descendants := make([]Role, 0)

	for _, child := range r.ChildRoles {
		descendants = append(descendants, child)
		// Recursively get descendants
		childDescendants := child.GetDescendants()
		descendants = append(descendants, childDescendants...)
	}

	return descendants
}

// GetAncestors returns all parent roles recursively up the hierarchy
func (r *Role) GetAncestors() []Role {
	ancestors := make([]Role, 0)

	if r.ParentRole != nil {
		ancestors = append(ancestors, *r.ParentRole)
		// Recursively get ancestors
		parentAncestors := r.ParentRole.GetAncestors()
		ancestors = append(ancestors, parentAncestors...)
	}

	return ancestors
}

// ValidateHierarchy checks for circular references in role hierarchy
func (r *Role) ValidateHierarchy() error {
	visited := make(map[uuid.UUID]bool)
	return r.validateHierarchyRecursive(visited)
}

func (r *Role) validateHierarchyRecursive(visited map[uuid.UUID]bool) error {
	// Check if we've visited this role (circular reference)
	if visited[r.ID] {
		return ErrCircularRoleHierarchy
	}

	visited[r.ID] = true

	// Check parent
	if r.ParentRole != nil {
		if err := r.ParentRole.validateHierarchyRecursive(visited); err != nil {
			return err
		}
	}

	return nil
}

// RoleAssignment represents the many-to-many relationship between users and roles
type RoleAssignment struct {
	UserID     uuid.UUID  `json:"user_id" gorm:"type:uuid;primaryKey"`
	RoleID     uuid.UUID  `json:"role_id" gorm:"type:uuid;primaryKey"`
	User       User       `json:"user" gorm:"foreignKey:UserID"`
	Role       Role       `json:"role" gorm:"foreignKey:RoleID"`
	AssignedAt time.Time  `json:"assigned_at" gorm:"default:CURRENT_TIMESTAMP"`
	AssignedBy *uuid.UUID `json:"assigned_by,omitempty" gorm:"type:uuid"`
	ExpiresAt  *time.Time `json:"expires_at,omitempty"`
}

// TableName overrides the table name
func (RoleAssignment) TableName() string {
	return "user_roles"
}

// IsExpired checks if the role assignment has expired
func (ra *RoleAssignment) IsExpired() bool {
	if ra.ExpiresAt == nil {
		return false
	}
	return time.Now().After(*ra.ExpiresAt)
}

// IsActive checks if the role assignment is currently active
func (ra *RoleAssignment) IsActive() bool {
	return !ra.IsExpired()
}

// RolePermission represents the many-to-many relationship between roles and permissions
type RolePermission struct {
	RoleID       uuid.UUID  `json:"role_id" gorm:"type:uuid;primaryKey"`
	PermissionID uuid.UUID  `json:"permission_id" gorm:"type:uuid;primaryKey"`
	Role         Role       `json:"role" gorm:"foreignKey:RoleID"`
	Permission   Permission `json:"permission" gorm:"foreignKey:PermissionID"`
	GrantedAt    time.Time  `json:"granted_at" gorm:"default:CURRENT_TIMESTAMP"`
	GrantedBy    *uuid.UUID `json:"granted_by,omitempty" gorm:"type:uuid"`
}

// TableName overrides the table name
func (RolePermission) TableName() string {
	return "role_permissions"
}

// RoleResponse is the response DTO for role information
type RoleResponse struct {
	ID             uuid.UUID            `json:"id"`
	Name           string               `json:"name"`
	Description    string               `json:"description"`
	HierarchyLevel int                  `json:"hierarchy_level"`
	ParentRoleID   *uuid.UUID           `json:"parent_role_id,omitempty"`
	Permissions    []PermissionResponse `json:"permissions,omitempty"`
	CreatedAt      time.Time            `json:"created_at"`
	UpdatedAt      time.Time            `json:"updated_at"`
}

// ToResponse converts Role to RoleResponse
func (r *Role) ToResponse() *RoleResponse {
	resp := &RoleResponse{
		ID:             r.ID,
		Name:           r.Name,
		Description:    r.Description,
		HierarchyLevel: r.HierarchyLevel,
		ParentRoleID:   r.ParentRoleID,
		CreatedAt:      r.CreatedAt,
		UpdatedAt:      r.UpdatedAt,
	}

	// Convert permissions
	if len(r.Permissions) > 0 {
		resp.Permissions = make([]PermissionResponse, len(r.Permissions))
		for i, perm := range r.Permissions {
			resp.Permissions[i] = *perm.ToResponse()
		}
	}

	return resp
}
