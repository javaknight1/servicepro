package services

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/javaknight1/servicepro/backend/internal/models"
)

// setupTestDB creates an in-memory SQLite database for testing
func setupTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		DisableForeignKeyConstraintWhenMigrating: true,
	})
	require.NoError(t, err)

	// Manually create tables with SQLite-compatible syntax
	// (AutoMigrate fails with PostgreSQL-specific uuid_generate_v4())
	err = db.Exec(`
		CREATE TABLE IF NOT EXISTS users (
			id TEXT PRIMARY KEY,
			email TEXT NOT NULL,
			password_hash TEXT,
			role TEXT DEFAULT 'user',
			email_verified INTEGER DEFAULT 0,
			verification_sent_at DATETIME,
			failed_login_count INTEGER DEFAULT 0,
			last_failed_login_at DATETIME,
			locked_until DATETIME,
			created_at DATETIME,
			updated_at DATETIME,
			deleted_at DATETIME
		)
	`).Error
	require.NoError(t, err)

	err = db.Exec(`
		CREATE TABLE IF NOT EXISTS payment_terms (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			description TEXT,
			term_type TEXT NOT NULL DEFAULT 'net_30',
			days_until_due INTEGER,
			discount_percentage REAL,
			discount_days INTEGER,
			late_fee_percentage REAL,
			late_fee_amount REAL,
			grace_period_days INTEGER DEFAULT 0,
			is_default INTEGER DEFAULT 0,
			is_active INTEGER DEFAULT 1,
			created_at DATETIME,
			updated_at DATETIME
		)
	`).Error
	require.NoError(t, err)

	err = db.Exec(`
		CREATE TABLE IF NOT EXISTS tax_rates (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			description TEXT,
			rate REAL NOT NULL,
			tax_type TEXT NOT NULL DEFAULT 'sales_tax',
			region TEXT,
			is_compound INTEGER DEFAULT 0,
			is_active INTEGER DEFAULT 1,
			effective_date DATE,
			expiry_date DATE,
			created_at DATETIME,
			updated_at DATETIME
		)
	`).Error
	require.NoError(t, err)

	err = db.Exec(`
		CREATE TABLE IF NOT EXISTS invoices (
			id TEXT PRIMARY KEY,
			invoice_number TEXT DEFAULT '',
			customer_id TEXT NOT NULL,
			job_id TEXT,
			quote_id TEXT,
			status TEXT NOT NULL DEFAULT 'draft',
			issue_date DATE NOT NULL,
			due_date DATE NOT NULL,
			sent_date DATETIME,
			viewed_date DATETIME,
			paid_date DATETIME,
			subtotal REAL DEFAULT 0,
			tax_amount REAL DEFAULT 0,
			discount_amount REAL DEFAULT 0,
			total_amount REAL DEFAULT 0,
			amount_paid REAL DEFAULT 0,
			payment_term_id TEXT,
			tax_rate_id TEXT,
			po_number TEXT,
			notes TEXT,
			terms_and_conditions TEXT,
			footer_text TEXT,
			created_by TEXT NOT NULL,
			updated_by TEXT,
			created_at DATETIME,
			updated_at DATETIME,
			deleted_at DATETIME
		)
	`).Error
	require.NoError(t, err)

	err = db.Exec(`
		CREATE TABLE IF NOT EXISTS invoice_lines (
			id TEXT PRIMARY KEY,
			invoice_id TEXT NOT NULL,
			description TEXT NOT NULL,
			quantity REAL NOT NULL DEFAULT 1,
			unit_price REAL NOT NULL,
			unit_of_measure TEXT DEFAULT 'each',
			discount_percentage REAL DEFAULT 0,
			discount_amount REAL DEFAULT 0,
			taxable INTEGER DEFAULT 1,
			tax_rate REAL DEFAULT 0,
			tax_amount REAL DEFAULT 0,
			line_total REAL DEFAULT 0,
			line_total_with_tax REAL DEFAULT 0,
			product_id TEXT,
			service_id TEXT,
			sort_order INTEGER DEFAULT 0,
			created_at DATETIME,
			updated_at DATETIME
		)
	`).Error
	require.NoError(t, err)

	err = db.Exec(`
		CREATE TABLE IF NOT EXISTS invoice_payments (
			id TEXT PRIMARY KEY,
			invoice_id TEXT NOT NULL,
			amount REAL NOT NULL,
			payment_method TEXT NOT NULL,
			payment_date DATE NOT NULL,
			reference_number TEXT,
			notes TEXT,
			created_by TEXT NOT NULL,
			created_at DATETIME,
			updated_at DATETIME
		)
	`).Error
	require.NoError(t, err)

	return db
}

