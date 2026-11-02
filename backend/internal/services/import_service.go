package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/javaknight1/servicepro/backend/internal/models"
	"github.com/javaknight1/servicepro/backend/internal/repository"
	"github.com/javaknight1/servicepro/backend/internal/tenancy"
	"github.com/javaknight1/servicepro/backend/pkg/clients/logging"
	csvpkg "github.com/javaknight1/servicepro/backend/pkg/csv"
)

// Import limits (finding F-23). The upload handler enforces the byte and row
// caps; files up to ImportInlineMaxRows are imported during the upload
// request, larger ones wait for POST /internal/jobs/imports.
const (
	ImportMaxUploadBytes int64 = 5 << 20 // 5 MiB
	ImportMaxRows              = 10000
	ImportInlineMaxRows        = 1000

	// importStaleAfter: a processing job with no progress for this long was
	// interrupted (instance stopped mid-import). It is failed, not retried:
	// re-running would create the already-imported rows twice.
	importStaleAfter = 15 * time.Minute
	// importInterruptedMessage is shown to the user on an interrupted job.
	importInterruptedMessage = "Import was interrupted before it finished; check your customers and upload the remaining rows again"
)

// importPayload is what a queued job carries in import_jobs.payload.
type importPayload struct {
	DuplicateHandling string                `json:"duplicate_handling"`
	DryRun            bool                  `json:"dry_run"`
	Rows              []models.CSVImportRow `json:"rows"`
}

// ImportRunResult summarizes one POST /internal/jobs/imports pass.
type ImportRunResult struct {
	Claimed     int   `json:"claimed" example:"2"`
	Completed   int   `json:"completed" example:"2"`
	Failed      int   `json:"failed" example:"0"`
	Interrupted int64 `json:"interrupted" example:"0"`
	Remaining   bool  `json:"remaining" example:"false"`
}

// ImportService handles customer import operations.
//
// Lifecycle: CreateImportJob stores the parsed rows as a pending job; a
// runner claims it atomically (pending → processing, one winner), imports the
// rows under the job's own tenant, records per-row failures in import_errors,
// then marks it completed/failed and clears the payload.
type ImportService struct {
	db           *gorm.DB
	customerRepo repository.CustomerRepositoryInterface
	now          func() time.Time
}

// NewImportService creates a new import service
func NewImportService(db *gorm.DB, customerRepo repository.CustomerRepositoryInterface) *ImportService {
	return &ImportService{db: db, customerRepo: customerRepo, now: time.Now}
}

// CreateImportJob stores a pending import job for the caller's tenant (taken
// from ctx by the tenancy plugin) with its rows and options.
func (s *ImportService) CreateImportJob(
	ctx context.Context,
	userID uuid.UUID,
	filename string,
	rows []models.CSVImportRow,
	duplicateHandling string,
	dryRun bool,
) (*models.ImportJob, error) {
	data, err := json.Marshal(importPayload{DuplicateHandling: duplicateHandling, DryRun: dryRun, Rows: rows})
	if err != nil {
		return nil, fmt.Errorf("marshal import payload: %w", err)
	}
	payload := string(data)
	job := &models.ImportJob{
		ID:        uuid.New(),
		UserID:    userID,
		Filename:  filename,
		Status:    models.ImportJobStatusPending,
		TotalRows: len(rows),
		Payload:   &payload,
	}
	if err := s.db.WithContext(ctx).Create(job).Error; err != nil {
		return nil, fmt.Errorf("failed to create import job: %w", err)
	}
	return job, nil
}

// GetImportJob retrieves an import job by ID (tenant-scoped: another
// tenant's job is not found).
func (s *ImportService) GetImportJob(ctx context.Context, jobID uuid.UUID) (*models.ImportJob, error) {
	var job models.ImportJob
	if err := s.db.WithContext(ctx).Where("id = ?", jobID).First(&job).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("import job not found")
		}
		return nil, fmt.Errorf("failed to get import job: %w", err)
	}
	return &job, nil
}

// GetImportErrors retrieves errors for an import job
func (s *ImportService) GetImportErrors(ctx context.Context, jobID uuid.UUID, limit int) ([]models.ImportError, error) {
	// import_errors has no tenant_id: verify the (tenant-scoped) parent job first.
	if _, err := s.GetImportJob(ctx, jobID); err != nil {
		return nil, err
	}
	var errs []models.ImportError
	query := s.db.WithContext(ctx).Where("job_id = ?", jobID).Order("row_number ASC")
	if limit > 0 {
		query = query.Limit(limit)
	}
	if err := query.Find(&errs).Error; err != nil {
		return nil, fmt.Errorf("failed to get import errors: %w", err)
	}
	return errs, nil
}

