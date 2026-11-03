-- 000010_service_locations (T186)
-- Customer sites: a customer (e.g. a property manager) has many service
-- locations, each with its own address, access notes and site contact. Jobs
-- may reference one (jobs.site_id); the API checks it belongs to the job's
-- customer and tenant.
--
-- Tenant-owned: tenant_id NOT NULL, a (tenant_id, ...) index and the same RLS
-- policy as migration 000007 (tenancy.DefaultTables lists the table too).

CREATE TABLE IF NOT EXISTS service_locations (
    id             UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id      UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    customer_id    UUID NOT NULL REFERENCES customers(id) ON DELETE CASCADE,
    name           VARCHAR(200) NOT NULL,
    address_street VARCHAR(100) NOT NULL DEFAULT '',
    address_city   VARCHAR(50)  NOT NULL DEFAULT '',
    address_state  VARCHAR(2)   NOT NULL DEFAULT '',
    address_zip    VARCHAR(10)  NOT NULL DEFAULT '',
    access_notes   TEXT,
    contact_name   VARCHAR(100),
    contact_phone  VARCHAR(20),
    contact_email  VARCHAR(255),
    is_active      BOOLEAN NOT NULL DEFAULT TRUE,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at     TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_service_locations_tenant_customer
    ON service_locations (tenant_id, customer_id) WHERE deleted_at IS NULL;

ALTER TABLE service_locations ENABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS tenant_isolation ON service_locations;
CREATE POLICY tenant_isolation ON service_locations
    USING (app_rls_bypass() OR tenant_id = app_current_tenant())
    WITH CHECK (app_rls_bypass() OR tenant_id = app_current_tenant());

ALTER TABLE jobs ADD COLUMN IF NOT EXISTS site_id UUID REFERENCES service_locations(id) ON DELETE SET NULL;
CREATE INDEX IF NOT EXISTS idx_jobs_tenant_site ON jobs (tenant_id, site_id) WHERE site_id IS NOT NULL;

-- Every existing customer gets a default site from its service address (or
-- its billing address when it has none), so existing work has somewhere to
-- point. Customers created later add their sites through the API.
INSERT INTO service_locations (tenant_id, customer_id, name,
                               address_street, address_city, address_state, address_zip)
SELECT c.tenant_id, c.id,
       LEFT(COALESCE(NULLIF(c.company_name, ''), c.first_name || ' ' || c.last_name), 200),
       COALESCE(c.service_address_street, c.billing_address_street),
       COALESCE(c.service_address_city, c.billing_address_city),
       TRIM(COALESCE(c.service_address_state, c.billing_address_state)),
       COALESCE(c.service_address_zip, c.billing_address_zip)
FROM customers c
WHERE c.deleted_at IS NULL
  AND NOT EXISTS (SELECT 1 FROM service_locations s WHERE s.customer_id = c.id);
