package email

import (
	"context"
	"testing"

	"github.com/javaknight1/servicepro/backend/config"
)

func TestNewClient_ProductionRefusesMock(t *testing.T) {
	cfg := &config.Config{}
	cfg.Server.Env = "production"
	if _, err := NewClient(context.Background(), cfg); err == nil {
		t.Fatal("expected an error when no email provider is configured in production")
	}
}

func TestDetectProvider_FallsBackToMockOutsideProduction(t *testing.T) {
	cfg := &config.Config{}
	cfg.Server.Env = "development"
	if p := detectProvider(cfg); p != ProviderMock {
		t.Fatalf("got %q", p)
	}
	cfg.Resend.APIKey = "re_x"
	if p := detectProvider(cfg); p != ProviderResend {
		t.Fatalf("got %q", p)
	}
}