// claim runs an UPDATE … RETURNING that moves at most one pending job to
// processing. Raw SQL is not rewritten by the tenancy plugin, so the caller's
// WHERE decides the scope.
func (s *ImportService) claim(ctx context.Context, where string, args ...any) (*models.ImportJob, error) {
	now := s.now()
	var jobs []models.ImportJob
	q := `UPDATE import_jobs SET status = 'processing', started_at = ?, updated_at = ? WHERE ` + where + ` RETURNING *`
	if err := s.db.WithContext(ctx).Raw(q, append([]any{now, now}, args...)...).Scan(&jobs).Error; err != nil {
		return nil, fmt.Errorf("claim import job: %w", err)
	}
	if len(jobs) == 0 {
		return nil, nil
	}
	return &jobs[0], nil
}

// ClaimImportJob claims one pending job of the caller's tenant. It returns
// nil when the job is not pending any more (another runner claimed it) or
// belongs to another tenant.
func (s *ImportService) ClaimImportJob(ctx context.Context, jobID uuid.UUID) (*models.ImportJob, error) {
	tenantID, err := tenancy.MustTenantID(ctx)
	if err != nil {
		return nil, err
	}
	return s.claim(ctx, `id = ? AND tenant_id = ? AND status = 'pending'`, jobID, tenantID)
}

// claimNext claims the oldest pending job of any tenant. FOR UPDATE SKIP
// LOCKED lets concurrent runners each take a different job.
func (s *ImportService) claimNext(ctx context.Context) (*models.ImportJob, error) {
	return s.claim(ctx, `id = (SELECT id FROM import_jobs WHERE status = 'pending'
		ORDER BY created_at, id LIMIT 1 FOR UPDATE SKIP LOCKED) AND status = 'pending'`)
}

// failStale fails jobs stuck in processing (no progress for importStaleAfter).
func (s *ImportService) failStale(ctx context.Context) (int64, error) {
	now := s.now()
	res := s.db.WithContext(ctx).Exec(`UPDATE import_jobs
		SET status = 'failed', error_summary = ?, completed_at = ?, updated_at = ?, payload = NULL
		WHERE status = 'processing' AND updated_at < ?`,
		importInterruptedMessage, now, now, now.Add(-importStaleAfter))
	if res.Error != nil {
		return 0, fmt.Errorf("fail stale import jobs: %w", res.Error)
	}
	return res.RowsAffected, nil
}

// ProcessPending is one POST /internal/jobs/imports pass: it fails
// interrupted jobs, then claims and runs pending jobs one at a time until none
// are left, maxJobs ran, or budget elapsed. Safe to run concurrently.
func (s *ImportService) ProcessPending(ctx context.Context, maxJobs int, budget time.Duration) (ImportRunResult, error) {
	var result ImportRunResult
	sysCtx := tenancy.WithSystem(ctx)

	n, err := s.failStale(sysCtx)
	if err != nil {
		return result, err
	}
	result.Interrupted = n

	deadline := s.now().Add(budget)
	for {
		if result.Claimed >= maxJobs || !s.now().Before(deadline) {
			result.Remaining = true // maybe; the next tick picks it up
			return result, nil
		}
		job, err := s.claimNext(sysCtx)
		if err != nil {
			return result, err
		}
		if job == nil {
			return result, nil
		}
		result.Claimed++
		finished, err := s.RunImportJob(ctx, job)
		if err != nil {
			return result, err
		}
		if finished.Status == models.ImportJobStatusFailed {
			result.Failed++
		} else {
			result.Completed++
		}
	}
}

// RunImportJob imports a claimed job's rows under the job's own tenant and
// returns the finished job. The tenant comes from the job row, never from the
// caller, so a runner can't import one tenant's rows into another.
func (s *ImportService) RunImportJob(ctx context.Context, job *models.ImportJob) (*models.ImportJob, error) {
	if job.TenantID == uuid.Nil {
		return nil, fmt.Errorf("import job %s has no tenant", job.ID)
	}
	ctx = tenancy.WithTenant(ctx, job.TenantID)

	var payload importPayload
	if job.Payload == nil || json.Unmarshal([]byte(*job.Payload), &payload) != nil {
		logging.Error(ctx, "[IMPORT-SERVICE] Unreadable import payload", map[string]any{"job_id": job.ID.String()})
		msg := "Import data could not be read; upload the file again"
		return s.finish(ctx, job, importStats{}, &msg, models.ImportJobStatusFailed)
	}

	stats := s.importRows(ctx, job.ID, payload)

	var summary *string
	if stats.failed > 0 {
		msg := fmt.Sprintf("%d rows failed to import", stats.failed)
		summary = &msg
	}
	status := models.ImportJobStatusCompleted
	if len(payload.Rows) > 0 && stats.failed == len(payload.Rows) {
		status = models.ImportJobStatusFailed
	}
	return s.finish(ctx, job, stats, summary, status)
}

type importStats struct{ processed, successful, failed int }

