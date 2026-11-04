package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"path/filepath"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/javaknight1/servicepro/backend/internal/api/apierror"
	"github.com/javaknight1/servicepro/backend/internal/imports"
	"github.com/javaknight1/servicepro/backend/internal/models"
	"github.com/javaknight1/servicepro/backend/internal/services"
	"github.com/javaknight1/servicepro/backend/pkg/clients/logging"
)

// Preview and list sizes.
const (
	importPreviewRows   = 20
	importPreviewErrors = 100
	importStatusErrors  = 100
	importListLimit     = 20
	// importMaxMappingBytes caps the mapping form field (a JSON object of
	// field → header).
	importMaxMappingBytes = 16 << 10
)

// ImportsHandler serves mapped imports (T193): Jobber / Housecall Pro
// exports and any CSV or XLSX with a column mapping. Every route is
// tenant-scoped through the request context; the route's permission
// matches the entity (customers.create, customers.update for sites,
// jobs.create, invoices.create).
type ImportsHandler struct {
	imports *services.ImportService
	now     func() time.Time
}

// NewImportsHandler creates the imports handler.
func NewImportsHandler(svc *services.ImportService) *ImportsHandler {
	return &ImportsHandler{imports: svc, now: time.Now}
}

// importUpload is a parsed upload: the table and how it maps.
type importUpload struct {
	filename string
	source   imports.Source
	entity   imports.Entity
	table    *imports.Table
	prepared *imports.Prepared
}

// readUpload parses the multipart upload (file, source, optional mapping),
// enforcing the size and row caps (F-23). It writes the error response and
// returns nil on failure.
func (h *ImportsHandler) readUpload(c *gin.Context) *importUpload {
	entity, err := imports.ParseEntity(c.Param("entity"))
	if err != nil {
		apierror.WriteResponse(c, http.StatusNotFound, models.ErrorResponse{Error: "not_found", Message: err.Error()})
		return nil
	}
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, imports.MaxFileBytes+64<<10)
	if err := c.Request.ParseMultipartForm(1 << 20); err != nil {
		var tooBig *http.MaxBytesError
		if errors.As(err, &tooBig) {
			writeTooLarge(c, imports.ErrTooLarge)
			return nil
		}
		apierror.WriteResponse(c, http.StatusBadRequest, models.ErrorResponse{Error: "invalid_request", Message: "request must be multipart/form-data with a file"})
		return nil
	}
	defer func() {
		if c.Request.MultipartForm != nil {
			_ = c.Request.MultipartForm.RemoveAll()
		}
	}()

	src := imports.SourceGeneric
	if v := c.Request.FormValue("source"); v != "" {
		if src, err = imports.ParseSource(v); err != nil {
			apierror.WriteResponse(c, http.StatusBadRequest, models.ErrorResponse{Error: "invalid_source", Message: err.Error()})
			return nil
		}
	}
	var mapping map[string]string
	if raw := c.Request.FormValue("mapping"); raw != "" {
		if len(raw) > importMaxMappingBytes || json.Unmarshal([]byte(raw), &mapping) != nil {
			apierror.WriteResponse(c, http.StatusBadRequest, models.ErrorResponse{Error: "invalid_mapping", Message: "mapping must be a JSON object of field → column header"})
			return nil
		}
	}

	file, header, err := c.Request.FormFile("file")
	if err != nil {
		apierror.WriteResponse(c, http.StatusBadRequest, models.ErrorResponse{Error: "invalid_request", Message: "file is required"})
		return nil
	}
	defer func() { _ = file.Close() }()
	if header.Size > imports.MaxFileBytes {
		writeTooLarge(c, imports.ErrTooLarge)
		return nil
	}
	var buf bytes.Buffer
	if _, err := io.Copy(&buf, io.LimitReader(file, imports.MaxFileBytes+1)); err != nil {
		apierror.Internal(c, err, "failed to read file")
		return nil
	}

	table, err := imports.ReadTable(buf.Bytes(), header.Filename)
	if err != nil {
		if errors.Is(err, imports.ErrTooLarge) || errors.Is(err, imports.ErrTooManyRows) {
			writeTooLarge(c, err)
			return nil
		}
		apierror.WriteResponse(c, http.StatusBadRequest, models.ErrorResponse{Error: "invalid_file", Message: err.Error()})
		return nil
	}
	prepared, err := imports.Prepare(table, src, entity, mapping)
	if err != nil {
		apierror.WriteResponse(c, http.StatusBadRequest, models.ErrorResponse{Error: "invalid_mapping", Message: err.Error()})
		return nil
	}
	return &importUpload{filename: filepath.Base(header.Filename), source: src, entity: entity, table: table, prepared: prepared}
}

