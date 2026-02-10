package services

import (
	"bytes"
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/xuri/excelize/v2"
	"gorm.io/gorm"

	"github.com/javaknight1/servicepro/backend/internal/models"
	"github.com/javaknight1/servicepro/backend/internal/utils/epoch"
)

// UnifiedExportService handles all export operations with progress tracking
type UnifiedExportService interface {
	// Async export operations
	StartExport(ctx context.Context, userID uuid.UUID, config *models.ExportConfig) (*models.ExportJob, error)
	GetExportStatus(ctx context.Context, jobID uuid.UUID) (*models.ExportJobResponse, error)
	CancelExport(ctx context.Context, jobID uuid.UUID) error
	GetExportFile(ctx context.Context, jobID uuid.UUID) (io.ReadCloser, *models.ExportJob, error)
	ListExports(ctx context.Context, userID uuid.UUID, limit int) ([]models.ExportJobResponse, error)
	CleanupExpiredExports(ctx context.Context) error

	// Sync export operations (for small datasets)
	ExportToCSV(ctx context.Context, data interface{}, config *models.ExportConfig) ([]byte, error)
	ExportToJSON(ctx context.Context, data interface{}, config *models.ExportConfig) ([]byte, error)
	ExportToExcel(ctx context.Context, data interface{}, config *models.ExportConfig) ([]byte, error)
	ExportToPDF(ctx context.Context, data interface{}, config *models.ExportConfig) ([]byte, error)

	// Progress tracking
	GetProgress(jobID uuid.UUID) *models.ExportProgress
	SubscribeToProgress(jobID uuid.UUID) <-chan models.ExportProgress
}

type unifiedExportService struct {
	db            *gorm.DB
	pdfService    *PDFService
	outputDir     string
	baseURL       string
	progressMap   map[uuid.UUID]*models.ExportProgress
	progressMutex sync.RWMutex
	subscribers   map[uuid.UUID][]chan models.ExportProgress
	subMutex      sync.RWMutex
}

// NewUnifiedExportService creates a new unified export service
func NewUnifiedExportService(db *gorm.DB, pdfService *PDFService, outputDir, baseURL string) UnifiedExportService {
	if outputDir == "" {
		outputDir = "./exports"
	}
	os.MkdirAll(outputDir, 0755)

	return &unifiedExportService{
		db:          db,
		pdfService:  pdfService,
		outputDir:   outputDir,
		baseURL:     baseURL,
		progressMap: make(map[uuid.UUID]*models.ExportProgress),
		subscribers: make(map[uuid.UUID][]chan models.ExportProgress),
	}
}

// StartExport initiates an async export job
func (s *unifiedExportService) StartExport(ctx context.Context, userID uuid.UUID, config *models.ExportConfig) (*models.ExportJob, error) {
	// Validate config
	if err := s.validateConfig(config); err != nil {
		return nil, fmt.Errorf("invalid export config: %w", err)
	}

	// Serialize query to JSON
	queryJSON, err := json.Marshal(config.Query)
	if err != nil {
		return nil, fmt.Errorf("failed to serialize query: %w", err)
	}

	// Create export job
	job := &models.ExportJob{
		ID:         uuid.New(),
		UserID:     userID,
		ExportType: config.ExportType,
		Format:     config.Format,
		Status:     models.ExportJobStatusPending,
		Query:      string(queryJSON),
		Encoding:   config.Encoding,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}

	if job.Encoding == "" {
		job.Encoding = "utf-8"
	}

	// Generate filename
	timestamp := time.Now().Format("20060102_150405")
	job.Filename = fmt.Sprintf("%s_%s%s", config.ExportType, timestamp, config.Format.GetExtension())
	job.FilePath = filepath.Join(s.outputDir, job.ID.String(), job.Filename)
	job.MimeType = config.Format.GetMimeType()

	// Save job to database
	if err := s.db.Create(job).Error; err != nil {
		return nil, fmt.Errorf("failed to create export job: %w", err)
	}

	// Initialize progress tracking
	s.initProgress(job.ID)

	// Capture job ID before starting goroutine to avoid race
	jobID := job.ID

	// Start async export in goroutine
	go s.processExport(context.Background(), jobID, config)

	return job, nil
}

