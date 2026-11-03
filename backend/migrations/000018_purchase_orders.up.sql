-- 000018_purchase_orders (T187)
-- PO numbers on quotes, service agreements and jobs, and tracked purchase
-- orders with an authorised amount.
--
-- quotes.po_number / service_agreements.po_number: the customer's PO for the
-- quoted work or the recurring service. A job converted from a quote, or a
-- visit an agreement generates, copies it into jobs.po_number. Invoices take
-- the most specific PO: the job's (or the agreement's), else the customer's
-- default_po_number (000011).
--
-- customer_purchase_orders: a PO the customer issued, with the amount it
-- authorises and an optional expiry. Invoices reference a PO by number
-- (invoices.po_number, same customer); billed = the totals of the
-- customer's issued invoices (not draft, not cancelled) with that number,
-- remaining = amount - billed. Sending an invoice that would take remaining
-- below zero, or against an expired PO, is a 409 the office can override
-- (the PO row is locked so concurrent sends serialise). One row per
-- (customer, po_number). Tenant-owned: tenant_id NOT NULL, (tenant_id, ...)
-- index, the RLS policy of 000007, tenancy.DefaultTables.

ALTER TABLE quotes ADD COLUMN IF NOT EXISTS po_number VARCHAR(100);
ALTER TABLE service_agreements ADD COLUMN IF NOT EXISTS po_number VARCHAR(100);
ALTER TABLE jobs ADD COLUMN IF NOT EXISTS po_number VARCHAR(100);

CREATE TABLE IF NOT EXISTS customer_purchase_orders (
    id          UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id   UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    customer_id UUID NOT NULL REFERENCES customers(id) ON DELETE CASCADE,
    po_number   VARCHAR(100) NOT NULL CHECK (btrim(po_number) <> ''),
    amount      NUMERIC(12, 2) NOT NULL CHECK (amount > 0),
    expires_on  DATE,
    notes       TEXT,
    created_by  UUID REFERENCES users(id) ON DELETE SET NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT customer_purchase_orders_number_key UNIQUE (tenant_id, customer_id, po_number)
);

-- Invoices are matched to a PO by (customer, po_number).
CREATE INDEX IF NOT EXISTS idx_invoices_tenant_customer_po
    ON invoices (tenant_id, customer_id, po_number) WHERE po_number IS NOT NULL;

ALTER TABLE customer_purchase_orders ENABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS tenant_isolation ON customer_purchase_orders;
CREATE POLICY tenant_isolation ON customer_purchase_orders
    USING (app_rls_bypass() OR tenant_id = app_current_tenant())
    WITH CHECK (app_rls_bypass() OR tenant_id = app_current_tenant());
