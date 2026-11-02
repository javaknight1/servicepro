package database

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/stdlib"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"github.com/javaknight1/servicepro/backend/config"
	"github.com/javaknight1/servicepro/backend/internal/tenancy"
	"github.com/javaknight1/servicepro/backend/pkg/clients/logging"
)

// NewGormDB creates a new GORM database connection with a custom query logger
func NewGormDB(cfg *config.DatabaseConfig) (*gorm.DB, *QueryLogger, error) {
	// Configure custom GORM query logger
	queryLogger := NewQueryLogger(&QueryLoggerConfig{
		SlowQueryThreshold: cfg.SlowQueryThreshold,
		AlertThreshold:     cfg.QueryAlertThreshold,
		LogAllQueries:      cfg.LogAllQueries,
	})

	gormConfig := &gorm.Config{
		Logger: queryLogger,
	}

	// Behind a transaction-mode pooler (PgBouncer on :6543) a
	// connection may change between statements, so server-side prepared
	// statements must not be used: switch pgx to the simple protocol.
	simpleProtocol := cfg.SimpleProtocol || UsesTransactionPooler(cfg.URL)

	pgCfg := postgres.Config{DSN: cfg.URL, PreferSimpleProtocol: simpleProtocol}
	if cfg.RLSEnforce {
		// Open the pool ourselves so every connection applies the caller's
		// tenant to its transactions (T154, tenancy.RLSConnector).
		connCfg, err := pgx.ParseConfig(cfg.URL)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to parse database URL: %w", err)
		}
		if simpleProtocol {
			connCfg.DefaultQueryExecMode = pgx.QueryExecModeSimpleProtocol
		}
		pgCfg = postgres.Config{Conn: sql.OpenDB(tenancy.RLSConnector(stdlib.GetConnector(*connCfg)))}
	}

	// Connect to database
	db, err := gorm.Open(postgres.New(pgCfg), gormConfig)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	// Get underlying SQL database
	sqlDB, err := db.DB()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get database instance: %w", err)
	}

	// Configure connection pool
	sqlDB.SetMaxOpenConns(cfg.MaxOpenConns)
	sqlDB.SetMaxIdleConns(cfg.MaxIdleConns)
	sqlDB.SetConnMaxLifetime(cfg.ConnMaxLifetime)

	// Verify connection
	if err := sqlDB.PingContext(context.Background()); err != nil {
		return nil, nil, fmt.Errorf("failed to ping database: %w", err)
	}

	logging.Info(context.Background(), "Successfully connected to PostgreSQL database via GORM", map[string]any{
		"max_open_conns":  cfg.MaxOpenConns,
		"simple_protocol": simpleProtocol,
		"rls_enforce":     cfg.RLSEnforce,
	})
	if cfg.RLSEnforce {
		warnIfRLSBypassed(sqlDB)
	}
	return db, queryLogger, nil
}

// warnIfRLSBypassed logs when RLS_ENFORCE is on but the connection's role
// is exempt from row-level security (superuser, BYPASSRLS, or owner of the
// tenant tables), so the backstop isn't actually in force.
func warnIfRLSBypassed(sqlDB *sql.DB) {
	ctx := context.Background()
	var role string
	var exempt bool
	err := sqlDB.QueryRowContext(ctx, `
		SELECT r.rolname, r.rolsuper OR r.rolbypassrls
		    OR pg_has_role(r.oid, (SELECT relowner FROM pg_class WHERE oid = 'public.customers'::regclass), 'MEMBER')
		FROM pg_roles r WHERE r.rolname = current_user`).Scan(&role, &exempt)
	if err != nil {
		logging.Warn(ctx, "Could not check whether the database role is subject to RLS", map[string]any{"error": err.Error()})
		return
	}
	if exempt {
		logging.Warn(ctx, "RLS_ENFORCE is on but the database role bypasses row-level security; connect as a member of servicepro_app", map[string]any{"role": role})
	}
}
