package repository

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/javaknight1/servicepro/backend/internal/models"
	"github.com/javaknight1/servicepro/backend/internal/utils"
)

// JobRepository implements JobRepositoryInterface using GORM
// Interface definition is in interface.go
type JobRepository struct {
	db *gorm.DB
}

// NewJobRepository creates a new job repository
func NewJobRepository(db *gorm.DB) *JobRepository {
	return &JobRepository{db: db}
}

// Create creates a new job
func (r *JobRepository) Create(job *models.Job) error {
	return r.db.Create(job).Error
}

// GetByID retrieves a job by ID with all relationships
func (r *JobRepository) GetByID(id uuid.UUID) (*models.Job, error) {
	var job models.Job
	err := r.db.
		Preload("Customer").
		Preload("Assignments").
		Preload("Materials").
		Preload("Notes").
		Where("id = ?", id).
		First(&job).Error

	if err != nil {
		return nil, err
	}
	return &job, nil
}

// GetByJobNumber retrieves a job by job number
func (r *JobRepository) GetByJobNumber(jobNumber string) (*models.Job, error) {
	var job models.Job
	err := r.db.
		Preload("Customer").
		Preload("Assignments").
		Preload("Materials").
		Preload("Notes").
		Where("job_number = ?", jobNumber).
		First(&job).Error

	if err != nil {
		return nil, err
	}
	return &job, nil
}

// Update updates a job
func (r *JobRepository) Update(job *models.Job) error {
	return r.db.Save(job).Error
}

// Delete soft deletes a job
func (r *JobRepository) Delete(id uuid.UUID) error {
	return r.db.Delete(&models.Job{}, id).Error
}

// List retrieves jobs with filtering and pagination
func (r *JobRepository) List(filter *models.JobListFilter) ([]models.Job, int64, error) {
	var jobs []models.Job
	var total int64

	query := r.db.Model(&models.Job{})

	// Apply filters
	if filter.CustomerID != nil {
		query = query.Where("customer_id = ?", *filter.CustomerID)
	}

	if filter.Status != nil {
		query = query.Where("status = ?", *filter.Status)
	}

	if filter.Priority != nil {
		query = query.Where("priority = ?", *filter.Priority)
	}

	if filter.JobType != nil {
		query = query.Where("job_type = ?", *filter.JobType)
	}

	if filter.AssignedUserID != nil {
		query = query.Joins("JOIN job_assignments ON job_assignments.job_id = jobs.id").
			Where("job_assignments.user_id = ?", *filter.AssignedUserID).
			Where("job_assignments.deleted_at IS NULL")
	}

	if filter.ScheduledStartFrom != nil {
		query = query.Where("scheduled_start_at >= ?", *filter.ScheduledStartFrom)
	}

	if filter.ScheduledStartTo != nil {
		query = query.Where("scheduled_start_at <= ?", *filter.ScheduledStartTo)
	}

	if filter.RequiresFollowUp != nil && *filter.RequiresFollowUp {
		query = query.Where("requires_follow_up = ?", true)
	}

	// Count total before pagination
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// Apply sorting (with SQL injection protection)
	sortField := "created_at"
	sortOrder := "DESC"
	if filter.SortBy != nil {
		sortField = *filter.SortBy
	}
	if filter.SortOrder != nil {
		sortOrder = *filter.SortOrder
	}
	orderClause := utils.SafeOrderClause(sortField, sortOrder, utils.JobAllowedSortColumns, "created_at")
	query = query.Order(orderClause)

	// Apply pagination
	if filter.Limit != nil {
		query = query.Limit(*filter.Limit)
	}
	if filter.Offset != nil {
		query = query.Offset(*filter.Offset)
	}

	// Preload relationships
	err := query.
		Preload("Customer").
		Preload("Assignments").
		Preload("Materials").
		Find(&jobs).Error

	return jobs, total, err
}

// GetByCustomer retrieves all jobs for a customer
func (r *JobRepository) GetByCustomer(customerID uuid.UUID, limit, offset int) ([]models.Job, int64, error) {
	var jobs []models.Job
	var total int64

	query := r.db.Model(&models.Job{}).Where("customer_id = ?", customerID)

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	err := query.
		Limit(limit).
		Offset(offset).
		Order("created_at DESC").
		Preload("Assignments").
		Preload("Materials").
		Find(&jobs).Error

	return jobs, total, err
}

