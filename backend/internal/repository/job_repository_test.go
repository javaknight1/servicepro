package repository

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/javaknight1/servicepro/backend/config"
	"github.com/javaknight1/servicepro/backend/internal/models"
	"github.com/javaknight1/servicepro/backend/pkg/database"
)

// setupTestDB creates a test database connection
func setupTestDB(t *testing.T) *gorm.DB {
	cfg := &config.DatabaseConfig{
		URL:             "postgres://postgres:postgres@localhost:5432/servicepro_test?sslmode=disable",
		MaxOpenConns:    25,
		MaxIdleConns:    5,
		ConnMaxLifetime: time.Hour,
	}

	db, _, err := database.NewGormDB(cfg)
	if err != nil {
		t.Skipf("Skipping test - database not available: %v", err)
	}
	return db
}

// createTestCustomer creates a customer for testing
func createTestCustomer(t *testing.T, db *gorm.DB) *models.Customer {
	t.Helper()
	customerRepo := NewCustomerRepository(db)

	email := "test" + uuid.New().String()[:8] + "@example.com"
	customer := &models.Customer{
		FirstName:            "Test",
		LastName:             "Customer",
		Email:                email,
		PhonePrimary:         "555-0100",
		BillingAddressStreet: "123 Main St",
		BillingAddressCity:   "Austin",
		BillingAddressState:  "TX",
		BillingAddressZip:    "78701",
		CustomerType:         models.CustomerTypeResidential,
		Status:               models.CustomerStatusActive,
	}

	err := customerRepo.Create(customer)
	require.NoError(t, err)
	return customer
}

// createTestJob creates a job for testing
func createTestJob(t *testing.T, db *gorm.DB, customerID uuid.UUID) *models.Job {
	t.Helper()
	repo := NewJobRepository(db)

	now := time.Now()
	scheduledStart := now.Add(24 * time.Hour)
	scheduledEnd := scheduledStart.Add(2 * time.Hour)

	createdBy := uuid.New()
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
		CreatedBy: createdBy,
	}

	err := repo.Create(job)
	require.NoError(t, err)
	return job
}

func TestJobRepository_Create(t *testing.T) {
	db := setupTestDB(t)
	repo := NewJobRepository(db)
	customer := createTestCustomer(t, db)
	defer db.Exec("DELETE FROM jobs WHERE customer_id = ?", customer.ID)
	defer db.Exec("DELETE FROM customers WHERE id = ?", customer.ID)

	now := time.Now()
	scheduledStart := now.Add(24 * time.Hour)
	scheduledEnd := scheduledStart.Add(2 * time.Hour)
	createdBy := uuid.New()

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
		CreatedBy: createdBy,
	}

	err := repo.Create(job)
	require.NoError(t, err)
	assert.NotEqual(t, uuid.Nil, job.ID)
	assert.NotEmpty(t, job.JobNumber)
	assert.NotZero(t, job.CreatedAt)
}

func TestJobRepository_GetByID(t *testing.T) {
	db := setupTestDB(t)
	repo := NewJobRepository(db)
	customer := createTestCustomer(t, db)
	job := createTestJob(t, db, customer.ID)
	defer db.Exec("DELETE FROM jobs WHERE id = ?", job.ID)
	defer db.Exec("DELETE FROM customers WHERE id = ?", customer.ID)

	retrieved, err := repo.GetByID(job.ID)
	require.NoError(t, err)
	assert.Equal(t, job.ID, retrieved.ID)
	assert.Equal(t, job.Title, retrieved.Title)
	assert.Equal(t, job.JobType, retrieved.JobType)
	assert.NotNil(t, retrieved.Customer)
	assert.Equal(t, customer.ID, retrieved.Customer.ID)
}

func TestJobRepository_GetByID_NotFound(t *testing.T) {
	db := setupTestDB(t)
	repo := NewJobRepository(db)

	_, err := repo.GetByID(uuid.New())
	assert.Error(t, err)
	assert.ErrorIs(t, err, gorm.ErrRecordNotFound)
}

func TestJobRepository_GetByJobNumber(t *testing.T) {
	db := setupTestDB(t)
	repo := NewJobRepository(db)
	customer := createTestCustomer(t, db)
	job := createTestJob(t, db, customer.ID)
	defer db.Exec("DELETE FROM jobs WHERE id = ?", job.ID)
	defer db.Exec("DELETE FROM customers WHERE id = ?", customer.ID)

	retrieved, err := repo.GetByJobNumber(job.JobNumber)
	require.NoError(t, err)
	assert.Equal(t, job.ID, retrieved.ID)
	assert.Equal(t, job.JobNumber, retrieved.JobNumber)
}

