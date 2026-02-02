package sms

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

// NewClient creates a new SMS client based on configuration.
//
// Provider selection order:
// 1. If cfg.SMS.TextBelt.APIKey != "" -> TextBelt (free tier for testing)
// 2. If cfg.AWS.SNS.Enabled && cfg.AWS.Region != "" -> AWS SNS
// 3. Fallback -> Mock
//
// Note: Providers must be registered by importing their packages with blank imports:
//
//	import _ "github.com/javaknight1/servicepro/backend/pkg/clients/sms/mock"
//	import _ "github.com/javaknight1/servicepro/backend/pkg/clients/sms/textbelt"
//	import _ "github.com/javaknight1/servicepro/backend/pkg/clients/sms/sns"
func NewClient(ctx context.Context, cfg *config.Config) (Client, error) {
	if cfg == nil {
		return nil, fmt.Errorf("config is required")
	}

	provider := detectProvider(cfg)
	logging.Info(ctx, "[SMS] Using provider", map[string]any{"provider": provider})

	constructorsMu.RLock()
	constructor, ok := constructors[provider]
	constructorsMu.RUnlock()

	if !ok {
		registered := GetRegisteredProviders()
		return nil, fmt.Errorf("SMS provider %q not registered; registered providers: %v. Import the provider package with a blank import, e.g.: import _ \"github.com/javaknight1/servicepro/backend/pkg/clients/sms/mock\"", provider, registered)
	}

	return constructor(ctx, cfg)
}

// detectProvider determines which provider to use based on configuration
func detectProvider(cfg *config.Config) Provider {
	// 1. Check for TextBelt configuration (free tier for production testing)
	if cfg.SMS.TextBelt.APIKey != "" {
		return ProviderTextBelt
	}

	// 2. Check for AWS SNS configuration
	if cfg.AWS.SNS.Enabled && cfg.AWS.Region != "" {
		return ProviderSNS
	}

	// 3. Fallback to mock (local development, unit tests)
	return ProviderMock
}

// MustNewClient creates a new SMS client or panics on error.
// This is useful for application initialization.
func MustNewClient(ctx context.Context, cfg *config.Config) Client {
	client, err := NewClient(ctx, cfg)
	if err != nil {
		panic(fmt.Sprintf("failed to create SMS client: %v", err))
	}
	return client
}
