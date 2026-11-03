-- 000009_stripe_connected_accounts (T169)
-- One Stripe Connect account (Accounts v2, direct charges) per tenant: the
-- Stripe account ID is created by the server and never accepted from a
-- client. Capability and requirement statuses are cached from the v2 account
-- (re-fetched from Stripe on webhooks and on the settings page) so pay links
-- can be gated without calling Stripe.

CREATE TABLE IF NOT EXISTS stripe_connected_accounts (
    id                   UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id            UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    stripe_account_id    TEXT NOT NULL,
    livemode             BOOLEAN NOT NULL DEFAULT false,
    -- v2 configuration.merchant.capabilities.*.status: '' (not requested yet),
    -- active, pending, restricted, rejected, unsupported
    card_payments_status TEXT NOT NULL DEFAULT '',
    ach_payments_status  TEXT NOT NULL DEFAULT '',
    payouts_status       TEXT NOT NULL DEFAULT '',
    -- v2 requirements.summary.minimum_deadline.status: '' (nothing due),
    -- eventually_due, currently_due, past_due
    requirements_status  TEXT NOT NULL DEFAULT '',
    synced_at            TIMESTAMPTZ,
    created_at           TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at           TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- One account per tenant; doubles as the (tenant_id) lookup index.
CREATE UNIQUE INDEX IF NOT EXISTS idx_stripe_connected_accounts_tenant ON stripe_connected_accounts (tenant_id);
-- Webhooks resolve the tenant from the event's account.
CREATE UNIQUE INDEX IF NOT EXISTS idx_stripe_connected_accounts_account ON stripe_connected_accounts (stripe_account_id);

-- Row-level security backstop (T154, same policy as 000007): the table is in
-- tenancy.DefaultTables(). Webhooks resolve accounts as system code
-- (app.bypass_rls), then act as the owning tenant.
ALTER TABLE stripe_connected_accounts ENABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS tenant_isolation ON stripe_connected_accounts;
CREATE POLICY tenant_isolation ON stripe_connected_accounts
    USING (app_rls_bypass() OR tenant_id = app_current_tenant())
    WITH CHECK (app_rls_bypass() OR tenant_id = app_current_tenant());
