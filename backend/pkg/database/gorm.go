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

	// Connect to database
	db, err := gorm.Open(postgres.Open(cfg.URL), gormConfig)
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
	if err := sqlDB.Ping(); err != nil {
		return nil, nil, fmt.Errorf("failed to ping database: %w", err)
	}

	logging.Info(context.Background(), "Successfully connected to PostgreSQL database via GORM", nil)
	return db, queryLogger, nil
}