// processExport handles the actual export processing
func (s *unifiedExportService) processExport(ctx context.Context, jobID uuid.UUID, config *models.ExportConfig) {
	startTime := time.Now()

	// Fetch fresh job from database to avoid race with caller
	var job models.ExportJob
	if err := s.db.First(&job, "id = ?", jobID).Error; err != nil {
		return
	}

	// Update job status to processing
	now := time.Now()
	job.Status = models.ExportJobStatusProcessing
	job.StartedAt = &now
	s.db.Save(&job)

	s.updateProgress(job.ID, models.ExportProgress{
		JobID:        job.ID,
		Status:       models.ExportJobStatusProcessing,
		CurrentPhase: "fetching",
		Message:      "Fetching data...",
	})

	// Fetch data based on export type
	data, totalRecords, err := s.fetchData(ctx, config)
	if err != nil {
		s.failJob(&job, fmt.Sprintf("Failed to fetch data: %v", err))
		return
	}

	job.TotalRecords = totalRecords
	s.db.Save(&job)

	s.updateProgress(job.ID, models.ExportProgress{
		JobID:        job.ID,
		Status:       models.ExportJobStatusProcessing,
		TotalRecords: totalRecords,
		CurrentPhase: "processing",
		Message:      fmt.Sprintf("Processing %d records...", totalRecords),
	})

	// Ensure output directory exists
	os.MkdirAll(filepath.Dir(job.FilePath), 0755)

	// Generate export based on format
	var exportData []byte
	switch config.Format {
	case models.ExportFormatCSV:
		exportData, err = s.ExportToCSV(ctx, data, config)
	case models.ExportFormatJSON:
		exportData, err = s.ExportToJSON(ctx, data, config)
	case models.ExportFormatExcel:
		exportData, err = s.ExportToExcel(ctx, data, config)
	case models.ExportFormatPDF:
		exportData, err = s.ExportToPDF(ctx, data, config)
	default:
		err = fmt.Errorf("unsupported export format: %s", config.Format)
	}

	if err != nil {
		s.failJob(&job, fmt.Sprintf("Failed to generate export: %v", err))
		return
	}

	s.updateProgress(job.ID, models.ExportProgress{
		JobID:            job.ID,
		Status:           models.ExportJobStatusProcessing,
		TotalRecords:     totalRecords,
		ProcessedRecords: totalRecords,
		Progress:         90,
		CurrentPhase:     "writing",
		Message:          "Writing file...",
	})

	// Write to file
	if err := os.WriteFile(job.FilePath, exportData, 0644); err != nil {
		s.failJob(&job, fmt.Sprintf("Failed to write file: %v", err))
		return
	}

	// Get file info
	fileInfo, err := os.Stat(job.FilePath)
	if err != nil {
		s.failJob(&job, fmt.Sprintf("Failed to get file info: %v", err))
		return
	}

	// Update job as completed
	completedAt := time.Now()
	expiresAt := completedAt.Add(24 * time.Hour) // Files expire after 24 hours
	job.Status = models.ExportJobStatusCompleted
	job.ProcessedRecords = totalRecords
	job.Progress = decimal.NewFromInt(100)
	job.FileSize = fileInfo.Size()
	job.DownloadURL = fmt.Sprintf("%s/api/v1/exports/%s/download", s.baseURL, job.ID.String())
	job.CompletedAt = &completedAt
	job.ExpiresAt = &expiresAt
	job.UpdatedAt = completedAt
	s.db.Save(&job)

	s.updateProgress(job.ID, models.ExportProgress{
		JobID:            job.ID,
		Status:           models.ExportJobStatusCompleted,
		TotalRecords:     totalRecords,
		ProcessedRecords: totalRecords,
		Progress:         100,
		CurrentPhase:     "completed",
		ElapsedTimeMs:    time.Since(startTime).Milliseconds(),
		Message:          "Export completed successfully",
	})
}

// fetchData fetches data based on export type
func (s *unifiedExportService) fetchData(ctx context.Context, config *models.ExportConfig) (interface{}, int, error) {
	switch config.ExportType {
	case models.ExportTypeCustomers:
		return s.fetchCustomers(ctx, config)
	case models.ExportTypeJobs:
		return s.fetchJobs(ctx, config)
	case models.ExportTypeInvoices:
		return s.fetchInvoices(ctx, config)
	case models.ExportTypeQuotes:
		return s.fetchQuotes(ctx, config)
	case models.ExportTypeRevenueReport:
		return s.fetchRevenueReport(ctx, config)
	case models.ExportTypeCustomerReport:
		return s.fetchCustomerReport(ctx, config)
	default:
		return nil, 0, fmt.Errorf("unsupported export type: %s", config.ExportType)
	}
}

func (s *unifiedExportService) fetchCustomers(ctx context.Context, config *models.ExportConfig) (interface{}, int, error) {
	var customers []models.Customer
	query := s.db.WithContext(ctx).Where("deleted_at IS NULL")

	// Apply filters from config.Query if provided
	if config.Query != nil {
		if q, ok := config.Query.(map[string]interface{}); ok {
			if customerType, exists := q["customer_type"]; exists && customerType != "" {
				query = query.Where("customer_type = ?", customerType)
			}
			if status, exists := q["status"]; exists && status != "" {
				query = query.Where("status = ?", status)
			}
			if state, exists := q["state"]; exists && state != "" {
				query = query.Where("billing_address_state = ?", state)
			}
		}
	}

	var count int64
	if err := query.Model(&models.Customer{}).Count(&count).Error; err != nil {
		return nil, 0, err
	}

	if err := query.Order("created_at DESC").Find(&customers).Error; err != nil {
		return nil, 0, err
	}

	return customers, int(count), nil
}

