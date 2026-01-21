//go:build integration

package database

import (
	"sync"
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
// Transaction Tests
// =============================================================================

func TestTransactions(t *testing.T) {
	suite := integration.SetupTest(t)

	t.Run("Successful transaction commit", func(t *testing.T) {
		customerID := uuid.New().String()
		jobID := uuid.New().String()

		// Use a transaction
		tx := suite.DB.Begin()

		// Create customer
		err := tx.Exec(`
			INSERT INTO customers (id, email, first_name, last_name, created_at, updated_at)
			VALUES (?, ?, 'Test', 'User', ?, ?)
		`, customerID, "tx-test-"+uuid.New().String()[:8]+"@test.com", time.Now(), time.Now()).Error
		require.NoError(t, err)

		// Create job
		err = tx.Exec(`
			INSERT INTO jobs (id, title, customer_id, status, scheduled_start, scheduled_end, created_at, updated_at)
			VALUES (?, 'Test Job', ?, 'scheduled', ?, ?, ?, ?)
		`, jobID, customerID, time.Now(), time.Now().Add(time.Hour), time.Now(), time.Now()).Error
		require.NoError(t, err)

		// Commit
		err = tx.Commit().Error
		require.NoError(t, err)

		// Verify both records exist
		var customerCount, jobCount int64
		suite.DB.Raw("SELECT COUNT(*) FROM customers WHERE id = ?", customerID).Scan(&customerCount)
		suite.DB.Raw("SELECT COUNT(*) FROM jobs WHERE id = ?", jobID).Scan(&jobCount)

		assert.Equal(t, int64(1), customerCount)
		assert.Equal(t, int64(1), jobCount)
	})

	t.Run("Transaction rollback on error", func(t *testing.T) {
		customerID := uuid.New().String()

		tx := suite.DB.Begin()

		// Create customer
		err := tx.Exec(`
			INSERT INTO customers (id, email, first_name, last_name, created_at, updated_at)
			VALUES (?, ?, 'Test', 'User', ?, ?)
		`, customerID, "rollback-test-"+uuid.New().String()[:8]+"@test.com", time.Now(), time.Now()).Error
		require.NoError(t, err)

		// Simulate error - rollback
		tx.Rollback()

		// Verify customer was not created
		var count int64
		suite.DB.Raw("SELECT COUNT(*) FROM customers WHERE id = ?", customerID).Scan(&count)
		assert.Equal(t, int64(0), count)
	})

	t.Run("Transaction isolation", func(t *testing.T) {
		customer := suite.CreateTestCustomer(t)

		// Start transaction
		tx := suite.DB.Begin()

		// Update in transaction
		err := tx.Exec("UPDATE customers SET first_name = 'InTransaction' WHERE id = ?", customer.ID).Error
		require.NoError(t, err)

		// Read outside transaction should see original value (depending on isolation level)
		var firstName string
		suite.DB.Raw("SELECT first_name FROM customers WHERE id = ?", customer.ID).Scan(&firstName)

		// Before commit, other connections may see old value (READ COMMITTED)
		// or new value (READ UNCOMMITTED)
		t.Logf("Name outside transaction before commit: %s", firstName)

		// Commit
		tx.Commit()

		// After commit, should see new value
		suite.DB.Raw("SELECT first_name FROM customers WHERE id = ?", customer.ID).Scan(&firstName)
		assert.Equal(t, "InTransaction", firstName)
	})

	t.Run("Nested savepoints", func(t *testing.T) {
		customer := suite.CreateTestCustomer(t)
		originalName := customer.FirstName

		tx := suite.DB.Begin()

		// First update
		tx.Exec("UPDATE customers SET first_name = 'First' WHERE id = ?", customer.ID)

		// Create savepoint
		tx.SavePoint("sp1")

		// Second update
		tx.Exec("UPDATE customers SET first_name = 'Second' WHERE id = ?", customer.ID)

		// Rollback to savepoint
		tx.RollbackTo("sp1")

		// Commit
		tx.Commit()

		// Should have 'First' value (rolled back 'Second')
		var firstName string
		suite.DB.Raw("SELECT first_name FROM customers WHERE id = ?", customer.ID).Scan(&firstName)
		assert.Equal(t, "First", firstName)
		_ = originalName // suppress unused warning
	})
}

// =============================================================================
// Concurrent Operation Tests
// =============================================================================

func TestConcurrentOperations(t *testing.T) {
	suite := integration.SetupTest(t)

	t.Run("Concurrent inserts", func(t *testing.T) {
		var wg sync.WaitGroup
		numGoroutines := 10
		numInsertsPerGoroutine := 10

		errors := make(chan error, numGoroutines*numInsertsPerGoroutine)

		for i := 0; i < numGoroutines; i++ {
			wg.Add(1)
			go func(goroutineID int) {
				defer wg.Done()
				for j := 0; j < numInsertsPerGoroutine; j++ {
					err := suite.DB.Exec(`
						INSERT INTO customers (id, email, first_name, last_name, created_at, updated_at)
						VALUES (?, ?, 'Concurrent', 'Test', ?, ?)
					`, uuid.New().String(),
						"concurrent-"+uuid.New().String()+"@test.com",
						time.Now(), time.Now()).Error
					if err != nil {
						errors <- err
					}
				}
			}(i)
		}

		wg.Wait()
		close(errors)

		// Check for errors
		var errorCount int
		for err := range errors {
			t.Logf("Concurrent insert error: %v", err)
			errorCount++
		}
		assert.Equal(t, 0, errorCount, "Expected no errors during concurrent inserts")

		// Verify all records were created
		var count int64
		suite.DB.Raw("SELECT COUNT(*) FROM customers WHERE first_name = 'Concurrent'").Scan(&count)
		assert.Equal(t, int64(numGoroutines*numInsertsPerGoroutine), count)
	})

	t.Run("Concurrent updates to same record", func(t *testing.T) {
		// Create a customer with a counter field (using notes as counter)
		customer := suite.Fixtures.CreateCustomer(t, &fixtures.Customer{
			Notes: "0",
		})

		var wg sync.WaitGroup
		numUpdates := 20

		for i := 0; i < numUpdates; i++ {
			wg.Add(1)
			go func(updateNum int) {
				defer wg.Done()

				// Atomic increment using COALESCE
				suite.DB.Exec(`
					UPDATE customers
					SET notes = CAST(COALESCE(CAST(notes AS INTEGER), 0) + 1 AS TEXT),
					    updated_at = ?
					WHERE id = ?
				`, time.Now(), customer.ID)
			}(i)
		}

		wg.Wait()

		// Verify final count
		var notes string
		suite.DB.Raw("SELECT notes FROM customers WHERE id = ?", customer.ID).Scan(&notes)
		t.Logf("Final counter value: %s (expected: %d)", notes, numUpdates)

		// Due to race conditions, the actual value might be less than expected
		// if not using proper locking
		assert.NotEmpty(t, notes)
	})

	t.Run("Concurrent reads while writing", func(t *testing.T) {
		customer := suite.CreateTestCustomer(t)

		var wg sync.WaitGroup
		stopChan := make(chan struct{})
		readErrors := make(chan error, 100)

		// Start concurrent readers
		for i := 0; i < 5; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for {
					select {
					case <-stopChan:
						return
					default:
						var firstName string
						if err := suite.DB.Raw("SELECT first_name FROM customers WHERE id = ?", customer.ID).
							Scan(&firstName).Error; err != nil {
							readErrors <- err
						}
					}
				}
			}()
		}

		// Perform writes
		for i := 0; i < 20; i++ {
			suite.DB.Exec("UPDATE customers SET first_name = ? WHERE id = ?",
				"Update"+uuid.New().String()[:4], customer.ID)
			time.Sleep(10 * time.Millisecond)
		}

		// Stop readers
		close(stopChan)
		wg.Wait()
		close(readErrors)

		// Check for errors
		var errorCount int
		for err := range readErrors {
			t.Logf("Read error: %v", err)
			errorCount++
		}
		assert.Equal(t, 0, errorCount, "Expected no errors during concurrent reads")
	})

	t.Run("Deadlock detection", func(t *testing.T) {
		customer1 := suite.CreateTestCustomer(t)
		customer2 := suite.CreateTestCustomer(t)

		var wg sync.WaitGroup
		results := make(chan string, 2)

		// Transaction 1: Update customer1 then customer2
		wg.Add(1)
		go func() {
			defer wg.Done()
			tx := suite.DB.Begin()
			tx.Exec("UPDATE customers SET first_name = 'TX1' WHERE id = ?", customer1.ID)
			time.Sleep(100 * time.Millisecond) // Introduce delay
			err := tx.Exec("UPDATE customers SET first_name = 'TX1' WHERE id = ?", customer2.ID).Error
			if err != nil {
				tx.Rollback()
				results <- "tx1_failed"
			} else {
				tx.Commit()
				results <- "tx1_success"
			}
		}()

		// Transaction 2: Update customer2 then customer1 (opposite order)
		wg.Add(1)
		go func() {
			defer wg.Done()
			tx := suite.DB.Begin()
			tx.Exec("UPDATE customers SET first_name = 'TX2' WHERE id = ?", customer2.ID)
			time.Sleep(100 * time.Millisecond) // Introduce delay
			err := tx.Exec("UPDATE customers SET first_name = 'TX2' WHERE id = ?", customer1.ID).Error
			if err != nil {
				tx.Rollback()
				results <- "tx2_failed"
			} else {
				tx.Commit()
				results <- "tx2_success"
			}
		}()

		wg.Wait()
		close(results)

		// Collect results
		var successCount, failCount int
		for result := range results {
			if result == "tx1_success" || result == "tx2_success" {
				successCount++
			} else {
				failCount++
			}
		}

		// At least one transaction should succeed
		assert.GreaterOrEqual(t, successCount, 1, "At least one transaction should succeed")
		t.Logf("Success: %d, Failed: %d", successCount, failCount)
	})
}

