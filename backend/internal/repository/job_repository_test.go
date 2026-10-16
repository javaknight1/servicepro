package repository

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/javaknight1/servicepro/backend/internal/models"
)

// jobFixture is one freshly seeded tenant with a job repository scoped to it.
type jobFixture struct {
	db     *gorm.DB
	tenant *testTenant
	repo   JobRepositoryInterface
}

func newJobFixture(t *testing.T) *jobFixture {
	t.Helper()
	db := openTestDB(t)
	return newJobFixtureOn(t, db)
}

// newJobFixtureOn seeds another tenant on an existing connection.
func newJobFixtureOn(t *testing.T, db *gorm.DB) *jobFixture {
	t.Helper()
	tt := createTestTenant(t, db)
	return &jobFixture{db: db, tenant: tt, repo: NewJobRepository(db).WithContext(tt.Ctx)}
}

// createTestCustomer creates a valid customer in the fixture's tenant.
func (f *jobFixture) createTestCustomer(t *testing.T) *models.Customer {
	t.Helper()
	suffix := uuid.New().String()[:8]
	customer := &models.Customer{
		FirstName:            "Test",
		LastName:             "Customer " + suffix,
		Email:                "test-" + suffix + "@example.com",
		PhonePrimary:         "512-555-0100",
		BillingAddressStreet: "123 Main St",
		BillingAddressCity:   "Austin",
		BillingAddressState:  "TX",
		BillingAddressZip:    "78701",
		CustomerType:         models.CustomerTypeResidential,
		Status:               models.CustomerStatusActive,
	}
	require.NoError(t, NewCustomerRepository(f.db).WithContext(f.tenant.Ctx).Create(customer))
	require.Equal(t, f.tenant.ID, customer.TenantID, "tenancy plugin assigns the caller's tenant")
	return customer
}

// createTestJob creates a scheduled job for customerID in the fixture's tenant.
func (f *jobFixture) createTestJob(t *testing.T, customerID uuid.UUID) *models.Job {
	t.Helper()
	now := time.Now()
	scheduledStart := now.Add(24 * time.Hour).Unix()
	scheduledEnd := now.Add(26 * time.Hour).Unix()

	job := &models.Job{
		CustomerID:       customerID,
		Title:            "Test Job",
		Description:      "Test Description",
		JobType:          models.JobTypeInstallation,
		Status:           models.JobStatusScheduled,
		Priority:         models.JobPriorityNormal,
		ScheduledStartAt: &scheduledStart,
		ScheduledEndAt:   &scheduledEnd,
		ServiceAddress: models.ServiceAddress{
			Street: "456 Service St",
			City:   "Austin",
			State:  "TX",
			Zip:    "78702",
		},
		CreatedBy: f.tenant.OwnerID,
	}
	require.NoError(t, f.repo.Create(job))
	return job
}

func containsJob(jobs []models.Job, id uuid.UUID) bool {
	for _, j := range jobs {
		if j.ID == id {
			return true
		}
	}
	return false
}

func TestJobRepository_Create(t *testing.T) {
	f := newJobFixture(t)
	customer := f.createTestCustomer(t)

	now := time.Now()
	scheduledStart := now.Add(24 * time.Hour).Unix()
	scheduledEnd := now.Add(26 * time.Hour).Unix()

	job := &models.Job{
		CustomerID:       customer.ID,
		Title:            "Installation Job",
		Description:      "Install new HVAC system",
		JobType:          models.JobTypeInstallation,
		Status:           models.JobStatusScheduled,
		Priority:         models.JobPriorityNormal,
		ScheduledStartAt: &scheduledStart,
		ScheduledEndAt:   &scheduledEnd,
		ServiceAddress: models.ServiceAddress{
			Street: "123 Main St",
			City:   "Austin",
			State:  "TX",
			Zip:    "78701",
		},
		CreatedBy: f.tenant.OwnerID,
	}

	err := f.repo.Create(job)
	require.NoError(t, err)
	assert.NotEqual(t, uuid.Nil, job.ID)
	assert.Equal(t, f.tenant.ID, job.TenantID)
	assert.NotEmpty(t, job.JobNumber)
	assert.NotZero(t, job.CreatedAt)
}

func TestJobRepository_GetByID(t *testing.T) {
	f := newJobFixture(t)
	customer := f.createTestCustomer(t)
	job := f.createTestJob(t, customer.ID)

	retrieved, err := f.repo.GetByID(job.ID)
	require.NoError(t, err)
	assert.Equal(t, job.ID, retrieved.ID)
	assert.Equal(t, job.Title, retrieved.Title)
	assert.Equal(t, job.JobType, retrieved.JobType)
	require.NotNil(t, retrieved.Customer)
	assert.Equal(t, customer.ID, retrieved.Customer.ID)
}

