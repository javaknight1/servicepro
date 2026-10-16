package services

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/javaknight1/servicepro/backend/internal/models"
)

type inviteFixture struct {
	db                   *gorm.DB
	svc                  *InvitationService
	tenantID, userID     uuid.UUID
	oldRole, invitedRole uuid.UUID
}

func setupInviteFixture(t *testing.T) *inviteFixture {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	sqlDB, err := db.DB()
	require.NoError(t, err)
	sqlDB.SetMaxOpenConns(1)

	createSQLiteTableForModel(t, db, &models.OrganizationInvitation{}, "")
	createSQLiteTableForModel(t, db, &models.User{}, "")
	createSQLiteTableForModel(t, db, &models.Role{}, "")
	createSQLiteTableForModel(t, db, &models.TenantUser{}, "UNIQUE (tenant_id, user_id)")

	f := &inviteFixture{db: db, svc: NewInvitationService(db, nil, ""),
		tenantID: uuid.New(), userID: uuid.New(), oldRole: uuid.New(), invitedRole: uuid.New()}
	now := time.Now()
	for _, r := range []struct {
		id   uuid.UUID
		name string
	}{{f.oldRole, "technician"}, {f.invitedRole, "manager"}} {
		require.NoError(t, db.Exec(`INSERT INTO roles (id, name, hierarchy_level, created_at, updated_at) VALUES (?, ?, 50, ?, ?)`,
			r.id.String(), r.name, now, now).Error)
	}
	require.NoError(t, db.Exec(`INSERT INTO users (id, email, created_at, updated_at) VALUES (?, 'pat@example.com', ?, ?)`,
		f.userID.String(), now, now).Error)
	return f
}

func (f *inviteFixture) invite(t *testing.T, token string, expires time.Time) {
	t.Helper()
	require.NoError(t, f.db.Exec(
		`INSERT INTO organization_invitations (id, tenant_id, email, role_id, token, status, invited_by, expires_at, created_at, updated_at)
		 VALUES (?, ?, 'Pat@Example.com', ?, ?, 'pending', ?, ?, ?, ?)`,
		uuid.New().String(), f.tenantID.String(), f.invitedRole.String(), token, uuid.New().String(), expires, time.Now(), time.Now()).Error)
}

// T226: a re-invited former member stayed deactivated with the old role
// (only accepted_at was updated).
func TestAcceptInvitation_ReactivatesFormerMember(t *testing.T) {
	f := setupInviteFixture(t)
	require.NoError(t, f.db.Exec(
		`INSERT INTO tenant_users (id, tenant_id, user_id, role_id, is_active, created_at, updated_at) VALUES (?, ?, ?, ?, false, ?, ?)`,
		uuid.New().String(), f.tenantID.String(), f.userID.String(), f.oldRole.String(), time.Now(), time.Now()).Error)
	f.invite(t, "tok-1", time.Now().Add(24*time.Hour))

	require.NoError(t, f.svc.AcceptInvitation(context.Background(), "tok-1", f.userID))

	var member models.TenantUser
	require.NoError(t, f.db.First(&member, "tenant_id = ? AND user_id = ?", f.tenantID, f.userID).Error)
	assert.True(t, member.IsActive)
	assert.Equal(t, f.invitedRole, member.RoleID)
	assert.NotNil(t, member.AcceptedAt)

	var inv models.OrganizationInvitation
	require.NoError(t, f.db.First(&inv, "token = ?", "tok-1").Error)
	assert.Equal(t, models.InvitationStatusAccepted, inv.Status)

	assert.ErrorIs(t, f.svc.AcceptInvitation(context.Background(), "tok-1", f.userID), ErrInvitationAlreadyUsed)
}

func TestAcceptInvitationOnRegistration_CreatesMembership(t *testing.T) {
	f := setupInviteFixture(t)
	f.invite(t, "tok-2", time.Now().Add(24*time.Hour))

	require.NoError(t, f.svc.AcceptInvitationOnRegistration(context.Background(), "tok-2", f.userID))
	var member models.TenantUser
	require.NoError(t, f.db.First(&member, "tenant_id = ? AND user_id = ?", f.tenantID, f.userID).Error)
	assert.Equal(t, f.invitedRole, member.RoleID)
}

func TestAcceptInvitation_ExpiredIsRecorded(t *testing.T) {
	f := setupInviteFixture(t)
	f.invite(t, "tok-3", time.Now().Add(-time.Hour))

	assert.ErrorIs(t, f.svc.AcceptInvitation(context.Background(), "tok-3", f.userID), ErrInvitationExpired)
	var inv models.OrganizationInvitation
	require.NoError(t, f.db.First(&inv, "token = ?", "tok-3").Error)
	assert.Equal(t, models.InvitationStatusExpired, inv.Status, "the expired status is saved")

	var members int64
	require.NoError(t, f.db.Model(&models.TenantUser{}).Count(&members).Error)
	assert.Zero(t, members)
}
