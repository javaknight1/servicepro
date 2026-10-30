package repository

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// T159: suppression is case-insensitive and idempotent (webhooks redeliver).
func TestEmailSuppressionRepository(t *testing.T) {
	db := openTestDB(t)
	repo := NewEmailSuppressionRepository(db)
	ctx := context.Background()
	addr := "Bounce-" + uuid.NewString() + "@Example.test"

	ok, err := repo.IsSuppressed(ctx, addr)
	require.NoError(t, err)
	assert.False(t, ok)

	require.NoError(t, repo.Suppress(ctx, addr, "bounced", "msg_1"))
	require.NoError(t, repo.Suppress(ctx, addr, "complained", "msg_2"), "redelivery is a no-op")

	ok, err = repo.IsSuppressed(ctx, " "+addr+" ")
	require.NoError(t, err)
	assert.True(t, ok)
	var n int64
	require.NoError(t, db.Raw(`SELECT count(*) FROM email_suppressions WHERE lower(email) = lower(?)`, addr).Scan(&n).Error)
	assert.EqualValues(t, 1, n)
}
