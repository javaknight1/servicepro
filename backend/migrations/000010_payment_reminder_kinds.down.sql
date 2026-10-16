DROP INDEX IF EXISTS uq_payment_reminders_invoice_kind_number;

-- Manual and automatic rows may share a number once the kinds are split;
-- renumber manual rows above the automatic range before restoring the old
-- constraint.
UPDATE payment_reminders SET reminder_number = reminder_number + 1000 WHERE is_manual;

ALTER TABLE payment_reminders DROP CONSTRAINT IF EXISTS unique_reminder_per_invoice;
ALTER TABLE payment_reminders ADD CONSTRAINT unique_reminder_per_invoice UNIQUE (invoice_id, reminder_number);

ALTER TABLE payment_reminders DROP COLUMN IF EXISTS is_manual;
