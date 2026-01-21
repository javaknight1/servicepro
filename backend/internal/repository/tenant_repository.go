package repository

import (
	"context"
	"errors"
	"fmt"
	"log"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/javaknight1/servicepro/backend/internal/models"
)

var (
	ErrTenantNotFound      = errors.New("tenant not found")
	ErrTenantSlugExists    = errors.New("tenant slug already exists")
	ErrUserNotInTenant     = errors.New("user not a member of this tenant")
	ErrUserAlreadyInTenant = errors.New("user is already a member of this tenant")
)

// TenantRepository handles tenant data operations
type TenantRepository struct {
	db *gorm.DB
}

// NewTenantRepository creates a new tenant repository
func NewTenantRepository(db *gorm.DB) *TenantRepository {
	return &TenantRepository{db: db}
}

// Create creates a new tenant
func (r *TenantRepository) Create(ctx context.Context, tenant *models.Tenant) error {
	// Check if slug already exists
	var count int64
	if err := r.db.WithContext(ctx).Model(&models.Tenant{}).
		Where("slug = ? AND deleted_at IS NULL", tenant.Slug).
		Count(&count).Error; err != nil {
		return fmt.Errorf("failed to check slug uniqueness: %w", err)
	}
	if count > 0 {
		return ErrTenantSlugExists
	}

	return r.db.WithContext(ctx).Create(tenant).Error
}

// GetByID retrieves a tenant by ID
func (r *TenantRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.Tenant, error) {
	var tenant models.Tenant
	err := r.db.WithContext(ctx).
		Where("id = ? AND deleted_at IS NULL", id).
		First(&tenant).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrTenantNotFound
		}
		return nil, err
	}
	return &tenant, nil
}

// GetBySlug retrieves a tenant by slug
func (r *TenantRepository) GetBySlug(ctx context.Context, slug string) (*models.Tenant, error) {
	var tenant models.Tenant
	err := r.db.WithContext(ctx).
		Where("slug = ? AND deleted_at IS NULL", slug).
		First(&tenant).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrTenantNotFound
		}
		return nil, err
	}
	return &tenant, nil
}

// Update updates a tenant
func (r *TenantRepository) Update(ctx context.Context, tenant *models.Tenant) error {
	// Check slug uniqueness if changed
	if tenant.Slug != "" {
		var count int64
		if err := r.db.WithContext(ctx).Model(&models.Tenant{}).
			Where("slug = ? AND id != ? AND deleted_at IS NULL", tenant.Slug, tenant.ID).
			Count(&count).Error; err != nil {
			return fmt.Errorf("failed to check slug uniqueness: %w", err)
		}
		if count > 0 {
			return ErrTenantSlugExists
		}
	}

	return r.db.WithContext(ctx).Save(tenant).Error
}

// Delete soft-deletes a tenant
func (r *TenantRepository) Delete(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).
		Where("id = ?", id).
		Delete(&models.Tenant{}).Error
}

// List retrieves all tenants with pagination
func (r *TenantRepository) List(ctx context.Context, page, pageSize int) ([]models.Tenant, int64, error) {
	var tenants []models.Tenant
	var total int64

	// Get total count
	if err := r.db.WithContext(ctx).Model(&models.Tenant{}).
		Where("deleted_at IS NULL").
		Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// Get paginated results
	offset := (page - 1) * pageSize
	if err := r.db.WithContext(ctx).
		Where("deleted_at IS NULL").
		Order("created_at DESC").
		Offset(offset).
		Limit(pageSize).
		Find(&tenants).Error; err != nil {
		return nil, 0, err
	}

	return tenants, total, nil
}

// GetUserTenants retrieves all tenants a user belongs to
func (r *TenantRepository) GetUserTenants(ctx context.Context, userID uuid.UUID) ([]models.Tenant, error) {
	var tenants []models.Tenant
	err := r.db.WithContext(ctx).
		Joins("JOIN tenant_users ON tenant_users.tenant_id = tenants.id").
		Where("tenant_users.user_id = ? AND tenant_users.is_active = ? AND tenants.deleted_at IS NULL AND tenants.is_active = ?",
			userID, true, true).
		Find(&tenants).Error
	if err != nil {
		return nil, err
	}
	return tenants, nil
}

