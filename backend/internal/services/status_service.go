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
	"gorm.io/gorm/clause"

	"github.com/javaknight1/servicepro/backend/internal/models"
	"github.com/javaknight1/servicepro/backend/internal/repository"
	"github.com/javaknight1/servicepro/backend/pkg/notifications"
)

var (
	// ErrConcurrentUpdate indicates concurrent update conflict
	ErrConcurrentUpdate = errors.New("concurrent update detected")

	// ErrJobLocked indicates the job is locked by another operation
	ErrJobLocked = errors.New("job is locked by another operation")
)

// StatusServiceInterface defines the interface for status management
type StatusServiceInterface interface {
	// Change job status with validation and notifications
	ChangeStatus(ctx context.Context, req *models.StatusChangeRequest, changedBy uuid.UUID) (*models.StatusChangeResponse, error)

	// Get status history for a job
	GetStatusHistory(ctx context.Context, jobID uuid.UUID, limit, offset int) (*models.StatusHistoryResponse, error)

	// Get allowed transitions for current status
	GetAllowedTransitions(ctx context.Context, jobID uuid.UUID) (*models.AllowedTransitionsResponse, error)

	// Bulk status change (for batch operations)
	BulkChangeStatus(ctx context.Context, requests []*models.StatusChangeRequest, changedBy uuid.UUID) ([]*models.StatusChangeResponse, []error)

	// Validate transition without applying
	ValidateStatusChange(ctx context.Context, jobID uuid.UUID, toStatus models.JobStatus) error
}

// StatusService implements status management with state machine pattern
type StatusService struct {
	db                  *gorm.DB
	jobRepo             repository.JobRepositoryInterface
	notificationService notifications.NotificationServiceInterface
	logger              *log.Logger

	// Mutex map for job-level locking
	lockMu   sync.RWMutex
	jobLocks map[uuid.UUID]*sync.Mutex

	// Metrics
	transitionCount sync.Map // map[string]int64 for tracking transitions
}

// NewStatusService creates a new status service
func NewStatusService(
	db *gorm.DB,
	jobRepo repository.JobRepositoryInterface,
	notificationService notifications.NotificationServiceInterface,
) *StatusService {
	return &StatusService{
		db:                  db,
		jobRepo:             jobRepo,
		notificationService: notificationService,
		logger:              log.New(log.Writer(), "[StatusService] ", log.LstdFlags),
		jobLocks:            make(map[uuid.UUID]*sync.Mutex),
	}
}

// acquireJobLock acquires a lock for a specific job
func (s *StatusService) acquireJobLock(jobID uuid.UUID) *sync.Mutex {
	s.lockMu.Lock()
	defer s.lockMu.Unlock()

	lock, exists := s.jobLocks[jobID]
	if !exists {
		lock = &sync.Mutex{}
		s.jobLocks[jobID] = lock
	}
	return lock
}

// releaseJobLock releases and removes the lock for a job (cleanup)
func (s *StatusService) releaseJobLock(jobID uuid.UUID) {
	s.lockMu.Lock()
	defer s.lockMu.Unlock()

	// Clean up lock after use to prevent memory leak
	delete(s.jobLocks, jobID)
}

