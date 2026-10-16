DROP INDEX IF EXISTS idx_tax_rates_tenant_name;

DELETE FROM role_permissions
WHERE permission_id IN (SELECT id FROM permissions WHERE name IN ('tax_rates.read', 'tax_rates.manage'));

DELETE FROM permissions WHERE name IN ('tax_rates.read', 'tax_rates.manage');
