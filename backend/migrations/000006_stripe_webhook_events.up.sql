-- ============================================================================
-- 000006_stripe_webhook_events (T109 / audit F-10)
--
-- Persistent, cross-instance dedupe for POST /api/v1/stripe/webhooks.
-- The handler INSERTs the Stripe event ID with ON CONFLICT DO NOTHING before
-- processing, marks it processed afterwards, and deletes the claim when
-- processing fails so Stripe's retry is processed again.
--
-- Also: a Checkout session can only ever be applied to one invoice once.
-- ============================================================================

CREATE TABLE IF NOT EXISTS stripe_webhook_events (
    event_id VARCHAR(255) PRIMARY KEY,
    event_type VARCHAR(100) NOT NULL,
    status VARCHAR(20) NOT NULL CHECK (status IN ('processing', 'processed')),
    received_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    processed_at TIMESTAMP WITH TIME ZONE
);

CREATE INDEX IF NOT EXISTS idx_stripe_webhook_events_received ON stripe_webhook_events(received_at);

CREATE UNIQUE INDEX IF NOT EXISTS idx_invoices_stripe_checkout_session
    ON invoices(stripe_checkout_session_id)
    WHERE stripe_checkout_session_id IS NOT NULL AND stripe_checkout_session_id <> '';
