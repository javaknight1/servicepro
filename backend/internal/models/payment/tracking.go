package payment

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/javaknight1/servicepro/backend/internal/utils"
)

// StatusTracker interface defines methods for tracking payment status
type StatusTracker interface {
	// Status operations
	GetStatus(ctx context.Context, paymentID uuid.UUID) (*PaymentStatusRecord, error)
	GetStatuses(ctx context.Context, query *StatusQuery) ([]PaymentStatusRecord, error)
	UpdateStatus(ctx context.Context, transition *StatusTransition) error
	BulkUpdateStatuses(ctx context.Context, transitions []*StatusTransition) error

	// History operations
	GetHistory(ctx context.Context, paymentID uuid.UUID) ([]PaymentStatusHistory, error)
	GetHistoryWithQuery(ctx context.Context, query *StatusHistoryQuery) (*StatusHistoryResponse, error)
	GetTimeline(ctx context.Context, paymentID uuid.UUID) (*StatusTimeline, error)

	// Statistics and reporting
	GetStatistics(ctx context.Context) (*StatusStatistics, error)
	GetTransitionMetrics(ctx context.Context) (*StatusTransitionMatrix, error)
	DetectAnomalies(ctx context.Context) ([]StatusAnomalyDetection, error)
	GetHealthStatus(ctx context.Context) (*StatusHealth, error)

	// Audit operations
	GetAuditLog(ctx context.Context, query *StatusAuditQuery) ([]StatusAuditLog, error)
	CreateAuditEntry(ctx context.Context, entry *StatusAuditLog) error
}

// StatusTrackerService implements the StatusTracker interface
type StatusTrackerService struct {
	db               *gorm.DB
	notifier         NotificationService
	mu               sync.RWMutex
	cache            map[uuid.UUID]*PaymentStatusRecord
	cacheExpiry      time.Duration
	anomalyDetectors []AnomalyDetector
}

// NotificationService interface for sending notifications
type NotificationService interface {
	SendStatusNotification(ctx context.Context, notification *PaymentStatusNotification) error
	QueueNotification(ctx context.Context, notification *PaymentStatusNotification) error
	CheckDeduplication(ctx context.Context, paymentID, recipientID uuid.UUID, notificationType NotificationType, statusChange string) (bool, error)
}

// AnomalyDetector interface for detecting payment anomalies
type AnomalyDetector interface {
	Detect(ctx context.Context, payment *PaymentStatusRecord, history []PaymentStatusHistory) (*StatusAnomalyDetection, error)
	Name() string
}

// NewStatusTrackerService creates a new status tracker service
func NewStatusTrackerService(db *gorm.DB, notifier NotificationService) *StatusTrackerService {
	return &StatusTrackerService{
		db:               db,
		notifier:         notifier,
		cache:            make(map[uuid.UUID]*PaymentStatusRecord),
		cacheExpiry:      5 * time.Minute,
		anomalyDetectors: []AnomalyDetector{},
	}
}

// RegisterAnomalyDetector registers a new anomaly detector
func (s *StatusTrackerService) RegisterAnomalyDetector(detector AnomalyDetector) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.anomalyDetectors = append(s.anomalyDetectors, detector)
}

// GetStatus retrieves the current status of a payment
func (s *StatusTrackerService) GetStatus(ctx context.Context, paymentID uuid.UUID) (*PaymentStatusRecord, error) {
	// Check cache first
	s.mu.RLock()
	if cached, exists := s.cache[paymentID]; exists {
		s.mu.RUnlock()
		return cached, nil
	}
	s.mu.RUnlock()

	var status PaymentStatusRecord
	err := s.db.WithContext(ctx).
		Where("payment_id = ?", paymentID).
		First(&status).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("payment status not found for payment %s", paymentID)
		}
		return nil, fmt.Errorf("failed to get payment status: %w", err)
	}

	// Update cache
	s.mu.Lock()
	s.cache[paymentID] = &status
	s.mu.Unlock()

	return &status, nil
}

