package services

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/javaknight1/servicepro/backend/internal/models"
	"github.com/javaknight1/servicepro/backend/pkg/clients/email"
	"github.com/javaknight1/servicepro/backend/pkg/clients/logging"
)

// Invitation errors
var (
	ErrInvitationNotFound      = errors.New("invitation not found")
	ErrInvitationExpired       = errors.New("invitation has expired")
	ErrInvitationAlreadyUsed   = errors.New("invitation has already been used")
	ErrUserAlreadyMember       = errors.New("user is already a member of this organization")
	ErrPendingInvitationExists = errors.New("a pending invitation already exists for this email")
)

// InvitationService handles organization invitations
type InvitationService struct {
	db          *gorm.DB
	emailClient email.Client
	appURL      string
}

// NewInvitationService creates a new invitation service
func NewInvitationService(db *gorm.DB, emailClient email.Client, appURL string) *InvitationService {
	return &InvitationService{
		db:          db,
		emailClient: emailClient,
		appURL:      appURL,
	}
}

// InviteMember invites a user to join an organization
func (s *InvitationService) InviteMember(ctx context.Context, tenantID uuid.UUID, email string, roleID uuid.UUID, invitedBy uuid.UUID) (*models.InvitationResponse, error) {
	// Check if user already exists
	var existingUser models.User
	userExists := s.db.WithContext(ctx).Where("email = ?", email).First(&existingUser).Error == nil

	// Get tenant details
	var tenant models.Tenant
	if err := s.db.WithContext(ctx).First(&tenant, tenantID).Error; err != nil {
		return nil, fmt.Errorf("failed to get tenant: %w", err)
	}

	// Get inviter details
	var inviter models.User
	if err := s.db.WithContext(ctx).First(&inviter, invitedBy).Error; err != nil {
		return nil, fmt.Errorf("failed to get inviter: %w", err)
	}

	// Get role details
	var role models.Role
	if err := s.db.WithContext(ctx).First(&role, roleID).Error; err != nil {
		return nil, fmt.Errorf("failed to get role: %w", err)
	}

	if userExists {
		// Check if user is already a member
		var existingMember models.TenantUser
		if err := s.db.WithContext(ctx).Where("tenant_id = ? AND user_id = ?", tenantID, existingUser.ID).First(&existingMember).Error; err == nil {
			return nil, ErrUserAlreadyMember
		}

		// Check for existing pending invitation
		var existingInvitation models.OrganizationInvitation
		if err := s.db.WithContext(ctx).
			Where("tenant_id = ? AND email = ? AND status = ?", tenantID, email, models.InvitationStatusPending).
			First(&existingInvitation).Error; err == nil {
			return nil, ErrPendingInvitationExists
		}

		// Create invitation record - don't add to tenant_users until they accept
		token, err := generateToken()
		if err != nil {
			return nil, fmt.Errorf("failed to generate token: %w", err)
		}

		invitation := &models.OrganizationInvitation{
			TenantID:  tenantID,
			Email:     email,
			RoleID:    roleID,
			Token:     token,
			Status:    models.InvitationStatusPending,
			InvitedBy: invitedBy,
			ExpiresAt: time.Now().Add(7 * 24 * time.Hour), // 7 days
		}

		if err := s.db.WithContext(ctx).Create(invitation).Error; err != nil {
			return nil, fmt.Errorf("failed to create invitation: %w", err)
		}

		// Send email to existing user
		inviterName := inviter.Email
		if inviter.FirstName != nil && inviter.LastName != nil {
			inviterName = *inviter.FirstName + " " + *inviter.LastName
		}

		acceptURL := fmt.Sprintf("%s/invitations/accept?token=%s", s.appURL, token)
		go s.sendExistingUserInviteEmail(ctx, email, tenant.Name, inviterName, role.Name, acceptURL)

		return &models.InvitationResponse{
			ID:             invitation.ID,
			TenantID:       tenantID,
			TenantName:     tenant.Name,
			Email:          email,
			RoleID:         roleID,
			RoleName:       role.Name,
			Status:         models.InvitationStatusPending,
			InvitedByName:  inviterName,
			ExpiresAt:      invitation.ExpiresAt,
			CreatedAt:      invitation.CreatedAt,
			UserExists:     true,
			UserRegistered: true,
		}, nil
	}

	// User doesn't exist - create invitation for registration
	// Check for existing pending invitation
	var existingInvitation models.OrganizationInvitation
	if err := s.db.WithContext(ctx).
		Where("tenant_id = ? AND email = ? AND status = ?", tenantID, email, models.InvitationStatusPending).
		First(&existingInvitation).Error; err == nil {
		// Update existing invitation
		existingInvitation.RoleID = roleID
		existingInvitation.InvitedBy = invitedBy
		existingInvitation.ExpiresAt = time.Now().Add(7 * 24 * time.Hour)
		if err := s.db.WithContext(ctx).Save(&existingInvitation).Error; err != nil {
			return nil, fmt.Errorf("failed to update invitation: %w", err)
		}

		// Resend email
		inviterName := inviter.Email
		if inviter.FirstName != nil && inviter.LastName != nil {
			inviterName = *inviter.FirstName + " " + *inviter.LastName
		}

		registerURL := fmt.Sprintf("%s/register?invitation=%s&email=%s", s.appURL, existingInvitation.Token, email)
		go s.sendNewUserInviteEmail(ctx, email, tenant.Name, inviterName, role.Name, registerURL)

		resp := existingInvitation.ToResponse(false)
		return &resp, nil
	}

	// Create new invitation
	token, err := generateToken()
	if err != nil {
		return nil, fmt.Errorf("failed to generate token: %w", err)
	}

	invitation := &models.OrganizationInvitation{
		TenantID:  tenantID,
		Email:     email,
		RoleID:    roleID,
		Token:     token,
		Status:    models.InvitationStatusPending,
		InvitedBy: invitedBy,
		ExpiresAt: time.Now().Add(7 * 24 * time.Hour), // 7 days
	}

	if err := s.db.WithContext(ctx).Create(invitation).Error; err != nil {
		return nil, fmt.Errorf("failed to create invitation: %w", err)
	}

	// Load relations for response
	invitation.Tenant = &tenant
	invitation.Role = &role
	invitation.Inviter = &inviter

	// Send email to new user
	inviterName := inviter.Email
	if inviter.FirstName != nil && inviter.LastName != nil {
		inviterName = *inviter.FirstName + " " + *inviter.LastName
	}

	registerURL := fmt.Sprintf("%s/register?invitation=%s&email=%s", s.appURL, token, email)
	go s.sendNewUserInviteEmail(ctx, email, tenant.Name, inviterName, role.Name, registerURL)

	resp := invitation.ToResponse(false)
	return &resp, nil
}

