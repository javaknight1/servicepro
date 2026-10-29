package sms

import (
	"context"
	"testing"

	"github.com/javaknight1/servicepro/backend/config"
)

// T256: production never falls back to the mock, which records texts as
// sent that nobody receives.
func TestNewClient_ProductionRefusesMock(t *testing.T) {
	cfg := &config.Config{}
	cfg.Server.Env = "production"
	if _, err := NewClient(context.Background(), cfg); err == nil {
		t.Fatal("expected an error when no SMS provider is configured in production")
	}
}

func TestDetectProvider(t *testing.T) {
	cfg := &config.Config{}
	cfg.Server.Env = "development"
	if p := detectProvider(cfg); p != ProviderMock {
		t.Fatalf("no provider configured: got %q, want mock", p)
	}
	cfg.SMS.TextBelt.APIKey = "tb_x"
	if p := detectProvider(cfg); p != ProviderTextBelt {
		t.Fatalf("TextBelt key set: got %q", p)
	}
}
