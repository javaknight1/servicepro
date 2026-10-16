-- T127: invoices and quotes carry the currency they are billed in, taken
-- from the tenant's currency setting when the document is created. Existing
-- documents were always charged in USD.
ALTER TABLE invoices ADD COLUMN IF NOT EXISTS currency VARCHAR(3) NOT NULL DEFAULT 'USD';
ALTER TABLE quotes ADD COLUMN IF NOT EXISTS currency VARCHAR(3) NOT NULL DEFAULT 'USD';
