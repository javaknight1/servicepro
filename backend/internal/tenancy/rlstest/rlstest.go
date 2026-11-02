// Package rlstest gives DB-backed tests a throwaway, fully migrated Postgres
// database and a login role that is subject to its row-level-security
// policies (T154). Test code only.
package rlstest

import (
	"crypto/rand"
	"encoding/hex"
	"net/url"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/javaknight1/servicepro/backend/config"
	"github.com/javaknight1/servicepro/backend/internal/tenancy"
	"github.com/javaknight1/servicepro/backend/pkg/database"
)

// Env is one throwaway database.
type Env struct {
	// Owner connects as the server's (superuser) test role: RLS never applies
	// to it. Use it to seed and to check what is really in the tables.
	Owner *gorm.DB
	// AppURL connects as a fresh login role in servicepro_app, which the
	// policies bind.
	AppURL string
}

// serverURL is the Postgres the throwaway database is created on:
// TEST_DATABASE_URL (any database on it; its role must be able to create
// databases and roles), else the local docker-compose server.
func serverURL() string {
	if u := os.Getenv("TEST_DATABASE_URL"); u != "" {
		return u
	}
	return "postgres://postgres:postgres@localhost:5432/servicepro_test?sslmode=disable"
}

func random(tb testing.TB) string {
	buf := make([]byte, 6)
	_, err := rand.Read(buf)
	require.NoError(tb, err)
	return hex.EncodeToString(buf)
}

// Setup creates and migrates a database plus an RLS-bound login role, and
// drops both when the test ends. It skips when no server is reachable.
func Setup(tb testing.TB) *Env {
	tb.Helper()
	base := serverURL()
	admin, err := gorm.Open(postgres.Open(base), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err == nil {
		if sqlDB, e := admin.DB(); e == nil {
			err = sqlDB.Ping()
		}
	}
	if err != nil {
		tb.Skipf("Skipping - database not available: %v", err)
	}
	var canCreate bool
	require.NoError(tb, admin.Raw(`SELECT rolsuper OR (rolcreatedb AND rolcreaterole) FROM pg_roles WHERE rolname = current_user`).Scan(&canCreate).Error)
	if !canCreate {
		tb.Skip("Skipping - TEST_DATABASE_URL's role can't create databases and roles")
	}

	suffix := random(tb)
	name := "sp_rls_" + suffix
	role, password := "sp_rls_app_"+suffix, random(tb)+random(tb)
	require.NoError(tb, admin.Exec("CREATE DATABASE "+name).Error)
	tb.Cleanup(func() {
		_ = admin.Exec("DROP DATABASE IF EXISTS " + name + " WITH (FORCE)").Error
		_ = admin.Exec("DROP ROLE IF EXISTS " + role).Error
		if sqlDB, err := admin.DB(); err == nil {
			_ = sqlDB.Close()
		}
	})

	u, err := url.Parse(base)
	require.NoError(tb, err)
	u.Path = "/" + name
	ownerURL := u.String()
	require.NoError(tb, database.RunMigrations(ownerURL))

	// The role is what production's API role would be: a plain login role
	// whose privileges come from servicepro_app (migration 000007).
	require.NoError(tb, admin.Exec("CREATE ROLE "+role+" LOGIN PASSWORD '"+password+"' IN ROLE servicepro_app").Error)
	u.User = url.UserPassword(role, password)
	appURL := u.String()

	owner, err := gorm.Open(postgres.Open(ownerURL), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	require.NoError(tb, err)
	tb.Cleanup(func() {
		if sqlDB, err := owner.DB(); err == nil {
			_ = sqlDB.Close()
		}
	})
	return &Env{Owner: owner, AppURL: appURL}
}

// OpenApp connects as the RLS-bound role the way cmd/main.go does with
// RLS_ENFORCE=true. withPlugin also installs the GORM tenancy plugin; leave
// it off to see what the database alone enforces.
func (e *Env) OpenApp(tb testing.TB, withPlugin bool) *gorm.DB {
	tb.Helper()
	db, _, err := database.NewGormDB(&config.DatabaseConfig{
		URL:             e.AppURL,
		RLSEnforce:      true,
		MaxOpenConns:    4,
		MaxIdleConns:    2,
		ConnMaxLifetime: time.Hour,
	})
	require.NoError(tb, err)
	db.Logger = logger.Default.LogMode(logger.Silent)
	tb.Cleanup(func() {
		if sqlDB, err := db.DB(); err == nil {
			_ = sqlDB.Close()
		}
	})
	if withPlugin {
		require.NoError(tb, tenancy.Register(db))
	}
	return db
}
