package services

import (
	"bytes"
	"context"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	apperrors "github.com/javaknight1/servicepro/backend/internal/errors"
	"github.com/javaknight1/servicepro/backend/internal/services/pdf"
	csvsafe "github.com/javaknight1/servicepro/backend/pkg/csv"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/xuri/excelize/v2"
	"gorm.io/gorm"

	"github.com/javaknight1/servicepro/backend/internal/models"
	"github.com/javaknight1/servicepro/backend/internal/tenancy"
	"github.com/javaknight1/servicepro/backend/internal/utils/epoch"
	"github.com/javaknight1/servicepro/backend/pkg/clients/logging"
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

	// Progress tracking (persisted on the export_jobs row, so any instance
	// can answer)
	GetProgress(ctx context.Context, jobID uuid.UUID) (*models.ExportProgress, error)
}

type unifiedExportService struct {
	db      *gorm.DB
	pdfGen  *pdf.Generator
	files   ExportFileStore
	baseURL string
}

// exportTimeout bounds one export run.
const exportTimeout = 2 * time.Minute

// NewUnifiedExportService creates a new unified export service
//
// files is where finished exports are stored: NewObjectExportStore in any
// deployed environment (T128), NewLocalExportStore for development and tests.
func NewUnifiedExportService(db *gorm.DB, pdfGen *pdf.Generator, files ExportFileStore, baseURL string) UnifiedExportService {
	if files == nil {
		files = NewLocalExportStore("./exports")
	}
	return &unifiedExportService{
		db:      db,
		pdfGen:  pdfGen,
		files:   files,
		baseURL: baseURL,
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
	// FilePath holds the storage key: exports/<tenant>/<job>/<file>.
	tenantID, err := tenancy.MustTenantID(ctx)
	if err != nil {
		return nil, err
	}
	job.FilePath = fmt.Sprintf("exports/%s/%s/%s", tenantID, job.ID, job.Filename)
	job.MimeType = config.Format.GetMimeType()

	// Save job to database
	if err := s.db.WithContext(ctx).Create(job).Error; err != nil {
		return nil, fmt.Errorf("failed to create export job: %w", err)
	}

	// Run the export inside the request (T128). A detached goroutine is
	// CPU-throttled on Cloud Run once the response is sent and is lost on
	// scale-down. WithoutCancel keeps a client disconnect from leaving a
	// half-written job; the tenant in ctx still scopes every query.
	runCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), exportTimeout)
	defer cancel()
	s.processExport(runCtx, job.ID, config)

	var done models.ExportJob
	if err := s.db.WithContext(ctx).First(&done, "id = ?", job.ID).Error; err != nil {
		return nil, fmt.Errorf("failed to load export job: %w", err)
	}
	return &done, nil
}

// setProgress persists progress fields on the job row, but only while the job
// is still pending/processing (a cancelled job stays cancelled).
func (s *unifiedExportService) setProgress(ctx context.Context, jobID uuid.UUID, fields map[string]interface{}) error {
	fields["updated_at"] = time.Now()
	res := s.db.WithContext(ctx).Model(&models.ExportJob{}).
		Where("id = ? AND status IN ?", jobID, []models.ExportJobStatus{models.ExportJobStatusPending, models.ExportJobStatusProcessing}).
		Updates(fields)
	if res.Error != nil {
		return fmt.Errorf("save export progress: %w", res.Error)
	}
	if res.RowsAffected == 0 {
		return errExportNotActive
	}
	return nil
}

var errExportNotActive = fmt.Errorf("export job is no longer active")

