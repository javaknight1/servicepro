package services

import (
	"testing"
	"time"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewTaxService(t *testing.T) {
	service := NewTaxService(nil, nil)
	assert.NotNil(t, service)
	assert.NotNil(t, service.config)
	assert.True(t, service.config.CacheEnabled)
	assert.Equal(t, 24*time.Hour, service.config.CacheTTL)
}

func TestNewTaxServiceWithCustomConfig(t *testing.T) {
	config := &TaxServiceConfig{
		CacheEnabled:  false,
		CacheTTL:      1 * time.Hour,
		DefaultState:  "NY",
		FallbackRate:  decimal.NewFromFloat(0.05),
		EnableLogging: false,
	}

	service := NewTaxService(nil, config)
	assert.NotNil(t, service)
	assert.False(t, service.config.CacheEnabled)
	assert.Equal(t, 1*time.Hour, service.config.CacheTTL)
	assert.Equal(t, "NY", service.config.DefaultState)
}

func TestValidateZipCode(t *testing.T) {
	service := NewTaxService(nil, nil)

	tests := []struct {
		name    string
		zipCode string
		wantErr bool
	}{
		{
			name:    "valid 5-digit zip",
			zipCode: "12345",
			wantErr: false,
		},
		{
			name:    "valid 9-digit zip",
			zipCode: "12345-6789",
			wantErr: false,
		},
		{
			name:    "invalid - letters",
			zipCode: "ABCDE",
			wantErr: true,
		},
		{
			name:    "invalid - too short",
			zipCode: "1234",
			wantErr: true,
		},
		{
			name:    "invalid - too long",
			zipCode: "123456",
			wantErr: true,
		},
		{
			name:    "invalid - wrong format",
			zipCode: "12345-678",
			wantErr: true,
		},
		{
			name:    "invalid - empty",
			zipCode: "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := service.validateZipCode(tt.zipCode)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Equal(t, ErrInvalidZipCode, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestGetTaxRate(t *testing.T) {
	service := NewTaxService(nil, &TaxServiceConfig{
		CacheEnabled:  false, // Disable cache for testing
		DefaultState:  "CA",
		FallbackRate:  decimal.NewFromFloat(0.0825),
		EnableLogging: false,
	})

	tests := []struct {
		name    string
		zipCode string
		wantErr bool
	}{
		{
			name:    "California zip code",
			zipCode: "90210",
			wantErr: false,
		},
		{
			name:    "New York zip code",
			zipCode: "10001",
			wantErr: false,
		},
		{
			name:    "Texas zip code",
			zipCode: "75001",
			wantErr: false,
		},
		{
			name:    "invalid zip code",
			zipCode: "invalid",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rateInfo, err := service.GetTaxRate(tt.zipCode)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, rateInfo)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, rateInfo)
				assert.Equal(t, tt.zipCode, rateInfo.ZipCode)
				assert.NotEmpty(t, rateInfo.State)
				assert.True(t, rateInfo.CombinedRate.GreaterThan(decimal.Zero))
				assert.NotEmpty(t, rateInfo.Rates)
			}
		})
	}
}

func TestGetTaxRateRateBreakdown(t *testing.T) {
	service := NewTaxService(nil, &TaxServiceConfig{
		CacheEnabled:  false,
		EnableLogging: false,
	})

	rateInfo, err := service.GetTaxRate("90210")
	require.NoError(t, err)
	require.NotNil(t, rateInfo)

	// Verify rate breakdown
	assert.NotEmpty(t, rateInfo.Rates)
	assert.True(t, rateInfo.StateRate.GreaterThan(decimal.Zero))

	// Verify combined rate equals sum of components
	expectedCombined := rateInfo.StateRate.
		Add(rateInfo.CountyRate).
		Add(rateInfo.CityRate).
		Add(rateInfo.LocalRate)

	assert.True(t, rateInfo.CombinedRate.Equal(expectedCombined))
}

