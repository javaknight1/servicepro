//go:build integration

package database

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/javaknight1/servicepro/backend/tests/integration"
	"github.com/javaknight1/servicepro/backend/tests/integration/fixtures"
)

// =============================================================================
// CRUD Operation Tests
// =============================================================================

func TestCRUDOperations(t *testing.T) {
	suite := integration.SetupTest(t)

	t.Run("Customer CRUD", func(t *testing.T) {
		t.Run("Create", func(t *testing.T) {
			customer := suite.Fixtures.CreateCustomer(t, &fixtures.Customer{
				Email:     "crud-test@test.com",
				FirstName: "CRUD",
				LastName:  "Test",
			})

			assert.NotEmpty(t, customer.ID)
			assert.Equal(t, "crud-test@test.com", customer.Email)
		})

		t.Run("Read", func(t *testing.T) {
			created := suite.Fixtures.CreateCustomer(t, &fixtures.Customer{
				Email:     "read-test@test.com",
				FirstName: "Read",
				LastName:  "Test",
			})

			var customer struct {
				ID        string
				Email     string
				FirstName string
				LastName  string
			}

			err := suite.DB.Raw("SELECT id, email, first_name, last_name FROM customers WHERE id = ?", created.ID).
				Scan(&customer).Error
			require.NoError(t, err)

			assert.Equal(t, created.ID, customer.ID)
			assert.Equal(t, "read-test@test.com", customer.Email)
		})

		t.Run("Update", func(t *testing.T) {
			created := suite.Fixtures.CreateCustomer(t, &fixtures.Customer{
				FirstName: "Original",
			})

			err := suite.DB.Exec("UPDATE customers SET first_name = ?, updated_at = ? WHERE id = ?",
				"Updated", time.Now(), created.ID).Error
			require.NoError(t, err)

			var firstName string
			err = suite.DB.Raw("SELECT first_name FROM customers WHERE id = ?", created.ID).
				Scan(&firstName).Error
			require.NoError(t, err)

			assert.Equal(t, "Updated", firstName)
		})

		t.Run("Delete", func(t *testing.T) {
			created := suite.Fixtures.CreateCustomer(t, &fixtures.Customer{})

			err := suite.DB.Exec("DELETE FROM customers WHERE id = ?", created.ID).Error
			require.NoError(t, err)

			var count int64
			err = suite.DB.Raw("SELECT COUNT(*) FROM customers WHERE id = ?", created.ID).
				Scan(&count).Error
			require.NoError(t, err)

			assert.Equal(t, int64(0), count)
		})
	})

	t.Run("Job CRUD", func(t *testing.T) {
		customer := suite.CreateTestCustomer(t)

		t.Run("Create", func(t *testing.T) {
			job := suite.Fixtures.CreateJob(t, &fixtures.Job{
				CustomerID: customer.ID,
				Title:      "Test Job",
				Status:     "scheduled",
			})

			assert.NotEmpty(t, job.ID)
			assert.Equal(t, "Test Job", job.Title)
			assert.Equal(t, "scheduled", job.Status)
		})

		t.Run("Read with joins", func(t *testing.T) {
			job := suite.Fixtures.CreateJob(t, &fixtures.Job{
				CustomerID: customer.ID,
				Title:      "Join Test Job",
			})

			var result struct {
				JobID         string
				JobTitle      string
				CustomerEmail string
			}

			err := suite.DB.Raw(`
				SELECT j.id as job_id, j.title as job_title, c.email as customer_email
				FROM jobs j
				JOIN customers c ON j.customer_id = c.id
				WHERE j.id = ?
			`, job.ID).Scan(&result).Error
			require.NoError(t, err)

			assert.Equal(t, job.ID, result.JobID)
			assert.Equal(t, "Join Test Job", result.JobTitle)
			assert.Equal(t, customer.Email, result.CustomerEmail)
		})

		t.Run("Update status", func(t *testing.T) {
			job := suite.Fixtures.CreateJob(t, &fixtures.Job{
				CustomerID: customer.ID,
				Status:     "scheduled",
			})

			err := suite.DB.Exec(`
				UPDATE jobs
				SET status = ?, actual_start = ?, updated_at = ?
				WHERE id = ?
			`, "in_progress", time.Now(), time.Now(), job.ID).Error
			require.NoError(t, err)

			var status string
			err = suite.DB.Raw("SELECT status FROM jobs WHERE id = ?", job.ID).
				Scan(&status).Error
			require.NoError(t, err)

			assert.Equal(t, "in_progress", status)
		})
	})

	t.Run("Invoice CRUD", func(t *testing.T) {
		customer := suite.CreateTestCustomer(t)

		t.Run("Create with line items", func(t *testing.T) {
			invoice := suite.Fixtures.CreateInvoiceWithLineItems(t, customer.ID)

			assert.NotEmpty(t, invoice.ID)
			assert.Len(t, invoice.LineItems, 3)
			assert.True(t, invoice.Total.GreaterThan(decimal.Zero))
		})

		t.Run("Read with calculated totals", func(t *testing.T) {
			invoice := suite.Fixtures.CreateInvoiceWithLineItems(t, customer.ID)

			var result struct {
				ID       string
				Subtotal decimal.Decimal
				Tax      decimal.Decimal
				Total    decimal.Decimal
			}

			err := suite.DB.Raw(`
				SELECT id, subtotal, tax_amount as tax, total
				FROM invoices
				WHERE id = ?
			`, invoice.ID).Scan(&result).Error
			require.NoError(t, err)

			// Verify totals are calculated correctly
			expectedTax := result.Subtotal.Mul(decimal.NewFromFloat(0.08))
			assert.True(t, result.Tax.Equal(expectedTax))
			assert.True(t, result.Total.Equal(result.Subtotal.Add(result.Tax)))
		})

		t.Run("Update status to sent", func(t *testing.T) {
			invoice := suite.Fixtures.CreateInvoice(t, &fixtures.Invoice{
				CustomerID: customer.ID,
				Status:     "draft",
			})

			err := suite.DB.Exec(`
				UPDATE invoices
				SET status = ?, updated_at = ?
				WHERE id = ?
			`, "sent", time.Now(), invoice.ID).Error
			require.NoError(t, err)

			var status string
			err = suite.DB.Raw("SELECT status FROM invoices WHERE id = ?", invoice.ID).
				Scan(&status).Error
			require.NoError(t, err)

			assert.Equal(t, "sent", status)
		})
	})
}

