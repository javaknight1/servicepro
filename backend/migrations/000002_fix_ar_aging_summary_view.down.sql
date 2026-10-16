-- Intentionally a no-op: 000001_initial_schema also creates
-- v_ar_aging_summary (and its down migration drops it), so rolling back to
-- version 1 must keep the view.
SELECT 1;
