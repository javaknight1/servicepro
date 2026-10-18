import { useEffect, useState } from 'react';
import { Loader2 } from 'lucide-react';
import { Button, Input, Modal } from '@components/shared';
import { invoiceService } from '@services/invoiceService';
import { formatDollars, roundToCents } from '@/utils/money';
import { getInvoiceActionError } from './invoiceActionError';
import { MANUAL_PAYMENT_METHODS } from './paymentMethods';

const today = () => new Date().toISOString().split('T')[0];

interface RefundModalProps {
  isOpen: boolean;
  invoiceId: string;
  /** Amount paid so far; pre-fills the refund and caps it. */
  amountPaid: number;
  onClose: () => void;
  /** Called after the API recorded the refund (reload the invoice). */
  onRefunded: () => void;
}

/**
 * Records money returned to the customer outside Stripe (cash, check, a
 * card refund done elsewhere). Stripe refunds arrive by webhook on their own.
 */
export function RefundModal({
  isOpen,
  invoiceId,
  amountPaid,
  onClose,
  onRefunded,
}: RefundModalProps) {
  const [amount, setAmount] = useState('');
  const [method, setMethod] = useState<string>('check');
  const [reason, setReason] = useState('');
  const [refundDate, setRefundDate] = useState(today());
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    if (isOpen) {
      setAmount(amountPaid > 0 ? amountPaid.toFixed(2) : '');
      setMethod('check');
      setReason('');
      setRefundDate(today());
      setError(null);
    }
  }, [isOpen, amountPaid]);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    const value = roundToCents(parseFloat(amount));
    if (!Number.isFinite(value) || value <= 0) {
      setError('Enter an amount greater than zero.');
      return;
    }
    if (value > amountPaid) {
      setError(
        `The refund is more than the amount paid (${formatDollars(amountPaid)}).`
      );
      return;
    }

    setIsSubmitting(true);
    setError(null);
    try {
      await invoiceService.recordRefund(invoiceId, {
        amount: value,
        method,
        reason: reason.trim() || undefined,
        refund_date: refundDate
          ? new Date(`${refundDate}T00:00:00Z`).toISOString()
          : undefined,
      });
      onRefunded();
    } catch (err) {
      setError(getInvoiceActionError(err, 'Recording the refund'));
    } finally {
      setIsSubmitting(false);
    }
  };

  return (
    <Modal
      isOpen={isOpen}
      onClose={isSubmitting ? () => {} : onClose}
      title="Record Refund"
      description={`Amount paid: ${formatDollars(amountPaid)}. Refunding all of it marks the invoice refunded.`}
    >
      <form onSubmit={handleSubmit} className="space-y-4" noValidate>
        {error && (
          <div
            role="alert"
            className="p-3 bg-red-50 border border-red-200 rounded-lg text-sm text-red-700"
          >
            {error}
          </div>
        )}
        <Input
          id="refund-amount"
          label="Refund amount"
          type="number"
          inputMode="decimal"
          min="0.01"
          step="0.01"
          value={amount}
          onChange={(e) => setAmount(e.target.value)}
          required
          fullWidth
        />
        <div className="flex flex-col">
          <label
            htmlFor="refund-method"
            className="mb-1.5 text-sm font-medium text-neutral-700"
          >
            Refund method
          </label>
          <select
            id="refund-method"
            value={method}
            onChange={(e) => setMethod(e.target.value)}
            className="px-3 py-2 border border-neutral-300 rounded-lg focus:ring-2 focus:ring-primary-500 focus:border-primary-500"
          >
            {MANUAL_PAYMENT_METHODS.map((m) => (
              <option key={m.value} value={m.value}>
                {m.label}
              </option>
            ))}
          </select>
        </div>
        <Input
          id="refund-date"
          label="Refund date"
          type="date"
          value={refundDate}
          onChange={(e) => setRefundDate(e.target.value)}
          fullWidth
        />
        <div className="flex flex-col">
          <label
            htmlFor="refund-reason"
            className="mb-1.5 text-sm font-medium text-neutral-700"
          >
            Reason
          </label>
          <textarea
            id="refund-reason"
            value={reason}
            onChange={(e) => setReason(e.target.value)}
            rows={2}
            className="px-3 py-2 border border-neutral-300 rounded-lg focus:ring-2 focus:ring-primary-500 focus:border-primary-500"
          />
        </div>
        <div className="flex justify-end gap-3 pt-2">
          <Button
            type="button"
            variant="secondary"
            onClick={onClose}
            disabled={isSubmitting}
          >
            Close
          </Button>
          <Button
            type="submit"
            variant="danger"
            disabled={isSubmitting}
            className="flex items-center gap-2"
          >
            {isSubmitting && <Loader2 className="h-4 w-4 animate-spin" />}
            Record Refund
          </Button>
        </div>
      </form>
    </Modal>
  );
}