func writeTooLarge(c *gin.Context, err error) {
	apierror.WriteResponse(c, http.StatusRequestEntityTooLarge, models.ErrorResponse{Error: "file_too_large", Message: err.Error()})
}

// PreviewImport maps and validates an uploaded file without saving anything.
// @Summary		Preview an import
// @Description	Reads an uploaded CSV or XLSX (max 10 MiB and 50,000 rows; first sheet of a workbook) exported from Jobber, Housecall Pro or anywhere else, maps its columns to the entity's fields (automatically by source, or through the mapping form field: a JSON object of field key → column header) and validates every row. Returns the headers, the fields, the mapping used, what it still lacks, counts of valid/skipped/failing rows, the first 20 rows as they would be saved and the first 100 row problems. Nothing is saved. Customers, sites, jobs and invoices are matched to existing customers only when the import runs. Permission: customers.create (customers), customers.update (sites), jobs.create (jobs), invoices.create (invoices).
// @Tags			Imports
// @Accept			multipart/form-data
// @Produce		json
// @Security		BearerAuth
// @Param			entity	path		string	true	"What the file holds"	Enums(customers, sites, jobs, invoices)
// @Param			file	formData	file	true	"CSV or XLSX file"
// @Param			source	formData	string	false	"Where the file came from (default generic)"	Enums(generic, jobber, housecall_pro)
// @Param			mapping	formData	string	false	"JSON object of field key → column header; replaces the automatic mapping"
// @Success		200		{object}	models.ImportPreviewResponse
// @Failure		400		{object}	models.ErrorResponse
// @Failure		401		{object}	models.ErrorResponse
// @Failure		403		{object}	models.ErrorResponse
// @Failure		404		{object}	models.ErrorResponse
// @Failure		413		{object}	models.ErrorResponse
// @Router			/imports/{entity}/preview [post]
func (h *ImportsHandler) PreviewImport(c *gin.Context) {
	up := h.readUpload(c)
	if up == nil {
		return
	}
	now := h.now()
	resp := models.ImportPreviewResponse{
		Source: string(up.source), Entity: string(up.entity), Filename: up.filename,
		Headers: up.table.Headers, Mapping: up.prepared.Columns.HeaderMapping(up.table.Headers),
		Missing: up.prepared.Missing, TotalRows: len(up.prepared.Records),
		Rows: []models.ImportPreviewRow{}, Errors: []models.ImportRowError{},
	}
	if resp.Missing == nil {
		resp.Missing = []string{}
	}
	for _, f := range imports.Fields(up.entity) {
		resp.Fields = append(resp.Fields, models.ImportField{Key: f.Key, Label: f.Label, Required: f.Required, Group: f.Group, Help: f.Help})
	}
	for i := range up.prepared.Records {
		rec := &up.prepared.Records[i]
		var rowErrs []models.ImportRowError
		if len(resp.Missing) == 0 {
			for _, e := range imports.Validate(up.source, up.entity, rec, now) {
				rowErrs = append(rowErrs, models.ImportRowError{Row: rec.Row, Field: e.Field, Message: e.Message})
			}
		}
		skip := rec.Values[imports.SkipKey]
		delete(rec.Values, imports.SkipKey)
		switch {
		case len(rowErrs) > 0:
			resp.ErrorRows++
		case skip != "":
			resp.SkipRows++
		default:
			resp.ValidRows++
		}
		for _, e := range rowErrs {
			if len(resp.Errors) < importPreviewErrors {
				resp.Errors = append(resp.Errors, e)
			}
		}
		if len(resp.Rows) < importPreviewRows {
			resp.Rows = append(resp.Rows, models.ImportPreviewRow{Row: rec.Row, Values: rec.Values, Errors: rowErrs, Skip: skip})
		}
	}
	c.JSON(http.StatusOK, resp)
}