func (s *unifiedExportService) fetchJobs(ctx context.Context, config *models.ExportConfig) (interface{}, int, error) {
	var jobs []models.Job
	query := s.db.WithContext(ctx).Where("deleted_at IS NULL")

	var count int64
	if err := query.Model(&models.Job{}).Count(&count).Error; err != nil {
		return nil, 0, err
	}

	if err := query.Order("created_at DESC").Find(&jobs).Error; err != nil {
		return nil, 0, err
	}

	return jobs, int(count), nil
}

func (s *unifiedExportService) fetchInvoices(ctx context.Context, config *models.ExportConfig) (interface{}, int, error) {
	var invoices []models.Invoice
	query := s.db.WithContext(ctx)

	var count int64
	if err := query.Model(&models.Invoice{}).Count(&count).Error; err != nil {
		return nil, 0, err
	}

	if err := query.Order("created_at DESC").Find(&invoices).Error; err != nil {
		return nil, 0, err
	}

	return invoices, int(count), nil
}

func (s *unifiedExportService) fetchQuotes(ctx context.Context, config *models.ExportConfig) (interface{}, int, error) {
	var quotes []models.Quote
	query := s.db.WithContext(ctx).Where("deleted_at IS NULL")

	var count int64
	if err := query.Model(&models.Quote{}).Count(&count).Error; err != nil {
		return nil, 0, err
	}

	if err := query.Order("created_at DESC").Find(&quotes).Error; err != nil {
		return nil, 0, err
	}

	return quotes, int(count), nil
}

func (s *unifiedExportService) fetchRevenueReport(ctx context.Context, config *models.ExportConfig) (interface{}, int, error) {
	// Revenue reports return aggregated data
	// Return a placeholder - actual implementation would use RevenueReportService
	return map[string]interface{}{
		"type":    "revenue_report",
		"message": "Revenue report data",
	}, 1, nil
}

func (s *unifiedExportService) fetchCustomerReport(ctx context.Context, config *models.ExportConfig) (interface{}, int, error) {
	// Customer reports return aggregated data
	return map[string]interface{}{
		"type":    "customer_report",
		"message": "Customer report data",
	}, 1, nil
}

