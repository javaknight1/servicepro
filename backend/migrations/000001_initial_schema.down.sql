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
