package services

import (
	"bytes"
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/javaknight1/servicepro/backend/internal/models"
	"github.com/javaknight1/servicepro/backend/internal/tenancy"
	storagemock "github.com/javaknight1/servicepro/backend/pkg/clients/storage/mock"
)

// exportCtx carries a tenant: exports are tenant-owned and their storage key
// includes the tenant (T128).
var exportCtx = tenancy.WithTenant(context.Background(), uuid.MustParse("7e57a000-0000-4000-8000-0000000000e1"))

func setupExportTestDB(t *testing.T) *gorm.DB {
	// Use a unique database name per test with shared cache mode
	// This allows goroutines within the same test to access the same in-memory database
	// while keeping tests isolated from each other
	dbName := fmt.Sprintf("file:%s?mode=memory&cache=shared", t.Name())
	db, err := gorm.Open(sqlite.Open(dbName), &gorm.Config{})
	require.NoError(t, err)

	// Set busy timeout to avoid "database is locked" errors when multiple
	// goroutines access the database concurrently
	sqlDB, err := db.DB()
	require.NoError(t, err)
	_, err = sqlDB.Exec("PRAGMA busy_timeout = 5000")
	require.NoError(t, err)

	// Create customers table
	err = db.Exec(`
		CREATE TABLE customers (
			tenant_id TEXT,
			id TEXT PRIMARY KEY,
			first_name TEXT NOT NULL,
			last_name TEXT NOT NULL,
			company_name TEXT,
			email TEXT NOT NULL,
			phone_primary TEXT,
			phone_secondary TEXT,
			customer_type TEXT NOT NULL DEFAULT 'residential',
			status TEXT NOT NULL DEFAULT 'prospect',
			billing_address_street TEXT,
			billing_address_city TEXT,
			billing_address_state TEXT,
			billing_address_zip TEXT,
			service_address_street TEXT,
			service_address_city TEXT,
			service_address_state TEXT,
			service_address_zip TEXT,
			notes TEXT,
			preferred_contact_method TEXT DEFAULT 'any',
			do_not_email BOOLEAN DEFAULT false,
			do_not_call BOOLEAN DEFAULT false,
			do_not_sms BOOLEAN DEFAULT false,
			do_not_mail BOOLEAN DEFAULT false,
			marketing_consent BOOLEAN DEFAULT false,
			marketing_consent_at DATETIME,
			marketing_consent_ip TEXT,
			created_at DATETIME NOT NULL,
			updated_at DATETIME NOT NULL,
			deleted_at DATETIME
		)
	`).Error
	require.NoError(t, err)

	// Create export_jobs table
	err = db.Exec(`
		CREATE TABLE export_jobs (
			tenant_id TEXT,
			id TEXT PRIMARY KEY,
			user_id TEXT NOT NULL,
			export_type TEXT NOT NULL,
			format TEXT NOT NULL,
			status TEXT NOT NULL DEFAULT 'pending',
			total_records INTEGER NOT NULL DEFAULT 0,
			processed_records INTEGER NOT NULL DEFAULT 0,
			progress DECIMAL(5,2) NOT NULL DEFAULT 0,
			filename TEXT,
			file_path TEXT,
			file_size INTEGER DEFAULT 0,
			mime_type TEXT,
			download_url TEXT,
			query TEXT,
			encoding TEXT DEFAULT 'utf-8',
			error_message TEXT,
			error_details TEXT,
			started_at DATETIME,
			completed_at DATETIME,
			expires_at DATETIME,
			created_at DATETIME NOT NULL,
			updated_at DATETIME NOT NULL
		)
	`).Error
	require.NoError(t, err)

	return db
}