// StartImport queues an import and runs its first chunk.
// @Summary		Start an import
// @Description	Uploads the same form as the preview and starts the import: the rows are stored as a job and imported in chunks of up to 20 seconds (the first one during this request; POST /imports/{entity}/{job_id}/run or the scheduled imports job runs the rest). Re-importing the same file is safe: customers, jobs and invoices it created before are recognised by their old ID (or email / invoice number) and skipped, sites by address. Rows that fail are listed in the job's errors. 400 missing_columns when the mapping lacks a required field. Permission as for the preview.
// @Tags			Imports
// @Accept			multipart/form-data
// @Produce		json
// @Security		BearerAuth
// @Param			entity	path		string	true	"What the file holds"	Enums(customers, sites, jobs, invoices)
// @Param			file	formData	file	true	"CSV or XLSX file"
// @Param			source	formData	string	false	"Where the file came from (default generic)"	Enums(generic, jobber, housecall_pro)
// @Param			mapping	formData	string	false	"JSON object of field key → column header; replaces the automatic mapping"
// @Success		202		{object}	models.ImportStatusResponse
// @Failure		400		{object}	models.ErrorResponse
// @Failure		401		{object}	models.ErrorResponse
// @Failure		403		{object}	models.ErrorResponse
// @Failure		404		{object}	models.ErrorResponse
// @Failure		413		{object}	models.ErrorResponse
// @Failure		500		{object}	models.ErrorResponse
// @Router			/imports/{entity} [post]
func (h *ImportsHandler) StartImport(c *gin.Context) {
	up := h.readUpload(c)
	if up == nil {
		return
	}
	if len(up.prepared.Missing) > 0 {
		apierror.WriteResponse(c, http.StatusBadRequest, models.ErrorResponse{
			Error: "missing_columns", Message: "map these before importing", Details: up.prepared.Missing,
		})
		return
	}
	userID, ok := importUserID(c)
	if !ok {
		return
	}
	ctx := c.Request.Context()
	job, err := h.imports.CreateMappedImportJob(ctx, userID, up.filename, up.source, up.entity, up.prepared.Records)
	if err != nil {
		apierror.Internal(c, err, "failed to create import job")
		return
	}
	job = h.runChunk(c, job.ID, job)
	if job == nil {
		return
	}
	h.writeStatus(c, http.StatusAccepted, job)
}

// RunImport runs the next chunk of a pending import.
// @Summary		Continue an import
// @Description	Claims the import (if it is pending) and imports rows for up to 20 seconds, then returns its status. The import page calls this until the job is completed or failed; a job another runner holds, or one already finished, is returned as it is. Permission as for the preview.
// @Tags			Imports
// @Produce		json
// @Security		BearerAuth
// @Param			entity	path		string	true	"Entity"	Enums(customers, sites, jobs, invoices)
// @Param			job_id	path		string	true	"Import job ID"
// @Success		200		{object}	models.ImportStatusResponse
// @Failure		400		{object}	models.ErrorResponse
// @Failure		404		{object}	models.ErrorResponse
// @Failure		500		{object}	models.ErrorResponse
// @Router			/imports/{entity}/{job_id}/run [post]
func (h *ImportsHandler) RunImport(c *gin.Context) {
	job := h.loadJob(c)
	if job == nil {
		return
	}
	if job.Status == models.ImportJobStatusPending {
		if job = h.runChunk(c, job.ID, job); job == nil {
			return
		}
	}
	h.writeStatus(c, http.StatusOK, job)
}

// GetImport returns an import's progress and first row errors.
// @Summary		Get an import
// @Description	Progress, counts (successful, skipped, failed) and the first 100 row errors of one of the tenant's imports of this entity. Permission: customers.list (customers, sites), jobs.list, invoices.list.
// @Tags			Imports
// @Produce		json
// @Security		BearerAuth
// @Param			entity	path		string	true	"Entity"	Enums(customers, sites, jobs, invoices)
// @Param			job_id	path		string	true	"Import job ID"
// @Success		200		{object}	models.ImportStatusResponse
// @Failure		400		{object}	models.ErrorResponse
// @Failure		404		{object}	models.ErrorResponse
// @Failure		500		{object}	models.ErrorResponse
// @Router			/imports/{entity}/{job_id} [get]
func (h *ImportsHandler) GetImport(c *gin.Context) {
	job := h.loadJob(c)
	if job == nil {
		return
	}
	h.writeStatus(c, http.StatusOK, job)
}

