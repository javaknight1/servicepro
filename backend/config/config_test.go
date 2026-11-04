package config

import (
	"strings"
	"testing"
)

// validConfig returns a configuration that passes configErrors for env.
func validConfig(env string) *Config {
	return &Config{
		Server: ServerConfig{Env: env},
		JWT:    JWTConfig{Secret: strings.Repeat("s", 32)},
		Auth:   AuthConfig{RegistrationMode: "invite_only"},
	}
}

func hasError(errs []string, substr string) bool {
	for _, e := range errs {
		if strings.Contains(e, substr) {
			return true
		}
	}
	return false
}

func TestConfigErrors_ValidDevelopmentConfigPasses(t *testing.T) {
	if errs := configErrors(validConfig("development"), ""); len(errs) != 0 {
		t.Fatalf("expected no errors, got %v", errs)
	}
}

// T212: an unset ENV on Cloud Run would silently run production with every
// development default (open registration, Swagger, insecure cookies).
func TestConfigErrors_DevelopmentEnvOnCloudRunFails(t *testing.T) {
	errs := configErrors(validConfig("development"), "servicepro-api")
	if !hasError(errs, "ENV is unset") {
		t.Fatalf("expected the Cloud Run ENV error, got %v", errs)
	}
}

// validProductionConfig mirrors what Cloud Run servicepro-api runs with.
func validProductionConfig() *Config {
	cfg := validConfig("production")
	cfg.Server.EdgeProxySecret = "edge"
	cfg.Server.InternalJobsSecret = "jobs"
	cfg.Cookie.Secure = true
	cfg.Database.URL = "postgresql://app@db.example/servicepro?sslmode=require"
	return cfg
}

func TestConfigErrors_ValidProductionConfigPasses(t *testing.T) {
	if errs := configErrors(validProductionConfig(), "servicepro-api"); len(errs) != 0 {
		t.Fatalf("expected no errors, got %v", errs)
	}
}

// T212: each of these fails open in production when missing, so startup
// must refuse instead.
func TestConfigErrors_ProductionRequiresSafeSettings(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*Config)
		want   string
	}{
		{"edge secret", func(c *Config) { c.Server.EdgeProxySecret = "" }, "EDGE_PROXY_SECRET"},
		{"secure cookies", func(c *Config) { c.Cookie.Secure = false }, "COOKIE_SECURE"},
		{"database url", func(c *Config) { c.Database.URL = DefaultDatabaseURL }, "DATABASE_URL"},
		{"startup migrations", func(c *Config) { c.Database.RunMigrationsOnStartup = true }, "RUN_MIGRATIONS_ON_STARTUP"},
		{"jobs token", func(c *Config) { c.Server.InternalJobsSecret = "" }, "INTERNAL_JOBS_SECRET"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := validProductionConfig()
			tt.mutate(cfg)
			if errs := configErrors(cfg, ""); !hasError(errs, tt.want) {
				t.Fatalf("expected an error naming %s, got %v", tt.want, errs)
			}
		})
	}
}

// Staging/preview environments keep their looser defaults.
func TestConfigErrors_ProductionRulesOnlyApplyToProduction(t *testing.T) {
	if errs := configErrors(validConfig("staging"), ""); len(errs) != 0 {
		t.Fatalf("expected no errors for staging, got %v", errs)
	}
}