// GetByStatus retrieves jobs by status
func (r *JobRepository) GetByStatus(status models.JobStatus) ([]models.Job, error) {
	var jobs []models.Job
	err := r.db.
		Where("status = ?", status).
		Preload("Customer").
		Preload("Assignments").
		Order("scheduled_start_at ASC").
		Find(&jobs).Error

	return jobs, err
}

// GetByAssignedUser retrieves jobs assigned to a user
func (r *JobRepository) GetByAssignedUser(userID uuid.UUID) ([]models.Job, error) {
	var jobs []models.Job
	err := r.db.
		Joins("JOIN job_assignments ON job_assignments.job_id = jobs.id").
		Where("job_assignments.user_id = ?", userID).
		Where("job_assignments.deleted_at IS NULL").
		Preload("Customer").
		Preload("Assignments").
		Preload("Materials").
		Order("scheduled_start_at ASC").
		Find(&jobs).Error

	return jobs, err
}

// GetScheduledJobs retrieves jobs scheduled within a date range
func (r *JobRepository) GetScheduledJobs(start, end time.Time) ([]models.Job, error) {
	var jobs []models.Job
	err := r.db.
		Where("scheduled_start_at BETWEEN ? AND ?", start, end).
		Where("status IN ?", []models.JobStatus{
			models.JobStatusScheduled,
			models.JobStatusInProgress,
		}).
		Preload("Customer").
		Preload("Assignments").
		Order("scheduled_start_at ASC").
		Find(&jobs).Error

	return jobs, err
}

// GetOverdueJobs retrieves jobs that are overdue
func (r *JobRepository) GetOverdueJobs() ([]models.Job, error) {
	var jobs []models.Job
	now := time.Now()

	err := r.db.
		Where("scheduled_end_at < ?", now).
		Where("status IN ?", []models.JobStatus{
			models.JobStatusScheduled,
			models.JobStatusInProgress,
			models.JobStatusOnHold,
		}).
		Preload("Customer").
		Preload("Assignments").
		Order("scheduled_end_at ASC").
		Find(&jobs).Error

	return jobs, err
}

// GetJobsRequiringFollowUp retrieves jobs requiring follow-up
func (r *JobRepository) GetJobsRequiringFollowUp() ([]models.Job, error) {
	var jobs []models.Job
	err := r.db.
		Where("requires_follow_up = ?", true).
		Where("follow_up_date IS NOT NULL").
		Preload("Customer").
		Order("follow_up_date ASC").
		Find(&jobs).Error

	return jobs, err
}

// AddAssignment adds a technician assignment to a job
func (r *JobRepository) AddAssignment(assignment *models.JobAssignment) error {
	return r.db.Create(assignment).Error
}

// RemoveAssignment soft deletes a job assignment
func (r *JobRepository) RemoveAssignment(assignmentID uuid.UUID) error {
	return r.db.Delete(&models.JobAssignment{}, assignmentID).Error
}

// GetJobAssignments retrieves all assignments for a job
func (r *JobRepository) GetJobAssignments(jobID uuid.UUID) ([]models.JobAssignment, error) {
	var assignments []models.JobAssignment
	err := r.db.
		Where("job_id = ?", jobID).
		Order("assigned_at DESC").
		Find(&assignments).Error

	return assignments, err
}

// AddMaterial adds a material to a job
func (r *JobRepository) AddMaterial(material *models.JobMaterial) error {
	return r.db.Create(material).Error
}

// UpdateMaterial updates a job material
func (r *JobRepository) UpdateMaterial(material *models.JobMaterial) error {
	return r.db.Save(material).Error
}

// DeleteMaterial soft deletes a job material
func (r *JobRepository) DeleteMaterial(materialID uuid.UUID) error {
	return r.db.Delete(&models.JobMaterial{}, materialID).Error
}

// GetJobMaterials retrieves all materials for a job
func (r *JobRepository) GetJobMaterials(jobID uuid.UUID) ([]models.JobMaterial, error) {
	var materials []models.JobMaterial
	err := r.db.
		Where("job_id = ?", jobID).
		Order("created_at DESC").
		Find(&materials).Error

	return materials, err
}

// AddNote adds a note to a job
func (r *JobRepository) AddNote(note *models.JobNote) error {
	return r.db.Create(note).Error
}

// GetJobNotes retrieves notes for a job
func (r *JobRepository) GetJobNotes(jobID uuid.UUID, includeInternal bool) ([]models.JobNote, error) {
	var notes []models.JobNote
	query := r.db.Where("job_id = ?", jobID)

	if !includeInternal {
		query = query.Where("is_internal = ?", false)
	}

	err := query.
		Order("created_at DESC").
		Find(&notes).Error

	return notes, err
}

