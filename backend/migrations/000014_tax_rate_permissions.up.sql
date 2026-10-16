-- T237: tax rates API permissions.
--   tax_rates.read   list/get the tenant's rates and the global defaults (every role)
--   tax_rates.manage create/update/delete the tenant's own rates (super_admin, admin)
INSERT INTO permissions (id, name, description, resource, action) VALUES
    ('41000000-0000-0000-0000-000000000001', 'tax_rates.read', 'View tax rates', 'tax_rates', 'read'),
    ('41000000-0000-0000-0000-000000000002', 'tax_rates.manage', 'Create, update and delete tax rates', 'tax_rates', 'manage')
ON CONFLICT (name) DO NOTHING;

-- super_admin and admin: read + manage
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM roles r CROSS JOIN permissions p
WHERE r.name IN ('super_admin', 'admin') AND p.name IN ('tax_rates.read', 'tax_rates.manage')
ON CONFLICT (role_id, permission_id) DO NOTHING;

-- manager, user, guest: read (they pick rates on invoices and quotes)
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM roles r CROSS JOIN permissions p
WHERE r.name IN ('manager', 'user', 'guest') AND p.name = 'tax_rates.read'
ON CONFLICT (role_id, permission_id) DO NOTHING;

-- Tenant rates are listed by tenant and name.
CREATE INDEX IF NOT EXISTS idx_tax_rates_tenant_name ON tax_rates (tenant_id, name);
