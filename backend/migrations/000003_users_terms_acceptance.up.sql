-- 000003_users_terms_acceptance (T255)
-- Records which version of the Terms of Service and Privacy Policy a user
-- accepted at sign-up, and when. NULL for users created before this existed.

ALTER TABLE users ADD COLUMN IF NOT EXISTS terms_version VARCHAR(32);
ALTER TABLE users ADD COLUMN IF NOT EXISTS terms_accepted_at TIMESTAMPTZ;