func createExportTestCustomers(t *testing.T, db *gorm.DB, count int) []models.Customer {
	customers := make([]models.Customer, count)

	states := []string{"CA", "TX", "NY", "FL", "WA"}
	types := []models.CustomerType{models.CustomerTypeResidential, models.CustomerTypeCommercial}
	statuses := []models.CustomerStatus{models.CustomerStatusActive, models.CustomerStatusInactive, models.CustomerStatusProspect}

	for i := 0; i < count; i++ {
		customer := models.Customer{
			ID:                   uuid.New(),
			FirstName:            "First" + string(rune('A'+i%26)),
			LastName:             "Last" + string(rune('A'+i%26)),
			Email:                "email" + string(rune('0'+i%10)) + "@example.com",
			PhonePrimary:         "555-000-" + string(rune('0'+i%10)) + string(rune('0'+(i/10)%10)) + string(rune('0'+(i/100)%10)) + string(rune('0'+(i/1000)%10)),
			CustomerType:         types[i%len(types)],
			Status:               statuses[i%len(statuses)],
			BillingAddressStreet: "123 Main St",
			BillingAddressCity:   "City" + string(rune('A'+i%26)),
			BillingAddressState:  states[i%len(states)],
			BillingAddressZip:    "90000",
			CreatedAt:            time.Now().Add(-time.Duration(i) * time.Hour),
			UpdatedAt:            time.Now(),
		}

		err := db.Create(&customer).Error
		require.NoError(t, err)
		customers[i] = customer
	}

	return customers
}

// TestExportToCSV tests CSV export functionality
func TestExportToCSV(t *testing.T) {
	db := setupExportTestDB(t)
	customers := createExportTestCustomers(t, db, 10)

	pdfService := NewPDFService("", "", "")
	service := NewUnifiedExportService(db, pdfService, NewLocalExportStore(t.TempDir()), "http://localhost")

	config := &models.ExportConfig{
		Format:     models.ExportFormatCSV,
		ExportType: models.ExportTypeCustomers,
		CSVOptions: models.DefaultCSVOptions(),
	}

	data, err := service.ExportToCSV(exportCtx, customers, config)

	require.NoError(t, err)
	assert.NotEmpty(t, data)

	// Parse CSV and verify
	reader := csv.NewReader(bytes.NewReader(data))
	records, err := reader.ReadAll()
	require.NoError(t, err)

	// Should have header + 10 data rows
	assert.Len(t, records, 11)

	// Verify header row
	header := records[0]
	assert.Contains(t, header, "first_name")
	assert.Contains(t, header, "last_name")
	assert.Contains(t, header, "email")
}

// TestExportToCSVWithBOM tests UTF-8 BOM for Excel compatibility
func TestExportToCSVWithBOM(t *testing.T) {
	db := setupExportTestDB(t)
	customers := createExportTestCustomers(t, db, 5)

	pdfService := NewPDFService("", "", "")
	service := NewUnifiedExportService(db, pdfService, NewLocalExportStore(t.TempDir()), "http://localhost")

	config := &models.ExportConfig{
		Format:     models.ExportFormatCSV,
		ExportType: models.ExportTypeCustomers,
		CSVOptions: &models.CSVExportOptions{
			Delimiter:     ",",
			IncludeHeader: true,
			BOM:           true,
		},
	}

	data, err := service.ExportToCSV(exportCtx, customers, config)

	require.NoError(t, err)

	// Check for UTF-8 BOM
	assert.True(t, len(data) >= 3)
	assert.Equal(t, byte(0xEF), data[0])
	assert.Equal(t, byte(0xBB), data[1])
	assert.Equal(t, byte(0xBF), data[2])
}

// TestExportToCSVWithDelimiter tests custom delimiters
func TestExportToCSVWithDelimiter(t *testing.T) {
	db := setupExportTestDB(t)
	customers := createExportTestCustomers(t, db, 3)

	pdfService := NewPDFService("", "", "")
	service := NewUnifiedExportService(db, pdfService, NewLocalExportStore(t.TempDir()), "http://localhost")

	testCases := []struct {
		name      string
		delimiter string
		expected  rune
	}{
		{"comma", ",", ','},
		{"tab", "tab", '\t'},
		{"semicolon", "semicolon", ';'},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			config := &models.ExportConfig{
				Format:     models.ExportFormatCSV,
				ExportType: models.ExportTypeCustomers,
				CSVOptions: &models.CSVExportOptions{
					Delimiter:     tc.delimiter,
					IncludeHeader: true,
					BOM:           false,
				},
			}

			data, err := service.ExportToCSV(exportCtx, customers, config)

			require.NoError(t, err)
			assert.NotEmpty(t, data)

			// Verify delimiter is used
			reader := csv.NewReader(bytes.NewReader(data))
			reader.Comma = tc.expected
			records, err := reader.ReadAll()
			require.NoError(t, err)
			assert.Len(t, records, 4) // header + 3 data rows
		})
	}
}

