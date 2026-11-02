-- 000007_rls_tenant_isolation (T154)
-- Postgres row-level security as a backstop for tenant isolation: even if
-- application code forgets a tenant filter, the database refuses to return
-- or modify another tenant's rows.
--
-- Policies read two transaction-local settings that the API sets at the
-- start of every transaction (internal/tenancy/rls.go, RLS_ENFORCE=true):
--   app.tenant_id   the caller's tenant
--   app.bypass_rls  'on' for trusted system code (webhooks, internal jobs,
--                   public token routes)
-- Neither set means no tenant rows at all (fail closed).
--
-- Tables are ENABLE (not FORCE) ROW LEVEL SECURITY: the tables' owner
-- (migrations, pg_dump backups, seeds) and superusers are exempt. The API is
-- bound by the policies when it connects as a login role that is a member of
-- servicepro_app (created here, NOLOGIN, without BYPASSRLS).
--
-- The tables are exactly tenancy.DefaultTables(); a test keeps them in sync.

DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'servicepro_app') THEN
        CREATE ROLE servicepro_app NOLOGIN NOBYPASSRLS;
    END IF;
END
$$;

GRANT USAGE ON SCHEMA public TO servicepro_app;
GRANT SELECT, INSERT, UPDATE, DELETE ON ALL TABLES IN SCHEMA public TO servicepro_app;
GRANT USAGE, SELECT, UPDATE ON ALL SEQUENCES IN SCHEMA public TO servicepro_app;
-- Tables and sequences created by later migrations (run as the owner).
ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT SELECT, INSERT, UPDATE, DELETE ON TABLES TO servicepro_app;
ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT USAGE, SELECT, UPDATE ON SEQUENCES TO servicepro_app;

-- Helpers the policies call. Plain STABLE SQL functions are inlined by the
-- planner, so tenant_id = app_current_tenant() can still use the
-- (tenant_id, ...) indexes.
CREATE OR REPLACE FUNCTION app_current_tenant() RETURNS uuid
    LANGUAGE sql STABLE
    AS $$ SELECT NULLIF(current_setting('app.tenant_id', true), '')::uuid $$;

CREATE OR REPLACE FUNCTION app_rls_bypass() RETURNS boolean
    LANGUAGE sql STABLE
    AS $$ SELECT COALESCE(current_setting('app.bypass_rls', true), '') = 'on' $$;

-- Owned tables: every row belongs to exactly one tenant.
DO $$
DECLARE
    t text;
BEGIN
    FOREACH t IN ARRAY ARRAY[
        'customers', 'jobs', 'quotes', 'invoices', 'payments', 'schedules',
        'recurring_schedules', 'export_jobs', 'import_jobs', 'payment_reminders',
        'revenue_transactions', 'revenue_summary_cache'
    ] LOOP
        EXECUTE format('ALTER TABLE %I ENABLE ROW LEVEL SECURITY', t);
        EXECUTE format('DROP POLICY IF EXISTS tenant_isolation ON %I', t);
        EXECUTE format(
            'CREATE POLICY tenant_isolation ON %I
                USING (app_rls_bypass() OR tenant_id = app_current_tenant())
                WITH CHECK (app_rls_bypass() OR tenant_id = app_current_tenant())', t);
    END LOOP;
END
$$;

-- SharedRead tables: tenant rows plus global defaults (tenant_id IS NULL).
-- Everyone reads the defaults; only system code may change them.
DO $$
DECLARE
    t text;
BEGIN
    FOREACH t IN ARRAY ARRAY['payment_terms', 'tax_rates', 'invoice_templates'] LOOP
        EXECUTE format('ALTER TABLE %I ENABLE ROW LEVEL SECURITY', t);
        EXECUTE format('DROP POLICY IF EXISTS tenant_isolation ON %I', t);
        EXECUTE format('DROP POLICY IF EXISTS tenant_shared_read ON %I', t);
        EXECUTE format(
            'CREATE POLICY tenant_isolation ON %I
                USING (app_rls_bypass() OR tenant_id = app_current_tenant())
                WITH CHECK (app_rls_bypass() OR tenant_id = app_current_tenant())', t);
        EXECUTE format(
            'CREATE POLICY tenant_shared_read ON %I FOR SELECT
                USING (tenant_id IS NULL)', t);
    END LOOP;
END
$$;
