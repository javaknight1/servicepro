# Stripe Payment Integration - Complete Documentation

## Table of Contents

1. [Overview](#overview)
2. [Features](#features)
3. [Installation & Setup](#installation--setup)
4. [Configuration](#configuration)
5. [Quick Start](#quick-start)
6. [API Reference](#api-reference)
7. [Webhook Events](#webhook-events)
8. [Testing](#testing)
9. [Security](#security)
10. [Error Handling](#error-handling)
11. [Best Practices](#best-practices)
12. [Troubleshooting](#troubleshooting)

---

## Overview

The Stripe Payment Integration provides a comprehensive solution for processing payments, managing customers, handling refunds, and processing webhook events. Built with the official Stripe Go SDK v76+, it offers enterprise-grade features including automatic retries, rate limiting, webhook signature verification, and comprehensive error handling.

### Key Capabilities

- **Payment Processing**: Create and manage Payment Intents
- **Customer Management**: Create and retrieve Stripe customers
- **Refunds**: Process full and partial refunds
- **Webhooks**: Secure webhook handling with signature verification
- **Event Processing**: Extensible event handler system
- **Security**: TLS 1.2+, PCI compliance considerations
- **Rate Limiting**: Built-in rate limiting for API calls
- **Idempotency**: Automatic idempotency key generation

---

## Features

### Payment Intent Operations

- **Create Payment Intent**: Initiate a new payment
- **Get Payment Intent**: Retrieve payment status
- **Confirm Payment Intent**: Confirm a payment
- **Cancel Payment Intent**: Cancel a pending payment
- **Capture Payment Intent**: Capture authorized funds (manual capture)

### Customer Operations

- **Create Customer**: Register a new Stripe customer
- **Get Customer**: Retrieve customer details
- **Attach Payment Methods**: Link payment methods to customers

### Refund Operations

- **Create Refund**: Process full or partial refunds
- **Refund Tracking**: Track refund status and history

### Webhook Processing

- **Signature Verification**: Secure webhook validation
- **Event Handling**: Extensible event processor
- **Idempotency**: Prevent duplicate event processing
- **Automatic Retry**: Built-in retry mechanism for failed handlers

---

## Installation & Setup

### Prerequisites

1. **Go 1.21+**
2. **Stripe Account** (test and live mode)
3. **API Keys** from Stripe Dashboard

### Install Stripe SDK

The Stripe SDK is already included in the project dependencies:

```bash
go get github.com/stripe/stripe-go/v76
```

### Environment Variables

Create a `.env` file or set environment variables:

```bash
# Stripe API Keys
STRIPE_SECRET_KEY=sk_test_your_secret_key_here
STRIPE_PUBLISHABLE_KEY=pk_test_your_publishable_key_here
STRIPE_WEBHOOK_SECRET=whsec_your_webhook_secret_here

# Optional Configuration
STRIPE_LOG_LEVEL=info
STRIPE_MAX_RETRIES=3
```

### Get Your API Keys

1. Go to https://dashboard.stripe.com/apikeys
2. Copy your **Test Mode** keys (for development)
3. Copy your **Live Mode** keys (for production)
4. Create a webhook endpoint to get the webhook secret

### Setup Webhook Endpoint

1. Go to https://dashboard.stripe.com/webhooks
2. Click "Add endpoint"
3. Enter your webhook URL: `https://your-domain.com/api/v1/stripe/webhooks`
4. Select events to listen for (or select "receive all events")
5. Copy the webhook signing secret

---

## Configuration

### Load Configuration

```go
import (
    stripeService "github.com/javaknight1/servicepro/backend/internal/services/stripe"
)

// Load from environment variables
config, err := stripeService.LoadFromEnv()
if err != nil {
    log.Fatal(err)
}

// Validate configuration
if err := config.Validate(); err != nil {
    log.Fatal(err)
}
```

### Manual Configuration

```go
config := &stripeService.Config{
    SecretKey:      "sk_test_...",
    PublishableKey: "pk_test_...",
    WebhookSecret:  "whsec_...",
    Environment:    stripeService.EnvironmentTest,
    MaxNetworkRetries: 3,
    RequestsPerSecond: 100,
    WebhookTolerance: 300 * time.Second,
}
```

### Configuration Options

| Option              | Type     | Default       | Description                              |
| ------------------- | -------- | ------------- | ---------------------------------------- |
| `SecretKey`         | string   | required      | Stripe secret key (sk*test* or sk*live*) |
| `PublishableKey`    | string   | required      | Stripe publishable key                   |
| `WebhookSecret`     | string   | required      | Webhook signing secret                   |
| `Environment`       | string   | auto-detected | test or live                             |
| `MaxNetworkRetries` | int      | 3             | Max retries for failed requests          |
| `ConnectTimeout`    | duration | 30s           | Connection timeout                       |
| `Timeout`           | duration | 80s           | Request timeout                          |
| `RequestsPerSecond` | int      | 100           | Rate limit for API calls                 |
| `WebhookTolerance`  | duration | 300s          | Max age for webhook events               |

---

## Quick Start

### 1. Initialize Stripe Client

```go
// In your main.go or setup function
config, err := stripeService.LoadFromEnv()
if err != nil {
    log.Fatal(err)
}

client, err := stripeService.NewClient(config)
if err != nil {
    log.Fatal(err)
}
```

### 2. Create a Payment Intent

```go
ctx := context.Background()

params := &stripeService.PaymentIntentParams{
    Amount:      1999, // $19.99 in cents
    Currency:    "usd",
    Description: strPtr("Order #12345"),
    Metadata: map[string]string{
        "order_id": "12345",
        "user_id":  "user_789",
    },
}

paymentIntent, err := client.CreatePaymentIntent(ctx, params)
if err != nil {
    log.Fatal(err)
}

fmt.Printf("Payment Intent created: %s\n", paymentIntent.ID)
fmt.Printf("Client Secret: %s\n", paymentIntent.ClientSecret)
```

### 3. Create a Customer

```go
params := &stripeService.CustomerParams{
    Email:       strPtr("customer@example.com"),
    Name:        strPtr("John Doe"),
    Description: strPtr("Premium customer"),
    Metadata: map[string]string{
        "user_id": "user_789",
    },
}

customer, err := client.CreateCustomer(ctx, params)
if err != nil {
    log.Fatal(err)
}

fmt.Printf("Customer created: %s\n", customer.ID)
```

### 4. Process a Refund

```go
params := &stripeService.RefundParams{
    PaymentIntentID: strPtr("pi_xxx"),
    Amount:          intPtr(1000), // $10.00 partial refund
    Reason:          strPtr("requested_by_customer"),
    Metadata: map[string]string{
        "refund_reason": "Product not as described",
    },
}

refund, err := client.CreateRefund(ctx, params)
if err != nil {
    log.Fatal(err)
}

fmt.Printf("Refund created: %s\n", refund.ID)
```

### 5. Setup Webhook Handler

```go
// Create event processor
eventProcessor := stripeService.NewEventProcessor(config.Logger)

// Register custom event handlers
eventProcessor.RegisterHandler(
    stripeService.EventPaymentIntentSucceeded,
    func(ctx context.Context, event *stripe.Event) error {
        pi, err := stripeService.ParsePaymentIntent(event)
        if err != nil {
            return err
        }

        // Handle successful payment
        log.Printf("Payment succeeded: %s", pi.ID)
        // Update database, send confirmation email, etc.

        return nil
    },
)

// Or use default handlers
eventProcessor.RegisterDefaultHandlers()

// Create webhook handler
webhookHandler := stripeService.NewWebhookHandler(config, eventProcessor)
```

---

## API Reference

### Base URL

```
https://your-domain.com/api/v1/stripe
```

All endpoints (except webhooks) require authentication with JWT Bearer token.

### Payment Intent Endpoints

#### 1. Create Payment Intent

**POST** `/payment-intents`

Create a new payment intent for processing a payment.

**Request Body:**

```json
{
  "amount": 19.99,
  "currency": "usd",
  "customer_id": "cus_xxx",
  "payment_method_id": "pm_xxx",
  "description": "Order #12345",
  "metadata": {
    "order_id": "12345"
  },
  "capture_method": "automatic",
  "confirmation_method": "automatic"
}
```

**Response:** `200 OK`

```json
{
  "id": "pi_xxx",
  "amount": 1999,
  "currency": "usd",
  "status": "requires_payment_method",
  "client_secret": "pi_xxx_secret_yyy",
  "customer_id": "cus_xxx",
  "description": "Order #12345",
  "metadata": {
    "order_id": "12345"
  },
  "created": "2024-01-15T10:30:00Z"
}
```

#### 2. Get Payment Intent

**GET** `/payment-intents/:id`

Retrieve a payment intent by ID.

**Response:** `200 OK`

#### 3. Confirm Payment Intent

**POST** `/payment-intents/:id/confirm`

Confirm a payment intent.

**Request Body (Optional):**

```json
{
  "payment_method_id": "pm_xxx"
}
```

**Response:** `200 OK`

#### 4. Cancel Payment Intent

**POST** `/payment-intents/:id/cancel`

Cancel a payment intent.

**Request Body (Optional):**

```json
{
  "reason": "requested_by_customer"
}
```

**Response:** `200 OK`

#### 5. Capture Payment Intent

**POST** `/payment-intents/:id/capture`

Capture an authorized payment (for manual capture).

**Request Body (Optional):**

```json
{
  "amount_to_capture": 15.99
}
```

**Response:** `200 OK`

### Customer Endpoints

#### 1. Create Customer

**POST** `/customers`

Create a new Stripe customer.

**Request Body:**

```json
{
  "email": "customer@example.com",
  "name": "John Doe",
  "phone": "+1234567890",
  "description": "Premium customer",
  "metadata": {
    "user_id": "user_789"
  },
  "payment_method_id": "pm_xxx"
}
```

**Response:** `200 OK`

```json
{
  "id": "cus_xxx",
  "email": "customer@example.com",
  "name": "John Doe",
  "phone": "+1234567890",
  "description": "Premium customer",
  "metadata": {
    "user_id": "user_789"
  },
  "default_payment_method": "pm_xxx",
  "created": "2024-01-15T10:30:00Z"
}
```

#### 2. Get Customer

**GET** `/customers/:id`

Retrieve a customer by ID.

**Response:** `200 OK`

### Refund Endpoints

#### 1. Create Refund

**POST** `/refunds`

Create a refund for a charge or payment intent.

**Request Body:**

```json
{
  "payment_intent_id": "pi_xxx",
  "amount": 10.0,
  "reason": "requested_by_customer",
  "metadata": {
    "refund_reason": "Product not as described"
  }
}
```

**Response:** `200 OK`

```json
{
  "id": "re_xxx",
  "amount": 1000,
  "currency": "usd",
  "payment_intent_id": "pi_xxx",
  "status": "succeeded",
  "reason": "requested_by_customer",
  "metadata": {
    "refund_reason": "Product not as described"
  },
  "created": "2024-01-15T10:30:00Z"
}
```

### Webhook Endpoint

#### Handle Webhook

**POST** `/webhooks`

Receives and processes Stripe webhook events.

**Headers:**

- `Stripe-Signature`: Webhook signature for verification

**Request Body:** Raw JSON from Stripe

**Response:** `200 OK`

```json
{
  "received": true,
  "event_id": "evt_xxx"
}
```

#### Get Webhook Statistics

**GET** `/webhooks/stats`

Returns webhook processing statistics.

**Response:** `200 OK`

```json
{
  "processed_events_count": 150
}
```

---

## Webhook Events

### Supported Event Types

#### Payment Intent Events

- `payment_intent.succeeded` - Payment was successful
- `payment_intent.payment_failed` - Payment failed
- `payment_intent.created` - Payment intent created
- `payment_intent.canceled` - Payment intent canceled
- `payment_intent.processing` - Payment is processing
- `payment_intent.requires_action` - Payment requires additional action

#### Charge Events

- `charge.succeeded` - Charge was successful
- `charge.failed` - Charge failed
- `charge.refunded` - Charge was refunded
- `charge.dispute.created` - Dispute was created

#### Customer Events

- `customer.created` - Customer was created
- `customer.updated` - Customer was updated
- `customer.deleted` - Customer was deleted

#### Refund Events

- `refund.created` - Refund was created
- `refund.updated` - Refund was updated
- `refund.failed` - Refund failed

### Custom Event Handlers

```go
// Register a custom handler for successful payments
eventProcessor.RegisterHandler(
    stripeService.EventPaymentIntentSucceeded,
    func(ctx context.Context, event *stripe.Event) error {
        // Parse the payment intent
        pi, err := stripeService.ParsePaymentIntent(event)
        if err != nil {
            return err
        }

        // Your custom logic
        log.Printf("Processing successful payment: %s", pi.ID)

        // Update database
        if err := updateInvoiceStatus(ctx, pi.Metadata["invoice_id"], "paid"); err != nil {
            return err
        }

        // Send confirmation email
        if err := sendPaymentConfirmation(ctx, pi.CustomerID); err != nil {
            log.Printf("Failed to send confirmation: %v", err)
            // Don't fail the event processing
        }

        return nil
    },
)
```

### Multiple Handlers

You can register multiple handlers for the same event:

```go
// First handler: Update database
eventProcessor.RegisterHandler(
    stripeService.EventPaymentIntentSucceeded,
    updateDatabaseHandler,
)

// Second handler: Send notification
eventProcessor.RegisterHandler(
    stripeService.EventPaymentIntentSucceeded,
    sendNotificationHandler,
)

// Both will be executed
```

---

## Testing

### Unit Tests

Run all Stripe tests:

```bash
go test ./internal/services/stripe/... -v
```

Run specific tests:

```bash
go test ./internal/services/stripe/... -v -run TestConfig_Validate
```

### Test Configuration

```go
func getTestConfig() *stripeService.Config {
    return &stripeService.Config{
        SecretKey:      "sk_test_fake123",
        PublishableKey: "pk_test_fake123",
        WebhookSecret:  "whsec_test_secret",
        Environment:    stripeService.EnvironmentTest,
    }
}
```

### Testing Webhooks

#### Using Stripe CLI

1. Install Stripe CLI:

```bash
brew install stripe/stripe-cli/stripe
```

2. Login to Stripe:

```bash
stripe login
```

3. Forward webhooks to local server:

```bash
stripe listen --forward-to localhost:8080/api/v1/stripe/webhooks
```

4. Trigger test events:

```bash
stripe trigger payment_intent.succeeded
stripe trigger customer.created
stripe trigger charge.refunded
```

#### Testing Signature Verification

```go
func TestWebhookSignature(t *testing.T) {
    config := getTestConfig()
    processor := stripeService.NewEventProcessor(nil)
    handler := stripeService.NewWebhookHandler(config, processor)

    // Simulate a webhook payload
    payload := []byte(`{"type":"payment_intent.succeeded"}`)

    // Generate a signature (you'll need Stripe CLI for real signatures)
    signature := "test_signature"

    err := handler.TestWebhookSignature(payload, signature)
    assert.Error(t, err) // Should fail with invalid signature
}
```

### Mock Testing

```go
// Mock Stripe client for testing
type MockStripeClient struct {
    CreatePaymentIntentFunc func(ctx context.Context, params *PaymentIntentParams) (*PaymentIntentResult, error)
}

func (m *MockStripeClient) CreatePaymentIntent(ctx context.Context, params *PaymentIntentParams) (*PaymentIntentResult, error) {
    if m.CreatePaymentIntentFunc != nil {
        return m.CreatePaymentIntentFunc(ctx, params)
    }
    return nil, errors.New("not implemented")
}
```

---

## Security

### PCI Compliance

**Never handle raw card data** on your server. The integration is designed to:

- Use Stripe.js on the frontend to collect card details
- Create Payment Methods client-side
- Only handle tokens/IDs server-side
- Never log or store card numbers

### TLS Requirements

- **Minimum TLS 1.2** for all Stripe API calls
- The Stripe Go SDK enforces this automatically
- Webhook endpoint must use HTTPS in production

### API Key Security

```go
// ✅ Good: Load from environment
config, err := stripeService.LoadFromEnv()

// ❌ Bad: Hardcode in source code
config := &Config{
    SecretKey: "hardcoded_secret_key", // NEVER DO THIS
}
```

**Best Practices:**

1. Use environment variables
2. Use secret management services (AWS Secrets Manager, etc.)
3. Never commit keys to version control
4. Use different keys for test/production
5. Rotate keys regularly

### Webhook Security

The webhook handler implements multiple security layers:

1. **Signature Verification**

   - Every webhook is verified using the webhook secret
   - Prevents unauthorized requests

2. **Timestamp Validation**

   - Rejects events older than 5 minutes (configurable)
   - Prevents replay attacks

3. **Idempotency**
   - Tracks processed events
   - Prevents duplicate processing

```go
// Webhook signature is automatically verified
handler.HandleHTTPWebhook(w, r)
// Will return 400 if signature is invalid
```

### Rate Limiting

Built-in rate limiting protects against API abuse:

```go
// Configure rate limit
config := &Config{
    RequestsPerSecond: 100, // Max 100 requests per second
}

// Rate limiter is automatically applied
client, _ := NewClient(config)
```

### Error Logging

**Never log sensitive data:**

```go
// ✅ Good: Log masked keys
log.Printf("Config: %s", config.MaskSecretKey())

// ❌ Bad: Log full keys
log.Printf("Secret: %s", config.SecretKey)
```

---

## Error Handling

### Error Types

#### 1. Configuration Errors

```go
config, err := stripeService.LoadFromEnv()
if err != nil {
    // Handle missing or invalid environment variables
    log.Fatal("Configuration error:", err)
}
```

#### 2. API Errors

```go
pi, err := client.CreatePaymentIntent(ctx, params)
if err != nil {
    // Check for specific Stripe errors
    if stripeErr, ok := err.(*stripe.Error); ok {
        switch stripeErr.Code {
        case stripe.ErrorCodeCardDeclined:
            // Handle declined card
        case stripe.ErrorCodeExpiredCard:
            // Handle expired card
        case stripe.ErrorCodeInsufficientFunds:
            // Handle insufficient funds
        default:
            // Handle other errors
        }
    }
}
```

#### 3. Webhook Errors

```go
response, err := handler.HandleWebhook(ctx, req)
if err != nil {
    // Log error but return 200 to prevent retries for invalid signatures
    log.Printf("Webhook error: %v", err)

    // Check response for details
    if response != nil && response.Error != "" {
        log.Printf("Error: %s", response.Error)
    }
}
```

### Retry Logic

The client automatically retries failed requests:

```go
config := &Config{
    MaxNetworkRetries: 3, // Retry up to 3 times
}
```

**Retries are attempted for:**

- Network errors
- 500, 502, 503, 504 status codes
- Connection timeouts

**Retries are NOT attempted for:**

- 400, 401, 403, 404 status codes
- Invalid parameters
- Authentication failures

---

## Best Practices

### 1. Idempotency

Always use idempotency keys for critical operations:

```go
idempotencyKey := uuid.New().String()

params := &stripeService.PaymentIntentParams{
    Amount:         1999,
    Currency:       "usd",
    IdempotencyKey: &idempotencyKey,
}

// If request fails and is retried, Stripe will return the same result
pi, err := client.CreatePaymentIntent(ctx, params)
```

### 2. Metadata Usage

Use metadata to link Stripe objects to your database:

```go
params := &stripeService.PaymentIntentParams{
    Amount:   1999,
    Currency: "usd",
    Metadata: map[string]string{
        "order_id":   "12345",
        "user_id":    "user_789",
        "invoice_id": "inv_456",
    },
}
```

### 3. Webhook Handling

**Always return 200 quickly:**

```go
func handlePaymentSuccess(ctx context.Context, event *stripe.Event) error {
    pi, _ := stripeService.ParsePaymentIntent(event)

    // Queue for background processing
    queue.Enqueue("process_payment", pi.ID)

    // Return quickly
    return nil
}
```

**Don't perform long-running operations in webhook handlers:**

- Use a job queue (Redis, RabbitMQ, etc.)
- Process asynchronously
- Return 200 within 5 seconds

### 4. Amount Handling

**Always use cents (smallest currency unit):**

```go
// ✅ Good: Use helper functions
amount := stripeService.ConvertDecimalToCents(decimal.NewFromFloat(19.99))

// ✅ Good: Manually calculate
amountCents := int64(19.99 * 100)

// ❌ Bad: Use float directly
params.Amount = 19.99 // Wrong! Stripe expects cents
```

### 5. Customer Management

**Link Stripe customers to your users:**

```go
// When creating a user
user := createUser(...)

// Create Stripe customer
customer, err := client.CreateCustomer(ctx, &stripeService.CustomerParams{
    Email: &user.Email,
    Metadata: map[string]string{
        "user_id": user.ID.String(),
    },
})

// Store Stripe customer ID
user.StripeCustomerID = customer.ID
saveUser(user)
```

### 6. Error Messages

**Provide user-friendly error messages:**

```go
func handlePaymentError(err error) string {
    if stripeErr, ok := err.(*stripe.Error); ok {
        switch stripeErr.Code {
        case stripe.ErrorCodeCardDeclined:
            return "Your card was declined. Please try another payment method."
        case stripe.ErrorCodeExpiredCard:
            return "Your card has expired. Please update your payment method."
        case stripe.ErrorCodeInsufficientFunds:
            return "Insufficient funds. Please try another card."
        default:
            return "Payment failed. Please try again or contact support."
        }
    }
    return "An unexpected error occurred. Please try again."
}
```

---

## Troubleshooting

### Common Issues

#### 1. Webhook Signature Verification Failed

**Problem:** Webhooks return 400 "signature verification failed"

**Solutions:**

- Verify webhook secret is correct
- Check that you're using raw request body (not parsed JSON)
- Ensure Stripe-Signature header is forwarded
- Check server time is synchronized (NTP)

```go
// Correct: Use raw body
body, _ := io.ReadAll(r.Body)

// Incorrect: Parse JSON first
var data map[string]interface{}
json.NewDecoder(r.Body).Decode(&data) // Don't do this for webhooks
```

#### 2. Rate Limit Exceeded

**Problem:** Getting rate limit errors

**Solutions:**

- Reduce RequestsPerSecond in config
- Implement exponential backoff
- Batch operations where possible

```go
config := &Config{
    RequestsPerSecond: 50, // Reduce from 100
}
```

#### 3. Invalid API Key

**Problem:** Authentication errors

**Solutions:**

- Verify API key format (sk*test* or sk*live*)
- Check environment variable is set
- Ensure no whitespace in key
- Verify key is active in Stripe dashboard

```bash
# Check environment variable
echo $STRIPE_SECRET_KEY

# Should start with sk_test_ or sk_live_
```

#### 4. Idempotency Key Mismatch

**Problem:** "Idempotency key used with different parameters"

**Solutions:**

- Generate new idempotency key for each unique request
- Don't reuse keys for different parameters
- Use UUID for idempotency keys

```go
// ✅ Good: New key for each request
key1 := uuid.New().String()
key2 := uuid.New().String()

// ❌ Bad: Reusing the same key
key := "same_key_for_all"
```

#### 5. Amount Validation Errors

**Problem:** "Amount must be at least $0.50 usd"

**Solutions:**

- Ensure amount is in cents (smallest currency unit)
- Check minimum amount for currency
- Verify amount is positive integer

```go
// Minimum amounts by currency
// USD: 50 cents ($0.50)
// EUR: 50 cents (€0.50)
// GBP: 30 pence (£0.30)
// JPY: 50 yen (¥50)

amount := stripeService.ConvertDecimalToCents(decimal.NewFromFloat(0.50))
```

### Debugging

#### Enable Debug Logging

```go
config := &Config{
    LogLevel: "debug",
    Logger:   &customLogger{},
}
```

#### Test with Stripe CLI

```bash
# Listen to all events
stripe listen

# Forward to local server
stripe listen --forward-to localhost:8080/api/v1/stripe/webhooks

# Trigger specific events
stripe trigger payment_intent.succeeded
stripe trigger payment_intent.payment_failed

# Test with specific amount
stripe trigger payment_intent.succeeded --add payment_intent:amount=2000
```

#### Check Stripe Dashboard

1. Go to https://dashboard.stripe.com/test/logs
2. View API requests and responses
3. Check webhook delivery attempts
4. Review failed events

---

## Support

For issues, questions, or contributions:

- **Stripe Documentation**: https://stripe.com/docs/api
- **Stripe Support**: https://support.stripe.com
- **GitHub Issues**: Report bugs and feature requests
- **Email**: support@servicepro.com

---

## License

Copyright (c) 2024 ServicePro. All rights reserved.
