package stripe

import (
	"context"
	"testing"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConfig_Validate(t *testing.T) {
	t.Run("Valid test configuration", func(t *testing.T) {
		config := &Config{
			SecretKey:         "sk_test_fake123",
			PublishableKey:    "pk_test_fake123",
			WebhookSecret:     "whsec_test_secret",
			Environment:       EnvironmentTest,
			MaxNetworkRetries: 3,
			ConnectTimeout:    30,
			Timeout:           80,
		}

		err := config.Validate()
		assert.NoError(t, err)
	})

	t.Run("Valid live configuration", func(t *testing.T) {
		config := &Config{
			SecretKey:         "sk_live_fake123",
			PublishableKey:    "pk_live_fake123",
			WebhookSecret:     "whsec_live_secret",
			Environment:       EnvironmentLive,
			MaxNetworkRetries: 3,
			ConnectTimeout:    30,
			Timeout:           80,
		}

		err := config.Validate()
		assert.NoError(t, err)
	})

	t.Run("Missing secret key", func(t *testing.T) {
		config := &Config{
			PublishableKey: "pk_test_fake123",
			WebhookSecret:  "whsec_test_secret",
		}

		err := config.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "secret key is required")
	})

	t.Run("Missing publishable key", func(t *testing.T) {
		config := &Config{
			SecretKey:     "sk_test_fake123",
			WebhookSecret: "whsec_test_secret",
		}

		err := config.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "publishable key is required")
	})

	t.Run("Missing webhook secret", func(t *testing.T) {
		config := &Config{
			SecretKey:      "sk_test_fake123",
			PublishableKey: "pk_test_fake123",
		}

		err := config.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "webhook secret is required")
	})

	t.Run("Mismatched test/live keys", func(t *testing.T) {
		config := &Config{
			SecretKey:         "sk_test_fake123",
			PublishableKey:    "pk_live_fake123", // Live key with test secret
			WebhookSecret:     "whsec_test_secret",
			MaxNetworkRetries: 3,
			ConnectTimeout:    30,
			Timeout:           80,
		}

		err := config.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "environments do not match")
	})

	t.Run("Invalid secret key format", func(t *testing.T) {
		config := &Config{
			SecretKey:         "invalid",
			PublishableKey:    "pk_test_fake123",
			WebhookSecret:     "whsec_test_secret",
			MaxNetworkRetries: 3,
			ConnectTimeout:    30,
			Timeout:           80,
		}

		err := config.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid secret key format")
	})

	t.Run("Negative max retries", func(t *testing.T) {
		config := &Config{
			SecretKey:         "sk_test_fake123",
			PublishableKey:    "pk_test_fake123",
			WebhookSecret:     "whsec_test_secret",
			MaxNetworkRetries: -1,
			ConnectTimeout:    30,
			Timeout:           80,
		}

		err := config.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "max network retries must be non-negative")
	})
}

func TestConfig_IsTestMode(t *testing.T) {
	config := &Config{
		Environment: EnvironmentTest,
	}

	assert.True(t, config.IsTestMode())
	assert.False(t, config.IsLiveMode())
}

func TestConfig_IsLiveMode(t *testing.T) {
	config := &Config{
		Environment: EnvironmentLive,
	}

	assert.True(t, config.IsLiveMode())
	assert.False(t, config.IsTestMode())
}

func TestConfig_MaskSecretKey(t *testing.T) {
	config := &Config{
		SecretKey: "sk_test_fake123",
	}

	masked := config.MaskSecretKey()
	assert.Contains(t, masked, "sk_test_fake")
	assert.Contains(t, masked, "...")
}

func TestConvertDecimalToCents(t *testing.T) {
	tests := []struct {
		name     string
		amount   decimal.Decimal
		expected int64
	}{
		{
			name:     "Simple amount",
			amount:   decimal.NewFromFloat(10.50),
			expected: 1050,
		},
		{
			name:     "Zero amount",
			amount:   decimal.NewFromFloat(0),
			expected: 0,
		},
		{
			name:     "Large amount",
			amount:   decimal.NewFromFloat(9999.99),
			expected: 999999,
		},
		{
			name:     "Single cent",
			amount:   decimal.NewFromFloat(0.01),
			expected: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ConvertDecimalToCents(tt.amount)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestConvertCentsToDecimal(t *testing.T) {
	tests := []struct {
		name     string
		cents    int64
		expected string
	}{
		{
			name:     "Simple amount",
			cents:    1050,
			expected: "10.5",
		},
		{
			name:     "Zero amount",
			cents:    0,
			expected: "0",
		},
		{
			name:     "Large amount",
			cents:    999999,
			expected: "9999.99",
		},
		{
			name:     "Single cent",
			cents:    1,
			expected: "0.01",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ConvertCentsToDecimal(tt.cents)
			assert.Equal(t, tt.expected, result.String())
		})
	}
}

func TestNewClient(t *testing.T) {
	t.Run("Valid configuration", func(t *testing.T) {
		config := &Config{
			SecretKey:         "sk_test_fake123",
			PublishableKey:    "pk_test_fake123",
			WebhookSecret:     "whsec_test_secret",
			Environment:       EnvironmentTest,
			MaxNetworkRetries: 3,
			ConnectTimeout:    30,
			Timeout:           80,
			RequestsPerSecond: 100,
		}

		client, err := NewClient(config)
		require.NoError(t, err)
		require.NotNil(t, client)
		assert.Equal(t, config, client.config)
	})

	t.Run("Invalid configuration", func(t *testing.T) {
		config := &Config{
			SecretKey: "", // Invalid
		}

		client, err := NewClient(config)
		assert.Error(t, err)
		assert.Nil(t, client)
	})
}

func TestRateLimiter(t *testing.T) {
	t.Run("Basic rate limiting", func(t *testing.T) {
		rl := newRateLimiter(10) // 10 requests per second
		defer rl.Stop()

		ctx := context.Background()

		// Should be able to make 10 requests immediately
		for i := 0; i < 10; i++ {
			err := rl.Wait(ctx)
			assert.NoError(t, err)
		}
	})

	t.Run("Context cancellation", func(t *testing.T) {
		rl := newRateLimiter(1)
		defer rl.Stop()

		// First, consume the initial token
		ctx := context.Background()
		err := rl.Wait(ctx)
		assert.NoError(t, err)

		// Now create a cancelled context - the next Wait should fail
		cancelCtx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		err = rl.Wait(cancelCtx)
		assert.Error(t, err)
		assert.Equal(t, context.Canceled, err)
	})
}

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()

	assert.NotNil(t, config)
	assert.Equal(t, 3, config.MaxNetworkRetries)
	assert.Equal(t, 100, config.RequestsPerSecond)
	assert.Equal(t, "info", config.LogLevel)
}