// ExportToCSV exports data to CSV format
func (s *unifiedExportService) ExportToCSV(ctx context.Context, data interface{}, config *models.ExportConfig) ([]byte, error) {
	var buf bytes.Buffer

	// Apply CSV options
	opts := config.CSVOptions
	if opts == nil {
		opts = models.DefaultCSVOptions()
	}

	// Add UTF-8 BOM if requested
	if opts.BOM {
		buf.Write([]byte{0xEF, 0xBB, 0xBF})
	}

	writer := csv.NewWriter(&buf)

	// Set delimiter
	if opts.Delimiter != "" {
		switch opts.Delimiter {
		case "tab":
			writer.Comma = '\t'
		case "semicolon":
			writer.Comma = ';'
		default:
			writer.Comma = ','
		}
	}

	// Process data based on type
	switch d := data.(type) {
	case []models.Customer:
		if err := s.writeCustomersCSV(writer, d, opts, config.Fields); err != nil {
			return nil, err
		}
	case []models.Job:
		if err := s.writeJobsCSV(writer, d, opts, config.Fields); err != nil {
			return nil, err
		}
	case []models.Invoice:
		if err := s.writeInvoicesCSV(writer, d, opts, config.Fields); err != nil {
			return nil, err
		}
	case []models.Quote:
		if err := s.writeQuotesCSV(writer, d, opts, config.Fields); err != nil {
			return nil, err
		}
	default:
		// Generic JSON fallback for complex types
		jsonData, err := json.Marshal(data)
		if err != nil {
			return nil, err
		}
		return jsonData, nil
	}

	writer.Flush()
	if err := writer.Error(); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

func (s *unifiedExportService) writeCustomersCSV(writer *csv.Writer, customers []models.Customer, opts *models.CSVExportOptions, fields []string) error {
	// Define all available fields
	allFields := []string{
		"id", "first_name", "last_name", "company_name", "email",
		"phone_primary", "phone_secondary", "customer_type", "status",
		"billing_address_street", "billing_address_city", "billing_address_state", "billing_address_zip",
		"service_address_street", "service_address_city", "service_address_state", "service_address_zip",
		"notes", "created_at", "updated_at",
	}

	// Use selected fields or all fields
	selectedFields := allFields
	if len(fields) > 0 {
		selectedFields = fields
	}

	// Write header
	if opts.IncludeHeader {
		if err := writer.Write(selectedFields); err != nil {
			return err
		}
	}

	// Write data rows
	for _, customer := range customers {
		row := s.customerToRow(customer, selectedFields)
		if err := writer.Write(row); err != nil {
			return err
		}
	}

	return nil
}

func (s *unifiedExportService) customerToRow(customer models.Customer, fields []string) []string {
	fieldValues := map[string]string{
		"id":                     customer.ID.String(),
		"first_name":             customer.FirstName,
		"last_name":              customer.LastName,
		"company_name":           ptrToStr(customer.CompanyName),
		"email":                  customer.Email,
		"phone_primary":          customer.PhonePrimary,
		"phone_secondary":        ptrToStr(customer.PhoneSecondary),
		"customer_type":          string(customer.CustomerType),
		"status":                 string(customer.Status),
		"billing_address_street": customer.BillingAddressStreet,
		"billing_address_city":   customer.BillingAddressCity,
		"billing_address_state":  customer.BillingAddressState,
		"billing_address_zip":    customer.BillingAddressZip,
		"service_address_street": ptrToStr(customer.ServiceAddressStreet),
		"service_address_city":   ptrToStr(customer.ServiceAddressCity),
		"service_address_state":  ptrToStr(customer.ServiceAddressState),
		"service_address_zip":    ptrToStr(customer.ServiceAddressZip),
		"notes":                  ptrToStr(customer.Notes),
		"created_at":             customer.CreatedAt.Format(time.RFC3339),
		"updated_at":             customer.UpdatedAt.Format(time.RFC3339),
	}

	row := make([]string, len(fields))
	for i, field := range fields {
		row[i] = fieldValues[field]
	}
	return row
}

func (s *unifiedExportService) writeJobsCSV(writer *csv.Writer, jobs []models.Job, opts *models.CSVExportOptions, fields []string) error {
	headers := []string{"id", "customer_id", "title", "description", "status", "scheduled_date", "created_at"}
	if opts.IncludeHeader {
		if err := writer.Write(headers); err != nil {
			return err
		}
	}

	for _, job := range jobs {
		row := []string{
			job.ID.String(),
			job.CustomerID.String(),
			job.Title,
			job.Description,
			string(job.Status),
			formatEpochPtr(job.ScheduledStartAt),
			job.CreatedAt.Format(time.RFC3339),
		}
		if err := writer.Write(row); err != nil {
			return err
		}
	}
	return nil
}

func (s *unifiedExportService) writeInvoicesCSV(writer *csv.Writer, invoices []models.Invoice, opts *models.CSVExportOptions, fields []string) error {
	headers := []string{"id", "invoice_number", "customer_id", "status", "subtotal", "tax", "total", "due_date", "created_at"}
	if opts.IncludeHeader {
		if err := writer.Write(headers); err != nil {
			return err
		}
	}

	for _, inv := range invoices {
		row := []string{
			inv.ID.String(),
			inv.InvoiceNumber,
			inv.CustomerID.String(),
			string(inv.Status),
			inv.Subtotal.String(),
			inv.TaxAmount.String(),
			inv.TotalAmount.String(),
			inv.DueDate.Format(time.RFC3339),
			inv.CreatedAt.Format(time.RFC3339),
		}
		if err := writer.Write(row); err != nil {
			return err
		}
	}
	return nil
}

func (s *unifiedExportService) writeQuotesCSV(writer *csv.Writer, quotes []models.Quote, opts *models.CSVExportOptions, fields []string) error {
	headers := []string{"id", "quote_number", "customer_id", "status", "total", "valid_until", "created_at"}
	if opts.IncludeHeader {
		if err := writer.Write(headers); err != nil {
			return err
		}
	}

	for _, quote := range quotes {
		row := []string{
			quote.ID.String(),
			quote.QuoteNumber,
			quote.CustomerID.String(),
			string(quote.Status),
			quote.Total.String(),
			quote.ValidUntil.Format(time.RFC3339),
			quote.CreatedAt.Format(time.RFC3339),
		}
		if err := writer.Write(row); err != nil {
			return err
		}
	}
	return nil
}

// ExportToJSON exports data to JSON format
func (s *unifiedExportService) ExportToJSON(ctx context.Context, data interface{}, config *models.ExportConfig) ([]byte, error) {
	var buf bytes.Buffer
	encoder := json.NewEncoder(&buf)
	encoder.SetIndent("", "  ")
	encoder.SetEscapeHTML(false)

	if err := encoder.Encode(data); err != nil {
		return nil, fmt.Errorf("failed to encode JSON: %w", err)
	}

	return buf.Bytes(), nil
}

// ExportToExcel exports data to Excel format
func (s *unifiedExportService) ExportToExcel(ctx context.Context, data interface{}, config *models.ExportConfig) ([]byte, error) {
	f := excelize.NewFile()

	opts := config.ExcelOptions
	if opts == nil {
		opts = models.DefaultExcelOptions()
	}

	sheetName := opts.SheetName
	if sheetName == "" {
		sheetName = "Export"
	}

	// Rename default sheet
	f.SetSheetName("Sheet1", sheetName)

	// Write data based on type
	switch d := data.(type) {
	case []models.Customer:
		if err := s.writeCustomersExcel(f, sheetName, d, opts); err != nil {
			return nil, err
		}
	case []models.Job:
		if err := s.writeJobsExcel(f, sheetName, d, opts); err != nil {
			return nil, err
		}
	case []models.Invoice:
		if err := s.writeInvoicesExcel(f, sheetName, d, opts); err != nil {
			return nil, err
		}
	case []models.Quote:
		if err := s.writeQuotesExcel(f, sheetName, d, opts); err != nil {
			return nil, err
		}
	default:
		// Generic handling - convert to JSON and write as text
		jsonData, _ := json.MarshalIndent(data, "", "  ")
		f.SetCellValue(sheetName, "A1", string(jsonData))
	}

	// Write to buffer
	var buf bytes.Buffer
	if err := f.Write(&buf); err != nil {
		return nil, fmt.Errorf("failed to write Excel file: %w", err)
	}

	return buf.Bytes(), nil
}

func (s *unifiedExportService) writeCustomersExcel(f *excelize.File, sheet string, customers []models.Customer, opts *models.ExcelExportOptions) error {
	headers := []string{"ID", "First Name", "Last Name", "Company", "Email", "Phone", "Type", "Status", "City", "State", "Created"}

	// Write headers
	for i, header := range headers {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		f.SetCellValue(sheet, cell, header)
	}

	// Style headers
	headerStyle, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true},
		Fill:      excelize.Fill{Type: "pattern", Color: []string{"#E0E0E0"}, Pattern: 1},
		Alignment: &excelize.Alignment{Horizontal: "center"},
	})
	f.SetCellStyle(sheet, "A1", fmt.Sprintf("%s1", string(rune('A'+len(headers)-1))), headerStyle)

	// Write data
	for i, customer := range customers {
		row := i + 2
		f.SetCellValue(sheet, fmt.Sprintf("A%d", row), customer.ID.String())
		f.SetCellValue(sheet, fmt.Sprintf("B%d", row), customer.FirstName)
		f.SetCellValue(sheet, fmt.Sprintf("C%d", row), customer.LastName)
		f.SetCellValue(sheet, fmt.Sprintf("D%d", row), ptrToStr(customer.CompanyName))
		f.SetCellValue(sheet, fmt.Sprintf("E%d", row), customer.Email)
		f.SetCellValue(sheet, fmt.Sprintf("F%d", row), customer.PhonePrimary)
		f.SetCellValue(sheet, fmt.Sprintf("G%d", row), string(customer.CustomerType))
		f.SetCellValue(sheet, fmt.Sprintf("H%d", row), string(customer.Status))
		f.SetCellValue(sheet, fmt.Sprintf("I%d", row), customer.BillingAddressCity)
		f.SetCellValue(sheet, fmt.Sprintf("J%d", row), customer.BillingAddressState)
		f.SetCellValue(sheet, fmt.Sprintf("K%d", row), customer.CreatedAt.Format("2006-01-02"))
	}

	// Freeze header row
	if opts.FreezePanes {
		f.SetPanes(sheet, &excelize.Panes{
			Freeze:      true,
			Split:       false,
			XSplit:      0,
			YSplit:      1,
			TopLeftCell: "A2",
			ActivePane:  "bottomLeft",
		})
	}

	// Auto filter
	if opts.AutoFilter && len(customers) > 0 {
		lastCol := string(rune('A' + len(headers) - 1))
		f.AutoFilter(sheet, fmt.Sprintf("A1:%s%d", lastCol, len(customers)+1), nil)
	}

	return nil
}

