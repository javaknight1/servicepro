-- ============================================================================
-- 000005_user_token_version (T110 / audit F-11)
--
-- Every JWT carries the user's token_version ("tv" claim). Logout and
-- password reset increment it, which revokes all outstanding access and
-- refresh tokens for that user.
-- ============================================================================

ALTER TABLE users ADD COLUMN IF NOT EXISTS token_version INTEGER NOT NULL DEFAULT 0;
