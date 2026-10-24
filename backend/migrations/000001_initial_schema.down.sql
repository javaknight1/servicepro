-- Reverses 000001_initial_schema (pre-launch squash of 000001–000014, 2026-10-24):
-- the folded-in sections are undone newest first, then the baseline.

-- ---- 000014_tax_rate_permissions ----
DROP INDEX IF EXISTS idx_tax_rates_tenant_name;

DELETE FROM role_permissions
WHERE permission_id IN (SELECT id FROM permissions WHERE name IN ('tax_rates.read', 'tax_rates.manage'));

DELETE FROM permissions WHERE name IN ('tax_rates.read', 'tax_rates.manage');

-- ---- 000013_document_currency ----
ALTER TABLE quotes DROP COLUMN IF EXISTS currency;
ALTER TABLE invoices DROP COLUMN IF EXISTS currency;

-- ---- 000012_live_uniqueness ----
-- Restoring the old global constraints fails if live data now relies on the
-- relaxed rules (a reused slug, or several non-pending invitations for one
-- email); resolve those rows by hand first.
DROP INDEX IF EXISTS uq_org_invitations_pending;
ALTER TABLE organization_invitations DROP CONSTRAINT IF EXISTS unique_pending_invitation;
ALTER TABLE organization_invitations ADD CONSTRAINT unique_pending_invitation UNIQUE (tenant_id, email);

DROP INDEX IF EXISTS uq_tenants_slug_live;
ALTER TABLE tenants DROP CONSTRAINT IF EXISTS tenants_slug_key;
ALTER TABLE tenants ADD CONSTRAINT tenants_slug_key UNIQUE (slug);

-- ---- 000011_customer_search_trgm ----
DROP INDEX IF EXISTS idx_customers_company_name_trgm;
DROP INDEX IF EXISTS idx_customers_email_trgm;
DROP INDEX IF EXISTS idx_customers_last_name_trgm;
DROP INDEX IF EXISTS idx_customers_first_name_trgm;

-- ---- 000010_payment_reminder_kinds ----
DROP INDEX IF EXISTS uq_payment_reminders_invoice_kind_number;

-- Manual and automatic rows may share a number once the kinds are split;
-- renumber manual rows above the automatic range before restoring the old
-- constraint.
UPDATE payment_reminders SET reminder_number = reminder_number + 1000 WHERE is_manual;

ALTER TABLE payment_reminders DROP CONSTRAINT IF EXISTS unique_reminder_per_invoice;
ALTER TABLE payment_reminders ADD CONSTRAINT unique_reminder_per_invoice UNIQUE (invoice_id, reminder_number);

ALTER TABLE payment_reminders DROP COLUMN IF EXISTS is_manual;

-- ---- 000009_invoice_refunds ----
DROP TABLE IF EXISTS invoice_refunds;

-- ---- 000008_tenant_counters ----
-- Restore the pre-000008 numbering objects (as in 000001_initial_schema).
CREATE SEQUENCE IF NOT EXISTS quote_number_seq START 1;

CREATE OR REPLACE FUNCTION generate_quote_number()
RETURNS TEXT AS $$
DECLARE
    next_num INTEGER;
BEGIN
    next_num := nextval('quote_number_seq');
    RETURN 'QUO-' || TO_CHAR(CURRENT_DATE, 'YYYY') || '-' || LPAD(next_num::TEXT, 4, '0');
END;
$$ LANGUAGE plpgsql;

CREATE OR REPLACE FUNCTION generate_invoice_number()
RETURNS TRIGGER AS $$
DECLARE
    next_number INTEGER;
    year_prefix VARCHAR(4);
BEGIN
    IF NEW.invoice_number IS NULL OR NEW.invoice_number = '' THEN
        year_prefix := TO_CHAR(CURRENT_DATE, 'YYYY');
        SELECT COALESCE(MAX(CAST(SUBSTRING(invoice_number FROM '\d+$') AS INTEGER)), 0) + 1
        INTO next_number FROM invoices WHERE invoice_number LIKE 'INV-' || year_prefix || '-%';
        NEW.invoice_number := 'INV-' || year_prefix || '-' || LPAD(next_number::TEXT, 5, '0');
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trigger_generate_invoice_number ON invoices;
CREATE TRIGGER trigger_generate_invoice_number
    BEFORE INSERT ON invoices FOR EACH ROW EXECUTE FUNCTION generate_invoice_number();

