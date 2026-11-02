package handlers

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/javaknight1/servicepro/backend/internal/api/apierror"
	"github.com/javaknight1/servicepro/backend/internal/models"
	"github.com/javaknight1/servicepro/backend/internal/services"
	"github.com/javaknight1/servicepro/backend/pkg/clients/logging"
	csvpkg "github.com/javaknight1/servicepro/backend/pkg/csv"
)

// ImportExportHandler handles import/export operations
type ImportExportHandler struct {
	importService *services.ImportService
}

// NewImportExportHandler creates a new import/export handler
func NewImportExportHandler(
	importService *services.ImportService,
) *ImportExportHandler {
	return &ImportExportHandler{
		importService: importService,
	}
}

// importInlineTimeout bounds an import run inside the upload request.
const importInlineTimeout = 2 * time.Minute

// ImportCustomers handles customer import
// @Summary		Import customers
// @Description	Imports customers from a CSV file (max 5 MiB and 10,000 rows). Files up to 1,000 rows are imported during the request (status completed/failed in the response); larger files are queued (status pending) and imported by the scheduled imports job. Poll GET /customers/import/{job_id} for progress and per-row errors.
// @Tags			Import/Export
// @Accept			multipart/form-data
// @Produce		json
// @Security		BearerAuth
// @Param			file				formData	file	true	"CSV file to import"
// @Param			format				formData	string	true	"File format (csv)"
// @Param			duplicate_handling	formData	string	false	"How to handle duplicates (error, skip, update)"
// @Param			dry_run				formData	bool	false	"Preview import without saving"
// @Success		202					{object}	models.ImportResponse
// @Failure		400					{object}	models.ErrorResponse
// @Failure		401					{object}	models.ErrorResponse
// @Failure		413					{object}	models.ErrorResponse
// @Failure		500					{object}	models.ErrorResponse
// @Router			/customers/import [post]
func (h *ImportExportHandler) ImportCustomers(c *gin.Context) {
	// Cap the upload before anything reads the body (F-23). The multipart
	// envelope adds a little over the file itself.
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, services.ImportMaxUploadBytes+64<<10)
	if err := c.Request.ParseMultipartForm(1 << 20); err != nil {
		var tooBig *http.MaxBytesError
		if errors.As(err, &tooBig) {
			apierror.WriteResponse(c, http.StatusRequestEntityTooLarge, models.ErrorResponse{
				Error:   "file_too_large",
				Message: fmt.Sprintf("file exceeds the maximum size of %d MiB", services.ImportMaxUploadBytes>>20),
			})
			return
		}
		apierror.WriteResponse(c, http.StatusBadRequest, models.ErrorResponse{
			Error:   "invalid_request",
			Message: "request must be multipart/form-data with a file",
		})
		return
	}
	defer func() {
		if c.Request.MultipartForm != nil {
			_ = c.Request.MultipartForm.RemoveAll()
		}
	}()

	// Parse form data
	var req models.ImportRequest
	if err := c.ShouldBind(&req); err != nil {
		apierror.BindError(c, err)
		return
	}

	// Set default duplicate handling
	if req.DuplicateHandling == "" {
		req.DuplicateHandling = "error"
	}

	// Get uploaded file
	file, header, err := c.Request.FormFile("file")
	if err != nil {
		apierror.WriteResponse(c, http.StatusBadRequest, models.ErrorResponse{
			Error:   "invalid_request",
			Message: "file is required",
		})
		return
	}
	defer file.Close()

	// Validate file format
	if req.Format != "csv" {
		apierror.WriteResponse(c, http.StatusBadRequest, models.ErrorResponse{
			Error:   "invalid_format",
			Message: "only CSV format is supported for import",
		})
		return
	}

	if header.Size > services.ImportMaxUploadBytes {
		apierror.WriteResponse(c, http.StatusRequestEntityTooLarge, models.ErrorResponse{
			Error:   "file_too_large",
			Message: fmt.Sprintf("file exceeds the maximum size of %d MiB", services.ImportMaxUploadBytes>>20),
		})
		return
	}

	// Read file content into buffer for multiple reads
	buf := new(bytes.Buffer)
	if _, err := io.Copy(buf, io.LimitReader(file, services.ImportMaxUploadBytes+1)); err != nil {
		apierror.Internal(c, err, "failed to read file")
		return
	}

	// Count rows
	rowCount, err := csvpkg.CountRows(bytes.NewReader(buf.Bytes()))
	if err != nil {
		apierror.WriteResponse(c, http.StatusBadRequest, models.ErrorResponse{
			Error:   "invalid_file",
			Message: "failed to read CSV: the file is not valid CSV",
		})
		return
	}

	if rowCount == 0 {
		apierror.WriteResponse(c, http.StatusBadRequest, models.ErrorResponse{
			Error:   "invalid_file",
			Message: "file contains no data rows",
		})
		return
	}

	if rowCount > services.ImportMaxRows {
		apierror.WriteResponse(c, http.StatusBadRequest, models.ErrorResponse{
			Error:   "file_too_large",
			Message: fmt.Sprintf("file exceeds maximum of %d rows", services.ImportMaxRows),
		})
		return
	}

	// Parse CSV
	parser := csvpkg.NewParser(bytes.NewReader(buf.Bytes()))
	rows, parseErrors, err := parser.ParseCustomers()
	if err != nil {
		apierror.WriteResponse(c, http.StatusBadRequest, models.ErrorResponse{
			Error:   "parse_error",
			Message: "failed to parse CSV: the file is not valid CSV",
		})
		return
	}

	// Check for parsing errors
	if len(parseErrors) > 0 {
		// Return first few errors
		errorMessages := make([]string, 0, len(parseErrors))
		for i, e := range parseErrors {
			if i >= 10 {
				errorMessages = append(errorMessages, fmt.Sprintf("... and %d more errors", len(parseErrors)-10))
				break
			}
			errorMessages = append(errorMessages, e.String())
		}

		apierror.WriteResponse(c, http.StatusBadRequest, models.ErrorResponse{
			Error:   "validation_error",
			Message: fmt.Sprintf("file contains %d validation errors", len(parseErrors)),
			Details: errorMessages,
		})
		return
	}

	// Get user ID from context (set by auth middleware)
	userID, exists := c.Get("user_id")
	if !exists {
		apierror.WriteResponse(c, http.StatusUnauthorized, models.ErrorResponse{
			Error:   "unauthorized",
			Message: "user not authenticated",
		})
		return
	}

	userUUID, ok := userID.(uuid.UUID)
	if !ok {
		apierror.Internal(c, fmt.Errorf("user_id has type %T", userID), "invalid user ID format")
		return
	}

	ctx := c.Request.Context()
	job, err := h.importService.CreateImportJob(ctx, userUUID, header.Filename, rows, req.DuplicateHandling, req.DryRun)
	if err != nil {
		apierror.Internal(c, err, "failed to create import job")
		return
	}

	message := "Import queued; it runs within a few minutes"
	if len(rows) <= services.ImportInlineMaxRows {
		// Small file: run it now. The claim makes this safe against the
		// scheduled runner picking the same job; whoever claims it runs it.
		// The run outlives a client disconnect so the job is never left
		// half-done in processing.
		runCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), importInlineTimeout)
		defer cancel()
		if claimed, err := h.importService.ClaimImportJob(runCtx, job.ID); err != nil {
			logging.Error(ctx, "[IMPORT-HANDLER] Inline claim failed; left for the imports job", map[string]any{"error": err.Error(), "job_id": job.ID.String()})
		} else if claimed != nil {
			finished, err := h.importService.RunImportJob(runCtx, claimed)
			if err != nil {
				apierror.Internal(c, err, "failed to run import job")
				return
			}
			job = finished
			message = "Import finished"
		}
	}

	c.JSON(http.StatusAccepted, models.ImportResponse{
		JobID:     job.ID,
		Status:    job.Status,
		Message:   message,
		TotalRows: len(rows),
	})
}

