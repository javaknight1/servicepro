# Stripe (Payment Processing)

## Overview

**Stripe** is a payment processing platform that handles credit card payments, subscriptions, invoicing, and more. It's the industry standard for SaaS billing.

### Why We Use Stripe

| Feature                | Benefit                    |
| ---------------------- | -------------------------- |
| **No Monthly Fee**     | Only pay per transaction   |
| **Subscriptions**      | Built-in recurring billing |
| **Webhooks**           | Real-time payment events   |
| **Developer-Friendly** | Excellent API and docs     |
| **PCI Compliant**      | Handles card data securely |
| **Global**             | 135+ currencies            |

### How ServicePro Uses Stripe

- **Subscriptions**: Monthly/yearly SaaS plans
- **Payment Methods**: Store customer cards securely
- **Invoicing**: Automatic invoice generation
- **Webhooks**: Handle payment events
- **Customer Portal**: Self-service billing management

---

## Pricing

| Fee Type            | Amount       |
| ------------------- | ------------ |
| Per transaction     | 2.9% + $0.30 |
| International cards | +1.5%        |
| Currency conversion | +1%          |
| Monthly fee         | $0           |
| Setup fee           | $0           |

**Example**: $100 charge = $3.20 fee (domestic card)

---

## Setup

### Option A: Web Browser Setup (Required First)