// ChangeStatus changes job status with full validation and transaction safety
func (s *StatusService) ChangeStatus(ctx context.Context, req *models.StatusChangeRequest, changedBy uuid.UUID) (*models.StatusChangeResponse, error) {
	startTime := time.Now()
	s.logger.Printf("Attempting status change for job %s from user %s to status %s", req.JobID, changedBy, req.ToStatus)

	// Acquire job-specific lock
	jobLock := s.acquireJobLock(req.JobID)
	jobLock.Lock()
	defer jobLock.Unlock()
	defer s.releaseJobLock(req.JobID)

	var response *models.StatusChangeResponse
	var job *models.Job

	// Use transaction for atomic operation
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Fetch job with pessimistic locking (SELECT ... FOR UPDATE)
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Preload("Assignments").
			First(&job, "id = ?", req.JobID).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrJobNotFound
			}
			return fmt.Errorf("failed to fetch job: %w", err)
		}

		// Store original status
		originalStatus := job.Status

		// Validate transition
		if err := models.ValidateTransition(job, req.ToStatus, req.Notes); err != nil {
			s.logger.Printf("Status transition validation failed: %v", err)
			return err
		}

		// No-op if status hasn't changed
		if originalStatus == req.ToStatus {
			s.logger.Printf("Status unchanged for job %s, skipping update", req.JobID)
			response = &models.StatusChangeResponse{
				JobID:          job.ID,
				JobNumber:      job.JobNumber,
				PreviousStatus: originalStatus,
				NewStatus:      req.ToStatus,
				Reason:         req.Reason,
				Notes:          req.Notes,
				ChangedBy:      changedBy,
				ChangedAt:      time.Now(),
			}
			return nil
		}

		// Update job status
		job.Status = req.ToStatus
		job.UpdatedBy = &changedBy

		// Update timestamps based on status
		now := time.Now()
		switch req.ToStatus {
		case models.JobStatusInProgress:
			if job.ActualStartAt == nil {
				job.ActualStartAt = &now
			}
		case models.JobStatusCompleted:
			if job.ActualEndAt == nil {
				job.ActualEndAt = &now
			}
			// Calculate actual duration if both start and end are set
			if job.ActualStartAt != nil && job.ActualEndAt != nil {
				duration := job.ActualEndAt.Sub(*job.ActualStartAt)
				job.ActualDuration = int(duration.Minutes())
			}
		}

		// Save job
		if err := tx.Save(&job).Error; err != nil {
			return fmt.Errorf("failed to update job status: %w", err)
		}

		// Create status transition record
		transition := &models.JobStatusTransition{
			JobID:          job.ID,
			FromStatus:     originalStatus,
			ToStatus:       req.ToStatus,
			Reason:         req.Reason,
			Notes:          req.Notes,
			ChangedBy:      changedBy,
			TransitionedAt: now,
		}

		if err := tx.Create(transition).Error; err != nil {
			return fmt.Errorf("failed to create transition record: %w", err)
		}

		// Build response
		response = &models.StatusChangeResponse{
			JobID:          job.ID,
			JobNumber:      job.JobNumber,
			PreviousStatus: originalStatus,
			NewStatus:      req.ToStatus,
			Reason:         req.Reason,
			Notes:          req.Notes,
			ChangedBy:      changedBy,
			ChangedAt:      now,
		}

		// Update metrics
		s.incrementTransitionCount(originalStatus, req.ToStatus)

		s.logger.Printf("Successfully changed status for job %s from %s to %s in %v",
			job.JobNumber, originalStatus, req.ToStatus, time.Since(startTime))

		return nil
	})

	if err != nil {
		return nil, err
	}

	// Send notifications asynchronously (outside transaction)
	go s.sendStatusChangeNotifications(job, response)

	return response, nil
}

// GetStatusHistory retrieves status change history for a job
func (s *StatusService) GetStatusHistory(ctx context.Context, jobID uuid.UUID, limit, offset int) (*models.StatusHistoryResponse, error) {
	s.logger.Printf("Fetching status history for job %s", jobID)

	var transitions []models.JobStatusTransition
	var total int64

	// Set defaults
	if limit <= 0 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}

	// Count total
	if err := s.db.WithContext(ctx).Model(&models.JobStatusTransition{}).
		Where("job_id = ?", jobID).
		Count(&total).Error; err != nil {
		return nil, fmt.Errorf("failed to count transitions: %w", err)
	}

	// Fetch transitions
	if err := s.db.WithContext(ctx).
		Where("job_id = ?", jobID).
		Preload("Job").
		Preload("ChangedByUser").
		Order("transitioned_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&transitions).Error; err != nil {
		return nil, fmt.Errorf("failed to fetch transitions: %w", err)
	}

	// Convert to response
	transitionResponses := make([]models.StatusTransitionResponse, len(transitions))
	for i, transition := range transitions {
		transitionResponses[i] = transition.ToResponse()
	}

	return &models.StatusHistoryResponse{
		Transitions: transitionResponses,
		Total:       total,
	}, nil
}

// GetAllowedTransitions returns allowed status transitions for a job
func (s *StatusService) GetAllowedTransitions(ctx context.Context, jobID uuid.UUID) (*models.AllowedTransitionsResponse, error) {
	s.logger.Printf("Fetching allowed transitions for job %s", jobID)

	// Fetch job
	job, err := s.jobRepo.GetByID(jobID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrJobNotFound
		}
		return nil, fmt.Errorf("failed to fetch job: %w", err)
	}

	// Get allowed transitions
	allowedTransitions := models.GetAllowedTransitions(job.Status)

	return &models.AllowedTransitionsResponse{
		JobID:              job.ID,
		CurrentStatus:      job.Status,
		AllowedTransitions: allowedTransitions,
		IsTerminal:         models.IsTerminalStatus(job.Status),
	}, nil
}

