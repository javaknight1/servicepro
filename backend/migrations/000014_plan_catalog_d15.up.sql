-- 000014_plan_catalog_d15 (T210, D15)
-- ServicePro's own plans: Solo $39/mo (1 user), Team $99/mo and Pro $199/mo
-- (unlimited users); annual = 10x monthly. Free and Basic are disabled for new
-- sign-ups; their existing subscriptions keep their locked-in price_cents.
-- The old Pro row becomes the new Pro (existing Pro subscriptions also keep
-- their locked-in price). New tiers carry no count limits, only team_members.
--
-- Trial: a nullable tenant_subscriptions.trial_ends_at. A no-card trial is an
-- ACTIVE subscription row (tier team, price 0, no Stripe subscription) with
-- trial_ends_at set, so every existing "status = 'active'" query, the
-- one-active-per-tenant unique index and get_tenant_subscription() keep
-- working unchanged. A 'trialing' membership_status value would have had to
-- be added to all of them (and ALTER TYPE ... ADD VALUE can't be used in the
-- transaction that adds it).

ALTER TABLE tenant_subscriptions ADD COLUMN IF NOT EXISTS trial_ends_at TIMESTAMPTZ;

COMMENT ON COLUMN tenant_subscriptions.trial_ends_at IS
    'End of the no-card trial (D15). NULL = not a trial. Set only on rows without a Stripe subscription.';

-- Legacy tiers: hidden from selection, still readable for grandfathered rows.
UPDATE membership_tiers SET enabled = FALSE, sort_order = 100 WHERE name = 'free';
UPDATE membership_tiers SET enabled = FALSE, sort_order = 101 WHERE name = 'basic';

INSERT INTO membership_tiers (id, name, display_name, description, monthly_price_cents, annual_price_cents, features, enabled, sort_order) VALUES
    (
        '00000000-0000-0000-0001-000000000004',
        'solo',
        'Solo',
        'For an owner-operator running the business alone',
        3900,
        39000,
        '{
            "team_members": 1,
            "advanced_reporting": false,
            "api_access": false,
            "custom_branding": true,
            "multi_entity": false,
            "highlights": [
                "Customers, sites and jobs",
                "Scheduling and the tech view",
                "Quotes, invoices and online payments"
            ]
        }'::jsonb,
        TRUE,
        1
    ),
    (
        '00000000-0000-0000-0001-000000000005',
        'team',
        'Team',
        'For a crew: unlimited users at one flat price',
        9900,
        99000,
        '{
            "team_members": -1,
            "advanced_reporting": false,
            "api_access": false,
            "custom_branding": true,
            "multi_entity": false,
            "highlights": [
                "Everything in Solo",
                "Dispatch for your whole crew",
                "Roles: owner, office, technician"
            ]
        }'::jsonb,
        TRUE,
        2
    )
ON CONFLICT (name) DO UPDATE SET
    display_name = EXCLUDED.display_name,
    description = EXCLUDED.description,
    monthly_price_cents = EXCLUDED.monthly_price_cents,
    annual_price_cents = EXCLUDED.annual_price_cents,
    features = EXCLUDED.features,
    enabled = EXCLUDED.enabled,
    sort_order = EXCLUDED.sort_order;

UPDATE membership_tiers SET
    display_name = 'Pro',
    description = 'For growing companies that need more reporting',
    monthly_price_cents = 19900,
    annual_price_cents = 199000,
    features = '{
        "team_members": -1,
        "advanced_reporting": true,
        "api_access": false,
        "custom_branding": true,
        "multi_entity": false,
        "highlights": [
            "Everything in Team",
            "Advanced reporting and exports",
            "Multiple business entities (coming later)"
        ]
    }'::jsonb,
    enabled = TRUE,
    sort_order = 3
WHERE name = 'pro';

-- Count limits are dropped from the new tiers (the table stays): only the
-- user limit remains.
DELETE FROM membership_limitations
WHERE tier_id = '00000000-0000-0000-0001-000000000003' AND resource_type <> 'team_members';

INSERT INTO membership_limitations (tier_id, resource_type, limit_value) VALUES
    ('00000000-0000-0000-0001-000000000004', 'team_members', 1),
    ('00000000-0000-0000-0001-000000000005', 'team_members', -1),
    ('00000000-0000-0000-0001-000000000003', 'team_members', -1)
ON CONFLICT (tier_id, resource_type) DO UPDATE SET limit_value = EXCLUDED.limit_value;
