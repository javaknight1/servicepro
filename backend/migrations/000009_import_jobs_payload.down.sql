DROP INDEX IF EXISTS idx_import_jobs_pending;
ALTER TABLE import_jobs DROP COLUMN IF EXISTS payload;