func (s *unifiedExportService) writeJobsExcel(f *excelize.File, sheet string, jobs []models.Job, opts *models.ExcelExportOptions) error {
	headers := []string{"ID", "Customer ID", "Title", "Status", "Scheduled Date", "Created"}

	for i, header := range headers {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		f.SetCellValue(sheet, cell, header)
	}

	for i, job := range jobs {
		row := i + 2
		f.SetCellValue(sheet, fmt.Sprintf("A%d", row), job.ID.String())
		f.SetCellValue(sheet, fmt.Sprintf("B%d", row), job.CustomerID.String())
		f.SetCellValue(sheet, fmt.Sprintf("C%d", row), job.Title)
		f.SetCellValue(sheet, fmt.Sprintf("D%d", row), string(job.Status))
		f.SetCellValue(sheet, fmt.Sprintf("E%d", row), formatEpochPtr(job.ScheduledStartAt))
		f.SetCellValue(sheet, fmt.Sprintf("F%d", row), job.CreatedAt.Format("2006-01-02"))
	}

	return nil
}

func (s *unifiedExportService) writeInvoicesExcel(f *excelize.File, sheet string, invoices []models.Invoice, opts *models.ExcelExportOptions) error {
	headers := []string{"ID", "Invoice #", "Customer ID", "Status", "Subtotal", "Tax", "Total", "Due Date", "Created"}

	for i, header := range headers {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		f.SetCellValue(sheet, cell, header)
	}

	for i, inv := range invoices {
		row := i + 2
		f.SetCellValue(sheet, fmt.Sprintf("A%d", row), inv.ID.String())
		f.SetCellValue(sheet, fmt.Sprintf("B%d", row), inv.InvoiceNumber)
		f.SetCellValue(sheet, fmt.Sprintf("C%d", row), inv.CustomerID.String())
		f.SetCellValue(sheet, fmt.Sprintf("D%d", row), string(inv.Status))
		subtotal, _ := inv.Subtotal.Float64()
		f.SetCellValue(sheet, fmt.Sprintf("E%d", row), subtotal)
		tax, _ := inv.TaxAmount.Float64()
		f.SetCellValue(sheet, fmt.Sprintf("F%d", row), tax)
		total, _ := inv.TotalAmount.Float64()
		f.SetCellValue(sheet, fmt.Sprintf("G%d", row), total)
		f.SetCellValue(sheet, fmt.Sprintf("H%d", row), inv.DueDate.Format("2006-01-02"))
		f.SetCellValue(sheet, fmt.Sprintf("I%d", row), inv.CreatedAt.Format("2006-01-02"))
	}

	return nil
}

