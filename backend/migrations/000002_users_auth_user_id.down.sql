DROP INDEX IF EXISTS idx_users_auth_user_id;
ALTER TABLE users DROP COLUMN IF EXISTS auth_user_id;