// GetImportStatus returns the status of an import job
// @Summary		Get import status
// @Description	Returns the status and progress of an import job
// @Tags			Import/Export
// @Accept			json
// @Produce		json
// @Security		BearerAuth
// @Param			job_id	path		string	true	"Import Job ID"
// @Success		200		{object}	models.ImportStatusResponse
// @Failure		400		{object}	models.ErrorResponse
// @Failure		404		{object}	models.ErrorResponse
// @Failure		500		{object}	models.ErrorResponse
// @Router			/customers/import/{job_id} [get]
func (h *ImportExportHandler) GetImportStatus(c *gin.Context) {
	jobIDStr := c.Param("job_id")
	jobID, err := uuid.Parse(jobIDStr)
	if err != nil {
		apierror.WriteResponse(c, http.StatusBadRequest, models.ErrorResponse{
			Error:   "invalid_request",
			Message: "invalid job ID",
		})
		return
	}

	job, err := h.importService.GetImportJob(c.Request.Context(), jobID)
	if err != nil {
		apierror.WriteResponse(c, http.StatusNotFound, models.ErrorResponse{
			Error:   "not_found",
			Message: "import job not found",
		})
		return
	}

	// Calculate progress
	progress := 0.0
	if job.TotalRows > 0 {
		progress = float64(job.ProcessedRows) / float64(job.TotalRows) * 100
	}

	// Get errors (limit to 100)
	importErrors, err := h.importService.GetImportErrors(c.Request.Context(), jobID, 100)
	if err != nil {
		apierror.Internal(c, err, "failed to retrieve import errors")
		return
	}

	// Convert to response format
	errorResponses := make([]models.ImportErrorResponse, len(importErrors))
	for i, e := range importErrors {
		errorResponses[i] = models.ImportErrorResponse{
			RowNumber: e.RowNumber,
			Field:     e.Field,
			Error:     e.Error,
			RowData:   e.RowData,
		}
	}

	response := models.ImportStatusResponse{
		JobID:          job.ID,
		Status:         job.Status,
		TotalRows:      job.TotalRows,
		ProcessedRows:  job.ProcessedRows,
		SuccessfulRows: job.SuccessfulRows,
		FailedRows:     job.FailedRows,
		Progress:       progress,
		ErrorSummary:   job.ErrorSummary,
		Errors:         errorResponses,
		StartedAt:      job.StartedAt,
		CompletedAt:    job.CompletedAt,
		CreatedAt:      job.CreatedAt,
	}

	c.JSON(http.StatusOK, response)
}

