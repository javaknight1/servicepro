package services

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"

	"github.com/javaknight1/servicepro/backend/internal/models"
	"github.com/javaknight1/servicepro/backend/internal/repository"
	"github.com/javaknight1/servicepro/backend/pkg/clients/logging"
	csvpkg "github.com/javaknight1/servicepro/backend/pkg/csv"
)

// ImportService handles customer import operations
type ImportService struct {
	db           *gorm.DB
	customerRepo repository.CustomerRepositoryInterface
	redisClient  *redis.Client
}

// NewImportService creates a new import service
func NewImportService(
	db *gorm.DB,
	customerRepo repository.CustomerRepositoryInterface,
	redisClient *redis.Client,
) *ImportService {
	return &ImportService{
		db:           db,
		customerRepo: customerRepo,
		redisClient:  redisClient,
	}
}

// CreateImportJob creates a new import job
func (s *ImportService) CreateImportJob(
	ctx context.Context,
	userID uuid.UUID,
	filename string,
	totalRows int,
) (*models.ImportJob, error) {
	job := &models.ImportJob{
		ID:        uuid.New(),
		UserID:    userID,
		Filename:  filename,
		Status:    models.ImportJobStatusPending,
		TotalRows: totalRows,
	}

	if err := s.db.Create(job).Error; err != nil {
		return nil, fmt.Errorf("failed to create import job: %w", err)
	}

	return job, nil
}

// GetImportJob retrieves an import job by ID
func (s *ImportService) GetImportJob(ctx context.Context, jobID uuid.UUID) (*models.ImportJob, error) {
	var job models.ImportJob
	if err := s.db.Where("id = ?", jobID).First(&job).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("import job not found")
		}
		return nil, fmt.Errorf("failed to get import job: %w", err)
	}
	return &job, nil
}

// GetImportErrors retrieves errors for an import job
func (s *ImportService) GetImportErrors(ctx context.Context, jobID uuid.UUID, limit int) ([]models.ImportError, error) {
	var errors []models.ImportError
	query := s.db.Where("job_id = ?", jobID).Order("row_number ASC")

	if limit > 0 {
		query = query.Limit(limit)
	}

	if err := query.Find(&errors).Error; err != nil {
		return nil, fmt.Errorf("failed to get import errors: %w", err)
	}

	return errors, nil
}

// UpdateJobStatus updates the status of an import job
func (s *ImportService) UpdateJobStatus(
	ctx context.Context,
	jobID uuid.UUID,
	status models.ImportJobStatus,
	errorSummary *string,
) error {
	updates := map[string]interface{}{
		"status": status,
	}

	if status == models.ImportJobStatusProcessing {
		now := time.Now()
		updates["started_at"] = &now
	}

	if status == models.ImportJobStatusCompleted || status == models.ImportJobStatusFailed {
		now := time.Now()
		updates["completed_at"] = &now
	}

	if errorSummary != nil {
		updates["error_summary"] = errorSummary
	}

	return s.db.Model(&models.ImportJob{}).Where("id = ?", jobID).Updates(updates).Error
}

// UpdateJobProgress updates the progress of an import job
func (s *ImportService) UpdateJobProgress(
	ctx context.Context,
	jobID uuid.UUID,
	processedRows, successfulRows, failedRows int,
) error {
	updates := map[string]interface{}{
		"processed_rows":  processedRows,
		"successful_rows": successfulRows,
		"failed_rows":     failedRows,
	}

	return s.db.Model(&models.ImportJob{}).Where("id = ?", jobID).Updates(updates).Error
}