// ListImports lists the tenant's recent imports of an entity.
// @Summary		List imports
// @Description	The tenant's 20 most recent imports of this entity, newest first (no row errors). Permission as for GET /imports/{entity}/{job_id}.
// @Tags			Imports
// @Produce		json
// @Security		BearerAuth
// @Param			entity	path		string	true	"Entity"	Enums(customers, sites, jobs, invoices)
// @Success		200		{object}	models.ImportJobListResponse
// @Failure		404		{object}	models.ErrorResponse
// @Failure		500		{object}	models.ErrorResponse
// @Router			/imports/{entity} [get]
func (h *ImportsHandler) ListImports(c *gin.Context) {
	entity, err := imports.ParseEntity(c.Param("entity"))
	if err != nil {
		apierror.WriteResponse(c, http.StatusNotFound, models.ErrorResponse{Error: "not_found", Message: err.Error()})
		return
	}
	jobs, err := h.imports.ListImportJobs(c.Request.Context(), []string{string(entity)}, importListLimit)
	if err != nil {
		apierror.Internal(c, err, "failed to list imports")
		return
	}
	resp := models.ImportJobListResponse{Jobs: make([]models.ImportStatusResponse, 0, len(jobs))}
	for i := range jobs {
		resp.Jobs = append(resp.Jobs, models.ImportStatusFromJob(&jobs[i], nil))
	}
	c.JSON(http.StatusOK, resp)
}

// loadJob reads the tenant's job named in the URL; it must be an import of
// the URL's entity (so a route's permission covers it).
func (h *ImportsHandler) loadJob(c *gin.Context) *models.ImportJob {
	entity, err := imports.ParseEntity(c.Param("entity"))
	if err != nil {
		apierror.WriteResponse(c, http.StatusNotFound, models.ErrorResponse{Error: "not_found", Message: err.Error()})
		return nil
	}
	id, err := uuid.Parse(c.Param("job_id"))
	if err != nil {
		apierror.WriteResponse(c, http.StatusBadRequest, models.ErrorResponse{Error: "invalid_request", Message: "invalid job ID"})
		return nil
	}
	job, err := h.imports.GetImportJob(c.Request.Context(), id)
	if err == nil && job.Entity != string(entity) {
		err = services.ErrImportJobNotFound
	}
	if err != nil {
		if errors.Is(err, services.ErrImportJobNotFound) {
			apierror.WriteResponse(c, http.StatusNotFound, models.ErrorResponse{Error: "not_found", Message: "import not found"})
			return nil
		}
		apierror.Internal(c, err, "failed to load import")
		return nil
	}
	return job
}

// runChunk claims a pending job of the caller's tenant and imports for up
// to services.ImportInlineBudget. The run outlives a client disconnect so a
// batch is never cut mid-way. It returns the job's latest state (nil after
// writing an error).
func (h *ImportsHandler) runChunk(c *gin.Context, jobID uuid.UUID, current *models.ImportJob) *models.ImportJob {
	ctx := c.Request.Context()
	runCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), services.ImportInlineBudget+10*time.Second)
	defer cancel()
	claimed, err := h.imports.ClaimImportJob(runCtx, jobID)
	if err != nil {
		logging.Error(ctx, "[IMPORTS-HANDLER] Claim failed; left for the imports job", map[string]any{"error": err.Error(), "job_id": jobID.String()})
		return current
	}
	if claimed == nil {
		return current // another runner has it, or it is done
	}
	job, err := h.imports.RunImportChunk(runCtx, claimed, services.ImportInlineBudget)
	if err != nil {
		apierror.Internal(c, err, "failed to run import")
		return nil
	}
	return job
}

func (h *ImportsHandler) writeStatus(c *gin.Context, status int, job *models.ImportJob) {
	errs, err := h.imports.GetImportErrors(c.Request.Context(), job.ID, importStatusErrors)
	if err != nil {
		apierror.Internal(c, err, "failed to load import errors")
		return
	}
	// Re-read the counts: a chunk run elsewhere may have moved them.
	if fresh, err := h.imports.GetImportJob(c.Request.Context(), job.ID); err == nil {
		job = fresh
	}
	c.JSON(status, models.ImportStatusFromJob(job, errs))
}

func importUserID(c *gin.Context) (uuid.UUID, bool) {
	v, exists := c.Get("user_id")
	id, ok := v.(uuid.UUID)
	if !exists || !ok {
		apierror.WriteResponse(c, http.StatusUnauthorized, models.ErrorResponse{Error: "unauthorized", Message: "user not authenticated"})
		return uuid.Nil, false
	}
	return id, true
}
