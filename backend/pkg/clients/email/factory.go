package email

import (
	"context"
	"fmt"
	"log"
	"sync"

	"github.com/javaknight1/servicepro/backend/config"
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
// 1. If cfg.Server.Env == "development" -> Mock
// 2. If cfg.Resend.APIKey != "" -> Resend
// 3. If cfg.AWS.Region != "" && cfg.SES.FromEmail != "" -> SES
// 4. Fallback -> Mock
//
// Note: Providers must be registered by importing their packages with blank imports:
//
//	import _ "github.com/javaknight1/servicepro/backend/pkg/clients/email/mock"
//	import _ "github.com/javaknight1/servicepro/backend/pkg/clients/email/ses"
//	import _ "github.com/javaknight1/servicepro/backend/pkg/clients/email/resend"
func NewClient(ctx context.Context, cfg *config.Config) (Client, error) {
	if cfg == nil {
		return nil, fmt.Errorf("config is required")
	}

	provider := detectProvider(cfg)
	log.Printf("Email client: using provider %q", provider)

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
	// 1. Development environment defaults to mock
	if cfg.Server.Env == "development" {
		return ProviderMock
	}

	// 2. Check for Resend credentials
	if cfg.Resend.APIKey != "" {
		return ProviderResend
	}

	// 3. Check for SES credentials
	if cfg.AWS.Region != "" && cfg.SES.FromEmail != "" {
		return ProviderSES
	}

	// 4. Fallback to mock
	return ProviderMock
}

// MustNewClient creates a new email client or panics on error.
// This is useful for application initialization.
func MustNewClient(ctx context.Context, cfg *config.Config) Client {
	client, err := NewClient(ctx, cfg)
	if err != nil {
		panic(fmt.Sprintf("failed to create email client: %v", err))
	}
	return client
}
