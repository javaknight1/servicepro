-- ============================================================================
-- DEVELOPMENT SEED DATA
-- ============================================================================
-- This file creates test data for local development.
-- Run with: make seed
--
-- Test User Credentials:
--   Email: dev@servicepro.local
--   Password: password123
--
-- DO NOT run this in production!
-- ============================================================================

-- Use a transaction so we can rollback on error
BEGIN;

-- ============================================================================
-- 1. TEST USER
-- ============================================================================
-- Password: password123 (bcrypt cost 12)
-- Generate new hash with: cd backend && go run -e 'auth.HashPassword("password123")'

INSERT INTO users (
    id,
    email,
    password_hash,
    role,
    first_name,
    last_name,
    email_verified,
    created_at,
    updated_at
) VALUES (
    '11111111-1111-1111-1111-111111111111',
    'dev@servicepro.local',
    '$2a$12$UOb3QkOkcLLHtFrsY0vTS.QzwTQxrgzSCJiUfHI/hrc1aHsH9Jit.',
    'user',
    'Dev',
    'User',
    TRUE,
    CURRENT_TIMESTAMP,
    CURRENT_TIMESTAMP
) ON CONFLICT (email) DO UPDATE SET
    password_hash = EXCLUDED.password_hash,
    first_name = EXCLUDED.first_name,
    last_name = EXCLUDED.last_name,
    email_verified = EXCLUDED.email_verified,
    updated_at = CURRENT_TIMESTAMP;

-- ============================================================================
-- 2. TEST TENANTS (ORGANIZATIONS)
-- ============================================================================

-- Tenant 1: Main test organization
INSERT INTO tenants (
    id,
    name,
    slug,
    owner_id,
    email,
    phone,
    is_active,
    created_at,
    updated_at
) VALUES (
    '22222222-2222-2222-2222-222222222222',
    'Acme Services',
    'acme-services',
    '11111111-1111-1111-1111-111111111111',
    'contact@acme-services.local',
    '555-123-4567',
    TRUE,
    CURRENT_TIMESTAMP,
    CURRENT_TIMESTAMP
) ON CONFLICT (slug) DO UPDATE SET
    name = EXCLUDED.name,
    owner_id = EXCLUDED.owner_id,
    email = EXCLUDED.email,
    phone = EXCLUDED.phone,
    updated_at = CURRENT_TIMESTAMP;

-- Tenant 2: Secondary organization for testing multi-tenant scenarios
INSERT INTO tenants (
    id,
    name,
    slug,
    owner_id,
    email,
    phone,
    is_active,
    created_at,
    updated_at
) VALUES (
    '33333333-3333-3333-3333-333333333333',
    'Beta Contractors',
    'beta-contractors',
    '11111111-1111-1111-1111-111111111111',
    'contact@beta-contractors.local',
    '555-987-6543',
    TRUE,
    CURRENT_TIMESTAMP,
    CURRENT_TIMESTAMP
) ON CONFLICT (slug) DO UPDATE SET
    name = EXCLUDED.name,
    owner_id = EXCLUDED.owner_id,
    email = EXCLUDED.email,
    phone = EXCLUDED.phone,
    updated_at = CURRENT_TIMESTAMP;

-- ============================================================================
-- 3. LINK USER TO TENANTS (with admin role)
-- ============================================================================

-- User as admin of Tenant 1
INSERT INTO tenant_users (
    id,
    tenant_id,
    user_id,
    role_id,
    is_active,
    accepted_at,
    created_at,
    updated_at
) VALUES (
    '44444444-4444-4444-4444-444444444441',
    '22222222-2222-2222-2222-222222222222',
    '11111111-1111-1111-1111-111111111111',
    '00000000-0000-0000-0000-000000000002', -- admin role
    TRUE,
    CURRENT_TIMESTAMP,
    CURRENT_TIMESTAMP,
    CURRENT_TIMESTAMP
) ON CONFLICT ON CONSTRAINT unique_tenant_user DO UPDATE SET
    role_id = EXCLUDED.role_id,
    is_active = EXCLUDED.is_active,
    updated_at = CURRENT_TIMESTAMP;

