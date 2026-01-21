package repository

import (
	"errors"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/javaknight1/servicepro/backend/internal/models"
)

// ScheduleRepository implements ScheduleRepositoryInterface
// Interface definition is in interface.go
type ScheduleRepository struct {
	db *gorm.DB
}

// NewScheduleRepository creates a new schedule repository
func NewScheduleRepository(db *gorm.DB) *ScheduleRepository {
	return &ScheduleRepository{db: db}
}

// Create creates a new schedule
func (r *ScheduleRepository) Create(schedule *models.Schedule) error {
	return r.db.Create(schedule).Error
}

// GetByID retrieves a schedule by ID
func (r *ScheduleRepository) GetByID(id uuid.UUID) (*models.Schedule, error) {
	var schedule models.Schedule
	err := r.db.Preload("Job").
		Preload("RecurringSchedule").
		First(&schedule, "id = ?", id).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("schedule not found")
		}
		return nil, err
	}

	return &schedule, nil
}

// Update updates a schedule
func (r *ScheduleRepository) Update(schedule *models.Schedule) error {
	return r.db.Save(schedule).Error
}

// Delete soft deletes a schedule
func (r *ScheduleRepository) Delete(id uuid.UUID) error {
	return r.db.Delete(&models.Schedule{}, "id = ?", id).Error
}

// List retrieves schedules with pagination and filters
func (r *ScheduleRepository) List(params *models.ScheduleQueryParams) ([]models.Schedule, int64, error) {
	var schedules []models.Schedule
	var total int64

	query := r.db.Model(&models.Schedule{})

	// Apply filters
	if params.StartDate != nil {
		query = query.Where("start_time >= ?", params.StartDate)
	}
	if params.EndDate != nil {
		query = query.Where("end_time <= ?", params.EndDate)
	}
	if params.TechID != nil {
		query = query.Where("? = ANY(assigned_tech_ids)", params.TechID)
	}
	if params.Confirmed != nil {
		query = query.Where("is_confirmed = ?", *params.Confirmed)
	}
	if params.Cancelled != nil {
		query = query.Where("is_cancelled = ?", *params.Cancelled)
	}

	// Count total
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// Apply pagination
	page := params.Page
	if page < 1 {
		page = 1
	}
	pageSize := params.PageSize
	if pageSize < 1 {
		pageSize = 50
	}

	offset := (page - 1) * pageSize

	// Fetch schedules
	err := query.Preload("Job").
		Preload("RecurringSchedule").
		Order("start_time ASC").
		Limit(pageSize).
		Offset(offset).
		Find(&schedules).Error

	return schedules, total, err
}

// GetByDateRange retrieves schedules within a date range
func (r *ScheduleRepository) GetByDateRange(startDate, endDate time.Time) ([]models.Schedule, error) {
	var schedules []models.Schedule
	err := r.db.Where("start_time >= ? AND end_time <= ?", startDate, endDate).
		Preload("Job").
		Order("start_time ASC").
		Find(&schedules).Error

	return schedules, err
}

// GetByTechnicianAndDateRange retrieves schedules for a technician within a date range
func (r *ScheduleRepository) GetByTechnicianAndDateRange(techID uuid.UUID, startDate, endDate time.Time) ([]models.Schedule, error) {
	var schedules []models.Schedule
	err := r.db.Where("? = ANY(assigned_tech_ids) AND start_time >= ? AND end_time <= ?",
		techID, startDate, endDate).
		Preload("Job").
		Order("start_time ASC").
		Find(&schedules).Error

	return schedules, err
}

// GetByJobID retrieves all schedules for a job
func (r *ScheduleRepository) GetByJobID(jobID uuid.UUID) ([]models.Schedule, error) {
	var schedules []models.Schedule
	err := r.db.Where("job_id = ?", jobID).
		Order("start_time ASC").
		Find(&schedules).Error

	return schedules, err
}

