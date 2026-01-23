# Payments API

Payment processing API integrated with Stripe for secure payment handling.

## Overview

The Payments API provides:

- Payment intent creation for flexible payment flows
- 3D Secure authentication support
- Saved payment method management
- Full and partial refunds
- Payment receipts

### Key Features

- **PCI Compliant**: Never handles raw card data server-side
- **3D Secure**: Automatic Strong Customer Authentication (SCA)
- **Multi-Currency**: USD, EUR, GBP, CAD, AUD, JPY
- **Idempotency**: Protection against duplicate operations

## Endpoints

| Method | Endpoint                | Description               |
| ------ | ----------------------- | ------------------------- |
| POST   | `/payments/intent`      | Create payment intent     |
| POST   | `/payments/confirm`     | Confirm payment           |
| GET    | `/payments/methods`     | Get saved payment methods |
| POST   | `/payments/refund`      | Create refund             |
| GET    | `/payments/receipt/:id` | Get payment receipt       |

## Create Payment Intent

Creates a new Stripe payment intent for processing a payment.

```http
POST /api/v1/payments/intent
```

### Request Body

```json
{
  "amount": 100.5,
  "currency": "usd",
  "description": "Payment for services",
  "invoice_id": "550e8400-e29b-41d4-a716-446655440000",
  "payment_method_id": "pm_1234567890",
  "customer_id": "cus_1234567890",
  "statement_descriptor": "ACME Corp",
  "receipt_email": "customer@example.com",
  "capture_method": "automatic",
  "save_payment_method": true,
  "metadata": {
    "order_number": "ORD-12345"
  }
}
```

### Fields

| Field                | Type    | Required | Description                                      |
| -------------------- | ------- | -------- | ------------------------------------------------ |
| amount               | decimal | Yes      | Amount in currency units (e.g., 100.50)          |
| currency             | string  | Yes      | 3-letter ISO code (USD, EUR, GBP, CAD, AUD, JPY) |
| description          | string  | No       | Payment description (max 500 chars)              |
| invoice_id           | UUID    | No       | Associated invoice                               |
| payment_method_id    | string  | No       | Stripe payment method ID                         |
| customer_id          | string  | No       | Stripe customer ID                               |
| statement_descriptor | string  | No       | Bank statement text (max 22 chars)               |
| receipt_email        | string  | No       | Email for receipt                                |
| capture_method       | string  | No       | "automatic" or "manual"                          |
| save_payment_method  | boolean | No       | Save for future use                              |
| metadata             | object  | No       | Custom key-value pairs (max 50)                  |

### Amount Validation

| Currency           | Minimum | Maximum     | Decimal Places |
| ------------------ | ------- | ----------- | -------------- |
| USD, EUR, CAD, AUD | $0.50   | $999,999.99 | 2              |
| GBP                | £0.30   | £999,999.99 | 2              |
| JPY                | ¥50     | ¥999,999    | 0              |

### Response

```json
{
  "payment_id": "550e8400-e29b-41d4-a716-446655440000",
  "client_secret": "pi_1234567890_secret_abcdefg",
  "status": "requires_payment_method",
  "amount": 100.5,
  "currency": "usd",
  "requires_action": false,
  "next_action": null
}
```

### Status Values

| Status                  | Description                     |
| ----------------------- | ------------------------------- |
| requires_payment_method | Waiting for payment method      |
| requires_confirmation   | Ready to be confirmed           |
| requires_action         | 3D Secure authentication needed |
| processing              | Payment is being processed      |
| succeeded               | Payment completed successfully  |

## Confirm Payment

Confirms a payment intent with optional payment method.

```http
POST /api/v1/payments/confirm
```

### Request Body

```json
{
  "payment_id": "550e8400-e29b-41d4-a716-446655440000",
  "payment_method_id": "pm_1234567890",
  "return_url": "https://yoursite.com/payment/complete"
}
```

### Response

```json
{
  "payment_id": "550e8400-e29b-41d4-a716-446655440000",
  "status": "succeeded",
  "amount": 100.5,
  "currency": "usd",
  "requires_action": false,
  "next_action": null,
  "receipt_url": "https://stripe.com/receipt/abc123"
}
```

## Get Payment Methods

Retrieves all saved payment methods for the authenticated user.

```http
GET /api/v1/payments/methods
```

### Response

```json
[
  {
    "id": "pm_1234567890",
    "type": "card",
    "card": {
      "brand": "visa",
      "last4": "4242",
      "exp_month": 12,
      "exp_year": 2025,
      "country": "US"
    },
    "billing_details": {
      "name": "John Doe",
      "email": "john@example.com",
      "address": {
        "line1": "123 Main St",
        "city": "San Francisco",
        "state": "CA",
        "postal_code": "94102",
        "country": "US"
      }
    },
    "created": "2025-01-15T10:30:00Z"
  }
]
```

## Create Refund

Creates a full or partial refund for a successful payment.

```http
POST /api/v1/payments/refund
```

### Request Body

```json
{
  "payment_id": "550e8400-e29b-41d4-a716-446655440000",
  "amount": 50.25,
  "reason": "requested_by_customer",
  "metadata": {
    "refund_reason": "Customer changed mind"
  }
}
```

### Fields

| Field      | Type    | Required | Description                                  |
| ---------- | ------- | -------- | -------------------------------------------- |
| payment_id | UUID    | Yes      | Payment to refund                            |
| amount     | decimal | No       | Partial refund amount (omit for full)        |
| reason     | string  | No       | duplicate, fraudulent, requested_by_customer |
| metadata   | object  | No       | Additional information                       |

