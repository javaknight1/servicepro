package services

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/javaknight1/servicepro/backend/internal/models"
	"github.com/javaknight1/servicepro/backend/internal/repository"
)

func realJobService(f *isolationFixture) *JobService {
	return NewJobService(
		repository.NewJobRepository(f.db).WithContext(f.ctxA),
		repository.NewCustomerRepository(f.db).WithContext(f.ctxA),
		f.db.WithContext(f.ctxA),
	)
}

func jobHistory(t *testing.T, f *isolationFixture, jobID interface{}) []models.JobStatusTransition {
	t.Helper()
	var rows []models.JobStatusTransition
	require.NoError(t, f.db.Where("job_id = ?", jobID).Order("transitioned_at, created_at").Find(&rows).Error)
	return rows
}

// Audit 7.M1 / 7.M2: the creation history row always failed (from_status ""
// into a NOT NULL enum, error swallowed) and assignments got AssignedAt = 0.
func TestCreateJob_RecordsHistoryAndAssignmentTime(t *testing.T) {
	f := setupIsolation(t)
	svc := realJobService(f)

	resp, err := svc.CreateJob(&models.CreateJobRequest{
		CustomerID: f.custA.ID, Title: "Fix AC", JobType: models.JobTypeRepair,
		Assignments: []models.CreateJobAssignmentRequest{{UserID: f.userA}},
	}, f.userA)
	require.NoError(t, err)

	rows := jobHistory(t, f, resp.ID)
	require.Len(t, rows, 1)
	assert.Equal(t, models.JobStatusNew, rows[0].FromStatus)
	assert.Equal(t, models.JobStatusNew, rows[0].ToStatus)

	var a models.JobAssignment
	require.NoError(t, f.db.Where("job_id = ?", resp.ID).First(&a).Error)
	assert.Greater(t, a.AssignedAt, int64(1_600_000_000), "AssignedAt is now, not 1970")
}

// Status and history are written in one transaction: if the history insert
// fails, the status change is rolled back.
func TestJobTransition_StatusAndHistoryAreAtomic(t *testing.T) {
	f := setupIsolation(t)
	svc := realJobService(f)
	resp, err := svc.CreateJob(&models.CreateJobRequest{CustomerID: f.custA.ID, Title: "Fix AC", JobType: models.JobTypeRepair}, f.userA)
	require.NoError(t, err)

	_, err = svc.TransitionStatus(resp.ID, models.JobStatusScheduled, "schedule_change", nil, f.userA, models.UserRoleAdmin)
	require.NoError(t, err)
	assert.Len(t, jobHistory(t, f, resp.ID), 2)

	require.NoError(t, f.db.Exec("DROP TABLE job_status_transitions").Error)
	_, err = svc.TransitionStatus(resp.ID, models.JobStatusEnRoute, "other", nil, f.userA, models.UserRoleAdmin)
	require.Error(t, err)

	var job models.Job
	require.NoError(t, f.db.WithContext(f.ctxA).First(&job, "id = ?", resp.ID).Error)
	assert.Equal(t, models.JobStatusScheduled, job.Status, "status change rolled back with the failed history insert")
}
