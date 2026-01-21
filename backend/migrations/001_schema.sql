-- ============================================================================
-- ServicePro Database Schema
-- Consolidated schema for fresh database setup
-- ============================================================================

-- ============================================================================
-- 1. EXTENSIONS
-- ============================================================================

CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pg_trgm";  -- For fuzzy text search

-- ============================================================================
-- 2. ENUM TYPES
-- ============================================================================

-- Customer enums
CREATE TYPE customer_type AS ENUM ('residential', 'commercial');
CREATE TYPE customer_status AS ENUM ('active', 'inactive', 'prospect');

-- Job enums
CREATE TYPE job_status AS ENUM ('scheduled', 'in_progress', 'on_hold', 'completed', 'cancelled');
CREATE TYPE job_priority AS ENUM ('low', 'normal', 'high', 'urgent');
CREATE TYPE job_type AS ENUM ('installation', 'maintenance', 'repair', 'inspection', 'emergency');

-- Quote enums
CREATE TYPE quote_status AS ENUM ('draft', 'sent', 'viewed', 'accepted', 'declined', 'expired');

-- Invoice enums
CREATE TYPE invoice_status AS ENUM ('draft', 'sent', 'viewed', 'paid', 'partially_paid', 'overdue', 'cancelled', 'refunded');
CREATE TYPE payment_term_type AS ENUM ('due_on_receipt', 'net_7', 'net_10', 'net_15', 'net_30', 'net_60', 'net_90', 'custom');
CREATE TYPE tax_type AS ENUM ('sales_tax', 'vat', 'gst', 'hst', 'exempt');

-- Template enums
CREATE TYPE template_status AS ENUM ('draft', 'active', 'archived', 'deprecated');

-- Notification enums
CREATE TYPE notification_type AS ENUM (
    'payment_due_soon', 'payment_overdue', 'payment_in_grace_period',
    'early_payment_discount_available', 'late_fee_applied',
    'payment_received', 'partial_payment_received'
);
CREATE TYPE notification_status AS ENUM ('pending', 'sent', 'failed', 'dismissed');
CREATE TYPE notification_channel AS ENUM ('email', 'sms', 'push', 'in_app', 'webhook');

-- ============================================================================
-- 3. HELPER FUNCTIONS
-- ============================================================================

-- Generic updated_at trigger function
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Validate email format
CREATE OR REPLACE FUNCTION validate_email(email_address TEXT)
RETURNS BOOLEAN AS $$
BEGIN
    RETURN email_address ~* '^[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Za-z]{2,}$';
END;
$$ LANGUAGE plpgsql IMMUTABLE;

-- Validate phone number (US format)
CREATE OR REPLACE FUNCTION validate_phone(phone TEXT)
RETURNS BOOLEAN AS $$
BEGIN
    RETURN phone ~ '^(\+?1[-.]?)?\(?\d{3}\)?[-.\s]?\d{3}[-.\s]?\d{4}$';
END;
$$ LANGUAGE plpgsql IMMUTABLE;

-- Validate US state code
CREATE OR REPLACE FUNCTION validate_state_code(state_code CHAR(2))
RETURNS BOOLEAN AS $$
BEGIN
    RETURN state_code ~ '^[A-Z]{2}$' AND state_code IN (
        'AL', 'AK', 'AZ', 'AR', 'CA', 'CO', 'CT', 'DE', 'FL', 'GA',
        'HI', 'ID', 'IL', 'IN', 'IA', 'KS', 'KY', 'LA', 'ME', 'MD',
        'MA', 'MI', 'MN', 'MS', 'MO', 'MT', 'NE', 'NV', 'NH', 'NJ',
        'NM', 'NY', 'NC', 'ND', 'OH', 'OK', 'OR', 'PA', 'RI', 'SC',
        'SD', 'TN', 'TX', 'UT', 'VT', 'VA', 'WA', 'WV', 'WI', 'WY',
        'DC', 'AS', 'GU', 'MP', 'PR', 'VI'
    );
END;
$$ LANGUAGE plpgsql IMMUTABLE;

-- Validate ZIP code (US format)
CREATE OR REPLACE FUNCTION validate_zip_code(zip TEXT)
RETURNS BOOLEAN AS $$
BEGIN
    RETURN zip ~ '^\d{5}(-\d{4})?$';
END;
$$ LANGUAGE plpgsql IMMUTABLE;

-- ============================================================================
-- 4. USERS & AUTHENTICATION
-- ============================================================================

CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    email VARCHAR(255) NOT NULL UNIQUE,
    password_hash VARCHAR(255) NOT NULL,
    role VARCHAR(50) NOT NULL DEFAULT 'user',
    email_verified BOOLEAN DEFAULT FALSE,
    verification_sent_at TIMESTAMP,
    failed_login_count INTEGER NOT NULL DEFAULT 0,
    last_failed_login_at TIMESTAMP,
    locked_until TIMESTAMP,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP WITH TIME ZONE
);

CREATE INDEX idx_users_email ON users(email);
CREATE INDEX idx_users_role ON users(role);
CREATE INDEX idx_users_email_verified ON users(email_verified);
CREATE INDEX idx_users_deleted_at ON users(deleted_at);
CREATE INDEX idx_users_locked_until ON users(locked_until) WHERE locked_until IS NOT NULL;

CREATE TRIGGER trigger_users_updated_at BEFORE UPDATE ON users
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- ============================================================================
-- 5. ROLES & PERMISSIONS (RBAC)
-- ============================================================================

CREATE TABLE roles (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(100) NOT NULL UNIQUE,
    description TEXT,
    parent_role_id UUID REFERENCES roles(id) ON DELETE SET NULL,
    hierarchy_level INTEGER DEFAULT 0,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP
);

CREATE TABLE permissions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(100) NOT NULL UNIQUE,
    description TEXT,
    resource VARCHAR(100) NOT NULL,
    action VARCHAR(50) NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP
);

CREATE TABLE role_permissions (
    role_id UUID NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
    permission_id UUID NOT NULL REFERENCES permissions(id) ON DELETE CASCADE,
    granted_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    granted_by UUID REFERENCES users(id) ON DELETE SET NULL,
    PRIMARY KEY (role_id, permission_id)
);

CREATE TABLE user_roles (
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role_id UUID NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
    assigned_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    assigned_by UUID REFERENCES users(id) ON DELETE SET NULL,
    expires_at TIMESTAMP,
    PRIMARY KEY (user_id, role_id)
);

CREATE INDEX idx_roles_name ON roles(name) WHERE deleted_at IS NULL;
CREATE INDEX idx_permissions_resource ON permissions(resource) WHERE deleted_at IS NULL;
CREATE INDEX idx_user_roles_user ON user_roles(user_id);
CREATE INDEX idx_user_roles_role ON user_roles(role_id);

-- ============================================================================
-- 6. CUSTOMERS
-- ============================================================================

