package services

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/javaknight1/servicepro/backend/internal/models"
)

func setupInviteDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	require.NoError(t, err)
	sqlDB, _ := db.DB()
	sqlDB.SetMaxOpenConns(1)
	createSQLiteTableForModel(t, db, &models.InviteCode{}, "")
	createSQLiteTableForModel(t, db, &models.OrganizationInvitation{}, "")
	return db
}

func TestInviteCode_ConsumeRules(t *testing.T) {
	svc := NewInviteCodeService(setupInviteDB(t))
	ctx := context.Background()

	code, ic, err := svc.Create(ctx, 2, time.Hour, nil, nil)
	require.NoError(t, err)
	assert.NotContains(t, ic.CodeHash, code, "plaintext is not stored")

	require.NoError(t, svc.Consume(ctx, code))
	// Case/dash-insensitive.
	require.NoError(t, svc.Consume(ctx, " "+lowerNoDash(code)+" "))
	assert.ErrorIs(t, svc.Consume(ctx, code), ErrRegistrationClosed, "max uses reached")

	svc.Release(ctx, code)
	require.NoError(t, svc.Consume(ctx, code), "released use can be reused")

	assert.ErrorIs(t, svc.Consume(ctx, "NOPE-NOPE-NOPE-NOPE"), ErrRegistrationClosed)

	revoked, ric, err := svc.Create(ctx, 0, 0, nil, nil)
	require.NoError(t, err)
	require.NoError(t, svc.Revoke(ctx, ric.ID))
	assert.ErrorIs(t, svc.Consume(ctx, revoked), ErrRegistrationClosed)
	assert.ErrorIs(t, svc.Revoke(ctx, uuid.New()), ErrInviteCodeNotFound)

	expired, eic, err := svc.Create(ctx, 1, time.Hour, nil, nil)
	require.NoError(t, err)
	past := time.Now().Add(-time.Minute)
	require.NoError(t, svc.db.Model(&models.InviteCode{}).Where("id = ?", eic.ID).Update("expires_at", past).Error)
	assert.ErrorIs(t, svc.Consume(ctx, expired), ErrRegistrationClosed)
}

func lowerNoDash(s string) string {
	out := make([]rune, 0, len(s))
	for _, r := range s {
		if r == '-' {
			continue
		}
		if r >= 'A' && r <= 'Z' {
			r += 'a' - 'A'
		}
		out = append(out, r)
	}
	return string(out)
}

func TestRegistrationGate(t *testing.T) {
	db := setupInviteDB(t)
	codes := NewInviteCodeService(db)
	invites := NewInvitationService(db, nil, "")
	ctx := context.Background()

	open := &RegistrationGate{Mode: RegistrationModeOpen}
	_, err := open.Admit(ctx, "a@x.test", nil, nil)
	assert.NoError(t, err)

	var nilGate *RegistrationGate
	_, err = nilGate.Admit(ctx, "a@x.test", nil, nil)
	assert.NoError(t, err)

	gate := &RegistrationGate{Mode: RegistrationModeInviteOnly, InviteCodes: codes, Invitations: invites}
	_, err = gate.Admit(ctx, "a@x.test", nil, nil)
	assert.True(t, errors.Is(err, ErrRegistrationClosed))

	code, _, err := codes.Create(ctx, 1, time.Hour, nil, nil)
	require.NoError(t, err)
	release, err := gate.Admit(ctx, "a@x.test", &code, nil)
	require.NoError(t, err)
	_, err = gate.Admit(ctx, "b@x.test", &code, nil)
	assert.ErrorIs(t, err, ErrRegistrationClosed, "single-use code")
	release()
	_, err = gate.Admit(ctx, "b@x.test", &code, nil)
	assert.NoError(t, err, "released after failed registration")

	inv := &models.OrganizationInvitation{
		ID: uuid.New(), TenantID: uuid.New(), Email: "Invitee@X.test", RoleID: uuid.New(), Token: "tok-1",
		Status: models.InvitationStatusPending, InvitedBy: uuid.New(), ExpiresAt: time.Now().Add(time.Hour),
	}
	require.NoError(t, db.Omit("Tenant", "Role", "Inviter").Create(inv).Error)
	tok := "tok-1"
	_, err = gate.Admit(ctx, "invitee@x.test", nil, &tok)
	assert.NoError(t, err, "pending invitation for the same email admits")
	_, err = gate.Admit(ctx, "someone-else@x.test", nil, &tok)
	assert.ErrorIs(t, err, ErrRegistrationClosed, "invitation for another email does not")
}
