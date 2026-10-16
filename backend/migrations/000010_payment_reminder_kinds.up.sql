-- ============================================================================
-- 000010_payment_reminder_kinds (T121 / audit 7.M8)
--
-- Manual reminders get their own numbering so they no longer take the
-- automatic schedule's (invoice_id, reminder_number) slots: a manual reminder
-- sent before day 7 used to become "#1" and the automatic first reminder was
-- skipped forever.
-- ============================================================================

ALTER TABLE payment_reminders ADD COLUMN IF NOT EXISTS is_manual BOOLEAN NOT NULL DEFAULT false;

ALTER TABLE payment_reminders DROP CONSTRAINT IF EXISTS unique_reminder_per_invoice;

CREATE UNIQUE INDEX IF NOT EXISTS uq_payment_reminders_invoice_kind_number
    ON payment_reminders(invoice_id, is_manual, reminder_number);