// DetectConflicts detects scheduling conflicts for a given schedule
func (r *ScheduleRepository) DetectConflicts(schedule *models.Schedule) ([]models.ScheduleConflict, error) {
	var conflicts []models.ScheduleConflict

	// Find overlapping schedules with same technicians
	if len(schedule.AssignedTechIDs) > 0 {
		var overlapping []models.Schedule
		err := r.db.Where("id != ? AND is_cancelled = ? AND deleted_at IS NULL", schedule.ID, false).
			Where("assigned_tech_ids && ?", schedule.AssignedTechIDs).
			Where("(start_time, end_time) OVERLAPS (?, ?)", schedule.StartTime, schedule.EndTime).
			Find(&overlapping).Error

		if err != nil {
			return nil, err
		}

		// Create conflict records
		for _, overlap := range overlapping {
			conflict := models.ScheduleConflict{
				ScheduleID1:  schedule.ID,
				ScheduleID2:  overlap.ID,
				ConflictType: models.ConflictTypeTechnicianOverlap,
				Severity:     models.ConflictSeverityHigh,
				Description:  "Technician assigned to overlapping schedules",
				DetectedAt:   time.Now(),
			}
			conflicts = append(conflicts, conflict)
		}
	}

	return conflicts, nil
}

// GetConflicts retrieves conflicts for a schedule
func (r *ScheduleRepository) GetConflicts(scheduleID uuid.UUID) ([]models.ScheduleConflict, error) {
	var conflicts []models.ScheduleConflict
	err := r.db.Where("(schedule_id_1 = ? OR schedule_id_2 = ?) AND is_resolved = ?",
		scheduleID, scheduleID, false).
		Preload("Schedule1").
		Preload("Schedule2").
		Find(&conflicts).Error

	return conflicts, err
}

// ResolveConflict marks a conflict as resolved
func (r *ScheduleRepository) ResolveConflict(conflictID uuid.UUID, resolvedBy uuid.UUID, notes *string) error {
	now := time.Now()
	return r.db.Model(&models.ScheduleConflict{}).
		Where("id = ?", conflictID).
		Updates(map[string]interface{}{
			"is_resolved":      true,
			"resolved_at":      &now,
			"resolved_by":      &resolvedBy,
			"resolution_notes": notes,
		}).Error
}

// CreateRecurring creates a new recurring schedule
func (r *ScheduleRepository) CreateRecurring(recurring *models.RecurringSchedule) error {
	return r.db.Create(recurring).Error
}

// GetRecurringByID retrieves a recurring schedule by ID
func (r *ScheduleRepository) GetRecurringByID(id uuid.UUID) (*models.RecurringSchedule, error) {
	var recurring models.RecurringSchedule
	err := r.db.First(&recurring, "id = ?", id).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("recurring schedule not found")
		}
		return nil, err
	}

	return &recurring, nil
}

// UpdateRecurring updates a recurring schedule
func (r *ScheduleRepository) UpdateRecurring(recurring *models.RecurringSchedule) error {
	return r.db.Save(recurring).Error
}

// DeleteRecurring soft deletes a recurring schedule
func (r *ScheduleRepository) DeleteRecurring(id uuid.UUID) error {
	return r.db.Delete(&models.RecurringSchedule{}, "id = ?", id).Error
}

// GetActiveRecurring retrieves all active recurring schedules
func (r *ScheduleRepository) GetActiveRecurring() ([]models.RecurringSchedule, error) {
	var recurring []models.RecurringSchedule
	err := r.db.Where("is_active = ?", true).
		Find(&recurring).Error

	return recurring, err
}

// GetScheduleCount returns the total number of schedules
func (r *ScheduleRepository) GetScheduleCount() (int64, error) {
	var count int64
	err := r.db.Model(&models.Schedule{}).Count(&count).Error
	return count, err
}

// GetConflictCount returns the total number of unresolved conflicts
func (r *ScheduleRepository) GetConflictCount() (int64, error) {
	var count int64
	err := r.db.Model(&models.ScheduleConflict{}).
		Where("is_resolved = ?", false).
		Count(&count).Error
	return count, err
}
