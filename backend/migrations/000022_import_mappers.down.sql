-- 000022_import_mappers (T193), reversed.
DROP TABLE IF EXISTS import_external_refs;
DROP INDEX IF EXISTS idx_import_jobs_tenant_created;
ALTER TABLE import_jobs DROP COLUMN IF EXISTS attempts;
ALTER TABLE import_jobs DROP COLUMN IF EXISTS skipped_rows;
ALTER TABLE import_jobs DROP COLUMN IF EXISTS entity;
ALTER TABLE import_jobs DROP COLUMN IF EXISTS source;
