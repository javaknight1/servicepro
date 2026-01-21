package services

import (
	"context"
	"errors"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/javaknight1/servicepro/backend/internal/models"
	"github.com/javaknight1/servicepro/backend/internal/repository"
)

var (
	// ErrTechnicianNotAvailable indicates technician is not available for the time slot
	ErrTechnicianNotAvailable = errors.New("technician is not available for the requested time slot")

	// ErrSkillMismatch indicates technician doesn't have required skills
	ErrSkillMismatch = errors.New("technician does not have required skills for this job")

	// ErrScheduleConflict indicates scheduling conflict exists
	ErrScheduleConflict = errors.New("schedule conflict detected")

	// ErrTechnicianNotFound indicates technician was not found
	ErrTechnicianNotFound = errors.New("technician not found")

	// ErrAssignmentNotFound indicates assignment was not found
	ErrAssignmentNotFound = errors.New("assignment not found")

	// ErrInvalidTimeSlot indicates invalid time slot
	ErrInvalidTimeSlot = errors.New("invalid time slot")
)

// TechnicianSkill represents a technician's skill
type TechnicianSkill string

const (
	SkillHVACInstallation TechnicianSkill = "hvac_installation"
	SkillHVACRepair       TechnicianSkill = "hvac_repair"
	SkillHVACMaintenance  TechnicianSkill = "hvac_maintenance"
	SkillPlumbing         TechnicianSkill = "plumbing"
	SkillElectrical       TechnicianSkill = "electrical"
	SkillCarpentry        TechnicianSkill = "carpentry"
	SkillGeneralRepair    TechnicianSkill = "general_repair"
)

// TechnicianAvailability represents technician availability for a time slot
type TechnicianAvailability struct {
	TechnicianID uuid.UUID `json:"technician_id"`
	Available    bool      `json:"available"`
	Conflicts    []string  `json:"conflicts,omitempty"`
	Workload     int       `json:"workload"` // Number of active assignments
	Skills       []string  `json:"skills"`
	SkillMatch   float64   `json:"skill_match"` // 0-100 percentage
}

// AssignmentRequest represents a request to assign a technician
type AssignmentRequest struct {
	JobID              uuid.UUID         `json:"job_id" binding:"required"`
	TechnicianID       uuid.UUID         `json:"technician_id" binding:"required"`
	Role               string            `json:"role" binding:"required"`
	RequiredSkills     []TechnicianSkill `json:"required_skills,omitempty"`
	PreferredStartTime *time.Time        `json:"preferred_start_time,omitempty"`
	EstimatedDuration  int               `json:"estimated_duration,omitempty"` // in minutes
}

// BulkAssignmentRequest represents a request to assign multiple technicians
type BulkAssignmentRequest struct {
	JobID       uuid.UUID           `json:"job_id" binding:"required"`
	Assignments []AssignmentRequest `json:"assignments" binding:"required,min=1"`
}

// AvailabilityCheckRequest represents a request to check availability
type AvailabilityCheckRequest struct {
	StartTime          time.Time         `json:"start_time" binding:"required"`
	EndTime            time.Time         `json:"end_time" binding:"required"`
	RequiredSkills     []TechnicianSkill `json:"required_skills,omitempty"`
	ExcludeTechnicians []uuid.UUID       `json:"exclude_technicians,omitempty"`
}

