package repository

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/javaknight1/servicepro/backend/internal/models"
)

// T257: /customers/stats returns active_customers (the dashboard reads it)
// and counts only the caller's tenant.
func TestCustomerRepository_GetSearchStats(t *testing.T) {
	a := newJobFixture(t)
	b := newJobFixtureOn(t, a.db)

	a.createTestCustomer(t)
	a.createTestCustomer(t)
	prospect := a.createTestCustomer(t)
	repoA := NewCustomerRepository(a.db).WithContext(a.tenant.Ctx)
	prospect.Status = models.CustomerStatusProspect
	require.NoError(t, repoA.Update(prospect))

	stats, err := repoA.GetSearchStats()
	require.NoError(t, err)
	assert.Equal(t, int64(3), stats.TotalCustomers)
	assert.Equal(t, int64(2), stats.ActiveCustomers)
	assert.Equal(t, int64(1), stats.Prospects)
	assert.Equal(t, int64(2), stats.ByStatus["active"])

	statsB, err := NewCustomerRepository(b.db).WithContext(b.tenant.Ctx).GetSearchStats()
	require.NoError(t, err)
	assert.Zero(t, statsB.TotalCustomers, "tenant B sees none of A's customers")
	assert.Zero(t, statsB.ActiveCustomers)
}
