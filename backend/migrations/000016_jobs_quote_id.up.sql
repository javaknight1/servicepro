-- 000016_jobs_quote_id (T028)
-- A job converted from an accepted quote (POST /quotes/{id}/convert-to-job)
-- points back at it. The partial unique index makes the conversion happen
-- at most once per quote: a second or concurrent conversion fails on the
-- index and the API answers 409 already_converted with the job's ID.
-- Deleting the job (soft delete) frees the quote to be converted again.
-- quote_id is a globally unique UUID, so the index needs no tenant_id
-- prefix; lookups by quote are always tenant-scoped in the API.

ALTER TABLE jobs ADD COLUMN IF NOT EXISTS quote_id UUID REFERENCES quotes(id) ON DELETE SET NULL;
CREATE UNIQUE INDEX IF NOT EXISTS idx_jobs_quote_id_key ON jobs (quote_id)
    WHERE quote_id IS NOT NULL AND deleted_at IS NULL;
