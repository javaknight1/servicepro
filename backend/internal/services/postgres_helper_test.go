package services

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"net/url"
	"os"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/javaknight1/servicepro/backend/internal/tenancy"
	"github.com/javaknight1/servicepro/backend/pkg/database"
)

// pgTestURLEnv names a Postgres server URL (any database on it; the role
// must be able to CREATE DATABASE) for tests that need real row locks,
// constraints and generated columns. Unset, those tests are skipped:
//
//	SERVICEPRO_TEST_PG_URL=postgres://postgres@127.0.0.1:5432/postgres?sslmode=disable \
//	  go test ./internal/services -run Postgres
const pgTestURLEnv = "SERVICEPRO_TEST_PG_URL"

// setupPostgres creates a throwaway database on $SERVICEPRO_TEST_PG_URL,
// applies every embedded migration, installs the tenancy plugin and drops
// the database when the test ends.
func setupPostgres(t *testing.T) *gorm.DB {
	t.Helper()
	base := os.Getenv(pgTestURLEnv)
	if base == "" {
		t.Skipf("%s not set; skipping Postgres-backed test", pgTestURLEnv)
	}

	admin, err := gorm.Open(postgres.Open(base), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	require.NoError(t, err)
	buf := make([]byte, 6)
	_, _ = rand.Read(buf)
	name := "sp_test_" + hex.EncodeToString(buf)
	require.NoError(t, admin.Exec("CREATE DATABASE "+name).Error)

	u, err := url.Parse(base)
	require.NoError(t, err)
	u.Path = "/" + name
	dsn := u.String()
	require.NoError(t, database.RunMigrations(dsn))

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	require.NoError(t, err)
	require.NoError(t, tenancy.Register(db))

	t.Cleanup(func() {
		if sqlDB, err := db.DB(); err == nil {
			_ = sqlDB.Close()
		}
		_ = admin.Exec("DROP DATABASE IF EXISTS " + name + " WITH (FORCE)").Error
		if sqlDB, err := admin.DB(); err == nil {
			_ = sqlDB.Close()
		}
	})
	return db
}

// pgTenant is one seeded tenant: its context, owner user and a customer.
type pgTenant struct {
	ctx        context.Context
	tenantID   uuid.UUID
	userID     uuid.UUID
	customerID uuid.UUID
}

// seedPGTenant inserts a user, a tenant owned by them and one customer.
func seedPGTenant(t *testing.T, db *gorm.DB) *pgTenant {
	t.Helper()
	sys := tenancy.WithSystem(context.Background())
	p := &pgTenant{tenantID: uuid.New(), userID: uuid.New(), customerID: uuid.New()}
	short := p.tenantID.String()[:8]
	require.NoError(t, db.WithContext(sys).Exec(
		`INSERT INTO users (id, email, password_hash) VALUES (?, ?, 'x')`, p.userID, "owner-"+short+"@example.com").Error)
	require.NoError(t, db.WithContext(sys).Exec(
		`INSERT INTO tenants (id, name, slug, owner_id) VALUES (?, ?, ?, ?)`, p.tenantID, "Tenant "+short, "t-"+short, p.userID).Error)
	require.NoError(t, db.WithContext(sys).Exec(
		`INSERT INTO customers (id, tenant_id, first_name, last_name, email, phone_primary,
			billing_address_street, billing_address_city, billing_address_state, billing_address_zip)
		 VALUES (?, ?, 'Pat', 'Customer', ?, '555-555-5555', '1 Main St', 'Springfield', 'IL', '62701')`,
		p.customerID, p.tenantID, "pat-"+short+"@example.com").Error)
	p.ctx = tenancy.WithTenant(context.Background(), p.tenantID)
	return p
}
