package services

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/javaknight1/servicepro/backend/pkg/auth"
	"github.com/javaknight1/servicepro/backend/pkg/clients/logging"
)

var (
	// ErrNeonEmailNotVerified is returned when the Neon Auth identity's email
	// is not verified. Neon Auth allows unverified sign-ups, so linking or
	// creating an account by email before verification would let anyone claim
	// someone else's address.
	ErrNeonEmailNotVerified = errors.New("neon auth email is not verified")
	// ErrAuthIdentityConflict is returned when the email belongs to a user
	// already linked to a different Neon Auth identity.
	ErrAuthIdentityConflict = errors.New("email is linked to another sign-in")
)

// NeonIdentity is a verified Neon Auth JWT's identity.
type NeonIdentity struct {
	AuthUserID    string
	Email         string
	EmailVerified bool
}

// NeonProvisionResult says what Provision did.
type NeonProvisionResult struct {
	UserID             uuid.UUID
	Created            bool // a new users row was created
	Linked             bool // an existing user was linked to this identity
	InvitationAccepted bool
}

// NeonProvisioningService turns a first Neon Auth sign-in into a ServicePro
// user (T158). It links an existing user with the same (verified) email, or
// creates one only when the registration gate admits it — the same
// invite-only rule as POST /auth/register (T106).
type NeonProvisioningService struct {
	db          *gorm.DB
	gate        *RegistrationGate
	invitations *InvitationService
}

// NewNeonProvisioningService creates the service. gate and invitations may be nil.
func NewNeonProvisioningService(db *gorm.DB, gate *RegistrationGate, invitations *InvitationService) *NeonProvisioningService {
	return &NeonProvisioningService{db: db, gate: gate, invitations: invitations}
}

// Provision links or creates the ServicePro user for a Neon Auth identity and,
// when an invitation token is given, accepts that invitation. It is
// idempotent: an identity that is already linked just returns its user.
func (s *NeonProvisioningService) Provision(ctx context.Context, id NeonIdentity, inviteCode, invitationToken *string) (*NeonProvisionResult, error) {
	if id.AuthUserID == "" {
		return nil, errors.New("neon identity has no user id")
	}
	if !id.EmailVerified {
		return nil, ErrNeonEmailNotVerified
	}
	email := strings.ToLower(strings.TrimSpace(id.Email))
	if err := auth.ValidateEmail(email); err != nil {
		return nil, err
	}

	res, err := s.linkOrCreate(ctx, id.AuthUserID, email, inviteCode, invitationToken)
	if err != nil {
		return nil, err
	}

	if invitationToken != nil && strings.TrimSpace(*invitationToken) != "" && s.invitations != nil {
		// As in POST /auth/register: a stale or mismatched invitation does not
		// undo the sign-in; the user can be re-invited.
		if err := s.invitations.AcceptInvitationOnRegistration(ctx, *invitationToken, res.UserID); err != nil {
			logging.Info(ctx, "[NEON-AUTH] Invitation not accepted on first sign-in", map[string]any{"user_id": res.UserID, "reason": err.Error()})
		} else {
			res.InvitationAccepted = true
		}
	}
	return res, nil
}

type userLink struct {
	ID         uuid.UUID
	AuthUserID *string
}

func (s *NeonProvisioningService) linkOrCreate(ctx context.Context, authUserID, email string, inviteCode, invitationToken *string) (*NeonProvisionResult, error) {
	db := s.db.WithContext(ctx)

	// Already linked.
	var linked []userLink
	if err := db.Raw(`SELECT id, auth_user_id FROM users WHERE auth_user_id = ? AND deleted_at IS NULL`, authUserID).Scan(&linked).Error; err != nil {
		return nil, fmt.Errorf("look up linked user: %w", err)
	}
	if len(linked) > 0 {
		return &NeonProvisionResult{UserID: linked[0].ID}, nil
	}

	// An existing user with this email: link it.
	var byEmail []userLink
	if err := db.Raw(`SELECT id, auth_user_id FROM users WHERE LOWER(email) = ? AND deleted_at IS NULL`, email).Scan(&byEmail).Error; err != nil {
		return nil, fmt.Errorf("look up user by email: %w", err)
	}
	if len(byEmail) > 0 {
		u := byEmail[0]
		if u.AuthUserID != nil && *u.AuthUserID != authUserID {
			return nil, ErrAuthIdentityConflict
		}
		upd := db.Exec(`UPDATE users SET auth_user_id = ?, email_verified = true, updated_at = NOW()
			WHERE id = ? AND auth_user_id IS NULL`, authUserID, u.ID)
		if upd.Error != nil {
			return nil, fmt.Errorf("link user: %w", upd.Error)
		}
		if upd.RowsAffected == 0 {
			return nil, ErrAuthIdentityConflict // linked concurrently to another identity
		}
		return &NeonProvisionResult{UserID: u.ID, Linked: true}, nil
	}

	// A new user: only if the registration gate admits it.
	release, err := s.gate.Admit(ctx, email, inviteCode, invitationToken)
	if err != nil {
		return nil, err
	}
	var newID uuid.UUID
	// No password: this user signs in through Neon Auth only. An empty hash
	// never matches in the legacy password login.
	err = db.Raw(`INSERT INTO users (email, password_hash, email_verified, auth_user_id)
		VALUES (?, '', true, ?) RETURNING id`, email, authUserID).Row().Scan(&newID)
	if err != nil {
		release()
		return nil, fmt.Errorf("create user: %w", err)
	}
	return &NeonProvisionResult{UserID: newID, Created: true}, nil
}
