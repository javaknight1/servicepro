package services

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/javaknight1/servicepro/backend/internal/models"
	"github.com/javaknight1/servicepro/backend/internal/repository"
	"github.com/javaknight1/servicepro/backend/pkg/clients/logging"
)

var (
	ErrInvalidTenantName     = errors.New("invalid tenant name")
	ErrInvalidTenantSlug     = errors.New("invalid tenant slug")
	ErrCannotRemoveOwner     = errors.New("cannot remove tenant owner")
	ErrCannotChangeOwnerRole = errors.New("cannot change owner's role")
	// ErrCannotChangeOwnRole is returned when a member tries to change their own role.
	ErrCannotChangeOwnRole = errors.New("cannot change your own role")
	// ErrRoleNotAssignable is returned for roles that can never be granted
	// through tenant membership (super_admin) or that don't exist.
	ErrRoleNotAssignable = errors.New("role cannot be assigned")
	// ErrInsufficientRoleLevel is returned when the actor tries to grant a
	// role above their own, or manage a member who outranks them.
	ErrInsufficientRoleLevel = errors.New("insufficient role level")
	// ErrMemberNotAddable is a deliberately generic error for AddMember so the
	// endpoint can't be used to probe which emails have accounts.
	ErrMemberNotAddable = errors.New("member could not be added")
)

// SuperAdminHierarchyLevel is the hierarchy level of the platform super_admin
// role. Roles at or above it are never assignable through the API.
const SuperAdminHierarchyLevel = 100

// MemberActor identifies the member performing a membership change and their
// role level in the tenant (from the tenant middleware, never the request).
type MemberActor struct {
	UserID    uuid.UUID
	RoleLevel int
}

// ValidateAssignableRole enforces the role-grant rules shared by direct member
// management and invitations: super_admin (or anything at its level) is never
// assignable, and nobody may grant a role above their own level.
func ValidateAssignableRole(role *models.Role, actor MemberActor) error {
	if role == nil || role.HierarchyLevel >= SuperAdminHierarchyLevel || role.Name == "super_admin" {
		return ErrRoleNotAssignable
	}
	if role.HierarchyLevel > actor.RoleLevel {
		return ErrInsufficientRoleLevel
	}
	return nil
}

// checkAssignableRole loads a role and validates it with ValidateAssignableRole.
func (s *TenantService) checkAssignableRole(ctx context.Context, roleID uuid.UUID, actor MemberActor) error {
	role, err := s.tenantRepo.GetRoleByID(ctx, roleID)
	if err != nil {
		return ErrRoleNotAssignable
	}
	return ValidateAssignableRole(role, actor)
}

// checkCanManageMember ensures the target member does not outrank the actor.
func (s *TenantService) checkCanManageMember(ctx context.Context, tenantID, targetUserID uuid.UUID, actor MemberActor) error {
	membership, err := s.tenantRepo.GetUserTenantMembership(ctx, targetUserID, tenantID)
	if err != nil {
		return err
	}
	if membership.Role != nil && membership.Role.HierarchyLevel > actor.RoleLevel {
		return ErrInsufficientRoleLevel
	}
	return nil
}

// slugRegex validates tenant slugs (alphanumeric and hyphens only)
var slugRegex = regexp.MustCompile(`^[a-z0-9][a-z0-9-]*[a-z0-9]$|^[a-z0-9]$`)

// PermissionCacheInvalidator interface for invalidating permission cache
type PermissionCacheInvalidator interface {
	InvalidateUserPermissions(ctx context.Context, userID uuid.UUID) error
}

// OwnerRoleName is the role a tenant's creator gets in the new tenant.
const OwnerRoleName = "admin"

// TenantService handles tenant business logic
type TenantService struct {
	tenantRepo      *repository.TenantRepository
	userRepo        repository.UserRepositoryInterface
	permInvalidator PermissionCacheInvalidator
	defaultRoleID   uuid.UUID // Default role for new members
}

// NewTenantService creates a new tenant service
func NewTenantService(tenantRepo *repository.TenantRepository, userRepo repository.UserRepositoryInterface) *TenantService {
	// Default role ID for new members (user role)
	defaultRoleID, _ := uuid.Parse("00000000-0000-0000-0000-000000000004")
	return &TenantService{
		tenantRepo:    tenantRepo,
		userRepo:      userRepo,
		defaultRoleID: defaultRoleID,
	}
}

