# Stripe Connect — Accounts v2, direct charges (T169)

Decisions: D9 (Accounts v2, direct charges, Stripe handles pricing), D10 (cards +0.4%, ACH 1% capped at $10),
D11 (payments off until money reaches the tenant). SDK: `github.com/stripe/stripe-go/v87` v87.1.0
(API version `2026-10-28.endive`). Sources: `stripe docs` pages `/connect/saas/tasks/create`,
`/connect/saas/tasks/onboard`, `/connect/direct-charges?platform=web&ui=stripe-hosted`,
`/connect/saas/tasks/app-fees`, `/connect/platform-pricing-tools` (fetched 2026-11-02).

## Spike (2026-11-02, test mode)

Run it with `STRIPE_CONNECT_SPIKE=1 go test ./internal/services/stripe/ -run TestConnectSpike -v`
(`backend/internal/services/stripe/connect_spike_test.go`; it refuses anything but a `sk_test_`/`rk_test_` key and
reads `STRIPE_SECRET_KEY` from the environment or `backend/.env`).

### Results

| Step | Result |
| --- | --- |
| stripe-go v87 has Accounts v2 | Yes: `client.V2CoreAccounts` (Create/Retrieve/Update/Close/List), `client.V2CoreAccountLinks`, v2 thin-event parsing (`ParseEventNotification`). Requests are built and sent correctly. |
| `POST /v2/core/accounts` | **Blocked**: `400 non_connect_platform_accounts_v2_access_blocked` — "Accounts v2 is not enabled for your sandbox merchant … onboard onto Connect at Settings → Connect → Platform setup". Connect has never been set up on this Stripe account. |
| `POST /v2/core/account_links` | Not reached (needs an account). |
| Direct charge with `application_fee_amount` | Not reached (needs an onboarded account). |
| Bank-only Checkout (`allowed_payment_method_types=[us_bank_account]`) | **Works** (on the platform): the session offers exactly `[us_bank_account]`. |
| Card Checkout (`excluded_payment_method_types=[us_bank_account]`) | **Works**: offers `[card klarna link affirm cashapp amazon_pay]` (dynamic payment methods, ACH removed). |

Once Rob finishes the Connect platform setup (below), re-run the spike: it then creates the v2 account, an onboarding
link, a bank-only Checkout Session on the connected account with a $10 application fee on $5,000, and a card
PaymentIntent with a 0.4% fee.

### What needs Rob (Dashboard, not code)

1. **Enable Connect on the platform** (test mode first): Dashboard → Settings → Connect → Platform setup
   (`/settings/connect/platform-setup`). Choose "SaaS platform" / connected accounts pay Stripe directly — this is the
   only thing blocking the spike and the whole feature.
2. **Platform profile and branding** (Settings → Connect → Branding): name, icon, colour shown in Stripe-hosted
   onboarding. Required before live onboarding.
3. **Connect webhook endpoint** (Developers → Webhooks → "Events on connected accounts") at
   `https://<api>/api/v1/webhooks/stripe/connect` for `checkout.session.completed`,
   `checkout.session.async_payment_succeeded`, `charge.refunded`, `account.updated`; its signing secret goes in
   `STRIPE_CONNECT_WEBHOOK_SECRET`.
4. **Event destination for v2 events** (Developers → Event destinations, "thin" payload) at
   `https://<api>/api/v1/webhooks/stripe/v2` for `v2.core.account[requirements].updated` and
   `v2.core.account[configuration.merchant].capability_status_updated`; secret → `STRIPE_V2_WEBHOOK_SECRET`.
5. Optional: **Public details** (Settings → Connect → Onboarding options) to collect statement descriptor and support
   phone during onboarding.
6. Locally, `stripe listen --forward-connect-to localhost:8080/api/v1/webhooks/stripe/connect` and
   `stripe listen --forward-thin-to localhost:8080/api/v1/webhooks/stripe/v2 --thin-events 'v2.core.account[*]'`
   replace items 3–4.

## Design (what the code does)

### Connected account (one per tenant)

`POST /v2/core/accounts` with:

