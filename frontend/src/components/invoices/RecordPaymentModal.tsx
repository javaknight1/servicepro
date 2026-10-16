import { useEffect, useState } from 'react';
import { Loader2 } from 'lucide-react';
import { Button, Input, Modal } from '@components/shared';
import { invoiceService } from '@services/invoiceService';
import { formatDollars, roundToCents } from '@/utils/money';
import { getInvoiceActionError } from './invoiceActionError';

/** Methods offered for a manually recorded payment (free text in the API). */
export const MANUAL_PAYMENT_METHODS = [
  { value: 'check', label: 'Check' },
  { value: 'cash', label: 'Cash' },
  { value: 'bank_transfer', label: 'Bank transfer / ACH' },
  { value: 'card', label: 'Card (outside ServicePro)' },
  { value: 'other', label: 'Other' },
] as const;

const today = () => new Date().toISOString().split('T')[0];

interface RecordPaymentModalProps {
  isOpen: boolean;
  invoiceId: string;
  /** Outstanding balance; pre-fills the amount and caps it. */
  amountDue: number;
  onClose: () => void;
  /** Called after the API recorded the payment (reload the invoice). */
  onRecorded: () => void;
}

export function RecordPaymentModal({
  isOpen,
  invoiceId,
  amountDue,
  onClose,
  onRecorded,
}: RecordPaymentModalProps) {
  const [amount, setAmount] = useState('');
  const [method, setMethod] = useState<string>('check');
  const [reference, setReference] = useState('');
  const [notes, setNotes] = useState('');
  const [paymentDate, setPaymentDate] = useState(today());
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    if (isOpen) {
      setAmount(amountDue > 0 ? amountDue.toFixed(2) : '');
      setMethod('check');
      setReference('');
      setNotes('');
      setPaymentDate(today());
      setError(null);
    }
  }, [isOpen, amountDue]);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    const value = roundToCents(parseFloat(amount));
    if (!Number.isFinite(value) || value <= 0) {
      setError('Enter an amount greater than zero.');
      return;
    }
    if (value > amountDue) {
      setError(
        `The amount is more than the balance due (${formatDollars(amountDue)}).`
      );
      return;
    }

    setIsSubmitting(true);
    setError(null);
    try {
      await invoiceService.recordPayment(invoiceId, {
        amount: value,
        payment_method: method,
        reference_number: reference.trim() || undefined,
        notes: notes.trim() || undefined,
        payment_date: paymentDate
          ? new Date(`${paymentDate}T00:00:00Z`).toISOString()
          : undefined,
      });
      onRecorded();
    } catch (err) {
      setError(getInvoiceActionError(err, 'Recording the payment'));
    } finally {
      setIsSubmitting(false);
    }
  };

  return (
    <Modal
      isOpen={isOpen}
      onClose={isSubmitting ? () => {} : onClose}
      title="Record Payment"
      description={`Balance due: ${formatDollars(amountDue)}`}
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
          id="payment-amount"
          label="Amount"
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
            htmlFor="payment-method"
            className="mb-1.5 text-sm font-medium text-neutral-700"
          >
            Payment method
          </label>
          <select
            id="payment-method"
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
          id="payment-reference"
          label="Reference (check number, transaction ID)"
          value={reference}
          onChange={(e) => setReference(e.target.value)}
          maxLength={100}
          fullWidth
        />
        <Input
          id="payment-date"
          label="Payment date"
          type="date"
          value={paymentDate}
          onChange={(e) => setPaymentDate(e.target.value)}
          fullWidth
        />
        <div className="flex flex-col">
          <label
            htmlFor="payment-notes"
            className="mb-1.5 text-sm font-medium text-neutral-700"
          >
            Notes
          </label>
          <textarea
            id="payment-notes"
            value={notes}
            onChange={(e) => setNotes(e.target.value)}
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
            variant="primary"
            disabled={isSubmitting}
            className="flex items-center gap-2"
          >
            {isSubmitting && <Loader2 className="h-4 w-4 animate-spin" />}
            Record Payment
          </Button>
        </div>
      </form>
    </Modal>
  );
}
