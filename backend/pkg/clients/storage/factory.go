package storage

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

// NewClient creates a new storage client based on configuration.
//
// Provider selection order:
// 1. If S3Compatible is configured (bucket set) -> S3 (works with any S3-compatible service)
// 2. Fallback -> Mock (for tests or when storage is not configured)
//
// Note: Providers must be registered by importing their packages with blank imports:
//
//	import _ "github.com/javaknight1/servicepro/backend/pkg/clients/storage/mock"
//	import _ "github.com/javaknight1/servicepro/backend/pkg/clients/storage/s3"
func NewClient(ctx context.Context, cfg *config.Config) (Client, error) {
	if cfg == nil {
		return nil, fmt.Errorf("config is required")
	}

	provider := detectProvider(cfg)
	// In production the in-memory mock loses every file on restart and
	// differs per instance (T256, D11). Refuse it; the caller runs without
	// object storage (PDFs are generated on demand, uploads are off) until
	// Cloudflare R2 is configured (T160).
	if provider == ProviderMock && cfg.Server.Env == "production" {
		return nil, fmt.Errorf("no object storage configured in production: set S3_COMPATIBLE_BUCKET and credentials (Cloudflare R2, T160)")
	}
	logging.Info(ctx, "[STORAGE] Using provider", map[string]any{"provider": provider})

	constructorsMu.RLock()
	constructor, ok := constructors[provider]
	constructorsMu.RUnlock()

	if !ok {
		registered := GetRegisteredProviders()
		return nil, fmt.Errorf("storage provider %q not registered; registered providers: %v. Import the provider package with a blank import, e.g.: import _ \"github.com/javaknight1/servicepro/backend/pkg/clients/storage/mock\"", provider, registered)
	}

	return constructor(ctx, cfg)
}

// detectProvider determines which provider to use based on configuration
func detectProvider(cfg *config.Config) Provider {
	// Use S3 if S3Compatible is configured (works with any S3-compatible service)
	if cfg.S3Compatible.Bucket != "" && cfg.S3Compatible.AccessKeyID != "" {
		return ProviderS3
	}

	// Fallback to mock (for tests or when storage is not configured)
	return ProviderMock
}
