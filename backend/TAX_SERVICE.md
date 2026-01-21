# Tax Calculation Service

A comprehensive tax calculation service with rate lookup, caching, exemption handling, and multi-jurisdiction support.

## Features

### ✅ Tax Rate Lookup

- Zip code-based rate lookup
- Multi-jurisdiction support (state, county, city, local)
- Combined rate calculation
- Effective date tracking
- Rate history support

### ✅ Tax Calculation

- Precise decimal calculations using `shopspring/decimal`
- Automatic rounding to 2 decimal places
- Tax breakdown by jurisdiction
- Support for exemptions
- Product category handling

### ✅ Exemption Handling

- Nonprofit organizations (EIN validation)
- Government entities
- Resale certificates
- Manufacturing exemptions
- Educational institutions
- Custom exemption validation

### ✅ Redis Caching

- Configurable TTL (default: 24 hours)
- Cache hit/miss tracking
- Manual cache invalidation
- Cache statistics
- Optional caching (can be disabled)

### ✅ Performance Optimization

- In-memory state rate lookup
- Redis caching layer
- Minimal database queries
- Efficient rate calculations

### ✅ Validation & Error Handling

- Zip code format validation (5 or 9 digit)
- Amount validation (non-negative)
- Exemption number format validation
- Clear error messages
- Type-safe error handling

## Architecture

```
┌──────────────────┐
│   API Handler    │
└────────┬─────────┘
         │
         ▼
┌──────────────────┐
│   Tax Service    │
└────────┬─────────┘
         │
    ┌────┴────┐
    ▼         ▼
┌────────┐ ┌───────┐
│ Redis  │ │  DB   │
│ Cache  │ │       │
└────────┘ └───────┘
```

## Usage

### Initialize the Service

```go
import (
    "github.com/javaknight1/servicepro/backend/internal/services"
    "github.com/redis/go-redis/v9"
    "github.com/shopspring/decimal"
    "time"
)

// Create Redis client
redisClient := redis.NewClient(&redis.Options{
    Addr: "localhost:6379",
})

// Configure service
config := &services.TaxServiceConfig{
    CacheEnabled:  true,
    CacheTTL:      24 * time.Hour,
    DefaultState:  "CA",
    FallbackRate:  decimal.NewFromFloat(0.0825),
    EnableLogging: true,
}

// Create service
taxService := services.NewTaxService(redisClient, config)
```

### Get Tax Rate by Zip Code

```go
// Lookup tax rate
rateInfo, err := taxService.GetTaxRate("90210")
if err != nil {
    log.Fatal(err)
}

fmt.Printf("Combined Rate: %s\n", rateInfo.CombinedRate)
fmt.Printf("State Rate: %s\n", rateInfo.StateRate)
fmt.Printf("County Rate: %s\n", rateInfo.CountyRate)
fmt.Printf("City Rate: %s\n", rateInfo.CityRate)

// Access breakdown
for _, rate := range rateInfo.Rates {
    fmt.Printf("%s: %s\n", rate.Name, rate.Rate)
}
```

### Calculate Tax

```go
// Basic calculation
request := &services.TaxCalculationRequest{
    Amount:  decimal.NewFromFloat(100.00),
    ZipCode: "90210",
}

result, err := taxService.CalculateTax(request)
if err != nil {
    log.Fatal(err)
}

fmt.Printf("Amount: %s\n", result.Amount)
fmt.Printf("Tax: %s\n", result.TaxAmount)
fmt.Printf("Total: %s\n", result.Total)

// View breakdown
for _, breakdown := range result.Breakdown {
    fmt.Printf("%s (%s): %s\n",
        breakdown.Name,
        breakdown.Rate,
        breakdown.TaxAmount,
    )
}
```

### Calculate Tax with Exemption