// AssignmentServiceInterface defines the interface for assignment service
type AssignmentServiceInterface interface {
	// Assignment operations
	CreateAssignment(ctx context.Context, req *AssignmentRequest, assignedBy uuid.UUID) (*models.JobAssignment, error)
	BulkAssignTechnicians(ctx context.Context, req *BulkAssignmentRequest, assignedBy uuid.UUID) ([]models.JobAssignment, []error)
	RemoveAssignment(ctx context.Context, assignmentID uuid.UUID, removedBy uuid.UUID) error
	UpdateAssignment(ctx context.Context, assignmentID uuid.UUID, role string, updatedBy uuid.UUID) error

	// Availability operations
	CheckTechnicianAvailability(ctx context.Context, technicianID uuid.UUID, startTime, endTime time.Time) (*TechnicianAvailability, error)
	GetAvailableTechnicians(ctx context.Context, req *AvailabilityCheckRequest) ([]*TechnicianAvailability, error)
	DetectConflicts(ctx context.Context, technicianID uuid.UUID, startTime, endTime time.Time, excludeJobID *uuid.UUID) ([]models.Job, error)

	// Skills operations
	CheckSkillMatch(ctx context.Context, technicianID uuid.UUID, requiredSkills []TechnicianSkill) (float64, error)
	GetTechniciansBySkill(ctx context.Context, skill TechnicianSkill) ([]uuid.UUID, error)

	// Workload operations
	GetTechnicianWorkload(ctx context.Context, technicianID uuid.UUID, startDate, endDate time.Time) (int, error)
	OptimizeAssignments(ctx context.Context, jobID uuid.UUID, requiredSkills []TechnicianSkill, startTime, endTime time.Time) ([]*TechnicianAvailability, error)
}

// AssignmentService handles assignment business logic
type AssignmentService struct {
	jobRepo  repository.JobRepositoryInterface
	userRepo repository.UserRepositoryGormInterface
	db       *gorm.DB
	mu       sync.RWMutex
	logger   *log.Logger
}

// NewAssignmentService creates a new assignment service
func NewAssignmentService(
	jobRepo repository.JobRepositoryInterface,
	userRepo repository.UserRepositoryGormInterface,
	db *gorm.DB,
) *AssignmentService {
	return &AssignmentService{
		jobRepo:  jobRepo,
		userRepo: userRepo,
		db:       db,
		logger:   log.New(log.Writer(), "[AssignmentService] ", log.LstdFlags),
	}
}

// CreateAssignment creates a new assignment with validation
func (s *AssignmentService) CreateAssignment(ctx context.Context, req *AssignmentRequest, assignedBy uuid.UUID) (*models.JobAssignment, error) {
	s.logger.Printf("Creating assignment for job %s, technician %s", req.JobID, req.TechnicianID)

	// Validate job exists
	job, err := s.jobRepo.GetByID(req.JobID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrJobNotFound
		}
		s.logger.Printf("Error fetching job: %v", err)
		return nil, err
	}

	// Validate technician exists
	technician, err := s.userRepo.GetByID(req.TechnicianID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrTechnicianNotFound
		}
		s.logger.Printf("Error fetching technician: %v", err)
		return nil, err
	}

	// Verify technician role
	if technician.Role != models.UserRoleTechnician {
		s.logger.Printf("User %s is not a technician (role: %s)", req.TechnicianID, technician.Role)
		return nil, errors.New("assigned user must have technician role")
	}

	// Check availability if job has scheduled times
	if job.ScheduledStartAt != nil && job.ScheduledEndAt != nil {
		availability, err := s.CheckTechnicianAvailability(ctx, req.TechnicianID, *job.ScheduledStartAt, *job.ScheduledEndAt)
		if err != nil {
			s.logger.Printf("Error checking availability: %v", err)
			return nil, err
		}

		if !availability.Available {
			s.logger.Printf("Technician %s is not available. Conflicts: %v", req.TechnicianID, availability.Conflicts)
			return nil, ErrTechnicianNotAvailable
		}
	}

	// Check skill match if required skills are specified
	if len(req.RequiredSkills) > 0 {
		skillMatch, err := s.CheckSkillMatch(ctx, req.TechnicianID, req.RequiredSkills)
		if err != nil {
			s.logger.Printf("Error checking skill match: %v", err)
			return nil, err
		}

		// Require at least 50% skill match
		if skillMatch < 50.0 {
			s.logger.Printf("Insufficient skill match: %.2f%%", skillMatch)
			return nil, ErrSkillMismatch
		}
	}

	// Create assignment
	now := time.Now()
	assignment := &models.JobAssignment{
		JobID:      req.JobID,
		UserID:     req.TechnicianID,
		Role:       req.Role,
		AssignedAt: now,
	}

	if err := s.jobRepo.AddAssignment(assignment); err != nil {
		s.logger.Printf("Error creating assignment: %v", err)
		return nil, err
	}

	s.logger.Printf("Successfully created assignment %s", assignment.ID)
	return assignment, nil
}