// AcceptInvitation accepts an invitation for an existing user
func (s *InvitationService) AcceptInvitation(ctx context.Context, token string, userID uuid.UUID) error {
	var invitation models.OrganizationInvitation
	if err := s.db.WithContext(ctx).Where("token = ?", token).First(&invitation).Error; err != nil {
		return ErrInvitationNotFound
	}

	if invitation.Status != models.InvitationStatusPending {
		return ErrInvitationAlreadyUsed
	}

	if invitation.IsExpired() {
		invitation.Status = models.InvitationStatusExpired
		s.db.WithContext(ctx).Save(&invitation)
		return ErrInvitationExpired
	}

	// Get user to verify email matches
	var user models.User
	if err := s.db.WithContext(ctx).First(&user, userID).Error; err != nil {
		return fmt.Errorf("user not found")
	}

	if !strings.EqualFold(user.Email, invitation.Email) {
		return fmt.Errorf("email mismatch: user email %s does not match invitation email %s", user.Email, invitation.Email)
	}

	now := time.Now()

	// Check if tenant_user already exists
	var existingTenantUser models.TenantUser
	if err := s.db.WithContext(ctx).Where("tenant_id = ? AND user_id = ?", invitation.TenantID, userID).First(&existingTenantUser).Error; err == nil {
		// Update existing record
		existingTenantUser.AcceptedAt = &now
		if err := s.db.WithContext(ctx).Save(&existingTenantUser).Error; err != nil {
			return fmt.Errorf("failed to update tenant user: %w", err)
		}
	} else {
		// Create new TenantUser
		tenantUser := &models.TenantUser{
			TenantID:   invitation.TenantID,
			UserID:     userID,
			RoleID:     invitation.RoleID,
			InvitedBy:  &invitation.InvitedBy,
			InvitedAt:  &invitation.CreatedAt,
			AcceptedAt: &now,
			IsActive:   true,
		}

		if err := s.db.WithContext(ctx).Create(tenantUser).Error; err != nil {
			return fmt.Errorf("failed to create tenant user: %w", err)
		}
	}

	// Update invitation status
	invitation.Status = models.InvitationStatusAccepted
	invitation.AcceptedAt = &now
	if err := s.db.WithContext(ctx).Save(&invitation).Error; err != nil {
		return fmt.Errorf("failed to update invitation: %w", err)
	}

	return nil
}

