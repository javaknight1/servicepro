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
