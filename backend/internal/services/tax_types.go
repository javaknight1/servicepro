package services

import (
	"time"

	"github.com/shopspring/decimal"
)

// TaxJurisdiction represents a tax jurisdiction
type TaxJurisdiction string

const (
	JurisdictionState  TaxJurisdiction = "state"
	JurisdictionCounty TaxJurisdiction = "county"
	JurisdictionCity   TaxJurisdiction = "city"
	JurisdictionLocal  TaxJurisdiction = "local"
)

// TaxExemptionType represents types of tax exemptions
type TaxExemptionType string

const (
	ExemptionNone        TaxExemptionType = "none"
	ExemptionNonProfit   TaxExemptionType = "nonprofit"
	ExemptionGovernment  TaxExemptionType = "government"
	ExemptionResale      TaxExemptionType = "resale"
	ExemptionManufacture TaxExemptionType = "manufacture"
	ExemptionEducational TaxExemptionType = "educational"
)

// TaxRate represents tax rate information for a jurisdiction
type TaxRate struct {
	Jurisdiction TaxJurisdiction `json:"jurisdiction"`
	Rate         decimal.Decimal `json:"rate"`
	Name         string          `json:"name"`
	Code         string          `json:"code"`
}

// TaxRateInfo contains complete tax rate information for a location
type TaxRateInfo struct {
	ZipCode       string          `json:"zip_code"`
	State         string          `json:"state"`
	County        string          `json:"county,omitempty"`
	City          string          `json:"city,omitempty"`
	StateRate     decimal.Decimal `json:"state_rate"`
	CountyRate    decimal.Decimal `json:"county_rate"`
	CityRate      decimal.Decimal `json:"city_rate"`
	LocalRate     decimal.Decimal `json:"local_rate"`
	CombinedRate  decimal.Decimal `json:"combined_rate"`
	Rates         []TaxRate       `json:"rates"`
	EffectiveDate time.Time       `json:"effective_date"`
	LastUpdated   time.Time       `json:"last_updated"`
	Source        string          `json:"source,omitempty"`
}

// TaxCalculationRequest contains information for tax calculation
type TaxCalculationRequest struct {
	Amount          decimal.Decimal  `json:"amount"`
	ZipCode         string           `json:"zip_code"`
	ExemptionType   TaxExemptionType `json:"exemption_type,omitempty"`
	ExemptionNumber string           `json:"exemption_number,omitempty"`
	ProductCategory string           `json:"product_category,omitempty"`
}

// TaxCalculationResult contains the result of a tax calculation
type TaxCalculationResult struct {
	Amount           decimal.Decimal  `json:"amount"`
	TaxableAmount    decimal.Decimal  `json:"taxable_amount"`
	ExemptAmount     decimal.Decimal  `json:"exempt_amount"`
	TaxRate          decimal.Decimal  `json:"tax_rate"`
	TaxAmount        decimal.Decimal  `json:"tax_amount"`
	Total            decimal.Decimal  `json:"total"`
	Breakdown        []TaxBreakdown   `json:"breakdown"`
	ExemptionApplied bool             `json:"exemption_applied"`
	ExemptionType    TaxExemptionType `json:"exemption_type,omitempty"`
	RateInfo         *TaxRateInfo     `json:"rate_info,omitempty"`
}

// TaxBreakdown shows tax breakdown by jurisdiction
type TaxBreakdown struct {
	Jurisdiction TaxJurisdiction `json:"jurisdiction"`
	Name         string          `json:"name"`
	Rate         decimal.Decimal `json:"rate"`
	TaxAmount    decimal.Decimal `json:"tax_amount"`
}

// TaxServiceConfig holds configuration for the tax service
type TaxServiceConfig struct {
	CacheEnabled  bool
	CacheTTL      time.Duration
	DefaultState  string
	FallbackRate  decimal.Decimal
	EnableLogging bool
}

// TaxServiceInterface defines the interface for tax service
type TaxServiceInterface interface {
	// GetTaxRate retrieves tax rate information for a zip code
	GetTaxRate(zipCode string) (*TaxRateInfo, error)

	// CalculateTax calculates tax for an amount and location
	CalculateTax(req *TaxCalculationRequest) (*TaxCalculationResult, error)

	// ValidateExemption validates a tax exemption
	ValidateExemption(exemptionType TaxExemptionType, exemptionNumber string) (bool, error)

	// UpdateTaxRate updates tax rate for a zip code (admin function)
	UpdateTaxRate(zipCode string, rateInfo *TaxRateInfo) error

	// InvalidateCache invalidates cached tax rates for a zip code
	InvalidateCache(zipCode string) error

	// GetCacheStats returns cache statistics
	GetCacheStats() (map[string]interface{}, error)
}
