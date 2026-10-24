package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/golang-migrate/migrate/v4"
	// pgx v5 database driver for golang-migrate (registers the pgx5:// scheme)
	_ "github.com/golang-migrate/migrate/v4/database/pgx/v5"
	"github.com/golang-migrate/migrate/v4/source/iofs"

	// pgx database/sql driver, used for the baseline probe
	_ "github.com/jackc/pgx/v5/stdlib"

	"github.com/javaknight1/servicepro/backend/migrations"
	"github.com/javaknight1/servicepro/backend/pkg/clients/logging"
)

const migrateLogPrefix = "[MIGRATE]"

// BaselineVersion is the first migration. 000001_initial_schema holds the
// whole pre-launch schema (the old 000001–000014 were squashed into it on
// 2026-10-24, before any shared database existed).
const BaselineVersion = 1

// baselineSentinelTable is a table created by the baseline migration. If it
// exists while schema_migrations does not, the database was built outside
// golang-migrate (the retired hand-applied 001_schema.sql) and is refused.
const baselineSentinelTable = "public.tenants"

// Schema states reported by SchemaStatus (and surfaced on /health/ready).
const (
	SchemaOK      = "ok"      // applied == expected
	SchemaAhead   = "ahead"   // applied > expected: newer additive migrations already applied (expand/contract) — fine
	SchemaBehind  = "behind"  // applied < expected: this build shipped ahead of its schema — refuse traffic
	SchemaDirty   = "dirty"   // a migration failed partway and needs manual repair (`migrate force N`)
	SchemaUnknown = "unknown" // could not determine (DB unreachable, etc.)
)

// UsesTransactionPooler reports whether a database URL points at a
// transaction-mode connection pooler (Supabase/Supavisor or PgBouncer on port
// 6543), where server-side prepared statements must not be used.
func UsesTransactionPooler(databaseURL string) bool {
	u, err := url.Parse(databaseURL)
	if err != nil || u.Host == "" {
		// key=value DSN or unparseable: fall back to a substring check
		return strings.Contains(databaseURL, ":6543") || strings.Contains(databaseURL, "port=6543")
	}
	return u.Port() == "6543"
}

// appendQueryParam adds key=value to a URL unless the key is already present.
func appendQueryParam(rawURL, key, value string) string {
	if strings.Contains(rawURL, key+"=") {
		return rawURL
	}
	sep := "?"
	if strings.Contains(rawURL, "?") {
		sep = "&"
	}
	return rawURL + sep + key + "=" + value
}

// migrateURL rewrites a postgres:// URL for golang-migrate's pgx v5 driver.
// Migrations always run with the simple protocol: it is safe against direct
// Postgres and every pooler mode, and migration throughput is irrelevant.
func migrateURL(databaseURL string) string {
	u := databaseURL
	u = strings.Replace(u, "postgresql://", "pgx5://", 1)
	u = strings.Replace(u, "postgres://", "pgx5://", 1)
	return appendQueryParam(u, "default_query_exec_mode", "simple_protocol")
}

// newMigrator builds a golang-migrate instance over the embedded migrations.
// Callers own the returned migrator and must Close it.
func newMigrator(databaseURL string) (*migrate.Migrate, error) {
	if databaseURL == "" {
		return nil, errors.New("database URL is empty")
	}
	source, err := iofs.New(migrations.FS, ".")
	if err != nil {
		return nil, fmt.Errorf("create migration source: %w", err)
	}
	m, err := migrate.NewWithSourceInstance("iofs", source, migrateURL(databaseURL))
	if err != nil {
		return nil, fmt.Errorf("create migrator: %w", err)
	}
	return m, nil
}

// refuseUntrackedSchema fails when the baseline's tables exist but
// schema_migrations has no version: a database built from the retired
// 001_schema.sql. It used to be adopted at BaselineVersion, but since the
// squash that version also contains the later changes, so adopting it would
// silently skip them. Such databases are local-only; rebuild them.
func refuseUntrackedSchema(databaseURL string, m *migrate.Migrate) error {
	if _, _, err := m.Version(); !errors.Is(err, migrate.ErrNilVersion) {
		return nil // already tracked (or a real error surfaced later by Up)
	}

	db, err := sql.Open("pgx", appendQueryParam(databaseURL, "default_query_exec_mode", "simple_protocol"))
	if err != nil {
		return fmt.Errorf("open database for baseline probe: %w", err)
	}
	defer func() { _ = db.Close() }()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var exists bool
	if err := db.QueryRowContext(ctx, "SELECT to_regclass($1) IS NOT NULL", baselineSentinelTable).Scan(&exists); err != nil {
		return fmt.Errorf("baseline probe: %w", err)
	}
	if exists {
		return errors.New("schema exists without migration history (built from the retired 001_schema.sql); rebuild it with `make db-fresh-seed`")
	}
	return nil
}

// RunMigrations applies all pending embedded migrations. A database created
// from the legacy 001_schema.sql is refused (refuseUntrackedSchema).
func RunMigrations(databaseURL string) error {
	ctx := context.Background()
	m, err := newMigrator(databaseURL)
	if err != nil {
		return err
	}
	defer func() { _, _ = m.Close() }()

	if err := refuseUntrackedSchema(databaseURL, m); err != nil {
		return err
	}

	err = m.Up()
	if err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("run migrations: %w", err)
	}

	version, dirty, _ := m.Version()
	if errors.Is(err, migrate.ErrNoChange) {
		logging.Info(ctx, fmt.Sprintf("%s No pending migrations", migrateLogPrefix), map[string]any{"version": version})
	} else {
		logging.Info(ctx, fmt.Sprintf("%s Migrations applied", migrateLogPrefix), map[string]any{"version": version, "dirty": dirty})
	}
	return nil
}