// TestExportToJSON tests JSON export functionality
func TestExportToJSON(t *testing.T) {
	db := setupExportTestDB(t)
	customers := createExportTestCustomers(t, db, 5)

	pdfService := NewPDFService("", "", "")
	service := NewUnifiedExportService(db, pdfService, NewLocalExportStore(t.TempDir()), "http://localhost")

	config := &models.ExportConfig{
		Format:     models.ExportFormatJSON,
		ExportType: models.ExportTypeCustomers,
	}

	data, err := service.ExportToJSON(exportCtx, customers, config)

	require.NoError(t, err)
	assert.NotEmpty(t, data)

	// Parse JSON and verify
	var parsed []models.Customer
	err = json.Unmarshal(data, &parsed)
	require.NoError(t, err)
	assert.Len(t, parsed, 5)
}

// TestExportToExcel tests Excel export functionality
func TestExportToExcel(t *testing.T) {
	db := setupExportTestDB(t)
	customers := createExportTestCustomers(t, db, 10)

	pdfService := NewPDFService("", "", "")
	service := NewUnifiedExportService(db, pdfService, NewLocalExportStore(t.TempDir()), "http://localhost")

	config := &models.ExportConfig{
		Format:     models.ExportFormatExcel,
		ExportType: models.ExportTypeCustomers,
		ExcelOptions: &models.ExcelExportOptions{
			SheetName:   "Customers",
			FreezePanes: true,
			AutoFilter:  true,
		},
	}

	data, err := service.ExportToExcel(exportCtx, customers, config)

	require.NoError(t, err)
	assert.NotEmpty(t, data)

	// Excel files start with PK (ZIP signature)
	assert.True(t, len(data) >= 2)
	assert.Equal(t, byte('P'), data[0])
	assert.Equal(t, byte('K'), data[1])
}

// TestExportWithFieldSelection tests exporting with specific fields
func TestExportWithFieldSelection(t *testing.T) {
	db := setupExportTestDB(t)
	customers := createExportTestCustomers(t, db, 5)

	pdfService := NewPDFService("", "", "")
	service := NewUnifiedExportService(db, pdfService, NewLocalExportStore(t.TempDir()), "http://localhost")

	config := &models.ExportConfig{
		Format:     models.ExportFormatCSV,
		ExportType: models.ExportTypeCustomers,
		Fields:     []string{"first_name", "last_name", "email"},
		CSVOptions: &models.CSVExportOptions{
			IncludeHeader: true,
			BOM:           false,
		},
	}

	data, err := service.ExportToCSV(exportCtx, customers, config)

	require.NoError(t, err)

	// Parse CSV and verify only selected fields
	reader := csv.NewReader(bytes.NewReader(data))
	records, err := reader.ReadAll()
	require.NoError(t, err)

	// Header should only have 3 fields
	assert.Len(t, records[0], 3)
	assert.Equal(t, "first_name", records[0][0])
	assert.Equal(t, "last_name", records[0][1])
	assert.Equal(t, "email", records[0][2])
}

