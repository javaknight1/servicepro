package handlers

import (
	"fmt"
	"io"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/javaknight1/servicepro/backend/internal/models"
	"github.com/javaknight1/servicepro/backend/internal/services"
)

// ExportHandler handles export-related API requests
type ExportHandler struct {
	exportService services.UnifiedExportService
}

// NewExportHandler creates a new export handler
func NewExportHandler(exportService services.UnifiedExportService) *ExportHandler {
	return &ExportHandler{
		exportService: exportService,
	}
}

// StartExportRequest represents a request to start an export
type StartExportRequest struct {
	Format          models.ExportFormat        `json:"format" binding:"required,oneof=csv json xlsx pdf"`
	ExportType      models.ExportType          `json:"export_type" binding:"required"`
	IncludeSections []string                   `json:"include_sections"`
	Fields          []string                   `json:"fields"`
	Encoding        string                     `json:"encoding"`
	Query           map[string]interface{}     `json:"query"`
	CSVOptions      *models.CSVExportOptions   `json:"csv_options"`
	PDFOptions      *models.PDFExportOptions   `json:"pdf_options"`
	ExcelOptions    *models.ExcelExportOptions `json:"excel_options"`
}

// StartExport initiates an async export job
// @Summary Start an export job
// @Description Initiates an async export job for the specified data type and format
// @Tags exports
// @Accept json
// @Produce json
// @Param request body StartExportRequest true "Export configuration"
// @Success 202 {object} models.ExportJobResponse
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/v1/exports [post]
func (h *ExportHandler) StartExport(c *gin.Context) {
	var req StartExportRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Get user ID from context (set by auth middleware)
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user not authenticated"})
		return
	}

	config := &models.ExportConfig{
		Format:          req.Format,
		ExportType:      req.ExportType,
		IncludeSections: req.IncludeSections,
		Fields:          req.Fields,
		Encoding:        req.Encoding,
		Query:           req.Query,
		CSVOptions:      req.CSVOptions,
		PDFOptions:      req.PDFOptions,
		ExcelOptions:    req.ExcelOptions,
	}

	job, err := h.exportService.StartExport(c.Request.Context(), userID.(uuid.UUID), config)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusAccepted, job.ToResponse())
}

// GetExportStatus returns the status of an export job
// @Summary Get export job status
// @Description Returns the current status of an export job
// @Tags exports
// @Produce json
// @Param id path string true "Export job ID"
// @Success 200 {object} models.ExportJobResponse
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Router /api/v1/exports/{id} [get]
func (h *ExportHandler) GetExportStatus(c *gin.Context) {
	idStr := c.Param("id")
	jobID, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid job ID"})
		return
	}

	response, err := h.exportService.GetExportStatus(c.Request.Context(), jobID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "export job not found"})
		return
	}

	c.JSON(http.StatusOK, response)
}

// GetExportProgress returns real-time progress of an export job
// @Summary Get export progress
// @Description Returns real-time progress information for an export job
// @Tags exports
// @Produce json
// @Param id path string true "Export job ID"
// @Success 200 {object} models.ExportProgress
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Router /api/v1/exports/{id}/progress [get]
func (h *ExportHandler) GetExportProgress(c *gin.Context) {
	idStr := c.Param("id")
	jobID, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid job ID"})
		return
	}

	progress := h.exportService.GetProgress(jobID)
	if progress == nil {
		// Fall back to database status
		response, err := h.exportService.GetExportStatus(c.Request.Context(), jobID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "export job not found"})
			return
		}
		c.JSON(http.StatusOK, models.ExportProgress{
			JobID:            response.ID,
			Status:           response.Status,
			TotalRecords:     response.TotalRecords,
			ProcessedRecords: response.ProcessedRecords,
			Progress:         response.Progress,
		})
		return
	}

	c.JSON(http.StatusOK, progress)
}

// StreamExportProgress streams real-time progress updates via SSE
// @Summary Stream export progress
// @Description Streams real-time progress updates for an export job via Server-Sent Events
// @Tags exports
// @Produce text/event-stream
// @Param id path string true "Export job ID"
// @Success 200 {string} string "SSE stream"
// @Failure 400 {object} map[string]string
// @Router /api/v1/exports/{id}/stream [get]
func (h *ExportHandler) StreamExportProgress(c *gin.Context) {
	idStr := c.Param("id")
	jobID, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid job ID"})
		return
	}

	// Set headers for SSE
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("Access-Control-Allow-Origin", "*")

	progressChan := h.exportService.SubscribeToProgress(jobID)

	// Stream progress updates
	c.Stream(func(w io.Writer) bool {
		select {
		case progress, ok := <-progressChan:
			if !ok {
				return false
			}
			c.SSEvent("progress", progress)

			// Close stream when completed or failed
			if progress.Status == models.ExportJobStatusCompleted ||
				progress.Status == models.ExportJobStatusFailed ||
				progress.Status == models.ExportJobStatusCancelled {
				return false
			}
			return true
		case <-c.Request.Context().Done():
			return false
		}
	})
}

