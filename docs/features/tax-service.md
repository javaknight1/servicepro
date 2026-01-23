# Tax Service

Comprehensive tax calculation with rate lookup, caching, and exemption handling.

## Overview

The Tax Service provides:

- Zip code-based tax rate lookup
- Multi-jurisdiction support (state, county, city)
- Automatic tax calculation
- Exemption handling (nonprofit, government, resale)
- Redis caching for performance

## Features

### Tax Rate Lookup

- Zip code-based rates
- State, county, city, and local rates
- Combined rate calculation
- Effective date tracking

### Tax Calculation

- Precise decimal calculations
- Automatic rounding (2 decimal places)
- Tax breakdown by jurisdiction
- Product category support

### Exemptions

- Nonprofit organizations (EIN validation)
- Government entities
- Resale certificates
- Manufacturing exemptions
- Educational institutions

### Caching

- Redis caching (24-hour TTL default)
- Cache hit/miss statistics
- Manual cache invalidation

## API Endpoints

| Method | Endpoint                  | Description          |
| ------ | ------------------------- | -------------------- |
| GET    | `/tax/rate/:zipCode`      | Get tax rate         |
| POST   | `/tax/calculate`          | Calculate tax        |
| POST   | `/tax/validate-exemption` | Validate exemption   |
| GET    | `/tax/stats`              | Get cache statistics |

## Get Tax Rate

```http
GET /api/v1/tax/rate/90210
```

### Response

```json
{
  "zip_code": "90210",
  "state": "CA",
  "county": "Los Angeles",
  "city": "Beverly Hills",
  "state_rate": "0.0725",
  "county_rate": "0.0100",
  "city_rate": "0.0050",
  "local_rate": "0.0000",
  "combined_rate": "0.0875",
  "rates": [
    {
      "name": "California State Tax",
      "rate": "0.0725",
      "jurisdiction": "state"
    },
    {
      "name": "Los Angeles County Tax",
      "rate": "0.0100",
      "jurisdiction": "county"
    },
    {
      "name": "Beverly Hills City Tax",
      "rate": "0.0050",
      "jurisdiction": "city"
    }
  ],
  "effective_date": "2024-01-01",
  "last_updated": "2024-01-15T10:00:00Z"
}
```

## Calculate Tax

```http
POST /api/v1/tax/calculate
```

### Basic Calculation

```json
{
  "amount": "100.00",
  "zip_code": "90210"
}
```

### Response

```json
{
  "amount": "100.00",
  "tax_rate": "0.0875",
  "tax_amount": "8.75",
  "total": "108.75",
  "breakdown": [
    { "name": "State Tax", "rate": "0.0725", "tax_amount": "7.25" },
    { "name": "County Tax", "rate": "0.0100", "tax_amount": "1.00" },
    { "name": "City Tax", "rate": "0.0050", "tax_amount": "0.50" }
  ],
  "exemption_applied": false
}
```

### With Exemption

```json
{
  "amount": "100.00",
  "zip_code": "90210",
  "exemption_type": "nonprofit",
  "exemption_number": "12-3456789"
}
```

### Response (Exempt)

```json
{
  "amount": "100.00",
  "tax_rate": "0.0875",
  "tax_amount": "0.00",
  "total": "100.00",
  "exemption_applied": true,
  "exemption_type": "nonprofit"
}
```

## Validate Exemption

```http
POST /api/v1/tax/validate-exemption
```

### Request

```json
{
  "exemption_type": "nonprofit",
  "exemption_number": "12-3456789"
}
```

### Response

```json
{
  "valid": true,
  "exemption_type": "nonprofit",
  "message": "Valid EIN format"
}
```

## Exemption Types

### Nonprofit (EIN Format)

```json
{
  "exemption_type": "nonprofit",
  "exemption_number": "12-3456789"
}
```

Format: `XX-XXXXXXX` (9-digit EIN with hyphen after first 2 digits)

### Government

```json
{
  "exemption_type": "government",
  "exemption_number": "GOV123"
}
```

Any valid identifier accepted.

### Resale Certificate

```json
{
  "exemption_type": "resale",
  "exemption_number": "RESALE12345"
}
```

