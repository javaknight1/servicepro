package analytics

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

// NewClient creates a new analytics client based on configuration.
//
// Provider selection order:
// 1. If PostHog API key is configured -> PostHog
// 2. Fallback -> Mock
func NewClient(ctx context.Context, cfg *config.Config) (Client, error) {
	if cfg == nil {
		return nil, fmt.Errorf("config is required")
	}

	provider := detectProvider(cfg)
	logging.Info(ctx, "[ANALYTICS] Using provider", map[string]any{"provider": string(provider)})

	constructorsMu.RLock()
	constructor, ok := constructors[provider]
	constructorsMu.RUnlock()

	if !ok {
		// Fall back to mock if requested provider isn't registered
		constructorsMu.RLock()
		constructor, ok = constructors[ProviderMock]
		constructorsMu.RUnlock()

		if !ok {
			return nil, fmt.Errorf("analytics provider %q not registered", provider)
		}
		logging.Info(ctx, "[ANALYTICS] Falling back to mock provider", nil)
	}

	return constructor(ctx, cfg)
}

// detectProvider determines which provider to use based on configuration
func detectProvider(cfg *config.Config) Provider {
	if cfg.Analytics.PostHogAPIKey != "" {
		return ProviderPostHog
	}
	return ProviderMock
}