// GetJobStats retrieves job statistics
func (r *JobRepository) GetJobStats() (map[string]interface{}, error) {
	stats := make(map[string]interface{})

	// Count by status
	var statusCounts []struct {
		Status models.JobStatus
		Count  int64
	}
	if err := r.db.Model(&models.Job{}).
		Select("status, COUNT(*) as count").
		Group("status").
		Scan(&statusCounts).Error; err != nil {
		return nil, err
	}
	stats["by_status"] = statusCounts

	// Count by priority
	var priorityCounts []struct {
		Priority models.JobPriority
		Count    int64
	}
	if err := r.db.Model(&models.Job{}).
		Select("priority, COUNT(*) as count").
		Group("priority").
		Scan(&priorityCounts).Error; err != nil {
		return nil, err
	}
	stats["by_priority"] = priorityCounts

	// Count by type
	var typeCounts []struct {
		JobType models.JobType
		Count   int64
	}
	if err := r.db.Model(&models.Job{}).
		Select("job_type, COUNT(*) as count").
		Group("job_type").
		Scan(&typeCounts).Error; err != nil {
		return nil, err
	}
	stats["by_type"] = typeCounts

	// Total jobs
	var total int64
	if err := r.db.Model(&models.Job{}).Count(&total).Error; err != nil {
		return nil, err
	}
	stats["total"] = total

	// Average cost (use COALESCE to handle NULL when no jobs have costs)
	var avgCost float64
	if err := r.db.Model(&models.Job{}).
		Select("COALESCE(AVG(actual_cost), 0)").
		Where("actual_cost > 0").
		Scan(&avgCost).Error; err != nil {
		return nil, err
	}
	stats["avg_cost"] = avgCost

	return stats, nil
}

// GetTechnicianWorkload retrieves workload statistics for a technician
func (r *JobRepository) GetTechnicianWorkload(userID uuid.UUID) (map[string]interface{}, error) {
	workload := make(map[string]interface{})

	// Active jobs count
	var activeCount int64
	if err := r.db.Model(&models.Job{}).
		Joins("JOIN job_assignments ON job_assignments.job_id = jobs.id").
		Where("job_assignments.user_id = ?", userID).
		Where("job_assignments.deleted_at IS NULL").
		Where("jobs.status IN ?", []models.JobStatus{
			models.JobStatusScheduled,
			models.JobStatusInProgress,
		}).
		Count(&activeCount).Error; err != nil {
		return nil, err
	}
	workload["active_jobs"] = activeCount

	// Total hours worked (use COALESCE to handle NULL when no hours recorded)
	var totalHours float64
	if err := r.db.Model(&models.JobAssignment{}).
		Select("COALESCE(SUM(hours_worked), 0)").
		Where("user_id = ?", userID).
		Scan(&totalHours).Error; err != nil {
		return nil, err
	}
	workload["total_hours"] = totalHours

	// Jobs by status
	var statusCounts []struct {
		Status models.JobStatus
		Count  int64
	}
	if err := r.db.Table("jobs").
		Select("jobs.status, COUNT(*) as count").
		Joins("JOIN job_assignments ON job_assignments.job_id = jobs.id").
		Where("job_assignments.user_id = ?", userID).
		Where("job_assignments.deleted_at IS NULL").
		Group("jobs.status").
		Scan(&statusCounts).Error; err != nil {
		return nil, err
	}
	workload["by_status"] = statusCounts

	return workload, nil
}

// CreateStatusTransition creates a new status transition record
func (r *JobRepository) CreateStatusTransition(transition *models.JobStatusTransition) error {
	return r.db.Create(transition).Error
}

// GetStatusHistory retrieves status transition history for a job
func (r *JobRepository) GetStatusHistory(jobID uuid.UUID, limit, offset int, sortOrder string) ([]models.JobStatusTransition, int64, error) {
	var transitions []models.JobStatusTransition
	var total int64

	query := r.db.Model(&models.JobStatusTransition{}).Where("job_id = ?", jobID)

	// Count total
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// Apply sorting
	order := "transitioned_at DESC"
	if sortOrder == "asc" {
		order = "transitioned_at ASC"
	}

	// Get transitions with preloaded relationships
	err := r.db.
		Where("job_id = ?", jobID).
		Preload("ChangedByUser").
		Preload("Job").
		Order(order).
		Limit(limit).
		Offset(offset).
		Find(&transitions).Error

	return transitions, total, err
}
