package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"regexp"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/shopspring/decimal"
)

var (
	// ErrInvalidZipCode is returned when zip code is invalid
	ErrInvalidZipCode = errors.New("invalid zip code")
	// ErrTaxRateNotFound is returned when tax rate is not found
	ErrTaxRateNotFound = errors.New("tax rate not found")
	// ErrInvalidExemption is returned when exemption is invalid
	ErrInvalidExemption = errors.New("invalid tax exemption")
	// ErrInvalidAmount is returned when amount is invalid
	ErrInvalidAmount = errors.New("invalid amount")
)

// TaxService implements TaxServiceInterface
type TaxService struct {
	redis  *redis.Client
	config *TaxServiceConfig
	// In-memory fallback rates by state
	stateRates map[string]decimal.Decimal
	// Cache stats
	cacheHits   int64
	cacheMisses int64
}

// NewTaxService creates a new tax service
func NewTaxService(redisClient *redis.Client, config *TaxServiceConfig) *TaxService {
	if config == nil {
		config = &TaxServiceConfig{
			CacheEnabled:  true,
			CacheTTL:      24 * time.Hour,
			DefaultState:  "CA",
			FallbackRate:  decimal.NewFromFloat(0.0825),
			EnableLogging: true,
		}
	}

	// Initialize state rates (sample data - would come from database in production)
	stateRates := map[string]decimal.Decimal{
		"CA": decimal.NewFromFloat(0.0725), // California
		"NY": decimal.NewFromFloat(0.0400), // New York
		"TX": decimal.NewFromFloat(0.0625), // Texas
		"FL": decimal.NewFromFloat(0.0600), // Florida
		"IL": decimal.NewFromFloat(0.0625), // Illinois
		"PA": decimal.NewFromFloat(0.0600), // Pennsylvania
		"OH": decimal.NewFromFloat(0.0575), // Ohio
		"GA": decimal.NewFromFloat(0.0400), // Georgia
		"NC": decimal.NewFromFloat(0.0475), // North Carolina
		"MI": decimal.NewFromFloat(0.0600), // Michigan
	}

	return &TaxService{
		redis:      redisClient,
		config:     config,
		stateRates: stateRates,
	}
}

// GetTaxRate retrieves tax rate information for a zip code
func (s *TaxService) GetTaxRate(zipCode string) (*TaxRateInfo, error) {
	// Validate zip code
	if err := s.validateZipCode(zipCode); err != nil {
		return nil, err
	}

	// Try cache first
	if s.config.CacheEnabled {
		if rateInfo, err := s.getTaxRateFromCache(zipCode); err == nil {
			s.cacheHits++
			if s.config.EnableLogging {
				log.Printf("Cache hit for zip code: %s", zipCode)
			}
			return rateInfo, nil
		}
		s.cacheMisses++
	}

	// Fetch from "database" (using mock data for now)
	rateInfo, err := s.fetchTaxRateFromDatabase(zipCode)
	if err != nil {
		return nil, err
	}

	// Cache the result
	if s.config.CacheEnabled {
		if err := s.cacheTaxRate(zipCode, rateInfo); err != nil {
			if s.config.EnableLogging {
				log.Printf("Failed to cache tax rate for %s: %v", zipCode, err)
			}
		}
	}

	return rateInfo, nil
}

// CalculateTax calculates tax for an amount and location
func (s *TaxService) CalculateTax(req *TaxCalculationRequest) (*TaxCalculationResult, error) {
	// Validate request
	if err := s.validateCalculationRequest(req); err != nil {
		return nil, err
	}

	// Get tax rate for location
	rateInfo, err := s.GetTaxRate(req.ZipCode)
	if err != nil {
		return nil, fmt.Errorf("failed to get tax rate: %w", err)
	}

	// Initialize result
	result := &TaxCalculationResult{
		Amount:           req.Amount,
		TaxableAmount:    req.Amount,
		ExemptAmount:     decimal.Zero,
		TaxRate:          rateInfo.CombinedRate,
		RateInfo:         rateInfo,
		ExemptionApplied: false,
	}

	// Apply exemption if provided
	if req.ExemptionType != ExemptionNone && req.ExemptionType != "" {
		if valid, err := s.ValidateExemption(req.ExemptionType, req.ExemptionNumber); err == nil && valid {
			result.ExemptionApplied = true
			result.ExemptionType = req.ExemptionType
			result.ExemptAmount = req.Amount
			result.TaxableAmount = decimal.Zero
		}
	}

	// Calculate tax amount
	result.TaxAmount = result.TaxableAmount.Mul(result.TaxRate).Round(2)
	result.Total = result.Amount.Add(result.TaxAmount)

	// Build breakdown
	result.Breakdown = s.buildTaxBreakdown(result.TaxableAmount, rateInfo)

	return result, nil
}

