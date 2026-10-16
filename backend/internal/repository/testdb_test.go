package repository

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/javaknight1/servicepro/backend/config"
	"github.com/javaknight1/servicepro/backend/internal/tenancy"
	"github.com/javaknight1/servicepro/backend/pkg/database"
)

// testDatabaseURL is the Postgres the repository tests use: TEST_DATABASE_URL
// (set by CI, which migrates it first), else the local docker-compose test db.
func testDatabaseURL() string {
	if u := os.Getenv("TEST_DATABASE_URL"); u != "" {
		return u
	}
	return "postgres://postgres:postgres@localhost:5432/servicepro_test?sslmode=disable"
}

// adminRoleID is the built-in "admin" role seeded by the baseline migration.
var adminRoleID = uuid.MustParse("00000000-0000-0000-0000-000000000002")

// openTestDB connects to the migrated test database and installs the tenancy
// plugin, exactly as cmd/main.go does. It skips when no database is reachable.
func openTestDB(tb testing.TB) *gorm.DB {
	tb.Helper()
	cfg := &config.DatabaseConfig{
		URL:             testDatabaseURL(),
		MaxOpenConns:    10,
		MaxIdleConns:    2,
		ConnMaxLifetime: time.Hour,
	}
	db, _, err := database.NewGormDB(cfg)
	if err != nil {
		tb.Skipf("Skipping - database not available: %v", err)
	}
	tb.Cleanup(func() {
		if sqlDB, err := db.DB(); err == nil {
			_ = sqlDB.Close()
		}
	})
	require.NoError(tb, tenancy.Register(db))
	return db
}

// testTenant is a freshly seeded tenant: its owner, the members added with
// addMember, and a context scoped to it (what the tenant middleware produces).
type testTenant struct {
	ID      uuid.UUID
	OwnerID uuid.UUID
	Ctx     context.Context
	users   []uuid.UUID
}

// createTestTenant inserts an owner user, a tenant and the owner's membership.
// Everything it (and the tenant's rows) created is removed when the test ends.
func createTestTenant(tb testing.TB, db *gorm.DB) *testTenant {
	tb.Helper()
	tt := &testTenant{ID: uuid.New()}
	short := tt.ID.String()[:8]
	require.NoError(tb, db.Exec(`INSERT INTO tenants (id, name, slug, owner_id) VALUES (?, ?, ?, ?)`,
		tt.ID, "Tenant "+short, "t-"+short, tt.newUser(tb, db, "owner")).Error)
	tt.OwnerID = tt.users[0]
	tt.addMembership(tb, db, tt.OwnerID)
	tt.Ctx = tenancy.WithTenant(context.Background(), tt.ID)

	tb.Cleanup(func() {
		// jobs.customer_id and jobs.created_by are ON DELETE RESTRICT, so jobs
		// go first; the tenant delete cascades to customers, memberships and
		// counters. Raw SQL is not tenant-rewritten.
		_ = db.Exec(`DELETE FROM jobs WHERE tenant_id = ?`, tt.ID).Error
		_ = db.Exec(`DELETE FROM tenants WHERE id = ?`, tt.ID).Error
		_ = db.Exec(`DELETE FROM users WHERE id IN ?`, tt.users).Error
	})
	return tt
}

// addMember creates a user with an active membership in the tenant.
func (tt *testTenant) addMember(tb testing.TB, db *gorm.DB) uuid.UUID {
	tb.Helper()
	id := tt.newUser(tb, db, "member")
	tt.addMembership(tb, db, id)
	return id
}

func (tt *testTenant) newUser(tb testing.TB, db *gorm.DB, kind string) uuid.UUID {
	tb.Helper()
	id := uuid.New()
	require.NoError(tb, db.Exec(`INSERT INTO users (id, email, password_hash) VALUES (?, ?, 'x')`,
		id, kind+"-"+id.String()[:12]+"@example.com").Error)
	tt.users = append(tt.users, id)
	return id
}

func (tt *testTenant) addMembership(tb testing.TB, db *gorm.DB, userID uuid.UUID) {
	tb.Helper()
	require.NoError(tb, db.Exec(`INSERT INTO tenant_users (tenant_id, user_id, role_id, is_active) VALUES (?, ?, ?, true)`,
		tt.ID, userID, adminRoleID).Error)
}
