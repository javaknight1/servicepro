# Invoice System Documentation

Comprehensive PostgreSQL-based invoice system with advanced features including payment tracking, tax rates, audit trails, and automated calculations.

## Table of Contents

- [Overview](#overview)
- [Database Schema](#database-schema)
- [Features](#features)
- [Tables](#tables)
- [Triggers and Functions](#triggers-and-functions)
- [Usage Examples](#usage-examples)
- [API Models](#api-models)

## Overview

The invoice system provides:

- **Invoice Management**: Complete CRUD operations with status tracking
- **Line Items**: Flexible line items with automatic total calculations
- **Payment Tracking**: Support for partial payments and multiple payment methods
- **Tax Rates**: Configurable tax rates by region and type
- **Payment Terms**: Flexible payment terms with early payment discounts and late fees
- **Audit Trail**: Complete audit log of all invoice changes
- **Automated Calculations**: Database triggers handle all financial calculations
- **Validation**: Built-in validation for data integrity

## Database Schema

### Entity Relationship Diagram

```
┌──────────────────┐
│  tax_rates       │
└────────┬─────────┘
         │
         │
┌────────┴─────────┐       ┌──────────────────┐
│  payment_terms   │       │  users           │
└────────┬─────────┘       └────────┬─────────┘
         │                          │
         │                          │
         │         ┌────────────────┴─────────────────┐
         └─────────┤  invoices                        │
                   │  - Sequential invoice numbers    │
                   │  - Status tracking               │
                   │  - Automatic totals              │
                   └────────┬─────────────────────────┘
                            │
                ┌───────────┴───────────┐
                │                       │
        ┌───────┴────────┐      ┌──────┴────────────┐
        │ invoice_lines  │      │ invoice_payments  │
        │ - Auto totals  │      │ - Payment history │
        └────────────────┘      └───────────────────┘
                │
        ┌───────┴────────────┐
        │ invoice_audit_log  │
        │ - Change history   │
        └────────────────────┘
```

## Features

### 1. Invoice Status State Machine

```
draft → sent → viewed → paid
   ↓      ↓       ↓       ↑
   └──────┴───────┴────→ partially_paid
          ↓
       overdue
          ↓
      cancelled / refunded
```

**Status Descriptions:**

- `draft`: Invoice is being created, not yet sent
- `sent`: Invoice has been sent to customer
- `viewed`: Customer has viewed the invoice
- `paid`: Invoice has been fully paid
- `partially_paid`: Some payment received, balance remains
- `overdue`: Invoice is past due date and unpaid
- `cancelled`: Invoice has been cancelled
- `refunded`: Payment has been refunded to customer

### 2. Automated Invoice Numbering

Invoice numbers are automatically generated in the format: `INV-YYYY-#####`

Example: `INV-2024-00001`, `INV-2024-00002`, etc.

The system:

- Auto-increments by year
- Zero-pads to 5 digits
- Ensures uniqueness

### 3. Automatic Total Calculations

All financial calculations are handled by database triggers:

```
Line Total = (Quantity × Unit Price) - Discount Amount
Tax Amount = Line Total × Tax Rate (if taxable)
Line Total with Tax = Line Total + Tax Amount

Invoice Subtotal = SUM(all line totals)
Invoice Tax Amount = SUM(all line tax amounts)
Invoice Total = Subtotal + Tax Amount - Invoice Discount
Amount Due = Total Amount - Amount Paid
```

### 4. Payment Tracking

The system supports:

- **Full payments**: Invoice marked as 'paid'
- **Partial payments**: Track multiple payments over time
- **Payment methods**: Credit card, check, cash, bank transfer, etc.
- **Reference tracking**: Store transaction/check numbers

### 5. Comprehensive Audit Trail

Every change to an invoice is logged:

- Status changes
- Amount modifications
- Field updates
- User who made the change
- Timestamp of change
- IP address and user agent (if available)

## Tables

### invoices

Main invoices table with financial details and status tracking.

**Key Fields:**

```sql
id                UUID PRIMARY KEY
invoice_number    VARCHAR(50) UNIQUE     -- Auto-generated
customer_id       UUID NOT NULL          -- FK to users
status            invoice_status         -- Current status
issue_date        DATE                   -- When invoice was issued
due_date          DATE                   -- When payment is due
subtotal          DECIMAL(12,2)          -- Auto-calculated
tax_amount        DECIMAL(12,2)          -- Auto-calculated
total_amount      DECIMAL(12,2)          -- Auto-calculated
amount_paid       DECIMAL(12,2)          -- Updated by payments
amount_due        DECIMAL(12,2)          -- Computed column
```

**Constraints:**

- `due_date` must be >= `issue_date`
- `total_amount` = `subtotal` + `tax_amount` - `discount_amount`
- `amount_paid` ≤ `total_amount`

**Indexes:**

- `customer_id`, `status`, `issue_date`, `due_date`
- Composite: `(customer_id, status)`, `(status, due_date)`

### invoice_lines

Individual line items for each invoice.

**Key Fields:**

```sql
id                  UUID PRIMARY KEY
invoice_id          UUID NOT NULL      -- FK to invoices
description         TEXT NOT NULL
quantity            DECIMAL(10,2)
unit_price          DECIMAL(12,2)
discount_amount     DECIMAL(12,2)
taxable             BOOLEAN
tax_rate            DECIMAL(10,4)
tax_amount          DECIMAL(12,2)
line_total          DECIMAL(12,2)      -- Computed
line_total_with_tax DECIMAL(12,2)      -- Computed
sort_order          INTEGER            -- Display order
```

**Auto-Calculated Fields:**

- `line_total` = `(quantity × unit_price) - discount_amount`
- `line_total_with_tax` = `line_total + tax_amount`

### invoice_payments

Payment records for invoices (supports partial payments).

**Key Fields:**

```sql
id                UUID PRIMARY KEY
invoice_id        UUID NOT NULL      -- FK to invoices
amount            DECIMAL(12,2)
payment_date      DATE
payment_method    VARCHAR(50)        -- credit_card, check, cash, etc.
reference_number  VARCHAR(100)       -- Transaction/check number
```

### tax_rates

Configurable tax rates by region and type.

**Key Fields:**

```sql
id              UUID PRIMARY KEY
name            VARCHAR(100)
rate            DECIMAL(10,4)      -- 0.0825 for 8.25%
tax_type        tax_type           -- sales_tax, vat, gst, hst, exempt
region          VARCHAR(100)       -- State/province/country
is_compound     BOOLEAN            -- Compound tax calculation
effective_date  DATE
expiry_date     DATE
```

**Tax Types:**

- `sales_tax`: US sales tax
- `vat`: Value Added Tax (Europe)
- `gst`: Goods and Services Tax (Canada, Australia)
- `hst`: Harmonized Sales Tax (Canadian provinces)
- `exempt`: Tax exempt

### payment_terms

Payment term configurations with discounts and late fees.

**Key Fields:**

```sql
id                   UUID PRIMARY KEY
name                 VARCHAR(100)
term_type            payment_term_type
days_until_due       INTEGER
discount_percentage  DECIMAL(5,2)      -- Early payment discount
discount_days        INTEGER           -- Days to qualify for discount
late_fee_percentage  DECIMAL(5,2)      -- Late payment penalty
grace_period_days    INTEGER
is_default           BOOLEAN
```

**Payment Term Types:**

- `due_on_receipt`: Payment due immediately
- `net_7`, `net_10`, `net_15`, `net_30`, `net_60`, `net_90`: Standard net terms
- `custom`: Custom payment terms

**Example Payment Terms:**

- **"Net 30"**: Payment due in 30 days
- **"2/10 Net 30"**: 2% discount if paid within 10 days, otherwise due in 30 days

### invoice_audit_log

Complete audit trail of all invoice changes.

**Key Fields:**

```sql
id              UUID PRIMARY KEY
invoice_id      UUID NOT NULL
action          VARCHAR(50)         -- created, updated, status_changed, etc.
field_name      VARCHAR(100)        -- Which field changed
old_value       TEXT                -- Previous value
new_value       TEXT                -- New value
from_status     invoice_status      -- Previous status (for status changes)
to_status       invoice_status      -- New status (for status changes)
changed_by      UUID                -- User who made the change
changed_by_type VARCHAR(50)         -- user, customer, system
```

## Triggers and Functions

### 1. generate_invoice_number()

**Purpose**: Auto-generate sequential invoice numbers
**Trigger**: `BEFORE INSERT` on `invoices`

```sql
INV-2024-00001
INV-2024-00002
...
INV-2025-00001  -- Resets each year
```

### 2. recalculate_invoice_totals()

**Purpose**: Recalculate invoice totals when line items change
**Trigger**: `AFTER INSERT/UPDATE/DELETE` on `invoice_lines`

Automatically updates:

- `invoices.subtotal`
- `invoices.tax_amount`
- `invoices.total_amount`

### 3. update_invoice_amount_paid()

**Purpose**: Update `amount_paid` and status when payments are added
**Trigger**: `AFTER INSERT/DELETE` on `invoice_payments`

Automatically:

- Sums all payments for invoice
- Updates `amount_paid`
- Changes status to `paid` if fully paid
- Changes status to `partially_paid` if partial payment
- Sets `paid_date` when fully paid

### 4. audit_invoice_changes()

**Purpose**: Log all invoice changes to audit log
**Trigger**: `AFTER INSERT/UPDATE/DELETE` on `invoices`

Logs:

- Invoice creation
- Status changes
- Amount changes
- Field updates
- Deletions

### 5. update_overdue_invoices()

**Purpose**: Mark invoices as overdue if past due date
**Type**: Manual function (can be called via cron)

```sql
SELECT update_overdue_invoices();
```

Should be run daily to update overdue status.

### 6. validate_invoice_for_sending()

**Purpose**: Validate invoice before sending to customer
**Type**: Validation function

```sql
SELECT * FROM validate_invoice_for_sending('invoice-uuid-here');
```

Returns:

```json
{
  "is_valid": true,
  "errors": []
}
```

Validates:

- Has at least one line item
- Total amount > 0
- Customer exists
- Due date is valid

## Usage Examples

### Creating an Invoice

```sql
-- 1. Create the invoice
INSERT INTO invoices (
    customer_id,
    status,
    issue_date,
    due_date,
    payment_term_id,
    tax_rate_id,
    notes,
    created_by
) VALUES (
    'customer-uuid',
    'draft',
    CURRENT_DATE,
    CURRENT_DATE + INTERVAL '30 days',
    (SELECT id FROM payment_terms WHERE term_type = 'net_30' LIMIT 1),
    (SELECT id FROM tax_rates WHERE region = 'CA' LIMIT 1),
    'Website development project',
    'user-uuid'
) RETURNING id;

-- 2. Add line items (invoice totals calculate automatically)
INSERT INTO invoice_lines (invoice_id, description, quantity, unit_price, taxable, sort_order)
VALUES
    ('invoice-uuid', 'Website Design', 40, 125.00, true, 1),
    ('invoice-uuid', 'Development', 60, 150.00, true, 2),
    ('invoice-uuid', 'Testing', 20, 100.00, true, 3);

-- 3. Validate before sending
SELECT * FROM validate_invoice_for_sending('invoice-uuid');

-- 4. Send invoice (update status)
UPDATE invoices
SET status = 'sent', sent_date = NOW()
WHERE id = 'invoice-uuid';
```

### Recording a Payment

```sql
-- Full payment
INSERT INTO invoice_payments (
    invoice_id,
    amount,
    payment_date,
    payment_method,
    reference_number,
    created_by
) VALUES (
    'invoice-uuid',
    5000.00,
    CURRENT_DATE,
    'bank_transfer',
    'TXN-2024-12345',
    'user-uuid'
);

-- The trigger automatically updates:
-- - invoices.amount_paid
-- - invoices.status (to 'paid')
-- - invoices.paid_date
```

### Partial Payments

```sql
-- First payment
INSERT INTO invoice_payments (invoice_id, amount, payment_date, payment_method, created_by)
VALUES ('invoice-uuid', 2000.00, CURRENT_DATE, 'credit_card', 'user-uuid');

-- Status automatically changes to 'partially_paid'

-- Second payment
INSERT INTO invoice_payments (invoice_id, amount, payment_date, payment_method, created_by)
VALUES ('invoice-uuid', 3000.00, CURRENT_DATE + 10, 'check', 'user-uuid');

-- Status automatically changes to 'paid' when total payments >= total amount
```

### Querying Invoices

```sql
-- Find all overdue invoices
SELECT * FROM invoice_summary
WHERE is_overdue = true
ORDER BY days_overdue DESC;

-- Revenue by month
SELECT * FROM revenue_by_month
ORDER BY month DESC
LIMIT 12;

-- Customer's invoice history
SELECT
    i.invoice_number,
    i.status,
    i.issue_date,
    i.total_amount,
    i.amount_paid,
    i.amount_due
FROM invoices i
WHERE i.customer_id = 'customer-uuid'
AND i.deleted_at IS NULL
ORDER BY i.issue_date DESC;

-- Invoice details with line items
SELECT
    i.invoice_number,
    i.status,
    i.total_amount,
    il.description,
    il.quantity,
    il.unit_price,
    il.line_total
FROM invoices i
JOIN invoice_lines il ON i.id = il.invoice_id
WHERE i.invoice_number = 'INV-2024-00001'
ORDER BY il.sort_order;

-- Payment history
SELECT
    ip.payment_date,
    ip.amount,
    ip.payment_method,
    ip.reference_number
FROM invoice_payments ip
JOIN invoices i ON ip.invoice_id = i.id
WHERE i.invoice_number = 'INV-2024-00001'
ORDER BY ip.payment_date;

-- Audit log for an invoice
SELECT
    ial.action,
    ial.from_status,
    ial.to_status,
    ial.field_name,
    ial.old_value,
    ial.new_value,
    ial.created_at
FROM invoice_audit_log ial
JOIN invoices i ON ial.invoice_id = i.id
WHERE i.invoice_number = 'INV-2024-00001'
ORDER BY ial.created_at DESC;
```

### Updating Invoice Status

```sql
-- Mark as sent
UPDATE invoices
SET status = 'sent', sent_date = NOW(), updated_by = 'user-uuid'
WHERE id = 'invoice-uuid';

-- Mark as viewed (typically done programmatically when customer views)
UPDATE invoices
SET status = 'viewed', viewed_date = NOW()
WHERE id = 'invoice-uuid';

-- Cancel invoice
UPDATE invoices
SET status = 'cancelled', updated_by = 'user-uuid'
WHERE id = 'invoice-uuid';

-- All status changes are logged in invoice_audit_log automatically
```

### Running Maintenance

```sql
-- Update overdue invoices (run daily)
SELECT update_overdue_invoices();

-- Clean up old audit logs (optional, run monthly)
DELETE FROM invoice_audit_log
WHERE created_at < NOW() - INTERVAL '2 years';
```

## API Models

### Go Structs (GORM)

See `backend/internal/models/invoice.go` for complete Go structs with GORM tags.

**Key Models:**

- `Invoice`: Main invoice model
- `InvoiceLine`: Line item model
- `InvoicePayment`: Payment record model
- `TaxRate`: Tax rate configuration
- `PaymentTerm`: Payment terms configuration
- `InvoiceAuditLog`: Audit log entry
- `InvoiceSummary`: View for invoice summaries
- `RevenueByMonth`: View for revenue statistics

### Filter and Response Types

```go
// Filter for listing invoices
type InvoiceFilter struct {
    CustomerID  *uuid.UUID
    Status      *InvoiceStatus
    FromDate    *time.Time
    ToDate      *time.Time
    MinAmount   *decimal.Decimal
    MaxAmount   *decimal.Decimal
    IsOverdue   *bool
    Search      string
    Page        int
    PageSize    int
    SortBy      string
    SortOrder   string
}

// Response for listing invoices
type InvoiceListResponse struct {
    Invoices   []Invoice
    Total      int64
    Page       int
    PageSize   int
    TotalPages int
}
```

## Best Practices

### 1. Invoice Creation Workflow

```
1. Create invoice in 'draft' status
2. Add all line items
3. Review calculated totals
4. Validate invoice
5. Change status to 'sent'
6. Send to customer (email, etc.)
```

### 2. Payment Processing

```
1. Receive payment from customer
2. Create invoice_payment record
3. System automatically:
   - Updates amount_paid
   - Updates invoice status
   - Sets paid_date if fully paid
4. Send payment confirmation
```

### 3. Overdue Management

```
1. Run update_overdue_invoices() daily (via cron)
2. Query for overdue invoices
3. Send reminder emails
4. Apply late fees if configured
```

### 4. Data Integrity

- Never manually update `subtotal`, `tax_amount`, `total_amount` - let triggers handle it
- Never manually update `amount_paid` - use `invoice_payments` table
- Always use `deleted_at` for soft deletes, never hard delete
- Use transactions for multi-step operations

### 5. Performance

- Use indexed fields in WHERE clauses
- Use views (`invoice_summary`, `revenue_by_month`) for reporting
- Consider partitioning `invoice_audit_log` if it grows large
- Add indexes for custom query patterns

## Security Considerations

1. **Access Control**: Implement row-level security or application-level checks
2. **Audit Trail**: Never delete audit log entries
3. **Sensitive Data**: PO numbers and reference numbers may contain sensitive info
4. **Payment Data**: Never store credit card numbers in this system
5. **Soft Deletes**: Use `deleted_at` to maintain audit trail

## Migration and Deployment

1. Run migration `005_create_invoice_system.sql` first
2. Run sample data migration `006_invoice_sample_data.sql` (optional, for testing)
3. Verify all triggers are created successfully
4. Test with sample data before production use
5. Set up cron job for `update_overdue_invoices()`

## License

MIT
