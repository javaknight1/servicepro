# Invoice System - Complete Implementation

A production-ready PostgreSQL-based invoice system with advanced features including automated calculations, payment tracking, tax management, and comprehensive audit trails.

## 📋 Table of Contents

- [Features](#features)
- [Files Created](#files-created)
- [Quick Start](#quick-start)
- [Architecture](#architecture)
- [Database Schema](#database-schema)
- [Usage Examples](#usage-examples)
- [API Integration](#api-integration)

## ✨ Features

### Core Functionality

- ✅ **Invoice CRUD Operations** - Complete create, read, update, delete
- ✅ **Automatic Invoice Numbering** - Sequential numbering with year prefix (INV-2024-00001)
- ✅ **Status Tracking** - 8 status states with state machine validation
- ✅ **Line Items** - Flexible line items with automatic calculations
- ✅ **Payment Tracking** - Support for full and partial payments
- ✅ **Tax Management** - Configurable tax rates by region and type
- ✅ **Payment Terms** - Flexible terms with early payment discounts and late fees

### Advanced Features

- ✅ **Automated Calculations** - All financial calculations handled by database triggers
- ✅ **Audit Trail** - Complete history of all invoice changes
- ✅ **Soft Deletes** - Maintain data integrity with soft delete functionality
- ✅ **Validation Functions** - Built-in validation before sending invoices
- ✅ **Overdue Detection** - Automatic overdue status updates
- ✅ **Revenue Views** - Pre-built views for reporting and analytics
- ✅ **Payment History** - Track multiple payments per invoice

### Data Integrity

- ✅ **Constraints** - Database-level validation rules
- ✅ **Triggers** - Automatic total recalculation
- ✅ **Indexes** - Optimized for common query patterns
- ✅ **Computed Columns** - Generated columns for derived values
- ✅ **Foreign Keys** - Referential integrity enforcement

## 📁 Files Created

### Migration Files

**`migrations/005_create_invoice_system.sql`** (588 lines)

- Complete database schema
- All tables, indexes, and constraints
- Triggers for automation
- Functions for validation and calculations
- Views for reporting
- Comprehensive comments

**`migrations/006_invoice_sample_data.sql`** (395 lines)

- Sample tax rates (US, Canada, Europe)
- Sample payment terms (Net 7/15/30/60/90, custom terms)
- Test data generation function
- Helpful query examples

### Model Files

**`internal/models/invoice.go`** (348 lines)

- Complete Go structs with GORM tags
- All enums and types
- Filter and response types
- Validation result types
- View models

### Documentation

**`INVOICE_SYSTEM.md`** (900+ lines)

- Complete system documentation
- Architecture overview
- Detailed table descriptions
- Usage examples
- Best practices
- Security considerations

**`docs/INVOICE_SQL_REFERENCE.md`** (600+ lines)

- Quick reference for common SQL operations
- CRUD examples
- Payment operations
- Reporting queries
- Maintenance procedures

**`docs/INVOICE_README.md`** (This file)

- System overview
- Quick start guide
- Architecture summary

## 🚀 Quick Start

### 1. Run Migrations

```bash
# Navigate to backend directory
cd backend

# Run the migrations (using your migration tool)
psql -d servicepro -f migrations/005_create_invoice_system.sql
psql -d servicepro -f migrations/006_invoice_sample_data.sql
```

### 2. Verify Installation

```sql
-- Check tables were created
SELECT tablename FROM pg_tables
WHERE schemaname = 'public'
AND tablename LIKE '%invoice%';

-- Should see:
-- invoices
-- invoice_lines
-- invoice_payments
-- invoice_audit_log
-- payment_terms
-- tax_rates

-- Check triggers were created
SELECT trigger_name, event_manipulation, event_object_table
FROM information_schema.triggers
WHERE trigger_name LIKE '%invoice%';
```

### 3. Create Your First Invoice

```sql
-- 1. Create invoice
INSERT INTO invoices (
    customer_id, status, issue_date, due_date,
    payment_term_id, tax_rate_id, notes, created_by
) VALUES (
    'customer-uuid',
    'draft',
    CURRENT_DATE,
    CURRENT_DATE + INTERVAL '30 days',
    (SELECT id FROM payment_terms WHERE term_type = 'net_30' LIMIT 1),
    (SELECT id FROM tax_rates WHERE is_active = true LIMIT 1),
    'First test invoice',
    'user-uuid'
) RETURNING id, invoice_number;

-- 2. Add line items
INSERT INTO invoice_lines (invoice_id, description, quantity, unit_price, taxable, sort_order)
VALUES
    ('invoice-id-from-above', 'Consulting Services', 10, 150.00, true, 1),
    ('invoice-id-from-above', 'Software License', 1, 500.00, true, 2);

-- 3. Check calculated totals
SELECT invoice_number, subtotal, tax_amount, total_amount
FROM invoices
WHERE id = 'invoice-id-from-above';
```

### 4. Record Payment

```sql
-- Record a full payment
INSERT INTO invoice_payments (
    invoice_id, amount, payment_date,
    payment_method, reference_number, created_by
) VALUES (
    'invoice-id',
    1725.00,  -- Total from step 3
    CURRENT_DATE,
    'bank_transfer',
    'TXN-001',
    'user-uuid'
);

-- Check invoice status (should be 'paid' now)
SELECT invoice_number, status, amount_paid, amount_due
FROM invoices
WHERE id = 'invoice-id';
```

## 🏗️ Architecture

### Database Schema Overview

```
┌─────────────────────────────────────────────────────────┐
│                     Invoice System                      │
└─────────────────────────────────────────────────────────┘
                            │
        ┌───────────────────┼───────────────────┐
        │                   │                   │
┌───────▼────────┐  ┌──────▼───────┐  ┌───────▼────────┐
│  tax_rates     │  │payment_terms │  │   users        │
│  - Region-based│  │  - Net terms │  │  - Customers   │
│  - Sales tax   │  │  - Discounts │  │  - Admin users │
│  - VAT/GST     │  │  - Late fees │  └────────────────┘
└────────────────┘  └──────────────┘           │
                                               │
                    ┌──────────────────────────▼──────┐
                    │       invoices                  │
                    │  - Auto number generation       │
                    │  - Status state machine         │
                    │  - Automatic calculations       │
                    │  - Soft delete support          │
                    └──────────┬──────────────────────┘
                               │
                 ┌─────────────┼─────────────┐
                 │             │             │
         ┌───────▼──────┐ ┌───▼─────────┐ ┌─▼─────────────┐
         │invoice_lines │ │invoice_     │ │invoice_audit_ │
         │- Auto totals │ │payments     │ │log            │
         │- Tax calc    │ │- Partial OK │ │- Full history │
         └──────────────┘ └─────────────┘ └───────────────┘
```

### Key Design Principles

1. **Database-Driven Logic**: All calculations and validations happen in the database
2. **Audit First**: Everything is logged automatically
3. **Immutable History**: Audit logs are never deleted
4. **Soft Deletes**: Invoices are never hard-deleted
5. **Computed Columns**: Use generated columns for derived values
6. **Trigger Automation**: Triggers handle state transitions and calculations

### Status State Machine

```
        ┌─────────┐
        │  draft  │ (Initial state)
        └────┬────┘
             │
        ┌────▼────┐
        │  sent   │ (Invoice sent to customer)
        └────┬────┘
             │
        ┌────▼─────┐
        │  viewed  │ (Customer viewed invoice)
        └────┬─────┘
             │
    ┌────────┼────────┐
    │                 │
┌───▼──────────┐  ┌──▼───┐
│partially_paid│  │ paid │ (Final state)
└───┬──────────┘  └──────┘
    │
┌───▼────┐
│overdue │ (Past due date)
└───┬────┘
    │
┌───▼────────┐
│cancelled / │ (Terminal states)
│ refunded   │
└────────────┘
```

## 💾 Database Schema

### Main Tables

#### invoices

- Primary invoice table
- Auto-generates invoice numbers
- Tracks status through state machine
- Stores financial totals (auto-calculated)
- Soft delete support

#### invoice_lines

- Individual line items
- Auto-calculates totals with triggers
- Supports discounts and tax
- Sort order for display

#### invoice_payments

- Payment history
- Supports partial payments
- Auto-updates invoice status
- Multiple payment methods

#### tax_rates

- Configurable tax rates
- Region-based (state, province, country)
- Multiple tax types (sales tax, VAT, GST, HST)
- Effective and expiry dates

#### payment_terms

- Configurable payment terms
- Early payment discounts
- Late fee configuration
- Grace periods

#### invoice_audit_log

- Complete change history
- Status transitions
- Field-level changes
- User tracking

### Views

#### invoice_summary

Pre-aggregated invoice data with:

- Line item count
- Overdue status
- Days overdue calculation

#### revenue_by_month

Monthly revenue statistics:

- Invoice count
- Total revenue
- Total paid
- Outstanding balance

## 📝 Usage Examples

### Complete Invoice Workflow

```sql
-- 1. Create invoice
WITH new_invoice AS (
    INSERT INTO invoices (customer_id, status, issue_date, due_date, payment_term_id, tax_rate_id, created_by)
    VALUES ('customer-uuid', 'draft', CURRENT_DATE, CURRENT_DATE + 30,
            (SELECT id FROM payment_terms WHERE term_type = 'net_30' LIMIT 1),
            (SELECT id FROM tax_rates WHERE region = 'CA' LIMIT 1),
            'user-uuid')
    RETURNING id, invoice_number
)
-- 2. Add line items
INSERT INTO invoice_lines (invoice_id, description, quantity, unit_price, taxable, sort_order)
SELECT id, 'Service 1', 10, 100.00, true, 1 FROM new_invoice UNION ALL
SELECT id, 'Service 2', 5, 200.00, true, 2 FROM new_invoice;

-- 3. Validate
SELECT * FROM validate_invoice_for_sending(
    (SELECT id FROM new_invoice)
);

-- 4. Send
UPDATE invoices
SET status = 'sent', sent_date = NOW()
WHERE id = (SELECT id FROM new_invoice);

-- 5. Record payment
INSERT INTO invoice_payments (invoice_id, amount, payment_date, payment_method, created_by)
SELECT id, total_amount, CURRENT_DATE + 15, 'bank_transfer', 'user-uuid'
FROM invoices
WHERE id = (SELECT id FROM new_invoice);
```

### Common Queries

```sql
-- Overdue invoices with customer info
SELECT
    i.invoice_number,
    i.due_date,
    i.amount_due,
    CURRENT_DATE - i.due_date as days_overdue,
    u.name as customer,
    u.email
FROM invoices i
JOIN users u ON i.customer_id = u.id
WHERE i.status = 'overdue'
ORDER BY days_overdue DESC;

-- Revenue this month
SELECT
    COUNT(*) as invoices,
    SUM(total_amount) as total,
    SUM(amount_paid) as paid,
    SUM(amount_due) as outstanding
FROM invoices
WHERE DATE_TRUNC('month', issue_date) = DATE_TRUNC('month', CURRENT_DATE)
AND deleted_at IS NULL;

-- Top 10 customers by revenue
SELECT
    u.name,
    COUNT(i.id) as invoice_count,
    SUM(i.total_amount) as total_revenue
FROM users u
JOIN invoices i ON u.id = i.customer_id
WHERE i.deleted_at IS NULL
GROUP BY u.id, u.name
ORDER BY total_revenue DESC
LIMIT 10;
```

## 🔌 API Integration

### Using Go Models

```go
import (
    "github.com/javaknight1/servicepro/backend/internal/models"
    "gorm.io/gorm"
)

// Create invoice
invoice := &models.Invoice{
    CustomerID:    customerUUID,
    Status:        models.InvoiceStatusDraft,
    IssueDate:     time.Now(),
    DueDate:       time.Now().AddDate(0, 0, 30),
    PaymentTermID: &paymentTermUUID,
    TaxRateID:     &taxRateUUID,
    CreatedBy:     userUUID,
}
db.Create(invoice)

// Add line items
lineItem := &models.InvoiceLine{
    InvoiceID:   invoice.ID,
    Description: "Consulting Services",
    Quantity:    decimal.NewFromInt(10),
    UnitPrice:   decimal.NewFromFloat(150.00),
    Taxable:     true,
    SortOrder:   1,
}
db.Create(lineItem)

// Get invoice with relationships
var invoice models.Invoice
db.Preload("Customer").
   Preload("Lines").
   Preload("Payments").
   Preload("PaymentTerm").
   Preload("TaxRate").
   First(&invoice, "invoice_number = ?", "INV-2024-00001")

// List invoices with filter
filter := &models.InvoiceFilter{
    Status:    &statusSent,
    Page:      1,
    PageSize:  20,
    SortBy:    "issue_date",
    SortOrder: "desc",
}

var invoices []models.Invoice
query := db.Model(&models.Invoice{})

if filter.Status != nil {
    query = query.Where("status = ?", *filter.Status)
}

query.Offset((filter.Page - 1) * filter.PageSize).
      Limit(filter.PageSize).
      Order(filter.SortBy + " " + filter.SortOrder).
      Find(&invoices)
```

## 📚 Additional Resources

- **[INVOICE_SYSTEM.md](../INVOICE_SYSTEM.md)** - Complete system documentation
- **[INVOICE_SQL_REFERENCE.md](INVOICE_SQL_REFERENCE.md)** - SQL query reference
- **[invoice.go](../internal/models/invoice.go)** - Go models with GORM tags

## 🔐 Security Considerations

1. **Row-Level Security**: Implement in your application layer
2. **Audit Logs**: Never delete, always retain for compliance
3. **Soft Deletes**: Use `deleted_at` to maintain audit trail
4. **Payment Data**: Never store full credit card numbers
5. **Access Control**: Implement proper authorization checks
6. **SQL Injection**: Always use prepared statements

## 🚀 Performance Tips

1. Use provided indexes for optimal query performance
2. Leverage views for complex reporting queries
3. Run `update_overdue_invoices()` via cron daily
4. Archive old audit logs periodically
5. Use connection pooling in production
6. Enable query caching where appropriate

## 📈 Next Steps

### Recommended Enhancements

1. **Email Integration**: Send invoices and reminders via email
2. **PDF Generation**: Generate PDF invoices
3. **Recurring Invoices**: Support for subscription billing
4. **Credit Notes**: Handle refunds and adjustments
5. **Multi-Currency**: Support for international invoicing
6. **Payment Gateway Integration**: Stripe, PayPal, etc.
7. **Late Fee Automation**: Automatically apply configured late fees
8. **Dunning Process**: Automated collection reminders

### Integration Points

- **Quote System**: Generate invoices from accepted quotes
- **Job System**: Link invoices to completed jobs
- **Customer Portal**: Allow customers to view and pay invoices
- **Accounting Export**: Export to QuickBooks, Xero, etc.
- **Reporting Dashboard**: Real-time revenue analytics

## 📄 License

MIT License - See LICENSE file for details

## 🤝 Contributing

Contributions welcome! Please:

1. Follow existing code style
2. Add tests for new features
3. Update documentation
4. Ensure migrations are reversible

---

**Built with ❤️ for ServicePro**