// finish records the final counts and status and clears the payload. It only
// touches a job this runner still holds (status = processing).
func (s *ImportService) finish(ctx context.Context, job *models.ImportJob, st importStats, summary *string, status models.ImportJobStatus) (*models.ImportJob, error) {
	now := s.now()
	updates := map[string]any{
		"status":          status,
		"processed_rows":  st.processed,
		"successful_rows": st.successful,
		"failed_rows":     st.failed,
		"error_summary":   summary,
		"completed_at":    &now,
		"payload":         gorm.Expr("NULL"),
	}
	res := s.db.WithContext(ctx).Model(&models.ImportJob{}).
		Where("id = ? AND status = ?", job.ID, models.ImportJobStatusProcessing).
		Updates(updates)
	if res.Error != nil {
		return nil, fmt.Errorf("finish import job: %w", res.Error)
	}
	if res.RowsAffected == 0 {
		// failStale already failed it (this run outlived importStaleAfter).
		logging.Warn(ctx, "[IMPORT-SERVICE] Job was no longer processing at finish", map[string]any{"job_id": job.ID.String()})
	}
	job.Status, job.ProcessedRows, job.SuccessfulRows, job.FailedRows = status, st.processed, st.successful, st.failed
	job.ErrorSummary, job.CompletedAt, job.Payload = summary, &now, nil
	return job, nil
}

// updateProgress records running counts (and bumps updated_at, which keeps
// the job from looking stale).
func (s *ImportService) updateProgress(ctx context.Context, jobID uuid.UUID, st importStats) {
	err := s.db.WithContext(ctx).Model(&models.ImportJob{}).Where("id = ?", jobID).Updates(map[string]any{
		"processed_rows": st.processed, "successful_rows": st.successful, "failed_rows": st.failed,
	}).Error
	if err != nil {
		logging.Error(ctx, "[IMPORT-SERVICE] Failed to update progress", map[string]any{"error": err.Error(), "job_id": jobID.String()})
	}
}

// importRows creates/updates customers for each row (ctx carries the job's
// tenant) and records every failed row in import_errors.
func (s *ImportService) importRows(ctx context.Context, jobID uuid.UUID, p importPayload) importStats {
	var st importStats
	var rowErrors []models.ImportError
	repo := s.customerRepo.WithContext(ctx)

	fail := func(rowNum int, row *models.CSVImportRow, field, msg string) {
		data, _ := json.Marshal(row)
		rowErrors = append(rowErrors, models.ImportError{JobID: jobID, RowNumber: rowNum, Field: field, Error: msg, RowData: string(data)})
	}

	for i := range p.Rows {
		row := &p.Rows[i]
		rowNum := i + 1
		ok := s.importRow(ctx, repo, row, p.DuplicateHandling, p.DryRun, rowNum, fail)
		st.processed++
		if ok {
			st.successful++
		} else {
			st.failed++
		}
		if st.processed%50 == 0 {
			s.updateProgress(ctx, jobID, st)
		}
	}

	if len(rowErrors) > 0 {
		if err := s.db.WithContext(ctx).CreateInBatches(&rowErrors, 500).Error; err != nil {
			logging.Error(ctx, "[IMPORT-SERVICE] Failed to save import errors", map[string]any{"error": err.Error(), "job_id": jobID.String()})
		}
	}
	return st
}

// importRow imports one row; it reports failures through fail and returns
// whether the row succeeded (a skipped duplicate counts as success).
func (s *ImportService) importRow(
	ctx context.Context,
	repo repository.CustomerRepositoryInterface,
	row *models.CSVImportRow,
	duplicateHandling string,
	dryRun bool,
	rowNum int,
	fail func(int, *models.CSVImportRow, string, string),
) bool {
	if errs := csvpkg.ValidateCustomerData(row); len(errs) > 0 {
		for _, msg := range errs {
			fail(rowNum, row, "", msg)
		}
		return false
	}

	existing, err := repo.GetByEmail(row.Email)
	if err != nil && !errors.Is(err, repository.ErrCustomerNotFound) {
		logging.Error(ctx, "[IMPORT-SERVICE] Duplicate lookup failed", map[string]any{"error": err.Error(), "row": rowNum})
		fail(rowNum, row, "", "could not save this row")
		return false
	}

	if existing != nil {
		switch duplicateHandling {
		case "skip":
			return true
		case "update":
			if dryRun {
				return true
			}
			customer := row.ToCustomer()
			customer.ID = existing.ID
			if err := repo.Update(customer); err != nil {
				fail(rowNum, row, "email", rowSaveError(ctx, err, rowNum))
				return false
			}
			return true
		default: // "error"
			fail(rowNum, row, "email", "duplicate email address")
			return false
		}
	}

	if dryRun {
		return true
	}
	if err := repo.Create(row.ToCustomer()); err != nil {
		fail(rowNum, row, "", rowSaveError(ctx, err, rowNum))
		return false
	}
	return true
}

// rowSaveError is the user-safe message for a failed write (raw database
// errors are logged, never stored for the client).
func rowSaveError(ctx context.Context, err error, rowNum int) string {
	if errors.Is(err, repository.ErrEmailAlreadyExists) {
		return "duplicate email address"
	}
	logging.Error(ctx, "[IMPORT-SERVICE] Row save failed", map[string]any{"error": err.Error(), "row": rowNum})
	return "could not save this row"
}
