-- 000013_invoice_payment_pending (T170)
-- A bank (ACH) payment completes Checkout days before the money settles.
-- While it settles the invoice is "payment pending": reminders skip it and
-- the pay link can't start a second payment. A column rather than a new
-- invoice_status value: the status machine is unchanged, and a failed bank
-- payment simply clears it, leaving the invoice in the status it had.
-- payment_pending_session_id is the Checkout Session that set it, so only
-- that session's success or failure clears it (never sent to clients).

ALTER TABLE invoices ADD COLUMN IF NOT EXISTS payment_pending_at TIMESTAMPTZ;
ALTER TABLE invoices ADD COLUMN IF NOT EXISTS payment_pending_session_id VARCHAR(255);