func TestJobRepository_Update(t *testing.T) {
	db := setupTestDB(t)
	repo := NewJobRepository(db)
	customer := createTestCustomer(t, db)
	job := createTestJob(t, db, customer.ID)
	defer db.Exec("DELETE FROM jobs WHERE id = ?", job.ID)
	defer db.Exec("DELETE FROM customers WHERE id = ?", customer.ID)

	job.Status = models.JobStatusScheduled
	job.Priority = models.JobPriorityHigh
	newTitle := "Updated Job Title"
	job.Title = newTitle

	err := repo.Update(job)
	require.NoError(t, err)

	updated, err := repo.GetByID(job.ID)
	require.NoError(t, err)
	assert.Equal(t, models.JobStatusScheduled, updated.Status)
	assert.Equal(t, models.JobPriorityHigh, updated.Priority)
	assert.Equal(t, newTitle, updated.Title)
}

func TestJobRepository_Delete(t *testing.T) {
	db := setupTestDB(t)
	repo := NewJobRepository(db)
	customer := createTestCustomer(t, db)
	job := createTestJob(t, db, customer.ID)
	defer db.Exec("DELETE FROM jobs WHERE id = ?", job.ID)
	defer db.Exec("DELETE FROM customers WHERE id = ?", customer.ID)

	err := repo.Delete(job.ID)
	require.NoError(t, err)

	// Should not find the deleted job (soft delete)
	_, err = repo.GetByID(job.ID)
	assert.Error(t, err)
}

func TestJobRepository_List(t *testing.T) {
	db := setupTestDB(t)
	repo := NewJobRepository(db)
	customer := createTestCustomer(t, db)
	defer db.Exec("DELETE FROM customers WHERE id = ?", customer.ID)

	// Create multiple jobs
	for i := 0; i < 5; i++ {
		job := createTestJob(t, db, customer.ID)
		defer db.Exec("DELETE FROM jobs WHERE id = ?", job.ID)
	}

	// Test basic list
	limit := 10
	offset := 0
	filter := &models.JobListFilter{
		Limit:  &limit,
		Offset: &offset,
	}

	jobs, total, err := repo.List(filter)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, total, int64(5))
	assert.GreaterOrEqual(t, len(jobs), 5)
}

func TestJobRepository_List_WithFilters(t *testing.T) {
	db := setupTestDB(t)
	repo := NewJobRepository(db)
	customer := createTestCustomer(t, db)
	defer db.Exec("DELETE FROM customers WHERE id = ?", customer.ID)

	// Create jobs with different statuses
	job1 := createTestJob(t, db, customer.ID)
	job1.Status = models.JobStatusScheduled
	job1.Priority = models.JobPriorityHigh
	repo.Update(job1)
	defer db.Exec("DELETE FROM jobs WHERE id = ?", job1.ID)

	job2 := createTestJob(t, db, customer.ID)
	job2.Status = models.JobStatusInProgress
	job2.Priority = models.JobPriorityNormal
	repo.Update(job2)
	defer db.Exec("DELETE FROM jobs WHERE id = ?", job2.ID)

	// Filter by customer
	limit := 10
	offset := 0
	filter := &models.JobListFilter{
		CustomerID: &customer.ID,
		Limit:      &limit,
		Offset:     &offset,
	}

	jobs, total, err := repo.List(filter)
	require.NoError(t, err)
	assert.Equal(t, int64(2), total)
	assert.Len(t, jobs, 2)

	// Filter by status
	status := models.JobStatusScheduled
	filter = &models.JobListFilter{
		Status: &status,
		Limit:  &limit,
		Offset: &offset,
	}

	jobs, total, err = repo.List(filter)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, total, int64(1))
	for _, job := range jobs {
		assert.Equal(t, models.JobStatusScheduled, job.Status)
	}

	// Filter by priority
	priority := models.JobPriorityHigh
	filter = &models.JobListFilter{
		Priority: &priority,
		Limit:    &limit,
		Offset:   &offset,
	}

	jobs, total, err = repo.List(filter)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, total, int64(1))
	for _, job := range jobs {
		assert.Equal(t, models.JobPriorityHigh, job.Priority)
	}
}

