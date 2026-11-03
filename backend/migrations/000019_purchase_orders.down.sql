-- Reverts 000019_purchase_orders.
DROP INDEX IF EXISTS idx_invoices_tenant_customer_po;
DROP TABLE IF EXISTS customer_purchase_orders;
ALTER TABLE jobs DROP COLUMN IF EXISTS po_number;
ALTER TABLE service_agreements DROP COLUMN IF EXISTS po_number;
ALTER TABLE quotes DROP COLUMN IF EXISTS po_number;