// GetInvitationByToken retrieves an invitation by token
func (s *InvitationService) GetInvitationByToken(ctx context.Context, token string) (*models.OrganizationInvitation, error) {
	var invitation models.OrganizationInvitation
	if err := s.db.WithContext(ctx).
		Preload("Tenant").
		Preload("Role").
		Preload("Inviter").
		Where("token = ?", token).
		First(&invitation).Error; err != nil {
		return nil, ErrInvitationNotFound
	}

	return &invitation, nil
}

// AcceptInvitationOnRegistration processes invitation acceptance during registration
func (s *InvitationService) AcceptInvitationOnRegistration(ctx context.Context, token string, userID uuid.UUID) error {
	var invitation models.OrganizationInvitation
	if err := s.db.WithContext(ctx).Where("token = ?", token).First(&invitation).Error; err != nil {
		return ErrInvitationNotFound
	}

	if invitation.Status != models.InvitationStatusPending {
		return ErrInvitationAlreadyUsed
	}

	if invitation.IsExpired() {
		invitation.Status = models.InvitationStatusExpired
		s.db.WithContext(ctx).Save(&invitation)
		return ErrInvitationExpired
	}

	// Create TenantUser
	now := time.Now()
	tenantUser := &models.TenantUser{
		TenantID:   invitation.TenantID,
		UserID:     userID,
		RoleID:     invitation.RoleID,
		InvitedBy:  &invitation.InvitedBy,
		InvitedAt:  &invitation.CreatedAt,
		AcceptedAt: &now,
		IsActive:   true,
	}

	if err := s.db.WithContext(ctx).Create(tenantUser).Error; err != nil {
		return fmt.Errorf("failed to create tenant user: %w", err)
	}

	// Update invitation status
	invitation.Status = models.InvitationStatusAccepted
	invitation.AcceptedAt = &now
	if err := s.db.WithContext(ctx).Save(&invitation).Error; err != nil {
		return fmt.Errorf("failed to update invitation: %w", err)
	}

	return nil
}

// GetPendingInvitationsForTenant returns all pending invitations for a tenant
func (s *InvitationService) GetPendingInvitationsForTenant(ctx context.Context, tenantID uuid.UUID) ([]models.InvitationResponse, error) {
	var invitations []models.OrganizationInvitation
	if err := s.db.WithContext(ctx).
		Preload("Role").
		Preload("Inviter").
		Where("tenant_id = ? AND status = ?", tenantID, models.InvitationStatusPending).
		Find(&invitations).Error; err != nil {
		return nil, fmt.Errorf("failed to get invitations: %w", err)
	}

	responses := make([]models.InvitationResponse, len(invitations))
	for i, inv := range invitations {
		// Check if user has registered (exists in users table)
		var existingUser models.User
		userExists := s.db.WithContext(ctx).Where("email = ?", inv.Email).First(&existingUser).Error == nil
		responses[i] = inv.ToResponse(userExists)
	}

	return responses, nil
}

