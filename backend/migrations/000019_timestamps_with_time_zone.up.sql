-- T273: every timestamp column carries a time zone.
--
-- 57 columns from the initial schema were `timestamp without time zone`. pgx
-- writes a Go time.Time into such a column as its *local* wall clock and reads
-- it back as UTC, so an API process running outside UTC (e.g. a laptop in US
-- Eastern) stored job notes, jobs, invoices, ... hours off. Production runs in
-- UTC, so every value already stored there is a UTC wall clock: converting with
-- `AT TIME ZONE 'UTC'` keeps each instant. The other timestamp columns were
-- already timestamptz.
--
-- Five analytics views read some of these columns; Postgres won't change a
-- column's type under a view, so they are dropped and recreated unchanged.
-- Only columns still without a time zone are converted, so a re-run is a no-op.

DROP VIEW IF EXISTS v_platform_overview;
DROP VIEW IF EXISTS v_feature_adoption;
DROP VIEW IF EXISTS v_job_metrics_daily;
DROP VIEW IF EXISTS v_invoice_metrics_daily;
DROP VIEW IF EXISTS v_ar_aging_summary;

DO $$
DECLARE
    col text;
BEGIN
    FOREACH col IN ARRAY ARRAY[
        'import_errors.created_at',
        'import_jobs.completed_at',
        'import_jobs.created_at',
        'import_jobs.started_at',
        'import_jobs.updated_at',
        'invoice_audit_log.created_at',
        'invoice_lines.created_at',
        'invoice_lines.updated_at',
        'invoice_payments.created_at',
        'invoice_payments.updated_at',
        'invoice_refunds.created_at',
        'invoice_refunds.updated_at',
        'invoice_templates.created_at',
        'invoice_templates.updated_at',
        'invoices.created_at',
        'invoices.deleted_at',
        'invoices.paid_date',
        'invoices.payment_token_expires_at',
        'invoices.payment_token_voided_at',
        'invoices.sent_date',
        'invoices.updated_at',
        'invoices.viewed_date',
        'job_assignments.created_at',
        'job_assignments.deleted_at',
        'job_assignments.updated_at',
        'job_materials.created_at',
        'job_materials.deleted_at',
        'job_materials.updated_at',
        'job_notes.created_at',
        'job_notes.deleted_at',
        'job_notes.updated_at',
        'jobs.created_at',
        'jobs.deleted_at',
        'jobs.updated_at',
        'payment_reminders.created_at',
        'payment_reminders.deleted_at',
        'payment_reminders.updated_at',
        'payment_terms.created_at',
        'payment_terms.updated_at',
        'payments.created_at',
        'payments.updated_at',
        'permissions.created_at',
        'permissions.deleted_at',
        'permissions.updated_at',
        'role_permissions.granted_at',
        'roles.created_at',
        'roles.deleted_at',
        'roles.updated_at',
        'tax_rates.created_at',
        'tax_rates.updated_at',
        'user_roles.assigned_at',
        'user_roles.expires_at',
        'users.created_at',
        'users.last_failed_login_at',
        'users.locked_until',
        'users.updated_at',
        'users.verification_sent_at'
    ] LOOP
        IF EXISTS (
            SELECT 1 FROM information_schema.columns
            WHERE table_schema = current_schema()
              AND table_name = split_part(col, '.', 1)
              AND column_name = split_part(col, '.', 2)
              AND data_type = 'timestamp without time zone'
        ) THEN
            EXECUTE format(
                'ALTER TABLE %I ALTER COLUMN %I TYPE timestamptz USING %I AT TIME ZONE ''UTC''',
                split_part(col, '.', 1), split_part(col, '.', 2), split_part(col, '.', 2));
        END IF;
    END LOOP;
END $$;

CREATE OR REPLACE VIEW v_platform_overview AS
SELECT
    (SELECT COUNT(*) FROM tenants WHERE deleted_at IS NULL) AS total_tenants,
    (SELECT COUNT(*) FROM tenants WHERE deleted_at IS NULL AND is_active = true) AS active_tenants,
    (SELECT COUNT(*) FROM users WHERE deleted_at IS NULL) AS total_users,
    (SELECT COUNT(*) FROM jobs WHERE deleted_at IS NULL) AS total_jobs,
    (SELECT COUNT(*) FROM quotes WHERE deleted_at IS NULL) AS total_quotes,
    (SELECT COUNT(*) FROM invoices WHERE deleted_at IS NULL) AS total_invoices;