CREATE TABLE customers (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    first_name VARCHAR(50) NOT NULL CHECK (LENGTH(TRIM(first_name)) > 0),
    last_name VARCHAR(50) NOT NULL CHECK (LENGTH(TRIM(last_name)) > 0),
    company_name VARCHAR(100),
    email VARCHAR(255) NOT NULL CHECK (validate_email(email)),
    phone_primary VARCHAR(20) NOT NULL CHECK (validate_phone(phone_primary)),
    phone_secondary VARCHAR(20) CHECK (phone_secondary IS NULL OR validate_phone(phone_secondary)),
    billing_address_street VARCHAR(100) NOT NULL,
    billing_address_city VARCHAR(50) NOT NULL,
    billing_address_state CHAR(2) NOT NULL CHECK (validate_state_code(billing_address_state)),
    billing_address_zip VARCHAR(10) NOT NULL CHECK (validate_zip_code(billing_address_zip)),
    service_address_street VARCHAR(100),
    service_address_city VARCHAR(50),
    service_address_state CHAR(2) CHECK (service_address_state IS NULL OR validate_state_code(service_address_state)),
    service_address_zip VARCHAR(10) CHECK (service_address_zip IS NULL OR validate_zip_code(service_address_zip)),
    customer_type customer_type NOT NULL DEFAULT 'residential',
    status customer_status NOT NULL DEFAULT 'prospect',
    notes TEXT,
    search_vector tsvector GENERATED ALWAYS AS (
        setweight(to_tsvector('english', COALESCE(first_name, '')), 'A') ||
        setweight(to_tsvector('english', COALESCE(last_name, '')), 'A') ||
        setweight(to_tsvector('english', COALESCE(company_name, '')), 'B') ||
        setweight(to_tsvector('english', COALESCE(email, '')), 'C')
    ) STORED,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP WITH TIME ZONE
);

CREATE UNIQUE INDEX idx_customers_email ON customers(email) WHERE deleted_at IS NULL;
CREATE INDEX idx_customers_name ON customers(last_name, first_name) WHERE deleted_at IS NULL;
CREATE INDEX idx_customers_phone ON customers(phone_primary) WHERE deleted_at IS NULL;
CREATE INDEX idx_customers_type ON customers(customer_type) WHERE deleted_at IS NULL;
CREATE INDEX idx_customers_status ON customers(status) WHERE deleted_at IS NULL;
CREATE INDEX idx_customers_search_vector ON customers USING GIN(search_vector);
CREATE INDEX idx_customers_deleted_at ON customers(deleted_at);

CREATE TRIGGER trigger_customers_updated_at BEFORE UPDATE ON customers
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- ============================================================================
-- 7. JOBS
-- ============================================================================

CREATE TABLE jobs (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    job_number VARCHAR(50) UNIQUE NOT NULL,
    customer_id UUID NOT NULL REFERENCES customers(id) ON DELETE RESTRICT,
    title VARCHAR(200) NOT NULL,
    description TEXT,
    job_type job_type NOT NULL,
    status job_status NOT NULL DEFAULT 'scheduled',
    priority job_priority NOT NULL DEFAULT 'normal',
    scheduled_start_at TIMESTAMP,
    scheduled_end_at TIMESTAMP,
    actual_start_at TIMESTAMP,
    actual_end_at TIMESTAMP,
    estimated_duration INT,
    actual_duration INT,
    service_street VARCHAR(100),
    service_city VARCHAR(50),
    service_state CHAR(2),
    service_zip VARCHAR(10),
    estimated_cost DECIMAL(10,2) DEFAULT 0,
    actual_cost DECIMAL(10,2) DEFAULT 0,
    tax_amount DECIMAL(10,2) DEFAULT 0,
    total_amount DECIMAL(10,2) DEFAULT 0,
    internal_notes TEXT,
    customer_notes TEXT,
    completion_notes TEXT,
    requires_follow_up BOOLEAN DEFAULT FALSE,
    follow_up_date TIMESTAMP,
    created_by UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    updated_by UUID REFERENCES users(id) ON DELETE RESTRICT,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP
);

CREATE INDEX idx_jobs_customer_id ON jobs(customer_id);
CREATE INDEX idx_jobs_status ON jobs(status);
CREATE INDEX idx_jobs_priority ON jobs(priority);
CREATE INDEX idx_jobs_scheduled_start ON jobs(scheduled_start_at);
CREATE INDEX idx_jobs_deleted_at ON jobs(deleted_at);

