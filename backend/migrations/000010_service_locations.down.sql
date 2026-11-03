DROP INDEX IF EXISTS idx_jobs_tenant_site;
ALTER TABLE jobs DROP COLUMN IF EXISTS site_id;
DROP TABLE IF EXISTS service_locations;
