-- 000004_email_suppressions (T159)
-- Addresses we must not email again: a permanent bounce or a spam complaint
-- reported by Resend's signed webhook. Platform-wide, not tenant-owned: the
-- address itself is undeliverable (or has complained) whichever tenant sent.
-- Inserts are idempotent (ON CONFLICT DO NOTHING on email).

CREATE TABLE IF NOT EXISTS email_suppressions (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email      TEXT NOT NULL,
    reason     VARCHAR(32) NOT NULL CHECK (reason IN ('bounced', 'complained')),
    event_id   TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_email_suppressions_email ON email_suppressions (lower(email));