-- User as admin of Tenant 2
INSERT INTO tenant_users (
    id,
    tenant_id,
    user_id,
    role_id,
    is_active,
    accepted_at,
    created_at,
    updated_at
) VALUES (
    '44444444-4444-4444-4444-444444444442',
    '33333333-3333-3333-3333-333333333333',
    '11111111-1111-1111-1111-111111111111',
    '00000000-0000-0000-0000-000000000002', -- admin role
    TRUE,
    CURRENT_TIMESTAMP,
    CURRENT_TIMESTAMP,
    CURRENT_TIMESTAMP
) ON CONFLICT ON CONSTRAINT unique_tenant_user DO UPDATE SET
    role_id = EXCLUDED.role_id,
    is_active = EXCLUDED.is_active,
    updated_at = CURRENT_TIMESTAMP;

-- ============================================================================
-- 4. ASSIGN FREE TIER MEMBERSHIP TO TENANTS
-- ============================================================================

-- Free tier subscription for Tenant 1
INSERT INTO tenant_subscriptions (
    id,
    tenant_id,
    tier_id,
    status,
    billing_cycle,
    price_cents,
    started_at,
    current_period_start,
    current_period_end,
    created_at,
    updated_at
) VALUES (
    '55555555-5555-5555-5555-555555555551',
    '22222222-2222-2222-2222-222222222222',
    '00000000-0000-0000-0001-000000000001', -- free tier
    'active',
    'monthly',
    0,
    CURRENT_TIMESTAMP,
    CURRENT_TIMESTAMP,
    CURRENT_TIMESTAMP + INTERVAL '1 month',
    CURRENT_TIMESTAMP,
    CURRENT_TIMESTAMP
) ON CONFLICT DO NOTHING;

-- Free tier subscription for Tenant 2
INSERT INTO tenant_subscriptions (
    id,
    tenant_id,
    tier_id,
    status,
    billing_cycle,
    price_cents,
    started_at,
    current_period_start,
    current_period_end,
    created_at,
    updated_at
) VALUES (
    '55555555-5555-5555-5555-555555555552',
    '33333333-3333-3333-3333-333333333333',
    '00000000-0000-0000-0001-000000000001', -- free tier
    'active',
    'monthly',
    0,
    CURRENT_TIMESTAMP,
    CURRENT_TIMESTAMP,
    CURRENT_TIMESTAMP + INTERVAL '1 month',
    CURRENT_TIMESTAMP,
    CURRENT_TIMESTAMP
) ON CONFLICT DO NOTHING;

-- ============================================================================
-- 5. TEST CUSTOMERS (for Tenant 1)
-- ============================================================================

