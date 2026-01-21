# Payment Processing API

Comprehensive documentation for the ServicePro payment processing API, built on Stripe.

## Table of Contents

1. [Overview](#overview)
2. [Authentication](#authentication)
3. [API Endpoints](#api-endpoints)
4. [Request/Response Models](#requestresponse-models)
5. [Error Handling](#error-handling)
6. [Testing](#testing)
7. [Security Considerations](#security-considerations)
8. [Integration Guide](#integration-guide)

## Overview

The Payment API provides a complete payment processing solution integrated with Stripe. It supports:

- Creating payment intents for flexible payment flows
- Confirming payments with 3D Secure support
- Managing saved payment methods
- Processing full and partial refunds
- Generating payment receipts

### Key Features

- **PCI Compliance**: Never handles raw card data server-side
- **3D Secure Support**: Automatic handling of Strong Customer Authentication (SCA)
- **Multi-Currency**: Support for USD, EUR, GBP, CAD, AUD, JPY
- **Idempotency**: Built-in protection against duplicate operations
- **Rate Limiting**: Stripe-level rate limiting for API protection
- **Comprehensive Validation**: Input validation for all payment operations

## Authentication

All payment endpoints require JWT authentication via the `Authorization` header:

```
Authorization: Bearer <jwt_token>
```

The JWT token must contain the user's ID (`user_id` claim) to process payments on their behalf.

## API Endpoints

### 1. Create Payment Intent

Creates a new Stripe payment intent for processing a payment.

**Endpoint:** `POST /api/payments/intent`

**Request Body:**

```json
{
  "amount": 100.5,
  "currency": "usd",
  "description": "Payment for services",
  "invoice_id": "550e8400-e29b-41d4-a716-446655440000",
  "order_id": "550e8400-e29b-41d4-a716-446655440001",
  "payment_method_id": "pm_1234567890",
  "customer_id": "cus_1234567890",
  "statement_descriptor": "ACME Corp",
  "receipt_email": "customer@example.com",
  "capture_method": "automatic",
  "save_payment_method": true,
  "metadata": {
    "order_number": "ORD-12345",
    "customer_note": "Rush delivery"
  }
}
```

**Field Descriptions:**

- `amount` (required): Payment amount in currency units (e.g., 100.50 for $100.50)
- `currency` (required): 3-letter ISO currency code (USD, EUR, GBP, CAD, AUD, JPY)
- `description` (optional): Description of the payment (max 500 characters)
- `invoice_id` (optional): UUID of associated invoice
- `order_id` (optional): UUID of associated order
- `payment_method_id` (optional): Stripe payment method ID to use
- `customer_id` (optional): Stripe customer ID (auto-populated from user if not provided)
- `statement_descriptor` (optional): Text on customer's bank statement (max 22 characters)
- `receipt_email` (optional): Email to send payment receipt
- `capture_method` (optional): "automatic" (default) or "manual"
- `save_payment_method` (optional): Whether to save the payment method for future use
- `metadata` (optional): Key-value pairs for custom data (max 50 entries)

**Validation Rules:**

- **Amount**:

  - Must be greater than 0
  - Maximum: $999,999.99
  - Minimum varies by currency:
    - USD, EUR, CAD, AUD: $0.50
    - GBP: £0.30
    - JPY: ¥50
  - Decimal places:
    - USD, EUR, GBP, CAD, AUD: 2 decimal places
    - JPY: 0 decimal places (whole numbers only)

- **Currency**:

  - Must be 3-letter ISO code
  - Supported: USD, EUR, GBP, CAD, AUD, JPY

- **Statement Descriptor**:

  - Max 22 characters
  - Only alphanumeric, spaces, dashes, and dots

- **Metadata**:
  - Max 50 key-value pairs
  - Keys: max 40 characters
  - Values: max 500 characters

**Response (200 OK):**

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

**Response Fields:**

- `payment_id`: Internal payment record ID
- `client_secret`: Use this with Stripe.js to collect payment method
- `status`: Current payment status
  - `requires_payment_method`: Waiting for payment method
  - `requires_confirmation`: Ready to be confirmed
  - `requires_action`: 3D Secure or other authentication needed
  - `processing`: Payment is being processed
  - `succeeded`: Payment completed successfully
- `requires_action`: Whether client-side action is needed
- `next_action`: Details of required action (if any)

**Error Responses:**

- `400 Bad Request`: Invalid request data or validation failure
- `401 Unauthorized`: Missing or invalid authentication
- `500 Internal Server Error`: Payment intent creation failed

**Example cURL:**

```bash
curl -X POST https://api.servicepro.com/api/payments/intent \
  -H "Authorization: Bearer YOUR_JWT_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "amount": 100.50,
    "currency": "usd",
    "description": "Service payment"
  }'
```

---

### 2. Confirm Payment

Confirms a payment intent, optionally providing a payment method.

**Endpoint:** `POST /api/payments/confirm`

**Request Body:**

```json
{
  "payment_id": "550e8400-e29b-41d4-a716-446655440000",
  "payment_method_id": "pm_1234567890",
  "return_url": "https://yoursite.com/payment/complete"
}
```

**Field Descriptions:**

- `payment_id` (required): UUID of the payment to confirm
- `payment_method_id` (optional): Stripe payment method ID to attach
- `return_url` (optional): URL to redirect after 3D Secure authentication

**Response (200 OK):**

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

**Status Values:**

- `succeeded`: Payment completed
- `requires_action`: Client needs to handle 3D Secure
- `processing`: Payment being processed
- `failed`: Payment failed

**Error Responses:**

- `400 Bad Request`: Invalid payment ID or validation failure
- `401 Unauthorized`: Missing or invalid authentication
- `403 Forbidden`: Payment belongs to different user
- `404 Not Found`: Payment not found
- `500 Internal Server Error`: Confirmation failed

**Example cURL:**

```bash
curl -X POST https://api.servicepro.com/api/payments/confirm \
  -H "Authorization: Bearer YOUR_JWT_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "payment_id": "550e8400-e29b-41d4-a716-446655440000",
    "payment_method_id": "pm_1234567890"
  }'
```

---

### 3. Get Payment Methods

Retrieves all saved payment methods for the authenticated user.

**Endpoint:** `GET /api/payments/methods`

**Response (200 OK):**

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
      "phone": "+1234567890",
      "address": {
        "line1": "123 Main St",
        "line2": "Apt 4",
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

**Response Fields:**

- `id`: Stripe payment method ID
- `type`: Payment method type (card, bank_transfer, etc.)
- `card`: Card details (for card type)
  - `brand`: Card brand (visa, mastercard, amex, etc.)
  - `last4`: Last 4 digits
  - `exp_month`: Expiration month
  - `exp_year`: Expiration year
  - `country`: Issuing country
- `billing_details`: Customer billing information
- `created`: Creation timestamp

**Notes:**

- Returns empty array if user has no saved payment methods
- Returns empty array if user doesn't have a Stripe customer ID

**Error Responses:**

- `401 Unauthorized`: Missing or invalid authentication
- `500 Internal Server Error`: Failed to retrieve payment methods

**Example cURL:**

```bash
curl -X GET https://api.servicepro.com/api/payments/methods \
  -H "Authorization: Bearer YOUR_JWT_TOKEN"
```

---

### 4. Create Refund

Creates a full or partial refund for a successful payment.

**Endpoint:** `POST /api/payments/refund`

**Request Body:**

```json
{
  "payment_id": "550e8400-e29b-41d4-a716-446655440000",
  "amount": 50.25,
  "reason": "requested_by_customer",
  "metadata": {
    "refund_reason": "Customer changed mind",
    "processed_by": "support_agent_123"
  }
}
```

**Field Descriptions:**

- `payment_id` (required): UUID of the payment to refund
- `amount` (optional): Refund amount for partial refund (omit for full refund)
- `reason` (optional): Reason for refund
  - `duplicate`: Duplicate charge
  - `fraudulent`: Fraudulent transaction
  - `requested_by_customer`: Customer requested refund
- `metadata` (optional): Additional refund information

**Validation Rules:**

- Payment must be in `succeeded` status
- Partial refund amount must not exceed remaining payment amount
- Total refunds cannot exceed original payment amount

**Response (200 OK):**

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

**Refund Status Values:**

- `succeeded`: Refund completed
- `pending`: Refund being processed
- `failed`: Refund failed

**Error Responses:**

- `400 Bad Request`: Invalid request or payment status
- `401 Unauthorized`: Missing or invalid authentication
- `403 Forbidden`: Payment belongs to different user
- `404 Not Found`: Payment not found
- `500 Internal Server Error`: Refund creation failed

**Example cURL:**

```bash
# Full refund
curl -X POST https://api.servicepro.com/api/payments/refund \
  -H "Authorization: Bearer YOUR_JWT_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "payment_id": "550e8400-e29b-41d4-a716-446655440000",
    "reason": "requested_by_customer"
  }'

# Partial refund
curl -X POST https://api.servicepro.com/api/payments/refund \
  -H "Authorization: Bearer YOUR_JWT_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "payment_id": "550e8400-e29b-41d4-a716-446655440000",
    "amount": 25.50,
    "reason": "requested_by_customer"
  }'
```

---

### 5. Get Payment Receipt

Retrieves detailed receipt information for a payment.

**Endpoint:** `GET /api/payments/receipt/{payment_id}`

**Path Parameters:**

- `payment_id`: UUID of the payment

**Response (200 OK):**

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

**Response Fields:**

- `payment_id`: Payment UUID
- `amount`: Original payment amount
- `currency`: Payment currency
- `status`: Payment status
- `payment_method_type`: Type of payment method used
- `description`: Payment description
- `receipt_url`: Stripe-hosted receipt URL
- `receipt_email`: Email receipt was sent to
- `processed_at`: When payment was processed
- `created_at`: When payment was created
- `card`: Card details (if card payment)
- `refunded_amount`: Total amount refunded
- `net_amount`: Remaining amount after refunds

**Error Responses:**

- `400 Bad Request`: Invalid payment ID format
- `401 Unauthorized`: Missing or invalid authentication
- `403 Forbidden`: Payment belongs to different user
- `404 Not Found`: Payment not found

**Example cURL:**

```bash
curl -X GET https://api.servicepro.com/api/payments/receipt/550e8400-e29b-41d4-a716-446655440000 \
  -H "Authorization: Bearer YOUR_JWT_TOKEN"
```

---

## Request/Response Models

### Payment Status Enum

```go
type PaymentStatus string

const (
    PaymentStatusPending    PaymentStatus = "pending"
    PaymentStatusProcessing PaymentStatus = "processing"
    PaymentStatusSucceeded  PaymentStatus = "succeeded"
    PaymentStatusFailed     PaymentStatus = "failed"
    PaymentStatusCanceled   PaymentStatus = "canceled"
    PaymentStatusRefunded   PaymentStatus = "refunded"
)
```

### Payment Method Type Enum

```go
type PaymentMethodType string

const (
    PaymentMethodCard         PaymentMethodType = "card"
    PaymentMethodBankTransfer PaymentMethodType = "bank_transfer"
    PaymentMethodACH          PaymentMethodType = "ach"
    PaymentMethodWallet       PaymentMethodType = "wallet"
)
```

### Database Models

#### Payment

```go
type Payment struct {
    ID                    uuid.UUID
    UserID                uuid.UUID
    InvoiceID             *uuid.UUID
    OrderID               *uuid.UUID
    StripePaymentIntentID string
    StripeCustomerID      string
    StripeChargeID        string
    Amount                decimal.Decimal
    Currency              string
    Status                PaymentStatus
    PaymentMethodType     PaymentMethodType
    Description           string
    StatementDescriptor   string
    Metadata              map[string]string
    ErrorCode             string
    ErrorMessage          string
    ReceiptEmail          string
    ReceiptURL            string
    ProcessedAt           *time.Time
    CreatedAt             time.Time
    UpdatedAt             time.Time
}
```

#### PaymentRefund

```go
type PaymentRefund struct {
    ID             uuid.UUID
    PaymentID      uuid.UUID
    StripeRefundID string
    Amount         decimal.Decimal
    Currency       string
    Status         string
    Reason         string
    Metadata       map[string]string
    ProcessedAt    *time.Time
    CreatedAt      time.Time
    UpdatedAt      time.Time
}
```

---

## Error Handling

### Error Response Format

All errors return a consistent JSON structure:

```json
{
  "error": "Error description",
  "details": "Additional error details (optional)"
}
```

### HTTP Status Codes

- `200 OK`: Request successful
- `400 Bad Request`: Invalid request data or validation failure
- `401 Unauthorized`: Missing or invalid authentication
- `403 Forbidden`: Access denied (e.g., accessing another user's payment)
- `404 Not Found`: Resource not found
- `500 Internal Server Error`: Server-side error

### Common Error Scenarios

#### Invalid Amount

```json
{
  "error": "Validation failed",
  "details": "amount must be at least 0.50 USD"
}
```

#### Unsupported Currency

```json
{
  "error": "Validation failed",
  "details": "currency XYZ is not supported"
}
```

#### Payment Not Found

```json
{
  "error": "Payment not found"
}
```

#### Access Denied

```json
{
  "error": "Access denied"
}
```

#### Stripe API Error

```json
{
  "error": "Failed to create payment intent",
  "details": "Your card was declined."
}
```

---

## Testing

### Running Tests

```bash
# Run all payment tests
go test ./internal/api/handlers/... -v -run Payment
go test ./internal/api/validators/... -v

# Run specific test
go test ./internal/api/handlers/... -v -run TestCreatePaymentIntent

# Run with coverage
go test ./internal/api/handlers/... -cover
go test ./internal/api/validators/... -cover
```

### Test Coverage

The payment API has comprehensive test coverage:

- **Validators**: 100% coverage of all validation rules

  - Amount validation (min/max, decimal places)
  - Currency validation
  - Metadata validation
  - Email validation
  - URL validation

- **Handlers**: Full coverage of all endpoints
  - Success scenarios
  - Validation errors
  - Authentication errors
  - Not found errors
  - Access denied errors

### Mock Testing

All tests use mocks for external dependencies:

```go
// Mock Stripe Client
mockStripe := new(MockStripeClient)
mockStripe.On("CreatePaymentIntent", mock.Anything, mock.Anything).
    Return(&stripe.PaymentIntentResult{...}, nil)

// Mock Repositories
mockPaymentRepo := new(MockPaymentRepository)
mockPaymentRepo.On("CreatePayment", mock.Anything).Return(nil)
```

### Test Data

Use these test card numbers with Stripe:

- **Success**: `4242 4242 4242 4242`
- **Requires 3D Secure**: `4000 0027 6000 3184`
- **Declined**: `4000 0000 0000 0002`
- **Insufficient funds**: `4000 0000 0000 9995`

---

## Security Considerations

### PCI Compliance

- **Never** send raw card data to your server
- Use Stripe.js or Stripe Mobile SDKs to collect payment information
- Card data goes directly from client to Stripe
- Only use `client_secret` and `payment_method_id` on your server

### API Key Security

- Store Stripe keys in environment variables
- Never commit keys to version control
- Use separate keys for test and production
- Rotate keys if compromised

### Authentication

- All endpoints require valid JWT authentication
- Token must contain `user_id` claim
- Tokens should have reasonable expiration (1-24 hours)

### Access Control

- Users can only access their own payments
- Payments are automatically linked to authenticated user
- Access to other users' payments returns `403 Forbidden`

### Rate Limiting

- Stripe client has built-in rate limiting (100 req/sec default)
- Uses token bucket algorithm
- Configurable via `STRIPE_REQUESTS_PER_SECOND` environment variable

### Webhook Security

- Webhook requests must have valid Stripe signature
- Signature verified using webhook secret
- Timestamp validation prevents replay attacks
- Idempotency tracking prevents duplicate processing

### Metadata Sanitization

- All user input is sanitized before storage
- Control characters removed
- Length limits enforced
- Dangerous characters filtered

---

## Integration Guide

### Frontend Integration Flow

#### 1. Create Payment Intent

```javascript
// Client-side: Create payment intent
const response = await fetch('/api/payments/intent', {
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

#### 2. Collect Payment Method

```javascript
// Client-side: Use Stripe.js to collect payment
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
  // Handle error
  console.error(error.message);
} else if (paymentIntent.status === 'succeeded') {
  // Payment succeeded
  console.log('Payment successful!');
}
```

#### 3. Handle 3D Secure

```javascript
// Stripe.js automatically handles 3D Secure authentication
// If required, it will show authentication modal
// After authentication, payment will continue automatically
```

### Backend Integration

#### Setting up Routes

```go
import (
    "github.com/gin-gonic/gin"
    "github.com/javaknight1/servicepro/backend/internal/api/routes"
)

func main() {
    router := gin.Default()
    api := router.Group("/api")

    // Setup payment routes
    err := routes.SetupPaymentRoutes(api, db, jwtSecret)
    if err != nil {
        log.Fatal(err)
    }

    router.Run(":8080")
}
```

#### Environment Variables

```bash
# Required
STRIPE_SECRET_KEY=sk_test_...
STRIPE_PUBLISHABLE_KEY=pk_test_...
STRIPE_WEBHOOK_SECRET=whsec_...

# Optional
STRIPE_MAX_NETWORK_RETRIES=3
STRIPE_REQUESTS_PER_SECOND=100
STRIPE_WEBHOOK_TOLERANCE=300
```

#### Database Migration

Run the payment tables migration:

```sql
-- Create payments table
CREATE TABLE payments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id),
    invoice_id UUID REFERENCES invoices(id),
    order_id UUID REFERENCES orders(id),
    stripe_payment_intent_id VARCHAR(255) UNIQUE NOT NULL,
    stripe_customer_id VARCHAR(255),
    stripe_charge_id VARCHAR(255),
    amount DECIMAL(12,2) NOT NULL,
    currency VARCHAR(3) NOT NULL DEFAULT 'usd',
    status VARCHAR(20) NOT NULL DEFAULT 'pending',
    payment_method_type VARCHAR(50),
    description TEXT,
    statement_descriptor VARCHAR(100),
    metadata JSONB,
    error_code VARCHAR(100),
    error_message TEXT,
    receipt_email VARCHAR(255),
    receipt_url TEXT,
    processed_at TIMESTAMP,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    INDEX idx_user_id (user_id),
    INDEX idx_status (status),
    INDEX idx_stripe_payment_intent_id (stripe_payment_intent_id),
    INDEX idx_stripe_customer_id (stripe_customer_id)
);

-- Create payment_refunds table
CREATE TABLE payment_refunds (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    payment_id UUID NOT NULL REFERENCES payments(id),
    stripe_refund_id VARCHAR(255) UNIQUE NOT NULL,
    amount DECIMAL(12,2) NOT NULL,
    currency VARCHAR(3) NOT NULL,
    status VARCHAR(20) NOT NULL,
    reason VARCHAR(100),
    metadata JSONB,
    processed_at TIMESTAMP,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    INDEX idx_payment_id (payment_id),
    INDEX idx_stripe_refund_id (stripe_refund_id)
);
```

---

## Additional Resources

### Stripe Documentation

- [Payment Intents API](https://stripe.com/docs/api/payment_intents)
- [Payment Methods API](https://stripe.com/docs/api/payment_methods)
- [Refunds API](https://stripe.com/docs/api/refunds)
- [Testing](https://stripe.com/docs/testing)

### Related Documentation

- [Stripe Integration Guide](./STRIPE_INTEGRATION.md)
- [Webhook Processing](./STRIPE_INTEGRATION.md#webhook-handling)
- [Event Types](./STRIPE_INTEGRATION.md#event-types)

### Support

For questions or issues:

- Check existing tests in `internal/api/handlers/payment_handler_test.go`
- Review validation rules in `internal/api/validators/payment_validator.go`
- Consult Stripe documentation
- Contact the development team

---

## Changelog

### Version 1.0.0 (2025-01-15)

- Initial release
- Payment intent creation
- Payment confirmation with 3D Secure support
- Payment methods retrieval
- Full and partial refunds
- Payment receipts
- Comprehensive validation
- Full test coverage