// GetStatuses retrieves multiple payment statuses based on query
func (s *StatusTrackerService) GetStatuses(ctx context.Context, query *StatusQuery) ([]PaymentStatusRecord, error) {
	db := s.db.WithContext(ctx).Model(&PaymentStatusRecord{})

	if len(query.PaymentIDs) > 0 {
		db = db.Where("payment_id IN ?", query.PaymentIDs)
	}

	if len(query.Statuses) > 0 {
		db = db.Where("status IN ?", query.Statuses)
	}

	if len(query.Categories) > 0 {
		db = db.Where("status_category IN ?", query.Categories)
	}

	if query.UpdatedAfter != nil {
		db = db.Where("updated_at >= ?", *query.UpdatedAfter)
	}

	if query.UpdatedBefore != nil {
		db = db.Where("updated_at <= ?", *query.UpdatedBefore)
	}

	if query.FinalOnly {
		finalStatuses := []PaymentStatus{
			StatusSucceeded, StatusPaid, StatusFailed, StatusDeclined,
			StatusExpired, StatusCanceled, StatusRefunded, StatusChargedBack,
		}
		db = db.Where("status IN ?", finalStatuses)
	}

	if query.NonFinalOnly {
		finalStatuses := []PaymentStatus{
			StatusSucceeded, StatusPaid, StatusFailed, StatusDeclined,
			StatusExpired, StatusCanceled, StatusRefunded, StatusChargedBack,
		}
		db = db.Where("status NOT IN ?", finalStatuses)
	}

	// Pagination
	if query.PageSize > 0 {
		offset := 0
		if query.Page > 0 {
			offset = (query.Page - 1) * query.PageSize
		}
		db = db.Limit(query.PageSize).Offset(offset)
	}

	var statuses []PaymentStatusRecord
	err := db.Order("updated_at DESC").Find(&statuses).Error
	if err != nil {
		return nil, fmt.Errorf("failed to get payment statuses: %w", err)
	}

	return statuses, nil
}

// UpdateStatus updates the status of a payment with history tracking
func (s *StatusTrackerService) UpdateStatus(ctx context.Context, transition *StatusTransition) error {
	// Validate transition
	if err := transition.Validate(); err != nil {
		return fmt.Errorf("invalid transition: %w", err)
	}

	// Start transaction
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Get current status with lock
		var currentStatus PaymentStatusRecord
		err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("payment_id = ?", transition.PaymentID).
			First(&currentStatus).Error

		if err != nil {
			if err == gorm.ErrRecordNotFound {
				// Create new status record
				currentStatus = PaymentStatusRecord{
					PaymentID: transition.PaymentID,
					Status:    transition.ToStatus,
					UpdatedBy: transition.UpdatedBy,
					Metadata:  transition.Metadata,
					Version:   1,
				}
				if err := tx.Create(&currentStatus).Error; err != nil {
					return fmt.Errorf("failed to create status record: %w", err)
				}
			} else {
				return fmt.Errorf("failed to get current status: %w", err)
			}
		} else {
			// Validate transition if not forced
			if !transition.Force {
				if !currentStatus.Status.CanTransitionTo(transition.ToStatus) {
					return fmt.Errorf("invalid transition from %s to %s", currentStatus.Status, transition.ToStatus)
				}
			}

			// Update status record
			previousStatus := currentStatus.Status
			currentStatus.PreviousStatus = &previousStatus
			currentStatus.Status = transition.ToStatus
			currentStatus.StatusCategory = transition.ToStatus.GetCategory()
			currentStatus.Metadata = transition.Metadata
			currentStatus.UpdatedBy = transition.UpdatedBy

			if err := tx.Save(&currentStatus).Error; err != nil {
				return fmt.Errorf("failed to update status: %w", err)
			}
		}

		// Create history record
		history := PaymentStatusHistory{
			PaymentID:     transition.PaymentID,
			FromStatus:    currentStatus.PreviousStatus,
			ToStatus:      transition.ToStatus,
			Reason:        transition.Reason,
			Metadata:      transition.Metadata,
			UpdatedBy:     transition.UpdatedBy,
			UpdatedByType: "system",
		}

		if err := tx.Create(&history).Error; err != nil {
			return fmt.Errorf("failed to create history record: %w", err)
		}

		// Create audit log
		audit := StatusAuditLog{
			PaymentID:       transition.PaymentID,
			Action:          "status_change",
			FromStatus:      currentStatus.PreviousStatus,
			ToStatus:        transition.ToStatus,
			PerformedBy:     transition.UpdatedBy,
			PerformedByType: "system",
			Reason:          transition.Reason,
			Success:         true,
			Metadata:        transition.Metadata,
		}

		if err := tx.Create(&audit).Error; err != nil {
			// Log but don't fail the transaction
			fmt.Printf("Warning: failed to create audit log: %v\n", err)
		}

		// Queue notifications if requested
		if s.notifier != nil && (transition.NotifyUser || transition.NotifyAdmin) {
			// This will be handled asynchronously
			go s.handleNotifications(context.Background(), &currentStatus, &history, transition)
		}

		// Invalidate cache
		s.mu.Lock()
		delete(s.cache, transition.PaymentID)
		s.mu.Unlock()

		return nil
	})
}