// MigrateDown rolls every migration back. Local-only escape hatch: production
// rollback is a forward-fix migration, never a down migration.
func MigrateDown(databaseURL string) error {
	m, err := newMigrator(databaseURL)
	if err != nil {
		return err
	}
	defer func() { _, _ = m.Close() }()

	if err := m.Down(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("roll back migrations: %w", err)
	}
	return nil
}

// MigrateDownSteps rolls back the last n applied migrations (n > 0).
func MigrateDownSteps(databaseURL string, n int) error {
	if n <= 0 {
		return fmt.Errorf("roll back migrations: step count must be positive, got %d", n)
	}
	m, err := newMigrator(databaseURL)
	if err != nil {
		return err
	}
	defer func() { _, _ = m.Close() }()

	if err := m.Steps(-n); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("roll back %d migration(s): %w", n, err)
	}
	return nil
}

// ForceMigrationVersion sets the recorded version (clearing the dirty flag)
// without running any SQL. Used to repair a dirty schema after fixing it by
// hand. version -1 means "no migrations applied".
func ForceMigrationVersion(databaseURL string, version int) error {
	m, err := newMigrator(databaseURL)
	if err != nil {
		return err
	}
	defer func() { _, _ = m.Close() }()

	if err := m.Force(version); err != nil {
		return fmt.Errorf("force version %d: %w", version, err)
	}
	return nil
}

// MigrationVersion reports the applied schema version and dirty flag.
// Returns version 0 with no error when nothing has been applied yet.
func MigrationVersion(databaseURL string) (version uint, dirty bool, err error) {
	m, mErr := newMigrator(databaseURL)
	if mErr != nil {
		return 0, false, mErr
	}
	defer func() { _, _ = m.Close() }()

	version, dirty, err = m.Version()
	if errors.Is(err, migrate.ErrNilVersion) {
		return 0, false, nil
	}
	if err != nil {
		return 0, false, fmt.Errorf("read migration version: %w", err)
	}
	return version, dirty, nil
}

// LatestMigrationVersion returns the highest migration version embedded in
// this binary, i.e. the schema version this build expects.
func LatestMigrationVersion() (uint, error) {
	entries, err := migrations.FS.ReadDir(".")
	if err != nil {
		return 0, fmt.Errorf("read embedded migrations: %w", err)
	}
	var latest uint
	for _, e := range entries {
		name := e.Name()
		if !strings.HasSuffix(name, ".up.sql") {
			continue
		}
		idx := strings.Index(name, "_")
		if idx <= 0 {
			continue
		}
		n, err := strconv.ParseUint(name[:idx], 10, 64)
		if err != nil {
			continue
		}
		if uint(n) > latest {
			latest = uint(n)
		}
	}
	return latest, nil
}

// SchemaState is a point-in-time comparison of applied vs expected schema.
type SchemaState struct {
	Status   string `json:"status"`
	Applied  uint   `json:"applied"`
	Expected uint   `json:"expected"`
	Error    string `json:"error,omitempty"`
}

// Healthy reports whether the instance may serve traffic with this schema.
func (s SchemaState) Healthy() bool {
	return s.Status == SchemaOK || s.Status == SchemaAhead
}

// SchemaStatus compares the database's applied version with the latest
// version embedded in this binary.
func SchemaStatus(databaseURL string) SchemaState {
	expected, err := LatestMigrationVersion()
	if err != nil {
		return SchemaState{Status: SchemaUnknown, Error: err.Error()}
	}
	applied, dirty, err := MigrationVersion(databaseURL)
	if err != nil {
		return SchemaState{Status: SchemaUnknown, Expected: expected, Error: err.Error()}
	}
	state := SchemaState{Applied: applied, Expected: expected}
	switch {
	case dirty:
		state.Status = SchemaDirty
	case applied < expected:
		state.Status = SchemaBehind
	case applied > expected:
		state.Status = SchemaAhead
	default:
		state.Status = SchemaOK
	}
	return state
}

// SchemaMonitor caches SchemaStatus for the readiness probe. golang-migrate
// opens its own connection, which is too expensive for every probe, so a
// healthy result is cached for the process lifetime (the schema only moves
// forward, and "ahead" is fine). Unhealthy results are re-checked at most
// once per recheckInterval so an instance recovers once `migrate up` runs.
type SchemaMonitor struct {
	databaseURL     string
	recheckInterval time.Duration
	check           func(string) SchemaState

	mu        sync.Mutex
	state     SchemaState
	checkedAt time.Time
}

// NewSchemaMonitor creates a monitor and performs the initial check.
func NewSchemaMonitor(databaseURL string) *SchemaMonitor {
	sm := &SchemaMonitor{
		databaseURL:     databaseURL,
		recheckInterval: 30 * time.Second,
		check:           SchemaStatus,
	}
	sm.refresh()
	return sm
}

func (sm *SchemaMonitor) refresh() {
	sm.state = sm.check(sm.databaseURL)
	sm.checkedAt = time.Now()
	if !sm.state.Healthy() {
		logging.Error(context.Background(), fmt.Sprintf("%s Schema is not ready for this build", migrateLogPrefix), map[string]any{
			"status": sm.state.Status, "applied": sm.state.Applied, "expected": sm.state.Expected, "error": sm.state.Error,
		})
	}
}

// State returns the cached schema state, re-checking if it was unhealthy and
// the recheck interval has elapsed.
func (sm *SchemaMonitor) State() SchemaState {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	if !sm.state.Healthy() && time.Since(sm.checkedAt) >= sm.recheckInterval {
		sm.refresh()
	}
	return sm.state
}
