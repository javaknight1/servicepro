-- 000014_plan_catalog_d15 down (T210)
-- Restores the pre-D15 catalog. Solo and Team are removed only when no
-- subscription references them; otherwise they are just disabled.
-- Subscriptions' price_cents were never changed, so nothing to restore there.

DELETE FROM membership_limitations
WHERE tier_id IN ('00000000-0000-0000-0001-000000000004', '00000000-0000-0000-0001-000000000005');

DELETE FROM membership_tiers mt
WHERE mt.name IN ('solo', 'team')
  AND NOT EXISTS (SELECT 1 FROM tenant_subscriptions ts WHERE ts.tier_id = mt.id);
UPDATE membership_tiers SET enabled = FALSE WHERE name IN ('solo', 'team');

UPDATE membership_tiers SET enabled = TRUE, sort_order = 1 WHERE name = 'free';
UPDATE membership_tiers SET enabled = TRUE, sort_order = 2 WHERE name = 'basic';

UPDATE membership_tiers SET
    display_name = 'Pro',
    description = 'Full-featured solution for professional teams',
    monthly_price_cents = 7900,
    annual_price_cents = 79000,
    features = '{
        "team_members": -1,
        "customers": -1,
        "jobs_per_month": -1,
        "quotes_per_month": -1,
        "storage_gb": 100,
        "advanced_reporting": true,
        "api_access": true,
        "custom_branding": true
    }'::jsonb,
    enabled = TRUE,
    sort_order = 3
WHERE name = 'pro';

INSERT INTO membership_limitations (tier_id, resource_type, limit_value) VALUES
    ('00000000-0000-0000-0001-000000000003', 'team_members', -1),
    ('00000000-0000-0000-0001-000000000003', 'customers', -1),
    ('00000000-0000-0000-0001-000000000003', 'jobs_per_month', -1),
    ('00000000-0000-0000-0001-000000000003', 'quotes_per_month', -1),
    ('00000000-0000-0000-0001-000000000003', 'storage_gb', 100)
ON CONFLICT (tier_id, resource_type) DO UPDATE SET limit_value = EXCLUDED.limit_value;

ALTER TABLE tenant_subscriptions DROP COLUMN IF EXISTS trial_ends_at;
