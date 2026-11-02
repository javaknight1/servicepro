-- 000006_drop_unused_tables (T263)
-- Drop tables no code reads or writes (checked 2026-11-02: no Go reference,
-- no view, function or trigger outside the table itself depends on them).
-- customer_activities and customer_metrics_cache lost their last reader when
-- T266 removed the customers report. The down migration recreates them empty.
--
-- subscriptions.plan_id references subscription_plans; subscriptions is unused
-- too but out of scope here, so only its foreign key is dropped.

ALTER TABLE IF EXISTS subscriptions DROP CONSTRAINT IF EXISTS subscriptions_plan_id_fkey;

DROP TABLE IF EXISTS plan_features;
DROP TABLE IF EXISTS subscription_plans;
DROP TABLE IF EXISTS analytics.aggregated_metrics;
DROP TABLE IF EXISTS revenue_alerts;
DROP TABLE IF EXISTS schedule_exceptions;
DROP TABLE IF EXISTS holidays;
DROP TABLE IF EXISTS customer_segments;
DROP TABLE IF EXISTS customer_metrics_cache;
DROP TABLE IF EXISTS customer_activities;
