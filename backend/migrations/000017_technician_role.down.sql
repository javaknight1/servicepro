-- Reverts 000017_technician_role. Technician members and pending invitations
-- fall back to the read-only 'user' role (tenant_users and
-- organization_invitations reference roles with ON DELETE RESTRICT).

UPDATE tenant_users
SET role_id = (SELECT id FROM roles WHERE name = 'user')
WHERE role_id = (SELECT id FROM roles WHERE name = 'technician');

UPDATE organization_invitations
SET role_id = (SELECT id FROM roles WHERE name = 'user')
WHERE role_id = (SELECT id FROM roles WHERE name = 'technician');

DELETE FROM user_roles WHERE role_id = (SELECT id FROM roles WHERE name = 'technician');
DELETE FROM role_permissions WHERE role_id = (SELECT id FROM roles WHERE name = 'technician');
DELETE FROM roles WHERE name = 'technician';

DELETE FROM role_permissions WHERE permission_id = (SELECT id FROM permissions WHERE name = 'jobs.view_all');
DELETE FROM permissions WHERE name = 'jobs.view_all';
