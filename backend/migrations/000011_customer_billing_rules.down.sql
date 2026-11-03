DROP TABLE IF EXISTS job_approvals;
ALTER TABLE customers DROP CONSTRAINT IF EXISTS customers_nte_limit_nonnegative;
ALTER TABLE customers DROP COLUMN IF EXISTS consolidated_billing;
ALTER TABLE customers DROP COLUMN IF EXISTS nte_limit;
ALTER TABLE customers DROP COLUMN IF EXISTS billing_approver_email;
ALTER TABLE customers DROP COLUMN IF EXISTS billing_approver_name;
ALTER TABLE customers DROP COLUMN IF EXISTS default_po_number;
ALTER TABLE customers DROP COLUMN IF EXISTS po_required;