Format: Alphanumeric, 6-15 characters

### Manufacturing

```json
{
  "exemption_type": "manufacture",
  "exemption_number": "MFG12345"
}
```

### Educational

```json
{
  "exemption_type": "educational",
  "exemption_number": "EDU98765"
}
```

## State Tax Rates

Sample rates (actual rates may vary):

| State          | Code | Base Rate |
| -------------- | ---- | --------- |
| California     | CA   | 7.25%     |
| New York       | NY   | 4.00%     |
| Texas          | TX   | 6.25%     |
| Florida        | FL   | 6.00%     |
| Illinois       | IL   | 6.25%     |
| Pennsylvania   | PA   | 6.00%     |
| Ohio           | OH   | 5.75%     |
| Georgia        | GA   | 4.00%     |
| North Carolina | NC   | 4.75%     |
| Michigan       | MI   | 6.00%     |

Note: County and city rates are added to base state rates.

## Error Responses

### Invalid Zip Code

```json
{
  "error": "invalid_zip_code",
  "message": "Invalid zip code format. Use 5-digit (12345) or 9-digit (12345-6789)"
}
```

### Rate Not Found

```json
{
  "error": "rate_not_found",
  "message": "Tax rate not found for zip code 00000"
}
```

### Invalid Exemption

```json
{
  "error": "invalid_exemption",
  "message": "Invalid EIN format. Expected XX-XXXXXXX"
}
```

### Invalid Amount

```json
{
  "error": "invalid_amount",
  "message": "Amount must be a non-negative number"
}
```

## Configuration

### Environment Variables

```bash
# Redis Configuration
REDIS_URL=redis://localhost:6379/0

# Tax Service Configuration
TAX_CACHE_ENABLED=true
TAX_CACHE_TTL=24h
TAX_DEFAULT_STATE=CA
TAX_FALLBACK_RATE=0.0825
```

### Service Configuration

```go
config := &TaxServiceConfig{
    CacheEnabled:  true,
    CacheTTL:      24 * time.Hour,
    DefaultState:  "CA",
    FallbackRate:  decimal.NewFromFloat(0.0825),
    EnableLogging: true,
}
```

## Cache Management

### Get Statistics

```http
GET /api/v1/tax/stats
```

```json
{
  "cache_hits": 1500,
  "cache_misses": 50,
  "hit_rate": 0.9677,
  "cached_zip_codes": 150
}
```

### Invalidate Cache

```http
DELETE /api/v1/tax/cache/:zipCode
```

Invalidates cached rate for specific zip code.

## Integration Example

### With Invoice Service

```go
func (s *InvoiceService) CreateInvoice(req *InvoiceRequest) (*Invoice, error) {
    // Calculate subtotal from line items
    subtotal := calculateSubtotal(req.Items)

    // Calculate tax
    taxReq := &TaxCalculationRequest{
        Amount:  subtotal,
        ZipCode: req.CustomerZipCode,
    }

    taxResult, err := s.taxService.CalculateTax(taxReq)
    if err != nil {
        return nil, err
    }

    // Create invoice
    invoice := &Invoice{
        Subtotal:  subtotal,
        TaxRate:   taxResult.TaxRate,
        TaxAmount: taxResult.TaxAmount,
        Total:     taxResult.Total,
    }

    return invoice, nil
}
```

## Performance

### Expected Latency

| Operation            | Cached | Uncached |
| -------------------- | ------ | -------- |
| Rate lookup          | ~500ns | ~50μs    |
| Tax calculation      | ~1μs   | ~50μs    |
| Exemption validation | ~100ns | ~100ns   |

### Cache Strategy

- Zip code rates cached for 24 hours
- Cache key: `tax:rate:{zipCode}`
- Automatic expiration
- Manual invalidation available

## Best Practices

1. **Always validate zip codes** before calculation
2. **Cache exemption certificates** to avoid repeated validation
3. **Log all calculations** for audit purposes
4. **Update rates periodically** from authoritative sources
5. **Handle rate not found** gracefully with fallback
6. **Use decimal precision** for financial calculations
