package logging

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

// NewClient creates a new logging client based on configuration.
//
// Provider selection: development -> Mock (readable text on stdout);
// everything else -> Stdout (structured JSON lines, collected by the platform,
// e.g. Cloud Run -> Cloud Logging). Remote log shippers (Better Stack, Loki,
// CloudWatch) were removed in T126; Better Stack is used for uptime only.
//
// Providers must be registered with a blank import:
//
//	import _ "github.com/javaknight1/servicepro/backend/pkg/clients/logging/mock"
func NewClient(ctx context.Context, cfg *config.Config) (Client, error) {
	if cfg == nil {
		return nil, fmt.Errorf("config is required")
	}

	provider := detectProvider(cfg)
	log.Printf("Logging client: using provider %q", provider)

	constructorsMu.RLock()
	constructor, ok := constructors[provider]
	constructorsMu.RUnlock()

	if !ok {
		registered := GetRegisteredProviders()
		return nil, fmt.Errorf("logging provider %q not registered; registered providers: %v. Import the provider package with a blank import", provider, registered)
	}

	return constructor(ctx, cfg)
}

// detectProvider determines which provider to use based on configuration
func detectProvider(cfg *config.Config) Provider {
	if cfg.Server.Env == "development" {
		return ProviderMock
	}
	return ProviderStdout
}

// MustNewClient creates a new logging client or panics on error.
// This is useful for application initialization.
func MustNewClient(ctx context.Context, cfg *config.Config) Client {
	client, err := NewClient(ctx, cfg)
	if err != nil {
		panic(fmt.Sprintf("failed to create logging client: %v", err))
	}
	return client
}