// BulkAssignTechnicians assigns multiple technicians concurrently
func (s *AssignmentService) BulkAssignTechnicians(ctx context.Context, req *BulkAssignmentRequest, assignedBy uuid.UUID) ([]models.JobAssignment, []error) {
	s.logger.Printf("Bulk assigning %d technicians to job %s", len(req.Assignments), req.JobID)

	var (
		wg          sync.WaitGroup
		mu          sync.Mutex
		assignments []models.JobAssignment
		errs        []error
	)

	// Process assignments concurrently
	for i := range req.Assignments {
		wg.Add(1)
		go func(assignReq AssignmentRequest) {
			defer wg.Done()

			// Ensure JobID is set from bulk request
			assignReq.JobID = req.JobID

			assignment, err := s.CreateAssignment(ctx, &assignReq, assignedBy)

			mu.Lock()
			defer mu.Unlock()

			if err != nil {
				errs = append(errs, fmt.Errorf("technician %s: %w", assignReq.TechnicianID, err))
			} else {
				assignments = append(assignments, *assignment)
			}
		}(req.Assignments[i])
	}

	wg.Wait()

	s.logger.Printf("Bulk assignment completed. Success: %d, Errors: %d", len(assignments), len(errs))
	return assignments, errs
}

// RemoveAssignment removes an assignment
func (s *AssignmentService) RemoveAssignment(ctx context.Context, assignmentID uuid.UUID, removedBy uuid.UUID) error {
	s.logger.Printf("Removing assignment %s by user %s", assignmentID, removedBy)

	if err := s.jobRepo.RemoveAssignment(assignmentID); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrAssignmentNotFound
		}
		s.logger.Printf("Error removing assignment: %v", err)
		return err
	}

	s.logger.Printf("Successfully removed assignment %s", assignmentID)
	return nil
}

// UpdateAssignment updates an assignment role
func (s *AssignmentService) UpdateAssignment(ctx context.Context, assignmentID uuid.UUID, role string, updatedBy uuid.UUID) error {
	s.logger.Printf("Updating assignment %s role to %s", assignmentID, role)

	// Get existing assignment
	var assignment models.JobAssignment
	if err := s.db.First(&assignment, assignmentID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrAssignmentNotFound
		}
		return err
	}

	// Update role
	assignment.Role = role
	if err := s.db.Save(&assignment).Error; err != nil {
		s.logger.Printf("Error updating assignment: %v", err)
		return err
	}

	s.logger.Printf("Successfully updated assignment %s", assignmentID)
	return nil
}

// CheckTechnicianAvailability checks if a technician is available for a time slot
func (s *AssignmentService) CheckTechnicianAvailability(ctx context.Context, technicianID uuid.UUID, startTime, endTime time.Time) (*TechnicianAvailability, error) {
	s.logger.Printf("Checking availability for technician %s from %s to %s", technicianID, startTime, endTime)

	if endTime.Before(startTime) {
		return nil, ErrInvalidTimeSlot
	}

	// Detect conflicts
	conflicts, err := s.DetectConflicts(ctx, technicianID, startTime, endTime, nil)
	if err != nil {
		return nil, err
	}

	// Get current workload
	workload, err := s.GetTechnicianWorkload(ctx, technicianID, startTime, endTime)
	if err != nil {
		s.logger.Printf("Error getting workload: %v", err)
		workload = 0 // Continue with 0 workload
	}

	// Get skills (placeholder - would need skills table in real implementation)
	skills := s.getTechnicianSkills(ctx, technicianID)

	availability := &TechnicianAvailability{
		TechnicianID: technicianID,
		Available:    len(conflicts) == 0,
		Conflicts:    make([]string, 0),
		Workload:     workload,
		Skills:       skills,
	}

	for _, conflict := range conflicts {
		availability.Conflicts = append(availability.Conflicts,
			fmt.Sprintf("Job %s (%s) from %s to %s",
				conflict.JobNumber, conflict.Title,
				conflict.ScheduledStartAt.Format(time.RFC3339),
				conflict.ScheduledEndAt.Format(time.RFC3339)))
	}

	return availability, nil
}

