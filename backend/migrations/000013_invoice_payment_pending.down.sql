ALTER TABLE invoices DROP COLUMN IF EXISTS payment_pending_session_id;
ALTER TABLE invoices DROP COLUMN IF EXISTS payment_pending_at;
