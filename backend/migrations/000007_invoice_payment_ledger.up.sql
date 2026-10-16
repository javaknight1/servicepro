-- ============================================================================
-- 000007_invoice_payment_ledger (T118 / correctness audit 7.C1, 7.H4)
--
-- Every payment (manual or Stripe) is now a row in invoice_payments, written
-- in the same transaction that updates invoices.amount_paid / status under a
-- row lock. Stripe payments carry their checkout session and payment intent
-- so a session can be applied at most once and refunds (T125) can find the
-- payment they reverse. Webhook-recorded payments have no staff user, so
-- created_by becomes nullable.
-- ============================================================================

ALTER TABLE invoice_payments ADD COLUMN IF NOT EXISTS stripe_checkout_session_id VARCHAR(255);
ALTER TABLE invoice_payments ADD COLUMN IF NOT EXISTS stripe_payment_intent_id VARCHAR(255);
ALTER TABLE invoice_payments ALTER COLUMN created_by DROP NOT NULL;

CREATE UNIQUE INDEX IF NOT EXISTS idx_invoice_payments_stripe_session
    ON invoice_payments(stripe_checkout_session_id)
    WHERE stripe_checkout_session_id IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_invoice_payments_stripe_payment_intent
    ON invoice_payments(stripe_payment_intent_id)
    WHERE stripe_payment_intent_id IS NOT NULL;

-- Backfill: Stripe payments applied before this migration only updated the
-- invoice. Give each such invoice its ledger row so amount_paid = SUM(payments).
INSERT INTO invoice_payments (invoice_id, amount, payment_date, payment_method, reference_number,
                              notes, stripe_checkout_session_id, stripe_payment_intent_id)
SELECT i.id,
       i.amount_paid - COALESCE((SELECT SUM(p.amount) FROM invoice_payments p WHERE p.invoice_id = i.id), 0),
       COALESCE(i.paid_date::date, CURRENT_DATE),
       'stripe',
       NULLIF(i.stripe_payment_intent_id, ''),
       'Backfilled by migration 000007',
       i.stripe_checkout_session_id,
       NULLIF(i.stripe_payment_intent_id, '')
FROM invoices i
WHERE i.stripe_checkout_session_id IS NOT NULL AND i.stripe_checkout_session_id <> ''
  AND i.amount_paid > COALESCE((SELECT SUM(p.amount) FROM invoice_payments p WHERE p.invoice_id = i.id), 0)
ON CONFLICT DO NOTHING;