func TestJobRepository_GetByCustomer(t *testing.T) {
	db := setupTestDB(t)
	repo := NewJobRepository(db)
	customer := createTestCustomer(t, db)
	defer db.Exec("DELETE FROM customers WHERE id = ?", customer.ID)

	// Create jobs for customer
	job1 := createTestJob(t, db, customer.ID)
	job2 := createTestJob(t, db, customer.ID)
	defer db.Exec("DELETE FROM jobs WHERE id IN (?, ?)", job1.ID, job2.ID)

	jobs, total, err := repo.GetByCustomer(customer.ID, 10, 0)
	require.NoError(t, err)
	assert.Equal(t, int64(2), total)
	assert.Len(t, jobs, 2)
}

func TestJobRepository_GetByStatus(t *testing.T) {
	db := setupTestDB(t)
	repo := NewJobRepository(db)
	customer := createTestCustomer(t, db)
	defer db.Exec("DELETE FROM customers WHERE id = ?", customer.ID)

	// Create jobs with specific status
	job := createTestJob(t, db, customer.ID)
	job.Status = models.JobStatusInProgress
	repo.Update(job)
	defer db.Exec("DELETE FROM jobs WHERE id = ?", job.ID)

	jobs, err := repo.GetByStatus(models.JobStatusInProgress)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(jobs), 1)

	found := false
	for _, j := range jobs {
		if j.ID == job.ID {
			found = true
			break
		}
	}
	assert.True(t, found)
}

func TestJobRepository_GetScheduledJobs(t *testing.T) {
	db := setupTestDB(t)
	repo := NewJobRepository(db)
	customer := createTestCustomer(t, db)
	defer db.Exec("DELETE FROM customers WHERE id = ?", customer.ID)

	// Create scheduled job
	now := time.Now()
	tomorrow := now.Add(24 * time.Hour)
	job := createTestJob(t, db, customer.ID)
	job.Status = models.JobStatusScheduled
	job.ScheduledStartAt = &tomorrow
	repo.Update(job)
	defer db.Exec("DELETE FROM jobs WHERE id = ?", job.ID)

	// Get scheduled jobs for next week
	start := now
	end := now.Add(7 * 24 * time.Hour)
	jobs, err := repo.GetScheduledJobs(start, end)
	require.NoError(t, err)

	found := false
	for _, j := range jobs {
		if j.ID == job.ID {
			found = true
			break
		}
	}
	assert.True(t, found)
}

func TestJobRepository_GetOverdueJobs(t *testing.T) {
	db := setupTestDB(t)
	repo := NewJobRepository(db)
	customer := createTestCustomer(t, db)
	defer db.Exec("DELETE FROM customers WHERE id = ?", customer.ID)

	// Create overdue job
	now := time.Now()
	yesterday := now.Add(-24 * time.Hour)
	job := createTestJob(t, db, customer.ID)
	job.Status = models.JobStatusInProgress
	job.ScheduledEndAt = &yesterday
	repo.Update(job)
	defer db.Exec("DELETE FROM jobs WHERE id = ?", job.ID)

	jobs, err := repo.GetOverdueJobs()
	require.NoError(t, err)

	found := false
	for _, j := range jobs {
		if j.ID == job.ID {
			found = true
			break
		}
	}
	assert.True(t, found)
}

func TestJobRepository_Assignments(t *testing.T) {
	db := setupTestDB(t)
	repo := NewJobRepository(db)
	customer := createTestCustomer(t, db)
	job := createTestJob(t, db, customer.ID)
	defer db.Exec("DELETE FROM jobs WHERE id = ?", job.ID)
	defer db.Exec("DELETE FROM customers WHERE id = ?", customer.ID)

	technicianID := uuid.New()
	assignment := &models.JobAssignment{
		JobID:  job.ID,
		UserID: technicianID,
		Role:   "Lead Technician",
	}

	// Add assignment
	err := repo.AddAssignment(assignment)
	require.NoError(t, err)
	defer db.Exec("DELETE FROM job_assignments WHERE id = ?", assignment.ID)

	// Get assignments
	assignments, err := repo.GetJobAssignments(job.ID)
	require.NoError(t, err)
	assert.Len(t, assignments, 1)
	assert.Equal(t, technicianID, assignments[0].UserID)
	assert.Equal(t, "Lead Technician", assignments[0].Role)

	// Remove assignment
	err = repo.RemoveAssignment(assignment.ID)
	require.NoError(t, err)

	// Verify removed
	assignments, err = repo.GetJobAssignments(job.ID)
	require.NoError(t, err)
	assert.Len(t, assignments, 0)
}

