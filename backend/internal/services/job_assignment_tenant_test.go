package services

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/javaknight1/servicepro/backend/internal/models"
	"github.com/javaknight1/servicepro/backend/internal/repository"
)

// T238: a job can only be assigned to an active member of the job's tenant.
// A user who exists but belongs to another tenant, an unknown user and an
// inactive member are all rejected with a typed 404 (it used to be a
// foreign-key violation → 500, or a silent cross-tenant assignment).

type assigneeFixture struct {
	*isolationFixture
	jobA     *models.Job
	memberA  uuid.UUID // active member of tenant A
	inactive uuid.UUID // inactive member of tenant A
	userB    uuid.UUID // active member of tenant B only
}

func setupAssignees(t *testing.T) *assigneeFixture {
	t.Helper()
	f := &assigneeFixture{
		isolationFixture: setupIsolation(t),
		memberA:          uuid.New(),
		inactive:         uuid.New(),
		userB:            uuid.New(),
	}
	f.addMember(t, f.ctxA, f.memberA, true)
	f.addMember(t, f.ctxA, f.inactive, false)
	f.addMember(t, f.ctxB, f.userB, true)

	f.jobA = &models.Job{ID: uuid.New(), CustomerID: f.custA.ID, Title: "Fix sink", Status: models.JobStatusNew, CreatedBy: f.userA}
	require.NoError(t, repository.NewJobRepository(f.db).WithContext(f.ctxA).Create(f.jobA))
	return f
}

func (f *assigneeFixture) assignmentCount(t *testing.T) int64 {
	t.Helper()
	var n int64
	require.NoError(t, f.db.Model(&models.JobAssignment{}).Where("job_id = ?", f.jobA.ID).Count(&n).Error)
	return n
}

func TestAssignTechnician_RejectsNonMembers(t *testing.T) {
	f := setupAssignees(t)
	svc := realJobService(f.isolationFixture)

	for name, userID := range map[string]uuid.UUID{
		"other tenant's user": f.userB,
		"unknown user":        uuid.New(),
		"inactive member":     f.inactive,
	} {
		t.Run(name, func(t *testing.T) {
			err := svc.AssignTechnician(f.jobA.ID, userID, "technician", f.userA, models.UserRoleAdmin)
			require.ErrorIs(t, err, ErrAssigneeNotFound)
		})
	}
	assert.Zero(t, f.assignmentCount(t), "no assignment written")

	require.NoError(t, svc.AssignTechnician(f.jobA.ID, f.memberA, "technician", f.userA, models.UserRoleAdmin))
	assert.Equal(t, int64(1), f.assignmentCount(t))
}

func TestCreateJob_RejectsAssigneeFromOtherTenant(t *testing.T) {
	f := setupAssignees(t)
	svc := realJobService(f.isolationFixture)

	_, err := svc.CreateJob(&models.CreateJobRequest{
		CustomerID: f.custA.ID, Title: "Fix AC", JobType: models.JobTypeRepair,
		Assignments: []models.CreateJobAssignmentRequest{{UserID: f.memberA}, {UserID: f.userB}},
	}, f.userA)
	require.ErrorIs(t, err, ErrAssigneeNotFound)

	var jobs int64
	require.NoError(t, f.db.WithContext(f.ctxA).Model(&models.Job{}).Count(&jobs).Error)
	assert.Equal(t, int64(1), jobs, "nothing created: only the fixture's job exists")
}

// The repository refuses the write too, whatever the caller checked.
func TestJobRepository_AddAssignmentChecksMembership(t *testing.T) {
	f := setupAssignees(t)
	repoA := repository.NewJobRepository(f.db).WithContext(f.ctxA)
	repoB := repository.NewJobRepository(f.db).WithContext(f.ctxB)

	newAssignment := func(userID uuid.UUID) *models.JobAssignment {
		return &models.JobAssignment{ID: uuid.New(), JobID: f.jobA.ID, UserID: userID, Role: "technician", AssignedAt: 1}
	}

	assert.ErrorIs(t, repoA.AddAssignment(newAssignment(f.userB)), repository.ErrAssigneeNotInTenant)
	assert.ErrorIs(t, repoA.AddAssignment(newAssignment(f.inactive)), repository.ErrAssigneeNotInTenant)
	assert.ErrorIs(t, repoB.AddAssignment(newAssignment(f.userB)), gorm.ErrRecordNotFound, "tenant B can't see tenant A's job")
	assert.Zero(t, f.assignmentCount(t))

	require.NoError(t, repoA.AddAssignment(newAssignment(f.memberA)))
	assert.Equal(t, int64(1), f.assignmentCount(t))

	ok, err := repoA.IsActiveTenantMember(uuid.Nil, f.memberA)
	require.NoError(t, err)
	assert.False(t, ok)
}