// TestExportJobLifecycle tests the complete export job lifecycle
func TestExportJobLifecycle(t *testing.T) {
	db := setupExportTestDB(t)
	createExportTestCustomers(t, db, 20)

	pdfService := NewPDFService("", "", "")
	service := NewUnifiedExportService(db, pdfService, NewLocalExportStore(t.TempDir()), "http://localhost")

	userID := uuid.New()

	config := &models.ExportConfig{
		Format:     models.ExportFormatCSV,
		ExportType: models.ExportTypeCustomers,
	}

	// Start export
	job, err := service.StartExport(exportCtx, userID, config)

	require.NoError(t, err)
	assert.NotEqual(t, uuid.Nil, job.ID)
	assert.Equal(t, userID, job.UserID)
	assert.Equal(t, models.ExportTypeCustomers, job.ExportType)
	assert.Equal(t, models.ExportFormatCSV, job.Format)

	// Wait for completion (with timeout)
	timeout := time.After(10 * time.Second)
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	var finalStatus *models.ExportJobResponse
	for {
		select {
		case <-timeout:
			t.Fatal("Export job timeout")
		case <-ticker.C:
			status, err := service.GetExportStatus(exportCtx, job.ID)
			require.NoError(t, err)

			if status.Status == models.ExportJobStatusCompleted ||
				status.Status == models.ExportJobStatusFailed {
				finalStatus = status
				goto done
			}
		}
	}
done:

	assert.NotNil(t, finalStatus)
	assert.Equal(t, models.ExportJobStatusCompleted, finalStatus.Status)
	assert.Equal(t, 20, finalStatus.TotalRecords)
	assert.Equal(t, float64(100), finalStatus.Progress)
}

// TestExportProgress: progress is persisted on the job row (T128), so a
// different service instance (another Cloud Run instance) sees it.
func TestExportProgress(t *testing.T) {
	db := setupExportTestDB(t)
	createExportTestCustomers(t, db, 100)

	pdfService := NewPDFService("", "", "")
	service := NewUnifiedExportService(db, pdfService, NewLocalExportStore(t.TempDir()), "http://localhost")

	job, err := service.StartExport(exportCtx, uuid.New(), &models.ExportConfig{
		Format:     models.ExportFormatCSV,
		ExportType: models.ExportTypeCustomers,
	})
	require.NoError(t, err)
	assert.Equal(t, models.ExportJobStatusCompleted, job.Status, "the export runs inside the request")

	other := NewUnifiedExportService(db, pdfService, NewLocalExportStore(t.TempDir()), "http://localhost")
	progress, err := other.GetProgress(exportCtx, job.ID)
	require.NoError(t, err)
	assert.Equal(t, models.ExportJobStatusCompleted, progress.Status)
	assert.Equal(t, float64(100), progress.Progress)
	assert.Equal(t, 100, progress.TotalRecords)
	assert.Equal(t, 100, progress.ProcessedRecords)
}

// T128: a cancelled job is not flipped back to completed/failed by the run.
func TestExportCancelledJobStaysCancelled(t *testing.T) {
	db := setupExportTestDB(t)
	createExportTestCustomers(t, db, 3)
	svc := NewUnifiedExportService(db, nil, NewLocalExportStore(t.TempDir()), "").(*unifiedExportService)

	job := &models.ExportJob{ID: uuid.New(), UserID: uuid.New(), ExportType: models.ExportTypeCustomers,
		Format: models.ExportFormatCSV, Status: models.ExportJobStatusPending, Filename: "c.csv",
		FilePath: t.TempDir() + "/c.csv", CreatedAt: time.Now(), UpdatedAt: time.Now()}
	require.NoError(t, db.Create(job).Error)
	require.NoError(t, svc.CancelExport(exportCtx, job.ID))

	svc.processExport(exportCtx, job.ID, &models.ExportConfig{Format: models.ExportFormatCSV, ExportType: models.ExportTypeCustomers})

	st, err := svc.GetExportStatus(exportCtx, job.ID)
	require.NoError(t, err)
	assert.Equal(t, models.ExportJobStatusCancelled, st.Status)
}