func (s *unifiedExportService) writeQuotesExcel(f *excelize.File, sheet string, quotes []models.Quote, opts *models.ExcelExportOptions) error {
	headers := []string{"ID", "Quote #", "Customer ID", "Status", "Total", "Valid Until", "Created"}

	for i, header := range headers {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		f.SetCellValue(sheet, cell, header)
	}

	for i, quote := range quotes {
		row := i + 2
		f.SetCellValue(sheet, fmt.Sprintf("A%d", row), quote.ID.String())
		f.SetCellValue(sheet, fmt.Sprintf("B%d", row), quote.QuoteNumber)
		f.SetCellValue(sheet, fmt.Sprintf("C%d", row), quote.CustomerID.String())
		f.SetCellValue(sheet, fmt.Sprintf("D%d", row), string(quote.Status))
		total, _ := quote.Total.Float64()
		f.SetCellValue(sheet, fmt.Sprintf("E%d", row), total)
		f.SetCellValue(sheet, fmt.Sprintf("F%d", row), quote.ValidUntil.Format("2006-01-02"))
		f.SetCellValue(sheet, fmt.Sprintf("G%d", row), quote.CreatedAt.Format("2006-01-02"))
	}

	return nil
}

// ExportToPDF exports data to PDF format
func (s *unifiedExportService) ExportToPDF(ctx context.Context, data interface{}, config *models.ExportConfig) ([]byte, error) {
	opts := config.PDFOptions
	if opts == nil {
		opts = models.DefaultPDFOptions()
	}

	// Build HTML content based on data type
	htmlContent := s.buildPDFHTML(data, config, opts)

	// Generate PDF using PDF service
	tempFile := filepath.Join(os.TempDir(), fmt.Sprintf("export_%s.pdf", uuid.New().String()))
	defer os.Remove(tempFile)

	pdfOpts := &PDFGenerationOptions{
		PageSize:        models.PageSize(opts.PageSize),
		PageOrientation: models.PageOrientation(opts.Orientation),
		MarginTop:       "15mm",
		MarginRight:     "10mm",
		MarginBottom:    "15mm",
		MarginLeft:      "10mm",
		DPI:             150,
		ImageQuality:    85,
	}

	if opts.ShowPageNumbers {
		pdfOpts.FooterHTML = `<div style="text-align: center; font-size: 10px; color: #666;">Page [page] of [topage]</div>`
	}

	_, err := s.pdfService.GeneratePDF(ctx, htmlContent, tempFile, pdfOpts)
	if err != nil {
		return nil, fmt.Errorf("failed to generate PDF: %w", err)
	}

	return os.ReadFile(tempFile)
}