// GetAvailableTechnicians gets all available technicians for a time slot concurrently
func (s *AssignmentService) GetAvailableTechnicians(ctx context.Context, req *AvailabilityCheckRequest) ([]*TechnicianAvailability, error) {
	s.logger.Printf("Finding available technicians from %s to %s", req.StartTime, req.EndTime)

	// Get all technicians (in real implementation, would query from users table)
	technicians := s.getAllTechnicians(ctx)

	var (
		wg             sync.WaitGroup
		mu             sync.Mutex
		availabilities []*TechnicianAvailability
	)

	// Check availability concurrently
	for _, techID := range technicians {
		// Skip excluded technicians
		excluded := false
		for _, excludeID := range req.ExcludeTechnicians {
			if techID == excludeID {
				excluded = true
				break
			}
		}
		if excluded {
			continue
		}

		wg.Add(1)
		go func(id uuid.UUID) {
			defer wg.Done()

			availability, err := s.CheckTechnicianAvailability(ctx, id, req.StartTime, req.EndTime)
			if err != nil {
				s.logger.Printf("Error checking availability for %s: %v", id, err)
				return
			}

			// Check skill match if required
			if len(req.RequiredSkills) > 0 {
				skillMatch, err := s.CheckSkillMatch(ctx, id, req.RequiredSkills)
				if err != nil {
					s.logger.Printf("Error checking skills for %s: %v", id, err)
					return
				}
				availability.SkillMatch = skillMatch
			} else {
				availability.SkillMatch = 100.0
			}

			mu.Lock()
			availabilities = append(availabilities, availability)
			mu.Unlock()
		}(techID)
	}

	wg.Wait()

	s.logger.Printf("Found %d technicians", len(availabilities))
	return availabilities, nil
}

// DetectConflicts detects scheduling conflicts for a technician
func (s *AssignmentService) DetectConflicts(ctx context.Context, technicianID uuid.UUID, startTime, endTime time.Time, excludeJobID *uuid.UUID) ([]models.Job, error) {
	// Get all jobs assigned to this technician
	jobs, err := s.jobRepo.GetByAssignedUser(technicianID)
	if err != nil {
		return nil, err
	}

	conflicts := make([]models.Job, 0)
	for _, job := range jobs {
		// Skip excluded job
		if excludeJobID != nil && job.ID == *excludeJobID {
			continue
		}

		// Check if job is in active status
		if job.Status != models.JobStatusScheduled && job.Status != models.JobStatusInProgress {
			continue
		}

		// Check time overlap
		if job.ScheduledStartAt != nil && job.ScheduledEndAt != nil {
			if timeOverlaps(*job.ScheduledStartAt, *job.ScheduledEndAt, startTime, endTime) {
				conflicts = append(conflicts, job)
			}
		}
	}

	return conflicts, nil
}

// CheckSkillMatch checks how well a technician's skills match requirements
func (s *AssignmentService) CheckSkillMatch(ctx context.Context, technicianID uuid.UUID, requiredSkills []TechnicianSkill) (float64, error) {
	// Get technician skills (placeholder - would query from skills table)
	techSkills := s.getTechnicianSkillsTyped(ctx, technicianID)

	if len(requiredSkills) == 0 {
		return 100.0, nil
	}

	matchCount := 0
	for _, required := range requiredSkills {
		for _, techSkill := range techSkills {
			if required == techSkill {
				matchCount++
				break
			}
		}
	}

	matchPercentage := (float64(matchCount) / float64(len(requiredSkills))) * 100.0
	return matchPercentage, nil
}