INSERT INTO customers (
    id,
    tenant_id,
    first_name,
    last_name,
    company_name,
    email,
    phone_primary,
    billing_address_street,
    billing_address_city,
    billing_address_state,
    billing_address_zip,
    customer_type,
    status,
    created_at,
    updated_at
) VALUES
    (
        '66666666-6666-6666-6666-666666666661',
        '22222222-2222-2222-2222-222222222222',
        'John',
        'Smith',
        'Smith Residence',
        'john.smith@example.com',
        '555-111-2222',
        '123 Main Street',
        'San Francisco',
        'CA',
        '94102',
        'residential',
        'active',
        CURRENT_TIMESTAMP,
        CURRENT_TIMESTAMP
    ),
    (
        '66666666-6666-6666-6666-666666666662',
        '22222222-2222-2222-2222-222222222222',
        'Jane',
        'Doe',
        'Doe Family',
        'jane.doe@example.com',
        '555-333-4444',
        '456 Oak Avenue',
        'Oakland',
        'CA',
        '94612',
        'residential',
        'active',
        CURRENT_TIMESTAMP,
        CURRENT_TIMESTAMP
    ),
    (
        '66666666-6666-6666-6666-666666666663',
        '22222222-2222-2222-2222-222222222222',
        'Robert',
        'Johnson',
        'Johnson Corp',
        'robert@johnsoncorp.example.com',
        '555-555-6666',
        '789 Business Blvd',
        'San Jose',
        'CA',
        '95112',
        'commercial',
        'active',
        CURRENT_TIMESTAMP,
        CURRENT_TIMESTAMP
    ),
    (
        '66666666-6666-6666-6666-666666666664',
        '22222222-2222-2222-2222-222222222222',
        'Emily',
        'Williams',
        NULL,
        'emily.williams@example.com',
        '555-777-8888',
        '321 Pine Street',
        'Berkeley',
        'CA',
        '94704',
        'residential',
        'prospect',
        CURRENT_TIMESTAMP,
        CURRENT_TIMESTAMP
    ),
    (
        '66666666-6666-6666-6666-666666666665',
        '22222222-2222-2222-2222-222222222222',
        'Michael',
        'Brown',
        'Brown & Associates',
        'michael@brownassoc.example.com',
        '555-999-0000',
        '654 Market Street',
        'San Francisco',
        'CA',
        '94103',
        'commercial',
        'active',
        CURRENT_TIMESTAMP,
        CURRENT_TIMESTAMP
    )
ON CONFLICT (id) DO UPDATE SET
    first_name = EXCLUDED.first_name,
    last_name = EXCLUDED.last_name,
    email = EXCLUDED.email,
    updated_at = CURRENT_TIMESTAMP;

-- ============================================================================
-- 6. TEST JOBS (for Tenant 1)
-- ============================================================================

INSERT INTO jobs (
    id,
    tenant_id,
    job_number,
    customer_id,
    title,
    description,
    job_type,
    status,
    priority,
    scheduled_start_at,
    estimated_duration,
    estimated_cost,
    service_street,
    service_city,
    service_state,
    service_zip,
    created_by,
    created_at,
    updated_at
) VALUES
    (
        '77777777-7777-7777-7777-777777777771',
        '22222222-2222-2222-2222-222222222222',
        'JOB-2024-001',
        '66666666-6666-6666-6666-666666666661',
        'Annual HVAC Maintenance',
        'Routine annual maintenance for HVAC system including filter replacement and system check.',
        'maintenance',
        'scheduled',
        'normal',
        CURRENT_TIMESTAMP + INTERVAL '2 days',
        120,
        250.00,
        '123 Main Street',
        'San Francisco',
        'CA',
        '94102',
        '11111111-1111-1111-1111-111111111111',
        CURRENT_TIMESTAMP,
        CURRENT_TIMESTAMP
    ),
    (
        '77777777-7777-7777-7777-777777777772',
        '22222222-2222-2222-2222-222222222222',
        'JOB-2024-002',
        '66666666-6666-6666-6666-666666666662',
        'Plumbing Repair - Kitchen Sink',
        'Fix leaking kitchen sink faucet and check for pipe damage.',
        'repair',
        'in_progress',
        'high',
        CURRENT_TIMESTAMP - INTERVAL '1 day',
        90,
        175.00,
        '456 Oak Avenue',
        'Oakland',
        'CA',
        '94612',
        '11111111-1111-1111-1111-111111111111',
        CURRENT_TIMESTAMP - INTERVAL '3 days',
        CURRENT_TIMESTAMP
    ),
    (
        '77777777-7777-7777-7777-777777777773',
        '22222222-2222-2222-2222-222222222222',
        'JOB-2024-003',
        '66666666-6666-6666-6666-666666666663',
        'Office Electrical Upgrade',
        'Install additional outlets and upgrade electrical panel for new office equipment.',
        'installation',
        'new',
        'normal',
        NULL,
        480,
        1500.00,
        '789 Business Blvd',
        'San Jose',
        'CA',
        '95112',
        '11111111-1111-1111-1111-111111111111',
        CURRENT_TIMESTAMP,
        CURRENT_TIMESTAMP
    ),
    (
        '77777777-7777-7777-7777-777777777774',
        '22222222-2222-2222-2222-222222222222',
        'JOB-2024-004',
        '66666666-6666-6666-6666-666666666661',
        'Water Heater Replacement',
        'Replace old water heater with new energy-efficient model.',
        'installation',
        'completed',
        'normal',
        CURRENT_TIMESTAMP - INTERVAL '7 days',
        240,
        850.00,
        '123 Main Street',
        'San Francisco',
        'CA',
        '94102',
        '11111111-1111-1111-1111-111111111111',
        CURRENT_TIMESTAMP - INTERVAL '10 days',
        CURRENT_TIMESTAMP - INTERVAL '7 days'
    ),
    (
        '77777777-7777-7777-7777-777777777775',
        '22222222-2222-2222-2222-222222222222',
        'JOB-2024-005',
        '66666666-6666-6666-6666-666666666665',
        'Emergency AC Repair',
        'AC unit not cooling - emergency service call.',
        'repair',
        'scheduled',
        'urgent',
        CURRENT_TIMESTAMP + INTERVAL '4 hours',
        60,
        350.00,
        '654 Market Street',
        'San Francisco',
        'CA',
        '94103',
        '11111111-1111-1111-1111-111111111111',
        CURRENT_TIMESTAMP,
        CURRENT_TIMESTAMP
    )