// CancelInvitation cancels a pending invitation
func (s *InvitationService) CancelInvitation(ctx context.Context, invitationID uuid.UUID) error {
	result := s.db.WithContext(ctx).
		Model(&models.OrganizationInvitation{}).
		Where("id = ? AND status = ?", invitationID, models.InvitationStatusPending).
		Update("status", models.InvitationStatusCancelled)

	if result.Error != nil {
		return fmt.Errorf("failed to cancel invitation: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return ErrInvitationNotFound
	}

	return nil
}

// ResendInvitation resends an invitation email
func (s *InvitationService) ResendInvitation(ctx context.Context, invitationID uuid.UUID) error {
	var invitation models.OrganizationInvitation
	if err := s.db.WithContext(ctx).
		Preload("Tenant").
		Preload("Role").
		Preload("Inviter").
		Where("id = ? AND status = ?", invitationID, models.InvitationStatusPending).
		First(&invitation).Error; err != nil {
		return ErrInvitationNotFound
	}

	// Extend expiration
	invitation.ExpiresAt = time.Now().Add(7 * 24 * time.Hour)
	if err := s.db.WithContext(ctx).Save(&invitation).Error; err != nil {
		return fmt.Errorf("failed to update invitation: %w", err)
	}

	// Check if user exists now
	var existingUser models.User
	userExists := s.db.WithContext(ctx).Where("email = ?", invitation.Email).First(&existingUser).Error == nil

	inviterName := invitation.Inviter.Email
	if invitation.Inviter.FirstName != nil && invitation.Inviter.LastName != nil {
		inviterName = *invitation.Inviter.FirstName + " " + *invitation.Inviter.LastName
	}

	if userExists {
		acceptURL := fmt.Sprintf("%s/invitations/accept?token=%s", s.appURL, invitation.Token)
		go s.sendExistingUserInviteEmail(ctx, invitation.Email, invitation.Tenant.Name, inviterName, invitation.Role.Name, acceptURL)
	} else {
		registerURL := fmt.Sprintf("%s/register?invitation=%s&email=%s", s.appURL, invitation.Token, invitation.Email)
		go s.sendNewUserInviteEmail(ctx, invitation.Email, invitation.Tenant.Name, inviterName, invitation.Role.Name, registerURL)
	}

	return nil
}

// sendExistingUserInviteEmail sends invitation email to an existing user
func (s *InvitationService) sendExistingUserInviteEmail(ctx context.Context, toEmail, orgName, inviterName, roleName, acceptURL string) {
	if s.emailClient == nil {
		logging.Warn(ctx, "[INVITATION] Email client not configured, skipping email", map[string]any{"to_email": toEmail})
		return
	}

	err := s.emailClient.SendOrganizationInviteEmail(ctx, toEmail, orgName, inviterName, roleName, acceptURL, true)
	if err != nil {
		logging.Error(ctx, "[INVITATION] Failed to send invite email", map[string]any{"to_email": toEmail, "error": err})
	} else {
		logging.Info(ctx, "[INVITATION] Sent organization invite email to existing user", map[string]any{"to_email": toEmail, "org_name": orgName})
	}
}

// sendNewUserInviteEmail sends invitation email to a new user
func (s *InvitationService) sendNewUserInviteEmail(ctx context.Context, toEmail, orgName, inviterName, roleName, registerURL string) {
	if s.emailClient == nil {
		logging.Warn(ctx, "[INVITATION] Email client not configured, skipping email", map[string]any{"to_email": toEmail})
		return
	}

	err := s.emailClient.SendOrganizationInviteEmail(ctx, toEmail, orgName, inviterName, roleName, registerURL, false)
	if err != nil {
		logging.Error(ctx, "[INVITATION] Failed to send invite email", map[string]any{"to_email": toEmail, "error": err})
	} else {
		logging.Info(ctx, "[INVITATION] Sent registration invite email to new user", map[string]any{"to_email": toEmail, "org_name": orgName})
	}
}

// generateToken generates a secure random token
func generateToken() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}