// ValidateExemption validates a tax exemption
func (s *TaxService) ValidateExemption(exemptionType TaxExemptionType, exemptionNumber string) (bool, error) {
	// Basic validation
	switch exemptionType {
	case ExemptionNone:
		return false, nil

	case ExemptionNonProfit:
		// Non-profit exemption should have an EIN format (XX-XXXXXXX)
		if matched, _ := regexp.MatchString(`^\d{2}-\d{7}$`, exemptionNumber); !matched {
			return false, ErrInvalidExemption
		}
		return true, nil

	case ExemptionGovernment:
		// Government exemptions are typically always valid
		return true, nil

	case ExemptionResale:
		// Resale certificate should be alphanumeric
		if matched, _ := regexp.MatchString(`^[A-Z0-9]{6,15}$`, exemptionNumber); !matched {
			return false, ErrInvalidExemption
		}
		return true, nil

	case ExemptionManufacture, ExemptionEducational:
		// These require valid certificate numbers
		if exemptionNumber == "" {
			return false, ErrInvalidExemption
		}
		return true, nil

	default:
		return false, ErrInvalidExemption
	}
}

// UpdateTaxRate updates tax rate for a zip code
func (s *TaxService) UpdateTaxRate(zipCode string, rateInfo *TaxRateInfo) error {
	if err := s.validateZipCode(zipCode); err != nil {
		return err
	}

	// In production, this would update the database
	// For now, just update the cache
	if s.config.CacheEnabled {
		return s.cacheTaxRate(zipCode, rateInfo)
	}

	return nil
}

// InvalidateCache invalidates cached tax rates for a zip code
func (s *TaxService) InvalidateCache(zipCode string) error {
	if !s.config.CacheEnabled || s.redis == nil {
		return nil
	}

	ctx := context.Background()
	cacheKey := fmt.Sprintf("tax:rate:%s", zipCode)

	if err := s.redis.Del(ctx, cacheKey).Err(); err != nil {
		return fmt.Errorf("failed to invalidate cache: %w", err)
	}

	if s.config.EnableLogging {
		log.Printf("Invalidated cache for zip code: %s", zipCode)
	}

	return nil
}

// GetCacheStats returns cache statistics
func (s *TaxService) GetCacheStats() (map[string]interface{}, error) {
	stats := map[string]interface{}{
		"cache_enabled": s.config.CacheEnabled,
		"cache_hits":    s.cacheHits,
		"cache_misses":  s.cacheMisses,
		"hit_rate":      0.0,
	}

	total := s.cacheHits + s.cacheMisses
	if total > 0 {
		stats["hit_rate"] = float64(s.cacheHits) / float64(total)
	}

	return stats, nil
}

// Private helper methods

func (s *TaxService) validateZipCode(zipCode string) error {
	// US ZIP code validation (5 digits or 5+4 format)
	matched, err := regexp.MatchString(`^\d{5}(-\d{4})?$`, zipCode)
	if err != nil || !matched {
		return ErrInvalidZipCode
	}
	return nil
}

func (s *TaxService) validateCalculationRequest(req *TaxCalculationRequest) error {
	if req.Amount.LessThan(decimal.Zero) {
		return ErrInvalidAmount
	}

	if err := s.validateZipCode(req.ZipCode); err != nil {
		return err
	}

	return nil
}

