import { useEffect, useState } from 'react';
import { Loader2 } from 'lucide-react';
import { Button, Modal } from '@components/shared';
import { invoiceService } from '@services/invoiceService';
import { getInvoiceActionError } from './invoiceActionError';

interface CancelInvoiceModalProps {
  isOpen: boolean;
  invoiceId: string;
  invoiceNumber?: string;
  onClose: () => void;
  /** Called after the API cancelled the invoice (reload it). */
  onCancelled: () => void;
}

/** Asks for the cancellation reason the API requires, then cancels. */
export function CancelInvoiceModal({
  isOpen,
  invoiceId,
  invoiceNumber,
  onClose,
  onCancelled,
}: CancelInvoiceModalProps) {
  const [reason, setReason] = useState('');
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    if (isOpen) {
      setReason('');
      setError(null);
    }
  }, [isOpen]);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    const trimmed = reason.trim();
    if (!trimmed) {
      setError('Enter a reason for cancelling this invoice.');
      return;
    }
    setIsSubmitting(true);
    setError(null);
    try {
      await invoiceService.cancelInvoice(invoiceId, trimmed);
      onCancelled();
    } catch (err) {
      setError(getInvoiceActionError(err, 'Cancelling the invoice'));
    } finally {
      setIsSubmitting(false);
    }
  };

  return (
    <Modal
      isOpen={isOpen}
      onClose={isSubmitting ? () => {} : onClose}
      title={
        invoiceNumber ? `Cancel Invoice ${invoiceNumber}` : 'Cancel Invoice'
      }
      description="The customer's pay link stops working. Invoices with payments can't be cancelled; refund them instead."
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
        <div className="flex flex-col">
          <label
            htmlFor="cancel-reason"
            className="mb-1.5 text-sm font-medium text-neutral-700"
          >
            Reason for cancelling *
          </label>
          <textarea
            id="cancel-reason"
            value={reason}
            onChange={(e) => setReason(e.target.value)}
            rows={3}
            required
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
            Keep Invoice
          </Button>
          <Button
            type="submit"
            variant="danger"
            disabled={isSubmitting}
            className="flex items-center gap-2"
          >
            {isSubmitting && <Loader2 className="h-4 w-4 animate-spin" />}
            Cancel Invoice
          </Button>
        </div>
      </form>
    </Modal>
  );
}
