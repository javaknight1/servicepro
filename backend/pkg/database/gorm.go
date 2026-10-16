package database

import (
	"context"
	"fmt"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"github.com/javaknight1/servicepro/backend/config"
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

	// Behind a transaction-mode pooler (Supabase/Supavisor on :6543) a
	// connection may change between statements, so server-side prepared
	// statements must not be used: switch pgx to the simple protocol.
	simpleProtocol := cfg.SimpleProtocol || UsesTransactionPooler(cfg.URL)

	// Connect to database
	db, err := gorm.Open(postgres.New(postgres.Config{
		DSN:                  cfg.URL,
		PreferSimpleProtocol: simpleProtocol,
	}), gormConfig)
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
	})
	return db, queryLogger, nil
}
