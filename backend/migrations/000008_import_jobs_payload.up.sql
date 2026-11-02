-- 000008_import_jobs_payload (T241)
-- Queued CSV imports carry their parsed rows and options in the database so
-- POST /internal/jobs/imports can claim and run them (Redis is optional in
-- production, so the old Redis queue was never drained). The payload is
-- cleared once the job finishes, so customer data does not linger here.

ALTER TABLE import_jobs ADD COLUMN IF NOT EXISTS payload JSONB;

-- The job runner claims the oldest pending job across all tenants.
CREATE INDEX IF NOT EXISTS idx_import_jobs_pending
    ON import_jobs (created_at) WHERE status = 'pending';