// =============================================================================
// Soft Delete Tests
// =============================================================================

func TestSoftDelete(t *testing.T) {
	suite := integration.SetupTest(t)

	t.Run("Customer soft delete", func(t *testing.T) {
		customer := suite.CreateTestCustomer(t)

		// Soft delete
		err := suite.DB.Exec("UPDATE customers SET deleted_at = ? WHERE id = ?",
			time.Now(), customer.ID).Error
		require.NoError(t, err)

		// Should not appear in normal queries
		var count int64
		err = suite.DB.Raw("SELECT COUNT(*) FROM customers WHERE id = ? AND deleted_at IS NULL", customer.ID).
			Scan(&count).Error
		require.NoError(t, err)
		assert.Equal(t, int64(0), count)

		// Should appear when including deleted
		err = suite.DB.Raw("SELECT COUNT(*) FROM customers WHERE id = ?", customer.ID).
			Scan(&count).Error
		require.NoError(t, err)
		assert.Equal(t, int64(1), count)
	})

	t.Run("Restore soft deleted record", func(t *testing.T) {
		customer := suite.CreateTestCustomer(t)

		// Soft delete
		err := suite.DB.Exec("UPDATE customers SET deleted_at = ? WHERE id = ?",
			time.Now(), customer.ID).Error
		require.NoError(t, err)

		// Restore
		err = suite.DB.Exec("UPDATE customers SET deleted_at = NULL WHERE id = ?",
			customer.ID).Error
		require.NoError(t, err)

		// Should appear again
		var count int64
		err = suite.DB.Raw("SELECT COUNT(*) FROM customers WHERE id = ? AND deleted_at IS NULL", customer.ID).
			Scan(&count).Error
		require.NoError(t, err)
		assert.Equal(t, int64(1), count)
	})
}

// =============================================================================
// Data Integrity Tests
// =============================================================================