// BulkUpdateStatuses updates multiple payment statuses
func (s *StatusTrackerService) BulkUpdateStatuses(ctx context.Context, transitions []*StatusTransition) error {
	errors := make([]error, 0)
	successCount := 0

	for _, transition := range transitions {
		if err := s.UpdateStatus(ctx, transition); err != nil {
			errors = append(errors, fmt.Errorf("payment %s: %w", transition.PaymentID, err))
		} else {
			successCount++
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("bulk update completed with %d successes and %d failures: %v", successCount, len(errors), errors)
	}

	return nil
}

// GetHistory retrieves the full history of status changes for a payment
func (s *StatusTrackerService) GetHistory(ctx context.Context, paymentID uuid.UUID) ([]PaymentStatusHistory, error) {
	var history []PaymentStatusHistory
	err := s.db.WithContext(ctx).
		Where("payment_id = ?", paymentID).
		Order("created_at ASC").
		Find(&history).Error

	if err != nil {
		return nil, fmt.Errorf("failed to get status history: %w", err)
	}

	// Calculate transition times
	for i := range history {
		if i > 0 {
			history[i].TransitionTime = history[i].CreatedAt.Sub(history[i-1].CreatedAt)
		}
	}

	return history, nil
}

// GetHistoryWithQuery retrieves status history with filtering
func (s *StatusTrackerService) GetHistoryWithQuery(ctx context.Context, query *StatusHistoryQuery) (*StatusHistoryResponse, error) {
	db := s.db.WithContext(ctx).Model(&PaymentStatusHistory{})

	// Apply filters
	if len(query.PaymentIDs) > 0 {
		db = db.Where("payment_id IN ?", query.PaymentIDs)
	}

	if len(query.FromStatuses) > 0 {
		db = db.Where("from_status IN ?", query.FromStatuses)
	}

	if len(query.ToStatuses) > 0 {
		db = db.Where("to_status IN ?", query.ToStatuses)
	}

	if len(query.Categories) > 0 {
		db = db.Where("to_category IN ?", query.Categories)
	}

	if query.UpdatedBy != nil {
		db = db.Where("updated_by = ?", *query.UpdatedBy)
	}

	if query.UpdatedByType != "" {
		db = db.Where("updated_by_type = ?", query.UpdatedByType)
	}

	if query.StartDate != nil {
		db = db.Where("created_at >= ?", *query.StartDate)
	}

	if query.EndDate != nil {
		db = db.Where("created_at <= ?", *query.EndDate)
	}

	// Get total count
	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, fmt.Errorf("failed to count history: %w", err)
	}

	// Apply sorting (with SQL injection protection)
	orderClause := utils.SafeOrderClause(
		query.OrderBy,
		query.OrderDir,
		utils.PaymentHistoryAllowedSortColumns,
		"created_at",
	)
	db = db.Order(orderClause)

	// Apply pagination
	if query.Limit > 0 {
		db = db.Limit(query.Limit)
	}
	if query.Offset > 0 {
		db = db.Offset(query.Offset)
	}

	var history []PaymentStatusHistory
	if err := db.Find(&history).Error; err != nil {
		return nil, fmt.Errorf("failed to get history: %w", err)
	}

	pageSize := query.Limit
	if pageSize == 0 {
		pageSize = int(total)
	}

	page := 1
	if query.Offset > 0 && pageSize > 0 {
		page = (query.Offset / pageSize) + 1
	}

	totalPages := int(total) / pageSize
	if int(total)%pageSize > 0 {
		totalPages++
	}

	return &StatusHistoryResponse{
		History:    history,
		Total:      total,
		Page:       page,
		PageSize:   pageSize,
		TotalPages: totalPages,
	}, nil
}

