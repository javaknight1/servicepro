-- Restore the webhooks flag as 000001 seeded it (true only on Pro).
UPDATE membership_tiers
SET features = features || jsonb_build_object('webhooks', name = 'pro')
WHERE NOT features ? 'webhooks';