func (s *unifiedExportService) buildPDFHTML(data interface{}, config *models.ExportConfig, opts *models.PDFExportOptions) string {
	var html strings.Builder

	html.WriteString(`<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <style>
        body { font-family: Arial, sans-serif; font-size: 12px; margin: 0; padding: 20px; }
        h1 { color: #333; border-bottom: 2px solid #4a90d9; padding-bottom: 10px; }
        table { width: 100%; border-collapse: collapse; margin-top: 20px; }
        th, td { padding: 8px; text-align: left; border-bottom: 1px solid #ddd; }
        th { background-color: #4a90d9; color: white; }
        tr:nth-child(even) { background-color: #f9f9f9; }
        .summary { background: #f5f5f5; padding: 15px; margin-bottom: 20px; border-radius: 5px; }
        .summary-item { display: inline-block; margin-right: 30px; }
        .footer { margin-top: 30px; text-align: center; color: #666; font-size: 10px; }
    </style>
</head>
<body>`)

	// Add title
	title := opts.Title
	if title == "" {
		title = fmt.Sprintf("%s Export", strings.Title(string(config.ExportType)))
	}
	html.WriteString(fmt.Sprintf("<h1>%s</h1>", title))

	// Add header text if provided
	if opts.HeaderText != "" {
		html.WriteString(fmt.Sprintf("<p>%s</p>", opts.HeaderText))
	}

	// Build table based on data type
	switch d := data.(type) {
	case []models.Customer:
		html.WriteString(s.buildCustomersPDFTable(d))
	case []models.Job:
		html.WriteString(s.buildJobsPDFTable(d))
	case []models.Invoice:
		html.WriteString(s.buildInvoicesPDFTable(d))
	case []models.Quote:
		html.WriteString(s.buildQuotesPDFTable(d))
	default:
		jsonData, _ := json.MarshalIndent(data, "", "  ")
		html.WriteString(fmt.Sprintf("<pre>%s</pre>", string(jsonData)))
	}

	// Add footer
	html.WriteString(fmt.Sprintf(`<div class="footer">Generated on %s</div>`, time.Now().Format("January 2, 2006 3:04 PM")))
	html.WriteString("</body></html>")

	return html.String()
}

func (s *unifiedExportService) buildCustomersPDFTable(customers []models.Customer) string {
	var html strings.Builder
	html.WriteString(fmt.Sprintf(`<div class="summary"><span class="summary-item"><strong>Total Customers:</strong> %d</span></div>`, len(customers)))
	html.WriteString(`<table><thead><tr><th>Name</th><th>Company</th><th>Email</th><th>Phone</th><th>Type</th><th>Status</th><th>City</th><th>State</th></tr></thead><tbody>`)

	for _, c := range customers {
		html.WriteString(fmt.Sprintf("<tr><td>%s %s</td><td>%s</td><td>%s</td><td>%s</td><td>%s</td><td>%s</td><td>%s</td><td>%s</td></tr>",
			c.FirstName, c.LastName, ptrToStr(c.CompanyName), c.Email, c.PhonePrimary,
			c.CustomerType, c.Status, c.BillingAddressCity, c.BillingAddressState))
	}

	html.WriteString("</tbody></table>")
	return html.String()
}

func (s *unifiedExportService) buildJobsPDFTable(jobs []models.Job) string {
	var html strings.Builder
	html.WriteString(fmt.Sprintf(`<div class="summary"><span class="summary-item"><strong>Total Jobs:</strong> %d</span></div>`, len(jobs)))
	html.WriteString(`<table><thead><tr><th>Title</th><th>Status</th><th>Scheduled</th><th>Created</th></tr></thead><tbody>`)

	for _, j := range jobs {
		html.WriteString(fmt.Sprintf("<tr><td>%s</td><td>%s</td><td>%s</td><td>%s</td></tr>",
			j.Title, j.Status, formatEpochPtr(j.ScheduledStartAt), j.CreatedAt.Format("2006-01-02")))
	}

	html.WriteString("</tbody></table>")
	return html.String()
}

func (s *unifiedExportService) buildInvoicesPDFTable(invoices []models.Invoice) string {
	var html strings.Builder
	html.WriteString(fmt.Sprintf(`<div class="summary"><span class="summary-item"><strong>Total Invoices:</strong> %d</span></div>`, len(invoices)))
	html.WriteString(`<table><thead><tr><th>Invoice #</th><th>Status</th><th>Subtotal</th><th>Tax</th><th>Total</th><th>Due Date</th></tr></thead><tbody>`)

	for _, inv := range invoices {
		html.WriteString(fmt.Sprintf("<tr><td>%s</td><td>%s</td><td>$%s</td><td>$%s</td><td>$%s</td><td>%s</td></tr>",
			inv.InvoiceNumber, inv.Status, inv.Subtotal.StringFixed(2), inv.TaxAmount.StringFixed(2),
			inv.TotalAmount.StringFixed(2), inv.DueDate.Format("2006-01-02")))
	}

	html.WriteString("</tbody></table>")
	return html.String()
}

func (s *unifiedExportService) buildQuotesPDFTable(quotes []models.Quote) string {
	var html strings.Builder
	html.WriteString(fmt.Sprintf(`<div class="summary"><span class="summary-item"><strong>Total Quotes:</strong> %d</span></div>`, len(quotes)))
	html.WriteString(`<table><thead><tr><th>Quote #</th><th>Status</th><th>Total</th><th>Valid Until</th><th>Created</th></tr></thead><tbody>`)

	for _, q := range quotes {
		html.WriteString(fmt.Sprintf("<tr><td>%s</td><td>%s</td><td>$%s</td><td>%s</td><td>%s</td></tr>",
			q.QuoteNumber, q.Status, q.Total.StringFixed(2), q.ValidUntil.Format("2006-01-02"), q.CreatedAt.Format("2006-01-02")))
	}

	html.WriteString("</tbody></table>")
	return html.String()
}