DROP TABLE IF EXISTS tenant_counters;

-- ---- 000007_invoice_payment_ledger ----
DROP INDEX IF EXISTS idx_invoice_payments_stripe_payment_intent;
DROP INDEX IF EXISTS idx_invoice_payments_stripe_session;
-- Webhook-recorded payments have no created_by; they cannot satisfy NOT NULL.
DELETE FROM invoice_payments WHERE created_by IS NULL;
ALTER TABLE invoice_payments ALTER COLUMN created_by SET NOT NULL;
ALTER TABLE invoice_payments DROP COLUMN IF EXISTS stripe_payment_intent_id;
ALTER TABLE invoice_payments DROP COLUMN IF EXISTS stripe_checkout_session_id;

-- ---- 000006_stripe_webhook_events ----
DROP INDEX IF EXISTS idx_invoices_stripe_checkout_session;
DROP TABLE IF EXISTS stripe_webhook_events;

-- ---- 000005_user_token_version ----
ALTER TABLE users DROP COLUMN IF EXISTS token_version;

-- ---- 000004_invite_codes ----
DROP TABLE IF EXISTS invite_codes;

-- ---- 000003_tenant_scoping ----
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

-- ---- 000002_fix_ar_aging_summary_view ----
-- Intentionally a no-op: 000001_initial_schema also creates
-- v_ar_aging_summary (and its down migration drops it), so rolling back to
-- version 1 must keep the view.
SELECT 1;

-- ---- 000001_initial_schema (baseline) ----
-- ============================================================================
-- 000001_initial_schema (down)
-- Drops every object created by 000001_initial_schema.up.sql.
--
-- DESTRUCTIVE: this removes all application data. It exists for local
-- development resets only (`go run ./cmd/migrate down`). The migrate CLI
-- refuses to run it when ENV=production unless ALLOW_PRODUCTION_DOWN=true.
-- Production rollback is always a forward-fix migration.
--
-- Extensions (uuid-ossp, pg_trgm) are intentionally left installed: they may
-- be shared with other schemas and are harmless to keep.
-- ============================================================================

-- Views (drop before the tables they read from)
DROP VIEW IF EXISTS v_ar_aging_summary CASCADE;
DROP VIEW IF EXISTS v_revenue_daily CASCADE;
DROP VIEW IF EXISTS v_quote_conversion CASCADE;
DROP VIEW IF EXISTS v_invoice_metrics_daily CASCADE;
DROP VIEW IF EXISTS v_job_metrics_daily CASCADE;
DROP VIEW IF EXISTS v_feature_adoption CASCADE;
DROP VIEW IF EXISTS v_tenant_growth_daily CASCADE;
DROP VIEW IF EXISTS v_platform_overview CASCADE;

-- Analytics schema (analytics.events, analytics.aggregated_metrics)
DROP SCHEMA IF EXISTS analytics CASCADE;