ON CONFLICT (job_number) DO UPDATE SET
    title = EXCLUDED.title,
    status = EXCLUDED.status,
    updated_at = CURRENT_TIMESTAMP;

-- ============================================================================
-- 7. TEST QUOTES (for Tenant 1)
-- ============================================================================

INSERT INTO quotes (
    id,
    tenant_id,
    customer_id,
    quote_number,
    status,
    valid_until,
    subtotal,
    tax_rate,
    tax_amount,
    total,
    notes,
    created_by,
    created_at,
    updated_at
) VALUES
    (
        '88888888-8888-8888-8888-888888888881',
        '22222222-2222-2222-2222-222222222222',
        '66666666-6666-6666-6666-666666666661',
        'QT-2024-001',
        'sent',
        CURRENT_TIMESTAMP + INTERVAL '30 days',
        1200.00,
        0.0875,
        105.00,
        1305.00,
        'Kitchen renovation estimate - includes materials and labor.',
        '11111111-1111-1111-1111-111111111111',
        CURRENT_TIMESTAMP - INTERVAL '5 days',
        CURRENT_TIMESTAMP - INTERVAL '5 days'
    ),
    (
        '88888888-8888-8888-8888-888888888882',
        '22222222-2222-2222-2222-222222222222',
        '66666666-6666-6666-6666-666666666663',
        'QT-2024-002',
        'draft',
        CURRENT_TIMESTAMP + INTERVAL '14 days',
        5500.00,
        0.0875,
        481.25,
        5981.25,
        'Full office HVAC system upgrade proposal.',
        '11111111-1111-1111-1111-111111111111',
        CURRENT_TIMESTAMP - INTERVAL '2 days',
        CURRENT_TIMESTAMP
    ),
    (
        '88888888-8888-8888-8888-888888888883',
        '22222222-2222-2222-2222-222222222222',
        '66666666-6666-6666-6666-666666666662',
        'QT-2024-003',
        'accepted',
        CURRENT_TIMESTAMP + INTERVAL '30 days',
        800.00,
        0.0875,
        70.00,
        870.00,
        'Bathroom plumbing repair and upgrade.',
        '11111111-1111-1111-1111-111111111111',
        CURRENT_TIMESTAMP - INTERVAL '10 days',
        CURRENT_TIMESTAMP - INTERVAL '8 days'
    ),
    (
        '88888888-8888-8888-8888-888888888884',
        '22222222-2222-2222-2222-222222222222',
        '66666666-6666-6666-6666-666666666664',
        'QT-2024-004',
        'sent',
        CURRENT_TIMESTAMP + INTERVAL '21 days',
        450.00,
        0.0875,
        39.38,
        489.38,
        'Initial consultation and basic electrical inspection.',
        '11111111-1111-1111-1111-111111111111',
        CURRENT_TIMESTAMP - INTERVAL '1 day',
        CURRENT_TIMESTAMP - INTERVAL '1 day'
    )
