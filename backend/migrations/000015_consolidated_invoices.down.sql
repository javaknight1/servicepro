DROP TABLE IF EXISTS consolidated_invoice_runs;
DROP TABLE IF EXISTS invoice_billed_items;
ALTER TABLE invoice_lines DROP COLUMN IF EXISTS approval_note;
ALTER TABLE invoice_lines DROP COLUMN IF EXISTS site_name;
ALTER TABLE invoices DROP COLUMN IF EXISTS billing_period_end;
ALTER TABLE invoices DROP COLUMN IF EXISTS billing_period_start;
