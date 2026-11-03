-- 000012_service_agreements (T167)
-- Service agreements: recurring work at one customer site (frequency, price,
-- optional technician). POST /internal/jobs/agreement-visits generates a
-- scheduled job per upcoming visit (14-day horizon, tenant time zone).
--
-- Decision: a new, properly modelled table instead of building on the
-- legacy recurring_schedules (epoch BIGINT dates, UTC time-of-day integers,
-- never wired into the binary). recurring_schedules already has tenant_id
-- NOT NULL, an index and the RLS policy (000001, 000007); it is left as is.
--
-- agreement_visits claims each (agreement, date) once: the generator
-- INSERTs ... ON CONFLICT DO NOTHING and creates the job only when its
-- insert won, in the same transaction, so reruns and concurrent runs never
-- duplicate a visit.
--
-- Both tables are tenant-owned (tenant_id NOT NULL, (tenant_id, ...)
-- indexes, the RLS policy of 000007, tenancy.DefaultTables).

CREATE TABLE IF NOT EXISTS service_agreements (
    id                     UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id              UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    customer_id            UUID NOT NULL REFERENCES customers(id) ON DELETE CASCADE,
    site_id                UUID NOT NULL REFERENCES service_locations(id) ON DELETE RESTRICT,
    title                  VARCHAR(200) NOT NULL,
    frequency              VARCHAR(20) NOT NULL
        CHECK (frequency IN ('weekdays', 'weekly', 'times_per_week', 'monthly')),
    weekday_mask           SMALLINT NOT NULL DEFAULT 0 CHECK (weekday_mask BETWEEN 0 AND 127),
    times_per_week         SMALLINT CHECK (times_per_week IS NULL OR times_per_week BETWEEN 1 AND 7),
    day_of_month           SMALLINT CHECK (day_of_month IS NULL OR day_of_month BETWEEN 1 AND 31),
    start_date             DATE NOT NULL,
    end_date               DATE,
    visit_start_minutes    INTEGER NOT NULL DEFAULT 1080 CHECK (visit_start_minutes BETWEEN 0 AND 1439),
    visit_duration_minutes INTEGER NOT NULL DEFAULT 60 CHECK (visit_duration_minutes BETWEEN 1 AND 1440),
    price_per_period       NUMERIC(12,2) NOT NULL CHECK (price_per_period >= 0),
    billing_period         VARCHAR(10) NOT NULL DEFAULT 'month' CHECK (billing_period IN ('month', 'visit')),
    assigned_technician_id UUID REFERENCES users(id) ON DELETE SET NULL,
    job_type               VARCHAR(50) NOT NULL DEFAULT 'maintenance',
    status                 VARCHAR(10) NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'paused')),
    notes                  TEXT,
    created_by             UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    created_at             TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at             TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at             TIMESTAMPTZ,
    CONSTRAINT service_agreements_end_after_start CHECK (end_date IS NULL OR end_date >= start_date)
);

CREATE INDEX IF NOT EXISTS idx_service_agreements_tenant_status
    ON service_agreements (tenant_id, status) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_service_agreements_tenant_customer
    ON service_agreements (tenant_id, customer_id) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_service_agreements_tenant_site
    ON service_agreements (tenant_id, site_id) WHERE deleted_at IS NULL;

CREATE TABLE IF NOT EXISTS agreement_visits (
    id           UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id    UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    agreement_id UUID NOT NULL REFERENCES service_agreements(id) ON DELETE CASCADE,
    visit_date   DATE NOT NULL,
    job_id       UUID REFERENCES jobs(id) ON DELETE SET NULL,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT agreement_visits_agreement_date_key UNIQUE (agreement_id, visit_date)
);

CREATE INDEX IF NOT EXISTS idx_agreement_visits_tenant_date ON agreement_visits (tenant_id, visit_date);

ALTER TABLE jobs ADD COLUMN IF NOT EXISTS agreement_id UUID REFERENCES service_agreements(id) ON DELETE SET NULL;
CREATE INDEX IF NOT EXISTS idx_jobs_tenant_agreement ON jobs (tenant_id, agreement_id) WHERE agreement_id IS NOT NULL;

DO $$
DECLARE
    t text;
BEGIN
    FOREACH t IN ARRAY ARRAY['service_agreements', 'agreement_visits'] LOOP
        EXECUTE format('ALTER TABLE %I ENABLE ROW LEVEL SECURITY', t);
        EXECUTE format('DROP POLICY IF EXISTS tenant_isolation ON %I', t);
        EXECUTE format(
            'CREATE POLICY tenant_isolation ON %I
                USING (app_rls_bypass() OR tenant_id = app_current_tenant())
                WITH CHECK (app_rls_bypass() OR tenant_id = app_current_tenant())', t);
    END LOOP;
END
$$;
