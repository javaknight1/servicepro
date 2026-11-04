package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/javaknight1/servicepro/backend/internal/imports"
	"github.com/javaknight1/servicepro/backend/internal/models"
	"github.com/javaknight1/servicepro/backend/internal/repository"
	"github.com/javaknight1/servicepro/backend/internal/tenancy"
	"github.com/javaknight1/servicepro/backend/pkg/clients/logging"
	csvpkg "github.com/javaknight1/servicepro/backend/pkg/csv"
)

// Import limits (finding F-23; reasoning in internal/imports). The upload
// handlers enforce the byte and row caps.
const (
	ImportMaxUploadBytes = imports.MaxFileBytes
	ImportMaxRows        = imports.MaxRows
	// ImportInlineMaxRows: customer-template files up to this size are run
	// during the upload request (still time-boxed; see ImportInlineBudget).
	ImportInlineMaxRows = 1000
	// ImportInlineBudget bounds the import work done inside an API request:
	// Cloud Run ends requests after 60 s. What is left stays pending for the
	// next chunk (the import page asks for one, or the scheduled job runs it).
	ImportInlineBudget = 20 * time.Second

	// importBatchRows: progress (the row cursor, counts and that batch's
	// row errors) is saved every this many rows, in one transaction.
	importBatchRows = 25
	// importStaleAfter: a processing job with no progress for this long was
	// interrupted (instance stopped mid-import). A mapped import resumes from
	// its cursor (its rows are idempotent) up to importMaxAttempts times; a
	// customer-template import fails: re-running it could double rows.
	importStaleAfter  = 5 * time.Minute
	importMaxAttempts = 3
	// importInterruptedMessage is shown to the user on an interrupted job.
	importInterruptedMessage = "Import was interrupted before it finished; check your customers and upload the remaining rows again"
	importGaveUpMessage      = "Import was interrupted several times; the rows imported so far are kept — upload the file again to finish"

	// importPayloadMapped marks a mapped (T193) payload.
	importPayloadMapped = 2
	// importSourceTemplate is import_jobs.source of a customer-template job.
	importSourceTemplate = "servicepro"
)

// importPayload is what a queued job carries in import_jobs.payload: the
// customer-template rows (Version 0) or mapped records (importPayloadMapped).
type importPayload struct {
	Version           int                   `json:"version,omitempty"`
	DuplicateHandling string                `json:"duplicate_handling,omitempty"`
	DryRun            bool                  `json:"dry_run,omitempty"`
	Rows              []models.CSVImportRow `json:"rows,omitempty"`
	Records           []imports.Record      `json:"records,omitempty"`
}

func (p *importPayload) total() int {
	if p.Version == importPayloadMapped {
		return len(p.Records)
	}
	return len(p.Rows)
}

// ImportRunResult summarizes one POST /internal/jobs/imports pass.
type ImportRunResult struct {
	Claimed     int   `json:"claimed" example:"2"`
	Completed   int   `json:"completed" example:"2"`
	Failed      int   `json:"failed" example:"0"`
	Interrupted int64 `json:"interrupted" example:"0"`
	Remaining   bool  `json:"remaining" example:"false"`
}

// ImportService handles import operations.
//
// Lifecycle: CreateImportJob / CreateMappedImportJob store the parsed rows
// as a pending job; a runner claims it atomically (pending → processing, one
// winner) and imports rows under the job's own tenant in time-boxed chunks.
// After each batch it saves the cursor (processed_rows), the counts and the
// batch's row errors; when its time is up it puts the job back to pending
// for the next chunk. The last chunk marks it completed/failed and clears
// the payload.
type ImportService struct {
	db           *gorm.DB
	customerRepo repository.CustomerRepositoryInterface
	now          func() time.Time
}

// NewImportService creates a new import service
func NewImportService(db *gorm.DB, customerRepo repository.CustomerRepositoryInterface) *ImportService {
	return &ImportService{db: db, customerRepo: customerRepo, now: time.Now}
}

// CreateImportJob stores a pending customer-template import job for the
// caller's tenant (taken from ctx by the tenancy plugin) with its rows and
// options.
func (s *ImportService) CreateImportJob(
	ctx context.Context,
	userID uuid.UUID,
	filename string,
	rows []models.CSVImportRow,
	duplicateHandling string,
	dryRun bool,
) (*models.ImportJob, error) {
	return s.createJob(ctx, userID, filename, importSourceTemplate, string(imports.EntityCustomers),
		importPayload{DuplicateHandling: duplicateHandling, DryRun: dryRun, Rows: rows}, len(rows))
}

