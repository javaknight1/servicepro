package services

import (
	"context"
	"errors"
	"fmt"
	"log"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/javaknight1/servicepro/backend/internal/models"
	"github.com/javaknight1/servicepro/backend/internal/repository"
)

var (
	ErrInvalidTenantName     = errors.New("invalid tenant name")
	ErrInvalidTenantSlug     = errors.New("invalid tenant slug")
	ErrCannotRemoveOwner     = errors.New("cannot remove tenant owner")
	ErrCannotChangeOwnerRole = errors.New("cannot change owner's role")
)

// slugRegex validates tenant slugs (alphanumeric and hyphens only)
var slugRegex = regexp.MustCompile(`^[a-z0-9][a-z0-9-]*[a-z0-9]$|^[a-z0-9]$`)

// PermissionCacheInvalidator interface for invalidating permission cache
type PermissionCacheInvalidator interface {
	InvalidateUserPermissions(ctx context.Context, userID uuid.UUID) error
}

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
			log.Printf("[WARN] Failed to invalidate permission cache for user %s: %v", userID, err)
		} else {
			log.Printf("[TENANT] Invalidated permission cache for user %s", userID)
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

	if err := s.tenantRepo.Create(ctx, tenant); err != nil {
		if errors.Is(err, repository.ErrTenantSlugExists) {
			return nil, ErrInvalidTenantSlug
		}
		return nil, fmt.Errorf("failed to create tenant: %w", err)
	}

	log.Printf("[TENANT] Created tenant: ID=%s, Name=%s, OwnerID=%s", tenant.ID, tenant.Name, ownerID)

	// Add owner as admin member
	adminRoleID, _ := uuid.Parse("00000000-0000-0000-0000-000000000002") // admin role
	now := time.Now()
	membership := &models.TenantUser{
		ID:         uuid.New(),
		TenantID:   tenant.ID,
		UserID:     ownerID,
		RoleID:     adminRoleID,
		AcceptedAt: &now,
		IsActive:   true,
	}

	log.Printf("[TENANT] Adding owner as member: MembershipID=%s, TenantID=%s, UserID=%s, RoleID=%s",
		membership.ID, membership.TenantID, membership.UserID, membership.RoleID)

	if err := s.tenantRepo.AddMember(ctx, membership); err != nil {
		log.Printf("[TENANT] ERROR adding member: %v", err)
		// Rollback tenant creation
		_ = s.tenantRepo.Delete(ctx, tenant.ID)
		return nil, fmt.Errorf("failed to add owner as member: %w", err)
	}

	log.Printf("[TENANT] Successfully added owner as member")

	// Invalidate permission cache for the owner so they get their new permissions
	s.invalidatePermissionCache(ctx, ownerID)

	// Verify the membership was created
	belongs, err := s.tenantRepo.UserBelongsToTenant(ctx, ownerID, tenant.ID)
	if err != nil {
		log.Printf("[TENANT] WARNING: Could not verify membership: %v", err)
	} else {
		log.Printf("[TENANT] Membership verification: UserBelongsToTenant=%v", belongs)
	}

	return tenant, nil
}

// GetTenant retrieves a tenant by ID
func (s *TenantService) GetTenant(ctx context.Context, id uuid.UUID) (*models.Tenant, error) {
	return s.tenantRepo.GetByID(ctx, id)
}

// GetTenantBySlug retrieves a tenant by slug
func (s *TenantService) GetTenantBySlug(ctx context.Context, slug string) (*models.Tenant, error) {
	return s.tenantRepo.GetBySlug(ctx, strings.ToLower(slug))
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

// ListTenants lists all tenants with pagination
func (s *TenantService) ListTenants(ctx context.Context, page, pageSize int) (*models.ListTenantsResponse, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	tenants, total, err := s.tenantRepo.List(ctx, page, pageSize)
	if err != nil {
		return nil, err
	}

	responses := make([]models.TenantResponse, len(tenants))
	for i, t := range tenants {
		responses[i] = t.ToResponse()
	}

	totalPages := int(total) / pageSize
	if int(total)%pageSize > 0 {
		totalPages++
	}

	return &models.ListTenantsResponse{
		Tenants:    responses,
		Total:      total,
		Page:       page,
		PageSize:   pageSize,
		TotalPages: totalPages,
	}, nil
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
func (s *TenantService) AddMember(ctx context.Context, tenantID uuid.UUID, req *models.AddMemberRequest, invitedBy uuid.UUID) (*models.TenantMemberResponse, error) {
	// Find user by email
	user, err := s.userRepo.GetByEmail(req.Email)
	if err != nil {
		return nil, fmt.Errorf("user not found with email: %s", req.Email)
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
func (s *TenantService) RemoveMember(ctx context.Context, tenantID, userID uuid.UUID) error {
	// Check if user is the owner
	tenant, err := s.tenantRepo.GetByID(ctx, tenantID)
	if err != nil {
		return err
	}
	if tenant.OwnerID == userID {
		return ErrCannotRemoveOwner
	}

	if err := s.tenantRepo.RemoveMember(ctx, tenantID, userID); err != nil {
		return err
	}

	// Invalidate permission cache for the removed member
	s.invalidatePermissionCache(ctx, userID)

	return nil
}

// UpdateMemberRole updates a member's role
func (s *TenantService) UpdateMemberRole(ctx context.Context, tenantID, userID, roleID uuid.UUID) error {
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

// UserBelongsToTenant checks if a user is a member of a tenant
func (s *TenantService) UserBelongsToTenant(ctx context.Context, userID, tenantID uuid.UUID) (bool, error) {
	return s.tenantRepo.UserBelongsToTenant(ctx, userID, tenantID)
}
