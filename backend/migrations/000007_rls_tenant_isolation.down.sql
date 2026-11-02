-- 000007_rls_tenant_isolation (T154), reversed.

DO $$
DECLARE
    t text;
BEGIN
    FOREACH t IN ARRAY ARRAY[
        'customers', 'jobs', 'quotes', 'invoices', 'payments', 'schedules',
        'recurring_schedules', 'export_jobs', 'import_jobs', 'payment_reminders',
        'revenue_transactions', 'revenue_summary_cache',
        'payment_terms', 'tax_rates', 'invoice_templates'
    ] LOOP
        EXECUTE format('DROP POLICY IF EXISTS tenant_isolation ON %I', t);
        EXECUTE format('DROP POLICY IF EXISTS tenant_shared_read ON %I', t);
        EXECUTE format('ALTER TABLE %I DISABLE ROW LEVEL SECURITY', t);
    END LOOP;
END
$$;

DROP FUNCTION IF EXISTS app_current_tenant();
DROP FUNCTION IF EXISTS app_rls_bypass();

DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'servicepro_app') THEN
        ALTER DEFAULT PRIVILEGES IN SCHEMA public REVOKE ALL ON TABLES FROM servicepro_app;
        ALTER DEFAULT PRIVILEGES IN SCHEMA public REVOKE ALL ON SEQUENCES FROM servicepro_app;
        REVOKE ALL ON ALL TABLES IN SCHEMA public FROM servicepro_app;
        REVOKE ALL ON ALL SEQUENCES IN SCHEMA public FROM servicepro_app;
        REVOKE ALL ON SCHEMA public FROM servicepro_app;
        -- The role is cluster-wide: other databases (or login roles that are
        -- members of it) may still depend on it. Keep it in that case.
        BEGIN
            DROP ROLE servicepro_app;
        EXCEPTION WHEN dependent_objects_still_exist THEN
            RAISE NOTICE 'servicepro_app still has privileges elsewhere; role kept';
        END;
    END IF;
END
$$;