CREATE OR REPLACE VIEW v_feature_adoption AS
SELECT
    t.id AS tenant_id,
    t.name AS tenant_name,
    t.created_at AS tenant_created_at,
    COALESCE(j.job_count, 0) AS job_count,
    COALESCE(q.quote_count, 0) AS quote_count,
    COALESCE(i.invoice_count, 0) AS invoice_count,
    COALESCE(p.payment_count, 0) AS payment_count
FROM tenants t
LEFT JOIN (
    SELECT tenant_id, COUNT(*) AS job_count
    FROM jobs WHERE deleted_at IS NULL
    GROUP BY tenant_id
) j ON j.tenant_id = t.id
LEFT JOIN (
    SELECT tenant_id, COUNT(*) AS quote_count
    FROM quotes WHERE deleted_at IS NULL
    GROUP BY tenant_id
) q ON q.tenant_id = t.id
LEFT JOIN (
    SELECT tenant_id, COUNT(*) AS invoice_count
    FROM invoices WHERE deleted_at IS NULL
    GROUP BY tenant_id
) i ON i.tenant_id = t.id
LEFT JOIN (
    SELECT tenant_id, COUNT(*) AS payment_count
    FROM payments
    GROUP BY tenant_id
) p ON p.tenant_id = t.id
WHERE t.deleted_at IS NULL
ORDER BY t.created_at;

CREATE OR REPLACE VIEW v_job_metrics_daily AS
SELECT
    DATE(created_at) AS day,
    COUNT(*) AS jobs_created,
    COUNT(*) FILTER (WHERE status = 'completed') AS jobs_completed,
    COUNT(*) FILTER (WHERE status = 'cancelled') AS jobs_cancelled,
    COUNT(*) FILTER (WHERE status = 'in_progress') AS jobs_in_progress
FROM jobs
WHERE deleted_at IS NULL
GROUP BY DATE(created_at)
ORDER BY day;

CREATE OR REPLACE VIEW v_invoice_metrics_daily AS
SELECT
    DATE(created_at) AS day,
    COUNT(*) AS invoices_created,
    COALESCE(SUM(total_amount), 0) AS total_amount,
    COUNT(*) FILTER (WHERE status = 'paid') AS invoices_paid,
    COUNT(*) FILTER (WHERE status = 'overdue') AS invoices_overdue,
    COALESCE(SUM(total_amount) FILTER (WHERE status = 'paid'), 0) AS paid_amount
FROM invoices
WHERE deleted_at IS NULL
GROUP BY DATE(created_at)
ORDER BY day;

CREATE OR REPLACE VIEW v_ar_aging_summary AS
SELECT
    age_bucket,
    COUNT(*) AS invoice_count,
    COALESCE(SUM(amount_due), 0) AS outstanding_amount
FROM (
    SELECT
        CASE
            WHEN due_date >= CURRENT_DATE THEN 'Current'
            WHEN due_date >= CURRENT_DATE - INTERVAL '30 days' THEN '1-30 days'
            WHEN due_date >= CURRENT_DATE - INTERVAL '60 days' THEN '31-60 days'
            WHEN due_date >= CURRENT_DATE - INTERVAL '90 days' THEN '61-90 days'
            ELSE '90+ days'
        END AS age_bucket,
        amount_due
    FROM invoices
    WHERE deleted_at IS NULL
      AND status NOT IN ('paid', 'cancelled', 'refunded')
      AND amount_due > 0
) aged
GROUP BY age_bucket
ORDER BY
    CASE age_bucket
        WHEN 'Current' THEN 1
        WHEN '1-30 days' THEN 2
        WHEN '31-60 days' THEN 3
        WHEN '61-90 days' THEN 4
        WHEN '90+ days' THEN 5
    END;