// ProcessImport processes an import job
func (s *ImportService) ProcessImport(
	ctx context.Context,
	jobID uuid.UUID,
	rows []models.CSVImportRow,
	duplicateHandling string,
	dryRun bool,
) error {
	// Update job status to processing
	if err := s.UpdateJobStatus(ctx, jobID, models.ImportJobStatusProcessing, nil); err != nil {
		return err
	}

	var processedRows, successfulRows, failedRows int
	var importErrors []models.ImportError

	for i, row := range rows {
		// Validate additional business rules
		validationErrors := csvpkg.ValidateCustomerData(&row)
		if len(validationErrors) > 0 {
			for _, errMsg := range validationErrors {
				importErrors = append(importErrors, models.ImportError{
					JobID:     jobID,
					RowNumber: i + 1,
					Error:     errMsg,
				})
			}
			failedRows++
			processedRows++
			continue
		}

		// Check for duplicates
		existingCustomer, err := s.customerRepo.GetByEmail(row.Email)
		if err == nil && existingCustomer != nil {
			// Email already exists
			switch duplicateHandling {
			case "skip":
				successfulRows++ // Count as success (skipped)
			case "update":
				if !dryRun {
					// Update existing customer
					customer := row.ToCustomer()
					customer.ID = existingCustomer.ID
					if err := s.customerRepo.Update(customer); err != nil {
						importErrors = append(importErrors, models.ImportError{
							JobID:     jobID,
							RowNumber: i + 1,
							Field:     "email",
							Error:     fmt.Sprintf("failed to update: %v", err),
						})
						failedRows++
					} else {
						successfulRows++
					}
				} else {
					successfulRows++ // Dry run
				}
			case "error":
				importErrors = append(importErrors, models.ImportError{
					JobID:     jobID,
					RowNumber: i + 1,
					Field:     "email",
					Error:     "duplicate email address",
				})
				failedRows++
			default:
				importErrors = append(importErrors, models.ImportError{
					JobID:     jobID,
					RowNumber: i + 1,
					Field:     "email",
					Error:     "duplicate email address",
				})
				failedRows++
			}
		} else {
			// Create new customer
			if !dryRun {
				customer := row.ToCustomer()
				if err := s.customerRepo.Create(customer); err != nil {
					importErrors = append(importErrors, models.ImportError{
						JobID:     jobID,
						RowNumber: i + 1,
						Error:     fmt.Sprintf("failed to create: %v", err),
					})
					failedRows++
				} else {
					successfulRows++
				}
			} else {
				successfulRows++ // Dry run
			}
		}

		processedRows++

		// Update progress every 10 rows
		if processedRows%10 == 0 {
			if err := s.UpdateJobProgress(ctx, jobID, processedRows, successfulRows, failedRows); err != nil {
				// Log error but continue processing
				logging.Error(ctx, "[IMPORT] Failed to update progress", map[string]any{"error": err})
			}
		}
	}

	// Save all import errors
	if len(importErrors) > 0 {
		if err := s.db.Create(&importErrors).Error; err != nil {
			logging.Error(ctx, "[IMPORT] Failed to save import errors", map[string]any{"error": err})
		}
	}

	// Final progress update
	if err := s.UpdateJobProgress(ctx, jobID, processedRows, successfulRows, failedRows); err != nil {
		logging.Error(ctx, "[IMPORT] Failed to update final progress", map[string]any{"error": err})
	}

	// Update job status
	var errorSummary *string
	if failedRows > 0 {
		summary := fmt.Sprintf("%d rows failed to import", failedRows)
		errorSummary = &summary
	}

	status := models.ImportJobStatusCompleted
	if failedRows == len(rows) {
		status = models.ImportJobStatusFailed
	}

	if err := s.UpdateJobStatus(ctx, jobID, status, errorSummary); err != nil {
		return err
	}

	return nil
}

// EnqueueImportJob enqueues an import job for background processing
func (s *ImportService) EnqueueImportJob(
	ctx context.Context,
	jobID uuid.UUID,
	rows []models.CSVImportRow,
	duplicateHandling string,
	dryRun bool,
) error {
	// Serialize the job data
	jobData := map[string]interface{}{
		"job_id":             jobID.String(),
		"rows":               rows,
		"duplicate_handling": duplicateHandling,
		"dry_run":            dryRun,
	}

	data, err := json.Marshal(jobData)
	if err != nil {
		return fmt.Errorf("failed to marshal job data: %w", err)
	}

	// Add to Redis queue
	queueKey := "import:queue"
	if err := s.redisClient.RPush(ctx, queueKey, data).Err(); err != nil {
		return fmt.Errorf("failed to enqueue job: %w", err)
	}

	return nil
}

// ProcessNextJob processes the next job from the queue
func (s *ImportService) ProcessNextJob(ctx context.Context) error {
	queueKey := "import:queue"

	// Pop from queue
	result, err := s.redisClient.BLPop(ctx, 5*time.Second, queueKey).Result()
	if err != nil {
		if err == redis.Nil {
			// No jobs in queue
			return nil
		}
		return fmt.Errorf("failed to dequeue job: %w", err)
	}

	if len(result) < 2 {
		return nil
	}

	// Deserialize job data
	var jobData map[string]interface{}
	if err := json.Unmarshal([]byte(result[1]), &jobData); err != nil {
		return fmt.Errorf("failed to unmarshal job data: %w", err)
	}

	// Extract job parameters
	jobIDStr, ok := jobData["job_id"].(string)
	if !ok {
		return fmt.Errorf("invalid job_id")
	}

	jobID, err := uuid.Parse(jobIDStr)
	if err != nil {
		return fmt.Errorf("invalid job_id format: %w", err)
	}

	duplicateHandling, _ := jobData["duplicate_handling"].(string)
	dryRun, _ := jobData["dry_run"].(bool)

	// Deserialize rows
	rowsData, _ := jobData["rows"].([]interface{})
	rows := make([]models.CSVImportRow, len(rowsData))
	for i, rowInterface := range rowsData {
		rowBytes, _ := json.Marshal(rowInterface)
		json.Unmarshal(rowBytes, &rows[i])
	}

	// Process the import
	return s.ProcessImport(ctx, jobID, rows, duplicateHandling, dryRun)
}
