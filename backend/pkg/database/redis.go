package database

import (
	"context"
	"fmt"

	"github.com/redis/go-redis/v9"

	"github.com/javaknight1/servicepro/backend/config"
	"github.com/javaknight1/servicepro/backend/pkg/clients/logging"
)

// NewRedisClient creates a new Redis client
func NewRedisClient(cfg *config.RedisConfig) (*redis.Client, error) {
	opt, err := redis.ParseURL(cfg.URL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse Redis URL: %w", err)
	}

	// Override with config values if provided
	if cfg.Password != "" {
		opt.Password = cfg.Password
	}
	opt.DB = cfg.DB

	client := redis.NewClient(opt)

	// Verify connection
	ctx := context.Background()
	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to ping Redis: %w", err)
	}

	logging.Info(ctx, "Successfully connected to Redis", nil)
	return client, nil
}
