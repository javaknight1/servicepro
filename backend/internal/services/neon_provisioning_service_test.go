package services

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/javaknight1/servicepro/backend/config"
	"github.com/javaknight1/servicepro/backend/internal/tenancy"
	"github.com/javaknight1/servicepro/backend/pkg/database"
)

// openProvisioningTestDB connects to the migrated Postgres test database
// (TEST_DATABASE_URL, as in internal/repository) or skips.
func openProvisioningTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		url = "postgres://postgres:postgres@localhost:5432/servicepro_test?sslmode=disable"
	}
	db, _, err := database.NewGormDB(&config.DatabaseConfig{URL: url, MaxOpenConns: 5, MaxIdleConns: 1, ConnMaxLifetime: time.Hour})
	if err != nil {
		t.Skipf("Skipping - database not available: %v", err)
	}
	t.Cleanup(func() {
		if sqlDB, err := db.DB(); err == nil {
			_ = sqlDB.Close()
		}
	})
	require.NoError(t, tenancy.Register(db))
	return db
}

type provisionFixture struct {
	db  *gorm.DB
	svc *NeonProvisioningService
	ctx context.Context
	tag string
}

func newProvisionFixture(t *testing.T) *provisionFixture {
	t.Helper()
	db := openProvisioningTestDB(t)
	tag := uuid.NewString()[:8]
	invitations := NewInvitationService(db, nil, "http://localhost")
	gate := &RegistrationGate{Mode: RegistrationModeInviteOnly, InviteCodes: NewInviteCodeService(db), Invitations: invitations}
	t.Cleanup(func() {
		like := "%" + tag + "%"
		_ = db.Exec(`DELETE FROM organization_invitations WHERE email LIKE ?`, like).Error
		_ = db.Exec(`DELETE FROM tenants WHERE slug LIKE ?`, like).Error
		_ = db.Exec(`DELETE FROM invite_codes WHERE note LIKE ?`, like).Error
		_ = db.Exec(`DELETE FROM users WHERE email LIKE ?`, like).Error
	})
	return &provisionFixture{db: db, svc: NewNeonProvisioningService(db, gate, invitations), ctx: context.Background(), tag: tag}
}

func (f *provisionFixture) email(name string) string { return name + "-" + f.tag + "@example.com" }

func (f *provisionFixture) identity(email string) NeonIdentity {
	return NeonIdentity{AuthUserID: "neon-" + uuid.NewString(), Email: email, EmailVerified: true}
}

func (f *provisionFixture) insertUser(t *testing.T, email string, authUserID *string) uuid.UUID {
	t.Helper()
	var id uuid.UUID
	require.NoError(t, f.db.Raw(`INSERT INTO users (email, password_hash, auth_user_id) VALUES (?, 'x', ?) RETURNING id`, email, authUserID).Row().Scan(&id))
	return id
}

func TestNeonProvision_RequiresVerifiedEmail(t *testing.T) {
	f := newProvisionFixture(t)
	f.insertUser(t, f.email("victim"), nil)

	id := f.identity(f.email("victim"))
	id.EmailVerified = false
	_, err := f.svc.Provision(f.ctx, id, nil, nil)
	require.ErrorIs(t, err, ErrNeonEmailNotVerified)
}

func TestNeonProvision_LinksExistingUserByEmailAndIsIdempotent(t *testing.T) {
	f := newProvisionFixture(t)
	existing := f.insertUser(t, f.email("owner"), nil)
	id := f.identity(strings.ToUpper(f.email("owner"))) // case-insensitive match

	res, err := f.svc.Provision(f.ctx, id, nil, nil)
	require.NoError(t, err)
	require.Equal(t, existing, res.UserID)
	require.True(t, res.Linked)

	again, err := f.svc.Provision(f.ctx, id, nil, nil)
	require.NoError(t, err)
	require.Equal(t, existing, again.UserID)
	require.False(t, again.Linked || again.Created)
}

func TestNeonProvision_RefusesEmailLinkedToAnotherIdentity(t *testing.T) {
	f := newProvisionFixture(t)
	f.insertUser(t, f.email("taken"), strPtr("neon-someone-else-"+f.tag))

	_, err := f.svc.Provision(f.ctx, f.identity(f.email("taken")), nil, nil)
	require.ErrorIs(t, err, ErrAuthIdentityConflict)
}

func TestNeonProvision_NewUserNeedsAnInvite(t *testing.T) {
	f := newProvisionFixture(t)

	_, err := f.svc.Provision(f.ctx, f.identity(f.email("stranger")), nil, nil)
	require.ErrorIs(t, err, ErrRegistrationClosed)

	_, err = f.svc.Provision(f.ctx, f.identity(f.email("stranger")), strPtr("NOT-A-CODE"), nil)
	require.ErrorIs(t, err, ErrInvalidInviteCode)

	var count int64
	require.NoError(t, f.db.Raw(`SELECT COUNT(*) FROM users WHERE email = ?`, f.email("stranger")).Scan(&count).Error)
	require.Zero(t, count)
}

func TestNeonProvision_NewUserWithInviteCode(t *testing.T) {
	f := newProvisionFixture(t)
	code, _, err := NewInviteCodeService(f.db).Create(f.ctx, 1, 0, nil, strPtr("test "+f.tag))
	require.NoError(t, err)

	id := f.identity(f.email("invited"))
	res, err := f.svc.Provision(f.ctx, id, &code, nil)
	require.NoError(t, err)
	require.True(t, res.Created)

	var authUserID string
	var verified bool
	require.NoError(t, f.db.Raw(`SELECT auth_user_id, email_verified FROM users WHERE id = ?`, res.UserID).Row().Scan(&authUserID, &verified))
	require.Equal(t, id.AuthUserID, authUserID)
	require.True(t, verified)

	// Single-use code is spent.
	_, err = f.svc.Provision(f.ctx, f.identity(f.email("second")), &code, nil)
	require.ErrorIs(t, err, ErrInvalidInviteCode)
}

func TestNeonProvision_NewUserWithInvitationJoinsTenant(t *testing.T) {
	f := newProvisionFixture(t)
	inviter := f.insertUser(t, f.email("inviter"), nil)
	tenantID := uuid.New()
	require.NoError(t, f.db.Exec(`INSERT INTO tenants (id, name, slug, owner_id) VALUES (?, ?, ?, ?)`,
		tenantID, "T "+f.tag, "t-"+f.tag, inviter).Error)
	token := "inv-" + uuid.NewString()
	require.NoError(t, f.db.Exec(`INSERT INTO organization_invitations (tenant_id, email, role_id, token, invited_by, expires_at)
		VALUES (?, ?, (SELECT id FROM roles WHERE name = 'user' LIMIT 1), ?, ?, NOW() + INTERVAL '1 day')`,
		tenantID, f.email("tech"), token, inviter).Error)

	res, err := f.svc.Provision(f.ctx, f.identity(f.email("tech")), nil, &token)
	require.NoError(t, err)
	require.True(t, res.Created)
	require.True(t, res.InvitationAccepted)

	var members int64
	require.NoError(t, f.db.Raw(`SELECT COUNT(*) FROM tenant_users WHERE tenant_id = ? AND user_id = ? AND is_active`, tenantID, res.UserID).Scan(&members).Error)
	require.Equal(t, int64(1), members)

	// Someone else holding the same token is not admitted.
	_, err = f.svc.Provision(f.ctx, f.identity(f.email("thief")), nil, &token)
	require.Error(t, err)
}
