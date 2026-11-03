DROP INDEX IF EXISTS idx_jobs_tenant_agreement;
ALTER TABLE jobs DROP COLUMN IF EXISTS agreement_id;
DROP TABLE IF EXISTS agreement_visits;
DROP TABLE IF EXISTS service_agreements;