// GetTechniciansBySkill gets all technicians with a specific skill
func (s *AssignmentService) GetTechniciansBySkill(ctx context.Context, skill TechnicianSkill) ([]uuid.UUID, error) {
	// Placeholder - would query from skills table
	technicians := s.getAllTechnicians(ctx)

	result := make([]uuid.UUID, 0)
	for _, techID := range technicians {
		skills := s.getTechnicianSkillsTyped(ctx, techID)
		for _, s := range skills {
			if s == skill {
				result = append(result, techID)
				break
			}
		}
	}

	return result, nil
}

// GetTechnicianWorkload gets the number of active assignments for a technician
func (s *AssignmentService) GetTechnicianWorkload(ctx context.Context, technicianID uuid.UUID, startDate, endDate time.Time) (int, error) {
	jobs, err := s.jobRepo.GetByAssignedUser(technicianID)
	if err != nil {
		return 0, err
	}

	count := 0
	for _, job := range jobs {
		// Count only active jobs
		if job.Status == models.JobStatusScheduled || job.Status == models.JobStatusInProgress {
			// Check if job falls within date range
			if job.ScheduledStartAt != nil {
				if job.ScheduledStartAt.After(startDate) && job.ScheduledStartAt.Before(endDate) {
					count++
				}
			}
		}
	}

	return count, nil
}

// OptimizeAssignments finds the best technicians for a job based on availability and skills
func (s *AssignmentService) OptimizeAssignments(ctx context.Context, jobID uuid.UUID, requiredSkills []TechnicianSkill, startTime, endTime time.Time) ([]*TechnicianAvailability, error) {
	s.logger.Printf("Optimizing assignments for job %s", jobID)

	req := &AvailabilityCheckRequest{
		StartTime:      startTime,
		EndTime:        endTime,
		RequiredSkills: requiredSkills,
	}

	availabilities, err := s.GetAvailableTechnicians(ctx, req)
	if err != nil {
		return nil, err
	}

	// Filter to only available technicians
	available := make([]*TechnicianAvailability, 0)
	for _, a := range availabilities {
		if a.Available {
			available = append(available, a)
		}
	}

	// Sort by skill match (descending) and workload (ascending)
	sortTechniciansByOptimization(available)

	return available, nil
}

// Helper functions

func timeOverlaps(start1, end1, start2, end2 time.Time) bool {
	return start1.Before(end2) && end1.After(start2)
}

func sortTechniciansByOptimization(technicians []*TechnicianAvailability) {
	// Simple bubble sort for demonstration
	n := len(technicians)
	for i := 0; i < n-1; i++ {
		for j := 0; j < n-i-1; j++ {
			// Sort by skill match (desc) then workload (asc)
			if technicians[j].SkillMatch < technicians[j+1].SkillMatch ||
				(technicians[j].SkillMatch == technicians[j+1].SkillMatch && technicians[j].Workload > technicians[j+1].Workload) {
				technicians[j], technicians[j+1] = technicians[j+1], technicians[j]
			}
		}
	}
}

// Placeholder helper functions (would be replaced with real database queries)

func (s *AssignmentService) getAllTechnicians(ctx context.Context) []uuid.UUID {
	// In real implementation, would query users table for technicians
	// For now, return empty slice
	return []uuid.UUID{}
}

func (s *AssignmentService) getTechnicianSkills(ctx context.Context, technicianID uuid.UUID) []string {
	// In real implementation, would query skills table
	// For now, return mock skills
	return []string{"hvac_installation", "hvac_repair", "general_repair"}
}

func (s *AssignmentService) getTechnicianSkillsTyped(ctx context.Context, technicianID uuid.UUID) []TechnicianSkill {
	// In real implementation, would query skills table
	// For now, return mock skills
	return []TechnicianSkill{
		SkillHVACInstallation,
		SkillHVACRepair,
		SkillGeneralRepair,
	}
}
