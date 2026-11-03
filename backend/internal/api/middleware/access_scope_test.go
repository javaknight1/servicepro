package middleware

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/javaknight1/servicepro/backend/internal/permissions"
	"github.com/javaknight1/servicepro/backend/internal/tenancy"
)

// fakeTenantPerms grants perms[tenant] to every user.
type fakeTenantPerms struct {
	perms map[uuid.UUID][]string
	err   error
}

func (f fakeTenantPerms) CheckPermission(_ context.Context, _, tenantID uuid.UUID, name string) (bool, error) {
	if f.err != nil {
		return false, f.err
	}
	for _, p := range f.perms[tenantID] {
		if p == name {
			return true, nil
		}
	}
	return false, nil
}

// T035: the access scope comes from permissions in the request's tenant, never
// from role names, and a role in one tenant says nothing about another.
func TestAccessScope(t *testing.T) {
	user := uuid.New()
	office, field := uuid.New(), uuid.New()
	scope := AccessScope(fakeTenantPerms{perms: map[uuid.UUID][]string{
		office: {permissions.JobsViewAll, permissions.InvoicesRead},
		field:  {permissions.JobsRead, permissions.JobsList, permissions.JobsUpdate},
	}})

	ctx, err := scope(context.Background(), user, office)
	require.NoError(t, err)
	_, limited := tenancy.AssignedJobsUser(ctx)
	assert.False(t, limited, "office staff see every job")
	assert.False(t, tenancy.MoneyHidden(ctx))

	ctx, err = scope(context.Background(), user, field)
	require.NoError(t, err)
	got, limited := tenancy.AssignedJobsUser(ctx)
	assert.True(t, limited, "technicians see assigned jobs only")
	assert.Equal(t, user, got)
	assert.True(t, tenancy.MoneyHidden(ctx))

	// No permissions at all (e.g. an unknown custom role): fail closed.
	ctx, err = scope(context.Background(), user, uuid.New())
	require.NoError(t, err)
	_, limited = tenancy.AssignedJobsUser(ctx)
	assert.True(t, limited)
	assert.True(t, tenancy.MoneyHidden(ctx))

	_, err = AccessScope(fakeTenantPerms{err: errors.New("db down")})(context.Background(), user, office)
	assert.Error(t, err)
}
