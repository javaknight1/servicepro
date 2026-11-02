-- Recreate the tables 000006 dropped, empty (definitions from
-- 000001_initial_schema.up.sql; the seed rows are not restored).

CREATE TABLE IF NOT EXISTS customer_activities (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    customer_id UUID NOT NULL REFERENCES customers(id) ON DELETE CASCADE,
    activity_type VARCHAR(50) NOT NULL,
    activity_date TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    reference_id UUID,
    reference_type VARCHAR(50),
    amount DECIMAL(12, 2),
    metadata JSONB,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_customer_activities_customer ON customer_activities(customer_id);
CREATE INDEX IF NOT EXISTS idx_customer_activities_date ON customer_activities(activity_date DESC);
CREATE INDEX IF NOT EXISTS idx_customer_activities_type ON customer_activities(activity_type);

CREATE TABLE IF NOT EXISTS customer_metrics_cache (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    customer_id UUID NOT NULL REFERENCES customers(id) ON DELETE CASCADE,
    total_revenue DECIMAL(12, 2) NOT NULL DEFAULT 0,
    total_jobs_revenue DECIMAL(12, 2) NOT NULL DEFAULT 0,
    average_job_value DECIMAL(12, 2) NOT NULL DEFAULT 0,
    total_jobs INT NOT NULL DEFAULT 0,
    completed_jobs INT NOT NULL DEFAULT 0,
    cancelled_jobs INT NOT NULL DEFAULT 0,
    pending_jobs INT NOT NULL DEFAULT 0,
    total_quotes INT NOT NULL DEFAULT 0,
    accepted_quotes INT NOT NULL DEFAULT 0,
    rejected_quotes INT NOT NULL DEFAULT 0,
    pending_quotes INT NOT NULL DEFAULT 0,
    quote_acceptance_rate DECIMAL(5, 2) NOT NULL DEFAULT 0,
    first_activity_date TIMESTAMP WITH TIME ZONE,
    last_activity_date TIMESTAMP WITH TIME ZONE,
    days_since_last_activity INT,
    total_interactions INT NOT NULL DEFAULT 0,
    customer_lifetime_days INT NOT NULL DEFAULT 0,
    average_days_between_jobs DECIMAL(10, 2),
    segment VARCHAR(50),
    segment_score DECIMAL(5, 2),
    last_updated TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    is_stale BOOLEAN NOT NULL DEFAULT false,
    CONSTRAINT unique_customer_metrics UNIQUE(customer_id)
);

CREATE INDEX IF NOT EXISTS idx_customer_metrics_segment ON customer_metrics_cache(segment);
CREATE INDEX IF NOT EXISTS idx_customer_metrics_revenue ON customer_metrics_cache(total_revenue DESC);

CREATE TABLE IF NOT EXISTS customer_segments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(50) NOT NULL UNIQUE,
    description TEXT,
    criteria JSONB NOT NULL,
    color VARCHAR(7),
    priority INT NOT NULL DEFAULT 0,
    is_active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS holidays (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(200) NOT NULL,
    date DATE NOT NULL,
    country VARCHAR(2) DEFAULT 'US',
    is_recurring BOOLEAN DEFAULT false,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP WITH TIME ZONE
);

CREATE INDEX IF NOT EXISTS idx_holidays_date ON holidays(date);

CREATE TABLE IF NOT EXISTS schedule_exceptions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    recurring_pattern_id UUID NOT NULL REFERENCES recurring_schedules(id) ON DELETE CASCADE,
    exception_date DATE NOT NULL,
    exception_type VARCHAR(50) NOT NULL CHECK (exception_type IN ('skip', 'reschedule', 'modify')),
    reason TEXT NOT NULL,
    rescheduled_date DATE,
    modifications JSONB,
    created_by UUID NOT NULL REFERENCES users(id),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP WITH TIME ZONE
);

CREATE TABLE IF NOT EXISTS revenue_alerts (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(200) NOT NULL,
    description TEXT,
    condition JSONB NOT NULL,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    last_triggered TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

DROP TRIGGER IF EXISTS trigger_revenue_alerts_updated_at ON revenue_alerts;
CREATE TRIGGER trigger_revenue_alerts_updated_at BEFORE UPDATE ON revenue_alerts
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TABLE IF NOT EXISTS analytics.aggregated_metrics (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    metric_name VARCHAR(100) NOT NULL,
    dimension VARCHAR(100) NOT NULL,
    dim_value VARCHAR(255) NOT NULL,
    tenant_id UUID,
    value DOUBLE PRECISION NOT NULL DEFAULT 0,
    count BIGINT NOT NULL DEFAULT 0,
    period_start TIMESTAMPTZ NOT NULL,
    period_end TIMESTAMPTZ NOT NULL,
    granularity VARCHAR(20) NOT NULL,
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_aggregated_metrics_name ON analytics.aggregated_metrics(metric_name);
CREATE INDEX IF NOT EXISTS idx_aggregated_metrics_period ON analytics.aggregated_metrics(period_start, period_end);

CREATE TABLE IF NOT EXISTS subscription_plans (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(100) NOT NULL,
    description TEXT,
    tier VARCHAR(50) NOT NULL,
    monthly_price BIGINT NOT NULL DEFAULT 0,
    annual_price BIGINT NOT NULL DEFAULT 0,
    quarterly_price BIGINT DEFAULT 0,
    currency VARCHAR(3) DEFAULT 'USD',
    stripe_price_id_monthly VARCHAR(255),
    stripe_price_id_annual VARCHAR(255),
    stripe_price_id_quarterly VARCHAR(255),
    stripe_product_id VARCHAR(255),
    trial_days INTEGER DEFAULT 0,
    is_active BOOLEAN DEFAULT TRUE,
    is_public BOOLEAN DEFAULT TRUE,
    sort_order INTEGER DEFAULT 0,
    metadata JSONB,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP WITH TIME ZONE
);

CREATE INDEX IF NOT EXISTS idx_subscription_plans_tier ON subscription_plans(tier);
CREATE INDEX IF NOT EXISTS idx_subscription_plans_active ON subscription_plans(is_active);

CREATE TABLE IF NOT EXISTS plan_features (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    plan_id UUID NOT NULL REFERENCES subscription_plans(id) ON DELETE CASCADE,
    feature_key VARCHAR(100) NOT NULL,
    feature_name VARCHAR(200) NOT NULL,
    feature_type VARCHAR(50) NOT NULL,
    value VARCHAR(500),
    numeric_value BIGINT,
    is_enabled BOOLEAN DEFAULT TRUE,
    description TEXT,
    sort_order INTEGER DEFAULT 0,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT unique_plan_feature UNIQUE (plan_id, feature_key)
);

-- subscriptions.plan_id → subscription_plans. NOT VALID: any subscriptions row
-- written while the table was gone points at no plan (the table is unused).
DO $$
BEGIN
    IF to_regclass('public.subscriptions') IS NOT NULL
       AND NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'subscriptions_plan_id_fkey') THEN
        ALTER TABLE subscriptions ADD CONSTRAINT subscriptions_plan_id_fkey
            FOREIGN KEY (plan_id) REFERENCES subscription_plans(id) NOT VALID;
    END IF;
END $$;