// TestCancelExport tests export cancellation
func TestCancelExport(t *testing.T) {
	db := setupExportTestDB(t)
	createExportTestCustomers(t, db, 10)

	pdfService := NewPDFService("", "", "")
	service := NewUnifiedExportService(db, pdfService, NewLocalExportStore(t.TempDir()), "http://localhost")

	userID := uuid.New()

	config := &models.ExportConfig{
		Format:     models.ExportFormatCSV,
		ExportType: models.ExportTypeCustomers,
	}

	job, err := service.StartExport(exportCtx, userID, config)
	require.NoError(t, err)

	// Try to cancel immediately (may or may not succeed depending on timing)
	_ = service.CancelExport(exportCtx, job.ID)

	// Get final status
	time.Sleep(200 * time.Millisecond)
	status, err := service.GetExportStatus(exportCtx, job.ID)
	require.NoError(t, err)

	// Status should be either cancelled or completed (if it finished before cancel)
	assert.True(t,
		status.Status == models.ExportJobStatusCancelled ||
			status.Status == models.ExportJobStatusCompleted,
		"Status should be cancelled or completed")
}

// TestListExports tests listing exports for a user
func TestListExports(t *testing.T) {
	db := setupExportTestDB(t)
	createExportTestCustomers(t, db, 5)

	pdfService := NewPDFService("", "", "")
	service := NewUnifiedExportService(db, pdfService, NewLocalExportStore(t.TempDir()), "http://localhost")

	userID := uuid.New()

	// Create multiple exports
	formats := []models.ExportFormat{models.ExportFormatCSV, models.ExportFormatJSON}
	for _, format := range formats {
		config := &models.ExportConfig{
			Format:     format,
			ExportType: models.ExportTypeCustomers,
		}
		_, err := service.StartExport(exportCtx, userID, config)
		require.NoError(t, err)
	}

	// List exports
	exports, err := service.ListExports(exportCtx, userID, 10)

	require.NoError(t, err)
	assert.Len(t, exports, 2)
}

// TestExportValidation tests config validation
func TestExportValidation(t *testing.T) {
	db := setupExportTestDB(t)

	pdfService := NewPDFService("", "", "")
	service := NewUnifiedExportService(db, pdfService, NewLocalExportStore(t.TempDir()), "http://localhost")

	userID := uuid.New()

	t.Run("missing format", func(t *testing.T) {
		config := &models.ExportConfig{
			ExportType: models.ExportTypeCustomers,
		}

		_, err := service.StartExport(exportCtx, userID, config)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "format")
	})

	t.Run("missing export type", func(t *testing.T) {
		config := &models.ExportConfig{
			Format: models.ExportFormatCSV,
		}

		_, err := service.StartExport(exportCtx, userID, config)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "export type")
	})
}

// TestCharacterEncoding tests proper character encoding
func TestCharacterEncoding(t *testing.T) {
	db := setupExportTestDB(t)

	// Create customer with special characters
	customer := models.Customer{
		ID:                   uuid.New(),
		FirstName:            "José",
		LastName:             "García",
		Email:                "jose@example.com",
		PhonePrimary:         "555-0000",
		CustomerType:         models.CustomerTypeResidential,
		Status:               models.CustomerStatusActive,
		BillingAddressStreet: "Calle España 日本語",
		BillingAddressCity:   "München",
		BillingAddressState:  "CA",
		BillingAddressZip:    "90000",
		CreatedAt:            time.Now(),
		UpdatedAt:            time.Now(),
	}
	err := db.Create(&customer).Error
	require.NoError(t, err)

	pdfService := NewPDFService("", "", "")
	service := NewUnifiedExportService(db, pdfService, NewLocalExportStore(t.TempDir()), "http://localhost")

	config := &models.ExportConfig{
		Format:     models.ExportFormatCSV,
		ExportType: models.ExportTypeCustomers,
		Encoding:   "utf-8",
		CSVOptions: &models.CSVExportOptions{
			IncludeHeader: true,
			BOM:           true, // BOM for proper UTF-8 handling
		},
	}

	data, err := service.ExportToCSV(exportCtx, []models.Customer{customer}, config)
	require.NoError(t, err)

	// Verify special characters are preserved
	content := string(data)
	assert.Contains(t, content, "José")
	assert.Contains(t, content, "García")
	assert.Contains(t, content, "München")
	assert.Contains(t, content, "日本語")
}