// GetTimeline creates a timeline visualization of status changes
func (s *StatusTrackerService) GetTimeline(ctx context.Context, paymentID uuid.UUID) (*StatusTimeline, error) {
	history, err := s.GetHistory(ctx, paymentID)
	if err != nil {
		return nil, err
	}

	if len(history) == 0 {
		return nil, fmt.Errorf("no history found for payment %s", paymentID)
	}

	// Get current status
	currentStatus, err := s.GetStatus(ctx, paymentID)
	if err != nil {
		return nil, err
	}

	timeline := &StatusTimeline{
		PaymentID:        paymentID,
		CurrentStatus:    currentStatus.Status,
		TotalTransitions: len(history),
		Timeline:         make([]StatusTimelineEntry, 0, len(history)),
		KeyMilestones:    make([]StatusMilestone, 0),
	}

	// Build timeline entries
	for i, h := range history {
		duration := time.Duration(0)
		if i < len(history)-1 {
			duration = history[i+1].CreatedAt.Sub(h.CreatedAt)
		} else {
			duration = time.Since(h.CreatedAt)
		}

		entry := StatusTimelineEntry{
			Status:        h.ToStatus,
			Category:      h.ToCategory,
			Timestamp:     h.CreatedAt,
			Duration:      duration,
			Reason:        h.Reason,
			UpdatedBy:     h.UpdatedBy,
			UpdatedByType: h.UpdatedByType,
			Metadata:      h.Metadata,
		}
		timeline.Timeline = append(timeline.Timeline, entry)

		// Add key milestones
		if s.isKeyMilestone(h.ToStatus) {
			milestone := StatusMilestone{
				Name:        s.getMilestoneName(h.ToStatus),
				Status:      h.ToStatus,
				Timestamp:   h.CreatedAt,
				Description: h.Reason,
			}
			timeline.KeyMilestones = append(timeline.KeyMilestones, milestone)
		}
	}

	// Calculate total duration
	if len(history) > 0 {
		timeline.Duration = time.Since(history[0].CreatedAt)
	}

	return timeline, nil
}

// GetStatistics calculates statistics about payment statuses
func (s *StatusTrackerService) GetStatistics(ctx context.Context) (*StatusStatistics, error) {
	var totalPayments int64
	if err := s.db.WithContext(ctx).Model(&PaymentStatusRecord{}).Count(&totalPayments).Error; err != nil {
		return nil, fmt.Errorf("failed to count payments: %w", err)
	}

	// Get status counts
	statusCounts := make(map[PaymentStatus]int64)
	var statusRows []struct {
		Status PaymentStatus
		Count  int64
	}

	err := s.db.WithContext(ctx).
		Model(&PaymentStatusRecord{}).
		Select("status, COUNT(*) as count").
		Group("status").
		Find(&statusRows).Error

	if err != nil {
		return nil, fmt.Errorf("failed to get status counts: %w", err)
	}

	for _, row := range statusRows {
		statusCounts[row.Status] = row.Count
	}

	// Get category counts
	categoryCounts := make(map[StatusCategory]int64)
	var categoryRows []struct {
		Category StatusCategory
		Count    int64
	}

	err = s.db.WithContext(ctx).
		Model(&PaymentStatusRecord{}).
		Select("status_category as category, COUNT(*) as count").
		Group("status_category").
		Find(&categoryRows).Error

	if err != nil {
		return nil, fmt.Errorf("failed to get category counts: %w", err)
	}

	for _, row := range categoryRows {
		categoryCounts[row.Category] = row.Count
	}

	// Calculate rates
	successCount := statusCounts[StatusSucceeded] + statusCounts[StatusPaid]
	failureCount := statusCounts[StatusFailed] + statusCounts[StatusDeclined]
	refundCount := statusCounts[StatusRefunded] + statusCounts[StatusPartiallyRefunded]
	disputeCount := statusCounts[StatusDisputed] + statusCounts[StatusChargedBack]

	stats := &StatusStatistics{
		TotalPayments:  totalPayments,
		StatusCounts:   statusCounts,
		CategoryCounts: categoryCounts,
		SuccessRate:    float64(successCount) / float64(totalPayments) * 100,
		FailureRate:    float64(failureCount) / float64(totalPayments) * 100,
		RefundRate:     float64(refundCount) / float64(totalPayments) * 100,
		DisputeRate:    float64(disputeCount) / float64(totalPayments) * 100,
		GeneratedAt:    time.Now(),
	}

	return stats, nil
}