-- Tables (reverse creation order; CASCADE also removes triggers, indexes and owned sequences)
DROP TABLE IF EXISTS payment_reminders CASCADE;
DROP TABLE IF EXISTS billing_events CASCADE;
DROP TABLE IF EXISTS payment_methods CASCADE;
DROP TABLE IF EXISTS tenant_subscriptions CASCADE;
DROP TABLE IF EXISTS membership_limitations CASCADE;
DROP TABLE IF EXISTS membership_tiers CASCADE;
DROP TABLE IF EXISTS revenue_alerts CASCADE;
DROP TABLE IF EXISTS revenue_summary_cache CASCADE;
DROP TABLE IF EXISTS revenue_transactions CASCADE;
DROP TABLE IF EXISTS organization_invitations CASCADE;
DROP TABLE IF EXISTS tenant_users CASCADE;
DROP TABLE IF EXISTS tenants CASCADE;
DROP TABLE IF EXISTS subscriptions CASCADE;
DROP TABLE IF EXISTS plan_features CASCADE;
DROP TABLE IF EXISTS subscription_plans CASCADE;
DROP TABLE IF EXISTS customer_metrics_cache CASCADE;
DROP TABLE IF EXISTS customer_segments CASCADE;
DROP TABLE IF EXISTS customer_activities CASCADE;
DROP TABLE IF EXISTS export_jobs CASCADE;
DROP TABLE IF EXISTS import_errors CASCADE;
DROP TABLE IF EXISTS import_jobs CASCADE;
DROP TABLE IF EXISTS invoice_templates CASCADE;
DROP TABLE IF EXISTS invoice_payments CASCADE;
DROP TABLE IF EXISTS invoice_audit_log CASCADE;
DROP TABLE IF EXISTS payments CASCADE;
DROP TABLE IF EXISTS invoice_lines CASCADE;
DROP TABLE IF EXISTS invoices CASCADE;
DROP TABLE IF EXISTS payment_terms CASCADE;
DROP TABLE IF EXISTS tax_rates CASCADE;
DROP TABLE IF EXISTS quote_items CASCADE;
DROP TABLE IF EXISTS quotes CASCADE;
DROP TABLE IF EXISTS holidays CASCADE;
DROP TABLE IF EXISTS schedule_exceptions CASCADE;
DROP TABLE IF EXISTS schedule_conflicts CASCADE;
DROP TABLE IF EXISTS schedules CASCADE;
DROP TABLE IF EXISTS recurring_schedules CASCADE;
DROP TABLE IF EXISTS job_status_transitions CASCADE;
DROP TABLE IF EXISTS job_notes CASCADE;
DROP TABLE IF EXISTS job_materials CASCADE;
DROP TABLE IF EXISTS job_assignments CASCADE;
DROP TABLE IF EXISTS jobs CASCADE;
DROP TABLE IF EXISTS customers CASCADE;
DROP TABLE IF EXISTS user_roles CASCADE;
DROP TABLE IF EXISTS role_permissions CASCADE;
DROP TABLE IF EXISTS permissions CASCADE;
DROP TABLE IF EXISTS roles CASCADE;
DROP TABLE IF EXISTS users CASCADE;

-- Sequences
DROP SEQUENCE IF EXISTS quote_number_seq CASCADE;

-- Functions
DROP FUNCTION IF EXISTS get_default_payment_method CASCADE;
DROP FUNCTION IF EXISTS set_default_payment_method CASCADE;
DROP FUNCTION IF EXISTS check_tenant_limit CASCADE;
DROP FUNCTION IF EXISTS get_tenant_subscription CASCADE;
DROP FUNCTION IF EXISTS get_user_tenant_role CASCADE;
DROP FUNCTION IF EXISTS user_belongs_to_tenant CASCADE;
DROP FUNCTION IF EXISTS get_user_tenant_ids CASCADE;
DROP FUNCTION IF EXISTS generate_invoice_number CASCADE;
DROP FUNCTION IF EXISTS generate_quote_number CASCADE;
DROP FUNCTION IF EXISTS calculate_quote_item_total CASCADE;
DROP FUNCTION IF EXISTS validate_zip_code CASCADE;
DROP FUNCTION IF EXISTS validate_state_code CASCADE;
DROP FUNCTION IF EXISTS validate_phone CASCADE;
DROP FUNCTION IF EXISTS validate_email CASCADE;
DROP FUNCTION IF EXISTS update_updated_at_column CASCADE;

-- Enum types
DROP TYPE IF EXISTS billing_cycle CASCADE;
DROP TYPE IF EXISTS membership_status CASCADE;
DROP TYPE IF EXISTS invitation_status CASCADE;
DROP TYPE IF EXISTS notification_channel CASCADE;
DROP TYPE IF EXISTS notification_status CASCADE;
DROP TYPE IF EXISTS notification_type CASCADE;
DROP TYPE IF EXISTS template_status CASCADE;
DROP TYPE IF EXISTS tax_type CASCADE;
DROP TYPE IF EXISTS payment_term_type CASCADE;
DROP TYPE IF EXISTS invoice_status CASCADE;
DROP TYPE IF EXISTS quote_status CASCADE;
DROP TYPE IF EXISTS job_type CASCADE;
DROP TYPE IF EXISTS job_priority CASCADE;
DROP TYPE IF EXISTS job_status CASCADE;
DROP TYPE IF EXISTS customer_status CASCADE;
DROP TYPE IF EXISTS customer_type CASCADE;