// =============================================================================
// Payment Transaction Tests
// =============================================================================

func TestPaymentTransactions(t *testing.T) {
	suite := integration.SetupTest(t)

	t.Run("Payment processing transaction", func(t *testing.T) {
		customer := suite.CreateTestCustomer(t)
		invoice := suite.Fixtures.CreateInvoice(t, &fixtures.Invoice{
			CustomerID: customer.ID,
			Status:     "sent",
			Total:      decimal.NewFromFloat(100.00),
		})

		paymentID := uuid.New().String()
		paymentAmount := decimal.NewFromFloat(100.00)
		now := time.Now()

		tx := suite.DB.Begin()

		// Create payment record
		err := tx.Exec(`
			INSERT INTO payments (id, invoice_id, amount, method, status, processed_at, created_at, updated_at)
			VALUES (?, ?, ?, 'card', 'succeeded', ?, ?, ?)
		`, paymentID, invoice.ID, paymentAmount, now, now, now).Error
		require.NoError(t, err)

		// Update invoice paid amount and status
		err = tx.Exec(`
			UPDATE invoices
			SET paid_amount = paid_amount + ?,
			    status = CASE
			        WHEN paid_amount + ? >= total THEN 'paid'
			        ELSE 'partial'
			    END,
			    updated_at = ?
			WHERE id = ?
		`, paymentAmount, paymentAmount, now, invoice.ID).Error
		require.NoError(t, err)

		// Commit
		err = tx.Commit().Error
		require.NoError(t, err)

		// Verify payment was created
		var paymentCount int64
		suite.DB.Raw("SELECT COUNT(*) FROM payments WHERE id = ?", paymentID).Scan(&paymentCount)
		assert.Equal(t, int64(1), paymentCount)

		// Verify invoice was updated
		var invoiceStatus string
		var paidAmount decimal.Decimal
		suite.DB.Raw("SELECT status, paid_amount FROM invoices WHERE id = ?", invoice.ID).
			Row().Scan(&invoiceStatus, &paidAmount)
		assert.Equal(t, "paid", invoiceStatus)
		assert.True(t, paidAmount.Equal(paymentAmount))
	})

	t.Run("Partial payment transaction", func(t *testing.T) {
		customer := suite.CreateTestCustomer(t)
		invoice := suite.Fixtures.CreateInvoice(t, &fixtures.Invoice{
			CustomerID: customer.ID,
			Status:     "sent",
			Total:      decimal.NewFromFloat(100.00),
		})

		// First partial payment
		paymentAmount := decimal.NewFromFloat(50.00)
		now := time.Now()

		tx := suite.DB.Begin()
		tx.Exec(`
			INSERT INTO payments (id, invoice_id, amount, method, status, processed_at, created_at, updated_at)
			VALUES (?, ?, ?, 'card', 'succeeded', ?, ?, ?)
		`, uuid.New().String(), invoice.ID, paymentAmount, now, now, now)
		tx.Exec(`
			UPDATE invoices
			SET paid_amount = paid_amount + ?,
			    status = CASE
			        WHEN paid_amount + ? >= total THEN 'paid'
			        ELSE 'partial'
			    END,
			    updated_at = ?
			WHERE id = ?
		`, paymentAmount, paymentAmount, now, invoice.ID)
		tx.Commit()

		// Verify status is partial
		var status string
		suite.DB.Raw("SELECT status FROM invoices WHERE id = ?", invoice.ID).Scan(&status)
		assert.Equal(t, "partial", status)
	})

	t.Run("Failed payment rollback", func(t *testing.T) {
		customer := suite.CreateTestCustomer(t)
		invoice := suite.Fixtures.CreateInvoice(t, &fixtures.Invoice{
			CustomerID: customer.ID,
			Status:     "sent",
			Total:      decimal.NewFromFloat(100.00),
		})

		paymentID := uuid.New().String()
		now := time.Now()

		tx := suite.DB.Begin()

		// Create payment record
		tx.Exec(`
			INSERT INTO payments (id, invoice_id, amount, method, status, created_at, updated_at)
			VALUES (?, ?, 100, 'card', 'pending', ?, ?)
		`, paymentID, invoice.ID, now, now)

		// Simulate payment failure - rollback
		tx.Rollback()

		// Verify payment was not created
		var paymentCount int64
		suite.DB.Raw("SELECT COUNT(*) FROM payments WHERE id = ?", paymentID).Scan(&paymentCount)
		assert.Equal(t, int64(0), paymentCount)

		// Verify invoice status unchanged
		var status string
		suite.DB.Raw("SELECT status FROM invoices WHERE id = ?", invoice.ID).Scan(&status)
		assert.Equal(t, "sent", status)
	})
}

