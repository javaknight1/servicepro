package services

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base32"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/javaknight1/servicepro/backend/internal/models"
)

// Registration modes (REGISTRATION_MODE).
const (
	RegistrationModeOpen       = "open"
	RegistrationModeInviteOnly = "invite_only"
)

var (
	// ErrRegistrationClosed is returned when invite_only mode is active and
	// the request carries neither an invite code nor an invitation.
	ErrRegistrationClosed = errors.New("registration requires a valid invite code or invitation")
	// ErrInvalidInviteCode is returned when the supplied invite code is
	// unknown, revoked, expired or used up.
	ErrInvalidInviteCode = errors.New("invite code is not valid")
	// ErrInvalidInvitation is returned when the supplied tenant invitation is
	// unknown, no longer pending, expired or addressed to another email.
	ErrInvalidInvitation = errors.New("invitation is not valid for this email")
	// ErrInviteCodeNotFound is returned when an invite code doesn't exist.
	ErrInviteCodeNotFound = errors.New("invite code not found")
)

// InviteCodeService manages platform invite codes.
type InviteCodeService struct {
	db *gorm.DB
}

// NewInviteCodeService creates an invite code service.
func NewInviteCodeService(db *gorm.DB) *InviteCodeService {
	return &InviteCodeService{db: db}
}

// HashInviteCode returns the stored form of a code. Codes are compared
// case-insensitively and ignoring dashes/spaces so they are easy to type.
func HashInviteCode(code string) string {
	normalized := strings.ToUpper(strings.NewReplacer("-", "", " ", "").Replace(strings.TrimSpace(code)))
	sum := sha256.Sum256([]byte(normalized))
	return hex.EncodeToString(sum[:])
}

func newInviteCode() (string, error) {
	b := make([]byte, 10) // 80 bits
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	raw := base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(b) // 16 chars
	return raw[0:4] + "-" + raw[4:8] + "-" + raw[8:12] + "-" + raw[12:16], nil
}

// Create generates a new code. maxUses 0 = unlimited; expiresIn 0 = never.
// The plaintext code is returned once and never stored.
func (s *InviteCodeService) Create(ctx context.Context, maxUses int, expiresIn time.Duration, createdBy *uuid.UUID, note *string) (string, *models.InviteCode, error) {
	if maxUses < 0 {
		return "", nil, fmt.Errorf("max uses must be >= 0")
	}
	code, err := newInviteCode()
	if err != nil {
		return "", nil, fmt.Errorf("generate invite code: %w", err)
	}
	ic := &models.InviteCode{
		ID:        uuid.New(),
		CodeHash:  HashInviteCode(code),
		MaxUses:   maxUses,
		CreatedBy: createdBy,
		Note:      note,
	}
	if expiresIn > 0 {
		t := time.Now().Add(expiresIn)
		ic.ExpiresAt = &t
	}
	if err := s.db.WithContext(ctx).Create(ic).Error; err != nil {
		return "", nil, fmt.Errorf("create invite code: %w", err)
	}
	return code, ic, nil
}

// List returns invite codes, newest first.
func (s *InviteCodeService) List(ctx context.Context, limit int) ([]models.InviteCode, error) {
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	var codes []models.InviteCode
	err := s.db.WithContext(ctx).Order("created_at DESC").Limit(limit).Find(&codes).Error
	return codes, err
}

// Revoke disables a code.
func (s *InviteCodeService) Revoke(ctx context.Context, id uuid.UUID) error {
	res := s.db.WithContext(ctx).Model(&models.InviteCode{}).
		Where("id = ? AND revoked_at IS NULL", id).
		Update("revoked_at", time.Now())
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return ErrInviteCodeNotFound
	}
	return nil
}

// Consume atomically uses one registration of a code. It fails with
// ErrInvalidInviteCode when the code is empty, unknown, revoked, expired or
// used up.
func (s *InviteCodeService) Consume(ctx context.Context, code string) error {
	if strings.TrimSpace(code) == "" {
		return ErrInvalidInviteCode
	}
	res := s.db.WithContext(ctx).Model(&models.InviteCode{}).
		Where("code_hash = ? AND revoked_at IS NULL AND (expires_at IS NULL OR expires_at > ?) AND (max_uses = 0 OR uses < max_uses)",
			HashInviteCode(code), time.Now()).
		Updates(map[string]interface{}{"uses": gorm.Expr("uses + 1"), "updated_at": time.Now()})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return ErrInvalidInviteCode
	}
	return nil
}

// Release gives back a use consumed for a registration that then failed.
func (s *InviteCodeService) Release(ctx context.Context, code string) {
	_ = s.db.WithContext(ctx).Model(&models.InviteCode{}).
		Where("code_hash = ? AND uses > 0", HashInviteCode(code)).
		Update("uses", gorm.Expr("uses - 1")).Error
}

// RegistrationGate enforces REGISTRATION_MODE on POST /auth/register.
type RegistrationGate struct {
	Mode        string
	InviteCodes *InviteCodeService
	Invitations *InvitationService
}

// Admit decides whether a registration may proceed. It returns a release
// function to call if the registration subsequently fails (so a consumed
// invite-code use is returned), or an error:
//
//   - ErrInvalidInviteCode when an invite code was supplied but is not valid
//     (and no valid invitation admitted the request);
//   - ErrInvalidInvitation when only an invitation was supplied and it is not
//     a pending, unexpired invitation for this email;
//   - ErrRegistrationClosed when neither was supplied.
//
// In invite_only mode the request must carry either a pending, unexpired
// tenant invitation addressed to this email, or a valid invite code.
func (g *RegistrationGate) Admit(ctx context.Context, email string, inviteCode, invitationToken *string) (func(), error) {
	noop := func() {}
	if g == nil || g.Mode != RegistrationModeInviteOnly {
		return noop, nil
	}
	hasInvitation := invitationToken != nil && strings.TrimSpace(*invitationToken) != ""
	hasCode := inviteCode != nil && strings.TrimSpace(*inviteCode) != ""

	if hasInvitation && g.Invitations != nil {
		if g.Invitations.IsPendingInvitationFor(ctx, *invitationToken, email) {
			return noop, nil
		}
	}
	if hasCode {
		if g.InviteCodes == nil {
			return noop, ErrInvalidInviteCode
		}
		err := g.InviteCodes.Consume(ctx, *inviteCode)
		if err == nil {
			code := *inviteCode
			return func() { g.InviteCodes.Release(context.WithoutCancel(ctx), code) }, nil
		}
		return noop, err
	}
	if hasInvitation {
		return noop, ErrInvalidInvitation
	}
	return noop, ErrRegistrationClosed
}