// CreateMappedImportJob stores a pending mapped import (T193) of records
// read from a Jobber / Housecall Pro / generic file for the caller's tenant.
func (s *ImportService) CreateMappedImportJob(
	ctx context.Context,
	userID uuid.UUID,
	filename string,
	src imports.Source,
	entity imports.Entity,
	records []imports.Record,
) (*models.ImportJob, error) {
	return s.createJob(ctx, userID, filename, string(src), string(entity),
		importPayload{Version: importPayloadMapped, Records: records}, len(records))
}

func (s *ImportService) createJob(ctx context.Context, userID uuid.UUID, filename, source, entity string, p importPayload, total int) (*models.ImportJob, error) {
	data, err := json.Marshal(p)
	if err != nil {
		return nil, fmt.Errorf("marshal import payload: %w", err)
	}
	payload := string(data)
	if len(filename) > 255 {
		filename = filename[:255]
	}
	job := &models.ImportJob{
		ID:        uuid.New(),
		UserID:    userID,
		Filename:  filename,
		Status:    models.ImportJobStatusPending,
		TotalRows: total,
		Source:    source,
		Entity:    entity,
		Payload:   &payload,
	}
	if err := s.db.WithContext(ctx).Create(job).Error; err != nil {
		return nil, fmt.Errorf("failed to create import job: %w", err)
	}
	return job, nil
}

// ErrImportJobNotFound: no such import job in the caller's tenant.
var ErrImportJobNotFound = errors.New("import job not found")

// GetImportJob retrieves an import job by ID (tenant-scoped: another
// tenant's job is not found).
func (s *ImportService) GetImportJob(ctx context.Context, jobID uuid.UUID) (*models.ImportJob, error) {
	tenantID, err := tenancy.MustTenantID(ctx)
	if err != nil {
		return nil, err
	}
	var job models.ImportJob
	if err := s.db.WithContext(ctx).Omit("payload").Where("id = ? AND tenant_id = ?", jobID, tenantID).First(&job).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrImportJobNotFound
		}
		return nil, fmt.Errorf("failed to get import job: %w", err)
	}
	return &job, nil
}

// ListImportJobs returns the caller's tenant's most recent import jobs of
// the given entities (all when none), newest first.
func (s *ImportService) ListImportJobs(ctx context.Context, entities []string, limit int) ([]models.ImportJob, error) {
	tenantID, err := tenancy.MustTenantID(ctx)
	if err != nil {
		return nil, err
	}
	q := s.db.WithContext(ctx).Omit("payload").Where("tenant_id = ?", tenantID).Order("created_at DESC").Limit(limit)
	if len(entities) > 0 {
		q = q.Where("entity IN ?", entities)
	}
	var jobs []models.ImportJob
	if err := q.Find(&jobs).Error; err != nil {
		return nil, fmt.Errorf("list import jobs: %w", err)
	}
	return jobs, nil
}

// GetImportErrors retrieves errors for an import job
func (s *ImportService) GetImportErrors(ctx context.Context, jobID uuid.UUID, limit int) ([]models.ImportError, error) {
	// import_errors has no tenant_id: verify the (tenant-scoped) parent job first.
	if _, err := s.GetImportJob(ctx, jobID); err != nil {
		return nil, err
	}
	var errs []models.ImportError
	query := s.db.WithContext(ctx).Where("job_id = ?", jobID).Order("row_number ASC, created_at ASC")
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
	q := `UPDATE import_jobs SET status = 'processing', started_at = COALESCE(started_at, ?), updated_at = ? WHERE ` + where + ` RETURNING *`
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

// failStale handles jobs stuck in processing (no progress for
// importStaleAfter): a mapped import goes back to pending to resume from its
// cursor (until importMaxAttempts); anything else fails.
func (s *ImportService) failStale(ctx context.Context) (int64, error) {
	now := s.now()
	cutoff := now.Add(-importStaleAfter)
	resumed := s.db.WithContext(ctx).Exec(`UPDATE import_jobs
		SET status = 'pending', attempts = attempts + 1, updated_at = ?
		WHERE status = 'processing' AND updated_at < ? AND source <> ? AND attempts + 1 < ?`,
		now, cutoff, importSourceTemplate, importMaxAttempts)
	if resumed.Error != nil {
		return 0, fmt.Errorf("resume stale import jobs: %w", resumed.Error)
	}
	failed := s.db.WithContext(ctx).Exec(`UPDATE import_jobs
		SET status = 'failed', error_summary = CASE WHEN source = ? THEN ? ELSE ? END,
		    completed_at = ?, updated_at = ?, payload = NULL
		WHERE status = 'processing' AND updated_at < ?`,
		importSourceTemplate, importInterruptedMessage, importGaveUpMessage, now, now, cutoff)
	if failed.Error != nil {
		return 0, fmt.Errorf("fail stale import jobs: %w", failed.Error)
	}
	return resumed.RowsAffected + failed.RowsAffected, nil
}

// ProcessPending is one POST /internal/jobs/imports pass: it recovers
// interrupted jobs, then claims and runs pending jobs one at a time until
// none are left, maxJobs ran, or budget elapsed (a job cut off by the budget
// goes back to pending). Safe to run concurrently.
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
		finished, err := s.RunImportChunk(ctx, job, max(deadline.Sub(s.now()), time.Nanosecond))
		if err != nil {
			return result, err
		}
		switch finished.Status {
		case models.ImportJobStatusFailed:
			result.Failed++
		case models.ImportJobStatusCompleted:
			result.Completed++
		default:
			result.Remaining = true
		}
	}
}