func (s *TaxService) getTaxRateFromCache(zipCode string) (*TaxRateInfo, error) {
	if s.redis == nil {
		return nil, errors.New("redis client not available")
	}

	ctx := context.Background()
	cacheKey := fmt.Sprintf("tax:rate:%s", zipCode)

	data, err := s.redis.Get(ctx, cacheKey).Result()
	if err != nil {
		return nil, err
	}

	var rateInfo TaxRateInfo
	if err := json.Unmarshal([]byte(data), &rateInfo); err != nil {
		return nil, err
	}

	return &rateInfo, nil
}

func (s *TaxService) cacheTaxRate(zipCode string, rateInfo *TaxRateInfo) error {
	if s.redis == nil {
		return nil
	}

	ctx := context.Background()
	cacheKey := fmt.Sprintf("tax:rate:%s", zipCode)

	data, err := json.Marshal(rateInfo)
	if err != nil {
		return err
	}

	return s.redis.Set(ctx, cacheKey, data, s.config.CacheTTL).Err()
}

func (s *TaxService) fetchTaxRateFromDatabase(zipCode string) (*TaxRateInfo, error) {
	// In production, this would query a real database
	// For now, use mock data based on zip code patterns

	// Extract state from zip code (simplified mapping)
	stateCode := s.zipToState(zipCode)
	stateRate, exists := s.stateRates[stateCode]
	if !exists {
		stateRate = s.config.FallbackRate
	}

	// Mock data - different rates based on zip code
	countyRate := decimal.NewFromFloat(0.01) // 1% county
	cityRate := decimal.NewFromFloat(0.0025) // 0.25% city
	localRate := decimal.NewFromFloat(0.0)   // No local tax

	// Some zip codes have special rates
	if zipCode[:3] == "900" { // Los Angeles area
		countyRate = decimal.NewFromFloat(0.0125)
		cityRate = decimal.NewFromFloat(0.005)
	} else if zipCode[:3] == "100" { // New York City
		cityRate = decimal.NewFromFloat(0.04875)
	}

	combinedRate := stateRate.Add(countyRate).Add(cityRate).Add(localRate)

	rateInfo := &TaxRateInfo{
		ZipCode:       zipCode,
		State:         stateCode,
		County:        "Sample County",
		City:          "Sample City",
		StateRate:     stateRate,
		CountyRate:    countyRate,
		CityRate:      cityRate,
		LocalRate:     localRate,
		CombinedRate:  combinedRate,
		EffectiveDate: time.Now().AddDate(0, 0, -30),
		LastUpdated:   time.Now(),
		Source:        "Mock Data",
		Rates: []TaxRate{
			{
				Jurisdiction: JurisdictionState,
				Rate:         stateRate,
				Name:         stateCode + " State Tax",
				Code:         stateCode,
			},
			{
				Jurisdiction: JurisdictionCounty,
				Rate:         countyRate,
				Name:         "County Tax",
				Code:         "COUNTY",
			},
			{
				Jurisdiction: JurisdictionCity,
				Rate:         cityRate,
				Name:         "City Tax",
				Code:         "CITY",
			},
		},
	}

	return rateInfo, nil
}

func (s *TaxService) zipToState(zipCode string) string {
	// Simplified zip code to state mapping (first 3 digits)
	prefix := zipCode[:3]

	// Sample mappings (would be complete in production)
	zipRanges := map[string]string{
		"900": "CA", "901": "CA", "902": "CA", "903": "CA", "904": "CA",
		"100": "NY", "101": "NY", "102": "NY", "103": "NY",
		"750": "TX", "751": "TX", "752": "TX",
		"320": "FL", "321": "FL", "322": "FL",
		"606": "IL", "607": "IL", "608": "IL",
	}

	if state, exists := zipRanges[prefix]; exists {
		return state
	}

	return s.config.DefaultState
}

func (s *TaxService) buildTaxBreakdown(taxableAmount decimal.Decimal, rateInfo *TaxRateInfo) []TaxBreakdown {
	breakdown := []TaxBreakdown{}

	for _, rate := range rateInfo.Rates {
		if rate.Rate.GreaterThan(decimal.Zero) {
			taxAmount := taxableAmount.Mul(rate.Rate).Round(2)
			breakdown = append(breakdown, TaxBreakdown{
				Jurisdiction: rate.Jurisdiction,
				Name:         rate.Name,
				Rate:         rate.Rate,
				TaxAmount:    taxAmount,
			})
		}
	}

	return breakdown
}