```go
// Nonprofit exemption
request := &services.TaxCalculationRequest{
    Amount:          decimal.NewFromFloat(100.00),
    ZipCode:         "90210",
    ExemptionType:   services.ExemptionNonProfit,
    ExemptionNumber: "12-3456789", // EIN format
}

result, err := taxService.CalculateTax(request)
if err != nil {
    log.Fatal(err)
}

if result.ExemptionApplied {
    fmt.Printf("Exemption applied: %s\n", result.ExemptionType)
    fmt.Printf("Tax: $0.00 (exempt)\n")
}

// Government exemption
request = &services.TaxCalculationRequest{
    Amount:          decimal.NewFromFloat(100.00),
    ZipCode:         "90210",
    ExemptionType:   services.ExemptionGovernment,
    ExemptionNumber: "GOV123",
}

// Resale certificate
request = &services.TaxCalculationRequest{
    Amount:          decimal.NewFromFloat(100.00),
    ZipCode:         "90210",
    ExemptionType:   services.ExemptionResale,
    ExemptionNumber: "RESALE12345",
}
```

### Validate Exemption

```go
// Validate nonprofit EIN
valid, err := taxService.ValidateExemption(
    services.ExemptionNonProfit,
    "12-3456789",
)

if err != nil {
    log.Printf("Validation error: %v", err)
}

if valid {
    fmt.Println("Exemption is valid")
}
```

### Cache Management

```go
// Get cache statistics
stats, err := taxService.GetCacheStats()
if err != nil {
    log.Fatal(err)
}

fmt.Printf("Cache Hits: %v\n", stats["cache_hits"])
fmt.Printf("Cache Misses: %v\n", stats["cache_misses"])
fmt.Printf("Hit Rate: %.2f%%\n", stats["hit_rate"].(float64)*100)

// Invalidate cache for specific zip code
err = taxService.InvalidateCache("90210")
if err != nil {
    log.Fatal(err)
}
```

### Update Tax Rate (Admin Function)

```go
// Create new rate info
rateInfo := &services.TaxRateInfo{
    ZipCode:      "90210",
    State:        "CA",
    County:       "Los Angeles",
    City:         "Beverly Hills",
    StateRate:    decimal.NewFromFloat(0.0725),
    CountyRate:   decimal.NewFromFloat(0.0100),
    CityRate:     decimal.NewFromFloat(0.0050),
    LocalRate:    decimal.Zero,
    CombinedRate: decimal.NewFromFloat(0.0875),
    EffectiveDate: time.Now(),
    LastUpdated:  time.Now(),
    Source:       "Tax Authority API",
}

// Update rate
err := taxService.UpdateTaxRate("90210", rateInfo)
if err != nil {
    log.Fatal(err)
}
```

## Exemption Types

### ExemptionNonProfit

- Format: `XX-XXXXXXX` (EIN format)
- Example: `12-3456789`
- Validation: Strict EIN format matching

### ExemptionGovernment

- Format: Any string
- Example: `GOV123`
- Validation: Always valid (government entities)

### ExemptionResale

- Format: Alphanumeric, 6-15 characters
- Example: `RESALE12345`
- Validation: Format matching

### ExemptionManufacture

- Format: Any non-empty string
- Example: `MFG12345`
- Validation: Non-empty check

### ExemptionEducational

- Format: Any non-empty string
- Example: `EDU98765`
- Validation: Non-empty check

## Tax Rate Structure

```go
type TaxRateInfo struct {
    ZipCode       string          // "90210"
    State         string          // "CA"
    County        string          // "Los Angeles"
    City          string          // "Beverly Hills"
    StateRate     decimal.Decimal // 0.0725 (7.25%)
    CountyRate    decimal.Decimal // 0.0100 (1.00%)
    CityRate      decimal.Decimal // 0.0050 (0.50%)
    LocalRate     decimal.Decimal // 0.0000 (0.00%)
    CombinedRate  decimal.Decimal // 0.0875 (8.75%)
    Rates         []TaxRate       // Breakdown by jurisdiction
    EffectiveDate time.Time       // When rate became effective
    LastUpdated   time.Time       // Last update timestamp
    Source        string          // "Tax Authority API"
}
```

## Supported States

Current state rates (sample data):

| State          | Code | Rate  |
| -------------- | ---- | ----- |
| California     | CA   | 7.25% |
| New York       | NY   | 4.00% |
| Texas          | TX   | 6.25% |
| Florida        | FL   | 6.00% |
| Illinois       | IL   | 6.25% |
| Pennsylvania   | PA   | 6.00% |
| Ohio           | OH   | 5.75% |
| Georgia        | GA   | 4.00% |
| North Carolina | NC   | 4.75% |
| Michigan       | MI   | 6.00% |