// createTestUser creates a test user
func createTestUser(t *testing.T, db *gorm.DB) uuid.UUID {
	user := models.User{
		ID:    uuid.New(),
		Email: "test@example.com",
	}
	require.NoError(t, db.Create(&user).Error)
	return user.ID
}

// createTestPaymentTerm creates a test payment term
func createTestPaymentTerm(t *testing.T, db *gorm.DB) uuid.UUID {
	term := models.PaymentTerm{
		ID:           uuid.New(),
		Name:         "Net 30",
		TermType:     models.PaymentTermNet30,
		DaysUntilDue: 30,
		IsActive:     true,
	}
	require.NoError(t, db.Create(&term).Error)
	return term.ID
}

// createTestTaxRate creates a test tax rate
func createTestTaxRate(t *testing.T, db *gorm.DB) uuid.UUID {
	rate := models.TaxRate{
		ID:       uuid.New(),
		Name:     "CA Sales Tax",
		Rate:     decimal.NewFromFloat(0.0825),
		TaxType:  models.TaxTypeSalesTax,
		Region:   "CA",
		IsActive: true,
	}
	require.NoError(t, db.Create(&rate).Error)
	return rate.ID
}

func TestInvoiceService_CreateInvoice(t *testing.T) {
	db := setupTestDB(t)
	service := NewInvoiceService(db)
	ctx := context.Background()

	userID := createTestUser(t, db)
	customerID := createTestUser(t, db)
	paymentTermID := createTestPaymentTerm(t, db)
	taxRateID := createTestTaxRate(t, db)

	t.Run("Create invoice successfully", func(t *testing.T) {
		invoice := &models.Invoice{
			CustomerID:    customerID,
			PaymentTermID: &paymentTermID,
			TaxRateID:     &taxRateID,
			Lines: []models.InvoiceLine{
				{
					Description: "Test Service",
					Quantity:    decimal.NewFromInt(10),
					UnitPrice:   decimal.NewFromFloat(100.00),
					Taxable:     true,
					SortOrder:   0,
				},
			},
		}

		created, err := service.CreateInvoice(ctx, invoice, userID)

		require.NoError(t, err)
		assert.NotNil(t, created)
		assert.NotEqual(t, uuid.Nil, created.ID)
		assert.Equal(t, customerID, created.CustomerID)
		assert.Equal(t, models.InvoiceStatusDraft, created.Status)
		assert.False(t, created.IssueDate.IsZero())
		assert.False(t, created.DueDate.IsZero())
		assert.Equal(t, userID, created.CreatedBy)
	})

	t.Run("Create invoice without customer ID fails", func(t *testing.T) {
		invoice := &models.Invoice{
			Lines: []models.InvoiceLine{
				{
					Description: "Test Service",
					Quantity:    decimal.NewFromInt(1),
					UnitPrice:   decimal.NewFromFloat(100.00),
				},
			},
		}

		_, err := service.CreateInvoice(ctx, invoice, userID)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "customer_id")
	})

	t.Run("Calculate due date from payment terms", func(t *testing.T) {
		issueDate := time.Now()
		invoice := &models.Invoice{
			CustomerID:    customerID,
			IssueDate:     issueDate,
			PaymentTermID: &paymentTermID,
			Lines: []models.InvoiceLine{
				{
					Description: "Test Service",
					Quantity:    decimal.NewFromInt(1),
					UnitPrice:   decimal.NewFromFloat(100.00),
				},
			},
		}

		created, err := service.CreateInvoice(ctx, invoice, userID)

		require.NoError(t, err)
		expectedDueDate := issueDate.AddDate(0, 0, 30)
		assert.Equal(t, expectedDueDate.Format("2006-01-02"), created.DueDate.Format("2006-01-02"))
	})
}

