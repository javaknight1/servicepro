-- ============================================================================
-- 000008_tenant_counters (T120 / correctness audit 7.C3, 7.C4)
--
-- Invoice, quote and job numbers come from one counter row per tenant and
-- kind, incremented with an upsert inside the create transaction
-- (models.NextDocumentNumber). The old schemes read MAX(number)+1 or
-- COUNT(today)+1 through the soft-delete scope while the unique constraint
-- includes deleted rows, so deleting the newest document made every later
-- create fail; they also raced and (before 000003) were global.
--
-- Numbers are unique per tenant already (000003). The never-firing invoice
-- number trigger and the global quote number function/sequence are dropped.
-- ============================================================================

CREATE TABLE IF NOT EXISTS tenant_counters (
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    kind VARCHAR(20) NOT NULL CHECK (kind IN ('invoice', 'quote', 'job')),
    next_value BIGINT NOT NULL DEFAULT 1 CHECK (next_value > 0),
    PRIMARY KEY (tenant_id, kind)
);

-- Start each existing tenant after its highest number in the new format
-- (deleted rows included). Numbers in older formats (JOB-YYYYMMDD-NNNN,
-- QUO-YYYY-NNNN) can't collide with PREFIX-NNNNNN; any that do are skipped
-- by the allocator.
INSERT INTO tenant_counters (tenant_id, kind, next_value)
SELECT tenant_id, 'invoice', MAX(CAST(SUBSTRING(invoice_number FROM 5) AS BIGINT)) + 1
FROM invoices WHERE invoice_number ~ '^INV-[0-9]{1,18}$'
GROUP BY tenant_id
ON CONFLICT (tenant_id, kind) DO UPDATE SET next_value = GREATEST(tenant_counters.next_value, EXCLUDED.next_value);

INSERT INTO tenant_counters (tenant_id, kind, next_value)
SELECT tenant_id, 'quote', MAX(CAST(SUBSTRING(quote_number FROM 5) AS BIGINT)) + 1
FROM quotes WHERE quote_number ~ '^QUO-[0-9]{1,18}$'
GROUP BY tenant_id
ON CONFLICT (tenant_id, kind) DO UPDATE SET next_value = GREATEST(tenant_counters.next_value, EXCLUDED.next_value);

INSERT INTO tenant_counters (tenant_id, kind, next_value)
SELECT tenant_id, 'job', MAX(CAST(SUBSTRING(job_number FROM 5) AS BIGINT)) + 1
FROM jobs WHERE job_number ~ '^JOB-[0-9]{1,18}$'
GROUP BY tenant_id
ON CONFLICT (tenant_id, kind) DO UPDATE SET next_value = GREATEST(tenant_counters.next_value, EXCLUDED.next_value);

DROP TRIGGER IF EXISTS trigger_generate_invoice_number ON invoices;
DROP FUNCTION IF EXISTS generate_invoice_number();
DROP FUNCTION IF EXISTS generate_quote_number();
DROP SEQUENCE IF EXISTS quote_number_seq;
