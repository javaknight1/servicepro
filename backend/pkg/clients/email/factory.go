package email

import (
	"context"
	"fmt"
	"sync"

	"github.com/javaknight1/servicepro/backend/config"
	"github.com/javaknight1/servicepro/backend/pkg/clients/logging"
)

// ConstructorFunc is a function that creates a Client
type ConstructorFunc func(ctx context.Context, cfg *config.Config) (Client, error)

var (
	constructorsMu sync.RWMutex
	constructors   = make(map[Provider]ConstructorFunc)
)

// RegisterProvider registers a constructor function for a provider.
// This is typically called from init() in provider implementation packages.
func RegisterProvider(p Provider, constructor ConstructorFunc) {
	constructorsMu.Lock()
	defer constructorsMu.Unlock()
	constructors[p] = constructor
}

// GetRegisteredProviders returns a list of registered providers
func GetRegisteredProviders() []Provider {
	constructorsMu.RLock()
	defer constructorsMu.RUnlock()

	providers := make([]Provider, 0, len(constructors))
	for p := range constructors {
		providers = append(providers, p)
	}
	return providers
}

// NewClient creates a new email client based on configuration.
//
// Provider selection order:
// 1. If cfg.SMTP.Host != "" -> SMTP (Mailpit for local development)
// 2. If cfg.Resend.APIKey != "" -> Resend
// 3. Fallback -> Mock (refused in production)
//
// Note: Providers must be registered by importing their packages with blank imports:
//
//	import _ "github.com/javaknight1/servicepro/backend/pkg/clients/email/mock"
//	import _ "github.com/javaknight1/servicepro/backend/pkg/clients/email/smtp"
//	import _ "github.com/javaknight1/servicepro/backend/pkg/clients/email/resend"
func NewClient(ctx context.Context, cfg *config.Config) (Client, error) {
	if cfg == nil {
		return nil, fmt.Errorf("config is required")
	}

	provider := detectProvider(cfg)
	logging.Info(ctx, "[EMAIL] Using provider", map[string]any{"provider": provider})

	// In production a missing email configuration must not silently fall
	// back to the mock: users would never get reset/verification/invite
	// emails (F-14). Fail startup instead.
	if provider == ProviderMock && cfg.Server.Env == "production" {
		return nil, fmt.Errorf("no email provider configured in production: set RESEND_API_KEY")
	}

	constructorsMu.RLock()
	constructor, ok := constructors[provider]
	constructorsMu.RUnlock()

	if !ok {
		registered := GetRegisteredProviders()
		return nil, fmt.Errorf("email provider %q not registered; registered providers: %v. Import the provider package with a blank import, e.g.: import _ \"github.com/javaknight1/servicepro/backend/pkg/clients/email/mock\"", provider, registered)
	}

	return constructor(ctx, cfg)
}

// detectProvider determines which provider to use based on configuration
func detectProvider(cfg *config.Config) Provider {
	// 1. Check for SMTP configuration (Mailpit for local development)
	if cfg.SMTP.Host != "" {
		return ProviderSMTP
	}

	// 2. Check for Resend credentials
	if cfg.Resend.APIKey != "" {
		return ProviderResend
	}

	// 3. Fallback to mock (unit tests, CI without Docker). Resend is the only
	// production provider (T159); SES was removed because its defaults made
	// it look configured and defeated the guard above.
	return ProviderMock
}
