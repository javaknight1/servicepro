package storage

import (
	"context"
	"testing"

	"github.com/javaknight1/servicepro/backend/config"
)

// T256: production never falls back to the in-memory mock, which loses every
// file on restart and differs per instance.
func TestNewClient_ProductionRefusesMock(t *testing.T) {
	cfg := &config.Config{}
	cfg.Server.Env = "production"
	if _, err := NewClient(context.Background(), cfg); err == nil {
		t.Fatal("expected an error when no object storage is configured in production")
	}
}

func TestDetectProvider(t *testing.T) {
	cfg := &config.Config{}
	if p := detectProvider(cfg); p != ProviderMock {
		t.Fatalf("no storage configured: got %q, want mock", p)
	}
	cfg.S3Compatible.Bucket = "uploads"
	cfg.S3Compatible.AccessKeyID = "key"
	if p := detectProvider(cfg); p != ProviderS3 {
		t.Fatalf("bucket + key set: got %q", p)
	}
}
