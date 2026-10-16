package stripe

import (
	"context"
	"errors"
	"strings"

	"github.com/shopspring/decimal"
	"github.com/stripe/stripe-go/v76"

	"github.com/javaknight1/servicepro/backend/config"
	"github.com/javaknight1/servicepro/backend/pkg/clients/logging"
)

// Default configuration values
const (
	defaultMaxNetworkRetries = 3
)

// Client is the main Stripe client wrapper
type Client struct {
	secretKey      string
	publishableKey string
	webhookSecret  string
	environment    string
}

// NewClient creates a new Stripe client
func NewClient(cfg *config.Config) (*Client, error) {
	if cfg == nil {
		return nil, errors.New("config is required")
	}

	if cfg.Stripe.SecretKey == "" {
		return nil, errors.New("stripe secret key is required")
	}

	if cfg.Stripe.PublishableKey == "" {
		return nil, errors.New("stripe publishable key is required")
	}

	// Set Stripe API key
	stripe.Key = cfg.Stripe.SecretKey

	// Configure Stripe SDK
	stripe.SetAppInfo(&stripe.AppInfo{
		Name:    "ServicePro",
		Version: "1.0.0",
		URL:     "https://servicepro.com",
	})

	// Set max network retries
	maxRetries := cfg.Stripe.MaxRetries
	if maxRetries <= 0 {
		maxRetries = defaultMaxNetworkRetries
	}
	stripe.SetBackend("api", stripe.GetBackendWithConfig(
		stripe.APIBackend,
		&stripe.BackendConfig{
			MaxNetworkRetries: stripe.Int64(int64(maxRetries)),
		},
	))

	// Determine environment from secret key prefix
	environment := "unknown"
	if len(cfg.Stripe.SecretKey) > 8 {
		if cfg.Stripe.SecretKey[:8] == "sk_test_" {
			environment = "test"
		} else if cfg.Stripe.SecretKey[:8] == "sk_live_" {
			environment = "live"
		}
	}

	client := &Client{
		secretKey:      cfg.Stripe.SecretKey,
		publishableKey: cfg.Stripe.PublishableKey,
		webhookSecret:  cfg.Stripe.WebhookSecret,
		environment:    environment,
	}

	logging.Info(context.Background(), "Stripe client initialized", map[string]any{"mode": environment})

	return client, nil
}

// ConvertDecimalToCents converts a USD-style (2-decimal) amount to cents,
// rounding half away from zero. Prefer ToMinorUnits with the real currency.
func ConvertDecimalToCents(amount decimal.Decimal) int64 {
	return ToMinorUnits(amount, "usd")
}

// Stripe's zero- and three-decimal currencies
// (https://docs.stripe.com/currencies#minor-units). Everything else uses 2.
var (
	zeroDecimalCurrencies = map[string]bool{
		"bif": true, "clp": true, "djf": true, "gnf": true, "jpy": true, "kmf": true, "krw": true, "mga": true,
		"pyg": true, "rwf": true, "ugx": true, "vnd": true, "vuv": true, "xaf": true, "xof": true, "xpf": true,
	}
	threeDecimalCurrencies = map[string]bool{"bhd": true, "jod": true, "kwd": true, "omr": true, "tnd": true}
)

// MinorUnitExponent returns how many decimal places a currency's smallest
// unit has (2 for USD cents, 0 for JPY, 3 for KWD).
func MinorUnitExponent(currency string) int32 {
	c := strings.ToLower(strings.TrimSpace(currency))
	switch {
	case zeroDecimalCurrencies[c]:
		return 0
	case threeDecimalCurrencies[c]:
		return 3
	}
	return 2
}

// ToMinorUnits converts an amount to the currency's smallest unit for Stripe,
// rounding half away from zero (never truncating: 10.005 USD is 1001 cents).
func ToMinorUnits(amount decimal.Decimal, currency string) int64 {
	return amount.Shift(MinorUnitExponent(currency)).Round(0).IntPart()
}

// FromMinorUnits converts a Stripe amount in the currency's smallest unit
// back to a decimal amount.
func FromMinorUnits(units int64, currency string) decimal.Decimal {
	return decimal.New(units, -MinorUnitExponent(currency))
}