// SetPermissionInvalidator sets the permission cache invalidator
// This is optional - if not set, permission cache won't be invalidated on membership changes
func (s *TenantService) SetPermissionInvalidator(invalidator PermissionCacheInvalidator) {
	s.permInvalidator = invalidator
}

// invalidatePermissionCache invalidates the permission cache for a user
func (s *TenantService) invalidatePermissionCache(ctx context.Context, userID uuid.UUID) {
	if s.permInvalidator != nil {
		if err := s.permInvalidator.InvalidateUserPermissions(ctx, userID); err != nil {
			logging.Warn(ctx, "[TENANT] Failed to invalidate permission cache for user", map[string]any{"user_id": userID, "error": err})
		} else {
			logging.Info(ctx, "[TENANT] Invalidated permission cache for user", map[string]any{"user_id": userID})
		}
	}
}

// CreateTenant creates a new tenant and adds the creator as owner
func (s *TenantService) CreateTenant(ctx context.Context, req *models.CreateTenantRequest, ownerID uuid.UUID) (*models.Tenant, error) {
	// Validate name
	if strings.TrimSpace(req.Name) == "" {
		return nil, ErrInvalidTenantName
	}

	// Validate and normalize slug
	slug := strings.ToLower(strings.TrimSpace(req.Slug))
	if !slugRegex.MatchString(slug) {
		return nil, ErrInvalidTenantSlug
	}

	// Create tenant
	tenant := &models.Tenant{
		ID:       uuid.New(),
		Name:     strings.TrimSpace(req.Name),
		Slug:     slug,
		OwnerID:  ownerID,
		Email:    req.Email,
		Phone:    req.Phone,
		IsActive: true,
	}

	if req.Settings != nil {
		tenant.Settings = *req.Settings
	}

	// Tenant, owner membership and default tier are created atomically
	// (T226): a failure leaves nothing behind and does not burn the slug.
	if err := s.tenantRepo.CreateWithOwner(ctx, tenant, OwnerRoleName, DefaultTierName); err != nil {
		if errors.Is(err, repository.ErrTenantSlugExists) {
			return nil, ErrInvalidTenantSlug
		}
		return nil, fmt.Errorf("failed to create tenant: %w", err)
	}

	logging.Info(ctx, "[TENANT] Created tenant", map[string]any{"tenant_id": tenant.ID, "owner_id": ownerID})

	// Invalidate permission cache for the owner so they get their new permissions
	s.invalidatePermissionCache(ctx, ownerID)

	return tenant, nil
}

// GetTenant retrieves a tenant by ID
func (s *TenantService) GetTenant(ctx context.Context, id uuid.UUID) (*models.Tenant, error) {
	return s.tenantRepo.GetByID(ctx, id)
}

// UpdateTenant updates a tenant
func (s *TenantService) UpdateTenant(ctx context.Context, id uuid.UUID, req *models.UpdateTenantRequest) (*models.Tenant, error) {
	tenant, err := s.tenantRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if req.Name != nil {
		name := strings.TrimSpace(*req.Name)
		if name == "" {
			return nil, ErrInvalidTenantName
		}
		tenant.Name = name
	}

	if req.Slug != nil {
		slug := strings.ToLower(strings.TrimSpace(*req.Slug))
		if !slugRegex.MatchString(slug) {
			return nil, ErrInvalidTenantSlug
		}
		tenant.Slug = slug
	}

	if req.Email != nil {
		tenant.Email = *req.Email
	}

	if req.Phone != nil {
		tenant.Phone = *req.Phone
	}

	if req.LogoURL != nil {
		tenant.LogoURL = *req.LogoURL
	}

	if req.IsActive != nil {
		tenant.IsActive = *req.IsActive
	}

	if req.Settings != nil {
		tenant.Settings = *req.Settings
	}

	if err := s.tenantRepo.Update(ctx, tenant); err != nil {
		if errors.Is(err, repository.ErrTenantSlugExists) {
			return nil, ErrInvalidTenantSlug
		}
		return nil, fmt.Errorf("failed to update tenant: %w", err)
	}

	return tenant, nil
}

// DeleteTenant deletes a tenant
func (s *TenantService) DeleteTenant(ctx context.Context, id uuid.UUID) error {
	return s.tenantRepo.Delete(ctx, id)
}

