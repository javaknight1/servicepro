-- 000020_job_attachments (T174)
-- Photos and files attached to a job: before/after shots, proof of service,
-- a site manager's work order PDF.
--
-- The bytes live in the private object store (S3-compatible: RustFS locally,
-- R2 in production) under object_key; the API never serves a public URL,
-- only short-lived presigned links. Photos are re-encoded server side (EXIF
-- stripped, orientation applied, longest side capped) and get a thumbnail at
-- thumb_key; files (PDF) are stored as uploaded and have no thumbnail.
--
-- Tenant-owned: tenant_id NOT NULL, a (tenant_id, job_id) index, the RLS
-- policy of 000007 and an entry in tenancy.DefaultTables. Deleting an
-- attachment removes the row and its objects (no soft delete: a deleted
-- photo must not linger in storage).

CREATE TABLE IF NOT EXISTS job_attachments (
    id                UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id         UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    job_id            UUID NOT NULL REFERENCES jobs(id) ON DELETE CASCADE,
    uploaded_by       UUID REFERENCES users(id) ON DELETE SET NULL,
    kind              VARCHAR(20) NOT NULL CHECK (kind IN ('photo', 'file')),
    object_key        VARCHAR(500) NOT NULL,
    thumb_key         VARCHAR(500),
    original_filename VARCHAR(255) NOT NULL DEFAULT '',
    content_type      VARCHAR(100) NOT NULL,
    size_bytes        BIGINT NOT NULL CHECK (size_bytes >= 0),
    width             INT,
    height            INT,
    caption           VARCHAR(500),
    created_at        TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_job_attachments_tenant_job
    ON job_attachments (tenant_id, job_id, created_at);

ALTER TABLE job_attachments ENABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS tenant_isolation ON job_attachments;
CREATE POLICY tenant_isolation ON job_attachments
    USING (app_rls_bypass() OR tenant_id = app_current_tenant())
    WITH CHECK (app_rls_bypass() OR tenant_id = app_current_tenant());
