# Invoice System - SQL Quick Reference

Common SQL queries and operations for the invoice system.

## Table of Contents

- [Invoice CRUD](#invoice-crud)
- [Payments](#payments)
- [Queries](#queries)
- [Reports](#reports)
- [Maintenance](#maintenance)

## Invoice CRUD

### Create Invoice

```sql
-- Basic invoice creation
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
    '550e8400-e29b-41d4-a716-446655440000',
    'draft',
    CURRENT_DATE,
    CURRENT_DATE + INTERVAL '30 days',
    (SELECT id FROM payment_terms WHERE term_type = 'net_30' LIMIT 1),
    (SELECT id FROM tax_rates WHERE region = 'CA' AND is_active = true LIMIT 1),
    'Project XYZ Invoice',
    '650e8400-e29b-41d4-a716-446655440000'
) RETURNING id, invoice_number;
```

### Add Line Items

```sql
-- Add multiple line items to an invoice
INSERT INTO invoice_lines (invoice_id, description, quantity, unit_price, taxable, sort_order)
VALUES
    ('invoice-uuid-here', 'Consulting Services - 10 hours', 10, 150.00, true, 1),
    ('invoice-uuid-here', 'Software License (annual)', 1, 999.00, true, 2),
    ('invoice-uuid-here', 'Setup Fee', 1, 250.00, true, 3);

-- Line with discount
INSERT INTO invoice_lines (
    invoice_id, description, quantity, unit_price,
    discount_percentage, discount_amount, taxable, sort_order
)
VALUES (
    'invoice-uuid-here',
    'Bulk Service Package',
    100, 50.00,
    10.00, 500.00,  -- 10% discount = $500
    true, 4
);
```

### Read Invoice

```sql
-- Get invoice with all details
SELECT
    i.*,
    u.email as customer_email,
    u.name as customer_name,
    pt.name as payment_terms,
    tr.name as tax_rate_name,
    tr.rate as tax_rate_value
FROM invoices i
LEFT JOIN users u ON i.customer_id = u.id
LEFT JOIN payment_terms pt ON i.payment_term_id = pt.id
LEFT JOIN tax_rates tr ON i.tax_rate_id = tr.id
WHERE i.id = 'invoice-uuid-here';

-- Get invoice with line items
SELECT
    i.invoice_number,
    i.status,
    i.total_amount,
    json_agg(
        json_build_object(
            'description', il.description,
            'quantity', il.quantity,
            'unit_price', il.unit_price,
            'line_total', il.line_total,
            'tax_amount', il.tax_amount
        ) ORDER BY il.sort_order
    ) as line_items
FROM invoices i
LEFT JOIN invoice_lines il ON i.id = il.invoice_id
WHERE i.invoice_number = 'INV-2024-00001'
GROUP BY i.id;
```

### Update Invoice

```sql
-- Update invoice status
UPDATE invoices
SET
    status = 'sent',
    sent_date = NOW(),
    updated_by = 'user-uuid-here'
WHERE id = 'invoice-uuid-here';

-- Update invoice details
UPDATE invoices
SET
    notes = 'Updated project notes',
    po_number = 'PO-2024-5678',
    updated_by = 'user-uuid-here'
WHERE id = 'invoice-uuid-here';

-- Apply discount to invoice
UPDATE invoices
SET
    discount_amount = 500.00,
    updated_by = 'user-uuid-here'
WHERE id = 'invoice-uuid-here';
-- Total will recalculate automatically
```

### Delete Invoice (Soft Delete)

```sql
-- Soft delete (sets deleted_at)
UPDATE invoices
SET deleted_at = NOW()
WHERE id = 'invoice-uuid-here';

-- Restore soft-deleted invoice
UPDATE invoices
SET deleted_at = NULL
WHERE id = 'invoice-uuid-here';
```

## Payments

### Record Full Payment

```sql
INSERT INTO invoice_payments (
    invoice_id,
    amount,
    payment_date,
    payment_method,
    reference_number,
    notes,
    created_by
) VALUES (
    'invoice-uuid-here',
    5000.00,
    CURRENT_DATE,
    'bank_transfer',
    'TXN-2024-12345',
    'Wire transfer received',
    'user-uuid-here'
);
-- Invoice status will automatically update to 'paid'
```

### Record Partial Payment

```sql
-- First partial payment
INSERT INTO invoice_payments (
    invoice_id, amount, payment_date, payment_method, created_by
)
VALUES (
    'invoice-uuid-here',
    2500.00,
    CURRENT_DATE,
    'credit_card',
    'user-uuid-here'
);
-- Invoice status automatically changes to 'partially_paid'

-- Second partial payment
INSERT INTO invoice_payments (
    invoice_id, amount, payment_date, payment_method, created_by
)
VALUES (
    'invoice-uuid-here',
    2500.00,
    CURRENT_DATE + INTERVAL '15 days',
    'credit_card',
    'user-uuid-here'
);
-- Invoice status automatically changes to 'paid' when fully paid
```

### View Payment History

```sql
SELECT
    i.invoice_number,
    i.total_amount,
    i.amount_paid,
    i.amount_due,
    ip.payment_date,
    ip.amount,
    ip.payment_method,
    ip.reference_number,
    ip.notes
FROM invoices i
LEFT JOIN invoice_payments ip ON i.id = ip.invoice_id
WHERE i.invoice_number = 'INV-2024-00001'
ORDER BY ip.payment_date;
```

## Queries

### Find Invoices by Status

```sql
-- All unpaid invoices
SELECT
    invoice_number,
    customer_id,
    issue_date,
    due_date,
    total_amount,
    amount_due
FROM invoices
WHERE status IN ('sent', 'viewed')
AND deleted_at IS NULL
ORDER BY issue_date DESC;

-- All paid invoices
SELECT
    invoice_number,
    paid_date,
    total_amount
FROM invoices
WHERE status = 'paid'
AND deleted_at IS NULL
ORDER BY paid_date DESC;
```

### Find Overdue Invoices

```sql
-- Using invoice_summary view
SELECT
    invoice_number,
    customer_id,
    issue_date,
    due_date,
    total_amount,
    amount_due,
    days_overdue
FROM invoice_summary
WHERE is_overdue = true
ORDER BY days_overdue DESC;

-- Or directly from invoices table
SELECT
    invoice_number,
    customer_id,
    due_date,
    total_amount,
    amount_due,
    CURRENT_DATE - due_date as days_overdue
FROM invoices
WHERE status IN ('sent', 'viewed', 'partially_paid', 'overdue')
AND due_date < CURRENT_DATE
AND amount_due > 0
AND deleted_at IS NULL
ORDER BY due_date;
```

### Search Invoices

```sql
-- Search by invoice number or customer
SELECT
    i.invoice_number,
    i.status,
    i.total_amount,
    u.name as customer_name,
    u.email as customer_email
FROM invoices i
JOIN users u ON i.customer_id = u.id
WHERE (
    i.invoice_number ILIKE '%2024%'
    OR u.name ILIKE '%John%'
    OR u.email ILIKE '%john%'
)
AND i.deleted_at IS NULL;

-- Search by date range
SELECT *
FROM invoices
WHERE issue_date BETWEEN '2024-01-01' AND '2024-12-31'
AND deleted_at IS NULL
ORDER BY issue_date DESC;

-- Search by amount range
SELECT *
FROM invoices
WHERE total_amount BETWEEN 1000.00 AND 5000.00
AND deleted_at IS NULL;
```

### Customer Invoice History

```sql
-- All invoices for a customer
SELECT
    invoice_number,
    status,
    issue_date,
    due_date,
    total_amount,
    amount_paid,
    amount_due,
    CASE
        WHEN status = 'paid' THEN 'Paid'
        WHEN due_date < CURRENT_DATE AND amount_due > 0 THEN 'Overdue'
        ELSE 'Outstanding'
    END as payment_status
FROM invoices
WHERE customer_id = 'customer-uuid-here'
AND deleted_at IS NULL
ORDER BY issue_date DESC;

-- Customer summary
SELECT
    customer_id,
    COUNT(*) as total_invoices,
    COUNT(*) FILTER (WHERE status = 'paid') as paid_invoices,
    COUNT(*) FILTER (WHERE status IN ('sent', 'viewed', 'partially_paid')) as outstanding_invoices,
    COUNT(*) FILTER (WHERE status = 'overdue') as overdue_invoices,
    SUM(total_amount) as total_billed,
    SUM(amount_paid) as total_paid,
    SUM(amount_due) as total_outstanding
FROM invoices
WHERE customer_id = 'customer-uuid-here'
AND deleted_at IS NULL
GROUP BY customer_id;
```

## Reports

### Revenue by Month

```sql
-- Using the view
SELECT
    TO_CHAR(month, 'YYYY-MM') as period,
    invoice_count,
    total_revenue,
    total_paid,
    total_outstanding,
    ROUND((total_paid::numeric / NULLIF(total_revenue, 0) * 100), 2) as collection_rate
FROM revenue_by_month
ORDER BY month DESC
LIMIT 12;

-- Or calculate manually
SELECT
    DATE_TRUNC('month', issue_date) as month,
    COUNT(*) as invoice_count,
    SUM(total_amount) as total_revenue,
    SUM(amount_paid) as total_paid,
    SUM(amount_due) as total_outstanding
FROM invoices
WHERE deleted_at IS NULL
GROUP BY DATE_TRUNC('month', issue_date)
ORDER BY month DESC;
```

### Aging Report

```sql
SELECT
    customer_id,
    COUNT(*) as total_invoices,
    SUM(amount_due) as total_outstanding,
    SUM(CASE WHEN CURRENT_DATE - due_date <= 0 THEN amount_due ELSE 0 END) as current,
    SUM(CASE WHEN CURRENT_DATE - due_date BETWEEN 1 AND 30 THEN amount_due ELSE 0 END) as days_1_30,
    SUM(CASE WHEN CURRENT_DATE - due_date BETWEEN 31 AND 60 THEN amount_due ELSE 0 END) as days_31_60,
    SUM(CASE WHEN CURRENT_DATE - due_date BETWEEN 61 AND 90 THEN amount_due ELSE 0 END) as days_61_90,
    SUM(CASE WHEN CURRENT_DATE - due_date > 90 THEN amount_due ELSE 0 END) as days_over_90
FROM invoices
WHERE status IN ('sent', 'viewed', 'partially_paid', 'overdue')
AND amount_due > 0
AND deleted_at IS NULL
GROUP BY customer_id
ORDER BY total_outstanding DESC;
```

### Tax Report

```sql
SELECT
    DATE_TRUNC('month', i.issue_date) as month,
    tr.name as tax_rate_name,
    tr.region,
    COUNT(i.id) as invoice_count,
    SUM(i.subtotal) as subtotal,
    SUM(i.tax_amount) as tax_collected,
    SUM(i.total_amount) as total_with_tax
FROM invoices i
LEFT JOIN tax_rates tr ON i.tax_rate_id = tr.id
WHERE i.status = 'paid'
AND i.deleted_at IS NULL
AND i.issue_date >= DATE_TRUNC('year', CURRENT_DATE)
GROUP BY DATE_TRUNC('month', i.issue_date), tr.name, tr.region
ORDER BY month DESC, tr.region;
```

### Top Customers by Revenue

```sql
SELECT
    u.id,
    u.name,
    u.email,
    COUNT(i.id) as invoice_count,
    SUM(i.total_amount) as total_revenue,
    SUM(i.amount_paid) as total_paid,
    AVG(i.total_amount) as avg_invoice_amount,
    MAX(i.issue_date) as last_invoice_date
FROM users u
JOIN invoices i ON u.id = i.customer_id
WHERE i.deleted_at IS NULL
GROUP BY u.id, u.name, u.email
ORDER BY total_revenue DESC
LIMIT 10;
```

### Payment Method Report

```sql
SELECT
    payment_method,
    COUNT(*) as payment_count,
    SUM(amount) as total_amount,
    AVG(amount) as avg_amount,
    MIN(amount) as min_amount,
    MAX(amount) as max_amount
FROM invoice_payments
WHERE payment_date >= DATE_TRUNC('month', CURRENT_DATE)
GROUP BY payment_method
ORDER BY total_amount DESC;
```

## Maintenance

### Update Overdue Invoices

```sql
-- Run this daily via cron
SELECT update_overdue_invoices();

-- Or manually
UPDATE invoices
SET status = 'overdue'
WHERE status IN ('sent', 'viewed', 'partially_paid')
AND due_date < CURRENT_DATE
AND amount_due > 0
AND deleted_at IS NULL;
```

### Validate Invoice

```sql
-- Validate before sending
SELECT * FROM validate_invoice_for_sending('invoice-uuid-here');
```

### Recalculate Totals

```sql
-- If totals seem incorrect, triggers should handle this automatically
-- But you can manually verify:

SELECT
    i.id,
    i.invoice_number,
    i.subtotal as current_subtotal,
    COALESCE(SUM(il.line_total), 0) as calculated_subtotal,
    i.tax_amount as current_tax,
    COALESCE(SUM(il.tax_amount), 0) as calculated_tax
FROM invoices i
LEFT JOIN invoice_lines il ON i.id = il.invoice_id
WHERE i.id = 'invoice-uuid-here'
GROUP BY i.id;
```

### Clean Up Old Audit Logs

```sql
-- Keep only last 2 years of audit logs
DELETE FROM invoice_audit_log
WHERE created_at < NOW() - INTERVAL '2 years';

-- Or archive to separate table
INSERT INTO invoice_audit_log_archive
SELECT * FROM invoice_audit_log
WHERE created_at < NOW() - INTERVAL '2 years';

DELETE FROM invoice_audit_log
WHERE created_at < NOW() - INTERVAL '2 years';
```

### Data Integrity Checks

```sql
-- Find invoices with mismatched totals
SELECT
    i.id,
    i.invoice_number,
    i.total_amount,
    i.subtotal + i.tax_amount - COALESCE(i.discount_amount, 0) as calculated_total
FROM invoices i
WHERE ABS(i.total_amount - (i.subtotal + i.tax_amount - COALESCE(i.discount_amount, 0))) > 0.01
AND deleted_at IS NULL;

-- Find invoices with invalid payment amounts
SELECT
    id,
    invoice_number,
    total_amount,
    amount_paid
FROM invoices
WHERE amount_paid > total_amount
AND deleted_at IS NULL;

-- Find invoices with no line items
SELECT
    i.id,
    i.invoice_number,
    i.status
FROM invoices i
LEFT JOIN invoice_lines il ON i.id = il.invoice_id
WHERE il.id IS NULL
AND i.status != 'draft'
AND i.deleted_at IS NULL;
```

## Performance Optimization

### Analyze Query Performance

```sql
-- Enable timing
\timing on

-- Explain query plan
EXPLAIN ANALYZE
SELECT * FROM invoices
WHERE customer_id = 'customer-uuid-here'
AND status = 'paid'
ORDER BY issue_date DESC;

-- Check index usage
SELECT
    schemaname,
    tablename,
    indexname,
    idx_scan,
    idx_tup_read,
    idx_tup_fetch
FROM pg_stat_user_indexes
WHERE tablename IN ('invoices', 'invoice_lines', 'invoice_payments')
ORDER BY idx_scan DESC;
```

### Vacuum and Analyze

```sql
-- Vacuum tables to reclaim space
VACUUM ANALYZE invoices;
VACUUM ANALYZE invoice_lines;
VACUUM ANALYZE invoice_payments;

-- Auto-vacuum should handle this, but can be run manually if needed
```

## Tips

1. **Always use prepared statements** in your application to prevent SQL injection
2. **Use transactions** for operations that modify multiple tables
3. **Let triggers handle calculations** - don't manually update totals
4. **Use soft deletes** (`deleted_at`) instead of hard deletes for audit trail
5. **Index wisely** - add indexes for frequently queried fields
6. **Monitor slow queries** - use `EXPLAIN ANALYZE` to optimize
7. **Batch updates** when possible for better performance
8. **Use views** (`invoice_summary`, `revenue_by_month`) for complex queries
