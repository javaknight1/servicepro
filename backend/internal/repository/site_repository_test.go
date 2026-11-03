package repository

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/javaknight1/servicepro/backend/internal/models"
)

// requireTable skips the test when the test database hasn't run the
// migration that creates table (a shared DB behind this branch).
func requireTable(tb testing.TB, db *gorm.DB, table string) {
	tb.Helper()
	var exists bool
	require.NoError(tb, db.Raw(`SELECT to_regclass(?) IS NOT NULL`, "public."+table).Scan(&exists).Error)
	if !exists {
		tb.Skipf("Skipping - table %s not migrated in the test database", table)
	}
}

// createTestCustomer inserts a customer into tt's tenant.
func createTestCustomer(tb testing.TB, db *gorm.DB, tt *testTenant) uuid.UUID {
	tb.Helper()
	id := uuid.New()
	require.NoError(tb, db.WithContext(tt.Ctx).Create(&models.Customer{
		ID: id, FirstName: "Pat", LastName: "Site", Email: "pat-" + id.String()[:12] + "@example.com",
		PhonePrimary: "555-555-5555", BillingAddressStreet: "1 Main St", BillingAddressCity: "Columbus",
		BillingAddressState: "OH", BillingAddressZip: "43215", CustomerType: models.CustomerTypeCommercial,
		Status: models.CustomerStatusActive,
	}).Error)
	return id
}

// T186: every site query is tenant-scoped; another tenant's site is "not found".
func TestSiteRepository_CrossTenant(t *testing.T) {
	db := openTestDB(t)
	requireTable(t, db, "service_locations")
	a, b := createTestTenant(t, db), createTestTenant(t, db)
	custA, custB := createTestCustomer(t, db, a), createTestCustomer(t, db, b)
	repo := NewSiteRepository(db)

	site := &models.Site{CustomerID: custA, Name: "Harborview Lofts", AddressStreet: "220 W Spring St", IsActive: true}
	// The tenant always comes from the caller: a TenantID on the value is overwritten.
	site.TenantID = b.ID
	require.NoError(t, repo.Create(a.Ctx, a.ID, site))
	assert.Equal(t, a.ID, site.TenantID)

	got, err := repo.GetForCustomer(a.Ctx, a.ID, custA, site.ID)
	require.NoError(t, err)
	assert.Equal(t, "Harborview Lofts", got.Name)

	// Tenant B sees nothing, by ID or by customer, and can't change it.
	_, err = repo.Get(b.Ctx, b.ID, site.ID)
	require.ErrorIs(t, err, ErrSiteNotFound)
	_, err = repo.GetForCustomer(b.Ctx, b.ID, custA, site.ID)
	require.ErrorIs(t, err, ErrSiteNotFound)
	list, err := repo.ListByCustomer(b.Ctx, b.ID, custA, true)
	require.NoError(t, err)
	assert.Empty(t, list)
	require.ErrorIs(t, repo.Update(b.Ctx, b.ID, &models.Site{ID: site.ID, CustomerID: custA, Name: "pwned"}), ErrSiteNotFound)
	require.ErrorIs(t, repo.Delete(b.Ctx, b.ID, custA, site.ID), ErrSiteNotFound)

	// Under the wrong customer of the right tenant it is not found either.
	_, err = repo.GetForCustomer(b.Ctx, b.ID, custB, site.ID)
	require.ErrorIs(t, err, ErrSiteNotFound)

	got, err = repo.Get(a.Ctx, a.ID, site.ID)
	require.NoError(t, err)
	assert.Equal(t, "Harborview Lofts", got.Name, "unchanged by tenant B")

	got.IsActive = false
	require.NoError(t, repo.Update(a.Ctx, a.ID, got))
	active, err := repo.ListByCustomer(a.Ctx, a.ID, custA, false)
	require.NoError(t, err)
	assert.Empty(t, active, "inactive site hidden")
	require.NoError(t, repo.Delete(a.Ctx, a.ID, custA, site.ID))
	_, err = repo.Get(a.Ctx, a.ID, site.ID)
	require.ErrorIs(t, err, ErrSiteNotFound)
}