func TestJobRepository_GetByID_NotFound(t *testing.T) {
	f := newJobFixture(t)

	_, err := f.repo.GetByID(uuid.New())
	assert.ErrorIs(t, err, gorm.ErrRecordNotFound)
}

func TestJobRepository_GetByJobNumber(t *testing.T) {
	f := newJobFixture(t)
	customer := f.createTestCustomer(t)
	job := f.createTestJob(t, customer.ID)

	retrieved, err := f.repo.GetByJobNumber(job.JobNumber)
	require.NoError(t, err)
	assert.Equal(t, job.ID, retrieved.ID)
	assert.Equal(t, job.JobNumber, retrieved.JobNumber)
}

func TestJobRepository_Update(t *testing.T) {
	f := newJobFixture(t)
	customer := f.createTestCustomer(t)
	job := f.createTestJob(t, customer.ID)

	job.Status = models.JobStatusInProgress
	job.Priority = models.JobPriorityHigh
	job.Title = "Updated Job Title"

	require.NoError(t, f.repo.Update(job))

	updated, err := f.repo.GetByID(job.ID)
	require.NoError(t, err)
	assert.Equal(t, models.JobStatusInProgress, updated.Status)
	assert.Equal(t, models.JobPriorityHigh, updated.Priority)
	assert.Equal(t, "Updated Job Title", updated.Title)
}

func TestJobRepository_Delete(t *testing.T) {
	f := newJobFixture(t)
	customer := f.createTestCustomer(t)
	job := f.createTestJob(t, customer.ID)

	require.NoError(t, f.repo.Delete(job.ID))

	// Soft-deleted jobs are not found.
	_, err := f.repo.GetByID(job.ID)
	assert.ErrorIs(t, err, gorm.ErrRecordNotFound)
}

func TestJobRepository_List(t *testing.T) {
	f := newJobFixture(t)
	customer := f.createTestCustomer(t)

	for i := 0; i < 5; i++ {
		f.createTestJob(t, customer.ID)
	}

	limit := 3
	offset := 0
	filter := &models.JobListFilter{Limit: &limit, Offset: &offset}

	jobs, total, err := f.repo.List(filter)
	require.NoError(t, err)
	assert.Equal(t, int64(5), total)
	assert.Len(t, jobs, 3)
}

func TestJobRepository_List_WithFilters(t *testing.T) {
	f := newJobFixture(t)
	customer := f.createTestCustomer(t)
	other := f.createTestCustomer(t)

	job1 := f.createTestJob(t, customer.ID)
	job1.Status = models.JobStatusScheduled
	job1.Priority = models.JobPriorityHigh
	require.NoError(t, f.repo.Update(job1))

	job2 := f.createTestJob(t, customer.ID)
	job2.Status = models.JobStatusInProgress
	job2.Priority = models.JobPriorityNormal
	require.NoError(t, f.repo.Update(job2))

	f.createTestJob(t, other.ID) // scheduled, normal priority

	limit := 10
	offset := 0

	// Filter by customer
	jobs, total, err := f.repo.List(&models.JobListFilter{CustomerID: &customer.ID, Limit: &limit, Offset: &offset})
	require.NoError(t, err)
	assert.Equal(t, int64(2), total)
	assert.Len(t, jobs, 2)

	// Filter by status
	status := models.JobStatusInProgress
	jobs, total, err = f.repo.List(&models.JobListFilter{Status: &status, Limit: &limit, Offset: &offset})
	require.NoError(t, err)
	assert.Equal(t, int64(1), total)
	require.Len(t, jobs, 1)
	assert.Equal(t, job2.ID, jobs[0].ID)

	// Filter by priority
	priority := models.JobPriorityHigh
	jobs, total, err = f.repo.List(&models.JobListFilter{Priority: &priority, Limit: &limit, Offset: &offset})
	require.NoError(t, err)
	assert.Equal(t, int64(1), total)
	require.Len(t, jobs, 1)
	assert.Equal(t, job1.ID, jobs[0].ID)
}

func TestJobRepository_GetByCustomer(t *testing.T) {
	f := newJobFixture(t)
	customer := f.createTestCustomer(t)
	other := f.createTestCustomer(t)

	f.createTestJob(t, customer.ID)
	f.createTestJob(t, customer.ID)
	f.createTestJob(t, other.ID)

	jobs, total, err := f.repo.GetByCustomer(customer.ID, 10, 0)
	require.NoError(t, err)
	assert.Equal(t, int64(2), total)
	assert.Len(t, jobs, 2)
}