func TestInvoiceService_GetInvoice(t *testing.T) {
	db := setupTestDB(t)
	service := NewInvoiceService(db)
	ctx := context.Background()

	userID := createTestUser(t, db)
	customerID := createTestUser(t, db)

	// Create test invoice
	invoice := &models.Invoice{
		CustomerID: customerID,
		IssueDate:  time.Now(),
		DueDate:    time.Now().AddDate(0, 0, 30),
		CreatedBy:  userID,
		Status:     models.InvoiceStatusDraft,
	}
	require.NoError(t, db.Create(invoice).Error)

	t.Run("Get existing invoice", func(t *testing.T) {
		retrieved, err := service.GetInvoice(ctx, invoice.ID)

		require.NoError(t, err)
		assert.NotNil(t, retrieved)
		assert.Equal(t, invoice.ID, retrieved.ID)
		assert.Equal(t, invoice.CustomerID, retrieved.CustomerID)
	})

	t.Run("Get non-existent invoice", func(t *testing.T) {
		_, err := service.GetInvoice(ctx, uuid.New())

		assert.Error(t, err)
		assert.Equal(t, ErrInvoiceNotFound, err)
	})
}

func TestInvoiceService_UpdateInvoice(t *testing.T) {
	db := setupTestDB(t)
	service := NewInvoiceService(db)
	ctx := context.Background()

	userID := createTestUser(t, db)
	customerID := createTestUser(t, db)

	// Create test invoice
	invoice := &models.Invoice{
		CustomerID: customerID,
		IssueDate:  time.Now(),
		DueDate:    time.Now().AddDate(0, 0, 30),
		CreatedBy:  userID,
		Status:     models.InvoiceStatusDraft,
		Notes:      "Original notes",
	}
	require.NoError(t, db.Create(invoice).Error)

	t.Run("Update invoice notes", func(t *testing.T) {
		updates := &models.Invoice{
			Notes: "Updated notes",
		}

		updated, err := service.UpdateInvoice(ctx, invoice.ID, updates, userID)

		require.NoError(t, err)
		assert.Equal(t, "Updated notes", updated.Notes)
	})

	t.Run("Cannot update paid invoice", func(t *testing.T) {
		// Mark invoice as paid
		db.Model(&models.Invoice{}).Where("id = ?", invoice.ID).Update("status", models.InvoiceStatusPaid)

		updates := &models.Invoice{
			Notes: "Trying to update",
		}

		_, err := service.UpdateInvoice(ctx, invoice.ID, updates, userID)

		assert.Error(t, err)
		assert.Equal(t, ErrInvoiceAlreadyPaid, err)
	})
}

func TestInvoiceService_DeleteInvoice(t *testing.T) {
	db := setupTestDB(t)
	service := NewInvoiceService(db)
	ctx := context.Background()

	userID := createTestUser(t, db)
	customerID := createTestUser(t, db)

	t.Run("Delete draft invoice", func(t *testing.T) {
		invoice := &models.Invoice{
			CustomerID: customerID,
			IssueDate:  time.Now(),
			DueDate:    time.Now().AddDate(0, 0, 30),
			CreatedBy:  userID,
			Status:     models.InvoiceStatusDraft,
		}
		require.NoError(t, db.Create(invoice).Error)

		err := service.DeleteInvoice(ctx, invoice.ID)

		assert.NoError(t, err)

		// Verify soft delete
		var deleted models.Invoice
		err = db.Unscoped().First(&deleted, invoice.ID).Error
		require.NoError(t, err)
		assert.NotNil(t, deleted.DeletedAt)
	})

	t.Run("Cannot delete paid invoice", func(t *testing.T) {
		invoice := &models.Invoice{
			CustomerID: customerID,
			IssueDate:  time.Now(),
			DueDate:    time.Now().AddDate(0, 0, 30),
			CreatedBy:  userID,
			Status:     models.InvoiceStatusPaid,
		}
		require.NoError(t, db.Create(invoice).Error)

		err := service.DeleteInvoice(ctx, invoice.ID)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "paid")
	})
}

