package repository

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/javaknight1/servicepro/backend/internal/models"
	"github.com/javaknight1/servicepro/backend/internal/tenancy"
)

// T176: sign-offs are tenant-scoped (another tenant's are "not found" and
// can't be replaced, reported or emailed), one per job, locked once
// emailed, and the email claim lets only one sender through.
func TestJobSignoffRepository_CrossTenantAndClaim(t *testing.T) {
	db := openTestDB(t)
	requireTable(t, db, "job_signoffs")
	a, b := createTestTenant(t, db), createTestTenant(t, db)
	custA := createTestCustomer(t, db, a)
	newJob := func() uuid.UUID {
		job := &models.Job{ID: uuid.New(), CustomerID: custA, Title: "Nightly cleaning", JobType: models.JobTypeMaintenance,
			Status: models.JobStatusInProgress, Priority: models.JobPriorityNormal, CreatedBy: a.OwnerID}
		require.NoError(t, db.WithContext(a.Ctx).Create(job).Error)
		return job.ID
	}
	jobID, otherJob := newJob(), newJob()
	repo := NewJobSignoffRepository(db)

	signoff := func(key string) *models.JobSignoff {
		return &models.JobSignoff{JobID: jobID, SignerName: "Ray Burke", SignatureKey: key, SignatureWidth: 900,
			SignatureHeight: 300, SignedAt: time.Now(), CapturedBy: &a.OwnerID, TenantID: b.ID}
	}
	first := signoff("sig-1.png")
	replaced, err := repo.Save(a.Ctx, a.ID, first)
	require.NoError(t, err)
	assert.Nil(t, replaced)
	assert.Equal(t, a.ID, first.TenantID, "tenant comes from the caller")

	got, err := repo.Get(a.Ctx, a.ID, jobID)
	require.NoError(t, err)
	require.NotNil(t, got.Capturer)
	status, err := repo.JobStatus(a.Ctx, a.ID, jobID)
	require.NoError(t, err)
	assert.Equal(t, models.JobStatusInProgress, status)
	contacts, err := repo.Contacts(a.Ctx, a.ID, jobID)
	require.NoError(t, err)
	assert.NotEmpty(t, contacts.CustomerEmail)
	src, err := repo.ReportSource(a.Ctx, a.ID, jobID)
	require.NoError(t, err)
	assert.Equal(t, a.ID, src.Tenant.ID)
	require.NotNil(t, src.Job.Customer)

	// Tenant B: nothing exists, nothing can be changed.
	ok, err := repo.JobVisible(b.Ctx, b.ID, jobID)
	require.NoError(t, err)
	assert.False(t, ok)
	ok, err = repo.JobVisible(b.Ctx, a.ID, jobID) // naming tenant A under B's context
	require.NoError(t, err)
	assert.False(t, ok)
	_, err = repo.Get(b.Ctx, b.ID, jobID)
	require.ErrorIs(t, err, ErrJobSignoffNotFound)
	_, err = repo.JobStatus(b.Ctx, b.ID, jobID)
	require.ErrorIs(t, err, ErrJobSignoffNotFound)
	_, err = repo.Contacts(b.Ctx, b.ID, jobID)
	require.ErrorIs(t, err, ErrJobSignoffNotFound)
	_, err = repo.ReportSource(b.Ctx, b.ID, jobID)
	require.ErrorIs(t, err, ErrJobSignoffNotFound)
	require.ErrorIs(t, repo.SetReport(b.Ctx, b.ID, jobID, "x.pdf", time.Now()), ErrJobSignoffNotFound)
	claimed, err := repo.ClaimEmail(b.Ctx, b.ID, jobID, true, time.Now(), time.Now())
	require.NoError(t, err)
	assert.False(t, claimed)
	// Tenant B can't attach a sign-off to tenant A's job.
	_, err = repo.Save(b.Ctx, b.ID, signoff("sig-b.png"))
	require.ErrorIs(t, err, ErrJobSignoffNotFound)
	again, err := repo.Get(a.Ctx, a.ID, jobID)
	require.NoError(t, err)
	assert.Equal(t, "sig-1.png", again.SignatureKey)

	// The technician row filter.
	tech := a.addMember(t, db)
	techCtx := tenancy.WithAssignedJobsOnly(a.Ctx, tech)
	ok, err = repo.JobVisible(techCtx, a.ID, jobID)
	require.NoError(t, err)
	assert.False(t, ok)
	_, err = repo.ReportSource(techCtx, a.ID, jobID)
	require.ErrorIs(t, err, ErrJobSignoffNotFound)

	// Replacing: same row, report cleared, old row returned.
	require.NoError(t, repo.SetReport(a.Ctx, a.ID, jobID, "report.pdf", time.Now()))
	second := signoff("sig-2.png")
	replaced, err = repo.Save(a.Ctx, a.ID, second)
	require.NoError(t, err)
	require.NotNil(t, replaced)
	assert.Equal(t, "sig-1.png", replaced.SignatureKey)
	assert.Equal(t, "report.pdf", *replaced.ReportKey)
	got, err = repo.Get(a.Ctx, a.ID, jobID)
	require.NoError(t, err)
	assert.Equal(t, first.ID, got.ID)
	assert.Equal(t, "sig-2.png", got.SignatureKey)
	assert.Nil(t, got.ReportKey)

	// Claim: one sender at a time; finished sends block a non-resend.
	now := time.Now()
	claimed, err = repo.ClaimEmail(a.Ctx, a.ID, jobID, false, now, now.Add(-10*time.Minute))
	require.NoError(t, err)
	assert.True(t, claimed)
	claimed, err = repo.ClaimEmail(a.Ctx, a.ID, jobID, false, now, now.Add(-10*time.Minute))
	require.NoError(t, err)
	assert.False(t, claimed, "a second sender is refused while the claim is fresh")
	claimed, err = repo.ClaimEmail(a.Ctx, a.ID, jobID, false, now.Add(time.Hour), now.Add(time.Minute))
	require.NoError(t, err)
	assert.True(t, claimed, "a stale claim is taken over")
	require.NoError(t, repo.FinishEmail(a.Ctx, a.ID, jobID, []string{"pm@example.com", "ray@example.com"}, now))
	got, err = repo.Get(a.Ctx, a.ID, jobID)
	require.NoError(t, err)
	assert.Equal(t, []string{"pm@example.com", "ray@example.com"}, got.Recipients())
	assert.Equal(t, 1, got.ReportEmailCount)
	assert.Nil(t, got.ReportEmailClaimedAt)
	claimed, err = repo.ClaimEmail(a.Ctx, a.ID, jobID, false, now, now.Add(-10*time.Minute))
	require.NoError(t, err)
	assert.False(t, claimed, "already emailed")
	claimed, err = repo.ClaimEmail(a.Ctx, a.ID, jobID, true, now, now.Add(-10*time.Minute))
	require.NoError(t, err)
	assert.True(t, claimed, "an explicit resend")
	require.NoError(t, repo.ReleaseEmail(a.Ctx, a.ID, jobID))

	// Emailed: the sign-off is locked.
	_, err = repo.Save(a.Ctx, a.ID, signoff("sig-3.png"))
	require.ErrorIs(t, err, ErrJobSignoffLocked)

	// The other job has none.
	_, err = repo.Get(a.Ctx, a.ID, otherJob)
	require.ErrorIs(t, err, ErrJobSignoffNotFound)
}
