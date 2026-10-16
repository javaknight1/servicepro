-- Restoring the old global constraints fails if live data now relies on the
-- relaxed rules (a reused slug, or several non-pending invitations for one
-- email); resolve those rows by hand first.
DROP INDEX IF EXISTS uq_org_invitations_pending;
ALTER TABLE organization_invitations DROP CONSTRAINT IF EXISTS unique_pending_invitation;
ALTER TABLE organization_invitations ADD CONSTRAINT unique_pending_invitation UNIQUE (tenant_id, email);

DROP INDEX IF EXISTS uq_tenants_slug_live;
ALTER TABLE tenants DROP CONSTRAINT IF EXISTS tenants_slug_key;
ALTER TABLE tenants ADD CONSTRAINT tenants_slug_key UNIQUE (slug);
