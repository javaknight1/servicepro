package database

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"

	"github.com/javaknight1/servicepro/backend/config"
	"github.com/javaknight1/servicepro/backend/pkg/clients/logging"
)

// Redis modes (reported by the health "redis" check).
const (
	RedisModeConnected        = "connected"          // shared Redis reachable at boot
	RedisModeInMemoryFallback = "in_memory_fallback" // REDIS_URL set but unreachable at boot
	RedisModeNotConfigured    = "not_configured"     // REDIS_URL unset
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
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := client.Ping(ctx).Err(); err != nil {
		_ = client.Close()
		return nil, fmt.Errorf("failed to ping Redis: %w", err)
	}

	logging.Info(ctx, "Successfully connected to Redis", nil)
	return client, nil
}

// RedisConnection is the Redis client the app uses plus how it was obtained.
type RedisConnection struct {
	Client *redis.Client
	// Mode is one of the RedisMode* constants.
	Mode string

	closeFn func()
}

// Close releases the client (and the in-memory server, if any).
func (r *RedisConnection) Close() {
	if r.closeFn != nil {
		r.closeFn()
	}
}

// ConnectRedisOrFallback connects to the configured Redis. When REDIS_URL is
// unset or Redis is unreachable at boot it logs a warning and starts an
// in-process, in-memory Redis-compatible server instead, so every caller
// (rate limiters, permission cache, password-reset / email-verification
// tokens, import queue) keeps working unchanged.
//
// The fallback is per process: state is not shared across instances and is
// lost on restart (pending password-reset / verification links stop working
// after a scale-to-zero). Use a shared Redis when running several instances.
func ConnectRedisOrFallback(cfg *config.RedisConfig) (*RedisConnection, error) {
	ctx := context.Background()
	mode := RedisModeNotConfigured
	if cfg.URL != "" {
		client, err := NewRedisClient(cfg)
		if err == nil {
			return &RedisConnection{Client: client, Mode: RedisModeConnected, closeFn: func() { _ = client.Close() }}, nil
		}
		mode = RedisModeInMemoryFallback
		logging.Warn(ctx, "[REDIS] Redis unreachable at boot; falling back to in-memory store (per-instance, not persisted)", map[string]any{"error": err.Error()})
	} else {
		logging.Warn(ctx, "[REDIS] REDIS_URL not configured; using in-memory store (per-instance, not persisted)", nil)
	}

	conn, err := newInMemoryRedis()
	if err != nil {
		return nil, err
	}
	conn.Mode = mode
	return conn, nil
}

// newInMemoryRedis starts an embedded miniredis server on 127.0.0.1 and a
// client for it. miniredis only expires keys when its clock is advanced, so a
// ticker advances it in real time (1s granularity); without it TTL-based rate
// limits and token expiry would never lapse.
func newInMemoryRedis() (*RedisConnection, error) {
	mr, err := miniredis.Run()
	if err != nil {
		return nil, fmt.Errorf("start in-memory redis: %w", err)
	}
	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})

	stop := make(chan struct{})
	var once sync.Once
	go func() {
		const tick = time.Second
		ticker := time.NewTicker(tick)
		defer ticker.Stop()
		last := time.Now()
		for {
			select {
			case <-stop:
				return
			case now := <-ticker.C:
				mr.FastForward(now.Sub(last))
				last = now
			}
		}
	}()

	return &RedisConnection{
		Client: client,
		closeFn: func() {
			once.Do(func() {
				close(stop)
				_ = client.Close()
				mr.Close()
			})
		},
	}, nil
}