func TestJobRepository_Materials(t *testing.T) {
	db := setupTestDB(t)
	repo := NewJobRepository(db)
	customer := createTestCustomer(t, db)
	job := createTestJob(t, db, customer.ID)
	defer db.Exec("DELETE FROM jobs WHERE id = ?", job.ID)
	defer db.Exec("DELETE FROM customers WHERE id = ?", customer.ID)

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

	// Add material
	err := repo.AddMaterial(material)
	require.NoError(t, err)
	defer db.Exec("DELETE FROM job_materials WHERE id = ?", material.ID)

	// Get materials
	materials, err := repo.GetJobMaterials(job.ID)
	require.NoError(t, err)
	assert.Len(t, materials, 1)
	assert.Equal(t, "HVAC Filter", materials[0].Name)
	assert.Equal(t, 2.0, materials[0].Quantity)

	// Update material
	material.Quantity = 3.0
	material.TotalCost = 77.97
	err = repo.UpdateMaterial(material)
	require.NoError(t, err)

	materials, err = repo.GetJobMaterials(job.ID)
	require.NoError(t, err)
	assert.Equal(t, 3.0, materials[0].Quantity)
	assert.Equal(t, 77.97, materials[0].TotalCost)

	// Delete material
	err = repo.DeleteMaterial(material.ID)
	require.NoError(t, err)

	materials, err = repo.GetJobMaterials(job.ID)
	require.NoError(t, err)
	assert.Len(t, materials, 0)
}

func TestJobRepository_Notes(t *testing.T) {
	db := setupTestDB(t)
	repo := NewJobRepository(db)
	customer := createTestCustomer(t, db)
	job := createTestJob(t, db, customer.ID)
	defer db.Exec("DELETE FROM jobs WHERE id = ?", job.ID)
	defer db.Exec("DELETE FROM customers WHERE id = ?", customer.ID)

	userID := uuid.New()
	note := &models.JobNote{
		JobID:      job.ID,
		UserID:     userID,
		Note:       "Customer requested morning appointment",
		IsInternal: false,
	}

	// Add note
	err := repo.AddNote(note)
	require.NoError(t, err)
	defer db.Exec("DELETE FROM job_notes WHERE id = ?", note.ID)

	internalNote := &models.JobNote{
		JobID:      job.ID,
		UserID:     userID,
		Note:       "Check inventory before visit",
		IsInternal: true,
	}

	err = repo.AddNote(internalNote)
	require.NoError(t, err)
	defer db.Exec("DELETE FROM job_notes WHERE id = ?", internalNote.ID)

	// Get all notes (including internal)
	notes, err := repo.GetJobNotes(job.ID, true)
	require.NoError(t, err)
	assert.Len(t, notes, 2)

	// Get only external notes
	notes, err = repo.GetJobNotes(job.ID, false)
	require.NoError(t, err)
	assert.Len(t, notes, 1)
	assert.False(t, notes[0].IsInternal)
}

func TestJobRepository_GetJobStats(t *testing.T) {
	db := setupTestDB(t)
	repo := NewJobRepository(db)
	customer := createTestCustomer(t, db)
	defer db.Exec("DELETE FROM customers WHERE id = ?", customer.ID)

	// Create jobs with different statuses and types
	job1 := createTestJob(t, db, customer.ID)
	job1.Status = models.JobStatusCompleted
	job1.JobType = models.JobTypeRepair
	job1.ActualCost = 150.00
	repo.Update(job1)
	defer db.Exec("DELETE FROM jobs WHERE id = ?", job1.ID)

	job2 := createTestJob(t, db, customer.ID)
	job2.Status = models.JobStatusInProgress
	job2.JobType = models.JobTypeInstallation
	repo.Update(job2)
	defer db.Exec("DELETE FROM jobs WHERE id = ?", job2.ID)

	stats, err := repo.GetJobStats()
	require.NoError(t, err)
	assert.NotNil(t, stats["total"])
	assert.NotNil(t, stats["by_status"])
	assert.NotNil(t, stats["by_priority"])
	assert.NotNil(t, stats["by_type"])
}
