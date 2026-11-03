package repository

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// T160 cross-tenant: profile pictures are visible only between users who
// share an active tenant.
func TestUsersShareTenant(t *testing.T) {
	db := openTestDB(t)
	repo := NewTenantRepository(db)
	a, b := createTestTenant(t, db), createTestTenant(t, db)
	memberA := a.addMember(t, db)
	ctx := context.Background()

	shared, err := repo.UsersShareTenant(ctx, a.OwnerID, memberA)
	require.NoError(t, err)
	assert.True(t, shared)

	shared, err = repo.UsersShareTenant(ctx, a.OwnerID, b.OwnerID)
	require.NoError(t, err)
	assert.False(t, shared, "different tenants")

	require.NoError(t, db.Exec(`UPDATE tenant_users SET is_active = false WHERE user_id = ?`, memberA).Error)
	shared, err = repo.UsersShareTenant(ctx, a.OwnerID, memberA)
	require.NoError(t, err)
	assert.False(t, shared, "inactive membership")
}