ON CONFLICT (quote_number) DO UPDATE SET
    status = EXCLUDED.status,
    subtotal = EXCLUDED.subtotal,
    total = EXCLUDED.total,
    updated_at = CURRENT_TIMESTAMP;

-- ============================================================================
-- 8. QUOTE ITEMS (for the quotes above)
-- ============================================================================

INSERT INTO quote_items (
    id,
    quote_id,
    description,
    quantity,
    unit_price,
    total,
    sort_order
) VALUES
    -- Items for Quote 1 (Kitchen renovation)
    ('99999999-9999-9999-9999-999999999901', '88888888-8888-8888-8888-888888888881', 'Labor - Plumbing work', 8, 75.00, 600.00, 1),
    ('99999999-9999-9999-9999-999999999902', '88888888-8888-8888-8888-888888888881', 'Kitchen faucet and fixtures', 1, 350.00, 350.00, 2),
    ('99999999-9999-9999-9999-999999999903', '88888888-8888-8888-8888-888888888881', 'Miscellaneous supplies', 1, 250.00, 250.00, 3),

    -- Items for Quote 2 (Office HVAC)
    ('99999999-9999-9999-9999-999999999904', '88888888-8888-8888-8888-888888888882', 'Commercial HVAC unit', 1, 3500.00, 3500.00, 1),
    ('99999999-9999-9999-9999-999999999905', '88888888-8888-8888-8888-888888888882', 'Installation labor', 16, 100.00, 1600.00, 2),
    ('99999999-9999-9999-9999-999999999906', '88888888-8888-8888-8888-888888888882', 'Ductwork modifications', 1, 400.00, 400.00, 3),

    -- Items for Quote 3 (Bathroom plumbing)
    ('99999999-9999-9999-9999-999999999907', '88888888-8888-8888-8888-888888888883', 'Pipe repair and replacement', 1, 400.00, 400.00, 1),
    ('99999999-9999-9999-9999-999999999908', '88888888-8888-8888-8888-888888888883', 'New shower valve', 1, 200.00, 200.00, 2),
    ('99999999-9999-9999-9999-999999999909', '88888888-8888-8888-8888-888888888883', 'Labor', 4, 50.00, 200.00, 3),

    -- Items for Quote 4 (Electrical inspection)
    ('99999999-9999-9999-9999-999999999910', '88888888-8888-8888-8888-888888888884', 'Electrical inspection fee', 1, 150.00, 150.00, 1),
    ('99999999-9999-9999-9999-999999999911', '88888888-8888-8888-8888-888888888884', 'Consultation (2 hours)', 2, 100.00, 200.00, 2),
    ('99999999-9999-9999-9999-999999999912', '88888888-8888-8888-8888-888888888884', 'Report preparation', 1, 100.00, 100.00, 3)
ON CONFLICT (id) DO UPDATE SET
    description = EXCLUDED.description,
    quantity = EXCLUDED.quantity,
    unit_price = EXCLUDED.unit_price,
    total = EXCLUDED.total;

COMMIT;

-- ============================================================================
-- SUMMARY
-- ============================================================================
--
-- Created:
--   - 1 test user (dev@servicepro.local / password123)
--   - 2 tenants (Acme Services, Beta Contractors)
--   - User linked to both tenants as admin
--   - 5 customers in Acme Services
--   - 5 jobs in Acme Services (various statuses)
--   - 4 quotes in Acme Services (various statuses)
--   - Quote line items for each quote
--
-- ============================================================================
