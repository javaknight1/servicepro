//go:build integration

package helpers

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// =============================================================================
// Test Database Manager
// =============================================================================

// TestDB manages the test database connection
type TestDB struct {
	*gorm.DB
	sqlDB      *sql.DB
	migrations []string
}

var (
	testDB     *TestDB
	testDBOnce sync.Once
	testDBMu   sync.Mutex
)

// NewTestDB creates a new test database connection
func NewTestDB(t *testing.T, databaseURL string) *TestDB {
	t.Helper()

	db, err := gorm.Open(postgres.Open(databaseURL), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("Failed to connect to test database: %v", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("Failed to get underlying sql.DB: %v", err)
	}

	// Configure connection pool for testing
	sqlDB.SetMaxOpenConns(10)
	sqlDB.SetMaxIdleConns(5)

	return &TestDB{
		DB:    db,
		sqlDB: sqlDB,
	}
}

// GetSharedTestDB returns a shared test database connection (singleton)
func GetSharedTestDB(t *testing.T, databaseURL string) *TestDB {
	t.Helper()

	testDBOnce.Do(func() {
		testDB = NewTestDB(t, databaseURL)
	})
	return testDB
}

// Close closes the database connection
func (db *TestDB) Close() error {
	if db.sqlDB != nil {
		return db.sqlDB.Close()
	}
	return nil
}

// =============================================================================
// Migration Management
// =============================================================================

// RunMigrations runs all migration files in order
func (db *TestDB) RunMigrations(t *testing.T, migrationsPath string) {
	t.Helper()
	testDBMu.Lock()
	defer testDBMu.Unlock()

	// Find all migration files
	files, err := filepath.Glob(filepath.Join(migrationsPath, "*.sql"))
	if err != nil {
		t.Fatalf("Failed to find migration files: %v", err)
	}

	// Sort migrations by filename (they should be numbered)
	sort.Strings(files)

	// Run each migration
	for _, file := range files {
		// Skip rollback migrations
		if strings.Contains(file, "_rollback") {
			continue
		}

		content, err := os.ReadFile(file)
		if err != nil {
			t.Fatalf("Failed to read migration file %s: %v", file, err)
		}

		if err := db.Exec(string(content)).Error; err != nil {
			// Log but don't fail - migration might already be applied
			t.Logf("Migration %s: %v", filepath.Base(file), err)
		}

		db.migrations = append(db.migrations, file)
	}
}

// =============================================================================
// Test Data Management
// =============================================================================

// CleanAllTables truncates all tables for a fresh test state
func (db *TestDB) CleanAllTables(t *testing.T) {
	t.Helper()
	testDBMu.Lock()
	defer testDBMu.Unlock()

	// Get all table names
	var tables []string
	err := db.Raw(`
		SELECT tablename FROM pg_tables
		WHERE schemaname = 'public'
		AND tablename NOT LIKE 'pg_%'
		AND tablename NOT IN ('schema_migrations', 'goose_db_version')
	`).Scan(&tables).Error
	if err != nil {
		t.Fatalf("Failed to get table names: %v", err)
	}

	// Disable foreign key checks temporarily
	db.Exec("SET session_replication_role = 'replica'")
	defer db.Exec("SET session_replication_role = 'origin'")

	// Truncate all tables
	for _, table := range tables {
		if err := db.Exec(fmt.Sprintf("TRUNCATE TABLE %s CASCADE", table)).Error; err != nil {
			t.Logf("Warning: Failed to truncate table %s: %v", table, err)
		}
	}
}

// CleanTables truncates specific tables
func (db *TestDB) CleanTables(t *testing.T, tables ...string) {
	t.Helper()
	testDBMu.Lock()
	defer testDBMu.Unlock()

	db.Exec("SET session_replication_role = 'replica'")
	defer db.Exec("SET session_replication_role = 'origin'")

	for _, table := range tables {
		if err := db.Exec(fmt.Sprintf("TRUNCATE TABLE %s CASCADE", table)).Error; err != nil {
			t.Logf("Warning: Failed to truncate table %s: %v", table, err)
		}
	}
}

// ResetSequences resets all sequences to 1
func (db *TestDB) ResetSequences(t *testing.T) {
	t.Helper()

	var sequences []string
	err := db.Raw(`
		SELECT sequence_name FROM information_schema.sequences
		WHERE sequence_schema = 'public'
	`).Scan(&sequences).Error
	if err != nil {
		t.Logf("Warning: Failed to get sequences: %v", err)
		return
	}

	for _, seq := range sequences {
		db.Exec(fmt.Sprintf("ALTER SEQUENCE %s RESTART WITH 1", seq))
	}
}

// =============================================================================
// Transaction Support for Test Isolation
// =============================================================================

// WithTransaction runs a test function within a transaction that gets rolled back
func (db *TestDB) WithTransaction(t *testing.T, fn func(tx *gorm.DB)) {
	t.Helper()

	tx := db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
			panic(r)
		}
		tx.Rollback()
	}()

	fn(tx)
}

// =============================================================================
// Test Context
// =============================================================================

// TestContext returns a context with test timeout
func TestContext(t *testing.T) context.Context {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	t.Cleanup(cancel)
	return ctx
}
