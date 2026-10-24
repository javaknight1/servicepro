package repository

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestUserRepository_GetIDByAuthUserID(t *testing.T) {
	db := openTestDB(t)
	tt := createTestTenant(t, db)
	repo := &UserRepositoryGorm{db: db}

	authID := "neon-" + uuid.NewString()
	require.NoError(t, db.Exec(`UPDATE users SET auth_user_id = ? WHERE id = ?`, authID, tt.OwnerID).Error)

	got, err := repo.GetIDByAuthUserID(authID)
	require.NoError(t, err)
	require.Equal(t, tt.OwnerID, got)

	_, err = repo.GetIDByAuthUserID("neon-unknown-" + uuid.NewString())
	require.ErrorIs(t, err, gorm.ErrRecordNotFound)

	// A soft-deleted user no longer resolves.
	require.NoError(t, db.Exec(`UPDATE users SET deleted_at = now() WHERE id = ?`, tt.OwnerID).Error)
	_, err = repo.GetIDByAuthUserID(authID)
	require.ErrorIs(t, err, gorm.ErrRecordNotFound)
}

func TestUserRepository_AuthUserIDUniqueAmongLiveUsers(t *testing.T) {
	db := openTestDB(t)
	tt := createTestTenant(t, db)
	other := tt.addMember(t, db)

	authID := "neon-" + uuid.NewString()
	require.NoError(t, db.Exec(`UPDATE users SET auth_user_id = ? WHERE id = ?`, authID, tt.OwnerID).Error)
	require.Error(t, db.Exec(`UPDATE users SET auth_user_id = ? WHERE id = ?`, authID, other).Error)
}
