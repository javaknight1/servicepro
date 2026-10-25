# Tax Rates

How sales tax works in ServicePro today: **flat, per-tenant tax rates** that a
user picks on a quote or invoice. There is no zip-code or jurisdiction lookup,
no tax-rate cache and no exemption validation.

## What exists

- **Tax rates table** (`tax_rates`): `name`, `description`, `rate` (a fraction:
  `0.0825` = 8.25%), `tax_type` (`sales_tax`, `vat`, `gst`, `hst`, `exempt`),
  `region` (free text, informational), `is_compound`, `is_active`,
  `effective_date` / `expiry_date`.
- **Ownership:** each tenant manages its own rates and also sees the global
  defaults (`tenant_id IS NULL`), which are read-only. Every query filters
  `tenant_id`; another tenant's rate is a 404.
- **Default rate:** a tenant can set a default tax rate in its settings
  (`DefaultTaxRateID`); quotes and invoices created without an explicit rate get
  it. (Settings UI for it: T235.)
- **Calculation** (`backend/internal/services/invoice_totals.go`, T225) — one
  server-side calculation, all `shopspring/decimal`, rounded half away from
  zero to cents:
  1. line gross = quantity × unit price; line discount (percentage or amount);
     line net = gross − discount
  2. subtotal = sum of line nets
  3. the invoice-level discount is applied **before** tax, allocated to the
     lines pro rata
  4. line tax = (line net − its discount share) × rate, taxable lines only
  5. total = subtotal − discount + tax

A rate that invoices still reference can't be deleted — deactivate it instead.

## API

All routes require auth and a tenant; reads need `tax_rates.read`, writes
`tax_rates.manage`.

| Method | Endpoint                | Description                                             |
| ------ | ----------------------- | ------------------------------------------------------- |
| GET    | `/api/v1/tax-rates`     | List own + global rates (`?active_only=true` to filter) |
| GET    | `/api/v1/tax-rates/:id` | Get one rate                                            |
| POST   | `/api/v1/tax-rates`     | Create a rate                                           |
| PUT    | `/api/v1/tax-rates/:id` | Update a rate (omitted fields unchanged)                |
| DELETE | `/api/v1/tax-rates/:id` | Delete a rate (409 if invoices still use it)            |

Create body:

```json
{
  "name": "Ohio sales tax",
  "rate": "0.0725",
  "tax_type": "sales_tax",
  "region": "OH",
  "is_compound": false,
  "is_active": true
}
```

`name` and `rate` are required; `tax_type` defaults to `sales_tax` and
`is_active` to `true`. Full request/response shapes: Swagger UI at
`/api/docs` (generated from `backend/internal/api/handlers/tax_rate_handler.go`).

Invoices and quotes reference a rate with `tax_rate_id` (or pass a direct
`tax_rate`, which overrides it).

## Not built

- Address- or zip-code-based rate lookup and multi-jurisdiction (state/county/city)
  breakdowns
- Exemption certificates (nonprofit, government, resale)
- Automatic tax via a provider (e.g. Stripe Tax)

None of these are scheduled; record a decision in TODO.md before adding one.