// BulkChangeStatus changes status for multiple jobs concurrently
func (s *StatusService) BulkChangeStatus(ctx context.Context, requests []*models.StatusChangeRequest, changedBy uuid.UUID) ([]*models.StatusChangeResponse, []error) {
	s.logger.Printf("Processing bulk status change for %d jobs", len(requests))

	var (
		wg        sync.WaitGroup
		mu        sync.Mutex
		responses []*models.StatusChangeResponse
		errs      []error
	)

	for _, req := range requests {
		wg.Add(1)
		go func(r *models.StatusChangeRequest) {
			defer wg.Done()

			response, err := s.ChangeStatus(ctx, r, changedBy)

			mu.Lock()
			defer mu.Unlock()

			if err != nil {
				errs = append(errs, fmt.Errorf("job %s: %w", r.JobID, err))
			} else {
				responses = append(responses, response)
			}
		}(req)
	}

	wg.Wait()

	s.logger.Printf("Bulk status change completed: %d succeeded, %d failed", len(responses), len(errs))

	return responses, errs
}

// ValidateStatusChange validates a status change without applying it
func (s *StatusService) ValidateStatusChange(ctx context.Context, jobID uuid.UUID, toStatus models.JobStatus) error {
	s.logger.Printf("Validating status change for job %s to %s", jobID, toStatus)

	// Fetch job with assignments for validation
	var job models.Job
	if err := s.db.WithContext(ctx).
		Preload("Assignments").
		First(&job, "id = ?", jobID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrJobNotFound
		}
		return fmt.Errorf("failed to fetch job: %w", err)
	}

	// Validate transition
	return models.ValidateTransition(&job, toStatus, nil)
}

// sendStatusChangeNotifications sends notifications for status changes
func (s *StatusService) sendStatusChangeNotifications(job *models.Job, response *models.StatusChangeResponse) {
	if s.notificationService == nil {
		s.logger.Printf("Notification service not available, skipping notifications")
		return
	}

	ctx := context.Background()

	// Prepare notification data
	subject := fmt.Sprintf("Job Status Updated: %s - %s", job.JobNumber, response.NewStatus)

	// Notify assigned technicians
	for _, assignment := range job.Assignments {
		if assignment.User == nil {
			continue
		}

		notificationReq := &notifications.NotificationRequest{
			RecipientID:    assignment.UserID,
			RecipientEmail: assignment.User.Email,
			Channel:        notifications.ChannelEmail,
			Priority:       s.getNotificationPriority(response.NewStatus),
			Subject:        subject,
			TemplateData: map[string]interface{}{
				"JobNumber":      job.JobNumber,
				"JobTitle":       job.Title,
				"PreviousStatus": response.PreviousStatus,
				"NewStatus":      response.NewStatus,
				"Reason":         response.Reason,
				"Notes":          response.Notes,
				"ChangedAt":      response.ChangedAt.Format(time.RFC1123),
			},
		}

		if _, err := s.notificationService.SendNotification(ctx, notificationReq); err != nil {
			s.logger.Printf("Failed to send notification to technician %s: %v", assignment.UserID, err)
		}
	}

	s.logger.Printf("Status change notifications sent for job %s", job.JobNumber)
}

// getNotificationPriority determines notification priority based on status
func (s *StatusService) getNotificationPriority(status models.JobStatus) notifications.NotificationPriority {
	switch status {
	case models.JobStatusInProgress:
		return notifications.PriorityHigh
	case models.JobStatusCompleted:
		return notifications.PriorityNormal
	case models.JobStatusCancelled:
		return notifications.PriorityHigh
	case models.JobStatusOnHold:
		return notifications.PriorityNormal
	default:
		return notifications.PriorityNormal
	}
}

// incrementTransitionCount increments metrics for a transition
func (s *StatusService) incrementTransitionCount(from, to models.JobStatus) {
	key := fmt.Sprintf("%s_to_%s", from, to)

	val, _ := s.transitionCount.LoadOrStore(key, int64(0))
	count := val.(int64)
	s.transitionCount.Store(key, count+1)
}

// GetTransitionMetrics returns transition metrics (for monitoring/analytics)
func (s *StatusService) GetTransitionMetrics() map[string]int64 {
	metrics := make(map[string]int64)

	s.transitionCount.Range(func(key, value interface{}) bool {
		metrics[key.(string)] = value.(int64)
		return true
	})

	return metrics
}