// Benchmark tests
func BenchmarkExportCSV_100Records(b *testing.B) {
	db := setupExportBenchmarkDB(b, 100)
	pdfService := NewPDFService("", "", "")
	service := NewUnifiedExportService(db, pdfService, NewLocalExportStore(b.TempDir()), "http://localhost")

	var customers []models.Customer
	db.Find(&customers)

	config := &models.ExportConfig{
		Format:     models.ExportFormatCSV,
		ExportType: models.ExportTypeCustomers,
		CSVOptions: models.DefaultCSVOptions(),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		service.ExportToCSV(exportCtx, customers, config)
	}
}

func BenchmarkExportCSV_1000Records(b *testing.B) {
	db := setupExportBenchmarkDB(b, 1000)
	pdfService := NewPDFService("", "", "")
	service := NewUnifiedExportService(db, pdfService, NewLocalExportStore(b.TempDir()), "http://localhost")

	var customers []models.Customer
	db.Find(&customers)

	config := &models.ExportConfig{
		Format:     models.ExportFormatCSV,
		ExportType: models.ExportTypeCustomers,
		CSVOptions: models.DefaultCSVOptions(),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		service.ExportToCSV(exportCtx, customers, config)
	}
}

func BenchmarkExportCSV_10000Records(b *testing.B) {
	db := setupExportBenchmarkDB(b, 10000)
	pdfService := NewPDFService("", "", "")
	service := NewUnifiedExportService(db, pdfService, NewLocalExportStore(b.TempDir()), "http://localhost")

	var customers []models.Customer
	db.Find(&customers)

	config := &models.ExportConfig{
		Format:     models.ExportFormatCSV,
		ExportType: models.ExportTypeCustomers,
		CSVOptions: models.DefaultCSVOptions(),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		service.ExportToCSV(exportCtx, customers, config)
	}
}

func BenchmarkExportJSON_1000Records(b *testing.B) {
	db := setupExportBenchmarkDB(b, 1000)
	pdfService := NewPDFService("", "", "")
	service := NewUnifiedExportService(db, pdfService, NewLocalExportStore(b.TempDir()), "http://localhost")

	var customers []models.Customer
	db.Find(&customers)

	config := &models.ExportConfig{
		Format:     models.ExportFormatJSON,
		ExportType: models.ExportTypeCustomers,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		service.ExportToJSON(exportCtx, customers, config)
	}
}

func BenchmarkExportExcel_1000Records(b *testing.B) {
	db := setupExportBenchmarkDB(b, 1000)
	pdfService := NewPDFService("", "", "")
	service := NewUnifiedExportService(db, pdfService, NewLocalExportStore(b.TempDir()), "http://localhost")

	var customers []models.Customer
	db.Find(&customers)

	config := &models.ExportConfig{
		Format:       models.ExportFormatExcel,
		ExportType:   models.ExportTypeCustomers,
		ExcelOptions: models.DefaultExcelOptions(),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		service.ExportToExcel(exportCtx, customers, config)
	}
}

func setupExportBenchmarkDB(b *testing.B, count int) *gorm.DB {
	// Use shared cache mode so all goroutines access the same in-memory database
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	if err != nil {
		b.Fatal(err)
	}

	db.Exec(`
		CREATE TABLE customers (
			tenant_id TEXT,
			id TEXT PRIMARY KEY,
			first_name TEXT NOT NULL,
			last_name TEXT NOT NULL,
			company_name TEXT,
			email TEXT NOT NULL,
			phone_primary TEXT,
			phone_secondary TEXT,
			customer_type TEXT NOT NULL DEFAULT 'residential',
			status TEXT NOT NULL DEFAULT 'prospect',
			billing_address_street TEXT,
			billing_address_city TEXT,
			billing_address_state TEXT,
			billing_address_zip TEXT,
			service_address_street TEXT,
			service_address_city TEXT,
			service_address_state TEXT,
			service_address_zip TEXT,
			notes TEXT,
			created_at DATETIME NOT NULL,
			updated_at DATETIME NOT NULL,
			deleted_at DATETIME
		)
	`)

	// Create test customers
	for i := 0; i < count; i++ {
		customer := models.Customer{
			ID:                   uuid.New(),
			FirstName:            "First" + string(rune('A'+i%26)),
			LastName:             "Last" + string(rune('A'+i%26)),
			Email:                "email" + string(rune('0'+i%10)) + "@example.com",
			PhonePrimary:         "555-0000",
			CustomerType:         models.CustomerTypeResidential,
			Status:               models.CustomerStatusActive,
			BillingAddressStreet: "123 Main St",
			BillingAddressCity:   "City",
			BillingAddressState:  "CA",
			BillingAddressZip:    "90000",
			CreatedAt:            time.Now(),
			UpdatedAt:            time.Now(),
		}
		db.Create(&customer)
	}

	return db
}

