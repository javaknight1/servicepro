-- 000015_consolidated_invoices (T188)
-- One invoice per customer per period: a line per site per agreement, plus
-- each completed extra-work job as its own line (POST /invoices/consolidate).
--
-- invoices.billing_period_start/end mark a consolidated invoice and its
-- service period (tenant-local calendar dates). invoice_lines.site_name is a
-- snapshot of the site the line is for (lines are grouped by it on screen
-- and in the PDF); approval_note names who approved extra work and for how
-- much. Snapshots, not foreign keys: an issued invoice must not change when
-- a site is renamed.
--
-- invoice_billed_items is the double-billing guard (T225): each row claims
-- exactly one billable thing for one invoice, and partial unique indexes
-- make every claim exclusive:
--   * (agreement_id, billing_month)  a monthly agreement for one calendar month
--   * agreement_visit_id             one visit of a per-visit agreement
--   * job_id                         one extra-work job
-- Claims are inserted in the same transaction as the invoice with
-- ON CONFLICT DO NOTHING; a lost claim rolls the whole invoice back.
-- Cancelling or deleting the invoice deletes its claims, so the work can be
-- billed again.
--
-- consolidated_invoice_runs claims (customer, period) for the scheduled
-- monthly run (POST /internal/jobs/consolidated-invoices), so reruns and
-- concurrent runs create at most one draft per customer and period.
--
-- Both tables are tenant-owned (tenant_id NOT NULL, (tenant_id, ...)
-- indexes, the RLS policy of 000007, tenancy.DefaultTables).

ALTER TABLE invoices ADD COLUMN IF NOT EXISTS billing_period_start DATE;
ALTER TABLE invoices ADD COLUMN IF NOT EXISTS billing_period_end DATE;

ALTER TABLE invoice_lines ADD COLUMN IF NOT EXISTS site_name VARCHAR(200);
ALTER TABLE invoice_lines ADD COLUMN IF NOT EXISTS approval_note VARCHAR(300);

CREATE TABLE IF NOT EXISTS invoice_billed_items (
    id                 UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id          UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    invoice_id         UUID NOT NULL REFERENCES invoices(id) ON DELETE CASCADE,
    invoice_line_id    UUID REFERENCES invoice_lines(id) ON DELETE SET NULL,
    agreement_id       UUID REFERENCES service_agreements(id) ON DELETE CASCADE,
    billing_month      DATE,
    agreement_visit_id UUID REFERENCES agreement_visits(id) ON DELETE CASCADE,
    job_id             UUID REFERENCES jobs(id) ON DELETE CASCADE,
    created_at         TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT invoice_billed_items_one_claim
        CHECK (num_nonnulls(billing_month, agreement_visit_id, job_id) = 1),
    CONSTRAINT invoice_billed_items_month_has_agreement
        CHECK (billing_month IS NULL OR agreement_id IS NOT NULL)
);

CREATE UNIQUE INDEX IF NOT EXISTS invoice_billed_items_agreement_month_key
    ON invoice_billed_items (agreement_id, billing_month) WHERE billing_month IS NOT NULL;
CREATE UNIQUE INDEX IF NOT EXISTS invoice_billed_items_visit_key
    ON invoice_billed_items (agreement_visit_id) WHERE agreement_visit_id IS NOT NULL;
CREATE UNIQUE INDEX IF NOT EXISTS invoice_billed_items_job_key
    ON invoice_billed_items (job_id) WHERE job_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_invoice_billed_items_tenant_invoice
    ON invoice_billed_items (tenant_id, invoice_id);

CREATE TABLE IF NOT EXISTS consolidated_invoice_runs (
    id           UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id    UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    customer_id  UUID NOT NULL REFERENCES customers(id) ON DELETE CASCADE,
    period_start DATE NOT NULL,
    period_end   DATE NOT NULL,
    invoice_id   UUID REFERENCES invoices(id) ON DELETE SET NULL,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT consolidated_invoice_runs_customer_period_key
        UNIQUE (tenant_id, customer_id, period_start, period_end)
);

DO $$
DECLARE
    t text;
BEGIN
    FOREACH t IN ARRAY ARRAY['invoice_billed_items', 'consolidated_invoice_runs'] LOOP
        EXECUTE format('ALTER TABLE %I ENABLE ROW LEVEL SECURITY', t);
        EXECUTE format('DROP POLICY IF EXISTS tenant_isolation ON %I', t);
        EXECUTE format(
            'CREATE POLICY tenant_isolation ON %I
                USING (app_rls_bypass() OR tenant_id = app_current_tenant())
                WITH CHECK (app_rls_bypass() OR tenant_id = app_current_tenant())', t);
    END LOOP;
END
$$;
