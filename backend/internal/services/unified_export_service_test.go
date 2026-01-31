package services

import (
	"bytes"
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/javaknight1/servicepro/backend/internal/models"
)

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
	service := NewUnifiedExportService(db, pdfService, "./test_exports", "http://localhost")

	config := &models.ExportConfig{
		Format:     models.ExportFormatCSV,
		ExportType: models.ExportTypeCustomers,
		CSVOptions: models.DefaultCSVOptions(),
	}

	data, err := service.ExportToCSV(context.Background(), customers, config)

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
	service := NewUnifiedExportService(db, pdfService, "./test_exports", "http://localhost")

	config := &models.ExportConfig{
		Format:     models.ExportFormatCSV,
		ExportType: models.ExportTypeCustomers,
		CSVOptions: &models.CSVExportOptions{
			Delimiter:     ",",
			IncludeHeader: true,
			BOM:           true,
		},
	}

	data, err := service.ExportToCSV(context.Background(), customers, config)

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
	service := NewUnifiedExportService(db, pdfService, "./test_exports", "http://localhost")

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

			data, err := service.ExportToCSV(context.Background(), customers, config)

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
	service := NewUnifiedExportService(db, pdfService, "./test_exports", "http://localhost")

	config := &models.ExportConfig{
		Format:     models.ExportFormatJSON,
		ExportType: models.ExportTypeCustomers,
	}

	data, err := service.ExportToJSON(context.Background(), customers, config)

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
	service := NewUnifiedExportService(db, pdfService, "./test_exports", "http://localhost")

	config := &models.ExportConfig{
		Format:     models.ExportFormatExcel,
		ExportType: models.ExportTypeCustomers,
		ExcelOptions: &models.ExcelExportOptions{
			SheetName:   "Customers",
			FreezePanes: true,
			AutoFilter:  true,
		},
	}

	data, err := service.ExportToExcel(context.Background(), customers, config)

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
	service := NewUnifiedExportService(db, pdfService, "./test_exports", "http://localhost")

	config := &models.ExportConfig{
		Format:     models.ExportFormatCSV,
		ExportType: models.ExportTypeCustomers,
		Fields:     []string{"first_name", "last_name", "email"},
		CSVOptions: &models.CSVExportOptions{
			IncludeHeader: true,
			BOM:           false,
		},
	}

	data, err := service.ExportToCSV(context.Background(), customers, config)

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
	service := NewUnifiedExportService(db, pdfService, "./test_exports", "http://localhost")

	userID := uuid.New()

	config := &models.ExportConfig{
		Format:     models.ExportFormatCSV,
		ExportType: models.ExportTypeCustomers,
	}

	// Start export
	job, err := service.StartExport(context.Background(), userID, config)

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
			status, err := service.GetExportStatus(context.Background(), job.ID)
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

// TestExportProgress tests progress tracking
func TestExportProgress(t *testing.T) {
	db := setupExportTestDB(t)
	createExportTestCustomers(t, db, 100)

	pdfService := NewPDFService("", "", "")
	service := NewUnifiedExportService(db, pdfService, "./test_exports", "http://localhost")

	userID := uuid.New()

	config := &models.ExportConfig{
		Format:     models.ExportFormatCSV,
		ExportType: models.ExportTypeCustomers,
	}

	job, err := service.StartExport(context.Background(), userID, config)
	require.NoError(t, err)

	// Track progress updates
	progressUpdates := []float64{}
	timeout := time.After(10 * time.Second)
	ticker := time.NewTicker(50 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-timeout:
			goto check
		case <-ticker.C:
			progress := service.GetProgress(job.ID)
			if progress != nil {
				progressUpdates = append(progressUpdates, progress.Progress)
				if progress.Status == models.ExportJobStatusCompleted {
					goto check
				}
			}
		}
	}
check:

	// Should have multiple progress updates
	assert.True(t, len(progressUpdates) > 0, "Should have progress updates")

	// Last progress should be 100
	if len(progressUpdates) > 0 {
		lastProgress := progressUpdates[len(progressUpdates)-1]
		assert.Equal(t, float64(100), lastProgress)
	}
}