func TestJobRepository_GetByStatus(t *testing.T) {
	f := newJobFixture(t)
	customer := f.createTestCustomer(t)

	job := f.createTestJob(t, customer.ID)
	job.Status = models.JobStatusInProgress
	require.NoError(t, f.repo.Update(job))
	scheduled := f.createTestJob(t, customer.ID)

	jobs, err := f.repo.GetByStatus(models.JobStatusInProgress)
	require.NoError(t, err)
	assert.Len(t, jobs, 1)
	assert.True(t, containsJob(jobs, job.ID))
	assert.False(t, containsJob(jobs, scheduled.ID))
}

func TestJobRepository_GetScheduledJobs(t *testing.T) {
	f := newJobFixture(t)
	customer := f.createTestCustomer(t)

	now := time.Now()
	tomorrow := now.Add(24 * time.Hour).Unix()
	job := f.createTestJob(t, customer.ID)
	job.ScheduledStartAt = &tomorrow
	require.NoError(t, f.repo.Update(job))

	nextMonth := now.Add(30 * 24 * time.Hour).Unix()
	later := f.createTestJob(t, customer.ID)
	later.ScheduledStartAt = &nextMonth
	require.NoError(t, f.repo.Update(later))

	jobs, err := f.repo.GetScheduledJobs(now.Unix(), now.Add(7*24*time.Hour).Unix())
	require.NoError(t, err)
	assert.True(t, containsJob(jobs, job.ID))
	assert.False(t, containsJob(jobs, later.ID))
}

func TestJobRepository_GetOverdueJobs(t *testing.T) {
	f := newJobFixture(t)
	customer := f.createTestCustomer(t)

	yesterday := time.Now().Add(-24 * time.Hour).Unix()
	job := f.createTestJob(t, customer.ID)
	job.Status = models.JobStatusInProgress
	job.ScheduledEndAt = &yesterday
	require.NoError(t, f.repo.Update(job))
	onTime := f.createTestJob(t, customer.ID) // ends tomorrow

	jobs, err := f.repo.GetOverdueJobs()
	require.NoError(t, err)
	assert.True(t, containsJob(jobs, job.ID))
	assert.False(t, containsJob(jobs, onTime.ID))
}

func TestJobRepository_Assignments(t *testing.T) {
	f := newJobFixture(t)
	customer := f.createTestCustomer(t)
	job := f.createTestJob(t, customer.ID)

	technicianID := f.tenant.addMember(t, f.db)
	assignment := &models.JobAssignment{
		JobID:  job.ID,
		UserID: technicianID,
		Role:   "Lead Technician",
	}

	require.NoError(t, f.repo.AddAssignment(assignment))

	assignments, err := f.repo.GetJobAssignments(job.ID)
	require.NoError(t, err)
	require.Len(t, assignments, 1)
	assert.Equal(t, technicianID, assignments[0].UserID)
	assert.Equal(t, "Lead Technician", assignments[0].Role)

	require.NoError(t, f.repo.RemoveAssignment(assignment.ID))

	assignments, err = f.repo.GetJobAssignments(job.ID)
	require.NoError(t, err)
	assert.Len(t, assignments, 0)
}

func TestJobRepository_AddAssignment_RejectsNonMember(t *testing.T) {
	f := newJobFixture(t)
	customer := f.createTestCustomer(t)
	job := f.createTestJob(t, customer.ID)
	outsider := newJobFixtureOn(t, f.db).tenant.OwnerID

	err := f.repo.AddAssignment(&models.JobAssignment{JobID: job.ID, UserID: outsider, Role: "Technician"})
	assert.ErrorIs(t, err, ErrAssigneeNotInTenant)
}

func TestJobRepository_Materials(t *testing.T) {
	f := newJobFixture(t)
	customer := f.createTestCustomer(t)
	job := f.createTestJob(t, customer.ID)

	description := "High-efficiency filter"
	material := &models.JobMaterial{
		JobID:       job.ID,
		Name:        "HVAC Filter",
		Description: &description,
		Quantity:    2.0,
		Unit:        "each",
		UnitCost:    25.99,
		TotalCost:   51.98,
	}

	require.NoError(t, f.repo.AddMaterial(material))

	materials, err := f.repo.GetJobMaterials(job.ID)
	require.NoError(t, err)
	require.Len(t, materials, 1)
	assert.Equal(t, "HVAC Filter", materials[0].Name)
	assert.Equal(t, 2.0, materials[0].Quantity)

	material.Quantity = 3.0
	material.TotalCost = 77.97
	require.NoError(t, f.repo.UpdateMaterial(material))

	materials, err = f.repo.GetJobMaterials(job.ID)
	require.NoError(t, err)
	require.Len(t, materials, 1)
	assert.Equal(t, 3.0, materials[0].Quantity)
	assert.Equal(t, 77.97, materials[0].TotalCost)

	require.NoError(t, f.repo.DeleteMaterial(material.ID))

	materials, err = f.repo.GetJobMaterials(job.ID)
	require.NoError(t, err)
	assert.Len(t, materials, 0)
}