**Note:** In production, rates would be loaded from a database or external API.

## Error Handling

```go
// Check for specific errors
result, err := taxService.CalculateTax(request)
if err != nil {
    switch {
    case errors.Is(err, services.ErrInvalidZipCode):
        // Handle invalid zip code
    case errors.Is(err, services.ErrTaxRateNotFound):
        // Handle rate not found
    case errors.Is(err, services.ErrInvalidExemption):
        // Handle invalid exemption
    case errors.Is(err, services.ErrInvalidAmount):
        // Handle invalid amount
    default:
        // Handle other errors
    }
}
```

## Testing

### Run All Tests

```bash
go test ./internal/services/ -run "Tax" -v
```

### Test Coverage

```bash
go test ./internal/services/ -run "Tax" -cover
```

### Test Cases Included

- **Validation Tests** (7 test cases)

  - Valid 5-digit zip codes
  - Valid 9-digit zip codes
  - Invalid formats
  - Empty/null values

- **Rate Lookup Tests** (4 test cases)

  - Different states
  - Rate breakdown
  - Combined rate calculation
  - Invalid zip codes

- **Calculation Tests** (5 test cases)

  - Basic calculation
  - With exemptions
  - Invalid amounts
  - Zero amounts
  - Large amounts

- **Exemption Tests** (10 test cases)

  - All exemption types
  - Valid/invalid formats
  - EIN validation
  - Resale certificate validation
  - Government exemptions

- **Edge Cases** (3 test cases)

  - Zero amount
  - Very large amounts
  - Decimal precision

- **Cache Tests** (2 test cases)
  - Cache statistics
  - Cache invalidation

## Performance Benchmarks

```bash
# Run benchmarks
go test ./internal/services/ -bench=Tax -benchmem
```

Expected performance:

- Rate lookup (cached): ~500ns
- Rate lookup (uncached): ~50μs
- Tax calculation: ~1μs
- Exemption validation: ~100ns

## Configuration

### TaxServiceConfig

```go
type TaxServiceConfig struct {
    CacheEnabled  bool            // Enable Redis caching
    CacheTTL      time.Duration   // Cache time-to-live
    DefaultState  string          // Fallback state code
    FallbackRate  decimal.Decimal // Fallback tax rate
    EnableLogging bool            // Enable debug logging
}
```

### Default Configuration

```go
config := &TaxServiceConfig{
    CacheEnabled:  true,
    CacheTTL:      24 * time.Hour,
    DefaultState:  "CA",
    FallbackRate:  decimal.NewFromFloat(0.0825),
    EnableLogging: true,
}
```

## Integration Example

### With Quote Service

```go
// In quote service
func (s *QuoteService) CreateQuote(req *QuoteRequest, userID uuid.UUID) (*QuoteResponse, error) {
    // ... create quote ...

    // Calculate tax
    taxReq := &services.TaxCalculationRequest{
        Amount:  quote.Subtotal,
        ZipCode: customer.ZipCode,
    }

    taxResult, err := s.taxService.CalculateTax(taxReq)
    if err != nil {
        return nil, fmt.Errorf("tax calculation failed: %w", err)
    }

    // Update quote with tax
    quote.TaxRate = taxResult.TaxRate
    quote.TaxAmount = taxResult.TaxAmount
    quote.Total = taxResult.Total

    // Save quote...
    return quoteResponse, nil
}
```

## Production Considerations

### Rate Data Source

In production, implement a rate provider interface:

```go
type TaxRateProvider interface {
    GetRate(zipCode string) (*TaxRateInfo, error)
    UpdateRate(zipCode string, rate *TaxRateInfo) error
}

// Implementations:
// - DatabaseProvider (PostgreSQL)
// - APIProvider (TaxJar, Avalara, etc.)
// - FileProvider (CSV/JSON files)
```

### Monitoring

```go
// Add metrics
metrics.Increment("tax.calculation.total")
metrics.Increment("tax.cache.hit")
metrics.Timing("tax.calculation.duration", duration)
```

### Audit Logging

```go
// Log tax calculations
logger.Info("tax calculation",
    "amount", request.Amount,
    "zipCode", request.ZipCode,
    "taxAmount", result.TaxAmount,
    "exemption", result.ExemptionApplied,
)
```

## License

MIT
