package repository

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/javaknight1/servicepro/backend/internal/models"
	"github.com/javaknight1/servicepro/backend/internal/tenancy"
)

// T174: attachments are tenant-scoped (another tenant's are "not found" and
// can't be changed), and JobVisible applies the technician row filter.
func TestJobAttachmentRepository_CrossTenant(t *testing.T) {
	db := openTestDB(t)
	requireTable(t, db, "job_attachments")
	a, b := createTestTenant(t, db), createTestTenant(t, db)
	custA := createTestCustomer(t, db, a)
	newJob := func() uuid.UUID {
		job := &models.Job{ID: uuid.New(), CustomerID: custA, Title: "Nightly cleaning", JobType: models.JobTypeMaintenance,
			Status: models.JobStatusScheduled, Priority: models.JobPriorityNormal, CreatedBy: a.OwnerID}
		require.NoError(t, db.WithContext(a.Ctx).Create(job).Error)
		return job.ID
	}
	jobID, otherJob := newJob(), newJob()

	repo := NewJobAttachmentRepository(db)
	thumb := "k_thumb.jpg"
	att := &models.JobAttachment{JobID: jobID, UploadedBy: &a.OwnerID, Kind: models.JobAttachmentPhoto,
		ObjectKey: "k.jpg", ThumbKey: &thumb, OriginalFilename: "IMG.jpg", ContentType: "image/jpeg", SizeBytes: 10,
		TenantID: b.ID}
	require.NoError(t, repo.Create(a.Ctx, a.ID, att))
	assert.Equal(t, a.ID, att.TenantID, "tenant comes from the caller")

	got, err := repo.Get(a.Ctx, a.ID, jobID, att.ID)
	require.NoError(t, err)
	require.NotNil(t, got.Uploader, "uploader is loaded for the name")
	n, err := repo.CountByJob(a.Ctx, a.ID, jobID)
	require.NoError(t, err)
	assert.Equal(t, int64(1), n)

	// Tenant B: the job isn't visible and the attachment doesn't exist.
	ok, err := repo.JobVisible(b.Ctx, b.ID, jobID)
	require.NoError(t, err)
	assert.False(t, ok)
	ok, err = repo.JobVisible(b.Ctx, a.ID, jobID) // even naming tenant A's ID under B's context
	require.NoError(t, err)
	assert.False(t, ok)
	_, err = repo.Get(b.Ctx, b.ID, jobID, att.ID)
	require.ErrorIs(t, err, ErrJobAttachmentNotFound)
	list, err := repo.ListByJob(b.Ctx, b.ID, jobID)
	require.NoError(t, err)
	assert.Empty(t, list)
	n, err = repo.CountByJob(b.Ctx, b.ID, jobID)
	require.NoError(t, err)
	assert.Zero(t, n)
	pwned := "pwned"
	require.ErrorIs(t, repo.UpdateCaption(b.Ctx, b.ID, jobID, att.ID, &pwned), ErrJobAttachmentNotFound)
	require.ErrorIs(t, repo.Delete(b.Ctx, b.ID, jobID, att.ID), ErrJobAttachmentNotFound)

	// Under another job of the right tenant it is not found either.
	_, err = repo.Get(a.Ctx, a.ID, otherJob, att.ID)
	require.ErrorIs(t, err, ErrJobAttachmentNotFound)

	// The technician row filter: visible only while actively assigned.
	tech := a.addMember(t, db)
	techCtx := tenancy.WithAssignedJobsOnly(a.Ctx, tech)
	ok, err = repo.JobVisible(techCtx, a.ID, jobID)
	require.NoError(t, err)
	assert.False(t, ok, "unassigned job is invisible to a technician")
	require.NoError(t, db.WithContext(a.Ctx).Create(&models.JobAssignment{
		JobID: jobID, UserID: tech, Role: "technician", AssignedAt: time.Now().Unix(),
	}).Error)
	ok, err = repo.JobVisible(techCtx, a.ID, jobID)
	require.NoError(t, err)
	assert.True(t, ok)
	ok, err = repo.JobVisible(techCtx, a.ID, otherJob)
	require.NoError(t, err)
	assert.False(t, ok)
	ok, err = repo.JobVisible(a.Ctx, a.ID, otherJob)
	require.NoError(t, err)
	assert.True(t, ok, "office staff see every job")

	caption := "Lobby floor"
	require.NoError(t, repo.UpdateCaption(a.Ctx, a.ID, jobID, att.ID, &caption))
	list, err = repo.ListByJob(a.Ctx, a.ID, jobID)
	require.NoError(t, err)
	require.Len(t, list, 1)
	assert.Equal(t, caption, *list[0].Caption)
	require.NoError(t, repo.Delete(a.Ctx, a.ID, jobID, att.ID))
	_, err = repo.Get(a.Ctx, a.ID, jobID, att.ID)
	require.ErrorIs(t, err, ErrJobAttachmentNotFound)

	// Deleted jobs are not visible.
	require.NoError(t, db.WithContext(tenancy.WithSystem(context.Background())).
		Exec(`UPDATE jobs SET deleted_at = NOW() WHERE id = ?`, otherJob).Error)
	ok, err = repo.JobVisible(a.Ctx, a.ID, otherJob)
	require.NoError(t, err)
	assert.False(t, ok)
}
