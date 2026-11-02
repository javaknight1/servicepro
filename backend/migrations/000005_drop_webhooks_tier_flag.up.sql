-- 000005_drop_webhooks_tier_flag (T266)
-- Outgoing webhooks were never built; drop the plan flag that advertised
-- them from every membership tier's features. Data only, no schema change.
UPDATE membership_tiers SET features = features - 'webhooks' WHERE features ? 'webhooks';