func TestCalculateTax(t *testing.T) {
	service := NewTaxService(nil, &TaxServiceConfig{
		CacheEnabled:  false,
		EnableLogging: false,
	})

	tests := []struct {
		name        string
		request     *TaxCalculationRequest
		wantErr     bool
		checkResult func(*testing.T, *TaxCalculationResult)
	}{
		{
			name: "basic calculation",
			request: &TaxCalculationRequest{
				Amount:  decimal.NewFromFloat(100.00),
				ZipCode: "90210",
			},
			wantErr: false,
			checkResult: func(t *testing.T, result *TaxCalculationResult) {
				assert.True(t, result.Amount.Equal(decimal.NewFromFloat(100.00)))
				assert.True(t, result.TaxableAmount.Equal(decimal.NewFromFloat(100.00)))
				assert.True(t, result.ExemptAmount.Equal(decimal.Zero))
				assert.True(t, result.TaxAmount.GreaterThan(decimal.Zero))
				assert.True(t, result.Total.GreaterThan(result.Amount))
				assert.False(t, result.ExemptionApplied)
				assert.NotEmpty(t, result.Breakdown)
			},
		},
		{
			name: "with nonprofit exemption",
			request: &TaxCalculationRequest{
				Amount:          decimal.NewFromFloat(100.00),
				ZipCode:         "90210",
				ExemptionType:   ExemptionNonProfit,
				ExemptionNumber: "12-3456789",
			},
			wantErr: false,
			checkResult: func(t *testing.T, result *TaxCalculationResult) {
				assert.True(t, result.ExemptionApplied)
				assert.Equal(t, ExemptionNonProfit, result.ExemptionType)
				assert.True(t, result.TaxableAmount.Equal(decimal.Zero))
				assert.True(t, result.ExemptAmount.Equal(decimal.NewFromFloat(100.00)))
				assert.True(t, result.TaxAmount.Equal(decimal.Zero))
				assert.True(t, result.Total.Equal(result.Amount))
			},
		},
		{
			name: "with government exemption",
			request: &TaxCalculationRequest{
				Amount:          decimal.NewFromFloat(100.00),
				ZipCode:         "90210",
				ExemptionType:   ExemptionGovernment,
				ExemptionNumber: "GOV123",
			},
			wantErr: false,
			checkResult: func(t *testing.T, result *TaxCalculationResult) {
				assert.True(t, result.ExemptionApplied)
				assert.Equal(t, ExemptionGovernment, result.ExemptionType)
				assert.True(t, result.TaxAmount.Equal(decimal.Zero))
			},
		},
		{
			name: "invalid amount",
			request: &TaxCalculationRequest{
				Amount:  decimal.NewFromFloat(-100.00),
				ZipCode: "90210",
			},
			wantErr: true,
		},
		{
			name: "invalid zip code",
			request: &TaxCalculationRequest{
				Amount:  decimal.NewFromFloat(100.00),
				ZipCode: "invalid",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := service.CalculateTax(tt.request)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, result)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)
				if tt.checkResult != nil {
					tt.checkResult(t, result)
				}
			}
		})
	}
}

func TestCalculateTaxBreakdown(t *testing.T) {
	service := NewTaxService(nil, &TaxServiceConfig{
		CacheEnabled:  false,
		EnableLogging: false,
	})

	request := &TaxCalculationRequest{
		Amount:  decimal.NewFromFloat(100.00),
		ZipCode: "90210",
	}

	result, err := service.CalculateTax(request)
	require.NoError(t, err)
	require.NotNil(t, result)

	// Verify breakdown exists
	assert.NotEmpty(t, result.Breakdown)

	// Verify breakdown totals match tax amount
	breakdownTotal := decimal.Zero
	for _, item := range result.Breakdown {
		assert.True(t, item.Rate.GreaterThan(decimal.Zero))
		assert.True(t, item.TaxAmount.GreaterThanOrEqual(decimal.Zero))
		assert.NotEmpty(t, item.Name)
		breakdownTotal = breakdownTotal.Add(item.TaxAmount)
	}

	// Breakdown total should equal tax amount (within rounding)
	diff := result.TaxAmount.Sub(breakdownTotal).Abs()
	assert.True(t, diff.LessThanOrEqual(decimal.NewFromFloat(0.01)))
}

func TestValidateExemption(t *testing.T) {
	service := NewTaxService(nil, nil)

	tests := []struct {
		name            string
		exemptionType   TaxExemptionType
		exemptionNumber string
		wantValid       bool
		wantErr         bool
	}{
		{
			name:            "no exemption",
			exemptionType:   ExemptionNone,
			exemptionNumber: "",
			wantValid:       false,
			wantErr:         false,
		},
		{
			name:            "valid nonprofit",
			exemptionType:   ExemptionNonProfit,
			exemptionNumber: "12-3456789",
			wantValid:       true,
			wantErr:         false,
		},
		{
			name:            "invalid nonprofit format",
			exemptionType:   ExemptionNonProfit,
			exemptionNumber: "123456789",
			wantValid:       false,
			wantErr:         true,
		},
		{
			name:            "valid government",
			exemptionType:   ExemptionGovernment,
			exemptionNumber: "GOV123",
			wantValid:       true,
			wantErr:         false,
		},
		{
			name:            "valid resale",
			exemptionType:   ExemptionResale,
			exemptionNumber: "ABC123XYZ",
			wantValid:       true,
			wantErr:         false,
		},
		{
			name:            "invalid resale format",
			exemptionType:   ExemptionResale,
			exemptionNumber: "abc",
			wantValid:       false,
			wantErr:         true,
		},
		{
			name:            "valid manufacture",
			exemptionType:   ExemptionManufacture,
			exemptionNumber: "MFG12345",
			wantValid:       true,
			wantErr:         false,
		},
		{
			name:            "manufacture missing number",
			exemptionType:   ExemptionManufacture,
			exemptionNumber: "",
			wantValid:       false,
			wantErr:         true,
		},
		{
			name:            "invalid exemption type",
			exemptionType:   TaxExemptionType("invalid"),
			exemptionNumber: "123",
			wantValid:       false,
			wantErr:         true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			valid, err := service.ValidateExemption(tt.exemptionType, tt.exemptionNumber)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			assert.Equal(t, tt.wantValid, valid)
		})
	}
}