// RunImportJob imports all of a claimed job's remaining rows (no time box)
// and returns the finished job.
func (s *ImportService) RunImportJob(ctx context.Context, job *models.ImportJob) (*models.ImportJob, error) {
	return s.RunImportChunk(ctx, job, 0)
}

// RunImportChunk imports a claimed job's rows from its cursor under the
// job's own tenant until they are done or budget (0 = no limit) is spent,
// and returns the job: finished (completed/failed), or back to pending with
// its progress saved. The tenant comes from the job row, never from the
// caller, so a runner can't import one tenant's rows into another.
func (s *ImportService) RunImportChunk(ctx context.Context, job *models.ImportJob, budget time.Duration) (*models.ImportJob, error) {
	if job.TenantID == uuid.Nil {
		return nil, fmt.Errorf("import job %s has no tenant", job.ID)
	}
	ctx = tenancy.WithTenant(ctx, job.TenantID)

	var payload importPayload
	if job.Payload == nil {
		return s.unreadable(ctx, job, errors.New("no payload"))
	}
	if err := json.Unmarshal([]byte(*job.Payload), &payload); err != nil {
		return s.unreadable(ctx, job, err)
	}
	run, err := s.rowRunner(ctx, job, &payload)
	if err != nil {
		return s.unreadable(ctx, job, err)
	}

	st := importStats{processed: job.ProcessedRows, successful: job.SuccessfulRows, failed: job.FailedRows, skipped: job.SkippedRows}
	total := payload.total()
	var deadline time.Time
	if budget > 0 {
		deadline = s.now().Add(budget)
	}
	for first := true; st.processed < total; first = false {
		// At least one batch per chunk, so every chunk makes progress.
		if !first && !deadline.IsZero() && !s.now().Before(deadline) {
			return s.release(ctx, job, st)
		}
		end := min(st.processed+importBatchRows, total)
		var rowErrors []models.ImportError
		for i := st.processed; i < end; i++ {
			out, errs := run(i)
			switch out {
			case rowImported:
				st.successful++
			case rowSkipped:
				st.skipped++
			default:
				st.failed++
			}
			for _, e := range errs {
				e.JobID = job.ID
				rowErrors = append(rowErrors, e)
			}
		}
		st.processed = end
		held, err := s.saveProgress(ctx, job.ID, st, rowErrors)
		if err != nil {
			return nil, err
		}
		if !held {
			// failStale took the job from this runner (it outlived
			// importStaleAfter); whoever holds it now owns its progress.
			logging.Warn(ctx, "[IMPORT-SERVICE] Job was no longer processing; stopping", map[string]any{"job_id": job.ID.String()})
			return job, nil
		}
	}

	var summary *string
	if st.failed > 0 {
		msg := fmt.Sprintf("%d rows failed to import", st.failed)
		if st.failed == 1 {
			msg = "1 row failed to import"
		}
		summary = &msg
	}
	status := models.ImportJobStatusCompleted
	if total > 0 && st.failed == total {
		status = models.ImportJobStatusFailed
	}
	return s.finish(ctx, job, st, summary, status)
}

func (s *ImportService) unreadable(ctx context.Context, job *models.ImportJob, err error) (*models.ImportJob, error) {
	logging.Error(ctx, "[IMPORT-SERVICE] Unreadable import payload", map[string]any{"job_id": job.ID.String(), "error": err.Error()})
	msg := "Import data could not be read; upload the file again"
	return s.finish(ctx, job, importStats{processed: job.ProcessedRows, successful: job.SuccessfulRows, failed: job.FailedRows, skipped: job.SkippedRows}, &msg, models.ImportJobStatusFailed)
}

