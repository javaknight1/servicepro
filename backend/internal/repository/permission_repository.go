package repository

import (
	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/javaknight1/servicepro/backend/internal/models"
)

// PermissionRepository handles permission-related data access
type PermissionRepository struct {
	db *gorm.DB
}

// NewPermissionRepository creates a new permission repository
func NewPermissionRepository(db *gorm.DB) *PermissionRepository {
	return &PermissionRepository{db: db}
}

// userRoleIDsCondition selects the role IDs that apply to a user in ONE tenant:
//   - platform roles from user_roles (explicit, operator-assigned; the API
//     never writes this table), and
//   - the user's active membership role in the given tenant only.
//
// Roles the user holds in other tenants are never included (F-2). When the
// tenant ID is uuid.Nil only platform roles apply.
//
// Bind order: userID, userID, tenantID.
const userRoleIDsCondition = `
		r.id IN (
			SELECT ur.role_id FROM user_roles ur
			WHERE ur.user_id = ?
			AND (ur.expires_at IS NULL OR ur.expires_at > CURRENT_TIMESTAMP)
		)
		OR
		r.id IN (
			SELECT tu.role_id FROM tenant_users tu
			WHERE tu.user_id = ?
			AND tu.tenant_id = ?
			AND tu.is_active = true
		)`

// GetUserPermissions retrieves the permissions a user holds in a tenant: the
// union of the user's platform roles (user_roles) and their membership role
// in tenant_users for that tenant only.
func (r *PermissionRepository) GetUserPermissions(userID, tenantID uuid.UUID) ([]models.Permission, error) {
	var permissions []models.Permission

	query := `
		SELECT DISTINCT p.*
		FROM permissions p
		INNER JOIN role_permissions rp ON rp.permission_id = p.id
		INNER JOIN roles r ON r.id = rp.role_id
		WHERE p.deleted_at IS NULL
		AND r.deleted_at IS NULL
		AND (` + userRoleIDsCondition + `
		)
	`

	if err := r.db.Raw(query, userID, userID, tenantID).Scan(&permissions).Error; err != nil {
		return nil, err
	}

	return permissions, nil
}

// GetUserRoles retrieves the user's platform roles plus their role in the given tenant.
func (r *PermissionRepository) GetUserRoles(userID, tenantID uuid.UUID) ([]models.Role, error) {
	var roles []models.Role

	query := `
		SELECT DISTINCT r.*
		FROM roles r
		WHERE r.deleted_at IS NULL
		AND (` + userRoleIDsCondition + `
		)
	`

	if err := r.db.Raw(query, userID, userID, tenantID).Scan(&roles).Error; err != nil {
		return nil, err
	}

	// Load permissions for each role
	for i := range roles {
		if err := r.db.Model(&roles[i]).Association("Permissions").Find(&roles[i].Permissions); err != nil {
			return nil, err
		}
	}

	return roles, nil
}

// GetRoleWithPermissions retrieves a role with all its permissions (including inherited)
func (r *PermissionRepository) GetRoleWithPermissions(roleID uuid.UUID) (*models.Role, error) {
	var role models.Role

	err := r.db.
		Where("id = ?", roleID).
		Preload("Permissions").
		Preload("ParentRole").
		First(&role).Error

	if err != nil {
		return nil, err
	}

	return &role, nil
}

// GetPermissionByName retrieves a permission by its name
func (r *PermissionRepository) GetPermissionByName(name string) (*models.Permission, error) {
	var permission models.Permission

	err := r.db.Where("name = ?", name).First(&permission).Error
	if err != nil {
		return nil, err
	}

	return &permission, nil
}

// GetPermissionByResourceAction retrieves a permission by resource and action
func (r *PermissionRepository) GetPermissionByResourceAction(resource, action string) (*models.Permission, error) {
	var permission models.Permission

	err := r.db.
		Where("resource = ? AND action = ?", resource, action).
		First(&permission).Error

	if err != nil {
		return nil, err
	}

	return &permission, nil
}

// GetUserHighestRole retrieves the user's highest role (by hierarchy level)
// among their platform roles and their membership role in the given tenant.
func (r *PermissionRepository) GetUserHighestRole(userID, tenantID uuid.UUID) (*models.Role, error) {
	var role models.Role

	query := `
		SELECT r.*
		FROM roles r
		WHERE r.deleted_at IS NULL
		AND (` + userRoleIDsCondition + `
		)
		ORDER BY r.hierarchy_level DESC
		LIMIT 1
	`

	if err := r.db.Raw(query, userID, userID, tenantID).Scan(&role).Error; err != nil {
		return nil, err
	}

	// Check if we actually got a role (uuid.Nil means no role found)
	if role.ID == uuid.Nil {
		return nil, gorm.ErrRecordNotFound
	}

	return &role, nil
}

// GetAllRoles retrieves all system roles
func (r *PermissionRepository) GetAllRoles() ([]models.Role, error) {
	var roles []models.Role

	err := r.db.
		Where("deleted_at IS NULL").
		Order("hierarchy_level DESC").
		Find(&roles).Error

	if err != nil {
		return nil, err
	}

	return roles, nil
}

// GetAllPermissions retrieves all system permissions
func (r *PermissionRepository) GetAllPermissions() ([]models.Permission, error) {
	var permissions []models.Permission

	err := r.db.
		Where("deleted_at IS NULL").
		Order("resource, action").
		Find(&permissions).Error

	if err != nil {
		return nil, err
	}

	return permissions, nil
}

// GetRolePermissions retrieves all permissions for a specific role
func (r *PermissionRepository) GetRolePermissions(roleID string) ([]models.Permission, error) {
	var permissions []models.Permission

	err := r.db.
		Table("permissions").
		Joins("INNER JOIN role_permissions ON role_permissions.permission_id = permissions.id").
		Where("role_permissions.role_id = ?", roleID).
		Where("permissions.deleted_at IS NULL").
		Find(&permissions).Error

	if err != nil {
		return nil, err
	}

	return permissions, nil
}