func TestDifferentJurisdictions(t *testing.T) {
	service := NewTaxService(nil, &TaxServiceConfig{
		CacheEnabled:  false,
		EnableLogging: false,
	})

	zipCodes := []struct {
		zipCode       string
		expectedState string
	}{
		{"90210", "CA"},
		{"10001", "NY"},
		{"75001", "TX"},
		{"32101", "FL"},
		{"60601", "IL"},
	}

	for _, tc := range zipCodes {
		t.Run("jurisdiction_"+tc.zipCode, func(t *testing.T) {
			rateInfo, err := service.GetTaxRate(tc.zipCode)
			require.NoError(t, err)
			assert.Equal(t, tc.expectedState, rateInfo.State)
			assert.True(t, rateInfo.CombinedRate.GreaterThan(decimal.Zero))
		})
	}
}

func TestEdgeCases(t *testing.T) {
	service := NewTaxService(nil, &TaxServiceConfig{
		CacheEnabled:  false,
		EnableLogging: false,
	})

	t.Run("zero amount", func(t *testing.T) {
		request := &TaxCalculationRequest{
			Amount:  decimal.Zero,
			ZipCode: "90210",
		}

		result, err := service.CalculateTax(request)
		require.NoError(t, err)
		assert.True(t, result.TaxAmount.Equal(decimal.Zero))
		assert.True(t, result.Total.Equal(decimal.Zero))
	})

	t.Run("very large amount", func(t *testing.T) {
		request := &TaxCalculationRequest{
			Amount:  decimal.NewFromFloat(1000000.00),
			ZipCode: "90210",
		}

		result, err := service.CalculateTax(request)
		require.NoError(t, err)
		assert.True(t, result.TaxAmount.GreaterThan(decimal.Zero))
		assert.True(t, result.Total.GreaterThan(result.Amount))
	})

	t.Run("decimal precision", func(t *testing.T) {
		request := &TaxCalculationRequest{
			Amount:  decimal.NewFromFloat(99.99),
			ZipCode: "90210",
		}

		result, err := service.CalculateTax(request)
		require.NoError(t, err)

		// Tax amount should be rounded to 2 decimals
		assert.Equal(t, 2, result.TaxAmount.Exponent())
	})
}

func TestGetCacheStats(t *testing.T) {
	service := NewTaxService(nil, &TaxServiceConfig{
		CacheEnabled:  false,
		EnableLogging: false,
	})

	stats, err := service.GetCacheStats()
	require.NoError(t, err)
	assert.NotNil(t, stats)
	assert.Contains(t, stats, "cache_enabled")
	assert.Contains(t, stats, "cache_hits")
	assert.Contains(t, stats, "cache_misses")
	assert.Contains(t, stats, "hit_rate")
}

func TestInvalidateCache(t *testing.T) {
	service := NewTaxService(nil, &TaxServiceConfig{
		CacheEnabled:  false,
		EnableLogging: false,
	})

	// Should not error even without Redis
	err := service.InvalidateCache("90210")
	assert.NoError(t, err)
}

func TestUpdateTaxRate(t *testing.T) {
	service := NewTaxService(nil, &TaxServiceConfig{
		CacheEnabled:  false,
		EnableLogging: false,
	})

	rateInfo := &TaxRateInfo{
		ZipCode:      "90210",
		State:        "CA",
		StateRate:    decimal.NewFromFloat(0.075),
		CombinedRate: decimal.NewFromFloat(0.095),
		LastUpdated:  time.Now(),
	}

	err := service.UpdateTaxRate("90210", rateInfo)
	assert.NoError(t, err)

	// Invalid zip should error
	err = service.UpdateTaxRate("invalid", rateInfo)
	assert.Error(t, err)
}

func TestMultipleExemptions(t *testing.T) {
	service := NewTaxService(nil, &TaxServiceConfig{
		CacheEnabled:  false,
		EnableLogging: false,
	})

	exemptions := []struct {
		exemptionType   TaxExemptionType
		exemptionNumber string
	}{
		{ExemptionNonProfit, "12-3456789"},
		{ExemptionGovernment, "GOV123"},
		{ExemptionResale, "RESALE123"},
	}

	for _, exemption := range exemptions {
		t.Run(string(exemption.exemptionType), func(t *testing.T) {
			request := &TaxCalculationRequest{
				Amount:          decimal.NewFromFloat(100.00),
				ZipCode:         "90210",
				ExemptionType:   exemption.exemptionType,
				ExemptionNumber: exemption.exemptionNumber,
			}

			result, err := service.CalculateTax(request)
			require.NoError(t, err)
			assert.True(t, result.ExemptionApplied)
			assert.True(t, result.TaxAmount.Equal(decimal.Zero))
		})
	}
}
