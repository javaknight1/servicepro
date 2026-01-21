## Payment Terms System Documentation

Complete payment terms management system with business logic for due date calculations, early payment discounts, late fees, and automated payment notifications.

## Table of Contents

- [Overview](#overview)
- [Features](#features)
- [Database Schema](#database-schema)
- [Business Logic](#business-logic)
- [API Endpoints](#api-endpoints)
- [Notification System](#notification-system)
- [Usage Examples](#usage-examples)
- [Testing](#testing)
- [Configuration](#configuration)

## Overview

The Payment Terms System provides comprehensive management of payment terms, including:

- **Payment term configuration** with flexible options
- **Automatic calculations** for due dates, discounts, and late fees
- **Timezone support** for global operations
- **Automated notifications** for payment events
- **Grace periods** for late payments
- **Early payment discounts** to incentivize prompt payment

## Features

### ✅ Payment Terms Management

- Create and manage payment terms
- Set default payment terms
- Configure discount periods and percentages
- Configure late fees (percentage + fixed amount)
- Set grace periods for overdue payments

### ✅ Business Logic

- **Due Date Calculation** - Automatic calculation based on payment terms
- **Early Payment Discounts** - Calculate discounts for early payments
- **Late Fee Calculation** - Automatic late fee calculation after grace period
- **Payment Status** - Real-time payment status with recommendations
- **Timezone Support** - Full timezone support for international operations

### ✅ Notification System

- Automated notifications for payment events
- Multiple notification channels (email, SMS, push, webhook)
- Configurable notification rules
- Retry logic for failed notifications
- Notification history tracking

### ✅ Notification Types

- Payment due soon (configurable days before due)
- Payment overdue
- Payment in grace period
- Early payment discount available
- Late fee applied
- Payment received (full/partial)

## Database Schema

### Payment Terms Table

```sql
CREATE TABLE payment_terms (
    id UUID PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    description TEXT,
    term_type payment_term_type NOT NULL,
    days_until_due INTEGER CHECK (days_until_due >= 0),
    discount_percentage DECIMAL(5, 2),
    discount_days INTEGER,
    late_fee_percentage DECIMAL(5, 2),
    late_fee_amount DECIMAL(10, 2),
    grace_period_days INTEGER DEFAULT 0,
    is_default BOOLEAN DEFAULT FALSE,
    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);
```

### Payment Notifications Table

```sql
CREATE TABLE payment_notifications (
    id UUID PRIMARY KEY,
    invoice_id UUID NOT NULL,
    notification_type notification_type NOT NULL,
    status notification_status NOT NULL DEFAULT 'pending',
    channel notification_channel NOT NULL DEFAULT 'email',
    recipient_id UUID,
    recipient_email VARCHAR(255),
    recipient_phone VARCHAR(50),
    subject VARCHAR(500),
    message TEXT NOT NULL,
    data JSONB,
    scheduled_at TIMESTAMP NOT NULL,
    sent_at TIMESTAMP,
    retry_count INTEGER DEFAULT 0,
    max_retries INTEGER DEFAULT 3,
    last_error TEXT,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);
```

### Payment Notification Rules Table

```sql
CREATE TABLE payment_notification_rules (
    id UUID PRIMARY KEY,
    name VARCHAR(200) NOT NULL,
    description TEXT,
    notification_type notification_type NOT NULL,
    days_before_due INTEGER,
    days_after_due INTEGER,
    is_active BOOLEAN DEFAULT TRUE,
    channels notification_channel[] NOT NULL,
    email_template_id UUID,
    sms_template_id UUID,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);
```

## Business Logic

### Due Date Calculation

Calculates the due date based on issue date and payment terms:

```go
dueDate = issueDate + daysUntilDue
```

**Features:**

- Timezone-aware calculations
- Configurable number of days
- Automatic handling of month/year boundaries

### Early Payment Discount

Calculates discount if payment is made within the discount period:

```go
if paymentDate <= (issueDate + discountDays) {
    discount = invoiceAmount * (discountPercentage / 100)
}
```

**Features:**

- Configurable discount percentage
- Configurable discount period
- Automatic discount expiration

### Late Fee Calculation

Calculates late fees after grace period expires:

```go
if currentDate > (dueDate + gracePeriodDays) {
    lateFee = (invoiceAmount * lateFeePercentage / 100) + lateFeeAmount
}
```

**Features:**

- Percentage-based fee
- Fixed amount fee
- Combined percentage + fixed
- Grace period support

### Payment Status

Determines the current payment status:

- **early** - Within discount period
- **on_time** - After discount period, before due date
- **grace_period** - After due date, within grace period
- **overdue** - After grace period ends

## API Endpoints

### Payment Terms CRUD

#### List Payment Terms

```http
GET /api/v1/payment-terms?active_only=true
Authorization: Bearer <token>
```

**Response:**

```json
[
  {
    "id": "uuid",
    "name": "Net 30",
    "description": "Payment due in 30 days",
    "term_type": "net_30",
    "days_until_due": 30,
    "discount_percentage": "2.00",
    "discount_days": 10,
    "late_fee_percentage": "5.00",
    "late_fee_amount": "25.00",
    "grace_period_days": 3,
    "is_default": true,
    "is_active": true
  }
]
```

#### Get Payment Term

```http
GET /api/v1/payment-terms/:id
Authorization: Bearer <token>
```

#### Get Default Payment Term

```http
GET /api/v1/payment-terms/default
Authorization: Bearer <token>
```

#### Create Payment Term

```http
POST /api/v1/payment-terms
Authorization: Bearer <token>
Content-Type: application/json

{
  "name": "Net 30 with 2/10",
  "description": "2% discount if paid within 10 days, otherwise net 30",
  "term_type": "net_30",
  "days_until_due": 30,
  "discount_percentage": 2.0,
  "discount_days": 10,
  "late_fee_percentage": 5.0,
  "late_fee_amount": 25.00,
  "grace_period_days": 3,
  "is_default": false,
  "is_active": true
}
```

#### Update Payment Term

```http
PUT /api/v1/payment-terms/:id
Authorization: Bearer <token>
Content-Type: application/json

{
  "discount_percentage": 3.0,
  "discount_days": 15
}
```

#### Delete Payment Term

```http
DELETE /api/v1/payment-terms/:id
Authorization: Bearer <token>
```

### Calculation Endpoints

#### Calculate Due Date

```http
POST /api/v1/payment-terms/calculate/due-date
Authorization: Bearer <token>
Content-Type: application/json

{
  "issue_date": "2024-01-15T00:00:00Z",
  "payment_term_id": "uuid",
  "timezone": "America/New_York"
}
```

**Response:**

```json
{
  "due_date": "2024-02-14T00:00:00-05:00",
  "issue_date": "2024-01-15T00:00:00Z",
  "timezone": "America/New_York"
}
```

#### Calculate Early Payment Discount

```http
POST /api/v1/payment-terms/calculate/discount
Authorization: Bearer <token>
Content-Type: application/json

{
  "invoice_amount": "1000.00",
  "payment_date": "2024-01-20T00:00:00Z",
  "issue_date": "2024-01-15T00:00:00Z",
  "payment_term_id": "uuid"
}
```

**Response:**

```json
{
  "discount_amount": "20.00",
  "eligible": true,
  "invoice_amount": "1000.00",
  "amount_after_discount": "980.00"
}
```

#### Calculate Late Fee

```http
POST /api/v1/payment-terms/calculate/late-fee
Authorization: Bearer <token>
Content-Type: application/json

{
  "invoice_amount": "1000.00",
  "current_date": "2024-02-20T00:00:00Z",
  "due_date": "2024-02-14T00:00:00Z",
  "payment_term_id": "uuid",
  "timezone": "UTC"
}
```

**Response:**

```json
{
  "late_fee_amount": "75.00",
  "applicable": true,
  "invoice_amount": "1000.00",
  "total_amount": "1075.00"
}
```

#### Get Payment Status

```http
POST /api/v1/payment-terms/calculate/status
Authorization: Bearer <token>
Content-Type: application/json

{
  "invoice_amount": "1000.00",
  "issue_date": "2024-01-15T00:00:00Z",
  "due_date": "2024-02-14T00:00:00Z",
  "payment_term_id": "uuid",
  "current_date": "2024-02-16T00:00:00Z",
  "timezone": "UTC"
}
```

**Response:**

```json
{
  "status": "grace_period",
  "due_date": "2024-02-14T00:00:00Z",
  "days_until_due": 0,
  "days_overdue": 2,
  "discount_eligible": false,
  "late_fee_applied": false,
  "in_grace_period": true,
  "grace_period_ends": "2024-02-17T00:00:00Z",
  "recommended_action": "Grace period ends 2024-02-17",
  "notification_required": true,
  "notification_type": "payment_in_grace_period"
}
```

#### Calculate Payment Details

```http
POST /api/v1/payment-terms/calculate/details
Authorization: Bearer <token>
Content-Type: application/json

{
  "invoice_amount": "1000.00",
  "issue_date": "2024-01-15T00:00:00Z",
  "payment_term_id": "uuid",
  "payment_date": "2024-01-20T00:00:00Z",
  "timezone": "UTC"
}
```

**Response:**

```json
{
  "due_date": "2024-02-14T00:00:00Z",
  "discount_available_until": "2024-01-25T00:00:00Z",
  "discount_amount": "20.00",
  "late_fee_amount": "0.00",
  "total_due": "980.00",
  "is_early_payment_eligible": true,
  "is_overdue": false,
  "days_overdue": 0
}
```

## Notification System

### Automated Triggers

The system automatically creates notifications based on database triggers:

1. **Payment Due Soon** - Triggered when invoice enters notification window (e.g., 7 days before due)
2. **Payment Overdue** - Triggered when invoice status changes to overdue
3. **Early Payment Discount** - Triggered when invoice is sent with early payment terms
4. **Payment Received** - Triggered when payment is recorded

### Notification Rules

Default notification rules are created during migration:

- **7 days before due** - Email notification
- **3 days before due** - Email + SMS notification
- **1 day before due** - Email + SMS + Push notification
- **Overdue** - Email + SMS notification

### Manual Notification Processing

```go
// Process pending notifications
processor := services.NewNotificationProcessor(db, emailSender, smsSender)
err := processor.ProcessBatch(ctx, 100)
```

### Check All Invoices

```go
// Manually trigger notification checks for all invoices
notificationService := services.NewPaymentNotificationService(db)
count, err := notificationService.CheckAllInvoicesForNotifications(ctx)
fmt.Printf("Processed %d invoices\n", count)
```

## Usage Examples

### Example 1: Standard Net 30 Terms

```go
// Create Net 30 payment term
term := &models.PaymentTerm{
    Name:            "Net 30",
    TermType:        models.PaymentTermNet30,
    DaysUntilDue:    30,
    GracePeriodDays: 3,
    IsDefault:       true,
    IsActive:        true,
}

err := paymentTermsService.CreatePaymentTerm(ctx, term)
```

### Example 2: 2/10 Net 30 (Early Payment Discount)

```go
// Create 2/10 Net 30 payment term
term := &models.PaymentTerm{
    Name:               "2/10 Net 30",
    TermType:           models.PaymentTermNet30,
    DaysUntilDue:       30,
    DiscountPercentage: decimal.NewFromFloat(2.0),
    DiscountDays:       10,
    GracePeriodDays:    3,
    IsActive:           true,
}

err := paymentTermsService.CreatePaymentTerm(ctx, term)
```

### Example 3: Calculate Payment for Invoice

```go
// Calculate comprehensive payment details
issueDate := time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC)
invoiceAmount := decimal.NewFromFloat(1000.00)
paymentDate := time.Date(2024, 1, 20, 0, 0, 0, 0, time.UTC)

details, err := paymentTermsService.CalculatePaymentDetails(
    ctx,
    invoiceAmount,
    issueDate,
    term.ID,
    &paymentDate,
    "America/New_York",
)

if details.IsEarlyPaymentEligible {
    fmt.Printf("Pay %s (save %s with early payment discount)\n",
        details.TotalDue, details.DiscountAmount)
} else if details.IsOverdue {
    fmt.Printf("Overdue by %d days. Total with late fee: %s\n",
        details.DaysOverdue, details.TotalDue)
}
```

### Example 4: Check Payment Status

```go
// Get current payment status
status, err := paymentTermsService.GetPaymentStatus(
    ctx,
    invoiceAmount,
    issueDate,
    dueDate,
    term.ID,
    time.Now(),
    "UTC",
)

if status.NotificationRequired {
    fmt.Printf("Notification: %s\n", status.NotificationType)
    fmt.Printf("Action: %s\n", status.RecommendedAction)
}
```

## Testing

### Run All Payment Terms Tests

```bash
go test ./internal/services -v -run TestPaymentTermsService
```

### Test Coverage

The test suite includes comprehensive coverage of:

- ✅ Payment term creation and validation
- ✅ Due date calculation with multiple timezones
- ✅ Early payment discount calculation
- ✅ Late fee calculation with grace periods
- ✅ Payment status determination
- ✅ Comprehensive payment details calculation
- ✅ Timezone handling across 6+ timezones
- ✅ CRUD operations
- ✅ Default payment term management

### Example Test

```go
func TestPaymentTermsService_CalculateEarlyPaymentDiscount(t *testing.T) {
    // Setup
    db := setupPaymentTermsTestDB(t)
    service := NewPaymentTermsService(db)
    ctx := context.Background()

    term := createTestPaymentTerm(t, db, 30, 2.0, 10)

    issueDate := time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC)
    paymentDate := issueDate.AddDate(0, 0, 5) // Pay on day 5
    invoiceAmount := decimal.NewFromFloat(1000.00)

    // Test
    discount, eligible, err := service.CalculateEarlyPaymentDiscount(
        ctx, invoiceAmount, paymentDate, issueDate, term.ID,
    )

    // Assert
    assert.NoError(t, err)
    assert.Equal(t, "20.00", discount.String())
    assert.True(t, eligible)
}
```

## Configuration

### Environment Variables

```bash
# Database
DB_HOST=localhost
DB_PORT=5432
DB_NAME=servicepro
DB_USER=postgres
DB_PASSWORD=password

# JWT
JWT_SECRET=your-secret-key

# Timezone (default)
DEFAULT_TIMEZONE=UTC

# Notifications
NOTIFICATION_BATCH_SIZE=100
NOTIFICATION_RETRY_MAX=3
NOTIFICATION_RETRY_DELAY=3600  # seconds
```

### Notification Configuration

Customize notification rules in the database:

```sql
-- Update notification rule
UPDATE payment_notification_rules
SET days_before_due = 5,
    channels = ARRAY['email', 'sms', 'push']::notification_channel[]
WHERE notification_type = 'payment_due_soon'
  AND name = 'Payment Due in 7 Days';

-- Add custom notification rule
INSERT INTO payment_notification_rules (name, description, notification_type, days_before_due, channels)
VALUES (
    'VIP Customer Reminder',
    'Special reminder for VIP customers',
    'payment_due_soon',
    14,
    ARRAY['email', 'sms', 'push']::notification_channel[]
);
```

## Integration

### Setup Routes

```go
import (
    "github.com/gin-gonic/gin"
    "github.com/javaknight1/servicepro/backend/internal/api/routes"
)

func main() {
    router := gin.Default()

    // API v1 group
    v1 := router.Group("/api/v1")

    // Setup payment terms routes
    routes.SetupPaymentTermsRoutes(v1, db, jwtSecret)

    router.Run(":8080")
}
```

### Run Migrations

```bash
# Apply payment notifications migration
migrate -path migrations -database "postgresql://user:pass@localhost:5432/dbname" up
```

### Background Job for Notifications

Set up a cron job or background worker to process notifications:

```go
func NotificationWorker(db *gorm.DB, emailSender EmailSender, smsSender SMSSender) {
    processor := services.NewNotificationProcessor(db, emailSender, smsSender)

    // Run every minute
    ticker := time.NewTicker(time.Minute)
    defer ticker.Stop()

    for range ticker.C {
        ctx := context.Background()
        if err := processor.ProcessBatch(ctx, 100); err != nil {
            log.Printf("Error processing notifications: %v", err)
        }
    }
}
```

## Best Practices

1. **Always use timezone-aware calculations** - Specify timezone for all date calculations
2. **Set appropriate grace periods** - Give customers time to make payments
3. **Configure reasonable late fees** - Balance revenue with customer relations
4. **Use early payment discounts** - Incentivize prompt payment
5. **Monitor notification delivery** - Check notification history regularly
6. **Test with different timezones** - Ensure calculations work globally
7. **Set realistic due date terms** - Consider your business needs and customer expectations

## Troubleshooting

### Notifications not sending

1. Check notification rules are active
2. Verify notification processor is running
3. Check notification history for errors
4. Verify email/SMS senders are configured

### Incorrect calculations

1. Verify timezone is correctly set
2. Check payment term configuration
3. Ensure dates are in correct format
4. Review grace period settings

### Late fees not applying

1. Check if grace period has expired
2. Verify late fee configuration
3. Ensure invoice is actually overdue
4. Check if late fees are enabled

## API Reference

All API endpoints support:

- JWT authentication via Bearer token
- JSON request/response format
- Timezone-aware calculations
- Comprehensive error messages
- Input validation

## Files Created

- `internal/services/payment_terms_service.go` (580 lines)
- `internal/services/payment_notification_service.go` (460 lines)
- `internal/services/payment_terms_service_test.go` (520 lines)
- `internal/api/handlers/payment_terms_handler.go` (560 lines)
- `internal/api/routes/payment_terms_routes.go` (40 lines)
- `migrations/007_create_payment_notifications.sql` (500+ lines)
- `docs/PAYMENT_TERMS_SYSTEM.md` (this file)

## License

MIT License
