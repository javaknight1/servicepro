-- Reverts 000003_tenant_scoping. Restores global uniqueness (fails if two
-- tenants now share a document number or customer email).

DROP INDEX IF EXISTS idx_import_jobs_tenant_user;
DROP INDEX IF EXISTS idx_export_jobs_tenant_user;
DROP INDEX IF EXISTS idx_schedules_tenant_start;
DROP INDEX IF EXISTS idx_payments_tenant_created;
DROP INDEX IF EXISTS idx_invoices_tenant_customer;
DROP INDEX IF EXISTS idx_invoices_tenant_due;
DROP INDEX IF EXISTS idx_invoices_tenant_status;
DROP INDEX IF EXISTS idx_quotes_tenant_customer;
DROP INDEX IF EXISTS idx_quotes_tenant_status;
DROP INDEX IF EXISTS idx_jobs_tenant_scheduled;
DROP INDEX IF EXISTS idx_jobs_tenant_customer;
DROP INDEX IF EXISTS idx_jobs_tenant_status;
DROP INDEX IF EXISTS idx_customers_tenant_created;
DROP INDEX IF EXISTS idx_customers_tenant_status;

DROP INDEX IF EXISTS idx_invoice_templates_one_default;
CREATE UNIQUE INDEX idx_invoice_templates_one_default
    ON invoice_templates(is_default) WHERE is_default = TRUE AND status = 'active';
DROP INDEX IF EXISTS idx_payment_terms_one_default;
CREATE UNIQUE INDEX idx_payment_terms_one_default ON payment_terms(is_default) WHERE is_default = TRUE;

DROP INDEX IF EXISTS idx_customers_tenant_email;
CREATE UNIQUE INDEX idx_customers_email ON customers(email) WHERE deleted_at IS NULL;

ALTER TABLE invoices DROP CONSTRAINT IF EXISTS invoices_tenant_invoice_number_key;
ALTER TABLE quotes   DROP CONSTRAINT IF EXISTS quotes_tenant_quote_number_key;
ALTER TABLE jobs     DROP CONSTRAINT IF EXISTS jobs_tenant_job_number_key;
ALTER TABLE invoices ADD CONSTRAINT invoices_invoice_number_key UNIQUE (invoice_number);
ALTER TABLE quotes   ADD CONSTRAINT quotes_quote_number_key UNIQUE (quote_number);
ALTER TABLE jobs     ADD CONSTRAINT jobs_job_number_key UNIQUE (job_number);

ALTER TABLE revenue_summary_cache DROP CONSTRAINT IF EXISTS revenue_summary_cache_tenant_period_key;
DELETE FROM revenue_summary_cache;
ALTER TABLE revenue_summary_cache DROP COLUMN IF EXISTS tenant_id;
ALTER TABLE revenue_summary_cache ADD UNIQUE (period_type, period_start, period_end, currency);

DROP INDEX IF EXISTS idx_revenue_transactions_tenant_date;
ALTER TABLE revenue_transactions DROP COLUMN IF EXISTS tenant_id;

ALTER TABLE import_jobs         ALTER COLUMN tenant_id DROP NOT NULL;
ALTER TABLE export_jobs         ALTER COLUMN tenant_id DROP NOT NULL;
ALTER TABLE recurring_schedules ALTER COLUMN tenant_id DROP NOT NULL;
ALTER TABLE schedules           ALTER COLUMN tenant_id DROP NOT NULL;
ALTER TABLE payments            ALTER COLUMN tenant_id DROP NOT NULL;
ALTER TABLE invoices            ALTER COLUMN tenant_id DROP NOT NULL;
ALTER TABLE quotes              ALTER COLUMN tenant_id DROP NOT NULL;
ALTER TABLE jobs                ALTER COLUMN tenant_id DROP NOT NULL;
ALTER TABLE customers           ALTER COLUMN tenant_id DROP NOT NULL;
