# Quotes API

REST API for quotes: create, send, accept or reject, revise, expire, and turn
an accepted quote into an invoice. The OpenAPI spec (Swagger UI at
`/api/docs`) is the source of truth for request/response shapes.

## Endpoints

All under `/api/v1`, authenticated and tenant-scoped.

| Method | Endpoint                      | Description                                  |
| ------ | ----------------------------- | -------------------------------------------- |
| GET    | `/quotes`                     | List quotes (pagination, filters, sorting)   |
| GET    | `/quotes/stats`               | Counts by status                             |
| GET    | `/quotes/number/:quoteNumber` | Get a quote by its number                    |
| GET    | `/quotes/:id`                 | Get a quote with its line items              |
| POST   | `/quotes`                     | Create a quote (totals computed server-side) |
| PUT    | `/quotes/:id`                 | Update a draft quote                         |
| DELETE | `/quotes/:id`                 | Soft delete (drafts only)                    |
| POST   | `/quotes/:id/send`            | draft → sent (generates the PDF, emails it)  |
| POST   | `/quotes/:id/accept`          | sent/viewed → accepted                       |
| POST   | `/quotes/:id/reject`          | sent/viewed → declined                       |
| POST   | `/quotes/:id/revise`          | sent/viewed/expired/declined → draft         |
| GET    | `/quotes/:id/pdf`             | Quote PDF                                    |
| GET    | `/customers/:id/quotes`       | A customer's quotes                          |

There is no `/quotes/:id/convert`, `/decline`, `/view` or `/history` endpoint
and no WebSocket feed. To invoice an accepted quote, create an invoice with its
`quote_id` (below).

## Status Flow

```
draft ──send──► sent ──accept──► accepted   (then: POST /invoices with quote_id, once)
  ▲              │  └──reject──► declined
  │              │
  │              └──(valid_until passed, hourly job)──► expired
  │
  └──revise── sent / viewed / declined / expired
```

| Status   | Meaning                                                 |
| -------- | ------------------------------------------------------- |
| draft    | Being edited — the only editable/deletable status       |
| sent     | Sent to the customer                                    |
| viewed   | Customer viewed it (status exists; nothing sets it yet) |
| accepted | Accepted — can be invoiced once                         |
| declined | Rejected via `POST /quotes/:id/reject`                  |
| expired  | `valid_until` passed while sent/viewed                  |

Expiry runs as `POST /internal/jobs/quote-expiry` (hourly from
`scheduled-jobs.yml`): one conditional update that moves sent/viewed quotes past
`valid_until` to `expired`, so a quote accepted or declined in the meantime is
untouched.

## Create Quote

```http
POST /api/v1/quotes
```

```json
{
  "customer_id": "uuid",
  "valid_until": "2026-11-30T00:00:00Z",
  "tax_rate": "0.0725",
  "notes": "Nightly janitorial, Building A",
  "terms": "Net 30",
  "items": [
    {
      "description": "Nightly cleaning (Mon–Fri)",
      "quantity": "22",
      "unit_price": "95.00",
      "sort_order": 1
    },
    {
      "description": "Supplies",
      "quantity": "1",
      "unit_price": "120.00",
      "sort_order": 2
    }
  ]
}
```

Required: `customer_id`, `valid_until`, at least one item (`description`,
`quantity`, `unit_price`). `tax_rate` is a fraction; omitted on create, the
tenant's default tax rate applies; omitted on update, the quote keeps its rate.
The response is the quote with `quote_number`, `status: "draft"`, `subtotal`,
tax, total and `is_expired`.

## Send, Accept, Reject, Revise

```http
POST /api/v1/quotes/:id/send
POST /api/v1/quotes/:id/accept
POST /api/v1/quotes/:id/reject
POST /api/v1/quotes/:id/revise
```

No request body. An invalid transition (e.g. accepting a draft) returns
**409** with the standard error envelope ([errors.md](./errors.md)); an
unknown or other-tenant quote is 404.

## Invoice an Accepted Quote

```http
POST /api/v1/invoices
```

```json
{ "customer_id": "uuid", "quote_id": "uuid", "lines": [ ... ] }
```

Only an accepted quote can be invoiced, and only once. See
[invoices.md](./invoices.md).

## Database

`quotes` and `quote_items` (see [DATABASE.md](../architecture/DATABASE.md)).
Quote numbers are per tenant (`tenant_counters`); totals are computed in the
service with `shopspring/decimal`.
