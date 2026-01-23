package handlers

import (
	"bytes"
	"fmt"
	"io"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/javaknight1/servicepro/backend/internal/models"
	"github.com/javaknight1/servicepro/backend/internal/services"
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

// ImportCustomers handles customer import
// POST /api/v1/customers/import
func (h *ImportExportHandler) ImportCustomers(c *gin.Context) {
	// Parse form data
	var req models.ImportRequest
	if err := c.ShouldBind(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "invalid_request",
			Message: err.Error(),
		})
		return
	}

	// Set default duplicate handling
	if req.DuplicateHandling == "" {
		req.DuplicateHandling = "error"
	}

	// Get uploaded file
	file, header, err := c.Request.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "invalid_request",
			Message: "file is required",
		})
		return
	}
	defer file.Close()

	// Validate file format
	if req.Format != "csv" {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "invalid_format",
			Message: "only CSV format is supported for import",
		})
		return
	}

	// Read file content into buffer for multiple reads
	buf := new(bytes.Buffer)
	if _, err := io.Copy(buf, file); err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "internal_error",
			Message: "failed to read file",
		})
		return
	}

	// Count rows
	rowCount, err := csvpkg.CountRows(bytes.NewReader(buf.Bytes()))
	if err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "invalid_file",
			Message: fmt.Sprintf("failed to read CSV: %v", err),
		})
		return
	}

	if rowCount == 0 {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "invalid_file",
			Message: "file contains no data rows",
		})
		return
	}

	if rowCount > 10000 {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "file_too_large",
			Message: "file exceeds maximum of 10,000 rows",
		})
		return
	}

	// Parse CSV
	parser := csvpkg.NewParser(bytes.NewReader(buf.Bytes()))
	rows, parseErrors, err := parser.ParseCustomers()
	if err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "parse_error",
			Message: fmt.Sprintf("failed to parse CSV: %v", err),
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

		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "validation_error",
			Message: fmt.Sprintf("file contains %d validation errors", len(parseErrors)),
			Details: errorMessages,
		})
		return
	}

	// Get user ID from context (set by auth middleware)
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, models.ErrorResponse{
			Error:   "unauthorized",
			Message: "user not authenticated",
		})
		return
	}

	userUUID, ok := userID.(uuid.UUID)
	if !ok {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "internal_error",
			Message: "invalid user ID format",
		})
		return
	}

	// Create import job
	job, err := h.importService.CreateImportJob(
		c.Request.Context(),
		userUUID,
		header.Filename,
		len(rows),
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "internal_error",
			Message: "failed to create import job",
		})
		return
	}

	// Enqueue job for background processing
	if err := h.importService.EnqueueImportJob(
		c.Request.Context(),
		job.ID,
		rows,
		req.DuplicateHandling,
		req.DryRun,
	); err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "internal_error",
			Message: "failed to enqueue import job",
		})
		return
	}

	c.JSON(http.StatusAccepted, models.ImportResponse{
		JobID:     job.ID,
		Status:    job.Status,
		Message:   "Import job created successfully",
		TotalRows: len(rows),
	})
}

// GetImportStatus returns the status of an import job
// GET /api/v1/customers/import/:job_id
func (h *ImportExportHandler) GetImportStatus(c *gin.Context) {
	jobIDStr := c.Param("job_id")
	jobID, err := uuid.Parse(jobIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "invalid_request",
			Message: "invalid job ID",
		})
		return
	}

	job, err := h.importService.GetImportJob(c.Request.Context(), jobID)
	if err != nil {
		c.JSON(http.StatusNotFound, models.ErrorResponse{
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
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "internal_error",
			Message: "failed to retrieve import errors",
		})
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
// GET /api/v1/customers/export
// Deprecated: Use POST /api/v1/exports with export_type="customers" for async exports
func (h *ImportExportHandler) ExportCustomers(c *gin.Context) {
	c.JSON(http.StatusGone, models.ErrorResponse{
		Error:   "deprecated",
		Message: "This endpoint has been deprecated. Use POST /api/v1/exports with export_type='customers' for async exports with progress tracking.",
	})
}

// GetImportTemplate returns a CSV template for import
// GET /api/v1/customers/import/template
func (h *ImportExportHandler) GetImportTemplate(c *gin.Context) {
	template := csvpkg.GetCSVTemplate()

	c.Header("Content-Type", "text/csv")
	c.Header("Content-Disposition", `attachment; filename="customer_import_template.csv"`)
	c.Data(http.StatusOK, "text/csv", template)
}
