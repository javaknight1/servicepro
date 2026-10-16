-- ============================================================================
-- 000002_fix_ar_aging_summary_view
--
-- The v_ar_aging_summary view in the original 001_schema.sql never applied:
-- its ORDER BY used the output alias age_bucket inside a CASE expression,
-- which Postgres rejects, and psql simply skipped the error. 000001 now
-- carries the corrected definition, but databases adopted at the baseline
-- (built from the old file) still lack the view. This (re)creates it;
-- CREATE OR REPLACE makes it a no-op for databases built from 000001.
-- ============================================================================

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