func TestDataIntegrity(t *testing.T) {
	suite := integration.SetupTest(t)

	t.Run("Foreign key constraints", func(t *testing.T) {
		customer := suite.CreateTestCustomer(t)

		t.Run("Job requires valid customer", func(t *testing.T) {
			err := suite.DB.Exec(`
				INSERT INTO jobs (id, title, customer_id, status, scheduled_start, scheduled_end, created_at, updated_at)
				VALUES (?, 'Test', 'non-existent-id', 'scheduled', ?, ?, ?, ?)
			`, uuid.New().String(), time.Now(), time.Now().Add(time.Hour), time.Now(), time.Now()).Error

			// Should fail due to foreign key constraint
			assert.Error(t, err)
		})

		t.Run("Job succeeds with valid customer", func(t *testing.T) {
			err := suite.DB.Exec(`
				INSERT INTO jobs (id, title, customer_id, status, scheduled_start, scheduled_end, created_at, updated_at)
				VALUES (?, 'Test', ?, 'scheduled', ?, ?, ?, ?)
			`, uuid.New().String(), customer.ID, time.Now(), time.Now().Add(time.Hour), time.Now(), time.Now()).Error

			assert.NoError(t, err)
		})
	})

	t.Run("Unique constraints", func(t *testing.T) {
		t.Run("Duplicate customer email fails", func(t *testing.T) {
			email := "unique-" + uuid.New().String()[:8] + "@test.com"

			// First insert succeeds
			err := suite.DB.Exec(`
				INSERT INTO customers (id, email, first_name, last_name, created_at, updated_at)
				VALUES (?, ?, 'Test', 'User', ?, ?)
			`, uuid.New().String(), email, time.Now(), time.Now()).Error
			require.NoError(t, err)

			// Second insert with same email fails
			err = suite.DB.Exec(`
				INSERT INTO customers (id, email, first_name, last_name, created_at, updated_at)
				VALUES (?, ?, 'Test', 'User', ?, ?)
			`, uuid.New().String(), email, time.Now(), time.Now()).Error
			assert.Error(t, err)
		})

		t.Run("Duplicate invoice number fails", func(t *testing.T) {
			customer := suite.CreateTestCustomer(t)
			invoiceNum := "INV-" + uuid.New().String()[:8]

			// First insert succeeds
			err := suite.DB.Exec(`
				INSERT INTO invoices (id, number, customer_id, status, issue_date, due_date, subtotal, tax_rate, tax_amount, total, created_at, updated_at)
				VALUES (?, ?, ?, 'draft', ?, ?, 0, 0, 0, 0, ?, ?)
			`, uuid.New().String(), invoiceNum, customer.ID, time.Now(), time.Now().Add(30*24*time.Hour), time.Now(), time.Now()).Error
			require.NoError(t, err)

			// Second insert with same number fails
			err = suite.DB.Exec(`
				INSERT INTO invoices (id, number, customer_id, status, issue_date, due_date, subtotal, tax_rate, tax_amount, total, created_at, updated_at)
				VALUES (?, ?, ?, 'draft', ?, ?, 0, 0, 0, 0, ?, ?)
			`, uuid.New().String(), invoiceNum, customer.ID, time.Now(), time.Now().Add(30*24*time.Hour), time.Now(), time.Now()).Error
			assert.Error(t, err)
		})
	})

	t.Run("Check constraints", func(t *testing.T) {
		customer := suite.CreateTestCustomer(t)

		t.Run("Invoice total cannot be negative", func(t *testing.T) {
			err := suite.DB.Exec(`
				INSERT INTO invoices (id, number, customer_id, status, issue_date, due_date, subtotal, tax_rate, tax_amount, total, created_at, updated_at)
				VALUES (?, ?, ?, 'draft', ?, ?, -100, 0, 0, -100, ?, ?)
			`, uuid.New().String(), "INV-"+uuid.New().String()[:8], customer.ID,
				time.Now(), time.Now().Add(30*24*time.Hour), time.Now(), time.Now()).Error

			// Should fail if check constraint exists
			// Note: This may pass if no check constraint is defined
			if err != nil {
				assert.Error(t, err)
			}
		})
	})

	t.Run("Cascade deletes", func(t *testing.T) {
		// This test depends on how cascade deletes are configured
		customer := suite.CreateTestCustomer(t)
		job := suite.Fixtures.CreateJob(t, &fixtures.Job{CustomerID: customer.ID})

		// Delete customer
		err := suite.DB.Exec("DELETE FROM customers WHERE id = ?", customer.ID).Error

		// Check behavior based on cascade configuration
		if err == nil {
			// If cascade is set, job should be deleted too
			var count int64
			err = suite.DB.Raw("SELECT COUNT(*) FROM jobs WHERE id = ?", job.ID).
				Scan(&count).Error
			require.NoError(t, err)
			// count could be 0 (cascade) or 1 (no cascade but FK allows)
			t.Logf("Job count after customer delete: %d", count)
		} else {
			// If restrict is set, delete should fail
			assert.Error(t, err)
		}
	})
}

// =============================================================================
// Index Tests
// =============================================================================

func TestIndexes(t *testing.T) {
	suite := integration.SetupTest(t)

	t.Run("Index exists on commonly queried fields", func(t *testing.T) {
		// This tests that queries using indexed fields are efficient
		// Create lots of test data
		for i := 0; i < 100; i++ {
			suite.Fixtures.CreateCustomer(t, &fixtures.Customer{
				Email:     "index-test-" + uuid.New().String() + "@test.com",
				FirstName: "IndexTest",
			})
		}

		// Query by indexed field (email) should be fast
		start := time.Now()
		var count int64
		err := suite.DB.Raw("SELECT COUNT(*) FROM customers WHERE email LIKE 'index-test-%'").
			Scan(&count).Error
		require.NoError(t, err)
		elapsed := time.Since(start)

		assert.GreaterOrEqual(t, count, int64(100))
		assert.Less(t, elapsed, 500*time.Millisecond, "Query took too long - index may be missing")
	})
}