### Validation

- Payment must be in `succeeded` status
- Partial refund cannot exceed remaining amount
- Total refunds cannot exceed original payment

### Response

```json
{
  "refund_id": "550e8400-e29b-41d4-a716-446655440001",
  "payment_id": "550e8400-e29b-41d4-a716-446655440000",
  "amount": "50.25",
  "currency": "usd",
  "status": "succeeded",
  "reason": "requested_by_customer",
  "created_at": "2025-01-15T10:30:00Z"
}
```

## Get Payment Receipt

Retrieves detailed receipt information for a payment.

```http
GET /api/v1/payments/receipt/:payment_id
```

### Response

```json
{
  "payment_id": "550e8400-e29b-41d4-a716-446655440000",
  "amount": "100.50",
  "currency": "usd",
  "status": "succeeded",
  "payment_method_type": "card",
  "description": "Service payment",
  "receipt_url": "https://stripe.com/receipt/abc123",
  "receipt_email": "customer@example.com",
  "processed_at": "2025-01-15T10:30:00Z",
  "created_at": "2025-01-15T10:25:00Z",
  "card": {
    "brand": "visa",
    "last4": "4242",
    "exp_month": 12,
    "exp_year": 2025
  },
  "refunded_amount": "25.00",
  "net_amount": "75.50"
}
```

## Error Handling

### Error Response Format

```json
{
  "error": "Error description",
  "details": "Additional error details"
}
```

### Common Errors

| Error                | Status | Description                            |
| -------------------- | ------ | -------------------------------------- |
| Invalid amount       | 400    | Amount below minimum or invalid format |
| Unsupported currency | 400    | Currency not supported                 |
| Payment not found    | 404    | Payment doesn't exist                  |
| Access denied        | 403    | Payment belongs to different user      |
| Card declined        | 400    | Payment method was declined            |

### Card Decline Codes

| Code               | Description                |
| ------------------ | -------------------------- |
| card_declined      | Generic decline            |
| expired_card       | Card has expired           |
| insufficient_funds | Not enough funds           |
| incorrect_cvc      | CVC verification failed    |
| processing_error   | Temporary processing error |

## Frontend Integration

### 1. Create Payment Intent

```javascript
const response = await fetch('/api/v1/payments/intent', {
  method: 'POST',
  headers: {
    Authorization: `Bearer ${jwtToken}`,
    'Content-Type': 'application/json',
  },
  body: JSON.stringify({
    amount: 100.5,
    currency: 'usd',
    description: 'Service payment',
  }),
});

const { client_secret, payment_id } = await response.json();
```

### 2. Collect Payment with Stripe.js

```javascript
const stripe = Stripe('pk_test_...');

const { error, paymentIntent } = await stripe.confirmCardPayment(
  client_secret,
  {
    payment_method: {
      card: cardElement,
      billing_details: {
        name: 'Customer Name',
        email: 'customer@example.com',
      },
    },
  }
);

if (error) {
  console.error(error.message);
} else if (paymentIntent.status === 'succeeded') {
  console.log('Payment successful!');
}
```

### 3. Handle 3D Secure

Stripe.js automatically handles 3D Secure authentication when required.

## Webhooks

The backend handles these Stripe webhook events:

| Event                         | Description       |
| ----------------------------- | ----------------- |
| payment_intent.succeeded      | Payment completed |
| payment_intent.payment_failed | Payment failed    |
| charge.refunded               | Refund processed  |
| charge.dispute.created        | Dispute opened    |

## Testing

### Test Card Numbers

| Card               | Number              | Result                  |
| ------------------ | ------------------- | ----------------------- |
| Success            | 4242 4242 4242 4242 | Payment succeeds        |
| 3D Secure          | 4000 0027 6000 3184 | Requires authentication |
| Declined           | 4000 0000 0000 0002 | Card declined           |
| Insufficient funds | 4000 0000 0000 9995 | Insufficient funds      |

### Test with cURL

```bash
# Create payment intent
curl -X POST http://localhost:8080/api/v1/payments/intent \
  -H "Authorization: Bearer YOUR_JWT_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "amount": 100.50,
    "currency": "usd",
    "description": "Test payment"
  }'

# Full refund
curl -X POST http://localhost:8080/api/v1/payments/refund \
  -H "Authorization: Bearer YOUR_JWT_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "payment_id": "550e8400-e29b-41d4-a716-446655440000",
    "reason": "requested_by_customer"
  }'
```

## Security

### PCI Compliance

- Never send raw card data to your server
- Use Stripe.js or Stripe Mobile SDKs
- Card data goes directly from client to Stripe
- Only use `client_secret` and `payment_method_id` server-side

### API Key Security

- Store Stripe keys in environment variables
- Never commit keys to version control
- Use separate keys for test/production
- Rotate keys if compromised

## Configuration

### Environment Variables

```bash
# Required
STRIPE_SECRET_KEY=sk_test_...
STRIPE_PUBLISHABLE_KEY=pk_test_...
STRIPE_WEBHOOK_SECRET=whsec_...

# Optional
STRIPE_MAX_NETWORK_RETRIES=3
STRIPE_REQUESTS_PER_SECOND=100
```

## Rate Limits

Stripe's rate limits apply:

- 100 requests/second in test mode
- Higher limits in production

The backend includes built-in rate limiting to prevent abuse.