1. **Create Account**
   - Go to [stripe.com](https://stripe.com)
   - Click "Start now"
   - Enter email, full name, password
   - Verify your email

2. **Complete Business Profile**
   - Business type (individual, company, etc.)
   - Business details (name, address, tax ID)
   - Bank account for payouts
   - Identity verification

3. **Get API Keys**
   - Go to [Developers > API Keys](https://dashboard.stripe.com/apikeys)
   - You'll see:
     - **Publishable key** (`pk_test_...` or `pk_live_...`)
     - **Secret key** (`sk_test_...` or `sk_live_...`)
   - Use test keys for development, live keys for production

4. **Create Products & Prices**
   - Go to [Products](https://dashboard.stripe.com/products)
   - Click "Add product"
   - Create subscription tiers:
     - **Free** ($0/month)
     - **Basic** ($29/month, $290/year)
     - **Pro** ($99/month, $990/year)
   - Copy Price IDs (`price_xxx`)

5. **Set Up Webhooks**
   - Go to [Developers > Webhooks](https://dashboard.stripe.com/webhooks)
   - Click "Add endpoint"
   - URL: `https://servicepro-api.fly.dev/api/v1/webhooks/stripe`
   - Events to listen for:
     - `customer.subscription.created`
     - `customer.subscription.updated`
     - `customer.subscription.deleted`
     - `invoice.paid`
     - `invoice.payment_failed`
     - `payment_intent.succeeded`
   - Copy webhook signing secret (`whsec_xxx`)

### Option B: CLI Setup (Stripe CLI)

```bash
# Install Stripe CLI
# macOS
brew install stripe/stripe-cli/stripe

# Windows (scoop)
scoop install stripe

# Login
stripe login

# List products
stripe products list

# Create product
stripe products create \
  --name="Pro Plan" \
  --description="Full access to all features"

# Create price
stripe prices create \
  --product=prod_xxx \
  --unit-amount=9900 \
  --currency=usd \
  --recurring-interval=month

# Listen to webhooks (local development)
stripe listen --forward-to localhost:8080/api/v1/webhooks/stripe

# Trigger test events
stripe trigger payment_intent.succeeded
```

---

## Configuration

### Environment Variables

```bash
# API Keys (use test keys for development)
STRIPE_SECRET_KEY=sk_test_xxx          # Server-side only
STRIPE_PUBLISHABLE_KEY=pk_test_xxx     # Can be exposed to frontend
STRIPE_WEBHOOK_SECRET=whsec_xxx        # Webhook signature verification

# Price IDs for subscription tiers
STRIPE_PRICE_FREE_MONTHLY=price_xxx
STRIPE_PRICE_BASIC_MONTHLY=price_xxx
STRIPE_PRICE_BASIC_YEARLY=price_xxx
STRIPE_PRICE_PRO_MONTHLY=price_xxx
STRIPE_PRICE_PRO_YEARLY=price_xxx

# Feature flag
STRIPE_ENABLED=true
```

### Setting in Fly.io

```bash
# Set production keys
fly secrets set STRIPE_SECRET_KEY="sk_live_xxx" --app servicepro-api
fly secrets set STRIPE_WEBHOOK_SECRET="whsec_xxx" --app servicepro-api
fly secrets set STRIPE_PRICE_BASIC_MONTHLY="price_xxx" --app servicepro-api
fly secrets set STRIPE_PRICE_BASIC_YEARLY="price_xxx" --app servicepro-api
fly secrets set STRIPE_PRICE_PRO_MONTHLY="price_xxx" --app servicepro-api
fly secrets set STRIPE_PRICE_PRO_YEARLY="price_xxx" --app servicepro-api
```

### Test vs Live Mode

| Environment | Key Prefix             | Card Numbers   |
| ----------- | ---------------------- | -------------- |
| Test        | `sk_test_`, `pk_test_` | Use test cards |
| Live        | `sk_live_`, `pk_live_` | Real cards     |

**Test Card Numbers**:

- Success: `4242 4242 4242 4242`
- Decline: `4000 0000 0000 0002`
- Requires auth: `4000 0025 0000 3155`
- Any future expiry, any CVC

---

## Common Operations

### Using Stripe CLI

```bash
# List customers
stripe customers list

# Create customer
stripe customers create --email="customer@example.com" --name="John Doe"

# List subscriptions
stripe subscriptions list

# Cancel subscription
stripe subscriptions cancel sub_xxx

# List invoices
stripe invoices list

# Retrieve payment intent
stripe payment_intents retrieve pi_xxx
```

### API Examples (cURL)

```bash
# List customers
curl https://api.stripe.com/v1/customers \
  -u $STRIPE_SECRET_KEY:

# Create customer
curl https://api.stripe.com/v1/customers \
  -u $STRIPE_SECRET_KEY: \
  -d email="customer@example.com" \
  -d name="John Doe"

# Create subscription
curl https://api.stripe.com/v1/subscriptions \
  -u $STRIPE_SECRET_KEY: \
  -d customer=cus_xxx \
  -d "items[0][price]"=price_xxx

# Cancel subscription
curl https://api.stripe.com/v1/subscriptions/sub_xxx \
  -u $STRIPE_SECRET_KEY: \
  -X DELETE
```

### Go Integration

```go
import "github.com/stripe/stripe-go/v76"
import "github.com/stripe/stripe-go/v76/customer"
import "github.com/stripe/stripe-go/v76/subscription"

stripe.Key = os.Getenv("STRIPE_SECRET_KEY")

// Create customer
params := &stripe.CustomerParams{
    Email: stripe.String("customer@example.com"),
    Name:  stripe.String("John Doe"),
}
c, err := customer.New(params)

// Create subscription
subParams := &stripe.SubscriptionParams{
    Customer: stripe.String(c.ID),
    Items: []*stripe.SubscriptionItemsParams{
        {Price: stripe.String(os.Getenv("STRIPE_PRICE_PRO_MONTHLY"))},
    },
}
sub, err := subscription.New(subParams)
```

---

## Management

### Dashboard Overview

1. **[Home](https://dashboard.stripe.com/dashboard)**: Revenue overview
2. **[Payments](https://dashboard.stripe.com/payments)**: All transactions
3. **[Customers](https://dashboard.stripe.com/customers)**: Customer list
4. **[Subscriptions](https://dashboard.stripe.com/subscriptions)**: Active subscriptions
5. **[Invoices](https://dashboard.stripe.com/invoices)**: All invoices
6. **[Products](https://dashboard.stripe.com/products)**: Pricing plans

### View Customer Details

```bash
# Via CLI
stripe customers retrieve cus_xxx

# Via dashboard
# Go to Customers > Click customer
```

### Issue Refund

```bash
# Full refund
stripe refunds create --charge=ch_xxx

# Partial refund
stripe refunds create --charge=ch_xxx --amount=500  # $5.00
```

### Update Subscription

```bash
# Change plan
stripe subscriptions update sub_xxx \
  --items[0][id]=si_xxx \
  --items[0][price]=price_new_plan

# Cancel at period end
stripe subscriptions update sub_xxx --cancel-at-period-end=true

# Cancel immediately
stripe subscriptions cancel sub_xxx
```

---

## Webhooks

### Handling Webhooks

```go
import "github.com/stripe/stripe-go/v76/webhook"

func handleWebhook(w http.ResponseWriter, r *http.Request) {
    payload, err := io.ReadAll(r.Body)
    if err != nil {
        http.Error(w, "Error reading request body", http.StatusBadRequest)
        return
    }

    // Verify signature
    event, err := webhook.ConstructEvent(
        payload,
        r.Header.Get("Stripe-Signature"),
        os.Getenv("STRIPE_WEBHOOK_SECRET"),
    )
    if err != nil {
        http.Error(w, "Invalid signature", http.StatusBadRequest)
        return
    }

    // Handle event types
    switch event.Type {
    case "customer.subscription.created":
        // Handle new subscription
    case "customer.subscription.deleted":
        // Handle cancellation
    case "invoice.paid":
        // Handle successful payment
    case "invoice.payment_failed":
        // Handle failed payment
    }

    w.WriteHeader(http.StatusOK)
}
```

### Important Events

| Event                                  | When              | Action              |
| -------------------------------------- | ----------------- | ------------------- |
| `customer.subscription.created`        | New subscription  | Activate plan       |
| `customer.subscription.updated`        | Plan changed      | Update plan         |
| `customer.subscription.deleted`        | Cancelled         | Downgrade to free   |
| `invoice.paid`                         | Payment succeeded | Extend subscription |
| `invoice.payment_failed`               | Payment failed    | Notify customer     |
| `customer.subscription.trial_will_end` | Trial ending      | Notify customer     |

### Local Webhook Testing

```bash
# Start webhook forwarding
stripe listen --forward-to localhost:8080/api/v1/webhooks/stripe

# In another terminal, trigger events
stripe trigger invoice.paid
stripe trigger customer.subscription.created
```

---

## Troubleshooting

### Payment Declined

**Symptom**: Payment fails, customer card declined

**Debugging**:

```bash
# Check payment intent
stripe payment_intents retrieve pi_xxx

# View decline reason
# Response includes: last_payment_error.decline_code
```

**Common Decline Codes**:

- `insufficient_funds`: Card has no money
- `lost_card`: Card reported lost
- `expired_card`: Card expired
- `incorrect_cvc`: Wrong CVC entered
- `processing_error`: Try again

### Webhook Not Received

**Symptom**: Stripe events not reaching your server

**Debugging**:

1. **Check webhook logs**
   - Go to Dashboard > Developers > Webhooks
   - Click your endpoint
   - View recent deliveries

2. **Check signature**
   - Ensure `STRIPE_WEBHOOK_SECRET` is correct
   - Must match the specific endpoint

3. **Check URL accessibility**
   ```bash
   curl -X POST https://servicepro-api.fly.dev/api/v1/webhooks/stripe
   # Should return 400 (missing signature) not 404
   ```

### Invalid API Key

**Symptom**: `Invalid API Key provided`

**Solutions**:

1. **Check key format**
   - Test: `sk_test_...`
   - Live: `sk_live_...`

2. **Check environment**
   - Don't mix test/live keys

3. **Check key permissions**
   - Restricted keys may lack permissions

### Subscription Not Updating

**Symptom**: Customer on wrong plan

**Debugging**:

```bash
# Check subscription status
stripe subscriptions retrieve sub_xxx

# Check customer's subscriptions
stripe subscriptions list --customer=cus_xxx
```

**Solutions**:

1. **Check webhook handling**
   - Ensure webhook events are processed

2. **Manual sync**
   - Fetch subscription from Stripe
   - Update local database

---

## Customer Portal

Stripe provides a hosted portal for customers to manage billing:

### Enable Customer Portal

1. Go to [Settings > Billing > Customer Portal](https://dashboard.stripe.com/settings/billing/portal)
2. Configure allowed features:
   - Update payment method
   - View invoices
   - Cancel subscription
   - Switch plans
3. Save configuration

### Generate Portal Link

```go
import "github.com/stripe/stripe-go/v76/billingportal/session"

params := &stripe.BillingPortalSessionParams{
    Customer:  stripe.String("cus_xxx"),
    ReturnURL: stripe.String("https://app.servicepro.com/settings"),
}
s, err := session.New(params)
// Redirect customer to s.URL
```

---

## Subscription Tiers

### Recommended Structure

```
Free Tier ($0/month)
├── 1 user
├── 10 customers
├── 5 jobs/month
└── Basic features

Basic Tier ($29/month or $290/year)
├── 5 users
├── 100 customers
├── Unlimited jobs
├── Email support
└── All core features

Pro Tier ($99/month or $990/year)
├── Unlimited users
├── Unlimited customers
├── Unlimited jobs
├── Priority support
├── API access
└── Custom branding
```

### Creating in Stripe

```bash
# Create products
stripe products create --name="Free" --description="Free tier"
stripe products create --name="Basic" --description="For small teams"
stripe products create --name="Pro" --description="For growing businesses"

# Create prices
stripe prices create --product=prod_free --unit-amount=0 --currency=usd --recurring-interval=month
stripe prices create --product=prod_basic --unit-amount=2900 --currency=usd --recurring-interval=month
stripe prices create --product=prod_basic --unit-amount=29000 --currency=usd --recurring-interval=year
stripe prices create --product=prod_pro --unit-amount=9900 --currency=usd --recurring-interval=month
stripe prices create --product=prod_pro --unit-amount=99000 --currency=usd --recurring-interval=year
```

---

## Security

### API Key Security

- **Never expose secret key** (`sk_*`) in frontend code
- Store in environment variables / Fly secrets
- Publishable key (`pk_*`) can be in frontend

### Webhook Security

- Always verify webhook signatures
- Use HTTPS endpoint
- Respond quickly (< 5 seconds)

### PCI Compliance

Stripe handles PCI compliance. You maintain compliance by:

- Never logging card numbers
- Using Stripe.js or Elements for card input
- Not storing card data on your servers

---

## Testing

### Test Card Numbers

| Scenario                | Card Number         |
| ----------------------- | ------------------- |
| Success                 | 4242 4242 4242 4242 |
| Decline                 | 4000 0000 0000 0002 |
| Insufficient funds      | 4000 0000 0000 9995 |
| Requires authentication | 4000 0025 0000 3155 |
| Processing error        | 4000 0000 0000 0119 |

### Test Webhook Events

```bash
# Successful payment
stripe trigger payment_intent.succeeded

# Subscription created
stripe trigger customer.subscription.created

# Invoice paid
stripe trigger invoice.paid

# Payment failed
stripe trigger invoice.payment_failed
```

---

## Useful Links

- [Stripe Documentation](https://stripe.com/docs)
- [Stripe API Reference](https://stripe.com/docs/api)
- [Stripe CLI](https://stripe.com/docs/stripe-cli)
- [Stripe Go SDK](https://github.com/stripe/stripe-go)
- [Stripe Test Cards](https://stripe.com/docs/testing)
- [Stripe Dashboard](https://dashboard.stripe.com)
- [Stripe Status](https://status.stripe.com)