// DownloadExport downloads the completed export file
// @Summary Download export file
// @Description Downloads the completed export file
// @Tags exports
// @Produce application/octet-stream
// @Param id path string true "Export job ID"
// @Success 200 {file} binary
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Failure 409 {object} map[string]string
// @Router /api/v1/exports/{id}/download [get]
func (h *ExportHandler) DownloadExport(c *gin.Context) {
	idStr := c.Param("id")
	jobID, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid job ID"})
		return
	}

	file, job, err := h.exportService.GetExportFile(c.Request.Context(), jobID)
	if err != nil {
		if err.Error() == "export not ready for download" {
			c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusNotFound, gin.H{"error": "export not found"})
		return
	}
	defer file.Close()

	// Set headers for download
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", job.Filename))
	c.Header("Content-Type", job.MimeType)
	c.Header("Content-Length", strconv.FormatInt(job.FileSize, 10))

	// Stream file to response
	c.Status(http.StatusOK)
	io.Copy(c.Writer, file)
}

// CancelExport cancels an in-progress export job
// @Summary Cancel export job
// @Description Cancels an in-progress export job
// @Tags exports
// @Produce json
// @Param id path string true "Export job ID"
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Router /api/v1/exports/{id}/cancel [post]
func (h *ExportHandler) CancelExport(c *gin.Context) {
	idStr := c.Param("id")
	jobID, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid job ID"})
		return
	}

	if err := h.exportService.CancelExport(c.Request.Context(), jobID); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "export cancelled successfully"})
}

// ListExports lists recent exports for the current user
// @Summary List exports
// @Description Lists recent export jobs for the current user
// @Tags exports
// @Produce json
// @Param limit query int false "Maximum number of exports to return" default(20)
// @Success 200 {array} models.ExportJobResponse
// @Failure 401 {object} map[string]string
// @Router /api/v1/exports [get]
func (h *ExportHandler) ListExports(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user not authenticated"})
		return
	}

	limitStr := c.DefaultQuery("limit", "20")
	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit < 1 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}

	exports, err := h.exportService.ListExports(c.Request.Context(), userID.(uuid.UUID), limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, exports)
}

// QuickExport performs a synchronous export for small datasets
// @Summary Quick export
// @Description Performs a synchronous export for small datasets (returns data directly)
// @Tags exports
// @Accept json
// @Produce application/octet-stream
// @Param request body StartExportRequest true "Export configuration"
// @Success 200 {file} binary
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/v1/exports/quick [post]
func (h *ExportHandler) QuickExport(c *gin.Context) {
	var req StartExportRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	config := &models.ExportConfig{
		Format:          req.Format,
		ExportType:      req.ExportType,
		IncludeSections: req.IncludeSections,
		Fields:          req.Fields,
		Encoding:        req.Encoding,
		Query:           req.Query,
		CSVOptions:      req.CSVOptions,
		PDFOptions:      req.PDFOptions,
		ExcelOptions:    req.ExcelOptions,
	}

	// For quick export, we fetch data and export synchronously
	// This should only be used for small datasets
	var data []byte
	var err error
	var filename string
	timestamp := "export"

	switch config.Format {
	case models.ExportFormatCSV:
		// Use the existing sync export methods
		data, err = h.exportService.ExportToCSV(c.Request.Context(), nil, config)
		filename = fmt.Sprintf("%s_%s.csv", config.ExportType, timestamp)
	case models.ExportFormatJSON:
		data, err = h.exportService.ExportToJSON(c.Request.Context(), nil, config)
		filename = fmt.Sprintf("%s_%s.json", config.ExportType, timestamp)
	case models.ExportFormatExcel:
		data, err = h.exportService.ExportToExcel(c.Request.Context(), nil, config)
		filename = fmt.Sprintf("%s_%s.xlsx", config.ExportType, timestamp)
	case models.ExportFormatPDF:
		data, err = h.exportService.ExportToPDF(c.Request.Context(), nil, config)
		filename = fmt.Sprintf("%s_%s.pdf", config.ExportType, timestamp)
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "unsupported format"})
		return
	}

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Set headers for download
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filename))
	c.Header("Content-Type", config.Format.GetMimeType())
	c.Header("Content-Length", strconv.Itoa(len(data)))

	c.Data(http.StatusOK, config.Format.GetMimeType(), data)
}

// GetSupportedFormats returns supported export formats and types
// @Summary Get supported formats
// @Description Returns list of supported export formats and data types
// @Tags exports
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Router /api/v1/exports/formats [get]
func (h *ExportHandler) GetSupportedFormats(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"formats": []map[string]string{
			{"value": "csv", "label": "CSV (Comma Separated)", "mime_type": models.ExportFormatCSV.GetMimeType()},
			{"value": "json", "label": "JSON", "mime_type": models.ExportFormatJSON.GetMimeType()},
			{"value": "xlsx", "label": "Excel (XLSX)", "mime_type": models.ExportFormatExcel.GetMimeType()},
			{"value": "pdf", "label": "PDF", "mime_type": models.ExportFormatPDF.GetMimeType()},
		},
		"export_types": []map[string]string{
			{"value": "customers", "label": "Customers"},
			{"value": "jobs", "label": "Jobs"},
			{"value": "invoices", "label": "Invoices"},
			{"value": "quotes", "label": "Quotes"},
			{"value": "revenue_report", "label": "Revenue Report"},
			{"value": "customer_report", "label": "Customer Report"},
		},
		"csv_options": map[string]interface{}{
			"delimiters": []map[string]string{
				{"value": ",", "label": "Comma"},
				{"value": "tab", "label": "Tab"},
				{"value": "semicolon", "label": "Semicolon"},
			},
		},
		"pdf_options": map[string]interface{}{
			"page_sizes":   []string{"A4", "Letter", "Legal"},
			"orientations": []string{"portrait", "landscape"},
		},
	})
}