// processExport generates the export file and records progress on the job row.
func (s *unifiedExportService) processExport(ctx context.Context, jobID uuid.UUID, config *models.ExportConfig) {
	var job models.ExportJob
	if err := s.db.WithContext(ctx).First(&job, "id = ?", jobID).Error; err != nil {
		return
	}

	now := time.Now()
	if err := s.setProgress(ctx, jobID, map[string]interface{}{
		"status": models.ExportJobStatusProcessing, "started_at": now,
	}); err != nil {
		s.failJob(ctx, &job, "Failed to start export")
		return
	}

	data, totalRecords, err := s.fetchData(ctx, config)
	if err != nil {
		s.failJob(ctx, &job, fmt.Sprintf("Failed to fetch data: %v", err))
		return
	}
	if err := s.setProgress(ctx, jobID, map[string]interface{}{
		"total_records": totalRecords, "progress": decimal.NewFromInt(30),
	}); err != nil {
		s.failJob(ctx, &job, "Export was cancelled or could not be saved")
		return
	}

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
		s.failJob(ctx, &job, fmt.Sprintf("Failed to generate export: %v", err))
		return
	}
	if err := s.setProgress(ctx, jobID, map[string]interface{}{
		"processed_records": totalRecords, "progress": decimal.NewFromInt(90),
	}); err != nil {
		s.failJob(ctx, &job, "Export was cancelled or could not be saved")
		return
	}

	if err := s.files.Put(ctx, job.FilePath, exportData, job.MimeType); err != nil {
		s.failJob(ctx, &job, "Failed to store export file")
		return
	}

	completedAt := time.Now()
	expiresAt := completedAt.Add(24 * time.Hour) // Files expire after 24 hours
	if err := s.setProgress(ctx, jobID, map[string]interface{}{
		"status":            models.ExportJobStatusCompleted,
		"processed_records": totalRecords,
		"progress":          decimal.NewFromInt(100),
		"file_size":         int64(len(exportData)),
		"download_url":      fmt.Sprintf("%s/api/v1/exports/%s/download", s.baseURL, jobID.String()),
		"completed_at":      completedAt,
		"expires_at":        expiresAt,
	}); err != nil {
		s.failJob(ctx, &job, "Export was cancelled or could not be saved")
	}
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
		if err := writer.Write(csvsafe.SafeRow(row)); err != nil {
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
		if err := writer.Write(csvsafe.SafeRow(row)); err != nil {
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
		if err := writer.Write(csvsafe.SafeRow(row)); err != nil {
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
		if err := writer.Write(csvsafe.SafeRow(row)); err != nil {
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
	f.SetCellStyle(sheet, "A1", fmt.Sprintf("%s1", string(rune('A'+len(headers)-1))), headerStyle) //nolint:gosec // G115: headers is a fixed list of 11 columns

	// Write data
	for i, customer := range customers {
		row := i + 2
		f.SetCellValue(sheet, fmt.Sprintf("A%d", row), csvsafe.SafeCell(customer.ID.String()))
		f.SetCellValue(sheet, fmt.Sprintf("B%d", row), csvsafe.SafeCell(customer.FirstName))
		f.SetCellValue(sheet, fmt.Sprintf("C%d", row), csvsafe.SafeCell(customer.LastName))
		f.SetCellValue(sheet, fmt.Sprintf("D%d", row), csvsafe.SafeCell(ptrToStr(customer.CompanyName)))
		f.SetCellValue(sheet, fmt.Sprintf("E%d", row), csvsafe.SafeCell(customer.Email))
		f.SetCellValue(sheet, fmt.Sprintf("F%d", row), csvsafe.SafeCell(customer.PhonePrimary))
		f.SetCellValue(sheet, fmt.Sprintf("G%d", row), csvsafe.SafeCell(string(customer.CustomerType)))
		f.SetCellValue(sheet, fmt.Sprintf("H%d", row), csvsafe.SafeCell(string(customer.Status)))
		f.SetCellValue(sheet, fmt.Sprintf("I%d", row), csvsafe.SafeCell(customer.BillingAddressCity))
		f.SetCellValue(sheet, fmt.Sprintf("J%d", row), csvsafe.SafeCell(customer.BillingAddressState))
		f.SetCellValue(sheet, fmt.Sprintf("K%d", row), csvsafe.SafeCell(customer.CreatedAt.Format("2006-01-02")))
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
		lastCol := string(rune('A' + len(headers) - 1)) //nolint:gosec // G115: headers is a fixed list of 11 columns
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
		f.SetCellValue(sheet, fmt.Sprintf("A%d", row), csvsafe.SafeCell(job.ID.String()))
		f.SetCellValue(sheet, fmt.Sprintf("B%d", row), csvsafe.SafeCell(job.CustomerID.String()))
		f.SetCellValue(sheet, fmt.Sprintf("C%d", row), csvsafe.SafeCell(job.Title))
		f.SetCellValue(sheet, fmt.Sprintf("D%d", row), csvsafe.SafeCell(string(job.Status)))
		f.SetCellValue(sheet, fmt.Sprintf("E%d", row), csvsafe.SafeCell(formatEpochPtr(job.ScheduledStartAt)))
		f.SetCellValue(sheet, fmt.Sprintf("F%d", row), csvsafe.SafeCell(job.CreatedAt.Format("2006-01-02")))
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
		f.SetCellValue(sheet, fmt.Sprintf("A%d", row), csvsafe.SafeCell(inv.ID.String()))
		f.SetCellValue(sheet, fmt.Sprintf("B%d", row), csvsafe.SafeCell(inv.InvoiceNumber))
		f.SetCellValue(sheet, fmt.Sprintf("C%d", row), csvsafe.SafeCell(inv.CustomerID.String()))
		f.SetCellValue(sheet, fmt.Sprintf("D%d", row), csvsafe.SafeCell(string(inv.Status)))
		subtotal, _ := inv.Subtotal.Float64()
		f.SetCellValue(sheet, fmt.Sprintf("E%d", row), subtotal)
		tax, _ := inv.TaxAmount.Float64()
		f.SetCellValue(sheet, fmt.Sprintf("F%d", row), tax)
		total, _ := inv.TotalAmount.Float64()
		f.SetCellValue(sheet, fmt.Sprintf("G%d", row), total)
		f.SetCellValue(sheet, fmt.Sprintf("H%d", row), csvsafe.SafeCell(inv.DueDate.Format("2006-01-02")))
		f.SetCellValue(sheet, fmt.Sprintf("I%d", row), csvsafe.SafeCell(inv.CreatedAt.Format("2006-01-02")))
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
		f.SetCellValue(sheet, fmt.Sprintf("A%d", row), csvsafe.SafeCell(quote.ID.String()))
		f.SetCellValue(sheet, fmt.Sprintf("B%d", row), csvsafe.SafeCell(quote.QuoteNumber))
		f.SetCellValue(sheet, fmt.Sprintf("C%d", row), csvsafe.SafeCell(quote.CustomerID.String()))
		f.SetCellValue(sheet, fmt.Sprintf("D%d", row), csvsafe.SafeCell(string(quote.Status)))
		total, _ := quote.Total.Float64()
		f.SetCellValue(sheet, fmt.Sprintf("E%d", row), total)
		f.SetCellValue(sheet, fmt.Sprintf("F%d", row), csvsafe.SafeCell(quote.ValidUntil.Format("2006-01-02")))
		f.SetCellValue(sheet, fmt.Sprintf("G%d", row), csvsafe.SafeCell(quote.CreatedAt.Format("2006-01-02")))
	}

	return nil
}

// ExportToPDF renders a list export as a gofpdf table (T260, D13). Only the
// tabular exports have a PDF layout; validateConfig rejects PDF for the rest.
func (s *unifiedExportService) ExportToPDF(ctx context.Context, data interface{}, config *models.ExportConfig) ([]byte, error) {
	if s.pdfGen == nil {
		return nil, fmt.Errorf("PDF generator is not configured")
	}
	opts := config.PDFOptions
	if opts == nil {
		opts = models.DefaultPDFOptions()
	}

	report, ok := buildPDFReport(data, config, opts)
	if !ok {
		return nil, fmt.Errorf("PDF is not available for %s exports", config.ExportType)
	}

	genOpts := &pdf.GenerationOptions{Watermark: opts.WatermarkText}
	page := s.pdfGen.GetConfig().PageDefaults
	if strings.EqualFold(opts.Orientation, "landscape") {
		page.Orientation = "L"
	}
	if size := normalizePageSize(opts.PageSize); size != "" {
		page.Size = size
	}
	genOpts.Page = &page
	if opts.FooterText != "" {
		branding := s.pdfGen.GetConfig().Branding
		branding.FooterText = opts.FooterText
		genOpts.Branding = &branding
	}

	var buf bytes.Buffer
	if err := s.pdfGen.GenerateToWriter(ctx, report, &buf, genOpts); err != nil {
		return nil, fmt.Errorf("failed to generate PDF: %w", err)
	}
	return buf.Bytes(), nil
}

// normalizePageSize maps an export option to a gofpdf page size ("" = default).
func normalizePageSize(size string) string {
	switch strings.ToLower(size) {
	case "a4":
		return "A4"
	case "letter":
		return "Letter"
	case "legal":
		return "Legal"
	default:
		return ""
	}
}

// pdfExportSupported reports whether an export type has a PDF layout.
func pdfExportSupported(t models.ExportType) bool {
	switch t {
	case models.ExportTypeCustomers, models.ExportTypeJobs, models.ExportTypeInvoices, models.ExportTypeQuotes:
		return true
	default:
		return false
	}
}

// buildPDFReport turns list-export rows into a table. gofpdf draws text, not
// markup, so user content can't inject HTML into the PDF (F-8).
func buildPDFReport(data interface{}, config *models.ExportConfig, opts *models.PDFExportOptions) (*pdf.TableReportData, bool) {
	report := &pdf.TableReportData{
		Title:       opts.Title,
		Subtitle:    opts.HeaderText,
		GeneratedAt: time.Now(),
	}
	if report.Title == "" {
		report.Title = fmt.Sprintf("%s Export", strings.Title(string(config.ExportType))) //nolint:staticcheck // ASCII export type names
	}

	var label string
	switch d := data.(type) {
	case []models.Customer:
		label = "Total Customers"
		report.Headers = []string{"Name", "Company", "Email", "Phone", "Type", "Status", "City", "State"}
		for _, c := range d {
			report.Rows = append(report.Rows, []string{c.FirstName + " " + c.LastName, ptrToStr(c.CompanyName), c.Email, c.PhonePrimary,
				string(c.CustomerType), string(c.Status), c.BillingAddressCity, c.BillingAddressState})
		}
	case []models.Job:
		label = "Total Jobs"
		report.Headers = []string{"Title", "Status", "Scheduled", "Created"}
		for _, j := range d {
			report.Rows = append(report.Rows, []string{j.Title, string(j.Status), formatEpochPtr(j.ScheduledStartAt), j.CreatedAt.Format("2006-01-02")})
		}
	case []models.Invoice:
		label = "Total Invoices"
		report.Headers = []string{"Invoice #", "Status", "Subtotal", "Tax", "Total", "Due Date"}
		for _, inv := range d {
			report.Rows = append(report.Rows, []string{inv.InvoiceNumber, string(inv.Status), "$" + inv.Subtotal.StringFixed(2),
				"$" + inv.TaxAmount.StringFixed(2), "$" + inv.TotalAmount.StringFixed(2), inv.DueDate.Format("2006-01-02")})
		}
	case []models.Quote:
		label = "Total Quotes"
		report.Headers = []string{"Quote #", "Status", "Total", "Valid Until", "Created"}
		for _, q := range d {
			report.Rows = append(report.Rows, []string{q.QuoteNumber, string(q.Status), "$" + q.Total.StringFixed(2),
				q.ValidUntil.Format("2006-01-02"), q.CreatedAt.Format("2006-01-02")})
		}
	default:
		return nil, false
	}
	if opts.IncludeSummary {
		report.Summary = fmt.Sprintf("%s: %d", label, len(report.Rows))
	}
	return report, true
}

// GetExportStatus returns the status of an export job
func (s *unifiedExportService) GetExportStatus(ctx context.Context, jobID uuid.UUID) (*models.ExportJobResponse, error) {
	var job models.ExportJob
	if err := s.db.WithContext(ctx).First(&job, "id = ?", jobID).Error; err != nil {
		return nil, err
	}

	response := job.ToResponse()
	return &response, nil
}

// CancelExport cancels an in-progress export job
func (s *unifiedExportService) CancelExport(ctx context.Context, jobID uuid.UUID) error {
	result := s.db.WithContext(ctx).Model(&models.ExportJob{}).
		Where("id = ? AND status IN ?", jobID, []string{string(models.ExportJobStatusPending), string(models.ExportJobStatusProcessing)}).
		Update("status", models.ExportJobStatusCancelled)

	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("export job not found or cannot be cancelled")
	}

	return nil
}

// GetExportFile returns the file for download
func (s *unifiedExportService) GetExportFile(ctx context.Context, jobID uuid.UUID) (io.ReadCloser, *models.ExportJob, error) {
	var job models.ExportJob
	if err := s.db.WithContext(ctx).First(&job, "id = ?", jobID).Error; err != nil {
		return nil, nil, err
	}

	if job.Status != models.ExportJobStatusCompleted {
		return nil, nil, fmt.Errorf("export not ready for download")
	}

	file, err := s.files.Open(ctx, job.FilePath)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to open export file: %w", err)
	}

	return file, &job, nil
}

// ListExports lists recent exports for a user
func (s *unifiedExportService) ListExports(ctx context.Context, userID uuid.UUID, limit int) ([]models.ExportJobResponse, error) {
	var jobs []models.ExportJob
	if err := s.db.WithContext(ctx).Where("user_id = ?", userID).Order("created_at DESC").Limit(limit).Find(&jobs).Error; err != nil {
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
	if err := s.db.WithContext(ctx).Where("expires_at < ? AND status = ?", time.Now(), models.ExportJobStatusCompleted).Find(&jobs).Error; err != nil {
		return err
	}

	for i := range jobs {
		job := &jobs[i]
		if job.FilePath != "" {
			if err := s.files.Delete(ctx, job.FilePath); err != nil {
				return fmt.Errorf("remove export file: %w", err)
			}
		}
		if err := s.db.WithContext(ctx).Delete(job).Error; err != nil {
			return fmt.Errorf("delete export job: %w", err)
		}
	}

	return nil
}

// GetProgress reports progress from the job row (tenant-scoped).
func (s *unifiedExportService) GetProgress(ctx context.Context, jobID uuid.UUID) (*models.ExportProgress, error) {
	var job models.ExportJob
	if err := s.db.WithContext(ctx).First(&job, "id = ?", jobID).Error; err != nil {
		return nil, err
	}
	p := &models.ExportProgress{
		JobID:            job.ID,
		Status:           job.Status,
		TotalRecords:     job.TotalRecords,
		ProcessedRecords: job.ProcessedRecords,
		Progress:         job.Progress.InexactFloat64(),
		CurrentPhase:     string(job.Status),
	}
	if job.StartedAt != nil {
		end := time.Now()
		if job.CompletedAt != nil {
			end = *job.CompletedAt
		}
		p.ElapsedTimeMs = end.Sub(*job.StartedAt).Milliseconds()
	}
	if job.ErrorMessage != nil {
		p.Message = *job.ErrorMessage
	}
	return p, nil
}

// failJob marks an active job failed. A cancelled job keeps its status.
func (s *unifiedExportService) failJob(ctx context.Context, job *models.ExportJob, errorMsg string) {
	completedAt := time.Now()
	err := s.setProgress(ctx, job.ID, map[string]interface{}{
		"status":        models.ExportJobStatusFailed,
		"error_message": errorMsg,
		"completed_at":  completedAt,
	})
	if err != nil && !errors.Is(err, errExportNotActive) {
		logExportError(ctx, job.ID, err)
	}
}

func (s *unifiedExportService) validateConfig(config *models.ExportConfig) error {
	if config.Format == "" {
		return fmt.Errorf("export format is required")
	}
	if config.ExportType == "" {
		return fmt.Errorf("export type is required")
	}
	if config.Format == models.ExportFormatPDF && !pdfExportSupported(config.ExportType) {
		return apperrors.Validation("PDF is not available for this export; use CSV, Excel or JSON",
			map[string]string{"format": "pdf is only available for customers, jobs, invoices and quotes"})
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

func logExportError(ctx context.Context, jobID uuid.UUID, err error) {
	logging.Error(ctx, "[EXPORT] Failed to record export failure", map[string]any{"job_id": jobID.String(), "error": err.Error()})
}
