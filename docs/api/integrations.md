# Integrations API — QuickBooks Online (T192)

One-way sync of a tenant's customers, invoices and payments to its own
QuickBooks Online company. Setup at Intuit and configuration:
[docs/ops/quickbooks.md](../ops/quickbooks.md). Swagger UI (`/api/docs`, tag
_Integrations_) is the source of truth for shapes.

Every route under `/tenants/:id/integrations/quickbooks` needs
**`integrations.manage`** in that tenant (owners and admins: the built-in
`super_admin` and `admin` roles; granted by migration 000023). Without Intuit
keys on the server every response says `configured: false` and connect answers
503 `quickbooks_not_configured`.

## Endpoints

| Method | Endpoint                                                        | Description                                                    |
| ------ | --------------------------------------------------------------- | -------------------------------------------------------------- |
| GET    | `/tenants/:id/integrations/quickbooks`                          | Status, mapping, counts, failed records, recent activity       |
| POST   | `/tenants/:id/integrations/quickbooks/connect`                  | Intuit consent URL (single use, 10 min)                        |
| GET    | `/integrations/quickbooks/callback`                             | Intuit's redirect; no session; redirects back to Settings      |
| POST   | `/tenants/:id/integrations/quickbooks/disconnect`               | Revoke at Intuit and forget the tokens                         |
| GET    | `/tenants/:id/integrations/quickbooks/options`                  | Income accounts, tax codes, deposit accounts of the company    |
| PUT    | `/tenants/:id/integrations/quickbooks/settings`                 | Save the mapping; failed records are retried                   |
| POST   | `/tenants/:id/integrations/quickbooks/sync`                     | Sync now (up to 25 s; the rest goes with the hourly job)       |
| POST   | `/tenants/:id/integrations/quickbooks/records/:record_id/retry` | Re-queue one failed record                                     |
| POST   | `/internal/jobs/quickbooks-sync` (server root)                  | Hourly job for all connected tenants (`X-Internal-Jobs-Token`) |

## Connect flow

1. `POST …/connect` → `{ "url": "https://appcenter.intuit.com/connect/oauth2?…" }`.
   The SPA navigates there. The `state` is HMAC-signed (key derived from
   `INTEGRATIONS_ENCRYPTION_KEY`) and names the tenant, the user and a nonce;
   only the nonce's SHA-256 is stored on the connection row, with a 10 minute
   expiry.
2. Intuit redirects to `/api/v1/integrations/quickbooks/callback?code&state&realmId`.
   The callback needs no session (Neon Auth bearer tokens aren't sent on a
   top-level navigation): it verifies the signature and expiry, consumes the
   nonce atomically (a replayed state fails), re-checks that the user still
   has `integrations.manage` in the tenant, exchanges the code and stores the
   tokens sealed with AES-256-GCM (bound to the tenant).
3. The browser lands on
   `/settings/organization?tab=integrations&quickbooks=connected`, or
   `…&quickbooks=error&reason=cancelled|expired|unavailable|failed`.

## Status

```json
{
  "configured": true,
  "status": "connected",
  "environment": "sandbox",
  "company_name": "Brightline Facility Services LLC",
  "sync_from": "2026-11-04",
  "last_sync_at": "2026-11-04T15:15:02Z",
  "income_account": { "id": "79", "name": "Cleaning Services" },
  "item": { "id": "112", "name": "ServicePro - Cleaning Services" },
  "tax_code": { "id": "3", "name": "OH 8.0%" },
  "counts": { "customers": 23, "invoices": 118, "payments": 96, "pending": 0, "errors": 1 },
  "errors": [
    {
      "id": "…", "entity": "invoice", "label": "Invoice INV-9 · Acme · $108.00",
      "status": "error", "attempts": 2,
      "last_error": "choose an income account in Settings → Integrations → QuickBooks"
    }
  ],
  "recent": [ … ]
}
```

`status` is `connected`, `reauth_required` (the refresh token was rejected:
reconnect) or `disconnected`.

## What is synced

- **Customers** — created (or linked to an existing customer with the same
  DisplayName) when one of their invoices syncs; edits are pushed afterwards.
- **Invoices** — sent (not draft, not cancelled), issued on/after `sync_from`.
  Lines use the mapped item; line discounts are netted, the invoice discount
  becomes a discount line, and ServicePro's tax total is sent so the totals
  match. `DocNumber` is the invoice number (max 21 characters).
- **Payments** — every recorded payment (manual or Stripe) on a synced invoice,
  applied to that invoice (`LinkedTxn`), deposited to the mapped account or
  Undeposited Funds. Created once, never updated.

Queueing happens right after an invoice is sent or a payment recorded, and a
sweep in every sync pass catches the rest (edits, Stripe webhooks,
consolidated invoices). See [docs/ops/quickbooks.md](../ops/quickbooks.md) for
idempotency, retries and known limits.
