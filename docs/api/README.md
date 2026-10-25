# ServicePro API Reference

REST API for the ServicePro platform. **Source of truth:** the generated
OpenAPI spec in `backend/docs/` (Swagger UI at `/api/docs` locally) and the
route files in `backend/internal/api/routes/`. These pages summarise it.

## Base URL

| Environment | Base URL                                                                                        |
| ----------- | ----------------------------------------------------------------------------------------------- |
| Local       | `http://localhost:8080/api/v1`                                                                  |
| Production  | `https://servicepro.sharkfins.xyz/api/v1` (proxied by the `servicepro-web` Worker to Cloud Run) |

In production the API only accepts `/api/*` traffic that came through the
Worker; the raw `*.run.app` URL answers 403 for `/api/*`.

## Authentication

The SPA authenticates with httpOnly cookies set by `POST /auth/login`; mutating
cookie requests must send `X-Requested-With: XMLHttpRequest`. Scripts can send
`Authorization: Bearer <token>` instead (an API access token, or a Neon Auth JWT
for a linked user). Tenant-scoped routes use the caller's membership;
`X-Tenant-ID` selects among the caller's tenants. Details:
[authentication.md](./authentication.md).

## Endpoints Overview

All paths below are relative to `/api/v1`.

### Auth and users

| Method      | Endpoint                                                         | Description               |
| ----------- | ---------------------------------------------------------------- | ------------------------- |
| POST        | `/auth/register`, `/auth/login`, `/auth/refresh`, `/auth/logout` | Account and session       |
| POST        | `/auth/reset-request`, `/auth/reset-password`                    | Password reset            |
| GET         | `/auth/verify?token=…`                                           | Verify email              |
| POST        | `/auth/resend-verification`                                      | Resend verification email |
| POST        | `/auth/neon/provision`                                           | Link a Neon Auth sign-in  |
| GET         | `/users/me`                                                      | Current user              |
| PATCH       | `/users/me/profile`                                              | Update profile            |
| POST/DELETE | `/users/me/profile-picture`                                      | Profile picture           |

See [authentication.md](./authentication.md).

### Tenants, members, roles

| Method         | Endpoint                                           | Description                         |
| -------------- | -------------------------------------------------- | ----------------------------------- |
| GET/POST       | `/tenants`                                         | List my tenants / create a tenant   |
| GET/PUT/DELETE | `/tenants/:id`                                     | Tenant                              |
| GET            | `/tenants/:id/me`                                  | My membership in the tenant         |
| GET/POST       | `/tenants/:id/members`                             | Members                             |
| DELETE         | `/tenants/:id/members/:userId`                     | Remove member                       |
| PUT            | `/tenants/:id/members/:userId/role`                | Change a member's role              |
| GET/POST       | `/tenants/:id/invitations`                         | Invitations                         |
| DELETE         | `/tenants/:id/invitations/:invitation_id`          | Cancel invitation                   |
| POST           | `/tenants/:id/invitations/:invitation_id/resend`   | Resend invitation                   |
| GET            | `/invitations/:token`                              | Look up an invitation               |
| POST           | `/invitations/:token/accept`                       | Accept an invitation                |
| GET            | `/roles`, `/roles/:id/permissions`, `/permissions` | Roles and permissions               |
| POST/GET       | `/admin/invite-codes`                              | Invite codes (platform super admin) |
| DELETE         | `/admin/invite-codes/:id`                          | Revoke an invite code               |

### Customers

`/customers` CRUD, `/customers/search`, `/advanced-search`, `/autocomplete`,
`/stats`, `/email/:email`, `/:id/jobs`, `/:id/quotes`, CSV import. See
[customers.md](./customers.md).

### Jobs and scheduling

`/jobs` CRUD, `/jobs/my`, `/scheduled`, `/overdue`, `/stats`, status actions
(`start`, `complete`, `cancel`, `hold`, `resume`, `transition`), assignments,
materials, notes, and `POST /conflicts/check`. See [jobs.md](./jobs.md).

### Quotes

`/quotes` CRUD, `send`, `accept`, `reject`, `revise`, `pdf`, `stats`,
`number/:quoteNumber`. See [quotes.md](./quotes.md).

### Invoices and payments

`/invoices` CRUD, `send`, `payments`, `refunds`, `cancel`, `pdf`,
`receipt/pdf`, `reminders`; `/settings/payment-reminders`; public pay links
under `/public/invoices/*`; `POST /stripe/webhooks`. See
[invoices.md](./invoices.md) and [payments.md](./payments.md).

### Payment terms and tax rates

| Method          | Endpoint                                                               | Description          |
| --------------- | ---------------------------------------------------------------------- | -------------------- |
| GET             | `/payment-terms`, `/payment-terms/default`, `/payment-terms/:id`       | Payment terms        |
| POST/PUT/DELETE | `/payment-terms`, `/payment-terms/:id`                                 | Manage payment terms |
| POST            | `/payment-terms/calculate/{due-date,discount,late-fee,status,details}` | Term calculations    |
| GET/POST        | `/tax-rates`                                                           | Tax rates            |
| GET/PUT/DELETE  | `/tax-rates/:id`                                                       | Tax rate             |

See [tax rates](../features/tax-service.md).

### Reports and exports

