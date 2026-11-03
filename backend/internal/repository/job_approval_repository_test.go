package repository

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/javaknight1/servicepro/backend/internal/models"
)

// T187: approvals are tenant-scoped; another tenant can't read, count or
// add to them.
func TestJobApprovalRepository_CrossTenant(t *testing.T) {
	db := openTestDB(t)
	requireTable(t, db, "job_approvals")
	a, b := createTestTenant(t, db), createTestTenant(t, db)
	custA := createTestCustomer(t, db, a)
	job := &models.Job{ID: uuid.New(), CustomerID: custA, Title: "Extra work", JobType: models.JobTypeRepair,
		Status: models.JobStatusNew, Priority: models.JobPriorityNormal, CreatedBy: a.OwnerID}
	require.NoError(t, db.WithContext(a.Ctx).Create(job).Error)

	repo := NewJobApprovalRepository(db)
	approval := &models.JobApproval{JobID: job.ID, ApproverEmail: "pm@example.com", Amount: decimal.NewFromInt(750),
		ApprovedAt: time.Now(), RecordedBy: a.OwnerID, TenantID: b.ID}
	require.NoError(t, repo.Create(a.Ctx, a.ID, approval))
	assert.Equal(t, a.ID, approval.TenantID, "tenant comes from the caller")

	ok, err := repo.Covers(a.Ctx, a.ID, job.ID, decimal.NewFromInt(750))
	require.NoError(t, err)
	assert.True(t, ok)
	ok, err = repo.Covers(a.Ctx, a.ID, job.ID, decimal.RequireFromString("750.01"))
	require.NoError(t, err)
	assert.False(t, ok)

	list, err := repo.ListForJob(b.Ctx, b.ID, job.ID)
	require.NoError(t, err)
	assert.Empty(t, list, "tenant B sees none of A's approvals")
	ok, err = repo.Covers(b.Ctx, b.ID, job.ID, decimal.Zero)
	require.NoError(t, err)
	assert.False(t, ok)

	// Tenant B naming A's tenant in its own context is refused by the plugin.
	err = repo.Create(b.Ctx, a.ID, &models.JobApproval{JobID: job.ID, ApproverEmail: "x@b.test",
		Amount: decimal.NewFromInt(1), ApprovedAt: time.Now(), RecordedBy: b.OwnerID})
	require.Error(t, err)

	list, err = repo.ListForJob(a.Ctx, a.ID, job.ID)
	require.NoError(t, err)
	assert.Len(t, list, 1)
}
