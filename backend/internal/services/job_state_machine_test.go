package services

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/javaknight1/servicepro/backend/internal/models"
	"github.com/javaknight1/servicepro/backend/pkg/validators"
)

func jobServiceWith(job *models.Job) (*JobService, *MockJobRepository) {
	repo := new(MockJobRepository)
	repo.On("GetByID", job.ID).Return(job, nil)
	repo.On("UpdateWithTransition", mock.AnythingOfType("*models.Job"), mock.AnythingOfType("*models.JobStatusTransition")).Return(nil)
	return NewJobService(repo, new(MockCustomerRepository), nil), repo
}

// T124 / audit 7.H7: the validator's rules had no "new", so jobs (created as
// new) could not be cancelled.
func TestTransitionStatus_CancelNewJob(t *testing.T) {
	job := createTestJob()
	job.Status = models.JobStatusNew
	svc, _ := jobServiceWith(job)
	reason := "customer called"

	resp, err := svc.TransitionStatus(job.ID, models.JobStatusCancelled, string(models.ReasonCancellation), &reason, job.CreatedBy, models.UserRoleAdmin)
	require.NoError(t, err)
	assert.Equal(t, models.JobStatusCancelled, resp.Status)
}

// Starting an en_route job is allowed and stamps ActualStartAt.
func TestTransitionStatus_StartFromEnRoute(t *testing.T) {
	job := createTestJob()
	job.Status = models.JobStatusEnRoute
	svc, repo := jobServiceWith(job)

	resp, err := svc.TransitionStatus(job.ID, models.JobStatusInProgress, string(models.ReasonStartWork), nil, job.CreatedBy, models.UserRoleAdmin)
	require.NoError(t, err)
	assert.Equal(t, models.JobStatusInProgress, resp.Status)
	assert.NotNil(t, job.ActualStartAt)
	repo.AssertCalled(t, "UpdateWithTransition", mock.Anything, mock.MatchedBy(func(tr *models.JobStatusTransition) bool {
		return tr.FromStatus == models.JobStatusEnRoute && tr.ToStatus == models.JobStatusInProgress
	}))
}

// A job moved to in_progress without ActualStartAt (the old UpdateJob path)
// can still be completed.
func TestTransitionStatus_CompleteWithoutActualStart(t *testing.T) {
	job := createTestJob()
	job.Status = models.JobStatusInProgress
	job.ActualStartAt = nil
	svc, _ := jobServiceWith(job)

	resp, err := svc.TransitionStatus(job.ID, models.JobStatusCompleted, "", nil, job.CreatedBy, models.UserRoleAdmin)
	require.NoError(t, err)
	assert.Equal(t, models.JobStatusCompleted, resp.Status)
	assert.NotNil(t, job.ActualStartAt)
	assert.NotNil(t, job.ActualEndAt)
}

// UpdateJob no longer changes status (no history, no timestamps).
func TestUpdateJob_RejectsStatusChange(t *testing.T) {
	job := createTestJob()
	svc, repo := jobServiceWith(job)
	to := models.JobStatusCompleted

	_, err := svc.UpdateJob(job.ID, &models.UpdateJobRequest{Status: &to}, job.CreatedBy, models.UserRoleAdmin)
	assert.ErrorIs(t, err, validators.ErrStatusNotUpdatable)
	repo.AssertNotCalled(t, "Update", mock.Anything)
	repo.AssertNotCalled(t, "UpdateWithTransition", mock.Anything, mock.Anything)
	assert.Equal(t, models.JobStatusScheduled, job.Status)
}

// Transitions the state machine forbids are refused.
func TestJobTransitions_InvalidMovesRejected(t *testing.T) {
	job := createTestJob()
	job.Status = models.JobStatusCancelled
	svc, _ := jobServiceWith(job)

	_, err := svc.TransitionStatus(job.ID, models.JobStatusInProgress, "", nil, job.CreatedBy, models.UserRoleAdmin)
	assert.ErrorIs(t, err, models.ErrInvalidStatusTransition)
	_, err = svc.TransitionStatus(job.ID, models.JobStatusScheduled, "other", nil, job.CreatedBy, models.UserRoleAdmin)
	assert.ErrorIs(t, err, models.ErrInvalidStatusTransition)
}

// T267: every job response carries its allowed next statuses from the one
// state machine; the UI's status actions read them instead of a copy.
func TestJobResponse_AllowedTransitions(t *testing.T) {
	job := createTestJob()
	job.Status = models.JobStatusNew
	assert.ElementsMatch(t, []models.JobStatus{models.JobStatusScheduled, models.JobStatusCancelled}, job.ToResponse().AllowedTransitions)

	svc, _ := jobServiceWith(job)
	resp, err := svc.TransitionStatus(job.ID, models.JobStatusScheduled, "", nil, job.CreatedBy, models.UserRoleAdmin)
	require.NoError(t, err)
	assert.ElementsMatch(t, models.GetAllowedTransitions(models.JobStatusScheduled), resp.AllowedTransitions,
		"the transition response carries the new status's next steps")

	job.Status = models.JobStatusCancelled
	allowed := job.ToResponse().AllowedTransitions
	assert.NotNil(t, allowed, "terminal status serializes as [] not null")
	assert.Empty(t, allowed)
}