// Helper methods
func (s *StatusTrackerService) isKeyMilestone(status PaymentStatus) bool {
	milestones := []PaymentStatus{
		StatusPaymentReceived,
		StatusAuthorized,
		StatusCaptured,
		StatusSucceeded,
		StatusPaid,
		StatusFailed,
		StatusRefunded,
	}

	for _, m := range milestones {
		if m == status {
			return true
		}
	}
	return false
}

func (s *StatusTrackerService) getMilestoneName(status PaymentStatus) string {
	names := map[PaymentStatus]string{
		StatusPaymentReceived: "Payment Received",
		StatusAuthorized:      "Payment Authorized",
		StatusCaptured:        "Payment Captured",
		StatusSucceeded:       "Payment Succeeded",
		StatusPaid:            "Payment Completed",
		StatusFailed:          "Payment Failed",
		StatusRefunded:        "Payment Refunded",
	}

	if name, exists := names[status]; exists {
		return name
	}
	return string(status)
}

func (s *StatusTrackerService) handleNotifications(ctx context.Context, status *PaymentStatusRecord, history *PaymentStatusHistory, transition *StatusTransition) {
	if s.notifier == nil {
		return
	}

	// Build status change string
	statusChange := fmt.Sprintf("%s->%s", "", transition.ToStatus)
	if history.FromStatus != nil {
		statusChange = fmt.Sprintf("%s->%s", *history.FromStatus, transition.ToStatus)
	}

	// TODO: Implement notification logic based on notification rules
	// This is a placeholder for the notification handling
	_ = statusChange // Suppress unused variable warning until notification logic is implemented
}

// GenerateDeduplicationHash generates a hash for deduplication
func GenerateDeduplicationHash(paymentID, recipientID uuid.UUID, notificationType NotificationType, statusChange string) string {
	data := fmt.Sprintf("%s:%s:%s:%s", paymentID, recipientID, notificationType, statusChange)
	hash := sha256.Sum256([]byte(data))
	return hex.EncodeToString(hash[:])
}

// Implement remaining interface methods as stubs (these would be implemented based on specific requirements)
func (s *StatusTrackerService) GetTransitionMetrics(ctx context.Context) (*StatusTransitionMatrix, error) {
	// TODO: Implement transition metrics calculation
	return &StatusTransitionMatrix{
		Transitions: make(map[string]map[string]*StatusTransitionMetrics),
		GeneratedAt: time.Now(),
	}, nil
}

func (s *StatusTrackerService) DetectAnomalies(ctx context.Context) ([]StatusAnomalyDetection, error) {
	// TODO: Implement anomaly detection
	return []StatusAnomalyDetection{}, nil
}

func (s *StatusTrackerService) GetHealthStatus(ctx context.Context) (*StatusHealth, error) {
	// TODO: Implement health status check
	return &StatusHealth{
		IsHealthy:          true,
		LastHealthCheck:    time.Now(),
		RecommendedActions: []string{},
	}, nil
}

func (s *StatusTrackerService) GetAuditLog(ctx context.Context, query *StatusAuditQuery) ([]StatusAuditLog, error) {
	db := s.db.WithContext(ctx).Model(&StatusAuditLog{})

	if len(query.PaymentIDs) > 0 {
		db = db.Where("payment_id IN ?", query.PaymentIDs)
	}

	if len(query.Actions) > 0 {
		db = db.Where("action IN ?", query.Actions)
	}

	if query.PerformedBy != nil {
		db = db.Where("performed_by = ?", *query.PerformedBy)
	}

	if query.PerformedByType != "" {
		db = db.Where("performed_by_type = ?", query.PerformedByType)
	}

	if query.StartDate != nil {
		db = db.Where("created_at >= ?", *query.StartDate)
	}

	if query.EndDate != nil {
		db = db.Where("created_at <= ?", *query.EndDate)
	}

	if query.SuccessOnly {
		db = db.Where("success = ?", true)
	}

	if query.FailuresOnly {
		db = db.Where("success = ?", false)
	}

	if query.Limit > 0 {
		db = db.Limit(query.Limit)
	}

	if query.Offset > 0 {
		db = db.Offset(query.Offset)
	}

	var logs []StatusAuditLog
	err := db.Order("created_at DESC").Find(&logs).Error
	if err != nil {
		return nil, fmt.Errorf("failed to get audit logs: %w", err)
	}

	return logs, nil
}

func (s *StatusTrackerService) CreateAuditEntry(ctx context.Context, entry *StatusAuditLog) error {
	return s.db.WithContext(ctx).Create(entry).Error
}
