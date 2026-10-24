-- 000002_users_auth_user_id (T158)
-- Links a ServicePro user to its Neon Auth (managed Better Auth) identity:
-- the `sub` claim of the Neon Auth JWT, i.e. neon_auth's user id. NULL until
-- the user signs in through Neon Auth. Unique among live users.

ALTER TABLE users ADD COLUMN IF NOT EXISTS auth_user_id TEXT;

CREATE UNIQUE INDEX IF NOT EXISTS idx_users_auth_user_id
    ON users (auth_user_id)
    WHERE auth_user_id IS NOT NULL AND deleted_at IS NULL;
