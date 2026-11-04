-- 000022_import_mappers (T193)
-- Imports from Jobber / Housecall Pro exports and mapped CSV/XLSX files.
--
-- import_jobs gains what a mapped import needs: the source and entity it
-- imports, a skipped-row count (rows already imported, or invoices already
-- paid) and an attempt counter. A mapped import runs in time-boxed chunks:
-- processed_rows is its cursor, and a job interrupted mid-chunk goes back to
-- pending and resumes (at most 3 attempts) instead of failing, because every
-- mapped row is idempotent (see import_external_refs).
--
-- import_external_refs records which tenant row each imported external
-- record became: (tenant, source, entity, external_id) is unique, so
-- importing the same file twice finds the rows it created instead of making
-- duplicates. external_id is the old system's ID (Jobber J-ID, Housecall Pro
-- customer ID, job/invoice number) or, when the file has none, a natural
-- key (e.g. the customer's email).
--
-- Tenant-owned: tenant_id NOT NULL, a (tenant_id, ...) unique index, the RLS
-- policy of 000007 and an entry in tenancy.DefaultTables.

ALTER TABLE import_jobs ADD COLUMN IF NOT EXISTS source VARCHAR(30) NOT NULL DEFAULT 'servicepro';
ALTER TABLE import_jobs ADD COLUMN IF NOT EXISTS entity VARCHAR(20) NOT NULL DEFAULT 'customers';
ALTER TABLE import_jobs ADD COLUMN IF NOT EXISTS skipped_rows INT NOT NULL DEFAULT 0;
ALTER TABLE import_jobs ADD COLUMN IF NOT EXISTS attempts INT NOT NULL DEFAULT 0;
CREATE INDEX IF NOT EXISTS idx_import_jobs_tenant_created ON import_jobs (tenant_id, created_at DESC);

CREATE TABLE IF NOT EXISTS import_external_refs (
    id            UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id     UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    source        VARCHAR(30)  NOT NULL,
    entity        VARCHAR(20)  NOT NULL,
    external_id   VARCHAR(255) NOT NULL,
    record_id     UUID NOT NULL,
    import_job_id UUID REFERENCES import_jobs(id) ON DELETE SET NULL,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT uq_import_external_refs UNIQUE (tenant_id, source, entity, external_id)
);

CREATE INDEX IF NOT EXISTS idx_import_external_refs_record
    ON import_external_refs (tenant_id, entity, record_id);

ALTER TABLE import_external_refs ENABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS tenant_isolation ON import_external_refs;
CREATE POLICY tenant_isolation ON import_external_refs
    USING (app_rls_bypass() OR tenant_id = app_current_tenant())
    WITH CHECK (app_rls_bypass() OR tenant_id = app_current_tenant());
