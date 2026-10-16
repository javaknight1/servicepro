-- ============================================================================
-- 000009_invoice_refunds (T125 / correctness audit 7.H5)
--
-- Refunds were ignored: charge.refunded was only logged, amount_paid never
-- went down and "refunded" was never set. Each refund is now a row here,
-- written in the same transaction that lowers invoices.amount_paid under the
-- invoice row lock. A child of invoices like invoice_payments (no tenant_id;
-- reached through the tenant-scoped invoice).
--
-- Named invoice_refunds, not payment_refunds: the latter belongs to the dead
-- models.Payment/PaymentRefund code (removed in T129) and never existed.
-- ============================================================================

CREATE TABLE IF NOT EXISTS invoice_refunds (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    invoice_id UUID NOT NULL REFERENCES invoices(id) ON DELETE CASCADE,
    amount DECIMAL(12, 2) NOT NULL CHECK (amount > 0),
    refund_date DATE NOT NULL DEFAULT CURRENT_DATE,
    method VARCHAR(50),
    reason TEXT,
    stripe_payment_intent_id VARCHAR(255),
    stripe_charge_id VARCHAR(255),
    created_by UUID REFERENCES users(id),
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_invoice_refunds_invoice ON invoice_refunds(invoice_id);
CREATE INDEX IF NOT EXISTS idx_invoice_refunds_stripe_payment_intent
    ON invoice_refunds(stripe_payment_intent_id) WHERE stripe_payment_intent_id IS NOT NULL;

DROP TRIGGER IF EXISTS trigger_invoice_refunds_updated_at ON invoice_refunds;
CREATE TRIGGER trigger_invoice_refunds_updated_at BEFORE UPDATE ON invoice_refunds
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
