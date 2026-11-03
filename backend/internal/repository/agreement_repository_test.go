package repository

import (
	"testing"
	"time"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/javaknight1/servicepro/backend/internal/models"
)

// T167: agreements are tenant-scoped; another tenant can't see or change them.
func TestAgreementRepository_CrossTenant(t *testing.T) {
	db := openTestDB(t)
	requireTable(t, db, "service_agreements")
	a, b := createTestTenant(t, db), createTestTenant(t, db)
	custA := createTestCustomer(t, db, a)
	site := &models.Site{CustomerID: custA, Name: "Lofts", IsActive: true}
	require.NoError(t, NewSiteRepository(db).Create(a.Ctx, a.ID, site))

	repo := NewAgreementRepository(db)
	agreement := &models.ServiceAgreement{
		CustomerID: custA, SiteID: site.ID, Title: "Nightly cleaning", Frequency: models.FrequencyWeekdays,
		StartDate: time.Date(2026, 11, 2, 0, 0, 0, 0, time.UTC), VisitStartMinutes: 18 * 60, VisitDurationMinutes: 60,
		PricePerPeriod: decimal.NewFromInt(2400), BillingPeriod: models.BillingPeriodMonth,
		JobType: models.JobTypeMaintenance, Status: models.AgreementStatusActive, CreatedBy: a.OwnerID,
		TenantID: b.ID,
	}
	require.NoError(t, repo.Create(a.Ctx, a.ID, agreement))
	assert.Equal(t, a.ID, agreement.TenantID, "tenant comes from the caller")

	got, err := repo.Get(a.Ctx, a.ID, agreement.ID)
	require.NoError(t, err)
	assert.Equal(t, "2026-11-02", got.StartDate.Format(models.DateLayout), "DATE round-trips")

	_, err = repo.Get(b.Ctx, b.ID, agreement.ID)
	require.ErrorIs(t, err, ErrAgreementNotFound)
	list, err := repo.List(b.Ctx, b.ID, nil)
	require.NoError(t, err)
	assert.Empty(t, list)
	got.Title = "pwned"
	require.ErrorIs(t, repo.Update(b.Ctx, b.ID, got), ErrAgreementNotFound)
	require.ErrorIs(t, repo.SetStatus(b.Ctx, b.ID, agreement.ID, models.AgreementStatusPaused), ErrAgreementNotFound)
	require.ErrorIs(t, repo.Delete(b.Ctx, b.ID, agreement.ID), ErrAgreementNotFound)

	got, err = repo.Get(a.Ctx, a.ID, agreement.ID)
	require.NoError(t, err)
	assert.Equal(t, "Nightly cleaning", got.Title)
	assert.Equal(t, models.AgreementStatusActive, got.Status)

	require.NoError(t, repo.SetStatus(a.Ctx, a.ID, agreement.ID, models.AgreementStatusPaused))
	paused := models.AgreementStatusPaused
	list, err = repo.List(a.Ctx, a.ID, &models.AgreementListFilter{Status: &paused})
	require.NoError(t, err)
	assert.Len(t, list, 1)
	require.NoError(t, repo.Delete(a.Ctx, a.ID, agreement.ID))
}
