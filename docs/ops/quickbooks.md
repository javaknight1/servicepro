# QuickBooks Online setup (T192)

ServicePro pushes each tenant's customers, invoices and payments one way into
the tenant's own QuickBooks Online company. The code is built and tested
against an in-memory fake QuickBooks. It stays **off** until the Intuit app
below exists and its keys are set: Settings → Integrations and the onboarding
wizard show QuickBooks as "Coming soon", and the hourly job does nothing.

How it works: [docs/api/integrations.md](../api/integrations.md).

## What Rob does at Intuit (one time)

1. **Developer account.** Sign up at <https://developer.intuit.com> with the
   ServicePro business Google account. Save the login in Bitwarden
   (`Intuit Developer`, folder _Dev & Cloud_) and verify the email address
   (production keys need a verified profile).
2. **Create the app.** Dashboard → _Create an app_ → **QuickBooks Online and
   Payments** → name it `ServicePro`. Scope: **`com.intuit.quickbooks.accounting`**
   only (no Payments, no OpenID scopes).
3. **Development redirect URIs.** App → _Development settings_ → _Keys &
   credentials_ → _Redirect URIs_, add both (exact, no trailing slash):
   - `https://servicepro.sharkfins.xyz/api/v1/integrations/quickbooks/callback`
   - `http://localhost:5173/api/v1/integrations/quickbooks/callback`
     (Intuit allows `http://localhost` for development keys only.)
4. **Sandbox company.** Dashboard → _Sandbox_: a US sandbox company
   (`Sandbox Company_US_1`) is created with the account; add one if it
   isn't there. Open it once and, in _Settings → Account and settings →
   Sales_, note whether automated sales tax is on (affects tax, see below).
5. **Development keys → Bitwarden.** Copy the development Client ID and Client
   Secret into a Bitwarden item `Intuit QuickBooks (ServicePro)` (custom
   hidden fields `client_id_dev`, `client_secret_dev`). Never paste them in
   chat or commit them.
6. **Try it locally** (see below), then on production with the sandbox
   company (`QUICKBOOKS_ENVIRONMENT=sandbox`).
7. **Production keys.** App → _Production settings_ → _Keys & credentials_.
   Intuit requires, before it shows production keys:
   - the app details: host domain `servicepro.sharkfins.xyz`, launch URL
     `https://servicepro.sharkfins.xyz/settings/organization?tab=integrations`,
     disconnect URL (same page), end-user licence agreement and privacy policy
     URLs (the Terms and Privacy pages, T207), app category;
   - production redirect URI
     `https://servicepro.sharkfins.xyz/api/v1/integrations/quickbooks/callback`
     (HTTPS only; no localhost);
   - the **app assessment questionnaire** (security and data-handling:
     how tokens are stored — AES-256-GCM at rest, keys in Secret Manager —,
     regulated industries, where the app is hosted — Cloud Run us-east5 and
     Neon us-east-2 —, how users disconnect). Intuit reviews it; expect
     days to a few weeks and possible follow-up questions. Start this early:
     it is the long pole of T192.
     Save the production Client ID / Secret in the same Bitwarden item
     (`client_id_prod`, `client_secret_prod`).
8. **App Store listing** is optional (only for a public listing on
   apps.intuit.com) and not needed for tenants to connect.

## Configuration

Backend env vars (`backend/.env.example` has the comments):

| Variable                      | Value                                                                    |
| ----------------------------- | ------------------------------------------------------------------------ |
| `QUICKBOOKS_CLIENT_ID`        | Intuit Client ID (secret store in production)                            |
| `QUICKBOOKS_CLIENT_SECRET`    | Intuit Client Secret (secret)                                            |
| `QUICKBOOKS_ENVIRONMENT`      | `sandbox` (development keys) or `production` (production keys)           |
| `QUICKBOOKS_REDIRECT_URI`     | Optional; default `FRONTEND_URL/api/v1/integrations/quickbooks/callback` |
| `INTEGRATIONS_ENCRYPTION_KEY` | 32 random bytes, base64 (`openssl rand -base64 32`); encrypts tokens     |
| `QUICKBOOKS_FAKE`             | Development only: in-memory fake QuickBooks, no Intuit keys needed       |

