package database

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"

	"github.com/javaknight1/servicepro/backend/config"
)

func TestConnectRedisOrFallback_NotConfigured(t *testing.T) {
	conn, err := ConnectRedisOrFallback(&config.RedisConfig{URL: ""})
	if err != nil {
		t.Fatalf("ConnectRedisOrFallback: %v", err)
	}
	defer conn.Close()

	if conn.Mode != RedisModeNotConfigured {
		t.Fatalf("mode = %q, want %q", conn.Mode, RedisModeNotConfigured)
	}
	ctx := context.Background()
	if err := conn.Client.Set(ctx, "k", "v", 0).Err(); err != nil {
		t.Fatalf("set: %v", err)
	}
	if got, _ := conn.Client.Get(ctx, "k").Result(); got != "v" {
		t.Fatalf("get = %q, want v", got)
	}
}

func TestConnectRedisOrFallback_UnreachableFallsBack(t *testing.T) {
	conn, err := ConnectRedisOrFallback(&config.RedisConfig{URL: "redis://127.0.0.1:1"})
	if err != nil {
		t.Fatalf("ConnectRedisOrFallback: %v", err)
	}
	defer conn.Close()
	if conn.Mode != RedisModeInMemoryFallback {
		t.Fatalf("mode = %q, want %q", conn.Mode, RedisModeInMemoryFallback)
	}
	if err := conn.Client.Ping(context.Background()).Err(); err != nil {
		t.Fatalf("fallback ping: %v", err)
	}
}

func TestConnectRedisOrFallback_Connected(t *testing.T) {
	mr := miniredis.RunT(t)
	conn, err := ConnectRedisOrFallback(&config.RedisConfig{URL: "redis://" + mr.Addr()})
	if err != nil {
		t.Fatalf("ConnectRedisOrFallback: %v", err)
	}
	defer conn.Close()
	if conn.Mode != RedisModeConnected {
		t.Fatalf("mode = %q, want %q", conn.Mode, RedisModeConnected)
	}
}

// The in-memory fallback must expire keys in real time, otherwise rate-limit
// windows and token TTLs would never lapse.
func TestInMemoryRedis_ExpiresKeys(t *testing.T) {
	if testing.Short() {
		t.Skip("waits for a TTL to lapse")
	}
	conn, err := newInMemoryRedis()
	if err != nil {
		t.Fatalf("newInMemoryRedis: %v", err)
	}
	defer conn.Close()

	ctx := context.Background()
	if err := conn.Client.Set(ctx, "ratelimit:ip:1.2.3.4", 1, time.Second).Err(); err != nil {
		t.Fatalf("set: %v", err)
	}
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if n, _ := conn.Client.Exists(ctx, "ratelimit:ip:1.2.3.4").Result(); n == 0 {
			return
		}
		time.Sleep(200 * time.Millisecond)
	}
	t.Fatal("key with 1s TTL still present after 5s")
}