| Method   | Endpoint                                                                                                       | Description           |
| -------- | -------------------------------------------------------------------------------------------------------------- | --------------------- |
| GET      | `/reports/revenue`, `/revenue/time-series`, `/revenue/categories`, `/revenue/customers`, `/revenue/chart-data` | Revenue               |
| GET      | `/reports/customers`, `/customers/{segments,types,geography,acquisition,top,at-risk,cohorts,chart-data}`       | Customers             |
| GET      | `/reports/ar-aging`, `/ar-aging/buckets`, `/ar-aging/buckets/:bucket/invoices`, `/ar-aging/customers`          | AR aging              |
| POST     | `/reports/{revenue,customers,ar-aging}/export`                                                                 | Report exports        |
| POST     | `/reports/{revenue,customers}/cache/refresh`                                                                   | Refresh report caches |
| GET/POST | `/exports`, `/exports/quick`, `/exports/:id{,/progress,/stream,/download,/cancel}`, `/exports/formats`         | Export jobs           |

See [import-export.md](../features/import-export.md).

### Billing and membership (the tenant's own plan)

`/billing/config`, `/tenants/:id/billing/{setup-intent,payment-methods,history}`,
`/membership-tiers`, `/tenants/:id/membership{,/history,/preview}`. See
[payments.md](./payments.md).

### Outside `/api/v1`

| Method | Endpoint                                                          | Description                                                         |
| ------ | ----------------------------------------------------------------- | ------------------------------------------------------------------- |
| GET    | `/version`                                                        | Build identity (`git_sha`, `app_version`)                           |
| GET    | `/health`, `/health/live`, `/health/ready`                        | Health; `ready` is 503 when the DB schema is behind/dirty           |
| POST   | `/internal/jobs/payment-reminders`, `/internal/jobs/quote-expiry` | Scheduled jobs (`X-Internal-Jobs-Token`); not proxied by the Worker |
| GET    | `/metrics`                                                        | Prometheus (bearer token outside development)                       |
| GET    | `/api/docs/*`                                                     | Swagger UI                                                          |

## Request/Response Format

JSON in and out (`Content-Type: application/json`), except file uploads
(multipart) and downloads (CSV/PDF/XLSX).

Single resources are returned as the object itself. Lists return a named array
with paging fields, e.g.:

```json
{ "invoices": [ ... ], "total": 100, "page": 1, "page_size": 20, "total_pages": 5 }
```

### Error Responses

```json
{
  "error": "validation_failed",
  "message": "Validation failed: email must be a valid email address",
  "request_id": "3f6c1b9e-2a4d-4e7b-9a51-0c8d2f1e6b7a",
  "fields": { "email": "email must be a valid email address" }
}
```

Every error uses this envelope; see [errors.md](errors.md) for the contract, the status-code
rules and the common codes. Every response carries an `X-Request-ID` header.

## HTTP Status Codes

| Code | Status                | Description                                          |
| ---- | --------------------- | ---------------------------------------------------- |
| 200  | OK                    | Request successful                                   |
| 201  | Created               | Resource created                                     |
| 204  | No Content            | Request successful, no content                       |
| 400  | Bad Request           | Invalid request data                                 |
| 401  | Unauthorized          | Authentication required                              |
| 403  | Forbidden             | Insufficient permissions, CSRF header missing        |
| 404  | Not Found             | Resource not found — including another tenant's rows |
| 409  | Conflict              | Duplicate, or an invalid state transition            |
| 410  | Gone                  | Deprecated endpoint (`GET /customers/export`)        |
| 429  | Too Many Requests     | Rate limit exceeded                                  |
| 500  | Internal Server Error | Server error (logged; generic message)               |

## Rate Limiting

Defaults per window of `RATE_LIMIT_WINDOW` (1 minute), configurable in
`backend/.env.example`:

| Caller          | Limit (`RATE_LIMIT_*`)          |
| --------------- | ------------------------------- |
| Anonymous       | 100 requests (`ANONYMOUS`)      |
| Authenticated   | 1000 requests (`AUTHENTICATED`) |
| Admin endpoints | 50 requests (`ADMIN`)           |

Login and password-reset requests have their own stricter throttles.
`/health` and `/version` are excluded. Responses carry:

```
X-RateLimit-Limit: 1000
X-RateLimit-Remaining: 995
X-RateLimit-Reset: 1234567890
Retry-After: 45   (when limited)
```

With Redis configured the counters are shared across instances; without it
they're per instance.

## Filtering and Pagination

```
GET /api/v1/invoices?status=sent&customer_id=<uuid>&page=1&page_size=20&sort_by=created_at&sort_order=desc
```

| Parameter  | Type    | Description                  |
| ---------- | ------- | ---------------------------- |
| page       | integer | Page number (default: 1)     |
| page_size  | integer | Items per page (default: 20) |
| sort_by    | string  | Field to sort by             |
| sort_order | string  | `asc` or `desc`              |

Filters differ per resource — check the endpoint page or Swagger.

## Common Data Types

- **IDs:** UUIDs.
- **Timestamps:** ISO 8601 (`2026-01-15T10:30:00Z`); some scheduling fields
  are integer epoch timestamps (see [jobs.md](./jobs.md)).
- **Money:** decimal strings with 2 places (`"1500.00"`), computed server-side
  with `shopspring/decimal`. Tax rates are fractions (`"0.0825"` = 8.25%).

## Testing the API

```bash
curl http://localhost:8080/health

curl -X POST http://localhost:8080/api/v1/customers \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{"first_name":"Jane","last_name":"Doe","email":"jane@example.com", ...}'
```

Log in first (`make seed` gives `dev@servicepro.local` / `password123`), or use
Swagger UI at http://localhost:8080/api/docs.

## API Versioning

The API is versioned via URL path (`/api/v1/`). Breaking changes will increment the version number.

## Not built

There are no `/payments/*`, `/templates/*` or `/schedules` endpoints, no
`/quotes/:id/convert` and no WebSocket API. Document templates are planned
(quote templates T060, terms-and-conditions templates T080).
