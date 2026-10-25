# ServicePro Database Architecture

> **Version**: 1.1
> **Last Updated**: 2026-10-25
> **Database**: PostgreSQL 18 — Neon in production (AWS us-east-2), Docker locally
>
> **Source of truth:** `backend/migrations/` (`000001_initial_schema.up.sql` +
> later pairs). The diagrams and table sketches below are a guide; where they
> differ from the migrations, the migrations win.

## Table of Contents

1. [Overview](#1-overview)
2. [Schema Diagram](#2-schema-diagram)
3. [Core Tables](#3-core-tables)
4. [Domain Tables](#4-domain-tables)
5. [Supporting Tables](#5-supporting-tables)
6. [Views and Functions](#6-views-and-functions)
7. [Indexes](#7-indexes)
8. [Partitioning](#8-partitioning)
9. [Data Integrity](#9-data-integrity)
10. [Performance Considerations](#10-performance-considerations)

---

## 1. Overview

### 1.1 Database Statistics

Counted from `000001_initial_schema.up.sql` (2026-10-25):

| Metric      | Value                                                       |
| ----------- | ----------------------------------------------------------- |
| Tables      | 53 (51 in `public`, 2 in `analytics`)                       |
| Views       | 8 (`v_*` reporting views, no materialized views)            |
| Functions   | 15                                                          |
| Enum types  | 16                                                          |
| Triggers    | 19 (17 `updated_at`, quote item totals, invoice numbering)  |

### 1.2 Naming Conventions

| Element      | Convention                 | Example                        |
| ------------ | -------------------------- | ------------------------------ |
| Tables       | snake_case, plural         | `customers`, `job_assignments` |
| Columns      | snake_case                 | `first_name`, `created_at`     |
| Primary Keys | `id` (UUID)                | `id UUID PRIMARY KEY`          |
| Foreign Keys | `{table}_id`               | `customer_id`, `job_id`        |
| Indexes      | `idx_{table}_{columns}`    | `idx_customers_email`          |
| Constraints  | `{table}_{type}_{columns}` | `customers_email_unique`       |

### 1.3 Common Patterns

All tables include:

- `id` - UUID primary key (auto-generated)
- `created_at` - Timestamp of creation
- `updated_at` - Timestamp of last update
- `deleted_at` - Soft delete timestamp (nullable)

### 1.4 Migrations

golang-migrate, embedded in the binary, **always additive** (CLAUDE.md).

| Migration                   | What                                                                                                   |
| --------------------------- | ------------------------------------------------------------------------------------------------------ |
| `000001_initial_schema`     | The whole pre-launch schema. Migrations 000001–000014 were squashed into it on 2026-10-24, before any shared database held data (one-off) |
| `000002_users_auth_user_id` | `users.auth_user_id` — links a user to its Neon Auth identity (T158)                                   |

Production migrations run in the release pipeline's `migrate` job, never at
server startup. Locally: `make migrate`, `make db-fresh-seed`.

---

## 2. Schema Diagram

### 2.1 High-Level Domain Model

```mermaid
erDiagram
    USERS ||--o{ CUSTOMERS : manages
    USERS ||--o{ JOBS : creates
    USERS ||--o{ QUOTES : creates
    USERS ||--o{ INVOICES : creates

    CUSTOMERS ||--o{ JOBS : has
    CUSTOMERS ||--o{ QUOTES : receives
    CUSTOMERS ||--o{ INVOICES : billed

    JOBS ||--o{ SCHEDULES : scheduled
    JOBS ||--|| INVOICES : billed_via

    QUOTES ||--|| JOBS : converts_to
    QUOTES ||--|| INVOICES : converts_to

    INVOICES ||--o{ PAYMENTS : receives
```

### 2.2 User Management Schema

```mermaid
erDiagram
    users ||--o{ user_roles : has
    roles ||--o{ user_roles : assigned_to
    roles ||--o{ role_permissions : has
    permissions ||--o{ role_permissions : granted_to
    roles ||--o| roles : parent_of

    users {
        uuid id PK
        string email UK
        string password_hash
        string role
        int failed_login_count
        timestamp last_failed_login_at
        timestamp locked_until
        boolean email_verified
        timestamp verification_sent_at
        timestamp created_at
        timestamp updated_at
        timestamp deleted_at
    }

    roles {
        uuid id PK
        string name UK
        string description
        uuid parent_role_id FK
        int hierarchy_level
        timestamp created_at
        timestamp updated_at
        timestamp deleted_at
    }

    permissions {
        uuid id PK
        string name UK
        string description
        string resource
        string action
        timestamp created_at
        timestamp updated_at
        timestamp deleted_at
    }

    user_roles {
        uuid user_id PK,FK
        uuid role_id PK,FK
        timestamp assigned_at
        uuid assigned_by FK
        timestamp expires_at
    }

    role_permissions {
        uuid role_id PK,FK
        uuid permission_id PK,FK
        timestamp granted_at
        uuid granted_by FK
        timestamp deleted_at
    }
```

### 2.3 Customer and Job Schema

```mermaid
erDiagram
    customers ||--o{ jobs : has
    jobs ||--o{ job_assignments : has
    jobs ||--o{ job_materials : uses
    jobs ||--o{ job_notes : has
    users ||--o{ job_assignments : assigned_to

    customers {
        uuid id PK
        string first_name
        string last_name
        string company_name
        string email UK
        string phone_primary
        string phone_secondary
        string billing_street
        string billing_city
        string billing_state
        string billing_zip
        string service_street
        string service_city
        string service_state
        string service_zip
        enum customer_type
        enum status
        text notes
        timestamp created_at
        timestamp updated_at
        timestamp deleted_at
    }

    jobs {
        uuid id PK
        string job_number UK
        uuid customer_id FK
        string title
        text description
        enum job_type
        enum status
        enum priority
        timestamp scheduled_start_at
        timestamp scheduled_end_at
        timestamp actual_start_at
        timestamp actual_end_at
        int estimated_duration
        int actual_duration
        string service_street
        string service_city
        string service_state
        string service_zip
        decimal estimated_cost
        decimal actual_cost
        decimal tax_amount
        decimal total_amount
        text internal_notes
        text customer_notes
        text completion_notes
        boolean requires_follow_up
        date follow_up_date
        uuid created_by FK
        uuid updated_by FK
        timestamp created_at
        timestamp updated_at
        timestamp deleted_at
    }

    job_assignments {
        uuid id PK
        uuid job_id FK
        uuid user_id FK
        string role
        timestamp assigned_at
        timestamp unassigned_at
        decimal hours_worked
        text notes
        timestamp created_at
        timestamp updated_at
        timestamp deleted_at
    }

    job_materials {
        uuid id PK
        uuid job_id FK
        string name
        text description
        string sku
        decimal quantity
        string unit
        decimal unit_cost
        decimal total_cost
        boolean billable
        timestamp created_at
        timestamp updated_at
        timestamp deleted_at
    }

    job_notes {
        uuid id PK
        uuid job_id FK
        uuid user_id FK
        text note
        boolean is_internal
        timestamp created_at
        timestamp updated_at
        timestamp deleted_at
    }
```

### 2.4 Quote and Invoice Schema

```mermaid
erDiagram
    customers ||--o{ quotes : receives
    quotes ||--o{ quote_items : contains
    customers ||--o{ invoices : billed
    invoices ||--o{ invoice_lines : contains
    invoices ||--o{ invoice_payments : receives
    invoices ||--o{ invoice_audit_log : tracks
    payment_terms ||--o{ invoices : applies_to
    tax_rates ||--o{ invoices : applies_to

    quotes {
        uuid id PK
        uuid customer_id FK
        string quote_number UK
        enum status
        date valid_until
        decimal subtotal
        decimal tax_rate
        decimal tax_amount
        decimal total
        text notes
        text terms
        uuid created_by FK
        uuid updated_by FK
        timestamp created_at
        timestamp updated_at
        timestamp deleted_at
    }

    quote_items {
        uuid id PK
        uuid quote_id FK
        text description
        decimal quantity
        decimal unit_price
        decimal total
        int sort_order
        timestamp created_at
        timestamp updated_at
        timestamp deleted_at
    }

    invoices {
        uuid id PK
        string invoice_number UK
        uuid customer_id FK
        uuid job_id FK
        uuid quote_id FK
        enum status
        date issue_date
        date due_date
        timestamp sent_date
        timestamp viewed_date
        timestamp paid_date
        decimal subtotal
        decimal tax_amount
        decimal discount_amount
        decimal total_amount
        decimal amount_paid
        decimal amount_due
        uuid payment_term_id FK
        uuid tax_rate_id FK
        string po_number
        text notes
        text terms_and_conditions
        text footer_text
        uuid created_by FK
        uuid updated_by FK
        timestamp created_at
        timestamp updated_at
        timestamp deleted_at
    }

    invoice_lines {
        uuid id PK
        uuid invoice_id FK
        text description
        decimal quantity
        decimal unit_price
        string unit_of_measure
        decimal discount_percentage
        decimal discount_amount
        boolean taxable
        decimal tax_rate
        decimal tax_amount
        decimal line_total
        decimal line_total_with_tax
        uuid product_id FK
        uuid service_id FK
        int sort_order
        timestamp created_at
        timestamp updated_at
    }

    invoice_payments {
        uuid id PK
        uuid invoice_id FK
        decimal amount
        date payment_date
        string payment_method
        string reference_number
        text notes
        uuid created_by FK
        timestamp created_at
        timestamp updated_at
    }

    payment_terms {
        uuid id PK
        string name
        text description
        enum term_type
        int days_until_due
        decimal discount_percentage
        int discount_days
        decimal late_fee_percentage
        decimal late_fee_amount
        int grace_period_days
        boolean is_default
        boolean is_active
        timestamp created_at
        timestamp updated_at
    }

    tax_rates {
        uuid id PK
        string name
        text description
        decimal rate
        enum tax_type
        string region
        boolean is_compound
        boolean is_active
        date effective_date
        date expiry_date
        timestamp created_at
        timestamp updated_at
    }
```

### 2.5 Payment Schema

```mermaid
erDiagram
    invoices ||--o{ payments : receives
    payments ||--o{ payment_refunds : may_have
    payments ||--|| payment_status : has
    payment_status ||--o{ payment_status_history : tracks

    payments {
        uuid id PK
        uuid invoice_id FK
        uuid customer_id FK
        string stripe_payment_intent_id
        string stripe_charge_id
        string stripe_customer_id
        decimal amount
        string currency
        enum status
        string payment_method
        text description
        jsonb metadata
        string error_code
        text error_message
        timestamp created_at
        timestamp updated_at
    }

    payment_refunds {
        uuid id PK
        uuid payment_id FK
        string stripe_refund_id
        decimal amount
        string reason
        enum status
        text notes
        uuid created_by FK
        timestamp created_at
        timestamp updated_at
    }

    payment_status {
        uuid id PK
        uuid payment_id FK,UK
        string status
        string previous_status
        string status_category
        jsonb metadata
        uuid updated_by FK
        timestamp updated_at
        timestamp created_at
        int version
    }

    payment_status_history {
        uuid id PK
        uuid payment_id FK
        string from_status
        string to_status
        string from_category
        string to_category
        text reason
        jsonb metadata
        uuid updated_by FK
        string updated_by_type
        timestamp created_at
    }
```

### 2.6 Scheduling Schema

```mermaid
erDiagram
    jobs ||--o{ schedules : has
    recurring_schedules ||--o{ schedules : generates
    recurring_schedules ||--o{ schedule_exceptions : has
    schedules ||--o{ schedule_conflicts : may_have

    schedules {
        uuid id PK
        uuid job_id FK
        string title
        text description
        timestamp start_time
        timestamp end_time
        boolean all_day
        string recurrence_type
        uuid recurring_schedule_id FK
        uuid[] assigned_tech_ids
        string location
        boolean is_confirmed
        timestamp confirmed_at
        uuid confirmed_by FK
        boolean is_cancelled
        timestamp cancelled_at
        uuid cancelled_by FK
        text cancellation_reason
        boolean reminders_enabled
        timestamp reminder_time
        string color
        text notes
        uuid created_by FK
        uuid updated_by FK
        timestamp created_at
        timestamp updated_at
        timestamp deleted_at
    }

    recurring_schedules {
        uuid id PK
        string title
        text description
        string recurrence_type
        string recurrence_rule
        date start_date
        date end_date
        int interval
        int[] days_of_week
        int day_of_month
        int month_of_year
        int occurrences
        time time_start
        time time_end
        int duration
        uuid[] assigned_tech_ids
        string location
        uuid job_template_id FK
        boolean is_active
        uuid created_by FK
        uuid updated_by FK
        timestamp created_at
        timestamp updated_at
        timestamp deleted_at
    }

    schedule_exceptions {
        uuid id PK
        uuid recurring_pattern_id FK
        date exception_date
        enum exception_type
        text reason
        date rescheduled_date
        jsonb modifications
        uuid created_by FK
        timestamp created_at
        timestamp updated_at
        timestamp deleted_at
    }

    schedule_conflicts {
        uuid id PK
        uuid schedule_id_1 FK
        uuid schedule_id_2 FK
        enum conflict_type
        string conflicting_resource
        enum severity
        text description
        boolean is_resolved
        timestamp resolved_at
        uuid resolved_by FK
        text resolution_notes
        timestamp detected_at
        timestamp created_at
        timestamp updated_at
    }

    holidays {
        uuid id PK
        string name
        date date
        string country
        boolean is_recurring
        timestamp created_at
        timestamp updated_at
        timestamp deleted_at
    }
```

---

## 3. Core Tables

### 3.1 Users Table

```sql
CREATE TABLE users (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email           VARCHAR(255) NOT NULL UNIQUE,
    password_hash   VARCHAR(255) NOT NULL,
    role            VARCHAR(50) NOT NULL DEFAULT 'user',
    failed_login_count INTEGER DEFAULT 0,
    last_failed_login_at TIMESTAMP,
    locked_until    TIMESTAMP,
    email_verified  BOOLEAN DEFAULT FALSE,
    verification_sent_at TIMESTAMP,
    created_at      TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at      TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    deleted_at      TIMESTAMP
);

-- Indexes
CREATE INDEX idx_users_email ON users(email) WHERE deleted_at IS NULL;
CREATE INDEX idx_users_locked_until ON users(locked_until) WHERE locked_until IS NOT NULL;
CREATE INDEX idx_users_role ON users(role) WHERE deleted_at IS NULL;
```

### 3.2 Customers Table

```sql
CREATE TABLE customers (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    first_name      VARCHAR(100) NOT NULL,
    last_name       VARCHAR(100) NOT NULL,
    company_name    VARCHAR(200),
    email           VARCHAR(255) UNIQUE,
    phone_primary   VARCHAR(20),
    phone_secondary VARCHAR(20),
    billing_street  VARCHAR(255),
    billing_city    VARCHAR(100),
    billing_state   CHAR(2),
    billing_zip     VARCHAR(10),
    service_street  VARCHAR(255),
    service_city    VARCHAR(100),
    service_state   CHAR(2),
    service_zip     VARCHAR(10),
    customer_type   customer_type NOT NULL DEFAULT 'residential',
    status          customer_status NOT NULL DEFAULT 'active',
    notes           TEXT,
    created_at      TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at      TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    deleted_at      TIMESTAMP
);

-- Full-text search
CREATE INDEX idx_customers_fulltext ON customers
    USING gin(to_tsvector('english',
        coalesce(first_name, '') || ' ' ||
        coalesce(last_name, '') || ' ' ||
        coalesce(company_name, '')));
```

---

## 4. Domain Tables

### 4.1 Jobs Table

```sql
CREATE TABLE jobs (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    job_number          VARCHAR(20) UNIQUE NOT NULL,
    customer_id         UUID NOT NULL REFERENCES customers(id),
    title               VARCHAR(255) NOT NULL,
    description         TEXT,
    job_type            job_type NOT NULL DEFAULT 'maintenance',
    status              job_status NOT NULL DEFAULT 'scheduled',
    priority            job_priority NOT NULL DEFAULT 'normal',
    scheduled_start_at  TIMESTAMP,
    scheduled_end_at    TIMESTAMP,
    actual_start_at     TIMESTAMP,
    actual_end_at       TIMESTAMP,
    estimated_duration  INTEGER, -- minutes
    actual_duration     INTEGER GENERATED ALWAYS AS (
        EXTRACT(EPOCH FROM (actual_end_at - actual_start_at)) / 60
    ) STORED,
    service_street      VARCHAR(255),
    service_city        VARCHAR(100),
    service_state       CHAR(2),
    service_zip         VARCHAR(10),
    estimated_cost      DECIMAL(10,2) DEFAULT 0,
    actual_cost         DECIMAL(10,2) DEFAULT 0,
    tax_amount          DECIMAL(10,2) DEFAULT 0,
    total_amount        DECIMAL(10,2) GENERATED ALWAYS AS (
        actual_cost + tax_amount
    ) STORED,
    internal_notes      TEXT,
    customer_notes      TEXT,
    completion_notes    TEXT,
    requires_follow_up  BOOLEAN DEFAULT FALSE,
    follow_up_date      DATE,
    created_by          UUID REFERENCES users(id),
    updated_by          UUID REFERENCES users(id),
    created_at          TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at          TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    deleted_at          TIMESTAMP
);

-- Job number generation trigger
CREATE OR REPLACE FUNCTION generate_job_number()
RETURNS TRIGGER AS $$
BEGIN
    NEW.job_number := 'JOB-' || TO_CHAR(CURRENT_DATE, 'YYYYMMDD') || '-' ||
        LPAD(CAST((SELECT COALESCE(MAX(
            CAST(SUBSTRING(job_number FROM 14) AS INTEGER)
        ), 0) + 1 FROM jobs
        WHERE job_number LIKE 'JOB-' || TO_CHAR(CURRENT_DATE, 'YYYYMMDD') || '-%'
        ) AS VARCHAR), 4, '0');
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER set_job_number
    BEFORE INSERT ON jobs
    FOR EACH ROW
    WHEN (NEW.job_number IS NULL)
    EXECUTE FUNCTION generate_job_number();
```

### 4.2 Invoices Table

```sql
CREATE TABLE invoices (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    invoice_number      VARCHAR(20) UNIQUE NOT NULL,
    customer_id         UUID NOT NULL REFERENCES customers(id),
    job_id              UUID REFERENCES jobs(id),
    quote_id            UUID REFERENCES quotes(id),
    status              invoice_status NOT NULL DEFAULT 'draft',
    issue_date          DATE NOT NULL DEFAULT CURRENT_DATE,
    due_date            DATE NOT NULL,
    sent_date           TIMESTAMP,
    viewed_date         TIMESTAMP,
    paid_date           TIMESTAMP,
    subtotal            DECIMAL(10,2) NOT NULL DEFAULT 0,
    tax_amount          DECIMAL(10,2) NOT NULL DEFAULT 0,
    discount_amount     DECIMAL(10,2) NOT NULL DEFAULT 0,
    total_amount        DECIMAL(10,2) NOT NULL DEFAULT 0,
    amount_paid         DECIMAL(10,2) NOT NULL DEFAULT 0,
    amount_due          DECIMAL(10,2) GENERATED ALWAYS AS (
        total_amount - amount_paid
    ) STORED,
    payment_term_id     UUID REFERENCES payment_terms(id),
    tax_rate_id         UUID REFERENCES tax_rates(id),
    po_number           VARCHAR(50),
    notes               TEXT,
    terms_and_conditions TEXT,
    footer_text         TEXT,
    created_by          UUID REFERENCES users(id),
    updated_by          UUID REFERENCES users(id),
    created_at          TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at          TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    deleted_at          TIMESTAMP,

    CONSTRAINT valid_due_date CHECK (due_date >= issue_date),
    CONSTRAINT valid_amounts CHECK (
        subtotal >= 0 AND
        tax_amount >= 0 AND
        discount_amount >= 0 AND
        total_amount >= 0 AND
        amount_paid >= 0
    )
);

-- Invoice number generation (INV-YYYY-00001)
CREATE OR REPLACE FUNCTION generate_invoice_number()
RETURNS TRIGGER AS $$
BEGIN
    NEW.invoice_number := 'INV-' || TO_CHAR(CURRENT_DATE, 'YYYY') || '-' ||
        LPAD(CAST((SELECT COALESCE(MAX(
            CAST(SUBSTRING(invoice_number FROM 10) AS INTEGER)
        ), 0) + 1 FROM invoices
        WHERE invoice_number LIKE 'INV-' || TO_CHAR(CURRENT_DATE, 'YYYY') || '-%'
        ) AS VARCHAR), 5, '0');
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;
```

---

## 5. Supporting Tables

### 5.1 Enum Types

```sql
CREATE TYPE customer_type AS ENUM ('residential', 'commercial');
CREATE TYPE customer_status AS ENUM ('active', 'inactive', 'prospect');

CREATE TYPE job_status AS ENUM ('new', 'scheduled', 'en_route', 'in_progress', 'on_hold',
                                'completed', 'invoiced', 'paid', 'cancelled');
CREATE TYPE job_priority AS ENUM ('low', 'normal', 'high', 'urgent');
CREATE TYPE job_type AS ENUM ('installation', 'maintenance', 'repair', 'inspection', 'emergency');

CREATE TYPE quote_status AS ENUM ('draft', 'sent', 'viewed', 'accepted', 'declined', 'expired');
CREATE TYPE invoice_status AS ENUM ('draft', 'sent', 'viewed', 'paid', 'partially_paid',
                                    'overdue', 'cancelled', 'refunded');

CREATE TYPE payment_term_type AS ENUM ('due_on_receipt', 'net_7', 'net_10', 'net_15',
                                       'net_30', 'net_60', 'net_90', 'custom');
CREATE TYPE tax_type AS ENUM ('sales_tax', 'vat', 'gst', 'hst', 'exempt');
CREATE TYPE template_status AS ENUM ('draft', 'active', 'archived', 'deprecated');

CREATE TYPE notification_type AS ENUM (
    'payment_due_soon', 'payment_overdue', 'payment_in_grace_period',
    'early_payment_discount_available', 'late_fee_applied',
    'payment_received', 'partial_payment_received'
);
CREATE TYPE notification_status AS ENUM ('pending', 'sent', 'failed', 'dismissed');
CREATE TYPE notification_channel AS ENUM ('email', 'sms', 'push', 'in_app', 'webhook');

CREATE TYPE invitation_status AS ENUM ('pending', 'accepted', 'expired', 'cancelled');
CREATE TYPE membership_status AS ENUM ('active', 'cancelled', 'expired', 'pending');
CREATE TYPE billing_cycle AS ENUM ('monthly', 'annual');
```

### 5.2 Audit Log Tables

```sql
CREATE TABLE invoice_audit_log (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    invoice_id      UUID NOT NULL REFERENCES invoices(id) ON DELETE CASCADE,
    action          VARCHAR(50) NOT NULL, -- 'created', 'updated', 'status_changed', etc.
    field_name      VARCHAR(100),
    old_value       TEXT,
    new_value       TEXT,
    from_status     invoice_status,
    to_status       invoice_status,
    changed_by      UUID REFERENCES users(id),
    changed_by_type VARCHAR(20), -- 'user', 'system', 'webhook'
    ip_address      INET,
    user_agent      TEXT,
    created_at      TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_invoice_audit_invoice_id ON invoice_audit_log(invoice_id);
CREATE INDEX idx_invoice_audit_created_at ON invoice_audit_log(created_at);
```

---

## 6. Views and Functions

### 6.1 Views

Eight plain (non-materialized) reporting views:

| View                      | Purpose                                   |
| ------------------------- | ----------------------------------------- |
| `v_platform_overview`     | Platform-wide totals (admin)              |
| `v_tenant_growth_daily`   | New tenants per day                       |
| `v_feature_adoption`      | Feature usage across tenants              |
| `v_job_metrics_daily`     | Job counts/status per day                 |
| `v_invoice_metrics_daily` | Invoice counts/amounts per day            |
| `v_quote_conversion`      | Quote → accepted conversion               |
| `v_revenue_daily`         | Revenue per day                           |
| `v_ar_aging_summary`      | Accounts-receivable aging buckets         |

Report endpoints that need caching use cache tables (`revenue_summary_cache`,
`customer_metrics_cache`) with explicit refresh endpoints, not materialized
views.

### 6.2 Functions

| Function                                                          | Purpose                                         |
| ----------------------------------------------------------------- | ----------------------------------------------- |
| `update_updated_at_column()`                                      | Trigger: keep `updated_at` current (17 tables)  |
| `validate_email`, `validate_phone`, `validate_state_code`, `validate_zip_code` | Check-constraint helpers           |
| `calculate_quote_item_total()`                                    | Trigger: quote line totals                      |
| `generate_quote_number`, `generate_invoice_number`                | Document numbering                              |
| `get_user_tenant_ids`, `user_belongs_to_tenant`, `get_user_tenant_role` | Tenant membership helpers                 |
| `get_tenant_subscription`, `check_tenant_limit`                   | Membership tier limits                          |
| `set_default_payment_method`, `get_default_payment_method`        | Tenant billing payment methods                  |

Quote expiry runs in the API (`POST /internal/jobs/quote-expiry`, hourly from
`scheduled-jobs.yml`), not as a database function.

---

## 7. Indexes

### 7.1 Index Strategy

| Index Type       | Use Case                 | Example                         |
| ---------------- | ------------------------ | ------------------------------- |
| B-tree (default) | Equality, range queries  | `WHERE status = 'active'`       |
| GIN              | Full-text search, arrays | `WHERE tags @> ARRAY['urgent']` |
| GiST             | Geometric, ranges        | `WHERE tsrange OVERLAPS`        |
| Partial          | Filtered queries         | `WHERE deleted_at IS NULL`      |
| Covering         | Index-only scans         | `INCLUDE (name, email)`         |

### 7.2 Key Indexes

```sql
-- Customers
CREATE INDEX idx_customers_email ON customers(email) WHERE deleted_at IS NULL;
CREATE INDEX idx_customers_status_type ON customers(status, customer_type) WHERE deleted_at IS NULL;
CREATE INDEX idx_customers_fulltext ON customers USING gin(
    to_tsvector('english', coalesce(first_name,'') || ' ' || coalesce(last_name,''))
);

-- Jobs
CREATE INDEX idx_jobs_customer ON jobs(customer_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_jobs_status ON jobs(status) WHERE deleted_at IS NULL;
CREATE INDEX idx_jobs_scheduled ON jobs(scheduled_start_at, scheduled_end_at)
    WHERE deleted_at IS NULL AND status = 'scheduled';
CREATE INDEX idx_jobs_status_priority ON jobs(status, priority) WHERE deleted_at IS NULL;

-- Invoices
CREATE INDEX idx_invoices_customer ON invoices(customer_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_invoices_status ON invoices(status) WHERE deleted_at IS NULL;
CREATE INDEX idx_invoices_due_date ON invoices(due_date)
    WHERE deleted_at IS NULL AND status IN ('sent', 'viewed');
CREATE INDEX idx_invoices_overdue ON invoices(due_date)
    WHERE deleted_at IS NULL AND status = 'overdue';

-- Schedules
CREATE INDEX idx_schedules_time_range ON schedules USING gist(
    tsrange(start_time, end_time)
) WHERE deleted_at IS NULL AND is_cancelled = FALSE;
CREATE INDEX idx_schedules_techs ON schedules USING gin(assigned_tech_ids)
    WHERE deleted_at IS NULL;

-- Payments
CREATE INDEX idx_payments_invoice ON payments(invoice_id);
CREATE INDEX idx_payments_status ON payments(status);
CREATE INDEX idx_payments_stripe ON payments(stripe_payment_intent_id);
```

---

## 8. Partitioning

No table is partitioned today. `revenue_transactions` is a plain table keyed by
`id`; revisit range partitioning by date only if volume demands it.

---

## 9. Data Integrity

### 9.1 Foreign Key Constraints

```sql
-- Cascade on delete (for child records)
ALTER TABLE invoice_lines
    ADD CONSTRAINT fk_invoice_lines_invoice
    FOREIGN KEY (invoice_id) REFERENCES invoices(id) ON DELETE CASCADE;

-- Restrict on delete (for important references)
ALTER TABLE jobs
    ADD CONSTRAINT fk_jobs_customer
    FOREIGN KEY (customer_id) REFERENCES customers(id) ON DELETE RESTRICT;

-- Set null on delete (for optional references)
ALTER TABLE invoices
    ADD CONSTRAINT fk_invoices_job
    FOREIGN KEY (job_id) REFERENCES jobs(id) ON DELETE SET NULL;
```

### 9.2 Check Constraints

```sql
-- Valid email format
ALTER TABLE users ADD CONSTRAINT valid_email
    CHECK (email ~* '^[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Za-z]{2,}$');

-- Valid phone format
ALTER TABLE customers ADD CONSTRAINT valid_phone
    CHECK (phone_primary IS NULL OR phone_primary ~ '^\+?[0-9]{10,15}$');

-- Valid state code
ALTER TABLE customers ADD CONSTRAINT valid_state
    CHECK (billing_state IS NULL OR billing_state ~ '^[A-Z]{2}$');

-- Valid amounts
ALTER TABLE invoices ADD CONSTRAINT valid_invoice_amounts
    CHECK (
        subtotal >= 0 AND
        tax_amount >= 0 AND
        discount_amount >= 0 AND
        total_amount >= 0 AND
        amount_paid >= 0 AND
        amount_paid <= total_amount
    );

-- Valid date ranges
ALTER TABLE jobs ADD CONSTRAINT valid_schedule
    CHECK (scheduled_end_at IS NULL OR scheduled_end_at > scheduled_start_at);
```

### 9.3 Triggers for Data Integrity

19 triggers in the baseline schema: 17 `BEFORE UPDATE` triggers calling
`update_updated_at_column()`, one calling `calculate_quote_item_total()` and one
calling `generate_invoice_number()`. Status transitions (jobs, quotes,
invoices) are enforced in the Go services, not in triggers.

---

## 10. Performance Considerations

### 10.1 Query Optimization

```sql
-- Use EXPLAIN ANALYZE for query planning
EXPLAIN (ANALYZE, BUFFERS, FORMAT TEXT)
SELECT c.*, COUNT(j.id) AS job_count
FROM customers c
LEFT JOIN jobs j ON j.customer_id = c.id AND j.deleted_at IS NULL
WHERE c.deleted_at IS NULL AND c.status = 'active'
GROUP BY c.id
ORDER BY job_count DESC
LIMIT 10;

-- Optimize with covering indexes
CREATE INDEX idx_jobs_customer_covering
ON jobs(customer_id)
INCLUDE (status, scheduled_start_at)
WHERE deleted_at IS NULL;
```

### 10.2 Connection Pooling

Production uses Neon's built-in PgBouncer: the runtime `DATABASE_URL` points at
the pooled (`-pooler`) host; migrations and `pg_dump` use the direct host
(`MIGRATION_DATABASE_URL`). The API's own pool is `DB_MAX_OPEN_CONNS` (default
10 outside development); keep Cloud Run `max-instances × DB_MAX_OPEN_CONNS`
under Neon's connection limit. `DB_SIMPLE_PROTOCOL=true` (automatic for a
port-6543 URL) disables server-side prepared statements for transaction
poolers.

### 10.3 Maintenance Tasks

```sql
-- Regular maintenance schedule
-- Run VACUUM ANALYZE nightly
VACUUM ANALYZE customers;
VACUUM ANALYZE jobs;
VACUUM ANALYZE invoices;

-- Reindex periodically (weekly)
REINDEX TABLE CONCURRENTLY customers;
REINDEX TABLE CONCURRENTLY jobs;
```

### 10.4 Monitoring Queries

```sql
-- Slow query log
ALTER SYSTEM SET log_min_duration_statement = '1000'; -- Log queries > 1s

-- Table sizes
SELECT
    schemaname,
    tablename,
    pg_size_pretty(pg_total_relation_size(schemaname||'.'||tablename)) AS total_size,
    pg_size_pretty(pg_relation_size(schemaname||'.'||tablename)) AS table_size,
    pg_size_pretty(pg_indexes_size(schemaname||'.'||tablename)) AS index_size
FROM pg_tables
WHERE schemaname = 'public'
ORDER BY pg_total_relation_size(schemaname||'.'||tablename) DESC;

-- Index usage
SELECT
    schemaname,
    tablename,
    indexname,
    idx_scan,
    idx_tup_read,
    idx_tup_fetch
FROM pg_stat_user_indexes
ORDER BY idx_scan DESC;

-- Unused indexes (candidates for removal)
SELECT
    schemaname,
    tablename,
    indexname,
    pg_size_pretty(pg_relation_size(indexrelid)) AS size
FROM pg_stat_user_indexes
WHERE idx_scan = 0 AND indexrelid::regclass::text NOT LIKE '%_pkey';
```

---

## Document History

| Version | Date    | Author           | Changes               |
| ------- | ------- | ---------------- | --------------------- |
| 1.0     | 2026-01 | Engineering Team | Initial documentation |
| 1.1     | 2026-10 | Engineering Team | Real counts, views, functions; migrations squash; Neon pooling; no partitioning |

---

_This document should be reviewed and updated when schema changes occur._