func TestJobRepository_Notes(t *testing.T) {
	f := newJobFixture(t)
	customer := f.createTestCustomer(t)
	job := f.createTestJob(t, customer.ID)

	note := &models.JobNote{
		JobID:      job.ID,
		UserID:     f.tenant.OwnerID,
		Note:       "Customer requested morning appointment",
		IsInternal: false,
	}
	require.NoError(t, f.repo.AddNote(note))

	internalNote := &models.JobNote{
		JobID:      job.ID,
		UserID:     f.tenant.OwnerID,
		Note:       "Check inventory before visit",
		IsInternal: true,
	}
	require.NoError(t, f.repo.AddNote(internalNote))

	notes, err := f.repo.GetJobNotes(job.ID, true)
	require.NoError(t, err)
	assert.Len(t, notes, 2)

	notes, err = f.repo.GetJobNotes(job.ID, false)
	require.NoError(t, err)
	require.Len(t, notes, 1)
	assert.False(t, notes[0].IsInternal)
}

func TestJobRepository_GetJobStats(t *testing.T) {
	f := newJobFixture(t)
	customer := f.createTestCustomer(t)

	job1 := f.createTestJob(t, customer.ID)
	job1.Status = models.JobStatusCompleted
	job1.JobType = models.JobTypeRepair
	job1.ActualCost = 150.00
	require.NoError(t, f.repo.Update(job1))

	job2 := f.createTestJob(t, customer.ID)
	job2.Status = models.JobStatusInProgress
	require.NoError(t, f.repo.Update(job2))

	stats, err := f.repo.GetJobStats()
	require.NoError(t, err)
	assert.Equal(t, int64(2), stats["total"])
	assert.InDelta(t, 150.0, stats["avg_cost"], 0.001)
	assert.NotNil(t, stats["by_status"])
	assert.NotNil(t, stats["by_priority"])
	assert.NotNil(t, stats["by_type"])
}

// TestJobRepository_CrossTenant: tenant B can neither see nor change tenant
// A's job (hard rule 1 — cross-tenant reads are "not found").
func TestJobRepository_CrossTenant(t *testing.T) {
	a := newJobFixture(t)
	b := newJobFixtureOn(t, a.db)
	customer := a.createTestCustomer(t)
	job := a.createTestJob(t, customer.ID)

	_, err := b.repo.GetByID(job.ID)
	assert.ErrorIs(t, err, gorm.ErrRecordNotFound, "GetByID")

	_, err = b.repo.GetByJobNumber(job.JobNumber)
	assert.ErrorIs(t, err, gorm.ErrRecordNotFound, "GetByJobNumber")

	limit, offset := 10, 0
	jobs, total, err := b.repo.List(&models.JobListFilter{Limit: &limit, Offset: &offset})
	require.NoError(t, err)
	assert.Zero(t, total)
	assert.Empty(t, jobs)

	jobs, total, err = b.repo.GetByCustomer(customer.ID, 10, 0)
	require.NoError(t, err)
	assert.Zero(t, total)
	assert.Empty(t, jobs)

	jobs, err = b.repo.GetByStatus(models.JobStatusScheduled)
	require.NoError(t, err)
	assert.False(t, containsJob(jobs, job.ID))

	// B can't assign its own member to A's job: the job lookup is scoped.
	err = b.repo.AddAssignment(&models.JobAssignment{JobID: job.ID, UserID: b.tenant.OwnerID, Role: "Technician"})
	assert.ErrorIs(t, err, gorm.ErrRecordNotFound, "AddAssignment")

	// B's delete matches no row, so A's job survives.
	require.NoError(t, b.repo.Delete(job.ID))
	still, err := a.repo.GetByID(job.ID)
	require.NoError(t, err)
	assert.Equal(t, job.ID, still.ID)

	stats, err := b.repo.GetJobStats()
	require.NoError(t, err)
	assert.Equal(t, int64(0), stats["total"])
}