// TestExportMimeTypes tests that correct MIME types are returned
func TestExportMimeTypes(t *testing.T) {
	testCases := []struct {
		format   models.ExportFormat
		expected string
	}{
		{models.ExportFormatCSV, "text/csv; charset=utf-8"},
		{models.ExportFormatJSON, "application/json; charset=utf-8"},
		{models.ExportFormatExcel, "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"},
		{models.ExportFormatPDF, "application/pdf"},
	}

	for _, tc := range testCases {
		t.Run(string(tc.format), func(t *testing.T) {
			assert.Equal(t, tc.expected, tc.format.GetMimeType())
		})
	}
}

// TestExportFileExtensions tests that correct file extensions are returned
func TestExportFileExtensions(t *testing.T) {
	testCases := []struct {
		format   models.ExportFormat
		expected string
	}{
		{models.ExportFormatCSV, ".csv"},
		{models.ExportFormatJSON, ".json"},
		{models.ExportFormatExcel, ".xlsx"},
		{models.ExportFormatPDF, ".pdf"},
	}

	for _, tc := range testCases {
		t.Run(string(tc.format), func(t *testing.T) {
			assert.Equal(t, tc.expected, tc.format.GetExtension())
		})
	}
}

// T128: export files go to object storage, so a download served by another
// instance (a fresh service with its own empty local disk) still works, and
// cleanup removes the object.
func TestExport_ObjectStorageDownloadFromAnyInstance(t *testing.T) {
	db := setupExportTestDB(t)
	createExportTestCustomers(t, db, 5)
	store := NewObjectExportStore(storagemock.NewClient(nil))

	userID := uuid.New()
	job, err := NewUnifiedExportService(db, nil, store, "").StartExport(exportCtx, userID, &models.ExportConfig{
		Format: models.ExportFormatCSV, ExportType: models.ExportTypeCustomers,
	})
	require.NoError(t, err)
	require.Equal(t, models.ExportJobStatusCompleted, job.Status)
	assert.Contains(t, job.FilePath, "exports/7e57a000-0000-4000-8000-0000000000e1/"+job.ID.String()+"/")

	other := NewUnifiedExportService(db, nil, store, "")
	rc, got, err := other.GetExportFile(exportCtx, job.ID)
	require.NoError(t, err)
	defer rc.Close()
	body, err := io.ReadAll(rc)
	require.NoError(t, err)
	assert.Equal(t, job.ID, got.ID)
	assert.Equal(t, int64(len(body)), got.FileSize)
	assert.Contains(t, string(body), "First")

	require.NoError(t, db.Model(&models.ExportJob{}).Where("id = ?", job.ID).Update("expires_at", time.Now().Add(-time.Hour)).Error)
	require.NoError(t, other.CleanupExpiredExports(exportCtx))
	_, err = store.Open(exportCtx, job.FilePath)
	assert.Error(t, err, "expired export object is deleted")
}

func TestExport_RequiresTenant(t *testing.T) {
	db := setupExportTestDB(t)
	_, err := NewUnifiedExportService(db, nil, NewLocalExportStore(t.TempDir()), "").StartExport(context.Background(), uuid.New(),
		&models.ExportConfig{Format: models.ExportFormatCSV, ExportType: models.ExportTypeCustomers})
	assert.ErrorIs(t, err, tenancy.ErrNoTenant)
}
