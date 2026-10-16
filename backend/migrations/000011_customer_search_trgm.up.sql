-- ============================================================================
-- 000011_customer_search_trgm (T123 / audit index list item 2)
--
-- Customer search, advanced search and autocomplete filter with
-- LOWER(col) LIKE '%term%' / 'term%'. Trigram GIN indexes on the same
-- expressions let Postgres use an index instead of scanning every customer.
-- ============================================================================

CREATE EXTENSION IF NOT EXISTS pg_trgm;

CREATE INDEX IF NOT EXISTS idx_customers_first_name_trgm
    ON customers USING gin (LOWER(first_name) gin_trgm_ops) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_customers_last_name_trgm
    ON customers USING gin (LOWER(last_name) gin_trgm_ops) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_customers_email_trgm
    ON customers USING gin (LOWER(email) gin_trgm_ops) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_customers_company_name_trgm
    ON customers USING gin (LOWER(company_name) gin_trgm_ops) WHERE deleted_at IS NULL;