All are optional: with the keys unset (or, outside development, without a
valid `INTEGRATIONS_ENCRYPTION_KEY`) the integration is simply off and the
server starts normally. Keep the encryption key stable: changing it makes the
stored tokens unreadable and every tenant has to reconnect. Generate it once
into Bitwarden and Secret Manager; never log or commit it.

Production: add `QUICKBOOKS_CLIENT_ID`, `QUICKBOOKS_CLIENT_SECRET` and
`INTEGRATIONS_ENCRYPTION_KEY` as Secret Manager secrets mapped into the Cloud
Run service (as the other secrets in `scripts/deploy-cloudrun.sh` /
`docs/ops/cloud-run-setup.md`), and `QUICKBOOKS_ENVIRONMENT` as a plain env var.
`FRONTEND_URL` must be `https://servicepro.sharkfins.xyz` so the default
redirect URI matches the one registered at Intuit.

### Local

- Without Intuit: `QUICKBOOKS_FAKE=true` in `backend/.env` (development
  only). Connect goes straight back to the callback with a fake company;
  sync writes to an in-memory store (lost on restart).
- With the development keys: set the two keys,
  `QUICKBOOKS_ENVIRONMENT=sandbox`, `FRONTEND_URL=http://localhost:5173`, and
  an `INTEGRATIONS_ENCRYPTION_KEY`; connect the sandbox company from Settings
  → Integrations.

## Operations

- **Sync job:** `POST /internal/jobs/quickbooks-sync`, hourly at :15 from
  `.github/workflows/scheduled-jobs.yml` (Sentry cron monitor
  `servicepro-quickbooks-sync`). Each call works for up to 40 s and reports
  `more: true` when work is left. Tenants also have **Sync now**.
- **Idempotency:** one worker per tenant at a time (a 90 s lease on
  `quickbooks_connections.sync_locked_until`); every record is mapped once in
  `quickbooks_sync_map`; creates carry an Intuit `requestid` that is reused
  on retry, and a retried invoice/customer create looks the record up first,
  so a lost response never creates a duplicate.
- **Failures:** per record, with exponential backoff (1 min doubling, max
  12 h) and at most 8 automatic attempts; listed in Settings → Integrations
  with **Retry**. Throttling (HTTP 429) and Intuit outages stay pending
  rather than erroring.
- **Authorization:** access tokens last 1 hour and are refreshed under a row
  lock (refresh tokens rotate); refresh tokens last ~100 days from the last
  refresh, so any tenant that syncs at least monthly never expires. A
  rejected refresh token marks the connection `reauth_required` and the tab
  asks the owner to reconnect.
- **Rate limits:** QuickBooks allows 500 requests/minute per company; one
  pass handles at most 150 records per tenant.

## Known limits (one-way sync, first version)

- Only invoices **issued on or after the connection day** are pushed (with
  their customer and payments); older ones are assumed to be in QuickBooks
  already. A different company on reconnect starts over from that day.
- **Sales tax:** lines are marked `TAX`/`NON` and the invoice carries
  ServicePro's tax total (`TxnTaxDetail.TotalTax`), optionally with the chosen
  tax code. Companies with QuickBooks **automated sales tax** may recalculate
  tax — verify on the sandbox before the first design partner and adjust
  (T192 follow-up) if totals differ.
- Not synced yet: refunds, cancelled invoices (no void in QuickBooks),
  deleted records, Stripe payouts/fees as bank deposits (mockup 14's
  "Processing fees" mapping), and anything from QuickBooks back to
  ServicePro (two-way, webhooks/CDC).
- An accountant's edit to a synced invoice in QuickBooks is overwritten by the
  next change in ServicePro (ServicePro is the source of truth).
