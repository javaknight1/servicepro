DROP INDEX IF EXISTS idx_invoice_payments_stripe_payment_intent;
DROP INDEX IF EXISTS idx_invoice_payments_stripe_session;
-- Webhook-recorded payments have no created_by; they cannot satisfy NOT NULL.
DELETE FROM invoice_payments WHERE created_by IS NULL;
ALTER TABLE invoice_payments ALTER COLUMN created_by SET NOT NULL;
ALTER TABLE invoice_payments DROP COLUMN IF EXISTS stripe_payment_intent_id;
ALTER TABLE invoice_payments DROP COLUMN IF EXISTS stripe_checkout_session_id;