type importStats struct{ processed, successful, failed, skipped int }

// rowOutcome is what happened to one row.
type rowOutcome int

const (
	rowFailed rowOutcome = iota
	rowImported
	rowSkipped
)

// rowRunner imports payload row i and returns its outcome and errors.
type rowRunner func(i int) (rowOutcome, []models.ImportError)

func (s *ImportService) rowRunner(ctx context.Context, job *models.ImportJob, p *importPayload) (rowRunner, error) {
	if p.Version == importPayloadMapped {
		m, err := s.newMappedImporter(ctx, job)
		if err != nil {
			return nil, err
		}
		return func(i int) (rowOutcome, []models.ImportError) { return m.importRecord(ctx, &p.Records[i]) }, nil
	}
	repo := s.customerRepo.WithContext(ctx)
	return func(i int) (rowOutcome, []models.ImportError) {
		var errs []models.ImportError
		row := &p.Rows[i]
		fail := func(rowNum int, row *models.CSVImportRow, field, msg string) {
			data, err := json.Marshal(row)
			if err != nil {
				data = nil
			}
			errs = append(errs, models.ImportError{RowNumber: rowNum, Field: field, Error: msg, RowData: string(data)})
		}
		if s.importRow(ctx, repo, row, p.DuplicateHandling, p.DryRun, i+1, fail) {
			return rowImported, nil
		}
		return rowFailed, errs
	}, nil
}

// saveProgress stores the cursor, counts and a batch's row errors in one
// transaction, only while this runner still holds the job (status
// processing). It reports whether it did.
func (s *ImportService) saveProgress(ctx context.Context, jobID uuid.UUID, st importStats, rowErrors []models.ImportError) (bool, error) {
	held := false
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		res := tx.Model(&models.ImportJob{}).Where("id = ? AND status = ?", jobID, models.ImportJobStatusProcessing).
			Updates(map[string]any{
				"processed_rows": st.processed, "successful_rows": st.successful,
				"failed_rows": st.failed, "skipped_rows": st.skipped,
			})
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			return nil
		}
		held = true
		if len(rowErrors) > 0 {
			return tx.CreateInBatches(&rowErrors, 500).Error
		}
		return nil
	})
	if err != nil {
		return false, fmt.Errorf("save import progress: %w", err)
	}
	return held, nil
}

// release puts a job whose chunk ran out of time back to pending (progress
// is already saved), for the next chunk.
func (s *ImportService) release(ctx context.Context, job *models.ImportJob, st importStats) (*models.ImportJob, error) {
	res := s.db.WithContext(ctx).Model(&models.ImportJob{}).
		Where("id = ? AND status = ?", job.ID, models.ImportJobStatusProcessing).
		Update("status", models.ImportJobStatusPending)
	if res.Error != nil {
		return nil, fmt.Errorf("release import job: %w", res.Error)
	}
	if res.RowsAffected > 0 {
		job.Status = models.ImportJobStatusPending
	}
	job.ProcessedRows, job.SuccessfulRows, job.FailedRows, job.SkippedRows = st.processed, st.successful, st.failed, st.skipped
	return job, nil
}

// finish records the final counts and status and clears the payload. It only
// touches a job this runner still holds (status = processing).
func (s *ImportService) finish(ctx context.Context, job *models.ImportJob, st importStats, summary *string, status models.ImportJobStatus) (*models.ImportJob, error) {
	now := s.now()
	updates := map[string]any{
		"status":          status,
		"processed_rows":  st.processed,
		"successful_rows": st.successful,
		"failed_rows":     st.failed,
		"skipped_rows":    st.skipped,
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
		// failStale already took it (this run outlived importStaleAfter).
		logging.Warn(ctx, "[IMPORT-SERVICE] Job was no longer processing at finish", map[string]any{"job_id": job.ID.String()})
		return job, nil
	}
	job.Status, job.ProcessedRows, job.SuccessfulRows, job.FailedRows, job.SkippedRows = status, st.processed, st.successful, st.failed, st.skipped
	job.ErrorSummary, job.CompletedAt, job.Payload = summary, &now, nil
	return job, nil
}

// importRow imports one customer-template row; it reports failures through
// fail and returns whether the row succeeded (a skipped duplicate counts as
// success).
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
