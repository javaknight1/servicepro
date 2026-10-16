-- ============================================================================
-- 000012_live_uniqueness (T226 / audit 4.2, 7.M12, 7.M13)
--
-- * Tenant slugs are unique among live tenants only. The global UNIQUE (slug)
--   kept the slug of a soft-deleted tenant (e.g. a failed signup) taken
--   forever.
-- * Only one *pending* invitation per (tenant, email). The old
--   UNIQUE (tenant_id, email) covered every status, so re-inviting someone
--   after a cancelled, expired or accepted invitation failed with a
--   duplicate-key error.
--
-- Customer email, the one-default payment term and the one-default invoice
-- template were already made per-tenant in 000003.
-- ============================================================================

ALTER TABLE tenants DROP CONSTRAINT IF EXISTS tenants_slug_key;
CREATE UNIQUE INDEX IF NOT EXISTS uq_tenants_slug_live
    ON tenants(slug) WHERE deleted_at IS NULL;

ALTER TABLE organization_invitations DROP CONSTRAINT IF EXISTS unique_pending_invitation;
CREATE UNIQUE INDEX IF NOT EXISTS uq_org_invitations_pending
    ON organization_invitations(tenant_id, LOWER(email)) WHERE status = 'pending';
