-- ============================================================================
-- Stripe Integration Migration
-- Adds Stripe-specific fields and tables for payment processing
-- ============================================================================

-- ============================================================================
-- 1. ADD STRIPE CUSTOMER ID TO TENANTS
-- ============================================================================

ALTER TABLE tenants
ADD COLUMN IF NOT EXISTS stripe_customer_id VARCHAR(255) UNIQUE;

CREATE INDEX IF NOT EXISTS idx_tenants_stripe_customer_id
ON tenants(stripe_customer_id) WHERE stripe_customer_id IS NOT NULL;

-- ============================================================================
-- 2. ADD STRIPE FIELDS TO TENANT SUBSCRIPTIONS
-- ============================================================================

ALTER TABLE tenant_subscriptions
ADD COLUMN IF NOT EXISTS stripe_subscription_id VARCHAR(255) UNIQUE,
ADD COLUMN IF NOT EXISTS stripe_price_id VARCHAR(255),
ADD COLUMN IF NOT EXISTS cancel_at_period_end BOOLEAN NOT NULL DEFAULT FALSE;

CREATE INDEX IF NOT EXISTS idx_tenant_subscriptions_stripe_id
ON tenant_subscriptions(stripe_subscription_id) WHERE stripe_subscription_id IS NOT NULL;

-- ============================================================================
-- 3. PAYMENT METHODS TABLE
-- ============================================================================

CREATE TABLE IF NOT EXISTS payment_methods (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    stripe_payment_method_id VARCHAR(255) NOT NULL UNIQUE,

    -- Card details (safe to store - not sensitive)
    card_brand VARCHAR(50) NOT NULL,        -- visa, mastercard, amex, discover, etc.
    card_last_four VARCHAR(4) NOT NULL,     -- Last 4 digits
    card_exp_month INTEGER NOT NULL,        -- 1-12
    card_exp_year INTEGER NOT NULL,         -- 4-digit year

    -- Metadata
    is_default BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_payment_methods_tenant
ON payment_methods(tenant_id);

CREATE INDEX IF NOT EXISTS idx_payment_methods_default
ON payment_methods(tenant_id, is_default) WHERE is_default = TRUE;

-- Trigger to update updated_at
CREATE TRIGGER trigger_payment_methods_updated_at
BEFORE UPDATE ON payment_methods
FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- ============================================================================
-- 4. BILLING EVENTS TABLE (for history/audit)
-- ============================================================================

CREATE TABLE IF NOT EXISTS billing_events (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    stripe_event_id VARCHAR(255) UNIQUE NOT NULL,
    event_type VARCHAR(100) NOT NULL,       -- invoice.paid, payment_failed, subscription.updated, etc.
    amount_cents INTEGER,
    currency VARCHAR(3) DEFAULT 'usd',
    status VARCHAR(50) NOT NULL,            -- succeeded, failed, pending
    description TEXT,
    stripe_invoice_id VARCHAR(255),
    stripe_invoice_url TEXT,                -- Link to Stripe-hosted invoice
    metadata JSONB,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_billing_events_tenant
ON billing_events(tenant_id);

CREATE INDEX IF NOT EXISTS idx_billing_events_created
ON billing_events(created_at DESC);

CREATE INDEX IF NOT EXISTS idx_billing_events_type
ON billing_events(event_type);

-- ============================================================================
-- 5. HELPER FUNCTION: SET DEFAULT PAYMENT METHOD
-- ============================================================================

CREATE OR REPLACE FUNCTION set_default_payment_method(
    p_tenant_id UUID,
    p_payment_method_id UUID
) RETURNS VOID AS $$
BEGIN
    -- Remove default from all other payment methods for this tenant
    UPDATE payment_methods
    SET is_default = FALSE, updated_at = CURRENT_TIMESTAMP
    WHERE tenant_id = p_tenant_id AND is_default = TRUE;

    -- Set the new default
    UPDATE payment_methods
    SET is_default = TRUE, updated_at = CURRENT_TIMESTAMP
    WHERE id = p_payment_method_id AND tenant_id = p_tenant_id;
END;
$$ LANGUAGE plpgsql;

-- ============================================================================
-- 6. HELPER FUNCTION: GET DEFAULT PAYMENT METHOD
-- ============================================================================

CREATE OR REPLACE FUNCTION get_default_payment_method(p_tenant_id UUID)
RETURNS TABLE (
    id UUID,
    stripe_payment_method_id VARCHAR,
    card_brand VARCHAR,
    card_last_four VARCHAR,
    card_exp_month INTEGER,
    card_exp_year INTEGER
) AS $$
BEGIN
    RETURN QUERY
    SELECT
        pm.id,
        pm.stripe_payment_method_id,
        pm.card_brand,
        pm.card_last_four,
        pm.card_exp_month,
        pm.card_exp_year
    FROM payment_methods pm
    WHERE pm.tenant_id = p_tenant_id AND pm.is_default = TRUE
    LIMIT 1;
END;
$$ LANGUAGE plpgsql STABLE;

-- ============================================================================
-- MIGRATION COMPLETE
-- ============================================================================

DO $$
BEGIN
    RAISE NOTICE 'Stripe integration migration completed successfully.';
END $$;
