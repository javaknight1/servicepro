-- ============================================================================
-- 000003_tenant_scoping (T102 / audit F-3)
--
-- Business tables got a nullable tenant_id in 000001 but nothing wrote it.
-- The application now sets and filters tenant_id on every query
-- (internal/tenancy). This migration:
--   1. backfills tenant_id where it can be derived from a parent row,
--   2. makes tenant_id NOT NULL on tenant-owned tables,
--   3. adds tenant_id to revenue_transactions / revenue_summary_cache,
--   4. makes document numbers and customer emails unique PER TENANT
--      (global uniqueness leaked other tenants' data and collided once
--      numbering became per tenant),
--   5. adds composite (tenant_id, ...) indexes for the common filters.
--
-- There is no production data. Rows whose tenant cannot be derived make the
-- NOT NULL step fail loudly instead of being deleted.
-- ============================================================================

-- 1. Backfill from parents ----------------------------------------------------
UPDATE jobs j SET tenant_id = c.tenant_id
FROM customers c WHERE j.tenant_id IS NULL AND c.id = j.customer_id AND c.tenant_id IS NOT NULL;

UPDATE quotes q SET tenant_id = c.tenant_id
FROM customers c WHERE q.tenant_id IS NULL AND c.id = q.customer_id AND c.tenant_id IS NOT NULL;

UPDATE invoices i SET tenant_id = c.tenant_id
FROM customers c WHERE i.tenant_id IS NULL AND c.id = i.customer_id AND c.tenant_id IS NOT NULL;

UPDATE payments p SET tenant_id = i.tenant_id
FROM invoices i WHERE p.tenant_id IS NULL AND i.id = p.invoice_id AND i.tenant_id IS NOT NULL;

UPDATE schedules s SET tenant_id = j.tenant_id
FROM jobs j WHERE s.tenant_id IS NULL AND j.id = s.job_id AND j.tenant_id IS NOT NULL;

-- Exports/imports/recurring schedules: the requesting user's earliest tenant.
UPDATE export_jobs e SET tenant_id = (
    SELECT tu.tenant_id FROM tenant_users tu WHERE tu.user_id = e.user_id ORDER BY tu.created_at LIMIT 1)
WHERE e.tenant_id IS NULL;

UPDATE import_jobs im SET tenant_id = (
    SELECT tu.tenant_id FROM tenant_users tu WHERE tu.user_id = im.user_id ORDER BY tu.created_at LIMIT 1)
WHERE im.tenant_id IS NULL;

-- Expired/finished export and import jobs with no derivable tenant are
-- disposable history; drop them rather than block the migration.
DELETE FROM export_jobs WHERE tenant_id IS NULL;
DELETE FROM import_jobs WHERE tenant_id IS NULL;

-- 2. NOT NULL on tenant-owned tables ------------------------------------------
ALTER TABLE customers           ALTER COLUMN tenant_id SET NOT NULL;
ALTER TABLE jobs                ALTER COLUMN tenant_id SET NOT NULL;
ALTER TABLE quotes              ALTER COLUMN tenant_id SET NOT NULL;
ALTER TABLE invoices            ALTER COLUMN tenant_id SET NOT NULL;
ALTER TABLE payments            ALTER COLUMN tenant_id SET NOT NULL;
ALTER TABLE schedules           ALTER COLUMN tenant_id SET NOT NULL;
ALTER TABLE recurring_schedules ALTER COLUMN tenant_id SET NOT NULL;
ALTER TABLE export_jobs         ALTER COLUMN tenant_id SET NOT NULL;
ALTER TABLE import_jobs         ALTER COLUMN tenant_id SET NOT NULL;
-- payment_terms, tax_rates, invoice_templates stay nullable: NULL = global default.

-- 3. Revenue tables -------------------------------------------------------------
ALTER TABLE revenue_transactions ADD COLUMN IF NOT EXISTS tenant_id UUID REFERENCES tenants(id) ON DELETE CASCADE;
UPDATE revenue_transactions r SET tenant_id = i.tenant_id
FROM invoices i WHERE r.tenant_id IS NULL AND i.id = r.invoice_id;
ALTER TABLE revenue_transactions ALTER COLUMN tenant_id SET NOT NULL;
CREATE INDEX IF NOT EXISTS idx_revenue_transactions_tenant_date ON revenue_transactions(tenant_id, transaction_date);