// UserBelongsToTenant checks if a user is a member of a tenant
func (r *TenantRepository) UserBelongsToTenant(ctx context.Context, userID, tenantID uuid.UUID) (bool, error) {
	log.Printf("[TENANT-REPO] UserBelongsToTenant: userID=%s, tenantID=%s", userID, tenantID)

	var count int64
	err := r.db.WithContext(ctx).Model(&models.TenantUser{}).
		Where("user_id = ? AND tenant_id = ? AND is_active = ?", userID, tenantID, true).
		Count(&count).Error
	if err != nil {
		log.Printf("[TENANT-REPO] UserBelongsToTenant: Error=%v", err)
		return false, err
	}

	log.Printf("[TENANT-REPO] UserBelongsToTenant: count=%d, result=%v", count, count > 0)
	return count > 0, nil
}

// AddMember adds a user to a tenant
func (r *TenantRepository) AddMember(ctx context.Context, membership *models.TenantUser) error {
	log.Printf("[TENANT-REPO] AddMember: tenantID=%s, userID=%s, roleID=%s",
		membership.TenantID, membership.UserID, membership.RoleID)

	// Check if already a member
	var count int64
	if err := r.db.WithContext(ctx).Model(&models.TenantUser{}).
		Where("tenant_id = ? AND user_id = ?", membership.TenantID, membership.UserID).
		Count(&count).Error; err != nil {
		log.Printf("[TENANT-REPO] AddMember: Error checking existing membership: %v", err)
		return err
	}
	if count > 0 {
		log.Printf("[TENANT-REPO] AddMember: User already a member (count=%d)", count)
		return ErrUserAlreadyInTenant
	}

	err := r.db.WithContext(ctx).Create(membership).Error
	if err != nil {
		log.Printf("[TENANT-REPO] AddMember: Error creating membership: %v", err)
	} else {
		log.Printf("[TENANT-REPO] AddMember: Successfully created membership")
	}
	return err
}

// RemoveMember removes a user from a tenant
func (r *TenantRepository) RemoveMember(ctx context.Context, tenantID, userID uuid.UUID) error {
	result := r.db.WithContext(ctx).
		Where("tenant_id = ? AND user_id = ?", tenantID, userID).
		Delete(&models.TenantUser{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrUserNotInTenant
	}
	return nil
}

// UpdateMemberRole updates a member's role in a tenant
func (r *TenantRepository) UpdateMemberRole(ctx context.Context, tenantID, userID, roleID uuid.UUID) error {
	result := r.db.WithContext(ctx).Model(&models.TenantUser{}).
		Where("tenant_id = ? AND user_id = ?", tenantID, userID).
		Update("role_id", roleID)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrUserNotInTenant
	}
	return nil
}

// GetTenantMembers retrieves all members of a tenant
func (r *TenantRepository) GetTenantMembers(ctx context.Context, tenantID uuid.UUID) ([]models.TenantUser, error) {
	var members []models.TenantUser
	err := r.db.WithContext(ctx).
		Preload("User").
		Preload("Role").
		Where("tenant_id = ? AND is_active = ?", tenantID, true).
		Find(&members).Error
	if err != nil {
		return nil, err
	}
	return members, nil
}

// GetUserTenantMembership gets a user's membership in a specific tenant
func (r *TenantRepository) GetUserTenantMembership(ctx context.Context, userID, tenantID uuid.UUID) (*models.TenantUser, error) {
	var membership models.TenantUser
	err := r.db.WithContext(ctx).
		Preload("Role").
		Where("user_id = ? AND tenant_id = ? AND is_active = ?", userID, tenantID, true).
		First(&membership).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrUserNotInTenant
		}
		return nil, err
	}
	return &membership, nil
}

// SetMemberActive activates or deactivates a tenant membership
func (r *TenantRepository) SetMemberActive(ctx context.Context, tenantID, userID uuid.UUID, isActive bool) error {
	return r.db.WithContext(ctx).Model(&models.TenantUser{}).
		Where("tenant_id = ? AND user_id = ?", tenantID, userID).
		Update("is_active", isActive).Error
}