```json
{
  "display_name": "<tenant name>", "contact_email": "<owner email>", "dashboard": "full",
  "identity": {"country": "us", "entity_type": "company"},
  "configuration": {
    "merchant": {"capabilities": {"card_payments": {"requested": true}, "ach_debit_payments": {"requested": true}}},
    "customer": {}
  },
  "defaults": {"currency": "usd", "responsibilities": {"fees_collector": "stripe", "losses_collector": "stripe"},
               "locales": ["en-US"]},
  "include": ["configuration.merchant", "requirements"],
  "metadata": {"tenant_id": "<uuid>"}
}
```

- `fees_collector: stripe` + `losses_collector: stripe` + `dashboard: full` is Stripe's SaaS default (D9: Stripe bills
  the tenant its processing fees; ServicePro pays no Connect fees; Stripe owns negative balances). Refunds, disputes
  and payouts happen in the tenant's own Stripe Dashboard.
- The customer configuration is there so platform billing (T210) can later charge the same v2 account instead of a
  separate v1 Customer.
- Idempotency key `connect-account:<tenant_id>` — a double click can never make two accounts. The account ID is stored
  in `stripe_connected_accounts` (migration 000009; `tenant_id` unique, never accepted from the client).

### Onboarding

`POST /v2/core/account_links`, `use_case.type=account_onboarding`, `collection_options.fields=eventually_due`
(up-front: one visit, fewer surprise holds), `return_url`/`refresh_url` = the SPA's Settings → Payments page. Links
are single-use and short-lived, so the API creates a fresh one on every click (idempotency key per tenant per minute).

### Readiness

The v1 `charges_enabled` flag is not used. Status comes from the v2 account:

- `configuration.merchant.capabilities.card_payments.status` (`active` = can take cards)
- `configuration.merchant.capabilities.ach_debit_payments.status` (`active` = can take bank payments)
- `configuration.merchant.capabilities.stripe_balance.payouts.status`
- `requirements.summary.minimum_deadline.status` (`currently_due` / `past_due` → "finish onboarding")

Synced (re-fetched from Stripe, never trusted from a payload) on: the v2 thin events above, `account.updated` on the
Connect endpoint, and `GET /tenants/:id/payments/status` while onboarding isn't finished (the user just came back from
Stripe).

### Invoice payment (direct charge)

The pay link is `GET /api/v1/public/invoices/pay/:token?method=bank|card`. Pay links appear only when
`INVOICE_PAYMENTS_ENABLED=true` (T261 master switch) **and** the invoice's tenant has `card_payments` active.

Checkout Session created **on the connected account** (`Stripe-Account: acct_…`, `params.SetStripeAccount`):

| Method | Session restriction | `payment_intent_data.application_fee_amount` |
| --- | --- | --- |
| bank | `allowed_payment_method_types=[us_bank_account]` (needs `ach_debit_payments` active) | 1% of the balance, capped at $10 (1000 cents) |
| card (default when ACH isn't active) | `excluded_payment_method_types=[us_bank_account]` | 0.4% of the balance |

The fee has to be fixed when the session is created, which is why the customer picks bank or card first (mockup 17);
the Platform Pricing Tool can't do it for us — it is only for platforms that pay Stripe's fees. Fees are integer cents
computed server-side from the invoice's decimal balance (round half up), never 0 and always below the charge amount.
Idempotency key: `invoice-checkout:<invoice>:<account>:<method>:<currency>:<units>:<hour>`.

### Webhooks

| Endpoint | Secret | Events |
| --- | --- | --- |
| `/webhooks/stripe` | `STRIPE_WEBHOOK_SECRET` | platform events (subscriptions, T122) |
| `/webhooks/stripe/connect` | `STRIPE_CONNECT_WEBHOOK_SECRET` | snapshot events from connected accounts (`event.account` set) |
| `/webhooks/stripe/v2` | `STRIPE_V2_WEBHOOK_SECRET` | v2 thin events (`v2.core.account[...]`) |

All three verify the signature and claim the event ID in `stripe_webhook_events` before processing (exactly-once).
The invoice handlers only apply a connected-account payment when `event.account` equals the stored account of the
invoice's tenant; anything else is logged and ignored. Payments are recorded through the existing
`MarkInvoiceAsPaid` ledger path (deduplicated per Checkout Session). ACH completes later:
`checkout.session.completed` arrives `unpaid`, then `checkout.session.async_payment_succeeded` records it (the
pending-state UI is T170).
