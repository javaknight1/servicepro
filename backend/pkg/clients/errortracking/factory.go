package errortracking

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

// NewClient creates a new error tracking client based on configuration.
//
// Provider selection order:
// 1. If cfg.Server.Env == "development" -> Mock
// 2. If cfg.Sentry.DSN != "" -> Sentry
// 3. Fallback -> Mock
//
// Note: Providers must be registered by importing their packages with blank imports:
//
//	import _ "github.com/javaknight1/servicepro/backend/pkg/clients/errortracking/mock"
//	import _ "github.com/javaknight1/servicepro/backend/pkg/clients/errortracking/sentry"
func NewClient(ctx context.Context, cfg *config.Config) (Client, error) {
	if cfg == nil {
		return nil, fmt.Errorf("config is required")
	}

	provider := detectProvider(cfg)
	log.Printf("Error tracking client: using provider %q", provider)

	constructorsMu.RLock()
	constructor, ok := constructors[provider]
	constructorsMu.RUnlock()

	if !ok {
		registered := GetRegisteredProviders()
		return nil, fmt.Errorf("error tracking provider %q not registered; registered providers: %v. Import the provider package with a blank import", provider, registered)
	}

	return constructor(ctx, cfg)
}

// detectProvider determines which provider to use based on configuration
func detectProvider(cfg *config.Config) Provider {
	// 1. Development environment defaults to mock
	if cfg.Server.Env == "development" {
		return ProviderMock
	}

	// 2. Check for Sentry DSN
	if cfg.Sentry.DSN != "" {
		return ProviderSentry
	}

	// 3. Fallback to mock
	return ProviderMock
}

// MustNewClient creates a new error tracking client or panics on error.
// This is useful for application initialization.
func MustNewClient(ctx context.Context, cfg *config.Config) Client {
	client, err := NewClient(ctx, cfg)
	if err != nil {
		panic(fmt.Sprintf("failed to create error tracking client: %v", err))
	}
	return client
}
