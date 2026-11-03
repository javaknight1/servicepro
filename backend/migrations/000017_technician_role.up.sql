-- T035: the built-in Technician role and the jobs.view_all permission.
--
-- Roles are global rows; a tenant membership (tenant_users.role_id) points at
-- one, so adding the role here makes it available to every tenant.
--
-- jobs.view_all: see and work on every job in the tenant. The API applies an
-- assignment row filter to job queries for callers without it, so existing
-- roles get it to keep seeing every job; the technician role does not.

INSERT INTO permissions (id, name, description, resource, action) VALUES
    ('31000000-0000-0000-0000-000000000007', 'jobs.view_all', 'View and manage every job, not only assigned ones', 'jobs', 'view')
ON CONFLICT (name) DO NOTHING;

INSERT INTO roles (id, name, description, hierarchy_level) VALUES
    ('00000000-0000-0000-0000-000000000006', 'technician', 'Field technician: sees and updates only their assigned jobs, never prices or payments', 30)
ON CONFLICT (name) DO NOTHING;

-- Every existing role keeps seeing every job.
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM roles r CROSS JOIN permissions p
WHERE r.name IN ('super_admin', 'admin', 'manager', 'user', 'guest') AND p.name = 'jobs.view_all'
ON CONFLICT (role_id, permission_id) DO NOTHING;

-- Technician: their assigned jobs only (status, notes, materials). No
-- customers, quotes, invoices, payments, reports, settings or team.
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM roles r CROSS JOIN permissions p
WHERE r.name = 'technician' AND p.name IN ('jobs.read', 'jobs.list', 'jobs.update')
ON CONFLICT (role_id, permission_id) DO NOTHING;
