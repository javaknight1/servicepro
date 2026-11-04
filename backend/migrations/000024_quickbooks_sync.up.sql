-- 000024_quickbooks_sync (T192)
-- One-way sync of customers, invoices and payments to a tenant's QuickBooks
-- Online company.
--
-- quickbooks_connections: one per tenant (tenant_id unique). The Intuit
-- OAuth tokens are stored encrypted (AES-256-GCM, pkg/secretbox, key from
-- INTEGRATIONS_ENCRYPTION_KEY, bound to the tenant): access_token_enc and
-- refresh_token_enc are never plaintext. Access tokens live 1 hour; refresh
-- tokens roll (each refresh returns a new one, valid ~100 days). A refresh
-- runs under SELECT ... FOR UPDATE on this row so concurrent refreshes can't
-- clobber each other's token. oauth_state_hash is the SHA-256 of the pending
-- connect request's single-use nonce (10 minute TTL).
-- Mapping settings (income account → item, tax code, deposit account) are
-- QuickBooks IDs with their display names.
--
-- quickbooks_sync_map: one row per synced record (tenant, entity, local id)
-- and its QuickBooks Id/SyncToken. It is also the work queue: status
-- pending rows are pushed by POST /internal/jobs/quickbooks-sync (claimed
-- with FOR UPDATE SKIP LOCKED), errors keep last_error and back off via
-- next_attempt_at. request_id is the Intuit requestid of the in-flight
-- create, reused on retries so a create that reached Intuit but whose answer
-- was lost is not duplicated. payload_hash skips unchanged updates.
--
-- Tenant-owned: tenant_id NOT NULL, (tenant_id, ...) indexes, the RLS policy
-- of 000007 and entries in tenancy.DefaultTables.
--
-- New permission integrations.manage (owners and admins: super_admin,
-- admin) gates connecting, configuring and syncing.

CREATE TABLE IF NOT EXISTS quickbooks_connections (
    id                       UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id                UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    status                   VARCHAR(20) NOT NULL DEFAULT 'disconnected'
                             CHECK (status IN ('connected', 'reauth_required', 'disconnected')),
    environment              VARCHAR(20) NOT NULL DEFAULT 'sandbox',
    realm_id                 VARCHAR(50),
    company_name             VARCHAR(255),
    access_token_enc         TEXT,
    access_token_expires_at  TIMESTAMPTZ,
    refresh_token_enc        TEXT,
    refresh_token_expires_at TIMESTAMPTZ,
    connected_by             UUID REFERENCES users(id) ON DELETE SET NULL,
    connected_at             TIMESTAMPTZ,
    disconnected_at          TIMESTAMPTZ,
    sync_from                DATE,
    oauth_state_hash         VARCHAR(64),
    oauth_state_expires_at   TIMESTAMPTZ,
    income_account_id        VARCHAR(50),
    income_account_name      VARCHAR(255),
    item_id                  VARCHAR(50),
    item_name                VARCHAR(255),
    tax_code_id              VARCHAR(50),
    tax_code_name            VARCHAR(255),
    deposit_account_id       VARCHAR(50),
    deposit_account_name     VARCHAR(255),
    last_sync_at             TIMESTAMPTZ,
    last_sync_error          TEXT,
    created_at               TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at               TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT uq_quickbooks_connections_tenant UNIQUE (tenant_id)
);

CREATE INDEX IF NOT EXISTS idx_quickbooks_connections_status
    ON quickbooks_connections (status, tenant_id);

ALTER TABLE quickbooks_connections ENABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS tenant_isolation ON quickbooks_connections;
CREATE POLICY tenant_isolation ON quickbooks_connections
    USING (app_rls_bypass() OR tenant_id = app_current_tenant())
    WITH CHECK (app_rls_bypass() OR tenant_id = app_current_tenant());

CREATE TABLE IF NOT EXISTS quickbooks_sync_map (
    id               UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id        UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    entity           VARCHAR(20) NOT NULL CHECK (entity IN ('customer', 'invoice', 'payment')),
    local_id         UUID NOT NULL,
    qbo_id           VARCHAR(50),
    sync_token       VARCHAR(20),
    status           VARCHAR(20) NOT NULL DEFAULT 'pending'
                     CHECK (status IN ('pending', 'synced', 'error')),
    attempts         INT NOT NULL DEFAULT 0,
    next_attempt_at  TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    claimed_at       TIMESTAMPTZ,
    request_id       VARCHAR(50),
    payload_hash     VARCHAR(64),
    local_updated_at TIMESTAMPTZ,
    last_synced_at   TIMESTAMPTZ,
    last_error       TEXT,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT uq_quickbooks_sync_map UNIQUE (tenant_id, entity, local_id)
);

CREATE INDEX IF NOT EXISTS idx_quickbooks_sync_map_queue
    ON quickbooks_sync_map (tenant_id, status, next_attempt_at);

ALTER TABLE quickbooks_sync_map ENABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS tenant_isolation ON quickbooks_sync_map;
CREATE POLICY tenant_isolation ON quickbooks_sync_map
    USING (app_rls_bypass() OR tenant_id = app_current_tenant())
    WITH CHECK (app_rls_bypass() OR tenant_id = app_current_tenant());

INSERT INTO permissions (id, name, description, resource, action) VALUES
    ('51000000-0000-0000-0000-000000000001', 'integrations.manage', 'Connect, configure and sync accounting integrations (QuickBooks)', 'integrations', 'manage')
ON CONFLICT (name) DO NOTHING;

INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM roles r CROSS JOIN permissions p
WHERE r.name IN ('super_admin', 'admin') AND p.name = 'integrations.manage'
ON CONFLICT (role_id, permission_id) DO NOTHING;