// GetExportStatus returns the status of an export job
func (s *unifiedExportService) GetExportStatus(ctx context.Context, jobID uuid.UUID) (*models.ExportJobResponse, error) {
	var job models.ExportJob
	if err := s.db.First(&job, "id = ?", jobID).Error; err != nil {
		return nil, err
	}

	response := job.ToResponse()
	return &response, nil
}

// CancelExport cancels an in-progress export job
func (s *unifiedExportService) CancelExport(ctx context.Context, jobID uuid.UUID) error {
	result := s.db.Model(&models.ExportJob{}).
		Where("id = ? AND status IN ?", jobID, []string{string(models.ExportJobStatusPending), string(models.ExportJobStatusProcessing)}).
		Update("status", models.ExportJobStatusCancelled)

	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("export job not found or cannot be cancelled")
	}

	s.updateProgress(jobID, models.ExportProgress{
		JobID:   jobID,
		Status:  models.ExportJobStatusCancelled,
		Message: "Export cancelled by user",
	})

	return nil
}

// GetExportFile returns the file for download
func (s *unifiedExportService) GetExportFile(ctx context.Context, jobID uuid.UUID) (io.ReadCloser, *models.ExportJob, error) {
	var job models.ExportJob
	if err := s.db.First(&job, "id = ?", jobID).Error; err != nil {
		return nil, nil, err
	}

	if job.Status != models.ExportJobStatusCompleted {
		return nil, nil, fmt.Errorf("export not ready for download")
	}

	file, err := os.Open(job.FilePath)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to open export file: %w", err)
	}

	return file, &job, nil
}

// ListExports lists recent exports for a user
func (s *unifiedExportService) ListExports(ctx context.Context, userID uuid.UUID, limit int) ([]models.ExportJobResponse, error) {
	var jobs []models.ExportJob
	if err := s.db.Where("user_id = ?", userID).Order("created_at DESC").Limit(limit).Find(&jobs).Error; err != nil {
		return nil, err
	}

	responses := make([]models.ExportJobResponse, len(jobs))
	for i, job := range jobs {
		responses[i] = job.ToResponse()
	}

	return responses, nil
}

// CleanupExpiredExports removes expired export files
func (s *unifiedExportService) CleanupExpiredExports(ctx context.Context) error {
	var jobs []models.ExportJob
	if err := s.db.Where("expires_at < ? AND status = ?", time.Now(), models.ExportJobStatusCompleted).Find(&jobs).Error; err != nil {
		return err
	}

	for _, job := range jobs {
		// Remove file
		if job.FilePath != "" {
			os.RemoveAll(filepath.Dir(job.FilePath))
		}

		// Delete job record
		s.db.Delete(&job)
	}

	return nil
}

// Progress tracking methods
func (s *unifiedExportService) initProgress(jobID uuid.UUID) {
	s.progressMutex.Lock()
	defer s.progressMutex.Unlock()

	s.progressMap[jobID] = &models.ExportProgress{
		JobID:  jobID,
		Status: models.ExportJobStatusPending,
	}
}

func (s *unifiedExportService) updateProgress(jobID uuid.UUID, progress models.ExportProgress) {
	s.progressMutex.Lock()
	s.progressMap[jobID] = &progress
	s.progressMutex.Unlock()

	// Notify subscribers
	s.subMutex.RLock()
	subs := s.subscribers[jobID]
	s.subMutex.RUnlock()

	for _, ch := range subs {
		select {
		case ch <- progress:
		default:
			// Channel full, skip
		}
	}
}

func (s *unifiedExportService) GetProgress(jobID uuid.UUID) *models.ExportProgress {
	s.progressMutex.RLock()
	defer s.progressMutex.RUnlock()
	return s.progressMap[jobID]
}

func (s *unifiedExportService) SubscribeToProgress(jobID uuid.UUID) <-chan models.ExportProgress {
	ch := make(chan models.ExportProgress, 10)

	s.subMutex.Lock()
	s.subscribers[jobID] = append(s.subscribers[jobID], ch)
	s.subMutex.Unlock()

	return ch
}

func (s *unifiedExportService) failJob(job *models.ExportJob, errorMsg string) {
	job.Status = models.ExportJobStatusFailed
	job.ErrorMessage = &errorMsg
	completedAt := time.Now()
	job.CompletedAt = &completedAt
	s.db.Save(job)

	s.updateProgress(job.ID, models.ExportProgress{
		JobID:   job.ID,
		Status:  models.ExportJobStatusFailed,
		Message: errorMsg,
	})
}

func (s *unifiedExportService) validateConfig(config *models.ExportConfig) error {
	if config.Format == "" {
		return fmt.Errorf("export format is required")
	}
	if config.ExportType == "" {
		return fmt.Errorf("export type is required")
	}
	return nil
}

// Helper functions
func ptrToStr(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func formatEpochPtr(t *int64) string {
	if t == nil {
		return ""
	}
	return epoch.EpochToTime(*t).Format("2006-01-02")
}