func TestInvoiceService_ListInvoices(t *testing.T) {
	db := setupTestDB(t)
	service := NewInvoiceService(db)
	ctx := context.Background()

	userID := createTestUser(t, db)
	customer1 := createTestUser(t, db)
	customer2 := createTestUser(t, db)

	// Create test invoices
	invoices := []models.Invoice{
		{
			CustomerID:  customer1,
			IssueDate:   time.Now().AddDate(0, 0, -10),
			DueDate:     time.Now().AddDate(0, 0, 20),
			CreatedBy:   userID,
			Status:      models.InvoiceStatusDraft,
			TotalAmount: decimal.NewFromFloat(100.00),
		},
		{
			CustomerID:  customer1,
			IssueDate:   time.Now().AddDate(0, 0, -5),
			DueDate:     time.Now().AddDate(0, 0, 25),
			CreatedBy:   userID,
			Status:      models.InvoiceStatusSent,
			TotalAmount: decimal.NewFromFloat(200.00),
		},
		{
			CustomerID:  customer2,
			IssueDate:   time.Now(),
			DueDate:     time.Now().AddDate(0, 0, 30),
			CreatedBy:   userID,
			Status:      models.InvoiceStatusPaid,
			TotalAmount: decimal.NewFromFloat(300.00),
		},
	}

	for i := range invoices {
		require.NoError(t, db.Create(&invoices[i]).Error)
	}

	t.Run("List all invoices", func(t *testing.T) {
		filter := &models.InvoiceFilter{
			Page:     1,
			PageSize: 10,
		}

		result, err := service.ListInvoices(ctx, filter)

		require.NoError(t, err)
		assert.Equal(t, int64(3), result.Total)
		assert.Len(t, result.Invoices, 3)
	})

	t.Run("Filter by customer", func(t *testing.T) {
		filter := &models.InvoiceFilter{
			CustomerID: &customer1,
			Page:       1,
			PageSize:   10,
		}

		result, err := service.ListInvoices(ctx, filter)

		require.NoError(t, err)
		assert.Equal(t, int64(2), result.Total)
		assert.Len(t, result.Invoices, 2)
	})

	t.Run("Filter by status", func(t *testing.T) {
		status := models.InvoiceStatusPaid
		filter := &models.InvoiceFilter{
			Status:   &status,
			Page:     1,
			PageSize: 10,
		}

		result, err := service.ListInvoices(ctx, filter)

		require.NoError(t, err)
		assert.Equal(t, int64(1), result.Total)
		assert.Len(t, result.Invoices, 1)
		assert.Equal(t, models.InvoiceStatusPaid, result.Invoices[0].Status)
	})

	t.Run("Filter by amount range", func(t *testing.T) {
		minAmount := decimal.NewFromFloat(150.00)
		maxAmount := decimal.NewFromFloat(250.00)
		filter := &models.InvoiceFilter{
			MinAmount: &minAmount,
			MaxAmount: &maxAmount,
			Page:      1,
			PageSize:  10,
		}

		result, err := service.ListInvoices(ctx, filter)

		require.NoError(t, err)
		assert.Equal(t, int64(1), result.Total)
		assert.Len(t, result.Invoices, 1)
		assert.Equal(t, decimal.NewFromFloat(200.00), result.Invoices[0].TotalAmount)
	})

	t.Run("Pagination works", func(t *testing.T) {
		filter := &models.InvoiceFilter{
			Page:     1,
			PageSize: 2,
		}

		result, err := service.ListInvoices(ctx, filter)

		require.NoError(t, err)
		assert.Equal(t, int64(3), result.Total)
		assert.Len(t, result.Invoices, 2)
		assert.Equal(t, 2, result.TotalPages)
	})
}

