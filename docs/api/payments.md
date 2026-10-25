# Payments API

How money moves through ServicePro today. There is **no `/payments/*`
resource**: payments are recorded against invoices, card payments go through
Stripe Checkout from the invoice's pay link, and Stripe tells us they
succeeded through a webhook.

> **Money rules** (CLAUDE.md): amounts are `shopspring/decimal` strings, totals
> are computed server-side, payment state changes in one transaction, and card
> payment state is confirmed **only** by a signature-verified, deduplicated
> Stripe webhook — never by the browser redirect. Today charges go to the
> platform Stripe account; Stripe Connect (money straight to the tenant) is
> T169.

## Endpoints

| Method | Endpoint                                      | Auth            | Description                                                   |
| ------ | --------------------------------------------- | --------------- | ------------------------------------------------------------- |
| POST   | `/api/v1/invoices/:id/send`                   | user            | Email the invoice with a fresh pay link (token)               |
| GET    | `/api/v1/public/invoices/validate/:token`     | public          | Is the pay-link token valid?                                  |
| GET    | `/api/v1/public/invoices/details/:token`      | public          | Invoice details for the pay page                              |
| GET    | `/api/v1/public/invoices/pay/:token`          | public          | Create a Stripe Checkout session and redirect (303) to it     |
| POST   | `/api/v1/stripe/webhooks`                     | Stripe signature| Stripe events (payments, refunds, subscriptions)              |
| POST   | `/api/v1/invoices/:id/payments`               | user            | Record an offline payment (cash, check, bank transfer, …)     |
| POST   | `/api/v1/invoices/:id/refunds`                | user            | Record a refund made outside Stripe                           |
| GET    | `/api/v1/invoices/:id/receipt/pdf`            | user            | Payment receipt PDF                                           |

Tenant billing (the tenant paying for its ServicePro plan) is separate:

| Method | Endpoint                                                  | Description                         |
| ------ | --------------------------------------------------------- | ----------------------------------- |
| GET    | `/api/v1/billing/config`                                  | Is Stripe enabled? Publishable key  |
| POST   | `/api/v1/tenants/:id/billing/setup-intent`                | SetupIntent to add a card           |
| GET    | `/api/v1/tenants/:id/billing/payment-methods`             | List saved payment methods          |
| POST   | `/api/v1/tenants/:id/billing/payment-methods`             | Attach a payment method             |
| PUT    | `/api/v1/tenants/:id/billing/payment-methods/:pmId/default` | Make it the default               |
| DELETE | `/api/v1/tenants/:id/billing/payment-methods/:pmId`       | Remove it                           |
| GET    | `/api/v1/tenants/:id/billing/history`                     | Billing history                     |
| GET    | `/api/v1/membership-tiers`, `/api/v1/tenants/:id/membership*` | Plans and the tenant's plan     |

`:pmId` is looked up in our own tenant-scoped rows; raw Stripe IDs from the
client are never trusted.

## Card payment flow (customer pays an invoice)

1. A user sends the invoice: `POST /invoices/:id/send` generates a pay-link
   token and emails the customer.
2. The customer opens the link; the pay page uses `validate` / `details`.
3. `GET /public/invoices/pay/:token` creates a Stripe Checkout session (with an
   idempotency key) and redirects to Stripe.
4. Stripe calls `POST /api/v1/stripe/webhooks` with `checkout.session.completed`.
   The webhook is signature-verified and deduplicated by event ID
   (`stripe_webhook_events`); the payment is applied idempotently per session
   and the receipt email goes out.
5. Refunds issued in Stripe arrive as `charge.refunded` and are recorded
   automatically.

The browser's return from Checkout changes nothing server-side.

## Record an offline payment

```http
POST /api/v1/invoices/:id/payments
```

```json
{
  "amount": "1082.50",
  "payment_date": "2024-01-20T00:00:00Z",
  "payment_method": "check",
  "reference_number": "1042",
  "notes": "Mailed check"
}
```

`amount` is required and must be positive. Paying the balance in full marks
the invoice `paid`; less marks it `partially_paid`. See
[invoices.md](./invoices.md) for refunds and statuses.

## Configuration

Backend (`backend/.env.example`): `STRIPE_SECRET_KEY`, `STRIPE_PUBLISHABLE_KEY`
(served by `GET /billing/config`), `STRIPE_WEBHOOK_SECRET`; the web build also
takes `VITE_STRIPE_PUBLISHABLE_KEY`.
Locally, `docker-compose.yml` has a `stripe-cli` service that forwards webhooks
to the API. Production webhook URL:
`https://servicepro.sharkfins.xyz/api/v1/stripe/webhooks`.

## Testing

Use Stripe test mode keys (`sk_test_…`) and Stripe's test cards (e.g.
`4242 4242 4242 4242`, any future expiry, any CVC).
