DROP INDEX IF EXISTS idx_jobs_quote_id_key;
ALTER TABLE jobs DROP COLUMN IF EXISTS quote_id;