func TestInvoiceService_RecordPayment(t *testing.T) {
	db := setupTestDB(t)
	service := NewInvoiceService(db)
	ctx := context.Background()

	userID := createTestUser(t, db)
	customerID := createTestUser(t, db)

	invoice := &models.Invoice{
		CustomerID:  customerID,
		IssueDate:   time.Now(),
		DueDate:     time.Now().AddDate(0, 0, 30),
		CreatedBy:   userID,
		Status:      models.InvoiceStatusSent,
		Subtotal:    decimal.NewFromFloat(1000.00),
		TaxAmount:   decimal.NewFromFloat(82.50),
		TotalAmount: decimal.NewFromFloat(1082.50),
		AmountPaid:  decimal.Zero,
	}
	require.NoError(t, db.Create(invoice).Error)

	t.Run("Record full payment", func(t *testing.T) {
		payment := &models.InvoicePayment{
			InvoiceID:     invoice.ID,
			Amount:        decimal.NewFromFloat(1082.50),
			PaymentDate:   time.Now(),
			PaymentMethod: "bank_transfer",
			CreatedBy:     userID,
		}

		created, err := service.RecordPayment(ctx, payment)

		require.NoError(t, err)
		assert.NotNil(t, created)
		assert.Equal(t, invoice.ID, created.InvoiceID)
		assert.Equal(t, decimal.NewFromFloat(1082.50), created.Amount)
	})

	t.Run("Cannot record negative payment", func(t *testing.T) {
		payment := &models.InvoicePayment{
			InvoiceID: invoice.ID,
			Amount:    decimal.NewFromFloat(-100.00),
			CreatedBy: userID,
		}

		_, err := service.RecordPayment(ctx, payment)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "greater than zero")
	})

	t.Run("Cannot record payment exceeding balance", func(t *testing.T) {
		payment := &models.InvoicePayment{
			InvoiceID: invoice.ID,
			Amount:    decimal.NewFromFloat(2000.00),
			CreatedBy: userID,
		}

		_, err := service.RecordPayment(ctx, payment)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "exceeds")
	})
}

func TestInvoiceService_SendInvoice(t *testing.T) {
	db := setupTestDB(t)
	service := NewInvoiceService(db)
	ctx := context.Background()

	userID := createTestUser(t, db)
	customerID := createTestUser(t, db)

	t.Run("Send valid invoice", func(t *testing.T) {
		invoice := &models.Invoice{
			CustomerID:  customerID,
			IssueDate:   time.Now(),
			DueDate:     time.Now().AddDate(0, 0, 30),
			CreatedBy:   userID,
			Status:      models.InvoiceStatusDraft,
			TotalAmount: decimal.NewFromFloat(100.00),
		}
		require.NoError(t, db.Create(invoice).Error)

		// Add line item
		line := &models.InvoiceLine{
			InvoiceID:   invoice.ID,
			Description: "Service",
			Quantity:    decimal.NewFromInt(1),
			UnitPrice:   decimal.NewFromFloat(100.00),
		}
		require.NoError(t, db.Create(line).Error)

		sent, err := service.SendInvoice(ctx, invoice.ID)

		require.NoError(t, err)
		assert.Equal(t, models.InvoiceStatusSent, sent.Status)
		assert.NotNil(t, sent.SentDate)
	})

	t.Run("Cannot send invoice without line items", func(t *testing.T) {
		invoice := &models.Invoice{
			CustomerID:  customerID,
			IssueDate:   time.Now(),
			DueDate:     time.Now().AddDate(0, 0, 30),
			CreatedBy:   userID,
			Status:      models.InvoiceStatusDraft,
			TotalAmount: decimal.Zero,
		}
		require.NoError(t, db.Create(invoice).Error)

		_, err := service.SendInvoice(ctx, invoice.ID)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "line item")
	})
}