// ExportCustomers handles customer export
// @Summary		Export customers (deprecated)
// @Description	Deprecated: Use POST /api/v1/exports with export_type="customers" for async exports
// @Tags			Import/Export
// @Accept			json
// @Produce		json
// @Security		BearerAuth
// @Deprecated
// @Failure		410	{object}	models.ErrorResponse
// @Router			/customers/export [get]
func (h *ImportExportHandler) ExportCustomers(c *gin.Context) {
	apierror.WriteResponse(c, http.StatusGone, models.ErrorResponse{
		Error:   "deprecated",
		Message: "This endpoint has been deprecated. Use POST /api/v1/exports with export_type='customers' for async exports with progress tracking.",
	})
}

// GetImportTemplate returns a CSV template for import
// @Summary		Get import template
// @Description	Downloads a CSV template for customer import
// @Tags			Import/Export
// @Accept			json
// @Produce		text/csv
// @Security		BearerAuth
// @Success		200	{file}	file
// @Router			/customers/import/template [get]
func (h *ImportExportHandler) GetImportTemplate(c *gin.Context) {
	template := csvpkg.GetCSVTemplate()

	c.Header("Content-Type", "text/csv")
	c.Header("Content-Disposition", `attachment; filename="customer_import_template.csv"`)
	c.Data(http.StatusOK, "text/csv", template)
}
