-- 000024_quickbooks_sync (T192), reversed.
DELETE FROM role_permissions
WHERE permission_id IN (SELECT id FROM permissions WHERE name = 'integrations.manage');
DELETE FROM permissions WHERE name = 'integrations.manage';
DROP TABLE IF EXISTS quickbooks_sync_map;
DROP TABLE IF EXISTS quickbooks_connections;
