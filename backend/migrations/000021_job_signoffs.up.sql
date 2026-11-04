-- 000021_job_signoffs (T176)
-- The customer's sign-off on a visit: who signed (name, optional title and
-- email), the signature image, when, and from which device (IP through
-- middleware.ClientIP, user agent, the signed-in user who held the phone).
-- The service report PDF built from the job is stored under report_key once
-- generated; report_emailed_at / report_recipients record when and to whom
-- it was emailed.
--
-- One sign-off per job (UNIQUE (tenant_id, job_id)). Signing again replaces
-- it (the old signature object is removed) until the report has been
-- emailed; after that the sign-off is the record the customer received and
-- is locked (the API answers 409).
--
-- Emailing is idempotent: report_email_claimed_at is claimed with a
-- conditional UPDATE before sending, so a retried or doubled request sends
-- at most once; a resend is an explicit action.
--
-- Tenant-owned: tenant_id NOT NULL, a (tenant_id, job_id) index (the unique
-- constraint), the RLS policy of 000007 and an entry in tenancy.DefaultTables.

CREATE TABLE IF NOT EXISTS job_signoffs (
    id                     UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id              UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    job_id                 UUID NOT NULL REFERENCES jobs(id) ON DELETE CASCADE,
    signer_name            VARCHAR(100) NOT NULL,
    signer_title           VARCHAR(100),
    signer_email           VARCHAR(255),
    signature_key          VARCHAR(500) NOT NULL,
    signature_width        INT NOT NULL,
    signature_height       INT NOT NULL,
    signed_at              TIMESTAMPTZ NOT NULL,
    ip_address             VARCHAR(45),
    user_agent             VARCHAR(500),
    captured_by            UUID REFERENCES users(id) ON DELETE SET NULL,
    report_key             VARCHAR(500),
    report_generated_at    TIMESTAMPTZ,
    report_email_claimed_at TIMESTAMPTZ,
    report_emailed_at      TIMESTAMPTZ,
    report_recipients      TEXT,
    report_email_count     INT NOT NULL DEFAULT 0,
    created_at             TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at             TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT uq_job_signoffs_tenant_job UNIQUE (tenant_id, job_id)
);

ALTER TABLE job_signoffs ENABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS tenant_isolation ON job_signoffs;
CREATE POLICY tenant_isolation ON job_signoffs
    USING (app_rls_bypass() OR tenant_id = app_current_tenant())
    WITH CHECK (app_rls_bypass() OR tenant_id = app_current_tenant());