// GetUserTenants gets all tenants a user belongs to
func (s *TenantService) GetUserTenants(ctx context.Context, userID uuid.UUID) ([]models.TenantResponse, error) {
	tenants, err := s.tenantRepo.GetUserTenants(ctx, userID)
	if err != nil {
		return nil, err
	}

	responses := make([]models.TenantResponse, len(tenants))
	for i, t := range tenants {
		responses[i] = t.ToResponse()

		// Get user's role in this tenant
		membership, err := s.tenantRepo.GetUserTenantMembership(ctx, userID, t.ID)
		if err == nil && membership.Role != nil {
			responses[i].Role = &membership.Role.Name
		}
	}

	return responses, nil
}

// AddMember adds a user to a tenant
func (s *TenantService) AddMember(ctx context.Context, tenantID uuid.UUID, req *models.AddMemberRequest, actor MemberActor) (*models.TenantMemberResponse, error) {
	if err := s.checkAssignableRole(ctx, req.RoleID, actor); err != nil {
		return nil, err
	}
	invitedBy := actor.UserID

	// Find user by email. Don't reveal whether the account exists.
	user, err := s.userRepo.GetByEmail(req.Email)
	if err != nil || user == nil {
		return nil, ErrMemberNotAddable
	}

	now := time.Now()
	membership := &models.TenantUser{
		ID:        uuid.New(),
		TenantID:  tenantID,
		UserID:    user.ID,
		RoleID:    req.RoleID,
		InvitedBy: &invitedBy,
		InvitedAt: &now,
		IsActive:  true,
	}

	if err := s.tenantRepo.AddMember(ctx, membership); err != nil {
		if errors.Is(err, repository.ErrUserAlreadyInTenant) {
			return nil, err
		}
		return nil, fmt.Errorf("failed to add member: %w", err)
	}

	// Invalidate permission cache for the new member
	s.invalidatePermissionCache(ctx, user.ID)

	// Get the created membership with relations
	membership, err = s.tenantRepo.GetUserTenantMembership(ctx, user.ID, tenantID)
	if err != nil {
		return nil, err
	}

	response := membership.ToMemberResponse()
	response.Email = user.Email
	return &response, nil
}

// RemoveMember removes a user from a tenant
func (s *TenantService) RemoveMember(ctx context.Context, tenantID, userID uuid.UUID, actor MemberActor) error {
	// Check if user is the owner
	tenant, err := s.tenantRepo.GetByID(ctx, tenantID)
	if err != nil {
		return err
	}
	if tenant.OwnerID == userID {
		return ErrCannotRemoveOwner
	}
	// Members may always leave; removing someone else requires not being outranked.
	if userID != actor.UserID {
		if err := s.checkCanManageMember(ctx, tenantID, userID, actor); err != nil {
			return err
		}
	}

	if err := s.tenantRepo.RemoveMember(ctx, tenantID, userID); err != nil {
		return err
	}

	// Invalidate permission cache for the removed member
	s.invalidatePermissionCache(ctx, userID)

	return nil
}

// UpdateMemberRole updates a member's role
func (s *TenantService) UpdateMemberRole(ctx context.Context, tenantID, userID, roleID uuid.UUID, actor MemberActor) error {
	if userID == actor.UserID {
		return ErrCannotChangeOwnRole
	}
	if err := s.checkAssignableRole(ctx, roleID, actor); err != nil {
		return err
	}
	if err := s.checkCanManageMember(ctx, tenantID, userID, actor); err != nil {
		return err
	}

	// Check if user is the owner - owner must remain admin
	tenant, err := s.tenantRepo.GetByID(ctx, tenantID)
	if err != nil {
		return err
	}
	if tenant.OwnerID == userID {
		return ErrCannotChangeOwnerRole
	}

	if err := s.tenantRepo.UpdateMemberRole(ctx, tenantID, userID, roleID); err != nil {
		return err
	}

	// Invalidate permission cache for the user whose role changed
	s.invalidatePermissionCache(ctx, userID)

	return nil
}

// GetTenantMembers gets all members of a tenant
func (s *TenantService) GetTenantMembers(ctx context.Context, tenantID uuid.UUID) ([]models.TenantMemberResponse, error) {
	members, err := s.tenantRepo.GetTenantMembers(ctx, tenantID)
	if err != nil {
		return nil, err
	}

	responses := make([]models.TenantMemberResponse, len(members))
	for i, m := range members {
		responses[i] = m.ToMemberResponse()
	}

	return responses, nil
}
