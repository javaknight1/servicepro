package metrics_test

import (
	"context"
	"testing"

	"github.com/javaknight1/servicepro/backend/config"
	"github.com/javaknight1/servicepro/backend/pkg/clients/metrics"
	_ "github.com/javaknight1/servicepro/backend/pkg/clients/metrics/mock"
)

// T126: in production with Prometheus off, the factory used to pick
// CloudWatch (never linked into the binary) and metrics creation failed.
// AWS config is gone since T159; production must still get a client.
func TestNewClient_NoCloudWatchFallback(t *testing.T) {
	cfg := &config.Config{}
	cfg.Server.Env = "production"

	if _, err := metrics.NewClient(context.Background(), cfg); err != nil {
		t.Fatalf("NewClient: %v", err)
	}
}