// TestCancelExport tests export cancellation
func TestCancelExport(t *testing.T) {
	db := setupExportTestDB(t)
	createExportTestCustomers(t, db, 10)

	pdfService := NewPDFService("", "", "")
	service := NewUnifiedExportService(db, pdfService, "./test_exports", "http://localhost")

	userID := uuid.New()

	config := &models.ExportConfig{
		Format:     models.ExportFormatCSV,
		ExportType: models.ExportTypeCustomers,
	}

	job, err := service.StartExport(context.Background(), userID, config)
	require.NoError(t, err)

	// Try to cancel immediately (may or may not succeed depending on timing)
	_ = service.CancelExport(context.Background(), job.ID)

	// Get final status
	time.Sleep(200 * time.Millisecond)
	status, err := service.GetExportStatus(context.Background(), job.ID)
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
	service := NewUnifiedExportService(db, pdfService, "./test_exports", "http://localhost")

	userID := uuid.New()

	// Create multiple exports
	formats := []models.ExportFormat{models.ExportFormatCSV, models.ExportFormatJSON}
	for _, format := range formats {
		config := &models.ExportConfig{
			Format:     format,
			ExportType: models.ExportTypeCustomers,
		}
		_, err := service.StartExport(context.Background(), userID, config)
		require.NoError(t, err)
	}

	// Wait for exports to complete
	time.Sleep(2 * time.Second)

	// List exports
	exports, err := service.ListExports(context.Background(), userID, 10)

	require.NoError(t, err)
	assert.Len(t, exports, 2)
}

// TestExportValidation tests config validation
func TestExportValidation(t *testing.T) {
	db := setupExportTestDB(t)

	pdfService := NewPDFService("", "", "")
	service := NewUnifiedExportService(db, pdfService, "./test_exports", "http://localhost")

	userID := uuid.New()

	t.Run("missing format", func(t *testing.T) {
		config := &models.ExportConfig{
			ExportType: models.ExportTypeCustomers,
		}

		_, err := service.StartExport(context.Background(), userID, config)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "format")
	})

	t.Run("missing export type", func(t *testing.T) {
		config := &models.ExportConfig{
			Format: models.ExportFormatCSV,
		}

		_, err := service.StartExport(context.Background(), userID, config)
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
	service := NewUnifiedExportService(db, pdfService, "./test_exports", "http://localhost")

	config := &models.ExportConfig{
		Format:     models.ExportFormatCSV,
		ExportType: models.ExportTypeCustomers,
		Encoding:   "utf-8",
		CSVOptions: &models.CSVExportOptions{
			IncludeHeader: true,
			BOM:           true, // BOM for proper UTF-8 handling
		},
	}

	data, err := service.ExportToCSV(context.Background(), []models.Customer{customer}, config)
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
	service := NewUnifiedExportService(db, pdfService, "./test_exports", "http://localhost")

	var customers []models.Customer
	db.Find(&customers)

	config := &models.ExportConfig{
		Format:     models.ExportFormatCSV,
		ExportType: models.ExportTypeCustomers,
		CSVOptions: models.DefaultCSVOptions(),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		service.ExportToCSV(context.Background(), customers, config)
	}
}

func BenchmarkExportCSV_1000Records(b *testing.B) {
	db := setupExportBenchmarkDB(b, 1000)
	pdfService := NewPDFService("", "", "")
	service := NewUnifiedExportService(db, pdfService, "./test_exports", "http://localhost")

	var customers []models.Customer
	db.Find(&customers)

	config := &models.ExportConfig{
		Format:     models.ExportFormatCSV,
		ExportType: models.ExportTypeCustomers,
		CSVOptions: models.DefaultCSVOptions(),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		service.ExportToCSV(context.Background(), customers, config)
	}
}

func BenchmarkExportCSV_10000Records(b *testing.B) {
	db := setupExportBenchmarkDB(b, 10000)
	pdfService := NewPDFService("", "", "")
	service := NewUnifiedExportService(db, pdfService, "./test_exports", "http://localhost")

	var customers []models.Customer
	db.Find(&customers)

	config := &models.ExportConfig{
		Format:     models.ExportFormatCSV,
		ExportType: models.ExportTypeCustomers,
		CSVOptions: models.DefaultCSVOptions(),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		service.ExportToCSV(context.Background(), customers, config)
	}
}

func BenchmarkExportJSON_1000Records(b *testing.B) {
	db := setupExportBenchmarkDB(b, 1000)
	pdfService := NewPDFService("", "", "")
	service := NewUnifiedExportService(db, pdfService, "./test_exports", "http://localhost")

	var customers []models.Customer
	db.Find(&customers)

	config := &models.ExportConfig{
		Format:     models.ExportFormatJSON,
		ExportType: models.ExportTypeCustomers,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		service.ExportToJSON(context.Background(), customers, config)
	}
}

func BenchmarkExportExcel_1000Records(b *testing.B) {
	db := setupExportBenchmarkDB(b, 1000)
	pdfService := NewPDFService("", "", "")
	service := NewUnifiedExportService(db, pdfService, "./test_exports", "http://localhost")

	var customers []models.Customer
	db.Find(&customers)

	config := &models.ExportConfig{
		Format:       models.ExportFormatExcel,
		ExportType:   models.ExportTypeCustomers,
		ExcelOptions: models.DefaultExcelOptions(),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		service.ExportToExcel(context.Background(), customers, config)
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