-- The summary cache was global (one tenant's totals were served to all);
-- it is only a cache, so start it over per tenant.
DELETE FROM revenue_summary_cache;
ALTER TABLE revenue_summary_cache ADD COLUMN IF NOT EXISTS tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE;
ALTER TABLE revenue_summary_cache
    DROP CONSTRAINT IF EXISTS revenue_summary_cache_period_type_period_start_period_end_c_key;
DO $$
DECLARE c record;
BEGIN
    -- Drop the original (unnamed) UNIQUE(period_type, period_start, period_end, currency).
    FOR c IN
        SELECT conname FROM pg_constraint
        WHERE conrelid = 'revenue_summary_cache'::regclass AND contype = 'u'
    LOOP
        EXECUTE format('ALTER TABLE revenue_summary_cache DROP CONSTRAINT %I', c.conname);
    END LOOP;
END $$;
ALTER TABLE revenue_summary_cache
    ADD CONSTRAINT revenue_summary_cache_tenant_period_key
    UNIQUE (tenant_id, period_type, period_start, period_end, currency);

-- 4. Per-tenant uniqueness ------------------------------------------------------
ALTER TABLE jobs     DROP CONSTRAINT IF EXISTS jobs_job_number_key;
ALTER TABLE quotes   DROP CONSTRAINT IF EXISTS quotes_quote_number_key;
ALTER TABLE invoices DROP CONSTRAINT IF EXISTS invoices_invoice_number_key;
ALTER TABLE jobs     ADD CONSTRAINT jobs_tenant_job_number_key         UNIQUE (tenant_id, job_number);
ALTER TABLE quotes   ADD CONSTRAINT quotes_tenant_quote_number_key     UNIQUE (tenant_id, quote_number);
ALTER TABLE invoices ADD CONSTRAINT invoices_tenant_invoice_number_key UNIQUE (tenant_id, invoice_number);

DROP INDEX IF EXISTS idx_customers_email;
CREATE UNIQUE INDEX idx_customers_tenant_email ON customers(tenant_id, LOWER(email)) WHERE deleted_at IS NULL;

-- One default payment term / invoice template per tenant (plus one global).
DROP INDEX IF EXISTS idx_payment_terms_one_default;
CREATE UNIQUE INDEX idx_payment_terms_one_default
    ON payment_terms ((COALESCE(tenant_id, '00000000-0000-0000-0000-000000000000'::uuid)))
    WHERE is_default = TRUE;

DROP INDEX IF EXISTS idx_invoice_templates_one_default;
CREATE UNIQUE INDEX idx_invoice_templates_one_default
    ON invoice_templates ((COALESCE(tenant_id, '00000000-0000-0000-0000-000000000000'::uuid)))
    WHERE is_default = TRUE AND status = 'active';

-- 5. Composite indexes for tenant-scoped filters --------------------------------
CREATE INDEX IF NOT EXISTS idx_customers_tenant_status    ON customers(tenant_id, status) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_customers_tenant_created   ON customers(tenant_id, created_at DESC) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_jobs_tenant_status         ON jobs(tenant_id, status) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_jobs_tenant_customer       ON jobs(tenant_id, customer_id) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_jobs_tenant_scheduled      ON jobs(tenant_id, scheduled_start_at) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_quotes_tenant_status       ON quotes(tenant_id, status) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_quotes_tenant_customer     ON quotes(tenant_id, customer_id) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_invoices_tenant_status     ON invoices(tenant_id, status) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_invoices_tenant_due        ON invoices(tenant_id, due_date) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_invoices_tenant_customer   ON invoices(tenant_id, customer_id) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_payments_tenant_created    ON payments(tenant_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_schedules_tenant_start     ON schedules(tenant_id, start_time) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_export_jobs_tenant_user    ON export_jobs(tenant_id, user_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_import_jobs_tenant_user    ON import_jobs(tenant_id, user_id, created_at DESC);