CREATE TRIGGER trigger_jobs_updated_at BEFORE UPDATE ON jobs
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TABLE job_assignments (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    job_id UUID NOT NULL REFERENCES jobs(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    role VARCHAR(50),
    assigned_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    unassigned_at TIMESTAMP,
    hours_worked DECIMAL(5,2) DEFAULT 0,
    notes TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP
);

CREATE INDEX idx_job_assignments_job ON job_assignments(job_id);
CREATE INDEX idx_job_assignments_user ON job_assignments(user_id);

CREATE TABLE job_materials (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    job_id UUID NOT NULL REFERENCES jobs(id) ON DELETE CASCADE,
    name VARCHAR(200) NOT NULL,
    description TEXT,
    sku VARCHAR(100),
    quantity DECIMAL(10,2) NOT NULL CHECK (quantity > 0),
    unit VARCHAR(20) NOT NULL,
    unit_cost DECIMAL(10,2) NOT NULL CHECK (unit_cost >= 0),
    total_cost DECIMAL(10,2) NOT NULL CHECK (total_cost >= 0),
    billable BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP
);

CREATE INDEX idx_job_materials_job ON job_materials(job_id);

CREATE TABLE job_notes (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    job_id UUID NOT NULL REFERENCES jobs(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    note TEXT NOT NULL,
    is_internal BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP
);

CREATE INDEX idx_job_notes_job ON job_notes(job_id);

-- ============================================================================
-- 8. SCHEDULING
-- ============================================================================

CREATE TABLE recurring_schedules (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    title VARCHAR(200) NOT NULL,
    description TEXT,
    recurrence_type VARCHAR(50) NOT NULL CHECK (recurrence_type IN ('daily', 'weekly', 'monthly', 'yearly', 'custom')),
    recurrence_rule TEXT,
    start_date DATE NOT NULL,
    end_date DATE,
    interval INTEGER DEFAULT 1 CHECK (interval >= 1),
    days_of_week INTEGER[],
    day_of_month INTEGER CHECK (day_of_month IS NULL OR (day_of_month >= 1 AND day_of_month <= 31)),
    month_of_year INTEGER CHECK (month_of_year IS NULL OR (month_of_year >= 1 AND month_of_year <= 12)),
    occurrences INTEGER,
    time_start TIME NOT NULL,
    time_end TIME NOT NULL,
    duration INTEGER,
    assigned_tech_ids UUID[],
    location VARCHAR(200),
    job_template_id UUID,
    is_active BOOLEAN DEFAULT TRUE,
    created_by UUID NOT NULL,
    updated_by UUID,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP WITH TIME ZONE
);

CREATE INDEX idx_recurring_schedules_active ON recurring_schedules(is_active) WHERE is_active = TRUE;

CREATE TABLE schedules (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    job_id UUID NOT NULL REFERENCES jobs(id) ON DELETE CASCADE,
    title VARCHAR(200) NOT NULL,
    description TEXT,
    start_time TIMESTAMP WITH TIME ZONE NOT NULL,
    end_time TIMESTAMP WITH TIME ZONE NOT NULL,
    all_day BOOLEAN DEFAULT FALSE,
    recurrence_type VARCHAR(50) DEFAULT 'none' CHECK (recurrence_type IN ('none', 'daily', 'weekly', 'monthly', 'yearly', 'custom')),
    recurring_schedule_id UUID REFERENCES recurring_schedules(id) ON DELETE SET NULL,
    assigned_tech_ids UUID[],
    location VARCHAR(200),
    notes TEXT,
    is_confirmed BOOLEAN DEFAULT FALSE,
    confirmed_at TIMESTAMP WITH TIME ZONE,
    confirmed_by UUID,
    is_cancelled BOOLEAN DEFAULT FALSE,
    cancelled_at TIMESTAMP WITH TIME ZONE,
    cancelled_by UUID,
    cancellation_reason TEXT,
    reminders_enabled BOOLEAN DEFAULT TRUE,
    reminder_time TIMESTAMP WITH TIME ZONE,
    color VARCHAR(7) CHECK (color IS NULL OR color ~ '^#[0-9A-Fa-f]{6}$'),
    created_by UUID NOT NULL,
    updated_by UUID,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP WITH TIME ZONE,
    CONSTRAINT schedules_end_after_start CHECK (end_time > start_time)
);

CREATE INDEX idx_schedules_job ON schedules(job_id);
CREATE INDEX idx_schedules_time_range ON schedules(start_time, end_time);
CREATE INDEX idx_schedules_assigned_techs ON schedules USING GIN(assigned_tech_ids);

CREATE TABLE schedule_conflicts (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    schedule_id_1 UUID NOT NULL REFERENCES schedules(id) ON DELETE CASCADE,
    schedule_id_2 UUID NOT NULL REFERENCES schedules(id) ON DELETE CASCADE,
    conflict_type VARCHAR(50) NOT NULL CHECK (conflict_type IN ('technician_overlap', 'location_overlap', 'equipment_overlap', 'time_overlap')),
    conflicting_resource UUID,
    severity VARCHAR(50) NOT NULL CHECK (severity IN ('low', 'medium', 'high', 'critical')),
    description TEXT,
    is_resolved BOOLEAN DEFAULT FALSE,
    resolved_at TIMESTAMP WITH TIME ZONE,
    resolved_by UUID,
    resolution_notes TEXT,
    detected_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT schedule_conflicts_different CHECK (schedule_id_1 != schedule_id_2)
);

CREATE TABLE schedule_exceptions (
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

CREATE TABLE holidays (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(200) NOT NULL,
    date DATE NOT NULL,
    country VARCHAR(2) DEFAULT 'US',
    is_recurring BOOLEAN DEFAULT false,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP WITH TIME ZONE
);

CREATE INDEX idx_holidays_date ON holidays(date);

-- ============================================================================
-- 9. QUOTES
-- ============================================================================

CREATE SEQUENCE IF NOT EXISTS quote_number_seq START 1;

CREATE TABLE quotes (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    customer_id UUID NOT NULL REFERENCES customers(id) ON DELETE CASCADE,
    quote_number VARCHAR(50) NOT NULL UNIQUE,
    status quote_status NOT NULL DEFAULT 'draft',
    valid_until TIMESTAMP WITH TIME ZONE NOT NULL,
    subtotal DECIMAL(12, 2) NOT NULL DEFAULT 0.00 CHECK (subtotal >= 0),
    tax_rate DECIMAL(5, 4) NOT NULL DEFAULT 0.0000 CHECK (tax_rate >= 0 AND tax_rate <= 1),
    tax_amount DECIMAL(12, 2) NOT NULL DEFAULT 0.00 CHECK (tax_amount >= 0),
    total DECIMAL(12, 2) NOT NULL DEFAULT 0.00 CHECK (total >= 0),
    notes TEXT,
    terms TEXT,
    created_by UUID NOT NULL REFERENCES users(id),
    updated_by UUID REFERENCES users(id),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP WITH TIME ZONE
);

CREATE INDEX idx_quotes_customer ON quotes(customer_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_quotes_status ON quotes(status) WHERE deleted_at IS NULL;
CREATE INDEX idx_quotes_created_at ON quotes(created_at DESC) WHERE deleted_at IS NULL;

CREATE TABLE quote_items (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    quote_id UUID NOT NULL REFERENCES quotes(id) ON DELETE CASCADE,
    description TEXT NOT NULL CHECK (TRIM(description) != ''),
    quantity DECIMAL(10, 2) NOT NULL DEFAULT 1.00 CHECK (quantity > 0),
    unit_price DECIMAL(12, 2) NOT NULL DEFAULT 0.00 CHECK (unit_price >= 0),
    total DECIMAL(12, 2) NOT NULL DEFAULT 0.00 CHECK (total >= 0),
    sort_order INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP WITH TIME ZONE
);

CREATE INDEX idx_quote_items_quote ON quote_items(quote_id) WHERE deleted_at IS NULL;

CREATE TRIGGER trigger_quotes_updated_at BEFORE UPDATE ON quotes
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- Quote item total calculation trigger
CREATE OR REPLACE FUNCTION calculate_quote_item_total()
RETURNS TRIGGER AS $$
BEGIN
    NEW.total = NEW.quantity * NEW.unit_price;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_calculate_quote_item_total
    BEFORE INSERT OR UPDATE ON quote_items
    FOR EACH ROW EXECUTE FUNCTION calculate_quote_item_total();

-- Quote number generator
CREATE OR REPLACE FUNCTION generate_quote_number()
RETURNS TEXT AS $$
DECLARE
    next_num INTEGER;
BEGIN
    next_num := nextval('quote_number_seq');
    RETURN 'QUO-' || TO_CHAR(CURRENT_DATE, 'YYYY') || '-' || LPAD(next_num::TEXT, 4, '0');
END;
$$ LANGUAGE plpgsql;

-- ============================================================================
-- 10. INVOICES & PAYMENTS
-- ============================================================================

CREATE TABLE tax_rates (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(100) NOT NULL,
    description TEXT,
    rate DECIMAL(10, 4) NOT NULL CHECK (rate >= 0 AND rate <= 1),
    tax_type tax_type NOT NULL DEFAULT 'sales_tax',
    region VARCHAR(100),
    is_compound BOOLEAN DEFAULT FALSE,
    is_active BOOLEAN DEFAULT TRUE,
    effective_date DATE,
    expiry_date DATE,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

CREATE TABLE payment_terms (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(100) NOT NULL,
    description TEXT,
    term_type payment_term_type NOT NULL DEFAULT 'net_30',
    days_until_due INTEGER CHECK (days_until_due >= 0),
    discount_percentage DECIMAL(5, 2) CHECK (discount_percentage >= 0 AND discount_percentage <= 100),
    discount_days INTEGER CHECK (discount_days >= 0),
    late_fee_percentage DECIMAL(5, 2) CHECK (late_fee_percentage >= 0 AND late_fee_percentage <= 100),
    late_fee_amount DECIMAL(10, 2) CHECK (late_fee_amount >= 0),
    grace_period_days INTEGER DEFAULT 0,
    is_default BOOLEAN DEFAULT FALSE,
    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

CREATE UNIQUE INDEX idx_payment_terms_one_default ON payment_terms(is_default) WHERE is_default = TRUE;

CREATE TABLE invoices (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    invoice_number VARCHAR(50) UNIQUE NOT NULL,
    customer_id UUID NOT NULL REFERENCES customers(id),
    job_id UUID REFERENCES jobs(id),
    quote_id UUID REFERENCES quotes(id),
    status invoice_status NOT NULL DEFAULT 'draft',
    issue_date DATE NOT NULL DEFAULT CURRENT_DATE,
    due_date DATE NOT NULL,
    sent_date TIMESTAMP,
    viewed_date TIMESTAMP,
    paid_date TIMESTAMP,
    subtotal DECIMAL(12, 2) NOT NULL DEFAULT 0.00 CHECK (subtotal >= 0),
    tax_amount DECIMAL(12, 2) NOT NULL DEFAULT 0.00 CHECK (tax_amount >= 0),
    discount_amount DECIMAL(12, 2) DEFAULT 0.00 CHECK (discount_amount >= 0),
    total_amount DECIMAL(12, 2) NOT NULL DEFAULT 0.00 CHECK (total_amount >= 0),
    amount_paid DECIMAL(12, 2) DEFAULT 0.00 CHECK (amount_paid >= 0),
    amount_due DECIMAL(12, 2) GENERATED ALWAYS AS (total_amount - amount_paid) STORED,
    payment_term_id UUID REFERENCES payment_terms(id),
    tax_rate_id UUID REFERENCES tax_rates(id),
    po_number VARCHAR(100),
    notes TEXT,
    terms_and_conditions TEXT,
    footer_text TEXT,
    created_by UUID NOT NULL REFERENCES users(id),
    updated_by UUID REFERENCES users(id),
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),
    deleted_at TIMESTAMP,
    CONSTRAINT valid_due_date CHECK (due_date >= issue_date)
);

CREATE INDEX idx_invoices_customer ON invoices(customer_id);
CREATE INDEX idx_invoices_status ON invoices(status);
CREATE INDEX idx_invoices_due_date ON invoices(due_date);
CREATE INDEX idx_invoices_deleted_at ON invoices(deleted_at) WHERE deleted_at IS NULL;

CREATE TRIGGER trigger_invoices_updated_at BEFORE UPDATE ON invoices
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- Invoice number generator
CREATE OR REPLACE FUNCTION generate_invoice_number()
RETURNS TRIGGER AS $$
DECLARE
    next_number INTEGER;
    year_prefix VARCHAR(4);
BEGIN
    IF NEW.invoice_number IS NULL OR NEW.invoice_number = '' THEN
        year_prefix := TO_CHAR(CURRENT_DATE, 'YYYY');
        SELECT COALESCE(MAX(CAST(SUBSTRING(invoice_number FROM '\d+$') AS INTEGER)), 0) + 1
        INTO next_number FROM invoices WHERE invoice_number LIKE 'INV-' || year_prefix || '-%';
        NEW.invoice_number := 'INV-' || year_prefix || '-' || LPAD(next_number::TEXT, 5, '0');
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_generate_invoice_number
    BEFORE INSERT ON invoices FOR EACH ROW EXECUTE FUNCTION generate_invoice_number();

CREATE TABLE invoice_lines (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    invoice_id UUID NOT NULL REFERENCES invoices(id) ON DELETE CASCADE,
    description TEXT NOT NULL,
    quantity DECIMAL(10, 2) NOT NULL DEFAULT 1.00 CHECK (quantity > 0),
    unit_price DECIMAL(12, 2) NOT NULL CHECK (unit_price >= 0),
    unit_of_measure VARCHAR(50) DEFAULT 'each',
    discount_percentage DECIMAL(5, 2) DEFAULT 0.00,
    discount_amount DECIMAL(12, 2) DEFAULT 0.00,
    taxable BOOLEAN DEFAULT TRUE,
    tax_rate DECIMAL(10, 4) DEFAULT 0.00,
    tax_amount DECIMAL(12, 2) DEFAULT 0.00,
    line_total DECIMAL(12, 2) GENERATED ALWAYS AS ((quantity * unit_price) - discount_amount) STORED,
    line_total_with_tax DECIMAL(12, 2) GENERATED ALWAYS AS (((quantity * unit_price) - discount_amount) + tax_amount) STORED,
    product_id UUID,
    service_id UUID,
    sort_order INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX idx_invoice_lines_invoice ON invoice_lines(invoice_id);

-- Payments table
CREATE TABLE payments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    invoice_id UUID REFERENCES invoices(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id),
    amount DECIMAL(12, 2) NOT NULL CHECK (amount > 0),
    payment_date DATE NOT NULL DEFAULT CURRENT_DATE,
    payment_method VARCHAR(50),
    reference_number VARCHAR(100),
    stripe_payment_id VARCHAR(255),
    stripe_charge_id VARCHAR(255),
    status VARCHAR(50) NOT NULL DEFAULT 'pending',
    notes TEXT,
    metadata JSONB DEFAULT '{}',
    created_by UUID REFERENCES users(id),
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX idx_payments_invoice ON payments(invoice_id);
CREATE INDEX idx_payments_user ON payments(user_id);
CREATE INDEX idx_payments_status ON payments(status);
CREATE INDEX idx_payments_stripe ON payments(stripe_payment_id);

CREATE TABLE invoice_audit_log (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    invoice_id UUID NOT NULL REFERENCES invoices(id) ON DELETE CASCADE,
    action VARCHAR(50) NOT NULL,
    field_name VARCHAR(100),
    old_value TEXT,
    new_value TEXT,
    from_status invoice_status,
    to_status invoice_status,
    changed_by UUID REFERENCES users(id),
    changed_by_type VARCHAR(50),
    ip_address INET,
    user_agent TEXT,
    created_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX idx_invoice_audit_invoice ON invoice_audit_log(invoice_id);

-- ============================================================================
-- 11. INVOICE TEMPLATES
-- ============================================================================

CREATE TABLE invoice_templates (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(200) NOT NULL,
    description TEXT,
    html_content TEXT NOT NULL,
    css_content TEXT,
    header_html TEXT,
    footer_html TEXT,
    status template_status NOT NULL DEFAULT 'draft',
    is_default BOOLEAN DEFAULT FALSE,
    version INTEGER DEFAULT 1,
    page_size VARCHAR(20) DEFAULT 'A4' CHECK (page_size IN ('A4', 'Letter', 'Legal', 'A5')),
    page_orientation VARCHAR(20) DEFAULT 'portrait' CHECK (page_orientation IN ('portrait', 'landscape')),
    margin_top DECIMAL(5,2) DEFAULT 10.0,
    margin_right DECIMAL(5,2) DEFAULT 10.0,
    margin_bottom DECIMAL(5,2) DEFAULT 10.0,
    margin_left DECIMAL(5,2) DEFAULT 10.0,
    logo_url TEXT,
    logo_position VARCHAR(50) DEFAULT 'top-left',
    logo_width DECIMAL(5,2),
    logo_height DECIMAL(5,2),
    watermark_text VARCHAR(200),
    watermark_opacity DECIMAL(3,2) DEFAULT 0.1 CHECK (watermark_opacity >= 0.0 AND watermark_opacity <= 1.0),
    watermark_rotation INTEGER DEFAULT 45 CHECK (watermark_rotation >= 0 AND watermark_rotation <= 360),
    watermark_enabled BOOLEAN DEFAULT FALSE,
    show_page_numbers BOOLEAN DEFAULT TRUE,
    page_number_format VARCHAR(100) DEFAULT 'Page {page} of {total}',
    page_number_position VARCHAR(50) DEFAULT 'bottom-center',
    custom_fonts JSONB,
    field_mappings JSONB,
    preview_data JSONB,
    parent_template_id UUID REFERENCES invoice_templates(id),
    version_notes TEXT,
    tags TEXT[],
    created_by UUID NOT NULL,
    updated_by UUID,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX idx_invoice_templates_status ON invoice_templates(status);
CREATE INDEX idx_invoice_templates_default ON invoice_templates(is_default) WHERE is_default = TRUE;

CREATE UNIQUE INDEX idx_invoice_templates_one_default
    ON invoice_templates(is_default) WHERE is_default = TRUE AND status = 'active';

-- ============================================================================
-- 12. IMPORT/EXPORT
-- ============================================================================

CREATE TABLE import_jobs (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    filename VARCHAR(255) NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'pending',
    total_rows INT NOT NULL DEFAULT 0,
    processed_rows INT NOT NULL DEFAULT 0,
    successful_rows INT NOT NULL DEFAULT 0,
    failed_rows INT NOT NULL DEFAULT 0,
    error_summary TEXT,
    started_at TIMESTAMP,
    completed_at TIMESTAMP,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_import_jobs_user ON import_jobs(user_id);
CREATE INDEX idx_import_jobs_status ON import_jobs(status);

CREATE TABLE import_errors (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    job_id UUID NOT NULL REFERENCES import_jobs(id) ON DELETE CASCADE,
    row_number INT NOT NULL,
    field VARCHAR(100),
    error TEXT NOT NULL,
    row_data TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_import_errors_job ON import_errors(job_id);

CREATE TABLE export_jobs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    export_type VARCHAR(50) NOT NULL CHECK (export_type IN ('customers', 'jobs', 'invoices', 'quotes', 'revenue_report', 'customer_report')),
    format VARCHAR(20) NOT NULL CHECK (format IN ('csv', 'json', 'xlsx', 'pdf')),
    status VARCHAR(20) NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'processing', 'completed', 'failed', 'cancelled')),
    total_records INTEGER NOT NULL DEFAULT 0,
    processed_records INTEGER NOT NULL DEFAULT 0,
    progress DECIMAL(5,2) NOT NULL DEFAULT 0,
    filename VARCHAR(255),
    file_path VARCHAR(500),
    file_size BIGINT DEFAULT 0,
    mime_type VARCHAR(100),
    download_url VARCHAR(500),
    query JSONB,
    encoding VARCHAR(20) DEFAULT 'utf-8',
    error_message TEXT,
    error_details TEXT,
    started_at TIMESTAMP WITH TIME ZONE,
    completed_at TIMESTAMP WITH TIME ZONE,
    expires_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_export_jobs_user ON export_jobs(user_id);
CREATE INDEX idx_export_jobs_status ON export_jobs(status);

-- ============================================================================
-- 13. CUSTOMER REPORTING
-- ============================================================================

CREATE TABLE customer_activities (
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

CREATE INDEX idx_customer_activities_customer ON customer_activities(customer_id);
CREATE INDEX idx_customer_activities_date ON customer_activities(activity_date DESC);
CREATE INDEX idx_customer_activities_type ON customer_activities(activity_type);

CREATE TABLE customer_segments (
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

CREATE TABLE customer_metrics_cache (
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

CREATE INDEX idx_customer_metrics_segment ON customer_metrics_cache(segment);
CREATE INDEX idx_customer_metrics_revenue ON customer_metrics_cache(total_revenue DESC);

-- ============================================================================
-- 14. SUBSCRIPTIONS
-- ============================================================================

CREATE TABLE subscription_plans (
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

CREATE INDEX idx_subscription_plans_tier ON subscription_plans(tier);
CREATE INDEX idx_subscription_plans_active ON subscription_plans(is_active);

CREATE TABLE plan_features (
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

CREATE TABLE subscriptions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    customer_id UUID NOT NULL,
    plan_id UUID NOT NULL REFERENCES subscription_plans(id),
    stripe_subscription_id VARCHAR(255) UNIQUE,
    stripe_customer_id VARCHAR(255),
    status VARCHAR(50) NOT NULL,
    billing_cycle VARCHAR(50) NOT NULL,
    start_date TIMESTAMP WITH TIME ZONE NOT NULL,
    current_period_start TIMESTAMP WITH TIME ZONE NOT NULL,
    current_period_end TIMESTAMP WITH TIME ZONE NOT NULL,
    trial_start TIMESTAMP WITH TIME ZONE,
    trial_end TIMESTAMP WITH TIME ZONE,
    canceled_at TIMESTAMP WITH TIME ZONE,
    ended_at TIMESTAMP WITH TIME ZONE,
    cancel_at_period_end BOOLEAN DEFAULT FALSE,
    cancellation_reason VARCHAR(50),
    cancellation_feedback TEXT,
    price_at_signup BIGINT,
    currency VARCHAR(3) DEFAULT 'USD',
    discount_id VARCHAR(255),
    discount_percent INTEGER,
    discount_ends_at TIMESTAMP WITH TIME ZONE,
    metadata JSONB,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP WITH TIME ZONE
);

CREATE INDEX idx_subscriptions_customer ON subscriptions(customer_id);
CREATE INDEX idx_subscriptions_status ON subscriptions(status);
CREATE INDEX idx_subscriptions_stripe ON subscriptions(stripe_subscription_id);

-- ============================================================================
-- 15. ANALYTICS SCHEMA
-- ============================================================================

CREATE SCHEMA IF NOT EXISTS analytics;

CREATE TABLE analytics.events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(100) NOT NULL,
    category VARCHAR(50) NOT NULL,
    user_id UUID,
    session_id VARCHAR(100),
    tenant_id UUID,
    properties JSONB DEFAULT '{}',
    timestamp TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    source VARCHAR(50) NOT NULL DEFAULT 'api',
    version VARCHAR(20),
    environment VARCHAR(20) NOT NULL DEFAULT 'production',
    ip_address VARCHAR(45),
    user_agent TEXT,
    referrer TEXT,
    device_type VARCHAR(20),
    country VARCHAR(2),
    region VARCHAR(100),
    city VARCHAR(100),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_events_name ON analytics.events(name);
CREATE INDEX idx_events_category ON analytics.events(category);
CREATE INDEX idx_events_user_id ON analytics.events(user_id) WHERE user_id IS NOT NULL;
CREATE INDEX idx_events_timestamp ON analytics.events(timestamp);
CREATE INDEX idx_events_properties ON analytics.events USING GIN(properties);

CREATE TABLE analytics.aggregated_metrics (
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

CREATE INDEX idx_aggregated_metrics_name ON analytics.aggregated_metrics(metric_name);
CREATE INDEX idx_aggregated_metrics_period ON analytics.aggregated_metrics(period_start, period_end);

-- ============================================================================
-- 16. TENANTS & MULTI-TENANCY
-- ============================================================================

-- Tenants table
CREATE TABLE tenants (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(200) NOT NULL,
    slug VARCHAR(100) UNIQUE NOT NULL,
    owner_id UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    email VARCHAR(255),
    phone VARCHAR(20),
    logo_url TEXT,
    settings JSONB DEFAULT '{}',
    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP WITH TIME ZONE
);

CREATE INDEX idx_tenants_slug ON tenants(slug) WHERE deleted_at IS NULL;
CREATE INDEX idx_tenants_owner ON tenants(owner_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_tenants_active ON tenants(is_active) WHERE deleted_at IS NULL AND is_active = TRUE;

CREATE TRIGGER trigger_tenants_updated_at BEFORE UPDATE ON tenants
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- Tenant users (membership) table
CREATE TABLE tenant_users (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role_id UUID NOT NULL REFERENCES roles(id) ON DELETE RESTRICT,
    invited_by UUID REFERENCES users(id) ON DELETE SET NULL,
    invited_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    accepted_at TIMESTAMP WITH TIME ZONE,
    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT unique_tenant_user UNIQUE (tenant_id, user_id)
);

CREATE INDEX idx_tenant_users_tenant ON tenant_users(tenant_id);
CREATE INDEX idx_tenant_users_user ON tenant_users(user_id);
CREATE INDEX idx_tenant_users_role ON tenant_users(role_id);
CREATE INDEX idx_tenant_users_active ON tenant_users(is_active) WHERE is_active = TRUE;

CREATE TRIGGER trigger_tenant_users_updated_at BEFORE UPDATE ON tenant_users
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- Add tenant_id to business tables (nullable for backwards compatibility)
ALTER TABLE customers ADD COLUMN tenant_id UUID REFERENCES tenants(id) ON DELETE CASCADE;
CREATE INDEX idx_customers_tenant ON customers(tenant_id) WHERE deleted_at IS NULL;

ALTER TABLE jobs ADD COLUMN tenant_id UUID REFERENCES tenants(id) ON DELETE CASCADE;
CREATE INDEX idx_jobs_tenant ON jobs(tenant_id) WHERE deleted_at IS NULL;

ALTER TABLE quotes ADD COLUMN tenant_id UUID REFERENCES tenants(id) ON DELETE CASCADE;
CREATE INDEX idx_quotes_tenant ON quotes(tenant_id) WHERE deleted_at IS NULL;

ALTER TABLE invoices ADD COLUMN tenant_id UUID REFERENCES tenants(id) ON DELETE CASCADE;
CREATE INDEX idx_invoices_tenant ON invoices(tenant_id) WHERE deleted_at IS NULL;

ALTER TABLE payments ADD COLUMN tenant_id UUID REFERENCES tenants(id) ON DELETE CASCADE;
CREATE INDEX idx_payments_tenant ON payments(tenant_id);

ALTER TABLE schedules ADD COLUMN tenant_id UUID REFERENCES tenants(id) ON DELETE CASCADE;
CREATE INDEX idx_schedules_tenant ON schedules(tenant_id) WHERE deleted_at IS NULL;

ALTER TABLE recurring_schedules ADD COLUMN tenant_id UUID REFERENCES tenants(id) ON DELETE CASCADE;
CREATE INDEX idx_recurring_schedules_tenant ON recurring_schedules(tenant_id) WHERE deleted_at IS NULL;

ALTER TABLE export_jobs ADD COLUMN tenant_id UUID REFERENCES tenants(id) ON DELETE CASCADE;
CREATE INDEX idx_export_jobs_tenant ON export_jobs(tenant_id);

ALTER TABLE import_jobs ADD COLUMN tenant_id UUID REFERENCES tenants(id) ON DELETE CASCADE;
CREATE INDEX idx_import_jobs_tenant ON import_jobs(tenant_id);

ALTER TABLE invoice_templates ADD COLUMN tenant_id UUID REFERENCES tenants(id) ON DELETE CASCADE;
CREATE INDEX idx_invoice_templates_tenant ON invoice_templates(tenant_id);

ALTER TABLE payment_terms ADD COLUMN tenant_id UUID REFERENCES tenants(id) ON DELETE CASCADE;
CREATE INDEX idx_payment_terms_tenant ON payment_terms(tenant_id);

ALTER TABLE tax_rates ADD COLUMN tenant_id UUID REFERENCES tenants(id) ON DELETE CASCADE;
CREATE INDEX idx_tax_rates_tenant ON tax_rates(tenant_id);

-- Helper functions for tenant operations
CREATE OR REPLACE FUNCTION get_user_tenant_ids(p_user_id UUID)
RETURNS UUID[] AS $$
BEGIN
    RETURN ARRAY(
        SELECT tenant_id FROM tenant_users
        WHERE user_id = p_user_id AND is_active = TRUE
    );
END;
$$ LANGUAGE plpgsql STABLE;

CREATE OR REPLACE FUNCTION user_belongs_to_tenant(p_user_id UUID, p_tenant_id UUID)
RETURNS BOOLEAN AS $$
BEGIN
    RETURN EXISTS (
        SELECT 1 FROM tenant_users
        WHERE user_id = p_user_id AND tenant_id = p_tenant_id AND is_active = TRUE
    );
END;
$$ LANGUAGE plpgsql STABLE;

CREATE OR REPLACE FUNCTION get_user_tenant_role(p_user_id UUID, p_tenant_id UUID)
RETURNS UUID AS $$
DECLARE
    v_role_id UUID;
BEGIN
    SELECT role_id INTO v_role_id FROM tenant_users
    WHERE user_id = p_user_id AND tenant_id = p_tenant_id AND is_active = TRUE;
    RETURN v_role_id;
END;
$$ LANGUAGE plpgsql STABLE;

-- ============================================================================
-- 17. SEED DATA
-- ============================================================================

-- Default roles
INSERT INTO roles (id, name, description, hierarchy_level) VALUES
    ('00000000-0000-0000-0000-000000000001', 'super_admin', 'Super administrator with full system access', 100),
    ('00000000-0000-0000-0000-000000000002', 'admin', 'Administrator with elevated privileges', 80),
    ('00000000-0000-0000-0000-000000000003', 'manager', 'Manager with team oversight capabilities', 60),
    ('00000000-0000-0000-0000-000000000004', 'user', 'Standard user with basic access', 40),
    ('00000000-0000-0000-0000-000000000005', 'guest', 'Guest user with read-only access', 20)
ON CONFLICT (name) DO NOTHING;

-- Default permissions (all resources)
INSERT INTO permissions (id, name, description, resource, action) VALUES
    -- Users (5)
    ('10000000-0000-0000-0000-000000000001', 'users.create', 'Create new users', 'users', 'create'),
    ('10000000-0000-0000-0000-000000000002', 'users.read', 'View user information', 'users', 'read'),
    ('10000000-0000-0000-0000-000000000003', 'users.update', 'Update user information', 'users', 'update'),
    ('10000000-0000-0000-0000-000000000004', 'users.delete', 'Delete users', 'users', 'delete'),
    ('10000000-0000-0000-0000-000000000005', 'users.list', 'List all users', 'users', 'list'),
    -- Roles (5)
    ('20000000-0000-0000-0000-000000000001', 'roles.create', 'Create new roles', 'roles', 'create'),
    ('20000000-0000-0000-0000-000000000002', 'roles.read', 'View role information', 'roles', 'read'),
    ('20000000-0000-0000-0000-000000000003', 'roles.update', 'Update role information', 'roles', 'update'),
    ('20000000-0000-0000-0000-000000000004', 'roles.delete', 'Delete roles', 'roles', 'delete'),
    ('20000000-0000-0000-0000-000000000005', 'roles.assign', 'Assign roles to users', 'roles', 'assign'),
    -- Customers (5)
    ('30000000-0000-0000-0000-000000000001', 'customers.create', 'Create new customers', 'customers', 'create'),
    ('30000000-0000-0000-0000-000000000002', 'customers.read', 'View customer information', 'customers', 'read'),
    ('30000000-0000-0000-0000-000000000003', 'customers.update', 'Update customer information', 'customers', 'update'),
    ('30000000-0000-0000-0000-000000000004', 'customers.delete', 'Delete customers', 'customers', 'delete'),
    ('30000000-0000-0000-0000-000000000005', 'customers.list', 'List all customers', 'customers', 'list'),
    -- Jobs (6)
    ('31000000-0000-0000-0000-000000000001', 'jobs.create', 'Create new jobs', 'jobs', 'create'),
    ('31000000-0000-0000-0000-000000000002', 'jobs.read', 'View job information', 'jobs', 'read'),
    ('31000000-0000-0000-0000-000000000003', 'jobs.update', 'Update job information', 'jobs', 'update'),
    ('31000000-0000-0000-0000-000000000004', 'jobs.delete', 'Delete jobs', 'jobs', 'delete'),
    ('31000000-0000-0000-0000-000000000005', 'jobs.list', 'List all jobs', 'jobs', 'list'),
    ('31000000-0000-0000-0000-000000000006', 'jobs.assign', 'Assign technicians to jobs', 'jobs', 'assign'),
    -- Quotes (6)
    ('32000000-0000-0000-0000-000000000001', 'quotes.create', 'Create new quotes', 'quotes', 'create'),
    ('32000000-0000-0000-0000-000000000002', 'quotes.read', 'View quote information', 'quotes', 'read'),
    ('32000000-0000-0000-0000-000000000003', 'quotes.update', 'Update quote information', 'quotes', 'update'),
    ('32000000-0000-0000-0000-000000000004', 'quotes.delete', 'Delete quotes', 'quotes', 'delete'),
    ('32000000-0000-0000-0000-000000000005', 'quotes.list', 'List all quotes', 'quotes', 'list'),
    ('32000000-0000-0000-0000-000000000006', 'quotes.approve', 'Approve or reject quotes', 'quotes', 'approve'),
    -- Invoices (6)
    ('33000000-0000-0000-0000-000000000001', 'invoices.create', 'Create new invoices', 'invoices', 'create'),
    ('33000000-0000-0000-0000-000000000002', 'invoices.read', 'View invoice information', 'invoices', 'read'),
    ('33000000-0000-0000-0000-000000000003', 'invoices.update', 'Update invoice information', 'invoices', 'update'),
    ('33000000-0000-0000-0000-000000000004', 'invoices.delete', 'Delete invoices', 'invoices', 'delete'),
    ('33000000-0000-0000-0000-000000000005', 'invoices.list', 'List all invoices', 'invoices', 'list'),
    ('33000000-0000-0000-0000-000000000006', 'invoices.execute', 'Send invoices to customers', 'invoices', 'execute'),
    -- Payments (4)
    ('34000000-0000-0000-0000-000000000001', 'payments.create', 'Record new payments', 'payments', 'create'),
    ('34000000-0000-0000-0000-000000000002', 'payments.read', 'View payment information', 'payments', 'read'),
    ('34000000-0000-0000-0000-000000000003', 'payments.list', 'List all payments', 'payments', 'list'),
    ('34000000-0000-0000-0000-000000000004', 'payments.refund', 'Process payment refunds', 'payments', 'admin'),
    -- Payment Terms (5)
    ('35000000-0000-0000-0000-000000000001', 'payment_terms.create', 'Create payment terms', 'payment_terms', 'create'),
    ('35000000-0000-0000-0000-000000000002', 'payment_terms.read', 'View payment terms', 'payment_terms', 'read'),
    ('35000000-0000-0000-0000-000000000003', 'payment_terms.update', 'Update payment terms', 'payment_terms', 'update'),
    ('35000000-0000-0000-0000-000000000004', 'payment_terms.delete', 'Delete payment terms', 'payment_terms', 'delete'),
    ('35000000-0000-0000-0000-000000000005', 'payment_terms.list', 'List payment terms', 'payment_terms', 'list'),
    -- Reports (3)
    ('36000000-0000-0000-0000-000000000001', 'reports.view', 'View reports', 'reports', 'view'),
    ('36000000-0000-0000-0000-000000000002', 'reports.export', 'Export reports', 'reports', 'export'),
    ('36000000-0000-0000-0000-000000000003', 'reports.admin', 'Administer report settings', 'reports', 'admin'),
    -- Exports (3)
    ('37000000-0000-0000-0000-000000000001', 'exports.create', 'Create data exports', 'exports', 'create'),
    ('37000000-0000-0000-0000-000000000002', 'exports.read', 'View export status', 'exports', 'read'),
    ('37000000-0000-0000-0000-000000000003', 'exports.list', 'List all exports', 'exports', 'list'),
    -- Tenants (6)
    ('38000000-0000-0000-0000-000000000001', 'tenants.create', 'Create new organizations', 'tenants', 'create'),
    ('38000000-0000-0000-0000-000000000002', 'tenants.read', 'View organization information', 'tenants', 'read'),
    ('38000000-0000-0000-0000-000000000003', 'tenants.update', 'Update organization settings', 'tenants', 'update'),
    ('38000000-0000-0000-0000-000000000004', 'tenants.delete', 'Delete organizations', 'tenants', 'delete'),
    ('38000000-0000-0000-0000-000000000005', 'tenants.list', 'List all organizations', 'tenants', 'list'),
    ('38000000-0000-0000-0000-000000000006', 'tenants.manage', 'Manage organization members', 'tenants', 'manage'),
    -- Settings (2)
    ('39000000-0000-0000-0000-000000000001', 'settings.read', 'View system settings', 'settings', 'read'),
    ('39000000-0000-0000-0000-000000000002', 'settings.update', 'Update system settings', 'settings', 'update')
ON CONFLICT (name) DO NOTHING;

-- Assign all permissions to super_admin
INSERT INTO role_permissions (role_id, permission_id)
SELECT '00000000-0000-0000-0000-000000000001', id FROM permissions
ON CONFLICT (role_id, permission_id) DO NOTHING;

-- Admin permissions (all except tenant deletion and role management)
INSERT INTO role_permissions (role_id, permission_id)
SELECT '00000000-0000-0000-0000-000000000002', id FROM permissions
WHERE name NOT IN ('tenants.delete', 'roles.create', 'roles.update', 'roles.delete')
ON CONFLICT (role_id, permission_id) DO NOTHING;

-- Manager permissions (CRUD on business resources, reports view)
INSERT INTO role_permissions (role_id, permission_id)
SELECT '00000000-0000-0000-0000-000000000003', id FROM permissions
WHERE name IN (
    'customers.create', 'customers.read', 'customers.update', 'customers.list',
    'jobs.create', 'jobs.read', 'jobs.update', 'jobs.list', 'jobs.assign',
    'quotes.create', 'quotes.read', 'quotes.update', 'quotes.list', 'quotes.approve',
    'invoices.create', 'invoices.read', 'invoices.update', 'invoices.list', 'invoices.execute',
    'payments.create', 'payments.read', 'payments.list',
    'payment_terms.read', 'payment_terms.list',
    'reports.view', 'reports.export',
    'exports.create', 'exports.read', 'exports.list',
    'tenants.read',
    'users.read', 'users.list',
    'roles.read',
    'settings.read'
)
ON CONFLICT (role_id, permission_id) DO NOTHING;

-- User permissions (read-only access)
INSERT INTO role_permissions (role_id, permission_id)
SELECT '00000000-0000-0000-0000-000000000004', id FROM permissions
WHERE name IN (
    'customers.read', 'customers.list',
    'jobs.read', 'jobs.list',
    'quotes.read', 'quotes.list',
    'invoices.read', 'invoices.list',
    'payments.read', 'payments.list',
    'payment_terms.read', 'payment_terms.list',
    'reports.view',
    'exports.read',
    'tenants.read',
    'users.read',
    'roles.read',
    'settings.read'
)
ON CONFLICT (role_id, permission_id) DO NOTHING;

-- Guest permissions (minimal read access)
INSERT INTO role_permissions (role_id, permission_id)
SELECT '00000000-0000-0000-0000-000000000005', id FROM permissions
WHERE name IN (
    'customers.read',
    'jobs.read',
    'quotes.read',
    'invoices.read',
    'payment_terms.read',
    'tenants.read',
    'settings.read'
)
ON CONFLICT (role_id, permission_id) DO NOTHING;

-- Default customer segments
INSERT INTO customer_segments (name, description, criteria, color, priority) VALUES
    ('high_value', 'High-value customers with significant revenue', '{"min_revenue": 5000, "min_jobs": 5}', '#22C55E', 100),
    ('regular', 'Regular customers with consistent activity', '{"min_revenue": 1000, "min_jobs": 2, "max_days_inactive": 180}', '#3B82F6', 80),
    ('occasional', 'Occasional customers with infrequent orders', '{"min_revenue": 100, "min_jobs": 1, "max_days_inactive": 365}', '#F59E0B', 60),
    ('at_risk', 'Customers showing signs of churn', '{"min_days_inactive": 90, "max_days_inactive": 180, "min_jobs": 1}', '#EF4444', 90),
    ('churned', 'Inactive customers', '{"min_days_inactive": 365}', '#6B7280', 40),
    ('new', 'Recently acquired customers', '{"max_lifetime_days": 30}', '#8B5CF6', 70)
ON CONFLICT (name) DO NOTHING;

-- Default subscription plans
INSERT INTO subscription_plans (name, description, tier, monthly_price, annual_price, quarterly_price, trial_days, is_active, is_public, sort_order) VALUES
    ('Free', 'Get started with basic features', 'free', 0, 0, 0, 0, true, true, 1),
    ('Starter', 'Perfect for small teams', 'starter', 2900, 29000, 7830, 14, true, true, 2),
    ('Professional', 'Advanced features for professional teams', 'professional', 7900, 79000, 21330, 14, true, true, 3),
    ('Business', 'Full-featured solution for large organizations', 'business', 19900, 199000, 53730, 14, true, true, 4),
    ('Enterprise', 'Custom solutions for enterprise needs', 'enterprise', 0, 0, 0, 30, true, true, 5)
ON CONFLICT DO NOTHING;

-- Default payment terms
INSERT INTO payment_terms (name, description, term_type, days_until_due, is_default, is_active) VALUES
    ('Due on Receipt', 'Payment due immediately upon receipt', 'due_on_receipt', 0, false, true),
    ('Net 15', 'Payment due within 15 days', 'net_15', 15, false, true),
    ('Net 30', 'Payment due within 30 days', 'net_30', 30, true, true),
    ('Net 60', 'Payment due within 60 days', 'net_60', 60, false, true)
ON CONFLICT DO NOTHING;

-- Default holidays (US)
INSERT INTO holidays (name, date, country, is_recurring) VALUES
    ('New Year''s Day', '2025-01-01', 'US', true),
    ('Independence Day', '2025-07-04', 'US', true),
    ('Veterans Day', '2025-11-11', 'US', true),
    ('Christmas Day', '2025-12-25', 'US', true)
ON CONFLICT DO NOTHING;

-- ============================================================================
-- REVENUE TRACKING
-- ============================================================================

-- Revenue transactions table
CREATE TABLE revenue_transactions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    payment_id UUID NOT NULL,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    invoice_id UUID REFERENCES invoices(id) ON DELETE SET NULL,
    order_id UUID,

    -- Transaction details
    transaction_type VARCHAR(50) NOT NULL CHECK (transaction_type IN ('payment', 'refund', 'chargeback', 'adjustment')),
    transaction_date TIMESTAMP WITH TIME ZONE NOT NULL,

    -- Amounts
    gross_amount DECIMAL(12,2) NOT NULL,
    net_amount DECIMAL(12,2) NOT NULL,
    fee_amount DECIMAL(12,2) NOT NULL DEFAULT 0,
    tax_amount DECIMAL(12,2) NOT NULL DEFAULT 0,
    discount_amount DECIMAL(12,2) NOT NULL DEFAULT 0,
    refund_amount DECIMAL(12,2) NOT NULL DEFAULT 0,

    -- Currency and exchange
    currency VARCHAR(3) NOT NULL DEFAULT 'USD',
    exchange_rate DECIMAL(10,4) DEFAULT 1.0000,
    base_currency_amount DECIMAL(12,2),

    -- Category and tags
    revenue_category VARCHAR(100),
    revenue_subcategory VARCHAR(100),
    tags TEXT[],

    -- Status
    status VARCHAR(20) NOT NULL DEFAULT 'completed' CHECK (status IN ('completed', 'pending', 'failed', 'reversed')),
    is_refunded BOOLEAN NOT NULL DEFAULT FALSE,
    is_disputed BOOLEAN NOT NULL DEFAULT FALSE,

    -- Metadata
    metadata JSONB,

    -- Timestamps
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_revenue_transactions_user ON revenue_transactions(user_id);
CREATE INDEX idx_revenue_transactions_date ON revenue_transactions(transaction_date);
CREATE INDEX idx_revenue_transactions_type ON revenue_transactions(transaction_type);
CREATE INDEX idx_revenue_transactions_status ON revenue_transactions(status);
CREATE INDEX idx_revenue_transactions_currency ON revenue_transactions(currency);
CREATE INDEX idx_revenue_transactions_category ON revenue_transactions(revenue_category);

CREATE TRIGGER trigger_revenue_transactions_updated_at BEFORE UPDATE ON revenue_transactions
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- Revenue summary cache table
CREATE TABLE revenue_summary_cache (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    period_type VARCHAR(20) NOT NULL CHECK (period_type IN ('daily', 'weekly', 'monthly', 'quarterly', 'yearly')),
    period_start TIMESTAMP WITH TIME ZONE NOT NULL,
    period_end TIMESTAMP WITH TIME ZONE NOT NULL,

    -- Aggregated metrics
    total_gross_revenue DECIMAL(12,2) NOT NULL DEFAULT 0,
    total_net_revenue DECIMAL(12,2) NOT NULL DEFAULT 0,
    total_fees DECIMAL(12,2) NOT NULL DEFAULT 0,
    total_taxes DECIMAL(12,2) NOT NULL DEFAULT 0,
    total_discounts DECIMAL(12,2) NOT NULL DEFAULT 0,
    total_refunds DECIMAL(12,2) NOT NULL DEFAULT 0,

    -- Transaction counts
    transaction_count INT NOT NULL DEFAULT 0,
    payment_count INT NOT NULL DEFAULT 0,
    refund_count INT NOT NULL DEFAULT 0,
    unique_customer_count INT NOT NULL DEFAULT 0,

    -- Average metrics
    average_transaction_value DECIMAL(12,2),
    average_order_value DECIMAL(12,2),

    -- Growth metrics
    growth_rate DECIMAL(5,2),
    previous_period_revenue DECIMAL(12,2),

    -- Metadata
    currency VARCHAR(3) DEFAULT 'USD',
    metadata JSONB,

    -- Cache control
    last_updated TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    is_stale BOOLEAN NOT NULL DEFAULT FALSE,

    UNIQUE(period_type, period_start, period_end, currency)
);

CREATE INDEX idx_revenue_summary_cache_period ON revenue_summary_cache(period_type, period_start, period_end);
CREATE INDEX idx_revenue_summary_cache_currency ON revenue_summary_cache(currency);

-- Revenue alerts table
CREATE TABLE revenue_alerts (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(200) NOT NULL,
    description TEXT,
    condition JSONB NOT NULL,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    last_triggered TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE TRIGGER trigger_revenue_alerts_updated_at BEFORE UPDATE ON revenue_alerts
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- ============================================================================
-- SCHEMA COMPLETE
-- ============================================================================

DO $$
BEGIN
    RAISE NOTICE 'ServicePro database schema created successfully.';
END $$;