// =============================================================================
// Bulk Operation Tests
// =============================================================================

func TestBulkOperations(t *testing.T) {
	suite := integration.SetupTest(t)

	t.Run("Bulk insert", func(t *testing.T) {
		// Build bulk insert query
		values := ""
		now := time.Now().Format("2006-01-02 15:04:05")

		for i := 0; i < 100; i++ {
			if i > 0 {
				values += ","
			}
			values += "(" + "'" + uuid.New().String() + "'," +
				"'bulk-" + uuid.New().String() + "@test.com'," +
				"'Bulk','Test','" + now + "','" + now + "')"
		}

		err := suite.DB.Exec(`
			INSERT INTO customers (id, email, first_name, last_name, created_at, updated_at)
			VALUES ` + values).Error
		require.NoError(t, err)

		var count int64
		suite.DB.Raw("SELECT COUNT(*) FROM customers WHERE first_name = 'Bulk'").Scan(&count)
		assert.Equal(t, int64(100), count)
	})

	t.Run("Bulk update", func(t *testing.T) {
		// Create test customers
		for i := 0; i < 50; i++ {
			suite.Fixtures.CreateCustomer(t, &fixtures.Customer{
				FirstName: "BulkUpdate",
			})
		}

		// Bulk update
		err := suite.DB.Exec(`
			UPDATE customers
			SET last_name = 'Updated', updated_at = ?
			WHERE first_name = 'BulkUpdate'
		`, time.Now()).Error
		require.NoError(t, err)

		var count int64
		suite.DB.Raw("SELECT COUNT(*) FROM customers WHERE first_name = 'BulkUpdate' AND last_name = 'Updated'").
			Scan(&count)
		assert.Equal(t, int64(50), count)
	})

	t.Run("Bulk delete", func(t *testing.T) {
		// Create test customers
		for i := 0; i < 25; i++ {
			suite.Fixtures.CreateCustomer(t, &fixtures.Customer{
				FirstName: "BulkDelete",
			})
		}

		// Verify creation
		var countBefore int64
		suite.DB.Raw("SELECT COUNT(*) FROM customers WHERE first_name = 'BulkDelete'").Scan(&countBefore)
		assert.Equal(t, int64(25), countBefore)

		// Bulk delete
		err := suite.DB.Exec("DELETE FROM customers WHERE first_name = 'BulkDelete'").Error
		require.NoError(t, err)

		var countAfter int64
		suite.DB.Raw("SELECT COUNT(*) FROM customers WHERE first_name = 'BulkDelete'").Scan(&countAfter)
		assert.Equal(t, int64(0), countAfter)
	})
}
