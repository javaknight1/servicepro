package services

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/javaknight1/servicepro/backend/internal/models"
	"github.com/javaknight1/servicepro/backend/internal/repository"
)

// T186: customer sites.

func setupSites(t *testing.T) (*isolationFixture, *SiteService) {
	t.Helper()
	f := setupIsolation(t)
	createSQLiteTableForModel(t, f.db, &models.Site{}, "")
	return f, NewSiteService(f.db, repository.NewSiteRepository(f.db))
}

func newSiteReq(name string) *models.CreateSiteRequest {
	return &models.CreateSiteRequest{
		Name: name, AddressStreet: "220 W Spring St", AddressCity: "Columbus", AddressState: "oh", AddressZip: "43215",
	}
}

func TestSiteService_CRUD(t *testing.T) {
	f, svc := setupSites(t)

	site, err := svc.Create(f.ctxA, f.custA.ID, newSiteReq("Harborview Lofts"))
	require.NoError(t, err)
	assert.Equal(t, "OH", site.AddressState)
	assert.True(t, site.IsActive, "active by default")
	assert.Equal(t, f.custA.ID, site.CustomerID)

	got, err := svc.Get(f.ctxA, f.custA.ID, site.ID)
	require.NoError(t, err)
	assert.Equal(t, "Harborview Lofts", got.Name)

	inactive := false
	notes := "Badge at loading dock"
	updated, err := svc.Update(f.ctxA, f.custA.ID, site.ID, &models.UpdateSiteRequest{AccessNotes: &notes, IsActive: &inactive})
	require.NoError(t, err)
	assert.False(t, updated.IsActive, "false is written, not skipped")
	require.NotNil(t, updated.AccessNotes)
	assert.Equal(t, notes, *updated.AccessNotes)

	list, err := svc.List(f.ctxA, f.custA.ID, false)
	require.NoError(t, err)
	assert.Empty(t, list, "inactive sites are hidden by default")
	list, err = svc.List(f.ctxA, f.custA.ID, true)
	require.NoError(t, err)
	assert.Len(t, list, 1)

	require.NoError(t, svc.Delete(f.ctxA, f.custA.ID, site.ID))
	_, err = svc.Get(f.ctxA, f.custA.ID, site.ID)
	require.ErrorIs(t, err, ErrSiteNotFound)
}

func TestSiteService_CrossTenant(t *testing.T) {
	f, svc := setupSites(t)
	siteA, err := svc.Create(f.ctxA, f.custA.ID, newSiteReq("A's site"))
	require.NoError(t, err)

	// Tenant B can't reach tenant A's customer or its sites.
	_, err = svc.List(f.ctxB, f.custA.ID, true)
	require.ErrorIs(t, err, ErrCustomerNotFound)
	_, err = svc.Get(f.ctxB, f.custA.ID, siteA.ID)
	require.ErrorIs(t, err, ErrCustomerNotFound)
	_, err = svc.Create(f.ctxB, f.custA.ID, newSiteReq("intruder"))
	require.ErrorIs(t, err, ErrCustomerNotFound)
	name := "renamed"
	_, err = svc.Update(f.ctxB, f.custA.ID, siteA.ID, &models.UpdateSiteRequest{Name: &name})
	require.ErrorIs(t, err, ErrCustomerNotFound)
	require.ErrorIs(t, svc.Delete(f.ctxB, f.custA.ID, siteA.ID), ErrCustomerNotFound)

	// Nor through its own customer's URL.
	_, err = svc.Get(f.ctxB, f.custB.ID, siteA.ID)
	require.ErrorIs(t, err, ErrSiteNotFound)
	require.ErrorIs(t, svc.Delete(f.ctxB, f.custB.ID, siteA.ID), ErrSiteNotFound)

	got, err := svc.Get(f.ctxA, f.custA.ID, siteA.ID)
	require.NoError(t, err)
	assert.Equal(t, "A's site", got.Name, "unchanged")
}

func TestSiteService_SiteUnderAnotherCustomerIsNotFound(t *testing.T) {
	f, svc := setupSites(t)
	other := &models.Customer{ID: uuid.New(), FirstName: "Other", LastName: "A", Email: "other@a.test", CustomerType: "commercial", Status: "active"}
	require.NoError(t, repository.NewCustomerRepository(f.db).WithContext(f.ctxA).Create(other))
	site, err := svc.Create(f.ctxA, f.custA.ID, newSiteReq("Lofts"))
	require.NoError(t, err)

	_, err = svc.Get(f.ctxA, other.ID, site.ID)
	require.ErrorIs(t, err, ErrSiteNotFound)
}

func TestCreateJob_SiteMustBelongToCustomer(t *testing.T) {
	f, sites := setupSites(t)
	other := &models.Customer{ID: uuid.New(), FirstName: "Other", LastName: "A", Email: "other@a.test", CustomerType: "commercial", Status: "active"}
	require.NoError(t, repository.NewCustomerRepository(f.db).WithContext(f.ctxA).Create(other))

	siteA, err := sites.Create(f.ctxA, f.custA.ID, newSiteReq("Lofts"))
	require.NoError(t, err)
	siteOther, err := sites.Create(f.ctxA, other.ID, newSiteReq("Other's site"))
	require.NoError(t, err)
	siteB, err := sites.Create(f.ctxB, f.custB.ID, newSiteReq("B's site"))
	require.NoError(t, err)
	inactive := false
	retired, err := sites.Create(f.ctxA, f.custA.ID, &models.CreateSiteRequest{
		Name: "Old", AddressStreet: "1 Old St", AddressCity: "Columbus", AddressState: "OH", AddressZip: "43215", IsActive: &inactive,
	})
	require.NoError(t, err)

	svc := realJobService(f)
	req := func(siteID uuid.UUID) *models.CreateJobRequest {
		return &models.CreateJobRequest{CustomerID: f.custA.ID, SiteID: &siteID, Title: "Nightly clean", JobType: models.JobTypeMaintenance}
	}
	for name, id := range map[string]uuid.UUID{
		"another customer's site": siteOther.ID,
		"another tenant's site":   siteB.ID,
		"inactive site":           retired.ID,
		"unknown site":            uuid.New(),
	} {
		t.Run(name, func(t *testing.T) {
			_, err := svc.CreateJob(req(id), f.userA)
			require.ErrorIs(t, err, ErrSiteCustomerMismatch)
		})
	}
	var n int64
	require.NoError(t, f.db.WithContext(f.ctxA).Model(&models.Job{}).Count(&n).Error)
	assert.Zero(t, n, "no job written")

	job, err := svc.CreateJob(req(siteA.ID), f.userA)
	require.NoError(t, err)
	require.NotNil(t, job.SiteID)
	assert.Equal(t, siteA.ID, *job.SiteID)
	assert.Equal(t, "220 W Spring St", job.ServiceAddress.Street, "site address used when none given")

	// Moving the job to another customer's site is rejected too.
	_, err = svc.UpdateJob(job.ID, &models.UpdateJobRequest{SiteID: &siteOther.ID}, f.userA, models.UserRoleAdmin)
	require.ErrorIs(t, err, ErrSiteCustomerMismatch)
}
