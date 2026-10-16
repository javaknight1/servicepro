package metrics_test

import (
	"context"
	"testing"

	"github.com/javaknight1/servicepro/backend/config"
	"github.com/javaknight1/servicepro/backend/pkg/clients/metrics"
	_ "github.com/javaknight1/servicepro/backend/pkg/clients/metrics/mock"
)

// T126: with an AWS region configured (SES/SNS) and Prometheus off, the
// factory used to pick CloudWatch, which was never linked into the binary,
// and metrics client creation failed.
func TestNewClient_NoCloudWatchFallback(t *testing.T) {
	cfg := &config.Config{}
	cfg.Server.Env = "production"
	cfg.AWS.Region = "us-east-1"

	if _, err := metrics.NewClient(context.Background(), cfg); err != nil {
		t.Fatalf("NewClient: %v", err)
	}
}
